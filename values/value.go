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
        "os"
)

// Scalar literal/value types. A literal is literrally written in the source code,
// a value is calculated from the program.
type (
        none struct {}

        AnyValue struct {
                none // same as none but holding a interface{}
                Any interface{}
        }
        
        IntValue struct {
                v int64
        }
        
        FloatValue struct {
                v float64
        }

        datetimeValue struct {
                v time.Time
        }
        DateTimeValue struct { datetimeValue }
        DateValue     struct { datetimeValue }
        TimeValue     struct { datetimeValue }
        
        UriValue struct {
                v *url.URL
        }

        StringValue struct {
                v string
        }

        BarewordValue struct {
                v string
        }

        IdentValue struct {
                Names []string
        }
        
        elements struct {
                elems []types.Value
        }

        BarecompValue struct {
                elements
        }

        BarefileValue struct {
                Name types.Value
                Ext string
        }
        
        PathValue struct {
                Segments []types.Value
        }

        FlagValue struct {
                Name types.Value
        }
        
        CompoundValue struct {
                elements
        }

        ListValue struct {
                elements
        }

        GroupValue struct {
                ListValue
        }

        MapValue struct {
                //elems map[types.Value]types.Value
                elems map[string]types.Value
        }

        PairValue struct { // key=value
                k types.Value
                v types.Value
        }
)

var None = &none{}

func Any(any interface{}) *AnyValue {
        return &AnyValue{ Any:any }
}

func IntLit(s string) (v *IntValue) {
        if i, e := strconv.ParseInt(s, 10, 64); e == nil {
                v = &IntValue{i}
        }
        return
}
func Int(s int64) (v *IntValue) {
        return &IntValue{s}
}

func FloatLit(s string) (v *FloatValue) {
        if f, e := strconv.ParseFloat(s, 64); e == nil {
                v = &FloatValue{f}
        }
        return
}
func Float(s float64) (v *FloatValue) {
        return &FloatValue{s}
}

func DateTimeLit(s string) (v *DateTimeValue) {
        // time.RFC3339Nano
        if t, e := time.Parse("2006-01-02T15:04:05.999999999Z07:00", s); e == nil {
                v = &DateTimeValue{datetimeValue{t}}
        }
        return
}
func DateTime(s time.Time) (v *DateTimeValue) {
        return &DateTimeValue{datetimeValue{s}}
}

func DateLit(s string) (v *DateValue) {
        if t, e := time.Parse("2006-01-02", s); e == nil {
                v = &DateValue{datetimeValue{t}}
        }
        return
}
func Date(s time.Time) (v *DateValue) {
        return &DateValue{datetimeValue{s}}
}

func TimeLit(s string) (v *TimeValue) {
        if t, e := time.Parse("15:04:05.999999999Z07:00", s); e == nil {
                v = &TimeValue{datetimeValue{t}}
        }
        return
}
func Time(t time.Time) (v *TimeValue) {
        return &TimeValue{datetimeValue{t}}
}

func UriLit(s string) (v *UriValue) {
        if u, e := url.Parse(s); e == nil {
                v = &UriValue{u}
        }
        return
}
func Uri(s *url.URL) (v *UriValue) {
        return &UriValue{s}
}

func StringLit(s string) (v *StringValue) {
        return &StringValue{s}
}
func String(s string) (v *StringValue) {
        return &StringValue{s}
}

func Ident(names... string) (v *IdentValue) {
        return &IdentValue{names}
}

func Bareword(s string) (v *BarewordValue) {
        return &BarewordValue{s}
}

func Barecomp(elems... types.Value) (v *BarecompValue) {
        return &BarecompValue{elements{elems}}
}

func Barefile(name types.Value, ext string) (v *BarefileValue) {
        return &BarefileValue{name, ext}
}

func Path(segments... types.Value) (v *PathValue) {
        return &PathValue{segments}
}

func Flag(name types.Value) (v *FlagValue) {
        return &FlagValue{name}
}

func Compound(elems... types.Value) (v *CompoundValue) {
        return &CompoundValue{elements{elems}}
}

func List(elems... types.Value) (v *ListValue) {
        return &ListValue{elements{elems}}
}

func Group(elems... types.Value) (v *GroupValue) {
        return &GroupValue{ListValue{elements{elems}}}
}

