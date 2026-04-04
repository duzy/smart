///
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"reflect"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

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
	pos, endPos Pos // token position
	tok  token  // one token look-ahead
	lit  string // token literal
	verb string
	name Value // if only 'def', TODO: considering []Value for nested template defs?
	params []Value
}

type parser struct{
	scanner

	// Next token
	pos, stop Pos // parsing and stop position
	tok token  // one token look-ahead
	lit string // token literal
	multibyte int

	comments  []*commentgroup
	leadComment *commentgroup // last lead comment
	lineComment *commentgroup // last line comment

	templates []*template

	imports []*use_spec // list of imports

	targets []Value // targets of current rule
	ruparas []*auto // parameters of current rule
	dialect  string // recipe dialect of current rule

	locals []map[string]*def
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
type codeblock      struct{ *automatic ; token }
type defval         struct{ original ; d *def}
type def_name       struct{ Context }
type braced         struct{ Context }
type p_auto_ctx     struct{ Context }
type foreach_txt    struct{ Context ; a *auto }
type grep_txt       struct{ Context ; o objbase ; a map[string]*auto }
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

func (p *codeblock) inner() Context { return p.Context }
func (p *codeblock) cast(t reflect.Type) Context {
    if reflect.TypeOf(p) == t { return p }
    return p.automatic.cast(t)
}
func (p *codeblock) ts(t string) (_ string) {
	return "{="+t+" "+p.token.String()+" "+ts(p.Context)+"}"
}

func (p p_undef) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_undef) inner() Context { return p.Context }
func (p p_undef) ts(t string) (_ string) {
	return "{="+t+" "+ts(p.Context)+"}"
}
func (p p_undef) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case is_undef: return true
	}
	return p.Context.do(ctx, op)
}

func (p p_perc) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_perc) inner() Context { return p.Context }
func (p p_perc) ts(t string) (_ string) {
	return fmt.Sprintf("{="+t+" %s}", ts(p.Context))
}
func (p p_perc) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case p_no_argumented: return true
	}
	return p.Context.do(ctx, op)
}

func (p p_glob) inner() Context { return p.Context }
func (p p_glob) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_glob) ts(t string) (_ string) { return "{="+t+" "+ts(p.Context)+"}" }
func (p p_glob) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case p_is_glob: return true
	}
	return p.Context.do(ctx, op)
}


type p_is_strcomp struct{}
func (p p_strcomp) inner() Context { return p.Context }
func (p p_strcomp) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_strcomp) ts(t string) (_ string) { return "{="+t+" "+ts(p.Context)+"}" }
func (p p_strcomp) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case p_is_strcomp: return true
	}
	return p.Context.do(ctx, op)
}

func (p p_params) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_params) inner() Context { return p.Context }
func (p p_params) ts(t string) (_ string) {
	return fmt.Sprintf("{="+t+" %s}", ts(p.Context))
}
func (p p_params) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case p_is_params: return true
	}
	return p.Context.do(ctx, op)
}

type is_auto_preserved struct{ s string }
type is_auto           struct{ s string }
type is_defname        struct{}

func (p p_auto_ctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_auto_ctx) inner() Context { return p.Context }
func (p p_auto_ctx) ts(t string) (_ string) {
	return fmt.Sprintf("{="+t+" %s}", ts(p.Context))
}
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
	case is_defname: return true
	}
	return p.Context.do(ctx, op)
}

func (p defval) inner() Context { return p.Context }
func (p defval) cast(t reflect.Type) Context { return icast(p,t) }
func (p defval) do(ctx Context, op any) any {
	switch t := op.(type) {
	case is_auto: return t.s != "0" && IsDigits(t.s)
	case keep_autos: return true
    case origin_def:
        if p.d != nil && (t.name == "" || t.name == p.d.name) { return p.d }
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
func (p *foreach_txt) ts(t string) (_ string) { return "{="+t+" "+ts(p.Context)+"}" }
func (p *foreach_txt) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
    case find_auto: if t.s == "_" { return p.a }
	case is_auto: if t.s == "_" { return true }
	case is_auto_preserved: if t.s == "_" { return true }
	}
	return p.Context.do(ctx, op)
}

func (p *grep_txt) inner() Context { return p.Context }
func (p *grep_txt) cast(t reflect.Type) Context { return icast(p,t) }
func (p *grep_txt) ts(t string) (_ string) { return "{="+t+" "+ts(p.Context)+"}" }
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
		p.a = map[string]*auto{"0":&auto{knownobject{p.o, "0"}}}
		for i, name := range t.SubexpNames() {
			if 0 < i {
				if name == "" { name = strconv.Itoa(i) }
				p.a[name] = &auto{knownobject{p.o, name}}
			}
		}
	}
	return p.Context.do(ctx, op)
}

func (p p_rule_ctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_rule_ctx) inner() Context { return p.Context }
func (p p_rule_ctx) ts(t string) string { return "{="+t+" "+ts(p.Context)+"}" }
func (p p_rule_ctx) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case is_auto:
		if IsDigits(t.s) { return true }
		if _, y := rule_autos[t.s]; y { return true }
	}
	return p.Context.do(ctx, op)
}

type add_recipe_line struct{ a []Value }
type is_recipe_start struct{}
type is_recipe       struct{ bool } // builtin or text

func (p *p_recipe) cast(t reflect.Type) Context { return icast(p,t) }
func (p *p_recipe) inner() Context { return p.Context }
func (p *p_recipe) ts(t string) string { return "{="+t+" "+ts(p.Context)+"}" }
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
func (p left_side) ts(t string) string { return fmt.Sprintf("{="+t+" %s}", ts(p.Context)) }
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

	if n := p.scanner.ch_bytes(); n > 1 { p.multibyte += n-1 }

	switch p.pos, p.tok, p.lit = p.scanner.scan(ctx); p.tok {
	case COMMENT, LINEND: p.multibyte = 0
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

func (p *parser) valbase(Context) valbase { return valbase{p.pos} }
func (p *parser) loc(a Pos) Position { return p.scanner.file.Position(a) }
func (p *parser) line() int { return p.scanner.file.Line(p.pos) }
func (p *parser) column() int { return utf8.RuneCount(p.scanner.src[p.scanner.offsetLine:p.scanner.offset]) }
func (p *parser) Position() (r Position) {
	if r = p.loc(p.pos); 0 < p.multibyte { r.Column -= p.multibyte }
	return
}

func (p *parser) is_file(s string) bool {
	return strings.HasSuffix(p.scanner.file.Name(), s)
}

func (p *parser) expect(ctx Context, tok token) Pos {
	var pos = p.pos
	if p.tok == tok {
		p.step(ctx) // move forward
	} else {
		debug(pc(ctx,p), "expect %v, not %v", tok, p.tok, trace{})
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
		debug(pc(ctx,p), "expect end of line, but %v", p.tok, trace{})
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
	case *word, *compound:
		if o := l.resolve(ctx, t, __string(ctx, lhs)); !isNull(o) {
			return _arrow(pos, tok, o, l.composite(ctx))
		} else if tok == SELECT_PROG2 {
			return _null(pos) // ignore
		} else {
			debug(ctx, _f("%v: '%v' is undefined (name=%v, obj=%v)", l.project, lhs, t, o),
				_f("%v: parser is here (name=%s, tok=%s)", l.project, lhs, tok),
				_f("%v: parser to go here (tok=%s, lit=%s)", l.project, l.p.tok, l.p.lit),
				trace{})
		}
	}
	return _arrow(pos, tok, lhs, l.composite(ctx))
}

func (p *parser) bare(ctx Context) Value {
	tok, lit, pos := p.tok, p.lit, p.pos
	if tok != WORD && lit == "" { lit = tok.String() }
	p.step(ctx) // consumes the current token
	return _word(pos, lit)
}

func (l ul) braced(ctx Context) (x Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "braced")) }

	pos := l.p.pos
	l.p.expect(ctx, LBRACE)

	ctx = braced{ctx}

	if l.p.tok == RBRACE {
		x = &null{l.p.valbase(ctx)}
		l.p.spaces(ctx)
		l.p.step(ctx) // consumes }
		return
	}

	switch l.p.tok {
	case LBRACK: // obsolete syntax: {[...]}
		debug(ctx, "syntax error, use {(modifier)} for modification", trace{})
		return
	case LPAREN:
		x = l.modification(ctx)
		l.p.spaces(ctx)
		l.p.expect(ctx, RBRACE)
		return
	case ASSIGN: // =
		l.p.step(ctx) // skips =
		switch l.p.tok {
		case AND:     return l.braced_and(ctx)
		case OR:      return l.braced_or(ctx)
		case FOR:     return l.braced_for(ctx)
		case FOREACH: return l.braced_foreach(ctx)
		case PROJECT: // {=project ...}
			l.p.next(ctx, true)
			x = l.braced_project(ctx)
			l.p.expect(ctx, RBRACE)
			return

		case BARE: // {=bare ...}
			l.p.next(ctx, true)
			x = l.p.bare(ctx)
			l.p.spaces(ctx)
			l.p.expect(ctx, RBRACE)
			return

		case RAW: // {=raw ...}
			l.p.next(ctx, true)
			x = &raw{l.p.valbase(ctx), __string(ctx, l.expr(ctx))}
			l.p.spaces(ctx)
			l.p.expect(ctx, RBRACE)
			return

		case UNDEF: // {=undef ...}
			l.p.next(ctx, true)
			x = undef{l.expr(ctx)}
			l.p.spaces(ctx)
			l.p.expect(ctx, RBRACE)
			return

		case NULL: // {=null}
			return l.braced_null(ctx)

		case NONE: // {=none ...}
			return l.braced_none(ctx)

		case ANSWER, BOOL, BOOLEAN, BIN, OCT, INT, HEX, FLOAT: // {=bin ...}, {=oct ...}, {=int ...}, {=hex ...}, {=float ...}
			return l.braced_type(ctx, l.p.tok)

		case TRUE, FALSE, YES, NO, ON, OFF: // {=true}, {=false}, {=yes}, {=no}, {=on}, {=off}
			return l.braced_const(ctx, l.p.tok)

		case FILE: // {=file ...}
			return l.braced_file(ctx)

		case PATH: // {=path ...}
			return l.braced_path(ctx)

		case GLOB: // {=glob ...}
			l.p.next(ctx, true)
			g := l.glob(ctx, nil)
			l.p.spaces(ctx)
			l.p.expect(ctx, RBRACE)
			return &globbrace{*g}

		case REGEX: // {=regex ...}
			l.p.step(ctx)
			l.p.scanner.addBits(isBraceRaw)

			// Trim leading spaces differently to avoid messing the scan states.
			// NOTE: the first SPACE and WORD do not become RAW.
			for l.p.tok == SPACE || (l.p.tok == RAW && l.p.lit == " ") { l.p.step(ctx) }
			return l.regex(ctx)

		case WORD:
			switch t := l.p.lit; t {
			case "here":
				l.p.next(ctx, true)
				l.p.expect(ctx, RBRACE)
				p := l.p.Position()
				x = &compound{elements{[]Value{
					_raw(l.p.pos, p.Filename), _punct(ctx, COLON),
					_decimal(l.p.pos, int64(p.Line)), _punct(ctx, COLON),
					_decimal(l.p.pos, int64(p.Column)), _punct(ctx, COLON),
				}}}
				return

			case "plain":
				x = &plain{elements{l.braced_plain(ctx)}, t}
				l.p.expect(ctx, RBRACE)
				return

			case "plainline":
				x = &plainline{elements{l.braced_plain(ctx)}}
				l.p.expect(ctx, RBRACE)
				return

			case "self":
				l.p.next(ctx, true)
				x = self{l.braced_project(ctx)}
				l.p.expect(ctx, RBRACE)
				return

			case "str": // $(string ...)
				x = l.braced_str(ctx)
				l.p.expect(ctx, RBRACE)
				return

			case "quote": // $(quote ...)
				x = l.braced_quote(ctx)
				l.p.expect(ctx, RBRACE)
				return

			case "word":
				x = l.braced_word(ctx)
				l.p.expect(ctx, RBRACE)
				return

			case "grep":
				if false {
					x = l.braced_word(ctx)
					l.p.expect(ctx, RBRACE)
					return
				}

			case "defs":
				x = l.braced_defs(ctx)
				l.p.expect(ctx, RBRACE)
				return

			case "fullname":
				x = l.braced_fullname(ctx)
				l.p.expect(ctx, RBRACE)
				return
			}

			debug(pc(ctx,l.p), "unsupported braced type: %v %v", l.p.tok, l.p.lit, trace{})
			return

		default:
			l.p.next(ctx, true)
			return
		}
	default: // {...}
		if v := l.values(ctx); len(v) == 0 {
			x = _null(pos)
		} else if len(v) == 1 {
			x = &disjunction{valbase{pos},v[0]}
		} else {
			x = &disjunction{valbase{pos},_list(v...)}
		}
		l.p.spaces(ctx)
		l.p.expect(ctx, RBRACE)
		return
	}
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
		switch arg.(type) {
		case *word, *compound/* , *qualword */:
			name := __string(ctx, arg)
			a := &auto{knownobject{objbase{valbase{arg.Pos()}, s}, name}}
			
			// CRITICAL FIX: The parameter must be resolvable by its explicit name 
			// inside the rule body (e.g., $(ARG1)), not just by its positional alias ($1).
			s.alias(ctx, a, name) // Map "ARG1" -> auto
			s.alias(ctx, a, strconv.Itoa(len(p.ruparas)+1)) // Map "1" -> auto
			
			p.ruparas = append(p.ruparas, a)
		default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
			debug(ctx, "bad parameter form (%v)", ts(arg), trace{})
		}
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
			debug(ctx, "unexpected colon", trace{})
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
			debug(ctx, "bad: %v %v; %v", l.p.tok, l.p.lit, values, trace{})
		}

		// If there's a comment right after the parsed expression, we break
		// the expression list to treat the end-of-line comment like a LINEND.
		if l.p.tok == EOF || l.p.tok == LINEND || l.p.lineComment != nil { break }
	}
	return
}

