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

        EvaluatedExpr struct {
                Data interface{}
                Expr
        }

	// A Bareword represents a word without decorations or an identifier.
	Bareword struct {
		ValuePos token.Pos // bareword position
		Value    string    // bareword value
	}

        Ident struct {
                *Bareword
		Sym Symbol   // denoted symbol; or nil
        }
        
	// A BasicLit node represents a literal of basic type.
	BasicLit struct {
		ValuePos token.Pos   // literal position
		Kind     token.Token // token.INT, token.FLOAT, token.CHAR, or token.STRING
		Value    string      // literal string; e.g. 42, 0x7f, 3.14, 1e-9, 2.4i, 'a', '\x7f', "foo" or `\m\n\o`
	}

        // A FlagExpr is a bare word leading by dash '-'.
        FlagExpr struct {
                DashPos token.Pos
                Name    Expr
        }

        // A CompoundLit node represents a composed list of expressions (not separated by spaces).
        CompoundLit struct {
                Lquote token.Pos
                Elems  []Expr
                Rquote token.Pos
        }

        // A Barecomp node represents a bare composing expression.
	Barecomp struct {
                Elems  []Expr
        }

        // A Barefile node represents a bare file expression (with extension).
        Barefile struct {
                Name    Expr      // basename
                ExtPos  token.Pos // extension position
                Ext     string    // extension
        }

        // A GlobExpr node represents an expression containing glob characters "*?".
        GlobExpr struct {
                TokPos token.Pos
                Tok token.Token
        }
        
        // A ListExpr node represents a list of expressions (seperated spaces).
        ListExpr struct {
                Elems []Expr
        }

        // GroupExpr is a expression surrounded by '(' and ')'.
        GroupExpr struct {
                Lparen token.Pos
                Elems []Expr
                Rparen token.Pos
        }

        // A PathExpr node represents a path (expressions concated by '/').
        PathExpr struct {
                PosBeg token.Pos
                Segments []Expr
                PosEnd token.Pos
        }

        // A SelectorExpr node represents an expression followed by a selector.
	SelectorExpr struct {
		X Expr // expression
		S Expr // field selector, or sub-selector
	}

        // RefExpr expression: &$(foo)
        RefExpr struct {
                Tok token.Pos  //
                X Expr         //
        }
        
        // CallExpr expression
        CallExpr struct {
                Dollar token.Pos  //
                Lparen token.Pos  //
                Name Expr         //
                Args []Expr       // *ListExpr
                Rparen token.Pos  //
                TokLp token.Token // left paren token
                Tok token.Token   //
        }

	// A PercExpr node represents a percent expression.
        PercExpr struct {
		X     Expr        // left operand (or nil)
		OpPos token.Pos   // position of '%'
		Y     Expr        // right operand
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
        // Currently only '.' is defined to compose names.
	BinaryExpr struct {
		X     Expr        // left operand
		OpPos token.Pos   // position of Op
		Op    token.Token // operator
		Y     Expr        // right operand
	}

	// A KeyValueExpr node represents 'key=value' pairs
	// in composite literals.
	//
	KeyValueExpr struct {
		Key   Expr
                Tok   token.Token
		Equal token.Pos // position of "="
		Value Expr
	}

        // A ModifierExpr node represents [...] expression
        ModifierExpr struct {
                Lbrack token.Pos
                Elems []Expr
                Rbrack token.Pos
        }

        RecipeExpr struct {
                Dialect string
		Doc     *CommentGroup // associated documentation; or nil
                TabPos  token.Pos
                Elems   []Expr
		Comment *CommentGroup // line comments after RPAREN; or nil
                LendPos token.Pos
        }

        ProgramExpr struct {
                Lang    int // TODO: language definition (default is recipes)
                Values  []Expr
                Scope   Scope // scope specific to the program
        }
)

func (d *BadExpr) Pos() token.Pos         { return d.From }
//func (d *Ident) Pos() token.Pos           { return d.NamePos }
func (d *Bareword) Pos() token.Pos        { return d.ValuePos }
func (d *BasicLit) Pos() token.Pos        { return d.ValuePos }
func (d *FlagExpr) Pos() token.Pos        { return d.DashPos }
func (d *CompoundLit) Pos() token.Pos     { return d.Lquote }
func (d *PathExpr) Pos() token.Pos        { return d.PosBeg }
func (d *SelectorExpr) Pos() token.Pos    { return d.X.Pos() }
func (d *GlobExpr) Pos() token.Pos        { return d.TokPos }
func (d *RefExpr) Pos() token.Pos         { return d.Tok }
func (d *CallExpr) Pos() token.Pos        { return d.Dollar }
func (d *Barecomp) Pos() token.Pos        { return d.Elems[0].Pos() }
func (d *Barefile) Pos() token.Pos        { return d.Name.Pos() }
func (d *ListExpr) Pos() token.Pos        { return d.Elems[0].Pos() }
func (d *GroupExpr) Pos() token.Pos       { return d.Lparen }
func (d *PercExpr) Pos() token.Pos        { return d.OpPos }
func (d *UnaryExpr) Pos() token.Pos       { return d.OpPos }
func (d *BinaryExpr) Pos() token.Pos      { return d.OpPos }
func (d *KeyValueExpr) Pos() token.Pos    { return d.Key.Pos() }
func (d *ModifierExpr) Pos() token.Pos    { return d.Lbrack }
func (d *RecipeExpr) Pos() token.Pos      { return d.TabPos }
func (d *ProgramExpr) Pos() token.Pos     { return d.Values[0].Pos() }

