///
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
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
)

const maxDigitAutoNum = 9

// A bailout panic is raised to indicate early termination.
type bailout struct{}

type use_spec struct {
	props []Value
}

type template struct {
	state scanstate
	end  *scanstate
	pos, endPos Pos // token position
	tok  token  // one token look-ahead
	lit  string // token literal
	verb string
	name Value // if only 'def', TODO: considering []Value for nested template defs?
	params []Value
}

type parser struct {
	scanner

	// Next token
	pos, stop Pos // parsing and stop position
	tok token  // one token look-ahead
	lit string // token literal

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
	p_is_params     struct{}
	p_is_undef      struct{}
	p_is_glob       struct{}
	p_no_argumented struct{}
	p_no_path       struct{}
)

type   aware_token struct { token }
type p_aware_comma struct { Context }
type p_aware_dot   struct { Context }
func (p p_aware_comma) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case aware_token: return t.token == COMMA
	}
	return p.Context.do(ctx, op)
}
func (p p_aware_dot) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case aware_token: return t.token == DOT
	}
	return p.Context.do(ctx, op)
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

type codeblock      struct { automatic ; token }
type p_defvalue_ctx struct { Context }
type p_braced_ctx   struct { Context }
type p_bare_ctx     struct { Context }
type p_auto_ctx     struct { Context }
type p_foreach_txt  struct { Context ; a *auto }
type p_group_ctx    struct { Context }
type p_left_ctx     struct { Context }
type p_modifier     struct { Context }
type p_params       struct { Context }
type p_path         struct { Context }
type p_perc         struct { Context }
type p_glob         struct { Context }
type p_regex        struct { Context }
type p_strcomp      struct { Context }
type p_undef_ctx    struct { Context }
type p_rule_ctx     struct { Context }
type recipe_ctx struct {
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

func (p p_undef_ctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_undef_ctx) inner() Context { return p.Context }
func (p p_undef_ctx) ts(t string) (_ string) {
	return "{="+t+" "+ts(p.Context)+"}"
}
func (p p_undef_ctx) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case p_is_undef: return true
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


type p_is_strcomp struct {}
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

type is_auto_preserved struct{ string }
type is_auto           struct{ string }
type is_defvalue       struct{}

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

func (p p_defvalue_ctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_defvalue_ctx) inner() Context { return p.Context }
func (p p_defvalue_ctx) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case is_auto: return IsDigits(t.string)
	case is_defvalue: return true
	}
	return p.Context.do(ctx, op)
}

func (p p_foreach_txt) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_foreach_txt) inner() Context { return p.Context }
func (p p_foreach_txt) ts(t string) (_ string) { return "{="+t+" "+ts(p.Context)+"}" }
func (p p_foreach_txt) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
    case find_auto: if t.string == "_" { return p.a }
	case is_auto: if t.string == "_" { return true }
	case is_auto_preserved: if t.string == "_" { return true }
	}
	return p.Context.do(ctx, op)
}

func (p p_rule_ctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_rule_ctx) inner() Context { return p.Context }
func (p p_rule_ctx) ts(t string) string { return "{="+t+" "+ts(p.Context)+"}" }
func (p p_rule_ctx) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case is_auto:
		if IsDigits(t.string) { return true }
		if _, y := rule_autos[t.string]; y { return true }
	}
	return p.Context.do(ctx, op)
}

type add_recipe_line struct{ a []Value }
type is_recipe_start struct{}
type is_recipe       struct{ bool } // builtin or text

func (p *recipe_ctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p *recipe_ctx) inner() Context { return p.Context }
func (p *recipe_ctx) ts(t string) (_ string) {
	return "{="+t+" "+ts(p.Context)+"}"
}
func (p *recipe_ctx) do(ctx Context, op any) (_ any) {
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

func (p p_left_ctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p p_left_ctx) inner() Context { return p.Context }
func (p p_left_ctx) ts(t string) string {
	return fmt.Sprintf("{="+t+" %s}", ts(p.Context))
}
func (p p_left_ctx) do(ctx Context, op any) (_ any) {
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

// Advance to the next token.
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

	p.pos, p.tok, p.lit = p.scanner.scan(ctx)
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

func (p *parser) Position() Position { return p.loc(p.pos) }
func (p *parser) loc(a Pos) Position { return Position(p.scanner.file.Position(a)) }
func (p *parser) line() int { return p.scanner.file.Line(p.pos) }
func (p *parser) valbase() valbase { return valbase{p.loc(p.pos)} }
func (p *parser) is_file(s string) bool {
	return strings.HasSuffix(p.scanner.file.Name(), s)
}

func (p *parser) expected(ctx Context, pos Pos, msg string, a... any) {
	if len(a) > 0 { msg = fmt.Sprintf(msg, a...) }
	if msg = "expected " + msg; pos == p.pos {
		// the error happened at the current position;
		// make the error message more specific
		if p.tok == SEMICOLON && p.lit == "\n" {
			msg += ", found newline"
		} else {
			msg += ", found " + p.tok.String()
			if p.tok.is_literal() {
				msg += " '" + p.lit + "'"
			}
		}
	}

	erro(pc(ctx,p), msg).trace()
}

func (p *parser) expect(ctx Context, tok token) Pos {
	var pos = p.pos
	if p.tok == tok {
		p.step(ctx) // move forward
	} else {
		p.expected(ctx, pos, "'"+tok.String()+"'")
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
		p.expected(ctx, p.pos, "'\\n'")
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

	tok := l.p.tok // the arrow '->' or '=>'
	pos := l.p.Position()

	ctx = pc(ctx, l.p)

	l.p.step(ctx) // skip '->' or '=>'

	switch t := lhs.(type) {
	case *arrow:
		if v := t.expand(ctx); isNull(v) {
			erro(ctx, "nil arrow: %v", lhs).trace()
		} else {
			lhs = v
		}
	case *word:
        switch t.s {
        case "use", "usee", "goals", "os", "mode":
			erro(ctx, "$:%s: is obsoleted, use $(.$s) instead", t.s, t.s).trace()
        default:
            if o := l.resolve(ctx, t, t.s); false {
				erro(ctx, "resolve '%v' failed", lhs)
				note(ctx, "parser is here (tok=%s)", tok)
				note(ctx, "parser to go here (tok=%s, lit=%s)", l.p.tok, l.p.lit).trace()
            } else if !isNull(o) {
				lhs = o
			} else if tok == SELECT_PROG2 {
				res = _null(_position(ctx)) // ignore
				return
			} else {
				erro(ctx, "%v: '%v' is undefined (name=%v, obj=%v)", l.project, lhs, t, o)
				note(ctx, "%v: parser is here (name=%s, tok=%s)", l.project, t.s, tok)
				note(ctx, "%v: parser to go here (tok=%s, lit=%s)", l.project, l.p.tok, l.p.lit).trace()
            }
        }
    case *compound: // for cases like '.foo'
		name := lhs.string(ctx)
        if o := l.resolve(ctx, t, name); false {
			erro(ctx, "resolve arrow object '%v' (%s) error", lhs, name).trace()
        } else if !isNull(o) {
			lhs = o
		} else if tok == SELECT_PROG2 {
			res = _null(_position(ctx)) // ignore
			return
		} else {
			erro(ctx, "'%v' is undefined", lhs).trace()
        }
	case *globpat:
		erro(ctx, "arrow of glob: %v", lhs).trace()
	}

	var rhs = l.composite(ctx)

	if checkpoints {
		if _, y := optionalize(ctx, rhs); y {
			erro(pc(ctx,rhs), "unexpected optional expr: %v", tv(rhs)).trace()
		} else if isNull(rhs) {
			erro(pc(ctx,rhs), "unexpected null expr: %v", ts(rhs)).trace()
		}
	}

	return _arrow(pos, tok, lhs, rhs)
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

	ctx = p_braced_ctx{ctx}

	l.p.expect(ctx, LBRACE)

	if l.p.tok == RBRACE {
		x = &null{l.p.valbase()}
		l.p.spaces(ctx)
		l.p.step(ctx) // consumes }
		return
	}

	if /* checkpoints && */ l.p.tok != LPAREN && !l.p.scanner.bits.isBrace() {
		erro(ctx, "wrong scanstate: %v, %v, %v", l.p.tok, l.p.lit, l.p.scanner.scanstate).trace()
	}

	if l.p.tok == LBRACK { // OBSOLETE: {[...]}
		erro(ctx, "syntax error, use {(modifier)} for modification").trace()
	} else if l.p.tok == LPAREN {
		x = l.modification(ctx)
		l.p.spaces(ctx)
		l.p.expect(ctx, RBRACE)
		return
	} else if l.p.tok == ASSIGN { // =
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
			x = &raw{valbase{l.p.Position()}, l.expr(ctx).string(ctx)}
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
			x = l.glob(ctx, nil)
			l.p.spaces(ctx)
			l.p.expect(ctx, RBRACE)
			return

		case REGEX: // {=regex ...}
			l.p.step(ctx)
			l.p.scanner.addBits(isBraceRaw)

			// Trim leading spaces differently to avoid messing the scan states.
			// NOTE: the first SPACE and WORD do not become RAW.
			for l.p.tok == SPACE || (l.p.tok == RAW && l.p.lit == " ") { l.p.step(ctx) }
			return l.p.regex(ctx)

		case WORD:
			switch t := l.p.lit; t {
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

			case "word":
				x = l.braced_word(ctx)
				l.p.expect(ctx, RBRACE)
				return

			case "fullname":
				x = l.braced_fullname(ctx)
				l.p.expect(ctx, RBRACE)
				return

			case "defs":
				x = l.braced_defs(ctx)
				l.p.expect(ctx, RBRACE)
				return
			}

			erro(pc(ctx,l.p), "unsupported braced type: %v %v", l.p.tok, l.p.lit).trace()
			return

		default:
			l.p.next(ctx, true)
		}
	} else { // {...}
		if v := l.values(ctx); len(v) == 0 {
			x = _null(pos)
		} else if len(v) == 1 {
			x = disjunction{v[0]}
		} else {
			x = disjunction{_list(v...)}
		}
		l.p.spaces(ctx)
		l.p.expect(ctx, RBRACE)
		return
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
			var a = s.auto(ctx, arg.string(ctx))
			s.alias(ctx, a, strconv.Itoa(len(p.ruparas)+1))
			p.ruparas = append(p.ruparas, a)
		default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
			erro(ctx, "bad parameter form (%v)", tv(arg)).trace()
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
			erro(ctx, "unexpected colon").trace()
		} else if l.p.spaces(ctx) ; !l.p.is_end_of_line() {
			var val = l.expr(selection{ctx})

			if x, y := val.(*globpat); y && x.len() == 1 {
				if z, y := x.elems[0].(*globrange); y {
					erro(ctx, "use {%v} instead", z.Value).debug()
				} else if z, y := x.elems[0].(*group); y {
					erro(ctx, "use {%v} instead", z.elems[0]).debug()
				} else {
					note(ctx, "use {%v} instead", x.elems[0]).debug()
				}
			}

			res = append(res, merge(val)...)
			if l.p.tok == SPACE { l.p.next(ctx, true) }
		}
	}
	return
}

// If lhs is set, result list elements which are identifiers are not resolved.
func (l ul) values(ctx Context, a ...any) (values []Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "values")) }

	for _, i := range a {
		switch v := i.(type) {
		case Value: values = append(values, v)
		default:
			erro(ctx, "unsupported value: %v", ts(i)).trace()
		}
	}

	for l.p.spaces(ctx); !l.p.is_list_term(ctx); l.p.spaces(ctx) {
		var prev = l.p.pos
		if values = append(values, l.expr(ctx)); l.p.pos == prev {
			erro(ctx, "bad: %v %v; %v", l.p.tok, l.p.lit, values).trace()
		}

		// If there's a comment right after the parsed expression, we break
		// the expression list to treat the end-of-line comment like a LINEND.
		if l.p.tok == EOF || l.p.tok == LINEND || l.p.lineComment != nil { break }
	}
	return
}

