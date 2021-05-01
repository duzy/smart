//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/ast"
        "extbit.io/smart/token"
        "extbit.io/smart/scanner"
        "path/filepath"
        "runtime/debug"
        "strconv"
        "strings"
        "unicode"
        "errors"
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
        parsingBuiltinCommand // recipe builtin

        // The composingNo* bits control the composing priority!
        
        // Bits to disable parsing ArgumentedExpr 
        composingNoArg = composingSELECT_PROP | composingDOT | composingDOTDOT | composingPATH | composingPERC
        composingNoPair = composingSELECT_PROP | composingDOT | composingPATH | composingPERC
        composingNoURL = composingSELECT_PROP | composingURL | composingDOT | composingPATH | composingGLOB | composingPERC | composingREXP /*| parsingColonName*/ | parsingSpecialRule
        composingNoPath = composingSELECT_PROP | composingURL | composingDOT | composingPATH | composingGLOB | composingPERC | composingREXP
        composingNoSelect = composingSELECT_PROP | composingDOT
        composingNoGlob = composingGLOB | composingPERC | composingREXP
        composingNoPerc = composingGLOB | composingPERC | composingREXP
        composingNoRexp = composingGLOB | composingPERC | composingREXP
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

type parser struct {
  *loader

  file    *token.File
  scanner scanner.Scanner

	// Comments
	comments  []*ast.CommentGroup
	leadComment *ast.CommentGroup // last lead comment
	lineComment *ast.CommentGroup // last line comment

	// Next token
	pos token.Pos   // token position
	tok token.Token // one token look-ahead
	lit string      // token literal

	// Error recovery
	// (used to limit the number of calls to syncXXX functions
	// w/o making scanning progress - avoids potential endless
	// loops across multiple parser functions during error recovery)
	syncPos token.Pos // last synchronization position
	syncCnt int       // number of calls to syncXXX without progress

	// Non-syntactic parser control
	exprLev int  // < 0: in control clause, >= 0: in expression
	inRhs   bool // if set, the parser is parsing a rhs expression

  bits parsingBits

	// Ordinary identifier scopes
	imports []*ast.UseSpec // list of imports

  params []*Def // parameters of current rule
  dialect string // recipe dialect of current rule
  configure bool // is parsing configure program?
}