func (d *BadExpr) End() token.Pos         { return d.From }
//func (d *Ident) End() token.Pos           { return token.Pos(int(d.NamePos) + len(d.Name)) }
func (d *Bareword) End() token.Pos        { return token.Pos(int(d.ValuePos) + len(d.Value)) }
func (d *BasicLit) End() token.Pos        { return token.Pos(int(d.ValuePos) + len(d.Value)) }
func (d *FlagExpr) End() token.Pos        { return d.Name.End() }
func (d *CompoundLit) End() token.Pos     { return d.Rquote + 1 }
func (d *Barecomp) End() token.Pos        { return d.Elems[len(d.Elems)-1].End() }
func (d *Barefile) End() token.Pos        { return token.Pos(int(d.ExtPos) + len(d.Ext)) }
func (d *ListExpr) End() token.Pos        { return d.Elems[len(d.Elems)-1].End() }
func (d *PathExpr) End() token.Pos        { return d.PosEnd }
func (d *SelectorExpr) End() token.Pos    { return d.S.End() }
func (d *GlobExpr) End() token.Pos        { return d.TokPos + 1 }
func (d *RefExpr) End() token.Pos         { return d.X.End() }
func (d *CallExpr) End() token.Pos        { return d.Rparen + 1 }
func (d *GroupExpr) End() token.Pos       { return d.Rparen + 1 }
func (d *PercExpr) End() token.Pos        { return d.OpPos + 1 }
func (d *UnaryExpr) End() token.Pos       { return d.OpPos + 1 }
func (d *BinaryExpr) End() token.Pos      { return d.OpPos + 1 }
func (d *KeyValueExpr) End() token.Pos    { return d.Value.End() }
func (d *ModifierExpr) End() token.Pos    { return d.Rbrack + 1 }
func (d *RecipeExpr) End() token.Pos      { return d.LendPos /*+ 1*/ }
func (d *ProgramExpr) End() token.Pos     { return d.Values[len(d.Values)-1].End() }

func (*BadExpr) exprNode()         {}
//func (*Ident) exprNode()           {}
func (*Bareword) exprNode()        {}
func (*BasicLit) exprNode()        {}
func (*FlagExpr) exprNode()        {}
func (*CompoundLit) exprNode()     {}
func (*Barecomp) exprNode()        {}
func (*Barefile) exprNode()        {}
func (*ListExpr) exprNode()        {}
func (*PathExpr) exprNode()        {}
func (*SelectorExpr) exprNode()    {}
func (*GlobExpr) exprNode()        {}
func (*RefExpr) exprNode()         {}
func (*CallExpr) exprNode()        {}
func (*GroupExpr) exprNode()       {}
func (*PercExpr) exprNode()        {}
func (*UnaryExpr) exprNode()       {}
func (*BinaryExpr) exprNode()      {}
func (*KeyValueExpr) exprNode()    {}
func (*ModifierExpr) exprNode()    {}
func (*RecipeExpr) exprNode()      {}
func (*ProgramExpr) exprNode()     {}

/* func NewBareword(name string) *Bareword { return &Bareword{token.NoPos, name, token.NoPos} }

func (id *Bareword) String() string {
	if id != nil {
		return id.Value
	}
	return "<nil>"
} */

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

        FilesSpec struct {
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
	//	token.IMPORT     *ImportSpec
	//	token.INCLUDE    *IncludeSpec
	//	token.INSTANCE   *InstanceSpec
	//	token.FILES      *FilesSpec
	//	token.EVAL       *EvalSpec
	//	token.USE        *UseSpec
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
		Tok     token.Token   // '=', ':=', '+=', '?=', etc.
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

        UseDefineClause struct {
                *DefineClause
        }
        UseRuleClause struct {
                *RuleClause
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

func (*UseDefineClause) exprNode() {}
func (*UseRuleClause) exprNode() {}

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
	Name       *Ident          // project/module name
	Scope      Scope          // module scope (this file only)
	Clauses    []Clause        // top-level declarations; or nil
	Imports    []*ImportSpec   // imports in this file
	Unresolved []*Ident        // unresolved identifiers in this file
	Comments   []*CommentGroup // list of all comments in the source file
        Files      map[string][]string
}

// A Project node represents a set of source files
// collectively building a Project.
//
type Project struct {
	Keypos  token.Pos          // position of "project" keyword
	Name    string             // project name
	Scope   Scope             // project scope across all files
	Imports map[string]Symbol // map of project id -> project symbol
	Files   map[string]*File   // source files by filename
        Runtime interface{}
}

func (*Project) Pos() token.Pos { return token.NoPos }
func (*Project) End() token.Pos { return token.NoPos }
