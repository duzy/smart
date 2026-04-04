//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"path/filepath"
	"unicode/utf8"
	"unicode"
	"strings"
	"fmt"
)

const (
	isStrcompLine scanbits = 1 << iota    // 0000000000000001
	isStrcompString  // 0000000000000010 "...."
	isCall           // 0000000000000100 $.....
	isCallParen      // 0000000000001000 $(...)              8
	isCallBrace      // 0000000000010000 ${...}             16
	isCallColonL     // 0000000000100000 $:....             32
	isCallColonR     // 0000000001000000 $:...:             64
	isGroup          // 0000000010000000 (...)             128
	isBrace          // 0000000100000000 {...}             256
	isBraceRaw       // 0000001000000000                   512
	isBracedPlain    // 0000010000000000                  1024
	isRecipes        // 0000100000000000                  2048
	isRecipeTab      // 0001000000000000 \t                4096
	isHashValid      // 0010000000000000 scan '#' as HASH token (commentsOff)
	isMaximumBit     // 1000000000000000                   8192
)

type scanstate struct {
	ch         rune  // current character
	offset     int   // character offset
	offsetRead int   // reading offset (position after current character)
	offsetLine int   // current line offset
	bitss []scanbits // scan bits stack
	bits    scanbits // scan bits
}

func (s *scanstate) ch_bytes() int { return s.offsetRead - s.offset }

func (s *scanstate) String() string {
	var t string
	switch s.ch {
	case '\n': t = "\\n"
	case '}': t = "\\}"
	default: t = string(s.ch)
	}
	return fmt.Sprintf("{=scan %s {%v %v %v} %016b %016b}",
		t, s.offsetLine, s.offset, s.offsetRead, s.bitss, s.bits)
}

func (s *scanstate) push(bits scanbits) (prev scanbits) {
	if prev = s.bits; prev != 0 {
		s.bitss = append(s.bitss, prev) // &^ isLineFeed
	}
	s.bits = bits
	return
}
func (s *scanstate) pop(bits scanbits) (prev scanbits) {
	if prev = s.bits ; bits == 0 || (s.bits&bits != 0) {
		if i := len(s.bitss); 0 == i {
			s.bits = 0
		} else {
			s.bits = s.bitss[i-1] //&^ isLineFeed
			s.bitss = s.bitss[0:i-1]
		}
	}
	return
}

func (s *scanstate) setBits(bits scanbits) (prev scanbits) {
	prev = s.bits
	s.bits = bits
	return
}

func (s *scanstate) addBits(bits scanbits) (prev scanbits) {
	prev = s.bits
	s.bits |= bits
	return
}

func (s *scanstate) remBits(bits scanbits) (prev scanbits) {
	prev = s.bits
	s.bits &^= bits
	return
}

func (s *scanstate) commentsOff() scanbits { return s.addBits(isHashValid) }
func (s *scanstate) recipes(v bool) {
	var bits = s.bits
	if v { bits |= isRecipes } else { bits &^= isRecipes }
	s.bits = bits
}

func (s *scanstate) canRecipe() (res bool) {
	if t := s.bits; (s.offsetLine == s.offset-1) && t.canRecipe() {
		res = !t.is(isCallParen|isCallBrace|isCallColonL|isCallColonR|isGroup)
	}
	return
}

func (s *scanstate) bit(bits scanbits) (res bool) {
	if res = s.bits&bits != 0; !res {
		for i := len(s.bitss)-1; 0 <= i; i -= 1 {
			if res = s.bitss[i]&bits != 0; res { break }
		}
	}
	return
}

const bom = 0xFEFF // byte order mark, only permitted as very first character

// A scanner holds the scanner's internal state while processing
// a given text.  It can be allocated as part of another data
// structure but must be initialized via Init before use.
//
// (See go.token)
type scanner struct { // immutable state
	file *tokfile     // source file handle
	dir  string       // directory portion of file.Name()
	src  []byte       // source
	mode scanmode     // scanning mode
	scanstate
}

