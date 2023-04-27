//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"path/filepath"
	"unicode"
	"unicode/utf8"
	"fmt"
)

const (
	isCompoundLine scanbits = 1 << iota    // 0000000000000001
	isCompoundString // 0000000000000010 "...."
	isCall           // 0000000000000100 $.....
	isCallParen      // 0000000000001000 $(...)            8
	isCallBrace      // 0000000000010000 ${...}            16
	isCallColonL     // 0000000000100000 $:....            32
	isCallColonR     // 0000000001000000 $:...:            64
	isGroup          // 0000000010000000 (...)             128
	isBrace          // 0000000100000000 {...}             256
	isRecipes        // 0000001000000000
	isRecipeTab      // 0000010000000000 \t
	isHashValid      // 0001000000000000 scan '#' as HASH token
	isMaximumBit     // 1000000000000000
)

type ScanState struct {
	ch         rune  // current character
	offset     int   // character offset
	readOffset int   // reading offset (position after current character)
	lineOffset int   // current line offset
	bitss []scanbits // scan bits stack
	bits    scanbits // scan bits
}

func (s ScanState) String() string {
	var t string
	switch s.ch {
	case '\n': t = "\\n"
	default  : t = string(s.ch)
	}
	// if s.ch != '\n' { t = string(s.ch) } else { t = "\\n" }
	return fmt.Sprintf("{%s, {%v %v %v}, %016b %016b}", t,
		s.lineOffset, s.offset, s.readOffset, s.bitss, s.bits)
}

func (s *ScanState) push(bits scanbits) {
	s.bitss = append(s.bitss, s.bits) // &^ isLineFeed
	s.bits = bits
}
func (s *ScanState) pop(bits scanbits) {
	if bits == 0 || (s.bits&bits != 0) {
		if n := len(s.bitss); 0 < n {
			s.bits = s.bitss[n-1] //&^ isLineFeed
			s.bitss = s.bitss[0:n-1]
		} else {
			s.bits = 0
		}
	}
}

func (s *ScanState) SetBits(bits scanbits) (prev scanbits) {
	prev = s.bits
	s.bits = bits
	return
}

func (s *ScanState) AddBits(bits scanbits) (prev scanbits) {
	prev = s.bits
	s.bits |= bits
	return
}

func (s *ScanState) RemBits(bits scanbits) (prev scanbits) {
	prev = s.bits
	s.bits &^= bits
	return
}

func (s *ScanState) CommentsOff() scanbits { return s.AddBits(isHashValid) }
func (s *ScanState) recipes(v bool) {
	var bits = s.bits
	if v { bits |= isRecipes } else { bits &^= isRecipes }
	s.bits = bits
}

func (s *ScanState) canRecipe() (res bool) {
	if t := s.bits; (s.lineOffset == s.offset-1) && t.canRecipe() {
		res = !t.is(isCallParen|isCallBrace|isCallColonL|isCallColonR|isGroup)
	}
	return
}

func (s *ScanState) bit(bits scanbits) (res bool) {
	if res = s.bits&bits != 0; !res {
		for i := len(s.bitss)-1; 0 <= i; i -= 1 {
			if res = s.bitss[i]&bits != 0; res { break }
		}
	}
	return
}

const bom = 0xFEFF // byte order mark, only permitted as very first character

// A Scanner holds the scanner's internal state while processing
// a given text.  It can be allocated as part of another data
// structure but must be initialized via Init before use.
//
// (See go.token)
type Scanner struct {
	// immutable state
	file *TokFile  // source file handle
	dir  string       // directory portion of file.Name()
	src  []byte       // source
	err  ErrorHandler // error reporting; or nil
	war  ErrorHandler // warning handler; or nil
	mode ScanMode         // scanning mode

	// scanning state
	ScanState

	// public state - ok to modify
	ErrorCount int // number of errors encountered

	Debug bool
}

func (s *Scanner) File() *TokFile { return s.file }

