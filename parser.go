///
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"extbit.io/smart/token"
	"extbit.io/smart/scanner"
	"path/filepath"
	"runtime/pprof"
	"runtime"
	"strconv"
	"strings"
	"unicode"
	"sync"
	"time"
	"fmt"
	"os"
)

type parseBits uint64
type specialRule int

const (
	parseGroup parseBits = 1<<iota // 0000000000000000000001
	parseArged        // 0000000000000000000010
	parseCall         // 0000000000000000000100
	parseDOT          // 0000000000000000001000
	parseDOTDOT       // 0000000000000000010000
	parseDepend0      // 0000000000000000100000
	parseGLOB         // 0000000000000001000000
	parseModifier     // 0000000000000010000000
	parsePATH         // 0000000000000100000000
	parsePERC         // 0000000000001000000000
	parseREXP         // 0000000000010000000000
	parseSELECT_PROP  // 0000000000100000000000
	parseURL          // 0000000001000000000000

	parseCompound      // 0000000010000000000000
	parseDefineClause  // 0000000100000000000000

	parseFilesSpec     // 0000001000000000000000  files ( ... )
	parseTemplateBlock // 0000010000000000000000
	parseUndefValue    // 0000100000000000000000

	parseSpecialRule   // 0001000000000000000000  e.g. :use ...:
	// parseColonName  // 0010000000000000000000  e.g. $:use:

	parseRecipeBuiltin // 0010000000000000000000  recipe builtin command
	parseRecipeText    // 0100000000000000000000
	parseRecipe = parseRecipeBuiltin | parseRecipeText

	// The parseNo* bits control the parsing priority!
	parseNoArg    = parseSELECT_PROP | parseDOT | parseDOTDOT | parsePATH | parsePERC
	parseNoPair   = parseSELECT_PROP | parseDOT | parsePATH | parsePERC
	parseNoURL    = parseSELECT_PROP | parseDOT | parsePATH | parseURL | parseGLOB | parsePERC | parseREXP /*| parseColonName*/ | parseSpecialRule
	parseNoPath   = parseSELECT_PROP | parseDOT | parsePATH | parseURL | parseGLOB | parsePERC | parseREXP
	parseNoSelect = parseSELECT_PROP | parseDOT
	parseNoGlob   = parseGLOB | parsePERC | parseREXP
	parseNoPerc   = parseGLOB | parsePERC | parseREXP
	parseNoRexp   = parseGLOB | parsePERC | parseREXP
)

const (
	specialRuleNor specialRule = iota // normal rules
	specialRuleUse // `use` rules
	specialRuleRec // recipe rules
)

const (
	userproj = "user"
	usecomment = ":user:"
)

type usespec struct {
	props []Value
}

type parsedFile struct {
	// TODO: doc *CommentGroup
	// TODO: comments *CommentGroup
	keyword token.Token // project, package or module
	position Position // position of the beginning, which has filename information
	name *barecomp // project/module name
	scope *Scope
	use []*usespec // imports
}

type parsedRuleData struct {
	position Position
	params  []string
	targets []Value
	depends []Value
	ordered []Value
	recipes []Value
	options []Value
	special specialRule
	config bool
}

type template struct {
	state scanner.State
	end *scanner.State
	pos, endPos token.Pos   // token position
	tok token.Token // one token look-ahead
	lit string      // token literal
	verb string
	name Value // if only 'def', TODO: considering []Value for nested template defs?
	params []Value
}

type autoctx struct {
	defs []*def // autos in the current context
}

type parser struct {
	Context

	scanner scanner.Scanner

	// Comments
	comments  []*CommentGroup
	leadComment *CommentGroup // last lead comment
	lineComment *CommentGroup // last line comment

	// Next token
	pos, stop token.Pos   // parsing and stop position
	tok token.Token // one token look-ahead
	lit string      // token literal

	templates []*template

	// Error recovery
	// (used to limit the number of calls to syncXXX functions
	// w/o making scanning progress - avoids potential endless
	// loops across multiple parser functions during error recovery)
	//syncPos token.Pos // last synchronization position
	//syncCnt int       // number of calls to syncXXX without progress

	// Non-syntactic parser control
	exprLev int  // < 0: in control clause, >= 0: in expression
	inRhs   bool // if set, the parser is parsing a rhs expression

	bits parseBits
    isIncludingConf bool // including configuration

	// Ordinary identifier scopes
	imports []*usespec // list of imports

	targets []Value // targets of current rule
	params []*def // parameters of current rule
	autos []*def // autos in the current context
	// autos *autoDefMap // TODO
	autop *Position // valid if in auto
	// auto *autoctx // TODO
	dialect string // recipe dialect of current rule
	configure bool // is parsing configure program?

	ddd bool // helps debug parsing via `eval -ddd=true{}`
}

func (p *parser) parser() *parser { return p }
func (p *parser) inner() Context { return p.Context }

func (p *parser) setbits(bits parseBits) { p.bits = bits }
func (p *parser) setbit(bit parseBits) (bits parseBits) {
	bits = p.bits
	p.bits |= bit
	return
}
func (p *parser) clearbit(bit parseBits) (bits parseBits) {
	bits = p.bits
	p.bits &= ^bit
	return
}

// ----------------------------------------------------------------------------
// Parsing support

func (p *parser) trace(a ...interface{}) { t_traverse.traceAt(p.Position(), a...) }

// Advance to the next token.
func (p *parser) scan() {
	// Because of one-token look-ahead, print the previous token
	// when tracing as it provides a more readable output. The
	// very first token (!p.pos.IsValid()) is not initialized
	// (it is token.ILLEGAL), so don't print it .
	if t_traverse.enabled && p.pos.IsValid() {
		s := p.tok.String()
		switch {
		case p.tok.IsLiteral():
			p.trace(s, p.lit)
		case p.tok.IsOperator(), p.tok.IsKeyword():
			p.trace("\"" + s + "\"")
		default:
			p.trace(s)
		}
	}

	var pos = p.pos
	p.pos, p.tok, p.lit = p.scanner.Scan()
	if false && p.lit == "none" { warn(p, "%v %v", p.tok, p.lit).debug(64); p.checkErrors(true) }
	if false && p.tok == token.EOF {
		erro(at(p,p.loc(pos)), "unexpected end of file").debug(1)
	}
}

// Consume a comment and return it and the line on which it ends.
func (p *parser) consumeComment() (comment *Comment, endline int) {
	// /*-style comments may end on a different line than where they start.
	// Scan the comment for '\n' chars and adjust endline accordingly.
	endline = p.scanner.File().Line(p.pos)
	if len(p.lit) > 1 && p.lit[1] == '*' {
		// don't use range here - no need to decode Unicode code points
		for i := 0; i < len(p.lit); i++ {
			if p.lit[i] == '\n' {
				endline++
			}
		}
	}

	comment = &Comment{ Pos: p.Position(), Text: p.lit }
	p.scan()

	return
}

