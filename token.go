//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	g "go/token"
	"strconv"
	"fmt"
)

type token int

// https://unicode-table.com/en/sets/arrows-symbols/
// 🕐🕑🕒🕓🕔🕕🕖🕗🕘🕙🕚🕛🕜🕝🕞🕟🕠🕡🕢🕣🕤🕥🕦🕧
// ┌────────────────────────────────┐
// ├────────────────────────────────┼ ⇔ ⇒
// ├────┬─────────────────┬─────────┤   ↑    ⇡
// ├┬───┼─────────────────┼─────────┘  ←·→  ⇠·⇢
// │├┬──┴─                └──┬──┐       ↓    ⇣
// ││└──              ─┐     │  ├─
// │└─────┬──   ────┬──┴──┬──┘  │      ⇤…⇥
// └──             ─┘           └─

// https://en.wikipedia.org/wiki/Mathematical_operators_and_symbols_in_Unicode
const (
	// Special tokens.
	ILLEGAL = token(iota)

	EOF      // end of file
	SPACE    // [ ]
	COMMENT  // #
	HASH     // # (same char as COMMENT, but different meaning)

	_literal_beg
	// Identifiers and basic type literals (these tokens stand for classes of literals)
	WORD
	BINARY   // 0b010101, 0B0111001
	OCTAL    // 0600, 0567
	INTEGER  // 12345
	HEXADECIMAL // 0x1234567890ABCDEF
	FLOATING    // 123.45
	DATETIME // 1979-05-27T07:32:00.999999-07:00 (internet date/time format - RFC3339)
	DATE     // 1979-05-27 (internet date format - RFC3339)
	TIME     // 07:32:00.999999 (internet time format - RFC3339)
	URL      // 'mailto:name@example.com' (uniform resource identifier - RFC3986)
	RAW      // raw strings
	ESCAPE   // \", \\n, etc. (see value.EscapeChar)
	STRING   // 'abc'
	STRVAL   // {abc}
	STRCOMP  // "abc $(foo) 123"
	_literal_end

	COMPOSED // the ending quote of a strcomp literal
	RECIPE   // tab to indicate a command recipe
	LINEND   // significannot line break (LF or CRLF)

	PROOT    // the root of a path, aka "" before the first '/' in a path
	PTAIL    // the tail of a path, aka "" after the last '/' in a path

	_operator_beg
	LANGLE    // <
	LBRACE    // {    left curly
	LBRACK    // [
	LPAREN    // (
	Lchevron    // ⟨ ⟪⟫ ｟｠ 〝〞
	Ltop_corner // ⌜
	Lbot_corner // ⌞
	Lsing_guil  // ‹
	Lguillemet  // «
	Rguillemet  // »
	Rsing_guil  // ›
	Rbot_corner // ⌟
	Rtop_corner // ⌝
	Rchevron    // ⟩
	RPAREN    // )
	RBRACK    // ]
	RBRACE    // }    right curly
	RANGLE    // >

	CARET     // ^ ˆ‸
	COMMA     // ,
	DOT       // .    period
	DOTDOT    // ..
	TILDE     // ~

	SELECT_PROP  // -> 'foo→xxx' (different from ' → ')
	SELECT_PROG1 // => 'foo⇒xxx' ('foo↦xxx' 'foo↣xxx' 'foo⇥xxx')
	SELECT_PROG2 // ~> 'foo⇢xxx' ('foo↦xxx' 'foo↣xxx' 'foo⇥xxx')
	// ⤌ ⤍	⤎ ⤏	⤐	⤑

	SEMICOLON // ;

	EXC       // !    exclamation
	QUE       // ?

	AT        // @
	SAST      // *    Single Asterisk
	DAST      // **   Double Asterisk
	ASTQ      // *?   Asterisk Que
	UNDERLINE // _

	CLOSURE   // &
	DELEGATE  // $

	MINUS // unary -
	PLUS  // unary +
	PCON  // path concatenation '/'
	PERC  // percent sign '%'(REM)

	_ruledelim_beg
	BAR       // |
	COLON     // :
	DOLON     // ::
	SOLON     // ;:
	_ruledelim_end

	// ⩵ ⩶
	_assign_beg
	ASSIGN     //   =       define a new symbol (don't override, neither !=)
	ASSIGN_SHI //   =+      shift (insert to the front)
	ASSIGN_ADD //  +=       append
	ASSIGN_QUE //  ?=       set if absent (defined, including empty)
	ASSIGN_EXC //  !=       execute a shell script and set a variable to its output (.SHELLSTATUS)
	// TODO: more assigns like !?=  !:=  !+=
	ASSIGN_CO1 //  := ≔     delegate-expanded (also override)
	ASSIGN_CO2 // ::= ⩴    all-expanded (POSIX standard)
	ASSIGN_CO3 // ;:=       all and unexpanded-force
	ASSIGN_SC1 //  ;=       unexpanded-force
	ASSIGN_SUB //  -=       remove
	ASSIGN_SAD // -+=       remove-append assign
	ASSIGN_SSH //  -=+      remove-shift assign
	_assign_end
	_operator_end

	_keyword_beg
	PROJECT    // project a
	CONFIGURE  // configure [...] TODO: use a different keyword
	USE        // use b
	ASSERT     // assert clause
	APPEND     // append values
	LOCAL      // declare local def names
	EVAL       // evaluate a builtin immediately
	EXPORT     // export ...
	INCLUDE    // include a.smart
	INSTANCE   // instance
	FILES      // files
	TEMPLATE   // template
	AND        // and
	OR         // or
	FOR        // for
	FOREACH    // foreach
	DONE       // done
	DEF        // def
	END        // end

	_constant_beg
	UNDEF   // `undef`
	NULL    // `null`
	NONE    // `none`
	BARE    // `bare`  // TODO
	PATH    // `path`  // TODO
	GLOB    // `glob`  // TODO
	REGEX   // `regex` // TODO
	FILE    // `file`  // TODO
	BIN     // `bin`
	OCT     // `oct`
	INT     // `int`
	HEX     // `hex`
	FLOAT   // `float`
	ANSWER  // `answer`
	BOOL    // `bool`
	BOOLEAN // `boolean`
	TRUE    // boolean `true`
	FALSE   // boolean `false`
	YES     // answer `yes`
	NO      // answer `no`
	ON      // option `on`
	OFF     // option `off`
	_constant_end
	_keyword_end = _constant_end
)

