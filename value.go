//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/token"
        "path/filepath"
        "time"
        "net/url"
        "reflect"
        "strconv"
        "strings"
        "fmt"
        "os"
)

const (
        trace_compare = false
        trace_prepare = false
        trace_workdir = true && trace_prepare
)

type expendwhat int

const (
        expendDelegate expendwhat = 1<<iota // $(...) -> ...
        expendClosure // &(...) -> $(...)
        expendBoth = expendDelegate | expendClosure
)

// Value represents a value of a type.
type Value interface {
        // Type returns the underlying type of the value.
        Type() Type

        // Lit returns the literal representations of the value.
        String() string

        // Strval returns the string form of the value.
        Strval() (string, error)

        // Integer returns the integer form of the value.
        Integer() (int64, error)

        // Float returns the float form of the value.
        Float() (float64, error)

        // Recursively detecting whether this value references
        // the object (to avoid loop-delegation).
        refs(v Value) bool

        closured() bool

        // &(...) -> $(...)
        // $(...) -> ......
        expend(what expendwhat) (Value, error)
}

type closurecontext []*Scope

var cloctx closurecontext

func setclosure(cc closurecontext) (saved closurecontext) {
        saved = cloctx; cloctx = cc; return
}

func scoping(a ...*Project) (saved closurecontext) {
        saved = cloctx
        for _, i := range a {
                cloctx = append(cloctx, i.Scope())
        }
        return
}

func (cc closurecontext) unshift(scopers ...*Scope) closurecontext {
        return append(scopers, cc...)
}

func (cc closurecontext) append(scopers ...*Scope) closurecontext {
        return append(cc, scopers...)
}

func (cc closurecontext) String() (s string) {
        s = "closure{"
        for i, scope := range cc {
                if i > 0 { s += ", " }
                s += scope.comment
        }
        s += "}"
        return
}

type value struct {}
func (_ *value) refs(_ Value) bool { return false }
func (_ *value) closured() bool { return false }
func (_ *value) Type() Type { return InvalidType }
func (_ *value) Integer() (int64, error) { return 0, nil }
func (_ *value) Float() (float64, error) { return 0, nil }
func (p *value) Strval() (string, error) { return fmt.Sprintf("{value %p}", p), nil }
func (p *value) String() string {return "{value}" }

type comparer struct {
        globe *Globe
        target Value
        objects []Value
        result []Value
}

type filedepend interface {
        // Compare as a target with the file prerequisite.
        filedependcompare(c *comparer, file *File) error
}

type pathdepend interface {
        // Compare as a target with the path prerequisite.
        pathdependcompare(c *comparer, path *Path) error
}

type comparable interface {
        // Compare as a prerequisite with the target c.target.
        compare(c *comparer) error
}

func NewComparer(globe *Globe, target Value) (c *comparer, err error) {
        if trace_compare {
                // fmt.Printf("compare:Target: %v (%T) (revealed: %v)\n", target, target, Reveal(target))
        }
        if target, err = target.expend(expendDelegate); err != nil { return }
        if target == nil || target.Type() == NoneType {
                err = break_bad("comparing no target")
        } else if /*t, _ := target.(comparable); t != nil*/true {
                c = &comparer{ globe, target, nil, nil }
        } else {
                err = fmt.Errorf("incomparable target (%T %v)", target, target)
        }
        return
}

func (c *comparer) Compare(value interface{}) (err error) {
        if v := reflect.ValueOf(value); v.Kind() == reflect.Slice {
                for i := 0; i < v.Len(); i++ {
                        var dep = v.Index(i).Interface()
                        if err = c.compare(dep); err != nil {
                                if trace_compare {
                                        fmt.Printf("compare: %v (%v) (%v)\n", err, c.target, dep)
                                }
                                break
                        }
                }
        } else {
                err = c.compare(value)
        }
        return
}

func (c *comparer) compare(value interface{}) (err error) {
        if p, _ := value.(comparable); p != nil {
                err = p.compare(c)
        } else {
                err = fmt.Errorf("Type '%T' is not comparable.", value)
        }
        return
}

// preparer prepares prerequisites of targets.
type preparer struct {
        //entry *RuleEntry // caller entry
        program *Program
        arguments []Value
        targets *List
        stem string // set by StemmedEntry
}

type prerequisite interface {
        prepare(pc *preparer) error
}

func (pc *preparer) updateall(value interface{}) (err error) {
        if v := reflect.ValueOf(value); v.Kind() == reflect.Slice {
                for i := 0; i < v.Len(); i++ {
                        if err = pc.update(v.Index(i).Interface()); err == nil {
                                // Good!
                        } else if ute, ok := err.(targetNotFoundError); ok {
                                if trace_prepare {
                                        fmt.Printf("prepare: target `%v' not found\n", ute.target)
                                }
                                break
                        } else if ufe, ok := err.(fileNotFoundError); ok {
                                if trace_prepare {
                                        fmt.Printf("prepare: file `%v' not found\n", ufe.file)
                                }
                                break
                        } else {
                                break
                        }
                }
        } else {
                err = pc.update(value)
        }
        return
}

func (pc *preparer) update(value interface{}) (err error) {
        if p, _ := value.(prerequisite); p != nil {
                err = p.prepare(pc)
        } else if value == nil {
                err = fmt.Errorf("updating nil prerequisite")
        } else {
                err = fmt.Errorf("`%T` is not prerequisite", value)
        }
        return
}

func (pc *preparer) updateTarget(target string) (err error) {
        err = pc.program.project.updateTarget(pc, target)
        return
}

func (pc *preparer) updateTargetValue(value Value) (err error) {
        var s string
        if s, err = value.Strval(); err == nil {
                 err = pc.updateTarget(s)
        }
        return
}

func (pc *preparer) execute(entry *RuleEntry, prog *Program) (err error) {
        if trace_prepare {
                fmt.Printf("prepare:Execute: %v (%v) (%v)\n", entry.target, prog.depends, entry.class)
                for i, depent := range prog.depends {
                        fmt.Printf("prepare:Execute: %v (depend[%d]: %T %v %v)\n", entry.target, i, depent, depent, pc.stem)
                }
        }

        var res Value

        // Pase pc.stem to the program, so that patterns will work.
        defer func(s string) { prog.stem = s } (prog.stem)
        prog.stem = pc.stem

        // Execute the updating program.
        if res, err = prog.Execute(entry, pc.arguments); err == nil {
                dd, _ := prog.scope.Lookup("@").(*Def).Call(entry.Position)
                if trace_prepare {
                        fmt.Printf("prepare:Execute: %v (%v) (append %s (%T)) (%v)\n",
                                entry.target, entry.class, dd, dd, entry.target)
                }
                switch t := dd.(type) {
                case *File: pc.targets.Append(t)
                case *Path:
                        if t.File != nil {
                                pc.targets.Append(t.File)
                        } else {
                                pc.targets.Append(t)
                        }
                default:
                        var s string
                        if s, err = dd.Strval(); err != nil {
                                return
                        }
                        pc.targets.Append(prog.project.SearchFile(s))
                }
                if res != nil && res.Type() != NoneType {
                        for _, elem := range merge(res) {
                                switch elem.(type) {
                                case *File: pc.targets.Append(elem)
                                }
                        }
                }
        } else {
                fmt.Fprintf(os.Stdout, "%s: %v\n", prog.position, err)
                if trace_prepare {
                        fmt.Printf("prepare:Execute: %v (%v) (error)\n", entry.target, prog.depends)
                }
        }
        return
}

type Argumented struct {
        Val Value
        Args []Value
}
func (p *Argumented) refs(v Value) bool {
        if p.Val.refs(v) { return true }
        for _, a := range p.Args {
                if a.refs(v) { return true }
        }
        return false
}
func (p *Argumented) closured() bool {
        if p.Val.closured() { return true }
        for _, a := range p.Args {
                if a.closured() { return true }
        }
        return false
}
func (p *Argumented) expend(w expendwhat) (res Value, err error) {
        var (v Value; args []Value)
        if v, err = p.Val.expend(w); err == nil {
                var num int
                args, num, err = expendall(w, p.Args...)
                if err == nil && (num > 0 || v != p.Val) {
                        res = &Argumented{ v, args }
                }
        }
        if err == nil && res == nil {
                res = p
        }
        return
}
func (p *Argumented) Type() Type { return ArgumentedType }
func (p *Argumented) Integer() (i int64, err error) {
        var s string
        if s, err = p.Strval(); err == nil {
                i, err = strconv.ParseInt(s, 10, 64)
        }
        return
}
func (p *Argumented) Float() (f float64, err error) {
        var s string
        if s, err = p.Strval(); err == nil {
                f, err = strconv.ParseFloat(s, 64)
        }
        return
}
func (p *Argumented) String() (s string) {
        for i, a := range p.Args {
                if i > 0 { s += "," }
                s += a.String()
        }
        s = fmt.Sprintf("%s(%s)", p.Val, s)
        return
}
func (p *Argumented) Strval() (s string, err error) {
        if s, err = p.Val.Strval(); err != nil {
                return
        }
        s += "("
        for i, a := range p.Args {
                if i > 0 {
                        s += ","
                }
                var v string
                if v, err = a.Strval(); err == nil {
                        s += v
                } else {
                        break
                }
        }
        s += ")"
        return
}

func (p *Argumented) prepare(pc *preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:Argumented: %v\n", p)
        }
        pc.arguments = p.Args // TODO: merge args with p.Args ??
        return pc.update(p.Val)
}