// Read the next Unicode char into s.ch.
// s.ch < 0 means end-of-file.
func (s *Scanner) next() {
	var newline = s.ch == '\n'

	if s.readOffset < len(s.src) {
		s.offset = s.readOffset
		if s.ch == '\n' {
			s.lineOffset = s.offset
			s.file.AddLine(s.offset)
		}
		var w int
		s.ch, w = s.pick(s.readOffset)
		s.readOffset += w
	} else {
		s.offset = len(s.src)
		if s.ch == '\n' {
			s.lineOffset = s.offset
			s.file.AddLine(s.offset)
		}
		s.ch = -1 // eof
	}

	switch {
	case /* s.bits.isLineFeed() */newline && s.ch == '\t':
		s.bits  |= isRecipeTab
		// s.bits &^= isLineFeed
	// case s.ch == '\n':
	// 	s.bits  |= isLineFeed
	// 	s.bits &^= isRecipeTab
	default:
		// s.bits &^= isLineFeed | isRecipeTab
		s.bits &^= isRecipeTab
	}

	if false && s.Debug { s.warn(s.offset, string(s.ch)) }
}

func (s *Scanner) pickNext() (ch rune, w int) {
	if n := s.readOffset + 1; n < len(s.src) { ch, w = s.pick(n) }
	return
}

func (s *Scanner) pick(offset int) (ch rune, w int) {
	switch ch, w = rune(s.src[offset]), 1; {
	case ch == 0: s.error(offset, "illegal character NUL")
	case ch >= 0x80: // Non ASCII
		ch, w = utf8.DecodeRune(s.src[offset:])
		if ch == utf8.RuneError && w == 1 {
			s.error(offset, "illegal UTF-8 encoding")
		} else if ch == bom && offset > 0 {
			s.error(offset, "illegal byte order mark")
		}
	}
	return
}

// An ErrorHandler may be provided to Scanner.Init. If a syntax error is
// encountered and a handler was installed, the handler is called with a
// position and an error message. The position points to the beginning of
// the offending token.
//
type ErrorHandler func(pos Position, msg string)

// A mode value is a set of flags (or 0).
// They control scanner behavior.
//
type ScanMode uint
type scanbits uint
func (bits scanbits) is(t scanbits)     bool { return bits&t != 0 }
func (bits scanbits) isCall()           bool { return bits&isCall != 0 }
func (bits scanbits) isCallZero()       bool { return bits&isCall != 0 && bits&(isCallParen|isCallBrace|isCallColonL) == 0 }
func (bits scanbits) isCallParen()      bool { return bits&isCallParen != 0 }
func (bits scanbits) isCallBrace()      bool { return bits&isCallBrace != 0 }
func (bits scanbits) isCallColonL()     bool { return bits&isCallColonL != 0 }
func (bits scanbits) isCallColonR()     bool { return bits&isCallColonR != 0 }
func (bits scanbits) isCommentsOff()    bool { return bits&isHashValid != 0 }
func (bits scanbits) isCompoundLine()   bool { return bits&isCompoundLine != 0 }
func (bits scanbits) isCompoundString() bool { return bits&isCompoundString != 0 }
func (bits scanbits) isGroup()          bool { return bits&isGroup != 0 }
func (bits scanbits) isBrace()          bool { return bits&isBrace != 0 }
func (bits scanbits) canRecipe()        bool { return bits&(isRecipeTab|isRecipes) != 0 }

func IsLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= 0x80 && unicode.IsLetter(ch)
}

func IsDigit(ch rune) bool {
	return ('0' <= ch && ch <= '9') || (ch >= 0x80 && unicode.IsDigit(ch))
}

// punctuation used as non-terminator
func IsUntermPunct(ch rune) bool {
	// Most chars accepted in URI (RFC3986)
	return ch == '-' || ch == '+' || ch == '@' /*|| ch == '.' || ch == '/'*/;
}

