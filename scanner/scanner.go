//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package scanner

import (
	"extbit.io/smart/token"
	"path/filepath"
	"unicode"
	"unicode/utf8"
	"fmt"
	//"os"
)

// A Scanner holds the scanner's internal state while processing
// a given text.  It can be allocated as part of another data
// structure but must be initialized via Init before use.
//
// (See go.token)
type Scanner struct {
	// immutable state
	file *token.File  // source file handle
	dir  string       // directory portion of file.Name()
	src  []byte       // source
	err  ErrorHandler // error reporting; or nil
	mode Mode         // scanning mode

	// scanning state
	ch         rune // current character
	offset     int  // character offset
	readOffset int  // reading offset (position after current character)
	lineOffset int  // current line offset

	bits    scanbits // scanning context bits
	bitsv []scanbits // scan bits vector

	// public state - ok to modify
	ErrorCount int // number of errors encountered

	Debug bool
}

const bom = 0xFEFF // byte order mark, only permitted as very first character

func (s *Scanner) bitsPush(bits scanbits) {
	s.bitsv = append(s.bitsv, s.bits)
	s.bits  = bits
}

func (s *Scanner) bitsPop(bits scanbits) {
	if s.bits.is(bits) {
		if n := len(s.bitsv) - 1; n >= 0 {
			s.bits  = s.bitsv[  n]
			s.bitsv = s.bitsv[0:n]
		} else {
			s.bits &^= bits
		}
		s.bits &^= isFirstOfLine // clear first-of-line bit
	}
}

func (s *Scanner) SetBits(bits scanbits) (prev scanbits) {
	prev = s.bits
	s.bits = bits
	return
}

func (s *Scanner) AddBits(bits scanbits) (prev scanbits) {
	prev  = s.bits
	s.bits |= bits
	return
}

func (s *Scanner) IsFirstOfLine() bool { return s.bits.isFirstOfLine() }

// Read the next Unicode char into s.ch.
// s.ch < 0 means end-of-file.
//
func (s *Scanner) next() {
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
	if s.bits.isFirstOfLine() && s.ch == '\t' {
		s.bits |= canBeRecipeTab
	} else if s.ch == '\n' {
		s.bits |= isFirstOfLine
	} else {
		s.bits &^= isFirstOfLine // clear first-of-line bit
	}
}

func (s *Scanner) pickNext() (ch rune, w int) {
	if n := s.readOffset + 1; n < len(s.src) {
		ch, w = s.pick(n)
	}
	return
}