// Consume a group of adjacent comments, add it to the parser's
// comments list, and return it together with the line at which
// the last comment in the group ends. A non-comment token or n
// empty lines terminate a comment group.
//
func (p *parser) consumeCommentGroup(n int) (comments *CommentGroup, endline int) {
	comments = new(CommentGroup)
	// add comment group to the comments list
	p.comments = append(p.comments, comments)

	endline = p.scanner.File().Line(p.pos)
	for p.tok == token.COMMENT && p.scanner.File().Line(p.pos) <= endline+n {
		var comment *Comment
		comment, endline = p.consumeComment()
		comments.List = append(comments.List, comment)
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
	if p.scan(); p.tok == token.COMMENT {
		var comment *CommentGroup
		var endline int

		// If the comment is on same line as the previous token; it
		// cannot be a lead comment but may be a line comment.
		if p.scanner.File().Line(p.pos) == p.scanner.File().Line(prev) {
			comment, endline = p.consumeCommentGroup(0)
			if p.scanner.File().Line(p.pos) != endline {
				// The next token is on a different line, thus
				// the last comment group is a line comment.
				p.lineComment = comment
			}
		}

		// consume successor comments, if any
		endline = -1
		for p.tok == token.COMMENT {
			comment, endline = p.consumeCommentGroup(1)
		}

		if endline+1 == p.scanner.File().Line(p.pos) {
			// The next token is following on the line immediately after the
			// comment group, thus the last comment group is a lead comment.
			p.leadComment = comment
		}
	}

	// if p.tok != token.LINEND && p.lineComment != nil { p.tok = token.LINEND }

	if p.ddd {
		var t = warn(p, "%v %v %v", p.tok, p.lit, p.scanner.GetState())
		if p.tok == token.COMPOUND { t.debug(12) }
		if p.tok == token.LINEND { t.debug(24) }
		p.checkErrors(true)
	} else if false {
		p.scanner.Debug = false
	}
}

func (p *parser) next(ws bool) { if p.step(); ws { p.spaces() } }

func (p *parser) spaces() {
	for p.lineComment == nil && p.tok != token.EOF {
		if p.tok == token.SPACE || (p.tok == token.RECIPE && p.bits&parseRecipeBuiltin != 0) {
			p.step()
		} else if p.tok == token.ESCAPE && p.lit == "\n" {
			if p.step(); p.tok == token.LINEND || p.lineComment != nil { break }
			if p.bits&parseRecipeBuiltin != 0 {
				TokFor: for p.tok != token.EOF {
					switch p.tok {
					case token.RECIPE: // TODO: using p.isRecipeStart()
						p.scanner.LeaveCompoundLineContext()
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
func (p *parser) loc(pos token.Pos) Position { return Position(p.scanner.File().Position(pos)) }
func (p *parser) posit() Context { return &positionContext{ p, p.Position() } } //at(p, p.Position())

// A bailout panic is raised to indicate early termination.
type bailout struct{}

func (p *parser) expected(pos token.Pos, msg string, a... interface{}) {
	if len(a) > 0 { msg = fmt.Sprintf(msg, a...) }
	if msg = "expected " + msg; pos == p.pos {
		// the error happened at the current position;
		// make the error message more specific
		if p.tok == token.SEMICOLON && p.lit == "\n" {
			msg += ", found newline"
		} else {
			msg += ", found '" + p.tok.String() + "'"
			if p.tok.IsLiteral() {
				msg += " " + p.lit
			}
		}
	}
	erro(at(p,p.loc(pos)), msg).debug(32)
}

func (p *parser) expect(tok token.Token) token.Pos {
	var pos = p.pos
	if p.tok != tok { p.expected(pos, "'"+tok.String()+"'") }
	p.step() // move forward
	return pos
}

func (p *parser) linend() (ok bool) {
	if p.lineComment != nil {
		p.lineComment, ok = nil, true
	} else if p.tok == token.EOF {
		ok = true
	} else if p.tok == token.LINEND {
		p.step(); ok = true
	} else {
		p.expected(p.pos, "'\\n'")
	}
	return
}

func (p *parser) isRecipeStart() (res bool) {
	if p.tok == token.RECIPE {
		res = true
	} else if p.tok == token.SPACE && p.lit == "\t" {
		p.tok, res = token.RECIPE, true // Fixes recipe \t
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
/*func (p *parser) _safePos(pos token.Pos) (res token.Pos) {
	defer func() {
		if recover() != nil {
			res = token.Pos(p.scanner.File().Base() + p.scanner.File().Size()) // EOF position
		}
	}()
	_ = p.scanner.File().Offset(pos) // trigger a panic if position is out-of-range
	return pos
}*/

// ----------------------------------------------------------------------------
// Barewords & Identifiers

func (p *parser) bare(lhs bool) (x Value) {
	var pos, tok, lit = p.Position(), p.tok, p.lit
	switch p.step(); tok {
	case token.BAREWORD: // okay
	case token.UNDEF:
		if p.tok == token.LBRACE { // undef{}, undef{ ... }
			if p.next(true); p.tok == token.RBRACE {
				x = &undef{&None{valbase{p.Position()}, nil}}
				p.step()
			} else if v := p.expr(p, false); v != nil {
				x = &undef{v}
				p.expect(token.RBRACE)
			} else {
				erro(at(p,pos), "undef invalid expression: %v, %v", p.tok, p.lit).debug(1)
			}
			return
		}
	case token.BARE: // TODO: bare{ ... }
		if p.tok == token.LBRACE { // file{ ... }
			erro(p, "TODO: %v", tok).debug(1) ; p.checkErrors(true)
			return
		}
	case token.REGEX: // TODO: regex{ ... }
		if p.tok == token.LBRACE { // file{ ... }
			erro(p, "TODO: %v", tok).debug(1) ; p.checkErrors(true)
			return
		}
	case token.FILE:
		if p.tok == token.LBRACE { // file{ ... }
			erro(p, "TODO: %v", tok).debug(1) ; p.checkErrors(true)
			return
		}
	case token.Bin, token.Oct, token.Int, token.Hex, token.Float:
		if p.tok == token.LBRACE { // answer{...}, bool{...}
			if p.next(true); p.tok == token.RBRACE {
				switch p.step(); tok {
				case token.Bin:   x = MakeBin(pos, 0)
				case token.Oct:   x = MakeOct(pos, 0)
				case token.Int:   x = MakeInt(pos, 0)
				case token.Hex:   x = MakeHex(pos, 0)
				case token.Float: x = MakeFloat(pos, 0.)
				}
			} else if v := p.expr(p, false); v == nil {
				// TODO: true{ expr }, yes{ expr }, ...
				erro(at(p,pos), "%s expects: %v, not %v %v", tok, token.RBRACE, p.tok, p.lit).debug(1)
			} else if p.spaces(); p.tok == token.RBRACE {
				if p.step(); tok == token.Float {
					var n, _ = v.Float(p)
					return MakeFloat(pos, n)
				}
				switch n, _ := v.Integer(p); tok {
				case token.Bin: return MakeBin(pos, n)
				case token.Oct: return MakeOct(pos, n)
				case token.Int: return MakeInt(pos, n)
				case token.Hex: return MakeHex(pos, n)
				}
			}
			return
		}
	case token.ANSWER, token.BOOL, token.NONE:
		if p.tok == token.LBRACE { // answer{...}, bool{...}
			if p.next(true); p.tok == token.RBRACE {
				switch pos := p.Position(); tok {
				case token.ANSWER: x = &answer{valbase{pos},false}
				case token.BOOL: x = &boolean{valbase{pos},false}
				case token.NONE: x = &None{valbase{pos},nil}
				}
				p.step()
				return
			}

			if tok == token.NONE {
				x = &None{valbase{pos}, p.expr(p, false)}
				p.spaces()
				p.expect(token.RBRACE)
				return
			}

			var ( pos = p.Position(); v bool )
			switch p.tok {
			case token.TRUE, token.YES: v = true
			case token.FALSE, token.NO: v = false
			default:
				if t := p.expr(p, false); t != nil {
					v = t.True(p)
				} else {
					erro(at(p,pos), "undef invalid expression: %v, %v", p.tok, p.lit).debug(1)
				}
			}
			p.spaces()
			switch p.expect(token.RBRACE); tok {
			case token.ANSWER: x = &answer{valbase{pos},v}
			case token.BOOL: x = &boolean{valbase{pos},v}
			}
			return
		}
	case token.TRUE, token.YES, token.FALSE, token.NO:
		if p.tok == token.LBRACE { // true{}, false{}, yes{}, no{}
			if p.next(true); p.tok == token.RBRACE {
				switch p.step(); tok {
				case token.TRUE, token.FALSE: x = MakeBoolean(pos, tok == token.TRUE)
				case token.YES , token.NO   : x = MakeAnswer( pos, tok == token.YES)
				}
			} else {
				// TODO: true{ expr }, yes{ expr }, ...
				erro(at(p,pos), "%s expects: %v, not %v %v", tok, token.RBRACE, p.tok, p.lit).debug(1)
			}
			return
		}
	case token.AT, token.DOT, token.DOTDOT: // TODO: parse token.DOT into Qualiword
		return &punctuation{valbase{pos}, tok} // lit = tok.String() // Special bareword.
	default:
		if tok.IsKeyword() { lit = tok.String() } else {
			if true {
				erro(at(p,pos), "%v %v -> %v %v", tok, lit, p.tok, p.lit)
			} else {
				p.expect(token.BAREWORD)
			}
			fail(p.Position(), "parsing: %v %v", p.tok, p.lit)
		}
	}

	return MakeBareword(pos, lit)
}

func (p *parser) selector(ctx Context) (res Value) {
	defer p.setbits(p.setbit(parseSELECT_PROP))
	res = p.expr(ctx, false)
	return
}

func (p *parser) selectExpr(lhs Value) (res Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Select")) }

	var (
		ctx = p.posit()
		tok = p.tok // the arrow '->' or '=>'
		loader = ctx.loader()
		proj = loader.Project()
	)
	p.step() // skip '->' or '=>'

	switch t := lhs.(type) {
	case *selection:
		if v := t.value(at(ctx, t.Position()), ident); isNil(v) {
			erro(ctx, "nil selection: %v", lhs).debug(1)
			return
		} else {
			lhs = v
		}
	case *bareword:
        switch t.string {
        case "use", "usee", "goals", "os", "mode":
			erro(ctx, "$:%s: is obsoleted, use $(.$s) instead", t.string, t.string).debug(1)
        default:
            if name, o := loader.resolveObject(lhs); false {
				erro(at(ctx,lhs.Position()), "resolve '%v' failed", lhs)
				erro(ctx, "parser is here (tok=%s)", tok)
				erro(at(ctx,p.Position()), "parser to go here (tok=%s, lit=%s)", p.tok, p.lit).debug(8)
                return
            } else if !isNil(o) {
				lhs = o
			} else if tok == token.SELECT_PROG2 {
				res = MakeNil(ctx.Position()) // ignore
				return
			} else {
				erro(at(ctx,lhs.Position()), "%v: '%v' is undefined (name=%v, obj=%v)", proj, lhs, name, o)
				erro(ctx, "%v: parser is here (name=%s, tok=%s)", proj, t.string, tok)
				erro(at(ctx,p.Position()), "%v: parser to go here (tok=%s, lit=%s)", proj, p.tok, p.lit).debug(16)
				return
            }
        }
    case *barecomp: // for cases like '.foo'
        if name, o := loader.resolveObject(t); false {
			erro(of(ctx,lhs), "resolve selection object '%v' (%s) error", lhs, name).debug(1)
			return
        } else if !isNil(o) {
			lhs = o
		} else if tok == token.SELECT_PROG2 {
			res = MakeNil(ctx.Position()) // ignore
			return
		} else {
			erro(of(ctx,lhs), "'%v' is undefined", lhs).debug(1)
			return
        }
	}

	if rhs := p.selector(ctx); isNil(rhs) {
		res = MakeNil(ctx.Position())
	} else {
		res = MakeSelection(ctx.Position(), tok, lhs, rhs)
	}

	if (p.tok == token.SELECT_PROP || p.tok == token.SELECT_PROG1 || p.tok == token.SELECT_PROG2) {
		res = p.selectExpr(res) // Continue the selection recursivly.
	}
	return
}

// ----------------------------------------------------------------------------
// Common productions

func (p *parser) isEndOfLine() bool {
	// If there's a comment right after the parsed expression, we break
	// the expression list to treat the end-of-line comment like a LINEND.
	return p.lineComment != nil || p.tok == token.LINEND || p.tok == token.EOF
}

func (p *parser) isEndOfList(lhs bool) bool {
	// If there's a comment right after the parsed expression, we break
	// the expression list to treat the end-of-line comment like a LINEND.
	if p.lineComment != nil || p.tok.IsListDelim() || (lhs && p.tok.IsAssign()) {
		return true
	}
	if (p.bits&parseRecipe != 0) && p.tok == token.RECIPE { // TODO: using p.isRecipeStart()
		return true
	}
	return false
}

func (p *parser) isEndOfURL(lhs bool) bool {
	return p.tok == token.SPACE || p.isEndOfLine() || p.isEndOfList(lhs)
}

func (p *parser) isEndOfDotConcat(lhs bool) bool {
	// Expressions like `FOO.BAR(xxx)` does not count.
	switch p.tok {
	case token.SPACE, token.LPAREN, token.COLON, token.PCON, token.ASSIGN: fallthrough
	case token.SELECT_PROP, token.SELECT_PROG1, token.SELECT_PROG2:
		return true
	}
	return p.isEndOfLine() || p.isEndOfList(lhs)
}

func (p *parser) depends(ctx Context, normal bool) (list []Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Depends")) }
	for p.tok != token.SEMICOLON && p.tok != token.BAR && !p.isEndOfLine() {
		if p.tok == token.COLON { // FIXME: this check is not working!
			// FIXME: detects unexpected colon ':'
			erro(p, "unexpected colon").debug(1)
			p.next(true) // just ignore this colon
		} else if p.spaces(); !p.isEndOfLine() {
			if len(list) == 0 {
				p.bits |= parseDepend0
			} else {
				p.bits &= ^parseDepend0
			}

			var val = p.expr(ctx, false)
			if ctx.checkErrors(true) > 0 {
				erro(ctx, "depend: %T %v", val, val).debug(1)
				return
			}

			if normal {
				if g, y := val.(*Group); y && len(g.Elems) == 1 {
					if g, y = g.Elems[0].(*Group); y {
						p.ruleParams(ctx, g.Elems)
						continue
					}
				}
			}

			list = append(list, val)
			if p.tok == token.SPACE { p.next(true) } //p.spaces()
		}
	}
	return
}

// If lhs is set, result list elements which are identifiers are not resolved.
func (p *parser) values(ctx Context, lhs bool) (list []Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "List")) }
	for p.spaces(); !p.isEndOfList(lhs); p.spaces() {
		var pos = p.pos
		if val := p.expr(ctx, lhs); p.pos == pos {
			erro(p, "nothing: %v %v; %v", p.tok, p.lit, list).debug(1)
			break
		} else { list = append(list, val) }

		// If there's a comment right after the parsed expression, we break
		// the expression list to treat the end-of-line comment like a LINEND.
		if p.lineComment != nil  { break }
		if p.tok == token.LINEND { break }
		if p.tok == token.EOF    { break }
		// if p.spaces(); p.isEndOfList(lhs) { break }
	}
	return
}

func (p *parser) list(ctx Context, lhs bool) *List {
	return MakeList(p.Position(), p.values(ctx, lhs)...)
}

func (p *parser) setRHS(v bool) (old bool) {
	old = p.inRhs; p.inRhs = v; return
}

func (p *parser) left(ctx Context) []Value {
	defer p.setRHS(p.setRHS(false))
	// Line comment of previous lines will break the parsing loop,
	// so we clear the previous line comment
	p.lineComment = nil
	return p.values(ctx, true)
}

func (p *parser) right(ctx Context) []Value {
	defer p.setRHS(p.setRHS(true))
	return p.values(ctx, false)
}

// ----------------------------------------------------------------------------
// Expressions

func (p *parser) group(lhs bool) *Group {
	if t_traverse.enabled { defer un(trace(t_traverse, "Group")) }

	defer p.setbits(p.setbit(parseGroup))
	p.clearbit(parseCall)

	var ctx = p.posit()
	p.next(true)

	var elems, converted = p.values(ctx, false), false
	for p.tok != token.RPAREN && p.tok != token.EOF {
		// if p.tok == token.COMMA { warn(ctx, "%020b: %v %v", p.bits, p.tok, p.lit).debug(1) }
		// if p.tok == token.COMMA { p.next(true) }
		switch p.tok {
		case token.BAR, token.COMMA, token.SEMICOLON:
			elems = append(elems, p.punctuation())
			p.spaces()
		}
		var next *List
		next = p.list(ctx, false)
		if !converted {
			elems = []Value{ MakeList(p.Position(), elems...), next }
			converted = true
		} else {
			elems = append(elems, next)
		}
	}
	p.expect(token.RPAREN)
	return MakeGroup(ctx.Position(), elems...)
}

func (p *parser) argumentedExpr(x Value) *Argumented {
	if t_traverse.enabled { defer un(trace(t_traverse, "Argumented")) }

	defer p.setbits(p.setbit(parseGroup))
	p.clearbit(parseCall)

	var ctx = p.posit()
	p.next(true) // skip token.LPAREN

	var a = []Value{ p.list(ctx, false) }
	for p.tok != token.RPAREN && p.tok != token.LINEND && p.tok != token.EOF {
		switch p.tok {
		case token.COMMA: p.next(true) // skip token.COMMA
		case token.BAR, token.SEMICOLON:
			if false {
				a = append(a, p.punctuation())
				p.spaces()
			} else {
				erro(ctx, "unexpected punctuation: %v", p.tok).debug(1)
			}
		}
		a = append(a, p.list(p.posit(), false))
	}
	p.expect(token.RPAREN)
	return MakeArgumented(x, a...)
}

func (p *parser) globMeta() (x *GlobMeta) {
	pos, tok := p.Position(), p.tok
	p.step()
	return MakeGlobMeta(pos, tok)
}

func (p *parser) globRange() (x *GlobRange) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Glob")) }

	var ctx = p.posit()
	p.expect(token.LBRACK) // skip '['

	chars := p.expr(ctx, false)
	p.expect(token.RBRACK) // skip ']'

	return MakeGlobRange(ctx.Position(), chars)
}

func (p *parser) globExpr(x Value) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "Glob")) }

	var pos = p.Position()
	var ctx = at(p, pos)

	var components []Value
	if !isNil(x) { components = []Value{ x } }

	// avoid nesting glob expressions
	defer p.setbits(p.setbit(parseGLOB))
ForGlobTok:
	for {
		if p.lineComment != nil { break ForGlobTok }
		switch p.tok {
		case token.PCON, token.RPAREN, token.COMMA, token.SPACE, token.LINEND, token.EOF:
			break ForGlobTok
		case token.STAR, token.QUE: // * ?
			x = p.globMeta()
		case token.LBRACK:
			// FIXME: '[...]' has been used for modifier expressions
			x = p.globRange()
		default:
			// FIXME: escaped glob metas/chars
			x = p.expr(ctx, false)
		}
		components = append(components, x)
	}
	if components == nil {
		erro(ctx, "nil glob expression (tok=%v, lit=%v)", p.tok, p.lit)
	}
	return MakeGlobPattern(pos, components...)
}

func (p *parser) percExpr(lhs bool, x Value) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "Perc")) }

	// avoid nesting percent expressions
	defer p.setbits(p.setbit(parsePERC))

	var (
		pos = p.pos
		y Value
	)
	if p.step(); pos+1 == p.pos { // joint, e.g. '%.o', but skip '% .o'
		switch p.tok {
		case token.COLON, token.COLON2,
			token.LPAREN, token.RPAREN,
			token.LBRACK, token.RBRACK,
			token.PCON,   token.SEMICOLON,
			token.COMMA,  token.SPACE,
			token.LINEND:
		case token.PERC: // %%
			p.step() // consume the second %
			position := p.Position()
			perc2 := MakePercPattern(position, nil, nil)
			if pos+2 == p.pos {
				switch p.tok {
				case token.PERC: // %%%
					erro(p, "too many %")
				case token.PCON: // FIXES: %%/xxx -> Path(%% xxx)
					x = MakePercPattern(position, x, perc2)
					return p.path(lhs, x)
				case token.COLON,    token.COLON2,
					token.LPAREN,    token.RPAREN,
					token.LBRACK,    token.RBRACK,
					token.LBRACE,
					token.SEMICOLON, token.COMMA,
					token.SPACE,     token.LINEND:
				default:
					var (
						yy = p.expr(p, false)
						_, ok = yy.(*Path)
					)
					if ok { erro(p, "incorrect: %v, %v", x, yy) }
					assert(!ok, "the second part of aaa%%bbb/foo/bar parsed incorrectly as path")
					perc2.Suffix = yy
				}
			}
			y = perc2
		default:
			y = p.expr(p, false)
		}
	}
	return MakePercPattern(p.loc(pos), x, y)
}

func (p *parser) regexp(ctx Context) (x Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Regexp")) }
	defer p.setbits(p.setbit(parseREXP)) // avoid nesting percent expressions

	var rx string
	ForRX: for p.expect(token.LBRACE); p.tok != token.EOF; p.scan() {
		if false { info(p, "regexp: %v '%v': %s\n", p.tok, p.lit, rx) }

		var esc bool
		if esc = p.tok == token.ESCAPE; esc {
			if rx += "\\" + p.lit; p.lit == "Q" {
				for p.scan(); p.tok != token.EOF; p.scan() {
					if p.tok == token.ESCAPE {
						if rx += "\\" + p.lit; p.lit == "E" {
							break
						}
					} else if p.lit != "" {
						rx += p.lit
					} else if s := p.tok.String(); s != "" {
						rx += s
					} else {
						erro(p, "regexp: %v '%v' ; %s\n", p.tok, p.lit, rx).debug(1)
					}
				}
			}
			continue
		}

		switch p.tok {
		case token.RBRACE: p.scan() ; break ForRX
		case token.LBRACE:
			rx += "{"
			for p.expect(token.LBRACE); p.tok != token.EOF; p.scan() {
				if p.tok == token.RBRACE { break } else
				if p.lit != "" { rx += p.lit } else
				if s := p.tok.String(); s != "" { rx += s } else {
					erro(p, "regexp: %v '%v' ; %s\n", p.tok, p.lit, rx).debug(1)
				}
			}
			rx += "}"
		default:
			if p.lit != "" { rx += p.lit } else
			if s := p.tok.String(); s != "" { rx += s } else {
				erro(p, "regexp: %v '%v' ; %s\n", p.tok, p.lit, rx).debug(1)
			}
		}
	}
	warn(p, "todo: regexp: %s; %v %v", rx, p.tok, p.lit).debug(1)
	return &RegexpPattern{valbase{p.Position()}} // TODO: correct regexp pattern value
}

func (p *parser) pair(x Value) *Pair {
	if t_traverse.enabled { defer un(trace(t_traverse, "Pair")) }

	var ctx = p.posit()
	p.step()

	var y Value
	if p.isEndOfList(false) {
		y = MakeNil(ctx.Position())
	} else {
		y = p.expr(ctx, false)
	}
	return MakePair(ctx.Position(), x, y)
}

func (p *parser) flagExpr(lhs bool) *Flag {
	if t_traverse.enabled { defer un(trace(t_traverse, "Flag")) }

	var ctx = p.posit()
	p.step() // skip dash '-'

	var x Value
	// Flag expressions, excluding "-)" "-]" "-}" "-\n", "-=", "-:", etc.
	if p.isEndOfLine() || p.isEndOfList(false) ||
		p.tok == token.SPACE || p.tok == token.RECIPE {
		x = MakeNil(ctx.Position())
	} else if false {
		x = p.expr(ctx, false)
	} else {
		x = p.unary(ctx, false)
	}
	if x == nil { erro(ctx, "nil flag name").debug(1) }
	return MakeFlagValue(ctx.Position(), x)
}

func (p *parser) negExpr(ctx Context, lhs bool) *negative {
	if t_traverse.enabled { defer un(trace(t_traverse, "Negative")) }
	p.expect(token.EXC)
	return Negative(p.expr(ctx, lhs))
}