type None struct { value }
func (p *None) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *None) Type() Type { return NoneType }
func (p *None) String() string {return "" }
func (p *None) Strval() (string, error) { return "", nil }
func (p *None) compare(c *comparer) (err error) { return }
func (p *None) filedependcompare(c *comparer, file *File) error { return nil }
func (p *None) pathdependcompare(c *comparer, path *Path) error { return nil }
func (p *None) prepare(pc *preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:None\n")
        }
        return nil 
}

type Nil struct { None }
type ModifierBar struct { None }
func (p *ModifierBar) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *ModifierBar) Strval() (string, error) { return "|", nil }
func (p *ModifierBar) String() string {return "|" }

type Any struct {
        Value interface{}
        value
}
func (p *Any) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *Any) Type() Type { return AnyType }

func MakeAny(v interface{}) *Any { return &Any{ Value:v } }

type integer struct {
        Value int64
}
func (p *integer) refs(_ Value) bool { return false }
func (p *integer) closured() bool { return false }
func (p *integer) Type() Type { return InvalidType }
func (p *integer) Integer() (int64, error) { return p.Value, nil }
func (p *integer) Float() (float64, error) { return float64(p.Value), nil }

type Bin struct { integer }
func (p *Bin) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *Bin) Type() Type { return BinType }
func (p *Bin) String() string { return fmt.Sprintf("0b%s", strconv.FormatInt(int64(p.Value),2)) }
func (p *Bin) Strval() (string, error) { return strconv.FormatInt(int64(p.Value),2), nil }

func MakeBin(i int64) *Bin { return &Bin{integer{i}} }
func ParseBin(s string) *Bin {
        if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
                s = s[2:]
        }
        if i, e := strconv.ParseInt(s, 2, 64); e == nil {
                return MakeBin(i)
        } else {
                panic(e)
        }
}

type Oct struct { integer }
func (p *Oct) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *Oct) Type() Type { return OctType }
func (p *Oct) String() string {
        /*if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{Oct '%s' !(%+v)}", s, e)
        }*/
        return fmt.Sprintf("0%s", strconv.FormatInt(int64(p.Value),8))
}
func (p *Oct) Strval() (string, error) { return strconv.FormatInt(int64(p.Value),8), nil }

func MakeOct(i int64) *Oct { return &Oct{integer{i}} }
func ParseOct(s string) *Oct {
        if strings.HasPrefix(s, "0") {
                s = s[1:]
        }
        if i, e := strconv.ParseInt(s, 8, 64); e == nil {
                return MakeOct(i)
        } else {
                panic(e)
        }
}

type Int struct { integer }
func (p *Int) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *Int) Type() Type { return IntType }
func (p *Int) String() string { return strconv.FormatInt(int64(p.Value),10) }
func (p *Int) Strval() (string, error) { return strconv.FormatInt(int64(p.Value),10), nil }

func MakeInt(i int64) *Int { return &Int{integer{i}} }
func ParseInt(s string) *Int {
        if i, e := strconv.ParseInt(s, 10, 64); e == nil {
                return MakeInt(i)
        } else {
                panic(e)
        }
}

type Hex struct { integer }
func (p *Hex) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *Hex) Type() Type { return HexType }
func (p *Hex) String() string { return fmt.Sprintf("0x%s", strconv.FormatInt(int64(p.Value),16)) }
func (p *Hex) Strval() (string, error) { return strconv.FormatInt(int64(p.Value),16), nil }

func MakeHex(i int64) *Hex { return &Hex{integer{i}} }
func ParseHex(s string) *Hex {
        if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
                s = s[2:]
        }
        if i, e := strconv.ParseInt(s, 16, 64); e == nil {
                return MakeHex(i)
        } else {
                panic(e)
        }
}

type Float struct {
        Value float64
}
func (p *Float) refs(_ Value) bool { return false }
func (p *Float) closured() bool { return false }
func (p *Float) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *Float) Type() Type { return FloatType }
func (p *Float) String() string { return strconv.FormatFloat(float64(p.Value),'g', -1, 64) }
func (p *Float) Strval() (string, error) { return strconv.FormatFloat(float64(p.Value),'g', -1, 64), nil }
func (p *Float) Integer() (int64, error) { return int64(p.Value), nil }
func (p *Float) Float() (float64, error) { return p.Value, nil }

func MakeFloat(f float64) *Float { return &Float{f} }
func ParseFloat(s string) *Float {
        if f, e := strconv.ParseFloat(strings.Replace(s, "_", "", -1), 64); e == nil {
                return MakeFloat(f)
        } else {
                panic(e)
        }
}


type DateTime struct {
        Value time.Time 
}
func (_ *DateTime) refs(_ Value) bool { return false }
func (_ *DateTime) closured() bool { return false }
func (p *DateTime) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *DateTime) Type() Type { return DateTimeType }
func (p *DateTime) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{DateTime '%s' !(%+v)}", s, e)
        }
}
func (p *DateTime) Strval() (string, error) { return time.Time(p.Value).Format("2006-01-02T15:04:05.999999999Z07:00"), nil } // time.RFC3339Nano
func (p *DateTime) Integer() (int64, error) { return p.Value.Unix(), nil }
func (p *DateTime) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

func MakeDateTime(s time.Time) *DateTime { return &DateTime{s} }
func ParseDateTime(s string) *DateTime {
        // time.RFC3339Nano
        if t, e := time.Parse("2006-01-02T15:04:05.999999999Z07:00", s); e == nil {
                return MakeDateTime(t)
        } else {
                panic(e)
        }
}

type Date struct { DateTime }
func (p *Date) Type() Type { return DateType }
func (p *Date) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{Date '%s' !(%+v)}", s, e)
        }
}
func (p *Date) Strval() (string, error) { return time.Time(p.Value).Format("2006-01-02"), nil }
func (p *Date) Integer() (int64, error) { return p.Value.Unix(), nil }
func (p *Date) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

func MakeDate(s time.Time) *Date { return &Date{DateTime{s}} }
func ParseDate(s string) *Date {
        if t, e := time.Parse("2006-01-02", s); e == nil {
                return MakeDate(t)
        } else {
                panic(e)
        }
}

type Time struct { DateTime }
func (p *Time) Type() Type { return TimeType }
func (p *Time) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{Time '%s' !(%+v)}", s, e)
        }
}
func (p *Time) Strval() (string, error) { return time.Time(p.Value).Format("15:04:05.999999999Z07:00"), nil }
func (p *Time) Integer() (int64, error) { return p.Value.Unix(), nil }
func (p *Time) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

func MakeTime(t time.Time) *Time { return &Time{DateTime{t}} }
func ParseTime(s string) *Time {
        if t, e := time.Parse("15:04:05.999999999Z07:00", s); e == nil {
                return MakeTime(t)
        } else {
                panic(e)
        }
}

type Uri struct {
        Value *url.URL
}
func (_ *Uri) refs(_ Value) bool { return false }
func (_ *Uri) closured() bool { return false }
func (p *Uri) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *Uri) Type() Type { return UriType }
func (p *Uri) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{Uri '%s' !(%+v)}", s, e)
        }
}
func (p *Uri) Strval() (string, error) { return p.Value.String(), nil }
func (p *Uri) Integer() (int64, error) { return int64(len(p.Value.String())), nil }
func (p *Uri) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

func MakeUri(s *url.URL) *Uri { return &Uri{s} }
func ParseUri(s string) *Uri {
        if u, e := url.Parse(s); e == nil {
                return MakeUri(u)
        } else {
                panic(e)
        }
}

type String struct {
        Value string
}
func (_ *String) refs(_ Value) bool { return false }
func (_ *String) closured() bool { return false }
func (p *String) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *String) Type() Type  { return StringType }
func (p *String) String() string { return fmt.Sprintf("'%s'", p.Value) }
func (p *String) Strval() (string, error) { return p.Value, nil }
func (p *String) Integer() (int64, error) { return strconv.ParseInt(p.Value, 10, 64) }
func (p *String) Float() (float64, error) { return strconv.ParseFloat(p.Value, 64) }

func (p *String) filedependcompare(c *comparer, d *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:String:File: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }

        var tt, dt time.Time
        if info, _ := os.Stat(p.Value); info != nil {
                tt = info.ModTime()
        }

        ds, err := d.Strval()
        if err != nil { return }
        if t, ok := c.globe.Timestamps[ds]; ok {
                dt = t
        } else if d.Info != nil {
                dt = d.Info.ModTime()
        } else if info, _ := os.Stat(ds); info != nil {
                dt = info.ModTime()
        } else {
                return break_bad("no such file '%v'", ds)
        }

        if dt.After(tt) {
                c.globe.Timestamps[p.Value] = dt // Or tt?
                err = nil // Returns nil to request update.
        } else {
                err = break_good("updated file '%s'", p)
        }
        return
}

