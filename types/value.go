//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import (
        "github.com/duzy/smart/token"
        "path/filepath"
        "regexp"
        "time"
        "net/url"
        "reflect"
        "strconv"
        "strings"
        "bytes"
        "fmt"
        "os"
        "io"
)

const (
        trace_compare = false
        trace_prepare = false
        trace_workdir = true && trace_prepare
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

        // disclose method, also prevents creating new Value type from
        // other packages.
        disclose(scope *Scope) (Value, error)

        // Recursively detecting whether this value is referencing
        // to the object (to avoid loop-delegation).
        referencing(o Object) bool
}

type ClosureContext []*Scope

func (cc ClosureContext) disclose(value Value) (res Value, err error) {
        for _, scope := range cc {
                if res, err = value.disclose(scope); res != nil && err == nil {
                        break
                }
                //fmt.Printf("disclose:%v: %v %v %v %v\n", i, value, res, err, scope)
        }
        return
}

func (cc *ClosureContext) Join(scope *Scope) bool {
        for _, s := range *cc {
                if scope == s { return false }
        }
        *cc = append(*cc, scope)
        return true
}

func (cc *ClosureContext) set(prev ClosureContext) { *cc = prev }
func (cc *ClosureContext) app(scopes... *Scope) ClosureContext {
        var prev = *cc
        for _, scope := range scopes {
                if scope != nil { cc.Join(scope) }
        }
        return prev
}

func NewClosureContext(scopes... *Scope) (cc ClosureContext) {
        for _, scope := range scopes {
                cc.Join(scope)
        }
        return
}

type value struct {}
func (*value) disclose(_ *Scope) (Value, error) { return nil, nil }
func (*value) referencing(_ Object) bool { return false }
func (*value) Type() Type { return InvalidType }
func (*value) String() string { return fmt.Sprintf("value{}") }
func (*value) Strval() (string, error) { return "", nil }
func (*value) Integer() (int64, error) { return 0, nil }
func (*value) Float() (float64, error) { return 0, nil }

type Comparer struct {
        //program *Program
        globe *Globe
        target comparable
        tarval Value
        objects []Value
        result []Value
}

// TODO: use it with (compare) modifier
type comparable interface {
        // Compare as a prerequisite with the target c.target.
        compare(c *Comparer) error

        // Compare as a target with the file prerequisite.
        compareFileDepend(c *Comparer, file *File) error

        // Compare as a target with the path prerequisite.
        comparePathDepend(c *Comparer, path *Path) error
}

func NewComparer(globe *Globe, target Value) (c *Comparer, err error) {
        if trace_compare {
                // fmt.Printf("compare:Target: %v (%T) (revealed: %v)\n", target, target, Reveal(target))
        }
        if target, err = Reveal(target); err != nil { return }
        if target == nil || target.Type() == NoneType {
                err = breakf(false, "comparing no target")
        } else if t, _ := target.(comparable); t != nil {
                c = &Comparer{ globe, t, target, nil, nil }
        } else {
                err = fmt.Errorf("incomparable target (%v)", target)
        }
        return
}

func (c *Comparer) Compare(value interface{}) (err error) {
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

func (c *Comparer) compare(value interface{}) (err error) {
        if p, _ := value.(comparable); p != nil {
                err = p.compare(c)
        } else {
                err = fmt.Errorf("Type '%T' is not comparable.", value)
        }
        return
}

// Preparer prepares prerequisites of targets.
type Preparer struct {
        project *Project
        program *Program
        entry *RuleEntry // caller entry
        arguments []Value
        targets *List
        stem string // set by PatternStem
        //source string // source target creating the stemmed entry
        //file *File
}

type prerequisite interface {
        prepare(pc *Preparer) error
}

func (pc *Preparer) setProject(proj *Project) *Project {
        prev := pc.project
        pc.project = proj
        return prev
}

func (pc *Preparer) Prepare(value interface{}) (err error) {
        if v := reflect.ValueOf(value); v.Kind() == reflect.Slice {
                for i := 0; i < v.Len(); i++ {
                        if err = pc.prepare(v.Index(i).Interface()); err == nil {
                                // Good!
                        } else if ute, ok := err.(unknownTargetError); ok {
                                if trace_prepare {
                                        fmt.Printf("prepare:unknown target %v (%v)\n", ute.target, pc.entry)
                                }
                                break
                        } else if ufe, ok := err.(unknownFileError); ok {
                                if trace_prepare {
                                        fmt.Printf("prepare:unknown file %v (%v)\n", ufe.file, pc.entry)
                                }
                                break
                        } else {
                                break
                        }
                }
        } else {
                err = pc.prepare(value)
        }
        return
}

func (pc *Preparer) prepare(value interface{}) (err error) {
        if p, _ := value.(prerequisite); p != nil {
                err = p.prepare(pc)
        } else {
                err = fmt.Errorf("Type '%T' is not prerequisite.", value)
        }
        return
}

func (pc *Preparer) forEachExternalCaller(f func (*Project) (bool, error)) (err error, brk bool) {
        var triedm = map[*Project]bool{ pc.program.project:true }
        if brk, err = f(pc.program.project); brk {
                return
        }
        for caller := pc.entry.caller; caller != nil; caller = caller.entry.caller {
                // Find the last caller.
                //if caller == pc || caller == pc.entry.caller { continue }
                //if tried, has := triedm[caller.entry.owner]; caller != pc.entry.caller && !(has&&tried) {
                if tried, has := triedm[caller.entry.owner]; !(has&&tried) {
                        if brk, err = f(caller.entry.owner); brk {
                                break
                        } else {
                                triedm[caller.entry.owner] = true
                        }
                }
        }
        return
}

func NewPreparer(prog *Program, entry *RuleEntry, args... Value) (pc *Preparer) {
        var stem string
        if entry.caller != nil {
                stem = entry.caller.stem
        }
        return &Preparer{ prog.project, prog, entry, args, new(List), stem }
}

type Argumented struct {
        Value
        Args []Value
}
//func (p *Argumented) Type() Type { return ArgumentedType }
func (p *Argumented) String() (s string) {
        s = p.Value.String()
        s += "("
        for i, a := range p.Args {
                if i > 0 {
                        s += ","
                }
                s += a.String()
        }
        s += ")"
        return
}
func (p *Argumented) Strval() (s string, err error) {
        if s, err = p.Value.Strval(); err != nil {
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

func (p *Argumented) prepare(pc *Preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:Argumented: %v (%v)\n", p, pc.entry)
        }
        pc.arguments = p.Args // TODO: merge args with p.Args ??
        return pc.Prepare(p.Value)
}

type None struct { value }
func (p *None) Type() Type { return NoneType }
func (p *None) compare(c *Comparer) (err error) { return }
func (p *None) compareFileDepend(c *Comparer, file *File) error { return nil }
func (p *None) comparePathDepend(c *Comparer, path *Path) error { return nil }
func (p *None) prepare(pc *Preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:None: (%v)\n", pc.entry)
        }
        return nil 
}

type Any struct {
        Value interface{}
        value
}
func (p *Any) Type() Type { return AnyType }

type integer struct {
        Value int64
}
func (p *integer) disclose(_ *Scope) (Value, error) { return nil, nil }
func (p *integer) referencing(_ Object) bool { return false }
func (p *integer) Integer() (int64, error) { return p.Value, nil }
func (p *integer) Float() (float64, error) { return float64(p.Value), nil }

type Bin struct { integer }
func (p *Bin) Type() Type          { return BinType }
func (p *Bin) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("Bin{%s}!(%s)", p.Value, e)
        }
}
func (p *Bin) Strval() (string, error) { return strconv.FormatInt(int64(p.Value),2), nil }

