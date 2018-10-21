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
        "strconv"
        "strings"
        "unicode"
        "errors"
        "fmt"
)

type parsingBits uint
const (
        composing parsingBits = 1<<iota
        composingSELECT
        composingPERIOD
        composingDOTDOT
        composingPCON
        composingPERC
        
        // Bits to disable parsing ArgumentedExpr 
        composingNoArg = composingSELECT | composingPERIOD | composingDOTDOT | composingPCON | composingPERC
        composingNoPair = composingSELECT | composingPERIOD | composingPCON | composingPERC
        composingNoPerc = 0
        composingNoSelect = composingSELECT
)

type parser struct {
        *loader
        
        file    *token.File
        scanner scanner.Scanner

	// Comments
	comments    []*ast.CommentGroup
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
	imports    []*ast.ImportSpec // list of imports
}

func (p *parser) init(l *loader, filename string, src []byte) {
        p.loader = l
        p.file = l.fset.AddFile(filename, -1, len(src))
        
	var m scanner.Mode
	if p.mode&ParseComments != 0 {
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

func trace(p *parser, msg string) *parser {
	p.trace(msg, "(")
	p.level(+1)
	return p
}

// Usage pattern: defer un(trace(p, "..."))
func un(p *parser) {
	p.level(-1)
	p.trace(")")
}

func (p *parser) trace(a ...interface{}) {
	p.traceAt(p.file.Position(p.pos), a...)
}

func (p *parser) error(pos token.Pos, err interface{}, a... interface{}) {
        p.errorAt(p.file.Position(pos), err, a...)
}

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
        /* if p.lineComment != nil {
                fmt.Printf("next: %v", p.lineComment.Text())
                p.lineComment = nil
                p.tok = token.LINEND
                return
        } */
        
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
        fmt.Printf("%s:info: ", p.file.Position(pos))
        fmt.Printf(s, a...)
}

func (p *parser) warn(pos token.Pos, s string, a... interface{}) {
        if !strings.HasSuffix(s, "\n") {
                s += "\n"
        }
        fmt.Printf("%s: ", p.file.Position(pos))
        fmt.Printf(s, a...)
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
                syncClause(p)
	}
}

// ----------------------------------------------------------------------------
// Parsing

