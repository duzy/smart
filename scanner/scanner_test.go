//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package scanner

import (
	"path/filepath"
        "testing"
        "github.com/duzy/smart/token"
)

var fset = token.NewFileSet()

type scanResult struct {
        offset int
        tok token.Token
        lit string
}

func TestInit(t *testing.T) {
	var s Scanner

	// 1st init
	src1 := "module a"
	f1 := fset.AddFile(filepath.Join("TestInit", "src1"), fset.Base(), len(src1))
	s.Init(f1, []byte(src1), nil, ScanComments)
	if f1.Size() != len(src1) {
		t.Errorf("bad file size: got %d, expected %d", f1.Size(), len(src1))
	}

        var (
                tok token.Token
                lit string
        )

	_, tok, _ = s.Scan() // module
	if tok != token.MODULE {
		t.Errorf("bad token: got %s, expected %s", tok, token.MODULE)
	}

	_, tok, lit = s.Scan() // a
	if tok != token.IDENT {
		t.Errorf("bad token: got %s, expected %s", tok, token.IDENT)
	}
        if lit != "a" {
		t.Errorf("bad literal: got %s, expected %s", lit, "a")
        }

	// 2nd init
	src2 := "v = abc"
	f2 := fset.AddFile(filepath.Join("TestInit", "src2"), fset.Base(), len(src2))
	s.Init(f2, []byte(src2), nil, ScanComments)
	if f2.Size() != len(src2) {
		t.Errorf("bad file size: got %d, expected %d", f2.Size(), len(src2))
	}
        
	_, tok, lit = s.Scan() // v
	if tok != token.IDENT {
		t.Errorf("bad token: got %s, expected %s", tok, token.IDENT)
	}
        if lit != "v" {
		t.Errorf("bad literal: got %s, expected %s", lit, "v")
        }

	_, tok, _ = s.Scan() // =
	if tok != token.ASSIGN {
		t.Errorf("bad token: got %s, expected %s", tok, token.ASSIGN)
	}

	_, tok, lit = s.Scan() // abc
	if tok != token.IDENT {
		t.Errorf("bad token: got %s, expected %s", tok, token.IDENT)
	}
        if lit != "abc" {
		t.Errorf("bad literal: got %s, expected %s", lit, "abc")
        }
        
	if s.ErrorCount != 0 {
		t.Errorf("found %d errors", s.ErrorCount)
	}
}

func TestStrings(t *testing.T) {
	var s Scanner

        src := `
string1 = 'a b c $a $b $c'
string2 = "a b c $a $b $c"
string3 = "a b c \"1 2 3\""

string_concate = $(string1)$(string2)

string4 = """
string line 1
string line 2
string line 3
"""

string5 = """\
    string line 1 \
    string line 2 \
    string line 3 \
    """

strings = 'abc' "xx $(string1) xx"

empty1 = ''
empty2 = ""
empty3 =

text1 = this-is-a-text
texts = this is a text array
`
	f := fset.AddFile(filepath.Join("TestStrings", "src"), fset.Base(), len(src))
	s.Init(f, []byte(src), nil, ScanComments)
	if f.Size() != len(src) {
		t.Errorf("bad file size: got %d, expected %d", f.Size(), len(src))
	}

        results := []scanResult{
                { 1, token.IDENT, `string1` },
                {-1, token.ASSIGN, `` },
                {-1, token.STRING, `'a b c $a $b $c'` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `string2` },
                {-1, token.ASSIGN, `` },
                {-1, token.STRING, `"a b c $a $b $c"` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `string3` },
                {-1, token.ASSIGN, `` },
                {-1, token.STRING, `"a b c \"1 2 3\""` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `string_concate` },
                {-1, token.ASSIGN, `` },
                {-1, token.CALL, `` },
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `string1` },
                {-1, token.RPAREN, `` },
                {-1, token.CALL, `` },
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `string2` },
                {-1, token.RPAREN, `` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `string4` },
                {-1, token.ASSIGN, `` },
                {-1, token.STRING, `"""
string line 1
string line 2
string line 3
"""` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `string5` },
                {-1, token.ASSIGN, `` },
                {-1, token.STRING, `"""\
    string line 1 \
    string line 2 \
    string line 3 \
    """` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `strings` },
                {-1, token.ASSIGN, `` },
                {-1, token.STRING, `'abc'` },
                {-1, token.STRING, `"xx $(string1) xx"` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `empty1` },
                {-1, token.ASSIGN, `` },
                {-1, token.STRING, `''` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `empty2` },
                {-1, token.ASSIGN, `` },
                {-1, token.STRING, `""` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `empty3` },
                {-1, token.ASSIGN, `` },
                {-1, token.LINEND, `` },
        }
        for i, r := range results {
                pos, tok, lit := s.Scan()
                if 0 <= r.offset && pos != s.file.Pos(r.offset) {
                        t.Errorf("%d: bad pos: got %d, expected %d (%s)", i, pos, s.file.Pos(r.offset), r.lit)
                }
                if tok != r.tok {
                        t.Errorf("%d: bad token: got %s, expected %s (%s)", i, tok, r.tok, r.lit)
                }
                if lit != r.lit {
                        t.Errorf("%d: bad literal: got %s, expected %s", i, lit, r.lit)
                }
        }
}