type Oct struct { integer }
func (p *Oct) Type() Type          { return OctType }
func (p *Oct) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("Oct{%s}!(%s)", p.Value, e)
        }
}
func (p *Oct) Strval() (string, error) { return strconv.FormatInt(int64(p.Value),8), nil }

type Int struct { integer }
func (p *Int) Type() Type          { return IntType }
func (p *Int) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("Int{%s}!(%s)", p.Value, e)
        }
}
func (p *Int) Strval() (string, error) { return strconv.FormatInt(int64(p.Value),10), nil }

type Hex struct { integer }
func (p *Hex) Type() Type          { return HexType }
func (p *Hex) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("Hex{%s}!(%s)", p.Value, e)
        }
}
func (p *Hex) Strval() (string, error) {
        return strconv.FormatInt(int64(p.Value),16), nil 
}

type Float struct {
        Value float64
}
func (p *Float) disclose(_ *Scope) (Value, error) { return nil, nil }
func (p *Float) referencing(_ Object) bool { return false }
func (p *Float) Type() Type        { return FloatType }
func (p *Float) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("Float{%v}!(%s)", p.Value, e)
        }
}
func (p *Float) Strval() (string, error) {
        return strconv.FormatFloat(float64(p.Value),'g', -1, 64), nil 
}
func (p *Float) Integer() (int64, error) { return int64(p.Value), nil }
func (p *Float) Float() (float64, error) { return p.Value, nil }

type DateTime struct {
        Value time.Time 
}
func (*DateTime) disclose(_ *Scope) (Value, error) { return nil, nil }
func (*DateTime) referencing(_ Object) bool { return false }
func (p *DateTime) Type() Type     { return DateTimeType }
func (p *DateTime) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("DateTime{%v}!(%s)", p.Value, e)
        }
}
func (p *DateTime) Strval() (string, error) { return time.Time(p.Value).Format("2006-01-02T15:04:05.999999999Z07:00"), nil } // time.RFC3339Nano
func (p *DateTime) Integer() (int64, error) { return p.Value.Unix(), nil }
func (p *DateTime) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

type Date struct { DateTime }
func (*Date) disclose(_ *Scope) (Value, error) { return nil, nil }
func (*Date) referencing(_ Object) bool { return false }
func (p *Date) Type() Type         { return DateType }
func (p *Date) String() string     {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("Date{%v}!(%s)", p.Value, e)
        }
}
func (p *Date) Strval() (string, error) { return time.Time(p.Value).Format("2006-01-02"), nil }
func (p *Date) Integer() (int64, error) { return p.Value.Unix(), nil }
func (p *Date) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

type Time struct { DateTime }
func (*Time) disclose(_ *Scope) (Value, error) { return nil, nil }
func (*Time) referencing(_ Object) bool { return false }
func (p *Time) Type() Type { return TimeType }
func (p *Time) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("Time{%v}!(%s)", p.Value, e)
        }
}
func (p *Time) Strval() (string, error) { return time.Time(p.Value).Format("15:04:05.999999999Z07:00"), nil }
func (p *Time) Integer() (int64, error) { return p.Value.Unix(), nil }
func (p *Time) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

type Uri struct {
        Value *url.URL
}
func (*Uri) disclose(_ *Scope) (Value, error) { return nil, nil }
func (*Uri) referencing(_ Object) bool { return false }
func (p *Uri) Type() Type { return UriType }
func (p *Uri) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("Uri{%v}!(%s)", p.Value, e)
        }
}
func (p *Uri) Strval() (string, error) { return p.Value.String(), nil }
func (p *Uri) Integer() (int64, error) { return int64(len(p.Value.String())), nil }
func (p *Uri) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

type String struct {
        Value string
}
func (*String) disclose(_ *Scope) (Value, error) { return nil, nil }
func (*String) referencing(_ Object) bool { return false }
func (p *String) Type() Type  { return StringType }
func (p *String) String() string {
        if strings.ContainsRune(p.Value, '\n') {
                return "\"" + strings.Replace(p.Value, "\n", "\\n", -1) + "\"" 
        } else {
                return "'" + p.Value + "'" 
        }
}
func (p *String) Strval() (string, error) { return p.Value, nil }
func (p *String) Integer() (int64, error) { return strconv.ParseInt(p.Value, 10, 64) }
func (p *String) Float() (float64, error) { return strconv.ParseFloat(p.Value, 64) }

func (p *String) prepare(pc *Preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:String: %v (%v)\n", p, pc.entry)
        }
        //pc.source = p.Value
        return pc.prepareTarget(p.Value)
}

type Bareword struct {
        Value string
}
func (_ *Bareword) disclose(_ *Scope) (Value, error) { return nil, nil }
func (_ *Bareword) referencing(_ Object) bool { return false }
func (p *Bareword) Type() Type     { return BarewordType }
func (p *Bareword) String() string { return fmt.Sprintf("Bareword{%s}", p.Value) }
func (p *Bareword) Strval() (string, error) { return p.Value, nil }
func (p *Bareword) Integer() (int64, error) { return strconv.ParseInt(p.Value, 10, 64) }
func (p *Bareword) Float() (float64, error) { return strconv.ParseFloat(p.Value, 64) }

func (p *Bareword) prepare(pc *Preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:Bareword: %v (%v)\n", p, pc.entry)
        }
        //pc.source = p.Value
        return pc.prepareTarget(p.Value)
}

func (pc *Preparer) prepareTargetValue(value Value) (err error) {
        var ( v Value; s string )
        if v, err = pc.program.cc.disclose(value); err != nil { return }
        if v == nil { v = value }
        if s, err = v.Strval(); err != nil { return }
        return pc.prepareTarget(s)
}

// patternPrepareError indicates an error occurred in preparing a pattern.
type patternPrepareError error
type unknownTargetError struct {
        error
        target string
}
type unknownFileError struct {
        error
        file *File
}

func (pc *Preparer) prepareTarget(target string) error {
        if err, brk := pc.explicitTarget(target); err != nil || brk {
                return err
        }
        if err, brk := pc.implicitTarget(target); err != nil || brk {
                return err
        }
        return unknownTargetError{
                fmt.Errorf("unknown target '%v'", target), 
                target,
        }
}

func (pc *Preparer) explicitTarget(target string) (error, bool) {
        return pc.forEachExternalCaller(func(project *Project) (trybrk bool, err error) {
                if trace_prepare {
                        fmt.Printf("prepare:Target: %v (project %s) (%v)\n", target, project.name, pc.entry)
                }
                if _, obj := project.scope.Find(target); obj != nil {
                        if trace_prepare {
                                fmt.Printf("prepare:Target: %v (found %v) (%v -> %v)\n", target, obj, project.name, pc.entry)
                        }
                        defer pc.setProject(pc.setProject(project))
                        err, trybrk = pc.Prepare(obj), true
                }
                return
        })
}