// syncClause advances to the next tok.
// Used for synchronization after an error.
//
func sync(p *parser, tok token.Token) {
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

// syncClause advances to the next declaration.
// Used for synchronization after an error.
//
func syncClause(p *parser) {
	for {
		switch p.tok {
		case token.IMPORT, token.INCLUDE, token.FILES, token.INSTANCE, token.USE, token.EXPORT, token.EVAL, token.DOCK:
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
        case *ast.KeyValueExpr:
        case *ast.PathSegExpr:
        case *ast.PathExpr:
        case *ast.PercExpr:
        case *ast.SelectionExpr:
        case nil:
                //p.warn(p.pos, "nil expression")
		p.errorExpected(p.pos, "nil expression")
		x = &ast.BadExpr{From:token.NoPos, To:token.NoPos}
	default:
		// all other nodes are not proper expressions
                //p.warn(x.Pos(), "bad expression (%T)\n", x)
		p.errorExpected(x.Pos(), "bad expression (%T)", x)
		x = &ast.BadExpr{From: x.Pos(), To: p.safePos(x.End())}
	}
	return x
}

// ----------------------------------------------------------------------------
// Barewords & Identifiers

func (p *parser) parseBadExpr(lhs bool) (x ast.Expr) {
        pos := p.pos
        p.warn(pos, "bad '%v'\n", p.tok)
        p.errorExpected(pos, "clause or expression")
        p.next() // go to next token
        return &ast.BadExpr{ From:pos, To:p.pos }
}

func (p *parser) parseBareword(lhs bool) (x ast.Expr) {
	var pos, value = p.pos, ""
        switch p.tok {
	case token.BAREWORD:
                value = p.lit
        case token.AT, token.PERIOD, token.DOTDOT:
                value = p.tok.String() // Special bareword.
        default:
                if p.tok.IsKeyword() {
                        value = p.tok.String()
                } else {
                        p.expect(token.BAREWORD)
                }
	}

        x = &ast.Bareword{ ValuePos: pos, Value:value }

        p.next() // skip bareword
        return
}

func (p *parser) parseSelect(lhs ast.Expr) (res ast.Expr) {
	if p.tracing.enabled {
		defer un(trace(p, "Select"))
	}

        tok := p.tok // the arrow '->' or '=>'
        p.next() // skip '->' or '=>'

        defer p.setbits(p.setbit(composingSELECT))

        rhs := p.checkExpr(p.parseExpr(false))
        res = &ast.SelectionExpr{ lhs, tok, rhs }
        if (p.tok == token.SELECT || p.tok == token.ARROW) && rhs.End() == p.pos {
                // Continue the selection recursivly.
                res = p.parseSelect(res)
        }
        return
}

// ----------------------------------------------------------------------------
// Common productions

func (p *parser) isEndOfList(lhs bool) bool {
        // If there's a comment right after the parsed expression, we break
        // the expression list to treat the end-of-line comment like a LINEND.
        if p.lineComment != nil {
                return true
        }
        if p.tok == token.RPAREN || p.tok == token.RBRACK || p.tok == token.RBRACE /*|| p.tok == token.COLON_RBK*/ {
                return true
        }
        if p.tok.IsRuleDelim() {
                return true
        }
        return p.tok == token.EOF || p.tok == token.LINEND || p.lineComment != nil ||
                p.tok == token.COMMA || (lhs && p.tok.IsAssign())
}

// If lhs is set, result list elements which are identifiers are not resolved.
func (p *parser) parseExprList(lhs bool) (list []ast.Expr) {
	if p.tracing.enabled {
		defer un(trace(p, "List"))
	}
        for !p.isEndOfList(lhs) {
                x := p.checkExpr(p.parseExpr(lhs))
                list = append(list, x)
                // If there's a comment right after the parsed expression, we break
                // the expression list to treat the end-of-line comment like a LINEND.
                if p.lineComment != nil {
                        break
                }
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
	if p.tracing.enabled {
		defer un(trace(p, "Group"))
	}
        
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
	if p.tracing.enabled {
		defer un(trace(p, "Argumented"))
	}

        p.next() // skip token.LPAREN
        
        var a = []ast.Expr{ p.parseListExpr(false) }
        for p.tok == token.COMMA {
                p.next() // skip token.COMMA
                a = append(a, p.parseListExpr(false))
        }

        return &ast.ArgumentedExpr{ X:x, Arguments:a, EndPos:p.expect(token.RPAREN) }
}

func (p *parser) parseGlobExpr(lhs bool) (x ast.Expr) {
	if p.tracing.enabled {
		defer un(trace(p, "Glob"))
	}
        
        x = &ast.GlobExpr{ TokPos:p.pos, Tok:p.tok }
        p.next() // skip '*'
        return x
}

func (p *parser) parsePercExpr(lhs bool, x ast.Expr) ast.Expr {
	if p.tracing.enabled {
		defer un(trace(p, "Perc"))
	}

        var (
                y ast.Expr
                pos = p.pos
        )
        if p.next(); pos+1 == p.pos { // joint, e.g. '%.o', but skip '% .o'
                y = p.checkExpr(p.parseExpr(false))
        }

        return &ast.PercExpr{
                X: x,
                OpPos: pos,
                Y: y,
        }
}

func (p *parser) parseKeyValueExpr(x ast.Expr) ast.Expr {
	if p.tracing.enabled {
		defer un(trace(p, "Pair"))
	}
        
        pos, tok := p.pos, p.tok; p.next()
        return &ast.KeyValueExpr{
                Key:   x,
                Tok:   tok,
                Equal: pos,
                Value: p.parseExpr(false),
        }
}

func (p *parser) parseFlagExpr(lhs bool) ast.Expr {
	if p.tracing.enabled {
		defer un(trace(p, "Flag"))
	}

        var (
                pos = p.pos
                x ast.Expr
        )
        // Flag expressions, excluding "-)" "-]" "-}" "-\n"
        if p.next(); p.tok == token.RPAREN || p.tok == token.RBRACK || p.tok == token.RBRACE ||
                p.tok == token.LINEND || p.lineComment != nil {
                x = &ast.Bareword{ ValuePos: p.pos }
        } else if false {
                x = p.checkExpr(p.parseExpr(false))
        } else {
                x = p.checkExpr(p.parseUnaryExpr(false))
        }
        return &ast.FlagExpr{ DashPos: pos, Name: x }
}

func (p *parser) parseBasicLit(lhs bool) ast.Expr {
        //fmt.Printf("%s %s\n", p.tok, p.lit)
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
        for p.tok != token.COMPOSED && p.tok != token.EOF {
                elems = append(elems, p.checkExpr(p.parseExpr(false)))
        }
        rpos = p.expect(token.COMPOSED)
        return &ast.CompoundLit{
                Lquote: lpos,
                Elems:  elems,
                Rquote: rpos,
        }
}

// Parses dot or dot-dot barecomp expressions and check against files.
//   .foo
//   .'foo'
//   ."foo"
//   .(foo)
//   ..foo
//   ..'foo'
func (p *parser) parseDotExpr(lhs bool, x ast.Expr) ast.Expr {
	if p.tracing.enabled {
		defer un(trace(p, "Dot"))
	}
        
        defer p.setbits(p.setbit(composingPERIOD))

        var comp *ast.Barecomp
        if comp, _ = x.(*ast.Barecomp); comp == nil {
                comp = &ast.Barecomp{ Elems:[]ast.Expr{ x } }
        }

        if p.tok == token.PERIOD {
                dot := &ast.Bareword{ p.pos, p.tok.String() }
                comp.Elems = append(comp.Elems, dot)
                p.next()
        }
        
        for comp.End() == p.pos {
                ext := p.checkExpr(p.parseExpr(false))
                comp.Elems = append(comp.Elems, ext)
                if p.tok != token.PERIOD || ext.End() != p.pos {
                        break 
                }
        }

        // FIXME: *.o => obj
        //   BUG: Barecomp{Glob . KeyValueExpr}
        //   FIX: KeyValueExpr{Barecomp, Bareword}

        x = comp

        // Processing barefile (must discluse in case of '$@.o', etc.).
        if v, e := p.eval(x, StringValue); e == nil && v != nil {
                var s string
                if s, e = v.Strval(); e != nil {
                        p.error(x.Pos(), "%s", e)
                }
                if file := p.File(s); file != nil {
                        x = &ast.Barefile{ x, file, v }
                }
        } else if e != nil {
                p.error(x.Pos(), e)
        }

        return x
}

func (p *parser) parsePathExpr(lhs bool, start ast.Expr) ast.Expr {
	if p.tracing.enabled {
		defer un(trace(p, "Path"))
	}
        
        defer p.setbits(p.setbit(composingPCON))

        var path *ast.PathExpr
        if path, _ = start.(*ast.PathExpr); path == nil {
                path = &ast.PathExpr{ Segments:[]ast.Expr{ start } }
        }

        BuildPath: for p.tok == token.PCON {
                var pos = p.pos
                for p.next(); p.tok == token.PCON && pos+1 == p.pos; {
                        pos = p.pos; p.next() // skips repeated '/' sequence
                }

                switch p.tok {
                case token.RPAREN, token.RBRACE:
                        break BuildPath
                default:if pos+1 < p.pos {
                        break BuildPath 
                }}
                
                x := p.checkExpr(p.parseComposedExpr(false)) // p.checkExpr(p.parseExpr(false))
                path.Segments = append(path.Segments, x)
                if p.tok != token.PCON || x.End() != p.pos {
                        break BuildPath
                }
        }

        /*if n := len(path.Segments); n > 1 {
                var name = path.Segments[n-1]
                if file := p.File(name); file != nil {
                        
                }
        }*/
        return path
}

func (p *parser) parseClosureDelegate() ast.Expr {
	if p.tracing.enabled {
		defer un(trace(p, "ClosureDelegate"))
	}
        
        var (
                lpos = token.NoPos
                rpos = token.NoPos
                pos  = p.pos
                tok  = p.tok
                name   ast.Expr
                rest   []ast.Expr
                tokLp  token.Token
        )
        switch p.next(); p.tok {
        case token.LPAREN, token.LBRACE:
                lpos, tokLp = p.pos, p.tok

                // skips LPAREN, LBRACE
                if name = p.checkExpr(p.parseNextExpr(false)); name == nil {
                        p.error(p.pos, "`%T` is nil", name)
                        return &ast.BadExpr{ From:p.pos, To:p.pos }
                }

                if (tokLp == token.LPAREN && p.tok != token.RPAREN) ||
                   (tokLp == token.LBRACE && p.tok != token.RBRACE) {
                        rest = append(rest, p.parseListExpr(false))
                        for p.tok == token.COMMA {
                                p.next()
                                rest = append(rest, p.parseListExpr(false))
                        }
                }
                switch tokLp {
                case token.LPAREN: rpos = p.expect(token.RPAREN)
                case token.LBRACE: rpos = p.expect(token.RBRACE)
                }
        default:
                // Only support $(...), disable $name.
                p.error(p.pos, "expecting `%v` or `%v`", token.LPAREN, token.LBRACE)
                return &ast.BadExpr{ From:p.pos, To:p.pos }
        }

        var resolved Object
        if _, ok := name.(*ast.ClosureExpr); ok {
                // closure names are resolved when used
        } else if v, err := p.eval(name, StringValue); err != nil {
                p.error(name.Pos(), err)
        } else if v == nil {
                p.error(name.Pos(), "`%v` evaluated to nil (of %T)", name, name)
        } else {
                switch tokLp {
                case token.LPAREN:
                        if resolved, err = p.resolve(v); err != nil {
                                p.error(name.Pos(), "%s", err)
                        } else if resolved == nil {
                                if tok == token.DOLLAR {
                                        p.error(name.Pos(), "`%v` is nil (%T)", v, v)
                                } else {
                                        s, _ := v.Strval()
                                        resolved = MakeUnknownObject(s)
                                }
                        } else if _, ok := resolved.(Caller); ok {
                                name = &ast.EvaluatedExpr{ name, v }
                        } else {
                                p.error(name.Pos(), "uncallable resolved `%T` (%T %v)", resolved, name, name)
                        }
                case token.LBRACE:
                        if resolved, err = p.find(v); err != nil {
                                p.error(name.Pos(), "%s", err)
                        } else if resolved == nil {
                                p.error(name.Pos(), "`%v` undefined (%T)", v, name)
                        } else if _, ok := resolved.(Executer); ok {
                                name = &ast.EvaluatedExpr{ name, v }
                        } else {
                                p.error(name.Pos(), "unexecutable resolved `%T` (%T %v)", resolved, name, v)
                        }
                }
        }

        /*if _, ok := name.(*ast.SelectionExpr); ok {
                p.warn(pos, "%T %v -> %T %v\n", name, name, resolved, resolved)
        }*/
        
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
        if tok == token.DOLLAR {
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

        resolved, err := p.resolve(&Bareword{s})
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
                Name: &ast.Bareword{ p.pos, s },
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
	if p.tracing.enabled {
		// defer un(trace(p, "Unary"))
	}
        switch p.tok {
        case token.BAREWORD, token.AT:
                return p.parseBareword(lhs)
                
        case token.BIN, token.OCT, token.INT, token.HEX, token.FLOAT,
             token.DATETIME, token.DATE, token.TIME, 
             token.URI, token.STRING, token.ESCAPE:
                return p.parseBasicLit(lhs)
                
        case token.COMPOUND:
                return p.parseCompoundLit(lhs)
                
        case token.STAR:
                return p.parseGlobExpr(lhs)

        case token.DOLLAR, token.AND: // delegate, closure
                return p.parseClosureDelegate()

        case token.LPAREN:
                return p.parseGroupExpr(lhs)

        case token.TILDE, token.PERIOD, token.DOTDOT: // ~ . ..
                var dots = p.tok.String()
                tok, pos, end := p.tok, p.pos, p.pos+token.Pos(len(dots))
                if p.next(); end != p.pos { // FIXME: ~user
                        // '~', '.' or '..' used as bareword
                        return &ast.Bareword{ pos, dots }
                } else if p.tok == token.PCON { // check /
                        return p.parsePathExpr(lhs, &ast.PathSegExpr{ pos, tok })
                } else if tok == token.PERIOD {
                        return p.parseDotExpr(lhs, &ast.Bareword{ pos, dots })
                } else if tok == token.TILDE { // TODO: ~user
                        return &ast.PathSegExpr{ pos, tok }
                } else {
                        p.error(pos, "unexpected path segment")
                        return &ast.BadExpr{ From:pos, To:p.pos }
                }
                
        case token.PCON:
                return p.parsePathExpr(lhs, &ast.PathSegExpr{ p.pos, p.tok })
                
        case token.PERC: // %bar (ie. no prefix)
                return p.parsePercExpr(lhs, nil)
                
        case token.MINUS:
                return p.parseFlagExpr(lhs)

        default:
                if p.tok.IsClosure() || p.tok.IsDelegate() {
                        return p.parseSpecialClosureDelegate(lhs)
                } else if p.tok.IsKeyword() { // keywords here are barewords
                        return p.parseBareword(lhs)
                }
        }
        return p.parseBadExpr(lhs)
}

func (p *parser) parseComposedExpr(lhs bool) (x ast.Expr) {
	if p.tracing.enabled {
		defer un(trace(p, "Composed"))
	}
        switch x = p.parseUnaryExpr(lhs); p.tok { // check composible expressions
        case token.SELECT, token.ARROW: // foo->bar  foo=>bar
                if p.bits&composingNoSelect == 0 {
                        x = p.parseSelect(x)
                }
        case token.PERC: // foo%bar
                if p.bits&composingNoPerc == 0 && x.End() == p.pos {
                        x = p.parsePercExpr(lhs, x)
                }
        case token.PERIOD: // foo.bar.baz.o
                if p.bits&composingPERIOD == 0 && x.End() == p.pos {
                        x = p.parseDotExpr(lhs, x)
                }
        case token.PCON: // ie. subdir/in/somewhere
                if p.bits&composingPCON == 0 && x.End() == p.pos {
                        // Path expressions, except '-I/path/to/include'
                        switch x.(type) {
                        case *ast.FlagExpr: // By pass these expressions.
                        default: x = p.parsePathExpr(lhs, x)
                        }
                }

        case token.STAR:        // TODO: glob: foo*bar
        case token.LBRACK:      // TODO: glob: foo[1-9]bar
        case token.QUE:         // TODO: glob: foo?bar

        case token.LBRACE:      // TODO: regexp: {^.*}.o   or token.REGEXP
        }
        return
}

func (p *parser) parseExpr(lhs bool) (x ast.Expr) {
	if false && p.tracing.enabled {
                defer un(trace(p, "Expression"))
	}
        if x = p.parseComposedExpr(lhs); !lhs {
                switch p.tok {
                case token.ASSIGN: // Example: '*.o = obj'
                        if !lhs && p.bits&composingNoPair == 0 {
                                x = p.parseKeyValueExpr(x)
                        }
                case token.LPAREN:
                        if p.bits&composingNoArg == 0 && x.End() == p.pos {
                                if _, ok := x.(*ast.ArgumentedExpr); ok {
                                        p.error(x.Pos(), "multiple argument assignment")
                                }
                                x = p.parseArgumentedExpr(x)
                        }

                case token.COMPOSED, token.COMMA, token.COLON:
                case token.RPAREN, token.RBRACK, token.RBRACE:
                case token.SELECT, token.ARROW, token.LINEND:
                        // Compose nothing at this point!

                default:if p.tok != token.EOF && x.End() == p.pos {
                        // further composing
                        var y = p.parseComposedExpr(lhs)
                        if comp, _ := x.(*ast.Barecomp); comp == nil {
                                x = &ast.Barecomp{Elems:[]ast.Expr{ x, y }}
                        } else {
                                comp.Elems = append(comp.Elems, y)
                        }
                        // fmt.Printf("composed: %v (%v)\n", x, y)
                }}
        }
        return x
}

func (p *parser) parseNextExpr(lhs bool) ast.Expr {
        p.next(); return p.parseExpr(lhs)
}

// ----------------------------------------------------------------------------
// Clauses & Declarations

type parseSpecFunc func(doc *ast.CommentGroup, keyword token.Token, iota int) ast.Spec

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

func (p *parser) parseImportSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
	spec := &ast.ImportSpec{ p.parseDirectiveSpec() }
        p.imports = append(p.imports, spec)
        p.loadImportSpec(spec)
        return spec
}

func (p *parser) parseIncludeSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        spec := &ast.IncludeSpec{ p.parseDirectiveSpec() }
        p.include(spec)
        return spec
}

func (p *parser) parseUseSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        spec := &ast.UseSpec{ p.parseDirectiveSpec() }
        p.use(spec)
        return spec

}

func (p *parser) parseInstanceSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        return &ast.InstanceSpec{ p.parseDirectiveSpec() }
}