func (s *Scanner) pick(offset int) (ch rune, w int) {
	switch ch, w = rune(s.src[offset]), 1; {
	case ch == 0:
		s.error(offset, "illegal character NUL")
	case ch >= 0x80: // Not ASCII!
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
type ErrorHandler func(pos token.Position, msg string)

// A mode value is a set of flags (or 0).
// They control scanner behavior.
//
type Mode uint
type scanbits uint
func (bits scanbits) isCompoundLine() bool { return bits&isCompoundLine != 0 }
func (bits scanbits) isCompoundString() bool { return bits&isCompoundString != 0 }
func (bits scanbits) isCompoundCallIdent() bool { return bits&isCompoundCallIdent != 0 }
func (bits scanbits) isCompoundCallParen() bool { return bits&isCompoundCallParen != 0 }
func (bits scanbits) isCompoundCallBrace() bool { return bits&isCompoundCallBrace != 0 }
func (bits scanbits) isCompoundCallColonL() bool { return bits&isCompoundCallColonL != 0 }
func (bits scanbits) isCompoundCallColonR() bool { return bits&isCompoundCallColonR != 0 }
func (bits scanbits) isCommentsOff() bool { return bits&NoComments != 0 }
func (bits scanbits) isFirstOfLine() bool { return bits&isFirstOfLine != 0 }
func (bits scanbits) canBeRecipe()   bool { return bits&canBeRecipes != 0 && bits&canBeRecipeTab != 0 }
func (bits scanbits) is(t scanbits)  bool { return bits&t != 0 }

const (
	isCompoundLine    scanbits = 1 << iota // 1
	isCompoundString     // "...."            2
	isCompoundCallIdent  // $.....            4
	isCompoundCallParen  // $(...)            8
	isCompoundCallBrace  // ${...}            16
	isCompoundCallColonL // $:....            32
	isCompoundCallColonR // $:...:            64
	isCompoundGroup      // (...)             128
	isFirstOfLine
	canBeRecipes
	canBeRecipeTab
	skipPostLineFeeds // TODO
	NoComments // don't scan comments, '#' will be treat as HASH token
)

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
func (s *Scanner) Init(file *token.File, src []byte, err ErrorHandler, mode Mode) {
	// Explicitly initialize all fields since a scanner may be reused.
	if file.Size() != len(src) {
		panic(fmt.Sprintf("file size (%d) does not match src len (%d)", file.Size(), len(src)))
	}
	s.file = file
	s.dir, _ = filepath.Split(file.Name())
	s.src = src
	s.err = err
	s.mode = mode

	s.ch = ' '
	s.offset = 0
	s.readOffset = 0
	s.lineOffset = 0
	s.bits = 0

	s.ErrorCount = 0

	s.next()
	if s.ch == bom {
		s.next() // ignore BOM at file beginning
	}
}

func (s *Scanner) LeaveCompoundLineContext() { s.bitsPop(isCompoundLine) }
func (s *Scanner) TrunRecipesOn()  { s.bits  |= canBeRecipes }
func (s *Scanner) TurnRecipesOff() { s.bits &^= canBeRecipes }

func (s *Scanner) IsCompoundLineContext() bool {
	return s.bits&isCompoundLine != 0
}

func (s *Scanner) error(offs int, msg string) {
	if s.err != nil {
		s.err(s.file.Position(s.file.Pos(offs)), msg)
	}
	s.ErrorCount++
}

/*
func (s *Scanner) skipUselessWhitespace(lf bool) {
skip:
	for s.readOffset < len(s.src) {
		switch s.ch {
		default: break skip
		case ' ', '\r': s.next()
		case '\t':
			if s.lineOffset < s.offset {
				s.next()
			} else {
				break skip
			}
		case '\\':
			if s.next(); s.ch == '\n' { // continual line
				if i := s.offset+1; i < len(s.src) && s.src[i] == '\n' {
					break skip // Avoid skipping \\\n\n
				}
				if s.bits&isCompoundLine == 0 {
					s.next() // Eat the '\n'
					// Eat tabs after a continual
					for s.ch == '\t' { s.next() }
				} else {
					// preserves the '\n'
					break skip
				}
			} else {
				// TODO: escape character
				s.next()//; break skip
			}
		}
	}
}*/

func (s *Scanner) scanComment() (res string) {
	for s.ch == ' '  || s.ch == '\t' { s.next() } // skip preceding spaces
	for s.ch != '\n' && s.ch != -1 {
		res += string(s.ch)
		s.next()
	}
	return
}

func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= 0x80 && unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9' || ch >= 0x80 && unicode.IsDigit(ch)
}

// punctuation used as non-terminator
func isUntermPunct(ch rune) bool {
	// Most chars accepted in URI (RFC3986)
	return ch == '-' || ch == '+' || ch == '@' /*|| ch == '.' || ch == '/'*/;
}

func isDatetimeTerminator(ch rune) bool {
	return  ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' ||
		ch == '(' || ch == ')' || ch == '{' || ch == '}' ||
		ch == '$' || ch == '#' || ch == '\\'
}

func (s *Scanner) scanIdentifier() string {
	// first char is letter (ensured)
	offs := s.offset
	Loop: for isLetter(s.ch) || isDigit(s.ch) || isUntermPunct(s.ch) /*|| s.ch == '\\'*/ {
		/* if ident && (isUntermPunct(s.ch) || s.ch == '\\') {
                        ident = false
                } */
		switch {
		/*case s.ch == '-' && ch == '>': // ->
                        break*/
		/*case s.ch == '\\':
                        switch s.next(); s.ch {
		            case '\n': break loop
		            default:
				s.error(s.offset-1, fmt.Sprintf("illegal ident escape %#U", s.ch))
                                break loop
                        }*/
		default:
			switch s.next(); s.ch { // Accept one char here.
			case '-': // Looking at SELECT operators, need to stop at '->'
				if n := s.offset + 1; n < len(s.src) {
					// No need UTF8 decoding!
					if ch := rune(s.src[n]); ch == '>' {
						break Loop
					}
				}
			}
		}
	}
	return string(s.src[offs:s.offset])
}

func (s *Scanner) scanCompoundString() (tok token.Token, lit string) {
	offs := s.offset
	switch s.ch {
	case '\\':
		if s.next(); s.scanEscape('"') {
			tok, lit = token.ESCAPE, string(s.src[offs+1:s.offset])
			//s.next() // escape
			return
		} else {
			tok, lit = token.ILLEGAL, string(s.src[offs:s.offset])
			s.error(s.offset-1, fmt.Sprintf("illegal compound escape %#U", s.ch))
			s.next() // discard
			return
		}
	case '"':
		tok = token.COMPOSED
		s.bitsPop(isCompoundString) //s.bits &= ^isCompoundString
		s.next() // take the ending '"'
		return
	case '\n': // compound string terminated by line feed (mistake)
		tok = token.LINEND
		s.bitsPop(isCompoundString) //s.bits &= ^isCompoundString
		s.next() // take the ending '\n'
		return
	case '&', '$': // Escapes '&', '$', but '&&' or '$$' is not escaped.
		if n := s.offset+1; n < len(s.src) && rune(s.src[n]) == s.ch {
			s.next() //! The first & or $
			s.next() //! The second & or $
			tok, lit = token.RAW, string(s.src[offs:s.offset])
		} else if s.ch == '$' {
			tok = token.DELEGATE // escape to do token.DELEGATE
		} else {
			tok = token.CLOSURE // escape to do token.CLOSURE
		}
		return
	}
LoopChar:
	for s.readOffset < len(s.src) {
		switch s.ch {
		case '\\', '$', '&', '"', '\n':
			// just break it out, further scanning will decide escape
			break LoopChar
		default:
			s.next()
		}
	}
	tok, lit = token.RAW, string(s.src[offs:s.offset])
	return 
}

func (s *Scanner) scanCompoundLine() (tok token.Token, lit string) {
	offs := s.offset
	switch s.ch {
	case '\\':
		s.next() // eat \
		tok, lit = token.ESCAPE, string(s.ch)
		s.next() // skip escaped character
		return
	case '\n':
		tok = token.LINEND
		s.bitsPop(isCompoundLine) //s.bits &= ^isCompoundLine
		s.next() // take the line-end
		return
	case '&', '$': // Escapes '&', '$', but '&&' and '$$' is not escaped.
		if n := s.offset+1; n < len(s.src) && rune(s.src[n]) == s.ch {
			s.next() //! The first & or $
			s.next() //! The second & or $
			tok, lit = token.RAW, string(s.src[offs:s.offset])
		} else if s.ch == '$' {
			tok = token.DELEGATE // escape to do token.DELEGATE
		} else {
			tok = token.CLOSURE // escape to do token.CLOSURE
		}
		return
	}
LoopChar:
	for s.ch != '\n' && s.readOffset < len(s.src){
		switch s.ch {
		case '\\', '$', '&':
			// just break it out, further scanning will decide
			break LoopChar
		default:
			s.next()
		}
	}
	return token.RAW, string(s.src[offs:s.offset])
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= ch && ch <= 'f':
		return int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F':
		return int(ch - 'A' + 10)
	}
	return 16 // larger than any legal digit val
}