func (pc *Preparer) implicitTarget(target string) (error, bool) {
        return pc.forEachExternalCaller(func(project *Project) (trybrk bool, err error) {
                if trace_prepare {
                        fmt.Printf("prepare:Target: %v (project %s) (%v)\n", target, project.name, pc.entry)
                }

                defer pc.setProject(pc.setProject(project))

                var pss []*PatternStem
                if pss, err = project.FindPatterns(target); err != nil {
                        return
                }

                for _, ps := range pss {
                        if trace_prepare {
                                fmt.Printf("prepare:Target: %v (stemmed %v) (%v -> %v)\n", target, ps, project.name, pc.entry)
                        }
                        ps.source = target // Bounds PatternStem with the source.
                        if err = ps.prepare(pc); err == nil {
                                trybrk = true; break // Updated successfully!
                        } else if _, ok := err.(patternPrepareError); ok {
                                if trace_prepare {
                                        fmt.Printf("prepare:Target: %v (error: %s)\n", target, err)
                                }
                                // Discard pattern unfit errors and caller stack.
                        } else {
                                trybrk = true; break // Update failed!
                        }
                }
                return
        })
}

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

func (p *Elements) disclose(scope *Scope) (elems []Value, num int, err error) {
        var v Value
        for _, elem := range p.Elems {
                if elem == nil { continue }
                if v, err = elem.disclose(scope); err != nil { return }
                if v != nil {
                        elem = v
                        num += 1
                }
                elems = append(elems, elem)
        }
        return
}

func (p *Elements) referencing(o Object) bool {
        for _, elem := range p.Elems {
                if elem != nil && elem.referencing(o) {
                        return true
                }
        }
        return false 
}

type Barecomp struct {
        Elements
}
func (p *Barecomp) Type() Type { return BarecompType }
func (p *Barecomp) String() (s string) {
        for _, e := range p.Elems {
                switch t := e.(type) {
                case *String: s += t.Value
                default: s += t.String()
                }
        }
        return fmt.Sprintf("Barecomp{%s}", s)
}
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

func (p *Barecomp) disclose(scope *Scope) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = p.Elements.disclose(scope); err == nil && num > 0 {
                res = &Barecomp{ Elements{ elems } }
        }
        return
}

func (p *Barecomp) prepare(pc *Preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:Barecomp: %v (%v)\n", p, pc.entry)
                for _, elem := range p.Elems {
                        fmt.Printf("prepare:Barecomp: %v (%v) (%v)\n", p, elem, pc.entry)
                }
        }
        return pc.prepareTargetValue(p)
}

type Barefile struct {
        Name Value
        File *File
}
func (p *Barefile) Type() Type { return BarefileType }
func (p *Barefile) String() string { return fmt.Sprintf("Barefile{%s}", p.Name.String()) }
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
func (p *Barefile) disclose(scope *Scope) (res Value, err error) {
        var name Value
        if name, err = p.Name.disclose(scope); err != nil {
                return
        } else if name != nil {
                res = &Barefile{ name, p.File }
        }
        return
}
func (p *Barefile) referencing(o Object) bool {
        return p.Name.referencing(o)
}

func (p *Barefile) compare(c *Comparer) (err error) {
        if trace_compare {
                fmt.Printf("compare:Barefile: %v (%v %T)\n", p.Name, c.target, c.target)
        }
        if p.File != nil {
                err = p.File.compare(c)
        } else {
                err = breakf(false, "no such file '%v'", p)
        }
        return
}

func (p *Barefile) compareFileDepend(c *Comparer, d *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:Barefile:File: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }
        if p.File != nil {
                err = p.File.compareFileDepend(c, d)
        } else {
                err = breakf(false, "no such file '%v'", p)
        }
        return
}

func (p *Barefile) comparePathDepend(c *Comparer, d *Path) (err error) {
        if trace_compare {
                fmt.Printf("compare:Barefile:Path: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }
        if p.File != nil {
                err = p.File.comparePathDepend(c, d)
        } else {
                err = breakf(false, "no such path '%v'", p)
        }
        return
}

func (p *Barefile) prepare(pc *Preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:Barefile: %v (%v -> %v)\n", p, pc.entry.owner.name, pc.entry)
        }
        if p.File != nil {
                if s, e := p.Name.Strval(); e != nil {
                        return e
                } else if s != p.File.Name {
                        p.File.Name = s // Fix it in case of '$@.o' was parsed and became '.o'.
                }
                return p.File.prepare(pc)
        } else {
                return pc.prepareTargetValue(p)
        }
}

type Glob struct {
        Tok token.Token
}
func (p *Glob) Type() Type { return GlobType }
func (p *Glob) String() (s string) { return fmt.Sprintf("Glob{%s}", p.Tok.String()) }
func (p *Glob) Strval() (string, error) { return p.Tok.String(), nil }
func (p *Glob) Integer() (int64, error) { return 0, nil }
func (p *Glob) Float() (float64, error) { return 0, nil }
func (p *Glob) disclose(scope *Scope) (Value, error) { return nil, nil }
func (p *Glob) referencing(o Object) bool { return false }

type Path struct {
        Elements
        File *File // if this path is pointed to a file, ie. the last element matched a FileMap
}
func (p *Path) String() (s string) {
        // TODO: add '/' for root dir
        var sep = true
        for i, seg := range p.Elems {
                if i > 0 && sep {
                        s += string(os.PathSeparator) 
                }
                s += seg.String()
                if ps, ok := seg.(*PathSeg); ok && ps != nil && ps.Value == '/' {
                        sep = false
                } else {
                        sep = true
                }
        }
        // TODO: add '/' if there's such a suffix
        return fmt.Sprintf("Path{%s}", s)
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
func (p *Path) disclose(scope *Scope) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = p.Elements.disclose(scope); err != nil { return }
        if num > 0 { res = &Path{ Elements{ elems }, p.File } }
        return
}

/*func (p *Path) Dir() (s string) { // Same as `filepath.Dir(p.Strval())`.
        if n := len(p.Elems); n > 0 {
                s = filepath.Base(p.Elems[n-1].Strval())
        }
        return filepath.Dir(p.Strval())
}

func (p *Path) Base() (s string) { // Same as `filepath.Base(p.Strval())`.
        if n := len(p.Elems); n > 0 {
                s = filepath.Base(p.Elems[n-1].Strval())
        }
        return
}*/

func (p *Path) compare(c *Comparer) (err error) {
        if trace_compare {
                fmt.Printf("compare:Path: %v (%v %T)\n", p, c.target, c.target)
        }
        return c.target.comparePathDepend(c, p)
}

func (p *Path) compareFileDepend(c *Comparer, d *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:Path:File: %v (%v) (depends: %v %v) (%v %T)\n", p, p.File, d, d.Info, c.target, c.target)
        }
        if p.File != nil {
                return p.File.compareFileDepend(c, d)
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
                return breakf(false, "no such directory '%v'", ds)
        }

        if dt.After(tt) {
                c.globe.Timestamps[ts] = dt // Or tt?
                err = nil // Returns nil to request update.
        } else {
                err = breakf(true, "updated path '%s'", p)
        }
        return
}

