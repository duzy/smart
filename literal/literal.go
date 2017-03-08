//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package literal

import (
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/token"
        "net/url"
        "strconv"
        "time"
)

// Scalar literal value types.
type (
        Int struct {
                types.Literal
                value int64
        }

        Float struct {
                types.Literal
                value float64
        }

        dt struct { // date-time
                types.Literal
                value time.Time
        }

        DateTime struct { dt }
        Date struct { dt }
        Time struct { dt }

        Uri struct {
                types.Literal
                value *url.URL //string
        }

        String struct {
                types.Literal
                value string
        }

        Bareword struct {
                types.Literal
                value string
        }
)

func NewInt(pos token.Pos, value string) (v *Int) {
        if i, e := strconv.ParseInt(value, 10, 64); e == nil {
                v = &Int{Literal: types.NewLiteral(types.Int, pos), value:i}
        }
        return
}

func NewFloat(pos token.Pos, value string) (v *Float) {
        if f, e := strconv.ParseFloat(value, 64); e == nil {
                v = &Float{Literal: types.NewLiteral(types.Float, pos), value:f}
        }
        return
}

func NewDateTime(pos token.Pos, value string) (v *DateTime) {
        // time.RFC3339Nano
        if t, e := time.Parse("2006-01-02T15:04:05.999999999Z07:00", value); e == nil {
                v = &DateTime{dt{Literal: types.NewLiteral(types.DateTime, pos), value:t}}
        }
        return
}

func NewDate(pos token.Pos, value string) (v *Date) {
        if t, e := time.Parse("2006-01-02", value); e == nil {
                v = &Date{dt{Literal: types.NewLiteral(types.Date, pos), value:t}}
        }
        return
}

func NewTime(pos token.Pos, value string) (v *Time) {
        if t, e := time.Parse("15:04:05.999999999Z07:00", value); e == nil {
                v = &Time{dt{Literal: types.NewLiteral(types.Time, pos), value:t}}
        }
        return
}

func NewUri(pos token.Pos, value string) (v *Uri) { // RFC 3986
        if u, e := url.Parse(value); e == nil {
                v = &Uri{Literal: types.NewLiteral(types.Uri, pos), value:u}
        }
        return
}

func NewString(pos token.Pos, value string) (v *String) {
        return &String{Literal: types.NewLiteral(types.String, pos), value:value}
}

func NewBareword(pos token.Pos, value string) (v *Bareword) {
        return &Bareword{Literal: types.NewLiteral(types.Bareword, pos), value:value}
}

func NewValue(pos token.Pos, tok token.Token, value string) (v types.Value) {
        switch tok {
        case token.INT:      v = NewInt(pos, value)
        case token.FLOAT:    v = NewFloat(pos, value)
        case token.DATETIME: v = NewDateTime(pos, value)
        case token.DATE:     v = NewDate(pos, value)
        case token.TIME:     v = NewTime(pos, value)
        case token.URI:      v = NewUri(pos, value)
        case token.STRING:   v = NewString(pos, value)
        case token.BAREWORD: v = NewBareword(pos, value)
        }
        return
}

func (v *Int) String() string      { return strconv.FormatInt(int64(v.value),10) } // Itoa
func (v *Float) String() string    { return strconv.FormatFloat(float64(v.value),'g', -1, 64) }
func (v *DateTime) String() string { return time.Time(v.value).Format("2006-01-02T15:04:05.999999999Z07:00") } // time.RFC3339Nano
func (v *Date) String() string     { return time.Time(v.value).Format("2006-01-02") }
func (v *Time) String() string     { return time.Time(v.value).Format("15:04:05.999999999Z07:00") }
func (v *Uri) String() string      { return "Uri" }
func (v *String) String() string   { return v.value }
func (v *Bareword) String() string { return v.value }
