//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package ast

import (
        "extbit.io/smart/token"
	"strings"
	//"unicode"
	//"unicode/utf8"
        "fmt"
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
	expr()
        String() string
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
                Expr
                Data interface{}
        }

	// A Bareword represents a word without decorations or an identifier.
	Bareword struct {
		ValuePos token.Pos // bareword position
		Value    string    // bareword value
	}

	// A Constant represents a word without decorations or an identifier.
	Constant struct {
		TokPos token.Pos // position
		Tok token.Token // constant token
	}

	// A BasicLit node represents a literal of basic type.
	BasicLit struct {
		ValuePos token.Pos   // literal position
		Kind     token.Token // token.INT, token.FLOAT, token.CHAR, or token.STRING
		Value    string      // literal string; e.g. 42, 0x7f, 3.14, 1e-9, 2.4i, 'a', '\x7f', "foo" or `\m\n\o`
                EndPos   token.Pos
	}

        // A GlobMeta node represents an glob meta characters "*?".
        GlobMeta struct {
                TokPos token.Pos
                Tok token.Token
        }

        // A GlobRange node represents an glob range term "[a-b]".
        GlobRange struct {
                Chars Expr
        }
        
        // A FlagExpr is a bare word leading by dash '-'.
        FlagExpr struct {
                DashPos token.Pos
                Name    Expr
        }

        // A NegExpr represents a negative expression '!...'.
        NegExpr struct {
                NegPos token.Pos
                Val    Expr
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
                Name    Expr     // basename
                File interface{} // *types.File
                Val interface{}  // name value, to avoid re-eval Name expr.
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
                Segments []Expr
                Path interface{} // *types.Path
        }

        PathSegExpr struct { // '/', '~', '.' (only like './'), '..' (only like './')
                TokPos token.Pos
                Tok token.Token
        }

        URLExpr struct {
                Scheme Expr
                Colon1 token.Pos // ':'
                SlashSlash token.Pos // '//'
                Username Expr
                Colon2 token.Pos // username:password
                Password Expr
                At token.Pos // '@'
                Host Expr
                Colon3 token.Pos // host:port
                Port Expr
                Path Expr
                Que token.Pos // '?'
                Query Expr
                NumSign token.Pos // '#'
                Fragment Expr
        }

        // Delegate expressions: $(foo a1,a2,a3), $(foo), $foo
        // Closure expressions: &(foo a1,a2,a3), &(foo), &foo
        ClosureDelegate struct {
                Position token.Position // file position of TokPos
                TokPos token.Pos  // position of $ or &
                Lparen token.Pos  // left paren position (could be zero)
                Name Expr         // name being referred
                Resolved Symbol   // resolved symbol by name
                Args []Expr       // *ListExpr
                Rparen token.Pos  // right paren position (could be zero)
                TokLp token.Token // left paren token (could be ILLEGAL)
                Tok token.Token   // $, $/, $., $1, etc. or &
        }

        ClosureExpr struct {
                ClosureDelegate
        }

        DelegateExpr struct {
                ClosureDelegate
        }

        SelectionExpr struct {
                Lhs Expr
                Tok token.Token
                Rhs Expr
        }

        ArgumentedExpr struct {
                X Expr
                Arguments []Expr
                EndPos token.Pos
        }

	// A PercExpr node represents a percent expression.
        PercExpr struct {
		X     Expr        // left operand (or nil)
		OpPos token.Pos   // position of '%'
		Y     Expr        // right operand
        }

	// A GlobExpr node represents a glob pattern expression.
        GlobExpr struct {
		Components []Expr
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

        // A ModifiersExpr node represents [...] expression
        ModifiersExpr struct {
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
                Params  []string
                Recipes []Expr
                Scope   Scope // scope specific to the program
        }
)