func (p *Path) comparePathDepend(c *Comparer, d *Path) (err error) {
        if trace_compare {
                fmt.Printf("compare:Path:Path: %v (%v) (depends: %v) (%v %T)\n", p, p.File, d, c.target, c.target)
        }
        if p.File != nil {
                return p.File.comparePathDepend(c, d)
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
                return breakf(false, "no such directory '%v'", ds)
        }

        if dt.After(tt) {
                c.globe.Timestamps[ts] = dt // Or tt?
                err = nil // Returns nil to request update.
        } else {
                err = breakf(true, "updated path '%s'", p)
        }
        return
}

func (p *Path) prepare(pc *Preparer) (err error) {
        if trace_prepare {
                if p.File != nil {
                        fmt.Printf("prepare:Path: %v (file: %v) (%v, %v)\n", p, p.File, pc.program.project.name, pc.entry)
                } else {
                        fmt.Printf("prepare:Path: %v (%v, %v)\n", p, pc.program.project.name, pc.entry)
                }
        }
        if p.File == nil {
                var ( s, e = p.Strval(); name = filepath.Base(s) )
                if e != nil { err = e; return }
                err, _ = pc.forEachExternalCaller(func(project *Project) (trybrk bool, err error) {
                        if project.isFile(name) {
                                if file := project.SearchFile(s); file != nil {
                                        p.File, trybrk = file, file.IsExists() //IsKnown()
                                }
                        }
                        if trace_prepare && p.File != nil {
                                fmt.Printf("prepare:Path: %v (found file '%v' in %v) (%v, %v)\n", p, p.File, project.name, pc.program.project.name, pc.entry)
                        }
                        return
                })
        }

        if p.File != nil {
                return p.File.prepare(pc)
        } else if e := pc.prepareTargetValue(p); e == nil {
                // Good!
        } else if ute, ok := e.(unknownTargetError); ok {
                if info, _ := os.Stat(ute.target); info == nil {
                        pc.targets.Append(p) // Append unknown path anyway.
                        if trace_prepare {
                                fmt.Printf("prepare:Path: %v (unknown path: %v) (%v)\n", p, ute.target, pc.entry)
                        }
                } else if info.IsDir() {
                        pc.targets.Append(p)
                        if trace_prepare {
                                fmt.Printf("prepare:Path: %v (found unknown path: %v) (%v)\n", p, ute.target, pc.entry)
                        }
                } else {
                        // Search this path target as a file.
                        p.File = pc.program.project.SearchFile(ute.target)
                        pc.targets.Append(p.File)
                        if trace_prepare {
                                fmt.Printf("prepare:Path: %v (found unknown target: %v) (file: %v) (%v)\n", p, ute.target, p.File.Fullname(), pc.entry)
                        }
                }
        } else {
                err = e
        }
        return
}

type PathSeg struct {
        Value rune 
        value
}
func (p *PathSeg) Type() Type { return PathSegType }
func (p *PathSeg) String() string { 
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("PathSeg{%s}!(%s)", p.Value, e)
        }
}
func (p *PathSeg) Strval() (s string, e error) {
        switch p.Value {
        case '/': s = "/"
        case '.': s = "."
        case '^': s = ".."
        default: e = fmt.Errorf("unknown pathseg (%s)", p.Value)
        }
        return
}

type File struct {
        value            // satisify Value interface
        Name string      // represented name (e.g. relative filename)
        Match *FileMap   // matched pattern (see 'files' directive)
        Dir string       // full directory in which the file should be or was found
        Sub string       // sub directory containing the file (aka. Project.SearchFile)
        Info os.FileInfo // file info if exists
}
func (p *File) Type() Type { return FileType }

func (p *File) String() string {
        return fmt.Sprintf("File{%s,Exists=%v,Known=%v}", p.Fullname(), p.IsExists(), p.IsKnown())
}

// Strval returns the relative filename (aka. Project.SearchFile).
func (p *File) Strval() (string, error) { return filepath.Join(p.Sub, p.Name), nil }

func (p *File) Fullname() string { return filepath.Join(p.Dir, p.Name) }
func (p *File) Basename() string {
        if p.Info != nil {
                return p.Info.Name()
        } else {
                return filepath.Base(p.Name)
        }
}

func (p *File) IsKnown() bool { return p.Match != nil }
func (p *File) IsExists() bool { return p.Info != nil }

func (p *File) compare(c *Comparer) (err error) {
        if trace_compare {
                fmt.Printf("compare:File: %v (%v %T)\n", p.Name, c.target, c.target)
        }
        return c.target.compareFileDepend(c, p)
}

func (p *File) compareFileDepend(c *Comparer, d *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:File:File: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
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
        } else if d.Info != nil {
                dt = d.Info.ModTime()
        } else if info, _ := os.Stat(ds); info != nil {
                dt = info.ModTime()
        } else {
                return breakf(false, "no such directory '%v'", ds)
        }

        if dt.After(tt) {
                c.globe.Timestamps[ts] = dt // Or tt?
                err = nil // Returns nil to request update.
        } else {
                err = breakf(true, "updated file '%s'", p)
        }
        return
}

func (p *File) comparePathDepend(c *Comparer, d *Path) (err error) {
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
                return breakf(false, "no such file '%v'", ds)
        }

        if dt.After(tt) {
                c.globe.Timestamps[ts] = dt // Or tt?
                err = nil // Returns nil to request update.
        } else {
                err = breakf(true, "updated file '%s'", p)
        }
        return
}

func (p *File) prepare(pc *Preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:File: %v (%v) (%v -> %v)\n", p.Name, p, pc.program.project.name, pc.entry)
        }
        if err, brk := p.explicitly(pc); err != nil || brk {
                return err
        }
        if err, brk := p.implicitly(pc); err != nil || brk {
                return err
        }
        if err, brk := p.search(pc); err != nil || brk {
                return err
        }
        return nil
}

func (p *File) explicitly(pc *Preparer) (error, bool) {
        return pc.forEachExternalCaller(func(project *Project) (trybrk bool, err error) {
                if trace_prepare {
                        fmt.Printf("prepare:File: %v (explicitly: %v in %v) (%v -> %v)\n", p.Name, p, project.name, pc.entry.owner.name, pc.entry)
                }
                // Find concrete entry (by file represented name)
                if _, obj := project.scope.Find(p.Name); obj != nil {
                        if trace_prepare {
                                fmt.Printf("prepare:File: %v (found %v) (%s) (%v -> %v)\n", p.Name, obj, project.name, pc.entry.owner.name, pc.entry)
                        }
                        defer pc.setProject(pc.setProject(project))
                        if err, trybrk = pc.Prepare(obj), true; err != nil {
                                // ...
                        }
                }
                return
        })
}