func (p *String) pathdependcompare(c *comparer, d *Path) (err error) {
        if trace_compare {
                fmt.Printf("compare:String:Path: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }

        var tt, dt time.Time
        if info, _ := os.Stat(p.Value); info != nil {
                tt = info.ModTime()
        }

        s, err := d.Strval()
        if err != nil { return }
        if t, ok := c.globe.Timestamps[s]; ok {
                dt = t
        } else if d.File != nil && d.File.Info != nil {
                dt = d.File.Info.ModTime()
        } else if info, _ := os.Stat(s); info != nil {
                dt = info.ModTime()
        } else {
                return break_bad("no such directory '%v'", s)
        }

        if dt.After(tt) {
                c.globe.Timestamps[p.Value] = dt // Or tt?
                err = nil // Returns nil to request update.
        } else {
                err = break_good("updated directory '%s'", p)
        }
        return
}

func (p *String) prepare(pc *preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:String: %v\n", p)
        }
        //pc.source = p.Value
        return pc.updateTarget(p.Value)
}

func MakeString(s string) *String { return &String{s} }

type Bareword struct {
        Value string
}
func (_ *Bareword) refs(_ Value) bool { return false }
func (_ *Bareword) closured() bool { return false }
func (p *Bareword) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *Bareword) Type() Type     { return BarewordType }
func (p *Bareword) String() string { return p.Value/*fmt.Sprintf("Bareword{%s}", p.Value)*/ }
func (p *Bareword) Strval() (string, error) { return p.Value, nil }
func (p *Bareword) Integer() (int64, error) { return strconv.ParseInt(p.Value, 10, 64) }
func (p *Bareword) Float() (float64, error) { return strconv.ParseFloat(p.Value, 64) }

func (p *Bareword) filedependcompare(c *comparer, d *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:Bareword:File: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }

        var tt, dt time.Time
        if info, _ := os.Stat(p.Value); info != nil {
                tt = info.ModTime()
        }

        ds, err := d.Strval()
        if err != nil { return }
        if t, ok := c.globe.Timestamps[ds]; ok {
                dt = t
        } else if d.Info != nil {
                dt = d.Info.ModTime()
        } else if info, _ := os.Stat(ds); info != nil {
                dt = info.ModTime()
        } else {
                return break_bad("no such file '%v'", ds)
        }

        if dt.After(tt) {
                c.globe.Timestamps[p.Value] = dt // Or tt?
                err = nil // Returns nil to request update.
        } else {
                err = break_good("updated file '%s'", p)
        }
        return
}

func (p *Bareword) pathdependcompare(c *comparer, d *Path) (err error) {
        if trace_compare {
                fmt.Printf("compare:Bareword:Path: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }

        var tt, dt time.Time
        if info, _ := os.Stat(p.Value); info != nil {
                tt = info.ModTime()
        }

        s, err := d.Strval()
        if err != nil { return }
        if t, ok := c.globe.Timestamps[s]; ok {
                dt = t
        } else if d.File != nil && d.File.Info != nil {
                dt = d.File.Info.ModTime()
        } else if info, _ := os.Stat(s); info != nil {
                dt = info.ModTime()
        } else {
                return break_bad("no such directory '%v'", s)
        }

        if dt.After(tt) {
                c.globe.Timestamps[p.Value] = dt // Or tt?
                err = nil // Returns nil to request update.
        } else {
                err = break_good("updated directory '%s'", p)
        }
        return
}

func (p *Bareword) prepare(pc *preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:Bareword: %v\n", p)
        }
        //pc.source = p.Value
        return pc.updateTarget(p.Value)
}

func MakeBareword(s string) *Bareword { return &Bareword{s} }

type Elements struct {
        Elems []Value
}
func (p *Elements) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }
func (p *Elements) Integer() (int64, error) {
        if n := len(p.Elems); n == 1 {
                // If there's only one element, treat it as a scalar.
                return p.Elems[0].Integer()
        } else {
                return int64(n), nil
        }
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

func (p *Elements) refs(v Value) bool {
        for _, elem := range p.Elems {
                if elem != nil && (elem == v || elem.refs(v)) {
                        return true
                }
        }
        return false 
}

func (p *Elements) closured() bool {
        for _, elem := range p.Elems {
                if elem.closured() { return true }
        }
        return false 
}

type Barecomp struct {
        Elements
}
func (p *Barecomp) Type() Type { return BarecompType }
func (p *Barecomp) Strval() (s string, e error) {
        for _, elem := range p.Elems {
                var v string
                if v, e = elem.Strval(); e == nil {
                        s += v
                } else {
                        break
                }
        }
        return
}
func (p *Barecomp) String() (s string) {
        for _, elem := range p.Elems {
                s += elem.String()
        }
        return
}

func (p *Barecomp) expend(w expendwhat) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = expendall(w, p.Elems...); err == nil {
                if num > 0 {
                        res = &Barecomp{ Elements{ elems } }
                } else {
                        res = p
                }
        }
        return
}

func (p *Barecomp) filedependcompare(c *comparer, d *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:Barecomp:File: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }

        var tt time.Time
        ts, err := p.Strval() // target name
        if err != nil { return }
        if info, _ := os.Stat(ts); info != nil {
                tt = info.ModTime()
        } else {
                return // Returns nil to request update.
        }

        var dt time.Time
        ds, err := d.Strval() // depend name
        if err != nil { return }
        if t, ok := c.globe.Timestamps[ds]; ok {
                dt = t
        } else if d.Info != nil {
                dt = d.Info.ModTime()
        } else if info, _ := os.Stat(ds); info != nil {
                dt = info.ModTime()
        } else {
                return break_bad("no such file '%v'", ds)
        }

        if dt.After(tt) {
                c.globe.Timestamps[ts] = dt // Or tt?
                err = nil // Returns nil to request update.
        } else {
                err = break_good("updated file '%s'", p)
        }
        return
}

func (p *Barecomp) pathdependcompare(c *comparer, d *Path) (err error) {
        if trace_compare {
                fmt.Printf("compare:Barecomp:Path: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }

        var tt time.Time
        ts, err := p.Strval()
        if err != nil { return }
        if info, _ := os.Stat(ts); info != nil {
                tt = info.ModTime()
        } else {
                return // Returns nil to request update.
        }

        var dt time.Time
        ds, err := d.Strval()
        if err != nil { return }
        if t, ok := c.globe.Timestamps[ds]; ok {
                dt = t
        } else if d.File != nil && d.File.Info != nil {
                dt = d.File.Info.ModTime()
        } else if info, _ := os.Stat(ds); info != nil {
                dt = info.ModTime()
        } else {
                return break_bad("no such file '%v'", ds)
        }

        if dt.After(tt) {
                c.globe.Timestamps[ts] = dt // Or tt?
                err = nil // Returns nil to request update.
        } else {
                err = break_good("updated file '%s'", p)
        }
        return
}

func (p *Barecomp) prepare(pc *preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:Barecomp: %v\n", p)
                for _, elem := range p.Elems {
                        fmt.Printf("prepare:Barecomp: %v (%v)\n", p, elem)
                }
        }
        return pc.updateTargetValue(p)
}

func MakeBarecomp(elems... Value) *Barecomp {
        return &Barecomp{Elements{elems}}
}

type Barefile struct {
        Name Value
        File *File
}
func (p *Barefile) refs(v Value) bool { return p.Name.refs(v) }
func (p *Barefile) closured() bool { return p.Name.closured() }
func (p *Barefile) expend(w expendwhat) (res Value, err error) {
        var name Value
        if name, err = p.Name.expend(w); err == nil {
                if name != nil {
                        res = &Barefile{ name, p.File }
                } else {
                        res = p
                }
        }
        return
}
func (p *Barefile) Type() Type { return BarefileType }
func (p *Barefile) String() string { return p.Name.String() }
func (p *Barefile) Strval() (string, error) { return p.Name.Strval() }
func (p *Barefile) Integer() (res int64, err error) {
        var ( str string; fi os.FileInfo )
        if str, err = p.Name.Strval(); err != nil { return }
        if fi, err = os.Stat(str); err == nil {
                res = fi.Size()
        }
        return
}
func (p *Barefile) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

func (p *Barefile) compare(c *comparer) (err error) {
        if trace_compare {
                fmt.Printf("compare:Barefile: %v (%v %T)\n", p.Name, c.target, c.target)
        }
        if p.File != nil {
                err = p.File.compare(c)
        } else {
                err = break_bad("no such file '%v'", p)
        }
        return
}

func (p *Barefile) filedependcompare(c *comparer, d *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:Barefile:File: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }
        if p.File != nil {
                err = p.File.filedependcompare(c, d)
        } else {
                err = break_bad("no such file '%v'", p)
        }
        return
}

func (p *Barefile) pathdependcompare(c *comparer, d *Path) (err error) {
        if trace_compare {
                fmt.Printf("compare:Barefile:Path: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }
        if p.File != nil {
                err = p.File.pathdependcompare(c, d)
        } else {
                err = break_bad("no such path '%v'", p)
        }
        return
}

func (p *Barefile) prepare(pc *preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:Barefile: %v\n", p)
        }
        if p.File != nil {
                if s, e := p.Name.Strval(); e != nil {
                        return e
                } else if s != p.File.Name {
                        p.File.Name = s // Fix it in case of '$@.o' was parsed and became '.o'.
                }
                return p.File.prepare(pc)
        } else {
                return pc.updateTargetValue(p)
        }
}

func MakeBarefile(name Value, file *File) *Barefile {
        return &Barefile{ name, file }
}

type Glob struct {
        Tok token.Token
}
func (p *Glob) refs(o Value) bool { return false }
func (p *Glob) closured() bool { return false }
func (p *Glob) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *Glob) Type() Type { return GlobType }
func (p *Glob) String() (s string) { return p.Tok.String() }
func (p *Glob) Strval() (string, error) { return p.Tok.String(), nil }
func (p *Glob) Integer() (int64, error) { return 0, nil }
func (p *Glob) Float() (float64, error) { return 0, nil }

func MakeGlob(tok token.Token) *Glob { return &Glob{tok} }

