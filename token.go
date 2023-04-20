//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"strconv"
)

type Token int

// https://en.wikipedia.org/wiki/Mathematical_operators_and_symbols_in_Unicode
const (
	// Special tokens.
	ILLEGAL Token = iota
	EOF
	SPACE
	COMMENT  // #
	HASH     // # (same char as COMMENT, but different meaning)

	literal_beg
	// Identifiers and basic type literals (these tokens stand for classes of literals)
	BAREWORD // abc
	BINARY   // 0b010101, 0B0111001
	OCTAL    // 0600, 0567
	INTEGER  // 12345
	HEXADECIMAL // 0x1234567890ABCDEF
	FLOATING    // 123.45
	DATETIME // 1979-05-27T07:32:00.999999-07:00 (internet date/time format - RFC3339)
	DATE     // 1979-05-27 (internet date format - RFC3339)
	TIME     // 07:32:00.999999 (internet time format - RFC3339)
	URI      // 'mailto:duzy.chan@example.com' (uniform resource identifier - RFC3986)
	RAW      // raw strings
	STRING   // 'abc'
	ESCAPE   // \", \\n, etc. (see value.EscapeChar)
	COMPOUND // "abc $(foo) 123"
	literal_end

	COMPOSED // the ending quote of a compound literal
	RECIPE   // tab to indicate a command recipe
	LINEND   // significant line break (LF or CRLF)

	operator_beg
	CARET     // ^
	LANGLE    // <
	LBRACE    // {    left curly
	LBRACK    // [
	LPAREN    // (
	COMMA     // ,
	DOT       // .    period
	DOTDOT    // ..
	TILDE     // ~
	SELECT_PROP // -> 'foo→xxx' (different from ' → ')
	SELECT_PROG1 // => 'foo⇒xxx' ('foo↦xxx' 'foo↣xxx' 'foo⇥xxx')
	SELECT_PROG2 // ~> 'foo⇢xxx' ('foo↦xxx' 'foo↣xxx' 'foo⇥xxx')
	// ⤌ ⤍	⤎ ⤏	⤐	⤑

	RPAREN    // )
	RBRACK    // ]
	RBRACE    // }    right curly
	RANGLE    // >

	SEMICOLON // ;

	EXC       // !    exclamation
	QUE       // ?

	ruledelim_beg
	BAR       // |
	COLON     // :
	COLON2    // ::
	ruledelim_end

	AT        // @
	STAR      // *

	// NOTE: don't change the order of closures and delegates, scanner
	// relys upon their order.
	closure_beg
	CLOSURE      // &
	CLOSURE_r    // &/
	CLOSURE_D    // &.
	CLOSURE_A    // &@
	CLOSURE_B    // &|
	CLOSURE_L    // &<
	CLOSURE_R    // &>
	CLOSURE_U    // &^
	CLOSURE_S    // &*
	CLOSURE_M    // &-
	CLOSURE_P    // &+
	CLOSURE_Q    // &?
	CLOSURE_0    // &0
	CLOSURE_1    // &1
	CLOSURE_2    // &2
	CLOSURE_3    // &3
	CLOSURE_4    // &4
	CLOSURE_5    // &5
	CLOSURE_6    // &6
	CLOSURE_7    // &7
	CLOSURE_8    // &8
	CLOSURE_9    // &9
	CLOSURE__    // &_
	closure_end

	delegate_beg
	DELEGATE      // $
	DELEGATE_r    // $/
	DELEGATE_D    // $.
	DELEGATE_A    // $@
	DELEGATE_B    // $|
	DELEGATE_L    // $<
	DELEGATE_R    // $>
	DELEGATE_U    // $^
	DELEGATE_S    // $*
	DELEGATE_M    // $-
	DELEGATE_P    // $+
	DELEGATE_Q    // $?
	DELEGATE_0    // $0
	DELEGATE_1    // $1
	DELEGATE_2    // $2
	DELEGATE_3    // $3
	DELEGATE_4    // $4
	DELEGATE_5    // $5
	DELEGATE_6    // $6
	DELEGATE_7    // $7
	DELEGATE_8    // $8
	DELEGATE_9    // $9
	DELEGATE__    // $_
	delegate_end

	assign_beg
	ASSIGN     //   =       define a new symbol (don't override, neither !=)
	SHI_ASSIGN //   =+      shift (insert to the front)
	ADD_ASSIGN //  +=       append
	QUE_ASSIGN //  ?=       set if absent (defined, including empty)
	EXC_ASSIGN //  !=       execute a shell script and set a variable to its output (.SHELLSTATUS)
	// TODO: more assigns like !?=  !:=  !+=
	SCO_ASSIGN //  := ≔     simply expanded (also override)
	DCO_ASSIGN // ::= ⩴    simply expanded (POSIX standard)
	SUB_ASSIGN //  -=       remove
	SAD_ASSIGN // -+=       remove-append assign
	SSH_ASSIGN //  -=+      remove-shift assign
	assign_end

	PLUS  // unary +
	MINUS // unary -
	PCON  // path concatenation '/'
	PERC  // percent sign '%'(REM)
	operator_end

	keyword_beg
	PROJECT    // project a
	PACKAGE    // package a
	MODULE     // module a
	CONFIGURE  // configure [...] TODO: use a different keyword
	USE        // use b
	ASSERT     // assert clause
	APPEND     // append values
	EVAL       // evaluate a builtin immediately
	EXPORT     // export ...
	INCLUDE    // include a.smart
	INSTANCE   // instance
	FILES      // files
	TEMPLATE   // template
	FOREACH    // foreach
	DONE       // done

	constant_beg
	UNDEF   // `undef`
	NONE    // `none`
	BARE    // `bare`  // TODO
	REGEX   // `regex` // TODO
	FILE    // `file`  // TODO
	BIN     // `bin`
	OCT     // `oct`
	INT     // `int`
	HEX     // `hex`
	FLOAT   // `float`
	ANSWER  // `answer`
	BOOL    // `bool`
	TRUE    // boolean `true`
	FALSE   // boolean `false`
	YES     // answer `yes`
	NO      // answer `no`
	constant_end
	keyword_end = constant_end
)