func (p *File) implicitly(pc *Preparer) (error, bool) {
        return pc.forEachExternalCaller(func(project *Project) (trybrk bool, err error) {
                if trace_prepare {
                        fmt.Printf("prepare:File: %v (implicitly: %v in %v) (%v -> %v)\n", p.Name, p, project.name, pc.entry.owner.name, pc.entry)
                }

                defer pc.setProject(pc.setProject(project))

                var pss []*PatternStem
                if pss, err = project.FindPatterns(p.Name); err != nil {
                        return
                }

                ForPatterns: for i, ps := range pss {
                        for _, prog := range ps.Patent.programs {
                                if trace_prepare {
                                        fmt.Printf("prepare:File: %v (implicitly:%d: %v : %v) (in %v)\n", p.Name, i, ps, prog.depends, project.name)
                                }
                                for _, dep := range prog.depends {
                                        var ( g, _ = dep.(*GlobPattern); ok bool )
                                        if g != nil {
                                                if ok, err = p.checkPatternDepend(pc, project, ps, prog, g); err != nil { return }
                                                if !ok { continue ForPatterns }
                                        }
                                }
                        }
                        ps.file = p // Bounds PatternStem with the File.
                        if err = ps.prepare(pc); err == nil {
                                trybrk = true; break ForPatterns // Updated successfully!
                        } else if _, ok := err.(patternPrepareError); ok {
                                if trace_prepare {
                                        fmt.Printf("prepare:File: %v (implicitly:%d: %v) (error: %s) (%s) (%v -> %v)\n", p.Name, i, ps, err, project.name, pc.entry.owner.name, pc.entry)
                                }
                        } else {
                                trybrk = true; break ForPatterns // Update failed!
                        }
                }
                return
        })
}

func (p *File) checkPatternDepend(pc *Preparer, project *Project, ps *PatternStem, prog *Program, g *GlobPattern) (res bool, err error) {
        var name string
        if name, err = g.MakeString(ps.Stem); err != nil { return }
        if file := project.ToFile(name); file != nil { // Matches a FileMap (IsKnown(), may exists or not)
                //fmt.Printf("prepare:File: %v (implicitly:=: %v in %s)\n", p.Name, file, project.name)
                if file.IsExists() {
                        if trace_prepare {
                                fmt.Printf("prepare:File: %v (implicitly: %v exists in %s) (%v -> %v)\n", p.Name, file, project.name, pc.entry.owner.name, pc.entry)
                        }
                        res = true
                } else if trace_prepare && false {
                        fmt.Printf("prepare:File: %v (implicitly: %v missing in %s) (%v -> %v)\n", p.Name, file, project.name, pc.entry.owner.name, pc.entry)
                }
        }
        if _, sym := project.scope.Find(name); sym != nil {
                if trace_prepare {
                        fmt.Printf("prepare:File: %v (implicitly: found %v in %s) (%v -> %v)\n", p.Name, sym, project.name, pc.entry.owner.name, pc.entry)
                }
                res = true
        }

        // TODO: recursive find patterns:
        /*if project.FindPatterns(name) != nil {
                res = true
        }*/
        return
}

func (p *File) search(pc *Preparer) (error, bool) {
        if p.IsExists() {
                if trace_prepare {
                        fmt.Printf("prepare:File: %v (search: exists %v) (%v)\n", p.Name, p, pc.entry)
                }
                pc.targets.Append(p)
                return nil, true
        }
        return pc.forEachExternalCaller(func(project *Project) (trybrk bool, err error) {
                str, err := p.Strval()
                if err != nil { return }
                if f := project.SearchFile(str); /*!f.IsKnown()*/f.IsKnown() || f.IsExists() {
                        if trace_prepare {
                                fmt.Printf("prepare:File: %v (search: known as %v but missing) (%v -> %v)\n",
                                        p.Name, f, project.name, pc.entry)
                        }
                        pc.targets.Append(f); trybrk = true
                } else {
                        if trace_prepare {
                                fmt.Printf("prepare:File: %v (search: unknown %v) (%v -> %v)\n",
                                        p.Name, p.Dir, project.name, pc.entry)
                        }
                        err = unknownFileError{ fmt.Errorf("unknown file '%v' (%v)", p.Name, f), p }
                }
                return
        })
}

type Flag struct {
        Name Value
}
func (p *Flag) Type() Type { return FlagType }
func (p *Flag) String() (s string) {
        s = "-" + p.Name.String()
        return
}
func (p *Flag) Strval() (s string, e error) {
        if s, e = p.Name.Strval(); e == nil { 
                 s = "-" + s
        }
        return
}
func (p *Flag) Integer() (int64, error) { return 0, nil }
func (p *Flag) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

func (p *Flag) disclose(scope *Scope) (res Value, err error) {
        var name Value
        if name, err = p.Name.disclose(scope); err != nil { return }
        if name != nil { res = &Flag{ name } }
        return
}

func (p *Flag) referencing(o Object) bool {
        return p.Name.referencing(o)
}
        
type Compound struct {
        Elements
}
func (p *Compound) String() (s string) {
        s = "\""
        for _, e := range p.Elems {
                switch t := e.(type) {
                case *String: s += t.Value
                default: s += t.String()
                }
        }
        s += "\""
        return
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

func (p *Compound) disclose(scope *Scope) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = p.Elements.disclose(scope); err != nil { return }
        if num > 0 { res = &Compound{ Elements{ elems } } }
        return
}

type List struct {
        Elements
}
func (p *List) Type() Type { return ListType }
func (p *List) String() (s string) {
        for i, e := range p.Elems {
                if 0 < i {
                        s += " "
                }
                s += e.String()
        }
        return
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

func (p *List) disclose(scope *Scope) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = p.Elements.disclose(scope); err != nil { return }
        if num > 0 { res = &List{ Elements{ elems } } }
        return
}

func (p *List) compare(c *Comparer) (err error) {
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

func (p *List) compareFileDepend(c *Comparer, d *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:List:File: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }
        if n := len(p.Elems); n == 1 {
                if elem, _ := p.Elems[0].(comparable); elem != nil {
                        err = elem.compareFileDepend(c, d)
                } else {
                        err = breakf(false, "incomparable target (%v)", p.Elems[0])
                }
        } else if n == 0 {
                err = breakf(false, "comparing empty list")
        } else {
                err = breakf(false, "comparing multiple targets (%v)", p)
        }
        return
}

func (p *List) comparePathDepend(c *Comparer, d *Path) (err error) {
        if trace_compare {
                fmt.Printf("compare:List:Path: %v (depends: %v) (%v %T)\n", p, d, c.target, c.target)
        }
        if n := len(p.Elems); n == 1 {
                if elem, _ := p.Elems[0].(comparable); elem != nil {
                        err = elem.comparePathDepend(c, d)
                } else {
                        err = breakf(false, "incomparable target (%v)", p.Elems[0])
                }
        } else if n == 0 {
                err = breakf(false, "comparing empty list")
        } else {
                err = breakf(false, "comparing multiple targets (%v)", p)
        }
        return
}

type Group struct {
        List
}
func (p *Group) Type() Type { return GroupType }
func (p *Group) String() string { return "(" + p.List.String() + ")" }
func (p *Group) Strval() (s string, err error) {
        if s, err = p.List.Strval(); err == nil {
                s = "(" + s + ")"
        }
        return
}

func (p *Group) disclose(scope *Scope) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = p.Elements.disclose(scope); err != nil { return }
        if num > 0 { res = &Group{ List{ Elements{ elems } } } }
        return
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
func (p *Pair) Type() Type { return PairType }
func (p *Pair) String() string { return p.Key.String() + "=" + p.Value.String() }
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

func (p *Pair) disclose(scope *Scope) (res Value, err error) {
        var k, v Value
        if k, err = p.Key.disclose(scope); err != nil { return }
        if v, err = p.Value.disclose(scope); err != nil { return }
        if k != nil || v != nil {
                if k == nil { k = p.Key }
                if v == nil { v = p.Value }
                res = &Pair{ k, v }
        }
        return
}

func (p *Pair) referencing(o Object) bool {
        return p.Key.referencing(o) || p.Value.referencing(o)
}

