//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import (
        "github.com/duzy/smart/token"
        "path/filepath"
        "time"
        "net/url"
        "strconv"
        "strings"
        "errors"
        "fmt"
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

        // disclosure method, also prevents creating new Value type from
        // other packages.
        disclosure(scope *Scope, args []Value) (Value, error)
}

type value struct {}
func (*value) disclosure(_ *Scope, _ []Value) (Value, error) { return nil, nil }
func (*value) Type() Type         { return InvalidType }
func (*value) Lit() string        { return "" }
func (*value) String() string     { return "" }
func (*value) Integer() int64     { return 0 }
func (*value) Float() float64     { return 0 }

type None struct { value }
func (p *None) Type() Type { return NoneType }

type Any struct {
        V interface{}
        value
}
func (p *Any) Type() Type    { return AnyType }

type Int struct {
        V int64
}
func (*Int) disclosure(_ *Scope, _ []Value) (Value, error) { return nil, nil }
func (p *Int) Type() Type          { return IntType }
func (p *Int) Lit() string         { return p.String() }
func (p *Int) String() string      { return strconv.FormatInt(int64(p.V),10) }
func (p *Int) Integer() int64      { return p.V }
func (p *Int) Float() float64      { return float64(p.V) }

type Float struct {
        V float64
}
func (*Float) disclosure(_ *Scope, _ []Value) (Value, error) { return nil, nil }
func (p *Float) Type() Type        { return FloatType }
func (p *Float) Lit() string       { return p.String() }
func (p *Float) String() string    { return strconv.FormatFloat(float64(p.V),'g', -1, 64) }
func (p *Float) Integer() int64    { return int64(p.V) }
func (p *Float) Float() float64    { return p.V }

type DateTime struct {
        V time.Time 
}
func (*DateTime) disclosure(_ *Scope, _ []Value) (Value, error) { return nil, nil }
func (p *DateTime) Type() Type     { return DateTimeType }
func (p *DateTime) Lit() string    { return p.String() }
func (p *DateTime) String() string { return time.Time(p.V).Format("2006-01-02T15:04:05.999999999Z07:00") } // time.RFC3339Nano
func (p *DateTime) Integer() int64 { return p.V.Unix() }
func (p *DateTime) Float() float64 { return float64(p.Integer()) }

type Date struct { DateTime }
func (*Date) disclosure(_ *Scope, _ []Value) (Value, error) { return nil, nil }
func (p *Date) Type() Type         { return DateType }
func (p *Date) Lit() string        { return p.String() }
func (p *Date) String() string     { return time.Time(p.V).Format("2006-01-02") }
func (p *Date) Integer() int64     { return p.V.Unix() }
func (p *Date) Float() float64     { return float64(p.Integer()) }

type Time struct { DateTime }
func (*Time) disclosure(_ *Scope, _ []Value) (Value, error) { return nil, nil }
func (p *Time) Type() Type         { return TimeType }
func (p *Time) Lit() string        { return p.String() }
func (p *Time) String() string     { return time.Time(p.V).Format("15:04:05.999999999Z07:00") }
func (p *Time) Integer() int64     { return p.V.Unix() }
func (p *Time) Float() float64     { return float64(p.Integer()) }

type Uri struct {
        V *url.URL
}
func (*Uri) disclosure(_ *Scope, _ []Value) (Value, error) { return nil, nil }
func (p *Uri) Type() Type          { return UriType }
func (p *Uri) Lit() string         { return p.String() }
func (p *Uri) String() string      { return p.V.String() }
func (p *Uri) Integer() int64      { return int64(len(p.V.String())) }
func (p *Uri) Float() float64      { return float64(p.Integer()) }

type String struct {
        V string
}
func (p *String) Type() Type  { return StringType }
func (p *String) Lit() string {
        if strings.ContainsRune(p.V, '\n') {
                return "\"" + strings.Replace(p.V, "\n", "\\n", -1) + "\"" 
        } else {
                return "'" + p.V + "'" 
        }
}
func (*String) disclosure(_ *Scope, _ []Value) (Value, error) { return nil, nil }
func (p *String) String() string   { return p.V }
func (p *String) Integer() int64   { i, _ := strconv.ParseInt(p.V, 10, 64); return i }
func (p *String) Float() float64   { return float64(p.Integer()) }

