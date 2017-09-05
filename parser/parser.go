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
        composingPERIOD
        composingDOTDOT
        composingPCON
        composingPERC
        
        // Bits to disable parsing ArgumentedExpr 
        composingNoArg = composingPERIOD | composingDOTDOT | composingPCON | composingPERC
        composingNoKeyword = composingPERIOD | composingPCON | composingPERC
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
        inUseRule bool // parsing use recipe

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

	p.mode = mode
	p.trace = mode&Trace != 0 // for convenience (p.trace is used frequently)

	p.next()
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
	if p.lit[1] == '*' {
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
		case token.IMPORT, token.INCLUDE, token.FILES, token.INSTANCE, token.USE, token.EXPORT, token.EVAL:
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
	case *ast.Globfile:
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

func (p *parser) parseBareword() (x ast.Expr) {
	pos, value, isFileName := p.pos, "", false
        switch p.tok {
	case token.BAREWORD:
		value = p.lit
		p.next()
                if end := token.Pos(int(pos) + len(value)); end == p.pos {
                        switch p.tok {
                        case token.PERIOD, token.DOTDOT, token.PCON:
                                //!< parseSelect will check IsFileName
                                //v := p.runtime.IsFileName(value)
                                //fmt.Printf("bareword: %v %v (%v) (%v)\n", value, p.lit, p.tok, v)
                        default:
                                isFileName = p.runtime.IsFileName(value)
                        }
                }
        case token.AT:
		value = p.tok.String()
		p.next()
        default:
		p.expect(token.BAREWORD) // use expect() error handling
	}
        
        x = &ast.Bareword{ ValuePos: pos, Value:value }
        
        if isFileName {
                ext := filepath.Ext(value)
                if len(ext) > 0 {
                        ext = ext[1:] // drop the '.'
                        pos = token.Pos(int(pos) + len(value) - len(ext))
                }
                x = &ast.Barefile{
                        Name: x,
                        ExtPos: pos,
                        Ext: ext,
                }
        }
        return x
}

func (p *parser) parseSelect(lhs bool, x ast.Expr) (res ast.Expr) {
	if p.trace {
		defer un(trace(p, "Selector"))
	}

        // Parse rhs of '.'...
        s := p.checkExpr(p.parseExpr(lhs))
        switch t := s.(type) {
        case *ast.Bareword:
                switch l := x.(type) {
                case *ast.Bareword:
                        if p.runtime.IsFileName(l.Value+"."+t.Value) {
                                return &ast.Barefile{ x, t.Pos(), t.Value }
                        }
                case *ast.GlobExpr:
                        if p.runtime.IsFileName("a."+t.Value) {
                                return &ast.Globfile{ l, t.Pos(), t.Value }
                        }
                }
        }

        //fmt.Printf("select: %T.%T (%v %v)\n", x, s, x, s)

        // Deal with lhs of '.', convert x into an Ident or Barefile
        var (
                operand types.Value
                value types.Value
                operandName string
                valueName string
                where = anywhere
        )
        switch t := x.(type) {
        case *ast.Bareword: operandName = t.Value
        case *ast.EvaluatedExpr:
                if t.Data != nil {
                        if operand = t.Data.(types.Value); operand != nil {
                                goto ComputeValueName
                        }
                }
                // Error...
                p.error(t.Pos(), "Evaluated select (left) operand `%T' invalid (%v).", t.Expr, t.Data)
                goto DoneSelect
        default:
                if v, e := p.runtime.Eval(x, disclosure); e == nil && v != nil {
                        operandName = v.Strval()
                } else if v == nil {
                        p.error(t.Pos(), "Select operand `%T' eval to nil.", t)
                } else {
                        p.error(t.Pos(), e)
                        p.error(t.Pos(), "Invalid select operand (%T).", t)
                }
        }

        if operandName == "@" {
                // If resolving @ in a rule (program) scope selection context,
                // e.g. '$(@.FOO)', Resolve have to ensure @ is pointing to the global
                // @ package.
                where = global
        }
        if sym := p.runtime.Resolve(operandName, where); sym != nil {
                operand = sym.(types.Value)
        }
        
        if operand == nil {
                p.error(x.Pos(), "Undefined select operand `%s' (%T).", operandName, x)
                goto DoneSelect
        }

        ComputeValueName: switch t := s.(type) {
        case *ast.Bareword: valueName = t.Value
        case *ast.GlobExpr: valueName = t.Tok.String()
        case *ast.EvaluatedExpr:
                if t.Data != nil {
                        if v, _ := t.Data.(types.Value); v != nil {
                                valueName = v.Strval()
                        }
                }
                if valueName == "" {
                        p.error(t.Pos(), "Evaluated select (right) operand `%T' invalid (%v).", t.Expr, t.Data)
                        goto DoneSelect
                }
        default:
                if v, e := p.runtime.Eval(s, disclosure); e == nil && v != nil {
                        valueName = v.Strval()
                } else if v == nil {
                        p.error(t.Pos(), "Select operand (%T) evals to nil.", t)
                        goto DoneSelect
                } else {
                        p.error(t.Pos(), e)
                        p.error(t.Pos(), "Invalid select operand `%T'.", t)
                }
        }
        //fmt.Printf("select: %v.%v (%T %T)\n", operandName, valueName, x, s)

        switch o := operand.(type) {
        case types.Object:
                var err error
                if value, err = o.Get(valueName); err != nil {
                        p.error(s.Pos(), err)
                        p.error(x.Pos(), "Selection `%s.%s' failed.", operandName, valueName)
                } else if value == nil {
                        p.error(s.Pos(), "No such property `%s' in `%s'.", valueName, operandName)
                } else if pn, _ := o.(*types.ProjectName); pn != nil {
                        // Detect diverged scope ().
                        switch v := value.(type) {
                        case *types.RuleEntry:
                                if false && pn.Project() != v.Project() {
                                        p.error(s.Pos(), "Name diverged `%v' (%v != %v).", valueName, pn.Project().Name(), v.Project().Name())
                                }
                        case *types.Def:
                        }
                }
        default:
                goto DoneSelect
        }

        DoneSelect: if value == nil {
                p.error(s.Pos(), "No such property `%v' (%T) in `%s'.", valueName, s, operandName)
        }

        //fmt.Printf("select: %T.%T -> %s.%s\n", x, s, operandName, valueName)
        //fmt.Printf("select: %T %v . %T %v\n", operand, operand, value, value)
        //fmt.Printf("select: %v.%v (%v %v)\n", operandName, valueName, operand, value)
        //fmt.Printf("select: %v.%v (%v %v)\n", operandName, valueName, operand, value)

        res = &ast.EvaluatedExpr{ s, value }
        if p.tok == token.PERIOD {
                p.next() // Drop '.' before continuing selecting.
                // Continue the selection recursivly
                res = p.parseSelect(lhs, res)
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
        if p.tok == token.RPAREN || p.tok == token.LBRACK /*|| p.tok == token.COLON_RBK*/ {
                return true
        }
        if p.tok.IsRuleDelim() {
                return true
        }
        return p.tok == token.EOF || p.tok == token.LINEND || 
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

func (p *parser) parseLhsList() []ast.Expr {
        // Line comment of previous Clause will break the parsing loop,
        // so we clear the previous line comment
        p.lineComment = nil
        
	old := p.inRhs
	p.inRhs = false
	list := p.parseExprList(true)
	/* switch p.tok {
	case token.ASSIGN:
		// lhs of a short variable declaration
		// but doesn't enter scope until later:
		// caller must call p.shortVarDecl(p.makeIdentList(list))
		// at appropriate time.
	case token.COLON:
		// lhs of a label declaration or a communication clause of a select
		// statement (parseLhsList is not called when parsing the case clause
		// of a switch statement):
		// - labels are declared by the caller of parseLhsList
		// - for communication clauses, if there is a stand-alone identifier
		//   followed by a colon, we have a syntax error; there is no need
		//   to resolve the identifier in that case
	default:
		// identifiers must be declared elsewhere
		// for _, x := range list {
		//	p.resolve(x)
		// }
	} */
	p.inRhs = old
	return list
}

func (p *parser) parseRhsList(sel bool) []ast.Expr {
	oldRhs := p.inRhs
	p.inRhs = true
	list := p.parseExprList(false)
	p.inRhs = oldRhs
	return list
}

// ----------------------------------------------------------------------------
// Expressions

func (p *parser) parseGroupExpr() ast.Expr {
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

func (p *parser) parseExpr0(lhs bool) ast.Expr {
        switch p.tok {
        case token.BAREWORD:
                return p.parseBareword()
                
        case token.BIN, token.OCT, token.INT, token.HEX, token.FLOAT,
             token.DATETIME, token.DATE, token.TIME, 
             token.URI, token.STRING, token.ESCAPE:
                pos, tok, lit := p.pos, p.tok, p.lit
                end := int(pos) + len(lit)
                switch tok {
                case token.STRING: end += 2
                }
                p.next()
                return &ast.BasicLit{
                        ValuePos: pos,
                        Kind: tok,
                        Value: lit,
                        EndPos: token.Pos(end),
                }
                
        case token.COMPOUND:
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
                
        case token.AT:
                pos := p.pos; p.next()
                return &ast.Bareword{ 
                        ValuePos: pos, 
                        Value: "@",
                }
                
        case token.STAR:
                pos, tok := p.pos, p.tok
                p.next()
                return &ast.GlobExpr{ TokPos:pos, Tok:tok }

        case token.DOLLAR, token.AND: // delegate, closure
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
                case token.LPAREN:
                        lpos, tokLp = p.pos, p.tok
                        p.next()
                        name = p.checkExpr(p.parseExpr(false))
                        if p.tok != token.RPAREN {
                                rest = append(rest, p.parseListExpr(false))
                                for p.tok == token.COMMA {
                                        p.next()
                                        rest = append(rest, p.parseListExpr(false))
                                }
                        }
                        rpos = p.expect(token.RPAREN)
                default:
                        // Only support $(...), disable $name.
                        if false {
                                // Parse name without composing expressions
                                name = p.checkExpr(p.parseExpr0(false))
                        } else {
                                p.error(p.pos, "Expecting `%v'.", token.LPAREN)
                                return &ast.BadExpr{ From:p.pos, To:p.pos }
                        }
                }

                var resolved RuntimeObj
                if a, _ := name.(*ast.EvaluatedExpr); a != nil {
                        if a.Data == nil {
                                p.error(name.Pos(), "Evaluated data is nil `%T'.", a.Expr)
                        } else if resolved = a.Data.(RuntimeObj); resolved == nil {
                                p.error(name.Pos(), "Unresolved reference `%T'.", a.Expr)
                        }
                } else if v, e := p.runtime.Eval(name, disclosure); e != nil {
                        p.error(pos, e)
                } else if v == nil {
                        p.error(pos, "Name `%T' eval to nil", name)
                } else {
                        a = &ast.EvaluatedExpr{ name, v }
                        if resolved = p.runtime.Resolve(v.Strval(), anywhere); resolved == nil {
                                p.error(name.Pos(), "Undefined reference `%v' (%T).", v.Strval(), name)
                        } else {
                                name = a
                        }
                }

                cd := ast.ClosureDelegate{
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

        case token.LPAREN:
                return p.parseGroupExpr()

        case token.PCON:
                pos, tok := p.pos, p.tok; p.next()
                //for p.tok == token.PCON { p.next() }
                return &ast.PathSegExpr{ pos, tok }
        case token.PERIOD, token.DOTDOT:
                pos, tok := p.pos, p.tok; p.next()
                if p.tok == token.PCON {
                        return &ast.PathSegExpr{ pos, tok }
                } else if tok == token.PERIOD {
                        // TODO: select from the current context
                        return &ast.Bareword{ pos, "." }
                } else {
                        // TODO: select from the parent context
                        return &ast.Bareword{ pos, ".." }
                }
                
        case token.PERC:
                var (
                        y ast.Expr
                        pos = p.pos
                )
                if p.next(); pos+1 == p.pos { // joint, e.g. '%.o', but skip '% .o'
                        y = p.checkExpr(p.parseExpr(false))
                        //fmt.Printf("PERC: %T %v %v(%v)\n", y, y, p.tok, p.lit)
                }
                return &ast.PercExpr{
                        X: nil,
                        OpPos: pos,
                        Y: y,
                }
                
        case token.MINUS:
                pos := p.pos
                p.next()
                x := p.checkExpr(p.parseExpr(false))
                return &ast.FlagExpr{
                        DashPos: pos,
                        Name: x,
                }

        /*case token.PLUS:
                tok, pos := p.tok, p.pos
                p.next()
                x := p.checkExpr(p.parseExpr(false))
                return &ast.UnaryExpr{
                        OpPos: pos,
                        Op: tok,
                        X: x,
                }*/

        case token.PROJECT, token.MODULE, token.USE, token.EXPORT, token.INCLUDE, 
             token.IMPORT, token.INSTANCE, token.FILES:
                if p.inRhs || p.bits&composingNoKeyword != 0 {
                        pos, lit := p.pos, p.lit
                        p.next()
                        // convert keyword into Bareword
                        return &ast.Bareword{
                                ValuePos: pos,
                                Value: lit,
                        }
                }
                fallthrough

        default:
                if p.tok.IsClosure() || p.tok.IsDelegate() {
                        pos, tok, s := p.pos, p.tok, p.tok.String()[1:]
                        p.next()

                        var where = anywhere
                        //if s == "/" || s == "." {
                        //        where = local
                        //}
                        resolved := p.runtime.Resolve(s, where)
                        if resolved == nil {
                                p.error(pos, "Undefined reference `%v' (%v).", s, tok)
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
                } else if p.tok == token.USE {
                        // Treat `use' here as a bareword.
                        tok, pos := p.tok, p.pos
                        p.next() // Drop `use'
                        return &ast.Bareword{ pos, tok.String() }
                }

                pos := p.pos
                //p.warn(pos, "weird token '%v'\n", p.tok)
                p.errorExpected(pos, "clause or expression")
                p.next() // go to next token
                return &ast.BadExpr{ From:pos, To:p.pos }
        }
}

func (p *parser) parseComposing(x ast.Expr, lhs bool) ast.Expr {
        //fmt.Printf("composing:%v: %T %v %v %v\t%v %v\n", (x.End() == p.pos), x, x, x.Pos(), x.End(), p.pos, p.tok)
        var joint = x.End() == p.pos && p.lineComment == nil
        switch p.tok {
        case token.COMPOSED, token.RPAREN, token.RBRACK, token.COMMA, token.COLON, token.LINEND:
        case token.ILLEGAL:
                p.error(p.pos, "illegal token")
        case token.ARROW, token.ASSIGN:
                if !lhs {
                        pos, tok := p.pos, p.tok
                        p.next()
                        x = &ast.KeyValueExpr{
                                Key:   x,
                                Tok:   tok,
                                Equal: pos,
                                Value: p.parseExpr(false),
                        }
                }
        case token.LPAREN:
                // ArgumentedExpr: foo(xxx)
                // Special: foo.o(xxx) - parse select, then argumented
                if joint && p.bits&composingNoArg == 0 {
                        y := p.parseGroupExpr().(*ast.GroupExpr)
                        x = &ast.ArgumentedExpr{ x, y.Elems, y.End() }
                }
        case token.PERIOD:
                // If it's selecting, don't enter parseSelect again.
                // The parseSelect procedure will check this '.'
                if joint && p.bits&composingPERIOD == 0 {
                        p.bits |= composingPERIOD
                        p.next() // Drop the '.' token.
                        x = p.parseSelect(lhs, p.checkExpr(x))
                        //fmt.Printf("parseSelect: select: %T %v (%v %v)\n", x, x, x.End(), p.pos)
                        if p.tok == token.LPAREN && x.End() == p.pos {
                                //fmt.Printf("parseSelect: compose: %T %v\n", x, x)
                                //x = p.parseComposing(x, lhs)
                                y := p.parseGroupExpr().(*ast.GroupExpr)
                                x = &ast.ArgumentedExpr{ x, y.Elems, y.End() }
                                //fmt.Printf("parseSelect: composed: %T %v\n", x, x)
                        }
                        p.bits &= ^composingPERIOD
                }
        case token.PCON:
                if joint && p.bits&composingPCON == 0 {
                        p.bits |= composingPCON
                        pat := &ast.PathExpr{ 
                                PosBeg: p.pos, 
                                Segments: []ast.Expr{x}, 
                                PosEnd: p.pos,
                        }
                        // Drop continual '/' tokens.
                        ConcatPath: for p.tok == token.PCON { p.next() }
                        y := p.checkExpr(p.parseExpr(lhs))
                        pat.Segments = append(pat.Segments, y)
                        pat.PosEnd = y.End()
                        if p.tok == token.PCON {
                                goto ConcatPath
                        }
                        x = pat
                        p.bits &= ^composingPCON
                }
        case token.PERC:
                if joint && p.bits&composingPERC == 0 {
                        p.bits |= composingPERC
                        x = &ast.PercExpr{
                                X: x,
                                OpPos: p.pos,
                                Y: p.checkExpr(p.parseExpr(lhs)),
                        }
                        p.bits &= ^composingPERC
                }
        default:
                if joint && p.bits&composing == 0 {
                        var y ast.Expr
                        p.bits |= composing
                Compose:
                        y = p.checkExpr(p.parseExpr(lhs))
                        //fmt.Printf("compose: %T %T\n", x, y)
                        x = &ast.Barecomp{ []ast.Expr{ x, y } }
                        if x.End() == p.pos && p.lineComment == nil {
                                switch p.tok {
                                case token.COMPOSED, token.RPAREN, token.RBRACK, token.COMMA, token.COLON, token.LINEND:
                                case /*token.LPAREN, */token.PERIOD, token.DOTDOT, token.PCON, token.PERC:
                                case token.ARROW, token.ASSIGN:
                                case token.ILLEGAL:
                                default:
                                        if p.tok.IsAssign() || p.tok.IsRuleDelim() {
                                                break
                                        }
                                        goto Compose
                                }
                        }
                        p.bits &= ^composing
                        return x
                }
        }
        return x
}

func (p *parser) parseExpr(lhs bool) (x ast.Expr) {
	if p.trace {
		defer un(trace(p, "Expression"))
	}
        x = p.parseComposing(p.parseExpr0(lhs), lhs)
        return
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
        if err := p.runtime.ClauseImport(spec); err != nil {
                p.error(spec.Pos(), err)
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
        files := make(map[string][]string)
        spec := &ast.FilesSpec{ p.parseDirectiveSpec() }
        for _, prop := range spec.Props {
                ee, _ := prop.(*ast.EvaluatedExpr)
                if ee == nil || ee.Data == nil {
                        p.error(prop.Pos(), "bad file spec (%T)", prop)
                        continue
                }
                switch v := ee.Data.(type) {
                case *types.Pair:
                        switch s := v.Key.Strval(); vv := v.Value.(type) {
                        case *types.Group:
                                for _, elem := range vv.Elems {
                                        files[s] = append(files[s], elem.Strval())
                                }
                        default:
                                files[s] = append(files[s], vv.Strval())
                        }
                case types.Value:
                        s := v.Strval()
                        files[s] = append(files[s], ".")
                default:
                        p.error(prop.Pos(), "bad file spec (%T)", prop)
                }
        }
        p.runtime.Files(files)
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
        if v, e := p.runtime.Eval(x, disclosure); e == nil {
                x = &ast.EvaluatedExpr{ x, v }
        } else {
                p.error(x.Pos(), "immediate (%s)", e)
        }

        // Append the prop `x'.
        props = append(props, x)

        // Parse the parameters.
        for p.tok != token.EOF {
                if p.tok == token.COMMA || p.tok == token.LINEND || p.tok == token.RPAREN {
                        break
                }
                if p.lineComment != nil {
                        // found a line comment at the end
                        comment = p.lineComment
                        break
                }
                x = p.parseExpr(false)
                if v, e := p.runtime.Eval(x, delegation); e == nil {
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
		defer un(trace(p, "Clause("+p.tok.String()+")"))
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
        if v, e := p.runtime.Eval(ident, disclosure); e == nil {
                name := v.Strval()

                // If doing '+=', the assignment will concate the value of the
                // symbol from the other scope with new one. But when working in
                // 'use' rule, this concation will not applied.
                if tok == token.ADD_ASSIGN && !p.inUseRule {
                        if sym := p.runtime.Resolve(name, anywhere); sym != nil {
                                prev, _ = sym.(*types.Def)
                        }
                }
                
                // Always work in the current runtime scope, so it won't affect
                // any base symbols. If p.inUseRule is set, it will be defining
                // a Definer in the 'use' rule scope. 
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
                                def.Assign(types.Delegate(prev))
                        }
                } else if s != nil {
                        p.error(ident.Pos(), "Name `%s' already taken, not def (%T).", name, s)
                } else {
                        p.error(ident.Pos(), "Failed defining `%s' (%v).", name, v)
                }
        } else {
                p.error(ident.Pos(), e)
                p.error(ident.Pos(), "error declosing name %T", ident)
        }
        
        var bits = delegation // assumes types.DefaultDef
        switch tok {
        case token.ADD_ASSIGN: // +=
                if def == nil {
                        bits = EvalBits(-1)
                } else if def.Origin() == types.ImmediateDef {
                        bits = immediate
                } else { // InvalidDef, DefaultDef, etc.
                        def.SetOrigin(types.DefaultDef)
                        bits = delegation
                }
        case token.QUE_ASSIGN: // ?=
                if alt {
                        // bypass any eval if already defined
                        bits = EvalBits(-1); break
                }
                bits = delegation
        case token.ASSIGN, token.EXC_ASSIGN: // =, !=
                if def != nil {
                        def.SetOrigin(types.DefaultDef)
                }
                bits = delegation
                // TODO: deal with p.inUseRule
        case token.SCO_ASSIGN, token.DCO_ASSIGN: // :=, ::=
                if def != nil {
                        def.SetOrigin(types.ImmediateDef)
                }
                bits = immediate
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

func (p *parser) parseBuiltinRecipeExpr(elems []ast.Expr) (x ast.Expr) {
        // Do left-hand-side parsing if in use rule
        switch x = p.parseExpr(p.inUseRule); {
        case p.tok.IsAssign():
                d := p.parseDefineClause(p.tok, x).(*ast.DefineClause)
                x = &ast.RecipeDefineClause{ d }

        case p.tok.IsRuleDelim():
                elems = append(elems, x)
                d := p.parseRuleClause(p.tok, elems).(*ast.RuleClause)
                x = &ast.RecipeRuleClause{ d }

        default:
                if v, e := p.runtime.Eval(x, delegation/*disclosure*/); e != nil {
                        p.error(x.Pos(), "%v (%T)", e, x)
                } else if v != nil {
                        if len(elems) == 0 {
                                if name := v.Strval(); name == "" {
                                        p.error(x.Pos(), "Empty recipe command `%v' (%T).", v, x)
                                } else if sym := p.runtime.Resolve(name, anywhere); sym == nil {
                                        p.error(x.Pos(), "Undefined recipe command `%v' (%v).", name, v)
                                } else {
                                        v = sym.(types.Value)
                                }
                        }
                        x = &ast.EvaluatedExpr{ x, v }
                } else {
                        p.error(x.Pos(), "Recipe `%T' eval to nil.", x)
                }
        }
        return
}

func (p *parser) parseRecipeExpr(dialect string, isUseRule bool) ast.Expr {
	if p.trace {
		defer un(trace(p, "Recipe"))
	}
        
        var (
                comment *ast.CommentGroup
                elems []ast.Expr
                doc = p.leadComment
                pos = p.pos
        )

        switch dialect {
        case "":
                p.scanner.LeaveCompoundLineContext()
                p.next() // skip RECIPE and parse in list mode
                p.inUseRule = isUseRule
                for p.tok != token.LINEND && p.tok != token.EOF {
                        elems = append(elems, p.parseBuiltinRecipeExpr(elems[:]))
                        if p.lineComment != nil {
                                comment = p.lineComment
                                break
                        }
                }
                p.inUseRule = false

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
                case *ast.Bareword:
                        name, pos = t.Value, t.Pos()
                        goto checkName
                case *ast.GroupExpr:
                        switch n := t.Elems[0].(type) {
                        case *ast.Bareword:
                                name, pos = n.Value, n.Pos()
                                goto checkName
                        case *ast.GroupExpr:
                                for _, elem := range n.Elems {
                                        //fmt.Printf("param: %T\n", elem)
                                        switch elem.(type) {
                                        case *ast.Bareword, *ast.Barecomp:
                                                if v, e := p.runtime.Eval(elem, disclosure); e == nil {
                                                        params = append(params, v.Strval())
                                                } else {
                                                        p.error(elem.Pos(), "bad parameter (%T, %v)", elem, e)
                                                }
                                        default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
                                                p.error(elem.Pos(), "bad parameter form (%T)", elem)
                                        }
                                }
                                goto next
                        case *ast.DelegateExpr, *ast.ClosureExpr, *ast.Barecomp, *ast.BasicLit:
                                v, e := p.runtime.Eval(n, disclosure)
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
                                p.error(n.Pos(), "unsupported dialect or modifier")
                                goto next
                        }
                }
                goto addModifier

                checkName: if p.runtime.IsDialect(name) {
                        if dialect == "" {
                                dialect = name
                        } else {
                                p.error(pos, "multi-dialect unsupported, already defined '%s'", dialect)
                                goto next
                        }
                } else if p.runtime.IsModifier(name) {
                        goto addModifier
                } else {
                        p.error(pos, "no dialect or modifier '%s'", name)
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
        "-",
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
                v, e := p.runtime.Eval(target, disclosure)
                if e != nil {
                        p.error(target.Pos(), e)
                        continue
                } else if v == nil {
                        p.error(target.Pos(), "Target `%T' is nil", target)
                        continue
                }

                var name = v.Strval()
                if name == "" {
                        p.error(target.Pos(), "Empty entry name.")
                        continue
                } else if name == "use" {
                        if i == 0 && len(targets) == 1 {
                                isUseRule = true
                        } else {
                                p.error(target.Pos(), "Mixing use with normal targets.")
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
                        p.error(target.Pos(), "Entry name `%s' invalid.", name)
                        continue
                }
                
                //p.warn(target.Pos(), "%v: %v (%T %v)", tarent, tarent.Class(), target, p.runtime.IsFileName(name))

                // Guessing target entry class, e.g. general, file, etc.
                var class = tarent.Class()
                switch target.(type) {
                case *ast.Barefile, *ast.PathExpr, *ast.PathSegExpr:
                        class = types.FileRuleEntry
                case *ast.Bareword:
                        if p.runtime.IsFileName(name) {
                                class = types.FileRuleEntry
                        }
                }
                tarent.SetClass(class)
                entries = append(entries, tarent)

                if scopeComment != "" {
                        scopeComment += " "
                }
                scopeComment += name
        }

        switch p.tok {
        case token.MINUS: // a '-' modifier is defining a rule as modifier
                p.next()
                // TODO: '-' modifier
        case token.EXC:
                p.next()
                // TODO: '!' modifier
        case token.QUE:
                p.next()
                // TODO: '?' modifier
        case token.LBRACK:
                // Parse modifiers in the program scope.
                dialect, params, modifier = p.parseModifierExpr()
        }

        if p.tok == token.COLON {
                p.next()
        }

        scope := p.runtime.OpenScope(fmt.Sprintf("rule %s", scopeComment))
        for _, s := range automatics {
                if sym, alt := p.runtime.Symbol(s, types.DefType); alt != nil {
                        p.error(p.pos, "Name `%s' already taken, not automatic (%T).", s, alt)
                } else if sym == nil {
                        // TODO: errors
                }
        }
        for _, s := range params {
                if sym, alt := p.runtime.Symbol(s, types.DefType); alt != nil {
                        p.error(p.pos, "Name `%s' already taken, not parameter (%T).", s, alt)
                } else if sym == nil {
                        // TODO: errors
                } else {
                        //sym.(*types.Def).Assign(values.String("xxxxxxxxxx"))
                }
        }
        for i := 1; i < 10; i += 1 {
                if sym, alt := p.runtime.Symbol(strconv.Itoa(i), types.DefType); alt != nil {
                        p.error(p.pos, "Name `%v' already taken, not numberred (%T).", i, alt)
                } else if sym == nil {
                        // TODO: errors
                }
        }

        // Parsing depends after automatics and parameters are defined, so that
        // the depend list can refer to automatics and parameters.
        if p.tok != token.LINEND {
                depends = p.parseRhsList(true)
                dependsLoop: for i, depend := range depends {
                        //fmt.Printf("depend: %T %v\n", depend, depend)

                        depval, err := p.runtime.Eval(depend, ruledepend)
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

                        //if scopeComment == "lib.a" {
                        //        fmt.Printf("parseRuleClause: %s: %T -> %T %v\n", scopeComment, depend, depval, depval)
                        //}

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
                recipes = append(recipes, p.parseRecipeExpr(dialect, isUseRule))
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
                return p.parseGenericClause(p.tok, p.expect(p.tok), p.parseIncludeSpec)
	case token.INSTANCE:
                return p.parseGenericClause(p.tok, p.expect(p.tok), p.parseInstanceSpec)
        case token.FILES:
                return p.parseGenericClause(p.tok, p.expect(p.tok), p.parseFilesSpec)
        case token.EVAL:
                return p.parseGenericClause(p.tok, p.expect(p.tok), p.parseEvalSpec)
	case token.USE:
                pos := p.expect(p.tok)
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
                //defer un(trace(p, "Clause("+p.tok.String()+")"))
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
		defer un(trace(p, "File"))
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
                        x := p.parseBareword()
                        if ident, _ = x.(*ast.Bareword); ident == nil {
                                p.error(p.pos, "invalid package name %T", x)
                        }
                }
                
                if ident.Value == "_" && p.mode&DeclarationErrors != 0 {
                        p.error(p.pos, "invalid package name _")
                }

                var params types.Value
                if p.tok == token.LPAREN {
                        value, err := p.runtime.Eval(p.parseGroupExpr(), disclosure)
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