type Path struct {
        Elements
        File *File // if this path is pointed to a file, ie. the last element matched a FileMap
}
func (p *Path) String() (s string) {
        var segs []string
        for _, elem := range p.Elems {
                segs = append(segs, elem.String())
        }
        return strings.Join(segs, PathSep)
}
func (p *Path) Strval() (s string, e error) {
        // TODO: add '/' for root dir
        var sep = true
        for i, seg := range p.Elems {
                if i > 0 && sep {
                        s += string(os.PathSeparator) 
                }
                var v string
                if v, e = seg.Strval(); e != nil { return }
                s += v
                if ps, ok := seg.(*PathSeg); ok && ps != nil && ps.Value == '/' {
                        sep = false
                } else {
                        sep = true
                }
        }
        // TODO: add '/' if there's such a suffix
        return
}
func (p *Path) Integer() (int64, error) { return 0, nil }
func (p *Path) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }
func (p *Path) Type() Type { return PathType }
func (p *Path) expend(w expendwhat) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = expendall(w, p.Elems...); err == nil {
                if num > 0 {
                        res = &Path{Elements{elems}, p.File}
                } else {
                        res = p
                }
        }
        return
}

func (p *Path) compare(c *comparer) (err error) {
        if trace_compare {
                fmt.Printf("compare:Path: %v (%v %T)\n", p, c.target, c.target)
        }
        if dep, ok := c.target.(pathdepend); ok {
                if err = dep.pathdependcompare(c, p); err != nil {
                        if p, ok := err.(*breaker); !ok || !(p != nil && p.good) {
                                err = fmt.Errorf("path: %v", err)
                        }
                }
        } else {
                err = fmt.Errorf("path: target is not path depend (%T %v)", c.target, c.target)
        }
        return
}

func (p *Path) filedependcompare(c *comparer, d *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:Path:File: %v (%v) (depends: %v %v) (%v %T)\n", p, p.File, d, d.Info, c.target, c.target)
        }
        if p.File != nil {
                return p.File.filedependcompare(c, d)
        }
                
        var tt time.Time
        ts, err := p.Strval()
        if err != nil { return }
        if p.File != nil && p.File.Info != nil {
                tt = p.File.Info.ModTime()
        } else if info, _ := os.Stat(ts); info != nil {
                tt = info.ModTime()
        } else {
                return // Returns nil to request update.
        }

        var dt time.Time
        ds, err := d.Strval()
        if err != nil { return }
        if t, ok := c.globe.Timestamps[ds]; ok {
                dt = t
        } else if d.Info != nil {
                dt = d.Info.ModTime()
        } else if info, _ := os.Stat(ds); info != nil {
                dt = info.ModTime()
        } else {
                return break_bad("no such directory '%v'", ds)
        }

        if dt.After(tt) {
                c.globe.Timestamps[ts] = dt // Or tt?
                err = nil // Returns nil to request update.
        } else {
                err = break_good("updated path '%s'", p)
        }
        return
}

func (p *Path) pathdependcompare(c *comparer, d *Path) (err error) {
        if trace_compare {
                fmt.Printf("compare:Path:Path: %v (%v) (depends: %v) (%v %T)\n", p, p.File, d, c.target, c.target)
        }
        if p.File != nil {
                return p.File.pathdependcompare(c, d)
        }

        var tt time.Time
        ts, err := p.Strval()
        if err != nil { return }
        if p.File != nil && p.File.Info != nil {
                tt = p.File.Info.ModTime()
        } else if info, _ := os.Stat(ts); info != nil {
                tt = info.ModTime()
        } else {
                return // Returns nil to request update.
        }

        var dt time.Time
        ds, err := d.Strval()
        if err != nil { return }
        if t, ok := c.globe.Timestamps[ds]; ok {
                dt = t
        } else if d.File != nil && d.File.Info != nil {
                dt = d.File.Info.ModTime()
        } else if info, _ := os.Stat(ds); info != nil {
                dt = info.ModTime()
        } else {
                return break_bad("no such directory '%v'", ds)
        }

        if dt.After(tt) {
                c.globe.Timestamps[ts] = dt // Or tt?
                err = nil // Returns nil to request update.
        } else {
                err = break_good("updated path '%s'", p)
        }
        return
}

func (p *Path) prepare(pc *preparer) (err error) {
        if trace_prepare {
                if p.File != nil {
                        fmt.Printf("prepare:Path: %v (file: %v) (%v)\n", p, p.File, pc.program.project.name)
                } else {
                        fmt.Printf("prepare:Path: %v (%v)\n", p, pc.program.project.name)
                }
        }

        var s string // path/file target
        if s, err = p.Strval(); err != nil {
                return
        }

        if p.File == nil {
                if pc.program.project.isFile(filepath.Base(s)) || pc.program.project.isFile(s) {
                        if p.File = pc.program.project.SearchFile(s); p.File != nil {
                                if trace_prepare {
                                        fmt.Printf("prepare:Path: %v (found file '%v' in %v)\n", p, p.File, pc.program.project.name)
                                }
                        }
                }
        }

        if p.File != nil {
                err = p.File.prepare(pc)
        } else if err = pc.updateTarget(s); err == nil {
                // Good!
        } else if e, ok := err.(targetNotFoundError); ok {
                if info, _ := os.Stat(e.target); info == nil {
                        pc.targets.Append(p) // Append unknown path anyway.
                        fmt.Printf("path.prepare: 1: %v\n", pc.targets)
                        if trace_prepare {
                                fmt.Printf("prepare:Path: %v (unknown path: %v)\n", p, e.target)
                        }
                } else if info.IsDir() {
                        pc.targets.Append(p)
                        fmt.Printf("path.prepare: 2: %v\n", pc.targets)
                        if trace_prepare {
                                fmt.Printf("prepare:Path: %v (found unknown path: %v)\n", p, e.target)
                        }
                } else {
                        // Search this path target as a file.
                        p.File = pc.program.project.SearchFile(e.target)
                        pc.targets.Append(p.File)
                        fmt.Printf("path.prepare: 3: %v\n", pc.targets)
                        if trace_prepare {
                                fmt.Printf("prepare:Path: %v (found unknown target: %v) (file: %v)\n", p, e.target, p.File.Fullname())
                        }
                }
                // Make it a path-not-found error.
                err = pathNotFoundError{ p }
        }
        return
}

func MakePath(segments... Value) (v *Path) {
        return &Path{Elements{segments}, nil}
}

type PathSeg struct {
        Value rune 
        value
}
func (p *PathSeg) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *PathSeg) Type() Type { return PathSegType }
func (p *PathSeg) String() string { 
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return "?"
        }
}
func (p *PathSeg) Strval() (s string, e error) {
        switch p.Value {
        case '/': s = "/"
        case '~': s = "~"
        case '.': s = "."
        case '^': s = ".."
        default: e = fmt.Errorf("unknown pathseg (%s)", p.Value)
        }
        return
}

func MakePathSeg(ch rune) *PathSeg { return &PathSeg{ Value:ch } }

type File struct {
        value            // satisify Value interface
        Name string      // constant represented name (e.g. relative filename)
        Match *FileMap   // matched pattern (see 'files' directive)
        Sub Value        // matched sub path (in Project.SearchFile), may be absolete 
        Dir string       // full directory where the file was or should be found
        Info os.FileInfo // file info if exists
}
func (p *File) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *File) Type() Type { return FileType }
func (p *File) String() string { return p.Name }
func (p *File) Strval() (string, error) {
        if p.Sub != nil {
                if s, err := p.Sub.Strval(); err != nil {
                        return "", err
                } else {
                        return filepath.Join(s, p.Name), nil
                }
        }
        return p.Name, nil
}

func (p *File) Fullname() (s string) {
        if filepath.IsAbs(p.Name) {
                s = p.Name
        } else {
                s = filepath.Join(p.Dir, p.Name)
        }
        return
}
func (p *File) Basename() string {
        if p.Info != nil {
                return p.Info.Name()
        } else {
                return filepath.Base(p.Name)
        }
}

func (p *File) exists() bool { return p.Info != nil }

func (p *File) compare(c *comparer) (err error) {
        if trace_compare {
                fmt.Printf("compare:File: %v (%v %T)\n", p.Name, c.target, c.target)
        }
        if dep, ok := c.target.(filedepend); ok {
                if err = dep.filedependcompare(c, p); err != nil {
                        if p, ok := err.(*breaker); !ok || !(p != nil && p.good) {
                                err = fmt.Errorf("file: %v", err)
                        }
                }
        } else {
                err = fmt.Errorf("entry: not path depend (%T %v)", c.target, c.target)
        }
        return
}

func (p *File) filedependcompare(c *comparer, d *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:File:File: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }
        
        var tt time.Time
        ts, err := p.Strval() // target name
        if err != nil { return }
        if p.Info != nil {
                tt = p.Info.ModTime()
        } else if p.Info, _ = os.Stat(ts); p.Info != nil {
                tt = p.Info.ModTime()
        } else {
                return // Returns nil to request update.
        }

        var dt time.Time
        ds, err := d.Strval() // depend name
        if err != nil { return }
        if t, ok := c.globe.Timestamps[ds]; ok {
                dt = t
        } else if d.Info != nil {
                dt = d.Info.ModTime()
        } else if info, _ := os.Stat(ds); info != nil {
                dt = info.ModTime()
        } else {
                return break_bad("no such file '%v'", ds)
        }

        if dt.After(tt) {
                c.globe.Timestamps[ts] = dt // Or tt?
                err = nil // Returns nil to request update.
        } else {
                err = break_good("updated file '%s'", p)
        }
        return
}

