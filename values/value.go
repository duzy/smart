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
        "strconv"
        "time"
)

var None = &types.NoneValue{}

func Any(v interface{}) *types.AnyValue {
        return &types.AnyValue{ V:v }
}

func IntLit(s string) (v *types.IntValue) {
        if i, e := strconv.ParseInt(s, 10, 64); e == nil {
                v = &types.IntValue{i}
        }
        return
}
func Int(s int64) (v *types.IntValue) {
        return &types.IntValue{s}
}

func FloatLit(s string) (v *types.FloatValue) {
        if f, e := strconv.ParseFloat(s, 64); e == nil {
                v = &types.FloatValue{f}
        }
        return
}
func Float(s float64) (v *types.FloatValue) {
        return &types.FloatValue{s}
}

func DateTimeLit(s string) (v *types.DateTimeValue) {
        // time.RFC3339Nano
        if t, e := time.Parse("2006-01-02T15:04:05.999999999Z07:00", s); e == nil {
                v = &types.DateTimeValue{t}
        }
        return
}
func DateTime(s time.Time) (v *types.DateTimeValue) {
        return &types.DateTimeValue{s}
}

func DateLit(s string) (v *types.DateValue) {
        if t, e := time.Parse("2006-01-02", s); e == nil {
                v = &types.DateValue{types.DateTimeValue{t}}
        }
        return
}
func Date(s time.Time) (v *types.DateValue) {
        return &types.DateValue{types.DateTimeValue{s}}
}

func TimeLit(s string) (v *types.TimeValue) {
        if t, e := time.Parse("15:04:05.999999999Z07:00", s); e == nil {
                v = &types.TimeValue{types.DateTimeValue{t}}
        }
        return
}
func Time(t time.Time) (v *types.TimeValue) {
        return &types.TimeValue{types.DateTimeValue{t}}
}

func UriLit(s string) (v *types.UriValue) {
        if u, e := url.Parse(s); e == nil {
                v = &types.UriValue{u}
        }
        return
}
func Uri(s *url.URL) (v *types.UriValue) {
        return &types.UriValue{s}
}

func StringLit(s string) (v *types.StringValue) {
        return &types.StringValue{s}
}
func String(s string) (v *types.StringValue) {
        return &types.StringValue{s}
}

func Bareword(s string) (v *types.BarewordValue) {
        return &types.BarewordValue{s}
}

func Barecomp(elems... types.Value) (v *types.BarecompValue) {
        return &types.BarecompValue{types.Elements{elems}}
}

func Barefile(name types.Value, ext string) (v *types.BarefileValue) {
        return &types.BarefileValue{name, ext}
}

func Path(segments... types.Value) (v *types.PathValue) {
        return &types.PathValue{segments}
}

func Flag(name types.Value) (v *types.FlagValue) {
        return &types.FlagValue{name}
}

func Compound(elems... types.Value) (v *types.CompoundValue) {
        return &types.CompoundValue{types.Elements{elems}}
}

func List(elems... types.Value) (v *types.ListValue) {
        return &types.ListValue{types.Elements{elems}}
}

func Group(elems... types.Value) (v *types.GroupValue) {
        return &types.GroupValue{types.ListValue{types.Elements{elems}}}
}

func Pair(k, v types.Value) (p *types.PairValue) {
        if k.Type().Info()&types.IsKeyName != 0 {
                p = &types.PairValue{nil, nil}
                p.SetKey(k)
                p.SetValue(v)
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
        default:   s = "\\" + s // give back the '\' character
        }
        return s
}

func Literal(tok token.Token, s string) (v types.Value) {
        switch tok {
        default:             v = None
        case token.INT:      v = IntLit(s)
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
