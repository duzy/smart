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
	parseModifier      // 0000000000000001000000
	parseBARE          // 0000000000000010000000
	parseGLOB          // 0000000000000010000000
	parsePATH          // 0000000000000100000000
	parsePERC          // 0000000000001000000000
	parseREXP          // 0000000000010000000000
	parseSELECT_PROP   // 0000000000100000000000
	parseURL           // 0000000001000000000000

	parseCompound      // 0000000010000000000000
	parseDefineClause  // 0000000100000000000000

	parseFilesSpec     // 0000001000000000000000  files ( ... )
	parseCodeBlock     // 0000010000000000000000
	parseUndefValue    // 0000100000000000000000
	parseForeachTempl  // 0001000000000000000000

	parseSpecialRule   // 0010000000000000000000  e.g. :use ...:

	parseRecipeBuiltin // 0100000000000000000000  recipe builtin command
	parseRecipeText    // 1000000000000000000000
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

const maxDigitAutoNum = 9

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
	state scanState
	end *scanState
	pos, endPos Pos   // token position
	tok Token // one token look-ahead
	lit string      // token literal
	verb string
	name Value // if only 'def', TODO: considering []Value for nested template defs?
	params []Value
}

type parse_left_hand_side struct { bool }

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
		var t = warn(p, "%v %v %v", p.tok, p.lit, p.scanner.scanState)
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
func (p *parser) ctx(ctx Context) Context { return &positionContext{ctx, p.Position()} }

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
			msg += ", found " + p.tok.String()
			if p.tok.IsLiteral() {
				msg += " '" + p.lit + "'"
			}
		}
	}
	erro(at(p,p.loc(pos)), msg).debug(32)
}

func (p *parser) expect(tok Token) Pos {
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

func (p *parser) bare(ctx Context, lhs bool) (x Value) {
	if true { defer dtrace(ctx, "parser.bare") }

	ctx = p.ctx(ctx)

	defer p.setbits(p.setbit(parseBARE))

	var tok, lit = p.tok, p.lit
	switch p.step(); tok {
	case BAREWORD: // okay
	case BARE:
		if p.tok == LBRACE { // bare{ ... }
			if p.next(true); p.tok != RBRACE {
				x = p.bare(p.ctx(ctx), false)
			}
			p.spaces()
			p.expect(RBRACE)
			return
		}
	case GLOB:
		if p.tok == LBRACE { // glob{ ... }
			if p.next(true); p.tok != RBRACE {
				x = p.glob(p.ctx(ctx), nil)
			}
			p.spaces()
			p.expect(RBRACE)
			return
		}
	case REGEX:
		if p.tok == LBRACE { // regex{...}
			if true {
				x = p.regexp(p.ctx(ctx))
			} else {
				if p.next(true); p.tok != RBRACE {
					x = p.regexp(p.ctx(ctx))
				}
				p.spaces()
				p.expect(RBRACE)
			}
			return
		}
	case FILE:
		if p.tok == LBRACE { // file{ ... }
			if p.next(true); p.tok != RBRACE {
				if v := p.expr(ctx); v != nil {
					var c = of(ctx, v)
					var s = v.string(c)
					var a = []interface{}{stat_nonexist{true}}
					if !isAbsOrRel(s) { a = append(a, stat_dir{ctx.Project().absPath}) }
					x = stat(c, s, a...)
				}
			}
			p.spaces()
			p.expect(RBRACE)
			return
		}
	case PATH:
		if p.tok == LBRACE { // path{ ... }
			if p.next(true); p.tok != RBRACE {
				x = p.path(ctx, false, p.expr(ctx))
			}
			p.spaces()
			p.expect(RBRACE)
			return
		}
	case BIN, OCT, INT, HEX, FLOAT:
		if p.tok == LBRACE { // bin{...}, oct{...}, int{...}, hex{...}, float{...}
			if p.next(true); p.tok == RBRACE {
				switch p.step(); tok {
				case BIN:   x = makeBin(p.Position(), 0)
				case OCT:   x = makeOct(p.Position(), 0)
				case INT:   x = makeInt(p.Position(), 0)
				case HEX:   x = makeHex(p.Position(), 0)
				case FLOAT: x = makeFloat(p.Position(), 0.)
				}
			} else if v := p.expr(ctx); v == nil {
				// TODO: true{ expr }, yes{ expr }, ...
				erro(ctx, "%s expects: %v, not %v %v", tok, RBRACE, p.tok, p.lit).debug(1)
			} else if p.spaces(); p.tok == RBRACE {
				if p.step(); tok == FLOAT {
					var n, _ = v.float(ctx)
					return makeFloat(p.Position(), n)
				}
				switch n, _ := v.int(ctx); tok {
				case BIN: return makeBin(p.Position(), n)
				case OCT: return makeOct(p.Position(), n)
				case INT: return makeInt(p.Position(), n)
				case HEX: return makeHex(p.Position(), n)
				}
			}
			return
		}
	case ANSWER:
		if p.tok == LBRACE { // answer{...}
			var v bool
			var pos = p.Position()
			if p.next(true); p.tok != RBRACE {
				switch p.tok {
				case  TRUE, YES: v = true  ; p.next(true)
				case FALSE,  NO: v = false ; p.next(true)
				default:
					if t := p.expr(ctx); t != nil {
						v = t.true(ctx)
					} else {
						erro(p.ctx(ctx), "invalid expression").debug(1)
					}
				}
			}
			p.spaces()
			p.expect(RBRACE)
			return &answer{boolean{valbase{pos},v}}
		}
	case BOOL, BOOLEAN:
		if p.tok == LBRACE { // bool{...}, boolean{...}
			var v bool
			var pos = p.Position()
			if p.next(true); p.tok != RBRACE {
				switch p.tok {
				case  TRUE, YES: v = true  ; p.next(true)
				case FALSE,  NO: v = false ; p.next(true)
				default:
					if t := p.expr(ctx); t != nil {
						v = t.true(ctx)
					} else {
						erro(p.ctx(ctx), "invalid expression").debug(1)
					}
				}
			}
			p.spaces()
			p.expect(RBRACE)
			return &boolean{valbase{pos},v}
		}
	case TRUE, FALSE:
		if p.tok == LBRACE { // true{...}, false{...}
			var v bool
			var pos = p.Position()
			if p.next(true); p.tok == RBRACE {
				v = tok == TRUE
			} else {
				v = tok == TRUE && p.expr(ctx).true(ctx)
			}
			p.spaces()
			p.expect(RBRACE)
			return &boolean{valbase{pos},v}
		}
	case YES, NO:
		if p.tok == LBRACE { // yes{...}, no{...}
			var v bool
			var pos = p.Position()
			if p.next(true); p.tok == RBRACE {
				v = tok == YES
			} else {
				v = tok == YES && p.expr(ctx).true(ctx)
			}
			p.spaces()
			p.expect(RBRACE)
			return &answer{boolean{valbase{pos},v}}
		}
	case NULL:
		if p.tok == LBRACE { // null{}
			var pos = p.Position()
			if p.next(true); p.tok != RBRACE {
				erro(ctx, "unexpected expression (%v '%s')", p.tok, p.lit).debug(1)
			}
			p.expect(RBRACE)
			return &null{valbase{pos}}
		}
	case NONE:
		if p.tok == LBRACE { // none{...}
			var x Value
			var pos = p.Position()
			for p.next(true); p.tok != RBRACE && p.tok != EOF; p.spaces() {
				if v := p.expr(ctx); x == nil {
					x = v
				} else if l, y := x.(*list); y {
					l.Elems = append(l.Elems, v)
				} else {
					x = &list{elements{[]Value{x,v}}}
				}
			}
			p.expect(RBRACE)
			return &none{valbase{pos},x}
		}
	case UNDEF:
		if p.tok == LBRACE { // undef{...}
			var x Value
			var pos = p.Position()
			if p.next(true); p.tok != RBRACE {
				x = p.expr(ctx)
			} else {
				x = &null{valbase{pos}}
			}
			p.spaces()
			p.expect(RBRACE)
			return &undef{x}
		}
	case AT, DOT, DOTDOT: // TODO: parse DOT into Qualiword
		return &punctuation{valbase{p.Position()}, tok} // lit = tok.String() // Special bareword.
	default:
		if tok.IsKeyword() {
			lit = tok.String()
		} else if true {
			erro(ctx, "%v %v -> %v %v", tok, lit, p.tok, p.lit).debug(1)
			// panic(failure{"parsing: %v %v",ia(p.Position(), p.tok, p.lit)})
			return
		} else {
			p.expect(BAREWORD)
		}
	}
	return makeBareword(ctx.Position(), lit)
}

func (p *parser) selector(ctx Context) (res Value) {
	defer p.setbits(p.setbit(parseSELECT_PROP))
	res = p.expr(ctx)
	return
}

func (p *parser) selectExpr(ctx Context, lhs Value) (res Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Select")) }

	var (
		tok = p.tok // the arrow '->' or '=>'
		loader = ctx.loader()
		proj = loader.Project()
	)
	ctx = p.ctx(ctx)
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
        switch t.s {
        case "use", "usee", "goals", "os", "mode":
			erro(ctx, "$:%s: is obsoleted, use $(.$s) instead", t.s, t.s).debug(1)
        default:
            if name, o := loader.resolve(lhs); false {
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
				erro(at(ctx,lhs.Position()), "%v: '%v' is undefined (name=%v, obj=%v)", proj, lhs, name, o)
				erro(ctx, "%v: parser is here (name=%s, tok=%s)", proj, t.s, tok)
				erro(at(ctx,p.Position()), "%v: parser to go here (tok=%s, lit=%s)", proj, p.tok, p.lit).debug(16)
				return
            }
        }
    case *barecomp: // for cases like '.foo'
        if name, o := loader.resolve(t); false {
			erro(of(ctx,lhs), "resolve selection object '%v' (%s) error", lhs, name).debug(1)
			return
        } else if !isNull(o) {
			lhs = o
		} else if tok == SELECT_PROG2 {
			res = makeNull(ctx.Position()) // ignore
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

			var val = p.expr(ctx)
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
func (p *parser) values(ctx Context, ii ...interface{}) (values []Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Values")) }

	var lhs bool

	for _, i := range ii {
		switch v := i.(type) {
		case parse_left_hand_side: lhs = v.bool
		case Value: values = append(values, v)
		default: erro(ctx, "unsupported value: %v{%v}", typeof(i), i).debug(5)
		}
	}

	for p.spaces(); !p.isEndOfList(lhs); p.spaces() {
		var prev = p.pos
		if values = append(values, p.expr(ctx, lhs)); p.pos == prev {
			erro(p, "bad: %v %v; %v", p.tok, p.lit, values).debug(1)
			break
		}

		// If there's a comment right after the parsed expression, we break
		// the expression list to treat the end-of-line comment like a LINEND.
		if p.tok == EOF || p.tok == LINEND || p.lineComment != nil { break }
	}
	return
}

func (p *parser) list(ctx Context, ii ...interface{}) *list {
	return makeList(p.values(ctx, ii...)...)
}

func (p *parser) group(ctx Context, lhs bool) *group {
	if t_traverse.enabled { defer un(trace(t_traverse, "Group")) }

	defer p.setbits(p.setbit(parseGroup))
	p.clearbit(parseCall) // for commas

	ctx = p.ctx(ctx)

	p.expect(LPAREN)
	p.spaces()

	var elems, converted = p.values(ctx), false
	for p.tok != RPAREN && p.tok != EOF {
		// if p.tok == COMMA { warn(ctx, "%020b: %v %v", p.bits, p.tok, p.lit).debug(1) }
		// if p.tok == COMMA { p.next(true) }
		switch p.tok {
		case BAR, COMMA, SEMICOLON:
			elems = append(elems, p.punctuation())
			p.spaces()
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
	if t_traverse.enabled { defer un(trace(t_traverse, "argumented")) }

	defer p.setbits(p.setbit(parseGroup))
	p.clearbit(parseCall)

	ctx = p.ctx(ctx)
	p.next(true) // skip LPAREN

	var a = []Value{ p.list(ctx) }
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
		a = append(a, p.list(p.ctx(ctx)))
	}
	p.expect(RPAREN)
	return makeArgumented(x, a...)
}

func (p *parser) globMeta(ctx Context) (x *GlobMeta) {
	pos, tok := p.Position(), p.tok
	p.step()
	return makeGlobMeta(pos, tok)
}

func (p *parser) globRange(ctx Context) (x *GlobRange) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Glob")) }

	ctx = p.ctx(ctx)
	p.expect(LBRACK) // skip '['

	chars := p.expr(ctx)
	p.expect(RBRACK) // skip ']'

	return makeGlobRange(ctx.Position(), chars)
}

func (p *parser) glob(ctx Context, x Value) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "Glob")) }

	ctx = p.ctx(ctx)

	// avoid nesting glob expressions
	defer p.setbits(p.setbit(parseGLOB))

	var components []Value
	if x != nil { components = []Value{x} }