func TestIntegers(t *testing.T) {
	var s Scanner

        src := `
integer1 = +100
integer2 = 99
integer3 = -38

integer4 = 10_000_000
integer5 = 1_2_3_4_5 # VALID but discouraged

octal1 = 01234567
octal2 = 01_0_000

hex1 = 0x123456789ABCDEF
hex2 = 0xAAAA_BBBB_1111

bin1 = 0b0011001100
bin2 = 0b1100110011
`
	f := fset.AddFile(filepath.Join("TestIntegers", "src"), fset.Base(), len(src))
	s.Init(f, []byte(src), nil, ScanComments)
	if f.Size() != len(src) {
		t.Errorf("bad file size: got %d, expected %d", f.Size(), len(src))
	}

        results := []scanResult{
                {-1, token.IDENT, `integer1` },
                {-1, token.ASSIGN, `` },
                {-1, token.ADD, `` },
                {-1, token.INT, `100` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `integer2` },
                {-1, token.ASSIGN, `` },
                {-1, token.INT, `99` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `integer3` },
                {-1, token.ASSIGN, `` },
                {-1, token.SUB, `` },
                {-1, token.INT, `38` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `integer4` },
                {-1, token.ASSIGN, `` },
                {-1, token.INT, `10_000_000` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `integer5` },
                {-1, token.ASSIGN, `` },
                {-1, token.INT, `1_2_3_4_5` },

                {-1, token.COMMENT, `# VALID but discouraged` },

                {-1, token.IDENT, `octal1` },
                {-1, token.ASSIGN, `` },
                {-1, token.INT, `01234567` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `octal2` },
                {-1, token.ASSIGN, `` },
                {-1, token.INT, `01_0_000` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `hex1` },
                {-1, token.ASSIGN, `` },
                {-1, token.INT, `0x123456789ABCDEF` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `hex2` },
                {-1, token.ASSIGN, `` },
                {-1, token.INT, `0xAAAA_BBBB_1111` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `bin1` },
                {-1, token.ASSIGN, `` },
                {-1, token.INT, `0b0011001100` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `bin2` },
                {-1, token.ASSIGN, `` },
                {-1, token.INT, `0b1100110011` },
                {-1, token.LINEND, `` },
        }
        for i, r := range results {
                pos, tok, lit := s.Scan()
                if 0 <= r.offset && pos != s.file.Pos(r.offset) {
                        t.Errorf("%d: bad pos: got %d, expected %d (%s)", i, pos, s.file.Pos(r.offset), r.lit)
                }
                if tok != r.tok {
                        t.Errorf("%d: bad token: got %s, expected %s (%s)", i, tok, r.tok, r.lit)
                }
                if lit != r.lit {
                        t.Errorf("%d: bad literal: got %s, expected %s", i, lit, r.lit)
                }
        }
}