func (d *BadExpr) Pos() token.Pos         { return d.From }
func (d *Bareword) Pos() token.Pos        { return d.ValuePos }
func (d *Constant) Pos() token.Pos        { return d.TokPos }
func (d *BasicLit) Pos() token.Pos        { return d.ValuePos }
func (d *FlagExpr) Pos() token.Pos        { return d.DashPos }
func (d *NegExpr) Pos() token.Pos         { return d.NegPos }
func (d *CompoundLit) Pos() token.Pos     { return d.Lquote }
func (d *PathExpr) Pos() (pos token.Pos) {
        if d.Segments == nil {
                pos = token.NoPos
        } else {
                pos = d.Segments[0].Pos()
        }
        return
}
func (d *URLExpr) Pos() token.Pos         { return d.Scheme.Pos() }
func (d *PathSegExpr) Pos() token.Pos     { return d.TokPos }
func (d *ClosureDelegate) Pos() token.Pos { return d.TokPos }
func (d *SelectionExpr) Pos() token.Pos   { return d.Lhs.Pos() }
func (d *ArgumentedExpr) Pos() token.Pos  { return d.X.Pos() }
func (d *Barecomp) Pos() (pos token.Pos) {
        if d.Elems == nil {
                pos = token.NoPos
        } else {
                pos = d.Elems[0].Pos()
        }
        return
}
func (d *Barefile) Pos() token.Pos        { return d.Name.Pos() }
func (d *ListExpr) Pos() (pos token.Pos) {
        if d.Elems == nil {
                pos = token.NoPos
        } else {
                pos = d.Elems[0].Pos()
        }
        return
}
func (d *GroupExpr) Pos() token.Pos       { return d.Lparen }
func (d *PercExpr) Pos() token.Pos        { return d.OpPos }
func (d *GlobExpr) Pos() token.Pos        { return d.Components[0].Pos() }
func (d *GlobMeta) Pos() token.Pos        { return d.TokPos }
func (d *GlobRange) Pos() token.Pos       { return d.Chars.Pos() - 1 }
func (d *KeyValueExpr) Pos() token.Pos    { return d.Key.Pos() }
func (d *ModifiersExpr) Pos() token.Pos   { return d.Lbrack }
func (d *RecipeExpr) Pos() token.Pos      { return d.TabPos }
func (d *ProgramExpr) Pos() token.Pos     { return d.Recipes[0].Pos() }

func (d *BadExpr) End() token.Pos         { return d.From }
func (d *Bareword) End() token.Pos        { return token.Pos(int(d.ValuePos) + len(d.Value)) }
func (d *Constant) End() token.Pos        { return token.Pos(int(d.TokPos) + len(d.Tok.String())) }
func (d *BasicLit) End() token.Pos        { return d.EndPos /*token.Pos(int(d.ValuePos) + len(d.Value))*/ }
func (d *FlagExpr) End() (pos token.Pos) {
        if d.Name == nil {
                pos = d.DashPos + 1
        } else {
                pos = d.Name.End()
        }
        return
}
func (d *NegExpr) End() token.Pos         { return d.Val.End() }
func (d *CompoundLit) End() token.Pos     { return d.Rquote + 1 }
func (d *Barecomp) End() token.Pos        { return d.Elems[len(d.Elems)-1].End() }
func (d *Barefile) End() token.Pos        { return d.Name.End() }
func (d *ListExpr) End() token.Pos        { return d.Elems[len(d.Elems)-1].End() }
func (d *PathExpr) End() token.Pos        { return d.Segments[len(d.Segments)-1].End() }
func (d *URLExpr) End() (pos token.Pos) {
        if d.Fragment != nil {
                pos = d.Fragment.End()
        } else if d.NumSign != token.NoPos {
                pos = d.NumSign
        } else if d.Query != nil {
                pos = d.Query.End()
        } else if d.Que != token.NoPos {
                pos = d.Que
        } else if d.Path != nil {
                pos = d.Path.End()
        } else if d.Port != nil {
                pos = d.Port.End()
        } else if d.Colon3 != token.NoPos {
                pos = d.Colon3
        } else if d.Host != nil {
                pos = d.Host.End()
        } else if d.At != token.NoPos {
                pos = d.At
        } else if d.Password != nil {
                pos = d.Password.End()
        } else if d.Colon2 != token.NoPos {
                pos = d.Colon2
        } else if d.Username != nil {
                pos = d.Username.End()
        } else if d.SlashSlash != token.NoPos {
                pos = d.SlashSlash
        } else if d.Colon1 != token.NoPos {
                pos = d.Colon1
        } else {
                pos = d.Scheme.End()
        }
        return
}
func (d *PathSegExpr) End() token.Pos     { if d.Tok == token.DOTDOT { return d.TokPos+2 } else { return d.TokPos+1 } }
func (d *ClosureDelegate) End() token.Pos {
        if d.TokLp == token.ILLEGAL {
                switch d.Tok {
                default: return d.TokPos + 2
                }
        }
        return d.Rparen + 1 
}
func (d *SelectionExpr) End() token.Pos   { return d.Rhs.End() }
func (d *ArgumentedExpr) End() token.Pos  { return d.EndPos }
func (d *GroupExpr) End() token.Pos       { return d.Rparen + 1 }
func (d *PercExpr) End() token.Pos        { return d.OpPos + 1 }
func (d *GlobExpr) End() token.Pos        { return d.Components[len(d.Components)-1].End() }
func (d *GlobMeta) End() token.Pos        { return d.TokPos + 1 }
func (d *GlobRange) End() token.Pos       { return d.Chars.End() + 1 }
func (d *KeyValueExpr) End() token.Pos    { return d.Value.End() }
func (d *ModifiersExpr) End() token.Pos   { return d.Rbrack + 1 }
func (d *RecipeExpr) End() token.Pos      { return d.LendPos /*+ 1*/ }
func (d *ProgramExpr) End() token.Pos     { return d.Recipes[len(d.Recipes)-1].End() }

