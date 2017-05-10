//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import (
        "github.com/duzy/smart/token"
        "time"
        "net/url"
        "strconv"
        "strings"
        "os"
)

// Value represents a value of a type.
type Value interface {
        // Type returns the underlying type of the value.
        Type() Type

        // Lit returns the literal representations of the value.
        Lit() string

        // String returns the string form of the value.
        String() string

        // Integer returns the integer form of the value.
        Integer() int64

        // Float returns the float form of the value.
        Float() float64
}

type value struct {}
func (*value) Type() Type         { return Invalid }
func (*value) Lit() string        { return "" }
func (*value) String() string     { return "" }
func (*value) Integer() int64     { return 0 }
func (*value) Float() float64     { return 0 }

type NoneValue struct { value }
func (p *NoneValue) Type() Type   { return None }

type AnyValue struct {
        value
        V interface{}
}
func (p *AnyValue) Type() Type    { return Any }

type IntValue struct {
        V int64
}
func (p *IntValue) Type() Type          { return Int }
func (p *IntValue) Lit() string         { return p.String() }
func (p *IntValue) String() string      { return strconv.FormatInt(int64(p.V),10) }
func (p *IntValue) Integer() int64      { return p.V }
func (p *IntValue) Float() float64      { return float64(p.V) }

type FloatValue struct {
        V float64
}
func (p *FloatValue) Type() Type        { return Float }
func (p *FloatValue) Lit() string       { return p.String() }
func (p *FloatValue) String() string    { return strconv.FormatFloat(float64(p.V),'g', -1, 64) }
func (p *FloatValue) Integer() int64    { return int64(p.V) }
func (p *FloatValue) Float() float64    { return p.V }

type DateTimeValue struct {
        V time.Time 
}
func (p *DateTimeValue) Type() Type     { return DateTime }
func (p *DateTimeValue) Lit() string    { return p.String() }
func (p *DateTimeValue) String() string { return time.Time(p.V).Format("2006-01-02T15:04:05.999999999Z07:00") } // time.RFC3339Nano
func (p *DateTimeValue) Integer() int64 { return p.V.Unix() }
func (p *DateTimeValue) Float() float64 { return float64(p.Integer()) }

type DateValue struct { DateTimeValue }
func (p *DateValue) Type() Type         { return Date }
func (p *DateValue) Lit() string        { return p.String() }
func (p *DateValue) String() string     { return time.Time(p.V).Format("2006-01-02") }
func (p *DateValue) Integer() int64     { return p.V.Unix() }
func (p *DateValue) Float() float64     { return float64(p.Integer()) }

type TimeValue struct { DateTimeValue }
func (p *TimeValue) Type() Type         { return Time }
func (p *TimeValue) Lit() string        { return p.String() }
func (p *TimeValue) String() string     { return time.Time(p.V).Format("15:04:05.999999999Z07:00") }
func (p *TimeValue) Integer() int64     { return p.V.Unix() }
func (p *TimeValue) Float() float64     { return float64(p.Integer()) }

type UriValue struct {
        V *url.URL
}
func (p *UriValue) Type() Type          { return Uri }
func (p *UriValue) Lit() string         { return p.String() }
func (p *UriValue) String() string      { return p.V.String() }
func (p *UriValue) Integer() int64      { return int64(len(p.V.String())) }
func (p *UriValue) Float() float64      { return float64(p.Integer()) }

type StringValue struct {
        V string
}
func (p *StringValue) Type() Type  { return String }
func (p *StringValue) Lit() string {
        if strings.ContainsRune(p.V, '\n') {
                return "\"" + strings.Replace(p.V, "\n", "\\n", -1) + "\"" 
        } else {
                return "'" + p.V + "'" 
        }
}
func (p *StringValue) String() string   { return p.V }
func (p *StringValue) Integer() int64   { i, _ := strconv.ParseInt(p.V, 10, 64); return i }
func (p *StringValue) Float() float64   { return float64(p.Integer()) }

type BarewordValue struct {
        V string
}
func (p *BarewordValue) Type() Type     { return Bareword }
func (p *BarewordValue) Lit() string    { return p.String() }
func (p *BarewordValue) String() string { return p.V }
func (p *BarewordValue) Integer() int64 { return 0 }
func (p *BarewordValue) Float() float64 { return float64(p.Integer()) }


        
type Elements struct {
        Elems []Value
}
func (p *Elements) Len() int                    { return len(p.Elems) }
func (p *Elements) Append(v... Value)           { p.Elems = append(p.Elems, v...) }
func (p *Elements) Get(n int) (v Value)         { if n>=0 && n<len(p.Elems) { v = p.Elems[n] }; return }
func (p *Elements) Slice(n int) (a []Value)     {
        if n>=0 && n<len(p.Elems) {
                a = p.Elems[n:]
        }
        return 
}
func (p *Elements) Take(n int) (v Value) {
        if x := len(p.Elems); n>=0 && n<x {
                v = p.Elems[n]
                p.Elems = append(p.Elems[0:n], p.Elems[n+1:]...)
        }
        return 
}
func (p *Elements) ToBarecomp() *BarecompValue { return &BarecompValue{*p} }
func (p *Elements) ToCompound() *CompoundValue { return &CompoundValue{*p} }
func (p *Elements) ToList() *ListValue         { return &ListValue{*p} }

type BarecompValue struct {
        Elements
}
func (p *BarecompValue) Lit() (s string) {
        for _, e := range p.Elems {
                s += e.Lit()
        }
        return
}
func (p *BarecompValue) String() (s string) {
        for _, e := range p.Elems {
                s += e.String()
        }
        return
}
func (p *BarecompValue) Type() Type     { return Barecomp }
func (p *BarecompValue) Integer() int64 { return int64(len(p.Elems)) }
func (p *BarecompValue) Float() float64 { return float64(p.Integer()) }

