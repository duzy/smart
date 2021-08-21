///
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"extbit.io/smart/token"
	"extbit.io/smart/scanner"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"fmt"
)

type parsingBits uint
const (
	composing parsingBits = 1<<iota
	composingSELECT_PROP
	composingDOT
	composingDOTDOT
	composingPATH
	composingGLOB
	composingPERC
	composingREXP
	composingURL
	composingModifier

	parsingFilesSpec // files ( ... )
	parsingSpecialRule // e.g. :use ...:
	//parsingColonName // e.g. $:use:
	parsingBuiltinCommand // recipe builtin command
	parsingRecipeText

	// The composingNo* bits control the composing priority!
	composingNoArg    = composingSELECT_PROP | composingDOT | composingDOTDOT | composingPATH | composingPERC
	composingNoPair   = composingSELECT_PROP | composingDOT | composingPATH | composingPERC
	composingNoURL    = composingSELECT_PROP | composingDOT | composingPATH | composingURL | composingGLOB | composingPERC | composingREXP /*| parsingColonName*/ | parsingSpecialRule
	composingNoPath   = composingSELECT_PROP | composingDOT | composingPATH | composingURL | composingGLOB | composingPERC | composingREXP
	composingNoSelect = composingSELECT_PROP | composingDOT
	composingNoGlob   = composingGLOB | composingPERC | composingREXP
	composingNoPerc   = composingGLOB | composingPERC | composingREXP
	composingNoRexp   = composingGLOB | composingPERC | composingREXP
)

type specialRule int
const (
	specialRuleNor specialRule = iota // normal rules
	specialRuleUse // `use` rules
	specialRuleRec // recipe rules
)

const (
	selfproj = "self"
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
	name *Barecomp // project/module name
	scope *Scope
	using []*usespec // imports
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
	//p *parser
	state scanner.ScanState
	pos token.Pos   // token position
	tok token.Token // one token look-ahead
	lit string      // token literal
	verb string
	params []Value
}

type parser struct {
	*loader

	file    *token.File
	scanner scanner.Scanner

	// Comments
	comments  []*CommentGroup
	leadComment *CommentGroup // last lead comment
	lineComment *CommentGroup // last line comment

	// Next token
	pos token.Pos   // token position
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

	bits parsingBits
    isIncludingConf bool // including configuration

	// Ordinary identifier scopes
	imports []*usespec // list of imports

	targets []Value // targets of current rule
	params []*Def // parameters of current rule
	dialect string // recipe dialect of current rule
	configure bool // is parsing configure program?
}

func (p *parser) init(l *loader, filename string, src []byte) {
	p.loader = l
	p.file = l.fset.AddFile(filename, -1, len(src))

	var m scanner.Mode
	if p.mode&ParseComments != 0 {
		//m = scanner.ScanComments
	}

	eh := func(pos token.Position, msg string) {
		diag.errorAt(Position(pos), "%s", msg)
	}
	p.scanner.Init(p.file, src, eh, m)
	p.next(true)
}

func (p *parser) setbits(bits parsingBits) { p.bits = bits }
func (p *parser) setbit(bit parsingBits) (bits parsingBits) {
	bits = p.bits
	p.bits |= bit
	return
}
func (p *parser) clearbit(bit parsingBits) (bits parsingBits) {
	bits = p.bits
	p.bits &= ^bit
	return
}

// ----------------------------------------------------------------------------
// Parsing support

func (p *parser) trace(a ...interface{}) { t_traverse.traceAt(p.position(), a...) }

// Advance to the next token.
func (p *parser) scanNext() {
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
	if false && p.tok == token.EOF {
		diag.errorAt(p.positionAt(pos), "unexpected end of file").debug(optionDebugErrors)
	}
}

// Consume a comment and return it and the line on which it ends.
func (p *parser) consumeComment() (comment *Comment, endline int) {
	// /*-style comments may end on a different line than where they start.
	// Scan the comment for '\n' chars and adjust endline accordingly.
	endline = p.file.Line(p.pos)
	if len(p.lit) > 1 && p.lit[1] == '*' {
		// don't use range here - no need to decode Unicode code points
		for i := 0; i < len(p.lit); i++ {
			if p.lit[i] == '\n' {
				endline++
			}
		}
	}

	comment = &Comment{ Pos: p.position(), Text: p.lit }
	p.scanNext()

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

	endline = p.file.Line(p.pos)
	for p.tok == token.COMMENT && p.file.Line(p.pos) <= endline+n {
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
func (p *parser) _next() {
	p.leadComment = nil
	p.lineComment = nil
	prev := p.pos
	if p.scanNext(); p.tok == token.COMMENT {
		var comment *CommentGroup
		var endline int

		// If the comment is on same line as the previous token; it
		// cannot be a lead comment but may be a line comment.
		if p.file.Line(p.pos) == p.file.Line(prev) {
			comment, endline = p.consumeCommentGroup(0)
			if p.file.Line(p.pos) != endline {
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

		if endline+1 == p.file.Line(p.pos) {
			// The next token is following on the line immediately after the
			// comment group, thus the last comment group is a lead comment.
			p.leadComment = comment
		}
	}
}

func (p *parser) next(skipWS bool) {
	if p._next(); skipWS && p.tok == token.SPACE { p._next() }
}

func (p *parser) skipSpaces() {
	for p.lineComment == nil && p.tok != token.EOF {
		if p.tok == token.SPACE  { p._next() } else
		if p.tok == token.ESCAPE && p.lit == "\n" {
			if p._next(); p.tok == token.LINEND { break }
			if p.bits&parsingBuiltinCommand != 0 {
				TokFor: for p.tok != token.EOF {
					switch p.tok {
					case token.RECIPE:
						p.scanner.LeaveCompoundLineContext()
						p._next()
					default: break TokFor
					}
				}
			}
		} else {
			break
		}
	}
}

func (p *parser) positionAt(pos token.Pos) Position { return Position(p.file.Position(pos)) }
func (p *parser) position() Position { return p.positionAt(p.pos) }

// A bailout panic is raised to indicate early termination.
type bailout struct{}

func (p *parser) info(pos token.Pos, s string, a... interface{}) {
	diag.infoAt(p.positionAt(pos), s, a...)
}
func (p *parser) warn(pos token.Pos, s string, a... interface{}) {
	diag.warnAt(p.positionAt(pos), s, a...)
}

func (p *parser) expected(pos token.Pos, msg string, a... interface{}) {
	if len(a) > 0 { msg = fmt.Sprintf(msg, a...) }
	msg = "expected " + msg
	if pos == p.pos {
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
	diag.errorAt(p.positionAt(pos), msg).
		debug(optionDebugErrors, 16)
}

func (p *parser) expect(tok token.Token) token.Pos {
	pos := p.pos
	if p.tok != tok {
		p.expected(pos, "'"+tok.String()+"'")
	}
	p._next() // make progress
	return pos
}

func (p *parser) expectLinend() {
	if p.lineComment != nil {
		// The line comment is treated as LINEND, simply ignore it.
		p.lineComment = nil
	} else if p.tok == token.LINEND {
		p._next()
	} else {
		p.expected(p.pos, "'\\n'")
	}
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
func (p *parser) _safePos(pos token.Pos) (res token.Pos) {
	defer func() {
		if recover() != nil {
			res = token.Pos(p.file.Base() + p.file.Size()) // EOF position
		}
	}()
	_ = p.file.Offset(pos) // trigger a panic if position is out-of-range
	return pos
}

// ----------------------------------------------------------------------------
// Barewords & Identifiers

func (p *parser) parseBarewordConstant(lhs bool) (x Value) {
	var pos, tok, value = p.pos, p.tok, ""
	switch tok {
	case token.BAREWORD:
		value = p.lit
	case token.AT, token.DOT, token.DOTDOT: // TODO: parse token.DOT into Qualiword
		value = tok.String() // Special bareword.
	default:
		if tok.IsKeyword() {
			value = tok.String()
		} else {
			p.expect(token.BAREWORD)
		}
	}

	p._next() // consumes the word

	switch position := p.positionAt(pos); tok {
	case token.TRUE:  x = MakeBoolean(position,  true)
	case token.FALSE: x = MakeBoolean(position,  false)
	case token.YES:   x = MakeAnswer(position,   true)
	case token.NO:    x = MakeAnswer(position,   false)
	default:          x = MakeBareword(position, value)
	}
	return
}

func (p *parser) parseSelector() (res Value) {
	defer p.setbits(p.setbit(composingSELECT_PROP))
	res = p.parseExpr(false)
	return
}

func (p *parser) parseSelect(lhs Value) (res Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Select")) }

	pos := p.pos
	tok := p.tok // the arrow '->' or '=>'
	p._next() // skip '->' or '=>'

	position := p.positionAt(pos)
	switch t := lhs.(type) {
	case *selection:
		diag.errorAt(position, "TODO: multiple selection: %v", lhs).
			debug(optionDebugErrors, 1)
	case *Bareword:
        switch t.string {
        case "goals": lhs = context.goals
        case "os":    lhs = context.globe.os.self
		case "mode":  lhs = context.mode
        case "usee":  lhs = p.project.using
        case "self":  lhs = p.project.self
        default:
            if o, err := p.resolve(lhs); err != nil {
				diag.errorOf(lhs, "resolve selection object '%v' error: %v", lhs, err).
					debug(optionDebugErrors, 1)
                return
            } else if o == nil {
                if tok == token.SELECT_PROG2 {
					res = MakeNil(position) // ignore
                } else {
					diag.errorOf(lhs, "'%v' is undefined", lhs).
						debug(optionDebugErrors, 1)
                }
                return
            } else {
                lhs = o
            }
        }
    case *Barecomp: // for cases like '.foo'
        if o, err := p.resolve(t); err != nil {
			diag.errorOf(lhs, "resolve selection object '%v' error: %v", lhs, err).
					debug(optionDebugErrors, 1)
			return
        } else if isNil(o) {
			if tok == token.SELECT_PROG2 {
				res = MakeNil(position) // ignore
			} else {
				diag.errorOf(lhs, "'%v' is undefined", lhs).
					debug(optionDebugErrors, 1)
			}
			return
		} else {
			lhs = o
        }
	}

	if rhs := p.parseSelector(); isNil(rhs) {
		res = MakeNil(position)
	} else {
		res = MakeSelection(position, tok, lhs, rhs)
	}

	if (p.tok == token.SELECT_PROP || p.tok == token.SELECT_PROG1 || p.tok == token.SELECT_PROG2) {
		res = p.parseSelect(res) // Continue the selection recursivly.
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
	if (p.bits&parsingRecipeText != 0) && p.tok == token.RECIPE {
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

func (p *parser) parseDependList() (list []Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Depends")) }
	for p.tok != token.SEMICOLON && p.tok != token.BAR && !p.isEndOfLine() {
		if p.tok == token.COLON { // FIXME: this check is not working!
			// FIXME: detects unexpected colon ':'
			diag.errorAt(p.position(), "unexpected colon").
				debug(optionDebugErrors, 1)
			p.next(true) // just ignore this colon
		} else {
			p.skipSpaces()
			list = append(list, p.parseExpr(false))
			p.skipSpaces()
		}
	}
	return
}

// If lhs is set, result list elements which are identifiers are not resolved.
func (p *parser) parseExprList(lhs bool) (list []Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "List")) }
	for p.skipSpaces(); !p.isEndOfList(lhs); {
		p.skipSpaces()
		list = append(list, p.parseExpr(lhs))
		p.skipSpaces()
		// If there's a comment right after the parsed expression, we break
		// the expression list to treat the end-of-line comment like a LINEND.
		if p.lineComment != nil  { break }
		if p.tok == token.LINEND { break }
		if p.tok == token.EOF    { break }
	}
	return
}

func (p *parser) parseListExpr(lhs bool) *List {
	return MakeList(p.position(), p.parseExprList(lhs)...)
}

func (p *parser) setRHS(v bool) (old bool) {
	old = p.inRhs; p.inRhs = v; return
}

func (p *parser) parseLhsList() []Value {
	defer p.setRHS(p.setRHS(false))
	// Line comment of previous lines will break the parsing loop,
	// so we clear the previous line comment
	p.lineComment = nil
	return p.parseExprList(true)
}

func (p *parser) parseRhsList() []Value {
	defer p.setRHS(p.setRHS(true))
	return p.parseExprList(false)
}

// ----------------------------------------------------------------------------
// Expressions

func (p *parser) parseGroupExpr(lhs bool) *Group {
	if t_traverse.enabled { defer un(trace(t_traverse, "Group")) }

	position := p.position()
	p.next(true)
	elems, converted := p.parseExprList(false), false
	for p.tok == token.COMMA {
		p.next(true)
		next := p.parseListExpr(false)
		if !converted {
			elems = []Value{ MakeList(p.position(), elems...), next }
			converted = true
		} else {
			elems = append(elems, next)
		}
	}
	p.expect(token.RPAREN)
	return MakeGroup(position, elems...)
}

func (p *parser) parseArgumentedExpr(x Value) *Argumented {
	if t_traverse.enabled { defer un(trace(t_traverse, "Argumented")) }

	p.next(true) // skip token.LPAREN

	var a = []Value{ p.parseListExpr(false) }
	for p.tok == token.COMMA {
		p.next(true) // skip token.COMMA
		a = append(a, p.parseListExpr(false))
	}
	p.expect(token.RPAREN)
	return MakeArgumented(x, a...)
}

func (p *parser) parseGlobMeta() (x *GlobMeta) {
	pos, tok := p.position(), p.tok
	p._next()
	return MakeGlobMeta(pos, tok)
}

func (p *parser) parseGlobRange() (x *GlobRange) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Glob")) }

	position := p.position()
	p.expect(token.LBRACK) // skip '['

	chars := p.parseExpr(false)
	p.expect(token.RBRACK) // skip ']'

	return MakeGlobRange(position, chars)
}

func (p *parser) parseGlobExpr(x Value) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "Glob")) }

	var (
		pos = p.position()
		components []Value
	)
	if !isNil(x) { components = []Value{ x } }

	// avoid nesting glob expressions
	defer p.setbits(p.setbit(composingGLOB))
ForGlobTok:
	for {
		if p.lineComment != nil { break ForGlobTok }
		switch p.tok {
		case token.RPAREN, token.COMMA, token.SPACE, token.LINEND, token.EOF:
			break ForGlobTok
		case token.STAR, token.QUE: // * ?
			x = p.parseGlobMeta()
		case token.LBRACK:
			// FIXME: '[...]' has been used for modifier expressions
			x = p.parseGlobRange()
		default:
			// FIXME: escaped glob metas/chars
			x = p.parseExpr(false)
		}
		components = append(components, x)
	}
	if components == nil {
		diag.errorAt(pos, "nil glob expression (tok=%v, lit=%v)", p.tok, p.lit)
	}
	return MakeGlobPattern(pos, components...)
}

func (p *parser) parsePercExpr(lhs bool, x Value) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "Perc")) }

	// avoid nesting percent expressions
	defer p.setbits(p.setbit(composingPERC))

	var (
		pos = p.pos
		y Value
	)
	if p._next(); pos+1 == p.pos { // joint, e.g. '%.o', but skip '% .o'
		switch p.tok {
		case token.COLON, token.COLON2,
			token.LPAREN, token.RPAREN,
			token.LBRACK, token.RBRACK,
			token.LBRACE, token.RCOLON,
			token.PCON,   token.SEMICOLON,
			token.COMMA,  token.SPACE,
			token.LINEND:
		case token.PERC: // %%
			p._next() // consume the second %
			position := p.position()
			perc2 := MakePercPattern(position, nil, nil)
			if pos+2 == p.pos {
				switch p.tok {
				case token.PERC: // %%%
					p.error(p.pos, "too many %")
				case token.PCON: // FIXES: %%/xxx -> Path(%% xxx)
					x = MakePercPattern(position, x, perc2)
					return p.parsePathExpr(lhs, x)
				case token.COLON,    token.COLON2,
					token.LPAREN,    token.RPAREN,
					token.LBRACK,    token.RBRACK,
					token.LBRACE,    token.RCOLON,
					token.SEMICOLON, token.COMMA,
					token.SPACE,     token.LINEND:
				default:
					var (
						yy = p.parseExpr(false)
						_, ok = yy.(*Path)
					)
					if ok { diag.errorAt(position, "incorrect: %v, %v", x, yy) }
					assert(!ok, "the second part of aaa%%bbb/foo/bar parsed incorrectly as path")
					perc2.Suffix = yy
				}
			}
			y = perc2
		default:
			y = p.parseExpr(false)
		}
	}
	return MakePercPattern(p.positionAt(pos), x, y)
}

