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
	"reflect"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const maxDigitAutoNum = 9

type usespec struct {
	props []Value
}

type parsedFile struct {
	// TODO: doc *commentGroup
	// TODO: comments *commentGroup
	keyword  token // project, package or module
	position Position // position of the beginning, which has filename information
	name  *barecomp // project/module name
	scope *scope
	use []*usespec // imports
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
	Context

	scanner scanner

	// Comments
	comments  []*commentGroup
	leadComment *commentGroup // last lead comment
	lineComment *commentGroup // last line comment

	// Next token
	pos, stop Pos // parsing and stop position
	tok token  // one token look-ahead
	lit string // token literal

	templates []*template

	imports []*usespec // list of imports

	targets []Value // targets of current rule
	ruparas []*auto // parameters of current rule
	dialect  string // recipe dialect of current rule
	configure  bool // is parsing configure program?

	dd bool // helps debug parsing via `eval -dd=true{}`
}

func (p *parser) cast(t reflect.Type) Context { return implcast(p,t) }
func (p *parser) do(ctx Context, op any) (res any) {
	switch op.(type) {
	case getParseAware: // return p.bits&(???) != 0
	}
	return p.Context.do(ctx, op)
}

type (
	getParseAware      struct{ token }
	getParseCanParams  struct{}
	getParseCanUndef   struct{}
	getParseGlob       struct{}
	getParseIncOpts    struct{}
	getParseIsAuto     struct{ string }
	getParseIsConf     struct{}
	getParseIsRecipe   struct{ bool } // builtin or text
	getParseLeftHandSide struct{}
)

type token_aware_context struct { Context ; token }
func (p token_aware_context) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case getParseAware: return p.token == t.token
	}
	return p.Context.do(ctx, op)
}

type parser_auto_context     struct { Context }
type parser_bare_context     struct { Context }
type parser_braced_context   struct { Context }
type parser_call_context     struct { Context }
type parser_code_context     struct { automatic }
type parser_defvalue_context struct { Context }
type parser_foreach_context  struct { Context }
type parser_glob_context     struct { Context }
type parser_group_context    struct { Context }
type parser_include_context  struct { Context ; o includeOpts }
type parser_left_context     struct { Context }
type parser_modifier_context struct { Context }
type parser_params_context   struct { Context }
type parser_path_context     struct { Context }
type parser_perc_context     struct { Context }
type parser_recipe_context   struct { Context ; builtin bool }
type parser_regex_context    struct { Context }
type parser_rule_context     struct { Context }
type parser_undef_context    struct { Context }

func (p parser_glob_context) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case getParseGlob: return true
	}
	return p.Context.do(ctx, op)
}

func (p parser_params_context) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case getParseCanParams: return true
	}
	return p.Context.do(ctx, op)
}

func (p parser_auto_context) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case getParseIsAuto: return true
	}
	return p.Context.do(ctx, op)
}

func (p parser_defvalue_context) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case getParseIsAuto: return IsDigits(t.string)
	}
	return p.Context.do(ctx, op)
}

func (p parser_foreach_context) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case getParseIsAuto: if t.string == "_" { return true }
	}
	return p.Context.do(ctx, op)
}

func (p parser_rule_context) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case getParseIsAuto:
		if IsDigits(t.string) { return true }
		if _, y := rule_autos[t.string]; y { return true }
	}
	return p.Context.do(ctx, op)
}

func (p parser_recipe_context) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case getParseIsRecipe: return t.bool == p.builtin
	}
	return p.Context.do(ctx, op)
}

func (p parser_include_context) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case getParseIncOpts: return &p.o
	case getParseIsConf : return p.o.isConfig
	}
	return p.Context.do(ctx, op)
}

func (p parser_left_context) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case getParseLeftHandSide: return true
	}
	return p.Context.do(ctx, op)
}

func (p parser_undef_context) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case getParseCanUndef: return true
	}
	return p.Context.do(ctx, op)
}

// ----------------------------------------------------------------------------
// Parsing support

func (p *parser) trace(a ...any) { l_traverse.traceAt(p.Position(), a...) }

// Advance to the next token.
func (p *parser) scan() {
	// Because of one-token look-ahead, print the previous token
	// when tracing as it provides a more readable output. The
	// very first token (!p.pos.IsValid()) is not initialized
	// (it is ILLEGAL), so don't print it .
	if l_traverse.enabled && p.pos.IsValid() {
		s := p.tok.String()
		switch {
		case p.tok.isLiteral():
			p.trace(s, p.lit)
		case p.tok.isOperator(), p.tok.isKeyword():
			p.trace("\"" + s + "\"")
		default:
			p.trace(s)
		}
	}

	var pos = p.pos
	p.pos, p.tok, p.lit = p.scanner.scan()
	if false && p.lit == "none" { warn(p, "%v %v", p.tok, p.lit).debug(64); flush(p.Context) }
	if false && p.tok == EOF {
		erro(at(p,p.loc(pos)), "unexpected end of file").debug()
	}
}

// Consume a comment and return it and the line on which it ends.
func (p *parser) consumeComment() (res *comment, endline int) {
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
	p.scan()

	return
}

// Consume a group of adjacent comments, add it to the parser's
// comments list, and return it together with the line at which
// the last comment in the group ends. A non-comment token or n
// empty lines terminate a comment group.
//
func (p *parser) consumecommentGroup(n int) (res *commentGroup, endline int) {
	res = new(commentGroup)
	p.comments = append(p.comments, res)
	endline = p.scanner.file.Line(p.pos)
	for p.tok == COMMENT && p.scanner.file.Line(p.pos) <= endline+n {
		var com *comment
		com, endline = p.consumeComment()
		res.list = append(res.list, com)
	}
	return
}

// Advance to the next non-comment token. In the process, collect
// any comment groups encountered, and remember the last lead and
// and line comments.
//
// A lead comment is a comment group that starts and ends in a
// line without any other tokens and that is followed by a non-comment
// token on the line immediately after the comment group.
//
// A line comment is a comment group that follows a non-comment
// token on the same line, and that has no tokens after it on the line
// where it ends.
//
// Lead and line comments may be considered documentation that is
// stored in the AST.
//
func (p *parser) step() {
	p.leadComment = nil
	p.lineComment = nil

	var prev = p.pos
	if p.scan(); p.tok == COMMENT {
		var comment *commentGroup
		var endline int

		// If the comment is on same line as the previous token; it
		// cannot be a lead comment but may be a line comment.
		if p.scanner.file.Line(p.pos) == p.scanner.file.Line(prev) {
			comment, endline = p.consumecommentGroup(0)
			if p.scanner.file.Line(p.pos) != endline {
				// The next token is on a different line, thus
				// the last comment group is a line comment.
				p.lineComment = comment
			}
		}

		// consume successor comments, if any
		endline = -1
		for p.tok == COMMENT {
			comment, endline = p.consumecommentGroup(1)
		}

		if endline+1 == p.scanner.file.Line(p.pos) {
			// The next token is following on the line immediately after the
			// comment group, thus the last comment group is a lead comment.
			p.leadComment = comment
		}
	}

	// if p.tok != LINEND && p.lineComment != nil { p.tok = LINEND }

	if p.dd {
		var t = warn(p, "%v %v %v", p.tok, p.lit, p.scanner.scanstate)
		if p.tok == COMPOUND { t.debug(12) }
		if p.tok == LINEND { t.debug(24) }
		flush(p.Context)
	}
}
func (p *parser) next(ctx Context, ws bool) { if p.step(); ws { p.spaces(ctx) } }
func (p *parser) spaces(ctx Context) {
	for p.lineComment == nil && p.tok != EOF {
		if p.tok == SPACE || (p.tok == RECIPE && can(ctx, getParseIsRecipe{true})) {
			p.step()
		} else if p.tok == ESCAPE && p.lit == "\n" {
			if p.step(); p.tok == LINEND || p.lineComment != nil { break }
			if can(ctx, getParseIsRecipe{true}) {
			TokFor:
				for p.tok != EOF {
					switch p.tok {
					case RECIPE: // TODO: using p.isRecipeStart()
						if true { p.scanner.pop(isCompoundLine) }
						p.step()
					default: break TokFor
					}
				}
			}
		} else {
			break
		}
	}
}

func (p *parser) Position() Position { return p.loc(p.pos) }
func (p *parser) loc(pos Pos) Position { return Position(p.scanner.file.Position(pos)) }
func (p *parser) valbase() valbase { return valbase{p.loc(p.pos)} }

// A bailout panic is raised to indicate early termination.
type bailout struct{}

func (p *parser) expected(pos Pos, msg string, a... any) {
	if len(a) > 0 { msg = fmt.Sprintf(msg, a...) }
	if msg = "expected " + msg; pos == p.pos {
		// the error happened at the current position;
		// make the error message more specific
		if p.tok == SEMICOLON && p.lit == "\n" {
			msg += ", found newline"
		} else {
			msg += ", found " + p.tok.String()
			if p.tok.isLiteral() {
				msg += " '" + p.lit + "'"
			}
		}
	}
	erro(at(p,p.loc(pos)), msg).debug(32)
}

func (p *parser) expect(tok token) Pos {
	var pos = p.pos
	if p.tok != tok && !(tok == LINEND && p.lineComment != nil) {
		p.expected(pos, "'"+tok.String()+"'")
	}
	p.step() // move forward
	return pos
}

func (p *parser) linend() (ok bool) {
	if p.lineComment != nil {
		p.lineComment, ok = nil, true
	} else if p.tok == EOF {
		ok = true
	} else if p.tok == LINEND {
		p.step(); ok = true
	} else {
		p.expected(p.pos, "'\\n'")
	}
	return
}

func (p *parser) isRecipeStart() (res bool) {
	if p.tok == RECIPE {
		res = true
	} else if p.tok == SPACE && p.lit == "\t" {
		p.tok, res = RECIPE, true // Fixes recipe \t
	}
	return
}

// ----------------------------------------------------------------------------
// Parsing

// safePos returns a valid file position for a given position: If pos
// is valid to begin with, safePos returns pos. If pos is out-of-range,
// safePos returns the EOF position.
//
// This is hack to work around "artificial" end positions in the AST which
// are computed by adding 1 to (presumably valid) token positions. If the
// token positions are invalid due to parse errors, the resulting end position
// may be past the file's EOF position, which would lead to panics if used
// later on.
//
/*
func (p *parser) _safePos(pos Pos) (res Pos) {
	defer func() {
		if recover() != nil {
			res = Pos(p.scanner.file.Base() + p.scanner.file.Size()) // EOF position
		}
	}()
	_ = p.scanner.file.Offset(pos) // trigger a panic if position is out-of-range
	return pos
}
*/

// ----------------------------------------------------------------------------
// Barewords & Identifiers

func (p *parser) bare(ctx Context) (x Value) {
	if false { defer trace(ctx) }

	var tok, lit, pos = p.tok, p.lit, ctx.Position()
	p.step()

	if tok != BAREWORD && lit == "" {
		lit = tok.String()
	}
	return makeBareword(pos, lit)
}