func TestDatetime(t *testing.T) {
	var s Scanner

        src := `
t1 = 1979-05-27T07:32:00Z
t2 = 1979-05-27T07:32:00-07:00
t3 = 1979-05-27T07:32:00.999999-07:00

t4 = 1979-05-27T07:32:00
t5 = 1979-05-27T07:32:00.999999

d1 = 1979-05-27

t6 = 07:32:00
t7 = 07:32:00.999999
`
	f := fset.AddFile(filepath.Join("TestDatetime", "src"), fset.Base(), len(src))
	s.Init(f, []byte(src), nil, ScanComments)
	if f.Size() != len(src) {
		t.Errorf("bad file size: got %d, expected %d", f.Size(), len(src))
	}

        results := []scanResult{
                {-1, token.IDENT, `t1` },
                {-1, token.ASSIGN, `` },
                {-1, token.DATETIME, `1979-05-27T07:32:00Z` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `t2` },
                {-1, token.ASSIGN, `` },
                {-1, token.DATETIME, `1979-05-27T07:32:00-07:00` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `t3` },
                {-1, token.ASSIGN, `` },
                {-1, token.DATETIME, `1979-05-27T07:32:00.999999-07:00` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `t4` },
                {-1, token.ASSIGN, `` },
                {-1, token.DATETIME, `1979-05-27T07:32:00` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `t5` },
                {-1, token.ASSIGN, `` },
                {-1, token.DATETIME, `1979-05-27T07:32:00.999999` },
                {-1, token.LINEND, `` },
                
                {-1, token.IDENT, `d1` },
                {-1, token.ASSIGN, `` },
                {-1, token.DATE, `1979-05-27` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `t6` },
                {-1, token.ASSIGN, `` },
                {-1, token.TIME, `07:32:00` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `t7` },
                {-1, token.ASSIGN, `` },
                {-1, token.TIME, `07:32:00.999999` },
                {-1, token.LINEND, `` },
        }
        for i, r := range results {
                pos, tok, lit := s.Scan()
                if 0 <= r.offset && pos != s.file.Pos(r.offset) {
                        t.Errorf("%d: bad pos: got %d, expected %d (%s)", i, pos, s.file.Pos(r.offset), r.lit)
                }
                if tok != r.tok {
                        t.Errorf("%d: bad token: got %s, expected %s (%s)", i, tok, r.tok, r.lit)
                }
                if lit != r.lit {
                        t.Errorf("%d: bad literal: got %s, expected %s", i, lit, r.lit)
                }
        }
}

func TestFloats(t *testing.T) {
	var s Scanner

        src := `
float1 = +1.0
float2 = 3.1415
float3 = - 0.001

float4 = 5e+22
float5 = 1e6
float6 = -2E-2

float7 = 3.1415e-100
float8 = 6.18_16_18_16
`
	f := fset.AddFile(filepath.Join("TestFloats", "src"), fset.Base(), len(src))
	s.Init(f, []byte(src), nil, ScanComments)
	if f.Size() != len(src) {
		t.Errorf("bad file size: got %d, expected %d", f.Size(), len(src))
	}

        results := []scanResult{
                {-1, token.IDENT, `float1` },
                {-1, token.ASSIGN, `` },
                {-1, token.ADD, `` },
                {-1, token.FLOAT, `1.0` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `float2` },
                {-1, token.ASSIGN, `` },
                {-1, token.FLOAT, `3.1415` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `float3` },
                {-1, token.ASSIGN, `` },
                {-1, token.SUB, `` },
                {-1, token.FLOAT, `0.001` },
                {-1, token.LINEND, `` },
                
                {-1, token.IDENT, `float4` },
                {-1, token.ASSIGN, `` },
                {-1, token.FLOAT, `5e+22` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `float5` },
                {-1, token.ASSIGN, `` },
                {-1, token.FLOAT, `1e6` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `float6` },
                {-1, token.ASSIGN, `` },
                {-1, token.SUB, `` },
                {-1, token.FLOAT, `2E-2` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `float7` },
                {-1, token.ASSIGN, `` },
                {-1, token.FLOAT, `3.1415e-100` },
                {-1, token.LINEND, `` },
                
                {-1, token.IDENT, `float8` },
                {-1, token.ASSIGN, `` },
                {-1, token.FLOAT, `6.18_16_18_16` },
                {-1, token.LINEND, `` },
        }
        for i, r := range results {
                pos, tok, lit := s.Scan()
                if 0 <= r.offset && pos != s.file.Pos(r.offset) {
                        t.Errorf("%d: bad pos: got %d, expected %d (%s)", i, pos, s.file.Pos(r.offset), r.lit)
                }
                if tok != r.tok {
                        t.Errorf("%d: bad token: got %s, expected %s (%s)", i, tok, r.tok, r.lit)
                }
                if lit != r.lit {
                        t.Errorf("%d: bad literal: got %s, expected %s", i, lit, r.lit)
                }
        }
}

