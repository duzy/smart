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
        IDENT  // abc
	INT    // 12345
	FLOAT  // 123.45
        DATETIME // 1979-05-27T07:32:00.999999-07:00 (internet date/time format - RFC3339)
        DATE     // 1979-05-27 (internet date format - RFC3339)
        TIME     // 07:32:00.999999 (internet time format - RFC3339)
        URI      // 'mailto:Duzy.Chan@example.com' (uniform resource identifier - RFC3986)
	STRING   // "abc" 'abc'
        RECIEPT  // tab to indicate a command reciept
	literal_end
        
	operator_beg
        LINEND  // significant line break (LF or CRLF)

	LPAREN  // (
	LBRACK  // [
	LBRACE  // {    left curly
	COMMA   // ,
	PERIOD  // .

	RPAREN  // )
	RBRACK  // ]
	RBRACE  // }    right curly
	SEMICOLON // ;

	COLON     // :
	COLON2    // ::
	COLON_EXC // :!:
	COLON_QUE // :?:
        COLON_LBK // :[
        COLON_LBE // :![
        COLON_LBQ // :?[
        COLON_RBK // ]:
        CALL    // $
        
        ASSIGN     // =
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

	PLUS  // +
	MINUS // -
	PCON  // path concatenation /
	operator_end

	keyword_beg
        PROJECT    // project a
        MODULE     // module a
        USE        // use b
        EXPORT
        INCLUDE    // include a.smart
        IMPORT     // import a.smart
        INSTANCE   // instance
	keyword_end
)

var tokens = [...]string{
        ILLEGAL: "ILLEGAL",
        EOF:     "EOF",
        COMMENT: "COMMENT",

        IDENT:    "IDENT",
        INT:      "INT",
        FLOAT:    "FLOAT",
        DATETIME: "DATETIME",
        DATE:     "DATE",
        TIME:     "TIME",
        URI:      "URI",
        STRING:   "STRING",

        LINEND:  "LINEND",
        RECIEPT: "RECIEPT",

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
        COLON_EXC: ":!:",
        COLON_QUE: ":?:",
        COLON_LBK: ":[",
        COLON_LBE: ":![",
        COLON_LBQ: ":?[",
        COLON_RBK: "]:",
        
	CALL: "$",
        
        ASSIGN:     "=",
        ADD_ASSIGN: "+=",

        PLUS: "+",
        MINUS: "-",
	PCON: "/",
        
        PROJECT:  "project",
        MODULE:   "module",
        USE:      "use",
        EXPORT:   "export",
        INCLUDE:  "include",
        IMPORT:   "import",
        INSTANCE: "instance",
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
	return IDENT
}

func (tok Token) IsLiteral() bool { return literal_beg < tok && tok < literal_end }
func (tok Token) IsOperator() bool { return operator_beg < tok && tok < operator_end }
func (tok Token) IsKeyword() bool { return keyword_beg < tok && tok < keyword_end }