var tokens = [...]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",
	SPACE:   "SPACE",
	COMMENT: "COMMENT",
	HASH:    "HASH",

	BAREWORD: "BAREWORD",
	BINARY:   "BINARY",
	OCTAL:    "OCTAL",
	INTEGER:  "INTEGER",
	HEXADECIMAL: "HEXADECIMAL",
	FLOATING:    "FLOATING",
	DATETIME: "DATETIME",
	DATE:     "DATE",
	TIME:     "TIME",
	URI:      "URI",
	RAW:      "RAW",
	STRING:   "STRING",
	ESCAPE:   "\\",
	COMPOUND: "COMPOUND",

	COMPOSED: "COMPOSED",
	RECIPE:   "RECIPE",
	LINEND:   "\\n", //"LINEND",

	CARET: "^",
	LANGLE: "<",
	LBRACE: "{",
	LBRACK: "[",
	LPAREN: "(",
	COMMA:  ",",
	DOT:    ".",
	DOTDOT: "..",
	TILDE:  "~",
	SELECT_PROP: "→", // foo->bar
	SELECT_PROG1: "⇒", // foo=>bar foo⇒bar
	SELECT_PROG2: "⇢", // foo~>bar foo⇢bar

	RPAREN:    ")",
	RBRACK:    "]",
	RBRACE:    "}",
	RANGLE:    ">",

	SEMICOLON: ";",

	EXC:       "!",
	QUE:       "?",

	BAR:       "|",
	COLON:     ":",
	COLON2:    "::",

	AT:        "@",
	STAR:      "*",

	CLOSURE:      "&",
	CLOSURE_r:    "&/",
	CLOSURE_D:    "&.",
	CLOSURE_A:    "&@",
	CLOSURE_B:    "&|",
	CLOSURE_L:    "&<",
	CLOSURE_R:    "&>",
	CLOSURE_U:    "&^",
	CLOSURE_S:    "&*",
	CLOSURE_M:    "&-",
	CLOSURE_P:    "&+",
	CLOSURE_Q:    "&Q",
	CLOSURE_0:    "&0",
	CLOSURE_1:    "&1",
	CLOSURE_2:    "&2",
	CLOSURE_3:    "&3",
	CLOSURE_4:    "&4",
	CLOSURE_5:    "&5",
	CLOSURE_6:    "&6",
	CLOSURE_7:    "&7",
	CLOSURE_8:    "&8",
	CLOSURE_9:    "&9",
	CLOSURE__:    "&_",

	DELEGATE:      "$",
	DELEGATE_r:    "$/",
	DELEGATE_D:    "$.",
	DELEGATE_A:    "$@",
	DELEGATE_B:    "$|",
	DELEGATE_L:    "$<",
	DELEGATE_R:    "$>",
	DELEGATE_U:    "$^",
	DELEGATE_S:    "$*",
	DELEGATE_M:    "$-",
	DELEGATE_P:    "$+",
	DELEGATE_Q:    "$?",
	DELEGATE_0:    "$0",
	DELEGATE_1:    "$1",
	DELEGATE_2:    "$2",
	DELEGATE_3:    "$3",
	DELEGATE_4:    "$4",
	DELEGATE_5:    "$5",
	DELEGATE_6:    "$6",
	DELEGATE_7:    "$7",
	DELEGATE_8:    "$8",
	DELEGATE_9:    "$9",
	DELEGATE__:    "$_",

	ASSIGN:     "=",
	SHI_ASSIGN: "=+",
	ADD_ASSIGN: "+=",
	QUE_ASSIGN: "?=",
	EXC_ASSIGN: "!=",
	SCO_ASSIGN: ":=",
	DCO_ASSIGN: "::=",
	SUB_ASSIGN: "-=",
	SAD_ASSIGN: "-+=",
	SSH_ASSIGN: "-=+",

	PLUS:  "+",
	MINUS: "-",
	PCON:  "/",
	PERC:  "%",

	PROJECT:    "project",
	PACKAGE:    "package",
	MODULE:     "module",
	CONFIGURE:  "configure",
	USE:        "use",
	ASSERT:     "assert",
	APPEND:     "append",
	EVAL:       "eval",
	EXPORT:     "export",
	INCLUDE:    "include",
	INSTANCE:   "instance",
	FILES:      "files",
	TEMPLATE:   "template",
	FOREACH:    "foreach",
	DONE:       "done",

	UNDEF:  "undef",
	NONE:   "none",
	BARE:   "bare",
	REGEX:  "regex",
	FILE:   "file",
	BIN:    "bin",
	OCT:    "oct",
	INT:    "int",
	HEX:    "hex",
	FLOAT:  "float",
	ANSWER: "answer",
	BOOL:   "bool",
	TRUE:   "true",
	FALSE:  "false",
	YES:    "yes",
	NO:     "no",
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

var keywords = make(map[string]Token)

func init() {
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
func (tok Token) IsConstant() bool { return constant_beg < tok && tok < constant_end }
func (tok Token) IsClosure() bool { return closure_beg < tok && tok < closure_end }
func (tok Token) IsDelegate() bool { return delegate_beg < tok && tok < delegate_end }
func (tok Token) IsAssign() bool { return assign_beg < tok && tok < assign_end }
func (tok Token) IsRuleDelim() bool { return ruledelim_beg < tok && tok < ruledelim_end }
func (tok Token) IsSelectProg() bool { return SELECT_PROG1 == tok || tok == SELECT_PROG2 }
func (tok Token) IsSelectProp() bool { return SELECT_PROP == tok }
func (tok Token) IsListDelim() bool {
	return tok.IsRuleDelim() ||
		tok == RPAREN || tok == RBRACK || tok == RBRACE ||
		tok == SEMICOLON || tok == COMMA || tok == LINEND ||
		tok == EOF
}