func (p *File) pathdependcompare(c *comparer, d *Path) (err error) {
        if trace_compare {
                fmt.Printf("compare:File:Path: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }

        var tt time.Time
        ts, err := p.Strval()
        if err != nil { return }
        if p.Info != nil {
                tt = p.Info.ModTime()
        } else if p.Info, _ = os.Stat(ts); p.Info != nil {
                tt = p.Info.ModTime()
        } else {
                return // Returns nil to request update.
        }

        var dt time.Time
        ds, err := d.Strval()
        if err != nil { return }
        if t, ok := c.globe.Timestamps[ds]; ok {
                dt = t
        } else if d.File != nil && d.File.Info != nil {
                dt = d.File.Info.ModTime()
        } else if info, _ := os.Stat(ds); info != nil {
                dt = info.ModTime()
        } else {
                return break_bad("no such file '%v'", ds)
        }

        if dt.After(tt) {
                c.globe.Timestamps[ts] = dt // Or tt?
                err = nil // Returns nil to request update.
        } else {
                err = break_good("updated file '%s'", p)
        }
        return
}

func (p *File) prepare(pc *preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:File: %v (%v) (%v)\n", p.Name, p.Dir, pc.program.project.name)
        }

        if p.Dir != "" {
                if info, err := os.Stat(p.Dir); err != nil || info == nil {
                        if err = os.MkdirAll(p.Dir, 0755); err != nil {
                                return err
                        }
                }
        }

        if err, brk := p.explicitly(pc); err != nil || brk {
                return err
        }
        if err, brk := p.implicitly(pc); err != nil || brk {
                return err
        }

        if p.exists() {
                if trace_prepare {
                        fmt.Printf("prepare:File: %v (search: exists %v)\n", p.Name, p)
                }
                pc.targets.Append(p)
        } else if pc.program.project.search(p) {
                if trace_prepare {
                        fmt.Printf("prepare:File: %v (search: known as %v but missing) (%v)\n",
                                p.Name, p, pc.program.project.name)
                }
                pc.targets.Append(p)
        } else {
                if trace_prepare {
                        fmt.Printf("prepare:File: %v (search: unknown %v) (%v)\n",
                                p.Name, p.Dir, pc.program.project.name)
                }
                return fileNotFoundError{ p }
        }
        return nil
}

func (p *File) explicitly(pc *preparer) (err error, trybrk bool) {
        if trace_prepare {
                fmt.Printf("prepare:File: %v (explicitly: %v in %v)\n", p.Name, p, pc.program.project.name)
        }
        var entry *RuleEntry
        // Find concrete entry (by file represented name)
        if entry, err = pc.program.project.resolveEntry(p.Name); err != nil {
                // ... error
        } else if entry == nil {
                // Search into the upper projects for matched a rule.
                for _, proj := range execstack.projects() {
                        if proj == pc.program.project { continue }
                        if entry, err = proj.resolveEntry(p.Name); err != nil {
                                break
                        } else if entry != nil {
                                err, trybrk = entry.prepare(pc), true
                                break
                        }
                }
        } else {
                err, trybrk = entry.prepare(pc), true
        }
        return
}

func (p *File) implicitly(pc *preparer) (err error, trybrk bool) {
        if trace_prepare {
                fmt.Printf("prepare:File: %v (implicitly: %v in %v)\n", p.Name, p, pc.program.project.name)
        }

        var pss []*StemmedEntry
        if pss, err = pc.program.project.resolvePatterns(p.Name); err != nil {
                return
        }

        ForPatterns: for i, ps := range pss {
                for _, prog := range ps.Patent.programs {
                        if trace_prepare {
                                fmt.Printf("prepare:File: %v (implicitly:%d: %v : %v) (in %v)\n", p.Name, i, ps, prog.depends, pc.program.project.name)
                        }
                        for _, dep := range prog.depends {
                                var ( g, _ = dep.(*GlobPattern); ok bool )
                                if g != nil {
                                        if ok, err = p.checkPatternDepend(pc, pc.program.project, ps, prog, g); err != nil { return }
                                        if !ok { continue ForPatterns }
                                }
                        }
                }
                ps.file = p // Bounds StemmedEntry with the File.
                if err = ps.prepare(pc); err == nil {
                        trybrk = true; break ForPatterns // Updated successfully!
                } else if _, ok := err.(patternPrepareError); ok {
                        if trace_prepare {
                                fmt.Printf("prepare:File: %v (implicitly:%d: %v) (error: %s) (%s)\n", p.Name, i, ps, err, pc.program.project.name)
                        }
                } else {
                        trybrk = true; break ForPatterns // Update failed!
                }
        }
        return
}

func (p *File) checkPatternDepend(pc *preparer, project *Project, ps *StemmedEntry, prog *Program, g *GlobPattern) (res bool, err error) {
        var name string
        if name, err = g.MakeString(ps.Stem); err != nil { return }
        if file := project.file(name); file != nil { // Matches a FileMap (IsKnown(), may exists or not)
                //fmt.Printf("prepare:File: %v (implicitly:=: %v in %s)\n", p.Name, file, project.name)
                if file.exists() {
                        if trace_prepare {
                                fmt.Printf("prepare:File: %v (implicitly: %v exists in %s)\n", p.Name, file, project.name)
                        }
                        res = true
                } else if trace_prepare && false {
                        fmt.Printf("prepare:File: %v (implicitly: %v missing in %s)\n", p.Name, file, project.name)
                }
        }
        if _, sym := project.scope.Find(name); sym != nil {
                if trace_prepare {
                        fmt.Printf("prepare:File: %v (implicitly: found %v in %s)\n", p.Name, sym, project.name)
                }
                res = true
        }

        // TODO: recursive find patterns:
        /*if project.FindPatterns(name) != nil {
                res = true
        }*/
        return
}

func MakeFile(s string) (fv *File) { return &File{ Name:s } }

type Flag struct {
        Name Value
}
func (p *Flag) refs(v Value) bool { return p.Name.refs(v) }
func (p *Flag) closured() bool { return p.Name.closured() }
func (p *Flag) expend(w expendwhat) (res Value, err error) {
        var name Value
        if name, err = p.Name.expend(w); err == nil {
                if name != nil {
                        res = &Flag{ name }
                } else {
                        res = p
                }
        }
        return
}
func (p *Flag) Type() Type { return FlagType }
func (p *Flag) String() (s string) { return fmt.Sprintf("-%s", p.Name.String()) }
func (p *Flag) Strval() (s string, e error) {
        if s, e = p.Name.Strval(); e == nil { 
                 s = "-" + s
        }
        return
}
func (p *Flag) Integer() (int64, error) { return 0, nil }
func (p *Flag) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

func MakeFlag(name Value) (v *Flag) { return &Flag{name} }
        
type Compound struct { // "compound string"
        Elements
}
func (p *Compound) String() (s string) {
        for _, elem := range p.Elems {
                s += elem.String()
        }
        return fmt.Sprintf(`"%s"`, s)
}
func (p *Compound) Strval() (s string, err error) {
        for _, e := range p.Elems {
                var v string
                if v, err = e.Strval(); err == nil {
                        s += v
                } else {
                        break
                }
        }
        return
}
func (p *Compound) Type() Type { return CompoundType }
func (p *Compound) Integer() (int64, error) { return int64(len(p.Elems)), nil }
func (p *Compound) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

func (p *Compound) expend(w expendwhat) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = expendall(w, p.Elems...); err == nil {
                if num > 0 {
                        res = &Compound{ Elements{ elems } }
                } else {
                        res = p
                }
        }
        return
}

func MakeCompound(elems... Value) (v *Compound) {
        return &Compound{Elements{elems}}
}

type List struct {
        Elements
}
func (p *List) Type() Type { return ListType }
func (p *List) String() (s string) {
        var strs []string
        for _, elem := range p.Elems {
                strs = append(strs, elem.String())
        }
        return strings.Join(strs, " ")
}
func (p *List) Strval() (s string, err error) {
        var x = 0
        for _, e := range p.Elems {
                var v string
                if v, err = e.Strval(); err == nil {
                        if v != "" {
                                if 0 < x { s += " " }
                                s += v
                                x += 1
                        }
                } else {
                        break
                }
        }
        return
}

func (p *List) expend(w expendwhat) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = expendall(w, p.Elems...); err == nil {
                if num > 0 {
                        res = &List{ Elements{ elems } }
                } else {
                        res = p
                }
        }
        return
}

func (p *List) compare(c *comparer) (err error) {
        if trace_compare {
                fmt.Printf("compare:List: %v (%v %T)\n", p, c.target, c.target)
        }
        for _, elem := range p.Elems {
                if err = c.compare(elem); err != nil {
                        break
                }
        }
        return
}

func (p *List) filedependcompare(c *comparer, d *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:List:File: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }
        if n := len(p.Elems); n == 1 {
                if elem, _ := p.Elems[0].(filedepend); elem != nil {
                        err = elem.filedependcompare(c, d)
                } else {
                        err = break_bad("list: incomparable target (%T %v)", p.Elems[0], p.Elems[0])
                }
        } else if n == 0 {
                err = break_bad("comparing empty list")
        } else {
                err = break_bad("comparing multiple targets (%v)", p)
        }
        return
}