func (l ul) list(ctx Context, a ...any) *list {
	return _list(l.values(ctx, a...)...)
}

func (l ul) group(ctx Context) *group {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "group")) }

	ctx = p_group_ctx{p_aware_comma{ctx}}

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
		next := l.list(ctx)

		if l.p.pos == p { erro(ctx, "syntax error").trace() }

		if !converted {
			elems = []Value{ _list(elems...), next }
			converted = true
		} else {
			elems = append(elems, next)
		}
	}
	l.p.expect(ctx, RPAREN)
	return _group(_position(ctx), elems...)
}

func (l ul) argumented(ctx Context, x Value) *argumented {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "argumented")) }

	ctx = p_group_ctx{p_aware_comma{ctx}}

	l.p.next(ctx, true) // skip LPAREN

	var a = []Value{ l.list(ctx) }
	for l.p.tok != RPAREN && l.p.tok != LINEND && l.p.tok != EOF {
		switch l.p.tok {
		case COMMA: l.p.next(ctx, true) // skip COMMA
		case BAR, SEMICOLON:
			if false {
				a = append(a, l.punct(ctx))
				l.p.spaces(ctx)
			} else {
				erro(ctx, "unexpected punctuation: %v", l.p.tok).trace()
			}
		}
		a = append(a, l.list(ctx))
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

		p := l.p.pos

		switch l.p.tok {
		case PCON, RBRACE, RPAREN, COMMA, SPACE, LINEND, EOF:
			return
		case SELECT_PROP, SELECT_PROG1, SELECT_PROG2:
			if truly(ctx, can_select{}) {
				return
			} else {
				v = l.punct(ctx)
			}
		case STAR, DAST, QUE:
			v = l.globmeta(ctx) // * ** ?
		case LBRACK:
			v = l.globrange(ctx) // [abc0-9xyz]
		case DOT:
			v = l.punct(ctx)
		default:
			v = l.unary(ctx)
		}

		if l.p.pos == p { erro(ctx, "syntax error").trace() }

		if checkpoints && truly(ctx, is_test_mode{}) && l.project != nil {
			switch l.project.spec {
			case "testdata/value/glob":
				if strings.Contains(ts(v), "{=path ") {
					erro(ctx, "%v %v %v", g, ts(v), l.p.tok).trace()
				}
			}
		}

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
					erro(ctx, "too many %").trace()
				default:
					switch perc.Suffix = l.expr(ctx); perc.Suffix.(type) {
					case *argumented, *path:
						erro(ctx, "incorrect: %v %v", x, ts(perc.Suffix)).trace()
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

func (p *parser) regex(ctx Context) (_ Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "regex")) }

	var rx string
	var pos = p.Position()

	ctx = p_regex{ctx}

	if checkpoints {
		if !p.scanner.bits.isBrace() {
			erro(ctx, "wrong scan state: %v", p.scanner.scanstate).trace()
		}
		if !p.scanner.bits.isBraceRaw() {
			erro(ctx, "wrong scan state: %v", p.scanner.scanstate).trace()
		}
	}

	for ; p.tok != RBRACE && p.tok != EOF; p.scan(ctx) {
		if p.tok == ESCAPE { rx += "\\" }
		rx += p.lit
	}

	p.expect(ctx, RBRACE)

	var err error
	var x = &regexpat{valbase{pos}, nil} // TODO: correct regexp pattern value
	if x.Regexp, err = regexp.Compile(rx); err != nil {
		erro(pc(ctx,p), "regex: %v", err).trace()
	}
	return x
}

func (l ul) pair(ctx Context, x Value) *pair {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "pair")) }

	l.p.step(ctx)

	var y Value
	if l.p.is_list_term(ctx) {
		y = _null(_position(ctx))
	} else {
		y = l.expr(ctx)
	}

	return makePair(x, y)
}

func (l ul) flag(ctx Context) flag {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "flag")) }

	l.p.step(ctx) // skip dash '-'

	// exclude "-)" "-]" "-}" "-\n", "-=", "-:", etc.
	if l.p.is_end_of_line() || l.p.is_list_term(ctx) || l.p.tok == SPACE || l.p.tok == RECIPE {
		return flag{_null(_position(ctx))}
	}

	var x = l.unary(ctx)

composeloop:
	for l.p.tok != EOF {
		p := l.p.pos
		switch l.p.tok {
		case DOT:
			x = l.dot(ctx, x)
		case PCON:
			x = l.path(ctx, x)
		case CLOSURE, DELEGATE, MINUS:
			x = compose(ctx, x, l.unary(ctx))
		default:
			if l.p.tok.is_closure() || l.p.tok.is_delegate() {
				x = compose(ctx, x, l.unary(ctx))
			} else {
				break composeloop
			}
		}
		if l.p.pos == p { erro(ctx, "syntax error").trace() }
	}

	if x == nil {
		erro(ctx, "nil flag name").trace()
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
	p := &punct{l.p.valbase(), l.p.tok}
	l.p.step(ctx)
	return p
}

func (l ul) escape(ctx Context) *escaped {
	v := &escaped{l.p.valbase(), l.p.lit}
	if l.p.scanner.bits&isRecipes != 0 {
		note(ctx, "%v %v", l.p.tok, l.p.lit).debug(2)
	}
	l.p.expect(ctx, ESCAPE)
	return v
}

