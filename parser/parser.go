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
        "unicode"
        "strings"
        "fmt"
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
        isUse   bool // parsing use recipe
        //noSel   bool
        
	// Ordinary identifier scopes
	unresolved []*ast.Bareword   // unresolved identifiers (reference to project symbols)
	imports    []*ast.ImportSpec // list of imports
}

func (p *parser) init(ctx *Context, fset *token.FileSet, filename string, src []byte, mode Mode) {
        p.Context = ctx
        p.file = fset.AddFile(filename, -1, len(src))
        
	var m scanner.Mode
	if mode&ParseComments != 0 {
		m = scanner.ScanComments
	}
	eh := func(pos token.Position, msg string) { p.errors.Add(pos, msg) }
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

func (p *parser) error(pos token.Pos, msg string) {
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

	p.errors.Add(epos, msg)
}

func (p *parser) errorExpected(pos token.Pos, msg string) {
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

// The unresolved object is a sentinel to mark identifiers that have been added
// to the list of unresolved identifiers. The sentinel is only used for verifying
// internal consistency.
var unresolved = new(types.Named)

// If x is an identifier, tryResolve attempts to resolve x by looking up
// the symbol it denotes. If no object is found and collectUnresolved is
// set, x is marked as unresolved and collected in the list of unresolved
// identifiers.
//
/*func (p *parser) tryResolve(x ast.Expr, collectUnresolved bool) {
	// nothing to do if x is not an identifier or the blank identifier
	ident, _ := x.(*ast.Ident)
	if ident == nil {
		return
	}
        
	assert(ident.Sym == nil, "identifier already declared or resolved")
        
	if ident.Value == "_" {
		return
	}
        
	// try to resolve the identifier
        if obj := p.runtime.Resolve(ident.Value); obj != nil {
                //fmt.Printf("resolved: %T (%v)\n", obj, obj)
                ident.Sym = obj.(ast.Symbol)
                return
        } else {
		ident.Sym = unresolved
        }

	// all local scopes are known, so any unresolved identifier
	// must be found either in the file scope, package scope
	// (perhaps in another file), or universe scope --- collect
	// them so that they can be resolved later
	if collectUnresolved {
		p.unresolved = append(p.unresolved, ident)
	}
}

func (p *parser) resolve(x ast.Expr) {
	p.tryResolve(x, true)
}

func (p *parser) identify(x ast.Expr) ast.Expr {
        switch t := x.(type) {
        case *ast.Bareword: 
                ident := &ast.Ident{ *t, nil }
                if p.resolve(ident); ident.Sym == nil || ident.Sym == unresolved {
                        p.warn(t.Pos(), fmt.Sprintf("%s undefined", ident.Value))
                        p.error(t.Pos(), fmt.Sprintf("identifier undefined (%s)", ident.Value))
                }
                //fmt.Printf("identify: %T %v (%p %p)\n", ident.Sym, ident.Sym, ident.Sym, unresolved)
                x = ident
        case *ast.EvaluatedExpr:
                ident := &ast.Ident{ ast.Bareword{ ValuePos:t.Pos() }, nil }
                switch data := t.Data.(type) {
                case types.Caller:
                        ident.Sym = t.Data.(ast.Symbol)
                        if o, _ := data.(types.Object); o != nil {
                                ident.Value = o.Name()
                        }
                default:
                        //fmt.Printf("identify: %T %T %v\n", t.Expr, t.Data, t.Data)
                        ident.Value = data.(types.Value).Strval()
                        if p.resolve(ident); ident.Sym == nil || ident.Sym == unresolved {
                                p.error(t.Pos(), fmt.Sprintf("identifier '%s' undefined", ident.Value))
                        }
                }
                if ident.Value == "" {
                        switch tx := t.Expr.(type) {
                        case *ast.Bareword: ident.Value = tx.Value
                        default:
                                p.error(t.Pos(), fmt.Sprintf("unsupported identifier expression (%T)", t.Expr))
                        }
                }
                //fmt.Printf("identify: %T %v\n", ident.Sym, ident.Sym)
                x = ident
        case *ast.Barefile, *ast.PathExpr:
                // Barefile and PathExpr are a special identifier itself.
        case *ast.Ident: 
                // Ignore silently.
        case *ast.BasicLit:
                if t.Kind == token.INT && len(t.Value) == 1 {
                        ident := &ast.Ident{ ast.Bareword{ t.Pos(), t.Value }, nil }
                        if p.resolve(ident); ident.Sym == nil {
                                p.error(t.Pos(), fmt.Sprintf("identifier '%s' undefined", ident.Value))
                        }
                        x = ident
                } else {
                        p.error(t.Pos(), fmt.Sprintf("bad identifier (%v)", t.Value))
                }
        default: 
                p.error(t.Pos(), fmt.Sprintf("unkown identifier (%T)", t))
        }
        return x
}*/

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
        case *ast.PathExpr:
        case *ast.PercExpr:
        case nil:
                p.warn(p.pos, "nil expression")
		p.errorExpected(p.pos, "nil expression")
		x = &ast.BadExpr{From:token.NoPos, To:token.NoPos}
	default:
		// all other nodes are not proper expressions
                p.warn(x.Pos(), "bad expression (%T)\n", x)
		p.errorExpected(x.Pos(), "bad expression")
		x = &ast.BadExpr{From: x.Pos(), To: p.safePos(x.End())}
	}
	return x
}

