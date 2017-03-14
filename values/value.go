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

// Scalar literal/value types.
type (
        value struct {
                typ types.Type
        }

        literal struct {
                value
                pos token.Pos
        }
        
        Int struct {
                literal
                value int64
        }

        Float struct {
                literal
                value float64
        }

        dt struct { // date-time
                literal
                value time.Time
        }

        DateTime struct { dt }
        Date struct { dt }
        Time struct { dt }

        Uri struct {
                literal
                value *url.URL //string
        }

        String struct {
                literal
                value string
        }

        Bareword struct {
                literal
                value string
        }

        Compound struct {
                pos, end token.Pos
                elems []types.Value
        }

        List struct {
                pos, end token.Pos
                elems []types.Value
        }

        Group struct {
                List
        }

        Map struct {
                pos, end token.Pos
                elems map[string]types.Value
        }
)

var None = &value{ types.None }

func NewIntLiteral(pos token.Pos, s string) (v *Int) {
        if i, e := strconv.ParseInt(s, 10, 64); e == nil {
                v = &Int{literal{value{types.Int}, pos}, i}
        }
        return
}
func NewInt(s int64) (v *Int) {
        return &Int{literal{value{types.Int}, token.NoPos}, s}
}

func NewFloatLiteral(pos token.Pos, s string) (v *Float) {
        if f, e := strconv.ParseFloat(s, 64); e == nil {
                v = &Float{literal{value{types.Float}, pos}, f}
        }
        return
}
func NewFloat(s float64) (v *Float) {
        return &Float{literal{value{types.Float}, token.NoPos}, s}
}

func NewDateTimeLiteral(pos token.Pos, s string) (v *DateTime) {
        // time.RFC3339Nano
        if t, e := time.Parse("2006-01-02T15:04:05.999999999Z07:00", s); e == nil {
                v = &DateTime{dt{literal{value{types.DateTime}, pos}, t}}
        }
        return
}
func NewDateTime(s time.Time) (v *DateTime) {
        return &DateTime{dt{literal{value{types.DateTime}, token.NoPos}, s}}
}

func NewDateLiteral(pos token.Pos, s string) (v *Date) {
        if t, e := time.Parse("2006-01-02", s); e == nil {
                v = &Date{dt{literal{value{types.Date}, pos}, t}}
        }
        return
}
func NewDate(s time.Time) (v *Date) {
        return &Date{dt{literal{value{types.Date}, token.NoPos}, s}}
}

func NewTimeLiteral(pos token.Pos, s string) (v *Time) {
        if t, e := time.Parse("15:04:05.999999999Z07:00", s); e == nil {
                v = &Time{dt{literal{value{types.Time}, pos}, t}}
        }
        return
}
func NewTime(s time.Time) (v *Time) {
        return &Time{dt{literal{value{types.Time}, token.NoPos}, s}}
}

func NewUriLiteral(pos token.Pos, s string) (v *Uri) { // RFC 3986
        if u, e := url.Parse(s); e == nil {
                v = &Uri{literal{value{types.Uri}, pos}, u}
        }
        return
}
func NewUri(s *url.URL) (v *Uri) {
        return &Uri{literal{value{types.Uri}, token.NoPos}, s}
}

func NewStringLiteral(pos token.Pos, s string) (v *String) {
        return &String{literal{value{types.String}, pos}, s}
}
func NewString(s string) (v *String) {
        return &String{literal{value{types.String}, token.NoPos}, s}
}

func NewBarewordLiteral(pos token.Pos, s string) (v *Bareword) {
        return &Bareword{literal{value{types.Bareword}, pos}, s}
}
func NewBareword(s string) (v *Bareword) {
        return &Bareword{literal{value{types.Bareword}, token.NoPos}, s}
}

func NewCompound(pos, end token.Pos, values... types.Value) (v *Compound) {
        return &Compound{pos: pos, end: end, elems: values}
}

func NewList(pos, end token.Pos, values... types.Value) (v *List) {
        return &List{pos: pos, end: end, elems: values}
}

