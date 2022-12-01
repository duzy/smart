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

type parsingBits uint64
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

	parsingCompound
	parsingDefineClause

	parsingFilesSpec // files ( ... )
	parsingTemplateBlock
	parsingUndefValue

	parsingSpecialRule // e.g. :use ...:
	//parsingColonName // e.g. $:use:

	parsingRecipeBuiltin // recipe builtin command
	parsingRecipeText
	parsingRecipe = parsingRecipeBuiltin | parsingRecipeText

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
	state scanner.ScanState
	end *scanner.ScanState
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
	*loader

	file    *token.File
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

	bits parsingBits
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
}
func (p *parser) inner() Context { return p.loader }
func (p *parser) String() string { return fmt.Sprintf("parser{%s}", p.loader) }

func (p *parser) init(ctx Context, l *loader, filename string, src []byte) {
	p.loader = l
	p.file = l.fset.AddFile(filename, -1, len(src))

	var m scanner.Mode
	if p.mode&ParseComments != 0 {
		//m = scanner.ScanComments
	}

	eh := func(pos token.Position, msg string) {
		errostack(p, 3, "%s", msg).at(Position(pos)).debug(128)
	}
	p.scanner.Init(p.file, src, eh, m)
	p.next(ctx, true)
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
func (p *parser) scanNext(ctx Context) {
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
		erro(ctx, "unexpected end of file").at(p.loc(pos)).debug(1)
	}
}

// Consume a comment and return it and the line on which it ends.
func (p *parser) consumeComment(ctx Context) (comment *Comment, endline int) {
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
	p.scanNext(ctx)

	return
}

// Consume a group of adjacent comments, add it to the parser's
// comments list, and return it together with the line at which
// the last comment in the group ends. A non-comment token or n
// empty lines terminate a comment group.
//
func (p *parser) consumeCommentGroup(ctx Context, n int) (comments *CommentGroup, endline int) {
	comments = new(CommentGroup)
	// add comment group to the comments list
	p.comments = append(p.comments, comments)

	endline = p.file.Line(p.pos)
	for p.tok == token.COMMENT && p.file.Line(p.pos) <= endline+n {
		var comment *Comment
		comment, endline = p.consumeComment(ctx)
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
func (p *parser) _next(ctx Context) {
	p.leadComment = nil
	p.lineComment = nil
	prev := p.pos
	if p.scanNext(ctx); p.tok == token.COMMENT {
		var comment *CommentGroup
		var endline int

		// If the comment is on same line as the previous token; it
		// cannot be a lead comment but may be a line comment.
		if p.file.Line(p.pos) == p.file.Line(prev) {
			comment, endline = p.consumeCommentGroup(ctx, 0)
			if p.file.Line(p.pos) != endline {
				// The next token is on a different line, thus
				// the last comment group is a line comment.
				p.lineComment = comment
			}
		}

		// consume successor comments, if any
		endline = -1
		for p.tok == token.COMMENT {
			comment, endline = p.consumeCommentGroup(ctx, 1)
		}

		if endline+1 == p.file.Line(p.pos) {
			// The next token is following on the line immediately after the
			// comment group, thus the last comment group is a lead comment.
			p.leadComment = comment
		}
	}
	// if p.tok != token.LINEND && p.lineComment != nil {
	// 	p.tok = token.LINEND
	// }
}

func (p *parser) next(ctx Context, skipWS bool) {
	if p._next(ctx); skipWS { p.skipSpaces(ctx) }
	// if p._next(); skipWS && p.tok == token.SPACE { p._next() }
	// if p._next(); skipWS {
	// 	if p.tok == token.SPACE ||
	// 		(p.tok == token.RECIPE && p.bits&parsingRecipeBuiltin != 0) {
	// 		p._next()
	// 	}
	// }
}

func (p *parser) skipSpaces(ctx Context) {
	for p.lineComment == nil && p.tok != token.EOF {
		if p.tok == token.SPACE || (p.tok == token.RECIPE && p.bits&parsingRecipeBuiltin != 0) {
			p._next(ctx)
		} else if p.tok == token.ESCAPE && p.lit == "\n" {
			if p._next(ctx); p.tok == token.LINEND || p.lineComment != nil { break }
			if p.bits&parsingRecipeBuiltin != 0 {
				TokFor: for p.tok != token.EOF {
					switch p.tok {
					case token.RECIPE: // TODO: using p.isRecipeStart()
						p.scanner.LeaveCompoundLineContext()
						p._next(ctx)
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
func (p *parser) loc(pos token.Pos) Position { return Position(p.file.Position(pos)) }
func (p *parser) posit(c Context) Context { return positional(c, p.Position()) }

// A bailout panic is raised to indicate early termination.
type bailout struct{}

func (p *parser) expected(ctx Context, pos token.Pos, msg string, a... interface{}) {
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
	erro(p, msg).at(p.loc(pos)).debug(24)
}

func (p *parser) expect(ctx Context, tok token.Token) token.Pos {
	pos := p.pos
	if p.tok != tok {
		p.expected(ctx, pos, "'"+tok.String()+"'")
	}
	p._next(ctx) // make progress
	return pos
}

func (p *parser) expectLinend(ctx Context) (ok bool) {
	if p.lineComment != nil {
		// The line comment is treated as LINEND, simply ignore it.
		p.lineComment, ok = nil, true
	} else if p.tok == token.EOF {
		ok = true
	} else if p.tok == token.LINEND {
		p._next(ctx); ok = true
	} else {
		p.expected(ctx, p.pos, "'\\n'")
	}
	return
}

func (p *parser) isRecipeStart(ctx Context) (res bool) {
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

func (p *parser) parseBarewordConstant(ctx Context, lhs bool) (x Value) {
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
			p.expect(ctx, token.BAREWORD)
		}
	}

	p._next(ctx) // consumes the word

	switch position := p.loc(pos); tok {
	case token.TRUE:  x = MakeBoolean(position,  true)
	case token.FALSE: x = MakeBoolean(position,  false)
	case token.YES:   x = MakeAnswer(position,   true)
	case token.NO:    x = MakeAnswer(position,   false)
	default:          x = MakeBareword(position, value)
	}
	return
}

func (p *parser) parseSelector(ctx Context) (res Value) {
	defer p.setbits(p.setbit(composingSELECT_PROP))
	res = p.parseExpr(ctx, false)
	return
}

func (p *parser) parseSelect(ctx Context, lhs Value) (res Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Select")) }

	position, tok := p.Position(), p.tok // the arrow '->' or '=>'
	ctx = positional(ctx, position)
	p._next(ctx) // skip '->' or '=>'

	var (
		proj = p.Project()
		okay bool
	)
	switch t := lhs.(type) {
	case *selection:
		if v := t.value(positional(ctx, t.Position())); isNil(v) {
			erro(ctx, "nil selection: %v", lhs).at(position).debug(1)
			return
		} else {
			lhs = v
		}
	case *Bareword:
        switch t.string {
        case "use", "usee": lhs = proj.use
        case "self": lhs = proj.self
        case "goals", "os", "mode":
			if lhs, okay = p.colonResolve(t.string); !okay {
				erro(ctx, `"%s" not defined`, t.string).debug(1)
				return
			}
        default:
            if name, o := p.resolveObject(lhs); false {
				erro(ctx, "resolve '%v' failed", lhs).at(lhs.Position())
				erro(ctx, "parser is here (tok=%s)", tok).at(position)
				erro(ctx, "parser to go here (tok=%s, lit=%s)", p.tok, p.lit).at(p.Position()).debug(8)
                return
            } else if !isNil(o) {
				lhs = o
			} else if tok == token.SELECT_PROG2 {
				res = MakeNil(position) // ignore
				return
			} else {
				erro(ctx, "%v: '%v' is undefined (name=%v, obj=%v)", proj, lhs, name, o).at(lhs.Position())
				erro(ctx, "%v: parser is here (name=%s, tok=%s)", proj, t.string, tok).at(position)
				erro(ctx, "%v: parser to go here (tok=%s, lit=%s)", proj, p.tok, p.lit).at(p.Position()).debug(16)
				return
            }
        }
    case *Barecomp: // for cases like '.foo'
        if name, o := p.resolveObject(t); false {
			erro(ctx, "resolve selection object '%v' (%s) error", lhs, name).of(lhs).debug(1)
			return
        } else if !isNil(o) {
			lhs = o
		} else if tok == token.SELECT_PROG2 {
			res = MakeNil(position) // ignore
			return
		} else {
			erro(ctx, "'%v' is undefined", lhs).of(lhs).debug(1)
			return
        }
	}

	if rhs := p.parseSelector(ctx); isNil(rhs) {
		res = MakeNil(position)
	} else {
		res = MakeSelection(position, tok, lhs, rhs)
	}

	if (p.tok == token.SELECT_PROP || p.tok == token.SELECT_PROG1 || p.tok == token.SELECT_PROG2) {
		res = p.parseSelect(ctx, res) // Continue the selection recursivly.
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
	if (p.bits&parsingRecipe != 0) && p.tok == token.RECIPE { // TODO: using p.isRecipeStart()
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

func (p *parser) parseDependList(ctx Context) (list []Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Depends")) }
	for p.tok != token.SEMICOLON && p.tok != token.BAR && !p.isEndOfLine() {
		if p.tok == token.COLON { // FIXME: this check is not working!
			// FIXME: detects unexpected colon ':'
			erro(p, "unexpected colon").at(p.Position()).debug(1)
			p.next(ctx, true) // just ignore this colon
		} else if p.skipSpaces(ctx); !p.isEndOfLine() {
			list = append(list, p.parseExpr(ctx, false))
			if p.tok == token.SPACE { p.next(ctx, true) } //p.skipSpaces(ctx)
		}
	}
	return
}

// If lhs is set, result list elements which are identifiers are not resolved.
func (p *parser) parseExprList(ctx Context, lhs bool) (list []Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "List")) }
	for p.skipSpaces(ctx); !p.isEndOfList(lhs); p.skipSpaces(ctx) {
		p.skipSpaces(ctx)
		list = append(list, p.parseExpr(ctx, lhs))
		// p.skipSpaces(ctx)
		// If there's a comment right after the parsed expression, we break
		// the expression list to treat the end-of-line comment like a LINEND.
		if p.lineComment != nil  { break }
		if p.tok == token.LINEND { break }
		if p.tok == token.EOF    { break }
	}
	return
}

func (p *parser) parseListExpr(ctx Context, lhs bool) *List {
	return MakeList(p.Position(), p.parseExprList(ctx, lhs)...)
}

func (p *parser) setRHS(v bool) (old bool) {
	old = p.inRhs; p.inRhs = v; return
}

func (p *parser) parseLhsList(ctx Context) []Value {
	defer p.setRHS(p.setRHS(false))
	// Line comment of previous lines will break the parsing loop,
	// so we clear the previous line comment
	p.lineComment = nil
	return p.parseExprList(ctx, true)
}

func (p *parser) parseRhsList(ctx Context) []Value {
	defer p.setRHS(p.setRHS(true))
	return p.parseExprList(ctx, false)
}

// ----------------------------------------------------------------------------
// Expressions

func (p *parser) parseGroupExpr(ctx Context, lhs bool) *Group {
	if t_traverse.enabled { defer un(trace(t_traverse, "Group")) }

	position := p.Position()
	ctx = p.posit(ctx)
	p.next(ctx, true)
	elems, converted := p.parseExprList(ctx, false), false
	for /*p.tok == token.COMMA*/p.tok != token.RPAREN && p.tok != token.EOF {
		//p.next(true) // skip token.COMMA
		switch p.tok {
		case token.BAR, token.COMMA, token.SEMICOLON:
			elems = append(elems, p.parsePunctuation(ctx))
			p.skipSpaces(ctx)
		}
		var next *List
		next = p.parseListExpr(ctx, false)
		if !converted {
			elems = []Value{ MakeList(p.Position(), elems...), next }
			converted = true
		} else {
			elems = append(elems, next)
		}
	}
	p.expect(ctx, token.RPAREN)
	return MakeGroup(position, elems...)
}

func (p *parser) parseArgumentedExpr(ctx Context, x Value) *Argumented {
	if t_traverse.enabled { defer un(trace(t_traverse, "Argumented")) }

	ctx = p.posit(ctx)
	p.next(ctx, true) // skip token.LPAREN

	var a = []Value{ p.parseListExpr(ctx, false) }
	for p.tok != token.RPAREN && p.tok != token.LINEND && p.tok != token.EOF {
		switch p.tok {
		case token.COMMA: p.next(ctx, true) // skip token.COMMA
		case token.BAR, token.SEMICOLON:
			if false {
				a = append(a, p.parsePunctuation(ctx))
				p.skipSpaces(ctx)
			} else {
				erro(ctx, "unexpected punctuation: %v", p.tok).debug(1)
			}
		}
		a = append(a, p.parseListExpr(p.posit(ctx), false))
	}
	p.expect(p.posit(ctx), token.RPAREN)
	return MakeArgumented(x, a...)
}

func (p *parser) parseGlobMeta(ctx Context) (x *GlobMeta) {
	pos, tok := p.Position(), p.tok
	p._next(p.posit(ctx))
	return MakeGlobMeta(pos, tok)
}

func (p *parser) parseGlobRange(ctx Context) (x *GlobRange) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Glob")) }

	position := p.Position()
	ctx = p.posit(ctx)
	p.expect(ctx, token.LBRACK) // skip '['

	chars := p.parseExpr(ctx, false)
	p.expect(ctx, token.RBRACK) // skip ']'

	return MakeGlobRange(position, chars)
}

func (p *parser) parseGlobExpr(ctx Context, x Value) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "Glob")) }

	ctx = p.posit(ctx)

	var components []Value
	if !isNil(x) { components = []Value{ x } }

	var pos = ctx.Position()

	// avoid nesting glob expressions
	defer p.setbits(p.setbit(composingGLOB))
ForGlobTok:
	for {
		if p.lineComment != nil { break ForGlobTok }
		switch p.tok {
		case token.PCON, token.RPAREN, token.COMMA, token.SPACE, token.LINEND, token.EOF:
			break ForGlobTok
		case token.STAR, token.QUE: // * ?
			x = p.parseGlobMeta(ctx)
		case token.LBRACK:
			// FIXME: '[...]' has been used for modifier expressions
			x = p.parseGlobRange(ctx)
		default:
			// FIXME: escaped glob metas/chars
			x = p.parseExpr(ctx, false)
		}
		components = append(components, x)
	}
	if components == nil {
		erro(ctx, "nil glob expression (tok=%v, lit=%v)", p.tok, p.lit)
	}
	return MakeGlobPattern(pos, components...)
}

func (p *parser) parsePercExpr(ctx Context, lhs bool, x Value) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "Perc")) }

	// avoid nesting percent expressions
	defer p.setbits(p.setbit(composingPERC))

	var (
		pos = p.pos
		y Value
	)
	if p._next(ctx); pos+1 == p.pos { // joint, e.g. '%.o', but skip '% .o'
		switch p.tok {
		case token.COLON, token.COLON2,
			token.LPAREN, token.RPAREN,
			token.LBRACK, token.RBRACK,
			token.LBRACE, token.RCOLON,
			token.PCON,   token.SEMICOLON,
			token.COMMA,  token.SPACE,
			token.LINEND:
		case token.PERC: // %%
			p._next(ctx) // consume the second %
			position := p.Position()
			perc2 := MakePercPattern(position, nil, nil)
			if pos+2 == p.pos {
				switch p.tok {
				case token.PERC: // %%%
					erro(p, "too many %")
				case token.PCON: // FIXES: %%/xxx -> Path(%% xxx)
					x = MakePercPattern(position, x, perc2)
					return p.parsePathExpr(ctx, lhs, x)
				case token.COLON,    token.COLON2,
					token.LPAREN,    token.RPAREN,
					token.LBRACK,    token.RBRACK,
					token.LBRACE,    token.RCOLON,
					token.SEMICOLON, token.COMMA,
					token.SPACE,     token.LINEND:
				default:
					var (
						yy = p.parseExpr(ctx, false)
						_, ok = yy.(*Path)
					)
					if ok { erro(p, "incorrect: %v, %v", x, yy).at(position) }
					assert(!ok, "the second part of aaa%%bbb/foo/bar parsed incorrectly as path")
					perc2.Suffix = yy
				}
			}
			y = perc2
		default:
			y = p.parseExpr(ctx, false)
		}
	}
	return MakePercPattern(p.loc(pos), x, y)
}

