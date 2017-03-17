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

type Clause interface {
	Node
	clauseNode()
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

        /* Ident struct {
                Bareword
		Sym *Symbol   // denoted symbol; or nil
        } */

	// A Bareword represents a word without decorations or an identifier.
	Bareword struct {
		ValuePos token.Pos // identifier position
		Value    string    // identifier name
	}

	// A BasicLit node represents a literal of basic type.
	BasicLit struct {
		ValuePos token.Pos   // literal position
		Kind     token.Token // token.INT, token.FLOAT, token.IMAG, token.CHAR, or token.STRING
		Value    string      // literal string; e.g. 42, 0x7f, 3.14, 1e-9, 2.4i, 'a', '\x7f', "foo" or `\m\n\o`
	}

        // A CompoundLit node represents a composed list of expressions (not separated by spaces).
        CompoundLit struct {
                BegPos token.Pos
                Elems []Expr
                EndPos token.Pos
                Quote  bool // is quoted compound literal?
        }

        // A ListExpr node represents a list of expressions (seperated spaces).
        ListExpr struct {
                Elems []Expr
        }

        // Group expression surrounded by '(' and ')'.
        GroupExpr struct {
                Lparen token.Pos
                Elems []Expr
                Rparen token.Pos
        }

        // Call expression
        CallExpr struct {
                Dollar token.Pos
                Lparen token.Pos
                Name Expr
                Args []Expr // *ListExpr
                Rparen token.Pos
                TokLp token.Token // left paren token
                Tok token.Token
        }

	// A UnaryExpr node represents a unary expression.
	// Currently only '+', '-' are defined for numbers.
	//
	UnaryExpr struct {
		OpPos token.Pos   // position of Op
		Op    token.Token // operator
		X     Expr        // operand
	}

	// A BinaryExpr node represents a binary expression.
	BinaryExpr struct {
		X     Expr        // left operand
		OpPos token.Pos   // position of Op
		Op    token.Token // operator
		Y     Expr        // right operand
	}

	// A KeyValueExpr node represents (key : value) pairs
	// in composite literals.
	//
	KeyValueExpr struct {
		Key   Expr
		Colon token.Pos // position of ":"
		Value Expr
	}

        // A ModifierExpr node represents [...] expression
        ModifierExpr struct {
                Lbrack token.Pos
                Elems []Expr
                Rbrack token.Pos
        }

        RecipeExpr struct {
		Doc     *CommentGroup // associated documentation; or nil
                TabPos  token.Pos
                Elems   []Expr
		Comment *CommentGroup // line comments after RPAREN; or nil
                LendPos token.Pos
        }

        ProgramExpr struct {
                Lang    int // TODO: language definition (default is recipes)
                Values  []Expr
        }
)

func (d *BadExpr) Pos() token.Pos         { return d.From }
func (d *Bareword) Pos() token.Pos        { return d.ValuePos }
func (d *BasicLit) Pos() token.Pos        { return d.ValuePos }
func (d *CompoundLit) Pos() token.Pos     { return d.BegPos }
func (d *CallExpr) Pos() token.Pos        { return d.Dollar }
func (d *ListExpr) Pos() token.Pos        { return d.Elems[0].Pos() }
func (d *GroupExpr) Pos() token.Pos       { return d.Lparen }
func (d *UnaryExpr) Pos() token.Pos       { return d.OpPos }
func (d *BinaryExpr) Pos() token.Pos      { return d.OpPos }
func (d *KeyValueExpr) Pos() token.Pos    { return d.Colon }
func (d *ModifierExpr) Pos() token.Pos    { return d.Lbrack }
func (d *RecipeExpr) Pos() token.Pos      { return d.TabPos }
func (d *ProgramExpr) Pos() token.Pos     { return d.Values[0].Pos() }

func (d *BadExpr) End() token.Pos         { return d.From }
func (d *Bareword) End() token.Pos        { return d.ValuePos }
func (d *BasicLit) End() token.Pos        { return d.ValuePos }
func (d *CompoundLit) End() token.Pos     { return d.EndPos }
func (d *ListExpr) End() token.Pos        { return d.Elems[len(d.Elems)-1].End() }
func (d *CallExpr) End() token.Pos        { return d.Rparen }
func (d *GroupExpr) End() token.Pos       { return d.Rparen }
func (d *UnaryExpr) End() token.Pos       { return d.OpPos }
func (d *BinaryExpr) End() token.Pos      { return d.OpPos }
func (d *KeyValueExpr) End() token.Pos    { return d.Colon }
func (d *ModifierExpr) End() token.Pos    { return d.Rbrack }
func (d *RecipeExpr) End() token.Pos      { return d.LendPos }
func (d *ProgramExpr) End() token.Pos     { return d.Values[len(d.Values)-1].End() }

func (*BadExpr) exprNode()         {}
func (*Bareword) exprNode()        {}
func (*BasicLit) exprNode()        {}
func (*CompoundLit) exprNode()     {}
func (*ListExpr) exprNode()        {}
func (*CallExpr) exprNode()        {}
func (*GroupExpr) exprNode()       {}
func (*UnaryExpr) exprNode()       {}
func (*BinaryExpr) exprNode()      {}
func (*KeyValueExpr) exprNode()    {}
func (*ModifierExpr) exprNode()    {}
func (*RecipeExpr) exprNode()      {}
func (*ProgramExpr) exprNode()     {}

func NewBareword(name string) *Bareword { return &Bareword{token.NoPos, name} }

