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
	STRING // "abc"
        SHSTR  // `echo shell command`
	literal_end
        
	operator_beg
        SEP     // spaces (or \\\n) used as an element separator
        LINEND  // significant line break (LF or CRLF)
        RECIEPT // tab to indicate a command reciept

        CONCAT  // concatenation

	LPAREN  // (
	LBRACK  // [
	LBRACE  // {    left curly
	COMMA   // ,
	PERIOD  // .

	RPAREN  // )
	RBRACK  // ]
	RBRACE  // }    right curly
	COLON   // :

        CALL    // $
        
        ASSIGN     // =
        /*
        QUE_ASSIGN // ?=
        SCO_ASSIGN // :=
        DCO_ASSIGN // ::=
        EXC_ASSIGN // !=     exclamation */
        ADD_ASSIGN // +=

	ADD // +
	SUB // -
	MUL // *
	QUO // /
	REM // %
	operator_end

	keyword_beg
        PROJECT    // project a
        MODULE     // module a
        USE        // use b
        EXPORT
        INCLUDE    // include a.smart
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
        STRING:   "STRING",

        SEP:     "SEP",
        LINEND:  "LINEND",
        RECIEPT: "RECIEPT",

        CONCAT: "CONCAT",

	LPAREN: "(",
	LBRACK: "[",
	LBRACE: "{",
	COMMA:  ",",
	PERIOD: ".",

	RPAREN:    ")",
	RBRACK:    "]",
	RBRACE:    "}",
	COLON:     ":",
        
	CALL: "$",
        
        ASSIGN:     "=",
        ADD_ASSIGN: "+=",

	ADD: "+",
	SUB: "-",
	MUL: "*",
	QUO: "/",
	REM: "%",
        
        PROJECT: "project",
        MODULE:  "module",
        USE:     "use",
        EXPORT:  "export",
        INCLUDE: "include",
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