func (p *parser) parseRegexpExpr(ctx Context) (x Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Regexp")) }

	// avoid nesting percent expressions
	defer p.setbits(p.setbit(composingREXP))

	erro(p, "todo: regexp")

	return
}

func (p *parser) parseKeyValueExpr(ctx Context, x Value) *Pair {
	if t_traverse.enabled { defer un(trace(t_traverse, "Pair")) }
	position := p.Position()
	ctx = p.posit(ctx)
	p._next(ctx)
	var y Value
	if p.isEndOfList(false) {
		y = MakeNil(position)
	} else {
		y = p.parseExpr(ctx, false)
	}
	return MakePair(position, x, y)
}

func (p *parser) parseFlagExpr(ctx Context, lhs bool) *Flag {
	if t_traverse.enabled { defer un(trace(t_traverse, "Flag")) }

	var (
		position = p.Position()
		x Value
	)

	ctx = p.posit(ctx)
	p._next(ctx) // skip dash '-'

	// Flag expressions, excluding "-)" "-]" "-}" "-\n", "-=", "-:", etc.
	if p.isEndOfLine() || p.isEndOfList(false) ||
		p.tok == token.SPACE || p.tok == token.RECIPE {
		x = MakeNil(position)
	} else if false {
		x = p.parseExpr(ctx, false)
	} else {
		x = p.parseUnaryExpr(ctx, false)
	}
	return MakeFlagValue(position, x)
}

func (p *parser) parseNegExpr(ctx Context, lhs bool) *negative {
	if t_traverse.enabled { defer un(trace(t_traverse, "Negative")) }
	p.expect(ctx, token.EXC)
	return Negative(p.parseExpr(ctx, lhs))
}

func (p *parser) parsePunctuation(ctx Context) *Punctuation {
	if t_traverse.enabled { defer un(trace(t_traverse, "punctuation")) }
	var tok = p.tok
	p._next(ctx)
	return &Punctuation{valbase{p.Position()}, tok}
}

