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
        "strconv"
        "unicode"
        "strings"
        "fmt"
)

type parser struct {
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
        
	// Ordinary identifier scopes
	pkgScope   *ast.Scope        // pkgScope.Outer == nil
	topScope   *ast.Scope        // top-most scope; may be pkgScope
	unresolved []*ast.Ident      // unresolved identifiers (reference to project symbols)
	imports    []*ast.ImportSpec // list of imports
}

func (p *parser) init(fset *token.FileSet, filename string, src []byte, mode Mode) {
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
                //syncStmt(p)
	}
}

// The unresolved object is a sentinel to mark identifiers that have been added
// to the list of unresolved identifiers. The sentinel is only used for verifying
// internal consistency.
var unresolved = new(ast.Symbol)

// If x is an identifier, tryResolve attempts to resolve x by looking up
// the object it denotes. If no object is found and collectUnresolved is
// set, x is marked as unresolved and collected in the list of unresolved
// identifiers.
//
func (p *parser) tryResolve(x ast.Expr, collectUnresolved bool) {
	// nothing to do if x is not an identifier or the blank identifier
	ident, _ := x.(*ast.Ident)
	if ident == nil {
		return
	}
	assert(ident.Sym == nil, "identifier already declared or resolved")
	if ident.Name == "_" {
		return
	}
	// try to resolve the identifier
	for s := p.topScope; s != nil; s = s.Outer {
		if sym := s.Lookup(ident.Name); sym != nil {
			ident.Sym = sym
			return
		}
	}
	// all local scopes are known, so any unresolved identifier
	// must be found either in the file scope, package scope
	// (perhaps in another file), or universe scope --- collect
	// them so that they can be resolved later
	if collectUnresolved {
		ident.Sym = unresolved
		p.unresolved = append(p.unresolved, ident)
	}
}

func (p *parser) resolve(x ast.Expr) {
	p.tryResolve(x, true)
}

// ----------------------------------------------------------------------------
// Scoping

func (p *parser) openScope() {
	p.topScope = ast.NewScope(p.topScope)
}

func (p *parser) closeScope() {
	p.topScope = p.topScope.Outer
}

// ----------------------------------------------------------------------------
// Parsing

func assert(cond bool, msg string) {
	if !cond {
		panic("go/parser internal error: " + msg)
	}
}

