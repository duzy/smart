//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package values

import (
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/token"
        "net/url"
        "strings"
        "strconv"
        "time"
        "fmt"
)

var None = types.UniversalNone

func Any(v interface{}) *types.Any {
        return &types.Any{ Value:v }
}

func BinLit(s string) (v *types.Bin) {
        if i, e := strconv.ParseInt(s, 2, 64); e == nil {
                v = Bin(i)
        }
        return
}
func Bin(i int64) (v *types.Bin) {
        v = new(types.Bin)
        v.Value = i
        return
}

func OctLit(s string) (v *types.Oct) {
        if i, e := strconv.ParseInt(s, 8, 64); e == nil {
                v = Oct(i)
        }
        return
}
func Oct(i int64) (v *types.Oct) {
        v = new(types.Oct)
        v.Value = i
        return
}

func IntLit(s string) (v *types.Int) {
        if i, e := strconv.ParseInt(s, 10, 64); e == nil {
                v = Int(i)
        }
        return
}
func Int(i int64) (v *types.Int) {
        v = new(types.Int)
        v.Value = i
        return
}

func HexLit(s string) (v *types.Hex) {
        if i, e := strconv.ParseInt(s, 16, 64); e == nil {
                v = Hex(i)
        }
        return
}
func Hex(i int64) (v *types.Hex) {
        v = new(types.Hex)
        v.Value = i
        return
}

func FloatLit(s string) *types.Float {
        f, _ := strconv.ParseFloat(strings.Replace(s, "_", "", -1), 64)
        return Float(f)
}
func Float(f float64) (v *types.Float) {
        v = new(types.Float)
        v.Value = f
        return
}

func DateTimeLit(s string) (v *types.DateTime) {
        // time.RFC3339Nano
        if t, e := time.Parse("2006-01-02T15:04:05.999999999Z07:00", s); e == nil {
                v = &types.DateTime{t}
        }
        return
}
func DateTime(s time.Time) (v *types.DateTime) {
        return &types.DateTime{s}
}

func DateLit(s string) (v *types.Date) {
        if t, e := time.Parse("2006-01-02", s); e == nil {
                v = &types.Date{types.DateTime{t}}
        }
        return
}
func Date(s time.Time) (v *types.Date) {
        return &types.Date{types.DateTime{s}}
}

func TimeLit(s string) (v *types.Time) {
        if t, e := time.Parse("15:04:05.999999999Z07:00", s); e == nil {
                v = &types.Time{types.DateTime{t}}
        }
        return
}
func Time(t time.Time) (v *types.Time) {
        return &types.Time{types.DateTime{t}}
}

func UriLit(s string) (v *types.Uri) {
        if u, e := url.Parse(s); e == nil {
                v = &types.Uri{u}
        }
        return
}
func Uri(s *url.URL) (v *types.Uri) {
        return &types.Uri{s}
}

func StringLit(s string) (v *types.String) {
        return &types.String{s}
}
func String(s string) (v *types.String) {
        return &types.String{s}
}

func Bareword(s string) (v *types.Bareword) {
        return &types.Bareword{s}
}

func Barecomp(elems... types.Value) (v *types.Barecomp) {
        return &types.Barecomp{types.Elements{elems}}
}

func Barefile(name types.Value, ext string) (v *types.Barefile) {
        return &types.Barefile{name, ext}
}

func Globfile(tok token.Token, ext string) (v *types.Globfile) {
        return &types.Globfile{tok, ext}
}

func PercentPattern(prefix, suffix types.Value) types.Pattern {
        if prefix == nil { prefix = None }
        if suffix == nil { suffix = None }
        return &types.PercentPattern{
                Prefix: prefix,
                Suffix: suffix,
        }
}

func Path(segments... types.Value) (v *types.Path) {
        return &types.Path{types.Elements{segments}}
}
func PathSeg(ch rune) (v *types.PathSeg) {
        return &types.PathSeg{ Value:ch }
}

func File(v types.Value, s string) (fv *types.File) {
        return &types.File{ Value:v, Name:s }
}

func Flag(name types.Value) (v *types.Flag) {
        return &types.Flag{name}
}

func Compound(elems... types.Value) (v *types.Compound) {
        return &types.Compound{types.Elements{elems}}
}

func List(elems... types.Value) (v *types.List) {
        return &types.List{types.Elements{elems}}
}

func Group(elems... types.Value) (v *types.Group) {
        return &types.Group{types.List{types.Elements{elems}}}
}

func Pair(k, v types.Value) (p *types.Pair) {
        if k.Type().Bits()&types.IsKeyName != 0 {
                p = &types.Pair{nil, nil}
                p.SetKey(k)
                p.SetValue(v)
        } else {
                panic(fmt.Errorf("'%T' is not key type", k))
        }
        return
}

func EscapeChar(s string) string {
        switch s {
        case "a":  s = "\a"
        case "b":  s = "\b"
        case "f":  s = "\f"
        case "n":  s = "\n"
        case "r":  s = "\r"
        case "t":  s = "\t"
        case "v":  s = "\v"
        case "\\": s = "\\"
        case "$":  s = "$"
        case "&":  s = "&"
        default:   s = "\\" + s // give back the '\' character
        }
        return s
}

func Literal(tok token.Token, s string) (v types.Value) {
        switch tok {
        default:             v = None
        case token.BIN:   v = BinLit(s)
        case token.OCT:      v = OctLit(s)
        case token.INT:      v = IntLit(s)
        case token.HEX:      v = HexLit(s)
        case token.FLOAT:    v = FloatLit(s)
        case token.DATETIME: v = DateTimeLit(s)
        case token.DATE:     v = DateLit(s)
        case token.TIME:     v = TimeLit(s)
        case token.URI:      v = UriLit(s)
        case token.BAREWORD: v = Bareword(s)
        case token.STRING:   v = StringLit(s)
        case token.ESCAPE:   v = StringLit(EscapeChar(s))
        }
        return
}

func Make(in interface{}) (out types.Value) {
        switch v := in.(type) {
        case int:         out = Int(int64(v))
        case int32:       out = Int(int64(v))
        case int64:       out = Int(v)
        case float32:     out = Float(float64(v))
        case float64:     out = Float(v)
        case string:      out = String(v)
        case time.Time:   out = DateTime(v) // FIXME: NewDate, NewTime
        case types.Value: out = v
        default:          out = None
        }
        return
}

func MakeAll(in... interface{}) (out []types.Value) {
        for _, v := range in {
                out = append(out, Make(v))
        }
        return
}