func (p *parser) parseBasicLit(ctx Context, lhs bool) (v Value) {
	pos, tok, lit := p.pos, p.tok, p.lit
	end := int(pos) + len(lit)
	switch tok {
	case token.STRING: end += 2
	}
	p._next(ctx)

	// ESCAPE is handled in value.EscapeChar
    var position = p.loc(pos)
	defer checkPanicsErrors(positional(p, position)) // panics from parse{int,float,hex,...}
    switch tok {
    case token.BAR: erro(p, "`|` is deprecated, changed the modifiers!").at(p.loc(pos))
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

func (p *parser) parseCompoundLit(ctx Context, lhs bool) *Compound {
	var (
		lpos = p.pos
		elems []Value
	)
	p._next(ctx)

	defer p.setbits(p.setbit(parsingCompound))

ForCompound:
	for p.tok != token.EOF && p.tok != token.COMPOSED {
		var x Value
		switch p.tok {
		default:           x = p.parseExpr(ctx, false)
		case token.RAW:    x = p.parseBasicLit(ctx, false)
		case token.LINEND:
			erro(p, "unexpected end of line for compound string")
			break ForCompound
		}
		if false && strings.Contains(x.String(), "\\\"") {
			warn(p, "%T %v", x, x).debug(1)
		}
		elems = append(elems, x)
	}
	p.expect(ctx, token.COMPOSED)
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
func (p *parser) parseDotExpr(ctx Context, lhs bool, x Value) (res *Barecomp) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Dot")) }

	defer p.setbits(p.setbit(composingDOT))

	var comp *Barecomp
	if x == nil { panic(fmt.Sprintf("nil dot (tok=%v)", p.tok)) }
	if comp, _ = x.(*Barecomp); comp == nil {
		comp = MakeBarecomp(x.Position())//(p.Position())
		comp.Elems = append(comp.Elems, x)
	}

	for /*comp.End() == p.pos && */!p.isEndOfDotConcat(lhs) {
		comp.Combine(p, p.parseComposedExpr(ctx, false))
		if p.tok == token.DOT /*&& comp.End() == p.pos*/ {
			var dot = MakeBareword(p.Position(), ".") // TODO: parse to Qualiword instead
			comp.Elems = append(comp.Elems, dot)
			p._next(ctx) // '.'
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

func (p *parser) parsePathExpr(ctx Context, lhs bool, start Value) *Path {
	if t_traverse.enabled { defer un(trace(t_traverse, "Path")) }

	defer p.setbits(p.setbit(composingPATH))

	var (
		position = start.Position() //p.Position()
		path *Path
		ok bool
	)
	if start == nil {
		erro(p, "bad closure/delegate name").at(position).debug(1)
		p._next(ctx)
		return MakePath(position) // empty path
	} else if path, ok = start.(*Path); !ok {
		path = MakePath(position, start)
	}

BuildPath:
	for p.tok == token.PCON {
		var pos = p.Position() // skips repeated '/' sequence
		for p._next(ctx); p.tok == token.PCON; p._next(ctx) { pos = p.Position() }
		switch p.tok {
		case token.RPAREN, token.LPAREN, token.RBRACE, token.LBRACE,
			 token.RCOLON, token.COMMA, token.SPACE, token.LINEND:
			// Encountered the tailing '/', append 'zero' segment.
			path.Elems = append(path.Elems, MakePathSeg(pos, 0))
			break BuildPath
		}

		var x = p.parseComposedExpr(ctx, false)
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

func (p *parser) parseURLExpr(ctx Context, lhs bool, scheme Value) (res Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "URL")) }

	defer p.setbits(p.setbit(composingURL))

	var (
		url = &URL{ Scheme:scheme }
		colon1 = p.expect(ctx, token.COLON) // consumes ':'
		colon2 = token.NoPos
		//colon3 = token.NoPos
		at = token.NoPos // @
	)

	if p.tok == token.PCON {
		p._next(ctx) // the first '/'
		if p.tok == token.PCON {
			p.expect(ctx, token.PCON) // the second '/'
		} else {
			erro(p, "TODO: URL path: %v (%T) (next: %s (%s))", scheme, scheme,  p.tok, p.lit).
				at(p.Position()).debug(1)
			res = MakeNil(p.Position())
			return
		}
	} else if !p.isEndOfURL(lhs) {
		erro(p, "TODO: URL: %v (%T) (next: %s (%s))", scheme, scheme,  p.tok, p.lit).
			at(p.loc(colon1)).debug(1)
		res = MakeNil(p.Position())
		return
	}

	if !p.isEndOfURL(lhs) {
		userOrHost := p.parseComposedExpr(ctx, false)
		if p.tok == token.COLON {
			url.Username, colon2 = userOrHost, p.pos
			p._next(ctx) // ':'
			if p.tok != token.AT && p.tok != token.PCON && !p.isEndOfURL(lhs) {
				url.Password = p.parseComposedExpr(ctx, false)
			}
		} else {
			url.Host = userOrHost
		}
		if p.tok == token.AT {
			p._next(ctx) // '@'
		}
	}
	if url.Host == nil && colon2 == token.NoPos && at == token.NoPos && !p.isEndOfURL(lhs) {
		url.Host = p.parseComposedExpr(ctx, false)
		if p.tok == token.COLON {
			//colon3 = p.pos
			p._next(ctx) // ':'
			if p.tok != token.SPACE && p.tok != token.LINEND {
				url.Port = p.parseComposedExpr(ctx, false)
			}
		}
	}
	if p.tok == token.PCON {
		url.Path = p.parsePathExpr(ctx, lhs, makePathSeg(p, p.tok))
	}
	// scanning '#' as token.HASH instead of token.COMMENT
	defer p.scanner.SetBits(p.scanner.AddBits(scanner.NoComments))
	if p.tok == token.QUE {
		p._next(ctx) // '?'
		if p.tok != token.HASH && !p.isEndOfURL(lhs) {
			url.Query = p.parseComposedExpr(ctx, false)
		}
	}
	if p.tok == token.HASH {
		p._next(ctx) // '#'
		if !p.isEndOfURL(lhs) {
			url.Fragment = p.parseComposedExpr(ctx, false)
		}
	}
	return url
}

func (p *parser) parseClosureDelegate(ctx Context) (result Value) {
	if t_traverse.enabled {	defer un(trace(t_traverse, "ClosureDelegate")) }

	// TODO:FIXME: push p.bits before entering a $(...) or &(...)
	// defer func(a parsingBits) { p.bits = a } (p.bits)
	// p.bits = p.bits & (parsingCompound | parsingFilesSpec | parsingRecipe)
	ctx = p.posit(ctx)

	var (
		scope = p.Scope()
		pos = p.pos
		tok = p.tok
		resolved Value // Object or *selection
		rest []Value
	)
	resolveConfig := func(val Value, name string) (obj Object) {
		if c := p.Project().configure; c != nil {
			obj = c.resolveObject(ctx, name)
		}
		return
	}

	const allowClosureName = true
	resolveObject := func(lPos Position, lTok token.Token, name Value) (str string, obj Value, okay bool) {
		var (
			proj = p.Project()
			err error
		)
		if sel, ok := name.(*selection); ok {
			if sel == nil {
				erro(ctx, "nil selection: %v", name).at(name.Position()).debug(1)
			} else if false {
				// NOTE: selected defs could have closured, have to preserve selection
				if obj, okay = sel, true; false {
					o := sel.object(ctx)
					v := sel.value(ctx)
					warn(ctx, "`%v`; %T %v", sel, o, o).of(name)
					warn(ctx, "`%v`; %T %v", sel, v, v).of(name)
					warn(ctx, "`%v`; closured = %v", sel, v.expandible(ctx, expandClosure)).of(name).debug(1)
				}
			} else if o := sel.object(ctx); o.DeclScope().comment == usecomment/*TODO: remove this check?*/ {
				obj, okay = unresolved(proj, name), true
			} else if isNil(o) {
				erro(ctx, "`%v` nil selection object", name).of(sel).debug(1)
			} else if v := sel.value(ctx); isNil(v) {
				erro(ctx, "`%v` not found in %v", sel.s, o).of(name).debug(1)
			} else if obj, okay = v.(Object); !okay {
				// return // just use the selected value
			}
			switch lTok {
			case token.LPAREN:
				if _, ok := obj.(Caller); !ok {
					erro(ctx, "selected object '%v' is not callable: %T %v", name, obj, obj).of(name).debug(16)
				}
			case token.LBRACE:
				if _, ok := obj.(Executer); !ok {
					erro(ctx, "selected object '%v' is not executer: %T %v", name, obj, obj).of(name).debug(1)
				}
			case token.LCOLON:
				erro(ctx, "selected object '%v' does not supported: %T %v", name, obj, obj).of(name).debug(1)
			}
			return
		}

		if val := name.expand(ctx, ident); val != name {
			if u, y := val.(unexpanded); y {
				obj, okay = unresolved(proj, u.Value), true
				return
			} else { name = val }
		}

		switch lTok {
		case token.LPAREN:
			if allowClosureName && name.expandible(ctx, expandDelegate|expandClosure) {
				obj, okay = unresolved(proj, name), true // recursive delegation or closure
				return
			} else if str, resolved = p.resolveObject(name); false {
				erro(ctx, "resolve '%v' (%s) failed", name, str).at(name.Position()).debug(1)
				return
			} else if str == "" {
				erro(ctx, "name '%v' is empty", name).at(name.Position()).debug(1)
				return
			} else if isNil(resolved) {
				if p.isIncludingConf {
					// Create an empty Def if it's referred in configuration.sm.
					def, _ := p.def(name.Position(), str)
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
					obj, okay = unresolved(proj, name), true // recursive delegation or closure
					return
				} else if p.bits&parsingUndefValue != 0 {
					obj, okay = unresolved(proj, undef{name}), true
					return
				}

				erro(ctx, "%v: %T %v -> '%s', is nil", proj, name, name, str).of(name)
				errostack(ctx, 128, "%v: %v", proj, ctx).of(name).debug(64)
			} else if sel, ok := resolved.(*selection); ok && sel != nil {
				obj, okay = sel, true
				return
			} else if caller, _ := resolved.(Caller); caller == nil {
				erro(ctx, "%v is not callable: %T", name, resolved).at(lPos).debug(16)
			} else if obj, okay = caller.(Object); !okay {
				erro(ctx, "%v is not object: %T", name, resolved).at(lPos).debug(16)
			} else if isNil(obj) {
				erro(ctx, "%v is nil: %T", name, resolved).at(lPos).debug(16)
			} else {
				return
			}
		case token.LBRACE:
			if allowClosureName && name.expandible(ctx, expandDelegate|expandClosure) {
				erro(ctx, "%v: name '%v' (%T) is closured", p.Project(), name, name).of(name).debug(1)
				return
			} else if resolved = p.resolveEntries(name); isNil(resolved) {
				if name.expandible(ctx, plain) {
					var s = name.Strval(ctx)
					erro(ctx, "resolved '%v' (aka. %s) is nil (project=%v)", name, s, proj).of(name).debug(1)
				} else {
					erro(ctx, "resolved '%v' is nil (project=%v)", name, proj).of(name).debug(1)
				}
			} else if exe, _ := resolved.(Executer); exe == nil {
				erro(ctx, "resolved '%v' of '%T' is not Executer", name, resolved).at(lPos).debug(1)
			} else if obj, okay = exe.(Object); !okay || isNil(obj) {
				erro(ctx, "resolved Executer '%v' of '%T' is not Object", name, resolved).at(lPos).debug(1)
			}
		case token.LCOLON:
			switch str = name.Strval(ctx); str {
			case "use", "usee": resolved = proj.use // TODO: move usee and self into ctx
			case "self": resolved = proj.self
			//TODO: case "ctd" : resolved = proj.ctd
			//TODO: case "cwd" : resolved = proj.cwd
			default: if o, found := ctx.colonResolve(str); found { resolved = o } else {
				erro(ctx, "unknown special property: %v", str, err).at(lPos).debug(1)
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
	switch p._next(ctx); p.tok {
	case token.LPAREN, token.LBRACE, token.LCOLON: // $(...), ${...}, $:...:
		var posLp = p.Position()
		tokLp  = p.tok
		p._next(ctx) // skips LPAREN, LBRACE, LCOLON

		var posName = p.Position()
		switch p.tok {
		case token.SPACE:
			erro(ctx, "unexpected spaces").at(posName).debug(1)
			return MakeNil(posName)
		case token.COLON:
			p._next(ctx);  posName = p.Position()
			warn(ctx, "colon").at(posName).debug(1)
		}

		if name = p.parseExpr(ctx, false); isNil(name) {
			erro(ctx, "%v: parsed name is nil", p.Project()).at(posName).debug(1)
		} else if !allowClosureName && name.expandible(ctx, expandDelegate|expandClosure) {
			erro(ctx, "%v: name '%v' (%T) is closured", p.Project(), name, name).at(posName).debug(1)
		} else if nameStr, obj, okay = resolveObject(posLp, tokLp, name); !okay {
			erro(ctx, "%v: name '%v' is unidentified", p.Project(), name).at(posName).debug(1)
		}
		if false && name.String() == ".test$1" {
			v := name.expand(ctx, plain)
			warnstack(ctx, 3, "%v: %T %v -> %T %v -> %T %v",
				nameStr, name, name, obj, obj, v, v).of(name).debug(1)
		}
		if def, y := obj.(*def); true && y && name.String() == ".test.v2" {
			v := name.expand(ctx, plain)
			warnstack(ctx, 3, "%v: %T %v -> %T %v -> %T %v ; %v",
				nameStr, name, name, obj, obj, v, v, def.origin).of(name).debug(1)
		}

		if  (tokLp == token.LPAREN && p.tok != token.RPAREN) ||
			(tokLp == token.LBRACE && p.tok != token.RBRACE) ||
			(tokLp == token.LCOLON && p.tok != token.RCOLON) {
			var autos []*def
			var savedAutos = p.autos
			var savedAutop = p.autop
			if nameStr == "" {
				// keep on...
			} else if nameStr == "auto" {
				if tokLp != token.LPAREN {
					erro(ctx, "%v: auto: incorrect left paren", p.Project()).at(posLp).debug(1)
				}
				p.skipSpaces(ctx) // skip the imediate spaces
				var al = p.parseListExpr(ctx, false)
				if rest = append(rest, al); p.tok == token.COMMA { p.next(ctx, true) }
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
					if s == "" { erro(ctx, "%v: auto: %v is empty",
						p.Project(), val).at(pos).debug(1) }
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
				rest = append(rest, p.parseListExpr(ctx, false))
				p.bits |= parsingUndefValue
				for ; p.tok == token.COMMA; {
					p.next(ctx, true) // consumes COMMA
					rest = append(rest, p.parseListExpr(ctx, false))
				}
				p.bits = savedBits
			} else if nameStr == "and" {
				p.bits |= parsingUndefValue
				for rest = append(rest, p.parseListExpr(ctx, false)); p.tok == token.COMMA; {
					p.next(ctx, true) // consumes COMMA
					rest = append(rest, p.parseListExpr(ctx, false))
				}
				p.bits = savedBits
			} else if nameStr == "or" {
				p.bits |= parsingUndefValue
				for rest = append(rest, p.parseListExpr(ctx, false)); p.tok == token.COMMA; {
					p.next(ctx, true) // consumes COMMA
					rest = append(rest, p.parseListExpr(ctx, false))
				}
				p.bits = savedBits
			} else {
				for rest = append(rest, p.parseListExpr(ctx, false)); p.tok == token.COMMA; {
					p.next(ctx, true) // consumes COMMA
					rest = append(rest, p.parseListExpr(ctx, false))
				}
			}
			p.autos = savedAutos
			p.autop = savedAutop
		}

		switch tokLp {
		case token.LPAREN: p.expect(ctx, token.RPAREN)
		case token.LBRACE: p.expect(ctx, token.RBRACE)
		case token.LCOLON: p.expect(ctx, token.RCOLON)
			if p.tok == token.ASSIGN { erro(ctx, "unexpected assignment").at(p.Position()).debug(1) }
		}

	default:
		if position := p.Position(); tok != token.CLOSURE { // $(...), disabled $name.
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v` or `%v` or quotes", token.LPAREN, token.LBRACE).at(position).debug(1)
			return MakeNil(position)
		} else if p.tok == token.STRING || p.tok == token.COMPOUND {
			var posLp = p.Position()
			tokLp = p.tok

			// &'xxxx' or &"xxxx"
			if name = p.parseExpr(ctx, false); isNil(name) {
				erro(ctx, "parsed name is nil").at(posLp).debug(1)
			} else if name.expandible(ctx, expandClosure) {
				erro(ctx, "name '%v' (%T) is closured (project=%v)", name, name, p.Project()).at(name.Position()).debug(1)
			} else if nameStr, obj, okay = resolveObject(posLp, tokLp, name); !okay {
				erro(ctx, "name '%v' is unidentified", name).at(name.Position()).debug(1)
			}
		} else {
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v`, `%v` or quotes", token.LPAREN, token.LBRACE).at(position).debug(1)
			return MakeNil(position)
		}
	}

	if isNil(obj) && p.Project().plugin != nil && p.Project().pluginScope != nil {
		if nameStr == "" && !isNil(name) { nameStr = name.Strval(ctx) }
		if nameStr == "" {
			erro(ctx, "strval name '%v' is empty", name).at(name.Position()).debug(1)
		} else {
			obj = p.Project().pluginScope.Lookup(nameStr)
		}
	}

	if position := p.loc(pos); tok.IsDelegate() {
		if isNil(obj) { erro(ctx, "resolved '%v' is nil (%T %v, tok=%v)",
			name, resolved, resolved, tok).at(name.Position()).debug(1) }
		return MakeDelegate(position, tokLp, obj, rest...);
	} else {
		if isNil(obj) { erro(ctx, "resolved '%v' is nil (%T %v), shall be 'unresolved' (tok=%v)",
			name, resolved, resolved, tok).at(name.Position()).debug(1) }
		return MakeClosure(position, tokLp, obj, rest...);
	}
}

func (p *parser) parseSpecialClosureDelegate(ctx Context, lhs bool) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "SpecialClosureDelegate")) }

	var obj Object
	var resolved Value
	var pos, tok, s = p.pos, p.tok, p.lit
	var position = p.loc(pos)
	p._next(ctx)


	if c := s[0]; /*p.bits&parsingDefineClause != 0*/true &&
		len(s) == 1 && (('0' <= c && c <= '9') /*|| c == '_'*/) {
		var scope = p.Scope()
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
		if obj == nil && p.bits&parsingTemplateBlock != 0 {
			if _, resolved = p.resolveObject(w); resolved != nil {
				obj, _ = resolved.(Object)
			}
		}
	DashNxt:
	} else if _, resolved = p.resolveObject(w); resolved == nil {
		erro(ctx, "'%v' is undefined (autos: %v)", s, p.autos).at(position).debug(16)
		return MakeNil(position)
	} else if def, y := resolved.(Caller); def == nil || !y {
		erro(ctx, "'%v' is not callable: %T", s, resolved).of(resolved).debug(6)
		return MakeNil(position)
	} else if obj, y = def.(Object); !y {
		erro(ctx, "'%v' is not object: %T", s, def).of(resolved).debug(6)
		return MakeNil(position)
	}

	if isNil(obj) {
		erro(ctx, "resolved '%v' is <nil>: %v (%T)", s, resolved, resolved).at(position).debug(1)
		return MakeNil(position)
	} else if tok.IsDelegate() {
		return MakeDelegate(position, tok, obj);
	} else {
		return MakeClosure(position, tok, obj);
	}
}

func (p *parser) parseUnaryExpr(ctx Context, lhs bool) (x Value) {
	if t_traverse.enabled && false { defer un(trace(t_traverse, "Unary")) }

	switch p.tok {
	case token.BAREWORD, token.AT:
		return p.parseBarewordConstant(ctx, lhs)

	case token.BIN, token.OCT, token.INT, token.HEX, token.FLOAT,
		token.DATETIME, token.DATE, token.TIME, token.URI,
		/*token.RAW,*/ token.STRING, token.ESCAPE:
		return p.parseBasicLit(ctx, lhs)

	case token.COMPOUND:
		return p.parseCompoundLit(ctx, lhs)

	case token.DELEGATE, token.CLOSURE: // delegate, closure
		return p.parseClosureDelegate(ctx)

	case token.LPAREN:
		return p.parseGroupExpr(ctx, lhs)

	case token.TILDE, token.DOT, token.DOTDOT: // ~ . ..
		var str = p.tok.String()
		tok, pos, end := p.tok, p.pos, p.pos+token.Pos(len(str))
		position := p.loc(pos)
		if p._next(ctx); end != p.pos { // FIXME: ~user
			// '~', '.' or '..' used as bareword
			return MakeBareword(position, str)
		} else if p.tok == token.PCON { // check /
			return p.parsePathExpr(ctx, lhs, makePathSeg(positional(ctx, position), tok))
		} else if tok == token.DOT || tok == token.DOTDOT { // TODO: parse to Qualiword instead
			if x = MakeBareword(p.loc(pos), str); p.bits&composingDOT == 0 {
				x = p.parseDotExpr(ctx, lhs, x)
			}
			return
		} else if tok == token.TILDE { // TODO: ~user
			return makePathSeg(positional(ctx, position), tok)
		} else {
			erro(ctx, "unexpected path: %v", tok).at(position).debug(1)
			return MakeNil(position)
		}

	case token.PCON: // The root of the path
		return p.parsePathExpr(ctx, lhs, makePathSeg(ctx, p.tok))

	case token.LBRACK:
		return p.parseModifiersExpr(ctx)

	case token.STAR, token.QUE/*, token.LBRACK*/: // * ? [
		return p.parseGlobExpr(ctx, nil) // (ie. no prefix)

	case token.PERC: // %bar (ie. no prefix)
		return p.parsePercExpr(ctx, lhs, nil)

	case token.LBRACE: // TODO: regexp: {^.*}   or token.REGEXP
		return p.parseRegexpExpr(ctx)

	case token.MINUS:
		return p.parseFlagExpr(ctx, lhs)

	case token.EXC:
		return p.parseNegExpr(ctx, lhs)

	case token.SEMICOLON, token.BAR:
		return p.parsePunctuation(ctx)

	default:
		if p.tok.IsClosure() || p.tok.IsDelegate() {
			return p.parseSpecialClosureDelegate(ctx, lhs)
		} else if p.tok.IsKeyword() { // keywords here are barewords
			return p.parseBarewordConstant(ctx, lhs)
		}
	}

	prompt(ctx, "%v: bad unary '%v' (lit=%s,lhs=%v)\n", p.file.Name(), p.tok, p.lit, lhs)
	if p.lineComment == nil {
		erro(ctx, "bad unary expression '%v'", p.tok).debug(32)
	} else {
		for _, comment := range p.lineComment.List {
			erro(ctx, "# %s", comment.Text).at(comment.Pos)
		}
		erro(ctx, "bad unary expression '%v'", p.tok).debug(32)
	}
	p._next(ctx) // go to the next token
	return MakeNil(p.Position())
}

func (p *parser) parseComposedExpr(ctx Context, lhs bool) (x Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Composed")) }

	switch x = p.parseUnaryExpr(ctx, lhs); p.tok { // check composible expressions
	case token.SELECT_PROP, token.SELECT_PROG1, token.SELECT_PROG2: // foo->bar  foo=>bar  foo~>bar
		if p.bits&composingNoSelect == 0 {
			// accepts 'foo=>bar', but 'foo => bar' is different
			x = p.parseSelect(ctx, x); break
		}

	case token.LBRACK: // xxx[(foo ...)]
		if p.bits&composingModifier == 0 {
			// FIXME: compose lhs x
			m := p.parseModifiersExpr(ctx)
			erro(ctx, "composing modifiers is ignored (unimplemented yet)").of(m)
		}
	case token.STAR, token.QUE/*, token.LBRACK*/: // foo*bar foo?bar foo[a-z]bar
		if p.bits&composingNoGlob == 0 {
			x = p.parseGlobExpr(ctx, x)
		}
	case token.PERC: // foo%bar
		// FIXME: %/foo/bar -> Path(% foo bar)
		if p.bits&composingNoPerc == 0 {
			x = p.parsePercExpr(ctx, lhs, x)
		}
	case token.DOT: // foo.bar.baz.o
		// FIXME: push bits when parsing $(...)
		if p.bits&composingDOT == 0 { // TODO: parse to Qualiword
			x = p.parseDotExpr(ctx, lhs, x)
		}
	case token.PCON: // ie. subdir/in/somewhere
		if p.bits&composingNoPath == 0 {
			// Path expressions, except '-I/path/to/include'
			switch x.(type) {
			case *Flag: // By pass expressions like -I/foo/bar.
			default: x = p.parsePathExpr(ctx, lhs, x)
			}
		}
	case token.COLON:
		if (p.bits&parsingRecipe != 0 || !lhs) && p.bits&composingNoURL == 0 {
			if isKnownURLScheme(x.Strval(positional(ctx, p.Position()))) {
				x = p.parseURLExpr(ctx, lhs, x)
			}
		}
	}
	return
}

func (p *parser) parseText(ctx Context) (res []Value) {
	if false && t_traverse.enabled { defer un(trace(t_traverse, "Text")) }
	for p.tok != token.EOF {
		if p.tok == token.SPACE { p.next(ctx, true) } else {
			res = append(res, p.parseExpr(ctx, false))
			if p.checkErrors(true) > 0 {
				warn(ctx, "parse text got %d errors", p.totalErrors()).debug(16)
				if options.failOnErrors { fail(p.Position(), "fail by %d errors", p.totalErrors()) }
			}
		}
	}
	return
}

func (p *parser) parseExpr(ctx Context, lhs bool) (x Value) {
	if false && t_traverse.enabled { defer un(trace(t_traverse, "Expression")) }

	var tok, lit = p.tok, p.lit
	if x = p.parseComposedExpr(ctx, lhs); isNil(x) {
		erro(ctx, "%v: invalid expression (tok=%v, lit=%v)", p.Project(), tok, lit).debug(6)
		return
	} else if lhs && p.tok.IsAssign() { return }

SwitchCompose:
	switch p.tok {
	case token.ASSIGN: // Example: '*.o = obj'
		if !lhs && p.bits&composingNoPair == 0 {
			x = p.parseKeyValueExpr(ctx, x)
		}
		return

	case token.SELECT_PROP, token.SELECT_PROG1, token.SELECT_PROG2:
		if p.bits&composingNoSelect == 0 {
			x = p.parseSelect(ctx, x)
			goto SwitchCompose // For example: foobar⇒run(-gen)
		}
		return

	case token.LPAREN:
		if p.bits&composingNoArg == 0 {
			if false {
				if _, ok := x.(*Argumented); ok { erro(ctx, "nested argumentation") }
			}
			if x = p.parseArgumentedExpr(ctx, x); !isNil(x) {
				goto SwitchCompose
			}
		}
		return

	case token.PCON:
		if p.bits&composingNoPath == 0 {
			// Path expressions, except '-I/path/to/include'
			switch x.(type) {
			case *Flag: // By pass expressions like -I/foo/bar.
			default: x = p.parsePathExpr(ctx, lhs, x)
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

	var y = p.parseComposedExpr(ctx, lhs)
	if _, ok := y.(*Path); ok {
		switch x.(type) {
		case *Flag: // okay: -Ifoo/bar, -Lfoo/bar
		case *Path: // okay: combine two paths
		case *String, *Compound, *delegate, *closure:
		default:
			warn(ctx, "barecomp a path: %v (%T), %v (%T) (next=%v)", x, x, y, y, p.tok).of(y).debug(1)
		}
	}

	// Further composing
	switch t := x.(type) {
	case *Barecomp: t.Combine(ctx, y)
	case *Path: t.Combine(ctx, y)
		if false { info(ctx, "%v (%v) (tok=%v)", t, y, p.tok).at(t.position) }
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
    dontOperate bool // e.g. -cond(false)
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

func (p *parser) _parseUseSpecProps(ctx Context, props []Value) (opts useOpts, params []Value, err error) {
    // Supported parameter forms:
    //      -param
    //      -param(value)
    //      -param=value
	ctx = p.posit(ctx)

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
                erro(ctx, "parameter `%v' unsupported `%T`", prop, prop).of(t.Key)
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
                erro(ctx, "parameter `%v' unsupported `%T`", prop, prop).of(t.value)
                return
            }
        default:
            erro(ctx, "parameter `%v` unsupported `%T`", prop, prop).of(prop)
            return
        }
    }
    return
}

func (p *parser) parseUseSpec(ctx Context, doc *CommentGroup, gOpts *genericClauseOpts, _ int) {
	if p.imports = append(p.imports, &usespec{ gOpts.spec }); gOpts.dontOperate {
		// TODO: maybe give some information
		return
	}

	ctx = p.posit(ctx)

	var (
		args = append(gOpts.vals, gOpts.spec[1:]...)
		specVal = gOpts.spec[0]
        specNames []string
		opts useOpts
	)
	args = parseOpts(ctx, &opts, 0, args...)
	for _, a := range args {
		if _, ok := a.(*Flag); ok || true {
			erro(ctx, "unkown use opts: %T %v", a, a).of(a).debug(1)
			return
		}
	}

	var arged []Value
	var specName string
	switch v := specVal.(type) {
    case *Pair:
        var s string
        if f, ok := v.Key.(*Flag); !ok {
            erro(ctx, "'%v' invalid use spec", v.Key).of(specVal)
            return
        } else if s = f.name.Strval(ctx); s != "list" {
            erro(ctx, "'%v' invalid use spec, do you mean -list?", v.Key).of(specVal)
            return
        }

        for _, val := range mergex(ctx, plain, v.Value) {
            if s = val.Strval(ctx); s == "" { continue }
            specNames = append(specNames, s)
        }
		goto loadSpecNames
	case *Argumented:
		arged, specVal = v.args, v.value
    }

	if specName = specVal.Strval(ctx); specName != "" {
		specNames = append(specNames, specName)
	}

loadSpecNames:
	if len(specNames) == 0 {
        erro(ctx, "empty use spec (%v)", specVal).of(specVal).debug(1)
        return
    }

	var wg sync.WaitGroup
    for _, specName = range specNames {
		var ctx = positional(ctx, specVal.Position())
		if true {
			p.loadUseSpecName(ctx, opts, specVal, specName, arged, args...)
		} else {
			var dc = diagContext{ Context: ctx } // redefine ctx
			wg.Add(1); go func() {
				defer checkPanicsErrors(&dc, true)
				defer func() {
					if len(dc.points) > 0 { dc.inner().diagnostic().nest(dc.points) }
					wg.Done()
				} ()
				p.loadUseSpecName(ctx, opts, specVal, specName, arged, args...)
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
		erro(ctx, "%v errors: use %v", errs, specNames).at(pos).debug(6)
		if true { fail(pos, "%s: use %v failed; %d errors", proj, specNames, errs) }
	}
	return
}

func (p *parser) parseIncludeSpec(ctx Context, doc *CommentGroup, gOpts *genericClauseOpts, _ int) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Spec")) }

	var opts = includeFileOpts{ genericClauseOpts: gOpts }
	if vals := parseOpts(ctx, &opts, 0, gOpts.vals...); len(vals) > 0 {
		// TODO: deal with the unparsed generic options
		warn(ctx, "unknown opts: %v", vals).debug(1)
	}

	if len(gOpts.spec) < 1 {
		erro(ctx, "expecting include file: %v", gOpts.spec).debug(1)
		return
	}

	var x = gOpts.spec[0]
	if p.skipSpaces(ctx); p.tok == token.COLON {
		switch x.(type) {
		case *File, *String, *Compound: // escape from file searching
		default: if file := p.project.FindFile(ctx, x.Strval(p)); file != nil {
			x = file
		} else if val := x.expand(ctx, plain); !isNil(val) && val != x {
			x = val
		}}

		x = p.parseRuleEntry(ctx, specialRuleNor, nil, []Value{x}) // this should return a RuleEntry
	}
	if !gOpts.dontOperate { p.includeFile(ctx, opts, x) }
}

func (p *parser) importFileMaps(ctx Context, public bool, paths ...Value) {
	if options.noImportFiles { return }

	var (
		opts = useOpts{ noVars:true, reuse:true, public:public }
		projects []*Project
		projMutx sync.Mutex
		wg sync.WaitGroup
	)
	for _, val := range paths {
		var (
			ctx = positional(ctx, val.Position())
			name = val.Strval(ctx)
		)
		if false { // FIXME: parellel loading failed
			wg.Add(1); go func() {
				defer checkPanicsErrors(ctx, true)
				defer wg.Done()
				var loaded = p.loadUseSpecName(ctx, opts, val, name, nil)
				projMutx.Lock()
				projects = append(projects, loaded)
				projMutx.Unlock()
			} ()
		} else {
			var loaded = p.loadUseSpecName(ctx, opts, val, name, nil)
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
					fm = &FileMap{ fm.project, fm.patts, fm.paths, opts.public }
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
func (p *parser) parseFilesSpec(ctx Context, doc *CommentGroup, gOpts *genericClauseOpts, _ int) {
	defer p.setbits(p.setbit(parsingFilesSpec))
	if len(gOpts.spec) != 1 {
		erro(ctx, "too many files properties: %v", gOpts.spec).debug(1)
		return
	}

	var path Value
	if p.tok == token.SELECT_PROG1 {
		p.next(ctx, true) // step forward with spaces skipped
		if p.tok == token.LINEND || p.lineComment != nil {
			erro(ctx, "expecting files path")
		}
		path = p.parseExpr(ctx, false)
	}
	if p.skipSpaces(ctx); p.lineComment != nil {
		//spec.Comment = p.lineComment
	}
	if gOpts.dontOperate {
		// TODO: maybe give some information
		return
	}

	ctx = p.posit(ctx)

	var (
		val = gOpts.spec[0]
		opts filesOpts
		pats []Value
	)
	parseOpts(ctx, &opts, 0, gOpts.vals...)

	if g, ok := val.(*Group); ok {
		pats = g.Elems
	} else if val.expandible(ctx, expandClosure) {
		pats = []Value{ val }
	} else {
		pats = mergex(ctx, plain, val)
	}

	if path == nil {
		if len(pats) == 1 { if a, ok := pats[0].(*Argumented); ok { if f, ok := a.value.(*Flag); ok {
			var name = f.name.Strval(ctx) // -import(paths...)
			switch name {
			case "import": p.importFileMaps(ctx, opts.public, a.args...); return
			default:
				erro(ctx, "invalid files flag: %v").of(f.name).debug(1)
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
			p.Project().mapfile(ctx, opts, values(files), nil)
			pats = newPats
		}
		if len(pats) > 0 {
			var paths = []Value{ MakeString(val.Position(), p.Project().absPath) }
			p.Project().mapfile(ctx, opts, pats, paths)
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
			var name = f.name.Strval(ctx) // -import => (paths...)
			switch name {
			case "import": p.importFileMaps(ctx, opts.public, paths...); return
			default:
				erro(ctx, "invalid files flag: %v").of(f.name).debug(1)
				return
			}
		}}
		p.Project().mapfile(ctx, opts, patsNew, paths)
	}
}

func (p *parser) evalConfiguration(ctx Context, gOpts *genericClauseOpts, props []Value) {
	if project := p.Project(); project == nil {
		erro(ctx, "configuration: nil project").debug(1)
	} else if project.configure == nil {
		erro(ctx, "configuration: no .configure for %v", project).debug(1)
	} else {
		if entry := project.configure.DefaultEntry(); entry == nil {
			// no init entry from .configure
		} else if _, traves := entry.Execute(ctx); len(traves) > 0 {
			for _, brk := range traves {
				if brk.what == traveFail {
					erro(ctx, "execute '%v' failed: %v", entry, brk).of(entry).debug(1)
				}
			}
		}
		if project.configured {
			prompt(ctx, "configuration: %v already configured\n", project)
		} else {
			var (
				okay bool
				cp *Project
				ce = &configureExecutor{ defs:make(map[string]*def) }
			)
			defer ce.close()
			for _, dep := range mergex(ctx, plain, props[1:]...) {
				switch prereq := dep.(type) {
				case *RuleEntry:
					if _, traves := prereq.Execute(ctx); len(traves) > 0 {
						for _, brk := range traves {
							if brk.what == traveFail {
								erro(ctx, "execute '%v' failed: %v", prereq, brk).of(prereq).debug(1)
							}
						}
					}
				default:
					erro(ctx, "prerequisite: unsupported %T %v", dep, dep).debug(1)
				}
			}
			for _, entry := range project.configs {
				if cp, okay = ce.execute(ctx, cp, entry); !okay {
					erro(ctx, "configure '%v' failed", entry).debug(1)
					break
				}
			}
			project.configured = true // let defaultContext.configure skip
		}
	}
}

func (p *parser) parseAssertSpec(ctx Context, doc *CommentGroup, gOpts *genericClauseOpts, _ int) {
	if !gOpts.dontOperate { assertion(p.posit(ctx), gOpts.generalOpts, gOpts.spec...) }
}

func (p *parser) parseEvalSpec(ctx Context, doc *CommentGroup, gOpts *genericClauseOpts, _ int) {
	var (
		prop0, resolved Value
		name string
	)

	if gOpts.dontOperate { return }
	if prop0 = gOpts.spec[0]; isNil(prop0) {
		erro(ctx, "illegal").debug(1)
		return
	}

	var position = prop0.Position()
	if !position.IsValid() {
		erro(ctx, "command name '%v' has invalid position", prop0).debug(1)
		return
	} else {
		ctx = positional(ctx, position)
	}

	if name, resolved = p.resolveObject(prop0); false {
		erro(ctx, "resolve '%v' failed", prop0).debug(1)
		return
	} else if isTrivial(resolved) {
		if name == "configuration" {
			// NOTE: see also defaultContext.configure()
			p.evalConfiguration(positional(ctx, position), gOpts, gOpts.spec)
			return
		}
		erro(ctx, "resolved '%v' is nil (options = %v)", prop0, *gOpts).debug(1)
		return
	}

	if b, ok := resolved.(*Builtin); !ok {
		erro(ctx, "resolved '%v' is not a command (%s)", prop0, typeof(resolved)).debug(1)
		return
	} else if !b.s.command {
		erro(ctx, "resolved builtin '%v' is not a command", prop0).debug(1)
		return
	}

	//p.evalspec(spec)

	// At the point of `eval` was represented, the closure context
	// might be empty. So we start closure with the current scope.
	//defer setclosure(setclosure(cloctx.unshift(p.scope)))
	var res Value
	switch op := prop0.(type) {
	case Caller: res = op.Call(ctx, gOpts.spec[1:]...)
	default:
		if name := op.Strval(ctx); name == "" {
			erro(ctx, "empty op: %T %s", op, op).debug(1)
		} else if _, obj := p.Scope().Find(name); obj == nil {
			erro(ctx, "`%s` undefined", name).debug(1)
		} else if f, _ := obj.(Caller); f == nil {
			erro(ctx, "`%T` is not caller (%s)", obj, name).debug(1)
		} else {
			res = f.Call(ctx, gOpts.spec[1:]...)
		}
	}

	if ctx.checkErrors(true); isTrivial(res) {
		return
	} else if false/*TODO: c, y := res.(code); y */ {
		// TODO: evalue code result
	}
}

func (p *parser) parseDirectiveSpec(ctx Context) (props []Value) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Spec")) }

	//var doc = p.leadComment
	var comment *CommentGroup

ParamsParseLoop: // Parse the directive parameters
	for p.tok != token.EOF {
		switch p.skipSpaces(ctx); p.tok {
		case token.COMMA, token.LINEND, token.RPAREN, token.RBRACE, token.RCOLON,
			token.SELECT_PROG1, token.COLON: break ParamsParseLoop
		}

		if p.lineComment != nil {
			comment = p.lineComment
			break
		}

		props = append(props, p.parseExpr(ctx, false))
	}
	if comment != nil { /* TODO: directive documments */ }
	return
}

func (p *parser) parseGenericClause(ctx Context, keyword token.Token, pos token.Pos, f parseSpecFunc) {
	if t_traverse.enabled { defer un(trace(t_traverse, "Clause("+keyword.String()+")")) }

	var opts = genericClauseOpts{ keyword: keyword }
	for p.skipSpaces(ctx); p.tok == token.MINUS; p.skipSpaces(ctx) {
		opts.all = append(opts.all, p.parseExpr(ctx, false))
	}
	opts.vals = parseOpts(ctx, &opts, expandZero, opts.all...)

	for _, cond := range opts.conds {
		if t := cond.True(positional(ctx, cond.Position())); !t {
			opts.dontOperate = true
			break
		}
	}

	if p.skipSpaces(ctx); p.tok == token.LPAREN {
		p.next(ctx, true)
		for iota := 0; p.tok != token.RPAREN && p.tok != token.EOF && (p.stop == 0 || p.pos < p.stop); iota++ {
			// TODO: collect documentation comments
			for p.tok == token.SPACE || p.tok == token.LINEND { p.next(ctx, true) }
			if p.tok == token.RPAREN || p.tok == token.EOF { break  }
			if opts.spec = p.parseDirectiveSpec(ctx); true {
				f(p.posit(ctx), p.leadComment, &opts, iota)
			}
			if p.tok == token.COMMA || p.tok == token.LINEND { p.next(ctx, true) }
		}
		if p.expect(ctx, token.RPAREN); p.tok != token.EOF { p.expectLinend(ctx) }
		return
	}

	if p.tok != token.LINEND && p.tok != token.EOF && (p.stop == 0 || p.pos < p.stop) {
		if opts.spec = p.parseDirectiveSpec(ctx); true {
			f(p.posit(ctx), nil, &opts, 0)
		}
		if p.tok == token.COMMA { p.next(ctx, true) }
	}
	if p.tok != token.EOF && (p.stop == 0 || p.pos < p.stop) {
		if p.lineComment == nil { p.expectLinend(ctx) }
	}
}

func (p *parser) parseDefineClause(ctx Context, tok token.Token, ident Value) (def *def) {
	if t_traverse.enabled { defer un(trace(t_traverse, fmt.Sprintf("Define(%s)", ident))) }

	// Only accept scoped identifiers if it's ":user:" program
	if p.Scope().comment == usecomment {
		switch i := ident.(type) {
		case *selection:
			erro(ctx, "should use scoped names instead of `%v`", i).of(ident)
		default:
			erro(ctx, "FIXME: unexpected name expression: %T", i).of(ident)
		}
		return
	}

	var savedBits = p.bits
	p.bits |= parsingDefineClause

	var (
		savedAutos = p.autos // save for $1, $2, $3, etc...
		// TODO: doc = p.leadComment
		// TODO: comment = p.lineComment
		position = p.loc(p.expect(ctx, tok))
		elems = p.parseRhsList(ctx)
		value Value
	)
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

	// NOTE: forcely put all explicit defs into project scope. It's important for defs enclosed
	//       in templates work.
	defer func(s *Scope) { p.scopes[0] = s } (p.Scope())
	p.scopes[0] = p.Project().scope

	var defs = p.define(positional(ctx, position), tok, ident, value)
	if n := len(defs); n > 0 {  def = defs[n-1] }
	return
}

func (p *parser) parseDefine(ctx Context, ident Value) (def *def) {
	return p.parseDefineClause(ctx, p.tok, ident)
}

func (p *parser) parseRecipeDefineClause(ctx Context, x Value) Value {
	// TODO: validate x ...
	return p.parseDefineClause(ctx, p.tok, x)
}

func (p *parser) parseRecipeRuleClause(ctx Context, elems []Value) (x Value) {
	return p.parseRuleEntry(ctx, specialRuleRec, nil, elems)
}

func (p *parser) parseRecipeExpr(ctx Context) Value {
	if t_traverse.enabled { defer un(trace(t_traverse, "Recipe")) }

	var (
		// TODO: comment *CommentGroup
		// TODO: doc = p.leadComment
		position = p.Position()
		elems []Value
		isList bool
	)

SwitchDialect:
	switch p.dialect {
	case "", "eval", "value":
		p.scanner.LeaveCompoundLineContext()
		p.next(ctx, true) // skip RECIPE or SEMICOLON and parse in list mode
		position = p.Position()
		if isList = true; !p.isEndOfLine() {
			defer p.setbit(p.setbit(parsingRecipeBuiltin))

			var (
				isValue = p.dialect == "value"
				x = p.parseExpr(ctx, /*!isValue*/false) // parse first expr of recipe
			)
			if isNil(x) {
				erro(ctx, "parsed value is nil").at(position)
			} else if isValue {
				// no resolving commands
			} else if t, ok := x.(*Bareword); !ok {
				// does nothing
			} else if _, sym := p.resolveObject(t); false {
				erro(ctx, "resolve '%v' failed", x).at(position)
			} else if isTrivial(sym) {
				erro(ctx, "resolved '%v' (from %v) is nil", t.string, x).of(x)
			} else if false {
				erro(ctx, "builtin command no more supported, use $(%s ...) instead", t.string).of(x)
			} else if b, ok := sym.(*Builtin); !ok {
				erro(ctx, "'%s' is not a command (%s)", t.string, typeof(sym)).of(x)
			} else if !b.s.command {
				erro(ctx, "'%s' is not a command, use $(%s ...) instead", t.string, t.string).of(x)
			} else {
				x = sym
			}

			if !isValue && p.tok.IsAssign() {
				elems = append(elems, p.parseRecipeDefineClause(ctx, x))
				break SwitchDialect
			}
			elems = append(elems, x)

			var cmdargs []Value
			for p.tok != token.EOF && p.tok != token.SEMICOLON && p.tok != token.LINEND && p.lineComment == nil {
				if p.skipSpaces(ctx); p.lineComment != nil {
					// TODO: comment = p.lineComment
					break
				}

				if p.tok.IsRuleDelim() {
					if false {
						x = p.parseRecipeRuleClause(ctx, elems) // RuleEntry
					} else {
						erro(ctx, "unsupported token: %s, %v", p.tok, elems).debug(1)
					}
				} else {
					x = p.parseExpr(ctx, false)
				}

				cmdargs = append(cmdargs, x)
				if p.tok == token.COMMA {
					p.next(ctx, true)
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
		p.next(ctx, true) // skip RECIPE or SEMICOLON and parse in line-string mode
		position = p.Position()
		for !p.isEndOfLine() {
			var bits = p.setbit(parsingRecipeText)
			var x Value
			switch p.tok {
			default:           x = p.parseExpr(ctx, false)
			case token.RAW:    x = p.parseBasicLit(ctx, false)
				/*
			case token.LINEND:
				erro(ctx, "unexpected end of line for compound string")
				break ForCompound*/
			}
			elems = append(elems, x)
			p.setbits(bits)
		}
	}
	if p.tok != token.EOF { p.expectLinend(ctx) }
    if len(elems) == 0 {
        return MakeNone(position)
    } else if isList {
        return MakeList(position, elems...)
    } else {
        return MakeCompound(position, elems...)
    }
}

func (p *parser) parseVarModifier(ctx Context, args []Value) (err error) {
	// Parsing (var a=xxx,b=yyy) definitions
	for _, elem := range args[1:] {
		var kv, ok = elem.(*Pair)
		if !ok || kv == nil {
			erro(ctx, "bad var form (%T)", elem).of(elem)
			continue
		}

		var name string
		var k, v = kv.Key, kv.Value
		if name = k.Strval(positional(ctx, k.Position())); name == "" {
			erro(ctx, "name '%v' is empty", k).of(k)
		}
		if def, alt := p.def(elem.Position(), name); alt != nil {
			erro(ctx, "Def '%v' already existed: %T", name, alt).of(k)
		} else if def != nil {
			var ctx = positional(ctx, v.Position())
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
	for _, t := range p.targets {
		var pos = t.Position()
		if !pos.IsValid() { pos = p.Position() }

		var (
			ctx = positional(ctx, pos)
			name = t.Strval(ctx)
		)

		var d, a = p.project.scope.define(ctx, /*defVoid*/DefConfig, name, nil)
		if d == nil && a != nil { if d, _ = a.(*def); d == nil {
			erro(ctx, "configure %v: already defined in '%v' as %v", t, p.project, a).debug(6)
			return
		}}
		if !d.position.IsValid() { d.position = pos }
	}
}

func (p *parser) parseModifyParams(ctx Context, args []Value) (err error) {
	for _, elem := range args {
		var ctx = positional(ctx, elem.Position())
		switch elem.(type) {
		case *Bareword, *Barecomp:
			var s = elem.Strval(ctx)
			var d, a = p.def(elem.Position(), s)
			if a != nil {
				var y bool
				if d, y = a.(*def); !y {
					erro(ctx, "%T '%s' already taken the name, no such parameter", a, s).of(elem)
				}
			}
			if d != nil {
				d.set(ctx, DefArg, nil)
			} else {
				erro(ctx, "'%s' is not defined", s).of(elem)
			}
			p.params = append(p.params, d)
			p.Scope().replace(ctx, strconv.Itoa(len(p.params)), d)
		default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
			erro(ctx, "bad parameter form (%T)", elem).of(elem)
		}
	}
	return
}

func (p *parser) parseModifiersExpr(ctx Context) *modifiergroup {
	if t_traverse.enabled { defer un(trace(t_traverse, "Modifiers")) }

	var (
		posLp = p.loc(p.expect(ctx, token.LBRACK))
		hasParameters bool // ((foo bar))
		elems []*modifier
	)

	defer func(a parsingBits) { p.bits = a }(p.bits)
	p.bits |= composingModifier

ForModifiersExpr:
	for p.tok != token.RBRACK && p.tok != token.EOF {
		p.skipSpaces(ctx)

		var (
			x = p.parseExpr(ctx, false)
			group *Group
			name string
		)
		if g, ok := x.(*Group); !ok {
			var xv = x.expand(ctx, expandDelegate/*TODO: expandInline or expandAuto*/)
			warn(ctx, "modifier: %T %v   →   %T %v", x, x, xv, xv).at(x.Position()).debug(1)
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
				p.parseVarModifier(ctx, group.Elems)
				continue ForModifiersExpr
			} else if name == "configure" {
				p.defineConfigureTargets(ctx)
				p.configure = true // set configure flag and define configure variables
			}
			goto checkNameAndAdd
		case *Group: // parameters: ((foo bar))
			hasParameters = true
			p.parseModifyParams(ctx, n.Elems)
			continue ForModifiersExpr
		case *delegate, *closure, *Barecomp, *String:
			var ctx = positional(ctx, n.Position())
			var v = mergex(ctx, plain, n)
			if name = v[0].Strval(ctx); name == "" {
				erro(ctx, "name '%v' is empty", n).of(n).debug(1)
				continue ForModifiersExpr
			}
			goto checkNameAndAdd
		default:
			erro(ctx, "unsupported dialect or modifier (%T): %v", group.Elems[0], group.Elems[0]).of(n).debug(1)
			continue ForModifiersExpr
		}

		goto addModifier

	checkNameAndAdd:
		if _, ok := dialects[name]; ok {
			if p.dialect == "" { p.dialect = name } else {
				erro(ctx, "multi-dialects unsupported, already defined '%s'", p.dialect).of(x).
					debug(1)
				continue ForModifiersExpr
			}
		} else if _, ok = modifiers[name]; !ok {
			erro(ctx, "`%s` no such dialect or modifier", name).of(x).debug(1)
			continue ForModifiersExpr
		}

	addModifier:
		if len(group.Elems) == 0 {
			erro(ctx, "empty modifier: %v", x).of(x).debug(1)
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
	p.skipSpaces(ctx)
	/*rpos := */p.expect(ctx, token.RBRACK)
	if len(elems) == 0 && !hasParameters {
		erro(ctx, "empty modifier group").at(posLp).debug(1)
	}
	if p.tok == token.COLON {
		erro(ctx, "unexpected colon after modifer").at(posLp).debug(1)
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

func (p *parser) parseRuleEntry(ctx Context, special specialRule, options, targets []Value) (result Value) {
	if ctx = p.posit(ctx); p.Project().keyword == token.PACKAGE {
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
	case specialRuleUse:
		scopeComment = fmt.Sprintf(usecomment)
	default:
		scopeComment = fmt.Sprintf("rule %v", targets)
	}
	defer p.closeScope(p.openScope(scopeComment))
	p.params = nil
	p.dialect = ""

	var position = ctx.Position()
	for _, s := range automatics {
		var def, alt = p.def(position, s)
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
		var def, alt = p.def(position, strconv.Itoa(i))
		if alt != nil {
			erro(ctx, "name `%v` already taken, not numberred (%T).", i, alt)
		} else if def == nil {
			erro(ctx, "'$%d' is not defined", i)
		} else {
			def.origin = DefAuto
		}
	}

	switch special {
	case specialRuleUse:
		if name, alt := p.Scope().ProjectName(ctx, selfproj, p.Project()); alt != nil {
			erro(ctx, "name `%s` already taken, not automatic (%T)", selfproj, alt)
		} else if name == nil {
			erro(ctx, "cannot define `%s` automatic", selfproj)
		}
		if name, alt := p.Scope().ProjectName(ctx, userproj, nil); alt != nil {
			erro(ctx, "name `%s` already taken, not automatic (%T)", userproj, alt)
		} else if name == nil {
			erro(ctx, "cannot define `%s` automatic", userproj)
		}
	}

	// NOTE: expand targets to speed up for later usage, it might spend lots of time in
	// project.entry while matching for entry looked up if not expanded right now.
	targets, _, _ = expand(ctx, plain & ^expandArgedArgs, targets...)

	defer func(t []Value) { p.targets = t } (p.targets)
	p.next(ctx, true) // skip rule delimeters and spaces
	p.targets = targets // save targets for later refering

	if p.tok != token.SEMICOLON && p.tok != token.BAR && !p.isEndOfLine() {
		depends = p.parseDependList(ctx)
	}
	if p.tok == token.BAR {
		p.next(ctx, true) // '|' starts the ordered prerequisites
		if p.tok != token.SEMICOLON && !p.isEndOfLine() {
			ordered = p.parseDependList(ctx)
		}
	}

	if p.tok == token.SEMICOLON { // ;
		// Parse inline recipe in the program scope.
		recipes = append(recipes, p.parseRecipeExpr(ctx))
	} else /*if p.tok == token.LINEND || p.lineComment != nil*/ {
		// Parse recipes in the program scope.
		p.scanner.TurnRecipesOn() // Turn on recipes before LINEND
		if p.expectLinend(ctx) /* take the new line */ {
			for p.tok != token.EOF && p.isRecipeStart(ctx) {
				recipes = append(recipes, p.parseRecipeExpr(ctx))
			}
		}
		p.scanner.TurnRecipesOff()
	}

	var params []string
	if t := targets[0]; p.configure {
		name := t.Strval(ctx)
		d, a := p.Project().scope.define(ctx, DefVoid, name, nil)
		if d == nil && a == nil {
			erro(ctx, "cannot define configure target '%v'", name).of(t)
		} else if a != nil {
			if _, ok := a.(*def); !ok {
				erro(ctx, "configure target '%v' already taken: %T %v", name, a, a).of(t)
			}
		}
		if d != nil && !d.position.IsValid() {
			d.position = t.Position()
		}
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

func (p *parser) parseSpecialRuleClause(ctx Context) Value {
	if t_traverse.enabled {
		defer un(trace(t_traverse, "SpecialRule"))
	}

	p.expect(ctx, token.COLON) // expect and skip ':'

	if p.tok != token.BAREWORD {
		erro(ctx, "unknown special rule")
		return nil
	}

	var name = p.lit
	switch name {
	case "user":
		if true {
			// Example usage of use.*:
			//    use.* ::= cflags(-unique) ldlibs(-unique -reverse)
			erro(ctx, ":user: rules are deprecated, use use.* instead!").debug(1)
		} else {
			var options []Value
			var pos = p.expect(ctx, token.BAREWORD) // USE
			var bits = p.setbit(parsingSpecialRule)
			// Options are *Flag or *Pair of a Flag.
			for p.tok == token.MINUS {
				opt := p.parseExpr(ctx, false)
				options = append(options, opt)
			}
			p.setbits(bits) // restore bits
			if p.tok.IsRuleDelim() {
				return p.parseRuleEntry(ctx, specialRuleUse, options, []Value{
					MakeBareword(p.loc(pos), name),
				})
			}

			erro(ctx, "expecting special rule terminator ':'")
		}
		return nil
	default:
		erro(ctx, "unknown special rule")
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

	defer p.closeScope(p.openScope("template block"))

	var position = p.Position()
	ctx = positional(ctx, position)

	if true { /* NOTE: vars has definition for "_" */ } else
	if def, alt := p.def(position, "_"); alt != nil {
		erro(ctx, "name `_' already taken, not automatic (%T).", alt)
	} else if def == nil {
		erro(ctx, "'_' can not be defined")
	} else {
		assert(def.value == nil, "initial automatic values must be nil")
		def.origin = DefAuto
	}

	for s, v := range vars {
		var def, alt = p.def(p.Position(), s)
		if alt == nil { def.set(ctx, DefAuto, v) } else {
			erro(ctx, "variable '%s' already taken", s).at(p.Position()).debug(1)
		}
	}

	var savedBits = p.bits
	p.bits |= parsingTemplateBlock
	for p.tok != token.EOF && p.pos < p.stop {
		if p.tok == token.LINEND || (p.tok == token.COMMENT && p.lineComment != nil) {
			p.next(ctx, true)
		} else {
			p.parseClause(ctx)
		}
	}
	p.bits = savedBits
}

func (p *parser) templateExpand(ctx Context, t *template, params []Value) {
	var count int64
	defer func(t time.Time, pos token.Pos, tok token.Token, lit string, state scanner.ScanState) {
        if d := time.Now().Sub(t); d > 1999*time.Millisecond {
			var c = time.Duration(count)
            infostack(ctx, 3, "slow: %v, %d * %v, prof-%d", d, count, d/c, pprofCounter).debug(1)
        }
		p.pos, p.tok, p.lit	 = pos, tok, lit
		p.scanner.SetState(state)
	} (time.Now(), p.pos, p.tok, p.lit, p.scanner.State())

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
				erro(ctx, "unexpected value: %T %v", a, a).of(a).debug(1)
				return
			} else if s = pair.Key.Strval(positional(ctx, pair.Key.Position())); s == "" {
				erro(ctx, "empty key: %T %v", pair.Key, pair.Key).of(a).debug(1)
				return
			} else if g, ok := pair.Value.(*Group); ok {
				pos = pair.Value.Position()
				elems = g.Elems
			} else {
				pos = pair.Value.Position()
				elems = append(elems, pair.Value)
			}

			var m = vars[s]
			m.elems = mergex(positional(ctx, pos), plain, elems...)
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
		erro(ctx, "expand template %v: %v", t.verb, params).at(p.Position()).debug(1)
	}
}
func (p *parser) callTemplate(ctx Context, t *template, name Value, args []Value) {
	var count int64
	defer func(t time.Time, pos token.Pos, tok token.Token, lit string, state scanner.ScanState) {
        if d := time.Now().Sub(t); d > 1999*time.Millisecond {
			var c = time.Duration(count)
            infostack(ctx, 3, "%v: slow: %v, %v, %d*%v", name, d, count, d/c).debug(1)
        }
		p.pos, p.tok, p.lit	 = pos, tok, lit
		p.scanner.SetState(state)
	} (time.Now(), p.pos, p.tok, p.lit, p.scanner.State())

	p.scanner.SetState(t.state)
	p.pos, p.tok, p.lit = t.pos, t.tok, t.lit

	// NOTE: a new scope is required for template expansion
	defer p.closeScope(p.openScope("template call "))

	var params = merge(t.params...)
	for i, param := range params {
		var s = param.Strval(ctx)
		if def, alt := p.def(p.Position(), s); alt != nil {
			erro(ctx, "duplicated parameter '%s'", s).at(param.Position()).debug(1)
		} else if i < len(args) {
			def.set(ctx, DefAuto, args[i])
		}
	}

	for p.tok != token.EOF && p.pos < t.endPos {
		if p.tok == token.LINEND ||
			(p.tok == token.COMMENT && p.lineComment != nil) {
			p.next(ctx, true)
		} else {
			p.parseClause(ctx)
		}
	}
}
func (p *parser) templateCall(ctx Context, name Value, args []Value) {
	for _, tmpl := range p.templates {
		if tmpl.name != nil && tmpl.name.cmp(ctx, name) == cmpEqual {
			p.callTemplate(positional(ctx, tmpl.name.Position()), tmpl, name, args)
			return
		}
	}
	erro(ctx, "undefined template: %v", name).of(name).debug(1)
}
func (p *parser) parseTemplateClause(ctx Context) {
	var starting = p.Position()
	p.expect(ctx, token.TEMPLATE) // expect and skip 'template'
	p.skipSpaces(ctx)
	if false { ctx = p.posit(ctx) }

	var (
		verb string
		arged *Argumented
		op = p.parseExpr(ctx, false)
	)
	if p.skipSpaces(ctx); p.tok == token.EOF {
		erro(ctx, "unexpected end of file after %v", op).of(op).debug(1)
		return
	} else if w, ok := op.(*Bareword); ok {
		verb = w.string
	} else if arged, ok = op.(*Argumented); !ok {
		erro(ctx, "unknown template verb: %v", op).of(op).debug(1)
		return
	}

	switch verb {
	case "end", "expand":
		erro(ctx, "unexpected verb: %s", verb).of(op).debug(1)
		return
	case "": if arged != nil {
		p.expect(ctx, token.LINEND)
		p.templateCall(ctx, arged.value, arged.args)
		return //true
	}}

	var params = p.parseExprList(ctx, false)
	// DONT: p.expect(ctx, token.LINEND)

	// TODO: parse template options - parseOpts
	params = mergex(ctx, plain, params...)

	var tmpl = &template{ state:p.scanner.State(), pos:p.pos, tok:p.tok, lit:p.lit }
	if verb == "def" {
		if len(params) != 1 {
			erro(ctx, "too many def params: %v", params).at(starting)
			return
		} else if arged, ok := params[0].(*Argumented); !ok {
			erro(ctx, "too many def params: %v", params).at(starting)
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
		var newline = p.tok == token.LINEND || p.lineComment != nil
		if p.lineComment == nil { p._next(ctx) } //else { newline = true }
		if !newline || p.tok != token.TEMPLATE {
			// info(ctx, "unexpected %v: %v (verb=%s, nested=%v, newline=%v)",
			// 	p.tok, p.lit, verb, nested, newline).debug(1)
			continue
		}

		var pos, stop = p.pos, p.stop
		if p.next(ctx, true); p.tok != token.BAREWORD {
			erro(ctx, "unexpected %v: %v (nested=%v)",
				p.tok, p.lit, nested).debug(1)
			return
		}

		if p.lit == "def" || p.lit == "for" || p.lit == "foreach" {
			nested += 1
		} else if p.lit == "expand" && (verb == "for" ||
			verb == "foreach") { if nested > 0 {
			nested -= 1
		} else {
			p.next(ctx, true) // consumes the 'expand'
			params := p.parseExprList(ctx, false)
			p.expect(ctx, token.LINEND)
			p.stop = pos
			p.templateExpand(ctx, tmpl, params)
			p.stop = stop
			return //true
		}} else if p.lit == "end" && (verb == "def") { if nested > 0 {
			nested -= 1
		} else {
			p.next(ctx, true) // consumes the 'end'
			p.expect(ctx, token.LINEND)
			state := p.scanner.State()
			tmpl.end, tmpl.endPos = &state, pos
			return //true
		}} else if false {
			erro(ctx, "unexpected template: %v (verb=%s, nested=%v)",
				p.tok, verb, nested).debug(1)
			return
		} else {
			continue
		}
	}
}

func (p *parser) parseClause(ctx Context) {
	if false && t_traverse.enabled {
		defer un(tracef(t_traverse, "parseClause(%v, %v)", p.tok, p.pos))
	}

	switch ctx = p.posit(ctx); p.tok {
	case token.USE:
		erro(ctx, "`%v` unexpected here", p.tok).debug(1)
		return
	case token.INCLUDE:
		p.parseGenericClause(ctx, token.INCLUDE, p.expect(ctx, token.INCLUDE), p.parseIncludeSpec)
		return
	case token.FILES:
		p.parseGenericClause(ctx, token.FILES, p.expect(ctx, token.FILES), p.parseFilesSpec)
		return
	case token.ASSERT:
		p.parseGenericClause(ctx, token.ASSERT, p.expect(ctx, token.ASSERT), p.parseAssertSpec)
		return
	case token.EVAL:
		p.parseGenericClause(ctx, token.EVAL, p.expect(ctx, token.EVAL), p.parseEvalSpec)
		return
	case token.COLON:
		p.parseSpecialRuleClause(ctx)
		return
	case token.TEMPLATE:
		p.parseTemplateClause(ctx)
		return
	}

	if t_traverse.enabled { defer un(trace(t_traverse, "Clause(?)")) }

	var x = p.parseExpr(ctx, true); p.skipSpaces(ctx)

	if p.tok.IsAssign() {
		p.parseDefine(ctx, x)
		return
	}

	var list = []Value{ x }
	if !p.tok.IsRuleDelim() {
		list = append(list, p.parseLhsList(ctx)...)
	}
	if p.tok.IsRuleDelim() {
		p.parseRuleEntry(ctx, specialRuleNor, nil, list)
		return
	}

	var isIncludingConf = p.isIncludingConf
	// for pp := p.loader.parser; !isIncludingConf && pp != nil && pp != p; {
	// 	isIncludingConf = pp.isIncludingConf
	// 	pp = p.loader.parser
	// }
	if isIncludingConf {
		warn(ctx, "bad clause: %v (kit=%s) after %v", p.tok, p.lit, list).debug(10)
	} else {
		erro(ctx, "bad clause: %v (lit=%s) after %v", p.tok, p.lit, list).debug(10)
	}
}

type projectDeclOpts struct {
	final bool `f,final`
	noDock bool `n,nod;n,nodock;nd,no-dock`
    traveUseLoop bool `b,break;l,loop` // don't recursively use this project
    multiUseAllowed bool `m,multi`  // this project is used multiple times
}

func (p *parser) parseFile(ctx Context) *parsedFile {
	if options.traceLaunch { defer un(trace(t_launch, "parser.parseFile")) }
	if t_traverse.enabled  { defer un(trace(t_traverse, "File '"+p.file.Name()+"'")) }
    if false { defer un(tracef(t_traverse, "parseFile(%s)", p.file.Name())) }

	// Don't bother parsing the rest if we had errors scanning the first token.
	// Likely not a Go source file at all.
	if p.countErrors() > 0 { return nil }

	ctx = p.posit(ctx)

	var (
		abs, rel, tmp string
		ident *Barecomp //Bareword
		identStr string
		implicitBase string // aka. foo.bar.Baz implicitly load base 'foo/bar'
		keyword  = p.tok
		filename = p.file.Name()
		position = ctx.Position()
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
		var d *def
		if p.mode&Flat == 0 {
			d, _ = p.def(position, ".")
			d.set(ctx, DefAuto, MakePathStr(position, rel))

			d, _ = p.def(position, "/")
			d.set(ctx, DefAuto, MakePathStr(position, abs))

			d, _ = p.def(position, "CTD") // Current Temp Directory, TODO: make it $:ctd:
			d.set(ctx, DefAuto, MakePathStr(position, tmp))

			d, _ = p.def(position, "CWD") // Current Work Directory, TODO: make it $:cwd:
			d.set(ctx, DefAuto, MakePathStr(position, abs))
		} else if d = s.FindDef("/");   d == nil {
			erro(ctx, "/ not in the scope: %v", s.comment).at(position)
		} else if d = s.FindDef(".");   d == nil {
			erro(ctx, ". not in the scope: %v", s.comment).at(position)
		} else if d = s.FindDef("CTD"); d == nil {
			erro(ctx, "CTD not in the scope: %v", s.comment).at(position)
		} else if d = s.FindDef("CWD"); d == nil {
			erro(ctx, "CWD not in the scope: %v", s.comment).at(position)
		}
	} else {
		erro(ctx, "opened invalid scope for %s", filename).at(position).debug(1)
		return nil
	}

	switch keyword {
	case token.CONFIGURE:
		switch p.next(ctx, true); p.tok {
		case token.DOT:
			if err := p.ParseConfigDir(abs, abs); err != nil {
				erro(ctx, "parsing configure directory failed, '%s': %v", abs, err)
			} else {
				p.next(ctx, true) // skip the '.' token and consequence spaces
			}

			basename := filepath.Base(filepath.Dir(filename))
			ident = MakeBarecomp(position, MakeBareword(position, basename))

		default:
			erro(ctx, "unknown configuration '%v', currently only 'configure .' is supported", p.tok)
		}
	case token.PROJECT, token.PACKAGE, token.MODULE:
		if p.mode&Flat != 0 {
			erro(ctx, "forbidden `%v` in flat file", p.tok)
		}

		p.next(ctx, true)

		// Options are *Flag or *Pair of a Flag.
		var (
			opts projectDeclOpts
			optVals []Value
			pos Position
		)
		for p.tok == token.MINUS {
			var opt = p.parseExpr(ctx, false);  p.skipSpaces(ctx)
			optVals = append(optVals, opt)
			if !pos.IsValid() { pos = opt.Position() }
		}
		if !pos.IsValid() { pos = p.Position() }
		if a := parseOpts(ctx, &opts, 0, optVals...); len(a) > 0 {
			for _, v := range a {
				erro(ctx, "unknown option '%v'", v).of(v).debug(1)
			}
			return nil
		}

		var linfo = p.loads[len(p.loads)-1]

		// Smart-lang spec:
		//   * the project clause is not a declaration;
		//   * the project name does not appear in any scope.
		if p.tok == token.LPAREN || p.tok == token.EOF || p.tok == token.LINEND || p.lineComment != nil {
			var dir = filepath.Dir(filename)
			if linfo.loadee != nil && linfo.absDir == dir {
				ident = MakeBarecomp(position, MakeBareword(position, linfo.loadee.name))
			} else if name := filepath.Base(filename); name == ".base" || name == dotConfigure {
				// NOTE: loading the .base or .configure file
				ident = MakeBarecomp(position, MakeBareword(position, name))
			} else if base := filepath.Base(dir); base != "" {
				// TODO: validate basename as a valid identifier
				ident = MakeBarecomp(position, MakeBareword(position, base))
			} else {
				erro(ctx, "invalid file: %v", filename).at(position).debug(1)
			}
		} else if p.tok == token.TILDE {
			/*if filename == confinitFilename {
                ident = &ast.Bareword{ ValuePos:pos, Value:"~" }
            } else*/ if ext := filepath.Ext(filename); ext != ".smart" {
				erro(ctx, "`%v` not a smart file", filepath.Base(filename)).
					at(p.Position()).debug(1)
			} else if s := strings.TrimSuffix(filepath.Base(filename), ext); s != "" {
				ident = MakeBarecomp(position, MakeBareword(position, s))
			} else {
				erro(ctx, "`%v` not tilde name", filepath.Base(filename)).
					at(p.Position()).debug(1)
			}
			p.next(ctx, true) // skip tilde
		} else {
			// var t = p.tok
			var implicitBaseSegs []string
			ident = MakeBarecomp(p.Position())
		ForProjectName:
			for p.tok != token.EOF && p.tok != token.SPACE {
				if w := p.parseBarewordConstant(ctx, false); w == nil {
					erro(ctx, "expecting a bareword").at(ident.Position()).debug(1)
				} else if word, ok := w.(*Bareword); !ok {
					erro(ctx, "expecting a bareword: %v (%T)", w, w).at(ident.Position()).debug(1)
				} else if ident.Combine(ctx, word); p.tok == token.DOT {
					ident.Combine(ctx, MakeBareword(p.Position(), ".")) // TODO: parse to Qualiword
					implicitBaseSegs = append(implicitBaseSegs, word.string)
					p._next(ctx) // '.'
				} else { break ForProjectName }
			}
			if p.skipSpaces(ctx); len(ident.Elems) == 0 {
				// erro(ctx, "package name is empty (tok=%v %v)", t, p.tok).debug(1)
				// return nil
			} else if len(implicitBaseSegs) > 0 {
				implicitBase = filepath.Join(implicitBaseSegs...)
			}
		}

		if identStr = ident.Strval(ctx); linfo.loadee != nil && identStr != linfo.loadee.name {
			warn(ctx, "%s: declare multiple project in the same directory", p.Project()).at(ident.position).debug(24)
		} else if identStr == "_" && p.mode&DeclarationErrors != 0 {
			erro(ctx, "package name '_' is preserved").at(ident.Position()).debug(1)
			return nil
		}

		// Don't bother parsing the rest if we had errors parsing the package clause.
		// Likely not a Go source file at all.
		if n := p.countErrors(); n > 0 {
			erro(ctx, "got %d errors parsing file: %s", filename).at(p.Position()).debug(1)
			return nil
		}

		var (
			loaderProj = p.project
			_, declared = linfo.declares[identStr]
		)
		if (p.mode&Flat == 0) && p.declare(positional(ctx, ident.Position()), keyword, ident, identStr, optVals) {
			// Change the 'default' owners into the new declared project
			if s := p.Scope(); s != nil {
				if def := s.FindDef("."  ); def != nil { def.owner = p.Project() }
				if def := s.FindDef("/"  ); def != nil { def.owner = p.Project() }
				if def := s.FindDef("CTD"); def != nil { def.owner = p.Project() }
				if def := s.FindDef("CWD"); def != nil { def.owner = p.Project() }
			} else {
				erro(ctx, "file scope is nil").at(position).debug(1)
			}
			// NOTE: do.smart is always the first loaded, so the loadee will be pointed to it
			if linfo.loadee == nil { linfo.loadee = p.Project() }
			defer func(proj *Project) {
				if false && loaderProj != nil && filepath.Base(filename) == "do.smart" {
					var ctx = positional(ctx, ident.Position())
					assert(p.project == proj, "diverged project: %v != %v", p.project, proj)
					//applyUseeVars(ctx, loaderProj, p.project)  // aka. ABC += $(use.ABC)
					applyUserVars(ctx, loaderProj, p.project) // aka. use.ABC += $(use.ABC)
					if loaderProj.name == "llvm.Analysis" {
						warn(ctx, "%v, %v", loaderProj, p.project).debug(24)
					}
				}
				p.closeCurrent(ident, identStr)
			} (p.project)
		}

		var basePos Position
		if implicitBase != "" { basePos = pos } else { basePos = p.Position() }
		if p.tok == token.LPAREN {
			for p.tok != token.EOF {
				for p.next(ctx, true); !p.isEndOfList(false); {
					p.skipSpaces(ctx)
					param := p.parseExpr(ctx, false)
					p.skipSpaces(ctx)

					//if p.lineComment != nil  { break }
					//if p.tok == token.LINEND { break }
					if p.tok == token.EOF {
						erro(ctx, "unexpected end of file while parsing bases").at(basePos).debug(1)
						return nil
					}

					var (
						ctx = positional(ctx, param.Position())
						t = parseOpts(ctx, &opts, 0, param)
					)
					if keyword == token.PACKAGE || opts.final {
						// No bases for PACKAGE or final project
					} else if !p.loadBases(ctx, linfo, "", merge(t...)...) {
						erro(ctx, "loading base '%v' failed", t).of(param).debug(1)
						return nil
					}
				}
				if p.tok != token.COMMA { break }
			}
			p.expect(ctx, token.RPAREN)
			if false { defer func() { warn(ctx, "%v", ident).debug(32) } () }
		} else if !p.loadBases(ctx, linfo, implicitBase) { // for special bases, e.g. .base
			erro(ctx, "loading bases failed").at(basePos).debug(1)
			return nil
		}
		if p.skipSpaces(ctx); p.tok != token.EOF {
			p.expectLinend(ctx)
		}
		if keyword != token.PACKAGE {
			p.loadProjectConfiguration(ident, identStr, declared)
			if !opts.noDock { p.loadProjectContainer(ident, identStr) }
		}
	case token.EOF:
		return nil
	default:
		if p.mode&Flat == 0 {
			p.expected(ctx, p.pos, "configure, project, module or package keyword")
		}
	}

	if p.mode&ModuleClauseOnly == 0 {
		if p.mode&Flat == 0 {
		ForInit:
			for p.tok != token.EOF {
				switch p.tok {
				case token.IMPORT:
					p.expected(ctx, p.pos, "`use`, keyword `import` is replaced by `use`")
				case token.LINEND:
					p.next(ctx, true) // skip empty lines
				case token.USE:
					p.parseGenericClause(ctx, p.tok, p.expect(ctx, token.USE), p.parseUseSpec)
				case token.ASSERT:
					p.parseGenericClause(ctx, p.tok, p.expect(ctx, token.ASSERT), p.parseAssertSpec)
				case token.EVAL:
					p.parseGenericClause(ctx, p.tok, p.expect(ctx, token.EVAL), p.parseEvalSpec)
				default:
					if p.tok.IsKeyword() { break ForInit }
					var x = p.parseExpr(ctx, true); p.skipSpaces(ctx)
					if p.tok.IsAssign() { p.parseDefine(ctx, x) } else
					if p.tok.IsRuleDelim() {
						if p.Project() == nil {
							erro(ctx, "no project declared before defining rules")
						} else {
							x = p.parseRuleEntry(ctx, specialRuleNor, nil, []Value{x})
						}
						break ForInit
					} else {
						erro(ctx, "unexpected %v (after %v)", p.tok, x)
					}
				}
			}
		}
		if p.mode&ImportsOnly == 0 {
			// rest of module body
			for /* p.totalErrors() == 0 && */ p.tok != token.EOF {
				if p.tok == token.LINEND ||
					(p.tok == token.COMMENT && p.lineComment != nil) {
					p.next(ctx, true)
				} else {
					p.parseClause(p.posit(ctx))
					if ctx.checkErrors(true) > 0 { break }
				}
			}
		}
	}

	return &parsedFile{
		// TODO: doc: doc,
		// TODO: comments: p.comments,
		keyword:  keyword,
		position: position,
		name:     ident,
		scope:    p.Scope(),
		use:      p.imports,
	}
}