func (p *parser) braced(ctx Context) (x Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "braced")) }

	var pos = p.Position()

	ctx = parser_braced_context{at(ctx, pos)}

	defer trace(ctx)

	p.expect(LBRACE)

	if p.tok == RBRACE {
		x = &null{p.valbase()}
		p.spaces(ctx)
		p.step() // consumes }
		return
	}

	if checkpoints {
		if p.tok != LPAREN && !p.scanner.bits.isBrace() {
			erro(at(ctx, p), "wrong scan state: %v, %v, %v", p.tok, p.lit, p.scanner.scanstate).debug(3)
		}
	}

	var typed token

	if p.tok == LBRACK { // OBSOLETE: {[...]}
		erro(at(ctx, p), "syntax error; for modification, use {(modifier)}").debug(3)
		return
	} else if p.tok == LPAREN {
		x = p.modification(ctx)
		p.spaces(ctx)
		p.expect(RBRACE)
		return
	} else if p.tok == ASSIGN { // =
		p.step() // skips =
		if p.tok == RBRACE {
			typed = NULL
		} else {
			switch typed = p.tok ; typed {
			case /* TODO: GLOB, */ REGEX:
				p.step() // skips the type name
				p.scanner.addBits(isBraceRaw)

				// Trim leading spaces differently to avoid messing the scan states.
				// NOTE: the first SPACE and BAREWORD do not become RAW.
				for p.tok == SPACE || (p.tok == RAW && p.lit == " ") { p.step() }
				if false { switch p.tok { case BAREWORD: p.tok = RAW }}
				if false { note(p, "%v %v %v", p.tok, p.lit, p.scanner.scanstate) }

			default:
				p.next(ctx, true)
			}
		}
		if p.tok == RBRACE {
			switch p.step(); typed {
			case BIN:   x = makeBinary(pos, 0)
			case OCT:   x = makeOctal(pos, 0)
			case INT:   x = makeDecimal(pos, 0)
			case HEX:   x = makeHexadecimal(pos, 0)
			case FLOAT: x = makeFloat(pos, 0.)
			case TRUE:  x = makeBoolean(pos, true)
			case FALSE: x = makeBoolean(pos, false)
			case YES:   x = makeAnswer(pos, true)
			case NO:    x = makeAnswer(pos, false)
			case ON:    x = makeOption(pos, true)
			case OFF:   x = makeOption(pos, false)
			case NONE:  x = makeNone(pos)
			case NULL:  x = makeNull(pos)
			default: erro(ctx, "expects braced value (%v)", typed).debug()
			}
			return
		}
	}

	switch typed {
	case BARE: // {=bare ... }
		x = p.bare(at(ctx, p))
		p.spaces(ctx)
		p.expect(RBRACE)
		return
	case GLOB: // {=glob ... }
		x = p.glob(at(ctx, p), nil)
		p.spaces(ctx)
		p.expect(RBRACE)
		return
	case REGEX: // {=regex ...}
		return p.regex(at(ctx, p))
	case FILE: // {=file ... }
		if v := p.expr(ctx); v != nil {
			var c = at(ctx, v)
			var s = v.string(c)
			var a = []any{stat_nonexist{true}}
			if !isAbsOrRel(s) { a = append(a, stat_dir{get_project(ctx).absPath}) }
			x = stat(c, s, a...)
		}
		p.spaces(ctx)
		p.expect(RBRACE)
		return
	case PATH: // {=path ... }
		if v := p.expr(ctx); v != nil {
			if t, y := v.(*path); !y {
				x = p.path(ctx, v)
			} else {
				x = t
			}
		}
		p.spaces(ctx)
		p.expect(RBRACE)
		return
	case BIN, OCT, INT, HEX, FLOAT: // ={bin ...}, {=oct ...}, {=int ...}, {=hex ...}, {=float ...}
		if v := p.expr(ctx); v == nil {
			erro(ctx, "%s expects: %v, not %v %v", typed, RBRACE, p.tok, p.lit).debug()
		} else if p.spaces(ctx); p.tok == RBRACE {
			if p.step(); typed == FLOAT {
				var n, _ = v.float(ctx)
				return makeFloat(pos, n)
			}
			switch n, _ := v.int(ctx); typed {
			case   BIN: return makeBinary(pos, n)
			case   OCT: return makeOctal(pos, n)
			case   INT: return makeDecimal(pos, n)
			case   HEX: return makeHexadecimal(pos, n)
			case FLOAT:
			}
		}
		return
	case ANSWER: // {=answer ...}
		var v bool
		switch p.tok {
		case  TRUE, YES: v = true  ; p.next(ctx, true)
		case FALSE,  NO: v = false ; p.next(ctx, true)
		default:
			if t := p.expr(ctx); t != nil {
				v = t.true(ctx)
			} else {
				erro(at(ctx, p), "invalid expression").debug()
			}
		}
		p.spaces(ctx)
		p.expect(RBRACE)
		return &answer{boolean{valbase{pos},v}}
	case BOOL, BOOLEAN: // {=bool ...}, {=boolean ...}
		var v bool
		switch p.tok {
		case  TRUE, YES,  ON: v = true  ; p.next(ctx, true)
		case FALSE,  NO, OFF: v = false ; p.next(ctx, true)
		default:
			if t := p.expr(ctx); t != nil {
				v = t.true(ctx)
			} else {
				erro(at(ctx, p), "invalid expression").debug()
			}
		}
		p.spaces(ctx)
		p.expect(RBRACE)
		return &boolean{valbase{pos},v}
	case TRUE, FALSE: // {=true ...}, {=false ...}
		var v = p.expr(ctx).true(ctx)
		p.spaces(ctx)
		p.expect(RBRACE)
		return &boolean{valbase{pos},(typed == TRUE && v)}
	case YES, NO: // {=yes ...}, {=no ...}
		var v = p.expr(ctx).true(ctx)
		p.spaces(ctx)
		p.expect(RBRACE)
		return &answer{boolean{valbase{pos},(typed == YES && v)}}
	case ON, OFF: // {=on ...}, {=off ...}
		var v = p.expr(ctx).true(ctx)
		p.spaces(ctx)
		p.expect(RBRACE)
		return &option{boolean{valbase{pos},(typed == ON && v)}}
	case RAW:
		s := p.expr(ctx).string(ctx)
		p.spaces(ctx)
		p.expect(RBRACE)
		return &raw{valbase{pos},s}
	case UNDEF: // {=undef ...}
		x = undef{p.expr(ctx)}
		p.spaces(ctx)
		p.expect(RBRACE)
		return
	case NONE: // {=none ...}
		var v Value
		for ; p.tok != RBRACE && p.tok != EOF; p.spaces(ctx) {
			if t := p.expr(ctx); v == nil {
				v = t
			} else if l, y := v.(*list); y {
				l.elems = append(l.elems, t)
			} else {
				v = &list{elements{[]Value{v,t}}}
			}
		}
		p.expect(RBRACE)
		return &none{valbase{pos},v}
	case /* DISJUNCTION, */ 0: // {...}
		if v := p.values(ctx); len(v) == 0 {
			x = makeNull(pos)
		} else if len(v) == 1 {
			x = disjunction{v[0]}
		} else {
			x = disjunction{makeList(v...)}
		}
		p.spaces(ctx)
		p.expect(RBRACE)
		return
	default:
		erro(ctx, "%v", typed).debug()
		return
	}
}

func (p *parser) selector(ctx Context) (res Value) {
	res = p.expr(ctx)
	return
}

func (p *parser) selectExpr(ctx Context, lhs Value) (res Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Select")) }

	var tok = p.tok // the arrow '->' or '=>'
	var l = _loader(ctx)

	ctx = at(ctx, p)
	p.step() // skip '->' or '=>'

	switch t := lhs.(type) {
	case *selection:
		if v := t.expand(at(ctx, t.Position())); isNull(v) {
			erro(ctx, "nil selection: %v", lhs).debug()
			return
		} else {
			lhs = v
		}
	case *bareword:
        switch t.s {
        case "use", "usee", "goals", "os", "mode":
			erro(ctx, "$:%s: is obsoleted, use $(.$s) instead", t.s, t.s).debug()
        default:
            if o := p.resolve(ctx, t, t.s); false {
				erro(at(ctx,lhs.Position()), "resolve '%v' failed", lhs)
				erro(ctx, "parser is here (tok=%s)", tok)
				erro(at(ctx,p.Position()), "parser to go here (tok=%s, lit=%s)", p.tok, p.lit).debug(8)
                return
            } else if !isNull(o) {
				lhs = o
			} else if tok == SELECT_PROG2 {
				res = makeNull(ctx.Position()) // ignore
				return
			} else {
				erro(at(ctx,lhs.Position()), "%v: '%v' is undefined (name=%v, obj=%v)", l.project, lhs, t, o)
				erro(ctx, "%v: parser is here (name=%s, tok=%s)", l.project, t.s, tok)
				erro(at(ctx,p.Position()), "%v: parser to go here (tok=%s, lit=%s)", l.project, p.tok, p.lit).debug(16)
				return
            }
        }
    case *barecomp: // for cases like '.foo'
		name := lhs.string(ctx)
        if o := p.resolve(ctx, t, name); false {
			erro(at(ctx,lhs), "resolve selection object '%v' (%s) error", lhs, name).debug()
			return
        } else if !isNull(o) {
			lhs = o
		} else if tok == SELECT_PROG2 {
			res = makeNull(ctx.Position()) // ignore
			return
		} else {
			erro(at(ctx,lhs), "'%v' is undefined", lhs).debug()
			return
        }
	case *globpat:
		if o, y := optionalize(ctx, lhs); y { lhs = o } else {
			erro(at(ctx,lhs), "selection of '%v' is undefined", lhs).debug()
		}
	}

	if rhs := p.selector(ctx); isNull(rhs) {
		res = makeNull(ctx.Position())
	} else {
		if v, y := optionalize(ctx, rhs); y { rhs = v } // foo→bar?
		res = makeSelection(ctx.Position(), tok, lhs, rhs)
	}

	if (p.tok == SELECT_PROP || p.tok == SELECT_PROG1 || p.tok == SELECT_PROG2) {
		res = p.selectExpr(ctx, res) // Continue the selection recursivly.
	}
	return
}

// ----------------------------------------------------------------------------
// Common productions

func (p *parser) isEndOfLine() bool {
	// If there's a comment right after the parsed expression, we break
	// the expression list to treat the end-of-line comment like a LINEND.
	return p.lineComment != nil || p.tok == LINEND || p.tok == EOF
}

func (p *parser) isEndOfList(ctx Context) bool {
	// If there's a comment right after the parsed expression, we break
	// the expression list to treat the end-of-line comment like a LINEND.
	if p.lineComment != nil || p.tok.isListDelim() || (can(ctx, getParseLeftHandSide{}) && p.tok.isAssign()) {
		return true
	}
	if can(ctx, getParseIsRecipe{false}) && p.tok == RECIPE { // TODO: using p.isRecipeStart()
		return true
	}
	return false
}

func (p *parser) isEndOfURL(ctx Context) bool {
	return p.tok == SPACE || p.isEndOfLine() || p.isEndOfList(ctx)
}

func (p *parser) isEndOfDotConcat(ctx Context) bool {
	// Expressions like `FOO.BAR(xxx)` does not count.
	switch p.tok {
	case SPACE, LPAREN, COLON, PCON, ASSIGN: fallthrough
	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2: return true
	}
	return p.isEndOfLine() || p.isEndOfList(ctx)
}

func (p *parser) rule_params(ctx Context, args []Value) (err error) {
	defer trace(ctx)

	var s = get_scope(ctx)

	if checkpoints {
		if !strings.HasPrefix(s.comment, "rule ") {
			erro(ctx, "wrong scope for rule params: %s", s.comment).debug()
		}
	}

	for _, arg := range args {
		switch ctx := at(ctx, arg) ; arg.(type) {
		case *bareword, *barecomp:
			var a = s.auto(ctx, arg.string(ctx))
			s.alias(ctx, a, strconv.Itoa(len(p.ruparas)+1))
			p.ruparas = append(p.ruparas, a)
		default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
			erro(ctx, "bad parameter form (%v)", ts(arg)).debug()
		}
	}
	return
}

func (p *parser) depends(ctx Context, params bool) (res []Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "depends")) }

	for p.tok != BAR && p.tok != SEMICOLON && !p.isEndOfLine() {
		if p.tok == COLON {
			// FIXME: this check is not working!
			// FIXME: detects unexpected colon ':'
			erro(p, "unexpected colon").debug()
			p.next(ctx, true) // just ignore this colon
		} else if p.spaces(ctx) ; !p.isEndOfLine() {
			var val Value
			if len(res) == 0 {
				val = p.expr(parser_params_context{ctx})
			} else {
				val = p.expr(ctx)
			}

			if params {
				if g, y := val.(*group); y && len(g.elems) == 1 {
					if g, y = g.elems[0].(*group); y {
						p.rule_params(ctx, g.elems)
						continue
					}
				}
			}

			res = append(res, merge(val)...)
			if p.tok == SPACE { p.next(ctx, true) }
		}
	}
	return
}

// If lhs is set, result list elements which are identifiers are not resolved.
func (p *parser) values(ctx Context, ii ...any) (values []Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Values")) }

	defer trace(ctx)

	for _, i := range ii {
		switch v := i.(type) {
		case Value: values = append(values, v)
		default: erro(ctx, "unsupported value: %v{%v}", typeof(i), i).debug(5)
		}
	}

	for p.spaces(ctx); !p.isEndOfList(ctx); p.spaces(ctx) {
		var prev = p.pos
		if values = append(values, p.expr(ctx)); p.pos == prev {
			erro(at(ctx,p), "bad: %v %v; %v", p.tok, p.lit, values).debug()
			break
		}

		// If there's a comment right after the parsed expression, we break
		// the expression list to treat the end-of-line comment like a LINEND.
		if p.tok == EOF || p.tok == LINEND || p.lineComment != nil { break }
	}
	return
}

func (p *parser) list(ctx Context, ii ...any) *list {
	return makeList(p.values(ctx, ii...)...)
}

func (p *parser) group(ctx Context) *group {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Group")) }

	ctx = parser_group_context{token_aware_context{at(ctx, p),COMMA}}

	p.expect(LPAREN)
	p.spaces(ctx)

	var elems, converted = p.values(ctx), false
	for p.tok != RPAREN && p.tok != EOF {
		// if p.tok == COMMA { warn(ctx, "%020b: %v %v", p.bits, p.tok, p.lit).debug() }
		// if p.tok == COMMA { p.next(ctx, true) }
		switch p.tok {
		case BAR, COMMA, SEMICOLON:
			elems = append(elems, p.punctuation())
			p.spaces(ctx)
		}
		var next *list
		next = p.list(ctx)
		if !converted {
			elems = []Value{ makeList(elems...), next }
			converted = true
		} else {
			elems = append(elems, next)
		}
	}
	p.expect(RPAREN)
	return makeGroup(ctx.Position(), elems...)
}

func (p *parser) argumentedExpr(ctx Context, x Value) *argumented {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "argumented")) }

	ctx = parser_group_context{token_aware_context{at(ctx, p),COMMA}}

	p.next(ctx, true) // skip LPAREN

	var a = []Value{ p.list(ctx) }
	for p.tok != RPAREN && p.tok != LINEND && p.tok != EOF {
		switch p.tok {
		case COMMA: p.next(ctx, true) // skip COMMA
		case BAR, SEMICOLON:
			if false {
				a = append(a, p.punctuation())
				p.spaces(ctx)
			} else {
				erro(ctx, "unexpected punctuation: %v", p.tok).debug()
			}
		}
		a = append(a, p.list(at(ctx, p)))
	}
	p.expect(RPAREN)
	return makeArgumented(x, a...)
}

func (p *parser) globmeta(ctx Context) (x *globmeta) {
	pos, tok := p.Position(), p.tok
	p.step()
	return makeGlobMeta(pos, tok)
}