func TestArrays(t *testing.T) {
	var s Scanner

        src := `
array1 = text1 text2 text3 '' 1 2 3 1.2 ( a b c 1 2 3 '' "")

array2 = \
  text1 \
  text2 \
  text3 \
  '' \
  1 \
  2 \
  3
`
	f := fset.AddFile(filepath.Join("TestArrays", "src"), fset.Base(), len(src))
	s.Init(f, []byte(src), nil, ScanComments)
	if f.Size() != len(src) {
		t.Errorf("bad file size: got %d, expected %d", f.Size(), len(src))
	}

        results := []scanResult{
                {-1, token.IDENT, `array1` },
                {-1, token.ASSIGN, `` },
                {-1, token.IDENT, `text1` },
                {-1, token.IDENT, `text2` },
                {-1, token.IDENT, `text3` },
                {-1, token.STRING, `''` },
                {-1, token.INT, `1` },
                {-1, token.INT, `2` },
                {-1, token.INT, `3` },
                {-1, token.FLOAT, `1.2` },
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `a` },
                {-1, token.IDENT, `b` },
                {-1, token.IDENT, `c` },
                {-1, token.INT, `1` },
                {-1, token.INT, `2` },
                {-1, token.INT, `3` },
                {-1, token.STRING, `''` },
                {-1, token.STRING, `""` },
                {-1, token.RPAREN, `` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `array2` },
                {-1, token.ASSIGN, `` }, // consequence \\n and spaces are ignored
                {-1, token.IDENT, `text1` },
                {-1, token.IDENT, `text2` },
                {-1, token.IDENT, `text3` },
                {-1, token.STRING, `''` },
                {-1, token.INT, `1` },
                {-1, token.INT, `2` },
                {-1, token.INT, `3` },
                {-1, token.LINEND, `` },
        }
        for i, r := range results {
                pos, tok, lit := s.Scan()
                if 0 <= r.offset && pos != s.file.Pos(r.offset) {
                        t.Errorf("%d: bad pos: got %d, expected %d (%s)", i, pos, s.file.Pos(r.offset), r.lit)
                }
                if tok != r.tok {
                        t.Errorf("%d: bad token: got %s, expected %s (%s)", i, tok, r.tok, r.lit)
                }
                if lit != r.lit {
                        t.Errorf("%d: bad literal: got %s, expected %s", i, lit, r.lit)
                }
        }
}

func TestMaps(t *testing.T) {
	var s Scanner

        src := `
map1 = (
   k1 value1,
   k2 value2,
   k3 value3,
   k4 value,
)

map2 = (  k1 v1, k2 'v2 v2', k3 "v3 v3 v3", k4 v4  )
`
	f := fset.AddFile(filepath.Join("TestMaps", "src"), fset.Base(), len(src))
	s.Init(f, []byte(src), nil, ScanComments)
	if f.Size() != len(src) {
		t.Errorf("bad file size: got %d, expected %d", f.Size(), len(src))
	}

        results := []scanResult{
                {-1, token.IDENT, `map1` },
                {-1, token.ASSIGN, `` },
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `k1` },
                {-1, token.IDENT, `value1` },
                {-1, token.COMMA, `` },
                {-1, token.IDENT, `k2` },
                {-1, token.IDENT, `value2` },
                {-1, token.COMMA, `` },
                {-1, token.IDENT, `k3` },
                {-1, token.IDENT, `value3` },
                {-1, token.COMMA, `` },
                {-1, token.IDENT, `k4` },
                {-1, token.IDENT, `value` },
                {-1, token.COMMA, `` },
                {-1, token.RPAREN, `` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `map2` },
                {-1, token.ASSIGN, `` },
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `k1` },
                {-1, token.IDENT, `v1` },
                {-1, token.COMMA, `` },
                {-1, token.IDENT, `k2` },
                {-1, token.STRING, `'v2 v2'` },
                {-1, token.COMMA, `` },
                {-1, token.IDENT, `k3` },
                {-1, token.STRING, `"v3 v3 v3"` },
                {-1, token.COMMA, `` },
                {-1, token.IDENT, `k4` },
                {-1, token.IDENT, `v4` },
                {-1, token.RPAREN, `` },
                {-1, token.LINEND, `` },
        }
        for i, r := range results {
                pos, tok, lit := s.Scan()
                if 0 <= r.offset && pos != s.file.Pos(r.offset) {
                        t.Errorf("%d: bad pos: got %d, expected %d (%s)", i, pos, s.file.Pos(r.offset), r.lit)
                }
                if tok != r.tok {
                        t.Errorf("%d: bad token: got %s, expected %s (%s)", i, tok, r.tok, r.lit)
                }
                if lit != r.lit {
                        t.Errorf("%d: bad literal: got %s, expected %s", i, lit, r.lit)
                }
        }
}