func (p *parser) punctuation() *punctuation {
	if t_traverse.enabled { defer un(trace(t_traverse, "punctuation")) }
	var pos, tok = p.Position(), p.tok
	p.step()
	return &punctuation{valbase{pos}, tok}
}

func (p *parser) literal(lhs bool) (v Value) {
	var ctx, tok, lit = p.posit(), p.tok, p.lit
	p.step()

	// ESCAPE is handled in value.EscapeChar
	defer checkFailure(ctx) // panics from parse{int,float,hex,...}
    switch position := ctx.Position(); tok {
    case token.BAR: erro(ctx, "`|` is deprecated, changed the modifiers!")
    case token.BIN:      v = ParseBin(position, lit)
    case token.OCT:      v = ParseOct(position, lit)
    case token.INT:      v = ParseInt(position, lit)
    case token.HEX:      v = ParseHex(position, lit)
    case token.FLOAT:    v = ParseFloat(position, lit)
    case token.DATETIME: v = ParseDateTime(position, lit)
    case token.DATE:     v = ParseDate(position, lit)
    case token.TIME:     v = ParseTime(position, lit)
    case token.URI:      v = ParseURL(position, lit)
    case token.BAREWORD: v = MakeBareword(position, lit)
    case token.STRING:   v = MakeString(position, lit)
    case token.ESCAPE:   v = /*MakeString*/MakeRaw(position, EscapeChar(lit))
    case token.RAW:      v = MakeRaw(position, lit)
    default: unreachable()
    }
	return
}

func (p *parser) compound(lhs bool) *Compound {
	var (
		ctx = p.posit()
		lpos = p.pos
		elems []Value
	)
	p.step()

	defer p.setbits(p.setbit(parseCompound))

	for p.tok != token.EOF && p.tok != token.COMPOSED && p.tok != token.LINEND {
		if p.tok == token.RAW {
			elems = append(elems, p.literal(false))
		} else {
			elems = append(elems, p.expr(ctx, false))
		}
	}
	p.expect(token.COMPOSED)
	return MakeCompound(p.loc(lpos), elems...)
}

// Parses dot composing expressions (TODO: check against file extensions).
//   .foo
//   .'foo'
//   ."foo"
//   .(foo)
//   ..foo
//   ..'foo'
//   .foo.bar
func (p *parser) dot(lhs bool, x Value) (res *barecomp) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Dot")) }

	defer p.setbits(p.setbit(parseDOT))

	var comp *barecomp
	if x == nil { panic(fmt.Sprintf("nil dot (tok=%v)", p.tok)) }
	if comp, _ = x.(*barecomp); comp == nil {
		comp = MakeBarecomp(x.Position())//(p.Position())
		comp.Elems = append(comp.Elems, x)
	}

	var ctx = p.posit()
	for /*comp.End() == p.pos && */!p.isEndOfDotConcat(lhs) {
		comp.Combine(ctx, p.composite(ctx, false))
		if p.tok == token.DOT /*&& comp.End() == p.pos*/ {
			// var dot = MakeBareword(p.Position(), ".") // TODO: parse to Qualiword instead
			var dot Value = &punctuation{valbase{p.Position()}, p.tok}
			comp.Elems = append(comp.Elems, dot)
			p.step() // '.'
		}
	}

	// FIXME: *.o => obj
	//   BUG: barecomp{Glob . KeyValueExpr}
	//   FIX: KeyValueExpr{barecomp, bareword}

	return comp
}

func makePathPun(ctx Context, tok token.Token) *PathPun {
	var r rune
    switch tok {
    case 0:            r = 0 // the tailing empty segment after '/', e.g. /foo/bar/
    case token.PCON:   r = '/' // TODO: should be NONE
    case token.TILDE:  r = '~'
    case token.DOT:    r = '.'
    case token.DOTDOT: r = '^' // 
    default:
		erro(ctx, "invalid path segment `%v`", tok)
		return nil
    }
	return MakePathPun(ctx.Position(), r)
}

func (p *parser) path(lhs bool, start Value) *Path {
	if t_traverse.enabled { defer un(trace(t_traverse, "Path")) }

	defer p.setbits(p.setbit(parsePATH))

	var (
		position = start.Position() //p.Position()
		ctx = at(p, position)
		path *Path
		ok bool
	)
	if start == nil {
		erro(ctx, "bad closure/delegate name").debug(1)
		p.step()
		return MakePath(position) // empty path
	} else if path, ok = start.(*Path); !ok {
		path = MakePath(position, start)
	}

BuildPath:
	for p.tok == token.PCON {
		var pos = p.Position() // skips repeated '/' sequence
		for p.step(); p.tok == token.PCON; p.step() { pos = p.Position() }
		switch p.tok {
		case token.RPAREN, token.LPAREN, token.RBRACE, token.LBRACE,
			 token.COMMA, token.SPACE, token.LINEND:
			// Encountered the tailing '/', append 'zero' segment.
			path.Elems = append(path.Elems, MakePathPun(pos, 0))
			break BuildPath
		}

		var x = p.composite(ctx, false)
		path.Elems = append(path.Elems, x)
		if p.tok == token.SPACE || p.isEndOfLine() {
			break BuildPath
		}
	}
	return path
}

func isKnownURLScheme(s string) (result bool) {
	switch strings.ToLower(s) {
	case "file", "http", "https", "ftp", "ftps", "mailto":
		result = true
	}
	return
}

func (p *parser) url(ctx Context, lhs bool, scheme Value) (res Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "URL")) }

	defer p.setbits(p.setbit(parseURL))

	var (
		url = &URL{ Scheme:scheme }
		colon1 = p.expect(token.COLON) // consumes ':'
		colon2 = token.NoPos
		//colon3 = token.NoPos
		a = token.NoPos // @
	)

	if p.tok == token.PCON {
		p.step() // the first '/'
		if p.tok == token.PCON {
			p.expect(token.PCON) // the second '/'
		} else {
			erro(ctx, "TODO: URL path: %v (%T) (next: %s (%s))", scheme, scheme,  p.tok, p.lit).debug(1)
			res = MakeNil(p.Position())
			return
		}
	} else if !p.isEndOfURL(lhs) {
		erro(at(ctx, p.loc(colon1)), "TODO: URL: %v (%T) (next: %s (%s))", scheme, scheme, p.tok, p.lit).debug(1)
		res = MakeNil(p.Position())
		return
	}

	if !p.isEndOfURL(lhs) {
		userOrHost := p.composite(ctx, false)
		if p.tok == token.COLON {
			url.Username, colon2 = userOrHost, p.pos
			p.step() // ':'
			if p.tok != token.AT && p.tok != token.PCON && !p.isEndOfURL(lhs) {
				url.Password = p.composite(ctx, false)
			}
		} else {
			url.Host = userOrHost
		}
		if p.tok == token.AT {
			p.step() // '@'
		}
	}
	if url.Host == nil && colon2 == token.NoPos && a == token.NoPos && !p.isEndOfURL(lhs) {
		url.Host = p.composite(ctx, false)
		if p.tok == token.COLON {
			//colon3 = p.pos
			p.step() // ':'
			if p.tok != token.SPACE && p.tok != token.LINEND {
				url.Port = p.composite(ctx, false)
			}
		}
	}
	if p.tok == token.PCON {
		url.Path = p.path(lhs, makePathPun(ctx, p.tok))
	}
	// scanning '#' as token.HASH instead of token.COMMENT
	defer p.scanner.SetBits(p.scanner.CommentsOff())
	if p.tok == token.QUE {
		p.step() // '?'
		if p.tok != token.HASH && !p.isEndOfURL(lhs) {
			url.Query = p.composite(ctx, false)
		}
	}
	if p.tok == token.HASH {
		p.step() // '#'
		if !p.isEndOfURL(lhs) {
			url.Fragment = p.composite(ctx, false)
		}
	}
	return url
}

func (p *parser) closuredelegate() (result Value) {
	if t_traverse.enabled {	defer un(trace(t_traverse, "ClosureDelegate")) }

	defer p.setbits(p.setbit(parseCall))

	var (
		ctx = p.posit()
		loader = ctx.loader()
		scope = loader.Scope()
		proj = loader.Project()
		tok = p.tok
		resolved Value // Object or *selection
		rest []Value
	)
	resolveConfig := func(val Value, name string) (obj Object) {
		if c := proj.configure; c != nil { obj = c.resolveObject(ctx, name) }
		return
	}

	const allowClosureName = true
	resolveObject := func(lPos Position, lTok token.Token, name Value) (str string, obj Value, okay bool) {
		if a, y := name.(*Argumented); y { name = a.value }
		if sel, y := name.(*selection); y {
			if sel == nil {
				erro(at(ctx,name.Position()), "nil selection: %v", name).debug(1)
			} else if v := sel.value(ctx, plain); v == nil {
				erro(of(ctx,name), "`%v` not selected nil value", sel).debug(1)
			} else if u, y := v.(unresolved); y {
				obj, okay = u, true
			} else if u, y := v.(unexpanded); y {
				if obj, okay = u.Value.(unresolved); !okay {
					obj, okay = unresolved{u.Value, proj}, true
				}
			} else if s, y := v.(selected); !y {
				erro(of(ctx,name), "`%v` not selected: %v (%T)", sel, v, v).debug(1)
			} else if obj, okay = s.Value.(Object); !okay {
				// return // just use the selected value
			}
			switch lTok {
			case token.LPAREN:
				if _, ok := obj.(Caller); !ok {
					v := sel.value(ctx, plain)
					erro(of(ctx,name), "selected object '%v' is not callable: %T %v ; %T %v ; %T %v", name, obj, obj, sel.o, sel.o, v, v).debug(16)
				}
			case token.LBRACE:
				if _, ok := obj.(Executer); !ok {
					erro(of(ctx,name), "selected object '%v' is not executer: %T %v", name, obj, obj).debug(1)
				}
			}
			return
		}

		if val := name.expand(ctx, ident); val != name {
			if u, y := val.(unexpanded); y {
				return str, unresolved{u.Value, proj}, true
			} else { name = val }
		}

		switch lTok {
		case token.LPAREN:
			if allowClosureName && name.expandible(ctx, expandDelegate|expandClosure) {
				return str, unresolved{name, proj}, true // recursive delegation or closure
			} else if str, resolved = loader.resolveObject(name); false {
				erro(at(ctx,name.Position()), "resolve '%v' (%s) failed", name, str).debug(1)
				return
			} else if str == "" {
				erro(at(ctx,name.Position()), "name '%v' is empty", name).debug(1)
				return
			} else if isNil(resolved) {
				if p.isIncludingConf {
					// Create an empty Def if it's referred in configuration.sm.
					def, _ := loader.def(name.Position(), str)
					def.origin = DefConfRef
					obj, okay = def, true
					return
				}
				for _, a := range p.autos {
					if okay = a.name == str; okay {
						obj = a
						return
					}
				}
				if tok != token.CLOSURE && p.autop != nil {
					var d = &def{
						origin: DefAuto, knownobject: knownobject{
							objbase{valbase{name.Position()}, scope, scope.project},
							str,
						},
					}
					p.autos = append([]*def{d}, p.autos...)
					obj, okay = d, true
					return
				}
				if obj = resolveConfig(name, str); !isNil(obj) {
					okay = true
					return
				} else if tok.IsClosure() || name.expandible(ctx, expandClosure|expandDelegate) ||
					refdef(ctx, name, defany) {
					obj, okay = unresolved{name, proj}, true // recursive delegation or closure
					return
				} else if p.bits&parseUndefValue != 0 {
					obj, okay = unresolved{undef{name}, proj}, true
					return
				}

				erro(of(ctx,name), "%v: %T %v -> '%s', is nil", proj, name, name, str)
				errostack(of(ctx,name), 32, "%v: %v", proj, ctx).debug(128)
				if ctx.checkErrors(true)>0 { /* fail(ctx.Position(), "undefined %v", name) */ }
			} else if obj, okay = resolved.(*selection); okay {
				return
			} else if obj, okay = resolved.(*Builtin); okay {
				return
			} else if caller, _ := resolved.(Caller); caller == nil {
				erro(at(ctx,lPos), "%v is not callable: %T", name, resolved).debug(16)
			} else if obj, okay = caller.(Object); !okay {
				erro(at(ctx,lPos), "%v is not object: %T", name, resolved).debug(16)
			} else if isNil(obj) {
				erro(at(ctx,lPos), "%v is nil: %T", name, resolved).debug(16)
			} else {
				return
			}
		case token.LBRACE:
			if allowClosureName && name.expandible(ctx, expandDelegate|expandClosure) {
				erro(of(ctx,name), "%v: name '%v' (%T) is closured", proj, name, name).debug(1)
				return
			} else if resolved = loader.resolveEntries(name); isNil(resolved) {
				if name.expandible(ctx, plain) {
					var s = name.Strval(ctx)
					erro(of(ctx,name), "resolved '%v' (aka. %s) is nil (project=%v)", name, s, proj).debug(1)
				} else {
					erro(of(ctx,name), "resolved '%v' is nil (project=%v)", name, proj).debug(1)
				}
			} else if exe, _ := resolved.(Executer); exe == nil {
				erro(at(ctx,lPos), "resolved '%v' of '%T' is not Executer", name, resolved).debug(1)
			} else if obj, okay = exe.(Object); !okay || isNil(obj) {
				erro(at(ctx,lPos), "resolved Executer '%v' of '%T' is not Object", name, resolved).debug(1)
			}
		}
		return
	}

	var (
		name Value
		nameStr string
		tokLp token.Token
		opts []Value
		obj Value
		okay bool
	)
	switch p.step(); p.tok {
	case token.LPAREN, token.LBRACE: // $(...), ${...}
		var posLp = p.Position()
		tokLp = p.tok ; p.step() // skips LPAREN, LBRACE

		var posName = p.Position()

		switch p.tok {
		case token.SPACE:
			erro(at(ctx,posName), "unexpected spaces").debug(1)
			return MakeNil(posName)
		case token.COLON:
			p.step();  posName = p.Position()
			warn(at(ctx,posName), "colon").debug(1)
		}

		if name = p.expr(ctx, false); isNil(name) {
			erro(at(ctx,posName), "%v: parsed name is nil", proj).debug(1)
		} else if a, y := name.(*Argumented); y {
			var args = merge(a.args...)
			for _, v := range args {
				if p, y := v.(*Pair);  y { v = p.Key }
				if _, y := v.(*Flag); !y {
					erro(of(ctx,v), "%v: not a Flag: %T %v", proj, v, v).debug(1)
				}
			}
			if true { name, opts = a.value, args }
		}

		if isNil(name) {/* error */} else
		if !allowClosureName && name.expandible(ctx, expandClosure|expandDelegate) {
			erro(at(ctx,posName), "%v: name '%v' (%T) is closured", proj, name, name).debug(1)
		} else if nameStr, obj, okay = resolveObject(posLp, tokLp, name); !okay {
			erro(at(ctx,posName), "%v: name '%v' is unidentified", proj, name).debug(1)
		}

		if false { if name.String() == "name?" {
			warnstack(ctx, 3, "%v %v ; %T %v", name, name, obj, obj).debug(1)
		}}
		if false && name.String() == ".test$1" {
			v := name.expand(ctx, plain)
			warnstack(of(ctx,name), 3, "%v: %T %v -> %T %v -> %T %v", nameStr, name, name, obj, obj, v, v).debug(1)
		}
		if false { if def, y := obj.(*def); y && name.String() == ".test.v2" {
			v := name.expand(ctx, plain)
			warnstack(of(ctx,name), 3, "%v: %T %v -> %T %v -> %T %v ; %v", nameStr, name, name, obj, obj, v, v, def.origin).debug(1)
		}}

		if  (tokLp == token.LPAREN && p.tok != token.RPAREN) ||
			(tokLp == token.LBRACE && p.tok != token.RBRACE) {
			var autos []*def
			var savedAutos = p.autos
			var savedAutop = p.autop
			if nameStr == "" {
				// keep on...
			} else if nameStr == "auto" {
				if tokLp != token.LPAREN {
					erro(at(ctx,posLp), "%v: auto: incorrect left paren", proj).debug(1)
				}
				p.spaces() // skip the imediate spaces
				var al = p.list(ctx, false)
				if rest = append(rest, al); p.tok == token.COMMA { p.next(true) }
				for _, val := range merge(al) {
					var pos = val.Position()
					var s string
					if kv, y := val.(*Pair); y {
						s = kv.Key.Strval(ctx)
						val = kv.Value
					} else {
						s = val.Strval(ctx)
						val = nil
					}
					if s == "" { erro(at(ctx,pos), "%v: auto: %v is empty", proj, val).debug(1) }
					var d = &def{
						origin: DefAuto, value: val, knownobject: knownobject{
							objbase{valbase{pos}, scope, scope.project}, s,
						},
					}
					d.position = posName
					autos = append(autos, d)
				}
				if tok != token.CLOSURE {
					p.autop = &posName // NOTE: this enables auto-delegation
				}
			} else if nameStr == "foreach" {
				var d = &def{
					origin: DefAuto, knownobject: knownobject{
						objbase{valbase{posName}, scope, scope.project}, "_",
					},
				}
				d.position = posName
				autos = append(autos, d)
			}

			if autos != nil { p.autos = append(autos, p.autos...) }
			if savedBits := p.bits; nameStr == "case" {
				rest = append(rest, p.list(ctx, false))
				p.bits |= parseUndefValue
				for ; p.tok == token.COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx, false))
				}
				p.bits = savedBits
			} else if nameStr == "and" {
				p.bits |= parseUndefValue
				for rest = append(rest, p.list(ctx, false)); p.tok == token.COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx, false))
				}
				p.bits = savedBits
			} else if nameStr == "or" {
				p.bits |= parseUndefValue
				for rest = append(rest, p.list(ctx, false)); p.tok == token.COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx, false))
				}
				p.bits = savedBits
			} else {
				for rest = append(rest, p.list(ctx, false)); p.tok == token.COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx, false))
				}
			}
			p.autos = savedAutos
			p.autop = savedAutop
		}

		switch tokLp {
		case token.LPAREN: p.expect(token.RPAREN)
		case token.LBRACE: p.expect(token.RBRACE)
		}

	default:
		if position := p.Position(); tok != token.CLOSURE { // $(...), disabled $name.
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v` or `%v` or quotes", token.LPAREN, token.LBRACE).debug(1)
			return MakeNil(position)
		} else if p.tok == token.STRING || p.tok == token.COMPOUND {
			var posLp = p.Position()
			tokLp = p.tok

			// &'xxxx' or &"xxxx"
			if name = p.expr(ctx, false); isNil(name) {
				erro(at(ctx,posLp), "parsed name is nil").debug(1)
			} else if name.expandible(ctx, expandClosure) {
				erro(at(ctx,name.Position()), "name '%v' (%T) is closured (project=%v)", name, name, proj).debug(1)
			} else if nameStr, obj, okay = resolveObject(posLp, tokLp, name); !okay {
				erro(at(ctx,name.Position()), "name '%v' is unidentified", name).debug(1)
			}
		} else {
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v`, `%v` or quotes, not %v %v", token.LPAREN, token.LBRACE, p.tok, p.lit).debug(1)
			return MakeNil(position)
		}
	}

	if isNil(obj) && proj.plugin != nil && proj.pluginScope != nil {
		if nameStr == "" && !isNil(name) { nameStr = name.Strval(ctx) }
		if nameStr == "" {
			erro(at(ctx,name.Position()), "strval name '%v' is empty", name).debug(1)
		} else {
			obj = proj.pluginScope.Lookup(nameStr)
		}
	}

	if true && opts == nil && len(rest) > 0 {
		// NOTE: Options (flags) in args are deprecated by $(wildcard(-foo) ...)
		for _, v := range merge(rest[0]) {
			if p, y := v.(*Pair); y { v = p.Key }
			if _, y := v.(*Flag); y { warn(of(ctx,v), "%v", v).debug(1) } else { break }
		}
	}

	if position := ctx.Position(); tok.IsDelegate() {
		if isNil(obj) { erro(at(ctx,name.Position()), "resolved '%v' is nil (%T %v, tok=%v)", name, resolved, resolved, tok).debug(1) }
		return MakeDelegate(position, tokLp, obj, opts, rest...);
	} else {
		if isNil(obj) { erro(at(ctx,name.Position()), "resolved '%v' is nil (%T %v), shall be 'unresolved' (tok=%v)", name, resolved, resolved, tok).debug(1) }
		return MakeClosure(position, tokLp, obj, opts, rest...);
	}
}

