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
)

// Value represents a value of a type.
type Value interface {
        // Type returns the underlying type of the value.
        Type() Type

        // Lit returns the literal representations of the value.
        String() string

        // Strval returns the string form of the value.
        Strval() string

        // Integer returns the integer form of the value.
        Integer() int64

        // Float returns the float form of the value.
        Float() float64

        // disclose method, also prevents creating new Value type from
        // other packages.
        disclose(scope *Scope) (Value, error)

        // Recursively detecting whether this value is referencing
        // to the object (to avoid loop-delegation).
        referencing(o Object) bool
}

type value struct {}
func (*value) disclose(_ *Scope) (Value, error) { return nil, nil }
func (*value) referencing(_ Object) bool { return false }
func (*value) Type() Type         { return InvalidType }
func (*value) String() string     { return "" }
func (*value) Strval() string     { return "" }
func (*value) Integer() int64     { return 0 }
func (*value) Float() float64     { return 0 }

// TODO: use it with (compare) modifier
type comparable interface {
        compare(target Value)
}

// Preparer prepares prerequisites of targets.
type Preparer struct {
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

func (pc *Preparer) Prepare(value interface{}) (err error) {
        if v := reflect.ValueOf(value); v.Kind() == reflect.Slice {
                for i := 0; i < v.Len(); i++ {
                        var prereq = v.Index(i).Interface();
                        /*if trace_prepare {
                                fmt.Printf("prepare: %v (%T)\n", prereq, prereq)
                        }*/
                        if err = pc.prepare(prereq); err == nil {
                                // Good!
                        } else if ute, ok := err.(unknownTargetError); ok {
                                fmt.Printf("prepare:FIXME: unknown target %v (%v)\n", ute.target, pc.entry)
                                break
                        } else if ufe, ok := err.(unknownFileError); ok {
                                fmt.Printf("prepare:FIXME: unknown file %v (%v)\n", ufe.file, pc.entry)
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

func (pc *Preparer) disctx() (context *Scope) {
        if context = pc.program.disctx; context == nil {
                context = pc.program.scope //pc.program.project.scope
        }
        return
}

func (pc *Preparer) Targets() *List { return pc.targets }

func (pc *Preparer) forEachExternalCaller(f func (*Project) (bool, error)) (err error, brk bool) {
        var triedm = map[*Project]bool{ pc.program.project:true }
        if brk, err = f(pc.program.project); brk /*|| err != nil*/ {
                return
        }
        for caller := pc.entry.caller; caller != nil; caller = caller.entry.caller {
                // Find the last caller.
                if caller == pc || caller == pc.entry.caller { continue }
                if tried, has := triedm[caller.entry.project]; caller != pc.entry.caller && !(has&&tried) {
                        if brk, err = f(caller.entry.project); brk /*|| err != nil*/ {
                                break
                        } else {
                                triedm[caller.entry.project] = true
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
        return &Preparer{ prog, entry, args, new(List), stem }
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
func (p *Argumented) Strval() (s string) {
        s = p.Value.Strval()
        s += "("
        for i, a := range p.Args {
                if i > 0 {
                        s += ","
                }
                s += a.Strval()
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
func (p *integer) Integer() int64      { return p.Value }
func (p *integer) Float() float64      { return float64(p.Value) }

type Bin struct { integer }
func (p *Bin) Type() Type          { return BinType }
func (p *Bin) String() string      { return p.Strval() }
func (p *Bin) Strval() string      { return strconv.FormatInt(int64(p.Value),2) }

type Oct struct { integer }
func (p *Oct) Type() Type          { return OctType }
func (p *Oct) String() string      { return p.Strval() }
func (p *Oct) Strval() string      { return strconv.FormatInt(int64(p.Value),8) }

type Int struct { integer }
func (p *Int) Type() Type          { return IntType }
func (p *Int) String() string      { return p.Strval() }
func (p *Int) Strval() string      { return strconv.FormatInt(int64(p.Value),10) }

type Hex struct { integer }
func (p *Hex) Type() Type          { return HexType }
func (p *Hex) String() string      { return p.Strval() }
func (p *Hex) Strval() string      { return strconv.FormatInt(int64(p.Value),16) }

type Float struct {
        Value float64
}
func (p *Float) disclose(_ *Scope) (Value, error) { return nil, nil }
func (p *Float) referencing(_ Object) bool { return false }
func (p *Float) Type() Type        { return FloatType }
func (p *Float) String() string    { return p.Strval() }
func (p *Float) Strval() string    { return strconv.FormatFloat(float64(p.Value),'g', -1, 64) }
func (p *Float) Integer() int64    { return int64(p.Value) }
func (p *Float) Float() float64    { return p.Value }

type DateTime struct {
        Value time.Time 
}
func (*DateTime) disclose(_ *Scope) (Value, error) { return nil, nil }
func (*DateTime) referencing(_ Object) bool { return false }
func (p *DateTime) Type() Type     { return DateTimeType }
func (p *DateTime) String() string { return p.Strval() }
func (p *DateTime) Strval() string { return time.Time(p.Value).Format("2006-01-02T15:04:05.999999999Z07:00") } // time.RFC3339Nano
func (p *DateTime) Integer() int64 { return p.Value.Unix() }
func (p *DateTime) Float() float64 { return float64(p.Integer()) }

type Date struct { DateTime }
func (*Date) disclose(_ *Scope) (Value, error) { return nil, nil }
func (*Date) referencing(_ Object) bool { return false }
func (p *Date) Type() Type         { return DateType }
func (p *Date) String() string     { return p.Strval() }
func (p *Date) Strval() string     { return time.Time(p.Value).Format("2006-01-02") }
func (p *Date) Integer() int64     { return p.Value.Unix() }
func (p *Date) Float() float64     { return float64(p.Integer()) }

type Time struct { DateTime }
func (*Time) disclose(_ *Scope) (Value, error) { return nil, nil }
func (*Time) referencing(_ Object) bool { return false }
func (p *Time) Type() Type         { return TimeType }
func (p *Time) String() string     { return p.Strval() }
func (p *Time) Strval() string     { return time.Time(p.Value).Format("15:04:05.999999999Z07:00") }
func (p *Time) Integer() int64     { return p.Value.Unix() }
func (p *Time) Float() float64     { return float64(p.Integer()) }

type Uri struct {
        Value *url.URL
}
func (*Uri) disclose(_ *Scope) (Value, error) { return nil, nil }
func (*Uri) referencing(_ Object) bool { return false }
func (p *Uri) Type() Type          { return UriType }
func (p *Uri) String() string      { return p.Strval() }
func (p *Uri) Strval() string      { return p.Value.String() }
func (p *Uri) Integer() int64      { return int64(len(p.Value.String())) }
func (p *Uri) Float() float64      { return float64(p.Integer()) }

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
func (p *String) Strval() string   { return p.Value }
func (p *String) Integer() int64   { i, _ := strconv.ParseInt(p.Value, 10, 64); return i }
func (p *String) Float() float64   { f, _ := strconv.ParseFloat(p.Value, 64); return f }

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
func (p *Bareword) String() string { return p.Value }
func (p *Bareword) Strval() string { return p.Value }
func (p *Bareword) Integer() int64 { return 0 }
func (p *Bareword) Float() float64 { return float64(p.Integer()) }

func (p *Bareword) prepare(pc *Preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:Bareword: %v (%v)\n", p, pc.entry)
        }
        //pc.source = p.Value
        return pc.prepareTarget(p.Value)
}

func (pc *Preparer) prepareTargetValue(value Value) error {
        if v, e := value.disclose(pc.disctx()); e != nil {
                return e
        } else {
                if v == nil { v = value }
                return pc.prepareTarget(v.Strval())
        }
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
                                fmt.Printf("prepare:Target: %v (found %v) (project %v, %v)\n", target, obj, project.name, pc.entry)
                        }
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
                for _, ps := range project.FindPatterns(target) {
                        if trace_prepare {
                                fmt.Printf("prepare:Target: %v (stemmed %v) (project %v, %v)\n", target, ps, project.name, pc.entry)
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

func (pc *Preparer) addFile(project *Project, name string) (err error) {
        if trace_prepare {
                fmt.Printf("prepare:AddFile: %v (project %v, %v)\n", name, project.name, pc.entry)
        }
        
        if f := project.SearchFile(name); f.Info != nil {
                pc.targets.Append(f)
        } else {
                var wd = project.AbsPath()
                if !filepath.IsAbs(name) {
                        name = filepath.Join(wd, name)
                }
                err = fmt.Errorf("No such file '%v'", name)
        }
        return
}

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

func (p *Elements) disclose(scope *Scope) ([]Value, int, error) {
        var elems []Value
        var num = 0 
        for _, elem := range p.Elems {
                //fmt.Printf("Elements.disclose: %T %v\n", elem, elem)
                if elem == nil {
                        continue
                }
                if v, e := elem.disclose(scope); e != nil {
                        return nil, 0, e
                } else if v != nil {
                        elem = v
                        num += 1
                }
                elems = append(elems, elem)
        }
        return elems, num, nil
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
func (p *Barecomp) String() (s string) {
        for _, e := range p.Elems {
                switch t := e.(type) {
                case *String: s += t.Value
                default: s += t.String()
                }
        }
        return
}
func (p *Barecomp) Strval() (s string) {
        for _, e := range p.Elems {
                s += e.Strval()
        }
        return
}
func (p *Barecomp) Type() Type     { return BarecompType }
func (p *Barecomp) Integer() int64 { return int64(len(p.Elems)) }
func (p *Barecomp) Float() float64 { return float64(p.Integer()) }

func (p *Barecomp) disclose(scope *Scope) (Value, error) {
        if elems, num, err := p.Elements.disclose(scope); err != nil {
                return nil, err
        } else if num > 0 {
                return &Barecomp{ Elements{ elems } }, nil
        }
        return nil, nil
}

func (p *Barecomp) prepare(pc *Preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:Barecomp: %v (%v)\n", p, pc.entry)
        }
        return pc.prepareTargetValue(p)
}

type Barefile struct {
        Name Value
        File *File
}
func (p *Barefile) Type() Type { return BarefileType }
func (p *Barefile) String() string { return p.Name.String() }
func (p *Barefile) Strval() string { return p.Name.Strval() }
func (p *Barefile) Integer() int64 { return 0 }
func (p *Barefile) Float() float64 { return float64(p.Integer()) }
func (p *Barefile) disclose(scope *Scope) (Value, error) {
        if name, err := p.Name.disclose(scope); err != nil {
                return nil, err
        } else if name != nil {
                return &Barefile{ name, p.File }, nil
        }
        return nil, nil
}
func (p *Barefile) referencing(o Object) bool {
        return p.Name.referencing(o)
}

func (p *Barefile) prepare(pc *Preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:Barefile: %v (project %v, %v)\n", p, pc.entry.project.name, pc.entry)
        }
        if p.File != nil {
                if s := p.Name.Strval(); s != p.File.Name {
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
func (p *Glob) String() (s string) { return p.Tok.String() }
func (p *Glob) Strval() string { return p.Tok.String() }
func (p *Glob) Integer() int64 { return 0 }
func (p *Glob) Float() float64 { return float64(p.Integer()) }
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
        return
}
func (p *Path) Strval() (s string) {
        // TODO: add '/' for root dir
        var sep = true
        for i, seg := range p.Elems {
                if i > 0 && sep {
                        s += string(os.PathSeparator) 
                }
                s += seg.Strval()
                if ps, ok := seg.(*PathSeg); ok && ps != nil && ps.Value == '/' {
                        sep = false
                } else {
                        sep = true
                }
        }
        // TODO: add '/' if there's such a suffix
        return
}
func (p *Path) Integer() int64     { return 0 }
func (p *Path) Float() float64     { return float64(p.Integer()) }
func (p *Path) Type() Type         { return PathType }
func (p *Path) disclose(scope *Scope) (Value, error) {
        if elems, num, err := p.Elements.disclose(scope); err != nil {
                return nil, err
        } else if num > 0 {
                return &Path{ Elements{ elems }, p.File }, nil
        }
        return nil, nil
}

func (p *Path) prepare(pc *Preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:Path: %v (%v)\n", p, pc.entry)
        }
        return pc.prepareTargetValue(p)
}

type PathSeg struct {
        Value rune 
        value
}
func (p *PathSeg) Type() Type { return PathSegType }
func (p *PathSeg) String() string { return p.Strval() }
func (p *PathSeg) Strval() (s string) {
        switch p.Value {
        case '/': s = "/"
        case '.': s = "."
        case '^': s = ".."
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

// Strval returns the relative filename (aka. Project.SearchFile).
func (p *File) Strval() string { return filepath.Join(p.Sub, p.Name) }
func (p *File) String() string { return p.Fullname() }

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

func (p *File) prepare(pc *Preparer) error {
        if trace_prepare {
                fmt.Printf("prepare:File: %v (%v) (project %v, %v)\n", p.Name, p, pc.program.project.name, pc.entry)
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
                        fmt.Printf("prepare:File: %v (%v explicitly) (project %s, %v)\n", p.Name, p, project.name, pc.entry)
                }
                // Find concrete entry (by file represented name)
                if _, obj := project.scope.Find(p.Name); obj != nil {
                        if trace_prepare {
                                fmt.Printf("prepare:File: %v (found %v) (project %v, %v)\n", p.Name, obj, project.name, pc.entry)
                        }
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
                        fmt.Printf("prepare:File: %v (%v implicitly) (project %s, %v)\n", p.Name, p, project.name, pc.entry)
                }
                for _, ps := range project.FindPatterns(p.Name) {
                        if trace_prepare {
                                fmt.Printf("prepare:File: %v (stemmed %v) (project %v, %v)\n", p.Name, ps, project.name, pc.entry)
                        }
                        ps.file = p // Bounds PatternStem with the File.
                        if err = ps.prepare(pc); err == nil {
                                trybrk = true; break // Updated successfully!
                        } else if _, ok := err.(patternPrepareError); ok {
                                if trace_prepare {
                                        fmt.Printf("prepare:File: %v (error: %s)\n", p.Name, err)
                                }
                        } else {
                                trybrk = true; break // Update failed!
                        }
                }
                return
        })
}

func (p *File) search(pc *Preparer) (error, bool) {
        return pc.forEachExternalCaller(func(project *Project) (trybrk bool, err error) {
                if p.IsKnown() {
                        if trace_prepare {
                                fmt.Printf("prepare:File: %v (known %v) (project %s, %v)\n",
                                        p.Name, p, project.name, pc.entry)
                        }
                        pc.targets.Append(p)
                        trybrk = true
                } else if f := project.SearchFile(p.Strval()); !f.IsKnown() {
                        if trace_prepare {
                                fmt.Printf("prepare:File: exists %v (%v) (project %s, %v)\n",
                                        f.Name, f, project.name, pc.entry)
                        }
                        pc.targets.Append(f)
                        trybrk = true
                } else {
                        if trace_prepare {
                                fmt.Printf("prepare:File: %v (%v) (unknown) (project %s, %v)\n",
                                        p.Name, p.Dir, project.name, pc.entry)
                        }
                        err = unknownFileError{ fmt.Errorf("unknown file '%v'", p.Name), p }
                }
                return
        })
}

type Flag struct {
        Name Value
}
func (p *Flag) String() (s string) {
        s = "-" + p.Name.String()
        return
}
func (p *Flag) Strval() string {
        return "-" + p.Name.Strval()
}
func (p *Flag) Integer() int64     { return 0 }
func (p *Flag) Float() float64     { return float64(p.Integer()) }
func (p *Flag) Type() Type         { return FlagType }

func (p *Flag) disclose(scope *Scope) (Value, error) {
        if name, err := p.Name.disclose(scope); err != nil {
                return nil, err
        } else if name != nil {
                return &Flag{ name }, nil
        }
        return nil, nil
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
func (p *Compound) Strval() (s string) {
        for _, e := range p.Elems {
                s += e.Strval()
        }
        return
}
func (p *Compound) Integer() int64 { return int64(len(p.Elems)) }
func (p *Compound) Float() float64 { return float64(p.Integer()) }
func (p *Compound) Type() Type     { return CompoundType }

func (p *Compound) disclose(scope *Scope) (Value, error) {
        if elems, num, err := p.Elements.disclose(scope); err != nil {
                return nil, err
        } else if num > 0 {
                return &Compound{ Elements{ elems } }, nil
        }
        return nil, nil
}

type List struct {
        Elements
}
func (p *List) String() (s string) {
        for i, e := range p.Elems {
                if 0 < i {
                        s += " "
                }
                s += e.String()
        }
        return
}
func (p *List) Strval() (s string) {
        var x = 0
        for _, e := range p.Elems {
                if v := e.Strval(); v != "" {
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

func (p *List) disclose(scope *Scope) (Value, error) {
        if elems, num, err := p.Elements.disclose(scope); err != nil {
                return nil, err
        } else if num > 0 {
                return &List{ Elements{ elems } }, nil
        }
        return nil, nil
}

type Group struct {
        List
}
func (p *Group) String() string {
        return "(" + p.List.String() + ")"
}
func (p *Group) Strval() string {
        return "(" + p.List.Strval() + ")"
}
func (p *Group) Type() Type { return GroupType }

func (p *Group) disclose(scope *Scope) (Value, error) {
        if elems, num, err := p.Elements.disclose(scope); err != nil {
                return nil, err
        } else if num > 0 {
                return &Group{ List{ Elements{ elems } } }, nil
        }
        return nil, nil
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
func (p *Pair) String() string {
        return p.Key.String() + "=" + p.Value.String()
}
func (p *Pair) Strval() string {
        return p.Key.Strval() + "=" + p.Value.Strval()
}
func (p *Pair) Integer() int64     { return p.Value.Integer() }
func (p *Pair) Float() float64     { return p.Value.Float() }
func (p *Pair) Type() Type         { return PairType }

func (p *Pair) SetValue(v Value)   { p.Value = v }
func (p *Pair) SetKey(k Value) {
        switch o := k.(type) {
        case *Pair:   k = o.Key
        }
        if k.Type().Bits()&IsKeyName != 0 {
                p.Key = k
        } else {
                panic(fmt.Errorf("'%T' is not key type", k))
        }
}

func (p *Pair) disclose(scope *Scope) (Value, error) {
        if k, err := p.Key.disclose(scope); err != nil {
                return nil, err
        } else if v, err := p.Value.disclose(scope); err != nil {
                return nil, err
        } else if k != nil || v != nil {
                if k == nil { k = p.Key }
                if v == nil { v = p.Value }
                return &Pair{ k, v }, nil
        }
        return nil, nil
}

func (p *Pair) referencing(o Object) bool {
        return p.Key.referencing(o) || p.Value.referencing(o)
}

// Delegate wraps '$(foo a,b,c)' into Valuer
type delegate struct {
        o Object
        a []Value
        dc *Scope // disclosed context
}
func (p *delegate) Type() Type         { return DelegateType }
func (p *delegate) String() (s string) {
        var na = len(p.a)
        s = "$"
        s += "(" //if na > 0 { s += "(" }
        if false {
                if sc := p.o.Parent(); sc != nil && sc.Comment() == "use"/*use scope*/ {
                        s += sc.Comment() + "->"
                } else if pp := p.o.Project(); pp != nil {
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
func (p *delegate) Strval() string   { return p.Value().Strval() }
func (p *delegate) Integer() int64   { return p.Value().Integer() }
func (p *delegate) Float() float64   { return p.Value().Float() }
func (p *delegate) Value() (res Value) {
        switch o := p.o.(type) {
        case Caller:
                if args, err := p.discloseArgs(p.o.Parent()); err == nil {
                        res, _ = o.Call(args...)
                }
        case Executer:
                // FIXME: disclosed context not applied?
                var scope = p.dc
                if scope == nil {
                        scope = p.o.Parent()
                }
                if args, err := p.discloseArgs(scope); err == nil {
                        if v, err := o.Execute(args...); err == nil {
                                res = &List{Elements{v}}
                        }
                }
        default:
                fmt.Printf("delegate.value: unknown (%T %v)\n", p.o, p.o)
        }
        if res == nil {
                res = UniversalNone
        }
        return
}

func (p *delegate) disclose(scope *Scope) (Value, error) {
        var (
                o Object
                a []Value
                n = 0
        )

        if v, e := p.o.disclose(scope); e != nil {
                return nil, e
        } else if v != nil {
                o, _ = v.(Object)
        } else {
                o = p.o
        }

        //fmt.Printf("delegate.disclose: %T -> %T\n", p.o, o)

        for _, t := range p.a {
                //fmt.Printf("delegate.disclose: a: %T %v\n", t, t)
                if v, e := t.disclose(scope); e != nil {
                        return nil, e
                } else if v != nil {
                        t, n = v, n+1
                }
                a = append(a, t)
        }

        if o != nil || n > 0 {
                return &delegate{ o, a, scope }, nil
        }
        return nil, nil
}

func (p *delegate) discloseArgs(scope *Scope) (args []Value, err error) {
        for _, a := range p.a {
                if v, e := Disclose(scope, a); e != nil {
                        // TODO: errors...
                        return nil, e
                } else if v != nil {
                        //fmt.Printf("delegate.value: %v -> %v\n", a, v)
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

func (p *delegate) prepare(pc *Preparer) (err error) {
        if trace_prepare {
                fmt.Printf("prepare:delegate: %v (project %v, %v)\n", p, pc.entry.project.name, pc.entry)
        }
        for _, d := range Join(Reveal(p)) {
                if err = pc.Prepare(d); err != nil {
                        break
                }
        }
        return
}

type closure struct {
        o Object
        a []Value
}

func (p *closure) Type() Type { return ClosureType }
func (p *closure) String() (s string) {
        var na = len(p.a)
        s = "&"
        s += "(" //if na > 0 { s += "(" }
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
func (p *closure) Strval() string {
        if o, _ := p.o.(Caller); o != nil {
                if v, e := o.Call(/* No arguments! */); e == nil {
                        return v.Strval()
                }
        }
        return p.o.Strval() 
}
func (p *closure) Integer() int64       { return p.o.Integer() }
func (p *closure) Float() float64       { return p.o.Float() }
func (p *closure) disclose(scope *Scope) (Value, error) {
        //fmt.Printf("closure.disclose: %T %v\n", p.o, p.o)
        var obj = p.o
        if _, o := scope.Find(p.o.Name()); o != nil {
                obj = o
        }

        // Disclose the p.o, it's value may have disclosures.
        if v, e := obj.disclose(scope); e != nil {
                return nil, e
        } else if o, _ := v.(Object); o != nil {
                obj = o
        }

        var (
                //scope = p.o.Parent()
                args []Value
        )
        for _, a := range p.a {
                if v, e := a.disclose(scope); e != nil {
                        return nil, e
                } else if v != nil {
                        //fmt.Printf("delegate.value: %v -> %v\n", a, v)
                        a = v
                }
                args = append(args, a)
        }

        switch o := obj.(type) {
        case Caller:
                return o.Call(args...)
        case Executer:
                if result, err := o.Execute(args...); err == nil {
                        return &List{Elements{result}}, nil
                } else {
                        return nil, err
                }
        default:
                err := fmt.Errorf("Unsupported closure object `%T' (%v)", obj, obj)
                return nil, err
        }
        return nil, nil
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
        if v, e := p.disclose(pc.disctx()); e != nil {
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
func (*Plain) disclose(_ *Scope) (Value, error) { return nil, nil }
func (*Plain) referencing(_ Object) bool { return false }
func (p *Plain) Type() Type  { return PlainType }
func (p *Plain) String() string {
        s := "(plain"
        if p.Name != "" {
                s += "(" + p.Name + ")"
        } 
        s += " " + p.Value + ")"
        return s
}
func (p *Plain) Strval() string   { return p.Value }
func (p *Plain) Integer() int64   { i, _ := strconv.ParseInt(p.Value, 10, 64); return i }
func (p *Plain) Float() float64   { f, _ := strconv.ParseFloat(p.Value, 64); return f }

type JSON struct {
        Value Value
}
func (*JSON) disclose(_ *Scope) (Value, error) { return nil, nil }
func (*JSON) referencing(_ Object) bool { return false }
func (p *JSON) Type() Type { return JSONType }
func (p *JSON) String() string { return "(json " + p.Value.String() + ")" }
func (p *JSON) Strval() string { return p.Value.Strval() }
func (p *JSON) Integer() int64 { return 0 }
func (p *JSON) Float() float64 { return 0 }

type XML struct {
        Value Value
}
func (*XML) disclose(_ *Scope) (Value, error) { return nil, nil }
func (*XML) referencing(_ Object) bool { return false }
func (p *XML) Type() Type { return XMLType }
func (p *XML) String() string { return "(json " + p.Value.String() + ")" }
func (p *XML) Strval() string { return p.Value.Strval() }
func (p *XML) Integer() int64 { return 0 }
func (p *XML) Float() float64 { return 0 }

type YAML struct {
        Value Value
}
func (*YAML) disclose(_ *Scope) (Value, error) { return nil, nil }
func (*YAML) referencing(_ Object) bool { return false }
func (p *YAML) Type() Type { return YAMLType }
func (p *YAML) String() string { return "(json " + p.Value.String() + ")" }
func (p *YAML) Strval() string { return p.Value.Strval() }
func (p *YAML) Integer() int64 { return 0 }
func (p *YAML) Float() float64 { return 0 }

type ExecBuffer struct {
        Tie io.Writer
        Buf *bytes.Buffer
}

func (p *ExecBuffer) Write(b []byte) (n int, err error) {
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
func (p *ExecResult) Integer() int64 { return int64(p.Status) }
func (p *ExecResult) Float() float64 { return float64(p.Status) }
func (p *ExecResult) Strval() (s string) {
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
        Match(s string) (matched bool, stem string)
}

type pattern struct {
}

func (p *pattern) Type() Type        { return PatternType }
func (p *pattern) Integer() int64    { return 0 }
func (p *pattern) Float() float64    { return 0 }
func (p *pattern) makeEntry(patent *RuleEntry, name, stem string) (entry *RuleEntry, err error) {
        if patent.class == GlobRuleEntry {
                entry = new(RuleEntry); *entry = *patent
                entry.name = name
                entry.stem = stem
                if patent.project.isFile(name) {
                        entry.class = StemmedFileEntry
                        if false && entry.file == nil {
                                entry.file = entry.project.SearchFile(name)
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

func (p *GlobPattern) Pos() *token.Position { return nil }
func (p *GlobPattern) String() string { return p.Strval() }
func (p *GlobPattern) Strval() (s string) {
        if p.Prefix != nil {
                s = p.Prefix.Strval()
        }
        s += "%"
        if p.Suffix != nil {
                s += p.Suffix.Strval()
        }
        return
}
func (p *GlobPattern) Match(s string) (matched bool, stem string) {
        if prefix := p.Prefix.Strval(); prefix == "" || strings.HasPrefix(s, prefix) {
                if suffix := p.Suffix.Strval(); suffix == "" || strings.HasSuffix(s, suffix) {
                        if a, b := len(prefix), len(s)-len(suffix); a < b {
                                matched, stem = true, s[a:b]
                        }
                }
        }
        return
}

func (p *GlobPattern) MakeString(stem string) string {
        return p.Prefix.Strval() + stem + p.Suffix.Strval()
}

func (p *GlobPattern) MakeConcreteEntry(patent *RuleEntry, stem string) (entry *RuleEntry, err error) {
        name := p.MakeString(stem)
        return p.makeEntry(patent, name, stem)
}

func (p *GlobPattern) referencing(o Object) bool {
        return p.Prefix.referencing(o) || p.Suffix.referencing(o)
}

func (p *GlobPattern) prepare(pc *Preparer) (err error) {
        if trace_prepare {
                fmt.Printf("prepare:GlobPattern: %v (%v) (stem: %v) (project %v, %v)\n", p, pc.entry.file, pc.stem, pc.entry.project.name, pc.entry)
        }
        if pc.stem == "" {
                err = fmt.Errorf("empty stem (%s, %v)", p, pc.entry)
                return
        }

        var target = p.MakeString(pc.stem)
        
        // Check if target is a file (if source entry is file).
        if f := pc.entry.file; f != nil {
                var file = &File{ Name: target, Dir: f.Dir, Sub: f.Sub }
                if trace_prepare {
                        fmt.Printf("prepare:GlobPattern: %v (%v) (stem: %v) (project %v, %v)\n", p, file, pc.stem, pc.entry.project.name, pc.entry)
                }
                if err = file.prepare(pc); err == nil {
                        return
                } else {
                        fmt.Fprintf(os.Stdout, "%s: %v\n", pc.entry.Position, err)
                        if trace_prepare {
                                fmt.Printf("prepare:GlobPattern: %v (%v) (error: %v) (%v) (project %v, %v)\n", p, pc.stem, err, file, pc.entry.project.name, pc.entry)
                        }
                }
        }

        if trace_prepare {
                fmt.Printf("prepare:GlobPattern: %v (stem: %v) (%v) (project %v, %v)\n", p, pc.stem, target, pc.entry.project.name, pc.entry)
        }

        if err = pc.prepareTarget(target); err == nil {
                return // Good!
        } else if ute, ok := err.(unknownTargetError); ok {
                if trace_prepare {
                        fmt.Printf("prepare:GlobPattern: %v (FIXME: unknown target %v) (%v) (%v)\n", p, ute.target, pc.stem, pc.entry)
                }
                //err = patternPrepareError(ute.error)
        } else if ufe, ok := err.(unknownFileError); ok {
                if trace_prepare {
                        fmt.Printf("prepare:GlobPattern: %v (FIXME: unknown file %v) (%v) (%v)\n", p, ufe.file, pc.stem, pc.entry)
                }
                //err = patternPrepareError(ute.error)
        } else {
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

func (p *RegexpPattern) Pos() *token.Position { return nil }
func (p *RegexpPattern) String() string { return p.Strval() }
func (p *RegexpPattern) Strval() (s string) { return "" }
func (p *RegexpPattern) Match(s string) (matched bool, stem string) {
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
        Call(args... Value) (Value, error)
}

type Executer interface {
        Execute(a... Value) (result []Value, err error)
}

type Poser interface {
        Pos() *token.Position
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

// Reveal expends delegates and Valuer recursively.
func Reveal(v Value) (res Value) {
        switch t := v.(type) {
        case *List:
                var (
                        elems []Value
                        num = 0 // number of revealed elements
                )
                for i, elem := range t.Elems {
                        elems = append(elems, Reveal(elem))
                        if elems[i] != t.Elems[i] {
                                num += 1
                        }
                }
                if num > 0 {
                        res = &List{Elements{elems}}
                } else {
                        res = t
                }
        case Valuer:
                res = Reveal(t.Value())
        default:
                res = v
        }
        return
}

// Disclose expends closures to normal value recursively.
func Disclose(scope *Scope, value Value) (Value, error) {
        if false {
                fmt.Printf("Disclose: %T %v\n", value, value)
        }
        if v, err := value.disclose(scope); err != nil {
                return nil, err
        } else if v != nil {
                value = v
        }
        return value, nil
}

// Join combines lists recursively into one list.
func Join(args... Value) (elems []Value) {
        for _, arg := range args {
                switch t := arg.(type) {
                case *List:
                        for _, elem := range t.Elems {
                                elems = append(elems, Join(elem)...)
                        }
                default:
                        elems = append(elems, t)
                }
        }
        return
}

// JoinReveal join revealed elements into one list.
func JoinReveal(args... Value) (elems []Value) {
        for _, elem := range args {
                elems = append(elems, Join(Reveal(elem))...)
        }
        return
}

// JoinEval join evaluated (disclosed and revealed) elements into one list.
func JoinEval(scope *Scope, args... Value) (elems []Value, err error) {
        for _, elem := range args {
                if elem, err = Disclose(scope, elem); err != nil {
                        break
                }
                elems = append(elems, Join(Reveal(elem))...)
        }
        return
}

func Delegate(obj Object, args... Value) Value {
        return &delegate{ obj, args, nil }
}

func Closure(obj Object, args... Value) Value {
        if obj == nil {
                panic("closure of nil")
        }
        return &closure{ obj, args }
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