func (l ul) literal(ctx Context) (_ Value) {
	tok, lit, pos := l.p.tok, l.p.lit, l.p.Position()

	l.p.step(ctx)

	// ESCAPE is handled in value.EscapeChar
	switch tok {
	case BAR: erro(ctx, "`|` is deprecated, change the modifiers!").trace()
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
		if l.p.pos == p { erro(ctx, "syntax error").trace() }
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

func (l ul) tilde(ctx Context) (res Value) {
	res = _punct(ctx, l.p.tok)
	l.p.step(ctx) // FIXME: ~user
	return
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

	ctx = p_aware_dot{ctx}

	l.p.step(ctx)

	if l.p.tok == PCON && x == nil {
		x = _punct(ctx, DOT)
	} else {
		t := &punct{valbase{_position(ctx)}, DOT}
		if x == nil {
			x = t
		} else {
			x = compose(ctx, x, t)
		}
	}

	for !l.p.is_dot_term(ctx) {
		p := l.p.pos
		x = compose(ctx, x, l.unary(ctx))

		if l.p.pos == p { erro(ctx, "syntax error").trace() }

		switch l.p.tok {
		case DOT:
			x = compose(ctx, x, &punct{l.p.valbase(), DOT})
			l.p.step(ctx) // skips '.'
		}
	}
	return x
}

func (l ul) path(ctx Context, start Value) (res *path) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "path")) }

	if start == nil {
		erro(ctx, "nil path starter").trace()
	}

	ctx = p_path{ctx}

	switch t := start.(type) {
	case     *path: res = t
	case   *strlit: res = makePath(splitPathStr(ctx, t.s)...)
	case  *strcomp: res = makePath(splitPathStr(ctx, t.string(ctx))...) // FIXME: dont final here
	default:        res = makePath(start)
	}

	for l.p.tok == PCON {
		// skips repeated '/' sequence
		for l.p.step(ctx); l.p.tok == PCON; l.p.step(ctx) {}

		switch l.p.tok {
		case LPAREN, LBRACE, RPAREN, RBRACE, COMMA, SPACE, LINEND:
			res.elems = append(res.elems, _punct(ctx, PTAIL)) // after the last '/'
			return
		}

		p := l.p.pos
		elem := l.unary(ctx)

		if l.p.pos == p { erro(ctx, "syntax error").trace() }
		if x, y := elem.(*list); y && x.len() == 1 {
			elem = x.elems[0]
		}

		switch l.p.tok {
		case DOT: // .
			elem = l.dot(ctx, elem)
		case STAR, DAST, QUE, LBRACK: // * ** ? [
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

	var u = &url{ Scheme: scheme }

	if false && checkpoints && truly(ctx, is_test_mode{}) {
		defer func(t token) {
			if l.project.name == "testvalue" {
				note(ctx, "%v %v", u, l.p.tok).debug(3)
			}
		} (l.p.tok)
	}

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
		h.elems = append(h.elems, x, _punct(ctx,DOT))

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
		if p == nil { p = makePath(_punct(ctx,PROOT)) }

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
				erro(ctx, "unexpected %v in url", l.p.tok).trace()
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
    if !pos.IsValid() { pos = _position(ctx) }
    if !pos.IsValid() { pos = l.p.Position() }
	if str == "" {
		erro(ctx, "resolve no-name : %v", ts(name)).trace()
	}

	if d := auto_find(ctx, str); d != nil {
		return d
	}

	var s = _scope(ctx)

	if checkpoints {
		if t := l.scope(); s != t {
			erro(ctx, "%s != %s", s.comment, t.comment).trace()
		}
	}

	if l.project == nil || s != l.project.scope {
		if _, o := s.find(str); o != nil {
			return o
		}
	}

	if l.project != nil {
		if o := l.project.resolve(ctx, str); o != nil {
			return o
		}
	}

	if truly(ctx, is_auto{str}) {
		if a := s.auto(ctx, str); a != nil {
			return a
		}
		erro(ctx, "auto unresolved: %v", ts(name)).trace()
	}

	if truly(ctx, is_config_mode{}) {
		// Create an empty def if referred in configuration.sm.
		result, _ = s.set(ctx, str, defConfRef)
		return
	}

	if l.project != nil {
		if c := l.project.configure; c != nil {
			return c.resolve(ctx, str)
		}
	}
    return
}

func (l ul) closuredelegate_obj(ctx Context, tok token, name Value, isClosure bool) (str string, obj Value) {
	if x, y := name.(*argumented); y { name = x.Value }
	if _, y := name.(cond); !y {
		if v := name.expand(ctx) ; v != nil {
			name = v
		} else {
			erro(ctx, "%v is nil", ts(name)).trace()
		}
	}

	switch t := name.(type) {
	case  *def: return t.name, t
	case entry: return t.ident(ctx), t
	}

	if indeterminate(ctx, name) {
		return "", name
	}

	str = name.string(ctx)

	if tok == LBRACE {
		if t := l.project._entries(ctx, name, false); t != nil {
			obj, _ = t[0].(object)
			return
		} else {
			erro(ctx, "resolved %v is nil", ts(name)).trace()
		}
	}

	if str == "" {
		switch name.(type) {
		case cond, *arrow: return "", name
		default:
			erro(ctx, "empty name: %s", ts(name)).trace()
		}
	}

	if t := l.resolve(ctx, name, str) ; t != nil {
		obj, _ = t.(object)
		return
	}

	if isClosure || truly(ctx, p_is_undef{}) {
		obj = name // recursive delegation or closure
		return
	}

	errostack(pc(ctx,name), 32, "undefined %v : %s → %s", name, ts(name), str).trace()
	return
}

func (l ul) auto_arg0(ctx Context, tokLp token, isClosure bool) (_ Value) {
	if tokLp != LPAREN {
		erro(ctx, "auto: incorrect left paren: %v", tokLp).trace()
	}

	ac := automatic{Context:ctx, defs:make(defs_map)}
	ctx = &ac

	var vals []Value
	var p = l.p
	for p.spaces(ctx); !p.is_list_term(ctx); p.spaces(ctx) {
		var val = l.expr(ctx)
		vals = append(vals, val)

		var s string
		if x, y := val.(*pair); y {
			s, val = x.key.string(ctx), x.val
		} else {
			s = val.string(ctx)
			val = nil
		}
		if s == "" {
			erro(ctx, "auto: %v is empty", val).trace()
		} else {
			ac.set(ctx, defVoid, s, val)
		}

		if p.tok == COMMA || p.tok == EOF || p.tok == LINEND || p.lineComment != nil {
			break
		}
	}

	return _list(vals...)
}

func (l ul) args(ctx Context, name string, tokLp token, isClosure bool) (args []Value) {
	switch name {
	case "auto":
		args = append(args, l.auto_arg0(ctx, tokLp, isClosure))
		if !isClosure { ctx = p_auto_ctx{ctx} }

	case "case":
		args = append(args, l.list(ctx))
		ctx = p_undef_ctx{ctx}

	case "foreach":
		o := objbase{valbase{l.p.Position()}, l.scope()}
		args = append(args, l.list(ctx))
		ctx = p_foreach_txt{ctx, &auto{knownobject{o, "_"}}}

	case "and", "or":
		ctx = p_undef_ctx{ctx}
		args = append(args, l.list(ctx))

	default:
		args = append(args, l.list(ctx))
	}

	for l.p.tok == COMMA {
		l.p.next(ctx, true) // consumes COMMA
		args = append(args, l.list(ctx))
	}
	return
}

func (l ul) abc(ctx Context, isClosure, special bool) (tok token, obj Value, args, opts []Value) {
	var name Value
	var str string

	tok, str = l.p.tok, l.p.lit

	l.p.step(ctx)

	if special {
		if obj = l.resolve(ctx, nil, str); obj == nil {
			errostack(ctx, 8, "not defined %v (name=%s)", tok, str).trace()
		}
		return
	}

	switch l.p.tok {
	case LPAREN, LBRACE: // $(...), ${...}
		tok = l.p.tok // use LPAREN, LBRACE
		l.p.step(ctx) // skips LPAREN, LBRACE

		if l.p.tok == SPACE {
			erro(ctx, "unexpected spaces").trace()
		}

		if name = l.expr(selection{ctx}); name == nil {
			erro(ctx, "%v : no name parsed", tok).trace()
		}

		// For examples: foo?  foo(a,b,c)?
		if x, y := optionalize(ctx, name); y { name = x }

		if a, y := name.(*argumented); y {
			name, opts = a.Value, merge(a.args...)

			// For examples: foo?(a,b,c)
			if x, y := optionalize(ctx, name); y { name = x }

			for _, v := range opts {
				if p, y := v.(*pair); y { v = p.key }
				if _, y := v.(flag); !y {
					erro(ctx, "not a flag: %v", ts(v)).trace()
				}
			}
		}

		if name == nil {
			erro(ctx, "name %v is nil", ts(name)).trace()
		}

		str, obj = l.closuredelegate_obj(ctx, tok, name, isClosure)

		if (tok == LPAREN && l.p.tok != RPAREN) || (tok == LBRACE && l.p.tok != RBRACE) {
			args = l.args(ctx, str, tok, isClosure)
		}

		switch tok {
		case LPAREN: l.p.expect(ctx, RPAREN)
		case LBRACE: l.p.expect(ctx, RBRACE)
		}

	default:
		if !isClosure { // $(...), disabled $name.
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v` or `%v` or quotes", LPAREN, LBRACE).trace()
		}

		if !(l.p.tok == STRING || l.p.tok == STRCOMP) {
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v`, `%v` or quotes, not %v %v", LPAREN, LBRACE, l.p.tok, l.p.lit).trace()
		}

		tok = l.p.tok

		// &'xxxx' or &"xxxx"
		if name = l.expr(ctx); name == nil {
			erro(ctx, "parsed name is nil").trace()
		}

		if indeterminate(ctx, name) {//, /* expandClosure */final
			erro(ctx, "name '%v' is closured", ts(name)).trace()
		}

		str, obj = l.closuredelegate_obj(ctx, tok, name, isClosure)
	}

	if obj == nil && str != "" {
		if l.project.ext.Plugin != nil {
			if t, e := l.project.ext.Lookup(str); e == nil && t != nil {
				erro(ctx, "TODO: convert ext symbol: %v : %v", name, ts(t)).trace()
			}
		}
	}
	return
}

func (l ul) closuredelegate(ctx Context, isClosure, special bool) (result Value) {
	if l_traverse.enabled {	defer un(l_trace(l_traverse, "closuredelegate")) }

	pos := l.p.Position()
	tok, obj, args, opts := l.abc(p_aware_comma{ctx}, isClosure, special)

	if obj == nil {
		erro(ctx, "%v : nil symbol", tok).trace()
	}

	if isClosure {
		return makeClosure(pos, tok, obj, opts, args...)
	} else if x, y := obj.(*def); y && x.o == defAuto {
		if !truly(ctx, is_auto_preserved{x.name}) {
			return x.value
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
				erro(pc(ctx,x), "syntax error: %v", ts(x)).trace()
			}
		}
	} ()

	switch l.p.tok {
	case ASSIGN: // example: '=xxx'
		if !truly(ctx, left_hand_side{}) {
			var v Value
			var p = l.p.loc(pos)
			if l.p.step(ctx); l.p.is_list_term(ctx) {
				v = _null(p)
			} else {
				v = l.expr(ctx)
			}
			return &pair{_null(p), v}
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
						erro(ctx, "bad url token: %v %v", l.p.tok, l.p.lit).trace()
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

	case CLOSURE:
		return l.closuredelegate(ctx, true, false)

	case DELEGATE:
		return l.closuredelegate(ctx, false, false)

	case ESCAPE: // \
		return l.escape(ctx)

	case LPAREN: // (
		return l.group(ctx)

	case LBRACE: // {
		return l.braced(ctx)

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

	case STAR, DAST, QUE, LBRACK: // * ** ? [    -- NOTE: ? is an exception
		return l.glob(ctx, nil)

	case MINUS:
		return l.flag(ctx)

	case EXC:
		return l.negative(ctx)

	case PCON: // The root of the path
		return _punct(ctx, PROOT)

	case TILDE: // ~
		return l.tilde(ctx)

	case DOT: // .
		return l.dot(ctx, nil)

	case DOTDOT: // . ..
		tok, pos := l.p.tok, l.p.Position()
		if l.p.step(ctx) ; l.p.tok == PCON {
			return _punct(ctx, tok)
		} else {
			return &punct{valbase{pos}, tok}
		}

	default:
		if t := l.p.tok.is_closure(); t || l.p.tok.is_delegate() {
			return l.closuredelegate(ctx, t, true)
		}
		if l.p.tok.is_keyword() { // keywords here are words
			return l.p.bare(ctx)
		}
	}

	if l.p.tok != EOF {
		ctx = pc(ctx, l.p.loc(pos))
		erro(ctx, "unexpected %v '%s'", l.p.tok, l.p.lit)
		note(ctx, "x: %v %v", x, ts(x))
		note(ctx, "%v", l.p.scanner.scanstate).trace()
	}

	if l.p.lineComment != nil {
		for _, c := range l.p.lineComment.comments {
			erro(ctx, "# %s", c.string).trace()
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
		return l.dot(ctx, x)//if truly(ctx, aware_token{DOT}) { x = l.dot(ctx, x) }
		// TODO: if truly(ctx, aware_token{DOTDOT}) { x = l.dotdot(ctx, x) }

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
	if checkpoints && truly(ctx, is_test_mode{}) { defer l.expr_check(ctx, &x) }

	x = l.composite(ctx)

	if checkpoints && truly(ctx, is_test_mode{}) { defer l.expr_check_1(ctx, x, l.p.tok, &x) }

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
	case COLON, COMPOSED, RPAREN, RBRACK, RBRACE, RAW, SPACE, SEMICOLON, LINEND, EOF:
		return // terminate

	case COMMA:
		if truly(ctx, aware_token{COMMA}) { return }

	case ASSIGN: // example: key=value
		if !truly(ctx, left_hand_side{}) {
			return l.pair(ctx, x)
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

	case QUE: // ?
		if !truly(ctx, p_is_glob{}) {
			l.p.step(ctx)
			x = condish(ctx, x)
			goto composeloop
		}

	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2:
		if truly(ctx, can_select{}) {
			x = l.arrow(ctx, x)
			goto composeloop
		}
	}

	if truly(ctx, p_is_strcomp{}) { return }

	var p = l.p.pos
	var y = l.unary(ctx)// NOTE: it's unary, not composite
	if l.p.pos == p {
		erro(ctx, "syntax error: %v (%v %v %v)", x, tv(x), tv(y), l.p.tok).trace()
	}

	x = compose(ctx, x, y)

	switch l.p.tok { case COMMENT, SPACE, LINEND, EOF: return }

	if 9999 < n { erro(ctx, "too many compose: %v (%d)", x, n).trace() }

	n += 1 ; goto composeloop // compose as many as possible
}

func (l ul) braced_and(ctx Context) (res Value) {
	var va []Value

	l.p.expect(ctx, AND) // consumes `and`

andloop:
	for l.p.tok != EOF {
		switch l.p.spaces(ctx); l.p.tok {
		case COMMA: l.p.step(ctx); continue andloop
		case RBRACE: break andloop
		}

		v := l.expr(p_aware_comma{ctx})
		w := v.expand(pc(_final(ctx), v))

		if false && strings.HasSuffix(l.p.scanner.file.Name(), "/configure/.base/.template") {
			note(pc(ctx,v), "%s : %v → %v : %v", l.project.name, tv(v), tv(w), l.p.tok).debug(3)
		}

		va = append(va, merge(w)...)
	}

	l.p.expect(ctx, RBRACE)

	for _, a := range va {
		if a.true(ctx) { res = a } else { return nil }
	}
	return
}

func (l ul) braced_or(ctx Context) (_ Value) {
	var va []Value

	l.p.expect(ctx, OR) // consumes `or`

orloop:
	for l.p.tok != EOF {
		switch l.p.spaces(ctx); l.p.tok {
		case COMMA: l.p.step(ctx); continue orloop
		case RBRACE: break orloop
		}

		v := l.expr(p_aware_comma{ctx})
		w := v.expand(pc(_final(ctx), v))

		if false && strings.HasSuffix(l.p.scanner.file.Name(), "/configure/.base/.template") {
			note(pc(ctx,v), "%s : %v → %v : %v", l.project.name, tv(v), tv(w), l.p.tok).debug(3)
		}

		va = append(va, merge(w)...)
	}

	l.p.expect(ctx, RBRACE)

	for _, a := range va {
		if a.true(ctx) { return a }
	}

	if checkpoints && truly(ctx, is_test_mode{}) {
		if l.project.name == "configure.base" {
			erro(pc(ctx,l.p.Position()), "nil: %v", va).trace()
		}
	}
	return
}

func (l ul) braced_for(ctx Context) (res Value) {
	l.p.expect(ctx, FOR) // consumes `for`

	erro(ctx, "TODO: {=for ...}").trace()

	l.p.expect(ctx, RBRACE)
	return
}

type foreach_text struct { Context }
func (f foreach_text) inner() Context { return f.Context }
func (f foreach_text) cast(t reflect.Type) Context { return icast(f, t) }
func (f foreach_text) do(c Context, o any) (_ any) {
    switch t := o.(type) {
    case find_auto:
		if s := t.string; s == "_" {
			var a *automatic
			switch t := f.Context.(type) {
			case *automatic: a = t
			case *builtin_foreach: a = &t.automatic
			}
			if a != nil {
				if _, y := a.defs[s]; !y {
					return
				}
			}
		}
    }
	return f.Context.do(c, o)
}

func (l ul) braced_foreach(ctx Context) Value {
	l.p.expect(ctx, FOREACH)
	l.p.spaces(ctx)

	var vals []Value

valsloop:
	for {
		switch l.p.tok {
		case COMMA, RBRACE, LINEND, END:
			break valsloop
		}

		ac := p_aware_comma{ctx}
		vals = append(vals, l.expr(ac))
		l.p.spaces(ac)
	}

	cc := automatic{ Context:ctx, defs:make(defs_map) }
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
				erro(ctx, "nil ; %v", l.p.tok).trace()
			}
		}
	}

	l.p.expect(ctx, RBRACE)

	var va []Value
	var recipe_start = truly(ctx, is_recipe_start{})
	for _, val := range vals {
		for _, elem := range merge(val.expand(_final(ctx))) {
			if isEmpty(elem) { continue }
			if indeterminate(ctx, elem) { elem = disjunction{elem} }

			// NOTE: don't use defAuto, it's for codeblock auto only
			cc.set(ctx, defVoid, "_", elem)

			var a = xmerge(final{foreach_text{&cc}}, temps...)
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
	var elems = expand(_final(ctx), l.braced_elems(ctx)...)

	if checkpoints && truly(ctx, is_test_mode{}) {
		defer l.braced_str_check(ctx, elems, &res)
	}

	for _, v := range elems {
		if indeterminate(ctx, v) {
			return &strval{valbase{pos}, elems}
		}
	}

	var s string
	for _, v := range elems { s += v.string(ctx) }
	return &strlit{valbase{pos}, s}
}

func (l ul) braced_word(ctx Context) (res Value) {
	l.p.next(ctx, true) // resumes 'word'

	var pos = l.p.Position()
	var elems = expand(_final(ctx), l.braced_elems(ctx)...)

	if checkpoints && truly(ctx, is_test_mode{}) {
		defer l.braced_word_check(ctx, elems, &res)
	}

	for _, v := range elems {
		if indeterminate(ctx, v) {
			erro(ctx, "indeterminate: %v", ts(v)).trace()
		}
	}

	var s string
	for _, v := range elems { s += v.string(ctx) }
	return &word{valbase{pos}, s}
}

func (l ul) braced_defs(ctx Context) (res Value) {
	var capture int

	l.p.step(ctx) // resumes 'defs'

	if l.p.tok == LPAREN {
		l.p.next(ctx, true) // resumes '('

		if l.p.tok == INTEGER {
			if i, e := strconv.Atoi(l.p.lit); e == nil {
				capture = i
			} else {
				erro(ctx, "wrong integer: %v", e).trace()
			}
		} else {
			erro(ctx, "expected integer, found %v '%s'", l.p.tok, l.p.lit).trace()
		}

		l.p.next(ctx, true)
		l.p.expect(ctx, RPAREN)
	}

	l.p.spaces(ctx)

	var pats = expand(_final(ctx), l.braced_elems(ctx)...)

	if checkpoints && truly(ctx, is_test_mode{}) {
		defer l.braced_defs_check(ctx, pats, &res)
	}

	for _, v := range pats {
		if indeterminate(ctx, v) {
			erro(ctx, "indeterminate: %v", ts(v)).trace()
		}
	}

	var vals []Value

defsloop:
	for name, _ := range l.project.elems {
		var pos Position
        var str string
        for _, pat := range pats {
            var neg bool
            if x, y := pat.(negative); y { pat, neg = x.Value, y }

            var a, _, c = pat.match(ctx, name)
            if a && neg { continue defsloop }
            if a || neg {
                if capture <= 0 || 0 == len(c) {
                    pos, str = pat.Position(), name
					if true { break }
                } else if capture <= len(c) {
                    pos, str = pat.Position(), c[capture-1]
					if true { break }
                }
            }
        }
		if str != "" {
			vals = append(vals, _word(pos, str))
		}
	}
	return ease(ctx, vals)
}

func (l ul) braced_file(ctx Context) (res Value) {
	l.p.next(ctx, true)

	var elems []Value

	for _, elem := range l.braced_elems(ctx) {
		if f := l.project.file(ctx, elem); f != nil {
			elems = append(elems, f)
			continue
		} else {
			var s = elem.string(ctx)
			var a = []any{stat_nonexist{true}}
			if !isAbsOrRel(s) {
				a = append(a, stat_dir{l.project.absPath})
			}
			if f = stat(ctx, s, a...); f != nil {
				elems = append(elems, f)
				continue
			}
		}
		erro(ctx, "not a file: %v", tv(elem)).trace()
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
		var v = elem.expand(_final(ctx))
		if indeterminate(ctx, v) {
			v = fullname{v}
		} else {
			v = as{v}.fullname(ctx, l.project)
		}
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
				erro(ctx, "unexpected token: %v", l.p.tok).trace()
			}
			l.p.next(ctx, true)
		case FLOAT:
			n = l.expr(ctx).float(ctx)
		default:
			n = l.expr(ctx).int(ctx)
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
		erro(ctx, "nil const, %v, %v %v", tok, l.p.tok, l.p.lit).trace()
	}

	l.p.spaces(ctx)
	l.p.expect(ctx, RBRACE)
	return
}

func (l ul) braced_const(ctx Context, tok token) (x Value) {
	l.p.next(ctx, true)

	var pos = l.p.Position()

	if l.p.spaces(ctx); l.p.tok != RBRACE {
		erro(ctx, "expecting right-brace, %v %v", l.p.tok, l.p.lit).trace()
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
		erro(ctx, "nil const, %v, %v %v", tok, l.p.tok, l.p.lit).trace()
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

	x = &none{valbase{pos},v}
	l.p.spaces(ctx)
	l.p.expect(ctx, RBRACE)
	return
}

func (l ul) braced_null(ctx Context) (x Value) {
	l.p.next(ctx, true)

	x = &null{valbase{l.p.Position()}}
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
	str := name.string(ctx)
	if str == "" {
		erro(ctx, "empty name : %s : %s", ts(name), str).trace()
	}

	if /* self && */ l.project.name == str {
		return l.project
	} else if o := l.resolve(ctx, name, str); o == nil {
		erro(ctx, "%s : undefined %s : %v", l.project, str, ts(name)).trace()
		return
	} else if x, y := o.(*project); !y && x != nil {
		erro(ctx, "%s : %v is not a project", l.project, ts(o)).trace()
		return
	} else {
		return x
	}
}

// ----------------------------------------------------------------------------
// Clauses & Declarations

type clauseopts struct {
	general_opts

    keyword token // e.g. use, files, eval, etc.

    skip bool // e.g. -cond({=false}), -if({=no})

	conds []Value `if,cond,where`

    values, remainder []Value // all values (unparsed) and remainder
	spec []Value
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
            switch s = t.Value.string(ctx); s {
            //case "nouse", "unuse": opts.unuse = true
            case "reuse": opts.reuse = true
            default: params = append(params, prop)
            }
        case *pair: // -param=value
            switch tt := t.key.(type) {
            case flag:
                switch s = tt.Value.string(ctx); s {
                case "use": useList = append(useList, t.val)
                default: params = append(params, prop)
                }
            default:
                erro(ctx, "parameter `%v' unsupported `%T`", prop, prop)
            }
        case *argumented: // -param(value)
            switch tt := t.Value.(type) {
            case flag:
                switch s = tt.Value.string(ctx); s {
                case "use": useList = append(useList, t.args...)
                default: params = append(params, prop)
                }
            default:
                erro(ctx, "parameter `%v' unsupported `%T`", prop, prop)
            }
        default:
            erro(ctx, "parameter `%v` unsupported `%T`", prop, prop)
        }
    }
    return
}