func (p *parser) specialClosureDelegate(ctx Context, lhs bool) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "SpecialClosureDelegate")) }

	var obj Object
	var resolved Value
	var pos, tok, s = p.pos, p.tok, p.lit
	var position = p.loc(pos)
	p.step()


	var loader = ctx.loader()
	if c := s[0]; /*p.bits&parseDefineClause != 0*/true &&
		len(s) == 1 && (('0' <= c && c <= '9') /*|| c == '_'*/) {
		var scope = loader.Scope()
		for _, a := range p.autos {
			if a.name == s { obj = a ; break }
		}
		if obj == nil {
			var d = &def{
				origin: DefAuto, knownobject: knownobject{
					objbase{valbase{position}, scope, scope.project},
					s,
				},
			}
			p.autos = append([]*def{d}, p.autos...)
			obj = d
		}
	} else if w := MakeBareword(position, s); s == "_" {
		for _, a := range p.autos {
			if a.name == s { obj = a ; goto DashNxt }
		}
		if obj == nil && p.bits&parseTemplateBlock != 0 {
			if _, resolved = loader.resolveObject(w); resolved != nil {
				obj, _ = resolved.(Object)
			}
		}
	DashNxt:
	} else if _, resolved = loader.resolveObject(w); resolved == nil {
		erro(ctx, "'%v' is undefined (autos: %v)", s, p.autos).debug(16)
		return MakeNil(position)
	} else if def, y := resolved.(Caller); def == nil || !y {
		erro(of(ctx,resolved), "'%v' is not callable: %T", s, resolved).debug(6)
		return MakeNil(position)
	} else if obj, y = def.(Object); !y {
		erro(of(ctx,resolved), "'%v' is not object: %T", s, def).debug(6)
		return MakeNil(position)
	}

	if isNil(obj) {
		erro(ctx, "resolved '%v' is <nil>: %v (%T)", s, resolved, resolved).debug(1)
		return MakeNil(position)
	} else if tok.IsDelegate() {
		return MakeDelegate(position, tok, obj, nil);
	} else {
		return MakeClosure(position, tok, obj, nil);
	}
}

func (p *parser) unary(ctx Context, lhs bool) (x Value) {
	if t_traverse.enabled && false { defer un(trace(t_traverse, "Unary")) }

	switch p.tok {
	case token.BAREWORD, token.AT:
		return p.bare(lhs)

	case token.BIN, token.OCT, token.INT, token.HEX, token.FLOAT,
		token.DATETIME, token.DATE, token.TIME, token.URI,
		/*token.RAW,*/ token.STRING, token.ESCAPE:
		return p.literal(lhs)

	case token.COMPOUND:
		return p.compound(lhs)

	case token.DELEGATE, token.CLOSURE: // delegate, closure
		return p.closuredelegate()

	case token.LPAREN:
		return p.group(lhs)

	case token.COMMA:
		if p.bits&parseCall == 0 {
			var tok, pos = p.tok, p.pos
			p.step()
			return &punctuation{valbase{p.loc(pos)}, tok}
		}

	case token.TILDE, token.DOT, token.DOTDOT: // ~ . ..
		var str = p.tok.String()
		tok, pos, end := p.tok, p.pos, p.pos+token.Pos(len(str))
		position := p.loc(pos)
		if p.step(); end != p.pos { // FIXME: ~user
			// '~', '.' or '..' used as bareword
			return &punctuation{valbase{position}, tok}
		} else if p.tok == token.PCON { // check /
			return p.path(lhs, makePathPun(at(ctx, position), tok))
		} else if tok == token.DOT || tok == token.DOTDOT { // TODO: parse to Qualiword instead
			x = &punctuation{valbase{position}, tok}
			if p.bits&parseDOT == 0 { x = p.dot(lhs, x) }
			return
		} else if tok == token.TILDE { // TODO: ~user
			return makePathPun(at(ctx, position), tok)
		} else {
			erro(ctx, "unexpected path: %v", tok).debug(1)
			return MakeNil(position)
		}

	case token.PCON: // The root of the path
		return p.path(lhs, makePathPun(ctx, p.tok))

	case token.LBRACK:
		return p.modifiers(ctx)

	case token.STAR, token.QUE/*, token.LBRACK*/: // * ? [
		return p.globExpr(nil) // (ie. no prefix)

	case token.PERC: // %bar (ie. no prefix)
		return p.percExpr(lhs, nil)

	case token.LBRACE: // TODO: regexp: {^.*}   or token.REGEXP
		return p.regexp(ctx)

	case token.MINUS:
		return p.flagExpr(lhs)

	case token.EXC:
		return p.negExpr(ctx, lhs)

	case token.SEMICOLON, token.BAR, token.PLUS:
		return p.punctuation()

	default:
		if p.tok.IsClosure() || p.tok.IsDelegate() {
			return p.specialClosureDelegate(ctx, lhs)
		} else if p.tok.IsKeyword() { // keywords here are barewords
			return p.bare(lhs)
		}
	}

	var s = p.scanner.GetState()
	if p.lineComment != nil {
		for _, comment := range p.lineComment.List {
			erro(at(p,comment.Pos), "# %s", comment.Text)
		}
	}
	erro(p, "bad unary expression '%v' (lit=%s, left=%v, scan=%v)", p.tok, p.lit, lhs, s).debug(32)

	p.step() // go to the next token
	return MakeNil(p.Position())
}

func (p *parser) isParametersGroup(x Value) (res bool) {
	if p.bits&parseDepend0 != 0 {
		if g, y := x.(*Group); y && len(g.Elems) == 1 {
			_, res = g.Elems[0].(*Group)
		}
	}
	return
}

func (p *parser) composite(ctx Context, lhs bool) (x Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Composed")) }

	switch x = p.unary(ctx, lhs); p.tok { // check composible expressions
	case token.SELECT_PROP, token.SELECT_PROG1, token.SELECT_PROG2: // foo->bar  foo=>bar  foo~>bar
		if p.bits&parseNoSelect == 0 {
			// accepts 'foo=>bar', but 'foo => bar' is different
			x = p.selectExpr(x); break
		}
	case token.LBRACK: // xxx[(foo ...)]
		if p.isParametersGroup(x) { break }
		if p.bits&parseModifier == 0 {
			// FIXME: compose lhs x
			if m := p.modifiers(ctx); false {
				erro(of(ctx,m), "composing modifiers is ignored (unimplemented yet)")
			} else {
				errostack(of(ctx,m), 3, "composing modifiers is ignored (%T %v)", x, x).debug(12)
			}
		}
	case token.STAR, token.QUE/*, token.LBRACK*/: // foo*bar foo?bar foo[a-z]bar
		if p.bits&parseNoGlob == 0 {
			x = p.globExpr(x)
		}
	case token.PERC: // foo%bar
		// FIXME: %/foo/bar -> Path(% foo bar)
		if p.bits&parseNoPerc == 0 {
			x = p.percExpr(lhs, x)
		}
	case token.DOT: // foo.bar.baz.o
		// FIXME: push bits when parsing $(...)
		if p.bits&parseDOT == 0 { // TODO: parse to Qualiword
			x = p.dot(lhs, x)
		}
	case token.PCON: // ie. subdir/in/somewhere
		if p.bits&parseNoPath == 0 {
			// Path expressions, except '-I/path/to/include'
			switch x.(type) {
			case *Flag: // By pass expressions like -I/foo/bar.
			default: x = p.path(lhs, x)
			}
		}
	case token.COLON:
		if (p.bits&parseRecipe != 0 || !lhs) && p.bits&parseNoURL == 0 {
			if isKnownURLScheme(x.Strval(at(ctx, p.Position()))) {
				x = p.url(ctx, lhs, x)
			}
		}
	}
	return
}

func (p *parser) text(ctx Context) (res []Value) {
	if false && t_traverse.enabled { defer un(trace(t_traverse, "Text")) }
	for p.tok != token.EOF {
		if p.tok == token.SPACE { p.next(true) } else {
			res = append(res, p.expr(ctx, false))
			if ctx.checkErrors(true) > 0 {
				warn(ctx, "parse text got %d errors", ctx.totalErrors()).debug(16)
				if options.failOnErrors { fail(p.Position(), "fail by %d errors", ctx.totalErrors()) }
			}
		}
	}
	return
}

func (p *parser) expr(ctx Context, lhs bool) (x Value) {
	if false && t_traverse.enabled { defer un(trace(t_traverse, "Expression")) }

	var tok, lit = p.tok, p.lit
	if x = p.composite(ctx, lhs); x == nil {
		erro(p, "invalid (tok=%v,%v; next=%v,%v)", tok, lit, p.tok, p.lit).debug(6)
		return
	} else if lhs && p.tok.IsAssign() { return
	} else if p.isParametersGroup(x)  { return }

SwitchCompose:
	switch p.tok {
	case token.ASSIGN: // Example: '*.o = obj'
		if !lhs && p.bits&parseNoPair == 0 { x = p.pair(x) }
		return

	case token.SELECT_PROP, token.SELECT_PROG1, token.SELECT_PROG2:
		if p.bits&parseNoSelect == 0 {
			x = p.selectExpr(x)
			goto SwitchCompose // For example: foobar⇒run(-gen)
		}
		return

	case token.LPAREN:
		if p.bits&parseNoArg == 0 {
			if false {
				if _, ok := x.(*Argumented); ok { erro(ctx, "nested argumentation") }
			}
			if x = p.argumentedExpr(x); !isNil(x) {
				goto SwitchCompose
			}
		}
		return

	case token.PCON:
		if p.bits&parseNoPath == 0 {
			// Path expressions, except '-I/path/to/include'
			switch x.(type) {
			case *Flag: // By pass expressions like -I/foo/bar.
			default: x = p.path(lhs, x)
			}
		}
		return // FIXES: a%%b/foo/bar -> Path(a%%b foo bar)

	case token.BAR:
		if _, ok := x.(*Group); ok { return } // in case of: [(var)|...]

	case token.COMMA:
		if p.bits&(parseArged|parseCall|parseGroup) != 0 { return }
		if p.bits&parseDefineClause == 0 {
			warn(p, "%016b: %T %v ; %v %v", p.bits, x, x, p.tok, p.lit).debug(1)
			return
		}

	case
		token.COMPOSED, token.COLON, token.SEMICOLON, token.RAW,
		token.RPAREN, token.RBRACK, token.RBRACE, token.SPACE,
		token.LINEND, token.EOF:
		return // No composition!
	}

	var y = p.composite(ctx, lhs)
	if _, ok := y.(*Path); ok {
		switch x.(type) {
		case *Flag: // okay: -Ifoo/bar, -Lfoo/bar
		case *Path: // okay: combine two paths
		case *String, *Compound, *delegate, *closure, *punctuation:
		case *barecomp:
		default:
			warn(of(ctx,y), "barecomp path: %T %v ; %v (next=%v)", x, x, y, p.tok).debug(1)
		}
	}

	// Further composing
	switch t := x.(type) {
	case *barecomp: t.Combine(ctx, y)
	case *Path: t.Combine(ctx, y)
		if false { info(at(ctx,t.position), "%v (%v) (tok=%v)", t, y, p.tok) }
	default:
		comp := MakeBarecomp(x.Position(), x)
		comp.Combine(ctx, y)
		x = comp
	}

	// Keep trying composing as long as possible
	switch p.tok {
	case token.SPACE, token.LINEND, token.EOF: break
	default: //case token.SELECT_PROG1, token.SELECT_PROG2, token.LPAREN:
		goto SwitchCompose
	}
	return
}

