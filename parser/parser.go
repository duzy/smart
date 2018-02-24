//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package parser

import (
        "github.com/duzy/smart/ast"
        "github.com/duzy/smart/token"
        "github.com/duzy/smart/scanner"
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
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
        *Context
        
        file    *token.File
        errors  scanner.Errors
        scanner scanner.Scanner

	// Tracing/debugging
	mode   Mode // parsing mode
	trace  bool // == (mode & Trace != 0)
	indent int  // indentation used for tracing output

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

func (p *parser) init(ctx *Context, fset *token.FileSet, filename string, src []byte, mode Mode) {
        p.Context = ctx
        p.file = fset.AddFile(filename, -1, len(src))
        
	var m scanner.Mode
	if mode&ParseComments != 0 {
		m = scanner.ScanComments
	}
        
	eh := func(pos token.Position, msg string) {
                p.errors.Add(pos, errors.New(msg))
        }
	p.scanner.Init(p.file, src, eh, m)

	p.mode = mode //| Trace
	p.trace = p.mode&Trace != 0 // for convenience (p.trace is used frequently)

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

func (p *parser) printTrace(a ...interface{}) {
	const dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
	const n = len(dots)
	pos := p.file.Position(p.pos)
	fmt.Printf("%5d:%3d: ", pos.Line, pos.Column)
	i := 2 * p.indent
	for i > n {
		fmt.Print(dots)
		i -= n
	}
	// i <= n
	fmt.Print(dots[0:i])
	fmt.Println(a...)
}

func trace(p *parser, msg string) *parser {
	p.printTrace(msg, "(")
	p.indent++
	return p
}

// Usage pattern: defer un(trace(p, "..."))
func un(p *parser) {
	p.indent--
	p.printTrace(")")
}

// Advance to the next token.
func (p *parser) next0() {
	// Because of one-token look-ahead, print the previous token
	// when tracing as it provides a more readable output. The
	// very first token (!p.pos.IsValid()) is not initialized
	// (it is token.ILLEGAL), so don't print it .
	if p.trace && p.pos.IsValid() {
		s := p.tok.String()
		switch {
		case p.tok.IsLiteral():
			p.printTrace(s, p.lit)
		case p.tok.IsOperator(), p.tok.IsKeyword():
			p.printTrace("\"" + s + "\"")
		default:
			p.printTrace(s)
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

func (p *parser) error(pos token.Pos, err interface{}, a... interface{}) {
	epos := p.file.Position(pos)

	// If AllErrors is not set, discard errors reported on the same line
	// as the last recorded error and stop parsing if there are more than
	// 10 errors.
	if p.mode&AllErrors == 0 {
		n := len(p.errors)
		if n > 0 && p.errors[n-1].Pos.Line == epos.Line {
			return // discard - likely a spurious error
		}
		if n > 10 {
			panic(bailout{})
		}
	}

        var s string
        switch t := err.(type) {
        case error:  p.errors.Add(epos, t); return
        case string: s = t
        default: s = fmt.Sprintf("%v", err)
        }
        if len(a) > 0 {
                s = fmt.Sprintf(s, a...)
        }
        p.errors.Add(epos, errors.New(s))
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

func assert(cond bool, msg string) {
	if !cond {
		panic("parser internal error: " + msg)
	}
}

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
        p.warn(pos, "weird '%v'\n", p.tok)
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
	if p.trace {
		defer un(trace(p, "Select"))
	}

        defer p.setbits(p.setbit(composingSELECT))

        var (
                object types.Value
                objectName string
                fieldValue types.Value
                fieldName string
                where = anywhere
                rhs = p.checkExpr(p.parseNextExpr(false)) // skip '->'
        )

        switch t := rhs.(type) {
        case *ast.Bareword: fieldName = t.Value        // foo->xxx
        case *ast.GlobExpr: fieldName = t.Tok.String() // foo->*
        case *ast.EvaluatedExpr: // foo->bar->xxx
                if t.Data != nil {
                        if v, _ := t.Data.(types.Value); v != nil {
                                fieldName = v.Strval()
                        }
                }
                if fieldName == "" {
                        p.error(t.Pos(), "Evaluated select (right) operand `%T' invalid (%v).", t.Expr, t.Data)
                        goto DoneSelect
                }
        default:
                if v, e := p.runtime.Eval(rhs, StringValue); e == nil && v != nil {
                        fieldName = v.Strval()
                } else if v == nil {
                        p.error(t.Pos(), "Select operand (%T) evals to nil.", t)
                        goto DoneSelect
                } else {
                        p.error(t.Pos(), e)
                        p.error(t.Pos(), "Invalid select operand `%T'.", t)
                }
        }

        switch t := lhs.(type) {
        case *ast.Bareword: objectName = t.Value
        case *ast.EvaluatedExpr:
                if t.Data != nil {
                        if object = t.Data.(types.Value); object != nil {
                                goto GetFieldValue
                        }
                }
                // Error...
                p.error(t.Pos(), "Evaluated select (left) operand `%T' invalid (%v).", t.Expr, t.Data)
                goto DoneSelect
        default:
                if v, e := p.runtime.Eval(lhs, StringValue); e == nil && v != nil {
                        objectName = v.Strval()
                } else if v == nil {
                        p.error(t.Pos(), "Select operand `%T' eval to nil.", t)
                } else {
                        p.error(t.Pos(), e)
                        p.error(t.Pos(), "Invalid select operand (%T).", t)
                }
        }
        if objectName == "@" {
                // If resolving @ in a rule (program) scope selection context,
                // e.g. '$(@->FOO)', Resolve have to ensure @ is pointing to the global
                // @ package.
                where = global
        }
        if sym := p.runtime.Resolve(objectName, where); sym != nil {
                object = sym.(types.Value)
        }
        if object == nil {
                p.error(lhs.Pos(), "Undefined select operand `%s' (%v).", objectName, lhs)
                goto DoneSelect
        }

        GetFieldValue: switch o := object.(type) {
        case types.Object:
                var err error
                if fieldValue, err = o.Get(fieldName); err != nil {
                        p.error(rhs.Pos(), err)
                        p.error(lhs.Pos(), "Selection `%s->%s' failed.", objectName, fieldName)
                } else if fieldValue == nil {
                        p.error(rhs.Pos(), "No such property `%s' in `%s'.", fieldName, objectName)
                } else if pn, _ := o.(*types.ProjectName); pn != nil {
                        // Detect diverged scope ().
                        switch v := fieldValue.(type) {
                        case *types.RuleEntry:
                                if false && pn.Project() != v.Project() {
                                        p.error(rhs.Pos(), "Name diverged `%v' (%v != %v).", fieldName, pn.Project().Name(), v.Project().Name())
                                }
                        case *types.Def:
                        }
                }
        default:
                p.error(lhs.Pos(), "Not an object `%T'.", lhs)
        }

        DoneSelect: res = &ast.EvaluatedExpr{ rhs, fieldValue }
        if p.tok == token.SELECT {
                p.next() // Drop '->' before next selection.
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
        if p.tok == token.RPAREN || p.tok == token.RBRACK /*|| p.tok == token.COLON_RBK*/ {
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
	if p.trace {
		defer un(trace(p, "List"))
	}
        for !p.isEndOfList(lhs) {
                list = append(list, p.checkExpr(p.parseExpr(lhs)))
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

func (p *parser) parseRhsList(sel bool) []ast.Expr {
        defer p.setRHS(p.setRHS(true))
	list := p.parseExprList(false)
	return list
}

// ----------------------------------------------------------------------------
// Expressions

func (p *parser) parseGroupExpr(lhs bool) ast.Expr {
	if p.trace {
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
	if p.trace {
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
	if p.trace {
		defer un(trace(p, "Glob"))
	}
        
        x = &ast.GlobExpr{ TokPos:p.pos, Tok:p.tok }
        p.next() // skip '*'
        return x
}

func (p *parser) parsePercExpr(lhs bool, x ast.Expr) ast.Expr {
	if p.trace {
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
	if p.trace {
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
	if p.trace {
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
	if p.trace {
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
        if v, e := p.runtime.Eval(x, StringValue); e == nil && v != nil {
                if file := p.runtime.File(v.Strval()); file != nil {
                        x = &ast.Barefile{ x, file, v }
                }
        } else if e != nil {
                p.error(x.Pos(), e)
        }

        return x
}

func (p *parser) parsePathExpr(lhs bool, start ast.Expr) ast.Expr {
	if p.trace {
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
                if file := p.runtime.File(name); file != nil {
                        
                }
        }*/
        return path
}

func (p *parser) parseClosureDelegate() ast.Expr {
	if p.trace {
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
                name = p.checkExpr(p.parseNextExpr(false)) // skipped LPAREN, LBRACE
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
                p.error(p.pos, "Expecting `%v' or `%v'.", token.LPAREN, token.LBRACE)
                return &ast.BadExpr{ From:p.pos, To:p.pos }
        }

        var resolved RuntimeObj
        if a, _ := name.(*ast.EvaluatedExpr); a != nil {
                if a.Data == nil {
                        p.error(name.Pos(), "Evaluated data is nil (%v).", a.Expr)
                } else if resolved = a.Data.(RuntimeObj); resolved == nil {
                        p.error(name.Pos(), "Unresolved reference (%v).", a.Expr)
                }
        } else if v, e := p.runtime.Eval(name, StringValue); e != nil {
                p.error(pos, e)
        } else if v == nil {
                p.error(pos, "Name `%T' eval to nil", name)
        } else {
                a = &ast.EvaluatedExpr{ name, v }
                if resolved = p.runtime.Resolve(v.Strval(), anywhere); resolved == nil {
                        p.error(name.Pos(), "Undefined reference `%v' (%T).", v.Strval(), name)
                } else {
                        name = a
                        switch tokLp {
                        case token.LPAREN:
                                if _, ok := resolved.(types.Caller); !ok {
                                        p.error(name.Pos(), "Uncallable resolved `%v' (%T).", v.Strval(), name)
                                }
                        case token.LBRACE:
                                if _, ok := resolved.(types.Executer); !ok {
                                        p.error(name.Pos(), "Unexecutible resolved `%v' (%T).", v.Strval(), name)
                                }
                        }
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
        if tok == token.DOLLAR {
                return &ast.DelegateExpr{ cd }
        } else {
                return &ast.ClosureExpr{ cd }
        }
}

func (p *parser) parseSpecialClosureDelegate(lhs bool) ast.Expr {
	if p.trace {
		defer un(trace(p, "SpecialClosureDelegate"))
	}

        pos, tok, s := p.pos, p.tok, p.tok.String()[1:]
        p.next()

        resolved := p.runtime.Resolve(s, anywhere)
        if resolved == nil {
                p.error(pos, "Undefined reference `%v' (%v).", s, tok)
        } else if _, ok := resolved.(types.Caller); !ok {
                p.error(pos, "Uncallable resolved `%T'.", resolved)
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
	if p.trace {
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

        case token.PERIOD, token.DOTDOT:
                var dots = p.tok.String()
                tok, pos, end := p.tok, p.pos, p.pos+token.Pos(len(dots))
                if p.next(); end != p.pos {
                        // '.' and '..' used as bareword
                        return &ast.Bareword{ pos, dots }
                } else if p.tok == token.PCON { // check / after . or ..
                        return p.parsePathExpr(lhs, &ast.PathSegExpr{ pos, tok })
                } else /*if tok == token.PERIOD*/ {
                        return p.parseDotExpr(lhs, &ast.Bareword{ pos, dots })
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
                } else {
                        return p.parseBadExpr(lhs)
                }
        }
}

func (p *parser) parseComposedExpr(lhs bool) (x ast.Expr) {
	if p.trace {
		defer un(trace(p, "Composed"))
	}
        switch x = p.parseUnaryExpr(lhs); p.tok { // check composible expressions
        case token.SELECT: // foo->bar
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
	if p.trace {
		// defer un(trace(p, "Expression"))
	}
        if x = p.parseComposedExpr(lhs); !lhs {
                switch p.tok {
                case token.ARROW, token.ASSIGN: // Example: '*.o => obj'
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
                case token.LINEND: // By pass the above tokens for further composion!
                default:if x.End() == p.pos { // further composing
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
        if err, n := p.runtime.ClauseImport(spec); err != nil {
                if n < 0 {
                        p.error(spec.Pos(), err)
                } else {
                        p.error(spec.Props[n].Pos(), err)
                }
        } else {
                p.imports = append(p.imports, spec)
        }
        return spec
}

func (p *parser) parseIncludeSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        spec := &ast.IncludeSpec{ p.parseDirectiveSpec() }
        if err := p.runtime.ClauseInclude(spec); err != nil {
                p.error(spec.Pos(), err)
        }
        return spec
}

func (p *parser) parseUseSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        spec := &ast.UseSpec{ p.parseDirectiveSpec() }
        if err := p.runtime.ClauseUse(spec); err != nil {
                p.error(spec.Pos(), err)
        }
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
                case *types.Pair:
                        var (
                                paths []string
                                s = v.Key.Strval()
                        )
                        switch vv := v.Value.(type) {
                        case *types.Group:
                                for _, elem := range vv.Elems {
                                        paths = append(paths, elem.Strval())
                                }
                        default:
                                paths = append(paths, vv.Strval())
                        }
                        p.runtime.MapFile(s, paths)
                case types.Value:
                        p.runtime.MapFile(v.Strval(), nil)
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
        } else if name, _ := ee.Data.(types.Value); name == nil {
                p.error(ee.Pos(), "Invalid eval symbol (%T).", ee.Data)
        } else if spec.Resolved = p.runtime.Resolve(name.Strval(), anywhere); spec.Resolved == nil {
                p.error(ee.Pos(), "Undefined eval symbol `%s' (%v).", name.Strval(), name)
        } else if err := p.runtime.ClauseEval(spec); err != nil {
                p.error(spec.Pos(), err)
        }
        return spec
}

func (p *parser) parseDockSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        spec := &ast.DockSpec{ p.parseDirectiveSpec() }
        if err := p.runtime.ClauseDock(spec); err != nil {
                p.error(spec.Pos(), err)
        }
        return spec
}

func (p *parser) parseDirectiveSpec() (gs ast.DirectiveSpec) {
	if p.trace {
		defer un(trace(p, "Spec"))
	}
        
        var (
                doc = p.leadComment
                comment *ast.CommentGroup
                props []ast.Expr
                x = p.parseExpr(false)
        )
        if v, e := p.runtime.Eval(x, StringValue); e == nil {
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
                if v, e := p.runtime.Eval(x, KeepClosures|KeepDelegates); e == nil {
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
	if p.trace {
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
                        p.expectLinend() // endpos = p.expect(token.LINEND)
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
	if p.trace {
		defer un(trace(p, "Define"))
	}

        var (
                doc = p.leadComment
                pos = p.expect(tok)

                elems = p.parseRhsList(false)
                comment = p.lineComment

                alt bool
                def, prev *types.Def
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
        if v, e := p.runtime.Eval(ident, StringValue); e == nil {
                var name string
                switch t := v.(type) {
                case types.Object: //*types.Def, *types.RuleEntry:
                        name = t.Name() // name is previously defined (e.g. in another scope)
                        ident = &ast.Bareword{ ident.Pos(), name }
                default: //case *types.Bareword:
                        name = v.Strval()
                }

                if name == "docker-exec-image" {
                        p.error(ident.Pos(), "deprecated, use dock instead")
                }

                // If doing '+=', the assignment will concate the value of the
                // symbol from the other scope with new one.
                if tok == token.ADD_ASSIGN {
                        if sym := p.runtime.Resolve(name, anywhere); sym != nil {
                                prev, _ = sym.(*types.Def)
                        }
                }
                
                // Always work in the current runtime scope, so it won't affect
                // any base symbols.
                if s, a := p.runtime.Symbol(name, types.DefType); a != nil {
                        switch tok {
                        case token.ASSIGN, token.EXC_ASSIGN:
                                p.error(ident.Pos(), "Already defined `%v' (%v).", name, v)
                        default:
                                def, _ = a.(*types.Def) // override the existing
                                if def == nil {
                                        p.error(ident.Pos(), "Name `%s' already taken, not def (%T).", name, a)
                                }
                        }
                        alt = true // it's the second defining this symbol
                } else if def, _ = s.(*types.Def); def != nil {
                        if prev != nil && prev != def /*&& prev.Parent() != def.Parent()*/ {
                                def.SetOrigin(prev.Origin())
                                def.Assign(types.Delegate(p.file.Position(pos), prev))
                        }
                } else if s != nil {
                        p.error(ident.Pos(), "Name `%s' already taken, not def (%T).", name, s)
                } else {
                        p.error(ident.Pos(), "Failed defining `%s' (%v).", name, v)
                }

                //fmt.Printf("define: %s: %T %v\n", name, value, value)
        } else {
                p.error(ident.Pos(), e)
                p.error(ident.Pos(), "error declosing name %T", ident)
        }
       
        var bits = KeepClosures|KeepDelegates // assumes types.DefaultDef
        switch tok {
        case token.ADD_ASSIGN: // +=
                if def == nil {
                        bits = EvalBits(-1)
                } else if def.Origin() == types.ImmediateDef {
                        bits = KeepClosures
                } else { // InvalidDef, DefaultDef, etc.
                        def.SetOrigin(types.DefaultDef)
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
                        def.SetOrigin(types.DefaultDef)
                }
                bits = KeepClosures|KeepDelegates
        case token.SCO_ASSIGN, token.DCO_ASSIGN: // :=, ::=
                if def != nil {
                        def.SetOrigin(types.ImmediateDef)
                }
                bits = KeepClosures
        default:
                p.error(pos, "Unsuported assign `%v'", tok)
                def, bits = nil, EvalBits(-1)
        }

        if bits != EvalBits(-1) && def != nil {
                if v, e := p.runtime.Eval(value, bits); e == nil {
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
                                def.Value = types.UniversalNone
                        }
                        value = &ast.EvaluatedExpr{ value, v }
                } else {
                        p.error(value.Pos(), e)
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
                case *types.Bareword:
                        if sym := p.runtime.Resolve(t.Value, anywhere); sym == nil {
                                p.error(elem.Pos(), "undefined builtin %v", t.Value)
                        } else {
                                elem.Data = sym
                        }
                }
        }
        
        x = p.parseExpr(true) // Do left-hand-side parsing if in use rule
        if v, e := p.runtime.Eval(x, KeepClosures|KeepDelegates); e != nil {
                p.error(x.Pos(), "%v (%T)", e, x)
        } else if v != nil {
                x = &ast.EvaluatedExpr{ x, v }
        } else {
                p.error(x.Pos(), "Recipe `%T' eval to nil.", x)
        }
        return
}

func (p *parser) parseRecipeExpr(dialect string) ast.Expr {
	if p.trace {
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
                        if v, e := p.runtime.Eval(x, KeepClosures|KeepDelegates); e != nil {
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
                                        v, e := p.runtime.Eval(kv.Key, StringValue)
                                        if e == nil {
                                                var name = v.Strval()
                                                if sym, alt := p.runtime.Symbol(name, types.DefType); alt != nil {
                                                        p.error(p.pos, "Name `%s' already taken (%T).", name, alt)
                                                } else if sym == nil {
                                                        // TODO: errors
                                                } else if v, e = p.runtime.Eval(kv.Value, KeepClosures|KeepDelegates); e == nil {
                                                        sym.(*types.Def).Assign(v)
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
                                                if v, e := p.runtime.Eval(elem, StringValue); e == nil {
                                                        var name = v.Strval()
                                                        if sym, alt := p.runtime.Symbol(name, types.DefType); alt != nil {
                                                                p.error(p.pos, "Name `%s' already taken, not parameter (%T).", name, alt)
                                                        } else if sym == nil {
                                                                // TODO: errors
                                                        } else {
                                                                //sym.(*types.Def).Assign(values.String("xxxxxxxxxx"))
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
                                v, e := p.runtime.Eval(n, StringValue)
                                if e != nil {
                                        p.error(n.Pos(), "%v (%v)", e, n)
                                        goto next
                                }
                                if name, pos = v.Strval(), x.Pos(); name == "" {
                                        p.error(n.Pos(), "empty name (%v)", n)
                                        goto next
                                }
                                goto checkName
                        default:
                                p.error(n.Pos(), "unsupported dialect or modifier (%T)", t.Elems[0])
                                goto next
                        }
                default:
                        p.error(x.Pos(), "unsupported modifier")
                        goto next
                }
                goto addModifier

                checkName: if types.IsDialect(name) {
                        if dialect == "" {
                                dialect = name
                        } else {
                                p.error(pos, "multi-dialect unsupported, already defined '%s'", dialect)
                                goto next
                        }
                } else if types.IsModifier(name) {
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

func (p *parser) closeScope(scope ast.Scope) {
        if err := p.runtime.CloseScope(scope); err != nil {
                p.error(p.pos, "close scope (%v)", err)
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
	if p.trace {
		defer un(trace(p, "Rule"))
	}

        var (
                doc = p.leadComment
                pos = p.expect(tok)
                entries []*types.RuleEntry
                modifier *ast.ModifierExpr
                program *ast.ProgramExpr
                depends []ast.Expr
                recipes []ast.Expr
                scopeComment string
                dialect string
                params []string
                isUseRule bool
        )

        for i, target := range targets {
                v, e := p.runtime.Eval(target, StringValue)
                if e != nil {
                        p.error(target.Pos(), e)
                        continue
                } else if v == nil {
                        p.error(target.Pos(), "Target `%T' is nil", target)
                        continue
                }

                var name = v.Strval()
                if name == "" {
                        p.error(target.Pos(), "empty name (%v)", target); continue
                } else if name == "use" {
                        if i == 0 && len(targets) == 1 {
                                isUseRule = true
                        } else {
                                p.error(target.Pos(), "mixed 'use' with normal targets")
                        }
                }

                var tarent *types.RuleEntry
                if sym, alt := p.runtime.Symbol(name, types.RuleEntryType); alt != nil {
                        if entry, ok := alt.(*types.RuleEntry); entry == nil {
                                p.error(target.Pos(), "Name `%s' already taken, not rule entry (%T).", name, alt)
                        /*} else if entry.Programs() != nil {
                                p.error(target.Pos(), "Rule `%s' already defined (%v).", name, entry.Class())*/
                        } else if ok {
                                //fmt.Printf("entry: %v already defined\n", entry)
                                tarent = entry
                        } else {
                                p.error(target.Pos(), "Invalid rule `%s'.", name)
                        }
                } else if sym != nil {
                        tarent = sym.(*types.RuleEntry)
                }
                if tarent == nil {
                        p.error(target.Pos(), "invalid name '%s'", name); continue
                }

                if !tarent.Position.IsValid() {
                        tarent.Position = p.file.Position(target.Pos())
                }

                // Guessing target entry class, e.g. general, file, etc.
                switch t := target.(type) {
                case *ast.Barefile:
                        if file, _ := t.File.(*types.File); file != nil {
                                tarent.SetExplicitFile(file)
                        } else {
                                p.error(target.Pos(), "unknown file (%v) (%v)", v, t)
                        }
                case *ast.PathExpr:
                        if path, _ := v.(*types.Path); path != nil {
                                tarent.SetExplicitPath(path)
                        } else {
                                p.error(target.Pos(), "unknown path (%v) (%v)", v, t)
                        }
                case *ast.Bareword, *ast.Barecomp:
                        if isUseRule {
                                tarent.SetClass(types.UseRuleEntry)
                        } else {
                                if file := p.runtime.File(name); file != nil {
                                        tarent.SetExplicitFile(file)
                                }
                        }
                }
                entries = append(entries, tarent)

                if scopeComment != "" {
                        scopeComment += " "
                }
                scopeComment += name
        }

        scope := p.runtime.OpenScope(fmt.Sprintf("rule %s", scopeComment))
        for _, s := range automatics {
                if sym, alt := p.runtime.Symbol(s, types.DefType); alt != nil {
                        p.error(p.pos, "Name `%s' already taken, not automatic (%T).", s, alt)
                } else if sym == nil {
                        // TODO: errors
                }
        }
        for i := 1; i < 10; i += 1 {
                if sym, alt := p.runtime.Symbol(strconv.Itoa(i), types.DefType); alt != nil {
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

        // Parsing depends after automatics and parameters are defined, so that
        // the depend list can refer to automatics and parameters.
        if p.tok != token.LINEND {
                depends = p.parseRhsList(true)
                dependsLoop: for i, depend := range depends {
                        //fmt.Printf("depend: %T %v\n", depend, depend)

                        depval, err := p.runtime.Eval(depend, KeepClosures|KeepDelegates|DependValue)
                        if err != nil {
                                p.error(depend.Pos(), err)
                                continue
                        } else if depval == nil {
                                p.error(depend.Pos(), "Invalid depend `%T'.", depend)
                                continue
                        } else {
                                // Detecting self dependency.
                                for _, entry := range entries {
                                        if entry == depval {
                                                //fmt.Printf("depend: %p %v (%v)\n", entry, depval, entry.Project().Name())
                                                p.error(depend.Pos(), "Depends on itself `%v' (%T).", entry, depend)
                                        }
                                }
                        }

                        //fmt.Printf("depend: %T -> %T %v\n", depend, depval, depval)

                        depends[i] = &ast.EvaluatedExpr{ depend, depval }
                        continue dependsLoop
                }
        }
        if p.tok != token.EOF {
                p.expectLinend()
        }

        var useContainScope ast.Scope
        if isUseRule {
                //useContainScope = p.runtime.OpenScope(p.pos, "use")
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

        if _, err := p.runtime.Rule(clause); err != nil {
                p.error(pos, err)
        }
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

        if p.trace {
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
	if p.trace {
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
                doc = p.leadComment
                pos = p.pos
        )

        scope := p.runtime.OpenScope(fmt.Sprintf("file %s", filename))
        if scope != nil {
                defer p.closeScope(scope)
                var (
                        sym RuntimeObj
                        wd = p.runtime.Getwd()
                        abs = filepath.Dir(filename)
                        rel , _ = filepath.Rel(wd, abs)
                )
                //if strings.HasSuffix(rel, abs) {
                //        rel = abs
                //}

                //fmt.Printf("filename=%v\n", filename)
                //fmt.Printf("wd=%v\n", wd)
                //fmt.Printf("abs=%v\n", abs)
                //fmt.Printf("rel=%v\n", rel)

                sym, _ = p.runtime.Symbol("/", types.DefType)
                sym.(*types.Def).Assign(values.String(abs))

                sym, _ = p.runtime.Symbol(".", types.DefType)
                sym.(*types.Def).Assign(values.String(rel))

                //relParent = filepath.Dir(rel)
                //if rel == "." && relParent == "." {
                //        relParent = ".."
                //}
                //fmt.Printf("%p: %v\n", sym, p.runtime.Resolve("/", anywhere))
                //fmt.Printf("%p: %v\n", sym, p.runtime.Resolve(".", anywhere))
        } else {
                p.error(p.pos, "open scope")
        }

        if keyword = p.tok; keyword == token.PROJECT || keyword == token.MODULE {
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

                var params types.Value
                if p.tok == token.LPAREN {
                        value, err := p.runtime.Eval(p.parseGroupExpr(false), StringValue)
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
                        if err := p.runtime.DeclareProject(ident, params); err != nil {
                                p.error(ident.Pos(), err)
                        } else {
                                defer p.runtime.CloseCurrentProject(ident)
                        }
                }
        } else if p.mode&Flat == 0 {
                p.errorExpected(pos, "package keyword")
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
