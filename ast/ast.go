//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package ast

import (
        "github.com/duzy/smart/token"
	"strings"
	//"unicode"
	//"unicode/utf8"
)

type Node interface {
        Pos() token.Pos
        End() token.Pos
}

type Decl interface {
	Node
	declNode()
}

type Stmt interface {
	Node
	stmtNode()
}

type Expr interface {
	Node
	exprNode()
}

// A Comment node represents a single #-style comment.
type Comment struct {
	Sharp token.Pos // position of "#" starting the comment
	Text  string    // comment text (excluding '\n')
}

func (c *Comment) Pos() token.Pos { return c.Sharp }
func (c *Comment) End() token.Pos { return token.Pos(int(c.Sharp) + len(c.Text)) }

// A CommentGroup represents a sequence of comments
// with no other tokens and no empty lines between.
type CommentGroup struct {
	List []*Comment // len(List) > 0
}

func (g *CommentGroup) Pos() token.Pos { return g.List[0].Pos() }
func (g *CommentGroup) End() token.Pos { return g.List[len(g.List)-1].End() }

func isWhitespace(ch byte) bool { return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' }

func stripTrailingWhitespace(s string) string {
	i := len(s)
	for i > 0 && isWhitespace(s[i-1]) {
		i--
	}
	return s[0:i]
}

// Text returns the text of the comment.
// Comment markers (#), the first space of a line comment, and
// leading and trailing empty lines are removed. Multiple empty lines are
// reduced to one, and trailing space on lines is trimmed. Unless the result
// is empty, it is newline-terminated.
//
func (g *CommentGroup) Text() string {
	if g == nil {
		return ""
	}
	comments := make([]string, len(g.List))
	for i, c := range g.List {
		comments[i] = string(c.Text)
	}

	lines := make([]string, 0, 10) // most comments are less than 10 lines
	for _, c := range comments {
		// Remove comment markers (#).
		// The parser has given us exactly the comment text.
                c = c[1:] // #-style comment (no newline at the end)
                // strip first space - required for Example tests
                if len(c) > 0 && c[0] == ' ' {
                        c = c[1:]
                }

		// Split on newlines.
		cl := strings.Split(c, "\n")

		// Walk lines, stripping trailing white space and adding to list.
		for _, l := range cl {
			lines = append(lines, stripTrailingWhitespace(l))
		}
	}

	// Remove leading blank lines; convert runs of
	// interior blank lines to a single blank line.
	n := 0
	for _, line := range lines {
		if line != "" || n > 0 && lines[n-1] != "" {
			lines[n] = line
			n++
		}
	}
	lines = lines[0:n]

	// Add final "" entry to get trailing newline from Join.
	if n > 0 && lines[n-1] != "" {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// An expression is represented by a tree consisting of one
// or more of the following concrete expression nodes.
//
type (
	// A BadExpr node is a placeholder for expressions containing
	// syntax errors for which no correct expression nodes can be
	// created.
	//
	BadExpr struct {
		From, To token.Pos // position range of bad expression
	}

	// An Ident node represents an identifier.
	Ident struct {
		NamePos token.Pos // identifier position
		Name    string    // identifier name
		Sym     *Symbol   // denoted object; or nil
	}

        
        
)

// A File node represents a Smart source file.
//
// The Comments list contains all comments in the source file in order of
// appearance, including the comments that are pointed to from other nodes
// via Doc and Comment fields.
//
type File struct {
	Doc        *CommentGroup   // associated documentation; or nil
	Package    token.Pos       // position of "package" keyword
	Name       *Ident          // package name
	Decls      []Decl          // top-level declarations; or nil
	Scope      *Scope          // package scope (this file only)
	//Imports    []*ImportSpec   // imports in this file
	//Unresolved []*Ident        // unresolved identifiers in this file
	Comments   []*CommentGroup // list of all comments in the source file
}

// A Package node represents a set of source files
// collectively building a Go package.
//
type Package struct {
	Name    string             // package name
	Scope   *Scope             // package scope across all files
	Imports map[string]*Symbol // map of package id -> package symbol
	Files   map[string]*File   // Go source files by filename
}

func (p *Package) Pos() token.Pos { return token.NoPos }
func (p *Package) End() token.Pos { return token.NoPos }