// ----------------------------------------------------------------------------
// Clauses & Declarations

type genericClauseOpts struct {
	generalOpts
	conds []Value `cond,if,where`

    keyword token.Token // e.g. use, files, eval, etc.
    skip bool // e.g. -cond(false)
    all  []Value // all option values (unparsed)
    vals []Value // remaining option values after parsed
	spec []Value
}

type parseSpecFunc func(Context, *CommentGroup, *genericClauseOpts, int)

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

func (p *parser) _parseUseSpecProps(props []Value) (opts useOpts, params []Value, err error) {
    // Supported parameter forms:
    //      -param
    //      -param(value)
    //      -param=value
	var ctx = p.posit()
    var useList []Value // TODO: apply useList
    for _, prop := range props {
        var s string
        switch t := prop.(type) {
        case *Flag:
            switch s = t.name.Strval(ctx); s {
            //case "nouse", "unuse": opts.unuse = true
            case "reuse": opts.reuse = true
            default: params = append(params, prop)
            }
        case *Pair: // -param=value
            switch tt := t.Key.(type) {
            case *Flag:
                switch s = tt.name.Strval(ctx); s {
                case "use": useList = append(useList, t.Value)
                default: params = append(params, prop)
                }
            default:
                erro(of(ctx,t.Key), "parameter `%v' unsupported `%T`", prop, prop)
                return
            }
        case *Argumented: // -param(value)
            switch tt := t.value.(type) {
            case *Flag:
                switch s = tt.name.Strval(ctx); s {
                case "use": useList = append(useList, t.args...)
                default: params = append(params, prop)
                }
            default:
                erro(of(ctx,t.value), "parameter `%v' unsupported `%T`", prop, prop)
                return
            }
        default:
            erro(of(ctx,prop), "parameter `%v` unsupported `%T`", prop, prop)
            return
        }
    }
    return
}

func (p *parser) use(ctx Context, doc *CommentGroup, g *genericClauseOpts, _ int) {
	if p.imports = append(p.imports, &usespec{ g.spec }); g.skip {
		// TODO: maybe give some information
		return
	}

	ctx = at(ctx, g.spec[0].Position()) // p.posit()

	var specVals, arged []Value
	switch v := g.spec[0].(type) {
	case *delegate:
        for _, val := range mergex(ctx, plain, v) {
            if !isTrivial(val) { specVals = append(specVals, val) }
		}
    case *Pair:
        var s string
        if f, ok := v.Key.(*Flag); !ok {
            erro(ctx, "'%v' invalid use spec", v.Key)
            return
        } else if s = f.name.Strval(ctx); s != "list" {
            erro(ctx, "'%v' invalid use spec, do you mean -list?", v.Key)
            return
        }

        for _, val := range mergex(ctx, plain, v.Value) {
            if !isTrivial(val) { specVals = append(specVals, val) }
        }
	case *Argumented:
        for _, val := range mergex(ctx, plain, v.value) {
            if !isTrivial(val) { specVals = append(specVals, val) }
        }
		arged = v.args
	default:
		specVals = append(specVals, v)
    }
	if len(specVals) == 0 {
        erro(ctx, "empty use spec: %v (%T)", g.spec[0], g.spec[0]).debug(1)
        return
    }

	var opts useOpts
	var args = parseOpts(ctx, &opts, 0, append(g.vals, g.spec[1:]...)...)
	for _, a := range args {
		if _, ok := a.(*Flag); ok || true {
			erro(of(ctx,a), "unkown use opts: %T %v", a, a).debug(1)
			return
		}
	}

	var wg sync.WaitGroup
	var loader = ctx.loader()
	for _, specVal := range specVals {
		if ctx := at(ctx, specVal.Position()); true {
			loader.use(ctx, opts, specVal, arged, args...)
		} else {
			var dc = diagContext{ Context: ctx } // redefine ctx
			wg.Add(1); go func() {
				defer checkFailure(&dc, true)
				defer func() {
					if len(dc.points) > 0 { dc.inner().diagnostic().nest(dc.points) }
					wg.Done()
				} ()
				loader.use(ctx, opts, specVal, arged, args...)
			} ()
		}
	}
	wg.Wait()

	if errs := ctx.checkErrors(true); errs > 0 {
		var (
			pos = p.Position()
			proj = loader.Project()
		)
        prompt(ctx, "%s: use %v failed; %d errors\n", proj, specVals, errs)
		erro(at(ctx,pos), "%v errors: use %v", errs, specVals).debug(6)
		if true { fail(pos, "%s: use %v failed; %d errors", proj, specVals, errs) }
	}
	return
}

func (p *parser) include(ctx Context, doc *CommentGroup, g *genericClauseOpts, _ int) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Spec")) }

	var opts = includeOpts{ genericClauseOpts: g }
	if vals := parseOpts(ctx, &opts, 0, g.vals...); len(vals) > 0 {
		// TODO: deal with the unparsed generic options
		warn(ctx, "unknown opts: %v", vals).debug(1)
	}

	if len(g.spec) < 1 {
		erro(ctx, "expecting include file: %v", g.spec).debug(1)
		return
	}

	var x = g.spec[0]
	var loader = ctx.loader()
	if p.spaces(); p.tok == token.COLON {
		switch x.(type) {
		case *File, *String, *Compound: // escape from file searching
		default: if file := loader.project.file(ctx, x.Strval(ctx)); file != nil {
			x = file
		} else if val := x.expand(ctx, plain); !isNil(val) && val != x {
			x = val
		}}

		x = p.rule(specialRuleNor, nil, []Value{x}) // this should return a RuleEntry
	}
	if !g.skip { loader.include(ctx, opts, x) }
}

func (p *parser) files(ctx Context, doc *CommentGroup, g *genericClauseOpts, _ int) {
	defer p.setbits(p.setbit(parseFilesSpec))
	if len(g.spec) != 1 {
		erro(ctx, "too many files properties: %v", g.spec).debug(1)
		return
	}

	var path Value
	if p.tok == token.SELECT_PROG1 {
		p.next(true) // step forward with spaces skipped
		if p.tok == token.LINEND || p.lineComment != nil {
			erro(ctx, "expecting files path")
		}
		path = p.expr(ctx, false)
	}
	if p.spaces(); p.lineComment != nil {
		//spec.Comment = p.lineComment
	}
	if g.skip {
		// TODO: maybe give some information
		return
	}

	ctx = p.posit()

	var (
		val = g.spec[0]
		opts cacher
		pats []Value
	)
	parseOpts(ctx, &opts, 0, g.vals...)

	if g, ok := val.(*Group); ok {
		pats = g.Elems
	} else if val.expandible(ctx, expandClosure) {
		pats = []Value{ val }
	} else {
		pats = mergex(ctx, plain, val)
	}

	if path == nil {
		if len(pats) == 1 { if a, ok := pats[0].(*Argumented); ok { if f, ok := a.value.(*Flag); ok {
			var name = f.name.Strval(ctx)
			switch name {
			default:
				// TODO: parse files options
				erro(of(ctx,f.name), "invalid files flag: %v").debug(1)
				return
			}
		}}}
		var ( files []*File; newPats []Value )
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
			var paths = []Value{ MakeString(val.Position(), ctx.Project().absPath) }
			opts.cache(ctx, pats, paths)
		}
	} else {
		var patsNew []Value
		for _, pat := range pats {
			if pat.expandible(ctx, expandClosure) {
				patsNew = append(patsNew, pat)
			} else {
				patsNew = append(patsNew, mergex(ctx, plain, pat)...)
			}
		}

		var paths []Value
		if g, ok := path.(*Group); ok {
			paths = g.Elems
		} else {
			paths = []Value{ path }
		}

		if len(patsNew) == 1 { if f, ok := patsNew[0].(*Flag); ok {
			var name = f.name.Strval(ctx)
			switch name {
			default:
				// TODO: parse files options
				erro(of(ctx,f.name), "invalid files flag: %v").debug(1)
				return
			}
		}}

		opts.cache(ctx, patsNew, paths)
	}
}

func (p *parser) evalConfiguration(ctx Context, g *genericClauseOpts, props []Value) {
	var project = ctx.Project()
	if project == nil {
		erro(ctx, "configuration: nil project").debug(1)
		return
	} else if project.configure == nil {
		erro(ctx, "configuration: no %s for %v", dotConfigure, project).debug(1)
		return
	}

	if entry := project.configure.defaultEntry; entry == nil {
		// no init entry from .configure
	} else if _, ts := entry.Execute(at(ctx, entry.Position())); len(ts) > 0 {
		// FIXME: the entry might be a configure operation (see configure/.base/do.smart)
		for _, brk := range ts {
			if brk.what == traveFail {
				erro(of(ctx,entry), "execute '%v' failed: %v", entry, brk).debug(1)
			}
		}
	}

	if ctx.checkErrors(true)>0 { return }
	if project.configured {
		prompt(ctx, "configuration: %v already configured\n", project)
		return
	}

	var (
		okay bool
		cp *Project
		ce = &configureExecutor{ defs:make(map[string]*def) }
	)
	defer ce.close()

	for _, dep := range mergex(ctx, plain, props[1:]...) {
		if re, y := dep.(*RuleEntry); !y {
			erro(ctx, "unsupported prerequisite: %T %v", dep, dep).debug(1)
		} else if _, ts := re.Execute(ctx); len(ts) > 0 {
			for _, brk := range ts {
				if brk.what == traveFail {
					erro(of(ctx,re), "execute '%v' failed: %v", re, brk).debug(1)
				}
			}
		}
	}

	if ctx.checkErrors(true)>0 { return }

	for _, entry := range project.configs {
		if cp, okay = ce.execute(at(ctx, entry.Position()), cp, entry); !okay {
			erro(ctx, "configure '%v' failed", entry).debug(1)
			break
		}
	}

	project.configured = true // relaxes universeContext.configure
}

func (p *parser) assert(ctx Context, doc *CommentGroup, g *genericClauseOpts, _ int) {
	if !g.skip { builtin{p.posit(), g.all, plain}.assert(g.spec...) }
}

func (p *parser) append(ctx Context, doc *CommentGroup, g *genericClauseOpts, _ int) {
	if !g.skip { builtin{p.posit(), g.all, plain}.append(g.spec...) }
}

func (p *parser) eval(ctx Context, doc *CommentGroup, g *genericClauseOpts, _ int) {
	var (
		prop0, resolved, res Value
		name string
	)

	if g.skip { return } else if g.spec == nil {
		var opts struct {
			// TODO: options
		}
		for _, op := range parseOpts(ctx, &opts, plain, g.all...) {
			var val Value
			if v, y := op.(*Pair); y { op, val = v.Key, v.Value }
			if v, y := op.(*Flag); y && v.name.Strval(ctx) == "ddd" {
				if false { warn(of(ctx,op), "todo: %v (%v)", v.name, val).debug(1) }
				p.ddd = val != nil && val.True(ctx)
			} else {
				erro(of(ctx,op), "unsupport flag: %T %v (%v)", v, v, val).debug(1)
			}
		}
		return
	} else if prop0 = g.spec[0]; isTrivial(prop0) {
		erro(ctx, "illegal").debug(1)
		return
	}

	var opts []Value
	if a, y := prop0.(*Argumented); y { prop0, opts = a.value, a.args }

	ctx = at(ctx, prop0.Position())

	var loader = ctx.loader()
	if name, resolved = loader.resolveObject(prop0); false {
		erro(ctx, "resolve '%v' failed", prop0).debug(1)
		return
	} else if !isTrivial(resolved) {
		// ...
	} else if name == "configuration" {
		// NOTE: see also universeContext.configure()
		p.evalConfiguration(ctx, g, g.spec)
		return
	} else {
		erro(ctx, "resolved '%v' is nil (options = %v)", prop0, *g).debug(1)
		return
	}

	// At the point of `eval` was represented, the closure context
	// might be empty. So we start closure with the current scope.
	if op, y := resolved.(*Builtin); !y {
		erro(ctx, "resolved '%v' is not a command (%s)", prop0, typeof(resolved)).debug(1)
		return
	} else if op.s.b&builtinCommand == 0 {
		erro(ctx, "resolved builtin '%v' is not a command", prop0).debug(1)
		return
	} else if op.s.f != nil {
		if false && strings.Contains(p.scanner.File().Name(), ".after.") {
			warn(ctx, "%v: %v", p.scanner.File().Name(), ctx.Scope()).debug(1)
			warn(ctx, "%v: %v", p.scanner.File().Name(), ctx).debug(1)
			ctx.checkErrors(true)
			for i, v := range g.spec[1:] {
				warn(ctx, "%v: %v %s - %T %v - %d", p.scanner.File().Name(), p.tok, p.lit, v, v, i).debug(1)
				ctx.checkErrors(true)
				warn(ctx, "%v: %v %s %v", p.scanner.File().Name(), p.tok, p.lit, v.expand(ctx, plain)).debug(1)
				ctx.checkErrors(true)
				warn(ctx, "%v: %v %s %v", p.scanner.File().Name(), p.tok, p.lit, mergex(ctx, plain, v)).debug(1)
				ctx.checkErrors(true)
			}
		}
		res = op.s.f(builtin{ctx, opts, plain}, g.spec[1:]...)
	}

	if ctx.checkErrors(true); isTrivial(res) { return }

	/* TODO: if c, y := res.(code); y { ... } */
}

func (p *parser) directiveSpec(ctx Context) (props []Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Spec")) }

	//var doc = p.leadComment
	var comment *CommentGroup

ParamsParseLoop: // Parse the directive parameters
	for p.tok != token.EOF {
		switch p.spaces(); p.tok {
		case token.COMMA, token.LINEND, token.RPAREN, token.RBRACE,
			token.SELECT_PROG1, token.COLON: break ParamsParseLoop
		}

		if p.lineComment != nil {
			comment = p.lineComment
			break
		}

		props = append(props, p.expr(ctx, false))
	}
	if comment != nil { /* TODO: directive documments */ }
	return
}

func (p *parser) spec(ctx Context, keyword token.Token, pos token.Pos, f parseSpecFunc) {
	if t_traverse.enabled { defer un(trace(t_traverse, "spec("+keyword.String()+")")) }

	var opts = genericClauseOpts{ keyword: keyword }
	for p.spaces(); p.tok == token.MINUS; p.spaces() {
		opts.all = append(opts.all, p.expr(ctx, false))
	}
	opts.vals = parseOpts(ctx, &opts, expandZero, opts.all...)

	for _, cond := range opts.conds {
		if t := cond.True(at(ctx, cond.Position())); !t {
			opts.skip = true
			break
		}
	}

	if p.spaces(); p.tok == token.LINEND {
		if keyword == token.EVAL { f(ctx, nil, &opts, 0) } else {
			erro(ctx, "%v: nil specs", keyword).debug(1)
		}
		return
	} else if p.spaces(); p.tok == token.LPAREN {
		p.next(true)
		for iota := 0; p.tok != token.RPAREN && p.tok != token.EOF && (p.stop == 0 || p.pos < p.stop); iota++ {
			// TODO: collect documentation comments
			for p.tok == token.SPACE || p.tok == token.LINEND { p.next(true) }
			if p.tok == token.RPAREN || p.tok == token.EOF { break  }
			if opts.spec = p.directiveSpec(ctx); true {
				f(ctx, p.leadComment, &opts, iota)
			}
			if p.tok == token.COMMA || p.tok == token.LINEND { p.next(true) }
		}
		p.expect(token.RPAREN)
		if p.spaces(); p.tok != token.EOF { p.linend() }
		return
	}

	if p.tok != token.LINEND && p.tok != token.EOF && (p.stop == 0 || p.pos < p.stop) {
		if opts.spec = p.directiveSpec(ctx); true { f(ctx, nil, &opts, 0) }
		if p.tok == token.COMMA { p.next(true) }
	}
	if p.tok != token.EOF && (p.stop == 0 || p.pos < p.stop) {
		if p.spaces(); p.lineComment == nil { p.linend() }
	}
}

