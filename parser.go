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
	// "reflect"
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

	comments  []*commentGroup
	leadComment *commentGroup // last lead comment
	lineComment *commentGroup // last line comment

	templates []*template

	imports []*use_spec // list of imports

	targets []Value // targets of current rule
	ruparas []*auto // parameters of current rule
	dialect  string // recipe dialect of current rule
	configure  bool // is parsing configure program?

	dd bool // helps debug parsing via `eval -dd=true{}`
}

type (
	getParseAware      struct{ token }
	getParseCanParams  struct{}
	getParseCanUndef   struct{}
	getParseGlob       struct{}
	getParseIncOpts    struct{}
	parse_is_auto     struct{ string }
	parse_is_conf     struct{}
	getParseIsFlag     struct{}
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

type parse_auto_context     struct { Context }
type parse_bare_context     struct { Context }
type parse_braced_context   struct { Context }
type parse_call_context     struct { Context }
type parse_code_context     struct { automatic }
type parse_defvalue_context struct { Context }
type parse_foreach_context  struct { Context }
type parse_glob_context     struct { Context }
type parse_group_context    struct { Context }
type parse_include_context  struct { Context ; o includeOpts }
type parse_left_context     struct { Context }
type parse_modifier_context struct { Context }
type parse_params_context   struct { Context }
type parse_path_context     struct { Context }
type parse_perc_context     struct { Context }
type parse_recipe_context   struct { Context ; builtin bool }
type parse_regex_context    struct { Context }
type parse_rule_context     struct { Context }
type parse_undef_context    struct { Context }

func (p parse_glob_context) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case getParseGlob: return true
	}
	return p.Context.do(ctx, op)
}

func (p parse_params_context) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case getParseCanParams: return true
	}
	return p.Context.do(ctx, op)
}

func (p parse_auto_context) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case parse_is_auto: return true
	}
	return p.Context.do(ctx, op)
}

func (p parse_defvalue_context) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case parse_is_auto: return IsDigits(t.string)
	}
	return p.Context.do(ctx, op)
}

func (p parse_foreach_context) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case parse_is_auto: if t.string == "_" { return true }
	}
	return p.Context.do(ctx, op)
}

func (p parse_rule_context) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case parse_is_auto:
		if IsDigits(t.string) { return true }
		if _, y := rule_autos[t.string]; y { return true }
	}
	return p.Context.do(ctx, op)
}

func (p parse_recipe_context) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case getParseIsRecipe: return t.bool == p.builtin
	}
	return p.Context.do(ctx, op)
}

func (p parse_include_context) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case getParseIncOpts: return &p.o
	case parse_is_conf : return p.o.isConfig
	case getParseIsFlag : return true
	}
	return p.Context.do(ctx, op)
}

func (p parse_left_context) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case getParseLeftHandSide: return true
	}
	return p.Context.do(ctx, op)
}

func (p parse_undef_context) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case getParseCanUndef: return true
	}
	return p.Context.do(ctx, op)
}