type BarefileValue struct {
        Name Value
        Ext string
}
func (p *BarefileValue) Type() Type { return Barefile }
func (p *BarefileValue) Lit() (s string) {
        s += p.Name.Lit()
        if p.Ext != "" {
                s += "." + p.Ext
        }
        return
}
func (p *BarefileValue) String() string {
        s := p.Name.String()
        if p.Ext != "" {
                s += "." + p.Ext
        }
        return s
}
func (p *BarefileValue) Integer() int64 { return 0 }
func (p *BarefileValue) Float() float64 { return float64(p.Integer()) }

type PathValue struct {
        Segments []Value
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
func (p *PathValue) String() (s string) {
        // TODO: add '/' for root dir
        for i, seg := range p.Segments {
                if i > 0 { s += string(os.PathSeparator) }
                s += seg.String()
        }
        // TODO: add '/' if there's such a suffix
        return
}
func (p *PathValue) Integer() int64     { return 0 }
func (p *PathValue) Float() float64     { return float64(p.Integer()) }
func (p *PathValue) Type() Type         { return Path }

type FlagValue struct {
        Name Value
}
func (p *FlagValue) Lit() (s string) {
        s = "-" + p.Name.Lit()
        return
}
func (p *FlagValue) String() string {
        return "-" + p.Name.String()
}
func (p *FlagValue) Integer() int64     { return 0 }
func (p *FlagValue) Float() float64     { return float64(p.Integer()) }
func (p *FlagValue) Type() Type         { return Flag }
        
type CompoundValue struct {
        Elements
}
func (p *CompoundValue) Lit() (s string) {
        s = "\""
        for _, e := range p.Elems {
                s += e.Lit()
        }
        s += "\""
        return
}
func (p *CompoundValue) String() (s string) {
        //s = "\""
        for _, e := range p.Elems {
                s += e.String()
        }
        //s += "\""
        return
}
func (p *CompoundValue) Integer() int64 { return int64(len(p.Elems)) }
func (p *CompoundValue) Float() float64 { return float64(p.Integer()) }
func (p *CompoundValue) Type() Type     { return Compound }

type ListValue struct {
        Elements
}
func (p *ListValue) Lit() (s string) {
        for i, e := range p.Elems {
                if 0 < i {
                        s += " "
                }
                s += e.Lit()
        }
        return
}
func (p *ListValue) String() (s string) {
        var x = 0
        for _, e := range p.Elems {
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
func (p *ListValue) Integer() int64     { return int64(len(p.Elems)) }
func (p *ListValue) Float() float64     { return float64(p.Integer()) }
func (p *ListValue) Type() Type         { return List }

type GroupValue struct {
        ListValue
}
func (p *GroupValue) Lit() string {
        return "(" + p.ListValue.Lit() + ")"
}
func (p *GroupValue) String() string {
        return "(" + p.ListValue.String() + ")"
}
func (p *GroupValue) Integer() int64    { return int64(len(p.ListValue.Elems)) }
func (p *GroupValue) Float() float64    { return float64(p.Integer()) }
func (p *GroupValue) Type() Type        { return Group }

type MapValue struct {
        Elems map[string]Value
}
/* func (p *MapValue) Lit() string {
        return "(" + p.ListValue.Lit() + ")"
}
func (p *Map) String() string {
        return "(" + p.List.String() + ")"
} */

type PairValue struct { // key=value
        K Value
        V Value
}
func (p *PairValue) Lit() string {
        return p.K.Lit() + "=" + p.V.Lit()
}
func (p *PairValue) String() string {
        return p.K.String() + "=" + p.V.String()
}
func (p *PairValue) Integer() int64     { return p.V.Integer() }
func (p *PairValue) Float() float64     { return p.V.Float() }
func (p *PairValue) Type() Type         { return Pair }

func (p *PairValue) Key() Value         { return p.K }
func (p *PairValue) Value() Value       { return p.V }
func (p *PairValue) SetValue(v Value)   { p.V = v }
func (p *PairValue) SetKey(k Value) {
        switch o := k.(type) {
        case *PairValue:   k = o.K
        //case *PairLiteral: k = o.K
        }
        if k.Type().Info()&IsKeyName != 0 {
                p.K = k
        } else {
                p.K = nil
        }
}

type Definer interface {
        Define(p *Project) (Value, error)
}

type DefinerValue interface {
        Value
        Definer
}

type Caller interface {
        Call(args... Value) (Value, error)
}

type CallerValue interface {
        Value
        Caller
}

type Unrefer interface {
        Unref(project *Project, s string, a... Value) (Value, error)
}

type UnreferValue interface {
        Value
        Unrefer
}

type Poser interface {
        Pos() *token.Position
}

type PoserValue interface {
        Value
        Poser
}

type Namer interface {
        Name() string
}

type Scoper interface {
        Scope() *Scope
}

type NameScoper interface {
        Namer
        Scoper
}

type positional struct {
        Value
        pos *token.Position
}

// Pos() returns the position of the value occurs position in file or nil.
func (p *positional) Pos() *token.Position { return p.pos }

// Positional wraps a value with a valid position
func Positional(v Value, pos *token.Position) Poser {
        if p, ok := v.(*positional); ok {
                p.pos = pos
                return p
        }
        return &positional{ v, pos }
}

type namescoper struct {
        name string
        scope *Scope
}

func (ns *namescoper) Name() string { return ns.name }
func (ns *namescoper) Scope() *Scope { return ns.scope }
func NameScope(name string, scope *Scope) NameScoper {
        return &namescoper{ name, scope }
}