var tokens = [...]string{
	ILLEGAL: "ILLEGAL",

	EOF:     "EOF",
	SPACE:   "SPACE",
	COMMENT: "COMMENT",
	HASH:    "HASH",

	WORD:     "WORD",
	BINARY:   "BINARY",
	OCTAL:    "OCTAL",
	INTEGER:  "INTEGER",
	HEXADECIMAL: "HEXADECIMAL",
	FLOATING:    "FLOATING",
	DATETIME: "DATETIME",
	DATE:     "DATE",
	TIME:     "TIME",
	URL:      "URL",
	RAW:      "RAW",
	STRING:   "STRING",
	STRVAL:   "STRVAL",
	STRCOMP:  "STRCOMP",

	COMPOSED: "COMPOSED",
	RECIPE:   "RECIPE",
	ESCAPE:   "\\",
	LINEND:   "\\n", //"LINEND",
	PROOT:    "", // the "" before the first '/' in a path
	PTAIL:    "", // the "" after the last '/' in a path

	LANGLE: "<",
	LBRACE: "{",
	LBRACK: "[",
	LPAREN: "(",
	Lchevron: "⟨",
	Ltop_corner: "⌜",
	Lbot_corner: "⌞",
	Lsing_guil: "‹",
	Lguillemet: "«",
	Rguillemet: "»",
	Rsing_guil: "›",
	Rbot_corner: "⌟",
	Rtop_corner: "⌝",
	Rchevron: "⟩",
	RPAREN: ")",
	RBRACK: "]",
	RBRACE: "}",
	RANGLE: ">",

	CARET:  "^",
	COMMA:  ",",
	DOT:    ".",
	DOTDOT: "..",
	TILDE:  "~",

	SELECT_PROP:  "→", // foo->bar
	SELECT_PROG1: "⇒", // foo=>bar foo⇒bar
	SELECT_PROG2: "⇢", // foo~>bar foo⇢bar

	SEMICOLON: ";",

	EXC:       "!",
	QUE:       "?",

	BAR:       "|",
	COLON:     ":",
	DOLON:     "::",
	SOLON:     ";:",

	AT:        "@",
	SAST:      "*",
	DAST:      "**",
	ASTQ:      "*?",
	UNDERLINE: "_",

	CLOSURE:   "&",
	DELEGATE:  "$",

	ASSIGN:     "=",
	ASSIGN_SHI: "=+",
	ASSIGN_ADD: "+=",
	ASSIGN_QUE: "?=",
	ASSIGN_EXC: "!=",
	ASSIGN_CO1: ":=",
	ASSIGN_CO2: "::=",
	ASSIGN_CO3: ";:=",

	ASSIGN_SC1: ";=",
	ASSIGN_SUB: "-=",
	ASSIGN_SAD: "-+=",
	ASSIGN_SSH: "-=+",

	MINUS: "-",
	PLUS:  "+",
	PCON:  "/",
	PERC:  "%",

	PROJECT:   "project",
	CONFIGURE: "configure",
	USE:       "use",
	ASSERT:    "assert",
	APPEND:    "append",
	LOCAL:     "local",
	EVAL:      "eval",
	EXPORT:    "export",
	INCLUDE:   "include",
	INSTANCE:  "instance",
	FILES:     "files",
	TEMPLATE:  "template",
	AND:       "and",
	OR:        "or",
	FOR:       "for",
	FOREACH:   "foreach",
	DONE:      "done",
	DEF:       "def",
	END:       "end",

	UNDEF:  "undef",
	NULL:   "null",
	NONE:   "none",
	BARE:   "bare",
	PATH:   "path",
	GLOB:   "glob",
	REGEX:  "regex",
	FILE:   "file",
	BIN:    "bin",
	OCT:    "oct",
	INT:    "int",
	HEX:    "hex",
	FLOAT:  "float",
	ANSWER: "answer",
	BOOL:   "bool",
	BOOLEAN:"boolean",
	TRUE:   "true",
	FALSE:  "false",
	YES:    "yes",
	NO:     "no",
	ON:     "on",
	OFF:    "off",
}