func (p *parser) ts(t string) string {
	return fmt.Sprintf("{=%s %v %s}", t, p.tok, p.scanner.file.Name())
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

	p.pos, p.tok, p.lit = p.scanner.scan()
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

	// if p.dd {
	// 	var t = warn(ctx, "%v %v %v", p.tok, p.lit, p.scanner.scanstate)
	// 	if p.tok == COMPOUND { t.debug(12) }
	// 	if p.tok == LINEND { t.debug(24) }
	// 	flush(ctx)
	// }
}
func (p *parser) next(ctx Context, ws bool) { if p.step(); ws { p.spaces(ctx) } }
func (p *parser) spaces(ctx Context) {
	for p.lineComment == nil && p.tok != EOF {
		if p.tok == SPACE || (p.tok == RECIPE && truly(ctx, getParseIsRecipe{true})) {
			p.step()
		} else if p.tok == ESCAPE && p.lit == "\n" {
			if p.step(); p.tok == LINEND || p.lineComment != nil { break }
			if truly(ctx, getParseIsRecipe{true}) {
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

func (p *parser) expected(ctx Context, pos Pos, msg string, a... any) {
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

	erro(at(ctx, p.loc(pos)), msg).trace()
}

func (p *parser) expect(ctx Context, tok token) Pos {
	var pos = p.pos
	if p.tok != tok && !(tok == LINEND && p.lineComment != nil) {
		p.expected(ctx, pos, "'"+tok.String()+"'")
	}
	p.step() // move forward
	return pos
}

func (p *parser) linend(ctx Context) (ok bool) {
	if p.lineComment != nil {
		p.lineComment, ok = nil, true
	} else if p.tok == EOF {
		ok = true
	} else if p.tok == LINEND {
		p.step(); ok = true
	} else {
		p.expected(ctx, p.pos, "'\\n'")
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
	var tok, lit, pos = p.tok, p.lit, _position(ctx)
	p.step()

	if tok != BAREWORD && lit == "" {
		lit = tok.String()
	}
	return makeBareword(pos, lit)
}

func (l unilo) braced(ctx Context) (x Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "braced")) }

	var p = l.p
	var pos = p.Position()

	ctx = parse_braced_context{at(ctx, pos)}

	p.expect(ctx, LBRACE)

	if p.tok == RBRACE {
		x = &null{p.valbase()}
		p.spaces(ctx)
		p.step() // consumes }
		return
	}

	if checkpoints {
		if p.tok != LPAREN && !p.scanner.bits.isBrace() {
			erro(at(ctx, p), "wrong scan state: %v, %v, %v", p.tok, p.lit, p.scanner.scanstate).trace()
		}
	}

	var typed token

	if p.tok == LBRACK { // OBSOLETE: {[...]}
		erro(at(ctx, p), "syntax error; for modification, use {(modifier)}").trace()
	} else if p.tok == LPAREN {
		x = l.modification(ctx)
		p.spaces(ctx)
		p.expect(ctx, RBRACE)
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
				if false { note(ctx, "%v %v %v", p.tok, p.lit, p.scanner.scanstate) }

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
			case FLOAT: x = makefloat(pos, 0.)
			case TRUE:  x = makeBoolean(pos, true)
			case FALSE: x = makeBoolean(pos, false)
			case YES:   x = makeAnswer(pos, true)
			case NO:    x = makeAnswer(pos, false)
			case ON:    x = makeOption(pos, true)
			case OFF:   x = makeOption(pos, false)
			case NONE:  x = makeNone(pos)
			case NULL:  x = makeNull(pos)
			default:
				erro(ctx, "expects braced value (%v)", typed).trace()
			}
			return
		}
	}

	switch typed {
	case BARE: // {=bare ... }
		x = p.bare(at(ctx, p))
		p.spaces(ctx)
		p.expect(ctx, RBRACE)
		return
	case GLOB: // {=glob ... }
		x = l.glob(at(ctx, p), nil)
		p.spaces(ctx)
		p.expect(ctx, RBRACE)
		return
	case REGEX: // {=regex ...}
		return p.regex(at(ctx, p))
	case FILE: // {=file ... }
		if v := l.expr(ctx); v != nil {
			var c = at(ctx, v)
			var s = v.string(c)
			var a = []any{stat_nonexist{true}}
			if !isAbsOrRel(s) { a = append(a, stat_dir{_project(ctx).absPath}) }
			x = stat(c, s, a...)
		}
		p.spaces(ctx)
		p.expect(ctx, RBRACE)
		return
	case PATH: // {=path ... }
		if v := l.expr(ctx); v != nil {
			if t, y := v.(*path); !y {
				x = l.path(ctx, v)
			} else {
				x = t
			}
		}
		p.spaces(ctx)
		p.expect(ctx, RBRACE)
		return
	case BIN, OCT, INT, HEX, FLOAT: // ={bin ...}, {=oct ...}, {=int ...}, {=hex ...}, {=float ...}
		if v := l.expr(ctx); v == nil {
			erro(ctx, "%s expects: %v, not %v %v", typed, RBRACE, p.tok, p.lit).trace()
		} else if p.spaces(ctx); p.tok == RBRACE {
			if p.step(); typed == FLOAT {
				return makefloat(pos, v.float(ctx))
			}
			switch n := v.int(ctx); typed {
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
			if t := l.expr(ctx); t != nil {
				v = t.true(ctx)
			} else {
				erro(at(ctx, p), "invalid expression").trace()
			}
		}
		p.spaces(ctx)
		p.expect(ctx, RBRACE)
		return &answer{boolean{valbase{pos},v}}
	case BOOL, BOOLEAN: // {=bool ...}, {=boolean ...}
		var v bool
		switch p.tok {
		case  TRUE, YES,  ON: v = true  ; p.next(ctx, true)
		case FALSE,  NO, OFF: v = false ; p.next(ctx, true)
		default:
			if t := l.expr(ctx); t != nil {
				v = t.true(ctx)
			} else {
				erro(at(ctx, p), "invalid expression").trace()
			}
		}
		p.spaces(ctx)
		p.expect(ctx, RBRACE)
		return &boolean{valbase{pos},v}
	case TRUE, FALSE: // {=true ...}, {=false ...}
		var v = l.expr(ctx).true(ctx)
		p.spaces(ctx)
		p.expect(ctx, RBRACE)
		return &boolean{valbase{pos},(typed == TRUE && v)}
	case YES, NO: // {=yes ...}, {=no ...}
		var v = l.expr(ctx).true(ctx)
		p.spaces(ctx)
		p.expect(ctx, RBRACE)
		return &answer{boolean{valbase{pos},(typed == YES && v)}}
	case ON, OFF: // {=on ...}, {=off ...}
		var v = l.expr(ctx).true(ctx)
		p.spaces(ctx)
		p.expect(ctx, RBRACE)
		return &option{boolean{valbase{pos},(typed == ON && v)}}
	case RAW:
		s := l.expr(ctx).string(ctx)
		p.spaces(ctx)
		p.expect(ctx, RBRACE)
		return &raw{valbase{pos},s}
	case UNDEF: // {=undef ...}
		x = undef{l.expr(ctx)}
		p.spaces(ctx)
		p.expect(ctx, RBRACE)
		return
	case NONE: // {=none ...}
		var v Value
		for ; p.tok != RBRACE && p.tok != EOF; p.spaces(ctx) {
			if t := l.expr(ctx); v == nil {
				v = t
			} else if l, y := v.(*list); y {
				l.elems = append(l.elems, t)
			} else {
				v = &list{elements{[]Value{v,t}}}
			}
		}
		p.expect(ctx, RBRACE)
		return &none{valbase{pos},v}
	case /* DISJUNCTION, */ 0: // {...}
		if v := l.values(ctx); len(v) == 0 {
			x = makeNull(pos)
		} else if len(v) == 1 {
			x = disjunction{v[0]}
		} else {
			x = disjunction{makeList(v...)}
		}
		p.spaces(ctx)
		p.expect(ctx, RBRACE)
		return
	default:
		erro(ctx, "%v", typed).trace()
	}
	return
}

func (l unilo) selector(ctx Context) (res Value) {
	res = l.expr(ctx)
	return
}

func (l unilo) parse_select(ctx Context, lhs Value) (res Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Select")) }

	var p = l.p
	var tok = p.tok // the arrow '->' or '=>'

	ctx = at(ctx, p)
	p.step() // skip '->' or '=>'

	switch t := lhs.(type) {
	case *selection:
		if v := t.expand(at(ctx, t)); isNull(v) {
			erro(ctx, "nil selection: %v", lhs).trace()
		} else {
			lhs = v
		}
	case *bareword:
        switch t.s {
        case "use", "usee", "goals", "os", "mode":
			erro(ctx, "$:%s: is obsoleted, use $(.$s) instead", t.s, t.s).trace()
        default:
            if o := l.resolve(ctx, t, t.s); false {
				erro(at(ctx,lhs), "resolve '%v' failed", lhs)
				erro(ctx, "parser is here (tok=%s)", tok)
				erro(at(ctx,p), "parser to go here (tok=%s, lit=%s)", p.tok, p.lit).trace()
            } else if !isNull(o) {
				lhs = o
			} else if tok == SELECT_PROG2 {
				res = makeNull(_position(ctx)) // ignore
				return
			} else {
				erro(at(ctx,lhs), "%v: '%v' is undefined (name=%v, obj=%v)", l.project, lhs, t, o)
				erro(ctx, "%v: parser is here (name=%s, tok=%s)", l.project, t.s, tok)
				erro(at(ctx,p), "%v: parser to go here (tok=%s, lit=%s)", l.project, p.tok, p.lit).trace()
            }
        }
    case *barecomp: // for cases like '.foo'
		name := lhs.string(ctx)
        if o := l.resolve(ctx, t, name); false {
			erro(at(ctx,lhs), "resolve selection object '%v' (%s) error", lhs, name).trace()
        } else if !isNull(o) {
			lhs = o
		} else if tok == SELECT_PROG2 {
			res = makeNull(_position(ctx)) // ignore
			return
		} else {
			erro(at(ctx,lhs), "'%v' is undefined", lhs).trace()
        }
	case *globpat:
		if o, y := optionalize(ctx, lhs); y { lhs = o } else {
			erro(at(ctx,lhs), "selection of '%v' is undefined", lhs).trace()
		}
	}

	if rhs := l.selector(ctx); isNull(rhs) {
		res = makeNull(_position(ctx))
	} else {
		if v, y := optionalize(ctx, rhs); y { rhs = v } // foo→bar?
		res = makeSelection(_position(ctx), tok, lhs, rhs)
	}

	if (p.tok == SELECT_PROP || p.tok == SELECT_PROG1 || p.tok == SELECT_PROG2) {
		res = l.parse_select(ctx, res) // Continue the selection recursivly.
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
	if p.lineComment != nil || p.tok.isListDelim() || (truly(ctx, getParseLeftHandSide{}) && p.tok.isAssign()) {
		return true
	}
	if truly(ctx, getParseIsRecipe{false}) && p.tok == RECIPE { // TODO: using p.isRecipeStart()
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
	var s = _scope(ctx)

	if checkpoints {
		if !strings.HasPrefix(s.comment, "rule ") {
			erro(ctx, "wrong scope for rule params: %s", s.comment).trace()
		}
	}

	for _, arg := range args {
		switch ctx := at(ctx, arg) ; arg.(type) {
		case *bareword, *barecomp:
			var a = s.auto(ctx, arg.string(ctx))
			s.alias(ctx, a, strconv.Itoa(len(p.ruparas)+1))
			p.ruparas = append(p.ruparas, a)
		default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
			erro(ctx, "bad parameter form (%v)", ts(arg)).trace()
		}
	}
	return
}

func (l unilo) depends(ctx Context, params bool) (res []Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "depends")) }

	var p = l.p
	for p.tok != BAR && p.tok != SEMICOLON && !p.isEndOfLine() {
		if p.tok == COLON {
			// FIXME: this check is not working!
			// FIXME: detects unexpected colon ':'
			erro(ctx, "unexpected colon").trace()
		} else if p.spaces(ctx) ; !p.isEndOfLine() {
			var val Value
			if len(res) == 0 {
				val = l.expr(parse_params_context{ctx})
			} else {
				val = l.expr(ctx)
			}

			if x, y := val.(*globpat); y && x.len() == 1 {
				if z, y := x.elems[0].(*globrange); y {
					note(at(ctx,val), "use {(modifier ...)} : %v", ts(z.Value)).debug()
				} else if z, y := x.elems[0].(*group); y {
					note(at(ctx,val), "use {(modifier ...)} : %v", ts(z.elems[0])).debug()
				} else {
					note(at(ctx,val), "use {(modifier ...)} : %v", ts(x.elems[0])).debug()
				}
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
func (l unilo) values(ctx Context, ii ...any) (values []Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Values")) }

	for _, i := range ii {
		switch v := i.(type) {
		case Value: values = append(values, v)
		default:
			erro(ctx, "unsupported value: %v", ts(i)).trace()
		}
	}

	var p = l.p
	for p.spaces(ctx); !p.isEndOfList(ctx); p.spaces(ctx) {
		var prev = p.pos
		if values = append(values, l.expr(ctx)); p.pos == prev {
			erro(at(ctx,p), "bad: %v %v; %v", p.tok, p.lit, values).trace()
		}

		// If there's a comment right after the parsed expression, we break
		// the expression list to treat the end-of-line comment like a LINEND.
		if p.tok == EOF || p.tok == LINEND || p.lineComment != nil { break }
	}
	return
}

func (l unilo) list(ctx Context, ii ...any) *list {
	return makeList(l.values(ctx, ii...)...)
}

func (l unilo) group(ctx Context) *group {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Group")) }

	p := l.p
	ctx = parse_group_context{token_aware_context{at(ctx, p),COMMA}}

	p.expect(ctx, LPAREN)
	p.spaces(ctx)

	var elems, converted = l.values(ctx), false
	for p.tok != RPAREN && p.tok != EOF {
		// if p.tok == COMMA { warn(ctx, "%020b: %v %v", p.bits, p.tok, p.lit).debug() }
		// if p.tok == COMMA { p.next(ctx, true) }
		switch p.tok {
		case BAR, COMMA, SEMICOLON:
			elems = append(elems, p.punctuation())
			p.spaces(ctx)
		}
		var next *list
		next = l.list(ctx)
		if !converted {
			elems = []Value{ makeList(elems...), next }
			converted = true
		} else {
			elems = append(elems, next)
		}
	}
	p.expect(ctx, RPAREN)
	return makeGroup(_position(ctx), elems...)
}

func (l unilo) argumentedExpr(ctx Context, x Value) *argumented {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "argumented")) }

	p := l.p
	ctx = parse_group_context{token_aware_context{at(ctx, p),COMMA}}

	p.next(ctx, true) // skip LPAREN

	var a = []Value{ l.list(ctx) }
	for p.tok != RPAREN && p.tok != LINEND && p.tok != EOF {
		switch p.tok {
		case COMMA: p.next(ctx, true) // skip COMMA
		case BAR, SEMICOLON:
			if false {
				a = append(a, p.punctuation())
				p.spaces(ctx)
			} else {
				erro(ctx, "unexpected punctuation: %v", p.tok).trace()
			}
		}
		a = append(a, l.list(at(ctx, p)))
	}
	p.expect(ctx, RPAREN)
	return makeArgumented(x, a...)
}

func (l unilo) globmeta(ctx Context) (x *globmeta) {
	pos, tok := l.p.Position(), l.p.tok
	l.p.step()
	return makeGlobMeta(pos, tok)
}

func (l unilo) globrange(ctx Context) (x *globrange) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "globrange")) }

	p := l.p
	ctx = at(ctx, p)
	p.expect(ctx, LBRACK) // skip '['

	chars := l.expr(ctx)
	p.expect(ctx, RBRACK) // skip ']'

	return makeGlobRange(chars)
}

func (l unilo) glob(ctx Context, x Value) (g *globpat) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "glob")) }

	p := l.p
	ctx = parse_glob_context{at(ctx, p)}

	if y := x == nil; y {
		g = &globpat{}
	} else if g, y = x.(*globpat); !y || g == nil {
		g = makeGlobPat(x)
	}

	for p.tok != RBRACE && p.tok != EOF && p.lineComment == nil {
		var v Value
		switch p.tok {
		case SELECT_PROP, SELECT_PROG1, SELECT_PROG2, PCON, RPAREN, COMMA, SPACE, LINEND, EOF:
			return
		case STAR, DAST, QUE:
			v = l.globmeta(ctx) // * ** ?
		case LBRACK:
			v = l.globrange(ctx) // [abc0-9xyz]
		default:
			v = l.expr(ctx)
		}
		g.elems = append(g.elems, v)
	}
	return
}

func (l unilo) perc(ctx Context, x Value) Value {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Perc")) }

	var p = l.p
	ctx = parse_perc_context{at(ctx, p)}

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
					erro(ctx, "too many %").trace()
				case PCON: // FIXES: %%/xxx -> Path(%% xxx)
					x = makePercpat(position, x, perc2)
					return l.path(ctx, x)
				case COLON,    DOLON,
					LPAREN,    RPAREN,
					LBRACK,    RBRACK,
					LBRACE,
					SEMICOLON, COMMA,
					SPACE,     LINEND:
				default:
					var (
						yy = l.expr(ctx)
						_, ok = yy.(*path)
					)
					if ok {
						erro(ctx, "incorrect: %v, %v", x, yy).trace()
					}
					assert(!ok, "the second part of aaa%%bbb/foo/bar parsed incorrectly as path")
					perc2.Suffix = yy
				}
			}
			y = perc2
		default:
			y = l.expr(ctx)
		}
	}
	return makePercpat(p.loc(pos), x, y)
}

func (p *parser) regex(ctx Context) (_ Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "regex")) }

	var rx string
	var pos = p.Position()

	ctx = parse_regex_context{at(ctx, p)}

	if checkpoints {
		if !p.scanner.bits.isBrace() {
			erro(ctx, "wrong scan state: %v", p.scanner.scanstate).trace()
		}
		if !p.scanner.bits.isBraceRaw() {
			erro(ctx, "wrong scan state: %v", p.scanner.scanstate).trace()
		}
	}

	for ; p.tok != RBRACE && p.tok != EOF; p.scan() {
		if p.tok == ESCAPE { rx += "\\" }
		rx += p.lit
	}

	p.expect(ctx, RBRACE)

	var err error
	var x = &regexpat{valbase{pos}, nil} // TODO: correct regexp pattern value
	if x.Regexp, err = regexp.Compile(rx); err != nil {
		erro(at(ctx,pos), "regex: %v", err).trace()
	}
	return x
}