func IsDatetimeTerminator(ch rune) bool {
	return  ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' ||
		ch == '(' || ch == ')' || ch == '{' || ch == '}' ||
		ch == '$' || ch == '#' || ch == '\\'
}

func IsIdentifier(ch rune) bool {
	return IsLetter(ch) || IsDigit(ch) || IsUntermPunct(ch) //|| ch == '\\'
}

// Init prepares the scanner s to tokenize the text src by setting the
// scanner at the beginning of src. The scanner uses the file set file
// for position information and it adds line information for each line.
// It is ok to re-use the same file when re-scanning the same file as
// line information which is already present is ignored. Init causes a
// panic if the file size does not match the src size.
//
// Calls to Scan will invoke the error handler err if they encounter a
// syntax error and err is not nil. Also, for each error encountered,
// the Scanner field ErrorCount is incremented by one. The mode parameter
// determines how comments are handled.
//
// Note that Init may call err if there is an error in the first character
// of the file.
//
func (s *Scanner) Init(file *TokFile, src []byte, mode ScanMode, err, war ErrorHandler) {
	// Explicitly initialize all fields since a scanner may be reused.
	if file.Size() != len(src) {
		panic(fmt.Sprintf("file size (%d) does not match src len (%d)", file.Size(), len(src)))
	}
	s.file = file
	s.dir, _ = filepath.Split(file.Name())
	s.src = src
	s.err = err
	s.war = war
	s.mode = mode

	s.ch = ' '
	s.offset = 0
	s.readOffset = 0
	s.lineOffset = 0
	s.bits = 0
	s.bitss = nil

	s.ErrorCount = 0

	// The BOM at file beginning will be discarded.
	if s.next(); s.ch == bom { s.next() }
}

func (s *Scanner) error(offs int, msg string) {
	if s.err != nil { s.err(s.file.Position(s.file.Pos(offs)), msg) }
	s.ErrorCount++
}
func (s *Scanner) warn(offs int, msg string) {
	if s.war != nil { s.war(s.file.Position(s.file.Pos(offs)), msg) }
}

func (s *Scanner) scanComment() (res string) {
	for s.ch == ' '  || s.ch == '\t' { s.next() } // skip preceding spaces

	var offs = s.offset
	for s.ch != '\n' && s.ch != -1 { s.next() }
	return string(s.src[offs:s.offset])
}

func (s *Scanner) scanIdentifier() string {
	var offs = s.offset
	for IsIdentifier(s.ch) {
		if s.next(); s.ch == '-' { // Looking for '->'
			var n = s.offset + 1 // No need UTF8 decoding!
			if n < len(s.src) && rune(s.src[n]) == '>' { break }
		}
	}
	return string(s.src[offs:s.offset])
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9': return int(ch - '0')
	case 'a' <= ch && ch <= 'f': return int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F': return int(ch - 'A' + 10)
	}
	return 16 // larger than any legal digit val
}

func (s *Scanner) scanMantissa(base int) {
	if digitVal(s.ch) < base { // first digit
		s.next()
		if true {
			for digitVal(s.ch) < base { s.next() }
		} else {
			// NOTE: disable '_' number separaters as ParseInt not support it and
			//       it's not recoverable from ints back to strings.
			for s.ch == '_' || digitVal(s.ch) < base {
				if s.ch == '_' {
					if s.next(); s.ch == '_' {
						s.error(s.offset-1, "invalid digit group")
						break
					}
				} else {
					s.next()
				}
			}
		}
	}
}