func (p *parser) globrange(ctx Context) (x *globrange) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Glob")) }

	ctx = at(ctx, p)
	p.expect(LBRACK) // skip '['

	chars := p.expr(ctx)
	p.expect(RBRACK) // skip ']'

	return makeGlobRange(ctx.Position(), chars)
}

func (p *parser) glob(ctx Context, x Value) (g *globpat) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Glob")) }

	ctx = parser_glob_context{at(ctx, p)}

	if y := x == nil; y {
		g = &globpat{}
	} else if g, y = x.(*globpat); !y || g == nil {
		g = makeGlobPat(ctx, x)
	}

	for p.tok != RBRACE && p.tok != EOF && p.lineComment == nil {
		var v Value
		switch p.tok {
		case SELECT_PROP, SELECT_PROG1, SELECT_PROG2, PCON, RPAREN, COMMA, SPACE, LINEND, EOF:
			return
		case STAR, DAST, QUE:
			v = p.globmeta(ctx) // * ** ?
		case LBRACK:
			v = p.globrange(ctx) // [abc0-9xyz]
		default:
			v = p.expr(ctx)
		}
		g.elems = append(g.elems, v)
	}
	return
}

func (p *parser) perc(ctx Context, x Value) Value {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Perc")) }

	ctx = parser_perc_context{at(ctx, p)}

	var (
		pos = p.pos
		y Value
	)
	if p.step(); pos+1 == p.pos { // joint, e.g. '%.o', but skip '% .o'
		switch p.tok {
		case COLON, DOLON,
			LPAREN, RPAREN,
			LBRACK, RBRACK,
			PCON,   SEMICOLON,
			COMMA,  SPACE,
			LINEND:
		case PERC: // %%
			p.step() // consume the second %
			position := p.Position()
			perc2 := makePercpat(position, nil, nil)
			if pos+2 == p.pos {
				switch p.tok {
				case PERC: // %%%
					erro(p, "too many %")
				case PCON: // FIXES: %%/xxx -> Path(%% xxx)
					x = makePercpat(position, x, perc2)
					return p.path(ctx, x)
				case COLON,    DOLON,
					LPAREN,    RPAREN,
					LBRACK,    RBRACK,
					LBRACE,
					SEMICOLON, COMMA,
					SPACE,     LINEND:
				default:
					var (
						yy = p.expr(ctx)
						_, ok = yy.(*path)
					)
					if ok { erro(p, "incorrect: %v, %v", x, yy) }
					assert(!ok, "the second part of aaa%%bbb/foo/bar parsed incorrectly as path")
					perc2.Suffix = yy
				}
			}
			y = perc2
		default:
			y = p.expr(ctx)
		}
	}
	return makePercpat(p.loc(pos), x, y)
}

func (p *parser) regex(ctx Context) (_ Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "regex")) }

	var rx string
	var pos = p.Position()

	ctx = parser_regex_context{at(ctx, p)}

	defer trace(ctx)

	if checkpoints {
		if !p.scanner.bits.isBrace() {
			erro(ctx, "wrong scan state: %v", p.scanner.scanstate).debug(3)
		}
		if !p.scanner.bits.isBraceRaw() {
			erro(ctx, "wrong scan state: %v", p.scanner.scanstate).debug(3)
		}
	}

	for ; p.tok != RBRACE && p.tok != EOF; p.scan() {
		if p.tok == ESCAPE { rx += "\\" }
		rx += p.lit
	}

	p.expect(RBRACE)

	var err error
	var x = &regexpat{valbase{pos}, nil} // TODO: correct regexp pattern value
	if x.Regexp, err = regexp.Compile(rx); err != nil {
		note(ctx, "regex: %v", rx)
		erro(at(p,pos), "regex: %v", err).debug(6)
	}
	return x
}

func (p *parser) pair(ctx Context, x Value) *pair {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "pair")) }

	ctx = at(ctx, p)
	p.step()

	var y Value
	if p.isEndOfList(ctx) {
		y = makeNull(ctx.Position())
	} else {
		y = p.expr(ctx)
	}

	return makePair(x, y)
}

func (p *parser) flag(ctx Context) flag {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "flag")) }

	ctx = at(ctx, p)
	p.step() // skip dash '-'

	var x Value
	// flag expressions, excluding "-)" "-]" "-}" "-\n", "-=", "-:", etc.
	if p.isEndOfLine() || p.isEndOfList(ctx) || p.tok == SPACE || p.tok == RECIPE {
		x = makeNull(ctx.Position())
	} else if false {
		x = p.expr(ctx)
	} else {
		x = p.unary(ctx)
		l: for p.tok == DOT || !(_operator_beg < p.tok && p.tok < _closure_beg) {
			switch p.tok {
			case COMMENT, HASH, SPACE, RECIPE, LINEND, EOF: break l
			case DELEGATE, CLOSURE: x = compose(ctx, x, p.unary(ctx))
			default: if p.tok.isClosure() || p.tok.isDelegate() {
				x = compose(ctx, x, p.unary(ctx))
			} else {
				break l
			}}
		}
	}
	if x == nil { erro(ctx, "nil flag name").debug() }
	return flag{x}
}

func (p *parser) negative(ctx Context) negative {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "negative")) }
	p.expect(EXC)
	return negative{p.expr(ctx)}
}

func (p *parser) punctuation() *punctuation {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "punctuation")) }
	var vb, tok = p.valbase(), p.tok
	p.step()
	return &punctuation{vb, tok}
}

func (p *parser) escape(ctx Context) (v Value) {
	var vb, lit = p.valbase(), p.lit
	p.expect(ESCAPE)
	return &escaped{vb, lit}
}

func (p *parser) literal(ctx Context) (v Value) {
	var tok, lit = p.tok, p.lit
	ctx = at(ctx, p)
	p.step()

	// ESCAPE is handled in value.EscapeChar
    switch position := ctx.Position(); tok {
    case BAR: erro(ctx, "`|` is deprecated, changed the modifiers!")
    case BINARY:      v = ParseBinary(position, lit)
    case OCTAL:       v = ParseOctal(position, lit)
    case INTEGER:     v = ParseDecimal(position, lit)
    case HEXADECIMAL: v = ParseHexadecimal(position, lit)
    case FLOATING:    v = ParseFloat(position, lit)
    case DATETIME:    v = ParseDateTime(position, lit)
    case DATE:        v = ParseDate(position, lit)
    case TIME:        v = ParseTime(position, lit)
    case URI:         v = ParseURL(position, lit)
    case BAREWORD:    v = makeBareword(position, lit)
    case STRING:      v = makeStrlit(position, lit)
    case RAW:         v = makeRaw(position, lit)
    default: unreachable()
    }
	return
}

func (p *parser) compound(ctx Context) *compound {
	var elems []Value

	p.step()

	for p.tok != EOF && p.tok != COMPOSED && p.tok != LINEND {
		if p.tok == RAW {
			elems = append(elems, p.literal(ctx))
		} else {
			elems = append(elems, p.expr(ctx))
		}
	}
	p.expect(COMPOSED)
	return makeCompound(elems...)
}

// Parses dot composing expressions (TODO: check against file extensions).
//   .foo
//   .'foo'
//   ."foo"
//   .(foo)
//   ..foo
//   ..'foo'
//   .foo.bar
func (p *parser) dot(ctx Context, x Value) (res Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Dot")) }

	defer trace(ctx)

	ctx = token_aware_context{at(ctx, p),DOT}

	for !p.isEndOfDotConcat(ctx) {
		x = compose(ctx, x, p.composite(ctx))
		if p.tok == DOT /*&& comp.End() == p.pos*/ {
			x = compose(ctx, x, &punctuation{p.valbase(), p.tok})
			p.step() // skips '.'
		}
	}

	return x
}

func (p *parser) path(ctx Context, start Value) (res *path) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Path")) }

	if start == nil {
		erro(ctx, "nil path starter").debug()
		return
	}

	ctx = parser_path_context{at(ctx, start)}

	switch t := start.(type) {
	case     *path: res = t
	case   *strlit: res = makePath(splitPathStr(ctx, t.s)...)
	case *compound: res = makePath(splitPathStr(ctx, t.string(ctx))...) // FIXME: dont final here
	default:        res = makePath(start)
	}

	for p.tok == PCON && p.tok != EOF {
		// skips repeated '/' sequence
		for p.step(); p.tok == PCON; p.step() {}

		switch p.tok {
		case LPAREN, LBRACE, RPAREN, RBRACE, COMMA, SPACE, LINEND:
			res.elems = append(res.elems, _pathpun(at(ctx, p), PTAIL)) // after the last '/'
			return
		}

		var t = p.composite(ctx)
		if x, y := t.(*path); y {
			res.elems = append(res.elems, x.elems...)
		} else {
			res.elems = append(res.elems, t)
		}

		if p.tok == SPACE || p.isEndOfLine() { return }
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

func (p *parser) url(ctx Context, scheme Value) (res Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "URL")) }

	var (
		url = &URL{ Scheme:scheme }
		colon1 = p.expect(COLON) // consumes ':'
		colon2 = NoPos
		//colon3 = NoPos
		a = NoPos // @
	)

	if p.tok == PCON {
		p.step() // the first '/'
		if p.tok == PCON {
			p.expect(PCON) // the second '/'
		} else {
			erro(ctx, "TODO: URL path: %v (%T) (next: %s (%s))", scheme, scheme,  p.tok, p.lit).debug()
			res = makeNull(p.Position())
			return
		}
	} else if !p.isEndOfURL(ctx) {
		erro(at(ctx, p.loc(colon1)), "TODO: URL: %v (%T) (next: %s (%s))", scheme, scheme, p.tok, p.lit).debug()
		res = makeNull(p.Position())
		return
	}

	if !p.isEndOfURL(ctx) {
		userOrHost := p.composite(ctx)
		if p.tok == COLON {
			url.Username, colon2 = userOrHost, p.pos
			p.step() // ':'
			if p.tok != AT && p.tok != PCON && !p.isEndOfURL(ctx) {
				url.Password = p.composite(ctx)
			}
		} else {
			url.Host = userOrHost
		}
		if p.tok == AT {
			p.step() // '@'
		}
	}
	if url.Host == nil && colon2 == NoPos && a == NoPos && !p.isEndOfURL(ctx) {
		url.Host = p.composite(ctx)
		if p.tok == COLON {
			//colon3 = p.pos
			p.step() // ':'
			if p.tok != SPACE && p.tok != LINEND {
				url.Port = p.composite(ctx)
			}
		}
	}
	if p.tok == PCON {
		url.Path = p.path(ctx, _pathpun(ctx, p.tok))
	}
	// scanning '#' as HASH instead of COMMENT
	defer p.scanner.setBits(p.scanner.commentsOff())
	if p.tok == QUE {
		p.step() // '?'
		if p.tok != HASH && !p.isEndOfURL(ctx) {
			url.Query = p.composite(ctx)
		}
	}
	if p.tok == HASH {
		p.step() // '#'
		if !p.isEndOfURL(ctx) {
			url.Fragment = p.composite(ctx)
		}
	}
	return url
}

func (p *parser) resolve(ctx Context, name Value, str string) (result Value) {
    var pos Position
    if !(name == nil) { pos = name.Position() }
    if !pos.IsValid() { pos = ctx.Position() }
    if !pos.IsValid() { pos = p.Position() }
    if str == "" {
		erro(ctx, "resolve no-name : %v", ts(name)).debug()
		return
	}

	var s = get_scope(ctx)

	if d := auto_find(ctx, str); d != nil {
		return d
	}

	if _, o := s.find(str); o != nil {
		return o
	}

	if can(ctx, getParseIsAuto{str}) {
		if a := s.auto(ctx, str); a != nil {
			return a
		} else {
			erro(ctx, "failed auto: %v", ts(name)).debug()
			return
		}
	}

	if can(ctx, getParseIsConf{}) {
		// Create an empty def if referred in configuration.sm.
		result, _ = s.set(ctx, str, defConfRef)
		return
	}

	if c := _loader(ctx).project.configure; c != nil {
		return c.resolve(ctx, str)
    }
    return
}

func (p *parser) closuredelegate_obj(ctx Context, lTok token, name Value, isClosure bool) (str string, obj Value) {
	defer trace(ctx) // backtrace on errors

	if x, y  := name.(*argumented) ; y { name = x.Value }
	if _, y  := name.(condval) ; !y {
		if v := name.expand(ctx) ; v == nil {
			erro(ctx, "%v is nil", ts(name)).debug()
			return
		} else {
			name = v
		}
	}

	if indeterminate(ctx, name) {
		return str, name
	}

	str = name.string(ctx)

	if lTok == LBRACE {
		if false {
			erro(ctx, "empty name: %s", ts(name)).debug()
			return
		}

		var t = get_project(ctx).resolveEntries(ctx, name, false)
		if t == nil {
			erro(ctx, "resolved %v is nil", ts(name)).debug()
			return
		} else {
			obj, _ = t[0].(Object)
			return
		}
	}

	if str == "" {
		switch name.(type) {
		case condval, *closure, *delegate, *selection:
			return str, name
		}
		erro(ctx, "%v is empty for name", ts(name)).debug()
		return
	}

	if t := p.resolve(ctx, name, str) ; t != nil {
		obj, _ = t.(Object)
		return
	}

	if isClosure || can(ctx, getParseCanUndef{}) || exable(ctx, name, nil) {
		obj = name // recursive delegation or closure
		return
	}

	erro(ctx, "resolve(%v) ⇒ nil", ts(name)).debug()
	return
}

