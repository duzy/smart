///
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const maxDigitAutoNum = 9

// A bailout panic is raised to indicate early termination.
type bailout struct{}

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
	// configure  bool // is parsing configure program?

	locals []map[string]*def

	dd bool // helps debug parsing via `eval -dd=true{}`
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

type is_braced struct{}
type codeblock      struct{ *automatic ; token }
type defval        struct{ original ; d *def}
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
type p_rule_ctx     struct{ Context }
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
func (p defval) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case is_auto: return t.s != "0" && IsDigits(t.s)
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

	res = &comment{p.Position(), p.lit}
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

func (p *parser) valbase(Context) valbase { return valbase{p.Position()} }
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

	pos, tok := lhs.Position(), l.p.tok // the arrow '->' or '=>'

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
	tok, lit, pos := p.tok, p.lit, p.Position()
	if tok != WORD && lit == "" { lit = tok.String() }
	p.step(ctx) // consumes the current token
	return _word(pos, lit)
}

func (l ul) braced(ctx Context) (x Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "braced")) }

	pos := l.p.Position()
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
					_raw(p, p.Filename), _punct(ctx, COLON),
					_decimal(p, int64(p.Line)), _punct(ctx, COLON),
					_decimal(p, int64(p.Column)), _punct(ctx, COLON),
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
	if truly(ctx, is_recipe{false}) && p.tok == RECIPE { // TODO: using p.recipe_start()
		return true
	}
	return false
}

func (p *parser) rule_params(ctx Context, args []Value) (err error) {
	var s = _scope(ctx)
	for _, arg := range args {
		switch arg.(type) {
		case *word, *compound:
			var a = s.auto(ctx, __string(ctx, arg))
			s.alias(ctx, a, strconv.Itoa(len(p.ruparas)+1))
			p.ruparas = append(p.ruparas, a)
		default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
			debug(ctx, "bad parameter form (%v)", tv(arg), trace{})
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

	pos := l.p.Position()
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

		if l.p.pos == p { debug(ctx, "syntax error", trace{}) }

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
	p, t := l.p.Position(), l.p.tok
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
			perc := makePercpat(l.p.Position(), nil, nil)
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
	return makePercpat(l.p.loc(pos), x, y)
}

type regex_subexp_auto struct{ *regexp.Regexp }

func (l ul) regex(ctx Context) (_ Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "regex")) }

	var rx string
	var pos = l.p.Position()

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
		return flag{&valbase{l.p.Position()}}
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
	tok, lit, pos := l.p.tok, l.p.lit, l.p.Position()

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

// Parses dot composing expressions (TODO: check against file extensions).
//   .foo
//   .'foo'
//   ."foo"
//   .(foo)
//   ..foo
//   ..'foo'
//   .foo.bar
func (l ul) dot(ctx Context, x Value) (_ Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "dot")) }

	if t := (&punct{l.p.valbase(ctx),DOT}); x == nil {
		x = t
	} else {
		x = prefix(ctx, x, t)
	}

	ctx = aware(ctx, DOT)
	l.p.step(ctx)

	for !l.p.is_dot_term(ctx) {
		p := l.p.pos
		x = prefix(ctx, x, l.unary(ctx))

		if l.p.pos == p { debug(ctx, "syntax error", trace{}) }

		switch l.p.tok {
		case DOT:
			x = prefix(ctx, x, &punct{l.p.valbase(ctx),DOT})
			l.p.step(ctx) // skips '.'
		}
	}
	return x
}