// Read the next Unicode char into s.ch, s.ch < 0 means end-of-file.
func (s *scanner) next(ctx Context) {
	var newline = s.ch == '\n'

	if s.offsetRead < len(s.src) {
		if s.offset = s.offsetRead; s.ch == '\n' {
			s.offsetLine = s.offset
			s.file.AddLine(s.offset)
		}
		var w int
		s.ch, w = s.pick(ctx, s.offsetRead)
		s.offsetRead += w
	} else {
		if s.offset = len(s.src); s.ch == '\n' {
			s.offsetLine = s.offset
			s.file.AddLine(s.offset)
		}
		s.ch = -1 // eof
	}

	if newline && s.ch == '\t' {
		s.bits |= isRecipeTab
		// s.bits &^= isLineFeed
	} else {
		// s.bits &^= isLineFeed | isRecipeTab
		s.bits &^= isRecipeTab
	}
}

func (s *scanner) pickNext(ctx Context) (ch rune, w int) {
	if n := s.offsetRead + 1; n < len(s.src) { ch, w = s.pick(ctx, n) }
	return
}

func (s *scanner) pick(ctx Context, offset int) (ch rune, w int) {
	switch ch, w = rune(s.src[offset]), 1; {
	case ch == 0:
		debug(pc(ctx,s.pos(offset)), "illegal character NUL", trace{})
	case ch >= 0x80: // Non ASCII
		if ch, w = utf8.DecodeRune(s.src[offset:]); ch == utf8.RuneError && w == 1 {
			debug(pc(ctx,s.pos(offset)), "illegal UTF-8 encoding", trace{})
		} else if ch == bom && offset > 0 {
			debug(pc(ctx,s.pos(offset)), "illegal byte order mark", trace{})
		} else if w > 1 {
			// CRITICAL FIX: Register the multibyte span instantly during parsing!
			// We pass the byte offset and the "extra" bytes (width - 1).
			s.file.AddSpan(offset, w - 1)
		}
	}
	return
}

// A mode value is a set of flags (or 0).
// They control scanner behavior.
type scanmode uint
type scanbits uint
func (bits scanbits) is(t scanbits)     bool { return bits&t != 0 }
func (bits scanbits) isCall()           bool { return bits&isCall != 0 }
func (bits scanbits) isCallZero()       bool { return bits&isCall != 0 && bits&(isCallParen|isCallBrace|isCallColonL) == 0 }
func (bits scanbits) isCallParen()      bool { return bits&isCallParen != 0 }
func (bits scanbits) isCallBrace()      bool { return bits&isCallBrace != 0 }
func (bits scanbits) isCallColonL()     bool { return bits&isCallColonL != 0 }
func (bits scanbits) isCallColonR()     bool { return bits&isCallColonR != 0 }
func (bits scanbits) isCommentsOff()    bool { return bits&isHashValid != 0 }
func (bits scanbits) isBrace()          bool { return bits&isBrace != 0 }
func (bits scanbits) isBraceRaw()       bool { return bits&isBraceRaw != 0 }
func (bits scanbits) isBracedPlain()    bool { return bits&isBracedPlain != 0 }
func (bits scanbits) isGroup()          bool { return bits&isGroup != 0 }
func (bits scanbits) isStrcompLine()    bool { return bits&isStrcompLine != 0 }
func (bits scanbits) isStrcompString()  bool { return bits&isStrcompString != 0 }
func (bits scanbits) canRecipe()        bool { return bits&(isRecipeTab|isRecipes) != 0 }

func IsLetter(r rune) bool {
	return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || r == '_' || r >= 0x80 && unicode.IsLetter(r)
}

func IsDigit(r rune) bool {
	return unicode.IsDigit(r) //('0' <= r && r <= '9') || (r >= 0x80 && unicode.IsDigit(r))
}

func IsDigits(s string) bool {
    return strings.IndexFunc(s, func(r rune) bool { return !IsDigit(r) }) < 0
}

// punctuation used as non-terminator
func IsUntermPunct(r rune) bool {
	// Most chars accepted in URL (RFC3986)
	return r == '@' || r == '+' /* || r == '-' || r == '.' || r == '/' */;
}

func IsDatetimeTerminator(r rune) bool {
	return  r == ' ' || r == '\t' || r == '\n' || r == '\r' ||
		r == '(' || r == ')' || r == '{' || r == '}' ||
		r == '$' || r == '#' || r == '\\'
}

func IsIdentifier(r rune) bool {
	return IsLetter(r) || IsDigit(r) || IsUntermPunct(r) //|| r == '\\'
}

