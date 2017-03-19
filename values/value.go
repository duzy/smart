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

// Scalar literal/value types. A literal is literrally written in the source code,
// a value is calculated from the program.
type (
        value struct {
                typ types.Type
        }
        
        IntValue struct {
                value
                v int64
        }
        
        FloatValue struct {
                value
                v float64
        }

        datetimeValue struct {
                value
                v time.Time
        }
        DateTimeValue struct { datetimeValue }
        DateValue     struct { datetimeValue }
        TimeValue     struct { datetimeValue }
        
        UriValue struct {
                value
                v *url.URL
        }

        StringValue struct {
                value
                v string
        }

        BarewordValue struct {
                value
                v string
        }

        CompoundValue struct {
                value
                elems []types.Value
        }

        ListValue struct {
                value
                elems []types.Value
        }

        GroupValue struct {
                ListValue
        }

        MapValue struct {
                value
                elems map[string]types.Value
                mv map[types.Value]types.Value
        }

        PairValue struct { // key=value
                value
                k types.Value
                v types.Value
        }

        IntLiteral struct {
                IntValue
                pos token.Pos
        }

        FloatLiteral struct {
                FloatValue
                pos token.Pos
        }

        DateTimeLiteral struct {
                DateTimeValue
                pos token.Pos
        }
        DateLiteral struct {
                DateValue
                pos token.Pos
        }
        TimeLiteral struct {
                TimeValue
                pos token.Pos
        }
        
        UriLiteral struct {
                UriValue
                pos token.Pos
        }

        StringLiteral struct {
                StringValue
                pos token.Pos
        }

        BarewordLiteral struct {
                BarewordValue
                pos token.Pos
        }

        CompoundLiteral struct {
                CompoundValue
                pos token.Pos
        }

        ListLiteral struct {
                ListValue
                pos token.Pos
        }

        GroupLiteral struct {
                GroupValue
                pos token.Pos
        }

        MapLiteral struct {
                MapValue
                pos token.Pos
        }

        PairLiteral struct {
                PairValue
                pos token.Pos
        }
)

var None = &value{ types.None }

func IntLit(pos token.Pos, s string) (v *IntLiteral) {
        if i, e := strconv.ParseInt(s, 10, 64); e == nil {
                v = &IntLiteral{IntValue{value{types.Int}, i}, pos}
        }
        return
}
func Int(s int64) (v *IntValue) {
        return &IntValue{value{types.Int}, s}
}

func FloatLit(pos token.Pos, s string) (v *FloatLiteral) {
        if f, e := strconv.ParseFloat(s, 64); e == nil {
                v = &FloatLiteral{FloatValue{value{types.Float}, f}, pos}
        }
        return
}
func Float(s float64) (v *FloatValue) {
        return &FloatValue{value{types.Float}, s}
}

func DateTimeLit(pos token.Pos, s string) (v *DateTimeLiteral) {
        // time.RFC3339Nano
        if t, e := time.Parse("2006-01-02T15:04:05.999999999Z07:00", s); e == nil {
                v = &DateTimeLiteral{DateTimeValue{datetimeValue{value{types.DateTime}, t}}, pos}
        }
        return
}
func DateTime(s time.Time) (v *DateTimeValue) {
        return &DateTimeValue{datetimeValue{value{types.DateTime}, s}}
}

func DateLit(pos token.Pos, s string) (v *DateLiteral) {
        if t, e := time.Parse("2006-01-02", s); e == nil {
                v = &DateLiteral{DateValue{datetimeValue{value{types.Date}, t}}, pos}
        }
        return
}
func Date(s time.Time) (v *DateValue) {
        return &DateValue{datetimeValue{value{types.Date}, s}}
}

func TimeLit(pos token.Pos, s string) (v *TimeLiteral) {
        if t, e := time.Parse("15:04:05.999999999Z07:00", s); e == nil {
                v = &TimeLiteral{TimeValue{datetimeValue{value{types.Time}, t}}, pos}
        }
        return
}
func Time(t time.Time) (v *TimeValue) {
        return &TimeValue{datetimeValue{value{types.Time}, t}}
}

