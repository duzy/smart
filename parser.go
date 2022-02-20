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
	"sync"
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
func (p *parser) inner() Context { return p.loader }
func (p *parser) String() string { return fmt.Sprintf("parser{%s}", p.loader) }

func (p *parser) init(l *loader, filename string, src []byte) {
	p.loader = l
	p.file = l.fset.AddFile(filename, -1, len(src))

	var m scanner.Mode
	if p.mode&ParseComments != 0 {
		//m = scanner.ScanComments
	}

	eh := func(pos token.Position, msg string) {
		erro(p, "%s", msg).at(Position(pos)).debug(1)
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

func (p *parser) trace(a ...interface{}) { t_traverse.traceAt(p.Position(), a...) }

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
		erro(p, "unexpected end of file").at(p.positionAt(pos)).debug(1)
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

	comment = &Comment{ Pos: p.Position(), Text: p.lit }
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
					case token.RECIPE: // TODO: using p.isRecipeStart()
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

func (p *parser) Position() Position { return p.positionAt(p.pos) }
func (p *parser) positionAt(pos token.Pos) Position { return Position(p.file.Position(pos)) }

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
	erro(p, msg).at(p.positionAt(pos)).debug(16)
}

func (p *parser) expect(tok token.Token) token.Pos {
	pos := p.pos
	if p.tok != tok {
		p.expected(pos, "'"+tok.String()+"'")
	}
	p._next() // make progress
	return pos
}

func (p *parser) expectLinend() (ok bool) {
	if p.lineComment != nil {
		// The line comment is treated as LINEND, simply ignore it.
		p.lineComment, ok = nil, true
	} else if p.tok == token.LINEND {
		p._next(); ok = true
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
			res = token.Pos(p.file.Base() + p.file.Size()) // EOF position
		}
	}()
	_ = p.file.Offset(pos) // trigger a panic if position is out-of-range
	return pos
}*/

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

	position, tok := p.Position(), p.tok // the arrow '->' or '=>'
	p._next() // skip '->' or '=>'

	var (
		proj = p.Project()
		okay bool
	)
	switch t := lhs.(type) {
	case *selection:
		if v, err := t.value(positional(p, t.Position())); err != nil {
			erro(p, "%v: selection failed: %v", lhs, err).at(position).debug(1)
			return
		} else if isNil(v) {
			erro(p, "nil selection: %v", lhs).at(position).debug(1)
			return
		} else {
			lhs = v
		}
	case *Bareword:
        switch t.string {
        case "usee": lhs = proj.using
        case "self": lhs = proj.self
        case "goals", "os", "mode":
			if lhs, okay = p.colonResolve(t.string); !okay {
				erro(p, `"%s" not defined`, t.string).debug(1)
				return
			}
        default:
            if name, o, err := p.resolveObject(lhs); err != nil {
				erro(p, "resolve '%v' failed: %v", lhs, err).at(lhs.Position())
				erro(p, "parser is here (tok=%s)", tok).at(position)
				erro(p, "parser to go here (tok=%s, lit=%s)", p.tok, p.lit).at(p.Position()).debug(8)
                return
            } else if !isNil(o) {
				lhs = o
			} else if tok == token.SELECT_PROG2 {
				res = MakeNil(position) // ignore
				return
			} else {
				erro(p, "%v: '%v' is undefined (name=%v, obj=%v)", proj, lhs, name, o).at(lhs.Position())
				erro(p, "%v: parser is here (name=%s, tok=%s)", proj, t.string, tok).at(position)
				erro(p, "%v: parser to go here (tok=%s, lit=%s)", proj, p.tok, p.lit).at(p.Position()).debug(16)
				return
            }
        }
    case *Barecomp: // for cases like '.foo'
        if name, o, err := p.resolveObject(t); err != nil {
			erro(p, "resolve selection object '%v' (%s) error: %v", lhs, name, err).of(lhs).debug(1)
			return
        } else if !isNil(o) {
			lhs = o
		} else if tok == token.SELECT_PROG2 {
			res = MakeNil(position) // ignore
			return
		} else {
			erro(p, "'%v' is undefined", lhs).of(lhs).debug(1)
			return
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
	if (p.bits&parsingRecipeText != 0) && p.tok == token.RECIPE { // TODO: using p.isRecipeStart()
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
			erro(p, "unexpected colon").at(p.Position()).debug(1)
			p.next(true) // just ignore this colon
		} else {
			p.skipSpaces()
			list = append(list, p.parseExpr(false))
			if p.tok == token.SPACE { p.next(true) } //p.skipSpaces()
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
	return MakeList(p.Position(), p.parseExprList(lhs)...)
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

	position := p.Position()
	p.next(true)
	elems, converted := p.parseExprList(false), false
	for /*p.tok == token.COMMA*/p.tok != token.RPAREN && p.tok != token.EOF {
		//p.next(true) // skip token.COMMA
		switch p.tok {
		case token.BAR, token.COMMA, token.SEMICOLON:
			elems = append(elems, p.parsePunctuation())
			p.skipSpaces()
		}
		var next *List
		next = p.parseListExpr(false)
		if !converted {
			elems = []Value{ MakeList(p.Position(), elems...), next }
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
	for /*p.tok == token.COMMA*/p.tok != token.RPAREN && p.tok != token.EOF {
		//p.next(true) // skip token.COMMA
		switch p.tok {
		case token.COMMA: p.next(true) // skip token.COMMA
		case token.BAR, token.SEMICOLON:
			if false {
				a = append(a, p.parsePunctuation())
				p.skipSpaces()
			} else {
				erro(p, "unexpected punctuation: %v", p.tok).debug(1)
			}
		}
		a = append(a, p.parseListExpr(false))
	}
	p.expect(token.RPAREN)
	return MakeArgumented(x, a...)
}

func (p *parser) parseGlobMeta() (x *GlobMeta) {
	pos, tok := p.Position(), p.tok
	p._next()
	return MakeGlobMeta(pos, tok)
}

func (p *parser) parseGlobRange() (x *GlobRange) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Glob")) }

	position := p.Position()
	p.expect(token.LBRACK) // skip '['

	chars := p.parseExpr(false)
	p.expect(token.RBRACK) // skip ']'

	return MakeGlobRange(position, chars)
}

func (p *parser) parseGlobExpr(x Value) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "Glob")) }

	var (
		pos = p.Position()
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
		erro(p, "nil glob expression (tok=%v, lit=%v)", p.tok, p.lit).at(pos)
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
			position := p.Position()
			perc2 := MakePercPattern(position, nil, nil)
			if pos+2 == p.pos {
				switch p.tok {
				case token.PERC: // %%%
					erro(p, "too many %")
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
					if ok { erro(p, "incorrect: %v, %v", x, yy).at(position) }
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

	erro(p, "todo: regexp")

	return
}

func (p *parser) parseKeyValueExpr(x Value) *Pair {
	if t_traverse.enabled { defer un(trace(t_traverse, "Pair")) }
	position := p.Position(); p._next()
	return MakePair(position, x, p.parseExpr(false))
}

func (p *parser) parseFlagExpr(lhs bool) *Flag {
	if t_traverse.enabled { defer un(trace(t_traverse, "Flag")) }

	var (
		position = p.Position()
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

func (p *parser) parsePunctuation() *Punctuation {
	if t_traverse.enabled { defer un(trace(t_traverse, "punctuation")) }
	var tok = p.tok
	p._next()
	return &Punctuation{valbase{p.Position()}, tok}
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
    case token.BAR: erro(p, "`|` is deprecated, changed the modifiers!").at(p.positionAt(pos))
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
			erro(p, "unexpected end of line for compound string")
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
		comp = MakeBarecomp(x.Position())//(p.Position())
		comp.Elems = append(comp.Elems, x)
	}

	for /*comp.End() == p.pos && */!p.isEndOfDotConcat(lhs) {
		comp.Combine(p, p.parseComposedExpr(false))
		if p.tok == token.DOT /*&& comp.End() == p.pos*/ {
			var dot = MakeBareword(p.Position(), ".") // TODO: parse to Qualiword instead
			comp.Elems = append(comp.Elems, dot)
			p._next() // '.'
		}
	}

	// FIXME: *.o => obj
	//   BUG: Barecomp{Glob . KeyValueExpr}
	//   FIX: KeyValueExpr{Barecomp, Bareword}

	return comp
}

func makePathSeg(ctx Context, tok token.Token) *PathSeg {
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
	return MakePathSeg(ctx.Position(), r)
}

func (p *parser) parsePathExpr(lhs bool, start Value) *Path {
	if t_traverse.enabled { defer un(trace(t_traverse, "Path")) }

	defer p.setbits(p.setbit(composingPATH))

	var (
		position = start.Position() //p.Position()
		path *Path
		ok bool
	)
	if start == nil {
		erro(p, "bad closure/delegate name").at(position).debug(1)
		p._next()
		return MakePath(position) // empty path
	} else if path, ok = start.(*Path); !ok {
		path = MakePath(position, start)
	}

BuildPath:
	for p.tok == token.PCON {
		var pos = p.Position() // skips repeated '/' sequence
		for p._next(); p.tok == token.PCON; p._next() { pos = p.Position() }
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
			erro(p, "TODO: URL path: %v (%T) (next: %s (%s))", scheme, scheme,  p.tok, p.lit).
				at(p.Position()).debug(1)
			res = MakeNil(p.Position())
			return
		}
	} else if !p.isEndOfURL(lhs) {
		erro(p, "TODO: URL: %v (%T) (next: %s (%s))", scheme, scheme,  p.tok, p.lit).
			at(p.positionAt(colon1)).debug(1)
		res = MakeNil(p.Position())
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
		url.Path = p.parsePathExpr(lhs, makePathSeg(p, p.tok))
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

func (p *parser) parseClosureDelegate() (result Value) {
	if t_traverse.enabled {	defer un(trace(t_traverse, "ClosureDelegate")) }

	// FIXME: push p.bits before entering a $(...) or &(...)
	defer func(a parsingBits) { p.bits = a } (p.bits)
	p.bits = 0 // start with zero bits

	var (
		ctx = positional(p, p.Position())
		pos = p.pos
		tok = p.tok
		resolved Value // Object or *selection
		rest []Value
	)

	resolveConfig := func(val Value, name string) (obj Object) {
		if p.Project().configure != nil {
			var err error
			if obj, err = p.Project().configure.resolveObject(ctx, name); err != nil {
				erro(p, "resolve configure '%s' failed: %v", name, err).at(val.Position()).debug(4)
			}
		}
		return
	}

	resolveObject := func(lPos Position, lTok token.Token, name Value) (str string, obj Value, okay bool) {
		if false { defer func() {
			if isNil(obj) /*&& !isNil(resolved)*/ {
				warn(p, "nil: %v (tok=%v%v, resolved=%T %v)", name, tok, lTok, resolved, resolved).at(name.Position()).debug(6)
			}
		} () }
		var proj = p.Project()
		var err error
		switch lTok {
		case token.LPAREN:
			if sel, ok := name.(*selection); ok {
				if sel == nil {
					erro(p, "nil selection: %v", name).at(name.Position()).debug(1)
				} else if o, err := sel.object(ctx); err == nil && o.DeclScope().comment == usecomment {
					obj, okay = unresolved(proj, name), true
				} else if err != nil {
					erro(p, "`%v` invalid delegate selection", name).of(sel).debug(1)
				} else if isNil(o) {
					erro(p, "`%v` nil selection object", name).of(sel).debug(1)
				} else if v, err := sel.value(ctx); err != nil {
					erro(p, "`%v` invalid delegate selection", name).of(name).debug(1)
				} else if isNil(v) {
					erro(p, "`%v` not found in %v", sel.s, o).of(name).debug(1)
				} else if obj, okay = v.(Object); !okay {
					return // just use the selected value
				}
			} else if str, resolved, err = p.resolveObject(name); err != nil {
				erro(p, "resolve '%v' (%s) failed: %v", name, str, err).at(name.Position()).debug(1)
			} else if str == "" {
				erro(p, "name '%v' is empty", name).at(name.Position()).debug(1)
			} else if isNil(resolved) {
				if p.isIncludingConf {
					// Create an empty Def if it's referred in configuration.sm.
					def, _ := p.def(name.Position(), str)
					def.origin = DefConfRef
					obj, okay = def, true
					return
				} else if obj = resolveConfig(name, str); !isNil(obj) {
					okay = true
					return
				} else if tok.IsClosure() || refdef(ctx, name, defany) || name.expandible(ctx, expandClosure) {
					obj, okay = unresolved(proj, name), true // recursive delegation or closure
					return
				} else if name.expandible(ctx, expandPlainValue) {
					erro(p, "%v: resolved '%v' (aka. %s) is nil", proj, name, str).of(name)
					errostack(p, 5, "%v: %v", proj, ctx).of(name).debug(16)
				} else {
					erro(p, "%v: resolved '%v' is nil", proj, name).of(name)
					errostack(p, 5, "%v: %v", proj, ctx).of(name).debug(16)
				}
			} else if sel, ok = resolved.(*selection); ok && sel != nil {
				obj, okay = sel, true
				return
			} else if caller, _ := resolved.(Caller); caller == nil {
				erro(p, "resolved '%v' is not callable: %T", name, resolved).at(lPos).debug(16)
			} else if obj, okay = caller.(Object); isNil(obj) || !okay {
				erro(p, "resolved '%v' is not object: %T", name, resolved).at(lPos).debug(16)
			} else if isNil(obj) {
				erro(p, "resolved '%v' is nil: %T", name, resolved).at(lPos).debug(16)
			} else {
				return
			}
		case token.LBRACE:
			if resolved, err = p.resolveEntries(name); err != nil {
				erro(p, "finding rule entry '%v' failed: %v", name, err).at(lPos).debug(1)
			} else if isNil(resolved) {
				if name.expandible(ctx, expandPlainValue) {
					s, _ := name.Strval(ctx)
					erro(p, "resolved '%v' (aka. %s) is nil (project=%v)", name, s, proj).of(name).debug(1)
				} else {
					erro(p, "resolved '%v' is nil (project=%v)", name, proj).of(name).debug(1)
				}
			} else if exe, _ := resolved.(Executer); exe == nil {
				erro(p, "resolved '%v' of '%T' is not Executer", name, resolved).at(lPos).debug(1)
			} else if obj, okay = exe.(Object); !okay || isNil(obj) {
				erro(p, "resolved Executer '%v' of '%T' is not Object", name, resolved).at(lPos).debug(1)
			}
		case token.LCOLON:
			if str, err = name.Strval(ctx); err != nil {
				erro(p, "error strval name '%v': %v", name, err).at(lPos).debug(1)
				return
			}
			switch str {
			case "usee":  resolved = proj.using // TODO: move usee and self into ctx
			case "self":  resolved = proj.self
			default: if o, found := ctx.colonResolve(str); found { resolved = o } else {
				erro(p, "unknown special property: %v", str, err).at(lPos).debug(1)
				return
			}}
			obj, okay = resolved, true
			return
		}
		return
	}

	var (
		name Value
		nameStr string
		tokLp token.Token
		obj Value
		okay bool
	)
	switch p._next(); p.tok {
	case token.LPAREN, token.LBRACE, token.LCOLON: // $(...), ${...}, $:...:
		posLp := p.Position()
		tokLp  = p.tok
		p._next() // skips LPAREN, LBRACE, LCOLON
		posName := p.Position()
		if /*posLp+1 != p.pos*/p.tok == token.SPACE {
			erro(p, "unexpected spaces").at(posName).debug(1)
			return MakeNil(posName)
		} else if name = p.parseExpr(false); isNil(name) {
			erro(p, "%v: parsed name is nil", p.Project()).at(posName).debug(1)
		} else if name.expandible(ctx, expandClosure) {
			erro(p, "%v: name '%v' (%T) is closured", p.Project(), name, name).at(posName).debug(1)
		} else if nameStr, obj, okay = resolveObject(posLp, tokLp, name); !okay {
			erro(p, "%v: name '%v' is unidentified", p.Project(), name).at(posName).debug(1)
		}

		if  (tokLp == token.LPAREN && p.tok != token.RPAREN) ||
			(tokLp == token.LBRACE && p.tok != token.RBRACE) ||
			(tokLp == token.LCOLON && p.tok != token.RCOLON) {
			for rest = append(rest, p.parseListExpr(false)); p.tok == token.COMMA; {
				p.next(true)
				rest = append(rest, p.parseListExpr(false))
			}
		}

		switch tokLp {
		case token.LPAREN: p.expect(token.RPAREN)
		case token.LBRACE: p.expect(token.RBRACE)
		case token.LCOLON: p.expect(token.RCOLON)
			if p.tok == token.ASSIGN { erro(p, "unexpected assignment").at(p.Position()).debug(1) }
		}

	default:
		if position := p.Position(); tok != token.CLOSURE { // $(...), disabled $name.
			// &(...), &{...}, &'...', &"..."
			erro(p, "expects `%v` or `%v` or quotes", token.LPAREN, token.LBRACE).at(position).debug(1)
			return MakeNil(position)
		} else if p.tok == token.STRING || p.tok == token.COMPOUND {
			var posLp = p.Position()
			tokLp = p.tok

			// &'xxxx' or &"xxxx"
			if name = p.parseExpr(false); isNil(name) {
				erro(p, "parsed name is nil").at(posLp).debug(1)
			} else if name.expandible(ctx, expandClosure) {
				erro(p, "name '%v' (%T) is closured (project=%v)", name, name, p.Project()).at(name.Position()).debug(1)
			} else if nameStr, obj, okay = resolveObject(posLp, tokLp, name); !okay {
				erro(p, "name '%v' is unidentified", name).at(name.Position()).debug(1)
			}
		} else {
			// &(...), &{...}, &'...', &"..."
			erro(p, "expects `%v`, `%v` or quotes", token.LPAREN, token.LBRACE).at(position).debug(1)
			return MakeNil(position)
		}
	}
	if isNil(obj) && p.Project().plugin != nil && p.Project().pluginScope != nil {
		var err error
		if nameStr == "" && !isNil(name) {
			if nameStr, err = name.Strval(ctx); err != nil {
				erro(p, "strval name '%v' failed: %v", name, err).at(name.Position()).debug(1)
				return
			}
		}
		if nameStr == "" {
			erro(p, "strval name '%v' is empty", name).at(name.Position()).debug(1)
		} else {
			obj = p.Project().pluginScope.Lookup(nameStr)
		}
	}
	if position := p.positionAt(pos); tok.IsDelegate() {
		if isNil(obj) {
			erro(p, "resolved '%v' is nil (%T %v, tok=%v)", name, resolved, resolved, tok).at(name.Position()).debug(1)
		}
		return MakeDelegate(position, tokLp, obj, rest...);
	} else {
		if isNil(obj) {
			erro(p, "resolved '%v' is nil (%T %v), shall be 'unresolved' (tok=%v)", name, resolved, resolved, tok).at(name.Position()).debug(1)
		}
		return MakeClosure(position, tokLp, obj, rest...);
	}
}

func (p *parser) parseSpecialClosureDelegate(lhs bool) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "SpecialClosureDelegate")) }

	var pos, tok, s = p.pos, p.tok, p.lit//p.tok.String()[1:]
	p._next()

	var (
		position = p.positionAt(pos)
		name = MakeBareword(position, s)
		nameStr, resolved, err = p.resolveObject(name)
		obj Object
	)
	if err != nil {
		erro(p, "resolve '%v' failed: %v", name, err).of(name).debug(6)
		return MakeNil(position)
	} else if resolved == nil {
		erro(p, "'%v' is undefined", name).of(name).debug(6)
		return MakeNil(position)
	} else if nameStr == "" {
		erro(p, "'%v' is empty", name).of(name).debug(6)
		return MakeNil(position)
	} else if def, ok := resolved.(Caller); def == nil || !ok {
		erro(p, "'%v' is not callable: %T", name, resolved).of(resolved).debug(6)
		return MakeNil(position)
	} else if obj, ok = def.(Object); obj == nil || !ok {
		erro(p, "'%v' is not object: %T", name, def).of(resolved).debug(6)
		return MakeNil(position)
	}

	if isNil(obj) {
		erro(p, "resolved '%v' is <nil>: %v (%T)", name, resolved, resolved).of(resolved).debug(1)
		return MakeNil(position)
	}  else if tok.IsDelegate() {
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
			return p.parsePathExpr(lhs, makePathSeg(positional(p, position), tok))
		} else if tok == token.DOT { // TODO: parse to Qualiword instead
			if x = MakeBareword(p.positionAt(pos), str); p.bits&composingDOT == 0 {
				x = p.parseDotExpr(lhs, x)
			}
			return
		} else if tok == token.TILDE { // TODO: ~user
			return makePathSeg(positional(p, position), tok)
		} else {
			erro(p, "unexpected path segment").at(position).debug(1)
			return MakeNil(position)
		}

	case token.PCON: // The root of the path
		return p.parsePathExpr(lhs, makePathSeg(p, p.tok))

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

	case token.SEMICOLON, token.BAR:
		return p.parsePunctuation()

	default:
		if p.tok.IsClosure() || p.tok.IsDelegate() {
			return p.parseSpecialClosureDelegate(lhs)
		} else if p.tok.IsKeyword() { // keywords here are barewords
			return p.parseBarewordConstant(lhs)
		}
	}

	prompt(p, "%v: bad unary '%v' (lit=%s,lhs=%v)\n", p.file.Name(), p.tok, p.lit, lhs)
	erro(p, "bad unary expression '%v'", p.tok).debug(32)
	p._next() // go to the next token
	return MakeNil(p.Position())
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
			erro(p, "composing modifiers is ignored (unimplemented yet)").of(m)
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
			if s, _ := x.Strval(positional(p, p.Position())); isKnownURLScheme(s) {
				x = p.parseURLExpr(lhs, x)
			}
		}
	}
	return
}