func (p *parser) parseRegexpExpr() (x Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Regexp")) }

	// avoid nesting percent expressions
	defer p.setbits(p.setbit(composingREXP))

	p.error(p.pos, "todo: regexp")

	return
}

func (p *parser) parseKeyValueExpr(x Value) *Pair {
	if t_traverse.enabled { defer un(trace(t_traverse, "Pair")) }
	position := p.position(); p._next()
	return MakePair(position, x, p.parseExpr(false))
}

func (p *parser) parseFlagExpr(lhs bool) *Flag {
	if t_traverse.enabled { defer un(trace(t_traverse, "Flag")) }

	var (
		position = p.position()
		x Value
	)

	p._next() // skip dash '-'

	// Flag expressions, excluding "-)" "-]" "-}" "-\n", "-=", "-:", etc.
	if p.isEndOfLine() || p.isEndOfList(false) {
		x = MakeNil(position)
	} else if false {
		x = p.parseExpr(false)
	} else {
		x = p.parseUnaryExpr(false)
	}
	return MakeFlagValue(position, x)
}

func (p *parser) parseNegExpr(lhs bool) *negative {
	if t_traverse.enabled { defer un(trace(t_traverse, "Negative")) }
	p.expect(token.EXC)
	return Negative(p.parseExpr(lhs))
}

func (p *parser) parseBasicLit(lhs bool) (v Value) {
	pos, tok, lit := p.pos, p.tok, p.lit
	end := int(pos) + len(lit)
	switch tok {
	case token.STRING: end += 2
	}
	p._next()
	// ESCAPE is handled in value.EscapeChar
    var position = Position(p.file.Position(pos))
    switch tok {
    case token.BAR: p.error(pos, "`|` is deprecated, changed the modifiers!")
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
    case token.ESCAPE:   v = MakeString(position, EscapeChar(lit))
    case token.RAW:      v = MakeRaw(position, lit)
    default: unreachable()
    }
	return
}

func (p *parser) parseCompoundLit(lhs bool) *Compound {
	var (
		lpos = p.pos
		elems []Value
	)
	p._next()
ForCompound:
	for p.tok != token.EOF && p.tok != token.COMPOSED {
		var x Value
		switch p.tok {
		default:           x = p.parseExpr(false)
		case token.RAW:    x = p.parseBasicLit(false)
		case token.LINEND:
			p.error(p.pos, "unexpected end of line for compound string")
			break ForCompound
		}
		elems = append(elems, x)
	}
	p.expect(token.COMPOSED)
	return MakeCompound(Position(p.file.Position(lpos)), elems...)
}

// Parses dot composing expressions (TODO: check against file extensions).
//   .foo
//   .'foo'
//   ."foo"
//   .(foo)
//   ..foo
//   ..'foo'
//   .foo.bar
func (p *parser) parseDotExpr(lhs bool, x Value) (res *Barecomp) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Dot")) }

	defer p.setbits(p.setbit(composingDOT))

	var comp *Barecomp
	if x == nil { panic(fmt.Sprintf("nil dot (tok=%v)", p.tok)) }
	if comp, _ = x.(*Barecomp); comp == nil {
		comp = MakeBarecomp(x.Position())//(p.position())
		comp.Elems = append(comp.Elems, x)
	}

	for /*comp.End() == p.pos && */!p.isEndOfDotConcat(lhs) {
		comp.Combine(p.parseComposedExpr(false))
		if p.tok == token.DOT /*&& comp.End() == p.pos*/ {
			var dot = MakeBareword(p.position(), ".") // TODO: parse to Qualiword instead
			comp.Elems = append(comp.Elems, dot)
			p._next() // '.'
		}
	}

	// FIXME: *.o => obj
	//   BUG: Barecomp{Glob . KeyValueExpr}
	//   FIX: KeyValueExpr{Barecomp, Bareword}

	return comp
}

func makePathSeg(pos Position, tok token.Token) *PathSeg {
	var r rune
    switch tok {
    case 0:            r = 0 // the tailing empty segment after '/', e.g. /foo/bar/
    case token.PCON:   r = '/' // TODO: should be NONE
    case token.TILDE:  r = '~'
    case token.DOT:    r = '.'
    case token.DOTDOT: r = '^' // 
    default:
		diag.errorAt(pos, "invalid path segment `%v`", tok)
		return nil
    }
	return MakePathSeg(pos, r)
}

