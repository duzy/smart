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
	COMMENT  // #
        
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
	LPAREN    // (
	LBRACK    // [
	LBRACE    // {    left curly
	COMMA     // ,
	PERIOD    // .
        DOTDOT    // ..
	TILDE     // ~
        SELECT_PROP // ->
        SELECT_PROG // =>

	RPAREN    // )
	RBRACK    // ]
	RBRACE    // }    right curly
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
        CLOSURE_R    // &/
        CLOSURE_D    // &.
        CLOSURE_A    // &@
        CLOSURE_B    // &|
        CLOSURE_L    // &<
        CLOSURE_U    // &^
        CLOSURE_S    // &*
        CLOSURE_M    // &-
        CLOSURE_P    // &+
        CLOSURE_Q    // &?
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
        DELEGATE_R    // $/
        DELEGATE_D    // $.
        DELEGATE_A    // $@
        DELEGATE_B    // $|
        DELEGATE_L    // $<
        DELEGATE_U    // $^
        DELEGATE_S    // $*
        DELEGATE_M    // $-
        DELEGATE_P    // $+
        DELEGATE_Q    // $?
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
        SCO_ASSIGN //  :=       simply expanded (also override)
        DCO_ASSIGN // ::=       simply expanded (POSIX standard)
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
        CONFIGURATION
        USE        // use b
        EVAL       // evaluate a builtin immediately
        EXPORT     // export ...
        INCLUDE    // include a.smart
        IMPORT     // import a.smart
        INSTANCE   // instance
        FILES      // files

        constant_beg
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
        RAW:      "RAW",
        STRING:   "STRING",
        ESCAPE:   "\\",
        COMPOUND: "COMPOUND",

        COMPOSED: "COMPOSED",
        RECIPE:   "RECIPE",
        LINEND:   "\\n", //"LINEND",

	LPAREN: "(",
	LBRACK: "[",
	LBRACE: "{",
	COMMA:  ",",
	PERIOD: ".",
        DOTDOT: "..",
        TILDE:  "~",
        SELECT_PROP: "->", // foo->bar
        SELECT_PROG: "=>", // foo=>bar

	RPAREN:    ")",
	RBRACK:    "]",
	RBRACE:    "}",
        SEMICOLON: ";",

        EXC:       "!",
        QUE:       "?",

        BAR:       "|",
	COLON:     ":",
        COLON2:    "::",

        AT:        "@",
        STAR:      "*",

	CLOSURE:      "&",
        CLOSURE_R:    "&/",
        CLOSURE_D:    "&.",
        CLOSURE_A:    "&@",
        CLOSURE_B:    "&|",
        CLOSURE_L:    "&<",
        CLOSURE_U:    "&^",
        CLOSURE_S:    "&*",
        CLOSURE_M:    "&-",
        CLOSURE_P:    "&+",
        CLOSURE_Q:    "&Q",
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
        DELEGATE_R:    "$/",
        DELEGATE_D:    "$.",
        DELEGATE_A:    "$@",
        DELEGATE_B:    "$|",
        DELEGATE_L:    "$<",
        DELEGATE_U:    "$^",
        DELEGATE_S:    "$*",
        DELEGATE_M:    "$-",
        DELEGATE_P:    "$+",
        DELEGATE_Q:    "$?",
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
        CONFIGURATION: "configuration",
        USE:        "use",
        EVAL:       "eval",
        EXPORT:     "export",
        INCLUDE:    "include",
        IMPORT:     "import",
        INSTANCE:   "instance",
        FILES:      "files",

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
func (tok Token) IsListDelim() bool {
        return tok.IsRuleDelim() ||
               tok == RPAREN || tok == RBRACK || tok == RBRACE ||
               tok == SEMICOLON || tok == COMMA || tok == LINEND ||
               tok == EOF
}