func (l unilo) pair(ctx Context, x Value) *pair {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "pair")) }

	p := l.p
	ctx = at(ctx, p)
	p.step()

	var y Value
	if p.isEndOfList(ctx) {
		y = makeNull(_position(ctx))
	} else {
		y = l.expr(ctx)
	}

	return makePair(x, y)
}

func (l unilo) flag(ctx Context) flag {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "flag")) }

	var p = l.p
	ctx = at(ctx, p)
	p.step() // skip dash '-'

	var x Value
	// flag expressions, excluding "-)" "-]" "-}" "-\n", "-=", "-:", etc.
	if p.isEndOfLine() || p.isEndOfList(ctx) || p.tok == SPACE || p.tok == RECIPE {
		x = makeNull(_position(ctx))
	} else if false {
		x = l.expr(ctx)
	} else {
		x = l.unary(ctx)
		l: for p.tok == DOT || !(_operator_beg < p.tok && p.tok < _closure_beg) {
			switch p.tok {
			case COMMENT, HASH, SPACE, RECIPE, LINEND, EOF: break l
			case DELEGATE, CLOSURE: x = compose(ctx, x, l.unary(ctx))
			default: if p.tok.isClosure() || p.tok.isDelegate() {
				x = compose(ctx, x, l.unary(ctx))
			} else {
				break l
			}}
		}
	}
	if x == nil {
		erro(ctx, "nil flag name").trace()
	}
	return flag{x}
}

func (l unilo) negative(ctx Context) negative {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "negative")) }
	l.p.expect(ctx, EXC)
	return negative{l.expr(ctx)}
}

func (p *parser) punctuation() *punctuation {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "punctuation")) }
	var vb, tok = p.valbase(), p.tok
	p.step()
	return &punctuation{vb, tok}
}

func (p *parser) escape(ctx Context) (v Value) {
	var vb, lit = p.valbase(), p.lit
	p.expect(ctx, ESCAPE)
	return &escaped{vb, lit}
}

func (p *parser) literal(ctx Context) (v Value) {
	var tok, lit = p.tok, p.lit
	ctx = at(ctx, p)
	p.step()

	// ESCAPE is handled in value.EscapeChar
    switch position := _position(ctx); tok {
    case BAR:
		erro(ctx, "`|` is deprecated, changed the modifiers!")
    case BINARY:      v = ParseBinary(position, lit)
    case OCTAL:       v = ParseOctal(position, lit)
    case INTEGER:     v = ParseDecimal(position, lit)
    case HEXADECIMAL: v = ParseHexadecimal(position, lit)
    case FLOATING:    v = parseFloat(position, lit)
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

func (l unilo) compound(ctx Context) *compound {
	var elems []Value

	p := l.p
	p.step()

	for p.tok != EOF && p.tok != COMPOSED && p.tok != LINEND {
		if p.tok == RAW {
			elems = append(elems, p.literal(ctx))
		} else {
			elems = append(elems, l.expr(ctx))
		}
	}
	p.expect(ctx, COMPOSED)
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
func (l unilo) dot(ctx Context, x Value) (res Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Dot")) }

	var p = l.p
	ctx = token_aware_context{at(ctx, p),DOT}

	for !p.isEndOfDotConcat(ctx) {
		x = compose(ctx, x, l.composite(ctx))
		if p.tok == DOT /*&& comp.End() == p.pos*/ {
			x = compose(ctx, x, &punctuation{p.valbase(), p.tok})
			p.step() // skips '.'
		}
	}

	return x
}

func (l unilo) path(ctx Context, start Value) (res *path) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Path")) }

	if start == nil {
		erro(ctx, "nil path starter").trace()
	}

	ctx = parse_path_context{at(ctx, start)}

	switch t := start.(type) {
	case     *path: res = t
	case   *strlit: res = makePath(splitPathStr(ctx, t.s)...)
	case *compound: res = makePath(splitPathStr(ctx, t.string(ctx))...) // FIXME: dont final here
	default:        res = makePath(start)
	}

	var p = l.p
	for p.tok == PCON && p.tok != EOF {
		// skips repeated '/' sequence
		for p.step(); p.tok == PCON; p.step() {}

		switch p.tok {
		case LPAREN, LBRACE, RPAREN, RBRACE, COMMA, SPACE, LINEND:
			res.elems = append(res.elems, _pathpun(at(ctx, p), PTAIL)) // after the last '/'
			return
		}

		var t = l.composite(ctx)
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

func (l unilo) url(ctx Context, scheme Value) (res Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "URL")) }

	var (
		url = &URL{ Scheme:scheme }

		p = l.p

		colon1 = p.expect(ctx, COLON) // consumes ':'
		colon2 = NoPos
		//colon3 = NoPos
		a = NoPos // @
	)

	if p.tok == PCON {
		p.step() // the first '/'
		if p.tok == PCON {
			p.expect(ctx, PCON) // the second '/'
		} else {
			erro(ctx, "TODO: URL path: %v (%T) (next: %s (%s))", scheme, scheme,  p.tok, p.lit).trace()
		}
	} else if !p.isEndOfURL(ctx) {
		erro(at(ctx, p.loc(colon1)), "TODO: URL: %v (%T) (next: %s (%s))", scheme, scheme, p.tok, p.lit).trace()
	}

	if !p.isEndOfURL(ctx) {
		userOrHost := l.composite(ctx)
		if p.tok == COLON {
			url.Username, colon2 = userOrHost, p.pos
			p.step() // ':'
			if p.tok != AT && p.tok != PCON && !p.isEndOfURL(ctx) {
				url.Password = l.composite(ctx)
			}
		} else {
			url.Host = userOrHost
		}
		if p.tok == AT {
			p.step() // '@'
		}
	}
	if url.Host == nil && colon2 == NoPos && a == NoPos && !p.isEndOfURL(ctx) {
		url.Host = l.composite(ctx)
		if p.tok == COLON {
			//colon3 = p.pos
			p.step() // ':'
			if p.tok != SPACE && p.tok != LINEND {
				url.Port = l.composite(ctx)
			}
		}
	}
	if p.tok == PCON {
		url.Path = l.path(ctx, _pathpun(ctx, p.tok))
	}
	// scanning '#' as HASH instead of COMMENT
	defer p.scanner.setBits(p.scanner.commentsOff())
	if p.tok == QUE {
		p.step() // '?'
		if p.tok != HASH && !p.isEndOfURL(ctx) {
			url.Query = l.composite(ctx)
		}
	}
	if p.tok == HASH {
		p.step() // '#'
		if !p.isEndOfURL(ctx) {
			url.Fragment = l.composite(ctx)
		}
	}
	return url
}

func (l unilo) resolve(ctx Context, name Value, str string) (result Value) {
    var pos Position
    if !(name == nil) { pos = name.Position() }
    if !pos.IsValid() { pos = _position(ctx) }
    if !pos.IsValid() { pos = l.p.Position() }
    if str == "" {
		erro(ctx, "resolve no-name : %v", ts(name)).trace()
	}

	if d := auto_find(ctx, str); d != nil {
		return d
	}

	var s = _scope(ctx)

	if _, o := s.find(str); o != nil {
		return o
	}

	if truly(ctx, parse_is_auto{str}) {
		if a := s.auto(ctx, str); a == nil {
			erro(ctx, "failed auto: %v", ts(name)).trace()
		} else {
			return a
		}
	}

	if truly(ctx, parse_is_conf{}) {
		// Create an empty def if referred in configuration.sm.
		result, _ = s.set(ctx, str, defConfRef)
		return
	}

	if c := l.project.configure; c != nil {
		return c.resolve(ctx, str)
    }
    return
}

func (l unilo) closuredelegate_obj(ctx Context, lTok token, name Value, isClosure bool) (str string, obj Value) {
	if x,  y := name.(*argumented) ; y { name = x.Value }
	if _,  y := name.(condval) ; !y {
		if v := name.expand(ctx) ; v == nil {
			erro(ctx, "%v is nil", ts(name)).trace()
		} else {
			name = v
		}
	}

	if indeterminate(ctx, name) {
		return str, name
	}

	str = name.string(ctx)

	if lTok == LBRACE {
		if t := _project(ctx).resolveEntries(ctx, name, false) ; t == nil {
			erro(at(ctx,name), "resolved %v is nil", ts(name)).trace()
		} else {
			obj, _ = t[0].(Object)
			return
		}
	}

	if str == "" {
		switch name.(type) {
		case condval, *selection:
			return str, name
		}
		erro(ctx, "empty name: %s", ts(name)).trace()
	}

	if t := l.resolve(ctx, name, str) ; t != nil {
		obj, _ = t.(Object)
		return
	}

	if isClosure || truly(ctx, getParseCanUndef{}) || dis_evoke(ctx, name, nil) {
		obj = name // recursive delegation or closure
		return
	}

	if true {
		note(ctx, "%v", _scope(ctx))
		note(ctx, "%v", _project(ctx).scope)
		for i, s := range l.s { note(ctx, "%d. %v", i, s) }
	}

	erro(ctx, "resolve %v ⇒ nil", ts(name)).trace()
	return
}

func (l unilo) auto_arg0(ctx Context, tokLp token, isClosure bool) (_ Value) {
	if tokLp != LPAREN {
		erro(ctx, "auto: incorrect left paren: %v", tokLp).trace()
	}

	var ac = automatic{ Context:ctx, defs:make(auto_defs) }
	ac.suppress = ac.has

	ctx = &ac

	var vals []Value
	var p = l.p
	for p.spaces(ctx); !p.isEndOfList(ctx); p.spaces(ctx) {
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
			erro(at(ctx, val), "auto: %v is empty", val).trace()
		} else {
			ac.set(at(ctx, val), s, val)
		}

		if p.tok == COMMA || p.tok == EOF || p.tok == LINEND || p.lineComment != nil {
			break
		}
	}


	return makeList(vals...)
}

func (l unilo) closuredelegate_args(ctx Context, name string, tokLp token, isClosure bool) (args []Value) {
	switch name {
	case "auto"    : args = append(args, l.auto_arg0(ctx, tokLp, isClosure)); if !isClosure { ctx = parse_auto_context{ctx} }
	case "case"    : args = append(args, l.list(ctx)); ctx = parse_undef_context{ctx}
	case "foreach" : args = append(args, l.list(ctx)); ctx = parse_foreach_context{ctx}
	case "and","or": ctx = parse_undef_context{ctx}; args = append(args, l.list(ctx))
	default:         args = append(args, l.list(ctx))
	}

	for l.p.tok == COMMA {
		l.p.next(ctx, true) // consumes COMMA
		args = append(args, l.list(ctx))
	}
	return
}