func UriLit(pos token.Pos, s string) (v *UriLiteral) {
        if u, e := url.Parse(s); e == nil {
                v = &UriLiteral{UriValue{value{types.Uri}, u}, pos}
        }
        return
}
func Uri(s *url.URL) (v *UriValue) {
        return &UriValue{value{types.Uri}, s}
}

func StringLit(pos token.Pos, s string) (v *StringLiteral) {
        return &StringLiteral{StringValue{value{types.String}, s}, pos}
}
func String(s string) (v *StringValue) {
        return &StringValue{value{types.String}, s}
}

func BarewordLit(pos token.Pos, s string) (v *BarewordLiteral) {
        return &BarewordLiteral{BarewordValue{value{types.Bareword}, s}, pos}
}
func Bareword(s string) (v *BarewordValue) {
        return &BarewordValue{value{types.Bareword}, s}
}

func CompoundLit(pos token.Pos, elems... types.Value) (v *CompoundLiteral) {
        return &CompoundLiteral{CompoundValue{value{types.Compound}, elems}, pos}
}
func Compound(elems... types.Value) (v *CompoundValue) {
        return &CompoundValue{value{types.Compound}, elems}
}

func ListLit(pos token.Pos, elems... types.Value) (v *ListLiteral) {
        return &ListLiteral{ListValue{value{types.List}, elems}, pos}
}
func List(elems... types.Value) (v *ListValue) {
        return &ListValue{value{types.List}, elems}
}

func GroupLit(pos token.Pos, elems... types.Value) (v *GroupLiteral) {
        return &GroupLiteral{GroupValue{ListValue{value{types.List}, elems}}, pos}
}
func Group(elems... types.Value) (v *GroupValue) {
        return &GroupValue{ListValue{value{types.List}, elems}}
}

func PairLit(pos token.Pos, k, v types.Value) (p *PairLiteral) {
        if k.Type().Info()&types.IsKeyName != 0 {
                p = &PairLiteral{PairValue{value{types.Pair}, nil, nil}, pos}
                p.SetKey(k)
                p.SetValue(v)
        }
        return
}
func Pair(k, v types.Value) (p *PairValue) {
        if k.Type().Info()&types.IsKeyName != 0 {
                p = &PairValue{value{types.Pair}, nil, nil}
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
        }
        return s
}

func Literal(pos token.Pos, tok token.Token, s string) (v types.Value) {
        switch tok {
        default:             v = None
        case token.INT:      v = IntLit(pos, s)
        case token.FLOAT:    v = FloatLit(pos, s)
        case token.DATETIME: v = DateTimeLit(pos, s)
        case token.DATE:     v = DateLit(pos, s)
        case token.TIME:     v = TimeLit(pos, s)
        case token.URI:      v = UriLit(pos, s)
        case token.BAREWORD: v = BarewordLit(pos, s)
        case token.STRING:   v = StringLit(pos, s)
        case token.ESCAPE:   v = StringLit(pos, EscapeChar(s))
        }
        return
}