func (l ul) group(ctx Context) *group {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "group")) }

	ctx = p_group_ctx{aware(ctx,COMMA)}

	pos := l.p.pos
	l.p.expect(ctx, LPAREN)
	l.p.spaces(ctx)

	var elems, converted = l.values(ctx), false
	for l.p.tok != RPAREN && l.p.tok != EOF {
		// if l.p.tok == COMMA { l.p.next(ctx, true) }
		switch l.p.tok {
		case BAR, COMMA, SEMICOLON:
			elems = append(elems, l.punct(ctx))
			l.p.spaces(ctx)
		}

		p := l.p.pos
		next := _list(l.values(ctx)...)

		if l.p.pos == p { debug(ctx, "syntax error", callstack{num:64}, trace{}) }

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
	default: debug(pc(ctx,l.p), "unexpect %v", l.p.tok, trace{})
	}

corner_loop:
	for l.p.spaces(ctx); l.p.tok != EOF; l.p.spaces(ctx) {
		var saved = l.p.pos
		if elems = append(elems, l.expr(ctx)); l.p.pos == saved {
			debug(ctx, "bad: %v %v; %v", l.p.tok, l.p.lit, elems, trace{})
		}

		// If there's a comment right after the parsed expression, we break
		// the expression list to treat the line-end comment like a LINEND.
		if l.p.lineComment != nil { break }

		switch l.p.tok { case Rbot_corner, Rtop_corner, LINEND: break corner_loop }
	}

	switch l.p.tok {
	case Rbot_corner, Rtop_corner: l.p.step(ctx)
	default: debug(pc(ctx,l.p), "unexpect %v", l.p.tok, trace{})
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
				debug(ctx, "unexpected punctuation: %v", l.p.tok, trace{})
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

		if l.p.pos == p { debug(ctx, "syntax error", trace{}) }

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
					debug(ctx, "too many %", trace{})
				default:
					switch perc.Suffix = l.expr(ctx); perc.Suffix.(type) {
					case *argumented, *path:
						debug(ctx, "incorrect: %v %v", x, ts(perc.Suffix), trace{})
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
		debug(ctx, "wrong scan state: %v", l.p.scanner.scanstate, trace{})
	}
	if !l.p.scanner.bits.isBraceRaw() {
		debug(ctx, "wrong scan state: %v", l.p.scanner.scanstate, trace{})
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
		if l.p.lit == "" {
			rx += l.p.tok.String()
		} else {
			rx += l.p.lit
		}
	}

	l.p.expect(ctx, RBRACE)

	var err error
	var x = &regexpat{valbase{pos}, nil} // TODO: correct regexp pattern value
	if x.Regexp, err = regexp.Compile(rx); err != nil {
		debug(pc(ctx,l.p), "regex: %v", err, trace{})
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
		if l.p.pos == p { debug(ctx, "syntax error", trace{}) }
	}

	if x == nil {
		debug(ctx, "nil flag name", trace{})
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
	p := &punct{l.p.valbase(ctx), l.p.tok}
	l.p.step(ctx)
	return p
}

func (l ul) escape(ctx Context) *escaped {
	v := &escaped{l.p.valbase(ctx), l.p.lit}
	l.p.expect(ctx, ESCAPE)
	return v
}

func (l ul) literal(ctx Context) (_ Value) {
	tok, lit, pos := l.p.tok, l.p.lit, l.p.pos

	l.p.step(ctx)

	// ESCAPE is handled in value.EscapeChar
	switch tok {
	case BAR: debug(ctx, "`|` is deprecated, change the modifiers!", trace{})
	case BINARY:      return ParseBinary(pos, lit)
	case OCTAL:       return ParseOctal(pos, lit)
	case INTEGER:     return ParseDecimal(pos, lit)
	case HEXADECIMAL: return ParseHexadecimal(pos, lit)
	case DATETIME:    return ParseDateTime(pos, lit)
	case DATE:        return ParseDate(pos, lit)
	case TIME:        return ParseTime(pos, lit)
	case URL:         return ParseURL(pos, lit)
	case FLOATING:    return parseFloat(pos, lit)
	case WORD:        return _word(pos, lit)
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
		if l.p.pos == p { debug(ctx, "syntax error", trace{}) }
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

		if l.p.pos == p { debug(ctx, "syntax error", trace{}) }

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
	if start == nil { debug(ctx, "nil path starter", trace{}) }

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
		for l.p.step(ctx); l.p.tok == PCON; l.p.step(ctx) {} // repeated '/'

		switch l.p.tok {
		case LPAREN, LBRACE, RPAREN, RBRACE, COMMA, SPACE, LINEND:
			res.elems = append(res.elems, &punct{l.p.valbase(ctx),PTAIL}) // after the last '/'
			return
		}

		var p = l.p.pos
		var elem = l.unary(ctx)
		if l.p.pos == p { debug(ctx, "syntax error", trace{}) }
		if x, y := elem.(*list); false && y && x.len() == 1 { elem = x.elems[0] }

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

func isKnownURLScheme(s string) (result bool) {
	switch strings.ToLower(s) {
	case "file", "http", "https", "ftp", "ftps", "mailto":
		result = true
	}
	return
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
		if p == nil { p = makePath(&punct{l.p.valbase(ctx),PROOT}) }

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
				debug(ctx, "unexpected %v in url", l.p.tok, trace{})
			}
		}
	}

	if l.p.tok == HASH {
		l.p.step(ctx) // '#'
		u.Fragment = l.unary(url_fragment{ctx})
	}

	return u
}

func (l ul) promptConfigurationLoads(ctx Context) bool { return __true(ctx, l.resolve(ctx, nil, "prompt-configuration-loads")) }
func (l ul) promptCachedConfigs(ctx Context) bool { return __true(ctx, l.resolve(ctx, nil, "prompt-cached-configs")) }
func (l ul) resolve(ctx Context, name Value, str string) (result Value) {
	var pos Pos
	if name != nil { pos = name.Pos() }
	if !pos.IsValid() { pos = l.p.pos }
	if !pos.IsValid() { pos = _pos(ctx) }
	if str == "" {
		debug(ctx, "resolve no-name : %v", ts(name), trace{})
	}

	if d := auto_find(ctx, str); d != nil {
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
		if _, o = s.find(str); o != nil { return o }
	}
	if l.project != nil {
		if o = l.project.resolve(ctx, str); o != nil { return o }
	}

	// CRITICAL FIX: Rule-specific auto delegates (like $@, $<, $*...) AND 
	// numeric arguments ($1, $2...) MUST be globally parseable so they can be 
	// embedded in reusable global macros or late-bound templates.
	_, isRuleAuto := rule_autos[str]

	switch {
	case isRuleAuto || IsDigits(str) || truly(ctx, is_auto{str}):
		// ZERO-ALLOCATION FIX: Do not permanently inject autos into the scope map! 
		// An auto is just a late-binding placeholder. Create and return it natively.
		return &auto{knownobject{objbase{valbase{pos}, s}, str}}
	case truly(ctx, is_config_mode{}):
		return s.def(ctx, defVoid, str)
	}	

	if l.project != nil {
		if c := l.project.configure; c != nil {
			return c.resolve(ctx, str)
		}
	}
    return
}

func (l ul) identity(ctx Context, tok token, name Value) (obj Value, str string, opts []Value) {
	var ic *ident_ctx

	ic, ctx = identity_ctx(ctx)

	switch x := name.(type) {
	case object:
		obj, str = x, ident(ctx, x)
		return
	case *argumented:
		obj, str, opts = l.identity(ctx, tok, x.Value)
		opts = append(opts, merge(x.args...)...)
		return
	}

	if str = ident(ctx, name); ic.nil > 0 {
		obj = name
		return
	} else if str == "" {
		if truly(ctx, opt_ident{}) {
			obj = name
			return
		}

		debug(pc(ctx,name), "empty ident: %v (nil=%d) : %s", name, ic.nil, ts(name), trace{})
	}

	switch tok {
	case LPAREN:
		if obj = l.resolve(ctx, name, str); obj != nil {
			return
		} else if truly(ctx, opt_ident{}) {
			obj = name
			return
		}
	case LBRACE:
		if e := l.project.entry(ctx, name); e == nil {
			debug(pc(ctx,name), "resolved nil: %s", ts(name), trace{})
		} else if _, ok := e.(object); !ok {
			debug(pc(ctx,name), "not an object: %v: %s", name, ts(e), trace{})
		} else {
			obj = name
			return
		}
	}

	debug(pc(ctx,name), "undefined %v → %v : %s", name, str, ts(name), callstack{num:32}, trace{})
	return
}