func (p *parser) parseFilesSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        spec := &ast.FilesSpec{ p.parseDirectiveSpec() }
        for _, prop := range spec.Props {
                ee, _ := prop.(*ast.EvaluatedExpr)
                if ee == nil || ee.Data == nil {
                        p.error(prop.Pos(), "bad file spec (%T)", prop)
                        continue
                }
                if false {
                        fmt.Printf("files: %T %v\n", ee.Data, ee.Data)
                }
                switch v := ee.Data.(type) {
                case *Pair:
                        var (
                                paths []string
                                s, e = v.Key.Strval()
                        )
                        if e != nil { p.error(prop.Pos(), "%s", e) }
                        switch vv := v.Value.(type) {
                        case *Group:
                                for _, elem := range vv.Elems {
                                        if s, e = elem.Strval(); e != nil { p.error(prop.Pos(), "%s", e) }
                                        paths = append(paths, s)
                                }
                        default:
                                if s, e = vv.Strval(); e != nil { p.error(prop.Pos(), "%s", e) }
                                paths = append(paths, s)
                        }
                        p.MapFile(s, paths)
                case Value:
                        if s, e := v.Strval(); e != nil { p.error(prop.Pos(), "%s", e) } else {
                                p.MapFile(s, nil)
                        }
                default:
                        p.error(prop.Pos(), "bad file spec (%T)", prop)
                }
        }
        return spec
}