func (s *Scanner) scanDatetime() (tok Token) {
	var (
		ch byte
		hasDate = false
		hasTime = false
		o = s.offset
		l = len(s.src)
	)
	if x := l-o; 8 <= x {
		for i := 0; i < 2; i++ {
			if ch = s.src[o+i]; ch < '0' || '9' < ch {
				goto exit
			}
		}
		if s.src[o+2] == ':' || s.src[o+5] == ':' {
			hasTime = true; goto checkTime
		}
		if s.src[o+4] == '-' || s.src[o+7] == '-' && 10 <= x {
			hasDate = true; goto checkDate
		}
	}

	goto exit

checkDate:
	// 4 digits fullyear (first two digit already checked)
	for i := 2; i < 4; i++ {
		if ch = s.src[o+i]; ch < '0' || '9' < ch {
			goto exit
		}
	}

	// month range is 01-12
	if ch = s.src[o+5]; ch != '0' && ch != '1' {
		s.error(o+5, "bad month"); goto exit
	}
	if ch = s.src[o+6]; ch < '0' || '9' < ch {
		s.error(o+6, "bad month"); goto exit
	}

	// month-day range is 01-28, 01-29, 01-30, 01-31 based on month/year
	if ch = s.src[o+8]; ch < '0' && '3' < ch {
		s.error(o+8, "bad month day"); goto exit
	}
	if ch = s.src[o+9]; ch < '0' || '9' < ch {
		s.error(o+9, "bad month day"); goto exit
	}

	if o += 10; o == l {
		goto success // 1979-05-27
	} else if ch = s.src[o]; IsDatetimeTerminator(rune(ch)) {
		goto success // 1979-05-27
	}

	if ch == 'T' || ch == 't' {
		o += 1 // consume 'T'
		hasTime = true
	} else {
		s.error(o, "bad time"); goto exit
	}

	if l-o < 9 || s.src[o+2] != ':' || s.src[o+5] != ':' {
		s.error(o, "illegal time"); goto exit
	}

checkTime:
	// hour range is 00-23
	if ch = s.src[o+0]; ch < '0' || '2' < ch {
		s.error(o+0, "bad hour"); goto exit
	}
	if ch = s.src[o+1]; ch < '0' || '9' < ch || ('3' < ch && s.src[o] == '2') {
		s.error(o+1, "bad hour"); goto exit
	}

	// minute range is 00-59
	if ch = s.src[o+3]; ch < '0' || '5' < ch {
		s.error(o+3, "bad minute"); goto exit
	}
	if ch = s.src[o+4]; ch < '0' || '9' < ch {
		s.error(o+4, "bad minute"); goto exit
	}

	// second ranges are 00-59 00-58, 00-59, 00-60 based on leap second rules
	if ch = s.src[o+6]; ch < '0' || '5' < ch {
		s.error(o+6, "bad second"); goto exit
	}
	if ch = s.src[o+7]; ch < '0' || '9' < ch {
		s.error(o+7, "bad second"); goto exit
	}

	if ch = s.src[o+8]; IsDatetimeTerminator(rune(ch)) {
		o += 8; goto success // consume 00:00:00
	} else if ch == 'Z' || ch == 'z' {
		o += 9; goto success // consume 00:00:00Z
	} else if ch == '.' {
		for o += 9; o < l; o++ {// consume 00:00:00.
			if ch = s.src[o]; ch == 'Z' || ch == 'z' {
				o += 1; goto success // consume 'Z'
			} else if IsDatetimeTerminator(rune(ch)) {
				goto success
			} else if ch == '+' || ch == '-' {
				o += 1; goto checkNumOffset // consume '+' or '-'
			} else if ch < '0' || '9' < ch {
				s.error(o, "bad secfrac"); goto exit
			}
		}
	} else if ch == '+' || ch == '-' {
		o += 9; goto checkNumOffset // consume 00:00:00+
	} else {
		s.error(o, "bad time"); goto exit
	}

checkNumOffset:
	if ch = s.src[o+2]; ch != ':' {
		s.error(o+2, "bad offset"); goto exit
	}

	// hour range is 00-23
	if ch = s.src[o+0]; ch < '0' || '2' < ch {
		s.error(o+0, "bad hour"); goto exit
	}
	if ch = s.src[o+1]; ch < '0' || '9' < ch || ('3' < ch && s.src[o] == '2') {
		s.error(o+1, "bad hour"); goto exit
	}

	// minute range is 00-59
	if ch = s.src[o+3]; ch < '0' || '5' < ch {
		s.error(o+3, "bad minute"); goto exit
	}
	if ch = s.src[o+4]; ch < '0' || '9' < ch {
		s.error(o+4, "bad minute"); goto exit
	}

	o += 5 // consume 00:00

success:
	for i := s.offset; i < o; i++ { s.next() }
	switch {
	case hasDate && hasTime: tok = DATETIME
	case hasDate && !hasTime: tok = DATE
	case !hasDate && hasTime: tok = TIME
	default: tok = ILLEGAL
	}
exit:
	return
}