func (p *parser) parsePathExpr(lhs bool, start Value) *Path {
	if t_traverse.enabled { defer un(trace(t_traverse, "Path")) }

	defer p.setbits(p.setbit(composingPATH))

	var (
		position = start.Position() //p.position()
		path *Path
		ok bool
	)
	if start == nil {
		diag.errorAt(position, "bad closure/delegate name").debug(optionDebugErrors,1)
		p._next()
		return MakePath(position) // empty path
	} else if path, ok = start.(*Path); !ok {
		path = MakePath(position, start)
	}

BuildPath:
	for p.tok == token.PCON {
		var pos = p.position() // skips repeated '/' sequence
		for p._next(); p.tok == token.PCON; p._next() { pos = p.position() }
		switch p.tok {
		case token.RPAREN, token.LPAREN, token.RBRACE, token.LBRACE,
			 token.RCOLON, token.COMMA, token.SPACE, token.LINEND:
			// Encountered the tailing '/', append 'zero' segment.
			path.Elems = append(path.Elems, MakePathSeg(pos, 0))
			break BuildPath
		}

		var x = p.parseComposedExpr(false)
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

func (p *parser) parseURLExpr(lhs bool, scheme Value) (res Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "URL")) }

	defer p.setbits(p.setbit(composingURL))

	var (
		url = &URL{ Scheme:scheme }
		colon1 = p.expect(token.COLON) // consumes ':'
		colon2 = token.NoPos
		//colon3 = token.NoPos
		at = token.NoPos // @
	)

	if p.tok == token.PCON {
		p._next() // the first '/'
		if p.tok == token.PCON {
			p.expect(token.PCON) // the second '/'
		} else {
			diag.errorAt(p.position(), "TODO: URL path: %v (%T) (next: %s (%s))",
				scheme, scheme,  p.tok, p.lit).debug(optionDebugErrors)
			res = MakeNil(p.position())
			return
		}
	} else if !p.isEndOfURL(lhs) {
		diag.errorAt(p.positionAt(colon1), "TODO: URL: %v (%T) (next: %s (%s))",
			scheme, scheme,  p.tok, p.lit).debug(optionDebugErrors)
		res = MakeNil(p.position())
		return
	}

	if !p.isEndOfURL(lhs) {
		userOrHost := p.parseComposedExpr(false)
		if p.tok == token.COLON {
			url.Username, colon2 = userOrHost, p.pos
			p._next() // ':'
			if p.tok != token.AT && p.tok != token.PCON && !p.isEndOfURL(lhs) {
				url.Password = p.parseComposedExpr(false)
			}
		} else {
			url.Host = userOrHost
		}
		if p.tok == token.AT {
			p._next() // '@'
		}
	}
	if url.Host == nil && colon2 == token.NoPos && at == token.NoPos && !p.isEndOfURL(lhs) {
		url.Host = p.parseComposedExpr(false)
		if p.tok == token.COLON {
			//colon3 = p.pos
			p._next() // ':'
			if p.tok != token.SPACE && p.tok != token.LINEND {
				url.Port = p.parseComposedExpr(false)
			}
		}
	}
	if p.tok == token.PCON {
		url.Path = p.parsePathExpr(lhs, makePathSeg(p.position(), p.tok))
	}
	// scanning '#' as token.HASH instead of token.COMMENT
	defer p.scanner.SetBits(p.scanner.AddBits(scanner.NoComments))
	if p.tok == token.QUE {
		p._next() // '?'
		if p.tok != token.HASH && !p.isEndOfURL(lhs) {
			url.Query = p.parseComposedExpr(false)
		}
	}
	if p.tok == token.HASH {
		p._next() // '#'
		if !p.isEndOfURL(lhs) {
			url.Fragment = p.parseComposedExpr(false)
		}
	}
	return url
}

func (p *parser) parseClosureDelegateName(tok token.Token) (name Value) {
	if t_traverse.enabled {	defer un(trace(t_traverse, "ClosureDelegateName")) }

	var pos = p.position()
	if name = p.parseExpr(false); isNil(name) {
		diag.errorAt(pos, "bad closure/delegate name").debug(optionDebugErrors)
		name = MakeNil(pos)
	}
	return
}

func (p *parser) parseClosureDelegate() (result Value) {
	if t_traverse.enabled {	defer un(trace(t_traverse, "ClosureDelegate")) }

	// FIXME: push p.bits before entering a $(...) or &(...)
	defer func(a parsingBits) { p.bits = a } (p.bits)
	p.bits = 0 // start with zero bits

	var (
		pos  = p.pos
		tok  = p.tok
		posLp = token.NoPos
		tokLp  token.Token
		name   Value
		rest []Value
		resolved Value // Object or *selection
		obj Value
	)

	resolveConfig := func() (obj Object) {
		if p.project.configure != nil {
			if s, err := name.Strval(); err != nil {
				diag.errorOf(name, "strval '%v' failed: %v", name, err)
			} else if obj, err = p.project.configure.resolveObject(s); err != nil {
				diag.errorOf(name, "resolve configure '%s' failed: %v", s, err)
			}
		}
		return
	}

	resolveObject := func() (okay bool) {
		var err error
		switch tokLp {
		case token.LPAREN:
			if resolved, err = p.resolve(name); err != nil {
				diag.errorOf(name, "resolve '%v' failed: %v", name, err)
			} else if isNil(resolved) {
				if p.isIncludingConf {
					if s, err := name.Strval(); err != nil {
						diag.errorOf(name, "strval '%v' error: %v", name, err)
					} else {
						// Create an empty Def if it's referred in configuration.sm.
						def, _ := p.def(name.Position(), s)
						def.origin = DefConfRef
						obj, okay = def, true
					}
				} else if sel, ok := name.(*selection); ok {
					if o, err := sel.object(); err == nil && o.DeclScope().comment == usecomment {
						obj, okay = unresolved(p.project, name), true
					} else if err != nil {
						diag.errorOf(sel, "`%v` invalid delegate selection", name)
					} else if isNil(o) {
						diag.errorOf(sel, "`%v` nil selection object", name)
					} else if v, err := sel.value(); err != nil {
						diag.errorOf(name, "`%v` invalid delegate selection", name)
					} else if isNil(v) {
						diag.errorOf(name, "`%v` not found in %v", sel.s, o)
					} else if obj, okay = v.(Object); !okay {
						// just use the selected value
					}
				} else if o := resolveConfig(); !isNil(o) {
					obj, okay = o, true
				} else if tok.IsClosure() || refdef(name, defany) || name.expandible(expandClosure) {
					obj, okay = unresolved(p.project, name), true // recursive delegation or closure
				} else if name.expandible(expandPlainValue) {
					s, _ := name.Strval()
					diag.errorOf(name, "resolved '%v' (aka. %s) is nil (project=%v)", name, s, p.project).
						debug(optionDebugErrors,1)
				} else {
					diag.errorOf(name, "resolved '%v' is nil (project=%v)", name, p.project).
						debug(optionDebugErrors,1)
				}
			} else if sel, _ := resolved.(*selection); !isNil(sel) {
				obj, okay = sel, true
			} else if def, _ := resolved.(Caller); def == nil {
				p.error(posLp, "resolved '%v' is not callable: %T", name, resolved)
			} else if obj, okay = def.(Object); isNil(obj) || !okay {
				p.error(posLp, "resolved '%v' is not object: %T", name, def)
			}
		case token.LBRACE:
			if resolved, err = p.find(name); err != nil {
				p.error(posLp, "finding rule entry '%v' failed: %v", name, err)
			} else if isNil(resolved) {
				if name.expandible(expandPlainValue) {
					s, _ := name.Strval()
					diag.errorOf(name, "resolved '%v' (aka. %s) is nil (project=%v)", name, s, p.project).
						debug(optionDebugErrors,1)
				} else {
					diag.errorOf(name, "resolved '%v' is nil (project=%v)", name, p.project).
						debug(optionDebugErrors,1)
				}
			} else if exe, _ := resolved.(Executer); exe == nil {
				p.error(posLp, "resolved '%v' of '%T' is not Executer", name, resolved)
			} else if obj, okay = exe.(Object); !okay || isNil(obj) {
				p.error(posLp, "resolved Executer '%v' of '%T' is not Object", name, resolved)
			}
		case token.LCOLON:
			s, err := name.Strval()
			if err != nil {
				p.error(posLp, "error strval name '%v': %v", name, err)
				return
			}
			switch s {
			case "goals": resolved = context.goals
			case "os":    resolved = context.globe.os.self
			case "mode":  resolved = context.mode
			case "usee":  resolved = p.project.using
			case "self":  resolved = p.project.self
			default:
				p.error(posLp, "unknown special property: %v", s, err)
				return
			}
			obj, okay = resolved, true
		}
		return
	}

	switch p._next(); p.tok {
	case token.LPAREN, token.LBRACE, token.LCOLON: // $(...), ${...}, $:...:
		posLp, tokLp = p.pos, p.tok

		p._next() // skips LPAREN, LBRACE, LCOLON
		if posLp+1 != p.pos {
			diag.errorAt(p.positionAt(posLp+1), "unexpected spaces").
				debug(optionDebugErrors, 1)
			return MakeNil(p.position())
		}

		if name = p.parseClosureDelegateName(tokLp); isNil(name) {
			diag.errorAt(p.positionAt(posLp), "parsed name is nil")
		} else if !name.expandible(expandClosure) && !resolveObject() {
			diag.errorOf(name, "name '%v' is unidentified (project=%v)", name, p.project).
				debug(optionDebugErrors, 1)
		}

		if  (tokLp == token.LPAREN && p.tok != token.RPAREN) ||
			(tokLp == token.LBRACE && p.tok != token.RBRACE) ||
			(tokLp == token.LCOLON && p.tok != token.RCOLON) {
			rest = append(rest, p.parseListExpr(false))
			for p.tok == token.COMMA {
				p.next(true)
				rest = append(rest, p.parseListExpr(false))
			}
		}

		switch tokLp {
		case token.LPAREN: p.expect(token.RPAREN)
		case token.LBRACE: p.expect(token.RBRACE)
		case token.LCOLON: p.expect(token.RCOLON)
			if p.tok == token.ASSIGN { diag.errorAt(p.position(), "unexpected assignment") }
		}

	default:
		if position := p.position(); tok != token.CLOSURE { // $(...), disabled $name.
			// &(...), &{...}, &'...', &"..."
			diag.errorAt(position, "expects `%v` or `%v` or quotes", token.LPAREN, token.LBRACE).debug(optionDebugErrors)
			return MakeNil(position)
		} else if p.tok == token.STRING || p.tok == token.COMPOUND {
			posLp, tokLp = p.pos, p.tok

			// &'xxxx' or &"xxxx"
			if name = p.parseExpr(false); isNil(name) {
				p.error(posLp, "parsed name is nil")
			} else if !name.expandible(expandClosure) && !resolveObject() {
				diag.errorOf(name, "name '%v' is unidentified", name)
			}
		} else {
			// &(...), &{...}, &'...', &"..."
			diag.errorAt(position, "expects `%v`, `%v` or quotes", token.LPAREN, token.LBRACE).debug(optionDebugErrors)
			return MakeNil(position)
		}
	}
	if isNil(obj) && p.project.plugin != nil && p.project.pluginScope != nil {
		s, err := name.Strval()
		if err != nil {
			p.error(pos, "strval name '%v' failed: %v", name, err)
			return
		}
		obj = p.project.pluginScope.Lookup(s)
	}

	if position := p.positionAt(pos); tok.IsDelegate() {
		return MakeDelegate(position, tokLp, obj, rest...);
	} else {
		return MakeClosure(position, tokLp, obj, rest...);
	}
}

func (p *parser) parseSpecialClosureDelegate(lhs bool) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "SpecialClosureDelegate")) }

	var pos, tok, s = p.pos, p.tok, p.tok.String()[1:]
	p._next()

	var (
		position = p.positionAt(pos)
		name = MakeBareword(position, s)
		resolved, err = p.resolve(name)
		obj Object
	)
	if err != nil {
		diag.errorOf(name, "resolve '%v' failed: %v", name, err).
			debug(optionDebugErrors, 1)
		return MakeNil(position)
	} else if resolved == nil {
		diag.errorOf(name, "resolved '%v' is nil", name).
			debug(optionDebugErrors, 1)
		return MakeNil(position)
	} else if def, ok := resolved.(Caller); def == nil || !ok {
		diag.errorOf(resolved, "resolved '%v' is not callable: %T", name, resolved).
			debug(optionDebugErrors, 1)
		return MakeNil(position)
	} else if obj, ok = def.(Object); obj == nil || !ok {
		diag.errorOf(resolved, "resolved '%v' is not object: %T", name, def).
			debug(optionDebugErrors, 1)
		return MakeNil(position)
	}

	if tok.IsDelegate() {
		return MakeDelegate(position, tok, obj);
	} else {
		return MakeClosure(position, tok, obj);
	}
}