func (p *parser) parseEvalSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        spec := &ast.EvalSpec{ p.parseDirectiveSpec(), nil }
        if ee, _ := spec.Props[0].(*ast.EvaluatedExpr); ee == nil {
                panic("expected evaluated expr (eval)")
        } else if name, _ := ee.Data.(Value); name == nil {
                p.error(ee.Pos(), "Invalid eval symbol (%T).", ee.Data)
        } else if s, err := name.Strval(); err != nil {
                p.error(spec.Props[0].Pos(), err)
        } else if spec.Resolved, err = p.resolve(&Bareword{s}); err != nil {
                p.error(spec.Pos(), err)
        } else if spec.Resolved == nil {
                p.error(ee.Pos(), "Undefined eval symbol `%s' (%v).", s, name)
        } else {
                p.evalspec(spec)
        }
        return spec
}

func (p *parser) parseDockSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        spec := &ast.DockSpec{ p.parseDirectiveSpec() }
        p.dock(spec)
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
                x = p.parseExpr(false)
        )
        if v, e := p.eval(x, StringValue); e == nil {
                x = &ast.EvaluatedExpr{ x, v }
        } else {
                p.error(x.Pos(), "illegal (%s)", e)
        }

        // Append the prop `x'.
        props = append(props, x)

        // Parse the parameters.
        ParamsParseLoop: for p.tok != token.EOF {
                switch p.tok {
                case token.COMMA, token.LINEND, token.RPAREN, token.RBRACE:
                        break ParamsParseLoop
                }
                if p.lineComment != nil {
                        // found a line comment at the end
                        comment = p.lineComment
                        break
                }
                x = p.parseExpr(false)
                if v, e := p.eval(x, KeepClosures|KeepDelegates); e == nil {
                        props = append(props, &ast.EvaluatedExpr{ x, v })
                } else {
                        p.error(x.Pos(), "immediate (%s)", e)
                }
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
                specs []ast.Spec
        )

	if p.tok == token.LPAREN {
		lparen = p.pos
		p.next()
		for iota := 0; p.tok != token.RPAREN && p.tok != token.EOF; iota++ {
			specs = append(specs, f(p.leadComment, keyword, iota))
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
                        specs = append(specs, f(nil, keyword, iota))
                        if p.lineComment != nil {
                                break
                        }
                        if p.tok == token.COMMA {
                                p.next()
                        }
                }
                if p.tok != token.EOF {
                        p.expectLinend()
                }
	}

	return &ast.GenericClause{
		Doc:    doc,
		TokPos: pos,
		Tok:    keyword,
		Lparen: lparen,
		Specs:  specs,
		Rparen: rparen,
	}
}