func NewGroup(pos, end token.Pos, values... types.Value) (v *Group) {
        return &Group{List{pos: pos, end: end, elems: values}}
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

func NewLiteralValue(pos token.Pos, tok token.Token, s string) (v types.Value) {
        switch tok {
        default:             v = None
        case token.INT:      v = NewIntLiteral(pos, s)
        case token.FLOAT:    v = NewFloatLiteral(pos, s)
        case token.DATETIME: v = NewDateTimeLiteral(pos, s)
        case token.DATE:     v = NewDateLiteral(pos, s)
        case token.TIME:     v = NewTimeLiteral(pos, s)
        case token.URI:      v = NewUriLiteral(pos, s)
        case token.BAREWORD: v = NewBarewordLiteral(pos, s)
        case token.STRING:   v = NewStringLiteral(pos, s)
        case token.ESCAPE:   v = NewStringLiteral(pos, EscapeChar(s))
        }
        return
}

func Make(in interface{}) (out types.Value) {
        switch v := in.(type) {
        case int:         out = NewInt(int64(v))
        case int32:       out = NewInt(int64(v))
        case int64:       out = NewInt(v)
        //case float:     out = NewFloat(float64(v))
        case float32:     out = NewFloat(float64(v))
        case float64:     out = NewFloat(v)
        case string:      out = NewString(v)
        case time.Time:   out = NewDateTime(v) // FIXME: NewDate, NewTime
        case types.Value: out = v;
        }
        return None
}

func MakeAll(in... interface{}) (out []types.Value) {
        for _, v := range in {
                out = append(out, Make(v))
        }
        return
}

func (p *value) Type() types.Type    { return p.typ }
func (p *Compound) Type() types.Type { return types.Compound }
func (p *List) Type() types.Type     { return types.List }
func (p *Group) Type() types.Type    { return types.Group }
func (p *Map) Type() types.Type      { return types.Map }

func (p *value) String() string    { return "" }
func (p *Int) String() string      { return strconv.FormatInt(int64(p.value),10) } // Itoa
func (p *Float) String() string    { return strconv.FormatFloat(float64(p.value),'g', -1, 64) }
func (p *DateTime) String() string { return time.Time(p.value).Format("2006-01-02T15:04:05.999999999Z07:00") } // time.RFC3339Nano
func (p *Date) String() string     { return time.Time(p.value).Format("2006-01-02") }
func (p *Time) String() string     { return time.Time(p.value).Format("15:04:05.999999999Z07:00") }
func (p *Uri) String() string      { return p.value.String() }
func (p *String) String() string   { return p.value }
func (p *Bareword) String() string { return p.value }
func (p *Compound) String() (s string) {
        for _, e := range p.elems {
                s += e.String()
        }
        return
}
func (p *List) String() (s string) {
        for i, e := range p.elems {
                if 0 < i {
                        s += " "
                }
                s += e.String()
        }
        return
}
func (p *Group) String() string {
        return "(" + p.List.String() + ")"
}
/* func (p *Map) String() string {
        return "(" + p.List.String() + ")"
} */

func (p *value) Integer() int64    { return 0 }
func (p *Int) Integer() int64      { return p.value }
func (p *Float) Integer() int64    { return int64(p.value) }
func (p *DateTime) Integer() int64 { return p.value.Unix() }
func (p *Date) Integer() int64     { return p.value.Unix() }
func (p *Time) Integer() int64     { return p.value.Unix() }
func (p *Uri) Integer() int64      { return int64(len(p.value.String())) }
func (p *String) Integer() int64   { i, _ := strconv.ParseInt(p.value, 10, 64); return i }
func (p *Bareword) Integer() int64 { return 0 }
func (p *Compound) Integer() int64 { return int64(len(p.elems)) }
func (p *List) Integer() int64     { return int64(len(p.elems)) }
func (p *Group) Integer() int64    { return int64(len(p.List.elems)) }

func (p *value) Float() float64    { return 0.0 }
func (p *Int) Float() float64      { return float64(p.value) }
func (p *Float) Float() float64    { return p.value }
func (p *DateTime) Float() float64 { return float64(p.Integer()) }
func (p *Date) Float() float64     { return float64(p.Integer()) }
func (p *Time) Float() float64     { return float64(p.Integer()) }
func (p *Uri) Float() float64      { return float64(p.Integer()) }
func (p *String) Float() float64   { return float64(p.Integer()) }
func (p *Bareword) Float() float64 { return float64(p.Integer()) }
func (p *Compound) Float() float64 { return float64(p.Integer()) }
func (p *List) Float() float64     { return float64(p.Integer()) }
func (p *Group) Float() float64    { return float64(p.Integer()) }

func (p *List) ToCompound() *Compound { return &Compound{token.NoPos, token.NoPos, p.elems} }

func (p *Compound) ToList() *List { return &List{token.NoPos, token.NoPos, p.elems} }
func (p *Group) ToList() *List    { return &p.List }