ForGlobTok:
	for p.tok != EOF && p.lineComment == nil {
		switch p.tok {
		case SELECT_PROP, SELECT_PROG1, SELECT_PROG2, PCON, RPAREN, COMMA, SPACE, LINEND, EOF:
			break ForGlobTok
		case STAR, DAST, QUE: // * ** ?
			x = p.globMeta(ctx)
		case LBRACK:
			// FIXME: '[...]' has been used for modifier expressions
			x = p.globRange(ctx)
		case RBRACE:
			if p.bits&parseBARE != 0 { break ForGlobTok }
			erro(p.ctx(ctx), "unexpected right-brace").debug(1)
		default:
			// FIXME: escaped glob metas/chars
			x = p.expr(ctx)
		}
		components = append(components, x)
	}
	if components == nil {
		erro(ctx, "nil glob expression (tok=%v, lit=%v)", p.tok, p.lit)
	}
	return makeGlobPattern(ctx, components...)
}

func (p *parser) perc(ctx Context, lhs bool, x Value) Value {
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
			perc2 := makePercPattern(position, nil, nil)
			if pos+2 == p.pos {
				switch p.tok {
				case PERC: // %%%
					erro(p, "too many %")
				case PCON: // FIXES: %%/xxx -> Path(%% xxx)
					x = makePercPattern(position, x, perc2)
					return p.path(ctx, lhs, x)
				case COLON,    DOLON,
					LPAREN,    RPAREN,
					LBRACK,    RBRACK,
					LBRACE,
					SEMICOLON, COMMA,
					SPACE,     LINEND:
				default:
					var (
						yy = p.expr(ctx)
						_, ok = yy.(*Path)
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
	return makePercPattern(p.loc(pos), x, y)
}

func (p *parser) regexp(ctx Context) (x Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Regexp")) }

	defer p.setbits(p.setbit(parseREXP)) // avoid nesting percent expressions

	var rx string
	var pos = p.Position()
	for p.expect(LBRACE); p.tok != RBRACE && p.tok != EOF; p.scan() {
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

	p.expect(RBRACE)

	var err error
	var exp *regexp.Regexp
	if exp, err = regexp.Compile(rx); err != nil {
		errostack(at(p,pos), 3, "regexp: %v", err).debug(6)
	}
	return &RegexpPattern{valbase{pos}, exp} // TODO: correct regexp pattern value
}

func (p *parser) pair(ctx Context, x Value) *pair {
	if t_traverse.enabled { defer un(trace(t_traverse, "pair")) }

	ctx = p.ctx(ctx)
	p.step()

	var y Value
	if p.isEndOfList(false) {
		y = makeNull(ctx.Position())
	} else {
		y = p.expr(ctx)
	}
	return makePair(ctx.Position(), x, y)
}

func (p *parser) flagExpr(ctx Context, lhs bool) flag {
	if t_traverse.enabled { defer un(trace(t_traverse, "flag")) }

	ctx = p.ctx(ctx)
	p.step() // skip dash '-'

	var x Value
	// flag expressions, excluding "-)" "-]" "-}" "-\n", "-=", "-:", etc.
	if p.isEndOfLine() || p.isEndOfList(false) || p.tok == SPACE || p.tok == RECIPE {
		x = makeNull(ctx.Position())
	} else if false {
		x = p.expr(ctx)
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
	return flag{x}
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

func (p *parser) escape(ctx Context, lhs bool) (v Value) {
	var pos, lit = p.Position(), p.lit
	p.expect(ESCAPE)
	return &escaped{valbase{pos}, lit}
}

func (p *parser) literal(ctx Context, lhs bool) (v Value) {
	var tok, lit = p.tok, p.lit
	ctx = p.ctx(ctx)
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
    case BAREWORD:    v = makeBareword(position, lit)
    case STRING:      v = makeStrlit(position, lit)
    case RAW:         v = makeRaw(position, lit)
    default: unreachable()
    }
	return
}

func (p *parser) compound(ctx Context, lhs bool) *compound {
	var elems []Value
	var lpos = p.pos
	p.step()

	defer p.setbits(p.setbit(parseCompound))

	for p.tok != EOF && p.tok != COMPOSED && p.tok != LINEND {
		if p.tok == RAW {
			elems = append(elems, p.literal(ctx, false))
		} else {
			elems = append(elems, p.expr(ctx))
		}
	}
	p.expect(COMPOSED)
	return makeCompound(p.loc(lpos), elems...)
}

// Parses dot composing expressions (TODO: check against file extensions).
//   .foo
//   .'foo'
//   ."foo"
//   .(foo)
//   ..foo
//   ..'foo'
//   .foo.bar
func (p *parser) dot(ctx Context, lhs bool, x Value) (res *barecomp) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Dot")) }

	defer p.setbits(p.setbit(parseDOT))

	if x == nil { panic(fmt.Sprintf("nil dot (tok=%v)", p.tok)) }

	ctx = p.ctx(ctx)

	var comp *barecomp
	if comp, _ = x.(*barecomp); comp == nil {
		comp = makeBarecomp(x.Position())//(p.Position())
		comp.Elems = append(comp.Elems, x)
	}

	for !p.isEndOfDotConcat(lhs) {
		comp.comp(ctx, p.composite(ctx, false))
		if p.tok == DOT /*&& comp.End() == p.pos*/ {
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

func pathpun(ctx Context, tok Token) *PathPun {
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
	return makePathPun(ctx.Position(), r)
}

func (p *parser) path(ctx Context, lhs bool, start Value) *Path {
	if t_traverse.enabled { defer un(trace(t_traverse, "Path")) }

	defer p.setbits(p.setbit(parsePATH))

	var (
		position = start.Position()
		path *Path
		ok bool
	)
	if ctx = at(ctx, position); start == nil {
		erro(ctx, "bad closure/delegate name").debug(1)
		p.step()
		return makePath(position) // empty path
	} else if path, ok = start.(*Path); !ok {
		path = makePath(position, start)
	}

BuildPath:
	for p.tok == PCON {
		var pos = p.Position() // skips repeated '/' sequence
		for p.step(); p.tok == PCON; p.step() { pos = p.Position() }
		switch p.tok {
		case RPAREN, LPAREN, RBRACE, LBRACE, COMMA, SPACE, LINEND:
			// Encountered the tailing '/', append 'zero' segment.
			path.Elems = append(path.Elems, makePathPun(pos, 0))
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
			res = makeNull(p.Position())
			return
		}
	} else if !p.isEndOfURL(lhs) {
		erro(at(ctx, p.loc(colon1)), "TODO: URL: %v (%T) (next: %s (%s))", scheme, scheme, p.tok, p.lit).debug(1)
		res = makeNull(p.Position())
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
		url.Path = p.path(ctx, lhs, pathpun(ctx, p.tok))
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

func (p *parser) expandTemplateBlockAuto(ctx Context, obj, val Value) (result Value) {
	if p.bits&parseCodeBlock == 0 { return val }
	if a, y := obj.(*auto); y && !(p.bits&parseForeachTempl != 0 && a.name_ == "_") {
		// Make clone to barecomp and Path for compose() and x.comp().
		switch t := val.expand(ctx, /* p.facet */strval); v := t.(type) {
		case  *barecomp: return makeBarecomp(v.Position(), v.Elems...)
		case      *Path: return makePath(v.Position(), v.Elems...)
		case unexpanded: /*return v.Value*/ break
		default: return v
		}
	}
	return val
}

func (p *parser) closuredelegate(ctx Context) (result Value) {
	if t_traverse.enabled {	defer un(trace(t_traverse, "ClosureDelegate")) }

	ctx = at(ctx, p.Position())

	const allowClosureName = true

	defer dtrace(ctx, "parser.closuredelegate")

	var (
		loader = ctx.loader()
		scope = loader.Scope()
		proj = loader.Project()
		tok = p.tok
		resolved Value // Object or *selection
		rest []Value
	)

	resolveConfig := func(val Value, name string) (obj Object) {
		if c := proj.configure; c != nil { obj = c.resolve(ctx, name) }
		return
	}

	resolve := func(lPos Position, lTok Token, name Value) (str string, obj Value, okay bool) {
		defer dtrace(ctx, "parser.closuredelegate.resolve") // backtrace on errors

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
			} else if str, resolved = loader.resolve(name); false {
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
					if okay = a.name(ctx) == str; okay {
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

				errostack(of(ctx,name), 6, "undefined '%v' (%v)", name, typeof(name)).debug(6) // ⇒
				return
			} else if _, okay = resolved.(invoker); okay {
				return str, resolved, okay
			} else if obj, okay = resolved.(Object); !okay {
				errostack(at(ctx,lPos), 6, "%v is not object: %T", name, resolved).debug(6)
				return
			} else {
				return
			}
		case LBRACE:
			if allowClosureName && name.expandable(ctx, expandDelegate|expandClosure) {
				erro(of(ctx,name), "%v: name '%v' (%T) is closured", proj, name, name).debug(1)
				return
			} else if resolved = loader.project.resolveEntries(ctx, name, false); isNull(resolved) {
				if name.expandable(ctx, plain) {
					var s = name.string(ctx)
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

	defer p.setbits(p.setbit(parseCall))

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
			return makeNull(posName)
		case COLON:
			p.step();  posName = p.Position()
			warn(at(ctx,posName), "colon").debug(1)
		}

		if name = p.expr(ctx); name == nil {
			erro(at(ctx,posName), "%v: parsed name is nil", proj).debug(1)
			return makeNull(posName)
		}

		if v, y := optionalize(ctx, name); y { name = v } // foo?  foo(a,b,c)?
		if a, y := name.(*argumented); y {
			var args = merge(a.args...)
			for _, v := range args {
				if p, y := v.(*pair);  y { v = p.Key }
				if _, y := v.(flag); !y {
					erro(of(ctx,v), "%v: not a Flag: %T %v", proj, v, v).debug(1)
				}
			}
			if true { name, opts = a.Value, args }
			if v, y := optionalize(ctx, name); y { name = v } // foo?(a,b,c)
		}

		if isNull(name) {
			// error
		} else if !allowClosureName && name.expandable(ctx, expandClosure|expandDelegate) {
			erro(at(ctx,posName), "%v: name '%v' (%T) is closured", proj, name, name).debug(1)
		} else if nameStr, obj, okay = resolve(posLp, tokLp, name); !okay {
			erro(at(ctx,posName), "%v: name '%v' is unidentified", proj, name).debug(1)
		}

		if  (tokLp == LPAREN && p.tok != RPAREN) ||
			(tokLp == LBRACE && p.tok != RBRACE) {
			var autos []*auto
			var savedAutos = p.autos
			var savedAutop = p.autop
			var savedBits  = p.bits
			if nameStr == "auto" {
				if tokLp != LPAREN { erro(at(ctx,posLp), "%v: auto: incorrect left paren", proj).debug(1) }
				p.spaces() // skip the imediate spaces
				var al = p.list(ctx)
				if rest = append(rest, al); p.tok == COMMA { p.next(true) }
				for _, val := range merge(al) {
					var pos = val.Position()
					var s string
					if kv, y := val.(*pair); y {
						s = kv.Key.string(ctx)
						val = kv.Value
					} else {
						s = val.string(ctx)
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

			if nameStr == "case" {
				rest = append(rest, p.list(ctx))
				for p.bits |= parseUndefValue; p.tok == COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx))
				}
			} else if nameStr == "and" {
				p.bits |= parseUndefValue
				for rest = append(rest, p.list(ctx)); p.tok == COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx))
				}
			} else if nameStr == "or" {
				p.bits |= parseUndefValue
				for rest = append(rest, p.list(ctx)); p.tok == COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx))
				}
			} else if nameStr == "foreach" {
				rest = append(rest, p.list(ctx))
				for p.bits |= parseForeachTempl; p.tok == COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx))
				}
			} else {
				if n, e := strconv.Atoi(nameStr); e == nil && n < 0 && n > maxDigitAutoNum {
					erro(of(ctx, name), "num auto too big: %v (max %v)", n, maxDigitAutoNum).debug(1)
				}
				for rest = append(rest, p.list(ctx)); p.tok == COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx))
				}
			}
			p.autos = savedAutos
			p.autop = savedAutop
			p.bits = savedBits
		}

		switch tokLp {
		case LPAREN: p.expect(RPAREN)
		case LBRACE: p.expect(RBRACE)
		}

	default:
		if position := p.Position(); tok != CLOSURE { // $(...), disabled $name.
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v` or `%v` or quotes", LPAREN, LBRACE).debug(1)
			return makeNull(position)
		} else if p.tok == STRING || p.tok == COMPOUND {
			var posLp = p.Position()
			tokLp = p.tok

			// &'xxxx' or &"xxxx"
			if name = p.expr(ctx); isNull(name) {
				erro(at(ctx,posLp), "parsed name is nil").debug(1)
			} else if name.expandable(ctx, expandClosure) {
				erro(at(ctx,name.Position()), "name '%v' (%T) is closured (project=%v)", name, name, proj).debug(1)
			} else if nameStr, obj, okay = resolve(posLp, tokLp, name); !okay {
				erro(at(ctx,name.Position()), "name '%v' is unidentified", name).debug(1)
			}
		} else {
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v`, `%v` or quotes, not %v %v", LPAREN, LBRACE, p.tok, p.lit).debug(1)
			return makeNull(position)
		}
	}

	if obj == nil && proj.plugin != nil && proj.pluginScope != nil {
		if nameStr == "" && !isNull(name) { nameStr = name.string(ctx) }
		if nameStr == "" {
			erro(at(ctx,name.Position()), "strval name '%v' is empty", name).debug(1)
		} else {
			obj = proj.pluginScope.Lookup(nameStr)
		}
	}

	if obj == nil {
		erro(at(ctx,name.Position()), "resolved '%v' is nil (%T %v, tok=%v)", name, resolved, resolved, tok).debug(1)
	}

	if pos := ctx.Position(); tok.IsDelegate() {
		var t = makeDelegate(pos, tokLp, obj, opts, rest...)
		result = p.expandTemplateBlockAuto(ctx, obj, t)
		if p.bits&parseCodeBlock != 0 && nameStr == "foreach" { if u, y := result.(unexpanded); y {
			noted(ctx, "%v %v", typeof(u.Value), u.Value).debug(1)
		}}
		return
	} else {
		return makeClosure(pos, tokLp, obj, opts, rest...)
	}
}