func (l ul) use(ctx Context, doc *commentgroup, g *clauseopts, _ int) {
	if l.p.imports = append(l.p.imports, &use_spec{ g.spec }); g.skip {
		// TODO: maybe give some information
		return
	}

	var specVal0 Value
	switch v := g.spec[0].(type) {
    case *pair:
        var s string
        if f, ok := v.key.(flag); !ok {
            erro(ctx, "'%v' invalid use spec", v.key)
        } else if s = f.Value.string(ctx); s != "list" {
            erro(ctx, "'%v' invalid use spec, do you mean -list?", v.key)
        }
		specVal0 = v.val
	case *argumented:
		specVal0, ctx = v.Value, &argumented_ctx{ctx, v}
	default:
		specVal0 = v
    }

	var specVals []Value
	for _, val := range xmerge(ctx, specVal0) {
		if !isTrivial(val) { specVals = append(specVals, val) }
	}
	if len(specVals) == 0 {
        erro(ctx, "empty use spec: %v", ts(g.spec[0])).trace()
    }

	var opts useopts
	var args = parse_opts(ctx, &opts, append(g.remainder, g.spec[1:]...)...)
	for _, a := range args {
		if _, ok := a.(flag); ok || true {
			erro(ctx, "unkown use opts: %v", ts(a)).trace()
		}
	}

	for _, specVal := range specVals {
		l.use_spec(ctx, opts, specVal, args...)
	}
	return
}