func (id *Bareword) String() string {
	if id != nil {
		return id.Value
	}
	return "<nil>"
}

// A declaration is represented by one of the following declaration nodes.
//
type (
	// The Spec type stands for any directive.
        // 
	Spec interface {
		Node
		specNode()
	}

	// An DirectiveSpec node represents a single directive spec.
        // 
	DirectiveSpec struct {
		Doc     *CommentGroup // associated documentation; or nil
		Props   []Expr        // instance property
		Comment *CommentGroup // line comments; or nil
		EndPos  token.Pos     // end of spec (overrides Path.Pos if nonzero)
	}
        
	// An ImportSpec node represents a single project import.
        // 
	ImportSpec struct {
                DirectiveSpec
	}

	// An IncludeSpec node represents a single project include.
        // 
        IncludeSpec struct {
                DirectiveSpec
        }

	// An UseSpec node represents a single project import.
        // 
	UseSpec struct {
                DirectiveSpec
	}
        
        // A InstanceSpec node represents a project instanciation.
        // 
	InstanceSpec struct {
                DirectiveSpec
        }

        // A EvalSpec node represents evaluation statements.
        // 
	EvalSpec struct {
                DirectiveSpec
        }
)

func (s *DirectiveSpec) Pos() token.Pos {
        return s.Props[0].Pos()
}

func (s *DirectiveSpec) End() token.Pos {
        return s.Props[len(s.Props)-1].End()
}

func (*DirectiveSpec) specNode() {}

// A declaration is represented by one of the following declaration nodes.
//
type (
	// A BadClause node is a placeholder for declarations containing
	// syntax errors for which no correct declaration nodes can be
	// created.
	//
	BadClause struct {
		From, To token.Pos // position range of bad declaration
	}

	// A GenericClause node (generic declaration node) represents an import,
	// use, instance declaration.
        //
        // A valid Lparen position (Lparen.Line > 0) indicates a parenthesized
        // declaration.
	//
	// Relationship between Tok value and Specs element type:
	//
	//	token.IMPORT   *ImportSpec
	//	token.INCLUDE  *IncludeSpec
	//	token.INSTANCE *InstanceSpec
	//	token.EVAL     *EvalSpec
	//	token.USE      *UseSpec
	//
	GenericClause struct {
		Doc    *CommentGroup // associated documentation; or nil
		TokPos token.Pos     // position of Tok
		Tok    token.Token   // IMPORT, USE, INCLUDE, INSTANCE
		Lparen token.Pos     // position of '(', if any
		Specs  []Spec
		Rparen token.Pos     // position of ')', if any
                //EndPos token.Pos     // Rparen or LINEND position
	}

	// A DefineClause node represents a definition of a symbol in a statement list.
        // 
	DefineClause struct {
		Doc     *CommentGroup // associated documentation; or nil
		TokPos  token.Pos     // position of Tok
		Tok     token.Token   // '=', ':='
		Name    Expr          // name for the defining symbol
                Value   Expr          // value of the definition
		Comment *CommentGroup // line comments; or nil
	}

	// A RuleClause node represents a rule declaration.
	RuleClause struct {
		Doc     *CommentGroup  // associated documentation; or nil
		Targets []Expr         // targets
                Depends []Expr         // prerequisites
                Modifier *ModifierExpr // modifier (e.g. [shell])
                Program Expr           // program (e.g. recipes)
                TokPos  token.Pos      // position of ':', '::', etc
		Tok     token.Token    // token ':', '::'
	}
)

func (d *BadClause) Pos() token.Pos    { return d.From }
func (d *GenericClause) Pos() token.Pos    { return d.TokPos }
func (d *DefineClause) Pos() token.Pos { return d.Name.Pos() }
func (d *RuleClause) Pos() token.Pos   { return d.TokPos }

func (d *BadClause) End() token.Pos { return d.To }
func (d *GenericClause) End() token.Pos {
	if d.Rparen.IsValid() {
		return d.Rparen + 1
	}
	return d.Specs[0].End()
}
func (d *DefineClause) End() token.Pos {
        return d.Name.Pos() 
}
func (d *RuleClause) End() token.Pos {
        return d.TokPos 
}

func (*BadClause) clauseNode()     {}
func (*GenericClause) clauseNode() {}
func (*DefineClause) clauseNode()  {}
func (*RuleClause) clauseNode()    {}

// A File node represents a Smart source file.
//
// The Comments list contains all comments in the source file in order of
// appearance, including the comments that are pointed to from other nodes
// via Doc and Comment fields.
//
type File struct {
	Doc        *CommentGroup   // associated documentation; or nil
	Keypos     token.Pos       // position of "module" or "project" keyword
        Keyword    token.Token     // e.g. "module", "project"
	Name       *Bareword       // project/module name
	Clauses    []Clause        // top-level declarations; or nil
	Scope      *Scope          // module scope (this file only)
	Imports    []*ImportSpec   // imports in this file
	Comments   []*CommentGroup // list of all comments in the source file
}

// A Module node represents a set of source files
// collectively building a Module.
//
type Module struct {
	Keypos  token.Pos          // position of "module" or "project" keyword
        Keyword token.Token        // e.g. "module", "project"
	Name    string             // project name
	Scope   *Scope             // project scope across all files
	Imports map[string]*Symbol // map of project id -> project symbol
	Files   map[string]*File   // source files by filename
}

func (*Module) Pos() token.Pos { return token.NoPos }
func (*Module) End() token.Pos { return token.NoPos }