func Make(in interface{}) (out types.Value) {
        switch v := in.(type) {
        case int:         out = Int(int64(v))
        case int32:       out = Int(int64(v))
        case int64:       out = Int(v)
        //case float:     out = Float(float64(v))
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

func (p *value) Type() types.Type       { return p.typ }

func (p *value) String() string         { return "" }
func (p *IntValue) String() string      { return strconv.FormatInt(int64(p.v),10) } // Itoa
func (p *FloatValue) String() string    { return strconv.FormatFloat(float64(p.v),'g', -1, 64) }
func (p *DateTimeValue) String() string { return time.Time(p.v).Format("2006-01-02T15:04:05.999999999Z07:00") } // time.RFC3339Nano
func (p *DateValue) String() string     { return time.Time(p.v).Format("2006-01-02") }
func (p *TimeValue) String() string     { return time.Time(p.v).Format("15:04:05.999999999Z07:00") }
func (p *UriValue) String() string      { return p.v.String() }
func (p *StringValue) String() string   { return p.v }
func (p *BarewordValue) String() string { return p.v }
func (p *CompoundValue) String() (s string) {
        for _, e := range p.elems {
                s += e.String()
        }
        return
}
func (p *ListValue) String() (s string) {
        for i, e := range p.elems {
                if 0 < i {
                        s += " "
                }
                s += e.String()
        }
        return
}
func (p *GroupValue) String() string {
        return "(" + p.ListValue.String() + ")"
}
/* func (p *Map) String() string {
        return "(" + p.List.String() + ")"
} */
func (p *PairValue) String() string {
        return p.k.String() + "=" + p.v.String()
}

func (p *value) Integer() int64         { return 0 }
func (p *IntValue) Integer() int64      { return p.v }
func (p *FloatValue) Integer() int64    { return int64(p.v) }
func (p *DateTimeValue) Integer() int64 { return p.v.Unix() }
func (p *DateValue) Integer() int64     { return p.v.Unix() }
func (p *TimeValue) Integer() int64     { return p.v.Unix() }
func (p *UriValue) Integer() int64      { return int64(len(p.v.String())) }
func (p *StringValue) Integer() int64   { i, _ := strconv.ParseInt(p.v, 10, 64); return i }
func (p *BarewordValue) Integer() int64 { return 0 }
func (p *CompoundValue) Integer() int64 { return int64(len(p.elems)) }
func (p *ListValue) Integer() int64     { return int64(len(p.elems)) }
func (p *GroupValue) Integer() int64    { return int64(len(p.ListValue.elems)) }
func (p *PairValue) Integer() int64     { return p.v.Integer() }

func (p *value) Float() float64         { return 0.0 }
func (p *IntValue) Float() float64      { return float64(p.v) }
func (p *FloatValue) Float() float64    { return p.v }
func (p *DateTimeValue) Float() float64 { return float64(p.Integer()) }
func (p *DateValue) Float() float64     { return float64(p.Integer()) }
func (p *TimeValue) Float() float64     { return float64(p.Integer()) }
func (p *UriValue) Float() float64      { return float64(p.Integer()) }
func (p *StringValue) Float() float64   { return float64(p.Integer()) }
func (p *BarewordValue) Float() float64 { return float64(p.Integer()) }
func (p *CompoundValue) Float() float64 { return float64(p.Integer()) }
func (p *ListValue) Float() float64     { return float64(p.Integer()) }
func (p *GroupValue) Float() float64    { return float64(p.Integer()) }
func (p *PairValue) Float() float64     { return p.v.Float() }

func (p *ListValue) Len() int                   { return len(p.elems) }
func (p *ListValue) Append(v types.Value)       { p.elems = append(p.elems, v) }
func (p *ListValue) Get(n int) (v types.Value)  { if n>=0 && n<len(p.elems) { v = p.elems[n] }; return }
func (p *ListValue) ToCompound() *CompoundValue { return &CompoundValue{value{types.Compound}, p.elems} }
func (p *GroupValue) ToList() *ListValue        { return &p.ListValue }
func (p *CompoundValue) ToList() *ListValue     { return &ListValue{value{types.List}, p.elems} }
func (p *CompoundValue) Append(v types.Value)   { p.elems = append(p.elems, v) }

func (p *PairValue) SetKey(k types.Value) {
        switch o := k.(type) {
        case *PairValue:   k = o.k
        case *PairLiteral: k = o.k
        }
        if k.Type().Info()&types.IsKeyName != 0 {
                p.k = k
        } else {
                p.k = nil
        }
}

func (p *PairValue) SetValue(v types.Value) {
        /* switch o := v.(type) {
        case *PairValue:   v = o.v
        case *PairLiteral: v = o.v
        } */
        p.v = v
}