func (l ul) files(ctx Context, doc *commentgroup, g *clauseopts, _ int) {
	if checkpoints && truly(ctx, is_test_mode{}) { defer l.files_check(ctx) }
	if len(g.spec) != 1 { erro(ctx, "too many properties: %v", g.spec).trace() }

	var p Value
	var patts, paths []Value

	if l.p.tok == SELECT_PROG1 {
		l.p.next(ctx, true) // step forward with spaces skipped
		if l.p.tok == LINEND || l.p.lineComment != nil {
			erro(ctx, "expecting files path").trace()
		}
		p = l.expr(ctx)
	}

	l.p.spaces(ctx)

	if g.skip { return }

	if t := parse_opts(ctx, &g.general_opts, g.remainder...); t != nil {
		erro(ctx, "unsupported opts: %v", t).trace()
	}

	if t := g.spec[0].expand(original{ctx, defExpand1}); t == nil {
		erro(ctx, "nil: %v", g.spec[0]).trace()
	} else if x, y := t.(*group); y {
		patts = merge(x.elems...)
	} else {
		patts = merge(t)
	}

	if p == nil {
		if len(patts) == 1 {
			if x, y := patts[0].(*argumented); y {
				if f, y := x.Value.(flag); y {
					switch f.Value.string(ctx) {
					default: // TODO: parse files options
						erro(ctx, "invalid files flag: %v").trace()
					}
				}
			}
		}
	} else {
		if len(patts) == 1 {
			if f, y := patts[0].(flag); y {
				switch f.Value.string(ctx) {
				default: // TODO: parse files options
					erro(ctx, "invalid files flag: %v").trace()
				}
			}
		}
		switch x := p.expand(original{ctx, defExpand1}).(type) {
		case *group: paths = x.elems
		default: paths = []Value{ x }
		}
	}

	t := l.project.map_files(ctx, patts, paths)

	if checkpoints && truly(ctx, is_test_mode{}) {
		l.files_check_2(ctx, p, patts, paths, t)
	}
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
			switch s := x.Value.string(ctx); s {
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
				erro(ctx, "unsupported flag: %v", tv(a)).trace()
			}
			continue
		}

		var s = a.string(ctx)
		if s == "" {
			erro(ctx, "empty local: %v", tv(a)).trace()
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
		var opts struct {
			optimize Value `opt,optimize`
		}

		for _, op := range parse_opts(_final(ctx), &opts, g.values...) {
			var val Value
			if v, y := op.(*pair); y { op, val = v.key, v.val }
			if v, y := op.(flag); y {
				switch t := val != nil && val.true(ctx); v.Value.string(ctx) {
				case "dd": l.p.dd = t
				case "ddd":
					if val == nil {
						l.ddd = "yes"
					} else if t, y := boolVal(val); y {
						if t { l.ddd = "yes" } else { l.ddd = "" }
					} else {
						l.ddd = val.string(ctx)
					}
				}
			} else {
				erro(ctx, "unsupport flag: %v (%v)", ts(v), val).trace()
			}
		}

		return
	}

	prop0 := g.spec[0]

	if isTrivial(prop0) {
		erro(ctx, "illegal").trace()
	}

	var opts []Value
	if a, y := prop0.(*argumented); y { prop0, opts = a.Value, a.args }

	name := prop0.string(ctx)
	switch name { case "-configuration", "configuration":
		erro(ctx, "configuration is done at parse time", prop0).trace()
	}

	resolved := l.resolve(ctx, prop0, name)

	switch x := resolved.(type) {
	case invoker:
		if b, y := x.(*builtin); y && !b.is_command() {
			erro(ctx, "resolved builtin '%v' is not a command", prop0).trace()
		}
		x.invoke(ctx, opts, g.spec[1:])
		return
	default:
		erro(ctx, "resolved '%v' is %s (%v)", prop0, typeof(resolved), *g).trace()
	}

	/* TODO: if c, y := res.(code); y { ... } */
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
		if t := cond.true(ctx); !t {
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

		erro(ctx, "%v: no specs, remainder: %v", keyword, opts.remainder).trace()

	case LPAREN:
		l.p.next(ctx, true)
		for iota := 0; l.p.tok != RPAREN && l.p.tok != EOF && (l.p.stop == 0 || l.p.pos < l.p.stop); iota++ {
			// TODO: collect documentation comments
			for l.p.tok == SPACE || l.p.tok == LINEND { l.p.next(ctx, true) }
			if l.p.tok == RPAREN || l.p.tok == EOF { break  }
			if opts.spec = l.directive(ctx); true {
				f(ctx, l.p.leadComment, &opts, iota)
			}
			if l.p.tok == COMMA || l.p.tok == LINEND { l.p.next(ctx, true) }
		}
		l.p.expect(ctx, RPAREN)
		l.p.spaces(ctx)
		if l.p.tok != EOF { l.p.linend(ctx) }
		return
	}

	if l.p.tok != LINEND && l.p.tok != EOF && (l.p.stop == 0 || l.p.pos < l.p.stop) {
		if opts.spec = l.directive(ctx); true { f(ctx, nil, &opts, 0) }
		if l.p.tok == COMMA { l.p.next(ctx, true) }
	}
	if l.p.tok != EOF && (l.p.stop == 0 || l.p.pos < l.p.stop) {
		if l.p.spaces(ctx); l.p.lineComment == nil { l.p.linend(ctx) }
	}
}

func for_ident_elems(ctx Context, elems, stems []Value, f func(elems, stems []Value)) {
    for i, elem := range elems {
		if x, y := elem.(*argumented); y {
			var prefix, suffix = elems[:i], elems[i+1:]
			for_idents(ctx, x, func(ident Value, stems2 []Value) {
				var head   = append(prefix, ident)
				var stems3 = append(stems , stems2...)
				for_ident_elems(ctx, suffix, stems3, func(elems, stems []Value) {
					f(append(head, elems...), stems)
				})
			})
			return
		}
	}
    f(elems, stems)
}