func (s *Scanner) scanNumber(seenDecimalPoint bool) (Token, string) {
	// digitVal(s.ch) < 10
	offs := s.offset
	tok := INTEGER

	if seenDecimalPoint {
		offs--
		tok = FLOAT
		s.scanMantissa(10)
		goto exponent
	}

	if t := s.scanDatetime(); t != ILLEGAL {
		tok = t; goto exit
	}

	if s.ch == '0' {
		// int or float
		offs := s.offset
		s.next()
		if s.ch == 'b' || s.ch == 'B' {
			// binary int
			s.next()
			s.scanMantissa(2)
			tok = BINARY
			if s.offset-offs <= 2 {
				// only scanned "0b" or "0B"
				s.error(offs, "illegal binary number")
			}
		} else if s.ch == 'x' || s.ch == 'X' {
			// hexadecimal int
			s.next()
			s.scanMantissa(16)
			tok = HEXADECIMAL
			if s.offset-offs <= 2 {
				// only scanned "0x" or "0X"
				s.error(offs, "illegal hexadecimal number")
			}
		} else {
			// octal int or float
			seenDecimalDigit := false
			s.scanMantissa(8)
			if s.ch == '8' || s.ch == '9' {
				// illegal octal int or float
				seenDecimalDigit = true
				s.scanMantissa(10)
			}
			if s.ch == '.' || s.ch == 'e' || s.ch == 'E' || s.ch == 'i' {
				goto fraction
			}
			// octal int
			if seenDecimalDigit {
				s.error(offs, "illegal octal number")
			}
			if s.offset-offs > 1 {
				tok = OCTAL
			} else {
				tok = INTEGER // just '0'
			}
		}
		goto exit
	}

	// decimal int or float
	s.scanMantissa(10)

fraction:
	if s.ch == '.' {
		if n := s.offset+2; n < len(s.src) {
			if ch := rune(s.src[n]); /*unicode.IsSpace(ch) { // 1. -> FLOAT 1.0
                                // do nothing here
                        } else if*/ !IsDigit(ch) { // 1.o -> INT 1    DOT .    STRING o
				goto exit
			}
		}
		tok = FLOAT
		s.next()
		s.scanMantissa(10)
	}

exponent:
	if s.ch == 'e' || s.ch == 'E' {
		tok = FLOAT
		s.next()
		if s.ch == '-' || s.ch == '+' {
			s.next()
		}
		s.scanMantissa(10)
	}

	/*
	if s.ch == 'i' {
		tok = IMAG
		s.next()
	} */

exit:
	return tok, string(s.src[offs:s.offset])
}

func (s *Scanner) scanEscape(quote rune) bool {
	var n int
	var base, max uint32
	var offs = s.offset
	switch s.ch {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '$', quote:
		s.next()
		return true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		n, base, max = 3, 8, 255
	case 'x':
		s.next()
		n, base, max = 2, 16, 255
	case 'u':
		s.next()
		n, base, max = 4, 16, unicode.MaxRune
	case 'U':
		s.next()
		n, base, max = 8, 16, unicode.MaxRune
	case '\n':
		s.next()
	default:
		var msg = "unknown escape sequence"
		if s.ch < 0 { msg = "escape sequence not terminated" }
		s.error(offs, msg)
		return false
	}

	var x uint32
	for n > 0 {
		d := uint32(digitVal(s.ch))
		if d >= base {
			var msg = fmt.Sprintf("illegal character %#U in escape sequence", s.ch)
			if s.ch < 0 { msg = "escape sequence not terminated" }
			s.error(s.offset, msg)
			return false
		}
		x = x*base + d
		s.next()
		n--
	}

	if x > max || 0xD800 <= x && x < 0xE000 {
		s.error(offs, "escape sequence is invalid Unicode code point")
		return false
	}

	return true
}