func (l ul) calling(ctx Context) (result Value) {
	var tok token
	var str string
	var name, obj Value
	var args, opts []Value
	var pos = l.p.pos
	var closure = l.p.tok.is_closure()

	// CRITICAL FIX: Suspend string modes so bare variables (like $1, $@) scan normally
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

		if l.p.tok == SPACE { debug(pc(ctx,l.p.pos), "unexpected spaces", trace{}) }

		name = l.expr(selection{ctx})

		if closure || optional(name) {
			obj, str, opts = l.identity(optional_ident(ctx), tok, name)
		} else {
			obj, str, opts = l.identity(ctx, tok, name)
		}

		if (tok == LPAREN && l.p.tok != RPAREN) || (tok == LBRACE && l.p.tok != RBRACE) {
			var cc Context = aware(ctx, COMMA)
			switch str {
			case "":
				l.p.spaces(cc)
				args = append(args, _list(l.values(cc)...))
			case "auto":
				if !closure { cc = p_auto_ctx{cc} }
				args = append(args, _list(l.values(cc)...))
			case "and", "or":
				cc = optional_ident(cc)
				args = append(args, _list(l.values(cc)...))
			case "case":
				args = append(args, _list(l.values(cc)...))
				cc = optional_ident(cc)
			case "foreach":
				a := &auto{knownobject{objbase{l.p.valbase(ctx),l.scope()},"_"}}
				args = append(args, _list(l.values(cc)...))
				cc = &foreach_txt{cc, a}
			case "grep":
				cc = &grep_txt{cc, objbase{l.p.valbase(ctx), l.scope()}, nil}
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
		str = name.String()

		// Safety trim: If it evaluated to a float, strip the trailing decimal to get the variable name
		// str = strings.TrimSuffix(str, ".")

		obj = l.resolve(ctx, name, str)

	case STRING:
		tok = l.p.tok
		name = l.literal(ctx) // $'xxxx'
		obj, str, opts = l.identity(ctx, tok, name)

	case STRCOMP:
		tok = l.p.tok
		name = l.strcomp(ctx) // $"xxxx"
		obj, str, opts = l.identity(ctx, tok, name)

	case WORD:
		switch l.p.lit {
		case "_":
			tok, str = UNDERLINE, l.p.lit
			name = &punct{l.p.valbase(ctx),tok}
			obj = l.resolve(ctx, name, str)
			l.p.step(ctx)

		default:
			debug(pc(ctx,l.p.pos), _f("unexpects %v", l.p.lit), trace{})
		}

	default: // case AT, BAR, DOT, SAST, QUE, MINUS, PLUS, PCON:
		tok = l.p.tok
		str = l.p.tok.String()
		
		// Fallback guard: In the rare case a RAW token still leaks through, intercept it.
		if tok == RAW {
			name = _raw(l.p.pos, l.p.lit)
			str = l.p.lit
			l.p.step(ctx)
		} else {
			name = l.punct(ctx) // $@, $?, $*, $/...
		}
		
		if obj = l.resolve(ctx, name, str); obj == nil {
			debug(pc(ctx, name.Pos()),
				_f("unexpected: tok=%v str=%v name=%v dialect=%v", tok, str, name, l.p.dialect),
				trace{})
		}
	}

	if obj == nil && str != "" {
		if l.project.ext.Plugin != nil {
			if t, e := l.project.ext.Lookup(str); e == nil && t != nil {
				debug(pc(ctx, name.Pos()),
					_f("unexpected: tok=%v str=%v name=%v dialect=%v", tok, str, name, l.p.dialect),
					trace{})
			}
		}
	}

	if obj == nil {
		debug(pc(ctx, name.Pos()),
			_f("nil symbol; tok=%v str=%v name=%v", tok, str, name),
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

	pos := l.p.pos

	defer func() {
		if pos == l.p.pos && x != nil {
			if z, y := x.(*punct); !y || z.token != PROOT {
				debug(pc(ctx,x), "syntax error: %v", ts(x), trace{})
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
						debug(ctx, "bad url token: %v %v", l.p.tok, l.p.lit, trace{})
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
		return &punct{l.p.valbase(ctx),PROOT}

	case TILDE: // ~
		return l.punct(ctx)

	case DOT: // .
		return l.dot(ctx, nil)

	case DOTDOT: // . ..
		tok, pos := l.p.tok, l.p.pos
		if l.p.step(ctx) ; l.p.tok == PCON {
			return &punct{l.p.valbase(ctx),tok}
		} else {
			return &punct{valbase{pos}, tok}
		}

	default:
		if l.p.tok.is_keyword() { // keywords here are words
			return l.p.bare(ctx)
		}
	}

	if l.p.tok != EOF {
		debug(pc(ctx, l.p.loc(pos)),
			_f("unexpected %v '%s'", l.p.tok, l.p.lit),
			_f("x: %v %v", x, ts(x)),
			_f("scanstate: %v", l.p.scanner.scanstate),
			callstack{num:16}, trace{})
	}

	if l.p.lineComment != nil {
		for _, c := range l.p.lineComment.comments {
			debug(ctx, "# %s", c.string, trace{})
		}
	}
	return
}

func (l ul) composite(ctx Context) (x Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "composite")) }

	x = l.unary(ctx)

	switch l.p.tok { // check composible expressions
	case PERC: // foo%bar ; FIXME: %/foo/bar -> {=path % foo bar}
		return l.perc(ctx, x)

	case DOT: // foo.bar.baz.o ; FIXME: push bits when parsing $(...)
		return l.dot(ctx, x)

	case QUE: // ?
		return l.glob(ctx, x)

	case COLON:
		if truly(ctx, is_recipe{false}) || !truly(ctx, left_hand_side{}) {
			if w, y := x.(*word); y && isKnownURLScheme(w.s) {
				return l.url(ctx, x)
			}
		}
	}
	return
}

func (l ul) expr(ctx Context) (x Value) {
	if false && l_traverse.enabled { defer un(l_trace(l_traverse, "expr")) }
	if false { defer func() { debug(pc(ctx,l.p), "%s", ts(x)) } () }

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
		debug(ctx, "syntax error: %v (%v %v %v)", x, ts(x), ts(y), l.p.tok, trace{})
	}

	x = prefix(ctx, x, y) // ⇒ xy

	switch l.p.tok { case COMMENT, SPACE, LINEND, EOF: return }

	if 9999 < n { debug(ctx, "too many compose: %v (%d)", x, n, trace{}) }

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

	debug(ctx, "TODO: {=for ...}", trace{})

	l.p.expect(ctx, RBRACE)
	return
}

type foreach_text struct{ Context }
func (f foreach_text) inner() Context { return f.Context }
func (f foreach_text) cast(t reflect.Type) Context { return icast(f, t) }
func (f foreach_text) do(c Context, o any) (_ any) {
    switch t := o.(type) {
    case find_auto:
		if t.s == "_" {
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
	cc.set(ctx, defVoid, "_", nil)

	var temps []Value
	switch l.p.spaces(ctx); l.p.tok {
	case RBRACE: return _null(l.p.pos)
	case COMMA:
		for l.p.step(ctx); l.p.tok != RBRACE; {
			l.p.spaces(ctx)
			if v := l.expr(foreach_text{&cc}); v != nil {
				temps = append(temps, v)
			} else {
				debug(ctx, "nil ; %v", l.p.tok, trace{})
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
			cc.set(ctx, defVoid, "_", elem)

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

func (l ul) braced_quote(ctx Context) (res Value) {
	l.p.next(ctx, true) // resumes 'quote'
    return &quote{list{elements{l.braced_elems(ctx)}}}
}

func (l ul) braced_word(ctx Context) (res Value) {
	l.p.next(ctx, true) // resumes 'word'

	var pos = l.p.pos
	var elems = expands(_final(ctx), l.braced_elems(ctx)...)

	var s string
	for i, v := range elems {
		if s != "" && 0 < i { s += " " }
		s += __string(ctx, v)
	}
	return &word{valbase{pos}, s}
}

type defcapture struct{ name string ; value Value }
type defcaps struct{ Value ; caps []*defcapture }
func (dc *defcaps) String() (s string) {
	s = "{=defcapture "+dc.Value.String()
	for _, cap := range dc.caps {
		s += " {"+cap.name+":"+cap.value.String()+"}"
	}
	s += "}"
	return
}

func (l ul) braced_defs(ctx Context) (res Value) {
	var capture []string

	l.p.step(ctx) // resumes 'defs'

	if l.p.tok == LPAREN {
		l.p.next(ctx, true) // resumes '('
		for l.p.tok != RPAREN && l.p.tok != EOF {
			switch l.p.tok {
			case COMMA:
			case INTEGER, WORD:
				capture = append(capture, l.p.lit)
			default:
				debug(pc(ctx,l.p), "unexpected %v '%s'", l.p.tok, l.p.lit, trace{})
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
			var name = _raw(pos, _k)
			var neg bool
			if x, y := pat.(negative); y { pat, neg = x.Value, y }

			a, _, _, c := match(pc(ctx, pat), pat, name)
			if a && neg { continue defsloop }
			if a || neg {
				var main Value = name
				var caps []*defcapture

				// Always capture $0 as the full name
				caps = append(caps, &defcapture{"0", main})

				if len(capture) == 0 {
					// Auto-numbered captures from stems: $1, $2...
					for i, stem := range c {
						s := strconv.Itoa(i+1)
						caps = append(caps, &defcapture{s, _raw(pos, __string(ctx, stem))})
					}
				} else {
					// Named captures
					// If specifically one numeric capture requested, use that as the main result value
					if len(capture) == 1 {
						if i, e := strconv.Atoi(capture[0]); e == nil && 0 < i && i <= len(c) {
							main = c[i-1]
						}
					}

					for i, name := range capture {
						var val Value
						if i < len(c) {
							val = _raw(pos, __string(ctx, c[i]))
						} else {
							val = &valbase{pos}
						}
						caps = append(caps, &defcapture{name, val})
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
		debug(pc(ctx,elem), "not a file: %v", ts(elem), trace{})
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
				debug(ctx, "unexpected token: %v", l.p.tok, trace{})
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
		debug(pc(ctx,l.p), "nil const, %v, %v %v", tok, l.p.tok, l.p.lit, trace{})
	}

	l.p.spaces(ctx)
	l.p.expect(ctx, RBRACE)
	return
}

func (l ul) braced_const(ctx Context, tok token) (x Value) {
	l.p.next(ctx, true)

	var pos = l.p.pos

	if l.p.spaces(ctx); l.p.tok != RBRACE {
		debug(pc(ctx,l.p), "expecting right-brace, %v %v", l.p.tok, l.p.lit, trace{})
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
		debug(pc(ctx,l.p), "nil const, %v, %v %v", tok, l.p.tok, l.p.lit, trace{})
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

	x = &null{l.p.valbase(ctx)}
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
	str := __string(ctx, name)
	if str == "" {
		debug(ctx, "empty name : %s : %s", ts(name), str, trace{})
	}

	if /* self && */ l.project.name == str {
		return l.project
	} else if o := l.resolve(ctx, name, str); o == nil {
		debug(pc(ctx,l.p), "%s : undefined %s : %v", l.project, str, ts(name), trace{})
		return
	} else if x, y := o.(*project); !y && x != nil {
		debug(pc(ctx,l.p), "%s : %v is not a project", l.project, ts(o), trace{})
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
        debug(pc(ctx,g.spec), "empty use spec: %v", ts(g.spec[0]), trace{})
    }

	var opts useopts
	var args = parseOpts(ctx, &opts, append(g.remainder, g.spec[1:]...)...)
	for _, a := range args {
		if _, y := a.(flag); y {
			debug(pc(ctx,a), "unkown use opts: %v", ts(a), trace{})
		}
	}

	for _, specVal := range specVals {
		l.use_spec(ctx, opts, specVal, args...)
	}
	return
}

func (l ul) files(ctx Context, doc *commentgroup, g *clause_opts, _ int) {
	// CRITICAL FIX: Prevent index out of bounds panic!
	if len(g.spec) == 0 {
		debug(ctx, "missing file specification properties", trace{})
		return // Halt immediately
	} else if len(g.spec) > 1 {
		debug(ctx, "too many properties: %v", g.spec, trace{})
	}

	var p Value
	var patts, paths []Value

	if l.p.tok == SELECT_PROG1 { // e.g., '=>' or '⇒'
		l.p.next(ctx, true) // step forward with spaces skipped
		if l.p.tok == LINEND || l.p.lineComment != nil {
			debug(ctx, "expecting files path after '⇒'", trace{})
		}
		p = l.expr(ctx)
	}

	l.p.spaces(ctx)

	if g.skip { return }

	if t := parseOpts(ctx, &g.general_opts, g.remainder...); t != nil {
		debug(ctx, "unsupported opts: %v", t, trace{})
	}

	// Expand the left-hand side (Patterns) eagerly
	if t := expand(original{ctx,defExpand1}, g.spec[0]); t == nil {
		debug(ctx, "nil file pattern: %v", g.spec[0], trace{})
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
						debug(ctx, "invalid files flag: %v", trace{})
					}
				}
			}
		}
	} else {
		if len(patts) == 1 {
			if f, y := patts[0].(flag); y {
				switch __string(ctx, f.Value) {
				default: // TODO: parse files options
					debug(ctx, "invalid files flag: %v", trace{})
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
	if !g.skip { call(ctx, "assert", g.remainder, g.spec...) }
}

func (p *parser) append(ctx Context, doc *commentgroup, g *clause_opts, _ int) {
	if !g.skip { call(ctx, "append", g.remainder, g.spec...) }
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
	var local map[string]*def
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
					for s, d := range last {
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
				debug(ctx, "unsupported flag: %v", ts(a), trace{})
			}
			continue
		}

		var s = __string(ctx, a)
		if s == "" {
			debug(ctx, "empty local: %v", ts(a), trace{})
		}

		if local == nil { local = make(map[string]*def) }

		var t *def
		if o := l.project.Lookup(s); o != nil {
			if x, y := o.(*def); y { t = new(def); *t = *x }
		}
		local[s] = t
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
			debug(ctx, "unsupport flag: %v (%v)", ts(op), val, trace{})
		}
		return
	}

	prop0 := g.spec[0]
	if isTrivial(prop0) {
		debug(pc(ctx,l.p), "illegal", trace{})
	}

	var name string
	var opts []Value
	if a, y := prop0.(*argumented); y { prop0, opts = a.Value, a.args }
	switch t := prop0.(type) {
	case *delegate:
		for i, x := range merge(expand(_final(ctx),t)) {
			switch t := x.(type) {
			case *pair:
				debug(pc(ctx,l.p), "%v → %v", t.key, t.val)
			case *word:
				if name != "" {
					debug(pc(ctx,l.p), "%v → %d. %v", prop0, i, x, trace{})
				} else {
					name = t.s
				}
			default:
				debug(pc(ctx,l.p), "%v → %d. %v", prop0, i, ts(x), trace{})
			}
		}
		return
	case *pair:
		debug(pc(ctx,l.p), "%v → %v", t.key, t.val)
	default:
		name = __string(ctx, t)
	}

	switch name {
	case "-configuration", "configuration":
		debug(pc(ctx,l.p), "configuration is done at parse time", trace{})
	case "":
		debug(pc(ctx,l.p), "empty eval command", trace{})
	}

	resolved := l.resolve(ctx, prop0, name)
	switch x := resolved.(type) {
	case evaler: x.eval(ctx, opts, expands(_final(ctx), g.spec[1:]...))
	case *builtin:
		switch x.name {
		case "plain": evoke(ctx, x, opts, g.spec[1:])
		}
	default:
		debug(pc(ctx,l.p), "resolved '%s' is not evaler: %v → %s", typeof(resolved), prop0, name, trace{})
	}

	// TODO: if c, y := res.(code); y { ... }
}

func (l ul) directive(ctx Context) (props []Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "spec")) }

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

		debug(ctx, "%v: no specs, remainder: %v", keyword, opts.remainder, trace{})

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
		forids(ctx, expand(_final(ctx), v), func(v Value, _ []Value) { ids = append(ids, v) })
	}

	pos, tok := l.p.pos, l.p.tok
	l.p.next(ctx, true) // the assign token

	// CRITICAL FIX: Parse the RHS exactly once! This prevents parser stream desynchronization 
	// when multiple variables are assigned, or when `?=` skips assignment for already-defined variables.
	rhsVals := l.values(ctx)

	for _, id := range ids {
		var alt object
		var d *def

		switch t := id.(type) {
		case *argumented:
			debug(ctx, "multiple defs: %v, args=%v", t.Value, t.args, trace{})

		case *group:
			debug(ctx, "multiple defs: %v", t.elems, trace{})

		case *arrow:
			if v := expand(_final(ctx), t); v == nil {
				debug(ctx, "%v is nil", ts(t), trace{})
			} else if x, y := v.(*def); !y {
				debug(ctx, "%v is not a def: %v", ts(t), ts(v), trace{})
			} else {
				d = x
			}

		default: // *word, *compound, *qualword, *path, flag:
			name := ident(def_name{ctx}, expand(ctx, t))

			if name == "" {
				debug(pc(ctx,t), "empty name: %s: `%v`", typeof(id), id, callstack{num:32}, trace{})
			} else if _, y := builtins[name]; y {
				debug(pc(ctx,t), "`%v` is a builtin name (%v)", ident, name, trace{})
			}
			if checkpoints { if illegal_name_prefix.MatchString(name) {
				debug(pc(ctx,t), "illegal name: %v", name, callstack{num:32}, trace{})
			}}

			prev := l.project.resolve(ctx, name)

			var isNew bool
			d, isNew = l.project._def(ctx, defInvalid, name)
			if isNew { d.pos = pos } // ensure def pos is correct

			if prev == nil || d == nil {
				// no derived value
			} else if x, y := prev.(*def); !y {
				// not a def
			} else if x == nil {
				debug(ctx, "prev def '%s' is nil", name, trace{})
			} else if x != d && x.scope != d.scope && alt == nil {
				switch tok {
				case ASSIGN_ADD, ASSIGN_SHI:
					if d.o == defVoid && d.o != x.o { d.origin(ctx, x.o) }
					if !isTrivial(x.value) { d.append(ctx, x.value) }
				}
			}
		}

		if d == nil {
			debug(ctx, "def is nil: %v", ts(ident), trace{})
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
				debug(ctx, "unknown: %v %v", _ctx.o, d.name, trace{})
			}
		case ASSIGN_SHI: // =+
			if d.o == defInvalid { d.o = defExpand0 }
			switch _ctx.o = d.o|defAssign2; {
			case d.o&defExpand0 != 0:
				d.val(_ctx, append(rhsVals, d.value))
			case d.o&(defExpand1|defExpand2|defExpand3) != 0:
				d.val(_ctx, append(expands(_ctx, rhsVals...), d.value))
			default:
				debug(ctx, "unknown: %v %v", _ctx.o, d.name, trace{})
			}
		case ASSIGN_SUB: // -=
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
						debug(ctx, "unknown: %v %v", _ctx.o, d.name, trace{})
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
		case ASSIGN_SAD, ASSIGN_SSH: // -+=, -=+
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
				debug(ctx, "unknown: %v %v", _ctx.o, d.name, trace{})
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
			case ASSIGN_SSH: _vals = append(vals, _vals...) // -=+
			}
			d.value = ease(ctx, _vals)
		default:
			debug(ctx, "unknown: %v %v %v", _ctx.o, d.name, tok, trace{})
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
	case "value", "":
		l.p.scanner.pop(isStrcompLine)
		l.p.next(ctx, true) // skip RECIPE or SEMICOLON and parse in list mode
		if p, isList = l.p.pos, true; !l.p.is_end_of_line() {
			var c = p_recipe{ctx, true, nil, nil} // builtin value recipe
			for l.p.tok != EOF && l.p.tok != SEMICOLON && l.p.tok != LINEND && l.p.lineComment == nil {
				elems = append(elems, l.expr(&c))
				if l.p.spaces(ctx); l.p.lineComment != nil { break }
			}
		}

	case "eval":
		l.p.scanner.pop(isStrcompLine)
		l.p.next(ctx, true) // skip RECIPE or SEMICOLON and parse in list mode
		if p, isList = l.p.pos, true; !l.p.is_end_of_line() {
			var x = l.expr(ctx) // parse first expr of recipe

			var a *argumented
			if a, _ = x.(*argumented); a != nil { x = a.Value }
			if x == nil {
				debug(pc(ctx,p), "parsed nil value, dialect=%s", l.p.dialect, trace{})
			}

			if l.p.dialect == "value" {
				// no resolving commands
			} else if t, y := x.(*word); !y {
				// does nothing
			} else if s := l.resolve(ctx, t, t.s); isTrivial(s) {
				debug(pc(ctx,p), "no such symbol: %v, %s → %s; dialect=%s", t.s, ts(x), ts(s), l.p.dialect, trace{})
			} else if _, y := s.(*builtin); !y {
				debug(pc(ctx,p), "'%s' is not a command (%s)", t.s, typeof(s), trace{})
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
					debug(ctx, "unsupported token: %s, %v", l.p.tok, elems, trace{})
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

		switch l.p.dialect { case "plain", "text": isPlainline = true }

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
				debug(ctx, "%v %v", c.elems, c.lines, trace{})
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

// Parsing (var a=xxx,b=yyy) definitions
func (p *parser) var_modifier(ctx Context, args ...Value) (err error) {
	for _, elem := range args {
		var kv, y = elem.(*pair)
		if !y || kv == nil {
			debug(ctx, "bad var form (%v)", ts(elem), trace{})
		}

		var v = kv.val
		if x, y := v.(*group); y { v = x.list() }

		_scope(ctx).def(ctx, defVoid, kv.key, v)
	}
	return
}

func (l ul) define_configs(ctx Context) {
	for _, t := range l.p.targets {
		l.project.def(ctx, defConfig, t)
	}
}

func (l ul) modifier(ctx Context) (res *modifier) {
	l.p.spaces(ctx)

	pos := l.p.pos
	l.p.expect(ctx, LPAREN)
	l.p.spaces(ctx)

	var elems []Value
	var val = l.expr(ctx)
	var name = __string(ctx, val)
	if name == "" {
		debug(pc(ctx,val), "unsupported modifier: %s", ts(name), trace{})
	} else if _, y := dialects[name]; y {
		if l.p.dialect == "" { l.p.dialect = name } else {
			debug(pc(ctx,l.p), "multi-dialects unsupported, already defined '%s'", l.p.dialect, trace{})
		}
	} else if _, y = modifiers[name]; !y {
		debug(pc(ctx,l.p), "no such dialect or modifier: %s", name, trace{})
	}

	for l.p.tok != RPAREN && l.p.tok != EOF {
		l.p.spaces(ctx)

		pos := l.p.pos

		if va := l.values(ctx); name == "var" {
			l.p.var_modifier(ctx, va...)
		} else if n := len(va); n == 1 {
			elems = append(elems, va[0])
		} else if n > 1 {
			elems = append(elems, &list{elements{va}})
		} else {
			elems = append(elems, &null{l.p.valbase(ctx)})
		}

		if l.p.tok == COMMA { l.p.next(ctx, true) }
		if l.p.pos == pos {
			debug(pc(ctx,l.p), "unsupported modifier arg: %v '%v'", l.p.tok, l.p.lit, trace{})
		}
	}

	l.p.expect(ctx, RPAREN)

	if val == nil && len(elems) == 0 {
		debug(pc(ctx,l.p), "empty modifier", trace{})
	}

	res = new(modifier)
	res.pos, res.elems = pos, append([]Value{val}, elems...)
	return
}

// example: {(modifier ...)}
func (l ul) modification(ctx Context) *modification {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "modification")) }

	var vb = l.p.valbase(ctx)
	var elems []*modifier
	for l.p.tok != EOF && l.p.tok != LINEND && l.p.tok != RBRACE {
		if m := l.modifier(ctx); m != nil { elems = append(elems, m) }
	}

	// l.p.expect(ctx, /* RBRACK */RBRACE)

	if len(elems) == 0 {
		debug(ctx, "empty modifier group", trace{})
	}
	if l.p.tok == COLON {
		debug(ctx, "unexpected colon after modifer", trace{})
	}
    return &modification{vb, elems}
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
var rule_autos = map[string]struct{}{
	"@" :struct{}{}, "%" :struct{}{}, "<" :struct{}{}, ">" :struct{}{}, "^" :struct{}{},
	"@D":struct{}{}, "%D":struct{}{}, "<D":struct{}{}, ">D":struct{}{}, "^D":struct{}{},
	"@F":struct{}{}, "%F":struct{}{}, "<F":struct{}{}, ">F":struct{}{}, "^F":struct{}{},
	"@'":struct{}{}, "%'":struct{}{}, "<'":struct{}{}, ">'":struct{}{}, "^'":struct{}{},
	"?" :struct{}{}, "+" :struct{}{}, "|" :struct{}{}, "*" :struct{}{},
	"?D":struct{}{}, "+D":struct{}{}, "|D":struct{}{}, "*D":struct{}{},
	"?F":struct{}{}, "+F":struct{}{}, "|F":struct{}{}, "*F":struct{}{},
	"?'":struct{}{}, "+'":struct{}{}, "|'":struct{}{}, "*'":struct{}{},
	"-" :struct{}{}, "~" :struct{}{}, // "<-":struct{}{}, "->":struct{}{},
}

func (l ul) rule(ctx Context, targets []Value) (result Value) {
	if l_traverse.enabled || debugSyntax(ctx, "rule") { defer un(l_trace(l_traverse, "rule")) }

	ctx = p_rule_ctx{ctx, l.p}

    if l.project != _scope(ctx).project {
		debug(ctx, "mismatched project/scope : %v", targets, trace{})
	}

	// TODO: doc = p.leadComment
	var depends, ordered, recipes []Value
	defer l.closescope(l.openscope(ts(targets)))
	defer func() { l.p.dialect, l.p.ruparas = "", nil } ()

	l.p.dialect = ""
	l.p.ruparas = nil

	// NOTE: expand targets to speed up for later usage, it might spend lots of time in
	// project.entry while matching for entry looked up if not expanded right now.
	targets = expands(_final(ctx), targets...)

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
            debug(ctx, "creating entry failed for %v", target, trace{})
        }

		res = append(res, entry)

        if x, y := entry.destiny().(flag); y && x.Value != nil {
			if prog.project.name != "~" {
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

	var args []Value
	var name = l.expr(ctx)
	if a, y := name.(*argumented); y { name, args = a.Value, a.args }

	t := &template{
		pos: p.pos, tok: p.tok, lit: p.lit,
		state: p.scanner.scanstate,
		name: name, params: args,
	}

	p.spaces(ctx)
	p.linend(ctx)

	var linend = true
	var nested = 0
	for { switch p.tok {
	default:
		linend = false
		p.next(ctx, true)

	case LINEND:
		linend = true
		p.next(ctx, true)

	case DEF:
		if linend { nested += 1 }
		linend = false
		p.next(ctx, true)

	case END:
		if !linend || 0 < nested {
			if linend { nested -= 1 }
			linend = false
			p.next(ctx, true)
			continue
		}

		pos := p.pos
		p.next(ctx, true)
		p.linend(ctx)

		state := p.scanner.scanstate
		t.end, t.endPos = &state, pos
		p.templates = append(p.templates, t)
		return

	case EOF:
		return
	}}
}

func (l ul) foreach_done(ctx Context) {
	if l.p.spaces(ctx); l.p.tok == LINEND {
		debug(ctx, "unexpected end of line", trace{})
	}

	l.p.expect(ctx, FOREACH)
	l.p.spaces(ctx)

	var vals = merge(expands(_final(ctx), l.values(ctx)...)...)
	var t = &template{
		pos:l.p.pos, tok:l.p.tok, lit:l.p.lit, state:l.p.scanner.scanstate,
	}

	l.p.spaces(ctx)
	l.p.linend(ctx)

	var nested = 0
	for l.p.tok != EOF {
		switch pos := l.p.pos; l.p.tok {
		case LINEND:
			l.p.next(ctx, true)

		case FOREACH, FOR:
			nested += 1
			for l.p.tok != EOF {
				if  l.p.next(ctx, true) ; l.p.tok == LINEND {
					l.p.next(ctx, true) ; break
				}
			}

		case DONE:
			if nested > 0 {
				nested -= 1
				for l.p.tok != EOF {
					if  l.p.next(ctx, true) ; l.p.tok == LINEND {
						l.p.next(ctx, true) ; break
					}
				}
				continue
			}

			l.p.next(ctx, true) // done
			l.p.linend(ctx)

			state := l.p.scanner.scanstate
			t.end, t.endPos = &state, pos

			defer func(s Pos) { l.p.stop = s } (l.p.stop)
			l.p.stop = t.endPos

			ac := automatic{Context:ctx, defs:make(def_map)}
			for _, val := range vals {
				if !isTrivial(val) {
					if x, y := val.(*defcaps); y {
						ac.set(&ac, defStatic, "_", x.Value)
						for _, c := range x.caps {
							ac.set(&ac, defStatic, c.name, c.value)
						}
					} else {
						ac.set(&ac, defStatic, "_", val)
					}
					l.codeblock(&ac, FOREACH, t)
				}
			}
			return

		default:
			for l.p.tok != EOF {
				if  l.p.next(ctx, true) ; l.p.tok == LINEND {
					l.p.next(ctx, true) ; break
				}
			}
		}
	}
}

func (l ul) for_done(ctx Context) {
	if l.p.spaces(ctx); l.p.tok == LINEND {
		debug(ctx, "unexpected end-of-line", trace{})
	}

	var opts struct{
		skipNil bool `skip-nil,skip-null,skipnil,skipnull,no-nil,no-null`
	}
	if  l.p.expect(ctx, FOR) ; l.p.tok == LPAREN {
		l.p.next(ctx, true) // LPAREN
		if vals := parseOpts(ctx, &opts, l.values(ctx)...); vals != nil {
			debug(ctx, "unexpected opts: %v", vals, trace{})
		}
		l.p.expect(ctx, RPAREN)
	}

	l.p.spaces(ctx)

	type  param struct{ name string ; elems []Value }
	type nparam struct{ p Pos ; a []*param ; n int }

	var params []*nparam
	var ac = automatic{Context:ctx, defs:make(def_map)}
	for l.p.spaces(ctx); l.p.tok != EOF && !l.p.is_end_of_line(); l.p.spaces(ctx) {
		if l.p.tok == AND && params == nil {
			debug(pc(ctx,l.p), "unexpected 'and'", trace{})
		} else if l.p.tok == AND || params == nil {
			params = append(params, &nparam{p:l.p.pos})
			if l.p.tok == AND { l.p.next(ctx, true); continue }
		}

		var pars = make(map[string]*param)
		var p = params[len(params)-1]
		for i, a := range merge(expand(&ac, l.expr(&ac))) {
			switch x := unbox(a).(type) {
			case *null: continue
			case *pair:
				var name = __string(ctx, x.key)
				if name == "" {
					debug(pc(ctx,a), "empty key %v", ts(x.key), trace{})
				}

				var par *param
				if pt, ok := pars[name]; ok {
					par = pt
				} else {
					par = new(param)
					par.name = name
					p.a = append(p.a, par)
					pars[name] = par
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
					if t, y := pars[cap.name]; y {
						t.elems = append(t.elems, cap.value)
					} else {
						t = &param{cap.name, []Value{cap.value}}
						p.a = append(p.a, t)
						pars[cap.name] = t
					}
					// CRITICAL FIX: Ensure p.n is updated for captured regex groups!
					if n := len(pars[cap.name].elems); n > p.n { p.n = n }
				}

			default:
				debug(pc(ctx,a), "unexpected %v ; %d. %v", ts(a), i, ac.defs, trace{})
			}
		}
	}

	var t = &template{pos:l.p.pos, tok:l.p.tok, lit:l.p.lit, state:l.p.scanner.scanstate}

	l.p.spaces(ctx)
	l.p.linend(ctx)

	var nested = 0
	for l.p.tok != EOF {
		switch pos := l.p.pos; l.p.tok {
		case LINEND:
			l.p.next(ctx, true) 

		case FOR, FOREACH:
			nested += 1
			for l.p.tok != EOF {
				if  l.p.next(ctx, true) ; l.p.tok == LINEND {
					l.p.next(ctx, true) ; break
				}
			}

		case DONE:
			if nested > 0 {
				nested -= 1
				for l.p.tok != EOF {
					if  l.p.next(ctx, true) ; l.p.tok == LINEND {
						l.p.next(ctx, true) ; break
					}
				}
				continue
			}

			l.p.next(ctx, true) // done
			l.p.linend(ctx)

			defer func(s Pos) { l.p.stop = s } (l.p.stop)

			t.end, t.endPos, l.p.stop = &l.p.scanner.scanstate, pos, pos

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
							ac.set(&ac, defStatic, a.name, a.elems[i])
						} else if opts.skipNil {
							continue outer
						} else {
							ac.set(&ac, defStatic, a.name, &null{valbase{_p.p}})
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
					l.codeblock(&ac, FOR, t)
				}
			}
			return

		default:
			for l.p.tok != EOF {
				if  l.p.next(ctx, true) ; l.p.tok == LINEND {
					l.p.next(ctx, true) ; break
				}
			}
		}
	}
}

var pprofCounter int

func (l ul) codeblock(ctx *automatic, op token, t *template) {
	l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate = t.pos, t.tok, t.lit, t.state

	if false && checkpoints {
		pprofCounter += 1
		defer cpu_profile(ctx, fmt.Sprintf("template-%05d.prof", pprofCounter), true)()
	}

	if !(l.p.pos < l.p.stop) {
		debug(ctx, "bad range: [%v %v) (%v)", l.p.pos, l.p.stop, t.name, trace{})
	}

	var c = codeblock{ctx, op}

	d_variant_target = false//strings.HasSuffix(l.project.spec, "modules/variant/.target")
	for l.p.tok != EOF && l.p.pos < l.p.stop {
		if l.p.tok == SPACE || l.p.tok == LINEND || (l.p.tok == COMMENT && l.p.lineComment != nil) {
			l.p.next(ctx, true)
		} else {
			if d_variant_target {
				if t.name != nil {
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
			debug(ctx, "%v", ts(e), trace{})
		} else if e = pprof.StartCPUProfile(fCpu); e != nil {
			fCpu.Close()
			debug(ctx, "%v: %v", profCpu, e, trace{})
		}
		defer func() {
			pprof.StopCPUProfile()
			fCpu.Close()

			var fMem, e = os.Create(profMem)
			if e != nil {
				debug(ctx, "%v", e, trace{})
			}

			runtime.GC() // update memory statistics
			e = pprof.WriteHeapProfile(fMem)
			fMem.Close()

			if e != nil {
				debug(ctx, "%v: %v", profMem, e, trace{})
			}
		} ()
	}

	defer func(t time.Time, pos Pos, tok token, lit string, state scanstate) {
		l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate = pos, tok, lit, state
	} (time.Now(), l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate)

	var ac = automatic{Context:ctx, defs:make(def_map)}

	for i, v := range t.params {
		if s := __string(ctx, v); s != "" {
			if i < len(params) {
				v = params[i]
			} else {
				v = _null(v.Pos())
			}
			ac.set(&ac, defStatic, s, v)
		} else {
			debug(ctx, "empty template param name: %v", ts(v), trace{})
		}
	}

	l.codeblock(&ac, LPAREN, t)
}

func (l ul) call(ctx Context, name Value, args []Value) (result bool) {
	for _, t := range l.p.templates {
		if t.name != nil && eq(ctx, t.name, name) {
			stop := l.p.stop
			l.p.stop = t.endPos
			l.repeat(ctx, t, args)
			l.p.stop = stop
			return true
		}
	}

	debug(ctx, "undefined template: %v", name, trace{})
	return
}

func (l ul) saveConfiguration(ctx Context) {
	if l.project == nil { debug(ctx, "nil project", callstack{num:32}, trace{}) }

	var configs = l.project.configs
	if configs == nil { return }

	var f = l.project.configuration_sm(ctx)

	// =========================================================
	// I/O OPTIMIZATION & STATE ENFORCEMENT
	// =========================================================
	if f != nil && f.filebase != nil {
		if f._dirty == 0 && f.exists() {
			// Enforce strict state transitions: Catch unwarranted saves!
			// debug(pc(ctx,f.fullname()), "%s: conflicted configuration.sm", l.project.name, trace{})
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
			debug(pc(pc(ctx,fn),c.fullname()), "%s: configuration already loaded", l.project.name, trace{})
		}
	}

	if e := os.MkdirAll(filepath.Dir(fn), os.FileMode(0755)); e != nil {
		debug(pc(ctx,fn), "make path %s failed: %v", filepath.Dir(fn), e, trace{})
	}

	// =========================================================
	// ATOMIC WRITE: Protect against concurrent '1:1: syntax error'
	// =========================================================
	var tmpFn = fn + ".tmp"
	var o, e = os.OpenFile(tmpFn, os.O_RDWR | os.O_CREATE | os.O_TRUNC, os.FileMode(0600))
	if e != nil {
		debug(pc(ctx,fn), "%s: %v", l.project.name, e, trace{})
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

func (l ul) configure_par(ctx Context, _op Value) (op Value, par map[string]Value) {
	var args []Value

	op, par = _op, make(map[string]Value)

	if x, y := op.(*argumented); y {
		if f, y := x.Value.(flag); y {
			op = f.Value
		} else {
			debug(pc(ctx,x.Value), "wrong configure word: %v", ts(x.Value), trace{})
		}
		args = xmerge(_final(ctx), x.args...)
	}

	for _, arg := range args {
		switch t := arg.(type) {
		case *pair:
			par[__string(ctx, t.key)] = t
		case *raw, *strlit, *strval, *strcomp:
			par["INFO"] = &pair{_word(t.Pos(),"INFO"), t}
		default:
			if !isTrivial(arg) {
				debug(pc(ctx,arg), "wrong arg: %s", ts(arg), trace{})
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

func (l ul) configure_val(ctx *execution, _op, op, val Value, par map[string]Value) (res Value) {
	var x, y = op.(*word)
	if !y {
		debug(pc(ctx,_op), "wrong configure word: %v %v %v", ts(_op), ts(op), ts(val), trace{})
	}

	switch x.s {
	case "answer":
		if val == nil { return _answer(op.Pos(), false) }
		return _answer(val.Pos(), __true(ctx, val))
	case "bool", "boolean":
		if val == nil { return _boolean(op.Pos(), false) }
		return _boolean(val.Pos(), __true(ctx, val))
	case "value":
		if val == nil { return _null(op.Pos()) }
		return expand(_final(ctx),val)
	}

	if l.project.configure == nil {
		debug(pc(ctx,op), "wrong configure: %v %v", ts(op), ts(val), trace{})
	}

	var ops = l.project.configure._entries(ctx, _op, false)
	if ops == nil {
		debug(pc(ctx,_op), "no configure ops: %v", _op, trace{})
	}

	var vals []Value
	for _, ent := range ops {
		var params []Value
		for _, prog := range ent.programs() {
			for _, p := range prog.params {
				w := _word(p.Pos(), ident(ctx, p))
				if x, y := par[w.s]; y {
					params = append(params, x)
				} else {
					switch w.s {
					case "TARGET": params = append(params, &pair{w, auto_get(ctx, "@")})
					case "VALUE":  params = append(params, &pair{w, val})
					case "LANG", "LANGUAGE":
						if ctx.language != "" {
							params = append(params, &pair{w, _word(w.pos, ctx.language)})
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
		forids(ctx, expand(_final(ctx),v), func(v Value, _ []Value) { ids = append(ids, v) })
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
				debug(pc(ctx,t), "needs cond value", trace{})
			}
		}

		if _op != nil {
			debug(pc(ctx,t), "configure op already defined: %v", _op, trace{})
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
			d, _ := exe.set(&exe, defVoid, "@", id)
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
			l.p.dialect = ""

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

			if x, y := par["INFO"]; y {
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
			d, _ := exe.set(&exe, defVoid, "@", id)
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
			l.p.dialect = ""

			var deps, vals []Value

		depsloop:
			for {
				switch l.p.tok { case SEMICOLON, LINEND, EOF: break depsloop }
				deps = append(deps, l.expr(cc)) ; l.p.spaces(ctx)
				exe.set(&exe, defVoid, "<", deps[0])
				exe.set(&exe, defVoid, ">", deps[len(deps)-1])
				exe.set(&exe, defVoid, "^", ease(ctx, deps))
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

			if x, y := par["INFO"]; y {
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

			var val = auto_get(&exe, "-")
			if val == nil && exe.recipes != nil && len(exe.interpreted) == 0 {
				if x, y := dialects[""]; y && x != nil {
					val = exe.interpret(cc, x, nil)
				}
			}

			if op != nil { val = l.configure_val(&exe, _op, op, val, par) }
			if val != nil { vals = append(vals, val) }

			for _, a := range exe.defers {
				if x, y := a.(*group); y {
					modify(ctx, x, true)
				} else {
					debug(pc(ctx,a), "defer: not a modifier: %s", ts(a), trace{})
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
		debug(pc(ctx,l.p), "%v: wrong configure", op, trace{})
	}
}

var d_variant_target bool
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
		debug(ctx, "unexpected %v", t, trace{})
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
			l.call(ctx, x.Value, x.args)
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
func (l ul) declareNew(ctx Context, pos Pos, name, filename string, opts *project_opts) (d *declare) {
	if x, y := l.declares[name]; y { return x }

	var sco = l.scope()
	var relPath = __string(ctx, sco.finddef(".")) // CRD
	var tmpPath = __string(ctx, sco.finddef(",")) // CTD
	var absPath string
	if x, y := do(ctx, abs_path{}).(string); y {
		absPath = x
	} else {
		absPath = __string(ctx, sco.finddef("/"))
	}

	var spec, _ = filepath.Rel(workBaseDir, absPath)

	if l.declares == nil { l.declares = make(map[string]*declare) }

	d = &declare{
		project: &project{
			pos: pos,
			absPath:  absPath,
			tmpPath:  tmpPath,
			rel:      relPath,
			spec:     spec,
			name:     name,
			opt:      *opts,
			use:      new(uselist),
		},
	}

	l.declares[name]  = d
	l.globe.loaded[d.absPath] = d.project

	// CRITICAL: Bubble up the declaration to parent{...} wrappers!
	do(ctx, declared_project{d.project})
	
	d.p = l.p
	d.s = l.loader.scope

	// CRITICAL: The scope must be initialized so valcache and wildcard work!
	d.scope = newscope(sco, d.project, name)
	d.scope.elems[".self"] = self{d.project}
	d.scope.elems[".usee"] = d.use
	d.use.owner_ = d.project
	d.use.scope = d.scope
	d.use.name = "usee"

	if l.globe.main == nil && spec != "" && name != "@" && name != "~" {
		for sco != nil && sco != l.globe.scope {
			if p := sco.project; p != nil && d.name == "@" {
				return
			}
			sco = sco.outer
		}
		l.globe.main = d.project
	}
	return
}

func (l ul) declare(ctx Context, ident Value, name, filename string, declOpts *project_opts) bool {
	if name == "@" {
		debug(ctx, "deprecated project name: @", trace{})
	}

	if _, o := l.find(name); o != nil { switch o.(type) {
	case *builtin: debug(ctx, "%v is a builtin, can't be project name", o, trace{})
	}}

	var prev = l.loader // nil if newly declared
	var dec = l.declareNew(ctx, ident.Pos(), name, filename, declOpts)
	if prev == nil || dec.project != prev.project {
		if prev != nil && prev.project != nil && prev.project != dec.project {
			if prev.project.name == dec.project.name {
				debug(ctx, "%s", prev.project.name, trace{})
			}
		}
		l.project, l.loader.scope = dec.project, dec.scope
	}

	if ll := _loader(l.loader.Context); ll != l.loader && ll == prev {
		if _, a := ll.project.projectname(ctx, name, dec.project); a != nil {
			if x, y := a.(*project); !y || x != dec.project {
				debug(ctx, "%v: name already taken : %v", name, ts(a), trace{})
			}
		}
	}

    if l.globe.main != nil && l.globe.main == l.project && l.project.name != "~" {
        for _, t := range l.globe.pairs {
            switch k := t.key.(type) {
            case *word, *compound:
                l.scope().def(ctx, defDecl, k, t.val)
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
					l.scope().def(ctx, defDecl, k, t.val)
				case flag:
					if false { debug(ctx, "unknown flag : %v", t) }
				default:
					debug(ctx, "unknown target : %v", ts(t))
				}
			}
		}
	}

    if err := l.loadPlugin(ctx); err != nil {
        debug(ctx, "load plugin failed: %v", err, trace{})
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
			debug(ctx, "unknown set: %v", ts(a), trace{})
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
				debug(ctx, "recursive derivation: %v ⇔ %v", ts(p.project), ts(t.project), trace{})
			}
		}

		if p.has_base(p.project) {
			debug(ctx, "duplication derivation: %v ⇔ %v", ts(p.project), ts(t.project), trace{})
		}

		if len(p.bases) == 0 {
			p.projectname(ctx, ".base", t.project)
		}

		p.bases = append(p.bases, t.project)
		return
	}
	return p.Context.do(ctx, op)
}

func (l ul) projectStart(ctx Context, filename string, isMainFile bool) (_ Value, _ string, _ bool) {
	l.p.next(ctx, true) // aka. the keyword

	var vals []Value
	for l.p.tok == MINUS {
		val := l.expr(ctx)
		l.p.spaces(ctx)

		if a, y := val.(*argumented); y {
			if f, y := a.Value.(flag); y {
				if w, y := f.Value.(*word); y {
					l.pre_project(ctx, w.s, a.args...)
					continue
				}
			}
		}

		vals = append(vals, val)
	}

	var opts project_opts
	if a := parseOpts(ctx, &opts, vals...); len(a) > 0 {
		debug(pc(ctx,filename), "unknown project option %v", ts(a), trace{})
	}

	var ident Value
	var implicitBase string // aka. foo.bar.Baz implicitly load base 'foo/bar'

	if l.p.tok == LPAREN || l.p.is_end_of_line() {
		var dir = filepath.Dir(filename)
		if l.project != nil && l.project.absPath == dir {
			ident = _word(l.p.pos, l.project.name)
		} else if s := filepath.Base(filename); s == dot_base || s == dot_configure {
			// NOTE: loading the .base or .configure file
			ident = _word(l.p.pos, s)
		} else if s := filepath.Base(dir); s != "" {
			// TODO: validate basename as a valid identifier
			ident = _word(l.p.pos, s)
		} else {
			debug(ctx, "invalid file: %v", filename, trace{})
		}
	} else if l.p.tok == TILDE { // `project ~`
		if ext := filepath.Ext(filename); ext != ".smart" {
			debug(ctx, "`%v` not a smart file", filepath.Base(filename), trace{})
		} else if s := strings.TrimSuffix(filepath.Base(filename), ext); s == "" {
			debug(ctx, "`%v` not tilde name", filepath.Base(filename), trace{})
		} else {
			ident = _word(l.p.pos, s)
		}
		l.p.next(ctx, true) // skip tilde
	} else { // bare `project`
		base, qw := makePath(), &qualword{}

		for l.p.tok != EOF && l.p.tok != SPACE {
			var w = l.p.bare(ctx)
			qw.elems = append(qw.elems, w)
			if l.p.tok == DOT {
				l.p.step(ctx) // skips '.'
				base.elems = append(base.elems, w)
			} else {
				break
			}
		}

		l.p.spaces(ctx)

		switch qw.len() {
		case 0:
			debug(pc(ctx,qw), "package name is empty (tok=%v)", l.p.tok, trace{})
		case 1:
			ident = qw.elems[0]
		default:
			ident = qw
		}

		if 0 < base.len() {
			implicitBase = __string(ctx, base)
		}
	}

	var name = __string(ctx, ident)

	if name == "-" || name == "_" {
		debug(ctx, "package name '%s' is preserved", name, trace{})
	}

	if p := l.project; p != nil && p.name != name {
		debug(ctx, "%v: multiple projects in the directory : %v", p, ident, trace{})
	}

	var _, prevDeclared = l.declares[name]
	if l.declare(ctx, ident, name, filename, &opts) {
		isMainFile = isMainFile && !prevDeclared;
	}

	if cc := (parent{ctx, l.project}); l.p.tok != LPAREN {
		l.bases(cc, implicitBase) // for special bases, e.g. .base
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
		l.container(ctx, ident, name)
	}
	return ident, name, isMainFile
}

func (l ul) close_project(ctx Context, name string) {
    var x, y = l.declares[name]

	if !y || x == nil {
		debug(ctx, "undeclared project: %v", name, trace{})
	}

    if l.project == nil {
        debug(ctx, "current project unset", trace{})
    }

    if l.project.name != name {
        debug(ctx, "current project is %s, not %s", l.project, name, trace{})
    }

    if l.project != x.project {
        debug(ctx, "project conflicts (%v, %v)", l.project, x.project, trace{})
    }

    l.p, l.loader.scope = x.p, x.s
}

func (l ul) parse(ctx Context, filename string) (_ bool) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "file '"+filename+"'")) }
	if l.traceLaunch { defer un(l_trace(l_launch, "parse_file")) }
	if checkpoints {
		if s := l.p.scanner.file.Name(); filename != s {
			debug(ctx, "%v: %s != %s", l.project, filename, s, trace{})
		}
	}

	if l.p.tok == EOF {
		debug(pc(ctx,l.p), "early end of file", callstack{num:64}, trace{})
		return
	}

	var abs string
	var isMainFile bool // aka do.smart, build.smart
	var flatmode = truly(ctx, is_flat_mode{})

	if flatmode {
		if l.project == nil {
			debug(pc(ctx,filename), "nil project", callstack{num:64}, trace{})
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
		debug(ctx, "%v: nil scope: %v", l.project, s, trace{})
	}

	defer l.closescope(l.openscope(bases(2, filename, true)))

	if flatmode {
		if l.p.tok == PROJECT {
			debug(pc(ctx,l.p), "project is forbidden in flat file", trace{})
		}
	} else {
		// CWD: Current Work Directory,     TODO: use $:cwd:
		// CTD: Current Temp Directory,     TODO: use $:ctd:
		// CRD: Current Relative Directory, TODO: use $:crd:
		var s = l.scope()
		if d := s.def(ctx, defVoid, "/", _pathStr(ctx, abs)); d != nil { s.alias(ctx, d, "CWD") }
		if d := s.def(ctx, defVoid, ".", _pathStr(ctx, rel)); d != nil { s.alias(ctx, d, "CRD") }
		if d := s.def(ctx, defVoid, ",", _pathStr(ctx, tmp)); d != nil { s.alias(ctx, d, "CTD") }
		if l.p.tok == PROJECT {
			var name string
			var prev = l.project
			_, name, isMainFile = l.projectStart(ctx, filename, isMainFile)
			if prev != l.project { defer l.close_project(ctx, name) }
		} else {
			debug(pc(ctx,l.p), "expect keyword 'project', not %v", l.p.tok, trace{})
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

    declares map[string]*declare

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
    if l.project != nil { s += " " + l.project.name }
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
    unique bool `uni,uniq,unique`
    remainder []Value // will be opts for unique
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
    if d.append(ctx, vals...); uo.unique {
        d.value = call(ctx, "unique", uo.remainder, merge(d.value)...)
    }
}
func usefor(ctx Context, user *project, f func(usevar, Value, Value, string)) {
    if o := user.resolve(ctx, "use.*"); o != nil {
        if d, y := o.(*def); y && d != nil {
            for _, spec := range merge(d.value) {
                var op usevar
                var val = spec
                if x, y := spec.(*argumented); y {
                    val = x.Value
                    op.remainder = parseOpts(_final(ctx), &op, x.args...)
                }
                if name := __string(ctx, val); name == "" {
                    if c := user.configure; c != nil {
                        note(ctx, "%v", ts(c.resolve(ctx, "use.*")))
                    }
                    debug(pc(ctx,o), "%v: empty use spec: %v", user, ts(spec), trace{})
                } else {
                    f(op, spec, val, name)
                }
            }
        }
    }
}
func (l ul) usevars0(ctx Context, user, usee *project) {
    usefor(ctx, user, func(op usevar, spec, val Value, name string) {
        var prefix string
        if m := name_prefix.FindStringSubmatch(name); m != nil {
            prefix, name = m[1], m[3]
        }

        var useDef *def
        if o := usee.Lookup(prefix+"use."+name); o != nil {
            if d, y := o.(*def); y && d != nil {
                useDef = d
            } else {
                debug(ctx, "use.%s: nil def: %T %v", name, o, o, trace{})
            }
        }
        if useDef == nil { return }

        var dd []*def

        // 1. use.XXX += $(use.XXX)
        {
            d, isNewDef := user._def(ctx, defVoid, useDef.ident(ctx))
            if isNewDef || isTrivial(d.value) {
                dd = append(dd, nonTrivialDefsFromBase(ctx, user, useDef.ident(ctx))...)
            }
            op.apply(closure_with(ctx, usee.scope), d, append(dd, useDef)...)
        }

        if useDef.value == nil || isTrivial(useDef.value) { return }

        // 2. XXX += $(use.XXX)
        {
            d, isNewDef := user._def(ctx, defVoid, name)
            if isNewDef && false {
                if dd == nil { dd = append(dd, nonTrivialDefsFromBase(ctx, user, useDef.ident(ctx))...) }
                dd = append(dd, nonTrivialDefsFromBase(ctx, user, name)...)
            }
            op.apply(closure_with(ctx, user.scope), d, append(dd, useDef)...)
        }
    })
    if false { debug(ctx, "%v ⇒ %v ; %v", user, usee, user.resolve(ctx, "use.*")) }
}
func (l ul) usevars(ctx Context, user, usee *project) {
	usefor(ctx, user, func(op usevar, spec, val Value, name string) {
		var prefix string
		if m := name_prefix.FindStringSubmatch(name); m != nil {
			prefix, name = m[1], m[3]
		}

		// OPTIMIZATION: Look up the variable directly without the "use." prefix.
		// This flawlessly maps user requests (like `-f`) directly to the usee's `-f`.
		var useDef *def
		if o := usee.Lookup(prefix + name); o != nil {
			if d, y := o.(*def); y && d != nil {
				useDef = d
			} else {
				debug(ctx, "use.%s: nil def: %T %v", name, o, o, trace{})
			}
		}
		if useDef == nil {
			return
		}

		var dd []*def

		// 1. Inherit the actual variable and apply configuration modifiers (like -unique)
		{
			d, isNewDef := user._def(ctx, defVoid, useDef.ident(ctx))
			if isNewDef || isTrivial(d.value) {
				dd = append(dd, nonTrivialDefsFromBase(ctx, user, useDef.ident(ctx))...)
			}
			op.apply(closure_with(ctx, usee.scope), d, append(dd, useDef)...)
		}
	})
}

func nonTrivialDefsFromBase(ctx Context, p *project, name string) (dd []*def) {
    for _, base := range p.bases {
        d, y := base.resolve(ctx, name).(*def)
        if y && d != nil && !isTrivial(d.value) {
            dd = append(dd, d)
        }
    }
    return
}

func (l ul) scope() *scope { return l.loader.scope }
func (l ul) search(ctx Context, spec string) (absPath string, isDir bool) {
    if checkpoints && l.project != nil && l.project.name == "variant.bootstrap" {
        defer func() {
            if absPath == "" {
                debug(ctx, "%v → %s %v", spec, absPath, isDir)
            }
        } ()
    }

    if spec == "." {
        debug(ctx, "self-search is not possible", trace{})
    } else if filepath.IsAbs(spec) {
        var s = spec
        if x, y := os.Stat(s); y == nil { return s, x.IsDir() }

        s = spec + ".smart"
        if x, y := os.Stat(s); y == nil { return s, x.IsDir() }

        s = spec + ".sm"
        if x, y := os.Stat(s); y == nil { return s, x.IsDir() }
    } else if spec == "~" || strings.HasPrefix(spec, "~") {
        debug(ctx, "%v : wrong spec : %s (tilde not allowed)", l.project, spec, trace{})
    } else if spec == ".." || has_prefix(spec, "."+pathSep, ".."+pathSep) {
        var s = spec
        var sx string

        if t := l.project.absPath; t != "" {
            if x, e := os.Stat(t); e != nil {
                debug(ctx, "%v", e, trace{})
            } else if !x.IsDir() {
                t = filepath.Dir(t)
            }

            sx = filepath.Join(t, s)

            if x, e := filepath.Abs(sx); e != nil {
                debug(ctx, "%v", e, trace{})
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
        debug(pc(ctx,specVal), "empty spec: %v", ts(specVal), trace{})
    } else if absPath, isDir = l.search(ctx, spec); absPath == "" {
        debug(pc(ctx,specVal), "missing `%s` (in %v)", spec, l.paths, trace{})
    } else {
        loaded, y = l.globe.loaded[absPath]

        for ll := _loader(l.loader.Context); ll != nil; ll = _loader(ll.Context) {
            if ll.project.absPath == absPath {
                debug(pc(ctx,specVal), "%s: loop detected", l.project, trace{})
            }
        }
    }

    defer func() {
        if loaded == nil {
            if false {
                debug(ctx, "%v not loaded (%v,dir=%v)", spec, absPath, isDir, trace{})
            }
            return
        }

        var scope = l.project.scope
        if p, _ := scope.Lookup(loaded.name).(*project); p == nil {
            if _, alt := scope.projectname(ctx, loaded.name, loaded); alt != nil {
                if p, y := alt.(*project); !y || p == nil {
                    debug(ctx, "%s: name already taken : %s", loaded.name, ts(alt), trace{})
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
            if loaded != nil { name = loaded.name }
            prompt(ctx, "%s├┴─\"%s\" ⇢ %s %s\n", l.verpre, spec, name, ds)
        } (time.Now())
    }

    if loaded != nil && !(/*opts.noVars || */opts.reuse) {
        if proj, res, isb := l.project.has_loaded(ctx, loaded, traveUseLoop) ; isb {
            // NOTE: proj could be nil
            prompt(ctx, "%v: %v is already a base\n", l.project, spec)
            debug(ctx, "`%s` is already a base (proj=%s)", spec, proj)
            debug(ctx, "%v", ctx, trace{})
        } else if res {
            // NOTE: proj could be nil
            prompt(ctx, "%v: %v already imported by %v\n", l.project, spec, proj)
            debug(ctx, "'%s' already imported by '%s'", spec, proj)
            debug(ctx, "%v", ctx, trace{})
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
            debug(ctx, "%s not loaded (%s)", spec, absPath, trace{})
        }
        if loaded == l.project {
            debug(ctx, "%v : overwrote by %v (dir=%v)", prev, loaded, isDir, trace{})
        }
    }

    if checkpoints && prev != l.project {
        debug(ctx, "active project changed: %v → %v, use %v", prev, l.project, loaded, trace{})
    }

    // Check against the current load list before appending loaded.
    for _, use := range l.project.use.list {
        var up = use.project
        if loaded == up {
            if !opts.noVars && !opts.files {
                debug(ctx, "%v: using `%s` multiple times: %v", l.project, spec, l.project.use.list, trace{})
            }
            return
        }

        var proj *project
        var res, isb bool
        if proj, res, isb = loaded.has_loaded(ctx, up, traveUseLoop); isb {
            if !l.project.has_base(up) {
                debug(ctx, "`%s` is already a base", spec, trace{})
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
            warn(ctx, "`%s` has already been imported by `%s` (from %s)", loaded, up, proj)
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
        debug(ctx, "current project is nil", trace{})
    }

    var g = _stat(ctx, "smart.go", l.project)
    if g == nil { return /* smart.go was not presented */ }

    var src = __string(ctx, g)
    s := strings.Replace(l.project.rel, "..", "_", -1)
    s = filepath.Join(filepath.Dir(joinTmpPath(ctx, "", "")), "plugins", s)

    var build = true

    so := _stat(ctx, /*l.project.name*/"plugin", stat_dir{s}, stat_nonexist{true})
    if s = so.fullname(); s == "" {
        debug(ctx, "file '%v' has empty fullname", so)
    } else if so.exists() && !l.buildPlugins {
        if so.info.ModTime().After(g.info.ModTime()) {
            build = false // Plugin already updated.
        }
    }
    if build { err = l.buildPlugin(ctx, s, src) }
    if err != nil { return }

    // Once plugin is opened, there's no need/way to close it.
    if l.project.ext.Plugin, err = plugin.Open(s); err == nil {
        var sym plugin.Symbol
        if sym, err = l.project.ext.Lookup("Init"); err != nil {
            debug(ctx, "nil plugin symbol Init", trace{})
        }
        if sym == nil {
            return // no initialization (optional)
        }
        switch init := sym.(type) {
        case func(Context) (error):
            if err = init(ctx); err == nil {
                return
            } else {
                debug(ctx, "plugin Init: %v", err, trace{})
            }
        default:
            debug(ctx, "wrong plugin Init: %T", sym, trace{})
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
        debug(ctx, "%v: cannot use itself", proj, trace{})
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
            debug(ctx, "empty string: %v", ts(specVal), trace{})
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
		debug(ctx, "unknown opts: %v", va[0], callstack{num:5}, trace{})
	}
	if len(g.spec) < 1 {
		debug(ctx, "expect include file: %v", g.spec, callstack{num:5}, trace{})
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
            debug(ctx, "%v: include entry failed : %s", x, ts(z), trace{})
        }

        val = x.target
    }

    var f, spec, fullname = l.spec_file(ctx, val)
    if (f == nil || !f.exists()) && opts.ifExists {
        return // ignore non-exists files
    }

    if spec == "" || fullname == "" {
        debug(ctx, "empty string: %v", ts(val), trace{})
    } else {
        var p, s = val.Pos(), l.trimSpecPath(ctx, spec)
        l.source(p_include{ctx, opts, p, s}, fullname, nil)
    }
    return
}

func (l ul) openscope(comment string) *scope {
    if false && l.traceLaunch { defer un(l_trace(l_launch, "openscope")) }

    var t = &term{} ; *t = l.term
    l.term = term{t, newscope(l.scope(), l.project, comment)}
    return t.scope
}

func (l ul) closescope(s *scope) {
    if false && l.traceLaunch { defer un(l_trace(l_launch, "closescope")) }
    if x, y := l.term.Context.(*term); y {
        var ctx Context = l.loader
        if l.p != nil { ctx = pc(l.loader, l.p.pos) }
        if x == &l.term {
            debug(ctx, "conflict term: %s", x.comment, trace{})
        }
        if x.scope != s {
            debug(ctx, "conflict scope: %s != %s", x.comment, s.comment, trace{})
        }
        l.term = *x
    }
}

// project example (base(var=value))
func (l ul) bases(ctx Context, implicitBase string, params ...Value) {
    if l.traceLaunch { defer un(l_trace(l_launch, "ul.bases")) }

    // For &(foobar) set from command line args
    if true { ctx = closure_with(ctx, l.scope) }

    var implicitBases []Value

    if f := _stat(ctx, dot_base, l.project) ; f != nil {
        if !f.info.IsDir() && (l.project.spec == dot_base /*|| l.project.spec == dot_configure*/) {
            // skip the regular file .base to avoid self loading recursively
        } else {
            implicitBases = append(implicitBases, f)
        }
    }

    if ss := strings.Split(l.project.name, ".") ; len(ss) > 2 && ss[len(ss)-1] == "base" {
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
                segs = append(segs, _word(_pos(ctx), s))
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
            debug(pc(ctx,x), "use -set(%v) instead", x, trace{})
        }

        var spec string
        var specVal Value
        if specVal = expand(_final(ctx), elem); specVal == nil { specVal = elem }

        if spec = __string(ctx, specVal) ; spec == "" {
            debug(ctx, "%v: empty base name: %v", l.project, ts(specVal), trace{})
        } else if strings.Contains(spec, "//") {
            note(ctx, "%v: invalid spec: %v → %v", l.project, elem, specVal)
            note(ctx, "%v: invalid spec: %v → %v", l.project, elem, spec)
            debug(ctx, trace{})
        } else if implicitBase != "" && spec == implicitBase {
            if i == implicitIndex {
                ctx = load_implicit{ctx}
            } else {
                debug(ctx, "%v: implicit base '%v' already loaded", l.project, elem, trace{})
            }
        }

        var abs string
        var isDir bool

        if x, y := to_file(elem); y && x.info != nil {
            abs, isDir = x.fullname(), x.info.IsDir()
        } else {
            abs, isDir = l.search(ctx, spec)
        }

        for _, base := range l.project.bases {
            if base.absPath == abs {
                debug(ctx, "duplicated base: %v : %v → %v (in %v)", base, elem, spec, trace{})
                continue paramsloop
            }
        }

        if cc := _abs_ctx(ctx, abs); isDir {
            l.directory(cc, spec, abs, nil)
        } else {
            l.file(cc, spec, abs, nil)
        }
    }

	usefor(ctx, l.project, func(op usevar, _, _ Value, name string) {
		var prefix string
		if m := name_prefix.FindStringSubmatch(name); m != nil {
			prefix, name = m[1], m[3]
		}

		// OPTIMIZATION: Prevent polluting project.elems with ghost variables.
		// We drop the "use." prefix and pull the base's true variable natively.
		var us = prefix + name //prefix+"use."+name

		// ====================================================================
		// OPTIMIZATION: Base Inheritance Ghost Elimination
		// Only instantiate the variable if the base actually has data for it!
		// ====================================================================
		var baseDefs = nonTrivialDefsFromBase(ctx, l.project, us)
		if len(baseDefs) == 0 {
			return // Skip ghost variable!
		}

		d := l.project.def(ctx, defVoid, us)
		op.apply(closure_with(ctx, l.project.scope), d, baseDefs...)
	})
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

    if s := f.fullname(); f.info == nil {
        debug(ctx, "%s: file not exists: %s", ident, s, trace{})
    } else if cc := pc(ctx, ident); f.info.IsDir() {
        l.directory(cc, dot_container, s, nil)
    } else {
        l.file(cc, filespec(l.workdir, s), s, nil)
    }

    if x, y := l.globe.loaded[f.fullname()]; y && x != nil {
        if name, _ := l.scope().Lookup(x.name).(*project) ; name == nil {
            debug(ctx, "%v: %v: `dock` is not a project", l.project.name, f, trace{})
        }

        var opts useopts
        // TODO: parse the useopts
        l.useProj(ctx, opts, x)
    }
    return
}

func is_configure_project(proj *project) bool {
    return proj == nil ||
        proj.name == dot_configure ||
        proj.name == "configure" ||
        proj.name == "configure.base"
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
        if d := l.project.resolveDef(ctx, ".autoload."+tag); d != nil && d.value != nil {
            for _, v := range merge(expand(_final(ctx),d.value)) {
                if isTrivial(v) {
                    continue
                } else if f, s, t := l.spec_file(ctx, v); f == nil || !f.exists() {
                    continue//debug(ctx, "no such source file: %v → %v", ts(d.value), ts(v), trace{})
                } else if s == "" || t == "" {
                    continue//debug(ctx, "empty string: %v → %v", ts(d.value), ts(v), trace{})
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

func (l ul) configuration(ctx Context, ident Value, _ string) {
	if false { defer un(l_tracef(l_traverse, "configuration(%v)", ident)) }
	if l.project.name == dot_configure { return }

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
				debug(ctx, "empty configure spec: %v", ts(v), trace{})
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
		cc.abs, cc.isDir = f.fullname(), f.info.IsDir()
	}

	if cc.abs == "" && l.project.opt.configure != nil {
		if !cc.local { cc.abs, cc.isDir = l.search(ctx, cc.configure) }
		if cc.abs == "" {
			debug(ctx, "%v: no such project: %s", l.project, cc.configure, trace{})
		}
	}

	if cc.abs == "" {
		if l.project.opt.configure != nil {
			debug(ctx, "%v: missing the default .configure", l.project, trace{})
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
		debug(ctx, "%s not loaded", cc.configure, trace{})
	}

	if x, y := l.project.Lookup(dot_configure).(*project); !y || x == nil {
		if _, alt := l.project.projectname(ctx, dot_configure, cc.declared); alt != nil {
			if p, y := alt.(*project); !y || p == nil {
				debug(ctx, "name `%s' already taken: %s", cc.declared.name, typeof(alt), trace{})
			}
		}
	}

	if c := l.project.configure; c != cc.declared {
		if c != nil && c != l.project {
			debug(ctx, "%s already specified", dot_configure, trace{})
		} else {
			l.project.configure = cc.declared
		}
	}

	// =========================================================
	// 2. Load Cache securely AFTER sandbox evaluation
	// Sets cc.configuration so `is_flat_mode` protects the parse.
	// =========================================================
	var c = l.project.configuration
	if c == nil { c = l.project.configuration_sm(ctx) }

	if c != nil && c.exists() && c.stat(ctx) != nil {
		if l.promptConfigurationLoads(ctx) {
			debug(pc(ctx, c), "cached configuration")
		}
		cc.configuration = c
		l.source(&cc, c.fullname(), nil) // Populates local definitions correctly
		l.project.configuration = c
	}

	for _, proj := range cc.declared.usees(true, false, false, false) {
		if e := l.useProj(ctx, useopts{}, proj); e != nil { 
			debug(ctx, "failed to use %v : %v", proj, e, trace{})
		}
	}
}

func (l ul) container(ctx Context, ident Value, identStr string) {
    if l.project.name != dot_container {
        if _, e := os.Stat(filepath.Join(l.project.absPath, ".dock")); e == nil {
            debug(ctx, "must rename .dock into .container !", trace{})
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
                debug(pc(ctx,filename), "invalid source : %v", ts(src), trace{})
            }

            if e != nil {
                debug(pc(ctx,filename), "copy bytes (%s) failed : %v", typeof(src), e, trace{})
            }
        }
        if 0 < n { return buf.Bytes() }
    }
    if t, e := ioutil.ReadFile(filename); e == nil {
        return t
    } else if _, y := e.(*fs.PathError); y {
        if (opts != nil && !opts.ifExists) {
            debug(pc(ctx,filename), "no such source file", trace{})
        }
        return
    } else {
        debug(pc(ctx,filename), "%v", e, trace{})
        return
    }
}

func (l ul) source(ctx Context, filename string, a_src any) (res Value) {
    if l.traceLaunch { defer un(l_trace(l_launch, "ul.source")) }

    defer func(p *parser) {
        if l.p == nil { debug(ctx, "nil parser", trace{}) }
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
        debug(ctx, "%v", err, trace{})
        return
    }

    defer fd.Close()

    var fs []os.FileInfo
    if fs, err = fd.Readdir(-1); err != nil || len(fs) == 0 { return }

    var ident = filepath.Base(pathname)
    if ident == "_" {
        debug(ctx, "invalid package name %s", ident, trace{})
    }

	var sof, _ = filepath.Rel(workBaseDir, pathname)
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
                debug(ctx, "parse config failed: %v", err, trace{})
            }
            if 0 < flush(ctx) { return } else { continue }
        }

        d := scope.def(ctx, defConfig, name)
        if d == nil {
            debug(ctx, "%v", name, trace{})
        }

        var v []byte
        if v, err = ioutil.ReadFile(fullname); err != nil {
            debug(ctx, "%v", err, trace{})
        }

        var s = string(v)
        if !utf8.ValidString(s) {
            debug(ctx, "%s: invalid UTF8 content", fullname)
        }

        d.set(ctx, _raw(l.p.pos, s))
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
        debug(ctx, "%v", err, trace{})
    }

    defer fd.Close()

    fis, err := fd.Readdir(-1)
    if err != nil {
        debug(ctx, "readdir: %v", err, trace{})
    }
    if len(fis) == 0 {
        debug(ctx, "no files underneath: %s", path, trace{})
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
        debug(pc(ctx,l.p), "%v: no such base: %v", l.project.name, spec, trace{})
    } else if !filepath.IsAbs(absPath) {
        debug(pc(ctx,l.p), "%v: not absolute path: %v", l.project.name, spec, trace{})
    }

    // Check loaded project.
    if p, y := l.globe.loaded[absPath]; y {
        if _, a := l.scope().projectname(ctx, p.name, p); a != nil {
            if x, y := a.(*project); !y || x == nil {
                debug(ctx, "name already taken: %v (%s).", p, typeof(a), trace{})
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
        debug(ctx, "%v: no such base: %v", l.project, spec, trace{})
    } else if !filepath.IsAbs(absDir) {
        debug(ctx, "%v: not absolute path: %v", l.project, spec, trace{})
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
                debug(c, "%v %v", d, loaded, trace{})
            }
        }
        if l.project != nil {
            if p, _ := l.project.Lookup(loaded.name).(*project); p == nil {
                if _, alt := l.project.projectname(ctx, loaded.name, loaded); alt != nil {
                    if x, y := alt.(*project); !y || x == nil {
                        debug(ctx, "`%s' already taken: %s", loaded.name, alt, trace{})
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

	var sof, _ = filepath.Rel(workBaseDir, absDir)
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
        debug(ctx, "%s not loaded (as %s)", spec, absDir, trace{})
    }

    if loaded, okay = l.globe.loaded[absDir]; okay && loaded != nil {
        return // Good!
    }

    if filepath.Base(spec) == "@" {
        return // Okay!
    }

    debug(ctx, "%s not loaded", spec, trace{})
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