func (p *parser) define(ctx Context, tok token.Token, ident Value) (def *def) {
	if t_traverse.enabled { defer un(trace(t_traverse, fmt.Sprintf("Define(%s)", ident))) }

	// Only accept scoped identifiers if it's ":user:" program
	if ctx.Scope().comment == usecomment {
		switch i := ident.(type) {
		case *selection:
			erro(of(ctx,ident), "should use scoped names instead of `%v`", i)
		default:
			erro(of(ctx,ident), "FIXME: unexpected name expression: %T", i)
		}
		return
	}

	var (
		savedAutos = p.autos // save for $1, $2, $3, etc...
		savedBits = p.bits
		// TODO: doc = p.leadComment
		// TODO: comment = p.lineComment
		position = p.loc(p.expect(tok))
		elems []Value
		value Value
	)
	p.bits |= parseDefineClause
	elems = p.right(ctx)
	p.autos = savedAutos
	p.bits = savedBits

	// Take the line comment, since the line comment is assigned.
	p.lineComment = nil

	// Create List value or use the first elem.
	if n := len(elems); n == 1 {
		value = elems[0]
	} else if n > 1 {
		value = MakeList(elems[0].Position(), elems...)
	}

	// NOTE: Put all explicit defs into project scope. It's important for defs enclosed
	//       in templates work.
	var loader = ctx.loader()
	if scope := loader.project.scope; len(loader.scopes) == 0 || loader.scopes[0] != scope {
		defer func(s []*Scope) { loader.scopes = s } (loader.scopes)
		loader.scopes = append([]*Scope{ scope }, loader.scopes...)
	}
	var defs = loader.define(at(ctx, position), tok, ident, value)
	if n := len(defs); n > 0 {  def = defs[n-1] }
	return
}

func (p *parser) recipe(ctx Context) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "Recipe")) }

	var (
		// TODO: comment *CommentGroup
		// TODO: doc = p.leadComment
		loader = ctx.loader()
		position = p.Position()
		elems []Value
		isList bool
	)

SwitchDialect:
	switch p.dialect {
	case "", "eval", "value":
		p.scanner.LeaveCompoundLineContext()
		p.next(true) // skip RECIPE or SEMICOLON and parse in list mode
		position = p.Position()
		if isList = true; !p.isEndOfLine() {
			defer p.setbit(p.setbit(parseRecipeBuiltin))

			var (
				isValue = p.dialect == "value"
				x = p.expr(ctx, /*!isValue*/false) // parse first expr of recipe
				n = isNil(x)
				a *Argumented
			)
			if !n { if a, _ = x.(*Argumented); a != nil { x = a.value } }
			if  n {
				erro(ctx, "parsed value is nil")
			} else if isValue {
				// no resolving commands
			} else if t, ok := x.(*bareword); !ok {
				// does nothing
			} else if _, sym := loader.resolveObject(t); false {
				erro(ctx, "resolve '%v' failed", x)
			} else if isTrivial(sym) {
				erro(of(ctx,x), "resolved '%v' (from %v) is nil", t.string, x)
			} else if false {
				erro(of(ctx,x), "builtin command no more supported, use $(%s ...) instead", t.string)
			} else if b, ok := sym.(*Builtin); !ok {
				erro(of(ctx,x), "'%s' is not a command (%s)", t.string, typeof(sym))
			} else if b.s.b&builtinCommand == 0 {
				erro(of(ctx,x), "'%s' is not a command, use $(%s ...) instead", t.string, t.string)
			} else {
				x = sym
			}

			if !isValue && p.tok.IsAssign() {
				elems = append(elems, p.define(ctx, p.tok, x))
				break SwitchDialect
			} else if a != nil {
				elems = append(elems, a)
			} else {
				elems = append(elems, x)
			}

			var cmdargs []Value
			for p.tok != token.EOF && p.tok != token.SEMICOLON && p.tok != token.LINEND && p.lineComment == nil {
				if p.spaces(); p.lineComment != nil {
					// TODO: comment = p.lineComment
					break
				}

				if p.tok.IsRuleDelim() {
					if false {
						x = p.rule(specialRuleRec, nil, elems) // RuleEntry
					} else {
						erro(ctx, "unsupported token: %s, %v", p.tok, elems).debug(1)
					}
				} else {
					x = p.expr(ctx, false)
				}

				cmdargs = append(cmdargs, x)
				if p.tok == token.COMMA {
					p.next(true)
					elems = append(elems, MakeList(p.Position(), cmdargs...))
					cmdargs = []Value{}
				}
				if p.lineComment != nil {
					// TODO: comment = p.lineComment
					break
				}
			}
			elems = append(elems, MakeList(p.Position(), cmdargs...))
		}

	default:
		p.next(true) // skip RECIPE or SEMICOLON and parse in line-string mode
		position = p.Position()
		for !p.isEndOfLine() {
			var x Value
			var bits = p.setbit(parseRecipeText)
			switch p.tok {
			default:           x = p.expr(ctx, false)
			case token.RAW:    x = p.literal(false)
				/*
			case token.LINEND:
				erro(ctx, "unexpected end of line for compound string")
				break ForCompound*/
			}
			p.setbits(bits)
			elems = append(elems, x)
		}
	}
	if p.spaces(); p.tok != token.EOF { p.linend() }
    if len(elems) == 0 {
        return MakeNone(position)
    } else if isList {
        return MakeList(position, elems...)
    } else {
        return MakeCompound(position, elems...)
    }
}

func (p *parser) movar(ctx Context, args []Value) (err error) {
	var loader = ctx.loader()
	// Parsing (var a=xxx,b=yyy) definitions
	for _, elem := range args[1:] {
		var kv, ok = elem.(*Pair)
		if !ok || kv == nil {
			erro(of(ctx,elem), "bad var form (%T)", elem)
			continue
		}

		var name string
		var k, v = kv.Key, kv.Value
		if name = k.Strval(at(ctx, k.Position())); name == "" {
			erro(of(ctx,k), "name '%v' is empty", k)
		}
		if def, alt := loader.def(elem.Position(), name); alt != nil {
			erro(of(ctx,k), "Def '%v' already existed: %T", name, alt)
		} else if def != nil {
			var ctx = at(ctx, v.Position())
			if g, ok := v.(*Group); ok {
				def.val(ctx, g.ToList(def.position))
			} else {
				def.val(ctx, v)
			}
		}
	}
	return
}

func (p *parser) defineConfigureTargets(ctx Context) {
	var loader = ctx.loader()
	for _, t := range p.targets {
		var pos = t.Position()
		if !pos.IsValid() { pos = p.Position() }

		var (
			ctx = at(ctx, pos)
			name = t.Strval(ctx)
		)

		var d, a = loader.project.scope.define(ctx, /*defVoid*/DefConfig, name, nil)
		if d == nil && a != nil { if d, _ = a.(*def); d == nil {
			erro(ctx, "configure %v: already defined in '%v' as %v", t, loader.project, a).debug(6)
			return
		}}
		if !d.position.IsValid() { d.position = pos }
	}
}

func (p *parser) ruleParams(ctx Context, args []Value) (err error) {
	var loader = ctx.loader()
	for _, elem := range args {
		var ctx = at(ctx, elem.Position())
		switch elem.(type) {
		case *bareword, *barecomp:
			var s = elem.Strval(ctx)
			var d, a = loader.def(elem.Position(), s)
			if a != nil {
				var y bool
				if d, y = a.(*def); !y {
					erro(of(ctx,elem), "%T '%s' already taken the name, no such parameter", a, s)
				}
			}
			if d != nil {
				d.set(ctx, DefArg, nil)
			} else {
				erro(of(ctx,elem), "'%s' is not defined", s)
			}
			p.params = append(p.params, d)
			ctx.Scope().replace(ctx, strconv.Itoa(len(p.params)), d)
		default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
			erro(of(ctx,elem), "bad parameter form (%T)", elem)
		}
	}
	return
}