func (p *parser) init(l *loader, filename string, src []byte) {
  p.loader = l
  p.file = l.fset.AddFile(filename, -1, len(src))

	var m scanner.Mode
	if p.tracemode&ParseComments != 0 {
		m = scanner.ScanComments
	}

	eh := func(pos token.Position, msg string) {
    p.errors.Add(pos, errors.New(msg))
  }
	p.scanner.Init(p.file, src, eh, m)

	p.next()
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

func (p *parser) trace(a ...interface{}) {
	p.traceAt(p.file.Position(p.pos), a...)
}

/*func (p *parser) error(pos token.Pos, err interface{}, a... interface{}) {
        var position = p.file.Position(pos)
        if e, ok := err.(error); ok {
                for _, t := range p.errors {
                        if e == t { return }
                }
                p.errors.Add(position, e)
        } else {
                p.errorAt(position, err, a...)
        }
}*/

// Advance to the next token.
func (p *parser) next0() {
	// Because of one-token look-ahead, print the previous token
	// when tracing as it provides a more readable output. The
	// very first token (!p.pos.IsValid()) is not initialized
	// (it is token.ILLEGAL), so don't print it .
	if p.tracing.enabled && p.pos.IsValid() {
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

        p.pos, p.tok, p.lit = p.scanner.Scan()
        if false && p.tok == token.EOF {
                fmt.Printf("parser.next0: %v, %v, %v\n", p.pos, p.tok, p.lit)
                debug.PrintStack()
        }
}

// Consume a comment and return it and the line on which it ends.
func (p *parser) consumeComment() (comment *ast.Comment, endline int) {
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

	comment = &ast.Comment{Sharp: p.pos, Text: p.lit}
        p.next0()

	return
}

// Consume a group of adjacent comments, add it to the parser's
// comments list, and return it together with the line at which
// the last comment in the group ends. A non-comment token or n
// empty lines terminate a comment group.
//
func (p *parser) consumeCommentGroup(n int) (comments *ast.CommentGroup, endline int) {
	var list []*ast.Comment
	endline = p.file.Line(p.pos)
	for p.tok == token.COMMENT && p.file.Line(p.pos) <= endline+n {
		var comment *ast.Comment
		comment, endline = p.consumeComment()
		list = append(list, comment)
	}

	// add comment group to the comments list
	comments = &ast.CommentGroup{List: list}
	p.comments = append(p.comments, comments)
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
func (p *parser) next() {
  /*if p.lineComment != nil {
    fmt.Fprintf(stderr, "next: %v", p.lineComment.Text())
    p.lineComment = nil
    p.tok = token.LINEND
    return
  }*/

	p.leadComment = nil
	p.lineComment = nil
	prev := p.pos
	if p.next0(); p.tok == token.COMMENT {
		var comment *ast.CommentGroup
		var endline int

		if p.file.Line(p.pos) == p.file.Line(prev) {
			// The comment is on same line as the previous token; it
			// cannot be a lead comment but may be a line comment.
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

// A bailout panic is raised to indicate early termination.
type bailout struct{}

func (p *parser) info(pos token.Pos, s string, a... interface{}) {
        if !strings.HasSuffix(s, "\n") {
                s += "\n"
        }
        fmt.Fprintf(stderr, "%s:info: ", p.file.Position(pos))
        fmt.Fprintf(stderr, s, a...)
}

func (p *parser) warn(pos token.Pos, s string, a... interface{}) {
        if !strings.HasSuffix(s, "\n") {
                s += "\n"
        }
        fmt.Fprintf(stderr, "%s: ", p.file.Position(pos))
        fmt.Fprintf(stderr, s, a...)
}

func (p *parser) errorExpected(pos token.Pos, msg string, a... interface{}) {
        if len(a) > 0 {
                msg = fmt.Sprintf(msg, a...)
        }
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
	p.error(pos, msg)
}

func (p *parser) expect(tok token.Token) token.Pos {
	pos := p.pos
	if p.tok != tok {
		p.errorExpected(pos, "'"+tok.String()+"'")
	}
	p.next() // make progress
	return pos
}

func (p *parser) expectLinend() {
        if p.lineComment != nil {
                // The line comment is treated as LINEND, simply ignore it.
                p.lineComment = nil
        } else if p.tok == token.LINEND {
                p.next()
        } else {
                p.errorExpected(p.pos, "'\n'")
                syncClause1(p)
	}
}

// ----------------------------------------------------------------------------
// Parsing

// syncClause advances to the next declaration.
// Used for synchronization after an error.
//
func syncClause1(p *parser) {
	for {
		switch p.tok {
		case token.USE/*token.IMPORT*/, token.INCLUDE, token.FILES, token.INSTANCE, token.EXPORT, token.EVAL:
			if p.pos == p.syncPos && p.syncCnt < 10 {
				p.syncCnt++
				return
			}
			if p.pos > p.syncPos {
				p.syncPos = p.pos
				p.syncCnt = 0
				return
			}
                // case token.ASSIGN:
                // case token.COLON:
		case token.EOF:
			return
		}
		p.next()
	}
}

// syncClause advances to the next tok.
// Used for synchronization after an error.
//
func syncClause2(p *parser, tok token.Token) {
	for {
		switch p.tok {
		case tok:
			if p.pos == p.syncPos && p.syncCnt < 10 {
				p.syncCnt++
				return
			}
			if p.pos > p.syncPos {
				p.syncPos = p.pos
				p.syncCnt = 0
				return
			}
		case token.EOF:
			return
		}
		p.next()
	}
}

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
func (p *parser) safePos(pos token.Pos) (res token.Pos) {
	defer func() {
		if recover() != nil {
			res = token.Pos(p.file.Base() + p.file.Size()) // EOF position
		}
	}()
	_ = p.file.Offset(pos) // trigger a panic if position is out-of-range
	return pos
}

// checkExpr checks that x is a valid expression (and not a Clause).
func (p *parser) checkExpr(x ast.Expr) ast.Expr {
	switch /*unparen(x)*/x.(type) {
        case *ast.ArgumentedExpr:
	case *ast.BadExpr:
	case *ast.Bareword:
	case *ast.Qualiword:
        case *ast.Constant:
	case *ast.BasicLit:
	case *ast.ClosureExpr:
	case *ast.CompoundLit:
	case *ast.DelegateExpr:
	case *ast.GlobExpr:
	case *ast.GroupExpr:
	case *ast.ListExpr: panic("unreachable")
        case *ast.Barecomp:
        case *ast.Barefile:
        case *ast.EvaluatedExpr:
        case *ast.FlagExpr:
        case *ast.NegExpr:
        case *ast.KeyValueExpr:
        case *ast.PathSegExpr:
        case *ast.PathExpr:
        case *ast.PercExpr:
        case *ast.SelectionExpr:
        case *ast.URLExpr:
        case *ast.ModifiersExpr:
        case nil:
                //p.warn(p.pos, "nil expression")
		p.error(p.pos, "nil expression")
		x = &ast.BadExpr{From:token.NoPos, To:token.NoPos}
	default:
		// all other nodes are not proper expressions
                //p.warn(x.Pos(), "bad expression (%T)\n", x)
		p.error(x.Pos(), "bad expression `%v` (%T)", x, x)
		x = &ast.BadExpr{From: x.Pos(), To: p.safePos(x.End())}
	}
	return x
}

// ----------------------------------------------------------------------------
// Barewords & Identifiers

func (p *parser) parseBarewordConstant(lhs bool) (x ast.Expr) {
	var pos, tok, value = p.pos, p.tok, ""
        switch tok {
	case token.BAREWORD:
                value = p.lit
        case token.AT, token.DOT, token.DOTDOT:
                value = tok.String() // Special bareword.
        default:
                if tok.IsKeyword() {
                        value = tok.String()
                } else {
                        p.expect(token.BAREWORD)
                }
	}

        p.next() // consumes the word

        /*if p.tok == token.DOT && int(p.pos) == int(pos)+len(value) {
                quali := &ast.Qualiword{ pos, []string{value} }
        ForDot:
                for p.tok == token.DOT {
                        pos = p.pos
                        p.next() // skip DOT
                        if p.pos != token.Pos(int(pos)+1) {
                                p.error(p.pos, "expecting next qualidfied word")
                                break
                        }
                        switch pos, tok = p.pos, p.tok; tok {
                        case token.BAREWORD:
                                value = p.lit
                        case token.AT, token.DOT, token.DOTDOT:
                                value = tok.String() // Special bareword.
                        default:
                                if tok.IsKeyword() {
                                        value = tok.String()
                                } else {
                                        p.error(p.pos, "qualify unrecognized word (%v)", tok)
                                        break ForDot
                                }
                        }
                        quali.Words = append(quali.Words, value)
                        p.next() // consume the 'word'
                        if int(p.pos) != int(pos)+len(value) { break }
                }
                x = quali
        } else*/ if tok.IsConstant() {
                x = &ast.Constant{ TokPos:pos, Tok:tok }
        } else {
                x = &ast.Bareword{ ValuePos:pos, Value:value }
        }
        return
}

func (p *parser) parseSelect(lhs ast.Expr) (res ast.Expr) {
	if p.tracing.enabled {
		defer un(trace(p, "Select"))
	}

        tok := p.tok // the arrow '->' or '=>'
        p.next() // skip '->' or '=>'

        defer p.setbits(p.setbit(composingSELECT_PROP))

        rhs := p.checkExpr(p.parseExpr(false))
        res = &ast.SelectionExpr{ lhs, tok, rhs }
        if (p.tok == token.SELECT_PROP || p.tok == token.SELECT_PROG1 || p.tok == token.SELECT_PROG2) && rhs.End() == p.pos {
                // Continue the selection recursivly.
                res = p.parseSelect(res)
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
        return p.lineComment != nil || p.tok.IsListDelim() || (lhs && p.tok.IsAssign())
}

func (p *parser) isEndOfURL(lhs bool) bool {
        return p.isEndOfLine() || p.isEndOfList(lhs)
}

func (p *parser) isEndOfDotConcat(lhs bool) bool {
        // Expressions like `FOO.BAR(xxx)` does not count.
        switch p.tok {
        case token.LPAREN, token.COLON, token.PCON, token.ASSIGN: fallthrough
        case token.SELECT_PROP, token.SELECT_PROG1, token.SELECT_PROG2:
                return true
        }
        return p.isEndOfLine() || p.isEndOfList(lhs)
}

func (p *parser) parseDependList() (list []ast.Expr) {
	if p.tracing.enabled { defer un(trace(p, "Depends")) }
        for p.tok != token.SEMICOLON && p.tok != token.BAR && !p.isEndOfLine() {
                if p.tok == token.COLON { // FIXME: this check is not working!
                        // FIXME: detects unexpected colon ':'
                        p.error(p.pos, "unexpected colon")
                        p.next() // just ignore this colon
                } else {
                        x := p.checkExpr(p.parseExpr(false))
                        list = append(list, x)
                }
        }
	return
}

// If lhs is set, result list elements which are identifiers are not resolved.
func (p *parser) parseExprList(lhs bool) (list []ast.Expr) {
	if p.tracing.enabled { defer un(trace(p, "List")) }
        for !p.isEndOfList(lhs) {
                x := p.checkExpr(p.parseExpr(lhs))
                list = append(list, x)
                // If there's a comment right after the parsed expression, we break
                // the expression list to treat the end-of-line comment like a LINEND.
                if p.lineComment != nil { break }
                if p.tok == token.LINEND { break }
                if p.tok == token.EOF { break }
        }
	return
}

func (p *parser) parseListExpr(lhs bool) *ast.ListExpr {
        return &ast.ListExpr{ p.parseExprList(lhs) }
}

func (p *parser) setRHS(v bool) (old bool) {
        old = p.inRhs; p.inRhs = v; return
}

func (p *parser) parseLhsList() []ast.Expr {
        // Line comment of previous Clause will break the parsing loop,
        // so we clear the previous line comment
        p.lineComment = nil
 
        defer p.setRHS(p.setRHS(false))
	list := p.parseExprList(true)
	return list
}

func (p *parser) parseRhsList() []ast.Expr {
        defer p.setRHS(p.setRHS(true))
	list := p.parseExprList(false)
	return list
}

// ----------------------------------------------------------------------------
// Expressions

func (p *parser) parseGroupExpr(lhs bool) ast.Expr {
	if p.tracing.enabled { defer un(trace(p, "Group")) }
        
        lpos := p.pos
        p.next()
        elems, converted := p.parseExprList(false), false
        for p.tok == token.COMMA {
                p.next()
                next := p.parseListExpr(false)
                if !converted {
                        elems = []ast.Expr{ &ast.ListExpr{ elems }, next }
                        converted = true
                } else {
                        elems = append(elems, next)
                }
        }
        rpos := p.expect(token.RPAREN)
        // FIXME: return BadExpr if RPAREN is unexpected
        return &ast.GroupExpr{
                Lparen: lpos,
                Elems: elems,
                Rparen: rpos,
        }
}

func (p *parser) parseArgumentedExpr(x ast.Expr) ast.Expr {
	if p.tracing.enabled { defer un(trace(p, "Argumented")) }

        p.next() // skip token.LPAREN
        
        var a = []ast.Expr{ p.parseListExpr(false) }
        for p.tok == token.COMMA {
                p.next() // skip token.COMMA
                a = append(a, p.parseListExpr(false))
        }

        return &ast.ArgumentedExpr{
                X:x, Arguments:a,
                EndPos: p.expect(token.RPAREN),
        }
}

func (p *parser) parseGlobMeta() (x ast.Expr) {
        tok, pos := p.tok, p.pos
        p.next()
        x = &ast.GlobMeta{ pos, tok }
        return
}

func (p *parser) parseGlobRange() (x ast.Expr) {
	if p.tracing.enabled { defer un(trace(p, "Glob")) }

        p.expect(token.LBRACK) // skip '['

        chars := p.parseExpr(false)

        p.expect(token.RBRACK) // skip ']'
        if chars.End()+1 != p.pos {
                p.error(p.pos, "unexpected extra spaces")
        }

        x = &ast.GlobRange{ chars }
        return
}

func (p *parser) parseGlobExpr(x ast.Expr) ast.Expr {
	if p.tracing.enabled { defer un(trace(p, "Glob")) }

        var components []ast.Expr
        if x != nil {
                components = append(components, x)
        }

        // avoid nesting glob expressions
        defer p.setbits(p.setbit(composingGLOB))

        ForTok: for {
                switch p.tok {
                case token.RPAREN, token.COMMA, token.LINEND, token.EOF:
                        break ForTok
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
                if p.lineComment != nil || x.End() != p.pos {
                        break ForTok
                }
        }
        
        return &ast.GlobExpr{ components }
}

func (p *parser) parsePercExpr(lhs bool, x ast.Expr) ast.Expr {
	if p.tracing.enabled { defer un(trace(p, "Perc")) }

        // avoid nesting percent expressions
        defer p.setbits(p.setbit(composingPERC))

        var (
                y ast.Expr
                pos = p.pos
        )
        if p.next(); pos+1 == p.pos { // joint, e.g. '%.o', but skip '% .o'
                switch p.tok {
                case token.COLON, token.COLON2,
                     token.LPAREN, token.RPAREN,
                     token.LBRACK, token.RBRACK,
                     token.LBRACE, token.RCOLON,
                     token.PCON, token.SEMICOLON,
                     token.COMMA, token.LINEND:
                case token.PERC: // %%
                        p.next() // consume the second %
                        perc2 := &ast.PercExpr{ OpPos:p.pos }
                        if pos+2 == p.pos {
                                switch p.tok {
                                case token.PERC: // %%%
                                        p.error(p.pos, "too many %")
                                case token.PCON: // FIXES: %%/xxx -> Path(%% xxx)
                                        x = &ast.PercExpr{ X:x, OpPos:p.pos, Y:perc2 }
                                        return p.parsePathExpr(lhs, x)
                                case token.COLON,     token.COLON2,
                                     token.LPAREN,    token.RPAREN,
                                     token.LBRACK,    token.RBRACK,
                                     token.LBRACE,    token.RCOLON,
                                     token.SEMICOLON, token.COMMA,
                                     token.LINEND:
                                default:
                                        perc2.Y = p.checkExpr(p.parseExpr(false))
                                }
                        }
                        y = perc2
                default:
                        y = p.checkExpr(p.parseExpr(false))
                }
        }

        return &ast.PercExpr{ X:x, OpPos:pos, Y:y }
}

func (p *parser) parseRegexpExpr() (x ast.Expr) {
	if p.tracing.enabled { defer un(trace(p, "Regexp")) }

        // avoid nesting percent expressions
        defer p.setbits(p.setbit(composingREXP))

        p.error(p.pos, "todo: regexp")

        return
}

func (p *parser) parseKeyValueExpr(x ast.Expr) ast.Expr {
	if p.tracing.enabled { defer un(trace(p, "Pair")) }

        pos, tok := p.pos, p.tok; p.next()
        return &ast.KeyValueExpr{
                Key:   x,
                Tok:   tok,
                Equal: pos,
                Value: p.parseExpr(false),
        }
}

func (p *parser) parseFlagExpr(lhs bool) ast.Expr {
	if p.tracing.enabled { defer un(trace(p, "Flag")) }

        var (
                pos = p.pos
                x ast.Expr
        )

        p.next() // skip dash '-'

        // Flag expressions, excluding "-)" "-]" "-}" "-\n", "-=", "-:", etc.
        if p.isEndOfLine() || p.isEndOfList(false) {
                x = nil
        } else if false {
                x = p.checkExpr(p.parseExpr(false))
        } else {
                x = p.checkExpr(p.parseUnaryExpr(false))
        }
        return &ast.FlagExpr{ DashPos: pos, Name: x }
}

func (p *parser) parseNegExpr(lhs bool) ast.Expr {
	if p.tracing.enabled { defer un(trace(p, "Negative")) }
        pos := p.expect(token.EXC)
        val := p.parseExpr(lhs)
        return &ast.NegExpr{ NegPos: pos, Val: val }
}

func (p *parser) parseBasicLit(lhs bool) ast.Expr {
        pos, tok, lit := p.pos, p.tok, p.lit
        end := int(pos) + len(lit)
        switch tok {
        case token.STRING: end += 2
        }
        p.next()
        // ESCAPE is handled in value.EscapeChar
        return &ast.BasicLit{
                ValuePos: pos,
                Kind: tok,
                Value: lit,
                EndPos: token.Pos(end),
        }
}

func (p *parser) parseCompoundLit(lhs bool) ast.Expr {
        var (
                lpos = p.pos
                elems []ast.Expr
                rpos token.Pos
        )
        p.next()
        ForCompound: for p.tok != token.EOF {
                switch p.tok {
                case token.COMPOSED: break ForCompound
                case token.LINEND:
                        p.error(p.pos, "unexpected end of line for compound string")
                        break ForCompound
                }
                elems = append(elems, p.checkExpr(p.parseExpr(false)))
        }
        rpos = p.expect(token.COMPOSED)
        return &ast.CompoundLit{
                Lquote: lpos,
                Elems:  elems,
                Rquote: rpos,
        }
}

// Parses dot composing expressions (TODO: check against file extensions).
//   .foo
//   .'foo'
//   ."foo"
//   .(foo)
//   ..foo
//   ..'foo'
//   .foo.bar
func (p *parser) parseDotExpr(lhs bool, x ast.Expr) (res ast.Expr) {
	if p.tracing.enabled { defer un(trace(p, "Dot")) }
        
        defer p.setbits(p.setbit(composingDOT))

        var comp *ast.Barecomp
        if x == nil { panic(fmt.Sprintf("nil dot (tok=%v)", p.tok)) }
        if comp, _ = x.(*ast.Barecomp); comp == nil {
                comp = new(ast.Barecomp)
                comp.Elems = append(comp.Elems, x)
        }

        for comp.End() == p.pos && !p.isEndOfDotConcat(lhs) {
                x = p.checkExpr(p.parseComposedExpr(false))
                if _, ok := x.(*ast.BadExpr); ok {
                        p.error(x.Pos(), "dot: bad expression")
                        break
                }
                
                comp.Combine(x)

                if p.tok == token.DOT && comp.End() == p.pos {
                        var dot = &ast.Bareword{p.pos, p.tok.String()}
                        comp.Elems = append(comp.Elems, dot)
                        p.next() // '.'
                }
        }

        // FIXME: *.o => obj
        //   BUG: Barecomp{Glob . KeyValueExpr}
        //   FIX: KeyValueExpr{Barecomp, Bareword}

        return comp
}

func (p *parser) parsePathExpr(lhs bool, start ast.Expr) ast.Expr {
	if p.tracing.enabled { defer un(trace(p, "Path")) }

        defer p.setbits(p.setbit(composingPATH))

        var path *ast.PathExpr
        if start == nil {
                var pos = p.pos
                p.next()
                p.error(pos, "bad closure/delegate name")
                return &ast.BadExpr{ From:pos, To:p.pos }
        } else if path, _ = start.(*ast.PathExpr); path == nil {
                path = &ast.PathExpr{ Segments:[]ast.Expr{ start } }
        }

        BuildPath: for p.tok == token.PCON {
                var pos = p.pos
                for p.next(); p.tok == token.PCON && pos+1 == p.pos; {
                        pos = p.pos; p.next() // skips repeated '/' sequence
                }

                switch p.tok {
                case token.RPAREN, token.RBRACE, token.RCOLON, token.LINEND:
                        // Encountered the tailing '/', append 'zero' segment.
                        seg := &ast.PathSegExpr{ pos, 0 }
                        path.Segments = append(path.Segments, seg)
                        break BuildPath
                default:if pos+1 < p.pos {
                        break BuildPath 
                }}
                
                x := p.checkExpr(p.parseComposedExpr(false))
                path.Segments = append(path.Segments, x)
                if x.End() != p.pos || p.isEndOfLine() {
                        break BuildPath
                }
        }
        return path
}

func (p *parser) parseURLExpr(lhs bool, scheme ast.Expr) (res ast.Expr) {
	if p.tracing.enabled { defer un(trace(p, "URL")) }

        defer p.setbits(p.setbit(composingURL))

        var url = &ast.URLExpr{Scheme:scheme}
        var end token.Pos

        url.Colon1 = p.expect(token.COLON) // consumes ':'
        end = url.Colon1+1

        if end == p.pos && p.tok == token.PCON {
                pos := p.pos
                end = p.pos+1
                p.next() // the first '/'
                if end == p.pos && p.tok == token.PCON {
                        p.expect(token.PCON) // the second '/'
                        url.SlashSlash = pos // '//'
                        end = url.SlashSlash+2
                } else {
                        panic(fmt.Sprintf("todo: url path (%s %s),", p.tok, p.lit))
                        return
                }
        } else if end == p.pos && !p.isEndOfURL(lhs) {
                panic(fmt.Sprintf("todo: url path (%s %s).", p.tok, p.lit))
                return
        }

        if end == p.pos && !p.isEndOfURL(lhs) {
                userOrHost := p.checkExpr(p.parseComposedExpr(false))
                end = userOrHost.End()
                if end == p.pos && p.tok == token.COLON {
                        url.Username, url.Colon2 = userOrHost, p.pos
                        end = url.Colon2 + 1
                        p.next() // ':'
                        if end == p.pos && p.tok != token.AT && p.tok != token.PCON && !p.isEndOfURL(lhs) {
                                url.Password = p.checkExpr(p.parseComposedExpr(false))
                                end = url.Password.End()
                        }
                } else {
                        url.Host = userOrHost
                }
        }
        if end == p.pos && p.tok == token.AT {
                url.At = p.pos
                p.next() // '@'
                end = url.At + 1
        }
        if end == p.pos && url.Host == nil && url.Colon2 == token.NoPos && url.At == token.NoPos && !p.isEndOfURL(lhs) {
                url.Host = p.checkExpr(p.parseComposedExpr(false))
                end = url.Host.End()
                if end == p.pos && p.tok == token.COLON {
                        url.Colon3 = p.pos
                        p.next() // ':'
                        end = url.Colon3 + 1
                        if end == p.pos {
                                url.Port = p.checkExpr(p.parseComposedExpr(false))
                                end = url.Port.End()
                        }
                }
        }
        if end == p.pos && p.tok == token.PCON {
                url.Path = p.parsePathExpr(lhs, &ast.PathSegExpr{ p.pos, p.tok })
                end = url.Path.End()
        }
        p.scanner.DontScanComment = true
        if end == p.pos && p.tok == token.QUE {
                url.Que = p.pos
                p.next() // '?'
                end = url.Que + 1
                if end == p.pos && p.tok != token.COMMENT && !p.isEndOfURL(lhs) {
                        url.Query = p.checkExpr(p.parseComposedExpr(false))
                        end = url.Query.End()
                }
        }
        if end == p.pos && p.tok == token.COMMENT {
                url.NumSign = p.pos
                p.next() // '#'
                end = url.NumSign + 1
                if end == p.pos && !p.isEndOfURL(lhs) {
                        url.Fragment = p.checkExpr(p.parseComposedExpr(false))
                        end = url.Fragment.End()
                }
        }
        p.scanner.DontScanComment = false
        return url
}

func (p *parser) parseClosureDelegateName(tok token.Token) (name ast.Expr) {
	if p.tracing.enabled {	defer un(trace(p, "ClosureDelegateName")) }

        /*
        if tok == token.LCOLON {
                // Set parsingColonName to avoid ':' being treated as URL.
                defer p.setbits(p.setbit(parsingColonName))
        }
        */

        var pos = p.pos
        if name = p.checkExpr(p.parseExpr(false)); name == nil {
                p.error(pos, "bad closure/delegate name")
                name = &ast.BadExpr{ From:pos, To:p.pos }
        }
        return
}

func (p *parser) parseClosureDelegate() ast.Expr {
	if p.tracing.enabled {	defer un(trace(p, "ClosureDelegate")) }
        
        // FIXME: push p.bits before entering a $(...) or &(...)
        defer func(a parsingBits) { p.bits = a }(p.bits)
        p.bits = 0 // start with zero bits

        var (
                lpos = token.NoPos
                rpos = token.NoPos
                pos  = p.pos
                tok  = p.tok
                name   ast.Expr
                rest   []ast.Expr
                tokLp  token.Token
                resolved Value // Object or *selection
        )
        switch p.next(); p.tok {
        case token.LPAREN, token.LBRACE, token.LCOLON: // $(...), ${...}, $:...:
                lpos, tokLp = p.pos, p.tok

                p.next() // skips LPAREN, LBRACE, LCOLON
                if lpos+1 != p.pos {
                        p.error(lpos+1, "unexpected spaces")
                        return &ast.BadExpr{ From:lpos+1, To:p.pos }
                }

                name = p.parseClosureDelegateName(tokLp)
                if bad, ok := name.(*ast.BadExpr); ok { return bad }
                if v := p.expr(name); v == nil {
                        p.error(name.Pos(), "name is nil (`%T`)", name)
                } else {
                        if !v.closured() {
                                var err error
                                switch tokLp {
                                case token.LPAREN: resolved, err = p.resolve(v)
                                case token.LBRACE: resolved, err = p.find(v)
                                case token.LCOLON:
                                        // TODO: check special var names
                                }
                                // add err to the parse error list
                                if err != nil { p.error(name.Pos(), "%v", err) }
                        }
                        name = &ast.EvaluatedExpr{ name, v }
                }

                if (tokLp == token.LPAREN && p.tok != token.RPAREN) ||
                   (tokLp == token.LBRACE && p.tok != token.RBRACE) ||
                   (tokLp == token.LCOLON && p.tok != token.RCOLON) {
                        rest = append(rest, p.parseListExpr(false))
                        for p.tok == token.COMMA {
                                p.next()
                                rest = append(rest, p.parseListExpr(false))
                        }
                }

                switch tokLp {
                case token.LPAREN: rpos = p.expect(token.RPAREN)
                case token.LBRACE: rpos = p.expect(token.RBRACE)
                case token.LCOLON: rpos = p.expect(token.RCOLON)
                        if name.End() != rpos {
                                p.error(name.Pos(), "special dont need extra spaces")
                        }
                }
        default:
                if tok != token.CLOSURE { // $(...), disabled $name.
                        // &(...), &{...}, &'...', &"..."
                        p.error(p.pos, "expects `%v` or `%v` or quotes", token.LPAREN, token.LBRACE)
                        return &ast.BadExpr{ From:p.pos, To:p.pos }
                } else if p.tok == token.STRING || p.tok == token.COMPOUND {
                        lpos, tokLp = p.pos, p.tok

                        // &'xxxx' or &"xxxx"
                        if name = p.checkExpr(p.parseExpr(false)); name == nil {
                                p.error(pos, "bad name expr, expecting quotes")
                                return &ast.BadExpr{ From:p.pos, To:p.pos }
                        } else if v := p.expr(name); v == nil {
                                p.error(name.Pos(), "name is nil (`%T`)", name)
                        } else {
                                if !v.closured() {
                                        var err error
                                        switch tokLp {
                                        case token.LPAREN: resolved, err = p.resolve(v)
                                        case token.LBRACE: resolved, err = p.find(v)
                                        case token.LCOLON:
                                                // TODO: check special names
                                        }
                                        if err != nil {
                                                p.error(name.Pos(), "name is nil: %v", err)
                                        }
                                }
                                name = &ast.EvaluatedExpr{ name, v }
                        }
                } else {
                        // &(...), &{...}, &'...', &"..."
                        p.error(p.pos, "expects `%v`, `%v` or quotes", token.LPAREN, token.LBRACE)
                        return &ast.BadExpr{ From:p.pos, To:p.pos }
                }
        }

        cd := ast.ClosureDelegate{
                Position: p.file.Position(pos),
                TokPos: pos,
                Lparen: lpos,
                Name: name,
                Resolved: resolved,
                Args: rest,
                Rparen: rpos,
                TokLp: tokLp,
                Tok: tok,
        }
        if tok == token.DELEGATE {
                return &ast.DelegateExpr{ cd }
        } else {
                return &ast.ClosureExpr{ cd }
        }
}

func (p *parser) parseSpecialClosureDelegate(lhs bool) ast.Expr {
	if p.tracing.enabled {
		defer un(trace(p, "SpecialClosureDelegate"))
	}

        pos, tok, s := p.pos, p.tok, p.tok.String()[1:]
        p.next()
        
        name := &ast.Bareword{p.pos, s}
        resolved, err := p.resolve(&Bareword{trivial{Position(p.file.Position(pos))},s})
        if err != nil {
                p.error(pos, "%v", err)
        } else if resolved == nil {
                p.error(pos, "`%v` is nil", s)
        } else if _, ok := resolved.(Caller); !ok {
                p.error(pos, "`%v` is not callable (%T)", s, resolved)
        }

        cd := ast.ClosureDelegate{
                TokPos: pos,
                Lparen: token.NoPos,
                Name: name,
                Resolved: resolved,
                Rparen: token.NoPos,
                Tok: tok,
        }
        if tok.IsDelegate() {
                return &ast.DelegateExpr{ cd }
        } else {
                return &ast.ClosureExpr{ cd }
        }
}

func (p *parser) parseUnaryExpr(lhs bool) (x ast.Expr) {
	if p.tracing.enabled { defer un(trace(p, "Unary")) }

        switch p.tok {
        case token.BAREWORD, token.AT:
                return p.parseBarewordConstant(lhs)

        case token.BIN, token.OCT, token.INT, token.HEX, token.FLOAT,
             token.DATETIME, token.DATE, token.TIME, token.URI,
             token.RAW, token.STRING, token.ESCAPE:
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
                if p.next(); end != p.pos { // FIXME: ~user
                        // '~', '.' or '..' used as bareword
                        return &ast.Bareword{ pos, str }
                } else if p.tok == token.PCON { // check /
                        return p.parsePathExpr(lhs, &ast.PathSegExpr{ pos, tok })
                } else if tok == token.DOT {
                        var x = &ast.Bareword{ pos, str }
                        if p.bits&composingDOT == 0 {
                                return p.parseDotExpr(lhs, x)
                        }
                        return x
                } else if tok == token.TILDE { // TODO: ~user
                        return &ast.PathSegExpr{ pos, tok }
                } else {
                        p.error(pos, "unexpected path segment")
                        return &ast.BadExpr{ From:pos, To:p.pos }
                }
                
        case token.PCON:
                return p.parsePathExpr(lhs, &ast.PathSegExpr{ p.pos, p.tok })

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

        if optionPrintStack { debug.PrintStack() }
        
        pos := p.pos
        p.warn(pos, "'%v' bad unary expression (lit=%v,lhs=%v)\n", p.tok, p.lit, lhs)
        p.errorExpected(pos, "unary expression")
        p.next() // go to next token
        return &ast.BadExpr{ From:pos, To:p.pos }
}

func (p *parser) parseComposedExpr(lhs bool) (x ast.Expr) {
	if p.tracing.enabled { defer un(trace(p, "Composed")) }
        switch x = p.parseUnaryExpr(lhs); p.tok { // check composible expressions
        case token.SELECT_PROP, token.SELECT_PROG1, token.SELECT_PROG2: // foo->bar  foo=>bar  foo~>bar
                if p.bits&composingNoSelect == 0 {
                        if x.End() == p.pos { // accepts 'foo=>bar', but 'foo => bar' is different
                                x = p.parseSelect(x); break
                        }
                }
                if (p.tok == token.SELECT_PROG1 || p.tok == token.SELECT_PROG2) /*&& p.bits&composingNoPair == 0*/ {
                        /*if x.End() < p.pos {
                                x = p.parseKeyValueExpr(x); break
                        }*/
                }

        case token.LBRACK: // xxx[(foo ...)]
                if p.bits&composingModifier == 0 && x.End() == p.pos {
                        // FIXME: compose lhs x
                        m := p.parseModifiersExpr()
                        p.error(m.Pos(), "composing modifiers is ignored (unimplemented yet)")
                }
        case token.STAR, token.QUE/*, token.LBRACK*/: // foo*bar foo?bar foo[a-z]bar
                if p.bits&composingNoGlob == 0 && x.End() == p.pos {
                        x = p.parseGlobExpr(x)
                }
        case token.PERC: // foo%bar
                // FIXME: %/foo/bar -> Path(% foo bar)
                if p.bits&composingNoPerc == 0 && x.End() == p.pos {
                        x = p.parsePercExpr(lhs, x)
                }
        case token.DOT: // foo.bar.baz.o
                // FIXME: push bits when parsing $(...)
                if p.bits&composingDOT == 0 && x.End() == p.pos {
                        x = p.parseDotExpr(lhs, x)
                }
        case token.PCON: // ie. subdir/in/somewhere
                if p.bits&composingNoPath == 0 && x.End() == p.pos {
                        // Path expressions, except '-I/path/to/include'
                        switch x.(type) {
                        case *ast.FlagExpr: // By pass these expressions.
                        default: x = p.parsePathExpr(lhs, x)
                        }
                }
        case token.COLON:
                if (p.bits&parsingBuiltinCommand != 0 || !lhs) && p.bits&composingNoURL == 0 && x.End() == p.pos {
                        x = p.parseURLExpr(lhs, x)
                }
        }
        /*if x.String() == "..." {
                fmt.Fprintf(stderr, "composed: %v %v %v\n", x, p.tok, p.lit)
                debug.PrintStack()
        }*/
        return
}

func (p *parser) parseExpr(lhs bool) (x ast.Expr) {
	if false && p.tracing.enabled { defer un(trace(p, "Expression")) }

        pos, tok := p.pos, p.tok
        if x = p.parseComposedExpr(lhs); x == nil {
                p.warn(pos, "`%v` invalid expression", tok)
                syncClause2(p, token.LINEND)
        } else if !lhs {
                switch p.tok {
                case token.ASSIGN: // Example: '*.o = obj'
                        if p.bits&composingNoPair == 0 {
                                x = p.parseKeyValueExpr(x)
                        }
                case token.SELECT_PROG1, token.SELECT_PROG2:
                        if p.bits&composingNoPair == 0 {
                                x = p.parseKeyValueExpr(x)
                        }

                case token.LPAREN:
                        if p.bits&composingNoArg == 0 && x.End() == p.pos {
                                if _, ok := x.(*ast.ArgumentedExpr); ok {
                                        p.error(x.Pos(), "multiple argument assignment")
                                }
                                x = p.parseArgumentedExpr(x)
                        }

                case token.COMPOSED, token.COMMA, token.COLON, token.SEMICOLON:
                case token.RPAREN, token.RBRACK, token.RBRACE, token.RCOLON:
                case token.SELECT_PROP, /*token.SELECT_PROG,*/ token.LINEND:
                        // Compose nothing at this point!

                default:if p.tok != token.EOF && x.End() == p.pos {
                        if p.tok == token.BAR {
                                // in case of: [(var)|...]
                                if _, ok := x.(*ast.GroupExpr); ok { break }
                        }

                        // further composing
                        var y = p.parseComposedExpr(lhs)
                        if comp, ok := x.(*ast.Barecomp); !ok || comp == nil {
                                comp = &ast.Barecomp{Elems:[]ast.Expr{x}}
                                comp.Combine(y)
                                x = comp
                        } else {
                                comp.Combine(y)
                        }
                        // fmt.Fprintf(stderr, "composed: %v (%v)\n", x, y)
                }}
        }
        return x
}

// ----------------------------------------------------------------------------
// Clauses & Declarations

type parseSpecFunc func(doc *ast.CommentGroup, generic *genericoptions, iota int) ast.Spec

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

func (p *parser) parseUseSpec(doc *ast.CommentGroup, generic *genericoptions, _ int) (res ast.Spec) {
	var spec = &ast.UseSpec{ p.parseDirectiveSpec() }
        p.imports = append(p.imports, spec)
        res = spec
        if generic.dontOperate {
                return
        }

        var ( opts importoptions ; err error )
        if _, err = parseFlags(generic.options, []string{
                "r,reusing",
        }, func(ru rune, v Value) {
                switch ru {
                case 'r': if opts.allowReuse, err = trueVal(v, false); err != nil { return }
                }
        }); err == nil {
                p.loadUseSpec(opts, spec)
        }
        return
}

func (p *parser) parseIncludeSpec(doc *ast.CommentGroup, generic *genericoptions, _ int) ast.Spec {
	if p.tracing.enabled {
		defer un(trace(p, "Spec"))
	}
        
        var (
                x = p.parseExpr(false)
                comment = p.lineComment
                props []ast.Expr
        )

        if p.tok == token.COLON {
                x = &ast.IncludeRuleClause{
                        p.parseRuleClause(p.tok, specialRuleNor, nil, []ast.Expr{x}),
                }
        }
        props = append(props, x)

        spec := &ast.IncludeSpec{ast.DirectiveSpec{
                Doc: doc,
                Props: props,
                Comment: comment,
                EndPos: p.pos,
        }}
        if !generic.dontOperate {
                p.include(spec)
        }
        return spec
}

func (p *parser) parseInstanceSpec(doc *ast.CommentGroup, generic *genericoptions, _ int) ast.Spec {
        return &ast.InstanceSpec{ p.parseDirectiveSpec() }
}

func (p *parser) parseConfigurationSpec(doc *ast.CommentGroup, generic *genericoptions, _ int) ast.Spec {
        name := p.parseExpr(false)
        define := p.parseDefineClause(p.tok, name)
        spec := &ast.ConfigurationSpec{ *define }
        if !generic.dontOperate {
                p.configuration(spec)
        }
        return spec
}

func (p *parser) parseFilesSpec(doc *ast.CommentGroup, generic *genericoptions, _ int) ast.Spec {
        defer p.setbits(p.setbit(parsingFilesSpec))
        spec := &ast.FilesSpec{ p.parseDirectiveSpec() }
        if generic.dontOperate { return spec }
        for _, prop := range spec.Props {
                switch v := p.expr(prop).(type) {
                case *Pair:
                        var pats, paths []Value
                        switch k := v.Key.(type) {
                        case *Group: pats = k.Elems
                        default: pats = append(pats, v.Key)
                        }
			for _, pat := range pats {
				if pat.closured() {
					if false { p.info(prop.Pos(), "%v", pat) }
					pats = append(pats, pat)
				} else {
					if v, err := mergeresult(ExpandAll(pat)); err != nil {
						p.error(prop.Pos(), "%v", err)
					} else {
						pats = append(pats, v...)
					}
				}
			}
			switch vv := v.Value.(type) {
			case *Group: paths = vv.Elems
			default: paths = append(paths, vv)
                        }
                        for _, k := range pats { p.project.mapfile(k, paths) }
                case Value:
                        var pats, paths []Value
                        paths = []Value{&String{trivial{v.Position()},p.project.absPath}}
                        switch g := v.(type) {
                        default: pats = append(pats, v)
                        case *Group: pats = g.Elems
                        }
                        for _, k := range pats { p.project.mapfile(k, paths) }
                default:
                        p.error(prop.Pos(), "bad file spec (%T)", prop)
                }
		/*if v := p.expr(prop); !(isNil(v) || isNone(v)) {
			p.project.files = append(p.project.files, v)
		} else {
                        p.error(prop.Pos(), "bad file spec (%T)", prop)
		}*/
        }
        return spec
}

func (p *parser) parseEvalSpec(doc *ast.CommentGroup, generic *genericoptions, _ int) ast.Spec {
        spec := &ast.EvalSpec{ p.parseDirectiveSpec(), nil }
        if prop0 := p.expr(spec.Props[0]); prop0 == nil {
                p.error(spec.Props[0].Pos(), "`%v` illegal", spec.Props[0])
        } else if name, err := prop0.Strval(); err != nil {
                p.error(spec.Props[0].Pos(), "%v", err)
        } else if spec.Resolved, err = p.resolve(&Bareword{trivial{prop0.Position()},name}); err != nil {
                p.error(spec.Pos(), "%v", err)
        } else if spec.Resolved == nil {
                p.error(spec.Props[0].Pos(), "no such command `%s' (%v).", name, prop0)
        } else if b, ok := spec.Resolved.(*Builtin); ok && (b.flag&builtinCommand) == 0 {
                p.error(spec.Props[0].Pos(), "'%s' (%v) builtin is not a command", name, prop0)
        } else if generic.dontOperate {
                // NOOP
        } else {
                p.evalspec(spec)
        }
        return spec
}

func (p *parser) parseDirectiveSpec() (gs ast.DirectiveSpec) {
	if p.tracing.enabled {
		defer un(trace(p, "Spec"))
	}
        
        var (
                doc = p.leadComment
                comment *ast.CommentGroup
                props []ast.Expr
        )

        props = append(props, p.parseExpr(false))

        // Parse the parameters.
        ParamsParseLoop: for p.tok != token.EOF {
                switch p.tok {
                case token.COMMA, token.LINEND, token.RPAREN, token.RBRACE, token.RCOLON:
                        break ParamsParseLoop
                }
                if p.lineComment != nil {
                        // found a line comment at the end
                        comment = p.lineComment
                        break
                }
                props = append(props, p.parseExpr(false))
        }
        return ast.DirectiveSpec{
                Doc: doc,
                Props: props,
                Comment: comment,
                EndPos: p.pos,
        }
}

func (p *parser) parseGenericClause(keyword token.Token, pos token.Pos, f parseSpecFunc) *ast.GenericClause {
	if p.tracing.enabled {
		defer un(trace(p, "Clause("+keyword.String()+")"))
	}

        var (
                doc = p.leadComment
                lparen, rparen token.Pos
                generic = genericoptions{
                        keyword: keyword,
                }
                specs []ast.Spec
        )

        for p.tok == token.MINUS {
                var conds []Value
                x := p.checkExpr(p.parseExpr(false))
                opt := p.expr(x)
                switch t := opt.(type) {
                case *Argumented:
                        if flag, ok := t.value.(*Flag); !ok {
                                // does nothing
                        } else if s, err := flag.name.Strval(); err != nil {
                                p.error(x.Pos(), "bad argumented option `%v` (%v)", x, t.value)
                        } else if s == "cond" {
                                conds = t.args
                        }
                case *Pair:
                        if flag, ok := t.Key.(*Flag); !ok {
                                // does nothing
                        } else if s, err := flag.name.Strval(); err != nil {
                                p.error(x.Pos(), "bad option key `%v` (%v)", x, t.Key)
                        } else if s == "cond" {
                                if g, ok := t.Value.(*Group); ok {
                                        conds = g.Elems
                                } else {
                                        conds = append(conds, t.Value)
                                }
                        }
                }
                if conds == nil {
                        generic.options = append(generic.options, opt)
                        continue
                }
                for _, cond := range conds {
                        if t, e := cond.True(); e != nil {
                                p.error(x.Pos(), "%v", e)
                        } else if !t {
                                generic.dontOperate = true
                                break
                        }
                }
        }

	if p.tok == token.LPAREN {
		lparen = p.pos
		p.next()
		for iota := 0; p.tok != token.RPAREN && p.tok != token.EOF; iota++ {
			specs = append(specs, f(p.leadComment, &generic, iota))
                        if p.tok == token.COMMA || p.tok == token.LINEND {
                                p.next()
                        }
		}
		rparen = p.expect(token.RPAREN)
                if p.tok != token.EOF {
                        p.expectLinend()
                }
	} else {
		for iota := 0; p.tok != token.LINEND && p.tok != token.EOF; iota++ {
                        spec := f(nil, &generic, iota)
                        specs = append(specs, spec)
                        if p.lineComment != nil {
                                break
                        }

                        // Checking for `include xxx:[...]`
                        if inc, _ := spec.(*ast.IncludeSpec); inc != nil && len(inc.Props) > 0 {
                                if p, ok := inc.Props[0].(*ast.IncludeRuleClause); ok && p != nil {
                                        goto GoodEnd
                                }
                        }

                        if p.tok == token.COMMA {
                                p.next()
                        }
                }
                if p.tok != token.EOF {
                        p.expectLinend()
                }
	}

GoodEnd:
        return &ast.GenericClause{
		Doc:    doc,
		TokPos: pos,
		Tok:    keyword,
		Lparen: lparen,
		Specs:  specs,
		Rparen: rparen,
	}
}

func (p *parser) parseDefineClause(tok token.Token, ident ast.Expr) *ast.DefineClause {
	if p.tracing.enabled { defer un(trace(p, fmt.Sprintf("Define(%s)", ident))) }

        // Only accept scoped identifiers if it's ":user:" program
        if p.scope.comment == usecomment {
                switch i := ident.(type) {
                case *ast.EvaluatedExpr:
                        if _, ok := i.Data.(*selection); !ok {
                                p.error(ident.Pos(), "should use scoped names: `%v` (%T)", i.Data, i.Data)
                        }
                default:
                        p.warn(ident.Pos(), "fixme: unexpected name expression: %T", i)
                }
        }

        var (
                doc = p.leadComment
                pos = p.expect(tok)
                elems = p.parseRhsList()
                comment = p.lineComment
                value ast.Expr
        )

        // Take it from parser, since the line comment is assigned
        // to the DefineClause.
        p.lineComment = nil

        // Create List value or use the first elem.
        if n := len(elems); n == 1 {
                value = elems[0]
        } else if n > 1 {
                value = &ast.ListExpr{ elems }
        }

        return &ast.DefineClause{
                Doc: doc,
                TokPos: pos,
                Tok: tok,
                Name: ident,
                Value: value,
                Comment: comment,
        }
}

func (p *parser) parseDefine(ident ast.Expr) (clause *ast.DefineClause) {
        clause = p.parseDefineClause(p.tok, ident)
        p.define(clause)
        return
}

func (p *parser) parseRecipeDefineClause(x ast.Expr) ast.Expr {
        // TODO: validate x ...
        d := p.parseDefineClause(p.tok, x)
        return &ast.RecipeDefineClause{ d }
}

func (p *parser) parseRecipeRuleClause(elems []ast.Expr) (x ast.Expr) {
        d := p.parseRuleClause(p.tok, specialRuleRec, nil, elems)
        return &ast.RecipeRuleClause{ d }
}

func (p *parser) parseRecipeExpr() ast.Expr {
	if p.tracing.enabled { defer un(trace(p, "Recipe")) }

        var (
                comment *ast.CommentGroup
                elems []ast.Expr
                doc = p.leadComment
                pos = p.pos
        )

SwitchDialect:
        switch p.dialect {
        case "", "eval", "value":
                p.scanner.LeaveCompoundLineContext()
                p.next() // skip RECIPE or SEMICOLON and parse in list mode
                if !p.isEndOfLine() {
                        var isVal = p.dialect == "value"
                        var bits = p.setbit(parsingBuiltinCommand)
                        var x = p.parseExpr(!isVal) // parse first expr of recipe
                        p.setbits(bits) // restore bits
                        if v := p.expr(x); v == nil {
                                p.error(x.Pos(), "`%v` is nil (%T)", x, x)
                        } else if t, ok := v.(*Bareword); ok && !isVal {
                                if sym, err := p.resolve(t); err != nil {
                                        p.error(x.Pos(), "resolve builtin: %v", err)
                                } else if sym == nil {
                                        p.error(x.Pos(), "undefined builtin %v", t.string)
                                } else {
                                        x = &ast.EvaluatedExpr{ x, sym }
                                }
                        } else {
                                x = &ast.EvaluatedExpr{ x, v }
                        }

                        if p.tok.IsAssign() {
                                elems = append(elems, p.parseRecipeDefineClause(x))
                                break SwitchDialect
                        }

                        elems = append(elems, x)
                        cmdarg := new(ast.ListExpr)
                        for p.tok != token.EOF && p.tok != token.SEMICOLON && p.tok != token.LINEND && p.lineComment == nil {
                                if p.tok.IsRuleDelim() {
                                        x = p.parseRecipeRuleClause(elems)
                                } else {
                                        x = p.parseExpr(true)
                                }

                                cmdarg.Elems = append(cmdarg.Elems, x)
                                if p.tok == token.COMMA {
                                        p.next()
                                        elems = append(elems, cmdarg)
                                        cmdarg = new(ast.ListExpr)
                                }
                                if p.lineComment != nil {
                                        comment = p.lineComment
                                        break
                                }
                        }
                        elems = append(elems, cmdarg)
                }

        default:
                p.next() // skip RECIPE or SEMICOLON and parse in line-string mode
                for !p.isEndOfLine() { elems = append(elems, p.parseExpr(false)) }
        }
        if p.tok != token.EOF { p.expectLinend() }
        if len(elems) > 0 { pos = elems[0].Pos() }
        return &ast.RecipeExpr{
                Dialect: p.dialect,
                Doc:     doc,
                TabPos:  pos,
                Elems:   elems,
                Comment: comment,
        }
}

func (p *parser) parseModifySetVar(args []ast.Expr) (err error) {
        // Parsing (var a=xxx,b=yyy) definitions
        for _, elem := range args[1:] {
                var kv, ok = elem.(*ast.KeyValueExpr)
                if !ok || kv == nil {
                        p.error(elem.Pos(), "bad var form (%T)", elem)
                        continue
                }
                var name string
                var k, v = p.expr(kv.Key), p.expr(kv.Value)
                if name, err = k.Strval(); err != nil {
                        p.error(kv.Key.Pos(), "%s", err)
                } else if name == "" {
                        p.error(kv.Key.Pos(), "'%v' name is empty ", kv.Key)
                }
                if def, alt := p.def(elem.Pos(), name); alt != nil {
                        p.error(kv.Key.Pos(), "%T '%s' already existed", alt, name)
                } else if def != nil {
                        if g, ok := v.(*Group); ok {
                                def.setval(g.ToList())
                        } else {
                                def.setval(v)
                        }
                }
        }
        return
}

func (p *parser) parseModifyParms(args []ast.Expr) (err error) {
        for _, elem := range args {
                switch elem.(type) {
                case *ast.Bareword, *ast.Barecomp:
                        var v = p.expr(elem)
                        var s string
                        if s, err = v.Strval(); err != nil {
                                p.error(elem.Pos(), "%s", err)
                        }
                        var def, alt = p.def(elem.Pos(), s)
                        if alt != nil {
                                var ok bool
                                if def, ok = alt.(*Def); !ok {
                                        p.error(elem.Pos(), "%T '%s' already taken the name, no such parameter", alt, s)
                                }
                        }
                        if def != nil {
                                def.set(DefArg, nil)
                        } else {
                                p.error(elem.Pos(), "'%s' is not defined", s)
                        }
                        p.params = append(p.params, def)
                        p.scope.replace(strconv.Itoa(len(p.params)), def)
                default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
                        p.error(elem.Pos(), "bad parameter form (%T)", elem)
                }
        }
        return
}

func (p *parser) parseModifiersExpr() *ast.ModifiersExpr {
	if p.tracing.enabled { defer un(trace(p, "Modifiers")) }

        var (
                lpos = p.expect(token.LBRACK)
                elems []ast.Expr
        )

        defer func(a parsingBits) { p.bits = a }(p.bits)
        p.bits |= composingModifier

ForModifiersExpr:
        for p.tok != token.RBRACK && p.tok != token.EOF {
                var (
                        x = p.checkExpr(p.parseExpr(false))
                        group, ok = x.(*ast.GroupExpr)
                        name string
                        pos token.Pos
                        err error
                )
                if pos == token.NoPos { /* unused */ }
                if !ok {
                        p.error(x.Pos(), "unsupported modifier")
                        continue ForModifiersExpr
                }
                if l, ok := group.Elems[0].(*ast.ListExpr); ok {
                        group.Elems = append([]ast.Expr{l.Elems[0]},
                                append(l.Elems[1:], group.Elems[1:]...)...)
                }

                switch n := group.Elems[0].(type) {
                case *ast.Bareword:
                        if name, pos = n.Value, n.Pos(); name == "var" {
                                p.parseModifySetVar(group.Elems)
                                continue ForModifiersExpr
                        } else if name == "configure" { p.configure = true }
                        goto checkName
                case *ast.GroupExpr: // parameters: ((foo bar))
                        p.parseModifyParms(n.Elems)
                        continue ForModifiersExpr
                case *ast.DelegateExpr, *ast.ClosureExpr, *ast.Barecomp, *ast.BasicLit:
                        var v []Value
                        if v, err = mergeresult(ExpandAll(p.expr(n))); err != nil {
                                p.error(n.Pos(), "%v", err)
                        } else if name, err = v[0].Strval(); err != nil {
                                p.error(n.Pos(), "%v", err)
                                continue ForModifiersExpr
                        } else if name == "" {
                                p.error(n.Pos(), "empty name (%v)", n)
                                continue ForModifiersExpr
                        }
                        pos = x.Pos()
                        goto checkName
                default:
                        p.error(n.Pos(), "unsupported dialect or modifier (%T): %v", group.Elems[0], group.Elems[0])
                        continue ForModifiersExpr
                }

                goto addModifier

        checkName:
                if _, ok = dialects[name]; ok {
                        if p.dialect == "" { p.dialect = name } else {
                                p.error(pos, "multi-dialects unsupported, already defined '%s'", p.dialect)
                                continue ForModifiersExpr
                        }
                } else if _, ok = modifiers[name]; !ok {
                        p.error(pos, "`%s` no such dialect or modifier", name)
                        continue ForModifiersExpr
                }
                
        addModifier:
                elems = append(elems, x)
        }
        rpos := p.expect(token.RBRACK)
        if p.tok == token.COLON { p.error(p.pos, "unexpected colon after modifer") }
        return &ast.ModifiersExpr{
                Lbrack: lpos,
                Elems: elems,
                Rbrack: rpos,
        }
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
        "@",  "%",  "<",  "?",  "^",  "+",  "|",  "*",  //
        "@D", "%D", "<D", "?D", "^D", "+D", "|D", "*D", //
        "@F", "%F", "<F", "?F", "^F", "+F", "|F", "*F", //
        "@'", "%'", "<'", "?'", "^'", "+'", "|'", "*'", //
        "-",  "~",
}

func (p *parser) parseRuleClause(tok token.Token, special specialRule, options, targets []ast.Expr) *ast.RuleClause {
        if p.project.keyword == token.PACKAGE {
                p.error(p.pos, "rules forbidden: %v", targets)
                return nil
        } else if p.tracing.enabled {
                defer un(trace(p, "Rule"))
        }

        var (
                doc = p.leadComment
                pos = p.expect(tok)
                depends []ast.Expr
                ordered []ast.Expr
                recipes []ast.Expr
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

        ls := p.openScope(scopeComment)
        for _, s := range automatics {
                var def, alt = p.def(p.pos, s)
                if alt != nil {
                        p.error(p.pos, "Name `%s' already taken, not automatic (%T).", s, alt)
                } else if def == nil {
                        p.error(p.pos, "'%s' is not defined", s)
                } else {
                        def.origin = DefAuto
                }
        }
        for i := 1; i < 10; i += 1 {
                var def, alt = p.def(p.pos, strconv.Itoa(i))
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

        if p.tok != token.SEMICOLON && p.tok != token.BAR && !p.isEndOfLine() {
                depends = p.parseDependList()
        }
        if p.tok == token.BAR {
                p.next() // '|' starts the ordered prerequisites
                if p.tok != token.SEMICOLON && !p.isEndOfLine() {
                        ordered = p.parseDependList()
                }
        }

        if p.tok == token.SEMICOLON { // :;
                // Parse inline recipe in the program scope.
                recipes = append(recipes, p.parseRecipeExpr())
        } else if p.tok == token.LINEND || p.lineComment != nil {
                // Proceed with the next line
                p.expectLinend() // Take the new line
                // Parse recipes in the program scope.
                for p.tok != token.EOF && p.tok == token.RECIPE {
                        recipes = append(recipes, p.parseRecipeExpr())
                }
        }

        var params []string
        if p.configure {
                if name, ok := targets[0].(*ast.Bareword); ok {
                        proj := p.project
                        d, a := proj.scope.define(proj, name.Value, nil)
                        if d == nil && a == nil {
                                p.error(targets[0].Pos(), "Cannot define configure target (%v)", name)
                        } else if a != nil {
                                if _, ok := a.(*Def); !ok {
                                        p.error(targets[0].Pos(), "Configure target name already taken (%T %v)", a, a)
                                }
                        }
                        if d != nil && !d.position.IsValid() {
                                d.position = Position(p.file.Position(name.Pos()))
                        }
                } else {
                        p.error(targets[0].Pos(), "Configure target is not bareword (%v, %T)", targets[0], targets[0])
                }
        } else {
                for _, d := range p.params { params = append(params, d.name) }
        }

        clause := &ast.RuleClause{
                Doc: doc,
                TokPos: pos,
                Tok: tok,
                Targets: p.convertBarefiles(targets),
                Depends: p.convertBarefiles(depends),
                Ordered: p.convertBarefiles(ordered),
                Position: p.file.Position(pos),
                Program: &ast.ProgramExpr{
                        Lang: 0, // FIXME: language definition
                        Configure: p.configure,
                        Params: params,
                        Recipes: recipes,
                        Scope: ls.scope,
                },
        }

        // Close the rule scope and go back to project scope. The current
        // scope must be project scope befor Rule.
        p.closeScope(ls)
        p.configure = false
        p.dialect = ""
        p.params = nil

        if special != specialRuleRec { p.rule(clause, special, options) }
        return clause
}

func (p *parser) parseSpecialRuleClause() ast.Clause {
	if p.tracing.enabled {
		defer un(trace(p, "SpecialRule"))
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

                var options []ast.Expr
                var pos = p.expect(token.BAREWORD) // USE
                var bits = p.setbit(parsingSpecialRule)
                // Options are *Flag or *Pair of a Flag.
                for p.tok == token.MINUS {
                        opt := p.checkExpr(p.parseExpr(false))
                        options = append(options, opt)
                }
                p.setbits(bits) // restore bits
                if p.tok.IsRuleDelim() {
                        return p.parseRuleClause(p.tok, specialRuleUse, options, []ast.Expr{
                                &ast.Bareword{ pos, name },
                        })
                }

                p.error(p.pos, "expecting special rule terminator ':'")
                return nil
        default:
                p.error(p.pos, "unknown special rule")
                return nil
        }
}

func (p *parser) parseClause(sync func(*parser)) ast.Clause {
 	switch p.tok {
        case token.USE/*token.IMPORT*/:
                pos := p.pos
                p.error(pos, "`%v` unexpected here", p.tok)
                syncClause1(p)
                return &ast.BadClause{From: pos, To: p.pos}
	case token.INCLUDE:
                return p.parseGenericClause(token.INCLUDE, p.expect(token.INCLUDE), p.parseIncludeSpec)
	case token.INSTANCE:
                return p.parseGenericClause(token.INSTANCE, p.expect(token.INSTANCE), p.parseInstanceSpec)
        case token.CONFIGURATION:
                return p.parseGenericClause(token.CONFIGURATION, p.expect(token.CONFIGURATION), p.parseConfigurationSpec)
        case token.FILES:
                return p.parseGenericClause(token.FILES, p.expect(token.FILES), p.parseFilesSpec)
        case token.EVAL:
                return p.parseGenericClause(token.EVAL, p.expect(token.EVAL), p.parseEvalSpec)
	/*case token.USE:
                var options []ast.Expr
                var pos = p.expect(token.USE)
                for p.tok == token.MINUS {
                        opt := p.checkExpr(p.parseExpr(false))
                        options = append(options, opt)
                }
                if p.tok.IsRuleDelim() {
                        return p.parseRuleClause(p.tok, specialRuleNor, options, []ast.Expr{
                                &ast.Bareword{ pos, token.USE.String() },
                        })
                }
                return p.parseGenericClause(token.USE, pos, p.parseUseSpec)*/
        case token.COLON:
                return p.parseSpecialRuleClause()
        }

        if p.tracing.enabled {
                defer un(trace(p, "Clause(?)"))
        }

        x := p.parseExpr(true)
        if p.tok.IsAssign() {
                return p.parseDefine(x)
        }

        list := []ast.Expr{ x }
        if !p.tok.IsRuleDelim() {
                list = append(list, p.parseLhsList()...)
        }
        if p.tok.IsRuleDelim() {
                return p.parseRuleClause(p.tok, specialRuleNor, nil, list)
        }

        pos := p.pos
        p.errorExpected(pos, "assign or colon")
        syncClause1(p)
        return &ast.BadClause{From: pos, To: p.pos}
}

func parseUsingNameProps(nameprops string) (name string, parts []string, optUnique, optReverse bool) {
	parts = strings.Split(nameprops, ".")
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

func (p *parser) applyUseeVars(pos token.Pos, proj *Project, using Value) {
	var userProj = p.scope.project // aka. p.project
	if s, e := using.Strval(); e == nil {
		position := Position(p.file.Position(pos))
		names := strings.Fields(s)
		for _, nameprops := range names {
			var (
				name, _, optUnique, optReverse = parseUsingNameProps(nameprops)
				usingVarName = fmt.Sprintf("using.%s", name)
				def *Def; alt Object
			)
			def, alt = proj.scope.define(proj, usingVarName, &None{trivial{position}})
			if def == nil && alt != nil { def, _ = alt.(*Def) }
			for _, base := range proj.bases {
				if obj, err := base.resolveObject(usingVarName); err == nil && !(isNil(obj) || isNone(obj)) {
					def.append(obj)
				}
			}
			if l, e := proj.using.Get(name); e == nil {
				if false { p.info(pos, "%v: %v; %v: %v; (user: %v)", proj, def, name, l, userProj) }
				e = def.append(l)
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
				p.error(pos, "%v: %v (usng.%s)", proj, e, name)
			}
		}
	} else {
		p.error(pos, "%v: %v", proj, using)
	}
}

func (p *parser) parseFile() *ast.File {
        if optionTraceLaunch { defer un(trace(t_launch, "parser.parseFile")) }
	if p.tracing.enabled { defer un(trace(p, "File '"+p.file.Name()+"'")) }

	// Don't bother parsing the rest if we had errors scanning the first token.
	// Likely not a Go source file at all.
	if p.errors.Len() != 0 { return nil }

        var (
                filename = p.file.Name()
                abs, rel, tmp string
        )
        if filename == confinitFilename {
                abs, rel = context.workdir, "."
                tmp = joinTmpPath(context.workdir, rel)
        } else {
                if p.tracemode&Flat != 0 {
                        abs = p.project.absPath
                } else {
                        abs = filepath.Dir(filename)
                }
                rel, _ = filepath.Rel(p.workdir, abs)
                tmp = joinTmpPath(p.workdir, rel)
        }

        var (
                doc, pos = p.leadComment, p.pos
                ls = p.openScope(fmt.Sprintf("file %s", filename))
        )
        if ls.scope != nil {
                defer p.closeScope(ls)
                var ( def *Def ; s = ls.scope )
                if p.tracemode&Flat == 0 {
                        var pos = Position(p.file.Position(p.pos))

                        def, _ = p.def(p.pos, "/")
                        def.set(DefAuto, MakePathStr(pos,abs))

                        def, _ = p.def(p.pos, ".")
                        def.set(DefAuto, MakePathStr(pos,rel))

                        def, _ = p.def(p.pos, "CTD") // Current Temp Directory
                        def.set(DefAuto, MakePathStr(pos,tmp))

                        def, _ = p.def(p.pos, "CWD") // Current Work Directory
                        def.set(DefAuto, MakePathStr(pos,abs))
                } else if def = s.FindDef("/"); def == nil {
                        p.error(p.pos, "/ not in the scope (%v)", s.comment)
                } else if def = s.FindDef("."); def == nil {
                        p.error(p.pos, ". not in the scope (%v)", s.comment)
                } else if def = s.FindDef("CTD"); def == nil {
                        p.error(p.pos, "CTD not in the scope (%v)", s.comment)
                } else if def = s.FindDef("CWD"); def == nil {
                        p.error(p.pos, "CWD not in the scope (%v)", s.comment)
                }
        } else {
                p.error(p.pos, "open scope")
        }

        var ident *ast.Bareword
        var keyword = p.tok
        switch keyword {
        case token.CONFIGURE:
                switch p.next(); p.tok {
                case token.DOT:
                        if err := p.ParseConfigDir(abs, abs); err != nil {
                                p.error(p.pos, "configure %v: %v", abs, err)
                        } else {
                                p.next() // drop the '.' token
                        }

                        basename := filepath.Base(filepath.Dir(filename))
                        ident = &ast.Bareword{ ValuePos: pos, Value: basename }

                default:
                        p.error(p.pos, "unknown configuration '%v', currently only 'configure .' is supported", p.tok)
                }
        case token.PROJECT, token.PACKAGE, token.MODULE:
                if p.tracemode&Flat != 0 {
                        p.error(p.pos, "forbidden `%v` in flat file", p.tok)
                }

                // TODO: generate ast.Project, ast.Package, ast.Module

                p.next()
                
                // Options are *Flag or *Pair of a Flag.
                var options []Value
                for p.tok == token.MINUS {
                        opt := p.checkExpr(p.parseExpr(false))
                        options = append(options, p.expr(opt))
                }
                
                // Smart-lang spec:
                //   * the project clause is not a declaration;
                //   * the project name does not appear in any scope.
                if p.tok == token.LPAREN || p.tok == token.LINEND {
                        s := filepath.Base(filepath.Dir(filename))
                        // TODO: validate basename as identifier
                        ident = &ast.Bareword{ ValuePos: pos, Value: s }
                } else if p.tok == token.TILDE {
                        if filename == confinitFilename {
                                ident = &ast.Bareword{ ValuePos:pos, Value:"~" }
                        } else if ext := filepath.Ext(filename); ext != ".smart" {
                                p.error(p.pos, "`%v` not a smart file", filepath.Base(filename))
                        } else if s := strings.TrimSuffix(filepath.Base(filename), ext); s == "~" {
                                ident = &ast.Bareword{ ValuePos: pos, Value: s }
                        } else {
                                p.error(p.pos, "`%v` not tilde name", filepath.Base(filename))
                        }
                        p.next() // skip tilde
                } else {
                        x := p.parseBarewordConstant(false)
                        if ident, _ = x.(*ast.Bareword); ident == nil {
                                p.error(p.pos, "invalid package name %T", x)
                        }
                }
                
                if ident.Value == "_" && p.tracemode&DeclarationErrors != 0 {
                        p.error(p.pos, "invalid package name _")
                }

                var params []Value
                if p.tok == token.LPAREN {
                        params = p.expr(p.parseGroupExpr(false)).(*Group).Elems
                }

                p.expectLinend()

                // Don't bother parsing the rest if we had errors parsing the package clause.
                // Likely not a Go source file at all.
                if p.errors.Len() != 0 { return nil }

                if p.tracemode&Flat == 0 {
                        if err := p.declare(keyword, ident, options, params); err != nil {
                                p.error(ident.Pos(), "%v", err)
                        } else {
				if filepath.Base(filename) == "build.smart" {
					defer func(pos token.Pos, proj *Project) {
						var using Value
						if o, e := proj.resolveObject("using.*"); e == nil && !isNil(o) {
							if def, ok := o.(*Def); ok && !isNil(def) { using = def.value }
						}
						if !isNil(using) {
							p.applyUseeVars(pos, proj, using)
						}
					} (ident.Pos(), p.project)
				}
                                defer p.closeCurrent(ident)
                        }
                }
        default:if p.tracemode&Flat == 0 {
                p.errorExpected(pos, "configure, project, module or package keyword")
        }}

	var clauses []ast.Clause
	if p.tracemode&ModuleClauseOnly == 0 {
                if p.tracemode&Flat == 0 {
                ForInit:
                        for p.tok != token.EOF {
                                switch p.tok {
                                case token.IMPORT:
                                        p.error(p.pos, "keyword `import` is replaced by `use`")
                                        syncClause1(p)
                                case token.LINEND:
                                        p.next() // skip empty lines
                                case token.USE:
                                        clauses = append(clauses, p.parseGenericClause(p.tok, p.expect(p.tok), p.parseUseSpec))
                                case token.EVAL:
                                        clauses = append(clauses, p.parseGenericClause(p.tok, p.expect(token.EVAL), p.parseEvalSpec))
                                default:
                                        if p.tok.IsKeyword() {
                                                break ForInit
                                        } else if x := p.parseExpr(true); p.tok.IsAssign() {
                                                clauses = append(clauses, p.parseDefine(x))
                                        } else if p.tok.IsRuleDelim() {
                                                if p.project == nil {
                                                        p.error(p.pos, "no project declared before defining rules")
                                                } else {
                                                        clause := p.parseRuleClause(p.tok, specialRuleNor, nil, []ast.Expr{x})
                                                        clauses = append(clauses, clause)
                                                }
                                                break ForInit
                                        } else {
                                                p.error(p.pos, "unexpected %v (after %v)", p.tok, x)
                                                syncClause1(p)
                                        }
                                }
                        }
                }
		if p.tracemode&ImportsOnly == 0 {
			// rest of module body
			for p.tok != token.EOF {
                                switch p.tok {
                                case token.LINEND:
                                        p.next() // skip empty lines
                                default:
                                        clauses = append(clauses, p.parseClause(syncClause1))
                                }
			}
		}
	}

	return &ast.File{
		Doc:        doc,
		KeyPos:     pos,
                Keyword:    keyword,
		Name:       ident,
		Scope:      ls.scope,
		Clauses:    clauses,
		Imports:    p.imports,
		Comments:   p.comments,
	}
}

func (p *parser) parseText() (res []ast.Expr) {
        if optionTraceLaunch { defer un(trace(t_launch, "parser.parseText")) }
        for p.tok != token.EOF {
                res = append(res, p.parseExpr(false))
        }
        return
}