func (p *parser) specialClosureDelegate(ctx Context, lhs bool) (result Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "SpecialClosureDelegate")) }

	var pos, tok, s = p.pos, p.tok, p.lit
	var position = p.loc(pos)

	p.step()

	var obj Object
	var loader = ctx.loader()
	var scope = loader.Scope()

	for _, a := range p.autos { if a.name(ctx) == s { obj = a ; break } }
	for _, a := range p.autos { if a.name(ctx) == s { obj = a ; break } }

	var resolved Value
	if obj == nil { if c := s[0]; len(s) == 1 && (('0' <= c && c <= '9') /*|| c == '_'*/) {
		a := &auto{knownobject{objbase{valbase{position}, scope, scope.project}, s}}
		p.autos = append([]*auto{a}, p.autos...)
		obj = a
	} else if w := makeBareword(position, s); s == "_" {
		if p.bits&parseCodeBlock != 0 {
			if _, resolved = loader.resolve(w); resolved != nil {
				obj, _ = resolved.(Object)
			}
		}
	} else if _, resolved = loader.resolve(w); resolved == nil {
		erro(ctx, "'%v' is undefined (autos: %v)", s, p.autos).debug(16)
		return makeNull(position)
	} else if t, y := resolved.(invoker); !y {
		erro(of(ctx,resolved), "'%v' is not callable: %T", s, resolved).debug(6)
		return makeNull(position)
	} else if obj, y = t.(Object); !y {
		erro(of(ctx,resolved), "'%v' is not object: %T", s, c).debug(6)
		return makeNull(position)
	}}

	if obj == nil {
		erro(ctx, "'%v' is <nil> (resolved: %T %v)", s, resolved, resolved).debug(1)
		return makeNull(position)
	}

	if p.bits&parseCodeBlock != 0 && cast[*universe](ctx).ddd == "template" { defer func() {
		var v Value
		if a, y := obj.(*auto); y { v = autoVal(ctx, a.name_) }
		noted(ctx, "%T %v ; %T %v ; %v", obj, obj, result, result, v).debug(16)
	}()}

	if tok.IsDelegate() {
		if result = makeDelegate(position, tok, obj, nil); tok == DELEGATE__ {
			result = placeholder{result}
		} else if DELEGATE_0 <= tok && tok <= DELEGATE_9 {
			result = digital{result}
		}
		return p.expandTemplateBlockAuto(ctx, obj, result)
	} else {
		if result = makeClosure(position, tok, obj, nil); tok == CLOSURE__ {
			result = placeholder{result}
		} else if CLOSURE_0 <= tok && tok <= CLOSURE_9 {
			result = digital{result}
		}
	}
	return
}