func (s *Scanner) scanMantissa(base int) {
	if digitVal(s.ch) < base { // first digit
		s.next()
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

func (s *Scanner) scanDatetime() (tok token.Token) {
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
	} else if ch = s.src[o]; isDatetimeTerminator(rune(ch)) {
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

	if ch = s.src[o+8]; isDatetimeTerminator(rune(ch)) {
		o += 8; goto success // consume 00:00:00
	} else if ch == 'Z' || ch == 'z' {
		o += 9; goto success // consume 00:00:00Z
	} else if ch == '.' {
		for o += 9; o < l; o++ {// consume 00:00:00.
			if ch = s.src[o]; ch == 'Z' || ch == 'z' {
				o += 1; goto success // consume 'Z'
			} else if isDatetimeTerminator(rune(ch)) {
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
	case hasDate && hasTime: tok = token.DATETIME
	case hasDate && !hasTime: tok = token.DATE
	case !hasDate && hasTime: tok = token.TIME
	default: tok = token.ILLEGAL
	}
exit:
	return
}

func (s *Scanner) scanNumber(seenDecimalPoint bool) (token.Token, string) {
	// digitVal(s.ch) < 10
	offs := s.offset
	tok := token.INT

	if seenDecimalPoint {
		offs--
		tok = token.FLOAT
		s.scanMantissa(10)
		goto exponent
	}

	if t := s.scanDatetime(); t != token.ILLEGAL {
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
			tok = token.BIN
			if s.offset-offs <= 2 {
				// only scanned "0b" or "0B"
				s.error(offs, "illegal binary number")
			}
		} else if s.ch == 'x' || s.ch == 'X' {
			// hexadecimal int
			s.next()
			s.scanMantissa(16)
			tok = token.HEX
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
				tok = token.OCT
			} else {
				tok = token.INT // just '0'
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
                        } else if*/ !isDigit(ch) { // 1.o -> INT 1    DOT .    STRING o
				goto exit
			}
		}
		tok = token.FLOAT
		s.next()
		s.scanMantissa(10)
	}

exponent:
	if s.ch == 'e' || s.ch == 'E' {
		tok = token.FLOAT
		s.next()
		if s.ch == '-' || s.ch == '+' {
			s.next()
		}
		s.scanMantissa(10)
	}

	/*
	if s.ch == 'i' {
		tok = token.IMAG
		s.next()
	} */

exit:
	return tok, string(s.src[offs:s.offset])
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

func (s *Scanner) scanEscape(quote rune) bool {
	offs := s.offset

	var n int
	var base, max uint32
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
		msg := "unknown escape sequence"
		if s.ch < 0 {
			msg = "escape sequence not terminated"
		}
		s.error(offs, msg)
		return false
	}

	var x uint32
	for n > 0 {
		d := uint32(digitVal(s.ch))
		if d >= base {
			msg := fmt.Sprintf("illegal character %#U in escape sequence", s.ch)
			if s.ch < 0 {
				msg = "escape sequence not terminated"
			}
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

func (s *Scanner) scanString(ml bool) string {
	// '"' opening already consumed
	offs := s.offset - 1
	if ml {
		offs -= 1
	}

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
		case '\\':
			s.scanEscape('"')
		case '$':
			//
		}
	}

	return string(s.src[offs:s.offset])
}

func (s *Scanner) Scan() (pos token.Pos, tok token.Token, lit string) {
	// current token start
	pos = s.file.Pos(s.offset)

	if s.offset >= len(s.src) || s.ch == -1 {
		tok = token.EOF; return
	} else if s.bits.isCompoundLine() {
		switch tok, lit = s.scanCompoundLine(); tok {
		case token.DELEGATE, token.CLOSURE: break
		default: return
		}
	} else if s.bits.isCompoundString() {
		switch tok, lit = s.scanCompoundString(); tok {
		case token.DELEGATE, token.CLOSURE: break
		default: return
		}
	} else if /*s.offset == s.lineOffset*/false {
		// remove line preceeding spaces
		//s.skipUselessWhitespace(true)
		if s.ch == -1 || s.offset == len(s.src) {
			tok = token.EOF
			return
		}
	}

	if isLetter(s.ch) {
		lit = s.scanIdentifier()
		if len(lit) > 1 {
			switch tok = token.Lookup(lit); {
			case tok == token.BAREWORD || tok.IsKeyword():
				// ...
			default:
				s.error(s.offset, "unexpected token '"+tok.String()+"'")
			}
		} else {
			tok = token.BAREWORD
		}
		if s.bits&isCompoundCallIdent != 0 {
			s.bitsPop(isCompoundCallIdent)//s.bits &= ^isCompoundCallIdent
		}
		//s.skipUselessWhitespace(false)
		return
	}

	if '0' <= s.ch && s.ch <= '9' {
		tok, lit = s.scanNumber(false)
		return
	}

	// determine token value
	var ch = s.ch
	switch s.next(); ch {
	case '#':
		if s.bits.isCommentsOff() {
			tok = token.HASH
			lit = string(ch)
		} else {
			tok = token.COMMENT
			lit = s.scanComment()
			s.next() // discard '\n'
		}
	case '@':
		tok = token.AT
	case '|':
		tok = token.BAR
	case '!':
		tok = token.EXC
		if s.ch == '=' {
			tok = token.EXC_ASSIGN
			s.next()
		}
	case '?':
		tok = token.QUE
		if s.ch == '=' {
			tok = token.QUE_ASSIGN
			s.next()
		}
	case '%':
		tok = token.PERC
	case '+':
		tok = token.PLUS
		if s.ch == '=' {
			tok = token.ADD_ASSIGN
			s.next()
		}
	case '???': // different from ' ??? '
		tok = token.SELECT_PROP
	case '-':
		if s.ch == '-' { // "-->" => "-", "->"
			if s.readOffset < len(s.src) && s.src[s.readOffset] == '>' {
				tok, lit = token.BAREWORD, "-"
			} else {
				tok = token.MINUS
			}
		} else if s.ch == '=' { // -=
			tok = token.SUB_ASSIGN
			s.next()
			if s.ch == '+' { // -=+
				tok = token.SSH_ASSIGN
				s.next()
			}
		} else if s.ch == '+' { // -+
			s.next()
			if s.ch == '=' { // -+=
				tok = token.SAD_ASSIGN
				s.next()
			} else {
				tok = token.ILLEGAL
			}
		} else if s.ch == '>' {
			tok = token.SELECT_PROP
			s.next()
		} else if '0' <= s.ch && s.ch <= '9' {
			tok, lit = s.scanNumber(false)
			lit = "-" + lit // minus number
		} else {
			tok = token.MINUS
		}
	case '/':
		tok = token.PCON
	case '\\':
		tok, lit = token.ESCAPE, string(s.ch)
		s.next() // eat escaped char
	case '\'':
		tok = token.STRING
		if s.ch == '\'' {
			s.next()
			if s.ch == '\'' { // '''
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
		if s.bits&isCompoundString != 0 {
			tok = token.COMPOSED
			s.bitsPop(isCompoundString) //s.bits &= ^isCompoundString
			s.next() // take the ending '"'
		} else {
			tok = token.COMPOUND
			s.bitsPush(isCompoundString) //s.bits |= isCompoundString
		}

	case '*':
		tok = token.STAR
	case '$', '&':
		isDelegate := ch == '$'
		tok, ch = token.CLOSURE, rune(s.src[s.readOffset-1])
		switch {
		case ch == '/': tok = token.CLOSURE_R
		case ch == '.': tok = token.CLOSURE_D
		case ch == '@': tok = token.CLOSURE_A
		case ch == '|': tok = token.CLOSURE_B
		case ch == '<': tok = token.CLOSURE_L
		case ch == '^': tok = token.CLOSURE_U
		case ch == '*': tok = token.CLOSURE_S
		case ch == '-': tok = token.CLOSURE_M
		case ch == '+': tok = token.CLOSURE_P
		case ch == '?': tok = token.CLOSURE_Q
		case ch == '1': tok = token.CLOSURE_1
		case ch == '2': tok = token.CLOSURE_2
		case ch == '3': tok = token.CLOSURE_3
		case ch == '4': tok = token.CLOSURE_4
		case ch == '5': tok = token.CLOSURE_5
		case ch == '6': tok = token.CLOSURE_6
		case ch == '7': tok = token.CLOSURE_7
		case ch == '8': tok = token.CLOSURE_8
		case ch == '9': tok = token.CLOSURE_9
		case ch == '_': tok = token.CLOSURE__
		}
		if token.CLOSURE < tok {
			lit = string(ch)
			s.next() // eat special
		} else {
			var bits scanbits
			switch ch {
			case '(': bits = isCompoundCallParen
			case '{': bits = isCompoundCallBrace
			case ':': bits = isCompoundCallColonL
			default:  bits = isCompoundCallIdent
			}
			s.bitsPush(bits)
		}
		if isDelegate {
			tok = token.Token(token.DELEGATE + (tok - token.CLOSURE))
		}
	case '(':
		tok, lit = token.LPAREN, string(ch)
		if s.bits&isCompoundCallParen == 0 {
			s.bitsPush(isCompoundGroup)
		}
	case ')':
		tok, lit = token.RPAREN, string(ch)
		s.bitsPop(isCompoundGroup|isCompoundCallParen)
	case '???': // =>
		tok = token.SELECT_PROG1
	case '???': // ~>
		tok = token.SELECT_PROG2
	case '=':
		if s.ch == '>' { // =>
			tok = token.SELECT_PROG1
			s.next() // concume the '>'
		} else if s.ch == '+' {
			tok = token.SHI_ASSIGN
			s.next()
		} else {
			tok = token.ASSIGN
		}
	case '\n':
		tok = token.LINEND
		s.bitsPop(isCompoundLine)
	case '\t':
		if s.bits.canBeRecipe() && (s.lineOffset == s.offset-1) && !s.bits.is(
			isCompoundCallParen|isCompoundCallBrace|isCompoundCallColonL|isCompoundCallColonR|
			isCompoundGroup,
		) {
			tok, lit = token.RECIPE, string(ch)
			s.bitsPush(isCompoundLine)
		} else {
			tok, lit = token.SPACE, string(ch)
			for s.ch == '\t' || s.ch == ' ' {
				lit += string(s.ch)
				s.next()
			}
		}
	case ' ':
		tok, lit = token.SPACE, string(ch)
		for s.ch == '\t' || s.ch == ' ' {
			lit += string(s.ch)
			s.next()
		}
		return
	case ',':
		tok = token.COMMA
	case '~':
		if s.ch == '>' { // ~>
			tok = token.SELECT_PROG2
			s.next() // concume the '>'
		} else {
			tok = token.TILDE
		}
	case '.':
		if tok = token.DOT; s.ch == '.' {
			tok = token.DOTDOT
			s.next()
		} else if isDigit(s.ch) {
			if n := s.offset-2; n > -1 && unicode.IsSpace(rune(s.src[n])) { // skip xxx.1
				tok, lit = s.scanNumber(true)
			}
		}
	case '???':
		tok = token.SCO_ASSIGN
	case '???':
		tok = token.DCO_ASSIGN
	case ':':
		if s.ch == '=' {
			tok = token.SCO_ASSIGN
			s.next() // consume '='
		} else if s.bits.isCompoundCallColonR() {
			tok = token.RCOLON
			s.bitsPop(isCompoundCallColonL|isCompoundCallColonR)
		} else if s.bits.isCompoundCallColonL() {
			tok = token.LCOLON
			s.bits |= isCompoundCallColonR
		} else if s.ch == ':' {
			tok = token.COLON2
			s.next() // consume the second ':'
			if s.ch == '=' {
				tok = token.DCO_ASSIGN
				s.next() // consume '='
			}
		} else {
			tok = token.COLON
		}
	case ';':
		tok = token.SEMICOLON
	case '[':
		tok = token.LBRACK
	case ']':
		tok = token.RBRACK
	case '{':
		tok = token.LBRACE
	case '}':
		tok = token.RBRACE
		s.bitsPop(isCompoundCallBrace) //s.bits &= ^isCompoundCallBrace
	default:
		// next reports unexpected BOMs - don't repeat
		if ch != bom {
			s.error(s.file.Offset(pos), fmt.Sprintf("illegal character %#U", ch))
		}
		tok = token.ILLEGAL
		lit = string(ch)
	}

	// eat consequence spaces
	if /*s.bits.is(isCompoundLine|isCompoundString|isCompoundCallParen|isCompoundCallBrace|isCompoundCallColon)*/false {
		//s.skipUselessWhitespace(false)
	}
	return
}