func (p *parser) parseUnaryExpr(lhs bool) (x Value) {
	if t_traverse.enabled && false { defer un(trace(t_traverse, "Unary")) }

	switch p.tok {
	case token.BAREWORD, token.AT:
		return p.parseBarewordConstant(lhs)

	case token.BIN, token.OCT, token.INT, token.HEX, token.FLOAT,
		token.DATETIME, token.DATE, token.TIME, token.URI,
		/*token.RAW,*/ token.STRING, token.ESCAPE:
		return p.parseBasicLit(lhs)

	case token.COMPOUND:
		return p.parseCompoundLit(lhs)

	case token.DELEGATE, token.CLOSURE: // delegate, closure
		return p.parseClosureDelegate()

	case token.LPAREN:
		return p.parseGroupExpr(lhs)

	case token.TILDE, token.DOT, token.DOTDOT: // ~ . ..
		var str = p.tok.String()
		tok, pos, end := p.tok, p.pos, p.pos+token.Pos(len(str))
		position := p.positionAt(pos)
		if p._next(); end != p.pos { // FIXME: ~user
			// '~', '.' or '..' used as bareword
			return MakeBareword(position, str)
		} else if p.tok == token.PCON { // check /
			return p.parsePathExpr(lhs, makePathSeg(position, tok))
		} else if tok == token.DOT { // TODO: parse to Qualiword instead
			if x = MakeBareword(p.positionAt(pos), str); p.bits&composingDOT == 0 {
				x = p.parseDotExpr(lhs, x)
			}
			return
		} else if tok == token.TILDE { // TODO: ~user
			return makePathSeg(p.positionAt(pos), tok)
		} else {
			diag.errorAt(position, "unexpected path segment").debug(optionDebugErrors)
			return MakeNil(position)
		}

	case token.PCON: // The root of the path
		return p.parsePathExpr(lhs, makePathSeg(p.position(), p.tok))

	case token.LBRACK:
		return p.parseModifiersExpr()

	case token.STAR, token.QUE/*, token.LBRACK*/: // * ? [
		return p.parseGlobExpr(nil) // (ie. no prefix)

	case token.PERC: // %bar (ie. no prefix)
		return p.parsePercExpr(lhs, nil)

	case token.LBRACE: // TODO: regexp: {^.*}   or token.REGEXP
		return p.parseRegexpExpr()

	case token.MINUS:
		return p.parseFlagExpr(lhs)

	case token.EXC:
		return p.parseNegExpr(lhs)

	default:
		if p.tok.IsClosure() || p.tok.IsDelegate() {
			return p.parseSpecialClosureDelegate(lhs)
		} else if p.tok.IsKeyword() { // keywords here are barewords
			return p.parseBarewordConstant(lhs)
		}
	}

	var position = p.position()
	diag.errorAt(position, "'%v' bad unary expression (lit=%s,lhs=%v)", p.tok, p.lit, lhs).
		debug(optionDebugErrors)

	p._next() // go to next token
	return MakeNil(position)
}

func (p *parser) parseComposedExpr(lhs bool) (x Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Composed")) }
	switch x = p.parseUnaryExpr(lhs); p.tok { // check composible expressions
	case token.SELECT_PROP, token.SELECT_PROG1, token.SELECT_PROG2: // foo->bar  foo=>bar  foo~>bar
		if p.bits&composingNoSelect == 0 {
			// accepts 'foo=>bar', but 'foo => bar' is different
			x = p.parseSelect(x); break
		}

	case token.LBRACK: // xxx[(foo ...)]
		if p.bits&composingModifier == 0 {
			// FIXME: compose lhs x
			m := p.parseModifiersExpr()
			diag.errorOf(m, "composing modifiers is ignored (unimplemented yet)")
		}
	case token.STAR, token.QUE/*, token.LBRACK*/: // foo*bar foo?bar foo[a-z]bar
		if p.bits&composingNoGlob == 0 {
			x = p.parseGlobExpr(x)
		}
	case token.PERC: // foo%bar
		// FIXME: %/foo/bar -> Path(% foo bar)
		if p.bits&composingNoPerc == 0 {
			x = p.parsePercExpr(lhs, x)
		}
	case token.DOT: // foo.bar.baz.o
		// FIXME: push bits when parsing $(...)
		if p.bits&composingDOT == 0 { // TODO: parse to Qualiword
			x = p.parseDotExpr(lhs, x)
		}
	case token.PCON: // ie. subdir/in/somewhere
		if p.bits&composingNoPath == 0 {
			// Path expressions, except '-I/path/to/include'
			switch x.(type) {
			case *Flag: // By pass expressions like -I/foo/bar.
			default: x = p.parsePathExpr(lhs, x)
			}
		}
	case token.COLON:
		if (p.bits&parsingBuiltinCommand != 0 || !lhs) && p.bits&composingNoURL == 0 {
			if s, _ := x.Strval(); isKnownURLScheme(s) {
				x = p.parseURLExpr(lhs, x)
			}
		}
	}
	return
}

func (p *parser) parseText() (res []Value) {
	if false && t_traverse.enabled { defer un(trace(t_traverse, "Text")) }
	for p.tok != token.EOF {
		res = append(res, p.parseExpr(false))
	}
	return
}

func (p *parser) parseExpr(lhs bool) (x Value) {
	if false && t_traverse.enabled { defer un(trace(t_traverse, "Expression")) }

	var pos, tok = p.pos, p.tok
	if x = p.parseComposedExpr(lhs); x == nil {
		p.error(pos, "`%v` invalid expression", tok)
		return
	}
	{
		if lhs && p.tok.IsAssign() { return }

	SwitchCompose:
		switch p.tok {
		case token.ASSIGN: // Example: '*.o = obj'
			if !lhs && p.bits&composingNoPair == 0 {
				x = p.parseKeyValueExpr(x)
			}
			return

		case token.SELECT_PROG1, token.SELECT_PROG2:
			if p.bits&composingNoSelect == 0 {
				x = p.parseSelect(x)
			}
			return

		case token.LPAREN:
			if p.bits&composingNoArg == 0 {
				if false { if _, ok := x.(*Argumented); ok { p.error(pos, "nested argumentation") }}
				if x = p.parseArgumentedExpr(x); !isNil(x) {
					goto SwitchCompose
				}
			}
			return

		case token.PCON:
			if p.bits&composingNoPath == 0 {
				// Path expressions, except '-I/path/to/include'
				switch x.(type) {
				case *Flag: // By pass expressions like -I/foo/bar.
				default: x = p.parsePathExpr(lhs, x)
				}
			}
			return // FIXES: a%%b/foo/bar -> Path(a%%b foo bar)

		case token.BAR:
			if _, ok := x.(*Group); ok { return } // in case of: [(var)|...]

		case token.COMPOSED, token.COMMA, token.COLON, token.SEMICOLON,
             token.RPAREN, token.RBRACK, token.RBRACE, token.RCOLON,
             token.RAW, token.SELECT_PROP, /*token.SELECT_PROG,*/
             token.SPACE, token.LINEND, token.EOF:
			// Compose nothing at this point!
			return
		}

		var y = p.parseComposedExpr(lhs)
		if _, ok := y.(*Path); ok {
			switch x.(type) {
			case *Flag: // okay: -Ifoo/bar, -Lfoo/bar
			case *Path: // okay: combine two paths
			case *String, *Compound, *delegate, *closure:
			default:
				diag.warnOf(y, "barecomp a path: %v (%T), %v (%T) (next=%v)", x, x, y, y, p.tok).
					debug(optionDebugErrors, 1)
			}
		}

		// Further composing
		switch t := x.(type) {
		case *Barecomp: t.Combine(y)
		case *Path: t.Combine(y)
			if false { diag.infoAt(t.position, "%v (%v) (tok=%v)", t, y, p.tok) }
		default:
			comp := MakeBarecomp(x.Position(), x)
			comp.Combine(y)
			x = comp
		}

		// Keep trying composing as long as possible
		switch p.tok {
		case token.SPACE, token.LINEND, token.EOF: break
		default: //case token.SELECT_PROG1, token.SELECT_PROG2, token.LPAREN:
			goto SwitchCompose
		}
	}
	return
}

// ----------------------------------------------------------------------------
// Clauses & Declarations

type parseSpecFunc func(doc *CommentGroup, generic *genericoptions, iota int)

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

// replaces parseUseProps
func (p *parser) parseUseSpecProps(props []Value) (opts importspecoptions, params []Value, err error) {
    // Supported parameter forms:
    //      -param
    //      -param(value)
    //      -param=value
    var useList []Value // TODO: apply useList
    for _, prop := range props {
        var s string
        switch t := prop.(type) {
        case *Flag:
            if s, err = t.name.Strval(); err != nil {
                diag.errorOf(prop, "invalid flag `%v` (%v)", prop, err)
                return
            }
            switch s {
            case "nouse", "unuse": opts.unuse = true
            case "reuse": opts.reuse = true
            default: params = append(params, prop)
            }
        case *Pair: // -param=value
            switch tt := t.Key.(type) {
            case *Flag:
                if s, err = tt.name.Strval(); err != nil {
                    diag.errorOf(t.Key, "invalid flag name `%v` (%v)", tt.name, err)
                    return
                }
                switch s {
                case "use": useList = append(useList, t.Value)
                default: params = append(params, prop)
                }
            default:
                diag.errorOf(t.Key, "parameter `%v' unsupported `%T`", prop, prop)
                return
            }
        case *Argumented: // -param(value)
            switch tt := t.value.(type) {
            case *Flag:
                if s, err = tt.name.Strval(); err != nil {
                    diag.errorOf(t.value, "invalid flag name `%v` (%v)", tt.name, err)
                    return
                }
                switch s {
                case "use": useList = append(useList, t.args...)
                default: params = append(params, prop)
                }
            default:
                diag.errorOf(t.value, "parameter `%v' unsupported `%T`", prop, prop)
                return
            }
        default:
            diag.errorOf(prop, "parameter `%v` unsupported `%T`", prop, prop)
            return
        }
    }
    return
}

type useOpts struct {
	reuse bool `r,reuse;r,reusing`
	noFiles bool `n,nofiles;nf,no-files`
}
func (p *parser) parseUseSpec(doc *CommentGroup, generic *genericoptions, _ int) {
	var props = p.parseDirectiveSpec()
	p.imports = append(p.imports, &usespec{ props })
	if generic.dontOperate { return }

	var (
		uo useOpts
		opts importoptions // allowReuse noFiles
		err error
	)
	if _, err = parseOpts(p.position(), &uo, generic.options...); err != nil {
		diag.errorAt(p.position(), "parse use opts failed: %v", err)
		return
	} else {
		opts.allowReuse = uo.reuse
		opts.noFiles = uo.noFiles
	}

	var (
        specOpts importspecoptions
        specNames []string
		specVal = props[0]
        params []Value
	)
	if specOpts, params, err = p.parseUseSpecProps(props[1:]); err != nil {
		diag.errorOf(specVal, "use opts error: %v", err)
		return
	}

    switch v := specVal.(type) {
    case *Pair:
        var s string
        if f, ok := v.Key.(*Flag); !ok {
            diag.errorOf(specVal, "'%v' invalid use spec", v.Key)
            return
        } else if s, err = f.name.Strval(); err != nil {
            diag.errorOf(specVal, "%s", err)
            return
        } else if s != "list" {
            diag.errorOf(specVal, "'%v' invalid use spec, do you mean -list?", v.Key)
            return
        }

        var list []Value
        if list, err = mergeresult(ExpandAll(v.Value)); err != nil {
            diag.errorOf(specVal, "%s", err)
            return
        }
        for _, val := range list {
            if s, err = val.Strval(); err != nil {
                diag.errorOf(specVal, "%s", err)
                return
            } else if s == "" { continue }
            specNames = append(specNames, s)
        }
    default:
        var specName string
        if specName, err = specVal.Strval(); err != nil {
            diag.errorOf(specVal, "%s", err)
            return
        } else if specName == "" { break }
        specNames = append(specNames, specName)
    }

    if len(specNames) == 0 {
        diag.errorOf(specVal, "empty use spec (%v)", specVal)
        return
    }
    for _, specName := range specNames {
        p.loadUseSpecName(opts, specVal, specName, &specOpts, params)
    }

    if err == nil && false {
        var using Object
        if using, err = p.project.resolveObject("using.*"); err == nil && !isNil(using) {
            if def, ok := using.(*Def); ok {
                diag.infoOf(specVal, "TODO: using: %T %v", def, def.value).debug(true, 1)
            }
        }
        err = nil
    }
	return
}