// syncDecl advances to the next declaration.
// Used for synchronization after an error.
//
func syncDecl(p *parser) {
	for {
		switch p.tok {
		case token.IMPORT, token.INCLUDE, token.INSTANCE, token.USE, token.EXPORT, token.EVAL:
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

// checkExpr checks that x is an expression (and not a type).
func (p *parser) checkExpr(x ast.Expr) ast.Expr {
	switch /*unparen(x)*/x.(type) {
	case *ast.BadExpr:
	case *ast.Ident:
	case *ast.BasicLit:
	case *ast.CallExpr:
	case *ast.GroupExpr:
	case *ast.GroupedListExpr:
		panic("unreachable")
	case *ast.UnaryExpr:
	case *ast.BinaryExpr:
        case *ast.KeyValueExpr:
	default:
		// all other nodes are not proper expressions
		p.errorExpected(x.Pos(), "expression")
		x = &ast.BadExpr{From: x.Pos(), To: p.safePos(x.End())}
	}
	return x
}

// ----------------------------------------------------------------------------
// Identifiers

func (p *parser) parseIdent() *ast.Ident {
	pos, name := p.pos, ""
	if p.tok == token.IDENT {
		name = p.lit
		p.next()
	} else {
		p.expect(token.IDENT) // use expect() error handling
	}
	return &ast.Ident{NamePos: pos, Name: name}
}

// ----------------------------------------------------------------------------
// Common productions

func (p *parser) isEndOfList(lhs bool) bool {
        if p.tok == token.RPAREN {
                // FIXME: check pared LPAREN?
                return true
        }
        if token.COLON <= p.tok && p.tok < token.CALL {
                return true;
        }
        return p.tok == token.EOF || p.tok == token.LINEND || 
                p.tok == token.COMMA || (lhs && (
                p.tok == token.ASSIGN))
}

// If lhs is set, result list elements which are identifiers are not resolved.
func (p *parser) parseExprList(lhs bool) (list []ast.Expr) {
	if p.trace {
		defer un(trace(p, "ExpressionList"))
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

func (p *parser) parseLhsList() []ast.Expr {
	old := p.inRhs
	p.inRhs = false
	list := p.parseExprList(true)
	switch p.tok {
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
		for _, x := range list {
			p.resolve(x)
		}
	}
	p.inRhs = old
	return list
}

func (p *parser) parseRhsList() []ast.Expr {
	old := p.inRhs
	p.inRhs = true
	list := p.parseExprList(false)
	p.inRhs = old
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
                next := p.parseExprList(false)
                if !converted {
                        elems = []ast.Expr{ 
                                &ast.GroupedListExpr{ elems },
                                &ast.GroupedListExpr{ next },
                        }
                        converted = true
                } else {
                        elems = append(elems, &ast.GroupedListExpr{ next })
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

func (p *parser) parseExpr(lhs bool) ast.Expr {
	if p.trace {
		defer un(trace(p, "Expression"))
	}
        switch p.tok {
        case token.IDENT, token.INT, token.FLOAT, token.DATETIME, token.DATE, token.TIME, token.URI, token.STRING, token.RECIEPT:
                pos, tok, lit := p.pos, p.tok, p.lit
                p.next()
                return &ast.BasicLit{
                        ValuePos: pos,
                        Kind: tok,
                        Value: lit,
                }

        case token.CALL:
                var (
                        lpos = token.NoPos
                        rpos = token.NoPos
                        pos = p.pos
                        name ast.Expr
                        rest []*ast.ArgExpr
                        tokLp token.Token
                )
                switch p.next(); p.tok {
                case token.LPAREN:
                        lpos, tokLp = p.pos, p.tok
                        p.next()
                        name = p.checkExpr(p.parseExpr(false))
                        if p.tok != token.RPAREN {
                                list := p.parseExprList(false)
                                rest = append(rest, &ast.ArgExpr{ list })
                                for p.tok == token.COMMA {
                                        p.next()
                                        list = p.parseExprList(false)
                                        rest = append(rest, &ast.ArgExpr{ list })
                                }
                        }
                        rpos = p.expect(token.RPAREN)
                default:
                        name = p.checkExpr(p.parseExpr(false))
                }
                return &ast.CallExpr{
                        Dollar: pos,
                        Lparen: lpos,
                        Name: name,
                        Args: rest,
                        Rparen: rpos,
                        TokLp: tokLp,
                }
                
        case token.LPAREN:
                return p.parseGroupExpr()
                
        //case token.LBRACK:
        //case token.LBRACE:
                
        case token.PLUS, token.MINUS:
                tok, pos := p.tok, p.pos
                p.next()
                x := p.checkExpr(p.parseExpr(false))
                return &ast.UnaryExpr{
                        OpPos: pos,
                        Op: tok,
                        X: x,
                }

        case token.PROJECT, token.MODULE, token.USE, token.EXPORT, token.INCLUDE, token.IMPORT, token.INSTANCE:
                if p.inRhs {
                        pos, lit := p.pos, p.lit
                        p.next()
                        // convert keyword into IDENT
                        return &ast.BasicLit{
                                ValuePos: pos,
                                Kind: token.IDENT,
                                Value: lit,
                        }
                }
                fallthrough

        default:
                pos := p.pos
                p.errorExpected(pos, "'"+p.tok.String()+"'")
                return &ast.BadExpr{ From:pos, To:p.pos }
        }
}

// ----------------------------------------------------------------------------
// Declarations

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

/*
func (p *parser) parseImportSpec(doc *ast.CommentGroup, _ token.Token, _ int, gs *ast.DirectiveSpec) ast.Spec {
	if p.trace {
		defer un(trace(p, "ImportSpec"))
	}

	var ident *ast.Ident
	switch p.tok {
	case token.PERIOD:
		ident = &ast.Ident{NamePos: p.pos, Name: "."}
		p.next()
	case token.IDENT:
		ident = p.parseIdent()
	}

	pos := p.pos
	var path string
	if p.tok == token.STRING {
		path = p.lit
		if !isValidImport(path) {
			p.error(pos, "invalid import path: "+path)
		}
		p.next()
	} else {
		p.expect(token.STRING) // use expect() error handling
	}
	//p.expectLinend() // call before accessing p.linecomment

	// collect imports
	spec := &ast.ImportSpec{
		Doc:     doc,
		Name:    ident,
		Path:    &ast.BasicLit{ValuePos: pos, Kind: token.STRING, Value: path},
		Comment: p.lineComment,
                EndPos:  p.pos,
	}
	p.imports = append(p.imports, spec)

	return spec
} */
func (p *parser) parseImportSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        gs := p.parseDirectiveSpec()
	return &ast.ImportSpec{ gs }
}

func (p *parser) parseIncludeSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        gs := p.parseDirectiveSpec()
        return &ast.IncludeSpec{ gs }
}

func (p *parser) parseUseSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        gs := p.parseDirectiveSpec()
        return &ast.UseSpec{ gs }
}

func (p *parser) parseInstanceSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        gs := p.parseDirectiveSpec()
        return &ast.InstanceSpec{ gs }
}

func (p *parser) parseEvalSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
        gs := p.parseDirectiveSpec()
        return &ast.EvalSpec{ gs }
}

func (p *parser) parseDirectiveSpec() (gs *ast.DirectiveSpec) {
	if p.trace {
		defer un(trace(p, "DirectiveSpec"))
	}
        
        var (
                doc = p.leadComment
                comment *ast.CommentGroup
                props []ast.Expr
        )
        props = append(props, p.parseExpr(false))
        for p.tok != token.EOF {
                if p.tok == token.COMMA || p.tok == token.LINEND || p.tok == token.RPAREN {
                        break
                }
                if p.lineComment != nil {
                        // found a line comment at the end
                        comment = p.lineComment
                        break
                }
                props = append(props, p.parseExpr(false))
        }
        return &ast.DirectiveSpec{
                Doc: doc,
                Props: props,
                Comment: comment,
                EndPos: p.pos,
        }
}

func (p *parser) parseDirectiveDecl(f parseSpecFunc) *ast.DirectiveDecl {
	if p.trace {
		defer un(trace(p, "Directive("+p.tok.String()+")"))
	}

        var (
                keyword = p.tok
                doc = p.leadComment
                pos = p.expect(keyword)
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
                p.expectLinend()
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
		p.expectLinend() // endpos = p.expect(token.LINEND)
	}

	return &ast.DirectiveDecl{
		Doc:    doc,
		TokPos: pos,
		Tok:    keyword,
		Lparen: lparen,
		Specs:  specs,
		Rparen: rparen,
                //EndPos: endpos,
	}
}

func (p *parser) parseDefineDecl(tok token.Token, ident ast.Expr) ast.Decl {
	if p.trace {
		defer un(trace(p, "Define"))
	}

        doc := p.leadComment
        pos := p.expect(tok)
        elems := p.parseRhsList()
        return &ast.DefineDecl{ 
                Doc: doc,
                TokPos: pos,
                Tok: tok,
                Name: ident,
                Elems: elems,
        }
}

func (p *parser) parseRuleDecl(tok token.Token, targets []ast.Expr) ast.Decl {
	if p.trace {
		defer un(trace(p, "Rule"))
	}

        doc := p.leadComment
        pos := p.expect(tok)
        depends := p.parseRhsList()
        return &ast.RuleDecl{
                Doc: doc,
                TokPos: pos,
                Tok: tok,
                Targets: targets,
                Depends: depends,
        }
}

func (p *parser) parseDecl(sync func(*parser)) ast.Decl {
	if p.trace {
		defer un(trace(p, "Declaration"))
	}

	var f parseSpecFunc
	switch p.tok {
	case token.INCLUDE:
		f = p.parseIncludeSpec

	case token.INSTANCE:
		f = p.parseInstanceSpec

        case token.EVAL:
		f = p.parseEvalSpec

	case token.USE:
		f = p.parseUseSpec

        default:
                switch list := p.parseLhsList(); {
                case p.tok == token.ASSIGN:
                        if len(list) == 1 {
                                return p.parseDefineDecl(p.tok, list[0])
                        } else {
                                p.errorExpected(p.pos, "assign to multiple names")
                                return &ast.BadDecl{From: p.pos, To: p.pos}
                        }
                case token.COLON <= p.tok && p.tok < token.CALL:
                        return p.parseRuleDecl(p.tok, list)
                default:
                        pos := p.pos
                        p.errorExpected(pos, "'"+p.tok.String()+"'")
                        sync(p)
                        return &ast.BadDecl{From: pos, To: p.pos}
                }
	}
	return p.parseDirectiveDecl(f)
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

	// package clause
	doc := p.leadComment
	pos := p.expect(token.PROJECT)
	// Smart-lang spec:
        //   * the project clause is not a declaration;
	//   * the project name does not appear in any scope.
	ident := p.parseIdent()
	if ident.Name == "_" && p.mode&DeclarationErrors != 0 {
		p.error(p.pos, "invalid package name _")
	}
	p.expectLinend()

	// Don't bother parsing the rest if we had errors parsing the package clause.
	// Likely not a Go source file at all.
	if p.errors.Len() != 0 {
		return nil
	}

	p.openScope()
	p.pkgScope = p.topScope
	var decls []ast.Decl
	if p.mode&ModuleClauseOnly == 0 {
		// import decls
		for p.tok == token.IMPORT {
			decls = append(decls, p.parseDirectiveDecl(p.parseImportSpec))
		}
		if p.mode&ImportsOnly == 0 {
			// rest of module body
			for p.tok != token.EOF {
                                 switch p.tok {
                                 case token.LINEND: 
                                         p.next() // skip empty lines
                                 default:
                                         decls = append(decls, p.parseDecl(syncDecl))
                                 }
			}
		}
	}
	p.closeScope()

	assert(p.topScope == nil, "unbalanced scopes")

	return &ast.File{
		Doc:        doc,
		Module:     pos,
		Name:       ident,
		Decls:      decls,
		Scope:      p.pkgScope,
		Imports:    p.imports,
		Comments:   p.comments,
	}
}