func (s *scanner) init(ctx Context, file *tokfile, src []byte, mode scanmode) {
	// Explicitly initialize all fields since a scanner may be reused.
	if file.Size() != len(src) {
		panic(fmt.Sprintf("file size (%d) does not match src len (%d)", file.Size(), len(src)))
	}

	s.file = file
	s.dir, _ = filepath.Split(file.Name())
	s.src = src
	s.mode = mode

	s.ch = ' '
	s.offset = 0
	s.offsetRead = 0
	s.offsetLine = 0
	s.bits = 0
	s.bitss = nil

	// The BOM at file beginning will be discarded.
	if s.next(ctx); s.ch == bom { s.next(ctx) }
}

func (s *scanner) pos(offs ...int) Position {
	if 0 < len(offs) {
		return s.file.Position(s.file.Pos(offs[0]))
	}
	return s.file.Position(s.file.Pos(s.offset))
}

func (s *scanner) scanComment(ctx Context) (res string) {
	for s.ch == ' '  || s.ch == '\t' { s.next(ctx) } // skip preceding spaces

	var offs = s.offset
	for s.ch != '\n' && s.ch != -1 { s.next(ctx) }

	// We should intern identifiers, words, and numbers. We should not intern
	// comments (they are long and rarely repeated) to avoid bloating the pool.
	return string(s.src[offs:s.offset])
}

func (s *scanner) scanIdentifier(ctx Context) string {
	var offs = s.offset
	for IsIdentifier(s.ch) {
		if s.next(ctx); /* s.ch == '-' */false { // Looking for '->'
			var n = s.offset + 1 // No need UTF8 decoding!
			if n < len(s.src) && rune(s.src[n]) == '>' { break }
		}
	}
	return internBytes(s.src[offs:s.offset])
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9': return int(ch - '0')
	case 'a' <= ch && ch <= 'f': return int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F': return int(ch - 'A' + 10)
	}
	return 16 // larger than any legal digit val
}

func (s *scanner) scanMantissa(ctx Context, base int) {
	if digitVal(s.ch) < base { // first digit
		s.next(ctx)
		if true {
			for digitVal(s.ch) < base { s.next(ctx) }
		} else {
			// NOTE: disable '_' number separaters as ParseInt not support it and
			//       it's not recoverable from ints back to strings.
			for s.ch == '_' || digitVal(s.ch) < base {
				if s.ch == '_' {
					if s.next(ctx); s.ch == '_' {
						debug(pc(ctx,s), "invalid digit group", trace{})
						break
					}
				} else {
					s.next(ctx)
				}
			}
		}
	}
}

func (s *scanner) scanDatetime(ctx Context) (tok token) {
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
		debug(pc(ctx,s.pos(o+5)), "bad month", trace{}); goto exit
	}
	if ch = s.src[o+6]; ch < '0' || '9' < ch {
		debug(pc(ctx,s.pos(o+6)), "bad month", trace{}); goto exit
	}

	// month-day range is 01-28, 01-29, 01-30, 01-31 based on month/year
	if ch = s.src[o+8]; ch < '0' && '3' < ch {
		debug(pc(ctx,s.pos(o+8)), "bad month day", trace{}); goto exit
	}
	if ch = s.src[o+9]; ch < '0' || '9' < ch {
		debug(pc(ctx,s.pos(o+9)), "bad month day", trace{}); goto exit
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
		debug(pc(ctx,s.pos(o)), "bad time", trace{}); goto exit
	}

	if l-o < 9 || s.src[o+2] != ':' || s.src[o+5] != ':' {
		debug(pc(ctx,s.pos(o)), "illegal time", trace{}); goto exit
	}