// ----------------------------------------------------------------------------
// Barewords & Identifiers

func (p *parser) parseBareword() *ast.Bareword {
	pos, value := p.pos, ""
        switch p.tok {
	case token.BAREWORD:
		value = p.lit
		p.next()
        case token.AT:
		value = p.tok.String()
		p.next()
        default:
		p.expect(token.BAREWORD) // use expect() error handling
	}
	return &ast.Bareword{
                ValuePos: pos, 
                Value: value,
        }
}

func (p *parser) parseSelect(lhs bool, x ast.Expr) (res ast.Expr) {
	if p.trace {
		defer un(trace(p, "Selector"))
	}

        // Parse rhs of '.'...
        s := p.checkExpr(p.parseExpr(lhs))
        switch t := s.(type) {
        case *ast.Bareword:
                if bw, ok := x.(*ast.Bareword); ok && p.runtime.IsFileName(bw.Value+"."+t.Value) {
                        return &ast.Barefile{ x, t.Pos(), t.Value }
                }

        case *ast.Barecomp:
                // Dealing with barecomps like 'foobar.o(arg)', this
                // algorithm converts splitting 'foobar', '.o(arg)' into
                // 'foobar.o' and '(arg)' (suppose '*.o' are files).
                bw1, b1 := x.(*ast.Bareword)
                bw2, b2 := t.Elems[0].(*ast.Bareword)
                if b1 && b2 {
                        if p.runtime.IsFileName(bw1.Value+"."+bw2.Value) {
                                t.Elems[0] = &ast.Barefile{
                                        x, t.Pos(), bw2.Value,
                                }
                                return t
                        }

                        // Reset the select operand
                        s = bw2

                        // Compose again at the end
                        defer func() {
                                t.Elems[0] = res
                                res = t
                        }()
                }
        }

        // Deal with lhs of '.', convert x into an Ident or Barefile
        var (
                operand types.Value
                value types.Value
                operandName string
                valueName string
        )
        switch t := x.(type) {
        case *ast.Bareword: operandName = t.Value
        case *ast.EvaluatedExpr:
                if operand = t.Data.(types.Value); operand != nil {
                        goto ComputeValueName
                } else {
                        p.error(t.Pos(), fmt.Sprintf("nil select operand %T (%v)", t.Data, t.Data))
                        goto DoneSelect
                }
        default:
                if v, e := p.runtime.Eval(x, disclosure); e == nil {
                        operandName = v.Strval()
                } else {
                        p.error(t.Pos(), fmt.Sprintf("invalid select operand %T (%v)", t, e))
                }
        }
        if sym := p.runtime.Resolve(operandName); sym != nil {
                operand = sym.(types.Value)
        } else {
                p.error(x.Pos(), fmt.Sprintf("undefiend select operand '%s'", operandName))
        }

        ComputeValueName: switch t := s.(type) {
        case *ast.Bareword: valueName = t.Value
        case *ast.EvaluatedExpr:
                if v, _ := t.Data.(types.Value); v != nil {
                        valueName = v.Strval()
                } else {
                        p.error(t.Pos(), fmt.Sprintf("nil select operand %T (%v)", t.Data, t.Data))
                }
                goto DoneSelect
        default:
                if v, e := p.runtime.Eval(s, disclosure); e == nil && v != nil {
                        valueName = v.Strval()
                } else if v == nil {
                        goto DoneSelect
                } else {
                        p.error(t.Pos(), fmt.Sprintf("invalid select operand %T (%v)", t, e))
                }
        }

        switch o := operand.(type) {
        case types.Object:
                var err error
                if value, err = o.Get(valueName); err != nil {
                        p.warn(s.Pos(), err.Error())
                } else if value == nil {
                        p.warn(s.Pos(), fmt.Sprintf("no such property (%s)", valueName))
                } else {
                        goto DoneSelect
                }
                p.error(s.Pos(), "select property failed")
        default:
                goto DoneSelect
        }
        
        DoneSelect: if value == nil {
                p.error(s.Pos(), fmt.Sprintf("symbol undefined (in %s)", operandName))
        }

        //fmt.Printf("select: %T . %T -> %s.%s\n", x, s, operandName, valueName)
        //fmt.Printf("select: %T %v . %T %v\n", operand, operand, value, value)
        
        res = &ast.EvaluatedExpr{ value, s }
        if p.tok == token.PERIOD {
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
        if token.COLON <= p.tok && p.tok <= token.QUE {
                return true
        }
        return p.tok == token.EOF || p.tok == token.LINEND || 
                p.tok == token.COMMA || (lhs && 
                (token.ASSIGN <= p.tok && p.tok <= token.DCO_ASSIGN))
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
                pos, lit := p.pos, p.lit
                p.next()
                return &ast.Bareword{
                        ValuePos: pos,
                        Value: lit,
                }
                
        case token.INT, token.FLOAT, token.DATETIME, 
             token.DATE, token.TIME, token.URI, token.STRING,
             token.ESCAPE:
                pos, tok, lit := p.pos, p.tok, p.lit
                p.next()
                return &ast.BasicLit{
                        ValuePos: pos,
                        Kind: tok,
                        Value: lit,
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
                
         /*case token.CALL_A, token.CALL_L, token.CALL_U, token.CALL_S, token.CALL_M,
              token.CALL_1, token.CALL_2, token.CALL_3, token.CALL_4, 
              token.CALL_5, token.CALL_6, token.CALL_7, token.CALL_8, 
              token.CALL_9, token.CALL_R, token.CALL_D, token.CALL_DD:
                pos, tok, s := p.pos, p.tok, p.tok.String()[1:]
                p.next()

                var ident = &ast.Ident{ &ast.Bareword{ p.pos, s }, nil }
                if p.resolve(ident); ident.Sym == nil {
                        p.error(pos, fmt.Sprintf("undefined '%s'", ident.Value))
                }
                return &ast.DelegateExpr{
                        ast.ClosureDelegate{
                                TokPos: pos,
                                Lparen: token.NoPos,
                                Name: ident,
                                Rparen: token.NoPos,
                                Tok: tok,
                        },
                }*/

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
                        /*oldSel := p.noSel
                        p.noSel = false*/
                        name = p.checkExpr(p.parseExpr(false))
                        if p.tok != token.RPAREN {
                                /*oldSel := p.noSel
                                p.noSel = true*/
                                rest = append(rest, p.parseListExpr(false))
                                for p.tok == token.COMMA {
                                        p.next()
                                        rest = append(rest, p.parseListExpr(false))
                                }
                                //p.noSel = oldSel
                        }
                        //p.noSel = oldSel
                        rpos = p.expect(token.RPAREN)
                default:
                        // Parse name without composing expressions
                        name = p.checkExpr(p.parseExpr0(false))
                }

                var resolved RuntimeObj
                if v, e := p.runtime.Eval(name, disclosure); e == nil {
                        name = &ast.EvaluatedExpr{ v, name }
                        resolved := p.runtime.Resolve(v.Strval())
                        if resolved != nil {
                                p.error(name.Pos(), fmt.Sprintf("undefined %v (%v)", v.Strval(), v))
                        }
                } else {
                        p.error(pos, fmt.Sprintf("failed eval name %T (%s)", name, e))
                }


                //fmt.Printf("ClosureDelegate: %T %v -> %T %v\n", name, name, ident, ident)
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

        //case token.PCON:
        case token.PERIOD:
                pos := p.pos
                p.next()
                return &ast.Bareword{ pos, "." }

        case token.PERC:
                pos := p.pos
                p.next()
                y := p.checkExpr(p.parseExpr(false))
                return &ast.PercExpr{
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
                if p.inRhs {
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

                        resolved := p.runtime.Resolve(s)
                        if resolved != nil {
                                p.error(pos, fmt.Sprintf("undefined %v (closure/delegate)", s))
                        }
                        
                        cd := ast.ClosureDelegate{
                                TokPos: pos,
                                Lparen: token.NoPos,
                                Name: &ast.Bareword{ p.pos, s },
                                Resolved: resolved,
                                Rparen: token.NoPos,
                                Tok: tok,
                        }
                        if p.tok.IsDelegate() {
                                return &ast.DelegateExpr{ cd }
                        } else {
                                return &ast.ClosureExpr{ cd }
                        }
                }

                pos := p.pos
                p.warn(pos, "weird token '%v'\n", p.tok)
                p.errorExpected(pos, "clause or expression")
                p.next() // go to next token
                return &ast.BadExpr{ From:pos, To:p.pos }
        }
}

func (p *parser) parseComposing(x ast.Expr, lhs bool) ast.Expr {
        //fmt.Printf("composing:%v: %T %v %v %v\t%v %v\n", (x.End() == p.pos), x, x, x.Pos(), x.End(), p.pos, p.tok)
        if (p.tok == token.ARROW || p.tok == token.ASSIGN) && !lhs {
                pos, tok := p.pos, p.tok
                p.next()
                x = &ast.KeyValueExpr{
                        Key:   x,
                        Tok:   tok,
                        Equal: pos,
                        Value: p.parseExpr0(false),
                }
        } else if p.tok == token.PERIOD {
                p.next()
                x = p.parseSelect(lhs, p.checkExpr(x))
        } else if p.tok == token.PCON {
                var (
                        segments []ast.Expr
                        pos = x.Pos()
                )

                good := func() bool {
                        switch x.(type) {
                        case *ast.Barefile: // good
                        case *ast.Bareword: // good
                        case *ast.Barecomp: // good
                        case *ast.BasicLit: // TODO: validate t.Kind...
                        default:
                                //fmt.Printf("%T %v (%v)\n", x, x, p.tok)
                                p.errorExpected(x.Pos(), fmt.Sprintf("path segment (%T)", x))
                                return false
                        }
                        return true
                }

                if good() {
                        segments = append(segments, x)
                        p.next() // next token after '/'
                        prev := p.pos
                        x = p.checkExpr(p.parseExpr(lhs))
                        if pp, ok := x.(*ast.PathExpr); ok {
                                segments = append(segments, pp.Segments...)
                        } else if x.Pos() == prev && good() {
                                segments = append(segments, x)
                        }
                }
                return &ast.PathExpr{ PosBeg:pos, Segments:segments, PosEnd:p.pos }
        } else if p.tok == token.PERC && x.End() == p.pos {
                return &ast.PercExpr{
                        X: x,
                        OpPos: p.pos,
                        Y: p.checkExpr(p.parseExpr(lhs)),
                }
        } else if p.tok == token.COMPOSED {
                // ignored
        } else if p.tok == token.RPAREN {
                // ignored
        } else if p.tok == token.RBRACK {
                // ignored
        } else if p.tok == token.COMMA {
                // ignored
        } else if p.tok == token.COLON {
                // ignored
        } else if p.tok == token.LINEND {
                // ignored
        } else if p.tok == token.ILLEGAL {
                p.error(p.pos, "illegal token")
        } else if x.End() == p.pos && p.lineComment == nil {
                //fmt.Printf("compose: %T %v %v %v\t%v %v\n", x, x, x.Pos(), x.End(), p.pos, p.tok)
                var (
                        elems = []ast.Expr{ x }
                        y = p.checkExpr(p.parseExpr(lhs))
                )
                if c, ok := y.(*ast.Barecomp); ok {
                        elems = append(elems, c.Elems...)
                } else {
                        elems = append(elems, y)
                }
                x = &ast.Barecomp{ elems }
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
                if _, ok := err.(scanner.Errors); ok {
                        fmt.Printf("%v\n", err)
                        p.error(spec.Pos(), "import error")
                } else {
                        p.warn(spec.Pos(), "%v", err)
                }
        } else {
                p.imports = append(p.imports, spec)
        }
        return spec
}

func (p *parser) parseIncludeSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        spec := &ast.IncludeSpec{ p.parseDirectiveSpec() }
        if err := p.runtime.ClauseInclude(spec); err != nil {
                if _, ok := err.(scanner.Errors); ok {
                        fmt.Printf("%v\n", err)
                        p.error(spec.Pos(), fmt.Sprintf("include error"))
                } else {
                        p.error(spec.Pos(), fmt.Sprintf("%v", err))
                }
        }
        return spec
}