func (l unilo) closuredelegate_abc(ctx Context, isClosure, special bool) (tok token, obj Value, args, opts []Value) {
	var name Value
	var str string
	var p = l.p
	tok, str = p.tok, p.lit ; p.step()

	if special {
		if obj = l.resolve(ctx, nil, str); obj == nil {
			erro(ctx, "not defined %v (name=%s)", tok, str).trace()
		}
		return
	}

	switch p.tok {
	case LPAREN, LBRACE: // $(...), ${...}
		tok = p.tok // use LPAREN, LBRACE
		p.step() // skips LPAREN, LBRACE

		if p.tok == SPACE {
			erro(ctx, "unexpected spaces").trace()
		}

		if name = l.expr(ctx); name == nil {
			erro(ctx, "%v : no name parsed", tok).trace()
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
					erro(at(ctx,v), "not a Flag: %v", ts(v)).trace()
				}
			}
		}

		if name == nil {
			erro(ctx, "name %v is nil").trace()
		}

		str, obj = l.closuredelegate_obj(ctx, tok, name, isClosure)

		if (tok == LPAREN && p.tok != RPAREN) || (tok == LBRACE && p.tok != RBRACE) {
			args = l.closuredelegate_args(ctx, str, tok, isClosure)
		}

		switch tok {
		case LPAREN: p.expect(ctx, RPAREN)
		case LBRACE: p.expect(ctx, RBRACE)
		}

	default:
		if !isClosure { // $(...), disabled $name.
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v` or `%v` or quotes", LPAREN, LBRACE).trace()
		}

		if !(p.tok == STRING || p.tok == COMPOUND) {
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v`, `%v` or quotes, not %v %v", LPAREN, LBRACE, p.tok, p.lit).trace()
		}

		pos := p.Position()
		tok = p.tok

		// &'xxxx' or &"xxxx"
		if name = l.expr(ctx); name == nil {
			erro(at(ctx,pos), "parsed name is nil").trace()
		}

		if indeterminate(ctx, name) {//, /* expandClosure */final
			erro(at(ctx,name), "name '%v' is closured", ts(name)).trace()
		}

		str, obj = l.closuredelegate_obj(at(ctx,name), tok, name, isClosure)
	}

	if obj == nil && str != "" {
		if proj := _project(ctx); proj.ext.Plugin != nil {
			if t, e := proj.ext.Lookup(str); e == nil && t != nil {
				erro(at(ctx,name), "TODO: convert ext symbol: %v : %v", name, ts(t)).trace()
			}
		}
	}
	return
}

func (l unilo) closuredelegate(ctx Context, isClosure, special bool) (result Value) {
	if l_traverse.enabled {	defer un(l_trace(l_traverse, "closuredelegate")) }

	p := l.p
	ctx = parse_call_context{token_aware_context{at(ctx, p), COMMA}}

	tok, obj, args, opts := l.closuredelegate_abc(ctx, isClosure, special)

	if obj == nil {
		erro(ctx, "%v : nil symbol", tok).trace()
	}

	if isClosure {
		return makeClosure(_position(ctx), tok, obj, opts, args...)
	} else if x, y := obj.(*def); y && x.origin == defCodeBlockAuto {
		return x.value
	} else {
		return makeDelegate(_position(ctx), tok, obj, opts, args...)
	}
}

func (l unilo) unary(ctx Context) (x Value) {
	if l_traverse.enabled && false { defer un(l_trace(l_traverse, "unary")) }

	var p = l.p
	switch p.tok {
	case ASSIGN: // Example: '=xxx'
		if !truly(ctx, getParseLeftHandSide{}) {
			var v Value
			var s = p.Position()
			if p.step(); p.isEndOfList(ctx) {
				v = makeNull(s)
			} else {
				v = l.expr(ctx)
			}
			return &pair{makeNull(s), v}
		}

	case BAREWORD:
		return p.bare(ctx)

	case BINARY, OCTAL, INTEGER, HEXADECIMAL, FLOATING, DATETIME, DATE, TIME, URI, STRING/*, RAW*/:
		return p.literal(ctx)

	case COMPOUND:
		return l.compound(ctx)

	case CLOSURE:
		return l.closuredelegate(ctx, true, false)

	case DELEGATE:
		return l.closuredelegate(ctx, false, false)

	case ESCAPE: // \
		return p.escape(ctx)

	case LPAREN: // (
		return l.group(ctx)

	case LBRACE: // {
		return l.braced(ctx)

	case COMMA:
		if v, y := do(ctx, getParseAware{COMMA}).(bool); !y || !v {
			return p.punctuation()
		}

	case AT, BAR, PLUS, SEMICOLON:
		return p.punctuation()

	case STAR, DAST, QUE, LBRACK: // * ** ? [
		return l.glob(ctx, nil) // ie. no prefix

	case PERC: // %bar (ie. no prefix)
		return l.perc(ctx, nil)

	case MINUS:
		return l.flag(ctx)

	case EXC:
		return l.negative(ctx)

	case PCON: // The root of the path
		return l.path(ctx, _pathpun(ctx, PROOT))

	case TILDE: // ~
		tok, ctx := p.tok, at(ctx, p)
		p.step() // TODO: ~user, aka $(HOME)
		return _pathpun(ctx, tok)

	case DOT, DOTDOT: // . ..
		pos, tok := p.Position(), p.tok
		switch p.step(); {
		case p.tok == PCON:
			ctx = at(ctx, pos)
			return l.path(ctx, _pathpun(ctx, tok))
		case tok == DOT, tok == DOTDOT:
			x = &punctuation{valbase{pos}, tok}
			if v, y := do(ctx, getParseAware{DOT}).(bool); !y || !v {
				x = l.dot(ctx, x)
			}
			return
		default:
			erro(at(ctx,pos), "unexpected token: %v, %v %s", tok, p.tok, p.lit).trace()
		}

	default:
		if t := p.tok.isClosure(); t || p.tok.isDelegate() {
			return l.closuredelegate(ctx, t, true)
		} else if p.tok.isKeyword() { // keywords here are barewords
			return l.p.bare(ctx)
		}
	}

	if p.lineComment != nil {
		for _, comment := range p.lineComment.list {
			erro(at(ctx,comment.pos), "# %s", comment.string).trace()
		}
	}

	if false { p.step() } // go to the next token

	erro(at(ctx, p), "bad: %v (lit=%s, scan=%v)", p.tok, p.lit, p.scanner.scanstate).trace()
	return
}

func (p *parser) isParametersGroup(ctx Context, x Value) (res bool) {
	if truly(ctx, getParseCanParams{}) {
		if g, y := x.(*group); y && len(g.elems) == 1 {
			_, res = g.elems[0].(*group)
		}
	}
	return
}

func (l unilo) composite(ctx Context) (x Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "composite")) }

	x = l.unary(ctx)

	switch l.p.tok { // check composible expressions
	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2: // foo→bar  foo⇒bar  foo~>bar
		x = l.parse_select(ctx, x) // accepts foo⇒bar , but foo⇒bar is different
		break

	case STAR, DAST, QUE, LBRACK: // * ** ? [
		if !truly(ctx, getParseGlob{}) {
			if l.p.tok == QUE {
				switch l.p.step() ; l.p.tok {
				case SPACE, RPAREN, RBRACK, RBRACE, COMMA, SELECT_PROP, SELECT_PROG1, SELECT_PROG2, LINEND:
					return condish(ctx, x)
				}
			}
			if _, y := x.(*globpat); !y {
				x = l.glob(ctx, x)
			}
		}

	case PERC: // foo%bar ; FIXME: %/foo/bar -> Path(% foo bar)
		x = l.perc(ctx, x)

	case DOT: // foo.bar.baz.o ; FIXME: push bits when parsing $(...)
		if v, y := do(ctx, getParseAware{DOT}).(bool); !y || !v {
			x = l.dot(ctx, x)
		}

		// TODO: parse to Qualword

	// case PCON: // ie. subdir/in/somewhere
	// 		switch x.(type) { // Path expressions, except '-I/path/to/include'
	// 		case flag: // By pass expressions like -I/foo/bar.
	// 		default: x = p.path(ctx, lhs, x)
	// 		}

	case COLON:
		if (truly(ctx, getParseIsRecipe{false}) || !truly(ctx, getParseLeftHandSide{})) {
			if isKnownURLScheme(x.string(at(ctx, l.p))) {
				x = l.url(ctx, x)
			}
		}
	}
	return
}

func (l unilo) expr(ctx Context) (x Value) {
	if false && l_traverse.enabled { defer un(l_trace(l_traverse, "expr")) }
	if count_error(ctx) > 0 {
		flush(ctx)
		return
	}

	var tok, lit = l.p.tok, l.p.lit

	if x = l.composite(ctx); x == nil {
		erro(at(ctx, l.p), "invalid (%v,%v; prev=%v,%v)", l.p.tok, l.p.lit, tok, lit).trace()
	}

	if truly(ctx, getParseGlob{}) { return }

	var lhs = truly(ctx, getParseLeftHandSide{})
	if l.p.tok.isAssign() && lhs { return }
	if l.p.isParametersGroup(ctx, x) { return }

	var n int

composeLoop:
	switch l.p.tok {
	case ASSIGN: // Example: 'key=value'
		if !lhs {
			x = l.pair(ctx, x)
		}
		return

	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2: // Example: foobar⇒run(-gen)
		x = l.parse_select(ctx, x)
		goto composeLoop

	case LPAREN:
		if x = l.argumentedExpr(ctx, x); x != nil {
			goto composeLoop
		}
		return

	case PCON:
		// Path, excepts '-I/path/to/include'
		switch x.(type) {
		case flag:
		default: x = l.path(ctx, x)
		}
		return // FIXES: a%%b/foo/bar -> Path(a%%b foo bar)

	case BAR: // Example: [(var)|...]
		if _, y := x.(*group); y { return }

	case COMMA:
		if v, y := do(ctx, getParseAware{COMMA}).(bool); y && v { return }

	case COMPOSED, COLON, RAW, RPAREN, RBRACK, RBRACE, SPACE, SEMICOLON, LINEND, EOF:
		return // terminate
	}

	x = compose(ctx, x, l.composite(ctx))

	switch l.p.tok { case SPACE, COMMENT, LINEND, EOF: return }

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
            }
        default:
            erro(at(ctx,prop), "parameter `%v` unsupported `%T`", prop, prop)
        }
    }
    return
}

func (l unilo) use(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	if l.p.imports = append(l.p.imports, &use_spec{ g.spec }); g.skip {
		// TODO: maybe give some information
		return
	}

	ctx = at(ctx, g.spec[0])

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
		specVal0, ctx = v.Value, &argumented_context{ctx, v.args}
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

	var opts useOpts
	var args = parseOpts(ctx, &opts, append(g.remainder, g.spec[1:]...)...)
	for _, a := range args {
		if _, ok := a.(flag); ok || true {
			erro(at(ctx,a), "unkown use opts: %v", ts(a)).trace()
		}
	}

	var wg sync.WaitGroup ; defer wg.Wait()

	for _, specVal := range specVals {
		var ctx = at(ctx, specVal)

		if true { // TODO: use goroutine?
			l.use_spec(ctx, opts, specVal, args...)
		} else {
			var dc = diagnostic{ Context: ctx }
			wg.Add(1)
			go func() {
				defer func() {
					if false { trace(&dc) }
					if len(dc.points) > 0 { _diagnostic(ctx).nest(dc.points) }
					wg.Done()
				} ()
				l.use_spec(ctx, opts, specVal, args...)
			} ()
		}
	}
	return
}