func TestCalls(t *testing.T) {
	var s Scanner

        src1 := `
# bare lets

$(let ((a "value of a")
       (b 'value of b')
       (c 'value of c'))
      (print "$a.$b.$c"))

$(let ( (a 1e-10) (b 2017-01-18) (c 19:25:30) )
      ( print "$a $b $c" ) )
`
	f1 := fset.AddFile(filepath.Join("TestCalls", "src1"), fset.Base(), len(src1))
	s.Init(f1, []byte(src1), nil, ScanComments)
	if f1.Size() != len(src1) {
		t.Errorf("bad file size: got %d, expected %d", f1.Size(), len(src1))
	}
        results1 := []scanResult{
                { 1, token.COMMENT, `# bare lets` },
                {-1, token.CALL, `` },
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `let` },
                {-1, token.LPAREN, `` },
                
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `a` },
                {-1, token.STRING, `"value of a"` },
                {-1, token.RPAREN, `` },
                
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `b` },
                {-1, token.STRING, `'value of b'` },
                {-1, token.RPAREN, `` },
                
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `c` },
                {-1, token.STRING, `'value of c'` },
                {-1, token.RPAREN, `` },
                
                {-1, token.RPAREN, `` }, // 'let' enclosed
                
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `print` },
                {-1, token.STRING, `"$a.$b.$c"` },
                {-1, token.RPAREN, `` },
                
                {-1, token.RPAREN, `` },
                {-1, token.LINEND, `` },

                {-1, token.CALL, `` },
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `let` },
                {-1, token.LPAREN, `` },
                
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `a` },
                {-1, token.FLOAT, `1e-10` },
                {-1, token.RPAREN, `` },
                
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `b` },
                {-1, token.DATE, `2017-01-18` },
                {-1, token.RPAREN, `` },
                
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `c` },
                {-1, token.TIME, `19:25:30` },
                {-1, token.RPAREN, `` },
                
                {-1, token.RPAREN, `` }, // 'let' enclosed
                
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `print` },
                {-1, token.STRING, `"$a $b $c"` },
                {-1, token.RPAREN, `` },
                
                {-1, token.RPAREN, `` },
                {-1, token.LINEND, `` },
        }
        for i, r := range results1 {
                pos, tok, lit := s.Scan()
                if 0 <= r.offset && pos != s.file.Pos(r.offset) {
                        t.Errorf("%d: bad pos: got %d, expected %d (%s)", i, pos, s.file.Pos(r.offset), r.lit)
                }
                if tok != r.tok {
                        t.Errorf("%d: bad token: got %s, expected %s (%s)", i, tok, r.tok, r.lit)
                }
                if lit != r.lit {
                        t.Errorf("%d: bad literal: got %s, expected %s", i, lit, r.lit)
                }
        }

        src2 := `
# binds

concat = $(bind (a b c) "$a.$b.$c")

v1 = $(concat 1 2 3)

v2 = $(concat "a" 'b' c)

`
	f2 := fset.AddFile(filepath.Join("TestCalls", "src2"), fset.Base(), len(src2))
	s.Init(f2, []byte(src2), nil, ScanComments)
	if f2.Size() != len(src2) {
		t.Errorf("bad file size: got %d, expected %d", f2.Size(), len(src2))
	}
        results2 := []scanResult{
                { 1, token.COMMENT, `# binds` },

                {-1, token.IDENT, `concat` },
                {-1, token.ASSIGN, `` },
                {-1, token.CALL, `` },
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `bind` },
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `a` },
                {-1, token.IDENT, `b` },
                {-1, token.IDENT, `c` },
                {-1, token.RPAREN, `` },
                {-1, token.STRING, `"$a.$b.$c"` },
                {-1, token.RPAREN, `` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `v1` },
                {-1, token.ASSIGN, `` },
                {-1, token.CALL, `` },
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `concat` },
                {-1, token.INT, `1` },
                {-1, token.INT, `2` },
                {-1, token.INT, `3` },
                {-1, token.RPAREN, `` },
                {-1, token.LINEND, `` },
                
                {-1, token.IDENT, `v2` },
                {-1, token.ASSIGN, `` },
                {-1, token.CALL, `` },
                {-1, token.LPAREN, `` },
                {-1, token.IDENT, `concat` },
                {-1, token.STRING, `"a"` },
                {-1, token.STRING, `'b'` },
                {-1, token.IDENT, `c` },
                {-1, token.RPAREN, `` },
                {-1, token.LINEND, `` },
        }
        for i, r := range results2 {
                pos, tok, lit := s.Scan()
                if 0 <= r.offset && pos != s.file.Pos(r.offset) {
                        t.Errorf("%d: bad pos: got %d, expected %d (%s)", i, pos, s.file.Pos(r.offset), r.lit)
                }
                if tok != r.tok {
                        t.Errorf("%d: bad token: got %s, expected %s (%s)", i, tok, r.tok, r.lit)
                }
                if lit != r.lit {
                        t.Errorf("%d: bad literal: got %s, expected %s", i, lit, r.lit)
                }
        }
}