checkTime:
	// hour range is 00-23
	if ch = s.src[o+0]; ch < '0' || '2' < ch {
		debug(pc(ctx,s.pos(o+0)), "bad hour", trace{}); goto exit
	}
	if ch = s.src[o+1]; ch < '0' || '9' < ch || ('3' < ch && s.src[o] == '2') {
		debug(pc(ctx,s.pos(o+1)), "bad hour", trace{}); goto exit
	}

	// minute range is 00-59
	if ch = s.src[o+3]; ch < '0' || '5' < ch {
		debug(pc(ctx,s.pos(o+3)), "bad minute", trace{}); goto exit
	}
	if ch = s.src[o+4]; ch < '0' || '9' < ch {
		debug(pc(ctx,s.pos(o+4)), "bad minute", trace{}); goto exit
	}

	// second ranges are 00-59 00-58, 00-59, 00-60 based on leap second rules
	if ch = s.src[o+6]; ch < '0' || '5' < ch {
		debug(pc(ctx,s.pos(o+6)), "bad second", trace{}); goto exit
	}
	if ch = s.src[o+7]; ch < '0' || '9' < ch {
		debug(pc(ctx,s.pos(o+7)), "bad second", trace{}); goto exit
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
				debug(pc(ctx,s.pos(o)), "bad secfrac", trace{}); goto exit
			}
		}
	} else if ch == '+' || ch == '-' {
		o += 9; goto checkNumOffset // consume 00:00:00+
	} else {
		debug(pc(ctx,s.pos(o)), "bad time", trace{}); goto exit
	}

checkNumOffset:
	if ch = s.src[o+2]; ch != ':' {
		debug(pc(ctx,s.pos(o+2)), "bad offset", trace{}); goto exit
	}

	// hour range is 00-23
	if ch = s.src[o+0]; ch < '0' || '2' < ch {
		debug(pc(ctx,s.pos(o+0)), "bad hour", trace{}); goto exit
	}
	if ch = s.src[o+1]; ch < '0' || '9' < ch || ('3' < ch && s.src[o] == '2') {
		debug(pc(ctx,s.pos(o+1)), "bad hour", trace{}); goto exit
	}

	// minute range is 00-59
	if ch = s.src[o+3]; ch < '0' || '5' < ch {
		debug(pc(ctx,s.pos(o+3)), "bad minute", trace{}); goto exit
	}
	if ch = s.src[o+4]; ch < '0' || '9' < ch {
		debug(pc(ctx,s.pos(o+4)), "bad minute", trace{}); goto exit
	}

	o += 5 // consume 00:00

success:
	for i := s.offset; i < o; i++ { s.next(ctx) }
	switch {
	case hasDate && hasTime: tok = DATETIME
	case hasDate && !hasTime: tok = DATE
	case !hasDate && hasTime: tok = TIME
	default: tok = ILLEGAL
	}

exit:
	return
}