// Delegate wraps '$(foo a,b,c)' into Valuer
type delegate struct {
        p token.Position
        o Object
        a []Value
        closure *Scope // disclosed context
}
func (p *delegate) Position() token.Position { return p.p }
func (p *delegate) Type() Type { return DelegateType }
func (p *delegate) String() (s string) {
        var na = len(p.a)
        s = "$"
        s += "(" //if na > 0 { s += "(" }
        if false {
                if sc := p.o.DeclScope(); sc != nil && sc.Comment() == "use"/*use scope*/ {
                        s += sc.Comment() + "->"
                } else if pp := p.o.OwnerProject(); pp != nil {
                        s += pp.Name() + "->"
                }
        }
        s += p.o.Name()
        if na > 0 {
                s += " "
                for i, a := range p.a {
                        if i > 0 { s += "," }
                        s += a.String()
                }
        }
        s += ")"
        return
}
func (p *delegate) Strval() (string, error) { if v, e := p.eval(); e == nil { return v.Strval() } else { return "", e }}
func (p *delegate) Integer() (int64, error) { if v, e := p.eval(); e == nil { return v.Integer() } else { return 0, e }}
func (p *delegate) Float() (float64, error) { if v, e := p.eval(); e == nil { return v.Float() } else { return 0, e }}
func (p *delegate) eval() (res Value, err error) {
        var (
                args []Value
                scope = p.closure
        )
        if scope == nil { scope = p.o.DeclScope() }

        // FIXME: disclosed context not applied?
        if args, err = p.discloseArgs(ClosureContext{scope}); err != nil {
                return
        }
        
        switch o := p.o.(type) {
        default: err = fmt.Errorf("unknown delegated object %v", o)
        case Caller:
                if res, err = o.Call(p.p, args...); err != nil {
                        err = fmt.Errorf("$(%s): %v", p.o.Name(), err)
                }
        case Executer:
                if args, err = o.Execute(p.p, args...); err != nil {
                        err = fmt.Errorf("${%s}: %v", p.o.Name(), err)
                } else {
                        res = &List{Elements{args}}
                }
        }
        if err != nil {
                fmt.Printf("%v: %v\n", p.p, err)
        } else if res == nil {
                res = UniversalNone 
        }
        return
}

func (p *delegate) disclose(scope *Scope) (res Value, err error) {
        var ( o = p.o; v Value; changed bool )
        if v, err = o.disclose(scope); err != nil { return }
        if v != nil {
                if o, _ = v.(Object); o != nil {
                        changed = true
                } else {
                        err = fmt.Errorf("invalid delegate %v", v)
                        return
                }
        }

        /*switch t := o.(type) {
        case *RuleEntry:
                for _, prog := range t.programs {
                        prog.closure = scope
                }
        }*/

        var args []Value
        for _, a := range p.a {
                if v, err = a.disclose(scope); err != nil { return }
                if v != nil { a, changed = v, true }
                args = append(args, a)
        }
        if changed && err == nil {
                res = &delegate{ p.p, o, args, scope }
        }
        return
}

func (p *delegate) discloseArgs(cc ClosureContext) (args []Value, err error) {
        for _, a := range p.a {
                if v, e := Disclose(cc, a); e != nil {
                        // TODO: errors...
                        return nil, e
                } else if v != nil {
                        a = v
                }
                args = append(args, a)
        }
        return
}

func (p *delegate) referencing(o Object) bool {
        if p.o == o || p.o.referencing(o) {
                return true
        }
        for _, a := range p.a {
                if a.referencing(o) {
                        return true
                }
        }
        return false
}

func (p *delegate) compare(c *Comparer) (err error) {
        if trace_compare {
                fmt.Printf("compare:delegate: %v (%v %T)\n", p, c.target, c.target)
        }
        var v Value
        if v, err = p.eval(); err == nil {
                err = c.compare(v)
        }
        return
}

func (p *delegate) compareFileDepend(c *Comparer, d *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:delegate:File: %v (%v %T)\n", p, c.target, c.target)
        }
        var value Value
        if value, err = p.eval(); err != nil { return }
        if comp, _ := value.(comparable); comp != nil {
                err = comp.compareFileDepend(c, d)
        } else {
                err = fmt.Errorf("incomparable target (%v)", value)
                if trace_compare {
                        fmt.Printf("compare:delegate:File: %v (incomparable: %v %T)\n", p, value, value)
                }
        }
        return
}

func (p *delegate) comparePathDepend(c *Comparer, d *Path) (err error) {
        if trace_compare {
                fmt.Printf("compare:delegate:Path: %v (%v %T)\n", p, c.target, c.target)
        }
        var value Value
        if value, err = p.eval(); err != nil { return }
        if comp, _ := value.(comparable); comp != nil {
                err = comp.comparePathDepend(c, d)
        } else {
                err = fmt.Errorf("incomparable target (%v)", value)
                if trace_compare {
                        fmt.Printf("compare:delegate:Path: %v (incomparable: %v %T)\n", p, value, value)
                }
        }
        return
}

func (p *delegate) prepare(pc *Preparer) (err error) {
        if trace_prepare {
                fmt.Printf("prepare:delegate: %v (%v -> %v)\n", p, pc.entry.owner.name, pc.entry)
        }
        var val Value
        if val, err = Reveal(p); err != nil { return }
        for _, d := range Join(val) {
                if err = pc.Prepare(d); err != nil { break }
        }
        return
}

type closure struct {
        p token.Position
        o Object
        a []Value
        closure *Scope
}

func (p *closure) Position() token.Position { return p.p }
func (p *closure) Type() Type { return ClosureType }
func (p *closure) String() (s string) {
        var na = len(p.a)
        s = "&"
        s += "("
        // FIXME: needs the original name value to represent the original form
        s += p.o.Name()
        if na > 0 {
                for i, a := range p.a {
                        if i > 0 { s += "," }
                        s += a.String()
                }
        }
        s += ")"
        return
}
func (p *closure) Integer() (int64, error) { return p.o.Integer() }
func (p *closure) Float() (float64, error) { return p.o.Float() }
func (p *closure) Strval() (s string, err error) {
        var v Value
        if v, err = p.eval(); err == nil {
                s, err = v.Strval()
        }
        return
}
func (p *closure) eval() (res Value, err error) {
        switch o := p.o.(type) {
        default: err = fmt.Errorf("unknown closure object %v", p.o)
        case Caller:
                if res, err = o.Call(p.p, p.a...); err != nil {
                        err = fmt.Errorf("&(%s): %v", p.o.Name(), err)
                }
        case Executer:
                if t, _ := o.(*RuleEntry); t != nil && p.closure != nil {
                        defer t.SetClosure(t.SetClosure(p.closure))
                }
                var a []Value
                if a, err = o.Execute(p.p, p.a...); err != nil {
                        err = fmt.Errorf("&{%s}: %v", p.o.Name(), err)
                } else {
                        res = &List{Elements{a}}
                }
        }
        if err != nil {
                fmt.Printf("%v: %v\n", p.p, err)
        } else if res == nil {
                res = UniversalNone 
        }
        return
}
func (p *closure) disclose(scope *Scope) (res Value, err error) {
        var ( o Object; v Value; changed bool )
        if _, o = scope.Find(p.o.Name()); o != nil {
                changed = true
        } else {
                o = p.o
        }

        // Disclose the object, it's value may have disclosures.
        if v, err = o.disclose(scope); err != nil { return }
        if v != nil {
                if o, _ = v.(Object); o != nil {
                        changed = true
                } else {
                        err = fmt.Errorf("invalid closure %v", v)
                        return
                }
        }

        /*switch t := o.(type) {
        case *RuleEntry:
                for _, prog := range t.programs {
                        prog.closure = scope
                }
        }*/

        var args []Value
        for _, a := range p.a {
                if v, err = a.disclose(scope); err != nil { return }
                if v != nil { a, changed = v, true }
                args = append(args, a)
        }
        if changed && err == nil {
                res = &closure{ p.p, o, args, scope }
        }
        return
}