func (l unilo) parse_include(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Spec")) }

	var opts = includeOpts{ clauseopts: g }
	if vals := parseOpts(ctx, &opts, g.remainder...); len(vals) > 0 {
		// TODO: deal with the unparsed generic options
		warn(ctx, "unknown opts: %v", vals).debug()
	}

	if len(g.spec) < 1 {
		erro(ctx, "expecting include file: %v", g.spec).trace()
	}

	var x = g.spec[0]//.expand(ctx, final|expandPlaceholder)
	if l.p.spaces(ctx); l.p.tok == COLON {
		switch x.(type) {
		case *File, *strlit, *compound: // escape from file searching
		default:
			if file := l.project.file(ctx, x.string(ctx)); file != nil {
				x = file
			} else if val := x.expand(ctx); !isNull(val) && val != x {//, final
				x = val
			}
		}

		x = l.parse_rule(ctx, nil, []Value{x}) // this should return a Rule
	}

	if !g.skip { l.include(ctx, x, opts) }
}

func (l unilo) files(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	if len(g.spec) != 1 {
		erro(ctx, "too many files properties: %v", g.spec).trace()
	}

	var path Value
	if l.p.tok == SELECT_PROG1 {
		l.p.next(ctx, true) // step forward with spaces skipped
		if l.p.tok == LINEND || l.p.lineComment != nil {
			erro(ctx, "expecting files path")
		}
		path = l.expr(ctx)
	}

	l.p.spaces(ctx)

	if l.p.lineComment != nil {
		//spec.Comment = p.lineComment
	}
	if g.skip {
		// TODO: maybe give some information
		return
	}

	ctx = at(ctx, l.p)

	var opts = cacher{ g.generalOpts }
	if rest := parseOpts(ctx, &opts, g.remainder...); rest != nil {
		erro(ctx, "unsupported opts: %v", rest).trace()
	}

	var pats []Value
	if x, y := g.spec[0].(*group); y {
		pats = x.elems
	} else if indeterminate(ctx, g.spec[0]) {
		pats = []Value{ g.spec[0] }
	} else {
		pats = xmerge(original{ctx, defExpand1}, g.spec[0])
	}

	if path == nil {
		if len(pats) == 1 { if a, ok := pats[0].(*argumented); ok { if f, y := a.Value.(flag); y {
			var name = f.Value.string(ctx)
			switch name {
			default:
				// TODO: parse files options
				erro(at(ctx,f.Value), "invalid files flag: %v").trace()
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
			var paths = []Value{ makeStrlit(g.spec[0].Position(), _project(ctx).absPath) }
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
					erro(at(ctx,f.Value), "invalid files flag: %v").trace()
				}
			}
		}

		opts.cache(ctx, patsNew, paths)
	}
}

func (p *parser) evalConfiguration(ctx Context, g *clauseopts, props []Value) {
	var project = _project(ctx)
	if project == nil {
		erro(ctx, "configuration: nil project").trace()
	} else if project.configure == nil {
		erro(ctx, "configuration: no %s for %v", dotConfigure, project).trace()
	}

	if entry := project.configure.defaultEntry; entry == nil {
		// no init entry from .configure
	} else {
		entry.execute(at(ctx, entry))
	}

	if flush(ctx)>0 { return }
	if project.configured {
		prompt(ctx, "configuration: %v already configured\n", project)
		return
	}

	var ce = configurecontext{Context:ctx} ; defer ce.close()

	for _, dep := range xmerge(ctx, props/* [1:] */...) {
		if re, y := dep.(*rule); !y {
			erro(ctx, "unsupported prerequisite: %T %v", dep, dep).trace()
		} else {
			re.execute(ctx)
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

func (l unilo) parse_eval(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
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
				erro(at(ctx,op), "unsupport flag: %v (%v)", ts(v), val).trace()
			}
		}

		// NOTE: see also universeContext.configure()
		if opts.configuration { l.p.evalConfiguration(ctx, g, g.spec) }
		return
	}

	prop0 := g.spec[0]

	if isTrivial(prop0) {
		erro(ctx, "illegal").trace()
	}

	ctx = at(ctx, prop0)

	var opts []Value
	if a, y := prop0.(*argumented); y { prop0, opts = a.Value, a.args }

	name := prop0.string(ctx)
	if name == "configuration" {
		erro(ctx, "use '-configuration' instead (%v)", prop0).trace()
	}

	resolved := l.resolve(ctx, prop0, name)

	switch x := resolved.(type) {
	case invoker:
		if b, y := x.(*builtin); y && !b.isCommand() {
			erro(ctx, "resolved builtin '%v' is not a command", prop0).trace()
		}
		x.invoke(ctx, opts, g.spec[1:])
		return
	default:
		erro(ctx, "resolved '%v' is %s (%v)", prop0, typeof(resolved), *g).trace()
	}

	/* TODO: if c, y := res.(code); y { ... } */
}

func (l unilo) directive(ctx Context) (props []Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "spec")) }

	//var doc = p.leadComment
	// var comment *commentGroup

DirParamsLoop:
	for l.p.tok != EOF {
		switch l.p.spaces(ctx); l.p.tok {
		case COMMA, LINEND, RPAREN, RBRACE,
			SELECT_PROG1, COLON: break DirParamsLoop
		}

		if l.p.lineComment != nil {
			// TODO: comment = p.lineComment
			break
		}

		props = append(props, l.expr(ctx))
	}
	return
}

func (l unilo) spec(ctx Context, keyword token, pos Pos, f parseSpecFunc) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "spec("+keyword.String()+")")) }

	var p = l.p
	var opts = clauseopts{ keyword: keyword }
	for p.spaces(ctx); p.tok == MINUS; p.spaces(ctx) {
		opts.values = append(opts.values, l.expr(ctx))
	}

	opts.remainder = parseOpts(ctx, &opts, opts.values...)

	for _, cond := range opts.conds {
		if t := cond.true(at(ctx, cond)); !t {
			opts.skip = true
			break
		}
	}

	p.spaces(ctx)

	switch p.tok {
	case LINEND:
		if keyword == EVAL {
			f(at(ctx, p), nil, &opts, 0)
			return
		} else {
			erro(at(ctx, p), "%v: nil specs", keyword).trace()
		}
	case LPAREN:
		p.next(ctx, true)
		for iota := 0; p.tok != RPAREN && p.tok != EOF && (p.stop == 0 || p.pos < p.stop); iota++ {
			// TODO: collect documentation comments
			for p.tok == SPACE || p.tok == LINEND { p.next(ctx, true) }
			if p.tok == RPAREN || p.tok == EOF { break  }
			if opts.spec = l.directive(ctx); true {
				f(at(ctx, p), p.leadComment, &opts, iota)
			}
			if p.tok == COMMA || p.tok == LINEND { p.next(ctx, true) }
		}
		p.expect(ctx, RPAREN)
		p.spaces(ctx)
		if p.tok != EOF { p.linend(ctx) }
		return
	}

	if p.tok != LINEND && p.tok != EOF && (p.stop == 0 || p.pos < p.stop) {
		if opts.spec = l.directive(ctx); true { f(ctx, nil, &opts, 0) }
		if p.tok == COMMA { p.next(ctx, true) }
	}
	if p.tok != EOF && (p.stop == 0 || p.pos < p.stop) {
		if p.spaces(ctx); p.lineComment == nil { p.linend(ctx) }
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
	if checkpoints && truly(ctx, is_test_mode{}) {
		defer p.define_check(ctx, tok, ident, value, &d)
	}

    var alt Object

    switch t := ident.(type) {
    case *argumented:
        erro(ctx, "TODO: multiple defs: %v, args=%v", t.Value, t.args).trace()

    case *group:
        erro(ctx, "TODO: multiple defs: %v", t.elems).trace()

    case *selection:
        if v := t.expand(final{ctx}); v == nil {
            erro(ctx, "%v is nil", ts(t)).trace()
        } else if x, y := v.(*def); !y {
            erro(ctx, "%v is not a def: %v", ts(t), ts(v)).trace()
        } else {
            d = x
        }

    default: // *bareword, *barecomp, *qualword, *path, flag:
        var name = t.string(ctx)
        if _, y := builtins[name]; y {
            erro(ctx, "`%v` is a builtin name (%v)", ident, name).trace()
        }

        // Resolve base value to derive.
		var proj = _project(ctx)
        var prev = proj.resolve(ctx, name)

        if d, alt = _scope(ctx).set(at(ctx, t), name, defUndetermined); alt == nil {
            if d == nil {
                erro(ctx, "`%s` is undefined (%v)", name, ts(t)).trace()
            }
        } else if tok == ASSIGN || tok == ASSIGN_EXC {
            if a, y := alt.(*def); !y {
                erro(ctx, "`%v` already defined (%T) (%v)", ident, alt, alt.owner()).trace()
            } else if a.owner() == proj && a.origin != defConfRef {
                erro(ctx, "`%v` already defined (%T)", ident, alt).trace()
            } else {
                d = a
            }
        } else if t, y := alt.(*def); !y {
            erro(ctx, "%s: object is not def: %s, %v", name, typeof(alt), ts(prev)).trace()
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
            erro(ctx, "prev def '%s' is nil", name).trace()
        } else if derived == d || (d.value != nil && d.value.refs(ctx, derived)) {
            // same def
        } else if d != nil && (tok == ASSIGN_ADD || tok == ASSIGN_SHI) && alt == nil {
            if d.origin == defVoid { d.origin = derived.origin }
            if !isTrivial(derived.value) { d.append(ctx, derived.value) }
        }
    }

    if d == nil {
        erro(ctx, "def is nil: %v", ts(ident)).trace()
    }

    if false { d.position = ident.Position() }

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
        erro(ctx, "unknown origin: %v %v %v", d.origin, d.name, tok).trace()
    }
    return
}

func (l unilo) assign_value(ctx Context, ident Value, tok token) (value Value) {
	defer l.closescope(l.openscope(fmt.Sprintf("def %v", ident)))

	vals := l.values(parse_defvalue_context{ctx})
	l.p.lineComment = nil
	return ease(ctx, vals)
}

func (l unilo) assign(ctx Context, ident Value) (res []*def) {
	if l_traverse.enabled || debugSyntax(ctx, "define") {
		defer un(l_trace(l_traverse, fmt.Sprintf("assign(%s)", ident)))
	}

	var tok = l.p.tok

	l.p.next(ctx, true) // assign token

	ctx = at(ctx, l.p)

	var value = l.assign_value(ctx, ident, tok)
	res = l.p.define_idents(ctx, tok, ident, value)

	if checkpoints {
		if len(res) == 0 {
			erro(ctx, "%v %v %v", ident, tok, ts(value)).trace()
		} else if len(res) == 1 && res[0].value == nil && value != nil {
			erro(ctx, "%v %v %v", ident, tok, ts(value)).trace()
		}
	}
	return
}

