//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package scanner

import (
	"path/filepath"
	"unicode"
	"unicode/utf8"
        "fmt"
        "github.com/duzy/smart/token"
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
	rdOffset   int  // reading offset (position after current character)
	lineOffset int  // current line offset
        parenDepth int  // number of nested parentheses
        sepAware   bool // spaces as separator
        lineSignificant bool // line is significant (non-empty)

	// public state - ok to modify
	ErrorCount int // number of errors encountered
}

const bom = 0xFEFF // byte order mark, only permitted as very first character

// Read the next Unicode char into s.ch.
// s.ch < 0 means end-of-file.
//
func (s *Scanner) next() {
	if s.rdOffset < len(s.src) {
		s.offset = s.rdOffset
		if s.ch == '\n' {
			s.lineOffset = s.offset
			s.file.AddLine(s.offset)
		}
		r, w := rune(s.src[s.rdOffset]), 1
		switch {
		case r == 0:
			s.error(s.offset, "illegal character NUL")
		case r >= 0x80:
			// not ASCII
			r, w = utf8.DecodeRune(s.src[s.rdOffset:])
			if r == utf8.RuneError && w == 1 {
				s.error(s.offset, "illegal UTF-8 encoding")
			} else if r == bom && s.offset > 0 {
				s.error(s.offset, "illegal byte order mark")
			}
		}
		s.rdOffset += w
		s.ch = r
	} else {
		s.offset = len(s.src)
		if s.ch == '\n' {
			s.lineOffset = s.offset
			s.file.AddLine(s.offset)
		}
		s.ch = -1 // eof
	}
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

const (
	ScanComments    Mode = 1 << iota // return comments as COMMENT tokens
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
	s.rdOffset = 0
	s.lineOffset = 0
	s.ErrorCount = 0

	s.next()
	if s.ch == bom {
		s.next() // ignore BOM at file beginning
	}
}

func (s *Scanner) error(offs int, msg string) {
	if s.err != nil {
		s.err(s.file.Position(s.file.Pos(offs)), msg)
	}
	s.ErrorCount++
}

func (s *Scanner) skipWhitespace(lf bool) {
	for s.ch == ' ' || s.ch == '\t' || s.ch == '\r' || s.ch == '\\' || (s.ch == '\n' && (s.parenDepth > 0 || lf)) {
                if s.ch == '\\' {
                        if s.next(); s.ch != '\n' {
				s.error(s.offset, fmt.Sprintf("illegal escape %#U", s.ch))
                                return
                        }
                }
		s.next()
	}
}

func (s *Scanner) scanComment() string {
	// initial '#' already consumed
	offs := s.offset - 1 // position of initial '#'

        s.next()
        for s.ch != '\n' && s.ch >= 0 {
                s.next()
        }

        if offs == s.lineOffset {
                // comment starts at the beginning of the current line
                //s.interpretLineComment(s.src[offs:s.offset])
        }

	return string(s.src[offs:s.offset])
}

func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= 0x80 && unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9' || ch >= 0x80 && unicode.IsDigit(ch)
}

func isDatetimeTerminator(ch rune) bool {
        return  ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || 
                ch == '(' || ch == ')' || ch == '$'
}

func (s *Scanner) scanIdentifier() string {
	offs := s.offset
	for isLetter(s.ch) || isDigit(s.ch) || s.ch == '-' || s.ch == '/' {
		s.next()
	}
	return string(s.src[offs:s.offset])
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
                        ch := s.ch
                        s.next()
                        if ch == '_' && s.ch == '_' {
                                s.error(s.offset-1, "invalid digit group")
                                break
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
        //fmt.Printf("datetime: %v\n", string(s.src[s.offset:o]))
        for i := s.offset; i < o; i++ {
                s.next()
        }
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
			if s.offset-offs <= 2 {
				// only scanned "0b" or "0B"
				s.error(offs, "illegal binary number")
			}
		} else if s.ch == 'x' || s.ch == 'X' {
			// hexadecimal int
			s.next()
			s.scanMantissa(16)
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
		}
		goto exit
	}

        // decimal int or float
        s.scanMantissa(10)

fraction:
	if s.ch == '.' {
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
        if ml {
                offs -= 1
        }

	for {
		ch := s.ch
		if (!ml && ch == '\n') || ch < 0 { // if ch < 0 {
			s.error(offs, "raw string literal not terminated")
			break
		}
		s.next()
		if ch == '\'' {
                        if !ml {
                                break
                        }
                        if s.ch == '\'' {                                
                                if s.next(); s.ch == '\'' {
                                        s.next()
                                        break
                                }
                        }
		}
	}

	return string(s.src[offs:s.offset])
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

func (s *Scanner) scanCall() {
        // TODO: $(...)
}

func (s *Scanner) scanString(ml bool) string {
	// '"' opening already consumed
	offs := s.offset - 1
        if ml {
                offs -= 1
        }

	for {
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
                        s.scanCall()
		}
	}

	return string(s.src[offs:s.offset])
}

