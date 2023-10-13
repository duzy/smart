//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "path/filepath"
    "testing"
)

var fset = NewFileSet()

type scanResult struct {
    offset int
    tok Token
    lit string
}

func testInit(t *testing.T) {
    var s Scanner

    // 1st init
    src1 := "module a"
    f1 := fset.AddFile(filepath.Join("TestInit", "src1"), fset.Base(), len(src1))
    s.Init(f1, []byte(src1), ScanMode(0), nil, nil)
    if f1.Size() != len(src1) {
        t.Errorf("bad file size: got %d, expected %d", f1.Size(), len(src1))
    }

    var (
        tok Token
        lit string
    )

    _, tok, _ = s.Scan() // module
    if tok != MODULE {
        t.Errorf("bad token: got %s, expected %s", tok, MODULE)
    }

    _, tok, lit = s.Scan()
    if tok != SPACE {
        t.Errorf("bad token: got %s, expected %s", tok, SPACE)
    }

    _, tok, lit = s.Scan() // a
    if tok != BAREWORD {
        t.Errorf("bad token: got %s, expected %s", tok, BAREWORD)
    }
    if lit != "a" {
        t.Errorf("bad literal: got %s, expected %s", lit, "a")
    }

    // 2nd init
    src2 := "v = abc"
    f2 := fset.AddFile(filepath.Join("TestInit", "src2"), fset.Base(), len(src2))
    s.Init(f2, []byte(src2), ScanMode(0), nil, nil)
    if f2.Size() != len(src2) {
        t.Errorf("bad file size: got %d, expected %d", f2.Size(), len(src2))
    }

    _, tok, lit = s.Scan() // v
    if tok != BAREWORD {
        t.Errorf("bad token: got %s, expected %s", tok, BAREWORD)
    }
    if lit != "v" {
        t.Errorf("bad literal: got %s, expected %s", lit, "v")
    }

    _, tok, lit = s.Scan()
    if tok != SPACE {
        t.Errorf("bad token: got %s, expected %s", tok, SPACE)
    }

    _, tok, _ = s.Scan() // =
    if tok != ASSIGN {
        t.Errorf("bad token: got %s, expected %s", tok, ASSIGN)
    }

    _, tok, lit = s.Scan()
    if tok != SPACE {
        t.Errorf("bad token: got %s, expected %s", tok, SPACE)
    }

    _, tok, lit = s.Scan() // abc
    if tok != BAREWORD {
        t.Errorf("bad token: got %s, expected %s", tok, BAREWORD)
    }
    if lit != "abc" {
        t.Errorf("bad literal: got %s, expected %s", lit, "abc")
    }

    if s.ErrorCount != 0 {
        t.Errorf("found %d errors", s.ErrorCount)
    }
}