func (l unilo) recipe(ctx Context) Value {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Recipe")) }

	var (
		// TODO: comment *commentGroup
		// TODO: doc = p.leadComment
		elems []Value
		isList, isPlainline bool
	)

	var p = l.p
	switch p.dialect {
	case "", "eval", "value":
		p.scanner.pop(isCompoundLine)
		p.next(ctx, true) // skip RECIPE or SEMICOLON and parse in list mode
		if isList = true; !p.isEndOfLine() {
			var a *argumented
			var x = l.expr(ctx) // parse first expr of recipe
			if x != nil {
				if a, _ = x.(*argumented); a != nil { x = a.Value }
			}
			if x == nil {
				erro(ctx, "parsed value is nil").trace()
			} else if p.dialect == "value" {
				// no resolving commands
			} else if t, y := x.(*bareword); !y {
				// does nothing
			} else if s := l.resolve(ctx, t, t.s); isTrivial(s) {
				erro(at(ctx,x), "no such symbol: %v, %s → %s; dialect=%s", t.s, ts(x), ts(s), p.dialect).trace()
			} else if b, y := s.(*builtin); !y {
				erro(at(ctx,x), "'%s' is not a command (%s)", t.s, typeof(s)).trace()
			} else if !b.isCommand() {
				erro(at(ctx,x), "'%s' is not a command, use $(%s ...) instead", t.s, t.s).trace()
			} else { x = s }

			if a != nil {
				elems, a.Value = append(elems, a), x
			} else {
				elems = append(elems, x)
			}

			var cmdargs []Value
			var c = parse_recipe_context{ctx, true} // builtin recipe

			for p.tok != EOF && p.tok != SEMICOLON && p.tok != LINEND && p.lineComment == nil {
				if p.spaces(ctx); p.lineComment != nil { break }
				if !p.tok.isRuleDelim() {
					x = l.expr(c)
				} else {
					erro(ctx, "unsupported token: %s, %v", p.tok, elems).trace()
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

		switch p.dialect {
		case "plain", "text": isPlainline = true
		}

		var c = parse_recipe_context{ctx, false} // builtin text
		for !p.isEndOfLine() {
			var x Value
			if p.tok == RAW {
				x = p.literal(c)
			} else {
				x = l.expr(c)
			}
			elems = append(elems, x)
		}
		p.scanner.pop(isCompoundLine)
	}
	if p.spaces(ctx); p.tok != EOF { p.linend(ctx) }
    if len(elems) == 0 {
        return makeNone(_position(ctx))
	} else if isPlainline {
		return &plainline{elements{merge(elems...)}}
    } else if isList {
        return makeList(elems...)
    } else {
        return makeCompound(elems...)
    }
}

// Parsing (var a=xxx,b=yyy) definitions
func (p *parser) movar(ctx Context, args ...Value) (err error) {
	var s = _scope(ctx)
	for _, elem := range args {
		var kv, y = elem.(*pair)
		if !y || kv == nil {
			erro(at(ctx,elem), "bad var form (%v)", ts(elem)).trace()
		}

		if d, a := s.set(at(ctx, elem), kv.key, defUndetermined); a != nil {
			erro(at(ctx,kv), "'%v' already defined: %v", kv.key, ts(a)).trace()
		} else if d == nil {
			erro(at(ctx,kv), "'%v' not defined", kv.key).trace()
		} else {
			var v = kv.val
			if g, y := v.(*group); y { v = g.list() }
			d.val(at(ctx,kv), v)
		}
	}
	return
}

func (p *parser) defineConfigureTargets(ctx Context) {
	var proj = _project(ctx)
	for _, t := range p.targets {
		var ctx = at(ctx, t)

		d, a := proj.set(ctx, t, defConfig)

		if d == nil && a != nil {
			if d, _ = a.(*def); d == nil {
				erro(ctx, "%v : already defined: %v", ts(t), ts(a)).trace()
			}
		}

		if d != nil && !d.position.IsValid() { d.position = t.Position() }
	}
}

func (l unilo) modifier(ctx Context) (res *modifier) {
	p := l.p
	p.spaces(ctx)

	ctx = parse_modifier_context{at(ctx, p)}

	p.expect(ctx, LPAREN)
	p.spaces(ctx)

	var name string
	var nameVal = l.expr(ctx)
	var elems []Value
	switch n := nameVal.(type) {
	case *bareword: name = n.s
	case *delegate, *closure:
		var v = xmerge(at(ctx, n), nameVal)//, final
		if len(v) == 0 {
			erro(ctx, "empty modifier name: %v", n).trace()
		}

		name, elems = v[0].string(ctx), v[1:]

	default:
		erro(ctx, "unsupported modifier: %v", ts(n)).trace()
	}

	var movar bool
	switch name {
	case "var": movar = true
	case "configure":
		p.defineConfigureTargets(ctx)
		p.configure = true // set configure flag and define configure variables
	case "":
		erro(ctx, "empty modifier name: %v", ts(nameVal)).trace()
	}

	if _, y := dialects[name]; y {
		if p.dialect != "" {
			erro(ctx, "multi-dialects unsupported, already defined '%s'", p.dialect).trace()
		}

		p.dialect = name
	} else if _, y = modifiers[name]; !y {
		erro(ctx, "`%s` no such dialect or modifier", name).trace()
	}

	for p.tok != RPAREN && p.tok != EOF {
		p.spaces(ctx)

		t := p.pos

		if vals := l.values(ctx); movar {
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
			erro(ctx, "unsupported modifier arg: %v '%v'", p.tok, p.lit).trace()
		}
	}

	p.expect(ctx, RPAREN)

	if nameVal == nil && len(elems) == 0 {
		erro(ctx, "empty modifier").trace()
	} else {
		res = new(modifier)
		res.position = _position(ctx)
		res.elems = append([]Value{nameVal}, elems...)
	}
	return
}

// example: {(modifier ...)}
func (l unilo) modification(ctx Context) *modification {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "modification")) }

	ctx = at(ctx, l.p)

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
// Similar to makefile automatic variables, see
//   * https://www.gnu.org/software/make/manual/html_node/Automatic-Variables.html#Automatic-Variables
var rule_autos = map[string]struct{}{
	"@" :struct{}{}, "%" :struct{}{}, "<" :struct{}{}, ">" :struct{}{}, "?" :struct{}{}, "^" :struct{}{}, "+" :struct{}{}, "|" :struct{}{}, "*" :struct{}{},
	"@D":struct{}{}, "%D":struct{}{}, "<D":struct{}{}, ">D":struct{}{}, "?D":struct{}{}, "^D":struct{}{}, "+D":struct{}{}, "|D":struct{}{}, "*D":struct{}{},
	"@F":struct{}{}, "%F":struct{}{}, "<F":struct{}{}, ">F":struct{}{}, "?F":struct{}{}, "^F":struct{}{}, "+F":struct{}{}, "|F":struct{}{}, "*F":struct{}{},
	"@'":struct{}{}, "%'":struct{}{}, "<'":struct{}{}, ">'":struct{}{}, "?'":struct{}{}, "^'":struct{}{}, "+'":struct{}{}, "|'":struct{}{}, "*'":struct{}{},
	"-" :struct{}{}, //"<-":struct{}{}, "->":struct{}{},
	"~" :struct{}{},
}

func (l unilo) parse_rule(ctx Context, optvals, targets []Value) (result Value) {
	if l_traverse.enabled || debugSyntax(ctx, "rule") {
		defer un(l_trace(l_traverse, "rule"))
	}

	ctx = parse_rule_context{at(ctx, l.p)}

	var proj = _project(ctx)
	if proj.keyword == PACKAGE {
		erro(ctx, "rules forbidden in package : %v", targets).trace()
	}
    if proj != _scope(ctx).project {
		erro(ctx, "mismatched project/scope : %v", targets).trace()
	}

	// TODO: doc = p.leadComment
	var depends, ordered, recipes []Value
	var position = _position(ctx)
	defer l.closescope(l.openscope(fmt.Sprintf("rule %v", targets)))
	defer func() {
		// Close the rule scope and go back to project scope. The current
		// scope must be project scope befor Rule.
		l.p.configure, l.p.dialect, l.p.ruparas = false, "", nil
	} ()

	l.p.dialect = ""
	l.p.ruparas = nil

	// NOTE: expand targets to speed up for later usage, it might spend lots of time in
	// project.entry while matching for entry looked up if not expanded right now.
	targets = expand(ctx, targets...)

	defer func(t []Value) { l.p.targets = t } (l.p.targets)
	l.p.targets = targets // save targets for later refering
	l.p.next(ctx, true) // skip rule delimeters and spaces

	if l.p.tok != SEMICOLON && l.p.tok != BAR && !l.p.isEndOfLine() {
		depends = l.depends(ctx, true)
	}
	if l.p.tok == BAR { // '|' starts the ordered prerequisites
		if l.p.next(ctx, true); l.p.tok != SEMICOLON && !l.p.isEndOfLine() {
			ordered = l.depends(ctx, false)
		}
	}

	if l.p.tok == SEMICOLON { // ;
		// Parse inline recipe in the program scope.
		recipes = append(recipes, l.recipe(ctx))
	} else /*if p.tok == LINEND || p.lineComment != nil*/ {
		// Parse recipes in the program scope.
		l.p.scanner.recipes(true) // Turn on recipes before LINEND.
		if l.p.linend(ctx) { // Take the new line.
			for l.p.tok != EOF && l.p.isRecipeStart() {
				recipes = append(recipes, l.recipe(ctx))
			}
		}
		l.p.scanner.recipes(false)
	}

	if t := targets[0]; l.p.configure {
		d, a := proj.set(ctx, t, defVoid)
		if d == nil && a == nil {
			erro(at(ctx,t), "configure target '%v' not defined", t).trace()
		} else if a == nil {
			// ...
		} else if _, y := a.(*def); !y {
			erro(at(ctx,t), "configure target '%v' already taken: %v", t, ts(a)).trace()
		}
		if d != nil && !d.position.IsValid() { d.position = t.Position() }
	}

	// TODO: lang: 0,

    var prog = program{
        configure: l.p.configure,
        language:  l.p.dialect,
        params:    l.p.ruparas,
        position:  position,
        project:   proj,
		depends:   depends,
		ordered:   ordered,
        recipes:   recipes,
    }

	if res := l.entries(at(ctx, position), &prog, targets, optvals); len(res) == 1 {
		result = res[0]
	} else if 1 < len(res) {
		result = _list_t[entry](res...)
	} else {
		result = makeNull(position)
	}
	return
}

func (l unilo) entries(ctx Context, prog *program, targets, options []Value) (res []entry) {
    for _, target := range targets {
        var ctx = at(ctx, target)

        if isTrivial(target) {
            if true { continue }
			erro(ctx, "trivial target; %v", targets).trace()
        }

        var entry = prog.project.entry(ctx, options, target, prog)
        if entry == nil {
            erro(ctx, "creating entry failed for %v", target).trace()
        }

		res = append(res, entry)

        if x, y := entry.destiny().(flag); y && x.Value != nil {
			if prog.project.name != "~" {
				var s = x.Value.string(ctx)
				l.globe.AddFlagEntry(s, entry)
			}
        } else if l.p.configure {
            if entry.patterned(ctx) {
                erro(ctx, "unsupported pattern configures: %v", target).trace()
            }
            prog.project.configs = append(prog.project.configs, entry)
        }
    }
    return
}

var pprofCounter int

func (l unilo) parse_def(ctx Context) {
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

func (l unilo) parse_foreach(ctx Context) {
	if l.p.spaces(ctx); l.p.tok == LINEND {
		erro(at(ctx, l.p), "unexpected end of line").trace()
	}

	l.p.expect(ctx, FOREACH)
	l.p.spaces(ctx)

	var params = l.values(ctx)
	var t = &template{
		pos: l.p.pos, tok: l.p.tok, lit: l.p.lit,
		state: l.p.scanner.scanstate,
	}

	l.p.spaces(ctx)
	l.p.linend(ctx)

	var nested = 0
	for l.p.tok != EOF { switch pos := l.p.pos; l.p.tok {
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

		var a = map[string]Value{ "_" : nil }
		for _, elem := range xmerge(final{ctx}, params...) {
			if indeterminate(ctx, elem) {
				erro(ctx, "indeterminate param: %v", ts(elem)).trace()
			} else if isTrivial(elem) {
				if false { info(ctx, "trivial: %v", ts(elem)).debug() }
			} else {
				a["_"] = elem
				l.parse_codeblock(ctx, t, a)
			}
		}
		return

	default:
		for l.p.tok != EOF {
			if l.p.next(ctx, true); l.p.tok == LINEND { l.p.next(ctx, true) ; break }
		}
	}}
}

func (l unilo) parse_for(ctx Context) {
	if l.p.spaces(ctx); l.p.tok == LINEND {
		erro(at(ctx, l.p), "unexpected end of line").trace()
	}

	var opts struct {
		skipNil bool `skip-nil,skip-null,skipnil,skipnull,no-nil,no-null`
		loose bool `loose`
	}

	if l.p.expect(ctx, FOR); l.p.tok == LPAREN {
		l.p.next(ctx, true) // LPAREN
		if vals := parseOpts(ctx, &opts, l.values(ctx)...); vals != nil {
			erro(at(ctx, vals[0]), "unexpected opts: %v", vals).trace()
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

	var params []*nparam
	var vars = map[string]Value{}
	for l.p.spaces(ctx); l.p.tok != EOF && !l.p.isEndOfLine(); l.p.spaces(ctx) {
		if l.p.tok == AND && params == nil {
			erro(at(ctx, l.p), "unexpected 'and'").trace()
		} else if l.p.tok == AND || params == nil {
			if params = append(params, &nparam{p:l.p.Position()}); l.p.tok == AND {
				l.p.next(ctx, true) // and
				continue
			}
		}

		var _v = params[len(params)-1]
		for _, a := range xmerge(at(ctx, l.p), l.expr(ctx)) {
			var elems []Value
			var s string

			if x, y := a.(*pair); !y {
				erro(at(ctx,a), "unexpected value: %v", ts(a)).trace()
			} else if s = x.key.string(at(ctx, x.key)); s == "" {
				erro(at(ctx,a), "empty key: %v", ts(x.key)).trace()
			} else if g, y := x.val.(*group); y {
				elems = g.elems
			} else {
				elems = append(elems, x.val)
			}

			// Make sure all elements are expanded.
			elems = xmerge(at(ctx, a), elems...)

			if _, y := vars[s]; y {
				erro(at(ctx, a), "duplicated key: %v", s).trace()
			} else {
				vars[s] = &null{valbase{a.Position()}}
			}

			if n := len(elems); n > _v.n { _v.n = n }

			_v.a = append(_v.a, &param{s, elems})
		}
	}

	var t = &template{
		pos: l.p.pos, tok: l.p.tok, lit: l.p.lit,
		state: l.p.scanner.scanstate, // verb: "for",
	}

	l.p.spaces(ctx)
	l.p.linend(ctx)

	var nested = 0
	for l.p.tok != EOF { switch pos := l.p.pos; l.p.tok {
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
			if _v.n > 0 { if num == 0 { num = _v.n } else { num *= _v.n } }
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
				for k, t := range params { if t.n == 0 {
					if true { continue outer }
				} else if k <= _i {
					if 0 < _i { i %= t.n }
				} else {
					if _i < e { i /= t.n }
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
			if !trivial { l.parse_codeblock(ctx, t, vars) }
		}
		return

	default:
		for l.p.tok != EOF {
			if l.p.next(ctx, true); l.p.tok == LINEND { l.p.next(ctx, true) ; break }
		}
	}}
}

func (l unilo) parse_codeblock(ctx Context, t *template, vars map[string]Value) {
	l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate = t.pos, t.tok, t.lit, t.state

	if false {
		pprofCounter += 1
		defer startCPUProfile(ctx, fmt.Sprintf("template-%05d.prof", pprofCounter), true)()
	}

	if !(l.p.pos < l.p.stop) {
		erro(at(ctx,l.p.loc(l.p.pos)), "bad range: [%v %v) (%v)", l.p.pos, l.p.stop, t.name).trace()
	}

	var c = parse_code_context{automatic{Context:ctx}}
	c.suppress, c.defs = c.has, make(auto_defs)

	if  _, y := vars["_"]; !y { vars["_"] = nil }

	for s, v := range vars {
		d, _ := c.set(&c, s, v)
		d.origin = defCodeBlockAuto
	}

	for l.p.tok != EOF && l.p.pos < l.p.stop {
		if l.p.tok == SPACE || l.p.tok == LINEND || (l.p.tok == COMMENT && l.p.lineComment != nil) {
			l.p.next(ctx, true)
		} else {
			l.parse_clause(&c)
		}
	}
}

func (l unilo) parse_repeat(ctx Context, t *template, params []Value) {
	defer func(t time.Time, pos Pos, tok token, lit string, state scanstate) {
		if l.ddd == "template.repeat" {
			// dont check time in ddd mode
		} else if d := time.Now().Sub(t); d > l.slow {
            warnstack(ctx, 3, "slow: %v, prof-%d", d, pprofCounter).debug()
        }

		l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate = pos, tok, lit, state
	} (time.Now(), l.p.pos, l.p.tok, l.p.lit, l.p.scanner.scanstate)

	// TODO: parseOpts(params) -> add option to turn off asFile in Context

	if false { pprofCounter += 1
		var (
			profCpu = fmt.Sprintf("template-%05d.cpu.prof", pprofCounter)
			profMem = fmt.Sprintf("template-%05d.mem.prof", pprofCounter)
			fCpu *os.File
			e error
		)
		if fCpu, e = os.Create(profCpu); e != nil {
			erro(ctx, "%T: %v", e, e).trace()
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

	var m = map[string]Value{}

	for i, v := range t.params { if s := v.string(ctx); s != "" {
		if i < len(params) { m[s] = params[i] } else {
			m[s] = makeNull(v.Position())
		}
	} else {
		erro(at(ctx,v), "empty template param name: %v %v", v, v).trace()
	}}

	l.parse_codeblock(ctx, t, m)
}

func (l unilo) call(ctx Context, name Value, args []Value) (result bool) {
	ctx = at(ctx, l.p)

	for _, t := range l.p.templates {
		if t.name != nil && eq(ctx, t.name, name) {
			stop := l.p.stop
			l.p.stop = t.endPos
			l.parse_repeat(ctx, t, args)
			l.p.stop = stop
			return true
		}
	}

	erro(ctx, "undefined template: %v", name).trace()
	return
}

func (l unilo) parse_clause(ctx Context) {
	if l_traverse.enabled { defer un(l_tracef(l_traverse, "clause(%v, %v)", l.p.tok, l.p.pos)) }

	l.p.spaces(ctx)

	ctx = at(ctx, l.p) // set position after any spaces

	if l.p.tok == LINEND || (l.p.tok == COMMENT && l.p.lineComment != nil) {
		l.p.next(ctx, true)
		return // noop clause
	}

	switch t := l.p.tok ; t {
	case USE, TEMPLATE:
		erro(ctx, "unexpected %v", t).trace()
	case  INCLUDE: l.spec(ctx, t, l.p.expect(ctx, t), l.parse_include); return
	case    FILES: l.spec(ctx, t, l.p.expect(ctx, t), l.files)        ; return
	case   ASSERT: l.spec(ctx, t, l.p.expect(ctx, t), l.p.assert)     ; return
	case   APPEND: l.spec(ctx, t, l.p.expect(ctx, t), l.p.append)     ; return
	case     EVAL: l.spec(ctx, t, l.p.expect(ctx, t), l.parse_eval)   ; return
	case      DEF: l.parse_def(ctx)    ; return
	case      FOR: l.parse_for(ctx)    ; return
	case  FOREACH: l.parse_foreach(ctx); return
	}

	var x = l.expr(parse_left_context{ctx})

	if l.p.spaces(ctx); l.p.tok.isAssign() {
		l.assign(ctx, x)
		return
	}

	if l.p.tok.isRuleDelim() {
		l.parse_rule(ctx, nil, []Value{x})
		return
	} else if a, y := x.(*argumented); y {
		l.call(ctx, a.Value, a.args)
		return
	}

	if vals := l.values(ctx, x); l.p.tok != EOF {
		return
	} else if strings.HasSuffix(l.p.scanner.file.Name(), pathSep+configuration_sm) {
		if false { note(ctx, "%v (kit=%s)", l.p.tok, l.p.lit).debug() }
	} else if truly(ctx, parse_is_conf{}) {
		note(ctx, "bad clause: %v (kit=%s) after %v", l.p.tok, l.p.lit, vals).debug(3)
	} else {
		erro(ctx, "bad clause: %v (lit=%s) after %v", l.p.tok, l.p.lit, vals).trace()
	}
}

type project_opt struct {
	configure Value `conf,configure` // detects dotConfigure if empty
	noDock bool `nodock,no-dock` // don't load container project
    traveUseLoop bool `break,loop` // don't recursively use this project
    multiUseAllowed bool `multi`  // this project is used multiple times
	final bool `final` // no bases
}

func (l unilo) declare(ctx Context, keyword token, ident Value, name string, declOpts *project_opt) (result bool) {
    if name == "@" {
        erro(ctx, "deprecated project name: @").trace()
    }

    if _, o := l.find(name); o != nil {
        if _, y := o.(*builtin); y {
            erro(ctx, "'%s' is a builtin name", name).trace()
        }
    }

	var prev = l.loader // nil if newly declared
	var dec = l.globe_declare(ctx, name, keyword)
	if prev == nil || dec.project != prev.project {
		l.project, l.s[0] = dec.project, dec.scope
	}

	if false {
		note(at(ctx, ident), "%s", ts(prev))
		note(at(ctx, ident), "%s", ts(l.loader))
		note(at(ctx, ident), "%s", ts(_loader(l.loader.Context)))
		note(ctx, "%s", ts(ctx)).debug()
	}

    if ll := _loader(l.loader.Context); ll != l.loader && ll == prev {
        if _, a := ll.project.projectname(ctx, name, dec.project); a != nil {
            if x, y := a.(*project); !y || x != dec.project {
                erro(at(ctx,a), "%v: name already taken : %v", name, ts(a)).trace()
            }
        }
    }

    if l.globe.main != nil && l.globe.main == l.project && l.project.name != "~" {
        for _, t := range l.globe.pairs {
            switch k := t.key.(type) {
            case *bareword, *barecomp:
                l.scope().set(at(ctx, t), k, defDecl, t.val)
            case flag:
                if false { warn(ctx, "unknown flag : %v", t).debug() }
            default:
                warn(ctx, "unknown target : %v", ts(t)).debug()
            }
        }
    }

	if x, y := do(ctx, get_arguments{}).([]Value); y && len(x) != 0 {
		for _, arg := range merge(x...) {
			switch t := arg.(type) {
			case *pair:
				switch k := t.key.(type) {
				case *bareword, *barecomp:
					l.scope().set(at(ctx, t), k, defDecl, t.val)
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

func (l unilo) parse_project(ctx Context, keyword token, isMainFile bool, filename string) (_ Value, _ string, _ bool) {
	var implicitBase string // aka. foo.bar.Baz implicitly load base 'foo/bar'
	var position = _position(ctx)

	l.p.next(ctx, true)

	ctx = at(ctx, l.p)

	var vals []Value
	for l.p.tok == MINUS {
		vals = append(vals, l.expr(ctx))
		l.p.spaces(ctx)
	}

	var opts project_opt
	if a := parseOpts(ctx, &opts, vals...); len(a) > 0 {
		for _, v := range a { erro(at(ctx,v), "unknown option %v", ts(v)).trace() }
		return
	}

	var ident Value
	var name string

	// Smart-lang spec:
	//   * the project clause is not a declaration;
	//   * the project name does not appear in any scope.
	if l.p.tok == LPAREN || l.p.tok == EOF || l.p.tok == LINEND || l.p.lineComment != nil {
		var dir = filepath.Dir(filename)
		if l.project != nil && l.project.absPath == dir {
			ident = makeBareword(position, l.project.name)
		} else if name = filepath.Base(filename); name == dotBase || name == dotConfigure {
			// NOTE: loading the .base or .configure file
			ident = makeBareword(position, name)
		} else if name = filepath.Base(dir); name != "" {
			// TODO: validate basename as a valid identifier
			ident = makeBareword(position, name)
		} else {
			erro(ctx, "invalid file: %v", filename).trace()
		}
	} else if l.p.tok == TILDE {
		if ext := filepath.Ext(filename); ext != ".smart" {
			erro(at(ctx, l.p), "`%v` not a smart file", filepath.Base(filename)).trace()
		} else if s := strings.TrimSuffix(filepath.Base(filename), ext); s == "" {
			erro(at(ctx, l.p), "`%v` not tilde name", filepath.Base(filename)).trace()
		} else {
			ident = makeBareword(position, s)
		}
		l.p.next(ctx, true) // skip tilde
	} else {
		base := makePath()
		comp := makeBarecomp()
		for l.p.tok != EOF && l.p.tok != SPACE {
			var w = l.p.bare(ctx)
			if comp = comp.suffix(ctx, w).(*barecomp); l.p.tok == DOT {
				t := &punctuation{valbase{l.p.Position()}, l.p.tok}
				comp = comp.suffix(ctx, t).(*barecomp)
				base.elems = append(base.elems, w)
				l.p.step() // '.'
			} else {
				break
			}
		}

		l.p.spaces(ctx)

		if len(comp.elems) == 0 {
			// erro(ctx, "package name is empty (tok=%v %v)", t, p.tok).trace()
			// return
		} else if 0 < base.len() {
			implicitBase = base.string(ctx)
		}

		ident = comp
	}

	name = ident.string(ctx)

	if l.project != nil && l.project.name != name {
		warnstack(ctx, 5, "%v: declared multiple projects in the same directory : %v", l.project, ident).debug()
	}

	if name == "-" || name == "_" {
		erro(ctx, "package name '%s' is preserved", name).trace()
	}

	if 0 < count_error(ctx) {
		// Don't bother parsing the rest on errors.
		erro(at(ctx, l.p), "got %d errors parsing file: %s", filename).trace()
	}

	var _, prevDeclared = l.declares[name]

	if l.declare(ctx, keyword, ident, name, &opts) {
		if l.project == nil {
			erro(ctx, "undeclared project: %v", ident).trace()
		}

		if false { defer l.closecurrent(ctx, name) }

		isMainFile = isMainFile && !prevDeclared;
	}

	if basePos := l.p.Position() ; l.p.tok == LPAREN {
		for l.p.tok != EOF {
			for l.p.next(ctx, true); !l.p.isEndOfList(ctx); {
				l.p.spaces(ctx)

				ctx := at(ctx, l.p)
				param := l.expr(parse_group_context{token_aware_context{ctx,COMMA}})
				l.p.spaces(ctx)

				//if p.lineComment != nil  { break }
				//if p.tok == LINEND { break }
				if l.p.tok == EOF {
					erro(ctx, "unexpected end of file while parsing bases").trace()
				}

				vals := parseOpts(ctx, &opts, param)
				if opts.final || keyword == PACKAGE { continue }
				if !l.bases(ctx, "", merge(vals...)...) {
					erro(ctx, "load bases failed: %v", vals).trace()
				}
			}
			if l.p.tok != COMMA { break }
		}
		l.p.expect(ctx, RPAREN)
	} else if !l.bases(ctx, implicitBase) { // for special bases, e.g. .base
		erro(at(ctx, basePos), "loading bases failed").trace()
	}

	if l.p.spaces(ctx) ; l.p.tok != EOF { l.p.linend(ctx) }

	if keyword != PACKAGE {
		l.configure(ctx, ident, name, prevDeclared)
		if !opts.noDock { l.container(ctx, ident, name) }
	}

	return ident, name, isMainFile
}

func isEntryFileName(s string) bool { return filepath.Base(s) == mainFileName }

func (l unilo) parse_file(ctx Context) (_ bool) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "file '"+l.p.scanner.file.Name()+"'")) }
	if l.traceLaunch { defer un(l_trace(l_launch, "parser.file")) }
	if 0 < count_error(ctx) { return }

	var keyword  = l.p.tok
	var filename = l.p.scanner.file.Name()
	var flatmode = truly(ctx, getParseIsFlag{})

	var abs string
	if flatmode {
		abs = _project(ctx).absPath
	} else {
		abs = filepath.Dir(filename)
	}

	var rel,  _ = filepath.Rel(l.workdir, abs)
	var tmp = joinTmpPath(ctx, l.workdir, rel)

	if false && checkpoints {
		spec, _ := filepath.Rel(workBaseDir, abs)
		note(ctx, "%v", ts(l.p))
		note(ctx, "%v (workbase)", workBaseDir)
		note(ctx, "%v", filename)
		note(ctx, "%v (workdir)", l.workdir)
		note(ctx, "%v (absdir)", abs)
		note(ctx, "%v", tmp)
		note(ctx, "rel=%v spec=%s", rel, spec)
		note(ctx, "%v", ts(ctx)).debug()
	}

	var sof, _ = filepath.Rel(workBaseDir, filename)
	defer l.closescope(l.openscope("file "+sof))

	if !flatmode {
		s := l.scope()
		s.set(ctx, ".",   defVoid, _pathstr(ctx, rel))
		s.set(ctx, "/",   defVoid, _pathstr(ctx, abs))
		s.set(ctx, "CWD", defVoid, _pathstr(ctx, abs)) // Current Work Directory, TODO: make it $:cwd:
		s.set(ctx, "CTD", defVoid, _pathstr(ctx, tmp)) // Current Temp Directory, TODO: make it $:ctd:
	}

	var isMainFile = isEntryFileName(filename) // aka. do.smart, build.smart

	switch keyword {
	case PACKAGE, MODULE:
		erro(ctx, "deprecated keyword: %s", keyword).trace()
	case CONFIGURE:
		// var position = l.p.Position()
		switch l.p.next(ctx, true); l.p.tok {
		case DOT:
			if err := l.config_dir(ctx, abs, abs); err != nil {
				erro(ctx, "parsing configure directory failed, '%s': %v", abs, err).trace()
			}

			l.p.next(ctx, true) // skip the '.' token and consequence spaces

			// var ident = makeBareword(position, filepath.Base(filepath.Dir(filename)))

		default:
			erro(ctx, "unknown configuration '%v', currently only 'configure .' is supported", l.p.tok).trace()
		}

	case PROJECT:
		if flatmode {
			erro(at(ctx, l.p), "project forbidden in flat file").trace()
		}

		var ident Value
		var name string
		var prev = l.project
		ident, name, isMainFile = l.parse_project(ctx, keyword, isMainFile, filename)
		if ident == nil {
			erro(ctx, "parse project failed").trace()
		}

		if l.project != prev {
			defer l.closecurrent(ctx, name)
		}

	case EOF:
		return
	default:
		if !flatmode {
			l.p.expected(ctx, l.p.pos, "configure, project, module or package keyword")
		}
	}

	var al = !flatmode && isMainFile
	if al { l.autoload(at(ctx, l.p), "declared") }

	if l.mode&ModuleClauseOnly == 0 {
		if !flatmode {
		declaration:
			for l.p.tok != EOF {
				switch tok := l.p.tok; tok {
				case LINEND, SPACE: l.p.next(ctx, true)
				case ASSERT, EVAL, FILES, INCLUDE: l.parse_clause(ctx)
				case USE: l.spec(ctx, tok, l.p.expect(ctx, tok), l.use)
				default: break declaration
				}
			}
		}

		if false && al { l.autoload(at(ctx, l.p), "amid") }

		if l.mode&ImportsOnly == 0 { // rest of module body
			for l.p.tok != EOF {
				l.parse_clause(ctx)
			}
		}
	}

	if al { l.autoload(at(ctx, l.p), "appendix") }
	if l.ddd == "parser.files" { l.ddd = "" }

	return l.mode&ImportsOnly != 0 || l.p.tok == EOF
}