func TestRules(t *testing.T) {
	var s Scanner

        src1 := `
# rules

prog: obj/file.o
	gcc -o $@ $<
obj/file.o: src/file.c
	gcc -c -o $@ $^
`
	f1 := fset.AddFile(filepath.Join("TestRules", "src1"), fset.Base(), len(src1))
	s.Init(f1, []byte(src1), nil, ScanComments)
	if f1.Size() != len(src1) {
		t.Errorf("bad file size: got %d, expected %d", f1.Size(), len(src1))
	}
        results1 := []scanResult{
                { 1, token.COMMENT, `# rules` },

                {-1, token.IDENT, `prog` },
                {-1, token.COLON, `` },
                {-1, token.IDENT, `obj/file.o` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `gcc -o $@ $<` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `obj/file.o` },
                {-1, token.COLON, `` },
                {-1, token.IDENT, `src/file.c` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `gcc -c -o $@ $^` },
                {-1, token.LINEND, `` },
        }
        for i, r := range results1 {
                pos, tok, lit := s.Scan()
                if 0 <= r.offset && pos != s.file.Pos(r.offset) {
                        t.Errorf("%d: bad pos: got %d, expected %d (%s)", i, pos, s.file.Pos(r.offset), r.lit)
                }
                if tok != r.tok {
                        t.Errorf("%d: bad token: got %s, expected %s (%s)", i, tok, r.tok, r.lit)
                }
                if lit != r.lit {
                        t.Errorf("%d: bad literal: got %s, expected %s", i, lit, r.lit)
                }
        }

        src2 := `
# rules

start:
	echo one
	echo one
	echo one
start::
	echo two
	echo two
	echo two
start::
	echo three
	echo three
	echo three
`
	f2 := fset.AddFile(filepath.Join("TestRules", "src2"), fset.Base(), len(src2))
	s.Init(f2, []byte(src2), nil, ScanComments)
	if f2.Size() != len(src2) {
		t.Errorf("bad file size: got %d, expected %d", f2.Size(), len(src2))
	}
        results2 := []scanResult{
                { 1, token.COMMENT, `# rules` },

                {-1, token.IDENT, `start` },
                {-1, token.COLON, `` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `echo one` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `echo one` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `echo one` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `start` },
                {-1, token.COLON2, `` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `echo two` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `echo two` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `echo two` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `start` },
                {-1, token.COLON2, `` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `echo three` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `echo three` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `echo three` },
                {-1, token.LINEND, `` },
        }
        for i, r := range results2 {
                pos, tok, lit := s.Scan()
                if 0 <= r.offset && pos != s.file.Pos(r.offset) {
                        t.Errorf("%d: bad pos: got %d, expected %d (%s)", i, pos, s.file.Pos(r.offset), r.lit)
                }
                if tok != r.tok {
                        t.Errorf("%d: bad token: got %s, expected %s (%s)", i, tok, r.tok, r.lit)
                }
                if lit != r.lit {
                        t.Errorf("%d: bad literal: got %s, expected %s", i, lit, r.lit)
                }
        }

        src3 := `
# rules

start:!:
	echo okay
start:?:
	test src/file.c
`
	f3 := fset.AddFile(filepath.Join("TestRules", "src3"), fset.Base(), len(src3))
	s.Init(f3, []byte(src3), nil, ScanComments)
	if f3.Size() != len(src3) {
		t.Errorf("bad file size: got %d, expected %d", f3.Size(), len(src3))
	}
        results3 := []scanResult{
                { 1, token.COMMENT, `# rules` },

                {-1, token.IDENT, `start` },
                {-1, token.COLON_EXC, `` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `echo okay` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `start` },
                {-1, token.COLON_QUE, `` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `test src/file.c` },
                {-1, token.LINEND, `` },
        }
        for i, r := range results3 {
                pos, tok, lit := s.Scan()
                if 0 <= r.offset && pos != s.file.Pos(r.offset) {
                        t.Errorf("%d: bad pos: got %d, expected %d (%s)", i, pos, s.file.Pos(r.offset), r.lit)
                }
                if tok != r.tok {
                        t.Errorf("%d: bad token: got %s, expected %s (%s)", i, tok, r.tok, r.lit)
                }
                if lit != r.lit {
                        t.Errorf("%d: bad literal: got %s, expected %s", i, lit, r.lit)
                }
        }
        
        src4 := `
# brack rules

start:[shell]:
	echo okay
start:![shell]:
	echo okay okay
start:?[shell]:
	test ok ok
`
	f4 := fset.AddFile(filepath.Join("TestRules", "src4"), fset.Base(), len(src4))
	s.Init(f4, []byte(src4), nil, ScanComments)
	if f4.Size() != len(src4) {
		t.Errorf("bad file size: got %d, expected %d", f4.Size(), len(src4))
	}
        results4 := []scanResult{
                { 1, token.COMMENT, `# brack rules` },

                {-1, token.IDENT, `start` },
                {-1, token.COLON_LBK, `` },
                {-1, token.IDENT, `shell` },
                {-1, token.COLON_RBK, `` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `echo okay` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `start` },
                {-1, token.COLON_LBE, `` },
                {-1, token.IDENT, `shell` },
                {-1, token.COLON_RBK, `` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `echo okay okay` },
                {-1, token.LINEND, `` },

                {-1, token.IDENT, `start` },
                {-1, token.COLON_LBQ, `` },
                {-1, token.IDENT, `shell` },
                {-1, token.COLON_RBK, `` },
                {-1, token.LINEND, `` },
                {-1, token.RECIEPT, `test ok ok` },
                {-1, token.LINEND, `` },
        }
        for i, r := range results4 {
                pos, tok, lit := s.Scan()
                if 0 <= r.offset && pos != s.file.Pos(r.offset) {
                        t.Errorf("%d: bad pos: got %d, expected %d (%s)", i, pos, s.file.Pos(r.offset), r.lit)
                }
                if tok != r.tok {
                        t.Errorf("%d: bad token: got %s, expected %s (%s)", i, tok, r.tok, r.lit)
                }
                if lit != r.lit {
                        t.Errorf("%d: bad literal: got %s, expected %s", i, lit, r.lit)
                }
        }
}
