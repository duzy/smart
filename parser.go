///
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"path/filepath"
	"runtime/pprof"
	"runtime"
	"strconv"
	"strings"
	"unicode"
	"regexp"
	"sync"
	"time"
	"fmt"
	"os"
)

type parseBits uint64
type specialRule int

const (
	parseGroup parseBits = 1<<iota // 0000000000000000000001
	parseArged         // 0000000000000000000010
	parseCall          // 0000000000000000000100
	parseDOT           // 0000000000000000001000
	parseDOTDOT        // 0000000000000000010000
	parseDepend0       // 0000000000000000100000
	parseGLOB          // 0000000000000001000000
	parseModifier      // 0000000000000010000000
	parsePATH          // 0000000000000100000000
	parsePERC          // 0000000000001000000000
	parseREXP          // 0000000000010000000000
	parseSELECT_PROP   // 0000000000100000000000
	parseURL           // 0000000001000000000000

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
	parseNoArg    = parseSELECT_PROP | parseDOT | parseDOTDOT /* | parsePATH */ | parsePERC
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
	keyword Token // project, package or module
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
	state ScanState
	end *ScanState
	pos, endPos Pos   // token position
	tok Token // one token look-ahead
	lit string      // token literal
	verb string
	name Value // if only 'def', TODO: considering []Value for nested template defs?
	params []Value
}