func joins(exprs... Expr) (s string) {
        for _, x := range exprs { s += fmt.Sprintf("%v", x) }
        return
}

func joinx(sep string, exprs... Expr) (s string) {
        for i, x := range exprs {
                if i > 0 { s += sep }
                s += fmt.Sprintf("%v", x)
        }
        return
}

func (x *BadExpr) String() string         { return fmt.Sprintf("BadExpr{%v,%v}", x.From, x.To) }
func (x *EvaluatedExpr) String() string   { return x.Expr.String() }
func (x *Bareword) String() string        { return x.Value }
func (x *Constant) String() string        { return x.Tok.String() }
func (x *BasicLit) String() string        { return x.Value }
func (x *FlagExpr) String() (s string) {
        if s = `-`; x.Name != nil {
                s += x.Name.String()
        }
        return
}
func (x *NegExpr) String() string         { return `!`+x.Val.String() }
func (x *CompoundLit) String() string     { return `"`+joins(x.Elems...)+`"` }
func (x *Barecomp) String() string        { return joins(x.Elems...) }
func (x *Barefile) String() string        { return x.Name.String() }
func (x *ListExpr) String() string        { return joinx(" ", x.Elems...) }
func (x *PathExpr) String() (s string) {
        if len(x.Segments) == 1 {
                s = x.Segments[0].String()
                if s == "" { s = "/" } // it's the root
        } else {
                s = joinx("/", x.Segments...)
        }
        return
}
func (x *URLExpr) String() (s string) {
        s = x.Scheme.String()
        if x.Colon1 != token.NoPos { s += ":" }
        if x.SlashSlash != token.NoPos { s += "//" }
        if x.Username != nil { s += x.Username.String() }
        if x.Colon2 != token.NoPos { s += ":" }
        if x.Password != nil { s += x.Password.String() }
        if x.At != token.NoPos { s += "@" }
        if x.Host != nil { s += x.Host.String() }
        if x.Colon3 != token.NoPos { s += ":" }
        if x.Port != nil { s += x.Port.String() }
        if x.Path != nil { s += x.Path.String() }
        if x.Que != token.NoPos { s += "?" }
        if x.Query != nil { s += x.Query.String() }
        if x.NumSign != token.NoPos { s += "#" }
        if x.Fragment != nil { s += x.Fragment.String() }
        return
}
func (x *PathSegExpr) String() (s string) {
        // The '/' and zero seg indicates the root and tailing empty.
        if x.Tok != 0 && x.Tok != token.PCON {
                s = x.Tok.String()
        }
        return
}
func (x *ClosureDelegate) String() (s string) {
        var a string
        if len(x.Args) > 0 {
                a = " " + joinx(" ", x.Args...)
        }
        switch x.TokLp {
        case token.ILLEGAL: // ie. $@ $< $^ $/ $- ...
                // Note that x.Name is redundant in this case!
                s = fmt.Sprintf("%s", x.Tok)
        case token.LPAREN:
                s = fmt.Sprintf("%v(%v%s)", x.Tok, x.Name, a)
        case token.LBRACE:
                s = fmt.Sprintf("%v{%v%s}", x.Tok, x.Name, a)
        }
        return
}
func (x *SelectionExpr) String() string   { return fmt.Sprintf("%v%s%v", x.Lhs, x.Tok, x.Rhs) }
func (x *ArgumentedExpr) String() string  { return fmt.Sprintf("%v(%v)", x.X, joinx(",", x.Arguments...)) }
func (x *GroupExpr) String() string       { return fmt.Sprintf("(%v)", joinx(" ", x.Elems...)) }
func (x *PercExpr) String() (s string) {
        if x.X != nil { s += fmt.Sprintf("%v", x.X) }
        s += `%` // infix
        if x.Y != nil { s += fmt.Sprintf("%v", x.Y) }
        return
}
func (x *GlobExpr) String() string        { return fmt.Sprintf("%v", joins(x.Components...)) }
func (x *GlobMeta) String() string        { return x.Tok.String() }
func (x *GlobRange) String() string       { return fmt.Sprintf("[%s]", x.Chars) }
func (x *KeyValueExpr) String() string    { return fmt.Sprintf("%v%s%v", x.Key, x.Tok, x.Value) }
func (x *ModifiersExpr) String() string   { return fmt.Sprintf("[%v]", joinx(" ", x.Elems...)) }
func (x *RecipeExpr) String() string      { return fmt.Sprintf("Recipe(%s){%v}", x.Dialect, x.Elems) }
func (x *ProgramExpr) String() string     { return fmt.Sprintf("Program(%v){%v}", x.Params, x.Recipes) }