func (s *scanner) scanNumber(ctx Context, seenDecimalPoint bool) (token, string) {
	// digitVal(s.ch) < 10
	offs := s.offset
	tok := INTEGER

	if seenDecimalPoint {
		offs--
		tok = FLOATING // CRITICAL FIX: Was FLOAT (which is a keyword!)
		s.scanMantissa(ctx, 10)
		goto exponent
	}

	if t := s.scanDatetime(ctx); t != ILLEGAL {
		tok = t; goto exit
	}

	if s.ch == '0' {
		// int or float
		offs := s.offset
		s.next(ctx)
		if s.ch == 'b' || s.ch == 'B' {
			// binary int
			s.next(ctx)
			s.scanMantissa(ctx, 2)
			tok = BINARY
			if s.offset-offs <= 2 {
				// only scanned "0b" or "0B"
				debug(pc(ctx,offs), "illegal binary number", trace{})
			}
		} else if s.ch == 'x' || s.ch == 'X' {
			// hexadecimal int
			s.next(ctx)
			s.scanMantissa(ctx, 16)
			tok = HEXADECIMAL
			if s.offset-offs <= 2 {
				// only scanned "0x" or "0X"
				debug(pc(ctx,offs), "illegal hexadecimal number", trace{})
			}
		} else {
			// octal int or float
			seenDecimalDigit := false
			s.scanMantissa(ctx, 8)
			if s.ch == '8' || s.ch == '9' {
				// illegal octal int or float
				seenDecimalDigit = true
				s.scanMantissa(ctx, 10)
			}
			if s.ch == '.' || s.ch == 'e' || s.ch == 'E' || s.ch == 'i' {
				goto fraction
			}
			// octal int
			if seenDecimalDigit {
				debug(pc(ctx,offs), "illegal octal number", trace{})
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
	s.scanMantissa(ctx, 10)

fraction:
	if s.ch == '.' {
		// Safety check 1: Must be followed by a digit. Prevents 1.$2 from breaking.
		if n := s.offset + 1; n < len(s.src) {
			if ch := rune(s.src[n]); !IsDigit(ch) { 
				goto exit
			}
		} else {
			goto exit
		}
		
		// Safety check 2 (The "+2 Hack"): Must have a second character after the dot 
		// that is ALSO a digit (or 'e'/'E'). This intentionally prevents 1-digit 
		// decimals (like .0 or .4) from being floats, forcing them to parse as 
		// structural qualwords for version strings (e.g., 25.4 or 25.4.0).
		if n := s.offset + 2; n < len(s.src) {
			ch := rune(s.src[n])
			if !IsDigit(ch) && ch != 'e' && ch != 'E' {
				goto exit
			}
		} else {
			goto exit
		}

		tok = FLOATING
		s.next(ctx)
		s.scanMantissa(ctx, 10)
	}

exponent:
	if s.ch == 'e' || s.ch == 'E' {
		tok = FLOATING // CRITICAL FIX: Was FLOAT (which is a keyword!)
		s.next(ctx)
		if s.ch == '-' || s.ch == '+' {
			s.next(ctx)
		}
		s.scanMantissa(ctx, 10)
	}

	/*
	if s.ch == 'i' {
		tok = IMAG
		s.next(ctx)
	} */

exit:
	return tok, internBytes(s.src[offs:s.offset])
}

func (s *scanner) scanEscape(ctx Context, quote rune) bool {
	var n int
	var base, max uint32
	var offs = s.offset
	switch s.ch {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '$', quote:
		s.next(ctx)
		return true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		n, base, max = 3, 8, 255
	case 'x':
		s.next(ctx)
		n, base, max = 2, 16, 255
	case 'u':
		s.next(ctx)
		n, base, max = 4, 16, unicode.MaxRune
	case 'U':
		s.next(ctx)
		n, base, max = 8, 16, unicode.MaxRune
	case '\n':
		s.next(ctx)
	default:
		var msg = "unknown escape sequence"
		if s.ch < 0 { msg = "escape sequence not terminated" }
		debug(pc(ctx,offs), msg, trace{})
		return false
	}

	var x uint32
	for n > 0 {
		d := uint32(digitVal(s.ch))
		if d >= base {
			var msg = fmt.Sprintf("illegal character %#U in escape sequence", s.ch)
			if s.ch < 0 { msg = "escape sequence not terminated" }
			debug(pc(ctx,offs), msg, trace{})
			return false
		}
		x = x*base + d
		s.next(ctx)
		n--
	}

	if x > max || 0xD800 <= x && x < 0xE000 {
		debug(pc(ctx,offs), "escape sequence is invalid Unicode code point", trace{})
		return false
	}

	return true
}

func (s *scanner) scanStrliting(ctx Context, ml bool) string {
	// '\'' opening already consumed
	offs := s.offset - 1
	if ml { offs -= 1 }

	for s.offsetRead < len(s.src) {
		ch := s.ch
		if (!ml && ch == '\n') || ch < 0 { // if ch < 0 {
			debug(pc(ctx,offs), "raw string literal not terminated", trace{})
			break
		}
		if ch == '\\' { s.next(ctx) } // escapes
		s.next(ctx)
		if ch == '\'' {
			if !ml { break }
			if s.ch == '\'' {
				if s.next(ctx); s.ch == '\'' {
					s.next(ctx)
					break
				}
			}
		}
	}

	return internBytes(s.src[offs+1:s.offset-1])
}

func (s *scanner) scanString(ctx Context, ml bool) string {
	// '"' opening already consumed
	offs := s.offset - 1
	if ml { offs -= 1 }

	for s.offsetRead < len(s.src) {
		ch := s.ch
		if (!ml && ch == '\n') || ch < 0 {
			debug(pc(ctx,offs), "string literal not terminated", trace{})
			break
		}
		s.next(ctx)
		if ch == '"' {
			if !ml {
				break
			}
			if s.ch == '"' {
				if s.next(ctx); s.ch == '"' {
					s.next(ctx)
					break
				}
			}
		}
		switch ch {
		case '\\': s.scanEscape(ctx, '"')
		case '$': //
		}
	}
	return internBytes(s.src[offs:s.offset])
}

func (s *scanner) scanStrcomp(ctx Context, q rune) (tok token, lit string) {
	if q != 0 && s.ch == q {
		switch q {
		case '"':
			tok = COMPOSED
			s.pop(isStrcompString)
			s.next(ctx) // take the ending '"'
			return
		case '}':
			tok = RBRACE
			s.pop(isBracedPlain)
			s.next(ctx) // take the ending '"'
			return
		}
	}

	var offs = s.offset

	switch s.ch {
	case '{': // mistaken strcomp string terminated with line feed
		if q == '}' {
			tok = LBRACE
			s.next(ctx)
			return
		}
	case '\n': // mistaken strcomp string terminated with line feed
		tok = LINEND
		s.pop(isStrcompString|isStrcompLine)
		if s.next(ctx); s.ch != '\t' { s.bits &^= isRecipes }
		return
	case '\\':
		if s.next(ctx); q == 0 { // skim the \ character
			tok, lit = ESCAPE, string(s.ch)
			s.next(ctx) // the escaped character
			if s.bits&isRecipes != 0 && s.ch == '\t' {
				s.next(ctx) // skip escaped recipe-tab
			}
		} else if s.scanEscape(ctx, q) {
			tok, lit = ESCAPE, string(s.src[offs+1:s.offset])
		} else {
			tok, lit = ILLEGAL, string(s.src[offs:s.offset])
			debug(pc(ctx,offs), "illegal strcomp escape %#U", s.ch, trace{})
			s.next(ctx) // discard
		}
		return
	case '&', '$': // Escapes '&', '$', but '&&' or '$$' is not escaped.
		if n := s.offset+1; n < len(s.src) && rune(s.src[n]) == s.ch {
			s.next(ctx) //! the first & or $
			s.next(ctx) //! the second & or $
		} else if s.ch == '$' {
			return DELEGATE, lit
		} else {
			return CLOSURE, lit
		}
	}

rawloop:
	for ; s.offsetRead < len(s.src) ; s.next(ctx) {
		switch s.ch {
		case '\\', '\n', '$', '&', q: break rawloop
		case '{':
			if i := s.offsetRead; i < len(s.src) && s.src[i] == '=' {
				// note(ctx, "%s", s.src[offs:i+1]).debug(5)
				return LBRACE, lit
			}
		}
	}

	tok, lit = RAW, internBytes(s.src[offs:s.offset])
	return
}

func (s *scanner) scan(ctx Context) (pos Pos, tok token, lit string) {
	switch pos = s.file.Pos(s.offset) ; {
	case s.offset >= len(s.src) || s.ch == -1: return pos, EOF, ""
	case s.bits.isBracedPlain()  : tok, lit = s.scanStrcomp(ctx, '}')
	case s.bits.isStrcompString(): tok, lit = s.scanStrcomp(ctx, '"')
	case s.bits.isStrcompLine()  : tok, lit = s.scanStrcomp(ctx, 0)
	}

	if tok != 0 {
		switch tok {
		case CLOSURE, DELEGATE, LBRACE:
		default: return
		}
		switch s.ch {
		case '$', '&', '{':
		default:
			debug(pc(ctx,s), "unexpected '%s'", string(s.src[s.offset:]), trace{})
		}
	}

	if IsDigit(s.ch) { // '0' <= s.ch && s.ch <= '9'
		tok, lit = s.scanNumber(ctx, false)
		return
	}

	if IsLetter(s.ch) {
		lit = s.scanIdentifier(ctx)
		// CRITICAL FIX: Downgrade keywords to WORD if they are immediately followed by a dash.
		// This prevents rule targets like `configure-input:` from being hijacked!
		if len(lit) > 1 && s.ch != '/' && s.ch != '.' && s.ch != '~' && s.ch != '-' {
			if tok = lookup_keyword(lit) ; !tok.is_keyword() && tok != WORD {
				debug(pc(ctx,s), "unexpected token '%s' %s", tok, lit, trace{})
			}
		} else {
			tok = WORD
		}
		if s.bits.isCallZero() { s.pop(/*isCall*/0) }
		return
	}

	var ch, offs = s.ch, s.offset

	s.next(ctx)

	if s.bits.isBraceRaw() {
		switch ch {
		case '$':
			if s.ch == '$' {
				s.next(ctx)
				return pos, RAW, string(ch)
			} else if false {
				debug(pc(ctx,s.pos(offs)), "%s %s", string(ch), string(s.ch))
			}
		case '&':
			if s.ch == '&' {
				s.next(ctx)
				return pos, RAW, string(ch)
			} else if false {
				debug(pc(ctx,s.pos(offs)), "%s %s", string(ch), string(s.ch))
			}
		case '\\':
			if tok = ESCAPE; IsDigit(s.ch) {
				_, lit = s.scanNumber(ctx, false)
			} else {
				lit = string(s.ch)
				s.next(ctx) // escape a single char
			}
			return
		case '{':
			s.push(isBraceRaw)
			return pos, RAW, string(ch)
		case '}':
			t := s.bits.isBrace()
			s.pop(isBrace|isBraceRaw)
			if t {
				return pos, RBRACE, ""
			} else {
				return pos, RAW, string(ch)
			}
		default:
			return pos, RAW, string(ch)
		}
	}

	switch ch {
	case '#':
		if s.bits.isCommentsOff() {
			tok = HASH
			lit = string(ch)
		} else {
			tok = COMMENT
			lit = s.scanComment(ctx)
			s.next(ctx) // discard '\n'
		}
	case '!':
		if tok = EXC; s.ch == '=' {
			tok = ASSIGN_EXC
			s.next(ctx)
		}
	case '?':
		if tok = QUE; s.ch == '=' {
			tok = ASSIGN_QUE
			s.next(ctx)
		}
	case '+':
		if tok = PLUS; s.ch == '=' {
			tok = ASSIGN_ADD
			s.next(ctx)
		}
	case '-':
		if s.ch == '-' { // "-->" => "-", "->"
			if s.offsetRead < len(s.src) && s.src[s.offsetRead] == '>' {
				tok, lit = WORD, "-"
			} else {
				tok = MINUS
			}
		} else if s.ch == '=' { // -=
			tok = ASSIGN_SUB
			s.next(ctx)
			if s.ch == '+' { // -=+
				tok = ASSIGN_SSH
				s.next(ctx)
			}
		} else if s.ch == '+' { // -+
			s.next(ctx)
			if s.ch == '=' { // -+=
				tok = ASSIGN_SAD
				s.next(ctx)
			} else {
				tok = ILLEGAL
			}
		} else if s.ch == '>' {
			tok = SELECT_PROP
			s.next(ctx)
		} else if '0' <= s.ch && s.ch <= '9' {
			tok, lit = s.scanNumber(ctx, false)
			lit = "-" + lit // minus number
		} else {
			tok = MINUS
		}
	case '\\':
		tok, lit = ESCAPE, string(s.ch)
		s.next(ctx) // eat escaped char
		if s.bits&isRecipes != 0 && s.ch == '\t' {
			s.next(ctx) // skip escaped recipe-tab
		}
	case '\'':
		if tok = STRING; s.ch == '\'' {
			if s.next(ctx); s.ch == '\'' { // '''
				lit = s.scanStrliting(ctx, true)
			} else if offs := s.offset - 2; false {
				lit = string(s.src[offs:s.offset])
			} else {
				lit = "" // empty string ''
			}
		} else {
			lit = s.scanStrliting(ctx, false)
		}
	case '"':
		if s.bits.isStrcompString() {
			debug(pc(ctx,s.pos(offs)), "composed", trace{})
		} else {
			tok = STRCOMP
			s.push(isStrcompString)
		}
	case '$', '&':
		if ch == '&' { tok = CLOSURE } else { tok = DELEGATE }
		if ch = rune(s.src[s.offset]); ch == '(' || ch == '{' {
			s.push(isCall)
		} else if false {
			s.push(isCall /* | isCallZero */)
		}
	case '(':
		tok = LPAREN
		if s.bits.isCallZero() { s.bits |= isCallParen } else { s.push(isGroup) }
	case ')':
		tok = RPAREN
		t := isCallParen|isGroup
		if s.bits&t == 0 {
			if n := len(s.bitss); n > 0 {
				if b := s.bitss[n-1]; b&t != 0 {
					// Fix nested right-paren in recipes
					s.bits, s.bitss = b, s.bitss[0:n-1]
					goto poprparen
				}
			}
			debug(pc(ctx,s.pos(offs)), "unexpected right-paren, %016b %016b", s.bits, s.bitss, trace{})
		}
		poprparen: s.pop(t)
	case '{':
		tok = LBRACE
		if s.bits.isCallZero() { s.bits |= isCallBrace } else { s.push(isBrace) }
	case '}':
		tok = RBRACE
		t := isCallBrace|isBrace
		if s.bits&t == 0 {
			if n := len(s.bitss); n > 0 {
				if b := s.bitss[n-1]; b&t != 0 {
					// Fix nested right-brace in recipes
					s.bits, s.bitss = b, s.bitss[0:n-1]
					goto poprbrace
				}
			}
			debug(pc(ctx,s.pos(offs)), "unexpected right-brace, %016b %016b", s.bits, s.bitss, trace{})
		}
		poprbrace: s.pop(t)
	case '=':
		if s.ch == '>' { // =>
			tok = SELECT_PROG1
			s.next(ctx) // concume the '>'
		} else if s.ch == '+' {
			tok = ASSIGN_SHI
			s.next(ctx)
		} else {
			tok = ASSIGN
		}
	case ' ', '\t': // ASCII 32, 9
		if ch == '\t' && s.canRecipe() {
			tok, lit = RECIPE, string(ch)
			s.push(isStrcompLine)
		} else {
			for s.ch == ' ' || s.ch == '\t' { s.next(ctx) }
			tok, lit = SPACE, internBytes(s.src[offs:s.offset])
		}
	case '~':
		if s.ch == '>' { s.next(ctx) // concume the '>' // ~>
			tok = SELECT_PROG2
		} else {
			tok = TILDE
		}
	case '.':
		if tok = DOT; s.ch == '.' { s.next(ctx) // consume the second '.'
			tok = DOTDOT
		} else if IsDigit(s.ch) {
			if n := s.offset-2; n > -1 && unicode.IsSpace(rune(s.src[n])) { // skip xxx.1
				tok, lit = s.scanNumber(ctx, true)
			}
		}
	case ':':
		if s.ch == '=' { s.next(ctx) // consume '='
			tok = ASSIGN_CO1
		} else if s.ch == ':' { s.next(ctx) // consume the second ':'
			if s.ch == '=' { s.next(ctx) // consume '='
				tok = ASSIGN_CO2
			} else {
				tok = DOLON
			}
		} else {
			tok = COLON
		}
	case '*':
		switch s.ch {
		case '*':
			s.next(ctx) // consume the second '*'
			tok = DAST
		case '?':
			s.next(ctx) // consume the '?'
			tok = ASTQ
		default:
			tok = SAST
		}
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
		tok = ASSIGN_CO1
	case '⩴':
		tok = ASSIGN_CO2
	case ';':
		if s.ch == '=' { tok = ASSIGN_SC1 ; s.next(ctx) } else
		if s.ch == ':' { tok = SOLON      ; s.next(ctx)
			if s.ch == '=' { tok = ASSIGN_CO3 ; s.next(ctx) }
		} else {
			tok = SEMICOLON
		}
	case '^':
		tok = CARET
	case '[':
		tok = LBRACK
	case ']':
		tok = RBRACK
	case '<':
		tok = LANGLE
	case '>':
		tok = RANGLE
	case '⟨':
		tok = Lchevron
	case '⟩':
		tok = Rchevron
	case '⌜':
		tok = Ltop_corner
	case '⌝':
		tok = Rtop_corner
	case '⌞':
		tok = Lbot_corner
	case '⌟':
		tok = Rbot_corner
	case '‹':
		tok = Lsing_guil
	case '›':
		tok = Rsing_guil
	case '«':
		tok = Lguillemet
	case '»':
		tok = Rguillemet
	case '\n': // ASCII 10, 13⇒\r
		tok = LINEND
		if s.pop(isStrcompLine); s.ch != '\t' { s.bits &^= isRecipes }
	default:
		// next reports unexpected BOMs - don't repeat
		if ch != bom {
			debug(pc(ctx,s.pos(s.file.Offset(pos))), "illegal %#U", ch, trace{})
		} else {
			tok = ILLEGAL
			lit = string(ch)
		}
	}
	return
}