func (p *parser) modifiers(ctx Context) *modifiergroup {
	if t_traverse.enabled { defer un(trace(t_traverse, "Modifiers")) }

	var (
		posLp = p.loc(p.expect(token.LBRACK))
		hasParameters bool // ((foo bar))
		elems []*modifier
	)

	defer func(a parseBits) { p.bits = a }(p.bits)
	p.bits |= parseModifier

ForModifiersExpr:
	for p.tok != token.RBRACK && p.tok != token.EOF {
		if p.spaces(); p.tok == token.RBRACK { goto rBrack }

		var (
			x = p.expr(ctx, false)
			group *Group
			name string
		)
		if ctx.checkErrors(true) > 0 {
			erro(at(ctx,x.Position()), "modifier: %T %v", x, x).debug(1)
			return nil
		} else if g, ok := x.(*Group); !ok {
			var xv = x.expand(ctx, expandDelegate/*TODO: expandInline or expandAuto*/)
			warn(at(ctx,x.Position()), "modifier: %T %v   →   %T %v", x, x, xv, xv).debug(1)
			continue ForModifiersExpr
		} else {
			group = g
		}
		if l, ok := group.Elems[0].(*List); ok {
			group.Elems = append([]Value{ l.Elems[0] }, append(l.Elems[1:], group.Elems[1:]...)...)
		}

		switch n := group.Elems[0].(type) {
		case *bareword:
			if name = n.string; name == "var" {
				p.movar(ctx, group.Elems)
				continue ForModifiersExpr
			} else if name == "configure" {
				p.defineConfigureTargets(ctx)
				p.configure = true // set configure flag and define configure variables
			}
			goto checkNameAndAdd
		case *Group: // parameters: ((foo bar))
			hasParameters = true
			if p.ruleParams(ctx, n.Elems); true {
				warn(ctx, "move parameters into depend list: %v", n).debug(1)
			}
			continue ForModifiersExpr
		case *delegate, *closure, *barecomp, *String:
			var ctx = at(ctx, n.Position())
			var v = mergex(ctx, plain, n)
			if name = v[0].Strval(ctx); name == "" {
				erro(of(ctx,n), "name '%v' is empty", n).debug(1)
				continue ForModifiersExpr
			}
			goto checkNameAndAdd
		default:
			erro(of(ctx,n), "unsupported dialect or modifier (%T): %v", group.Elems[0], group.Elems[0]).debug(1)
			continue ForModifiersExpr
		}

		goto addModifier

	checkNameAndAdd:
		if _, ok := dialects[name]; ok {
			if p.dialect == "" { p.dialect = name } else {
				erro(of(ctx,x), "multi-dialects unsupported, already defined '%s'", p.dialect).
					debug(1)
				continue ForModifiersExpr
			}
		} else if _, ok = modifiers[name]; !ok {
			erro(of(ctx,x), "`%s` no such dialect or modifier", name).debug(1)
			continue ForModifiersExpr
		}

	addModifier:
		if len(group.Elems) == 0 {
			erro(of(ctx,x), "empty modifier: %v", x).debug(1)
		} else {
			var m = &modifier{
                valbase: valbase{group.Position()},
                name: group.Elems[0],
            }
            if len(group.Elems) > 1 {
                m.args = group.Elems[1:]
            }
			elems = append(elems, m)
		}
	}
	p.spaces()
	rBrack: p.expect(token.RBRACK)
	if len(elems) == 0 && !hasParameters {
		erro(at(ctx,posLp), "empty modifier group").debug(1)
	}
	if p.tok == token.COLON {
		erro(at(ctx,posLp), "unexpected colon after modifer").debug(1)
	}
    return &modifiergroup{ valbase: valbase{posLp}, modifiers: elems }
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
var automatics = []string{
	"@" , "%" , "<" , ">" , "?" , "^" , "+" , "|" , "*" , //
	"@D", "%D", "<D", ">D", "?D", "^D", "+D", "|D", "*D", //
	"@F", "%F", "<F", ">F", "?F", "^F", "+F", "|F", "*F", //
	"@'", "%'", "<'", ">'", "?'", "^'", "+'", "|'", "*'", //
	"-" , "~" ,
}

func (p *parser) rule(special specialRule, optvals, targets []Value) (result Value) {
	var ctx = p.posit()
	if ctx.Project().keyword == token.PACKAGE {
		erro(ctx, "rules forbidden: %v", targets).debug(1)
		return nil
	} else if t_traverse.enabled {
		defer un(trace(t_traverse, "Rule"))
	}

	var (
		// TODO: doc = p.leadComment
		depends []Value
		ordered []Value
		recipes []Value
		scopeComment string
	)
	switch special {
	case specialRuleUse: scopeComment = fmt.Sprintf(usecomment)
	default:             scopeComment = fmt.Sprintf("rule %v", targets)
	}

	var loader = ctx.loader()
	defer loader.closeScope(loader.openScope(scopeComment))
	p.params = nil
	p.dialect = ""

	var position = ctx.Position()
	for _, s := range automatics {
		var def, alt = loader.def(position, s)
		if alt != nil {
			erro(ctx, "name `%s' already taken, not automatic (%T).", s, alt)
		} else if def == nil {
			erro(ctx, "'%s' is not defined", s)
		} else {
			assert(def.value == nil, "initial automatic values must be nil")
			def.origin = DefAuto
		}
	}
	for i := 1; i < 10; i += 1 {
		var def, alt = loader.def(position, strconv.Itoa(i))
		if alt != nil {
			erro(ctx, "name `%v` already taken, not numberred (%T).", i, alt)
		} else if def == nil {
			erro(ctx, "'$%d' is not defined", i)
		} else {
			def.origin = DefAuto
		}
	}

	// switch special {
	// case specialRuleUse:
	// 	if name, alt := ctx.Scope().ProjectName(ctx, selfproj, ctx.Project()); alt != nil {
	// 		erro(ctx, "name `%s` already taken, not automatic (%T)", selfproj, alt)
	// 	} else if name == nil {
	// 		erro(ctx, "cannot define `%s` automatic", selfproj)
	// 	}
	// 	if name, alt := ctx.Scope().ProjectName(ctx, userproj, nil); alt != nil {
	// 		erro(ctx, "name `%s` already taken, not automatic (%T)", userproj, alt)
	// 	} else if name == nil {
	// 		erro(ctx, "cannot define `%s` automatic", userproj)
	// 	}
	// }

	// NOTE: expand targets to speed up for later usage, it might spend lots of time in
	// project.entry while matching for entry looked up if not expanded right now.
	targets, _, _ = (plain&^expandArgedArgs).expand(ctx, targets...)

	defer func(t []Value) { p.targets = t } (p.targets)
	p.next(true) // skip rule delimeters and spaces
	p.targets = targets // save targets for later refering

	if p.tok != token.SEMICOLON && p.tok != token.BAR && !p.isEndOfLine() {
		depends = p.depends(ctx, true)
	}
	if p.tok == token.BAR { // '|' starts the ordered prerequisites
		if p.next(true); p.tok != token.SEMICOLON && !p.isEndOfLine() {
			ordered = p.depends(ctx, false)
		}
	}

	if p.tok == token.SEMICOLON { // ;
		// Parse inline recipe in the program scope.
		recipes = append(recipes, p.recipe(ctx))
	} else /*if p.tok == token.LINEND || p.lineComment != nil*/ {
		// Parse recipes in the program scope.
		p.scanner.Recipes(true) // Turn on recipes before LINEND.
		if p.linend() { // Take the new line.
			for p.tok != token.EOF && p.isRecipeStart() {
				recipes = append(recipes, p.recipe(ctx))
			}
		}
		p.scanner.Recipes(false)
	}

	var params []string
	if t := targets[0]; p.configure {
		name := t.Strval(ctx)
		d, a := ctx.Project().scope.define(ctx, DefVoid, name, nil)
		if d == nil && a == nil {
			erro(of(ctx,t), "cannot define configure target '%v'", name)
		} else if a != nil {
			if _, ok := a.(*def); !ok {
				erro(of(ctx,t), "configure target '%v' already taken: %T %v", name, a, a)
			}
		}
		if d != nil && !d.position.IsValid() { d.position = t.Position() }
	} else {
		for _, d := range p.params { params = append(params, d.name) }
	}

	parsedData := &parsedRuleData{
		// TODO: lang: 0,
		params:   params,
		position: position,
		config:   p.configure,
		targets:  barefilize(ctx, targets...),
		depends:  barefilize(ctx, depends...),
		ordered:  barefilize(ctx, ordered...),
		recipes:  recipes,
		options:  optvals,
		special:  special,
	}

	if special != specialRuleRec {
		var res []Entry
		var loader = ctx.loader()
		if res = loader.rule(parsedData); len(res) == 1 {
			result = res[0]
		} else if len(res) > 1 {
			var list = MakeList(res[0].Position())
			for _, v := range res { list.Elems = append(list.Elems, v) }
			result = list
		} else {
			result = MakeNil(parsedData.position)
		}
	}

	// Close the rule scope and go back to project scope. The current
	// scope must be project scope befor Rule.
	p.configure = false
	p.dialect = ""
	p.params = nil
	return
}

func (p *parser) specialRule() Value {
	if t_traverse.enabled {
		defer un(trace(t_traverse, "SpecialRule"))
	}

	p.expect(token.COLON) // expect and skip ':'

	if p.tok != token.BAREWORD {
		erro(p, "unknown special rule")
		return nil
	}

	var name = p.lit
	switch name {
	case "user":
		if true {
			// Example usage of use.*:
			//    use.* ::= cflags(-unique) ldlibs(-unique -reverse)
			erro(p, ":user: rules are deprecated, use use.* instead!").debug(1)
		} else {
			var options []Value
			var pos = p.expect(token.BAREWORD) // USE
			var bits = p.setbit(parseSpecialRule)
			var ctx = p.posit()
			// Options are *Flag or *Pair of a Flag.
			for p.tok == token.MINUS {
				opt := p.expr(ctx, false)
				options = append(options, opt)
			}
			p.setbits(bits) // restore bits
			if p.tok.IsRuleDelim() {
				return p.rule(specialRuleUse, options, []Value{
					MakeBareword(p.loc(pos), name),
				})
			}

			erro(p, "expecting special rule terminator ':'")
		}
		return nil
	default:
		erro(p, "unknown special rule")
		return nil
	}
}

var pprofCounter int
func (p *parser) templateBlock(ctx Context, t *template, vars map[string]Value, expandParams []Value) {
	p.scanner.SetState(t.state)
	p.pos, p.tok, p.lit = t.pos, t.tok, t.lit

	// TODO: deal with expandParams

	// NOTE: comment here will affect loader.def()
	if false { pprofCounter += 1
		var (
			profCpu = fmt.Sprintf("template-%05d.cpu.prof", pprofCounter)
			profMem = fmt.Sprintf("template-%05d.mem.prof", pprofCounter)
			fCpu *os.File
			e error
		)
		if fCpu, e = os.Create(profCpu); e != nil {
			erro(ctx, "%T: %v", e, e).debug(1)
			return
		} else if e = pprof.StartCPUProfile(fCpu); e != nil {
			erro(ctx, "%v: %v", profCpu, e).debug(1)
			fCpu.Close() ; return
		}
		defer func() {
			pprof.StopCPUProfile()
			fCpu.Close()

			var fMem, e = os.Create(profMem)
			if e != nil { erro(ctx, "%v", e).debug(1) }

			runtime.GC() // update memory statistics
			e = pprof.WriteHeapProfile(fMem)
			fMem.Close()

			if e != nil { erro(ctx, "%v: %v", profMem, e).debug(1) }
		} ()
	}

	var loader = ctx.loader()
	defer loader.closeScope(loader.openScope("template block"))

	ctx = p.posit()

	var position = ctx.Position()
	// var cc = autoContext{ Context:ctx, defs:make(autoDefMap) }

	for s, v := range vars {
		if def, alt := loader.def(position, s); alt == nil {
			def.set(ctx, DefAuto, v)
		} else {
			erro(ctx, "variable '%s' already taken", s).debug(1)
		}
	}

	// ctx = &cc

	var savedBits = p.bits
	p.bits |= parseTemplateBlock
	for p.tok != token.EOF && p.pos < p.stop {
		if p.tok == token.LINEND || (p.tok == token.COMMENT && p.lineComment != nil) {
			p.next(true)
		} else {
			p.clause()
		}
	}
	p.bits = savedBits
}

func (p *parser) templateExpand(ctx Context, t *template, params []Value) {
	var count int64
	defer func(t time.Time, pos token.Pos, tok token.Token, lit string, state scanner.State) {
		if ddd {/* dont check time in ddd mode */} else
        if d := time.Now().Sub(t); d > time.Duration(options.slow)*time.Millisecond {
			var c = time.Duration(count)
            warnstack(ctx, 3, "slow: %v, %d * %v, prof-%d", d, count, d/c, pprofCounter).debug(1)
        }
		p.pos, p.tok, p.lit	 = pos, tok, lit
		p.scanner.SetState(state)
	} (time.Now(), p.pos, p.tok, p.lit, p.scanner.GetState())

	// TODO: parseOpts(params) -> add option to turn off asFile in Context

	if false { pprofCounter += 1
		var (
			profCpu = fmt.Sprintf("template-%05d.cpu.prof", pprofCounter)
			profMem = fmt.Sprintf("template-%05d.mem.prof", pprofCounter)
			fCpu *os.File
			e error
		)
		if fCpu, e = os.Create(profCpu); e != nil {
			erro(ctx, "%T: %v", e, e).debug(1)
			return
		} else if e = pprof.StartCPUProfile(fCpu); e != nil {
			erro(ctx, "%v: %v", profCpu, e).debug(1)
			fCpu.Close() ; return
		}
		defer func() {
			pprof.StopCPUProfile()
			fCpu.Close()

			var fMem, e = os.Create(profMem)
			if e != nil { erro(ctx, "%v", e).debug(1) }

			runtime.GC() // update memory statistics
			e = pprof.WriteHeapProfile(fMem)
			fMem.Close()

			if e != nil { erro(ctx, "%v: %v", profMem, e).debug(1) }
		} ()
	}

	switch t.verb {
	case "foreach": // foreach val1 val2 val3 val4 ...
		for _, elem := range mergex(ctx, plain, t.params...) {
			if isTrivial(elem) { continue }
			p.templateBlock(ctx, t, map[string]Value{ "_" : elem }, params)
			count += 1
		}
	case "for": // for name1=(val1 val2 val3 ...) name2=(val1 val2 val3)
		var (
			vars = make(map[string]struct{
				elems []Value
			})
			num int
		)
		for _, a := range t.params {
			var (
				pos Position
				elems []Value
				s string
			)
			if pair, ok := a.(*Pair); !ok {
				erro(of(ctx,a), "unexpected value: %T %v", a, a).debug(1)
				return
			} else if s = pair.Key.Strval(at(ctx, pair.Key.Position())); s == "" {
				erro(of(ctx,a), "empty key: %T %v", pair.Key, pair.Key).debug(1)
				return
			} else if g, ok := pair.Value.(*Group); ok {
				pos = pair.Value.Position()
				elems = g.Elems
			} else {
				pos = pair.Value.Position()
				elems = append(elems, pair.Value)
			}

			var m = vars[s]
			m.elems = mergex(at(ctx, pos), plain, elems...)
			if n := len(m.elems); n > num { num = n }
			vars[s] = m // overwrite
		}
		for i := 0; i < num; i += 1 {
			var _1trivial bool
			var m = make(map[string]Value)
			for name, s := range vars {
				var elem Value
				if i < len(s.elems) { elem = s.elems[i] }
				if false { warn(ctx, "%s %v", name, elem) }
				_1trivial = isTrivial(elem)
				m[name] = elem
			}
			_1trivial = _1trivial && len(m) == 1

			if len(m) > 0 && !_1trivial { p.templateBlock(ctx, t, m, params) }
			count += 1
		}
	default:
		erro(p, "expand template %v: %v", t.verb, params).debug(1)
	}
}
func (p *parser) callTemplate(ctx Context, t *template, name Value, args []Value) {
	var count int64
	defer func(t time.Time, pos token.Pos, tok token.Token, lit string, state scanner.State) {
        if d := time.Now().Sub(t); d > 1999*time.Millisecond {
			var c = time.Duration(count)
            infostack(ctx, 3, "%v: slow: %v, %v, %d*%v", name, d, count, d/c).debug(1)
        }
		p.pos, p.tok, p.lit	 = pos, tok, lit
		p.scanner.SetState(state)
	} (time.Now(), p.pos, p.tok, p.lit, p.scanner.GetState())

	p.scanner.SetState(t.state)
	p.pos, p.tok, p.lit = t.pos, t.tok, t.lit

	// NOTE: a new scope is required for template expansion
	var loader = ctx.loader()
	defer loader.closeScope(loader.openScope("template call "))

	var params = merge(t.params...)
	for i, param := range params {
		var s = param.Strval(ctx)
		if def, alt := loader.def(p.Position(), s); alt != nil {
			erro(at(ctx,param.Position()), "duplicated parameter '%s'", s).debug(1)
		} else if i < len(args) {
			def.set(ctx, DefAuto, args[i])
		}
	}

	for p.tok != token.EOF && p.pos < t.endPos {
		if p.tok == token.LINEND ||
			(p.tok == token.COMMENT && p.lineComment != nil) {
			p.next(true)
		} else {
			p.clause()
		}
	}
}
func (p *parser) templateCall(ctx Context, name Value, args []Value) {
	for _, tmpl := range p.templates {
		if tmpl.name != nil && eq(ctx, tmpl.name, name) {
			p.callTemplate(at(ctx, tmpl.name.Position()), tmpl, name, args)
			return
		}
	}
	erro(of(ctx,name), "undefined template: %v", name).debug(1)
}
func (p *parser) template(ctx Context) {
	defer ctx.checkErrors(true)

	var (
		starting = p.Position()
		arged *Argumented
		verb string
	)
	p.expect(token.TEMPLATE) // expect and skip 'template'
	p.spaces()

	var op = p.expr(ctx, false) ; p.spaces()
	if p.tok == token.EOF {
		erro(of(ctx,op), "unexpected end of file after %v", op).debug(1)
		return
	} else if w, ok := op.(*bareword); ok {
		verb = w.string
	} else if arged, ok = op.(*Argumented); !ok {
		erro(of(ctx,op), "unknown template verb: %v", op).debug(1)
		return
	}

	switch verb {
	case "end", "expand":
		erro(of(ctx,op), "unexpected verb: %s", verb).debug(1)
		return
	case "": if arged != nil {
		p.expect(token.LINEND)
		p.templateCall(ctx, arged.value, arged.args)
		return //true
	}}

	var params = mergex(ctx, plain, p.values(ctx, false)...)
	// TODO: parse template options - parseOpts

	var tmpl = &template{ state:p.scanner.GetState(), pos:p.pos, tok:p.tok, lit:p.lit }
	if verb == "def" {
		if len(params) != 1 {
			erro(at(ctx,starting), "too many def params: %v", params)
			return
		} else if arged, ok := params[0].(*Argumented); !ok {
			erro(at(ctx,starting), "too many def params: %v", params)
			return
		} else {
			tmpl.name, tmpl.params = arged.value, arged.args
			p.templates = append(p.templates, tmpl)
		}
	} else {
		tmpl.verb, tmpl.params = verb, params
	}

	var nested int
	for p.tok != token.EOF {
		if p.tok == token.LINEND || p.lineComment != nil {
			if p.spaces(); p.tok == token.EOF { return }
		}
		if p.tok != token.TEMPLATE { p.step(); continue }
		if false { info(p, "%v: %v", p.tok, p.scanner.GetState()).debug(1) }

		var pos, stop = p.pos, p.stop
		if p.next(true); p.tok != token.BAREWORD && p.tok != token.FOREACH {
			erro(p, "%v: %v (nested=%v)", p.tok, p.lit, nested).debug(1)
			return
		}

		if p.lit == "def" || p.lit == "for" || p.lit == "foreach" {
			nested += 1
		} else if p.lit == "expand" && (verb == "for" || verb == "foreach") {
			if nested > 0 { nested -= 1 } else {
				p.next(true) // consumes the 'expand'
				params := p.values(ctx, false)
				p.expect(token.LINEND)
				p.stop = pos
				p.templateExpand(ctx, tmpl, params)
				p.stop = stop
				return //true
			}
		} else if p.lit == "end" && (verb == "def") {
			if nested > 0 { nested -= 1 } else {
				p.next(true) // consumes the 'end'
				p.expect(token.LINEND)
				state := p.scanner.GetState()
				tmpl.end, tmpl.endPos = &state, pos
				return //true
			}
		} else if false {
			erro(p, "unexpected template: %v (verb=%s, nested=%v)", p.tok, verb, nested).debug(1)
			return
		} else {
			continue
		}
	}
}

func (p *parser) clause() {
	if false && t_traverse.enabled {
		defer un(tracef(t_traverse, "clause(%v, %v)", p.tok, p.pos))
	}

	var x Value
	var ctx = p.posit()
	defer func() {
		if options.debugParsing("clause") {
			warn(ctx, "parser.clause: %s %v; %v %v", typeof(x), x, p.tok, p.lit).debug(6)
		}
		if ctx.checkErrors(true) > 0 {
			errostack(ctx, 5, "clause: %s(%v); %v %v", typeof(x), x, p.tok, p.lit).debug(4)
			fail(p.Position(), "parser.clause")
		}
	} ()

	var tok = p.tok // TODO: allow assigns like: `eval := xxx`
	if /* TODO: p.spaces(); !p.tok.IsAssign() && !p.tok.IsRuleDelim() */true {
		switch tok {
		case token.USE:
			erro(ctx, "`%v` unexpected here", p.tok).debug(10)
			return
		case token.INCLUDE:
			p.spec(ctx, tok, p.expect(tok), p.include)
			return
		case token.FILES:
			p.spec(ctx, tok, p.expect(tok), p.files)
			return
		case token.ASSERT:
			p.spec(ctx, tok, p.expect(tok), p.assert)
			return
		case token.APPEND:
			p.spec(ctx, tok, p.expect(tok), p.append)
		case token.EVAL:
			p.spec(ctx, tok, p.expect(tok), p.eval)
			return
		case token.COLON:
			p.specialRule()
			return
		case token.TEMPLATE:
			p.template(ctx)
			return
		case token.FOREACH:
			warn(ctx, "%v %v", p.tok, p.lit).debug(1)
			p.next(true)
			return
		case token.DONE:
			warn(ctx, "%v %v", p.tok, p.lit).debug(1)
			p.next(true)
			return
		default:
			x = p.expr(ctx, true)
			p.spaces()
		}
	}

	if t_traverse.enabled { defer un(trace(t_traverse, "Clause(?)")) }

	if p.tok.IsAssign() {
		if options.debugParsing("define") {
			warn(p, "parser.clause: %s(%v); %v %v", typeof(x), x, p.tok, p.lit).debug(1)
			ctx.checkErrors(true)
		}
		p.define(ctx, p.tok, x)
		return
	}

	var list = []Value{ x }
	if !p.tok.IsRuleDelim() {
		list = append(list, p.left(ctx)...)
	}

	if p.tok.IsRuleDelim() {
		if options.debugParsing("rule") {
			warn(p, "parser.clause: %s(%v); %v %v", typeof(x), x, p.tok, p.lit).debug(1)
			ctx.checkErrors(true)
		}
		p.rule(specialRuleNor, nil, list)
		return
	}

	if p.tok != token.EOF { return }

	var isIncludingConf = p.isIncludingConf
	if false {
		var loader = ctx.loader()
		for pp := loader.p; !isIncludingConf && pp != nil && pp != p; {
			isIncludingConf = pp.isIncludingConf
			pp = loader.p
		}
	} else if isIncludingConf {
		warn(ctx, "bad clause: %v (kit=%s) after %v", p.tok, p.lit, list).debug(10)
	} else {
		erro(ctx, "bad clause: %v (lit=%s) after %v", p.tok, p.lit, list).debug(10)
	}
}

type projectDeclOpts struct {
	configure bool `c,conf,configure` // detects dotConfigure if empty
	configureName string `configure-with`
	noDock bool `n,nd,nod,nodock,no-dock` // don't load container project
    traveUseLoop bool `b,break;l,loop` // don't recursively use this project
    multiUseAllowed bool `m,multi`  // this project is used multiple times
	final bool `f,final`
}

func (p *parser) file(ctx Context) *parsedFile {
	if options.traceLaunch { defer un(trace(t_launch, "parser.file")) }
	if t_traverse.enabled  { defer un(trace(t_traverse, "File '"+p.scanner.File().Name()+"'")) }
    if false { defer un(tracef(t_traverse, "file(%s)", p.scanner.File().Name())) }

	// Don't bother parsing the rest if we had errors scanning the first token.
	// Likely not a Go source file at all.
	if ctx.countErrors() > 0 { return nil }

	var (
		abs, rel, tmp string
		ident *barecomp
		identStr string
		implicitBase string // aka. foo.bar.Baz implicitly load base 'foo/bar'
		keyword  = p.tok
		filename = p.scanner.File().Name()
		isMainFile = isEntryFileName(filename)
		position = ctx.Position()
		loader = ctx.loader()
	)
	assert(loader != nil, "nil loader")
	assert(loader == loader, "bad loader")
	defer loader.closeScope(loader.openScope(fmt.Sprintf("file %s", filename)))

	if options.debugFileEntry {
		warn(p, "parser.file: %v %v", p.tok, p.scanner.GetState()).debug(1)
	}

	/*if filename == confinitFilename {
        abs, rel = context.workdir, "."
        tmp = joinTmpPath(context.workdir, rel)
	} else*/ {
		if loader.mode&Flat != 0 {
			abs = ctx.Project().absPath
		} else {
			abs = filepath.Dir(filename)
		}
		rel, _ = filepath.Rel(loader.WorkDir(), abs)
		tmp = joinTmpPath(ctx, loader.WorkDir(), rel)
	}

	if s := ctx.Scope(); s != nil {
		//defer p.closeScope()
		var d *def
		if loader.mode&Flat == 0 {
			d, _ = loader.def(position, ".")
			d.set(ctx, DefAuto, MakePathStr(position, rel))

			d, _ = loader.def(position, "/")
			d.set(ctx, DefAuto, MakePathStr(position, abs))

			d, _ = loader.def(position, "CTD") // Current Temp Directory, TODO: make it $:ctd:
			d.set(ctx, DefAuto, MakePathStr(position, tmp))

			d, _ = loader.def(position, "CWD") // Current Work Directory, TODO: make it $:cwd:
			d.set(ctx, DefAuto, MakePathStr(position, abs))
		} else if d = s.FindDef("/");   d == nil {
			erro(ctx, "/ not in the scope: %v", s.comment)
		} else if d = s.FindDef(".");   d == nil {
			erro(ctx, ". not in the scope: %v", s.comment)
		} else if d = s.FindDef("CTD"); d == nil {
			erro(ctx, "CTD not in the scope: %v", s.comment)
		} else if d = s.FindDef("CWD"); d == nil {
			erro(ctx, "CWD not in the scope: %v", s.comment)
		}
	} else {
		erro(ctx, "opened invalid scope for %s", filename).debug(1)
		return nil
	}

	switch position = p.Position(); keyword {
	case token.PACKAGE, token.MODULE:
		erro(ctx, "deprecated keyword '%s'", keyword).debug(1)
		return nil
	case token.CONFIGURE:
		switch p.next(true); p.tok {
		case token.DOT:
			if err := loader.ParseConfigDir(abs, abs); err != nil {
				erro(ctx, "parsing configure directory failed, '%s': %v", abs, err)
			} else {
				p.next(true) // skip the '.' token and consequence spaces
			}

			var basename = filepath.Base(filepath.Dir(filename))
			ident = MakeBarecomp(position, MakeBareword(position, basename))

		default:
			erro(ctx, "unknown configuration '%v', currently only 'configure .' is supported", p.tok)
		}
	case token.PROJECT:
		if loader.mode&Flat != 0 { erro(ctx, "forbidden `%v` in flat file", p.tok) }

		p.next(true)

		// Options are *Flag or *Pair of a Flag.
		var (
			opts projectDeclOpts
			optVals []Value
			pos Position
		)
		for p.tok == token.MINUS {
			var opt = p.expr(ctx, false);  p.spaces()
			optVals = append(optVals, opt)
			if !pos.IsValid() { pos = opt.Position() }
		}
		if !pos.IsValid() { pos = p.Position() }
		if a := parseOpts(ctx, &opts, 0, optVals...); len(a) > 0 {
			for _, v := range a {
				erro(of(ctx,v), "unknown option '%v'", v).debug(1)
			}
			return nil
		}

		var linfo = uni.globe.loads[len(uni.globe.loads)-1]

		// Smart-lang spec:
		//   * the project clause is not a declaration;
		//   * the project name does not appear in any scope.
		if p.tok == token.LPAREN || p.tok == token.EOF || p.tok == token.LINEND || p.lineComment != nil {
			var dir = filepath.Dir(filename)
			if linfo.loadee != nil && linfo.absDir == dir {
				ident = MakeBarecomp(position, MakeBareword(position, linfo.loadee.name))
			} else if name := filepath.Base(filename); name == dotBase || name == dotConfigure {
				// NOTE: loading the .base or .configure file
				ident = MakeBarecomp(position, MakeBareword(position, name))
			} else if base := filepath.Base(dir); base != "" {
				// TODO: validate basename as a valid identifier
				ident = MakeBarecomp(position, MakeBareword(position, base))
			} else {
				erro(ctx, "invalid file: %v", filename).debug(1)
			}
		} else if p.tok == token.TILDE {
			/*if filename == confinitFilename {
                ident = &ast.Bareword{ ValuePos:pos, Value:"~" }
            } else*/ if ext := filepath.Ext(filename); ext != ".smart" {
				erro(p, "`%v` not a smart file", filepath.Base(filename)).debug(1)
			} else if s := strings.TrimSuffix(filepath.Base(filename), ext); s != "" {
				ident = MakeBarecomp(position, MakeBareword(position, s))
			} else {
				erro(p, "`%v` not tilde name", filepath.Base(filename)).debug(1)
			}
			p.next(true) // skip tilde
		} else {
			// var t = p.tok
			var implicitBaseSegs []string
			ident = MakeBarecomp(p.Position())
		ForProjectName:
			for p.tok != token.EOF && p.tok != token.SPACE {
				if w := p.bare(false); w == nil {
					erro(at(ctx,ident.Position()), "expecting a bareword").debug(1)
				} else if word, ok := w.(*bareword); !ok {
					erro(at(ctx,ident.Position()), "expecting a bareword: %v (%T)", w, w).debug(1)
				} else if ident.Combine(ctx, word); p.tok == token.DOT {
					ident.Combine(ctx, &punctuation{valbase{p.Position()}, p.tok}) // TODO: parse to Qualiword
					implicitBaseSegs = append(implicitBaseSegs, word.string)
					p.step() // '.'
				} else { break ForProjectName }
			}
			if p.spaces(); len(ident.Elems) == 0 {
				// erro(ctx, "package name is empty (tok=%v %v)", t, p.tok).debug(1)
				// return nil
			} else if len(implicitBaseSegs) > 0 {
				implicitBase = filepath.Join(implicitBaseSegs...)
			}
		}

		if identStr = ident.Strval(ctx); linfo.loadee != nil && identStr != linfo.loadee.name {
			warn(at(ctx,ident.position), "%s: declare multiple project in the same directory", ctx.Project()).debug(24)
		} else if identStr == "_" && loader.mode&DeclarationErrors != 0 {
			erro(at(ctx,ident.Position()), "package name '_' is preserved").debug(1)
			return nil
		}

		// Don't bother parsing the rest if we had errors parsing the package clause.
		// Likely not a Go source file at all.
		if n := loader.countErrors(); n > 0 {
			erro(p, "got %d errors parsing file: %s", filename).debug(1)
			return nil
		}

		var (
			loaderProj  = loader.project
			_, declared = linfo.declares[identStr]
		)
		if (loader.mode&Flat == 0) && loader.declare(at(ctx, ident.Position()), keyword, ident, identStr, &opts) {
			// Change the 'default' owners into the new declared project
			if s := ctx.Scope(); s != nil {
				if def := s.FindDef("."  ); def != nil { def.owner = ctx.Project() }
				if def := s.FindDef("/"  ); def != nil { def.owner = ctx.Project() }
				if def := s.FindDef("CTD"); def != nil { def.owner = ctx.Project() }
				if def := s.FindDef("CWD"); def != nil { def.owner = ctx.Project() }
			} else {
				erro(ctx, "file scope is nil").debug(1)
			}
			// NOTE: do.smart is always the first loaded, so the loadee will be pointed to it
			if linfo.loadee == nil { linfo.loadee = ctx.Project() }
			defer func(proj *Project) {
				if false && loaderProj != nil && filepath.Base(filename) == entryFileName {
					var ctx = at(ctx, ident.Position())
					assert(loader.project == proj, "diverged project: %v != %v", loader.project, proj)
					//applyUseeVars(ctx, loaderProj, p.project)  // aka. ABC += $(use.ABC)
					applyUserVars(ctx, loaderProj, loader.project) // aka. use.ABC += $(use.ABC)
					if loaderProj.name == "llvm.Analysis" {
						warn(ctx, "%v, %v", loaderProj, loader.project).debug(24)
					}
				}
				loader.closeCurrent(ident, identStr)
			} (loader.project)

			isMainFile = isMainFile && !declared;
		}

		var basePos Position
		if implicitBase != "" { basePos = pos } else { basePos = p.Position() }
		if p.tok == token.LPAREN {
			var bits = p.setbit(parseGroup)
			for p.tok != token.EOF {
				for p.next(true); !p.isEndOfList(false); {
					p.spaces()
					param := p.expr(ctx, false)
					p.spaces()

					//if p.lineComment != nil  { break }
					//if p.tok == token.LINEND { break }
					if p.tok == token.EOF {
						erro(at(ctx,basePos), "unexpected end of file while parsing bases").debug(1)
						p.setbits(bits) ; return nil
					}

					var (
						ctx = at(ctx, param.Position())
						t = parseOpts(ctx, &opts, 0, param)
					)
					if keyword == token.PACKAGE || opts.final {
						// No bases for PACKAGE or final project
					} else if !loader.bases(ctx, linfo, "", merge(t...)...) {
						erro(of(ctx,param), "loading base '%v' failed", t).debug(1)
						p.setbits(bits) ; return nil
					}
				}
				if p.tok != token.COMMA { break }
			}
			p.setbits(bits)
			p.expect(token.RPAREN)
			if false { defer func() { warn(ctx, "%v", ident).debug(32) } () }
		} else if !loader.bases(ctx, linfo, implicitBase) { // for special bases, e.g. .base
			erro(at(ctx,basePos), "loading bases failed").debug(1)
			return nil
		}

		if p.spaces(); p.tok != token.EOF { p.linend() }
		if keyword != token.PACKAGE {
			loader.configuration(ctx, linfo, ident, identStr, declared)
			if !opts.noDock { loader.loadProjectContainer(ctx, ident, identStr) }
		}
	case token.EOF:
		return nil
	default:
		if loader.mode&Flat == 0 {
			p.expected(p.pos, "configure, project, module or package keyword")
		}
	}

	if !ddd && options.debugFiles != nil {
		for _, s := range options.debugFiles {
			if ddd = strings.Contains(filename, s); ddd { break }
		}
	}

	var auto = (loader.mode&Flat == 0) && isMainFile //&& isEntryFileName(filename)
	if auto { loader.autoAfter(p.posit(), "declare") }
	if loader.mode&ModuleClauseOnly == 0 {
		if loader.mode&Flat == 0 {
			ForInit: for p.tok != token.EOF {
				switch tok := p.tok; tok {
				case token.LINEND: p.next(true) // skip empty lines
				case token.USE:
					p.spec(ctx, tok, p.expect(tok), p.use)
				case token.ASSERT:
					p.spec(ctx, tok, p.expect(tok), p.assert)
				case token.APPEND:
					p.spec(ctx, tok, p.expect(tok), p.append)
				case token.EVAL:
					p.spec(ctx, tok, p.expect(tok), p.eval)
				case token.TEMPLATE:
					p.template(ctx)
				case token.FOREACH:
					warn(ctx, "%v %v", p.tok, p.lit).debug(1)
					p.next(true)
				case token.DONE:
					warn(ctx, "%v %v", p.tok, p.lit).debug(1)
					p.next(true)
				default:
					if p.tok.IsKeyword() { break ForInit }
					var x = p.expr(ctx, true); p.spaces()
					if p.tok.IsAssign() { p.define(ctx, p.tok, x) } else
					if p.tok.IsRuleDelim() {
						if ctx.Project() == nil {
							erro(ctx, "no project declared before defining rules")
						} else {
							x = p.rule(specialRuleNor, nil, []Value{x})
						}
						break ForInit
					} else {
						erro(ctx, "unexpected %v (after %v)", p.tok, x)
					}
				}
			}
		}
		if false && auto { loader.autoAfter(p.posit(), "amid") }
		if loader.mode&ImportsOnly == 0 { // rest of module body
			for /* p.totalErrors() == 0 && */ p.tok != token.EOF {
				if p.tok == token.LINEND || (p.tok == token.COMMENT && p.lineComment != nil) {
					p.next(true)
				} else if p.clause(); ctx.checkErrors(true) > 0 {
					break
				}
			}
		}
	}
	if auto { loader.autoAfter(ctx, "appendix") }
	if ddd && options.debugFiles != nil { ddd = false }

	return &parsedFile{
		// TODO: doc: doc,
		// TODO: comments: p.comments,
		keyword:  keyword,
		position: position,
		name:     ident,
		scope:    ctx.Scope(),
		use:      p.imports,
	}
}
