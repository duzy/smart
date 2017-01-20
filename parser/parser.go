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

	// Ordinary identifier scopes
	pkgScope   *ast.Scope        // pkgScope.Outer == nil
	topScope   *ast.Scope        // top-most scope; may be pkgScope
	//imports    []*ast.ImportSpec // list of imports
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

// expectClosing is like expect but provides a better error message
// for the common case of a missing comma before a newline.
//
func (p *parser) expectClosing(tok token.Token, context string) token.Pos {
	if p.tok != tok && p.tok == token.SEMICOLON && p.lit == "\n" {
		p.error(p.pos, "missing ',' before newline in "+context)
		p.next()
	}
	return p.expect(tok)
}

func (p *parser) expectLinend() {
        if p.tok == token.LINEND {
                p.next()
        } else {
                p.errorExpected(p.pos, "'\n'")
                //syncStmt(p)
	}
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

func (p *parser) parseIdent() *ast.Ident {
	pos := p.pos
	name := "_"
	if p.tok == token.IDENT {
		name = p.lit
		p.next()
	} else {
		p.expect(token.IDENT) // use expect() error handling
	}
	return &ast.Ident{NamePos: pos, Name: name}
}

func assert(cond bool, msg string) {
	if !cond {
		panic("go/parser internal error: " + msg)
	}
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
	// Go spec: The package clause is not a declaration;
	// the package name does not appear in any scope.
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
	if p.mode&PackageClauseOnly == 0 {
		// import decls
                /*
		for p.tok == token.IMPORT {
			decls = append(decls, p.parseGenDecl(token.IMPORT, p.parseImportSpec))
		}

		if p.mode&ImportsOnly == 0 {
			// rest of package body
			for p.tok != token.EOF {
				decls = append(decls, p.parseDecl(syncDecl))
			}
		} */
	}
	p.closeScope()
	assert(p.topScope == nil, "unbalanced scopes")

	return &ast.File{
		Doc:        doc,
		Package:    pos,
		Name:       ident,
		Decls:      decls,
		Scope:      p.pkgScope,
		//Imports:    p.imports,
		//Unresolved: p.unresolved[0:i],
		Comments:   p.comments,
	}
}