func (l ul) path(ctx Context, start Value) (res *path) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "path")) }
	if start == nil { debug(ctx, "nil path starter", trace{}) }

	ctx = p_path{ctx}

	switch t := start.(type) {
	case *path: res = t
	case *strlit:
		res = makePath(splitPathStr(pc(ctx,t.position), t.s)...)
	case *strcomp:
		res = makePath(splitPathStr(pc(ctx,t.Position()), __string(ctx, t))...) // FIXME: dont final here
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

	var h *compound
	var p *path

hostloop:
	for l.p.tok == DOT {
		if h == nil { h = &compound{} }
		h.elems = append(h.elems, x, &punct{l.p.valbase(ctx),DOT})

		l.p.step(ctx) // '.'

		switch x = l.unary(ctx); l.p.tok {
		case PCON, QUE, HASH:
			h.elems = append(h.elems, x)
			break hostloop
		case LINEND, EOF:
			h.elems = append(h.elems, x)
			u.Host = h
			return u
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

func (l ul) resolve(ctx Context, name Value, str string) (result Value) {
	var pos Position
	if name != nil { pos = name.Position() }
	if !pos.valid() { pos = l.p.Position() }
	if !pos.valid() { pos = _position(ctx) }
	if str == "" {
		debug(ctx, "resolve no-name : %v", ts(name), trace{})
	}

	if d := auto_find(ctx, str); d != nil {
		return d
	}

	var o object
	var s = _scope(ctx)

	defer func() {
		switch x := o.(type) {
		case *builtin:
			t := *x
			t.position = name.Position()
			result = &t
		}
	} ()

	if l.project == nil || s != l.project.scope {
		if _, o = s.find(str); o != nil { return o }
	}
	if l.project != nil {
		if o = l.project.resolve(ctx, str); o != nil { return o }
	}

	switch {
	case truly(ctx, is_auto{str}):
		if a := s.auto(pc(ctx,pos), str); a != nil { return a }
		debug(ctx, "bad auto: %v", ts(name), trace{})
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
	// case cond:
	// 	obj, str, opts = l.identity(optional_ident(ctx), tok, x.Value)
	// 	obj = cond{obj}
	// 	return
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

		errostack(pc(ctx,name), 32, "empty ident: %v (nil=%d) : %s", name, ic.nil, ts(name), trace{})
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

	errostack(pc(ctx,name), 32, "undefined %v → %v : %s", name, str, ts(name), trace{})
	return
}

func (l ul) calling(ctx Context) (result Value) {
	var tok token
	var str string
	var name, obj Value
	var args, opts []Value
	var pos = l.p.Position()
	var closure = l.p.tok.is_closure()

	l.p.step(ctx) // $ &

	switch l.p.tok {
	case LPAREN, LBRACE: // $(...), ${...}
		tok = l.p.tok // use LPAREN, LBRACE
		l.p.step(ctx) // skips LPAREN, LBRACE

		if l.p.tok == SPACE { debug(ctx, "unexpected spaces", trace{}) }

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

	case INTEGER:
		tok = l.p.tok
		name = l.literal(ctx) // $0, $1...
		str = name.String()
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
			debug(ctx, "unexpects %v", l.p.lit, trace{})
		}

	default: // case AT, BAR, DOT, SAST, QUE, MINUS, PLUS, PCON:
		tok, str = l.p.tok, l.p.tok.String()
		name = l.punct(ctx) // $@, $?, $*, $/...
		if obj = l.resolve(ctx, name, str); obj == nil {
			debug(ctx, "unexpects %v %v (dialect=%s)", tok, name, l.p.dialect, trace{})
		}
	}

	if obj == nil && str != "" {
		if l.project.ext.Plugin != nil {
			if t, e := l.project.ext.Lookup(str); e == nil && t != nil {
				debug(ctx, "TODO: convert ext symbol: %v : %v", name, ts(t), trace{})
			}
		}
	}

	if obj == nil {
		debug(ctx, "%v : nil symbol", tok, trace{})
	}

	if closure {
		return makeClosure(pos, tok, obj, opts, args...)
	}

	if x, y := obj.(*def); y && x.o == defStatic {
		if !truly(ctx, is_auto_preserved{x.name}) {
			return _loc(x.value, name.Position())
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
			var x = &valbase{l.p.Position()}
			if l.p.step(ctx); l.p.is_list_term(ctx) {
				return &pair{x, &valbase{l.p.Position()}}
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
		tok, pos := l.p.tok, l.p.Position()
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
		ctx = pc(ctx, l.p.loc(pos))
		debug(ctx, "unexpected %v '%s'", l.p.tok, l.p.lit)
		note(ctx, "x: %v %v", x, ts(x))
		note(ctx, "scanstate: %v", l.p.scanner.scanstate, trace{})
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
				return &pair{x, &valbase{l.p.Position()}}
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
	pos := l.p.Position()
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
	case RBRACE: return _null(l.p.Position())
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

	var pos = l.p.Position()
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

	var pos = l.p.Position()
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
func (dc *defcaps) ts(ctx Context, t string) (s string) {
	s = "{="+t+" "+ts(dc.Value,ctx)
	for _, cap := range dc.caps {
		s += " {"+cap.name+":"+ts(cap.value,ctx)+"}"
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
			var pos = pat.Position()
			var name = _raw(pos, _k)
			var neg bool
			if x, y := pat.(negative); y { pat, neg = x.Value, y }

			a, _, c := match(pc(ctx, pat), pat, name)
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
							val = _raw(pos, "")
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
		debug(pc(ctx,elem), "not a file: %v", tv(elem), trace{})
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
		var v = expand(_final(ctx),elem)
		v = as{v}.fullname(ctx, l.project)
		elems = append(elems, v)
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
	var pos = l.p.Position()
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

	var pos = l.p.Position()

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
	var pos = l.p.Position()
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

type clauseopts struct{
	general_opts

    keyword token // e.g. use, files, eval, etc.

    skip bool // e.g. -cond({=false}), -if({=no})

	conds []Value `if,cond,where`

    values, remainder, spec []Value // all values (unparsed) and remainder
}

type parseSpecFunc func(Context, *commentgroup, *clauseopts, int)

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

func (l ul) use(ctx Context, doc *commentgroup, g *clauseopts, _ int) {
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
	var args = parse_opts(ctx, &opts, append(g.remainder, g.spec[1:]...)...)
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

func (l ul) files(ctx Context, doc *commentgroup, g *clauseopts, _ int) {
	if len(g.spec) != 1 { debug(ctx, "too many properties: %v", g.spec, trace{}) }

	var p Value
	var patts, paths []Value

	if l.p.tok == SELECT_PROG1 {
		l.p.next(ctx, true) // step forward with spaces skipped
		if l.p.tok == LINEND || l.p.lineComment != nil {
			debug(ctx, "expecting files path", trace{})
		}
		p = l.expr(ctx)
	}

	l.p.spaces(ctx)

	if g.skip { return }

	if t := parse_opts(ctx, &g.general_opts, g.remainder...); t != nil {
		debug(ctx, "unsupported opts: %v", t, trace{})
	}

	if t := expand(original{ctx,defExpand1}, g.spec[0]); t == nil {
		debug(ctx, "nil: %v", g.spec[0], trace{})
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
		switch x := expand(original{ctx,defExpand1},p).(type) {
		case *group: paths = x.elems
		default: paths = []Value{ x }
		}
	}

	map_files(ctx, l.project, patts, paths)
}

func (p *parser) assert(ctx Context, doc *commentgroup, g *clauseopts, _ int) {
	if !g.skip { call(ctx, "assert", g.remainder, g.spec...) }
}

func (p *parser) append(ctx Context, doc *commentgroup, g *clauseopts, _ int) {
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

func (l ul) local(ctx Context, _ *commentgroup, g *clauseopts, _ int) {
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
				debug(ctx, "unsupported flag: %v", tv(a), trace{})
			}
			continue
		}

		var s = __string(ctx, a)
		if s == "" {
			debug(ctx, "empty local: %v", tv(a), trace{})
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

func (l ul) eval(ctx Context, doc *commentgroup, g *clauseopts, _ int) {
	if g.skip { return }
	if g.spec == nil {
		var opts struct{
			optimize Value `opt,optimize`
		}
		for _, op := range parse_opts(_final(ctx), &opts, g.values...) {
			var val Value
			if v, y := op.(*pair); y { op, val = v.key, v.val }
			if v, y := op.(flag); y {
				switch t := val != nil && __true(ctx, val); __string(ctx, v.Value) {
				case "dd": l.p.dd = t
				case "ddd":
					if val == nil {
						l.ddd = "yes"
					} else if t, y := boolVal(val); y {
						if t { l.ddd = "yes" } else { l.ddd = "" }
					} else {
						l.ddd = __string(ctx, val)
					}
				}
			} else {
				debug(ctx, "unsupport flag: %v (%v)", ts(v), val, trace{})
			}
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
	case "":
		debug(pc(ctx,l.p), "empty eval command", trace{})
	case "-configuration", "configuration":
		debug(pc(ctx,l.p), "configuration is done at parse time", trace{})
	}

	resolved := l.resolve(ctx, prop0, name)
	switch x := resolved.(type) {
	case invoker:
		x.invoke(ctx, opts, expands(_final(ctx), g.spec[1:]...))
	default:
		debug(pc(ctx,l.p), "resolved is %s: %v → %s", typeof(resolved), prop0, name, trace{})
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

	var opts = clauseopts{ keyword: keyword }
	for l.p.spaces(ctx); l.p.tok == MINUS; l.p.spaces(ctx) {
		opts.values = append(opts.values, l.expr(ctx))
	}

	opts.remainder = parse_opts(ctx, &opts, opts.values...)

	for _, cond := range opts.conds {
		if t := __true(ctx, cond); !t {
			opts.skip = true
			break
		}
	}

	l.p.spaces(ctx)

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
		forids(ctx, expand(_final(ctx),v), func(v Value, _ []Value) { ids = append(ids, v) })
	}

	tok := l.p.tok
	ctx = pc(ctx, l.p.Position())
	l.p.next(ctx, true) // the assign token

	_pos, _tok, _lit, _ss := l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate
	for _, id := range ids {
		var alt object
		var d *def

		switch t := id.(type) {
		case *argumented:
			debug(ctx, "multiple defs: %v, args=%v", t.Value, t.args, trace{})

		case *group:
			debug(ctx, "multiple defs: %v", t.elems, trace{})

		case *arrow:
			if v := expand(_final(ctx),t); v == nil {
				debug(ctx, "%v is nil", ts(t), trace{})
			} else if x, y := v.(*def); !y {
				debug(ctx, "%v is not a def: %v", ts(t), ts(v), trace{})
			} else {
				d = x
			}

		default: // *word, *compound, *qualword, *path, flag:
			name := ident(def_name{ctx}, expand(ctx, t))
			if _, y := builtins[name]; y {
				debug(pc(ctx,t), "`%v` is a builtin name (%v)", ident, name, trace{})
			}

			prev := l.project.resolve(ctx, name)
			d = l.project.def(ctx, defInvalid, name)

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

		l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate = _pos, _tok, _lit, _ss

		_ctx := defval{original{ctx,0},d}

		if !d.position.valid() { d.position = l.p.Position() }

		switch tok {
		case ASSIGN_EXC: // !=
			_ctx.o = defExecute
			d.origin(ctx, _ctx.o)
			d.val(_ctx, l.values(_ctx))
		case ASSIGN: // =
			_ctx.o = defExpand0
			d.origin(ctx, _ctx.o)
			d.val(_ctx, l.values(_ctx))
		case ASSIGN_CO1: // :=
			_ctx.o = defExpand1
			d.origin(ctx, _ctx.o)
			d.val(_ctx, expands(_ctx, l.values(_ctx)...))
		case ASSIGN_CO2: // ::=
			_ctx.o = defExpand2
			d.origin(ctx, _ctx.o)
			d.val(_ctx, expands(_ctx, l.values(_ctx)...))
		case ASSIGN_CO3: // ;:=
			_ctx.o = defExpand3
			d.origin(ctx, _ctx.o)
			d.val(_ctx, expands(_ctx, l.values(_ctx)...))
		case ASSIGN_QUE: // ?=
			if d.o == defInvalid {
				_ctx.o = defAssign0
				d.origin(ctx, defExpand0)
				d.val(_ctx, l.values(_ctx))
			}
		case ASSIGN_ADD: // +=
			if d.o == defInvalid { d.o = defExpand0 }
			switch _ctx.o = d.o|defAssign1; {
			case d.o&defExpand0 != 0:
				d.set(_ctx, nil, l.values(_ctx)...)
			case d.o&(defExpand1|defExpand2|defExpand3) != 0:
				d.set(_ctx, nil, expands(_ctx, l.values(_ctx)...)...)
			default:
				debug(ctx, "unknown: %v %v", _ctx.o, d.name, trace{})
			}
		case ASSIGN_SHI: // =+
			if d.o == defInvalid { d.o = defExpand0 }
			switch _ctx.o = d.o|defAssign2; {
			case d.o&defExpand0 != 0:
				d.val(_ctx, append(l.values(_ctx), d.value))
			case d.o&(defExpand1|defExpand2|defExpand3) != 0:
				d.val(_ctx, append(expands(_ctx, l.values(_ctx)...), d.value))
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
						vals = l.values(_ctx)
					case d.o&(defExpand1|defExpand2|defExpand3) != 0:
						vals = expands(_ctx, l.values(_ctx)...)
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
				vals = l.values(_ctx)
			case d.o&(defExpand1|defExpand2|defExpand3) != 0:
				vals = expands(_ctx, l.values(_ctx)...)
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
	var p Position
	var elems []Value
	var isList, isPlainline bool

	switch l.p.dialect {
	case "value", "":
		l.p.scanner.pop(isStrcompLine)
		l.p.next(ctx, true) // skip RECIPE or SEMICOLON and parse in list mode
		p = l.p.Position()
		if isList = true; !l.p.is_end_of_line() {
			var c = p_recipe{ctx, true, nil, nil} // builtin value recipe
			for l.p.tok != EOF && l.p.tok != SEMICOLON && l.p.tok != LINEND && l.p.lineComment == nil {
				elems = append(elems, l.expr(&c))
				if l.p.spaces(ctx); l.p.lineComment != nil { break }
			}
		}

	case "eval":
		l.p.scanner.pop(isStrcompLine)
		l.p.next(ctx, true) // skip RECIPE or SEMICOLON and parse in list mode
		p = l.p.Position()
		if isList = true; !l.p.is_end_of_line() {
			var x = l.expr(ctx) // parse first expr of recipe

			var a *argumented
			if a, _ = x.(*argumented); a != nil { x = a.Value }
			if x == nil {
				errostack(pc(ctx,p), 16, "parsed nil value, dialect=%s", l.p.dialect, trace{})
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
		p = l.p.Position()

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

	pos := l.p.Position()
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
	res.position = pos
	res.elems = append([]Value{val}, elems...)
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
		errostack(ctx, 5, "empty modifier group", trace{})
	}
	if l.p.tok == COLON {
		errostack(ctx, 5, "unexpected colon after modifer", trace{})
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
	"@" :struct{}{}, "%" :struct{}{}, "<" :struct{}{}, ">" :struct{}{},
	"@D":struct{}{}, "%D":struct{}{}, "<D":struct{}{}, ">D":struct{}{},
	"@F":struct{}{}, "%F":struct{}{}, "<F":struct{}{}, ">F":struct{}{},
	"@'":struct{}{}, "%'":struct{}{}, "<'":struct{}{}, ">'":struct{}{},
	"^" :struct{}{}, "+" :struct{}{}, "|" :struct{}{}, "*" :struct{}{},
	"^D":struct{}{}, "+D":struct{}{}, "|D":struct{}{}, "*D":struct{}{},
	"^F":struct{}{}, "+F":struct{}{}, "|F":struct{}{}, "*F":struct{}{},
	"^'":struct{}{}, "+'":struct{}{}, "|'":struct{}{}, "*'":struct{}{},
	"?" :struct{}{},
	"?D":struct{}{},
	"?F":struct{}{},
	"?'":struct{}{},
	"-" :struct{}{},
	"~" :struct{}{},
	//"<-":struct{}{}, "->":struct{}{},
}

func (l ul) rule(ctx Context, targets []Value) (result Value) {
	if l_traverse.enabled || debugSyntax(ctx, "rule") { defer un(l_trace(l_traverse, "rule")) }

	ctx = p_rule_ctx{ctx}

    if l.project != _scope(ctx).project {
		debug(ctx, "mismatched project/scope : %v", targets, trace{})
	}

	// TODO: doc = p.leadComment
	var depends, ordered, recipes []Value
	defer l.closescope(l.openscope(tv(targets)))
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
		language:  l.p.dialect,
		params:    l.p.ruparas,
		project:   l.project,
		position:  targets[0].Position(),
		depends:   depends,
		ordered:   ordered,
		recipes:   recipes,
	}

	if res := l.entries(ctx, &prog, targets); 1 == len(res) {
		return res[0]
	} else if 1 < len(res) {
		return list_t[entry](res...)
	} else {
		return _null(prog.position)
	}
}

func (l ul) entries(ctx Context, prog *program, targets []Value) (res []entry) {
	for _, target := range targets {
        if isTrivial(target) { continue }

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

var pprofCounter int

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
		case FOREACH:
			l.p.next(ctx, true) // foreach
			nested += 1

		case DONE:
			if nested > 0 { nested -= 1 ; continue }

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
		if vals := parse_opts(ctx, &opts, l.values(ctx)...); vals != nil {
			debug(ctx, "unexpected opts: %v", vals, trace{})
		}
		l.p.expect(ctx, RPAREN)
	}

	l.p.spaces(ctx)

	type  param struct{ name string ; elems []Value }
	type nparam struct{ p Position ; a []*param ; n int }

	var params []*nparam
	var ac = automatic{Context:ctx, defs:make(def_map)}
	for l.p.spaces(ctx); l.p.tok != EOF && !l.p.is_end_of_line(); l.p.spaces(ctx) {
		if l.p.tok == AND && params == nil {
			debug(pc(ctx,l.p), "unexpected 'and'", trace{})
		} else if l.p.tok == AND || params == nil {
			params = append(params, &nparam{p:l.p.Position()})
			if l.p.tok == AND { l.p.next(ctx, true); continue }
		}

		var pars = make(map[string]*param)
		var p = params[len(params)-1]
		for i, a := range merge(expand(&ac, l.expr(&ac))) {
			switch x := unbox(a).(type) {
			case *null: continue
			case *pair:
				var par = new(param)
				if par.name = __string(ctx, x.key); par.name == "" {
					debug(pc(ctx,a), "empty key %v", ts(x.key), trace{})
				}
				if g, y := x.val.(*group); y {
					par.elems = merge(g.elems...)
				} else {
					par.elems = merge(x.val)
				}
				if n := len(par.elems); n > p.n { p.n = n }
				if _, y := ac.defs[par.name]; !y { ac.set(&ac, defStatic, par.name, nil) }

				p.a = append(p.a, par)

			case *defcaps:
				for _, cap := range x.caps {
					if t, y := pars[cap.name]; y {
						t.elems = append(t.elems, cap.value)
					} else {
						t = &param{cap.name, []Value{cap.value}}
						p.a = append(p.a, t)
						pars[cap.name] = t
					}
				}

			default:
				errostack(pc(ctx,a), 6, "unexpected %v ; %d. %v", ts(a), i, ac.defs, trace{})
			}
		}
	}

	var t = &template{pos:l.p.pos, tok:l.p.tok, lit:l.p.lit, state:l.p.scanner.scanstate}

	l.p.spaces(ctx)
	l.p.linend(ctx)

	var nested = 0
	for l.p.tok != EOF {
		switch pos := l.p.pos; l.p.tok {
		case FOR:
			l.p.next(ctx, true) // for
			nested += 1

		case DONE:
			if nested > 0 { nested -= 1 ; continue }

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

			var e int = len(params)-1
		outer:
			for n := 0; n < num; n += 1 {
				for _i, _p := range params {
					// i[0]    = (n % 1) / b    (a = n * n * ..., k-1)
					// i[1..l] = (n % a) / b    (b = n * ..., k-2)
					// i[l+1]  = (n % a) / 1
					var i = n

					// Two implements: 1. compact, 2. TODO: expand (loose)
					//    1. compact: use the minimum nparam, skip elements after it (DONE)
					//    2. expand: use the maximum nparam, treat every part the same (TODO)

					// 1. compact mode
					for k, t := range params {
						if t.n == 0 {
							if true { continue outer }
						} else if k <= _i {
							if 0 < _i { i %= t.n }
						} else {
							if _i < e { i /= t.n }
						}
					}

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

	for l.p.tok != EOF && l.p.pos < l.p.stop {
		if l.p.tok == SPACE || l.p.tok == LINEND || (l.p.tok == COMMENT && l.p.lineComment != nil) {
			l.p.next(ctx, true)
		} else {
			l.clause(&c)
		}
	}
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
			debug(ctx, "%v", tv(e), trace{})
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
				v = _null(v.Position())
			}
			ac.set(&ac, defStatic, s, v)
		} else {
			debug(ctx, "empty template param name: %v", tv(v), trace{})
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

func (l ul) configure_save(ctx Context) {
	if l.project == nil {
		debug(ctx, "nil project", trace{})
	}

	var configs = l.project.configs
	if configs == nil { return }

	if l.promptEnteringDirectory {
		l.promptEnteringDirectory = false
		promptLeavingDirectory(ctx, l.project.absPath)
		flush(ctx)
	}

	var f = l.project.configuration_sm(ctx)
	var fn = f.fullname()

	if checkpoints {
		if c := l.project.configuration; c != nil && c.fullname() != fn {
			errostack(pc(pc(ctx,fn),c.fullname()), 3, "%s: configuration already loaded", l.project.name, trace{})
		}
	}

	if e := os.MkdirAll(filepath.Dir(fn), os.FileMode(0755)); e != nil {
		errostack(pc(ctx,fn), 3, "make path %s failed: %v", filepath.Dir(fn), e, trace{})
	}

	var fm = os.O_RDWR | os.O_CREATE | os.O_TRUNC
	var o, e = os.OpenFile(fn, fm, os.FileMode(0600))
	if e != nil {
		errostack(pc(ctx,fn), 3, "%s: %v", l.project.name, e, trace{})
	}
	defer func() {
		if o.Close(); 0 < diagCount(ctx, diagError) { os.Remove(fn) }
	} ()

	fmt.Fprintf(o, "# %s (%s)\n", l.project.name, l.project.spec)

	for _, c := range configs {
		fmt.Fprintf(o, "configure %s =", c.name)
		if c.value != nil { fmt.Fprintf(o, " %v", c.value) }
		fmt.Fprintf(o, "\n")
	}

	fmt.Fprintf(o, "\n# %d configs, %s\n", len(configs), l.project)

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
			errostack(pc(ctx,x.Value), 8, "wrong configure word: %v", tv(x.Value), trace{})
		}
		args = xmerge(_final(ctx), x.args...)
	}

	for _, arg := range args {
		switch t := arg.(type) {
		case *pair:
			par[__string(ctx, t.key)] = t
		case *raw, *strlit, *strval, *strcomp:
			par["INFO"] = &pair{_word(t.Position(),"INFO"), t}
		default:
			if !isTrivial(arg) {
				errostack(pc(ctx,arg), 8, "wrong arg: %s", ts(arg), trace{})
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
		errostack(pc(ctx,_op), 8, "wrong configure word: %v %v %v", tv(_op), tv(op), tv(val), trace{})
	}
	switch x.s {
	case "answer":
		if val == nil { return _answer(op.Position(), false) }
		return _answer(val.Position(), __true(ctx, val))
	case "bool", "boolean":
		if val == nil { return _boolean(op.Position(), false) }
		return _boolean(val.Position(), __true(ctx, val))
	case "value":
		if val == nil { return _null(op.Position()) }
		return expand(_final(ctx),val)
	}

	if l.project.configure == nil {
		errostack(pc(ctx,op), 8, "wrong configure: %v %v", tv(op), tv(val), trace{})
	}

	var ops = l.project.configure._entries(ctx, _op, false)
	if ops == nil {
		errostack(pc(ctx,_op), 8, "no configure ops: %v", _op, trace{})
	}

	var vals []Value
	for _, ent := range ops {
		var params []Value
		for _, prog := range ent.programs() {
			for _, p := range prog.params {
				w := _word(p.Position(), ident(ctx, p))
				if x, y := par[w.s]; y {
					params = append(params, x)
				} else {
					switch w.s {
					case "TARGET": params = append(params, &pair{w, auto_get(ctx, "@")})
					case "VALUE":  params = append(params, &pair{w, val})
					case "LANG", "LANGUAGE":
						if ctx.language != "" {
							lang := _word(w.position, ctx.language)
							params = append(params, &pair{w, lang})
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
		l.p.linend(ctx)
		l.p.spaces(ctx)
		for l.p.tok != EOF {
			l.configure1(ctx)
			l.p.spaces(ctx)
			if l.p.tok == RPAREN {
				l.p.next(ctx, true)
				break
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
				errostack(pc(ctx,t), 3, "needs cond value", trace{})
			}
		}

		if _op != nil {
			errostack(pc(ctx,t), 3, "configure op already defined: %v", _op, trace{})
		}

		_op = t
		break
	}

	op, par := l.configure_par(ctx, _op)
	exe := execution{
		automatic:automatic{Context:pc(ctx,op), defs:make(def_map)},
		start:time.Now(), proj:l.project,
	}

	if l.p.tok == ASSIGN {
		l.p.next(ctx, true) // skips the '=' token

		pos, tok, lit, sst := l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate
		for _, id := range ids {
			d, _ := exe.set(&exe, defVoid, "@", id)
			d.position = id.Position()

			d, _ = l.configure_set(ctx, __string(ctx, id)) // aka. l.project.set
			d.position = id.Position()

			cc := defval{original{&exe,defExpand1},d}
			l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate = pos, tok, lit, sst
			l.p.dialect = ""

			d.set(ctx, ease(ctx, expands(cc, l.values(cc)...)))
			l.p.lineComment = nil
		}
		return
	} else if l.p.tok == COLON {
		l.p.next(ctx, true) // skips the ':' token

		pos, tok, lit, sst := l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate
		for _, id := range ids {
			d, _ := exe.set(&exe, defVoid, "@", id)
			d.position = id.Position()

			d, _ = l.configure_set(ctx, __string(ctx, id)) // aka. l.project.set
			d.position = id.Position()

			cc := defval{original{&exe,defConfig|defExpand1},d}
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

				// _info = true

				a := prompt(pc(ctx,op), "%s …", s)
				defer func(i int) {
					if diagCount(ctx, diagInfo, diagWarn, diagError) <= i {
						s = __string(ctx, ease(ctx, vals))
						s = strings.Replace(s, "\n", "\\n", -1)

						b := prompt(ctx, "… %s\n", s)
						flush(ctx)

						if checkpoints {
							l.configure_val_check(&exe, d.name, op, vals, a, b)
						}
					}
				} (diagCount(ctx, diagInfo, diagWarn, diagError))
			}

			if _no_cond {
				d.set(cc, _null(id.Position()))
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
					errostack(pc(ctx,a), 8, "defer: not a modifier: %s", ts(a), trace{})
				}
			}

			d.set(cc, ease(ctx, vals))
		}
		return
	} else if l.p.tok.is_assign() {
		errostack(pc(ctx,l.p), 8, "%v: only '=' can set a configure", op, trace{})
	} else {
		errostack(pc(ctx,l.p), 8, "%v: wrong configure", op, trace{})
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
		errostack(ctx, 6, "unexpected %v", t, trace{})
	}

	var vals []Value

	for l.p.tok != LINEND && l.p.tok != EOF {
		var x = l.expr(left_side{ctx})

		l.p.spaces(ctx)

		vals = append(vals, x)

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

// project returns a new project for the given project path and name;
// the name must not be the blank identifier.
// The project is not complete and contains no explicit imports.
func (l ul) new_declare(ctx Context, pos Position, name, filename string, opts *project_opts) (d *declare) {
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
    if x, y := l.globe.loaded[absPath]; y {
        prompt(ctx, "%s: %v : already declared : %s\n", absPath, x, filename)
        errostack(ctx, 5, "%s %s %s : %v", name, relPath, spec, l.project, trace{})
    }

    if l.declares == nil { l.declares = make(map[string]*declare) }

	d = &declare{
		project: &project{
			position: /* l.p.Position() */pos,
			absPath: absPath,
			tmpPath: tmpPath,
			rel: relPath,
			spec: spec,
			name: name,
			opt: *opts,
			use: new(uselist), // TODO: use scopename instead?
		},
	}

    l.declares[name]  = d
    l.globe.loaded[d.absPath] = d.project

	do(ctx, declared_project{d.project})

    d.p = l.p
    d.s = l.loader.scope
    d.scope = newscope(d.position, sco, d.project, name)
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

type project_opts struct{
	configure Value `conf,config,configure` // detects dot_configure if empty
	traveUseLoop bool `break,loop` // don't recursively use this project
	multiUseAllowed bool `multi`  // this project is used multiple times
}

func (l ul) declare(ctx Context, ident Value, name, filename string, declOpts *project_opts) (_ bool) {
	if name == "@" {
		debug(ctx, "deprecated project name: @", trace{})
	}

    if _, o := l.find(name); o != nil {
        if x, y := o.(*builtin); y {
            debug(ctx, "%v is a builtin name", x, trace{})
        }
    }

	var prev = l.loader // nil if newly declared
	var dec = l.new_declare(ctx, ident.Position(), name, filename, declOpts)
	if prev == nil || dec.project != prev.project {
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
    return true
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

func (l ul) proj(ctx Context, filename string, isMainFile bool) (_ Value, _ string, _ bool) {
	var implicitBase string // aka. foo.bar.Baz implicitly load base 'foo/bar'

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

	var ident Value
	var opts project_opts
	if a := parse_opts(ctx, &opts, vals...); len(a) > 0 {
		errostack(pc(ctx,filename), 3, "unknown project option %v", ts(a), trace{})
	}

	if l.p.tok == LPAREN || l.p.tok == EOF || l.p.tok == LINEND || l.p.lineComment != nil {
		var dir = filepath.Dir(filename)
		if l.project != nil && l.project.absPath == dir {
			ident = _word(l.p.Position(), l.project.name)
		} else if s := filepath.Base(filename); s == dot_base || s == dot_configure {
			// NOTE: loading the .base or .configure file
			ident = _word(l.p.Position(), s)
		} else if s := filepath.Base(dir); s != "" {
			// TODO: validate basename as a valid identifier
			ident = _word(l.p.Position(), s)
		} else {
			debug(ctx, "invalid file: %v", filename, trace{})
		}
	} else if l.p.tok == TILDE {
		if ext := filepath.Ext(filename); ext != ".smart" {
			debug(ctx, "`%v` not a smart file", filepath.Base(filename), trace{})
		} else if s := strings.TrimSuffix(filepath.Base(filename), ext); s == "" {
			debug(ctx, "`%v` not tilde name", filepath.Base(filename), trace{})
		} else {
			ident = _word(l.p.Position(), s)
		}
		l.p.next(ctx, true) // skip tilde
	} else {
		base, comp := makePath(), _compound()

		for l.p.tok != EOF && l.p.tok != SPACE {
			var w = l.p.bare(ctx)
			if  comp = prefix(ctx, comp, w).(*compound) ; l.p.tok == DOT {
				comp = prefix(ctx, comp, l.punct(ctx)).(*compound)
				base.elems = append(base.elems, w)
			} else {
				break
			}
		}

		l.p.spaces(ctx)

		switch comp.len() {
		case 0:
			debug(pc(ctx,comp), "package name is empty (tok=%v)", l.p.tok, trace{})
		case 1:
			ident = comp.elems[0]
		default:
			ident = comp
		}

		if 0 < base.len() {
			implicitBase = __string(ctx, base)
		}
	}

	var name = __string(ctx, ident)

	if p := l.project; p != nil && p.name != name {
		errostack(ctx, 5, "%v: multiple projects in the directory : %v", p, ident, trace{})
	}

	if name == "-" || name == "_" {
		debug(ctx, "package name '%s' is preserved", name, trace{})
	}

	var _, prevDeclared = l.declares[name]

	if l.declare(ctx, ident, name, filename, &opts) {
		if l.project == nil {
			debug(ctx, "undeclared project: %v", ident, trace{})
		}
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

	var keyword  = l.p.tok
	var flatmode = truly(ctx, is_flat_mode{})

	var abs string
	var isMainFile bool // aka do.smart, build.smart

	if flatmode {
		abs = l.project.absPath
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

	var rel,_ = filepath.Rel(l.workdir, abs)
	var tmp   = joinTmpPath(ctx, l.workdir, rel)

	if s := l.scope(); /* p == nil || */ s == nil {
		errostack(ctx, 3, "%v: nil scope: %v", l.project, s, trace{})
	}

	defer l.closescope(l.openscope(bases(filename, 2, true)))

	if checkpoints {
		if s := l.p.scanner.file.Name(); filename != s {
			errostack(ctx, 3, "%v: %s != %s", l.project, filename, s, trace{})
		}
	}

	if !flatmode {
		// CWD: Current Work Directory,     TODO: use $:cwd:
		// CTD: Current Temp Directory,     TODO: use $:ctd:
		// CRD: Current Relative Directory, TODO: use $:crd:
		var s = l.scope()
		if d := s.def(ctx, defVoid, "/", _pathStr(ctx, abs)); d != nil { s.alias(ctx, d, "CWD") }
		if d := s.def(ctx, defVoid, ".", _pathStr(ctx, rel)); d != nil { s.alias(ctx, d, "CRD") }
		if d := s.def(ctx, defVoid, ",", _pathStr(ctx, tmp)); d != nil { s.alias(ctx, d, "CTD") }
	}

	switch keyword {
	case PROJECT:
		if flatmode {
			debug(pc(ctx,l.p), "project is forbidden in flat file", trace{})
		}

		var name string
		var prev = l.project

		_, name, isMainFile = l.proj(ctx, filename, isMainFile)
		if prev != l.project { defer l.close_project(ctx, name) }

	case EOF:
		return

	default:
		if !flatmode {
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