func (*BadExpr) expr()         {}
func (*Bareword) expr()        {}
func (*Constant) expr()        {}
func (*BasicLit) expr()        {}
func (*FlagExpr) expr()        {}
func (*NegExpr) expr()         {}
func (*CompoundLit) expr()     {}
func (*Barecomp) expr()        {}
func (*Barefile) expr()        {}
func (*ListExpr) expr()        {}
func (*PathExpr) expr()        {}
func (*URLExpr) expr()         {}
func (*PathSegExpr) expr()     {}
func (*ClosureDelegate) expr() {}
func (*SelectionExpr) expr()   {}
func (*ArgumentedExpr) expr()  {}
func (*GroupExpr) expr()       {}
func (*PercExpr) expr()        {}
func (*GlobExpr) expr()        {}
func (*GlobMeta) expr()        {}
func (*GlobRange) expr()       {}
func (*KeyValueExpr) expr()    {}
func (*ModifiersExpr) expr()   {}
func (*RecipeExpr) expr()      {}
func (*ProgramExpr) expr()     {}

func (d *Barecomp) Combine(x Expr) {
        if o, ok := x.(*Barecomp); ok {
                for _, elem := range o.Elems {
                        d.Combine(elem)
                }
        } else {
                d.Elems = append(d.Elems, x)
        }
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
        
	// An UseSpec node represents a single project import.
        // 
	UseSpec struct {
                DirectiveSpec
	}

	// An IncludeSpec node represents a single project include.
        // 
        IncludeSpec struct {
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
                Resolved Symbol // resolved symbol
        }

        ConfigurationSpec struct {
                DefineClause
        }
)