func (p *parser) parseText() (res []Value) {
	if false && t_traverse.enabled { defer un(trace(t_traverse, "Text")) }
	for p.tok != token.EOF {
		if p.tok == token.SPACE { p.next(true) } else {
			res = append(res, p.parseExpr(false))
			if p.checkErrors(true) > 0 {
				warn(p, "parse text got %d errors", p.totalErrors()).debug(16)
				if options.failOnErrors { fail(p.Position(), "fail by %d errors", p.totalErrors()) }
			}
		}
	}
	return
}

func (p *parser) parseExpr(lhs bool) (x Value) {
	if false && t_traverse.enabled { defer un(trace(t_traverse, "Expression")) }

	var tok, lit = p.tok, p.lit
	if x = p.parseComposedExpr(lhs); isNil(x) {
		erro(p, "%v: invalid expression (tok=%v, lit=%v)", p.Project(), tok, lit).debug(6)
		return
	} else if lhs && p.tok.IsAssign() { return }

SwitchCompose:
	switch p.tok {
	case token.ASSIGN: // Example: '*.o = obj'
		if !lhs && p.bits&composingNoPair == 0 {
			x = p.parseKeyValueExpr(x)
		}
		return

	case token.SELECT_PROP, token.SELECT_PROG1, token.SELECT_PROG2:
		if p.bits&composingNoSelect == 0 {
			x = p.parseSelect(x)
			goto SwitchCompose // For example: foobar⇒run(-gen)
		}
		return

	case token.LPAREN:
		if p.bits&composingNoArg == 0 {
			if false {
				if _, ok := x.(*Argumented); ok { erro(p, "nested argumentation") }
			}
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
		token.RAW, token.SPACE, token.LINEND, token.EOF:
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
			warn(p, "barecomp a path: %v (%T), %v (%T) (next=%v)", x, x, y, y, p.tok).of(y).debug(1)
		}
	}

	// Further composing
	switch t := x.(type) {
	case *Barecomp: t.Combine(p, y)
	case *Path: t.Combine(p, y)
		if false { info(p, "%v (%v) (tok=%v)", t, y, p.tok).at(t.position) }
	default:
		comp := MakeBarecomp(x.Position(), x)
		comp.Combine(p, y)
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

func (p *parser) _parseUseSpecProps(props []Value) (opts useOpts, params []Value, err error) {
    // Supported parameter forms:
    //      -param
    //      -param(value)
    //      -param=value
	var ctx = positional(p, p.Position())
    var useList []Value // TODO: apply useList
    for _, prop := range props {
        var s string
        switch t := prop.(type) {
        case *Flag:
            if s, err = t.name.Strval(ctx); err != nil {
                erro(p, "invalid flag `%v` (%v)", prop, err).of(prop)
                return
            }
            switch s {
            //case "nouse", "unuse": opts.unuse = true
            case "reuse": opts.reuse = true
            default: params = append(params, prop)
            }
        case *Pair: // -param=value
            switch tt := t.Key.(type) {
            case *Flag:
                if s, err = tt.name.Strval(ctx); err != nil {
                    erro(p, "invalid flag name `%v` (%v)", tt.name, err).of(t.Key)
                    return
                }
                switch s {
                case "use": useList = append(useList, t.Value)
                default: params = append(params, prop)
                }
            default:
                erro(p, "parameter `%v' unsupported `%T`", prop, prop).of(t.Key)
                return
            }
        case *Argumented: // -param(value)
            switch tt := t.value.(type) {
            case *Flag:
                if s, err = tt.name.Strval(ctx); err != nil {
                    erro(p, "invalid flag name `%v` (%v)", tt.name, err).of(t.value)
                    return
                }
                switch s {
                case "use": useList = append(useList, t.args...)
                default: params = append(params, prop)
                }
            default:
                erro(p, "parameter `%v' unsupported `%T`", prop, prop).of(t.value)
                return
            }
        default:
            erro(p, "parameter `%v` unsupported `%T`", prop, prop).of(prop)
            return
        }
    }
    return
}

func (p *parser) parseUseSpec(doc *CommentGroup, generic *genericoptions, _ int) {
	var props = p.parseDirectiveSpec()
	if p.imports = append(p.imports, &usespec{ props }); generic.dontOperate {
		// TODO: maybe give some information
		return
	}

	var (
		ctx = positional(p, p.Position())
		args = append(generic.options, props[1:]...)
		specVal = props[0]
        specNames []string
		opts useOpts
		err error
	)
	if args, err = parseOpts(ctx, &opts, args...); err != nil {
		erro(ctx, "parse use opts failed: %v", err).debug(1)
		return
	}
	for _, a := range args {
		if _, ok := a.(*Flag); ok || true {
			erro(ctx, "unkown use opts: %T %v", a, a).of(a).debug(1)
			return
		}
	}

	switch v := specVal.(type) {
    case *Pair:
        var s string
        if f, ok := v.Key.(*Flag); !ok {
            erro(p, "'%v' invalid use spec", v.Key).of(specVal)
            return
        } else if s, err = f.name.Strval(ctx); err != nil {
            erro(p, "%s", err).of(specVal)
            return
        } else if s != "list" {
            erro(p, "'%v' invalid use spec, do you mean -list?", v.Key).of(specVal)
            return
        }

        var list []Value
        if list, err = expandmerge2(ctx, expandPlainValue, v.Value); err != nil {
            erro(p, "%s", err).of(specVal)
            return
        }
        for _, val := range list {
            if s, err = val.Strval(ctx); err != nil {
                erro(p, "%s", err).of(specVal)
                return
            } else if s == "" { continue }
            specNames = append(specNames, s)
        }
    default:
        var specName string
        if specName, err = specVal.Strval(ctx); err != nil {
            erro(p, "%s", err).of(specVal)
            return
        } else if specName == "" { break }
        specNames = append(specNames, specName)
    }

    if len(specNames) == 0 {
        erro(p, "empty use spec (%v)", specVal).of(specVal).debug(1)
        return
    }

	var wg sync.WaitGroup
    for _, specName := range specNames {
		if true {
			p.loadUseSpecName(opts, specVal, specName, args...)
		} else {
			var dc = diagContext{ Context: ctx } // redefine ctx
			wg.Add(1); go func() {
				defer checkPanicsErrors(&dc, true)
				defer func() {
					if len(dc.points) > 0 { dc.inner().diagnostic().nest(dc.points) }
					wg.Done()
				} ()
				p.loadUseSpecName(opts, specVal, specName, args...)
			} ()
		}
    }
	wg.Wait()

	if errs := ctx.checkErrors(true); errs > 0 {
		var (
			pos = p.Position()
			proj = p.Project()
		)
        prompt(ctx, "%s: use %v failed; %d errors\n", proj, specNames, errs)
		erro(p, "%v errors: use %v", errs, specNames).at(pos).debug(6)
		if true { fail(pos, "%s: use %v failed; %d errors", proj, specNames, errs) }
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
		p.includeFile(p.Position(), x)
	}
}

func (p *parser) importFileMaps(ctx Context, public bool, paths ...Value) {
	if options.noImportFiles {
		return
	}

	var (
		opts = useOpts{ noVars:true, reuse:true, public:public }
		projects []*Project
		projMutx sync.Mutex
		wg sync.WaitGroup
	)
	for _, val := range paths {
		var (
			ctx = positional(p, val.Position())
			name, err = val.Strval(ctx)
		)
        if err != nil {
          erro(ctx, "strval '%v' failed: %v", val, err).debug(1)
          return
        }
		if false { // FIXME: parellel loading failed
			wg.Add(1); go func() {
				defer checkPanicsErrors(ctx, true)
				defer wg.Done()
				var loaded = p.loadUseSpecName(opts, val, name)
				projMutx.Lock()
				projects = append(projects, loaded)
				projMutx.Unlock()
			} ()
		} else {
			var loaded = p.loadUseSpecName(opts, val, name)
			projects = append(projects, loaded)
		}
	}
	wg.Wait()

	p.importFileMaps1(ctx, opts, projects...)
}

func (p *parser) importFileMaps1(ctx Context, opts useOpts, projects ...*Project) {
	if !opts.public && opts.filesPub { opts.public = true }
	for _, proj := range projects {
		var filemaps = p.Project()._filemap_
		for _, fm := range proj.filemaps(ctx, false, false) {
			if fm.public {
				if !opts.public {
					fm = &FileMap{ fm.project, fm.pattern, fm.paths, opts.public }
				}
				filemaps = uniqueAppendFilemap(ctx, filemaps, fm)
			}
		}
		p.Project()._filemap_ = filemaps
	}
}

type filesOpts struct {
	public bool `p,pub;p,public`
}
func (p *parser) parseFilesSpec(doc *CommentGroup, generic *genericoptions, _ int) {
	defer p.setbits(p.setbit(parsingFilesSpec))
	var props = p.parseDirectiveSpec()
	if len(props) != 1 {
		erro(p, "too many files properties: %v", props).at(p.Position()).debug(1)
		return
	}

	var path Value
	if p.tok == token.SELECT_PROG1 {
		p.next(true) // step forward with spaces skipped
		if p.tok == token.LINEND || p.lineComment != nil {
			erro(p, "expecting files path")
		}
		path = p.parseExpr(false)
	}
	if p.skipSpaces(); p.lineComment != nil {
		//spec.Comment = p.lineComment
	}
	if generic.dontOperate {
		// TODO: maybe give some information
		return
	}

	var (
		ctx = positional(p, p.Position())
		opts filesOpts
		pats []Value
		err error
		val = props[0]
	)
	if _, err = parseOpts(ctx, &opts, generic.options...); err != nil {
		erro(p, "parse use opts failed: %v", err).at(p.Position()).debug(1)
		return
	} else if g, ok := val.(*Group); ok {
		pats = g.Elems
	} else if val.expandible(ctx, expandClosure) {
		pats = []Value{ val }
	} else if pats, err = expandmerge2(ctx, expandPlainValue, val); err != nil {
		erro(ctx, "%s", err).of(val)
		return
	}
	if path == nil {
		if len(pats) == 1 { if a, ok := pats[0].(*Argumented); ok { if f, ok := a.value.(*Flag); ok {
			var name, err = f.name.Strval(ctx) // -import(paths...)
			if err != nil {
				erro(ctx, "strval '%v' failed: %v", f.name, err).of(f.name).debug(1)
				return
			}
			switch name {
			case "import": p.importFileMaps(ctx, opts.public, a.args...); return
			default:
				erro(ctx, "invalid files flag: %v").of(f.name).debug(1)
				return
			}
		}}}
		var ( files []*File; newPats []Value )
		for _, pat := range pats {
			if f, ok := pat.(*File); ok {
				files = append(files, f)
			} else {
				newPats = append(newPats, pat)
			}
		}
		if len(files) > 0 {
			if false {
				for _, file := range files { p.Project().mapfile(ctx, opts, file, nil) }
			} else {
				val = MakeListOrScalar(val.Position(), values(files))
				p.Project().mapfile(ctx, opts, val, nil)
			}
			pats = newPats
		}
		if len(pats) > 0 {
			var paths = []Value{ MakeString(val.Position(), p.Project().absPath) }
			for _, pat := range pats { p.Project().mapfile(ctx, opts, pat, paths) }
		}
	} else {
		var patsNew []Value
		for _, pat := range pats {
			if pat.expandible(ctx, expandClosure) {
				patsNew = append(patsNew, pat)
			} else if v, err := expandmerge2(ctx, expandPlainValue, pat); err != nil {
				erro(p, "merge value '%v' failed: %v", pat, err).of(pat)
			} else {
				patsNew = append(patsNew, v...)
			}
		}

		var paths []Value
		if g, ok := path.(*Group); ok {
			paths = g.Elems
		} else {
			paths = []Value{ path }
		}

		if len(patsNew) == 1 { if f, ok := patsNew[0].(*Flag); ok {
			var name, err = f.name.Strval(ctx) // -import => (paths...)
			if err != nil {
				erro(ctx, "strval '%v' failed: %v", f.name, err).of(f.name).debug(1)
				return
			}
			switch name {
			case "import": p.importFileMaps(ctx, opts.public, paths...); return
			default:
				erro(ctx, "invalid files flag: %v").of(f.name).debug(1)
				return
			}
		}}
		for _, k := range patsNew { p.Project().mapfile(ctx, opts, k, paths) }
	}
}

func (p *parser) parseEvalSpec(doc *CommentGroup, generic *genericoptions, _ int) {
	var (
		position = p.Position()
		props = p.parseDirectiveSpec()
		resolved Value
		err error
	)
	if prop0 := props[0]; isNil(prop0) {
		erro(p, "illegal").at(position).debug(1)
	} else if position = prop0.Position(); !position.IsValid() {
		erro(p, "command name '%v' has invalid position", prop0).at(position).debug(1)
	} else if _, resolved, err = p.resolveObject(prop0); err != nil {
		erro(p, "resolve '%v' failed: %v", prop0, err).at(position).debug(1)
	} else if isNil(resolved) {
		erro(p, "resolved '%v' is nil", prop0).at(position).debug(1)
	} else if b, ok := resolved.(*Builtin); ok && (b.flag&builtinCommand) == 0 {
		erro(p, "resolved builtin '%v' is not a command: %T", prop0, resolved).at(position).debug(1)
	} else if !generic.dontOperate { //p.evalspec(spec)
        // At the point of `eval` was represented, the closure context
        // might be empty. So we start closure with the current scope.
        //defer setclosure(setclosure(cloctx.unshift(p.scope)))
		var ( ctx = positional(p, position); res Value )
        switch op := prop0.(type) {
        case Caller: res = op.Call(ctx, props[1:]...)
        default:
            var name string
            if name, err = op.Strval(ctx); err != nil {
                erro(p, "strval '%s' failed: %v", op, err).at(position).debug(1)
            } else if _, obj := p.Scope().Find(name); obj == nil {
                erro(p, "`%s` undefined", name).at(position).debug(1)
            } else if f, _ := obj.(Caller); f == nil {
                erro(p, "`%T` is not caller (%s)", obj, name).at(position).debug(1)
            } else {
                res = f.Call(ctx, props[1:]...)
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
			ctx = positional(p, x.Position())
			conds []Value
		)
		switch t := x.(type) {
		case *Argumented:
			if flag, ok := t.value.(*Flag); !ok {
				// does nothing
			} else if s, err := flag.name.Strval(ctx); err != nil {
				erro(p, "strval flag name '%v' failed: %v", flag.name, err).at(x.Position())
			} else if s == "cond" {
				conds = t.args
			}
		case *Pair:
			if flag, ok := t.Key.(*Flag); !ok {
				// does nothing
			} else if s, err := flag.name.Strval(ctx); err != nil {
				erro(p, "strval flag name '%v' failed: %v", flag.name, err).at(x.Position())
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
			if t, e := cond.True(ctx); e != nil {
				erro(p, "conditon casting '%v' failed: %v", cond, e).at(x.Position())
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
	if p.Scope().comment == usecomment {
		switch i := ident.(type) {
		case *selection:
			erro(p, "should use scoped names instead of `%v`", i).of(ident)
		default:
			erro(p, "FIXME: unexpected name expression: %T", i).of(ident)
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
		value = MakeList(p.Position(), elems...)
	}

	// NOTE: forcely put all explicit defs into project scope. It's important for defs enclosed
	//       in templates work.
	defer func(s *Scope) { p.scopes[0] = s } (p.Scope())
	p.scopes[0] = p.Project().scope

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
		position = p.Position()
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
				isValue = p.dialect == "value"
				x = p.parseExpr(!isValue) // parse first expr of recipe
			)
			if isNil(x) {
				erro(p, "parsed value is nil").at(position)
			} else if t, ok := x.(*Bareword); ok && !isValue {
				if _, sym, err := p.resolveObject(t); err != nil {
					erro(p, "resolve '%v' failed: %v", x, err).at(position)
				} else if isNil(sym) {
					erro(p, "resolved '%v' (from %v) is nil", t.string, x).at(position)
				} else {
					x = sym
				}
			}

			if !isValue && p.tok.IsAssign() {
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
		for !p.isEndOfLine() {
			var bits = p.setbit(parsingRecipeText)
			var x Value
			switch p.tok {
			default:           x = p.parseExpr(false)
			case token.RAW:    x = p.parseBasicLit(false)
				/*
			case token.LINEND:
				erro(p, "unexpected end of line for compound string")
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
        return MakeList(p.Position(), elems...)
    } else {
        return MakeCompound(position, elems...)
    }
}

func (p *parser) parseModifySetVar(args []Value) (err error) {
	// Parsing (var a=xxx,b=yyy) definitions
	for _, elem := range args[1:] {
		var kv, ok = elem.(*Pair)
		if !ok || kv == nil {
			erro(p, "bad var form (%T)", elem).of(elem)
			continue
		}
		var name string
		var k, v = kv.Key, kv.Value
		if name, err = k.Strval(positional(p, k.Position())); err != nil {
			erro(p, "strval '%v' failed: %v", k, err).of(k)
		} else if name == "" {
			erro(p, "name '%v' is empty", k).of(k)
		}
		if def, alt := p.def(elem.Position(), name); alt != nil {
			erro(p, "Def '%v' already existed: %T", name, alt).of(k)
		} else if def != nil {
			var ctx = positional(p, v.Position())
			if g, ok := v.(*Group); ok {
				def.val(ctx, g.ToList(def.position))
			} else {
				def.val(ctx, v)
			}
		}
	}
	return
}

func (p *parser) defineConfigureTargets() {
	for _, t := range p.targets {
		var pos = t.Position()
		if !pos.IsValid() { pos = p.Position() }

		var (
			ctx = positional(p, pos)
			name, err = t.Strval(ctx)
		)
		if err != nil {
			erro(ctx, "strval target '%v' failed: %v", t, err).of(t).debug(6)
			return
		}

		var def, alt = p.project.scope.define(ctx, /*DefVoid*/DefConfig, name, nil)
		if def == nil && alt != nil { if def, _ = alt.(*Def); def == nil {
			erro(ctx, "configure %v: already defined in '%v' as %v", t, p.project, alt).debug(6)
			return
		}}
		if !def.position.IsValid() { def.position = pos }
	}
}

func (p *parser) parseModifyParams(args []Value) (err error) {
	for _, elem := range args {
		var ctx = positional(p, elem.Position())
		switch elem.(type) {
		case *Bareword, *Barecomp:
			var s string
			if s, err = elem.Strval(ctx); err != nil {
				erro(p, "strval '%v' failed: %v", elem, err).of(elem)
				return
			}
			var def, alt = p.def(elem.Position(), s)
			if alt != nil {
				var ok bool
				if def, ok = alt.(*Def); !ok {
					erro(p, "%T '%s' already taken the name, no such parameter", alt, s).of(elem)
				}
			}
			if def != nil {
				def.set(ctx, DefArg, nil)
			} else {
				erro(p, "'%s' is not defined", s).of(elem)
			}
			p.params = append(p.params, def)
			p.Scope().replace(ctx, strconv.Itoa(len(p.params)), def)
		default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
			erro(p, "bad parameter form (%T)", elem).of(elem)
		}
	}
	return
}

func (p *parser) parseModifiersExpr() *modifiergroup {
	if t_traverse.enabled { defer un(trace(t_traverse, "Modifiers")) }

	var (
		posLp = p.positionAt(p.expect(token.LBRACK))
		hasParameters bool // ((foo bar))
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
			//erro(p, "invalid modifier: %T %v", x, x).at(g.position).debug(1)
			var xv Value
			if xv, err = x.expand(p, expandDelegate/*TODO: expandInline or expandAuto*/); err != nil {
				erro(p, "invalid modifier: %T %v", x, x).at(g.position).debug(1)
			} else {
				warn(p, "modifier: %T %v   →   %T %v", x, x, xv, xv).at(x.Position()).debug(1)
			}
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
				p.defineConfigureTargets()
				p.configure = true // set configure flag and define configure variables
			}
			goto checkNameAndAdd
		case *Group: // parameters: ((foo bar))
			hasParameters = true
			p.parseModifyParams(n.Elems)
			continue ForModifiersExpr
		case *delegate, *closure, *Barecomp, *String:
			var ( ctx = positional(p, n.Position()) ; v []Value )
			if v, err = expandmerge2(ctx, expandPlainValue, n); err != nil {
				erro(p, "merge '%v' failed: %v", v, err).of(n).
					debug(1)
			} else if name, err = v[0].Strval(ctx); err != nil {
				erro(p, "strval '%v' failed: %v", v[0], err).of(n).
					debug(1)
				continue ForModifiersExpr
			} else if name == "" {
				erro(p, "name '%v' is empty", n).of(n).
					debug(1)
				continue ForModifiersExpr
			}
			goto checkNameAndAdd
		default:
			erro(p, "unsupported dialect or modifier (%T): %v", group.Elems[0], group.Elems[0]).of(n).debug(1)
			continue ForModifiersExpr
		}

		goto addModifier

	checkNameAndAdd:
		if _, ok := dialects[name]; ok {
			if p.dialect == "" { p.dialect = name } else {
				erro(p, "multi-dialects unsupported, already defined '%s'", p.dialect).of(x).
					debug(1)
				continue ForModifiersExpr
			}
		} else if _, ok = modifiers[name]; !ok {
			erro(p, "`%s` no such dialect or modifier", name).of(x).debug(1)
			continue ForModifiersExpr
		}

	addModifier:
		if len(group.Elems) == 0 {
			erro(p, "empty modifier: %v", x).of(x).debug(1)
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
	if len(elems) == 0 && !hasParameters {
		erro(p, "empty modifier group").at(posLp).debug(1)
	}
	if p.tok == token.COLON {
		erro(p, "unexpected colon after modifer").at(posLp).debug(1)
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

func (p *parser) parseRuleEntry(special specialRule, options, targets []Value) (result Value) {
	if p.Project().keyword == token.PACKAGE {
		erro(p, "rules forbidden: %v", targets).at(p.Position()).debug(1)
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

	var (
		position = p.Position()
		ctx = positional(p, position)
		err error
	)

	defer p.closeScope(p.openScope(scopeComment))

	for _, s := range automatics {
		var def, alt = p.def(position, s)
		if alt != nil {
			erro(p, "name `%s' already taken, not automatic (%T).", s, alt)
		} else if def == nil {
			erro(p, "'%s' is not defined", s)
		} else {
			def.origin = DefAuto
		}
	}
	for i := 1; i < 10; i += 1 {
		var def, alt = p.def(position, strconv.Itoa(i))
		if alt != nil {
			erro(p, "name `%v` already taken, not numberred (%T).", i, alt)
		} else if def == nil {
			erro(p, "'$%d' is not defined", i)
		} else {
			def.origin = DefAuto
		}
	}

	switch special {
	case specialRuleUse:
		if name, alt := p.Scope().ProjectName(ctx, selfproj, p.Project()); alt != nil {
			erro(p, "name `%s` already taken, not automatic (%T)", selfproj, alt)
		} else if name == nil {
			erro(p, "cannot define `%s` automatic", selfproj)
		}
		if name, alt := p.Scope().ProjectName(ctx, userproj, nil); alt != nil {
			erro(p, "name `%s` already taken, not automatic (%T)", userproj, alt)
		} else if name == nil {
			erro(p, "cannot define `%s` automatic", userproj)
		}
	}

	// NOTE: expand targets to speed up for later usage, it might spend lots of time in
	// project.entry while matching for entry looked up if not expanded right now.
	if true {
		targets, _, err = expandall2(ctx, expandPlainValue, targets...)
		if err != nil { erro(p, "expand targets '%v' failed: %v", targets, err) }
	} else {
		var ta []Value
		for _, t := range targets {
			if t.expandible(ctx, expandClosure) {
				if false { info(ctx, "target: %T %v", t, t).of(t) }
				ta = append(ta, t)
			} else if a, _, e := expandall2(ctx, expandPlainValue, t); e == nil {
				ta = append(ta, a...)
			} else {
				erro(p, "expand targets '%v' failed: %v", targets, e)
			}
		}
		targets = ta
	}

	defer func(t []Value) { p.targets = t } (p.targets)
	p.next(true) // skip rule delimeters and spaces
	p.targets = targets // save targets for later refering

	if p.tok != token.SEMICOLON && p.tok != token.BAR && !p.isEndOfLine() {
		depends = p.parseDependList()
	}
	if p.tok == token.BAR {
		p.next(true) // '|' starts the ordered prerequisites
		if p.tok != token.SEMICOLON && !p.isEndOfLine() {
			ordered = p.parseDependList()
		}
	}

	p.scanner.TrunRecipesOn() // Turn on recipes before LINEND
	if p.tok == token.SEMICOLON { // ;
		// Parse inline recipe in the program scope.
		recipes = append(recipes, p.parseRecipeExpr())
	} else if /*p.tok == token.LINEND || p.lineComment != nil*/p.expectLinend() {
		//p.expectLinend() // Take the new line
		// Parse recipes in the program scope.
		for p.tok != token.EOF && p.isRecipeStart() {
			recipes = append(recipes, p.parseRecipeExpr())
		}
	}
	p.scanner.TurnRecipesOff()

	var params []string
	if t := targets[0]; p.configure {
		if name, err := t.Strval(ctx); err != nil {
			erro(p, "strval configure target '%v' failed: %v", t, err).of(t)
		} else {
			d, a := p.Project().scope.define(ctx, DefVoid, name, nil)
			if d == nil && a == nil {
				erro(p, "cannot define configure target '%v'", name).of(t)
			} else if a != nil {
				if _, ok := a.(*Def); !ok {
					erro(p, "configure target '%v' already taken: %T %v", name, a, a).of(t)
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
		var res []Entry
		if res = p.rule(parsedData); len(res) == 1 {
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

func (p *parser) parseSpecialRuleClause() Value {
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
			// Example usage of using.*:
			//    using.* ::= cflags(-unique) ldlibs(-unique -reverse)
			erro(p, ":user: rules are deprecated, use using.* instead!").debug(1)
		} else {
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

			erro(p, "expecting special rule terminator ':'")
		}
		return nil
	default:
		erro(p, "unknown special rule")
		return nil
	}
}

func (p *parser) expandForeach(t *template, vars map[string]Value, params []Value, endPos token.Pos) {
	p.scanner.SetState(t.state)
	p.pos, p.tok, p.lit = t.pos, t.tok, t.lit

	// TODO: deal with params
	defer p.closeScope(p.openScope("template expansion ")) // NOTE: comment here will affect loader.def()

	//t_traverse.tracef("%v %v", t_traverse.elapsed(), vars)
	var ctx = positional(p, p.Position())
	for s, v := range vars {
		var def, alt = p.def(p.Position(), s)
		if alt == nil { def.set(ctx, DefAuto, v) } else {
			erro(p, "variable '%s' already taken", s).at(p.Position()).debug(1)
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
	switch t.verb {
	case "foreach":
		for _, a := range t.params {
			p.expandForeach(t, map[string]Value{ "_" : a }, params, endPos)
		}
	case "for":
		for _, a := range t.params {
			if pair, ok := a.(*Pair); ok {
				if s, e := pair.Key.Strval(positional(p, pair.Key.Position())); e != nil {
					erro(p, "expand template %v", e).of(pair.Key).debug(1)
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
		erro(p, "expand template %v: %v", t.verb, params).at(p.Position()).debug(1)
	}
}

func (p *parser) parseTemplateClause() (end bool) {
	var pos = p.pos
	p.expect(token.TEMPLATE) // expect and skip 'template'
	p.skipSpaces()

	var verb string
	var op = p.parseExpr(false); p.skipSpaces()
	if w, ok := op.(*Bareword); ok { verb = w.string } else {
		erro(p, "unknown template verb: %v", op).of(op).debug(1)
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
		erro(p, "TODO: save template for later usage: ", params).of(op).debug(true)
		return
	}

	var ( ctx = positional(p, p.Position()); err error )
	if params, err = expandmerge2(ctx, expandPlainValue, params...); err != nil {
		erro(p, "merge params %s failed", err).at(p.positionAt(pos))
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
	var position = p.Position()
	switch p.tok {
	case token.USE:
		erro(p, "`%v` unexpected here", p.tok).at(position).debug(1)
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

	erro(p, "bad clause: %v (%s) after %v", p.tok, p.lit, list).debug(6)
}

type projectDeclOpts struct {
	final bool `f,final`
	noDock bool `n,nod;n,nodock;nd,no-dock`
    breakUseLoop bool `b,break;l,loop`  // don't recursively use this project
    multiUseAllowed bool `m,multi`  // this project is used multiple times
}

func (p *parser) parseFile() *parsedFile {
	if options.traceLaunch { defer un(trace(t_launch, "parser.parseFile")) }
	if t_traverse.enabled { defer un(trace(t_traverse, "File '"+p.file.Name()+"'")) }
    if false { defer un(tracef(t_traverse, "parseFile(%s)", p.file.Name())) }

	// Don't bother parsing the rest if we had errors scanning the first token.
	// Likely not a Go source file at all.
	if p.countErrors() > 0 { return nil }

	var (
		abs, rel, tmp string
		ident *Barecomp //Bareword
		identStr string
		implicitBase string // aka. foo.bar.Baz implicitly load base 'foo/bar'
		keyword = p.tok
		filename = p.file.Name()
		position = p.Position()
		ctx = positional(p, position)
	)
	defer p.closeScope(p.openScope(fmt.Sprintf("file %s", filename)))
	/*if filename == confinitFilename {
                abs, rel = context.workdir, "."
                tmp = joinTmpPath(context.workdir, rel)
        } else*/ {
		if p.mode&Flat != 0 {
			abs = p.Project().absPath
		} else {
			abs = filepath.Dir(filename)
		}
		rel, _ = filepath.Rel(p.WorkDir(), abs)
		tmp = joinTmpPath(ctx, p.WorkDir(), rel)
	}

	if s := p.Scope(); s != nil {
		//defer p.closeScope()
		var def *Def
		if p.mode&Flat == 0 {
			def, _ = p.def(position, ".")
			def.set(ctx, DefAuto, MakePathStr(position, rel))

			def, _ = p.def(position, "/")
			def.set(ctx, DefAuto, MakePathStr(position, abs))

			def, _ = p.def(position, "CTD") // Current Temp Directory
			def.set(ctx, DefAuto, MakePathStr(position, tmp))

			def, _ = p.def(position, "CWD") // Current Work Directory
			def.set(ctx, DefAuto, MakePathStr(position, abs))
		} else if def = s.FindDef("/"); def == nil {
			erro(p, "/ not in the scope: %v", s.comment).at(position)
		} else if def = s.FindDef("."); def == nil {
			erro(p, ". not in the scope: %v", s.comment).at(position)
		} else if def = s.FindDef("CTD"); def == nil {
			erro(p, "CTD not in the scope: %v", s.comment).at(position)
		} else if def = s.FindDef("CWD"); def == nil {
			erro(p, "CWD not in the scope: %v", s.comment).at(position)
		}
	} else {
		erro(p, "opened invalid scope for %s", filename).at(position).debug(1)
		return nil
	}

	switch keyword {
	case token.CONFIGURE:
		switch p.next(true); p.tok {
		case token.DOT:
			if err := p.ParseConfigDir(abs, abs); err != nil {
				erro(p, "parsing configure directory failed, '%s': %v", abs, err)
			} else {
				p.next(true) // skip the '.' token and consequence spaces
			}

			basename := filepath.Base(filepath.Dir(filename))
			ident = MakeBarecomp(position, MakeBareword(position, basename))

		default:
			erro(p, "unknown configuration '%v', currently only 'configure .' is supported", p.tok)
		}
	case token.PROJECT, token.PACKAGE, token.MODULE:
		if p.mode&Flat != 0 {
			erro(p, "forbidden `%v` in flat file", p.tok)
		}

		p.next(true)

		// Options are *Flag or *Pair of a Flag.
		var ( opts projectDeclOpts; optVals []Value; pos Position )
		for p.tok == token.MINUS {
			var opt = p.parseExpr(false);  p.skipSpaces()
			optVals = append(optVals, opt)
			if !pos.IsValid() { pos = opt.Position() }
		}
		if !pos.IsValid() { pos = p.Position() }
		if a, e := parseOpts(ctx, &opts, optVals...); e != nil {
			erro(p, "parse project decl opts failed: %v", e).at(pos).debug(1)
			return nil
		} else if len(a) > 0 {
			for _, v := range a {
				erro(p, "unknown option '%v'", v).of(v).debug(1)
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
				erro(p, "`%v` not a smart file", filepath.Base(filename)).
					at(p.Position()).debug(1)
			} else if s := strings.TrimSuffix(filepath.Base(filename), ext); s != "" {
				ident = MakeBarecomp(position, MakeBareword(position, s))
			} else {
				erro(p, "`%v` not tilde name", filepath.Base(filename)).
					at(p.Position()).debug(1)
			}
			p.next(true) // skip tilde
		} else {
			var implicitBaseSegs []string
			ident = MakeBarecomp(p.Position())
		ForProjectName:
			for p.tok != token.EOF && p.tok != token.SPACE {
				if w := p.parseBarewordConstant(false); w == nil {
					erro(p, "expecting a bareword").
						at(ident.Position()).debug(1)
				} else if word, ok := w.(*Bareword); !ok {
					erro(p, "expecting a bareword: %v (%T)", w, w).
						at(ident.Position()).debug(1)
				} else if ident.Combine(p, word); p.tok == token.DOT {
					ident.Combine(p, MakeBareword(p.Position(), ".")) // TODO: parse to Qualiword
					implicitBaseSegs = append(implicitBaseSegs, word.string)
					p._next() // '.'
				} else { break ForProjectName }
			}
			p.skipSpaces()
			if len(ident.Elems) == 0 {
				erro(p, "package name is empty").at(p.Position()).debug(1)
				return nil
			} else if len(implicitBaseSegs) > 0 {
				implicitBase = filepath.Join(implicitBaseSegs...)
			}
		}

		var err error
		if identStr, err = ident.Strval(ctx); err != nil {
			erro(p, "strval '%v' failed: %v", ident, err).at(ident.Position()).debug(1)
			return nil
		} else if linfo.loadee != nil && identStr != linfo.loadee.name {
			warn(p, "%s: declare multiple project in the same directory", p.Project()).at(ident.position).debug(24)
		} else if identStr == "_" && p.mode&DeclarationErrors != 0 {
			erro(p, "package name '_' is preserved").at(ident.Position()).debug(1)
			return nil
		}

		// Don't bother parsing the rest if we had errors parsing the package clause.
		// Likely not a Go source file at all.
		if n := p.countErrors(); n > 0 {
			erro(p, "got %d errors parsing file: %s", filename).at(p.Position()).debug(1)
			return nil
		}

		var (
			loaderProj = p.project
			_, declared = linfo.declares[identStr]
		)
		if (p.mode&Flat == 0) && p.declare(keyword, ident, identStr, optVals) {
			// Change the 'default' owners into the new declared project
			if s := p.Scope(); s != nil {
				if def := s.FindDef("."  ); def != nil { def.owner = p.Project() }
				if def := s.FindDef("/"  ); def != nil { def.owner = p.Project() }
				if def := s.FindDef("CTD"); def != nil { def.owner = p.Project() }
				if def := s.FindDef("CWD"); def != nil { def.owner = p.Project() }
			} else {
				erro(p, "file scope is nil").at(position).debug(1)
			}
			// NOTE: build.smart is always the first loaded, so the loadee will be pointed to it
			if linfo.loadee == nil { linfo.loadee = p.Project() }
			defer func(proj *Project) {
				if false && loaderProj != nil && filepath.Base(filename) == "build.smart" {
					var ctx = positional(p, ident.Position())
					assert(p.project == proj, "diverged project: %v != %v", p.project, proj)
					//applyUseeVars(ctx, loaderProj, p.project)  // aka. ABC += $(using.ABC)
					applyUsingVars(ctx, loaderProj, p.project) // aka. using.ABC += $(using.ABC)
					if loaderProj.name == "llvm.Analysis" {
						warn(ctx, "%v, %v", loaderProj, p.project).debug(24)
					}
				}
				p.isLoadingBases = false
				p.closeCurrent(ident, identStr)
			} (p.project)
		}

		var basePos Position
		if implicitBase != "" { basePos = pos } else { basePos = p.Position() }
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
						erro(p, "unexpected end of file while parsing bases").at(basePos).debug(1)
						return nil
					}
					var t, e = parseOpts(positional(p, param.Position()), &opts, param)
					if false { info(p, "%v, %v, %v -> %v", p.Project(), ident, param, t).at(position).debug(1) }
					if ; e != nil {
						erro(p, "parse opt '%v' failed: %v", param, e).of(param).debug(1)
						return nil
					} else if keyword == token.PACKAGE || opts.final {
						// No bases for PACKAGE or final project
					} else if !p.loadBases(basePos, linfo, /*implicitBase*/"", merge(t...)...) {
						erro(p, "loading base '%v' failed", t).of(param).debug(1)
						return nil
					}
				}
				if p.tok != token.COMMA { break }
			}
			p.isLoadingBases = false
			p.expect(token.RPAREN)
		} else if !p.loadBases(basePos, linfo, implicitBase) { // for special bases, e.g. .base
			erro(p, "loading bases failed").at(basePos).debug(1)
			return nil
		}
		p.expectLinend()
		if keyword != token.PACKAGE {
			p.loadProjectConfiguration(ident, identStr, declared)
			if !opts.noDock { p.loadProjectContainer(ident, identStr) }
		}
	case token.EOF:
		return nil
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
						if p.Project() == nil {
							erro(p, "no project declared before defining rules")
						} else {
							x = p.parseRuleEntry(specialRuleNor, nil, []Value{x})
						}
						break ForInit
					} else {
						erro(p, "unexpected %v (after %v)", p.tok, x)
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
		scope:      p.Scope(),
		using:      p.imports,
	}
}