func (p *parser) unary(ctx Context, lhs bool) (x Value) {
	if t_traverse.enabled && false { defer un(trace(t_traverse, "Unary")) }

	defer dtrace(ctx, "parser.unary")

	switch p.tok {
	case BAREWORD, AT:
		return p.bare(ctx, lhs)

	case BINARY, OCTAL, INTEGER, HEXADECIMAL, FLOATING, DATETIME, DATE, TIME, URI, STRING/*, RAW*/:
		return p.literal(ctx, lhs)

	case COMPOUND:
		return p.compound(ctx, lhs)

	case DELEGATE, CLOSURE: // delegate, closure
		return p.closuredelegate(ctx)

	case ESCAPE:
		return p.escape(ctx, lhs)

	case LPAREN:
		return p.group(ctx, lhs)

	case COMMA:
		if p.bits&parseCall == 0 {
			var tok, pos = p.tok, p.Position()
			p.step()
			return &punctuation{valbase{pos}, tok}
		}

	case LBRACE: // {
		if p.scan(); p.tok == RBRACE {
			x = &null{valbase{p.Position()}}
		} else {
			x = &list{elements{p.values(ctx)}}
		}
		p.expect(RBRACE)
		return

	case TILDE: // ~ ; TODO: ~user
		tok, ctx := p.tok, p.ctx(ctx)
		p.step()
		return pathpun(ctx, tok)

	case DOT, DOTDOT: // . ..
		var str = p.tok.String()
		tok, loc, end := p.tok, p.loc(p.pos), p.pos+Pos(len(str))
		if p.step(); end != p.pos { // FIXME: ~user
			return &punctuation{valbase{loc}, tok}
		} else if p.tok == PCON { // check /
			return p.path(ctx, lhs, pathpun(at(ctx, loc), tok))
		} else if tok == DOT || tok == DOTDOT { // TODO: parse to Qualiword instead
			x = &punctuation{valbase{loc}, tok}
			if p.bits&parseDOT == 0 { x = p.dot(ctx, lhs, x) }
			return
		} else {
			erro(ctx, "unexpected path: %v", tok).debug(1)
			return &null{valbase{loc}}
		}

	case PCON: // The root of the path
		return p.path(ctx, lhs, pathpun(ctx, p.tok))

	case LBRACK:
		return p.modification(ctx)

	case STAR, DAST, QUE/*, LBRACK*/: // * ? [
		return p.glob(ctx, nil) // (ie. no prefix)

	case PERC: // %bar (ie. no prefix)
		return p.perc(ctx, lhs, nil)

	case MINUS:
		return p.flagExpr(ctx, lhs)

	case EXC:
		return p.negExpr(ctx, lhs)

	case SEMICOLON, BAR, PLUS:
		return p.punctuation()

	default:
		if p.tok.IsClosure() || p.tok.IsDelegate() {
			return p.specialClosureDelegate(ctx, lhs)
		} else if p.tok.IsKeyword() { // keywords here are barewords
			return p.bare(ctx, lhs)
		}
	}

	if p.lineComment != nil { for _, comment := range p.lineComment.List {
		erro(at(p,comment.Pos), "# %s", comment.Text).debug(1)
	}}

	erro(p, "bad: %v (lit=%s, left=%v, bits=%022b, scan=%v)",
		p.tok, p.lit, lhs, p.bits, p.scanner.scanState).debug(1)

	p.step() // go to the next token
	return makeNull(p.Position())
}

func (p *parser) isParametersGroup(x Value) (res bool) {
	if p.bits&parseDepend0 != 0 { if g, y := x.(*group); y && len(g.Elems) == 1 {
		_, res = g.Elems[0].(*group)
	}}
	return
}

func (p *parser) composite(ctx Context, lhs bool) (x Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Composed")) }

	defer dtrace(ctx, "parser.composite")

	x = p.unary(ctx, lhs)

	switch p.tok { // check composible expressions
	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2: // foo->bar  foo=>bar  foo~>bar
		// accepts 'foo=>bar', but 'foo => bar' is different
		if p.bits&parseNoSelect == 0 { x = p.selectExpr(ctx, x); break }
	case LBRACK: // xxx[(foo ...)]
		if p.isParametersGroup(x) { break }
		if p.bits&parseModifier == 0 {
			// FIXME: compose lhs x
			if m := p.modification(ctx); false {
				erro(of(ctx,m), "composing modification is ignored (unimplemented yet)")
			} else {
				errostack(of(ctx,m), 3, "composing modification is ignored (%T %v)", x, x).debug(12)
			}
		}
	case STAR, DAST, QUE/*, LBRACK*/: // foo*bar foo?bar foo[a-z]bar
		if p.bits&parseNoGlob == 0 { x = p.glob(ctx, x) }
	case PERC: // foo%bar
		// FIXME: %/foo/bar -> Path(% foo bar)
		if p.bits&parseNoPerc == 0 { x = p.perc(ctx, lhs, x) }
	case DOT: // foo.bar.baz.o
		// FIXME: push bits when parsing $(...)
		if p.bits&parseDOT == 0 { x = p.dot(ctx, lhs, x) } // TODO: parse to Qualiword
	case PCON: // ie. subdir/in/somewhere
		if p.bits&parseNoPath == 0 {
			switch x.(type) { // Path expressions, except '-I/path/to/include'
			case flag: // By pass expressions like -I/foo/bar.
			default: x = p.path(ctx, lhs, x)
			}
		}
	case COLON:
		if (p.bits&parseRecipe != 0 || !lhs) && p.bits&parseNoURL == 0 {
			if isKnownURLScheme(x.string(at(ctx, p.Position()))) { x = p.url(ctx, lhs, x) }
		}
	}
	return
}

func (p *parser) text(ctx Context) (res []Value) {
	if false && t_traverse.enabled { defer un(trace(t_traverse, "Text")) }
	for p.tok != EOF { if p.tok == SPACE { p.next(true) } else {
		res = append(res, p.expr(ctx))
		if ctx.dia().flush() > 0 { total := ctx.dia().totalErrors()
			warn(ctx, "parse text got %d errors", total).debug(16)
			if cast[*universe](ctx).failOnErrors {
				panic(failure{"fail by %d errors",ia(p.Position(), total)})
			}
		}
	}}
	return
}