func (p *parser) parseUseSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        spec := &ast.UseSpec{ p.parseDirectiveSpec() }
        if err := p.runtime.ClauseUse(spec); err != nil {
                if _, ok := err.(scanner.Errors); ok { // TODO: impossible
                        fmt.Printf("%v\n", err)
                        p.error(spec.Pos(), fmt.Sprintf("use error"))
                } else {
                        p.error(spec.Pos(), fmt.Sprintf("%v", err))
                }
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
                        p.error(prop.Pos(), fmt.Sprintf("bad file spec (%T)", prop))
                        continue
                }
                switch v := ee.Data.(type) {
                case *types.Pair:
                        switch s := v.K.Strval(); vv := v.V.(type) {
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
                        p.error(prop.Pos(), fmt.Sprintf("bad file spec (%T)", prop))
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
                p.error(ee.Pos(), fmt.Sprintf("invalid eval symbol (%T)", ee.Data))
        } else if spec.Resolved = p.runtime.Resolve(name.Strval()); spec.Resolved == nil {
                p.error(ee.Pos(), fmt.Sprintf("undefined eval symbol %s (%v)", name.Strval(), name))
        } else if err := p.runtime.ClauseEval(spec); err != nil {
                if _, ok := err.(scanner.Errors); ok { // TODO: impossible
                        fmt.Printf("%v\n", err) // Just print out the scanner errors
                        p.error(spec.Pos(), fmt.Sprintf("eval error"))
                } else {
                        p.error(spec.Pos(), fmt.Sprintf("%v", err))
                }
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
                props = append(props, &ast.EvaluatedExpr{ v, x })
        } else {
                p.error(x.Pos(), fmt.Sprintf("immediate (%s)", e))
        }
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
                        props = append(props, &ast.EvaluatedExpr{ v, x })
                } else {
                        p.error(x.Pos(), fmt.Sprintf("immediate (%s)", e))
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

        var value ast.Expr
        
        doc := p.leadComment
        pos := p.expect(tok)

        //oldSel := p.noSel
        //p.noSel = true
        elems := p.parseRhsList(false)
        comment := p.lineComment
        //p.noSel = oldSel

        // Take it from parser, since the line comment is assigned
        // to the DefineClause.
        p.lineComment = nil

        if n := len(elems); n == 1 {
                value = elems[0]
        } else if n > 1 {
                value = &ast.ListExpr{ elems }
        }
        
        if tok == token.SCO_ASSIGN || tok == token.DCO_ASSIGN {
                if v, e := p.runtime.Eval(value, delegation); e == nil {
                        value = &ast.EvaluatedExpr{ v, value }
                } else {
                        p.error(pos, fmt.Sprintf("immediate (%s)", e))
                }
        }

        clause := &ast.DefineClause{ 
                Doc: doc,
                TokPos: pos,
                Tok: tok,
                Name: ident,
                Value: value,
                Comment: comment,
        }

        var name string
        switch n := clause.Name.(type) {
        case *ast.Bareword: name = n.Value
        default:
                p.warn(pos, fmt.Sprintf("%T (%v)", n, n))
                p.error(pos, "unsupported name")
        }
        
        if !p.isUse && name != "" {
                if sym, err := p.runtime.Define(clause); err != nil {
                        if _, ok := err.(scanner.Errors); ok { // TODO: impossible
                                fmt.Printf("%v\n", err)
                                p.error(pos, fmt.Sprintf("define %s error", name))
                        } else {
                                p.warn(pos, fmt.Sprintf("define %s (%v)", name, err))
                                p.error(pos, fmt.Sprintf("%v: %v", name, err))
                        }
                } else if sym != nil {
                        //fmt.Printf("parseDefineClause: %T %v\n", sym, sym)
                } else {
                        // FIXME: ...
                }
        }
        return clause
}