func for_idents(ctx Context, idents Value, f func(ident Value, stem []Value)) {
    switch t := idents.(type) {
    case *argumented:
        var args = xmerge(ctx, t.args...)
		for_idents(ctx, t.Value, func(ident Value, stems []Value) {
			for _, arg := range args {
				if !isTrivial(arg) {
					f(compose(ctx, ident, arg), append(stems, arg))
				}
			}
		})
    case *compound:
        for_ident_elems(ctx, t.elems, nil, func(elems, stems []Value) {
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

func (l ul) define(ctx Context, tok token, ident, value Value) (d *def) {
	if checkpoints && truly(ctx, is_test_mode{}) {
		defer define_check(ctx, tok, ident, value, &d)
	}

    var alt object

    switch t := ident.(type) {
    case *argumented:
        erro(ctx, "TODO: multiple defs: %v, args=%v", t.Value, t.args).trace()

    case *group:
        erro(ctx, "TODO: multiple defs: %v", t.elems).trace()

    case *arrow:
        if v := t.expand(_final(ctx)); v == nil {
            erro(ctx, "%v is nil", ts(t)).trace()
        } else if x, y := v.(*def); !y {
            erro(ctx, "%v is not a def: %v", ts(t), ts(v)).trace()
        } else {
            d = x
        }

    default: // *word, *compound, *qualword, *path, flag:
        var name = t.string(ctx)
        if _, y := builtins[name]; y {
            erro(ctx, "`%v` is a builtin name (%v)", ident, name).trace()
        }

        // Resolve base value to derive.
        var prev = l.project.resolve(ctx, name)

        if d, alt = l.project.set(ctx, name, defUndetermined); alt == nil {
            if d == nil {
                erro(ctx, "`%s` is undefined (%v)", name, tv(t)).trace()
			}
        } else if tok == ASSIGN || tok == ASSIGN_EXC {
            if a, y := alt.(*def); !y {
                erro(ctx, "`%v` already defined: %s (%v)", ident, tv(alt), alt.owner()).trace()
            } else if a.owner() == l.project && a.o != defConfRef {
                erro(ctx, "`%v` already defined: %s", ident, tv(alt)).trace()
            } else {
                d = a
            }
        } else if t, y := alt.(*def); !y {
            erro(ctx, "%s: object is not def: %s, %v", name, tv(alt), tv(prev)).trace()
		} else {
           d = t
        }

        if prev == nil || d == nil {
            // no derived value
        } else if x, y := prev.(*def); !y {
            // not a def
        } else if x == nil {
            erro(ctx, "prev def '%s' is nil", name).trace()
		} else if x != d && x.scope != d.scope && alt == nil {
			switch tok {
			case ASSIGN_ADD, ASSIGN_SHI:
				if d.o == defVoid && d.o != x.o { d.origin(ctx, x.o) }
				if !isTrivial(x.value) { d.append(ctx, x.value) }
			}
		}
    }

    if d == nil {
        erro(ctx, "def is nil: %v", ts(ident)).trace()
    }

    if false { d.position = ident.Position() }

	if truly(ctx, is_config_mode{}) {
		defer func() { d.o = defConfig } ()
	}

    switch tok {
    case ASSIGN    : d.set(ctx, defExpand0, value); return //   =
    case ASSIGN_CO1: d.set(ctx, defExpand1, value); return //  :=
    case ASSIGN_CO2: d.set(ctx, defExpand2, value); return // ::=
    case ASSIGN_EXC: d.set(ctx, defExecute, value); return //  !=
    case ASSIGN_QUE: // ?=
		if alt == nil { d.set(ctx, d.o, value) }
		return
    case ASSIGN_ADD: // +=
		if !isTrivial(value) { d.set(ctx, d.o, nil, merge(  value)...) }
		return
    case ASSIGN_SHI: // =+
		if !isTrivial(value) { d.set(ctx, d.o, value, merge(d.value)...) }
		return
    case ASSIGN_SUB:
		if d.value != nil {
			if dv := merge(d.value); len(dv) > 0 { // -=
				var vals []Value
				var sub = merge(value)
			outer1:
				for _, v := range dv {
					for _, sv := range sub { if v.cmp(ctx, sv) == cmpEqual { continue outer1 }}
					vals = append(vals, v)
				}
				d.value = ease(ctx, vals)
			}
		}
		return
    case ASSIGN_SAD, ASSIGN_SSH: // -+=, -=+
        var vals []Value
        var sub = merge(value)
        if d.value != nil {
            if dv := merge(d.value); len(dv) > 0 {
            outer2:
                for _, v := range dv {
                    for _, sv := range sub {
                        if v.cmp(ctx, sv) == cmpEqual { continue outer2 }
                    }
                    vals = append(vals, v)
                }
            }
        }
		switch tok {
		case ASSIGN_SAD: vals = append(vals, sub...) // -+=
		case ASSIGN_SSH: vals = append(sub, vals...) // -=+
		}
        d.value = ease(ctx, vals)
		return
    default:
        erro(ctx, "unknown origin: %v %v %v", d.o, d.name, tok).trace()
		return
    }
}

func (l ul) defvalue(ctx Context, idents []Value, tok token) (value Value) {
	defer l.closescope(l.openscope(tv(idents)))

	vals := l.values(p_defvalue_ctx{ctx})
	l.p.lineComment = nil
	return ease(ctx, vals)
}

func (l ul) assign(ctx Context, idents []Value) (res []*def) {
	if l_traverse.enabled || debugSyntax(ctx, "define") {
		defer un(l_trace(l_traverse, fmt.Sprintf("assign(%s)", idents)))
	}

	var tok = l.p.tok

	l.p.next(ctx, true) // the assign token

	var value = l.defvalue(ctx, idents, tok)

	for _, ids := range idents {
		var ds []*def
		var _ids = ids.expand(_final(ctx))

		for_idents(ctx, _ids, func(ident Value, stems []Value) {
			if d := l.define(ctx, tok, ident, value); d != nil { ds = append(ds, d) }
		})

		if checkpoints {
			var ctx = pc(ctx, ids)
			if len(ds) == 0 && false {
				erro(ctx, "%v : %v %v %v", tv(ids), tv(_ids), tok, ts(value)).trace()
			}
			if len(ds) == 1 && ds[0].value == nil && value != nil && !isNull(value) {
				erro(ctx, "%v : %v %v %v", tv(ids), tv(_ids), tok, ts(value)).trace()
			}
		}

		res = append(res, ds...)
	}

	if checkpoints && truly(ctx, is_test_mode{}) {
		def_idents_check(ctx, idents, value, res)
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
	case "", "value":
		l.p.scanner.pop(isStrcompLine)
		l.p.next(ctx, true) // skip RECIPE or SEMICOLON and parse in list mode
		p = l.p.Position()
		if isList = true; !l.p.is_end_of_line() {
			var c = recipe_ctx{ctx, true, nil, nil} // builtin value recipe
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
				errostack(pc(ctx,p), 16, "parsed nil value, dialect=%s", l.p.dialect).trace()
			}

			if l.p.dialect == "value" {
				// no resolving commands
			} else if t, y := x.(*word); !y {
				// does nothing
			} else if s := l.resolve(ctx, t, t.s); isTrivial(s) {
				erro(pc(ctx,p), "no such symbol: %v, %s → %s; dialect=%s", t.s, ts(x), ts(s), l.p.dialect).trace()
			} else if b, y := s.(*builtin); !y {
				erro(pc(ctx,p), "'%s' is not a command (%s)", t.s, typeof(s)).trace()
			} else if !b.is_command() {
				erro(pc(ctx,p), "'%s' is not a command, use $(%s ...) instead", t.s, t.s).trace()
			} else { x = s }

			if a != nil {
				elems, a.Value = append(elems, a), x
			} else {
				elems = append(elems, x)
			}

			var cmdargs []Value
			var c = recipe_ctx{ctx, true, nil, nil} // builtin recipe

			for l.p.tok != EOF && l.p.tok != SEMICOLON && l.p.tok != LINEND && l.p.lineComment == nil {
				if l.p.spaces(ctx); l.p.lineComment != nil { break }
				if !l.p.tok.is_rule_delim() {
					x = l.expr(&c)
				} else {
					erro(ctx, "unsupported token: %s, %v", l.p.tok, elems).trace()
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

		switch l.p.dialect {
		case "plain", "text": isPlainline = true
		}

		var c = recipe_ctx{ctx, false, nil, nil} // builtin text
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
				erro(ctx, "%v %v", c.elems, c.lines).trace()
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
			erro(ctx, "bad var form (%v)", ts(elem)).trace()
		}

		var v = kv.val
		if x, y := v.(*group); y { v = x.list() }

		if d, a := _scope(ctx).set(ctx, kv.key, defUndetermined, v); a != nil {
			erro(ctx, "'%v' already defined: %v", kv.key, ts(a)).trace()
		} else if d == nil {
			erro(ctx, "'%v' not defined", kv.key).trace()
		}
	}
	return
}

func (l ul) define_configs(ctx Context) {
	for _, t := range l.p.targets {
		if d, a := l.project.set(ctx, t, defConfig); d == nil {
			erro(ctx, "%v : not defined : %v", ts(t), ts(a)).trace()
		} else if a != nil {
			erro(ctx, "%v : already defined : %v", ts(t), ts(a)).trace()
		}
	}
}

func (l ul) modifier(ctx Context) (res *modifier) {
	l.p.spaces(ctx)

	ctx = p_modifier{ctx}

	l.p.expect(ctx, LPAREN)
	l.p.spaces(ctx)

	var name string
	var elems []Value
	var val = l.expr(ctx)
	switch n := val.(type) {
	case *word: name = n.s
	case *delegate, *closure:
		var v = xmerge(_final(ctx), val)
		if len(v) == 0 {
			erro(pc(ctx,val), "empty modifier name: %v", n).trace()
		}

		name, elems = v[0].string(ctx), v[1:]

	default:
		erro(pc(ctx,val), "unsupported modifier: %v", ts(n)).trace()
	}

	if _, y := dialects[name]; y {
		if l.p.dialect == "" {
			l.p.dialect = name
		} else {
			erro(pc(ctx,l.p), "multi-dialects unsupported, already defined '%s'", l.p.dialect).trace()
		}
	} else if _, y = modifiers[name]; !y {
		errostack(pc(ctx,l.p), 24, "`%s` no such dialect or modifier", name).trace()
	}

	for l.p.tok != RPAREN && l.p.tok != EOF {
		l.p.spaces(ctx)

		t := l.p.pos

		if va := l.values(ctx); name == "var" {
			l.p.var_modifier(ctx, va...)
		} else if n := len(va); n == 1 {
			elems = append(elems, va[0])
		} else if n > 1 {
			elems = append(elems, &list{elements{va}})
		} else {
			elems = append(elems, &null{l.p.valbase()})
		}

		if l.p.tok == COMMA { l.p.next(ctx, true) }
		if l.p.pos == t {
			erro(pc(ctx,l.p), "unsupported modifier arg: %v '%v'", l.p.tok, l.p.lit).trace()
		}
	}

	l.p.expect(ctx, RPAREN)

	if val == nil && len(elems) == 0 {
		erro(pc(ctx,l.p), "empty modifier").trace()
	} else {
		res = new(modifier)
		res.position = _position(ctx)
		res.elems = append([]Value{val}, elems...)
	}
	return
}

// example: {(modifier ...)}
func (l ul) modification(ctx Context) *modification {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "modification")) }

	var elems []*modifier
	for l.p.tok != EOF && l.p.tok != LINEND && l.p.tok != RBRACE {
		if m := l.modifier(ctx); m != nil { elems = append(elems, m) }
	}

	// l.p.expect(ctx, /* RBRACK */RBRACE)

	if len(elems) == 0 {
		errostack(ctx, 5, "empty modifier group").trace()
	}
	if l.p.tok == COLON {
		errostack(ctx, 5, "unexpected colon after modifer").trace()
	}
    return &modification{valbase{_position(ctx)}, elems }
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

func (l ul) rule(ctx Context, optvals, targets []Value) (result Value) {
	if l_traverse.enabled || debugSyntax(ctx, "rule") { defer un(l_trace(l_traverse, "rule")) }

	ctx = p_rule_ctx{ctx}

    if l.project != _scope(ctx).project {
		erro(ctx, "mismatched project/scope : %v", targets).trace()
	}

	// TODO: doc = p.leadComment
	var depends, ordered, recipes []Value
	defer l.closescope(l.openscope(tv(targets)))
	defer func() { l.p.dialect, l.p.ruparas = "", nil } ()

	l.p.dialect = ""
	l.p.ruparas = nil

	// NOTE: expand targets to speed up for later usage, it might spend lots of time in
	// project.entry while matching for entry looked up if not expanded right now.
	targets = expand(_final(ctx), targets...)

	if checkpoints && truly(ctx, is_test_mode{}) {
		if targets != nil { l.rule_check_targets(ctx, targets) }
		defer l.rule_check(ctx, targets, &result)
	}

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

	if res := l.entries(ctx, &prog, targets, optvals); 1 == len(res) {
		return res[0]
	} else if 1 < len(res) {
		return list_t[entry](res...)
	} else {
		return _null(prog.position)
	}
}

func (l ul) entries(ctx Context, prog *program, targets, options []Value) (res []entry) {
	for _, target := range targets {
        if isTrivial(target) {
            if true { continue }
			erro(ctx, "trivial target; %v", targets).trace()
        }

        var entry = prog.project.new_entry(ctx, options, target, prog)
        if entry == nil {
            erro(ctx, "creating entry failed for %v", target).trace()
        }

		res = append(res, entry)

        if x, y := entry.destiny().(flag); y && x.Value != nil {
			if prog.project.name != "~" {
				var s = x.Value.string(ctx)
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
	if a, y := name.(*argumented); y {
		name, args = a.Value, a.args
	}

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
		erro(ctx, "unexpected end of line").trace()
	}

	pos := l.p.Position()

	l.p.expect(ctx, FOREACH)
	l.p.spaces(ctx)

	var vals = xmerge(_final(ctx), l.values(ctx)...)
	var t = &template{
		pos: l.p.pos, tok: l.p.tok, lit: l.p.lit,
		state: l.p.scanner.scanstate,
	}

	l.p.spaces(ctx)
	l.p.linend(ctx)

	if checkpoints && truly(ctx, is_test_mode{}) {
		if false && l.p.is_file("/configure/.base/.template") {
			defer func(t time.Time) {
				if d := time.Since(t); 1*time.Millisecond <= d {
					note(pc(ctx,pos), "%v, %d values", d, len(vals)).debug()
				}
			} (time.Now())
		}
	}

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

			var a = map[string]Value{}
			for i, val := range vals {
				if indeterminate(ctx, val) {
					if false {
						erro(ctx, "indeterminate: %d. %v → %v", i, tv(val), val.expand(_final(ctx))).trace()
					}
				} else if !isTrivial(val) {
					a["_"] = val
					l.codeblock(ctx, FOREACH, t, a)
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
		erro(ctx, "unexpected end-of-line").trace()
	}

	var opts struct {
		skipNil bool `skip-nil,skip-null,skipnil,skipnull,no-nil,no-null`
	}

	pos := l.p.Position()

	if l.p.expect(ctx, FOR); l.p.tok == LPAREN {
		l.p.next(ctx, true) // LPAREN
		if vals := parse_opts(ctx, &opts, l.values(ctx)...); vals != nil {
			erro(ctx, "unexpected opts: %v", vals).trace()
		}
		l.p.expect(ctx, RPAREN)
	}

	l.p.spaces(ctx)

	type param struct {
		name string
		elems []Value
	}

	type nparam struct {
		p Position
		a []*param
		n int
	}

	t0 := time.Now()

	var npars int
	var params []*nparam
	var vars = map[string]Value{}
	for l.p.spaces(ctx); l.p.tok != EOF && !l.p.is_end_of_line(); l.p.spaces(ctx) {
		if l.p.tok == AND && params == nil {
			erro(ctx, "unexpected 'and'").trace()
		} else if l.p.tok == AND || params == nil {
			if params = append(params, &nparam{p:l.p.Position()}); l.p.tok == AND {
				l.p.next(ctx, true) // and
				continue
			}
		}

		var _v = params[len(params)-1]
		for _, a := range xmerge(ctx, l.expr(ctx)) {
			var elems []Value
			var s string

			if x, y := a.(*pair); !y {
				erro(ctx, "unexpected value: %v", ts(a)).trace()
			} else if s = x.key.string(ctx); s == "" {
				erro(ctx, "empty key: %v", ts(x.key)).trace()
			} else if g, y := x.val.(*group); y {
				elems = g.elems
			} else {
				elems = append(elems, x.val)
			}

			// Make sure all elements are expanded.
			elems = xmerge(_final(ctx), elems...)

			if _, y := vars[s]; y {
				erro(ctx, "duplicated key: %v", s).trace()
			} else {
				vars[s] = &null{valbase{a.Position()}}
			}

			if n := len(elems); n > _v.n { _v.n = n }

			_v.a = append(_v.a, &param{s, elems})
			npars += len(elems)
		}
	}

	var t = &template{
		pos:l.p.pos, tok:l.p.tok, lit:l.p.lit,
		state:l.p.scanner.scanstate, // verb: "for",
	}

	l.p.spaces(ctx)
	l.p.linend(ctx)

	if checkpoints && truly(ctx, is_test_mode{}) {
		if false && l.p.is_file("/configure/.base/.template") {
			defer func(t time.Time) {
				if d := time.Since(t); 1*time.Millisecond <= d {
					note(pc(ctx,pos), "%v, %v, %d params", time.Since(t0)-d, d, npars).debug()
				}
			} (time.Now())
		}
	}

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

			state := l.p.scanner.scanstate
			t.end, t.endPos, l.p.stop = &state, pos, pos

			var num int
			for _, _v := range params {
				if _v.n > 0 {
					if num == 0 {
						num = _v.n
					} else {
						num *= _v.n
					}
				}
			}

			var e int = len(params)-1
		outer:
			for n := 0; n < num; n += 1 {
				for _i, _v := range params {
					// i[0]    = (n % 1) / b    (a = n * n * ..., k-1)
					// i[1..l] = (n % a) / b    (b = n * ..., k-2)
					// i[l+1]  = (n % a) / 1
					var i int = n

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

					for _, v := range _v.a {
						if i < len(v.elems) {
							vars[v.name] = v.elems[i]
						} else {
							vars[v.name] = &null{valbase{_v.p}}
							if opts.skipNil { continue outer }
						}
					}
				}

				var trivial bool = len(vars) == 0
				if !trivial {
					for _, v := range vars {
						if trivial = isTrivial(v); !trivial { break }
					}
				}
				if !trivial {
					l.codeblock(ctx, FOR, t, vars)
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

func (l ul) codeblock(ctx Context, op token, t *template, vars map[string]Value) {
	l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate = t.pos, t.tok, t.lit, t.state

	if false && checkpoints && truly(ctx, is_test_mode{}) {
		pprofCounter += 1
		defer cpu_profile(ctx, fmt.Sprintf("template-%05d.prof", pprofCounter), true)()
	}

	if !(l.p.pos < l.p.stop) {
		erro(ctx, "bad range: [%v %v) (%v)", l.p.pos, l.p.stop, t.name).trace()
	}

	var c = codeblock{automatic{Context:ctx, defs:make(defs_map)}, op}

	if checkpoints && truly(ctx, is_test_mode{}) {
		l.codeblock_check(ctx, op, vars)
	}

	// NOTE: defAuto is only used in this case!
	for s, v := range vars { c.set(&c, defAuto, s, v) }

	for l.p.tok != EOF && l.p.pos < l.p.stop {
		if l.p.tok == SPACE || l.p.tok == LINEND || (l.p.tok == COMMENT && l.p.lineComment != nil) {
			l.p.next(ctx, true)
		} else {
			l.clause(&c)
		}
	}
}

func (l ul) repeat(ctx Context, t *template, params []Value) {
	if false { pprofCounter += 1
		var (
			profCpu = fmt.Sprintf("template-%05d.cpu.prof", pprofCounter)
			profMem = fmt.Sprintf("template-%05d.mem.prof", pprofCounter)
			fCpu *os.File
			e error
		)
		if fCpu, e = os.Create(profCpu); e != nil {
			erro(ctx, "%v", tv(e)).trace()
		} else if e = pprof.StartCPUProfile(fCpu); e != nil {
			fCpu.Close()
			erro(ctx, "%v: %v", profCpu, e).trace()
		}
		defer func() {
			pprof.StopCPUProfile()
			fCpu.Close()

			var fMem, e = os.Create(profMem)
			if e != nil {
				erro(ctx, "%v", e).trace()
			}

			runtime.GC() // update memory statistics
			e = pprof.WriteHeapProfile(fMem)
			fMem.Close()

			if e != nil {
				erro(ctx, "%v: %v", profMem, e).trace()
			}
		} ()
	}

	defer func(t time.Time, pos Pos, tok token, lit string, state scanstate) {
		l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate = pos, tok, lit, state
	} (time.Now(), l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate)

	var m = map[string]Value{}

	for i, v := range t.params {
		if s := v.string(ctx); s != "" {
			if i < len(params) {
				m[s] = params[i]
			} else {
				m[s] = _null(v.Position())
			}
		} else {
			erro(ctx, "empty template param name: %v", tv(v)).trace()
		}
	}

	l.codeblock(ctx, LPAREN, t, m)
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

	erro(ctx, "undefined template: %v", name).trace()
	return
}

func (l ul) configure_save(ctx Context) {
	var configs = l.project.configs
	if configs == nil { return }

	if l.promptEnteringDirectory {
		l.promptEnteringDirectory = false
		t := promptLeavingDirectory(ctx, l.project.absPath)
		do(ctx, prompt_ab{nil, t})
		flush(ctx)
	}

	var f = l.project.configuration_sm(ctx)
	var fn = f.fullname()

	if checkpoints && truly(ctx, is_test_mode{}) {
		if c := l.project.configuration; c != nil && c.fullname() != fn {
			erro(ctx, "%v: %s != %s", l.project.name, c.fullname(), fn).trace()
		}
		defer l.configure_save_check(ctx, configs, f, fn)
	}

	if e := os.MkdirAll(filepath.Dir(fn), os.FileMode(0755)); e != nil {
		erro(ctx, "make path %s failed: %v", filepath.Dir(fn), e).trace()
	}

	var fm = os.O_RDWR | os.O_CREATE | os.O_TRUNC

	if o, e := os.OpenFile(fn, fm, os.FileMode(0600)); e != nil {
		erro(ctx, "%v: %v", l.project.name, e).trace()
	} else {
		defer o.Close()

        fmt.Fprintf(o, "# %s (%s)\n", l.project.name, l.project.spec) // configuration

		for _, c := range configs {
			fmt.Fprintf(o, "configure %s = %s\n", c.name, c.value)
		}

		fmt.Fprintf(o, "\n# %d configs, %s\n", len(configs), l.project)
		l.project.configuration = f // saved configuration.sm
	}
}

func (l ul) configure_set(ctx Context, name string, vals ...Value) {
	if d, a := l.project.set(ctx, name, defConfig, vals...); d == nil {
		erro(pc(ctx,l.p), "cannot configure %s (alt=%v)", name, tv(a)).trace()
	} else if a != nil {
		erro(pc(ctx,l.p), "conflict configure %s: %v", name, ts(a)).trace()
	} else {
		l.project.configs = append(l.project.configs, d)
	}
}

func promptEnteringDirectory(ctx Context, s string) (_ *diagpoint) {
	return prompt(ctx, "smake: Entering directory '%s'\n", s).diagpoint
}

func promptLeavingDirectory(ctx Context, s string) (_ *diagpoint) {
	return prompt(ctx, "smake: Leaving directory '%s'\n", s).diagpoint
}

type prompt_ab struct{ a, b *diagpoint }

func (l ul) configure_conv(ctx *execution, vt, val Value) (res Value) {
	var args []Value
	var op Value = vt
	if x, y := vt.(*argumented); y {
		if f, y := x.Value.(flag); y {
			op = f.Value
		} else {
			erro(pc(ctx,x.Value), "wrong conv word: %v %v", tv(x.Value), tv(val)).trace()
		}
		if false { vt = x.Value }
		args = xmerge(_final(ctx), x.args...)
	}

	if x, y := op.(*word); !y {
		erro(pc(ctx,vt), "wrong conv word: %v %v", tv(vt), tv(val)).trace()
	} else {
		switch x.s {
		case "answer":
			return _answer(val.Position(), val.true(ctx))
		case "bool", "boolean":
			return _boolean(val.Position(), val.true(ctx))
		case "value":
			return val.expand(ctx)
		}

		if l.project.configure == nil {
			erro(pc(ctx,vt), "wrong conv: %v %v", tv(vt), tv(val)).trace()
		}

		var _params = make(map[string]Value)
		for _, arg := range args {
			switch t := arg.(type) {
			case *pair:
				_params[t.key.string(ctx)] = t
			case *raw, *strlit, *strval, *strcomp:
				_params["INFO"] = &pair{_word(t.Position(),"INFO"), t}

				if !l.promptEnteringDirectory {
					l.promptEnteringDirectory = true
					t := promptEnteringDirectory(ctx, l.project.absPath)
					do(ctx, prompt_ab{t, nil})
				}

				s := t.string(ctx)
				a := prompt(pc(ctx,vt), "%s …", s)
				defer func(i int) {
					if count_diag(ctx, diagInfo, diagWarn, diagError) <= i {
						if res != nil {
							s = res.string(ctx)
						} else {
							s = "<nil>"
						}
						b := prompt(ctx, "… %s\n", s)
						do(ctx, prompt_ab{a.diagpoint, b.diagpoint})
						flush(ctx)
					}
				} (count_diag(ctx, diagInfo, diagWarn, diagError))

			default:
				if !isTrivial(arg) {
					erro(pc(ctx,arg), "wrong arg: %s", tv(arg)).trace()
				}
			}
		}

		var vals []Value
		for _, ent := range l.project.configure._entries(ctx, vt, false) {
			var params []Value
			for _, prog := range ent.programs() {
				for _, p := range prog.params {
					pos := p.Position()
					w := _word(pos, p.ident(ctx))
					if x, y := _params[w.s]; y {
						params = append(params, x)
					} else {
						switch /* strings.ToUpper */(w.s) {
						case "TARGET": params = append(params, &pair{w, auto_get(ctx, "@")})
						case "VALUE":  params = append(params, &pair{w, val})
						case "LANG", "LANGUAGE":
							if ctx.language != "" {
								params = append(params, &pair{w, _strlit(pos, ctx.language)})
							}
						}
					}
				}
			}
			vals = append(vals, ent.execute(ctx, params...)...)
		}

		return ease(ctx, vals)
	}
	return
}

func (l ul) configure_one(ctx Context) {
	var nv = l.expr(ctx)
	l.p.spaces(ctx)

	var skip bool
	var name string
	switch x := nv.(type) {
	case *word: name = x.s
	case *compound:
		for _, v := range x.elems {
			if w, y := v.(*word); y {
				name += w.s
			} else {
				errostack(pc(ctx,nv), 16, "unsupported configure: %v : %v", ts(nv), ts(v)).trace()
			}
		}
	default:
		errostack(pc(ctx,nv), 16, "unsupported configure: %v", ts(nv)).trace()
	}
	if name == "" {
		errostack(pc(ctx,nv), 16, "empty configure: %v", ts(nv)).trace()
	}

	if d := l.project.def(ctx, name); d != nil && d.o == defConfig {
		skip = true
	}

	if checkpoints && truly(ctx, is_test_mode{}) {
		defer l.configure_check(ctx, name)
	}

	var vt Value
	var conds []Value
	for l.p.tok == MINUS {
		t := l.expr(ctx)
		l.p.spaces(ctx)

		if a, y := t.(*argumented); y {
			if x, y := a.Value.(flag); y {
				switch x.Value.String() {
				case "cond":
					conds = append(conds, a.args...)
					continue
				}
			}
		} else if x, y := t.(flag); y && x.Value.String() == "cond" {
			erro(pc(ctx,x.Value), "needs cond value").trace()
		}

		vt = t
		break
	}

	if l.p.tok == ASSIGN {
		l.p.next(ctx, true) // skips the '=' token

		var vals []Value
		if l.p.tok != LINEND && l.p.lineComment == nil {
			vals = append(vals, l.expr(ctx))
		}

		if !skip {
			l.configure_set(ctx, name, vals...)
		}
		return
	} else if l.p.tok.is_assign() {
		erro(pc(ctx,l.p), "%v: only '=' can assign a configure", nv).trace()
	}

	var deps, recipes []Value
	if l.p.tok == COLON {
		l.p.next(ctx, true) // skips the ':' token

		defer func() { l.p.dialect = "" } ()

		for l.p.tok != EOF {
			deps = append(deps, l.expr(ctx))
			l.p.spaces(ctx)

			switch l.p.tok {
			case SEMICOLON, LINEND:
				goto dorecipes
			}
		}
		if l.p.dialect != "" {
			note(pc(ctx,l.p), "configure %v").debug()
		}
	}

dorecipes:
	if l.p.tok == SEMICOLON { // ;
		recipes = append(recipes, l.recipe(ctx)...)
	} else /*if p.tok == LINEND || p.lineComment != nil*/ {
		l.p.scanner.recipes(true) // turn on recipes before LINEND.
		if l.p.linend(ctx) { // Take the new line.
			for l.p.recipe_start() {
				recipes = append(recipes, l.recipe(ctx)...)
			}
		}
		l.p.scanner.recipes(false)
	}

	if skip { return }

	var exe = execution{
		automatic:automatic{Context:pc(ctx,nv), defs:make(defs_map)},
		recs:make(map[Value]int), start:time.Now(), proj:l.project,
		recipes:recipes, language:l.p.dialect,
	}

	defer func() {
		for _, a := range exe.defers {
			if x, y := a.(*group); y {
				modify(ctx, x, true)
			} else {
				erro(pc(ctx,a), "defer: not a modifier: %s", ts(a)).trace()
			}
		}
	} ()

    exe.set(ctx, defVoid, "@", nv)

    for _, exe.prerequisite = range deps { exe.prerequisite.traverse(&exe) }

	var res = auto_get(&exe, "-")
	if res == nil && recipes != nil && len(exe.interpreted) == 0 {
		if x, y := dialects[""]; y && x != nil {
			res = exe.interpret(&exe, x, nil)
		}
	}

	if vt != nil {
		res = l.configure_conv(&exe, vt, res)
	}

	var vals []Value
	if res != nil { vals = append(vals, res) }

	l.configure_set(ctx, name, vals...)
}

func (l ul) configure(ctx Context) {
	l.p.expect(ctx, CONFIGURE)
	l.p.spaces(ctx)
	if l.p.tok == LPAREN {
		l.p.next(ctx, true)
		l.p.linend(ctx)
		l.p.spaces(ctx)
		for l.p.tok != EOF {
			l.configure_one(ctx)
			l.p.spaces(ctx)
			if l.p.tok == RPAREN {
				l.p.next(ctx, true)
				break
			}
		}
	} else {
		l.configure_one(ctx)
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
	case   INCLUDE: l.spec(ctx, t, l.p.expect(ctx, t), l.include)   ; return
	case      EVAL: l.spec(ctx, t, l.p.expect(ctx, t), l.eval)      ; return
	case    ASSERT: l.spec(ctx, t, l.p.expect(ctx, t), l.p.assert)  ; return
	case    APPEND: l.spec(ctx, t, l.p.expect(ctx, t), l.p.append)  ; return
	case     FILES: l.spec(ctx, t, l.p.expect(ctx, t), l.files)     ; return
	case     LOCAL: l.spec(ctx, t, l.p.expect(ctx, t), l.local)     ; return
	case       DEF: l.def_end(ctx)     ; return
	case       FOR: l.for_done(ctx)    ; return
	case   FOREACH: l.foreach_done(ctx); return
	case CONFIGURE: l.configure(ctx)   ; return
	case USE, TEMPLATE:
		erro(ctx, "unexpected %v", t).trace()
	}

	var vals []Value

	for l.p.tok != LINEND && l.p.tok != EOF {
		var x = l.expr(p_left_ctx{ctx})

		l.p.spaces(ctx)

		vals = append(vals, x)

		if l.p.tok.is_assign() {
			l.assign(ctx, vals)
			return
		}

		if l.p.tok.is_rule_delim() {
			l.rule(ctx, nil, vals)
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
func (l ul) new_declare(ctx Context, name, filename string, opts *project_opts) (d *declare) {
	if x, y := l.declares[name]; y { return x }

    var sco = l.scope()
    var relPath = sco.finddef(".").string(ctx) // CRD
    var tmpPath = sco.finddef(",").string(ctx) // CTD

    var absPath string
	if x, y := do(ctx, abs_path{}).(string); y {
		absPath = x
	} else {
		absPath = sco.finddef("/").string(ctx)
	}

    var spec, _ = filepath.Rel(workBaseDir, absPath)
    if x, y := l.globe.loaded[absPath]; y {
        prompt(ctx, "%s: %v : already declared : %s\n", absPath, x, filename)
        errostack(ctx, 5, "%s %s %s : %v", name, relPath, spec, l.project).trace()
    }

    if l.declares == nil { l.declares = make(map[string]*declare) }

	d = &declare{
		project: &project{
			position: l.p.Position(),//_position(ctx),
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
    d.scope = newscope(d.position, sco, d.project, /* "project "+ */name)
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

type project_opts struct {
	configure Value `conf,config,configure` // detects dot_configure if empty
	traveUseLoop bool `break,loop` // don't recursively use this project
	multiUseAllowed bool `multi`  // this project is used multiple times
	final bool `final` // no bases
}

func (l ul) declare(ctx Context, ident Value, name, filename string, declOpts *project_opts) (_ bool) {
	if name == "@" {
		erro(ctx, "deprecated project name: @").trace()
	}

    if _, o := l.find(name); o != nil {
        if x, y := o.(*builtin); y {
            erro(ctx, "%v is a builtin name", x).trace()
        }
    }

	var prev = l.loader // nil if newly declared
	var dec = l.new_declare(ctx, name, filename, declOpts)
	if prev == nil || dec.project != prev.project {
		l.project, l.loader.scope = dec.project, dec.scope
	}

    if ll := _loader(l.loader.Context); ll != l.loader && ll == prev {
        if _, a := ll.project.projectname(ctx, name, dec.project); a != nil {
            if x, y := a.(*project); !y || x != dec.project {
                erro(ctx, "%v: name already taken : %v", name, ts(a)).trace()
            }
        }
    }

    if l.globe.main != nil && l.globe.main == l.project && l.project.name != "~" {
        for _, t := range l.globe.pairs {
            switch k := t.key.(type) {
            case *word, *compound:
                l.scope().set(ctx, k, defDecl, t.val)
            case flag:
                if false { warn(ctx, "unknown flag : %v", t).debug() }
            default:
                warn(ctx, "unknown target : %v", ts(t)).debug()
            }
        }
    }

	if x := try[[]Value](ctx, get_args{}); len(x) != 0 {
		for _, arg := range merge(x...) {
			switch t := arg.(type) {
			case *pair:
				switch k := t.key.(type) {
				case *word, *compound:
					l.scope().set(ctx, k, defDecl, t.val)
				case flag:
					if false { warn(ctx, "unknown flag : %v", t).debug() }
				default:
					warn(ctx, "unknown target : %v", ts(t)).debug()
				}
			}
		}
	}

    if err := l.loadPlugin(ctx); err != nil {
        erro(ctx, "load plugin failed: %v", err).trace()
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
			if d, _ := s.set(ctx, t.key.string(ctx), defVoid, t.val); d != nil {}
		default:
			erro(ctx, "unknown set: %v", ts(a)).trace()
		}
	}
}

type declared_project struct { *project }

type parent struct { Context ; *project }
func (p parent) cast(t reflect.Type) Context { return icast(p,t) }
func (p parent) inner() Context { return p.Context }
func (p parent) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case declared_project:
		if t.has_base(p.project) {
			if true { return }

			prompt(ctx, "%s: %s : %s\n", p.absPath, p.project, t.loop_base_path(ctx, p.project, ""))
			if true {
				notestack(ctx, 10, "recursive derivation: %v ⇔ %v", ts(p.project), ts(t.project)).debug(5)
				return
			} else {
				errostack(ctx, 10, "recursive derivation: %v ⇔ %v", ts(p.project), ts(t.project)).trace()
			}
		}

		if p.has_base(p.project) {
			errostack(ctx, 10, "duplication derivation: %v ⇔ %v", ts(p.project), ts(t.project)).trace()
		}

		if len(p.bases) == 0 {
			p.projectname(ctx, ".base", t.project)
		}

		p.bases = append(p.bases, t.project)
		return
	}
	return p.Context.do(ctx, op)
}

func (l ul) projec(ctx Context, filename string, isMainFile bool) (_ Value, _ string, _ bool) {
	var implicitBase string // aka. foo.bar.Baz implicitly load base 'foo/bar'

	l.p.next(ctx, true) // aka. the keyword

	var vals []Value
	for l.p.tok == MINUS {
		val := l.expr(ctx); l.p.spaces(ctx)

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
	if a := parse_opts(ctx, &opts, vals...); len(a) > 0 {
		erro(ctx, "unknown project option %v", ts(a)).trace()
	}

	var ident Value

	// Smart-lang spec:
	//   * the project clause is not a declaration;
	//   * the project name does not appear in any scope.
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
			erro(ctx, "invalid file: %v", filename).trace()
		}
	} else if l.p.tok == TILDE {
		if ext := filepath.Ext(filename); ext != ".smart" {
			erro(ctx, "`%v` not a smart file", filepath.Base(filename)).trace()
		} else if s := strings.TrimSuffix(filepath.Base(filename), ext); s == "" {
			erro(ctx, "`%v` not tilde name", filepath.Base(filename)).trace()
		} else {
			ident = _word(l.p.Position(), s)
		}
		l.p.next(ctx, true) // skip tilde
	} else {
		base, comp := makePath(), _compound()

		for l.p.tok != EOF && l.p.tok != SPACE {
			var w = l.p.bare(ctx)
			if  comp = comp.suffix(ctx, w).(*compound) ; l.p.tok == DOT {
				comp = comp.suffix(ctx, l.punct(ctx)).(*compound)
				base.elems = append(base.elems, w)
			} else {
				break
			}
		}

		l.p.spaces(ctx)

		if comp.len() == 0 {
			// erro(ctx, "package name is empty (tok=%v %v)", t, p.tok).trace()
		} else if 0 < base.len() {
			implicitBase = base.string(ctx)
		}

		ident = comp
	}

	var name = ident.string(ctx)

	if p := l.project; p != nil && p.name != name {
		errostack(ctx, 5, "%v: multiple projects in the directory : %v", p, ident).trace()
	}

	if name == "-" || name == "_" {
		erro(ctx, "package name '%s' is preserved", name).trace()
	}

	var _, prevDeclared = l.declares[name]

	if l.declare(ctx, ident, name, filename, &opts) {
		if l.project == nil {
			erro(ctx, "undeclared project: %v", ident).trace()
		}
		isMainFile = isMainFile && !prevDeclared;
	}

	var cc = parent{ctx, l.project}

	if l.p.tok != LPAREN {
		l.bases(cc, implicitBase) // for special bases, e.g. .base
	} else {
		var cc0 = p_group_ctx{p_aware_comma{ctx}}
		for l.p.tok != EOF {
			for l.p.next(ctx, true); !l.p.is_list_term(ctx); {
				l.p.spaces(ctx)

				v := parse_opts(ctx, &opts, l.expr(cc0))

				if opts.final {
					if v != nil {
						erro(ctx, "no bases for final project: %v", ts(v)).trace()
					}
					continue
				}

				l.bases(cc, "", merge(v...)...)
			}
			if l.p.tok != COMMA { break }
		}
		l.p.expect(ctx, RPAREN)
	}

	if checkpoints && truly(ctx, is_test_mode{}) {
		l.project_check_bases(ctx)
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
		erro(ctx, "undeclared project: %v", name).trace()
	}

    if l.project == nil {
        erro(ctx, "current project unset").trace()
    }

    if l.project.name != name {
        erro(ctx, "current project is %s, not %s", l.project, name).trace()
    }

    if l.project != x.project {
        erro(ctx, "project conflicts (%v, %v)", l.project, x.project).trace()
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


	var rel,_ = filepath.Rel(l.workdir, abs)
	var tmp   = joinTmpPath(ctx, l.workdir, rel)

	if s := l.scope(); /* p == nil || */ s == nil {
		erro(ctx, "%v: nil scope: %v", l.project, s).trace()
	}

	defer l.closescope(l.openscope(bases(2, filename, true)))

	if checkpoints {
		if s := l.p.scanner.file.Name(); filename != s {
			erro(ctx, "%v: %s != %s", l.project, filename, s).trace()
		}
		if truly(ctx, is_test_mode{}) {
			defer l.parse_file_check_1(ctx, abs, rel, tmp)
		}
	}

	if !flatmode {
		// CWD: Current Work Directory,     TODO: use $:cwd:
		// CTD: Current Temp Directory,     TODO: use $:ctd:
		// CRD: Current Relative Directory, TODO: use $:crd:
		var s = l.scope()
		if d,_ := s.set(ctx, "/", defVoid, _pathstr(ctx, abs)); d != nil { s.alias(ctx, d, "CWD") }
		if d,_ := s.set(ctx, ".", defVoid, _pathstr(ctx, rel)); d != nil { s.alias(ctx, d, "CRD") }
		if d,_ := s.set(ctx, ",", defVoid, _pathstr(ctx, tmp)); d != nil { s.alias(ctx, d, "CTD") }
	}

	switch keyword {
	case PROJECT:
		if flatmode {
			erro(pc(ctx,l.p), "project is forbidden in flat file").trace()
		}

		var name string
		var prev = l.project

		_, name, isMainFile = l.projec(ctx, filename, isMainFile)
		if prev != l.project { defer l.close_project(ctx, name) }
		if checkpoints { l.parse_file_check_new_project(ctx) }

	case EOF:
		return

	default:
		if !flatmode {
			l.p.expected(ctx, l.p.pos, "configure, project, module or package keyword")
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

	if checkpoints && truly(ctx, is_test_mode{}) {
		l.parse_file_check_2(ctx, filename)
	}

	return l.mode&ImportsOnly != 0 || l.p.tok == EOF
}