type parser struct {
	Context
	facet

	scanner Scanner

	// Comments
	comments  []*CommentGroup
	leadComment *CommentGroup // last lead comment
	lineComment *CommentGroup // last line comment

	// Next token
	pos, stop Pos // parsing and stop position
	tok Token // one token look-ahead
	lit string // token literal

	templates []*template

	// Error recovery
	// (used to limit the number of calls to syncXXX functions
	// w/o making scanning progress - avoids potential endless
	// loops across multiple parser functions during error recovery)
	//syncPos Pos // last synchronization position
	//syncCnt int       // number of calls to syncXXX without progress

	// Non-syntactic parser control
	exprLev int  // < 0: in control clause, >= 0: in expression
	inRhs   bool // if set, the parser is parsing a rhs expression

	bits parseBits
    isIncludingConf bool // including configuration

	// Ordinary identifier scopes
	imports []*usespec // list of imports

	targets []Value // targets of current rule
	params []*auto // parameters of current rule
	autos []*auto // autos in the current context
	autop *Position // valid if in auto
	dialect string // recipe dialect of current rule
	configure bool // is parsing configure program?

	dd bool // helps debug parsing via `eval -dd=true{}`
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
	// (it is ILLEGAL), so don't print it .
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
	if false && p.lit == "none" { warn(p, "%v %v", p.tok, p.lit).debug(64); p.dia().flush() }
	if false && p.tok == EOF {
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
	for p.tok == COMMENT && p.scanner.File().Line(p.pos) <= endline+n {
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
	if p.scan(); p.tok == COMMENT {
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
		for p.tok == COMMENT {
			comment, endline = p.consumeCommentGroup(1)
		}

		if endline+1 == p.scanner.File().Line(p.pos) {
			// The next token is following on the line immediately after the
			// comment group, thus the last comment group is a lead comment.
			p.leadComment = comment
		}
	}

	// if p.tok != LINEND && p.lineComment != nil { p.tok = LINEND }

	if p.dd {
		var t = warn(p, "%v %v %v", p.tok, p.lit, p.scanner.ScanState)
		if p.tok == COMPOUND { t.debug(12) }
		if p.tok == LINEND { t.debug(24) }
		p.dia().flush()
	} else if false {
		p.scanner.Debug = false
	}
}
func (p *parser) next(ws bool) { if p.step(); ws { p.spaces() } }
func (p *parser) spaces() {
	for p.lineComment == nil && p.tok != EOF {
		if p.tok == SPACE || (p.tok == RECIPE && p.bits&parseRecipeBuiltin != 0) {
			p.step()
		} else if p.tok == ESCAPE && p.lit == "\n" {
			if p.step(); p.tok == LINEND || p.lineComment != nil { break }
			if p.bits&parseRecipeBuiltin != 0 {
				TokFor: for p.tok != EOF {
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
func (p *parser) loc(pos Pos) Position { return Position(p.scanner.File().Position(pos)) }
func (p *parser) ctx() Context { return &positionContext{p, p.Position()} }

// A bailout panic is raised to indicate early termination.
type bailout struct{}

func (p *parser) expected(pos Pos, msg string, a... interface{}) {
	if len(a) > 0 { msg = fmt.Sprintf(msg, a...) }
	if msg = "expected " + msg; pos == p.pos {
		// the error happened at the current position;
		// make the error message more specific
		if p.tok == SEMICOLON && p.lit == "\n" {
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

func (p *parser) expect(tok Token) Pos {
	var pos = p.pos
	if p.tok != tok { p.expected(pos, "'"+tok.String()+"'") }
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
/*func (p *parser) _safePos(pos Pos) (res Pos) {
	defer func() {
		if recover() != nil {
			res = Pos(p.scanner.File().Base() + p.scanner.File().Size()) // EOF position
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
	case BAREWORD: // okay
	case UNDEF:
		if p.tok == LBRACE { // undef{}, undef{ ... }
			if p.next(true); p.tok == RBRACE {
				x = &undef{&none{valbase{p.Position()}, nil}}
				p.step()
			} else if v := p.expr(p, false); v != nil {
				x = &undef{v}
				p.expect(RBRACE)
			} else {
				erro(at(p,pos), "undef invalid expression: %v, %v", p.tok, p.lit).debug(1)
			}
			return
		}
	case BARE: // TODO: bare{ ... }
		if p.tok == LBRACE { // file{ ... }
			erro(p, "TODO: %v", tok).debug(1) ; p.dia().flush()
			return
		}
	case REGEX: // TODO: regex{ ... }
		if p.tok == LBRACE { // file{ ... }
			erro(p, "TODO: %v", tok).debug(1) ; p.dia().flush()
			return
		}
	case FILE:
		if p.tok == LBRACE { // file{ ... }
			erro(p, "TODO: %v", tok).debug(1) ; p.dia().flush()
			return
		}
	case BIN, OCT, INT, HEX, FLOAT:
		if p.tok == LBRACE { // answer{...}, bool{...}
			if p.next(true); p.tok == RBRACE {
				switch p.step(); tok {
				case BIN:   x = MakeBin(pos, 0)
				case OCT:   x = MakeOct(pos, 0)
				case INT:   x = MakeInt(pos, 0)
				case HEX:   x = MakeHex(pos, 0)
				case FLOAT: x = MakeFloat(pos, 0.)
				}
			} else if v := p.expr(p, false); v == nil {
				// TODO: true{ expr }, yes{ expr }, ...
				erro(at(p,pos), "%s expects: %v, not %v %v", tok, RBRACE, p.tok, p.lit).debug(1)
			} else if p.spaces(); p.tok == RBRACE {
				if p.step(); tok == FLOAT {
					var n, _ = v.Float(p)
					return MakeFloat(pos, n)
				}
				switch n, _ := v.Integer(p); tok {
				case BIN: return MakeBin(pos, n)
				case OCT: return MakeOct(pos, n)
				case INT: return MakeInt(pos, n)
				case HEX: return MakeHex(pos, n)
				}
			}
			return
		}
	case ANSWER, BOOL, NONE:
		if p.tok == LBRACE { // answer{...}, bool{...}
			if p.next(true); p.tok == RBRACE {
				switch pos := p.Position(); tok {
				case ANSWER: x = makeAnswer(pos, false)
				case BOOL: x = MakeBoolean(pos, false)
				case NONE: x = MakeNone(pos)
				}
				p.step()
				return
			}

			if tok == NONE {
				x = &none{valbase{pos}, p.expr(p, false)}
				p.spaces()
				p.expect(RBRACE)
				return
			}

			var ( pos = p.Position(); v bool )
			switch p.tok {
			case TRUE, YES: v = true
			case FALSE, NO: v = false
			default:
				if t := p.expr(p, false); t != nil {
					v = t.True(p)
				} else {
					erro(at(p,pos), "undef invalid expression: %v, %v", p.tok, p.lit).debug(1)
				}
			}
			p.spaces()
			switch p.expect(RBRACE); tok {
			case ANSWER: x = makeAnswer(pos, v)
			case BOOL: x = MakeBoolean(pos, v)
			}
			return
		}
	case TRUE, YES, FALSE, NO, NULL:
		if p.tok == LBRACE { // true{}, false{}, yes{}, no{}, null{}
			if p.next(true); p.tok == RBRACE {
				switch p.step(); tok {
				case YES , NO   : x = makeAnswer( pos, tok == YES)
				case TRUE, FALSE: x = MakeBoolean(pos, tok == TRUE)
				case NULL: x = MakeNull(pos)
				}
			} else {
				// TODO: true{ expr }, yes{ expr }, ...
				erro(at(p,pos), "%s expects: %v, not %v %v", tok, RBRACE, p.tok, p.lit).debug(1)
			}
			return
		}
	case AT, DOT, DOTDOT: // TODO: parse DOT into Qualiword
		return &punctuation{valbase{pos}, tok} // lit = tok.String() // Special bareword.
	default:
		if tok.IsKeyword() { lit = tok.String() } else {
			if true {
				erro(at(p,pos), "%v %v -> %v %v", tok, lit, p.tok, p.lit)
			} else {
				p.expect(BAREWORD)
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
		ctx = p.ctx()
		tok = p.tok // the arrow '->' or '=>'
		loader = ctx.loader()
		proj = loader.Project()
	)
	p.step() // skip '->' or '=>'

	switch t := lhs.(type) {
	case *selection:
		if v := t.value(at(ctx, t.Position()), ident); isNull(v) {
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
            } else if !isNull(o) {
				lhs = o
			} else if tok == SELECT_PROG2 {
				res = MakeNull(ctx.Position()) // ignore
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
        } else if !isNull(o) {
			lhs = o
		} else if tok == SELECT_PROG2 {
			res = MakeNull(ctx.Position()) // ignore
			return
		} else {
			erro(of(ctx,lhs), "'%v' is undefined", lhs).debug(1)
			return
        }
	case *GlobPattern:
		if o, y := optionalize(ctx, lhs); y { lhs = o } else {
			erro(of(ctx,lhs), "selection of '%v' is undefined", lhs).debug(1)
		}
	}

	if rhs := p.selector(ctx); isNull(rhs) {
		res = MakeNull(ctx.Position())
	} else {
		if v, y := optionalize(ctx, rhs); y { rhs = v } // foo→bar?
		res = MakeSelection(ctx.Position(), tok, lhs, rhs)
	}

	if (p.tok == SELECT_PROP || p.tok == SELECT_PROG1 || p.tok == SELECT_PROG2) {
		res = p.selectExpr(res) // Continue the selection recursivly.
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

func (p *parser) isEndOfList(lhs bool) bool {
	// If there's a comment right after the parsed expression, we break
	// the expression list to treat the end-of-line comment like a LINEND.
	if p.lineComment != nil || p.tok.IsListDelim() || (lhs && p.tok.IsAssign()) {
		return true
	}
	if (p.bits&parseRecipe != 0) && p.tok == RECIPE { // TODO: using p.isRecipeStart()
		return true
	}
	return false
}

func (p *parser) isEndOfURL(lhs bool) bool {
	return p.tok == SPACE || p.isEndOfLine() || p.isEndOfList(lhs)
}

func (p *parser) isEndOfDotConcat(lhs bool) bool {
	// Expressions like `FOO.BAR(xxx)` does not count.
	switch p.tok {
	case SPACE, LPAREN, COLON, PCON, ASSIGN: fallthrough
	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2: return true
	}
	return p.isEndOfLine() || p.isEndOfList(lhs)
}

func (p *parser) depends(ctx Context, normal bool) (list []Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Depends")) }
	for p.tok != SEMICOLON && p.tok != BAR && !p.isEndOfLine() {
		if p.tok == COLON { // FIXME: this check is not working!
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
			if ctx.dia().flush() > 0 {
				erro(ctx, "depend: %T %v", val, val).debug(1)
				return
			}

			if normal {
				if g, y := val.(*group); y && len(g.Elems) == 1 {
					if g, y = g.Elems[0].(*group); y {
						p.ruleParams(ctx, g.Elems)
						continue
					}
				}
			}

			list = append(list, val)
			if p.tok == SPACE { p.next(true) } //p.spaces()
		}
	}
	return
}
// If lhs is set, result list elements which are identifiers are not resolved.
func (p *parser) values(ctx Context, lhs bool) (list []Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "List")) }
	for p.spaces(); !p.isEndOfList(lhs); p.spaces() { var pos = p.pos
		if val := p.expr(ctx, lhs); p.pos == pos {
			erro(p, "nothing: %v %v; %v", p.tok, p.lit, list).debug(1)
			break
		} else { list = append(list, val) }

		// If there's a comment right after the parsed expression, we break
		// the expression list to treat the end-of-line comment like a LINEND.
		if p.lineComment != nil  { break }
		if p.tok == LINEND { break }
		if p.tok == EOF    { break }
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

func (p *parser) group(lhs bool) *group {
	if t_traverse.enabled { defer un(trace(t_traverse, "Group")) }

	defer p.setbits(p.setbit(parseGroup))
	p.clearbit(parseCall)

	var ctx = p.ctx()
	p.next(true)

	var elems, converted = p.values(ctx, false), false
	for p.tok != RPAREN && p.tok != EOF {
		// if p.tok == COMMA { warn(ctx, "%020b: %v %v", p.bits, p.tok, p.lit).debug(1) }
		// if p.tok == COMMA { p.next(true) }
		switch p.tok {
		case BAR, COMMA, SEMICOLON:
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
	p.expect(RPAREN)
	return MakeGroup(ctx.Position(), elems...)
}

func (p *parser) argumentedExpr(x Value) *argumented {
	if t_traverse.enabled { defer un(trace(t_traverse, "argumented")) }

	defer p.setbits(p.setbit(parseGroup))
	p.clearbit(parseCall)

	var ctx = p.ctx()
	p.next(true) // skip LPAREN

	var a = []Value{ p.list(ctx, false) }
	for p.tok != RPAREN && p.tok != LINEND && p.tok != EOF {
		switch p.tok {
		case COMMA: p.next(true) // skip COMMA
		case BAR, SEMICOLON:
			if false {
				a = append(a, p.punctuation())
				p.spaces()
			} else {
				erro(ctx, "unexpected punctuation: %v", p.tok).debug(1)
			}
		}
		a = append(a, p.list(p.ctx(), false))
	}
	p.expect(RPAREN)
	return makeArgumented(x, a...)
}

func (p *parser) globMeta() (x *GlobMeta) {
	pos, tok := p.Position(), p.tok
	p.step()
	return MakeGlobMeta(pos, tok)
}

func (p *parser) globRange() (x *GlobRange) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Glob")) }

	var ctx = p.ctx()
	p.expect(LBRACK) // skip '['

	chars := p.expr(ctx, false)
	p.expect(RBRACK) // skip ']'

	return MakeGlobRange(ctx.Position(), chars)
}

func (p *parser) globExpr(x Value) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "Glob")) }

	var pos = p.Position()
	var ctx = at(p, pos)
	var components []Value
	if !isNull(x) { components = []Value{ x } }

	// avoid nesting glob expressions
	defer p.setbits(p.setbit(parseGLOB))

ForGlobTok:
	for {
		if p.lineComment != nil { break ForGlobTok }
		switch p.tok {
		case SELECT_PROP, SELECT_PROG1, SELECT_PROG2, PCON, RPAREN, COMMA, SPACE, LINEND, EOF:
			break ForGlobTok
		case STAR, DAST, QUE: // * ** ?
			x = p.globMeta()
		case LBRACK:
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
		case COLON, DOLON,
			LPAREN, RPAREN,
			LBRACK, RBRACK,
			PCON,   SEMICOLON,
			COMMA,  SPACE,
			LINEND:
		case PERC: // %%
			p.step() // consume the second %
			position := p.Position()
			perc2 := MakePercPattern(position, nil, nil)
			if pos+2 == p.pos {
				switch p.tok {
				case PERC: // %%%
					erro(p, "too many %")
				case PCON: // FIXES: %%/xxx -> Path(%% xxx)
					x = MakePercPattern(position, x, perc2)
					return p.path(lhs, x)
				case COLON,    DOLON,
					LPAREN,    RPAREN,
					LBRACK,    RBRACK,
					LBRACE,
					SEMICOLON, COMMA,
					SPACE,     LINEND:
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

	var pos = p.Position()
	var rx string

ForRX:
	for p.expect(LBRACE); p.tok != EOF; p.scan() {
		var esc bool
		if esc = p.tok == ESCAPE; esc {
			if rx += "\\" + p.lit; p.lit == "Q" {
				for p.scan(); p.tok != EOF; p.scan() {
					if p.tok == ESCAPE {
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
		case RBRACE: p.scan() ; break ForRX
		case LBRACE:
			rx += "{"
			for p.expect(LBRACE); p.tok != EOF; p.scan() {
				if p.tok == RBRACE { break } else
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

	var err error
	var exp *regexp.Regexp
	if exp, err = regexp.Compile(rx); err != nil {
		errostack(at(p,pos), 3, "regexp: %v", err).debug(6)
	}
	return &RegexpPattern{valbase{pos}, exp} // TODO: correct regexp pattern value
}

func (p *parser) pair(x Value) *pair {
	if t_traverse.enabled { defer un(trace(t_traverse, "pair")) }

	var ctx = p.ctx()
	p.step()

	var y Value
	if p.isEndOfList(false) {
		y = MakeNull(ctx.Position())
	} else {
		y = p.expr(ctx, false)
	}
	return MakePair(ctx.Position(), x, y)
}

func (p *parser) flagExpr(lhs bool) *flag {
	if t_traverse.enabled { defer un(trace(t_traverse, "flag")) }

	var ctx = p.ctx()
	p.step() // skip dash '-'

	var x Value
	// flag expressions, excluding "-)" "-]" "-}" "-\n", "-=", "-:", etc.
	if p.isEndOfLine() || p.isEndOfList(false) || p.tok == SPACE || p.tok == RECIPE {
		x = MakeNull(ctx.Position())
	} else if false {
		x = p.expr(ctx, false)
	} else {
		x = p.unary(ctx, false)
		l: for p.tok == DOT || !(operator_beg < p.tok && p.tok < closure_beg) {
			switch p.tok {
			case COMMENT, HASH, SPACE, RECIPE, LINEND, EOF: break l
			case DELEGATE, CLOSURE: x = compose(ctx, x, p.unary(ctx, false))
			default: if p.tok.IsClosure() || p.tok.IsDelegate() {
				x = compose(ctx, x, p.unary(ctx, false))
			} else {
				break l
			}}
		}
	}
	if x == nil { erro(ctx, "nil flag name").debug(1) }
	return makeFlagValue(ctx.Position(), x)
}

func (p *parser) negExpr(ctx Context, lhs bool) *negative {
	if t_traverse.enabled { defer un(trace(t_traverse, "Negative")) }
	p.expect(EXC)
	return Negative(p.expr(ctx, lhs))
}

func (p *parser) punctuation() *punctuation {
	if t_traverse.enabled { defer un(trace(t_traverse, "punctuation")) }
	var pos, tok = p.Position(), p.tok
	p.step()
	return &punctuation{valbase{pos}, tok}
}

func (p *parser) literal(lhs bool) (v Value) {
	var ctx, tok, lit = p.ctx(), p.tok, p.lit
	p.step()

	// ESCAPE is handled in value.EscapeChar
    switch position := ctx.Position(); tok {
    case BAR: erro(ctx, "`|` is deprecated, changed the modifiers!")
    case BINARY:      v = ParseBin(position, lit)
    case OCTAL:       v = ParseOct(position, lit)
    case INTEGER:     v = ParseInt(position, lit)
    case HEXADECIMAL: v = ParseHex(position, lit)
    case FLOATING:    v = ParseFloat(position, lit)
    case DATETIME:    v = ParseDateTime(position, lit)
    case DATE:        v = ParseDate(position, lit)
    case TIME:        v = ParseTime(position, lit)
    case URI:         v = ParseURL(position, lit)
    case BAREWORD:    v = MakeBareword(position, lit)
    case STRING:      v = MakeString(position, lit)
    case ESCAPE:      v = MakeRaw(position, EscapeChar(lit))
    case RAW:         v = MakeRaw(position, lit)
    default: unreachable()
    }
	return
}

func (p *parser) compound(lhs bool) *Compound {
	var (
		ctx = p.ctx()
		lpos = p.pos
		elems []Value
	)
	p.step()

	defer p.setbits(p.setbit(parseCompound))

	for p.tok != EOF && p.tok != COMPOSED && p.tok != LINEND {
		if p.tok == RAW {
			elems = append(elems, p.literal(false))
		} else {
			elems = append(elems, p.expr(ctx, false))
		}
	}
	p.expect(COMPOSED)
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

	var ctx = p.ctx()
	for /*comp.End() == p.pos && */!p.isEndOfDotConcat(lhs) {
		comp.comp(ctx, p.composite(ctx, false))
		if p.tok == DOT /*&& comp.End() == p.pos*/ {
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

func makePathPun(ctx Context, tok Token) *PathPun {
	var r rune
    switch tok {
    case 0:            r = 0 // the tailing empty segment after '/', e.g. /foo/bar/
    case PCON:   r = '/' // TODO: should be NONE
    case TILDE:  r = '~'
    case DOT:    r = '.'
    case DOTDOT: r = '^' // 
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
	for p.tok == PCON {
		var pos = p.Position() // skips repeated '/' sequence
		for p.step(); p.tok == PCON; p.step() { pos = p.Position() }
		switch p.tok {
		case RPAREN, LPAREN, RBRACE, LBRACE, COMMA, SPACE, LINEND:
			// Encountered the tailing '/', append 'zero' segment.
			path.Elems = append(path.Elems, MakePathPun(pos, 0))
			break BuildPath
		}

		var x = p.composite(ctx, false)
		path.Elems = append(path.Elems, x)
		if p.tok == SPACE || p.isEndOfLine() {
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
			erro(ctx, "TODO: URL path: %v (%T) (next: %s (%s))", scheme, scheme,  p.tok, p.lit).debug(1)
			res = MakeNull(p.Position())
			return
		}
	} else if !p.isEndOfURL(lhs) {
		erro(at(ctx, p.loc(colon1)), "TODO: URL: %v (%T) (next: %s (%s))", scheme, scheme, p.tok, p.lit).debug(1)
		res = MakeNull(p.Position())
		return
	}

	if !p.isEndOfURL(lhs) {
		userOrHost := p.composite(ctx, false)
		if p.tok == COLON {
			url.Username, colon2 = userOrHost, p.pos
			p.step() // ':'
			if p.tok != AT && p.tok != PCON && !p.isEndOfURL(lhs) {
				url.Password = p.composite(ctx, false)
			}
		} else {
			url.Host = userOrHost
		}
		if p.tok == AT {
			p.step() // '@'
		}
	}
	if url.Host == nil && colon2 == NoPos && a == NoPos && !p.isEndOfURL(lhs) {
		url.Host = p.composite(ctx, false)
		if p.tok == COLON {
			//colon3 = p.pos
			p.step() // ':'
			if p.tok != SPACE && p.tok != LINEND {
				url.Port = p.composite(ctx, false)
			}
		}
	}
	if p.tok == PCON {
		url.Path = p.path(lhs, makePathPun(ctx, p.tok))
	}
	// scanning '#' as HASH instead of COMMENT
	defer p.scanner.SetBits(p.scanner.CommentsOff())
	if p.tok == QUE {
		p.step() // '?'
		if p.tok != HASH && !p.isEndOfURL(lhs) {
			url.Query = p.composite(ctx, false)
		}
	}
	if p.tok == HASH {
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

	const allowClosureName = true

	var (
		ctx = p.ctx()
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

	resolveObject := func(lPos Position, lTok Token, name Value) (str string, obj Value, okay bool) {
		if a, y := name.(*argumented); y { name = a.Value }
		if sel, y := name.(*selection); y {
			if sel == nil {
				erro(at(ctx,name.Position()), "nil selection: %v", name).debug(1)
			} else if v := sel.value(ctx, ident); v == nil {
				erro(of(ctx,name), "`%v` not selected nil value", sel).debug(1)
			} else if u, y := v.(unexpanded); y {
				if u.Value != sel { obj, okay = u.Value.(unresolved) } else if false {
					warn(at(ctx,sel.position), "unexpanded: %T %v %v %T %v", sel.o, sel.o, sel.t, sel.s, sel.s).debug(1)
				}
				if !okay { obj, okay = unresolved{u.Value, proj}, true }
			} else if s, y := v.(selected); !y {
				erro(of(ctx,name), "`%v` not selected: %v (%T)", sel, v, v).debug(1)
			} else if obj, okay = s.Value.(Object); !okay {
				// return // just use the selected value
			}

			switch lTok {
			case LPAREN:
				if _, y := obj.(invoker); !y { v := sel.value(ctx, plain)
					erro(of(ctx,name), "selected object '%v' is not invoker: %T %v ; %T %v ; %T %v", name, obj, obj, sel.o, sel.o, v, v).debug(16)
				}
			case LBRACE:
				if _, y := obj.(executer); !y {
					erro(of(ctx,name), "selected object '%v' is not executer: %T %v", name, obj, obj).debug(1)
				}
			}
			return
		}

		if val := name.expand(ctx, p.facet); val != name {
			if u, y := val.(unexpanded); y {
				return str, unresolved{u.Value, proj}, true
			} else { name = val }
		}

		switch lTok {
		case LPAREN:
			if allowClosureName && name.expandable(ctx, expandDelegate|expandClosure) {
				return str, unresolved{name, proj}, true // recursive delegation or closure
			} else if str, resolved = loader.resolveObject(name); false {
				erro(at(ctx,name.Position()), "resolve '%v' (%s) failed", name, str).debug(1)
				return
			} else if str == "" {
				erro(at(ctx,name.Position()), "name '%v' is empty", name).debug(1)
				return
			} else if isNone(resolved) {
				erro(at(ctx,lPos), "%v is none: %T", name, resolved).debug(16)
				return
			} else if isNull(resolved) {
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
				if tok != CLOSURE && p.autop != nil {
					var d = &auto{knownobject{
						objbase{valbase{name.Position()}, scope, scope.project},
						str,
					}}
					p.autos = append([]*auto{d}, p.autos...)
					obj, okay = d, true
					return
				}
				if obj = resolveConfig(name, str); !isNull(obj) {
					okay = true
					return
				} else if tok.IsClosure() || name.expandable(ctx, expandClosure|expandDelegate) ||
					refdef(ctx, name, _DefAny) {
					obj, okay = unresolved{name, proj}, true // recursive delegation or closure
					return
				} else if p.bits&parseUndefValue != 0 {
					obj, okay = unresolved{undef{name}, proj}, true
					return
				}

				erro(of(ctx,name), "%v: %v %v ⇒ '%s', is nil", proj, typeof(name), name, str)
				errostack(of(ctx,name), 32).debug(128)
				if ctx.dia().flush()>0 { /* fail(ctx.Position(), "undefined %v", name) */ }
			// } else if obj, okay = resolved.(*selection); okay {
			// 	return
			// } else if obj, okay = resolved.(*builtin); okay {
			// 	return
			// } else if obj, okay = resolved.(*self); okay {
			// 	return
			// } else if obj, okay = resolved.(*projectname); okay {
			// 	return
			// } else if obj, okay = resolved.(*scopename); okay {
			// 	return
			} else if _, okay = resolved.(invoker); okay {
				return str, resolved, okay
			} else if obj, okay = resolved.(Object); !okay {
				erro(at(ctx,lPos), "%v is not object: %T", name, resolved).debug(16)
				return
			} else {
				return
			}
		case LBRACE:
			if allowClosureName && name.expandable(ctx, expandDelegate|expandClosure) {
				erro(of(ctx,name), "%v: name '%v' (%T) is closured", proj, name, name).debug(1)
				return
			} else if resolved = loader.resolveEntries(name); isNull(resolved) {
				if name.expandable(ctx, plain) {
					var s = name.Strval(ctx)
					erro(of(ctx,name), "resolved '%v' (aka. %s) is nil (project=%v)", name, s, proj).debug(1)
				} else {
					erro(of(ctx,name), "resolved '%v' is nil (project=%v)", name, proj).debug(1)
				}
			} else if obj, okay = resolved.(Object); !okay {
				erro(at(ctx,lPos), "resolved '%v' of '%T' is not Object", name, resolved).debug(1)
			} else if isNull(obj) {
				erro(at(ctx,lPos), "%v is nil: %T", name, resolved).debug(16)
			} else if isNone(obj) {
				erro(at(ctx,lPos), "%v is none: %T", name, resolved).debug(16)
			} else if exe, _ := obj.(executer); exe == nil {
				erro(at(ctx,lPos), "resolved '%v' of '%T' is not executer", name, resolved).debug(1)
			}
		}
		return
	}

	var (
		name Value
		nameStr string
		tokLp Token
		opts []Value
		obj Value
		okay bool
	)
	switch p.step(); p.tok {
	case LPAREN, LBRACE: // $(...), ${...}
		var posLp = p.Position()
		tokLp = p.tok ; p.step() // skips LPAREN, LBRACE

		var posName = p.Position()
		switch p.tok {
		case SPACE:
			erro(at(ctx,posName), "unexpected spaces").debug(1)
			return MakeNull(posName)
		case COLON:
			p.step();  posName = p.Position()
			warn(at(ctx,posName), "colon").debug(1)
		}

		if name = p.expr(ctx, false); name == nil {
			erro(at(ctx,posName), "%v: parsed name is nil", proj).debug(1)
			return MakeNull(posName)
		}
		if false { if name.String() == "-g!$_" {
			noted(ctx, "%T %v %v", name, name, name.Strval(ctx)).debug(5)
		}}

		if v, y := optionalize(ctx, name); y { name = v } // foo?  foo(a,b,c)?
		if a, y := name.(*argumented); y {
			var args = merge(a.args...)
			for _, v := range args {
				if p, y := v.(*pair);  y { v = p.Key }
				if _, y := v.(*flag); !y {
					erro(of(ctx,v), "%v: not a Flag: %T %v", proj, v, v).debug(1)
				}
			}

			if true { name, opts = a.Value, args }
			if v, y := optionalize(ctx, name); y { name = v } // foo?(a,b,c)
		}

		if isNull(name) {/* error */} else
		if !allowClosureName && name.expandable(ctx, expandClosure|expandDelegate) {
			erro(at(ctx,posName), "%v: name '%v' (%T) is closured", proj, name, name).debug(1)
		} else if nameStr, obj, okay = resolveObject(posLp, tokLp, name); !okay {
			erro(at(ctx,posName), "%v: name '%v' is unidentified", proj, name).debug(1)
		}

		if  (tokLp == LPAREN && p.tok != RPAREN) ||
			(tokLp == LBRACE && p.tok != RBRACE) {
			var autos []*auto
			var savedAutos = p.autos
			var savedAutop = p.autop
			if nameStr == "auto" {
				if tokLp != LPAREN { erro(at(ctx,posLp), "%v: auto: incorrect left paren", proj).debug(1) }
				p.spaces() // skip the imediate spaces
				var al = p.list(ctx, false)
				if rest = append(rest, al); p.tok == COMMA { p.next(true) }
				for _, val := range merge(al) {
					var pos = val.Position()
					var s string
					if kv, y := val.(*pair); y {
						s = kv.Key.Strval(ctx)
						val = kv.Value
					} else {
						s = val.Strval(ctx)
						val = nil
					}
					if s == "" { erro(at(ctx,pos), "%v: auto: %v is empty", proj, val).debug(1) }
					var a = &auto{knownobject{objbase{valbase{pos}, scope, scope.project}, s}}
					a.position = posName ; if false { a.set(ctx, val) }
					autos = append(autos, a)
				}
				if tok != CLOSURE { p.autop = &posName /* NOTE: to enable auto-delegation */}
			} else if nameStr == "foreach" {
				var a = &auto{knownobject{objbase{valbase{posName}, scope, scope.project}, "_"}}
				a.position = posName
				autos = append(autos, a)
			}

			if autos != nil { p.autos = append(autos, p.autos...) }
			if savedBits := p.bits; nameStr == "case" {
				rest = append(rest, p.list(ctx, false))
				p.bits |= parseUndefValue
				for ; p.tok == COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx, false))
				}
				p.bits = savedBits
			} else if nameStr == "and" {
				p.bits |= parseUndefValue
				for rest = append(rest, p.list(ctx, false)); p.tok == COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx, false))
				}
				p.bits = savedBits
			} else if nameStr == "or" {
				p.bits |= parseUndefValue
				for rest = append(rest, p.list(ctx, false)); p.tok == COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx, false))
				}
				p.bits = savedBits
			} else {
				for rest = append(rest, p.list(ctx, false)); p.tok == COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx, false))
				}
			}
			p.autos = savedAutos
			p.autop = savedAutop
		}

		switch tokLp {
		case LPAREN: p.expect(RPAREN)
		case LBRACE: p.expect(RBRACE)
		}

	default:
		if position := p.Position(); tok != CLOSURE { // $(...), disabled $name.
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v` or `%v` or quotes", LPAREN, LBRACE).debug(1)
			return MakeNull(position)
		} else if p.tok == STRING || p.tok == COMPOUND {
			var posLp = p.Position()
			tokLp = p.tok

			// &'xxxx' or &"xxxx"
			if name = p.expr(ctx, false); isNull(name) {
				erro(at(ctx,posLp), "parsed name is nil").debug(1)
			} else if name.expandable(ctx, expandClosure) {
				erro(at(ctx,name.Position()), "name '%v' (%T) is closured (project=%v)", name, name, proj).debug(1)
			} else if nameStr, obj, okay = resolveObject(posLp, tokLp, name); !okay {
				erro(at(ctx,name.Position()), "name '%v' is unidentified", name).debug(1)
			}
		} else {
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v`, `%v` or quotes, not %v %v", LPAREN, LBRACE, p.tok, p.lit).debug(1)
			return MakeNull(position)
		}
	}

	if isNull(obj) && proj.plugin != nil && proj.pluginScope != nil {
		if nameStr == "" && !isNull(name) { nameStr = name.Strval(ctx) }
		if nameStr == "" {
			erro(at(ctx,name.Position()), "strval name '%v' is empty", name).debug(1)
		} else {
			obj = proj.pluginScope.Lookup(nameStr)
		}
	}

	if true && opts == nil && len(rest) > 0 {
		// NOTE: Options (flags) in args are deprecated by $(wildcard(-foo) ...)
		for _, v := range merge(rest[0]) {
			if p, y := v.(*pair); y { v = p.Key }
			if f, y := v.(*flag); y { if s := f.String(); s != "-foo" &&
				s != "-std" && s != "-lunwind" && s != "-x" && s != "-" &&
				!(s[0] == '-' && s[len(s)-1] == '-') { warn(of(ctx,v), "%v", f).debug(1) }
			} else { break }
		}
	}

	if position := ctx.Position(); tok.IsDelegate() {
		if isNull(obj) { erro(at(ctx,name.Position()), "resolved '%v' is nil (%T %v, tok=%v)", name, resolved, resolved, tok).debug(1) }
		return MakeDelegate(position, tokLp, obj, opts, rest...)
	} else {
		if isNull(obj) { erro(at(ctx,name.Position()), "resolved '%v' is nil (%T %v), shall be 'unresolved' (tok=%v)", name, resolved, resolved, tok).debug(1) }
		return MakeClosure(position, tokLp, obj, opts, rest...)
	}
}

func (p *parser) specialClosureDelegate(ctx Context, lhs bool) (result Value) {
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
		for _, a := range p.autos { if a.name == s { obj = a ; break } }
		if obj == nil {
			var a = &auto{knownobject{
				objbase{valbase{position}, scope, scope.project}, s,
			}}
			p.autos = append([]*auto{a}, p.autos...)
			obj = a
		}
	} else if w := MakeBareword(position, s); s == "_" {
		for _, a := range p.autos { if a.name == s { obj = a ; goto DashNxt } }
		if obj == nil && p.bits&parseTemplateBlock != 0 {
			if _, resolved = loader.resolveObject(w); resolved != nil {
				obj, _ = resolved.(Object)
			}
		}
	DashNxt:
	} else if _, resolved = loader.resolveObject(w); resolved == nil {
		erro(ctx, "'%v' is undefined (autos: %v)", s, p.autos).debug(16)
		return MakeNull(position)
	} else if c, y := resolved.(invoker); c == nil || !y {
		erro(of(ctx,resolved), "'%v' is not callable: %T", s, resolved).debug(6)
		return MakeNull(position)
	} else if obj, y = c.(Object); !y {
		erro(of(ctx,resolved), "'%v' is not object: %T", s, c).debug(6)
		return MakeNull(position)
	}

	if isNull(obj) {
		erro(ctx, "resolved '%v' is <nil>: %v (%T)", s, resolved, resolved).debug(1)
		return MakeNull(position)
	}

	var isDigital, isPlaceholder bool
	if tok.IsDelegate() {
		if result = MakeDelegate(position, tok, obj, nil); tok == DELEGATE__ {
			isPlaceholder = true
		} else if DELEGATE_0 <= tok && tok <= DELEGATE_9 {
			isDigital = true
		}
	} else {
		if result = MakeClosure(position, tok, obj, nil); tok == CLOSURE__ {
			isPlaceholder = true
		} else if CLOSURE_0 <= tok && tok <= CLOSURE_9 {
			isDigital = true
		}
	}
	if true {
		if isDigital { result = digital{ result }} else
		if isPlaceholder { result = placeholder{ result }}
	}
	return
}

func (p *parser) unary(ctx Context, lhs bool) (x Value) {
	if t_traverse.enabled && false { defer un(trace(t_traverse, "Unary")) }

	switch p.tok {
	case BAREWORD, AT:
		return p.bare(lhs)

	case BINARY, OCTAL, INTEGER, HEXADECIMAL, FLOATING,
		DATETIME, DATE, TIME, URI,
		/*RAW,*/ STRING, ESCAPE:
		return p.literal(lhs)

	case COMPOUND:
		return p.compound(lhs)

	case DELEGATE, CLOSURE: // delegate, closure
		return p.closuredelegate()

	case LPAREN:
		return p.group(lhs)

	case COMMA:
		if p.bits&parseCall == 0 {
			var tok, pos = p.tok, p.pos
			p.step()
			return &punctuation{valbase{p.loc(pos)}, tok}
		}

	case TILDE, DOT, DOTDOT: // ~ . ..
		var str = p.tok.String()
		tok, pos, end := p.tok, p.pos, p.pos+Pos(len(str))
		position := p.loc(pos)
		if p.step(); end != p.pos { // FIXME: ~user
			// '~', '.' or '..' used as bareword
			return &punctuation{valbase{position}, tok}
		} else if p.tok == PCON { // check /
			return p.path(lhs, makePathPun(at(ctx, position), tok))
		} else if tok == DOT || tok == DOTDOT { // TODO: parse to Qualiword instead
			x = &punctuation{valbase{position}, tok}
			if p.bits&parseDOT == 0 { x = p.dot(lhs, x) }
			return
		} else if tok == TILDE { // TODO: ~user
			return makePathPun(at(ctx, position), tok)
		} else {
			erro(ctx, "unexpected path: %v", tok).debug(1)
			return MakeNull(position)
		}

	case PCON: // The root of the path
		return p.path(lhs, makePathPun(ctx, p.tok))

	case LBRACK:
		return p.modifiers(ctx)

	case STAR, DAST, QUE/*, LBRACK*/: // * ? [
		return p.globExpr(nil) // (ie. no prefix)

	case PERC: // %bar (ie. no prefix)
		return p.percExpr(lhs, nil)

	case LBRACE: // TODO: regexp: {^.*}   or REGEXP
		return p.regexp(ctx)

	case MINUS:
		return p.flagExpr(lhs)

	case EXC:
		return p.negExpr(ctx, lhs)

	case SEMICOLON, BAR, PLUS:
		return p.punctuation()

	default:
		if p.tok.IsClosure() || p.tok.IsDelegate() {
			return p.specialClosureDelegate(ctx, lhs)
		} else if p.tok.IsKeyword() { // keywords here are barewords
			return p.bare(lhs)
		}
	}

	var s = p.scanner.ScanState
	if p.lineComment != nil {
		for _, comment := range p.lineComment.List {
			erro(at(p,comment.Pos), "# %s", comment.Text)
		}
	}
	erro(p, "bad unary expression '%v' (lit=%s, left=%v, bits=%022b, scan=%v)", p.tok, p.lit, lhs, p.bits, s).debug(32)

	p.step() // go to the next token
	return MakeNull(p.Position())
}

func (p *parser) isParametersGroup(x Value) (res bool) {
	if p.bits&parseDepend0 != 0 {
		if g, y := x.(*group); y && len(g.Elems) == 1 {
			_, res = g.Elems[0].(*group)
		}
	}
	return
}

func (p *parser) composite(ctx Context, lhs bool) (x Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Composed")) }

	switch x = p.unary(ctx, lhs); p.tok { // check composible expressions
	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2: // foo->bar  foo=>bar  foo~>bar
		// accepts 'foo=>bar', but 'foo => bar' is different
		if p.bits&parseNoSelect == 0 { x = p.selectExpr(x); break }
	case LBRACK: // xxx[(foo ...)]
		if p.isParametersGroup(x) { break }
		if p.bits&parseModifier == 0 {
			// FIXME: compose lhs x
			if m := p.modifiers(ctx); false {
				erro(of(ctx,m), "composing modifiers is ignored (unimplemented yet)")
			} else {
				errostack(of(ctx,m), 3, "composing modifiers is ignored (%T %v)", x, x).debug(12)
			}
		}
	case STAR, DAST, QUE/*, LBRACK*/: // foo*bar foo?bar foo[a-z]bar
		if p.bits&parseNoGlob == 0 { x = p.globExpr(x) }
	case PERC: // foo%bar
		// FIXME: %/foo/bar -> Path(% foo bar)
		if p.bits&parseNoPerc == 0 { x = p.percExpr(lhs, x) }
	case DOT: // foo.bar.baz.o
		// FIXME: push bits when parsing $(...)
		if p.bits&parseDOT == 0 { x = p.dot(lhs, x) } // TODO: parse to Qualiword
	case PCON: // ie. subdir/in/somewhere
		if p.bits&parseNoPath == 0 {
			switch x.(type) { // Path expressions, except '-I/path/to/include'
			case *flag: // By pass expressions like -I/foo/bar.
			default: x = p.path(lhs, x)
			}
		}
	case COLON:
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
	for p.tok != EOF {
		if p.tok == SPACE { p.next(true) } else {
			res = append(res, p.expr(ctx, false))
			if ctx.dia().flush() > 0 {
				warn(ctx, "parse text got %d errors", ctx.dia().totalErrors()).debug(16)
				if options.failOnErrors { fail(p.Position(), "fail by %d errors", ctx.dia().totalErrors()) }
			}
		}
	}
	return
}

func (p *parser) expr(ctx Context, lhs bool) (x Value) {
	if false && t_traverse.enabled { defer un(trace(t_traverse, "Expression")) }

	if false { defer assured(ctx) }

	var tok, lit = p.tok, p.lit
	if x = p.composite(ctx, lhs); x == nil {
		erro(p, "invalid (tok=%v,%v; next=%v,%v)", tok, lit, p.tok, p.lit).debug(6)
		return
	} else if lhs && p.tok.IsAssign() { return
	} else if p.isParametersGroup(x)  { return }

SwitchCompose:
	switch p.tok {
	case ASSIGN: // Example: '*.o = obj'
		if !lhs && p.bits&parseNoPair == 0 { x = p.pair(x) }
		return

	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2:
		if p.bits&parseNoSelect == 0 {
			x = p.selectExpr(x)
			goto SwitchCompose // For example: foobar⇒run(-gen)
		}
		return

	case LPAREN:
		if p.bits&parseNoArg == 0 { if x = p.argumentedExpr(x); x != nil {
			goto SwitchCompose
		}}
		return

	case PCON:
		if p.bits&parseNoPath == 0 {
			// Path expressions, except '-I/path/to/include'
			switch x.(type) {
			case *flag: // By pass expressions like -I/foo/bar.
			default: x = p.path(lhs, x)
			}
		}
		return // FIXES: a%%b/foo/bar -> Path(a%%b foo bar)

	case BAR:
		if _, ok := x.(*group); ok { return } // in case of: [(var)|...]

	case COMMA:
		if p.bits&(parseArged|parseCall|parseGroup) != 0 { return }
		if p.bits&parseDefineClause == 0 {
			warn(p, "%016b: %T %v ; %v %v", p.bits, x, x, p.tok, p.lit).debug(1)
			return
		}

	case
		COMPOSED, COLON, SEMICOLON, RAW,
		RPAREN, RBRACK, RBRACE, SPACE,
		LINEND, EOF:
		return // No composition!
	}

	var y = p.composite(ctx, lhs)
	if _, ok := y.(*Path); ok {
		switch x.(type) {
		case *flag: // okay: -Ifoo/bar, -Lfoo/bar
		case *Path: // okay: combine two paths
		case *String, *Compound, *delegate, *closure, *punctuation:
		case *barecomp:
		default:
			warn(of(ctx,y), "barecomp path: %T %v ; %v (next=%v)", x, x, y, p.tok).debug(1)
		}
	}

	// Further composing
	x = compose(ctx, x, y)

	// Keep trying composing as long as possible
	switch p.tok {
	case SPACE, LINEND, EOF: break
	default: //case SELECT_PROG1, SELECT_PROG2, LPAREN:
		goto SwitchCompose
	}
	return
}

// ----------------------------------------------------------------------------
// Clauses & Declarations

type clauseOpts struct {
	generalOpts
	conds []Value `if,cond,where`

    keyword Token // e.g. use, files, eval, etc.
    skip bool // e.g. -cond(false{}), -if(no{})

    values, remainder []Value // all values (unparsed) and remainder
	spec []Value
}

type parseSpecFunc func(Context, *CommentGroup, *clauseOpts, int)

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
	var ctx = p.ctx()
    var useList []Value // TODO: apply useList
    for _, prop := range props {
        var s string
        switch t := prop.(type) {
        case *flag:
            switch s = t.name.Strval(ctx); s {
            //case "nouse", "unuse": opts.unuse = true
            case "reuse": opts.reuse = true
            default: params = append(params, prop)
            }
        case *pair: // -param=value
            switch tt := t.Key.(type) {
            case *flag:
                switch s = tt.name.Strval(ctx); s {
                case "use": useList = append(useList, t.Value)
                default: params = append(params, prop)
                }
            default:
                erro(of(ctx,t.Key), "parameter `%v' unsupported `%T`", prop, prop)
                return
            }
        case *argumented: // -param(value)
            switch tt := t.Value.(type) {
            case *flag:
                switch s = tt.name.Strval(ctx); s {
                case "use": useList = append(useList, t.args...)
                default: params = append(params, prop)
                }
            default:
                erro(of(ctx,t.Value), "parameter `%v' unsupported `%T`", prop, prop)
                return
            }
        default:
            erro(of(ctx,prop), "parameter `%v` unsupported `%T`", prop, prop)
            return
        }
    }
    return
}

func (p *parser) use(ctx Context, doc *CommentGroup, g *clauseOpts, _ int) {
	if p.imports = append(p.imports, &usespec{ g.spec }); g.skip {
		// TODO: maybe give some information
		return
	}

	ctx = at(ctx, g.spec[0].Position()) // p.ctx()

	var specVals, arged []Value
	switch v := g.spec[0].(type) {
	case *delegate:
        for _, val := range xmerge(ctx, plain, v) {
            if !isTrivial(val) { specVals = append(specVals, val) }
		}
    case *pair:
        var s string
        if f, ok := v.Key.(*flag); !ok {
            erro(ctx, "'%v' invalid use spec", v.Key)
            return
        } else if s = f.name.Strval(ctx); s != "list" {
            erro(ctx, "'%v' invalid use spec, do you mean -list?", v.Key)
            return
        }

        for _, val := range xmerge(ctx, plain, v.Value) {
            if !isTrivial(val) { specVals = append(specVals, val) }
        }
	case *argumented:
        for _, val := range xmerge(ctx, plain, v.Value) {
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
	var args = parseOpts(ctx, &opts, 0, append(g.remainder, g.spec[1:]...)...)
	for _, a := range args {
		if _, ok := a.(*flag); ok || true {
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
			var dc = diaContext{ Context: ctx } // redefine ctx
			wg.Add(1); go func() {
				if false { defer assured(&dc, true) }
				defer func() {
					if len(dc.points) > 0 { dc.inner().dia().nest(dc.points) }
					wg.Done()
				} ()
				loader.use(ctx, opts, specVal, arged, args...)
			} ()
		}
	}
	wg.Wait()

	if errs := ctx.dia().flush(); errs > 0 {
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

func (p *parser) include(ctx Context, doc *CommentGroup, g *clauseOpts, _ int) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Spec")) }

	var opts = includeOpts{ clauseOpts: g }
	if vals := parseOpts(ctx, &opts, 0, g.remainder...); len(vals) > 0 {
		// TODO: deal with the unparsed generic options
		warn(ctx, "unknown opts: %v", vals).debug(1)
	}

	if len(g.spec) < 1 {
		erro(ctx, "expecting include file: %v", g.spec).debug(1)
		return
	}

	var x = g.spec[0]//.expand(ctx, strval|expandPlaceholder)
	var loader = ctx.loader()
	if p.spaces(); p.tok == COLON {
		switch x.(type) {
		case *File, *String, *Compound: // escape from file searching
		default: if file := loader.project.file(ctx, x.Strval(ctx)); file != nil {
			x = file
		} else if val := x.expand(ctx, strval); !isNull(val) && val != x {
			x = val
		}}

		x = p.rule(specialRuleNor, nil, []Value{x}) // this should return a Rule
	}
	if !g.skip { loader.include(ctx, opts, x) }
}

func (p *parser) files(ctx Context, doc *CommentGroup, g *clauseOpts, _ int) {
	defer p.setbits(p.setbit(parseFilesSpec))

	if len(g.spec) != 1 {
		erro(ctx, "too many files properties: %v", g.spec).debug(1)
		return
	}

	var path Value
	if p.tok == SELECT_PROG1 {
		p.next(true) // step forward with spaces skipped
		if p.tok == LINEND || p.lineComment != nil {
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

	ctx = p.ctx()

	var (
		val = g.spec[0]
		opts = cacher{generalOpts:g.generalOpts}
		pats []Value
	)
	parseOpts(ctx, &opts, 0, g.remainder...)

	if g, ok := val.(*group); ok {
		pats = g.Elems
	} else if val.expandable(ctx, expandClosure) {
		pats = []Value{ val }
	} else {
		pats = xmerge(ctx, plain, val)
	}

	if path == nil {
		if len(pats) == 1 { if a, ok := pats[0].(*argumented); ok { if f, ok := a.Value.(*flag); ok {
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
			if pat.expandable(ctx, expandClosure) {
				patsNew = append(patsNew, pat)
			} else {
				patsNew = append(patsNew, xmerge(ctx, plain, pat)...)
			}
		}

		var paths []Value
		if g, ok := path.(*group); ok {
			paths = g.Elems
		} else {
			paths = []Value{ path }
		}

		if len(patsNew) == 1 { if f, ok := patsNew[0].(*flag); ok {
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

func (p *parser) evalConfiguration(ctx Context, g *clauseOpts, props []Value) {
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
	} else if _, ts := entry.execute(at(ctx, entry.Position())); len(ts) > 0 {
		// FIXME: the entry might be a configure operation (see configure/.base/do.smart)
		for _, brk := range ts {
			if brk.what == traveFail {
				erro(of(ctx,entry), "execute '%v' failed: %v", entry, brk).debug(1)
			}
		}
	}

	if ctx.dia().flush()>0 { return }
	if project.configured {
		prompt(ctx, "configuration: %v already configured\n", project)
		return
	}

	var (
		okay bool
		cp *Project
		ce = configureExecutor{Context:ctx}
	)
	defer ce.close()

	for _, dep := range xmerge(ctx, strval, props/* [1:] */...) {
		if re, y := dep.(*rule); !y {
			erro(ctx, "unsupported prerequisite: %T %v", dep, dep).debug(1)
		} else if _, ts := re.execute(ctx); len(ts) > 0 {
			for _, brk := range ts { if brk.what == traveFail {
				erro(of(ctx,re), "execute '%v' failed: %v", re, brk).debug(1)
			}}
		}
	}

	if ctx.dia().flush()>0 { return }

	for _, entry := range project.configs {
		if cp, okay = ce.execute(cp, entry); !okay {
			erro(ctx, "configure '%v' failed", entry).debug(1)
			break
		}
	}

	project.configured = true // relaxes universeContext.configure
}

func (p *parser) assert(ctx Context, doc *CommentGroup, g *clauseOpts, _ int) {
	if !g.skip { call(ctx, "assert", plain, g.remainder, g.spec...) }
}

func (p *parser) append(ctx Context, doc *CommentGroup, g *clauseOpts, _ int) {
	if !g.skip { call(ctx, "append", plain, g.remainder, g.spec...) }
}

func (p *parser) eval(ctx Context, doc *CommentGroup, g *clauseOpts, _ int) {
	var (
		prop0, resolved, res Value
		name string
	)

	if g.skip { return } else if g.spec == nil {
		var opts struct {
			configuration bool `configuration`
			optimize Value `o,opt,optimize`
		}
		for _, op := range parseOpts(ctx, &opts, plain, g.values...) {
			var val Value
			if v, y := op.(*pair); y { op, val = v.Key, v.Value }
			if v, y := op.(*flag); y {
				switch t := val != nil && val.True(ctx); v.name.Strval(ctx) {
				case "dd": p.dd = t
				case "ddd":
					if u := ctx.universe(); val == nil { u.ddd = "yes" } else {
						u.ddd = val.Strval(ctx)
					}
				}
			} else {
				erro(of(ctx,op), "unsupport flag: %T %v (%v)", v, v, val).debug(1)
			}
		}

		// NOTE: see also universeContext.configure()
		if opts.configuration { p.evalConfiguration(ctx, g, g.spec) }
		return
	} else if prop0 = g.spec[0]; isTrivial(prop0) {
		erro(ctx, "illegal").debug(1)
		return
	}

	var opts []Value
	if a, y := prop0.(*argumented); y { prop0, opts = a.Value, a.args }

	ctx = at(ctx, prop0.Position())

	var loader = ctx.loader()
	if name, resolved = loader.resolveObject(prop0); false {
		erro(ctx, "resolve '%v' failed", prop0).debug(1)
		return
	} else if name == "configuration" {
		erro(ctx, "use '-configuration' instead (%v)", prop0).debug(1)
		return
	} else if x, y := resolved.(invoker); y {
		if b, y := x.(*builtin); y && !b.isCommand() {
			erro(ctx, "resolved builtin '%v' is not a command", prop0).debug(1)
			return
		}
		res = x.invoke(ctx, plain, opts, g.spec[1:])
	} else {
		erro(ctx, "resolved '%v' is %s (%v)", prop0, typeof(resolved), *g).debug(1)
		return
	}

	if ctx.dia().flush(); isTrivial(res) { return }

	/* TODO: if c, y := res.(code); y { ... } */
}

func (p *parser) directiveSpec(ctx Context) (props []Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Spec")) }

	//var doc = p.leadComment
	var comment *CommentGroup

ParamsParseLoop: // Parse the directive parameters
	for p.tok != EOF {
		switch p.spaces(); p.tok {
		case COMMA, LINEND, RPAREN, RBRACE,
			SELECT_PROG1, COLON: break ParamsParseLoop
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

func (p *parser) spec(ctx Context, keyword Token, pos Pos, f parseSpecFunc) {
	if t_traverse.enabled { defer un(trace(t_traverse, "spec("+keyword.String()+")")) }

	var opts = clauseOpts{ keyword: keyword }
	for p.spaces(); p.tok == MINUS; p.spaces() {
		opts.values = append(opts.values, p.expr(ctx, false))
	}
	opts.remainder = parseOpts(ctx, &opts, expandZero, opts.values...)

	for _, cond := range opts.conds {
		if t := cond.True(at(ctx, cond.Position())); !t {
			opts.skip = true
			break
		}
	}

	if p.spaces(); p.tok == LINEND {
		if keyword == EVAL { f(p.ctx(), nil, &opts, 0) } else {
			erro(p.ctx(), "%v: nil specs", keyword).debug(1)
		}
		return
	} else if p.tok == LPAREN {
		p.next(true)
		for iota := 0; p.tok != RPAREN && p.tok != EOF && (p.stop == 0 || p.pos < p.stop); iota++ {
			// TODO: collect documentation comments
			for p.tok == SPACE || p.tok == LINEND { p.next(true) }
			if p.tok == RPAREN || p.tok == EOF { break  }
			if opts.spec = p.directiveSpec(ctx); true {
				f(ctx, p.leadComment, &opts, iota)
			}
			if p.tok == COMMA || p.tok == LINEND { p.next(true) }
		}
		p.expect(RPAREN)
		if p.spaces(); p.tok != EOF { p.linend() }
		return
	}

	if p.tok != LINEND && p.tok != EOF && (p.stop == 0 || p.pos < p.stop) {
		if opts.spec = p.directiveSpec(ctx); true { f(ctx, nil, &opts, 0) }
		if p.tok == COMMA { p.next(true) }
	}
	if p.tok != EOF && (p.stop == 0 || p.pos < p.stop) {
		if p.spaces(); p.lineComment == nil { p.linend() }
	}
}

func (p *parser) define(ctx Context, tok Token, ident Value) (def *def) {
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
		savedFacet = p.facet
		// TODO: doc = p.leadComment
		// TODO: comment = p.lineComment
		position = p.loc(p.expect(tok))
		elems []Value
		value Value
	)
	switch tok {
	case CO1_ASSIGN: p.facet |= expandDelegate
	case CO2_ASSIGN: p.facet |= expandDelegate|expandClosure
	case SM1_ASSIGN:
	case SM2_ASSIGN:
	default: if false { warn(ctx, "todo: decide expand facet: %v", tok).debug(1) }
	}

	p.bits |= parseDefineClause
	elems = p.right(ctx)
	p.autos = savedAutos
	p.bits = savedBits
	p.facet = savedFacet

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
		p.scanner.pop(isCompoundLine)
		p.next(true) // skip RECIPE or SEMICOLON and parse in list mode
		position = p.Position()
		if isList = true; !p.isEndOfLine() {
			var (
				isValue = p.dialect == "value"
				x = p.expr(ctx, /*!isValue*/false) // parse first expr of recipe
				a *argumented
			)
			if !isNull(x) { if a, _ = x.(*argumented); a != nil { x = a.Value } }
			if isNull(x) {
				erro(ctx, "parsed value is nil")
			} else if isValue {
				// no resolving commands
			} else if t, y := x.(*bareword); !y {
				// does nothing
			} else if _, sym := loader.resolveObject(t); false {
				erro(ctx, "resolve '%v' failed", x)
			} else if isTrivial(sym) {
				erro(of(ctx,x), "resolved '%v' (from %v) is nil", t.string, x)
			} else if false {
				erro(of(ctx,x), "builtin command no more supported, use $(%s ...) instead", t.string)
			} else if b, y := sym.(*builtin); !y {
				erro(of(ctx,x), "'%s' is not a command (%s)", t.string, typeof(sym))
			} else if !b.isCommand() {
				erro(of(ctx,x), "'%s' is not a command, use $(%s ...) instead", t.string, t.string)
			} else { x = sym }

			if !isValue && p.tok.IsAssign() {
				elems = append(elems, p.define(ctx, p.tok, x))
				break SwitchDialect
			} else if a != nil {
				elems, a.Value = append(elems, a), x
			} else {
				elems = append(elems, x)
			}

			var cmdargs []Value

			p.setbit(parseRecipeBuiltin)
			for p.tok != EOF && p.tok != SEMICOLON && p.tok != LINEND && p.lineComment == nil {
				if p.spaces(); p.lineComment != nil { break }
				if !p.tok.IsRuleDelim() { x = p.expr(ctx, false) } else
				if false { x = p.rule(specialRuleRec, nil, elems) } else {
					erro(ctx, "unsupported token: %s, %v", p.tok, elems).debug(1)
				}
				if cmdargs = append(cmdargs, x); p.tok == COMMA {
					p.next(true)
					elems = append(elems, MakeList(p.Position(), cmdargs...))
					cmdargs = []Value{}
				}
				if p.lineComment != nil { break }
			}
			p.clearbit(parseRecipeBuiltin)
			elems = append(elems, MakeList(p.Position(), cmdargs...))
		}

	default:
		p.scanner.push(isCompoundLine) // NOTE: scanner does not set isCompoundLine correctly, fixit here
		p.next(true) // skip RECIPE or SEMICOLON and parse in line-string mode
		position = p.Position()
		p.setbit(parseRecipeText)
		for !p.isEndOfLine() {
			var x Value
			if p.tok == RAW {
				x = p.literal(false)
			} else {
				x = p.expr(ctx, false)
			}
			elems = append(elems, x)
		}
		p.clearbit(parseRecipeText)
		p.scanner.pop(isCompoundLine)
	}
	if p.spaces(); p.tok != EOF { p.linend() }
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
	for _, elem := range args {
		var kv, ok = elem.(*pair)
		if !ok || kv == nil {
			erro(of(ctx,elem), "bad var form (%T)", elem).debug(1)
			continue
		}

		var name string
		var k, v = kv.Key, kv.Value
		if name = k.Strval(at(ctx, k.Position())); name == "" {
			erro(of(ctx,k), "name '%v' is empty", k).debug(1)
		}

		if def, alt := loader.def(elem.Position(), name); alt != nil {
			erro(of(ctx,k), "'%v' already defined: %T", name, alt).debug(1)
		} else if def == nil {
			erro(of(ctx,k), "'%v' not defined", name).debug(1)
		} else {
			if g, y := v.(*group); y { v = g.ToList(def.position) }
			def.val(at(ctx, v.Position()), v)
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
			var d, a = loader.auto(of(ctx,elem), s)
			if a != nil { var y bool
				if d, y = a.(*auto); !y {
					erro(of(ctx,elem), "%T '%s' already taken the name, no such parameter", a, s)
				}
			}
			p.params = append(p.params, d)
			ctx.Scope().replace(ctx, strconv.Itoa(len(p.params)), d)
		default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
			erro(of(ctx,elem), "bad parameter form (%T)", elem)
		}
	}
	return
}

func (p *parser) modifiers(ctx Context) *modification {
	if t_traverse.enabled { defer un(trace(t_traverse, "Modifiers")) }

	var (
		posLp = p.loc(p.expect(LBRACK))
		hasParameters bool // ((foo bar))
		elems []*modifier
	)

	defer func(a parseBits) { p.bits = a }(p.bits)
	p.bits |= parseModifier

ForModifiersExpr:
	for p.tok != RBRACK && p.tok != EOF {
		if p.spaces(); p.tok == RBRACK { goto rBrack }

		var (
			x = p.expr(ctx, false)
			g *group
			name string
		)
		if y := false; ctx.dia().flush() > 0 {
			erro(at(ctx,x.Position()), "modifier: %T %v", x, x).debug(1)
			return nil
		} else if g, y = x.(*group); !y {
			var xv = x.expand(ctx, expandDelegate/*TODO: expandInline or expandAuto*/)
			erro(at(ctx,x.Position()), "modifier: %T %v   →   %T %v", x, x, xv, xv).debug(1)
			continue ForModifiersExpr
		}
		if l, y := g.Elems[0].(*List); y {
			g.Elems = append([]Value{ l.Elems[0] }, append(l.Elems[1:], g.Elems[1:]...)...)
		}

		switch n := g.Elems[0].(type) {
		case *bareword:
			if name = n.string; name == "var" {
				p.movar(ctx, merge(g.Elems[1:]...))
				continue ForModifiersExpr
			} else if name == "configure" {
				p.defineConfigureTargets(ctx)
				p.configure = true // set configure flag and define configure variables
			}
			goto checkNameAndAdd
		case *group: // parameters: ((foo bar))
			hasParameters = true
			if p.ruleParams(ctx, n.Elems); true {
				warn(ctx, "move parameters into depend list: %v", n).debug(1)
			}
			continue ForModifiersExpr
		case *delegate, *closure, *barecomp, *String:
			var ctx = at(ctx, n.Position())
			var v = xmerge(ctx, plain, n)
			if name = v[0].Strval(ctx); name == "" {
				erro(of(ctx,n), "name '%v' is empty", n).debug(1)
				continue ForModifiersExpr
			}
			goto checkNameAndAdd
		default:
			erro(of(ctx,n), "unsupported dialect or modifier (%T): %v", g.Elems[0], g.Elems[0]).debug(1)
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
		if len(g.Elems) == 0 {
			erro(of(ctx,x), "empty modifier: %v", x).debug(1)
		} else {
			elems = append(elems, &modifier{ *g })
		}
	}
	p.spaces()
	rBrack: p.expect(RBRACK)
	if len(elems) == 0 && !hasParameters {
		erro(at(ctx,posLp), "empty modifier group").debug(1)
	}
	if p.tok == COLON {
		erro(at(ctx,posLp), "unexpected colon after modifer").debug(1)
	}
    return &modification{ valbase: valbase{posLp}, list: elems }
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
	var ctx = p.ctx()
	if ctx.Project().keyword == PACKAGE {
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
		position = ctx.Position()
	)
	switch special {
	case specialRuleUse: scopeComment = fmt.Sprintf(usecomment)
	default:             scopeComment = fmt.Sprintf("rule %v", targets)
	}

	var loader = ctx.loader()
	defer loader.closeScope(loader.openScope(scopeComment))
	p.params = nil
	p.dialect = ""

	for _, s := range automatics {
		if a, alt := loader.auto(ctx, s); alt != nil {
			erro(ctx, "name `%s' already taken, not automatic (%T).", s, alt)
		} else if a == nil {
			erro(ctx, "'%s' is not defined", s)
		}
	}
	for i := 1; i < 10; i += 1 {
		if a, alt := loader.auto(ctx, strconv.Itoa(i)); alt != nil {
			erro(ctx, "name `%v` already taken, not numberred (%T).", i, alt)
		} else if a == nil {
			erro(ctx, "'$%d' is not defined", i)
		}
	}

	// switch special {
	// case specialRuleUse:
	// 	if name, alt := ctx.Scope().projectname(ctx, selfproj, ctx.Project()); alt != nil {
	// 		erro(ctx, "name `%s` already taken, not automatic (%T)", selfproj, alt)
	// 	} else if name == nil {
	// 		erro(ctx, "cannot define `%s` automatic", selfproj)
	// 	}
	// 	if name, alt := ctx.Scope().projectname(ctx, userproj, nil); alt != nil {
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

	if p.tok != SEMICOLON && p.tok != BAR && !p.isEndOfLine() {
		depends = p.depends(ctx, true)
	}
	if p.tok == BAR { // '|' starts the ordered prerequisites
		if p.next(true); p.tok != SEMICOLON && !p.isEndOfLine() {
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
		targets:  targets, //barefilize(ctx, targets...),
		depends:  depends, //barefilize(ctx, depends...),
		ordered:  ordered, //barefilize(ctx, ordered...),
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
			result = MakeNull(parsedData.position)
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

	p.expect(COLON) // expect and skip ':'

	if p.tok != BAREWORD {
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
			var pos = p.expect(BAREWORD) // USE
			var bits = p.setbit(parseSpecialRule)
			var ctx = p.ctx()
			// Options are *flag or *pair of a Flag.
			for p.tok == MINUS {
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
	p.pos, p.tok, p.lit, p.scanner.ScanState = t.pos, t.tok, t.lit, t.state

	// TODO: deal with expandParams

	// NOTE: comment here will affect loader.def()
	if false { pprofCounter += 1
		var name = fmt.Sprintf("template-%05d.prof", pprofCounter)
		defer startCPUProfile(ctx, name, true)()
	}

	var loader = ctx.loader()
	defer loader.closeScope(loader.openScope("template block"))

	ac := autoContext{ Context:at(ctx, p.Position()), defs:make(autoDefMap) }
	ctx = &ac

	for s, v := range vars { if a, alt := loader.auto(ctx, s); alt != nil {
		erro(ctx, "name '%s' already taken (%v)", s, typeof(alt)).debug(1)
	} else {
		a.set(ctx, v)
	}}

	var savedBits = p.bits
	p.bits |= parseTemplateBlock
	for p.tok != EOF && p.pos < p.stop {
		if p.tok == LINEND || (p.tok == COMMENT && p.lineComment != nil) {
			p.next(true)
		} else {
			p.clause(ctx)
		}
	}
	p.bits = savedBits
}

func (p *parser) templateExpand(ctx Context, t *template, params []Value) {
	var count int64
	defer func(t time.Time, pos Pos, tok Token, lit string, state ScanState) {
		if ctx.universe().ddd == "template.expand" {/* dont check time in ddd mode */} else
        if d := time.Now().Sub(t); d > time.Duration(options.slow)*time.Millisecond {
			var c = time.Duration(count)
            warnstack(ctx, 3, "slow: %v, %d * %v, prof-%d", d, count, d/c, pprofCounter).debug(1)
        }
		p.pos, p.tok, p.lit, p.scanner.ScanState = pos, tok, lit, state
	} (time.Now(), p.pos, p.tok, p.lit, p.scanner.ScanState)

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
		for _, elem := range xmerge(ctx, plain, t.params...) {
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
			if pair, ok := a.(*pair); !ok {
				erro(of(ctx,a), "unexpected value: %T %v", a, a).debug(1)
				return
			} else if s = pair.Key.Strval(at(ctx, pair.Key.Position())); s == "" {
				erro(of(ctx,a), "empty key: %T %v", pair.Key, pair.Key).debug(1)
				return
			} else if g, ok := pair.Value.(*group); ok {
				pos = pair.Value.Position()
				elems = g.Elems
			} else {
				pos = pair.Value.Position()
				elems = append(elems, pair.Value)
			}

			var m = vars[s]
			m.elems = xmerge(at(ctx, pos), plain, elems...)
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
	defer func(t time.Time, pos Pos, tok Token, lit string, state ScanState) {
        if d := time.Now().Sub(t); d > 1999*time.Millisecond {
			var c = time.Duration(count)
            infostack(ctx, 3, "%v: slow: %v, %v, %d*%v", name, d, count, d/c).debug(1)
        }
		p.pos, p.tok, p.lit, p.scanner.ScanState = pos, tok, lit, state
	} (time.Now(), p.pos, p.tok, p.lit, p.scanner.ScanState)

	p.pos, p.tok, p.lit, p.scanner.ScanState = t.pos, t.tok, t.lit, t.state

	// NOTE: a new scope is required for template expansion
	var loader = ctx.loader()
	defer loader.closeScope(loader.openScope("template call "))

	var params = merge(t.params...)
	for i, param := range params {
		var s = param.Strval(ctx)
		if a, alt := loader.auto(of(p,param), s); alt != nil {
			erro(at(ctx,param.Position()), "duplicated parameter '%s'", s).debug(1)
		} else if i < len(args) {
			a.set(ctx, args[i])
		}
	}

	for p.tok != EOF && p.pos < t.endPos {
		if p.tok == LINEND ||
			(p.tok == COMMENT && p.lineComment != nil) {
			p.next(true)
		} else {
			p.clause(ctx)
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
	defer ctx.dia().flush()

	var (
		starting = p.Position()
		arged *argumented
		verb string
	)
	p.expect(TEMPLATE) // expect and skip 'template'
	p.spaces()

	var op = p.expr(ctx, false) ; p.spaces()
	if p.tok == EOF {
		erro(of(ctx,op), "unexpected end of file after %v", op).debug(1)
		return
	} else if w, ok := op.(*bareword); ok {
		verb = w.string
	} else if arged, ok = op.(*argumented); !ok {
		erro(of(ctx,op), "unknown template verb: %v", op).debug(1)
		return
	}

	switch verb {
	case "end", "expand":
		erro(of(ctx,op), "unexpected verb: %s", verb).debug(1)
		return
	case "": if arged != nil {
		p.expect(LINEND)
		p.templateCall(ctx, arged.Value, arged.args)
		return //true
	}}

	var params = xmerge(ctx, plain, p.values(ctx, false)...)
	// TODO: parse template options - parseOpts

	var tmpl = &template{ state:p.scanner.ScanState, pos:p.pos, tok:p.tok, lit:p.lit }
	if verb == "def" {
		if len(params) != 1 {
			erro(at(ctx,starting), "too many def params: %v", params)
			return
		} else if arged, ok := params[0].(*argumented); !ok {
			erro(at(ctx,starting), "too many def params: %v", params)
			return
		} else {
			tmpl.name, tmpl.params = arged.Value, arged.args
			p.templates = append(p.templates, tmpl)
		}
	} else {
		tmpl.verb, tmpl.params = verb, params
	}

	var nested int
	for p.tok != EOF {
		if p.tok == LINEND || p.lineComment != nil {
			if p.spaces(); p.tok == EOF { return }
		}
		if p.tok != TEMPLATE { p.step(); continue }
		if false { info(p, "%v: %v", p.tok, p.scanner.ScanState).debug(1) }

		var pos, stop = p.pos, p.stop
		if p.next(true); p.tok != BAREWORD && p.tok != FOREACH {
			erro(p, "%v: %v (nested=%v)", p.tok, p.lit, nested).debug(1)
			return
		}

		if p.lit == "def" || p.lit == "for" || p.lit == "foreach" {
			nested += 1
		} else if p.lit == "expand" && (verb == "for" || verb == "foreach") {
			if nested > 0 { nested -= 1 } else {
				p.next(true) // consumes the 'expand'
				params := p.values(ctx, false)
				p.expect(LINEND)
				p.stop = pos
				p.templateExpand(ctx, tmpl, params)
				p.stop = stop
				return //true
			}
		} else if p.lit == "end" && (verb == "def") {
			if nested > 0 { nested -= 1 } else {
				p.next(true) // consumes the 'end'
				p.expect(LINEND)
				state := p.scanner.ScanState
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

func (p *parser) clause(ctx Context) {
	if t_traverse.enabled { defer un(tracef(t_traverse, "clause(%v, %v)", p.tok, p.pos)) }

	var x Value
	defer func() {
		if options.debugParsing(ctx, "clause") {
			warn(ctx, "parser.clause: %s %v; %v %v", typeof(x), x, p.tok, p.lit).debug(6)
		}
		if ctx.dia().flush() > 0 {
			errostack(ctx, 5, "clause: %s(%v); %v %v", typeof(x), x, p.tok, p.lit).debug(64)
			fail(p.Position(), "parser.clause")
		}
	} ()

	var tok = p.tok // TODO: allow assigns like: `eval := xxx`
	if /* TODO: p.spaces(); !p.tok.IsAssign() && !p.tok.IsRuleDelim() */true {
		switch tok {
		case USE:
			erro(ctx, "`%v` unexpected here", p.tok).debug(10)
			return
		case INCLUDE:
			p.spec(ctx, tok, p.expect(tok), p.include)
			return
		case FILES:
			p.spec(ctx, tok, p.expect(tok), p.files)
			return
		case ASSERT:
			p.spec(ctx, tok, p.expect(tok), p.assert)
			return
		case APPEND:
			p.spec(ctx, tok, p.expect(tok), p.append)
		case EVAL:
			p.spec(ctx, tok, p.expect(tok), p.eval)
			return
		case COLON:
			p.specialRule()
			return
		case TEMPLATE:
			p.template(ctx)
			return
		case FOREACH:
			warn(ctx, "%v %v", p.tok, p.lit).debug(1)
			p.next(true)
			return
		case DONE:
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
		if options.debugParsing(ctx, "define") {
			warn(p, "parser.clause: %s(%v); %v %v", typeof(x), x, p.tok, p.lit).debug(1)
			ctx.dia().flush()
		}
		p.define(ctx, p.tok, x)
		return
	}

	var list = []Value{ x }
	if !p.tok.IsRuleDelim() {
		list = append(list, p.left(ctx)...)
	}

	if p.tok.IsRuleDelim() {
		if options.debugParsing(ctx, "rule") {
			warn(p, "parser.clause: %s(%v); %v %v", typeof(x), x, p.tok, p.lit).debug(1)
			ctx.dia().flush()
		}
		p.rule(specialRuleNor, nil, list)
		return
	}

	if p.tok != EOF { return }

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
	configureFlag Value `c,conf,configure` // detects dotConfigure if empty
	noDock bool `n,nd,nod,nodock,no-dock` // don't load container project
    traveUseLoop bool `b,break;l,loop` // don't recursively use this project
    multiUseAllowed bool `m,multi`  // this project is used multiple times
	final bool `f,final`
}

func (p *parser) file(ctx Context) *parsedFile {
	if options.traceLaunch { defer un(trace(t_launch, "parser.file")) }
	if t_traverse.enabled  { defer un(trace(t_traverse, "File '"+p.scanner.File().Name()+"'")) }
    if false { defer un(tracef(t_traverse, "file(%s)", p.scanner.File().Name())) }
	if ctx.dia().countErrors() > 0 {
		// Don't bother parsing the rest if errors scanned,
		// likely not a Go source file at all.
		errostack(ctx, 5, "got errors").debug(10)
		return nil
	}

	var (
		ident *barecomp
		identStr string
		implicitBase string // aka. foo.bar.Baz implicitly load base 'foo/bar'
		abs, rel, tmp string
		loader = ctx.loader()
		position = ctx.Position()
		keyword  = p.tok
		filename = p.scanner.File().Name()
		isMainFile = isEntryFileName(filename)
	)
	assert(loader != nil, "nil loader")
	assert(loader == loader, "bad loader")
	defer loader.closeScope(loader.openScope(fmt.Sprintf("file %s", filename)))
	if options.debugFileEntry {
		warn(p, "parser.file: %v %v", p.tok, p.scanner.ScanState).debug(1)
	}

	if loader.mode&Flat != 0 {
		abs = ctx.Project().absPath
	} else {
		abs = filepath.Dir(filename)
	}
	rel, _ = filepath.Rel(loader.WorkDir(), abs)
	tmp = joinTmpPath(ctx,loader.WorkDir(), rel)

	if s := ctx.Scope(); s != nil { var d *def
		//defer p.closeScope()
		if loader.mode&Flat == 0 {
			d, _ = loader.def(position, ".")
			d.set(ctx, DefVoid, pathStr(ctx, position, rel))

			d, _ = loader.def(position, "/")
			d.set(ctx, DefVoid, pathStr(ctx, position, abs))

			d, _ = loader.def(position, "CTD") // Current Temp Directory, TODO: make it $:ctd:
			d.set(ctx, DefVoid, pathStr(ctx, position, tmp))

			d, _ = loader.def(position, "CWD") // Current Work Directory, TODO: make it $:cwd:
			d.set(ctx, DefVoid, pathStr(ctx, position, abs))
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
	case PACKAGE, MODULE:
		erro(ctx, "deprecated keyword: %s", keyword).debug(1)
		return nil
	case CONFIGURE:
		switch p.next(true); p.tok {
		case DOT:
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
	case PROJECT:
		if loader.mode&Flat != 0 { erro(ctx, "forbidden `%v` in flat file", p.tok) }

		p.next(true)

		var ( // Options are *flag or *pair of a flag.
			opts projectDeclOpts
			optVals []Value
			pos Position
		)
		for p.tok == MINUS {
			var opt = p.expr(ctx, false);  p.spaces()
			optVals = append(optVals, opt)
			if !pos.IsValid() { pos = opt.Position() }
		}
		if !pos.IsValid() { pos = p.Position() }
		if a := parseOpts(ctx, &opts, 0, optVals...); len(a) > 0 {
			for _, v := range a { erro(of(ctx,v), "unknown option '%v'", v).debug(1) }
			return nil
		}

		var g = ctx.Globe()
		var linfo = g.loads[len(g.loads)-1]

		// Smart-lang spec:
		//   * the project clause is not a declaration;
		//   * the project name does not appear in any scope.
		if p.tok == LPAREN || p.tok == EOF || p.tok == LINEND || p.lineComment != nil {
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
		} else if p.tok == TILDE {
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
			for p.tok != EOF && p.tok != SPACE {
				if w := p.bare(false); w == nil {
					erro(at(ctx,ident.Position()), "expecting a bareword").debug(1)
				} else if word, ok := w.(*bareword); !ok {
					erro(at(ctx,ident.Position()), "expecting a bareword: %v (%T)", w, w).debug(1)
				} else if ident.comp(ctx, word); p.tok == DOT {
					ident.comp(ctx, &punctuation{valbase{p.Position()}, p.tok}) // TODO: parse to Qualiword
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
		if n := loader.dia().countErrors(); n > 0 {
			erro(p, "got %d errors parsing file: %s", filename).debug(1)
			return nil
		}

		var _, declared = linfo.declares[identStr]
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
			defer loader.closeCurrent(ident, identStr)
			isMainFile = isMainFile && !declared;
		}

		var basePos Position
		if implicitBase != "" { basePos = pos } else { basePos = p.Position() }
		if p.tok == LPAREN {
			var bits = p.setbit(parseGroup)
			for p.tok != EOF {
				for p.next(true); !p.isEndOfList(false); {
					p.spaces()
					param := p.expr(ctx, false)
					p.spaces()

					//if p.lineComment != nil  { break }
					//if p.tok == LINEND { break }
					if p.tok == EOF {
						erro(at(ctx,basePos), "unexpected end of file while parsing bases").debug(1)
						p.setbits(bits) ; return nil
					}

					var (
						ctx = at(ctx, param.Position())
						t = parseOpts(ctx, &opts, 0, param)
					)
					if keyword == PACKAGE || opts.final {
						// No bases for PACKAGE or final project
					} else if !loader.bases(ctx, linfo, "", merge(t...)...) {
						errostack(of(ctx,param), 3, "loading base '%v' failed", t).debug(10)
						p.setbits(bits) ; return nil
					}
				}
				if p.tok != COMMA { break }
			}
			p.setbits(bits)
			p.expect(RPAREN)
			if false { defer func() { warn(ctx, "%v", ident).debug(32) } () }
		} else if !loader.bases(ctx, linfo, implicitBase) { // for special bases, e.g. .base
			erro(at(ctx,basePos), "loading bases failed").debug(1)
			return nil
		}

		if p.spaces(); p.tok != EOF { p.linend() }
		if keyword != PACKAGE {
			loader.loadConfiguration(ctx, linfo, ident, identStr, declared)
			if !opts.noDock { loader.loadProjectContainer(ctx, ident, identStr) }
		}
	case EOF:
		return nil
	default:
		if loader.mode&Flat == 0 {
			p.expected(p.pos, "configure, project, module or package keyword")
		}
	}

	var u = ctx.universe()
	if options.debugFiles != nil && u.ddd == "" { for _, s := range options.debugFiles {
		if strings.Contains(filename, s) { u.ddd = "parser.files" ; break }
	}}

	var auto = (loader.mode&Flat == 0) && isMainFile //&& isEntryFileName(filename)
	if auto { loader.after(p.ctx(), "declare") }
	if loader.mode&ModuleClauseOnly == 0 {
		if loader.mode&Flat == 0 {
			ForInit: for p.tok != EOF {
				switch tok := p.tok; tok {
				case LINEND: p.next(true) // skip empty lines
				case USE:
					p.spec(ctx, tok, p.expect(tok), p.use)
				case ASSERT:
					p.spec(ctx, tok, p.expect(tok), p.assert)
				case APPEND:
					p.spec(ctx, tok, p.expect(tok), p.append)
				case EVAL:
					p.spec(ctx, tok, p.expect(tok), p.eval)
				case TEMPLATE:
					p.template(ctx)
				case FOREACH:
					warn(ctx, "%v %v", p.tok, p.lit).debug(1)
					p.next(true)
				case DONE:
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
		if false && auto { loader.after(p.ctx(), "amid") }
		if loader.mode&ImportsOnly == 0 { // rest of module body
			for /* p.dia().totalErrors() == 0 && */ p.tok != EOF {
				if p.tok == LINEND || (p.tok == COMMENT && p.lineComment != nil) {
					p.next(true)
				} else if p.clause(p.ctx()); ctx.dia().flush() > 0 {
					break
				}
			}
		}
	}
	if auto { loader.after(ctx, "appendix") }
	if options.debugFiles != nil && u.ddd == "parser.files" { u.ddd = "" }

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