func (p *parser) auto_arg0(ctx Context, tokLp token, isClosure bool) (_ Value) {
	if tokLp != LPAREN {
		erro(ctx, "auto: incorrect left paren: %v", tokLp).debug()
		return
	}

	defer trace(ctx)

	var ac = automatic{ Context:ctx, defs:make(auto_defs) }
	ac.suppress = ac.has

	ctx = &ac

	var vals []Value
	for p.spaces(ctx); !p.isEndOfList(ctx); p.spaces(ctx) {
		var val = p.expr(ctx)
		vals = append(vals, val)

		var s string
		if x, y := val.(*pair); y {
			s, val = x.key.string(ctx), x.val
		} else {
			s = val.string(ctx)
			val = nil
		}
		if s == "" {
			erro(at(ctx, val), "auto: %v is empty", val).debug()
		} else {
			ac.set(at(ctx, val), s, val)
		}

		if p.tok == COMMA || p.tok == EOF || p.tok == LINEND || p.lineComment != nil {
			break
		}
	}


	return makeList(vals...)
}

func (p *parser) closuredelegate_args(ctx Context, name string, tokLp token, isClosure bool) (args []Value) {
	switch name {
	case "auto"    : args = append(args, p.auto_arg0(ctx, tokLp, isClosure)); if !isClosure { ctx = parser_auto_context{ctx} }
	case "case"    : args = append(args, p.list(ctx)); ctx = parser_undef_context{ctx}
	case "foreach" : args = append(args, p.list(ctx)); ctx = parser_foreach_context{ctx}
	case "and","or": ctx = parser_undef_context{ctx}; args = append(args, p.list(ctx))
	default:         args = append(args, p.list(ctx))
	}

	for p.tok == COMMA {
		p.next(ctx, true) // consumes COMMA
		args = append(args, p.list(ctx))
	}
	return
}

func (p *parser) closuredelegate_abc(ctx Context, isClosure, special bool) (tok token, obj Value, args, opts []Value) {
	defer trace(ctx)

	var name Value
	var str string

	tok, str = p.tok, p.lit ; p.step()

	if special {
		if obj = p.resolve(ctx, nil, str); obj == nil {
			erro(ctx, "not defined %v (name=%s)", tok, str).debug()
		}
		return
	}

	switch p.tok {
	case LPAREN, LBRACE: // $(...), ${...}
		tok = p.tok // use LPAREN, LBRACE
		p.step() // skips LPAREN, LBRACE

		if p.tok == SPACE {
			erro(ctx, "unexpected spaces").debug()
			return
		}

		if name = p.expr(ctx); name == nil {
			erro(ctx, "%v : no name parsed", tok).debug()
			return
		}

		// For examples: foo?  foo(a,b,c)?
		if x, y := optionalize(ctx, name); y { name = x }

		if a, y := name.(*argumented); y {
			name, opts = a.Value, merge(a.args...)

			// For examples: foo?(a,b,c)
			if x, y := optionalize(ctx, name); y { name = x }

			for _, v := range opts {
				if p, y := v.(*pair);  y { v = p.key }
				if _, y := v.(flag); !y {
					erro(at(ctx,v), "not a Flag: %v", ts(v)).debug()
					return
				}
			}
		}

		if name == nil {
			erro(ctx, "name %v is nil").debug()
			return
		}

		str, obj = p.closuredelegate_obj(ctx, tok, name, isClosure)

		if (tok == LPAREN && p.tok != RPAREN) || (tok == LBRACE && p.tok != RBRACE) {
			args = p.closuredelegate_args(ctx, str, tok, isClosure)
		}
		switch tok {
		case LPAREN: p.expect(RPAREN)
		case LBRACE: p.expect(RBRACE)
		}

	default:
		if !isClosure { // $(...), disabled $name.
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v` or `%v` or quotes", LPAREN, LBRACE).debug()
			return
		}

		if !(p.tok == STRING || p.tok == COMPOUND) {
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v`, `%v` or quotes, not %v %v", LPAREN, LBRACE, p.tok, p.lit).debug()
			return
		}

		pos := p.Position()
		tok = p.tok

		// &'xxxx' or &"xxxx"
		if name = p.expr(ctx); name == nil {
			erro(at(ctx,pos), "parsed name is nil").debug()
			return
		}

		if indeterminate(ctx, name) {//, /* expandClosure */final
			erro(at(ctx,name), "name '%v' is closured", ts(name)).debug()
			return
		}

		str, obj = p.closuredelegate_obj(at(ctx,name), tok, name, isClosure)
	}

	if obj == nil && str != "" {
		if proj := get_project(ctx); proj.ext.Plugin != nil {
			if t, e := proj.ext.Lookup(str); e == nil && t != nil {
				erro(at(ctx,name), "TODO: convert ext symbol: %v : %v", name, ts(t)).debug()
				return
			}
		}
	}
	return
}

func (p *parser) closuredelegate(ctx Context, isClosure, special bool) (result Value) {
	if l_traverse.enabled {	defer un(l_trace(l_traverse, "closuredelegate")) }

	ctx = parser_call_context{token_aware_context{at(ctx, p),COMMA}}

	defer trace(ctx)

	tok, obj, args, opts := p.closuredelegate_abc(ctx, isClosure, special)

	if obj == nil {
		erro(ctx, "%v : nil symbol", tok).debug()
		return makeNull(ctx.Position())
	}

	if isClosure {
		return makeClosure(ctx.Position(), tok, obj, opts, args...)
	} else if x, y := obj.(*def); y && x.origin == defCodeBlockAuto {
		return x.value
	} else {
		return makeDelegate(ctx.Position(), tok, obj, opts, args...)
	}
}

func (p *parser) unary(ctx Context) (x Value) {
	if l_traverse.enabled && false { defer un(l_trace(l_traverse, "unary")) }

	defer trace(ctx)

	switch p.tok {
	case ASSIGN: // Example: '=xxx'
		if !can(ctx, getParseLeftHandSide{}) {
			var v Value
			var s = p.Position()
			if p.step(); p.isEndOfList(ctx) {
				v = makeNull(s)
			} else {
				v = p.expr(ctx)
			}
			return &pair{makeNull(s), v}
		}

	case BAREWORD:
		return p.bare(ctx)

	case BINARY, OCTAL, INTEGER, HEXADECIMAL, FLOATING, DATETIME, DATE, TIME, URI, STRING/*, RAW*/:
		return p.literal(ctx)

	case COMPOUND:
		return p.compound(ctx)

	case CLOSURE:
		return p.closuredelegate(ctx, true, false)

	case DELEGATE:
		return p.closuredelegate(ctx, false, false)

	case ESCAPE: // \
		return p.escape(ctx)

	case LPAREN: // (
		return p.group(ctx)

	case LBRACE: // {
		return p.braced(ctx)

	case COMMA:
		if v, y := do(ctx, getParseAware{COMMA}).(bool); !y || !v {
			return p.punctuation()
		}

	case AT, BAR, PLUS, SEMICOLON:
		return p.punctuation()

	case STAR, DAST, QUE, LBRACK: // * ** ? [
		return p.glob(ctx, nil) // ie. no prefix

	case PERC: // %bar (ie. no prefix)
		return p.perc(ctx, nil)

	case MINUS:
		return p.flag(ctx)

	case EXC:
		return p.negative(ctx)

	case PCON: // The root of the path
		return p.path(ctx, _pathpun(ctx, PROOT))

	case TILDE: // ~
		tok, ctx := p.tok, at(ctx, p)
		p.step() // TODO: ~user, aka $(HOME)
		return _pathpun(ctx, tok)

	case DOT, DOTDOT: // . ..
		pos, tok := p.Position(), p.tok
		switch p.step(); {
		case p.tok == PCON:
			ctx = at(ctx, pos)
			return p.path(ctx, _pathpun(ctx, tok))
		case tok == DOT, tok == DOTDOT:
			x = &punctuation{valbase{pos}, tok}
			if v, y := do(ctx, getParseAware{DOT}).(bool); !y || !v {
				x = p.dot(ctx, x)
			}
			return
		default:
			erro(at(ctx,pos), "unexpected token: %v, %v %s", tok, p.tok, p.lit).debug()
			return makeNull(pos)
		}

	default:
		if t := p.tok.isClosure(); t || p.tok.isDelegate() {
			return p.closuredelegate(ctx, t, true)
		} else if p.tok.isKeyword() { // keywords here are barewords
			return p.bare(ctx)
		}
	}

	if p.lineComment != nil {
		for _, comment := range p.lineComment.list {
			erro(at(p,comment.pos), "# %s", comment.string).debug()
		}
	}

	erro(p, "bad: %v (lit=%s, scan=%v)", p.tok, p.lit, p.scanner.scanstate).debug()

	p.step() // go to the next token

	return makeNull(ctx.Position())
}

func (p *parser) isParametersGroup(ctx Context, x Value) (res bool) {
	if can(ctx, getParseCanParams{}) {
		if g, y := x.(*group); y && len(g.elems) == 1 {
			_, res = g.elems[0].(*group)
		}
	}
	return
}

func (p *parser) composite(ctx Context) (x Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "composite")) }

	defer trace(ctx)

	x = p.unary(ctx)

	switch p.tok { // check composible expressions
	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2: // foo->bar  foo=>bar  foo~>bar
		// accepts 'foo=>bar', but 'foo => bar' is different
		x = p.selectExpr(ctx, x)
		break

	case STAR, DAST, QUE, LBRACK: // * ** ? [
		if !can(ctx, getParseGlob{}) {
			if p.tok == QUE {
				switch p.step(); p.tok {
				case SPACE, RPAREN, RBRACK, RBRACE, COMMA, SELECT_PROP, SELECT_PROG1, SELECT_PROG2, LINEND:
					return condish(ctx, x)
				}
			}
			if _, y := x.(*globpat); !y {
				x = p.glob(ctx, x)
			}
		}

	case PERC: // foo%bar ; FIXME: %/foo/bar -> Path(% foo bar)
		x = p.perc(ctx, x)

	case DOT: // foo.bar.baz.o ; FIXME: push bits when parsing $(...)
		if v, y := do(ctx, getParseAware{DOT}).(bool); !y || !v {
			x = p.dot(ctx, x)
		}

		// TODO: parse to Qualiword

	// case PCON: // ie. subdir/in/somewhere
	// 		switch x.(type) { // Path expressions, except '-I/path/to/include'
	// 		case flag: // By pass expressions like -I/foo/bar.
	// 		default: x = p.path(ctx, lhs, x)
	// 		}

	case COLON:
		if (can(ctx, getParseIsRecipe{false}) || !can(ctx, getParseLeftHandSide{})) {
			if isKnownURLScheme(x.string(at(ctx, p))) { x = p.url(ctx, x) }
		}
	}
	return
}

func (p *parser) expr(ctx Context) (x Value) {
	if false && l_traverse.enabled { defer un(l_trace(l_traverse, "expr")) }

	defer trace(ctx)
	defer flush(ctx)

	var tok, lit = p.tok, p.lit

	if x = p.composite(ctx); x == nil {
		erro(p, "invalid (%v,%v; prev=%v,%v)", p.tok, p.lit, tok, lit).debug(6)
		return
	}

	if can(ctx, getParseGlob{}) { return }

	var lhs = can(ctx, getParseLeftHandSide{})
	if p.tok.isAssign() && lhs { return }
	if p.isParametersGroup(ctx, x) { return }

	var n int

composeLoop:
	switch p.tok {
	case ASSIGN: // Example: 'key=value'
		if !lhs {
			x = p.pair(ctx, x)
		}
		return

	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2: // Example: foobar⇒run(-gen)
		x = p.selectExpr(ctx, x)
		goto composeLoop

	case LPAREN:
		if x = p.argumentedExpr(ctx, x); x != nil {
			goto composeLoop
		}
		return

	case PCON:
		// Path, excepts '-I/path/to/include'
		switch x.(type) {
		case flag:
		default: x = p.path(ctx, x)
		}
		return // FIXES: a%%b/foo/bar -> Path(a%%b foo bar)

	case BAR: // Example: [(var)|...]
		if _, y := x.(*group); y { return }

	case COMMA:
		if v, y := do(ctx, getParseAware{COMMA}).(bool); y && v { return }

	case COMPOSED, COLON, RAW, RPAREN, RBRACK, RBRACE, SPACE, SEMICOLON, LINEND, EOF:
		return // terminate
	}

	x = compose(ctx, x, p.composite(ctx))

	switch p.tok { case SPACE, COMMENT, LINEND, EOF: return }

	n += 1 ; goto composeLoop // compose as many as possible
}

// ----------------------------------------------------------------------------
// Clauses & Declarations

type clauseopts struct {
	generalOpts

    keyword token // e.g. use, files, eval, etc.

    skip bool // e.g. -cond(false{}), -if(no{})

	conds []Value `if,cond,where`

    values, remainder []Value // all values (unparsed) and remainder
	spec []Value
}

type parseSpecFunc func(Context, *commentGroup, *clauseopts, int)

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

func (p *parser) _parseUseSpecProps(ctx Context, props []Value) (opts useOpts, params []Value, err error) {
	ctx = at(ctx, p)

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
                erro(at(ctx,t.key), "parameter `%v' unsupported `%T`", prop, prop)
                return
            }
        case *argumented: // -param(value)
            switch tt := t.Value.(type) {
            case flag:
                switch s = tt.Value.string(ctx); s {
                case "use": useList = append(useList, t.args...)
                default: params = append(params, prop)
                }
            default:
                erro(at(ctx,t.Value), "parameter `%v' unsupported `%T`", prop, prop)
                return
            }
        default:
            erro(at(ctx,prop), "parameter `%v` unsupported `%T`", prop, prop)
            return
        }
    }
    return
}