/*
func (s *Scanner) scanText() string {
	offs := s.offset - 1

	for {
		ch := s.ch
		if ch == '\n' || ch < 0 {
			s.error(offs, "string literal not terminated")
			break
		}
		s.next()
                if s.ch == ' ' || s.ch == '\t' || s.ch == '\n' || s.ch == '\r' {
			break
		}
                if ch == '$' {
                        s.scanCall()
		}
	}

	return string(s.src[offs:s.offset])
} */

func (s *Scanner) scanSep() (tok token.Token, lit string) {
	offs := s.offset - 1
        
	for s.ch == ' ' || s.ch == '\t' || s.ch == '\\' || (s.ch == '\n' && s.parenDepth > 0) {
                ch := s.ch
		s.next()
                if ch == '\\' {
                        if s.ch != '\n' {
                                s.error(s.offset, "\\ not following \\n")
                                break
                        }
                        s.next()
                }
	}

        if s.ch == ')' {
                s.next() // consume ')'
                s.parenDepth--
                tok = token.RPAREN
        } else {
                tok = token.SEP
        }

        lit = string(s.src[offs:s.offset])
        return
}

func (s *Scanner) Scan() (pos token.Pos, tok token.Token, lit string) {
//scanAgain:
        if !s.sepAware || s.offset == 0 {
                s.skipWhitespace(true)
        }

	// current token start
	pos = s.file.Pos(s.offset)

        skipWhitespaceAfter := false

	// determine token value
	switch ch := s.ch; {
	case isLetter(ch):
		lit = s.scanIdentifier()
		if len(lit) > 1 {
			switch tok = token.Lookup(lit); tok {
                        case token.IDENT, token.PROJECT, token.MODULE, token.USE, token.EXPORT, token.INCLUDE:
                                // ...
			}
                } else {
                        tok = token.IDENT
                }
	case '0' <= ch && ch <= '9':
                tok, lit = s.scanNumber(false)
        default:
                s.next() // always progress to the next
                switch ch {
                case '#':
                        tok, lit = token.COMMENT, s.scanComment()
                        skipWhitespaceAfter = true
                case '+':
                        tok = token.ADD
                        skipWhitespaceAfter = true
                case '-':
                        tok = token.SUB
                        skipWhitespaceAfter = true
                case '*':
                        tok = token.MUL
                        skipWhitespaceAfter = true
                case '/':
                        tok = token.QUO
                        skipWhitespaceAfter = true
                case '%':
                        tok = token.REM
                        skipWhitespaceAfter = true
                //case '\\':
                case '\'':
                        tok = token.STRING
                        if s.ch == '\'' {
                                s.next()
                                if s.ch == '\'' { // '''
                                        lit = s.scanRawString(true)
                                } else {
                                        // got empty string ''
                                        offs := s.offset - 2
                                        lit = string(s.src[offs:s.offset])
                                }
                        } else {
                                lit = s.scanRawString(false)
                        }
                case '"':
                        tok = token.STRING
                        if s.ch == '"' {
                                s.next()
                                if s.ch == '"' { // """
                                        lit = s.scanString(true)
                                } else {
                                        // got empty string ""
                                        offs := s.offset - 2
                                        lit = string(s.src[offs:s.offset])
                                }
                        } else {
                                lit = s.scanString(false)
                        }
                case '$':
                        tok = token.CALL
                case '(':
                        tok = token.LPAREN
                        s.parenDepth++
                        skipWhitespaceAfter = true
                case ')':
                        if s.parenDepth == 0 {
				s.error(s.offset, "unexpected right parenthesis")
                        } else {
                                tok = token.RPAREN
                                s.parenDepth--
                        }
                case '=':
                        tok = token.ASSIGN
                        s.sepAware = true
                        skipWhitespaceAfter = true
                case '\n':
                        if s.parenDepth == 0 {
                                tok = token.LINEND
                                s.sepAware = false
                                skipWhitespaceAfter = true
                        } else {
                                tok, lit = s.scanSep()
                        }
                case ',':
                        tok = token.COMMA
                        skipWhitespaceAfter = true
                default:
			// next reports unexpected BOMs - don't repeat
			if ch != bom {
				s.error(s.file.Offset(pos), fmt.Sprintf("illegal character %#U", ch))
			}
			tok = token.ILLEGAL
			lit = string(ch)
                }
                if (s.sepAware && (ch == ' ' || ch == '\t')) { // FIXME: \\\n
                        tok, lit = s.scanSep()
                }
	}

        if skipWhitespaceAfter {
                // eat consequence spaces                
                s.skipWhitespace(false)
        }
	return
}