func (p *closure) referencing(o Object) bool {
        if p.o == o {
                return true
        }
        for _, a := range p.a {
                if a.referencing(o) {
                        return true
                }
        }
        return false
}

func (p *closure) prepare(pc *Preparer) (err error) {
        if trace_prepare {
                fmt.Printf("prepare:closure: %v (%v)\n", p, pc.entry)

        }
        if v, e := pc.program.cc.disclose(p); e != nil {
                err = e
        } else if v == nil {
                err = fmt.Errorf("preparing nil closure (%v)", p)
        } else {
                err = pc.Prepare(v)
        }
        return
}

// Value returned by (plain) modifier.
type Plain struct {
        Value string
        Name string
}
func (p *Plain) disclose(_ *Scope) (Value, error) { return nil, nil }
func (p *Plain) referencing(_ Object) bool { return false }
func (p *Plain) Type() Type  { return PlainType }
func (p *Plain) String() string {
        s := "(plain"
        if p.Name != "" {
                s += "(" + p.Name + ")"
        } 
        s += " " + p.Value + ")"
        return s
}
func (p *Plain) Strval() (string, error) { return p.Value, nil }
func (p *Plain) Integer() (int64, error) { return strconv.ParseInt(p.Value, 10, 64) }
func (p *Plain) Float() (float64, error) { return strconv.ParseFloat(p.Value, 64) }

type JSON struct {
        Value Value
}
func (p *JSON) disclose(_ *Scope) (Value, error) { return nil, nil }
func (p *JSON) referencing(_ Object) bool { return false }
func (p *JSON) Type() Type { return JSONType }
func (p *JSON) String() string { return "(json " + p.Value.String() + ")" }
func (p *JSON) Strval() (string, error) { return p.Value.Strval() }
func (p *JSON) Integer() (int64, error) { return 0, nil }
func (p *JSON) Float() (float64, error) { return 0, nil }

type XML struct {
        Value Value
}
func (p *XML) disclose(_ *Scope) (Value, error) { return nil, nil }
func (p *XML) referencing(_ Object) bool { return false }
func (p *XML) Type() Type { return XMLType }
func (p *XML) String() string { return "(json " + p.Value.String() + ")" }
func (p *XML) Strval() (string, error) { return p.Value.Strval() }
func (p *XML) Integer() (int64, error) { return 0, nil }
func (p *XML) Float() (float64, error) { return 0, nil }

type YAML struct {
        Value Value
}
func (p *YAML) disclose(_ *Scope) (Value, error) { return nil, nil }
func (p *YAML) referencing(_ Object) bool { return false }
func (p *YAML) Type() Type { return YAMLType }
func (p *YAML) String() string { return "(json " + p.Value.String() + ")" }
func (p *YAML) Strval() (string, error) { return p.Value.Strval() }
func (p *YAML) Integer() (int64, error) { return 0, nil }
func (p *YAML) Float() (float64, error) { return 0, nil }

type ExecBuffer struct {
        Tie io.Writer
        Buf *bytes.Buffer
        Line *regexp.Regexp
        Subm [][][][]byte
        line []byte
}

func (p *ExecBuffer) Write(b []byte) (n int, err error) {
        if p.Line != nil {
                i := bytes.Index(b, []byte("\n"))
                if i == -1 {
                        p.line = append(p.line, b...)
                } else {
                        p.line = append(p.line, b[:i]...)
                }
                if m := p.Line.FindAllSubmatch(p.line, -1); m != nil {
                        p.Subm = append(p.Subm, m)
                }
                if i != -1 {
                        p.line = b[i+1:]
                }
        }
        if p.Tie != nil {
                if n, err = p.Tie.Write(b); err != nil {
                        return
                }
        }
        if p.Buf != nil {
                if n, err = p.Buf.Write(b); err != nil {
                        return
                }
        }
        if err == nil && n == 0 {
                // Returns the number of bytes to avoid "short write" errors.
                // The real bytes written is discarded.
                n = len(b)
        }
        return
}

type ExecResult struct {
        Stdout ExecBuffer
        Stderr ExecBuffer
        Status int
}
func (p *ExecResult) disclose(_ *Scope) (Value, error) { return nil, nil }
func (p *ExecResult) referencing(_ Object) bool { return false }
func (p *ExecResult) Type() Type { return ExecResultType }
func (p *ExecResult) Integer() (int64, error) { return int64(p.Status), nil }
func (p *ExecResult) Float() (float64, error) { return float64(p.Status), nil }
func (p *ExecResult) Strval() (s string, err error) {
        if p.Stdout.Buf != nil {
                s = p.Stdout.Buf.String()
        }
        return
}
func (p *ExecResult) String() string {
        var s bytes.Buffer
        fmt.Fprintf(&s, "(ExecResult status=%d", p.Status)
        if p.Stdout.Buf != nil {
                fmt.Fprintf(&s, " stdout=%S", p.Stdout.Buf)
        }
        if p.Stderr.Buf != nil {
                fmt.Fprintf(&s, " stdout=%S", p.Stderr.Buf)
        }
        fmt.Fprintf(&s, ")")
        return s.String()
}

// Pattern
type Pattern interface {
        Value
        MakeConcreteEntry(patent *RuleEntry, stem string) (entry *RuleEntry, err error)
        Match(s string) (matched bool, stem string, err error)
}

type pattern struct {
}

func (p *pattern) Type() Type        { return PatternType }
func (p *pattern) Integer() (int64, error) { return 0, nil }
func (p *pattern) Float() (float64, error) { return 0, nil }
func (p *pattern) makeEntry(patent *RuleEntry, name, stem string) (entry *RuleEntry, err error) {
        if patent.class == GlobRuleEntry {
                entry = new(RuleEntry); *entry = *patent
                entry.name = name
                entry.stem = stem
                if patent.owner.isFile(filepath.Base(name)) {
                        entry.class = StemmedFileEntry
                        if false && entry.file == nil {
                                entry.file = entry.owner.SearchFile(name)
                        }
                } else {
                        entry.class = StemmedRuleEntry
                }
        } else {
                err = fmt.Errorf("make entry `%s' (%s): invalid class `%v'", name, stem, patent.class)
        }
        return
}

func (*pattern) disclose(_ *Scope) (Value, error) { return nil, nil }

// GlobPattern represents glob expressions (e.g. '%.o', '[a-z].o', 'a?a.o')
// FIXME: PercPattern -> %.o
//        GlobPattern -> [a-z].o a?a.o
type GlobPattern struct {
        pattern
        Prefix Value
        Suffix Value
}