func (p *parser) use(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	if p.imports = append(p.imports, &usespec{ g.spec }); g.skip {
		// TODO: maybe give some information
		return
	}

	ctx = at(ctx, g.spec[0])

	defer trace(ctx)

	var specVals, arged []Value
	switch v := g.spec[0].(type) {
	case *delegate:
        for _, val := range xmerge(ctx, v) {
            if !isTrivial(val) { specVals = append(specVals, val) }
		}
    case *pair:
        var s string
        if f, ok := v.key.(flag); !ok {
            erro(ctx, "'%v' invalid use spec", v.key)
            return
        } else if s = f.Value.string(ctx); s != "list" {
            erro(ctx, "'%v' invalid use spec, do you mean -list?", v.key)
            return
        }

        for _, val := range xmerge(ctx, v.val) {
            if !isTrivial(val) { specVals = append(specVals, val) }
        }
	case *argumented:
        for _, val := range xmerge(ctx, v.Value) {
            if !isTrivial(val) { specVals = append(specVals, val) }
        }
		arged = v.args
	default:
		specVals = append(specVals, v)
    }
	if len(specVals) == 0 {
        erro(ctx, "empty use spec: %v", ts(g.spec[0])).debug()
        return
    }

	var opts useOpts
	var args = parseOpts(ctx, &opts, append(g.remainder, g.spec[1:]...)...)
	for _, a := range args {
		if _, ok := a.(flag); ok || true {
			erro(at(ctx,a), "unkown use opts: %v", ts(a)).debug()
			return
		}
	}

	var wg sync.WaitGroup
	var l = _loader(ctx)
	for _, specVal := range specVals {
		if ctx := at(ctx, specVal.Position()); true {
			l.usespec(ctx, opts, specVal, arged, args...)
		} else {
			var dc = diagnostic{ Context: ctx }
			wg.Add(1); go func() {
				defer func() {
					if false { trace(&dc) }
					if len(dc.points) > 0 { _diagnostic(ctx).nest(dc.points) }
					wg.Done()
				} ()
				l.usespec(ctx, opts, specVal, arged, args...)
			} ()
		}
	}

	wg.Wait()
	return
}

func (p *parser) include(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Spec")) }

	var opts = includeOpts{ clauseopts: g }
	if vals := parseOpts(ctx, &opts, g.remainder...); len(vals) > 0 {
		// TODO: deal with the unparsed generic options
		warn(ctx, "unknown opts: %v", vals).debug()
	}

	if len(g.spec) < 1 {
		erro(ctx, "expecting include file: %v", g.spec).debug()
		return
	}

	var x = g.spec[0]//.expand(ctx, final|expandPlaceholder)
	var l = _loader(ctx)
	if p.spaces(ctx); p.tok == COLON {
		switch x.(type) {
		case *File, *strlit, *compound: // escape from file searching
		default: if file := l.project.file(ctx, x.string(ctx)); file != nil {
			x = file
		} else if val := x.expand(ctx); !isNull(val) && val != x {//, final
			x = val
		}}

		x = p.rule(ctx, nil, []Value{x}) // this should return a Rule
	}

	if !g.skip { l.include(ctx, x, opts) }
}

func (p *parser) files(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	defer trace(ctx)

	if len(g.spec) != 1 {
		erro(ctx, "too many files properties: %v", g.spec).debug()
		return
	}

	var path Value
	if p.tok == SELECT_PROG1 {
		p.next(ctx, true) // step forward with spaces skipped
		if p.tok == LINEND || p.lineComment != nil {
			erro(ctx, "expecting files path")
		}
		path = p.expr(ctx)
	}

	p.spaces(ctx)

	if p.lineComment != nil {
		//spec.Comment = p.lineComment
	}
	if g.skip {
		// TODO: maybe give some information
		return
	}

	ctx = at(ctx, p)

	var opts = cacher{ g.generalOpts }
	if rest := parseOpts(ctx, &opts, g.remainder...); rest != nil {
		erro(ctx, "unsupported opts: %v", rest).debug()
		return
	}

	var pats []Value
	if x, y := g.spec[0].(*group); y {
		pats = x.elems
	} else if indeterminate(ctx, g.spec[0]) {
		pats = []Value{ g.spec[0] }
	} else {
		pats = xmerge(evaluation{ctx, defExpand1}, g.spec[0])
	}

	if path == nil {
		if len(pats) == 1 { if a, ok := pats[0].(*argumented); ok { if f, y := a.Value.(flag); y {
			var name = f.Value.string(ctx)
			switch name {
			default:
				// TODO: parse files options
				erro(at(ctx,f.Value), "invalid files flag: %v").debug()
				return
			}
		}}}

		var (
			files []*File
			newPats []Value
		)
		for _, pat := range pats {
			if f, ok := toFile(pat); ok {
				files = append(files, f)
			} else {
				newPats = append(newPats, pat)
			}
		}
		if len(files) > 0 {
			opts.cache(ctx, values(files), nil)
			pats = newPats
		}
		if len(pats) > 0 {
			var paths = []Value{ makeStrlit(g.spec[0].Position(), get_project(ctx).absPath) }
			opts.cache(ctx, pats, paths)
		}
	} else {
		var patsNew []Value
		for _, pat := range pats {
			if indeterminate(ctx, pat) {
				patsNew = append(patsNew, pat)
			} else {
				patsNew = append(patsNew, xmerge(ctx, pat)...)
			}
		}

		var paths []Value
		if g, ok := path.(*group); ok {
			paths = g.elems
		} else {
			paths = []Value{ path }
		}

		if len(patsNew) == 1 {
			if f, ok := patsNew[0].(flag); ok {
				var name = f.Value.string(ctx)
				switch name {
				default:
					// TODO: parse files options
					erro(at(ctx,f.Value), "invalid files flag: %v").debug()
					return
				}
			}
		}

		opts.cache(ctx, patsNew, paths)
	}
}

func (p *parser) evalConfiguration(ctx Context, g *clauseopts, props []Value) {
	var project = get_project(ctx)
	if project == nil {
		erro(ctx, "configuration: nil project").debug()
		return
	} else if project.configure == nil {
		erro(ctx, "configuration: no %s for %v", dotConfigure, project).debug()
		return
	}

	if entry := project.configure.defaultEntry; entry == nil {
		// no init entry from .configure
	} else if _, ts := entry.execute(at(ctx, entry.Position())); len(ts) > 0 {
		// FIXME: the entry might be a configure operation (see configure/.base/do.smart)
		for _, brk := range ts {
			if brk.what == traveFail {
				erro(at(ctx,entry), "execute '%v' failed: %v", entry, brk).debug()
			}
		}
	}

	if flush(ctx)>0 { return }
	if project.configured {
		prompt(ctx, "configuration: %v already configured\n", project)
		return
	}

	var ce = configurecontext{Context:ctx} ; defer ce.close()

	for _, dep := range xmerge(ctx, props/* [1:] */...) {//, final
		if re, y := dep.(*rule); !y {
			erro(ctx, "unsupported prerequisite: %T %v", dep, dep).debug()
		} else if _, ts := re.execute(ctx); len(ts) > 0 {
			for _, brk := range ts { if brk.what == traveFail {
				erro(at(ctx,re), "execute '%v' failed: %v", re, brk).debug()
			}}
		}
	}

	if flush(ctx)>0 { return }

	/***/ promptEnteringDirectory(ctx, project.absPath)
	defer promptLeavingDirectory(ctx, project.absPath)

	for _, entry := range project.configs { ce.execute(entry) }

	project.configured = true // relaxes configure()
}

func (p *parser) assert(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	if !g.skip { call(final{ctx}, "assert", g.remainder, g.spec...) }
}

func (p *parser) append(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	if !g.skip { call(final{ctx}, "append", g.remainder, g.spec...) }
}

func (p *parser) eval(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	defer trace(ctx)

	if g.skip { return }
	if g.spec == nil {
		var opts struct {
			configuration bool `configuration`
			optimize Value `opt,optimize`
		}
		for _, op := range parseOpts(ctx, &opts, g.values...) {//, plain
			var val Value
			if v, y := op.(*pair); y { op, val = v.key, v.val }
			if v, y := op.(flag); y {
				switch t := val != nil && val.true(ctx); v.Value.string(ctx) {
				case "dd": p.dd = t
				case "ddd":
					if u := _universe(ctx); val == nil {
						u.ddd = "yes"
					} else if t, y := boolVal(val); y {
						if t { u.ddd = "yes" } else { u.ddd = "" }
					} else {
						u.ddd = val.string(ctx)
					}
				}
			} else {
				erro(at(ctx,op), "unsupport flag: %v (%v)", ts(v), val).debug()
			}
		}

		// NOTE: see also universeContext.configure()
		if opts.configuration { p.evalConfiguration(ctx, g, g.spec) }
		return
	}

	prop0 := g.spec[0]

	if isTrivial(prop0) {
		erro(ctx, "illegal").debug()
		return
	}

	ctx = at(ctx, prop0)

	var opts []Value
	if a, y := prop0.(*argumented); y { prop0, opts = a.Value, a.args }

	name := prop0.string(ctx)
	if name == "configuration" {
		erro(ctx, "use '-configuration' instead (%v)", prop0).debug()
		return
	}

	resolved := p.resolve(ctx, prop0, name)

	switch x := resolved.(type) {
	case invoker:
		if b, y := x.(*builtin); y && !b.isCommand() {
			erro(ctx, "resolved builtin '%v' is not a command", prop0).debug()
			return
		}
		x.invoke(ctx, opts, g.spec[1:])
		return
	default:
		erro(ctx, "resolved '%v' is %s (%v)", prop0, typeof(resolved), *g).debug()
		return
	}

	/* TODO: if c, y := res.(code); y { ... } */
}

func (p *parser) directive(ctx Context) (props []Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "spec")) }

	defer trace(ctx)

	//var doc = p.leadComment
	var comment *commentGroup

ParamsParseLoop: // Parse the directive parameters
	for p.tok != EOF {
		switch p.spaces(ctx); p.tok {
		case COMMA, LINEND, RPAREN, RBRACE,
			SELECT_PROG1, COLON: break ParamsParseLoop
		}

		if p.lineComment != nil {
			comment = p.lineComment
			break
		}

		props = append(props, p.expr(ctx))
	}
	if comment != nil { /* TODO: directive documments */ }
	return
}

func (p *parser) spec(ctx Context, keyword token, pos Pos, f parseSpecFunc) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "spec("+keyword.String()+")")) }

	defer trace(ctx)

	var opts = clauseopts{ keyword: keyword }
	for p.spaces(ctx); p.tok == MINUS; p.spaces(ctx) {
		opts.values = append(opts.values, p.expr(ctx))
	}
	opts.remainder = parseOpts(ctx, &opts, opts.values...)

	for _, cond := range opts.conds {
		if t := cond.true(at(ctx, cond.Position())); !t {
			opts.skip = true
			break
		}
	}

	if p.spaces(ctx); p.tok == LINEND {
		if keyword == EVAL { f(at(ctx, p), nil, &opts, 0) } else {
			erro(at(ctx, p), "%v: nil specs", keyword).debug()
		}
		return
	} else if p.tok == LPAREN {
		p.next(ctx, true)
		for iota := 0; p.tok != RPAREN && p.tok != EOF && (p.stop == 0 || p.pos < p.stop); iota++ {
			// TODO: collect documentation comments
			for p.tok == SPACE || p.tok == LINEND { p.next(ctx, true) }
			if p.tok == RPAREN || p.tok == EOF { break  }
			if opts.spec = p.directive(ctx); true {
				f(at(ctx, p), p.leadComment, &opts, iota)
			}
			if p.tok == COMMA || p.tok == LINEND { p.next(ctx, true) }
		}
		p.expect(RPAREN)
		if p.spaces(ctx); p.tok != EOF { p.linend() }
		return
	}

	if p.tok != LINEND && p.tok != EOF && (p.stop == 0 || p.pos < p.stop) {
		if opts.spec = p.directive(ctx); true { f(ctx, nil, &opts, 0) }
		if p.tok == COMMA { p.next(ctx, true) }
	}
	if p.tok != EOF && (p.stop == 0 || p.pos < p.stop) {
		if p.spaces(ctx); p.lineComment == nil { p.linend() }
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
                if isTrivial(arg) { continue }
                f(makeBarecomp(arg), append(stems, arg))
            }
        })
    case *barecomp:
        for_ident_elems(ctx, t.elems, nil, func(elems, stems []Value) {
            if len(stems) == 0 {
				f(t, stems)
			} else {
                f(makeBarecomp(elems...), stems)
            }
        })
    default:
        f(t, nil)
    }
}

func (p *parser) define_idents(ctx Context, tok token, idents, value Value) (defs []*def) {
    for_idents(ctx, idents, func(ident Value, _ []Value) {
        if d := p.define(ctx, tok, ident, value); d != nil {
			defs = append(defs, d)
		}
    })
    return
}