func (s *Scanner) scanRawString(ml bool) string {
	// '\'' opening already consumed
	offs := s.offset - 1
	if ml { offs -= 1 }

	for s.readOffset < len(s.src) {
		ch := s.ch
		if (!ml && ch == '\n') || ch < 0 { // if ch < 0 {
			s.error(offs, "raw string literal not terminated")
			break
		}
		if ch == '\\' { s.next() } // escapes
		s.next()
		if ch == '\'' {
			if !ml { break }
			if s.ch == '\'' {
				if s.next(); s.ch == '\'' {
					s.next()
					break
				}
			}
		}
	}

	return string(s.src[offs+1:s.offset-1])
}

func (s *Scanner) scanString(ml bool) string {
	// '"' opening already consumed
	offs := s.offset - 1
	if ml { offs -= 1 }

	for s.readOffset < len(s.src) {
		ch := s.ch
		if (!ml && ch == '\n') || ch < 0 {
			s.error(offs, "string literal not terminated")
			break
		}
		s.next()
		if ch == '"' {
			if !ml {
				break
			}
			if s.ch == '"' {
				if s.next(); s.ch == '"' {
					s.next()
					break
				}
			}
		}
		switch ch {
		case '\\': s.scanEscape('"')
		case '$': //
		}
	}
	return string(s.src[offs:s.offset])
}

func (s *Scanner) scanCompound(q rune) (tok Token, lit string) {
	var offs = s.offset
	if q != 0 && s.ch == q {
		tok = COMPOSED
		s.pop(isCompoundString)
		s.next() // take the ending '"'
		return
	}
	switch s.ch {
	case '\n': // mistaken compound string terminated with line feed
		tok = LINEND
		s.pop(isCompoundString|isCompoundLine)
		if s.next(); s.ch != '\t' { s.bits &^= isRecipes }
		return
	case '\\':
		if s.next(); q == 0 { // skim the \ character
			tok, lit = ESCAPE, string(s.ch)
			s.next() // the escaped character
		} else if s.scanEscape(/*'"'*/q) {
			tok, lit = ESCAPE, string(s.src[offs+1:s.offset])
		} else {
			tok, lit = ILLEGAL, string(s.src[offs:s.offset])
			s.error(offs, fmt.Sprintf("illegal compound escape %#U", s.ch))
			s.next() // discard
		}
		return
	case '&', '$': // Escapes '&', '$', but '&&' or '$$' is not escaped.
		if n := s.offset+1; n < len(s.src) && rune(s.src[n]) == s.ch {
			s.next() //! The first & or $
			s.next() //! The second & or $
		} else if s.ch == '$' {
			return DELEGATE, lit
		} else {
			return CLOSURE, lit
		}
	}

ScanLoop:
	for ; s.readOffset < len(s.src); s.next() {
		switch s.ch { case '\\', '\n', '$', '&', q: break ScanLoop }
	}
	tok, lit = RAW, string(s.src[offs:s.offset])
	return
}