func (tok token) String() (s string) {
	if 0 <= tok && tok < token(len(tokens)) { s = tokens[tok] }
	if s == "" {
		switch tok {
		case PROOT, PTAIL: return
		default:
			return "token(" + strconv.Itoa(int(tok)) + ")"
		}
	}
	return
}

var keywords = make(map[string]token)

func init() {
	for i := _keyword_beg + 1; i < _keyword_end; i++ {
		if s := tokens[i]; s != "" { keywords[s] = i }
	}
}

// lookup_keyword maps an identifier to its keyword token or IDENT (if not a keyword).
//
func lookup_keyword(ident string) token {
	if t, y := keywords[ident]; y { return t }
	return WORD
}

func (tok token) is_literal() bool          { return   _literal_beg < tok && tok <   _literal_end }
func (tok token) is_operator() bool         { return  _operator_beg < tok && tok <  _operator_end }
func (tok token) is_keyword() bool          { return   _keyword_beg < tok && tok <   _keyword_end }
func (tok token) is_constant() bool         { return  _constant_beg < tok && tok <  _constant_end }
func (tok token) is_closure() bool          { return  CLOSURE == tok }
func (tok token) is_closure_delegate() bool { return  CLOSURE == tok || tok == DELEGATE }
func (tok token) is_delegate() bool         { return  DELEGATE == tok }
func (tok token) is_assign() bool           { return    _assign_beg < tok && tok <    _assign_end }
func (tok token) is_rule_delim() bool       { return _ruledelim_beg < tok && tok < _ruledelim_end }
func (tok token) is_list_delim() bool {
	switch tok {
	case RPAREN, RBRACK, RBRACE, Rbot_corner, Rtop_corner, Rsing_guil, Rguillemet, Rchevron, SEMICOLON, COMMA, LINEND, EOF:
		return true
	}
	return tok.is_rule_delim()
}

type line_column_s string
func line_column(a any) string {
	if a == nil { return "0:0" }
	switch t := a.(type) {
	case Position:
		return fmt.Sprintf("%d:%d", t.Line, t.Column)
	case positioner:
		var p = t.Position()
		return fmt.Sprintf("%d:%d", p.Line, p.Column)
	case Context:
		var p = _position(t)
		return fmt.Sprintf("%d:%d", p.Line, p.Column)
	case []Value:
		if len(t) == 0 { return "0:0" }
		return line_column(t[0])
	}
	panic(failureUnreachable(fmt.Sprint(a))) // unreachable(a)
}

/*
  Struct Position:
	Filename string  -- filename, if any
	Offset   int     -- offset, starting at 0
	Line     int     -- line number, starting at 1
	Column   int     -- column number, starting at 1 (byte count)
*/
type Position struct { g.Position }
func (p *Position) valid() bool { return p.Filename != "" && p.Line > 0 }
func (p *Position) same(o *Position) bool {
	return p == o ||
		p.Filename == o.Filename && p.Line == o.Line &&
		p.Column == o.Column && p.Offset == o.Offset
}
func (p *Position) sameLoc(o *Position) bool {
	return p == o ||
		p.Filename == o.Filename && p.Line == o.Line &&
		p.Column == o.Column
}
func (p *Position) sameLine(o *Position) bool {
	return p == o || (p.Filename == o.Filename && p.Line == o.Line)
}

func atoi(a any) (res int) {
	switch t := a.(type) {
	case string: res, _ = strconv.Atoi(t)
	case []byte: res, _ = strconv.Atoi(string(t))
	}
	return
}

const NoPos Pos = Pos(g.NoPos)

type Pos g.Pos
func (p Pos) IsValid() bool { return g.Pos(p).IsValid() }

type tokfile struct { *g.File }
func (f *tokfile) string() string { return f.Name() }
func (f *tokfile) Offset(p Pos) int { return f.File.Offset(g.Pos(p)) }
func (f *tokfile) Line(p Pos) int { return f.File.Line(g.Pos(p)) }
func (f *tokfile) Pos(offset int) Pos { return Pos(f.File.Pos(offset)) }
func (f *tokfile) Position(p Pos) Position { return Position{f.File.PositionFor(g.Pos(p), true)} }

type fileset struct { *g.FileSet }
func _fileset() *fileset { return &fileset{ g.NewFileSet() } }
func (s *fileset) AddFile(filename string, base, size int) *tokfile {
	return &tokfile{ s.FileSet.AddFile(filename, base, size) }
}
func (s *fileset) Iterate(f func(*tokfile) bool) {
	s.FileSet.Iterate(func(a *g.File) bool { return f(&tokfile{a}) })
}