func (s *DirectiveSpec) Pos() token.Pos { return s.Props[0].Pos() }
func (s *DirectiveSpec) End() token.Pos { return s.Props[len(s.Props)-1].End() }
func (_ *DirectiveSpec) specNode() {}

func (_ *ConfigurationSpec) specNode() {}

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
	//	token.USE        *UseSpec
	//	token.INCLUDE    *IncludeSpec
	//	token.INSTANCE   *InstanceSpec
	//	token.FILES      *FilesSpec
	//	token.EVAL       *EvalSpec
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
                //Sym     Symbol        // the symbol created for definition
		Name    Expr          // name for the defining symbol
                Value   Expr          // value of the definition
		Comment *CommentGroup // line comments; or nil
	}

	// A RuleClause node represents a rule declaration.
	RuleClause struct {
		Doc      *CommentGroup  // associated documentation; or nil
		Targets  []Expr         // targets
                Depends  []Expr         // normal prerequisites
                Ordered  []Expr         // ordered prerequisites
                Program  Expr           // program (e.g. recipes)
                Position token.Position
                TokPos   token.Pos      // position of ':', '::', etc
		Tok      token.Token    // token ':', '::'
	}

        RecipeDefineClause struct {
                *DefineClause
        }
        RecipeRuleClause struct {
                *RuleClause
        }
        IncludeRuleClause struct {
                *RuleClause
        }
)

func (d *BadClause) Pos() token.Pos    { return d.From }
func (d *GenericClause) Pos() token.Pos    { return d.TokPos }
func (d *DefineClause) Pos() token.Pos { return d.Name.Pos() }
func (d *RuleClause) Pos() token.Pos   { return /*d.TokPos*/d.Targets[0].Pos() }

func (d *BadClause) End() token.Pos { return d.To }
func (d *GenericClause) End() token.Pos {
	if d.Rparen.IsValid() {
		return d.Rparen + 1
	}
	return d.Specs[0].End()
}
func (d *DefineClause) End() token.Pos { return d.Name.Pos() }
func (d *RuleClause) End() token.Pos { return d.TokPos }

func (d *DefineClause) String() string {
        return fmt.Sprintf("%s %s %v", d.Name, d.Tok, d.Value)
}

func (d *RecipeDefineClause) String() string {
        return "\t" + d.DefineClause.String()
}

func (d *RuleClause) String() string {
        var targets []string
        for _, t := range d.Targets {
                targets = append(targets, fmt.Sprintf("%v", t))
        }
        return strings.Join(targets, " ")
}

func (*BadClause) clauseNode()     {}
func (*GenericClause) clauseNode() {}
func (*DefineClause) clauseNode()  {}
func (*RuleClause) clauseNode()    {}

func (*RecipeDefineClause) expr() {}
func (*RecipeRuleClause) expr() {}
func (*IncludeRuleClause) expr() {}

// A File node represents a Smart source file.
//
// The Comments list contains all comments in the source file in order of
// appearance, including the comments that are pointed to from other nodes
// via Doc and Comment fields.
//
type File struct {
	Doc        *CommentGroup   // associated documentation; or nil
	KeyPos     token.Pos       // position of "project", "package" or "module" keyword
        Keyword    token.Token     // e.g. "project", "package", "module"
	Name       *Bareword       // project/module name
	Scope      Scope           // module scope (this file only)
	Clauses    []Clause        // top-level declarations; or nil
	Imports    []*UseSpec      // imports in this file
	//Unresolved []*Bareword     // unresolved identifiers in this file
	Comments   []*CommentGroup // list of all comments in the source file
}

// A Project node represents a set of source files
// collectively building a Project.
//
type Project struct {
	KeyPos  token.Pos         // position of "project" keyword
	Name    string            // project name
	Scope   Scope             // project scope across all files
	Imports map[string]Symbol // map of project id -> project symbol
	Files   map[string]*File  // source files by filename
        Runtime interface{}
}

func (p *Project) Pos() token.Pos { return p.KeyPos }
func (_ *Project) End() token.Pos { return token.NoPos }