func (p *parser) parseIncludeSpec(doc *CommentGroup, generic *genericoptions, _ int) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Spec")) }

	var (
		x = p.parseExpr(false)
		// TODO: comment = p.lineComment
		//props []Value
	)

	if p.tok == token.COLON {
		x = p.parseRuleEntry(specialRuleNor, nil, []Value{x}) // this should return a RuleEntry
	}

	if !generic.dontOperate {
		p.includeFile(p.position(), x)
	}
}

func (p *parser) parseFilesSpec(doc *CommentGroup, generic *genericoptions, _ int) {
	defer p.setbits(p.setbit(parsingFilesSpec))
	var (
		props = p.parseDirectiveSpec()
		path Value
	)
	if len(props) != 1 {
		diag.errorAt(p.position(), "too many files properties: %v", props)
		return
	}
	if p.tok == token.SELECT_PROG1 {
		p.next(true) // step forward with spaces skipped
		if p.tok == token.LINEND || p.lineComment != nil {
			p.error(p.pos, "expecting files path")
		}
		path = p.parseExpr(false)
	}
	if p.skipSpaces(); p.lineComment != nil {
		//fmt.Fprintf(stderr, "%v: %v %v %v\n", p.file.Position(p.pos), p.tok, p.lineComment, spec.Comment)
		//spec.Comment = p.lineComment
	}
	if generic.dontOperate {
		// TODO: maybe give some information
		return
	}
	var (
		val = props[0]
		pats []Value
	)
	if g, ok := val.(*Group); ok {
		pats = g.Elems
	} else {
		pats = []Value{ val }
	}
	if path == nil {
		var paths = []Value{ MakeString(val.Position(), p.project.absPath) }
		for _, k := range pats { p.project.mapfile(k, paths) }
	} else {
		var patsNew []Value
		for _, pat := range pats {
			if pat.expandible(expandClosure) {
				patsNew = append(patsNew, pat)
			} else {
				if v, err := mergeresult(ExpandAll(pat)); err != nil {
					diag.errorOf(pat, "merge value '%v' failed: %v", pat, err)
				} else {
					patsNew = append(patsNew, v...)
				}
			}
		}
		var paths []Value
		if g, ok := path.(*Group); ok {
			paths = g.Elems
		} else {
			paths = []Value{ path }
		}
		for _, k := range patsNew { p.project.mapfile(k, paths) }
	}
}

func (p *parser) parseEvalSpec(doc *CommentGroup, generic *genericoptions, _ int) {
	var (
		position = p.position()
		props = p.parseDirectiveSpec()
		resolved Value
		err error
	)
	if prop0 := props[0]; isNil(prop0) {
		diag.errorAt(position, "illegal").
			debug(optionDebugErrors, 1)
	} else if position = prop0.Position(); !position.IsValid() {
		diag.errorAt(position, "command name '%v' has invalid position", prop0).
			debug(optionDebugErrors, 1)
	} else if resolved, err = p.resolve(prop0); err != nil {
		diag.errorAt(position, "resolve '%v' failed: %v", prop0, err).
			debug(optionDebugErrors, 1)
	} else if isNil(resolved) {
		diag.errorAt(position, "resolved '%v' is nil", prop0).
			debug(optionDebugErrors, 1)
	} else if b, ok := resolved.(*Builtin); ok && (b.flag&builtinCommand) == 0 {
		diag.errorAt(position, "resolved builtin '%v' is not a command: %T", prop0, resolved).
			debug(optionDebugErrors, 1)
	} else if !generic.dontOperate { //p.evalspec(spec)
        // At the point of `eval` was represented, the closure context
        // might be empty. So we start closure with the current scope.
        defer setclosure(setclosure(cloctx.unshift(p.scope)))
		var res Value
        switch op := prop0.(type) {
        case Caller: res = op.Call(position, props[1:]...)
        default:
            var ( name string; err error )
            if name, err = op.Strval(); err != nil {
                diag.errorAt(position, "strval '%s' failed: %v", op, err).
					debug(optionDebugErrors, 1)
            } else if _, obj := p.scope.Find(name); obj == nil {
                diag.errorAt(position, "`%s` undefined", name).
					debug(optionDebugErrors, 1)
            } else if f, _ := obj.(Caller); f == nil {
                diag.errorAt(position, "`%T` is not caller (%s)", obj, name).
					debug(optionDebugErrors, 1)
            } else {
                res = f.Call(position, props[1:]...)
            }
        }
		if !isNil(res) {
			// TODO: using res value
		}
 	}
}

func (p *parser) parseDirectiveSpec() (props []Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Spec")) }

	var (
		//doc = p.leadComment
		comment *CommentGroup
	)

	p.skipSpaces()

	// Parse the directive expression
	props = append(props, p.parseExpr(false))

ParamsParseLoop: // Parse the directive parameters
	for p.tok != token.EOF {
		if p.skipSpaces(); p.lineComment != nil { comment = p.lineComment; break }
		switch p.tok {
		case token.SELECT_PROG1, token.COMMA, token.LINEND, token.RPAREN, token.RBRACE, token.RCOLON:
			break ParamsParseLoop
		}
		props = append(props, p.parseExpr(false))
		if p.skipSpaces(); p.tok == token.LINEND { break }
		if p.lineComment != nil { comment = p.lineComment; break }
	}
	if comment != nil {
		// TODO: ...
	}
	return
}

func (p *parser) parseGenericClause(keyword token.Token, pos token.Pos, f parseSpecFunc) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Clause("+keyword.String()+")")) }

	p.skipSpaces()

	var (
		//doc = p.leadComment
		generic = genericoptions{ keyword: keyword }
		//lparen, rparen token.Pos
	)

	for p.tok == token.MINUS {
		var (
			x = p.parseExpr(false)
			conds []Value
		)
		switch t := x.(type) {
		case *Argumented:
			if flag, ok := t.value.(*Flag); !ok {
				// does nothing
			} else if s, err := flag.name.Strval(); err != nil {
				diag.errorAt(x.Position(), "strval flag name '%v' failed: %v", flag.name, err)
			} else if s == "cond" {
				conds = t.args
			}
		case *Pair:
			if flag, ok := t.Key.(*Flag); !ok {
				// does nothing
			} else if s, err := flag.name.Strval(); err != nil {
				diag.errorAt(x.Position(), "strval flag name '%v' failed: %v", flag.name, err)
			} else if s == "cond" {
				if g, ok := t.Value.(*Group); ok {
					conds = g.Elems
				} else {
					conds = append(conds, t.Value)
				}
			}
		}
		if conds == nil {
			generic.options = append(generic.options, x)
			continue
		}
		for _, cond := range conds {
			if t, e := cond.True(); e != nil {
				diag.errorAt(x.Position(), "conditon casting '%v' failed: %v", cond, e)
			} else if !t {
				generic.dontOperate = true
				break
			}
		}
	}

	if p.skipSpaces(); p.tok == token.LPAREN {
		//lparen = p.pos
		p.next(true)
		if false { fmt.Fprintf(stderr, "%v: GenericSpec.1: %v(%s)\n", p.file.Position(p.pos), p.tok, p.lit) }
		for iota := 0; p.tok != token.RPAREN && p.tok != token.EOF; iota++ {
			// TODO: collect documentation comments
			for p.tok == token.SPACE || p.tok == token.LINEND { p.next(true) }
			if p.tok == token.RPAREN || p.tok == token.EOF { break  }
			f(p.leadComment, &generic, iota)
			if p.tok == token.COMMA || p.tok == token.LINEND { p.next(true) }
		}
		/*rparen = */p.expect(token.RPAREN)
		if p.tok != token.EOF { p.expectLinend() }
	} else {
		if false { fmt.Fprintf(stderr, "%v: GenericSpec.3: %v(%s)\n", p.file.Position(p.pos), p.tok, p.lit) }
		for iota := 0; p.tok != token.LINEND && p.tok != token.EOF; iota++ {
			f(nil, &generic, iota)
			if p.lineComment != nil { break }

			// Checking for `include xxx:[...]`
			/* FIXME: if inc, _ := spec.(*ast.IncludeSpec); inc != nil && len(inc.Props) > 0 {
				if p, ok := inc.Props[0].(*ast.IncludeRuleClause); ok && p != nil {
					goto GoodEnd
				}
			}*/

			if p.tok == token.COMMA { p.next(true) }
		}
		if p.tok != token.EOF {
			p.expectLinend()
		}
	}
	//GoodEnd:
}

func (p *parser) parseDefineClause(tok token.Token, ident Value) (def *Def) {
	if t_traverse.enabled { defer un(trace(t_traverse, fmt.Sprintf("Define(%s)", ident))) }

	// Only accept scoped identifiers if it's ":user:" program
	if p.scope.comment == usecomment {
		switch i := ident.(type) {
		case *selection:
			diag.errorOf(ident, "should use scoped names instead of `%v`", i)
		default:
			diag.errorOf(ident, "FIXME: unexpected name expression: %T", i)
		}
		return
	}

	var (
		// TODO: doc = p.leadComment
		// TODO: comment = p.lineComment
		position = p.positionAt(p.expect(tok))
		elems = p.parseRhsList()
		value Value
	)

	// Take the line comment, since the line comment is assigned.
	p.lineComment = nil

	// Create List value or use the first elem.
	if n := len(elems); n == 1 {
		value = elems[0]
	} else if n > 1 {
		value = MakeList(p.position(), elems...)
	}

	// NOTE: forcely put all explicit defs into project scope. It's important for defs enclosed
	//       in templates work.
	defer func(s *Scope) { p.scope = s } (p.scope)
	p.scope = p.project.scope

	var defs = p.determine(position, tok, ident, value)
	if n := len(defs); n > 0 {  def = defs[n-1] }
	return
}

func (p *parser) parseDefine(ident Value) (def *Def) {
	return p.parseDefineClause(p.tok, ident)
}

func (p *parser) parseRecipeDefineClause(x Value) Value {
	// TODO: validate x ...
	return p.parseDefineClause(p.tok, x)
}

func (p *parser) parseRecipeRuleClause(elems []Value) (x Value) {
	return p.parseRuleEntry(specialRuleRec, nil, elems)
}

func (p *parser) parseRecipeExpr() Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "Recipe")) }

	var (
		// TODO: comment *CommentGroup
		// TODO: doc = p.leadComment
		position = p.position()
		elems []Value
	)

