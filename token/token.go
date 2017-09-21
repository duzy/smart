//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package token

import (
        "strconv"
)

type Token int

const (
        // Special tokens.
	ILLEGAL Token = iota
	EOF
	COMMENT
        
	literal_beg
	// Identifiers and basic type literals
	// (these tokens stand for classes of literals)
        BAREWORD // abc
        BIN      // 0b010101, 0B0111001
        OCT      // 0600, 0567
	INT      // 12345
        HEX      // 0x1234567890ABCDEF
	FLOAT    // 123.45
        DATETIME // 1979-05-27T07:32:00.999999-07:00 (internet date/time format - RFC3339)
        DATE     // 1979-05-27 (internet date format - RFC3339)
        TIME     // 07:32:00.999999 (internet time format - RFC3339)
        URI      // 'mailto:Duzy.Chan@example.com' (uniform resource identifier - RFC3986)
	STRING   // 'abc'
        ESCAPE   // \", \\n, etc.
        COMPOUND // "abc $(foo) 123"
	literal_end

        COMPOSED // the ending quote of a compound literal
        RECIPE   // tab to indicate a command recipe
        LINEND   // significant line break (LF or CRLF)

	operator_beg
	LPAREN    // (
	LBRACK    // [
	LBRACE    // {    left curly
	COMMA     // ,
	PERIOD    // .
        DOTDOT    // ..
        SELECT    // ->

	RPAREN    // )
	RBRACK    // ]
	RBRACE    // }    right curly
	SEMICOLON // ;

        ruledelim_beg
	COLON     // :
	COLON2    // ::
	EXC       // !          exclamation
	QUE       // ?
        ruledelim_end

        AT        // @
        STAR      // *

        // NOTE: don't change the order of closures and delegates, scanner
        // relys upon their order.
        closure_beg
        AND      // &
        AND_R    // &/
        AND_D    // &.
        AND_A    // &@
        AND_L    // &<
        AND_U    // &^
        AND_S    // &*
        AND_M    // &-
        AND_1    // &1
        AND_2    // &2
        AND_3    // &3
        AND_4    // &4
        AND_5    // &5
        AND_6    // &6
        AND_7    // &7
        AND_8    // &8
        AND_9    // &9
        closure_end
        delegate_beg
        DOLLAR      // $
        DOLLAR_R    // $/
        DOLLAR_D    // $.
        DOLLAR_A    // $@
        DOLLAR_L    // $<
        DOLLAR_U    // $^
        DOLLAR_S    // $*
        DOLLAR_M    // $-
        DOLLAR_1    // $1
        DOLLAR_2    // $2
        DOLLAR_3    // $3
        DOLLAR_4    // $4
        DOLLAR_5    // $5
        DOLLAR_6    // $6
        DOLLAR_7    // $7
        DOLLAR_8    // $8
        DOLLAR_9    // $9
        delegate_end

        assign_beg
        ASSIGN     //   =       define a new symbol (don't override, neither !=)
        ADD_ASSIGN //  +=       append
        QUE_ASSIGN //  ?=       set if absent (defined, including empty)
        EXC_ASSIGN //  !=       execute a shell script and set a variable to its output (.SHELLSTATUS)
        // TODO: more assigns like !?=  !:=  !+=
        SCO_ASSIGN //  :=       simply expanded (also override)
        DCO_ASSIGN // ::=       simply expanded (POSIX standard)
        assign_end
        
        ARROW // arrow =>

	PLUS  // unary +
	MINUS // unary -
	PCON  // path concatenation '/'
        PERC  // percent sign '%'(REM)
	operator_end

	keyword_beg
        PROJECT    // project a
        MODULE     // module a
        USE        // use b
        EVAL       // evaluate a builtin immediately
        EXPORT     // export ...
        INCLUDE    // include a.smart
        IMPORT     // import a.smart
        INSTANCE   // instance
        FILES      // files
	keyword_end
)