func (p *List) pathdependcompare(c *comparer, d *Path) (err error) {
        if trace_compare {
                fmt.Printf("compare:List:Path: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }
        if n := len(p.Elems); n == 1 {
                if elem, _ := p.Elems[0].(pathdepend); elem != nil {
                        err = elem.pathdependcompare(c, d)
                } else {
                        err = break_bad("list: incomparable target (%T %v)", p.Elems[0], p.Elems[0])
                }
        } else if n == 0 {
                err = break_bad("comparing empty list")
        } else {
                err = break_bad("comparing multiple targets (%v)", p)
        }
        return
}

func MakeList(elems... Value) *List { return &List{Elements{elems}} }

type Group struct {
        List
}
func (p *Group) Type() Type { return GroupType }
func (p *Group) String() string {
        var strs []string
        for _, elem := range p.Elems {
                strs = append(strs, elem.String())
        }
        return fmt.Sprintf("(%s)", strings.Join(strs, " "))
}
func (p *Group) Strval() (s string, err error) {
        if s, err = p.List.Strval(); err == nil {
                s = "(" + s + ")"
        }
        return
}

func (p *Group) expend(w expendwhat) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = expendall(w, p.Elems...); err == nil {
                if num > 0 {
                        res = &Group{ List{ Elements{ elems } } }
                } else {
                        res = p
                }
        }
        return
}

func MakeGroup(elems... Value) (v *Group) {
        return &Group{List{Elements{elems}}}
}

//type Map struct {
//        Elems map[string]Value
//}
/* func (p *Map) String() string {
        return "(" + p.List.String() + ")"
}
func (p *Map) Strval() string {
        return "(" + p.List.Strval() + ")"
} */

type Pair struct { // key=value
        Key Value
        Value Value
}
func (p *Pair) refs(v Value) bool { return p.Key.refs(v) || p.Value.refs(v) }
func (p *Pair) closured() bool { return p.Key.closured() || p.Value.closured() }
func (p *Pair) expend(x expendwhat) (res Value, err error) {
        var k, v Value
        if k, err = p.Key.expend(x); err == nil {
                if v, err = p.Value.expend(x); err == nil {
                        if k != nil || v != nil {
                                if k == nil { k = p.Key }
                                if v == nil { v = p.Value }
                                res = &Pair{ k, v }
                        } else {
                                res = p
                        }
                }
        }
        return
}
func (p *Pair) Type() Type { return PairType }
func (p *Pair) String() string {
        return fmt.Sprintf("%s=%s", p.Key.String(), p.Value.String())
}
func (p *Pair) Strval() (s string, err error) {
        var k, v string
        if k, err = p.Key.Strval(); err != nil {
                if v, err = p.Value.Strval(); err == nil {
                        s = k + "=" + v
                }
        }
        return
}
func (p *Pair) Integer() (int64, error) { return p.Value.Integer() }
func (p *Pair) Float() (float64, error) { return p.Value.Float() }

func (p *Pair) SetValue(v Value) { p.Value = v }
func (p *Pair) SetKey(k Value) {
        switch o := k.(type) {
        case *Pair: k = o.Key
        }
        if k.Type().Bits()&IsKeyName != 0 {
                p.Key = k
        } else {
                panic(fmt.Errorf("'%T' is not key type", k))
        }
}

func MakePair(k, v Value) (p *Pair) {
        if k.Type().Bits()&IsKeyName != 0 {
                p = &Pair{nil, nil}
                p.SetKey(k)
                p.SetValue(v)
        } else {
                panic(fmt.Errorf("'%T' is not key type", k))
        }
        return
}

type closuredelegate struct {
        p token.Position
        l token.Token
        o Object
        a []Value
}

func (p *closuredelegate) Position() token.Position { return p.p }
func (p *closuredelegate) string(t string) (s string) { // source representation
        for i, a := range p.a {
                if i == 0 { s = " " } else { s += "," }
                s += a.String()
        }
        switch name := p.o.Name(); p.l {
        case token.COLON: s = fmt.Sprintf("%s:%s%s:", t, name, s)
        case token.LPAREN: s = fmt.Sprintf("%s(%s%s)", t, name, s)
        case token.LBRACE: s = fmt.Sprintf("%s{%s%s}", t, name, s)
        case token.STRING, token.COMPOUND:
                s = fmt.Sprintf("%s%s%s", t, name, s)
        case token.ILLEGAL:
                if len(name) == 1 && len(s) == 0 {
                        s = fmt.Sprintf("%s%s", t, name)
                } else {
                        s = fmt.Sprintf("%s[%s%s]", t, name, s)
                }
        default:
                s = fmt.Sprintf("%s[%s%s]!(%v)", t, name, s, p.l)
        }
        return
}

// Delegate wraps '$(foo a,b,c)' into Valuer
type delegate struct { closuredelegate }
func (p *delegate) Type() Type { return DelegateType }
func (p *delegate) String() (s string) { return p.string("$") }
func (p *delegate) Strval() (string, error) { if v, e := p.expend(expendDelegate); e == nil { return v.Strval() } else { return "", e }}
func (p *delegate) Integer() (int64, error) { if v, e := p.expend(expendDelegate); e == nil { return v.Integer() } else { return 0, e }}
func (p *delegate) Float() (float64, error) { if v, e := p.expend(expendDelegate); e == nil { return v.Float() } else { return 0, e }}
func (p *delegate) expend(w expendwhat) (res Value, err error) {
        switch {
        case w&expendClosure != 0:
                if res, err = p.disclose(); err != nil {
                        return
                }
                if res != nil && w&expendDelegate != 0 {
                        res, err = res.expend(expendDelegate)
                }
        case w&expendDelegate != 0:
                if res, err = p.reveal(); err != nil {
                        return
                }
                if res != nil && w&expendClosure != 0 {
                        res, err = res.expend(expendClosure)
                }
        }
        if err == nil && res == nil { res = p }
        return
}

func (p *delegate) reveal() (res Value, err error) {
        var args []Value
        if args, _, err = expendall(expendClosure, p.a...); err != nil {
                return
        }

        switch o := p.o.(type) {
        default: err = fmt.Errorf("unknown delegation `%v` (%T)", o, o)
        case Caller:
                if res, err = o.Call(p.p, args...); err != nil {
                        if p.o.Name() != "error" {
                                err = fmt.Errorf("%v (%s)", err, p)
                        } else {
                                return
                        }
                }
        case Executer:
                if args, err = o.Execute(p.p, args...); err != nil {
                        if p.o.Name() != "error" {
                                err = fmt.Errorf("%v (%s)", err, p)
                        } else {
                                return
                        }
                } else {
                        res = &List{Elements{args}}
                }
        }

        if err != nil {
                //fmt.Printf("%v: %v\n", p.p, err)
        } else if res == nil {
                res = universalnone
        }
        return
}

func (p *delegate) disclose() (res Value, err error) {
        var ( o = p.o; v Value; changed bool )
        if v, err = o.expend(expendClosure); err != nil { return }
        if v != nil {
                if o, _ = v.(Object); o != nil {
                        changed = true
                } else {
                        err = fmt.Errorf("invalid delegate %v", v)
                        return
                }
        }

        var args []Value
        for _, a := range p.a {
                if v, err = a.expend(expendClosure); err != nil { return }
                if v != nil { a, changed = v, true }
                args = append(args, a)
        }
        if err == nil {
                if changed {
                        res = &delegate{closuredelegate{ p.p, p.l, o, args }}
                } else {
                        res = p
                }
        }
        return
}

func (p *delegate) refs(v Value) bool {
        if p.o == v || p.o.refs(v) {
                return true
        }
        for _, a := range p.a {
                if a.refs(v) {
                        return true
                }
        }
        return false
}

func (p *delegate) closured() bool {
        if p.o.closured() { return true }
        for _, a := range p.a {
                if a.closured() { return true }
        }
        return false
}

func (p *delegate) compare(c *comparer) (err error) {
        if trace_compare {
                fmt.Printf("compare:delegate: %v (%v %T)\n", p, c.target, c.target)
        }
        var v Value
        if v, err = p.expend(expendDelegate); err == nil {
                err = c.compare(v)
        }
        return
}

func (p *delegate) filedependcompare(c *comparer, d *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:delegate:File: %v (%v %T)\n", p, c.target, c.target)
        }
        var value Value
        if value, err = p.expend(expendDelegate); err != nil { return }
        if comp, _ := value.(filedepend); comp != nil {
                err = comp.filedependcompare(c, d)
        } else {
                err = fmt.Errorf("delegate: incomparable target (%T %v)", value, value)
                if trace_compare {
                        fmt.Printf("compare:delegate:File: %v (incomparable: %v %T)\n", p, value, value)
                }
        }
        return
}

func (p *delegate) pathdependcompare(c *comparer, d *Path) (err error) {
        if trace_compare {
                fmt.Printf("compare:delegate:Path: %v (%v %T)\n", p, c.target, c.target)
        }
        var value Value
        if value, err = p.expend(expendDelegate); err != nil { return }
        if comp, _ := value.(pathdepend); comp != nil {
                err = comp.pathdependcompare(c, d)
        } else {
                err = fmt.Errorf("delegate: incomparable target (%T %v)", value, value)
                if trace_compare {
                        fmt.Printf("compare:delegate:Path: %v (incomparable: %v %T)\n", p, value, value)
                }
        }
        return
}

func (p *delegate) prepare(pc *preparer) (err error) {
        if trace_prepare {
                fmt.Printf("prepare:delegate: %v\n", p)
        }
        var val Value
        if val, err = p.expend(expendDelegate); err != nil { return }
        for _, d := range merge(val) {
                if err = pc.update(d); err != nil { break }
        }
        return
}