SwitchDialect:
	switch p.dialect {
	case "", "eval", "value":
		p.scanner.LeaveCompoundLineContext()
		p.next(true) // skip RECIPE or SEMICOLON and parse in list mode
		if !p.isEndOfLine() {
			defer p.setbit(p.setbit(parsingBuiltinCommand))
			var (
				isVal = p.dialect == "value"
				x = p.parseExpr(!isVal) // parse first expr of recipe
			)
			if isNil(x) {
				diag.errorAt(position, "parsed value is nil")
			} else if t, ok := x.(*Bareword); ok && !isVal {
				if sym, err := p.resolve(t); err != nil {
					diag.errorAt(position, "resolve '%v' failed: %v", x, err)
				} else if isNil(sym) {
					diag.errorAt(position, "resolved '%v' (from %v) is nil", t.string, x)
				} else {
					x = sym
				}
			}

			if p.tok.IsAssign() {
				elems = append(elems, p.parseRecipeDefineClause(x))
				break SwitchDialect
			}

			elems = append(elems, x)
			var cmdargs []Value
			for p.tok != token.EOF && p.tok != token.SEMICOLON && p.tok != token.LINEND && p.lineComment == nil {
				p.skipSpaces()

				if p.tok.IsRuleDelim() {
					x = p.parseRecipeRuleClause(elems) // RuleEntry
				} else {
					x = p.parseExpr(true)
				}

				cmdargs = append(cmdargs, x)
				if p.tok == token.COMMA {
					p.next(true)
					elems = append(elems, MakeList(p.position(), cmdargs...))
					cmdargs = []Value{}
				}
				if p.lineComment != nil {
					// TODO: comment = p.lineComment
					break
				}
			}
			elems = append(elems, MakeList(p.position(), cmdargs...))
		}

	default:
		p.next(true) // skip RECIPE or SEMICOLON and parse in line-string mode
		for !p.isEndOfLine() {
			var bits = p.setbit(parsingRecipeText)
			var x Value
			switch p.tok {
			default:           x = p.parseExpr(false)
			case token.RAW:    x = p.parseBasicLit(false)
				/*
			case token.LINEND:
				p.error(p.pos, "unexpected end of line for compound string")
				break ForCompound*/
			}
			elems = append(elems, x)
			p.setbits(bits)
		}
	}
	if p.tok != token.EOF { p.expectLinend() }
    if len(elems) == 0 {
        return MakeNone(position)
    } else if p.dialect == "" || p.dialect == "eval" {
        return MakeList(p.position(), elems...)
    } else {
        return MakeCompound(position, elems...)
    }
}

func (p *parser) parseModifySetVar(args []Value) (err error) {
	// Parsing (var a=xxx,b=yyy) definitions
	for _, elem := range args[1:] {
		var kv, ok = elem.(*Pair)
		if !ok || kv == nil {
			diag.errorOf(elem, "bad var form (%T)", elem)
			continue
		}
		var name string
		var k, v = kv.Key, kv.Value
		if name, err = k.Strval(); err != nil {
			diag.errorOf(k, "strval '%v' failed: %v", k, err)
		} else if name == "" {
			diag.errorOf(k, "name '%v' is empty", k)
		}
		if def, alt := p.def(elem.Position(), name); alt != nil {
			diag.errorOf(k, "Def '%v' already existed: %T", name, alt)
		} else if def != nil {
			if g, ok := v.(*Group); ok {
				def.val(g.ToList(def.position))
			} else {
				def.val(v)
			}
		}
	}
	return
}

func (p *parser) defineConfigureModifierVars() {
	for _, t := range p.targets {
		if name, e := t.Strval(); e != nil {
			diag.errorOf(t, "strval target '%v' failed: %v", t, e)
		} else if def, alt := p.def(t.Position(), name); alt != nil {
			diag.errorOf(t, "configuration '%s' already defined: %v", name, alt)
		} else if def != nil {
			if false { diag.infoOf(def, "configure: %v", def) }
		}
	}
}

func (p *parser) parseModifyParams(args []Value) (err error) {
	for _, elem := range args {
		switch elem.(type) {
		case *Bareword, *Barecomp:
			var s string
			if s, err = elem.Strval(); err != nil {
				diag.errorOf(elem, "strval '%v' failed: %v", elem, err)
			}
			var def, alt = p.def(elem.Position(), s)
			if alt != nil {
				var ok bool
				if def, ok = alt.(*Def); !ok {
					diag.errorOf(elem, "%T '%s' already taken the name, no such parameter", alt, s)
				}
			}
			if def != nil {
				def.set(DefArg, nil)
			} else {
				diag.errorOf(elem, "'%s' is not defined", s)
			}
			p.params = append(p.params, def)
			p.scope.replace(strconv.Itoa(len(p.params)), def)
		default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
			diag.errorOf(elem, "bad parameter form (%T)", elem)
		}
	}
	return
}