func (p *GlobPattern) String() string { return fmt.Sprintf("%s%%%s", p.Prefix.String(), p.Suffix.String()) }
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
func (p *GlobPattern) Match(s string) (matched bool, stem string, err error) {
        var prefix, suffix string
        if prefix, err = p.Prefix.Strval(); err != nil && prefix == "" || strings.HasPrefix(s, prefix) {
                if suffix, err = p.Suffix.Strval(); err != nil && suffix == "" || strings.HasSuffix(s, suffix) {
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

func (p *GlobPattern) MakeConcreteEntry(patent *RuleEntry, stem string) (entry *RuleEntry, err error) {
        var name string
        if name, err = p.MakeString(stem); err != nil { return }
        return p.makeEntry(patent, name, stem)
}

func (p *GlobPattern) referencing(o Object) bool {
        return p.Prefix.referencing(o) || p.Suffix.referencing(o)
}

func (p *GlobPattern) prepare(pc *Preparer) (err error) {
        if trace_prepare {
                fmt.Printf("prepare:GlobPattern: %v(%v) (from %v) (%v -> %v)\n", p, pc.stem, pc.entry.file, pc.entry.owner.name, pc.entry)
        }
        if pc.stem == "" {
                err = fmt.Errorf("empty stem (%s, %v)", p, pc.entry)
                return
        }

        var target string
        if target, err = p.MakeString(pc.stem); err != nil { return }

        // Check if target is a file (if source entry is file).
        if brk := false; pc.entry.file != nil { //! See also `File.checkPatternDepend`.
                err, brk = pc.forEachExternalCaller(func(project *Project) (trybrk bool, err error) {
                        if file := project.SearchFile(target); file.IsKnown() || file.IsExists() {
                                if trace_prepare {
                                        fmt.Printf("prepare:GlobPattern: %v(%v) (file %v in %s) (%v -> %v)\n", p, pc.stem, file, project.name, pc.entry.owner.name, pc.entry)
                                }
                                defer pc.setProject(pc.setProject(project))
                                err, trybrk = file.prepare(pc), true
                        } else if _, sym := project.scope.Find(target); sym != nil {
                                if trace_prepare {
                                        fmt.Printf("prepare:GlobPattern: %v(%v) (found %v in %v) (%v -> %v)\n", p, pc.stem, sym, project.name, pc.entry.owner.name, pc.entry)
                                }
                                defer pc.setProject(pc.setProject(project))
                                err, trybrk = pc.prepare(sym), true
                        }
                        return
                })
                if err != nil || brk {
                        return
                }
        }
        
        if trace_prepare {
                fmt.Printf("prepare:GlobPattern: %v(%v) (target %v) (%v -> %v)\n", p, pc.stem, target, pc.entry.owner.name, pc.entry)
        }
        if err = pc.prepareTarget(target); err == nil {
                return // Good!
        } else {
                if trace_prepare {
                        fmt.Printf("prepare:GlobPattern: %v (error: %v) (%v) (%v)\n", p, err, pc.stem, pc.entry)
                }
                err = patternPrepareError(err)
        }
        return
}

// TODO: implement regexp pattern
type RegexpPattern struct {
        pattern
}

func NewRegexpPattern() Pattern {
        return &RegexpPattern{}
}

func (p *RegexpPattern) String() string {
        if s, e := p.Strval(); e == nil {
                return fmt.Sprintf("RegexpPattern{%s}", s)
        } else {
                return fmt.Sprintf("RegexpPattern{%s}!(%s)", s, e)
        }
}
func (p *RegexpPattern) Strval() (s string, err error) { return "", nil }
func (p *RegexpPattern) Match(s string) (matched bool, stem string, err error) {
        // TODO: regexp matching...
        return
}
func (p *RegexpPattern) MakeConcreteEntry(patent *RuleEntry, stem string) (entry *RuleEntry, err error) {
        // TODO: creating new match entry
        return
}

func (p *RegexpPattern) referencing(_ Object) bool { return false }

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

func revealElems(elems []Value) (result []Value, changed int, err error) {
        for _, elem := range elems {
                var v Value
                if v, err = Reveal(elem); err != nil { return }
                if v != elem { changed += 1 }
                result = append(result, v)
        }
        return
}

// Reveal expends delegates and Valuer recursively.
func Reveal(v Value) (res Value, err error) {
        switch t := v.(type) {
        case *delegate: res, err = t.eval()
        case *List:
                var ( elems []Value; changed = 0 )
                if elems, changed, err = revealElems(t.Elems); err != nil { return }
                if changed > 0 { res = &List{Elements{elems}} }
        }
        if res == nil { res = v }
        return
}

// Disclose expends closures to normal value recursively.
func Disclose(cc ClosureContext, value Value) (res Value, err error) {
        if false {
                fmt.Printf("Disclose: %T %v\n", value, value)
        }
        if res, err = cc.disclose(value); err != nil { return }
        if res == nil { res = value }
        return
}

// Join combines lists recursively into one list.
func Join(args... Value) (elems []Value) {
        for _, arg := range args {
                /* switch t := arg.(type) {
                case *List:
                        for _, elem := range t.Elems {
                                elems = append(elems, Join(elem)...)
                        }
                default:
                        elems = append(elems, t)
                } */
                if l, _ := arg.(*List); l != nil {
                        elems = append(elems, Join(l.Elems...)...)
                } else {
                        elems = append(elems, arg)
                }
        }
        return
}

// JoinReveal join revealed elements into one list.
func JoinReveal(args... Value) (elems []Value, err error) {
        for _, elem := range Join(args...) {
                if elem, err = Reveal(elem); err != nil { break }
                elems = append(elems, Join(elem)...)
        }
        return
}

// JoinEval join evaluated (disclosed and revealed) elements into one list.
func JoinEval(cc ClosureContext, args... Value) (elems []Value, err error) {
        for _, elem := range Join(args...) {
                if elem, err = Disclose(cc, elem); err != nil { break }
                if elem, err = Reveal(elem); err != nil { break }
                if l, _ := elem.(*List); l != nil {
                        var a []Value
                        if a, err = JoinEval(cc, l.Elems...); err != nil { return }
                        elems = append(elems, a...)
                } else {
                        elems = append(elems, elem)
                }
        }
        return
}

func Eval(cc ClosureContext, value Value) (res Value, err error) {
        if value, err = Disclose(cc, value); err != nil { return }
        res, err = Reveal(value); return
}

func Delegate(pos token.Position, obj Object, args... Value) Value {
        return &delegate{ pos, obj, args, nil }
}

func Closure(pos token.Position, obj Object, args... Value) Value {
        if obj == nil {
                panic("closure of nil")
        }
        return &closure{ pos, obj, args, nil }
}

func Refs(a Value, o Object) bool {
        return a.referencing(o)
}

func strval(s string) Value { return &String{s} }

func MakeListOrScalar(elems []Value) (res Value) {
        if x := len(elems); x == 0 {
                res = UniversalNone
        } else if x == 1 {
                res = elems[0]
        } else {
                res = &List{Elements{elems}}
        }
        return
}

func Scalar(v Value, t Type) (res Value) {
        if v.Type() == t {
                res = v
        } else if l, _ := v.(*List); l != nil && l.Len() > 0 {
                res = Scalar(l.Elems[0], t)
        }
        return
}
