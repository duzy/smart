//
//  Copyright (C) 2012-2026, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "bufio"
    "sync"
    "os/exec"
	"bytes"
	"context"
	"encoding/base64"
	enc_bin "encoding/binary"
    enc_json "encoding/json"
    enc_xml "encoding/xml"
    // enc_csv "encoding/csv"
    // enc_yaml "encoding/yaml"
	gt "go/token"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash/fnv" // "hash/maphash"
    "hash/crc64"
	"io/fs"
	"math"
	neturl "net/url"
	"path/filepath"
    "plugin"
	"reflect"
	"regexp"
	"runtime"
    "runtime/pprof"
	rt_debug "runtime/debug"
	"strconv"
	"strings"
	"sort"
	"slices"
    "sync/atomic"
    "syscall"
	"time"
	"unsafe"
	"unicode"
	"unicode/utf8"
	"os"
	"io"
	"io/ioutil"
	"net/http"
)

type hashbytes [sha256.Size]byte

const (
    recursiveTraversalClosurePre  = false
    recursiveTraversalClosurePost = false
    recursiveTraversalClosure = true
    detectTraverseLoops = true // turn on/off traverse loop detection
	shredderChars = "./-#~_*?" // ' ' \t \n \r
	escaperChars = "\"\r\n"
)

type cmpres int
const (
	cmpLprefix cmpres = -2 // L is smaller then R, and L is the prefix of R
	cmpSmaller cmpres = -1 // L is smaller then R
	cmpEqual   cmpres =  0 // L is equal to R
	cmpGreater cmpres =  1 // L is greater than R
	cmpRprefix cmpres =  2 // L is greater than R, and R is the prefix of L
)
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
const (
    existenceMatterless existence = 1<<iota
    existenceConfirmed
    existenceNegated
)
func (n existence) String() (s string) {
    switch n {
    case existenceMatterless: s = "matterless"
    case existenceConfirmed:  s = "confirmed"
    case existenceNegated:    s = "negated"
    }
    return
}

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

func sortstrs(s []string) []string { sort.Strings(s) ; return s }
func sf(f string, i ...any) string { return fmt.Sprintf(f, i...) }
func ssf(ss []string, a ...any) (res []string) {
    for _, s := range ss {
		if strings.Contains(s, "%") { s = fmt.Sprintf(s, a...) }
		res = append(res, s)
	}
    return
}

type origin_def struct{ name Symbol }
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

func closure_resolve(ctx Context, name Symbol) object {
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
    if val := auto_get(ctx, symAt); val == nil || isTrivial(val) {
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
        erro(ctx, "hashing recipes failed: %v", err)
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
        erro(ctx, "compute recipes hash failed: %v", err)
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
        erro(ctx, "target is nil")
    }

    if isTrivial(target) {
        erro(ctx, "trivial target : %s", ts(target))
    }

    if n := len(calleeErrs); n > 0 /*&& t.stems == nil*/ {
        var numRealErrs = 0
        for _, err := range calleeErrs {
            erro(ctx, "%v: %v", target, err)
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
            erro(ctx, "waiting for def '%v': %v", def.name, def.value)
        }
        return
    }

    if opts.ExecResults {
        // Waiting for command (shell/python/etc.) exec result
        if val := auto_get(ctx, symDash); val != nil {
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
	v := expand(ctx, val)

	// CRITICAL FIX: Fast-Path Extraction
	// Catch *file pointers before scalarize() flattens them into strings!
	var current = v
	for {
		switch t := current.(type) {
		case *file: return t
		case fullfile: return t.file
		case *barefile: return t.file
		case *loc: current = t.Value; continue
		case *xloc: current = t.Value; continue
		case fullname: current = t.Value; continue
		case *list:
			if len(t.elems) == 1 { current = t.elems[0]; continue }
		case *def:
			if t.value != nil { current = t.value; continue }
		}
		break
	}

	// Fallback for raw strings and unexpanded paths
	v = scalarize(v)
	if x, y := v.(fullname); y { v = x.Value }
	switch t := v.(type) {
	case *file: return t
	case fullfile: return t.file
	case *barefile: return t.file
	case *rule: return as_file(ctx, t.target, projs...)
	case *def: if t.value != nil { return as_file(ctx, t.value, projs...) }
	case *list: if a := t.elems; len(a) == 1 { return as_file(ctx, a[0], projs...) }
	case *word, *qualword, *compound, *path:
		if projs == nil {
			if p := _project(ctx); p != nil { projs = append(projs, p) }
		}
		for _, p := range projs {
			if f := p.file(ctx, t); f != nil { return f }
		}
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
	// OPTIMIZATION 3: Decoupled String Fallback
	// Prevents double-evaluation of the AST if the file lookup fails
	if f := as_file(ctx, val, projs...); f != nil {
		return f.fullname(), filepath.IsAbs(f.fullname())
	}
	return __string(ctx, val), false
}

func as_fullname(ctx Context, val Value, projs ...*project) (res fullname) {
	if f := as_file(ctx, val, projs...); f != nil { res.Value = f } else {
		// Only panic if the expanded value fundamentally cannot be a file
		debug(pc(ctx,val),
			_f("%v: nil file : %v", val, ts(val,ctx)),
			_f("%v → %v", val, expand(ctx,val)),
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
	default: erro(ctx, "err splitpath: %v %s", a, ts(a), callstack{num:10})
	}
	return
}

// joinPathSegs is different from filepath.Join, which trims and discards empty segments
func joinPathSegs(segs ...string) string { return strings.Join(segs, pathSep) }
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
		case  *scanner   : p = t.offsetPos(n...)
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
            if t.name == symEmpty || t.name == x.name { return x }
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

type symident struct{}
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
	case symident: return true
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
	var ic *ident_ctx
	if c, ok := ctx.(*ident_ctx); ok { ic = c }

	switch t := x.(type) {
	case *loc:
		return ident(ctx, t.Value)
	case *xloc:
		return ident(ctx, t.Value)
	case *closure:
		s := ident_opt(ctx, "&", x, closure_ident{})
		if s == "" && ic != nil { ic.nil++ }
		return s
	case *delegate:
		s := ident_opt(ctx, "$", x, delegate_ident{})
		if s == "" && ic != nil { ic.nil++ }
		return s
	case *file:
		return t.filestub.name.String()
	case *path:
		var b strings.Builder
		for i, elem := range t.elems {
			s := ident(ctx, elem)
			if i > 0 && s != "" && !strings.HasSuffix(b.String(), pathSep) {
				b.WriteString(pathSep)
			}
			b.WriteString(s)
		}
		return b.String()
	case *project:
		return t.name.String()
	case *uselist:
		return t.name.String()
	case *def:
		return t.name.String()
	case *defcaps:
		return ident(ctx, t.Value) // The main captured value (e.g., $0)
	case *word:
		return t.s.String()
	case *raw:
		return t.s
	case *strlit:
		return `'` + t.s + `'`
	case *punct:
		return t.token.String()
	case *globmeta:
		return t.token.String()
	case *valbase, *null, *none, *regexpat, nil: // CRITICAL FIX: Added *regexpat
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
		for i, elem := range t.elems {
			if i > 0 {
				b.WriteByte('.')
			}
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
			if lst, ok := elem.(*list); ok && lst.len() > 0 {
				b.WriteString("⌜")
				b.WriteString(ident(ctx, elem))
				b.WriteString("⌟")
			} else {
				b.WriteString(ident(ctx, elem))
			}
		}
		return b.String()
	case *strcomp:
		var b strings.Builder
		b.WriteByte('"')
		for _, elem := range t.elems {
			if lst, ok := elem.(*list); ok && lst.len() > 0 {
				b.WriteString("⌜")
				b.WriteString(ident(ctx, elem))
				b.WriteString("⌟")
			} else {
				b.WriteString(ident(ctx, elem))
			}
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
		if ic != nil {
			ic.nil++
		} else {
			erro(pc(ctx, x), "no ident for %T: %v", x, x, callstack{num: 24})
		}
		return ""
	}
}
func ident_any(ctx Context, x any) string {
	switch t := x.(type) {
	case  Value: return ident(ctx, t)
	case Symbol: return t.String()
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
	var o = enc_bin.LittleEndian
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
			erro(ctx, "fnv1: unsupported type : %s", ts(v))
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
	case *conjunction: return fnv1(ctx, p.kind(), p.sep, p.list)
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

func unbox(a any) any {
    switch t := a.(type) {
    case self: return t.project
	case fullname: return unbox(t.Value)
    case *loc: return unbox(t.Value)
    case *xloc: return unbox(t.Value)
	case *argumented: return unbox(t.Value)
    case *list: if len(t.elems) == 1 { return unbox(t.elems[0]) }
    case flag: return flag{unbox(t.Value).(Value)}
		// case *pair: return &pair{unbox(t.key).(Value), unbox(t.val).(Value)}
    }
    return a
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
		res := make([]Value, 0, len(t.elems)*2-1)
		for i, e := range t.elems {
			if i > 0 { res = append(res, implicitDot) }
			if !isEmpty(e) { res = append(res, e) }
		}
		return res
	}
	return []Value{v}
}

// tokenStr securely maps structural tokens to their string literals
// for mathematical comparison and fragmentation.
func tokenStr(t token) string {
	switch t {
	case DOT:  return "."
	case DASH: return "-"
	case PCON: return "/"
	case PROOT, PTAIL:
		return "" // Virtual segments consume 0 characters in string fragmentation!
	default: return t.String()
	}
}

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

// __symbol efficiently resolves an AST Value to its Symbol ID.
// It guarantees zero string allocations for static identifiers and
// only falls back to evaluation/interning for dynamic nodes.
func __symbol(ctx Context, val Value) Symbol {
	if val == nil {
		return symEmpty
	}

	// Unbox safety: Ensure unboxing doesn't panic if it returns nil
	if val = unbox(val).(Value); val == nil { return symEmpty }
	if dc, ok := val.(*defcaps); ok { val = dc.Value } // Unbox!

	switch v := val.(type) {
	case *word:
		return v.s
	case *pair:
		return __symbol(ctx, v.key)
	case interface{ sym() Symbol }:
		return v.sym()

	// AST Numeric Types - Zero-Allocation Fast Paths
	case *integer: // Base struct, in case the parser ever emits it directly
		if 0 <= v.int64 && v.int64 <= 9 {
			return sym_0 + Symbol(v.int64)
		}
		return intern(strconv.FormatInt(v.int64, 10))

	case *decimal:
		if 0 <= v.int64 && v.int64 <= 9 {
			return sym_0 + Symbol(v.int64)
		}
		return intern(v.String()) // Safe fallback using the struct's String() method

	case *float:
		// Check if the float represents a clean single-digit integer (e.g., 1.0 -> 1)
		if v.float64 == float64(int64(v.float64)) && 0 <= v.float64 && v.float64 <= 9 {
			return sym_0 + Symbol(int64(v.float64))
		}
		return intern(v.String())

	// Other integer bases (Binary, Octal, Hexadecimal)
	// We just pass these to intern() via their String() method.
	case *binary, *octal, *hexadecimal, *datetime, *Date, *Time:
		return intern(v.String())
	}

	// Fallback for dynamically evaluated nodes (e.g. SRC_$(ARCH)_FILES)
	var str string
	if truly(ctx, symident{}) {
		str = ident(ctx, val)
	} else {
		str = __string(ctx, val)
	}
	return intern(str)
}

func symbolize(v Value) (res []Symbol) { // renamed from `underlay`
	if checkpoints { defer check_symbolize(v)(&res) }
	switch t := v.(type) {
	case *valbase, *null, *none, *undef: return
	case *loc: return symbolize(t.Value)
	case *xloc: return symbolize(t.Value)
	case fullname: return symbolize(t.Value)
	case *globmeta: return []Symbol{intern(t.token.String())}
	case *punct: return []Symbol{intern(t.token.String())}
	case *answer: return []Symbol{_if(t.bool,symYes,symNo)}
	case *boolean: return []Symbol{_if(t.bool,symTrue,symFalse)}
	case *prediction: return []Symbol{_if(t.bool,symYes,symNo)}
	case *option: return []Symbol{_if(t.bool,symOn,symOff)}
	case *word: return getSymSeq(t.s)
	case *def: return getSymSeq(t.name)
	case *file: return getSymSeq(t.name)
	case *uselist: return getSymSeq(t.name)
	case *builtin: return getSymSeq(t.name)
	case *project: return getSymSeq(t.name)
	case self: return getSymSeq(t.project.name)

	case *integer: // Base struct, in case the parser ever emits it directly
		var s Symbol
		if 0 <= t.int64 && t.int64 <= 9 {
			s = sym_0 + Symbol(t.int64)
		} else {
			s = intern(strconv.FormatInt(t.int64, 10))
		}
		return []Symbol{s}
	case *decimal:
		var s Symbol
		if 0 <= t.int64 && t.int64 <= 9 {
			s = sym_0 + Symbol(t.int64)
		} else {
			s = intern(t.String())
		}
		return []Symbol{s}
	case *float:
		var s Symbol
		if t.float64 == float64(int64(t.float64)) && 0 <= t.float64 && t.float64 <= 9 {
			s = sym_0 + Symbol(int64(t.float64))
		} else {
			s = intern(t.String())
		}
		return []Symbol{s}

	case *binary, *octal, *hexadecimal, *datetime, *Date, *Time:
		return []Symbol{intern(t.String())}

	case flag:
		res = append([]Symbol{symDash},symbolize(t.Value)...)
	case *regexpat:
		res = []Symbol{intern(t.Regexp.String())}
	case *percpat:
		res = append(res, symbolize(t.Prefix)...)
		res = append(res, symPercent) // %
		res = append(res, symbolize(t.Suffix)...)
	case *pair:
		res = append(res, symbolize(t.key)...)
		res = append(res, symEqualSign) // =
		res = append(res, symbolize(t.val)...)
	case *qualword:
		for i, elem := range t.elems {
			if i > 0 { res = append(res, symDot) }
			res = append(res, symbolize(elem)...)
		}
	case *path:
		for i, elem := range t.elems {
			if i > 0 { res = append(res, symSlash) }

			// Intercept and ignore PROOT/PTAIL.
			// Their presence is naturally translated into slashes by the `i > 0` logic!
			if p, ok := elem.(*punct); ok && (p.token == PROOT || p.token == PTAIL) {
				continue
			}

			res = append(res, symbolize(elem)...)
		}
	case *url:
		// 1. Scheme (e.g., "https:")
		if t.Scheme != nil {
			res = append(res, symbolize(t.Scheme)...)
			res = append(res, symColon)
		}

		// 2. Authority (User, Password, Host, Port)
		// If there is a Host or User, we inject the standard "//" boundary
		if t.Host != nil || t.Username != nil {
			res = append(res, symSlash, symSlash) // "//"

			var hasUser bool
			if t.Username != nil {
				res = append(res, symbolize(t.Username)...)
				hasUser = true
			}
			if t.Password != nil {
				res = append(res, symColon)
				res = append(res, symbolize(t.Password)...)
				hasUser = true
			}
			if hasUser {
				res = append(res, symAt) // "@"
			}

			if t.Host != nil {
				res = append(res, symbolize(t.Host)...)
			}
			if t.Port != nil {
				res = append(res, symColon)
				res = append(res, symbolize(t.Port)...)
			}
		}

		// 3. Path
		if t.Path != nil {
			res = append(res, symbolize(t.Path)...)
		}

		// 4. Query (e.g., "?key=val&foo=bar")
		if len(t.Query) > 0 {
			res = append(res, symQues) // "?"
			for i, q := range t.Query {
				if i > 0 { res = append(res, symAmpersand) } // "&"
				res = append(res, symbolize(q)...)
			}
		}

		// 5. Fragment (e.g., "#section1")
		if t.Fragment != nil {
			res = append(res, symHash)
			res = append(res, symbolize(t.Fragment)...)
		}
	case *list:
		for i, elem := range t.elems {
			if i > 0 { res = append(res, symSpace) }
			res = append(res, symbolize(elem)...)
		}
	case *compound, *globpat, *strcomp:
		for _, elem := range t.(slicer).slice() {
			res = append(res, symbolize(elem)...)
		}
	case *closure:
		return symbolize_delegate(symAmpersand, &t.delegate)
	case *delegate:
		return symbolize_delegate(symDollarSign, t)
	case *strlit:
        // Semantic Equality: 'foo' (strlit) == foo (word)
		// getSymSeq unpacks the hologram into a flat []Symbol slice
		return getSymSeq(intern(t.s))
	case *raw:
		// FIXME: a raw value may store strings from p.Stdout.Buf.String()!
        // Semantic Equality: "foo" (raw) == foo (word)
		// TODO: symbolize_string(trivial_ctx{t.pos}, t.s)
		return getSymSeq(intern(t.String()))
	}
    return
}

// example $(touch(-p -x -y) a1, a2)
func symbolize_delegate(sigil Symbol, d *delegate) []Symbol {
	var res []Symbol

	// Determine the boundaries based on the lexical token
	var open, close Symbol
	switch d.l {
	case STRCOMP: open, close = symQuotation, symQuotation // `"`
	case STRING : open, close = symApostrophe, symApostrophe // `'`
	case LPAREN : open, close = symLparen, symRparen
	case LBRACE : open, close = symLbrace, symRbrace
	case ILLEGAL: open, close = symLbrack, symRbrack
	// INTEGER and default have no enclosing boundaries
	}

	// 1. Sigil (e.g., '$' for DELEGATE or '&' for CLOSURE)
	res = append(res, sigil)

	// 2. Open Boundary
	if open != symEmpty { res = append(res, open) }

	// 3. Target (e.g., "touch")
	if d.x != nil {
		res = append(res, symbolize(d.x)...)
	}

	// 4. Options (e.g., "(-p -x -y)")
	if len(d.o) > 0 {
		res = append(res, symLparen)
		for i, opt := range d.o {
			if i > 0 { res = append(res, symSpace) } // ' ' between options
			res = append(res, symbolize(opt)...)
		}
		res = append(res, symRparen)
	}

	// 5. Arguments (e.g., " a1,a2")
	if len(d.a) > 0 {
		res = append(res, symSpace) // ' ' separating target/options from args
		for i, arg := range d.a {
			if i > 0 { res = append(res, symComma) } // ',' between args
			res = append(res, symbolize(arg)...)
		}
	}

	// 6. Close Boundary
	if close != symEmpty { res = append(res, close) }

	return res
}

// TODO: verification required!
func symbolize_string(ctx Context, str string) (res []Symbol) {
	// Protect the engine from crashing if the scanner hits dirty stdout data
	// (e.g., unterminated quotes, bad escapes).
	defer func() { recover() }()

	var s scanner
	// Ensure you pass a dummy tokfile or valid pos to avoid nil pointers inside s.offsetPos
	s.init(ctx, nil, []byte(str), 0)

	s.scan(ctx) // Prime the scanner
	for s.tok != EOF {
		if s.sym != symEmpty {
			res = append(res, s.sym)
		} else if s.lit != "" {
			res = append(res, intern(s.lit))
		} else {
			// Fallback for punctuation tokens without lit/sym (e.g., '{', '}')
			res = append(res, Symbol(s.tok)) // FIXME: currently token and Symbol are not identically convertable
		}
		s.scan(ctx)
	}
	return res // FIXED: Was returning nil
}

// cmp_symbol handles atomic comparisons and returns the strings for dynamic fragmentation.
func cmp_symbol(ctx Context, x, y Symbol) (result cmpres, sx, sy string) {
	if checkpoints { defer check_cmp_symbol(ctx, x, y) (&result) }

	vocab.RLock()
	metaL := vocab.symetas[x]
	metaR := vocab.symetas[y]
	vocab.RUnlock()

	sx, sy = metaL.Text, metaR.Text

	// 1. Rank comparison (Globs)
	if rL, rR := metaL.Rank(), metaR.Rank(); rL > 0 || rR > 0 {
		if res := cmp_rank(rL, rR); res != cmpEqual {
			return res, sx, sy
		}
	}

	// 2. Numeric comparison
	if kL, kR := metaL.Kind(), metaR.Kind(); kL > 0 && kR > 0 {
		var iL, iR int64
		var fL, fR float64

		if kL == NumInt { iL = int64(vocab.numbers[metaL.Idx]) } else { fL = math.Float64frombits(vocab.numbers[metaL.Idx]) }
		if kR == NumInt { iR = int64(vocab.numbers[metaR.Idx]) } else { fR = math.Float64frombits(vocab.numbers[metaR.Idx]) }

		if kL == NumInt && kR == NumInt {
			if iL < iR { return cmpSmaller, sx, sy }
			if iL > iR { return cmpGreater, sx, sy }
		} else {
			if kL == NumInt { fL = float64(iL) }
			if kR == NumInt { fR = float64(iR) }
			if fL < fR { return cmpSmaller, sx, sy }
			if fL > fR { return cmpGreater, sx, sy }
		}
	}

	// 3. String Prefix & Alphabetical Fallback
	if len(sx) < len(sy) && strings.HasPrefix(sy, sx) { return cmpLprefix, sx, sy }
	if len(sy) < len(sx) && strings.HasPrefix(sx, sy) { return cmpRprefix, sx, sy }

	if sx < sy { return cmpSmaller, sx, sy }
	if sx > sy { return cmpGreater, sx, sy }
	return cmpEqual, sx, sy
}

func cmp_symbols(ctx Context, x, y []Symbol) (result cmpres) {
	if checkpoints { defer check_cmp_symbols(ctx, x, y) (&result) }

	i, lenX, lenY := 0, len(x), len(y)

	for i < lenX && i < lenY {
		lx, ly := x[i], y[i]

		if lx == ly { // FAST PATH: O(1) Integer Equality
			i++
			continue
		}

		// 2. Fragmentation (Dynamic Concatenation Alignment)
		switch res, sx, sy := cmp_symbol(ctx, lx, ly); res {
		case cmpLprefix:
			tail := sy[len(sx):]
			restX := x[i+1:]
			restY := append([]Symbol{intern(tail)}, y[i+1:]...)
			return cmp_symbols(ctx, restX, restY)
		case cmpRprefix:
			tail := sx[len(sy):]
			restX := append([]Symbol{intern(tail)}, x[i+1:]...)
			restY := y[i+1:]
			return cmp_symbols(ctx, restX, restY)
		default:
			// Hard mismatch (e.g., "foo" vs "bar", or "10" vs "2")
			return res
		}
	}

	// Boundary Tie-breakers
	if lenX < lenY { return cmpLprefix }
	if lenX > lenY { return cmpRprefix }
	return cmpEqual
}

func cmp(ctx Context, l, r Value) (result cmpres) {
	if checkpoints { defer check_cmp(ctx, l, r) (&result) }
    return cmp_symbols(ctx, symbolize(l), symbolize(r))
}

func cmps(ctx Context, l, r []Value) cmpres {
	lenL, lenR := len(l), len(r)
	limit := lenL
	if lenR < limit {
		limit = lenR
	}

	for i := 0; i < limit; i++ {
		if res := cmp(ctx, l[i], r[i]); res != cmpEqual {
			return res
		}
	}

	if lenL < lenR { return cmpSmaller }
	if lenL > lenR { return cmpGreater }
	return cmpEqual
}

func cmp_time(l, r time.Time) cmpres {
    switch {
    case l.Before(r): return cmpSmaller
    case l.After(r) : return cmpGreater
    }
	return cmpEqual // l.Equal(r)
}

func eq(x Context, a, b Value) bool { return cmp(x, a, b) == cmpEqual }
func equal(x Context, a, b Value, yes ...bool) (res bool) {
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
	case *word: return t.s == symEmpty || t.s.String() == ""
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
	case *word: return t.s == symEmpty || t.s.String() == ""
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
		switch ty := y.(type) {
		case *path:
			if len(ty.elems) > 0 {
				if _, ok := ty.elems[0].(*punct); ok {
					tail := ty.elems[1:]
					if len(tail) == 0 {
						tail = []Value{&word{valbase{ty.Pos()}, symEmpty}}
					}
					return &path{elements{append(dup(tx.elems), tail...)}}
				}
				fused := prefix(ctx, tx.elems[len(tx.elems)-1], ty.elems[0])
				res := append(dup(tx.elems[:len(tx.elems)-1]), fused)
				return &path{elements{append(res, ty.elems[1:]...)}}
			}
			return tx
		}
		return &path{elements{append(dup(tx.elems[:len(tx.elems)-1]), prefix(ctx, tx.elems[len(tx.elems)-1], y))}}
	case *globpat:
		switch ty := y.(type) {
		case *compound: return &globpat{elements{append(tx.elems, ty.elems...)}}
		case *globpat: return &globpat{elements{append(tx.elems, ty.elems...)}}
		default: return &globpat{elements{append(tx.elems, y)}}
		}
	case *percpat:
		return &percpat{tx.valbase, tx.Prefix, prefix(ctx, tx.Suffix, y)}
	case *compound:
		switch ty := y.(type) {
		case *path:
			if len(ty.elems) > 0 {
				if _, ok := ty.elems[0].(*punct); ok {
					tail := ty.elems[1:]
					if len(tail) == 0 {
						tail = []Value{&word{valbase{ty.Pos()}, symEmpty}}
					}
					return &path{elements{append([]Value{tx}, tail...)}}
				}
				return &path{elements{append([]Value{prefix(ctx, tx, ty.elems[0])}, ty.elems[1:]...)}}
			}
			return tx
		case *compound: return &compound{elements{append(tx.elems, ty.elems...)}}
		case *globpat: return &globpat{elements{append(tx.elems, ty.elems...)}}
		case *qualword:
			if len(ty.elems) == 0 { return tx }
			// CRITICAL FIX 1: Allow the qualword to attach to the INNERMOST element of the compound
			if i := len(tx.elems)-1; 0 <= i {
				return &compound{elements{append(dup(tx.elems[:i]), prefix(ctx, tx.elems[i], ty))}}
			}
			return tx
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
	case *qualword:
		if len(tx.elems) == 0 { return y }
		switch ty := y.(type) {
		case *qualword:
			if len(ty.elems) == 0 { return tx }
			// CRITICAL FIX 2: If the last element is complex (like a compound), pass the ENTIRE right-hand qualword into it!
			switch tx.elems[len(tx.elems)-1].(type) {
			case *compound, *path, *globpat, *percpat:
				return &qualword{elements{append(dup(tx.elems[:len(tx.elems)-1]), prefix(ctx, tx.elems[len(tx.elems)-1], ty))}}
			}
			// Otherwise flatten the qualwords normally
			return &qualword{elements{append(append(dup(tx.elems[:len(tx.elems)-1]), prefix(ctx, tx.elems[len(tx.elems)-1], ty.elems[0])), ty.elems[1:]...)}}
		case *punct:
			if ty.token == TILDE {
				return &qualword{elements{append(dup(tx.elems[:len(tx.elems)-1]), prefix(ctx, tx.elems[len(tx.elems)-1], y))}}
			}
			return &compound{elements{[]Value{tx, ty}}}
		default:
			return &qualword{elements{append(dup(tx.elems[:len(tx.elems)-1]), prefix(ctx, tx.elems[len(tx.elems)-1], y))}}
		}
	case *defcaps:
		return prefix(ctx, tx.Value, y)
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
	case *percpat:
		switch x.(type) {
		case *valbase, *null, *none, nil:
			return &percpat{ty.valbase, ty.Prefix, ty.Suffix}
		default:
			return &percpat{ty.valbase, prefix(ctx, x, ty.Prefix), ty.Suffix}
		}
	case *qualword:
		if len(ty.elems) == 0 { return x }
		// CRITICAL FIX: If the qualword starts with a placeholder, x replaces it!
		switch ty.elems[0].(type) {
		case *valbase, *null, *none, nil:
			res := dup(ty.elems)
			res[0] = x
			return &qualword{elements{res}}
		default:
			return &qualword{elements{append([]Value{prefix(ctx, x, ty.elems[0])}, ty.elems[1:]...)}}
		}
	case *path:
		if len(ty.elems) > 0 {
			if _, ok := ty.elems[0].(*punct); ok {
				tail := ty.elems[1:]
				if len(tail) == 0 {
					tail = []Value{&word{valbase{ty.Pos()}, symEmpty}}
				}
				return &path{elements{append([]Value{x}, tail...)}}
			}
			// CRITICAL FIX: If the path starts with a placeholder, x replaces it!
			switch ty.elems[0].(type) {
			case *valbase, *null, *none, nil:
				res := dup(ty.elems)
				res[0] = x
				return &path{elements{res}}
			}
			return &path{elements{append([]Value{prefix(ctx, x, ty.elems[0])}, ty.elems[1:]...)}}
		}
		return x
	case *compound:
		switch ty.elems[0].(type) {
		case *pair: erro(ctx, "%v", ty.elems)
		// CRITICAL FIX: Avoid completely losing `x` by safely replacing the placeholder
		case *valbase, *null, *none, nil:
			res := dup(ty.elems)
			res[0] = x
			return &compound{elements{res}}
		default: return &compound{elements{append([]Value{x}, ty.elems...)}}
		}
	case *defcaps:
		return prefix(ctx, x, ty.Value)
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
func (l *loc) String() string {
	if l == nil || l.Value == nil {
		return ""
	}
	return l.Value.String()
}

// xloc is a synthetic wrapper for values generated from external runtime
// files (like grep) that do not exist in the compiler's FileSet.
type xloc struct { Value ; pos Position }
func (p *xloc) Pos() Pos { return 0 }
func (p *xloc) Position() Position { return p.pos }
func (l *xloc) String() string {
	if l == nil || l.Value == nil {
		return ""
	}
	return l.Value.String()
}

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
                    erro(pc(ctx,p), "partial stencil: %v, %v, %v, %v", a, v, rest, stems)
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
type url struct{ // scheme://user:pass@host:port/path?query#fragment
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
type word struct{ valbase; s Symbol }
func (_ *word) kind() Kind { return KindWord }
func (p *word) String() string { return p.s.String() }

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

type conjunction struct{ list ; sep Value }
func (p *conjunction) kind() Kind { return KindConjunction }
func (p *conjunction) String() (s string) {
	if list := p.list.String(); p.sep == nil || isTrivial(p.sep) {
		if list == "" {
			return "{=join}"
		} else {
			return "{=join "+list+"}"
		}
	} else if sep := p.sep.String(); list == "" {
		return "{=join ("+sep+")}"
	} else {
		return "{=join ("+sep+") "+list+"}"
	}
}

func redis(v Value) Value { v, _ = _redis(v); return v }

func _redis(v Value) (Value, bool) {
	if v == nil { return nil, false }

	switch t := v.(type) {
	case *closure, *delegate:
		return &disjunction{valbase{v.Pos()}, v}, true
	case *loc:
		if res, dis := _redis(t.Value); dis {
			return &loc{res, t.pos}, true
		}
	case *xloc:
		if res, dis := _redis(t.Value); dis {
			return &xloc{res, t.pos}, true
		}
	case flag:
		if res, dis := _redis(t.Value); dis {
			return flag{res}, true
		}
	case *pair:
		key, d1 := _redis(t.key)
		val, d2 := _redis(t.val)
		if d1 || d2 {
			return &pair{key, val}, true
		}
	case *arrow:
		o, d1 := _redis(t.o)
		s, d2 := _redis(t.s)
		if d1 || d2 {
			return &arrow{t.valbase, t.t, o, s}, true
		}
	case *percpat:
		p, d1 := _redis(t.Prefix)
		s, d2 := _redis(t.Suffix)
		if d1 || d2 {
			return &percpat{t.valbase, p, s}, true
		}
	case *compound:
		if e, dis := _redis_elems(t.elems); dis {
			return &compound{elements{e}}, true
		}
	case *path:
		if e, dis := _redis_elems(t.elems); dis {
			return &path{elements{e}}, true
		}
	case *list:
		if e, dis := _redis_elems(t.elems); dis {
			return &list{elements{e}}, true
		}
	case *globpat:
		if e, dis := _redis_elems(t.elems); dis {
			return &globpat{elements{e}}, true
		}
	case *strcomp:
		if e, dis := _redis_elems(t.elems); dis {
			return &strcomp{elements{e}}, true
		}
	}

	// Fast Path: No children changed, so return the exact original pointer!
	return v, false
}

// _redis_elems uses a lazy Copy-on-Write strategy.
// It returns the exact original slice if no children are modified.
func _redis_elems(vals []Value) ([]Value, bool) {
	var res []Value
	var changed bool

	for i, val := range vals {
		e, d := _redis(val)
		if d {
			if !changed {
				// Lazy allocation! We only allocate a new backing array
				// the exact moment we detect a modification.
				res = make([]Value, len(vals))
				copy(res, vals[:i]) // Copy the unmodified prefix
				changed = true
			}
			res[i] = e
		} else if changed {
			res[i] = e
		}
	}

	if changed {
		return res, true
	}

	// Zero allocations if nothing changed
	return vals, false
}

type  ex_disjunction struct{}
type     disjunction struct{ valbase ; val Value }
func (p *disjunction) kind() Kind { return KindDisjunction }
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
				// Strip location wrappers ONLY for paths during concatenation!
				// This bubbles the path up to fuse with `.c` while preserving braces for lists.
				if _, isPath := v.(*path); isPath && (len(a) > 0 || len(tail) > 0) {
					res = append(res, com_qualword(ctx, dup(a), append([]Value{v}, tail...))...)
				} else {
					res = append(res, com_qualword(ctx, append(dup(a), &loc{v, t.pos}), tail)...)
				}
			}
			return
		case *xloc:
			for _, v := range com_qualword(ctx, nil, []Value{t.Value}) {
				// CRITICAL FIX: Strip location wrappers ONLY for paths during concatenation!
				if _, isPath := v.(*path); isPath && (len(a) > 0 || len(tail) > 0) {
					res = append(res, com_qualword(ctx, dup(a), append([]Value{v}, tail...))...)
				} else {
					res = append(res, com_qualword(ctx, append(dup(a), &xloc{v, t.pos}), tail)...)
				}
			}
			return
		case *compound:
			for _, v := range com(ctx, nil, t.elems) {
				res = append(res, com_qualword(ctx, append(dup(a), v), tail)...)
			}
			return
		case *qualword:
			// Unpack ONLY explicit qualwords to flatten the AST safely.
			res = append(res, com_qualword(ctx, dup(a), append(t.elems, tail...))...)
			return
		case *path:
			if len(t.elems) == 0 {
				res = append(res, com_qualword(ctx, dup(a), tail)...)
				return
			}
			if len(t.elems) == 1 {
				res = append(res, com_qualword(ctx, dup(a), append([]Value{t.elems[0]}, tail...))...)
				return
			}

			firsts := com_qualword(ctx, dup(a), []Value{t.elems[0]})
			lasts := com_qualword(ctx, nil, append([]Value{t.elems[len(t.elems)-1]}, tail...))

			for _, f := range firsts {
				for _, l := range lasts {
					p := &path{elements{make([]Value, len(t.elems))}}
					p.elems[0] = f
					for j := 1; j < len(t.elems)-1; j++ {
						p.elems[j] = t.elems[j]
					}
					p.elems[len(t.elems)-1] = l
					res = append(res, p)
				}
			}
			return
		default:
			val := expand(ctx, elem)

			if val != nil && val != elem {
				switch val.(type) {
				case *disjunction, *loc, *xloc, *compound, *path:
					// CRITICAL RE-INJECTION: We must bounce structural wrappers (like *xloc)
					// back to the top of the function so the switch can peel them open and find the path!
					res = append(res, com_qualword(ctx, dup(a), append([]Value{val}, tail...))...)
					return
				}
			}

			// For all other types (like *list), append them natively.
			// This preserves {} formatting and prevents stack overflow loops!
			a = append(a, val)
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

type is_plain_only struct{}
type plain_ctx struct { Context ; only bool }
func (p plain_ctx) inner() Context { return p.Context }
func (p plain_ctx) cast(t reflect.Type) Context { return icast(p, t) }
func (p plain_ctx) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case is_plain_only: return p.only
    }
    return p.Context.do(ctx, op)
}

// Value returned by (plain) modifier.
type plain struct { elements ; name Symbol }
func (_ *plain) kind() Kind { return KindPlain }
func (p *plain) String() (s string) {
    s = "{=plain"
    if t := p.name.String(); t != "" { s += "("+t+")" }
    for _, v := range p.elems { s += " " + v.String() }
    s += "}"
    return
}

type is_plainline struct {}
type plainline_ctx struct { Context }
func (p plainline_ctx) inner() Context { return p.Context }
func (p plainline_ctx) cast(t reflect.Type) Context { return icast(p, t) }
func (p plainline_ctx) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case is_plainline: return true
    }
    return p.Context.do(ctx, op)
}

type plainline struct { elements }
func (_ *plainline) kind() Kind { return KindPlainLine }
func (p *plainline) String() (s string) {
    s = "{=plainline"
    if p.elems != nil {
        s += " "
        for _, v := range p.elems { s += v.String() }
    }
    s += "}"
    return
}
func (p *plainline) float(ctx Context) (_ float64) {
    if p.len() > 0 { return __float(ctx, p.elems[0]) }
    return
}
func (p *plainline) int(ctx Context) (_ int64) {
    if p.len() > 0 { return __int(ctx, p.elems[0]) }
    return
}

type plainint struct{}
func (p *plainint) evaluate(ctx Context, args ...Value) (_ Value) {
    var res = &plain{}
    var exe = _execution(ctx)
    var opts struct { general_opts }

    if args = parseOpts(ctx, &opts, args...) ; len(args) > 0 {
        res.name = __symbol(ctx, args[0])
        exe.language = res.name
    }

    for _, recipe := range exe.recipes {
        res.elems = append(res.elems, expand(_final(ctx), recipe))
    }

    if false && len(res.elems) == 1 {
        if x, y := res.elems[0].(*plainline); y {
            res.elems = merge(x.elems...)
        }
    }

    if checkpoints {
        p.evaluate_check(ctx, args, exe.recipes, res)
    }
    return res
}

func multiline(ctx Context, recipes... Value) (res string) {
    var (
        x = len(recipes)-1
        w = new(bytes.Buffer)
    )
    for n, recipe := range recipes {
        if fmt.Fprint(w, __string(ctx, recipe)); n < x { fmt.Fprint(w, "\n") }
    }
    res = w.String()
    return
}

type XML struct { Value }
func (p *XML) String() string { return "(xml " + p.Value.String() + ")" }
func (p *XML) _cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*XML); ok {
        assert(ok, "value is not XML")
        res = cmp(ctx, p.Value, a.Value)
    }
    return
}

/*
<books number="3">
  <book id="1">
    <title>book one</title>
  </book>
  <book id="2">
    <title>book two</title>
  </book>
  <book id="3"> <title>  abc  </title> </book>
</books>

Converted into:

(
    books number=3
    (
        book id=1
        (title 'book one')
    )
    (
        book id=2
        (title 'book two')
    )
    (
        book id=3
        (title '  abc  ')
    )
)

   TODO: implement the new xml format:

   xml{books(number=3
       book(id=1 title('book one'))
       book(id=2 title('book two'))
       book(id=3 title('   abc   '))
   )}

*/
func DecodeXML(ctx Context, source string, ws bool) (result Value) {
    var (
        pos = _pos(ctx)
        stack []*group
        nodes []*group
        tok enc_xml.Token
        err error
    )
    xd := enc_xml.NewDecoder(strings.NewReader(source))
    for tok, err = xd.Token(); err == nil; tok, err = xd.Token() {
        switch elem := tok.(type) {
        case enc_xml.ProcInst:
            // TODO: ...
        case enc_xml.StartElement:
            nn := _group(pos, _word(pos, intern(elem.Name.Local)))
            for _, a := range elem.Attr {
                var k, v Value
                k = _word(pos, intern(a.Name.Local))
                v = &strlit{valbase{pos},a.Value}
                if s := a.Name.Space; s != "" {
                    k = _group(pos, &strlit{valbase{pos},s}, k)
                }
                nn.append(makePair(k, v))
            }
            if x := len(stack); x > 0 {
                stack[x-1].append(nn)
            } else {
                nodes = append(nodes, nn)
            }
            stack = append(stack, nn)
        case enc_xml.EndElement:
            if x := len(stack); x > 0 {
                stack = stack[0:x-1]
            } else {
                // FIXME: report illegal xml
            }
        case enc_xml.CharData:
            if x := len(stack); x > 0 {
                node, s := stack[x-1], string(elem)
                if ws {
                    node.append(&strlit{valbase{pos},s})
                } else {
                    if s = strings.TrimSpace(s); s != "" {
                        node.append(&strlit{valbase{pos},s})
                    }
                }
            }
        case enc_xml.Directive:
            // TODO: ...
        case enc_xml.Comment:
            // TODO: ...
        }
    }
    if x := len(nodes); x > 1 {
        g := _group(pos)
        for _, node := range nodes {
            g.append(node)
        }
        result = g
    } else if x == 1 {
        result = nodes[0]
    }
    if err != io.EOF {
        erro(ctx, "%v", err)
    }
    return
}

type xml struct { whitespace bool }
func (p *xml) evaluate(ctx Context, args ...Value) (result Value) {
    var source = multiline(ctx, _execution(ctx).recipes...)
    if v := DecodeXML(ctx, source, p.whitespace); v != nil {
        return &XML{ v }
    } else {
        return &XML{ _none(_pos(ctx)) }
    }
}

type JSON struct { Value }
func (p *JSON) String() string { return "(json " + p.Value.String() + ")" }
func (p *JSON) _cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*JSON); ok {
        assert(ok, "value is not JSON")
        res = cmp(ctx, p.Value, a.Value)
    }
    return
}

type jsonDecodeState struct {
    dec *enc_json.Decoder
    stack []*group
    nodes []*group
}
func (ds *jsonDecodeState) decode() {}

/*
   TODO: implement the new json format:

   json{books(number=3
       book(id=1 title('book one'))
       book(id=2 title('book two'))
       book(id=3 title('   abc   '))
   )}
 */
func DecodeJSON(ctx Context, source string) (result Value) {
    var (
        pos Pos = _pos(ctx)
        stack []*group
        nodes []Value
        node *group
        value Value
        t, v enc_json.Token
        s string
        err error
		symArray = intern("array")
		symObject = intern("object")
    )
    jd := enc_json.NewDecoder(strings.NewReader(source))
LoopJSON:
    for {
        if t, err = jd.Token(); err != nil { break }
        x := len(stack)
        //prompt(ctx, "%T: %v\n", t, t)
    SwitchNodeType:
        switch node, value = nil, nil; d := t.(type) {
        case enc_json.Delim:
            switch d {
            case '[':
                nn := _group(pos, _word(pos, symArray))
                if x == 0 {
                    nodes = append(nodes, nn)
                } else {
                    node, value = stack[x-1], nn
                }
                stack = append(stack, nn) // APPEND
                break SwitchNodeType
            case '{':
                nn := _group(pos, _word(pos, symObject))
                if x == 0 {
                    nodes = append(nodes, nn)
                } else {
                    node, value = stack[x-1], nn
                }
                stack = append(stack, nn) // APPEND
                break SwitchNodeType
            case '}':
                if x == 0 {
                    err = errorIllJson; break LoopJSON
                }
                if k := stack[x-1].at(0); k == nil {
                    if s = __string(ctx, k); s != coreSymbols[int(symObject)] {
                        err = errorIllJson; break LoopJSON
                    }
                }
                stack = stack[0:x-1] // POP
                continue LoopJSON
            case ']':
                if x == 0 {
                    err = errorIllJson; break LoopJSON
                }
                if k := stack[x-1].at(0); k == nil {
                    if s = __string(ctx, k); s != coreSymbols[int(symArray)] {
                        err = errorIllJson; break LoopJSON
                    }
                }
                stack = stack[0:x-1] // POP
                continue LoopJSON
            default:
                err = errorIllJson; break LoopJSON
            }
        case string:
            var sv = &strlit{valbase{pos},d}
            if x == 0 {
                nodes = append(nodes, sv)
                break
            }

            node = stack[x-1]
            if k := node.at(0); k != nil {
                var kind string
                if kind = __string(ctx, k); kind == coreSymbols[int(symArray)] {
                    node.append(sv); continue
                } else if kind != coreSymbols[int(symObject)] {
                    err = errorIllJson; break LoopJSON
                }
            }

            // Get value token
            if !jd.More() {
                err = errorIllJson; break LoopJSON
            } else if v, err = jd.Token(); err != nil {
                break LoopJSON
            }

            switch vd := v.(type) {
            case enc_json.Delim:
                var vn *group
                switch vd {
                case '[': vn = _group(pos, _word(pos, symArray))
                case '{': vn = _group(pos, _word(pos, symObject))
                default: err = errorIllJson; break LoopJSON
                }
                stack = append(stack, vn)
                node.append(makePair(sv, vn))
            case string:
                node.append(makePair(sv, _strlit(pos, vd)))
            case float64:
                node.append(makePair(sv, _float(pos, vd)))
            case nil: // null
                node.append(makePair(sv, _word(pos, intern("null"))))
            default:
                err = errorIllJson; break LoopJSON
            }
            //prompt(ctx, "node: %v\n", node)
        case float64:
            if v := Value(_float(pos, d)); x == 0 {
                nodes = append(nodes, v)
            } else {
                node, value = stack[x-1], v
            }
        case nil: // null
            if v := Value(_word(pos, intern("null"))); x == 0 {
                nodes = append(nodes, v)
            } else {
                node, value = stack[x-1], v
            }
        default:
            err = errorIllJson; break LoopJSON
        }
        if node != nil && value != nil {
            if k := node.at(0); k != nil {
                if s = __string(ctx, k); s != coreSymbols[int(symArray)] {
                    err = errorIllJson; break LoopJSON
                }
            }
            node.append(value)
        }
    }
    if x := len(nodes); x == 1 {
        result = nodes[0]
    } else {
        g := _group(pos)
        for _, v := range nodes {
            g.append(v)
        }
        result = g
    }
    if err != io.EOF {
        erro(ctx, "%v", err)
    }
    return
}

type json struct {}
func (_ *json) evaluate(ctx Context, args ...Value) (result Value) {
    var recipes = _execution(ctx).recipes
    var source = multiline(ctx, recipes...)
    if v := DecodeJSON(ctx, source); v != nil {
        return &JSON{ result }
    } else {
        return &JSON{ _none(recipes[0].Pos()) }
    }
}

type YAML struct { Value }
func (p *YAML) String() string { return "(yaml " + p.Value.String() + ")" }

/*
   TODO: implement the yaml format:

   yaml{books(number=3
       book(id=1 title('book one'))
       book(id=2 title('book two'))
       book(id=3 title('   abc   '))
   )}
 */
func DecodeYAML(ctx Context, source string, ws bool) (result Value) {
    erro(ctx, "TODO: implement DecodeYAML")
    return
}

type yaml struct { whitespace bool }
func (p *yaml) evaluate(ctx Context, args ...Value) (result Value) {
    var source = multiline(ctx, _execution(ctx).recipes...)
    if v := DecodeYAML(ctx, source, p.whitespace); v != nil {
        return &YAML{ result }
    } else {
        return &YAML{ _none(_pos(ctx)) }
    }
}

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

// Returns []Symbol, meaning ZERO string allocations and ZERO filepath parsing!
func splitFileName(ctx Context, val Value) (dir, name Symbol) {
	if f, _ := to_file(val); f != nil {
		// Pure Symbol domain using the new __sym helpers
		return __symPathJoin(f.dir, f.sub), f.name
	}

	// 2. Pure Symbol-Domain Splitting for raw sequences
	seq := symbolize(val)

	// Scan backwards for the last symSlash
	for i := len(seq) - 1; i >= 0; i-- {
		if seq[i] == symSlash {
			if i == 0 {
				// Path is "/foo" -> dir is "/", name is "foo"
				return internSeq(seq[:1]), internSeq(seq[1:])
			}
			// Path is "a/b/foo" -> dir is "a/b", name is "foo"
			return internSeq(seq[:i]), internSeq(seq[i+1:]) // Zero allocation slice pointers!
		}
	}

	// 3. No slash found. Path is "foo" -> dir is ".", name is "foo"
	return symDot, internSeq(seq)
}

// __symDir returns the directory portion of a Symbol as a new Symbol.
func __symDir(sym Symbol) Symbol {
	seq := getSymSeq(sym)

	// Scan backwards for the last symSlash
	for i := len(seq) - 1; i >= 0; i-- {
		if seq[i] == symSlash {
			if i == 0 {
				return internSeq(seq[:1]) // Path is "/foo" -> dir is "/"
			}
			return internSeq(seq[:i])     // Path is "a/b/foo" -> dir is "a/b"
		}
	}
	return symDot
}

// __symBase returns the file name portion of a Symbol as a new Symbol.
func __symBase(sym Symbol) Symbol {
	seq := getSymSeq(sym)

	for i := len(seq) - 1; i >= 0; i-- {
		if seq[i] == symSlash {
			return internSeq(seq[i+1:])
		}
	}
	return sym // No slash found, the base is the whole symbol!
}

// __symPathJoin normalizes and joins symbols just like filepath.Join,
// but entirely within the zero-allocation Symbol domain!
func __symPathJoin(syms ...Symbol) Symbol {
	// First pass: Calculate max capacity to avoid reallocations
	var cap int
	var isAbs bool

	for _, sym := range syms {
		if sym != symEmpty {
			seq := getSymSeq(sym)
			cap += len(seq) + 1 // +1 for potential slashes
			if !isAbs && len(seq) > 0 && seq[0] == symSlash {
				isAbs = true
			}
		}
	}
	if cap == 0 { return symDot }

	out := make([]Symbol, 0, cap)
	if isAbs { out = append(out, symSlash) }

	for _, sym := range syms {
		if sym == symEmpty { continue }
		seq := getSymSeq(sym)

		var segStart = 0
		for i := 0; i <= len(seq); i++ {
			// A segment boundary is either a symSlash or the end of the sequence
			if i == len(seq) || seq[i] == symSlash {
				if i > segStart {
					seg := seq[segStart:i]

					// Evaluate the exact segment
					isDot := len(seg) == 1 && seg[0] == symDot
					isDotDot := (len(seg) == 1 && seg[0] == symDotDot) ||
					            (len(seg) == 2 && seg[0] == symDot && seg[1] == symDot)

					if isDot {
						// Segment is "./" -> do nothing
					} else if isDotDot {
						// Segment is "../" -> Try to pop the parent directory
						lastSlash := -1
						for j := len(out) - 1; j >= 0; j-- {
							if out[j] == symSlash {
								lastSlash = j
								break
							}
						}

						var lastSeg []Symbol
						if lastSlash == -1 {
							lastSeg = out
						} else {
							lastSeg = out[lastSlash+1:]
						}

						lastIsDotDot := (len(lastSeg) == 1 && lastSeg[0] == symDotDot) ||
						                (len(lastSeg) == 2 && lastSeg[0] == symDot && lastSeg[1] == symDot)

						if !lastIsDotDot && len(lastSeg) > 0 {
							// Pop the parent!
							if lastSlash == -1 {
								out = out[:0]
							} else {
								if lastSlash == 0 && isAbs {
									out = out[:1] // Keep the root slash
								} else {
									out = out[:lastSlash]
								}
							}
						} else if !isAbs {
							// Can't pop, so append ".."
							if len(out) > 0 && out[len(out)-1] != symSlash {
								out = append(out, symSlash)
							}
							out = append(out, symDot, symDot) // Store as [., .] for consistency
						}
					} else {
						// Normal segment (e.g. "foo.txt", "*.h", "inc")
						if len(out) > 0 && out[len(out)-1] != symSlash {
							out = append(out, symSlash)
						}
						out = append(out, seg...)
					}
				}
				segStart = i + 1
			}
		}
	}

	if len(out) == 0 {
		if isAbs { return symSlash }
		return symDot
	}

	return internSeq(out)
}

// isSeqPrefix performs an O(1) bounds-checked integer prefix comparison.
func isSeqPrefix(sSeq, pSeq []Symbol) bool {
	if len(pSeq) > len(sSeq) { return false }
	for i := range pSeq {
		if sSeq[i] != pSeq[i] { return false }
	}
	return true
}

// __symHasPrefix checks if one Symbol conceptually starts with another Symbol's sequence.
func __symHasPrefix(sym, prefix Symbol) bool {
	if sym == prefix { return true }
	if prefix == symEmpty { return true }
	return isSeqPrefix(getSymSeq(sym), getSymSeq(prefix))
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
    if truly(ctx, wants_fullfile{}) { return fullfile{f} }
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

type must_stamp_file struct{ *file }
type stamp_file_ctx struct{ Context }
func (c stamp_file_ctx) cast(t reflect.Type) Context { return icast(c,t) }
func (c stamp_file_ctx) inner() Context { return c.Context }
func (c stamp_file_ctx) do(ctx Context, op any) any {
    switch op.(type) {
    case must_stamp_file: return true
    }
    return c.Context.do(ctx, op)
}

type statinfo struct{
    file *file
    next *statinfo
}

func (si *statinfo) mod() (res time.Time) {
	var maxNano int64

	// O(1) pure integer loop: no interface methods, no time.Time allocations
	for p := si; p != nil; p = p.next {
		// p.file.exists() safely ensures filebase isn't nil and _mtime != 0
		if p.file != nil && p.file.exists() {
			if p.file._mtime > maxNano {
				maxNano = p.file._mtime
			}
		}
	}

	// Only construct the Go time.Time struct ONCE at the very end if a file exists
	if maxNano > 0 {
		// Unix(sec, nsec) seamlessly converts our UnixNano back to a time.Time
		res = time.Unix(0, maxNano)
	}

	return // returns zero time.Time{} if no files existed
}

func (si *statinfo) exists() (res existence) {
	res = existenceMatterless

	for p := si; p != nil; p = p.next {
		if p.file != nil { // matterless is nil file
			// p.file.exists() inherently checks p._mtime != 0 now!
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

type filestub struct {
	dir     Symbol    // Holographically shreds machine paths.
	sub     Symbol    // Substitute, hidden. Portable: e.g., intern("src/core")
	name    Symbol    // Name, representation. Portable: e.g., intern("main.c")
	filemap *filemap
	other   *filestub
}

type filebase struct {
	stub         filestub    // cycled-list of file stubs of different projects
	_updatedDeps []Value     // any updated deps (Moved up for pointer alignment)
	_mtime       int64       // ModTime().UnixNano() (0 means non-existent)
	_size        int64       // Size() in bytes
	_travin      int32       // Graph traversal state
	_traved      int32       // Graph traversal state
	_dirty       int32       // Graph dirty state
	_updated     bool        // true if this file has been updated
	_isDir       bool        // Mode().IsDir()
}

func (p *filebase) exists() bool { return p != nil && p._mtime != 0 }

type file struct{
	valbase
	*filebase
	*filestub
}

func (p *file) String() string {
	// Fast string concatenation using the Stringer interface
	return "{=file " + p.name.String() + "}"
}

func (p *file) fullname() string {
	return filepath.Join(p.dir.String(), p.sub.String(), p.name.String())
}

func (p *file) basename() (s string) {
	return filepath.Base(p.name.String())
}

func (p *file) searchInMatchedPaths(ctx Context, proj *project) (res bool) {
	if p.filemap != nil {
		var f = p.filemap.stat(ctx, p.name.String())
		if f != nil && f.exists() {
			// Sync the primitive data directly
			p._mtime, p._size, p._isDir = f._mtime, f._size, f._isDir
			res = true
		}
	}
	return
}

func (p *file) stamp(ctx Context) (res []*file) {
	var fn = p.fullname()
	if fn == "" {
		erro(ctx, "file `%s` has no fullname", p.name)
	}

	var e error
	var info os.FileInfo

	if info, e = os.Stat(fn); e != nil {
		p._mtime = 0 // Mark as non-existent
		if truly(ctx, must_stamp_file{p}) {
			ctx = pc(pc(ctx, strings.TrimSuffix(fn, ".x")), fn+".log")
			if _, y := e.(*fs.PathError); y {
				erro(ctx, "no such file: %s", p.name)
			} else {
				erro(ctx, "%v", e)
			}
		}
		return
	} else if info == nil {
		p._mtime = 0
		if truly(ctx, must_stamp_file{p}) {
			ctx = pc(pc(ctx, strings.TrimSuffix(fn, ".x")), fn+".log")
			erro(ctx, "no such file: %s", p.name)
		}
		return
	}

	// Extract to primitives!
	p._mtime = info.ModTime().UnixNano()
	p._size  = info.Size()
	p._isDir = info.IsDir()

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
	if p.exists() {
		// good already
	} else if info, err := os.Stat(p.fullname()); err == nil {
		// Unpack to primitives
		p._mtime = info.ModTime().UnixNano()
		p._size  = info.Size()
		p._isDir = info.IsDir()
	} else if x, y := err.(*fs.PathError); y {
		if false {
			erro(ctx, "%v: %v", trimPromptString(x.Path), x.Err)
		}
		return
	} else {
		erro(ctx, "file.stat: %v", err)
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

func (p *file) change(dir, sub, name Symbol) (okay bool) {
	// 1. Fast Path: If the O(1) symbols match exactly, we are already on the right stub!
	if p.dir == dir && p.sub == sub && p.name == name {
		return true
	}

	// 2. OS-Boundary Check: Pure Symbol Equality!
	// We join the parts into a single Symbol ID and compare the integers.
	if __symPathJoin(dir, sub, name) == __symPathJoin(p.dir, p.sub, p.name) {
		var head = &p.filebase.stub
		for stub := p.filestub; stub != nil; stub = stub.other {
			// Blazing fast 3-cycle integer check!
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

type stat_dir struct { string }
type stat_sub struct { string }
type stat_nonexist struct { bool }
type stat_fileinfo struct{ os.FileInfo }

func _stat(ctx Context, a0 any, aa ...any) (_ *file) {
	var sub, dir, name Symbol
	var fileInfo os.FileInfo
	var nonexist bool

	// 1. Immediately enter the Symbol Domain!
	if v, ok := a0.(Value); ok {
		name = __symbol(ctx, v)
	} else {
		name = intern(__string(ctx, a0))
	}

	for _, a := range aa {
		switch t := a.(type) {
		case *project: dir = intern(t.absPath)
		case stat_dir: dir = intern(t.string)
		case stat_sub: sub = intern(t.string)
		case stat_fileinfo: fileInfo = t.FileInfo
		case stat_nonexist: nonexist = t.bool
		default: erro(ctx, "invalid stat arg: %v", ts(a,ctx))
		}
	}

	var u = _universe(ctx)
	var fullname Symbol

	// 2. Cleaned Path Resolution Logic (Pure Sequence Domain)
	if filepath.IsAbs(name.String()) {
		fullname = name
		fullSeq := getSymSeq(fullname)
		dirSeq  := getSymSeq(dir)

		if dir != symEmpty && isSeqPrefix(fullSeq, dirSeq) && len(fullSeq) > len(dirSeq) && fullSeq[len(dirSeq)] == symSlash {
			tailSeq := fullSeq[len(dirSeq)+1:]
			if sub == symEmpty {
				name = internSeq(tailSeq)
			} else {
				subSeq := getSymSeq(sub)
				if isSeqPrefix(tailSeq, subSeq) && len(tailSeq) > len(subSeq) && tailSeq[len(subSeq)] == symSlash {
					name = internSeq(tailSeq[len(subSeq)+1:])
				}
			}
		} else {
			dir = symEmpty
		}
	} else if filepath.IsAbs(sub.String()) {
		fullname = __symPathJoin(sub, name)
		if dir == symEmpty {
			dir = sub
			sub = symEmpty
		} else if sub == dir {
			sub = symEmpty
		} else {
			if __symHasPrefix(sub, dir) {
				dirSeq := getSymSeq(dir)
				subSeq := getSymSeq(sub)

				if len(subSeq) > len(dirSeq) && subSeq[len(dirSeq)] == symSlash {
					sub = internSeq(subSeq[len(dirSeq)+1:])
				} else {
					sub = internSeq(subSeq[len(dirSeq):])
				}
			} else {
				debug(ctx,
					_f("conflicted sub/dir: %s", fullname.String()),
					_f("sub=%v", sub),
					_f("dir=%v", dir),
					callstack{num:16}, trace{})
			}
		}
	} else if filepath.IsAbs(dir.String()) {
		fullname = __symPathJoin(dir, sub, name)
	} else {
		dir = __symPathJoin(intern(_workdir(ctx)), dir)
		fullname = __symPathJoin(dir, sub, name)
	}

	cleanFullnameStr := fullname.String()

	// 3. First Pass: Fast lock to retrieve or initialize the cache entry shell
	u.statmutex.Lock()
	base, exists := u.statcache[cleanFullnameStr]
	if !exists {
		// Explicitly map fields to avoid zero-initialization errors
		base = &filebase{stub: filestub{dir: dir, sub: sub, name: name}}
		base.stub.other = &base.stub
		u.statcache[cleanFullnameStr] = base
	}
	u.statmutex.Unlock()

	// 4. Heavy I/O: os.Stat executes concurrently
	if base._mtime == 0 && fileInfo == nil { fileInfo, _ = os.Stat(cleanFullnameStr) }

	// 5. Second Pass: Re-acquire lock to safely update the shared base and stubs
	u.statmutex.Lock(); defer u.statmutex.Unlock()

	// 6. Extraction: Drop the FileInfo interface and map primitives to the struct!
	if fileInfo != nil && base._mtime == 0 {
		base._mtime = fileInfo.ModTime().UnixNano()
		base._size  = fileInfo.Size()
		base._isDir = fileInfo.IsDir()
	}

	if base._mtime == 0 && !nonexist { return nil }

	var head = &base.stub
	var stub *filestub

	if checkpoints {
		for stub = head; stub != nil; stub = stub.other {
			s1 := cleanFullnameStr
			s2 := __symPathJoin(stub.dir, stub.sub, stub.name).String()
			if s1 != s2 {
				debug(ctx,
					_f("fullname '%s' conflicted", cleanFullnameStr),
					_f("panic: (%s, %v, %v) %s", stub.dir.String(), stub.sub.String(), stub.name.String(), s1),
					_f("panic: (%s, %s, %s) %s", dir.String(), sub.String(), name.String(), s2),
					callstack{num:16}, trace{})
			}
			if stub.other == head { break }
		}
	}

	// 7. The Ultimate Cache Retrieval: Blazing fast O(1) Integer Math!
	for stub = head; stub != nil; stub = stub.other {
		if stub.dir == dir && stub.sub == sub && stub.name == name {
			return &file{valbase{_pos(ctx)}, base, stub}
		}
		if stub.other == head { break }
	}

	// If no matching stub was found, link a new one
	stub = &filestub{dir: dir, sub: sub, name: name, other: head.other}; head.other = stub
	return &file{valbase{_pos(ctx)}, base, stub}
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
		if false { erro(ctx, "flag name is trivial") }
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
    for i := strings.IndexAny(s, escaperChars); i != -1; {
        if _, err = buf.WriteString(s[:i]); err != nil {
            panic(err) // erro(ctx, "%v", err)
            return
        }

        var esc string
        switch s[i] {
        case '"':  esc = `\"`
        case '\r': esc = `\r`
        case '\n': esc = `\n`
        }
        if _, err = buf.WriteString(esc); err != nil {
            panic(err) // erro(ctx, "%v", err)
            return
        }

        s = s[i+1:]
        i = strings.IndexAny(s, escaperChars)
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
	if p == nil { return "" }

    var strs []string
    for _, elem := range p.elems {
        if elem != nil {
			if s := elem.String(); s != "" {
				strs = append(strs, s)
			}
		}
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
                    erro(ctx, "unsupported name type: %T %v", name, name)
                }
            }
        }
        if w != nil {
            switch w.s {
            case intern("plain"), intern("json"), intern("yaml"), intern("xml"):
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
    l   token // left paren
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
    case   *def: s += x.name.String()
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
				if t := project_resolve(ctx, intern(ident(ctx, x))); t != nil { x = t }
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
				if t := closure_resolve(ctx, intern(ident(ctx, x))); t != nil { x = t }
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
			// erro(ctx, "'%v' is not value type (%T)", a, a)
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
            erro(ctx, "no path name for `%s`", val)
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
            erro(ctx, "no path name for `%s`", val)
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
func _if_cmp[T Value](c Context, r cmpres, t, f T) T { if cmp(c, t, f) == r { return t } else { return f } }
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
	case *raw: return t.s
	case *word: return t.s.String()
	case *regexpat: return t.Regexp.String()
	case *file: return t.filestub.name.String()
	case *filestub: return t.name.String()
	case *project: return t.name.String()
	case self: return t.name.String()
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
			} else if cmp(ctx, val, v.(Value)) != cmpEqual {
				res = __string(ctx, val)
			}
		}
	case fullname:
		if v := t.Value; v != nil {
			if x, y := v.(*file); y {
				if x == nil {
					erro(ctx, "nil file")
				} else if x.filestub == nil {
					erro(ctx, "nil file stub")
				}
				return x.fullname()
			}
			return __string(ctx, v)
		}
	case *conjunction: // see also $(join ...)
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
				if x, y := d.x.(*builtin); y && x.name == symForeach {
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
			res += u.project.name.String()
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
	case self: return t.name != symEmpty
	case *project: return t.name != symEmpty
	case *file: return t.filestub.name != symEmpty
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
		s := t.s.String()
		switch s {
		case "false", "False", "FALSE", "no", "No", "NO": return false
		case "true", "True", "TRUE", "yes", "Yes", "YES": return true
		}
		return t.s != symEmpty && s != ""
	case *arrow, *closure, *delegate:
		if v = expand(ctx, t); !equal(ctx, v, t) {
			if checkpoints {
				if v.String() == t.String() {
					erro(pc(ctx,t), "%v %v (%v)", t, v, equal(ctx, v, t))
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
	case *barefile: if t.file != nil && t.file.exists() { return t.file._size }
	case *list: if t.len() > 0 { return __int(ctx, t.elems[0]) }
	case *plain: if t.len() > 0 { return __int(ctx, t.elems[0]) }
	case *plainline: if t.len() > 0 { return __int(ctx, t.elems[0]) }
	case negative: if t.Value != nil && !__true(ctx, t.Value) { return 1 }
	case flag: if t.Value != nil { return -__int(ctx, t.Value) }
	case *raw:
		if i, e := strconv.ParseInt(t.s, 10, 64); e == nil {
			return i
		} else {
			erro(ctx, "%v", e)
		}
	case *word:
		if i, e := strconv.ParseInt(t.s.String(), 10, 64); e == nil {
			return i
		} else {
			erro(ctx, "%v", e)
		}
	case *strlit:
		if i, e := strconv.ParseInt(t.s, 10, 64); e == nil {
			return i
		} else {
			erro(ctx, "%v", e)
		}
	case *strcomp:
		if i, e := strconv.ParseInt(__string(ctx, t), 10, 64); e == nil {
			return i
		} else {
			erro(ctx, "%v", e)
		}
	case *strval:
		if i, e := strconv.ParseInt(__string(ctx, t), 10, 64); e == nil {
			return i
		} else {
			erro(ctx, "%v", e)
		}
	case *compound:
		if n := len(t.elems); n > 0 {
			if i, y := t.elems[0].(*decimal); y {
				switch n {
				case 1: res = i.int64
				case 2:
					if w, y := t.elems[1].(*word); y {
						if  (w.s == intern("st") && i.int64%1 == 0) ||
							(w.s == intern("nd") && i.int64%2 == 0) ||
							(w.s == intern("rd") && i.int64%3 == 0) ||
							(w.s == intern("th")) { res = i.int64 }
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
					erro(pc(ctx,t), "%v %v (%v)", t, v, equal(ctx, v, t))
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
				erro(ctx, "%v", e)
			}
		}
	case *raw:
		if f, e := strconv.ParseFloat(t.s, 64); e == nil {
			return f
		} else {
			erro(ctx, "%v", e)
		}
	case *word:
		if f, e := strconv.ParseFloat(t.s.String(), 64); e == nil {
			return f
		} else {
			erro(ctx, "%v", e)
		}
	case *strlit:
		if f, e := strconv.ParseFloat(t.s, 64); e == nil {
			return f
		} else {
			erro(ctx, "%v", e)
		}
	case *strcomp:
		if f, e := strconv.ParseFloat(__string(ctx, t), 64); e == nil {
			return f
		} else {
			erro(ctx, "%v", e)
		}
	case *strval:
		if f, e := strconv.ParseFloat(__string(ctx, t), 64); e == nil {
			return f
		} else {
			erro(ctx, "%v", e)
		}
	case *arrow, *closure, *delegate:
		if v = expand(ctx, t); !equal(ctx, v, t) {
			if checkpoints {
				if v.String() == t.String() {
					erro(pc(ctx,t), "%v %v (%v)", t, v, equal(ctx, v, t))
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
		case symAuto:
			scope.force_collapse = true
			scope.skip_expansion = map[int]bool{-1: true} // Skip ALL arguments. Let __auto handle them!
		case symForeach, symGrep:
			scope.force_collapse = true
			scope.skip_expansion = map[int]bool{1: true} // Skip expanding the locally-bound body argument
		case symAddprefix, symAddsuffix, symJoin, symConjunct:
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
					// Only rebuild if the resolved target actually exists!
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
                        if v := closure_resolve(ctx, intern(str)); v != nil { o = v }
                    default:
                        erro(pc(ctx,t), "%v %v %v", o, t.t, ss)
                    }
                case truly(ctx, delegate_t{}):
                    switch t.t {
                    case SELECT_PROG1, SELECT_PROG2:
                        if v := project_entry(ctx, o); v != nil { o = v }
                    case SELECT_PROP:
                        if v := project_resolve(ctx, intern(str)); v != nil { o = v }
                    default:
                        erro(pc(ctx,t), "%v %v %v", o, t.t, ss)
                    }
                default:
                    erro(pc(ctx,t), "%v %v", o, ss)
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
		if t.String() == `{&(target.arch)}{&(target.sub)}-{&(target.vendor)}-{&(target.sys)}-{&(target.abi)}` {
			var v = t.elems[0].(*disjunction).val
			debug(ctx, "%v %v, %v %v", t, vals, v, __string(ctx, v))
		}
        if res = ease(pc(ctx,v), vals); checkpoints && false { check(ctx, res, v) }
        return
	case *qualword:
        // Expand elements, but we do NOT use com() because dots are strict boundaries,
        // not whitespace-separated compound elements.
        var vals = com_qualword(&comctx{ctx, 0}, nil, t.elems)
        if res = ease(ctx, vals); checkpoints { check(ctx, res, v) }
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
	case *conjunction:
		// 1. Expand the children
		sep := expand(ctx, t.sep)
		vals := expands(ctx, t.list.elems...)

		// 2. Check if they are fully expanded (no closures remaining)
		_, dynamicVals := _redis_elems(vals)
		_, dynamicSep := _redis(sep)

		// 3. If still dynamic, preserve the conjunction
		if dynamicVals || dynamicSep {
			return &conjunction{list{elements{vals}}, sep}
		}

		// 4. FAST PATH: All expanded! Eagerly collapse into a compound.
		var valid []Value
		for _, v := range vals {
			if !isEmpty(v) {
				valid = append(valid, v)
			}
		}

		if len(valid) == 0 {
			// Using t.list.Pos() to safely return a positional null
			return _null(t.list.Pos())
		}

		c := &compound{}
		c.app(valid[0])

		if sep != nil {
			for _, v := range valid[1:] {
				c.app(sep, v)
			}
		} else {
			for _, v := range valid[1:] {
				c.app(v)
			}
		}

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
	case *defcaps:
		// Unbox the defcaps to its embedded base Value (the primary capture string)
		if false { debug(ctx, "%v", t) }
		return v//expand(ctx, t.Value)
    case *valbase, *answer, *boolean, *binary, *def, *none, *null, *punct, *word, *globmeta, *octal, *decimal, *hexadecimal, *escaped, *raw, *regexpat, *project, *file, self, undef, nil:
        if false && v == nil { debug(pc(ctx,v), "%v", v) } //, *modification
        return v
    default:
        if checkpoints { erro(pc(ctx,v), "%v", ts(v,ctx), callstack{stop:"smart.runcase"}) }
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
    case *project: return t.resolve(ctx, intern(s))
    case self: return t.resolve(ctx, intern(s))
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
		sym := intern(s)
        for _, u := range t.list {
            if !u.opts.noVars {
                if o := u.project.Lookup(sym); o != nil {
                    vals = append(vals, o)
                }
            }
        }
        return ease(ctx, vals)
    default:
        erro(pc(ctx,v), "cannot sel: %v %v", ts(t, ctx), s)
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
			erro(ctx, "no such field: %s.builtinbase", _v.Elem().Type())
		} else if f.CanAddr() {
			b := (*builtinbase)(unsafe.Pointer(f.Addr().Pointer()))
			b.evocation = ctx
		} else if f = _v.Elem().FieldByName("evocation"); !f.IsValid() {
			erro(ctx, "no such field: %s.evocation", _v.Elem().Type())
		} else if f.CanSet() {
			f.Set(reflect.ValueOf(ctx))
		} else if f.CanAddr() && f.Addr().CanSet() {
			f.Addr().SetPointer(unsafe.Pointer(ctx))
		} else {
			erro(ctx, "cannot set field: %s.evocation", _v.Elem().Type())
		}

		if o != nil { ctx.o = _opts(ctx, _v, o) }

		if x, y := _v.Interface().(builtin_x); y {
			res = ease(ctx, x.x())
		} else {
			erro(pc(ctx,x), "no method: %v", t.t.Name())
		}

	case *uselist:
		if len(t.list) == 0 {
			return nil
		}

		var targets []Value
		visited := make(map[*project]bool)

		// Local closure to traverse the dependency DAG safely
		var collect func(ul *uselist)
		collect = func(ul *uselist) {
			for _, usee := range ul.list {
				// Break circular dependencies (A -> B -> A)
				if visited[usee.project] {
					continue
				}
				visited[usee.project] = true

				if entry := usee.project.main; entry != nil {
					targets = append(targets, entry)
				}

				// Optional: If you need to deeply resolve nested uselists instantly
				// if sub, ok := usee.project.Lookup("uselist").(*uselist); ok {
				// 	collect(sub)
				// }
			}
		}

		// Start the traversal
		collect(t)

		// Execute all collected entry points
		if len(targets) == 0 { return nil }
		return ease(ctx, targets)

	case *closure, *delegate, *word, *raw, *strlit, *compound, *qualword, *globpat, *arrow, flag:
		do(ctx, _not_evoker{})

    default:
		if x != nil && !isTrivial(x) {
			erro(pc(ctx,x), "%s", ts(x,ctx), callstack{num:10})
		}
    }
	return
}

type executer interface { execute(Context, ...Value) []Value }
type evaler interface { eval(Context, []Value, []Value) Value }
type eval struct { accumulation bool ; o origin }
func (p *eval) evaluate(ctx Context, args ...Value) (_ Value) {
    var exe = _execution(ctx)
    if exe == nil {
        erro(ctx, "wrong eval context: %v", ts(ctx))
    }

    var list []Value
    var opts struct { general_opts }
    args = parseOpts(_final(ctx), &opts, args...)

    for _, recipe := range exe.recipes {
        var vals = merge(recipe)

        if n := len(vals); n < 1 {
            if false { list = append(list, recipe) }
            continue
        }

        var op = vals[0]
        var ov []Value // opt-vals
        if a, y := op.(*argumented); y { op, ov = a.Value, a.args }

        switch t := op.(type) {
        case *returner:
            return ease(ctx, t.vals)

        case evaler:
            if v := t.eval(ctx, ov, vals[1:]); v != nil {
                if p.accumulation {
                    list = append(list, v)
                } else {
                    list = []Value{ v }
                }
            }

        case executer:
            if a := t.execute(ctx, vals[1:]...); a != nil {
                if p.accumulation {
                    list = append(list, a...)
                } else {
                    list = a
                }
            }

        case *undetermined:
            if p.accumulation {
                list = append(list, vals...)
            } else {
                list = vals
            }

        default:
            if p.o != 0 {
                vals = expands(ctx, vals...)
            }
            if p.accumulation {
                list = append(list, vals...)
            } else {
                list = vals
            }
        }
    }

    return ease(ctx, list)
}

// Note that it's is also used with Sscanf.
const (
    fmtExitStatus = "exit status %d"
    maxPromptStr = 48
    maxWorkers = 3
    maxRetries = 1
)

type exec_opts struct {
    general_opts
    logname *fullname "log"
    forRecipe Value `forrecipe,forrecipes,for-recipe,for-recipes`
    forStdout Value `forstdout,for-stdout,for-out`
    forStderr Value `forstderr,for-stderr,for-err`
    result    Value `result,return`
    removeOnFail bool `drop-fail,drop-failure,remove-failure,remove-on-fail`
    zeroStatusErrors bool `zero-status-errors`
    zeroErrs    bool `no-error,no-errors,zero-errors` // require zero error scaned from STDERR
    report      bool `report,report-stamp,verbose-stamp`
    silent      bool `silent,silent-errors` // silent errors
    stdin       bool `stdin,input`
    stdoutBuf   bool `stdout`
    stderrBuf   bool `stderr`
    stdoutTie   bool `tie-out,tie-stdout` // tied with log
    stderrTie   bool `tie-err,tie-stderr` // tied with log
    scanStdout  bool `scan-stdout,scan-out`
    scanStderr  bool `scan-stderr,scan-err`
    scanInfos   bool `scan-infos`
    parallel    bool `parallel,no-order`
    path        bool `path`
    prompt      bool `prompt,msg`
    promptSrc   bool `prompt-src,prompt-source,verbose-source`
    note        bool `note`
    cmd         string `cmd`
    tie         string `tie` // all, both, stdout, stderr, out, err
    _workdir    string `cd,dir,workdir,work-dir,work-directory`
}

type exitstatus struct { int }
func (p *exitstatus) Error() string { return fmt.Sprintf(fmtExitStatus, p.int) }

var (
    defaultShell = "bash"
    udots = []byte("…")

    workingMutex = new(sync.Mutex)
    working atomic.Value // number of working executions

    stdout = &std_writer{io:os.Stdout}
    stderr = &std_writer{io:os.Stderr}

    rxExitStatus        = regexp.MustCompile(`^exit status (\-?[0-9]+)$`)
    rxFileNotFound      = regexp.MustCompile(`'(.+?)' file not found$`)
    rxCodeLinePanic     = regexp.MustCompile(`^([^:]+?):(\d+):(\d+): *((?:fatal )?error|warning): *(.+)$`)
    rxIgnoringDirectory = regexp.MustCompile(`^(ignoring (?:duplicate|nonexistent) directory) "(.*?)"`)
    rxLdManyMinVersions = regexp.MustCompile(`^(?:[^:]+?: )+(passed two min versions \((.+?)\) for platform macOS\. Using (.+)\.)`)

    rxArNoMembers     = regexp.MustCompile(`ar: no archive members specified`)
    rxArNoSuchFileDir = regexp.MustCompile(`ar: (.+?): No such file or directory`)

    rxShellNoSuchFileDir = regexp.MustCompile(`^bash:(?: line ([0-9]+?):)? (.+?): No such file or directory`)

    rxGitNotRepo = regexp.MustCompile(`^fatal: (not a git repository): '(.+?)'`)

    rxDockerCannotConnect   = regexp.MustCompile(`Cannot connect to the Docker daemon at (.*?)\. Is the docker daemon running\?`)
    rxDockerConNotRunning   = regexp.MustCompile(`Error response from daemon: (Container (.+?) is not running)`)
    rxDockerNoSuchContainer = regexp.MustCompile(`Error.*: No such container: (.*)`)
    rxDockerNetworkNotFound = regexp.MustCompile(`Error.*: (network (.*) not found)\.`)

    rxIncludedFrom     = regexp.MustCompile(`In file included from (.+?):(\d+):(?:(\d+):)?`)
    rxPyFileLineIn     = regexp.MustCompile(`^\s*File "(.+?)", line (\d+), in (.+)`)
    rxPyFileNotFound   = regexp.MustCompile(`FileNotFoundError: \[Errno (\d+)\] No such file or directory: '(.*?)'`)
    rxPyModuleNotFound = regexp.MustCompile(`ModuleNotFoundError: No module named '(.*?)'`)

    // ld: warning: passed two min versions (15.0, 23.2) for platform macOS. Using 23.2.
    rxNoticeLines = []*regexp.Regexp{
        regexp.MustCompile(`ld: library '[^']+' not found`),
    }

    rxZeroStatusErrors = map[*regexp.Regexp]struct{}{
        rxShellNoSuchFileDir:struct{}{},
    }

    matchcontexts = map[*regexp.Regexp]func(*exec_buffer, []byte, [][]byte)Context{
        rxCodeLinePanic: func(p *exec_buffer, line []byte, sm [][]byte) Context {
            return p.sc(sm[1], sm[2], sm[3], 0) // TODO: column(line, sm[4])
        },
    }

    commonerrors = map[*regexp.Regexp]func(Context, []byte, [][]byte){
        rxExitStatus: func(c Context, line []byte, sm [][]byte) {
            if string(sm[1]) != "0" { erro(c, "%s", sm[0]) }
        },
        rxShellNoSuchFileDir: func(c Context, line []byte, sm [][]byte) {
            erro(c, "no such command '%s'", sm[2])
        },
        regexp.MustCompile(`(.+?): (.+?):( command)? not found`): func(c Context, line []byte, sm [][]byte) {
            erro(c, "%s: command not found", sm[2])
        },
        regexp.MustCompile(`the input device is not a TTY`): func(c Context, line []byte, sm [][]byte) {
            erro(c, "%s", sm[0])
        },
    }

    // `(?P<first>\d+)\.(\d+).(?P<second>\d+)`
    knownerrors = map[*regexp.Regexp]map[*regexp.Regexp]func(Context, []byte, [][]byte){
        regexp.MustCompile(`^(?:.*/)?clang`):map[*regexp.Regexp]func(Context, []byte, [][]byte){
            rxCodeLinePanic: func(c Context, line []byte, sm [][]byte) {
                t := string(sm[4])
                s := string(sm[5])
                switch t {
                case "warning":
                    debug(c, "%s", s)
                default:
                    erro(c, "%s", s) // "error", "fatal error"
                }
                if m := rxFileNotFound.FindStringSubmatch(s); m != nil {
                    do(c, missing_file{m[1]})
                }
            },

            rxIgnoringDirectory: func(c Context, line []byte, sm [][]byte) {
                debug(pc(c,sm[2]), 5, "%s", sm[1])
            },

            rxLdManyMinVersions: func(c Context, line []byte, sm [][]byte) {
                debug(c, 5, "%s", sm[1])
            },

            regexp.MustCompile(`  +"([^"]+?)", referenced from:`): func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },
            regexp.MustCompile(`undef: *(.+)`): func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },

            regexp.MustCompile(`((?:clang|wasm|(?:[^\.]+\.)?l?ld)(?:\-.+?)?): (error|warning): *(.+)`): func(c Context, line []byte, sm [][]byte) {
                if truly(c, is_configure{}) && string(sm[2]) == "warning" { return }
                debug(c, "%s", sm[0])
            },
            regexp.MustCompile(`((?:clang|wasm|(?:[^\.]+\.)?l?ld)(?:\-.+?)?): could not parse object file (.+?): '(.+)', using libLTO version '(.+?)' file '(.+?)' for architecture (.+)`): func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },
            regexp.MustCompile(`((?:clang|wasm|(?:[^\.]+\.)?l?ld)(?:\-.+?)?): library not found for (.+)`): func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },

            regexp.MustCompile(`(.+?): Too many positional arguments specified!`): func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },
        },
        regexp.MustCompile(`^(?:.*/)?ar`):map[*regexp.Regexp]func(Context, []byte, [][]byte){
            rxArNoSuchFileDir: func(c Context, line []byte, sm [][]byte) {
                debug(c, "'%s' file not found", filepath.Base(string(sm[1])))
            },
            rxArNoMembers: func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },
        },
        regexp.MustCompile(`^(?:.*?bash -c|.*?)git`):map[*regexp.Regexp]func(Context, []byte, [][]byte){
            rxGitNotRepo: func(c Context, line []byte, sm [][]byte) {
                debug(pc(c,sm[2]), "%s", sm[1])
            },
        },
        regexp.MustCompile(`^(?:.*/)?python`):map[*regexp.Regexp]func(Context, []byte, [][]byte){
            rxIncludedFrom: func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },
            rxPyFileLineIn: func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },
            rxPyFileNotFound: func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },
            rxPyModuleNotFound: func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },
        },
        regexp.MustCompile(`^(?:.*/)?docker`):map[*regexp.Regexp]func(Context, []byte, [][]byte){
            rxDockerCannotConnect: func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },
            rxDockerConNotRunning: func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },
            rxDockerNoSuchContainer: func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },
            rxDockerNetworkNotFound: func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },
        },
        regexp.MustCompile(`^(?:.*/)?protoc`):map[*regexp.Regexp]func(Context, []byte, [][]byte){
            regexp.MustCompile(`^(.+?\.proto): File not found\.`): func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },
            regexp.MustCompile(`^(.+?\.proto):(\d+):(\d+): Import "(.+?)" was not found or had errors.`): func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },
            regexp.MustCompile(`^(.+?\.proto):(\d+):(\d+): "(.+?)" is not defined.`): func(c Context, line []byte, sm [][]byte) {
                debug(c, "%s", sm[0])
            },
        },
        regexp.MustCompile(`^(?:.*/)?echo`):map[*regexp.Regexp]func(Context, []byte, [][]byte){
        },
    }
)

func init() { working.Store(0) }

func trimPromptString(str string) string { return trimPromptStringX(str, maxPromptStr) }
func trimPromptStringX(str string, x int) (s string) {
    var segs = strings.Split(str, pathSep)
    if len(segs) <= 1 {
        if n, m := len(str), maxPromptStr; n > m {
            s = "…" + str[n-m:]
        } else {
            s = str
        }
        return
    }

    var i, n int
    for i = len(segs)-1; i >= 0; i -= 1 {
        n += len(segs[i]) + 1
        if n > x {
            var j = i - 1
            if j < 0 { j = i }
            segs[j] = "…"
            s = filepath.Join(segs[j:]...)
            return
        }
    }

    s = str
    return
}

type std_writer struct {
    sync.Mutex
    io io.Writer
    suffixDots bool
}

func (w *std_writer) Write(p []byte) (n int, err error) {
    w.Lock(); defer w.Unlock()
    if w.suffixDots {
        if !bytes.HasPrefix(p, udots) {
            w.io.Write([]byte("\n"))
        }
        w.suffixDots = false
    }
    if n, err = w.io.Write(p); bytes.HasSuffix(p, udots) {
        w.suffixDots = true
    }
    return
}

type exec_log struct {
    sync.Mutex
    writer *bufio.Writer
    filename string
    lines int
}
func (p *exec_log) Write(b []byte) (n int, err error) {
    if p.writer != nil {
		p.Lock(); defer p.Unlock()
		if p.writer != nil {
			p.lines += bytes.Count(b, []byte("\n"))
			n, err = p.writer.Write(b)
		}
	}
    return
}
func (p *exec_log) createWriter(file *os.File, dir, cmd string) {
    p.writer = bufio.NewWriter(file)
    fmt.Fprintf(p, "-*- mode: compilation; default-directory: \"%s\" -*-\n", dir)
    fmt.Fprintf(p, "Compilation started at %v\n\n", time.Now())
    fmt.Fprintf(p, "%s\n", cmd)
}

func is_notice_line(s string) (_ bool) {
    for _, x := range rxNoticeLines {
        if x.MatchString(s) { return true }
    }
    return
}

type exec_buffer struct {
    *exec_ctx

    Tie  io.Writer
    Buf *bytes.Buffer
    line bytes.Buffer // works done line by line
    lnum int // line number

    wrote uint64

    forLine Value
}
func (p *exec_buffer) Write(b []byte) (n int, err error) {
    var expandForLine = p.forLine != nil && !isTrivial(p.forLine)

    if p.Buf != nil {
        if n, err = p.Buf.Write(b); err != nil { return }
    }
    if p.log != nil {
        if _, err = p.log.Write(b); err != nil { return }
    }
    if p.Tie != nil {
        if n, err = p.Tie.Write(b); err != nil { return }
    }
    if err == nil && n == 0 {
        // Returns the number of bytes to avoid "short write" errors.
        // The real bytes written is discarded.
        n = len(b)
    }

    p.wrote += uint64(n)

    scanLine := expandForLine ||
        (p.scanStdout && p == &p.Stdout) ||
        (p.scanStderr && p == &p.Stderr)

    if !scanLine { return }
    if false && truly(p, is_rule{rxConfigRuleHeaders}) {
        note(p, "%s %s", do(p, execution_lang{}), p.sh)
    }

    for slice := b[:]; len(slice) > 0; {
        var i = bytes.Index(slice, []byte("\n"))
        if i == -1 {
            p.line.Write(slice)
            slice = nil
        } else {
            p.lnum += 1
            p.line.Write(slice[:i+1])
            slice = slice[i+1:]

            var line = p.line.Bytes()

            if checkpoints {
                p.check_line(string(line), p.lnum)
            }

            if expandForLine {
                c := p.exec_ctx
                c.line.s = string(line)
                c.lino.int64 = int64(p.lnum)
                v := expand(_final(p.Context), p.forLine)
                if !isNull(v) && is_notice_line(c.line.s) {
                    note(p, "%v : %d. %s → %v", p.forLine, line, c.line.s, ts(v))
                }
            }

            k := func(rx *regexp.Regexp, f func(Context, []byte, [][]byte)) {
                if sm := rx.FindSubmatch(line); sm != nil {
                    if p.zeroStatusErrors && rxZeroStatusErrors != nil {
                        if _, y := rxZeroStatusErrors[rx]; y {
                            p.resetStatusZero = true
                        }
                    }
                    c := Context(p)
                    if x, y := matchcontexts[rx]; y {
                        c = x(p, line, sm)
                    } else {
                        c = p.pc(0)
                    }
                    if !truly(c, is_configure_ignore{rx, sm}) {
                        f(c, line, sm)
                    }
                }
            }
            for rx, f := range commonerrors { k(rx, f) }
            for rx, f := range p.known { k(rx, f) }

            p.line.Reset()
        }
    }
    return
}
func (p *exec_buffer) startDockerDaemon(pos Position, ctx Context, container *project, sock string) (err error) {
    var c = exec.Command("dockerd") //c.Stdout, c.Stderr = stdout, stderr
    if err = c.Run(); err != nil {
        if p.report {
            erro(ctx, "dokcer daemon not running (at %s)", sock)
        }
    } else {
        // TODO: start docker daemon
    }
    return
}
func (p *exec_buffer) filepath(s string) string {
    if p._workdir != "" && !filepath.IsAbs(s) { s = filepath.Join(p._workdir, s) }
    return s
}
func (p *exec_buffer) covpos(s1, s2, s3 string) (pos Position) {
    pos.Filename  = p.filepath(s1)
    pos.Line,   _ = strconv.Atoi(s2)
    pos.Column, _ = strconv.Atoi(s3)
    return
}
func (p *exec_buffer) lpos(column int) Position {
    var pos Position
    if p.log != nil {
        pos.Filename, pos.Line, pos.Column = p.log.filename, p.lnum, column
    }
    return pos
}
func (p *exec_buffer) pc(column int) Context {
    return pc(p, p.lpos(column))
}
func (p *exec_buffer) sc(b1, b2, b3 []byte, column int) Context {
    s1, s2, s3 := string(b1), string(b2), string(b3)
    return pc(pc(p,s1,atoi(s2),atoi(s3)), p.lpos(column))
}

type exec_result struct {
    valbase
    values []Value
    Stdout exec_buffer
    Stderr exec_buffer
    Status int // aka. exit code
}
func (p *exec_result) String() string {
    var s bytes.Buffer
    fmt.Fprintf(&s, "{=exec {=status %d}", p.Status)
    if p.Stdout.Buf != nil { fmt.Fprintf(&s, " {=stdout %v}", p.Stdout.Buf) }
    if p.Stderr.Buf != nil { fmt.Fprintf(&s, " {=stderr %v}", p.Stderr.Buf) }
    fmt.Fprintf(&s, "}")
    return s.String()
}

type is_exec struct{}
type no_exec struct{}
type exec_ctx struct {
    Context

    exec_opts
    exec_result

    line strlit
    lino decimal

    log *exec_log
    logPos Position

    target Value
    targetName string

    retried map[string]bool // work with containerToRun
    containerToRun string   // work with retried
    container *project

    num int

    sh *exec.Cmd
    args []string

    known map[*regexp.Regexp]func(Context, []byte, [][]byte)

    start time.Time

    resetStatusZero bool
}
func (p *exec_ctx) inner() Context { return p.Context }
func (p *exec_ctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p *exec_ctx) ts(t string) (s string) {
    s = "{=" + t
    if p.sh != nil {
        s += " " + filepath.Base(p.sh.Path)
    }
    s += " " + ts(p.Context) + "}"
    return
}
func (p *exec_ctx) do(ctx Context, op any) any {
    switch op.(type) {
    case is_exec: return true
    case wants_fullfile: return p.fullname
    }
    return p.Context.do(ctx, op)
}

func (p *exec_ctx) runContainerAndRetry(exe *execution) (err error) {
    if p.container == nil {
        erro(p.Context, "no container")
    } else if maxRetries < p.num {
        fmt.Fprintf(p.sh.Stderr, "\n---- Retried %d times\n", p.num)
        return
    }

    var (
        name = p.containerToRun
        sh = p.sh
    )

    fmt.Fprintf(sh.Stderr, "\n---- Run container '%s'\n", name)
    if entries := p.container._entries(p.Context, "run", false); entries != nil {
        for _, run := range entries {
            run.execute(p.Context, nil)
        }
    } else {
        erro(p.Context, "%s⇒run undefined", p.container)
    }

    fmt.Fprintf(sh.Stderr, "\n---- Retry the command in %s:", name)
    if false {
        fmt.Fprintf(sh.Stderr, "\n%s:\n    %v", sh.Path, strings.Join(sh.Args, "\n    "))
        fmt.Fprintf(sh.Stderr, "\n\naka:\n    %s", sh)
        fmt.Fprintf(sh.Stderr, "\n----\n")
    } else {
        fmt.Fprintf(sh.Stderr, "\n")
    }

    p.sh = exec.Command(sh.Path, sh.Args[1:]...) // must ignore Args[0]
    p.sh.Stdout, p.sh.Stderr, p.sh.Stdin = sh.Stdout, sh.Stderr, sh.Stdin
    p.sh.Dir, p.sh.Env = sh.Dir, sh.Env
    if err = p.run(exe); err != nil {
        fmt.Fprintf(sh.Stderr, "\n---- Retry failed: %s\n", err)
    }
    return
}

// DEPRECATED
func (p *exec_ctx) DEPRECATED_ensureContainerRunning(containerName string) (err error) {
    var (
        stdoutR, stdoutW = io.Pipe()
        stderrR, stderrW = io.Pipe()
        enviro = os.Environ()
        cmd = exec.Command(`docker`, `ps`,
            `--filter`, `status=running`,
            //`--filter`, fmt.Sprintf(`ancestor=%s`, image),
            `--filter`, fmt.Sprintf(`name=%s`, containerName),
            `--format`, `{{.ID}}\t{{.Image}}\t{{.Names}}`,
        )
        foundID, foundImage string
    )
    cmd.Stdout, cmd.Stderr, cmd.Env = stdoutW, stderrW, enviro
    defer stdoutW.Close()
    defer stderrW.Close()

    go func(r io.Reader) {
        var buf = bufio.NewReader(r)
        for {
            s, e := buf.ReadString('\n')
            if e != nil {
                break
            }
            if fields := strings.Split(s, "\t"); len(fields) == 3 {
                if names := strings.Split(fields[2], ","); len(names) > 0 {
                    foundID, foundImage = fields[0], fields[1]
                    if foundImage == "" { /* FIXME: unused */ }
                }
            }
        }
    } (stdoutR)

    go func(r io.Reader) {
        var buf = bufio.NewReader(r)
        for {
            s, e := buf.ReadString('\n')
            if e != nil {
                break
            }
            prompt(p.Context, "%s", s)
        }
    } (stderrR)

    if err = cmd.Run(); err == nil && foundID == "" {
        if entries := p.container._entries(p.Context, "run", false); entries != nil {
            for _, run := range entries {
                run.execute(p.Context, nil)
            }
        } else {
            erro(p.Context, "%s⇒run undefined", p.container)
        }
    } else if err != nil {
        erro(p.Context, "%v", err)
    }
    return
}

func (p *exec_ctx) skips(tag string) bool {
    if p.retried == nil { p.retried = make(map[string]bool) }
    var a, b = p.retried[tag]
    return a && b
}

func (p *exec_ctx) run(exe *execution) (err error) {
    if p.containerToRun != "" {
        p.retried[p.containerToRun] = true // mark it to skip next time
        err = p.runContainerAndRetry(exe)
        p.containerToRun = ""
        return
    }

    if checkpoints { defer p.run_check(exe) }

    exe.Add(1)
    p.num += 1

    run := func(c *exec.Cmd) {
        defer exe.Done()

        err = c.Run();

        if err == nil {
            err = p.check()
        } else if x, y := err.(*exec.ExitError); y {
            if p.Status = x.ExitCode(); p.Status == 0 { err = p.check() } // success!
            if p.resetStatusZero { p.Status = 0 }
        } else {
            erro(p.Context, "exec failed: %v", err)
            return
        }
    }

    if true { run(p.sh) } else { go run(p.sh) }
    if p.note {
        prompt(p, "%s\n", p.sh)
        if buf := p.Stdout.Buf; buf != nil { prompt(p, "%s", buf) }
        debug(pc(p,auto_get(p,symAt)), "status=%v", p.Status)
    }
    return
}

func (p *exec_ctx) check() (err error) {
    if (!p.silent || p.debug>0) && (/* len(p.scannedDiags) > 0 || */ p.Status != 0 || err != nil) {
        if p.silent /* || p.retStatus */ {
            err = nil
        } else if p.Status != 0 {
            err = &exitstatus{ p.Status } // set or convert error
        }

        var en, wn, in int
        // for _, rec := range p.scannedDiags {
        //     switch rec.dt {
        //     case diagError: en += rec.num
        //     case diagWarn:  wn += rec.num
        //     case diagInfo:  in += rec.num
        //     }
        // }

        var ctx = p.Context
        if en > 0 || p.Status != 0 || err != nil {
            prompt(ctx, "exec: failure (status=%d; err=%v); target=%s\n", p.Status, err, p.targetName)
        } else if wn > 0 {
            prompt(ctx, "%v: %d warnings\n", p.targetName, wn)
        }

        // for i, rec := range p.scannedDiags {
        //     if !p.scanInfos && rec.dt == diagInfo { continue }
        //     if !p.logPos.IsValid() { p.logPos = rec.position }
        //     if i == 0 && !rec.position.same(&rec.position) {
        //         diag(ctx, rec.dt, rec.msg)//
        //     }
        //     if rec.num > 1 {
        //         diag(ctx, rec.dt, `%s (%d)`, rec.msg, rec.num)//
        //     } else {
        //         diag(ctx, rec.dt, rec.msg)//
        //     }
        //     if n := (en+wn+in)-(i+1); i == 8 && 0 < n {
        //         diag(ctx, rec.dt, "%d more...", n)//
        //         break
        //     }
        // }

        var pos = _position(ctx)
        if !p.logPos.IsValid() && p.log != nil {
            p.logPos.Filename = p.log.filename
            p.logPos.Line = p.Stderr.log.lines + 1
        } else {
            p.logPos = pos
        }

        var diffLogPos = !p.logPos.sameLine(&pos)
        var str, _, _ = entryIndicator(ctx, _entry(ctx))
        if (/* !p.retStatus && */ p.Status != 0) || en > 0 {
            if p.removeOnFail {
                if e := os.RemoveAll(p.targetName); e != nil {
                    warn(ctx, "remove: %v", e)
                }
            }

            if diffLogPos && en > 0 { debug(ctx, "%v: %d known errors", str, en) }
            erro(p, "%v: exit status %d", str, p.Status)
        } else if wn > 0 {
            if diffLogPos { warn(ctx, "%v: %d known warnings", str, wn) }
            warn(p, "%v: exit status %d", str, p.Status)
            warn(ctx, "%v: %d known warnings", str, wn)
            warnstack(ctx, 3)
        } else if in > 0 && p.scanInfos {
            if diffLogPos { info(ctx, "%v: %d known messages", str, in) }
            info(p, "%v: exit status %d", str, p.Status)
            info(ctx, "%v: %d known messages", str, in)
            infostack(ctx, 8)
        }

        // if p.retStatus {
        //   if p.zeroErrs && en == 0 && err == nil {
        //     p.vals = append(p.vals, _decimal(p.logPos, int64(p.Status)))
        //   } else {
        //     p.vals = append(p.vals, _none(p.logPos))
        //   }
        // } else if p.Status != 0 || err != nil {
        //   // break
        // }
    }
    return
}

func (ctx *exec_ctx) sources(recipes []Value) (sources []*raw) {
    var a1 *strlit
    var a2 *decimal
    var ac *automatic
    if ctx.forRecipe != nil {
        a1, a2 = &strlit{}, &decimal{}
        ac = &automatic{Context:ctx, defs:make(def_map)}
        ac.args(ac.Context, []Value{a1, a2})
    }

    var pos Pos
    var source string
    for i, recipe := range recipes {
        if !pos.IsValid() { pos = recipe.Pos() }

        var cc Context = _final(pc(ctx, pos))
        var s = __string(cc, recipe)

        if checkpoints {
            ctx.sources_check(cc, i, recipe, s)
        }

        if s = strings.TrimRightFunc(s, unicode.IsSpace); s == "" {
            source += "\n" // an empty line
            continue
        } else {
            // Escape '$$' sequences.
            s = strings.Replace(s, "$$", "$", -1)

            // Duplicate all %
            //s = strings.Replace(s, "%", "%%", -1)

            source += s
        }

        if strings.HasSuffix(source, "\\") {
            source += "\n" // append the line feed
            if i < len(recipes) { continue }
        }

        // Remove tabs in line breakings.
        source = strings.Replace(source, "\\\n\t", "\\\n", -1)
        sources = append(sources, &raw{valbase{pos}, source})

        if ctx.forRecipe != nil {
            a1.pos, a1.s     = pos, source
            a2.pos, a2.int64 = pos, int64(len(sources)+1)
            ac.Context = ctx
            expand(_final(ac), ctx.forRecipe)
        }

        pos, source = 0, ""
    }

    if len(sources) == 0 && 0 < len(recipes) {
        erro(ctx, "empty recipes: %v", recipes)
    }
    return
}

func (ctx *exec_ctx) exec(cmd, opt string) {
    var exe = _execution(ctx)
    var env, sep = exe.env(ctx)
    var envs string
    var logFile *os.File

    for i, s := range env[sep:] {
        if i > 0 { envs += " && " }
        if k := strings.Index(s, "="); k > 0 {
            envs += fmt.Sprintf(`%s%s`, s[:k+1], strconv.Quote(s[k+2:]))
        }
    }

    defer func() {
        if ctx.log != nil && ctx.log.writer != nil { ctx.log.writer.Flush() }
        if logFile != nil { logFile.Close() }
        if false && ctx.log != nil && ctx.log.filename != "" {
            if ctx.Stdout.wrote == 0 && ctx.Stderr.wrote == 0 {
                os.Remove(ctx.log.filename)
            }
        }

        ctx.Stdout.exec_ctx = nil
        ctx.Stderr.exec_ctx = nil
        ctx.container = nil
        ctx.sh = nil

        if !ctx.silent && !is_configurecontext(ctx) && ctx.target != nil {
            var files = stampFile(stamp_file_ctx{ctx}, ctx.target)
            if !ctx.prompt && ctx.report { reportFileUpdates(ctx, files) }
        }

        if !ctx.silent && ctx.prompt {
            var ps = ctx.cmd + trimPromptString(ctx.targetName)
            if _execution(exe.Context) == nil { ps += " …… ok" }
            if ps != "" {
                var s = time.Now().Sub(ctx.start).String()
                if n := ctx.Stdout.wrote; n > 0 { s += fmt.Sprintf(", stdout=%d bytes", n) }
                if n := ctx.Stderr.wrote; n > 0 { s += fmt.Sprintf(", stderr=%d bytes", n) }
                if t := exe.dirt; t != "" { s += "; " + t }
                prompt(ctx, "%s (exec %s)\n", ps, s)
            }
        }
    } ()

    ctx.Stdout.forLine = ctx.forStdout
    ctx.Stderr.forLine = ctx.forStderr

    if ctx.forStdout != nil || ctx.forStderr != nil {
        ac := automatic{Context:ctx.Context, defs:make(def_map)}
        ac.args(ac.Context, []Value{&ctx.line, &ctx.lino})
        if x, y := ac.defs[intern("1")]; y {
            ac.defs[intern("_")] = x // alias
        } else {
            erro(ctx, "wrong args: %v", ac.defs)
        }
        ctx.Context = &ac
    }

    if ctx.stdoutBuf { ctx.Stdout.Buf = new(bytes.Buffer) }
    if ctx.stderrBuf { ctx.Stderr.Buf = new(bytes.Buffer) }
    if ctx.stdoutTie { ctx.Stdout.Tie = stdout }
    if ctx.stderrTie { ctx.Stderr.Tie = stderr }
    if ctx.logname != nil {
        ctx.log = &exec_log{ filename: __string(ctx, ctx.logname) }
    }

    var srcs = ctx.sources(exe.recipes)

    if ctx.log == nil || ctx.log.filename == "" {
        // no log required
    } else if err := os.MkdirAll(filepath.Dir(ctx.log.filename), os.FileMode(0755)); err != nil {
        erro(ctx, "%v", err)
    } else if logFile, err = os.Create(ctx.log.filename); err != nil {
        erro(ctx, "%v", err)
    } else {
        cmdline := joinraws("\n", srcs...)
        ctx.log.createWriter(logFile, ctx._workdir, cmdline)
    }

    ctx.Stdout.exec_ctx = ctx
    ctx.Stderr.exec_ctx = ctx
    ctx.start = time.Now()

    var noExec = truly(ctx, no_exec{})
    for i, src := range srcs {
        if src.trim("@"); src.s == "" { continue }
        if ctx.promptSrc && !ctx.prompt {
            s := src.s
            s = strings.Replace(s, "\n", "\\n", -1)
            s = strings.Replace(s, "\\\\n", "\\\n", -1)
            prompt(ctx, "%s\n", s)
        }

        if cmd == "docker" && len(envs) > 0 { src.s = envs+" && "+src.s }
        if noExec { continue }

        ctx.known = nil
        for rx, m := range knownerrors {
            if rx.MatchString(src.s) { ctx.known = m }
        }
        if ctx.known == nil {
            note(ctx, "unknown: %s", src)
        }

        ctx.sh = exec.Command(cmd, ctx.args...)
        ctx.sh.Env = env
        ctx.sh.Dir = ctx._workdir // always set command work directory
        ctx.sh.Stdout = &ctx.Stdout
        ctx.sh.Stderr = &ctx.Stderr
        if ctx.stdin {
            ctx.sh.Args = append(ctx.sh.Args, "-ti")
            ctx.sh.Stdin = os.Stdin
        }
        if   opt != "" { ctx.sh.Args = append(ctx.sh.Args, opt) }
        if src.s != "" { ctx.sh.Args = append(ctx.sh.Args, src.s) }

        var e = ctx.run(exe)
        if checkpoints { ctx.exec_check(i, src, e) }
        if e != nil || ctx.Status != 0 { break }
    }
}

type executor struct {
    cmd, opt string
    contained bool
}
func (p *executor) evaluate(ctx Context, args ...Value) (result Value) {
    var prog = _program(ctx)
    if prog == nil {
        erro(ctx, "needs program context to exec: %v", ctx)
    }

    if false && truly(ctx, is_test_univ{}) {
        defer note(ctx, "%v %v %v", p.cmd, args, result)
    }

    var cmd = p.cmd
    var ec = &exec_ctx{Context:ctx}
    ec.exec_result.pos = _pos(ctx)
    ec.scanStderr = true

    args = parseOpts(ctx, &ec.exec_opts, args...)

    if !ec.prompt { ec.prompt = ec.cmd != "" }

    var resKind, resType string
    var resValue Value
    var trim = func(s string) string { return s }

    if r := ec.result; r != nil {
        for {
            var x, y = r.(*argumented)
            if !y { break }
            if len(x.args) != 1 {
                erro(pc(ctx,x), "wrong result spec: %v", x)
            }

            switch s := __string(ctx, x.Value); s {
            case "trim": trim = strings.TrimSpace
            default: resType = s
            }

            if l, y := x.args[0].(*list); y {
                if l.len() != 1 {
                    erro(pc(ctx,x), "wrong result spec: %v", x)
                }
                r = l.elems[0]
            }
        }
        if x, y := r.(*pair); y {
            resKind = __string(ctx, x.key)
            resValue = x.val
        } else {
            resKind = __string(ctx, r)
        }
        switch resKind {
        case "stdout": ec.stdoutBuf = true
        case "stderr": ec.stderrBuf = true
        }
    }

    switch ec.tie {
    case "stdout", "out": ec.stdoutTie = true
    case "stderr", "err": ec.stderrTie = true
    case "all", "both":
        ec.stdoutTie = true
        ec.stderrTie = true
    }

    if t := auto_target_value(ctx); patterned(ctx,t) {
        erro(ctx, "target is pattern: %v", ec.target)
    } else {
        ec.target = t
        if _, y := t.(flag); !y {
            ec.targetName, _ = as_fullname_string(ctx, ec.target)
        }
    }

    for i, v := range args {
        var s string
        if i == 0 && p.contained {
            if s = __string(ctx, v); s == "shell" { cmd = defaultShell }
        } else if s = strings.TrimSpace(__string(ctx, v)); s != "" {
            ec.args = append(ec.args, s)
        }
    }

    if p.contained {
        var proj = _project(ctx)
        if proj == nil {
            erro(ctx, "nil project")
        }

        if proj.name == symDotContainer {
            ec.container = proj
        } else if _, sym := proj.find(symDotContainer); sym != nil {
            ec.container = sym.(*project)
        }
        if ec.container == nil {
            erro(ctx, "%s: nil container", proj.name)
        }

        var stringify = func(name string) (str string) {
            var ctx = closure_with(ctx, ec.container.scope)
            if obj := ec.container.resolve(ctx, intern(name)); obj != nil {
                if d, _ := obj.(*def); d != nil {
                    if v := evoke(ctx, d, nil, nil); v != nil {
                        if str = __string(ctx, v); str == "-" {
                            // if v, err = def.DiscloseValue(ec.container); err == nil && v != nil {
                            //   if str, err = __string(ctx, v); str == "" { str = "-" }
                            //   prompt(ctx, "%v: %v (%v)\n", name, str, def)
                            // }
                        }
                    }
                }
            }
            return
        }

        var containerName = stringify("container")
        if containerName == "" {
            erro(ctx, ".container.name undefined")
        }

        var containerImage = stringify("image")
        if containerImage == "" {
            erro(ctx, ".container.image undefined")
        }

        ec.args = append(ec.args, "exec", containerName, cmd)
        cmd = "docker"
    }

    if ec._workdir == "" {
        ec._workdir = prog.workdir(ctx)
        if ec._workdir == "" {
            erro(ctx, "workdir is empty")
        }
    }

    if ec.path {
        if s := filepath.Dir(ec.targetName); s != "" && s != "." && s != "/" {
            if e := os.MkdirAll(s, os.FileMode(0755)); e != nil {
                erro(ctx, "make path '%s' for target failed: %v", s, e)
            }
        }
    }

    ec.exec(cmd, p.opt)

    if ec.result != nil {
        if resValue != nil {
            var s string
            switch resKind {
            case "stdout": s = trim(ec.Stdout.Buf.String())
            case "stderr": s = trim(ec.Stderr.Buf.String())
            case "status": s = fmt.Sprintf("%v", ec.Status)
            }
            switch v := __string(ctx, resValue) == s ; resType {
            case "answer": return _answer(_pos(ctx), v)
            case "option": return _option(_pos(ctx), v)
            default:       return _boolean(_pos(ctx), v)
            }
        } else {
            switch resKind {
            case "stdout": return _raw(_pos(ctx), trim(ec.Stdout.Buf.String()))
            case "stderr": return _raw(_pos(ctx), trim(ec.Stderr.Buf.String()))
            case "status": return _decimal(_pos(ctx), int64(ec.Status))
            }
        }
    }
    return &ec.exec_result
}

var execCommandClang = regexp.MustCompile(`^@?(?:/(?:[^/]*/)+)?(clang(?:\+{2})?)$`)
var execExistFlagPath = map[*regexp.Regexp][]*regexp.Regexp{
    execCommandClang: []*regexp.Regexp{
        regexp.MustCompile(`^-([IL]|include|(?:i(?:(?:framework)?with)|-)?sysroot|(?:cxx-|stdlib(?:\+\+)?)?isystem(?:-after)?|iframework)=?([[:alnum:]_\-/]+)?$`),
    },
}

func correctCommandFlags(ctx Context, source string, w bool) string {
    var flags []string
    var fields = strings.Fields(source)
    if len(fields) > 0 { flags = fields[:1] }

fieldsloop:
    for i := 1; i < len(fields); i += 1 {
        var field = fields[i]

        for rx, rxs := range execExistFlagPath {
            if rx.MatchString(fields[0]) {
                for _, rx := range rxs {
                    var m = rx.FindStringSubmatch(field)
                    if len(m) == 0 { continue }

                    var f bool
                    var s = m[2]
                    if s == "" {
                        if i += 1; i == len(fields) { break fieldsloop }
                        s, f = fields[i], true
                    }

                    if _, e := os.Stat(s); e != nil {
                        if w { debug(ctx, "ignoring nonexistent path: %v", s) }
                        continue fieldsloop // skip nonexistent path flags
                    } else if f {
                        flags = append(flags, field)
                        field = s
                    }
                }
            }
        }

        flags = append(flags, field)
    }
    return strings.Join(flags, " ")
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
	case *word: return t.s.String(), true
	case *strlit: return t.s, true
	case *project: return t.name.String(), true
	case *globmeta: return t.token.String(), true
	case *punct: return t.token.String(), true // (PROOT, PTAIL).String() is empty
	case token: return t.String(), true
	case int: return strconv.Itoa(t), true
	case int64: return strconv.FormatInt(t, 10), true
	case float64: return strconv.FormatFloat(t, 'g', -1, 64), true
	case *decimal: return strconv.FormatInt(t.int64, 10), true
	case *float: return strconv.FormatFloat(t.float64, 'g', -1, 64), true
	case *boolean: if t.bool { return "true", true } else { return "false", true }
	case *file: if t.filestub != nil { return t.filestub.name.String(), true } else { return "", false }
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
					parts = append(parts, _rw(t.p, t.s))
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
		val := _rw(pos, str[:size])

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
								if m, r, rm, s, _, _, wt := forwardGlobComp(ctx, suffix, concat(_rw(pos, str[i:]), gap[k+1:])); m {
									stemParts := concat(gap[:k], _rw(pos, str[:i]))
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
					r = []Value{_rw(vals[iV].Pos(), matchedStr)}
				}

				// CRITICAL FIX: Flatten the stem produced by the DAST explosion!
				// The stem for a '%' must be a single string, not a nested compound of atoms!
				if len(s) > 0 {
					stemStr := ""
					for _, v := range unpack(s[0]) {
						stemStr += getScalarSubstr(ctx, v, 0, -1)
					}
					s[0] = _rw(vals[iV].Pos(), stemStr)
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

						resAtom := _rw(vals[iV].Pos(), valStr[:matchedLen])
						stemAtom := _rw(vals[iV].Pos(), stemStr)

						var nextVals []Value
						hasRem := matchedLen < len(valStr)
						if hasRem {
							nextVals = append(nextVals, _rw(vals[iV].Pos(), valStr[matchedLen:]))
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
		val := _rw(pos, str[len(str)-size:])

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
								if m, r, rm, s, _, _, wt := backwardGlobComp(ctx, prefix, concat(gap[:k-1], _rw(pos, str[:i]))); m {
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
					r = []Value{_rw(vals[iV].Pos(), matchedStr)}
				}

				// CRITICAL FIX: Flatten the stem produced by the DAST explosion!
				// In backward matching, the current stem is appended at the END of the slice.
				if len(s) > 0 {
					idx := len(s) - 1
					stemStr := ""
					for _, v := range unpack(s[idx]) {
						stemStr += getScalarSubstr(ctx, v, 0, -1)
					}
					s[idx] = _rw(vals[iV].Pos(), stemStr)
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

						resAtom := _rw(vals[iV].Pos(), valStr[idx:])
						stemAtom := _rw(vals[iV].Pos(), stemStr)

						var nextVals []Value
						hasRem := idx > 0
						if hasRem {
							nextVals = append(nextVals, _rw(vals[iV].Pos(), valStr[:idx]))
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
							if i < len(str) { testVals = unpack(_rw(targetPos, str[i:])) }
							mSuf, rSuf, remSuf, sSuf, _, _, _ = forwardGlobComp(ctx, sufAtoms, testVals)
						} else {
							mSuf = true
							if i < len(str) { remSuf = unpack(_rw(targetPos, str[i:])) }
						}

						if mSuf {
							var gapAtoms []Value
							if i > 0 { gapAtoms = []Value{_rw(targetPos, str[:i])} }

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
							if i > 0 { testVals = unpack(_rw(targetPos, str[:i])) }
							mPre, rPre, remPre, sPre, _, _, _ = forwardGlobComp(ctx, preAtoms, testVals)
							if !mPre || len(remPre) > 0 {
								if mB, rB, remB, sB, _, _, _ := backwardGlobComp(ctx, preAtoms, testVals); mB {
									mPre, rPre, remPre, sPre = mB, rB, remB, sB
								}
							}
						} else {
							mPre = true
							if i > 0 { remPre = unpack(_rw(targetPos, str[:i])) }
						}

						if mPre {
							var gapAtoms []Value
							if i < len(str) { gapAtoms = []Value{_rw(targetPos, str[i:])} }

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
			res = _rw(val.Pos(), pStr)
			rem = _rw(val.Pos(), vStr[:len(vStr)-len(pStr)])
			return true, res, rem
		}
	} else {
		if strings.HasPrefix(vStr, pStr) {
			res = _rw(val.Pos(), pStr)
			rem = _rw(val.Pos(), vStr[len(pStr):])
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
		switch segs := splitPathStr(ctx, p.name.String()); len(segs) {
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
		switch segs := splitPathStr(ctx, v.name.String()); len(segs) {
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
		if p.Regexp == nil { erro(ctx, "err match: <nil-regexp> %s", ts(val), callstack{num: 10}) }
		if sm := p.Regexp.FindStringSubmatch(__string(ctx, val)); sm != nil {
			res = _rw(val.Pos(), sm[0])
			for _, s := range sm[1:] { stems = append(stems, _rw(val.Pos(), s)) }
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
				erro(ctx, "patterned prefix: %T %v", p.Prefix, p.Prefix)
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
    case *conjunction: return patterned(ctx, t.list.elems) || patterned(ctx, t.sep)
    case *argumented: return patterned(ctx, t.Value) || patterned(ctx, t.args)
    case *quote: return patterned(ctx, t.elems)
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
        erro(pc(ctx,a), "%v", ts(a,ctx))
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
        erro(pc(ctx,a), "%v", ts(a,ctx))
        return
    }
}

func traverse(ctx Context, val Value) {
	switch p := val.(type) {
	case *loc: traverse(ctx, p.Value)
	case *xloc: traverse(ctx, p.Value)
	case *list: for _, e := range p.elems { traverse(ctx, e) }
	case *argumented: if !isTrivial(p.Value) { traverse(p.ctx(ctx), p.Value) }
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
		// CRITICAL FIX: Use the native symAt constant instead of symAt!
		var target = auto_get(ctx, symAt)

		if target == nil {
			erro(ctx, "$@ is not defined")
		}

		if _entry(ctx) == p {
			var proj = _project(ctx)

			if c := cast[*term](ctx); c != nil {
				// ZERO-ALLOCATION MATH: Use symAt here too!
				if t := auto_get(c, symAt); t != nil && eq(ctx, t, target) {
					if true { warn(ctx, "%v: %v: %v\n", proj, p, t) }
					// FIXES: skip traversal as it's closure...
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
	case *closure, *delegate, *arrow:
		if val := expand(ctx,p); !isTrivial(val) {
			updated(ctx, val) // NOTE: ensure that updated flag is correct (see rule.updated)
			traverse(ctx, val)
		} else if _, isArrow := p.(*arrow); isArrow {
			erro(ctx, "traverse trivial arrow: %v: %s", p, ts(p,ctx), callstack{num: 10})
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
		if sym := __symbol(ctx, p.elems[0]); sym == symEmpty {
			erro(ctx, "empty name: %v", p.elems[0])
			// Pass the fast Symbol natively into interpret{}
		} else if truly(ctx, interpret{sym, p.elems[1:]}) {
			modify(ctx, &p.group, true)
		}
	case *modification:
		if e := _execution(ctx); e != nil { e.Wait() }
		for _, m := range p.list { traverse(ctx, m) }
	case *compound, *word, *strlit, *strval, *strcomp, *qualword, *path, *percpat, *globpat, *regexpat, flag:
		do(ctx, act_traverse{p})
	default:
		erro(pc(ctx,val), "unsupported traverse: %v: %s", val, ts(val,ctx), callstack{num:10})
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

	// ADD THE FIX HERE:
	// Reverse the slice in-place to restore original left-to-right order
	for i, j := 0, len(elems)-1; i < j; i, j = i+1, j-1 {
		elems[i], elems[j] = elems[j], elems[i]
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
				default  : v = _rw(pos, s)
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
            v = _rw(pos, s)
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
    case    bare : elems = append(elems, _word(_pos(ctx), intern(string(t))))
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
    case []string: for _, s := range t { elems = append(elems, _strlit(_pos(ctx), s )) }
    case   []bare: for _, s := range t { elems = append(elems, _word(_pos(ctx), intern(string(s)))) }
	case  []*file: for _, f := range t { elems = append(elems, f) }
    default: erro(ctx, "unsupported value: %v", ts(t,ctx), callstack{num:16})
    }

    switch len(elems) {
    case 0 : return _null(_pos(ctx))
    case 1 : return elems[0]
    default: return &list{elements{elems}}
    }
}

func scalarize(v Value) (res Value) {
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
		// If the node ACTUALLY has no position data whatsoever
		s += "="//"0:0"
	}

	if s == "=" { s += t } else if t != "" { s += ":" + t }

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
	case      *file: return "{="+t+" "+x.filestub.name.String()+"}"
	case   fullfile: return "{="+t+" "+x.filestub.name.String()+"}"
	case   *valbase: return "{"+lp(c, x.pos, "")+"}"
	case       *loc: return "{"+lp(c, x.pos, "", x.Value)+"}" // lp handles cc internally
	case      *xloc: return "{"+lp(c, x.pos, "", x.Value)+"}" // lp handles cc internally
	case      *auto: return "{"+lp(c, x.pos, t)+" "+x.name.String()+"}"
	case       *def: return "{"+lp(c, x.pos, t)+" "+x.name.String()+"}"
	case   *project: return "{"+lp(c, x.pos, t)+" "+x.name.String()+"}"
	case       self: return "{"+lp(c, x.pos, t)+" "+x.name.String()+"}"
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
		for _, cap := range x.caps { s += " {"+cap.name.String()+":"+ts(cap.value,cc)+"}" }
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
	case token:
		switch x {
		case PROOT: s = "PROOT"
		case PTAIL: s = "PTAIL"
		default: s = x.String()
		}
		return "{=token "+s+"}"
	case *disjunction:
		return "{"+lp(c, x.pos, t, x.val)+"}"
	case *conjunction:
		return fmt.Sprintf("{=%s %s %s}", t, ts(&x.list, cc), ts(x.sep, cc))

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

		if p, ok := x.(*plain); ok && p.name != symEmpty {
			s += "(" + p.name.String() + ")"
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
func _word(pos Pos, w Symbol) *word { return &word{valbase{pos},w} }
func _raw(pos Pos, s string) Value { if s == "" { return &valbase{pos} } else { return &raw{valbase{pos},s} } }
func _rw(pos Pos, s string) Value {
	if s == "" { return &valbase{pos} }
	if len(s) < 64 {
		vocab.RLock()
		sym, ok := vocab.symbols[s]
		vocab.RUnlock()
		if ok { return &word{valbase{pos},sym} }
	}
	return &raw{valbase{pos},s}
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
func makePunct(pos Pos, t token) *punct { return &punct{valbase{pos}, t} }
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

type evoke_x           struct{ name Symbol }
type evoke_builtin     struct{ name Symbol }
type evoke_def         struct{ name Symbol }
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
        if x, y := p.x.(*builtin); y && (t.name == symEmpty || t.name == x.name) {
            return x
        }
    case evoke_def:
        if x, y := p.x.(*def); y && (t.name == symEmpty || t.name == x.name) {
            return x
        }
    case evoke_x:
        if p.x != nil && (t.name == symEmpty || t.name.String() == ident(ctx, p.x)) {
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
            erro(pc(ctx,p.x), "%s : %s", p.x, ts(p.x))
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

func call(ctx Context, name Symbol, o []Value, a ...Value) (res Value) {
    if v := _universe(ctx).lookup(name); v != nil { res = evoke(ctx, v, o, a) }
    return
}

var         name_prefix = regexp.MustCompile(`^((android|darwin|linux|bsd|ios|windows|mingw|[^~]+)~)(.+)$`)
var illegal_name_prefix = regexp.MustCompile(`^use\.(android|darwin|linux|bsd|ios|windows|mingw|[^~]+)~`)

// A scope maintains a set of objects;
// TODO: remote scope struct, use scopeContext instead
type scope struct {
	mutex sync.Mutex
	elems map[Symbol]object
	project *project
	outer *scope
	comment string
}

func new_scope(outer *scope, owner *project, c string) (s *scope) {
	return &scope{outer:outer, project:owner, comment:c, elems:make(map[Symbol]object)}
}

func (s *scope) hasOuter(outer *scope) bool {
	return s.outer != nil && (s.outer == outer || s.outer.hasOuter(outer))
}

func (s *scope) copyElems() (result map[Symbol]object) {
	s.mutex.Lock(); defer s.mutex.Unlock()
	result = make(map[Symbol]object, len(s.elems))
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
func (s *scope) names() []Symbol {
	s.mutex.Lock() ; defer s.mutex.Unlock()
	var i = 0
	var names = make([]Symbol, len(s.elems))
	for name := range s.elems {
		names[i] = name
		i++
	}
	// sort.Strings(names)
	return names
}

// Lookup returns the object in scope s with the given name if such an
// object exists; otherwise the result is nil.
func (s *scope) Lookup(name Symbol) object {
	s.mutex.Lock() ; defer s.mutex.Unlock()
	return s.lookup(name)
}
func (s *scope) lookup(name Symbol) (obj object) {
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
func (s *scope) find(name Symbol) (res *scope, obj object) {
	if obj = s.lookup(name) ; obj != nil {
		return s,obj
	} else if  s.outer != nil  {
		return s.outer.find(name)
	}
	return
}

func (s *scope) finddef(name Symbol) (d *def) {
	if _, o := s.find(name) ; o != nil { d, _ = o.(*def) }
	return
}

func (s *scope) resolve(name Symbol) (obj object) {
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

	if name := intern(ident(ctx, obj)); name == symEmpty {
		debug(pc(ctx,obj), "no ident: %v", obj)
		return nil
	} else if alt := s.elems[name]; alt != nil {
		return alt
	} else {
		s.replace(ctx, name, obj)
		return nil
	}
}

func (s *scope) replace(ctx Context, name Symbol, obj object) {
	switch o := obj.(type) {
	case interface { setscope(Symbol, *scope) }:
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

func (s *scope) projectName(ctx Context, name Symbol, project *project) (p *project, a object) {
	s.mutex.Lock() ; defer s.mutex.Unlock()
	if a = s.elems[name] ; a == nil {
		p = project
		s.replace(ctx, name, p)
	}
	return
}

func (s *scope) builtin(ctx Context, name Symbol, f reflect.Type) (res *builtin, a object) {
	s.mutex.Lock() ; defer s.mutex.Unlock()
	if a = s.elems[name] ; a == nil {
		res = &builtin{knownobject{objbase{scope:s}, name}, f}
		s.replace(ctx, name, res)
	}
	return
}

func (s *scope) _auto(ctx Context, name Symbol) (a *auto, o object) {
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

func (s *scope) auto(ctx Context, name Symbol) (a *auto) {
	var y bool
	var o object
	if a, o = s._auto(ctx, name); o != nil {
		if a, y = o.(*auto); !y {
			debug(ctx, "name already taken (%s)", typeof(o))
		}
	}
	return
}

func (s *scope) alias(ctx Context, o object, alias ...Symbol) {
	for _, a := range alias { s.elems[a] = o }
}

func (s *scope) _def(ctx Context, o origin, id any, vals ...Value) (d *def, isNew bool) {
	var name string = ident_any(ctx, id)
	if name == "" {
		erro(ctx, "empty name: %s: `%v`", typeof(id), id, callstack{num:16})
	}
	if checkpoints { if illegal_name_prefix.MatchString(name) {
		erro(ctx, "illegal name: %v", name, callstack{num:16})
	}}

	s.mutex.Lock()

	var sym = intern(name)
	if a, y := s.elems[sym]; !y || a == nil {
		d = new(def)
		d.name, d.pos, d.scope = sym, _pos(ctx), s
		s.elems[sym] = d
		isNew = true
	} else {
		d, _ = a.(*def)
	}

	s.mutex.Unlock()

	if o != defInvalid && o != d.o {
		if d.o == defInvalid { d.o = o } else {
			erro(pc(ctx, d), "%v: conflicts origin: %v <> %v", id, d.o, o)
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
func (p *objbase) setscope(name Symbol, s *scope) {
    if p.scope != s {
        if p.scope != nil { delete(p.scope.elems, name) }
        p.scope = s
    }
}

type knownobject struct{ // generally named objects
	objbase
	name Symbol // single, or group name if containing '(*)' and corresponding members
}
func (p *knownobject) kind() Kind { return p.objbase.kind()|KindKnownObject }
func (p *knownobject) String() string { return fmt.Sprintf("{=object %s}", p.name) }
func (p *knownobject) ident(Context) string { return p.name.String() }

// CRITICAL FIX: Directly expose the integer Symbol!
// Any struct that embeds knownobject will automatically inherit this method.
func (p *knownobject) sym() Symbol { return p.name }

// FIXME: locking for MT processing
var usePrepared = make(map[*project]int)

type use struct {
    valbase
    project *project
    params []Value
    opts useopts
}
func (_ *use) kind() Kind { return KindUse }
func (p *use) String() string {
	if len(p.params) == 0 { return p.project.name.String() }
	return fmt.Sprintf("%v(%v)", p.project.name, p.params)
}

type uselist struct {
    owner_ *project
    name Symbol
    scope *scope
    list []*use
}
func (_ *uselist) kind() Kind { return KindUse }
func (p *uselist) owner() *project { return p.owner_ }
func (p *uselist) Pos() (pos Pos) {
    if len(p.list) > 0 { pos = p.list[0].Pos() }
    return
}
func (p *uselist) String() string {
    var s string
    for i, elem := range p.list {
        if i > 0 { s += "," }
        s += elem.project.name.String()
    }
    return fmt.Sprintf("%s", s)
}
func (p *uselist) append(ctx Context, proj *project, params []Value, opts useopts) {
    for _, elem := range p.list {
        if elem.project == proj {
            return
        }
    }
    p.list = append(p.list, &use{valbase{_pos(ctx)},proj,params,opts})
}

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

type def_map map[Symbol]*def
func (m def_map) len() int { return len(m) }
func (m def_map) String() (s string) {
    seen := make(map[Symbol]struct{}) // NOTE: digits alias: 1 2 3...
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

type find_auto struct{ s Symbol }
type set_auto  struct{ o origin; s Symbol; v Value }
type res_auto  struct{ d *def; v Value }
type automatic struct{
    Context
    sync.RWMutex
    defs def_map
    params map[Symbol]*auto
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
func (ac *automatic) amend(ctx Context, name Symbol, val Value) (out *def, res Value) {
    if d, _ := ac.do(ctx, find_auto{name}).(*def); d == nil {
        return ac.set(ctx, defVoid, name, val)
    } else if res = d.value; d.value != val {
        out, d.value = d, val
    }
    return
}
func (ac *automatic) has(s Symbol) (y bool) { _, y = ac.defs[s]; return }
func (ac *automatic) set(ctx Context, o origin, name Symbol, val Value) (out *def, old Value) {
    if name == symDash && val != nil {
        if x, y := val.(*def); y && x.o != defConfig {
            erro(ctx, "set $- to def (%v): %v", x.o, x)
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
    type arg struct{ id, name Symbol ; value Value }

    if vals == nil { return }

    var argn int // setup named/number parameters ($1, $2, etc.)
    var args = make(map[Symbol]*arg, len(vals)) // compact args: combine duplicated pairs

    for i, val := range vals {
        a := &arg{ id: intern(strconv.Itoa(argn+1)) }

        if p, y := val.(*pair); y {
            if a.name = __symbol(ctx, p.key); a.name == symEmpty {
                erro(pc(ctx,a), "empty name: %v", p.key)
                return
            }

            if ac.params != nil {
                if _, y = ac.params[a.name]; !y {
                    var keys = reflect.ValueOf(ac.params).MapKeys()
                    erro(pc(ctx,a), "unknown arg#%d: %v ; known: %v", i, p, keys)
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
            if a.name == symEmpty { a.name = a.id }
        }

        if a.id != a.name { args[a.id] = a }
        args[a.name] = a
        argn += 1

        if d, _ := ac.set(ctx, defParam, a.name, a.value); d == nil {
            erro(ac, "arg '%s' not set", a.name)
            return
        }

        if d, y := ac.defs[a.name]; !y || d == nil {
            erro(ac, "arg '%s' not set", a.name)
            return
        } else if a.id != symEmpty && a.id != a.name {
            ac.Lock()
            ac.defs[a.id] = d // NOTE: alias or replacement
            ac.Unlock()
        }
    }
    return
}

func auto_find(ctx Context, name Symbol) (d *def) {
    d, _ = do(ctx, find_auto{name}).(*def)
    return
}

func auto_get(ctx Context, name Symbol) (_ Value) {
    if d := auto_find(ctx, name); d != nil { return d.value }
    return
}

func auto_set(ctx Context, o origin, name Symbol, val Value) (_ *def, _ Value) {
    t, _ := do(ctx, set_auto{o, name, val}).(res_auto)
    return t.d, t.v
}

type auto struct{ knownobject }
func (a *auto) kind() Kind { return a.knownobject.kind()|KindAuto }
func (a *auto) String() string { return a.name.String() }
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
        erro(ctx, "set auto failed: %v: %v %v", a.name, value, app)
    }
}
func (a *auto) isDigit() bool { return IsDigits(a.name.String()) }
func (a *auto) isPlaceholder() bool { return a.name == symUnderscore }
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
        if IsDigits(t.s.String()) {
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
	if d == nil { return "" }

    var value Value
    {
        // d.Lock()
        value = d.value
        // d.Unlock()
    }

    if s = d.name.String() + d.streq(); value != nil {
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
            erro(pc(ctx,d), "%v: %v → %v", d.name, d.o, o)
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
        erro(pc(ctx,value), "%v: execute command failed: %v", d.name, e)
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
func (p *builtin) String() string { return p.name.String() }
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
        erro(ctx, "execute pattern entry: %v", p.target)
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
func _stems(ctx Context) []Value {
    res, _ := do(ctx, get_stems{}).([]Value)
    return res
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
        name = joinPathSegs(a...)
    } else {
        erro(ctx, "unexpected result: %v", ts(res))
    }
    return // NOTE: also `globMatchFile(ctx, pat, str, true)`
}

func (p *filemap) stat(ctx Context, name string) (res *file) {
    var patts = p.patts
    if len(patts) == 0 {
        erro(ctx, "no map patterns: %v", p)
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
        erro(ctx, "%s → %v", name, p.patts)
    }

    for _, path := range p.paths {
        if isNull(path) {
            erro(ctx, _f("nil path: name=%s",  name), _f("nil path: %v", p))
        } else if isNone(path) {
            debug(ctx, _f("nil path: name=%s",  name), _f("nil path: %v", p))
            continue
        }

        var dir, sub string

        if sub = __string(ctx, path); sub == "" {
            if false {
                erro(ctx, "empty filemap path: %v, patterns=%v", path, patts)
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


// =============================================================================
// 1. Core Data Structures & Interning
// =============================================================================

const mapThreshold = 16

type Symbol uint32

const (
	symEmpty Symbol = iota // ""
	symSpace

	sym_0
	sym_1
	sym_2
	sym_3
	sym_4
	sym_5
	sym_6
	sym_7
	sym_8
	sym_9

	symAmpersand    // "&"
	symDollarSign   // "$"
	symDash         // "-"
	symUnderscore   // "_"
	symApostrophe   // '
	symQuotation    // "
	symColon        // :
	symComma        // ,
	symTilde        // ~
	symDot          // .
	symDotDot       // ..
	symSlash        // /
	symBackslash    // \
	symLparen       // (
	symRparen       // )
	symLbrace       // {
	symRbrace       // }
	symLbrack       // [
	symRbrack       // ]
	symHash         // #

	symEqualSign    //  =   ASSIGN
	symUnshiSign    //  =+  ASSIGN_USH
	symAddeqSign    // +=   ASSIGN_ADD

	symAt           // @
	symAtD          // @D
	symAtF          // @F
	symAtA          // @'
	symBar          // |
	symBarD         // |D
	symBarF         // |F
	symBarA         // |'
	symCaret        // ^
	symCaretD       // ^D
	symCaretF       // ^F
	symCaretA       // ^'
	symLangle       // <
	symLangleD      // <D
	symLangleF      // <F
	symLangleA      /// <'
	symRangle       // >
	symRangleD      // >D
	symRangleF      // >F
	symRangleA      // >'
	symPercent      // %
	symPercentD     // %D
	symPercentF     // %F
	symPercentA     // %'
	symAsterisk     // *
	symAsteriskD    // *D
	symAsteriskF    // *F
	symAsteriskA    // *'
	symQues         // ?
	symQuesD        // ?D
	symQuesF        // ?F
	symQuesA        // ?'
	symPlus         // +
	symPlusD        // +D
	symPlusF        // +F
	symPlusA        // +'

	symAsteriskQues // *?
	symAsteriskAst  // **

	symCWD // Current Work Directory, aliases `$/`
	symCRD // Current Relative Directory, aliases `$.`
	symCTD // Current Temp Directory, aliases `$,`
	symARGS
	symSMART      // aka os.Args[0]
	symSMART_ARGS // aka os.Args[1:], NOTE: "_" is a shredder, SMART_ARGS must be after ARGS and SMART

	symTrue
	symFalse
	symYes
	symNo
	symOn
	symOff

	symOS        // os
	symMode      // mode
	symGoals     // goals
	symSmart     // smart

	symMailto
	symFtp
	symFtps
	symHttp
	symHttps

	symDock
	symShell
	symPython
	symPerl
	symPlain
	symPlainLine
	symText
	symJson
	symXml
	symYaml

	symAssert
	symAppend
	symEval
	symValue
	symConfigure
	symConfiguration

	symAuto
	symAutoload
	symAnswer
	symBool
	symBoolean
	symDefer
	symVar
	symSet
	symDep
	symEnv

	symStr
	symSelf
	symHere
	symWord
	symQuote
	symDefs
	symGlob
	symRegex
	symFullname

	symForeach
	symUnique
	symGrep
	symAddprefix
	symAddsuffix
	symConjunct
	symFilter
	symJoin
	symSelect
	symDebug

	symPrint
	symPrompt
	symPreserve
	symExpand
	symString
	symStringify
	symReveal
	symDisclose
	symClosure

	symCd
	symMkdir
	symSudo
	symFork
	symWait
	symStamp
	symTouch
	symDeps

	symCheck
	symCase
	symCond
	symIf
	symWhere
	symOnce
	symDirty
	symBy

	symTypeof
	symOrigin
	symDefined
	symPosition
	symDate
	symError
	symWarning
	symSure
	symTrace
	symDefor

	symOr
	symAnd
	symNot
	symXor
	symEqual // 'equal', not '='
	symEq
	symNe
	symMatch
	symGreater
	symLess

	symIfeq
	symIfne
	symIfarg
	symIfdef
	symFor
	symCount
	symCall
	symList
	symWhich

	symDivide
	symDiv
	symMultiply
	symMul
	symMinus // 'minus', not '-'

	symElement
	symField
	symFields
	symSplit
	symUses
	symBare
	symPath
	symFinalize
	symResolve
	symStrip
	symTrim
	symExt

	symTitle
	symIndent
	symUppercase
	symLowercase
	symSubst // substitute
	symSubstitute
	symSubstring
	symPatsubst
	symContains
	symBase
	symBase2
	symBase3
	symBase4
	symBase5
	symBase6
	symBase7
	symBase8
	symBase9
	symBases
	symChopdir
	symDir
	symDir2
	symDir3
	symDir4
	symDir5
	symDir6
	symDir7
	symDir8
	symDir9
	symDirs
	symUndir
	symUndir2
	symUndir3
	symUndir4
	symUndir5
	symUndir6
	symUndir7
	symUndir8
	symUndir9
	symUndirs
	symReldir
	symFile
	symStat
	symWildcard
	symPrintf
	symClean
	symGit
	symGitdir

	symChdir
	symRename
	symRemove
	symLink
	symSymlink
	symTruncate
	symReturn

	symRead
	symRelative
	symOut
	symDecode
	symBase64
	symLeft
	symRight
	symPrefix
	symSuffix
	symCopy
	symWrite
	symUpdate
	symInput
	symContainer

	symReadDir
	symRelativeDir
	symNotEqual
	symFilterOut
	symDecodeBase64
	symEncodeBase64

	symTrimLeft
	symTrimRight
	symTrimPrefix
	symTrimSuffix
	symTrimExt

	symSplitQuote
	symSplitQuoteJoin
	symSplitJoinQuote
	symCopyFile
	symTouchFile
	symWriteFile
	symReadFile
	symUpdateFile
	symConfigureInput
	symConfigureFile

	symGitAhead
	symGitModified
	symServeHttp

	symDotBase      // .base
	symDotConfigure // .configure
	symDotContainer // .container
	symDotOS        // .os
	symDotMode      // .mode
	symDotGoals     // .goals
	symDotSmart     // .smart

	symUnderline     = symUnderscore
	symWildcardOne   = symAsterisk // *
	symWildcardChar  = symQues // ?
	symWildcardAny   = symAsteriskAst // **
	symWildcardShort = symAsteriskQues // *?
)

// WARNING: The order of this slice is strictly mapped to the integer consts above.
// ALL atomic primitives (punctuation, keywords) MUST be placed at the top.
// ALL composite words (like "copy-file") MUST be placed at the bottom.
// Do not mix them, or recursive shredding will shift the hardcoded IDs!
var coreSymbols = []string{
	"", " ", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9",

	// --- *START* punctuations ---
	"&", "$", "-", "_", "'", `"`, ":", ",", "~", ".", "..", "/", `\`,
	"(", ")", "{", "}", "[", "]", "#", "=", "=+", "+=",

	"@", "@D", "@F", "@'",
	"|", "|D", "|F", "|'",
	"^", "^D", "^F", "^'",
	"<", "<D", "<F", "<'",
	">", ">D", ">F", ">'",
	"%", "%D", "%F", "%'",
	"*", "*D", "*F", "*'",
	"?", "?D", "?F", "?'",
	"+", "+D", "+F", "+'",
	"*?", "**",
	// --- *END* punctuations ---

	// --- *START* monolithic strings ---
	"CWD", "CRD", "CTD", "ARGS", "SMART", "SMART_ARGS",

	"true", "false", "yes", "no", "on", "off",

	"os", "mode", "goals", "smart",

	"mailto", "ftp", "ftps", "http", "https",

	"dock", "shell", "python", "perl", "plain", "plainline", "text", "json", "xml", "yaml",
	"assert", "append", "eval", "value", "configure", "configuration",

	"auto", "autoload", "answer", "bool", "boolean", "defer", "var", "set", "dep", "env",
	"str", "self", "here", "word", "quote", "defs", "glob", "regex", "fullname",

	"foreach", "unique", "grep", "addprefix", "addsuffix", "conjunct", "filter", "join", "select", "debug",
	"print", "prompt", "preserve", "expand", "string", "stringify", "reveal", "disclose", "closure",
	"cd", "mkdir", "sudo", "fork", "wait", "stamp", "touch", "deps",
	"check", "case", "cond", "if", "where", "once", "dirty", "by",

	"typeof", "origin", "defined", "position", "date", "error", "warning", "sure", "trace", "defor",
	"or", "and", "not", "xor", "equal", "eq", "ne", "match", "greater", "less",
	"ifeq", "ifne", "ifarg", "ifdef", "for", "count", "call", "list", "which",
	"divide", "div", "multiply", "mul", "minus",
	"element", "field", "fields", "split", "uses", "bare", "path", "finalize", "resolve", "strip",
	"trim", "ext",
	"title", "indent", "uppercase", "lowercase", "subst", "substitute", "substring", "patsubst", "contains",
	"base", "base2", "base3", "base4", "base5", "base6", "base7", "base8", "base9", "bases", "chopdir",
	"dir", "dir2", "dir3", "dir4", "dir5", "dir6", "dir7", "dir8", "dir9", "dirs",
	"undir", "undir2", "undir3", "undir4", "undir5", "undir6", "undir7", "undir8", "undir9", "undirs",
	"reldir", "file", "stat", "wildcard", "printf", "clean", "git", "gitdir",
	"chdir", "rename", "remove", "link", "symlink", "truncate", "return",

	"read", "relative", "out", "decode", "base64", "left", "right", "prefix",
	"suffix", "copy", "write", "update", "input", "container",

	// --- *END* monolithic strings ---
	// --- *SEPARATOR* of Atomic and Composite Symbols (ID shifting occurs after this point) ---

	"read-dir", "relative-dir", "not-equal",
	"filter-out", "decode-base64", "encode-base64",
	"trim-left", "trim-right", "trim-prefix", "trim-suffix", "trim-ext",
	"split-quote", "split-quote-join", "split-join-quote",
	"copy-file", "touch-file", "write-file", "read-file", "update-file", "configure-input", "configure-file",
	"git-ahead", "git-modified", "serve-http",

	".base", ".configure", ".container", ".os", ".mode", ".goals", ".smart",
}

func (sym Symbol) String() string {
	vocab.RLock()
	defer vocab.RUnlock()
	if int(sym) < len(vocab.symetas) {
		return vocab.symetas[sym].Text
	}
	return "<invalid-symbol-id>"
}

const (
	NumNaN uint8 = 0 // Raw string, Idx is ignored
	NumInt uint8 = 1 // Idx points to `numbers`
	NumFlt uint8 = 2 // Idx points to `numbers`
	SymSeq uint8 = 3 // Idx points to `sequences`
)

// SymMeta is precisely 24 bytes. Fits nearly 3 per CPU cache line!
type SymMeta struct {
	Text     string // 16 bytes (String header: pointer + length)
	Idx      int32  // 4 bytes (Index into `numbers` OR `sequences` based on Kind)
	Flags    uint8  // 1 byte (Bits for Kind (0-1) and Rank (2-5))
	// 3 bytes implicit compiler padding
}

// Inline helper methods for clean access
func (m SymMeta) Kind() uint8 { return m.Flags & 0x03 }
func (m SymMeta) Rank() int { return int((m.Flags >> 2) & 0x0F) }

type vocabulary struct{
	sync.RWMutex
	symetas   []SymMeta
	numbers   []uint64   // Side-table for numbers!
	sequences [][]Symbol // Side-table for shredded sequences!
	seqsyms map[uint64][]Symbol // maps a sequence hash to a Symbol list (to resolve collisions)
	symbols map[string]Symbol
}

var vocab vocabulary

func init() {
	size := len(coreSymbols) + mapThreshold

	// 1. Initialize globals immediately
	vocab.symbols = make(map[string]Symbol, size)
	vocab.symetas = make([]SymMeta, 0, size)
	vocab.numbers = make([]uint64, 0, size/20)
	vocab.sequences = make([][]Symbol, 0, size/20)
	vocab.seqsyms = make(map[uint64][]Symbol)

	// PASS 1: Reserve the EXACT integer IDs for core constants (0, 1, 2...)
	// This ensures `symDot` is exactly where it belongs, even if later
	// symbols trigger recursive shredding.
	for i, s := range coreSymbols {
		vocab.symbols[s] = Symbol(i)
		vocab.symetas = append(vocab.symetas, SymMeta{Text: s, Idx: -1}) // Placeholder
	}

	// PASS 2: Safely calculate metadata and shred sequences.
	// If "copy-file" recursively calls internLocked("copy"), "copy" will
	// safely be appended to the END of the vocab slice!
	for i, s := range coreSymbols {
		vocab.symetas[i] = makeSymMetaLocked(s, Symbol(i))
	}

	keywords = make(map[Symbol]token, OFF - PROJECT)
	for i := PROJECT; i <= OFF; i++ {
		if s := tokens[i]; s != "" { keywords[internLocked(s)] = i }
	}
}

// internLocked assumes vocabM.Lock() is already held by the caller!
func internLocked(s string) Symbol {
	if sym, ok := vocab.symbols[s]; ok {
		return sym
	}

	// 1. Claim ID and reserve slot immediately
	sym := Symbol(len(vocab.symetas))
	vocab.symbols[s] = sym
	vocab.symetas = append(vocab.symetas, SymMeta{Text: s, Idx: -1}) // Placeholder

	// 2. Evaluate metadata safely
	meta := makeSymMetaLocked(s, sym)

	// 3. Overwrite placeholder
	vocab.symetas[sym] = meta

	// 4. Register newly shredded strings into the fast-path map!
	if meta.Kind() == SymSeq {
		h := hashSeq(vocab.sequences[meta.Idx])
		vocab.seqsyms[h] = append(vocab.seqsyms[h], sym)
	}
	return sym
}

// INLINEABLE: Blazing fast ASCII comparison with zero function overhead, the explicit
// OR-chain above is the fastest instruction sequence the Go compiler can generate
func isShredderChar(ch byte) bool { return ch == '~' || ch == '.' || ch == '-' || ch == '/' || ch == '#' || ch == '_' || ch == '*' || ch == '?' }
func isShredderChar0(ch byte) bool { return strings.IndexByte(shredderChars, ch) >= 0 }

// makeSymMetaLocked isolates the logic so it can be used by both
// internLocked AND the init() Two-Pass bootstrap.
func makeSymMetaLocked(s string, sym Symbol) SymMeta {
	meta := SymMeta{ Text: s, Idx: -1 }
	kind := NumNaN

	// 0. The Ultimate O(1) Atomic Bypass!
	// Explicit bypass for Scanner Tokens and Automatic Variables.
	// Any symbol defined in the const block between & and CWD is strictly atomic.
	if symAmpersand <= sym && sym < symCWD {
		meta.Flags = kind | (uint8(globRank(s)) << 2)
		return meta
	}

	// 1. Fast boundary check for pure numbers
	if len(s) > 0 && (s[0] == '-' || s[0] == '.' || ('0' <= s[0] && s[0] <= '9')) {
		if val, err := strconv.ParseInt(s, 10, 64); err == nil {
			kind = NumInt
			meta.Idx = int32(len(vocab.numbers))
			vocab.numbers = append(vocab.numbers, uint64(val))
		} else if val, err := strconv.ParseFloat(s, 64); err == nil {
			kind = NumFlt
			meta.Idx = int32(len(vocab.numbers))
			vocab.numbers = append(vocab.numbers, math.Float64bits(val))
		}
	}

	// 2. Shredding (Sequences & Natural Number Extraction)
	if kind == NumNaN {
		var start int
		var seq []Symbol
		var didShred bool

		for i := 0; i < len(s); i++ {
			ch := s[i]
			isPunct := isShredderChar(ch)
			isDigit := '0' <= ch && ch <= '9'

			split := false
			if isPunct {
				split = true
			} else if i > start {
				// Detect transition between letters and digits (Natural Sort boundary)
				prev := s[i-1]
				prevIsDigit := '0' <= prev && prev <= '9'
				prevIsPunct := isShredderChar(prev)

				if !prevIsPunct && (isDigit != prevIsDigit) {
					split = true
				}
			}

			if split {
				didShred = true
				if i > start {
					seq = append(seq, internLocked(s[start:i]))
				}
				if isPunct {
					// --- COMPOSITE PUNCTUATION/WILDCARD LOOKAHEAD ---
					if ch == '*' && i+1 < len(s) {
						if s[i+1] == '*' {
							seq = append(seq, symAsteriskAst) // "**"
							i++ // Skip the second '*'
							start = i + 1
							continue
						} else if s[i+1] == '?' {
							seq = append(seq, symAsteriskQues) // "*?"
							i++ // Skip the '?'
							start = i + 1
							continue
						}
					}

					// Standard Punctuation/Wildcards
					switch ch {
					case '~': seq = append(seq, symTilde)
					case '.': seq = append(seq, symDot)
					case '-': seq = append(seq, symDash)
					case '/': seq = append(seq, symSlash)
					case '#': seq = append(seq, symHash)
					case '_': seq = append(seq, symUnderscore)
					case '*': seq = append(seq, symAsterisk)
					case '?': seq = append(seq, symQues)
					}
					start = i + 1 // Punctuation is consumed into a symbol
				} else {
					start = i // A new alpha or digit block starts right here
				}
			}
		}

		if didShred {
			kind = SymSeq
			meta.Idx = int32(len(vocab.sequences))
			if start < len(s) {
				seq = append(seq, internLocked(s[start:]))
			}
			vocab.sequences = append(vocab.sequences, seq)
		}
	}

	meta.Flags = kind | (uint8(globRank(s)) << 2)
	return meta
}

// intern takes a string, registers it if new, and returns its unique Symbol ID.
func intern(s string) Symbol {
	// Fast Path: Bypass Map Hashing for Core Constants
	switch s {
	case "":   return symEmpty
	case "*":  return symWildcardOne
	case "?":  return symWildcardChar
	case "**": return symWildcardAny
	case "*?": return symWildcardShort
	}
	if false && checkpoints {
		if len(s)>1 && strings.ContainsAny(s, shredderChars) {
			panic(`illegal symbol contains any of "`+shredderChars+`": `+s)
		}
	}

	// Zero-allocation fast-path lookup
	vocab.RLock()
	if sym, ok := vocab.symbols[s]; ok {
		vocab.RUnlock()
		return sym
	}
	vocab.RUnlock()

	// Slow-path registration
	vocab.Lock()
	sym := internLocked(s)
	vocab.Unlock()
	return sym
}

// internBytes is optimized for scanner loops to avoid allocating strings.
func internBytes(b []byte) Symbol {
	// Hardware-Level Fast Path
	if len(b) == 0 {
		return symEmpty
	} else if len(b) == 1 {
		switch b[0] {
		case '*': return symWildcardOne
		case '?': return symWildcardChar
		}
	} else if len(b) == 2 && b[0] == '*' {
		switch b[1] {
		case '*': return symWildcardAny
		case '?': return symWildcardShort
		}
	}
	if false && checkpoints {
		if bytes.ContainsAny(b, shredderChars) {
			panic(`illegal symbol contains any of "`+shredderChars+`": `+string(b))
		}
	}

	s := string(b)

	// Zero-allocation map lookup
	vocab.RLock()
	if sym, ok := vocab.symbols[s]; ok {
		vocab.RUnlock()
		return sym
	}
	vocab.RUnlock()

	vocab.Lock()
	sym := internLocked(s)
	vocab.Unlock()
	return sym
}

func getSymSeq(sym Symbol) (seq []Symbol) {
	vocab.RLock()
	meta := vocab.symetas[sym]
	if meta.Kind() == SymSeq {
		seq = vocab.sequences[meta.Idx]
	} else {
		seq = []Symbol{sym}
	}
	vocab.RUnlock()
	return
}

// hashSeq uses the FNV-1a 64-bit algorithm. It is incredibly fast and
// has exceptional distribution for short integer arrays.
func hashSeq(seq []Symbol) uint64 {
	var h uint64 = 14695981039346656037
	for _, s := range seq {
		h ^= uint64(s)
		h *= 1099511628211
	}
	return h
}

// isSeqEqual provides O(1) bounds-checked integer array comparison.
func isSeqEqual(a, b []Symbol) bool {
	if len(a) != len(b) { return false }
	for i := range a {
		if a[i] != b[i] { return false }
	}
	return true
}

// internSeq converts a slice of symbols into a unique Symbol ID with zero allocations on lookup.
func internSeq(seq []Symbol) Symbol {
	if len(seq) == 0 { return symEmpty }
	if len(seq) == 1 { return seq[0] }

	h := hashSeq(seq)

	// 1. Zero-Allocation Fast Path: Hash Lookup (FIXED)
	vocab.RLock()
	for _, sym := range vocab.seqsyms[h] {
		meta := vocab.symetas[sym]
		// MUST verify it is a SymSeq before accessing the sequences slice!
		if meta.Kind() == SymSeq && isSeqEqual(vocab.sequences[meta.Idx], seq) {
			vocab.RUnlock()
			return sym
		}
	}
	vocab.RUnlock()

	// 2. Slow Path: Register New Sequence
	vocab.Lock()
	defer vocab.Unlock()

	for _, sym := range vocab.seqsyms[h] {
		meta := vocab.symetas[sym]
		if meta.Kind() == SymSeq && isSeqEqual(vocab.sequences[meta.Idx], seq) {
			return sym
		}
	}

	var sb strings.Builder
	for _, s := range seq {
		sb.WriteString(vocab.symetas[s].Text)
	}
	str := sb.String()

	// FIXED: Only register to the sequence map if the existing symbol is actually a sequence!
	if sym, ok := vocab.symbols[str]; ok {
		if vocab.symetas[sym].Kind() == SymSeq {
			vocab.seqsyms[h] = append(vocab.seqsyms[h], sym)
		}
		return sym
	}

	sym := Symbol(len(vocab.symetas))
	vocab.symbols[str] = sym

	permSeq := make([]Symbol, len(seq))
	copy(permSeq, seq)

	meta := SymMeta{Text: str, Idx: int32(len(vocab.sequences))}
	meta.Flags = SymSeq | (uint8(globRank(str)) << 2)

	vocab.sequences = append(vocab.sequences, permSeq)
	vocab.seqsyms[h] = append(vocab.seqsyms[h], sym)
	vocab.symetas = append(vocab.symetas, meta)

	return sym
}

// nodeEntry preserves order
type nodeEntry struct {
	k Symbol
	v *valcache
}

// valcache: Hybrid Trie Node (Slice 'o' for order/compactness, Map 'v' for speed)
type valcache struct {
	a []any                // Payload: Rules, Filemaps
	o []nodeEntry          // Primary Storage: Compact & Ordered
	v map[Symbol]*valcache // Acceleration Index: Created only when len(o) >= mapThreshold
}

func (p *valcache) String() (s string) { // NOTE: for debug
	for i, a := range p.a {
		var t string
		switch v := a.(type) {
		case filemap: t = v.String()
		case *rule: t = v.target.String()
		}
		s += fmt.Sprintf("%d:%s,", i, t)
	}
	for _, n := range p.o {
		c, _ := p.get(n.k)
		s += fmt.Sprintf("%v:%v,", n.k, c)
	}
	if s != "" { s = s[:len(s)-1] } // aka. TrimSuffix(s, ",")
	return "{"+s+"}"
}

// =============================================================================
// 2. Valcache Methods
// =============================================================================

// get must also be updated to work natively with Symbols
func (c *valcache) get(sym Symbol) (*valcache, bool) {
	// Fast path: map lookup (O(1) integer hash)
	if c.v != nil {
		n, ok := c.v[sym]
		return n, ok
	}

	// Slow path: linear scan (Extremely fast because it's just comparing uint32s!)
	for _, entry := range c.o {
		if entry.k == sym {
			return entry.v, true
		}
	}
	return nil, false
}

// add natively accepts a Symbol, completely bypassing string hashing and allocations!
func (c *valcache) add(sym Symbol) *valcache {
	if n, ok := c.get(sym); ok {
		return n
	}

	child := new(valcache)
	c.o = append(c.o, nodeEntry{k: sym, v: child})

	if len(c.o) == mapThreshold {
		c.v = make(map[Symbol]*valcache, len(c.o))
		for _, entry := range c.o {
			c.v[entry.k] = entry.v
		}
	} else if len(c.o) > mapThreshold {
		c.v[sym] = child
	}

	return child
}

// =============================================================================
// 3. Path Processing
// =============================================================================

// tokenizePaths parse a path pattern (Extended Glob) into tokens
func tokenizePaths(path string) (results [][][]Symbol) {
	// Expand braces first: foo/{a,b} -> [foo/a, foo/b]
	for _, p := range expandBraces(path) {
		results = append(results, tokenizePath(p))
	}
	return
}

// tokenizePath convert a path into tokens
func tokenizePath(path string) [][]Symbol {
	return tokenizeSegments(strings.Split(path, "/"))
}

// tokenizeSegments translates raw path segments into cached Symbol arrays
func tokenizeSegments(parts []string) [][]Symbol {
	ss := make([][]Symbol, len(parts))

	for i, part := range parts {
		// Optimization: If no meta-chars, intern the whole segment natively
		if !strings.ContainsAny(part, "*?.[") {
			ss[i] = []Symbol{intern(part)}
			continue
		}

		var tokens []Symbol
		start := 0

		for j := 0; j < len(part); {
			c := part[j]
			if c == '*' || c == '?' || c == '.' || c == '[' {
				// Flush preceding literal
				if j > start {
					tokens = append(tokens, intern(part[start:j]))
				}

				if c == '*' {
					// Check for ** or *?
					if j+1 < len(part) {
						if part[j+1] == '*' {
							tokens = append(tokens, symWildcardAny)
							j += 2; start = j; continue
						} else if part[j+1] == '?' {
							tokens = append(tokens, symWildcardShort)
							j += 2; start = j; continue
						}
					}
					tokens = append(tokens, symWildcardOne)
					j++
				} else if c == '?' {
					// Micro-optimization: Bypass intern() entirely for the '?' wildcard
					tokens = append(tokens, symWildcardChar)
					j++
				} else if c == '[' {
					// Capture [a-z] as one token
					end := strings.IndexByte(part[j:], ']')
					if end != -1 {
						// Intern the whole set "[a-z]"
						tokens = append(tokens, intern(part[j:j+end+1]))
						j += end + 1
					} else {
						tokens = append(tokens, intern("["))
						j++
					}
				} else {
					// Capture . (Dots)
					tokens = append(tokens, intern("."))
					j++
				}
				start = j
			} else {
				j++
			}
		}
		// Flush trailing literal
		if start < len(part) {
			tokens = append(tokens, intern(part[start:]))
		}
		ss[i] = tokens
	}
	return ss
}

// expandBraces Recursive Brace Expander (One-pass)
func expandBraces(text string) []string {
	res, _ := expandBracesAt(text, 0)
	return res
}

// expandBracesAt is the recursive core
// It returns the list of expanded strings found at this level, and the index where it stopped.
func expandBracesAt(s string, idx int) ([]string, int) {
	var parts []string // The comma-separated options at this level

	// We need to track the "current working string" for the current comma-option.
	// However, because we might encounter a nested brace {a,b} inside an option,
	// we actually need a list of "current prefixes" that we are building.
	// Let's simplify:
	// The standard way to do this recursively is to parse the *structure* first,
	// then generate the combinations.
	// But to do it in one pass as you asked:

	// Actually, the logic "prefix + middles[n] + suffix" is slightly complex
	// to do purely linearly because 'suffix' hasn't been parsed yet.

	// Better approach for "One Pass":
	// 1. Scan until '{', ',', or '}'.
	// 2. If '{': Recurse. Get [m1, m2]. Cartesian product with current prefixes.
	// 3. If ',': Finish current set of strings, start new set.
	// 4. If '}': Return.

	currentSet := []string{""} // Start with one empty prefix

	i := idx
	for i < len(s) {
		char := s[i]

		if char == '{' {
			// Recursion: parse the content inside {...}
			middles, newIdx := expandBracesAt(s, i+1)
			i = newIdx // Advance to after the matching '}'

			// Cartesian Product: append each middle to each current prefix
			var nextSet []string
			for _, prefix := range currentSet {
				for _, mid := range middles {
					nextSet = append(nextSet, prefix+mid)
				}
			}
			currentSet = nextSet

		} else if char == '}' {
			// Found closing brace for THIS level.
			// We are done with this specific brace block.
			// Return our results and the current index (to let caller continue)
			return combine(parts, currentSet), i

		} else if char == ',' {
			// Found a comma at THIS level.
			// 1. Commit currentSet to parts.
			parts = combine(parts, currentSet)
			// 2. Reset currentSet for the next option
			currentSet = []string{""}

		} else {
			// Literal character
			// Append char to all strings in currentSet
			for k := range currentSet {
				currentSet[k] += string(char)
			}
		}
		i++
	}

	// End of string reached (implicit closing brace)
	return combine(parts, currentSet), i
}

// Helper to merge the final set into the results
func combine(existing []string, current []string) []string {
	if len(current) == 0 { return existing }
	return append(existing, current...)
}

// =============================================================================
// 4. Cache & Uncache Logic
// =============================================================================

func cache(ctx Context, c *valcache, ss [][]Symbol) *valcache {
	for _, segment := range ss {
		for _, token := range segment {
			c = c.add(token) // token is now a Symbol!
		}
	}
	return c
}

func uncache(ctx Context, root *valcache, ss [][]Symbol) (r []*valcache) {
	seen := make(map[*valcache]bool)
	seenShadows := make(map[*valcache]struct{})
	seenAmpNodes := make(map[*valcache]struct{}) // Query-level & edge lock
	var shadowsToSearch []*valcache              // Queue to defer shadow searches

	// Pre-intern the closure symbol so the tight loop doesn't hit the map
	symAmpersand := intern("&")

	fullvalue := do(ctx, fullvalue{}).(Value)
	fullmatch := func(c *valcache) (res bool) {
		if full, exists := seen[c]; exists { return full }
		if res = c.matchPayload(ctx, fullvalue); res { r = append(r, c) }
		seen[c] = res
		return
	}

	const priorDynamicClosureShadowing = true

	var f0 func(*valcache, [][]Symbol, int, int, int) bool
	f0 = func(c *valcache, ss [][]Symbol, i, j, k int) (found bool) {
		if c == nil { return false }

		// 1. Success Condition
		if i == len(ss) { return fullmatch(c) }

		// 2. Segment Boundary
		if j == len(ss[i]) { return f0(c, ss, i+1, 0, 0) }

		s := ss[i][j]
		sStr := s.String() // Retrieve the string representation for character-level math

		// 3. Token Boundary
		if k == len(sStr) { return f0(c, ss, i, j+1, 0) }

		// ---------------------------------------------------------------------
		// DYNAMIC CLOSURE SHADOWING
		// ---------------------------------------------------------------------
		var doDynamicClosureShadowing func()
		if x, y := c.get(symAmpersand); !y { doDynamicClosureShadowing = func() {} } else {
			doDynamicClosureShadowing = func() {
				// CRITICAL FIX: Lock the specific '&' edge!
				if _, ok := seenAmpNodes[x]; ok { return } else { seenAmpNodes[x] = struct{}{} }

				if proj := _project(ctx); proj != nil {
					for _, payload := range x.a {
						if shadow := proj.shadow(ctx, payload); shadow != nil {
							if _, ok := seenShadows[shadow]; !ok {
								seenShadows[shadow] = struct{}{}
								shadowsToSearch = append(shadowsToSearch, shadow)
							}
						}
					}
				}
			}
		}
		if priorDynamicClosureShadowing { doDynamicClosureShadowing() }

		// 4. Input Wildcard (Integer Switching!)
		switch s {
		case symWildcardOne: // "*"
			for _, entry := range c.o {
				if f0(entry.v, ss, i, j, 0) { found = true } // Greedy Consume (Trie)
			}
			if f0(c, ss, i, j+1, 0) { found = true } // Stop (Consume Input)
			return found

		case symWildcardShort: // "*?"
			if f0(c, ss, i, j+1, 0) { found = true } // Stop (Non-Greedy)
			for _, entry := range c.o {
				if f0(entry.v, ss, i, j, 0) { found = true } // Continue
			}
			return found

		case symWildcardAny: // "**"
			for _, entry := range c.o {
				if f0(entry.v, ss, i, j, 0) { found = true } // Greedy Consume
			}
			if f0(c, ss, i, j+1, 0) { found = true } // Stop
			return found

		case symWildcardChar: // "?"
			for _, entry := range c.o {
				keyStr := entry.k.String()
				// Match Literal or Set
				if (len(keyStr) == 1 && !isWildcardMeta(entry.k)) || (len(keyStr) > 2 && keyStr[0] == '[') {
					if f0(entry.v, ss, i, j+1, 0) { found = true }
				}
				// Match Trie Wildcards (*, **, *?)
				if isWildcardMeta(entry.k) {
					if f0(entry.v, ss, i, j+1, 0) { found = true } // Stop
					if f0(c, ss, i, j+1, 0) { found = true }       // Continue
				}
			}
			return found
		}

		// ---------------------------------------------------------------------
		// 5. TRIE WILDCARD LOGIC
		// ---------------------------------------------------------------------

		// A. Compressed Node Match
		if k == 0 && (len(ss[i]) > 1 || s == symWildcardChar) {
			for _, entry := range c.o {
				if isWildcardMeta(entry.k) { continue }

				// Assumes consumeCompressed has been updated to take (Symbol, []Symbol)
				if n, ok := consumeCompressed(entry.k, ss[i][j:]); ok {
					if f0(entry.v, ss, i, j+n, 0) { found = true }
				}
			}
		}

		// B. Literal / Prefix Match
		if k < len(sStr) {
			for _, entry := range c.o {
				key := entry.k
				if isWildcardMeta(key) || key == symWildcardChar { continue }

				// OPTIMIZATION: 1-Cycle Exact Integer Match Bypass
				if k == 0 && key == s {
					if f0(entry.v, ss, i, j+1, 0) { found = true }
					continue
				}

				keyStr := key.String()

				if len(keyStr) > 2 && keyStr[0] == '[' {
					if matchCharSet(keyStr, sStr[k]) {
						if f0(entry.v, ss, i, j, k+1) { found = true }
					}
				} else if len(keyStr) > 0 { // Literal partial prefix match
					if strings.HasPrefix(sStr[k:], keyStr) {
						// Advance k by the length of the matched prefix.
						if f0(entry.v, ss, i, j, k+len(keyStr)) { found = true }
					}
				}
			}
		}

		// D. Trie Wildcards

		if x, y := c.get(symWildcardChar); y {
			if f0(x, ss, i, j, k+1) { found = true }
		}

		// Handle "*" (WildcardOne) in Trie
		if x, y := c.get(symWildcardOne); y {
			if f0(x, ss, i, j, k) { found = true }          // Transition (Match 0)

			nextJ, nextK := j, k+1
			if nextK == len(sStr) { nextJ++; nextK = 0 }

			if nextJ < len(ss[i]) {
				if f0(c, ss, i, nextJ, nextK) { found = true } // Consume (Match 1+)
			} else {
				if f0(x, ss, i, nextJ, nextK) { found = true } // End of Segment
			}
		}

		// Handle "*?" (WildcardShort) in Trie
		if x, y := c.get(symWildcardShort); y {
			if f0(x, ss, i, j, k) { found = true }          // Transition (Match 0) - Prioritized

			nextJ, nextK := j, k+1
			if nextK == len(sStr) { nextJ++; nextK = 0 }

			if f0(c, ss, i, nextJ, nextK) { found = true }  // Consume (Match 1+)
		}

		// Handle "**" (WildcardAny) in Trie
		if x, y := c.get(symWildcardAny); y {
			if f0(c, ss, i+1, 0, 0) { found = true }      // Consume Segment - Prioritized
			if f0(x, ss, i, j, k) { found = true }        // Transition (Match 0)
		}

		// ---------------------------------------------------------------------
		// DYNAMIC CLOSURE SHADOWING
		// ---------------------------------------------------------------------
		if !priorDynamicClosureShadowing { doDynamicClosureShadowing() }
		return found
	}

	// 1. Traverse the Static Trie and collect all static matches
	f0(root, ss, 0, 0, 0)

	// 2. Flush the queue: Traverse the Discovered Shadow Tries
	for _, shadow := range shadowsToSearch {
		f0(shadow, ss, 0, 0, 0)
	}

	return
}

// Helper to instantly identify meta symbols (O(1) CPU cycles)
func isWildcardMeta(sym Symbol) bool {
	return sym >= symWildcardOne && sym <= symWildcardShort
}

// Returns number of tokens consumed (n) and success (ok).
func consumeCompressed(nodeKey Symbol, tokens []Symbol) (int, bool) {
	keyStr := nodeKey.String()
	keyIdx, tokIdx := 0, 0

	for keyIdx < len(keyStr) {
		if tokIdx >= len(tokens) { return 0, false } // Not enough tokens

		tSym := tokens[tokIdx]

		// If input has complex wildcards, abort optimization (let recursion handle it)
		if tSym == symWildcardAny || tSym == symWildcardOne { return 0, false }

		if tSym == symWildcardChar { // "?"
			keyIdx++ // Consumes 1 char of nodeKey
			tokIdx++ // Consumes 1 token
			continue
		}

		tStr := tSym.String()

		// Literal Match (e.g. tStr="z" matches keyStr="zz" at index 0)
		if strings.HasPrefix(keyStr[keyIdx:], tStr) {
			keyIdx += len(tStr)
			tokIdx++
		} else {
			return 0, false
		}
	}
	// Must consume exactly the whole nodeKey
	return tokIdx, true
}

// No changes required! Call this using the string extracted by the caller.
func matchCharSet(pattern string, char byte) bool {
	// Simplified parser for [a-z0-9]
	inner := pattern[1 : len(pattern)-1]
	for i := 0; i < len(inner); i++ {
		if i+2 < len(inner) && inner[i+1] == '-' {
			start, end := inner[i], inner[i+2]
			if char >= start && char <= end { return true }
			i += 2
		} else if inner[i] == char {
			return true
		}
	}
	return false
}

func canStartMatch(c *valcache, segment []Symbol) bool {
	if len(segment) == 0 { return false }

	firstToken := segment[0]

	// 1. Fast Path: Check exact full-token match natively using integer hashing
	if _, ok := c.get(firstToken); ok { return true }

	// 2. Hardware Wildcards
	if _, ok := c.get(symWildcardChar); ok { return true }
	if _, ok := c.get(symWildcardOne); ok { return true }

	firstStr := firstToken.String()
	if len(firstStr) == 0 { return false }

	// 3. Scan edges for partial matches and sets (Bypasses speculative interning!)
	firstByte := firstStr[0]
	for _, entry := range c.o {
		keyStr := entry.k.String()
		if len(keyStr) == 0 { continue }

		// Check for 1-char partial prefix match natively
		if keyStr[0] == firstByte { return true }

		// Check character sets [a-z]
		if keyStr[0] == '[' && matchCharSet(keyStr, firstByte) {
			return true
		}
	}

	return false
}

// =============================================================================

type matched_filemap struct{ filemap ; value Value }
type matched_rule struct{ *rule ; value Value }
func (t matched_filemap) String() string {
	s1, s2 := t.filemap.String(), t.value.String()
	if s1 == s2 { return "{=matched_filemap "+s1+"}" }
	return "{=matched_filemap "+s1+" name="+s2+"}"
}
func (t matched_rule) String() string {
	s1, s2 := t.rule.String(), t.value.String()
	if s1 == s2 { return "{=matched_rule "+s1+"}" }
	return "{=matched_rule "+s1+" name="+s2+"}"
}

func (p *valcache) matchPayload(ctx Context, fullvalue Value) (ok bool) {
	for _, a := range p.a {
		switch a := a.(type) {
		case filemap:
			// Ensure matched == true AND there is no unconsumed remainder
			if matched, res, rem, _ := match(ctx, a.pattern, fullvalue); matched && rem == nil {
				var a = filemap{a._filemap, a.pattern}
				if 0 < do(ctx, matched_filemap{a, res}).(int) {
					ok = true
				} else {
					erro(ctx, "%v %v", ts(a), res, callstack{num:10})
				}
			}
		case *rule:
			// Ensure matched == true AND there is no unconsumed remainder
			if matched, res, rem, _ := match(ctx, a.target, fullvalue); matched && rem == nil {
				var a = &rule{a.target, a.arged, a.program}
				if 0 < do(ctx, matched_rule{a, res}).(int) {
					ok = true
				} else {
					erro(ctx, "%v %v", ts(a), res, callstack{num:10})
				}
			}
		default:
			erro(ctx, "%v", ts(a), callstack{num:10})
		}
	}
	return
}

type hit_segs struct{ *valcache ; s [][]Symbol }
type fullvalue struct{}
type fullctx struct{ Context ; any }
func (p *fullctx) do(ctx Context, op any) any {
	switch op.(type) {
	case fullvalue: return p.any
	}
	return p.Context.do(ctx, op)
}

// toks handles pure string splitting. We intercept the string here BEFORE it
// hits tokenizeSegments, ensuring "**.c" natively compiles as "**/*.c".
func toks(ctx Context, c *valcache, segs ...string) hit_segs {
	var norm []string
	for _, seg := range segs {
		// Option 1: Normalize intra-string path closures
		if strings.Contains(seg, "**.") {
			norm = append(norm, strings.ReplaceAll(seg, "**.", "**/*."))
		} else {
			norm = append(norm, seg)
		}
	}

	// CRITICAL FIX: tokenizeSegments was upgraded to return [][]Symbol natively.
	// We no longer need to loop and intern manually here!
	return hit_segs{c, tokenizeSegments(norm)}
}

// tokc delegates to toks, so it automatically inherits the normalization
// applied above when prefix is evaluated.
func tokc(ctx Context, c *valcache, comp *compound) hit_segs {
	var prefix string
	for _, e := range comp.elems {
		if _, isClosure := unbox(e).(*closure); isClosure {
			var ss [][]Symbol
			if prefix != "" {
				ss = append(ss, toks(ctx, c, prefix).s...)
			}
			ss = append(ss, []Symbol{symAmpersand}) // Integer injection!
			return hit_segs{c, ss}
		}
		prefix += __string(ctx, e)
	}
	if prefix != "" { return toks(ctx, c, prefix) }
	return hit_segs{c, nil}
}

func tokg(ctx Context, c *valcache, g *globpat) hit_segs {
	var s []Symbol // Now an array of Symbols!

	for i, e := range g.elems {
		if _, isClosure := unbox(e).(*closure); isClosure {
			var ss [][]Symbol
			if len(s) > 0 { ss = append(ss, s) }
			ss = append(ss, []Symbol{symAmpersand})
			return hit_segs{c, ss}
		}

		str := __string(ctx, e)

		if strings.Contains(str, "**.") {
			str = strings.ReplaceAll(str, "**.", "**/*.")
		}

		// Intern the single glob element (e.g. "**", "foo", "bar")
		s = append(s, intern(str))

		if str == "**" && i+1 < len(g.elems) {
			nextStr := __string(ctx, g.elems[i+1])
			if nextStr == "." || strings.HasPrefix(nextStr, ".") {
				// Inject the pre-interned wildcard directly!
				s = append(s, symWildcardOne)
			}
		}
	}
	return hit_segs{c, [][]Symbol{s}}
}

// tokp natively delegates to tokc/tokg/toks, so it requires no normalization logic of its own.
func tokp(ctx Context, c *valcache, p *path) hit_segs {
	var ss [][]Symbol // CRITICAL FIX: Changed from [][]string

	if false && checkpoints { defer func() {
		if t := tokenizePath(__string(ctx, p)); sf("%s",ss) != sf("%s",t) {
			erro(ctx, "%v: %v != %v", p, ss, t)
		}
	}()}

	for _, e := range p.elems {
		switch t := unbox(e).(type) {
		case *closure:
			ss = append(ss, []Symbol{symAmpersand}) // CRITICAL FIX: Changed from "&"
			return hit_segs{c, ss}

		case *compound:
			res := tokc(ctx, c, t)
			ss = append(ss, res.s...)

			// CRITICAL FIX: Compare against SymClosure, not "&"
			if len(res.s) > 0 && len(res.s[len(res.s)-1]) == 1 && res.s[len(res.s)-1][0] == symAmpersand {
				return hit_segs{c, ss}
			}

		case *globpat:
			res := tokg(ctx, c, t)
			ss = append(ss, res.s...)

			// CRITICAL FIX: Compare against SymClosure, not "&"
			if len(res.s) > 0 && len(res.s[len(res.s)-1]) == 1 && res.s[len(res.s)-1][0] == symAmpersand {
				return hit_segs{c, ss}
			}

		default:
			ss = append(ss, toks(ctx, c, __string(ctx, t)).s...)
		}
	}
	return hit_segs{c, ss}
}

func _tokq(ctx Context, c *valcache, t *qualword) hit_segs {
	// 1. unpack(t) correctly interleaves the dots: [word(foo), punct(.), word(c)]
	// 2. Wrap it in a temporary compound so tokc can natively extract the hit_segs!
	return tokc(ctx, c, &compound{elements{unpack(t)}})	// Let the universal flattener safely interleave the structural dots!
}
func tokq(ctx Context, c *valcache, t *qualword) hit_segs {
	// Interleave punct(.) on the fly so it hits the cache edges correctly
	res := make([]Value, 0, len(t.elems)*2-1)
	for i, e := range t.elems { if i > 0 { res = append(res, implicitDot) }
		res = append(res, e)
	}
	return tokc(ctx, c, &compound{elements{res}})
}

func _hit(ctx Context, c *valcache, k Value) (r []*valcache) {
	if checkpoints { defer func(s string) {
		if  truly(ctx, propCache)   { check_cache(ctx, k, s, c, r) }
		if  truly(ctx, propUncache) { check_uncache(ctx, k, s, c, r) }
		if !truly(ctx, propCache|propUncache) { erro(ctx, "%v %v", k, c, callstack{num:10}) }
	}(c.String())}

	switch t := k.(type) {
	case *argumented: return _hit(ctx, c, t.Value)
	case *loc       : return _hit(ctx, c, t.Value)
	case *rule      : return _hit(ctx, c, t.target)
	case *closure   : return do_hit(&fullctx{ctx,t}, toks(ctx, c, "&")) // Return directly to avoid duplication
	case *compound  : r = do_hit(&fullctx{ctx,t}, tokc(ctx, c, t))
	case *globpat   : r = do_hit(&fullctx{ctx,t}, tokg(ctx, c, t))
	case *path      : r = do_hit(&fullctx{ctx,t}, tokp(ctx, c, t))
	case *qualword  : r = do_hit(&fullctx{ctx,t}, tokq(ctx, c, t)) // CRITICAL FIX: Route qualword to tokq
	case *percpat   : r = do_hit(&fullctx{ctx,t}, toks(ctx, c))
	case *regexpat  : r = do_hit(&fullctx{ctx,t}, toks(ctx, c))
	case *strval    : r = do_hit(&fullctx{ctx,t}, toks(ctx, c, `{`+__string(ctx,t)+`}`))
	case *strlit    : r = do_hit(&fullctx{ctx,t}, toks(ctx, c, `'`+__string(ctx,t.s)+`'`))
	case *strcomp   : r = do_hit(&fullctx{ctx,t}, toks(ctx, c, `"`+__string(ctx,t)+`"`))
	default:
		segs := strings.Split(__string(ctx, k), pathSep)
		r = do_hit(&fullctx{ctx,t}, toks(ctx, c, segs...))
	}

	// Universal Closure Inclusion: For any non-recursive query, unconditionally
	// fetch the dynamic closures branch ("&") and append it to the results.
	if false { r = append(r, do_hit(&fullctx{ctx, k}, toks(ctx, c, "&"))...) }
	return r
}
func do_hit(c Context, a any) (r []*valcache) { r, _ = do(c,a).([]*valcache); return }
func hit(ctx Context, c *valcache, k Value) []*valcache { return _hit(ctx, c, k) }

type   cache_t struct{ Context }
type uncache_t struct{ Context ; a []any }

func (c cache_t) do(ctx Context, op any) any {
	switch t := op.(type) {
	case property: if t&propCache != 0 { return true }
	case hit_segs: return []*valcache{cache(ctx, t.valcache, t.s)}
	}
	return c.Context.do(ctx, op)
}

func (u *uncache_t) do(ctx Context, op any) (res any) {
	switch t := op.(type) {
	case property: if t&propUncache != 0 { return true }
	case hit_segs: return uncache(ctx, t.valcache, t.s)
	case matched_filemap, matched_rule:
		u.a = append(u.a, t); return len(u.a)
	}
	return u.Context.do(ctx, op)
}

func map_files(ctx Context, p *project, patts, paths []Value) (res []filemap) {
	var base = &_filemap{p, patts, paths}
	for _, patt := range patts {
		switch patt.(type) {
		case *valbase, *null, *none:
			continue
		}
		if c := hit(cache_t{ctx}, &p.filemap, patt); c != nil {
			switch len(c) {
			case 1:
				c0, f := c[0], filemap{base, patt}

				// CRITICAL FIX: Deduplicate! Check if the pattern is already in this slot.
				var exists bool
				for _, a := range c0.a {
					if f0, ok := a.(filemap); ok {
						if cmp(ctx, f0.pattern, patt) == cmpEqual {
							exists = true
							break
						}
					}
				}

				// Only append if it's genuinely new to this slot
				if !exists { c0.a = append(c0.a, f) }
				res = append(res, f)

			default:
				erro(pc(ctx,patt), "too many cached: %v %v", ts(patt,ctx), c)
			}
		} else {
			erro(pc(ctx,patt), "cache failed: %v", ts(patt,ctx))
		}
	}
	return
}

func map_entry(ctx Context, p *project, target Value, prog *program) (entry entry) {
	var patterned = patterned(ctx, target)
	if !patterned {
		switch target.(type) {
		case *barefile, *file, *path, *percpat, *globpat, *regexpat:
			goto skip_file
		}
		if t := p.file(ctx, target); t != nil { target, t.pos = t, target.Pos() }
	skip_file:
	}

	var args []Value // e.g. for pattern filtering
	switch t := target.(type) {
	case *argumented: target, args = t.Value, merge(t.args...)
	case *group: erro(ctx, "not supported target: %v", t)
	}

	if c := hit(cache_t{ctx}, &p.entries, target); c == nil {
		erro(ctx, "uncachable for: %v | %s", target, ts(target,ctx))
	} else {
		switch len(c) {
		case 1:
			// CRITICAL FIX: Check if a rule for this target already exists in the slot!
			var found *rule
			for _, a := range c[0].a {
				if r, ok := a.(*rule); ok && cmp(ctx, r.target, target) == cmpEqual {
					found = r
					break
				}
			}

			if found != nil {
				// Target exists! Just merge the new program into the existing rule's slice.
				if prog != nil { found.program = append(found.program, prog) }
				entry = found
			} else {
				// Target is new! Create and append a fresh rule.
				r := &rule{target:target, arged:args, program:[]*program{prog}}
				if patterned { p.patterns = append(p.patterns, r) }
				if p.main == nil { p.main = r }
				c[0].a = append(c[0].a, r)
				entry = r
			}

		default:
			erro(pc(ctx,target), "too many cached: %v %v", ts(target,ctx), c)
		}
	}
	return
}

func unmap[T any](ctx Context, c *valcache, key any) (res []T) {
	var u = &uncache_t{ctx, nil}
	var k Value

	if v, ok := key.(Value); ok {
		k = v
	} else {
		str := __string(ctx, key)
		// Emulate the parser: if the raw string is a path, pack it into a *path AST node.
		if strings.Contains(str, pathSep) {
			if segs := splitPathStr(ctx, str); len(segs) > 1 {
				k = packPath(segs)
			} else {
				k = _rw(_pos(ctx), str)
			}
		} else {
			k = _rw(_pos(ctx), str)
		}
	}

	var x = hit(u, c, k)
	for _, a := range u.a {
		switch t := a.(type) {
		case T:
			res = append(res, t)
		default:
			erro(ctx, "%v %v", ts(key), ts(a))
		}
	}
	if checkpoints { check_unmap(u, key, c, x) }
	return
}

func unmap_entries(ctx Context, p *project, key any, m map[*project]struct{}) (res []entry) {
	if m == nil { m = map[*project]struct{}{} } else if _, y := m[p]; y { return }
	if m != nil { m[p] = struct{}{} }
	if res = unmap[entry](ctx, &p.entries, key); res != nil { return }
	for _, b := range p.bases {
		if res = unmap_entries(ctx, b, key, m); res != nil { return }
	}
	if c := p.configure; false && c != nil {
		if res = unmap_entries(ctx, c, key, m); res != nil { return }
	}
	return
}

func unmap_files(ctx Context, p *project, key any, m map[*project]struct{}) (res []matched_filemap) {
	if m == nil { m = map[*project]struct{}{} } else if _, y := m[p]; y { return }
	if m != nil { m[p] = struct{}{} }
	if res = unmap[matched_filemap](ctx, &p.filemap, key); res != nil { return }
	for _, b := range p.bases {
		if res = unmap_files(ctx, b, key, m); res != nil { return }
	}
	if c := p.configure; c != nil {
		if res = unmap_files(ctx, c, key, m); res != nil { return }
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

	configure     *project // .configure project
	configuration *file    // configuration.sm if saved or loaded

	// --- OS I/O Domain (Strictly Strings) ---
	// DO NOT intern these! They are machine-specific and break portability/caching.
	absPath string
	tmpPath string

	// --- Logical Routing Domain (Portable Symbols) ---
	// These are identical across all developer machines and safe to intern/hash.
	name Symbol // e.g., intern("core")
	rel  Symbol // path segment relative to the baseWorkDir
	spec Symbol // relative to search-paths as a specification

	// Explicitly tracks flags defined via use.XXX (e.g. intern("-l"))
    exports []Symbol

	filemap valcache
	entries valcache

	shadowsMu sync.RWMutex
	shadows   map[Value]*shadowCache

	patterns []*rule // order is important
	configs  []*def  // configure entries
	main     entry

	ext project_ext
	opt project_opts
}
func (_ *project) kind() Kind { return KindObject|KindKnownObject|KindProject }
func (p *project) Pos() Pos { return p.pos }
func (p *project) owner() *project { return p.scope.project }
func (p *project) String() string { return "{=project "+p.name.String()+"}" }
func (p *project) stencil(_ Context, stems []string) (Value, []string) { return p, stems }

// Helper to deduplicate exports during parsing
func (p *project) addExport(sym Symbol) {
    for _, e := range p.exports {
        if e == sym { return }
    }
    p.exports = append(p.exports, sym)
}

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
func (p self) String() string { return "{=self "+p.name.String()+"}" }

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
                    erro(ctx, "%s ⇒ %v → %v → ''", m.value, v, t)
                }
            } else if false {
                erro(ctx, "%s ⇒ %v → %v → ''", m.value, v, t)
            }
        } else {
            erro(ctx, "%s ⇒ %v", m.value, v)
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
		if d = p.resolveDef(ctx, intern(t)); d != nil { break }
	}

	if d == nil {
		erro(ctx, "%v: tmp is not defined", p)
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
			_f("%v", p.resolveDef(ctx, intern("outtmp"))),
			_f("%v", p.resolveDef(ctx, intern("target.tmp"))),
			_f("%v", p.resolveDef(ctx, intern("target.out"))),
			_f("%v", p.resolveDef(ctx, intern("target.triple"))),
			_f("%v", p.resolveDef(ctx, intern("rel.remnant"))),
			_f("%v", p.resolveDef(ctx, intern("rel.chop"))),
			_f("%v", p.resolveDef(ctx, intern("variant.tag"))),
			trace{})
    }

    if f = _stat(ctx, name, stat_dir{d}, stat_nonexist{true}); f == nil {
        erro(ctx, "%v: not a file: %v : %v", p, name, d)
    }

    if false && checkpoints { tempfile_check(ctx, p, name, d, f) }
    return
}

func (p *project) configuration_sm(ctx Context) (f *file) {
    if f = p.tempfile(ctx, configuration_sm); f == nil {
        erro(ctx, "%v: no file %s", p, configuration_sm)
    }
    return
}

func project_entry(c Context, s any, a ...bool) entry { return _project(c).entry(c, s, a...) }
func project_resolve(c Context, s Symbol) object { return _project(c).resolve(c, s) }

func (p *project) resolveDef(ctx Context, name Symbol) (res *def) {
    if o := p.resolve(ctx, name); o != nil { res, _ = o.(*def) }
    return
}

func (p *project) resolve(ctx Context, name Symbol) (obj object) {
    if _, obj = p.find(name); obj != nil { return }

    if p.ext.Plugin != nil {
        if sym, e := p.ext.Lookup(name.String()); e == nil && sym != nil {
            erro(ctx, "TODO: convert ext symbol: %v: %s", name, typeof(sym))
        }
    }

    for _, base := range p.bases {
        if base.has_base(p) {
            erro(ctx, "recursive derivation: %v ⇔ %v", ts(p,ctx), ts(base,ctx))
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
        if 1 < n { erro(c, "%v : %d entries", name, n) }
        return entries[0]
    }
    return
}

func (p *project) resolvePatterns(ctx Context, v Value, s Symbol) (res []*stemmed_rule) { // FIX: s is now Symbol
	var t1, t2 time.Time

	defer func(t0 time.Time) {
		var t = time.Now()
		if d := t.Sub(t0); d > 1*time.Second {
			var ( d1 = t1.Sub(t0) ; d2 = t2.Sub(t1) ; d3 = t.Sub(t2) ; n int )
			var a = auto_get(ctx, symAt)
			for sc := _stemmed(ctx); sc != nil; n += 1 {
				if c := inner(sc); c != nil { sc = _stemmed(c) } else { break }
			}

			var dps []*diag_point
			var pos = _position(ctx)
			// s natively prints via fmt Stringer interface
			dps = append(dps, _f("%v: slow: %v: %v, %v %v %v", pos, p, d, d1, d2, d3))
			dps = append(dps, _f("%v: slow: %v: %v: %v %v, %d nests", pos, p, a, v, p.patterns, n))

			sStr := s.String() // Extract string once for the loop
			for _, pat := range p.patterns {
				var pt = pat.target
				var pa = pat.arged

				// raw struct likely expects a string, so we pass sStr
				var full, r, _, stems = match(ctx, pt, &raw{valbase{pt.Pos()}, sStr})
				var m = joinp(ctx, r)
				dps = append(dps, _f("%v: slow: %v%v: %v: %v %v %v, %v ; %v", pos, pt, pa, s, full, r, stems, m))
			}
			debug(ctx, dps, diagtext{})
		}
	} (time.Now())

	// Assuming resolvePatterns123 hasn't been upgraded to Symbol yet.
	// If it has, just pass 's'. If not, pass 's.String()'.
	if res, t1, t2 = p.resolvePatterns123(ctx, v, s.String()); false && len(res) > 0 {
		for _, t := range res {
			if f, _ := to_file(t.target); f != nil {
				f.pos = t.Pos()
			} else if f = p.file(ctx, s.String()); f != nil { // p.file expects a string for VFS lookup
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

func (p *project) _isa(s Symbol) (_ bool) {
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
        erro(ctx, "%v: %v", p, err)
    } else if depth > 128 {
        err = fmt.Errorf("exceeds maximum base depth (%d) (start=%v, target=%v)", depth, top, proj)
        debug(ctx, "%v: %v", p, err)
        debug(ctx, "start: %v", top)
        erro(ctx, "target: %v", proj)
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
            erro(ctx, "%v: %v", p, err)
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
    if s == "" { s = p.name.String() }
    for _, base := range p.bases {
        if t := s + " → " + base.name.String(); base == _p {
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

type execution_lang struct{}
type missing_file struct{ file string }

func _execution(c Context) *execution { return cast[*execution](c) }

type interpret struct{ name Symbol ; args []Value }
type execution struct{
    automatic
    sync.Mutex
    sync.WaitGroup

    by dirtyOpts

    proj *project
    prog *program
    recipes []Value
    language Symbol

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
            auto_set(ctx, defVoid, symBar, _list(p.ordered...)) // |
        } else {
            p.targets = append(p.targets, target)
            auto_set(ctx, defVoid, symCaret, _list(p.targets...)) // ^
            auto_set(ctx, defVoid, symLangle, p.targets[0]) // <
            auto_set(ctx, defVoid, symRangle, p.targets[len(p.targets)-1]) // >
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
    } else if targets := merge(auto_get(p, symAt)); len(targets) == 0 {
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

func (p *execution) interp(ctx Context, name Symbol, args []Value) (res bool) {
    if len(p.interpreted) == 0 && 0 < len(p.recipes) && name == symConfigure {
        if x, y := dialects[symEval]; y && x != nil {
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
        if x, y := target.(*file); y && strings.HasSuffix(x.name.String(), ".log") {
            defer func() {
                var cc = pc(pc(ctx,auto_get(ctx,symRangle)),x.fullname())
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
        if d, prev := auto_set(ctx, defVoid, symDash, res); d == nil {
            _, ent, _ := entryIndicator(ctx, _entry(ctx))
            prompt(ctx, "%v: %s\n", ent, intername(i))
            erro(ctx, "set buffer value failed: %v → %v", prev, res)
        }
    }

    p.interpreted = append(p.interpreted, i)

    if _, _, e := p.updateRecipesHash(ctx, target); e != nil {
        _, ent, _ := entryIndicator(ctx, _entry(ctx))
        prompt(ctx, "%v: %s\n", ent, intername(i))
        erro(ctx, "update recipes hash failed: %v", e)
    }
    return
}

func isDirty(ctx Context, target Value, a ...Value) (dirty bool) {
    var opts, y = do(ctx, propDirtyOpts).(*dirtyOpts)
    if !y {
        erro(ctx, "nil dirtyopts : %v", ts(ctx))
        return
    }
    if len(updatedDeps(ctx, target)) > 0 { return true }
    if v := auto_get(ctx, intern("^")); v != nil { a = append(a, v) }
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
        erro(ctx, "recipes changed: %v", e)
        return
    } else if y {
        outdated, reason = true, "recipes changed"
    } else if !opts.checksum {
        // does nothing
    } else {
        erro(ctx, "FIXME: check prerequisites against the saved checksums")
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

// Upgraded return signature: prereqFinal is now a Symbol
func probPrereqValue(ctx Context, projects []*project, val Value) (prereqValue, prereqPattern Value, prereqFinal Symbol, prereqFile *file) {
	var mapPrereqFile = func(name Value) {
		var ms = unmap_files(ctx, _project(ctx), name, nil)
		if ms != nil {
			defer func() { if prereqFile == nil {
				var dps []*diag_point
				for _, m := range ms { dps = append(dps, _f("%v, skipped %v", name, m)) }
				dps = append(dps, _f("skipped %d, projects %v", len(ms), projects))
				debug(ctx, dps)
			}}()
		}

		if prereqFile = select_file(ctx, ms); prereqFile != nil {
			// Assuming file.name is still a string; if you update the file struct
			// to use Symbol later, you can remove the intern() call here.
			prereqValue, prereqFinal = prereqFile, prereqFile.name
		} else if prereqValue != nil {
			if f, y := to_file(prereqValue); y {
				prereqFile, prereqFinal = f, f.name
			} else {
				prereqFinal = intern(__string(ctx, prereqValue))
				if _, y := prereqValue.(*path); y {
					// Fallback to string for OS-level stat calls
					if f := _stat(ctx, prereqFinal.String()); f != nil { prereqFile, prereqValue = f, f }
				}
			}
		}
	}

	prereqValue = val
	if _, y := prereqValue.(object); y { return }

	if !patterned(ctx, prereqValue) {
		switch prereqValue.(type) {
		case flag, *strlit, *strcomp: // skip checking files for performance
		default: mapPrereqFile(prereqValue)
		}
		return
	}

	var stems = _stems(ctx)
	if len(stems) == 0 {
		if false { erro(ctx, "%v: no stems, %v", prereqValue, ctx) }
		return
	}

	var stemVals []Value
	for _, s := range stems { stemVals = append(stemVals, s) }

	var rest []Value
	prereqPattern = prereqValue
	prereqValue, rest = stencil(ctx, prereqPattern, stemVals)
	if isTrivial(prereqValue) {
		erro(ctx, "%v: empty stencil with %v", prereqPattern, stems)
	} else if len(rest) > 0 {
		erro(ctx, "%v: partial stencil with %v, rest=%v", prereqPattern, stems, rest)
	}

	mapPrereqFile(prereqValue)
	return
}

func (p *execution) traverse(ctx Context, prereqValue Value) {
	var (
		targetValue   Value
		prereqPattern Value
		prereqFinal   Symbol // CRITICAL FIX: Upgraded to Symbol
		prereqFile    *file

		concreteList []entry
		stemmedList  []*stemmed_rule
	)

	if targetValue = auto_target_value(ctx); targetValue == nil {
		erro(ctx, "%v: target is nil\n", prereqFinal) // %v will call prereqFinal.String()
	} else if isTrivial(targetValue) {
		erro(ctx, "%v: target is trivial (%T)\n", prereqFinal, targetValue)
	}

	var projs = []*project{ p.proj }

	if len(projs) == 0 {
		erro(ctx,
			_f("%v", closure_projects(ctx)),
			_f("%v: %v → %v: no projects", p.proj, targetValue, prereqValue))
	}

	prereqValue, prereqPattern, prereqFinal, prereqFile = probPrereqValue(ctx, projs, prereqValue)

	if f := prereqFile; f != nil {
		if f._travin += 1; f._travin > 1 { return }
	}

	// Recursion detection -- simply return to break it if looped.
	if detectTraverseLoops {
		if eq(ctx, targetValue, prereqValue) {
			erro(ctx,
				_f("%v: %v: self dependency, consider using [(once)] to avoid\n", targetValue, prereqValue),
				_f("recursion: %T %v", prereqValue, prereqValue),
				_f("recursion: %T %v", targetValue, targetValue),
				_f("recursion: %v : %v ; in %v", targetValue, prereqFile, projs))
		}
		for c := p; c != nil; c = c.caller() {
			if val := auto_get(c, symAt); val != nil && eq(c, val, prereqValue) {
				var f = as_file(ctx, targetValue, projs...)
				if true && f == nil {
					debug(ctx,
						_f("%v: %v: recursion detected, consider using [(once)] to avoid\n", targetValue, prereqValue),
						_f("recursion: %T %v", prereqValue, prereqValue),
						_f("recursion: %T %v", targetValue, targetValue),
						_f("recursion: %v : %v ; in %v", targetValue, prereqFile, projs))
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

			// CRITICAL FIX: The ultimate payoff of Symbol Interning!
			// Assuming your `word` struct was upgraded to use `sym Symbol` instead of `s string`.
			// This check is now an O(1) integer comparison!
			if w, k := targetValue.(*word); k && w.s == prereqFinal {
				continue // target resolve to itself, does nothing
			}

			traverse(ctx, entry)
		}
	}

	if d := time.Now().Sub(t1); 60*time.Second < d {
		var dps []*diag_point
		for _, concrete := range concreteList {
			pos := do(ctx, get_fatpos{concrete.Pos()})
			dps = append(dps, _f("%v: slow: %v %v\n", pos, concrete, targetValue))
		}
		dps = append(dps, _f("%v: slow: %v: %v %v (%d concretes)\n", _position(ctx), targetValue, prereqValue, d, len(concreteList)))
		debug(ctx, dps, diagtext{})
	}

	if prereqFile != nil && prereqFile.exists() {
		p.traved(ctx, targetValue, prereqValue, prereqPattern, prereqFile)
		return
	}

	t2 := time.Now()

	for _, proj := range projs {
		for _, p := range proj.patterns { assert(patterned(ctx, p.target), "not pattern") }

		// NOTE: Be sure to update `resolvePatterns` signature to accept a Symbol
		// for the third argument instead of a string!
		patterns := proj.resolvePatterns(ctx, prereqValue, prereqFinal)
		if len(patterns) == 0 { continue }

		stemmedList = append(stemmedList, patterns...)

		for _, entry := range patterns { traverse(ctx, entry) }
	}

	if d := time.Now().Sub(t2); 60*time.Second < d {
		var dps []*diag_point
		for _, stemmed  := range stemmedList {
			pos := do(ctx, get_fatpos{stemmed.Pos()})
			dps = append(dps, _f("%v: slow: %v\n", pos, stemmed))
		}
		dps = append(dps, _f("%v: slow: %v: %v %v (%d stemmed)\n", _position(ctx), targetValue, prereqValue, d, len(stemmedList)))
		debug(ctx, dps, diagtext{})
	}

	p.traved(ctx, targetValue, prereqValue, prereqPattern, prereqFile)
	return // no operation
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
    language  Symbol
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
        if o = proj.resolve(ctx, symCWD); isTrivial(o) {
            if o = proj.resolve(ctx, symSlash); isTrivial(o) {
                erro(ctx, "both $(CWD) and $/ are trivial")
            }
        }
        if v := expand(_final(ctx),o); v == nil {
            erro(ctx, "trivial %v", ts(o,ctx))
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
        erro(ctx, "%v: nil entry target", target)
    }

    switch a := target.(type) {
    case *strlit, *strcomp: // NOTE: skip strings to optimize speed from searching
    case *file: if a._traved > 1 { return } // alreadyUpdated = a.info != nil && a.updated
    case fullfile: if a._traved > 1 { return } // alreadyUpdated = a.info != nil && a.updated
    default:
        if _, y := a.(flag); y {
            if s := prog.project.name.String(); s == "configure" || s == "configure.base" {
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
    var a = []Value{ auto_get(cc, symAt) }
    var depth, loop int = 0, -1

callerloop:
    for c := ctx.caller(); c != nil; c = c.caller() {
        if _program(c) == prog {
            if depth += 1; depth == maxCallRecursion { break callerloop }
            var t = auto_get(c, symAt)
            for i, v := range a {
                if eq(ctx, t, v) { loop = i; break callerloop }
            }
            if loop < 0 { a = append(a, t) }
        }
    }

    if 0 <= loop {
        var o = cast[*term](cc)
        var v = auto_get(o, symAt)
        var t = auto_get(cc, symAt)
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
        erro(ctx, "loop, (depth=%d, %v, %v)\n", depth, a[loop], a)
    }

    if depth < maxCallRecursion {
        // continues
    } else if c := ctx.caller(); c != nil {
        ctx.traceLevel = c.traceLevel

        var tt = auto_get(c, symAt)
        var s, _ = as_fullname_string(ctx, tt)
        prompt(ctx, "%v: max recursion call (%d)\n", s, depth)
        debug(ctx, "max recursion call (%d)\n", depth)

        const collapse = false

        for ; c != nil; c = c.caller() {
            var n int

            if collapse {
                for next := c.caller(); next != nil; next = next.caller() {
                    if d := auto_get(next, symAt); d == nil { continue } else
                    if t := d; t != nil && eq(ctx, t, tt) { n += 1;  continue }
                    if _program(next) == _program(c) { n += 1; c = next } else { break }
                }
            }

            if prog, t := _program(c), auto_get(c, symAt); prog == nil {
                erro(ctx, "%v (@=%v)", _entry(ctx), tt)
                break
            } else if n > 0 {
                erro(ctx, "%v (repeated %d times)", t, n)
            } else if !collapse {
                erro(ctx, "%v : %v", t, auto_get(c, intern(">")))
            } else if depth -= 1; maxCallRecursion - depth > 5 {
                erro(ctx, "%v ... (%d)", t, maxCallRecursion - depth)
                break
            } else {
                erro(ctx, "%v : %v", t, auto_get(c, intern(">")))
            }

            flush(ctx) // dump immediately
        }

        erro(ctx, depth, "#>", _entry(ctx))
    }
}

func (prog *program) result_or_default_interpret(ctx *execution) (res Value) {
    if res = auto_get(ctx, symDash); res != nil {
        return
    }
    if len(ctx.interpreted) == 0 {
        if x, y := dialects[symEmpty]; y && x != nil {
            return ctx.interpret(ctx, x, nil)
        }
        erro(ctx, "no default dialect")
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
                erro(exe, "defer: not a modifier: %s", ts(a))
            }
        }

        if res == nil { res = auto_get(exe, symDash) }
        if res == nil { res = exe.defval }
        if res != nil { do(exe.Context, default_value{res}) }

        exe.defval = nil
    } ()

    for _, param := range prog.params {
        if  exe.params == nil  {
            exe.params = make(map[Symbol]*auto, len(prog.params))
        }
        exe.params[param.name] = param
    }

    if checkpoints { prog.execute_check_0(exe) }

    // NOTE: set "@" before setting auto args
    // Select the right target value before setting parameters,
    // because the target could be overrided by parameters.
    exe.set(exe, defVoid, symAt, prog.target(exe)) // @

    if t := _stems(exe); t != nil {
        exe.set(exe, defVoid, symAsterisk, ease(exe, t)) // "*"
    }

    exe.do(exe, init_args{&exe.automatic})
    exe.prerequisites(prog.depends, false)
    exe.prerequisites(prog.ordered, true)

    if checkpoints { prog.execute_check_1(exe) }

    if len(prog.recipes) == 0  { return }
    return prog.result_or_default_interpret(exe)
}

func _configurecontext(c Context) *configurecontext { return cast[*configurecontext](c) }
func is_configurecontext(ctx Context) bool { return _configurecontext(ctx) != nil }

type silent_configure struct {}

type configurecontext struct {
    Context
    current *project
    silent bool
    file *os.File
    writer *bufio.Writer
    defs map[Symbol]struct{}
    done map[*def]struct{}
}
func (cc *configurecontext) inner() Context { return cc.Context }
func (cc *configurecontext) cast(t reflect.Type) Context { return icast(cc, t) }
func (cc *configurecontext) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case silent_configure: if cc.silent { return true }
    }
    return cc.Context.do(ctx, op)
}
func (cc *configurecontext) open_configuration_sm(ctx Context, p *project) (res *os.File) {
    if f := p.configuration_sm(ctx); f == nil {
        erro(ctx, "%p: nil configuration file", p)
    } else if s := f.fullname(); s == "" {
        erro(ctx, "empty configuration file name: %v", f)
    } else if e := os.MkdirAll(filepath.Dir(s), os.FileMode(0755)); e != nil {
        erro(ctx, "make path %s failed: %v", filepath.Dir(s), e)
    } else {
        res, e = os.OpenFile(s, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0600))
        if e != nil { erro(ctx, "%v", e) }
    }
    return
}
func (cc *configurecontext) execute(ctx *execution, e entry) {
	var d *def
	var sym Symbol // CRITICAL FIX: Upgraded from string to Symbol
	var p = e.owner()

	if checkpoints {
		// defer cc.execute_check(ctx, e, p, &sym, &d) // ensure execute_check signature accepts *Symbol
	}

	if p != cc.current && p != nil {
		cc.defs = make(map[Symbol]struct{}) // CRITICAL FIX: Upgrade map to use Symbol!

		// NOTE: configuration.sm is created for every project
		var f = cc.open_configuration_sm(ctx, p)
		if f != nil {
			if cc.writer != nil {
				if e := cc.writer.Flush(); e != nil {
					erro(ctx, "%v", e)
				}
			}
			if cc.file != nil {
				if e := cc.file.Close(); e != nil {
					erro(ctx, "%v", e)
				}
			}
		}

		cc.file, cc.writer = f, bufio.NewWriter(f)
		fmt.Fprintf(cc.writer, "# %s (%s) configuration\n", p.spec, p.name)

		if !cc.silent {
			s := _universe(ctx).trimSpecPath(ctx, p.spec.String())
			prompt(ctx, "configure %s …… (%s)\n", p.name, s)
			if true { flush(ctx) }
		}

		cc.current = p
	}

	e.execute(ctx)

	// 1. Evaluate the AST node to its final string.
	// 2. Intern it immediately into the fast integer domain!
	sym = intern(__string(ctx, e.destiny()))

	if _, y := cc.defs[sym]; y { return } // O(1) integer hash check

	if d = cc.current.finddef(sym); d == nil {
		// %s works perfectly here because Symbol satisfies the Stringer interface!
		erro(ctx, "%v: `%s` not configured", cc.current, sym)
		return
	}

	cc.defs[sym] = struct{}{}

	// d.name is already a Symbol, so we can use it natively
	if nameSym := d.name; d.value == nil {
		// Set <nil> value with exec-assign ('!=') to a None value.
		fmt.Fprintf(cc.writer, "%s !=\n", nameSym)
	} else {
		// Because d.value is an AST Value, d.value.String() is correct here.
		fmt.Fprintf(cc.writer, "%s = %v\n", nameSym, d.value.String())
	}
	return
}
func (cc *configurecontext) close() {
    if cc.writer != nil { if e := cc.writer.Flush(); e != nil {} }
    if cc.file != nil   { if e := cc.file.Close();   e != nil {} }
}

func scanExitStatts(err error) (n, status int) {
    switch e := err.(type) {
    case *exitstatus: n, status = 1, e.int
    default: n, _ = fmt.Sscanf(err.Error(), fmtExitStatus, &status)
    }
    return
}

type filewalkFunc func(file *file, err error) error

func walkFileInfos(ctx Context, root string, pats []Value, fn filepath.WalkFunc) (err error) {
    return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if err != nil { return err }
    ForPats:
        for _, p := range pats {
            var matched bool
            if matched, _, _, _ = match(ctx, p, _pathStr(ctx, path)); !matched {
                matched, _, _, _ = match(ctx, p, _pathStr(ctx, filepath.Base(path)))
            }
            if matched {
                if err = fn(path, info, err); err != nil {
                    break ForPats
                }
            }
        }
        return err
    })
}

func walkFiles(ctx Context, root string, pats []Value, fn filewalkFunc) error {
    return walkFileInfos(ctx, root, pats, func(path string, info os.FileInfo, err error) error {
        if err != nil { return err }
        var rel string
        if rel, err = filepath.Rel(root, path); err != nil {
            return err
        }
        return fn(_stat(ctx, rel, stat_dir{root}, stat_fileinfo{info}), err)
    })
}

var configuredFiles = make(map[string]*scope, 8)

type (
    configureconvertArgs func([]Value, *bytes.Buffer) []Value
    configureconvertFunc func(string, *bytes.Buffer)
    configureconvertOpts struct {
        general_opts
        mode     os.FileMode `mode`
        makePath bool `path`
        mustConf bool `mustconfig,mustconf,must-conf,must-config,nc,needsconfig,needs-config`
        reconfig bool `reconfig`
        update   bool `update`
    }
)
func configureconvert(ctx *execution, dealArgs configureconvertArgs, dealData configureconvertFunc, opts *configureconvertOpts, args ...Value) (_ Value) {
	var (
		closured = closure_projects(ctx)
		filename string
		f *file
		target Value
	)

	args = parseOpts(ctx, opts, args...)

	if target = auto_get(ctx, symAt); isTrivial(target) {
		erro(ctx, "'@' is not defined")
	} else if f, filename, _ = as_fullname_file(ctx, target, closured...); f == nil {
		if depend := auto_get(ctx,symRangle); !isTrivial(depend) {
			panic(traveTargetNotDefinedFile)
		} else {
			erro(ctx,
				_f("%v: not defined as file", __string(ctx, target)),
				_f("%v", ts(target,ctx)))
		}
		return
	} else if filename == "" {
		erro(ctx, "%v: empty fullname", target)
	}

	if _, prev := auto_set(ctx, defVoid, symAt, f); opts.debug>0 {
		debug(ctx, "configure-file: %s->%s (%v -> %v)", f.name.String(), filename, ts(prev,ctx), ts(f,ctx))
	}

	// 1. O(1) Cache Sync
	if !f.exists() { f.stat(ctx) }
	
	if f != nil && 0 < opts.debug {
		debug(ctx, "configure-file: %v: %v (%s)", auto_get(ctx,symAt), f.fullname(), closured)
	}

	if len(ctx.proj.configs) == 0 {
		// no need to check configuration
	} else if cfg := ctx.proj.configuration_sm(ctx); cfg == nil || !cfg.exists() {
		prompt(ctx, "%v\n", filename)
		if opts.mustConf {
			var d = opts.debug ; if d == 0 { d = 1 }
			erro(ctx, "no configuration (%v), try -conf first, in %v", cfg, ctx.proj)
		} else {
			debug(ctx, "no configuration (%v), try -conf first, in %v", cfg, ctx.proj)
		}
	}

	// Check previously configured files, we only configure once unless
	// optReconfig is true.
	var closure *scope
	if configuredFiles != nil {
		var okay bool
		closure, okay = configuredFiles[filename]
		if okay && closure != nil && !opts.reconfig { return }
	}

	defer func(s string, c *scope) { configuredFiles[s] = c } (filename, closure)

	var data bytes.Buffer
	if h := auto_get(ctx, symDash); !isNull(h) { args = append(args, h) }
	if dealArgs != nil { args = dealArgs(args, &data) }
	if dealData != nil {
		for _, arg := range args {
			if str := __string(ctx, arg); str != "" {
				dealData(str, &data)
			}
		}
	}

	if data.Len() == 0 {
		erro(ctx, "empty configuration data",
			_f("%v: %v %v", filename, auto_get(ctx,symAt), auto_get(ctx,symRangle)))
	} else if cfg := ctx.proj.configuration_sm(ctx); (cfg == nil || !cfg.exists()) && opts.debug>0 {
		// NOTE: TrimSpace to ease emacs *compilation* parse errors
		debug(ctx, "%v: %v\n%s\n", filename, auto_get(ctx,symAt), strings.TrimSpace(data.String()))
	}

	var (
		e error
		same bool
		status string
	)
	
	if opts.verbose { defer func(st time.Time) {
		if same {
			if true { return } else { status = "unchanged" }
		} else if status == "" {
			status = fmt.Sprintf("outdated (%s)", filename)
		}

		var d = time.Since(st)
		prompt(ctx, "update %v …… %s (in %v)\n", trimPromptString(filename), status, d)
		if d := opts.debug; d>0 { debug(ctx, "%v (%v)", auto_get(ctx, symAt), d) }
	}(time.Now())}

	// 2. Pure Integer Time Math (Zero GC)
	if f.exists() {
		if same, e = crc64CheckFileModeContent(ctx, filename, data.Bytes(), opts.mode); e != nil {
			erro(ctx, " crc64 checksum failed: %v", e)
			return
		}
		if same {
			var tt int64 = f._mtime // Blazing fast integer baseline
			for _, d := range merge(ctx.targets...) {
				if depFile, y := to_file(d); y && depFile.exists() { 
					if dt := depFile._mtime; dt > tt { tt = dt }
				}
			}
			
			// Only allocate time.Time if we physically need to execute the touch operation
			if tt > f._mtime { e = touch(ctx, f, 0, false, time.Unix(0, tt)) }
			return f
		}
	} else if dir := filepath.Dir(filename); opts.makePath && dir != "." && dir != string(filepath.Separator) {
		if e = os.MkdirAll(dir, os.FileMode(0755)); e != nil {
			erro(ctx, " %v", e)
		}
	}

	// 3. Modern Go standard library
	if e = os.WriteFile(filename, data.Bytes(), opts.mode); e != nil {
		erro(ctx, " %v", e)
	}

	// 4. Guaranteed Cache Update
	f.stat(ctx) // Automatically fetches the new ModTime and Size into the VFS cache
	if !f.exists() {
		erro(ctx, " failed to stat generated file: %v", filename)
	}

	if 0 < opts.debug {
		status = fmt.Sprintf("configured (%s, %d bytes)", filename, data.Len())
	} else {
		status = fmt.Sprintf("configured (%d bytes)", data.Len())
	}
	return f
}

type modifier_configureinput struct { modifier_ }

func (ctx *modifier_configureinput) x(args ...Value) (result any) {
	var opts = configureconvertOpts{ mode: os.FileMode(0600) }

	var dealArgs = func(args []Value, out *bytes.Buffer) []Value {
		var p = _project(ctx)

		// intern("configure.names") evaluates once at runtime, but ideally
		// you should pre-intern this globally or use a constant if it's frequent!
		if x, y := p.Lookup(intern("configure.names")).(*def); y {
			args = append(args, xmerge(ctx, x.value)...)
		}

		// CRITICAL FIX 1: Upgrade to integer hash map
		var configs = make(map[Symbol]*def)

		for _, a := range args {
			// CRITICAL FIX 2: Evaluate and immediately cross into integer domain
			var sym = intern(__string(ctx, a))

			if _, ok := configs[sym]; ok {
				continue
			} else if obj := p.resolve(ctx, sym); obj == nil { // Make sure p.resolve accepts Symbol!
				erro(ctx, "undefined %v", sym) // Symbol natively prints as string
				return nil
			} else if def, ok := obj.(*def); ok {
				configs[sym] = def
			}
		}

		for _, c := range p.configs {
			// CRITICAL FIX 3: Since 'c' is a *def, we completely bypass ident()
			// and just grab its pre-interned Symbol name!
			var sym = c.name

			if def, ok := p.Lookup(sym).(*def); ok { // Make sure p.Lookup accepts Symbol!
				configs[sym] = def
			}
		}

		for _, def := range configs {
			// Symbol implements Stringer, so %s works perfectly for def.name
			fmt.Fprintf(out, "#undef %s\n", def.name)
		}
		return args
	}

	return configureconvert(_execution(ctx), dealArgs, nil, &opts, args...)
}

// configure-file modifier (see also builtinConfigureFile), example usage:
//
//     config.h: config.h.in [(configure-file)]
//
type modifier_configurefile struct { modifier_ }
func (ctx *modifier_configurefile) x(args ...Value) (result any) {
    var opts = configureconvertOpts{ mode: os.FileMode(0600) }
    var convert = func(str string, out *bytes.Buffer) {
        configurestring(ctx, out, _project(ctx), str)
    }
    return configureconvert(_execution(ctx), nil, convert, &opts, args...)
}

// extract-configuration extracts configuration from C/C++ files, example usage:
//
//      config.h.in:[(extract-configuration)]: $(wildcard *.cpp)
//
type modifier_extractconfiguration struct { modifier_
    mode os.FileMode "mode"
    makePath bool "path"
    target string "target"
    rxs []*regexp.Regexp "rx,regex" // regexp.Compile(s)
}
func (ctx *modifier_extractconfiguration) x(args ...Value) (result any) {
	var pats []Value
	var pos = _position(ctx)
	for _, arg := range args {
		switch a := arg.(type) {
		case *group: pats = append(pats, a.elems...)
		default:     pats = append(pats, a)
		}
	}

	if len(pats) == 0 {
		erro(ctx, "extract-configuration: missing file names (patterns)")
		return
	}

	if len(ctx.rxs) == 0 {
		erro(ctx, "extract-configuration: missing -rx=... flags")
		return
	}

	if ctx.target == "" { ctx.target = "configuration" }

	var outFile string
	if target := auto_get(ctx,symAt); isNull(target) {
		erro(ctx, " target '@' is undefined")
		return
	} else {
		outFile = __string(ctx, target)
	}

	if ctx.makePath {
		if err := os.MkdirAll(filepath.Dir(outFile), os.FileMode(0755)); err != nil {
			erro(ctx, " make path failed: %v", err)
			return
		}
	}

	var fil, err = os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, ctx.mode)
	if err != nil {
		erro(ctx, " open file failed: %v", err)
		return
	}

	var out = bufio.NewWriter(fil)
	defer func() { out.Flush() ; fil.Close() } ()

	var depends, sources []Value
	if d := auto_get(ctx, symCaret); !isTrivial(d) { depends = xmerge(ctx, d) }

	var patsVal = ease(ctx, pats)
	for _, depend := range depends {
		var a []Value
		switch d := depend.(type) {
		case *file:
			if a = merge(call(ctx, symFilter, nil, patsVal, d)); a != nil {
				sources = append(sources, a...)
			}
		case *path:
			var s = __string(ctx, d)
			err = walkFiles(ctx, s, pats, func(file *file, err error) error {
				if err == nil { sources = append(sources, file) }
				return err
			})
		default:
			var s = __string(ctx, d)
			var dir = filepath.Dir(s)
			var name = filepath.Base(s)
			
			// Because _stat now accepts standard OS strings and auto-interns them, this remains unchanged
			var f = _stat(ctx, name, stat_dir{dir})
			
			if f == nil || !f.exists() {
				erro(ctx, " extract-configuration: `%s` file not found", name)
				return
			} else if f._isDir { // Ultra-fast boolean check replaces interface call
				err = walkFiles(ctx, s, pats, func(f *file, err error) error {
					if err == nil { sources = append(sources, f) }
					return err
				})
			} else if a = merge(call(ctx, symFilter, nil, patsVal, d)); a != nil {
				sources = append(sources, a...)
			}
		}
	}

	var exprs = make(map[string]struct{})

sourceloop:
	for _, source := range sources {
		var s string
		switch t := source.(type) {
		case *file: s = t.fullname()
		default:    s = __string(ctx, t)
		}

		var f *os.File
		if f, err = os.Open(s); err != nil {
			prompt(ctx, "%v: (configure) %v: %v\n", pos, source, err)
			continue sourceloop
		}

		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanLines)
	scanloop:
		for scanner.Scan() {
			var s = scanner.Text()
			for _, x := range ctx.rxs {
				if sm := x.FindStringSubmatch(s); sm != nil {
					exprs[sm[1]] = struct{}{}
					break scanloop
				}
			}
		}

		f.Close()
	}

	var keys []string
	for x := range exprs { keys = append(keys, x) }
	sort.Strings(keys)

	for _, k := range keys { fmt.Fprintf(out, "#%s :{(configure)}\n", k) }

	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "%s:{(configure -check)}\\\n", ctx.target)
	for _, k := range keys { fmt.Fprintf(out, "  %s \\\n", k) }
	fmt.Fprintf(out, "\n")
	return
}

const (
    vertag = "dev" // dev, alpha, beta, final
)

type property uint64

const (
    propDirtyOpts property = 1<<iota
    propErros
    propReversal
    propCache
    propUncache
)

type (
    mark_dirty     struct{ a []Value }
    act_dirt       struct{ a []Value }
    act_count_dia  struct{ t []diagtype }
    act_traversed  struct{ v Value }
    act_traverse   struct{ v Value }
    init_args      struct{ *automatic }
	set_workdir    struct{ s string }
    get_workdir    struct{}
    get_args       struct{}
	get_fatpos     struct{ p Pos }
    get_position   struct{}
    get_project    struct{}
    get_scope      struct{}
    get_closure_scopes struct{}
    no_position    struct{}
    on_errors      struct{ i int }
    param_name     struct{ i int }
    is_good_with   struct{ p property ; a []any }
    is_test_case   struct{}
    is_test_mode   struct{}
    is_test_univ   struct{}
)

func _workdir(ctx Context) (s string) {
	s, _ = do(ctx, get_workdir{}).(string)
	return
}

func paramName(ctx Context, n int) (s Symbol) {
	s, _ = do(ctx, param_name{n}).(Symbol)
    return
}

func diagCount(ctx Context, t ...diagtype) (i int) {
    i, _ = do(ctx, act_count_dia{t}).(int)
    return
}

type Context interface {
	cast(reflect.Type) Context
	do(Context, any) any
}

func do(c Context, o any) any { return c.do(c, o) }

func truly(ctx Context, ops ...any) (_ bool) {
	for _, op := range ops {
		switch t := do(ctx, op).(type) {
		case []*valcache: return len(t) > 0
		case bool: return t
		}
	}
	return
}

func try[T any](ctx Context, op any) (_ T) {
    if ctx != nil {
		if x, y := do(ctx, op).(T); y {
			return x
		}
	}
    return
}

func cast[T Context](ctx Context) (res T) {
    if ctx != nil {
        if t := ctx.cast(reflect.TypeOf(res)); t != nil {
          return t.(T)
        }
    }
    return
}

func icast(ctx Context, t reflect.Type) (res Context) {
    if v := reflect.ValueOf(ctx); v.Type() == t {
        res = ctx
    } else if i := _inner(v); i != nil {
        res = i.(Context).cast(t)
    }
    return
}

func _inner(v reflect.Value) (i Context) {
	if x, y := v.Interface().(interface{ inner() Context }); y {
		return x.inner()
	} else if t := v.Type(); t.Kind() == reflect.Struct {
		for n := 0; false && n < v.NumField(); n++ {
			if ft := t.Field(n); ft.Anonymous {
				var fv = v.FieldByIndex(ft.Index)
				if fv.CanInterface() {
					if f := fv.Interface(); ft.Name == "Context" {
						i, _ = f.(Context)
						return
					} else if i, y = f.(Context); y {
						return
					}
				}
				if fv.Type().Kind() == reflect.Struct && fv.CanAddr() {
					if fv = fv.Addr(); fv.CanInterface() {
						if i, y = fv.Interface().(Context) ; y {
							return
						}
					}
				}
			}
		}
		if x, y := t.FieldByName("Context"); y && x.Anonymous {
			if v = v.FieldByIndex(x.Index); v.IsValid() {
				if i, y = v.Interface().(Context) ; y {
					return
				}
				if false && v.Type().Kind() == reflect.Struct && v.CanAddr() {
					if i, y = v.Addr().Interface().(Context) ; y {
						return
					}
				}
			}
		} else if false {
			for n := 0; n < v.NumField(); n++ {
				if f := t.Field(n); f.Anonymous {
					var fv = v.FieldByIndex(f.Index)
					if fv.CanInterface() {
						if i, y = fv.Interface().(Context); y {
							return
						}
					}
					if fv.Type().Kind() == reflect.Struct && fv.CanAddr() {
						if fv = fv.Addr(); fv.CanInterface() {
							if i, y = fv.Interface().(Context) ; y {
								return
							}
						}
					}
				}
			}
		}
	} else if t.Kind() == reflect.Pointer {
		i = _inner(v.Elem())
	}
	return
}

func inner(c Context) Context { return _inner(reflect.ValueOf(c)) }

func _scope(ctx Context) (s *scope) {
    s, _ = do(ctx, get_scope{}).(*scope)
    return
}

func _project(ctx Context) (p *project) {
    p, _ = do(ctx, get_project{}).(*project)
    return
}

func auto_target_value(ctx Context) (res Value) {
    if val := auto_get(ctx, symAt); val == nil {
        if false { erro(ctx, "target is nil") }
    } else if v := expand(ctx, val); v == nil {
        erro(ctx, "multiple targets: %v → %v", val, v)
    } else {
        res = scalarize(v)
    }
    return
}

func auto_target_valstr(ctx Context) (val Value, str string) {
    if val = auto_target_value(ctx); val == nil {
        if false { erro(ctx, "target is nil") }
    } else {
        str, _ = as_fullname_string(ctx, val)
    }
    return
}

type stopframe string
type skipint int
type frames int
type callstack struct{
	num, frames, skip int
	stop string
}

var (
    callstackLine1 = regexp.MustCompile(`^(?:extbit\.io/)?((?:smart\.\(.+?\)\.)?.+?)(\(.*\))$`)
    callstackLine2 = regexp.MustCompile(`^	(.*?:\d+)(?: \+.*)?$`)
    callstackPanic = regexp.MustCompile(`^panic(\(.+\))$`)
    callstackSkips = regexp.MustCompile(`^(?:(?:testing\.tRunner`+
		`|created by testing\.(\*T)\.Run in goroutine [0-9]+`+ // skips: |erro|recovered
		`|(?:extbit\.io/)?(?:.+?)smart\.(?:do(?:_hit)?|tr(?:ace|uly|y)|(?:\*diagnostic|diagtracer)\.trace)`+
		`|runtime\.Goexit)\(.+\)|exit status [0-9]+)$`)
)
func _callstack(s string, i, j int, args ...any) (res []byte) {
    var nums []int
	var stop string
    var v = bytes.Split(rt_debug.Stack(), []byte{'\n'})

    i += 2 // skips this func
	for _, a := range args {
		switch t := a.(type) {
		case bool: if !t { return /* d */ }
		case int: nums = append(nums, t)
		case stopframe: stop, j = string(t), len(v) / 2
		case skipint: i += int(t) * 2
		case frames: if 0 < t { j += int(t) } else { j += len(v) / 2 }
		}
	}

	switch len(nums) {
	case 0: j += 1
	case 1: j += nums[0]
	case 2: j += nums[1]; i += nums[0]*2
	default: panic("too many stack nums")
	}

    var wasPanic bool
    for 0 < j && i+1 < len(v) {
        if callstackSkips.Match(v[i]) { i += 2; continue }

		sm1 := callstackLine1.FindSubmatch(v[i+0]) //extbit.io/smart.recovered(...)
		sm2 := callstackLine2.FindSubmatch(v[i+1]) //	/.../src/context.go:123 +0x456

		if sm1 != nil && sm2 != nil { n := i
			switch string(sm1[1]) {
			case stop: i, j = len(v), 0
			case "panic":
				if false { fmt.Printf("%s: %s `%s`\n", sm2[1], v[i+0], v[i+1]) }
				i, wasPanic = i+2, true
				continue
			}

			var e string
			if i, j = i+1, j-1; 0 < j && i < len(v) {
				if wasPanic { wasPanic, e = false, "	<---- panic" }
			} else {
				e = fmt.Sprintf("  (%d more)", (len(v)-n)/2)
			}

            res = append(res, sm2[1]...)
            res = append(res, []byte(":"+s+" ")...)
            res = append(res, sm1[1]...)
            res = append(res, sm1[2]...)
            res = append(res, []byte(e+"\n")...)
        } else {
			i += 1
		}
    }
    return
}

func debugSyntax(ctx Context, s string) (res bool) {
    if u := _universe(ctx); u != nil {
        for _, t := range u.debugSyntax { if res = t == s; res { break } }
    }
    return
}

const (
    diagInfo diagtype = iota
    diagWarn
    diagError
    diagPrompt
)

type diagtype int
type diagadd_cs_i int
type diagadd_cs_j int
type diagtext struct{}
type diagpoint struct{
    t diagtype
    position Position
    message string
	panic any
    stack []byte // see also rt_debug.Stack()
}

type too_many_diags       struct{ int }
type too_many_erros       struct{ int }
type trace_errors         struct{ Context ; int }
type trace_evoke_loop_err struct{ Context ; Value }
type trace_evoke_loop     struct{ Context }
type trace_val            struct{ int ; val Value }
type trace_ctx            struct{ int }
type trace                struct{}
type evoke_loop_null      struct{}
type evoke_loop_panic     struct{}

func (x trace_evoke_loop) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case evoke_loop_panic: return true
    }
    return x.Context.do(ctx, op)
}

func (t trace_evoke_loop_err) String() string {
    return "evoke loop: " + ts(t.Value)
}

func (t trace_errors) String() string {
    return fmt.Sprintf("trace %d errors, %v", t.int, ts(t.Context))
}

func (t too_many_diags) String() string { return fmt.Sprintf("too many diagnostics (%d)", t.int) }
func (t too_many_erros) String() string { return fmt.Sprintf("too many errors (%d)", t.int) }

// NOTE: never recover test_fail in recovered, it will break the test runner
type test_fail struct{ Context; i int }
type test_failed struct{}

func (t test_fail) Error() string { return typeof(t.Context)+": test fail" }
func (_ test_failed) Error() string { return "test failed" }

func recovered(ctx Context) {
    var ( n int ; te trace_errors )

	for e := recover(); e != nil; e = recover() {
		switch n += 1 ; t := e.(type) {
		case              failure: erro(t.Context, t.Error())
		case                Value: erro(ctx, "trace: %s", ts(t))
		case               string: erro(ctx, "trace: %s", t)
		case        runtime.Error: erro(ctx, "trace: %s", t.Error())
		case       too_many_diags: erro(ctx, "too many diagnostics (%v)", t.int)
		case       too_many_erros: erro(ctx, "too many errors (%v)", t.int)
		case trace_evoke_loop_err: erro(pc(ctx,t.Value), "evoke loop (%s)", ts(t.Value))
		case trace_errors: te = t
		case test_fail:
			if t.i += 1; t.i == 1 {
				debug(ctx, "%s: failed (%d panics)", typeof(t.Context), n, callstack{frames:-1})
			}
			if flush(ctx); true { runtime.Goexit() } else if false { panic(test_failed{}) } else { return }
		default:
			if true { panic(e) } else { note(ctx, "%s: %v", typeof(e), e) }
		}
	}

    if 0 < n {
        debug(ctx, "%s (%d panics)", typeof(te.Context), n, callstack{frames:-1})
        flush(ctx)
    }

    if false && truly(ctx, is_test_mode{}) {
        if te.Context != nil && 0 < te.int {
            panic(test_fail{te.Context, 0}) // rethrow to break the test runner
        }
    }
}

const diagnostic_limit = 10_000
var   diagnostic_limit_erros = 520
var   diagnostic_limit_bytes = 1_000_000

func _diagnostic(c Context) *diagnostic { return cast[*diagnostic](c) }
func _f(f string, a ...any) *diag_point { return &diag_point{0, f, a} }

type diag_struct struct{ t diagtype; f string; a []any }
type diag_trace diag_struct
type diag_point diag_struct
type diag_flush struct{}
type diagnostic struct{
    Context
    sync.Mutex
    points []*diagpoint
    erros int // number of flushed erros
    flushed int // in bytes
}
func (d *diagnostic) aquire() func() { d.Lock(); return d.Unlock }
func (d *diagnostic) cast(t reflect.Type) Context { return icast(d, t) }
func (d *diagnostic) inner() Context { return d.Context }
func (d *diagnostic) do(ctx Context, op any) (_ any) {
    switch t := op.(type) {
    case property: if t&propErros != 0 { return d.erros }
    case diag_flush   : return d.flush(ctx)
	case diag_point   : return d.point(ctx, t.t, t.f, t.a...)
    case act_count_dia: return d.count(t.t...)
    }
    if d.Context == nil { return }
    return d.Context.do(ctx, op)
}
func (d *diagnostic) add(p *diagpoint) *diagpoint {
    defer d.aquire()()
    if i := len(d.points); diagnostic_limit < i {
        panic(too_many_diags{i})
    }
	d.points = append(d.points, p)
	return p
}
func (d *diagnostic) point(ctx Context, dt diagtype, f string, args ...any) *diagpoint {
	if dt != diagPrompt { f = strings.TrimSpace(f) }
	return d.add(&diagpoint{dt, _position(ctx), fmt.Sprintf(f, args...), nil, nil})
}
func (d *diagnostic) count(dt ...diagtype) (errs int) {
	defer d.aquire()()
	for _, d := range d.points {
		for _, t := range dt {
			if d.t == t { errs += 1 ; break }
		}
	}
	return
}
func (d *diagnostic) flush(ctx Context) (errs int) {
    const count_bytes = false

	defer func() { if d.erros += errs ; errs > 0 { do(ctx, on_errors{errs}) }} ()

	print := func(p *diagpoint, pend bool) (_ bool) {
		defer func() {
			if x, y := diagnostic_limit_erros, d.erros; 0 < x && x < y {
				if false { d.erros = 0 } // reset to avoid causing next panics
				panic(too_many_erros{y})
			}
			if x, y := diagnostic_limit_bytes, d.flushed; 0 < x && x < y && false {
				if false { d.flushed = 0 } // reset to avoid causing next panics
				panic(too_many_diags{y})
			}
		} ()

        pos, msg := p.position.String(), p.message

        if count_bytes {
            d.flushed += len(pos) + len(msg)
        } else {
            d.flushed += 1
        }

		if p.panic != nil {
			msg += fmt.Sprintf(": %v", p.panic)
		}

		switch p.t {
		case diagInfo: fmt.Fprintf(stderr, "%v:info: %s\n", pos, msg)
		case diagWarn: fmt.Fprintf(stderr, "%v:warning: %s\n", pos, msg)
		case diagPrompt:
			if msg != "" { fmt.Fprintf(stderr, "%s", msg) }
			if pend && !strings.HasSuffix(msg, "\n") { return true }
		case diagError:
			if errs += 1 ; p.stack == nil {
				fmt.Fprintf(stderr, "%v:error: %s\n", pos, msg)
			} else {
				fmt.Fprintf(stderr, "%v: %s\n", pos, msg)
			}
		}

        if p.stack != nil {
            fmt.Fprintf(stderr, "%s\n", bytes.TrimSpace(p.stack))
            if count_bytes {
                d.flushed += len(p.stack)
            } else {
                d.flushed += 1 + bytes.Count(p.stack, []byte("\n"))
            }
        }
        return
    }

	d.Lock(); defer d.Unlock()
	for 0 < len(d.points) {
		var point = d.points[0]
		d.points = d.points[1:]
		if print(point, true); 16 < errs {
			fmt.Fprintf(stderr, "%v: too many errors (%d)\n", _position(ctx), errs)
		}
	}
    return
}

func flush(ctx Context) (i int) { i, _ = do(ctx, diag_flush{}).(int); return }

func prompt(ctx Context, f any, a ...any) *diagpoint { return debug(ctx, f, append(a, diagtext{})...) }
func info(ctx Context, f any, a ...any) *diagpoint { return debug(ctx, f, append(a, diagInfo, diagadd_cs_i(1))...) }
func warn(ctx Context, f any, a ...any) *diagpoint { return debug(ctx, f, append(a, diagWarn, diagadd_cs_i(1))...) }
func erro(ctx Context, f any, a ...any) *diagpoint { return debug(ctx, f, append(a, diagError, diagadd_cs_i(1))...) }
func note(ctx Context, f any, a ...any) *diagpoint {
	// 'note' is a prompt that explicitly wants the position prefix
	return debug(ctx, sf("%v:%v", _position(ctx), f), append(a, diagPrompt)...)
}

// Helper function to replace diagstack behaviour seamlessly
func debugstack(ctx Context, n int, dt diagtype, a ...any) *diagpoint {
	var f any = ""
	if len(a) > 0 {
		if x, y := a[0].(string); y {
			f, a = x, a[1:] // separate the format string and args
		} else {
			f, a = a[0], a[1:] // handle non-string leading args safely
		}
	}
	return debug(ctx, f, append(a, dt, trace_ctx{n})...)
}

func infostack(ctx Context, n int, a ...any) *diagpoint { return debugstack(ctx, n, diagInfo, a...) }
func warnstack(ctx Context, n int, a ...any) *diagpoint { return debugstack(ctx, n, diagWarn, a...) }
func errostack(ctx Context, n int, a ...any) *diagpoint { return debugstack(ctx, n, diagError, a...) }
func notestack(ctx Context, n int, a ...any) *diagpoint { return debugstack(ctx, n, diagPrompt, a...) }

var _debug_m sync.Mutex
func debug(ctx Context, f any, a ...any) *diagpoint {
	_debug_m.Lock(); defer _debug_m.Unlock()

	var tr = false
	var trCtx int
	var trVal int
	var noCS bool
	var cs callstack
	var cs_prefix = "info:"
	var cs_i, cs_j int = 5, 0
	var dt diagtype
	var dias []*diag_point
	var args []any

	for _, a := range a {
		switch t := a.(type) {
		case diagtype: dt = t
		case trace: tr = true
		case trace_ctx: trCtx = t.int
		case trace_val: trVal = t.int
		case diagadd_cs_i: cs_i += int(t)
		case diagadd_cs_j: cs_j += int(t)
		case diagtext: if noCS = true; dt == 0 { dt = diagPrompt }
		case callstack:
			if 0 < t.num     { cs.num = t.num }
			if 0 < t.skip    { cs.skip = t.skip }
			if 0 != t.frames { cs.frames = t.frames }
			if "" != t.stop  { cs.stop = t.stop }
		case []*diag_point:
			dias = append(dias, t...)
		case *diag_point:
			dias = append(dias, t)
		default:
			args = append(args, t)
		}
	}

	var p *diagpoint
	var s string

	// CRITICAL FIX: The engine's internal logger automatically prepends the position
	// for Info, Warn, and Error. We must leave 's' empty to avoid double prefixes!
	// We only auto-prepend for direct calls to debug() where no 'dt' is specified.
	if dt == 0 && !noCS {
		s = _position(ctx).String() + ": "
	}
	if dt == 0 {
		dt = diagPrompt
	}

	switch t := f.(type) {
	case *diag_point:
		if t.f != "" {
			nl := "\n"
			if noCS { nl = "" } // Bypass newline for bare prompts
			p, _ = do(ctx, diag_point{dt, s+t.f+nl, t.a}).(*diagpoint)
		}
	case []*diag_point:
		for _, t := range t {
			nl := "\n"
			if noCS { nl = "" } // Bypass newline for bare prompts
			p, _ = do(ctx, diag_point{dt, s+t.f+nl, t.a}).(*diagpoint)
		}
	case string:
		if noCS {
			// CRITICAL FIX: Pass the raw string exactly as-is for bare prompts
			p, _ = do(ctx, diag_point{dt, s+t, args}).(*diagpoint)
		} else {
			for _, t := range strings.Split(t, "\n") {
				if t == "" { continue }
				p, _ = do(ctx, diag_point{dt, s+t+"\n", args}).(*diagpoint)
			}
		}
	default:
		nl := "\n"
		if noCS { nl = "" } // Bypass newline for bare prompts
		p, _ = do(ctx, diag_point{dt, s+typeof(t)+": %v"+nl, args}).(*diagpoint)
	}

	if n := trCtx; n > 0 {
		var pos = _position(ctx)
		for c := inner(ctx); c != nil && 0 < n && pos.valid(); c = inner(c) {
			var _pos = _position(c)
			if !_pos.valid() || _pos.same(&pos) { continue }

			n -= 1
			pos = _pos

			proj := _project(c)
			s := pos.String() + ": " + typeof(c) + ": "
			if proj == nil {
				s += "<nil>"
			} else {
				s += proj.name.String()
			}

			if e := _entry(c); e != nil {
				if t, _, _ := entryIndicator(c, e); t == "" {
					s += ": " + ident(ctx, e)
				} else {
					s += ": " + t
				}
			}

			if false { s += " ; " + ts(c) }

			p, _ = do(c, diag_point{dt, s+"\n", nil}).(*diagpoint)
		}
	}

	// CRITICAL ADDITION: trace_val{n} iterates over resolved Value args to print their structural positions
	if n := trVal; n > 0 {
		for _, arg := range args {
			if n <= 0 { break }
			if v, ok := arg.(Value); ok {
				n -= 1
				posStr := ""
				if pos := v.Pos(); pos != 0 { // or pos.IsValid() if supported by your Pos type
					// Dynamically resolve the compact AST Pos into a fat Position using the bridge!
					if fat, ok := do(ctx, get_fatpos{pos}).(Position); ok && fat.valid() {
						posStr = fat.String() + ": "
					} else {
						// Fallback to the thin pos if fat resolution fails for any reason
						posStr = fmt.Sprintf("%v: ", pos)
					}
				}

				str := posStr + typeof(v)
				if false { str += " ; " + ts(v) }

				p, _ = do(ctx, diag_point{dt, str+"\n", nil}).(*diagpoint)
			}
		}
	}

	for _, d := range dias {
		p, _ = do(ctx, diag_point{dt, s+d.f+"\n", d.a}).(*diagpoint)
	}

	if noCS { return p }
	if args = []any{}; p == nil { return nil }
	if cs.num > 0     { args = append(args, cs.num) }
	if cs.skip > 0    { args = append(args, skipint(cs.skip)) }
	if cs.frames != 0 { args = append(args, frames(cs.frames)) }
	if cs.stop != ""  { args = append(args, stopframe(cs.stop)) }
	if p.stack = _callstack(cs_prefix, cs_i, cs_j, args...); true { flush(ctx) }
	if tr {
		if truly(ctx, is_test_mode{}) {
			p.panic = test_fail{ctx, 0}
		} else {
			p.panic = trace_errors{ctx, diagCount(ctx, diagError)}
		}
		panic(p.panic)
	}
	return p
}

func _position(ctx Context) (_ Position) {
	switch x := do(ctx, get_position{}).(type) {
	case Position:
		// Fast path: Context natively provided a fat Position (e.g., from the universe or *xloc)
		if x.valid() { return x }
	case Pos:
		// Resolution path: Context provided a compact AST Pos.
		// We dynamically resolve it into a fat Position using our bridge!
		if x.IsValid() {
			if fat, ok := do(ctx, get_fatpos{x}).(Position); ok {
				return fat
			}
		}
	}

	// Fallback (matches your `else if true { return }` logic)
	return
}

func _pos(ctx Context) Pos {
	// 1. Context/Evaluation Path: Extract a compact Pos from the runtime context stack!
	// This allows posctx and evocation to inject specific AST node positions.
	switch x := do(ctx, get_position{}).(type) {
	case Pos:
		if x.IsValid() { return x }
	case positioner:
		if p := x.Pos(); p.IsValid() { return p }
	}

	// 2. Parse-Time Fallback: Extract the exact compact integer
	// offset from the active parser if no context explicitly overrides it.
	if p, ok := do(ctx, get_parser{}).(*parser); ok && p != nil {
		if p.pos.IsValid() { return p.pos }
	}

	return 0 // 0 represents NoPos
}

func walkSmartBaseDirs(ctx Context, cwd string, vis func(string) bool) (s string) {
    for s = cwd ; s != "" ; {
        var f = _stat(ctx, ".smart", stat_dir{s})
        if f != nil && f._mtime != 0 && f._isDir && !vis(s) { break }
        if up := filepath.Dir(s); up == s { break } else { s = up }
    }
    if s == "" { s = cwd }
    return
}

func joinTmpPath(ctx Context, base, rel string) string {
    if baseTmpPath == "" {
        var s = walkSmartBaseDirs(ctx, base, func(d string) bool {
            return false // return the first found
        })
        if s == "" {
            // FIXME: Windows system temporary path.
            s = filepath.Join("/", "tmp")
        }
        baseTmpPath = s
    }
    if s := filepath.Dir(rel); s != "" {
        if strings.HasSuffix(base, s) {
            // In case like '/foo/bar/a/b/c/x'+'a/b/c/x', we set
            // rel to 'x' to produce 'foo/bar/.smart/tmp/a/b/c/x'.
            rel = filepath.Base(rel)
        } else if t, _ := filepath.Rel(baseTmpPath, base); strings.HasPrefix(t, ".smart"+pathSep) {
            // In case like '/foo/bar/.smart/a/b/x'+'a/e/f/x', we set
            // base to '/foo/bar/.smart' to produce 'foo/bar/.smart/tmp/a/e/f/x'.
            v1 := strings.Split(t, pathSep)
            v2 := strings.Split(s, pathSep)
            for i := len(v1)-1; i >= 0; i -= 1 {
                if v1[i] == v2[0] {
                    base = filepath.Join(v1[i-1:]...)
                    break
                }
            }
        }
    }
    if s, err := filepath.Rel(baseTmpPath, filepath.Join(base, rel)); err == nil {
        rel = s
    }
    if s := ".smart"+pathSep; strings.HasPrefix(rel, s) { // .smart/
        rel = strings.TrimPrefix(rel, s)
        if s = "modules"+pathSep; strings.HasPrefix(rel, s) { // modules/
            rel = strings.TrimPrefix(rel, s)
        }
    }
    rel = strings.Replace(rel, "..", "_", -1)
    if strings.HasPrefix(rel, "tmp"+pathSep) {
        return filepath.Join(baseTmpPath, ".smart", rel)
    }
    return filepath.Join(baseTmpPath, ".smart", "tmp", rel)
}

func positionForDir(dir string) (pos Position) {
    if strings.HasSuffix(dir, mainFileName) || strings.HasSuffix(dir, deprFileName) {
        pos.Filename = dir
    } else if _, e := os.Stat(filepath.Join(dir, mainFileName)); e == nil {
        pos.Filename = filepath.Join(dir, mainFileName)
    } else if _, e := os.Stat(filepath.Join(dir, deprFileName)); e == nil {
        pos.Filename = filepath.Join(dir, deprFileName)
    } else {
        pos.Filename = dir
    }
    pos.Line = 1
    return
}

func (ctx *universe) loadSearchPaths(s string) (paths []string) {
	// 1. Bulk I/O: Read the entire file at once instead of line-by-line syscalls
	data, err := os.ReadFile(filepath.Join(s, ".search"))
	if err != nil {
		return // File doesn't exist or can't be read, return nil
	}

	// Iterate over bytes directly to avoid string allocations for comments/blank lines
	lines := bytes.Split(data, []byte{'\n'})

	// Pre-allocate paths slice to minimize dynamic array resizing
	paths = make([]string, 0, len(lines))

	for _, bLine := range lines {
		// Fast in-place byte trimming
		bLine = bytes.TrimSpace(bLine)

		// Zero-allocation checks for empty lines and comments
		if len(bLine) == 0 || bLine[0] == '#' {
			continue
		}

		// Only allocate a string once we know it's a valid path payload
		line := string(bLine)

		if filepath.IsAbs(line) {
			line = filepath.Clean(line)
		} else {
			// PERF: filepath.Join inherently calls filepath.Clean internally.
			// Doing Clean(Join()) was doing double-work.
			line = filepath.Join(s, line)
		}

		// Syscall: Stat the file to ensure it's a directory
		if fi, err := os.Stat(line); err == nil && fi.IsDir() {
			paths = append(paths, line)
		}
	}
	return paths
}

type main_ctx struct{ Context }
func (m main_ctx) inner() Context { return m.Context }
func (m main_ctx) cast(t reflect.Type) Context { return icast(m, t) }
func (m main_ctx) do(c Context, op any) any {
	return m.Context.do(c, op)
}

func Main() {
	if checkpoints { panic("Smart in testmode!") }

    ctx := new_universe()
    ctx.load(main_ctx{ctx})

    if ctx.flush(ctx) > 0 {
        prompt(ctx, "loading work got %d errors\n", ctx.erros)
    } else if ctx.help {
        do_helpscreen(ctx)
    } else if ctx.printFlags {
        print_flag_trace(ctx)
    } else if ctx.printConfig {
        print_configuration(ctx)
    } else if numUpdatedPlugins > 0 { // see buildPlugin
        prompt(ctx, "plugins updated, please relaunch.\n")
    } else if result := ctx.run(); ctx.flush(ctx) > 0 {
        prompt(ctx, "run work got %d errors\n", ctx.erros)
    } else if result != nil {
        for i, v := range result {
            if s := ""; v == nil {
                s = "<nil>"
            } else if s = strings.TrimSpace(__string(ctx, v)); s == "" {
                continue
            } else if i == 0 {
                fmt.Fprintf(stderr, "%s", s)
            } else {
                fmt.Fprintf(stderr, ", %s", s)
            }
        }
        fmt.Fprintf(stderr, "\n")
    }
}

var searchPaths searchlist
var baseTmpPath string // the base tmp path initialized only once.
var baseWorkDir = func () string {
    if s, e := os.Getwd(); e == nil { return s } else { panic(e) }
} ()

type searchlist []string
func (sl *searchlist) String() string { return fmt.Sprint(*sl) }
func (sl *searchlist) set(s string) error {
    *sl = append(*sl, strings.Split(s, ",")...)
    return nil
}
func (sl *searchlist) has(s string) (_ bool) {
    for _, p := range *sl { if p == s { return true }}
    return
}

type hooks struct {
    assert func(Context, Value, bool) bool
    debug func(Context, string, []Value)
    error func(Context, string, []Value)
}

type packagetype uint8

const (
    packageUnknown packagetype = iota
    packageSmart  // smart package
    packageConfig // pkgconfig
)

type packageinfo struct {
    *project
    t packagetype // smart, pkgconfig, cmake, etc.
}

// ResolvePosition converts a compact AST 'Pos' back into a human-readable 'Position'
func ResolvePosition(ctx Context, v Value) Position {
	if v == nil {
		return Position{}
	}

	// 1. Runtime / Synthetic Position Escape Hatch
	// If the value is explicitly wrapped in an external location (*xloc), use its fat struct!
	if l, ok := v.(*xloc); ok {
		return l.pos
	}

	// (Optional but robust) Check if any other node implements the fat Position interface
	if p, ok := v.(interface{ Position() Position }); ok {
		if pos := p.Position(); pos.valid() {
			return pos
		}
	}

	// 2. Parse-Time Position Resolution
	pos := v.Pos() // The compact integer
	if !pos.IsValid() {
		return Position{} // NoPos
	}

	// Safely retrieve the fset from the universe
	if u := _universe(ctx); u != nil && u.fset != nil {
		return u.fset.Position(pos)
	}

	return Position{}
}

func _universe(c Context) *universe { return cast[*universe](c) }

type universe struct {
    diagnostic
    commandline

    *scope
    *globe

	launchTime time.Time

    fset *fileset

    workdir string
    prefix  string // FIXME: prefix for distribution
    paths   searchlist
    statcache map[string]*filebase // file.fullname() -> File
    statmutex sync.Mutex

    hooks hooks
}
func (ctx *universe) String() string { return "universe" }
func (ctx *universe) inner() Context { return &ctx.diagnostic }
func (ctx *universe) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.diagnostic.cast(t)
}
func (ctx *universe) _position() (p Position) {
    if ctx.globe != nil && ctx.globe.main != nil && ctx.fset != nil {
		p = ctx.fset.Position(ctx.globe.main.pos)
    } else {
		p.Filename, p.Line, p.Column = _workdir(ctx), 0, 0
	}
    return
}
func (ctx *universe) ts(t string) string {
    var s = ts(ctx.Context)
    if  s == "{}"  {
        s, _ = filepath.Rel(baseWorkDir, ctx.workdir)
        s = bases(3, s, "testdata", true)
        if s == "." || s == "" { return "{="+t+"}" }
    }
    return "{="+t+" "+s+"}"
}
func (ctx *universe) trimSpecPath(c Context, spec string) string {
    spec = strings.ReplaceAll(spec, "../", "")
    for _, s := range ctx.paths {
        if s += pathSep; strings.HasPrefix(spec, s) {
            spec = strings.TrimPrefix(spec, s)
            break
        }
    }
    if s := ctx.workdir+pathSep; strings.HasPrefix(spec, s) {
        spec = strings.TrimPrefix(spec, s)
    }
    return spec
}
func (ctx *universe) do(_ctx Context, op any) (res any) {
    switch t := op.(type) {
    case on_errors:
        if ctx.panicFailureOnFlushedErrors && truly(_ctx, is_test_mode{}) {
            if 0 < t.i { panic(_failure(ctx, "got %d errors", t.i)) }
            res = true
        }
        return

	case get_fatpos:
		var p Position
		if ctx.fset != nil && t.p.IsValid() { p = ctx.fset.Position(t.p) } // FIX: Use the receiver 'ctx'
		return p

    case get_position:
        p := Position{}
        p.Filename = ctx.workdir
        return

    case get_workdir: return ctx.workdir
    case get_scope: if ctx.scope != nil { return ctx.scope }
    case get_project: if ctx.globe != nil { return ctx.globe.main }
    case no_exec: if ctx.noExec { return ctx.noExec }
	case is_test_mode: if ctx.testMode { return true }
    case is_test_univ: return ctx.testMode
    }
    return ctx.diagnostic.do(_ctx, op)
}

type commandline struct {
    help            bool `h,help`

    debug           bool `d,db,debug`
    debugErrors     bool `de,dberro,debug-errors`
    debugWarns      bool `dw,dbwarn,debug-warns`
    debugInfos      bool `di,dbinfo,debug-infos`
    debugPrompt     bool `dp,dbprom,debug-prompt`
    debugSyntax []string `ds,dbsyntax,debug-syntax`

    profile         bool `profile`
    cpuProfile      string `cpu-profile`
    memProfile      string `mem-profile`

    printConfig     bool `opts,print-options,printoptions`
    printFlags      bool `flags,print-flags,printflags`

    buildPlugins    bool `bp,bup,build-plugins,buildplugins`

    silentOptionalArrow bool

    verbose         bool `v,verb,verbose`
    verboseBreaks   bool `vb,vbrk,verbose-breaks`
    verboseChecks   bool `vc,vchk,verbose-checks`
    verboseImport   bool `vi,vimp,verbose-import`
    verboseParse    bool `vp,vpar,verbose-parsing`
    verboseUsing    bool `vu,vuse,verbose-using`
    verboseExecFlags bool `vxf,verbose-exec-flag`

    allowClosureFilemap bool `cf,closure-filemap,closure-files`

    cleanDotCache   bool `clcac,clean-cache,clear-cache;rmc,rm-cache`
    cleanDotDeps    bool `cldep,clean-deps,clear-deps;rmd,rm-deps`
    cleanDotGrep    bool `clgrp,clean-grep,clear-grep;rmg,rm-grep`
    cleanTmpDirs    bool `cltmp,clean-temp,clear-temp;rmt,rm-temp`

    checkLoadGraph  bool `ckld,check-loads`

    reconfigure     bool `rc,reconf,reconfig,reconfigure`

    saveGrepSource  bool `savgs,save-grep-source`

    noRun           bool `nor,no-run`
    noExec          bool `nox,ne,no-exec,no-execute`  // optionNoExec
    noDeps          bool `nod,no-deps`
    noGrep          bool `nog,no-grep`
    noDepsGrep      bool `nodg,ngd,no-deps-grep,no-grep-deps`
    noImportFiles   bool `noif,no-import-files`

    parallel        bool `par,para,parallel`

    testMode        bool `test,test-mode`
    fastMode        bool `fast,fast-mode`
    errorUncache    bool `eu,error-uncache,error-no-cache`
    panicFailureOnFlushedErrors bool `foe,fail-on-errors`

    traceLaunch     bool `tl,trace-launch`
    traceParsing    bool `tp,trace-parse`
    traceExecutor   bool `te,trace-executor`
    traceExec       bool `tx,trace-exec`
    traceEntering   bool `ti,trace-entering`
    traceConfig     bool `tc,trace-config`

    slow time.Duration `slow` // time.Millisecond
}

func _commandline() commandline { return commandline{
    debugPrompt: true,
    debugErrors: true,
    debugWarns:  true,
    debugInfos:  true,

    fastMode: true,
    parallel: false, // FIXME: program.traverse not working in parallel

    panicFailureOnFlushedErrors: true,
    silentOptionalArrow: false,

    slow: 2999 * time.Millisecond,
}}

func new_universe(ii ...any) (ctx *universe) {
	ctx = &universe{
		launchTime: time.Now(),
		statcache:  make(map[string]*filebase),
		scope:      new_scope(nil, nil, `universe`),
		fset:       new_fileset(),
		paths:      searchPaths,
		workdir:    baseWorkDir,
	}

	cl := true
	for _, i := range ii {
		switch t := i.(type) {
		case  hooks:       ctx.hooks =  t
		case *hooks:       ctx.hooks = *t
		case  commandline: ctx.commandline, cl =  t, false
		case *commandline: ctx.commandline, cl = *t, false
		case set_workdir:
			ctx.workdir = t.s
			if _, e := os.Stat(t.s); e != nil {
				panic(e)
			}
		}
	}
	if cl { ctx.commandline = _commandline() }

	// Bootstrap top-level AST arguments
	ctx.scope.def(ctx, defVoid, symSMART, ease(ctx, os.Args[0]))
	ctx.scope.def(ctx, defVoid, symSMART_ARGS, ease(ctx, os.Args[1:]))

	// Pre-register all builtin functions
	for name, f := range builtins {
		if _, alt := ctx.scope.builtin(ctx, name, f); alt != nil {
			panic(fmt.Sprintf("builtin '%s' already defined", name))
		}
	}

	ctx.globe = &globe{
		scope:       new_scope(ctx.scope, nil, `globe`),
		args:        make(map[Value][]Value),
		flagEntries: make(map[string][]entry),
		loaded:      make(map[string]*project),
	}

	var pos Pos = 0 // ctx._position()
	ctx.globe.os    = ctx.globe.def(ctx, defVoid, symDotOS, _rw(pos, runtime.GOOS))
	ctx.globe.mode  = ctx.globe.def(ctx, defVoid, symDotMode, _null(pos))
	ctx.globe.goals = ctx.globe.def(ctx, defVoid, symDotGoals, _none(pos))

	// Pre-allocate to prevent heap thrashing during directory walk
	var paths = make(searchlist, 0, 4)

	// == Hermetic Test Isolation ==
	// Inject the testdata modules as the highest priority search path
	if checkpoints {
		paths = append(paths, filepath.Join(baseWorkDir, "testdata", ".smart", "modules"))
	} else {
		walkSmartBaseDirs(ctx, ctx.workdir, func(s string) bool {
			if baseTmpPath == "" { baseTmpPath = s }
			paths = append(paths, filepath.Join(s, ".smart", "modules"))
			return true
		})
	}

	// NOTE: yields <prefix>usr/lib/smart/modules
	// NOTE: ctx.prefix could be empty, thus starting from root "/" if so.
	prefix := ctx.prefix ; if prefix == "" { prefix = pathSep }
	usrLib := filepath.Join(prefix+"usr", "lib", "smart", "modules") // e.g. /usr/lib/smart/modules
	paths = append(paths, filepath.Clean(usrLib))

	// Allocate a perfectly sized new array to safely prepend the paths.
	cache := make([]string, 0, len(paths)+len(ctx.paths))
	ctx.paths = append(append(cache, paths...), ctx.paths...)

	// Load dynamic search directories
	for _, s := range paths {
		if t := ctx.loadSearchPaths(s); len(t) > 0 { ctx.paths = t }
	}
	return ctx
}

func (ctx *universe) addPaths(paths ...string) (err error) {
	for _, s := range paths {
		if s, err = filepath.Abs(s); err != nil { break }
		if i, _ := os.Stat(s); i != nil && i.IsDir() {
			debug(ctx, "%s", s)
			ctx.paths = append(ctx.paths, s)
		} else {
			erro(ctx, "not a directory: '%s'", s, trace{})
		}
	}
	return nil
}

func cpu_profile(ctx Context, name string, heap ...bool) func() {
    var fn string
    if filepath.IsAbs(name) {
		fn = name
	} else if m := _universe(ctx).globe.main; m == nil {
		erro(ctx, "%s: main project is nil", name, trace{})
	} else if f := m.tempfile(ctx, name); f == nil {
        fn = filepath.Join(_workdir(ctx), name)
    } else {
        fn = f.fullname()
    }

    f, e := os.Create(fn)
    if e != nil {
        erro(ctx, "%s", ts(e), trace{})
    } else if e = pprof.StartCPUProfile(f); e != nil {
        erro(ctx, "%s", ts(e), trace{})
    }
    return func() { if f != nil {
        if e != nil { pprof.StopCPUProfile() }
        if heap != nil && heap[0] { runtime.GC() // update memory statistics
            if e = pprof.WriteHeapProfile(f); e != nil {
                erro(ctx, "WriteHeapProfile: %v", e, trace{})
            }
        }
        f.Close()
    }}
}

func heap_profile(ctx Context, name string) func() {
    var fn string
    if filepath.IsAbs(name) {
		fn = name
	} else if m := _universe(ctx).globe.main; m == nil {
		erro(ctx, "%s: main project is nil", name, trace{})
	} else if f := m.tempfile(ctx, name); f == nil {
        fn = filepath.Join(_workdir(ctx), name)
    } else {
        fn = f.fullname()
    }

    f, e := os.Create(fn)
    if e != nil {
        erro(ctx, "%s", ts(e), trace{})
    }
    return func() { if f != nil {
        if e != nil { pprof.StopCPUProfile() }
        runtime.GC() // update memory statistics

		e = pprof.WriteHeapProfile(f)
        f.Close()

        if e != nil {
            erro(ctx, "%v", ts(e), trace{})
        }
    }}
}

func updateGoal(ctx Context, goal Value, args []Value) (result []Value) {
    switch g := goal.(type) {
    case *rule:
        var ok bool
        if result, ok = execute_entry(ctx, g, args...); !ok {
            erro(ctx, "update '%v' failed", g)
        }
    default:
        erro(ctx, "not an entry: %v", ts(goal, ctx))
    }
    return
}

func (l ul) parseArgs(a []string) {
    var args []Value
	var base = l.workdir

	if s := strings.Join(a, " "); s != "" {
		if v := l.text(l.universe, base, s); v != nil {
			args = parseOpts(l.universe, &l.commandline, merge(v)...)
		}
	}

    if v := l.fastMode; v { // Turn off many things for fast mode:
        //l.noImportFiles = v
        l.noDepsGrep = v
        l.noDeps = v
        l.noGrep = v
    }

    var mode = new(word)

    for _, target := range args {
        switch t := target.(type) {
        case *pair: l.globe.pairs = append(l.globe.pairs, t)
        case  flag: l.globe.flags = append(l.globe.flags, t)
            if s := __string(l.universe, t.Value); s == "clean" {
                mode.pos, mode.s = t.Pos(), symClean
            }
        case *argumented:
            l.globe.args[t.Value] = t.args
            if f, y := t.Value.(flag); y {
                l.globe.flags = append(l.globe.flags, f)
            } else {
                l.globe.goals.append(l.universe, t/*.Value*/)
            }
        default:
            l.globe.goals.append(l.universe, t)
        }
    }

    if mode.s == symEmpty { mode.s = symGoals }

    l.globe.mode.value = mode
}

func (u *universe) load(ctx Context) {
    if u.traceLaunch { defer un(l_trace(l_launch, "universe.load")) }

    if false { loadGrepCache(ctx) }

    if s := filepath.Join(u.workdir, mainFileName); s != "" {
        if _, e := os.Stat(s); e != nil {
            s = filepath.Join(u.workdir, deprFileName)
            if _, e := os.Stat(s); e != nil { s = "" }
        }
    }

    u.globe.top = &loader{term:term{ctx, u.globe.scope}}

    l := ul{u, u.globe.top}
    l.parseArgs(os.Args[1:])

    if u.profile {
		f, e := os.Create(filepath.Join(baseWorkDir, "load.cpu.auto.prof"))
        if e != nil {
            erro(ctx, "%v", e, trace{})
			return
        } else if e := pprof.StartCPUProfile(f); e != nil {
			erro(ctx, "could not start CPU profile: %v", e, trace{})
			return
		}
        defer func() {
			pprof.StopCPUProfile()
			f.Close()

			f, e = os.Create(filepath.Join(baseWorkDir, "load.mem.auto.prof"))
            if e != nil {
                erro(ctx, "%v", e, trace{})
            }

			runtime.GC() // update memory statistics

			e = pprof.WriteHeapProfile(f)
			if f.Close(); e != nil {
				erro(ctx, "could not start CPU profile: %v", e, trace{})
			}
        } ()
    }

    if u.verboseImport { prompt(ctx, "┌→%s\n", u.workdir) }

    defer func(t time.Time) {
        if d := time.Now().Sub(t); u.verboseImport {
            var name Symbol
            if p := _project(u.globe.top); p != nil { name = p.name }
            prompt(ctx, "└·%s … (%s)\n", name, d)
        } else if false && u.slow < d {
            debug(pc(ctx, u.workdir), "slow loading (%v)!!\n", d)
        }
    } (time.Now())

    spec, _ := filepath.Rel(baseWorkDir, u.workdir)
    l.directory(l.loader, spec, u.workdir, nil)

    if l.globe.main == nil {
        erro(ctx, "nothing loaded", trace{})
    }
    return
}

func (u *universe) run() (result []Value) {
	if u.noRun { return }

	var main = u.globe.main
	if main == nil {
		erro(u, "no targets to update `%v`", u.globe.goals, trace{})
	}

	var ctx Context = closure_with(u, main.scope)
	if u.verbose { debug(ctx, "goal: %v", main) }

	removeTempDirs(ctx)

	if u.profile || u.cpuProfile != "" {
		var name = u.cpuProfile
		if name == "" { name = "cpu.profile" }
		defer cpu_profile(ctx, name, true)()
	}
	if u.profile || u.memProfile != "" {
		var name = u.memProfile
		if name == "" { name = "mem.profile" }
		defer heap_profile(ctx, name)()
	}

	var done bool
	for _, flag := range u.globe.flags {
		if u.verboseExecFlags { info(ctx, "%v", flag) }

		var s = __string(ctx, flag.Value)
		var args, _ = u.globe.args[flag]
		var entries, _ = u.globe.flagEntries[s]
		for _, entry := range entries {
			if u.verboseExecFlags { info(ctx, "%v", entry); flush(ctx) }

			res := entry.execute(ctx, args...)
			result = append(result, res...)
			done = true
		}
	}
	if done { return }

	var updated int
	var goals []Value
	var collect func(proj *project, vals []Value) bool
	collect = func(proj *project, vals []Value) bool {
		if len(vals) == 0 {
			if entry := proj.main; entry != nil {
				goals = append(goals, entry)
			}
			return true
		}
		for _, goal := range vals {
			switch t := goal.(type) {
			case *null, *none: // just ignore
			case *word:
				// NOTE: t.s is now a Symbol. If _entries expects a string, use t.s.String()
				// If _entries takes any/Value, t.s can be passed directly.
				if entries := proj._entries(ctx, t.s.String(), true); entries == nil {
					erro(ctx, "no such entry `%s`", t.s)
					return false
				} else {
					for _, entry := range entries { goals = append(goals, entry) }
				}
			case *delegate:
				var s = __string(ctx, t)
				if entries := proj._entries(ctx, s, true); entries == nil {
					erro(ctx, "no such entry `%s` (via `%v`)", s, t)
					return false
				} else {
					for _, entry := range entries { goals = append(goals, entry) }
				}
			case flag:
				var s = __string(ctx, t)
				if entries := proj._entries(ctx, s, true); entries == nil {
					erro(ctx, "no such entry `%s` (via `%v`)", s, t)
					return false
				} else {
					for _, entry := range entries { goals = append(goals, entry) }
				}
			case *argumented:
				{
					var (
						sStr = __string(ctx, t.Value)
						sSym = intern(sStr) // CRITICAL FIX: Cross into integer domain!
						args = merge(t.args...)
						found int
					)
					for _, p := range u.globe.loaded {
						// Fast O(1) integer matching!
						if p.name == sSym || p.spec == sSym {
							found += 1
							if !collect(p, args) { return false }
						}
					}
					if found == 0 {
						erro(ctx, `"%s" not loaded: %v`, sStr, args)
						return false
					}
				}
			default:
				erro(ctx, "%v: unknown target: %v (%s)", proj, goal, typeof(goal))
				return false
			}
		}
		return true
	}

	if collect(main, merge(u.globe.goals.value)) {
		if len(goals) == 0 {
			if entry := main.main; entry != nil {
				goals = append(goals, entry)
			}
		}
		for _, goal := range goals {
			args, _ := u.globe.args[goal]
			result = append(result, updateGoal(ctx, goal, args)...)
			updated += 1
		}
	}
	return
}

// A globe represents a global execution context.
type globe struct {
    *scope

    top    *loader
    main   *project
    loaded map[string]*project // loaded projects

    args map[Value][]Value
    flagEntries map[string][]entry
    flags []flag
    pairs []*pair

    os    *def
    goals *def
    mode  *def
}

func (g *globe) SetScopeOuter(scope *scope) { scope.outer = g.scope }
func (g *globe) AddFlagEntry(name string, entry entry) {
    flags, _ := g.flagEntries[name]
    flags     = append(flags, entry)
    g.flagEntries[name] = flags
    return
}

var (
    errorIllImport = errors.New("illegal import spec")
    errorIllJson   = errors.New("illegal json format")
    errorIllName   = errors.New("illegal name")
    errorIllXml    = errors.New("illegal xml format")
    errorNilExec   = errors.New("execute nil program")
    errorNoEntry   = errors.New("no matched rule")
    errorUpdated   = errors.New("target updated")
)

type (
    failureAssert      string
    failureUnreachable string
    failureTargetNotFound struct { project *project; target string }
    failurePathNotFound   struct { project *project; path *path }
    failureFileNotFound   struct { project *project; file *file }
    failure     struct { Context; reason string }
    termination struct { position Position }
)

func _failure(ctx Context, a ...any) failure {
    var s string
    if y := false; 0 < len(a) {
        if s, y = a[0].(string); y {
            if 1 < len(a) {
                s = fmt.Sprintf(s, a[1:]...)
            }
        }
    }
    return failure{ctx, s}
}

func (f *failure) Error() (s string) {
    s = "failed"
    if f.Context != nil { s += " : "+ts(f.Context) }
    if f.reason != "" { s += " : "+f.reason }
    return
}

func (s failureAssert) Error() string { return string(s) }
func (s failureUnreachable) Error() string { return string(s) }

func (e failureTargetNotFound) Error() string {
    return fmt.Sprintf("%s: %v: target not found", e.project.name, e.target)
}

func (e failurePathNotFound) Error() string {
    return fmt.Sprintf("%s: %v: path not found", e.project.name, e.path)
}

func (e failureFileNotFound) Error() string {
    if s, t := e.file.fullname(), e.file.filestub.name.String(); t == s { // e.project.name
        return fmt.Sprintf(`"%v" not found`, t)
    } else {
        return fmt.Sprintf(`"%v" not found (at %s)`, t, s) //trimPromptString(s)
    }
}

func assert(cond bool, s string, a ...any) {
    if !cond { panic(failureAssert(fmt.Sprintf(s, a...))) }
}

func unreachable(a ...any) {
    panic(failureUnreachable(fmt.Sprint(a...)))
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

var (
	l_t        = time.Now()
	l_config   = &ltracing{ tm: l_t }
	l_exec     = &ltracing{ tm: l_t }
	l_launch   = &ltracing{ tm: l_t }
	l_load     = &ltracing{ tm: l_t }
	l_parse    = &ltracing{ tm: l_t } // UNUSED
	l_traverse = &ltracing{ tm: l_t }
)

type l_tracer interface {
	elapsed() time.Duration
	tracef(string,...any)
	trace(...any)
	level(int)
}

func l_trace(t l_tracer, s string) l_tracer {
	t.trace(s, "(")
	t.level(+1)
	t.tracef("%v", t.elapsed())
	return t
}

func l_tracef(t l_tracer, f string, a ...any) l_tracer {
	t.trace(fmt.Sprintf(f, a...), "(")
	t.level(+1)
	t.tracef("%v", t.elapsed())
	return t
}

// Usage:
//   defer un(trace(p, "..."))
//   defer un(tracef(p, "..."))
//   defer un(tr(p, "..."))
//   defer un(tt(p, t, "..."))
func un(t l_tracer) {
	t.tracef("%v", t.elapsed())
	t.level(-1)
	t.trace(")")
}

func tr(t l_tracer, i Value) l_tracer {
	t.tracef("%s (", ts(i))
	t.level(+1)
	t.tracef("%v", t.elapsed())
    return t
}

func tt(t l_tracer, ctx Context, i Value) l_tracer {
    // Note that t.args and t.arguments are different, they're
    // target execution args and argumented-prerequisite args.
    var a string = ts(_entry(ctx).destiny())
    if false { a += " " + ts(ctx) }
    t.trace(a, ":", ts(i), "(")
    t.level(+1)
	t.tracef("%v", t.elapsed())
    return t
}

type ltracing struct {
	// Tracing/debugging
	all bool
	enabled bool // (mode&Trace != 0)
	indent int  // indentation used for tracing output
	tm time.Time
}

// Printing fields (splitted by \t).
//var lenPrintField = lenPrintTab * 1

const (
	// Tab size helps formatting fields.
	lenPrintTab = 8

	dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
	ndots = len(dots)
)

func fprintIndentDots(w io.Writer, indent int, a ...any) {
	i := 2 * indent
	for i > ndots {
		fmt.Fprint(w, dots)
		i -= ndots
	}
	// i <= n
	fmt.Fprint(w, dots[0:i])
	if false && len(a) > 0 {
		fmt.Fprintln(w, a...)
	} else {
		var fieldLen = 0
		for i, v := range a {
			if r, ok := v.(rune); ok && r == '\t' {
				const sps = "                         "
				if m := fieldLen % lenPrintTab; m > 0 {
					if m > len(sps) { m = len(sps)-1 }
					fmt.Fprint(w, sps[:m])
				}
				fieldLen = 0
			} else if s := fmt.Sprint(v); s != "" {
				if i > 0 {
					fmt.Fprint(w, " ", s)
					fieldLen += len(s) + 1
				} else {
					fmt.Fprint(w, s)
					fieldLen += len(s)
				}
			}
		}
		fmt.Fprintln(w)
	}
}

func printIndentDots(indent int, a ...any) {
	fprintIndentDots(stderr, indent, a...)
}

func (p *ltracing) traceAt(pos Position, a ...any) {
	fmt.Fprintf(stderr, "%7d:%3d: ", pos.Line, pos.Column)
	printIndentDots(p.indent, a...)
}

func (p *ltracing) trace(a ...any) {
	printIndentDots(p.indent, a...)
}

func (p *ltracing) tracef(s string, a ...any) {
	printIndentDots(p.indent, fmt.Sprintf(s, a...))
}

func (p *ltracing) level(n int) {
	p.indent += n
}

func (p *ltracing) elapsed() time.Duration {
	if false && p.tm.IsZero() { p.tm = time.Now() }
	return time.Now().Sub(p.tm)
}

type interpreter interface {
    evaluate(Context, ...Value) Value
}

// CRITICAL FIX: Upgraded to map[Symbol]interpreter
var dialects = map[Symbol]interpreter{
	symEmpty:   &eval{ o:defExpand0 }, // Use your constant for ""
	symEval:    &eval{ o:defExpand1 },
	symValue:   &eval{ accumulation:true },
	symShell:   &executor{ cmd:"bash",   opt:"-c", contained:false },
	symPython:  &executor{ cmd:"python", opt:"-c", contained:false },
	symPerl:    &executor{ cmd:"perl",   opt:"-e", contained:false },
	symDock:    &executor{ cmd:"sh",     opt:"-c", contained:true },
	symPlain:   &plainint{},
	symJson:    &json{},
	symXml:     &xml{ whitespace:false },
	symYaml:    &yaml{ whitespace:false },
}

func intername(i interpreter) (s Symbol) {
    for k, d := range dialects {
        if d == i { s = k; break }
    }
    return
}

// https://unicode-table.com/en/sets/arrows-symbols/
// https://en.wikipedia.org/wiki/Mathematical_operators_and_symbols_in_Unicode
// 🕐🕑🕒🕓🕔🕕🕖🕗🕘🕙🕚🕛🕜🕝🕞🕟🕠🕡🕢🕣🕤🕥🕦🕧
// ┌────────────────────────────────┐
// ├────────────────────────────────┼ ⇔ ⇒
// ├────┬─────────────────┬─────────┤   ↑    ⇡
// ├┬───┼─────────────────┼─────────┘  ←·→  ⇠·⇢
// │├┬──┴─     arrows     └──┬──┐       ↓    ⇣
// ││└─────────────────┐     │  ├─
// │└─────┬──   ────┬──┴──┬──┘  │      ⇤…⇥
// └──    ┘        ─┘           └─

type token int

const (
	// Special tokens.
	ILLEGAL token = iota // 0 - illegal token represents no valid token

	EOF      // 1 - end of file
	SPACE    // [ ]
	COMMENT  // #
	HASH     // # (same char as COMMENT, but different meaning)

	// _literal_beg
	// Identifiers and basic type literals (these tokens stand for classes of literals)
	WORD
	BINARY   // 0b010101, 0B0111001
	OCTAL    // 0600, 0567
	INTEGER  // 12345
	HEXADECIMAL // 0x1234567890ABCDEF
	FLOATING    // 123.45
	DATETIME // 1979-05-27T07:32:00.999999-07:00 (internet date/time format - RFC3339)
	DATE     // 1979-05-27 (internet date format - RFC3339)
	TIME     // 07:32:00.999999 (internet time format - RFC3339)
	URL      // 'mailto:name@example.com' (uniform resource identifier - RFC3986)
	RAW      // raw strings
	ESCAPE   // \", \\n, etc. (see value.EscapeChar)
	STRING   // 'abc'
	STRVAL   // {abc}
	STRCOMP  // "abc $(foo) 123"
	// _literal_end

	COMPOSED // the ending quote of a strcomp literal
	RECIPE   // tab to indicate a command recipe
	LINEND   // significannot line break (LF or CRLF)

	PROOT    // the root of a path, aka the virtual segment "" before the first '/' in a path
	PTAIL    // the tail of a path, aka the virtual segment "" after the last '/' in a path

	// _operator_beg
	LANGLE    // <
	LBRACE    // {    left curly
	LBRACK    // [
	LPAREN    // (
	Lchevron    // ⟨ ⟪⟫ ｟｠ 〝〞
	Ltop_corner // ⌜
	Lbot_corner // ⌞
	Lsing_guil  // ‹
	Lguillemet  // «
	Rguillemet  // »
	Rsing_guil  // ›
	Rbot_corner // ⌟
	Rtop_corner // ⌝
	Rchevron    // ⟩
	RPAREN    // )
	RBRACK    // ]
	RBRACE    // }    right curly
	RANGLE    // >

	CARET     // ^ ˆ‸
	COMMA     // ,
	DOT       // .    period
	DOTDOT    // ..
	TILDE     // ~

	SELECT_PROP  // -> 'foo→xxx' (different from ' → ')
	SELECT_PROG1 // => 'foo⇒xxx' ('foo↦xxx' 'foo↣xxx' 'foo⇥xxx')
	SELECT_PROG2 // ~> 'foo⇢xxx' ('foo↦xxx' 'foo↣xxx' 'foo⇥xxx')
	// ⤌ ⤍	⤎ ⤏	⤐	⤑

	SEMICOLON // ;

	EXC       // !    exclamation
	QUE       // ?

	AT        // @
	SAST      // *    Single Asterisk
	DAST      // **   Double Asterisk
	ASTQ      // *?   Asterisk Que
	UNDERLINE // _    Underscore

	CLOSURE   // &
	DELEGATE  // $

	MINUS // unary -
	PLUS  // unary +
	PCON  // path concatenation '/'
	PERC  // percent sign '%'(REM)

	// _ruledelim_beg
	BAR       // |
	COLON     // :
	DOLON     // ::
	SOLON     // ;:
	// _ruledelim_end

	// ⩵ ⩶
	// _assign_beg
	ASSIGN     //   =       define a new symbol (don't override, neither !=)
	ASSIGN_USH //   =+      unshift (add to front, versus: 'shift': remove from front; 'pop': remove from end)
	ASSIGN_ADD //  +=       append (add to end, 'push', versus: 'prepend', 'unshift': add to front)
	ASSIGN_QUE //  ?=       set if absent (defined, including empty)
	ASSIGN_EXC //  !=       execute a shell script and set a variable to its output (.SHELLSTATUS)
	// TODO: more assigns like !?=  !:=  !+=
	ASSIGN_CO1 //  := ≔     delegate-expanded (also override)
	ASSIGN_CO2 // ::= ⩴    all-expanded (POSIX standard)
	ASSIGN_CO3 // ;:=       all and unexpanded-force
	ASSIGN_SC1 //  ;=       unexpanded-force
	ASSIGN_POP //  -=       pop (remove from end, versus: 'shift': remove from front)
	ASSIGN_SAD // -+=       pop-append assign
	ASSIGN_SUS //  -=+      pop-unshift assign
	// _assign_end
	// _operator_end

	// _keyword_beg
	PROJECT    // project a
	CONFIGURE  // configure [...] TODO: use a different keyword
	USE        // use b
	ASSERT     // assert clause
	APPEND     // append values
	LOCAL      // declare local def names
	EVAL       // evaluate a builtin immediately
	EXPORT     // export ...
	INCLUDE    // include a.smart
	INSTANCE   // instance
	FILES      // files
	TEMPLATE   // template
	AND        // and
	OR         // or
	FOR        // for
	FOREACH    // foreach
	DONE       // done
	DEF        // def
	END        // end

	// _constant_beg
	UNDEF   // `undef`
	NULL    // `null`
	NONE    // `none`
	BARE    // `bare`  // TODO
	PATH    // `path`  // TODO
	GLOB    // `glob`  // TODO
	REGEX   // `regex` // TODO
	FILE    // `file`  // TODO
	BIN     // `bin`
	OCT     // `oct`
	INT     // `int`
	HEX     // `hex`
	FLOAT   // `float`
	ANSWER  // `answer`
	BOOL    // `bool`
	BOOLEAN // `boolean`
	TRUE    // boolean `true`
	FALSE   // boolean `false`
	YES     // answer `yes`
	NO      // answer `no`
	ON      // option `on`
	OFF     // option `off`
	// _constant_end
	// _keyword_end = _constant_end

	DASH = MINUS
)

var tokens = [...]string{
	ILLEGAL: "ILLEGAL",

	EOF:     "EOF",
	SPACE:   "SPACE",
	COMMENT: "COMMENT",
	HASH:    "HASH",

	WORD:     "WORD",
	BINARY:   "BINARY",
	OCTAL:    "OCTAL",
	INTEGER:  "INTEGER",
	HEXADECIMAL: "HEXADECIMAL",
	FLOATING:    "FLOATING",
	DATETIME: "DATETIME",
	DATE:     "DATE",
	TIME:     "TIME",
	URL:      "URL",
	RAW:      "RAW",
	STRING:   "STRING",
	STRVAL:   "STRVAL",
	STRCOMP:  "STRCOMP",

	COMPOSED: "COMPOSED",
	RECIPE:   "RECIPE",
	ESCAPE:   "\\",
	LINEND:   "\\n", //"LINEND",
	PROOT:    "", // the "" before the first '/' in a path
	PTAIL:    "", // the "" after the last '/' in a path

	LANGLE: "<",
	LBRACE: "{",
	LBRACK: "[",
	LPAREN: "(",
	Lchevron: "⟨",
	Ltop_corner: "⌜",
	Lbot_corner: "⌞",
	Lsing_guil: "‹",
	Lguillemet: "«",
	Rguillemet: "»",
	Rsing_guil: "›",
	Rbot_corner: "⌟",
	Rtop_corner: "⌝",
	Rchevron: "⟩",
	RPAREN: ")",
	RBRACK: "]",
	RBRACE: "}",
	RANGLE: ">",

	CARET:  "^",
	COMMA:  ",",
	DOT:    ".",
	DOTDOT: "..",
	TILDE:  "~",

	SELECT_PROP:  "→", // foo->bar
	SELECT_PROG1: "⇒", // foo=>bar foo⇒bar
	SELECT_PROG2: "⇢", // foo~>bar foo⇢bar

	SEMICOLON: ";",

	EXC:       "!",
	QUE:       "?",

	BAR:       "|",
	COLON:     ":",
	DOLON:     "::",
	SOLON:     ";:",

	AT:        "@",
	SAST:      "*",
	DAST:      "**",
	ASTQ:      "*?",
	UNDERLINE: "_",

	CLOSURE:   "&",
	DELEGATE:  "$",

	ASSIGN:     "=",
	ASSIGN_USH: "=+",
	ASSIGN_ADD: "+=",
	ASSIGN_QUE: "?=",
	ASSIGN_EXC: "!=",
	ASSIGN_CO1: ":=",
	ASSIGN_CO2: "::=",
	ASSIGN_CO3: ";:=",

	ASSIGN_SC1: ";=",
	ASSIGN_POP: "-=",
	ASSIGN_SAD: "-+=",
	ASSIGN_SUS: "-=+",

	MINUS: "-", // DASH
	PLUS:  "+",
	PCON:  "/",
	PERC:  "%",

	PROJECT:   "project",
	CONFIGURE: "configure",
	USE:       "use",
	ASSERT:    "assert",
	APPEND:    "append",
	LOCAL:     "local",
	EVAL:      "eval",
	EXPORT:    "export",
	INCLUDE:   "include",
	INSTANCE:  "instance",
	FILES:     "files",
	TEMPLATE:  "template",
	AND:       "and",
	OR:        "or",
	FOR:       "for",
	FOREACH:   "foreach",
	DONE:      "done",
	DEF:       "def",
	END:       "end",

	UNDEF:  "undef",
	NULL:   "null",
	NONE:   "none",
	BARE:   "bare",
	PATH:   "path",
	GLOB:   "glob",
	REGEX:  "regex",
	FILE:   "file",
	BIN:    "bin",
	OCT:    "oct",
	INT:    "int",
	HEX:    "hex",
	FLOAT:  "float",
	ANSWER: "answer",
	BOOL:   "bool",
	BOOLEAN:"boolean",
	TRUE:   "true",
	FALSE:  "false",
	YES:    "yes",
	NO:     "no",
	ON:     "on",
	OFF:    "off",
}

func (tok token) String() (s string) {
	if 0 <= tok && tok < token(len(tokens)) { s = tokens[tok] }
	if s == "" {
		switch tok {
		case PROOT, PTAIL: return
		default:
			return "token(" + strconv.Itoa(int(tok)) + ")"
		}
	}
	return
}

var keywords map[Symbol]token

// lookup_keyword maps an identifier to its keyword token or IDENT (if not a keyword).
func lookup_keyword(s Symbol) token {
	if t, y := keywords[s]; y { return t }
	return WORD
}

func (tok token) is_literal() bool          { return WORD <= tok && tok <= STRCOMP }
func (tok token) is_operator() bool         { return LANGLE <= tok && tok < PROJECT }
func (tok token) is_keyword() bool          { return PROJECT <= tok && tok <= OFF }
func (tok token) is_constant() bool         { return UNDEF <= tok && tok <= OFF }
func (tok token) is_closure() bool          { return CLOSURE == tok }
func (tok token) is_closure_delegate() bool { return CLOSURE == tok || tok == DELEGATE }
func (tok token) is_delegate() bool         { return DELEGATE == tok }
func (tok token) is_assign() bool           { return ASSIGN <= tok && tok <= ASSIGN_SUS }
func (tok token) is_rule_delim() bool       { return BAR <= tok && tok <= SOLON }
func (tok token) is_list_delim() bool {
	switch tok {
	case RPAREN, RBRACK, RBRACE, Rbot_corner, Rtop_corner, Rsing_guil, Rguillemet, Rchevron, SEMICOLON, COMMA, LINEND, EOF:
		return true
	}
	return tok.is_rule_delim()
}

/*
  Struct Position:
	Filename string  -- filename, if any
	Offset   int     -- offset, starting at 0
	Line     int     -- line number, starting at 1
	Column   int     -- column number, starting at 1 (byte count)
*/
type Position struct { gt.Position }
func (p *Position) valid() bool { return p.Filename != "" && p.Line > 0 }
func (p *Position) same(o *Position) bool {
	return p == o ||
		p.Filename == o.Filename && p.Line == o.Line &&
		p.Column == o.Column && p.Offset == o.Offset
}
func (p *Position) sameLoc(o *Position) bool {
	return p == o ||
		p.Filename == o.Filename && p.Line == o.Line &&
		p.Column == o.Column
}
func (p *Position) sameLine(o *Position) bool {
	return p == o || (p.Filename == o.Filename && p.Line == o.Line)
}

func atoi(a any) (res int) {
	switch t := a.(type) {
	case string: res, _ = strconv.Atoi(t)
	case []byte: res, _ = strconv.Atoi(string(t))
	}
	return
}

const NoPos Pos = Pos(gt.NoPos)

type Pos gt.Pos
func (p Pos) IsValid() bool { return gt.Pos(p).IsValid() }

// mbInfo stores the byte offset of a multi-byte character and how many "extra"
// bytes it consumes compared to a standard 1-byte ASCII character.
type mbInfo struct {
	offset int
	extra  int
}

type tokfile struct {
	*gt.File
	sync.RWMutex // Protects the slice during concurrent parsing/tracing
	mb []mbInfo  // The sparse multibyte index
}

func (f *tokfile) Offset(p Pos) int { return f.File.Offset(gt.Pos(p)) }
func (f *tokfile) Line(p Pos) int { return f.File.Line(gt.Pos(p)) }
func (f *tokfile) Pos(offset int) Pos { return Pos(f.File.Pos(offset)) }

// CRITICAL FIX: Mathematical Correction for Rune Columns
func (f *tokfile) Position(p Pos) Position {
	gpos := gt.Pos(p)
	pos := f.File.PositionFor(gpos, true)

	// Translate Byte Column to Rune Column using the sparse index
	if pos.Column > 1 {
		f.RLock() // Protect slice read
		if len(f.mb) > 0 {
			lineStartPos := f.File.LineStart(pos.Line)
			lineStartOffset := f.File.Offset(lineStartPos)
			targetOffset := f.File.Offset(gpos)

			// Binary search: Find the first multibyte char on or after this line
			startIdx := sort.Search(len(f.mb), func(i int) bool {
				return f.mb[i].offset >= lineStartOffset
			})

			// Accumulate the extra bytes occurring before our target token
			extraBytes := 0
			for i := startIdx; i < len(f.mb); i++ {
				if f.mb[i].offset >= targetOffset { break }
				extraBytes += f.mb[i].extra
			}

			// Mathematically collapse the byte column into a precise rune column!
			pos.Column -= extraBytes
		}
		f.RUnlock()
	}

	return Position{pos}
}

// AddSpan registers a multi-byte character during scanningt.
func (f *tokfile) AddSpan(offset, extraBytes int) {
	t := mbInfo{ offset:offset, extra:extraBytes }
	f.Lock()
	f.mb = append(f.mb, t)
	f.Unlock()
}

type fileset struct {
	*gt.FileSet
	files map[*gt.File]*tokfile
	sync.RWMutex
}

func new_fileset() *fileset { return &fileset{
	FileSet: gt.NewFileSet(),
	files: make(map[*gt.File]*tokfile),
}}

func (s *fileset) AddFile(filename string, base, size int) *tokfile {
	gf := s.FileSet.AddFile(filename, base, size)
	tf := &tokfile{File: gf}

	s.Lock()
	s.files[gf] = tf
	s.Unlock()

	return tf
}

func (s *fileset) Iterate(f func(*tokfile) bool) {
	s.FileSet.Iterate(func(a *gt.File) bool {
		s.RLock()
		tf, ok := s.files[a]
		s.RUnlock()
		if ok { return f(tf) }
		return f(&tokfile{File: a})
	})
}

func (s *fileset) Position(p Pos) Position {
	gpos := gt.Pos(p)
	if gf := s.FileSet.File(gpos); gf != nil {
		s.RLock()
		tf, ok := s.files[gf]
		s.RUnlock()
		if ok {
			return tf.Position(p)
		}
	}
	return Position{s.FileSet.Position(gpos)}
}

const (
	isStrcompLine scanbits = 1<<iota // 0000000000000001
	isStrcompString  // 0000000000000010 "...."
	isCall           // 0000000000000100 $.....
	isCallParen      // 0000000000001000 $(...)              8
	isCallBrace      // 0000000000010000 ${...}             16
	isCallColonL     // 0000000000100000 $:....             32
	isCallColonR     // 0000000001000000 $:...:             64
	isGroup          // 0000000010000000 (...)             128
	isBrace          // 0000000100000000 {...}             256
	isBraceRaw       // 0000001000000000                   512
	isBracedPlain    // 0000010000000000                  1024
	isRecipes        // 0000100000000000                  2048
	isRecipeTab      // 0001000000000000 \t               4096
	isHashValid      // 0010000000000000 scan '#' as HASH token (commentsOff)
	isMaximumBit     // 1000000000000000                  8192
)

const bom = 0xFEFF // byte order mark, only permitted as very first character

func IsLetter(r rune) bool {
	return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || r == '_' || r >= 0x80 && unicode.IsLetter(r)
}

func IsDigit(r rune) bool {
	return unicode.IsDigit(r) //('0' <= r && r <= '9') || (r >= 0x80 && unicode.IsDigit(r))
}

func IsDigits(s string) bool {
    return strings.IndexFunc(s, func(r rune) bool { return !IsDigit(r) }) < 0
}

// punctuation used as non-terminator
func IsUntermPunct(r rune) bool {
	// Most chars accepted in URL (RFC3986)
	return r == '@' || r == '+' /* || r == '-' || r == '.' || r == '/' */;
}

func IsDatetimeTerminator(r rune) bool {
	return  r == ' ' || r == '\t' || r == '\n' || r == '\r' ||
		r == '(' || r == ')' || r == '{' || r == '}' ||
		r == '$' || r == '#' || r == '\\'
}

func IsIdentifier(r rune) bool {
	return IsLetter(r) || IsDigit(r) || IsUntermPunct(r) //|| r == '\\'
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9': return int(ch - '0')
	case 'a' <= ch && ch <= 'f': return int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F': return int(ch - 'A' + 10)
	}
	return 16 // larger than any legal digit val
}

// A mode value is a set of flags (or 0).
// They control scanner behavior.
type scanmode uint
type scanbits uint
func (bits scanbits) is(t scanbits)     bool { return bits&t != 0 }
func (bits scanbits) isCall()           bool { return bits&isCall != 0 }
func (bits scanbits) isCallZero()       bool { return bits&isCall != 0 && bits&(isCallParen|isCallBrace|isCallColonL) == 0 }
func (bits scanbits) isCallParen()      bool { return bits&isCallParen != 0 }
func (bits scanbits) isCallBrace()      bool { return bits&isCallBrace != 0 }
func (bits scanbits) isCallColonL()     bool { return bits&isCallColonL != 0 }
func (bits scanbits) isCallColonR()     bool { return bits&isCallColonR != 0 }
func (bits scanbits) isCommentsOff()    bool { return bits&isHashValid != 0 }
func (bits scanbits) isBrace()          bool { return bits&isBrace != 0 }
func (bits scanbits) isBraceRaw()       bool { return bits&isBraceRaw != 0 }
func (bits scanbits) isBracedPlain()    bool { return bits&isBracedPlain != 0 }
func (bits scanbits) isGroup()          bool { return bits&isGroup != 0 }
func (bits scanbits) isStrcompLine()    bool { return bits&isStrcompLine != 0 }
func (bits scanbits) isStrcompString()  bool { return bits&isStrcompString != 0 }
func (bits scanbits) canRecipe()        bool { return bits&(isRecipeTab|isRecipes) != 0 }

type scanstate struct {
	ch         rune  // current character
	offset     int   // character offset
	offsetRead int   // reading offset (position after current character)
	offsetLine int   // current line offset
	bitss []scanbits // scan bits stack
	bits    scanbits // scan bits

	// --- NEW: Token Value Payloads ---
	pos Pos
	tok token
	sym Symbol // Fast integer channel for identifiers, keywords, paths
	lit string // Slow string channel for giant string literals / raw text
}

func (s *scanstate) ch_bytes() int { return s.offsetRead - s.offset }
func (s *scanstate) String() string {
	var t string
	switch s.ch {
	case '\n': t = "\\n"
	case '}': t = "\\}"
	default: t = string(s.ch)
	}
	return fmt.Sprintf("{=scanstate '%s' {%v %v %v} %016b %016b {tok=%v pos=%v sym=%v lit=%s}}",
		t, s.offsetLine, s.offset, s.offsetRead, s.bitss, s.bits, s.tok, s.pos, s.sym, s.lit)
}

func (s *scanstate) push(bits scanbits) (prev scanbits) {
	if prev = s.bits; prev != 0 {
		s.bitss = append(s.bitss, prev) // &^ isLineFeed
	}
	s.bits = bits
	return
}
func (s *scanstate) pop(bits scanbits) (prev scanbits) {
	if prev = s.bits ; bits == 0 || (s.bits&bits != 0) {
		if i := len(s.bitss); 0 == i {
			s.bits = 0
		} else {
			s.bits = s.bitss[i-1] //&^ isLineFeed
			s.bitss = s.bitss[0:i-1]
		}
	}
	return
}

func (s *scanstate) setBits(bits scanbits) (prev scanbits) {
	prev = s.bits
	s.bits = bits
	return
}

func (s *scanstate) addBits(bits scanbits) (prev scanbits) {
	prev = s.bits
	s.bits |= bits
	return
}

func (s *scanstate) remBits(bits scanbits) (prev scanbits) {
	prev = s.bits
	s.bits &^= bits
	return
}

func (s *scanstate) commentsOff() scanbits { return s.addBits(isHashValid) }
func (s *scanstate) recipes(v bool) {
	var bits = s.bits
	if v { bits |= isRecipes } else { bits &^= isRecipes }
	s.bits = bits
}

func (s *scanstate) canRecipe() (res bool) {
	if t := s.bits; (s.offsetLine == s.offset-1) && t.canRecipe() {
		res = !t.is(isCallParen|isCallBrace|isCallColonL|isCallColonR|isGroup)
	}
	return
}

func (s *scanstate) bit(bits scanbits) (res bool) {
	if res = s.bits&bits != 0; !res {
		for i := len(s.bitss)-1; 0 <= i; i -= 1 {
			if res = s.bitss[i]&bits != 0; res { break }
		}
	}
	return
}

// A scanner holds the scanner's internal state while processing
// a given text.  It can be allocated as part of another data
// structure but must be initialized via Init before use.
//
// (See go.token)
type scanner struct { // immutable state
	file *tokfile     // source file handle
	dir  string       // directory portion of file.Name()
	src  []byte       // source
	mode scanmode     // scanning mode
	scanstate
}

// Read the next Unicode char into s.ch, s.ch < 0 means end-of-file.
func (s *scanner) next(ctx Context) {
	var newline = s.ch == '\n'

	if s.offsetRead < len(s.src) {
		if s.offset = s.offsetRead; s.ch == '\n' {
			s.offsetLine = s.offset
			s.file.AddLine(s.offset)
		}
		var w int
		s.ch, w = s.pick(ctx, s.offsetRead)
		s.offsetRead += w
	} else {
		if s.offset = len(s.src); s.ch == '\n' {
			s.offsetLine = s.offset
			s.file.AddLine(s.offset)
		}
		s.ch = -1 // eof
	}

	if newline && s.ch == '\t' {
		s.bits |= isRecipeTab
		// s.bits &^= isLineFeed
	} else {
		// s.bits &^= isLineFeed | isRecipeTab
		s.bits &^= isRecipeTab
	}
}

func (s *scanner) pickNext(ctx Context) (ch rune, w int) {
	if n := s.offsetRead + 1; n < len(s.src) { ch, w = s.pick(ctx, n) }
	return
}

func (s *scanner) pick(ctx Context, offset int) (ch rune, w int) {
	switch ch, w = rune(s.src[offset]), 1; {
	case ch == 0:
		erro(pc(ctx,s.offsetPos(offset)), "illegal character NUL")
	case ch >= 0x80: // Non ASCII
		if ch, w = utf8.DecodeRune(s.src[offset:]); ch == utf8.RuneError && w == 1 {
			erro(pc(ctx,s.offsetPos(offset)), "illegal UTF-8 encoding")
		} else if ch == bom && offset > 0 {
			erro(pc(ctx,s.offsetPos(offset)), "illegal byte order mark")
		} else if w > 1 {
			// CRITICAL FIX: Register the multibyte span instantly during parsing!
			// We pass the byte offset and the "extra" bytes (width - 1).
			s.file.AddSpan(offset, w - 1)
		}
	}
	return
}

func (s *scanner) init(ctx Context, file *tokfile, src []byte, mode scanmode) {
	// Explicitly initialize all fields since a scanner may be reused.
	if sz, l := file.Size(), len(src); sz != l {
		debug(ctx, "file size (%d) does not match src len (%d)", sz, l, ctx)
	}

	s.file = file
	s.dir, _ = filepath.Split(file.Name())
	s.src = src
	s.mode = mode

	s.ch = ' '
	s.offset = 0
	s.offsetRead = 0
	s.offsetLine = 0
	s.bits = 0
	s.bitss = nil
	s.pos = 0
	s.tok = ILLEGAL
	s.sym = symEmpty
	s.lit = ""

	// The BOM at file beginning will be discarded.
	if s.next(ctx); s.ch == bom { s.next(ctx) }
}

func (s *scanner) offsetPos(offs ...int) Position {
	if 0 < len(offs) {
		return s.file.Position(s.file.Pos(offs[0]))
	}
	return s.file.Position(s.file.Pos(s.offset))
}

func (s *scanner) scanComment(ctx Context) (res string) {
	for s.ch == ' '  || s.ch == '\t' { s.next(ctx) } // skip preceding spaces

	var offs = s.offset
	for s.ch != '\n' && s.ch != -1 { s.next(ctx) }

	// We should intern identifiers, words, and numbers. We should not intern
	// comments (they are long and rarely repeated) to avoid bloating the pool.
	return string(s.src[offs:s.offset])
}

func (s *scanner) scanIdentifier(ctx Context) {
	var offs = s.offset
	for IsIdentifier(s.ch) {
		if s.next(ctx); /* s.ch == '-' */false { // Looking for '->'
			var n = s.offset + 1 // No need UTF8 decoding!
			if n < len(s.src) && rune(s.src[n]) == '>' { break }
		}
	}
	s.sym = internBytes(s.src[offs:s.offset])
	return
}

func (s *scanner) scanMantissa(ctx Context, base int) {
	if digitVal(s.ch) < base { // first digit
		s.next(ctx)
		if true {
			for digitVal(s.ch) < base { s.next(ctx) }
		} else {
			// NOTE: disable '_' number separaters as ParseInt not support it and
			//       it's not recoverable from ints back to strings.
			for s.ch == '_' || digitVal(s.ch) < base {
				if s.ch == '_' {
					if s.next(ctx); s.ch == '_' {
						erro(pc(ctx,s), "invalid digit group")
						break
					}
				} else {
					s.next(ctx)
				}
			}
		}
	}
}

func (s *scanner) scanDatetime(ctx Context) (tok token) {
	var (
		ch byte
		hasDate = false
		hasTime = false
		o = s.offset
		l = len(s.src)
	)
	if x := l-o; 8 <= x {
		for i := 0; i < 2; i++ {
			if ch = s.src[o+i]; ch < '0' || '9' < ch {
				goto exit
			}
		}
		if s.src[o+2] == ':' || s.src[o+5] == ':' {
			hasTime = true; goto checkTime
		}
		if s.src[o+4] == '-' || s.src[o+7] == '-' && 10 <= x {
			hasDate = true; goto checkDate
		}
	}

	goto exit

checkDate:
	// 4 digits fullyear (first two digit already checked)
	for i := 2; i < 4; i++ {
		if ch = s.src[o+i]; ch < '0' || '9' < ch {
			goto exit
		}
	}

	// month range is 01-12
	if ch = s.src[o+5]; ch != '0' && ch != '1' {
		erro(pc(ctx,s.offsetPos(o+5)), "bad month"); goto exit
	}
	if ch = s.src[o+6]; ch < '0' || '9' < ch {
		erro(pc(ctx,s.offsetPos(o+6)), "bad month"); goto exit
	}

	// month-day range is 01-28, 01-29, 01-30, 01-31 based on month/year
	if ch = s.src[o+8]; ch < '0' && '3' < ch {
		erro(pc(ctx,s.offsetPos(o+8)), "bad month day"); goto exit
	}
	if ch = s.src[o+9]; ch < '0' || '9' < ch {
		erro(pc(ctx,s.offsetPos(o+9)), "bad month day"); goto exit
	}

	if o += 10; o == l {
		goto success // 1979-05-27
	} else if ch = s.src[o]; IsDatetimeTerminator(rune(ch)) {
		goto success // 1979-05-27
	}

	if ch == 'T' || ch == 't' {
		o += 1 // consume 'T'
		hasTime = true
	} else {
		erro(pc(ctx,s.offsetPos(o)), "bad time"); goto exit
	}

	if l-o < 9 || s.src[o+2] != ':' || s.src[o+5] != ':' {
		erro(pc(ctx,s.offsetPos(o)), "illegal time"); goto exit
	}

checkTime:
	// hour range is 00-23
	if ch = s.src[o+0]; ch < '0' || '2' < ch {
		erro(pc(ctx,s.offsetPos(o+0)), "bad hour"); goto exit
	}
	if ch = s.src[o+1]; ch < '0' || '9' < ch || ('3' < ch && s.src[o] == '2') {
		erro(pc(ctx,s.offsetPos(o+1)), "bad hour"); goto exit
	}

	// minute range is 00-59
	if ch = s.src[o+3]; ch < '0' || '5' < ch {
		erro(pc(ctx,s.offsetPos(o+3)), "bad minute"); goto exit
	}
	if ch = s.src[o+4]; ch < '0' || '9' < ch {
		erro(pc(ctx,s.offsetPos(o+4)), "bad minute"); goto exit
	}

	// second ranges are 00-59 00-58, 00-59, 00-60 based on leap second rules
	if ch = s.src[o+6]; ch < '0' || '5' < ch {
		erro(pc(ctx,s.offsetPos(o+6)), "bad second"); goto exit
	}
	if ch = s.src[o+7]; ch < '0' || '9' < ch {
		erro(pc(ctx,s.offsetPos(o+7)), "bad second"); goto exit
	}

	if ch = s.src[o+8]; IsDatetimeTerminator(rune(ch)) {
		o += 8; goto success // consume 00:00:00
	} else if ch == 'Z' || ch == 'z' {
		o += 9; goto success // consume 00:00:00Z
	} else if ch == '.' {
		for o += 9; o < l; o++ {// consume 00:00:00.
			if ch = s.src[o]; ch == 'Z' || ch == 'z' {
				o += 1; goto success // consume 'Z'
			} else if IsDatetimeTerminator(rune(ch)) {
				goto success
			} else if ch == '+' || ch == '-' {
				o += 1; goto checkNumOffset // consume '+' or '-'
			} else if ch < '0' || '9' < ch {
				erro(pc(ctx,s.offsetPos(o)), "bad secfrac"); goto exit
			}
		}
	} else if ch == '+' || ch == '-' {
		o += 9; goto checkNumOffset // consume 00:00:00+
	} else {
		erro(pc(ctx,s.offsetPos(o)), "bad time"); goto exit
	}

checkNumOffset:
	if ch = s.src[o+2]; ch != ':' {
		erro(pc(ctx,s.offsetPos(o+2)), "bad offset"); goto exit
	}

	// hour range is 00-23
	if ch = s.src[o+0]; ch < '0' || '2' < ch {
		erro(pc(ctx,s.offsetPos(o+0)), "bad hour"); goto exit
	}
	if ch = s.src[o+1]; ch < '0' || '9' < ch || ('3' < ch && s.src[o] == '2') {
		erro(pc(ctx,s.offsetPos(o+1)), "bad hour"); goto exit
	}

	// minute range is 00-59
	if ch = s.src[o+3]; ch < '0' || '5' < ch {
		erro(pc(ctx,s.offsetPos(o+3)), "bad minute"); goto exit
	}
	if ch = s.src[o+4]; ch < '0' || '9' < ch {
		erro(pc(ctx,s.offsetPos(o+4)), "bad minute"); goto exit
	}

	o += 5 // consume 00:00

success:
	for i := s.offset; i < o; i++ { s.next(ctx) }
	switch {
	case hasDate && hasTime: tok = DATETIME
	case hasDate && !hasTime: tok = DATE
	case !hasDate && hasTime: tok = TIME
	default: tok = ILLEGAL
	}

exit:
	return
}

func (s *scanner) scanNumber(ctx Context, seenDecimalPoint bool) token {
	// digitVal(s.ch) < 10
	offs := s.offset
	tok := INTEGER

	if seenDecimalPoint {
		offs--
		tok = FLOATING // CRITICAL FIX: Was FLOAT (which is a keyword!)
		s.scanMantissa(ctx, 10)
		goto exponent
	}

	if t := s.scanDatetime(ctx); t != ILLEGAL {
		tok = t; goto exit
	}

	if s.ch == '0' {
		// int or float
		offs := s.offset
		s.next(ctx)
		if s.ch == 'b' || s.ch == 'B' {
			// binary int
			s.next(ctx)
			s.scanMantissa(ctx, 2)
			tok = BINARY
			if s.offset-offs <= 2 {
				// only scanned "0b" or "0B"
				erro(pc(ctx,offs), "illegal binary number")
			}
		} else if s.ch == 'x' || s.ch == 'X' {
			// hexadecimal int
			s.next(ctx)
			s.scanMantissa(ctx, 16)
			tok = HEXADECIMAL
			if s.offset-offs <= 2 {
				// only scanned "0x" or "0X"
				erro(pc(ctx,offs), "illegal hexadecimal number")
			}
		} else {
			// octal int or float
			seenDecimalDigit := false
			s.scanMantissa(ctx, 8)
			if s.ch == '8' || s.ch == '9' {
				// illegal octal int or float
				seenDecimalDigit = true
				s.scanMantissa(ctx, 10)
			}
			if s.ch == '.' || s.ch == 'e' || s.ch == 'E' || s.ch == 'i' {
				goto fraction
			}
			// octal int
			if seenDecimalDigit {
				erro(pc(ctx,offs), "illegal octal number")
			}
			if s.offset-offs > 1 {
				tok = OCTAL
			} else {
				tok = INTEGER // just '0'
			}
		}
		goto exit
	}

	// decimal int or float
	s.scanMantissa(ctx, 10)

fraction:
	if s.ch == '.' {
		// Safety check 1: Must be followed by a digit. Prevents 1.$2 from breaking.
		if n := s.offset + 1; n < len(s.src) {
			if ch := rune(s.src[n]); !IsDigit(ch) {
				goto exit
			}
		} else {
			goto exit
		}

		// Safety check 2 (The "+2 Hack"): Must have a second character after the dot
		// that is ALSO a digit (or 'e'/'E'). This intentionally prevents 1-digit
		// decimals (like .0 or .4) from being floats, forcing them to parse as
		// structural qualwords for version strings (e.g., 25.4 or 25.4.0).
		if n := s.offset + 2; n < len(s.src) {
			ch := rune(s.src[n])
			if !IsDigit(ch) && ch != 'e' && ch != 'E' {
				goto exit
			}
		} else {
			goto exit
		}

		tok = FLOATING
		s.next(ctx)
		s.scanMantissa(ctx, 10)
	}

exponent:
	if s.ch == 'e' || s.ch == 'E' {
		tok = FLOATING // CRITICAL FIX: Was FLOAT (which is a keyword!)
		s.next(ctx)
		if s.ch == '-' || s.ch == '+' {
			s.next(ctx)
		}
		s.scanMantissa(ctx, 10)
	}

	/*
	if s.ch == 'i' {
		tok = IMAG
		s.next(ctx)
	} */

exit:
	s.lit = string(s.src[offs:s.offset])
	return tok
}

func (s *scanner) scanEscape(ctx Context, quote rune) bool {
	var n int
	var base, max uint32
	var offs = s.offset
	switch s.ch {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '$', quote:
		s.next(ctx)
		return true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		n, base, max = 3, 8, 255
	case 'x':
		s.next(ctx)
		n, base, max = 2, 16, 255
	case 'u':
		s.next(ctx)
		n, base, max = 4, 16, unicode.MaxRune
	case 'U':
		s.next(ctx)
		n, base, max = 8, 16, unicode.MaxRune
	case '\n':
		s.next(ctx)
	default:
		var msg = "unknown escape sequence"
		if s.ch < 0 { msg = "escape sequence not terminated" }
		erro(pc(ctx,offs), msg)
		return false
	}

	var x uint32
	for n > 0 {
		d := uint32(digitVal(s.ch))
		if d >= base {
			var msg = fmt.Sprintf("illegal character %#U in escape sequence", s.ch)
			if s.ch < 0 { msg = "escape sequence not terminated" }
			erro(pc(ctx,offs), msg)
			return false
		}
		x = x*base + d
		s.next(ctx)
		n--
	}

	if x > max || 0xD800 <= x && x < 0xE000 {
		erro(pc(ctx,offs), "escape sequence is invalid Unicode code point")
		return false
	}

	return true
}

func (s *scanner) scanStrlit(ctx Context, ml bool) string {
	// '\'' opening already consumed
	offs := s.offset - 1
	if ml { offs -= 1 }

	for s.offsetRead < len(s.src) {
		ch := s.ch
		if (!ml && ch == '\n') || ch < 0 { // if ch < 0 {
			erro(pc(ctx,offs), "raw string literal not terminated")
			break
		}
		if ch == '\\' { s.next(ctx) } // escapes
		s.next(ctx)
		if ch == '\'' {
			if !ml { break }
			if s.ch == '\'' {
				if s.next(ctx); s.ch == '\'' {
					s.next(ctx)
					break
				}
			}
		}
	}

	return string(s.src[offs+1:s.offset-1])
}

func (s *scanner) scanString(ctx Context, ml bool) string {
	// '"' opening already consumed
	offs := s.offset - 1
	if ml { offs -= 1 }

	for s.offsetRead < len(s.src) {
		ch := s.ch
		if (!ml && ch == '\n') || ch < 0 {
			erro(pc(ctx,offs), "string literal not terminated")
			break
		}
		s.next(ctx)
		if ch == '"' {
			if !ml {
				break
			}
			if s.ch == '"' {
				if s.next(ctx); s.ch == '"' {
					s.next(ctx)
					break
				}
			}
		}
		switch ch {
		case '\\': s.scanEscape(ctx, '"')
		case '$': //
		}
	}
	return string(s.src[offs:s.offset])
}

func (s *scanner) scanStrcomp(ctx Context, q rune) (tok token) {
	if q != 0 && s.ch == q {
		switch q {
		case '"':
			s.pop(isStrcompString)
			s.next(ctx) // take the ending '"'
			return COMPOSED
		case '}':
			s.pop(isBracedPlain)
			s.next(ctx) // take the ending '}'
			return RBRACE
		}
	}

	var offs = s.offset

	switch s.ch {
	case '{':
		if q == '}' {
			s.next(ctx)
			return LBRACE
		}
	case '\n':
		s.pop(isStrcompString|isStrcompLine)
		if s.next(ctx); s.ch != '\t' { s.bits &^= isRecipes }
		return LINEND
	case '\\':
		if s.next(ctx); q == 0 {
			s.lit = string(s.ch) // String channel!
			s.next(ctx)
			if s.bits&isRecipes != 0 && s.ch == '\t' {
				s.next(ctx)
			}
			return ESCAPE
		} else if s.scanEscape(ctx, q) {
			s.lit = string(s.src[offs+1:s.offset]) // String channel!
			return ESCAPE
		} else {
			s.lit = string(s.src[offs:s.offset]) // String channel!
			erro(pc(ctx,offs), "illegal strcomp escape %#U", s.ch)
			s.next(ctx)
			return ILLEGAL
		}
	case '&', '$':
		if n := s.offset+1; n < len(s.src) && rune(s.src[n]) == s.ch {
			s.next(ctx)
			s.next(ctx)
		} else if s.ch == '$' {
			return DELEGATE // No s.lit needed here
		} else {
			return CLOSURE  // No s.lit needed here
		}
	}

rawloop:
	for ; s.offsetRead < len(s.src) ; s.next(ctx) {
		switch s.ch {
		case '\\', '\n', '$', '&', q: break rawloop
		case '{':
			if i := s.offsetRead; i < len(s.src) && s.src[i] == '=' {
				return LBRACE // No s.lit needed here
			}
		}
	}

	// CRITICAL FIX: Safe, standard string allocation. Do NOT intern this!
	s.lit = string(s.src[offs:s.offset])
	return RAW
}

func (s *scanner) scan(ctx Context) {
	// 1. Reset both channels at the start of the scan
	s.tok, s.sym, s.lit, s.pos = ILLEGAL, symEmpty, "", s.file.Pos(s.offset)

	if false && checkpoints { defer func() {
		debug(pc(ctx, s.pos),
			_f("%s", string(s.src[s.offset:])),
			_f("tok=%v sym=%s lit='%s'", s.tok, s.sym, s.lit),
			callstack{num:10})
	}()}

	switch {
	case s.offset >= len(s.src) || s.ch == -1: s.tok = EOF; return
	case s.bits.isBracedPlain()  : s.tok = s.scanStrcomp(ctx, '}')
	case s.bits.isStrcompString(): s.tok = s.scanStrcomp(ctx, '"')
	case s.bits.isStrcompLine()  : s.tok = s.scanStrcomp(ctx, 0)
	}

	if s.tok != 0 {
		switch s.tok {
		case CLOSURE, DELEGATE, LBRACE:
		default: return
		}
		switch s.ch {
		case '$', '&', '{':
		default:
			erro(pc(ctx,s), "unexpected '%s'", string(s.src[s.offset:]))
		}
	}

	if IsDigit(s.ch) { // '0' <= s.ch && s.ch <= '9'
		s.tok = s.scanNumber(ctx, false)
		return
	}

	if IsLetter(s.ch) {
		s.scanIdentifier(ctx) // Fast Channel!

		// CRITICAL FIX: Downgrade keywords to WORD if they are immediately followed by a dash.
		if s.sym != symEmpty && s.ch != '/' && s.ch != '.' && s.ch != '~' && s.ch != '-' {
			// (If lookup_keyword is upgraded to take a Symbol, pass s.sym directly here!)
			if s.tok = lookup_keyword(s.sym) ; !s.tok.is_keyword() && s.tok != WORD {
				erro(pc(ctx,s), "unexpected token '%s' %s", s.tok, s.sym)
			}
		} else {
			s.tok = WORD
		}
		if s.bits.isCallZero() { s.pop(/*isCall*/0) }
		return
	}

	var ch, offs = s.ch, s.offset

	s.next(ctx)

	if s.bits.isBraceRaw() {
		switch ch {
		case '$':
			if s.ch == '$' {
				s.next(ctx)
				s.tok, s.lit = RAW, string(ch)
				return
			} else if false {
				debug(pc(ctx,s.offsetPos(offs)), "%s %s", string(ch), string(s.ch))
			}
		case '&':
			if s.ch == '&' {
				s.next(ctx)
				s.tok, s.lit = RAW, string(ch)
				return
			} else if false {
				debug(pc(ctx,s.offsetPos(offs)), "%s %s", string(ch), string(s.ch))
			}
		case '\\':
			if s.tok = ESCAPE; IsDigit(s.ch) {
				s.scanNumber(ctx, false)
			} else {
				s.lit = string(s.ch)
				s.next(ctx) // escape a single char
			}
			return
		case '{':
			s.push(isBraceRaw)
			s.tok, s.lit = RAW, string(ch)
			return
		case '}':
			t := s.bits.isBrace()
			s.pop(isBrace|isBraceRaw)
			if t {
				s.tok = RBRACE
				return
			} else {
				s.tok, s.lit = RAW, string(ch)
				return
			}
		default:
			s.tok, s.lit = RAW, string(ch)
			return
		}
	}

	switch ch {
	case '#':
		if s.bits.isCommentsOff() {
			s.tok, s.lit = HASH, string(ch)
		} else {
			s.tok, s.lit = COMMENT, s.scanComment(ctx)
			s.next(ctx) // discard '\n'
		}
	case '!':
		if s.tok = EXC; s.ch == '=' {
			s.tok = ASSIGN_EXC
			s.next(ctx)
		}
	case '?':
		if s.tok = QUE; s.ch == '=' {
			s.tok = ASSIGN_QUE
			s.next(ctx)
		}
	case '+':
		if s.tok = PLUS; s.ch == '=' {
			s.tok = ASSIGN_ADD
			s.next(ctx)
		}
	case '-':
		if s.ch == '-' { // "-->" => "-", "->"
			if s.offsetRead < len(s.src) && s.src[s.offsetRead] == '>' {
				// CRITICAL FIX: Words belong in the sym channel!
				s.tok, s.sym = WORD, symDash
			} else {
				s.tok = MINUS
			}
		} else if s.ch == '=' { // -=
			s.tok = ASSIGN_POP
			s.next(ctx)
			if s.ch == '+' { // -=+
				s.tok = ASSIGN_SUS
				s.next(ctx)
			}
		} else if s.ch == '+' { // -+
			s.next(ctx)
			if s.ch == '=' { // -+=
				s.tok = ASSIGN_SAD
				s.next(ctx)
			} else {
				s.tok = ILLEGAL
			}
		} else if s.ch == '>' {
			s.tok = SELECT_PROP
			s.next(ctx)
		} else if '0' <= s.ch && s.ch <= '9' {
			s.tok = s.scanNumber(ctx, false)
			s.lit = "-" + s.lit // minus number
		} else {
			s.tok = MINUS
		}
	case '\\':
		s.tok, s.lit = ESCAPE, string(s.ch)
		s.next(ctx) // eat escaped char
		if s.bits&isRecipes != 0 && s.ch == '\t' {
			s.next(ctx) // skip escaped recipe-tab
		}
	case '\'':
		if s.tok = STRING; s.ch == '\'' {
			if s.next(ctx); s.ch == '\'' { // '''
				s.lit = s.scanStrlit(ctx, true)
			} else if offs := s.offset - 2; false {
				s.lit = string(s.src[offs:s.offset])
			} else {
				s.lit = "" // empty string ''
			}
		} else {
			s.lit = s.scanStrlit(ctx, false)
		}
	case '"':
		if s.bits.isStrcompString() {
			erro(pc(ctx,s.offsetPos(offs)), "composed")
		} else {
			s.tok = STRCOMP
			s.push(isStrcompString)
		}
	case '$', '&':
		if ch == '&' { s.tok = CLOSURE } else { s.tok = DELEGATE }
		if ch = rune(s.src[s.offset]); ch == '(' || ch == '{' {
			s.push(isCall)
		} else if false {
			s.push(isCall /* | isCallZero */)
		}
	case '(':
		s.tok = LPAREN
		if s.bits.isCallZero() { s.bits |= isCallParen } else { s.push(isGroup) }
	case ')':
		s.tok = RPAREN
		t := isCallParen|isGroup
		if s.bits&t == 0 {
			if n := len(s.bitss); n > 0 {
				if b := s.bitss[n-1]; b&t != 0 {
					s.bits, s.bitss = b, s.bitss[0:n-1]
					goto poprparen
				}
			}
			erro(pc(ctx,s.offsetPos(offs)), "unexpected right-paren, %016b %016b", s.bits, s.bitss)
		}
		poprparen: s.pop(t)
	case '{':
		s.tok = LBRACE
		if s.bits.isCallZero() { s.bits |= isCallBrace } else { s.push(isBrace) }
	case '}':
		s.tok = RBRACE
		t := isCallBrace|isBrace
		if s.bits&t == 0 {
			if n := len(s.bitss); n > 0 {
				if b := s.bitss[n-1]; b&t != 0 {
					s.bits, s.bitss = b, s.bitss[0:n-1]
					goto poprbrace
				}
			}
			erro(pc(ctx,s.offsetPos(offs)), "unexpected right-brace, %016b %016b", s.bits, s.bitss)
		}
		poprbrace: s.pop(t)
	case '=':
		if s.ch == '>' {
			s.tok = SELECT_PROG1
			s.next(ctx)
		} else if s.ch == '+' {
			s.tok = ASSIGN_USH
			s.next(ctx)
		} else {
			s.tok = ASSIGN
		}
	case ' ', '\t':
		if ch == '\t' && s.canRecipe() {
			s.tok, s.lit = RECIPE, string(ch)
			s.push(isStrcompLine)
		} else {
			for s.ch == ' ' || s.ch == '\t' { s.next(ctx) }
			s.tok, s.lit = SPACE, string(s.src[offs:s.offset])
		}
	case '~':
		if s.ch == '>' { s.next(ctx)
			s.tok = SELECT_PROG2
		} else {
			s.tok = TILDE
		}
	case '.':
		if s.tok = DOT; s.ch == '.' { s.next(ctx)
			s.tok = DOTDOT
		} else if IsDigit(s.ch) {
			if n := s.offset-2; n > -1 && unicode.IsSpace(rune(s.src[n])) {
				s.tok = s.scanNumber(ctx, true)
			}
		}
	case ':':
		if s.ch == '=' { s.next(ctx)
			s.tok = ASSIGN_CO1
		} else if s.ch == ':' { s.next(ctx)
			if s.ch == '=' { s.next(ctx)
				s.tok = ASSIGN_CO2
			} else {
				s.tok = DOLON
			}
		} else {
			s.tok = COLON
		}
	case '*':
		switch s.ch {
		case '*': s.next(ctx); s.tok = DAST
		case '?': s.next(ctx); s.tok = ASTQ
		default: s.tok = SAST
		}
	case '%': s.tok = PERC
	case '@': s.tok = AT
	case '|': s.tok = BAR
	case '/': s.tok = PCON
	case ',': s.tok = COMMA
	case '→': s.tok = SELECT_PROP
	case '⇒': s.tok = SELECT_PROG1
	case '⇢': s.tok = SELECT_PROG2
	case '≔': s.tok = ASSIGN_CO1
	case '⩴': s.tok = ASSIGN_CO2
	case ';':
		if s.ch == '=' { s.tok = ASSIGN_SC1 ; s.next(ctx) } else
		if s.ch == ':' { s.tok = SOLON      ; s.next(ctx)
			if s.ch == '=' { s.tok = ASSIGN_CO3 ; s.next(ctx) }
		} else {
			s.tok = SEMICOLON
		}
	case '^': s.tok = CARET
	case '[': s.tok = LBRACK
	case ']': s.tok = RBRACK
	case '<': s.tok = LANGLE
	case '>': s.tok = RANGLE
	case '⟨': s.tok = Lchevron
	case '⟩': s.tok = Rchevron
	case '⌜': s.tok = Ltop_corner
	case '⌟': s.tok = Rbot_corner
	case '⌝': s.tok = Rtop_corner
	case '⌞': s.tok = Lbot_corner
	case '‹': s.tok = Lsing_guil
	case '›': s.tok = Rsing_guil
	case '«': s.tok = Lguillemet
	case '»': s.tok = Rguillemet
	case '\n':
		s.tok = LINEND
		if s.pop(isStrcompLine); s.ch != '\t' { s.bits &^= isRecipes }
	default:
		if ch != bom {
			erro(pc(ctx,s.offsetPos(s.file.Offset(s.pos))), "illegal %#U", ch)
		} else {
			s.tok, s.lit = ILLEGAL, string(ch)
		}
	}
	return
}

const (
    dot_base      = ".base"
    dot_configure = ".configure"
    dot_container = ".container"

    mainFileName = "do.smart"
    altrFileName = "work.smart"
    deprFileName = "build.smart"

	maxDigitAutoNum = 9
    optSortErrors = false
)

type ResolveBits int

const (
    FromBase ResolveBits = 1<<iota
    FromProject
    FromGlobe
    FromHere

    FindDef
    FindRule

    anywhere = FromHere
    local    = FromProject
    global   = FromGlobe
    nonlocal = FromGlobe | FromBase | FromProject
)

type EvalBits int

const (
    KeepClosures EvalBits = 1<<iota
    KeepDelegates

    // Wants value for rule depends.
    DependValue

    // Wants v.string(ctx), expands delegates and closures,
    // turn off KeepClosures, KeepDelegates.
    StringValue = 0
)

// A Mode value is a set of flags (or 0).
// They control the amount of source code parsed and other optional
// parser functionality.
type Mode uint

const (
    ModuleClauseOnly Mode = 1<<iota // stop parsing after project or module clause
    ImportsOnly               // stop parsing after import declarations
    ParseComments             // parse comments and add them to AST
    AllErrors
)

type use_spec struct{
	props []Value
}

type template struct{
	state scanstate
	end  *scanstate
	endPos Pos // token position
	name Symbol // 'def' name or empty
	params []Value
}

type parser struct{
	scanner

	stop Pos // parsing and stop position

	templates []*template

	imports []*use_spec // list of imports
	targets []Value // targets of current rule
	ruparas []*auto // parameters of current rule
	dialect  Symbol // recipe dialect of current rule

	locals []map[Symbol]*def

	comments  []*commentgroup
	leadComment *commentgroup // last lead comment
	lineComment *commentgroup // last line comment
}

type (
	left_hand_side  struct{}
	is_undef        struct{}
	p_is_params     struct{}
	p_is_glob       struct{}
	p_no_argumented struct{}
	p_no_path       struct{}
)

type      aware_c     struct{ token }
type      aware_token struct{ token }
type    p_aware       struct{ Context; token }
func (p p_aware) ts(t string) (_ string) {
	return "{="+t+" "+p.token.String()+" "+ts(p.Context)+"}"
}
func (p p_aware) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case aware_token: return t.token == p.token
	case aware_c: if t.token == p.token { return p }
	}
	return p.Context.do(ctx, op)
}

func aware(ctx Context, t token) (res Context) {
	// if x, y := do(ctx, aware_c{t}).(p_aware); y { return x }
	return p_aware{ctx, t}
}

type can_select struct{}
type selection struct{ Context }
func (p selection) cast(t reflect.Type) Context { return icast(p,t) }
func (p selection) inner() Context { return p.Context }
func (p selection) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case can_select: return true
	}
	return p.Context.do(ctx, op)
}

type is_braced  struct{}
type keep_autos struct{}
type defval         struct{ original ; d *def }
type def_name       struct{ Context }
type braced         struct{ Context }
type p_auto_ctx     struct{ Context }
type foreach_txt    struct{ Context ; a *auto }
type grep_txt       struct{ Context ; o objbase ; a map[Symbol]*auto }
type p_group_ctx    struct{ Context }
type left_side      struct{ Context }
type p_params       struct{ Context }
type p_path         struct{ Context }
type p_perc         struct{ Context }
type p_glob         struct{ Context }
type p_regex        struct{ Context }
type p_strcomp      struct{ Context }
type p_undef        struct{ Context }
type p_rule_ctx     struct{ Context ; p *parser }
type p_recipe struct{
	Context
	builtin bool
	lines [][]Value
	elems []Value
}

type codeblock struct{ *automatic }
func (p *codeblock) inner() Context { return p.Context }
func (p *codeblock) cast(t reflect.Type) Context {
    if reflect.TypeOf(p) == t { return p }
    return p.automatic.cast(t)
}

func (p p_undef) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_undef) inner() Context { return p.Context }
func (p p_undef) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case is_undef: return true
	}
	return p.Context.do(ctx, op)
}

func (p p_perc) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_perc) inner() Context { return p.Context }
func (p p_perc) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case p_no_argumented: return true
	}
	return p.Context.do(ctx, op)
}

func (p p_glob) inner() Context { return p.Context }
func (p p_glob) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_glob) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case p_is_glob: return true
	}
	return p.Context.do(ctx, op)
}


type p_is_strcomp struct{}
func (p p_strcomp) inner() Context { return p.Context }
func (p p_strcomp) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_strcomp) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case p_is_strcomp: return true
	}
	return p.Context.do(ctx, op)
}

func (p p_params) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_params) inner() Context { return p.Context }
func (p p_params) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case p_is_params: return true
	}
	return p.Context.do(ctx, op)
}

type is_auto_preserved struct{ s Symbol }
type is_auto           struct{ s Symbol }
type is_defname        struct{}

func (p p_auto_ctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_auto_ctx) inner() Context { return p.Context }
func (p p_auto_ctx) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case is_auto: return true
	}
	return p.Context.do(ctx, op)
}

func (p def_name) cast(t reflect.Type) Context { return icast(p,t) }
func (p def_name) inner() Context { return p.Context }
func (p def_name) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case is_defname, symident: return true
	}
	return p.Context.do(ctx, op)
}

func (p defval) inner() Context { return p.Context }
func (p defval) cast(t reflect.Type) Context { return icast(p,t) }
func (p defval) do(ctx Context, op any) any {
	switch t := op.(type) {
	case is_auto: return t.s != sym_0 && (sym_0 <= t.s && t.s <= sym_0)//IsDigits(t.s)
	case keep_autos: return true
    case origin_def:
        if p.d != nil && (t.name == symEmpty || t.name == p.d.name) { return p.d }
	}
	return p.original.do(ctx, op)
}

func (p braced) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case is_braced: return true
	}
	return p.Context.do(ctx, op)
}

func (p *foreach_txt) inner() Context { return p.Context }
func (p *foreach_txt) cast(t reflect.Type) Context { return icast(p,t) }
func (p *foreach_txt) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
    case find_auto: if t.s == symUnderscore { return p.a }
	case is_auto: if t.s == symUnderscore { return true }
	case is_auto_preserved: if t.s == symUnderscore { return true }
	}
	return p.Context.do(ctx, op)
}

func (p *grep_txt) inner() Context { return p.Context }
func (p *grep_txt) cast(t reflect.Type) Context { return icast(p,t) }
func (p *grep_txt) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case is_auto_preserved:
		if p.a != nil {
			if _, y := p.a[t.s]; y { return true }
		}
	case is_auto:
		if p.a != nil {
			if _, y := p.a[t.s]; y { return true }
		}
    case find_auto:
		if p.a != nil {
			if x, y := p.a[t.s]; y { return x }
		}
	case regex_subexp_auto:
		p.a = map[Symbol]*auto{sym_0:&auto{knownobject{p.o, sym_0}}}
		for i, name := range t.SubexpNames() {
			if 0 < i {
				if name == "" { name = strconv.Itoa(i) }
				var sym = intern(name)
				p.a[sym] = &auto{knownobject{p.o, sym}}
			}
		}
	}
	return p.Context.do(ctx, op)
}

type is_rule_ctx struct{}
func (p p_rule_ctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_rule_ctx) inner() Context { return p.Context }
func (p p_rule_ctx) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case is_rule_ctx: return true
	case is_auto:
		if sym_0 <= t.s && t.s <= sym_9 { return true }
		if _, y := rule_autos[t.s]; y { return true }
	}
	return p.Context.do(ctx, op)
}

type add_recipe_line struct{ a []Value }
type is_recipe_start struct{}
type is_recipe       struct{ bool } // builtin or text

func (p *p_recipe) cast(t reflect.Type) Context { return icast(p,t) }
func (p *p_recipe) inner() Context { return p.Context }
func (p *p_recipe) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case is_recipe:
		return t.bool == p.builtin
	case is_recipe_start:
		return p.elems == nil
	case add_recipe_line:
		p.lines = append(p.lines, t.a)
		return len(p.lines)
	}
	return p.Context.do(ctx, op)
}

func (p left_side) cast(t reflect.Type) Context { return icast(p,t) }
func (p left_side) inner() Context { return p.Context }
func (p left_side) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case left_hand_side: return true
	}
	return p.Context.do(ctx, op)
}

func (p *parser) ts(_t string) string {
	t, s := p.tok.String(), p.scanner.file.Name()
	return "{="+_t+" "+t+" "+s+"}"
}

// ----------------------------------------------------------------------------
// Parsing support

func (p *parser) trace(a ...any) { l_traverse.traceAt(p.Position(), a...) }

func (p *parser) scan(ctx Context) {
	// Because of one-token look-ahead, print the previous token
	// when tracing as it provides a more readable output. The
	// very first token (!p.pos.IsValid()) is not initialized
	// (it is ILLEGAL), so don't print it .
	if l_traverse.enabled && p.pos.IsValid() {
		s := p.tok.String()
		switch {
		case p.tok.is_literal():
			p.trace(s, p.lit)
		case p.tok.is_operator(), p.tok.is_keyword():
			p.trace("\"" + s + "\"")
		default:
			p.trace(s)
		}
	}

	// if n := p.scanner.ch_bytes(); n > 1 { p.multibyte += n-1 }

	switch p.scanner.scan(ctx); p.tok {
	// case COMMENT, LINEND: p.multibyte = 0
	}
}

func (p *parser) step(ctx Context) {
	p.leadComment = nil
	p.lineComment = nil

	var prev = p.pos
	if p.scan(ctx); p.tok == COMMENT {
		var comment *commentgroup
		var endline int

		// If the comment is on same line as the previous token;
		// it cannot be a lead comment but may be a line comment.
		if p.scanner.file.Line(p.pos) == p.scanner.file.Line(prev) {
			comment, endline = p.commentgroup(ctx, 0)
			if p.scanner.file.Line(p.pos) != endline {
				// The next token is on a different line, thus
				// the last comment group is a line comment.
				p.lineComment = comment
			}
		}

		// consume successor comments, if any
		endline = -1
		for p.tok == COMMENT {
			comment, endline = p.commentgroup(ctx, 1)
		}

		if endline+1 == p.scanner.file.Line(p.pos) {
			// The next token is following on the line immediately after the
			// comment group, thus the last comment group is a lead comment.
			p.leadComment = comment
		}
	}

	if false && p.tok != LINEND && p.lineComment != nil { p.tok = LINEND }
}

func (p *parser) next(ctx Context, ws bool) {
	if p.step(ctx); ws { p.spaces(ctx) }
}

func (p *parser) spaces(ctx Context) {
	for p.lineComment == nil && p.tok != EOF {
		if p.tok == SPACE || (p.tok == RECIPE && truly(ctx, is_recipe{true})) {
			p.step(ctx)
		} else if p.tok == ESCAPE && p.lit == "\n" {
			if p.step(ctx); p.tok == LINEND || p.lineComment != nil { break }
			if truly(ctx, is_recipe{true}) {
			tokloop:
				for p.tok != EOF {
					switch p.tok {
					case RECIPE: // TODO: using p.recipe_start()
						if true { p.scanner.pop(isStrcompLine) }
						p.step(ctx)
					default:
						break tokloop
					}
				}
			}
		} else {
			break
		}
	}
}

func (p *parser) forwardLine(ctx Context) {
	for p.tok != EOF {
		if p.next(ctx, true) ; p.tok == LINEND {
			p.next(ctx, true) ; break
		}
	}
}

func (p *parser) emptyLines(ctx Context) {
	for p.spaces(ctx); p.tok == LINEND; p.spaces(ctx) { p.step(ctx) }
}

func (p *parser) comment(ctx Context) (res *comment, endline int) {
	// /*-style comments may end on a different line than where they start.
	// Scan the comment for '\n' chars and adjust endline accordingly.
	endline = p.scanner.file.Line(p.pos)
	if len(p.lit) > 1 && p.lit[1] == '*' {
		// don't use range here - no need to decode Unicode code points
		for i := 0; i < len(p.lit); i++ {
			if p.lit[i] == '\n' {
				endline++
			}
		}
	}

	res = &comment{p.pos, p.lit}
	p.scan(ctx)

	return
}

func (p *parser) commentgroup(ctx Context, n int) (res *commentgroup, endline int) {
	res = new(commentgroup)
	p.comments = append(p.comments, res)
	endline = p.scanner.file.Line(p.pos)
	for p.tok == COMMENT && p.scanner.file.Line(p.pos) <= endline+n {
		var com *comment
		com, endline = p.comment(ctx)
		res.comments = append(res.comments, com)
	}
	return
}

func (p *parser) loc(a Pos) Position { return p.scanner.file.Position(a) }
func (p *parser) Position() (r Position) { return p.loc(p.pos) }

func (p *parser) is_file(s string) bool {
	return strings.HasSuffix(p.scanner.file.Name(), s)
}

func (p *parser) expect(ctx Context, tok token) Pos {
	var pos = p.pos
	if p.tok == tok {
		p.step(ctx) // move forward
	} else {
		erro(pc(ctx,p), "expect %v, not %v", tok, p.tok, callstack{num:10})
	}
	return pos
}

func (p *parser) linend(ctx Context) (ok bool) {
	if p.lineComment != nil {
		p.lineComment, ok = nil, true
	} else if p.tok == EOF {
		ok = true
	} else if p.tok == LINEND {
		p.step(ctx); ok = true
	} else {
		erro(pc(ctx,p), "expect end of line, but %v", p.tok, callstack{num:10})
	}
	return
}

func (p *parser) recipe_start() (res bool) {
	if p.tok == RECIPE {
		res = true
	} else if p.tok == SPACE && p.lit == "\t" {
		p.tok, res = RECIPE, true // FIXES recipe \t
	}
	return
}

// ----------------------------------------------------------------------------
// Words & Identifiers

func (l ul) arrow(ctx Context, lhs Value) (res Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "arrow")) }

	pos, tok := lhs.Pos(), l.p.tok // the arrow '->' or '=>'
	ctx = pc(ctx, pos)
	l.p.step(ctx) // skip '->' or '=>'

	switch t := lhs.(type) {
	case *arrow:
		return _arrow(pos, tok, t, l.composite(ctx))
	case *word:
		if o := l.resolve(ctx, t, t.s); !isNull(o) {
			return _arrow(pos, tok, o, l.composite(ctx))
		} else if tok == SELECT_PROG2 {
			return _null(pos) // ignore
		} else {
			debug(ctx,
				_f("%v: '%v' is undefined (name=%v, obj=%v)", l.project, lhs, t, o),
				_f("%v: parser is here (name=%s, tok=%s)", l.project, lhs, tok),
				_f("%v: parser to go here (tok=%s, lit=%s)", l.project, l.p.tok, l.p.lit),
				trace{})
		}
	case *compound:
		// Fast-path extraction
		if o := l.resolve(ctx, t, intern(__string(ctx, lhs))); !isNull(o) {
			return _arrow(pos, tok, o, l.composite(ctx))
		} else if tok == SELECT_PROG2 {
			return _null(pos) // ignore
		} else {
			debug(ctx,
				_f("%v: '%v' is undefined (name=%v, obj=%v)", l.project, lhs, t, o),
				_f("%v: parser is here (name=%s, tok=%s)", l.project, lhs, tok),
				_f("%v: parser to go here (tok=%s, lit=%s)", l.project, l.p.tok, l.p.lit),
				trace{})
		}
	}
	return _arrow(pos, tok, lhs, l.composite(ctx))
}

func (p *parser) bare(ctx Context) *word {
	w := _word(p.pos, p.sym)
	p.step(ctx)
	return w
}

func (l ul) braced(ctx Context) (x Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "braced")) }

	pos := l.p.pos

	if l.p.expect(ctx, LBRACE); truly(ctx, is_rule_ctx{}) {
		if l.p.tok == LINEND && l.p.tok != ASSIGN { l.p.emptyLines(ctx) }
	}

	ctx = braced{ctx}

	switch l.p.tok {
	case LBRACK: // obsolete syntax: {[...]}
		erro(ctx, "syntax error, use {(modifier)} for modification")
	case LBRACE:
		erro(ctx, "syntax error, use {=join (sep) list} for conjunction")
		if false { // conjunction: {{list}sep}
			l.p.next(ctx, true) // consume the inner '{'

			v := l.values(ctx)  // parse the inner list elements
			l.p.spaces(ctx)
			l.p.expect(ctx, RBRACE) // consume the inner '}'

			s := l.values(ctx)  // parse the separator

			x = &conjunction{list{elements{v}}, ease(ctx, s)}

			l.p.spaces(ctx)
			l.p.expect(ctx, RBRACE) // consume the outer '}'
		}
	case LPAREN:
		x = l.modification(ctx)
		l.p.spaces(ctx)
		l.p.expect(ctx, RBRACE)
	case ASSIGN: // =
		switch l.p.step(ctx); l.p.tok {
		case AND:     x = l.braced_and(ctx)
		case OR:      x = l.braced_or(ctx)
		case FOR:     x = l.braced_for(ctx)
		case FOREACH: x = l.braced_foreach(ctx)
		case PROJECT: // {=project ...}
			l.p.next(ctx, true)
			x = l.braced_project(ctx)
			l.p.expect(ctx, RBRACE)

		case BARE: // {=bare ...}
			l.p.next(ctx, true)
			x = l.p.bare(ctx)
			l.p.spaces(ctx)
			l.p.expect(ctx, RBRACE)

		case RAW: // {=raw ...}
			l.p.next(ctx, true)
			x = &raw{valbase{l.p.pos}, __string(ctx, l.expr(ctx))}
			l.p.spaces(ctx)
			l.p.expect(ctx, RBRACE)

		case UNDEF: // {=undef ...}
			l.p.next(ctx, true)
			x = undef{l.expr(ctx)}
			l.p.spaces(ctx)
			l.p.expect(ctx, RBRACE)

		case NULL: // {=null}
			x = l.braced_null(ctx)

		case NONE: // {=none ...}
			x = l.braced_none(ctx)

		case ANSWER, BOOL, BOOLEAN, BIN, OCT, INT, HEX, FLOAT: // {=bin ...}, {=oct ...}, {=int ...}, {=hex ...}, {=float ...}
			x = l.braced_type(ctx, l.p.tok)

		case TRUE, FALSE, YES, NO, ON, OFF: // {=true}, {=false}, {=yes}, {=no}, {=on}, {=off}
			x = l.braced_const(ctx, l.p.tok)

		case FILE: // {=file ...}
			x = l.braced_file(ctx)

		case PATH: // {=path ...}
			x = l.braced_path(ctx)

		case GLOB: // {=glob ...}
			l.p.next(ctx, true)
			g := l.glob(ctx, nil)
			l.p.spaces(ctx)
			l.p.expect(ctx, RBRACE)
			x = &globbrace{*g}

		case REGEX: // {=regex ...}
			l.p.step(ctx)
			l.p.scanner.addBits(isBraceRaw)

			// Trim leading spaces differently to avoid messing the scan states.
			// NOTE: the first SPACE and WORD do not become RAW.
			for l.p.tok == SPACE || (l.p.tok == RAW && l.p.lit == " ") { l.p.step(ctx) }
			x = l.regex(ctx)

		case WORD:
			switch l.p.sym {
			case symJoin: // {=join [(sep)] [list...]}
				l.p.next(ctx, true) // consume 'join'
				l.p.spaces(ctx)

				var sep Value
				if l.p.tok == LPAREN {
					l.p.next(ctx, true) // consume '('

					s := l.values(ctx)  // parse the separator
					l.p.spaces(ctx)
					l.p.expect(ctx, RPAREN) // consume ')'
					l.p.spaces(ctx)

					sep = ease(ctx, s)
				}

				v := l.values(ctx) // parse the rest of the list

				x = &conjunction{list{elements{v}}, sep}

				l.p.expect(ctx, RBRACE) // consume '}'

			case symHere:
				l.p.next(ctx, true)
				l.p.expect(ctx, RBRACE)
				p := l.p.Position()
				x = &compound{elements{[]Value{
					_raw(l.p.pos, p.Filename), _punct(ctx, COLON),
					_decimal(l.p.pos, int64(p.Line)), _punct(ctx, COLON),
					_decimal(l.p.pos, int64(p.Column)), _punct(ctx, COLON),
				}}}

			case symPlain:
				x = &plain{elements{l.braced_plain(ctx)}, l.p.sym}
				l.p.expect(ctx, RBRACE)

			case symPlainLine:
				x = &plainline{elements{l.braced_plain(ctx)}}
				l.p.expect(ctx, RBRACE)

			case symSelf:
				l.p.next(ctx, true)
				x = self{l.braced_project(ctx)}
				l.p.expect(ctx, RBRACE)

			case symStr: // $(string ...)
				x = l.braced_str(ctx)
				l.p.expect(ctx, RBRACE)

			case symQuote: // $(quote ...)
				x = l.braced_quote(ctx)
				l.p.expect(ctx, RBRACE)

			case symWord:
				x = l.braced_word(ctx)
				l.p.expect(ctx, RBRACE)

			case symGrep:
				if false {
					x = l.braced_word(ctx)
					l.p.expect(ctx, RBRACE)
				}

			case symDefs:
				x = l.braced_defs(ctx)
				l.p.expect(ctx, RBRACE)

			case symFullname:
				x = l.braced_fullname(ctx)
				l.p.expect(ctx, RBRACE)

			default:
				erro(pc(ctx,l.p), "unsupported braced type: %v %v", l.p.tok, l.p.lit)
			}

		default:
			l.p.next(ctx, true)
		}
	case RBRACE:
		x = &null{valbase{l.p.pos}}
		l.p.spaces(ctx)
		l.p.step(ctx) // consumes }
	default: // {...}
		switch v := l.values(ctx); len(v) {
		case 0 : x = _null(pos)
		case 1 : x = &disjunction{valbase{pos},v[0]}
		default: x = &disjunction{valbase{pos},_list(v...)}
		}
		l.p.spaces(ctx)
		l.p.expect(ctx, RBRACE)
	}
	return
}

func (l ul) braced_elems(ctx Context) (elems []Value) {
	for l.p.tok != RBRACE && l.p.tok != EOF {
		switch l.p.tok {
		case SPACE:
			l.p.spaces(ctx)
		case LBRACE:
			elems = append(elems, l.braced(ctx))
		case RAW:
			elems = append(elems, l.literal(ctx))
		default:
			elems = append(elems, l.expr(ctx))
		}
	}
	return
}

// ----------------------------------------------------------------------------
// Common productions

func (p *parser) is_end_of_line() bool {
	// If there's a comment right after the parsed expression, we break
	// the expression list to treat the end-of-line comment like a LINEND.
	return p.lineComment != nil || p.tok == LINEND || p.tok == EOF
}

func (p *parser) is_list_term(ctx Context) bool {
	// If there's a comment right after the parsed expression, we break
	// the expression list to treat the end-of-line comment like a LINEND.
	if p.lineComment != nil || p.tok.is_list_delim() || (truly(ctx, left_hand_side{}) && p.tok.is_assign()) {
		return true
	}
	return p.tok == RECIPE && truly(ctx, is_recipe{false})
}

func (p *parser) rule_params(ctx Context, args []Value) (err error) {
	var s = _scope(ctx)
	for _, arg := range args {
		var a *auto
		switch t := arg.(type) {
		case *word:
			a = &auto{knownobject{objbase{valbase{arg.Pos()}, s}, t.s}}
			s.alias(ctx, a, t.s) // Map "ARG1" -> auto

		case *compound/* , *qualword */:
			name := intern(__string(ctx, arg))
			a = &auto{knownobject{objbase{valbase{arg.Pos()}, s}, name}}
			s.alias(ctx, a, name) // Map "ARG1" -> auto

		default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
			erro(ctx, "bad parameter form (%v)", ts(arg))
			return
		}

		n := strconv.Itoa(len(p.ruparas)+1)
		s.alias(ctx, a, intern(n)) // Map "1" -> auto
		p.ruparas = append(p.ruparas, a)
	}
	return
}

func (l ul) depends(ctx Context, params bool) (res []Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "depends")) }

	l.p.spaces(ctx)

	if params && l.p.tok == LPAREN {
		if g := l.group(p_params{ctx}); 0 < g.len() {
			if x, y := g.elems[0].(*group); y && g.len() == 1 { g = x }
			l.p.rule_params(ctx, g.elems)
		}
	}

	for l.p.tok != BAR && l.p.tok != SEMICOLON && !l.p.is_end_of_line() {
		if l.p.tok == COLON {
			// FIXME: this check is not working!
			// FIXME: detects unexpected colon ':'
			erro(ctx, "unexpected colon")
		} else if l.p.spaces(ctx) ; !l.p.is_end_of_line() {
			var val = l.expr(selection{ctx})

			if x, y := val.(*globpat); y && x.len() == 1 {
				if z, y := x.elems[0].(*globrange); y {
					debug(ctx, "use {%v} instead", z.Value)
				} else if z, y := x.elems[0].(*group); y {
					debug(ctx, "use {%v} instead", z.elems[0])
				} else {
					debug(ctx, "use {%v} instead", x.elems[0])
				}
			}

			res = append(res, merge(val)...)
			if l.p.tok == SPACE { l.p.next(ctx, true) }
		}
	}
	return
}

// If lhs is set, result list elements which are identifiers are not resolved.
func (l ul) values(ctx Context) (values []Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "values")) }

	for l.p.spaces(ctx); !l.p.is_list_term(ctx); l.p.spaces(ctx) {
		var prev = l.p.pos
		if values = append(values, l.expr(ctx)); l.p.pos == prev {
			erro(ctx, "bad: %v %v; %v", l.p.tok, l.p.lit, values)
		}

		// If there's a comment right after the parsed expression, we break
		// the expression list to treat the end-of-line comment like a LINEND.
		if l.p.tok == EOF || l.p.tok == LINEND || l.p.lineComment != nil { break }
	}
	return
}

func (l ul) group(ctx Context) *group {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "group")) }

	// Query the context to see if we are allowed to parse across lines
	allowMultiline := truly(ctx, is_directive_ctx{})

	ctx = p_group_ctx{aware(ctx,COMMA)}

	pos := l.p.pos
	l.p.expect(ctx, LPAREN)

	// Conditionally consume leading newlines
	if allowMultiline {
		for l.p.tok == SPACE || l.p.tok == LINEND { l.p.next(ctx, true) }
	} else {
		l.p.spaces(ctx)
	}

	var elems, converted = l.values(ctx), false
	for l.p.tok != RPAREN && l.p.tok != EOF {

		// Conditionally consume newlines between expressions
		if allowMultiline && l.p.tok == LINEND {
			for l.p.tok == LINEND || l.p.tok == SPACE { l.p.next(ctx, true) }
			continue
		}

		switch l.p.tok {
		case BAR, COMMA, SEMICOLON, SELECT_PROG1:
			elems = append(elems, l.punct(ctx))

			// Conditionally consume newlines after delimiters
			if allowMultiline {
				for l.p.tok == SPACE || l.p.tok == LINEND { l.p.next(ctx, true) }
			} else {
				l.p.spaces(ctx)
			}
		}

		p := l.p.pos
		next := _list(l.values(ctx)...)

		if l.p.pos == p { erro(ctx, "syntax error", callstack{num:64}) }

		if !converted {
			elems = []Value{ _list(elems...), next }
			converted = true
		} else {
			elems = append(elems, next)
		}
	}
	l.p.expect(ctx, RPAREN)
	return _group(pos, elems...)
}

func (l ul) corner_list(ctx Context) *list { // ⌜a b c⌟
	if l_traverse.enabled { defer un(l_trace(l_traverse, "corner")) }

	var elems []Value
	switch l.p.tok {
	case Lbot_corner, Ltop_corner: l.p.step(ctx)
	default: erro(pc(ctx,l.p), "unexpect %v", l.p.tok)
	}

corner_loop:
	for l.p.spaces(ctx); l.p.tok != EOF; l.p.spaces(ctx) {
		var saved = l.p.pos
		if elems = append(elems, l.expr(ctx)); l.p.pos == saved {
			erro(ctx, "bad: %v %v; %v", l.p.tok, l.p.lit, elems)
		}

		// If there's a comment right after the parsed expression, we break
		// the expression list to treat the line-end comment like a LINEND.
		if l.p.lineComment != nil { break }

		switch l.p.tok { case Rbot_corner, Rtop_corner, LINEND: break corner_loop }
	}

	switch l.p.tok {
	case Rbot_corner, Rtop_corner: l.p.step(ctx)
	default: erro(pc(ctx,l.p), "unexpect %v", l.p.tok)
	}

	return _list(elems...)
}

func (l ul) argumented(ctx Context, x Value) *argumented {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "argumented")) }

	ctx = p_group_ctx{aware(ctx,COMMA)}

	l.p.next(ctx, true) // skip LPAREN

	var a = []Value{ _list(l.values(ctx)...) }
	for l.p.tok != RPAREN && l.p.tok != LINEND && l.p.tok != EOF {
		switch l.p.tok {
		case COMMA: l.p.next(ctx, true) // skip COMMA
		case BAR, SEMICOLON:
			if false {
				a = append(a, l.punct(ctx))
				l.p.spaces(ctx)
			} else {
				erro(ctx, "unexpected punctuation: %v", l.p.tok)
			}
		}
		a = append(a, _list(l.values(ctx)...))
	}
	l.p.expect(ctx, RPAREN)
	return _argumented(x, a...)
}

func (l ul) globmeta(ctx Context) (x *globmeta) {
	p, t := l.p.pos, l.p.tok
	l.p.step(ctx)
	return _globmeta(p, t)
}

func (l ul) globrange(ctx Context) (x *globrange) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "globrange")) }

	l.p.expect(ctx, LBRACK) // skip '['

	chars := l.expr(ctx)

	l.p.expect(ctx, RBRACK) // skip ']'

	return _globrange(chars)
}

func (l ul) glob(ctx Context, x Value) (g *globpat) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "glob")) }

	ctx = p_glob{ctx}

	if y := x == nil; y {
		g = &globpat{}
	} else if g, y = x.(*globpat); !y || g == nil {
		g = _globpat(x)
	}

	for l.p.lineComment == nil {
		var v Value
		var p = l.p.pos

		switch l.p.tok {
		case PCON,RBRACE,RPAREN,COMMA,SELECT_PROP,SELECT_PROG1,SELECT_PROG2,SPACE,LINEND,EOF:
			return
		case SAST, DAST, ASTQ, QUE:
			v = l.globmeta(ctx) // * ** *? ?
		case LBRACK:
			v = l.globrange(ctx) // [abc0-9xyz]
		case DOT:
			v = l.punct(ctx)
		default:
			v = l.unary(ctx)
		}

		if l.p.pos == p { erro(ctx, "syntax error") }

		g.elems = append(g.elems, v)
	}
	return
}

func is_perc_term(t token) (_ bool) {
	switch t {
	case COMMA,COLON,DOLON,LPAREN,RPAREN,LBRACK,RBRACK,LBRACE,PCON,SEMICOLON,SPACE,LINEND:
		return true
	}
	return
}

func (l ul) perc(ctx Context, x Value) Value {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "perc")) }

	ctx = p_perc{ctx}

	var y Value
	var pos = l.p.pos

	l.p.step(ctx)

	if pos+1 == l.p.pos && !is_perc_term(l.p.tok) { // joint, e.g. '%.o', but skip '% .o'
		switch l.p.tok {
		case PERC: // %%
			l.p.step(ctx) // consume the second %
			perc := makePercpat(l.p.pos, nil, nil)
			if pos+2 == l.p.pos && !is_perc_term(l.p.tok) {
				switch l.p.tok {
				case PERC: // %%%
					erro(ctx, "too many %")
				default:
					switch perc.Suffix = l.expr(ctx); perc.Suffix.(type) {
					case *argumented, *path:
						erro(ctx, "incorrect: %v %v", x, ts(perc.Suffix))
					}
				}
			}
			y = perc
		default:
			y = l.unary(ctx)//expr(ctx)
		}
	}
	return makePercpat(pos, x, y)
}

type regex_subexp_auto struct{ *regexp.Regexp }

func (l ul) regex(ctx Context) (_ Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "regex")) }

	var rx string
	var pos = l.p.pos

	ctx = p_regex{ctx}

	if !l.p.scanner.bits.isBrace() {
		erro(ctx, "wrong scan state: %v", &l.p.scanner.scanstate)
	}
	if !l.p.scanner.bits.isBraceRaw() {
		erro(ctx, "wrong scan state: %v", &l.p.scanner.scanstate)
	}

rxloop:
	for ; l.p.tok != RBRACE && l.p.tok != EOF; l.p.scan(ctx) {
		if l.p.tok == ESCAPE { rx += "\\" }
		switch l.p.tok {
		case CLOSURE, DELEGATE:
			if v := l.calling(ctx); v != nil {
				rx += __string(ctx, v)
				if l.p.tok == RBRACE { break rxloop }
			} else {
				debug(pc(ctx,l.p), "bad closure: %v", l.p.tok)
			}
		}
		// CRITICAL FIX: Check the sym channel first!
		if l.p.sym != symEmpty {
			rx += l.p.sym.String()
		} else if l.p.lit != "" {
			rx += l.p.lit
		} else {
			// This safely handles punctuation like '<', '[', '?'
			// because your tokens array defines them as literal strings!
			rx += l.p.tok.String()
		}
	}

	l.p.expect(ctx, RBRACE)

	var err error
	var x = &regexpat{valbase{pos}, nil} // TODO: correct regexp pattern value
	if x.Regexp, err = regexp.Compile(rx); err != nil {
		erro(pc(ctx,l.p), "regex: %v", err)
	} else {
		do(ctx, regex_subexp_auto{x.Regexp})
	}
	return x
}

func (l ul) flag(ctx Context) flag {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "flag")) }

	l.p.step(ctx) // skip dash '-'

	// exclude "-)" "-]" "-}" "-\n", "-=", "-:", etc.
	if l.p.is_end_of_line() || l.p.is_list_term(ctx) || l.p.tok == SPACE || l.p.tok == RECIPE {
		return flag{&valbase{l.p.pos}}
	}

	var x = l.unary(ctx)

composeloop:
	for l.p.tok != EOF {
		p := l.p.pos
		switch l.p.tok {
		case DOT:
			x = l.dot(ctx, x)
		case CLOSURE, DELEGATE, MINUS:
			x = prefix(ctx, x, l.unary(ctx))
		case PCON:
			break composeloop //x = l.path(ctx, x)
		default:
			if l.p.tok.is_closure() || l.p.tok.is_delegate() {
				x = prefix(ctx, x, l.unary(ctx))
			} else {
				break composeloop
			}
		}
		if l.p.pos == p { erro(ctx, "syntax error") }
	}

	if x == nil {
		erro(ctx, "nil flag name")
	}
	return flag{x}
}

func (l ul) negative(ctx Context) negative {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "negative")) }
	l.p.expect(ctx, EXC)
	return negative{l.expr(ctx)}
}

func (l ul) punct(ctx Context) *punct {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "punctuation")) }
	p := &punct{valbase{l.p.pos}, l.p.tok}
	l.p.step(ctx)
	return p
}

func (l ul) escape(ctx Context) *escaped {
	v := &escaped{valbase{l.p.pos}, l.p.lit}
	l.p.expect(ctx, ESCAPE)
	return v
}

func (l ul) literal(ctx Context) (_ Value) {
	tok, sym, lit, pos := l.p.tok, l.p.sym, l.p.lit, l.p.pos

	l.p.step(ctx)

	// ESCAPE is handled in value.EscapeChar
	switch tok {
	case BAR: erro(ctx, "`|` is deprecated, change the modifiers!")
	case BINARY:      return ParseBinary(pos, lit)
	case OCTAL:       return ParseOctal(pos, lit)
	case INTEGER:     return ParseDecimal(pos, lit)
	case HEXADECIMAL: return ParseHexadecimal(pos, lit)
	case DATETIME:    return ParseDateTime(pos, lit)
	case DATE:        return ParseDate(pos, lit)
	case TIME:        return ParseTime(pos, lit)
	case URL:         return ParseURL(pos, lit)
	case FLOATING:    return parseFloat(pos, lit)
	case WORD:        return _word(pos, sym)
	case RAW:         return _raw(pos, lit)
	case STRING:      return _strlit(pos, lit)
	}

	unreachable()
	return
}

func (l ul) strcomp(ctx Context) *strcomp {
	var elems []Value

	l.p.step(ctx)

	for l.p.tok != EOF && l.p.tok != COMPOSED && l.p.tok != LINEND {
		var p = l.p.pos
		if l.p.tok == RAW {
			elems = append(elems, l.literal(ctx))
		} else {
			elems = append(elems, l.expr(p_strcomp{ctx}))
		}
		if l.p.pos == p { erro(ctx, "syntax error") }
	}

	l.p.expect(ctx, COMPOSED)

	return _strcomp(elems...)
}

func (p *parser) is_dot_term(ctx Context) bool {
	// Expressions like `FOO.BAR(xxx)` does not count.
	switch p.tok {
	case SPACE, LPAREN, COLON, PCON, ASSIGN: fallthrough
	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2:
		if true || truly(ctx, can_select{}) { return true }
	}
	return p.is_end_of_line() || p.is_list_term(ctx)
}

// Parses dot composing expressions (e.g., .foo, foo.bar.baz, 1.10.1)
func (l ul) dot(ctx Context, x Value) (_ Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "dot")) }

	var q *qualword
	if t, ok := x.(*qualword); ok {
		q = t
	} else {
		q = &qualword{}
		if x == nil {
			// Leading dot (e.g., .foo): append an empty raw string to preserve semantic structure
			q.elems = append(q.elems, &valbase{l.p.pos})
		} else {
			q.elems = append(q.elems, x)
		}
	}

	ctx = aware(ctx, DOT)
	l.p.step(ctx) // skips the initial '.'

	var trailingDot = true

	for !l.p.is_dot_term(ctx) {
		p := l.p.pos
		y := l.unary(ctx)

		if l.p.pos == p { erro(ctx, "syntax error") }

		// Flatten nested qualwords if they emerge from evaluation
		if subQ, ok := y.(*qualword); ok {
			q.elems = append(q.elems, subQ.elems...)
		} else if y != nil {
			q.elems = append(q.elems, y)
		}

		if l.p.tok == DOT {
			trailingDot = true
			l.p.step(ctx) // skips '.'
		} else {
			trailingDot = false
			break
		}
	}

	if trailingDot {
		// Trailing dot (e.g., foo.): append an empty raw string
		q.elems = append(q.elems, &valbase{l.p.pos})
	}

	return q
}

func (l ul) path(ctx Context, start Value) (res *path) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "path")) }
	if start == nil { erro(ctx, "nil path starter") }

	ctx = p_path{ctx}

	switch t := start.(type) {
	case *path: res = t
	case *strlit:
		res = makePath(splitPathStr(pc(ctx,t.pos), t.s)...)
	case *strcomp:
		res = makePath(splitPathStr(pc(ctx,t.Pos()), __string(ctx, t))...) // FIXME: dont final here
	default:
		res = makePath(start)
	}

	for l.p.tok == PCON {
		var pos = l.p.pos
		for l.p.step(ctx); l.p.tok == PCON; l.p.step(ctx) {} // repeated '/'

		switch l.p.tok {
		case LPAREN, LBRACE, RPAREN, RBRACE, COMMA, SPACE, LINEND:
			res.elems = append(res.elems, &punct{valbase{pos}, PTAIL}) // after the last '/'
			return
		}

		pos = l.p.pos

		var elem = l.unary(ctx)
		if l.p.pos == pos { erro(ctx, "syntax error", trace{}) }
		if false { if x, y := elem.(*list); y && x.len() == 1 { elem = x.elems[0] } }

		switch l.p.tok {
		case DOT: // .
			elem = l.dot(ctx, elem)
		case SAST, DAST, ASTQ, QUE, LBRACK: // * ** ? [
			elem = l.glob(ctx, elem)
		case PERC: // %
			elem = l.perc(ctx, elem)
		}

		if x, y := elem.(*path); y {
			res.elems = append(res.elems, x.elems...)
		} else {
			res.elems = append(res.elems, elem)
		}

		if l.p.tok == SPACE || l.p.is_end_of_line() { return }
	}
	return
}

var schemeNames = []string{"file", "http", "https", "ftp", "ftps", "mailto"}
var schemes = []Symbol{symFile, symHttp, symHttps, symFtp, symFtps, symMailto}

func isKnownScheme(s Symbol) bool {
	// 2. FAST PATH: Zero-allocation integer comparison.
	// (A slice of 6 integers fits neatly in a single CPU cache line,
	// making this loop actually faster than a map lookup!)
	for _, scheme := range schemes {
		if scheme == s { return true }
	}

	// 3. SLOW PATH: Case-insensitive fallback
	// Cross the boundary to string, but ONLY if we absolutely have to.
	str := s.String()
	lower := strings.ToLower(str)

	// OPTIMIZATION: If the string was already entirely lowercase,
	// we know it didn't match the fast-path, so it's definitely not a scheme.
	// This saves a useless loop execution for normal unknown words (like "compile").
	if str == lower {
		return false
	}

	for _, name := range schemeNames {
		if name == lower { return true }
	}

	return false
}

type is_url          struct{}
type is_url_query    struct{}
type is_url_fragment struct{}
type url_encoding struct{ Context }
type url_query    struct{ Context }
type url_fragment struct{ Context }

func (u url_encoding) cast(t reflect.Type) Context { return icast(u,t) }
func (u url_encoding) inner() Context { return u.Context }
func (u url_encoding) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case is_url: return true
	}
	return u.Context.do(ctx, op)
}

func (u url_query) cast(t reflect.Type) Context { return icast(u,t) }
func (u url_query) inner() Context { return u.Context }
func (u url_query) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case is_url_query: return true
	}
	return u.Context.do(ctx, op)
}

func (u url_fragment) cast(t reflect.Type) Context { return icast(u,t) }
func (u url_fragment) inner() Context { return u.Context }
func (u url_fragment) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case is_url_fragment: return true
	}
	return u.Context.do(ctx, op)
}

func (l ul) url(ctx Context, scheme Value) (res Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "url")) }

	var u = &url{Scheme:scheme}

	l.p.expect(ctx, COLON) // consumes ':'
	l.p.expect(ctx, PCON) // the first '/'
	l.p.expect(ctx, PCON) // the second '/'

	ctx = url_encoding{ctx}

	var x = l.unary(ctx)

	switch l.p.tok {
	case AT:
		u.Username = x
		x = l.unary(ctx)
	case COLON:
		u.Username = x
		l.p.step(ctx) // ':'
		u.Password = l.unary(ctx)
		l.p.expect(ctx, AT)
		x = l.unary(ctx)
	case LINEND, EOF:
		u.Host = x
		return u
	}

	var h *qualword
	var p *path

hostloop:
	for l.p.tok == DOT {
		if h == nil {
			h = &qualword{}
			if q, ok := x.(*qualword); ok {
				h.elems = append(h.elems, q.elems...)
			} else {
				h.elems = append(h.elems, x)
			}
		}

		l.p.step(ctx) // '.'

		switch x = l.unary(ctx); l.p.tok {
		case PCON, QUE, HASH:
			h.elems = append(h.elems, x)
			break hostloop
		case LINEND, EOF:
			h.elems = append(h.elems, x)
			u.Host = h
			return u
		default:
			h.elems = append(h.elems, x)
		}
	}

	if h == nil {
		u.Host = x
	} else {
		u.Host = h
	}

pathloop:
	for l.p.tok == PCON {
		if p == nil { p = makePath(&punct{valbase{l.p.pos}, PROOT}) }

		l.p.step(ctx) // '/'

		switch l.p.tok {
		case QUE, HASH:
			break pathloop
		default:
			x = l.unary(ctx)
			p.elems = append(p.elems, x)
		}
	}
	if p != nil {
		u.Path = p
	}

	// scan '#' differently to comments
	defer l.p.scanner.setBits(l.p.scanner.commentsOff())

	if l.p.tok == QUE {
		l.p.step(ctx) // '?'

	queryloop:
		for {
			switch l.p.tok {
			case HASH, LINEND, EOF:
				break queryloop
			}

			x = l.unary(url_query{ctx})

			if l.p.tok == ASSIGN {
				l.p.step(ctx) // '='
				x = &pair{ x, l.unary(url_query{ctx}) }
			}

			u.Query = append(u.Query, x)

			if l.p.tok == CLOSURE {
				l.p.step(ctx) // '&'
			} else if l.p.tok == PERC {
				erro(ctx, "unexpected %v in url", l.p.tok)
			}
		}
	}

	if l.p.tok == HASH {
		l.p.step(ctx) // '#'
		u.Fragment = l.unary(url_fragment{ctx})
	}

	return u
}

func (l ul) check(ctx Context, str string) bool { return __true(ctx, l.resolve(ctx, nil, intern(str))) }
func (l ul) promptCachedConfigs(ctx Context) bool { return l.check(ctx, "prompt-cached-configs") }
func (l ul) promptConfigurationLoads(ctx Context) bool { return l.check(ctx, "prompt-configuration-loads") }

func (l ul) resolve(ctx Context, name Value, sym Symbol) (result Value) { // CRITICAL FIX: sym Symbol
	var pos Pos
	if name != nil { pos = name.Pos() }
	if !pos.IsValid() { pos = l.p.pos }
	if !pos.IsValid() { pos = _pos(ctx) }

	if sym == symEmpty {
		erro(ctx, "resolve no-name : %v", ts(name, ctx))
	}

	if d := auto_find(ctx, sym); d != nil { // auto_find upgraded to accept Symbol
		return d
	}

	var o object
	var s = _scope(ctx)

	if name != nil { defer func() {
		switch x := o.(type) {
		case *builtin:
			t := *x
			t.pos = name.Pos()
			result = &t
		}
	}()}

	if l.project == nil || s != l.project.scope {
		if _, o = s.find(sym); o != nil { return o } // s.find upgraded
	}
	if l.project != nil {
		if o = l.project.resolve(ctx, sym); o != nil { return o } // project.resolve upgraded
	}

	// CRITICAL FIX: rule_autos map must be updated to map[Symbol]...
	_, isRuleAuto := rule_autos[sym]

	switch {
	// sym.String() is cheap, but you could also do IsDigits directly on the Symbol ID
	// if you pre-intern digits, but String() is perfectly fine here.
	case isRuleAuto || IsDigits(sym.String()) || truly(ctx, is_auto{sym}):
		// ZERO-ALLOCATION FIX!
		return &auto{knownobject{objbase{valbase{pos}, s}, sym}}
	case truly(ctx, is_config_mode{}):
		return s.def(ctx, defVoid, sym) // defVoid logic upgraded to Symbol
	}

	if l.project != nil {
		if c := l.project.configure; c != nil {
			return c.resolve(ctx, sym)
		}
	}
	return
}

// Upgraded return signature: sym Symbol
func (l ul) identity(ctx Context, tok token, name Value) (obj Value, sym Symbol, opts []Value) {
	var ic *ident_ctx

	ic, ctx = identity_ctx(ctx)

	if ic.nil > 0 {
		obj = name
		return
	}

	switch x := name.(type) {
	case object:
		// 1. Fast-path for named objects (like *def, *project, *file)
		obj, sym = x, __symbol(ctx, x)
		return

	case *argumented:
		obj, sym, opts = l.identity(ctx, tok, x.Value)
		opts = append(opts, merge(x.args...)...)
		return

	case *word:
		// 2. Fast-path for bare words (which are Values, not objects)
		sym = x.s

	default:
		// 3. Slow-path fallback for dynamically evaluated nodes
		sym = intern(ident(ctx, name))
	}
	if sym == symEmpty {
		if truly(ctx, opt_ident{}) {
			obj = name
			return
		}

		erro(pc(ctx,name), "empty ident: %v (nil=%d) : %s", name, ic.nil, ts(name,ctx))
	}

	switch tok {
	case LPAREN:
		if obj = l.resolve(ctx, name, sym); obj != nil {
			return
		} else if truly(ctx, opt_ident{}) {
			obj = name
			return
		}
	case LBRACE:
		if e := l.project.entry(ctx, name); e == nil {
			erro(pc(ctx,name), "resolved nil: %s", ts(name,ctx))
		} else if _, ok := e.(object); !ok {
			erro(pc(ctx,name), "not an object: %v: %s", name, ts(e,ctx))
		} else {
			obj = name
			return
		}
	}

	erro(pc(ctx,name), "undefined %v → %v : %s", name, sym, ts(name,ctx), callstack{num:32})
	return
}

func (l ul) calling(ctx Context) (result Value) {
	var tok token
	var sym Symbol // CRITICAL FIX: Upgraded to Symbol
	var name, obj Value
	var args, opts []Value
	var pos = l.p.pos
	var closure = l.p.tok.is_closure()

	// Suspend string modes so bare variables (like $1, $@) scan normally
	suspended := l.p.scanner.bits & (isStrcompLine | isStrcompString | isBracedPlain | isBraceRaw)
	if suspended != 0 {
		l.p.scanner.bits &^= suspended
	}

	l.p.step(ctx) // skips $ or &

	// Restore string modes after the variable's leading token has been safely consumed
	if suspended != 0 {
		l.p.scanner.bits |= suspended
	}

	switch l.p.tok {
	case LPAREN, LBRACE: // $(...), ${...}
		tok = l.p.tok // use LPAREN, LBRACE
		l.p.step(ctx) // skips LPAREN, LBRACE

		if l.p.tok == SPACE { erro(pc(ctx,l.p.pos), "unexpected spaces") }

		name = l.expr(selection{ctx})

		if closure || optional(name) {
			obj, sym, opts = l.identity(optional_ident(ctx), tok, name)
		} else {
			obj, sym, opts = l.identity(ctx, tok, name)
		}

		if (tok == LPAREN && l.p.tok != RPAREN) || (tok == LBRACE && l.p.tok != RBRACE) {
			var cc Context = aware(ctx, COMMA)

			// ZERO-ALLOCATION FIX: This is now a pure O(1) integer switch!
			switch sym {
			case symEmpty:
				l.p.spaces(cc)
				args = append(args, _list(l.values(cc)...))
			case symAuto:
				if !closure { cc = p_auto_ctx{cc} }
				args = append(args, _list(l.values(cc)...))
			case intern("and"), intern("or"): // Or use predefined constants if available
				cc = optional_ident(cc)
				args = append(args, _list(l.values(cc)...))
			case intern("case"):
				args = append(args, _list(l.values(cc)...))
				cc = optional_ident(cc)
			case symForeach:
				a := &auto{knownobject{objbase{valbase{l.p.pos}, l.scope()}, symUnderscore}}
				args = append(args, _list(l.values(cc)...))
				cc = &foreach_txt{cc, a}
			case symGrep:
				cc = &grep_txt{cc, objbase{valbase{l.p.pos}, l.scope()}, nil}
				args = append(args, _list(l.values(cc)...))
			default:
				args = append(args, _list(l.values(cc)...))
			}

			for l.p.tok == COMMA {
				l.p.next(cc, true) // consumes comma
				args = append(args, _list(l.values(cc)...))
			}
		}

		switch tok {
		case LPAREN: l.p.expect(ctx, RPAREN)
		case LBRACE: l.p.expect(ctx, RBRACE)
		}

	case BINARY, OCTAL, INTEGER, HEXADECIMAL, FLOATING:
		tok = l.p.tok
		name = l.literal(ctx) // $0, $1, $1.2, $0x1...
		sym = __symbol(ctx, name)
		obj = l.resolve(ctx, name, sym)

	case STRING, STRCOMP:
		tok = l.p.tok
		if tok == STRING { name = l.literal(ctx) } else { name = l.strcomp(ctx) }
		obj, sym, opts = l.identity(ctx, tok, name)

	case WORD:
		switch l.p.sym { // CRITICAL FIX: Read the scanner's high-speed integer channel!
		case symUnderscore: // "_"
			tok, sym = UNDERLINE, symUnderscore
			name = &punct{valbase{l.p.pos}, tok}
			obj = l.resolve(ctx, name, sym)
			l.p.step(ctx)
		default:
			erro(pc(ctx,l.p.pos), _f("unexpects %v", l.p.sym))
		}

	default: // case AT, BAR, DOT, SAST, QUE, MINUS, PLUS, PCON:
		tok = l.p.tok

		// Fallback guard: In the rare case a RAW token still leaks through, intercept it.
		if tok == RAW {
			name = _raw(l.p.pos, l.p.lit)
			sym = intern(l.p.lit)
			l.p.step(ctx)
		} else {
			name = l.punct(ctx) // $@, $?, $*, $/...
			if w, ok := name.(*word); ok {
				sym = w.s
			} else {
				sym = intern(tok.String())
			}
		}

		if obj = l.resolve(ctx, name, sym); obj == nil {
			debug(pc(ctx, name.Pos()),
				_f("unexpected: tok=%v sym=%v name=%v dialect=%v", tok, sym, name, l.p.dialect),
				trace{})
		}
	}

	if obj == nil && sym != symEmpty {
		if l.project.ext.Plugin != nil {
			// External plugin interface might still expect a string, so we call .String() safely here.
			if t, e := l.project.ext.Lookup(sym.String()); e == nil && t != nil {
				debug(pc(ctx, name.Pos()),
					_f("unexpected: tok=%v sym=%v name=%v dialect=%v", tok, sym, name, l.p.dialect),
					trace{})
			}
		}
	}

	if obj == nil {
		debug(pc(ctx, name.Pos()),
			_f("nil symbol; tok=%v sym=%v name=%v", tok, sym, name),
			callstack{num:32}, trace{})
	}

	if closure {
		return makeClosure(pos, tok, obj, opts, args...)
	}

	if x, y := obj.(*def); y && x.o == defStatic {
		if !truly(ctx, is_auto_preserved{x.name}) {
			return _loc(x.value, name.Pos())
		}
	}
	return makeDelegate(pos, tok, obj, opts, args...)
}

func (l ul) unary(ctx Context) (x Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "unary")) }

	// ---> DEFENSIVE PRIMER FIX <---
	// If the parser hits unary and the token is still 0 (uninitialized),
	// force the scanner to read the first token.
	if l.p.tok == ILLEGAL && l.p.lit == "" {
		l.p.next(ctx, true)
		if l.p.tok == EOF {
			return nil
		}
	}

	pos := l.p.pos

	defer func() {
		if pos == l.p.pos && x != nil {
			if z, y := x.(*punct); !y || z.token != PROOT {
				erro(pc(ctx,x), "syntax error: %v", ts(x,ctx))
			}
		}
	} ()

	switch l.p.tok {
	case ASSIGN: // example: '=xxx'
		if !truly(ctx, left_hand_side{}) {
			var x = &valbase{l.p.pos}
			if l.p.step(ctx); l.p.is_list_term(ctx) {
				return &pair{x, &valbase{l.p.pos}}
			} else {
				return &pair{x, l.expr(ctx)}
			}
		}

	case WORD:
		if x = l.p.bare(ctx) ; l.p.tok == PERC && truly(ctx, is_url_query{}) {
			var comp = _compound(x)
			for l.p.tok == PERC {
				comp.elems = append(comp.elems, l.punct(ctx))

				// See https://en.wikipedia.org/wiki/Query_string#URL_encoding
				// It should be decoded as '%HH' here, but we just treat it as a literal.
			urlpercloop:
				for l.p.tok != PERC {
					switch l.p.tok {
					case BINARY, OCTAL, INTEGER, HEXADECIMAL, WORD:
						comp.elems = append(comp.elems, l.literal(ctx))
					case HASH:
						break urlpercloop
					default:
						erro(ctx, "bad url token: %v %v", l.p.tok, l.p.lit)
					}
				}
			}
			x = comp
		}
		return

	case BINARY, OCTAL, INTEGER, HEXADECIMAL, FLOATING, DATETIME, DATE, TIME, URL, STRING/*, RAW*/:
		return l.literal(ctx)

	case STRCOMP:
		return l.strcomp(ctx)

	case ESCAPE: // \
		return l.escape(ctx)

	case LPAREN: // (
		return l.group(ctx)

	case LBRACE: // {
		return l.braced(ctx)

	case Lbot_corner, Ltop_corner: // ⌜a b c⌟  ⌞a b c⌝
		return l.corner_list(ctx)

	case LANGLE, RANGLE: // < >
		return l.punct(ctx)

	case CLOSURE, DELEGATE:
		return l.calling(ctx)

 	case COMMA:
		if !truly(ctx, aware_token{COMMA}) {
			return l.punct(ctx)
		}

	case AT, BAR, PLUS, SEMICOLON:
		return l.punct(ctx)

	case PERC: // %bar (no prefix)
		if truly(ctx, is_url_query{}) {
			return l.punct(ctx)
		} else {
			return l.perc(ctx, nil)
		}

	case SAST, DAST, ASTQ, QUE, LBRACK: // * ** *? ? [    -- NOTE: ? is an exception
		return l.glob(ctx, nil)

	case MINUS:
		return l.flag(ctx)

	case EXC:
		return l.negative(ctx)

	case PCON: // The root of the path
		return &punct{valbase{l.p.pos}, PROOT}

	case TILDE: // ~
		return l.punct(ctx)

	case DOT: // .
		return l.dot(ctx, nil)

	case DOTDOT: // . ..
		tok, pos := l.p.tok, l.p.pos
		if l.p.step(ctx) ; l.p.tok == PCON {
			return &punct{valbase{l.p.pos}, tok}
		} else {
			return &punct{valbase{pos}, tok}
		}

	default:
		if l.p.tok.is_keyword() { // keywords here are words
			return l.p.bare(ctx)
		}
	}

	if l.p.tok != EOF {
		erro(pc(ctx, /* l.p.loc */(pos)),
			_f("unexpected {tok=%v sym=%s lit=%s}", l.p.tok, l.p.sym, l.p.lit),
			_f("%v", &l.p.scanner.scanstate),
			callstack{num:16}, trace{})
	}

	if l.p.lineComment != nil {
		for _, c := range l.p.lineComment.comments {
			erro(ctx, "# %s", c.string)
		}
	}
	return
}

func (l ul) composite(ctx Context) (x Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "composite")) }

	x = l.unary(ctx)

	switch l.p.tok { // check composible expressions
	case PERC: // foo%bar ; FIXME: %/foo/bar -> {=path % foo bar}
		// Block punctuation from becoming the prefix of a pattern
		if _, isPunct := x.(*punct); isPunct {
			return x // Fall back to expr() composeloop!
		}
		return l.perc(ctx, x)

	case DOT: // foo.bar.baz.o ; FIXME: push bits when parsing $(...)
		return l.dot(ctx, x)

	case QUE: // ?
		return l.glob(ctx, x)

	case COLON:
		if truly(ctx, is_recipe{false}) || !truly(ctx, left_hand_side{}) {
			if w, ok := x.(*word); ok && isKnownScheme(w.s) {
				return l.url(ctx, x)
			}
		}
	}
	return
}

func (l ul) expr(ctx Context) (x Value) {
	if false && l_traverse.enabled { defer un(l_trace(l_traverse, "expr")) }
	if false { defer func() { debug(pc(ctx,l.p), "%s", ts(x,ctx)) } () }

	x = l.composite(ctx)

	if truly(ctx, left_hand_side{}) && l.p.tok.is_assign() { return }
	if truly(ctx, p_is_glob{}) { return }
	if truly(ctx, p_is_params{}) {
		if g, y := x.(*group); y && g.len() == 1 {
			if _, y = g.elems[0].(*group); y { return }
		}
	}

	var n int

composeloop: // parses right-fixes
	switch l.p.tok {
	case COLON, COMPOSED, RPAREN, RBRACK, RBRACE, Rbot_corner, Rtop_corner, Rchevron, RAW, SPACE, SEMICOLON, LINEND, EOF:
		return

	case COMMA:
		if truly(ctx, aware_token{COMMA}) { return }

	case ASSIGN: // example: key=value
		if !truly(ctx, left_hand_side{}) {
			if l.p.step(ctx); l.p.is_list_term(ctx) {
				return &pair{x, &valbase{l.p.pos}}
			} else {
				return &pair{x, l.expr(ctx)}
			}
		}

	case LPAREN:
		if !truly(ctx, p_no_argumented{}) {
			x = l.argumented(ctx, x)
			goto composeloop
		}

	case PCON: // path, except -I/path/to/include
		if !truly(ctx, p_no_path{}) {
			x = l.path(ctx, x)
			goto composeloop
		}

	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2:
		if truly(ctx, can_select{}) {
			x = l.arrow(ctx, x)
			goto composeloop
		}

	case QUE: // ?
		x = l.glob(ctx, x)
		goto composeloop
	}

	if truly(ctx, p_is_strcomp{}) { return }

	var p = l.p.pos
	var y = l.unary(ctx)// NOTE: it's unary, not composite
	if l.p.pos == p {
		erro(ctx, "syntax error: %v (%v %v %v)", x, ts(x,ctx), ts(y,ctx), l.p.tok)
	}

	x = prefix(ctx, x, y) // ⇒ xy

	switch l.p.tok { case COMMENT, SPACE, LINEND, EOF: return }

	if 9999 < n { erro(ctx, "too many compose: %v (%d)", x, n) }

	n += 1; goto composeloop // compose as many as possible
}

func (l ul) braced_and(ctx Context) (res Value) {
	var vs []Value

	l.p.expect(ctx, AND) // consumes `and`

andloop:
	for {
		switch l.p.spaces(ctx); l.p.tok {
		case COMMA: l.p.step(ctx); continue andloop
		case RBRACE, EOF: break andloop
		}

		v := l.expr(aware(ctx,COMMA))
		vs = append(vs, merge(expand(_final(pc(ctx, v)), v))...)
	}

	l.p.expect(ctx, RBRACE)

	for _, a := range vs { if __true(ctx, a) { res = a } else { return nil } }
	return
}

func (l ul) braced_or(ctx Context) (_ Value) {
	l.p.expect(ctx, OR) // consumes `or`

	var va []Value

orloop:
	for l.p.tok != EOF {
		switch l.p.spaces(ctx); l.p.tok {
		case  COMMA: l.p.step(ctx); continue orloop
		case RBRACE:                   break orloop
		}

		v := l.expr(aware(ctx,COMMA))
		w := expand(_final(pc(ctx, v)),v)
		va = append(va, merge(w)...)
	}

	l.p.expect(ctx, RBRACE)

	for _, a := range va {
		if __true(ctx, a) { return a }
	}
	return
}

func (l ul) braced_for(ctx Context) (res Value) {
	l.p.expect(ctx, FOR) // consumes `for`

	erro(ctx, "TODO: {=for ...}")

	l.p.expect(ctx, RBRACE)
	return
}

type foreach_text struct{ Context }
func (f foreach_text) inner() Context { return f.Context }
func (f foreach_text) cast(t reflect.Type) Context { return icast(f, t) }
func (f foreach_text) do(c Context, o any) (_ any) {
    switch t := o.(type) {
    case find_auto:
		if t.s == symUnderscore {
			var a *automatic
			switch t := f.Context.(type) {
			case *automatic: a = t
			case *__foreach: a = &t.automatic
			}
			if a != nil {
				if _, y := a.defs[t.s]; !y {
					return
				}
			}
		}
    }
	return f.Context.do(c, o)
}

func (l ul) braced_foreach(ctx Context) Value {
	pos := l.p.pos
	l.p.expect(ctx, FOREACH)
	l.p.spaces(ctx)

	var vals []Value

valsloop:
	for {
		switch l.p.tok {
		case COMMA, RBRACE, LINEND, END:
			break valsloop
		}

		ac := aware(ctx,COMMA)
		vals = append(vals, l.expr(ac))
		l.p.spaces(ac)
	}

	cc := automatic{ Context:ctx, defs:make(def_map) }
	cc.set(ctx, defVoid, symUnderscore, nil)

	var temps []Value
	switch l.p.spaces(ctx); l.p.tok {
	case RBRACE: return _null(l.p.pos)
	case COMMA:
		for l.p.step(ctx); l.p.tok != RBRACE; {
			l.p.spaces(ctx)
			if v := l.expr(foreach_text{&cc}); v != nil {
				temps = append(temps, v)
			} else {
				erro(ctx, "nil ; %v", l.p.tok)
			}
		}
	}

	l.p.expect(ctx, RBRACE)

	var va []Value
	var recipe_start = truly(ctx, is_recipe_start{})
	for _, val := range vals {
		for _, elem := range merge(expand(_final(ctx),val)) {
			if isEmpty(elem) { continue }
			if /* indeterminate(ctx, elem) */true { elem = &disjunction{valbase{pos},elem} }

			// NOTE: don't use defStatic, it's for codeblock auto only
			cc.set(ctx, defVoid, symUnderscore, elem)

			var a = xmerge(_final(foreach_text{&cc}), temps...)
			if recipe_start && l.p.tok == LINEND {
				do(ctx, add_recipe_line{a})
			} else {
				va = append(va, a...)
			}
		}
	}
	return ease(ctx, va)
}

func (l ul) braced_str(ctx Context) (res Value) {
	l.p.next(ctx, true) // resumes 'str'

	var pos = l.p.pos
	var elems = expands(_final(ctx), l.braced_elems(ctx)...)

	if /* indeterminate(ctx, elems...) */true {
		return &strval{valbase{pos}, elems}
	}

	var s string
	for i, v := range elems {
		if s != "" && 0 < i { s += " " }
		s += __string(ctx, v)
	}
	return &strlit{valbase{pos}, s}
}

func (l ul) braced_word(ctx Context) (res Value) {
	l.p.next(ctx, true) // resumes 'word'

	var pos = l.p.pos
	var elems = expands(_final(ctx), l.braced_elems(ctx)...)

	// PERFORMANCE FIX: Use strings.Builder instead of `s += " "`
	var b strings.Builder
	var first = true

	for _, v := range elems {
		str := __string(ctx, v)
		if str == "" {
			continue // Skip empty evaluations to prevent double-spaces
		}
		if !first {
			b.WriteByte(' ')
		}
		b.WriteString(str)
		first = false
	}

	// CRITICAL FIX: The final string is built. Lock it into the Symbol
	// vocabulary pool and assign the integer ID to the *word!
	return &word{valbase{pos}, intern(b.String())}
}

func (l ul) braced_quote(ctx Context) (res Value) {
	l.p.next(ctx, true) // resumes 'quote'
    return &quote{list{elements{l.braced_elems(ctx)}}}
}

type defcap struct {
	name  Symbol
	value Value
}

type defcaps struct {
	Value
	caps []*defcap
}

func (dc *defcaps) String() string {
	// Use strings.Builder to prevent heap-thrashing
	var b strings.Builder

	b.WriteString("{=defcaps ")
	if dc.Value != nil {
		b.WriteString(dc.Value.String())
	}

	for _, cap := range dc.caps {
		b.WriteString(" {")
		// Extract the native string from the Symbol instantly
		b.WriteString(cap.name.String())
		b.WriteString(":")
		if cap.value != nil {
			b.WriteString(cap.value.String())
		}
		b.WriteString("}")
	}

	b.WriteString("}")
	return b.String()
}

func (l ul) braced_defs(ctx Context) (res Value) {
	// 1. The capture list to our fast integer channel
	var capture []Symbol

	l.p.step(ctx) // resumes 'defs'

	if l.p.tok == LPAREN {
		l.p.next(ctx, true) // resumes '('
		for l.p.tok != RPAREN && l.p.tok != EOF {
			switch l.p.tok {
			case COMMA:
			case INTEGER, WORD:
				// CRITICAL FIX 1: Safely handle INTEGER tokens!
				// If the scanner didn't populate sym, intern it from the literal.
				sym := l.p.sym
				if sym == symEmpty {
					sym = intern(l.p.lit)
				}
				capture = append(capture, sym)
			default:
				erro(pc(ctx,l.p), "unexpected %v '%s'", l.p.tok, l.p.lit)
			}
			l.p.next(ctx, true)
		}
		l.p.expect(ctx, RPAREN)
	}

	l.p.spaces(ctx)

	var pats = expands(_final(ctx), l.braced_elems(ctx)...)
	var ac = _automatic(ctx)
	var vals []Value

defsloop:
	for _k, _ := range l.project.elems {
		for _, pat := range pats {
			var pos = pat.Pos()
			var name = _word(pos, _k)
			var neg bool
			if x, y := pat.(negative); y { pat, neg = x.Value, y }

			a, _, _, c := match(pc(ctx, pat), pat, name)
			if a && neg { continue defsloop }
			if a || neg {
				var main Value = name
				var caps []*defcap

				// Use the exact predefined integer for $0
				caps = append(caps, &defcap{sym_0, main})

				if len(capture) == 0 {
					// Auto-numbered captures from stems: $1, $2...
					for i, stem := range c {
						var sSym Symbol
						// ZERO-ALLOCATION MATH: sym_0 to sym_9 are sequential iotas!
						if i < 9 {
							sSym = sym_1 + Symbol(i)
						} else {
							sSym = intern(strconv.Itoa(i + 1))
						}
						caps = append(caps, &defcap{sSym, stem})
					}
				} else {
					// Named captures
					if len(capture) == 1 {
						sym := capture[0]
						// Fast-path index extraction for numeric captures $1-$9
						if sym >= sym_1 && sym <= sym_9 {
							if idx := int(sym - sym_0); idx <= len(c) {
								main = c[idx-1]
							}
						} else if i, e := strconv.Atoi(sym.String()); e == nil && 0 < i && i <= len(c) {
							// Slow-path fallback for $10+
							main = c[i-1]
						}
					}

					// CRITICAL FIX 2: Removed the rogue 'else' block!
					// We must ALWAYS bind the variables to caps, even if len == 1.
					for i, sym := range capture {
						var val Value
						if i < len(c) {
							val = c[i]
						} else {
							val = &valbase{pos}
						}
						caps = append(caps, &defcap{sym, val})
					}
				}

				dc := &defcaps{main, caps}
				if ac != nil {
					for _, cap := range caps {
						ac.set(ctx, defStatic, cap.name, cap.value)
					}
				}
				vals = append(vals, dc)
				break // matched this name
			}
		}
	}
	return ease(ctx, vals)
}

func (l ul) braced_file(ctx Context) (res Value) {
	l.p.next(ctx, true)

	var elems []Value

	for _, elem := range l.braced_elems(ctx) {
		if f := l.project.file(pc(ctx,elem), elem); f != nil {
			elems = append(elems, f)
			continue
		} else {
			var s = __string(ctx, elem)
			var a = []any{stat_nonexist{true}}
			if !isAbsOrRel(s) {
				a = append(a, stat_dir{l.project.absPath})
			}
			if f = _stat(ctx, s, a...); f != nil {
				elems = append(elems, f)
				continue
			}
		}
		erro(pc(ctx,elem), "not a file: %v", ts(elem))
	}

	res = ease(ctx, elems)

	l.p.spaces(ctx)
	l.p.expect(ctx, RBRACE)
	return
}

func (l ul) braced_fullname(ctx Context) (res Value) {
	l.p.next(ctx, true) // resumes 'fullname'

	var elems []Value
	for _, elem := range l.braced_elems(ctx) {
		// CRITICAL FIX: Do not expand eagerly at parse time!
		// Preserve variables like $1 by just building the AST node.
		elems = append(elems, fullname{elem})
	}
	return ease(ctx, elems)
}

func (l ul) braced_plain(ctx Context) (elems []Value) {
	l.p.scanner.push(isBracedPlain)
	l.p.next(ctx, true) // aka LBRACE
	if l.p.tok == RAW { l.p.lit = trimLeftSpaces(l.p.lit) }
	elems = l.braced_elems(ctx)
	l.p.scanner.pop(isBracedPlain)
	return
}

func (l ul) braced_type(ctx Context, tok token) (x Value) {
	l.p.next(ctx, true)

	var n any
	var pos = l.p.pos
	if tok == BOOLEAN { tok = BOOL }
	if l.p.spaces(ctx); l.p.tok != RBRACE {
		switch tok {
		case ANSWER, BOOL:
			switch l.p.tok {
			case  TRUE, YES,  ON: n = true
			case FALSE,  NO, OFF: n = false
			default:
				erro(ctx, "unexpected token: %v", l.p.tok)
			}
			l.p.next(ctx, true)
		case FLOAT:
			n = __float(ctx, l.expr(ctx))
		default:
			n = __int(ctx, l.expr(ctx))
		}
	}

	switch tok {
	case ANSWER: x = _answer(pos,      n.(bool))
	case   BOOL: x = _boolean(pos,     n.(bool))
	case    BIN: x = _binary(pos,      n.(int64))
	case    OCT: x = _octal(pos,       n.(int64))
	case    INT: x = _decimal(pos,     n.(int64))
	case    HEX: x = _hexadecimal(pos, n.(int64))
	case  FLOAT: x = _float(pos,       n.(float64))
	}

	if x == nil {
		erro(pc(ctx,l.p), "nil const, %v, %v %v", tok, l.p.tok, l.p.lit)
	}

	l.p.spaces(ctx)
	l.p.expect(ctx, RBRACE)
	return
}

func (l ul) braced_const(ctx Context, tok token) (x Value) {
	l.p.next(ctx, true)

	var pos = l.p.pos

	if l.p.spaces(ctx); l.p.tok != RBRACE {
		erro(pc(ctx,l.p), "expecting right-brace, %v %v", l.p.tok, l.p.lit)
	}

	switch tok {
	case   YES: x = _answer(pos, true)
	case    NO: x = _answer(pos, false)
	case  TRUE: x = _boolean(pos, true)
	case FALSE: x = _boolean(pos, false)
	case    ON: x = _option(pos, true)
	case   OFF: x = _option(pos, false)
	}

	if x == nil {
		erro(pc(ctx,l.p), "nil const, %v, %v %v", tok, l.p.tok, l.p.lit)
	}

	l.p.spaces(ctx)
	l.p.expect(ctx, RBRACE)
	return
}

func (l ul) braced_none(ctx Context) (x Value) {
	l.p.next(ctx, true)

	var v Value
	var pos = l.p.pos
	for ; l.p.tok != RBRACE && l.p.tok != EOF; l.p.spaces(ctx) {
		if t := l.expr(ctx); v == nil {
			v = t
		} else if x, y := v.(*list); y {
			x.elems = append(x.elems, t)
		} else {
			v = &list{elements{[]Value{v, t}}}
		}
	}

	x = &none{valbase{pos}/*,v*/}
	l.p.spaces(ctx)
	l.p.expect(ctx, RBRACE)
	return
}

func (l ul) braced_null(ctx Context) (x Value) {
	l.p.next(ctx, true)

	x = &null{valbase{l.p.pos}}
	l.p.spaces(ctx)
	l.p.expect(ctx, RBRACE)
	return
}

func (l ul) braced_path(ctx Context) (x Value) {
	l.p.next(ctx, true)
	if v := l.expr(ctx); v != nil {
		if t, y := v.(*path); !y {
			x = l.path(ctx, v)
		} else {
			x = t
		}
	}
	l.p.spaces(ctx)
	l.p.expect(ctx, RBRACE)
	return
}

func (l ul) braced_project(ctx Context) (_ *project) {
	name := l.expr(ctx)
	sym := intern(__string(ctx, name))
	if sym == symEmpty {
		erro(ctx, "empty name : %s : %s", ts(name,ctx), sym)
	}

	if l.project.name == sym {
		return l.project
	} else if o := l.resolve(ctx, name, sym); o == nil {
		erro(pc(ctx,l.p), "%s : undefined %s : %v", l.project, sym, ts(name,ctx))
		return
	} else if x, y := o.(*project); !y && x != nil {
		erro(pc(ctx,l.p), "%s : %v is not a project", l.project, ts(o,ctx))
		return
	} else {
		return x
	}
}

// ----------------------------------------------------------------------------
// Clauses & Declarations

type clause_opts struct{
	general_opts

    keyword token // e.g. use, files, eval, etc.

    skip bool // e.g. -cond({=false}), -if({=no})

	conds []Value `if,cond,where`

    values, remainder, spec []Value // all values (unparsed) and remainder
}

type parseSpecFunc func(Context, *commentgroup, *clause_opts, int)

func isValidImport(lit string) bool {
	const illegalChars = `!"#$%&'()*,:;<=>?[\]^{|}` + "`\uFFFD"
	s, _ := strconv.Unquote(lit) // go/scanner returns a legal string literal
	for _, r := range s {
		if !unicode.IsGraphic(r) || unicode.IsSpace(r) || strings.ContainsRune(illegalChars, r) {
			return false
		}
	}
	return s != ""
}

func (p *parser) _parseUseSpecProps(ctx Context, props []Value) (opts useopts, params []Value, err error) {
    // Supported parameter forms:
    //      -param
    //      -param(value)
    //      -param=value
    var useList []Value // TODO: apply useList
    for _, prop := range props {
        var s string
        switch t := prop.(type) {
        case flag:
            switch s = __string(ctx, t.Value); s {
            //case "nouse", "unuse": opts.unuse = true
            case "reuse": opts.reuse = true
            default: params = append(params, prop)
            }
        case *pair: // -param=value
            switch tt := t.key.(type) {
            case flag:
                switch s = __string(ctx, tt.Value); s {
                case "use": useList = append(useList, t.val)
                default: params = append(params, prop)
                }
            default:
                debug(ctx, "parameter `%v' unsupported `%T`", prop, prop)
            }
        case *argumented: // -param(value)
            switch tt := t.Value.(type) {
            case flag:
                switch s = __string(ctx, tt.Value); s {
                case "use": useList = append(useList, t.args...)
                default: params = append(params, prop)
                }
            default:
                debug(ctx, "parameter `%v' unsupported `%T`", prop, prop)
            }
        default:
            debug(ctx, "parameter `%v` unsupported `%T`", prop, prop)
        }
    }
    return
}

func (l ul) use(ctx Context, doc *commentgroup, g *clause_opts, _ int) {
	if l.p.imports = append(l.p.imports, &use_spec{g.spec}); g.skip {
		// TODO: maybe give some information
		return
	}

	var specVal0 Value
	switch v := g.spec[0].(type) {
    case *pair:
        var s string
        if f, y := v.key.(flag); !y {
            debug(ctx, "'%v' invalid use spec", v.key)
        } else if s = __string(ctx, f.Value); s != "list" {
            debug(ctx, "'%v' invalid use spec, do you mean -list?", v.key)
        }
		specVal0 = v.val
	case *argumented:
		specVal0, ctx = v.Value, v.ctx(ctx)
	default:
		specVal0 = v
    }

	var specVals []Value
	for _, val := range xmerge(ctx, specVal0) {
		if !isTrivial(val) { specVals = append(specVals, val) }
	}
	if len(specVals) == 0 {
        erro(pc(ctx,g.spec), "empty use spec: %v", ts(g.spec[0]))
    }

	var opts useopts
	var args = parseOpts(ctx, &opts, append(g.remainder, g.spec[1:]...)...)
	for _, a := range args {
		if _, y := a.(flag); y {
			erro(pc(ctx,a), "unkown use opts: %v", ts(a))
		}
	}

	for _, specVal := range specVals {
		l.use_spec(ctx, opts, specVal, args...)
	}
	return
}

func (l ul) files(ctx Context, doc *commentgroup, g *clause_opts, _ int) {
	if len(g.spec) == 0 {
		erro(ctx, "missing file specification properties", trace{}) // Halt immediately
	}
	if len(g.spec) > 1 {
		erro(ctx, "too many properties: %v", g.spec, trace{})
	}

	var p Value
	var patts, paths []Value

	if l.p.tok == SELECT_PROG1 { // e.g., '=>' or '⇒'
		l.p.next(ctx, true) // step forward with spaces skipped
		if l.p.tok == LINEND || l.p.lineComment != nil {
			erro(ctx, "expecting files path after '⇒'")
		}
		p = l.expr(ctx)
	}

	l.p.spaces(ctx)

	if g.skip { return }

	if t := parseOpts(ctx, &g.general_opts, g.remainder...); t != nil {
		erro(ctx, "unsupported opts: %v", t)
	}

	// Expand the left-hand side (Patterns) eagerly
	if t := expand(original{ctx,defExpand1}, g.spec[0]); t == nil {
		erro(ctx, "nil file pattern: %v", g.spec[0], trace{})
	} else if x, y := t.(*group); y {
		patts = merge(x.elems...)
	} else {
		patts = merge(t)
	}

	if p == nil {
		if len(patts) == 1 {
			if x, y := patts[0].(*argumented); y {
				if f, y := x.Value.(flag); y {
					switch __string(ctx, f.Value) {
					default: // TODO: parse files options
						erro(ctx, "invalid files flag: %v")
					}
				}
			}
		}
	} else {
		if len(patts) == 1 {
			if f, y := patts[0].(flag); y {
				switch __string(ctx, f.Value) {
				default: // TODO: parse files options
					erro(ctx, "invalid files flag: %v")
				}
			}
		}

		// Expand the right-hand side (Paths) eagerly
		switch x := expand(original{ctx,defExpand1}, p).(type) {
		case *group:
			paths = x.elems
		default:
			// CRITICAL FIX: Prevent appending nil to paths
			if x != nil {
				paths = []Value{ x }
			}
		}
	}

	// Route into the Virtual File System
	map_files(ctx, l.project, patts, paths)
}

func (p *parser) assert(ctx Context, doc *commentgroup, g *clause_opts, _ int) {
	if !g.skip { call(ctx, symAssert, g.remainder, g.spec...) }
}

func (p *parser) append(ctx Context, doc *commentgroup, g *clause_opts, _ int) {
	if !g.skip { call(ctx, symAppend, g.remainder, g.spec...) }
}

func (l ul) clear_locals() {
	// l.project.mutex.Lock()
	for i := len(l.p.locals)-1; 0 <= i; i -= 1 {
		for s, d := range l.p.locals[i] {
			if d == nil {
				delete(l.project.elems, s)
			} else if o := l.project.Lookup(s); o != nil {
				if x, y := o.(*def); y { *x = *d }
			}
		}
	}
	// l.project.mutex.Unlock()
	l.p.locals = nil
}

func (l ul) local(ctx Context, _ *commentgroup, g *clause_opts, _ int) {
	var local map[Symbol]*def
	var vals = xmerge(_final(ctx), append(g.remainder, g.spec...)...)

	for _, a := range vals {
		if x, y := a.(flag); y {
			switch s := __string(ctx, x.Value); s {
			case "clear":
				l.clear_locals()
			case "pop":
				if i := len(l.p.locals); 0 < i {
					var last = l.p.locals[i-1]
					l.p.locals = l.p.locals[:i-1]
					// l.project.mutex.Lock()
					for s, d := range last { // s is correctly a Symbol here from the map
						if d == nil {
							delete(l.project.elems, s)
						} else if false {
							l.project.elems[s] = d
						} else if o := l.project.Lookup(s); o != nil {
							if x, y := o.(*def); y { *x = *d }
						}
					}
					// l.project.mutex.Unlock()
				}
			default:
				erro(ctx, "unsupported flag: %v", ts(a,ctx))
			}
			continue
		}

		// CRITICAL FIX: Extract the Symbol directly to avoid string allocations!
		var sym = __symbol(ctx, a)
		if sym == symEmpty {
			erro(ctx, "empty local: %v", ts(a,ctx))
		}

		if local == nil { local = make(map[Symbol]*def) }

		var t *def
		if o := l.project.Lookup(sym); o != nil {
			if x, y := o.(*def); y { t = new(def); *t = *x }
		}
		local[sym] = t // Assign directly using the integer channel!
	}

	if local != nil { l.p.locals = append(l.p.locals, local) }
}

func (l ul) eval(ctx Context, doc *commentgroup, g *clause_opts, _ int) {
	if g.skip { return }
	if g.spec == nil {
		var opts struct{
			optimize Value `opt,optimize`
		}
		for _, op := range parseOpts(_final(ctx), &opts, g.values...) {
			var val Value
			if v, y := op.(*pair); y { op, val = v.key, v.val }
			erro(ctx, "unsupport flag: %v (%v)", ts(op), val)
		}
		return
	}

	prop0 := g.spec[0]
	if isTrivial(prop0) {
		erro(pc(ctx,l.p), "illegal")
	}

	// CRITICAL FIX: Upgrade from string to Symbol
	var sym Symbol = symEmpty
	var opts []Value
	if a, y := prop0.(*argumented); y { prop0, opts = a.Value, a.args }
	switch t := prop0.(type) {
	case *delegate:
		for i, x := range merge(expand(_final(ctx),t)) {
			switch t := x.(type) {
			case *pair:
				debug(pc(ctx,l.p), "%v → %v", t.key, t.val)
			case *word:
				if sym != symEmpty {
					erro(pc(ctx,l.p), "%v → %d. %v", prop0, i, x)
				} else {
					sym = t.s // Fast-path integer grab!
				}
			default:
				erro(pc(ctx,l.p), "%v → %d. %v", prop0, i, ts(x))
			}
		}
		return
	case *pair:
		debug(pc(ctx,l.p), "%v → %v", t.key, t.val)
	default:
		// CRITICAL FIX: Extract the Symbol directly to avoid string allocations!
		sym = __symbol(ctx, prop0)
	}

	// ZERO-ALLOCATION MATCHING: We compare integers instead of strings!
	// (Note: Go allows function calls in switch cases. intern() is fast enough
	// here, but you could also add symConfiguration to your constants!)
	switch sym {
	case intern("-configuration"), intern("configuration"):
		erro(pc(ctx,l.p), "configuration is done at parse time")
	case symEmpty:
		erro(pc(ctx,l.p), "empty eval command")
	}

	// Pass the fast Symbol into the resolver!
	resolved := l.resolve(ctx, prop0, sym)

	switch x := resolved.(type) {
	case evaler: x.eval(ctx, opts, expands(_final(ctx), g.spec[1:]...))
	case *builtin:
		// Assuming builtin.name was upgraded to a Symbol when we updated knownobject
		switch x.name {
		case intern("plain"): evoke(ctx, x, opts, g.spec[1:])
		}
	default:
		erro(pc(ctx,l.p), "resolved '%s' is not evaler: %v → %v", typeof(resolved), prop0, sym)
	}

	// TODO: if c, y := res.(code); y { ... }
}

type is_directive_ctx struct{}
type directive_ctx struct { Context }
func (c directive_ctx) do(ctx Context, op any) any {
	switch op.(type) {
	case is_directive_ctx: return true
	}
	return c.Context.do(ctx, op)
}

func (l ul) directive(ctx Context) (props []Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "directive")) }

	ctx = directive_ctx{ctx}

paramsloop:
	for l.p.tok != EOF {
		l.p.spaces(ctx)

		if l.p.lineComment != nil {
			// TODO: comment = p.lineComment
			break
		}

		switch l.p.tok {
		case COLON, COMMA, RPAREN, RBRACE, LINEND: break paramsloop
		case SELECT_PROP, SELECT_PROG1, SELECT_PROG2:
			if /* truly(ctx, can_select{}) */true { break paramsloop }
		}

		props = append(props, l.expr(ctx))
	}
	return
}

func (l ul) spec(ctx Context, keyword token, pos Pos, f parseSpecFunc) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "spec("+keyword.String()+")")) }

	var opts = clause_opts{ keyword: keyword }
	for l.p.spaces(ctx); l.p.tok == MINUS; l.p.spaces(ctx) {
		opts.values = append(opts.values, l.expr(ctx))
	}

	opts.remainder = parseOpts(ctx, &opts, opts.values...)

	for _, cond := range opts.conds {
		if t := __true(ctx, cond); !t {
			opts.skip = true
			break
		}
	}

	l.p.spaces(ctx)
	ctx = pc(ctx, l.p.pos)

	switch l.p.tok {
	case LINEND:
		switch keyword {
		case EVAL, LOCAL:
			f(ctx, nil, &opts, 0)
			return
		}

		erro(ctx, "%v: no specs, remainder: %v", keyword, opts.remainder)

	case LPAREN:
		l.p.next(ctx, true)
		for iota := 0; l.p.tok != RPAREN && l.p.tok != EOF && (l.p.stop == 0 || l.p.pos < l.p.stop); iota++ {
			// TODO: collect documentation comments
			for l.p.tok == SPACE || l.p.tok == LINEND { l.p.next(ctx, true) }
			if l.p.tok == RPAREN || l.p.tok == EOF { break  }

			opts.spec = l.directive(ctx)

			f(ctx, l.p.leadComment, &opts, iota)

			if l.p.tok == COMMA || l.p.tok == LINEND { l.p.next(ctx, true) }
		}
		l.p.expect(ctx, RPAREN)
		l.p.spaces(ctx)
		if l.p.tok != EOF { l.p.linend(ctx) }
		return
	}

	if l.p.tok != LINEND && l.p.tok != EOF && (l.p.stop == 0 || l.p.pos < l.p.stop) {
		opts.spec = l.directive(ctx)

		f(ctx, nil, &opts, 0)

		if l.p.tok == COMMA { l.p.next(ctx, true) }
	}

	if l.p.tok != EOF && (l.p.stop == 0 || l.p.pos < l.p.stop) {
		if l.p.spaces(ctx); l.p.lineComment == nil { l.p.linend(ctx) }
	}
}

func forid_elems(ctx Context, elems, stems []Value, f func(elems, stems []Value)) {
    for i, elem := range elems {
		if x, y := elem.(*argumented); y {
			var prefix, suffix = elems[:i], elems[i+1:]
			forids(ctx, x, func(ident Value, stems2 []Value) {
				var head   = append(prefix, ident)
				var stems3 = append(stems , stems2...)
				forid_elems(ctx, suffix, stems3, func(elems, stems []Value) {
					f(append(head, elems...), stems)
				})
			})
			return
		}
	}
    f(elems, stems)
}

func forids(ctx Context, idents Value, f func(Value, []Value)) {
    switch t := idents.(type) {
    case *argumented:
        var args = xmerge(ctx, t.args...)
		forids(ctx, t.Value, func(ident Value, stems []Value) {
			for _, arg := range args {
				if !isTrivial(arg) {
					f(prefix(ctx, ident, arg), append(stems, arg))
				}
			}
		})
    case *compound:
        forid_elems(ctx, t.elems, nil, func(elems, stems []Value) {
            if len(stems) == 0 {
				f(t, stems)
			} else {
                f(_compound(elems...), stems)
            }
        })
    default:
        f(t, nil)
    }
}

func (l ul) assign(ctx Context, idents []Value) (res []*def) {
	if l_traverse.enabled || debugSyntax(ctx, "assign") {
		defer un(l_trace(l_traverse, fmt.Sprintf("assign(%s)", idents)))
	}

	ids := []Value{}
	for _, v := range idents {
		// Use def_name{_final(ctx)} to prevent evaluating
		// the variable to its value when resolving the LHS identifier!
		forids(ctx, expand(def_name{_final(ctx)}, v), func(v Value, _ []Value) { ids = append(ids, v) })
	}

	pos, tok := l.p.pos, l.p.tok
	l.p.next(ctx, true) // the assign token

	// Parse the RHS exactly once! This prevents parser stream desynchronization
	// when multiple variables are assigned, or when `?=` skips assignment for already-defined variables.
	rhsVals := l.values(ctx)

	for _, id := range ids {
		var alt object
		var d *def

		switch t := id.(type) {
		case *argumented:
			erro(ctx, "multiple defs: %v, args=%v", t.Value, t.args)

		case *group:
			erro(ctx, "multiple defs: %v", t.elems)

		case *arrow:
			if v := expand(_final(ctx), t); v == nil {
				erro(ctx, "%v is nil", ts(t,ctx))
			} else if x, y := v.(*def); !y {
				erro(ctx, "%v is not a def: %v", ts(t,ctx), ts(v,ctx))
			} else {
				d = x
			}

		default: // *word, *compound, *qualword, *path, flag:
			var sym = __symbol(def_name{ctx}, id)

			if sym == symEmpty {
				erro(pc(ctx,t), "empty name: %s: `%v`", typeof(id), id, callstack{num:32})
			} else if _, y := builtins[sym]; y { // Assuming builtins map is upgraded to map[Symbol]...
				erro(pc(ctx,t), "`%v` is a builtin name (%v)", ident, sym)
			}

			if checkpoints {
				// RegEx requires a string, so we safely extract it here.
				if illegal_name_prefix.MatchString(sym.String()) {
					erro(pc(ctx,t), "illegal name: %v", sym, callstack{num:32})
				}
			}

			// NATIVE INTEGER ROUTING!
			prev := l.project.resolve(ctx, sym)

			var isNew bool
			// _def must be upgraded to accept sym Symbol instead of a string!
			d, isNew = l.project._def(ctx, defInvalid, sym)
			if isNew { d.pos = pos // ensure def pos is correct
				// Or add this logic inside project._def?
				if nameStr := sym.String(); strings.HasPrefix(nameStr, "use.") {
					// Extract the target flag (e.g. "use.-l" -> "-l")
					var targetName string
					if m := name_prefix.FindStringSubmatch(nameStr); m != nil {
						targetName = m[3] // Handles complex prefixes if applicable
					} else {
						targetName = strings.TrimPrefix(nameStr, "use.")
					}

					// Register it as a public export!
					l.project.addExport(intern(targetName))
				}
			}

			if prev == nil || d == nil {
				// no derived value
			} else if x, y := prev.(*def); !y {
				// not a def
			} else if x == nil {
				erro(ctx, "prev def '%s' is nil", sym)
			} else if x != d && x.scope != d.scope && alt == nil {
				switch tok {
				case ASSIGN_ADD, ASSIGN_USH:
					if d.o == defVoid && d.o != x.o { d.origin(ctx, x.o) }
					if !isTrivial(x.value) { d.append(ctx, x.value) }
				}
			}
		}

		if d == nil {
			erro(ctx, "def is nil: %v", ts(id,ctx)) // Fixed 'ident' typo to 'id'
		}

		_ctx := defval{original{ctx, 0}, d}

		if !d.pos.IsValid() { d.pos = l.p.pos }

		switch tok {
		case ASSIGN_EXC: // !=
			d.pos, _ctx.o = pos, defExecute
			d.origin(ctx, _ctx.o)
			d.val(_ctx, rhsVals)
		case ASSIGN: // =
			d.pos, _ctx.o = pos, defExpand0
			d.origin(ctx, _ctx.o)
			d.val(_ctx, rhsVals)
		case ASSIGN_CO1: // :=
			d.pos, _ctx.o = pos, defExpand1
			d.origin(ctx, _ctx.o)
			d.val(_ctx, expands(_ctx, rhsVals...))
		case ASSIGN_CO2: // ::=
			d.pos, _ctx.o = pos, defExpand2
			d.origin(ctx, _ctx.o)
			d.val(_ctx, expands(_ctx, rhsVals...))
		case ASSIGN_CO3: // ;:=
			d.pos, _ctx.o = pos, defExpand3
			d.origin(ctx, _ctx.o)
			d.val(_ctx, expands(_ctx, rhsVals...))
		case ASSIGN_QUE: // ?=
			if d.o == defInvalid {
				d.pos, _ctx.o = pos, defAssign0
				d.origin(ctx, defExpand0)
				d.val(_ctx, rhsVals)
			}
		case ASSIGN_ADD: // +=
			if d.o == defInvalid { d.o = defExpand0 }
			switch _ctx.o = d.o|defAssign1; {
			case d.o&defExpand0 != 0:
				d.set(_ctx, nil, rhsVals...)
			case d.o&(defVoid|defExpand0|defExpand1|defExpand2|defExpand3) != 0:
				d.set(_ctx, nil, expands(_ctx, rhsVals...)...)
			default:
				erro(ctx, "unknown: %v %v", _ctx.o, d.name)
			}
		case ASSIGN_USH: // =+
			if d.o == defInvalid { d.o = defExpand0 }
			switch _ctx.o = d.o|defAssign2; {
			case d.o&defExpand0 != 0:
				d.val(_ctx, append(rhsVals, d.value))
			case d.o&(defExpand1|defExpand2|defExpand3) != 0:
				d.val(_ctx, append(expands(_ctx, rhsVals...), d.value))
			default:
				erro(ctx, "unknown: %v %v", _ctx.o, d.name)
			}
		case ASSIGN_POP: // -=
			if d.o == defInvalid { d.o = defExpand0 }
			if d.value != nil {
				if dv := merge(d.value); len(dv) > 0 {
					var vals, _vals []Value
					switch _ctx.o = d.o|defAssign3; {
					case d.o&defExpand0 != 0:
						vals = rhsVals
					case d.o&(defExpand1|defExpand2|defExpand3) != 0:
						vals = expands(_ctx, rhsVals...)
					default:
						erro(ctx, "unknown: %v %v", _ctx.o, d.name)
					}
				outer1:
					for _, v := range dv {
						for _, s := range vals {
							if cmp(ctx, v, s) == cmpEqual { continue outer1 }
						}
						_vals = append(_vals, v)
					}
					d.value = ease(ctx, _vals)
				}
			}
		case ASSIGN_SAD, ASSIGN_SUS: // -+=, -=+
			var vals, _vals []Value
			if d.o == defInvalid { d.o = defExpand0 }
			if tok == ASSIGN_SAD {
				_ctx.o = d.o|defAssign4
			} else {
				_ctx.o = d.o|defAssign5
			}
			switch {
			case d.o&defExpand0 != 0:
				vals = rhsVals
			case d.o&(defExpand1|defExpand2|defExpand3) != 0:
				vals = expands(_ctx, rhsVals...)
			default:
				erro(ctx, "unknown: %v %v", _ctx.o, d.name)
			}
			if d.value != nil {
				if dv := merge(d.value); len(dv) > 0 {
				outer2:
					for _, v := range dv {
						for _, sv := range vals {
							if cmp(ctx, v, sv) == cmpEqual { continue outer2 }
						}
						_vals = append(_vals, v)
					}
				}
			}
			switch tok {
			case ASSIGN_SAD: _vals = append(_vals, vals...) // -+=
			case ASSIGN_SUS: _vals = append(vals, _vals...) // -=+
			}
			d.value = ease(ctx, _vals)
		default:
			erro(ctx, "unknown: %v %v %v", _ctx.o, d.name, tok)
		}

		l.p.lineComment = nil

		res = append(res, d)
	}
	return
}

func (l ul) recipe(ctx Context) (recipes []Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "recipe")) }

	// TODO: comment *commentgroup
	// TODO: doc = p.leadComment
	var p Pos
	var elems []Value
	var isList, isPlainline bool

	switch l.p.dialect {
	case symEmpty, symValue:
		l.p.scanner.pop(isStrcompLine)
		l.p.next(ctx, true) // skip RECIPE or SEMICOLON and parse in list mode
		if p, isList = l.p.pos, true; !l.p.is_end_of_line() {
			var c = p_recipe{ctx, true, nil, nil} // builtin value recipe
			for l.p.tok != EOF && l.p.tok != SEMICOLON && l.p.tok != LINEND && l.p.lineComment == nil {
				elems = append(elems, l.expr(&c))
				if l.p.spaces(ctx); l.p.lineComment != nil { break }
			}
		}

	case symEval:
		l.p.scanner.pop(isStrcompLine)
		l.p.next(ctx, true) // skip RECIPE or SEMICOLON and parse in list mode
		if p, isList = l.p.pos, true; !l.p.is_end_of_line() {
			var x = l.expr(ctx) // parse first expr of recipe

			var a *argumented
			if a, _ = x.(*argumented); a != nil { x = a.Value }
			if x == nil {
				erro(pc(ctx,p), "parsed nil value, dialect=%s", l.p.dialect)
			}

			if l.p.dialect == symValue {
				// no resolving commands
			} else if t, y := x.(*word); !y {
				// does nothing
			} else if s := l.resolve(ctx, t, t.s); isTrivial(s) {
				erro(pc(ctx,p), "no such symbol: %v, %s → %s; dialect=%s", t.s, ts(x), ts(s), l.p.dialect)
			} else if _, y := s.(*builtin); !y {
				erro(pc(ctx,p), "'%s' is not a command (%s)", t.s, typeof(s))
			} else {
				x = s
			}

			if a != nil {
				elems, a.Value = append(elems, a), x
			} else {
				elems = append(elems, x)
			}

			var cmdargs []Value
			var c = p_recipe{ctx, true, nil, nil} // builtin recipe

			for l.p.tok != EOF && l.p.tok != SEMICOLON && l.p.tok != LINEND && l.p.lineComment == nil {
				if l.p.spaces(ctx); l.p.lineComment != nil { break }
				if !l.p.tok.is_rule_delim() {
					x = l.expr(&c)
				} else {
					erro(ctx, "unsupported token: %s, %v", l.p.tok, elems)
				}
				if cmdargs = append(cmdargs, x); l.p.tok == COMMA {
					l.p.next(ctx, true)
					elems = append(elems, _list(cmdargs...))
					cmdargs = []Value{}
				}
				if l.p.lineComment != nil { break }
			}

			elems = append(elems, _list(cmdargs...))
		}

	default:
		l.p.scanner.push(isStrcompLine) // NOTE: scanner does not set isStrcompLine correctly, fixit here
		l.p.next(ctx, true) // skip RECIPE or SEMICOLON and parse in line-string mode
		p = l.p.pos

		switch l.p.dialect { case symPlain, symText: isPlainline = true }

		var c = p_recipe{ctx, false, nil, nil} // builtin text
		for !l.p.is_end_of_line() {
			var x Value
			if l.p.tok == RAW {
				x = l.literal(&c)
			} else {
				x = l.expr(&c)
			}
			if c.lines == nil {
				c.elems = append(c.elems, x)
			}
		}
		l.p.scanner.pop(isStrcompLine)

		if c.lines != nil {
			if c.elems != nil {
				erro(ctx, "%v %v", c.elems, c.lines)
			}
			for _, a := range c.lines {
				recipes = append(recipes, &plainline{elements{merge(a...)}})
			}
			return
		}

		elems = c.elems
	}

	if l.p.spaces(ctx) ; l.p.tok != EOF { l.p.linend(ctx) }

    if len(elems) == 0 {
        return []Value{ _none(p) }
    } else if isList {
        return []Value{ _list(elems...) }
	} else if isPlainline {
		return []Value{ &plainline{elements{merge(elems...)}} }
	} else {
		return []Value{ &recipe{strcomp{elements{elems}}} }
    }
}

// Parsing (var a=xxx,b=yyy) or (var a) definitions
func (p *parser) var_modifier(ctx Context, args ...Value) (err error) {
	for _, elem := range args {
		sym, pos := __symbol(ctx, elem), elem.Pos()

		// Optional but recommended safety check
		if sym == symEmpty {
			erro(ctx, "empty var name in modifier: %v", ts(elem,ctx))
			continue
		}

		// Pass the Symbol directly into auto() and the elems map!
		if d := _scope(ctx).auto(ctx, sym); d != nil {
			d.pos = pos
			if false {
				val := _scope(ctx).elems[sym] // NATIVE INTEGER MAP LOOKUP!
				debug(ctx, "%v ; %v ; %s", d, val, _scope(ctx).comment)
			}
		}
	}
	return
}

func (l ul) define_configs(ctx Context) {
	for _, t := range l.p.targets {
		l.project.def(ctx, defConfig, t)
	}
}

func (l ul) modifier(ctx Context) (res *modifier) {
	pos := l.p.pos
	l.p.expect(ctx, LPAREN)
	l.p.spaces(ctx)

	var elems []Value
	var val = l.expr(ctx)

	// CRITICAL FIX: Fast-path Symbol extraction!
	var sym = __symbol(ctx, val)
	if sym == symEmpty {
		erro(pc(ctx,val), "unsupported modifier: %s", ts(val,ctx))
		// Assuming `dialects` and `modifiers` maps are upgraded to `map[Symbol]...`
	} else if _, y := dialects[sym]; y {
		if l.p.dialect == symEmpty {
			l.p.dialect = sym // Safely bridge back to string if dialect requires it
		} else {
			erro(pc(ctx,l.p), "multi-dialects unsupported, already defined '%s'", l.p.dialect)
		}
	} else if _, y := modifiers[sym]; !y {
		erro(pc(ctx,l.p), "no such dialect or modifier: %s", sym)
	}

	for l.p.tok != RPAREN && l.p.tok != EOF {
		l.p.spaces(ctx)
		pos := l.p.pos

		va := l.values(ctx)

		// 1. Parse-time local-define (Forward Declaration)
		// ZERO-ALLOCATION MATCHING: Pure integer comparison!
		if sym == symVar { l.p.var_modifier(ctx, va...) }

		// 2. CRITICAL FIX: Removed 'else'. We MUST append to elems
		// so the AST retains the arguments for runtime!
		if n := len(va); n == 1 {
			elems = append(elems, va[0])
		} else if n > 1 {
			elems = append(elems, &list{elements{va}})
		} else {
			elems = append(elems, &null{valbase{l.p.pos}})
		}

		if l.p.tok == COMMA { l.p.next(ctx, true) }
		if l.p.pos == pos {
			erro(pc(ctx,l.p), "unsupported modifier arg: %v '%v'", l.p.tok, l.p.lit)
		}
	}

	l.p.expect(ctx, RPAREN)

	if val == nil && len(elems) == 0 {
		erro(pc(ctx,l.p), "empty modifier")
	}

	res = new(modifier)
	res.pos, res.elems = pos, append([]Value{val}, elems...)
	return
}

// example: {(modifier ...) ...}
func (l ul) modification(ctx Context) *modification {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "modification")) }

	var elems []*modifier
	var pos = l.p.pos

	for l.p.tok != RBRACE && l.p.tok != EOF {
		l.p.spaces(ctx)

		// Use 'continue' to safely consume consecutive newlines and comments.
		if l.p.tok == SPACE || l.p.tok == LINEND || l.p.tok == COMMENT {
			l.p.next(ctx, true)
			continue
		}

		if m := l.modifier(ctx); m != nil { elems = append(elems, m) }
	}

	if len(elems) == 0 {
		erro(ctx, "empty modifier group")
	}
	if l.p.tok == COLON {
		erro(ctx, "unexpected colon after modifer")
	}
    return &modification{valbase{pos}, elems}
}

//
// $@      The file name of the target of the rule. If the target is an archive member, then ‘$@’ is the name of the archive file. In a pattern rule that has multiple targets (see Introduction to Pattern Rules), ‘$@’ is the name of whichever target caused the rule’s recipe to be run.
// $%      The target member name, when the target is an archive member. See Archives. For example, if the target is foo.a(bar.o) then ‘$%’ is bar.o and ‘$@’ is foo.a. ‘$%’ is empty when the target is not an archive member.
// $<      The name of the first prerequisite. If the target got its recipe from an implicit rule, this will be the first prerequisite added by the implicit rule (see Implicit Rules).
// $?      The names of all the prerequisites that are newer than the target, with spaces between them. For prerequisites which are archive members, only the named member is used (see Archives).
// $^      The names of all the prerequisites, with spaces between them. For prerequisites which are archive members, only the named member is used (see Archives). A target has only one prerequisite on each other file it depends on, no matter how many times each file is listed as a prerequisite. So if you list a prerequisite more than once for a target, the value of $^ contains just one copy of the name. This list does not contain any of the order-only prerequisites; for those see the ‘$|’ variable, below.
// $+      This is like ‘$^’, but prerequisites listed more than once are duplicated in the order they were listed in the makefile. This is primarily useful for use in linking commands where it is meaningful to repeat library file names in a particular order.
// $|      The names of all the order-only prerequisites, with spaces between them.
//         Order-only prerequisites can be specified by placing a pipe symbol (|) in the prerequisites list: any prerequisites to the left of the pipe symbol are normal; any prerequisites to the right are order-only.
// $*      The stem with which an implicit rule matches (see How Patterns Match). If the target is dir/a.foo.b and the target pattern is a.%.b then the stem is dir/foo. The stem is useful for constructing names of related files.
//         In a static pattern rule, the stem is part of the file name that matched the ‘%’ in the target pattern.
//         In an explicit rule, there is no stem; so ‘$*’ cannot be determined in that way. Instead, if the target name ends with a recognized suffix (see Old-Fashioned Suffix Rules), ‘$*’ is set to the target name minus the suffix. For example, if the target name is ‘foo.c’, then ‘$*’ is set to ‘foo’, since ‘.c’ is a suffix. GNU make does this bizarre thing only for compatibility with other implementations of make. You should generally avoid using ‘$*’ except in implicit rules or static pattern rules.
//         If the target name in an explicit rule does not end with a recognized suffix, ‘$*’ is set to the empty string for that rule.
//
// $-      the execution result
// $~      the grep modifier result
// $/      the current absolute path
// $.      the current relative path
// $,      the current temporary path
//
// Similar to makefile automatic variables, see
//   * https://www.gnu.org/software/make/manual/html_node/Automatic-Variables.html#Automatic-Variables
var rule_autos = map[Symbol]struct{}{
	symAt:        struct{}{}, // @
	symAtD:       struct{}{}, // @D
	symAtF:       struct{}{}, // @F
	symAtA:       struct{}{}, // @'
	symBar:       struct{}{}, // |
	symBarD:      struct{}{}, // |D
	symBarF:      struct{}{}, // |F
	symBarA:      struct{}{}, // |'
	symCaret:     struct{}{}, // ^
	symCaretD:    struct{}{}, // ^D
	symCaretF:    struct{}{}, // ^F
	symCaretA:    struct{}{}, // ^'
	symLangle:    struct{}{}, // <
	symLangleD:   struct{}{}, // <D
	symLangleF:   struct{}{}, // <F
	symLangleA:   struct{}{}, // <'
	symRangle:    struct{}{}, // >
	symRangleD:   struct{}{}, // >D
	symRangleF:   struct{}{}, // >F
	symRangleA:   struct{}{}, // >'
	symPercent:   struct{}{}, // %
	symPercentD:  struct{}{}, // %D
	symPercentF:  struct{}{}, // %F
	symPercentA:  struct{}{}, // %'
	symAsterisk:  struct{}{}, // *
	symAsteriskD: struct{}{}, // *D
	symAsteriskF: struct{}{}, // *F
	symAsteriskA: struct{}{}, // *'
	symQues:      struct{}{}, // ?
	symQuesD:     struct{}{}, // ?D
	symQuesF:     struct{}{}, // ?F
	symQuesA:     struct{}{}, // ?'
	symPlus:      struct{}{}, // +
	symPlusD:     struct{}{}, // +D
	symPlusF:     struct{}{}, // +F
	symPlusA:     struct{}{}, // +'
	symDash:      struct{}{}, // -
	symTilde:     struct{}{}, // ~
}

func (l ul) rule(ctx Context, targets []Value) (result Value) {
	if l_traverse.enabled || debugSyntax(ctx, "rule") { defer un(l_trace(l_traverse, "rule")) }

	ctx = p_rule_ctx{ctx, l.p}

    if l.project != _scope(ctx).project {
		erro(ctx, "mismatched project/scope : %v", targets)
	}

	// NOTE: expand targets to speed up for later usage, it might spend lots of time in
	// project.entry while matching for entry looked up if not expanded right now.
	targets = expands(_final(ctx), targets...)

	var scopeComment string
	if len(targets) == 1 {
		scopeComment = targets[0].String()
	} else {
		scopeComment = sf("%v", targets)
	}

	// TODO: doc = p.leadComment
	var depends, ordered, recipes []Value
	defer l.closescope(l.openscope(scopeComment))
	defer func() { l.p.dialect, l.p.ruparas = symEmpty, nil } ()

	l.p.dialect = symEmpty
	l.p.ruparas = nil

	defer func(t []Value) { l.p.targets = t } (l.p.targets)
	l.p.targets = targets // save targets for later refering
	l.p.next(ctx, true) // skip rule delimeters and spaces

	if l.p.tok != SEMICOLON && l.p.tok != BAR && !l.p.is_end_of_line() {
		depends = l.depends(ctx, true)
	}
	if l.p.tok == BAR { // '|' starts the ordered prerequisites
		l.p.next(ctx, true)
		if l.p.tok != SEMICOLON && !l.p.is_end_of_line() {
			ordered = l.depends(ctx, false)
		}
	}

	if l.p.tok == SEMICOLON { // ;
		// Parse inline recipe in the program scope.
		recipes = append(recipes, l.recipe(ctx)...)
	} else /*if p.tok == LINEND || p.lineComment != nil*/ {
		// Parse recipes in the program scope.
		l.p.scanner.recipes(true) // Turn on recipes before LINEND.
		if l.p.linend(ctx) { // Take the new line.
			for l.p.recipe_start() {
				recipes = append(recipes, l.recipe(ctx)...)
			}
		}
		l.p.scanner.recipes(false)
	}

	var prog = program{
		pos: targets[0].Pos(),
		language:  l.p.dialect,
		params:    l.p.ruparas,
		project:   l.project,
		depends:   depends,
		ordered:   ordered,
		recipes:   recipes,
	}

	if res := l.entries(ctx, &prog, targets); 1 == len(res) {
		return res[0]
	} else if 1 < len(res) {
		return list_t[entry](res...)
	} else {
		return _null(prog.pos)
	}
}

func (l ul) entries(ctx Context, prog *program, targets []Value) (res []entry) {
	for _, target := range targets {
        if isTrivial(target) { continue }

		// CRITICAL FIX: Route pattern rules to the patterns slice, NOT the valcache!
		if patterned(ctx, target) {
			r := &rule{target: target, program: []*program{prog}}
			prog.project.patterns = append(prog.project.patterns, r)
			res = append(res, r)
			continue
		}

        var entry = map_entry(pc(ctx,target), prog.project, target, prog)
        if entry == nil {
            erro(ctx, "creating entry failed for %v", target)
        }

		res = append(res, entry)

        if x, y := entry.destiny().(flag); y && x.Value != nil {
			if prog.project.name != symTilde { // "~"
				var s = __string(ctx, x.Value)
				l.globe.AddFlagEntry(s, entry)
			}
        }
    }
    return
}

func (l ul) def_end(ctx Context) {
	p := l.p
	p.spaces(ctx)
	p.expect(ctx, DEF)
	p.spaces(ctx)

	name := l.expr(ctx)
	p.spaces(ctx)
	p.linend(ctx)

	t := &template{ state:p.scanner.scanstate }
	if a, y := name.(*argumented); y { name, t.params = a.Value, a.args }
	t.name = __symbol(ctx, name)

	var nested = 0
	for p.tok != EOF {
		switch pos := p.pos; p.tok {
		case SPACE, LINEND:
			p.next(ctx, true)

		case DEF: nested += 1
			p.forwardLine(ctx) // Fast-forward rest of the line

		case END:
			if nested > 0 { nested -= 1
				p.forwardLine(ctx) // Fast-forward rest of the line
				continue
			}

			// We found the true, un-nested end of the block!
			p.next(ctx, true)
			p.linend(ctx)

			state := p.scanner.scanstate
			t.end, t.endPos = &state, pos
			p.templates = append(p.templates, t)
			return

		default:
			// Not a block keyword.
			p.forwardLine(ctx) // Fast-forward the rest of the line instantly!
		}
	}
}

func (l ul) foreach_done(ctx Context) {
	if l.p.spaces(ctx); l.p.tok == LINEND {
		erro(ctx, "unexpected end of line")
	}

	l.p.expect(ctx, FOREACH)
	l.p.spaces(ctx)

	var vals = xmerge(_final(ctx), l.values(ctx)...)

	l.p.spaces(ctx)
	l.p.linend(ctx)

	t := &template{ state:l.p.scanner.scanstate }

	var nested = 0
	for l.p.tok != EOF {
		switch pos := l.p.pos; l.p.tok {
		case SPACE, LINEND:
			l.p.next(ctx, true)

		case FOREACH, FOR: nested += 1
			l.p.forwardLine(ctx)

		case DONE:
			if nested > 0 { nested -= 1
				l.p.forwardLine(ctx)
				continue
			}

			l.p.next(ctx, true) // done
			l.p.linend(ctx)

			state, savedStop := l.p.scanner.scanstate, l.p.stop
			l.p.stop, t.endPos, t.end = pos, pos, &state

			var dd bool
			var dps []*diag_point
			if false && checkpoints {
				dd = strings.HasSuffix(l.project.spec.String(), "testdata/template") ||
					strings.HasSuffix(l.project.spec.String(), "testdata/template/foreach")
				if dd { for _, v := range vals { dps = append(dps, _f("%v", v)) } }
			}

			ac := automatic{Context:ctx, defs:make(def_map)}
			for _, val := range vals {
				if !isTrivial(val) {
					if x, y := val.(*defcaps); y {
						ac.set(&ac, defStatic, symUnderscore, x.Value)
						if checkpoints {
							if dd { dps = append(dps, _f("'%v' %v", symUnderscore, x.Value)) }
						}
						for _, c := range x.caps {
							ac.set(&ac, defStatic, c.name, c.value)
							if checkpoints {
								if dd { dps = append(dps, _f("'%v' %v", c.name, c.value)) }
							}
						}
					} else {
						ac.set(&ac, defStatic, symUnderscore, val)
						if checkpoints {
							if dd { dps = append(dps, _f("'%v' %v", symUnderscore, val)) }
						}
					}
					l.codeblock(&ac, t)
				}
			}
			if checkpoints { if len(dps) > 0 { debug(pc(ctx,vals), dps) } }

			// Jump the parser past the 'done' token so the
			// rest of the file can be parsed safely!
			l.p.scanner.scanstate = *t.end
			l.p.stop = savedStop
			return

		default:
			l.p.forwardLine(ctx)
		}
	}
}

func (l ul) for_done(ctx Context) {
	if l.p.spaces(ctx); l.p.tok == LINEND {
		erro(ctx, "unexpected end-of-line")
	}

	var opts struct{
		skipNil bool `skip-nil,skip-null,skipnil,skipnull,no-nil,no-null`
	}
	if  l.p.expect(ctx, FOR) ; l.p.tok == LPAREN {
		l.p.next(ctx, true) // LPAREN
		if vals := parseOpts(ctx, &opts, l.values(ctx)...); vals != nil {
			erro(ctx, "unexpected opts: %v", vals)
		}
		l.p.expect(ctx, RPAREN)
	}

	l.p.spaces(ctx)

	// You correctly upgraded these to Symbol!
	type  param struct{ name Symbol ; elems []Value }
	type nparam struct{ p Pos ; a []*param ; n int }

	var params []*nparam
	var ac = automatic{Context:ctx, defs:make(def_map)}
	for l.p.spaces(ctx); l.p.tok != EOF && !l.p.is_end_of_line(); l.p.spaces(ctx) {
		if l.p.tok == AND && params == nil {
			erro(pc(ctx,l.p), "unexpected 'and'")
		} else if l.p.tok == AND || params == nil {
			params = append(params, &nparam{p:l.p.pos})
			if l.p.tok == AND { l.p.next(ctx, true); continue }
		}

		var pars = make(map[Symbol]*param)
		var p = params[len(params)-1]
		for i, a := range merge(expand(&ac, l.expr(&ac))) {
			switch x := unbox(a).(type) {
			case *null: continue
			case *pair:
				// CRITICAL FIX: Use __symbol to extract the integer natively!
				var sym = __symbol(def_name{ctx}, expand(def_name{ctx}, x.key))

				if sym == symEmpty {
					erro(pc(ctx,a), "empty key %v", ts(x.key,ctx))
				}

				var par *param
				if pt, ok := pars[sym]; ok {
					par = pt
				} else {
					par = new(param)
					par.name = sym
					p.a = append(p.a, par)
					pars[sym] = par
				}

				if g, y := x.val.(*group); y {
					par.elems = append(par.elems, merge(g.elems...)...)
				} else {
					par.elems = append(par.elems, merge(x.val)...)
				}

				if n := len(par.elems); n > p.n { p.n = n }
				if _, y := ac.defs[par.name]; !y { ac.set(&ac, defStatic, par.name, nil) }

			case *defcaps:
				for _, cap := range x.caps {
					// cap.name is already a Symbol from our previous defcap upgrade!
					if t, y := pars[cap.name]; y {
						t.elems = append(t.elems, cap.value)
					} else {
						t = &param{cap.name, []Value{cap.value}}
						p.a = append(p.a, t)
						pars[cap.name] = t
					}
					// Ensure p.n is updated for captured regex groups!
					if n := len(pars[cap.name].elems); n > p.n { p.n = n }
				}

			default:
				erro(pc(ctx,a), "unexpected %v ; %d. %v", ts(a,ctx), i, ac.defs)
			}
		}
	}

	l.p.spaces(ctx)
	l.p.linend(ctx)

	t := &template{ state:l.p.scanner.scanstate }

	var nested = 0
	for l.p.tok != EOF {
		switch pos := l.p.pos; l.p.tok {
		case SPACE, LINEND:
			l.p.next(ctx, true)

		case FOR, FOREACH: nested += 1
			l.p.forwardLine(ctx)

		case DONE:
			if nested > 0 { nested -= 1
				l.p.forwardLine(ctx)
				continue
			}

			l.p.next(ctx, true) // done
			l.p.linend(ctx)

			state, savedStop := l.p.scanner.scanstate, l.p.stop
			l.p.stop, t.endPos, t.end = pos, pos, &state

			var dd bool
			var dps []*diag_point
			if false && checkpoints {
				dd = strings.HasSuffix(l.project.spec.String(), "testdata/template") ||
					strings.HasSuffix(l.project.spec.String(), "testdata/template/foreach")
				if dd { for _, p := range params {
					dps = append(dps, _f("%v: %v", p.p, p.n))
					for i, a := range p.a {
						dps = append(dps, _f(" %d. '%s' %v", i, a.name, a.elems))
					}
				} }
			}

			var num int
			for _, _p := range params {
				if _p.n > 0 {
					if num == 0 {
						num = _p.n
					} else {
						num *= _p.n
					}
				}
			}

		outer:
			for n := 0; n < num; n += 1 {
				for _i, _p := range params {
					var i = n

					if _p.n == 0 {
						continue outer
					}

					for k := len(params) - 1; k > _i; k-- {
						if params[k].n > 0 {
							i /= params[k].n
						}
					}
					i %= _p.n

					for _, a := range _p.a {
						if i < len(a.elems) {
							// a.name is already a Symbol!
							ac.set(&ac, defStatic, a.name, a.elems[i])
							if checkpoints {
								if dd { dps = append(dps, _f("%d/%d: '%v' %v", n, num, a.name, a.elems[i])) }
							}
						} else if opts.skipNil {
							continue outer
						} else {
							ac.set(&ac, defStatic, a.name, &null{valbase{_p.p}})
							if checkpoints {
								if dd { dps = append(dps, _f("%d/%d: ''%v' <null>", n, num, a.name)) }
							}
						}
					}
				}

				var _trivial = len(ac.defs) == 0
				if !_trivial {
					for _, d := range ac.defs {
						if _trivial = isTrivial(d.value); !_trivial { break }
					}
				}
				if !_trivial {
					l.codeblock(&ac, t)
				}
			}
			if checkpoints { if len(dps) > 0 { debug(pc(ctx,l.p), dps) } }

			// Jump the parser past the 'done' token so the
			// rest of the file can be parsed safely!
			l.p.scanner.scanstate = *t.end
			l.p.stop = savedStop
			return

		default:
			l.p.forwardLine(ctx)
		}
	}
}

var d_variant_target bool
var pprofCounter int

func (l ul) codeblock(ctx *automatic, t *template) {
	l.p.scanner.scanstate = t.state

	if false && checkpoints {
		pprofCounter += 1
		defer cpu_profile(ctx, fmt.Sprintf("template-%05d.prof", pprofCounter), true)()
	}

	if !(l.p.pos < l.p.stop) {
		erro(ctx, "bad range: [%v %v) (%v)", l.p.pos, l.p.stop, t.name)
	}

	var c = codeblock{ctx}

	d_variant_target = false//strings.HasSuffix(l.project.spec, "modules/variant/.target")
	for l.p.tok != EOF && l.p.pos < l.p.stop {
		if l.p.tok == SPACE || l.p.tok == LINEND || (l.p.tok == COMMENT && l.p.lineComment != nil) {
			l.p.next(ctx, true)
		} else {
			if d_variant_target {
				if t.name != symEmpty {
					prompt(ctx, "%s: codeblock: %v: %v\n", l.p.pos, t.name, l.p.lit)
				} else if l.p.tok == FOR {
					prompt(ctx, "%s: codeblock: %v\n", l.p.pos, l.p.lit)
				}
			}
			l.clause(&c)
		}
	}
	d_variant_target = false
}

func (l ul) repeat(ctx Context, t *template, params []Value) {
	if false {
		pprofCounter += 1

		var (
			profCpu = fmt.Sprintf("template-%05d.cpu.prof", pprofCounter)
			profMem = fmt.Sprintf("template-%05d.mem.prof", pprofCounter)
			fCpu *os.File
			e error
		)
		if fCpu, e = os.Create(profCpu); e != nil {
			erro(ctx, "%v", ts(e))
		} else if e = pprof.StartCPUProfile(fCpu); e != nil {
			fCpu.Close()
			erro(ctx, "%v: %v", profCpu, e)
		}
		defer func() {
			pprof.StopCPUProfile()
			fCpu.Close()

			var fMem, e = os.Create(profMem)
			if e != nil {
				erro(ctx, "%v", e)
			}

			runtime.GC() // update memory statistics
			e = pprof.WriteHeapProfile(fMem)
			fMem.Close()

			if e != nil {
				erro(ctx, "%v: %v", profMem, e)
			}
		} ()
	}

	// Note: If you added 'sym' to your parser struct (l.p.sym),
	// you may want to add it to this state-restoration defer as well!
	defer func(t time.Time, pos Pos, tok token, lit string, state scanstate) {
		l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate = pos, tok, lit, state
	} (time.Now(), l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate)

	var ac = automatic{Context:ctx, defs:make(def_map)}

	for i, v := range t.params {
		// CRITICAL FIX: Use the 1-line __symbol wrapper to grab the integer ID instantly!
		if sym := __symbol(ctx, v); sym != symEmpty {
			var arg Value
			if i < len(params) {
				arg = params[i]
			} else {
				arg = _null(v.Pos())
			}

			// NATIVE INTEGER ROUTING: Bind the argument to the local scope using the Symbol
			ac.set(&ac, defStatic, sym, arg)
		} else {
			erro(ctx, "empty template param name: %v", ts(v))
		}
	}

	l.codeblock(&ac, t)
}

func (l ul) call(ctx Context, name Symbol, args []Value) (result bool) {
	for _, t := range l.p.templates {
		if t.name != symEmpty && t.name == name {
			stop := l.p.stop
			l.p.stop = t.endPos
			l.repeat(ctx, t, args)
			l.p.stop = stop
			return true
		}
	}

	erro(ctx, "undefined template: %v", name)
	return
}

func (l ul) saveConfiguration(ctx Context) {
	if l.project == nil { erro(ctx, "nil project", callstack{num:32}) }

	var configs = l.project.configs
	if configs == nil { return }

	var f = l.project.configuration_sm(ctx)

	// =========================================================
	// I/O OPTIMIZATION & STATE ENFORCEMENT
	// =========================================================
	if f != nil && f.filebase != nil {
		if f._dirty == 0 && f.exists() {
			// Enforce strict state transitions: Catch unwarranted saves!
			// erro(pc(ctx,f.fullname()), "%s: conflicted configuration.sm", l.project.name)
		}
		if !f._updated && f.exists() {
			return // Cache is in perfect sync with disk. Bail out instantly!
		}
	}

	if l.promptEnteringDirectory {
		l.promptEnteringDirectory = false
		promptLeavingDirectory(ctx, l.project.absPath)
		flush(ctx)
	}

	var fn = f.fullname()

	if checkpoints {
		if c := l.project.configuration; c != nil && c.fullname() != fn {
			erro(pc(pc(ctx,fn),c.fullname()), "%s: configuration already loaded", l.project.name)
		}
	}

	if e := os.MkdirAll(filepath.Dir(fn), os.FileMode(0755)); e != nil {
		erro(pc(ctx,fn), "make path %s failed: %v", filepath.Dir(fn), e)
	}

	// =========================================================
	// ATOMIC WRITE: Protect against concurrent '1:1: syntax error'
	// =========================================================
	var tmpFn = fn + ".tmp"
	var o, e = os.OpenFile(tmpFn, os.O_RDWR | os.O_CREATE | os.O_TRUNC, os.FileMode(0600))
	if e != nil {
		erro(pc(ctx,fn), "%s: %v", l.project.name, e)
		return
	}
	defer func() {
		o.Close() // Must close before rename/remove
		if 0 < diagCount(ctx, diagError) {
			os.Remove(tmpFn)
		} else {
			// Instantly swap the file. Readers NEVER see a truncated/empty file!
			os.Rename(tmpFn, fn)
		}
	} ()

	fmt.Fprintf(o, "# %s (%s)\n", l.project.name, l.project.spec)

	for _, c := range configs {
		fmt.Fprintf(o, "configure %s =", c.name)

		// =========================================================
		// SAFE SERIALIZATION: Prevent `{}` and `[]` parsing crashes
		// =========================================================
		if c.value != nil && !isTrivial(c.value) {
			if s := c.value.String(); s != "" && s != "{}" && s != "[]" {
				fmt.Fprintf(o, " %s", s)
			}
		}
		fmt.Fprintf(o, "\n")
	}

	fmt.Fprintf(o, "\n# %d configs.\n", len(configs))

	// Reset the flags now that it's successfully flushed to disk!
	if f.filebase != nil {
		f._updated = false
		f._dirty = 0 // Or false, depending on your dirty type
	}

	l.project.configuration = f // saved configuration.sm
}

func (l ul) configure_set(ctx Context, name string, vals ...Value) (d *def, isNew bool) {
	if d, isNew = l.project._def(ctx, defConfig, name, vals...); d != nil && isNew {
		l.project.configs = append(l.project.configs, d)
	}
	return
}

func promptEnteringDirectory(ctx Context, s string) *diagpoint {
	return prompt(ctx, "smake: Entering directory '%s'\n", s)
}

func promptLeavingDirectory(ctx Context, s string) *diagpoint {
	return prompt(ctx, "smake: Leaving directory '%s'\n", s)
}

var rxConfigRuleHeaders  = regexp.MustCompile(`^\-headers\-`)
var rxConfigRuleFunction = regexp.MustCompile(`^\-function\-`)
var rxConfigRuleSymbol   = regexp.MustCompile(`^\-symbol\-`)
var rxConfigIgnoreErrors_function = []*regexp.Regexp{
	regexp.MustCompile(`call to undeclared library function '(.+?)' with type '(.+?)'; *(.+)`),
	regexp.MustCompile(`use of undeclared identifier '(.+?)'`),
}

func configure_ignore(ctx Context, rx *regexp.Regexp, s [][]byte) (_ bool) {
	switch rx {
	case rxCodeLinePanic:
		for _, t := range rxConfigIgnoreErrors_function {
			if t.Match(s[5]) { return true }
		}
		debug(ctx, "%s %s", rx, do(ctx, is_rule{rxConfigRuleFunction}))
	case rxIgnoringDirectory, rxLdManyMinVersions:
		if false { debug(pc(ctx,s[2]), "%s", s[1]) }
		return true
	}
	if false {
		debug(ctx, "%s %s", rx, s[0])
	}
	return
}

// CRITICAL FIX: Upgraded return signature to map[Symbol]Value
func (l ul) configure_par(ctx Context, _op Value) (op Value, par map[Symbol]Value) {
	var args []Value

	op, par = _op, make(map[Symbol]Value)

	if x, y := op.(*argumented); y {
		if f, y := x.Value.(flag); y {
			op = f.Value
		} else {
			erro(pc(ctx,x.Value), "wrong configure word: %v", ts(x.Value,ctx))
		}
		args = xmerge(_final(ctx), x.args...)
	}

	for _, arg := range args {
		switch t := arg.(type) {
		case *pair:
			// 1. Fast-path extraction for the pair's key!
			sym := __symbol(ctx, t.key)
			if sym == symEmpty {
				erro(pc(ctx, t.key), "empty parameter key: %v", ts(t.key, ctx))
			}
			par[sym] = t

		case *raw, *strlit, *strval, *strcomp:
			// 2. Synthesize the "INFO" key completely in the integer domain!
			// (If you use this often, consider adding symInfo to your constants)
			sym := intern("INFO")
			par[sym] = &pair{_word(t.Pos(), sym), t}

		default:
			if !isTrivial(arg) {
				erro(pc(ctx,arg), "wrong arg: %s", ts(arg,ctx))
			}
		}
	}
	return
}

type is_configure struct{}
type is_configure_ignore struct{ rx *regexp.Regexp ; s [][]byte }
type p_configure struct{ Context }
func (p p_configure) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_configure) inner() Context { return p.Context }
func (p p_configure) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case is_configure: return true
	case is_configure_ignore:
		if configure_ignore(ctx, t.rx, t.s) { return true }
	}
	return p.Context.do(ctx, op)
}

// CRITICAL FIX: The signature MUST accept map[Symbol]Value from configure_par!
func (l ul) configure_val(ctx *execution, _op, op, val Value, par map[Symbol]Value) (res Value) {
	// 1. Extract the integer Symbol natively!
	opSym := __symbol(ctx, op)
	if opSym == symEmpty {
		erro(pc(ctx,_op), "wrong configure word: %v %v %v", ts(_op,ctx), ts(op,ctx), ts(val,ctx))
	}

	// 2. NATIVE INTEGER ROUTING! (Fixes the string-to-Symbol compiler mismatch)
	switch opSym {
	case symAnswer:
		if val == nil { return _answer(op.Pos(), false) }
		return _answer(val.Pos(), __true(ctx, val))
	case symBool, symBoolean: // (Assuming you added symBoolean to your constants)
		if val == nil { return _boolean(op.Pos(), false) }
		return _boolean(val.Pos(), __true(ctx, val))
	case symValue:
		if val == nil { return _null(op.Pos()) }
		return expand(_final(ctx),val)
	}

	if l.project.configure == nil {
		erro(pc(ctx,op), "wrong configure: %v %v", ts(op,ctx), ts(val,ctx))
	}

	ops := l.project.configure._entries(ctx, _op, false)
	if ops == nil {
		erro(pc(ctx,_op), "no configure ops: %v", _op)
	}

	var vals []Value
	for _, ent := range ops {
		var params []Value
		for _, prog := range ent.programs() {
			for _, p := range prog.params {

				// 3. ZERO-ALLOCATION FIX: Extract param symbol natively!
				sym := __symbol(ctx, p)
				w := _word(p.Pos(), sym)

				// 4. O(1) Integer Map Lookup!
				if x, ok := par[sym]; ok {
					params = append(params, x)
				} else {
					// 5. Integer Switch!
					// (You can also define symTarget, symLanguage in your constants later)
					switch sym {
					case intern("TARGET"):
						// Eagerly expand the target value from the calling context
						// so it doesn't dynamically fetch the local rule's overwritten `@` later!
						targetVal := expand(ctx, auto_get(ctx, symAt))
						params = append(params, &pair{w, targetVal})
					case intern("VALUE"):
						params = append(params, &pair{w, val})
					case intern("LANG"), intern("LANGUAGE"):
						if ctx.language != symEmpty {
							// 6. Cross the boundary cleanly: Intern the ctx.language string!
							params = append(params, &pair{w, _word(w.Pos(), ctx.language)})
						}
					}
				}
			}
		}
		vals = append(vals, ent.execute(p_configure{ctx}, params...)...)
	}
	return ease(ctx, vals)
}

func (l ul) configure(ctx Context) {
	l.p.next(ctx, true) // aka CONFIGURE
	if l.p.tok == LPAREN {
		l.p.next(ctx, true)

		for l.p.tok != EOF {
			l.p.spaces(ctx)

			for l.p.tok == LINEND || l.p.lineComment != nil {
				l.p.linend(ctx)
				l.p.spaces(ctx)
			}

			if l.p.tok == RPAREN {
				l.p.next(ctx, true)
				break
			}

			l.configure1(ctx)

			l.p.spaces(ctx)
			if l.p.tok == RPAREN {
				l.p.next(ctx, true)
				break
			}

			if l.p.tok == LINEND || l.p.lineComment != nil {
				l.p.linend(ctx)
			}
		}
	} else {
		l.configure1(ctx)
	}
}

func (l ul) configure1(ctx Context) {
	ids := []Value{}
	for _, v := range merge(l.expr(ctx)) {
		// CRITICAL FIX: Protect LHS configuration variables from early evaluation
		forids(ctx, expand(def_name{_final(ctx)}, v), func(v Value, _ []Value) { ids = append(ids, v) })
	}

	l.p.spaces(ctx)

	var _op Value
	var _no_cond bool
minusloop:
	for l.p.tok == MINUS {
		t := l.expr(ctx)
		l.p.spaces(ctx)

		switch t := t.(type) {
		case *argumented:
			if x, y := t.Value.(flag); y {
				switch x.Value.String() {
				case "cond":
					for _, a := range xmerge(_final(ctx), t.args...) {
						if !__true(ctx, a) {
							_no_cond = true
							break
						}
					}
					continue minusloop
				}
			}
		case flag:
			if t.Value.String() == "cond" {
				erro(pc(ctx,t), "needs cond value")
			}
		}

		if _op != nil {
			erro(pc(ctx,t), "configure op already defined: %v", _op)
		}

		_op = t
		break
	}

	op, par := l.configure_par(ctx, _op)
	exe := execution{
		automatic:automatic{Context:pc(ctx,op), defs:make(def_map)},
		start:time.Now(), proj:l.project,
	}

	if l.p.tok == ASSIGN || l.p.tok == SEMICOLON {
		l.p.next(ctx, true) // skips the '=' or ';' token

		pos, tok, lit, sst := l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate
		for _, id := range ids {
			d, _ := exe.set(&exe, defVoid, symAt, id)
			d.pos = id.Pos()

			d, isNew := l.configure_set(ctx, __string(ctx, id)) // aka. l.project.set
			d.pos = id.Pos()

			isCached := !isNew && d.value != nil
			if isCached && isTrivial(d.value) {
				isCached = false // Force the engine to re-execute the recipe.
				d.value = nil // Safe override! Clear the failed state.
			}

			cc := defval{original{&exe, defConfig|defExpand1}, d}
			l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate = pos, tok, lit, sst
			l.p.dialect = symEmpty

			var vals []Value
			newVal := ease(ctx, expands(cc, l.values(cc)...))

			// Force type conversion before caching check (e.g. {=true} -> {=yes})
			if op != nil { newVal = l.configure_val(&exe, _op, op, newVal, par) }

			// =========================================================
			// CRITICAL FIX: Safe Cache Bypass!
			// =========================================================
			if isCached && equal(ctx, d.value, newVal) {
				if l.promptCachedConfigs(ctx) {
					prompt(ctx, "%v:info: cached %v\n", do(ctx, get_fatpos{d.pos}), d)
					flush(ctx)
				}
				l.p.lineComment = nil
				continue // Match found! Bypass prompt and execution completely.
			}

			if x, y := par[intern("INFO")]; y {
				if !l.promptEnteringDirectory {
					l.promptEnteringDirectory = true
					promptEnteringDirectory(ctx, l.project.absPath)
				}

				var s string
				if p, y := x.(*pair); y {
					s = __string(ctx, p.val)
				} else {
					s = __string(ctx, x)
				}

				a := prompt(pc(ctx,op), "%s …", s)
				defer func(i int) {
					if diagCount(ctx, diagInfo, diagWarn, diagError) <= i {
						s = __string(ctx, ease(ctx, vals))
						s = strings.Replace(s, "\n", "\\n", -1)

						b := prompt(ctx, "… %s\n", s)
						flush(ctx)

						if checkpoints { l.configure_val_check(&exe, d.name, op, vals, a, b) }
					}
				} (diagCount(ctx, diagInfo, diagWarn, diagError))
			}

			if newVal != nil { vals = append(vals, newVal) }

			// CRITICAL FIX: Empty the config value before setting to avoid panic on stale cache overwrite!
			d.value = nil
			d.set(ctx, newVal)

			if f := l.project.configuration_sm(ctx); f.filebase != nil && d.value != nil {
				f._updated = true
				f._dirty += 1
			}

			l.p.lineComment = nil
		}
		return
	} else if l.p.tok == COLON || l.p.is_end_of_line() {
		if l.p.tok == COLON { l.p.next(ctx, true) }

		pos, tok, lit, sst := l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate
		for _, id := range ids {
			d, _ := exe.set(&exe, defVoid, symAt, id)
			d.pos = id.Pos()

			d, isNew := l.configure_set(ctx, __string(ctx, id)) // aka. l.project.set
			d.pos = id.Pos()

			isCached := !isNew && d.value != nil
			if isCached && isTrivial(d.value) {
				isCached = false // Force the engine to re-execute the recipe.
				d.value = nil // Safe override! Clear the failed state.
			}

			cc := defval{original{&exe, defConfig|defExpand1}, d}
			l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate = pos, tok, lit, sst
			l.p.dialect = symEmpty

			var deps, vals []Value

		depsloop:
			for {
				switch l.p.tok { case SEMICOLON, LINEND, EOF: break depsloop }
				deps = append(deps, l.expr(cc)) ; l.p.spaces(ctx)
				exe.set(&exe, defVoid, symLangle, deps[0])
				exe.set(&exe, defVoid, symRangle, deps[len(deps)-1])
				exe.set(&exe, defVoid, symCaret, ease(ctx, deps))
			}

			exe.language = l.p.dialect

			if l.p.tok == SEMICOLON { // ;
				exe.recipes = append(exe.recipes, l.recipe(cc)...)
			} else {
				l.p.scanner.recipes(true) // turn on recipes before LINEND.
				if l.p.linend(ctx) { // take the new line.
					for l.p.recipe_start() {
						exe.recipes = append(exe.recipes, l.recipe(cc)...)
					}
				}
				l.p.scanner.recipes(false)
			}

			// Bypass Prompt & Recipes if valid cached value exists
			if isCached {
				if l.promptCachedConfigs(ctx) {
					prompt(ctx, "%v:info: cached %v\n", do(ctx, get_fatpos{d.pos}), d)
					flush(ctx)
				}
				continue
			}

			if x, y := par[intern("INFO")]; y {
				if !l.promptEnteringDirectory {
					l.promptEnteringDirectory = true
					promptEnteringDirectory(ctx, l.project.absPath)
				}

				var s string
				if p, y := x.(*pair); y {
					s = __string(ctx, p.val)
				} else {
					s = __string(ctx, x)
				}

				a := prompt(pc(ctx,op), "%s …", s)
				defer func(i int) {
					if diagCount(ctx, diagInfo, diagWarn, diagError) <= i {
						s = __string(ctx, ease(ctx, vals))
						s = strings.Replace(s, "\n", "\\n", -1)

						b := prompt(ctx, "… %s\n", s)
						flush(ctx)

						if checkpoints { l.configure_val_check(&exe, d.name, op, vals, a, b) }
					}
				} (diagCount(ctx, diagInfo, diagWarn, diagError))
			}

			if _no_cond {
				d.set(cc, _null(id.Pos()))
				continue
			}

			for _, exe.prerequisite = range deps { traverse(&exe, exe.prerequisite) }

			var val = auto_get(&exe, symDash)
			if val == nil && exe.recipes != nil && len(exe.interpreted) == 0 {
				if x, y := dialects[symEmpty]; y && x != nil {
					val = exe.interpret(cc, x, nil)
				}
			}

			if op != nil { val = l.configure_val(&exe, _op, op, val, par) }
			if val != nil { vals = append(vals, val) }

			for _, a := range exe.defers {
				if x, y := a.(*group); y {
					modify(ctx, x, true)
				} else {
					erro(pc(ctx,a), "defer: not a modifier: %s", ts(a))
				}
			}

			// Empty the config value before setting to avoid panic on stale cache overwrite!
			d.value = nil
			d.set(cc, ease(ctx, vals))

			if f := l.project.configuration_sm(ctx); f.filebase != nil && d.value != nil {
				f._updated = true
				f._dirty += 1
			}
		}
		return
	} else {
		erro(pc(ctx,l.p), "%v: wrong configure", op)
	}
}

func (l ul) clause(ctx Context) {
	if l_traverse.enabled {
		defer un(l_tracef(l_traverse, "clause(%v, %v)", l.p.tok, l.p.pos))
	}

	l.p.spaces(ctx)

	if l.p.tok == LINEND || (l.p.tok == COMMENT && l.p.lineComment != nil) {
		l.p.next(ctx, true)
		return // noop clause
	}

	switch t := l.p.tok ; t {
	case   INCLUDE: l.spec(ctx, t, l.p.expect(ctx, t), l.include) ; return
	case    ASSERT: l.spec(ctx, t, l.p.expect(ctx, t), l.p.assert); return
	case    APPEND: l.spec(ctx, t, l.p.expect(ctx, t), l.p.append); return
	case     FILES: l.spec(ctx, t, l.p.expect(ctx, t), l.files)   ; return
	case     LOCAL: l.spec(ctx, t, l.p.expect(ctx, t), l.local)   ; return
	case      EVAL: l.spec(ctx, t, l.p.expect(ctx, t), l.eval)    ; return
	case       DEF: l.def_end(ctx)     ; return
	case       FOR: l.for_done(ctx)    ; return
	case   FOREACH: l.foreach_done(ctx); return
	case CONFIGURE: l.configure(ctx)   ; return
	case USE, TEMPLATE:
		erro(ctx, "unexpected %v", t)
	}

	var vals []Value

	for l.p.tok != LINEND && l.p.tok != EOF {
		var x = l.expr(left_side{ctx})

		l.p.spaces(ctx)

		vals = append(vals, x)
		if d_variant_target {
			prompt(ctx, "%s: clause: %v %v\n", l.p.pos, x, l.p.tok)
		}

		if l.p.tok.is_assign() {
			l.assign(ctx, vals)
			return
		}

		if l.p.tok.is_rule_delim() {
			l.rule(ctx, vals)
			return
		}
	}

	for _, v := range vals {
		if x, y := v.(*argumented); y {
			l.call(ctx, __symbol(ctx, x.Value), x.args)
		} else {
			erro(pc(ctx,v), "unexpected %v", ts(v,ctx), trace{})
		}
	}
}

type project_opts struct{
	configure Value `configure` // detects dot_configure if empty
	traveUseLoop bool `break,loop` // don't recursively use this project
	multiUseAllowed bool `multi`  // this project is used multiple times
}

// project returns a new project for the given project path and name;
// the name must not be the blank identifier.
// The project is not complete and contains no explicit imports.
// CRITICAL FIX: Upgraded name and declares map to use Symbol!
func (l ul) declareNew(ctx Context, pos Pos, name Symbol, filename string, opts *project_opts) (d *declare) {
	if x, y := l.declares[name]; y { return x }

	var sco = l.scope()
	// These predefined names from scope are safe to extract as strings for path logic.
	var relPath = __string(ctx, sco.finddef(symDot))
	var tmpPath = __string(ctx, sco.finddef(symComma))
	var absPath string
	if x, y := do(ctx, abs_path{}).(string); y {
		absPath = x
	} else {
		absPath = __string(ctx, sco.finddef(symSlash))
	}

	var spec, _ = filepath.Rel(baseWorkDir, absPath)

	if l.declares == nil { l.declares = make(map[Symbol]*declare) }

	d = &declare{
		project: &project{
			pos:      pos,
			absPath:  absPath,
			tmpPath:  tmpPath,
			rel:      intern(relPath),
			spec:     intern(spec),
			name:     name, // CRITICAL FIX: Native Symbol assignment!
			opt:      *opts,
			use:      new(uselist),
		},
	}

	l.declares[name] = d
	l.globe.loaded[d.absPath] = d.project

	do(ctx, declared_project{d.project})

	d.p = l.p
	d.s = l.loader.scope

	// name is safely passed as the scope name
	d.scope = new_scope(sco, d.project, name.String())
	d.scope.elems[intern(".self")] = self{d.project} // Map keys in def_map must be Symbol!
	d.scope.elems[intern(".usee")] = d.use
	d.use.owner_ = d.project
	d.use.scope = d.scope
	d.use.name = intern("usee") // Also upgraded if use.name requires it

	if l.globe.main == nil && spec != "" && name != symAt && name != symTilde {
		for sco != nil && sco != l.globe.scope {
			if p := sco.project; p != nil && d.name == symAt { // Assumes d.name is Symbol
				return
			}
			sco = sco.outer
		}
		l.globe.main = d.project
	}
	return
}

func (l ul) declare(ctx Context, ident Value, name Symbol, filename string, declOpts *project_opts) bool {
	if name == symAt {
		erro(ctx, "deprecated project name: @")
	}

	if _, o := l.find(name); o != nil { switch o.(type) {
	case *builtin: erro(ctx, "%v is a builtin, can't be project name", o)
	}}

	var prev = l.loader
	var dec = l.declareNew(ctx, ident.Pos(), name, filename, declOpts)
	if prev == nil || dec.project != prev.project {
		if prev != nil && prev.project != nil && prev.project != dec.project {
			if prev.project.name == dec.project.name {
				erro(ctx, "%s", prev.project.name) // name is a Symbol, %s calls String() safely
			}
		}
		l.project, l.loader.scope = dec.project, dec.scope
	}

	if ll := _loader(l.loader.Context); ll != l.loader && ll == prev {
		if _, a := ll.project.projectName(ctx, name, dec.project); a != nil { // projectName should accept Symbol
			if x, y := a.(*project); !y || x != dec.project {
				erro(ctx, "%v: name already taken : %v", name, ts(a))
			}
		}
	}

	if l.globe.main != nil && l.globe.main == l.project && l.project.name != symTilde {
		for _, t := range l.globe.pairs {
			switch k := t.key.(type) {
			case *word, *compound:
				// Ensure def accepts a Symbol, extract it natively!
				l.scope().def(ctx, defDecl, __symbol(ctx, k), t.val)
			case flag:
				if false { debug(ctx, "unknown flag : %v", t) }
			default:
				debug(ctx, "unknown target : %v", ts(t))
			}
		}
	}

	if x := try[[]Value](ctx,get_args{}); len(x) != 0 {
		for _, arg := range merge(x...) {
			switch t := arg.(type) {
			case *pair:
				switch k := t.key.(type) {
				case *word, *compound:
					l.scope().def(ctx, defDecl, __symbol(ctx, k), t.val)
				case flag:
					if false { debug(ctx, "unknown flag : %v", t) }
				default:
					debug(ctx, "unknown target : %v", ts(t))
				}
			}
		}
	}

	if err := l.loadPlugin(ctx); err != nil {
		erro(ctx, "load plugin failed: %v", err)
	}
	return l.project != nil
}

func (l ul) pre_project(ctx Context, op string, args ...Value) (_ bool) {
	switch op {
	case "set":
		l.pre_project_set(ctx, merge(args...)...)
		return true
	}
	return
}

func (l ul) pre_project_set(ctx Context, args ...Value) {
	var s = l.scope()
	for _, a := range args {
		switch t := a.(type) {
		case *pair:
			s.def(ctx, defVoid, t.key, t.val)
		default:
			erro(ctx, "unknown set: %v", ts(a))
		}
	}
}

type declared_project struct{ *project }

type parent struct{ Context ; *project }
func (p parent) cast(t reflect.Type) Context { return icast(p,t) }
func (p parent) inner() Context { return p.Context }
func (p parent) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case declared_project:
		if t.has_base(p.project) {
			if true { return }

			prompt(ctx, "%s: %s : %s\n", p.absPath, p.project, t.loop_base_path(ctx, p.project, ""))
			if true {
				debug(ctx, "recursive derivation: %v ⇔ %v", ts(p.project), ts(t.project))
				return
			} else {
				erro(ctx, "recursive derivation: %v ⇔ %v", ts(p.project), ts(t.project))
			}
		}

		if p.has_base(p.project) {
			erro(ctx, "duplication derivation: %v ⇔ %v", ts(p.project), ts(t.project))
		}

		if len(p.bases) == 0 {
			p.projectName(ctx, symDotBase, t.project)
		}

		p.bases = append(p.bases, t.project)
		return
	}
	return p.Context.do(ctx, op)
}

func (l ul) projectStart(ctx Context, filename string, isMainFile bool) (_ Value, _ Symbol, _ bool) { // Return type upgraded to Symbol!
	l.p.next(ctx, true)

	var vals []Value
	for l.p.tok == MINUS {
		val := l.expr(ctx)
		l.p.spaces(ctx)

		if a, y := val.(*argumented); y {
			if f, y := a.Value.(flag); y {
				if w, y := f.Value.(*word); y {
					// Pre-project flags (e.g. -j4). Pass the fast-path string/symbol.
					l.pre_project(ctx, w.s.String(), a.args...)
					continue
				}
			}
		}

		vals = append(vals, val)
	}

	var opts project_opts
	if a := parseOpts(ctx, &opts, vals...); len(a) > 0 {
		erro(pc(ctx,filename), "unknown project option %v", ts(a))
	}

	var ident Value
	var implicitBase string

	if l.p.tok == LPAREN || l.p.is_end_of_line() {
		var dir = filepath.Dir(filename)
		if l.project != nil && l.project.absPath == dir {
			ident = _word(l.p.pos, l.project.name) // l.project.name is natively a Symbol!
		} else if s := filepath.Base(filename); s == dot_base || s == dot_configure {
			ident = _word(l.p.pos, intern(s)) // Cross the boundary: intern the string!
		} else if s := filepath.Base(dir); s != "" {
			ident = _word(l.p.pos, intern(s)) // Cross the boundary
		} else {
			erro(ctx, "invalid file: %v", filename)
		}
	} else if l.p.tok == TILDE {
		if ext := filepath.Ext(filename); ext != ".smart" {
			erro(ctx, "`%v` not a smart file", filepath.Base(filename))
		} else if s := strings.TrimSuffix(filepath.Base(filename), ext); s == "" {
			erro(ctx, "`%v` not tilde name", filepath.Base(filename))
		} else {
			ident = _word(l.p.pos, intern(s)) // Cross the boundary
		}
		l.p.next(ctx, true)
	} else {
		base, qw := makePath(), &qualword{}

		for l.p.tok != EOF && l.p.tok != SPACE {
			var w = l.p.bare(ctx)
			qw.elems = append(qw.elems, w)
			if l.p.tok == DOT {
				l.p.step(ctx)
				base.elems = append(base.elems, w)
			} else {
				break
			}
		}

		l.p.spaces(ctx)

		switch qw.len() {
		case 0:
			erro(pc(ctx,qw), "package name is empty (tok=%v)", l.p.tok)
		case 1:
			ident = qw.elems[0]
		default:
			ident = qw
		}

		if 0 < base.len() {
			implicitBase = __string(ctx, base)
		}
	}

	// CRITICAL FIX: Extract the project name as a Symbol using our wrapper!
	var name = __symbol(ctx, ident)

	if name == symDash || name == symUnderscore {
		erro(ctx, "package name '%s' is preserved", name)
	}

	if p := l.project; p != nil && p.name != name {
		erro(ctx, "%v: multiple projects in the directory : %v", p, ident)
	}

	var _, prevDeclared = l.declares[name]
	if l.declare(ctx, ident, name, filename, &opts) {
		isMainFile = isMainFile && !prevDeclared;
	}

	if cc := (parent{ctx, l.project}); l.p.tok != LPAREN {
		l.bases(cc, implicitBase)
	} else {
		var cc0 = p_group_ctx{aware(ctx,COMMA)}
		for l.p.tok != EOF {
			for l.p.next(ctx, true); !l.p.is_list_term(ctx); l.p.spaces(ctx) {
				l.bases(cc, "", merge(l.expr(cc0))...)
			}
			if l.p.tok != COMMA { break }
		}
		l.p.expect(ctx, RPAREN)
	}

	if l.p.spaces(ctx) ; l.p.tok != EOF { l.p.linend(ctx) }

	if isMainFile {
		l.configuration(ctx, ident, name)
		l.container(ctx, ident, name.String())
	}
	return ident, name, isMainFile
}

func (l ul) close_project(ctx Context, name Symbol) {
    var x, y = l.declares[name]

	if !y || x == nil {
		erro(ctx, "undeclared project: %v", name)
	}

    if l.project == nil {
        erro(ctx, "current project unset")
    }

    if l.project.name != name {
        erro(ctx, "current project is %s, not %s", l.project, name)
    }

    if l.project != x.project {
        erro(ctx, "project conflicts (%v, %v)", l.project, x.project)
    }

    l.p, l.loader.scope = x.p, x.s
}

func (l ul) parse(ctx Context, filename string) (_ bool) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "file '"+filename+"'")) }
	if l.traceLaunch { defer un(l_trace(l_launch, "parse_file")) }
	if checkpoints {
		if s := l.p.scanner.file.Name(); filename != s {
			erro(ctx, "%v: %s != %s", l.project, filename, s)
		}
	}

	if l.p.tok == EOF {
		erro(pc(ctx,l.p), "early end of file", callstack{num:64})
		return
	}

	var abs string
	var isMainFile bool // aka do.smart, build.smart
	var flatmode = truly(ctx, is_flat_mode{})

	if flatmode {
		if l.project == nil {
			erro(pc(ctx,filename), "nil project", callstack{num:64})
		} else {
			abs = l.project.absPath
		}
	} else {
		switch filepath.Base(filename) {
		case dot_base, dot_configure:
			abs = filename
		case mainFileName, deprFileName:
			abs = filepath.Dir(filename); isMainFile = true
		default:
			abs = filepath.Dir(filename)
		}
	}

	ctx = pc(ctx, filename, 1, 1)

	var rel, _ = filepath.Rel(l.workdir, abs)
	var tmp    = joinTmpPath(ctx, l.workdir, rel)

	if s := l.scope(); /* p == nil || */ s == nil {
		erro(ctx, "%v: nil scope: %v", l.project, s)
	}

	defer l.closescope(l.openscope(bases(2, filename, true)))

	if flatmode {
		if l.p.tok == PROJECT {
			erro(pc(ctx,l.p), "project is forbidden in flat file")
		}
	} else {
		// CWD: Current Work Directory,     TODO: use $:cwd:
		// CTD: Current Temp Directory,     TODO: use $:ctd:
		// CRD: Current Relative Directory, TODO: use $:crd:
		var s = l.scope()
		if d := s.def(ctx, defVoid, symSlash, _pathStr(ctx, abs)); d != nil { s.alias(ctx, d, symCWD) }
		if d := s.def(ctx, defVoid, symDot, _pathStr(ctx, rel)); d != nil { s.alias(ctx, d, symCRD) }
		if d := s.def(ctx, defVoid, symComma, _pathStr(ctx, tmp)); d != nil { s.alias(ctx, d, symCTD) }
		if l.p.tok == PROJECT {
			var name Symbol
			var prev = l.project
			_, name, isMainFile = l.projectStart(ctx, filename, isMainFile)
			if prev != l.project { defer l.close_project(ctx, name) }
		} else {
			erro(pc(ctx,l.p), "expect keyword 'project', not %v", l.p.tok)
		}
	}

	var autoload = !flatmode && isMainFile
	if  autoload { l.autoload(ctx, "declared") }

	if l.mode&ModuleClauseOnly == 0 {
		if !flatmode {
		declaration:
			for l.p.tok != EOF {
				switch t := l.p.tok ; t {
				case LINEND, SPACE: l.p.next(ctx, true)
				case USE: l.spec(ctx, t, l.p.expect(ctx, t), l.use)
				case APPEND, ASSERT, EVAL, FILES, INCLUDE: l.clause(ctx)
				default: break declaration
				}
			}
		}

		if false && autoload { l.autoload(ctx, "amid") }

		if l.mode&ImportsOnly == 0 {
			for l.p.tok != EOF { l.clause(ctx) }
		}
	}

	if autoload { l.autoload(ctx, "appendix") }

	l.clear_locals()

	return l.mode&ImportsOnly != 0 || l.p.tok == EOF
}

type get_parser struct{}

type is_implicit_load struct{}

// implicit load, e.g. via foo.bar.Baz (implicitly loads foo/bar for base of Baz)
type load_implicit struct{ Context }
func (p load_implicit) cast(t reflect.Type) Context { return icast(p,t) }
func (p load_implicit) inner() Context { return p.Context }
func (p load_implicit) do(ctx Context, op any) any {
    switch op.(type) {
    case is_implicit_load: return true
    }
    return p.Context.do(ctx, op)
}

type abs_path struct{}
type abs_ctx struct{ Context ; abs string }
func (p *abs_ctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p *abs_ctx) inner() Context { return p.Context }
func (p *abs_ctx) ts(string) string { return "{"+posstr(p.abs)+" "+ts(p.Context)+"}" }
func (p *abs_ctx) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case abs_path: return p.abs
    case get_position:
        var pos, _ = p.Context.do(ctx, op).(Position)
        if !pos.valid() { pos.Filename, pos.Line = p.abs, 1 }
        return pos
    }
    return p.Context.do(ctx, op)
}

func _abs_ctx(ctx Context, abs string) Context {
    if do(ctx, abs_path{}) == abs { return ctx }
    return &abs_ctx{ctx, abs}
}

type declare struct {
    *project  // save loader.project -- the active project being loading
    p *parser // save loader.p
    s *scope  // save loader.term.s
}

func _loader(c Context) *loader { return cast[*loader](c) }

type loader struct {
    term             // .s -> declare.s
    p *parser        // -> declare.p
    project *project // -> declare.project -- the current project
    promptEnteringDirectory bool

    declares map[Symbol]*declare

    mode Mode

    verpre string // verbose prefix
}
func (l *loader) inner() Context { return &l.term }
func (l *loader) cast(t reflect.Type) Context {
    if reflect.TypeOf(l) == t { return l }
    return l.term.cast(t)
}
func (l *loader) ts(string) (s string) {
    s = "{=loader"
    if l.project != nil { s += " " + l.project.name.String() }
    if l.Context != nil { s += " " + ts(l.Context) }
    s += "}"
    return
}
func (l *loader) do(ctx Context, op any) any {
    switch op.(type) {
    case is_implicit_load: return false
    case get_parser:  return l.p
    case get_project: return l.project
	case get_position: if l.p != nil { return l.p.pos } // CRITICAL FIX: Return compact Pos!
    case get_scope: if false { return l.project.scope }
    case get_closure_scopes:
        var t, _ = l.term.do(ctx, op).([]*scope)
        return append(t, l.project.scope)
    }
	return l.term.do(ctx, op)
}

type ul struct{ *universe ; *loader }

type useopts struct {
	noUse  bool `nu,nouse,uu,unuse` // TODO
    noVars bool `nv,novars,no-vars`
	files  bool `f,files` // NOTE: see also '-import(xxxx)'
	reuse  bool `r,ru,reuse,reusing`
    vars []Value `var,vars`
}

type usevar struct {
	unique  bool `uni,uniq,unique`
	reverse bool `rev,reverse`
	auto    bool `auto`
	remainder []Value
}
func (uo *usevar) apply(ctx Context, d *def, u ...*def) {
	var vals []Value
	for _, u := range u {
		for _, v := range merge(u.value) {
			if t, y := v.(*def); y && t != nil {
				vals = append(vals, merge(t.value)...)
			} else {
				vals = append(vals, v)
			}
		}
	}
	if len(vals) == 0 {
		return
	}

	// 1. Standard Append (Always add to the right)
	d.append(ctx, vals...)

	// 2. Delegate to __unique
	if uo.unique {
		var opts = uo.remainder

		// CRITICAL FIX: Synthesize the `-reverse` flag for the AST macro!
		if uo.reverse {
			// We inject the flag so `__unique` parses it and sets `ctx.reverse = true`
			opts = append(opts, _word(_pos(ctx), intern("-reverse")))
		}

		d.value = call(ctx, symUnique, opts, merge(d.value)...)
	}
}

func (l ul) usevars(ctx Context, user, usee *project) {
	for _, targetSym := range usee.exports {
		targetName := targetSym.String()
		lookupSym := intern("use." + targetName)

		// 1. Fetch the Payload from the library
		var useDef *def
		if o := usee.Lookup(lookupSym); o != nil {
			if d, y := o.(*def); y && d != nil {
				useDef = d
			}
		}

		if useDef == nil || isTrivial(useDef.value) {
			continue
		}

		// 2. Apply hardcoded properties automatically
		var op usevar
		if targetName == "-l" || targetName == "-L" || targetName == "-I" || strings.HasPrefix(targetName, "-no") {
			op.unique = true
		}
		if targetName == "-l" || targetName == "ldlibs" {
			op.reverse = true
		}

		var dd []*def

		// 3. Export downstream: use.XXX += $(use.XXX)
		{
			d, isNewDef := user._def(ctx, defVoid, lookupSym)
			if isNewDef || isTrivial(d.value) {
				dd = append(dd, nonTrivialDefsFromBase(ctx, user, lookupSym)...)
			}
			op.apply(closure_with(ctx, usee.scope), d, append(dd, useDef)...)
		}

		// 4. Apply locally: XXX += $(use.XXX)
		{
			d, isNewDef := user._def(ctx, defVoid, targetSym)
			if isNewDef && false {
				if dd == nil {
					dd = append(dd, nonTrivialDefsFromBase(ctx, user, lookupSym)...)
				}
				dd = append(dd, nonTrivialDefsFromBase(ctx, user, targetSym)...)
			}
			op.apply(closure_with(ctx, user.scope), d, append(dd, useDef)...)
		}
	}
}

// Ensure the signature matches (which you already correctly anticipated):
func nonTrivialDefsFromBase(ctx Context, p *project, name Symbol) (dd []*def) {
	for _, base := range p.bases {
		var d, y = base.resolve(ctx, name).(*def)
		if y && d != nil && !isTrivial(d.value) {
			dd = append(dd, d)
		}
	}
	return
}

func (l ul) scope() *scope { return l.loader.scope }
func (l ul) search(ctx Context, spec string) (absPath string, isDir bool) {
    if spec == "." {
        erro(ctx, "self-search is not possible")
    } else if filepath.IsAbs(spec) {
        var s = spec
        if x, y := os.Stat(s); y == nil { return s, x.IsDir() }

        s = spec + ".smart"
        if x, y := os.Stat(s); y == nil { return s, x.IsDir() }

        s = spec + ".sm"
        if x, y := os.Stat(s); y == nil { return s, x.IsDir() }
    } else if spec == "~" || strings.HasPrefix(spec, "~") {
        erro(ctx, "%v : wrong spec : %s (tilde not allowed)", l.project, spec)
    } else if spec == ".." || has_prefix(spec, "."+pathSep, ".."+pathSep) {
        var s = spec
        var sx string

        if t := l.project.absPath; t != "" {
            if x, e := os.Stat(t); e != nil {
                erro(ctx, "%v", e)
            } else if !x.IsDir() {
                t = filepath.Dir(t)
            }

            sx = filepath.Join(t, s)

            if x, e := filepath.Abs(sx); e != nil {
                erro(ctx, "%v", e)
            } else {
                s = x
            }
        }

        if x, e := os.Stat(s); e == nil { return s, x.IsDir() }

        sx = s + ".smart"
        if x, e := os.Stat(sx); e == nil { return sx, x.IsDir() }

        sx = s + ".sm"
        if x, e := os.Stat(sx); e == nil { return sx, x.IsDir() }
    } else {
        for _, base := range l.paths {
            var s = filepath.Join(base, spec)
            if !filepath.IsAbs(base) { s = filepath.Join(l.workdir, s) }
            if x, e := os.Stat(s); e == nil { return s, x.IsDir() }
        }
    }
    return
}

func (l ul) use_spec(ctx Context, opts useopts, specVal Value, params ...Value) (loaded *project) {
    var absPath, spec string
    var isDir, traveUseLoop bool
    if x, y := specVal.(*project); y {
        loaded = x
    } else if spec = __string(ctx, specVal); spec == "" {
        erro(pc(ctx,specVal), "empty spec: %v", ts(specVal))
    } else if absPath, isDir = l.search(ctx, spec); absPath == "" {
        erro(pc(ctx,specVal), "missing `%s` (in %v)", spec, l.paths)
    } else {
        loaded, y = l.globe.loaded[absPath]

        for ll := _loader(l.loader.Context); ll != nil; ll = _loader(ll.Context) {
            if ll.project.absPath == absPath {
                erro(pc(ctx,specVal), "%s: loop detected", l.project)
            }
        }
    }

    defer func() {
        if loaded == nil {
            if false {
                erro(ctx, "%v not loaded (%v,dir=%v)", spec, absPath, isDir)
            }
            return
        }

        var scope = l.project.scope
        if p, _ := scope.Lookup(loaded.name).(*project); p == nil {
            if _, alt := scope.projectName(ctx, loaded.name, loaded); alt != nil {
                if p, y := alt.(*project); !y || p == nil {
                    erro(ctx, "%s: name already taken : %s", loaded.name, ts(alt))
                }
            }
        }
    } ()

    if l.verboseImport {
        if /* len(l.loadStack) > 1 */ false {
            defer func(s string) { l.verpre = s } (l.verpre)
            l.verpre += "│"
        }
        if opts.reuse {
            prompt(ctx, "%s├┬→\"%s\" (reuse, %s)\n", l.verpre, spec, absPath)
        } else {
            prompt(ctx, "%s├┬→\"%s\" (%s)\n", l.verpre, spec, absPath)
        }
        defer func(t time.Time) {
            var name string
            var d = time.Now().Sub(t)//*time.Millisecond // µs, ms, s
            var ds = fmt.Sprintf("(%s)", d)
            if d>=1*time.Second { ds = fmt.Sprintf("▶%s◀",ds) }
            if loaded != nil { name = loaded.name.String() }
            prompt(ctx, "%s├┴─\"%s\" ⇢ %s %s\n", l.verpre, spec, name, ds)
        } (time.Now())
    }

    if loaded != nil && !(/*opts.noVars || */opts.reuse) {
        if proj, res, isb := l.project.has_loaded(ctx, loaded, traveUseLoop) ; isb {
            erro(ctx,
				_f("%v: %v is already a base\n", l.project, spec),
				_f("`%s` is already a base (proj=%s)", spec, proj))
        } else if res {
            erro(ctx,
				_f("%v: %v already imported by %v\n", l.project, spec, proj),
				_f("'%s' already imported by '%s'", spec, proj))
        }
    }

    var prev = l.project

    if loaded == nil {
        if cc := pc(ctx, specVal); isDir {
            l.directory(cc, spec, absPath, nil)
        } else {
            l.file(cc, spec, absPath, nil)
        }
        if loaded, _ = l.globe.loaded[absPath]; loaded == nil {
            erro(ctx, "%s not loaded (%s)", spec, absPath)
        }
        if loaded == l.project {
            erro(ctx, "%v : overwrote by %v (dir=%v)", prev, loaded, isDir)
        }
    }

    if checkpoints && prev != l.project {
        erro(ctx, "active project changed: %v → %v, use %v", prev, l.project, loaded)
    }

    // Check against the current load list before appending loaded.
    for _, use := range l.project.use.list {
        var up = use.project
        if loaded == up {
            if !opts.noVars && !opts.files {
                erro(ctx, "%v: using `%s` multiple times: %v", l.project, spec, l.project.use.list)
            }
            return
        }

        var proj *project
        var res, isb bool
        if proj, res, isb = loaded.has_loaded(ctx, up, traveUseLoop); isb {
            if !l.project.has_base(up) {
                erro(ctx, "`%s` is already a base", spec)
            }
        } else if res && !use.opts.reuse && !up.opt.multiUseAllowed && !loaded.opt.multiUseAllowed {
            warn(ctx, "`%s` has already imported `%s` (from %s)", loaded, up, proj)
            if loaded != up { warn(ctx, "project %s", loaded) }
            if proj != up { warn(ctx, "project %s", proj) }
            debug(ctx, "project %s", up)
        }

        if proj, res, isb = up.has_loaded(ctx, loaded, traveUseLoop); isb {
            debug(ctx, "`%s` is already base of `%s` (%s)", loaded, up, proj)
        } else if res && !use.opts.reuse && !loaded.opt.multiUseAllowed {
            debug(ctx, "`%s` has already been imported by `%s` (from %s)", loaded, up, proj)
        }
    }

    if l.verboseImport {
        defer func(t time.Time) {
            prompt(ctx, "%s├┤ %s:import(%s) (%s)\n", l.verpre, l.project, spec, time.Since(t))
        } (time.Now()) //*time.Millisecond // µs, ms, s ┼
    }

    l.useProj(ctx, opts, loaded, params...)
    return
}

const pluginDifferentVersionError = `plugin was built with a different version of package`
var numUpdatedPlugins = 0

func (l ul) buildPlugin(ctx Context, s, src string) (err error) {
    if l.traceLaunch { defer un(l_trace(l_launch, "ul.buildPlugin")) }

    prompt(ctx, "smart: Build %v …", src)
    dir, _ := filepath.Split(src)
    o := &bytes.Buffer{}
    c := exec.Command("go", "build", "-buildmode=plugin", "-o", s)
    c.Stdout, c.Stderr, c.Dir = o, o, dir
    if err = c.Run(); err == nil {
        numUpdatedPlugins += 1
        prompt(ctx, "… ok\n")
        prompt(ctx, "smart: Plugin updated, please relaunch.\n")
        os.Exit(0)
    } else {
        prompt(ctx, "… error\n")
        prompt(ctx, "%s", o)
    }
    return
}

func (l ul) loadPlugin(ctx Context) (err error) {
    if l.traceLaunch { defer un(l_trace(l_launch, "ul.loadPlugin")) }
    if l.project == nil {
        erro(ctx, "current project is nil")
    }

    var g = _stat(ctx, "smart.go", l.project)
    if g == nil { return /* smart.go was not presented */ }

    var src = __string(ctx, g)
    s := strings.Replace(l.project.rel.String(), "..", "_", -1)
    s = filepath.Join(filepath.Dir(joinTmpPath(ctx, "", "")), "plugins", s)

    var build = true

    so := _stat(ctx, /*l.project.name*/"plugin", stat_dir{s}, stat_nonexist{true})
    if s = so.fullname(); s == "" {
        debug(ctx, "file '%v' has empty fullname", so)
    } else if so.exists() && !l.buildPlugins {
		if so._mtime > g._mtime {
            build = false // Plugin already updated.
        }
    }
    if build { err = l.buildPlugin(ctx, s, src) }
    if err != nil { return }

    // Once plugin is opened, there's no need/way to close it.
    if l.project.ext.Plugin, err = plugin.Open(s); err == nil {
        var sym plugin.Symbol
        if sym, err = l.project.ext.Lookup("Init"); err != nil {
            erro(ctx, "nil plugin symbol Init")
        }
        if sym == nil {
            return // no initialization (optional)
        }
        switch init := sym.(type) {
        case func(Context) (error):
            if err = init(ctx); err == nil {
                return
            } else {
                erro(ctx, "plugin Init: %v", err)
            }
        default:
            erro(ctx, "wrong plugin Init: %T", sym)
        }
    } else if strings.Contains(err.Error(), pluginDifferentVersionError) {
        err = l.buildPlugin(ctx, s, src)
    }
    return
}

func (l ul) useProj(ctx Context, opts useopts, proj *project, params ...Value) (err error) {
    if l.verboseUsing {
        defer func(t time.Time) {
            var d = time.Now().Sub(t)
            prompt(ctx, "use(%15s) %s ⇒ %v\n", d, l.project, l.project.use)
        } (time.Now())
    }

    if proj == l.project {
        erro(ctx, "%v: cannot use itself", proj)
    }

    if l.project.isUsingDirectly(proj) {
        return
    }

    // Add to the project using list, so that the use path is correct.
    if l.project.use.append(ctx, proj, params, opts); !opts.noVars {
        // aka.     XXX += $(use.XXX)
        // aka. use.XXX += $(use.XXX)
        l.usevars(ctx, l.project, proj)
        if 0 < len(opts.vars) {
            for _, v := range opts.vars {
                warn(ctx, "var: %T %v", v, v)
            }
            debug(ctx, "TODO: %d vars to import", len(opts.vars))
        }
    }
    return
}

func (l ul) spec_file(ctx Context, specVal Value) (res *file, spec, fullname string) {
    switch t := specVal.(type) {
    case *file:
        if !t.exists() { _ = t.stat(ctx) }
        return t, ident(ctx,t), t.fullname()
    default:
        if spec = __string(ctx, specVal) ; spec == "" {
            erro(ctx, "empty string: %v", ts(specVal))
        }

        var f = l.project.file(ctx, specVal)
        if f == nil {
            if filepath.IsAbs(spec) {
                f = _stat(ctx, spec)
            } else {
                var d string
                var ll = _loader(l.loader.Context)
                if ll != nil { d = ll.project.absPath } else { d = l.project.absPath }
                f = _stat(ctx, spec, stat_dir{d})
            }
        } else if !f.exists() {
            _ = f.stat(ctx)
        }

        if f != nil {
            res, fullname = f, f.fullname()
        }
        return
    }
}

type get_include_opts struct{}
type get_include_spec struct{}
type is_flat_mode struct{}

type include_opts struct { *clause_opts
    ifExists bool `if-exists,ifexists`
}

type p_include struct {
    Context
    o include_opts
    p Pos
    spec string
}
func (p p_include) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_include) inner() Context { return p.Context }
func (p p_include) ts(t string) string { return "{="+t+" "+p.spec+" "+ts(p.Context)+"}" }
func (p p_include) do(ctx Context, op any) (_ any) {
	switch op.(type) {
    case get_position: if p.p.IsValid() { return p.p }
	case get_include_opts: return &p.o
    case get_include_spec: return p.spec
	case is_flat_mode    : return true
	}
	return p.Context.do(ctx, op)
}

func (l ul) include(ctx Context, doc *commentgroup, g *clause_opts, _ int) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "include")) }

	var opts = include_opts{ clause_opts: g }
	if va := parseOpts(ctx, &opts, g.remainder...); len(va) > 0 {
		erro(ctx, "unknown opts: %v", va[0], callstack{num:5})
	}
	if len(g.spec) < 1 {
		erro(ctx, "expect include file: %v", g.spec, callstack{num:5})
	}

	var val = expand(_final(ctx), g.spec[0])

	if l.p.spaces(ctx); l.p.tok == COLON {
		switch val.(type) {
		case *file, *strlit, *strcomp: // escape from file searching
		default: if f := l.project.file(ctx, val); f != nil { val = f }
		}
		val = l.rule(ctx, []Value{val}) // this should return a Rule
	}

    if g.skip { return }

    ctx = pc(ctx, g.spec[0])

    // Execute the rule entry to update include source.
    if x, y := val.(*rule); y && x != nil {
        if z, y := execute_entry(ctx, x); !y {
            erro(ctx, "%v: include entry failed : %s", x, ts(z))
        }

        val = x.target
    }

    var f, spec, fullname = l.spec_file(ctx, val)
    if (f == nil || !f.exists()) && opts.ifExists {
        return // ignore non-exists files
    }

    if spec == "" || fullname == "" {
        erro(ctx, "empty string: %v", ts(val))
    } else {
        var p, s = val.Pos(), l.trimSpecPath(ctx, spec)
        l.source(p_include{ctx, opts, p, s}, fullname, nil)
    }
    return
}

func (l ul) openscope(comment string) *scope {
    var t = &term{} ; *t = l.term
    l.term = term{t, new_scope(l.scope(), l.project, comment)}
    return t.scope
}

func (l ul) closescope(s *scope) {
    if x, y := l.term.Context.(*term); y {
        var ctx Context = l.loader
        if l.p != nil { ctx = pc(l.loader, l.p.pos) }
        if x == &l.term {
            erro(ctx, "conflict term: %s", x.comment)
        }
        if x.scope != s {
            erro(ctx, "conflict scope: %s != %s", x.comment, s.comment)
        }
        l.term = *x
    }
}

// project example (base(var=value))
func (l ul) bases(ctx Context, implicitBase string, params ...Value) {
	var implicitBases []Value

    if true { ctx = closure_with(ctx, l.scope) }

    if f := _stat(ctx, dot_base, l.project); f != nil {
        if !f._isDir && l.project.spec == symDotBase {
            // skip the regular file .base to avoid self loading recursively
        } else {
            implicitBases = append(implicitBases, f)
        }
    }

    if ss := strings.Split(l.project.name.String(), ".") ; len(ss) > 2 && ss[len(ss)-1] == "base" {
        var numBaseParams int
        for _, elem := range params {
            if x, y := elem.(*list); y && len(x.elems) == 1 { elem = x.elems[0] }
            if x, y := elem.(*argumented); y { elem = x.Value }
            if _, y := elem.(*pair); y { continue }
            numBaseParams += 1
        }
        if numBaseParams == 0 {
            var segs []Value
            for _, s := range ss[:len(ss)-2] {
                segs = append(segs, _word(_pos(ctx), intern(s)))
            }
            implicitBases = append(implicitBases, makePath(segs...))
            implicitBase = "" // discard the implicit base
        }
    }

    var implicitIndex int
    if  implicitBase != ""  {
        implicitIndex = len(implicitBases)
        implicitBases = append(implicitBases, _pathStr(ctx, implicitBase))
    }

paramsloop:
    for i, elem := range append(implicitBases, params...) {
        if x, y := elem.(*list); y && len(x.elems) == 1 {
            elem = x.elems[0]
        }
        if x, y := elem.(*argumented); y {
            elem, ctx = x.Value, x.ctx(ctx)
        }
        if x, y := elem.(*pair); y {
            erro(pc(ctx,x), "use -set(%v) instead", x)
        }

        var spec string
        var specVal Value
        if specVal = expand(_final(ctx), elem); specVal == nil { specVal = elem }

        if spec = __string(ctx, specVal) ; spec == "" {
            erro(ctx, "%v: empty base name: %v", l.project, ts(specVal,ctx))
        } else if strings.Contains(spec, "//") {
            erro(ctx,
				_f("%v: invalid spec: %v → %v", l.project, elem, specVal),
				_f("%v: invalid spec: %v → %v", l.project, elem, spec))
        } else if implicitBase != "" && spec == implicitBase {
            if i == implicitIndex {
                ctx = load_implicit{ctx}
            } else {
                erro(ctx, "%v: implicit base '%v' already loaded", l.project, elem)
            }
        }

        var abs string
        var isDir bool

        if x, y := to_file(elem); y && x._mtime != 0 {
            abs, isDir = x.fullname(), x._isDir
        } else {
            abs, isDir = l.search(ctx, spec)
        }

        for _, base := range l.project.bases {
            if base.absPath == abs {
                erro(ctx, "duplicated base: %v : %v → %v (in %v)", base, elem, spec)
                continue paramsloop
            }
        }

        if cc := _abs_ctx(ctx, abs); isDir {
            l.directory(cc, spec, abs, nil)
        } else {
            l.file(cc, spec, abs, nil)
        }
    }

	// Apply the project's own exports to itself
	for _, targetSym := range l.project.exports {
		targetName := targetSym.String()
		lookupSym := intern("use." + targetName)

		d := l.project.def(ctx, defVoid, lookupSym)

		var op usevar
		if targetName == "-l" || targetName == "-L" || targetName == "-I" || strings.HasPrefix(targetName, "-no") {
			op.unique = true
		}
		if targetName == "-l" || targetName == "ldlibs" {
			op.reverse = true
		}

		op.apply(closure_with(ctx, l.project.scope), d, nonTrivialDefsFromBase(ctx, l.project, lookupSym)...)
	}
	return
}

func filespec(workdir, filename string) (spec string) {
    switch dir, base := filepath.Split(filename); base {
    case dot_base, dot_configure: spec = base
    default: spec, _ = filepath.Rel(workdir, dir)
    }
    return
}

func (l ul) dot_container(ctx Context, ident Value, identStr string, f *file) {
    if l.traceLaunch { defer un(l_trace(l_launch, "ul.dot_container")) }

    if s := f.fullname(); f._mtime == 0 {
        erro(ctx, "%s: file not exists: %s", ident, s)
    } else if cc := pc(ctx, ident); f._isDir {
        l.directory(cc, dot_container, s, nil)
    } else {
        l.file(cc, filespec(l.workdir, s), s, nil)
    }

    if x, y := l.globe.loaded[f.fullname()]; y && x != nil {
        if name, _ := l.scope().Lookup(x.name).(*project) ; name == nil {
            erro(ctx, "%v: %v: `dock` is not a project", l.project.name, f)
        }

        var opts useopts
        // TODO: parse the useopts
        l.useProj(ctx, opts, x)
    }
    return
}

func is_configure_project(proj *project) bool {
    return proj == nil ||
        proj.name == symDotConfigure ||
        proj.name == symConfigure ||
        proj.name == intern("configure.base")
}

type l_filename string
type is_autoload struct{ string }

type p_autoload struct{
    Context
    p Pos
    v Value
    l []string
}
func (a *p_autoload) cast(t reflect.Type) Context { return icast(a,t) }
func (a *p_autoload) inner() Context { return a.Context }
func (a *p_autoload) ts(t string) string {
    return "{="+t+" "+bases(2,a.v.String(),true)+" "+ts(a.Context)+"}"
}
func (a *p_autoload) do(ctx Context, op any) (_ any) {
    switch t := op.(type) {
    case get_position: if a.p.IsValid() { return a.p }
    case is_flat_mode: return true
    case is_autoload:
        if t.string == "" { return true }
        return strings.HasSuffix(__string(ctx, a.v), t.string)
    case l_filename:
        if t == "" || t == "?" {
            // noop
        } else if t == "-" {
            a.l = nil
        } else {
            a.l = append(a.l, string(t))
        }
        return a.l
    }
    return a.Context.do(ctx, op)
}

func (l ul) autoload(ctx Context, tag string) {
    if !is_configure_project(l.project) {
        if d := l.project.resolveDef(ctx, intern(".autoload."+tag)); d != nil && d.value != nil {
            for _, v := range merge(expand(_final(ctx),d.value)) {
                if isTrivial(v) {
                    continue
                } else if f, s, t := l.spec_file(ctx, v); f == nil || !f.exists() {
                    continue//erro(ctx, "no such source file: %v → %v", ts(d.value), ts(v))
                } else if s == "" || t == "" {
                    continue//erro(ctx, "empty string: %v → %v", ts(d.value), ts(v))
                } else {
                    l.source(&p_autoload{ctx,l.p.pos,v,nil}, t, nil)
                }
            }
        }
    }
}

type is_config_mode struct{}

type configure_ctx struct {
    Context
    abs, configure string
    local, isDir bool
    configuration *file // configuration.sm
    declared *project
}
func (cc *configure_ctx) cast(t reflect.Type) Context { return icast(cc,t) }
func (cc *configure_ctx) inner() Context { return cc.Context }
func (cc *configure_ctx) do(ctx Context, op any) (_ any) {
    switch t := op.(type) {
	case is_config_mode: if cc.configuration != nil { return true }
	case is_flat_mode: if cc.configuration != nil { return true }
    case abs_path: return cc.abs
    case declared_project:
        if t.absPath == cc.abs {
            cc.declared = t.project
            return
        }
    }
    return cc.Context.do(ctx, op)
}

func (l ul) configuration(ctx Context, ident Value, _ Symbol) {
	if false { defer un(l_tracef(l_traverse, "configuration(%v)", ident)) }
	if l.project.name == symDotConfigure { return }

	const cs = "configure"

	var cc = configure_ctx{Context:ctx}
	var f *file

	if v := l.project.opt.configure; v != nil {
		if x, y := v.(*boolean); y {
			if !x.bool { return }
			cc.configure = cs
		} else {
			cc.configure = __string(ctx, v)
			if cc.configure == "" {
				erro(ctx, "empty configure spec: %v", ts(v))
			} else if cc.configure == "." {
				cc.configure, cc.local = cs, true
			}
		}
	} else if f = _stat(ctx, cs, l.project); f != nil {
		cc.configure, cc.local = cs, true
	} else if f = _stat(ctx, dot_configure, l.project); f != nil {
		cc.configure, cc.local = dot_configure, true
	}

	if f == nil && cc.configure != "" {
		if filepath.IsAbs(cc.configure) {
			f = _stat(ctx, cc.configure)
		} else {
			f = _stat(ctx, cc.configure, l.project)
		}
	}

	if f != nil && f.exists() {
		cc.abs, cc.isDir = f.fullname(), f._isDir
	}

	if cc.abs == "" && l.project.opt.configure != nil {
		if !cc.local { cc.abs, cc.isDir = l.search(ctx, cc.configure) }
		if cc.abs == "" {
			erro(ctx, "%v: no such project: %s", l.project, cc.configure)
		}
	}

	if cc.abs == "" {
		if l.project.opt.configure != nil {
			erro(ctx, "%v: missing the default .configure", l.project)
		}
		if l.project.configure == nil { l.project.configure = l.project }
		return
	}

	// =========================================================
	// 1. Parse the `.configure` sandbox FIRST using clean `cc`
	// Ensures `is_flat_mode` evaluates to false so the sandbox
	// parses smoothly and `projectStart` initializes it properly.
	// =========================================================
	if cc.Context = pc(cc.Context, ident); cc.isDir {
		l.directory(&cc, cc.configure, cc.abs, nil)
	} else {
		l.file(&cc, filespec(l.workdir, cc.abs), cc.abs, nil)
	}

	if cc.declared == nil {
		erro(ctx, "%s not loaded", cc.configure)
	}

	if x, y := l.project.Lookup(symDotConfigure).(*project); !y || x == nil {
		if _, alt := l.project.projectName(ctx, symDotConfigure, cc.declared); alt != nil {
			if p, y := alt.(*project); !y || p == nil {
				erro(ctx, "name `%s' already taken: %s", cc.declared.name, typeof(alt))
			}
		}
	}

	if c := l.project.configure; c != cc.declared {
		if c != nil && c != l.project {
			erro(ctx, "%s already specified", dot_configure)
		} else {
			l.project.configure = cc.declared
		}
	}

	// =========================================================
	// 2. Load Cache securely AFTER sandbox evaluation
	// Sets cc.configuration so `is_flat_mode` protects the parse.
	// =========================================================
	if c := l.project.configuration; c != nil {
		// ALREADY LOADED: Just pass it to the context and skip sourcing.
		cc.configuration = c
	} else if c = l.project.configuration_sm(ctx); c != nil && c.exists() && c.stat(ctx) != nil {
		if l.promptConfigurationLoads(ctx) {
			debug(pc(ctx, c), "cached configuration (%p)", l.project, callstack{num:1})
		}
		cc.configuration = c
		l.source(&cc, c.fullname(), nil) // Populates local definitions correctly
		l.project.configuration = c
	}

	for _, proj := range cc.declared.usees(true, false, false, false) {
		if e := l.useProj(ctx, useopts{}, proj); e != nil {
			erro(ctx, "failed to use %v : %v", proj, e)
		}
	}
}

func (l ul) container(ctx Context, ident Value, identStr string) {
    if l.project.name != symDotContainer {
        if _, e := os.Stat(filepath.Join(l.project.absPath, ".dock")); e == nil {
            erro(ctx, "must rename .dock into .container !")
        }

        // Looking for project specific .container module
        if f := _stat(ctx, dot_container, l.project); f != nil && f.exists() {
            l.dot_container(ctx, ident, identStr, f)
            return
        }

        // Looking for .smart/.container
        walkSmartBaseDirs(ctx, l.project.absPath, func(s string) bool {
            d := stat_dir{filepath.Join(s, ".smart")}
            f := _stat(ctx, dot_container, d)
            if f != nil && f.exists() {
                l.dot_container(ctx, ident, identStr, f)
            }
            return false
        })
    }
    return
}

// If src != nil, load_source_bytes converts src to a []byte if possible;
// otherwise it returns an error. If src == nil, load_source_bytes returns
// the result of reading the file specified by filename.
func load_source_bytes(ctx Context, opts *include_opts, filename string, source ...any) (_ []byte) {
    if 0 < len(source) {
        var n int
        var buf bytes.Buffer
        for _, src := range source {
            if src == nil { continue } else { n += 1 }

            var e error
            switch s := src.(type) {
            case        string: _, e = buf.Write([]byte(s))
            case        []byte: _, e = buf.Write(s)
            case *bytes.Buffer: _, e = buf.Write(s.Bytes())
            case     io.Reader: _, e = io.Copy(&buf, s)
            default:
                erro(pc(ctx,filename), "invalid source : %v", ts(src))
            }

            if e != nil {
                erro(pc(ctx,filename), "copy bytes (%s) failed : %v", typeof(src), e)
            }
        }
        if 0 < n { return buf.Bytes() }
    }
    if t, e := ioutil.ReadFile(filename); e == nil {
        return t
    } else if _, y := e.(*fs.PathError); y {
        if (opts != nil && !opts.ifExists) {
            erro(pc(ctx,filename), "no such source file")
        }
        return
    } else {
        erro(pc(ctx,filename), "%v", e)
        return
    }
}

func (l ul) source(ctx Context, filename string, a_src any) (res Value) {
    if l.traceLaunch { defer un(l_trace(l_launch, "ul.source")) }

    defer func(p *parser) {
        if l.p == nil { erro(ctx, "nil parser") }
        l.p = p
    } (l.p)

    var opts, _ = do(ctx, get_include_opts{}).(*include_opts)
	var text = load_source_bytes(ctx, opts, filename, a_src)
    if text == nil { return }

    l.p = &parser{}
    l.p.scanner.init(ctx, l.fset.AddFile(filename, -1, len(text)), text, 0)
	l.p.next(ctx, true) // starts scanning

    if truly(ctx, is_text{}) {
        return ease(ctx, l.values(ctx))
    }

    l.parse(src(ctx,nil), filename)

    if truly(ctx, is_flat_mode{}) {
        return
    } else {
        return l.project
    }
}

// parseConfigDir parses a configuration directory, where
//     * pathname - is the original pathname (symlink or 'configure' smart file)
//     * linked - is the destination directory pathname to be really iterated
func (l ul) parseConfigDir(ctx Context, pathname, linked string) (err error) {
    var fd *os.File // Directory of the destination.
    if fd, err = os.Open(linked); err != nil {
        erro(ctx, "%v", err)
        return
    }

    defer fd.Close()

    var fs []os.FileInfo
    if fs, err = fd.Readdir(-1); err != nil || len(fs) == 0 { return }

    var ident = filepath.Base(pathname)
    if ident == "_" {
        erro(ctx, "invalid package name %s", ident)
    }

	var sof, _ = filepath.Rel(baseWorkDir, pathname)
    defer l.closescope(l.openscope(bases(2, sof, true)))

    var scope = l.scope()

    for _, f := range fs {
        var name = f.Name()
        if has_prefix(name, "~") || has_suffix(name, ".#", ".smart", ".sm") {
            continue
        }

        var fullname = filepath.Join(linked, name)
        if f.Mode()&os.ModeSymlink != 0 {
            var ( l string; t os.FileInfo )
            if l, err = os.Readlink(fullname); err != nil { continue }
            if !filepath.IsAbs(l) { l = filepath.Join(linked, l) }
            if t, err = os.Stat(l); err != nil { continue }
            if t.IsDir() { continue }
        }

        if f.IsDir() {
            if err = l.parseConfigDir(ctx, filepath.Join(pathname, name), fullname); err != nil {
                erro(ctx, "parse config failed: %v", err)
            }
            if 0 < flush(ctx) { return } else { continue }
        }

        d := scope.def(ctx, defConfig, name)
        if d == nil {
            erro(ctx, "%v", name)
        }

        var v []byte
        if v, err = ioutil.ReadFile(fullname); err != nil {
            erro(ctx, "%v", err)
        }

        var s = string(v)
        if !utf8.ValidString(s) {
            debug(ctx, "%s: invalid UTF8 content", fullname)
        }

        d.set(ctx, _rw(l.p.pos, s))
    }
    return
}

func nonsource(name string, mo os.FileMode) (_ bool) {
    if  !mo.IsRegular() || name == "" || name == configuration_sm || strings.HasPrefix(name, ".#") ||
        !(strings.HasSuffix(name, ".smart") || strings.HasSuffix(name, ".sm")) { return true }
    return
}

func (l ul) sources(ctx Context, path string, filter func(os.FileInfo) bool) (sources []string) {
    fd, err := os.Open(path)
    if err != nil {
        erro(ctx, "%v", err)
    }

    defer fd.Close()

    fis, err := fd.Readdir(-1)
    if err != nil {
        erro(ctx, "readdir: %v", err)
    }
    if len(fis) == 0 {
        erro(ctx, "no files underneath: %s", path)
    }

    var first = fis[0]
	for i := 1; i < len(fis); i += 1 {
		var s = fis[i].Name()
		if s == mainFileName || (s == deprFileName && first.Name() != mainFileName) {
			fis[0], fis[i] = fis[i], first
		}
	}

    for _, d := range fis {
        var name, mo = d.Name(), d.Mode()
        if nonsource(name, mo) || filter != nil && filter(d) { continue }

        var filename = filepath.Join(path, name)
        var linked,_ = _readlink(ctx, filename, d)

        if false && (name == "configure.smart" || name == "configure.sm") && (linked != "" || mo.IsDir()) {
            if l.parseConfigDir(ctx, filepath.Dir(filename), linked) != nil { return }
            continue
        }

        sources = append(sources, filename)
    }
    return
}

// ul.load loads script from a file or source code
func (l ul) file(ctx Context, spec, absPath string, source any) {
    if l.traceLaunch { defer un(l_trace(l_launch, "ul.load")) }

    if absPath == "" {
        erro(pc(ctx,l.p), "%v: no such base: %v", l.project.name, spec)
    } else if !filepath.IsAbs(absPath) {
        erro(pc(ctx,l.p), "%v: not absolute path: %v", l.project.name, spec)
    }

    // Check loaded project.
    if p, y := l.globe.loaded[absPath]; y {
        if _, a := l.scope().projectName(ctx, p.name, p); a != nil {
            if x, y := a.(*project); !y || x == nil {
                erro(ctx, "name already taken: %v (%s).", p, typeof(a))
            }
        }
        do(ctx, declared_project{p})
        return
    }

    var lo = l
    if l.project != nil {
        lo.loader = &loader{term:term{ctx, l.scope()}}
        ctx = lo.loader
    }

    lo.source(ctx, absPath, source)
	lo.saveConfiguration(ctx)
    return
}

func (l ul) directory(ctx Context, spec, absDir string, filter func(os.FileInfo) bool) {
    if absDir == "" {
        erro(ctx, "%v: no such base: %v", l.project, spec)
    } else if !filepath.IsAbs(absDir) {
        erro(ctx, "%v: not absolute path: %v", l.project, spec)
    }

    var okay bool
    var loaded *project
	defer func(t time.Time, proj *project) {
		if spec == "." { spec = absDir }
		if loaded == nil { return }
		if l.globe.main == nil { l.globe.main = loaded }
		if proj != nil {
			if d := proj.resolveDef(ctx, loaded.name); d != nil {
				c := pc(ctx, d.value)
				note(c, "conflicts project name %v", ts(d))
				note(c, "conflicts project name %v", loaded)
				erro(c, "%v %v", d, loaded)
			}
		}
		if l.project != nil {
			if p, _ := l.project.Lookup(loaded.name).(*project); p == nil {
				if _, alt := l.project.projectName(ctx, loaded.name, loaded); alt != nil {
					if x, y := alt.(*project); !y || x == nil {
						erro(ctx, "`%s' already taken: %s", loaded.name, alt)
					}
				}
			}
		}
	} (time.Now(), l.project)

    // Check previously loaded project.
    if loaded, okay = l.globe.loaded[absDir]; okay && loaded != nil {
        do(ctx, declared_project{loaded})
        return
    }

    var lo = l
    if l.project != nil {
        lo.loader = &loader{term:term{ctx,nil}}
        ctx = lo.loader
    }

	var sof, _ = filepath.Rel(baseWorkDir, absDir)
    defer lo.closescope(lo.openscope(bases(2, sof, true)))
    defer lo.saveConfiguration(ctx)

    // Use globe outer scope to avoid conflicting with other unrelated projects.
    lo.scope().outer = lo.globe.scope

    var cc = _abs_ctx(ctx, absDir)
    for _, s := range lo.sources(cc, absDir, filter) { lo.source(cc, s, nil) }

    if len(lo.declares) == 0 && filepath.Base(spec) != "@" {
        if truly(ctx, is_implicit_load{}) {
            debug(ctx, "%s not loaded (as %s, implicitly)", spec, absDir)
            return // okay for implicit loading
        }

        for s, m := range l.globe.loaded { debug(ctx, "%v: %v", s, m) }
        erro(ctx, "%s not loaded (as %s)", spec, absDir)
    }

    if loaded, okay = l.globe.loaded[absDir]; okay && loaded != nil {
        return // Good!
    }

    if filepath.Base(spec) == "@" {
        return // Okay!
    }

    erro(ctx, "%s not loaded", spec)
    return
}

type is_text struct{}
type loadtext_ctx struct{ Context }
func (p loadtext_ctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p loadtext_ctx) inner() Context { return p.Context }
func (p loadtext_ctx) do(ctx Context, op any) any {
	switch op.(type) {
	case is_text: return true
	}
	return p.Context.do(ctx, op)
}

func (l ul) text(ctx Context, filename string, text string) Value {
    if l.globe.main == nil {
        l.loader.scope = l.globe.os.scope
    } else {
        l.loader.scope = l.globe.main.scope
    }
    return l.source(loadtext_ctx{ctx}, filename, text)
}


// NOTE: all single character opt names/shortcuts should be preserved for general purposes.
type general_opts struct {
    debug    int  `db,dbg,debug` // NOTE: compatible with 'bool'
    stack    int  `stack,stack-number`
    fail     bool `fail` // fail on errors
    fullname bool `full,fullname,fullfile`
    silent   bool `silent` // force silent, contrast 'verbose'
    timing   bool `time,timing`
    verbose  bool `verb,verbose` // prompts more information
    warning  bool `warn,warning` // prompts more warnings
}

type modifier_ struct { Context ; general_opts }
type modifier_v interface{ v(...Value) any }
type modifier_x interface{ x(...Value) any }
type modifier_y interface{ x(*execution, ...Value) any }

var modifier_v_t = reflect.TypeOf((*modifier_v)(nil)).Elem()
var modifier_x_t = reflect.TypeOf((*modifier_x)(nil)).Elem()
var modifier_y_t = reflect.TypeOf((*modifier_y)(nil)).Elem()
var modifiers = map[Symbol]reflect.Type{
	symDebug:       reflect.TypeOf((*modifier_debug)(nil)).Elem(),
	symPrint:       reflect.TypeOf((*modifier_print)(nil)).Elem(),
	symPrompt:      reflect.TypeOf((*modifier_prompt)(nil)).Elem(),

	symPreserve:    reflect.TypeOf((*modifier_preserve)(nil)).Elem(),
	symExpand:      reflect.TypeOf((*modifier_expand)(nil)).Elem(),
	symPlain:       reflect.TypeOf((*modifier_plain)(nil)).Elem(),
	symStringify:   reflect.TypeOf((*modifier_stringify)(nil)).Elem(),
	symReveal:      reflect.TypeOf((*modifier_reveal)(nil)).Elem(),
	symDisclose:    reflect.TypeOf((*modifier_disclose)(nil)).Elem(),
	symClosure:     reflect.TypeOf((*modifier_closure)(nil)).Elem(),

	symSelect:      reflect.TypeOf((*modifier_select)(nil)).Elem(),

	symEnv:         reflect.TypeOf((*modifier_env)(nil)).Elem(), // interpreter environments
	symDep:         reflect.TypeOf((*modifier_dep)(nil)).Elem(),
	symVar:         reflect.TypeOf((*modifier_var)(nil)).Elem(),
	symSet:         reflect.TypeOf((*modifier_set)(nil)).Elem(),
	symDefer:       reflect.TypeOf((*modifier_defer)(nil)).Elem(),

	symCd:          reflect.TypeOf((*modifier_cd)(nil)).Elem(),
	symMkdir:       reflect.TypeOf((*modifier_mkdir)(nil)).Elem(),
	symSudo:        reflect.TypeOf((*modifier_sudo)(nil)).Elem(),
	symTouch:       reflect.TypeOf((*modifier_touch)(nil)).Elem(),
	symGrep:        reflect.TypeOf((*modifier_grep)(nil)).Elem(),
	symDeps:        reflect.TypeOf((*modifier_deps)(nil)).Elem(),

	symCopyFile:       reflect.TypeOf((*modifier_copyfile)(nil)).Elem(),
	symWriteFile:      reflect.TypeOf((*modifier_writefile)(nil)).Elem(),
	symReadFile:       reflect.TypeOf((*modifier_readfile)(nil)).Elem(),
	symUpdateFile:     reflect.TypeOf((*modifier_updatefile)(nil)).Elem(),
	symConfigureInput: reflect.TypeOf((*modifier_configureinput)(nil)).Elem(),
	symConfigureFile:  reflect.TypeOf((*modifier_configurefile)(nil)).Elem(),
	// symConfigure:       reflect.TypeOf((*modifier_configure)(nil)).Elem(),

	symWait:         reflect.TypeOf((*modifier_wait)(nil)).Elem(),
	symStamp:        reflect.TypeOf((*modifier_stamp)(nil)).Elem(),

	symCheck:        reflect.TypeOf((*modifier_check)(nil)).Elem(),
	symAssert:       reflect.TypeOf((*modifier_assert)(nil)).Elem(),
	symCase:         reflect.TypeOf((*modifier_case)(nil)).Elem(),
	symCond:         reflect.TypeOf((*modifier_cond)(nil)).Elem(),
	symIf:           reflect.TypeOf((*modifier_cond)(nil)).Elem(),
	symWhere:        reflect.TypeOf((*modifier_cond)(nil)).Elem(),
	symOnce:         reflect.TypeOf((*modifier_once)(nil)).Elem(),
	symFork:         reflect.TypeOf((*modifier_fork)(nil)).Elem(),

	symGitAhead:    reflect.TypeOf((*modifier_gitahead)(nil)).Elem(),
	symGitModified: reflect.TypeOf((*modifier_gitmodified)(nil)).Elem(),

	symBy:           reflect.TypeOf((*modifier_setDirtyPats)(nil)).Elem(),
	symDirty:        reflect.TypeOf((*modifier_predictDirty)(nil)).Elem(),
}

type is_modify struct{}
type    modify_ctx struct{ Context }
func (c modify_ctx) inner() Context { return c.Context }
func (c modify_ctx) cast(t reflect.Type) Context { return icast(c,t) }
func (c modify_ctx) do(ctx Context, op any) any {
    switch op.(type) {
    case is_modify: return true
    }
    return c.Context.do(ctx, op)
}

func modify(ctx Context, g *group, hyphen bool) (res Value) {
    var name, args = __symbol(ctx, g.elems[0]), g.elems[1:]

    if t, y := modifiers[name]; !y {
        _, e, _ := entryIndicator(ctx, _entry(ctx))
        erro(ctx,
			_f("%v: %s failed for %s\n", e, name, _project(ctx)),
			_f("unknown modifier: %s (args=%v)", name, args))
    } else {
        var exe = _execution(ctx)
        var mv = reflect.New(t)
        var mi = mv.Interface()
        var fv modifier_v
        var fx modifier_x
        var fy modifier_y
        if !hyphen {
            if fv, _ = mi.(modifier_v); fv == nil {
                erro(ctx, "%v: no method: (*%s).v(...)", name, typeof(mi))
            }
        } else if fx, _ = mi.(modifier_x); fx == nil {
            if fy, _ = mi.(modifier_y); fy == nil {
                erro(ctx, "%v: no method: (*%s).x(...)", name, typeof(mi))
            } else if exe == nil {
                erro(ctx, "%v: nil execution: (*%s).x(...)", name, typeof(mi))
            }
        }

        if c := mv.Elem().FieldByName("Context"); c.IsValid() {
            c.Set(reflect.ValueOf(modify_ctx{pc(ctx, g)})) // c.Type().String() == "smart.Context"
        } else {
            erro(ctx, "%v: no field: %s.Context", name, typeof(mi))
        }

        args = _opts(ctx, mv, args)

        if fv != nil {
            res = ease(ctx, fv.v(args...))
        } else if fx != nil {
            res = ease(ctx, fx.x(args...))
        } else if fy != nil {
            res = ease(ctx, fy.x(exe, args...))
        }
    }

    if !hyphen {
        // $- remains
    } else if res == nil {
        res = _null(g.pos) // $- remains too
    } else if name == symDefer || name == symSet || name == symVar {
        erro(ctx, "invalid result: (set ...) ⇒ %v", res)
    } else if a := _automatic(ctx); a != nil {
        a.amend(ctx, symDash, res)
    }
    return
}

type modifier struct { group }
func (m *modifier) kind() Kind { return m.group.kind()|KindModifier }
func (m *modifier) _cmp(ctx Context, v Value) (_ cmpres) {
    if x, y := v.(*modifier); y { return cmp(ctx, &m.group, &x.group) }
    return
}

type modification struct { valbase ; list []*modifier }
func (_ *modification) kind() Kind { return KindModification }
func (g *modification) _cmp(ctx Context, v Value) (res cmpres) {
    if o, y := v.(*modification); y && len(g.list) == len(o.list) {
        for i, m := range g.list {
            if t := cmp(ctx, m, o.list[i]); t != cmpEqual { return t }
        }
        res = cmpEqual
    }
    return
}
func (g *modification) String() (s string) {
    s = "{"
    for i, m := range g.list {
        if i > 0 { s += " " }
        if m != nil { s += m.String() }
    }
    s += "}"
    return
}

func getGroupElem(value Value, n int, v Value) Value {
    if g, y := value.(*group); y {
        if elem := g.at(n); elem != nil {
            v = elem
        }
    }
    return v
}

func promptShellResult(ctx Context, value Value, n int) {
    if g, y := value.(*group); y && g != nil {
        if elem := g.at(0); elem != nil {
            if str := __string(ctx, elem); str == "shell" {
                if elem = g.at(n); elem != nil {
                    if str = __string(ctx, elem); strings.HasSuffix(str, "\n") {
                        prompt(ctx, "%s", str)
                    } else if str != "" {
                        prompt(ctx, "%s\n", str)
                    }
                }
            }
        }
    }
    return
}

type modifier_debug struct { modifier_
    cond   Value `if,cond,where,when`
    info []Value `info`
    warn []Value `warn`
    erro []Value `err,erro,error`
    checkOutdated bool `dirty,checkdirty,check-dirty,check-outdated`
    trave int `tr,trave,traverse`
    s int `stack,stack-number`
    n int `count,num,call-number`
}
func (ctx *modifier_debug) x(args ...Value) (result any) {
    if ctx.cond != nil && !__true(ctx, ctx.cond) { return }
    if ctx.s == 0 && ctx.stack > 0 { ctx.s = ctx.stack }
    if ctx.n == 0 && ctx.debug > 0 { ctx.n = ctx.debug }
    for _, v := range ctx.info { info(ctx, "%s", __string(ctx, v)) }
    for _, v := range ctx.warn { warn(ctx, "%s", __string(ctx, v)) }
    for _, v := range ctx.erro { erro(ctx, "%s", __string(ctx, v)) }

    var (
        target  = auto_get(ctx, symAt) // @
        depends = auto_get(ctx, symCaret) // ^
    )
    if ctx.checkOutdated && target != nil {
        var (
            ordered = auto_get(ctx, symBar) // |
            grepped = auto_get(ctx, symTilde) // ~
            tt = statFile(ctx, target).mod()
        )
        if tt.IsZero() {
            debug(ctx, "target not exists: %v", target)
            return
        }
        for _, dep := range merge(depends, ordered, grepped) {
            if dt := statFile(ctx, dep).mod(); dt.After(tt) {
                debug(ctx, "%v: outdated by %v (%v)", target, dep, dt.Sub(tt))
            }
        }
    }
    if len(ctx.info) == 0 && len(ctx.warn) == 0 && len(ctx.erro) == 0 {
		var dps []*diag_point
		for _, a := range args {
			pos := do(ctx, get_fatpos{a.Pos()})
			dps = append(dps, _f("%v: %v: %v\n", pos, target, a))
		}

		var aa = []any{ diagtext{} }
		if ctx.s > 0 { aa = append(aa, trace_ctx{ctx.s}) }
		// if ctx.n > 0 { /* d.debug(ctx.n) */ }
		debug(pc(ctx,args), dps, aa...)
    }
    return
}

type modifier_print struct { modifier_
    stdout bool `o,stdout`
    stderr bool `e,stderr` // TODO: = true
    reset  bool `r,reset`
}
func (ctx *modifier_print) x(args ...Value) (result any) {
    var content string
    if val := auto_get(ctx, symDash); val != nil { content = __string(ctx, val) }
    if ctx.stdout { fmt.Fprint(stdout, content) }
    if ctx.stderr { fmt.Fprint(stderr, content) }
    if ctx.reset  { auto_set(ctx, defVoid, symDash, _none(_pos(ctx))) }
    return
}

type modifier_prompt struct { modifier_ }
func (ctx *modifier_prompt) x(args ...Value) (result any) {
    if len(args) == 0 {
        if h := auto_get(ctx, symDash); h != nil {
            prompt(ctx, "%s\n", __string(ctx, h))
        }
    } else {
        for _, a := range args { prompt(ctx, "%s\n", __string(ctx, a)) }
    }
    return
}

type modifier_preserve struct { modifier_ }
func (ctx *modifier_preserve) v(args ...Value) (result any) {
    return args
}

type modifier_expand struct { modifier_ }
func (ctx *modifier_expand) v(args ...Value) (result any) {
    result = expands(ctx, args...)
    return
}

type modifier_plain struct { modifier_ }
func (ctx *modifier_plain) v(args ...Value) (result any) {
    result = expands(_final(ctx), args...)
    return
}

type modifier_stringify struct { modifier_ }
func (ctx *modifier_stringify) v(args ...Value) (result any) {
    result = expands(_final(ctx), args...)
    return
}

type modifier_reveal struct { modifier_ }
func (ctx *modifier_reveal) v(args ...Value) (result any) {
    result = expands(original{ctx,defExpand1}, args...)
    return
}

type modifier_disclose struct { modifier_ }
func (ctx *modifier_disclose) v(args ...Value) (result any) {
    result = expands(original{ctx,defExpand2}, args...)
    return
}

// select element by index from group result: (select 0)
type modifier_select struct { modifier_ }
func (ctx *modifier_select) x(args ...Value) (_ any) {
    if h := auto_get(ctx, symDash); h == nil {
        erro(ctx, "no pipe value $-")
    } else if x, y := h.(*group); y {
        var vals []Value
        for _, a := range xmerge(ctx, args...) {
            vals = append(vals, x.at(int(__int(ctx, a))))
        }
        return vals
    }
    return
}

type modifier_env struct { modifier_ }
func (ctx *modifier_env) x(args ...Value) (result any) {
    if exe := _execution(ctx); exe != nil {
        for _, a := range xmerge(ctx, args...) {
            if p, y := a.(*pair); y {
                exe._env = append(exe._env, p)
            } else {
                erro(ctx, "env: not a pair value: %s", ts(a,ctx), callstack{num:16})
            }
        }
    }
    return
}

type modifier_var struct { modifier_ }
func (ctx *modifier_var) x(args ...Value) any {
	for _, arg := range args {
		// 1. ONE-LINE EXTRACTION! __symbol natively handles *pair keys, *words, and dynamic nodes.
		sym := __symbol(ctx, arg)

		if sym == symEmpty {
			erro(ctx, "%v is unsupported (try: foo=value)", ts(arg,ctx))
			continue
		}

		var value Value
		switch a := arg.(type) {
		case *pair:
			if x, y := a.val.(*group); y {
				value = x.list()
			} else {
				// CRITICAL: Do NOT expand here. Store the raw AST node (e.g., $(file $(a).c))
				value = a.val
			}
		default:
			// For *word or any dynamically resolved identifier, assign _null
			value = _null(arg.Pos())
		}

		// Re-bind the raw node into the local runtime scope using the native Symbol!
		_scope(ctx).def(ctx, defVoid, sym, value)
	}
	return nil
}

// examples:
//     [(set name=value)]    set $(name) to 'value'
//     [(set name)]          clear $(name)
//     [(set -)]             clear $-
type modifier_set struct { modifier_ }
func (ctx *modifier_set) x(args ...Value) (_ any) {
	for _, arg := range args {
		var value Value

		// 2. ONE-LINE EXTRACTION!
		sym := __symbol(ctx, arg)

		switch a := arg.(type) {
		case *pair:
			// NOTE: pair.Value is not expanded, need to do it again.
			value = expand(_final(ctx), a.val)
			if value == nil {
				value = a.val
			}
		case flag:
			// Flags might have an empty value, which defaults to the dash symbol
			sym = __symbol(ctx, a.Value)
			if sym == symEmpty {
				sym = symDash // NATIVE INTEGER CONSTANT!
			}
			value = _null(a.Pos())
		}

		if sym == symEmpty {
			erro(ctx, "%v is unsupported (try: foo=value)", ts(arg))
			continue
		}

		// 3. INTEGER ASSIGNMENT: Assuming auto_set was upgraded to accept sym Symbol
		var d, _ = auto_set(ctx, defVoid, sym, value)
		if d == nil {
			erro(ctx, "no auto set: %v : %v", sym, ts(ctx))
		}

		// 4. FAST INTEGER CHECK: Replaces `name == "@"`
		if sym == symAt {
			var f, s, _ = as_fullname_file(ctx, value)
			if ctx.verbose {
				// CRITICAL FIX: Safe VFS fallback to prevent panic on unbound files
				var traved int
				if f != nil {
					traved = int(f._traved)
				}

				var ts = trimPromptString(s)
				prompt(ctx, "%s …… traversed (%d)\n", ts, traved)
			}
		}
	}
	return
}

type modifier_defer struct { modifier_ }
func (ctx *modifier_defer) x(a ...Value) (_ any) {
    if x := _execution(ctx); x != nil { x.defers = append(x.defers, a...) }
    return
}

type modifier_dep struct { modifier_ }
func (ctx *modifier_dep) x(args ...Value) any {
	for _, a := range args { traverse(ctx, a) }
	return nil
}

type modifier_setDirtyPats struct { modifier_
    pats []Value
}
func (ctx *modifier_setDirtyPats) x(args ...Value) (result any) {
    var opts, y = do(ctx, propDirtyOpts).(*dirtyOpts)
    if y { ctx.pats = parseOpts(_final(ctx), opts, args...) }
    return
}

// create closure context for the traversal
type modifier_closure struct { modifier_
    target Value `@,target`
}
func (ctx *modifier_closure) x(exe *execution, args ...Value) (result any) {
	// Closure the caller program, the context will be restored when execution is finished.
	var cc = exe.Context
	exe.Context = closure_with(cc)

	if false && cast[*term](ctx) != exe.Context {
		erro(ctx, "wrong closure_with")
	}

	var proj = _project(ctx)

	// CRITICAL FIX: Upgraded internal helper to accept sym Symbol
	var set = func(sym Symbol, val Value) (t Value) {
		var noop bool
		if v, y := val.(*boolean); y {
			if !v.bool { noop = true }
		} else if isTrivial(val) {
			erro(ctx, "trivial target: %T %v", val, val)
		} else if true {
			t = expand(ctx,val) //, plain
		} else {
			t = val
		}

		if l, y := t.(*list); y && len(l.elems) == 1 { t = l.elems[0] }

		// FAST INTEGER LOOKUP!
		if !noop && isTrivial(t) { t = auto_get(ctx, sym)  }

		if t != nil {
			// FAST INTEGER ASSIGNMENT!
			auto_set(ctx, defVoid, sym, t) // aka (set @=&@)
		} else if !noop {
			erro(ctx, "%v: %s is nil", proj, sym)
		}
		return
	}

	var target Value
	if ctx.target != nil {
		if target = expand(ctx,ctx.target); target == ctx.target {
			// CRITICAL FIX: Use the native symAt constant instead of symAt
			if t := auto_get(cc, symAt); t != nil {
				target = t
			}
		}
	}

	if ctx.verbose { var t = target
		debug(ctx, "%v: @: %v ⇒ %v %v", proj, ctx.target, typeof(t), t)
	}

	if target != nil {
		// FAST CALL: Pass symAt instead of "@"
		var ( t = set(symAt, target) ; f *file ; s string ; y bool ; n int )
		if f, s, y = as_fullname_file(ctx, t); !y {
			s = __string(ctx, t)
		} else {
			n = int(f._traved)
		}

		if n > 1 {
			if ctx.verbose {
				var ts = trimPromptString(s)
				prompt(ctx, "%s …… traversed (%d, %v)\n", ts, n)
				if false { debug(ctx, "%v, %v, (%d)", f, s, n) }
			}
			return
		}

		// FIXME: if isInnerauto_get(ctx, t.Value) {
		//      erro(ctx, "loop: %v", t)
		//      return
		// }
	}

	if proj == nil {
		erro(ctx, "%T: nil project in the context", ctx)
	} else if scope := proj.scope; scope == nil {
		erro(ctx, "empty closure context")
	} else if def := scope.finddef(symSlash); def == nil {
		erro(ctx, "&/ is undefined")
	} else if dir := __string(ctx, def.value); dir == "" { // Directory path must remain a string
		erro(ctx, "&/ is empty")
	} else if !filepath.IsAbs(dir) {
		erro(ctx, "&/ is relative")
	} else /* if err := enter(ctx, dir); err == nil */ {
		exe.changedWD = dir
	}
	return
}

type modifier_cd struct{ modifier_
    path bool `path`
    printEnter bool `print-enter`
    printLeave bool `print-leave`
}
func (ctx *modifier_cd) x(args ...Value) (result any) {
    if (ctx.printEnter || ctx.printLeave) && len(args) == 0 { return }
    if len(args) == 1 {
        var dir = __string(ctx, args[0])
        if dir == "" {
            // TODO: do something special
            return
        }

        var proj = _project(ctx)
        if !filepath.IsAbs(dir) { dir = filepath.Join(proj.absPath, dir) }
        if ctx.path && dir != "." && dir != ".." && dir != pathSep {// mkdir -p
            if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                debug(ctx, "make path '%s' failed: %v", dir, err)
            }
        }
        if exe := _execution(ctx); exe != nil { exe.changedWD = dir }
    } else {
        erro(ctx, "wrong number of cd args: %v", args)
    }
    return
}

type modifier_mkdir struct { modifier_
    mode os.FileMode `mode`
}
func (ctx *modifier_mkdir) x(args ...Value) (result any) {
    if ctx.mode == 0 {
        ctx.mode = os.FileMode(0755)
    } else {
        ctx.mode |= os.FileMode(0111)
    }
    if len(args) == 0 {
        if v := auto_get(ctx, symAt); !isTrivial(v) { args = append(args, v) }
    }
    for _, a := range xmerge(_final(ctx), args...) {
        var s string
        if x, y := a.(*file); y {
            s = x.fullname()
        } else {
            s = __string(ctx, a)
        }
        if strings.Contains(s, " /") || strings.Contains(s, " ./") || strings.Contains(s, " ../") {
            erro(ctx, "multiple paths (%v): '%v'", typeof(a), s)
        } else if strings.Contains(s, " ") {
            debug(ctx, "path containing spaces (%v): '%v'", typeof(a), s)
        }
        if e := os.MkdirAll(s, ctx.mode); e != nil {
            erro(ctx, "path: %v(%v) ⇒ %s: %v", typeof(a), a, s, e)
        }
    }
    return
}

type modifier_sudo struct { modifier_ }
func (ctx *modifier_sudo) x(args ...Value) (result any) {
    erro(ctx, "TODO: sudo modifier is not implemented yet")
    return
}

func parseDependList(ctx Context, dependList *list) (depends *list) {
    depends = new(list)
    for _, depend := range dependList.elems {
        switch d := depend.(type) {
        case *list:
            if dl := parseDependList(ctx, d); dl != nil {
                depends.elems = append(depends.elems, dl.elems...)
            }
        case *exec_result:
            if d.Status != 0 {
                erro(ctx, "bad status %v", d.Status)
            } else {
                depends.append(d)
            }
        case *rule, *strlit, *file:
            depends.append(d)
        default:
            erro(ctx, "unsupported entry depend `%v' (%v)", depend, _program(ctx).depends)
        }
    }
    return
}

type langInfoT struct {
    rxs []*regexp.Regexp
    sys []*regexp.Regexp
}

var langInfos = map[string]*langInfoT{
    "asm": &langInfoT{
        []*regexp.Regexp{
            regexp.MustCompile(`^\s*#\s*include\s*"(.+)".*$`),
        },
        []*regexp.Regexp{
            regexp.MustCompile(`^\s*#\s*include\s*<(.+)>.*$`),
        },
    },
    "c": &langInfoT{
        []*regexp.Regexp{
            regexp.MustCompile(`^\s*#\s*include\s*"(.+)".*$`),
        },
        []*regexp.Regexp{
            regexp.MustCompile(`^\s*#\s*include\s*<(.+)>.*$`),
        },
    },
    "i": &langInfoT{
        []*regexp.Regexp{
            regexp.MustCompile(`^\s*include\s*"(.+)".*$`),
        },
        []*regexp.Regexp{
        },
    },
}

func init () {
    if info, ok := langInfos["c"]; ok {
        langInfos["c++"] = info
        langInfos["clang"] = info
        langInfos["objc"] = info
        langInfos["objc++"] = info
    }
    if info, ok := langInfos["i"]; ok {
        langInfos["include"] = info
        langInfos["TableGen"] = info
        langInfos["td"] = info
    }
}

var grepCacheFilebase = make(map[*filebase]*grepCacheFiles)
type grepCacheFiles struct {
    file *file
    list []*file
}
type greptouch struct {
	target         Value
	targetFullName string
	targetDir      string  // The string directory path used by the OS and _stat
	targetMtime    int64   // EXILED: targetInfo os.FileInfo
	files          []Value
}
type grepctx struct {
    *modifier_grep
    greptouch
    report bool // discard or report missing greps
    rxs []*greprex
    done map[string]int
    savedGrepFileName string
    savedGrepFile *file
    save *bufio.Writer
}
type greprex struct{ bool ; *regexp.Regexp }
func (g *greptouch) work(ctx Context, gc *grepctx) (err error) {
	if g.targetMtime == 0 {
		erro(ctx, "'%v' not exists", g.target)
	}
	
	// Directly use our cached primitive! Zero interface calls.
	var maxNano int64 = g.targetMtime 
	
	for _, val := range g.files {
		var file *file
		if file, _ = to_file(val); file == nil {
			erro(ctx, "'%v' is not file (%T)", val, val)
			continue
		}
		
		// VFS-native stat extraction
		if !file.exists() && !file.isSysFile() {
			file.stat(ctx) 
			if !file.exists() && gc.debug>0 { 
				warn(ctx, "'%v' info is nil (%s)", file, file.fullname()) 
			}
		}
		
		if !file.exists() {/* ... */} else
		if file._mtime > maxNano { // Pure O(1) Integer Math!
			maxNano = file._mtime
			if gc.debug>0 { 
				warn(ctx, "touch %v → %v (%v)", g.target, file, time.Unix(0, maxNano)) 
			}
		}
	}
	
	// Only allocate a time.Time if we actually crossed the threshold
	if maxNano > g.targetMtime {
		newTime := time.Unix(0, maxNano)
		if err = os.Chtimes(g.targetFullName, newTime, newTime); err != nil {
			erro(ctx, "%v", err)
		}
	}
	return
}
func (g *grepctx) isTargetFile(ctx Context, _f *file) (res bool) {
    if _f == nil {
        // ...
    } else if g.target == _f {
        res = true
    } else if s, _ := as_fullname_string(ctx, g.target); s == g.targetFullName {
        res = true
    } else if f, y := to_file(g.target); y && ident(ctx,f) == ident(ctx,_f) {
        res = true
    }
    return
}

var grepcache = make(map[string][]Value)
var grepcacheM sync.Mutex // avoid fatal error: concurrent map writes

func loadGrepCache(ctx Context) {
    s := joinTmpPath(ctx, "", "cache")
    f, err := os.Open(s)
    if err != nil { return } else { defer f.Close() }
    var ( list []Value ; k string )
    scanner := bufio.NewScanner(f)
	// Allocate a 64KB initial buffer, but allow it to grow up to 10MB per line!
    const maxCapacity = 10 * 1024 * 1024
    buf := make([]byte, 0, 64*1024)
    scanner.Buffer(buf, maxCapacity)
    scanner.Split(bufio.ScanLines)
    for scanner.Scan() {
        s = scanner.Text()
        if strings.HasPrefix(s, ":") { //
            if k != "" && len(list) > 0 {
                grepcache[k] = list
            }
            if len(list) > 0 { list = list[:0] }
            k = s[1:]
        } else {
            a := strings.Split(s, "|")
            if len(a) == 3 {
                file := _stat(ctx, a[0], stat_sub{a[1]}, stat_dir{a[2]})
                if file != nil {
                    list = append(list, file)
                }
            }
        }
    }
}

func saveGrepCache(ctx Context) {
    s := joinTmpPath(ctx, "", "cache")
    f, err := os.OpenFile(s, os.O_RDWR|os.O_CREATE, 0666)
    if err != nil { return } else { defer f.Close() }
    var w = bufio.NewWriter(f)    ; defer w.Flush()
    grepcacheM.Lock(); defer grepcacheM.Unlock()
    for k, l := range grepcache {
        if len(l) == 0 { continue }
        fmt.Fprintf(w, ":%s\n", k)
        for _, v := range l {
            var file, ok = to_file(v)
            if !ok { continue }
            fmt.Fprintf(w, "%s|%s|%s\n", ident(ctx,file), file.sub, file.dir)
        }
    }
}

func searchGreppedName(ctx Context, gp Position, gc *grepctx, sys bool, name string) (res *file) {
	var isAbs, isRel bool
	if isAbs = filepath.IsAbs(name); isAbs {
		res = _stat(ctx, name, stat_nonexist{true})
	} else if isRel = isRelPath(name); isRel { // relative to targetDir
		res = _stat(ctx, name, stat_dir{gc.targetDir}, stat_nonexist{true})
	} else if res = findfile(ctx, name); res != nil && res.exists() {
		return // found existed file
	}

	// System files are not treated as missing nor collected
	// for further updating, just discard them immediately.
	if !sys && res != nil && res.filemap != nil && len(res.filemap.paths) == 1 {
		// system files defined by `files ((foo.xxx) ⇒ -)`
		if f, ok := res.filemap.paths[0].(flag); ok {
			sys = isNone(f.Value) || isNull(f.Value)
		}
	}
	if !sys && gc.debug>0 {
		// Translated to the cleaner _f() pattern
		debug(ctx,
			_f("%v: %v → %v (exists=%v, sys=%v, from %v)", _entry(ctx), gc.target, name, res.exists(), sys, _project(ctx)),
			callstack{num:gc.debug},
		)
	}
	if sys || (res != nil && res.exists()) { return }

	// relative to target directory
	var alt = _stat(ctx, name, stat_dir{gc.targetDir})
	if alt != nil { res = alt; return }

	// Check for bare non-system sub-paths:
	//   foo/bar/name.xxx
	// We search base name 'name.xxx' again:
	var s = filepath.Dir(name) // e.g: foo/bar

	// Search 'name.xxx' and check dir for
	// 'foo/bar' suffix. We use it if found.
	alt = findfile(ctx, filepath.Base(name))
	if alt != nil && strings.HasSuffix(alt.dir.String(), pathSep+s) {
		dirStr := strings.TrimSuffix(alt.dir.String(), pathSep+s)

		// --- GATEWAY TO THE SYMBOL DOMAIN ---
		dirSym  := intern(dirStr)
		sSym    := intern(s)
		nameSym := intern(name)
		altSym  := __symbol(ctx, alt) // Replaces ident(ctx, alt)

		ok1 := alt.change(dirSym, sSym, altSym)    // <dir>, foo/bar, name.xxx
		ok2 := alt.change(dirSym, symEmpty, nameSym) // <dir>, "", foo/bar/name.xxx
		res  = alt

		if checkpoints {
			if !ok1 {
				debug(ctx, _f("unchanged: %s %s %s", dirStr, s, altSym.String()), trace{})
			}
			if !ok2 {
				debug(ctx, _f("unchanged: %s %s", dirStr, altSym.String()), trace{})
			}
		}
	} else if res == nil {
		for _, inc := range gc.incs {
			if res = _stat(ctx, name, stat_dir{__string(ctx, inc)}); res != nil {
				if false { debug(ctx, _f("%v in %v", res, inc)) }
				return
			}
		}
		if res == nil { res = _stat(ctx, name, stat_nonexist{true}) }

		p := _project(ctx)
		debug(ctx,
			_f("'%s' not found in %v", name, p),
			_f("grepped '%s' has no target dir in %v", name, p),
			_f("from project %v (for %v)", p, name))
	}
	return
}

func searchGrepped(ctx Context, gp Position, gc *grepctx, sys bool, name string) (file *file, err error) {
	if file = searchGreppedName(ctx, gp, gc, sys, name); file == nil {
		// The 'name' is not matching the files database.
		if gc.discard { return }
		// FIXME: missing-file error
	} else if gc.isTargetFile(ctx, file) {
		return
	} else if !file.exists() && gc.discard {
		return
	} else if gc.files = append(gc.files, file); false && gc.touch {
		// FIXED: Directly use the 64-bit integer primitive!
		var targetNano = gc.targetMtime
		
		if !file.exists() && !file.isSysFile() {
			file.stat(ctx)
			if false || gc.debug>0 {
				debug(ctx, "'%v' info is nil (%s)", file, file.fullname(), callstack{num:gc.debug})
			}
		}
		
		if !file.exists() {/* ... */} else
		if file._mtime > targetNano { // Pure Integer math!
			if true || gc.debug>0 {
				debug(ctx, "touch %v → %v (%v)", gc.target, file, time.Unix(0, file._mtime), callstack{num:gc.debug})
			}
			tv := _universe(ctx).launchTime // time.Now()
			if err = os.Chtimes(gc.targetFullName, tv, tv); err != nil {
				erro(ctx, "chtimes failed: %v", err)
			} else {
				// FIXED: Update the cached primitive directly instead of nil-ing an interface!
				gc.targetMtime = tv.UnixNano()
			}
		}
	}

	// Report missing files, but system files are not treated as missing.
	if !gc.report {
		// ...
	} else if file == nil {
		info(ctx, "%s: `%s` not found", _project(ctx).name, name)
	} else if !file.exists() {
		info(ctx, "%s: `%s` file not existed", _project(ctx).name, name)
	}
	return
}

func tempfile(ctx Context, prefix, hashee0 string, hasheeN... any) (file *file, err error) {
    var nameHash = sha256.New() // HashByte -> [sha256.Size]byte
    if _, err = fmt.Fprint(nameHash, prefix, hashee0); err != nil {
        erro(ctx, "hashing failed: %v", err)
    } else if _, err = fmt.Fprint(nameHash, hasheeN...); err != nil {
        erro(ctx, "hashing failed: %v", err)
    } else if nameSum := nameHash.Sum(nil); len(nameSum) != nameHash.Size() {
        erro(ctx, "hash sum invalid: %v", len(nameSum))
    } else if project := _project(ctx); project == nil {
        erro(ctx, "current project is nil: %v", ctx)
    } else {
        // Make names like .deps/00/da/bef0cc203d80fa25e0e2d3760518ee1b16bd641f99b9059468cfbbe8f096
        // .deps/??/??/????????????????????????????????????????????????????????????
        // .grep/??/??/????????????????????????????????????????????????????????????
        // .cache/??/??/????????????????????????????????????????????????????????????
        file = project.tempfile(ctx, filepath.Join(prefix, // e.g. ".deps", ".grep"
            fmt.Sprintf("%x", nameSum[ :1]),
            fmt.Sprintf("%x", nameSum[1:2]),
            fmt.Sprintf("%x", nameSum[2: ]),
        ))
    }
    return
}

func removeTempDirs(ctx Context, cleanDirs ...string) {
    var uni = _universe(ctx)
    if len(cleanDirs) == 0 {
        var clean =  uni.cleanTmpDirs
        if  clean || uni.cleanDotCache { cleanDirs = append(cleanDirs, ".cache") }
        if  clean || uni.cleanDotDeps  { cleanDirs = append(cleanDirs, ".deps") }
        if  clean || uni.cleanDotGrep  { cleanDirs = append(cleanDirs, ".grep") }
    }
    for _, dir := range cleanDirs {
        if file, err := tempfile(ctx, dir, ""); err != nil {
            erro(ctx, "%v", err)
        } else if s := file.fullname(); s == "" {
            erro(ctx, `"%v" has no fullname`, file)
        } else if s = filepath.Dir(filepath.Dir(filepath.Dir(s))); s == "" {
            erro(ctx, `"%v" is invalid temp dir`, file.fullname())
        } else if err = os.RemoveAll(s); err != nil {
            erro(ctx, "%v", err)
        } else if false {
            debug(ctx, "%s: removed %v", _project(ctx), s)
        } else {
            prompt(ctx, "%s: removed %v\n", _project(ctx), s)
        }
    }
}

func getSavedDepsFileName(ctx Context, targetFullName string, strs []string) (filename string, err error) {
    var ( file *file; hashees []any )
    for _, s := range strs { hashees = append(hashees, s) }
    if file, err = tempfile(ctx, ".deps", targetFullName, hashees...); err != nil {
        erro(ctx, "get .deps temp file failed: %v", err)
    } else {
        filename, _ = as_fullname_string(ctx, file)
    }
    return
}

func getSavedGrepFileName(ctx Context, targetFullName string) (filename string, err error) {
    var ( file *file )
    if file, err = tempfile(ctx, ".grep", targetFullName); err != nil {
        erro(ctx, "get .grep temp file failed: %v", err)
    } else {
        filename, _ = as_fullname_string(ctx, file)
    }
    return
}

func loadSavedGrepFile(ctx Context, gc *grepctx) (okay bool, err error) {
	if gc.savedGrepFileName, err = getSavedGrepFileName(ctx, gc.targetFullName); err != nil {
		erro(ctx, "get saved grep filename failed: %v", err)
	} else if gc.savedGrepFile = _stat(ctx, gc.savedGrepFileName); gc.savedGrepFile == nil {
		return // No saved grepfile yet!
	}

	var f, ok = to_file(gc.target)
	if !ok {
		f = _stat(ctx, gc.targetFullName)
		if f != nil { gc.target = f }
	}
	
	// Pure integer comparison replaces t.After()
	if f != nil && f.exists() {
		// Check previously saved grep file info.
		if gc.savedGrepFile.exists() && f._mtime > gc.savedGrepFile._mtime {
			return
		}
	}

	var savedGrepOSFile *os.File
	if savedGrepOSFile, err = os.Open(gc.savedGrepFileName); err != nil {
		erro(ctx, "open saved grep filename failed: %v", err)
	}
	defer savedGrepOSFile.Close()

	var gp Position
	gp.Filename = gc.targetFullName // gc.savedGrepFileName

	scanner := bufio.NewScanner(savedGrepOSFile)
	// Allocate a 64KB initial buffer, but allow it to grow up to 10MB per line!
	const maxCapacity = 10 * 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxCapacity)
	scanner.Split(bufio.ScanLines)
	
	for scanner.Scan() {
		var s = scanner.Text() //gp.Line += 1
		var ( sys int; name string )
		if n, e := fmt.Sscanf(s, "%d %d %d %s", &sys, &gp.Line, &gp.Column, &name); e == nil && n == 4 {
			var f *file
			if f, err = searchGrepped(ctx, gp, gc, sys == 1, name); err != nil {
				erro(ctx, "search grepped filename failed: %v", err)
			} else if f != nil {
				if /* f.pos = gp */; gc.isTargetFile(ctx, f) { continue }
			} else if sys != 1 && !gc.discard {
				debug(ctx,
					_f("%s is nil file", name),
					_f("grepped %s is nil", name),
					_f("from project %v", _project(ctx)))
			}
		}
	}
	
	// Replaced direct interface assignment with primitive extraction!
	if info, err := savedGrepOSFile.Stat(); err != nil {
		erro(ctx, "stat saved grep filename error: %v", err)
	} else {
		gc.savedGrepFile._mtime = info.ModTime().UnixNano()
		gc.savedGrepFile._size  = info.Size()
		gc.savedGrepFile._isDir = info.IsDir()
		okay = true
	}
	return
}

func grepTargetFile(ctx Context, gc *grepctx) (err error) {
    var ( f *os.File )
    if f, err = os.Open(gc.targetFullName); err != nil {
        erro(ctx, "%v", err)
    } else { defer func() { err = f.Close() } () }

    var gp Position
    gp.Filename = gc.targetFullName

    scanner := bufio.NewScanner(f)
	// Allocate a 64KB initial buffer, but allow it to grow up to 10MB per line!
    const maxCapacity = 10 * 1024 * 1024
    buf := make([]byte, 0, 64*1024)
    scanner.Buffer(buf, maxCapacity)
    scanner.Split(bufio.ScanLines)
ForScan:
    for scanner.Scan() {
        var s = scanner.Text(); gp.Line += 1
        for _, x := range gc.rxs {
            if sm := x.FindStringSubmatch(s); len(sm) > 1 && sm[1] != "" {
                var ( f *file ; name = sm[1]; sys = x.bool ) //strings.IndexFunc(s, isNotSpace)
                if gp.Column = strings.Index(s, name); gc.save != nil {
                    var d = 0 ; if sys { d = 1 } // system files
                    fmt.Fprintf(gc.save, "%d %d %d %s\n", d, gp.Line, gp.Column, name)
                }
                if f, err = searchGrepped(ctx, gp, gc, sys, name); err != nil {
                    erro(ctx, "search grepped '%s' failed: %v", name, err)
                } else if f != nil {
                    if /* f.pos = gp */; gc.isTargetFile(ctx, f) { continue }
                } else if !sys && !gc.discard {
                    debug(ctx,
						_f("%s is nil file", name),
						_f("grepped %s is nil", name),
						_f("from project %v", _project(ctx)))
                }
                continue ForScan // found one
            }
        }
    }
    return
}

func grep(ctx Context, gc *grepctx) (err error) { // TODO: using ctx.grepping() to replace grepctx
	var targetName string
	
	switch v := gc.target.(type) {
	case *file:
		targetName = v.name.String() // Direct Walled Garden access, bypassing ident() overhead
		gc.targetMtime = v._mtime    // O(1) primitive assignment
		gc.targetFullName = v.fullname()
		gc.targetDir = filepath.Dir(gc.targetFullName)
		if v.isSysFile() { return }
	default:
		gc.targetDir = _project(ctx).absPath
		targetName = __string(ctx, v)
		if filepath.IsAbs(targetName) {
			gc.targetFullName = targetName
		} else {
			gc.targetFullName = filepath.Join(gc.targetDir, targetName)
		}
		
		if file := _stat(ctx, gc.targetFullName); file == nil || !file.exists() {
			erro(ctx, "grep: '%s' not found (%v)", gc.targetFullName, gc.target)
		} else {
			gc.targetMtime = file._mtime // Map directly to primitive
		}
	}
	
	if err != nil {
		erro(ctx, "grep target %s: %v", targetName, err)
	}

	// Replaces `gc.targetInfo == nil` with our raw integer zero-check
	if gc.targetMtime == 0 { return } 
	
	if gc.done == nil { gc.done = make(map[string]int) }
	if !filepath.IsAbs(gc.targetFullName) {
		erro(ctx, "grep: '%s' is not abs", gc.targetFullName)
	} else {
		gc.done[gc.targetFullName] += 1
	}
	if n, done := gc.done[gc.targetFullName]; done && n > 1 {
		if gc.debug>0 {
			erro(ctx, "%v (done %v)", gc.targetFullName, n)
		}
		return
	}

	const infos = false

	if false { defer un(tt(l_traverse, _execution(ctx), gc.target)) }

	defer func(restore []Value) {
		var t = _execution(ctx)
		var touch = gc.greptouch // copy greptouch value
		if len(touch.files) > 0 {
			grepcacheM.Lock()
			grepcache[gc.targetFullName] = touch.files
			grepcacheM.Unlock()
		} else if false {
			var gp Position
			gp.Filename, gp.Line = gc.targetFullName, 1
			debug(ctx, "grepped zero files: %v", gc.targetFullName)
		}
		gc.files = restore
		if gc.debug>0 {
			debug(ctx, "grepped: %s → %v (grepped=%v) (saved=%s)\n",
				gc.target, touch.files, len(t.grepped), gc.savedGrepFile,
				callstack{num:gc.debug})
		}
		for _, gc.target = range touch.files {
			if t.grepped = append(t.grepped, gc.target); !gc.recursive {
				continue
			} else if err = grep(ctx, gc); err != nil {
				erro(ctx, "grep files (deferred): %v", err)
			}
		}
		if err == nil && gc.touch {
			if err = touch.work(ctx, gc); err != nil {
				erro(ctx, "grep touch failed: %v", err)
			}
		}
	} (gc.files)

	gc.files = nil

	var (
		cached bool
		savedGrepFile *os.File
		savedGrepFileLoaded bool
	)
	{
		grepcacheM.Lock()
		gc.files, cached = grepcache[gc.targetFullName]
		grepcacheM.Unlock()
	}
	if cached && len(gc.files) > 0 {
		if gc.debug>0 {
			erro(ctx, "grepcache: %v → %v", gc.targetFullName, gc.files)
		}
		return
	} else if infos {
		debug(ctx, "grepcache: %s files=%d", gc.targetFullName, len(gc.files))
	}

	if savedGrepFileLoaded, err = loadSavedGrepFile(ctx, gc); err != nil {
		erro(ctx, "load saved grepfile failed: %v", err)
	} else if savedGrepFileLoaded && len(gc.files) > 0 {
		if infos {
			debug(ctx, "loadSavedGrepFile: %v files=%d grepped=%d",
				gc.targetFullName, len(gc.files), len(_execution(ctx).grepped))
		}
		return
	}
	if dir := filepath.Dir(gc.savedGrepFileName); dir != "." && dir != ".." {
		if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
			erro(ctx, "make grep dir failed: %v", err)
		}
	}

	var uni = _universe(ctx)
	if uni.saveGrepSource {
		var (
			perm = os.FileMode(0600)
			data = []byte(gc.targetFullName)
			name = gc.savedGrepFileName + ".src"
		)
		// Upgraded to modern os.WriteFile (ioutil is deprecated)
		if err = os.WriteFile(name, data, perm); err != nil {
			erro(ctx, "grep write file: %v", err)
		} else if false {
			debug(ctx, "saved grep %s", name)
		}
	}
	if savedGrepFile, err = os.Create(gc.savedGrepFileName); err != nil {
		erro(ctx, "grep create %s: %v", gc.savedGrepFileName, err)
	}

	gc.save = bufio.NewWriter(savedGrepFile)
	defer func() {
		gc.save.Flush()
		savedGrepFile.Close()
	} ()

	if err = grepTargetFile(ctx, gc); err != nil && !gc.discard {
		erro(ctx, "grep target file: %v", err)
	} else {
		err = nil // discard any errors
	}
	return
}

var stopgrep = 0

// grep - grep files from target, example usage:
//
//      (grep -file -x='\s*#\s*include\s*<(.*)>')
//
// https://github.com/google/re2/wiki/Syntax
type modifier_grep struct { modifier_
    discard bool `c,cast;dc,discard;dm,discard-missing;im,ignore-missing`
    fileinc bool `f,file;f,files` // work with the 'incs' field TODO: = true
    langs []string `l,lang;lan,language`
    sys []string `s,sys;ss,system`        // matching system includes
    reg []string `re,reg;regx,regex;x,rx` // matching user includes
    incs []Value `i,inc;i,include` // include search paths, also 'fileinc' field
    touch bool `t,touch;t,touch-outdate;t,touch-outdated`
    recursive bool `a,all;r,recur;rr,recursive`
    noTraverse bool `n,notraverse;nt,no-traverse;go,grep-only`
}
func (ctx *modifier_grep) x(args ...Value) (result any) {
    var uni = _universe(ctx)
    if false && uni.noDepsGrep || uni.noGrep { return }

    var gc = grepctx{ modifier_grep:ctx }
    // gc.fileinc = true // grep files by default
    gc.incs = xmerge(ctx, gc.incs...)//, plain
    for _, s := range gc.sys { gc.rxs = append(gc.rxs, &greprex{true , regexp.MustCompile(s)}) }
    for _, s := range gc.reg { gc.rxs = append(gc.rxs, &greprex{false, regexp.MustCompile(s)}) }
    for _, s := range gc.langs {
        if info, ok := langInfos[s]; ok && info != nil {
            for _, re := range info.rxs { gc.rxs = append(gc.rxs, &greprex{false, re}) }
            for _, re := range info.sys { gc.rxs = append(gc.rxs, &greprex{true , re}) }
        } else {
            erro(ctx, "lang '%s' is unknown", s)
        }
    }
    if len(gc.rxs) == 0 {
        erro(ctx, "no grep expressions: %v %v %v %v", gc.sys, gc.reg, gc.langs, args)
    }

    var (
        target = auto_get(ctx, symAt)
        targets = args
        grepped = _execution(ctx).grepped
    )
    if len(targets) == 0 {
        if target == nil || isNull(target) || isNone(target) {
            erro(ctx, "no grep target")
        } else {
            targets = append(targets, target)
        }
    }

    if gc.debug > 0 {
        debug(ctx, "grep files: %v %v %v\n", target, gc.rxs, args, callstack{num:gc.debug})
    }
    if gc.verbose {
        defer func(ts time.Time) {
            var s string
            if len(targets) == 1 { s = targets[0].String() } else {
                for _, v := range targets {
                    if s != "" { s += ", " }
                    if len(s) > 32 { s += "..."; break } else {
                        s += v.String()
                    }
                }
            }
            debug(ctx, "Grep %v …… (%d files in %v)\n", s, len(grepped), time.Now().Sub(ts))
        } (time.Now())
    }

    var pc = _execution(ctx)
    var tar = target
    defer func(v bool) { pc.grepping = v } (pc.grepping)
    pc.grepping = true

    for _, target := range targets {
        if isNull(target) {
            erro(ctx, "found nil grep target for %v", tar)
        }
        if isNone(target) {
            erro(ctx, "grep target '%v' is none for %v", target, tar)
        }

        gc.target, pc.grepped = target, nil
        if err := grep(ctx, &gc); err != nil {
            erro(ctx, "grep files from %v failed: %v", target, err)
        } else if gc.noTraverse {
            // does nothing
        } else if len(pc.grepped) > 0 {
            for _, val := range pc.grepped {
                traverse(ctx, val)
            }
        }
        grepped = append(grepped, pc.grepped...)
    }
    pc.grepped = grepped

    if !gc.noTraverse {
        auto_set(ctx.Context, defVoid, symTilde, _none(_pos(ctx)))
        pc.grepped = nil
    } else {
        result = ease(ctx, pc.grepped)
    }
    return
}

type dep_context struct { diagnostic }
func (ctx *dep_context) inner() Context { return &ctx.diagnostic }
func (ctx *dep_context) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.diagnostic.cast(t)
}

func parseDeps(ctx Context, targetVal Value, targetStr string, savedDepsFile *file, savedDepsFileName, deps string) (files []Value) {
	var (
		proj = _project(ctx)
		targetFullName, _ = as_fullname_string(ctx, targetVal)
		filesMux sync.Mutex
		firstWord string
		err error
	)
	var findDepFile = func(name string) (file *file) {
		if filepath.IsAbs(name) {
			file = _stat(ctx, name, stat_nonexist{true})
		} else if file = proj.file(ctx, name); file != nil && file.exists() {
			// good!
		} else {
			// fail!
		}
		return
	}
	var ignored = func(fullname string) (res bool) {
		if fullname == targetFullName { return true }
		return
	}
	var addFile = func(file *file) {
		filesMux.Lock()
		files = append(files, file)
		filesMux.Unlock()
	}
	var (
		missing = make(map[string]Position)
		missMux sync.Mutex
	)

	var depFile = func(ctx Context, depPos Position, word string) {
		var dc = dep_context{diagnostic{ Context: ctx }}

		ctx = &dc

		if i := strings.Index(word, " "); i > 0 {
			debug(ctx, "ignore dep with spaces: %v", word)
		} else if file := findDepFile(word); file == nil {
			prompt(ctx, "%v: unknown dep\n", file)
			if savedDepsFile != nil {
				warn(ctx, "unknown dep '%v' for '%v'", word, firstWord)
				warn(ctx, "from here: %s", word)
				if filepath.IsAbs(firstWord) {
					var wp Position
					wp.Filename, wp.Line = firstWord, 1
					warn(ctx, "in here: %v", word)
				}
				debug(ctx, "for project %v", proj)
			} else {
				debug(ctx, "unknown dep '%v' for '%v'", word, firstWord)
				debug(ctx, "from here: %s", word)
				if filepath.IsAbs(firstWord) {
					var wp Position
					wp.Filename, wp.Line = firstWord, 1
					debug(ctx, "in here: %v", word)
				}
				debug(ctx, "for project %v", proj)
			}
		} else if ignored(file.fullname()) {
			//continue // dep is the target itself
		} else {
			traverse(ctx, file)
			addFile(file)
		}

		var n int
		if savedDepsFile == nil {
			if n = flush(dc.Context); n > 0 { // aka. dc.points = nil
				var s = trimPromptString(targetVal.String())
				prompt(ctx, "%v: %d errors counted\n", word, n)
				debug(ctx, `%v: %d errors for "%s", dep "%s"`, proj, n, s, word)
				erro(ctx, `%v: %v`, ctx)
			}
		} else {
			if n = diagCount(dc.Context, diagError); n > 0 {
				// reset to reduce diags as we wish to continue with the errors
				dc.points, dc.erros = nil, 0
				debug(ctx, "%v: %d errors counted\n", word, n)
			}
		}
		if n > 0 {
			missMux.Lock()
			missing[word] = depPos
			missMux.Unlock()
		}
		return
	}

	var (
		wordRecs = make(map[string]int)
		firstDep string
		depPos Position
	)
	depPos.Filename = savedDepsFileName
	for l, line := range strings.Split(deps, "\n") {
		var words = line
		if i := strings.Index(words, ":"); i > 0 { words = strings.TrimSpace(words[i+1:]) }
		if words = strings.TrimSpace(strings.TrimRight(words, "\\\r\t ")); words == "" {
			continue // empty line
		}
		for _, word := range strings.Fields(words) {
			depPos.Line, depPos.Column = l + 1, strings.Index(line, word) + 1
			if /*l == 1 && w == 0 &&*/firstWord == "" { firstWord = word }
			if wordRecs[word] += 1; wordRecs[word] == 1 {
				if firstDep != "" {
					// keep going...
				} else if firstDep = word; savedDepsFile == nil {
					// no need to compare
				} else if firstDepFile := _stat(ctx, firstDep); firstDepFile == nil || !firstDepFile.exists() {
					return nil // requests to update savedDepsFile
				} else if firstDepFile._mtime > savedDepsFile._mtime { // PURE O(1) INTEGER MATH!
					return nil // requests to update savedDepsFile
				}
				depFile(ctx, depPos, word)
			}
		}
	}
	if len(missing) > 0 {
		prompt(ctx, "%v: %d deps missing, removing deps file\n", savedDepsFileName, len(missing))
		if savedDepsFile == nil || savedDepsFileName == "" {
			// deps files not saved yet
		} else if err = os.Remove(savedDepsFileName); err != nil {
			for s, _ := range missing { debug(ctx, `missing "%v"`, s) }
			debug(ctx, `%v: "%v" %d deps missing in "%v"`, proj, targetVal, len(missing), savedDepsFileName)
			erro(ctx, "%s", ts(ctx))
		} else {
			for s, _ := range missing { warn(ctx, `missing "%v"`, s) }
			debug(ctx, `%v: "%v" missing %d deps (%v in total)`, proj, targetVal, len(missing), len(files))
			files = nil // To update savedDepsFileName
		}
	}
	return
}

func loadSavedDepsAndCheckOutdated(ctx Context, args []string) (savedDepsFileName string, files []Value) {
	var (
		savedDepsBytes []byte
		err error
	)
	if targetVal, targetStr := auto_target_valstr(ctx); targetVal == nil {
		erro(ctx, "target is nil")
	} else if targetStr == "" {
		erro(ctx, "target '%v' is empty", targetVal)
	} else if savedDepsFileName, err = getSavedDepsFileName(ctx, targetStr, args); err != nil {
		erro(ctx, "get saved deps filename failed: %v", err)
	} else if savedDepsFileName == "" {
		erro(ctx, "empty saved deps filename", savedDepsFileName)
	// Safety fix: explicitly check .exists() instead of just checking if the struct is allocated
	} else if savedDepsFile := _stat(ctx, savedDepsFileName); savedDepsFile == nil || !savedDepsFile.exists() {
		// no saved deps file
	} else if savedDepsBytes, err = os.ReadFile(savedDepsFileName); err != nil {
		erro(ctx, "can't open saved deps file: %v", savedDepsFileName, err)
	} else if files = parseDeps(ctx, targetVal, targetStr, savedDepsFile, savedDepsFileName, string(savedDepsBytes)); len(files) > 0 {
		if false { debug(ctx, "loaded deps %s (%d files)", savedDepsFileName, len(files)) }
		
		var savedDepsFileModTime = savedDepsFile._mtime // Map directly to primitive int64
		
		for _, val := range files { 
			if file, ok := to_file(val); !ok {
				// ignore
			} else if !file.exists() || file._mtime > savedDepsFileModTime { // Zero interface allocation!
				files = nil // need to reload if outdated or missing
				return
			}
		}
	}
	return
}

func traverseMissingDep(ctx Context, dep string) (res bool) {
    if proj := _project(ctx); proj == nil {
        erro(ctx,
			_f("%s: traverse dep failed, project %v\n", dep, proj),
			_f("%s: no current project for dep", dep))
    } else if f := proj.file(ctx, dep); f == nil {
        erro(ctx,
			_f("%s: dep is unknown file; project %v\n", dep, proj),
			_f("%v: %s is unknown file", proj, dep))
    } else {
        traverse(ctx, f)
    }
    return true
}

func traverseMissingDeps(ctx Context, lastTry string, errBytes []byte) (res bool, tried string) {
    const promptErrors bool = false
    const promptBeforeTraverse bool = promptErrors && true
    for _, m := range rxFileNotFound.FindAllSubmatch(errBytes, -1) {
        if promptBeforeTraverse { prompt(ctx, "%s\n", m[0]) }
        if dep := string(m[4]); dep == lastTry {
            return false, ""
        } else if res = traverseMissingDep(ctx, dep); !res {
            prompt(ctx, "%s: dep missing, project %v\n", m[4], _project(ctx))
            prompt(ctx, "%s\n", m[0]) // prompt the entire error line
            erro(ctx, "%v", ctx)
        } else if tried == "" { tried = dep }
    }
    return
}

type modifier_deps struct { modifier_
    useClang bool `cl,clang`
    useGcc bool `gcc`
    addMissing bool `am,add-missing,mg,missing-goal`
    lang string `lang,language`
    flags []Value `flags,opts`
    cc string `cc,compiler`
}
func (ctx *modifier_deps) x(args ...Value) (result any) {
    var uni = _universe(ctx)
    if uni.noDepsGrep || uni.noDeps { return }

    // NOTE: parse opts for (deps) before expanding the args, because we share args
    //       with the compilers!
    var err error
    var targetVal Value
    var targetStr string
    if targetVal, targetStr = auto_target_valstr(ctx); targetVal == nil {
        erro(ctx, "target is nil")
    } else if targetStr == "" {
        erro(ctx, "target '%v' is empty", targetVal)
    }

    var files []Value
    if ctx.verbose {
        defer func(ts time.Time) {
            var s string
            if val := auto_get(ctx, symAt); val != nil { s = val.String() }
            prompt(ctx, "Deps %v …… (%d files in %v)\n", s, len(files), time.Now().Sub(ts))
        } (time.Now())
    }

CorrectCC:
    switch ctx.cc {
    case "cl"   : ctx.cc = "clang"; goto CorrectCC
    case "gc"   : ctx.cc = "gcc"  ; goto CorrectCC
    case "clang": ctx.useClang = true
    case "gcc"  : ctx.useGcc   = true
    case "":
        if ctx.useGcc   { ctx.cc = "gcc" }
        if ctx.useClang { ctx.cc = "clang" }
    default:
        if base := filepath.Base(ctx.cc); base == "" {
            erro(ctx, "unsupported cc: %v", ctx.cc)
        } else if strings.HasPrefix(base, "clang") { ctx.useClang = true
        } else if strings.HasPrefix(base, "gcc")   { ctx.useGcc   = true }
    }

    var _MM, _MG bool
    var ca []string
    var flags = xmerge(_final(ctx), ctx.flags...)
    for _, f := range flags {
        switch s := strings.TrimSpace(__string(ctx, f)); s {
        case "-MM": ca, _MM = append(ca, s), true // only user headers
        case "-MD": break // discard, use -M or -MM instead
        case "-MP": break // discard, not creating phony target
        case "-MV": break // discard, not using NMake/Jom format
        case "-MG": break // discard, add later for missing headers
        case "-M" : break // discard, add later for both user and system headers
        case "-c" : break // discard, compile flag
        case ""   : break // discard, empty string
        default: ca = append(ca, s)
        }
    }
    if !_MM { ca = append(ca, "-M")  } // both user and system headers
    if !_MG && ctx.addMissing { ca = append(ca, "-MG") } // add missing headers
    for _, a := range args {
        var s, y = as_fullname_string(ctx, a)
        if y { s = strings.TrimSpace(s) }
        switch s {
        case "-M", "-MM", "-MG", "-MD", "-MV", "-MP", "-Os", "-O1", "-O2", "-O3",
            "-c", "-shared", "-static", "-fPIC", "-fvisibility-inlines-hidden",
            "-fcxx-modules", "-fmodules", "-fmodules-ts", "":
            break // discard unused args
        default: ca = append(ca, s)
        }
    }

    var proj = _project(ctx)

    savedDepsFileName, files := loadSavedDepsAndCheckOutdated(ctx, ca)

    if len(files) == 0 {
        var (
            cc = exec.Command(ctx.cc, ca...)
            stdout bytes.Buffer
            stderr bytes.Buffer
            retried string
        )
    retryCC:
        cc.Stdout, cc.Stderr = &stdout, &stderr
        if err = cc.Run(); err != nil {
            var okay = false
            if okay, retried = traverseMissingDeps(ctx, retried, stderr.Bytes()); okay {
                cc = exec.Command(ctx.cc, ca...)
                stdout.Reset()
                stderr.Reset()
                goto retryCC
            }
            prompt(ctx, "%v: failed command '%s':\n", proj, ctx.cc)
            prompt(ctx, "%s \\\n  %s\n----------\n", cc.Path, strings.Join(ca, " \\\n  "))
            prompt(ctx, "%s\n----------\n%s----------\n", &stdout, &stderr)
            debug(ctx, "%s: %s deps failed: %v", proj, filepath.Base(ctx.cc), err)
            erro(ctx, "%s: %v", proj, ctx)
        }
        if stderr.Reset(); savedDepsFileName == "" {
            stdout.Reset()
            erro(ctx, "empty saved deps file name: %v", savedDepsFileName)
        }

        var savedDepsFile *file = nil//_stat(ctx, savedDepsFileName)
        if files = parseDeps(ctx, targetVal, targetStr, savedDepsFile, savedDepsFileName, stdout.String()); len(files) == 0 {
            debug(ctx, "parse deps file failed") // not saving if failed
        } else if err = os.MkdirAll(filepath.Dir(savedDepsFileName), os.FileMode(0755)); err != nil {
            erro(ctx, "make path '%s' failed: %v", filepath.Dir(savedDepsFileName), err)
        } else if err = ioutil.WriteFile(savedDepsFileName, stdout.Bytes(), os.FileMode(0666)); err != nil {
            erro(ctx, "save deps file failed: %v", err)
        }
        stdout.Reset() // release buffers (optional)
    }

    if t := _execution(ctx); t != nil && len(files) > 0 {
        t.grepped = append(t.grepped, files...)
    }
    return
}

type modifier_touch struct { modifier_
    path bool `p,path`
    mode os.FileMode `m,mode`
}
func (ctx *modifier_touch) x(args ...Value) (result any) {
    if len(args) == 0 { if val := auto_get(ctx, symAt); val != nil { args = append(args, val) }}

    var files []*file
    for _, arg := range args {
        if err := touch(ctx, arg, uint32(ctx.mode), ctx.path); err != nil {
            erro(ctx, "touch '%v' failed: %v", arg, err)
        } else {
            files = append(files, stampFile(stamp_file_ctx{ctx}, arg)...)
        }
    }

    var p = _program(ctx)
    if false && ctx.verbose { reportFileUpdates(ctx, files) }
    if len(p.getModifiers(ctx, "stamp")) > 0 {
        debug(ctx, "no need to use a (stamp) after (touch)")
    }
    return
}

// (check status=1 stdout="foobar" stderr="")
// (check file=filename.txt)
// (check dir=directory)
// (check var=(NAME,VALUE))
type modifier_check struct { modifier_
    trim bool `trim,trim-string`
    answer bool `answer`
    boolean bool `bool,boolean,res,result`
    silent bool `slient`
    exists bool `exist,exists`
    regular bool `reg,regular`
    isdir bool `isdir,is-dir`
    good bool `good`
    file Value `file`
    dir Value `dir`
}
func (ctx *modifier_check) x(args ...Value) (_ any) {
	var pos = _pos(ctx)
	var makeResult func(bool) Value // returns results only if non-nil
	if ctx.answer {
		makeResult = func(v bool) Value { return _answer(pos, v) }
	} else if ctx.boolean ||
		(ctx.file != nil && (ctx.exists || ctx.regular || ctx.isdir)) ||
		(ctx.dir  != nil && (ctx.exists || ctx.regular || ctx.isdir)) {
		makeResult = func(v bool) Value { return _boolean(pos, v) }
	}

	var res bool
	var values []Value
	var checkfile = func (val Value, dir bool) {
		if val == nil {
			erro(pc(ctx, val), "nil file value to check")
		} else if x, y := val.(*boolean); y {
			if x.bool { val = auto_get(ctx, symAt) } else { val = nil }
		}

		var s string
		var f *file
		if f, res = to_file(val); res {
			// best case
		} else if s = __string(ctx, val); filepath.IsAbs(s) {
			if f = _stat(ctx, s); f != nil { res = true }
		} else if f = findfile(ctx, s); f != nil { res = true }

		if f != nil {
			if !dir || ctx.regular {
				res = f.exists()
			} else if dir || ctx.isdir {
				// UPGRADED: Replaced interface call with pure primitive boolean
				res = f.exists() && f._isDir
			} else if ctx.exists {
				res = f.exists()
			}
		}

		if makeResult != nil {
			values = append(values, makeResult(res))
		} else if !res {
			erro(pc(ctx, val), "'%v' is not file", val)
		}
	}

	if ctx.file != nil { checkfile(ctx.file, false) }
	if ctx.dir  != nil { checkfile(ctx.dir, true) }

	var program = _program(ctx)
	var value = auto_get(ctx, symDash)

argsloop:
	for _, arg := range args {
		var p, y = arg.(*pair)
		if !y {
			if res = __true(ctx, arg); makeResult != nil {
				values = append(values, makeResult(res))
			} else {
				erro(ctx, "value '%v' is false", arg)
			}
			continue
		}

		var key, str string
		switch key = __string(ctx, p.key); key {
		case "status":
			var exeres, _ = value.(*exec_result)
			if exeres == nil {
				erro(ctx, "not exec result: %v ", ts(value))
			}

			var num = __int(ctx, p.val)
			if ctx.verbose {
				prompt(ctx, "checking status ")
				if num != 0 { prompt(ctx, "== %d ", num) }
				prompt(ctx, "…")
			}

			var good = exeres.Status == int(num)
			if ctx.verbose {
				var s string
				if good { s = "yes" } else { s = "no" }
				prompt(ctx, "… %s (%d)\n", s, exeres.Status)
			}

			if ctx.debug > 0 {
				var tar = auto_get(ctx, symAt)
				var val = auto_get(ctx, symDash)
				debug(ctx,
					_f("%v: %v", _entry(ctx), tar),
					_f("hyphen=%v", val),
					_f("status=%v", exeres.Status))
			}

			if makeResult != nil {
				values = append(values, makeResult(good))
			} else if !good {
				erro(ctx, "bad status (%v) (expects %v)", exeres.Status, p.val)
				break argsloop
			}
		case "stdout", "stderr":
			var exeres, _ = value.(*exec_result)
			if exeres == nil {
				erro(ctx, "value '%v' (%T) is not exec result", value, value)
			} else { /*exeres.wg.Wait()*/ }

			if ctx.verbose {
				prompt(ctx, "checking %s (status=%d) … ", key, exeres.Status)
			}

			if 0 < ctx.debug {
				var tar = auto_get(ctx, symAt)
				var val = auto_get(ctx, symDash)
				debug(ctx,
					_f("%v: %v", _entry(ctx), tar),
					_f("hyphen=%v", val),
					_f("status=%v", exeres.Status),
					callstack{num:ctx.debug})
			}

			var v *bytes.Buffer
			switch key {
			case "stdout": v = exeres.Stdout.Buf
			case "stderr": v = exeres.Stderr.Buf
			default: unreachable()
			}

			if v == nil {
				erro(ctx, "bad %s (expects %v)", key, p.val)
				break argsloop
			}

			str = __string(ctx, p.val)
			if ctx.trim { str = strings.TrimSpace(str) }

			if res := v.String() == str; makeResult != nil {
				values = append(values, makeResult(res))
			} else if !res {
				erro(ctx, "bad %s (%v) (expects %v)", key, v, p.val)
				break argsloop
			}
		case "file", "dir": // file=xxx and dir=xxx, same as -file=xxx and -dir=xxx
			var ( f *file; res bool )
			if f, res = to_file(p.val); res {
				// ok
			} else if str = __string(ctx, p.val); filepath.IsAbs(str) {
				if f = _stat(ctx, str); f != nil {
					// ok
				}
			} else if f = findfile(ctx, str); f != nil {
				// ok
			}
			
			// UPGRADED: 0-allocation primitive bounds checking!
			switch key {
			case "file": res = f != nil && f.exists() && !f._isDir 
			case "dir":  res = f != nil && f.exists() && f._isDir
			default: unreachable()
			}
			
			if makeResult != nil {
				values = append(values, makeResult(res))
			} else if !res {
				erro(ctx, "`%v` is not %s", p.val, key)
				break argsloop
			}
		case "var":
			var g, ok = p.val.(*group)
			if !ok {
				erro(ctx, "`%v` is not a group value", p.val)
				break argsloop
			}
			for _, elem := range g.elems {
				switch p := elem.(type) {
				case *pair:
					var a, b string
					var def = program.project.finddef(__symbol(ctx, p.key))
					if def != nil {
						a = __string(ctx, p.val)
						b = __string(ctx, def.value)
						if res := a != b; makeResult != nil {
							values = append(values, makeResult(res))
						} else if !res {
							erro(ctx, "`%v` != `%v`", p.key, p.val)
							break argsloop
						}
					} else if makeResult != nil {
						values = append(values, makeResult(false))
					} else {
						erro(ctx, "`%v` is not defined", p.key)
						break argsloop
					}
				default:
					erro(ctx, "`%v` unsupported checks", elem)
					break argsloop
				}
			}
		default:
			erro(ctx, "unknown check for %v → %v", p.key, p.val)
			break argsloop
		}
	}
	return values
}

type copyopts struct {
    program *program
    path, update bool
    mode os.FileMode
    head Value
    foot Value
    files, copied int
    bytes int64
}

func copyRegular(ctx Context, src, dst string, opts *copyopts) (err error) {
    var def1 = auto_find(ctx, intern("1"))
    var def2 = auto_find(ctx, intern("2"))
    defer func(v1, v2 Value) { def1.value, def2.value = v1, v2 } (def1.value, def2.value)

    var pos = _pos(ctx)
    def1.value = _strlit(pos, dst)
    def2.value = _strlit(pos, src)

    var head, foot string
    if opts.head != nil { head = __string(ctx, opts.head) }
    if opts.foot != nil { foot = __string(ctx, opts.foot) }

    // Compare mod time for update mode
    if opts.files += 1; opts.update {
        if st2, e := os.Stat(dst); e == nil && st2 != nil {
            var st1 os.FileInfo
            if st1, err = os.Stat(src); err != nil { debug(ctx, "%v", err); return }
            if st1 != nil && (st1.Size()+int64(len(head))+int64(len(foot))) == st2.Size() {
                if st2.ModTime().After(st1.ModTime()) { return }
            }
            if false { prompt(ctx, "%s: %s (%v,%v)\n", pos, dst, st1.Size(), st2.Size()) }
        }
    }

    var srcFile, dstFile *os.File
    if srcFile, err = os.Open(src); err != nil { debug(ctx, "%v", err); return } else {
        defer srcFile.Close()
    }

    // sys default file mode is 0666
    if opts.path { // Make path (mkdir -p)
        if p := filepath.Dir(dst); p != "." && p != "/" {
            err = os.MkdirAll(p, os.FileMode(0755))
            if err != nil { debug(ctx, "%v", err); return }
        }
    }

    if opts.mode == 0 { opts.mode = os.FileMode(0640) }

    dstFile, err = os.OpenFile(dst, os.O_CREATE|os.O_RDWR|os.O_TRUNC, opts.mode)
    if err != nil { debug(ctx, "%v", err); return } else { defer dstFile.Close() }

    srcBuf := bufio.NewReader(srcFile)
    dstBuf := bufio.NewWriter(dstFile)
    if head != "" {
        var n int
        if n, err = dstBuf.WriteString(head); err != nil { debug(ctx, "%v", err); return }
        opts.bytes += int64(n)
    }

    var n int64
    if n, err = io.Copy(dstBuf, srcBuf); err != nil { debug(ctx, "%v", err); } else {
        if opts.bytes += n; foot != "" {
            var n int
            if n, err = dstBuf.WriteString(foot); err != nil { debug(ctx, "%v", err); return }
            opts.bytes += int64(n)
        }
        if err == nil {
			if err = dstBuf.Flush(); err != nil {
                debug(ctx, "flush failed during copy: %v", err)
                return
            }
            opts.copied += 1
        }
    }
    return
}

func copySymlink(ctx Context, src, dst string, opts *copyopts) (err error) {
    err = errors.New("copy symlink unimplemented")
    return
}

func copyDir(ctx Context, src, dst string, opts *copyopts) (err error) {
    if dst != "." && dst != "/" { // Make path (mkdir -p)
        err = os.MkdirAll(dst, os.FileMode(0755))
        if err != nil { return }
    }

    var fis []os.FileInfo
    if fis, err = ioutil.ReadDir(src); err != nil {
        return
    }
    for _, fi := range fis {
        ss := filepath.Join(src, fi.Name())
        sd := filepath.Join(dst, fi.Name())
        err = copyFile(ctx, fi, ss, sd, opts)
        if err != nil { break }
    }
    return
}

func copyFile(ctx Context, srcFi os.FileInfo, src, dst string, opts *copyopts) (err error) {
    if m := srcFi.Mode(); m&os.ModeSymlink != 0 {
        if opts.mode == 0 { opts.mode = srcFi.Mode() }
        err = copySymlink(ctx, src, dst, opts)
    } else if srcFi.IsDir() {
        err = copyDir(ctx, src, dst, opts)
    } else if m.IsRegular() {
        if opts.mode == 0 { opts.mode = srcFi.Mode() }
        err = copyRegular(ctx, src, dst, opts)
    } else {
        err = fmt.Errorf("copying non-regular files/dirs (%s)", src)
    }
    return
}

// (copy-file -p)
// (copy-file -p,filename)
// (copy-file -p,filename,source)
type modifier_copyfile struct { modifier_
	path bool "p,path"
	recursive bool "r,recursive"
	override bool "o,override"
	update bool "u,update"
	quick bool "q,quick"
	mode os.FileMode "m,mode"
	head Value "h,head"
	foot Value "f,foot"
}
func (ctx *modifier_copyfile) x(args ...Value) (result any) {
	var target Value
	var source Value
	if len(args) > 0 {
		target = args[0]
	} else {
		target = auto_get(ctx, symAt)
	}
	if len(args) > 1 {
		source = args[1]
	} else {
		source = auto_get(ctx, intern("<"))
	}

	// Get target and source filenames
	var (
		project = _project(ctx)
		filename, srcname string
		// UPGRADED: time.Time replaced by pure int64 primitives
		filemtime, srcmtime int64 
	)
	
	switch tv := target.(type) {
	case *file:
		filename = tv.fullname()
		if tv.exists() { filemtime = tv._mtime }
	default:
		filename = __string(ctx, target)
		if file := project.file(ctx, filename); file != nil {
			target, filename = file, file.fullname()
			if file.exists() { filemtime = file._mtime }
		}
	}
	
	switch tv := source.(type) {
	case *file:
		srcname = tv.fullname()
		if tv.exists() { srcmtime = tv._mtime }
	default:
		srcname = __string(ctx, source)
		if file := project.file(ctx, srcname); file != nil {
			source, srcname = file, file.fullname()
			if file.exists() { srcmtime = file._mtime }
		}
	}

	// PURE O(1) INTEGER MATH!
	if filemtime != 0 && filemtime > srcmtime {
		if ctx.update {
			if ctx.verbose { prompt(ctx, "update %v …", target) }
		} else if ctx.override {
			if ctx.verbose { prompt(ctx, "override %v …", target) }
		} else {
			if ctx.verbose { prompt(ctx, "copy %v …… already existed!\n", target) }
			if !ctx.silent { erro(ctx, "file already existed (%s)", target) }
			return
		}
	} else if ctx.verbose {
		if ctx.update {
			prompt(ctx, "Checking %v …", target)
		} else {
			prompt(ctx, "Copy %v …", target)
		}
	}

	if ctx.quick {
		var file = _stat(ctx, filename, stat_nonexist{true})
		// UPGRADED: Replaced file.info != nil with file.exists()
		if file == nil || file.exists() {
			if ctx.verbose { prompt(ctx, "… Good\n") }
			return
		}
	}

	var program = _program(ctx)
	var copts = &copyopts{
		program, ctx.path||ctx.recursive,
		ctx.update, ctx.mode, ctx.head, ctx.foot,
		0, 0, 0,
	}
	var file *file
	
	// UPGRADED: VFS-native existence check
	if file = _stat(ctx, srcname, stat_nonexist{true}); file == nil || !file.exists() {
		erro(ctx, "'%s' source file not found", srcname)
	} else if !file._isDir { // UPGRADED: Boolean primitive check
		
		// COLD PATH: We are physically copying a file. 
		// We do a localized os.Stat to get the exact OS file mode (permissions) 
		// without bloating our VFS filebase cache!
		info, err := os.Stat(srcname)
		if err != nil {
			erro(ctx, "stat source failed: %v", err)
			return
		}
		
		if ctx.mode == 0 { ctx.mode = info.Mode() }
		if err := copyFile(ctx, info, srcname, filename, copts); err != nil {
			erro(ctx, "%v", err)
		}
	} else if ctx.recursive {
		if err := copyDir(ctx, srcname, filename, copts); err != nil {
			erro(ctx, "%v", err)
		}
	} else {
		erro(ctx, "`%v` is a directory (use -r to solve it)", source)
	}

	if ctx.verbose {
		if copts.copied == 0 {
			prompt(ctx, "… Good (%d files)\n", copts.files)
		} else if copts.copied == 1 {
			prompt(ctx, "… Copied %d bytes\n", copts.bytes)
		} else {
			prompt(ctx, "… Copied %d bytes (%d/%d)\n", copts.bytes, copts.copied, copts.files)
		}
	}
	return
}

type modifier_writefile struct { modifier_ }
func (ctx *modifier_writefile) x(args ...Value) (result any) {
    args = xmerge(ctx, args...) //, plain

    var (
        target = auto_get(ctx, symAt)
        filename, str string
        f *os.File
    )
    if target == nil {
        erro(ctx, "target is undefined")
    }

    defer func() {
        if filename != "" { os.Remove(filename); f = nil }
        if f == nil {
            erro(ctx, "file %s not generated", target)
        }
    } ()

    filename, _ = as_fullname_string(ctx, target)

    if h := auto_get(ctx, symDash); h == nil {
        erro(ctx, "buffer value is nil")
    } else {
        str = __string(ctx, h)
    }

    var err error
    if f, err = os.Create(filename); err != nil {
        erro(ctx, "%v", err)
    } else if _, err = f.WriteString(str); err != nil {
        f.Close()
        erro(ctx, "%v", err)
    } else {
        result = _stat(ctx, filename)
        f.Close()
    }
    return
}

type modifier_readfile struct { modifier_
    head Value "h,head"
    foot Value "f,foot"
}
func (ctx *modifier_readfile) x(args ...Value) (result any) {
    var (
        filename string
        file *file
        target Value
    )
    if n := len(args); n > 1 {
        erro(ctx, "too many files: %v", args)
    } else if n == 1 {
        target = args[0]
    } else {
        target = auto_get(ctx, symAt)
    }

    if isTrivial(target) {
        erro(ctx, "%v: target is trivial (%v)", target, args)
    } else if file, filename, _ = as_fullname_file(ctx, target); file == nil {
        if val := auto_get(ctx, symRangle); val != nil {
            panic(traveTargetNotDefinedFile)
        } else if true {
            erro(ctx, _f("%v: not a file: %s", target, ts(target,ctx)))
        }
        return
    } else if filename == "" {
        erro(ctx, "%v: empty fullname", target)
    }

	if bytes, err := ioutil.ReadFile(filename); err == nil {
		var b strings.Builder

		// Pre-grow the buffer to exactly the size we need to prevent re-allocations
		headStr, footStr := "", ""
		if ctx.head != nil { headStr = __string(ctx, ctx.head) }
		if ctx.foot != nil { footStr = __string(ctx, ctx.foot) }

        b.Grow(len(headStr) + len(bytes) + len(footStr))
        b.WriteString(headStr)
        b.Write(bytes) // Writes the byte slice directly, no string cast needed!
        b.WriteString(footStr)

        auto_set(ctx.Context, defVoid, symDash, _raw(_pos(ctx), b.String()))
        auto_set(ctx.Context, defVoid, intern("-file"), file)
    } else {
        erro(ctx, "%v: %v ; stems=%v", target, err, _stems(ctx))
    }
    return
}

var crc64Table = crc64.MakeTable(crc64.ECMA)//crc64.ISO

func crc64CheckFileModeContent(ctx Context, filename string, content []byte, perm os.FileMode) (same bool, err error) {
    var f *os.File
    if f, err = os.Open(filename); err == nil && f != nil {
        defer f.Close()

        var s os.FileInfo
        if s, err = f.Stat(); err != nil { return false, err }

        // Fast Path: If sizes differ, they cannot be the same. Skip hashing!
        if s.Size() != int64(len(content)) {
            return false, nil
        }

        if perm != 0 && s.Mode().Perm() != perm {
            if err = f.Chmod(perm); err != nil { return }
        }

        w1 := crc64.New(crc64Table)
        w2 := crc64.New(crc64Table)
        if _, err = io.Copy(w1, f); err != nil { return }
        if _, err = w2.Write(content); err != nil { return }
        if w1.Sum64() == w2.Sum64() { same = true }
    }
    return
}

func crc64CompareFileChecksum(ctx Context, filename1, filename2 string) (same bool, err error) {
    var s []byte
    if s, err = ioutil.ReadFile(filename1); err != nil {
        erro(ctx, "%v", err)
        return
    }
    return crc64CheckFileModeContent(ctx, filename2, s, 0)
}

type modifier_updatefile struct { modifier_
	verbFilename bool `verbfile,verb-filename`
	path   bool `p,path,makedir,make-dir,makepath,make-path`
	zero   bool `zero,empty,allow-zero,allow-empty`
	keep   bool `keep,keep-file`
	append bool `app,append,append-content`
	mode os.FileMode "mode"
}
func (ctx *modifier_updatefile) x(args ...Value) (result any) {
	assert(ctx.mode != 0, "zero file mode")

	var target Value
	var content string
	var filename string
	if len(args) > 0 { target = args[0] }

	if isTrivial(target) { target = auto_get(ctx, symAt) }
	if isTrivial(target) {
		erro(ctx, "update-file: no file target")
	} else if t := as_fullname(ctx, target); t.Value == nil {
		erro(ctx, "update-file: not a file: %v", ts(target))
	} else if filename = __string(ctx, t); filename == "" {
		erro(ctx, "update-file: empty fullname: %v", ts(target))
	}

	if checkpoints {
		defer func() {
			ctx.x_check(target, filename, content, args, result)
		} ()
	}

	if ctx.path { // Make path (mkdir -p)
		if p := filepath.Dir(filename); p != "." && p != string(filepath.Separator) {
			// Pure OS level stat. We are doing physical disk mutation, so we bypass 
			// the VFS cache here to avoid caching intermediate/incomplete directory states.
			if fi, _ := os.Stat(p); fi != nil && !fi.IsDir() {
				if e := os.Remove(p); e != nil {
					erro(ctx, "%v (%v)", e, ts(target))
				}
			}
			if e := os.MkdirAll(p, os.FileMode(0755)); e != nil {
				if proj := _project(ctx); proj != nil {
					info(ctx, "%v: %v %v", filename, proj, unmap_files(ctx, proj, _pathStr(ctx, filename), nil))
					info(ctx, "%v: %v %v", filename, proj, proj.file(ctx, filename))
					erro(ctx, "%v: %v (%v)", filename, e, ts(target))
				}
				return
			}
		}
	}

	// Check existed file content checksum
	var exeres *exec_result
	if val := auto_get(ctx, symDash); val == nil {
		// no buffer value
	} else if content = __string(ctx, val); false && strings.Contains(content, `"\"`) {
		prompt(ctx, "%v: %T\n", filename, val)
		panic(_failure(ctx, "%s", filename))
	} else {
		exeres, _ = val.(*exec_result)
	}

	if content != "" {
		// good to go
	} else if ctx.zero {
		if ctx.verbose || ctx.debug > 0 {
			debug(ctx, "empty content for '%v'", target, callstack{num:ctx.debug})
		}
	} else {
		if ctx.keep {
			// keep file
		} else if file := _stat(ctx, filename); file != nil && file.exists() && file._size == 0 {
			// UPGRADED: 0-allocation primitive bounds checking!
			// Invalidate the cache mathematically by zeroing the timestamp
			file._mtime = 0 
			if err := os.Remove(filename); err != nil {
				erro(ctx, "remove file failed: %v", err)
			}
		}
		if exeres != nil {
			if exeres.Stdout.log != nil {
				var pos Position
				pos.Filename = exeres.Stdout.log.filename
				pos.Line = exeres.Stdout.log.lines + 1
				debug(ctx, "empty stdout")
			}
			if exeres.Stderr.log != nil && exeres.Stdout.log != exeres.Stderr.log {
				var pos Position
				pos.Filename = exeres.Stderr.log.filename
				pos.Line = exeres.Stderr.log.lines + 1
				debug(ctx, "empty stderr")
			}
		}

		if v := auto_get(ctx, symDash); v == nil {
			prompt(ctx, "%s:1: empty content\n", filename)
		} else {
			prompt(ctx, "%s:1: empty content: %v\n", filename, v)
		}
		erro(ctx, "empty content for '%v'", target)
	}

	var (
		wrote int
		same bool
		err error
	)
	if ctx.verbose {
		defer func(st time.Time) {
			var f string
			if ctx.verbFilename {
				f = trimPromptString(filename)
			} else {
				f = trimPromptString(target.String())
			}

			var s string
			if err != nil { s = err.Error() } else if same {
				if true { return } else { s = "unchanged" }
			} else if ctx.debug > 0 {
				s = fmt.Sprintf("changed (%d bytes, %s)", wrote, filename)
			} else {
				s = fmt.Sprintf("changed (%d bytes)", wrote)
			}
			prompt(ctx, "update %v …… %s (in %v)\n", f, s, time.Since(st))
		} (time.Now())
	}

	if same, err = crc64CheckFileModeContent(ctx, filename, []byte(content), ctx.mode); err != nil {
		if _, ok := err.(*os.PathError); ok {
			err = nil // discard path error (e.g. no such file or directory)
		} else {
			erro(ctx, "crc64 checksum failed: %v", err)
		}
	} else if same {
		//removeCallerUpdated(ctx, target) // remove timestamp updated
		result = _stat(ctx, filename)
		return
	}

	// COLD PATH: Create or update the physical file with new content
	var f *os.File
	var m = os.O_RDWR | os.O_CREATE
	if ctx.append { m |= os.O_APPEND } else { m |= os.O_TRUNC }
	
	if f, err = os.OpenFile(filename, m, ctx.mode); err != nil {
		erro(ctx, "open file failed: %v", err)
	} else if f != nil {
		defer func() {
			if err = f.Close(); err != nil {
				os.Remove(filename)
				erro(ctx, "close file '%s' failed: %v", filename, err)
			}

			// Re-enter the Walled Garden: Sync the fresh disk state back into the VFS!
			if t := _stat(ctx, filename); t == nil {
				prompt(ctx, "%s: invalid file\n", filename)
				erro(ctx, "%v: invalid file '%s'", _project(ctx), filename)
			} else {
				// t.stamp() will natively pull the new ModTime and Size into the primitive struct fields
				var fs = t.stamp(stamp_file_ctx{ctx})
				if false && ctx.verbose { reportFileUpdates(ctx, fs) }
				result = t // resulting the updated file
			}
		} ()
		if wrote, err = f.WriteString(content); err != nil {
			erro(ctx, "write content failed: %v", err)
		}
	} else {
		erro(ctx, "%v not updated", target)
	}
	return
}

type modifier_wait struct { modifier_
    stdout   bool "o,stdout"
    stderr   bool "e,stderr"
    status   bool "s,status"
    trim     bool "t,trim" // trim heading and tailing spaces of the result
    execRes  bool "x,exec"
    noTarget bool `nt,no-target`
    asType string "a,as"
}
func (ctx *modifier_wait) x(args ...Value) (result any) {
    var (
        waitForExecResult = ctx.stdout || ctx.stderr || ctx.status || ctx.execRes
        stampCurrentTarget = !ctx.noTarget
        target Value = auto_get(ctx, symAt)
        execRes *exec_result
        err error
    )
    if ctx.verbose {
        defer func (st time.Time) {
            var s string; if err != nil { s = "fail" } else { s = "done" }
            prompt(ctx, "Wait %v …… %s, result=%v\n", target, s, execRes)
            if ctx.debug>0 { debug(ctx, "%v", execRes) }
        } (time.Now())
    }

    // Wait for prerequisites and/or execution
    _, _, execRes = wait(ctx, waitopts{ctx.verbose, waitForExecResult, stampCurrentTarget})
    if execRes == nil { return }

    var (
        pos = _pos(ctx)
        a []Value
        s string
        v Value
    )
    if ctx.stdout {
        // TODO: warn(ctx, "deprecated (wait -stdout), use (shell -stdout) instead; %v", execRes).debug()
        if b := execRes.Stdout.Buf; b != nil { s = b.String() }
        if ctx.trim { s = strings.TrimSpace(s) }
        switch ctx.asType {
        case "answer": v = _answer (pos,(s == "yes"))
        case "bool":   v = _boolean(pos,(s == "true"))
        default:       v = _strlit(pos,s)
        }
        a = append(a, v)
    }
    if ctx.stderr {
        // TODO: warn(ctx, "deprecated (wait -stderr), use (shell -stderr) instead; %v", execRes).debug()
        if b := execRes.Stderr.Buf; b != nil { s = b.String() }
        if ctx.trim { s = strings.TrimSpace(s) }
        switch ctx.asType {
        case "answer": v = _answer (pos,(s == "yes"))
        case "bool":   v = _boolean(pos,(s == "true"))
        default:       v = _strlit(pos,s)
        }
        a = append(a, v)
    }
    if ctx.status {
        // TODO: warn(ctx, "deprecated (wait -status), use (shell -status) instead; %v", execRes).debug()
        a = append(a, _decimal(pos,int64(execRes.Status)))
    }

    if len(a) > 0 { result = ease(ctx, a) }
    return
}

func reportFileUpdates(ctx Context, fs []*file) {
	var start = _execution(ctx).start
	var startNano = start.UnixNano() // O(1) primitive baseline

	for _, f := range fs {
		var d = time.Since(start)
		
		// Blazing fast primitive math
		if f._mtime > startNano {
			prompt(ctx, "Updated %v (%v)\n", f.name.String(), d)
		} else {
			// Only allocate time.Time for the slow-path logging
			var mod = time.Unix(0, f._mtime)
			prompt(ctx, "File %v not changed (%v, ModTime=%v)\n", f, d, mod)
			debug(ctx,
				_f("incorrect timestamp: %v (JobTime=%v, ModTime=%v)", f, start, mod),
				_f("the target path name is: %v", f.fullname()),
				_f("try 'touch' the target %v if the path name and command are correct", f),
				_f("you may ignore the warnings if all correct"))
		}
	}
}

type modifier_stamp struct { modifier_
    prompt bool "prompt"
    next   bool "nxt,next"  // traveNext if failed to stamp
    error  bool "err,error" // traveErro if failed to stamp
}
func (ctx *modifier_stamp) x(args ...Value) (result any) {
    var target = auto_target_value(ctx)

    if isNull(target) {
        prompt(ctx, "%v\n", _project(ctx))
        erro(ctx, "stamp(%v) failed", target)
    }

    var v = stampFile(stamp_file_ctx{ctx}, target)
    if v != nil { return /* Done! */ }

    prompt(ctx, "%v: %v\n", target, _project(ctx))
    if ctx.next {
        panic(traverse_state{_position(ctx),traverse_next})
    } else if ctx.error {
        erro(ctx, "stamp(%v) error")
    } else {
        if f, y := target.(*file); y {
            erro(ctx, "failed stamp(%v): %v %v", target, f.fullname(), f._mtime)
        } else {
            erro(ctx, "failed stamp(%v) (%T)", target, target)
        }
    }
    return
}

type modifier_assert struct { modifier_
    msg string `msg,message`
}
func (ctx *modifier_assert) v(args ...Value) (_ any) { ctx.z(args...) ; return }
func (ctx *modifier_assert) x(args ...Value) (_ any) { ctx.z(args...) ; return }
func (ctx *modifier_assert) z(args ...Value) (_ any) {
    var u = _universe(ctx)
    for _, a := range args {
        if a == nil {
            erro(ctx, "assert: nil")
        }

        if _, y := a.(*punct); y { continue }

        v := expand(_final(ctx),a)
        b := v != nil && __true(ctx, v)
        f := u.hooks.assert

        if (f != nil && f(ctx, v, b)) || b {
            continue
        } else if ctx.msg == "" {
            var s string
            if v != nil { s = __string(ctx, v) }
            erro(pc(ctx,a), "assert: %v → %v → '%s'", a, v, s)
        } else {
            erro(pc(ctx,a), "assert: %v → %v: %s", a, v, ctx.msg)
        }
    }
    return
}

type modifier_cond struct { modifier_ }
func (ctx *modifier_cond) x(args ...Value) (result any) {
    // TODO: make it lisp-like (cond), e.g.:
    //     (cond
    //       ((condition) ...)
    //       (true{} ...))
    for _, a := range args {
        if a == nil { debug(ctx, "nil arg") }
        if a == nil || !__true(ctx.Context, a) {
            panic(traverse_state{_position(ctx),traverse_done})
        }
    }
    return _boolean(_pos(ctx), true)
}

type modifier_case struct { modifier_ }
func (ctx *modifier_case) x(args ...Value) (result any) {
    for _, a := range args {
        if __true(ctx.Context, a) {
            panic(traverse_state{_position(ctx),traverse_case})
        }
    }

    if ctx.verbose { prompt(ctx, "%v", auto_get(ctx, symAt)) }
    panic(traverse_state{_position(ctx),traverse_next})
    return
}

type modifier_predictDirty struct { modifier_ }
func (ctx *modifier_predictDirty) x(args ...Value) (result any) {
    if res := _execution(ctx).dirty(ctx, args...); res {
        return makePrediction(_pos(ctx), res, "")
    } else {
        panic(traverse_state{_pos(ctx), traverse_done})
    }
}

type modifier_fork struct { modifier_
    wd string `workdir,work-dir`
}
func (ctx *modifier_fork) _x(args ...Value) (result Value) {
    var (
        attr syscall.ProcAttr
        argv []string
        prog = _program(ctx)
    )
    for _, a := range args { argv = append(argv, __string(ctx, a)) }

    if ctx.wd != "" {
        attr.Dir = ctx.wd
    } else if attr.Dir = prog.workdir(ctx); attr.Dir == "" {
        erro(ctx, "empty workdir")
    }

    attr.Env, _ = _execution(ctx).env(ctx)
    attr.Files = []uintptr{ // FIXME: see Cmd.Start() for files pipes
        os.Stdin .Fd(),
        os.Stdout.Fd(),
        os.Stderr.Fd(),
    }

    if exe, err := os.Executable(); err != nil {
        erro(ctx, "fork: %v: %v", os.Args[0], err)
    } else if pid, err := syscall.ForkExec(exe, argv, &attr); err != nil {
        erro(ctx, "fork: %v: %v", exe, err)
    } else if pid == 0 {
        erro(ctx, "fork: pid is zero")
    } else {
        // TODO: status code, etc.
    }
    return
}
func (ctx *modifier_fork) x(args ...Value) (result any) {
    var (
        prog = _program(ctx)
        argv []string
        wd string
    )
    for _, a := range args { argv = append(argv, __string(ctx, a)) }

    if ctx.wd != "" {
        wd = ctx.wd
    } else if wd = prog.workdir(ctx); wd == "" {
        erro(ctx, "empty workdir")
    }

    var exe, err = os.Executable()
    if err != nil {
        erro(ctx, "fork: %v: %v", os.Args[0], err)
    }

    var cmd = exec.Command(exe, argv...)
    cmd.Stdout, cmd.Stderr = stdout, stderr
    cmd.Env, _ = _execution(ctx).env(ctx)

    if err = cmd.Run(); err != nil {
        erro(ctx, "fork: %v: %v", exe, err)
    } else {
        // TODO: status code, etc.
    }
    return
}

type modifier_gitmodified struct { modifier_ }
func (ctx *modifier_gitmodified) x(args ...Value) (result any) {
    var out = new(bytes.Buffer)
    var git = exec.Command("git", "status")
    git.Stdout, git.Stderr = out, os.Stderr
    if err := git.Run(); err != nil {
        erro(ctx, "git failed: %v", err)
    }

    // TODO: check also for `Changes not staged for commit:`

    var rx = regexp.MustCompile(`\n\tmodified:[\ctx ]*(.+?)\n`)
    var sm = rx.FindAllSubmatch(out.Bytes(), -1)
    if len(sm) > 0 {
        var pos = _pos(ctx)
        var pred = makePrediction(pos, false, "")
        if result = pred; len(args) == 0 {
            pred.bool, pred.s = true, "modified"
            return
        }
        for _, a := range args {
            var s = __string(ctx, a)
            for _, v := range sm {
                if false { prompt(ctx, "%s: %s\n%v\n", pos, s, v[1]) }
                if s == string(v[1]) {
                    pred.bool, pred.s = true, "modified: "+s
                    return
                }
            }
        }
    }
    return
}

type modifier_gitahead struct { modifier_ }
func (ctx *modifier_gitahead) x(args ...Value) (result any) {
    var out = new(bytes.Buffer)
    var git = exec.Command("git", "status")
    git.Stdout, git.Stderr = out, os.Stderr
    if err := git.Run(); err != nil {
        erro(ctx, "git: %v", err)
    }

    // TODO: check also for `Changes not staged for commit:`

    var rx = regexp.MustCompile(`\nYour branch is ahead of '(.+?)' by`)
    var sm = rx.FindAllSubmatch(out.Bytes(), 1)
    if len(sm) > 0 {
        result = makePrediction(_pos(ctx), true, "Work branch has new commits to push")
    }
    return
}

var (
    onceMutex sync.Mutex
    onceCache0 map[entry]map[Value]int
    onceCache1 map[*program]map[Value]int
    onceSHA256Mutex sync.Mutex
    onceSHA256Cache = make(map[hashbytes]int,64)
)

func onceCacheTest0(ctx Context, target Value) (n int) {
    var rec map[Value]int
    var ent = _entry(ctx)
    if x, y := ent.(*stemmed_rule); y { ent = x.rule }

    onceMutex.Lock(); defer onceMutex.Unlock()
    if onceCache0 == nil { onceCache0 = make(map[entry]map[Value]int, 64) }
    if rec, _ = onceCache0[ent]; rec == nil {
        rec = make(map[Value]int)
        onceCache0[ent] = rec
    }

    rec[target] += 1
    n = rec[target]
    return
}

func onceCacheTest1(ctx Context, target Value) (n int) {
    var (
        prog = _program(ctx)
        rec map[Value]int
    )

    onceMutex.Lock(); defer onceMutex.Unlock()
    if onceCache1 == nil { onceCache1 = make(map[*program]map[Value]int,64) }
    if rec, _ = onceCache1[prog]; rec == nil { rec = make(map[Value]int)
        onceCache1[prog] = rec
    }

    rec[target] += 1
    n = rec[target]
    return
}

func onceCacheTest2(ctx Context, target Value) (n int) {
    var (
        program = _program(ctx)
        h = sha256.New()
        entry = _entry(ctx)
    )
    if stemmed, ok := entry.(*stemmed_rule); ok {
        entry = stemmed.rule
    }

    // NOTE: ensure 'entry', 'program' and 'target' are unique.
    if true {
        fmt.Fprintf(h, "%p", program)
    } else if false {
        // // FIXME: not unique combination
        // fmt.Fprintf(h, "%p", entry)
        fmt.Fprintf(h, "%T%p", entry, entry)
    } else {
        // // FIXME: not unique combination
        // fmt.Fprintf(h, "%p%p", entry, program)
        fmt.Fprintf(h, "%T%p%p", entry, entry, program)
    }

    for _, t := range merge(target) {
        if f, ok := to_file(t); ok {
            fmt.Fprintf(h, "%s", f.fullname())
        } else {
            fmt.Fprintf(h, "%s", __string(ctx, t))
        }
    }

    var sum hashbytes
    copy(sum[:], h.Sum(nil))
    return onceSHA256Test(ctx, sum)
}

func onceSHA256Test(ctx Context, sum hashbytes) (n int) {
    onceSHA256Mutex.Lock()
    n = onceSHA256Cache[sum]+1
    onceSHA256Cache[sum] = n
    onceSHA256Mutex.Unlock()
    return
}

func onceSHA256(ctx *modifier_once, target Value, args ...Value) (n int) {
    var (
        program = _program(ctx)
        entry = _entry(ctx)
        h = sha256.New()
    )
    if stemmed, ok := entry.(*stemmed_rule); ok {
        entry = stemmed.rule
    }

    if true {
        // // NOTE: entry and program are unique, since (once) is for runtime, we use their addresses.
        // fmt.Fprintf(h, "%p%p", entry, program)
        fmt.Fprintf(h, "%T%p%p", entry, entry, program)
    } else {
        fmt.Fprintf(h, "%v%v", _position(ctx), program.pos)
    }

    for _, a := range args {
        s, _ := as_fullname_string(ctx, a)
        fmt.Fprintf(h, "%s", s)
    }

    var sum hashbytes
    copy(sum[:], h.Sum(nil))
    return onceSHA256Test(ctx, sum)
}

type modifier_once struct { modifier_
    checksum bool `cs,checksum,sha,sha256,sum,hash`
    forval Value `for` // TODO: (once -for=$@)
}
func (ctx *modifier_once) x(args ...Value) (result any) {
    // TODO: (once)           --> once for the Rule, aka entry.doneOnce = true
    // TODO: (once -for=$@)   --> once for $@, aka entry.onces[$(expand $@)] = true
    var target Value = auto_get(ctx, symAt)

    const onceAlgo = 2 // avaialbe: 0, 1, 2

    if isTrivial(target) {
        erro(ctx, "once: no target $@, %v", args)
    } else if ctx.checksum {
        onceSHA256(ctx, target, append([]Value{target}, args...)...)
    } else if onceAlgo == 2 {
        onceCacheTest2(ctx, target)
    } else if onceAlgo == 1 {
        onceCacheTest1(ctx, target)
    } else {
        onceCacheTest0(ctx, target)
    }
    return
}

const benchmark = true
const (
    builtinCallable uint = 0
    builtinCommand       = 1<<(iota-1)
    builtinForce
)

type builtincalls struct{}
func _builtincalls(ctx Context) (_ string) {
    if s, y := do(ctx, builtincalls{}).(string); y {
        return strings.Replace(s, "(%s)", "", -1)
    }
    return
}

type builtinbase struct{ *evocation ; general_opts }
func (c *builtinbase) inner() Context { return c.evocation }
func (c *builtinbase) cast(t reflect.Type) Context {
    if reflect.TypeOf((*builtinbase)(nil)) == t { return c }
    if reflect.TypeOf(c) == t { return c }
    return c.evocation.cast(t)
}
func (c *builtinbase) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case builtincalls:
        var s = c.x.String()+"(%s)"
        if x, y := c.evocation.do(ctx, op).(string); y && x != "" {
            s = fmt.Sprintf(x, s)
        }
        return s
    }
    return c.evocation.do(ctx, op)
}
func (c *builtinbase) ts(string) string {
    if true {
        var s = c.defs.String()
        if s != "" { s += " " }
        return "{="+c.x.String()+" "+s+ts(c.Context)+"}"
    } else if false {
        return "{="+c.x.String()+" "+ts(c.evocation)+"}"
    } else {
        return "{="+c.x.String()+" "+ts(c.Context)+"}"
    }
}
func (c *builtinbase) _a(force bool) (skip bool) {
    c.evocation.a = expands(c, c.evocation.a...)
    return
}

type builtin_x interface{ x() any }
var builtin_x_t = reflect.TypeOf((*builtin_x)(nil)).Elem()
var builtins = map[Symbol]reflect.Type {
	symTypeof:    reflect.TypeOf((*__typeof)(nil)).Elem(),
	symOrigin:    reflect.TypeOf((*__origin)(nil)).Elem(),
	symDefined:   reflect.TypeOf((*__defined)(nil)).Elem(),

	symPosition:  reflect.TypeOf((*__position)(nil)).Elem(),
	symDate:      reflect.TypeOf((*__date)(nil)).Elem(),

	symDebug:     reflect.TypeOf((*__debug)(nil)).Elem(),
	symError:     reflect.TypeOf((*__error)(nil)).Elem(),
	symWarning:   reflect.TypeOf((*__warning)(nil)).Elem(),
	symAssert:    reflect.TypeOf((*__assert)(nil)).Elem(),
	symSure:      reflect.TypeOf((*__sure)(nil)).Elem(),
	symTrace:     reflect.TypeOf((*__trace)(nil)).Elem(),

	symDefor:     reflect.TypeOf((*__defor)(nil)).Elem(), // $(defor $(x),$(y),$(z))  <=>  $(ifdef x,$(y),$(z))
    symOr:        reflect.TypeOf((*__or)(nil)).Elem(),
    symAnd:       reflect.TypeOf((*__and)(nil)).Elem(),
    symNot:       reflect.TypeOf((*__not)(nil)).Elem(),
    symXor:       reflect.TypeOf((*__xor)(nil)).Elem(),

    symEqual:     reflect.TypeOf((*__equal)(nil)).Elem(),
    symNe:        reflect.TypeOf((*__unequal)(nil)).Elem(),
    symNotEqual: reflect.TypeOf((*__unequal)(nil)).Elem(),
    symMatch:     reflect.TypeOf((*__match)(nil)).Elem(),

    symGreater:   reflect.TypeOf((*__greater)(nil)).Elem(),
    symLess:      reflect.TypeOf((*__less)(nil)).Elem(),

    symCase:      reflect.TypeOf((*__case)(nil)).Elem(),
    symIf:        reflect.TypeOf((*__if)(nil)).Elem(),
    symIfeq:      reflect.TypeOf((*__ifeq)(nil)).Elem(),
    symIfne:      reflect.TypeOf((*__ifne)(nil)).Elem(),
    symIfarg:     reflect.TypeOf((*__ifarg)(nil)).Elem(),
    symIfdef:     reflect.TypeOf((*__ifdef)(nil)).Elem(),

    symFor:       reflect.TypeOf((*__for)(nil)).Elem(),
    symForeach:   reflect.TypeOf((*__foreach)(nil)).Elem(),
    symCount:     reflect.TypeOf((*__count)(nil)).Elem(),

    symAuto:      reflect.TypeOf((*__auto)(nil)).Elem(),
    // symVar:       reflect.TypeOf((*__var)(nil)).Elem(),

    symCall:      reflect.TypeOf((*__call)(nil)).Elem(),
    symDefs:      reflect.TypeOf((*__defs)(nil)).Elem(),

    symValue:     reflect.TypeOf((*__value)(nil)).Elem(),
    symList:      reflect.TypeOf((*__list)(nil)).Elem(),
    symEnv:       reflect.TypeOf((*__env)(nil)).Elem(),

    symShell:     reflect.TypeOf((*__shell)(nil)).Elem(),
    symWhich:     reflect.TypeOf((*__which)(nil)).Elem(),

    symPlus:      reflect.TypeOf((*__plus)(nil)).Elem(),
    symMinus:     reflect.TypeOf((*__minus)(nil)).Elem(),
    symMultiply:  reflect.TypeOf((*__multiply)(nil)).Elem(),
    symMul:       reflect.TypeOf((*__multiply)(nil)).Elem(),
    symDivide:    reflect.TypeOf((*__divide)(nil)).Elem(),
    symDiv:       reflect.TypeOf((*__divide)(nil)).Elem(),

    symJoin:       reflect.TypeOf((*__join)(nil)).Elem(),
    symConjunct:   reflect.TypeOf((*__conjunct)(nil)).Elem(), // concat
    symQuote:      reflect.TypeOf((*__quote)(nil)).Elem(),
    symUnique:     reflect.TypeOf((*__unique)(nil)).Elem(),

    symSplit:          reflect.TypeOf((*__split)(nil)).Elem(),
    symSplitQuote:     reflect.TypeOf((*__splitquote)(nil)).Elem(),
    symSplitQuoteJoin: reflect.TypeOf((*__splitquotejoin)(nil)).Elem(),
    symSplitJoinQuote: reflect.TypeOf((*__splitjoinquote)(nil)).Elem(),

    symElement:      reflect.TypeOf((*__element)(nil)).Elem(),
    symField:        reflect.TypeOf((*__field)(nil)).Elem(),
    symFields:       reflect.TypeOf((*__fields)(nil)).Elem(),

    // symUsee:         reflect.TypeOf((*__usee)(nil)).Elem(),
    symUses:         reflect.TypeOf((*__uses)(nil)).Elem(),

    symBare:         reflect.TypeOf((*__bare)(nil)).Elem(),
    symPath:         reflect.TypeOf((*__path)(nil)).Elem(),
    symWord:         reflect.TypeOf((*__word)(nil)).Elem(),
    symFinalize:     reflect.TypeOf((*__finalize)(nil)).Elem(),
    symResolve:      reflect.TypeOf((*__resolve)(nil)).Elem(),
    symStrip:        reflect.TypeOf((*__trim)(nil)).Elem(),
    symTrim:         reflect.TypeOf((*__trim)(nil)).Elem(),
    symTrimLeft:    reflect.TypeOf((*__trimleft)(nil)).Elem(),
    symTrimRight:   reflect.TypeOf((*__trimright)(nil)).Elem(),
    symTrimPrefix:  reflect.TypeOf((*__trimprefix)(nil)).Elem(),
    symTrimSuffix:  reflect.TypeOf((*__trimsuffix)(nil)).Elem(),
    symTrimExt:     reflect.TypeOf((*__trimext)(nil)).Elem(),

    symGitdir:       reflect.TypeOf((*__gitdir)(nil)).Elem(),

    symAddprefix:    reflect.TypeOf((*__addprefix)(nil)).Elem(),
    symAddsuffix:    reflect.TypeOf((*__addsuffix)(nil)).Elem(),

    symTitle:        reflect.TypeOf((*__title)(nil)).Elem(),
    symIndent:       reflect.TypeOf((*__indent)(nil)).Elem(),
    symSubstring:    reflect.TypeOf((*__substring)(nil)).Elem(),
    symUppercase:    reflect.TypeOf((*__uppercase)(nil)).Elem(),
    symLowercase:    reflect.TypeOf((*__lowercase)(nil)).Elem(),

    // https://www.gnu.org/software/make/manual/html_node/Text-Functions.html
    symSubst:        reflect.TypeOf((*__subst)(nil)).Elem(), // substitute
    symSubstitute:   reflect.TypeOf((*__subst)(nil)).Elem(),
    symPatsubst:     reflect.TypeOf((*__patsubst)(nil)).Elem(),

    symContains:     reflect.TypeOf((*__contains)(nil)).Elem(),
    symFilter:       reflect.TypeOf((*__filter)(nil)).Elem(),
    symFilterOut:   reflect.TypeOf((*__filterout)(nil)).Elem(),

    symDecodeBase64: reflect.TypeOf((*__decodebase64)(nil)).Elem(),
    symEncodeBase64: reflect.TypeOf((*__encodebase64)(nil)).Elem(),
    /* TODO:
    `encode-base32`
    `decode-base32`
    `encode-json`
    `decode-json`
    `encode-xml`
    `decode-xml`
    `encode-hex`
    `decode-hex`
    `encode-csv`
    `decode-csv` */

    symExt:        reflect.TypeOf((*__ext)(nil)).Elem(),

    symBase:      reflect.TypeOf((*__base1)(nil)).Elem(),
    symBase2:      reflect.TypeOf((*__base2)(nil)).Elem(),
    symBase3:      reflect.TypeOf((*__base3)(nil)).Elem(),
    symBase4:      reflect.TypeOf((*__base4)(nil)).Elem(),
    symBase5:      reflect.TypeOf((*__base5)(nil)).Elem(),
    symBase6:      reflect.TypeOf((*__base6)(nil)).Elem(),
    symBase7:      reflect.TypeOf((*__base7)(nil)).Elem(),
    symBase8:      reflect.TypeOf((*__base8)(nil)).Elem(),
    symBase9:      reflect.TypeOf((*__base9)(nil)).Elem(),
    symBases:      reflect.TypeOf((*__bases)(nil)).Elem(),

    symChopdir:    reflect.TypeOf((*__chopdir)(nil)).Elem(),

    symDir:        reflect.TypeOf((*__dir)(nil)).Elem(),
    symDir2:       reflect.TypeOf((*__dir2)(nil)).Elem(),
    symDir3:       reflect.TypeOf((*__dir3)(nil)).Elem(),
    symDir4:       reflect.TypeOf((*__dir4)(nil)).Elem(),
    symDir5:       reflect.TypeOf((*__dir5)(nil)).Elem(),
    symDir6:       reflect.TypeOf((*__dir6)(nil)).Elem(),
    symDir7:       reflect.TypeOf((*__dir7)(nil)).Elem(),
    symDir8:       reflect.TypeOf((*__dir8)(nil)).Elem(),
    symDir9:       reflect.TypeOf((*__dir9)(nil)).Elem(),
    symDirs:       reflect.TypeOf((*__dirs)(nil)).Elem(),

    symUndir:      reflect.TypeOf((*__undir1)(nil)).Elem(),
    symUndir2:     reflect.TypeOf((*__undir2)(nil)).Elem(),
    symUndir3:     reflect.TypeOf((*__undir3)(nil)).Elem(),
    symUndir4:     reflect.TypeOf((*__undir4)(nil)).Elem(),
    symUndir5:     reflect.TypeOf((*__undir5)(nil)).Elem(),
    symUndir6:     reflect.TypeOf((*__undir6)(nil)).Elem(),
    symUndir7:     reflect.TypeOf((*__undir7)(nil)).Elem(),
    symUndir8:     reflect.TypeOf((*__undir8)(nil)).Elem(),
    symUndir9:     reflect.TypeOf((*__undir9)(nil)).Elem(),
    symUndirs:     reflect.TypeOf((*__undirs)(nil)).Elem(),

    symReldir:      reflect.TypeOf((*__reldir)(nil)).Elem(),
    symRelativeDir: reflect.TypeOf((*__reldir)(nil)).Elem(),

    symFile:         reflect.TypeOf((*__file)(nil)).Elem(),
    symStat:         reflect.TypeOf((*__stat)(nil)).Elem(),// stat (deprecates file-exists)
    symGlob:         reflect.TypeOf((*__glob)(nil)).Elem(),
    symWildcard:     reflect.TypeOf((*__wildcard)(nil)).Elem(),

    symReadDir:     reflect.TypeOf((*__readdir)(nil)).Elem(),  // io/ioutil/ioutil.go
    symReadFile:    reflect.TypeOf((*__readfile)(nil)).Elem(), // io/ioutil/ioutil.go

    symGrep:         reflect.TypeOf((*__grep)(nil)).Elem(),

    // commands ------------------------------------------------------------------
    symPrint:        reflect.TypeOf((*__print)(nil)).Elem(),
    symPrintf:       reflect.TypeOf((*__printf)(nil)).Elem(),

    symPlain:        reflect.TypeOf((*__plain)(nil)).Elem(),

    symAppend:       reflect.TypeOf((*__append)(nil)).Elem(),
    // symUnshift:      reflect.TypeOf((*__unshift)(nil)).Elem(),
    // symPop:          reflect.TypeOf((*__pop)(nil)).Elem(),

    symWriteFile:   reflect.TypeOf((*__writefile)(nil)).Elem(), // io/ioutil/ioutil.go
    symTouchFile:   reflect.TypeOf((*__touchfile)(nil)).Elem(),  // io/ioutil/ioutil.go

    symMkdir:        reflect.TypeOf((*__mkdir)(nil)).Elem(),     // os/file.go
    symChdir:        reflect.TypeOf((*__chdir)(nil)).Elem(),     // os/file.go
    symRename:       reflect.TypeOf((*__rename)(nil)).Elem(),    // os/file.go
    symRemove:       reflect.TypeOf((*__remove)(nil)).Elem(),    // os/file_*.go
    symLink:         reflect.TypeOf((*__link)(nil)).Elem(),      // os/file_*.go
    symSymlink:      reflect.TypeOf((*__symlink)(nil)).Elem(),   // os/file_*.go
    symTruncate:     reflect.TypeOf((*__truncate)(nil)).Elem(),  // os/file_*.go

    symReturn:       reflect.TypeOf((*__return)(nil)).Elem(),
    symServeHttp:   reflect.TypeOf((*__servehttp)(nil)).Elem(),
}

func escapedString(ctx Context, v Value) (s string) {
    if p, ok := v.(*strlit); ok {
        s = strings.Replace(__string(ctx, p), "\\'", "'", -1)
    } else {
        s = __string(ctx, v)
    }
    return
}

func isNotSpace(r rune) bool {
    return !unicode.IsSpace(r)
}

func isRelPath(filename string) (res bool) {
    // This implementation replaces:
    //      strings.HasPrefix(filename, "."+pathSep)
    //      strings.HasPrefix(filename, ".."+pathSep)
    var ( s = "."+pathSep ; n = len(filename) )
    if n > 1 && filename[0] == s[0] {
        if filename[1] == s[0] && n > 2 {
            res = filename[2] == s[1]
        } else if filename[1] == s[1] {
            res = true
        }
    }
    return
}

func isAbsOrRel(filename string) bool {
    return filepath.IsAbs(filename) || isRelPath(filename)
}

func trimLeftSpaces(s string) string {
    return strings.TrimLeftFunc(s, unicode.IsSpace)
}

func trimRightSpaces(s string) string {
    return strings.TrimRightFunc(s, unicode.IsSpace)
}

func _set(ctx Context, val reflect.Value, v Value) {
    switch val.Kind() {
    case reflect.Bool:
        val.SetBool(__true(ctx, v))
    case reflect.Float32, reflect.Float64:
        val.SetFloat(__float(ctx, v))
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        val.SetInt(__int(ctx, v))
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        val.SetUint(uint64(__int(ctx, v)))
    case reflect.String:
        val.SetString(__string(ctx, v))
    case reflect.Slice:
        if p := reflect.New(val.Type().Elem()); p.Kind() == reflect.Ptr {
            var t = p.Elem()
            _set(ctx, t, v)
            val.Set(reflect.Append(val, t))
        }
    case reflect.Interface:
        switch val.Type().String() {
        case "smart.Value":
            val.Set(reflect.ValueOf(v))
        default:
            erro(pc(ctx,v), "option type unsupported: %v → %v, %v", ts(v), val.Kind(), val.Type())
        }
    case reflect.Ptr:
        switch val.Type().Elem().String() {
        case "smart.fullname":
            if t := as_fullname(fullfile_ctx{ctx}, v); t.Value != nil {
                val.Set(reflect.ValueOf(&t))
            } else {
                debug(pc(ctx,v), _f("%v → %v", v, as_file(ctx, v)),
					_f("not a file: %v → %s", ts(v), ts(expand(ctx, v))), trace{})
            }
        case "smart.file":
            if t := as_file(ctx, v); t != nil {
                val.Set(reflect.ValueOf(t))
            } else {
                erro(pc(ctx,v), _f("not a file: %v → %s", ts(v), ts(expand(ctx, v))))
            }
        case "regexp.Regexp":
            if rx, e := regexp.Compile(__string(ctx, v)); e == nil {
                val.Set(reflect.ValueOf(rx))
            } else {
                erro(pc(ctx,v), "wrong regexp: %v: %v", ts(v), e)
            }
        default:
            erro(pc(ctx,v), "option type unsupported: %v → %v, %v", ts(v), val.Elem().Kind(), val.Type().Elem())
        }
    default:
        switch val.Type().String() {
        case "fs.FileMode", "os.FileMode": // aka. reflect.Uint32
            var t = __int(ctx, v)
            if t == 0 { debug(pc(ctx,v), "zero file mode") }
            val.SetUint(uint64(t))
        case "regexp.Regexp": // aka. reflect.Ptr
            erro(pc(ctx,v), "TODO: regexp: %v → %v, %v", ts(v), val.Kind(), val.Type())
        default:
            erro(pc(ctx,v), "option type unsupported: %v → %v, %v", ts(v), val.Kind(), val.Type())
        }
    }
}

func _opt(ctx Context, tag reflect.StructTag, field reflect.Value, args ...Value) (rest []Value) {
    var val = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
    var opts []string // opt names

    if tag == "" { return args }
    if t := string(tag)[:]; t != "" {
        for {
            if i := strings.IndexAny(t, ";, "); 0 <= i {
                opts = append(opts, t[:i])
                t = t[i+1:]
            } else {
                opts = append(opts, t)
                break
            }
        }
    }

outer:
    for _, arg := range args {
        var y bool
        var f flag
        var value Value

        // 1. Extract the flag identity safely FIRST
        switch t := arg.(type) {
        case        flag: f, y = t, true; value = _boolean(t.Pos(), true)
        case       *pair: if f, y = t.key.(flag);   y { value = t.val }
        case *argumented: if f, y = t.Value.(flag); y { value = ease(ctx, t.args) }
        }

        // 2. CRITICAL FIX: Only skip if the flag name ITSELF is a pattern (e.g. -I%),
        // DO NOT skip if only the arguments contain patterns (like -cond($(depfiles?)))
        if y && !patterned(ctx, f) {
            for i := 0; i < len(opts); i += 1 {
                if _, match := f.opt(ctx, opts[i]); match {
                    _set(ctx, val, value)
                    continue outer
                }
            }
        }

        rest = append(rest, arg)
    }

    switch val.Type().String() {
    case "fs.FileMode", "os.FileMode":
        if val.Uint() == 0 { val.SetUint(0640) }
    }
    return
}

func _opts(ctx Context, opts reflect.Value, args []Value) (rest []Value) {
    if args == nil { return }

    if opts.Kind() != reflect.Ptr {
        erro(ctx, "opts must be ptr: %v", opts.Kind())
    } else if opts = opts.Elem(); opts.Kind() != reflect.Struct {
        erro(ctx, "opts is not ptr of struct: %v", opts.Kind())
    }

    rest = merge(args...)

    var builtin, general, modifier, clause, dots reflect.Value
    var ot = opts.Type()
    for i := 0; i < ot.NumField(); i += 1 {
        var ft, fv = ot.Field(i), opts.Field(i)
        if ft.Tag == "..." {
            dots = fv
        } else if t := fv.Type(); fv.Kind() != reflect.Struct {
            if ft.Anonymous && ft.Name == "Context" && t.String() == "smart.Context" {
				continue
            }

			// CRITICAL FIX: Intercept embedded *clause_opts (Pointer)
            if ft.Anonymous && ft.Name == "clause_opts" && fv.Kind() == reflect.Ptr {
                if !fv.IsNil() { clause = fv }
                continue
            }

            rest = _opt(ctx, ft.Tag, fv, rest...)
        } else if !ft.Anonymous {
            continue
        } else if ft.Name == "general_opts" {
            general = fv.Addr()
		} else if ft.Name == "clause_opts" {
			clause = fv.Addr() // Support if it was embedded by value instead of pointer
        } else if strings.HasPrefix(ft.Name, "__") {
            if builtin.IsValid() { debug(ctx, "embedded multiple builtins: %v", ft) }
            builtin = fv.Addr()
        } else if strings.HasPrefix(ft.Name, "modifier_") {
            if modifier.IsValid() { debug(ctx, "embedded multiple modifiers: %v", ft) }
            modifier = fv.Addr()
        }
    }
    if  general.IsValid() { rest = _opts(ctx,  general, rest) }
	if   clause.IsValid() { rest = _opts(ctx,   clause, rest) } // Inherit clause_opts!
    if  builtin.IsValid() { rest = _opts(ctx,  builtin, rest) }
    if modifier.IsValid() { rest = _opts(ctx, modifier, rest) }
    if dots.IsValid() && rest != nil {
        _set(ctx, dots, ease(ctx, rest))
        rest = nil
    }
    return
}
func parseOpts(ctx Context, store any, vals ...Value) []Value {
    return _opts(ctx, reflect.ValueOf(store), vals)
}

// see https://go.dev/doc/tutorial/generics
func _opts_[Opts any](ctx Context, args ...Value) (opts Opts, res []Value) {
    res = parseOpts(ctx, &opts, args...)
    return
}

func _parseHeadArgs(ctx Context, store any, args ...Value) (head, rest []Value) {
    if len(args) == 0 {
        // zero args
    } else if head = parseOpts(ctx, store, args[0]); len(head) > 0 {
        rest = args[1:] //xmerge(ctx, args[1:]...)
    } else if len(args) == 1 {
        // done
    } else if head = xmerge(ctx, args[1]); len(args) > 2 {
        rest = args[2:] //xmerge(ctx, args[2:]...)
    }
    return
}

func _parseHeadArgsMerge(ctx Context, store any, args ...Value) (res []Value) {
    var head, rest = _parseHeadArgs(ctx, store, args...)
    res = append(head, rest...)
    return
}

func _parseHeadArgsRequired(ctx Context, store any, args ...Value) (head, rest []Value) {
    head, rest = _parseHeadArgs(ctx, store, args...)
    if len(head) == 0 || len(rest) == 0 {
        erro(ctx, "insufficient number of arguments")
    }
    return
}

type __noop struct { builtinbase }
func (ctx *__noop) inner() Context { return &ctx.builtinbase }
func (ctx *__noop) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__noop) x() (_ any) { return }

type __typeof struct { builtinbase }
func (ctx *__typeof) inner() Context { return &ctx.builtinbase }
func (ctx *__typeof) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__typeof) x() (res any) {
    var vals []Value
    for _, a := range ctx.a {
        // Arguments are passed in a list:
        //   $(fun abc)             args: (abc)
        //   $(fun a,b,c)           args: (a),(b),(c)
        //   $(fun a b c,1 2 3)     args: (a b c),(1 2 3)
        vals = append(vals, _word(a.Pos(), intern(typeof(a))))
    }
    return vals
}

type __origin struct { builtinbase }
func (ctx *__origin) inner() Context { return &ctx.builtinbase }
func (ctx *__origin) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__origin) x() (res any) {
    var vals []Value
    var scope = _scope(ctx)
    for _, a := range ctx.a {
        if s := __symbol(ctx, a); s == symEmpty {
            vals = append(vals, _null(a.Pos()))
        } else if d := scope.finddef(s); d != nil {
            vals = append(vals, _word(a.Pos(), intern(d.o.String())))
        } else {
            vals = append(vals, _null(a.Pos()))
        }
    }
    return vals
}

type __defined struct { builtinbase }
func (ctx *__defined) inner() Context { return &ctx.builtinbase }
func (ctx *__defined) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__defined) x() (_ any) {
    var scope = _scope(ctx)
    for _, arg := range ctx.a {
		d := scope.finddef(__symbol(ctx, arg))
        if d != nil && !isTrivial(d.value) {
            return true
        }
    }
    return
}

type __position struct { builtinbase
    filename bool `filename`
    filenameQuoted bool `quote-filename,quoted-filename`
    line bool `ln,line`
    column bool `col,column`
    addLine int `add,add-line`
    addColumn int `add-column`
}
func (ctx *__position) inner() Context { return &ctx.builtinbase }
func (ctx *__position) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__position) x() (res any) {
	var vals []Value
	var pos = _position(ctx.Context) // Fat Position for string formatting
	var p = _pos(ctx.Context)        // Compact Pos for AST construction

	if ctx.filename {
		vals = append(vals, _raw(p, pos.Filename))
	} else if ctx.filenameQuoted {
		vals = append(vals, _raw(p, "\""+pos.Filename+"\""))
	}

	if ctx.line   { vals = append(vals, _decimal(p, int64(pos.Line + ctx.addLine))) }
	if ctx.column { vals = append(vals, _decimal(p, int64(pos.Column + ctx.addColumn))) }

	if len(vals) == 0 { return _raw(p, pos.String()) }
	if len(vals) == 1 { return vals[0] }
	return vals
}

type __date struct { builtinbase
    time bool `time,now`
}
func (ctx *__date) inner() Context { return &ctx.builtinbase }
func (ctx *__date) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__date) x() (res any) {
    if t := time.Now(); len(ctx.a) > 0 {
        var vals []Value
        for _, a := range ctx.a {
            var s string
            if s = __string(ctx, a); s == "" {
                s = t.String()
            } else if s = t.Format(s); s == "" {
                s = fmt.Sprintf("%v", t)
            }
            vals = append(vals, _strlit(a.Pos(), s))
        }
        return vals
    } else if ctx.time {
        res = makeTime(_pos(ctx), t)
    } else {
        res = makeDate(_pos(ctx), t)
    }
    return
}

type __debug struct { builtinbase
    s int `stack`
    n int `num`
}
func (ctx *__debug) inner() Context { return &ctx.builtinbase }
func (ctx *__debug) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__debug) x() (res any) {
    var s bytes.Buffer
	for i, a := range ctx.a { if i > 0 { fmt.Fprintf(&s, " ") }
		fmt.Fprintf(&s, "%v", __string(ctx, a))
	}
    if hook := _universe(ctx).hooks.debug; hook != nil {
        hook(ctx, s.String(), ctx.a)
    } else if true {
		prompt(ctx, "%s: %s\n", _position(ctx), s.String()); flush(ctx)
	} else {
        debug(ctx, "%s", s, callstack{num:ctx.n})
    }
    return
}

type __error struct { builtinbase }
func (ctx *__error) inner() Context { return &ctx.builtinbase }
func (ctx *__error) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__error) x() (res any) {
    var s bytes.Buffer
	for i, a := range ctx.a { if i > 0 { fmt.Fprintf(&s, " ") }
		fmt.Fprintf(&s, "%v", __string(ctx, a))
	}
    if hook := _universe(ctx).hooks.error; hook != nil {
        hook(ctx, s.String(), ctx.a)
	} else {
        erro(ctx, "%s", s)
    }
    return
}

type __warning struct { builtinbase }
func (ctx *__warning) inner() Context { return &ctx.builtinbase }
func (ctx *__warning) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__warning) x() (res any) {
    var s bytes.Buffer
    for i, a := range ctx.a {
        if i > 0 { fmt.Fprintf(&s, " ") }
        fmt.Fprintf(&s, "%s", __string(ctx, a))
    }
    debug(ctx, "%s", s.String())
    return
}

type __assert struct { builtinbase ; msg string `msg,message` }
func (ctx *__assert) inner() Context { return &ctx.builtinbase }
func (ctx *__assert) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__assert) x() (res any) {
    var d = ctx.debug ; if d < 1 { d = 1 }
    var s = ctx.stack ; if s < 1 { s = 1 }
    var t = diagError ; if ctx.warning { t = diagWarn }
    var hook = _universe(ctx).hooks.assert

    if ctx.a == nil && hook != nil && !hook(ctx, nil, false) {
        prompt(ctx, "assert: %v\n", ctx.a)
        debug(ctx, s, t, callstack{num:d})
    }

    for _, a := range expands(ctx, ctx.a...) {
        if a == nil {
            erro(ctx, "nil argument")
            continue
        }

        var c = pc(ctx, a)
        var y = __true(c, a)
        if hook != nil && hook(c, a, y) || y {
            continue
        }

        debug(c, s, t, "%v ⇒ '%s'", ts(a), __string(c, a), callstack{num:d})
    }

    if ctx.fail { panic(_failure(ctx)) }
    return
}

type __sure struct { builtinbase }
func (ctx *__sure) inner() Context { return &ctx.builtinbase }
func (ctx *__sure) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__sure) x() (res any) {
    for _, a := range ctx.a {
        if !__true(ctx, a) {
            erro(ctx, "assert: %v", ts(a))
        }
    }
    return ctx.a
}

type __trace struct { builtinbase }
func (ctx *__trace) inner() Context { return &ctx.builtinbase }
func (ctx *__trace) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trace) x() (res any) {
    for _, a := range ctx.a {
        note(ctx, "%v", ts(a), trace{})
    }
    return
}

// $(defor $(x),$(y),$(z)) is identical to $(if $(defined $(x)),$(x),...)
type __defor struct { builtinbase } // aka. defined-or
func (ctx *__defor) inner() Context { return &ctx.builtinbase }
func (ctx *__defor) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__defor) x() (res any) {
    for _, a := range merge(ctx.a...) {
        erro(ctx, "TODO: %v", ts(a))

        var unres bool
        if unres {
            continue
        } else {
            res = a
            break
        }
    }
    return
}

type __or struct { builtinbase }
func (ctx *__or) inner() Context { return &ctx.builtinbase }
func (ctx *__or) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__or) x() (res any) {
	for _, a := range merge(ctx.a...) {
		if a = expand(ctx, a); __true(ctx, a) { return a }
	}
	return
}

type __and struct { builtinbase }
func (ctx *__and) inner() Context { return &ctx.builtinbase }
func (ctx *__and) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__and) x() (res any) {
    for _, a := range merge(ctx.a...) {
        if a = expand(ctx, a); __true(ctx, a) { res = a } else { return nil }
    }
    return
}

// $(not x y z) ⇒ (not (or x y z))
// $(not x,y,z) ⇒ (and (not x) (not y) (not z))
type __not struct { builtinbase }
func (ctx *__not) inner() Context { return &ctx.builtinbase }
func (ctx *__not) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__not) x() (res any) {
    var t bool
    for _, a := range ctx.a { if t = __true(ctx, expand(ctx, a)); t { break } }
    return !t
}

type __xor struct { builtinbase }
func (ctx *__xor) inner() Context { return &ctx.builtinbase }
func (ctx *__xor) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__xor) x() (res any) {
    if vals := merge(ctx.a...); len(vals) > 1 {
        var t = __true(ctx, expand(ctx, vals[0]))
        for _, a := range vals[1:] {
            if __true(ctx, expand(ctx, a)) != t {
                return _boolean(a.Pos(), true)
            }
        }
    }
    return
}

type __unequal struct { builtinbase }
func (ctx *__unequal) inner() Context { return &ctx.builtinbase }
func (ctx *__unequal) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__unequal) x() (_ any) {
    if len(ctx.a) != 2 {
        debug(ctx, _f("unequal: wrong number of arguments: %v", ctx.a),
			_f("try: $(unequal <value-list>,<value-list>)"), trace{})
    }

    var a = expand(_final(ctx), ctx.a[0])
    var b = expand(_final(ctx), ctx.a[1])
    var t = cmp(ctx, a, b) != cmpEqual

    if t {
        return _boolean(_pos(ctx), true)
    } else if n := ctx.debug; n>0 {
        if l, y := a.(*list); y {
            var v = l.elems[0]
            warn(ctx, "unequal: a: %T(len=%d), %T %v", a, len(l.elems), v, v)
        } else {
            warn(ctx, "unequal: a: %T %v", a, a)
        }
        if l, y := b.(*list); y {
            var v = l.elems[0]
            warn(ctx, "unequal: b: %T(len=%d), %T %v", b, len(l.elems), v, v)
        } else {
            warn(ctx, "unequal: b: %T %v", b, b)
        }
        debug(ctx, "unequal: %v", t, callstack{num:n})
    } else if len(ctx.a)>2 {
        debug(ctx, "unequal: extra args specified: %v", ctx.a[2])
    }
    return
}

type __equal struct { builtinbase; str bool `str,string` }
func (ctx *__equal) inner() Context { return &ctx.builtinbase }
func (ctx *__equal) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__equal) x() (_ any) {
    if len(ctx.a) != 2 {
        debug(ctx, "wrong number of arguments: %v", ctx.a)
        note(ctx, "try: $(equal <value-list>,<value-list>)", trace{})
    }

    args := expands(ctx, ctx.a...)

    if a, b := args[0], args[1]; ctx.str {
        return __string(ctx, a) == __string(ctx, b)
    } else {
        return cmp(ctx, a, b) == cmpEqual
    }
}

type __greater struct { builtinbase; str bool `str,string` }
func (ctx *__greater) inner() Context { return &ctx.builtinbase }
func (ctx *__greater) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__greater) x() (res any) {
    if len(ctx.a) != 2 {
        debug(ctx, "wrong number of arguments: %v", ctx.a)
        note(ctx, "try: $(greater <value-list>,<value-list>)", trace{})
    }

    args := expands(ctx, ctx.a...)

    if a, b := args[0], args[1]; ctx.str {
        if __string(ctx, a) > __string(ctx, b) { return true }
    } else {
        if cmp(ctx, a, b) == cmpGreater { return true }
    }
    return
}

type __less struct { builtinbase; str bool `str,string` }
func (ctx *__less) inner() Context { return &ctx.builtinbase }
func (ctx *__less) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__less) x() (res any) {
    if len(ctx.a) != 2 {
        debug(ctx, "wrong number of arguments: %v", ctx.a)
        note(ctx, "try: $(greater <value-list>,<value-list>)", trace{})
    }

    args := expands(ctx, ctx.a...)

    if a, b := args[0], args[1]; ctx.str {
        if __string(ctx, a) < __string(ctx, b) { return true }
    } else {
        if cmp(ctx, a, b) == cmpSmaller { return true }
    }
    return
}

// $(match val1 val2 val3, a b c d...)
// $(match -rx=r1 -rx=r2 -rx=r3, a b c d...)
type __match struct { builtinbase
    regexps []*regexp.Regexp //`re,rx,reg,regex,regexp`
    negated bool `ne,neg,negated,negative,not`
	shallow bool `shallow` // TODO: shallow match without expand $(xx) or &(yy)
    all bool `all`
}
func (ctx *__match) inner() Context { return &ctx.builtinbase }
func (ctx *__match) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__match) x() (result any) {
    if n := len(ctx.a); n < 2 {
        erro(ctx, "wrong arguments, try: $(match <regexp-list>,<value-list-1>,...)")
    }

    var leftList, rightList []Value

    if true {
        leftList, rightList = xmerge(ctx, ctx.a[0]), xmerge(ctx, ctx.a[1:]...)
    } else {
        leftList, rightList = merge(ctx.a[0]), merge(ctx.a[1:]...)
    }

    var res *boolean

    if ctx.negated {
        defer func() {
            if res != nil {
                res.bool = !res.bool
            } else {
                result = _boolean(_pos(ctx), true)
            }
        } ()
    }

    for _, left := range leftList {
        for _, right := range rightList {
            var matched bool
            if !patterned(ctx, left) && patterned(ctx, right) {
                matched, _, _, _ = match(ctx, right, left)
            } else {
                matched, _, _, _ = match(ctx, left, right)
            }
            if matched {
                if res == nil { res = _boolean(_pos(ctx), true) }
                if !ctx.all { return res }
            } else if ctx.all {
                res = nil
                return res
            }
        }
    }

    if res != nil { result = res }
    return
}
func (ctx *__match) _x() (res any) {
    var patList, valList []Value
    if n := len(ctx.a); n < 1 {
        erro(ctx, "wrong arguments, try: $(match <regexp-list>,<value-list>,...)")
    }

    if len(ctx.a) > 1 {
        patList = merge(ctx.a[0])
        valList = merge(ctx.a[1:]...)
    } else {
        valList = merge(ctx.a[0])
    }
    if ctx.debug > 0 {
        var ( n = len(ctx.a) ; d = ctx.debug )
        debug(ctx, "match: %v %v %v, %d", ctx.regexps, patList, valList, n, callstack{num:d})
    }

    var pos = _pos(ctx)
ForValList:
    for _, val := range valList {
        if isTrivial(val) { continue ForValList }

        var str = __string(ctx, val)
        for _, rx := range ctx.regexps {
            var matched = rx.MatchString(str);
            if ctx.negated { matched = !matched }
            if matched {
                if ctx.all {
                    if res == nil { res = _boolean(pos, true) }
                } else {
                    return _boolean(pos, true)
                }
            } else if ctx.all {
                return nil
            }
        }
        for _, pat := range patList {
            var matched, _, _, _ = match(ctx, pat, val)
            if ctx.negated { matched = !matched }
            if matched {
                if ctx.all {
                    if res == nil { res = _boolean(pos, true) }
                } else {
                    return _boolean(pos, true)
                }
            } else if ctx.all {
                return nil
            }
        }

        if ctx.debug > 0 {
            debug(ctx, _f("match: %v", str), _f("match: %v %T", val, val))
        }
    }
    return
}

// 1: $(case     (a 'xxx') (b 'yyy') (c 'zzz') (yes 'else'))
// 2: $(case val (a 'xxx') (b 'yyy') (c 'zzz') ('if none or nil'))
// 3: $(case val (a 'xxx') (b 'yyy') (c 'zzz') (- 'if none or nil'))
// 4: $(case val (a 'xxx') (b 'yyy') (c -) (- -))
type __case struct { builtinbase }
func (ctx *__case) inner() Context { return &ctx.builtinbase }
func (ctx *__case) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__case) x() (res any) {
    var val Value
    var args = merge(ctx.a...)
    if len(args) == 0 {
        return
    }
    if _, y := args[0].(*group); !y {
        val = expand(ctx, args[0])
        args = args[1:]
    }

    var def []Value
    for _, arg := range args { if g, y := arg.(*group); y && len(g.elems)>0 {
        if n := len(g.elems); val != nil && isNone(val) && n == 1 {
            return g.elems[0]
        } else if n == 1 {
            def = append(def, g.elems[0])
            continue
        }

        var collect bool
        var v = expand(ctx, g.elems[0])
        if val == nil && v != nil && __true(ctx, v) {
            collect = true
        } else if val != nil && isTrivial(val) {
            if isTrivial(v) {
                collect = true
            } else if f, y := v.(flag); y && isNull(f.Value) {
                collect = true
            }
        } else if val != nil && cmp(ctx, val, v) == cmpEqual {
            collect = true
        }
        if !collect { continue }

        var vals []Value
        for _, v := range g.elems[1:] {
            if f, y := v.(flag); !y || isNull(f.Value) {
                vals = append(vals, v)
            }
        }
        return vals
    } else {
        erro(pc(ctx,arg), "unexpected case: %v", ts(arg))
    }}
    return
}

type __if struct { builtinbase }
func (ctx *__if) inner() Context { return &ctx.builtinbase }
func (ctx *__if) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__if) ts(string) string {
    if n := len(ctx.a); n > 0 {
        var s = ctx.a[0].String()
        if n > 1 { s += ","+ctx.a[1].String() }
        if n > 2 { s += ","+ctx.a[2].String() }
        if n > 3 { s += ","+ctx.a[3].String() }
        if s != "" { s += " " }
        return "{=if "+s+ts(ctx.Context)+"}"
    } else {
        return "{=if "+ts(ctx.Context)+"}"
    }
}
func (ctx *__if) x() (res any) {
	if 1 < len(ctx.a) {
		if __true(ctx, expand(ctx, ctx.a[0])) {
			return expand(ctx, ctx.a[1])
		} else {
			return expands(ctx, ctx.a[2:]...)
		}
	}
	return
}

type __ifarg struct { builtinbase }
func (ctx *__ifarg) inner() Context { return &ctx.builtinbase }
func (ctx *__ifarg) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__ifarg) x() (_ any) {
    if 1 < len(ctx.a) {
        if d := auto_find(ctx, __symbol(ctx, ctx.a[0])); d != nil && !isTrivial(d.value) {
            return ctx.a[1]
        } else {
            return ease(ctx, ctx.a[2:])
        }
    }
    return
}

type __ifdef struct { builtinbase }
func (ctx *__ifdef) inner() Context { return &ctx.builtinbase }
func (ctx *__ifdef) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__ifdef) x() (_ any) {
    if 1 < len(ctx.a) {
        if d := _scope(ctx).finddef(__symbol(ctx, ctx.a[0])); d != nil && !isTrivial(d.value) {
            return ctx.a[1]
        } else {
            return ease(ctx, ctx.a[2:])
        }
    }
    return
}

type __ifeq struct { builtinbase }
func (ctx *__ifeq) inner() Context { return &ctx.builtinbase }
func (ctx *__ifeq) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__ifeq) x() (_ any) {
    if 2 < len(ctx.a) {
        if equal(ctx, ctx.a[0], ctx.a[1]) {
            return ctx.a[2]
        } else {
            return ease(ctx, ctx.a[3:])
        }
    }
    return
}

type __ifne struct { builtinbase }
func (ctx *__ifne) inner() Context { return &ctx.builtinbase }
func (ctx *__ifne) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__ifne) x() (_ any) {
    if 2 < len(ctx.a) {
        if !equal(ctx, ctx.a[0], ctx.a[1]) {
            return ctx.a[2]
        } else {
            return ease(ctx, ctx.a[3:])
        }
    }
    return
}

type __for struct { builtinbase ; empty bool `allow-empty,empty` }
func (ctx *__for) inner() Context { return &ctx.builtinbase }
func (ctx *__for) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__for) x() (res any) {
    erro(ctx, "TODO: $(for): %v", ts(ctx.a))
    return
}

type __foreach struct {
	builtinbase
	empty  bool `allow-empty,empty`
	unique bool `unique`
}
func (ctx *__foreach) inner() Context { return &ctx.builtinbase }
func (ctx *__foreach) cast(t reflect.Type) Context {
	if reflect.TypeOf(ctx) == t { return ctx }
	return ctx.builtinbase.cast(t)
}
func (ctx *__foreach) x() (res any) {
	if len(ctx.a) == 0 {
		return
	}

	var vals []Value

	// FIX 1: Use []Value to safely handle hash collisions
	var um map[uint64][]Value
	if ctx.unique {
		um = make(map[uint64][]Value)
	}

	for _, val := range merge(expand(ctx, ctx.a[0])) {
		if !ctx.empty && isEmpty(val) {
			continue
		} else if ctx.unique {
			var t = hash(ctx, val)
			var isDuplicate bool

			// Resolve potential hash collisions via structural equality
			if existing, found := um[t]; found {
				for _, ev := range existing {
					if equal(ctx, ev, val) {
						isDuplicate = true
						break
					}
				}
			}

			if isDuplicate {
				continue
			}
			um[t] = append(um[t], val)
		}

		// NOTE: don't use defStatic (it's for codeblock auto)
		// redis() will now perfectly receive the raw *auto node!
		ctx.set(ctx, defVoid, symUnderscore, redis(val))

		for _, v := range merge(expands(ctx, ctx.a[1:]...)...) {
			if !ctx.empty && isEmpty(v) {
				continue
			}
			// FIX 2: Safely fallback to the macro's position to avoid nil panic
			if v == nil {
				v = _null(_pos(ctx))
			}
			vals = append(vals, v)
		}
	}
	return vals
}

type __count struct { builtinbase ; vals []Value `value` }
func (ctx *__count) inner() Context { return &ctx.builtinbase }
func (ctx *__count) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__count) x() (res any) {
    var num int64
    var vals = valvec(ctx.vals)
    for _, a := range ctx.a {
        if __true(ctx, a) || vals.has2(ctx, a) { num += 1 }
    }
    return num
}

type __env struct { builtinbase }
func (ctx *__env) inner() Context { return &ctx.builtinbase }
func (ctx *__env) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__env) x() (res any) {
    var vals []Value
    for _, a := range ctx.a {
        if val := expand(ctx, a); isTrivial(val) {
            continue
        } else if s := strings.TrimSpace(__string(ctx, val)); s != "" {
            vals = append(vals, _rw(a.Pos(), os.Getenv(s)))
        }
    }
    return vals
}

type __auto struct { builtinbase }
func (ctx *__auto) inner() Context { return &ctx.builtinbase }
func (ctx *__auto) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__auto) x() (_ any) {
    if 0 < len(ctx.a) {
        for _, a := range merge(ctx.o...) {
            switch t := a.(type) {
            case *pair:
                if k := __symbol(ctx, t.key); k == symEmpty {
                    erro(pc(ctx,a), "empty name: %v : %s", t.key, ts(t.key,ctx))
                } else {
                    ctx.set(ctx, defVoid, k, t.val)
                }
            default:
                erro(pc(ctx,a), "wrong auto def: %s : %s", a, ts(a,ctx))
            }
        }
        return expands(ctx, ctx.a...)
    }
    return
}

type __var struct { builtinbase }
func (ctx *__var) inner() Context { return &ctx.builtinbase }
func (ctx *__var) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__var) x() (res any) {
    return // TODO: ???
}

type __call struct { builtinbase ; closure bool `closure` }
func (ctx *__call) inner() Context { return &ctx.builtinbase }
func (ctx *__call) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__call) _x() (res any) {
    var vals []Value
    for _, a := range merge(ctx.a[0]) {
        var x Value
        var s = __symbol(ctx, a)
        if s == symEmpty {
            erro(ctx, "empty string: %v : %v", a, ts(a,ctx))
        } else if ctx.closure {
            x = closure_resolve(ctx, s)
        } else {
            x = project_resolve(ctx, s)
        }
        if x == nil { x = auto_get(ctx, s) }
        if x != nil {
            if v := evoke(ctx, x, nil, ctx.a[1:]); v != nil {
                vals = append(vals, v)
            }
        }
    }
    return vals
}
func (ctx *__call) x() (_ any) {
    var s = ctx.a[0].String()
    for _, v := range ctx.a[1:] { s += " " + v.String() }
    erro(ctx, "deprecated $(call %s), use $(%s)", s, s)
    return
}

type __value struct { builtinbase ; closure bool `closure` }
func (ctx *__value) inner() Context { return &ctx.builtinbase }
func (ctx *__value) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__value) x() (res any) {
    var vals []Value
    var p = _project(ctx)
    for _, a := range merge(ctx.a...) {
        var s = __symbol(ctx, a)
        if s != symEmpty {
            var x Value
            if ctx.closure {
                x = closure_resolve(ctx, s)
            } else {
                x = p.resolve(ctx, s)
            }
            if x == nil { x = auto_get(ctx, s) }
            if x != nil {
                if d, y := x.(*def); y {
                    vals = append(vals, d.value)
                }
            }
        }
    }
    return vals
}

type __defs struct { builtinbase
    n int `num,number`
    r int `capture`
	sort bool `sort`
}
func (ctx *__defs) inner() Context { return &ctx.builtinbase }
func (ctx *__defs) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__defs) x() (_ any) {
	var pos = _pos(ctx)
    var pats, names []Value
    for _, v := range merge(ctx.a...) { pats = append(pats, v) }

defsloop:
    for k, _ := range _project(ctx).elems {
		var name = _word(pos, k)
        for _, pat := range pats {
            var neg bool
            if x, y := pat.(negative); y { pat, neg = x.Value, y }

            var a, _, _, c = match(pc(ctx, pat), pat, name)
            if a && neg { continue defsloop }
            if a || neg {
                if ctx.r <= 0 || 0 == len(c) {
					names = append(names, name)
                } else if ctx.r <= len(c) {
					names = append(names, c[ctx.r-1])
                }
				continue defsloop
            }
        }
    }
	if ctx.sort { slices.SortFunc(names, func(a, b Value) int { return int(cmp(ctx, a, b)) }) }
    return names
}

type __list struct { builtinbase }
func (ctx *__list) inner() Context { return &ctx.builtinbase }
func (ctx *__list) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__list) x() (res any) {
    return ctx.a
}

type __plain struct { builtinbase
    scope_ bool `findscope,find-scope,scope`
}
func (ctx *__plain) inner() Context { return &ctx.builtinbase }
func (ctx *__plain) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__plain) x() (res any) {
    var scope = _scope(ctx)
    for _, a := range ctx.a {
        var ( o object ; s = __symbol(ctx, a) )
        if ctx.scope_ { _, o = scope.find(s) } else { o = project_resolve(ctx, s) }
        if o == nil {
            erro(ctx, "no such symbol: %s", s)
        } else if d, y := o.(*def); !y {
            erro(ctx, "not a def: %s: %v", s, typeof(o))
        } else if d.value != nil {
            d.value = expand(ctx, d.value)
        }
    }
    return
}

type __shell struct { builtinbase }
func (ctx *__shell) inner() Context { return &ctx.builtinbase }
func (ctx *__shell) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__shell) x() (res any) {
    var err error
    var vals []Value
    var pos = _pos(ctx)
    for _, a := range ctx.a {
        var bufout, buferr bytes.Buffer
        var s = __string(ctx, a)
        sh := exec.Command("sh", "-c", s)
        sh.Stdout, sh.Stderr = &bufout, &buferr
        if err = sh.Run(); err != nil {
            s = strings.TrimSpace(buferr.String())
            if !strings.HasPrefix(s, ":") { s = ":\n" + s }
            prompt(ctx, "%s%s\n", __string(ctx, a), s)
            erro(ctx, "%s", err)
            return
        }
        val := _raw(pos, strings.TrimSpace(bufout.String()))
        vals = append(vals, val)
        bufout.Reset()
        buferr.Reset()
    }
    return vals
}

type __which struct { builtinbase }
func (ctx *__which) inner() Context { return &ctx.builtinbase }
func (ctx *__which) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__which) x() (res any) {
    var vals []Value
	var pos = _pos(ctx)
    for _, a := range ctx.a {
        if s, err := exec.LookPath(__string(ctx, a)); err != nil {
            erro(ctx, "%v", err)
        } else if s != "" {
            vals = append(vals, _raw(pos, s))
        }
    }
    return vals
}

type __servehttp struct { builtinbase
    ssl bool `ssl`
    host string `host`
    port int `port`
}
func (ctx *__servehttp) inner() Context { return &ctx.builtinbase }
func (ctx *__servehttp) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__servehttp) x() (res any) {
    if ctx.port == 0 { ctx.port = 80 }
    if ctx.ssl {
        erro(ctx, "'serve-http(-ssl)' is unimplemented yet")
    }

    var server = http.Server{}
    server.Addr = fmt.Sprintf("%s:%d", ctx.host, ctx.port)
    info(ctx, "serving http at %v ...", server.Addr)

    var root string
    var quit = func(w http.ResponseWriter, r *http.Request) {
        var s = "<font color=red>stop serving '%s' close in a second ...</font>"
        io.WriteString(w, fmt.Sprintf(s, root))
        go func() {
            time.Sleep(1 * time.Second)
            server.Shutdown(context.Background())
        } ()
    }

    http.HandleFunc("/-/end",  quit)
    http.HandleFunc("/-/quit", quit)
    http.HandleFunc("/-/shut", quit)

    if ctx.a == nil {
        http.Handle("/", http.FileServer(http.Dir(_workdir(ctx))))
    } else {
        for _, a := range ctx.a {
            var s = __string(ctx, a)
            info(ctx, "serving files %v ...", s)
            http.Handle("/", http.FileServer(http.Dir(s)))
        }
    }

    flush(ctx)

    var err = server.ListenAndServe()
    if err != nil && err != http.ErrServerClosed {
        erro(ctx, "%s", err)
    }
    return
}

type __append struct { builtinbase
    auto    bool `auto`
    closure bool `closure`
}
func (ctx *__append) inner() Context { return &ctx.builtinbase }
func (ctx *__append) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__append) x() (_ any) {
    if len(ctx.a) < 2 {
        erro(ctx, "insufficient number of arguments: %v", ctx.a)
    }

    var names []Value
    if names = merge(ctx.a[0]); len(names) == 0 {
        debug(ctx, "append to nowhere: %v", ts(ctx.a[0]))
        return
    }

    var vals []Value
    for _, a := range names {
        var s = __symbol(ctx, a)
        var d *def
        if s == symEmpty {
            erro(ctx, "'%v' is empty for name", a)
        } else if ctx.auto {
            d = auto_find(ctx, s)
        } else if ctx.closure {
            debug(ctx, "closure: %v", a) // d = closure_finddef(ctx, s)
        } else if o := project_resolve(ctx, s); o != nil {
            d, _ = o.(*def)
        }
        if d == nil {
            erro(ctx, "%v → %s is undefined", a, s)
        } else {
            if vals == nil {
                if vals = merge(ctx.a[1:]...); len(vals) == 0 {
                    debug(ctx, "append no values: %v", ctx.a[1:])
                    return
                }
            }
            d.append(ctx, vals...)
        }
    }
    return
}

type __plus struct { builtinbase ; int bool `int,integer` }
func (ctx *__plus) inner() Context { return &ctx.builtinbase }
func (ctx *__plus) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__plus) x() (res any) {
    if ctx.int {
        var num int64
        for n, a := range ctx.a {
            var i = __int(ctx, a)
            if n == 0 { num = i } else { num += i }
        }
        return _decimal(_pos(ctx), num)
    } else {
        var num float64
        for n, a := range ctx.a {
            var f = __float(ctx, a)
            if n == 0 { num = f } else { num += f }
        }
        return _float(_pos(ctx), num)
    }
}

type __minus struct { builtinbase ; int bool `int,integer` }
func (ctx *__minus) inner() Context { return &ctx.builtinbase }
func (ctx *__minus) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__minus) x() (res any) {
    if ctx.int {
        var num int64
        for n, a := range ctx.a {
            var i = __int(ctx, a)
            if n == 0 { num = i } else { num -= i }
        }
        return _decimal(_pos(ctx), num)
    } else {
        var num float64
        for n, a := range ctx.a {
            var f = __float(ctx, a)
            if n == 0 { num = f } else { num -= f }
        }
        return _float(_pos(ctx), num)
    }
}

type __multiply struct { builtinbase ; int bool `int,integer` }
func (ctx *__multiply) inner() Context { return &ctx.builtinbase }
func (ctx *__multiply) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__multiply) x() (res any) {
    if ctx.int {
        var num int64
        for n, a := range ctx.a {
            var i = __int(ctx, a)
            if n == 0 { num = i } else { num *= i }
        }
        return num
    } else {
        var num float64
        for n, a := range ctx.a {
            var f = __float(ctx, a)
            if n == 0 { num = f } else { num *= f }
        }
        return num
    }
}

type __divide  struct { builtinbase ; int bool `int,integer` }
func (ctx *__divide) inner() Context { return &ctx.builtinbase }
func (ctx *__divide) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__divide) x() (res any) {
    if ctx.int {
        var num int64
        for n, a := range ctx.a {
            var i = __int(ctx, a)
            if n == 0 { num = i } else { num /= i } // FIXME: NaN
        }
        return num
    } else {
        var num float64
        for n, a := range ctx.a {
            var f = __float(ctx, a)
            if n == 0 { num = f } else { num /= f } // FIXME: NaN
        }
        return num
    }
}

type __unique struct { builtinbase
    reverse  bool `reverse`
    keepAuto bool `auto,keepauto,keep-auto`
    unexpand bool `unexpand,noexpand,no-expand`
}
func (ctx *__unique) inner() Context { return &ctx.builtinbase }
func (ctx *__unique) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__unique) x() (_ any) {
    var args = ctx.a
    var t1, t2 time.Time

    if false { defer func() {
        t3 := time.Now()
        d0 := t3.Sub(t1)
        d1 := t2.Sub(t1)
        d2 := t3.Sub(t2)
        if d0 > 1*time.Second {
            for _, a := range args { __string(ctx, a) }
            t4 := time.Now()
            d3 := t4.Sub(t3)
            for i, a := range args { if i > 0 { cmp(ctx, a, args[i-1]) } }
            t5 := time.Now()
            d4 := t5.Sub(t4)
            // for i, a := range args { if i > 0 { eq(ctx, a, args[i-1]) } }
            for i, a := range args { if i > 0 { equal(ctx, a, args[i-1]) } }
            t6 := time.Now()
            d5 := t6.Sub(t5)
            var args2 []Value
            var seen = make(map[uint64]struct{})
            for _, a := range args {
                c := hash(ctx, a)
                if _, y := seen[c]; y {
                    note(ctx, "%v")
                } else {
                    seen[c] = struct{}{}
                }
                var t = true
                for _, b := range args2 {
                    if equal(ctx, a, b) { t = false ; break }
                }
                if t { args2 = append(args2, a) }
            }
            t7 := time.Now()
            d6 := t7.Sub(t6)
            debug(ctx, "%v %v %v (%v, %v, %v, %v, %d %d)", d0, d1, d2, d3, d4, d5, d6, len(args), len(args2))
            t7 = time.Now()
            unique(ctx, args...)
            d6 = t7.Sub(t6)
            erro(ctx, "unique: %v", d6)
        }
    }()}

    t1 = time.Now()

    if ctx.unexpand {
        args =  merge(args...)
    } else {
        args = xmerge(ctx, args...)
    }

    t2 = time.Now()

    if ctx.reverse {
        return reverse_unique(ctx, args...)
    } else {
        return         unique(ctx, args...)
    }
}

type __join struct { builtinbase }
func (ctx *__join) inner() Context { return &ctx.builtinbase }
func (ctx *__join) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__join) x() any {
	l := len(ctx.a)
	if l == 0 {
		return nil
	}

	var sep Value
	var vals []Value

	if l == 1 {
		vals = merge(ctx.a...)
	} else {
		sep = scalarize(ctx.a[l-1])
		vals = merge(ctx.a[:l-1]...)
	}

	// 1. Use the Re-Disjunction pass purely for DETECTION.
	// We deliberately discard the mutated AST to prevent `*disjunction` wrapping!
	_, dynamicVals := _redis_elems(vals)
	_, dynamicSep := _redis(sep)

	// 2. If dynamic closures exist, return the pure, unmutated conjunction.
	// The conjunction natively handles its own closures without redis!
	if dynamicVals || dynamicSep {
		return &conjunction{list{elements{vals}}, sep}
	}

	// 3. FAST PATH: All elements are perfectly static. Eagerly join them!
	var valid []Value
	for _, v := range vals {
		if !isEmpty(v) {
			valid = append(valid, v)
		}
	}

	if len(valid) == 0 {
		return nil
	}

	t := &compound{}
	t.app(valid[0])

	if sep != nil {
		for _, v := range valid[1:] {
			t.app(sep, v)
		}
	} else {
		for _, v := range valid[1:] {
			t.app(v)
		}
	}

	return t
}

type __conjunct struct { builtinbase }
func (ctx *__conjunct) inner() Context { return &ctx.builtinbase }
func (ctx *__conjunct) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__conjunct) x() (_ any) {
	if l := len(ctx.a); 0 < l {
		var con = new(conjunction)
		if l < 2 {
			con.elems = merge(ctx.a...)
		} else {
			con.elems = merge(ctx.a[:l-1]...)
			con.sep  = ctx.a[l-1]
		}
		return con
	}
	return
}

type __quote struct { builtinbase }
func (ctx *__quote) inner() Context { return &ctx.builtinbase }
func (ctx *__quote) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__quote) x() any {
    return &quote{list{elements{ctx.a}}}
}

type __quotejoin struct { builtinbase }
func (ctx *__quotejoin) inner() Context { return &ctx.builtinbase }
func (ctx *__quotejoin) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__quotejoin) x() (res any) {
    var sep string
    var args = merge(ctx.a...)
    if l := len(args); l > 1 {
        sep = __string(ctx, args[l-1])
        args = args[:l-1]
    }
    if l := len(args); l > 0 {
        var fields []string
        for _, a := range args[1:] {
            if v := __string(ctx, a); v != "" { fields = append(fields, v) }
        }
        res = _strlit(_pos(ctx), strconv.Quote(strings.Join(fields, sep)))
    } else {
        res = _none(_pos(ctx))
    }
    return
}

// $(split .,1.2.3)
type __split struct { builtinbase
    sep string `sep,separator`
}
func (ctx *__split) inner() Context { return &ctx.builtinbase }
func (ctx *__split) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__split) x() (res any) {
    if 0 < len(ctx.a) {
        var fields []Value
        var sep = ctx.sep
        if sep == "" { sep = __string(ctx, ctx.a[0]) }
        for _, a := range ctx.a[1:] {
            for _, s := range strings.Split(__string(ctx, a), sep) {
                fields = append(fields, _strlit(a.Pos(), s))
            }
        }
        return fields
    }
    return
}

func quotestrings(value Value) {
    switch v := value.(type) {
    case *strlit: v.s = strconv.Quote(v.s)
    case *list:
        for _, elem := range v.elems {
            quotestrings(elem)
        }
    }
    return
}

func joinstrings(ctx Context, value Value, sep string) (res Value, err error) {
    if sep == "" { sep = " " }
ValueType:
    switch v := value.(type) {
    case *strlit: res = value
    case *list:
        var strs []string
        for _, elem := range v.elems {
            var ( v Value; s string )
            if v, err = joinstrings(ctx, elem, sep); err != nil { break ValueType }
            if s = __string(ctx, v); s != "" { strs = append(strs, s) }
        }
        res = _strlit(value.Pos(), strings.Join(strs, sep))
    }
    return
}

// TODO: deprecate this and add -quote to __split
type __splitquote struct { __split }
func (ctx *__splitquote) inner() Context { return &ctx.builtinbase }
func (ctx *__splitquote) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__splitquote) x() (res any) {
    res = ctx.__split.x()
    if v, y := res.(Value); y && v != nil { quotestrings(v) }
    return
}

// TODO: deprecate this and add -quote to __split
type __splitquotejoin struct { __split }
func (ctx *__splitquotejoin) inner() Context { return &ctx.builtinbase }
func (ctx *__splitquotejoin) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__splitquotejoin) x() (res any) {
    res = ctx.__split.x()
    if val, y := res.(Value); y && val != nil {
        var err error
        var sep string
        if l := len(ctx.a); l > 1 {
            sep = __string(ctx, ctx.a[l-1])
            ctx.a = ctx.a[:l-1]
        }
        if res, err = joinstrings(ctx, val, sep); err != nil {
            erro(ctx, "%v", err)
        }
    }
    return
}

type __splitjoinquote struct { __split }
func (ctx *__splitjoinquote) inner() Context { return &ctx.builtinbase }
func (ctx *__splitjoinquote) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__splitjoinquote) x() (res any) {
    res = ctx.__split.x()
    if val, y := res.(Value); y && val != nil {
        var err error
        var sep string
        if l := len(ctx.a); l > 1 {
            sep = __string(ctx, ctx.a[l-1])
            ctx.a = ctx.a[:l-1]
        }

        var v Value
        if v, err = joinstrings(ctx, val, sep); err != nil {
            erro(ctx, "%v", err)
        } else {
            res = _strlit(_pos(ctx), strconv.Quote(__string(ctx, v)))
        }
    }
    return
}

type __element struct { builtinbase }
func (ctx *__element) inner() Context { return &ctx.builtinbase }
func (ctx *__element) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__element) x() (res any) {
	var elems []Value
	for _, o := range ctx.o {
		x, ok := unbox(expand(ctx, o)).(Value)
		if ok && x.kind()&KindInteger != 0 {
			var i = int(__int(ctx, x))
			for _, v := range ctx.a {
				if a := xmerge(ctx, v); 0 <= i && i < len(a) {
					elems = append(elems, a[i])
				}
			}
		}
	}
	return elems
}

type __field struct { builtinbase }
func (ctx *__field) inner() Context { return &ctx.builtinbase }
func (ctx *__field) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__field) x() (res any) {
	var elems []Value
	for _, o := range ctx.o {
		x, ok := unbox(expand(ctx, o)).(Value)
		if ok && x.kind()&KindInteger != 0 {
			var i = int(__int(ctx, x))
			for _, v := range ctx.a {
				if a := strings.Fields(__string(ctx, v)); 0 < i && i <= len(a) {
					elems = append(elems, _rw(v.Pos(), a[i-1]))
				}
			}
		}
	}
	return elems
}

type __fields struct { builtinbase }
func (ctx *__fields) inner() Context { return &ctx.builtinbase }
func (ctx *__fields) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__fields) x() any {
	var elems []Value

	if len(ctx.a) == 2 {
		x, ok := unbox(expand(ctx, ctx.a[0])).(Value)
		if ok && x.kind()&KindInteger != 0 {
			var i = int(__int(ctx, x))
			for _, v := range ctx.a[1:] {
				if a := strings.Fields(__string(ctx, v)); 0 <= i && i < len(a) {
					elems = append(elems, _rw(v.Pos(), a[i]))
				}
			}
			return elems
		}
	}

	for _, v := range ctx.a {
		for _, s := range strings.Fields(__string(ctx, v)) {
			elems = append(elems, _rw(v.Pos(), s))
		}
	}
	return elems
}

type __usee struct { builtinbase }
func (ctx *__usee) inner() Context { return &ctx.builtinbase }
func (ctx *__usee) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__usee) x() (res any) {
    var proj = _project(ctx)
    if proj == nil {
        erro(ctx, "unknown current context")
    }

    var vals []Value
    for _, a := range ctx.a {
        v := sel(ctx, proj.use, __string(ctx, a))
        if v != nil { vals = append(vals, v.(Value)) }
    }
    if vals == nil { res = vals }
    return
}

type __uses struct { builtinbase }
func (ctx *__uses) inner() Context { return &ctx.builtinbase }
func (ctx *__uses) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__uses) x() (res any) {
    var proj = _project(ctx)
    if proj == nil {
        erro(ctx, "unknown current context")
    }

    var found bool

outer:
    for _, a := range ctx.a {
        var s = __symbol(ctx, a)
        for _, u := range proj.use.list {
            found = u.project.name == s
            if found { break outer }
        }
    }

    if found { res = found }
    return
}

type __path struct { builtinbase }
func (ctx *__path) inner() Context { return &ctx.builtinbase }
func (ctx *__path) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__path) x() any {
    var res []Value
    for _, a := range ctx.a {
        if x, y := a.(*path); y {
            res = append(res, x)
        } else {
            res = append(res, _pathStr(ctx, __string(ctx, a)))
        }
    }
    return res
}

type __bare struct { builtinbase
    name bool `name,filename,file-name,non-full,not-full`
}
func (ctx *__bare) inner() Context { return &ctx.builtinbase }
func (ctx *__bare) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__bare) x() (res any) {
    var vals []Value
    for _, a := range ctx.a {
        switch p := a.Pos(); t := a.(type) {
        case *strlit, *strcomp:
            a = _rw(p, __string(ctx, a))
        case *file:
            a = _rw(p, ident(ctx,t))
        case fullfile:
            if ctx.name {
                a = _rw(p, ident(ctx,t))
            } else {
                a = _rw(p, __string(ctx,t))
            }
        }
        vals = append(vals, a)
    }
    return vals
}

type __word struct { builtinbase }
func (ctx *__word) inner() Context { return &ctx.builtinbase }
func (ctx *__word) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__word) x() (res any) {
    var vals []Value
    for _, a := range ctx.a {
        if _, y := a.(*word); !y {
            a = _word(a.Pos(), __symbol(ctx, a))
        }
        vals = append(vals, a)
    }
    return vals
}

type __resolve struct { builtinbase
    closure bool `closure`
    // expand bool `expand`
}
func (ctx *__resolve) inner() Context { return &ctx.builtinbase }
func (ctx *__resolve) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__resolve) x() (res any) {
    if 0 < len(ctx.a) {
        var resolve func(Context, Symbol) object
        if ctx.closure {
            resolve = closure_resolve
        } else {
            resolve = project_resolve
        }

        var vals []Value
        for _, a := range merge(ctx.a...) {
            var name = __symbol(ctx, a)
            if o := resolve(ctx, name); o == nil {
                erro(ctx, "%v is nil : %v", a, ts(a))
            } else if x, y := o.(*def); !y {
                erro(ctx, "%v is not def : %v : %v", a, o, ts(o))
            } else if x.value != nil {
                vals = append(vals, merge(x.value)...)
            }
        }
        return vals
    }
    return
}

type __finalize struct { builtinbase }
func (ctx *__finalize) inner() Context { return &ctx.builtinbase }
func (ctx *__finalize) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__finalize) x() any {
    return expands(_final(ctx), ctx.a...)
}

type __filter struct { builtinbase
    stem bool `stem`
    neg bool
}
func (ctx *__filter) inner() Context { return &ctx.builtinbase }
func (ctx *__filter) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}

func (ctx *__filter) _x0(pats []Value, values ...Value) (result []Value) {
	defer func(t0 time.Time) {
		if d := time.Since(t0); d > 1*time.Second {
			debug(ctx,
				_f("slow: %v\n", d),
				_f("slow: %d result, %v\n", len(result), result),
				_f("slow: %d pats, %v\n", len(pats), pats))
		}
	}(time.Now())

	if len(values) == 0 {
		return
	}

	// 1. OPTIMIZATION: Pre-allocate capacity to eliminate slice growth allocations.
	// Filter-out (ctx.neg) typically keeps most items, while filter keeps fewer.
	capacity := len(values) / 2
	if ctx.neg {
		capacity = len(values)
	}
	result = make([]Value, 0, capacity)

	// 2. OPTIMIZATION: Memoization cache for repetitive AST nodes.
	// This prevents executing O(N * M) match() calls for duplicate values.
	type matchResult struct {
		matched bool
		val     Value // Stores the stem or the original value
	}
	cache := make(map[string]matchResult, capacity)

	for _, v := range values {
		// Use the string representation as the identity key. Expanded AST nodes
		// (like *word or *strlit) that represent the same text will hit the cache.
		var key string
		if v != nil {
			key = v.String()
		}

		// Fast-path: If we've already matched this exact string against the patterns, reuse it!
		if cached, exists := cache[key]; exists {
			if cached.matched {
				if !ctx.neg && cached.val != nil {
					result = append(result, cached.val)
				}
			} else {
				if ctx.neg && v != nil {
					result = append(result, v)
				}
			}
			continue
		}

		var matched bool
		var matchedVal Value

		// Slow-path: Check the value against all patterns
		for _, pat := range pats {
			if full, _, _, stems := match(ctx, pat, v); full {
				matched = true
				if ctx.stem {
					matchedVal = ease(ctx, stems)
				} else {
					matchedVal = v
				}
				break // Stop checking patterns once we have a match
			}
		}

		// Save the computed result into the cache for future identical values
		cache[key] = matchResult{
			matched: matched,
			val:     matchedVal,
		}

		// Apply filter vs filter-out logic
		if matched {
			if !ctx.neg {
				if matchedVal != nil {
					result = append(result, matchedVal)
				}
			}
		} else {
			if ctx.neg {
				if v != nil {
					result = append(result, v)
				}
			}
		}
	}
	return
}
func (ctx *__filter) _x(pats []Value, values ...Value) (result []Value) {
	if benchmark { defer func(t0 time.Time) {
		if d := time.Since(t0); d > 25*time.Millisecond {
			debug(ctx,
				_f("slow: %d values, %v, %d result", len(values), d, len(result)),
				_f("slow: %d pats: %v", len(pats), pats),
				callstack{num:5,stop:"smart.evoke"})
		}
	}(time.Now())}

	if len(values) == 0 {
		return
	}

	// 1. OPTIMIZATION: Pre-allocate capacity
	capacity := len(values) / 2
	if ctx.neg {
		capacity = len(values)
	}
	result = make([]Value, 0, capacity)

	// 2. OPTIMIZATION: Pre-compile fast-path matchers for patterns!
	// This completely bypasses the heavy, recursive AST match() function
	// for standard strings and percent-wildcards (like `%.in` or `CMakeLists.txt`).
	type fastPattern struct {
		isExact  bool
		exact    string
		isPerc   bool
		prefix   string
		suffix   string
		isRegex  bool
		regex    *regexp.Regexp
		fallback Value
	}

	fastPats := make([]fastPattern, 0, len(pats))
	for _, pat := range pats {
		fp := fastPattern{fallback: pat}

		p := unloc(pat)
		if pp, ok := p.(*percpat); ok {
			pfx, ok1 := staticStr(pp.Prefix)

			sufVal := pp.Suffix
			if inner, isPP := unloc(sufVal).(*percpat); isPP && isEmpty(inner.Prefix) {
				sufVal = inner.Suffix // Handle %%
			}
			suf, ok2 := staticStr(sufVal)

			if ok1 && ok2 {
				fp.isPerc = true
				fp.prefix = pfx
				fp.suffix = suf
			}
		} else if rx, ok := p.(*regexpat); ok {
			fp.isRegex = true
			fp.regex = rx.Regexp
		} else if s, ok := staticStr(p); ok && !patterned(ctx, p) {
			fp.isExact = true
			fp.exact = s
		}

		fastPats = append(fastPats, fp)
	}

	// 3. OPTIMIZATION: Memoization cache using the raw string identity.
	type matchResult struct {
		matched bool
		val     Value
	}
	cache := make(map[string]matchResult, capacity)

	for _, v := range values {
		var key string
		var vStr string
		if v != nil {
			vStr = quickStr(ctx, v)
			key = vStr // Use evaluated string as cache key for maximum hit rate
		}

		if cached, exists := cache[key]; exists {
			if cached.matched {
				if !ctx.neg && cached.val != nil {
					result = append(result, cached.val)
				}
			} else {
				if ctx.neg && v != nil {
					result = append(result, v)
				}
			}
			continue
		}

		var matched bool
		var matchedVal Value

		// Run the lightning-fast checks
		for _, fp := range fastPats {
			if fp.isExact {
				if vStr == fp.exact {
					matched = true
					if ctx.stem { matchedVal = _rw(v.Pos(), "") } else { matchedVal = v }
					break
				}
			} else if fp.isPerc {
				if len(vStr) >= len(fp.prefix)+len(fp.suffix) &&
					strings.HasPrefix(vStr, fp.prefix) &&
					strings.HasSuffix(vStr, fp.suffix) {
					matched = true
					if ctx.stem {
						stemStr := vStr[len(fp.prefix) : len(vStr)-len(fp.suffix)]
						matchedVal = _rw(v.Pos(), stemStr)
					} else {
						matchedVal = v
					}
					break
				}
			} else if fp.isRegex && fp.regex != nil {
				if fp.regex.MatchString(vStr) {
					matched = true
					matchedVal = v
					break
				}
			} else {
				// Heavy AST Matcher Fallback (Rarely used now!)
				if full, _, _, stems := match(ctx, fp.fallback, v); full {
					matched = true
					if ctx.stem {
						matchedVal = ease(ctx, stems)
					} else {
						matchedVal = v
					}
					break
				}
			}
		}

		cache[key] = matchResult{
			matched: matched,
			val:     matchedVal,
		}

		if matched {
			if !ctx.neg {
				if matchedVal != nil {
					result = append(result, matchedVal)
				}
			}
		} else {
			if ctx.neg {
				if v != nil {
					result = append(result, v)
				}
			}
		}
	}
	return
}

func (ctx *__filter) x() (res any) {
	if len(ctx.a) > 1 {
		var i int
		var pats = merge(expand(ctx, ctx.a[0]))

		if len(pats) > 0 {
			i = 1 // good
		} else if pats = merge(ctx.a[1]); len(pats) == 0 {
			erro(ctx, "no patterns: %v", ctx.a)
			return
		} else {
			i = 2
		}

		if i <= len(ctx.a) {
			res = ctx._x(pats, merge(expands(ctx, ctx.a[i:]...)...)...)
		} else {
			erro(ctx, "out of index: %d > %d, %v", i, len(ctx.a), ctx.a)
		}
	}
	return
}

// $(filter-out pattern…,text)
type __filterout struct { __filter }
func (ctx *__filterout) inner() Context { return &ctx.builtinbase }
func (ctx *__filterout) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__filterout) x() any {
    ctx.neg = true ; return ctx.__filter.x()
}

type __substring struct { builtinbase }
func (ctx *__substring) inner() Context { return &ctx.builtinbase }
func (ctx *__substring) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__substring) x() (_ any) {
	var res []Value
	if n := len(ctx.a); n > 1 {
		var v1, v2 = ctx.a[0], ctx.a[1]
		var a, b = intVal(ctx, v1, -1), intVal(ctx, v2, -1)

		if ctx.a = ctx.a[2:]; a < -1 && b < -1 {
			erro(ctx, "wrong indices (%v, %v)", v1, v2)
		}

		if a > b { t := a; a = b; b = t } // swap the wrong order
		if a == -1 { a = b }
		if a == -1 { return }

		// 1. Expand list arguments using merge()
		for _, arg := range merge(ctx.a...) {
			originalStr := __string(ctx, arg)
			s := originalStr
			i := len(s)

			// 2. Safely compute the substring bounds to prevent Go panics
			if i <= a {
				s = ""
			} else if b == -1 || b >= i {
				// If b is omitted (-1) or extends beyond the string, take the rest
				s = s[a:]
			} else {
				// Safely slice up to b
				s = s[a:b]
			}

			// 3. TYPE & MEMORY SAFETY FIX
			if s == originalStr {
				// Zero-mutation fast path: preserve the exact original AST node
				res = append(res, arg)
			} else {
				// Text changed: return a raw, garbage-collectable value
				res = append(res, _rw(arg.Pos(), s))
			}
		}
	}
	return res
}

// $(subst from,to,text)
type __subst struct { builtinbase }
func (ctx *__subst) inner() Context { return &ctx.builtinbase }
func (ctx *__subst) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__subst) x() (_ any) {
	var res []Value
	if len(ctx.a) > 2 {
		var s2 = __string(ctx, ctx.a[1])

		// 1. Expand and collect all 'from' substrings
		var fromStrs []string
		for _, arg := range merge(ctx.a[0]) {
			fromStrs = append(fromStrs, __string(ctx, arg))
		}

		// 2. Iterate over the text targets
		for _, arg := range merge(ctx.a[2:]...) {
			originalStr := __string(ctx, arg)
			s := originalStr

			// Apply all replacements
			for _, s1 := range fromStrs {
				if s1 != "" {
					s = strings.Replace(s, s1, s2, -1)
				}
			}

			// 3. TYPE & MEMORY SAFETY FIX
			if s == originalStr {
				// Zero-mutation fast path: preserve the exact original AST node (and type)
				res = append(res, arg)
			} else {
				// Text changed: return a raw, garbage-collectable value
				res = append(res, _rw(arg.Pos(), s))
			}
		}
	}
	return res
}

func coupleVal(ctx Context, v Value, str string) (_ Value) {
	// Use unbox just in case v is wrapped in *loc or *argumented
	switch unbox(v).(type) {
	case *strlit, *strcomp:
		return _strlit(_pos(ctx), str)
	case *path:
		return _pathStr(ctx, str)
	case *file, fullfile:
		// REFINED: Treat files as paths if they contain slashes, otherwise words.
		if strings.Contains(str, pathSep) { return _pathStr(ctx, str) }
		return _rw(_pos(ctx), str)
	default:
		if strings.Contains(str, pathSep) { return _pathStr(ctx, str) }
		return _rw(_pos(ctx), str)
	}
}

// $(patsubst pattern,replacement,text)
// TODO: supports: $(var:pattern=replacement)
// TODO: supports: $(var:suffix=replacement)
// TODO: support flags -name and -full for name-only and full-name-only matching
type __patsubst struct { builtinbase
    fullfiles bool `fullfile,fullfiles`
    filter    bool `filter,filter-unmatched,match-only` // Added filter option
}
func (ctx *__patsubst) inner() Context { return &ctx.builtinbase }
func (ctx *__patsubst) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__patsubst) matchPats(pats []Value, a Value) (ok bool, pat Value, stems []Value) {
	var rem Value
	for _, pat = range pats {
		if ok, _, rem, stems = match(ctx, pat, a); ok && (rem == nil || isEmpty(rem)) {
			return true, pat, stems
		}
	}
	return false, nil, nil
}
func (ctx *__patsubst) srcFile(proj *project, src Value) (srcFile *file, full bool) {
	var ok bool
	if srcFile, ok = to_file(src); ok {
		if full = ctx.fullfiles; !full { _, full = unbox(src).(fullfile) }
	}
	return
}
func (ctx *__patsubst) x() (_ any) {
	if benchmark { defer func(t0 time.Time) {
		if d := time.Since(t0); d > 25*time.Millisecond {
			debug(ctx,
				_f("slow: %d values, %v", len(ctx.a), d),
				callstack{num:5,stop:"smart.evoke"})
		}
	}(time.Now())}

	var srcPats, dstPats, sources, res []Value
	if nil != ctx.a {
		var l = len(ctx.a)
		if 0 < l { srcPats = xmerge(ctx, ctx.a[0]) }
		if 1 < l { dstPats = xmerge(ctx, ctx.a[1]) }
		if 2 < l { sources = xmerge(ctx, ctx.a[2:]...) }
	}

	var proj = _project(ctx)
	for _, src := range sources {
		var srcFile, full = ctx.srcFile(proj, src)

		// This now returns an exact, cleanly stripped stem (e.g., "foo")
		var ok, /* srcPat */_, stems = ctx.matchPats(srcPats, src)

		if !ok {
			// Pass-through unmatched files seamlessly
			if !ctx.filter && !isTrivial(src) { res = append(res, src) }
			continue
		}

		for _, dstPat := range dstPats {
			// Ignore ramnant check to allow static replacements
			if val, _ := stencil(ctx, dstPat, stems); isNull(val) {
				erro(ctx, "nil stencil: %v", dstPat)
			} else if srcFile != nil {
				// If the source was a file, we want to maintain its "file-like" identity
				// without incurring the cost of a full VFS/trie lookup.
				dst := proj.file(ctx, val)

				var str string

				// TIER 2: If AST mapping failed, flatten to string and force VFS resolution
				if dst == nil {
					if str = __string(ctx, val); str != "" {
						dst = _stat(ctx, str, stat_nonexist{true})
					}
				}

				// TIER 3: If VFS totally rejects it, degrade to a pure string substitution
				if dst == nil {
					if str != "" {
						res = append(res, coupleVal(pc(ctx, dstPat), src, str))
					}
					continue // Safely move to next pattern
				}

				if dst == nil {
					erro(ctx, "%v → %v (nil file)", srcFile, val, callstack{num:5})
				} else if dst.pos = src.Pos(); full {
					res = append(res, fullfile{dst})
				} else {
					res = append(res, dst)
				}
			} else {
				if str := __string(ctx, val); str != "" {
					// Uses the coupleVal function for standard string replacements
					res = append(res, coupleVal(pc(ctx, dstPat), src, str))
				}
			}
		}
	}
	return res
}

type __title struct { builtinbase }
func (ctx *__title) inner() Context { return &ctx.builtinbase }
func (ctx *__title) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__title) x() any {
    var res []Value
    for _, a := range ctx.a {
        switch t := a.(type) {
        case interface{ change(func(string) string) Value }:
            a = t.change(strings.Title)
        default:
            a = _rw(a.Pos(), strings.Title(__string(ctx, a)))
        }
    }
    return res
}

type __uppercase struct { builtinbase }
func (ctx *__uppercase) inner() Context { return &ctx.builtinbase }
func (ctx *__uppercase) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__uppercase) x() any {
    var res []Value
    for _, a := range ctx.a {
        switch t := a.(type) {
        case interface{ change(func(string) string) Value }:
            a = t.change(strings.ToUpper)
        default:
            a = _rw(a.Pos(), strings.ToUpper(__string(ctx, a)))
        }
        res = append(res, a)
    }
    return res
}

type __lowercase struct { builtinbase }
func (ctx *__lowercase) inner() Context { return &ctx.builtinbase }
func (ctx *__lowercase) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__lowercase) x() any {
    var res []Value
    for _, a := range ctx.a {
        switch t := a.(type) {
        case interface{ change(func(string) string) Value }:
            a = t.change(strings.ToLower)
        default:
            a = _rw(a.Pos(), strings.ToLower(__string(ctx, a)))
        }
        res = append(res, a)
    }
    return res
}

type __trim struct { builtinbase }
func (ctx *__trim) inner() Context { return &ctx.builtinbase }
func (ctx *__trim) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trim) x() any {
    var res []Value
    var cutset string
    var f func(string) string
    if cutset == "" {
        f = strings.TrimSpace
    } else {
        f = func(s string) string { return strings.Trim(s, cutset) }
    }
    for _, a := range merge(ctx.a...) {
        switch t := a.(type) {
        case interface{ change(func(string) string) Value }:
            a = t.change(f)
        default:
            a = _rw(a.Pos(), f(__string(ctx, a)))
        }
        res = append(res, a)
    }
    return res
}

type __trimleft struct { builtinbase }
func (ctx *__trimleft) inner() Context { return &ctx.builtinbase }
func (ctx *__trimleft) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trimleft) x() any {
    var res []Value
    var cutset string
    var f func(string) string
    if cutset == "" {
        f = func(s string) string { return strings.TrimLeftFunc(s, unicode.IsSpace) }
    } else {
        f = func(s string) string { return strings.TrimLeft(s, cutset) }
    }
    for _, a := range ctx.a {
        switch t := a.(type) {
        case interface{ change(func(string) string) Value }:
            a = t.change(f)
        default:
            a = _rw(a.Pos(), f(__string(ctx, a)))
        }
        res = append(res, a)
    }
    return res
}

type __trimright struct { builtinbase }
func (ctx *__trimright) inner() Context { return &ctx.builtinbase }
func (ctx *__trimright) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trimright) x() any {
    var res []Value
    var cutset string
    var f func(string) string
    if cutset == "" {
        f = func(s string) string { return strings.TrimRightFunc(s, unicode.IsSpace) }
    } else {
        f = func(s string) string { return strings.TrimRight(s, cutset) }
    }
    for _, a := range ctx.a {
        switch t := a.(type) {
        case interface{ change(func(string) string) Value }:
            a = t.change(f)
        default:
            a = _rw(a.Pos(), f(__string(ctx, a)))
        }
        res = append(res, a)
    }
    return res
}

// $(trim-prefix foo%, fooxxx foo123)
// $(trim-prefix %/foo, xxx/foo/a/b/c)
// $(trim-prefix %%/foo, xxx/yyy/zzz/foo/a/b/c)
type __trimprefix struct { builtinbase }
func (ctx *__trimprefix) inner() Context { return &ctx.builtinbase }
func (ctx *__trimprefix) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trimprefix) x() any {
	var res []Value
	var prefixes = merge(expand(ctx, ctx.a[0]))

	for _, val := range merge(expands(ctx, ctx.a[1:]...)...) {
		var remainder Value = val

		for _, prefix := range prefixes {
			matched, _, rem, _ := match(ctx, prefix, remainder)

			if matched { // The prefix pattern was completely satisfied!
				if rem == nil || isEmpty(rem) {
					remainder = _null(val.Pos()) // Fully consumed target
					break // Nothing left to trim, safe to break
				} else {
					remainder = rem // Prefix trimmed, keep the rest for chaining!
				}
			}
		}

		res = append(res, remainder)
	}
	return res
}

type __trimsuffix struct { builtinbase }
func (ctx *__trimsuffix) inner() Context { return &ctx.builtinbase }
func (ctx *__trimsuffix) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trimsuffix) x() any {
	var res []Value
	var suffixes = merge(expand(ctx, ctx.a[0]))

	for _, val := range merge(expands(ctx, ctx.a[1:]...)...) {
		var remainder Value = val

		for _, suffix := range suffixes {
			matched, _, rem, _ := match(reversal{ctx}, suffix, remainder)

			if matched { // The suffix pattern was completely satisfied!
				if rem == nil || isEmpty(rem) {
					remainder = _null(val.Pos()) // Fully consumed target
					break // Nothing left to trim, safe to break
				} else {
					remainder = rem // Suffix trimmed, keep the rest for chaining!
				}
			}
		}

		res = append(res, remainder)
	}
	return res
}

type __trimext struct { __trim
    all bool `all`
    ext []string `ext`
}
func (ctx *__trimext) inner() Context { return &ctx.builtinbase }
func (ctx *__trimext) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trimext) x() any {
	var res []Value
	var patterns []Value
	var values []Value

	// 1. Parse arguments: optional explicit extension patterns vs auto-detect
	if len(ctx.a) > 1 {
		patterns = merge(expand(ctx, ctx.a[0]))
		for _, a := range ctx.a[1:] {
			values = append(values, merge(expands(ctx, a)...)...)
		}
	} else {
		for _, a := range ctx.a {
			values = append(values, merge(expands(ctx, a)...)...)
		}
	}

	// 2. Process values
	for _, val := range values {
		var currentVal = val

		// Loop for recursive trimming (if ctx.all is set, e.g., tar.gz -> tar -> "")
		for {
			var matched bool
			var remainder Value = currentVal

			if len(patterns) > 0 {
				// A. Explicit Pattern Mode (behaves like trim-suffix)
				for _, pat := range patterns {
					full, r, rem, _ := match(reversal{ctx}, pat, currentVal)
					if full {
						matched = true; remainder = _null(currentVal.Pos()); break
					} else if r != nil {
						matched = true; remainder = rem; break
					}
				}
			} else {
				// B. Auto-Detect Mode (uses filepath.Ext logic)
				// We must peek at the string representation to find the extension.
				// This is safe because we only stringify to *find* the extension,
				// then use match() to *remove* it, preserving the AST structure of the prefix.
				str := quickStr(ctx, currentVal)
				if ext := filepath.Ext(str); ext != "" {
					// Construct a raw pattern from the detected extension
					pat := _rw(currentVal.Pos(), ext)

					full, r, rem, _ := match(reversal{ctx}, pat, currentVal)
					if full {
						matched = true; remainder = _null(currentVal.Pos())
					} else if r != nil {
						matched = true; remainder = rem
					}
				}
			}

			if matched {
				currentVal = remainder
				if ctx.all { continue } // Repeat if --all flag is active
			}
			break // Stop if no match or not recursive
		}
		res = append(res, currentVal)
	}
	return res
}

type __gitdir struct { builtinbase }
func (ctx *__gitdir) inner() Context { return &ctx.builtinbase }
func (ctx *__gitdir) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__gitdir) x() (_ any) {
    var vals []Value
    for _, a := range merge(ctx.a...) {
        var s = __string(ctx, a)
        if !strings.HasSuffix(s, "/.git") {
            s = filepath.Join(s, ".git")
        }
        if i, e := os.Stat(s); e != nil {
            a = _pathStr(pc(ctx, a), s) // CRITICAL FIX
        } else if m := i.Mode(); m.IsDir() {
            a = _pathStr(pc(ctx, a), s) // CRITICAL FIX
        } else if m.IsRegular() {
            if b, e := ioutil.ReadFile(s); e != nil {
                erro(ctx, "%v", e)
            } else if !bytes.HasPrefix(b, []byte("gitdir:")) {
                erro(ctx, "%s", b)
            } else {
                t := string(bytes.TrimSpace(b[7:]))
                s = filepath.Join(filepath.Dir(s), t)
                a = _pathStr(pc(ctx, a), s) // CRITICAL FIX
            }
        } else {
            erro(pc(ctx,a), "%v", s)
        }
        vals = append(vals, a)
    }
    return vals
}

type __addprefix struct { builtinbase }
func (ctx *__addprefix) inner() Context { return &ctx.builtinbase }
func (ctx *__addprefix) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__addprefix) x() any {
	if len(ctx.a) < 2 {
		return nil
	}

	var res []Value
	var prefixes = merge(expand(ctx, ctx.a[0]))
	for _, w := range merge(expands(ctx, ctx.a[1:]...)...) {
		if !isEmpty(w) {
			for _, p := range prefixes {
				// CRITICAL: Apply redis to the FINAL combined AST!
				res = append(res, redis(prefix(ctx, p, w)))
			}
		}
	}
	return res
}

type __addsuffix struct { builtinbase }
func (ctx *__addsuffix) inner() Context { return &ctx.builtinbase }
func (ctx *__addsuffix) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__addsuffix) x() any {
	if len(ctx.a) < 2 {
		return nil
	}

	var res []Value
	var suffixes = merge(expand(ctx, ctx.a[0]))
	for _, w := range merge(expands(ctx, ctx.a[1:]...)...) {
		if !isEmpty(w) {
			for _, s := range suffixes {
				// CRITICAL: Apply redis to the FINAL combined AST!
				res = append(res, redis(prefix(ctx, w, s)))
			}
		}
	}
	return res
}

type __print struct{ builtinbase
	noErrs bool `noerrs,noerrors,no-errs,no-errors`
	noWarn bool `nowarn,nowarns,no-warn,no-warns`
	f string `...`
}
func (ctx *__print) inner() Context { return &ctx.builtinbase }
func (ctx *__print) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__print) x() (_ any) {
    if ctx.noErrs && 0 < diagCount(ctx, diagError) { return }
    if ctx.noWarn && 0 < diagCount(ctx, diagWarn)  { return }

    var sb bytes.Buffer
    var x = len(ctx.a)
    for i, a := range ctx.a {
        if a == nil { continue }
        if 0 < i && i < x { fmt.Fprintf(&sb, " ") }
        fmt.Fprintf(&sb, "%s", escapedString(ctx, a))
    }
    prompt(ctx, sb.String())
    return
}

type __printf struct{ builtinbase }
func (ctx *__printf) inner() Context { return &ctx.builtinbase }
func (ctx *__printf) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__printf) x() (_ any) {
    if len(ctx.a) < 1 {
        erro(ctx, "not enough args, try $(printf 'format', ...)")
    }

    var vals = merge(ctx.a[0])
    if len(vals) != 1 {
        erro(ctx, "not enough args, try $(printf 'format', ...)")
    }

    var i int
    var a []any
    var f = __string(ctx, vals[0])

outer:
    for _, v := range merge(ctx.a[1:]...) {
    fmtloop:
        for i < len(f) {
            if f[i] != '%' { i += 1; continue }
            for i += 1; i < len(f); i += 1 {
                switch f[i] {
                case '%': continue fmtloop
                case '+', '-', '#', ' ', '.', '0', '1', '2', '3',
                    '4', '5', '6', '7', '8', '9': continue
                case 'c', 'd', 'o', 'O', 'q', 'U':
                    a = append(a, __int(ctx, v))
                    continue outer
                case 'e', 'E', 'f', 'F', 'g', 'G':
                    a = append(a, __float(ctx, v))
                    continue outer
                case 'b', 'x', 'X':
                    switch k := v.kind(); {
                    case k&KindInteger != 0:
                        a = append(a, __int(ctx, v))
                        continue outer
                    case k&KindFloat != 0:
                        a = append(a, __float(ctx, v))
                        continue outer
                    default:
                        if t, e := strconv.Atoi(__string(ctx, v)) ; e == nil { a = append(a, t) } else {
                            erro(ctx, "%v: %v", v, e)
                        }
                        continue outer
                    }
                case 'v':
                    a = append(a, v/* .string(ctx) */)
                    continue outer
                case 't', 'T':
                    a = append(a, v)
                    continue outer
                }
            }
        }
    }
    return fmt.Sprintf(f, a...)
}

type __indent struct { builtinbase }
func (ctx *__indent) inner() Context { return &ctx.builtinbase }
func (ctx *__indent) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__indent) x() (res any) {
    var l []Value
    var s string // indent
    if x := len(ctx.a); x > 0 {
        if v, ok := scalarize(ctx.a[0]).(*decimal); ok {
            ctx.a, s = ctx.a[1:], strings.Repeat(" ", int(v.int64))
        } else {
            erro(ctx, "requires integer argument (first|last)")
        }
    }
    for _, a := range ctx.a {
        var lines []string
        for _, line := range strings.Split(__string(ctx, a), "\n") {
            lines = append(lines, s + line)
        }
        l = append(l, _strlit(a.Pos(), strings.Join(lines, "\n")))
    }
    return l
}

type __findstring struct { builtinbase }
func (ctx *__findstring) inner() Context { return &ctx.builtinbase }
func (ctx *__findstring) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__findstring) x() (res any) {
    // TODO: $(findstring find,text)
    return
}

// $(contains a b c, v1 v2 …)
// $(contains a b c1 -or c2, v1 v2 …)          -- xx
// $(contains a b c1 -or c2 -or c3, v1 v2 …)   -- xx
// $(contains a b -or=(c1 c2 c3), v1 v2 …)     -- xx
type __contains struct { builtinbase
    match  bool `match,pat,pattern`
    string bool `str,string`
}
func (ctx *__contains) inner() Context { return &ctx.builtinbase }
func (ctx *__contains) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__contains) x() (_ any) {
    var list = xmerge(ctx, ctx.a[1:]...)
	for _, val := range xmerge(ctx, ctx.a[0]) {
		var s string
		if ctx.string { s = __string(ctx, val) }

		for _, elem := range list {
			var t bool
			if ctx.match || patterned(ctx, val) {
				t, _, _, _ = match(swapped_ctx{ctx}, val, elem)
			} else if ctx.string {
				t = __string(ctx, elem) == s
			} else {
				t = cmp(ctx, val, elem) == cmpEqual
			}

			if t {
				// ANY semantics: Return truthy immediately upon the first intersection!
				return _loc(toBool(elem, true), _pos(ctx))
			}
		}
	}
	return
}

// $(sort(-unique -reverse) list...)
// Sorts the AST elements lexically based on their string representation/glob rank.
type __sort struct { builtinbase
	reverse bool `reverse`
	unique  bool `unique`
}
func (ctx *__sort) inner() Context { return &ctx.builtinbase }
func (ctx *__sort) cast(t reflect.Type) Context {
	if reflect.TypeOf(ctx) == t { return ctx }
	return ctx.builtinbase.cast(t)
}
func (ctx *__sort) DEPRECATED_x() (res any) {
	var vals []Value
	f := func(a, b Value) int { return int(cmp(ctx, a, b)) }
	for _, v := range merge(ctx.a...) {
		var i, found = slices.BinarySearchFunc(vals, v, f)
		if !ctx.unique || (ctx.unique && !found) { vals = slices.Insert(vals, i, v) }
	}
	return vals
}
func (ctx *__sort) x() (res any) {
	// 1. Collect all items flat (O(N) time)
	vals := merge(ctx.a...)

	// 2. High-speed in-place sort (O(N log N) time)
	slices.SortFunc(vals, func(a, b Value) int {
		return int(cmp(ctx, a, b))
	})

	// 3. High-speed contiguous deduplication (O(N) time)
	if ctx.unique {
		vals = slices.CompactFunc(vals, func(a, b Value) bool {
			// CompactFunc drops adjacent elements that return true
			return cmp(ctx, a, b) == cmpEqual
		})
	}
	return vals
}

// $(wordlist start, end, list...)
// Returns the slice of the list from index 's' to 'e' (0-based, exclusive end like Go/Python).
type __wordlist struct { builtinbase }
func (ctx *__wordlist) inner() Context { return &ctx.builtinbase }
func (ctx *__wordlist) cast(t reflect.Type) Context {
	if reflect.TypeOf(ctx) == t { return ctx }
	return ctx.builtinbase.cast(t)
}
func (ctx *__wordlist) x() (res any) {
	if len(ctx.a) < 2 {
		erro(ctx, "wordlist requires indices: $(wordlist start, end, list...)")
		return
	}

	s := int(__int(ctx, expand(ctx, ctx.a[0])))
	e := int(__int(ctx, expand(ctx, ctx.a[1])))
	vals := xmerge(ctx, ctx.a[2:]...)

	// Intuitive, safe clamping for out-of-bounds indices
	if s < 0 { s = 0 }
	if e > len(vals) { e = len(vals) }
	if s >= e || s >= len(vals) { return } // Native empty return

	return vals[s:e]
}

// $(words list...)
// Returns the number of elements in the AST list.
type __words struct { builtinbase }
func (ctx *__words) inner() Context { return &ctx.builtinbase }
func (ctx *__words) cast(t reflect.Type) Context {
	if reflect.TypeOf(ctx) == t { return ctx }
	return ctx.builtinbase.cast(t)
}
func (ctx *__words) x() (res any) {
	// ease() natively handles converting int64 to the proper AST numeric node.
	return int64(len(xmerge(ctx, ctx.a...)))
}

// $(firstword list...)
// Returns the first element of the list.
type __firstword struct { builtinbase }
func (ctx *__firstword) inner() Context { return &ctx.builtinbase }
func (ctx *__firstword) cast(t reflect.Type) Context {
	if reflect.TypeOf(ctx) == t { return ctx }
	return ctx.builtinbase.cast(t)
}
func (ctx *__firstword) x() (res any) {
	vals := xmerge(ctx, ctx.a...)
	if len(vals) > 0 {
		return vals[0]
	}
	return
}

// $(lastword list...)
// Returns the last element of the list.
type __lastword struct { builtinbase }
func (ctx *__lastword) inner() Context { return &ctx.builtinbase }
func (ctx *__lastword) cast(t reflect.Type) Context {
	if reflect.TypeOf(ctx) == t { return ctx }
	return ctx.builtinbase.cast(t)
}
func (ctx *__lastword) x() (res any) {
	vals := xmerge(ctx, ctx.a...)
	if n := len(vals); n > 0 {
		return vals[n-1]
	}
	return
}

type __encodebase64 struct { builtinbase }
func (ctx *__encodebase64) inner() Context { return &ctx.builtinbase }
func (ctx *__encodebase64) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__encodebase64) x() (res any) {
	if 0 < len(ctx.a) {
		pos := _pos(ctx)
		buf := new(bytes.Buffer)
		enc := base64.NewEncoder(base64.StdEncoding, buf)
		for _, a := range ctx.a { enc.Write([]byte(__string(ctx, a))) }
		enc.Close()
		res = _strlit(pos, buf.String())
	}
	return
}

type __decodebase64 struct { builtinbase }
func (ctx *__decodebase64) inner() Context { return &ctx.builtinbase }
func (ctx *__decodebase64) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__decodebase64) x() (_ any) {
    if 0 < len(ctx.a) {
        var res []Value
        for _, a := range ctx.a {
            var s = __string(ctx, a)
            if dat, err := base64.StdEncoding.DecodeString(s); err != nil {
                erro(ctx, "decode '%s' failed: %v", s, err)
            } else {
                res = append(res, _strlit(a.Pos(), string(dat)))
            }
        }
        return ease(ctx, res)
    }
    return
}

type __ext struct { builtinbase }
func (ctx *__ext) inner() Context { return &ctx.builtinbase }
func (ctx *__ext) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__ext) x() (_ any) {
    var res []Value
    for _, a := range merge(ctx.a...) {
        res = append(res, _strlit(a.Pos(), filepath.Ext(__string(ctx, a))))
    }
    if 0 < len(res) { return res }
    return
}

func _bases(n int, s string, a ...any) (d, b string) {
	d, b = filepath.Dir(s), filepath.Base(s)
	if d == "" || d == "." {
		return
	}

	var k = 1

	// 1. Process the explicit depth 'n' unconditionally
	for i := n - k; i > 0; i -= 1 {
		if d == "/" || d == "." || d == "" { break }
		b = filepath.Join(filepath.Base(d), b)
		d = filepath.Dir(d)
		k += 1
	}

	// 2. Process optional string boundaries or boolean prefixing
	if a != nil {
		for _, arg := range a {
			switch t := arg.(type) {
			case bool:
				if filepath.IsAbs(d) {
					b = filepath.Join(d, b)
				} else if t {
					b = filepath.Join("…", b)
				}
			case string:
				for d != "/" && d != "." && d != "" && len(d)+len(b) < len(s) {
					k += 1
					if base := filepath.Base(d); base == t {
						d = filepath.Dir(d)
						break
					} else {
						b = filepath.Join(base, b)
						d = filepath.Dir(d)
					}
				}
			}
		}
	}
	return
}

func bases(n int, s string, a ...any) (b string) {
	_, b = _bases(n, s, a...)
	return
}

type __bases struct { builtinbase ; n int `num,size,count` }
func (ctx *__bases) inner() Context { return &ctx.builtinbase }
func (ctx *__bases) cast(t reflect.Type) Context {
	if reflect.TypeOf(ctx) == t { return ctx }
	return ctx.builtinbase.cast(t)
}
func (ctx *__bases) x() any {
	var vals []Value
	for _, a := range ctx.a {
		var s string

		// 1. Enter the String Domain: We need raw strings for path math
		if ctx.fullname {
			s, _ = as_fullname_string(ctx, a)
		} else {
			s = __string(ctx, a)
		}

		if s != "" {
			_, s = _bases(ctx.n, s)
			switch t := strings.Split(s, pathSep); len(t) {
			case 0:
			case 1:
				// 2. Cross Back: Intern the resulting string into a Symbol!
				vals = append(vals, _word(a.Pos(), intern(s)))
			default:
				var p = new(path)
				for _, part := range t {
					// 3. Cross Back: Intern each path segment into a Symbol!
					p.elems = append(p.elems, _word(a.Pos(), intern(part)))
				}
				vals = append(vals, p)
			}
		}
	}
	return vals
}

type __base1 struct { __bases }
func (ctx *__base1) inner() Context { return &ctx.__bases }
func (ctx *__base1) cast(t reflect.Type) Context {
	if reflect.TypeOf(ctx) == t { return ctx }
	return ctx.__bases.cast(t)
}
func (ctx *__base1) x() any { ctx.n = 1; return ctx.__bases.x() }

type __base2 struct { __bases }
func (ctx *__base2) inner() Context { return &ctx.__bases }
func (ctx *__base2) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base2) x() any { ctx.n = 2; return ctx.__bases.x() }

type __base3 struct { __bases }
func (ctx *__base3) inner() Context { return &ctx.__bases }
func (ctx *__base3) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base3) x() any { ctx.n = 3; return ctx.__bases.x() }

type __base4 struct { __bases }
func (ctx *__base4) inner() Context { return &ctx.__bases }
func (ctx *__base4) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base4) x() any { ctx.n = 4; return ctx.__bases.x() }

type __base5 struct { __bases }
func (ctx *__base5) inner() Context { return &ctx.__bases }
func (ctx *__base5) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base5) x() any { ctx.n = 5; return ctx.__bases.x() }

type __base6 struct { __bases }
func (ctx *__base6) inner() Context { return &ctx.__bases }
func (ctx *__base6) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base6) x() any { ctx.n = 6; return ctx.__bases.x() }

type __base7 struct { __bases }
func (ctx *__base7) inner() Context { return &ctx.__bases }
func (ctx *__base7) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base7) x() any { ctx.n = 7; return ctx.__bases.x() }

type __base8 struct { __bases }
func (ctx *__base8) inner() Context { return &ctx.__bases }
func (ctx *__base8) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base8) x() any { ctx.n = 8; return ctx.__bases.x() }

type __base9 struct { __bases }
func (ctx *__base9) inner() Context { return &ctx.__bases }
func (ctx *__base9) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base9) x() any { ctx.n = 9; return ctx.__bases.x() }

func dirs(n int, s string) (_ string) {
    for n > 0 {
        s = filepath.Dir(s)
        n -= 1
    }
    return s
}

type __dir struct { __dirs ; sub Value `has,contain,contains` }
func (ctx *__dir) inner() Context { return &ctx.__dirs }
func (ctx *__dir) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir) x() (_ any) {
    var sub string
    if ctx.sub != nil {
        sub = __string(ctx, ctx.sub)
    }
    if sub == "" {
        ctx.n = 1
        return ctx.__dirs.x()
    }

    var l []Value
    for _, a := range merge(ctx.a...) {
        var s string
        if ctx.fullname {
            s, _ = as_fullname_string(ctx, a)
        } else {
            s = __string(ctx, a)
        }
        for {
            var d = filepath.Dir(s)
            if d == "" || d == s { break } else { s = d }
            if _, e := os.Stat(filepath.Join(d,sub)); e == nil {
                l = append(l, _pathStr(pc(ctx, a), d)) // CRITICAL FIX
                break
            }
        }
    }
    return l
}

type __dirs struct { builtinbase ; n int `num,size,count` }
func (ctx *__dirs) inner() Context { return &ctx.builtinbase }
func (ctx *__dirs) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__dirs) x() any {
	var l []Value
	for _, a := range merge(ctx.a...) {
		var s string

		if ctx.fullname {
			s, _ = as_fullname_string(ctx, a)
		} else {
			s = __string(ctx, a)
		}

		s = dirs(ctx.n, s)

		// 1. DUMP THE _stat!
		// 2. Everything is just a path representation until the traversal engine
		//    actually DEMANDS a file!
		if s != "" {
			l = append(l, _pathStr(pc(ctx, a), s))
		}
	}
	return l
}

type __dir1 struct { __dirs }
func (ctx *__dir1) inner() Context { return &ctx.__dirs }
func (ctx *__dir1) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir1) x() any { ctx.n = 1; return ctx.__dirs.x() }

type __dir2 struct { __dirs }
func (ctx *__dir2) inner() Context { return &ctx.__dirs }
func (ctx *__dir2) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir2) x() any { ctx.n = 2; return ctx.__dirs.x() }

type __dir3 struct { __dirs }
func (ctx *__dir3) inner() Context { return &ctx.__dirs }
func (ctx *__dir3) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir3) x() any { ctx.n = 3; return ctx.__dirs.x() }

type __dir4 struct { __dirs }
func (ctx *__dir4) inner() Context { return &ctx.__dirs }
func (ctx *__dir4) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir4) x() any { ctx.n = 4; return ctx.__dirs.x() }

type __dir5 struct { __dirs }
func (ctx *__dir5) inner() Context { return &ctx.__dirs }
func (ctx *__dir5) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir5) x() any { ctx.n = 5; return ctx.__dirs.x() }

type __dir6 struct { __dirs }
func (ctx *__dir6) inner() Context { return &ctx.__dirs }
func (ctx *__dir6) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir6) x() any { ctx.n = 6; return ctx.__dirs.x() }

type __dir7 struct { __dirs }
func (ctx *__dir7) inner() Context { return &ctx.__dirs }
func (ctx *__dir7) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir7) x() any { ctx.n = 7; return ctx.__dirs.x() }

type __dir8 struct { __dirs }
func (ctx *__dir8) inner() Context { return &ctx.__dirs }
func (ctx *__dir8) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir8) x() any { ctx.n = 8; return ctx.__dirs.x() }

type __dir9 struct { __dirs }
func (ctx *__dir9) inner() Context { return &ctx.__dirs }
func (ctx *__dir9) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir9) x() any { ctx.n = 9; return ctx.__dirs.x() }

type __undirs struct { builtinbase ; n int `num,size,count` }
func (ctx *__undirs) inner() Context { return &ctx.builtinbase }
func (ctx *__undirs) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__undirs) x() any {
    var l []Value
    for _, a := range ctx.a {
        var s string
        if ctx.fullname {
            s, _ = as_fullname_string(ctx, a)
        } else {
            s = __string(ctx, a)
        }
        var v = strings.Split(s, pathSep)
        if i := len(v); i == 0 {
            // v is empty
        } else if ctx.n < i {
            v = v[ctx.n:]
        } else {
            v = v[i-1:] // empty
        }
        l = append(l, _pathStr(pc(ctx, a), filepath.Join(v...))) // CRITICAL FIX
    }
    return l
}

type __undir1 struct { __undirs }
func (ctx *__undir1) inner() Context { return &ctx.__undirs }
func (ctx *__undir1) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir1) x() any { ctx.n = 1; return ctx.__undirs.x() }

type __undir2 struct { __undirs }
func (ctx *__undir2) inner() Context { return &ctx.__undirs }
func (ctx *__undir2) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir2) x() any { ctx.n = 2; return ctx.__undirs.x() }

type __undir3 struct { __undirs }
func (ctx *__undir3) inner() Context { return &ctx.__undirs }
func (ctx *__undir3) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir3) x() any { ctx.n = 3; return ctx.__undirs.x() }

type __undir4 struct { __undirs }
func (ctx *__undir4) inner() Context { return &ctx.__undirs }
func (ctx *__undir4) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir4) x() any { ctx.n = 4; return ctx.__undirs.x() }

type __undir5 struct { __undirs }
func (ctx *__undir5) inner() Context { return &ctx.__undirs }
func (ctx *__undir5) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir5) x() any { ctx.n = 5; return ctx.__undirs.x() }

type __undir6 struct { __undirs }
func (ctx *__undir6) inner() Context { return &ctx.__undirs }
func (ctx *__undir6) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir6) x() any { ctx.n = 6; return ctx.__undirs.x() }

type __undir7 struct { __undirs }
func (ctx *__undir7) inner() Context { return &ctx.__undirs }
func (ctx *__undir7) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir7) x() any { ctx.n = 7; return ctx.__undirs.x() }

type __undir8 struct { __undirs }
func (ctx *__undir8) inner() Context { return &ctx.__undirs }
func (ctx *__undir8) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir8) x() any { ctx.n = 8; return ctx.__undirs.x() }

type __undir9 struct { __undirs }
func (ctx *__undir9) inner() Context { return &ctx.__undirs }
func (ctx *__undir9) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir9) x() any { ctx.n = 9; return ctx.__undirs.x() }

type __chopdir struct { builtinbase }
func (ctx *__chopdir) inner() Context { return &ctx.builtinbase }
func (ctx *__chopdir) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__chopdir) x() (res any) {
    var l []Value
    var n = 0
    if x := len(ctx.a); x > 0 {
        if v, ok := scalarize(ctx.a[0]).(*decimal); ok {
            ctx.a, n = ctx.a[1:], int(v.int64)
        } else if v, ok := scalarize(ctx.a[x-1]).(*decimal); ok {
            ctx.a, n = ctx.a[:x-1], int(v.int64)
        } else {
            erro(ctx, "require (first/last) integer argument (first=%T, last=%T)", ctx.a[0], ctx.a[x-1])
            return
        }
    }
    for _, a := range ctx.a {
        var v = strings.Split(__string(ctx, a), pathSep)
        if i := len(v); 0 < i {
            if n < 0 { n = i + n }
            if 0 <= n && n+1 < i {
                v = append(v[0:n], v[n+1:]...)
            } else {
                v = append(v[0:n])
            }
            if len(v) > 0 && v[0] == "" {
                v[0] = pathSep // for absolute paths
            }
        }
        l = append(l, _strlit(a.Pos(), filepath.Join(v...)))
    }
    return l
}

type __reldir struct { builtinbase }
func (ctx *__reldir) inner() Context { return &ctx.builtinbase }
func (ctx *__reldir) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__reldir) x() (res any) {
    var err error
    var l []Value
    var t string
    for i, a := range ctx.a {
        if s := __string(ctx, a); i == 0 {
            t = s
        } else if s, err = filepath.Rel(t, s); err == nil {
            l = append(l, _strlit(a.Pos(), s))
        } else {
            debug(ctx, "%v", err)
        }
    }
    return l
}

type __mkdir struct { builtinbase
    all bool `all,p,path`
}
func (ctx *__mkdir) inner() Context { return &ctx.builtinbase }
func (ctx *__mkdir) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__mkdir) x() (res any) {
    for i, nargs := 0, len(ctx.a); i < nargs; i += 1 {
        var (
            a = ctx.a[i]
            perm = os.FileMode(0755)
            name string
        )
        switch t := a.(type) {
        case *pair: // mkdir name ⇒ perm name ⇒ perm
            name = __string(ctx, t.key)
            perm = filePerm(ctx, t.val, uint32(perm))
        case *group: // mkdir (name perm) (name perm)
            if t.len() == 2 {
                name = __string(ctx, t.at(0))
                perm = filePerm(ctx, t.at(1), uint32(perm))
            } else {
                erro(ctx, "Wrong size of list `%v'", t)
            }
        case *list: // mkdir name perm, name perm, ...
            if t.len() == 2 {
                name = __string(ctx, t.at(0))
                perm = filePerm(ctx, t.at(1), uint32(perm))
            } else {
                erro(ctx, "Wrong size of list `%v'", t)
            }
        default: // mkdir name perm, name perm, ...
            name = __string(ctx, ctx.a[i])
            if i+1 < nargs {
                perm = filePerm(ctx, ctx.a[i+1], uint32(perm))
                i += 1
            }
        }
        if ctx.all {
            if err := os.MkdirAll(name, perm); err != nil {
                erro(ctx, "%v", err)
            }
        } else {
            if err := os.Mkdir(name, perm); err != nil {
                erro(ctx, "%v", err)
            }
        }
    }
    return
}

type __chdir struct { builtinbase }
func (ctx *__chdir) inner() Context { return &ctx.builtinbase }
func (ctx *__chdir) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__chdir) x() (res any) {
    if len(ctx.a) == 1 {
        var str = __string(ctx, ctx.a[0])
        if err := lockCD(str, 0); err != nil {
            erro(ctx, "%v", err)
        }
    } else {
        debug(ctx, "wrong number of arguments: %v", len(ctx.a))
    }
    return
}

type __rename struct { builtinbase }
func (ctx *__rename) inner() Context { return &ctx.builtinbase }
func (ctx *__rename) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__rename) x() (res any) {
    for i, nargs := 0, len(ctx.a); i < nargs; i += 1 {
        var (
            a = ctx.a[i]
            oldname, newname string
        )
        switch t := a.(type) {
        case *pair: // rename oldname=newname
            oldname = __string(ctx, t.key)
            newname = __string(ctx, t.val)
        case *group: // rename (oldname newname) (old new)
            if t.len() == 2 {
                oldname = __string(ctx, t.at(0))
                newname = __string(ctx, t.at(1))
            } else {
                erro(ctx, "wrong size of group `%v'", t)
            }
        case *list: // rename oldname newname, old new, ...
            if t.len() == 2 {
                oldname = __string(ctx, t.at(0))
                newname = __string(ctx, t.at(1))
            } else {
                erro(ctx, "wrong size of list `%v'", t)
            }
        default: // rename newname oldname  newname oldname ...
            if i+1 < nargs {
                oldname = __string(ctx, ctx.a[i+0])
                newname = __string(ctx, ctx.a[i+1])
                i += 1
            } else {
                erro(ctx, "Wrong arguments `%v'", ctx.a)
            }
        }
        if err := os.Rename(oldname, newname); err != nil {
            erro(ctx, "%v", err)
        }
    }
    return
}

type __remove struct { builtinbase
    skip string `skip`
    ignoreMissing bool `ig,ignore,ignore-missing,ignore-not-found`
    warnNotFile bool `warn-not-file`
    all bool `all,recursive`
}
func (ctx *__remove) inner() Context { return &ctx.builtinbase }
func (ctx *__remove) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__remove) x() (res any) {
    var opts = ctx
    var remove func(Context, Value)
    var removeFile = func(ctx Context, f *file) {
        var err error
        var s = f.fullname()
        if opts.skip != "" {
            if strings.HasPrefix(s, opts.skip) { return } else
            if strings.HasPrefix(ident(ctx,f), opts.skip) { return }
        }
        if opts.all { err = os.RemoveAll(s) } else { err = os.Remove(s) }
        if err != nil {
            erro(ctx, _f("remove: %v", err), _f("remove: %v → %s", f, s))
            return
        }
        if d := opts.debug; d>0 { debug(ctx, "remove %s (%s)", f, s, callstack{num:d}) }
        if opts.verbose { prompt(ctx, "removed %s\n", f) }
    }
    var removePath = func(ctx Context, p *path) {
        var err error
        var s = __string(ctx, p)
        if opts.skip != "" {
            if strings.HasPrefix(s, opts.skip) { return }
        }
        if opts.all { err = os.RemoveAll(s) } else {
            erro(ctx, "remove path: %v", p)
            return
        }
        if err != nil {
            erro(ctx, _f("remove: %v", err), _f("remove: %v", p))
            return
        }
        if d := opts.debug; d>0 { debug(ctx, "remove %s", s, callstack{num:d}) }
        if opts.verbose { prompt(ctx, "removed %s\n", s) }
    }
    var removePat = func(ctx Context, pat Value) {
        erro(ctx, "TODO: remove: %v", ts(pat))
    }

    remove = func(ctx Context, v Value) {
        if _, y := v.(*none); y {
            return
        } else if isTrivial(v) {
            debug(ctx, "triviality: %v (%T)", v, v)
        } else if l, y := v.(*list); y {
            for _, v := range l.elems { remove(ctx, v) }
        } else if d, y := v.(*delegate); y {
            debug(ctx, "delegate: %v (%T, %v, %v)", d.x, d.x, d.o, d.a)
        } else if patterned(ctx,v) {
            removePat(ctx, v)
        } else if f, y := v.(*file); y {
            removeFile(ctx, f)
        } else if f = findfile(ctx, __string(ctx, v)); f != nil {
            removeFile(ctx, f)
        } else if p, y := v.(*path); y {
            removePath(ctx, p)
        } else if !opts.ignoreMissing {
            erro(ctx, "not file: %v (%T)", v, v)
        }
    }
    for _, a := range ctx.a {
        ctx := ctx.Context
        remove(ctx, expand(ctx, a))
    }

    if opts.debug > 0 { debug(ctx, "%v", ctx.a) }
    if opts.debug > 0 && flush(ctx) > 0 {
        erro(ctx, "remove errors")
    }
    return
}

type __truncate struct { builtinbase }
func (ctx *__truncate) inner() Context { return &ctx.builtinbase }
func (ctx *__truncate) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__truncate) x() (res any) {
    for i, nargs := 0, len(ctx.a); i < nargs; i += 1 {
        var (
            a = ctx.a[i]
            name string
            size int64
        )
        switch t := a.(type) {
        case *pair: // truncate name ⇒ size old ⇒ new
            name = __string(ctx, t.key)
            size = __int(ctx, t.val)
        case *group: // truncate (name size) (old new)
            if t.len() == 2 {
                name = __string(ctx, t.at(0))
                size = __int(ctx, t.at(1))
            } else {
                erro(ctx, "Wrong size of group `%v'", t)
                break
            }
        case *list: // truncate name size, old new, ...
            if t.len() == 2 {
                name = __string(ctx, t.at(0))
                size = __int(ctx, t.at(1))
            } else {
                erro(ctx, "Wrong size of list `%v'", t)
                break
            }
        default: // truncate name size  name size ...
            if i+1 < nargs {
                name = __string(ctx, ctx.a[i+0])
                size = __int(ctx, ctx.a[i+1])
                i += 1
            } else {
                erro(ctx, "Wrong arguments `%v'", ctx.a)
                break
            }
        }
        if err := os.Truncate(name, size); err != nil {
            erro(ctx, "%v", err)
            break
        }
    }
    return
}

type __link struct { builtinbase }
func (ctx *__link) inner() Context { return &ctx.builtinbase }
func (ctx *__link) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__link) x() (res any) {
    for i, nargs := 0, len(ctx.a); i < nargs; i += 1 {
		var (
			a = ctx.a[i]
			oldname, newname string
		)
        switch t := a.(type) {
        case *pair: // link oldname ⇒ newname old ⇒ new
            oldname = __string(ctx, t.key)
            newname = __string(ctx, t.val)
        case *group: // link (oldname newname) (old new)
            if t.len() == 2 {
                oldname = __string(ctx, t.at(0))
                newname = __string(ctx, t.at(1))
            } else {
                erro(ctx, "Wrong size of group `%v'", t)
                break
            }
        case *list: // link oldname newname, old new, ...
            if t.len() == 2 {
                oldname = __string(ctx, t.at(0))
                newname = __string(ctx, t.at(1))
            } else {
                erro(ctx, "Wrong size of list `%v'", t)
                break
            }
        default: // link oldname newname  oldname newname ...
            if i+1 < nargs {
                oldname = __string(ctx, ctx.a[i+0])
                newname = __string(ctx, ctx.a[i+1])
                i += 1
            } else {
                erro(ctx, "Wrong arguments `%v'", ctx.a)
                break
            }
        }
        if err := os.Link(oldname, newname); err != nil {
            erro(ctx, "%v", err)
            break
        }
    }
    return
}

func _readlink(ctx Context, filename string, d os.FileInfo) (_ string, linked bool) {
    fn, linkpath := filename, filepath.Dir(filename)
    for d.Mode()&os.ModeSymlink != 0 {
        linkname, e := os.Readlink(fn)

        if e != nil {
            prompt(ctx, "%s: readlink failed\n", fn)
            erro(ctx, "%v", e)
            return
        }

        var rel = !filepath.IsAbs(linkname)
        if rel {
            linkname = filepath.Join(linkpath, linkname)
            linkpath = filepath.Dir(linkname)
        }

        if d, e = os.Lstat(linkname); e != nil {
            prompt(ctx, "%s: lstat %s\n", fn, linkname)
            erro(ctx, "%v", e)
            return
        }

        fn, linked = linkname, true
    }
    return fn, linked
}

func readlink(ctx Context, filename string) (_ string, _ bool) {
    if d, e := os.Stat(filename); e == nil {
        return _readlink(ctx, filename, d)
    }
    return
}

/* Example:
   foo: foobar
	symlink -pluv $< $@
*/
type __symlink struct { builtinbase
    path     bool `path`
    force    bool `force,overwrite`
    update   bool `update`
    relative bool `rel,relative`
}
func (ctx *__symlink) inner() Context { return &ctx.builtinbase }
func (ctx *__symlink) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__symlink) x() (res any) {
outer:
	for i, na := 0, len(ctx.a); i < na; i += 1 {
		var (
			opts = *ctx // make a copy
			srcNameVal, dstNameVal Value

			// 1. Symbol Domain (Internal VM)
			srcDirSym, srcNameSym  Symbol
			dstDirSym, dstNameSym  Symbol

			// 2. OS String Domain (Physical Disk)
			srcName   , dstName    string
			srcDir    , dstDir     string

			aa []Value
		)
		switch t := ctx.a[i].(type) {
		case *pair: // symlink srcName=dstName srcName=>dstName...
			srcNameVal, dstNameVal = t.key, t.val
		case *group: // symlink (-u srcName dstName) (-v srcName dstName)...
			if aa = parseOpts(ctx, &opts, t.elems...); len(aa) != 2 {
				erro(ctx, "expects two values for group")
				return
			} else {
				srcNameVal, dstNameVal = aa[0], aa[1]
			}
		case *list: // XXX: symlink old new, old new, ...
			if aa = parseOpts(ctx, &opts, t.elems...); len(aa) != 2 {
				erro(ctx, "expects two values for list")
				return
			} else {
				srcNameVal, dstNameVal = aa[0], aa[1]
			}
		default:// Multiple pairs of names:
			// symlink  new old, new old ...
			// symlink  new old  new old ...
			if i+1 < na {
				srcNameVal = ctx.a[i+0]
				dstNameVal = ctx.a[i+1]
				i += 1
			} else {
				var a = auto_get(ctx,symAt)//"@"
				var l = auto_get(ctx,symLangle)//"<"
				var r = auto_get(ctx,symRangle)//">"
				erro(ctx,
					_f("expects pair of names (%T %v)", t, t),
					_f("symlink: args=%v → %v", ctx.a, t),
					_f("symlink: %v, %v, %v", a, l, r),
				)
				return
			}
		}

		// Use the new allocation-free Symbol helpers
		if srcDirSym, srcNameSym = splitFileName(ctx, srcNameVal); srcNameSym == symEmpty {
			erro(ctx,
				_f("empty src filename (%T)", srcNameVal),
				_f("symlink: src=%v", srcNameVal),
				_f("symlink: args=%v", ctx.a),
			)
			return
		}
		if dstDirSym, dstNameSym = splitFileName(ctx, dstNameVal); dstNameSym == symEmpty {
			erro(ctx,
				_f("empty dest filename (%T)", dstNameVal),
				_f("symlink: dest=%v", dstNameVal),
				_f("symlink: args=%v", ctx.a),
			)
			return
		}

		// --- GATEWAY TO THE OS DOMAIN ---
		// Translate the Symbol holograms back into concrete OS strings for the hard drive
		srcDir, srcName = srcDirSym.String(), srcNameSym.String()
		dstDir, dstName = dstDirSym.String(), dstNameSym.String()

		var src = srcName
		var dst = dstName

		// Everything from here down remains perfectly standard Go OS logic!
		if !filepath.IsAbs(src) { src = filepath.Join(srcDir, srcName) }
		if !filepath.IsAbs(dst) { dst = filepath.Join(dstDir, dstName) }

		if _, err := os.Stat(src); err != nil {
			erro(ctx,
				_f("%v", err),
			)
			return
		}

		if !opts.relative {/* no rel required */} else
		if s, e := filepath.Rel(filepath.Dir(dst), src); e != nil {
			erro(ctx,
				_f("%v", e),
				_f("symlink: %s: rel(%s, %s)\n", dstName, dst, src))
			return
		} else {
			if false {
				debug(ctx,
					_f("%v %v\t%s", srcDir, srcName, src),
					_f("%v %v\t%s", dstDir, dstName, dst),
					_f("%v", s))
			}
			src = s
		}

		if !opts.path {/* no mkdir */} else
		if dstDir == "" || dstDir == "." || dstDir == string(filepath.Separator) {
			// no need to mkdir: . or /
		} else if err := os.MkdirAll(dstDir, os.FileMode(0755)); err != nil {
			erro(ctx, "%v", err)
			return
		}

		var rm bool
		if rm = opts.force; rm {
			// overwrite...
		} else if s, e := os.Readlink(dst); e != nil {
			if false { erro(ctx,
				_f("%v: readlink failed (%T)", dstName, e),
				_f("%v", e))}
		} else if rm = s != src; !rm {
			continue outer
		}

		if rm { if e := os.Remove(dst); e != nil {
			erro(ctx,
				_f("%v: remove old symlink failed (%T)", dstName, e),
				_f("%v", e))
			return
		}}
		if err := os.Symlink(src, dst); err != nil {
			if opts.verbose { prompt(ctx, "… %s\n", err) }
			break
		} else if opts.verbose {
			var d = trimPromptString(dstName)
			var s = filepath.Base(srcName)
			prompt(ctx, "%s → %s …… ok\n", d, s)
		}
	}
	return
}

type __stat struct { builtinbase
    symbol bool `sym,symbol,symlink,link`
    file   bool `file`
    dir    bool `dir`
}
func (ctx *__stat) inner() Context { return &ctx.builtinbase }
func (ctx *__stat) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__stat) x() (res any) {
	if len(ctx.a) == 0 { return }

	var proj = _project(ctx)
	if proj == nil {
		erro(ctx, "unknown project")
		return
	}

	var vals []Value
	var check = func(f *file) {
		if f != nil && f.exists() {
			// Fast Path: O(1) checks directly against the primitive cache
			if (ctx.dir  && f._isDir) || 
			   (ctx.file && !f._isDir) || 
			   (!ctx.dir && !ctx.file && !ctx.symbol) {
				vals = append(vals, f)
				return
			}
			
			// Cold Path: If they explicitly want to test for a symlink, we bypass 
			// the VFS (which resolves links) and ask the OS directly via Lstat.
			if ctx.symbol {
				if fi, err := os.Lstat(f.fullname()); err == nil && fi.Mode()&os.ModeSymlink != 0 {
					vals = append(vals, f)
				}
			}
		}
	}

	var checkstat = func(a Value) {
		var f *file
		var s string
		if s = __string(ctx, a); filepath.IsAbs(s) {
			f = _stat(ctx, s)
		} else {
			f = _stat(ctx, s, proj) // aka stat_dir{proj.absPath}
		}
		if f == nil { f = proj.file(ctx, s) }
		if f != nil { check(f) }
	}

	for _, a := range merge(ctx.a...) {
		switch t := a.(type) {
		case *file: check(t)
		case *path: checkstat(a)
		default:    checkstat(a)
		}
	}
	return vals
}

type __file struct { builtinbase
    exists bool `exist,exists,must,must-exist,required`
    report bool `report,reportmissing,report-missing`
    ignore bool `ignore,ignore-missing,missing,nonexist,non-exist`
}
func (ctx *__file) inner() Context { return &ctx.builtinbase }
func (ctx *__file) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__file) x() any {
	var res []Value
	var proj = _project(ctx)
	for _, a := range merge(ctx.a...) {
		if checkpoints {
			if s := a.String(); strings.Contains(s, "{}") || strings.HasSuffix(s, ".x") {
				debug(pc(ctx,a), _f("%v", a),
					_f("a: %v", _scope(ctx).resolve(intern("a"))),
					_f("s: %v", _scope(ctx).resolve(intern("s"))),
					_f("x: %v", _scope(ctx).resolve(intern("x"))),
					_f("o: %v", _scope(ctx).resolve(intern("o"))),
					_f("@: %v", _scope(ctx).resolve(symAt)),
					callstack{num:16}, trace{})
			}
		}
		if x, y := to_file(a); y {
			if !ctx.exists || x.exists() /* || x.stat(ctx) != nil */ {
				res = append(res, try_fullfile(ctx, x))
			} else if ctx.report {
				debug(ctx, "no such file {%v %v %v}", x.dir, x.sub, x.name)
			}
			continue
		}

		var mapped = select_files(ctx, unmap_files(ctx, proj, a, nil))

		// CRITICAL FIX: If the string maps to no physical files, but the file
		// is not strictly required to exist, forcefully retrieve/create its VFS node!
		if len(mapped) == 0 && !ctx.exists {
			if f := proj.file(ctx, a); f != nil {
				mapped = []*file{f}
			}
		}

		for _, f := range mapped {
			if !ctx.exists || f.exists() {
				res = append(res, try_fullfile(ctx, f))
			} else if ctx.ignore {
				if ctx.verbose { debug(ctx, "%v → %v", ts(a,ctx), f) }
			} else if ctx.exists {
				erro(ctx, `not a file: %v : %s ; %s`, a, ts(a,ctx), ts(res,ctx))
			}
		}
	}
	return res
}

type __glob struct { builtinbase
    symbol bool `sym,symlink,symbol`
    dir bool `dir,directory`
    file bool `file`
}
func (ctx *__glob) inner() Context { return &ctx.builtinbase }
func (ctx *__glob) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__glob) x() (_ any) {
    var cwd string // TODO: get current work directory
    var proj *project
    if proj = _project(ctx); proj == nil {
        erro(ctx, "unknown current cntext")
    }

    var res []Value
    for _, a := range ctx.a {
        var ( str string; names []string )
        if str = __string(ctx, a); !filepath.IsAbs(str) {
            str = filepath.Join(cwd, str)
        }

        var err error
        if names, err = filepath.Glob(str); err != nil {
            erro(ctx, "glob '%v' failed: %v", str, err)
        }
        for _, name := range names {
            // TODO: ctx.dir, ctx.file, ctx.symbol
            res = append(res, _pathStr(ctx, name))
        }
    }
    return res
}

func readDirNames(ctx Context, sd string, errorMissing bool) (names []string) {
	if false {
		if f, err := os.Stat(sd); err != nil {
			if errorMissing {
				erro(ctx, "%v", err)
			}
			return
		} else if !f.IsDir() {
			erro(ctx, "not dir: %v", sd)
			return
		}
	}

	// Use os.ReadDir (Go 1.16+) instead of os.Open + Readdirnames.
	// It is faster and gives us DirEntry, which knows IsDir() without extra syscalls.
	entries, err := os.ReadDir(sd)
	if err != nil {
		if errorMissing { erro(ctx, "readdir: %v", err) }
		return
	}

	names = make([]string, 0, len(entries))

	for _, entry := range entries {
		if name := entry.Name(); entry.IsDir() {
			// Directories heavily repeat (src, bin, pkg, internal). Intern them!
			names = append(names, name)
		} else {
			// Files are often unique artifacts (obj_123.o). Leave them raw to protect GC.
			// Optional: You can explicitly intern safe, known file types here if desired:
			// if strings.HasSuffix(name, ".smart") { name = intern(name) }

			names = append(names, name)
		}
	}
	return
}

// stepPattern advances the AST pattern by one directory level safely.
func stepPattern(ctx Context, pat Value, name string) (nextPats []Value) {
	if l, ok := pat.(*list); ok {
		for _, e := range l.elems {
			nextPats = append(nextPats, stepPattern(ctx, e, name)...)
		}
		return
	}

	if p, ok := pat.(*path); ok {
		if len(p.elems) == 0 { return }
		first := p.elems[0]

		if isMultiWildcard(first) {
			// ** spans directories, so it must remain in the pattern for children
			nextPats = append(nextPats, pat)

			// ** can also consume 0 segments, so we check if the NEXT element matches this directory
			if len(p.elems) > 1 {
				okMatch, _, _, _ := match(ctx, p.elems[1], _rw(p.Pos(), name))
				if okMatch {
					if len(p.elems) > 2 {
						var nextPat Value
						if len(p.elems) == 3 {
							nextPat = p.elems[2]
						} else {
							nextPat = &path{elements{p.elems[2:]}}
						}
						nextPats = append(nextPats, nextPat)
					}
				}
			}
		} else {
			// Standard strict segment match
			okMatch, _, _, _ := match(ctx, first, _rw(p.Pos(), name))
			if okMatch {
				if len(p.elems) > 1 {
					var nextPat Value
					if len(p.elems) == 2 {
						nextPat = p.elems[1]
					} else {
						nextPat = &path{elements{p.elems[1:]}}
					}
					nextPats = append(nextPats, nextPat)
				}
			}
		}
		return
	}

	// Single segment global wildcards (like `**.h` without path wrapping)
	if isMultiWildcard(pat) {
		nextPats = append(nextPats, pat)
	}

	return
}

type __wildcard struct {
	builtinbase
	includeMissing bool    `include,includemissing,include-missing,missing,all`
	ignoreMissing  bool    `ignore,ignoremissing,ignore-missing`
	errorMissing   bool    `err,error,errormissing,error-missing,no-missing`
	exclude        []Value `exclude,except,no,not`
	filetype       string  `type` // dir, file, etc.
	dir            string  `dir,directory`
	sort           bool    `sort`
	cache          bool    `cache` // The explicit caching opt-in flag!
	files          []*file
}

func (ctx *__wildcard) inner() Context { return &ctx.builtinbase }
func (ctx *__wildcard) cast(t reflect.Type) Context {
	if reflect.TypeOf(ctx) == t { return ctx }
	return ctx.builtinbase.cast(t)
}

func (ctx *__wildcard) collect(f *file) {
	if f != nil {
		if ctx.sort {
			i, found := slices.BinarySearchFunc(ctx.files, f, func(a, b *file) int {
				// CRITICAL FIX: Extract the strings to perform alphabetical sorting!
				// Comparing a.name < b.name directly only compares their integer Symbol IDs.
				strA := a.name.String()
				strB := b.name.String()

				switch {
				case strA < strB: return -1
				case strA > strB: return 1
				}
				return 0
			})
			if !found { ctx.files = slices.Insert(ctx.files, i, f) }
		} else {
			ctx.files = append(ctx.files, f)
		}
	}
}

func (ctx *__wildcard) directory(topDir string, pats ...Value) {
	var ne = ctx.includeMissing && !ctx.ignoreMissing

	pats = merge(pats...)

	var walk func(dir string, currentPats []Value)
	walk = func(dir string, currentPats []Value) {
		names := readDirNames(ctx, filepath.Join(topDir, dir), ctx.errorMissing)

	NamesLoop:
		for _, name := range names {
			dn := name
			if dir != "" && dir != "." { dn = joinPathSegs(dir, name) }

			// 1. Exclude Check (Must be an absolute match!)
			dnPath := _pathStr(ctx, dn)
			for _, x := range ctx.exclude {
				if ok, _, rem, _ := match(ctx, x, dnPath); ok && rem == nil {
					continue NamesLoop
				}
			}

			// 2. Absolute Match Check (Determines if THIS path is a collected result)
			matched := false
			for _, pat := range pats {
				full, _, rem, _ := match(ctx, pat, dnPath)
				if full && rem == nil {
					matched = true
					break
				}
			}

			// 3. Step the patterns (Determines ONLY if we should recurse deeper)
			var nextPats []Value
			for _, pat := range currentPats {
				nextPats = append(nextPats, stepPattern(ctx, pat, name)...)
			}

			if len(nextPats) > 1 {
				var unique []Value
				seen := make(map[string]struct{})
				for _, np := range nextPats {
					s := np.String()
					if _, ok := seen[s]; !ok {
						seen[s] = struct{}{}
						unique = append(unique, np)
					}
				}
				nextPats = unique
			}

			var f *file
			var isDir bool
			if matched || len(nextPats) > 0 {
				// Fetch the canonical *file object from the engine
				f = _stat(ctx, dn, stat_dir{topDir}, stat_nonexist{ne})

				if f == nil {
					continue // Invalid file state
				}

				if f._mtime != 0 {
					isDir = f._isDir
				} else if !ne {
					continue // File doesn't exist and we aren't explicitly including missing
				}
			}

			// 4. Collection (Strictly relies on Absolute Match!)
			if matched {
				validType := false
				switch strings.ToLower(ctx.filetype) {
				case "f", "file": validType = !isDir
				case "d", "dir":  validType = isDir
				case "":          validType = true
				default:          validType = true
				}

				if validType { ctx.collect(f) }
			}

			// 5. Recursive Descent
			if isDir && len(nextPats) > 0 {
				walk(dn, nextPats)
			}
		}
	}

	walk("", pats)
}

func (ctx *__wildcard) project(p *project, pats ...Value) {
	var ne = ctx.includeMissing && !ctx.ignoreMissing

	for _, argPat := range pats {
		for _, a := range unmap_files(ctx, p, argPat, nil) {
			for _, mapPat := range a.filemap.patterns(ctx) {
				var search = _if_cmp(ctx, cmpSmaller, argPat, mapPat)
				var isSearchPat = patterned(ctx, search)
				for _, v := range merge(expands(_final(ctx), a.filemap.paths...)...) {
					if dir := __string(ctx, v); isSearchPat {
						ctx.directory(dir, search)
					} else {
						ctx.collect(_stat(ctx, __string(ctx, search), stat_dir{dir}, stat_nonexist{ne}))
					}
				}
			}
		}
	}
}

func (ctx *__wildcard) x() any {
	if len(ctx.exclude) > 0 {
		ctx.exclude = xmerge(_final(ctx.Context), ctx.exclude...)
	}

	pats := merge(ctx.a...)

	// ====================================================================
	// FAST PATH: Explicit Persistent Disk Caching
	// ====================================================================
	var cacheFile string
	var p = _project(ctx)
	var t0 = time.Now()

	if ctx.cache && p != nil {
		// Generate a deterministic hash based on the directory and all search patterns
		var key = sha256.New()
		fmt.Fprintf(key, "dir:%s|type:%s|sort:%v|ex:%v", ctx.dir, ctx.filetype, ctx.sort, ctx.exclude)
		for _, pat := range pats {
			fmt.Fprintf(key, "|pat:%s", __string(ctx, pat))
		}

		var h hashbytes
		copy(h[:], key.Sum(nil))

		// Map to $(outtmp)/.hash/globs/<hash>
		dir := getHashDir(ctx, h[:])
		cacheFile = filepath.Join(dir, "globs", fmt.Sprintf("%x.txt", h))

		// Try to read the cache!
		if b, err := os.ReadFile(cacheFile); err == nil {
			var cachedFiles []Value
			for _, pathStr := range strings.Split(string(b), "\n") {
				if pathStr == "" { continue }

				// Rehydrate the cached string back into an interned *file AST node
				var f *file
				if filepath.IsAbs(pathStr) {
					f = _stat(ctx, pathStr, stat_nonexist{true})
				} else {
					f = _stat(ctx, pathStr, stat_dir{ctx.dir}, stat_nonexist{true})
				}

				if f == nil { f = p.file(ctx, pathStr) }
				if f != nil { cachedFiles = append(cachedFiles, f) }
			}
			if false { debug(pc(ctx,cacheFile), "%v", time.Since(t0)) }
			return cachedFiles // Done! Bypassed all OS traversal!
		}
	}

	// ====================================================================
	// SLOW PATH: OS File System Traversal
	// ====================================================================
	if ctx.dir == "" {
		ctx.project(p, pats...)
	} else {
		ctx.directory(ctx.dir, pats...)
	}

	// ====================================================================
	// WRITE CACHE
	// ====================================================================
	if ctx.cache && cacheFile != "" && len(ctx.files) > 0 {
		if err := os.MkdirAll(filepath.Dir(cacheFile), 0700); err == nil {
			var sb strings.Builder
			var ctxDirSym = intern(ctx.dir)
			for _, f := range ctx.files {
				// Write the relative sub-path if it exists, otherwise full name
				if f.dir == ctxDirSym {
					sb.WriteString(filepath.Join(f.sub.String(), f.name.String()))
				} else {
					sb.WriteString(f.fullname())
				}
				sb.WriteString("\n")
			}
			os.WriteFile(cacheFile, []byte(sb.String()), 0644)
			if false { debug(pc(ctx,cacheFile), "%v", time.Since(t0)) }
		} else {
			erro(ctx, "failed to create cache dir: %v", err)
		}
	}

	return ctx.files
}

type __readdir struct { builtinbase }
func (ctx *__readdir) inner() Context { return &ctx.builtinbase }
func (ctx *__readdir) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__readdir) x() (res any) {
    var l []Value
    for _, a := range ctx.a {
        if fis, err := ioutil.ReadDir(__string(ctx, a)); err == nil {
            v := new(list)
            for _, fi := range fis {
                v.append(_strlit(a.Pos(), fi.Name()))
            }
            l = append(l, v)
        } else {
            break //l = append(l, _none(pos))
        }
    }
    return l
}

type __readfile struct { builtinbase
    trim      bool `ta,trim,trim-all`
    trimLeft  bool `tl,trim-left`
    trimRight bool `tr,trim-right`
}
func (ctx *__readfile) inner() Context { return &ctx.builtinbase }
func (ctx *__readfile) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__readfile) x() (res any) {
    var l []Value
    var closured = closure_projects(ctx)
    for _, v := range ctx.a {
        if o := as_fullname(ctx, v, closured...); o.Value == nil {
            erro(ctx, "%v is not a file", v)
        } else if s, e := ioutil.ReadFile(__string(ctx,o)); e != nil {
            erro(ctx, "read file failed: %v", e)
        } else {
            if ctx.trim      { s = bytes.TrimFunc     (s, unicode.IsSpace) } else
            if ctx.trimLeft  { s = bytes.TrimLeftFunc (s, unicode.IsSpace) } else
            if ctx.trimRight { s = bytes.TrimRightFunc(s, unicode.IsSpace) }
            l = append(l, _strlit(v.Pos(), string(s)))
        }
    }
    return l
}

type __writefile struct { builtinbase
    path bool `path`
}
func (ctx *__writefile) inner() Context { return &ctx.builtinbase }
func (ctx *__writefile) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__writefile) x() (res any) {
    // $(write-file filename,content)
    // $(write-file -p filename,content)
outer:
    for i := 0; i < len(ctx.a); i += 1 {
        var (
            a = ctx.a[i]
            name, data string
            perm = os.FileMode(0600)
        )
        switch t := a.(type) {
        case *pair: // write-file name=text name=text
            name = __string(ctx, t.key)
            data = __string(ctx, t.val)
        case *group: // write-file (name text) (name text 0660)
            if n := t.len(); n < 4 && n > 0 {
                name = __string(ctx, t.at(0))
                if n > 1 { data = __string(ctx, t.at(1)) }
                if n > 2 { perm = filePerm(ctx, t.at(2),0600) }
            } else {
                erro(ctx, "Wrong size of group `%v'", t)
            }
        case *list: // write-file name text, name text 0660, ...
            if n := t.len(); n < 4 && n > 0 {
                name = __string(ctx, t.at(0))
                if n > 1 { data = __string(ctx, t.at(1)) }
                if n > 2 { perm = filePerm(ctx, t.at(2),0600) }
            } else {
                erro(ctx, "Wrong size of list `%v'", t)
            }
        default: // write-file name text 0660  name text 0660 ...
            name = __string(ctx, ctx.a[i])
            if i+1 < len(ctx.a) {
                data = __string(ctx, ctx.a[i+1])
                i += 1
            }
            if i+1 < len(ctx.a) {
                perm = filePerm(ctx, ctx.a[i+1],0600)
                i += 1
            }
        }
        if name == "" {
            continue outer
        } else if dir := filepath.Dir(name); ctx.path && dir != "." && dir != pathSep {
            if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                erro(ctx, "%v", err)
            }
        }
        if err := ioutil.WriteFile(name, []byte(data), perm); err != nil {
            erro(ctx, "%v", err)
        }
    }
    return
}

func touch(ctx Context, file Value, optMode uint32, optPath bool, ts ...time.Time) (err error) {
	var a, filename, c = as_fullname_file(ctx, file)

	if filename == "" {
		erro(ctx, "touch: empty file name: %v (%v, %v, %v)", file, typeof(file), a, c)
	} else if d := filepath.Dir(filename); optPath && d != "." && d != string(filepath.Separator) {
		if err = os.MkdirAll(d, os.FileMode(optMode|0733)); err != nil {
			erro(ctx, "touch: %v", err)
		}
	}

	var (
		mode = os.FileMode(optMode)
		ta, tm time.Time
		m os.FileMode
	)
	if len(ts) > 0 { ta = ts[0] } else { ta = time.Now() }
	if len(ts) > 1 { tm = ts[1] } else { tm = time.Now() }
	
	// COLD PATH: We don't cache os.FileMode anymore, so just pull it straight from the disk
	if fi, e := os.Stat(filename); e == nil && fi != nil {
		m = fi.Mode()
	} else {
		var f *os.File
		if m = mode; m == 0 { m = os.FileMode(0600); mode = m }
		if f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, m&os.ModePerm); err != nil {
			erro(ctx, "touch: %v", err)
		} else if err = f.Close(); err != nil {
			erro(ctx, "touch: %v", err)
		}
	}
	
	if err == nil {
		if err = os.Chtimes(filename, ta, tm); err != nil {
			erro(ctx, "touch: %v", err)
		}
		
		// RE-ENTER THE WALLED GARDEN: 
		// Now that the OS disk is updated, we must instantly sync the physical 
		// timestamp back into our `int64` VFS cache!
		if fi, _ := to_file(file); fi != nil {
			fi.stat(ctx)
		} else if fi := _stat(ctx, filename); fi != nil {
			fi.stat(ctx)
		}
	}
	
	if err == nil && mode != 0 && m != 0 && mode != m {
		if err = os.Chmod(filename, mode); err != nil {
			erro(ctx, "touch: %v", err)
		}
	}
	return
}

type __touchfile struct { builtinbase
    mode os.FileMode `mode`
    path bool `path`
}
func (ctx *__touchfile) inner() Context { return &ctx.builtinbase }
func (ctx *__touchfile) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__touchfile) x() (res any) {
    // $(touch-file filename)
    // $(touch-file -p filename)
    for i := 0; i < len(ctx.a); i += 1 {
        if err := touch(ctx, ctx.a[i], uint32(ctx.mode), ctx.path); err != nil {
            erro(ctx, "%v", err)
            break
        }
    }
    return
}

type __grep struct { builtinbase }
func (ctx *__grep) inner() Context { return &ctx.builtinbase }
func (ctx *__grep) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__grep) x() (_ any) {
    var args = ctx.a
    var nargs = len(args)
    if !(nargs == 2 || nargs == 3) {
        erro(ctx, "wrong args, try $(grep {=regex '^example$'},$0,$(file))")
        return
    }

    var result Value
    var rxs []*regexp.Regexp // TODO: move it into builtinGrepOpts
    var rvs = merge(args[0])
    switch nargs {
    case 2:   args = args[1:]
    case 3: result = args[1]; args = args[2:]
    }

    for _, a := range rvs {
        if x, y := a.(*regexpat); y {
            rxs = append(rxs, x.Regexp)
        } else if s := __string(ctx, a); s == "" {
            erro(ctx, "empty regexp")
            return
        } else if r, e := regexp.Compile(s); e != nil {
            erro(ctx, "%v", e)
            return
        } else {
            rxs = append(rxs, r)
        }
    }

    var p Position
    var res []Value
    for _, a := range merge(args...) {
        var c = pc(ctx, a)
        if x, y := a.(*file); y {
            p.Filename = x.fullname()
        } else {
            p.Filename = __string(ctx, a)
        }

        var e error
        var f *os.File
		if p.Filename == "" {
			erro(c, "empty filename: %s", ts(a,ctx))
			return
		} else if f, e = os.Open(p.Filename); e != nil {
			erro(c, "%s ; %s", e, ts(a,ctx))
			return
		} else {
			defer f.Close()
		}

        s := bufio.NewScanner(f)
        s.Split(bufio.ScanLines)
        p.Line, p.Column = 0, 0
        for s.Scan() {
            text := s.Text()
            p.Line += 1

            for _, rx := range rxs {
                // := rx.FindStringSubmatch(text)
                si := rx.FindStringSubmatchIndex(text)
                if si == nil { continue }

                var val Value

                ctx.defs = make(def_map) // ensure a clear defs map
                for i, n := range rx.SubexpNames() {
                    if n == "" { n = strconv.Itoa(i) }

                    var t string
                    var a, b = si[2*i], si[2*i+1]
                    if 0 <= a && 0 < b { p.Column, t = 1+a, text[a:b] }

                    var v = &xloc{_rw(0, t), p}
                    ctx.set(pc(c,p), defVoid, intern(n), v)

                    if i == 0 && result == nil { val = v } else
                    if 0 < i && a < 0 { p.Column += utf8.RuneCountInString(t) }
                }
                if result != nil { val = expand(_final(c), result) }
                res = append(res, val)

                if checkpoints { ctx.check(rx, text, result, val) }
            }
        }
    }
    return ease(ctx, res)
}

var (
    rsAutoconf  = `AC_(CHECK_(FILES?|FUNCS?|HEADERS?|PROG|SIZEOF|TOOL)|DEFINE)\(([^\)]*?)\)`
    rsConfigRef = `[$%]\{([^\s\}]+)\}|@([^\s\@]+)@`
    rsConfigure = `^[\t ]*#[\t ]*(define|undef|smartdefine|smartdefine01|cmakedefine|cmakedefine01)[\t ]+([A-Za-z0-9_]+)(?:[\t ]+([^\n]*))?$`
    rxAutoconf  = regexp.MustCompile(rsAutoconf)
    rxConfigure = regexp.MustCompile(fmt.Sprintf(`(?m:%s)`, rsConfigure)) // m: multilines
    rxConfigRef = regexp.MustCompile(rsConfigRef)
)

func (p *project) strExpandConfig(ctx Context, s string) (result string, err error) {
    var pos Position
    var index, line = 0, 0
    var res = new(bytes.Buffer)
    if v := auto_get(ctx, intern("-file")); v != nil {
        if x, y := to_file(v); y { pos.Filename = x.fullname() }
    }
    for _, m := range rxConfigRef.FindAllStringSubmatchIndex(s, -1) {
        line += strings.Count(s[index:m[0]], "\n")
        pos.Line = 1 + line
        pos.Column = m[0] - index - strings.LastIndex(s[index:m[0]], "\n")

        fmt.Fprint(res, s[index:m[0]])
        index = m[1] // reset index immediately to keep forward

        var name string
        switch {
        case m[2] > m[0] && m[3] > m[2]: name = s[m[2]:m[3]] // ${VAR}
        case m[4] > m[0] && m[5] > m[4]: name = s[m[4]:m[5]] //  @VAR@
        }

        var d *def
        var val Value
        if d = p.resolveDef(ctx, intern(name)); d == nil {
            if true {
                debug(ctx,
					_f("%v: %v undefined\n", pos, name),
					_f("in %v", p))
            }
            continue
        } else if val = evoke(ctx, d, nil, nil); isNull(val) {
            if f := p.configuration_sm(ctx); f == nil {
                erro(ctx, "%v: configuration file not defined", name, f)
                return
            } else if !f.exists() {
                erro(ctx,
					_f("%s: file not exists (for %v)\n", f.fullname(), name),
					_f("%v: configuration file not exists, try -conf first", name))
                return
            }
            continue
        }

        switch t := val.(type) {
        case *undef, undef: // FIXME: fmt.Fprintf(res, "#undef")
        case *answer, *boolean:
            fmt.Fprintf(res, "%d", __int(ctx, t))
        case *group:
            fmt.Fprintf(res, "%s", __string(ctx, parseGroupValue(ctx, t)))
        case *plain:
            fmt.Fprintf(res, "%s", t.String())
        default:
            fmt.Fprintf(res, "%s", __string(ctx, val))
        }
    }
    if index < len(s) { fmt.Fprint(res, s[index:]) }
    result = res.String()
    return
}

// https://www.gnu.org/software/autoconf/manual/autoconf-2.67/autoconf.html
func autoconf(ctx Context, out *bytes.Buffer, p *project, str string) (err error) {
    var num int
    for _, m := range rxAutoconf.FindAllStringSubmatch(str, -1) {
        info(ctx, "TODO: %v", m)
        num += 1
    }
    debug(ctx, "TODO: %d", num)
    return
}

func configurestring(ctx Context, out *bytes.Buffer, p *project, str string) {
    if s, e := p.strExpandConfig(ctx, str); e != nil {
        erro(ctx, "%v : %v", str, e)
    } else {
        str = s
    }

    var index = 0

    for _, ii := range rxConfigure.FindAllStringSubmatchIndex(str, -1) {
        if _, e := out.WriteString(str[index:ii[0]]); e != nil {
            erro(ctx, "%v", e)
        }

        index = ii[1]

		var (
			d *def
			t bool
			s string
			verb = str[ii[2]:ii[3]]
			name = str[ii[4]:ii[5]]
			hasv = ii[6] > ii[0] && ii[7] > ii[6]
		)
        if d = p.resolveDef(ctx, intern(name)); d != nil {
            if v := evoke(ctx, d, nil, nil); v == nil {
                // noop, TODO: or #undef?
            } else if _, t := v.(*undef); t {
                _, e := out.WriteString(fmt.Sprintf("#undef /* %s */", name))
                if e != nil {
                    erro(ctx, "%v", e)
                } else {
                    continue
                }
            } else {
                t = __true(ctx, v)
            }
        }

        switch verb {
        case "define":
            if hasv /*&& !(def == nil || d.value == nil)*/ {
                v := str[ii[6]:ii[7]]
                s = fmt.Sprintf("#define %s %s", name, v)
            } else {
                s = fmt.Sprintf("#define %s", name)
            }
        case "undef":
            if d == nil {
                s = fmt.Sprintf("#undef %s", name)
            } else if isNull(d.value) || isNone(d.value) {
                s = fmt.Sprintf("#undef %s /* %v */", name, d.value)
            } else if va := expand(ctx, d.value); va != nil {
                switch v := va.(type) {
                case *answer, *boolean:
                    if b := __true(ctx, v); b {
                        s = fmt.Sprintf("#define %s 1 /* %s %v */", name, typeof(v), v)
                    } else {
                        s = fmt.Sprintf("#undef %s /* %s %v */", name, typeof(v), v)
                    }
                case *strlit:
                    s = strings.Replace(v.s, "\"", "\\\"", -1)
                    s = fmt.Sprintf("#define %s \"%s\"", name, v.s)
                default:
                    s = fmt.Sprintf("#define %s %v /* %s */", name, v, typeof(v))
                }
            } else {
                var v = d.value
                s = fmt.Sprintf("#define %s %v /* %s %v */", name, typeof(v), v, va)
            }
        case "smartdefine", "cmakedefine":
            if !t {
                s = fmt.Sprintf("/* #undef %s */", name)
            } else if hasv {
                v := str[ii[6]:ii[7]]
                s = fmt.Sprintf("#define %s %s", name, v)
            } else {
                s = fmt.Sprintf("#define %s", name)
            }
        case "smartdefine01", "cmakedefine01":
            if !t {
                s = fmt.Sprintf("#define %s 0", name)
            } else if hasv {
                v := str[ii[6]:ii[7]]
                s = fmt.Sprintf("#define %s 1 /* %s */", name, v)
            } else {
                s = fmt.Sprintf("#define %s 1", name)
            }
        }

        if _, e := out.WriteString(s); e != nil {
            erro(ctx, "%v", e)
        }
    }

    if len(str) <= index {
        return
    }

    if _, e := out.WriteString(str[index:]); e != nil {
        erro(ctx, "%v", e)
    }
    return
}

type __return struct { builtinbase }
func (ctx *__return) inner() Context { return &ctx.builtinbase }
func (ctx *__return) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__return) x() (res any) {
    return &returner{valbase{_pos(ctx)}, ctx.a}
}


type _benchmark struct {
    tag string
    start, spot time.Time
    spent time.Duration
    num int64
    frames []*_benchmark
}

type _benchspot struct {
    tag string
    n int64
    d time.Duration
    x time.Duration
}

type _sum struct {
    n int64
    d time.Duration
}

var (
    // benchmarkM sync.Mutex
    // benchmark_ *_benchmark = &_benchmark{ tag:"benchmark" }
    benchspotM sync.Mutex
    benchspot = make(map[string]*_benchspot,64)
)

type bencher interface { bench(t time.Time) }
func bench(i bencher, t time.Time) { i.bench(t) }

func (frame *_benchmark) report(w io.Writer, indent int, up *_benchmark) {
    var s string
    if frame.num > 0 {
        var d time.Duration
        if up != nil { d = frame.spot.Sub(up.start) }
        s = fmt.Sprintf("%s (n=%v, d=%v, s=%v) (", frame.tag, frame.num, frame.spent, d)
    } else {
        s = fmt.Sprintf("%s (d=%v) (", frame.tag, frame.spent)
    }
    fprintIndentDots(w, indent, s)
    for _, sub := range frame.frames { sub.report(w, indent + 2, frame) }
    fprintIndentDots(w, indent, ")")
}

func (frame *_benchmark) _tag() string {
    var s = frame.tag
    if i := strings.Index(s, "("); i > 0 { s = s[0:i] }
    return s
}

func (frame *_benchmark) sum(res map[string]_sum) map[string]_sum {
    var t = frame._tag()
    var s = res[t]
    s.n += frame.num
    s.d += frame.spent
    res[t] = s
    for _, p := range frame.frames { p.sum(res) }
    return res
}

func (frame *_benchmark) _summary() (res map[string]_sum) {
    res = make(map[string]_sum,16)
    for _, f := range frame.frames { f.sum(res) }
    return
}

func (frame *_benchmark) summary(w io.Writer) {
    var m = frame._summary()
    var tags []string
    for tag, _ := range m { tags = append(tags, tag) }
    for i := 0; i < len(tags); i += 1 {
        for j := i+1; j < len(tags); j += 1 {
            a, b := tags[i], tags[j]
            if m[a].d < m[b].d {
                tags[i] = b
                tags[j] = a
            }
        }
    }
    for _, tag := range tags {
        var p = m[tag]
        fmt.Fprintf(w, "%s ", tag)
        var i = 40 - len(tag)
        for i > ndots {
            fmt.Fprint(w, dots)
            i -= ndots
        }
        fmt.Fprint(w, dots[0:i])
        fmt.Fprintf(w, "{ %d, %s, %s }\n", p.n, p.d, time.Duration(int64(p.d)/p.n))
    }
}

func benchspot_report(w io.Writer) {
    var tags []string
    for tag, _ := range benchspot { tags = append(tags, tag) }
    for i := 0; i < len(tags); i += 1 {
        for j := i+1; j < len(tags); j += 1 {
            a, b := tags[i], tags[j]
            if benchspot[a].d < benchspot[b].d {
                tags[i] = b
                tags[j] = a
            }
        }
    }
    for _, tag := range tags {
        fe := benchspot[tag]
        fmt.Fprintf(w, "%s ", tag)
        var i = 40 - len(tag)
        for i > ndots {
            fmt.Fprint(w, dots)
            i -= ndots
        }
        fmt.Fprint(w, dots[0:i])
        fmt.Fprintf(w, "{ %d, %s, %s }\n", fe.n, fe.d, fe.x)
    }
}


func do_helpscreen(ctx Context) {
    prompt(ctx, `Build your projects the smart way!

Usage:

    smart -help[(arguments)]
    smart -configure[(arguments)]
    smart -reconfigure[(arguments)]
`)
    for name, _ := range _universe(ctx).globe.flagEntries {
        if name == "" { continue }
        prompt(ctx, `
    smart -%s[(arguments)]`, name)
    }

    prompt(ctx, `

Basic:

   -h
   -help
    Display this help screen.

   -c
   -configure
    Configure all projects underneath the work directory.

   -r
   -reconfigure
    Reconfigures all projects underneath the work directory.

`)

    print_flag_entries(ctx)
    print_help_entries(ctx)
    print_options(ctx)

    prompt(ctx, `
Issues:

    * https://github.com/extbit/smart/issues
    * https://bugs.extbit.io/smart/report (not ready yet)

`)
}

func print_flag_entries(ctx Context) {
        prompt(ctx, "Defined:\n")
        for name, entries := range _universe(ctx).globe.flagEntries {
                if len(entries) == 0 || name == "" { continue }
                prompt(ctx, `
   -%s`, name)
        }
        prompt(ctx, "\n\n")
}

func print_flag_trace(ctx Context) {
        for name, entries := range _universe(ctx).globe.flagEntries {
                if name == "" { continue }
                for _, entry := range entries {
                        prompt(ctx, "%s: %v\n", entry.Pos(), entry)
                }
        }
}

func print_help_entries(ctx Context) {
}

func print_options(ctx Context) {
    type opt struct { entry entry; infos []Value }

    var opts []opt

    // _universe(ctx).config(func(proj *project, entry entry) {
    //     var infos = ruleOptionInfos(ctx, entry)
    //     if infos != nil { opts = append(opts, opt{entry, infos}) }
    // }, nil, nil)

    if len(opts) == 0 { return }

    prompt(ctx, "Configure:\n\n")
    for _, opt := range opts {
        prompt(ctx, "    %v:\n", opt.entry)
        for _, info := range opt.infos {
            prompt(ctx, "        %s\n", __string(ctx, info))
        }
    }
}

func print_configuration(ctx Context) {
    prompt(ctx, `Configuration:
`)

    var configs = make(map[*project][]entry)

    // _universe(ctx).config(func(proj *project, entry entry) {
    //     entries, _ := configs[proj]
    //     entries = append(entries, entry)
    //     configs[proj] = entries
    // }, nil, nil)

    for project, entries := range configs {
        prompt(ctx, `
    %s`, project.spec)
        for _, entry := range entries {
            prompt(ctx, `
        %s`, entry)
        }
    }

    prompt(ctx, "\n")
}

func ruleOptionInfos(ctx Context, e entry) (infos []Value) {
    for _, p := range e.programs() {
        for _, depend := range p.depends {
            g, ok := depend.(*modification)
            if!ok { continue }
            for _, m := range g.list {
                if __string(ctx, m.elems[0]) != "configure" { continue }
                for _, arg := range m.elems[1:] {
                    a, ok := arg.(*argumented)
                    if!ok { continue }
                    f, ok := a.Value.(flag)
                    if!ok { continue }
                    if __string(ctx, f.Value) != "option" { continue }
                    for _, v := range a.args {
                        if p, ok := v.(*pair); ok {
                            if __string(ctx, p.key) != "info" { continue }
                            v = p.val
                        }
                        infos = append(infos, v)
                    }
                    return
                }
            }
        }
    }
    return
}