func testStrings(t *testing.T) {
    src := `
string1 = 'a b c $a $b $c 1'
string2 = "a b c $a $b $c 2"
string3 = "a b c \"1 2 3\""

string_concate = $(string1)$(string2)

strings = 'abc' "xx $(string1) xx"

empty1 = ''
empty2 = ""
empty3 =

text1 = this-is-a-text
texts = this is a text array
`
    var s Scanner
    f := fset.AddFile(filepath.Join("TestStrings", "src"), fset.Base(), len(src))
    s.Init(f, []byte(src), ScanMode(0), nil, nil)
    if f.Size() != len(src) {
        t.Errorf("bad file size: got %d, expected %d", f.Size(), len(src))
    }

    results := []scanResult{
        { 0, LINEND, "" },

        { 1, BAREWORD, `string1` },
        {-1, SPACE, ` ` },
        {-1, ASSIGN, `` },
        {-1, SPACE, ` ` },
        {-1, STRING, `a b c $a $b $c 1` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `string2` },
        {-1, SPACE, ` ` },
        {-1, ASSIGN, `` },
        {-1, SPACE, ` ` },
        {-1, COMPOUND, `` }, // "a b c $a $b $c 2"
        {-1, RAW, `a b c ` },
        {-1, DELEGATE, `` },
        {-1, BAREWORD, `a` },
        {-1, RAW, ` ` },
        {-1, DELEGATE, `` },
        {-1, BAREWORD, `b` },
        {-1, RAW, ` ` },
        {-1, DELEGATE, `` },
        {-1, BAREWORD, `c` },
        {-1, RAW, ` 2` },
        {-1, COMPOSED, `` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `string3` },
        {-1, SPACE, ` ` },
        {-1, ASSIGN, `` },
        {-1, SPACE, ` ` },
        {-1, COMPOUND, `` }, // "a b c \"1 2 3\""
        {-1, RAW, `a b c ` },
        {-1, ESCAPE, `"` },
        {-1, RAW, `1 2 3` },
        {-1, ESCAPE, `"` },
        {-1, COMPOSED, `` },
        {-1, LINEND, `` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `string_concate` },
        {-1, SPACE, ` ` },
        {-1, ASSIGN, `` },
        {-1, SPACE, ` ` },
        {-1, DELEGATE, `` },
        {-1, LPAREN, `(` },
        {-1, BAREWORD, `string1` },
        {-1, RPAREN, `)` },
        {-1, DELEGATE, `` },
        {-1, LPAREN, `(` },
        {-1, BAREWORD, `string2` },
        {-1, RPAREN, `)` },
        {-1, LINEND, `` },
        {-1, LINEND, `` },

// string4 = """
// string line 1
// string line 2
// string line 3
// """
//
//         {-1, BAREWORD, `string4` },
//         {-1, SPACE, ` ` },
//         {-1, ASSIGN, `` },
//         {-1, SPACE, ` ` },
//         {-1, COMPOUND, `` }, // """
//         {-1, RAW, `
// string line 1
// string line 2
// string line 3
// ` },
//         {-1, COMPOSED, `` }, // """
//         {-1, LINEND, `` },
//         {-1, LINEND, `` },

// string5 = """\
//     string line 1 \
//     string line 2 \
//     string line 3 \
//     """
//
//         {-1, BAREWORD, `string5` },
//         {-1, SPACE, ` ` },
//         {-1, ASSIGN, `` },
//         {-1, SPACE, ` ` },
//         {-1, COMPOUND, `` }, // """
//         {-1, STRING, `\
//     string line 1 \
//     string line 2 \
//     string line 3 \
//     ` },
//         {-1, COMPOSED, `` }, // """
//         {-1, LINEND, `` },
//         {-1, LINEND, `` },

        {-1, BAREWORD, `strings` },
        {-1, SPACE, ` ` },
        {-1, ASSIGN, `` },
        {-1, SPACE, ` ` },
        {-1, STRING, `abc` },
        {-1, SPACE, ` ` },
        {-1, COMPOUND, `` }, // "xx $(string1) xx"
        {-1, RAW, `xx ` },
        {-1, DELEGATE, `` },
        {-1, LPAREN, `(` },
        {-1, BAREWORD, `string1` },
        {-1, RPAREN, `)` },
        {-1, RAW, ` xx` },
        {-1, COMPOSED, `` },
        {-1, LINEND, `` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `empty1` },
        {-1, SPACE, ` ` },
        {-1, ASSIGN, `` },
        {-1, SPACE, ` ` },
        {-1, STRING, `` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `empty2` },
        {-1, SPACE, ` ` },
        {-1, ASSIGN, `` },
        {-1, SPACE, ` ` },
        {-1, COMPOUND, `` }, // ""
        {-1, COMPOSED, `` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `empty3` },
        {-1, SPACE, ` ` },
        {-1, ASSIGN, `` },
        {-1, LINEND, `` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `text1` },
        {-1, SPACE, ` ` },
        {-1, ASSIGN, `` },
        {-1, SPACE, ` ` },
        {-1, BAREWORD, `this-is-a-text` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `texts` },
        {-1, SPACE, ` ` },
        {-1, ASSIGN, `` },
        {-1, SPACE, ` ` },
        {-1, BAREWORD, `this` },
        {-1, SPACE, ` ` },
        {-1, BAREWORD, `is` },
        {-1, SPACE, ` ` },
        {-1, BAREWORD, `a` },
        {-1, SPACE, ` ` },
        {-1, BAREWORD, `text` },
        {-1, SPACE, ` ` },
        {-1, BAREWORD, `array` },
        {-1, LINEND, `` },
    }
    for i, r := range results {
        var pos, tok, lit = s.Scan()
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

// TODO: TestIntegers
func testIntegers(t *testing.T) {
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
    var s Scanner
    f := fset.AddFile(filepath.Join("TestIntegers", "src"), fset.Base(), len(src))
    s.Init(f, []byte(src), ScanMode(0), nil, nil)
    if f.Size() != len(src) {
        t.Errorf("bad file size: got %d, expected %d", f.Size(), len(src))
    }

    results := []scanResult{
        {-1, LINEND, `` },

        {-1, BAREWORD, `integer1` },
        {-1, SPACE, ` ` },
        {-1, ASSIGN, `` },
        {-1, SPACE, ` ` },
        {-1, PLUS, `` },
        {-1, INTEGER, `100` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `integer2` },
        {-1, SPACE, ` ` },
        {-1, ASSIGN, `` },
        {-1, SPACE, ` ` },
        {-1, INTEGER, `99` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `integer3` },
        {-1, SPACE, ` ` },
        {-1, ASSIGN, `` },
        {-1, SPACE, ` ` },
        {-1, INTEGER, `-38` },
        {-1, LINEND, `` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `integer4` },
        {-1, SPACE, ` ` },
        {-1, ASSIGN, `` },
        {-1, SPACE, ` ` },
        {-1, INTEGER, `10_000_000` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `integer5` },
        {-1, SPACE, ` ` },
        {-1, ASSIGN, `` },
        {-1, SPACE, ` ` },
        {-1, INTEGER, `1_2_3_4_5` },
        {-1, COMMENT, `# VALID but discouraged` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `octal1` },
        {-1, ASSIGN, `` },
        {-1, OCTAL, `01234567` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `octal2` },
        {-1, ASSIGN, `` },
        {-1, OCTAL, `01_0_000` },
        {-1, LINEND, `` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `hex1` },
        {-1, ASSIGN, `` },
        {-1, HEXADECIMAL, `0x123456789ABCDEF` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `hex2` },
        {-1, ASSIGN, `` },
        {-1, HEXADECIMAL, `0xAAAA_BBBB_1111` },
        {-1, LINEND, `` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `bin1` },
        {-1, ASSIGN, `` },
        {-1, BINARY, `0b0011001100` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `bin2` },
        {-1, ASSIGN, `` },
        {-1, BINARY, `0b1100110011` },
        {-1, LINEND, `` },
        {-1, LINEND, `` },
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

// TODO: TestDatetime
func testDatetime(t *testing.T) {
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
    var s Scanner
    f := fset.AddFile(filepath.Join("TestDatetime", "src"), fset.Base(), len(src))
    s.Init(f, []byte(src), ScanMode(0), nil, nil)
    if f.Size() != len(src) {
        t.Errorf("bad file size: got %d, expected %d", f.Size(), len(src))
    }

    results := []scanResult{
        {-1, BAREWORD, `t1` },
        {-1, ASSIGN, `` },
        {-1, DATETIME, `1979-05-27T07:32:00Z` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `t2` },
        {-1, ASSIGN, `` },
        {-1, DATETIME, `1979-05-27T07:32:00-07:00` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `t3` },
        {-1, ASSIGN, `` },
        {-1, DATETIME, `1979-05-27T07:32:00.999999-07:00` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `t4` },
        {-1, ASSIGN, `` },
        {-1, DATETIME, `1979-05-27T07:32:00` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `t5` },
        {-1, ASSIGN, `` },
        {-1, DATETIME, `1979-05-27T07:32:00.999999` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `d1` },
        {-1, ASSIGN, `` },
        {-1, DATE, `1979-05-27` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `t6` },
        {-1, ASSIGN, `` },
        {-1, TIME, `07:32:00` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `t7` },
        {-1, ASSIGN, `` },
        {-1, TIME, `07:32:00.999999` },
        {-1, LINEND, `` },
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

// TODO: TestFloats
func testFloats(t *testing.T) {
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
    var s Scanner
    f := fset.AddFile(filepath.Join("TestFloats", "src"), fset.Base(), len(src))
    s.Init(f, []byte(src), ScanMode(0), nil, nil)
    if f.Size() != len(src) {
        t.Errorf("bad file size: got %d, expected %d", f.Size(), len(src))
    }

    results := []scanResult{
        {-1, BAREWORD, `float1` },
        {-1, ASSIGN, `` },
        {-1, PLUS, `` },
        {-1, FLOAT, `1.0` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `float2` },
        {-1, ASSIGN, `` },
        {-1, FLOAT, `3.1415` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `float3` },
        {-1, ASSIGN, `` },
        {-1, MINUS, `` },
        {-1, FLOAT, `0.001` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `float4` },
        {-1, ASSIGN, `` },
        {-1, FLOAT, `5e+22` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `float5` },
        {-1, ASSIGN, `` },
        {-1, FLOAT, `1e6` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `float6` },
        {-1, ASSIGN, `` },
        {-1, MINUS, `` },
        {-1, FLOAT, `2E-2` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `float7` },
        {-1, ASSIGN, `` },
        {-1, FLOAT, `3.1415e-100` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `float8` },
        {-1, ASSIGN, `` },
        {-1, FLOAT, `6.18_16_18_16` },
        {-1, LINEND, `` },
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

// TODO: TestArrays
func testArrays(t *testing.T) {
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
    var s Scanner
    f := fset.AddFile(filepath.Join("TestArrays", "src"), fset.Base(), len(src))
    s.Init(f, []byte(src), ScanMode(0), nil, nil)
    if f.Size() != len(src) {
        t.Errorf("bad file size: got %d, expected %d", f.Size(), len(src))
    }

    results := []scanResult{
        {-1, BAREWORD, `array1` },
        {-1, ASSIGN, `` },
        {-1, BAREWORD, `text1` },
        {-1, BAREWORD, `text2` },
        {-1, BAREWORD, `text3` },
        {-1, STRING, `''` },
        {-1, INT, `1` },
        {-1, INT, `2` },
        {-1, INT, `3` },
        {-1, FLOAT, `1.2` },
        {-1, LPAREN, `` },
        {-1, BAREWORD, `a` },
        {-1, BAREWORD, `b` },
        {-1, BAREWORD, `c` },
        {-1, INT, `1` },
        {-1, INT, `2` },
        {-1, INT, `3` },
        {-1, STRING, `''` },
        {-1, STRING, `""` },
        {-1, RPAREN, `` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `array2` },
        {-1, ASSIGN, `` }, // consequence \\n and spaces are ignored
        {-1, BAREWORD, `text1` },
        {-1, BAREWORD, `text2` },
        {-1, BAREWORD, `text3` },
        {-1, STRING, `''` },
        {-1, INT, `1` },
        {-1, INT, `2` },
        {-1, INT, `3` },
        {-1, LINEND, `` },
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

// TODO: TestMaps
func testMaps(t *testing.T) {
    src := `
map1 = (
   k1 value1,
   k2 value2,
   k3 value3,
   k4 value,
)

map2 = (  k1 v1, k2 'v2 v2', k3 "v3 v3 v3", k4 v4  )
`
    var s Scanner
    f := fset.AddFile(filepath.Join("TestMaps", "src"), fset.Base(), len(src))
    s.Init(f, []byte(src), ScanMode(0), nil, nil)
    if f.Size() != len(src) {
        t.Errorf("bad file size: got %d, expected %d", f.Size(), len(src))
    }

    results := []scanResult{
        {-1, BAREWORD, `map1` },
        {-1, ASSIGN, `` },
        {-1, LPAREN, `` },
        {-1, BAREWORD, `k1` },
        {-1, BAREWORD, `value1` },
        {-1, COMMA, `` },
        {-1, BAREWORD, `k2` },
        {-1, BAREWORD, `value2` },
        {-1, COMMA, `` },
        {-1, BAREWORD, `k3` },
        {-1, BAREWORD, `value3` },
        {-1, COMMA, `` },
        {-1, BAREWORD, `k4` },
        {-1, BAREWORD, `value` },
        {-1, COMMA, `` },
        {-1, RPAREN, `` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `map2` },
        {-1, ASSIGN, `` },
        {-1, LPAREN, `` },
        {-1, BAREWORD, `k1` },
        {-1, BAREWORD, `v1` },
        {-1, COMMA, `` },
        {-1, BAREWORD, `k2` },
        {-1, STRING, `'v2 v2'` },
        {-1, COMMA, `` },
        {-1, BAREWORD, `k3` },
        {-1, STRING, `"v3 v3 v3"` },
        {-1, COMMA, `` },
        {-1, BAREWORD, `k4` },
        {-1, BAREWORD, `v4` },
        {-1, RPAREN, `` },
        {-1, LINEND, `` },
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

// TODO: TestCalls
func testCalls(t *testing.T) {
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
    s.Init(f1, []byte(src1), ScanMode(0), nil, nil)
    if f1.Size() != len(src1) {
        t.Errorf("bad file size: got %d, expected %d", f1.Size(), len(src1))
    }
    results1 := []scanResult{
        { 1, COMMENT, `# bare lets` },
        {-1, DELEGATE, `` },
        {-1, LPAREN, `` },
        {-1, BAREWORD, `let` },
        {-1, LPAREN, `` },

        {-1, LPAREN, `` },
        {-1, BAREWORD, `a` },
        {-1, STRING, `"value of a"` },
        {-1, RPAREN, `` },

        {-1, LPAREN, `` },
        {-1, BAREWORD, `b` },
        {-1, STRING, `'value of b'` },
        {-1, RPAREN, `` },

        {-1, LPAREN, `` },
        {-1, BAREWORD, `c` },
        {-1, STRING, `'value of c'` },
        {-1, RPAREN, `` },

        {-1, RPAREN, `` }, // 'let' enclosed

        {-1, LPAREN, `` },
        {-1, BAREWORD, `print` },
        {-1, STRING, `"$a.$b.$c"` },
        {-1, RPAREN, `` },

        {-1, RPAREN, `` },
        {-1, LINEND, `` },

        {-1, DELEGATE, `` },
        {-1, LPAREN, `` },
        {-1, BAREWORD, `let` },
        {-1, LPAREN, `` },

        {-1, LPAREN, `` },
        {-1, BAREWORD, `a` },
        {-1, FLOAT, `1e-10` },
        {-1, RPAREN, `` },

        {-1, LPAREN, `` },
        {-1, BAREWORD, `b` },
        {-1, DATE, `2017-01-18` },
        {-1, RPAREN, `` },

        {-1, LPAREN, `` },
        {-1, BAREWORD, `c` },
        {-1, TIME, `19:25:30` },
        {-1, RPAREN, `` },

        {-1, RPAREN, `` }, // 'let' enclosed

        {-1, LPAREN, `` },
        {-1, BAREWORD, `print` },
        {-1, STRING, `"$a $b $c"` },
        {-1, RPAREN, `` },

        {-1, RPAREN, `` },
        {-1, LINEND, `` },
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
    s.Init(f2, []byte(src2), ScanMode(0), nil, nil)
    if f2.Size() != len(src2) {
        t.Errorf("bad file size: got %d, expected %d", f2.Size(), len(src2))
    }
    results2 := []scanResult{
        { 1, COMMENT, `# binds` },

        {-1, BAREWORD, `concat` },
        {-1, ASSIGN, `` },
        {-1, DELEGATE, `` },
        {-1, LPAREN, `` },
        {-1, BAREWORD, `bind` },
        {-1, LPAREN, `` },
        {-1, BAREWORD, `a` },
        {-1, BAREWORD, `b` },
        {-1, BAREWORD, `c` },
        {-1, RPAREN, `` },
        {-1, STRING, `"$a.$b.$c"` },
        {-1, RPAREN, `` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `v1` },
        {-1, ASSIGN, `` },
        {-1, DELEGATE, `` },
        {-1, LPAREN, `` },
        {-1, BAREWORD, `concat` },
        {-1, INT, `1` },
        {-1, INT, `2` },
        {-1, INT, `3` },
        {-1, RPAREN, `` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `v2` },
        {-1, ASSIGN, `` },
        {-1, DELEGATE, `` },
        {-1, LPAREN, `` },
        {-1, BAREWORD, `concat` },
        {-1, STRING, `"a"` },
        {-1, STRING, `'b'` },
        {-1, BAREWORD, `c` },
        {-1, RPAREN, `` },
        {-1, LINEND, `` },
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

// TODO: TestRules
func testRules(t *testing.T) {
    var s Scanner

    src1 := `
# rules

prog: obj/file.o
    gcc -o $@ $<
obj/file.o: src/file.c
    gcc -c -o $@ $^
`
    f1 := fset.AddFile(filepath.Join("TestRules", "src1"), fset.Base(), len(src1))
    s.Init(f1, []byte(src1), ScanMode(0), nil, nil)
    if f1.Size() != len(src1) {
        t.Errorf("bad file size: got %d, expected %d", f1.Size(), len(src1))
    }
    results1 := []scanResult{
        { 1, COMMENT, `# rules` },

        {-1, BAREWORD, `prog` },
        {-1, COLON, `` },
        {-1, BAREWORD, `obj` },
        {-1, PCON, `` },
        {-1, BAREWORD, `file.o` },
        {-1, LINEND, `` },
        {-1, RECIPE, `gcc -o $@ $<` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `obj` },
        {-1, PCON, `` },
        {-1, BAREWORD, `file.o` },
        {-1, COLON, `` },
        {-1, BAREWORD, `src` },
        {-1, PCON, `` },
        {-1, BAREWORD, `file.c` },
        {-1, LINEND, `` },
        {-1, RECIPE, `gcc -c -o $@ $^` },
        {-1, LINEND, `` },
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
    s.Init(f2, []byte(src2), ScanMode(0), nil, nil)
    if f2.Size() != len(src2) {
        t.Errorf("bad file size: got %d, expected %d", f2.Size(), len(src2))
    }
    results2 := []scanResult{
        { 1, COMMENT, `# rules` },

        {-1, BAREWORD, `start` },
        {-1, COLON, `` },
        {-1, LINEND, `` },
        {-1, RECIPE, `echo one` },
        {-1, LINEND, `` },
        {-1, RECIPE, `echo one` },
        {-1, LINEND, `` },
        {-1, RECIPE, `echo one` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `start` },
        {-1, DOLON, `` },
        {-1, LINEND, `` },
        {-1, RECIPE, `echo two` },
        {-1, LINEND, `` },
        {-1, RECIPE, `echo two` },
        {-1, LINEND, `` },
        {-1, RECIPE, `echo two` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `start` },
        {-1, DOLON, `` },
        {-1, LINEND, `` },
        {-1, RECIPE, `echo three` },
        {-1, LINEND, `` },
        {-1, RECIPE, `echo three` },
        {-1, LINEND, `` },
        {-1, RECIPE, `echo three` },
        {-1, LINEND, `` },
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
    s.Init(f3, []byte(src3), ScanMode(0), nil, nil)
    if f3.Size() != len(src3) {
        t.Errorf("bad file size: got %d, expected %d", f3.Size(), len(src3))
    }
    results3 := []scanResult{
        { 1, COMMENT, `# rules` },

        {-1, BAREWORD, `start` },
        // {-1, COLON_EXC, `` },
        {-1, LINEND, `` },
        {-1, RECIPE, `echo okay` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `start` },
        // {-1, COLON_QUE, `` },
        {-1, LINEND, `` },
        {-1, RECIPE, `test src/file.c` },
        {-1, LINEND, `` },
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
    s.Init(f4, []byte(src4), ScanMode(0), nil, nil)
    if f4.Size() != len(src4) {
        t.Errorf("bad file size: got %d, expected %d", f4.Size(), len(src4))
    }
    results4 := []scanResult{
        { 1, COMMENT, `# brack rules` },

        {-1, BAREWORD, `start` },
        // {-1, COLON_LBK, `` },
        {-1, BAREWORD, `shell` },
        // {-1, COLON_RBK, `` },
        {-1, LINEND, `` },
        {-1, RECIPE, `echo okay` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `start` },
        // {-1, COLON_LBE, `` },
        {-1, BAREWORD, `shell` },
        // {-1, COLON_RBK, `` },
        {-1, LINEND, `` },
        {-1, RECIPE, `echo okay okay` },
        {-1, LINEND, `` },

        {-1, BAREWORD, `start` },
        // {-1, COLON_LBQ, `` },
        {-1, BAREWORD, `shell` },
        // {-1, COLON_RBK, `` },
        {-1, LINEND, `` },
        {-1, RECIPE, `test ok ok` },
        {-1, LINEND, `` },
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

// TODO: TestProgConstructs
func testProgConstructs(t *testing.T) {
    var s Scanner

    src1 := `
project A

include modules/foo.smart

instance
`
    f1 := fset.AddFile(filepath.Join("TestProgConstructs", "src1"), fset.Base(), len(src1))
    s.Init(f1, []byte(src1), ScanMode(0), nil, nil)
    if f1.Size() != len(src1) {
        t.Errorf("bad file size: got %d, expected %d", f1.Size(), len(src1))
    }
    results1 := []scanResult{
        {-1, PROJECT, `project` },
        {-1, BAREWORD, `A` },
        {-1, LINEND, `` },

        {-1, INCLUDE, `include` },
        {-1, BAREWORD, `modules` },
        {-1, PCON, `` },
        {-1, BAREWORD, `foo.smart` },
        {-1, LINEND, `` },

        {-1, INSTANCE, `instance` },
        {-1, LINEND, `` },
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
module M1

use ( M2 M3 )
use (
  M4
  M5
)
`
    f2 := fset.AddFile(filepath.Join("TestProgConstructs", "src2"), fset.Base(), len(src2))
    s.Init(f2, []byte(src2), ScanMode(0), nil, nil)
    if f2.Size() != len(src2) {
        t.Errorf("bad file size: got %d, expected %d", f2.Size(), len(src2))
    }
    results2 := []scanResult{
        {-1, MODULE, `module` },
        {-1, BAREWORD, `M1` },
        {-1, LINEND, `` },

        {-1, USE, `use` },
        {-1, LPAREN, `` },
        {-1, BAREWORD, `M2` },
        {-1, BAREWORD, `M3` },
        {-1, RPAREN, `` },
        {-1, LINEND, `` },

        {-1, USE, `use` },
        {-1, LPAREN, `` },
        {-1, BAREWORD, `M4` },
        {-1, BAREWORD, `M5` },
        {-1, RPAREN, `` },
        {-1, LINEND, `` },
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