func (p *parser) expr(ctx Context, ab ...bool) (x Value) {
	if false && t_traverse.enabled { defer un(trace(t_traverse, "Expression")) }

	defer dtrace(ctx, "parser.expr")

	var tok, lit = p.tok, p.lit
	var lhs bool ; if len(ab)>0 { lhs = ab[0] }

	if x = p.composite(ctx, lhs); x == nil {
		erro(at(ctx, p.Position()), "invalid (tok=%v,%v; next=%v,%v)", tok, lit, p.tok, p.lit).debug(6)
		return
	}

	if lhs && p.tok.IsAssign() { return }
	if p.isParametersGroup(x)  { return }

	var n int

SwitchCompose:
	switch p.tok {
	case ASSIGN: // Example: '*.o = obj'
		if !lhs && p.bits&parseNoPair == 0 { x = p.pair(ctx, x) }
		return

	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2:
		// For example: foobar⇒run(-gen)
		if p.bits&parseNoSelect == 0 { x = p.selectExpr(ctx, x); goto SwitchCompose }
		return

	case LPAREN:
		if p.bits&parseNoArg == 0 { if x = p.argumentedExpr(ctx, x); x != nil {
			goto SwitchCompose
		}}
		return

	case PCON:
		if p.bits&parseNoPath == 0 {
			// Path expressions, except '-I/path/to/include'
			switch x.(type) {
			case flag: // By pass expressions like -I/foo/bar.
			default: x = p.path(ctx, lhs, x)
			}
		}
		return // FIXES: a%%b/foo/bar -> Path(a%%b foo bar)

	case BAR:
		if _, y := x.(*group); y { return } // in case of: [(var)|...]

	case COMMA:
		if p.bits&(parseArged|parseCall|parseGroup|parseModifier) != 0 { return }
		if p.bits&(parseDefineClause) == 0 {
			warn(p, "%v{%v} %v '%v' (%016b)", typeof(x), x, p.tok, p.lit, p.bits).debug(1)
			return
		}

	case COMPOSED, COLON, SEMICOLON, RAW, RPAREN, RBRACK, RBRACE, SPACE, LINEND, EOF:
		return // No composition!
	}

	var y = p.composite(ctx, lhs)

	if _, t := y.(*Path); t {
		switch x.(type) {
		case  flag: // okay: -Ifoo/bar, -Lfoo/bar
		case *Path: // okay: combine two paths
		case *barecomp:
		case *strlit, *compound, *delegate, *closure, *punctuation:
		default: warn(of(ctx,y), "barecomp path: %T %v ; %v (next=%v)", x, x, y, p.tok).debug(1)
		}
	}

	// Make the first a clone, because $(auto-xxx) 'points' to the same value
    if false && n == 0 { switch t := x.(type) {
    case *barecomp: x = makeBarecomp(t.Position(), t.Elems...)
    case *Path: x = makePath(t.Position(), t.Elems...)
    }}

	x, n = compose(ctx, x, y), n+1 // concat

	// Keep trying composing as long as possible
	switch p.tok {
	case SPACE, LINEND, EOF: return
	default: goto SwitchCompose
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

func (p *parser) _parseUseSpecProps(ctx Context, props []Value) (opts useOpts, params []Value, err error) {
	ctx = p.ctx(ctx)

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
            switch tt := t.Key.(type) {
            case flag:
                switch s = tt.Value.string(ctx); s {
                case "use": useList = append(useList, t.Value)
                default: params = append(params, prop)
                }
            default:
                erro(of(ctx,t.Key), "parameter `%v' unsupported `%T`", prop, prop)
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

	defer dtrace(ctx, "parser.use")

	var specVals, arged []Value
	switch v := g.spec[0].(type) {
	case *delegate:
        for _, val := range xmerge(ctx, plain, v) {
            if !isTrivial(val) { specVals = append(specVals, val) }
		}
    case *pair:
        var s string
        if f, ok := v.Key.(flag); !ok {
            erro(ctx, "'%v' invalid use spec", v.Key)
            return
        } else if s = f.Value.string(ctx); s != "list" {
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
		if _, ok := a.(flag); ok || true {
			erro(of(ctx,a), "unkown use opts: %T %v", a, a).debug(1)
			return
		}
	}

	var wg sync.WaitGroup
	var loader = ctx.loader()
	for _, specVal := range specVals {
		if ctx := at(ctx, specVal.Position()); true {
			loader.usespec(ctx, opts, specVal, arged, args...)
		} else {
			var dc = diaContext{ Context: ctx } // redefine ctx
			wg.Add(1); go func() {
				defer func() { if false { assured(&dc, true) }
					if len(dc.points) > 0 { dc.inner().dia().nest(dc.points) }
					wg.Done()
				} ()
				loader.usespec(ctx, opts, specVal, arged, args...)
			} ()
		}
	}

	wg.Wait()
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
	var l = ctx.loader()
	if p.spaces(); p.tok == COLON {
		switch x.(type) {
		case *File, *strlit, *compound: // escape from file searching
		default: if file := l.project.file(ctx, x.string(ctx)); file != nil {
			x = file
		} else if val := x.expand(ctx, strval); !isNull(val) && val != x {
			x = val
		}}

		x = p.rule(ctx, specialRuleNor, nil, []Value{x}) // this should return a Rule
	}

	if !g.skip { l.include(ctx, opts, x) }
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
		path = p.expr(ctx)
	}

	if p.spaces(); p.lineComment != nil {
		//spec.Comment = p.lineComment
	}
	if g.skip {
		// TODO: maybe give some information
		return
	}

	ctx = p.ctx(ctx)

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
		if len(pats) == 1 { if a, ok := pats[0].(*argumented); ok { if f, ok := a.Value.(flag); ok {
			var name = f.Value.string(ctx)
			switch name {
			default:
				// TODO: parse files options
				erro(of(ctx,f.Value), "invalid files flag: %v").debug(1)
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
			var paths = []Value{ makeStrlit(val.Position(), ctx.Project().absPath) }
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

		if len(patsNew) == 1 { if f, ok := patsNew[0].(flag); ok {
			var name = f.Value.string(ctx)
			switch name {
			default:
				// TODO: parse files options
				erro(of(ctx,f.Value), "invalid files flag: %v").debug(1)
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

	var ce = configureExecutor{Context:ctx} ; defer ce.close()

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

	/***/ promptEnteringDirectory(ctx, project.absPath)
	defer promptLeavingDirectory(ctx, project.absPath)

	for _, entry := range project.configs { ce.execute(entry) }

	project.configured = true // relaxes configure()
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

	defer dtrace(ctx, "parser.eval")

	if g.skip { return } else if g.spec == nil {
		var opts struct {
			configuration bool `configuration`
			optimize Value `o,opt,optimize`
		}
		for _, op := range parseOpts(ctx, &opts, plain, g.values...) {
			var val Value
			if v, y := op.(*pair); y { op, val = v.Key, v.Value }
			if v, y := op.(flag); y {
				switch t := val != nil && val.true(ctx); v.Value.string(ctx) {
				case "dd": p.dd = t
				case "ddd":
					if u := cast[*universe](ctx); val == nil {
						u.ddd = "yes"
					} else if t, y := boolVal(val); y {
						if t { u.ddd = "yes" } else { u.ddd = "" }
					} else {
						u.ddd = val.string(ctx)
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
	if name, resolved = loader.resolve(prop0); false {
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

	if isTrivial(res) { return }

	/* TODO: if c, y := res.(code); y { ... } */
}

func (p *parser) directive(ctx Context) (props []Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "spec")) }

	defer dtrace(ctx, "parser.directive")

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

		props = append(props, p.expr(ctx))
	}
	if comment != nil { /* TODO: directive documments */ }
	return
}

func (p *parser) spec(ctx Context, keyword Token, pos Pos, f parseSpecFunc) {
	if t_traverse.enabled { defer un(trace(t_traverse, "spec("+keyword.String()+")")) }

	defer dtrace(ctx, "parser.spec")

	var opts = clauseOpts{ keyword: keyword }
	for p.spaces(); p.tok == MINUS; p.spaces() {
		opts.values = append(opts.values, p.expr(ctx))
	}
	opts.remainder = parseOpts(ctx, &opts, expandZero, opts.values...)

	for _, cond := range opts.conds {
		if t := cond.true(at(ctx, cond.Position())); !t {
			opts.skip = true
			break
		}
	}

	if p.spaces(); p.tok == LINEND {
		if keyword == EVAL { f(p.ctx(ctx), nil, &opts, 0) } else {
			erro(p.ctx(ctx), "%v: nil specs", keyword).debug(1)
		}
		return
	} else if p.tok == LPAREN {
		p.next(true)
		for iota := 0; p.tok != RPAREN && p.tok != EOF && (p.stop == 0 || p.pos < p.stop); iota++ {
			// TODO: collect documentation comments
			for p.tok == SPACE || p.tok == LINEND { p.next(true) }
			if p.tok == RPAREN || p.tok == EOF { break  }
			if opts.spec = p.directive(ctx); true {
				f(p.ctx(ctx), p.leadComment, &opts, iota)
			}
			if p.tok == COMMA || p.tok == LINEND { p.next(true) }
		}
		p.expect(RPAREN)
		if p.spaces(); p.tok != EOF { p.linend() }
		return
	}

	if p.tok != LINEND && p.tok != EOF && (p.stop == 0 || p.pos < p.stop) {
		if opts.spec = p.directive(ctx); true { f(ctx, nil, &opts, 0) }
		if p.tok == COMMA { p.next(true) }
	}
	if p.tok != EOF && (p.stop == 0 || p.pos < p.stop) {
		if p.spaces(); p.lineComment == nil { p.linend() }
	}
}

func (p *parser) assign_value(ctx Context, tok Token) (value Value) {
	defer func(a []*auto, b parseBits, f facet) {
		p.autos, p.bits, p.facet, p.lineComment = a, b, f, nil
	} (p.autos, p.bits, p.facet)

	p.bits |= parseDefineClause

	switch tok {
	case CO1_ASSIGN: p.facet |= expandDelegate
	case CO2_ASSIGN: p.facet |= expandDelegate|expandClosure
	case SM1_ASSIGN:
	case SM2_ASSIGN:
	default: if false { warn(ctx, "todo: decide expand facet: %v", tok).debug(1) }
	}

	var elems = p.values(ctx)

	// Create List value or use the first elem.
	if n := len(elems); n == 1 {
		value = elems[0]
	} else if n > 1 {
		value = makeList(elems...)
	}
	return
}

func (p *parser) assign(ctx Context, ident Value) (def *def) {
	if t_traverse.enabled { defer un(trace(t_traverse, fmt.Sprintf("assign(%s)", ident))) }

	var tok = p.tok

	p.next(true) // assign token

	ctx = p.ctx(ctx)

	// TODO: doc = p.leadComment
	// TODO: comment = p.lineComment
	var value = p.assign_value(ctx, tok)

	// NOTE: Put all explicit defs into project scope. It's important for defs enclosed
	//       in templates work.
	var loader = ctx.loader()
	if scope := loader.project.scope; len(loader.scopes) == 0 || loader.scopes[0] != scope {
		defer func(s []*Scope) { loader.scopes = s } (loader.scopes)
		loader.scopes = append([]*Scope{ scope }, loader.scopes...)
	}

	var defs = loader.define(ctx, tok, ident, value)
	if n := len(defs); n > 0 {  def = defs[n-1] }
	return
}

func (p *parser) recipe(ctx Context) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "Recipe")) }

	var (
		// TODO: comment *CommentGroup
		// TODO: doc = p.leadComment
		position = p.Position()
		loader = ctx.loader()
		elems []Value
		isList bool
	)

	switch p.dialect {
	case "", "eval", "value":
		p.scanner.pop(isCompoundLine)
		p.next(true) // skip RECIPE or SEMICOLON and parse in list mode
		position = p.Position()
		if isList = true; !p.isEndOfLine() {
			var a *argumented
			var x = p.expr(ctx) // parse first expr of recipe
			if x != nil { if a, _ = x.(*argumented); a != nil { x = a.Value } }
			if x == nil {
				erro(ctx, "parsed value is nil").debug(1)
			} else if p.dialect == "value" {
				// no resolving commands
			} else if t, y := x.(*bareword); !y {
				// does nothing
			} else if _, sym := loader.resolve(t); false {
				erro(ctx, "resolve '%v' failed", x).debug(1)
			} else if isTrivial(sym) {
				erro(of(ctx,x), "resolved '%v' (from %v) is nil", t.s, x).debug(1)
			} else if false {
				erro(of(ctx,x), "builtin command no more supported, use $(%s ...) instead", t.s).debug(1)
			} else if b, y := sym.(*builtin); !y {
				erro(of(ctx,x), "'%s' is not a command (%s)", t.s, typeof(sym)).debug(1)
			} else if !b.isCommand() {
				erro(of(ctx,x), "'%s' is not a command, use $(%s ...) instead", t.s, t.s).debug(1)
			} else { x = sym }

			if a != nil {
				elems, a.Value = append(elems, a), x
			} else {
				elems = append(elems, x)
			}

			var cmdargs []Value

			p.setbit(parseRecipeBuiltin)
			for p.tok != EOF && p.tok != SEMICOLON && p.tok != LINEND && p.lineComment == nil {
				if p.spaces(); p.lineComment != nil { break }
				if !p.tok.IsRuleDelim() { x = p.expr(ctx) } else
				if false { x = p.rule(ctx, specialRuleRec, nil, elems) } else {
					erro(ctx, "unsupported token: %s, %v", p.tok, elems).debug(1)
				}
				if cmdargs = append(cmdargs, x); p.tok == COMMA {
					p.next(true)
					elems = append(elems, makeList(cmdargs...))
					cmdargs = []Value{}
				}
				if p.lineComment != nil { break }
			}
			p.clearbit(parseRecipeBuiltin)
			elems = append(elems, makeList(cmdargs...))
		}

	default:
		p.scanner.push(isCompoundLine) // NOTE: scanner does not set isCompoundLine correctly, fixit here
		p.next(true) // skip RECIPE or SEMICOLON and parse in line-string mode
		position = p.Position()
		p.setbit(parseRecipeText)
		for !p.isEndOfLine() {
			var x Value
			if p.tok == RAW {
				x = p.literal(ctx, false)
			} else {
				x = p.expr(ctx)
			}
			elems = append(elems, x)
		}
		p.clearbit(parseRecipeText)
		p.scanner.pop(isCompoundLine)
	}
	if p.spaces(); p.tok != EOF { p.linend() }
    if len(elems) == 0 {
        return makeNone(position)
    } else if isList {
        return makeList(elems...)
    } else {
        return makeCompound(position, elems...)
    }
}

// Parsing (var a=xxx,b=yyy) definitions
func (p *parser) movar(ctx Context, args ...Value) (err error) {
	var loader = ctx.loader()
	for _, elem := range args {
		var kv, ok = elem.(*pair)
		if !ok || kv == nil {
			erro(of(ctx,elem), "bad var form (%T)", elem).debug(1)
			continue
		}

		var name string
		var k, v = kv.Key, kv.Value
		if name = k.string(at(ctx, k.Position())); name == "" {
			erro(of(ctx,k), "name '%v' is empty", k).debug(1)
		}

		if def, alt := loader.def(elem.Position(), name); alt != nil {
			erro(of(ctx,k), "'%v' already defined: %T", name, alt).debug(1)
		} else if def == nil {
			erro(of(ctx,k), "'%v' not defined", name).debug(1)
		} else {
			if g, y := v.(*group); y { v = g.list() }
			def.val(at(ctx,v.Position()), v)
		}
	}
	return
}

func (p *parser) defineConfigureTargets(ctx Context) {
	var loader = ctx.loader()
	for _, t := range p.targets {
		var pos = t.Position()
		if !pos.IsValid() { pos = p.Position() }

		var ctx = at(ctx, pos)
		var name = t.string(ctx)
		var d, a = loader.project.scope.define(ctx, DefConfig, name, nil)
		if d == nil && a != nil { if d, _ = a.(*def); d == nil {
			erro(ctx, "configure %v: already defined in '%v' as %v", t, loader.project, a).debug(6)
			return
		}}

		if !d.position.IsValid() { d.position = pos }
	}
}

func (p *parser) ruleParams(ctx Context, args []Value) (err error) {
	var scope = ctx.Scope()
	for _, elem := range args { var ctx = at(ctx, elem.Position())
		switch elem.(type) {
		case *bareword, *barecomp:
			p.params = append(p.params, scope.auto(ctx, elem.string(ctx), strconv.Itoa(len(p.params)+1)))
		default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
			erro(of(ctx,elem), "bad parameter form (%T)", elem)
		}
	}
	return
}

func (p *parser) modifier(ctx Context) (res *modifier) {
	p.spaces()

	defer p.setbits(p.setbit(parseModifier))
	p.clearbit(parseCall) // for commas

	ctx = p.ctx(ctx)

	p.expect(LPAREN)
	p.spaces()

	var name string
	var nameVal = p.expr(ctx)
	var elems []Value
	switch n := nameVal.(type) {
	case *bareword: name = n.s
	case *delegate, *closure:
		var ctx = at(ctx, n.Position())
		var v = xmerge(ctx, strval, nameVal)
		if len(v) == 0 {
			erro(ctx, "empty modifier name: %v", n).debug(1)
			return
		}
		name, elems = v[0].string(ctx), v[1:]
	default:
		erro(ctx, "unsupported modifier: %v{%v}", typeof(n), n).debug(1)
		return
	}

	var movar bool
	switch name {
	case "var": movar = true
	case "configure":
		p.defineConfigureTargets(ctx)
		p.configure = true // set configure flag and define configure variables
	case "":
		erro(ctx, "empty modifier name: %v{%v}", typeof(nameVal), nameVal).debug(1)
		return
	}

	if _, ok := dialects[name]; ok {
		if p.dialect == "" { p.dialect = name } else {
			erro(ctx, "multi-dialects unsupported, already defined '%s'", p.dialect).debug(1)
			return
		}
	} else if _, ok = modifiers[name]; !ok {
		erro(ctx, "`%s` no such dialect or modifier", name).debug(1)
		return
	}

	for p.tok != RPAREN && p.tok != EOF {
		p.spaces()

		t := p.pos

		if vals := p.values(ctx); movar {
			p.movar(ctx, vals...) // TODO: define var one by one
		} else if n := len(vals); n == 1 {
			elems = append(elems, vals[0])
		} else if n > 1 {
			elems = append(elems, &list{elements{vals}})
		} else {
			elems = append(elems, &null{valbase{p.Position()}})
		}

		if p.tok == COMMA { p.next(true) }
		if p.pos == t {
			erro(ctx, "unsupported modifier arg: %v '%v'", p.tok, p.lit).debug(1)
			return
		}
	}

	p.expect(RPAREN)

	if nameVal == nil && len(elems) == 0 {
		erro(ctx, "empty modifier").debug(1)
	} else {
		res = new(modifier)
		res.position = ctx.Position()
		res.Elems = append([]Value{nameVal}, elems...)
	}
	return
}

func (p *parser) modification(ctx Context) *modification {
	if t_traverse.enabled { defer un(trace(t_traverse, "modification")) }

	// defer p.setbits(p.setbit(parseModification))

	ctx = at(ctx, p.loc(p.expect(LBRACK)))

	var elems []*modifier
	for p.tok != EOF && p.tok != LINEND && p.tok != RBRACK {
		if m := p.modifier(ctx); m != nil { elems = append(elems, m) }
	}

	p.expect(RBRACK)

	if len(elems) == 0 {
		errostack(ctx, 5, "empty modifier group").debug(1)
	}
	if p.tok == COLON {
		errostack(ctx, 5, "unexpected colon after modifer").debug(1)
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
var automatics = []string{
	"@" , "%" , "<" , ">" , "?" , "^" , "+" , "|" , "*" , //
	"@D", "%D", "<D", ">D", "?D", "^D", "+D", "|D", "*D", //
	"@F", "%F", "<F", ">F", "?F", "^F", "+F", "|F", "*F", //
	"@'", "%'", "<'", ">'", "?'", "^'", "+'", "|'", "*'", //
	"-" , "~" ,
}

func (p *parser) rule(ctx Context, special specialRule, optvals, targets []Value) (result Value) {
	if ctx = p.ctx(ctx); ctx.Project().keyword == PACKAGE {
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
		scopeComment = fmt.Sprintf("rule %v", targets)
		position = ctx.Position()
	)

	var loader = ctx.loader()
	defer loader.closeScope(loader.openScope(scopeComment))
	p.params = nil
	p.dialect = ""

	var scope = loader.Scope()
	for _, s := range automatics {
		if a := scope.auto(ctx, s); a == nil { erro(ctx, "'%s' is not defined", s).debug(1) }
	}
	for i := 1; i < 10; i += 1 { s := strconv.Itoa(i)
		if a := scope.auto(ctx, s); a == nil { erro(ctx, "'%s' is not defined", s).debug(1) }
	}

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
		name := t.string(ctx)
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
		for _, d := range p.params { params = append(params, d.name(ctx)) }
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
			result = _makeList[Entry](res...)
		} else {
			result = makeNull(parsedData.position)
		}
	}

	// Close the rule scope and go back to project scope. The current
	// scope must be project scope befor Rule.
	p.configure = false
	p.dialect = ""
	p.params = nil
	return
}

func (p *parser) specialRule(ctx Context) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "SpecialRule")) }

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
			var ctx = p.ctx(ctx)
			// Options are flag or *pair of a Flag.
			for p.tok == MINUS {
				opt := p.expr(ctx)
				options = append(options, opt)
			}
			p.setbits(bits) // restore bits
			if p.tok.IsRuleDelim() {
				return p.rule(ctx, specialRuleUse, options, []Value{
					makeBareword(p.loc(pos), name),
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

func (p *parser) def(ctx Context) {
	defer dtrace(ctx, "parser.def")

	if p.spaces(); p.tok == LINEND {
		erro(p.ctx(ctx), "unexpected end of line").debug(1)
		return
	}

	p.expect(DEF)
	p.spaces()

	var args []Value
	var name = p.expr(ctx)

	if a, y := name.(*argumented); y {
		name, args = a.Value, a.args
	}

	t := &template{
		pos: p.pos, tok: p.tok, lit: p.lit, // verb: "def",
		state: p.scanner.scanState,
		name: name, params: args,
	}

	p.spaces()
	p.linend()

	var nested = 0
	for p.tok != EOF { switch pos := p.pos; p.tok {
	case DEF:
		p.next(true)
		nested += 1

	case END:
		if nested > 0 { nested -= 1 ; continue }

		p.next(true)
		p.linend()

		state := p.scanner.scanState
		t.end, t.endPos = &state, pos
		p.templates = append(p.templates, t)
		return

	default:
		for p.tok != EOF {
			if p.next(true); p.tok == LINEND { p.next(true) ; break }
		}
	}}
}

func (p *parser) foreach(ctx Context) {
	defer dtrace(ctx, "parser.foreach")

	if p.spaces(); p.tok == LINEND {
		erro(p.ctx(ctx), "unexpected end of line").debug(1)
		return
	}

	p.expect(FOREACH)
	p.spaces()

	var params = p.values(ctx)
	var t = &template{
		pos: p.pos, tok: p.tok, lit: p.lit,
		state: p.scanner.scanState, // verb: "foreach",
	}

	p.spaces()
	p.linend()

	var nested = 0
	for p.tok != EOF { switch pos := p.pos; p.tok {
	case FOREACH:
		p.next(true) // foreach
		nested += 1

	case DONE:
		if nested > 0 { nested -= 1 ; continue }

		p.next(true) // done
		p.linend()

		state := p.scanner.scanState
		t.end, t.endPos = &state, pos

		defer func(s Pos) { p.stop = s } (p.stop)
		p.stop = t.endPos

		var a = map[string]Value{ "_" : nil }
		for _, elem := range xmerge(ctx, plain, params...) {
			if !isTrivial(elem) { a["_"] = elem ; p.codeblock(ctx, t, a) }
		}
		return

	default:
		for p.tok != EOF {
			if p.next(true); p.tok == LINEND { p.next(true) ; break }
		}
	}}
}

func (p *parser) for_(ctx Context) {
	defer dtrace(ctx, "parser.for")

	if p.spaces(); p.tok == LINEND {
		erro(p.ctx(ctx), "unexpected end of line").debug(1)
		return
	}

	var opts struct {
		skipNil bool `sn,skip-nil,skipnil,skip-null,skipnull`
		loose bool `loose`
	}

	if p.expect(FOR); p.tok == LPAREN { p.next(true) // LPAREN
		if vals := parseOpts(ctx, &opts, 0, p.values(ctx)...); vals != nil {
			erro(of(ctx, vals[0]), "unexpected opts: %v", vals).debug(1)
		}
		p.expect(RPAREN)
	}

	p.spaces()

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
	for p.spaces(); p.tok != EOF && !p.isEndOfLine(); p.spaces() {
		if p.tok == AND && params == nil {
			erro(p.ctx(ctx), "unexpected 'and'").debug(1)
			continue
		} else if p.tok == AND || params == nil {
			if params = append(params, &nparam{p:p.Position()}); p.tok == AND {
				p.next(true) // and
				continue
			}
		}

		var _v = params[len(params)-1]
		for _, a := range xmerge(p.ctx(ctx), plain, p.expr(ctx)) {
			var (
				elems []Value
				s string
			)

			if x, y := a.(*pair); !y {
				erro(of(ctx,a), "unexpected value: %v(%v)", typeof(a), a).debug(1)
				return
			} else if s = x.Key.string(at(ctx, x.Key.Position())); s == "" {
				erro(of(ctx,a), "empty key: %v(%v)", typeof(x.Key), x.Key).debug(1)
				return
			} else if g, y := x.Value.(*group); y {
				elems = g.Elems
			} else {
				elems = append(elems, x.Value)
			}

			// Make sure all elements are expanded.
			elems = xmerge(of(ctx, a), plain, elems...)

			if _, y := vars[s]; y {
				erro(of(ctx, a), "duplicated key: %v", s).debug(1)
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
		state: p.scanner.scanState, // verb: "for",
	}

	p.spaces()
	p.linend()

	var nested = 0
	for p.tok != EOF { switch pos := p.pos; p.tok {
	case FOR:
		p.next(true) // for
		nested += 1

	case DONE:
		if nested > 0 { nested -= 1 ; continue }

		p.next(true) // done
		p.linend()

		defer func(s Pos) { p.stop = s } (p.stop)

		state := p.scanner.scanState
		t.end, t.endPos, p.stop = &state, pos, pos

		var num int
		for _, _v := range params { if _v.n > 0 {
			if num == 0 { num = _v.n } else { num *= _v.n }
		}}

		var l int = len(params)-1
		outer: for n := 0; n < num; n += 1 {
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
			if p.next(true); p.tok == LINEND { p.next(true) ; break }
		}
	}}
}

func (p *parser) codeblock(ctx Context, t *template, vars map[string]Value) {
	p.pos, p.tok, p.lit, p.scanner.scanState = t.pos, t.tok, t.lit, t.state

	// NOTE: comment here will affect loader.def()
	if false { pprofCounter += 1
		var name = fmt.Sprintf("template-%05d.prof", pprofCounter)
		defer startCPUProfile(ctx, name, true)()
	}

	defer dtrace(ctx, "parser.codeblock")

	if !(p.pos < p.stop) {
		erro(at(ctx,p.loc(p.pos)), "bad range: [%v %v) (%v)", p.pos, p.stop, t.name).debug(10)
	}

	var loader = ctx.loader()

	defer loader.closeScope(loader.openScope("codeblock"))

	ctx = &autoContext{
		Context: p.ctx(ctx),
		defs: make(autoDefMap),
	}

	var scope = loader.Scope()

	for s, v := range vars { if a := scope.auto(ctx, s); a != nil {
		a.set(ctx, v)
	} else {
		erro(ctx, "`%s` not defined", s).debug(1)
	}}

	defer func(v parseBits) { p.bits = v } (p.bits)

	p.bits |= parseCodeBlock

	for p.tok != EOF && p.pos < p.stop {
		if p.tok == SPACE || p.tok == LINEND || (p.tok == COMMENT && p.lineComment != nil) {
			p.next(true)
		} else {
			p.clause(ctx)
		}
	}
}

func (p *parser) repeat(ctx Context, t *template, params []Value) {
	defer func(t time.Time, pos Pos, tok Token, lit string, state scanState) {
		if u := cast[*universe](ctx); u.ddd == "template.repeat" {
			// dont check time in ddd mode
		} else if d := time.Now().Sub(t); d > u.slow {
            warnstack(ctx, 3, "slow: %v, prof-%d", d, pprofCounter).debug(1)
        }

		if ctx.dia().error() { erro(ctx, "template errors").debug(1) }

		p.pos, p.tok, p.lit, p.scanner.scanState = pos, tok, lit, state
	} (time.Now(), p.pos, p.tok, p.lit, p.scanner.scanState)

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

	var m = map[string]Value{}

	for i, v := range t.params { if s := v.string(ctx); s != "" {
		if i < len(params) { m[s] = params[i] } else {
			m[s] = makeNull(v.Position())
		}
	} else {
		erro(of(ctx,v), "empty template param name: %v %v", v, v).debug(1)
	}}

	p.codeblock(ctx, t, m)
}

func (p *parser) call(ctx Context, name Value, args []Value) (result bool) {
	ctx = p.ctx(ctx)

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
	if t_traverse.enabled { defer un(tracef(t_traverse, "clause(%v, %v)", p.tok, p.pos)) }

	var x Value
	var tok = p.tok // TODO: allow assigns like: `eval := xxx`

	defer dtrace(ctx, "parser.clause")

	defer func() { if cast[*universe](ctx).debugParsing(ctx, "clause") {
		warn(ctx, "clause: %v %v ; %v %v", typeof(x), x, p.tok, p.lit).debug(6)
	}} ()

	switch p.spaces(); tok {
	case  INCLUDE: p.spec(ctx, tok, p.expect(tok), p.include); return
	case    FILES: p.spec(ctx, tok, p.expect(tok), p.files); return
	case   ASSERT: p.spec(ctx, tok, p.expect(tok), p.assert); return
	case   APPEND: p.spec(ctx, tok, p.expect(tok), p.append); return
	case     EVAL: p.spec(ctx, tok, p.expect(tok), p.eval); return
	case    COLON: p.specialRule(ctx); return
	case      DEF: p.def(ctx); return
	case      FOR: p.for_(ctx); return
	case  FOREACH: p.foreach(ctx); return
	case   LINEND, SPACE: p.next(true) // skip empty lines
	case USE, TEMPLATE:
		erro(ctx, "`%v` unexpected here", p.tok).debug(10)
		return
	}

	x = p.expr(ctx, true)

	if p.spaces(); p.tok.IsAssign() {
		if cast[*universe](ctx).debugParsing(ctx, "define") {
			warn(p, "parser.clause: %s(%v); %v %v", typeof(x), x, p.tok, p.lit).debug(1)
			ctx.dia().flush()
		}
		p.assign(ctx, x)
		return
	}

	if p.tok.IsRuleDelim() {
		if cast[*universe](ctx).debugParsing(ctx, "rule") {
			warn(p, "parser.clause: %s(%v); %v %v", typeof(x), x, p.tok, p.lit).debug(1)
			ctx.dia().flush()
		}
		p.rule(ctx, specialRuleNor, nil, []Value{x})
		return
	} else if a, y := x.(*argumented); y {
		p.call(ctx, a.Value, a.args)
		return
	}

	if vals := p.values(ctx, x); p.tok != EOF {
		return
	} else if strings.HasSuffix(p.scanner.File().Name(), PathSep+configuration_sm) {
		if false { warn(ctx, "%v (kit=%s)", p.tok, p.lit).debug(1) }
	} else if p.isIncludingConf {
		warn(ctx, "bad clause: %v (kit=%s) after %v", p.tok, p.lit, vals).debug(3)
	} else {
		erro(ctx, "bad clause: %v (lit=%s) after %v", p.tok, p.lit, vals).debug(10)
	}
}

func (p *parser) setDefaultVars(ctx Context, filename, abs, rel, tmp string) (res bool) {
	var s = ctx.Scope()
	if s == nil {
		erro(ctx, "invalid scope").debug(1)
		return
	}

	var d *def

	if loader := ctx.loader(); loader.mode&Flat == 0 {
		var position = ctx.Position()

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

	return true
}

type projectDeclOpts struct {
	configure Value `c,conf,configure` // detects dotConfigure if empty
	noDock bool `n,nd,nod,nodock,no-dock` // don't load container project
    traveUseLoop bool `b,break;l,loop` // don't recursively use this project
    multiUseAllowed bool `m,multi`  // this project is used multiple times
	final bool `final` // no bases
}

func (p *parser) file(ctx Context) *parsedFile {
	if t_traverse.enabled  { defer un(trace(t_traverse, "File '"+p.scanner.File().Name()+"'")) }
	if cast[*universe](ctx).traceLaunch { defer un(trace(t_launch, "parser.file")) }
	if ctx.dia().error() { return nil }

	defer dtrace(ctx, "parser.file")

	var (
		ident *barecomp
		identStr string
		implicitBase string // aka. foo.bar.Baz implicitly load base 'foo/bar'
		abs, rel, tmp string
		loader   = ctx.loader()
		position = ctx.Position()
		keyword  = p.tok
		filename = p.scanner.File().Name()
		isMainFile = isEntryFileName(filename)
	)

	assert(loader != nil, "nil loader")

	defer loader.closeScope(loader.openScope(fmt.Sprintf("file %s", filename)))

	if loader.mode&Flat != 0 {
		abs = ctx.Project().absPath
	} else {
		abs = filepath.Dir(filename)
	}

	rel, _ = filepath.Rel(loader.workDir(), abs)
	tmp = joinTmpPath(ctx,loader.workDir(), rel)

	if !p.setDefaultVars(ctx, filename, abs, rel, tmp) { return nil }

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
			ident = makeBarecomp(position, makeBareword(position, basename))

		default:
			erro(ctx, "unknown configuration '%v', currently only 'configure .' is supported", p.tok)
		}
	case PROJECT:
		if loader.mode&Flat != 0 { erro(ctx, "forbidden `%v` in flat file", p.tok) }

		p.next(true)

		var ( // Options are flag or *pair of a flag.
			opts projectDeclOpts
			optVals []Value
			pos Position
		)
		for p.tok == MINUS {
			var opt = p.expr(ctx);  p.spaces()
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
				ident = makeBarecomp(position, makeBareword(position, linfo.loadee.name))
			} else if name := filepath.Base(filename); name == dotBase || name == dotConfigure {
				// NOTE: loading the .base or .configure file
				ident = makeBarecomp(position, makeBareword(position, name))
			} else if base := filepath.Base(dir); base != "" {
				// TODO: validate basename as a valid identifier
				ident = makeBarecomp(position, makeBareword(position, base))
			} else {
				erro(ctx, "invalid file: %v", filename).debug(1)
			}
		} else if p.tok == TILDE {
			/*if filename == confinitFilename {
                ident = &ast.Bareword{ ValuePos:pos, Value:"~" }
            } else*/ if ext := filepath.Ext(filename); ext != ".smart" {
				erro(p, "`%v` not a smart file", filepath.Base(filename)).debug(1)
			} else if s := strings.TrimSuffix(filepath.Base(filename), ext); s != "" {
				ident = makeBarecomp(position, makeBareword(position, s))
			} else {
				erro(p, "`%v` not tilde name", filepath.Base(filename)).debug(1)
			}
			p.next(true) // skip tilde
		} else {
			// var t = p.tok
			var implicitBaseSegs []string
			ident = makeBarecomp(p.Position())
		ForProjectName:
			for p.tok != EOF && p.tok != SPACE {
				if w := p.bare(ctx, false); w == nil {
					erro(at(ctx,ident.Position()), "expecting a bareword").debug(1)
				} else if word, ok := w.(*bareword); !ok {
					erro(at(ctx,ident.Position()), "expecting a bareword: %v (%T)", w, w).debug(1)
				} else if ident.comp(ctx, word); p.tok == DOT {
					ident.comp(ctx, &punctuation{valbase{p.Position()}, p.tok}) // TODO: parse to Qualiword
					implicitBaseSegs = append(implicitBaseSegs, word.s)
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

		if identStr = ident.string(ctx); linfo.loadee != nil && identStr != linfo.loadee.name {
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
					param := p.expr(ctx)
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
			loader.configure(ctx, linfo, ident, identStr, declared)
			if !opts.noDock { loader.container(ctx, ident, identStr) }
		}
	case EOF:
		return nil
	default:
		if loader.mode&Flat == 0 {
			p.expected(p.pos, "configure, project, module or package keyword")
		}
	}

	var auto = (loader.mode&Flat == 0) && isMainFile //&& isEntryFileName(filename)
	if auto { loader.autoload(p.ctx(ctx), "declared") }
	if loader.mode&ModuleClauseOnly == 0 {
		if loader.mode&Flat == 0 { ForDeclare: for p.tok != EOF {
			switch tok := p.tok; tok {
			case USE: p.spec(ctx, tok, p.expect(tok), p.use)
			case LINEND, SPACE: p.next(true) // skip empty lines
			case ASSERT, EVAL, FILES, INCLUDE: p.clause(ctx)
			default: break ForDeclare
			}
		}}

		if false && auto { loader.autoload(p.ctx(ctx), "amid") }

		if loader.mode&ImportsOnly == 0 { // rest of module body
			for /* !p.dia().error() && */ p.tok != EOF {
				if p.tok == LINEND || (p.tok == COMMENT && p.lineComment != nil) {
					p.next(true)
				} else if p.clause(p.ctx(ctx)); ctx.dia().flush() > 0 {
					break
				}
			}
		}
	}
	if auto { loader.autoload(p.ctx(ctx), "appendix") }

	if  cast[*universe](ctx).ddd == "parser.files" {
		cast[*universe](ctx).ddd = ""
	}

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