type Bareword struct {
        V string
}
func (*Bareword) disclosure(_ *Scope, _ []Value) (Value, error) { return nil, nil }
func (p *Bareword) Type() Type     { return BarewordType }
func (p *Bareword) Lit() string    { return p.String() }
func (p *Bareword) String() string { return p.V }
func (p *Bareword) Integer() int64 { return 0 }
func (p *Bareword) Float() float64 { return float64(p.Integer()) }
        
type Elements struct {
        Elems []Value
}
func (p *Elements) Float() float64 { return float64(p.Integer()) }
func (p *Elements) Integer() int64 {
        var n = len(p.Elems)
        if n == 1 {
                // If there's only one element, treat it as a scalar.
                return p.Elems[0].Integer()
        }
        return int64(n)
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
func (p *Elements) ToBarecomp() *Barecomp { return &Barecomp{*p} }
func (p *Elements) ToCompound() *Compound { return &Compound{*p} }
func (p *Elements) ToList() *List         { return &List{*p} }

func (p *Elements) disclosureElems(scope *Scope, args []Value) ([]Value, int, error) {
        var elems []Value
        var num = 0 
        for _, elem := range p.Elems {
                //fmt.Printf("disclosureElems: %T %v\n", elem, elem)
                if v, e := elem.disclosure(scope, args); e != nil {
                        return nil, 0, e
                } else if v != nil {
                        elem = v
                        num += 1
                }
                elems = append(elems, elem)
        }
        return elems, num, nil
}

type Barecomp struct {
        Elements
}
func (p *Barecomp) Lit() (s string) {
        for _, e := range p.Elems {
                s += e.Lit()
        }
        return
}
func (p *Barecomp) String() (s string) {
        for _, e := range p.Elems {
                s += e.String()
        }
        return
}
func (p *Barecomp) Type() Type     { return BarecompType }
func (p *Barecomp) Integer() int64 { return int64(len(p.Elems)) }
func (p *Barecomp) Float() float64 { return float64(p.Integer()) }

func (p *Barecomp) disclosure(scope *Scope, args []Value) (Value, error) {
        if elems, num, err := p.disclosureElems(scope, args); err != nil {
                return nil, err
        } else if num > 0 {
                return &Barecomp{ Elements{ elems } }, nil
        }
        return nil, nil
}

type Barefile struct {
        Name Value
        Ext string
}
func (p *Barefile) Type() Type { return BarefileType }
func (p *Barefile) Lit() (s string) {
        s += p.Name.Lit()
        if p.Ext != "" {
                s += "." + p.Ext
        }
        return
}
func (p *Barefile) String() string {
        s := p.Name.String()
        if p.Ext != "" {
                s += "." + p.Ext
        }
        return s
}
func (p *Barefile) Integer() int64 { return 0 }
func (p *Barefile) Float() float64 { return float64(p.Integer()) }

func (p *Barefile) disclosure(scope *Scope, args []Value) (Value, error) {
        if name, err := p.Name.disclosure(scope, args); err != nil {
                return nil, err
        } else if name != nil {
                return &Barefile{ name, p.Ext }, nil
        }
        return nil, nil
}

type Path struct {
        Elements
}
func (p *Path) Lit() (s string) {
        // TODO: add '/' for root dir
        for i, seg := range p.Elems {
                if i > 0 { s += string(os.PathSeparator) }
                s += seg.Lit()
        }
        // TODO: add '/' if there's such a suffix
        return
}
func (p *Path) String() (s string) {
        // TODO: add '/' for root dir
        for i, seg := range p.Elems {
                if i > 0 { s += string(os.PathSeparator) }
                s += seg.String()
        }
        // TODO: add '/' if there's such a suffix
        return
}
func (p *Path) Integer() int64     { return 0 }
func (p *Path) Float() float64     { return float64(p.Integer()) }
func (p *Path) Type() Type         { return PathType }

func (p *Path) disclosure(scope *Scope, args []Value) (Value, error) {
        if elems, num, err := p.disclosureElems(scope, args); err != nil {
                return nil, err
        } else if num > 0 {
                return &Path{ Elements{ elems } }, nil
        }
        return nil, nil
}

type File struct {
        Value  // original represented name (e.g. Barefile)
        Name string  // represented name (e.g. relative filename)
        Dir string   // directory in which the file should be or was found
        Info os.FileInfo // file info if exists
}
func (p *File) Type() Type { return FileType }
func (p *File) String() string { return filepath.Join(p.Dir, p.Name) }

func (p *File) disclosure(scope *Scope, args []Value) (Value, error) {
        if v, err := p.Value.disclosure(scope, args); err != nil {
                return nil, err
        } else if v != nil {
                return &File{ v, p.Name, p.Dir, p.Info }, nil
        }
        return nil, nil
}

type Flag struct {
        Name Value
}
func (p *Flag) Lit() (s string) {
        s = "-" + p.Name.Lit()
        return
}
func (p *Flag) String() string {
        return "-" + p.Name.String()
}
func (p *Flag) Integer() int64     { return 0 }
func (p *Flag) Float() float64     { return float64(p.Integer()) }
func (p *Flag) Type() Type         { return FlagType }

func (p *Flag) disclosure(scope *Scope, args []Value) (Value, error) {
        if name, err := p.Name.disclosure(scope, args); err != nil {
                return nil, err
        } else if name != nil {
                return &Flag{ name }, nil
        }
        return nil, nil
}
        
type Compound struct {
        Elements
}
func (p *Compound) Lit() (s string) {
        s = "\""
        for _, e := range p.Elems {
                s += e.Lit()
        }
        s += "\""
        return
}
func (p *Compound) String() (s string) {
        //s = "\""
        for _, e := range p.Elems {
                s += e.String()
        }
        //s += "\""
        return
}
func (p *Compound) Integer() int64 { return int64(len(p.Elems)) }
func (p *Compound) Float() float64 { return float64(p.Integer()) }
func (p *Compound) Type() Type     { return CompoundType }

func (p *Compound) disclosure(scope *Scope, args []Value) (Value, error) {
        if elems, num, err := p.disclosureElems(scope, args); err != nil {
                return nil, err
        } else if num > 0 {
                return &Compound{ Elements{ elems } }, nil
        }
        return nil, nil
}

type List struct {
        Elements
}
func (p *List) Lit() (s string) {
        for i, e := range p.Elems {
                if 0 < i {
                        s += " "
                }
                s += e.Lit()
        }
        return
}
func (p *List) String() (s string) {
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
func (p *List) Type() Type         { return ListType }

func (p *List) disclosure(scope *Scope, args []Value) (Value, error) {
        if elems, num, err := p.disclosureElems(scope, args); err != nil {
                return nil, err
        } else if num > 0 {
                return &List{ Elements{ elems } }, nil
        }
        return nil, nil
}

type Group struct {
        List
}
func (p *Group) Lit() string {
        return "(" + p.List.Lit() + ")"
}
func (p *Group) String() string {
        return "(" + p.List.String() + ")"
}
func (p *Group) Type() Type        { return GroupType }

func (p *Group) disclosure(scope *Scope, args []Value) (Value, error) {
        if elems, num, err := p.disclosureElems(scope, args); err != nil {
                return nil, err
        } else if num > 0 {
                return &Group{ List{ Elements{ elems } } }, nil
        }
        return nil, nil
}

//type Map struct {
//        Elems map[string]Value
//}
/* func (p *Map) Lit() string {
        return "(" + p.List.Lit() + ")"
}
func (p *Map) String() string {
        return "(" + p.List.String() + ")"
} */

type Pair struct { // key=value
        K Value
        V Value
}
func (p *Pair) Lit() string {
        return p.K.Lit() + "=" + p.V.Lit()
}
func (p *Pair) String() string {
        return p.K.String() + "=" + p.V.String()
}
func (p *Pair) Integer() int64     { return p.V.Integer() }
func (p *Pair) Float() float64     { return p.V.Float() }
func (p *Pair) Type() Type         { return PairType }

func (p *Pair) Key() Value         { return p.K }
func (p *Pair) Value() Value       { return p.V }
func (p *Pair) SetValue(v Value)   { p.V = v }
func (p *Pair) SetKey(k Value) {
        switch o := k.(type) {
        case *Pair:   k = o.K
        //case *PairLiteral: k = o.K
        }
        if k.Type().Bits()&IsKeyName != 0 {
                p.K = k
        } else {
                p.K = nil
        }
}

func (p *Pair) disclosure(scope *Scope, args []Value) (Value, error) {
        if k, err := p.K.disclosure(scope, args); err != nil {
                return nil, err
        } else if v, err := p.V.disclosure(scope, args); err != nil {
                return nil, err
        } else if k != nil || v != nil {
                if k == nil { k = p.K }
                if v == nil { v = p.V }
                return &Pair{ k, v }, nil
        }
        return nil, nil
}

type Closure struct {
        Object
        Name Value
}

func (r *Closure) Type() Type { return ClosureType }
func (r *Closure) Lit() string { return "&" + r.Name.Lit() }
func (r *Closure) String() string { return "&" + r.Name.String() }
func (r *Closure) disclosure(scope *Scope, args []Value) (Value, error) {
        //fmt.Printf("disclosure: Closure: %T %v\n", r.Name, r.Name)
        var (
                name, err = r.Name.disclosure(scope, args)
                obj Object
                a []Value
        )
        if err != nil {
                return nil, err
        } else if name == nil {
                name = r.Name
        }

        switch t := name.(type) {
        case *Bareword, *Barecomp: 
                obj = scope.Find(t.String())
        case *Group:
                if len(t.Elems) > 0 {
                        obj = scope.Find(t.Elems[0].String())
                        a = t.Elems[1:]
                }
        case *List:
                if len(t.Elems) > 0 {
                        obj = scope.Find(t.Elems[0].String())
                        a = t.Elems[1:]
                }
        }
        if obj == nil {
                obj = r.Object
        }

        //fmt.Printf("%v -> %v (%v)\n", name, obj, r.Object)

        if obj == nil {
                s := fmt.Sprintf("nil closure (%T %v)", r.Name, r.Name)
                return nil, errors.New(s)
        } else if c, ok := obj.(Caller); ok && c != nil {
                a = append(a, args...)
                return c.Call(a...)
        } else {
                return obj, nil
        }

        return nil, nil
}

// Pattern
type Pattern interface {
        Value
        Program() Program
        NewEntry(stem string) *RuleEntry
        Match(s string) (matched bool, stem string)
}

type pattern struct {
        parent *Scope
        project *Project
        program Program
}

func (p *pattern) Type() Type        { return InvalidType }
func (p *pattern) Integer() int64    { return 0 }
func (p *pattern) Float() float64    { return 0 }
func (p *pattern) Program() Program  { return p.program }
func (p *pattern) entry(name, stem string) (entry *RuleEntry) {
        var kind = PatternRuleEntry
        if p.project != nil && p.project.IsFile(name) {
                kind = PatternFileRuleEntry
        }
        entry = p.parent.NewRuleEntry(p.project, kind, name)
        entry.parent = p.parent
        entry.project = p.project
        entry.program = p.program
        entry.stem = stem
        return
}

func (*pattern) disclosure(_ *Scope, _ []Value) (Value, error) { return nil, nil }

type PercentPattern struct {
        pattern
        prefix Value
        suffix Value
}

func NewPercentPattern(m *Project, prefix, suffix Value) Pattern {
        return &PercentPattern{pattern:pattern{project:m}, prefix:prefix, suffix:suffix }
}

func (p *PercentPattern) Lit() string { return p.String() }
func (p *PercentPattern) Pos() *token.Position { return nil }
func (p *PercentPattern) String() (s string) {
        if p.prefix != nil {
                s = p.prefix.String()
        }
        s += "%"
        if p.suffix != nil {
                s += p.suffix.String()
        }
        return
}
func (p *PercentPattern) Match(s string) (matched bool, stem string) {
        /*
        if pp, _ := p.prefix.(*PercentPattern); pp != nil {
        }
        if pp, _ := p.suffix.(*PercentPattern); pp != nil {
        } */
        if prefix := p.prefix.String(); prefix == "" || strings.HasPrefix(s, prefix) {
                if suffix := p.suffix.String(); suffix == "" || strings.HasSuffix(s, suffix) {
                        if a, b := len(prefix), len(s)-len(suffix); a < b {
                                matched, stem = true, s[a:b]
                        }
                }
        }
        return
}

func (p *PercentPattern) NewEntry(stem string) (entry *RuleEntry) {
        name := p.prefix.String() + stem + p.suffix.String()
        entry = p.entry(name, stem)
        return
}

type RegexpPattern struct {
        pattern
}

func NewRegexpPattern() Pattern {
        return &RegexpPattern{}
}

func (p *RegexpPattern) Lit() string { return p.String() }
func (p *RegexpPattern) Pos() *token.Position { return nil }
func (p *RegexpPattern) String() (s string) { return "" }
func (p *RegexpPattern) Match(s string) (matched bool, stem string) {
        // TODO: regexp matching...
        return
}
func (p *RegexpPattern) NewEntry(stem string) (entry *RuleEntry) {
        // TODO: creating new match entry
        return
}

type Definer interface {
        Define(p *Project) (Value, error)
}

//type DefinerValue interface {
//        Value
//        Definer
//}

type Valuer interface {
        Value() Value
}

type Caller interface {
        Call(args... Value) (Value, error)
}

//type CallerValue interface {
//        Value
//        Caller
//}

//type Unrefer interface {
//        Unref(project *Project, s string, a... Value) (Value, error)
//}

//type UnreferValue interface {
//        Value
//        Unrefer
//}

type Poser interface {
        Pos() *token.Position
}

//type PoserValue interface {
//        Value
//        Poser
//}

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

func Eval(v Value) (res Value) {
        switch t := v.(type) {
        case Valuer:
                res = Eval(t.Value())
        case *List:
                for i, elem := range t.Elems {
                        t.Elems[i] = Eval(elem)
                }
                res = t
        default:
                res = v
        }
        return
}

func EvalElems(args... Value) (elems []Value) {
        for _, arg := range args {
                switch t := Eval(arg).(type) {
                case *List:
                        for _, elem := range t.Elems {
                                elems = append(elems, EvalElems(elem)...)
                        }
                default:
                        elems = append(elems, t)
                }
        }
        return
}

func disclosure(scope *Scope, value Value, args []Value) (Value, error) {
        //fmt.Printf("disclosure: %T %v\n", value, value)
        if v, err := value.disclosure(scope, args); err != nil {
                return nil, err
        } else if v != nil {
                value = v
        }
        return value, nil
}

func Disclosure(scope *Scope, value Value, args... Value) (Value, error) {
        return disclosure(scope, value, args)
}

type delegate struct {
        o Object // TODO: use Caller instead
        a []Value
}

func (p *delegate) Type() Type          { return p.o.Type() }
func (p *delegate) Lit() string         { return p.o.Lit() }
func (p *delegate) String() string      { return p.o.String() }
func (p *delegate) Integer() int64      { return p.o.Integer() }
func (p *delegate) Float() float64      { return p.o.Float() }
func (p *delegate) Value(context *Scope) (v Value) {
        scope := p.o.Parent()
        if IsDummy(p.o) {
                if s := scope.Find(p.o.Name()); s != nil {
                        p.o = s
                }
        }
        //fmt.Printf("delegate: %T %v\n", p.o, p.o)
        if c, ok := p.o.(Caller); ok {
                var args []Value
                for _, a := range p.a {
                        if v, e := Disclosure(scope, a); e != nil {
                                // TODO: errors...
                                return UniversalNone
                        } else if v != nil {
                                //fmt.Printf("delegate.value: %v -> %v\n", a, v)
                                a = v
                        }
                        args = append(args, a)
                }
                v, _ = c.Call(args...)
        }
        if v == nil {
                v = UniversalNone
        }
        return v 
}

func (p *delegate) disclosure(scope *Scope, args []Value) (Value, error) {
        value := p.Value(scope)
        if v, e := value.disclosure(scope, args); e != nil {
                return nil, e
        } else if v != nil {
                //fmt.Printf("delegate.disclosure: %T %v -> %T %v -> %T %v\n", p.o, p.o, value, value, v, v)
                value = v
        }
        return value, nil
}

func Delegate(obj Object, args... Value) Value {
        //if d, ok := obj.(*delegate); ok {
        //        return d
        //} else {
        return &delegate{ obj, args }
}