func (p *parser) parseUseRecipe() (x ast.Expr) {
        p.isUse = true
        defer func() {
                p.isUse = false;
        }()

        x = p.parseExpr(true) // left-hand-side parsing

        switch {
        case token.ASSIGN <= p.tok && p.tok <= token.DCO_ASSIGN:
                d := p.parseDefineClause(p.tok, x).(*ast.DefineClause)
                x = &ast.UseDefineClause{ d }
        default:
                p.warn(x.Pos(), "bad statement '%v' (%T)\n", x, x)
                p.error(p.pos, fmt.Sprintf("unimplemented use statement: %T", x))
                sync(p, token.LINEND)
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
        
        if dialect == "" {
                p.scanner.LeaveCompoundLineContext()
                p.next() // skip RECIPE and parse in list mode

                for p.tok != token.LINEND && p.tok != token.EOF {
                        if isUseRule {
                                elems = append(elems, p.parseUseRecipe())
                        } else {
                                elems = append(elems, p.parseExpr(false))
                        }
                        if p.lineComment != nil {
                                comment = p.lineComment
                                break
                        }
                }
                /* if !isUseRule && len(elems) > 0 {
                        if v, e := p.runtime.Eval(elems[0], delegation); e == nil {
                                fmt.Printf("parseRecipeExpr: %T %v\n", v, v)
                                elems[0] = &ast.EvaluatedExpr{ v, elems[0] }
                        } else {
                                p.error(elems[0].Pos(), fmt.Sprintf("immediate (%s)", e))
                        }
                } /**/
        } else {
                p.next() // skip RECIPE and parse in line-string mode
                for p.tok != token.LINEND && p.tok != token.EOF {
                        elems = append(elems, p.parseExpr(false))
                        //fmt.Printf("recipe: %v %v %v\n", x, p.tok, p.lit)
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
                                                        p.error(elem.Pos(), fmt.Sprintf("bad parameter (%T, %v)", elem, e))
                                                }
                                        default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
                                                p.error(elem.Pos(), fmt.Sprintf("bad parameter form (%T)", elem))
                                        }
                                }
                                goto next
                        case *ast.DelegateExpr, *ast.ClosureExpr, *ast.Barecomp, *ast.BasicLit:
                                v, e := p.runtime.Eval(n, disclosure)
                                if e != nil {
                                        p.error(n.Pos(), fmt.Sprintf("%v (%v)", e, n))
                                        goto next
                                }
                                if name, pos = v.Strval(), x.Pos(); name == "" {
                                        p.error(n.Pos(), fmt.Sprintf("empty name (%v)", n))
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
                                p.error(pos, fmt.Sprintf("multi-dialect unsupported, already defined '%s'", dialect))
                                goto next
                        }
                } else if p.runtime.IsModifier(name) {
                        goto addModifier
                } else {
                        p.error(pos, fmt.Sprintf("no dialect or modifier '%s'", name))
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
                p.error(p.pos, fmt.Sprintf("close scope (%v)", err))
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
                        //p.warn(target.Pos(), fmt.Sprintf("target %T (%v): %v", target, target, e))
                        p.error(target.Pos(), fmt.Sprintf("target %T: %v", target, e))
                        continue
                } else if v == nil {
                        //p.warn(target.Pos(), fmt.Sprintf("target %T (%v): nil", target, target))
                        p.error(target.Pos(), fmt.Sprintf("target %T: nil", target))
                        continue
                }

                var name = v.Strval()
                if name == "" {
                        p.error(target.Pos(), "empty entry name")
                        continue
                } else if name == "use" {
                        if i == 0 && len(targets) == 1 {
                                isUseRule = true
                        } else {
                                p.error(target.Pos(), "mixing use with normal rules")
                        }
                }

                var tarent *types.RuleEntry
                if sym, alt := p.runtime.Symbol(name, types.RuleEntryType); alt != nil {
                        if entry, ok := alt.(*types.RuleEntry); entry == nil {
                                p.warn(target.Pos(), fmt.Sprintf("'%s' already taken (%T)", name, alt))
                                p.error(target.Pos(), fmt.Sprintf("name '%s' already taken", name))
                        } else if entry.Program() != nil {
                                p.warn(target.Pos(), fmt.Sprintf("rule entry already defined (%s)", name))
                                p.error(target.Pos(), fmt.Sprintf("rule already defined (%s)", name))
                        } else if ok {
                                tarent = entry
                        } else {
                                p.error(target.Pos(), fmt.Sprintf("invalid rule (%s)", name))
                        }
                } else if sym != nil {
                        tarent = sym.(*types.RuleEntry)
                } else {
                        p.warn(target.Pos(), fmt.Sprintf("target entry (%s)", name))
                        p.error(target.Pos(), fmt.Sprintf("bad entry name (%s)", name))
                        continue
                }

                if tarent == nil {
                        p.error(target.Pos(), fmt.Sprintf("bad entry name (%s)", name))
                        continue
                }

                //p.warn(target.Pos(), fmt.Sprintf("%v: %v (%T %v)", tarent, tarent.Class(), target, p.runtime.IsFileName(name)))

                // Guessing target entry class, e.g. general, file, etc.
                var class = tarent.Class()
                switch target.(type) {
                case *ast.Barefile, *ast.PathExpr:
                        class = types.FileRuleEntry
                case *ast.Bareword:
                        if p.runtime.IsFileName(name) {
                                class = types.FileRuleEntry
                        }
                }
                tarent.SetClass(class)

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

        scope := p.runtime.OpenScope(p.pos, fmt.Sprintf("rule %s", scopeComment))
        for _, s := range automatics {
                if sym, alt := p.runtime.Symbol(s, types.DefineType); alt != nil {
                        p.warn(p.pos, fmt.Sprintf("'%s' already taken", s))
                        p.error(p.pos, fmt.Sprintf("name '%s' already taken", s))
                } else if sym == nil {
                        // TODO: errors
                }
        }
        for _, s := range params {
                if sym, alt := p.runtime.Symbol(s, types.DefineType); alt != nil {
                        p.warn(p.pos, fmt.Sprintf("'%s' already taken", s))
                } else if sym == nil {
                        // TODO: errors
                }
        }
        for i := 1; i < 10; i += 1 {
                if sym, alt := p.runtime.Symbol(strconv.Itoa(i), types.DefineType); alt != nil {
                        p.warn(p.pos, fmt.Sprintf("'%v' already taken", i))
                } else if sym == nil {
                        // TODO: errors
                }
        }

        // Parsing depends after automatics and parameters are defined, so that
        // the depend list can refer to automatics and parameters.
        if p.tok != token.LINEND {
                depends = p.parseRhsList(true)
                dependsLoop: for i, depend := range depends {
                        var (
                                args []types.Value
                        )
                        switch dep := depend.(type) {
                        case *ast.Barecomp:
                                if g := dep.Elems[len(dep.Elems)-1].(*ast.GroupExpr); g != nil {
                                        dep.Elems = dep.Elems[:len(dep.Elems)-1]
                                        for _, elem := range g.Elems {
                                                v, e := p.runtime.Eval(elem, delegation)
                                                if e != nil {
                                                        p.error(depend.Pos(), fmt.Sprintf("bad arg (%s)", e))
                                                        continue
                                                }
                                                args = append(args, v)
                                        }
                                }
                                if nelems := len(dep.Elems); nelems == 1 {
                                        depend = dep.Elems[0]
                                } else if nelems == 0 {
                                        p.error(depend.Pos(), fmt.Sprintf("bad depend"))
                                }
                        }

                        depval, err := p.runtime.Eval(depend, ruledepend)
                        if err != nil {
                                p.error(depend.Pos(), fmt.Sprintf("%s", err))
                                continue
                        }

                        fmt.Printf("depend: %T -> %T %v\n", depend, depval, depval)
                        depends[i] = &ast.EvaluatedExpr{ depval, depend }
                        continue dependsLoop

                        switch depval.Type() {
                        case types.RuleEntryType, types.PatternType, types.ClosureType, types.DelegateType, types.ListType:
                                depends[i] = &ast.EvaluatedExpr{ depval, depend }
                                continue dependsLoop
                        }
                        
                        var (
                                name = depval.Strval()
                                depent *types.RuleEntry
                        )
                        // Resolve the name first (this will look into the bases too),
                        if sym := p.runtime.Resolve(name); sym != nil {
                                if entry, ok := sym.(*types.RuleEntry); ok && entry != nil {
                                        depval = sym.(types.Value)
                                        depent = entry // depval.(*types.RuleEntry)
                                } else {
                                        p.warn(depend.Pos(), fmt.Sprintf("'%s' already taken (%T)", name, sym))
                                        p.error(depend.Pos(), fmt.Sprintf("name '%s' already taken", name))
                                }
                        } else if sym, alt := p.runtime.Symbol(name, types.RuleEntryType); alt != nil {
                                if true {
                                        depval = alt.(types.Value)
                                        depent = depval.(*types.RuleEntry)
                                } else {
                                        p.warn(depend.Pos(), fmt.Sprintf("'%s' already taken (%T)", name, alt))
                                        p.error(depend.Pos(), fmt.Sprintf("name '%s' already taken", name))
                                }
                        } else if sym != nil { // New entry (not previously defined)!
                                depval = sym.(types.Value)
                                depent = depval.(*types.RuleEntry)
                                class := depent.Class()
                                switch depend.(type) {
                                case *ast.Barefile, *ast.PathExpr:
                                        class = types.FileRuleEntry
                                case *ast.Bareword:
                                        if p.runtime.IsFileName(name) {
                                                class = types.FileRuleEntry
                                        }
                                default:
                                        p.error(depend.Pos(), fmt.Sprintf("unknown depend class '%v' (%T) (%T)", depval, depval, depend))
                                }
                                depent.SetClass(class)
                        } else {
                                p.error(depend.Pos(), fmt.Sprintf("bad depend (%v %v)", depval, args))
                                continue
                        }

                        //fmt.Printf("depend: %T %v -> %T %v\n", depend, depend, depval, depval)
                        
                        if len(args) > 0 {
                                //fmt.Printf("depend: %v %v\n", depent, args)
                                if depent != nil {
                                        depval = &types.ArgumentedEntry{ depent, args }
                                } else {
                                        //depval = &types.Argumented{ depval, args }
                                        p.error(depend.Pos(), fmt.Sprintf("argumented depend (%v %v)", depval, args))
                                }
                        }
                        depends[i] = &ast.EvaluatedExpr{ depval, depend }
                }
        }
        if p.tok != token.EOF {
                p.expectLinend()
        }
        
        // Parse recipes in the program scope.
        for p.tok == token.RECIPE {
                recipes = append(recipes, p.parseRecipeExpr(dialect, isUseRule))
        }

        program = &ast.ProgramExpr{
                Lang: 0, // FIXME: language definition
                Params: params,
                Values: recipes,
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

        // Close the rule scope and go back to project scope. The current
        // scope must be project scope befor DeclareRule.
        p.closeScope(scope)

        if _, err := p.runtime.DeclareRule(clause); err != nil {
                p.error(pos, fmt.Sprintf("%s", err))
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
                if token.COLON <= p.tok && p.tok <= token.QUE {
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
        if !p.tok.IsRuleDelim() /*p.tok < token.COLON || token.CALL <= p.tok*/ {
                list = append(list, p.parseLhsList()...)
        }
        if p.tok.IsRuleDelim() /*token.COLON <= p.tok && p.tok <= token.QUE*/ {
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

        scope := p.runtime.OpenScope(p.pos, fmt.Sprintf("file %s", filename))
        if scope != nil {
                defer p.closeScope(scope)
                var (
                        sym RuntimeObj
                        abs = filepath.Dir(filename)
                        rel, _ = filepath.Rel(p.runtime.Getwd(), abs)
                )
                sym, _ = p.runtime.Symbol("/", types.DefineType)
                sym.(*types.Def).Set(values.String(abs))
                sym, _ = p.runtime.Symbol(".", types.DefineType)
                sym.(*types.Def).Set(values.String(rel))
                //relParent = filepath.Dir(rel)
                //if rel == "." && relParent == "." {
                //        relParent = ".."
                //}
                //fmt.Printf("%v\n", filename)
                //fmt.Printf("%v\n", p.runtime.Resolve("/"))
                //fmt.Printf("%v\n", p.runtime.Resolve("."))
        } else {
                p.error(p.pos, fmt.Sprintf("open scope"))
        }

        if keyword = p.tok; keyword == token.PROJECT || keyword == token.MODULE {
                if p.mode&Flat != 0 {
                        p.error(p.pos, fmt.Sprintf("forbidden %v in flat file", p.tok))
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
                        ident = p.parseBareword()
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
                                p.error(p.pos, fmt.Sprintf("immediate (%v)", err))
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
                                if _, ok := err.(scanner.Errors); ok {
                                        fmt.Printf("%v\n", err)
                                } else {
                                        p.warn(ident.Pos(), fmt.Sprintf("project %s (%v)", ident.Value, err))
                                }
                                p.error(ident.Pos(), fmt.Sprintf("declare %s error", ident.Value))
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

	// resolve global iers within the same file
	/*i := 0
	for _, ident := range p.unresolved {
		// i <= index for current ident
		assert(ident.Sym == unresolved, "symbol already resolved")
                
                // also removes unresolved sentinel
                //ident.Sym = scope.Resolve(ident.Value)
                ident.Sym = p.runtime.Resolve(ident.Value)
		if ident.Sym == unresolved {
                        p.warn(ident.Pos(), fmt.Sprintf("%s is undefined", ident.Value))
			p.unresolved[i] = ident
			i++
		}
	}/**/

	return &ast.File{
		Doc:        doc,
		Keypos:     pos,
                Keyword:    keyword,
		Name:       ident,
		Scope:      scope,
		Clauses:    clauses,
		Imports:    p.imports,
                //Unresolved: p.unresolved[0:i],
		Comments:   p.comments,
	}
}
