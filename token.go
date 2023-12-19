//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	got "go/token"
	"strconv"
)

type token int

const clocks = "🕐🕑🕒🕓🕔🕕🕖🕗🕘🕙🕚🕛🕜🕝🕞🕟🕠🕡🕢🕣🕤🕥🕦🕧" // ⇒

// https://en.wikipedia.org/wiki/Mathematical_operators_and_symbols_in_Unicode
const (
	// Special tokens.
	ILLEGAL = token(iota)
	EOF
	SPACE
	COMMENT  // #
	HASH     // # (same char as COMMENT, but different meaning)

	_literal_beg
	// Identifiers and basic type literals (these tokens stand for classes of literals)
	BAREWORD
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
	_literal_end

	COMPOSED // the ending quote of a compound literal
	RECIPE   // tab to indicate a command recipe
	LINEND   // significant line break (LF or CRLF)

	PROOT    // the root of a path, aka "" before the first '/' in a path
	PTAIL    // the tail of a path, aka "" after the last '/' in a path

	_operator_beg
	CARET     // ^
	LANGLE    // <
	LBRACE    // {    left curly
	LBRACK    // [
	LPAREN    // (
	COMMA     // ,
	DOT       // .    period
	DOTDOT    // ..
	TILDE     // ~
	SELECT_PROP  // -> 'foo→xxx' (different from ' → ')
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

	AT        // @
	STAR      // *    Single Asterisk
	DAST      // **   Double Asterisk

	PLUS  // unary +
	MINUS // unary -
	PCON  // path concatenation '/'
	PERC  // percent sign '%'(REM)

	_ruledelim_beg
	BAR       // |
	COLON     // :
	DOLON     // ::
	SOLON     // ;:
	_ruledelim_end

	// NOTE: don't change the order of closures and delegates, scanner
	// relys upon their order.
	_closure_beg
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
	_closure_end

	_delegate_beg
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
	_delegate_end

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
	AND        // and
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
	PROOT:    "", // the "" before the first '/' in a path
	PTAIL:    "", // the "" after the last '/' in a path

	CARET:  "^",
	LANGLE: "<",
	LBRACE: "{",
	LBRACK: "[",
	LPAREN: "(",
	COMMA:  ",",
	DOT:    ".",
	DOTDOT: "..",
	TILDE:  "~",
	SELECT_PROP:  "→", // foo->bar
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
	DOLON:     "::",
	SOLON:     ";:",

	AT:        "@",
	STAR:      "*",
	DAST:      "**",

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
	AND:        "and",
	FOR:        "for",
	FOREACH:    "foreach",
	DONE:       "done",
	DEF:        "def",
	END:        "end",

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
	if CLOSURE_r  != CLOSURE+1  { panic(CLOSURE_r) }
	if DELEGATE_r != DELEGATE+1 { panic(DELEGATE_r) }

	for i := _keyword_beg + 1; i < _keyword_end; i++ {
		if s := tokens[i]; s != "" { keywords[s] = i }
	}
}

// lookupKeyword maps an identifier to its keyword token or IDENT (if not a keyword).
//
func lookupKeyword(ident string) token {
	if t, y := keywords[ident]; y { return t }
	return BAREWORD
}

func (tok token) isLiteral() bool         { return _literal_beg   <  tok && tok <  _literal_end }
func (tok token) isOperator() bool        { return _operator_beg  <  tok && tok <  _operator_end }
func (tok token) isKeyword() bool         { return _keyword_beg   <  tok && tok <  _keyword_end }
func (tok token) isConstant() bool        { return _constant_beg  <  tok && tok <  _constant_end }
func (tok token) isClosure() bool         { return _closure_beg   <  tok && tok <  _closure_end }
func (tok token) isClosureDelegate() bool { return _closure_beg   <  tok && tok <  _delegate_end }
func (tok token) isDelegate() bool        { return _delegate_beg  <  tok && tok <  _delegate_end }
func (tok token) isAssign() bool          { return _assign_beg    <  tok && tok <  _assign_end }
func (tok token) isRuleDelim() bool       { return _ruledelim_beg <  tok && tok <  _ruledelim_end }
func (tok token) isSelectProg() bool      { return SELECT_PROG1   == tok || tok == SELECT_PROG2 }
func (tok token) isSelectProp() bool      { return SELECT_PROP    == tok }
func (tok token) isListDelim() bool {
	switch tok {
	case RPAREN, RBRACK, RBRACE, SEMICOLON, COMMA, LINEND, EOF:
		return true
	}
	return tok.isRuleDelim()
}

/*
  Struct Position:
	Filename string  -- filename, if any
	Offset   int     -- offset, starting at 0
	Line     int     -- line number, starting at 1
	Column   int     -- column number, starting at 1 (byte count)
*/
type Position struct { got.Position }
func (p *Position) _valid() bool { return p.Filename != "" && p.Line > 0 }
func (p *Position) IsValid() bool { return p._valid() && p.Column > 0 && p.Offset >= 0 }
func (p *Position) SameLine(o *Position) bool {
	return p == o || (p.Filename == o.Filename && p.Line == o.Line)
}
func (p *Position) Same(o *Position) bool {
	return p == o ||
		p.Filename == o.Filename && p.Line == o.Line &&
		p.Column == o.Column && p.Offset == o.Offset
}

func makePosition(filename string, line, column int) (pos Position) {
	pos.Filename = filename
	pos.Line     = line
	pos.Column   = column
	return
}

func convPosition(filename, line, column string) (pos Position) {
	pos.Filename  = filename
	pos.Line, _   = strconv.Atoi(line)
	pos.Column, _ = strconv.Atoi(column)
	return
}

const NoPos Pos = Pos(got.NoPos)

type Pos got.Pos

func (p Pos) IsValid() bool {
	return got.Pos(p).IsValid()
}

type TokFile struct {
	*got.File
}

func (f *TokFile) string() string {
	return f.Name() //fmt.Sprintf("{%s}", f.Name())
}

func (f *TokFile) Offset(p Pos) int {
	return f.File.Offset(got.Pos(p))
}

func (f *TokFile) Line(p Pos) int {
	return f.File.Line(got.Pos(p))
}

func (f *TokFile) Pos(offset int) Pos {
	return Pos(f.File.Pos(offset))
}

func (f *TokFile) PositionFor(p Pos, adjusted bool) (pos Position) {
	return Position{ f.File.PositionFor(got.Pos(p), adjusted) }
}

func (f *TokFile) Position(p Pos) (pos Position) {
	return Position{ f.File.Position(got.Pos(p)) }
}

type FileSet struct {
	*got.FileSet
}

// NewFileSet creates a new file set.
func NewFileSet() *FileSet {
	return &FileSet{ got.NewFileSet() }
}

func (s *FileSet) AddFile(filename string, base, size int) *TokFile {
	return &TokFile{ s.FileSet.AddFile(filename, base, size) }
}

func (s *FileSet) Iterate(f func(*TokFile) bool) {
	s.FileSet.Iterate(func(file *got.File) bool {
		return f(&TokFile{ file })
	})
}