func (p *parser) define(ctx Context, tok token, ident, value Value) (d *def) {
    defer trace(ctx)

	if checkpoints {
		defer func() {
			if d == nil {
				erro(ctx, "%v %v %v", ident, tok, ts(value)).debug()
			} else if d.value == nil && value != nil {
				erro(ctx, "%v %v %v", ident, tok, ts(value)).debug()
			}
		} ()
	}

    var alt Object

    switch t := ident.(type) {
    case *argumented:
        erro(ctx, "TODO: multiple defs: %v, args=%v", t.Value, t.args).debug()
        return

    case *group:
        erro(ctx, "TODO: multiple defs: %v", t.elems).debug()
        return

    case *selection:
        if v := t.expand(final{ctx}); v == nil {
            erro(ctx, "%v is nil", ts(t)).debug()
            return
        } else if x, y := v.(*def); !y {
            erro(ctx, "%v is not a def: %v", ts(t), ts(v)).debug()
            return
        } else {
            d = x
        }

    default: // *bareword, *barecomp, *qualiword, *path, flag:
        var name = t.string(ctx)
        if _, y := builtins[name]; y {
            erro(ctx, "`%v` is a builtin name (%v)", ident, name).debug()
            return
        }

        // Resolve base value to derive.
		var proj = get_project(ctx)
        var prev = proj.resolve(ctx, name)

        if d, alt = get_scope(ctx).set(at(ctx, t), name, defUndetermined); alt == nil {
            if d == nil {
                erro(ctx, "`%s` is undefined (%v)", name, ts(t)).debug()
                return
            }
        } else if tok == ASSIGN || tok == ASSIGN_EXC {
            if a, y := alt.(*def); !y {
                erro(ctx, "`%v` already defined (%T) (%v)", ident, alt, alt.owner()).debug()
                return
            } else if a.owner() == proj && a.origin != defConfRef {
                erro(ctx, "`%v` already defined (%T)", ident, alt).debug()
                return
            } else {
                d = a
            }
        } else if t, y := alt.(*def); !y {
            erro(ctx, "%s: object is not def: %s, %v", name, typeof(alt), ts(prev)).debug()
			return
		} else {
           d = t
        }

        if prev == nil {
            // no derived value
        } else if prev.owner() == proj {
            // not derivable def if they are from the same project
        } else if derived, y := prev.(*def); !y {
            // not a def
        } else if derived == nil {
            erro(ctx, "prev def '%s' is nil", name).debug()
        } else if derived == d || (d.value != nil && d.value.refs(ctx, derived)) {
            // same def
        } else if d != nil && (tok == ASSIGN_ADD || tok == ASSIGN_SHI) && alt == nil {
            if d.origin == defVoid { d.origin = derived.origin }
            if !isTrivial(derived.value) { d.append(ctx, derived.value) }
        }
    }

    if d == nil {
        erro(ctx, "def is nil: %v", ts(ident)).debug()
        return
    }

    d.position = ident.Position()

    switch tok {
    case ASSIGN    :                       d.set(ctx, defExpand0, value) //   =
    case ASSIGN_CO1:                       d.set(ctx, defExpand1, value) //  :=
    case ASSIGN_CO2:                       d.set(ctx, defExpand2, value) // ::=
    case ASSIGN_EXC:                       d.set(ctx, defExecute, value) //  !=
    case ASSIGN_QUE: if    alt == nil    { d.set(ctx, d.origin, value) } // ?=
    case ASSIGN_ADD: if!isTrivial(value) { d.set(ctx, d.origin, nil,   merge(  value)...) } // +=
    case ASSIGN_SHI: if!isTrivial(value) { d.set(ctx, d.origin, value, merge(d.value)...) } // =+
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
    default:
        erro(ctx, "unknown origin: %v %v %v", d.origin, d.name, tok).debug()
    }
    return
}

func (p *parser) assign_value(ctx Context, tok token) (value Value) {
	var l = _loader(ctx)
	defer l.closescope(l.openscope(fmt.Sprintf("def")))

	vals := p.values(parser_defvalue_context{ctx})
	p.lineComment = nil
	return ease(ctx, vals)
}

func (p *parser) assign(ctx Context, ident Value) (def *def) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, fmt.Sprintf("assign(%s)", ident))) }

	var tok = p.tok

	p.next(ctx, true) // assign token

	ctx = at(ctx, p)

	defer trace(ctx)

	// TODO: doc = p.leadComment
	// TODO: comment = p.lineComment
	var value = p.assign_value(ctx, tok)

	// NOTE: Put all explicit defs into project scope. It's important for defs enclosed
	//       in templates work.
	var l = _loader(ctx)
	if s := l.project.scope ; len(l.s) == 0 || l.s[0] != s {
		defer func(s []*scope) { l.s = s } (l.s)
		l.s = append([]*scope{s}, l.s...)
	}

	var defs = p.define_idents(ctx, tok, ident, value)
	if n := len(defs); n > 0 { def = defs[n-1] }

	if checkpoints {
		if def == nil {
			erro(ctx, "%v %v %v", ident, tok, ts(value)).debug()
		} else if def.value == nil && value != nil {
			erro(ctx, "%v %v %v", ident, tok, ts(value)).debug()
		}
	}
	return
}

func (p *parser) recipe(ctx Context) Value {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Recipe")) }

	var (
		// TODO: comment *commentGroup
		// TODO: doc = p.leadComment
		elems []Value
		isList bool
	)

	switch p.dialect {
	case "", "eval", "value":
		p.scanner.pop(isCompoundLine)
		p.next(ctx, true) // skip RECIPE or SEMICOLON and parse in list mode
		if isList = true; !p.isEndOfLine() {
			var a *argumented
			var x = p.expr(ctx) // parse first expr of recipe
			if x != nil { if a, _ = x.(*argumented); a != nil { x = a.Value } }
			if x == nil {
				erro(ctx, "parsed value is nil").debug()
			} else if p.dialect == "value" {
				// no resolving commands
			} else if t, y := x.(*bareword); !y {
				// does nothing
			} else if sym := p.resolve(ctx, t, t.s); false {
				erro(ctx, "resolve '%v' failed", x).debug()
			} else if isTrivial(sym) {
				erro(at(ctx,x), "resolved '%v' (from %v) is nil", t.s, x).debug()
			} else if false {
				erro(at(ctx,x), "builtin command no more supported, use $(%s ...) instead", t.s).debug()
			} else if b, y := sym.(*builtin); !y {
				erro(at(ctx,x), "'%s' is not a command (%s)", t.s, typeof(sym)).debug()
			} else if !b.isCommand() {
				erro(at(ctx,x), "'%s' is not a command, use $(%s ...) instead", t.s, t.s).debug()
			} else { x = sym }

			if a != nil {
				elems, a.Value = append(elems, a), x
			} else {
				elems = append(elems, x)
			}

			var cmdargs []Value
			var c = parser_recipe_context{ctx, true} // builtin recipe

			for p.tok != EOF && p.tok != SEMICOLON && p.tok != LINEND && p.lineComment == nil {
				if p.spaces(ctx); p.lineComment != nil { break }
				if !p.tok.isRuleDelim() {
					x = p.expr(c)
				} else {
					erro(ctx, "unsupported token: %s, %v", p.tok, elems).debug()
				}
				if cmdargs = append(cmdargs, x); p.tok == COMMA {
					p.next(ctx, true)
					elems = append(elems, makeList(cmdargs...))
					cmdargs = []Value{}
				}
				if p.lineComment != nil { break }
			}

			elems = append(elems, makeList(cmdargs...))
		}

	default:
		p.scanner.push(isCompoundLine) // NOTE: scanner does not set isCompoundLine correctly, fixit here
		p.next(ctx, true) // skip RECIPE or SEMICOLON and parse in line-string mode

		var c = parser_recipe_context{ctx, false} // builtin text
		for !p.isEndOfLine() {
			var x Value
			if p.tok == RAW {
				x = p.literal(c)
			} else {
				x = p.expr(c)
			}
			elems = append(elems, x)
		}
		p.scanner.pop(isCompoundLine)
	}
	if p.spaces(ctx); p.tok != EOF { p.linend() }
    if len(elems) == 0 {
        return makeNone(ctx.Position())
    } else if isList {
        return makeList(elems...)
    } else {
        return makeCompound(elems...)
    }
}

// Parsing (var a=xxx,b=yyy) definitions
func (p *parser) movar(ctx Context, args ...Value) (err error) {
	defer trace(ctx)
	var s = get_scope(ctx)
	for _, elem := range args {
		var kv, y = elem.(*pair)
		if !y || kv == nil {
			erro(at(ctx,elem), "bad var form (%v)", ts(elem)).debug()
			continue
		}

		if d, a := s.set(at(ctx, elem), kv.key, defUndetermined); a != nil {
			erro(at(ctx,kv), "'%v' already defined: %v", kv.key, ts(a)).debug()
		} else if d == nil {
			erro(at(ctx,kv), "'%v' not defined", kv.key).debug()
		} else {
			var v = kv.val
			if g, y := v.(*group); y { v = g.list() }
			d.val(at(ctx,kv), v)
		}
	}
	return
}

func (p *parser) defineConfigureTargets(ctx Context) {
	var proj = get_project(ctx)
	for _, t := range p.targets {
		var ctx = at(ctx, t)

		d, a := proj.set(ctx, t, defConfig)

		if d == nil && a != nil {
			if d, _ = a.(*def); d == nil {
				erro(ctx, "%v : already defined: %v", ts(t), ts(a)).debug()
				trace(ctx)
				return
			}
		}

		if d != nil && !d.position.IsValid() { d.position = t.Position() }
	}
}

func (p *parser) modifier(ctx Context) (res *modifier) {
	p.spaces(ctx)

	ctx = parser_modifier_context{at(ctx, p)}

	p.expect(LPAREN)
	p.spaces(ctx)

	var name string
	var nameVal = p.expr(ctx)
	var elems []Value
	switch n := nameVal.(type) {
	case *bareword: name = n.s
	case *delegate, *closure:
		var v = xmerge(at(ctx, n), nameVal)//, final
		if len(v) == 0 {
			erro(ctx, "empty modifier name: %v", n).debug()
			return
		}

		name, elems = v[0].string(ctx), v[1:]

	default:
		erro(ctx, "unsupported modifier: %v", ts(n)).debug()
		return
	}

	var movar bool
	switch name {
	case "var": movar = true
	case "configure":
		p.defineConfigureTargets(ctx)
		p.configure = true // set configure flag and define configure variables
	case "":
		erro(ctx, "empty modifier name: %v", ts(nameVal)).debug()
		return
	}

	if _, y := dialects[name]; y {
		if p.dialect != "" {
			erro(ctx, "multi-dialects unsupported, already defined '%s'", p.dialect).debug()
			return
		}

		p.dialect = name
	} else if _, y = modifiers[name]; !y {
		erro(ctx, "`%s` no such dialect or modifier", name).debug()
		return
	}

	for p.tok != RPAREN && p.tok != EOF {
		p.spaces(ctx)

		t := p.pos

		if vals := p.values(ctx); movar {
			p.movar(ctx, vals...) // TODO: define var one by one
		} else if n := len(vals); n == 1 {
			elems = append(elems, vals[0])
		} else if n > 1 {
			elems = append(elems, &list{elements{vals}})
		} else {
			elems = append(elems, &null{p.valbase()})
		}

		if p.tok == COMMA { p.next(ctx, true) }
		if p.pos == t {
			erro(ctx, "unsupported modifier arg: %v '%v'", p.tok, p.lit).debug()
			return
		}
	}

	p.expect(RPAREN)

	if nameVal == nil && len(elems) == 0 {
		erro(ctx, "empty modifier").debug()
	} else {
		res = new(modifier)
		res.position = ctx.Position()
		res.elems = append([]Value{nameVal}, elems...)
	}
	return
}

func (p *parser) modification(ctx Context) *modification {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "modification")) }

	// defer p.setbits(p.setbit(parseModification))

	ctx = at(ctx, p) // at(ctx, p.loc(p.expect(LBRACK)))

	var elems []*modifier
	for p.tok != EOF && p.tok != LINEND && p.tok != /* RBRACK */RBRACE {
		if m := p.modifier(ctx); m != nil { elems = append(elems, m) }
	}

	// p.expect(/* RBRACK */RBRACE)

	if len(elems) == 0 {
		errostack(ctx, 5, "empty modifier group").debug()
	}
	if p.tok == COLON {
		errostack(ctx, 5, "unexpected colon after modifer").debug()
	}
    return &modification{valbase{ctx.Position()}, elems }
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
// Similar to makefile automatic variables, see
//   * https://www.gnu.org/software/make/manual/html_node/Automatic-Variables.html#Automatic-Variables
var rule_autos = map[string]struct{}{
	"@" :struct{}{}, "%" :struct{}{}, "<" :struct{}{}, ">" :struct{}{}, "?" :struct{}{}, "^" :struct{}{}, "+" :struct{}{}, "|" :struct{}{}, "*" :struct{}{},
	"@D":struct{}{}, "%D":struct{}{}, "<D":struct{}{}, ">D":struct{}{}, "?D":struct{}{}, "^D":struct{}{}, "+D":struct{}{}, "|D":struct{}{}, "*D":struct{}{},
	"@F":struct{}{}, "%F":struct{}{}, "<F":struct{}{}, ">F":struct{}{}, "?F":struct{}{}, "^F":struct{}{}, "+F":struct{}{}, "|F":struct{}{}, "*F":struct{}{},
	"@'":struct{}{}, "%'":struct{}{}, "<'":struct{}{}, ">'":struct{}{}, "?'":struct{}{}, "^'":struct{}{}, "+'":struct{}{}, "|'":struct{}{}, "*'":struct{}{},
	"-" :struct{}{},
	"~" :struct{}{},
}