var tokens = [...]string{
        ILLEGAL: "ILLEGAL",
        EOF:     "EOF",
        COMMENT: "COMMENT",

        BAREWORD: "BAREWORD",
        BIN:      "BIN",
        OCT:      "OCT",
        INT:      "INT",
        HEX:      "HEX",
        FLOAT:    "FLOAT",
        DATETIME: "DATETIME",
        DATE:     "DATE",
        TIME:     "TIME",
        URI:      "URI",
        STRING:   "STRING",
        ESCAPE:   "\\",
        COMPOUND: "COMPOUND",
        COMPOSED: "COMPOSED",
        RECIPE:   "RECIPE",
        LINEND:   "LINEND",

	LPAREN: "(",
	LBRACK: "[",
	LBRACE: "{",
	COMMA:  ",",
	PERIOD: ".",
        DOTDOT: "..",
        SELECT: "->",

	RPAREN:    ")",
	RBRACK:    "]",
	RBRACE:    "}",

	COLON:     ":",
        COLON2:    "::",
        EXC:       "!",
        QUE:       "?",

        AT:        "@",
        STAR:      "*",

	AND:      "&",
        AND_R:    "&/",
        AND_D:    "&.",
        AND_A:    "&@",
        AND_L:    "&<",
        AND_U:    "&^",
        AND_S:    "&*",
        AND_M:    "&-",
        AND_1:    "&1",
        AND_2:    "&2",
        AND_3:    "&3",
        AND_4:    "&4",
        AND_5:    "&5",
        AND_6:    "&6",
        AND_7:    "&7",
        AND_8:    "&8",
        AND_9:    "&9",

	DOLLAR:      "$",
        DOLLAR_R:    "$/",
        DOLLAR_D:    "$.",
        DOLLAR_A:    "$@",
        DOLLAR_L:    "$<",
        DOLLAR_U:    "$^",
        DOLLAR_S:    "$*",
        DOLLAR_M:    "$-",
        DOLLAR_1:    "$1",
        DOLLAR_2:    "$2",
        DOLLAR_3:    "$3",
        DOLLAR_4:    "$4",
        DOLLAR_5:    "$5",
        DOLLAR_6:    "$6",
        DOLLAR_7:    "$7",
        DOLLAR_8:    "$8",
        DOLLAR_9:    "$9",

        ASSIGN:     "=",
        ADD_ASSIGN: "+=",
        QUE_ASSIGN: "?=",
        EXC_ASSIGN: "!=",
        SCO_ASSIGN: ":=",
        DCO_ASSIGN: "::=",

        ARROW:      "=>",

        PLUS:  "+",
        MINUS: "-",
	PCON:  "/",
	PERC:  "%",
        
        PROJECT:    "project",
        MODULE:     "module",
        USE:        "use",
        EVAL:       "eval",
        EXPORT:     "export",
        INCLUDE:    "include",
        IMPORT:     "import",
        INSTANCE:   "instance",
        FILES:      "files",
}

func (tok Token) String() (s string) {
	if 0 <= tok && tok < Token(len(tokens)) {
		s = tokens[tok]
	}
	if s == "" {
		s = "token(" + strconv.Itoa(int(tok)) + ")"
	}
	return
}

var keywords map[string]Token

func init() {
	keywords = make(map[string]Token)
	for i := keyword_beg + 1; i < keyword_end; i++ {
		keywords[tokens[i]] = i
	}
}

// Lookup maps an identifier to its keyword token or IDENT (if not a keyword).
//
func Lookup(ident string) Token {
	if tok, is_keyword := keywords[ident]; is_keyword {
		return tok
	}
	return BAREWORD
}

func (tok Token) IsLiteral() bool { return literal_beg < tok && tok < literal_end }
func (tok Token) IsOperator() bool { return operator_beg < tok && tok < operator_end }
func (tok Token) IsKeyword() bool { return keyword_beg < tok && tok < keyword_end }
func (tok Token) IsClosure() bool { return closure_beg < tok && tok < closure_end }
func (tok Token) IsDelegate() bool { return delegate_beg < tok && tok < delegate_end }
func (tok Token) IsAssign() bool { return assign_beg < tok && tok < assign_end }
func (tok Token) IsRuleDelim() bool { return ruledelim_beg < tok && tok < ruledelim_end }