func Pair(k, v types.Value) (p *PairValue) {
        if k.Type().Info()&types.IsKeyName != 0 {
                p = &PairValue{nil, nil}
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

func (p *none) Type() types.Type        { return types.None }
func (p *none) Pos() *token.Position    { return nil }
func (p *none) Lit() string             { return "" }
func (p *none) String() string          { return "" }
func (p *none) Integer() int64          { return 0 }
func (p *none) Float() float64          { return 0 }

func (p *IntValue) Lit() string         { return p.String() }
func (p *FloatValue) Lit() string       { return p.String() }
func (p *DateTimeValue) Lit() string    { return p.String() }
func (p *DateValue) Lit() string        { return p.String() }
func (p *TimeValue) Lit() string        { return p.String() }
func (p *UriValue) Lit() string         { return p.String() }
func (p *StringValue) Lit() string {
        if strings.ContainsRune(p.v, '\n') {
                return "\"" + strings.Replace(p.v, "\n", "\\n", -1) + "\"" 
        } else {
                return "'" + p.v + "'" 
        }
}
func (p *BarewordValue) Lit() string     { return p.String() }
func (p *BarecompValue) Lit() (s string) {
        for _, e := range p.elems {
                s += e.Lit()
        }
        return
}
func (p *BarefileValue) Lit() (s string) {
        s += p.Name.Lit()
        if p.Ext != "" {
                s += "." + p.Ext
        }
        return
}
func (p *PathValue) Lit() (s string) {
        // TODO: add '/' for root dir
        for i, seg := range p.Segments {
                if i > 0 { s += string(os.PathSeparator) }
                s += seg.Lit()
        }
        // TODO: add '/' if there's such a suffix
        return
}
func (p *FlagValue) Lit() (s string) {
        s = "-" + p.Name.Lit()
        return
}
func (p *CompoundValue) Lit() (s string) {
        s = "\""
        for _, e := range p.elems {
                s += e.Lit()
        }
        s += "\""
        return
}
func (p *ListValue) Lit() (s string) {
        for i, e := range p.elems {
                if 0 < i {
                        s += " "
                }
                s += e.Lit()
        }
        return
}
func (p *GroupValue) Lit() string {
        return "(" + p.ListValue.Lit() + ")"
}
/* func (p *MapValue) Lit() string {
        return "(" + p.ListValue.Lit() + ")"
} */
func (p *PairValue) Lit() string {
        return p.k.Lit() + "=" + p.v.Lit()
}

func (p *IntValue) String() string      { return strconv.FormatInt(int64(p.v),10) } // Itoa
func (p *FloatValue) String() string    { return strconv.FormatFloat(float64(p.v),'g', -1, 64) }
func (p *DateTimeValue) String() string { return time.Time(p.v).Format("2006-01-02T15:04:05.999999999Z07:00") } // time.RFC3339Nano
func (p *DateValue) String() string     { return time.Time(p.v).Format("2006-01-02") }
func (p *TimeValue) String() string     { return time.Time(p.v).Format("15:04:05.999999999Z07:00") }
func (p *UriValue) String() string      { return p.v.String() }
func (p *StringValue) String() string   { return p.v }
func (p *BarewordValue) String() string { return p.v }
func (p *BarefileValue) String() string {
        s := p.Name.String()
        if p.Ext != "" {
                s += "." + p.Ext
        }
        return s
}
func (p *PathValue) String() (s string) {
        // TODO: add '/' for root dir
        for i, seg := range p.Segments {
                if i > 0 { s += string(os.PathSeparator) }
                s += seg.String()
        }
        // TODO: add '/' if there's such a suffix
        return
}
func (p *FlagValue) String() string {
        return "-" + p.Name.String()
}
func (p *BarecompValue) String() (s string) {
        for _, e := range p.elems {
                s += e.String()
        }
        return
}
func (p *CompoundValue) String() (s string) {
        //s = "\""
        for _, e := range p.elems {
                s += e.String()
        }
        //s += "\""
        return
}
func (p *ListValue) String() (s string) {
        var x = 0
        for _, e := range p.elems {
                if v := e.String(); v != "" {
                        if 0 < x {
                                s += " "
                        }
                        s += v
                        x += 1
                }
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

func (p *IntValue) Integer() int64      { return p.v }
func (p *FloatValue) Integer() int64    { return int64(p.v) }
func (p *DateTimeValue) Integer() int64 { return p.v.Unix() }
func (p *DateValue) Integer() int64     { return p.v.Unix() }
func (p *TimeValue) Integer() int64     { return p.v.Unix() }
func (p *UriValue) Integer() int64      { return int64(len(p.v.String())) }
func (p *StringValue) Integer() int64   { i, _ := strconv.ParseInt(p.v, 10, 64); return i }
func (p *BarecompValue) Integer() int64 { return int64(len(p.elems)) }
func (p *BarewordValue) Integer() int64 { return 0 }
func (p *BarefileValue) Integer() int64 { return 0 }
func (p *PathValue) Integer() int64     { return 0 }
func (p *FlagValue) Integer() int64     { return 0 }
func (p *CompoundValue) Integer() int64 { return int64(len(p.elems)) }
func (p *ListValue) Integer() int64     { return int64(len(p.elems)) }
func (p *GroupValue) Integer() int64    { return int64(len(p.ListValue.elems)) }
func (p *PairValue) Integer() int64     { return p.v.Integer() }


func (p *IntValue) Float() float64      { return float64(p.v) }
func (p *FloatValue) Float() float64    { return p.v }
func (p *DateTimeValue) Float() float64 { return float64(p.Integer()) }
func (p *DateValue) Float() float64     { return float64(p.Integer()) }
func (p *TimeValue) Float() float64     { return float64(p.Integer()) }
func (p *UriValue) Float() float64      { return float64(p.Integer()) }
func (p *StringValue) Float() float64   { return float64(p.Integer()) }
func (p *BarewordValue) Float() float64 { return float64(p.Integer()) }
func (p *BarecompValue) Float() float64 { return float64(p.Integer()) }
func (p *BarefileValue) Float() float64 { return float64(p.Integer()) }
func (p *PathValue) Float() float64     { return float64(p.Integer()) }
func (p *FlagValue) Float() float64     { return float64(p.Integer()) }
func (p *CompoundValue) Float() float64 { return float64(p.Integer()) }
func (p *ListValue) Float() float64     { return float64(p.Integer()) }
func (p *GroupValue) Float() float64    { return float64(p.Integer()) }
func (p *PairValue) Float() float64     { return p.v.Float() }

func (p *IntValue) Type() types.Type      { return types.Int }
func (p *FloatValue) Type() types.Type    { return types.Float }
func (p *DateTimeValue) Type() types.Type { return types.DateTime }
func (p *DateValue) Type() types.Type     { return types.Date }
func (p *TimeValue) Type() types.Type     { return types.Time }
func (p *UriValue) Type() types.Type      { return types.Uri }
func (p *StringValue) Type() types.Type   { return types.String }
func (p *IdentValue) Type() types.Type    { return types.Ident }
func (p *BarewordValue) Type() types.Type { return types.Bareword }
func (p *BarecompValue) Type() types.Type { return types.Barecomp }
func (p *BarefileValue) Type() types.Type { return types.Barefile }
func (p *PathValue) Type() types.Type     { return types.Path }
func (p *FlagValue) Type() types.Type     { return types.Flag }
func (p *CompoundValue) Type() types.Type { return types.Compound }
func (p *GroupValue) Type() types.Type    { return types.Group }
func (p *ListValue) Type() types.Type     { return types.List }
func (p *PairValue) Type() types.Type     { return types.Pair }

func (p *IntValue) Pos() *token.Position      { return nil }
func (p *FloatValue) Pos() *token.Position    { return nil }
func (p *DateTimeValue) Pos() *token.Position { return nil }
func (p *DateValue) Pos() *token.Position     { return nil }
func (p *TimeValue) Pos() *token.Position     { return nil }
func (p *UriValue) Pos() *token.Position      { return nil }
func (p *StringValue) Pos() *token.Position   { return nil }
func (p *IdentValue) Pos() *token.Position    { return nil }
func (p *BarewordValue) Pos() *token.Position { return nil }
func (p *BarecompValue) Pos() *token.Position { return nil }
func (p *BarefileValue) Pos() *token.Position { return nil }
func (p *PathValue) Pos() *token.Position     { return nil }
func (p *FlagValue) Pos() *token.Position     { return nil }
func (p *CompoundValue) Pos() *token.Position { return nil }
func (p *GroupValue) Pos() *token.Position    { return nil }
func (p *ListValue) Pos() *token.Position     { return nil }
func (p *PairValue) Pos() *token.Position     { return nil }

func (p *elements) Len() int                      { return len(p.elems) }
func (p *elements) Append(v... types.Value)       { p.elems = append(p.elems, v...) }
func (p *elements) Get(n int) (v types.Value)     { if n>=0 && n<len(p.elems) { v = p.elems[n] }; return }
func (p *elements) Slice(n int) (a []types.Value) {
        if n>=0 && n<len(p.elems) {
                a = p.elems[n:]
        }
        return 
}
func (p *elements) Take(n int) (v types.Value) {
        if x := len(p.elems); n>=0 && n<x {
                v = p.elems[n]
                p.elems = append(p.elems[0:n], p.elems[n+1:]...)
        }
        return 
}
func (p *elements) ToBarecomp() *BarecompValue { return &BarecompValue{*p} }
func (p *elements) ToCompound() *CompoundValue { return &CompoundValue{*p} }
func (p *elements) ToList() *ListValue         { return &ListValue{*p} }

func (p *PairValue) Key() types.Value { return p.k }
func (p *PairValue) Value() types.Value { return p.v }
func (p *PairValue) SetValue(v types.Value) { p.v = v }
func (p *PairValue) SetKey(k types.Value) {
        switch o := k.(type) {
        case *PairValue:   k = o.k
        //case *PairLiteral: k = o.k
        }
        if k.Type().Info()&types.IsKeyName != 0 {
                p.k = k
        } else {
                p.k = nil
        }
}