func (p *parser) rule(ctx Context, optvals, targets []Value) (result Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "rule")) }

	ctx = parser_rule_context{at(ctx, p)}

	defer trace(ctx)

	var proj = get_project(ctx)
	if proj.keyword == PACKAGE {
		erro(ctx, "rules forbidden in package : %v", targets).debug()
		return
	}
    if proj != get_scope(ctx).project {
		erro(ctx, "mismatched project/scope : %v", targets).debug()
		return
	}

	// TODO: doc = p.leadComment
	var depends, ordered, recipes []Value
	var position = ctx.Position()
	var l = _loader(ctx)
	defer l.closescope(l.openscope(fmt.Sprintf("rule %v", targets)))
	defer func() {
		// Close the rule scope and go back to project scope. The current
		// scope must be project scope befor Rule.
		p.configure, p.dialect, p.ruparas = false, "", nil
	} ()

	p.dialect = ""
	p.ruparas = nil

	// NOTE: expand targets to speed up for later usage, it might spend lots of time in
	// project.entry while matching for entry looked up if not expanded right now.
	targets = expand(ctx, targets...)

	defer func(t []Value) { p.targets = t } (p.targets)
	p.targets = targets // save targets for later refering
	p.next(ctx, true) // skip rule delimeters and spaces

	if p.tok != SEMICOLON && p.tok != BAR && !p.isEndOfLine() {
		depends = p.depends(ctx, true)
	}
	if p.tok == BAR { // '|' starts the ordered prerequisites
		if p.next(ctx, true); p.tok != SEMICOLON && !p.isEndOfLine() {
			ordered = p.depends(ctx, false)
		}
	}

	if p.tok == SEMICOLON { // ;
		// Parse inline recipe in the program scope.
		recipes = append(recipes, p.recipe(ctx))
	} else /*if p.tok == LINEND || p.lineComment != nil*/ {
		// Parse recipes in the program scope.
		p.scanner.recipes(true) // Turn on recipes before LINEND.
		if p.linend() { // Take the new line.
			for p.tok != EOF && p.isRecipeStart() {
				recipes = append(recipes, p.recipe(ctx))
			}
		}
		p.scanner.recipes(false)
	}

	if t := targets[0]; p.configure {
		d, a := proj.set(ctx, t, defVoid)
		if d == nil && a == nil {
			erro(at(ctx,t), "configure target '%v' not defined", t)
		} else if a == nil {
			// ...
		} else if _, y := a.(*def); !y {
			erro(at(ctx,t), "configure target '%v' already taken: %v", t, ts(a))
		}
		if d != nil && !d.position.IsValid() { d.position = t.Position() }
	}

	// TODO: lang: 0,

    var prog = program{
        configure: p.configure,
        language:  p.dialect,
        params:    p.ruparas,
        position:  position,
        project:   proj,
		depends:   depends,
		ordered:   ordered,
        recipes:   recipes,
    }

	//targets = barefilize(ctx, targets...)
	//depends = barefilize(ctx, depends...)
	//ordered = barefilize(ctx, ordered...)
	if res := p.entries(at(ctx, position), &prog, targets, optvals); len(res) == 1 {
		result = res[0]
	} else if 1 < len(res) {
		result = _list_t[entry](res...)
	} else {
		result = makeNull(position)
	}
	return
}

func (p *parser) entries(ctx Context, prog *program, targets, options []Value) (res []entry) {
	defer trace(ctx)
    for _, target := range targets {
        var ctx = at(ctx, target)

        if isTrivial(target) {
            if true { continue }
			erro(ctx, "trivial target; %v", targets).debug()
			return
        }

        var entry = prog.project.entry(ctx, options, target, prog)
        if entry == nil {
            erro(ctx, "creating entry failed for %v", target).debug()
            return
        }

		res = append(res, entry)

        if x, y := entry.destiny().(flag); y && x.Value != nil {
			if prog.project.name != "~" {
				var s = x.Value.string(ctx)
				_universe(p.Context).globe.AddFlagEntry(s, entry)
			}
        } else if p.configure {
            if entry.patterned(ctx) {
                erro(ctx, "unsupported pattern configures: %v", target).debug()
                return
            }
            prog.project.configs = append(prog.project.configs, entry)
        }
    }
    return
}

var pprofCounter int

func (p *parser) def(ctx Context) {
	defer trace(ctx)

	p.spaces(ctx)
	p.expect(DEF)
	p.spaces(ctx)

	var args []Value
	var name = p.expr(ctx)
	if a, y := name.(*argumented); y {
		name, args = a.Value, a.args
	}

	t := &template{
		pos: p.pos, tok: p.tok, lit: p.lit,
		state: p.scanner.scanstate,
		name: name, params: args,
	}

	p.spaces(ctx)
	p.linend()

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
		p.linend()

		state := p.scanner.scanstate
		t.end, t.endPos = &state, pos
		p.templates = append(p.templates, t)
		return

	case EOF:
		return
	}}
}

func (p *parser) foreach(ctx Context) {
	defer trace(ctx)

	if p.spaces(ctx); p.tok == LINEND {
		erro(at(ctx, p), "unexpected end of line").debug()
		return
	}

	p.expect(FOREACH)
	p.spaces(ctx)

	var params = p.values(ctx)
	var t = &template{
		pos: p.pos, tok: p.tok, lit: p.lit,
		state: p.scanner.scanstate,
	}

	p.spaces(ctx)
	p.linend()

	var nested = 0
	for p.tok != EOF { switch pos := p.pos; p.tok {
	case FOREACH:
		p.next(ctx, true) // foreach
		nested += 1

	case DONE:
		if nested > 0 { nested -= 1 ; continue }

		p.next(ctx, true) // done
		p.linend()

		state := p.scanner.scanstate
		t.end, t.endPos = &state, pos

		defer func(s Pos) { p.stop = s } (p.stop)
		p.stop = t.endPos

		var a = map[string]Value{ "_" : nil }
		for _, elem := range xmerge(final{ctx}, params...) {
			if indeterminate(ctx, elem) {
				erro(ctx, "indeterminate param: %v", ts(elem)).debug()
			} else if isTrivial(elem) {
				if false { info(ctx, "trivial: %v", ts(elem)).debug() }
			} else {
				a["_"] = elem
				p.codeblock(ctx, t, a)
			}
		}
		return

	default:
		for p.tok != EOF {
			if p.next(ctx, true); p.tok == LINEND { p.next(ctx, true) ; break }
		}
	}}
}

func (p *parser) for_(ctx Context) {
	defer trace(ctx)

	if p.spaces(ctx); p.tok == LINEND {
		erro(at(ctx, p), "unexpected end of line").debug()
		return
	}

	var opts struct {
		skipNil bool `skip-nil,skip-null,skipnil,skipnull,no-nil,no-null`
		loose bool `loose`
	}

	if p.expect(FOR); p.tok == LPAREN {
		p.next(ctx, true) // LPAREN
		if vals := parseOpts(ctx, &opts, p.values(ctx)...); vals != nil {
			erro(at(ctx, vals[0]), "unexpected opts: %v", vals).debug()
		}
		p.expect(RPAREN)
	}

	p.spaces(ctx)

	type param struct {
		name string
		elems []Value
	}

	type nparam struct {
		p Position
		a []*param
		n int
	}

	var params []*nparam
	var vars = map[string]Value{}
	for p.spaces(ctx); p.tok != EOF && !p.isEndOfLine(); p.spaces(ctx) {
		if p.tok == AND && params == nil {
			erro(at(ctx, p), "unexpected 'and'").debug()
			continue
		} else if p.tok == AND || params == nil {
			if params = append(params, &nparam{p:p.Position()}); p.tok == AND {
				p.next(ctx, true) // and
				continue
			}
		}

		var _v = params[len(params)-1]
		for _, a := range xmerge(at(ctx, p), p.expr(ctx)) {
			var elems []Value
			var s string

			if x, y := a.(*pair); !y {
				erro(at(ctx,a), "unexpected value: %v", ts(a)).debug()
				return
			} else if s = x.key.string(at(ctx, x.key.Position())); s == "" {
				erro(at(ctx,a), "empty key: %v", ts(x.key)).debug()
				return
			} else if g, y := x.val.(*group); y {
				elems = g.elems
			} else {
				elems = append(elems, x.val)
			}

			// Make sure all elements are expanded.
			elems = xmerge(at(ctx, a), elems...)

			if _, y := vars[s]; y {
				erro(at(ctx, a), "duplicated key: %v", s).debug()
				return
			} else {
				vars[s] = &null{valbase{a.Position()}}
			}

			if n := len(elems); n > _v.n { _v.n = n }

			_v.a = append(_v.a, &param{s, elems})
		}
	}

	var t = &template{
		pos: p.pos, tok: p.tok, lit: p.lit,
		state: p.scanner.scanstate, // verb: "for",
	}

	p.spaces(ctx)
	p.linend()

	var nested = 0
	for p.tok != EOF { switch pos := p.pos; p.tok {
	case FOR:
		p.next(ctx, true) // for
		nested += 1

	case DONE:
		if nested > 0 { nested -= 1 ; continue }

		p.next(ctx, true) // done
		p.linend()

		defer func(s Pos) { p.stop = s } (p.stop)

		state := p.scanner.scanstate
		t.end, t.endPos, p.stop = &state, pos, pos

		var num int
		for _, _v := range params {
			if _v.n > 0 { if num == 0 { num = _v.n } else { num *= _v.n } }
		}

		var l int = len(params)-1
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
				for k, t := range params { if t.n == 0 {
					if true { continue outer }
				} else if k <= _i {
					if 0 < _i { i %= t.n }
				} else {
					if _i < l { i /= t.n }
				}}

				for _, v := range _v.a { if i < len(v.elems) {
					vars[v.name] = v.elems[i]
				} else {
					vars[v.name] = &null{valbase{_v.p}}
					if opts.skipNil { continue outer }
				}}
			}

			var trivial bool = len(vars) == 0
			if !trivial { for _, v := range vars {
				if trivial = isTrivial(v); !trivial { break }
			}}
			if !trivial { p.codeblock(ctx, t, vars) }
		}
		return

	default:
		for p.tok != EOF {
			if p.next(ctx, true); p.tok == LINEND { p.next(ctx, true) ; break }
		}
	}}
}

func (p *parser) codeblock(ctx Context, t *template, vars map[string]Value) {
	p.pos, p.tok, p.lit, p.scanner.scanstate = t.pos, t.tok, t.lit, t.state

	defer trace(ctx)

	if false {
		pprofCounter += 1
		defer startCPUProfile(ctx, fmt.Sprintf("template-%05d.prof", pprofCounter), true)()
	}

	if !(p.pos < p.stop) {
		erro(at(ctx,p.loc(p.pos)), "bad range: [%v %v) (%v)", p.pos, p.stop, t.name).debug(10)
	}

	var c = parser_code_context{automatic{Context:ctx}}
	c.suppress, c.defs = c.has, make(auto_defs)

	if  _, y := vars["_"]; !y { vars["_"] = nil }

	for s, v := range vars {
		d, _ := c.set(&c, s, v)
		d.origin = defCodeBlockAuto
	}

	for p.tok != EOF && p.pos < p.stop {
		if p.tok == SPACE || p.tok == LINEND || (p.tok == COMMENT && p.lineComment != nil) {
			p.next(ctx, true)
		} else {
			p.clause(&c)
		}
	}
}

func (p *parser) repeat(ctx Context, t *template, params []Value) {
	defer func(t time.Time, pos Pos, tok token, lit string, state scanstate) {
		if u := _universe(ctx); u.ddd == "template.repeat" {
			// dont check time in ddd mode
		} else if d := time.Now().Sub(t); d > u.slow {
            warnstack(ctx, 3, "slow: %v, prof-%d", d, pprofCounter).debug()
        }

		if _diagnostic(ctx).error() { erro(ctx, "template errors").debug() }

		p.pos, p.tok, p.lit, p.scanner.scanstate = pos, tok, lit, state
	} (time.Now(), p.pos, p.tok, p.lit, p.scanner.scanstate)

	// TODO: parseOpts(params) -> add option to turn off asFile in Context

	if false { pprofCounter += 1
		var (
			profCpu = fmt.Sprintf("template-%05d.cpu.prof", pprofCounter)
			profMem = fmt.Sprintf("template-%05d.mem.prof", pprofCounter)
			fCpu *os.File
			e error
		)
		if fCpu, e = os.Create(profCpu); e != nil {
			erro(ctx, "%T: %v", e, e).debug()
			return
		} else if e = pprof.StartCPUProfile(fCpu); e != nil {
			erro(ctx, "%v: %v", profCpu, e).debug()
			fCpu.Close() ; return
		}
		defer func() {
			pprof.StopCPUProfile()
			fCpu.Close()

			var fMem, e = os.Create(profMem)
			if e != nil { erro(ctx, "%v", e).debug() }

			runtime.GC() // update memory statistics
			e = pprof.WriteHeapProfile(fMem)
			fMem.Close()

			if e != nil { erro(ctx, "%v: %v", profMem, e).debug() }
		} ()
	}

	var m = map[string]Value{}

	for i, v := range t.params { if s := v.string(ctx); s != "" {
		if i < len(params) { m[s] = params[i] } else {
			m[s] = makeNull(v.Position())
		}
	} else {
		erro(at(ctx,v), "empty template param name: %v %v", v, v).debug()
	}}

	p.codeblock(ctx, t, m)
}

func (p *parser) call(ctx Context, name Value, args []Value) (result bool) {
	ctx = at(ctx, p)

	for _, t := range p.templates { if t.name != nil && eq(ctx, t.name, name) {
		stop := p.stop
		p.stop = t.endPos
		p.repeat(ctx, t, args)
		p.stop = stop
		return true
	}}

	erro(ctx, "undefined template: %v", name).debug(3)
	return
}