type closure struct { closuredelegate }
func (p *closure) Type() Type { return ClosureType }
func (p *closure) String() (s string) { return p.string("&") }
func (p *closure) Integer() (int64, error) {
        if p.o == nil {
                return 0, nil
        }
        return p.o.Integer()
}
func (p *closure) Float() (float64, error) {
        if p.o == nil {
                return 0, nil
        }
        return p.o.Float()
}
func (p *closure) Strval() (s string, err error) {
        var v Value

        // &(...) -> $(...)
        if v, err = p.expend(expendClosure); err != nil {
                return
        } else if v == nil {
                //err = fmt.Errorf("{closure %+v &<nil>}", p.o)
                return
        }

        // $(...) -> .....
        if v, err = v.expend(expendDelegate); err != nil {
                return
        } else if v != nil {
                s, err = v.Strval()
        } else {
                //err = fmt.Errorf("{closure %+v $<nil>}", p.o)
        }
        return
}
func (p *closure) expend(w expendwhat) (res Value, err error) {
        switch {
        case w&expendClosure != 0:
                if res, err = p.disclose(); err != nil {
                        return
                }
                if res != nil && w&expendDelegate != 0 {
                        res, err = res.expend(expendDelegate)
                }
        case w&expendDelegate != 0:
                if res, err = p.reveal(); err != nil {
                        return
                }
                if res != nil && w&expendClosure != 0 {
                        res, err = res.expend(expendClosure)
                }
        }
        if err == nil && res == nil { res = p }
        return
}
func (p *closure) reveal() (res Value, err error) {
        if p.o == nil { return }

        var ( t Value; o Object )
        if t, err = p.o.expend(expendDelegate); err != nil { return }
        if t != nil {
                if o, _ = t.(Object); o == nil {
                        err = fmt.Errorf("closure of non-object (%T)", t)
                        return
                }
        }
        
        var ( a []Value; num int )
        for _, v := range p.a {
                if t, err = v.expend(expendDelegate); err != nil { return }
                if t == nil { t = v } else { num = num + 1 }
                a = append(a, t)
        }

        if o != nil || num > 1 {
                res = &closure{closuredelegate{ p.p, p.l, o, a }}
        }
        return
}
func (p *closure) disclose() (res Value, err error) {
        if p.o == nil { return nil, nil }

        var ( o Object; v Value; changed bool )
        SeeL: switch name := p.o.Name(); p.l {
        case token.LPAREN, token.ILLEGAL:
                for _, scope := range cloctx {
                        if scope != scope.project.scope {
                                // inquire non-project scope first
                                if _, o = scope.Find(name); o != nil {
                                        changed = true; break SeeL
                                }
                        }
                        if o, err = scope.project.resolveObject(name); err != nil {
                                return
                        } else if o != nil {
                                changed = true; break SeeL
                        }
                }
        case token.LBRACE, token.STRING, token.COMPOUND:
                for _, scope := range cloctx {
                        if o, err = scope.project.resolveEntry(name); err != nil {
                                return
                        } else if o != nil {
                                if p.l == token.LBRACE {
                                        changed = true; break SeeL
                                } else {
                                        // &'xxx' and &"xxx" are not disclosed into
                                        // the resolved objects instead of converting
                                        // into delegates.
                                        res = o; return
                                }
                        }
                }
        default:
                err = fmt.Errorf("unknown closure `&%+v%+v`", p.l, name)
                return
        }

        if o == nil { o = p.o } // assert changed == false

        // Disclose the object, which may contain closures.
        if v, err = o.expend(expendClosure); err != nil {
                return
        } else if v != nil {
                var ok bool
                if o, ok = v.(Object); ok && o != nil {
                        changed = true
                } else {
                        err = fmt.Errorf("invalid closure %+v", v)
                        return
                }
        }

        var args []Value
        for _, a := range p.a {
                if v, err = a.expend(expendClosure); err != nil { return }
                if v != nil { a, changed = v, true }
                args = append(args, a)
        }

        if changed && err == nil {
                res = &delegate{closuredelegate{ p.p, p.l, o, args }}
        }
        return
}

func (p *closure) refs(v Value) bool {
        if p.o == v {
                return true
        }
        for _, a := range p.a {
                if a.refs(v) {
                        return true
                }
        }
        return false
}

func (p *closure) closured() bool { return true }

func (p *closure) prepare(pc *preparer) (err error) {
        if trace_prepare {
                fmt.Printf("prepare:closure: %v\n", p)

        }
        if v, e := p.expend(expendClosure); e != nil {
                err = e
        } else if v == nil {
                err = fmt.Errorf("undefined closure target `%v`", p.o.Name())
                fmt.Fprintf(os.Stderr, "%s: %v\n", p.p, err)
        } else {
                //fmt.Fprintf(os.Stderr, "%s: %T %+v\n", p.p, v, v)
                err = pc.update(v)
        }
        return
}

type selection struct {
        t token.Token
        o Value // Object or selection
        s Value
}

func (p *selection) Type() Type { return SelectionType }
func (p *selection) String() string {
        return fmt.Sprintf("%v%s%v", p.o, p.t, p.s)
}

func (p *selection) object() (o Object, err error) {
        if s, ok := p.o.(*selection); ok {
                var v Value
                if v, err = s.value(); err != nil {
                        // sth's wrong!
                } else if o, _ = v.(Object); o == nil {
                        err = fmt.Errorf("selection.object: `%s` is nil", s.String())
                }
        } else if o, ok = p.o.(Object); !ok {
                err = fmt.Errorf("selection.object: `%v` is not object but `%T`", p.o, p.o)
        }
        return
}

func (p *selection) value() (v Value, err error) {
        var o Object
        if p.s == nil {
                err = fmt.Errorf("selection.value: nil prop `%s`", p.String())
        } else if o, err = p.object(); err != nil {
                // sth's wrong!
        } else if s := ""; o != nil {
                if s, err = p.s.Strval(); err == nil {
                        if pn, ok := o.(*ProjectName); ok && p.t == token.SELECT_PROG {
                                var entry *RuleEntry
                                if entry, err = pn.project.resolveEntry(s); err != nil {
                                        return
                                } else if entry == nil {
                                        err = fmt.Errorf("selection.value: no entry `%s` (%+v)", s, p.String())
                                } else {
                                        v = entry
                                }
                        } else if v, err = o.Get(s); err != nil {
                                //fmt.Printf("selection: %v: %v\n", p, err)
                        }
                }
        } else /*if o == nil*/ {
                err = fmt.Errorf("selection.value: nil object `%s`", p.String())
        }
        return
}

func (p *selection) Strval() (s string, err error) {
        var v Value
        if v, err = p.value(); err != nil {
                // sth's wrong
        } else if v != nil {
                s, err = v.Strval()
        } else {
                err = fmt.Errorf("selection.strval: `%s` is nil", p.String())
        }
        return
}

func (p *selection) Integer() (int64, error) {
        if s, err := p.Strval(); err == nil {
                return strconv.ParseInt(s, 10, 64)
        } else {
                return 0, err
        }
}
func (p *selection) Float() (float64, error) {
        if s, err := p.Strval(); err == nil {
                return strconv.ParseFloat(s, 64)
        } else {
                return 0, err
        }
}

func (p *selection) refs(v Value) bool { return p.o.refs(v) || p.s.refs(v) }
func (p *selection) closured() bool { return p.o.closured() || p.s.closured() }
func (p *selection) expend(w expendwhat) (res Value, err error) {
        var o, s Value
        if p.o != nil {
                if o, err = p.o.expend(w); err != nil {
                        return
                } else if o == nil { o = p.o }
        }
        if p.s != nil {
                if s, err = p.s.expend(w); err != nil {
                        return
                } else if s == nil { s = p.s }
        }
        if o != p.o || s != p.s {
                res = &selection{ p.t, o, s }
        } else {
                res = p
        }
        return
}

func (p *selection) prepare(pc *preparer) (err error) {
        var v Value
        if v, err = p.value(); err != nil {
                // sth's wrong
        } else if v == nil {
                err = fmt.Errorf("`%v` is nil", p)
        } else {
                err = pc.update(v)
        }
        return
}

// Pattern
type Pattern interface {
        Value
        concrete(patent *RuleEntry, stem string) (entry *RuleEntry, err error)
        match(s string) (matched bool, stem string, err error)
}

type pattern struct {
}

func (p *pattern) Type() Type        { return PatternType }
func (p *pattern) Integer() (int64, error) { return 0, nil }
func (p *pattern) Float() (float64, error) { return 0, nil }
func (p *pattern) concrete(patent *RuleEntry, target, stem string) (entry *RuleEntry, err error) {
        entry = new(RuleEntry); *entry = *patent
        if proj := patent.OwnerProject(); proj.isFile(/*filepath.Base(target)*/target) {
                if file := proj.SearchFile(target); file != nil {
                        entry.target = file
                }
        } else {
                entry.target = &String{ target }
        }
        return
}