func (p *parser) parseDefineClause(tok token.Token, ident ast.Expr) ast.Clause {
	if p.tracing.enabled {
		defer un(trace(p, "Define"))
	}

        var (
                doc = p.leadComment
                pos = p.expect(tok)

                elems = p.parseRhsList()
                comment = p.lineComment

                alt bool
                def, prev *Def
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

        // Create the definition.
        if v, e := p.eval(ident, StringValue); e == nil {
                var name string
                switch t := v.(type) {
                case Object: //*Def, *RuleEntry:
                        name = t.Name() // name is previously defined (e.g. in another scope)
                        ident = &ast.Bareword{ ident.Pos(), name }
                default: //case *Bareword:
                        if name, e = v.Strval(); e != nil {
                                p.error(ident.Pos(), "%s", e)
                        }
                }

                // If doing '+=', the assignment will concate the value of the
                // symbol from the other scope with new one.
                if tok == token.ADD_ASSIGN {
                        if sym, err := p.resolve(v); err != nil {
                                p.error(ident.Pos(), "%s", err)
                        } else if sym != nil {
                                prev, _ = sym.(*Def)
                        }
                }
                
                // Always work in the current runtime scope, so it won't affect
                // any base symbols.
                if s, a := p.def(name); a != nil {
                        switch tok {
                        case token.ASSIGN, token.EXC_ASSIGN:
                                p.error(ident.Pos(), "Already defined `%v' (%v).", name, v)
                        default:
                                def, _ = a.(*Def) // override the existing
                                if def == nil {
                                        p.error(ident.Pos(), "Name `%s' already taken, not def (%T).", name, a)
                                }
                        }
                        alt = true // it's the second defining this symbol
                } else if def, _ = s.(*Def); def != nil {
                        if prev != nil && prev != def {
                                def.SetOrigin(prev.Origin())
                                def.Assign(MakeDelegate(p.file.Position(pos), token.LPAREN, prev))
                        }
                } else if s != nil {
                        p.error(ident.Pos(), "Name `%s' already taken, not def (%T).", name, s)
                } else {
                        p.error(ident.Pos(), "Failed defining `%s' (%v).", name, v)
                }

                //fmt.Printf("define: %s %s %v (%T)\n", name, tok, value, value)
        } else {
                p.error(ident.Pos(), e)
                p.error(ident.Pos(), "error declosing name %T", ident)
        }
       
        var bits = KeepClosures|KeepDelegates // assumes DefaultDef
        switch tok {
        case token.ADD_ASSIGN: // +=
                if def == nil {
                        bits = EvalBits(-1)
                } else if def.Origin() == ImmediateDef {
                        bits = KeepClosures
                } else { // InvalidDef, DefaultDef, etc.
                        def.SetOrigin(DefaultDef)
                        bits = KeepClosures|KeepDelegates
                }
        case token.QUE_ASSIGN: // ?=
                if alt {
                        // bypass any eval if already defined
                        bits = EvalBits(-1); break
                }
                bits = KeepClosures|KeepDelegates
        case token.ASSIGN, token.EXC_ASSIGN: // =, !=
                if def != nil {
                        def.SetOrigin(DefaultDef)
                }
                bits = KeepClosures|KeepDelegates
        case token.SCO_ASSIGN, token.DCO_ASSIGN: // :=, ::=
                if def != nil {
                        def.SetOrigin(ImmediateDef)
                }
                bits = KeepClosures
        default:
                p.error(pos, "Unsuported assign `%v'", tok)
                def, bits = nil, EvalBits(-1)
        }

        if bits != EvalBits(-1) && def != nil {
                if v, e := p.eval(value, bits); e == nil {
                        switch tok {
                        case token.EXC_ASSIGN:
                                v, e = def.AssignExec(v)
                        case token.ADD_ASSIGN:
                                v, e = def.Append(v)
                        default:
                                v, e = def.Assign(v)
                        }
                        if e != nil {
                                p.error(value.Pos(), e)
                        } else if def.Value == nil {
                                //! Avoid creating a <nil> Def.
                                def.Value = UniversalNone
                        }
                        value = &ast.EvaluatedExpr{ value, v }
                } else {
                        p.error(pos, "%v (value=%v)", e, value)
                }
        }

        return &ast.DefineClause{
                Doc: doc,
                TokPos: pos,
                Tok: tok,
                Sym: def,
                Name: ident,
                Value: value,
                Comment: comment,
        }
}

func (p *parser) parseRecipeDefineClause(x ast.Expr) ast.Expr {
        // TODO: validate x ...
        d := p.parseDefineClause(p.tok, x).(*ast.DefineClause)
        return &ast.RecipeDefineClause{ d }
}

func (p *parser) parseRecipeRuleClause(elems []ast.Expr) (x ast.Expr) {
        var names = elems
        d := p.parseRuleClause(p.tok, names).(*ast.RuleClause)
        x = &ast.RecipeRuleClause{ d }
        return
}

func (p *parser) parseRecipeBuiltin(elems []ast.Expr) (x ast.Expr) {
        if elem, ok := elems[0].(*ast.EvaluatedExpr); ok {
                // Resolve builtin names.
                switch t := elem.Data.(type) {
                case *Bareword:
                        if sym, err := p.resolve(&Bareword{t.Value}); err != nil {
                                p.error(elem.Pos(), "%v", err)
                        } else if sym == nil {
                                p.error(elem.Pos(), "undefined builtin %v", t.Value)
                        } else {
                                elem.Data = sym
                        }
                }
        }
        
        x = p.parseExpr(true) // Do left-hand-side parsing if in use rule
        if v, e := p.eval(x, KeepClosures|KeepDelegates); e != nil {
                p.error(x.Pos(), "%v (%T)", e, x)
        } else if v != nil {
                x = &ast.EvaluatedExpr{ x, v }
        } else {
                p.error(x.Pos(), "Recipe `%T' eval to nil.", x)
        }
        return
}

func (p *parser) parseRecipeExpr(dialect string) ast.Expr {
	if p.tracing.enabled {
		defer un(trace(p, "Recipe"))
	}
        
        var (
                comment *ast.CommentGroup
                elems []ast.Expr
                doc = p.leadComment
                pos = p.pos
        )

        SwitchDialect: switch dialect {
        case "":
                p.scanner.LeaveCompoundLineContext()
                p.next() // skip RECIPE and parse in list mode
                if p.tok != token.LINEND && p.tok != token.EOF {
                        x := p.parseExpr(true) // parse first expr of recipe
                        if v, e := p.eval(x, KeepClosures|KeepDelegates); e != nil {
                                p.error(x.Pos(), "%v (%T)", e, x)
                        } else if v == nil {
                                p.error(x.Pos(), "Recipe `%T' eval to nil.", x)
                        } else {
                                x = &ast.EvaluatedExpr{ x, v }
                        }

                        if p.tok.IsAssign() {
                                elems = append(elems, p.parseRecipeDefineClause(x))
                                break SwitchDialect
                        }

                        elems = append(elems, x)
                        cmdarg := new(ast.ListExpr)
                        for p.tok != token.LINEND && p.tok != token.EOF {
                                if p.tok.IsRuleDelim() {
                                        x = p.parseRecipeRuleClause(elems)
                                        // assert(p.tok == token.LINEND || <DONE>)
                                } else {
                                        x = p.parseRecipeBuiltin(elems)
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
                p.next() // skip RECIPE and parse in line-string mode
                for p.tok != token.LINEND && p.tok != token.EOF {
                        elems = append(elems, p.parseExpr(false))
                }
        }
        if p.tok != token.EOF {
                p.expectLinend()
        }
        return &ast.RecipeExpr{
                Dialect: dialect,
                Doc:     doc,
                TabPos:  pos,
                Elems:   elems,
                Comment: comment,
        }
}

func (p *parser) parseModifierExpr() (string, []string, *ast.ModifierExpr) {
        var (
                lpos = p.expect(token.LBRACK)
                elems []ast.Expr
                dialect string
                params []string
        )
        for p.tok != token.RBRACK && p.tok != token.EOF {
                var (
                        x = p.checkExpr(p.parseExpr(false))
                        name string
                        pos token.Pos
                )
                
                switch t := x.(type) {
                /*case *ast.Bareword:
                        name, pos = t.Value, t.Pos()
                        goto checkName*/
                case *ast.GroupExpr:
                        switch n := t.Elems[0].(type) {
                        case *ast.Bareword:
                                if name, pos = n.Value, n.Pos(); name != "var" {
                                        goto checkName
                                }
                                for _, elem := range t.Elems[1:] {
                                        //fmt.Printf("var: %T\n", elem)
                                        kv, _ := elem.(*ast.KeyValueExpr)
                                        if  kv == nil {
                                                p.error(elem.Pos(), "bad var form (%T)", elem)
                                                continue
                                        }
                                        v, e := p.eval(kv.Key, StringValue)
                                        if e == nil {
                                                var name string
                                                if name, e = v.Strval(); e != nil { p.error(p.pos, "%s", e) }
                                                if sym, alt := p.def(name); alt != nil {
                                                        p.error(p.pos, "Name `%s' already taken (%T).", name, alt)
                                                } else if sym == nil {
                                                        // TODO: errors
                                                } else if v, e = p.eval(kv.Value, KeepClosures|KeepDelegates); e == nil {
                                                        sym.(*Def).Assign(v)
                                                        //fmt.Printf("var: %v\n", sym)
                                                }
                                        }
                                        if e != nil {
                                                p.error(elem.Pos(), "bad var (%T, %v)", elem, e)
                                        }
                                }
                                goto next
                        case *ast.GroupExpr:
                                for _, elem := range n.Elems {
                                        //fmt.Printf("param: %T\n", elem)
                                        switch elem.(type) {
                                        case *ast.Bareword, *ast.Barecomp:
                                                if v, e := p.eval(elem, StringValue); e == nil {
                                                        var name string
                                                        if name, e = v.Strval(); e != nil { p.error(p.pos, "%s", e) }
                                                        if sym, alt := p.def(name); alt != nil {
                                                                p.error(p.pos, "Name `%s' already taken, not parameter (%T).", name, alt)
                                                        } else if sym == nil {
                                                                // TODO: errors
                                                        } else {
                                                                //sym.(*Def).Assign(MakeString("xxxxxxxxxx"))
                                                        }
                                                        params = append(params, name)
                                                } else {
                                                        p.error(elem.Pos(), "bad parameter (%T, %v)", elem, e)
                                                }
                                        default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
                                                p.error(elem.Pos(), "bad parameter form (%T)", elem)
                                        }
                                }
                                goto next
                        case *ast.DelegateExpr, *ast.ClosureExpr, *ast.Barecomp, *ast.BasicLit:
                                v, e := p.eval(n, StringValue)
                                if e != nil {
                                        p.error(n.Pos(), "%v (%v)", e, n); goto next
                                }
                                if name, e = v.Strval(); e != nil {
                                        p.error(n.Pos(), "%v", e); goto next
                                } else if name == "" {
                                        p.error(n.Pos(), "empty name (%v)", n); goto next
                                }
                                pos = x.Pos()
                                goto checkName
                        default:
                                p.error(n.Pos(), "unsupported dialect or modifier (%T)", t.Elems[0])
                                goto next
                        }
                default:
                        p.error(x.Pos(), "unsupported modifier"); goto next
                }
                goto addModifier

                checkName: if IsDialect(name) {
                        if dialect == "" {
                                dialect = name
                        } else {
                                p.error(pos, "multi-dialect unsupported, already defined '%s'", dialect)
                                goto next
                        }
                } else if IsModifier(name) {
                        goto addModifier
                } else {
                        p.error(pos, "No such dialect or modifier `%s'", name)
                        goto next
                }
                
                addModifier: elems = append(elems, x)
                next: if p.tok == token.COMMA {
                        p.next() // TODO: grouping modifiers
                }
        }
        rpos := p.expect(token.RBRACK)
        return dialect, params, &ast.ModifierExpr{
                Lbrack: lpos,
                Elems: elems,
                Rbrack: rpos,
        }
}

// https://www.gnu.org/software/make/manual/html_node/Automatic-Variables.html#Automatic-Variables
var automatics = []string{
        "@",  "%",  "<",  "?",  "^",  "+",  "|",  "*",  //
        "@D", "%D", "<D", "?D", "^D", "+D", "|D", "*D", //
        "@F", "%F", "<F", "?F", "^F", "+F", "|F", "*F", //
        "@'", "%'", "<'", "?'", "^'", "+'", "|'", "*'", //
        "-", "CWD"/* Current Work Directory */,
}

func (p *parser) parseRuleClause(tok token.Token, targets []ast.Expr) ast.Clause {
	if p.tracing.enabled {
		defer un(trace(p, "Rule"))
	}

        var (
                doc = p.leadComment
                pos = p.expect(tok)
                modifier *ast.ModifierExpr
                program *ast.ProgramExpr
                depends []ast.Expr
                recipes []ast.Expr
                scopeComment string
                dialect string
                params []string
                isUseRule bool
        )

        scope := p.openScope(fmt.Sprintf("rule %s", scopeComment))
        for _, s := range automatics {
                if sym, alt := p.def(s); alt != nil {
                        p.error(p.pos, "Name `%s' already taken, not automatic (%T).", s, alt)
                } else if sym == nil {
                        // TODO: errors
                }
        }
        for i := 1; i < 10; i += 1 {
                if sym, alt := p.def(strconv.Itoa(i)); alt != nil {
                        p.error(p.pos, "Name `%v' already taken, not numberred (%T).", i, alt)
                } else if sym == nil {
                        // TODO: errors
                }
        }

        switch p.tok {
        case token.LBRACK: // [
                // Parse modifiers in the program scope.
                dialect, params, modifier = p.parseModifierExpr()
        case token.MINUS, token.EXC, token.QUE: // - ! ?
                p.error(p.pos, "modifier '%v' unimplemented", p.tok)
                p.next()
        }

        if p.tok == token.COLON {
                p.next()
        }
        if p.tok != token.LINEND {
                depends = p.parseExprList(false)
        }
        if p.tok != token.EOF {
                p.expectLinend()
        }

        var useContainScope ast.Scope
        if isUseRule {
                //useContainScope = p.OpenScope(p.pos, "use")
        }

        // Parse recipes in the program scope.
        for p.tok == token.RECIPE {
                recipes = append(recipes, p.parseRecipeExpr(dialect))
        }

        program = &ast.ProgramExpr{
                Lang: 0, // FIXME: language definition
                Params: params,
                Recipes: recipes,
                Scope: scope,
        }
        
        clause := &ast.RuleClause{
                Doc: doc,
                TokPos: pos,
                Tok: tok,
                Targets: targets,
                Depends: depends,
                Program: program,
                Modifier: modifier,
                Position: p.file.Position(pos),
        }

        if useContainScope != nil {
                p.closeScope(useContainScope)
        }
        
        // Close the rule scope and go back to project scope. The current
        // scope must be project scope befor Rule.
        p.closeScope(scope)
        p.rule(clause)
        return clause
}

func (p *parser) parseClause(sync func(*parser)) ast.Clause {
 	switch p.tok {
	case token.INCLUDE:
                return p.parseGenericClause(token.INCLUDE, p.expect(token.INCLUDE), p.parseIncludeSpec)
	case token.INSTANCE:
                return p.parseGenericClause(token.INSTANCE, p.expect(token.INSTANCE), p.parseInstanceSpec)
        case token.FILES:
                return p.parseGenericClause(token.FILES, p.expect(token.FILES), p.parseFilesSpec)
        case token.EVAL:
                return p.parseGenericClause(token.EVAL, p.expect(token.EVAL), p.parseEvalSpec)
        case token.DOCK:
                p.warn(p.pos, "dock clause is deprecated, use dock package instead")
                return p.parseGenericClause(token.DOCK, p.expect(token.DOCK), p.parseDockSpec)
	case token.USE:
                pos := p.expect(token.USE)
                if p.tok.IsRuleDelim() {
                        var list = []ast.Expr{
                                &ast.Bareword{
                                        ValuePos: pos, 
                                        Value: "use",
                                },
                        }
                        return p.parseRuleClause(p.tok, list)
                } else {
                        return p.parseGenericClause(token.USE, pos, p.parseUseSpec)
                }
        }

        if p.tracing.enabled {
                defer un(trace(p, "Clause(?)"))
        }

        x := p.parseExpr(true)
        if p.tok.IsAssign() {
                return p.parseDefineClause(p.tok, x)
        }

        list := []ast.Expr{ x }
        if !p.tok.IsRuleDelim() {
                list = append(list, p.parseLhsList()...)
        }
        if p.tok.IsRuleDelim() {
                return p.parseRuleClause(p.tok, list)
        }

        pos := p.pos
        p.errorExpected(pos, "assign or colon")
        sync(p)
        return &ast.BadClause{From: pos, To: p.pos}
}

func (p *parser) parseFile() *ast.File {
	if p.tracing.enabled {
		defer un(trace(p, "File '"+p.file.Name()+"'"))
	}

	// Don't bother parsing the rest if we had errors scanning the first token.
	// Likely not a Go source file at all.
	if p.errors.Len() != 0 {
		return nil
	}

        var (
                keyword token.Token
                ident *ast.Bareword
                filename = p.file.Name()
                abs = filepath.Dir(filename)
                rel , _ = filepath.Rel(p.workdir, abs)
                doc = p.leadComment
                pos = p.pos
        )
        //if strings.HasSuffix(rel, abs) {
        //        rel = abs
        //}

        scope := p.openScope(fmt.Sprintf("file %s", filename))
        if scope != nil {
                defer p.closeScope(scope)
                var sym Object

                sym, _ = p.def("/")
                sym.(*Def).Assign(MakeString(abs))

                sym, _ = p.def(".")
                sym.(*Def).Assign(MakeString(rel))
        } else {
                p.error(p.pos, "open scope")
        }

        if keyword = p.tok; keyword == token.CONFIGURE {
                switch p.next(); p.tok {
                case token.PERIOD:
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
        } else if keyword == token.PROJECT || keyword == token.MODULE {
                if p.mode&Flat != 0 {
                        p.error(p.pos, "forbidden %v in flat file", p.tok)
                }

                p.next()
                
                // Smart-lang spec:
                //   * the project clause is not a declaration;
                //   * the project name does not appear in any scope.
                if p.tok == token.LPAREN || p.tok == token.LINEND {
                        basename := filepath.Base(filepath.Dir(filename))
                        // TODO: validate base for identifier 
                        ident = &ast.Bareword{
                                ValuePos: pos,
                                Value: basename,
                        }
                } else {
                        x := p.parseBareword(false)
                        if ident, _ = x.(*ast.Bareword); ident == nil {
                                p.error(p.pos, "invalid package name %T", x)
                        }
                }
                
                if ident.Value == "_" && p.mode&DeclarationErrors != 0 {
                        p.error(p.pos, "invalid package name _")
                }

                var params Value
                if p.tok == token.LPAREN {
                        value, err := p.eval(p.parseGroupExpr(false), StringValue)
                        if err == nil {
                                params = value
                        } else {
                                p.error(p.pos, err)
                        }
                }

                p.expectLinend()

                // Don't bother parsing the rest if we had errors parsing the package clause.
                // Likely not a Go source file at all.
                if p.errors.Len() != 0 {
                        return nil
                }

                if p.mode&Flat == 0 {
                        if err := p.DeclareProject(ident, params); err != nil {
                                p.error(ident.Pos(), err)
                        } else {
                                defer p.CloseCurrentProject(ident)
                        }
                }
        } else if p.mode&Flat == 0 {
                p.errorExpected(pos, "configure, project or module keyword")
        } else {
                // TODO: Enter previously delcared project sope!
        }

	var clauses []ast.Clause
	if p.mode&ModuleClauseOnly == 0 {
                if p.mode&Flat == 0 {
                        // import clauses
                        for p.tok == token.IMPORT {
                                clauses = append(clauses, p.parseGenericClause(p.tok, p.expect(p.tok), p.parseImportSpec))
                        }
                }
		if p.mode&ImportsOnly == 0 {
			// rest of module body
			for p.tok != token.EOF {
                                 switch p.tok {
                                 case token.LINEND:
                                         p.next() // skip empty lines
                                 case token.IMPORT:
                                         p.errorExpected(p.pos, "'"+p.tok.String()+"'")
                                 default:
                                         clauses = append(clauses, p.parseClause(syncClause))
                                 }
			}
		}
	}

	return &ast.File{
		Doc:        doc,
		Keypos:     pos,
                Keyword:    keyword,
		Name:       ident,
		Scope:      scope,
		Clauses:    clauses,
		Imports:    p.imports,
		Comments:   p.comments,
	}
}

func (p *parser) parseText() (res []ast.Expr) {
        for p.tok != token.EOF {
                res = append(res, p.parseExpr(false))
        }
        return
}