func (p *parser) clause(ctx Context) {
	if l_traverse.enabled { defer un(l_tracef(l_traverse, "clause(%v, %v)", p.tok, p.pos)) }

	var x Value
	var tok = p.tok // TODO: allow assigns like: `eval := xxx`

	defer trace(ctx)
	defer func() {
		if debugSyntax(ctx, "clause") {
			warn(ctx, "clause: %v %v ; %v %v", typeof(x), x, p.tok, p.lit).debug(6)
		}
	} ()

	switch p.spaces(ctx); tok {
	case  INCLUDE: p.spec(ctx, tok, p.expect(tok), p.include); return
	case    FILES: p.spec(ctx, tok, p.expect(tok), p.files); return
	case   ASSERT: p.spec(ctx, tok, p.expect(tok), p.assert); return
	case   APPEND: p.spec(ctx, tok, p.expect(tok), p.append); return
	case     EVAL: p.spec(ctx, tok, p.expect(tok), p.eval); return
	case      DEF: p.def(ctx); return
	case      FOR: p.for_(ctx); return
	case  FOREACH: p.foreach(ctx); return
	case   LINEND, SPACE: p.next(ctx, true) // skip empty lines
	case USE, TEMPLATE:
		erro(ctx, "`%v` unexpected here", p.tok).debug()
		return
	}

	x = p.expr(parser_left_context{ctx})

	if p.spaces(ctx); p.tok.isAssign() {
		if debugSyntax(ctx, "define") {
			note(p, "parser.clause: %v; %v %v", ts(x), p.tok, p.lit).debug()
			flush(ctx)
		}
		p.assign(ctx, x)
		return
	}

	if p.tok.isRuleDelim() {
		if debugSyntax(ctx, "rule") {
			note(p, "parser.clause: %v; %v %v", ts(x), p.tok, p.lit).debug()
			flush(ctx)
		}
		p.rule(ctx, nil, []Value{x})
		return
	} else if a, y := x.(*argumented); y {
		p.call(ctx, a.Value, a.args)
		return
	}

	if vals := p.values(ctx, x); p.tok != EOF {
		return
	} else if strings.HasSuffix(p.scanner.file.Name(), pathSep+configuration_sm) {
		if false { note(ctx, "%v (kit=%s)", p.tok, p.lit).debug() }
	} else if can(ctx, getParseIsConf{}) {
		note(ctx, "bad clause: %v (kit=%s) after %v", p.tok, p.lit, vals).debug(3)
	} else {
		erro(ctx, "bad clause: %v (lit=%s) after %v", p.tok, p.lit, vals).debug()
	}
}

func (p *parser) setDefaultVars(ctx Context, filename, abs, rel, tmp string) (res bool) {
	var s = get_scope(ctx)
	if s == nil {
		erro(ctx, "invalid scope").debug()
		return
	}

	if _loader(ctx).mode&Flat == 0 {
		s.set(ctx, ".",   defVoid, _pathstr(ctx, rel))
		s.set(ctx, "/",   defVoid, _pathstr(ctx, abs))
		s.set(ctx, "CWD", defVoid, _pathstr(ctx, abs)) // Current Work Directory, TODO: make it $:cwd:
		s.set(ctx, "CTD", defVoid, _pathstr(ctx, tmp)) // Current Temp Directory, TODO: make it $:ctd:
		return true
	}

	if d := s.findDef("/");   d == nil {
		erro(ctx, "/ not in the scope: %v", s.comment).debug()
		return
	}
	if d := s.findDef(".");   d == nil {
		erro(ctx, ". not in the scope: %v", s.comment).debug()
		return
	}
	if d := s.findDef("CTD"); d == nil {
		erro(ctx, "CTD not in the scope: %v", s.comment).debug()
		return
	}
	if d := s.findDef("CWD"); d == nil {
		erro(ctx, "CWD not in the scope: %v", s.comment).debug()
		return
	}
	return true
}

type project_opt struct {
	configure Value `conf,configure` // detects dotConfigure if empty
	noDock bool `nodock,no-dock` // don't load container project
    traveUseLoop bool `break,loop` // don't recursively use this project
    multiUseAllowed bool `multi`  // this project is used multiple times
	final bool `final` // no bases
}

func isEntryFileName(s string) bool { return filepath.Base(s) == entryFileName }

func (p *parser) file(ctx Context) *parsedFile {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "file '"+p.scanner.file.Name()+"'")) }
	if _universe(ctx).traceLaunch { defer un(l_trace(l_launch, "parser.file")) }
	if _diagnostic(ctx).error() { return nil }

	defer trace(ctx)

	var (
		ident *barecomp
		identStr string
		implicitBase string // aka. foo.bar.Baz implicitly load base 'foo/bar'
		abs, rel, tmp string
		position = ctx.Position()
		keyword  = p.tok
		filename = p.scanner.file.Name()
		isMainFile = isEntryFileName(filename)
	)

	var l = _loader(ctx)
	defer l.closescope(l.openscope(fmt.Sprintf("file %s", filename)))

	if l.mode&Flat != 0 {
		abs = get_project(ctx).absPath
	} else {
		abs = filepath.Dir(filename)
	}

	rel, _ = filepath.Rel(_workdir(l), abs)
	tmp = joinTmpPath(ctx,_workdir(l), rel)

	if !p.setDefaultVars(ctx, filename, abs, rel, tmp) { return nil }

	switch position = p.Position(); keyword {
	case PACKAGE, MODULE:
		erro(ctx, "deprecated keyword: %s", keyword).debug()
		return nil
	case CONFIGURE:
		switch p.next(ctx, true); p.tok {
		case DOT:
			if err := l.ParseConfigDir(abs, abs); err != nil {
				erro(ctx, "parsing configure directory failed, '%s': %v", abs, err)
			} else {
				p.next(ctx, true) // skip the '.' token and consequence spaces
			}

			var basename = filepath.Base(filepath.Dir(filename))
			ident = makeBarecomp(makeBareword(position, basename))

		default:
			erro(ctx, "unknown configuration '%v', currently only 'configure .' is supported", p.tok)
		}
	case PROJECT:
		if l.mode&Flat != 0 { erro(ctx, "forbidden `%v` in flat file", p.tok) }

		p.next(ctx, true)

		var ( // Options are flag or *pair of a flag.
			opts project_opt
			optVals []Value
			pos Position
		)
		for p.tok == MINUS {
			var opt = p.expr(ctx);  p.spaces(ctx)
			optVals = append(optVals, opt)
			if !pos.IsValid() { pos = opt.Position() }
		}
		if !pos.IsValid() { pos = p.Position() }
		if a := parseOpts(ctx, &opts, optVals...); len(a) > 0 {
			for _, v := range a { erro(at(ctx,v), "unknown option %v", ts(v)).debug() }
			return nil
		}

		var g = _universe(ctx).globe
		var linfo = g.loads[len(g.loads)-1]

		// Smart-lang spec:
		//   * the project clause is not a declaration;
		//   * the project name does not appear in any scope.
		if p.tok == LPAREN || p.tok == EOF || p.tok == LINEND || p.lineComment != nil {
			var dir = filepath.Dir(filename)
			if linfo.loadee != nil && linfo.absDir == dir {
				ident = makeBarecomp(makeBareword(position, linfo.loadee.name))
			} else if name := filepath.Base(filename); name == dotBase || name == dotConfigure {
				// NOTE: loading the .base or .configure file
				ident = makeBarecomp(makeBareword(position, name))
			} else if base := filepath.Base(dir); base != "" {
				// TODO: validate basename as a valid identifier
				ident = makeBarecomp(makeBareword(position, base))
			} else {
				erro(ctx, "invalid file: %v", filename).debug()
			}
		} else if p.tok == TILDE {
			/*if filename == confinitFilename {
                ident = &ast.Bareword{ ValuePos:pos, Value:"~" }
            } else*/ if ext := filepath.Ext(filename); ext != ".smart" {
				erro(p, "`%v` not a smart file", filepath.Base(filename)).debug()
			} else if s := strings.TrimSuffix(filepath.Base(filename), ext); s != "" {
				ident = makeBarecomp(makeBareword(position, s))
			} else {
				erro(p, "`%v` not tilde name", filepath.Base(filename)).debug()
			}
			p.next(ctx, true) // skip tilde
		} else {
			base := makePath()
			ident = makeBarecomp()
			for p.tok != EOF && p.tok != SPACE {
				var w = p.bare(ctx)
				if ident = ident.suffix(ctx, w).(*barecomp); p.tok == DOT {
					t := &punctuation{valbase{p.Position()}, p.tok}
					// TODO: parse to Qualiword
					ident = ident.suffix(ctx, t).(*barecomp)
					base.elems = append(base.elems, w)
					p.step() // '.'
				} else { break }
			}
			if p.spaces(ctx); len(ident.elems) == 0 {
				// erro(ctx, "package name is empty (tok=%v %v)", t, p.tok).debug()
				// return nil
			} else if 0 < base.len() {
				implicitBase = base.string(ctx)
			}
		}

		if identStr = ident.string(ctx); linfo.loadee != nil && identStr != linfo.loadee.name {
			warn(at(ctx,ident), "%s: declare multiple project in the same directory", get_project(ctx)).debug(24)
		} else if identStr == "_" && l.mode&DeclarationErrors != 0 {
			erro(at(ctx,ident), "package name '_' is preserved").debug()
			return nil
		}

		// Don't bother parsing the rest if we had errors parsing the package clause.
		if n := _diagnostic(l.Context).countError(); n > 0 {
			erro(p, "got %d errors parsing file: %s", filename).debug()
			return nil
		}

		var _, declared = linfo.declares[identStr]
		if (l.mode&Flat == 0) && l.declare(at(ctx, ident), keyword, ident, identStr, &opts) {
			// Change the 'default' owners into the new declared project
			if s := get_scope(ctx); s != nil {
				if d := s.findDef("."  ); d != nil { d.scope = s }
				if d := s.findDef("/"  ); d != nil { d.scope = s }
				if d := s.findDef("CTD"); d != nil { d.scope = s }
				if d := s.findDef("CWD"); d != nil { d.scope = s }
			} else {
				erro(ctx, "file scope is nil").debug()
			}

			// NOTE: do.smart is always the first loaded, so the loadee will be pointed to it
			if linfo.loadee == nil { linfo.loadee = get_project(ctx) }
			defer l.closeCurrent(ident, identStr)

			isMainFile = isMainFile && !declared;
		}

		var basePos Position
		if implicitBase != "" { basePos = pos } else { basePos = p.Position() }
		if p.tok == LPAREN {
			for p.tok != EOF {
				for p.next(ctx, true); !p.isEndOfList(ctx); {
					p.spaces(ctx)

					ctx := at(ctx, p)
					param := p.expr(parser_group_context{token_aware_context{ctx,COMMA}})
					p.spaces(ctx)

					//if p.lineComment != nil  { break }
					//if p.tok == LINEND { break }
					if p.tok == EOF {
						erro(ctx, "unexpected end of file while parsing bases").debug()
						return nil
					}

					vals := parseOpts(ctx, &opts, param)
					if opts.final || keyword == PACKAGE { continue }
					if !l.bases(ctx, linfo, "", merge(vals...)...) {
						erro(ctx, "load bases failed: %v", vals).debug()
						return nil
					}
				}
				if p.tok != COMMA { break }
			}
			p.expect(RPAREN)
		} else if !l.bases(ctx, linfo, implicitBase) { // for special bases, e.g. .base
			erro(at(ctx,basePos), "loading bases failed").debug()
			return nil
		}

		if p.spaces(ctx); p.tok != EOF { p.linend() }
		if keyword != PACKAGE {
			l.configure(ctx, linfo, ident, identStr, declared)
			if !opts.noDock { l.container(ctx, ident, identStr) }
		}
	case EOF:
		return nil
	default:
		if l.mode&Flat == 0 {
			p.expected(p.pos, "configure, project, module or package keyword")
		}
	}

	var auto = (l.mode&Flat == 0) && isMainFile //&& isEntryFileName(filename)
	if auto { l.autoload(at(ctx, p), "declared") }
	if l.mode&ModuleClauseOnly == 0 {
		if l.mode&Flat == 0 { ForDeclare: for p.tok != EOF {
			switch tok := p.tok; tok {
			case USE: p.spec(ctx, tok, p.expect(tok), p.use)
			case LINEND, SPACE: p.next(ctx, true) // skip empty lines
			case ASSERT, EVAL, FILES, INCLUDE: p.clause(ctx)
			default: break ForDeclare
			}
		}}

		if false && auto { l.autoload(at(ctx, p), "amid") }

		if l.mode&ImportsOnly == 0 { // rest of module body
			for /* !_diagnostic(p.Context).error() && */ p.tok != EOF {
				if p.tok == LINEND || (p.tok == COMMENT && p.lineComment != nil) {
					p.next(ctx, true)
				} else if p.clause(at(ctx, p)); flush(ctx) > 0 {
					break
				}
			}
		}
	}
	if auto { l.autoload(at(ctx, p), "appendix") }

	if  _universe(ctx).ddd == "parser.files" {
		_universe(ctx).ddd = ""
	}

	return &parsedFile{
		// TODO: doc: doc,
		// TODO: comments: p.comments,
		keyword:  keyword,
		position: position,
		name:     ident,
		scope:    get_scope(ctx),
		use:      p.imports,
	}
}