func (p *parser) parseModifiersExpr() *modifiergroup {
	if t_traverse.enabled { defer un(trace(t_traverse, "Modifiers")) }

	var (
		posLp = p.positionAt(p.expect(token.LBRACK))
		elems []*modifier
	)

	defer func(a parsingBits) { p.bits = a }(p.bits)
	p.bits |= composingModifier

ForModifiersExpr:
	for p.tok != token.RBRACK && p.tok != token.EOF {
		p.skipSpaces()

		var (
			x = p.parseExpr(false)
			group *Group
			name string
			err error
		)
		if g, ok := x.(*Group); !ok {
			diag.errorAt(g.position, "modifier not represented by group: %T", x).
				debug(optionDebugErrors, 1)
			continue ForModifiersExpr
		} else {
			group = g
		}
		if l, ok := group.Elems[0].(*List); ok {
			group.Elems = append([]Value{ l.Elems[0] }, append(l.Elems[1:], group.Elems[1:]...)...)
		}

		switch n := group.Elems[0].(type) {
		case *Bareword:
			if name = n.string; name == "var" {
				p.parseModifySetVar(group.Elems)
				continue ForModifiersExpr
			} else if name == "configure" {
				p.defineConfigureModifierVars()
				p.configure = true // set configure flag and define configure variables
			}
			goto checkNameAndAdd
		case *Group: // parameters: ((foo bar))
			p.parseModifyParams(n.Elems)
			continue ForModifiersExpr
		case *delegate, *closure, *Barecomp, *String:
			var v []Value
			if v, err = mergeresult(ExpandAll(n)); err != nil {
				diag.errorOf(n, "merge '%v' failed: %v", v, err).
					debug(optionDebugErrors, 1)
			} else if name, err = v[0].Strval(); err != nil {
				diag.errorOf(n, "strval '%v' failed: %v", v[0], err).
					debug(optionDebugErrors, 1)
				continue ForModifiersExpr
			} else if name == "" {
				diag.errorOf(n, "name '%v' is empty", n).
					debug(optionDebugErrors, 1)
				continue ForModifiersExpr
			}
			goto checkNameAndAdd
		default:
			diag.errorOf(n, "unsupported dialect or modifier (%T): %v", group.Elems[0], group.Elems[0]).
				debug(optionDebugErrors, 1)
			continue ForModifiersExpr
		}

		goto addModifier

	checkNameAndAdd:
		if _, ok := dialects[name]; ok {
			if p.dialect == "" { p.dialect = name } else {
				diag.errorOf(x, "multi-dialects unsupported, already defined '%s'", p.dialect).
					debug(optionDebugErrors, 1)
				continue ForModifiersExpr
			}
		} else if _, ok = modifiers[name]; !ok {
			diag.errorOf(x, "`%s` no such dialect or modifier", name).
				debug(optionDebugErrors, 1)
			continue ForModifiersExpr
		}

	addModifier:
		if len(group.Elems) == 0 {
			diag.errorOf(x, "empty modifier: %v", x).
				debug(optionDebugErrors, 1)
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
	p.skipSpaces()
	/*rpos := */p.expect(token.RBRACK)
	if len(elems) == 0 {
		diag.errorAt(posLp, "empty modifier group").
			debug(optionDebugErrors, 1)
	}
	if p.tok == token.COLON {
		diag.errorAt(posLp, "unexpected colon after modifer").
			debug(optionDebugErrors, 1)
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
	"@" , "%" , "<" , "?" , "^" , "+" , "|" , "*" ,  //
	"@D", "%D", "<D", "?D", "^D", "+D", "|D", "*D", //
	"@F", "%F", "<F", "?F", "^F", "+F", "|F", "*F", //
	"@'", "%'", "<'", "?'", "^'", "+'", "|'", "*'", //
	"-" , "~" ,
}

func (p *parser) parseRuleEntry(special specialRule, options, targets []Value) (result Value) {
	if p.project.keyword == token.PACKAGE {
		p.error(p.pos, "rules forbidden: %v", targets)
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

	p.params = nil
	p.dialect = ""

	switch special {
	case specialRuleUse:
		scopeComment = fmt.Sprintf(usecomment)
	default:
		scopeComment = fmt.Sprintf("rule %v", targets)
	}

	position := p.position()
	ls := p.openScope(scopeComment)
	for _, s := range automatics {
		var def, alt = p.def(position, s)
		if alt != nil {
			p.error(p.pos, "name `%s' already taken, not automatic (%T).", s, alt)
		} else if def == nil {
			p.error(p.pos, "'%s' is not defined", s)
		} else {
			def.origin = DefAuto
		}
	}
	for i := 1; i < 10; i += 1 {
		var def, alt = p.def(position, strconv.Itoa(i))
		if alt != nil {
			p.error(p.pos, "name `%v` already taken, not numberred (%T).", i, alt)
		} else if def == nil {
			p.error(p.pos, "'$%d' is not defined", i)
		} else {
			def.origin = DefAuto
		}
	}

	switch special {
	case specialRuleUse:
		if name, alt := p.scope.ProjectName(p.project, selfproj, p.project); alt != nil {
			p.error(p.pos, "name `%s` already taken, not automatic (%T)", selfproj, alt)
		} else if name == nil {
			p.error(p.pos, "cannot define `%s` automatic", selfproj)
		}
		if name, alt := p.scope.ProjectName(p.project, userproj, nil); alt != nil {
			p.error(p.pos, "name `%s` already taken, not automatic (%T)", userproj, alt)
		} else if name == nil {
			p.error(p.pos, "cannot define `%s` automatic", userproj)
		}
	}

	// NOTE: expand targets to speed up for later usage, it might spend lots of time in
	// project.entry while matching for entry looked up if not expanded right now.
	var err error
	if true {
		targets, err = ExpandAll(targets...)
		if err != nil { p.error(p.pos, "expand targets '%v' failed: %v", targets, err) }
	} else {
		var ta []Value
		for _, t := range targets {
			if t.expandible(expandClosure) {
				if false { diag.infoOf(t, "target: %T %v", t, t) }
				ta = append(ta, t)
			} else if a, e := ExpandAll(t); e == nil {
				ta = append(ta, a...)
			} else {
				p.error(p.pos, "expand targets '%v' failed: %v", targets, e)
			}
		}
		targets = ta
	}

	defer func(t []Value) { p.targets = t } (p.targets)
	p.next(true) // skip rule delimeters and spaces
	p.targets = targets // save targets for further usage

	if p.tok != token.SEMICOLON && p.tok != token.BAR && !p.isEndOfLine() {
		depends = p.parseDependList()
	}
	if p.tok == token.BAR {
		p.next(true) // '|' starts the ordered prerequisites
		if p.tok != token.SEMICOLON && !p.isEndOfLine() {
			ordered = p.parseDependList()
		}
	}

	if p.tok == token.SEMICOLON { // :;
		// Parse inline recipe in the program scope.
		recipes = append(recipes, p.parseRecipeExpr())
	} else if p.tok == token.LINEND || p.lineComment != nil {
		p.scanner.TrunRecipesOn() // Turn on recipes before LINEND
		p.expectLinend() // Take the new line
		// Parse recipes in the program scope.
		for p.tok != token.EOF && p.tok == token.RECIPE {
			recipes = append(recipes, p.parseRecipeExpr())
		}
		p.scanner.TurnRecipesOff()
	}

	var params []string
	if t := targets[0]; p.configure {
		if name, err := t.Strval(); err != nil {
			diag.errorOf(t, "strval configure target '%v' failed: %v", t, err)
		} else {
			d, a := p.project.scope.define(p.project, name, nil)
			if d == nil && a == nil {
				diag.errorOf(t, "cannot define configure target '%v'", name)
			} else if a != nil {
				if _, ok := a.(*Def); !ok {
					diag.errorOf(t, "configure target '%v' already taken: %T %v", name, a, a)
				}
			}
			if d != nil && !d.position.IsValid() {
				d.position = t.Position()
			}
		}
	} else {
		for _, d := range p.params { params = append(params, d.name) }
	}

	parsedData := &parsedRuleData{
		// TODO: lang: 0,
		params:   params,
		position: position,
		config:   p.configure,
		targets:  p.convertBarefiles(targets),
		depends:  p.convertBarefiles(depends),
		ordered:  p.convertBarefiles(ordered),
		recipes:  recipes,
		options:  options,
		special:  special,
	}

	if special != specialRuleRec {
		var res []*RuleEntry
		if res = p.rule(parsedData); len(res) == 1 {
			result = res[0]
		} else if len(res) > 1 {
			var list = MakeList(res[0].position)
			for _, v := range res { list.Elems = append(list.Elems, v) }
			result = list
		} else {
			result = MakeNil(parsedData.position)
		}
	}

	// Close the rule scope and go back to project scope. The current
	// scope must be project scope befor Rule.
	p.closeScope(ls)
	p.configure = false
	p.dialect = ""
	p.params = nil
	return
}

func (p *parser) parseSpecialRuleClause() Value {
	if t_traverse.enabled {
		defer un(trace(t_traverse, "SpecialRule"))
	}

	p.expect(token.COLON) // expect and skip ':'

	if p.tok != token.BAREWORD {
		p.error(p.pos, "unknown special rule")
		return nil
	}

	var name = p.lit
	switch name {
	case "user":
		if true {
			p.error(p.pos, ":user: rules are deprecated, use using.* instead!")
			return nil
		}

		var options []Value
		var pos = p.expect(token.BAREWORD) // USE
		var bits = p.setbit(parsingSpecialRule)
		// Options are *Flag or *Pair of a Flag.
		for p.tok == token.MINUS {
			opt := p.parseExpr(false)
			options = append(options, opt)
		}
		p.setbits(bits) // restore bits
		if p.tok.IsRuleDelim() {
			return p.parseRuleEntry(specialRuleUse, options, []Value{
				MakeBareword(p.positionAt(pos), name),
			})
		}

		p.error(p.pos, "expecting special rule terminator ':'")
		return nil
	default:
		p.error(p.pos, "unknown special rule")
		return nil
	}
}

func (p *parser) expandForeach(t *template, vars map[string]Value, params []Value, endPos token.Pos) {
	p.scanner.SetState(t.state)
	p.pos, p.tok, p.lit = t.pos, t.tok, t.lit

	// TODO: deal with params
	defer p.closeScope(p.openScope("template expansion ")) // NOTE: comment here will affect loader.def()

	//t_traverse.tracef("%v %v", t_traverse.elapsed(), vars)
	for s, v := range vars {
		var def, alt = p.def(p.position(), s)
		if alt == nil { def.set(DefAuto, v) } else {
			diag.errorAt(p.position(), "variable '%s' already taken", s).
				debug(optionDebugErrors)
		}
	}

	for p.tok != token.EOF && p.pos < endPos {
		switch p.tok {
		case token.LINEND: p.next(true)
		default:
			//t_traverse.tracef("%v %v", t_traverse.elapsed(), p.tok)
			p.parseClause(endPos)
		}
	}
}

func (p *parser) expandTemplate(params []Value, endPos token.Pos) {
	defer func(pos token.Pos, tok token.Token, lit string, state scanner.ScanState) {
		p.pos, p.tok, p.lit	 = pos, tok, lit
		p.scanner.SetState(state)
	}(p.pos, p.tok, p.lit, p.scanner.State())

	// template foreach val1 val2 val3 val4 ...
	// template for name1=(val1 val2 val3 ...) name2=(val1 val2 val3)
	var (
		l = len(p.templates)
		t = p.templates[l-1]
	)
	if false { diag.errorAt(p.position(), "%v", cloctx) }
	switch t.verb {
	case "foreach":
		for _, a := range t.params {
			p.expandForeach(t, map[string]Value{ "_" : a }, params, endPos)
		}
	case "for":
		for _, a := range t.params {
			if pair, ok := a.(*Pair); ok {
				if s, e := pair.Key.Strval(); e != nil {
					diag.errorOf(pair.Key, "expand template %v", e).
						debug(optionDebugErrors)
				} else if g, ok := pair.Value.(*Group); ok {
					for _, v := range g.Elems {
						p.expandForeach(t, map[string]Value{ s : v }, params, endPos)
					}
				} else {
					p.expandForeach(t, map[string]Value{ s : pair.Value }, params, endPos)
				}
			}
		}
	default:
		diag.errorAt(p.position(), "expand template %v: %v", t.verb, params).
			debug(optionDebugErrors)
	}
}

func (p *parser) parseTemplateClause() (end bool) {
	var pos = p.pos
	p.expect(token.TEMPLATE) // expect and skip 'template'
	p.skipSpaces()

	var verb string
	var op = p.parseExpr(false); p.skipSpaces()
	if w, ok := op.(*Bareword); ok { verb = w.string } else {
		diag.errorOf(op, "unknown template verb: %v", op).
			debug(optionDebugErrors)
		return
	}

	var params = p.parseExprList(false); p.expect(token.LINEND)
	if false { defer un(tracef(t_traverse, "parseTemplateClause(%v, %v, %v)", verb, len(params), pos)) }

	switch verb {
	case "expand":
		p.expandTemplate(params, pos)
		end = true
		return
	case "save":
		diag.errorOf(op, "TODO: save template for later usage: ", params).debug(true)
		return
	}

	var err error
	if params, err = mergeresult(ExpandAll(params...)); err != nil {
		diag.errorAt(p.positionAt(pos), "merge params %s failed", err)
		return
	}

	p.templates = append(p.templates, &template{
		state: p.scanner.State(),
		pos: p.pos, tok: p.tok, lit: p.lit,
		verb: verb, params: params,
	})

ForToken:
	for newline := false; p.tok != token.EOF; {
		switch p._next(); p.tok {
		case token.SPACE:
		case token.LINEND: newline = true
		case token.TEMPLATE:
			if !newline { continue ForToken }
			if p.parseTemplateClause() { break ForToken }
		default:
			newline = false
		}
	}
	return
}

func (p *parser) parseClause(endPos token.Pos) {
	if false { defer un(tracef(t_traverse, "parseClause(%v, %v)", p.tok, p.pos)) }
	var position = p.position()
	switch p.tok {
	case token.USE:
		diag.errorAt(position, "`%v` unexpected here", p.tok).debug(optionDebugErrors)
		return
	case token.INCLUDE:
		p.parseGenericClause(token.INCLUDE, p.expect(token.INCLUDE), p.parseIncludeSpec)
		return
	case token.FILES:
		p.parseGenericClause(token.FILES, p.expect(token.FILES), p.parseFilesSpec)
		return
	case token.EVAL:
		p.parseGenericClause(token.EVAL, p.expect(token.EVAL), p.parseEvalSpec)
		return
	case token.COLON:
		p.parseSpecialRuleClause()
		return
	case token.TEMPLATE:
		p.parseTemplateClause()
		return
	}

	if t_traverse.enabled { defer un(trace(t_traverse, "Clause(?)")) }

	var x = p.parseExpr(true); p.skipSpaces()
	if p.tok.IsAssign() {
		p.parseDefine(x)
		return
	}

	var list = []Value{ x }
	if !p.tok.IsRuleDelim() {
		list = append(list, p.parseLhsList()...)
	}
	if p.tok.IsRuleDelim() {
		p.parseRuleEntry(specialRuleNor, nil, list)
		return
	}

	diag.errorAt(p.position(), "bad clause: %v (%s) after %v", p.tok, p.lit, list).
		debug(optionDebugErrors, 6)
}

func parseUsingNameProps(nameprops string) (name string, parts []string, optUnique, optReverse bool) {
	parts = strings.Split(nameprops, ".")
	name = parts[0]
	for _, s := range parts[1:] {
		switch s {
		case "unique", "uniq", "uni": optUnique = true
		case "reverse", "rever", "rev": optReverse = true
		case "unirev": optUnique, optReverse = true, true
		default: name += "." + s // add anything else back to the name
		}
	}
	return
}

func (p *parser) applyUseeVars(position Position, proj *Project, using Value) {
	var userProj = p.scope.project // aka. p.project
	if s, e := using.Strval(); e == nil {
		names := strings.Fields(s)
		for _, nameprops := range names {
			var (
				name, _, optUnique, optReverse = parseUsingNameProps(nameprops)
				usingVarName = fmt.Sprintf("using.%s", name)
				def *Def; alt Object
			)
			def, alt = proj.scope.define(proj, usingVarName, MakeNone(position))
			if def == nil && alt != nil { def, _ = alt.(*Def) }
			for _, base := range proj.bases {
				if obj, err := base.resolveObject(usingVarName); err == nil && !(isNil(obj) || isNone(obj)) {
					def.append(obj)
				}
			}
			if l, e := proj.using.Get(name); e == nil {
				if e = def.append(l); e != nil {
					diag.errorAt(position, "append using value: %v (user=%v)", e, userProj)
				}
				if optUnique {
					var cc = setclosure(cloctx.unshift(proj.scope))
					defer func() { if cc != nil { setclosure(cc) } } ()
					if optReverse {
						var flag = MakeFlag(position, "r")
						def.value = builtinUnique(position, flag, def.value)
					} else {
						def.value = builtinUnique(position, def.value)
					}
					cc = nil
				}
			} else {
				diag.errorAt(position, "%v: %v (using.%s)", proj, e, name)
			}
		}
	} else {
		diag.errorAt(position, "applying usee '%v' (%v) failed: %v", using, proj, e)
	}
}

type projectDeclOpts struct {
	final bool `f,final`
	noDock bool `n,nod;n,nodock;nd,no-dock`
    breakUseLoop bool `b,break;l,loop`  // don't recursively use this project
    multiUseAllowed bool `m,multi`  // this project is used multiple times
}

func (p *parser) parseFile() *parsedFile {
	if  optionTraceLaunch { defer un(trace(t_launch, "parser.parseFile")) }
	if t_traverse.enabled { defer un(trace(t_traverse, "File '"+p.file.Name()+"'")) }
    if false { defer un(tracef(t_traverse, "parseFile(%s)", p.file.Name())) }

	// Don't bother parsing the rest if we had errors scanning the first token.
	// Likely not a Go source file at all.
	if diag.numErrors() > 0 { return nil }

	var (
		abs, rel, tmp string
		ident *Barecomp //Bareword
		identStr string
		implicitBase string // aka. foo.bar.Baz implicitly load base 'foo/bar'
		keyword = p.tok
		filename = p.file.Name()
		position = p.position()
		ls = p.openScope(fmt.Sprintf("file %s", filename))
	)
	/*if filename == confinitFilename {
                abs, rel = context.workdir, "."
                tmp = joinTmpPath(context.workdir, rel)
        } else*/ {
		if p.mode&Flat != 0 {
			abs = p.project.absPath
		} else {
			abs = filepath.Dir(filename)
		}
		rel, _ = filepath.Rel(p.workdir, abs)
		tmp = joinTmpPath(p.workdir, rel)
	}

	if ls.scope != nil {
		defer p.closeScope(ls)
		var ( def *Def ; s = ls.scope )
		if p.mode&Flat == 0 {
			def, _ = p.def(position, ".")
			def.set(DefAuto, MakePathStr(position, rel))

			def, _ = p.def(position, "/")
			def.set(DefAuto, MakePathStr(position, abs))

			def, _ = p.def(position, "CTD") // Current Temp Directory
			def.set(DefAuto, MakePathStr(position, tmp))

			def, _ = p.def(position, "CWD") // Current Work Directory
			def.set(DefAuto, MakePathStr(position, abs))
		} else if def = s.FindDef("/"); def == nil {
			diag.errorAt(position, "/ not in the scope: %v", s.comment)
		} else if def = s.FindDef("."); def == nil {
			diag.errorAt(position, ". not in the scope: %v", s.comment)
		} else if def = s.FindDef("CTD"); def == nil {
			diag.errorAt(position, "CTD not in the scope: %v", s.comment)
		} else if def = s.FindDef("CWD"); def == nil {
			diag.errorAt(position, "CWD not in the scope: %v", s.comment)
		}
	} else {
		diag.errorAt(position, "opened invalid scope for %s", filename).
			debug(optionDebugErrors, 1)
		return nil
	}

	switch keyword {
	case token.CONFIGURE:
		switch p.next(true); p.tok {
		case token.DOT:
			if err := p.ParseConfigDir(abs, abs); err != nil {
				p.error(p.pos, "parsing configure directory failed, '%s': %v", abs, err)
			} else {
				p.next(true) // skip the '.' token and consequence spaces
			}

			basename := filepath.Base(filepath.Dir(filename))
			ident = MakeBarecomp(position, MakeBareword(position, basename))

		default:
			p.error(p.pos, "unknown configuration '%v', currently only 'configure .' is supported", p.tok)
		}
	case token.PROJECT, token.PACKAGE, token.MODULE:
		if p.mode&Flat != 0 {
			p.error(p.pos, "forbidden `%v` in flat file", p.tok)
		}

		p.next(true)

		// Options are *Flag or *Pair of a Flag.
		var ( opts projectDeclOpts; options []Value; pos Position )
		for p.tok == token.MINUS {
			var opt = p.parseExpr(false);  p.skipSpaces()
			options = append(options, opt)
			if !pos.IsValid() { pos = opt.Position() }
		}
		if !pos.IsValid() { pos = p.position() }
		if a, e := parseOpts(pos, &opts, options...); e != nil {
			diag.errorAt(pos, "parse project decl opts failed: %v", e).
				debug(optionDebugErrors, 1)
			return nil
		} else if len(a) > 0 {
			for _, v := range a {
				diag.errorOf(v, "unknown option '%v'", v).
					debug(optionDebugErrors, 1)
			}
			return nil
		}

		var linfo = p.loads[len(p.loads)-1]

		// Smart-lang spec:
		//   * the project clause is not a declaration;
		//   * the project name does not appear in any scope.
		if p.tok == token.LPAREN || p.tok == token.LINEND || p.lineComment != nil {
			var (
				dir = filepath.Dir(filename)
				base = filepath.Base(dir)
			)
			if linfo.loadee != nil && linfo.absDir == dir {
				ident = MakeBarecomp(position, MakeBareword(position, linfo.loadee.name))
			} else {
				// TODO: validate basename as a valid identifier
				ident = MakeBarecomp(position, MakeBareword(position, base))
			}
		} else if p.tok == token.TILDE {
			/*if filename == confinitFilename {
                ident = &ast.Bareword{ ValuePos:pos, Value:"~" }
            } else*/ if ext := filepath.Ext(filename); ext != ".smart" {
				diag.errorAt(p.position(), "`%v` not a smart file", filepath.Base(filename)).
					debug(optionDebugErrors, 1)
			} else if s := strings.TrimSuffix(filepath.Base(filename), ext); s != "" {
				ident = MakeBarecomp(position, MakeBareword(position, s))
			} else {
				diag.errorAt(p.position(), "`%v` not tilde name", filepath.Base(filename)).
					debug(optionDebugErrors, 1)
			}
			p.next(true) // skip tilde
		} else {
			var implicitBaseSegs []string
			ident = MakeBarecomp(p.position())
		ForProjectName:
			for p.tok != token.EOF && p.tok != token.SPACE {
				if w := p.parseBarewordConstant(false); w == nil {
					diag.errorAt(ident.Position(), "expecting a bareword").
						debug(optionDebugErrors, 1)
				} else if word, ok := w.(*Bareword); !ok {
					diag.errorAt(ident.Position(), "expecting a bareword: %v (%T)", w, w).
						debug(optionDebugErrors, 1)
				} else if ident.Combine(word); p.tok == token.DOT {
					ident.Combine(MakeBareword(p.position(), ".")) // TODO: parse to Qualiword
					implicitBaseSegs = append(implicitBaseSegs, word.string)
					p._next() // '.'
				} else { break ForProjectName }
			}
			p.skipSpaces()
			if len(ident.Elems) == 0 {
				diag.errorAt(p.position(), "package name is empty").
					debug(optionDebugErrors, 1)
				return nil
			} else if len(implicitBaseSegs) > 0 {
				implicitBase = filepath.Join(implicitBaseSegs...)
			}
		}

		var err error
		if identStr, err = ident.Strval(); err != nil {
			diag.errorAt(ident.Position(), "strval '%v' failed: %v", ident, err).
				debug(optionDebugErrors, 1)
			return nil
		} else if linfo.loadee != nil && identStr != linfo.loadee.name {
			diag.warnAt(ident.position, "declare multiple project in the same directory").
				debug(optionDebugErrors, 1)
		} else if identStr == "_" && p.mode&DeclarationErrors != 0 {
			diag.errorAt(ident.Position(), "package name '_' is preserved").
				debug(optionDebugErrors, 1)
			return nil
		}

		// Don't bother parsing the rest if we had errors parsing the package clause.
		// Likely not a Go source file at all.
		if n := diag.numErrors(); n > 0 {
			diag.errorAt(p.position(), "got %d errors parsing file: %s", filename).
				debug(optionDebugErrors, 1)
			return nil
		}

		var _, declared = linfo.declares[identStr]
		if (p.mode&Flat == 0) && p.declare(keyword, ident, identStr, options) {
			// Change the 'default' owners into the new declared project
			if s := ls.scope; s != nil {
				if def := s.FindDef("."  ); def != nil { def.owner = p.project }
				if def := s.FindDef("/"  ); def != nil { def.owner = p.project }
				if def := s.FindDef("CTD"); def != nil { def.owner = p.project }
				if def := s.FindDef("CWD"); def != nil { def.owner = p.project }
			} else {
				diag.errorAt(position, "file scope is nil").
					debug(optionDebugErrors, 1)
			}
			if linfo.loadee == nil {
				// NOTE: build.smart is always the first loaded, so the loadee will be pointed to it
				linfo.loadee = p.project
			}
			defer func(proj *Project) {
				if filepath.Base(filename) == "build.smart" {
					var using Value
					if o, e := proj.resolveObject("using.*"); e == nil && !isNil(o) {
						if def, ok := o.(*Def); ok && !isNil(def) { using = def.value }
					}
					if !isNil(using) { p.applyUseeVars(ident.Position(), proj, using) }
				}
				p.isLoadingBases = false
				p.closeCurrent(ident, identStr)
			} (p.project)
		}

		var basePos Position
		if implicitBase != "" { basePos = pos } else { basePos = p.position() }
		if p.tok == token.LPAREN {
			p.isLoadingBases = true
			for p.tok != token.EOF {
				for p.next(true); !p.isEndOfList(false); {
					p.skipSpaces()
					param := p.parseExpr(false)
					p.skipSpaces()
					//if p.lineComment != nil  { break }
					//if p.tok == token.LINEND { break }
					if p.tok == token.EOF {
						diag.errorAt(basePos, "unexpected end of file while parsing bases").
							debug(optionDebugErrors, 1)
						return nil
					}
					if t, e := parseOpts(param.Position(), &opts, param); e != nil {
						diag.errorOf(param, "parse opt '%v' failed: %v", param, e).
							debug(optionDebugErrors, 1)
						return nil
					} else if keyword == token.PACKAGE || opts.final {
						// No bases for PACKAGE or final project
					} else if !p.loadBases(basePos, linfo, /*implicitBase*/"", merge(t...)...) {
						diag.errorOf(param, "loading base '%v' failed", t).
							debug(optionDebugErrors, 1)
						return nil
					}
				}
				if p.tok != token.COMMA { break }
			}
			p.isLoadingBases = false
			p.expect(token.RPAREN)
		} else if !p.loadBases(basePos, linfo, implicitBase) { // for special bases, e.g. .base
			diag.errorAt(basePos, "loading bases failed").
				debug(optionDebugErrors, 1)
			return nil
		}
		p.expectLinend()
		if keyword != token.PACKAGE {
			p.loadProjectConfiguration(ident, identStr, declared)
			if !opts.noDock { p.loadProjectContainer(ident, identStr) }
		}
	default:
		if p.mode&Flat == 0 {
			p.expected(p.pos, "configure, project, module or package keyword")
		}
	}

	if p.mode&ModuleClauseOnly == 0 {
		if p.mode&Flat == 0 {
		ForInit:
			for p.tok != token.EOF {
				switch p.tok {
				case token.IMPORT:
					p.expected(p.pos, "`use`, keyword `import` is replaced by `use`")
				case token.LINEND:
					p.next(true) // skip empty lines
				case token.USE:
					p.parseGenericClause(p.tok, p.expect(p.tok), p.parseUseSpec)
				case token.EVAL:
					p.parseGenericClause(p.tok, p.expect(token.EVAL), p.parseEvalSpec)
				default:
					if p.tok.IsKeyword() { break ForInit }
					var x = p.parseExpr(true); p.skipSpaces()
					if p.tok.IsAssign() { p.parseDefine(x) } else
					if p.tok.IsRuleDelim() {
						if p.project == nil {
							p.error(p.pos, "no project declared before defining rules")
						} else {
							x = p.parseRuleEntry(specialRuleNor, nil, []Value{x})
						}
						break ForInit
					} else {
						p.error(p.pos, "unexpected %v (after %v)", p.tok, x)
					}
				}
			}
		}
		if p.mode&ImportsOnly == 0 {
			// rest of module body
			for p.tok != token.EOF {
				switch p.tok {
				case token.LINEND:
					p.next(true) // skip empty lines
				default:
					p.parseClause(token.NoPos)
				}
			}
		}
	}

	return &parsedFile{
		// TODO: doc: doc,
		// TODO: comments: p.comments,
		keyword:    keyword,
		position:   position,
		name:       ident,
		scope:      ls.scope,
		using:      p.imports,
	}
}