func (s *Scanner) Scan() (pos Pos, tok Token, lit string) {
	switch pos = s.file.Pos(s.offset); {
	case s.offset >= len(s.src) || s.ch == -1: return pos, EOF, ""
	case s.bits.isCompoundLine()  : tok, lit = s.scanCompound(0)
	case s.bits.isCompoundString(): tok, lit = s.scanCompound('"')
	}
	if tok != 0 && tok != CLOSURE && tok != DELEGATE { return }
	if tok != 0 { if s.ch != '$' && s.ch != '&' { s.error(s.offset, string(s.src[s.offset:])) }}

	if IsDigit(s.ch) { // '0' <= s.ch && s.ch <= '9'
		tok, lit = s.scanNumber(false)
		return
	}
	if IsLetter(s.ch) {
		if lit = s.scanIdentifier(); len(lit) > 1 && s.ch != '/' && s.ch != '.' {
			if tok = Lookup(lit); !tok.IsKeyword() && tok != BAREWORD {
				s.error(s.offset, "unexpected token '"+tok.String()+"' "+lit)
			}
		} else { tok = BAREWORD }
		if s.bits.isCallZero() { s.pop(/*isCall*/0) }
		return
	}

	var (
		offs = s.offset
		ch = s.ch
	)
	switch s.next(); ch {
	case '#':
		if s.bits.isCommentsOff() {
			tok = HASH
			lit = string(ch)
		} else {
			tok = COMMENT
			lit = s.scanComment()
			s.next() // discard '\n'
		}
	case '!':
		if tok = EXC; s.ch == '=' {
			tok = EXC_ASSIGN
			s.next()
		}
	case '?':
		if tok = QUE; s.ch == '=' {
			tok = QUE_ASSIGN
			s.next()
		}
	case '+':
		if tok = PLUS; s.ch == '=' {
			tok = ADD_ASSIGN
			s.next()
		}
	case '-':
		if s.ch == '-' { // "-->" => "-", "->"
			if s.readOffset < len(s.src) && s.src[s.readOffset] == '>' {
				tok, lit = BAREWORD, "-"
			} else {
				tok = MINUS
			}
		} else if s.ch == '=' { // -=
			tok = SUB_ASSIGN
			s.next()
			if s.ch == '+' { // -=+
				tok = SSH_ASSIGN
				s.next()
			}
		} else if s.ch == '+' { // -+
			s.next()
			if s.ch == '=' { // -+=
				tok = SAD_ASSIGN
				s.next()
			} else {
				tok = ILLEGAL
			}
		} else if s.ch == '>' {
			tok = SELECT_PROP
			s.next()
		} else if '0' <= s.ch && s.ch <= '9' {
			tok, lit = s.scanNumber(false)
			lit = "-" + lit // minus number
		} else {
			tok = MINUS
		}
	case '\\':
		tok, lit = ESCAPE, string(s.ch)
		s.next() // eat escaped char
	case '\'':
		if tok = STRING; s.ch == '\'' {
			if s.next(); s.ch == '\'' { // '''
				lit = s.scanRawString(true)
			} else if offs := s.offset - 2; false {
				lit = string(s.src[offs:s.offset])
			} else {
				lit = "" // empty string ''
			}
		} else {
			lit = s.scanRawString(false)
		}
	case '"':
		if s.bits.isCompoundString() { s.error(offs, "composed") }
		tok = COMPOUND
		s.push(isCompoundString)
	case '$', '&':
		var isDelegate = ch == '$' // assert(s.offset == s.readOffset-1)
		switch tok, ch = CLOSURE, rune(s.src[s.offset]); ch {
		case '/' : tok = CLOSURE_r
		case '.' : tok = CLOSURE_D
		case '@' : tok = CLOSURE_A
		case '|' : tok = CLOSURE_B
		case '<' : tok = CLOSURE_L
		case '>' : tok = CLOSURE_R
		case '^' : tok = CLOSURE_U
		case '*' : tok = CLOSURE_S
		case '-' : tok = CLOSURE_M
		case '+' : tok = CLOSURE_P
		case '?' : tok = CLOSURE_Q
		case '0' : tok = CLOSURE_0
		case '1' : tok = CLOSURE_1
		case '2' : tok = CLOSURE_2
		case '3' : tok = CLOSURE_3
		case '4' : tok = CLOSURE_4
		case '5' : tok = CLOSURE_5
		case '6' : tok = CLOSURE_6
		case '7' : tok = CLOSURE_7
		case '8' : tok = CLOSURE_8
		case '9' : tok = CLOSURE_9
		case '_' : tok = CLOSURE__
		}
		if CLOSURE < tok {
			lit = string(ch)
			s.next() // eat special
		} else if ch == '(' || ch == '{' || ch == ':' {
			s.push(isCall)
		} else {
			s.push(isCall /* | isCallZero */)
		}
		if isDelegate { tok = Token(DELEGATE + (tok - CLOSURE)) }
	case '(':
		tok, lit = LPAREN, string(ch)
		if s.bits.isCallZero() { s.bits |= isCallParen } else { s.push(isGroup) }
	case ')':
		tok, lit = RPAREN, string(ch)
		if s.bits&(isCallParen|isGroup) == 0 { s.error(offs, "unexpected paren") }
		s.pop(isGroup|isCallParen)
	case '{':
		tok = LBRACE
		if s.bits.isCallZero() { s.bits |= isCallBrace } else { s.push(isBrace) }
	case '}':
		tok = RBRACE
		if s.bits&(isCallBrace|isBrace) == 0 { s.error(offs, "unexpected brace") }
		s.pop(isBrace|isCallBrace)
	case '=':
		if s.ch == '>' { // =>
			tok = SELECT_PROG1
			s.next() // concume the '>'
		} else if s.ch == '+' {
			tok = SHI_ASSIGN
			s.next()
		} else {
			tok = ASSIGN
		}
	case ' ', '\t':
		if ch == '\t' && s.canRecipe() {
			tok, lit = RECIPE, string(ch)
			s.push(isCompoundLine)
		} else {
			for s.ch == ' ' || s.ch == '\t' { s.next() }
			tok, lit = SPACE, string(s.src[offs:s.offset])
		}
	case '~':
		if s.ch == '>' { // ~>
			tok = SELECT_PROG2
			s.next() // concume the '>'
		} else {
			tok = TILDE
		}
	case '.':
		if tok = DOT; s.ch == '.' {
			tok = DOTDOT
			s.next()
		} else if IsDigit(s.ch) {
			if n := s.offset-2; n > -1 && unicode.IsSpace(rune(s.src[n])) { // skip xxx.1
				tok, lit = s.scanNumber(true)
			}
		}
	case ':':
		if s.ch == '=' {
			tok = SCO_ASSIGN
			s.next() // consume '='
		} else if s.ch == ':' {
			tok = COLON2
			s.next() // consume the second ':'
			if s.ch == '=' {
				tok = DCO_ASSIGN
				s.next() // consume '='
			}
		} else {
			tok = COLON
		}
	case '*':
		tok = STAR
	case '%':
		tok = PERC
	case '@':
		tok = AT
	case '|':
		tok = BAR
	case '/':
		tok = PCON
	case ',':
		tok = COMMA
	case '→': // different from ' → '
		tok = SELECT_PROP
	case '⇒': // =>
		tok = SELECT_PROG1
	case '⇢': // ~>
		tok = SELECT_PROG2
	case '≔':
		tok = SCO_ASSIGN
	case '⩴':
		tok = DCO_ASSIGN
	case ';':
		tok = SEMICOLON
	case '^':
		tok = CARET
	case '<':
		tok = LANGLE
	case '>':
		tok = RANGLE
	case '[':
		tok = LBRACK
	case ']':
		tok = RBRACK
	case '\n':
		tok = LINEND
		if s.pop(isCompoundLine); s.ch != '\t' { s.bits &^= isRecipes }
	default:
		// next reports unexpected BOMs - don't repeat
		if ch != bom { s.error(s.file.Offset(pos), fmt.Sprintf("illegal %#U", ch)) }
		tok = ILLEGAL
		lit = string(ch)
	}
	return
}