// GlobPattern represents glob expressions (e.g. '%.o', '[a-z].o', 'a?a.o')
// FIXME: PercPattern -> %.o
//        GlobPattern -> [a-z].o a?a.o
type GlobPattern struct {
        pattern
        Prefix Value
        Suffix Value
}
func (p *GlobPattern) expend(_ expendwhat) (Value, error) { return p, nil }
func (p *GlobPattern) String() string {
        /*if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{GlobPattern '%s' !(%+v)}", s, e)
        }*/
        return fmt.Sprintf("%s%%%s", p.Prefix.String(), p.Suffix.String())
}
func (p *GlobPattern) Strval() (s string, err error) {
        if p.Prefix != nil {
                var v string
                if v, err = p.Prefix.Strval(); err == nil {
                        s = v
                } else {
                        return
                }
        }
        s += "%"
        if p.Suffix != nil {
                var v string
                if v, err = p.Suffix.Strval(); err == nil {
                        s += v
                } else {
                        return
                }
        }
        return
}
func (p *GlobPattern) match(s string) (matched bool, stem string, err error) {
        var prefix, suffix string
        if prefix, err = p.Prefix.Strval(); err == nil && strings.HasPrefix(s, prefix) {
                if suffix, err = p.Suffix.Strval(); err == nil && strings.HasSuffix(s, suffix) {
                        if a, b := len(prefix), len(s)-len(suffix); a < b {
                                matched, stem = true, s[a:b]
                        }
                }
        }
        return
}

func (p *GlobPattern) MakeString(stem string) (s string, err error) {
        if s, err = p.Prefix.Strval(); err == nil {
                var v string
                if v, err = p.Suffix.Strval(); err == nil {
                        s += stem + v
                }
        }
        return
}

func (p *GlobPattern) concrete(patent *RuleEntry, stem string) (entry *RuleEntry, err error) {
        var target string
        if target, err = p.MakeString(stem); err == nil {
                entry = &RuleEntry{
                        patent.class, &String{ target },
                        patent.programs, patent.Position,
                }
                return
        }
        return
}

func (p *GlobPattern) refs(v Value) bool { return p.Prefix.refs(v) || p.Suffix.refs(v) }
func (p *GlobPattern) closured() bool { return p.Prefix.closured() || p.Suffix.closured() }

func (p *GlobPattern) prepare(pc *preparer) (err error) {
        if trace_prepare {
                fmt.Printf("prepare:GlobPattern: %v(%v)\n", p, pc.stem)
        }
        if pc.stem == "" {
                err = fmt.Errorf("empty stem (%s)", p)
                return
        }

        var target string
        if target, err = p.MakeString(pc.stem); err != nil { return }

        // Check if target is a file (if source entry is file).
        for i := len(execstack)-1; i >= 0; i -= 1 {
                prog := execstack[i]
                if file := prog.project.file(target); file == nil {
                        continue
                } else if file.exists() {
                        if trace_prepare {
                                fmt.Printf("prepare:GlobPattern: %v(%v) (file %v in %s)\n", p, pc.stem, file, prog.project.name)
                        }
                        // File exists, but we still prepare it to call
                        // it's dependencies if any.
                        err = file.prepare(pc)
                        return
                }

                // Try update the file target via rules if not found.
                if err = prog.project.updateTarget(pc, target); err == nil {
                        continue
                } else {
                        return
                }
        }

        if trace_prepare {
                fmt.Printf("prepare:GlobPattern: %v(%v) (target %v)\n", p, pc.stem, target)
        }
        if err = pc.updateTarget(target); err == nil {
                return // Good!
        } else {
                if trace_prepare {
                        fmt.Printf("prepare:GlobPattern: %v (error: %v) (%v)\n", p, err, pc.stem)
                }
                err = patternPrepareError(err)
        }
        return
}

func MakeGlobPattern(prefix, suffix Value) Pattern {
        if prefix == nil { prefix = universalnone }
        if suffix == nil { suffix = universalnone }
        return &GlobPattern{
                Prefix: prefix,
                Suffix: suffix,
        }
}

// TODO: implement regexp pattern
type RegexpPattern struct {
        pattern
}

func NewRegexpPattern() Pattern {
        return &RegexpPattern{}
}

func (p *RegexpPattern) expend(_ expendwhat) (Value, error) { return p, nil }

func (p *RegexpPattern) String() string { return "{RegexpPattern}" }
func (p *RegexpPattern) Strval() (s string, err error) { return "", nil }
func (p *RegexpPattern) match(s string) (matched bool, stem string, err error) {
        panic("TODO: regexp matching...")
        return
}
func (p *RegexpPattern) concrete(patent *RuleEntry, stem string) (entry *RuleEntry, err error) {
        panic("TODO: creating new match entry")
        return
}

func (p *RegexpPattern) closured() bool { return false }
func (p *RegexpPattern) refs(_ Value) bool { return false }

type Valuer interface {
        Value() Value
}

type Caller interface {
        Call(pos token.Position, args... Value) (Value, error)
}

type Executer interface {
        Execute(pos token.Position, a... Value) (result []Value, err error)
}

type Positioner interface {
        Position() token.Position
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
        pos token.Position
}

// Position() returns the position of the value occurs position in file or nil.
func (p *positional) Position() token.Position { return p.pos }

// Positional wraps a value with a valid position
func Positional(v Value, pos token.Position) Positioner {
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

// Reveal reveals delegated component and Valuer recursively.
/*func Reveal(value Value) (res Value, err error) {
        if value == nil {
                err = fmt.Errorf("reveal nil value")
        } else if res, err = value.expend(expendDelegate); res == nil && err == nil {
                res = value
        }
        return
}*/

func RevealAll(values ...Value) (res []Value, err error) {
        for _, v := range values {
                //if v, err = Reveal(v); err != nil { break }
                if v, err = v.expend(expendDelegate); err != nil { break }
                if v != nil { res = append(res, v) }
        }
        return
}

// Disclose expends closures to normal value recursively.
/*func Disclose(value Value) (res Value, err error) {
        if false {
                fmt.Printf("Disclose: %T %v\n", value, value)
        }
        if value == nil {
                err = fmt.Errorf("disclose nil value")
        } else if res, err = value.expend(expendClosure); res == nil && err == nil {
                res = value
        }
        return
}*/

func DiscloseAll(values ...Value) (res []Value, err error) {
        for _, v := range values {
                //if v, err = Disclose(v); err != nil { break }
                if v, err = v.expend(expendClosure); err != nil { break }
                if v != nil { res = append(res, v) }
        }
        return
}

// Merge combines lists recursively into one list. Previously called Join.
func merge(args... Value) (elems []Value) {
        for _, arg := range args {
                if l, _ := arg.(*List); l != nil {
                        elems = append(elems, merge(l.Elems...)...)
                } else {
                        elems = append(elems, arg)
                }
        }
        return
}

func mergeresult(res []Value, err error) ([]Value, error) {
        if err == nil { res = merge(res...) }
        return res, err
}

/*func Expend(value Value) (res Value, err error) {
        // Performs: &(...) -> $(...)
        if value, err = value.expend(expendClosure); err == nil {
                // Performs: $(...) -> ...
                value, err = value.expend(expendDelegate)
        }
        return
}*/

func expendall(w expendwhat, values ...Value) (res []Value, num int, err error) {
        var v Value
        for _, elem := range values {
                if elem == nil {
                        panic(fmt.Sprintf("nil in %v\n", values))
                }
                if v, err = elem.expend(w); err == nil {
                        if v != elem { num += 1 }
                        res = append(res, v)
                } else {
                        break //res = append(res, elem)
                }
        }
        return
}

func ExpendAll(values ...Value) (res []Value, err error) {
        if res, _, err = expendall(expendBoth, values...); err == nil {
                // second expend to ensure having real value
                res, _, err = expendall(expendBoth, res...)
        }
        return
}

func MakeDelegate(pos token.Position, tok token.Token, obj Object, args... Value) Value {
        return &delegate{closuredelegate{ pos, tok, obj, args }}
}

func MakeClosure(pos token.Position, tok token.Token, obj Object, args... Value) Value {
        if obj == nil { panic("closure of nil") }
        return &closure{closuredelegate{ pos, tok, obj, args }}
}

func Refs(a Value, v Value) bool { return a.refs(v) }

func MakeListOrScalar(elems []Value) (res Value) {
        if x := len(elems); x > 1 {
                res = &List{Elements{elems}}
        } else if x == 1 {
                res = elems[0]
        } else {
                res = universalnone
        }
        return
}

func Scalar(v Value, t Type) (res Value) {
        if v.Type() == t {
                res = v
        } else if l, o := v.(*List); l != nil && o && l.Len() > 0 {
                res = Scalar(l.Elems[0], t)
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

func ParseLiteral(tok token.Token, s string) (v Value) {
        switch tok {
        default:             v = universalnone
        case token.BAR:      v = modifierbar
        case token.BIN:      v = ParseBin(s)
        case token.OCT:      v = ParseOct(s)
        case token.INT:      v = ParseInt(s)
        case token.HEX:      v = ParseHex(s)
        case token.FLOAT:    v = ParseFloat(s)
        case token.DATETIME: v = ParseDateTime(s)
        case token.DATE:     v = ParseDate(s)
        case token.TIME:     v = ParseTime(s)
        case token.URI:      v = ParseUri(s)
        case token.BAREWORD: v = MakeBareword(s)
        case token.STRING:   v = MakeString(s)
        case token.ESCAPE:   v = MakeString(EscapeChar(s))
        }
        return
}

func Make(in interface{}) (out Value) {
        switch v := in.(type) {
        case int:       out = MakeInt(int64(v))
        case int32:     out = MakeInt(int64(v))
        case int64:     out = MakeInt(v)
        case float32:   out = MakeFloat(float64(v))
        case float64:   out = MakeFloat(v)
        case string:    out = MakeString(v)
        case time.Time: out = MakeDateTime(v) // FIXME: NewDate, NewTime
        case Value:     out = v
        default:        out = universalnone
        }
        return
}

func MakeAll(in... interface{}) (out []Value) {
        for _, v := range in {
                out = append(out, Make(v))
        }
        return
}
