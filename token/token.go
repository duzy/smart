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
	INT      // 12345
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

	RPAREN    // )
	RBRACK    // ]
	RBRACE    // }    right curly
	SEMICOLON // ;

	COLON     // :
	COLON2    // ::
	EXC // !
	QUE // ?
        CALL      // $
        CALL_R    // $/
        CALL_D    // $.
        CALL_A    // $@
        CALL_L    // $<
        CALL_U    // $^
        CALL_S    // $*
        CALL_M    // $-
        CALL_1    // $1
        CALL_2    // $2
        CALL_3    // $3
        CALL_4    // $4
        CALL_5    // $5
        CALL_6    // $6
        CALL_7    // $7
        CALL_8    // $8
        CALL_9    // $9

       
        ASSIGN    // =
        /*
        QUE_ASSIGN // ?=
        SCO_ASSIGN // :=
        DCO_ASSIGN // ::=
        EXC_ASSIGN // !=     exclamation */
        ADD_ASSIGN // +=

        /*
	ADD // +
	SUB // -
	MUL // *
	QUO // /
	REM // % */

	PLUS  // unary +
	MINUS // unary -
	PCON  // path concatenation '/'
        PERC  // percent sign '%'
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
        EXTENSIONS // extensions
        FILES      // files
	keyword_end
)

var tokens = [...]string{
        ILLEGAL: "ILLEGAL",
        EOF:     "EOF",
        COMMENT: "COMMENT",

        BAREWORD: "BAREWORD",
        INT:      "INT",
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

	RPAREN:    ")",
	RBRACK:    "]",
	RBRACE:    "}",
        
	COLON:     ":",
        COLON2:    "::",
        EXC: "!",
        QUE: "?",
	CALL:      "$",
        CALL_R:    "$/",
        CALL_D:    "$.",
        CALL_A:    "$@",
        CALL_L:    "$<",
        CALL_U:    "$^",
        CALL_S:    "$*",
        CALL_M:    "$-",
        CALL_1:    "$1",
        CALL_2:    "$2",
        CALL_3:    "$3",
        CALL_4:    "$4",
        CALL_5:    "$5",
        CALL_6:    "$6",
        CALL_7:    "$7",
        CALL_8:    "$8",
        CALL_9:    "$9",

        ASSIGN:     "=",
        ADD_ASSIGN: "+=",

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
        EXTENSIONS: "extensions",
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
