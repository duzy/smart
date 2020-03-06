//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/token"
        "crypto/sha256"
        "path/filepath"
        "runtime/debug" // debug.PrintStack()
        "net/url"
        "reflect"
        "strconv"
        "strings"
        "bytes"
        "sync"
        "time"
        "math"
        "fmt"
        "os"
)

const (
        enable_assertions = true
        enable_grep_bench = true
)

type HashBytes [sha256.Size]byte

type (
        cmpres int
        existence int
        expandwhat int
)
const (
        cmpUnknown cmpres = 0
        cmpSmaller     = -1 // meaningless so far
        cmpGreater     = 1  // meaningless so far
        cmpEqual       = 2
)
const (
        existenceMatterless existence = 1<<iota
        existenceConfirmed
        existenceNegated
)
const (
        expandDelegate expandwhat = 1<<iota // $(...)  ->  ......
        expandClosure // &(...)   ->  $(...)
        expandCaller // foo=...   ->  ...
        expandPath // $(...)/foo  ->  /path/to/foo
        expandAll = expandDelegate | expandClosure | expandCaller | expandPath
)

func (v cmpres) String() (s string) {
        switch v {
        case cmpUnknown: s = "unknown"
        case cmpSmaller: s = "smaller"
        case cmpGreater: s = "greater"
        case cmpEqual:   s = "equal"
        }
        return
}

func (v existence) String() (s string) {
        switch v {
        case existenceMatterless: s = "matterless"
        case existenceConfirmed:  s = "confirmed"
        case existenceNegated:    s = "negated"
        }
        return
}

// Value represents a value of a type.
type Value interface {
        Positioner // The position where the value appears (or NoPos).

        // Lit returns the literal representations of the value.
        String() string

        // Strval returns the string form of the value.
        Strval() (string, error)

        // Integer returns the integer form of the value.
        Integer() (int64, error)

        // Float returns the float form of the value.
        Float() (float64, error)

        // Returns true if the value can be evaluated as 'true', 'yes', etc.
        True() (bool, error)

        // Equality compare.
        cmp(v Value) cmpres

        // Returns the modification time.
        mod(t *traversal) (time.Time, error)

        // Returns value existence (as a target)
        exists() existence

        // Stamp the value if it's file (update FileInfo).
        stamp(t *traversal) ([]*File, error)

        // Recursively detecting whether this value references
        // the object (to avoid loop-delegation).
        refs(v Value) bool

        closured() bool
        refdef(origin DefOrigin) bool

        // &(...) -> $(...)
        // $(...) -> ......
        expand(what expandwhat) (Value, error)

        traverse(t *traversal) error
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

type updatedtarget struct {
        target Value
        prerequisites []*updatedtarget
}

func (p *updatedtarget) String() string {
        if len(p.prerequisites) > 0 {
                return fmt.Sprintf("%v→%v", p.target, p.prerequisites)
        }
        return p.target.String()
}

func newUpdatedTarget(target Value, prerequisites ...*updatedtarget) *updatedtarget {
        if def, ok := target.(*Def); ok { target = def.value }
        return &updatedtarget{target, prerequisites}
}

// traversal prepares prerequisites of targets.
type traversal struct {
        program *Program
        project *Project // program.project or caller.project (if (closure))
        closure *Scope // program.scope or caller.closure (if (closure))

        def struct {
                params []*Def // $0, $1, $2, ...
                target   *Def // $@
                depends  *Def // $^
                depend0  *Def // $<
                ordered  *Def // $|
                grepped  *Def // $~
                updated  *Def // $?
                buffer   *Def // $-
                stem     *Def // $*
        }

        visited map[Value]int

        group *sync.WaitGroup
        caller *traversal
        calleeErrs []error
        calleeErrsM sync.Mutex

        entry *RuleEntry // caller entry (target)
        args, arguments []Value // target and argumented prerequisite args

        target0 *Def
        targets *Def
        grepped []Value

        updated []*updatedtarget // prerequisites newer than the target (from comparer) ($?)
        stems   []string // set by StemmedEntry

        traceLevel int

        breakers []*breaker
        interpreted []interpreter

        print bool // printing work directories (Entering/Leaving)
        debug bool
}

// Usage pattern: defer un(tt(pc, "..."))
func tt(t *traversal, i Value) *traversal {
        // Note that t.args and t.arguments are different, they're
        // target execution args and argumented-prerequisite args.
        var a string
        if tar := t.entry.target; len(t.args) > 0 {
                a = fmt.Sprintf("%s{%s}%s", typeof(tar), tar, t.args)
        } else {
                a = fmt.Sprintf("%s{%v}", typeof(tar), tar)
        }
        var b = fmt.Sprintf("%s{%v}", typeof(i), i)
        t.trace(a, ":", b, "(")
        t.level(+1)
        return t
}
func (t *traversal) level(n int) { t.traceLevel += n }
func (t *traversal) trace(a ...interface{}) { printIndentDots(t.traceLevel, a...) }
func (t *traversal) tracef(s string, a ...interface{}) { printIndentDots(t.traceLevel, fmt.Sprintf(s, a...)) }

func (t *traversal) addNewTarget(target Value) {
        if isNil(target) || isNone(target) { return }
        if t.targets.value == target { return }
        if t.targets.value.cmp(target) == cmpEqual { return }
        if targets, ok := t.targets.value.(*List); ok {
                for _, t := range targets.Elems {
                        if t == target || t.cmp(target) == cmpEqual { return }
                }
        }
        t.targets.append(target)
        if t.target0 != nil && (isNone(t.target0.value) || isNil(t.target0.value)) {
                t.target0.value = target
        }
}

func (t *traversal) depth() (res int) {
        for c := t.caller; c != nil; c = c.caller { res += 1 }
        return
}

func (t *traversal) calleeStart() {
        t.group.Add(1)
}

func (t *traversal) calleeDone(err error) {
        if err != nil { t.calleeError(err) }
        t.group.Done()
}

func (t *traversal) calleeError(err error) {
        t.calleeErrsM.Lock(); defer t.calleeErrsM.Unlock()
        t.calleeErrs = append(t.calleeErrs, err)
}

func (t *traversal) calleeErrors() (errs []error) {
        t.calleeErrsM.Lock(); defer t.calleeErrsM.Unlock()
        errs = t.calleeErrs
        return
}

func (t *traversal) dispatch(i interface{}) (err error) {
        if optionEnableBenchmarks && false {  defer bench(mark(fmt.Sprintf("traversal.dispatch(%s=%v)", typeof(i), i))) }

        var pos = t.def.target.position
        if v := reflect.ValueOf(i); v.Kind() == reflect.Slice {
                for n := 0; err == nil && n < v.Len(); n++ {
                        if optionEnableBenchmarks && false {
                                i := v.Index(n).Interface()
                                a, b := mark(fmt.Sprintf("%v: %s %v", n, typeof(i), i))
                                err = t.dispatch(i)
                                bench(a, b)
                        } else {
                                err = t.dispatch(v.Index(n).Interface())
                        }
                }
        } else if i == nil {
                err = errorf(pos, "updating nil prerequisite")
        } else if value, ok := i.(Value); !ok {
                err = errorf(pos, "'%v' is invalid", value)
        } else if isNil(value) { // this could happen
                err = errorf(pos, "updating nil prerequisite")
        } else {
                if false { fmt.Fprintf(stderr, "dispatch: %T %v\n", value, value) }
                err = value.traverse(t)
        }
        return
}

func (t *traversal) filestub(p *Project, file *File, stub *filestub) (okay bool, err error) {
        if optionEnableBenchspots { defer bench(spot("traversal.filestub")) }

        /// Searching entries from the most derived project.
        var entry *RuleEntry
        if entry, err = p.resolveEntry(stub.name); err != nil { return } else
        if entry != nil { err, okay = entry.traverse(t), true;  return }

        /// Searching patterns from the most derived project.
        var entries []*StemmedEntry
        if entries, err = p.resolvePatterns(stub); err != nil { return }
        ForEntries: for _, entry := range entries {
                for _, prog := range entry.programs {
                        var okay bool
                        if  okay, err = checkPatternDepends(t, p, entry, prog); err != nil { break ForEntries }
                        if !okay { continue ForEntries }
                }
                if err = entry.file(t, file); err == nil { okay = true }
                break
        }
        return
}

func (t *traversal) closureProjects() (projects []*Project) {
        projects = []*Project{ t.project }

        if t.program.project != t.project {
                projects = append(projects, t.program.project)
        }

        for c := t; c != nil; c = c.caller {
                if t.closure == c.closure {
                        var proj = c.project
                        for _, p := range projects {
                                if proj == p { proj = nil; break }
                        }
                        if proj != nil { projects = append(projects, proj) }
                }
        }
        return
}

func (t *traversal) forClosureProject(f func(*Project) (bool, error)) (okay bool, err error) {
        var projects = t.closureProjects()
        for _, proj := range projects {
                if okay, err = f(proj); okay || err != nil { break }
        }
        return
}

func (t *traversal) file(file *File) (err error) {
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("traversal.file(%v)", file))) }
        if optionEnableBenchspots { defer bench(spot("traversal.file")) }

        var okay bool
        var projects = t.closureProjects()
        for _, project := range projects {
                var entry *RuleEntry
                if entry, err = project.resolveEntry(file.name); err != nil { err = wrap(file.position, err); return }
                if entry != nil { //if err = t.dispatch(entry); err != nil { err = wrap(file.position, err) }; return }
                        if okay, err = entry.tryTraverse(t); okay { return } else
                        if err != nil { err = wrap(file.position, err); return }
                }
        }

        for _, project := range projects {
                var entries []*StemmedEntry
                if entries, err = project.resolvePatterns(file.name); err != nil { err = wrap(file.position, err); return }
                ForEntry: for _, entry := range entries {
                        for _, prog := range entry.programs {
                                var good bool
                                if good, err = checkPatternDepends(t, project, entry, prog); err != nil { err = wrap(file.position, err); break ForEntry }
                                if!good { continue ForEntry }
                        }
                        if err = entry.file(t, file); err == nil {
                                okay = true // entry executed
                                break ForEntry
                        } else {
                                err = wrap(file.position, err)
                                return
                        }
                }

                if okay { break } else if exists(file) { okay = true } else
                if file != nil { okay = file.searchInMatchedPaths(project) } else
                if alt := project.matchFile(file.name); alt != nil { okay = exists(alt) }
                if!okay && false {
                        s, _ := file.Strval()
                        e, _ := project.resolveEntry(file.name)
                        fmt.Fprintf(stderr, "%s: %s: %v (%v) (%s)\n", project, file.position, file, e, s)
                        debug.PrintStack()
                }
                if okay { return }
        }

        if !okay && err == nil {
                err = wrap(file.position, fileNotFoundError{t.project, file})
                if optionTraceTraversal { t.tracef("%v: file({%s,%s,%s}): not found", t.project, file.dir, file.sub, file.name) }
        }
        return
}

func (t *traversal) target(pos Position, target string) (err error) {
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("traversal.target(%v)", target))) }
        if optionEnableBenchspots { defer bench(spot("traversal.target")) }

        var okay bool
        var projects = t.closureProjects()
        for _, project := range projects {
                var entry *RuleEntry
                if entry, err = project.resolveEntry(target); err != nil { err = wrap(pos, err); return }
                if entry != nil { if err = entry.traverse(t); err != nil { err = wrap(pos, err); }; return }

                var obj Object
                if obj, err = project.resolveObject(target); err != nil { err = wrap(pos, err); return } else
                if obj != nil {
                        if okay, err = obj.tryTraverse(t); err != nil { err = wrap(pos, err); return } else
                        if okay { return }
                }

                if file := project.matchFile(target); file != nil {
                        file.position = pos // Change the position for tracing
                        t.addNewTarget(file) // Add new file target

                        var names = make(map[string]bool)
                        for stub := file.filestub; true; stub = stub.other {
                                names[stub.name] = true // mark to avoid trying many times
                                if okay, err = t.filestub(project, file, stub); err != nil { err = wrap(pos, err); return }
                                if okay { file.filestub = stub; return }
                                if stub.other == file.filestub { break }
                        }

                        // Try other names
                        var name string
                        for s, i := file.name, strings.LastIndex(file.name, PathSep); s != "" && i >= 0; i = strings.LastIndex(s, PathSep) {
                                if i == 0 { name = file.fullname() } else { name = filepath.Join(s[i+1:], name) }
                                s = s[:i] // strip off the prefix

                                if _, tried := names[name]; tried { continue }
                                names[name] = true // mark to avoid duplication

                                var sub = filepath.Join(file.sub, s)
                                var stub = &filestub{ file.dir, sub, name, file.match, file.filestub.other }
                                file.filestub.other = stub

                                if okay, err = t.filestub(project, file, stub); err != nil { err = wrap(pos, err); return }
                                if okay { file.filestub = stub; break }
                        }

                        // Check file existance
                        if okay { break } else
                        if exists(file) { okay = true } else
                        if file != nil { okay = file.searchInMatchedPaths(project) }
                        if!okay { err = wrap(pos, fileNotFoundError{project, file}) }
                        return
                }
        }

        for _, project := range projects {
                var entries []*StemmedEntry
                if entries, err = project.resolvePatterns(target); err != nil { err = wrap(pos, err); return }
                ForEntry: for _, entry := range entries {
                        for _, prog := range entry.programs {
                                var good bool
                                if good, err = checkPatternDepends(t, project, entry, prog); err != nil { err = wrap(pos, err); break ForEntry }
                                if!good { continue ForEntry }
                        }

                        // Associate StemmedEntry with the target.
                        if err = entry._target(t, target); err == nil { okay = true; return }
                }
        }

        if !okay && err == nil {
                err = wrap(pos, targetNotFoundError{t.project, target})
                if optionTraceTraversal { t.tracef("%v: `target(%s)` not found", t.project, target) }
        }

        if false && err != nil { debug.PrintStack() }
        return
}

func (t *traversal) appendUpdated(updated *updatedtarget) {
        if t.def.target.value == updated.target { return }
        if t.def.target.value.cmp(updated.target) == cmpEqual { return }
        for _, u := range t.updated { // check if already added
                if u.target == updated.target { return }
                if u.target.cmp(updated.target) == cmpEqual { return }
        }
        t.updated = append(t.updated, updated)
        for c := t.caller; c != nil; c = c.caller { // clear update loop
                if false {
                        if c.def.target.value == t.def.target.value { return }
                } else {
                        if c.def.target.value == updated.target { return }
                }
        }
        if c := t.caller; c != nil {
                if false && updated.target.String() == "..." {
                        var (s string; m time.Time)
                        m, _ = updated.target.mod(t)
                        s, _ = updated.target.Strval()
                        fmt.Fprintf(stderr, "%s:\t%v %v\n", updated.target.Position(), m, s)
                        m, _ = t.def.target.value.mod(t)
                        s, _ = t.def.target.value.Strval()
                        fmt.Fprintf(stderr, "%s:\t%v %v\n", t.def.target.value.Position(), m, s)
                }
                c.appendUpdated(newUpdatedTarget(t.def.target.value, updated))
        }
}

func (t *traversal) removeUpdated(target Value) (removed []*updatedtarget) {
        for i, u := range t.updated {
                if u.target == target || u.target.cmp(target) == cmpEqual {
                        removed = append(removed, u)
                        t.updated = append(t.updated[:i], t.updated[i+1:]...)
                        if t.caller != nil && len(t.updated) == 0 {
                                t.caller.removeUpdated(t.def.target.value)
                        }
                }
        }
        return
}

func (t *traversal) removeCallerUpdated(target Value) {
        if t.caller != nil {
                // if strings.HasSuffix(target.String(), "...") { fmt.Fprintf(stderr, "%v: %v %v %v\n", target.Position(), target, t.updated, t.caller.updated) }
                for _, u := range t.caller.removeUpdated(target) {
                        for _, uu := range u.prerequisites {
                                t.removeUpdated(uu.target)
                        }
                }
                // if strings.HasSuffix(target.String(), "...") { fmt.Fprintf(stderr, "%v: %v %v %v\n", target.Position(), target, t.updated, t.caller.updated) }
        }
}

func (t *traversal) hashDir(k []byte) string {
        dir := t.program.project.tmpPath
        h := fmt.Sprintf("%x", k[:2]) // HEX of the first two bytes
        return filepath.Join(dir, ".hash", h[0:1], h[1:2], h[2:3], h[3:])
}

func (t *traversal) cmdHash(values ...Value) (k, v HashBytes, err error) {
        var (
                key = sha256.New()
                val = sha256.New()
                str string
        )
        if str, err = t.def.target.value.Strval(); err != nil { return }
        fmt.Fprintf(key, "%s", t.program.project.absPath)
        fmt.Fprintf(key, "%v", str)

        for _, value := range values {
                if false {
                        // FIXME: Strval() varies when &(var) is used
                        if str, err = value.Strval(); err != nil { return }
                        fmt.Fprintf(val, "%v", str)
                } else {
                        fmt.Fprintf(val, "%v", value)
                }
        }
        copy(k[:], key.Sum(nil))
        copy(v[:], val.Sum(nil))
        return
}

func (t *traversal) updateRecipesHash() (k, v HashBytes, err error) {
        if k, v, err = t.cmdHash(t.program.recipes...); err != nil {
                return
        }

        var dir = t.hashDir(k[:])
        var name = filepath.Join(dir, fmt.Sprintf("%x", k))
        if f, e := os.Open(name); e == nil {
                defer f.Close()

                var h []byte
                if n, e := fmt.Fscanf(f, "%x", &h); e != nil {
                        err = e; return
                } else if n == 1 && bytes.Equal(v[:], h) {
                        return
                }
        }

        if err = os.MkdirAll(dir, 0700); err != nil {
                return
        } else if f, e := os.Create(name); e == nil {
                defer f.Close()
                _, err = fmt.Fprintf(f, "%x", v)
        } else {
                err = e
        }
        return
}

func (t *traversal) isRecipesDirty() (dirty bool, err error) {
        var k, v HashBytes
        if k, v, err = t.cmdHash(t.program.recipes...); err != nil {
                return
        }

        var dir = t.hashDir(k[:])
        var name = filepath.Join(dir, fmt.Sprintf("%x", k))
        if f, e := os.Open(name); e == nil {
                defer f.Close()

                var h []byte
                if n, e := fmt.Fscanf(f, "%x", &h); e != nil {
                        err = e
                } else if n == 1 {
                        dirty = !bytes.Equal(v[:], h)
                }
        }
        return
}

func (t *traversal) wait(pos Position) (err error) {
        if optionEnableBenchmarks && false { defer bench(mark("traversal.wait")) }
        t.group.Wait()
        if e := t.calleeErrors(); len(e) > 0 {
                err = wrap(pos, e...)
        }
        return
}

type elemkind int
const (
        elemNoQuote elemkind = 1<<iota
        elemNoBrace
        elemExpand
)

type elemstrer interface {
        elemstr(o Object, k elemkind) string
}

func elementString(o Object, elem Value, k elemkind) (s string) {
        if p, ok := elem.(elemstrer); ok { s = p.elemstr(o, k) } else
        if elem != nil { s = elem.String() }
        return
}

type trivial struct { position Position }
func (_ *trivial) refs(_ Value) (res bool) { return }
func (_ *trivial) closured() (res bool) { return }
func (_ *trivial) refdef(origin DefOrigin) (res bool) { return }
func (_ *trivial) expand(_ expandwhat) (v Value, err error) { return }
func (_ *trivial) cmp(_ Value) (res cmpres) { return }
func (_ *trivial) mod(t *traversal) (res time.Time, err error) { return }
func (_ *trivial) exists() existence { return existenceMatterless }
func (_ *trivial) stamp(t *traversal) (file []*File, err error) { return }
func (t *trivial) Position() (res Position) { return t.position }
func (_ *trivial) True() (res bool, err error) { return }
func (_ *trivial) Integer() (i int64, err error) { return }
func (_ *trivial) Float() (f float64, err error) { return }
func (_ *trivial) String() (s string) { return }
func (_ *trivial) Strval() (s string, err error) { return }
func (_ *trivial) traverse(t *traversal) (err error) { return }

func exists(v Value) bool {
        // FIXME: returns true if existenceMatterless??
        return v != nil && v.exists() == existenceConfirmed
}

type Argumented struct {
        value Value
        args []Value
}
func (p *Argumented) refs(v Value) bool {
        if p.value.refs(v) { return true }
        for _, a := range p.args {
                if a.refs(v) { return true }
        }
        return false
}
func (p *Argumented) closured() bool {
        if p.value.closured() { return true }
        for _, a := range p.args {
                if a.closured() { return true }
        }
        return false
}
func (p *Argumented) refdef(origin DefOrigin) bool {
        return p.value.refdef(origin)
}
func (p *Argumented) expand(w expandwhat) (res Value, err error) {
        var ( v Value; args []Value )
        if v, err = p.value.expand(w); err == nil {
                if v != p.value {
                        var num int
                        args, num, err = expandall(w, p.args...)
                        if err == nil && (num > 0 || v != p.value) {
                                res = &Argumented{ v, args }
                        }
                }
        }
        if err == nil && res == nil { res = p }
        return
}
func (p *Argumented) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Argumented); ok {
                if res = p.value.cmp(a.value); res == cmpEqual {
                        // FIXME: check p.args against a.args?
                }
        }
        return
}
func (p *Argumented) stamp(t *traversal) ([]*File, error) { return p.value.stamp(t) }
func (p *Argumented) exists() existence { return p.value.exists() }
func (p *Argumented) mod(t *traversal) (time.Time, error) {
        // FIXME: p.value maybe not the real target (depending on the arguments)
        return p.value.mod(t)
}
func (p *Argumented) Position() Position { return p.value.Position() }
func (p *Argumented) True() (res bool, err error) {
        if p.value != nil { res, err = p.value.True() }
        return
}
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
func (p *Argumented) elemstr(o Object, k elemkind) (s string) {
        for i, a := range p.args {
                if i > 0 { s += "," }
                s += elementString(o, a, k)
        }
        s = fmt.Sprintf("%s(%s)", elementString(o, p.value, k), s)
        return
}
func (p *Argumented) String() (s string) { return p.elemstr(nil, 0) }
func (p *Argumented) Strval() (s string, err error) {
        if s, err = p.value.Strval(); err != nil {
                return
        }
        s += "("
        for i, a := range p.args {
                if i > 0 { s += "," }
                var v string
                if v, err = a.Strval(); err == nil { s += v } else {
                        break
                }
        }
        s += ")"
        return
}
func (p *Argumented) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        //!< IMPORTANT! - Don't merge-expand arguments here!
        //!< Arguments should be passed to Execute as it's
        //!< represented.
        defer func(a []Value) { t.arguments = a } (t.arguments)
        t.arguments = p.args
        err = t.dispatch(p.value)
        return
}
func (p *Argumented) checkPatternDepends(t *traversal, project *Project, se *StemmedEntry, prog *Program) (ok, res1 bool, err error) {
        switch v := p.value.(type) {
        case Pattern:
                ok = true
                res1, err = checkPatternDepend(t, project, se, prog, v)
        case *Argumented:
                ok, res1, err = v.checkPatternDepends(t, project, se, prog)
        }
        return
}

type None struct { trivial }
func (p *None) expand(_ expandwhat) (Value, error) { return p, nil }
func (_ *None) cmp(v Value) (res cmpres) { 
        if _, ok := v.(*None); ok { res = cmpEqual }
        return
}

type Nil struct { trivial }
func (p *Nil) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Nil) cmp(v Value) (res cmpres) {
        if _, ok := v.(*Nil); ok { res = cmpEqual }
        return
}

func isNone(v Value) (t bool) { _, t = v.(*None); return }
func isNil(v Value) (t bool) {
        if _, t = v.(*Nil); !t {
                var vv = reflect.ValueOf(v)
                if v == nil || (vv.Kind() == reflect.Ptr && vv.IsNil()) {
                        t = true
                }
        }
        return
}

// Any is used to box an arbitrary value
type Any struct { value interface{} }
func (p *Any) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Any); ok {
                assert(ok, "value is not Any")
                if v1, ok := p.value.(Value); ok {
                        if v2, ok := a.value.(Value); ok {
                                res = v1.cmp(v2)
                        }
                } else if p.value == a.value {
                        res = cmpEqual
                }
        }
        return
}
func (p *Any) stamp(t *traversal) (files []*File, err error) {
        if a, ok := p.value.(Value); ok { files, err = a.stamp(t) }
        return
}
func (p *Any) exists() existence {
        if a, ok := p.value.(Value); ok { return a.exists() }
        return existenceMatterless
}
func (p *Any) mod(t *traversal) (res time.Time, err error) {
        if a, ok := p.value.(Value); ok { res, err = a.mod(t) }
        return
}
func (p *Any) expand(w expandwhat) (res Value, err error) {
        if v, ok := p.value.(Value); ok {
                res, err = v.expand(w)
        } else {
                res = p
        }
        return
}
func (p *Any) refs(o Value) (res bool) {
        if v, ok := p.value.(Value); ok { res = v.refs(o) }
        return
}
func (p *Any) refdef(origin DefOrigin) (res bool) {
        if v, ok := p.value.(Value); ok { res = v.refdef(origin) }
        return
}
func (p *Any) closured() (res bool) {
        if v, ok := p.value.(Value); ok { res = v.closured() }
        return
}
func (p *Any) Position() (res Position) {
        if v, ok := p.value.(Positioner); ok { res = v.Position() }
        return
}
func (p *Any) True() (t bool, err error) {
        switch v := p.value.(type) {
        case Value:     t, err = v.True()
        case float32:   t = math.Abs(float64(v))-0 >= FloatEpsilon
        case float64:   t = math.Abs(v)-0 >= FloatEpsilon
        case int64:     t = v != 0
        case int:       t = v != 0
        case bool:      t = v
        }
        return
}
func (p *Any) Float() (res float64, err error) {
        switch v := p.value.(type) {
        case Value: res, err = v.Float()
        case float32: res = float64(v)
        case float64: res = v
        case int: res = float64(v)
        case int64: res = float64(v)
        case bool: if v { res = FloatEpsilon }
        }
        return
}
func (p *Any) Integer() (res int64, err error) {
        switch v := p.value.(type) {
        case Value: res, err = v.Integer()
        case float32: res = int64(v)
        case float64: res = int64(v)
        case int: res = int64(v)
        case int64: res = v
        case bool: if v { res = 1 }
        }
        return
}
func (p *Any) Strval() (s string, err error) {
        s = fmt.Sprintf("%v", p.value)
        return
}
func (p *Any) String() string { return fmt.Sprintf("<%v>", p.value) }
func (p *Any) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        if v, ok := p.value.(Value); ok {
                err = v.traverse(t)
        }
        return 
}

type negative struct { trivial; x Value }
func (p *negative) refs(o Value) bool { return p.x.refs(o) }
func (p *negative) closured() bool { return p.x.closured() }
func (p *negative) expand(w expandwhat) (res Value, err error) {
        var v Value
        if v, err = p.x.expand(w); err != nil { return }
        if v == p.x { res = p } else { res = &negative{p.trivial,v} }
        return
}
func (p *negative) cmp(v Value) (res cmpres) {
        if a, ok := v.(*negative); ok { res = p.x.cmp(a.x) }
        return
}
func (p *negative) True() (res bool, err error) {
        if p.x != nil { res, err = p.x.True() }
        if err == nil { res = !res }
        return
}
func (p *negative) elemstr(o Object, k elemkind) string { return `!`+elementString(o, p.x, k) }
func (p *negative) String() (s string) { return p.elemstr(nil, 0) }
func (p *negative) Strval() (s string, err error) {
        var t bool
        if t, err = p.x.True(); err == nil {
                s = fmt.Sprintf("%v", !t)
        }
        return
}
func (p *negative) Float() (res float64, err error) {
        var t bool
        if t, err = p.x.True(); err == nil && !t {
                res = FloatEpsilon
        }
        return
}
func (p *negative) Integer() (res int64, err error) {
        var t bool
        if t, err = p.x.True(); err == nil && !t {
                res = 1
        }
        return
}
func (p *negative) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        if p.x != nil { err = p.x.traverse(t) }
        return
}

func Negative(val Value) *negative { return &negative{trivial{val.Position()},val} }

type boolean struct { trivial; bool }
func (p *boolean) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *boolean) True() (bool, error) { return p.bool, nil }
func (p *boolean) Strval() (string, error) { return p.String(), nil }
func (p *boolean) String() (s string) {
        if p.bool { s = "true" } else { s = "false" }
        return
}
func (p *boolean) Float() (v float64, err error) {
        if p.bool { v = 1. }
        return
}
func (p *boolean) Integer() (v int64, err error) {
        if p.bool { v = 1 }
        return
}
func (p *boolean) cmp(v Value) (res cmpres) {
        if a, ok := v.(*option); ok {
                if p.bool == a.bool {
                        res = cmpEqual
                } else if !p.bool && a.bool {
                        res = cmpSmaller
                } else if p.bool && !a.bool {
                        res = cmpGreater
                }
        } else if a, ok := v.(*answer); ok {
                if p.bool == a.bool {
                        res = cmpEqual
                } else if !p.bool && a.bool {
                        res = cmpSmaller
                } else if p.bool && !a.bool {
                        res = cmpGreater
                }
        } else if a, ok := v.(*boolean); ok {
                if p.bool == a.bool {
                        res = cmpEqual
                } else if !p.bool && a.bool {
                        res = cmpSmaller
                } else if p.bool && !a.bool {
                        res = cmpGreater
                }
        }
        return
}

type answer struct { trivial; bool }
func (p *answer) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *answer) True() (bool, error) { return p.bool, nil }
func (p *answer) Strval() (string, error) { return p.String(), nil }
func (p *answer) String() (s string) {
        if p.bool { s = "yes" } else { s = "no" }
        return
}
func (p *answer) Float() (v float64, err error) {
        if p.bool { v = 1. }
        return
}
func (p *answer) Integer() (v int64, err error) {
        if p.bool { v = 1 }
        return
}
func (p *answer) cmp(v Value) (res cmpres) {
        if a, ok := v.(*option); ok {
                if p.bool == a.bool {
                        res = cmpEqual
                } else if !p.bool && a.bool {
                        res = cmpSmaller
                } else if p.bool && !a.bool {
                        res = cmpGreater
                }
        } else if a, ok := v.(*answer); ok {
                if p.bool == a.bool {
                        res = cmpEqual
                } else if !p.bool && a.bool {
                        res = cmpSmaller
                } else if p.bool && !a.bool {
                        res = cmpGreater
                }
        } else if a, ok := v.(*boolean); ok {
                if p.bool == a.bool {
                        res = cmpEqual
                } else if !p.bool && a.bool {
                        res = cmpSmaller
                } else if p.bool && !a.bool {
                        res = cmpGreater
                }
        }
        return
}

type option struct { trivial; bool }
func (p *option) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *option) True() (bool, error) { return p.bool, nil }
func (p *option) Strval() (string, error) { return p.String(), nil }
func (p *option) String() (s string) {
        if p.bool { s = "on" } else { s = "off" }
        return
}
func (p *option) Float() (v float64, err error) {
        if p.bool { v = 1. }
        return
}
func (p *option) Integer() (v int64, err error) {
        if p.bool { v = 1 }
        return
}
func (p *option) cmp(v Value) (res cmpres) {
        if a, ok := v.(*option); ok {
                if p.bool == a.bool {
                        res = cmpEqual
                } else if !p.bool && a.bool {
                        res = cmpSmaller
                } else if p.bool && !a.bool {
                        res = cmpGreater
                }
        } else if a, ok := v.(*answer); ok {
                if p.bool == a.bool {
                        res = cmpEqual
                } else if !p.bool && a.bool {
                        res = cmpSmaller
                } else if p.bool && !a.bool {
                        res = cmpGreater
                }
        } else if a, ok := v.(*boolean); ok {
                if p.bool == a.bool {
                        res = cmpEqual
                } else if !p.bool && a.bool {
                        res = cmpSmaller
                } else if p.bool && !a.bool {
                        res = cmpGreater
                }
        }
        return
}

type prediction struct {
        boolean
        reason string
}
func (p *prediction) expand(_ expandwhat) (Value, error) { return p, nil }

type integer struct {
        trivial
        int64
}
func (p *integer) True() (bool, error) { return p.int64 != 0, nil }
func (p *integer) Integer() (int64, error) { return p.int64, nil }
func (p *integer) Float() (float64, error) { return float64(p.int64), nil }
func (p *integer) cmp(v Value) (res cmpres) {
        i, e := v.Integer()
        assert(e == nil, "%T: %v", v, e)
        if p.int64 == i {
                res = cmpEqual
        } else if p.int64 < i {
                res = cmpSmaller
        } else if p.int64 > i {
                res = cmpGreater
        }
        return
}

type Bin struct { integer }
func (p *Bin) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Bin) String() string { return fmt.Sprintf("0b%s", strconv.FormatInt(int64(p.int64),2)) }
func (p *Bin) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),2), nil }

type Oct struct { integer }
func (p *Oct) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Oct) String() string { return fmt.Sprintf("0%s", strconv.FormatInt(int64(p.int64),8)) }
func (p *Oct) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),8), nil }

type Int struct { integer }
func (p *Int) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Int) String() string { return strconv.FormatInt(int64(p.int64),10) }
func (p *Int) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),10), nil }

type Hex struct { integer }
func (p *Hex) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Hex) String() string { return fmt.Sprintf("0x%s", strconv.FormatInt(int64(p.int64),16)) }
func (p *Hex) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),16), nil }

const FloatEpsilon = 1e-15 /* 1e-16 */
type Float struct {
        trivial
        float64
} // IEEE-754 64-bit binary floating-point
func (p *Float) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Float) True() (bool, error) { return math.Abs(p.float64)-0 > FloatEpsilon, nil }
func (p *Float) String() string { return strconv.FormatFloat(float64(p.float64),'g', -1, 64) }
func (p *Float) Strval() (string, error) { return strconv.FormatFloat(float64(p.float64),'g', -1, 64), nil }
func (p *Float) Integer() (int64, error) { return int64(p.float64), nil }
func (p *Float) Float() (float64, error) { return p.float64, nil }
func (p *Float) cmp(v Value) (res cmpres) {
        if _, ok := v.(*Float); ok {
                f, e := v.Float()
                assert(e == nil, "%T: %v", v, e)
                if p.float64 == f {
                        res = cmpEqual
                } else if p.float64 < f {
                        res = cmpSmaller
                } else if p.float64 > f {
                        res = cmpGreater
                }
        }
        return
}

type DateTime struct {
        trivial
        t time.Time
}
func (p *DateTime) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *DateTime) True() (bool, error) { return !p.t.IsZero(), nil }
func (p *DateTime) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{DateTime '%s' !(%+v)}", s, e)
        }
}
func (p *DateTime) Strval() (string, error) { return time.Time(p.t).Format("2006-01-02T15:04:05.999999999Z07:00"), nil } // time.RFC3339Nano
func (p *DateTime) Integer() (int64, error) { return p.t.Unix(), nil }
func (p *DateTime) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }
func (p *DateTime) cmp(v Value) (res cmpres) {
        var vt time.Time
        switch a := v.(type) {
        case *DateTime:
                vt = a.t
        case *Date:
                vt = a.t
        case *Time:
                vt = a.t
        default:
                return
        }
        if p.t.Equal(vt) {
                res = cmpEqual
        } else if p.t.Before(vt) {
                res = cmpSmaller
        } else /*if p.t.After(vt)*/ {
                res = cmpGreater
        }
        return
}

func ParseDateTime(pos Position, s string) *DateTime {
        // time.RFC3339Nano
        if t, e := time.Parse("2006-01-02T15:04:05.999999999Z07:00", s); e == nil {
                return &DateTime{trivial{pos},t}
        } else {
                panic(e)
        }
}

type Date struct { DateTime }
func (p *Date) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{Date '%s' !(%+v)}", s, e)
        }
}
func (p *Date) Strval() (string, error) { return time.Time(p.t).Format("2006-01-02"), nil }
func (p *Date) Integer() (int64, error) { return p.t.Unix(), nil }
func (p *Date) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

type Time struct { DateTime }
func (p *Time) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{Time '%s' !(%+v)}", s, e)
        }
}
func (p *Time) Strval() (string, error) { return time.Time(p.t).Format("15:04:05.999999999Z07:00"), nil }
func (p *Time) Integer() (int64, error) { return p.t.Unix(), nil }
func (p *Time) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

// ie. https://en.wikipedia.org/wiki/URL
// ▶▶─<scheme>─(:)┬──────────────────────────────────────┬<path>┬───────────┬┬──────────────┬─▶◀
//                └(//)┬──────────────┬<host>┬──────────┬┘      └(?)─<query>┘└(#)─<fragment>┘
//                     └<userinfo>─(@)┘      └(:)─<port>┘
type URL struct {
        trivial
        Scheme Value
        Username Value
        Password Value
        Host Value
        Port Value
        Path Value
        Query Value
        Fragment Value
}
func (p *URL) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *URL) True() (bool, error) { return p.String() != "", nil }
func (p *URL) elemstr(o Object, k elemkind) (s string) {
        if s = elementString(o, p.Scheme, k); s == "" { return }
        if s += ":"; p.Host == nil {
                // ...
        } else if _, ok := p.Host.(*None); ok {
                var host string
                if host = elementString(o, p.Host, k); host == "" { return }
                s += "//"
                if p.Username == nil {
                        // ...
                } else if isNone(p.Username) {
                        var user string
                        if user = elementString(o, p.Username, k); user != "" {
                                s += user + "@"
                        }
                }
                s += host
                if p.Port == nil {
                        // ...
                } else if _, ok := p.Port.(*None); ok {
                        var port string
                        if port = elementString(o, p.Port, k); port != "" {
                                s += ":" + port
                        }
                }
        }
        if p.Path == nil {
                // ...
        } else if _, ok := p.Path.(*None); ok {
                var path string
                if path = elementString(o, p.Path, k); path != "" {
                        //if !strings.HasPrefix(path, PathSep) { s += PathSep }
                        s += path
                }
        }
        if p.Query == nil {
                // ...
        } else if _, ok := p.Query.(*None); ok {
                var query string
                if query = elementString(o, p.Query, k); query != "" {
                        s += "?" + query
                }
        }
        if p.Fragment == nil {
                // ...
        } else if _, ok := p.Fragment.(*None); ok {
                var fragment string
                if fragment = elementString(o, p.Fragment, k); fragment != "" {
                        s += "#" + fragment
                }
        }
        return
}
func (p *URL) String() string { return p.elemstr(nil, 0) }
func (p *URL) Strval() (s string, err error) {
        if s, err = p.Scheme.Strval(); err != nil { return }
        if s += ":"; p.Host != nil && !isNone(p.Host) {
                var host string
                if host, err = p.Host.Strval(); err != nil { return }
                s += "//"
                if p.Username != nil && !isNone(p.Username) {
                        var user string
                        if user, err = p.Username.Strval(); err != nil { return }
                        s += user
                        if p.Password != nil {
                                var pass string
                                s += ":"
                                if pass, err = p.Password.Strval(); err != nil { return }
                                s += pass
                        }
                        s += "@"
                }
                s += host
                if p.Port != nil && !isNone(p.Port) {
                        var port string
                        if port, err = p.Port.Strval(); err != nil { return }
                        s += ":" + port
                }
        }
        if p.Path != nil && !isNone(p.Path) {
                var path string
                if path, err = p.Path.Strval(); err != nil { return }
                //if !strings.HasPrefix(path, PathSep) { s += PathSep }
                s += path
        }
        if p.Query != nil && !isNone(p.Query) {
                var query string
                if query, err = p.Query.Strval(); err != nil { return }
                s += "?" + query
        }
        if p.Fragment != nil && !isNone(p.Fragment) {
                var fragment string
                if fragment, err = p.Fragment.Strval(); err != nil { return }
                s += "#" + fragment
        }
        return
}
func (p *URL) Integer() (i int64, err error) {
        var s string
        if s, err = p.Strval(); err == nil {
                i = int64(len(s))
        }
        return
}
func (p *URL) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }
func (p *URL) cmp(v Value) (res cmpres) {
        if a, ok := v.(*URL); ok {
                if p.Scheme == nil || a.Scheme == nil { return }
                if p.Scheme.cmp(a.Scheme) != cmpEqual { return }
                if p.Username != nil {
                        if a.Username == nil { return }
                        if p.Username.cmp(a.Username) != cmpEqual { return }
                }
                if p.Password != nil {
                        if a.Password == nil { return }
                        if p.Password.cmp(a.Password) != cmpEqual { return }
                }
                if p.Host != nil {
                        if a.Host == nil { return }
                        if p.Host.cmp(a.Host) != cmpEqual { return }
                }
                if p.Port != nil {
                        if a.Port == nil { return }
                        if p.Port.cmp(a.Port) != cmpEqual { return }
                }
                if p.Path != nil {
                        if a.Path == nil { return }
                        if p.Path.cmp(a.Path) != cmpEqual { return }
                }
                if p.Query != nil {
                        if a.Query == nil { return }
                        if p.Query.cmp(a.Query) != cmpEqual { return }
                }
                if p.Fragment != nil {
                        if a.Fragment == nil { return }
                        if p.Fragment.cmp(a.Fragment) != cmpEqual { return }
                }
                res = cmpEqual
        }
        return
}
func (p *URL) Validate() (res *url.URL) {
        panic(fmt.Sprintf("validate %s", p))
        return
}

type Raw struct {
        trivial
        string
}
func (p *Raw) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Raw) True() (bool, error) { return p.string != "", nil }
func (p *Raw) String() string { return p.string }
func (p *Raw) Strval() (string, error) { return p.string, nil }
func (p *Raw) Integer() (int64, error) { return strconv.ParseInt(p.string, 10, 64) }
func (p *Raw) Float() (float64, error) { return strconv.ParseFloat(p.string, 64) }
func (p *Raw) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Raw); ok && p.string == a.string {
                res = cmpEqual
        }
        return
}

type String struct {
        trivial
        string
}
func (p *String) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *String) True() (bool, error) { return p.string != "", nil }
func (p *String) elemstr(o Object, k elemkind) (s string) {
        if k&elemNoQuote == 0 { s = `'`+p.string+`'` } else { s = p.string }
        return
}
func (p *String) String() string { return p.elemstr(nil, 0) }
func (p *String) Strval() (string, error) { return strings.Replace(p.string, "\\\"", "\"", -1), nil }
func (p *String) Integer() (int64, error) { return strconv.ParseInt(p.string, 10, 64) }
func (p *String) Float() (float64, error) { return strconv.ParseFloat(p.string, 64) }
func (p *String) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        if false { err = t.target(p.position, p.string) } else {
                err = errorf(p.position, "cant traverse string yet")
        }
        return
}
func (p *String) cmp(v Value) (res cmpres) {
        if a, ok := v.(*String); ok {
                if p.string == a.string {
                        res = cmpEqual
                } else if p.string < a.string {
                        res = cmpSmaller
                } else /*if p.string > a.string*/ {
                        res = cmpGreater
                }
        }
        return
}

func isTrueString(s string) (t bool) {
        switch strings.ToLower(s) {
        case "false", "no" , "off", "force_off", "0", "": t = false
        case "true" , "yes", "on" , "force_on" , "1": t = true
        default: t = true
        }
        return
}

type Bareword struct {
        trivial
        string
}
func (p *Bareword) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Bareword) True() (bool, error) { return isTrueString(p.string), nil }
func (p *Bareword) String() string { return p.string }
func (p *Bareword) Strval() (string, error) { return p.string, nil }
func (p *Bareword) Integer() (int64, error) { return strconv.ParseInt(p.string, 10, 64) }
func (p *Bareword) Float() (float64, error) { return strconv.ParseFloat(p.string, 64) }
func (p *Bareword) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        err = t.target(p.position, p.string)
        return
}
func (p *Bareword) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Bareword); ok {
                if p.string == a.string {
                        res = cmpEqual
                } else if p.string > a.string {
                        res = cmpSmaller
                } else if p.string < a.string {
                        res = cmpGreater
                }
        }
        return
}

type Qualiword struct {
        trivial
        words []string
}
func (p *Qualiword) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Qualiword) True() (bool, error) { return len(p.words)!=0, nil }
func (p *Qualiword) String() string { return strings.Join(p.words,".") }
func (p *Qualiword) Strval() (string, error) { return p.String(), nil }
func (p *Qualiword) Integer() (int64, error) { return int64(len(p.words)), nil }
func (p *Qualiword) Float() (float64, error) { return float64(len(p.words)), nil }
func (p *Qualiword) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        err = t.target(p.position, p.String())
        return
}
func (p *Qualiword) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Qualiword); ok {
                var n int
                var al, pl = len(a.words), len(p.words)
                for i, w := range p.words {
                        if al <= i {
                                break
                        } else if w == a.words[n] {
                                if n += 1; n == al && al == pl {
                                        res = cmpEqual
                                } else {
                                        continue
                                }
                        } else if w > a.words[n] {
                                res = cmpSmaller // cmpGreater??
                        } else {
                                res = cmpGreater // cmpSmaller??
                        }
                        break
                }
        }
        return
}

type elements struct { Elems []Value }
func (p *elements) Len() int                    { return len(p.Elems) }
func (p *elements) Append(v... Value)           { p.Elems = append(p.Elems, v...) }
func (p *elements) Get(n int) (v Value)         { if n>=0 && n<len(p.Elems) { v = p.Elems[n] }; return }
func (p *elements) Slice(n int) (a []Value) {
        if n>=0 && n<len(p.Elems) {
                a = p.Elems[n:]
        }
        return 
}
func (p *elements) Take(n int) (v Value) {
        if x := len(p.Elems); n>=0 && n<x {
                v = p.Elems[n]
                p.Elems = append(p.Elems[0:n], p.Elems[n+1:]...)
        }
        return 
}
func (p *elements) ToBarecomp() *Barecomp { return &Barecomp{trivial{},*p} }
func (p *elements) ToCompound() *Compound { return &Compound{trivial{},*p} }
func (p *elements) ToList() *List { return &List{*p} }
func (p *elements) True() (t bool, err error) { // (or elems...)
        for _, elem := range p.Elems {
                if elem == nil { continue }
                if t, err = elem.True(); t || err != nil {
                        break
                }
        }
        return
}
func (p *elements) refs(v Value) bool {
        for _, elem := range p.Elems {
                if elem != nil && (elem == v || elem.refs(v)) {
                        return true
                }
        }
        return false 
}
func (p *elements) closured() bool {
        for _, elem := range p.Elems {
                if elem.closured() { return true }
        }
        return false 
}
func (p *elements) refdef(origin DefOrigin) bool {
        for _, elem := range p.Elems {
                if elem.refdef(origin) { return true }
        }
        return false 
}
func (p *elements) cmpElems(elems []Value) (res cmpres) {
        if len(p.Elems) == len(elems) {
                for i, elem := range p.Elems {
                        if elem == nil { continue } else
                        if other := elems[i]; other == nil { continue } else
                        if elem.cmp(other) != cmpEqual { return cmpUnknown }
                }
                res = cmpEqual
        }
        return
}

type Barecomp struct { trivial ; elements }
func (p *Barecomp) refs(v Value) bool { return p.elements.refs(v) }
func (p *Barecomp) refdef(origin DefOrigin) bool { return p.elements.refdef(origin) }
func (p *Barecomp) closured() bool { return p.elements.closured() }
func (p *Barecomp) Strval() (s string, e error) {
        for _, elem := range p.Elems {
                var v string
                if elem == nil { continue } else
                if v, e = elem.Strval(); e == nil { s += v } else { break }
        }
        return
}
func (p *Barecomp) elemstr(o Object, k elemkind) (s string) {
        for _, elem := range p.Elems {
                s += elementString(o, elem, k)
        }
        return
}
func (p *Barecomp) True() (bool, error) { return p.elements.True() }
func (p *Barecomp) String() (s string) { return p.elemstr(nil, 0) }
func (p *Barecomp) Integer() (res int64, err error) {
        if len(p.Elems) == 2 {
                if i, ok := p.Elems[0].(*Int); ok {
                        var n = i.int64
                        if w, ok := p.Elems[1].(*Bareword); ok {
                                ;    if (w.string == "st" && n%1 == 0) ||
                                        (w.string == "nd" && n%2 == 0) ||
                                        (w.string == "rd" && n%3 == 0) ||
                                        (w.string == "th") { res = n }
                        }
                }
        }
        return
}
func (p *Barecomp) expand(w expandwhat) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = expandall(w, p.Elems...); err == nil {
                if num > 0 {
                        res = &Barecomp{p.trivial,elements{elems}}
                } else {
                        res = p
                }
        }
        return
}
func (p *Barecomp) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        var target string
        if target, err = p.Strval(); err == nil {
                err = t.target(p.position, target)
        }
        return
}
func (p *Barecomp) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Barecomp); ok { res = p.cmpElems(a.Elems) }
        return
}

// Barefile works like an alias of a File, the Strval() is identical to File.
type Barefile struct {
        trivial
        Name Value
        File *File
}
func (p *Barefile) refs(v Value) bool { return p.Name.refs(v) }
func (p *Barefile) closured() bool { return p.Name.closured() }
func (p *Barefile) expand(w expandwhat) (res Value, err error) {
        var name Value
        if name, err = p.Name.expand(w); err == nil {
                if name != p.Name {
                        res = &Barefile{p.trivial,name,p.File}
                } else {
                        res = p
                }
        }
        return
}
func (p *Barefile) True() (t bool, err error) {
        if p.File != nil { t, err = p.File.True() }
        return
}
func (p *Barefile) elemstr(o Object, k elemkind) (s string) { return elementString(o, p.Name, k) }
func (p *Barefile) String() string { return p.elemstr(nil, 0) }
func (p *Barefile) Strval() (string, error) {
        if p.File != nil {
                return p.File.Strval()
        } else {
                return p.Name.Strval()
        }
}
func (p *Barefile) Integer() (res int64, err error) {
        if exists(p.File) {
                res = p.File.info.Size()
        }
        return
}
func (p *Barefile) Float() (float64, error) {
        i, e := p.Integer()
        return float64(i), e
}
func (p *Barefile) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        if p.File == nil { // it happens if p.Name refers argument
                var target string
                if target, err = p.Strval(); err != nil { return }
                
                var okay bool
                okay, err = t.forClosureProject(func(project *Project) (bool, error) {
                        p.File = project.matchFile(target)
                        return p.File != nil, nil
                })
                if !okay || p.File == nil {
                        err = errorf(p.position, "barefile '%s' not found", target)
                        return
                }
        }
        if p.File != nil { err = p.File.traverse(t) } else {
                err = errorf(p.position, "barefile '%s' is nil", p)
        }
        return
}
func (p *Barefile) stamp(t *traversal) (files []*File, err error) {
        if p.File != nil { files, err = p.File.stamp(t) }
        return
}
func (p *Barefile) exists() (res existence) {
        if p.File != nil {
                res = p.File.exists()
        } else {
                res = existenceNegated
        }
        return
}
func (p *Barefile) mod(t *traversal) (res time.Time, err error) {
        if p.File != nil { res, err = p.File.mod(t) }
        return
}
func (p *Barefile) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Barefile); ok { res = p.Name.cmp(a.Name) }
        return
}

type GlobMeta struct {
        trivial
        token.Token
}
func (p *GlobMeta) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *GlobMeta) String() string { return p.Token.String() }
func (p *GlobMeta) Strval() (string, error) { return p.Token.String(), nil }
func (p *GlobMeta) cmp(v Value) (res cmpres) {
        if a, ok := v.(*GlobMeta); ok && p.Token == a.Token {
                res = cmpEqual
        }
        return
}

// `[a-b]`, `[abc]`, ...
// `a-b`, `abc`, `a$(var)c`, `a$(spaces)c`...
type GlobRange struct { trivial ; Chars Value }
func (p *GlobRange) refs(v Value) bool { return p.Chars.refs(v) }
func (p *GlobRange) closured() bool { return p.Chars.closured() }
func (p *GlobRange) expand(w expandwhat) (Value, error) {
        if v, err := p.Chars.expand(w); err != nil {
                return nil, err
        } else if v != p.Chars {
                return &GlobRange{p.trivial,v}, nil
        } else {
                return p, nil
        }
}
func (p *GlobRange) elemstr(o Object, k elemkind) (s string) {
        return fmt.Sprintf("[%s]", elementString(o, p.Chars, k))
}
func (p *GlobRange) String() (s string) { return p.elemstr(nil, 0) }
func (p *GlobRange) Strval() (s string, err error) {
        var chars string
        if chars, err = p.Chars.Strval(); err == nil {
                s = fmt.Sprintf("[%s]", chars)
        }
        return
}
func (p *GlobRange) cmp(v Value) (res cmpres) {
        if a, ok := v.(*GlobRange); ok { res = p.Chars.cmp(a.Chars) }
        return
}

// Path is addressing a file (dynamically), the real located file varies
// base on 'elements' and the context.
type Path struct {
        trivial
        elements
}
func (p *Path) elemstr(o Object, k elemkind) (s string) {
        for i, elem := range p.Elems {
                var v = elementString(o, elem, k)
                if i > 0 {
                        s += PathSep + v
                } else if v != "" {
                        s += v
                } else if len(p.Elems) == 1 {
                        s += PathSep
                }
        }
        return
}
func (p *Path) String() (s string) { return p.elemstr(nil, 0) }
func (p *Path) Strval() (s string, e error) {
        for i, seg := range p.Elems {
                if seg == nil {
                        e = fmt.Errorf("`%s` nil path segment", p)
                        return
                }

                var v string
                if v, e = seg.Strval(); e != nil { return }
                if i > 0 {
                        s += PathSep + v
                } else if v != "" {
                        s += v
                } else if len(p.Elems) == 1 {
                        s += PathSep
                }
        }
        return
}
func (p *Path) True() (t bool, err error) {
        // FIXME: return p.exists() ??
        if false {
                t = p.exists() == existenceConfirmed
        } else {
                for _, elem := range p.Elems {
                        if t, err = elem.True(); err != nil || !t {
                                break
                        }
                }
        }
        return
}
func (p *Path) refs(v Value) (res bool) { return p.elements.refs(v) }
func (p *Path) closured() (res bool) { return p.elements.closured() }
func (p *Path) refdef(origin DefOrigin) bool { return p.refdef(origin) }
func (p *Path) expand(w expandwhat) (res Value, err error) {
        var (elems []Value; num int)
        if elems, num, err = expandall(w, p.Elems...); err != nil { return }
        if w&expandPath != 0 {
                var vals []Value
                for _, elem := range elems {
                        switch v := elem.(type) {
                        case *String:
                                segs := MakePathStr(elem.Position(),v.string).Elems
                                vals = append(vals, segs...)
                        default:
                                vals = append(vals, elem)
                        }
                }
                elems = vals
        }
        if num > 0 {
                res = &Path{p.trivial,elements{elems}}
        } else {
                res = p
        }
        return
}
func (p *Path) pathname(stems []string) (pathname string, err error) {// the addressed file target
        var rest []string // unmatched path segmants
        if len(stems) == 0 { pathname, err = p.Strval()  } else
        if pathname, rest, err = p.stencil(stems); err != nil {
                // ...
        } else if len(rest) > 0 {
                //err = errorf(p.position, "partial match: %v", rest)
        }
        return
}
func (p *Path) stamp(t *traversal) (files []*File, err error) {
        var pathname string
        if pathname, err = p.Strval(); err == nil {
                if pathname == "" {
                        err = errorf(p.position, "no pathname for `%s`", p)
                } else if file := stat(p.position,pathname,"","",nil); file != nil {
                        files, err = file.stamp(t)
                }
        }
        return
}
func (p *Path) exists() existence {
        if pathname, err := p.Strval(); err == nil {
                if file := stat(p.position, pathname,"","",nil); file != nil {
                        return existenceConfirmed
                }
        }
        return existenceNegated
}
func (p *Path) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }

        // Path pathname.
        var pathname string // the addressed file target
        if pathname, err = p.pathname(t.stems); err == nil && pathname == "" {
                err = errorf(p.position, "path matches no target: %v", p); return
        } else if err != nil { err = wrap(p.position, err); return }

        // Stat the file by pathname.
        var file = stat(p.position, pathname, "", ""/*, nil*/)
        if optionTraceTraversal { t.tracef("Path: file=%v (exists=%v) (pathname=%s)", file, file.exists(), pathname) }
        if file == nil { err = t.target(p.position,pathname) } else { err = file.traverse(t) }
        if err != nil { err = wrap(p.position, err) }
        if optionTraceTraversal { t.tracef("Path: file=%v (exists=%v) (pathname=%s)", file, file.exists(), pathname) }
        return
}
func (p *Path) mod(t *traversal) (res time.Time, err error) {
        var pathname string // the addressed file target
        if pathname, err = p.pathname(t.stems); err != nil {
                // oops
        } else if pathname == "" {
                err = errorf(p.position, "path matches no target: %v", p)
        } else if file := stat(p.position, pathname, "", "", nil); file != nil && file.info != nil {
                res = file.info.ModTime()
        }
        return
}
func (p *Path) isPattern() (result bool) {
        for _, seg := range p.Elems {
                _, result = seg.(Pattern)
                if result { return }
        }
        return
}
func (p *Path) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Path); ok { res = p.cmpElems(a.Elems) }
        return
}
func (p *Path) match(i interface{}) (result string, stems []string, err error) {
        if optionEnableBenchspots { defer bench(spot("Path.match")) }
        var retained []string
        result, retained, stems, err = p.partialMatch(i)
        if len(retained) > 0 {
                // clear results if not fully matched
                result, stems = "", nil
        }
        return
}
func (p *Path) partialMatch(i interface{}) (result string, retained, stems []string, err error) {
        switch t := i.(type) {
        case *filestub: i = filepath.Join(t.dir, t.sub, t.name)
        case *File:     i = filepath.Join(t.dir, t.sub, t.name)
        }
        return p.match1(i)
}
func (p *Path) match1(i interface{}) (result string, retained, stems []string, err error) {
        var (
                srcs []string
                segs []Value
                idx = 0
        )
        if segs, err = ExpandAll(p.Elems...); err != nil { return } else {
                switch t := i.(type) {
                case []string: srcs = t
                case   string: srcs = strings.Split(t, PathSep)
                case Value:
                        var s string
                        if s, err = t.Strval(); err == nil {
                                srcs = strings.Split(s, PathSep)
                        } else { return }
                default: unreachable("path.match1: %T %v", i, i)
                }
        }
        ForPathSegs: for n, seg := range segs {
                if len(srcs) <= idx { break ForPathSegs }

                var ( s string ; r []string )
                switch t := seg.(type) {
                case *Path:// Note that Path is also a Pattern!
                        var ss []string
                        if s, r, ss, err = t.match1(srcs[idx:]); err != nil {
                                break ForPathSegs
                        } else if s == "" {
                                return
                        } else if len(r) > 0 {
                                // not fully matched
                        }
                        stems = append(stems, ss...)
                        idx += len(strings.Split(s, PathSep))
                case Pattern:// Note that Path is also a Pattern!
                        var ss []string

                        // Special case for "/%%/" to match many segs at once.
                        if ok, prefix, suffix := percperc(t); ok && isNone(prefix) && isNone(suffix) {
                                if n+1 < len(segs) && idx+1 < len(srcs) {
                                        // Find 'next' matched seg, ie. %%/next/xxx
                                        var next = segs[n+1] // e.g. '.smart' like in '%%/.smart/modules'
                                        switch t := next.(type) {
                                        case Pattern:
                                                for x, src := range srcs[idx+1:] {
                                                        var res string
                                                        res, ss, err = t.match(src)
                                                        if err != nil { return }
                                                        if res != "" && len(ss) > 0 {
                                                                end := idx + 1 + x
                                                                stem := strings.Join(srcs[idx:end], PathSep)
                                                                stems = append(stems, stem)
                                                                idx = end
                                                                continue ForPathSegs //break //
                                                        }
                                                }
                                        default:
                                                for x, src := range srcs[idx+1:] {
                                                        var res string
                                                        if res, err = t.Strval(); err != nil {
                                                                return
                                                        }
                                                        if res == src {
                                                                end := idx + 1 + x
                                                                stem := strings.Join(srcs[idx:end], PathSep)
                                                                stems = append(stems, stem)
                                                                idx = end
                                                                continue ForPathSegs //break //
                                                        }
                                                }
                                        }
                                } else if n+1 == len(segs) && idx < len(srcs) {
                                        // Matching the last seg, ie. /foo/bar/%% <-> /foo/bar/x/y/z,
                                        // where 'segs[n] == %%' and 'srcs[idx] == x'
                                        stem := strings.Join(srcs[idx:], PathSep)
                                        stems = append(stems, stem)
                                        idx = len(srcs)
                                        break ForPathSegs
                                } else if len(srcs) < len(segs) {
                                        // No matches, e.g.
                                        //   '%%/xxx.txt' <-> 'xxx.txt'
                                        break ForPathSegs
                                } else if false && len(srcs) > 1 { // FIXME: this matches '%%/xxx.txt' to 'xxx.txt'
                                        stem := strings.Join(srcs[idx:], PathSep) // WRONG!
                                        stems = append(stems, stem)
                                        idx = len(srcs)
                                        break ForPathSegs
                                } else {
                                        // FIXME: this matches '%%/xxx.txt' to 'xxx.txt'
                                        fmt.Fprintf(stderr, "Path.match1.FIXME: %v !> %v (n=%v, idx=%v)\n", segs, srcs, n, idx)
                                        break ForPathSegs
                                }
                        }
                        
                        if s, ss, err = t.match(srcs[idx]); err != nil {
                                break ForPathSegs
                        } else if s == "" || ss == nil {
                                return
                        }
                        stems = append(stems, ss...)
                        idx += 1
                default:
                        if s, err = seg.Strval(); err != nil {
                                break ForPathSegs
                        }
                        for _, s := range strings.Split(s, PathSep) {
                                if srcs[idx] != s { return }
                                idx += 1
                        }
                }
        }

        if 0 < idx && idx <= len(srcs) {
                // Don't use filepath.Join as in the case that ""
                // have to be root "/".
                result = strings.Join(srcs[:idx], PathSep)
                retained = srcs[idx:]
        }
        return
}
func (p *Path) stencil(stems []string) (result string, rest []string, err error) {
        var (
                strs []string
                segs []Value
        )
        if segs, err = ExpandAll(p.Elems...); err != nil {
                return
        }

ForPathSegs:
        for _, seg := range segs {
                var s string
                switch t := seg.(type) {
                case Pattern: // including *Path
                        if len(stems) > 0 {
                                s, stems, err = t.stencil(stems)
                                if err != nil { return }
                                strs = append(strs, s)
                                continue ForPathSegs
                        }
                }
                if s, err = seg.Strval(); err != nil {
                        break ForPathSegs
                } else {
                        strs = append(strs, s)
                }
        }
        if err == nil {
                result = strings.Join(strs, PathSep)
                rest = stems // the rest stems
        }
        return
}

type PathSeg struct {
        trivial
        rune
}
func (p *PathSeg) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *PathSeg) String() (s string) { 
        var e error
        if s, e = p.Strval(); e != nil { s = "?" }
        return
}
func (p *PathSeg) Strval() (s string, e error) {
        switch p.rune {
        case '/': s = "" // the first '/', aka. root -- PathSep is added when joining
        case '~': s = "~"
        case '.': s = "."
        case '^': s = ".."
        case 0:   s = "" // empty segment after the last '/', e.g. /foo/bar/ 
        default:  e = fmt.Errorf("unknown pathseg (%s)", p.rune)
        }
        return
}
func (p *PathSeg) cmp(v Value) (res cmpres) {
        if a, ok := v.(*PathSeg); ok && p.rune == a.rune { res = cmpEqual }
        return
}

type filestub struct {
        dir string       // full directory where the file was or should be found
        sub string       // matched sub path (see Project.search), may be Dir (absoletep path)
        name string      // constant represented name (e.g. relative filename)
        match *FileMap   // matched pattern (see 'files' directive)
        other *filestub  // pointed to another stub (in a different project) of the same file
}

type filebase struct {
        stub filestub    // cycled-list of file stubs of different projects
        info os.FileInfo // file info if exists
        updated bool // true if this file has been updated by a program
}

var filecache = make(map[string]*filebase) // File.fullname() -> File
var statmutex = new(sync.Mutex)

func (p *filestub) subname() (s string) {
        if isAbsOrRel(p.sub) {
                s = p.name
        } else {
                s = filepath.Join(p.sub, p.name)
        }
        return
}
func (p *filebase) exists() (res existence) {
        if p.info != nil {
                res = existenceConfirmed
        } else {
                res = existenceNegated
        }
        return
}

func stat(pos Position, name, sub, dir string, infos ...os.FileInfo) (file *File) {
        var ( base *filebase ; stub *filestub ; fullname string )

        statmutex.Lock()
        defer statmutex.Unlock()

        // Trims / suffix
        if dir != "" { dir = filepath.Clean(dir) }
        if sub != "" { sub = filepath.Clean(sub) }
        if name!= "" { name= filepath.Clean(name) }

        if filepath.IsAbs(name) {
                if fullname = name; dir == "" {
                        //dir, sub = filepath.Dir(fullname), ""
                        //name = filepath.Base(fullname)
                } else if strings.HasPrefix(fullname, dir+PathSep) {
                        tail := fullname[len(dir)+1:]
                        //sub  = filepath.Dir(tail)
                        //name = filepath.Base(tail)
                        if sub == "" { name = tail } else
                        if strings.HasPrefix(fullname, sub+PathSep) {
                                name = tail[len(sub)+1:]
                        }
                } else if dir != "" {
                        if true { dir = "" } else if false {
                                if optionPrintStack || true { debug.PrintStack() }
                                unreachable(errorf(pos, "dir name conflicts: %s <-> %s (sub=%v)", dir, name, sub))
                        } else {
                                return
                        }
                }
        } else if filepath.IsAbs(sub) {
                fullname = filepath.Join(sub, name)
                if dir == "" {
                        dir = sub // trims / suffix
                        sub = "" // .
                } else if sub == dir {
                        sub = "" // .
                } else if strings.HasPrefix(sub, dir) {
                        sub = strings.TrimPrefix(sub, dir)
                        sub = strings.TrimPrefix(sub, PathSep)
                        sub = filepath.Clean(sub)
                } else if false {
                        dir = sub
                        sub = ""
                } else {
                        unreachable("conflicted sub/dir: ", sub, " ", dir) //return
                }
        } else if filepath.IsAbs(dir) {
                fullname = filepath.Join(dir, sub, name)
        } else {
                fullname = filepath.Join(context.workdir, dir, sub, name)
        }

        fullname = filepath.Clean(fullname)

        if enable_assertions {
                assert(filepath.IsAbs(fullname), "`%s` is not abs {%s %s %s}", fullname, name, sub, dir)
                if filepath.IsAbs(name) {
                        assert(dir == "", "`%s` invalid file{%s %s %s}", fullname, dir, sub, name)
                        assert(sub == "", "`%s` invalid file{%s %s %s}", fullname, dir, sub, name)
                }
                assert(!filepath.IsAbs(sub), "`%s` sub is abs", sub)
                
                if filepath.IsAbs(name) {
                        s := name
                        assert(fullname == s, "`%s` conflicted fullname (%s)", fullname, s)
                } else if filepath.IsAbs(sub) {
                        s := filepath.Join(sub, name)
                        assert(fullname == s, "`%s` conflicted fullname (%s)", fullname, s)
                } else if filepath.IsAbs(dir) {
                        s := filepath.Join(dir, sub, name)
                        assert(fullname == s, "`%s` conflicted fullname (%s)", fullname, s)
                } else {
                        s := filepath.Join(context.workdir, dir, sub, name)
                        assert(fullname == s, "`%s` conflicted fullname (%s)", fullname, s)
                }
        }

        var addNotExisted bool
        var info os.FileInfo
        if len(infos) == 1 {
                if info = infos[0]; info == nil {
                        addNotExisted = true
                }
                if enable_assertions && info != nil {
                        assert(info.Name() == filepath.Base(fullname), "`%s` file name conflicted", info.Name())
                }
        } else if len(infos) > 1 {
                unreachable("too many file infos")
        }

        var okay bool
        if base, okay = filecache[fullname]; okay {
                if base.info == nil {
                        if info == nil { info, _ = os.Stat(fullname) }
                        if info == nil && !addNotExisted {
                                return nil // file not exists
                        }
                        base.info = info
                }

                var head = &base.stub
                /*
                if enable_assertions {
                        for stub = head; stub != nil ; stub = stub.other {
                                s := filepath.Join(stub.dir, stub.sub, stub.name)
                                assert(fullname == s, "(%s %s %s) fullname conflicted", stub.dir, stub.sub, stub.name)
                                if stub.other == head { break }
                        }
                }
                */
                for stub = head; stub != nil; stub = stub.other {
                        if stub.dir == dir && stub.sub == sub && stub.name == name {
                                goto GotFile
                        }
                        if stub.other == head { break }
                }

                stub = &filestub{ dir, sub, name, nil, head.other }
                head.other = stub
        } else {
                if info == nil {
                        info, _ = os.Stat(fullname)
                        if info == nil && !addNotExisted {
                                return nil // file not exists
                        }
                }

                base = &filebase{ filestub{ dir, sub, name, nil, nil }, info, false }
                base.stub.other = &base.stub
                stub = &base.stub
                filecache[fullname] = base
        }
GotFile:
        file = &File{trivial{pos},base,stub} // FIXME: needs position information
        if enable_assertions {
                if !addNotExisted {
                        assert(exists(file), "`%s` file not existed", fullname)
                }
                assert(file.name == name, "(%s %s %s).name != %s", file.name, file.sub, file.dir, name)
                assert(file.sub == sub, "(%s %s %s).sub != %s", file.name, file.sub, file.dir, sub)
                if file.dir != dir {
                        var head = &base.stub
                        for stub := head; stub != nil; stub = stub.other {
                                fmt.Fprintf(stderr, "stat: %s %s %s\n", stub.dir, stub.sub, stub.name)
                                if stub.other == head { break }
                        }
                }
                assert(file.dir == dir, "(%s %s %s).dir != %s", file.name, file.sub, file.dir, dir)
                //assert(file.dir != "", "(%s %s %s) empty dir", file.name, file.sub, file.dir)
                if exists(file) {
                        assert(file.info != nil, "(%s %s %s) info is nil", file.name, file.sub, file.dir)
                        assert(file.info.Name() == filepath.Base(file.name), "(%s %s %s) name conflicted", file.name, file.sub, file.dir)
                        s := filepath.Join(file.dir, file.sub, file.name)
                        assert(file.fullname() == s, "(%s %s %s) fullname conflicted (%s)", file.dir, file.sub, file.name, s)
                }
        }
        return
}

type File struct {
        trivial
        *filebase
        *filestub
}
func (p *File) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *File) True() (t bool, err error) {
        if p.name != "" {
                t = true // p.exists() == existenceConfirmed
        }
        return
}
func (p *File) String() string { return p.name }
func (p *File) Strval() (s string, err error) { s = p.fullname(); return }
func (p *File) BaseName() (s string) {
        if p.info != nil { s = p.info.Name() } else {
                s = filepath.Base(p.name)
        }
        return
}
func (p *File) fullname() (s string) {
        return filepath.Join(p.dir, p.sub, p.name)
}
func (p *File) searchInMatchedPaths(proj *Project) (res bool) {
        if p.match != nil {
                var pre string
                // FIXME: File should keep both 'match' and 'pre',
                // or just remove searchInMatchedPaths
                f := p.match.stat(proj.absPath, pre, p.name)
                if f.info != nil { p.info, res = f.info, true }
        }
        return
}
func (p *File) stamp(t *traversal) (files []*File, err error) {
        var fullname string
        if fullname = p.fullname(); fullname == "" {
                err = errorf(p.position, "no fullname for `%s`", p)
                return
        }

        var ot time.Time
        if p.info != nil { ot = p.info.ModTime() }
        if p.info, err = os.Stat(fullname); err != nil { return }
        if p.info != nil {
                var nt = p.info.ModTime()
                context.globe.stamp(fullname, nt)
                p.updated = nt.After(ot)
                files = append(files, p)

                var target = t.def.target.value
                var cmp = target.cmp(p)
                if cmp == cmpEqual && t.caller != nil {
                        // Add to caller context
                        t.caller.appendUpdated(newUpdatedTarget(p))
                        target = t.caller.def.target.value
                } else {
                        t.appendUpdated(newUpdatedTarget(p))
                }
                if optionTraceTraversal {
                        t.tracef("stamp: %v (%v, %v, %v)", p, nt.Sub(ot),
                                target, cmp)
                }
        }
        return
}
func (p *File) exists() existence {
        if p != nil && p.filebase != nil {
                return p.filebase.exists()
        } else {
                return existenceNegated
        }
}
func (p *File) isSysFile() (res bool) {
        if p.match != nil && len(p.match.Paths) == 1 {
                // system files defined by:
                //     files (
                //         (foo.xxx) ⇒ -
                //     )
                if f, ok := p.match.Paths[0].(*Flag); ok {
                        res = isNone(f.name) || isNil(f.name)
                        //fmt.Fprintf(stderr, "sys: %v %v %v\n", p, res, p.match)
                }
        }
        return
}
func (p *File) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        if optionTraceExec { defer un(trace(t_exec, fmt.Sprintf("File %v", p))) }
        if p.isSysFile() { if optionTraceTraversal { t.tracef("isSysFile: %v", p, p.isSysFile()) }
                return
        }

        // Add new file target, no matter it's going to be updated or not.
        t.addNewTarget(p)

        // FIXES: checks none-File file target
        switch a := t.def.target.value.(type) {
        case *Barecomp: // convert barecomp path into a real Path
                var v = a.Elems[0]
                if p, ok := v.(*Path); ok {
                        a.Elems = append(p.Elems[len(p.Elems)-1:], a.Elems[1:]...)
                        p.Elems[len(p.Elems)-1] = a
                        t.def.target.value = p
                        if optionTraceTraversal { t.tracef("FIX: barecomp path: %v", p) }
                } else {
                        var s string
                        if s, err = a.Strval(); err != nil { return }
                        if file := t.project.matchFile(s); file != nil {
                                t.def.target.value = file
                                if optionTraceTraversal { t.tracef("FIX: barecomp file: %v", p) }
                        }
                }
        }

        /*if v := t.def.target.value; strings.Contains(v.String(), "isl_srcdir.") || strings.Contains(p.name, "isl_srcdir.") {
                fmt.Fprintf(stderr, "%s: %v: %v: %v: %v (%v) (%v) (File.traverse 1)\n", t.project, p.position, t.entry, v, p, t.target0.value, t.targets.value)
        }*/
        if err = t.file(p); err != nil { return }
        /*if v := t.def.target.value; strings.Contains(v.String(), "isl_srcdir.") || strings.Contains(p.name, "isl_srcdir.") {
                fmt.Fprintf(stderr, "%s: %v: %v: %v: %v (%v) (%v) (File.traverse 2)\n", t.project, p.position, t.entry, v, p, t.target0.value, t.targets.value)
        }*/

        if optionTraceTraversal {
                var a = t.def.target.value
                var t1, _ = a.mod(t)
                var t2, _ = p.mod(t)
                t.tracef("%s: %v (%v)", typeof(a), a, t1)
                t.tracef("%s: %v (%v)", typeof(p), p, t2)
        }

        if p.info == nil { return }

        // Note that the file maybe not traversed yet at this point. But we
        // still have to check mod-time.
        var a time.Time
        if a, err = t.def.target.value.mod(t); err != nil { return }
        if!a.IsZero() && p.info.ModTime().After(a) { // a.IsZero() indicates the target not exists
                if optionTraceTraversal { t.tracef("updated: %v", p) }
                t.appendUpdated(newUpdatedTarget(p))
        }
        return
}

// check pattern depends to find out if all depends are updatable
// or updated/exists.
func checkPatternDepends(t *traversal, project *Project, se *StemmedEntry, prog *Program) (res bool, err error) {
        if optionEnableBenchspots { defer bench(spot("checkPatternDepends")) }
        if len(prog.depends) == 0 {
                // Pattern is always good as no depends to check.
                return true, nil
        }

        // Set arguments in case that depends may refer to a parameter.
        if prog.params == nil || t.arguments == nil {
                // no need to set arguments
        } else {
                var params []*Def
                if params, err = prog.args(t.arguments); err != nil { return } else
                if len(params) > 0 {
                        defer func(none *None) {
                                for _, param := range params {
                                        param.set(DefDefault, none)
                                }
                        } (&None{trivial{prog.position}})
                }
        }

        var checkedPatterns = 0
        for _, dep := range prog.depends {
                switch d := dep.(type) {
                case Pattern:
                        res, err = checkPatternDepend(t, project, se, prog, d)
                        checkedPatterns += 1
                        if err != nil { return }
                        if !res { break }
                case *Argumented:
                        var ok, res1 bool
                        ok, res1, err = d.checkPatternDepends(t, project, se, prog)
                        if err != nil { return }
                        if ok && !res1 { break }
                        res = res1
                default:
                        /*
                        var name, str string
                        var rest []string // rest stems
                        name, rest, err = se.stencil(se.Stems)
                        if err != nil { return }
                        if len(rest) > 0 { panic("FIXME: unhandled stems") }
                        if str, err = dep.Strval(); err != nil { return }
                        if res = str == name; !res { break }
                        */
                }
        }
        if !res && checkedPatterns == 0 {
                // If there's no pattern depends, we're good to use the
                // pattern to update target.
                res = true
        }
        return
}

func checkPatternDepend(t *traversal, project *Project, se *StemmedEntry, prog *Program, pat Pattern) (res bool, err error) {
        if optionEnableBenchspots { defer bench(spot("checkPatternDepend")) }

        var name string
        var rest []string // rest stems
        if name, rest, err = pat.stencil(se.Stems); err != nil { return }
        if false && len(rest) > 0 { panic("FIXME: unhandled stems") }

        // Check concrete rules for the name, it's 'exists' if there's a
        // non-empty rule for the name.
        var entry *RuleEntry
        if entry, err = project.resolveEntry(name); err != nil {
                return
        } else if entry != nil {
                var recipes int
                for _, prog := range entry.programs {
                        recipes += len(prog.recipes)
                }
                if recipes > 0 { return true, nil }
        }

        // Check pattern rules for the name, it's 'exists' if there's a
        // non-empty pattern rule for the name.
        var ses []*StemmedEntry
        if ses, err = project.resolvePatterns(name); err != nil {
                return
        } else if len(ses) > 0 {
        ForPatterns:
                for _, se := range ses {
                        var recipes int
                        for _, prog := range se.programs {
                                var ok bool
                                ok, err = checkPatternDepends(t, project, se, prog)
                                if !ok { continue ForPatterns }
                        }
                        if recipes > 0 { return true, nil }
                }
        }

        // Matches a FileMap (IsKnown(), may exists or not)
        if exists(project.matchFile(name)) { return true, nil }
        if filepath.IsAbs(name) {
                if exists(stat(project.position, name, "", "")) { return true, nil }
        }

        // TODO: check filepath.Join(project.absPath, name)
        return
}
func (p *File) mod(t *traversal) (res time.Time, err error) {
        if p.info == nil { p.info, /*err*/_ = os.Stat(p.fullname()) }
        if err != nil { err = wrap(p.position, err)
                if optionPrintStack || true {
                        fmt.Fprintf(stderr, "%s: %v: %v (%v)\n", t.project, p.position, p, p.match)
                        debug.PrintStack()
                }
        } else if p.info != nil { res = p.info.ModTime() }
        return
}
func (p *File) cmp(v Value) (res cmpres) {
        if v == nil {
                // ...
        } else if a, ok := v.(*File); ok {
                if a == nil {
                        //assert(a != nil, "nil file")
                } else if p.filebase == a.filebase {
                        res = cmpEqual
                } else if p.fullname() == a.fullname() {
                        s := fmt.Sprintf("\na: %s %s %s (%s)", p.dir, p.sub, p.name, p.fullname())
                        s += fmt.Sprintf("\nb: %s %s %s (%s)", a.dir, a.sub, a.name, a.fullname())
                        unreachable("same files differed: ", p.name, " != ", a.name, s)
                } else if false /*p.dir != a.dir && p.sub == a.sub && p.name == a.name*/ {
                        s := fmt.Sprintf("\n      a: %s: %s %s", p.name, p.dir, p.sub)
                        s += fmt.Sprintf("\n      b: %s: %s %s", a.name, a.dir, a.sub)
                        fmt.Fprintf(stderr, "%s: warning: files may differ: %s != %s :%s\n", p.position, p.name, a.name, s)
                }
        }
        return
}

func (p *File) change(dir, sub, name string) (okay bool) {
        var fullname = filepath.Join(dir, sub, name)
        if p.fullname() == fullname {
                var head = &p.filebase.stub
                for stub := p.filestub; stub != nil; stub = stub.other {
                        if stub.dir == dir && stub.sub == sub && stub.name == name {
                                p.filestub, okay = stub, true
                                return
                        }
                        if stub.other == head { break }
                }
                
                p.filestub = &filestub{ dir, sub, name, nil, head.other }
                head.other, okay = p.filestub, true
                
                if enable_assertions {
                        assert(p.fullname() == fullname, "Changed invalid File")
                }
        }
        return
}

type FileContent struct {
        file *File
        content []byte
}

type Flag struct { trivial ; name Value }
func (p *Flag) refs(v Value) bool { return p.name.refs(v) }
func (p *Flag) closured() bool { return p.name.closured() }
func (p *Flag) expand(w expandwhat) (res Value, err error) {
        var name Value
        if name, err = p.name.expand(w); err == nil {
                if name != p.name {
                        res = &Flag{p.trivial,name}
                } else {
                        res = p
                }
        }
        return
}
func (p *Flag) True() (t bool, err error) { return p.name.True() }
func (p *Flag) elemstr(o Object, k elemkind) (s string) { return "-" + elementString(o, p.name, k) }
func (p *Flag) String() (s string) { return p.elemstr(nil, 0) }
func (p *Flag) Strval() (s string, e error) {
        if p.name == nil {
                s = "-"
        } else if  _, ok := p.name.(*None); ok {
                s = "-"
        } else if s, e = p.name.Strval(); e == nil {
                s = "-" + s
        }
        return
}
func (p *Flag) opts(try bool, opts ...string) (runes []rune, names []string, err error) {
        switch t := p.name.(type) {
        case *Flag:
                runes, names, err = t.opts(try, opts...)
        case *String:
                for _, opt := range opts {
                        if t.string == opt { names = append(names, opt) }
                }
                if !try && len(names) == 0 {
                        err = errorf(p.Position(), "unknown flag (known: %s)", strings.Join(opts, ", "))
                }
        case *Bareword:
                for _, opt := range opts {
                        if i := strings.IndexRune(opt, ','); i == 0 {
                                if t.string == opt[1:] {
                                        names = append(names, opt)
                                }
                        } else if i > 0 {
                                if t.string == opt[i+1:] {
                                        runes = append(runes, rune(opt[0]))
                                        names = append(names, opt[i+1:])
                                } else if t.string ==  opt[0:i]/*strings.ContainsAny(t.string, opt[0:i])*/ {
                                        runes = append(runes, rune(opt[0]))
                                        names = append(names, opt[i+1:])
                                }
                        }
                }
                if !try && (len(runes) == 0 || len(names) == 0) {
                        err = errorf(p.Position(), "unknown flag (known: %s)", strings.Join(opts, ", "))
                }
        }
        if enable_assertions {
                assert(len(runes) == len(names), "unmatched opts lengths")
        }
        return
}
func (p *Flag) cmp(v Value) (res cmpres) {
        if v == nil {
                // ...
        } else if a, ok := v.(*Flag); ok {
                res = p.name.cmp(a.name)
        }
        return
}
func (p *Flag) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        var s string
        if s, err = p.Strval(); err == nil {
                err = t.target(p.position, s)
        }
        return
}

const escapedChars = "\"\r\n"
      
type Compound struct { trivial ; elements } // "compound string"
func (p *Compound) expand(w expandwhat) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = expandall(w, p.Elems...); err == nil {
                if num > 0 {
                        res = &Compound{p.trivial,elements{elems}}
                } else {
                        res = p
                }
        }
        return
}
func (p *Compound) elemstr(o Object, k elemkind) (s string) {
        var tk = k|elemNoQuote
        for _, elem := range p.Elems { s += elementString(o, elem, tk) }
        if k&elemNoQuote != 0 { return }
        var err error
        var buf bytes.Buffer
        buf.WriteString(`"`)
        defer func() {
                buf.WriteString(`"`)
                s = buf.String()
        } ()
        for i := strings.IndexAny(s, escapedChars); i != -1; {
		if _, err = buf.WriteString(s[:i]); err != nil {
                        err = wrap(p.position, err)
			return
		}
                var esc string
                switch s[i] {
                case '"':  esc = `\"`
                case '\r': esc = `\r`
                case '\n': esc = `\n`
                }
                s = s[i+1:]
                if _, err = buf.WriteString(esc); err != nil {
                        err = wrap(p.position, err)                        
			return
                }
                i = strings.IndexAny(s, escapedChars)
        }
        if _, err = buf.WriteString(s); err != nil {
                err = wrap(p.position, err)
        }
        return
}
func (p *Compound) String() string { return p.elemstr(nil, 0) }
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
func (p *Compound) Float() (f float64, err error) {
        var s string
        if s, err = p.Strval(); err == nil {
                f, err = strconv.ParseFloat(s, 64)
        }
        return
}
func (p *Compound) Integer() (i int64, err error) {
        var s string
        if s, err = p.Strval(); err == nil {
                i, err = strconv.ParseInt(s, 10, 64)
        }
        return
}
func (p *Compound) True() (bool, error) { return p.elements.True() }
func (p *Compound) refs(v Value) bool { return p.elements.refs(v) }
func (p *Compound) closured() bool { return p.elements.closured() }
func (p *Compound) refdef(origin DefOrigin) bool { return p.refdef(origin) }
func (p *Compound) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Compound); ok {
                s1, e := p.Strval()
                if e != nil { return }
                s2, e := a.Strval()
                if e != nil { return }
                if s1 == s2 { res = cmpEqual }
        }
        return
}

type List struct { elements }
func (p *List) elemstr(o Object, k elemkind) (s string) {
        var strs []string
        for _, elem := range p.Elems {
                strs = append(strs, elementString(o, elem, k))
        }
        return strings.Join(strs, " ")
}
func (p *List) Position() (pos Position) {
        if len(p.Elems) > 0 {
                pos = p.Elems[0].Position()
        }
        return
}
func (p *List) Float() (float64, error) {
        i, e := p.Integer(); return float64(i), e
}
func (p *List) Integer() (int64, error) {
        if n := len(p.Elems); n == 1 {
                // If there's only one element, treat it as a scalar.
                return p.Elems[0].Integer()
        } else {
                return int64(n), nil
        }
}
func (p *List) String() (s string) { return p.elemstr(nil, 0) }
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

func (p *List) expand(w expandwhat) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = expandall(w, p.Elems...); err == nil {
                if num > 0 {
                        res = &List{ elements{ elems } }
                } else {
                        res = p
                }
        }
        return
}

func (p *List) traverse(t *traversal) (err error) {
        if len(p.Elems) == 0 { return }
        if optionTraceTraversal { defer un(tt(t, p)) }
        for _, v := range p.Elems {
                if err = v.traverse(t); err != nil { break }
        }
        return
}

func (p *List) exists() (res existence) {
        res = existenceMatterless
ForElems:
        for _, elem := range p.Elems {
                switch elem.exists() {
                case existenceMatterless:
                case existenceConfirmed:
                        res = existenceConfirmed
                case existenceNegated:
                        res = existenceNegated
                        break ForElems
                }
        }
        return
}

func (p *List) stamp(t *traversal) (files []*File, err error) {
        for _, elem := range p.Elems {
                var a []*File
                if a, err = elem.stamp(t); err != nil { break }
                files = append(files, a...)
        }
        return
}

func (p *List) mod(t *traversal) (res time.Time, err error) {
        var a time.Time
        for _, elem := range p.Elems {
                if a, err = elem.mod(t); err == nil { break } else
                if a.After(res) { res = a }
        }
        return
}

func (p *List) cmp(v Value) (res cmpres) {
        if a, ok := v.(*List); ok { res = p.cmpElems(a.Elems) }
        return
}

type Group struct { trivial ; List }
func (p *Group) elemstr(o Object, k elemkind) string {
        var strs []string
        for _, elem := range p.Elems {
                strs = append(strs, elementString(o, elem, k))
        }
        return fmt.Sprintf("(%s)", strings.Join(strs, " "))
}
func (p *Group) mod(t *traversal) (time.Time, error) { return p.trivial.mod(t) }
func (p *Group) Position() Position { return p.trivial.Position() }
func (p *Group) Float() (float64, error) { return p.trivial.Float() }
func (p *Group) Integer() (int64, error) { return p.trivial.Integer() }
func (p *Group) True() (t bool, err error) {
        t = len(p.List.Elems) > 0
        return
}
func (p *Group) String() string { return p.elemstr(nil, 0) }
func (p *Group) Strval() (s string, err error) {
        if s, err = p.List.Strval(); err == nil {
                s = "(" + s + ")"
        }
        return
}
func (p *Group) traverse(t *traversal) (err error) { return }
func (p *Group) stamp(t *traversal) (files []*File, err error) { return }
func (p *Group) exists() existence { return p.List.exists() }
func (p *Group) expand(w expandwhat) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = expandall(w, p.Elems...); err == nil {
                if num > 0 {
                        res = &Group{p.trivial,List{elements{elems}}}
                } else {
                        res = p
                }
        }
        return
}
func (p *Group) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Group); ok { res = p.cmpElems(a.Elems) }
        return
}

func parseGroupValue(g *Group) (result Value) {
        if len(g.Elems) == 0 { return g }
        var word *Bareword
        switch kind := g.Elems[0].(type) {
        case *Bareword: word = kind
        case *Group: if len(kind.Elems) > 0 {
                word, _ = kind.Elems[0].(*Bareword)
        }}
        if word != nil {
                switch word.string {
                case "plain", "json", "yaml", "xml":
                        result = &List{elements{g.Elems[1:]}}
                }
        }
        if result == nil { result = g }
        return
}

type Pair struct { // key=value
        trivial
        Key Value
        Value Value
}
func (p *Pair) refs(v Value) bool { return p.Key.refs(v) || p.Value.refs(v) }
func (p *Pair) closured() bool { return p.Key.closured() || p.Value.closured() }
func (p *Pair) expand(x expandwhat) (res Value, err error) {
        var k, v Value
        res = p // set the original value
        if k, err = p.Key.expand(x); err == nil {
                if v, err = p.Value.expand(x); err == nil {
                        if k != p.Key || v != p.Value {
                                if k == nil { k = p.Key }
                                if v == nil { v = p.Value }
                                res = &Pair{p.trivial,k,v}
                        }
                }
        }
        return
}
func (p *Pair) True() (t bool, err error) {
        if t, err = p.Value.True(); err == nil && !t {
                t, err = p.Key.True()
        }
        return
}
func (p *Pair) elemstr(o Object, k elemkind) string {
        return elementString(o, p.Key, k)+`=`+elementString(o, p.Value, k)
}
func (p *Pair) String() string { return p.elemstr(nil, 0) }
func (p *Pair) Strval() (s string, err error) {
        var k, v string
        if k, err = p.Key.Strval(); err == nil {
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
        p.Key = k
}
func (p *Pair) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Pair); ok {
                if p.Key.cmp(a.Key) == cmpEqual {
                        if p.Value.cmp(a.Value) == cmpEqual {
                                res = cmpEqual
                        }
                }
        }
        return
}

type closuredelegate struct {
        l token.Token
        x Value
        a []Value
}
func (p *closuredelegate) string(o Object, k elemkind) (s string) { // source representation
        for i, a := range p.a {
                if i == 0 { s = " " } else { s += "," }
                s += elementString(o, a, k)
        }

        var name string
        switch x := p.x.(type) {
        case *selection: name = x.String()
        case Object: name = x.Name()
        }

        switch p.l {
        case token.LCOLON:
                if p.x == context.globe.os.self {
                        s = ":os:"
                } else {
                        s = fmt.Sprintf(":%s%s:", name, s)
                }
        case token.LPAREN: s = fmt.Sprintf("(%s%s)", name, s)
        case token.LBRACE:
                if k&elemNoBrace == 0 {
                        s = fmt.Sprintf("{%s%s}", name, s)
                } else {
                        s = fmt.Sprintf("(%s%s)", name, s)
                }
        case token.STRING, token.COMPOUND:
                s = fmt.Sprintf("%s%s", name, s)
        case token.ILLEGAL:
                if len(name) == 1 && len(s) == 0 {
                        s = fmt.Sprintf("%s", name)
                } else {
                        s = fmt.Sprintf("[%s%s]", name, s)
                }
        default:
                s = fmt.Sprintf("[%s%s]!(%v)", name, s, p.l)
        }
        return
}

// Delegate wraps '$(foo a,b,c)' into Valuer
type delegate struct { trivial ; closuredelegate }
func (p *delegate) True() (t bool, err error) {
        var v Value
        if v, err = p.expand(expandAll); err == nil {
                t, err = v.True()
        }
        return
}
func (p *delegate) elemstr(o Object, k elemkind) (s string) {
        if k&elemExpand == 0 {
                s = "$"+p.string(o, k)
        } else if v, e := p.expand(expandDelegate); e == nil {
                s = elementString(o, v, k)
        }
        return
}
func (p *delegate) String() (s string) { return p.elemstr(nil, 0) }
func (p *delegate) value() (v Value, err error) {
        if v, err = p.expand(expandDelegate); err == nil {
                if v == p { // d, ok := v.(*delegate); ok && d == p
                        err = errorf(p.position, "self delegation (%v)", p)
                        if optionPrintStack {
                                fmt.Fprintf(stderr, "%s: %v (%s)\n", p.position, p, typeof(p.x))
                                debug.PrintStack()
                        }
                }
        } else { err = wrap(p.position, err) }
        return
}
func (p *delegate) Strval() (s string, err error) {
        var v Value
        if v, err = p.value(); err == nil { s, err = v.Strval() }
        return
}
func (p *delegate) Integer() (i int64, err error) {
        var v Value
        if v, err = p.value(); err == nil { i, err = v.Integer() }
        return
}
func (p *delegate) Float() (f float64, err error) {
        var v Value
        if v, err = p.value(); err == nil { f, err = v.Float() }
        return
}
func (p *delegate) expand(w expandwhat) (res Value, err error) {
        switch {
        default: res = p
        case w&expandClosure != 0:
                if res, err = p.disclose(); err != nil { return }
                if res != nil && w&expandDelegate != 0 {
                        res, err = res.expand(expandDelegate)
                } else if res == nil { res = p }
        case w&expandDelegate != 0:
                if res, err = p.reveal(); err != nil { return }
                if err == nil && res == nil {
                        if false && optionPrintStack {
                                s, _ := p.x.Strval()
                                fmt.Fprintf(stderr, "%s: %v (%s) (%s)\n", p.position, p.x, typeof(p.x), s)
                                if false { debug.PrintStack() }
                        }
                }
                if res != nil && res == p {
                        err = errorf(p.position, "self delegation (%v)", p)
                        if optionPrintStack {
                                fmt.Fprintf(stderr, "%s\n", err)
                                debug.PrintStack()
                        }
                } else if res != nil && w&expandClosure != 0 {
                        res, err = res.expand(expandClosure)
                }
                if err == nil && res == nil {
                        res = &None{trivial{p.position}}
                }
        }
        return
}
func (p *delegate) reveal() (res Value, err error) {
        if isNil(p.x) { return nil, nil }

        var ( o Object; selected bool )
        switch t := p.x.(type) {
        case Object: o = t
        case *selection:
                if n, ok := t.o.(*ProjectName); ok {
                        defer setclosure(setclosure(cloctx.unshift(n.project.scope)))
                        if false && optionPrintStack {
                                fmt.Fprintf(stderr, "%s: %v %v\n", p.position, t, cloctx)
                                debug.PrintStack()
                        }
                }

                var ( v Value; ok bool )
                if v, err = t.value(); err != nil {
                        err = wrap(p.position, err)
                        return
                } else if o, ok = v.(Object); !ok {
                        res = v
                        return
                }

                if false && optionPrintStack {
                        fmt.Fprintf(stderr, "%s: %v ⇒ %v (%s)\n", p.position, p.x, v, typeof(v))
                        if false { debug.PrintStack() }
                }

                selected = true
        }

        var args []Value
        if args, _, err = expandall(expandClosure, p.a...); err != nil { return }

        var v Value
        switch x := o.(type) {
        default: err = errorf(p.position, "%s '%v' is unknown delegation", typeof(x), x)
        case Caller:
                if res, err = x.Call(p.position, args...); err != nil {
                        if o, ok := x.(Object); ok && o.Name() != "error" {
                                err = wrap(p.position, err)
                        } else {
                                return
                        }
                } else if selected && res != nil {
                        if v, err = res.expand(expandClosure); err != nil { 
                                err = wrap(p.position, err)
                                return
                        } else if v != nil && v != res {
                                res = v
                        }
                }
                if false && optionPrintStack && selected {
                        s, _ := o.Strval()
                        fmt.Fprintf(stderr, "%s: %v; %v; %v (%s)\n", p.position, o, s, res, typeof(res))
                        if false { debug.PrintStack() }
                }
        case Executer:
                if args, err = x.Execute(p.position, args...); err != nil {
                        if o, ok := x.(Object); ok && o.Name() != "error" {
                                err = wrap(p.position, err)
                        } else {
                                return
                        }
                } else { res = &List{elements{args}} }
        }

        if false && selected && res == nil && err == nil {
                fmt.Fprintf(stderr, "%s: %v ⇒ %v (%s) (%v)\n", p.position, p.x, res, typeof(res), o)
                debug.PrintStack()
        }

        if false && optionPrintStack && selected && (res == nil || res == p) {
                fmt.Fprintf(stderr, "%s: %v ⇒ %v (%s)\n", p.position, p.x, res, typeof(res))
                debug.PrintStack()
        }

        if err == nil && res == nil { res = &None{trivial{p.position}} }
        return
}
func (p *delegate) disclose() (res Value, err error) {
        var ( x = p.x; v Value; changed bool )
        if v, err = x.expand(expandClosure); err != nil { return }
        if v != nil && v != x { changed, x = true, v }

        var args []Value
        for _, a := range p.a {
                if v, err = a.expand(expandClosure); err != nil { return }
                if v != nil { a, changed = v, true }
                args = append(args, a)
        }
        if err == nil {
                if changed {
                        res = &delegate{p.trivial,closuredelegate{p.l,x,args}}
                } else {
                        res = p
                }
        }
        return
}
func (p *delegate) refs(v Value) bool {
        if p.x == v || p.x.refs(v) { return true }
        for _, a := range p.a {
                if a.refs(v) { return true }
        }
        return false
}
func (p *delegate) closured() bool {
        if p.x.closured() { return true }
        for _, a := range p.a {
                if a.closured() { return true }
        }
        return false
}
func (p *delegate) refdef(origin DefOrigin) (res bool) {
        if d, ok := p.x.(*Def); ok { res = d.origin == origin }
        return
}
func (p *delegate) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        var val Value
        if val, err = p.expand(expandAll); err == nil {
                err = t.dispatch(val)
        }
        return
}
func (p *delegate) mod(t *traversal) (res time.Time, err error) {
        var val Value
        if val, err = p.expand(expandAll); err == nil {
                res, err = val.mod(t)
        }
        return
}
func (p *delegate) cmp(v Value) (res cmpres) {
        if a, ok := v.(*delegate); ok {
                // FIXME: compare the expanded value instead??
                if p.x.cmp(a.x) == cmpEqual && len(p.a) == len(a.a) {
                        for i, t := range p.a {
                                if t.cmp(a.a[i]) != cmpEqual { return }
                        }
                        res = cmpEqual
                }
        } else if d, ok := p.x.(*Def); ok && len(p.a) == 0 {
                res = d.value.cmp(v)
        }
        return
}

type closure struct { trivial ; closuredelegate }
func (p *closure) True() (t bool, err error) {
        var v Value
        if v, err = p.expand(expandAll); err == nil {
                t, err = v.True()
        }
        return
}
func (p *closure) elemstr(o Object, k elemkind) (s string) {
        if k&elemExpand == 0 {
                s = "&"+p.string(o, k)
        } else if v, e := p.expand(expandDelegate); e == nil {
                s = elementString(o, v, k)
        }
        return
}
func (p *closure) String() (s string) { return p.elemstr(nil, 0) }
func (p *closure) Strval() (s string, err error) {
        var v Value

        // &(...) -> $(...)
        if v, err = p.expand(expandClosure); err != nil {
                return
        } else if v == nil {
                //err = fmt.Errorf("{closure %+v &<nil>}", p.o)
                return
        }

        // $(...) -> .....
        if v, err = v.expand(expandDelegate); err != nil {
                return
        } else if v != nil {
                s, err = v.Strval()
        } else {
                //err = fmt.Errorf("{closure %+v $<nil>}", p.o)
        }
        return
}
func (p *closure) expand(w expandwhat) (res Value, err error) {
        switch {
        case w&expandClosure != 0:
                if res, err = p.disclose(); err != nil { return }
                if res != nil && w&expandDelegate != 0 {
                        res, err = res.expand(expandDelegate)
                }
        case w&expandDelegate != 0:
                if res, err = p.reveal(); err != nil { return }
                if res != nil && w&expandClosure != 0 {
                        res, err = res.expand(expandClosure)
                }
        }
        if err == nil && res == nil { res = p }
        return
}
func (p *closure) reveal() (res Value, err error) {
        if p.x == nil { return }

        var ( t Value; x = p.x )
        if t, err = p.x.expand(expandDelegate); err != nil { return }
        if t != nil && t != x { x = t }
        
        var ( a []Value; num int )
        for _, v := range p.a {
                if t, err = v.expand(expandDelegate); err != nil { return }
                if t == nil { t = v } else { num = num + 1 }
                a = append(a, t)
        }

        if x != nil || num > 1 {
                res = &closure{p.trivial,closuredelegate{p.l,x,a}}
        }
        return
}
func (p *closure) disclose() (res Value, err error) {
        if isNil(p.x) { return nil, nil }
        
        var o Object
        switch t := p.x.(type) {
        case Object: o = t
        case *selection:
                if n, ok := t.o.(*ProjectName); ok {
                        defer setclosure(setclosure(cloctx.unshift(n.project.scope)))
                        if false && optionPrintStack {
                                fmt.Fprintf(stderr, "%s: %v %v\n", p.position, t, cloctx)
                                debug.PrintStack()
                        }
                }

                var ( v Value; ok bool )
                if v, err = t.value(); err != nil {
                        err = wrap(p.position, err)
                        return
                } else if o, ok = v.(Object); !ok {
                        // Does nothing!
                        return
                }
        }

        var changed bool
        var name = o.Name()
        ClosureTok: switch p.l {
        case token.LPAREN, token.ILLEGAL:
                for _, scope := range cloctx {
                        var s Object
                        if scope.project == nil {
                                if _, s = scope.Find(name); !isNil(s) {
                                        o, changed = s, true
                                        break ClosureTok
                                }
                                continue
                        }
                        if scope != scope.project.scope {
                                // inquire non-project scope first
                                if _, s = scope.Find(name); !isNil(s) {
                                        o, changed = s, true
                                        break ClosureTok
                                }
                        }
                        if s, err = scope.project.resolveObject(name); err != nil { return }
                        if !isNil(s) { o, changed = s, true; break ClosureTok }
                }
        case token.LBRACE, token.STRING, token.COMPOUND:
                for _, scope := range cloctx {
                        var s Object
                        if s, err = scope.project.resolveEntry(name); err != nil { return }
                        if !isNil(s) {
                                if p.l == token.LBRACE {
                                        o, changed = s, true
                                        break ClosureTok
                                }
                                
                                // &'xxx' and &"xxx" are resolving
                                // objects in the closure context.
                                res = s; return
                        }
                }
        default:
                err = errorf(p.position, "unknown closure `&%+v%+v`", p.l, name)
                return
        }

        var v Value
        if isNil(o) {
                err = errorf(p.position, "'%s' is nil (%T %v)", name, p.x, p.x)
                if optionPrintStack {
                        fmt.Fprintf(stderr, "%v\n%v\n", err, cloctx)
                        debug.PrintStack()
                }
                return
        } else if v, err = o.expand(expandClosure); err != nil {
                err = wrap(p.position, err)
                return
        } else if !isNil(v) {
                var ( s Object; ok bool )
                if s, ok = v.(Object); !ok || isNil(s) {
                        err = errorf(p.position, "invalid closure %+v", v)
                        return
                }

                o, changed = s, true
        }

        var args []Value
        for _, a := range p.a {
                if v, err = a.expand(expandClosure); err != nil { return }
                if !isNil(v) { a, changed = v, true }
                args = append(args, a)
        }

        if changed && err == nil {
                res = &delegate{p.trivial,closuredelegate{p.l,o,args}}
        }
        return
}
func (p *closure) refs(v Value) bool {
        if p.x == v { return true }
        for _, a := range p.a { if a.refs(v) { return true }}
        return false
}
func (p *closure) closured() bool { return true }
func (p *closure) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        if v, e := p.expand(expandClosure); e != nil { err = e } else
        if v == nil {
                //err = fmt.Errorf("undefined closure target `%v`", p.o.Name())
                //fmt.Fprintf(stderr, "%s: closure.prepare: %v\n", p.position, err)
                err = errorf(p.position, "invalid closure (%v)", p.x)
        } else {
                err = t.dispatch(v)
        }
        return
}
func (p *closure) mod(t *traversal) (res time.Time, err error) {
        var val Value
        if val, err = p.expand(expandAll); err == nil {
                res, err = val.mod(t)
        }
        return
}
func (p *closure) cmp(v Value) (res cmpres) {
        if a, ok := v.(*closure); ok {
                // FIXME: compare the expanded value instead??
                if p.x.cmp(a.x) == cmpEqual && len(p.a) == len(a.a) {
                        for i, t := range p.a {
                                if t.cmp(a.a[i]) != cmpEqual { return }
                        }
                        res = cmpEqual
                }
        }
        return
}

type selection struct {
        trivial
        t token.Token
        o Value // Object or selection
        s Value
}
func (p *selection) True() (t bool, err error) {
        var v Value
        if v, err = p.value(); err == nil {
                t, err = v.True()
        }
        return
}
func (p *selection) elemstr(o Object, k elemkind) (s string) {
        if _, ok := p.o.(*usinglist); ok { s = "usee" } else {
                s = elementString(o, p.o, k)
        }
        s += p.t.String() + elementString(o, p.s, k)
        return
}
func (p *selection) String() string { return p.elemstr(nil, 0) }
func (p *selection) objectName() (s string) {
        switch t := p.o.(type) {
        case Object: s = t.Name()
        }
        return
}
func (p *selection) propName() (s string) {
        switch t := p.s.(type) {
        case Object: s = t.Name()
        case *Bareword: s = t.string
        case *String: s = t.string
        }
        return
}
func (p *selection) object() (o Object, err error) {
        if s, ok := p.o.(*selection); ok {
                var v Value
                if v, err = s.value(); err != nil {
                        // sth's wrong!
                } else if o, _ = v.(Object); o == nil {
                        err = errorf(p.position, "selection.object: `%s` is nil", s.String())
                }
        } else if o, ok = p.o.(Object); !ok {
                err = errorf(p.position, "selection.object: %T '%v' is not object", p.o, p.o)
        }
        return
}
func (p *selection) value() (v Value, err error) {
        var o Object
        if p.s == nil {
                err = errorf(p.position, "selection.value: nil prop `%s`", p.String())
        } else if o, err = p.object(); err != nil {
                // sth's wrong!
        } else if s := ""; o != nil {
                if s, err = p.s.Strval(); err == nil {
                        if pn, ok := o.(*ProjectName); ok && (p.t == token.SELECT_PROG1 || p.t == token.SELECT_PROG2) {
                                var entry *RuleEntry
                                if entry, err = pn.project.resolveEntry(s); err != nil {
                                        return
                                } else if entry == nil {
                                        err = errorf(p.position, "selection.value: no entry `%s` (%+v)", s, p.String())
                                } else {
                                        v = entry
                                }
                        } else if v, err = o.Get(s); err != nil {
                                err = wrap(p.position, err)
                                if false && optionPrintStack {
                                        fmt.Fprintf(stderr, "%s: %v %v\n", p.position, p, cloctx)
                                        debug.PrintStack()
                                }
                        }
                }
        } else /*if o == nil*/ {
                err = errorf(p.position, "selection.value: nil object `%s`", p.String())
        }
        return
}
func (p *selection) Strval() (s string, err error) {
        if n, ok := p.o.(*ProjectName); ok && n != nil {
                defer setclosure(setclosure(cloctx.unshift(n.project.scope)))
                if false && optionPrintStack {
                        fmt.Fprintf(stderr, "%s: %v %v\n", p.position, p, cloctx)
                        debug.PrintStack()
                }
        }

        var v Value
        if v, err = p.value(); err != nil {
                err = wrap(p.position, err)
        } else if v != nil {
                if s, err = v.Strval(); err != nil { err = wrap(p.position, err) }
                if false && optionPrintStack {
                        fmt.Fprintf(stderr, "%s: %v → %v\n", p.position, v, s)
                        debug.PrintStack()
                }
        } else if false {
                err = errorf(p.position, "selection.strval: `%s` is nil", p.String())
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
func (p *selection) expand(w expandwhat) (res Value, err error) {
        var o, s Value
        if p.o != nil {
                if o, err = p.o.expand(w); err != nil { return } else
                if o == nil { o = p.o }
        }
        if p.s != nil {
                if s, err = p.s.expand(w); err != nil { return } else
                if s == nil { s = p.s }
        }
        if o != p.o || s != p.s {
                res = &selection{p.trivial,p.t,o,s}
        } else {
                res = p
        }
        return
}
func (p *selection) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        var v Value
        if v, err = p.value(); err != nil {
                // sth's wrong
        } else if v == nil {
                err = fmt.Errorf("`%v` is nil", p)
        } else {
                err = t.dispatch(v)
        }
        return
}
func (p *selection) mod(t *traversal) (res time.Time, err error) {
        var v Value
        if v, err = p.value(); err == nil {
                if v == nil {
                        err = errorf(p.position, "selection is nil")
                } else {
                        res, err = v.mod(t)
                }
        }
        return
}
func (p *selection) cmp(v Value) (res cmpres) {
        if a, ok := v.(*selection); ok {
                if p.o.cmp(a.o) == cmpEqual && p.s.cmp(a.s) == cmpEqual {
                        if p.t == a.t { res = cmpEqual }
                }
        }
        return
}

type partialMatcher interface {
        partialMatch(i interface{}) (result string, rest, stems []string, err error)
}

// TODO: endingMatcher is not implemented (e.g. $(trim-suffix .%, a.xxx b.xxx))
type endingMatcher interface {
        endingMatch(i interface{}) (result string, rest, stems []string, err error)
}

// Pattern
type Pattern interface {
        Value
        match(i interface{}) (s string, stems []string, err error)
        stencil(stems []string) (s string, rest []string, err error)
}

// PercPattern represents percent pattern expressions (e.g. '%.o')
type PercPattern struct {
        trivial // TODO: supporting multiple %: foo%bar%xxx
        Prefix Value
        Suffix Value
}
func (p *PercPattern) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *PercPattern) elemstr(o Object, k elemkind) (s string) {
        s  = elementString(o, p.Prefix, k) + `%`
        s += elementString(o, p.Suffix, k)
        return
}
func (p *PercPattern) String() string { return p.elemstr(nil, 0) }
func (p *PercPattern) Strval() (s string, err error) {
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
func (p *PercPattern) match(i interface{}) (result string, stems []string, err error) {
        if optionEnableBenchspots { defer bench(spot("PercPattern.match")) }
        var s string
        switch t := i.(type) {
        case string: s = t
        case *File: s = t.name
        case *filestub: s = t.name
        case Value:
                s, err = t.Strval()
                if err != nil { return }
        default:
                unreachable(fmt.Sprintf("perc.match: %T %v", i, i))
        }

        var prefix string
        if p.Prefix == nil {
                // ...
        } else if !isNone(p.Prefix) {
                prefix, err = p.Prefix.Strval()
                if err != nil { return }
                if !strings.HasPrefix(s, prefix) {
                        return
                }
        }

        switch t := p.Suffix.(type) {
        case *None:
                if a, b := len(prefix), len(s); a < b {
                        stems = []string{ s[a:] }
                        result = s
                }                
        case Pattern:
                if a, b := len(prefix), len(s); a < b {
                        var res string
                        res, stems, err = t.match(s[a:])
                        if res != "" && len(stems) > 0 {
                                result = s
                        }
                }
        default:
                var suffix string
                suffix, err = t.Strval()
                if err != nil { break }
                if !strings.HasSuffix(s, suffix) { break }
                if a, b := len(prefix), len(s)-len(suffix); a < b {
                        stems = []string{ s[a:b] }
                        result = s
                }
        }
        return
}
func (p *PercPattern) stencil(stems []string) (s string, rest []string, err error) {
        if optionEnableBenchmarks && false { defer bench(mark(fmt.Sprintf("PercPattern.stencil(%v)", p))) }
        if optionEnableBenchspots { defer bench(spot("PercPattern.stencil")) }

        if !isNone(p.Prefix) {
                // FIXME: the prefix could be Glob, Regexp, etc.
                s, err = p.Prefix.Strval()
                if err != nil { return }
        }

        var v string
        if isNone(p.Suffix) {
                s += stems[0]
                rest = stems[1:]
        } else if pp, ok := p.Suffix.(*PercPattern); ok {
                // patterns like '%%...' use only one stem,
                // patterns like '%xxx%...' use multiple stems.
                if !isNone(pp.Prefix) {
                        s += stems[0]
                        stems = stems[1:]
                }
                if v, rest, err = pp.stencil(stems); err == nil { s += v }
        } else if pp, ok := p.Suffix.(Pattern); ok {
                if v, rest, err = pp.stencil(stems); err == nil { s += v }
        } else if v, err = p.Suffix.Strval(); err == nil {
                s += stems[0] + v
                rest = stems[1:]
        }
        return
}
func (p *PercPattern) refs(v Value) bool { return p.Prefix.refs(v) || p.Suffix.refs(v) }
func (p *PercPattern) closured() bool { return p.Prefix.closured() || p.Suffix.closured() }
func (p *PercPattern) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("PercPattern.traverse(%v)", p))) }
        if optionEnableBenchspots { defer bench(spot("PercPattern.traverse")) }
        if t.stems == nil { err = errorf(p.position, "no stems"); return }
        var ( rest []string; target string )
        if target, rest, err = p.stencil(t.stems); err != nil {
                // oops...
        } else if len(rest) > 0 || target == "" {
                // just relax
        } else if err = t.target(p.position, target); err == nil {
                //t.addNewTarget(&String{trivial{p.position},target})
        }
        return
}
func (p *PercPattern) cmp(v Value) (res cmpres) {
        if a, ok := v.(*PercPattern); ok {
                if p.Prefix.cmp(a.Prefix) == cmpEqual {
                        if p.Suffix.cmp(a.Suffix) == cmpEqual {
                                res = cmpEqual
                        }
                }
        }
        return
}

// Check for patterns like foo%%bar
func percperc(p Pattern) (t bool, prefix, suffix Value) {
        if p1, ok := p.(*PercPattern); ok {
                if p2, ok := p1.Suffix.(*PercPattern); ok {
                        // assert(isNone(p2.Prefix))
                        prefix = p1.Prefix
                        suffix = p2.Suffix
                        t = true
                }
        }
        return
}

// GlobPattern represents glob pattern expressions (e.g. '*.o', '[a-z].o', 'a?a.o')
// 
// The pattern syntax is:
//
//	pattern:
//		{ term }
//	term:
//		'*'         matches any sequence of non-Separator characters
//		'?'         matches any single non-Separator character
//		'[' [ '^' ] { character-range } ']'
//		            character class (must be non-empty)
//		c           matches character c (c != '*', '?', '\\', '[')
//		'\\' c      matches character c
//
//	character-range:
//		c           matches character c (c != '\\', '-', ']')
//		'\\' c      matches character c
//		lo '-' hi   matches character c for lo <= c <= hi
type GlobPattern struct {
        trivial
        Components []Value
}
func (p *GlobPattern) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *GlobPattern) elemstr(o Object, k elemkind) (s string) {
        for _, comp := range p.Components {
                s += elementString(o, comp, k)
        }
        return
}
func (p *GlobPattern) String() (s string) { return p.elemstr(nil, 0) }
func (p *GlobPattern) Strval() (s string, err error) {
        for _, comp := range p.Components {
                var v string
                if v, err = comp.Strval(); err != nil {
                        return
                }
                s += v
        }
        return
}
func (p *GlobPattern) match(i interface{}) (result string, stems []string, err error) {
        if optionEnableBenchspots { defer bench(spot("GlobPattern.match")) }
        var pat, s string
        switch t := i.(type) {
        case string: s = t
        case *File: s = t.name
        case *filestub: s = t.name
        case Value:
                s, err = t.Strval()
                if err != nil { return }
        default:
                unreachable("glob.match: %T %v", i, i)
        }
        if pat, err = p.Strval(); err == nil {
                var matched bool
                matched, err = filepath.Match(pat, s)
                if matched { result = s }
        }
        // FIXME: calculate stems from matching
        return
}
func (p *GlobPattern) stencil(stems []string) (s string, rest []string, err error) {
        unreachable(fmt.Sprintf("Unimplemented GlobPattern stencil %v (stems=%v)", p, stems))
        return
}
/*
func (p *GlobPattern) concrete(patent *RuleEntry, stem string) (entry *RuleEntry, err error) {
        var target string
        if target, err = p.stencil(stem); err == nil {
                entry, err = p.pattern.concrete(patent, target, stem)
        }
        return
}
*/
func (p *GlobPattern) refs(v Value) (res bool) {
        for _, comp := range p.Components {
                if res = comp.refs(v); res { break }
        }
        return
}
func (p *GlobPattern) closured() (res bool) {
        for _, comp := range p.Components {
                if res = comp.closured(); res { break }
        }
        return
}
func (p *GlobPattern) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        if t.stems == nil { return }

        var target string
        var rest []string
        if target, rest, err = p.stencil(t.stems); err != nil {
                // oops...
        } else if len(rest) > 0 || target == "" {
                // just relax
        } else {
                err = t.target(p.position, target)
        }
        return
}
func (p *GlobPattern) cmp(v Value) (res cmpres) {
        if a, ok := v.(*GlobPattern); ok {
                if len(p.Components) == len(a.Components) {
                        for i, c := range p.Components {
                                if c.cmp(a.Components[i]) != cmpEqual {
                                        return
                                }
                        }
                        res = cmpEqual
                }
        }
        return
}

// TODO: implement regexp pattern
type RegexpPattern struct { trivial }
func (p *RegexpPattern) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *RegexpPattern) String() string { return "{RegexpPattern}" }
func (p *RegexpPattern) Strval() (s string, err error) { return "", nil }
func (p *RegexpPattern) match(i interface{}) (result string, stems []string, err error) {
        if optionEnableBenchspots { defer bench(spot("RegexpPattern.match")) }
        unreachable("regexp.match: %T %v", i, i)
        return
}
func (p *RegexpPattern) stencil(stems []string) (s string, rest []string, err error) {
        unreachable("regexp.stencil: %v", stems)
        return
}
func (p *RegexpPattern) cmp(v Value) (res cmpres) {
        if a, ok := v.(*RegexpPattern); ok {
                if a != nil { /* FIXME: ... */ }
        }
        return
}

func NewRegexpPattern() Pattern {
        return &RegexpPattern{}
}

type Valuer interface {
        Value() Value
}

type Caller interface {
        Call(pos Position, args... Value) (Value, error)
}

type Executer interface {
        Execute(pos Position, a... Value) (result []Value, err error)
}

type Positioner interface {
        Position() Position
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
func Reveal(values ...Value) (res []Value, err error) {
        for _, v := range values {
                //if v, err = Reveal(v); err != nil { break }
                if v, err = v.expand(expandDelegate); err != nil { break }
                if v != nil { res = append(res, v) }
        }
        return
}

// Disclose expands closures to normal value recursively.
func Disclose(values ...Value) (res []Value, err error) {
        for _, v := range values {
                if v, err = v.expand(expandClosure); err != nil { break }
                if v != nil { res = append(res, v) }
        }
        return
}

func values(args... interface{}) (elems []Value) {
        for _, a := range args {
                if v, ok := a.(Value); ok {
                        elems = append(elems, v)
                } else {
                        unreachable()
                }
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

func trueVal(v Value, i bool) (res bool, err error) {
        if res = i; v != nil { res, err = v.True() }
        return
}

func int64Val(v Value, i int64) (res int64, err error) {
        if res = i; v != nil { res, err = v.Integer() }
        return
}

func intVal(v Value, i int) (res int, err error) {
        if res = i; v != nil {
                var i int64
                if i, err = v.Integer(); err == nil {
                        res = int(i)
                }
        }
        return
}

func uintVal(v Value, i uint32) (res uint32, err error) {
        if res = i; v != nil {
                var i int64
                if i, err = v.Integer(); err == nil {
                        res = uint32(i)
                }
        }
        return
}

func permVal(v Value, i uint32) (res os.FileMode, err error) {
        if i, err = uintVal(v, i); err == nil {
                res = os.FileMode(i) & os.ModePerm
        }
        return
}

var expanddepth int64 = 0
func expandall(w expandwhat, values ...Value) (res []Value, num int, err error) {
        defer func(i int64) { expanddepth = i } (expanddepth)
        if expanddepth += 1; expanddepth > 128 {
                err = fmt.Errorf("exceeds maximum expand depth")
                return
        }

        var v Value
        for _, elem := range values {
                if elem == nil {
                        res = append(res, &Nil{})
                } else if v, err = elem.expand(w); err == nil {
                        if v != elem { num += 1 }
                        res = append(res, v)
                } else {
                        break //res = append(res, elem)
                }
        }
        return
}

func ExpandAll(values ...Value) (res []Value, err error) {
        if res, _, err = expandall(expandAll, values...); err == nil {
                // second expand to ensure having real value
                res, _, err = expandall(expandAll, res...)
        }
        return
}

func Refs(a Value, v Value) bool { return a.refs(v) }

func Scalar(v Value) (res Value) {
        if l, o := v.(*List); l != nil && o && l.Len() > 0 {
                res = Scalar(l.Elems[0])
        } else {
                res = v
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

func MakeAnswer(pos Position, v bool) (res Value) {
        if v {
                res = &boolean{trivial{pos},true}
        } else {
                res = &boolean{trivial{pos},false}
        }
        return
}
func MakeBoolean(pos Position, v bool) (res Value) {
        if v {
                res = &answer{trivial{pos},true}
        } else {
                res = &answer{trivial{pos},false}
        }
        return
}
func MakeBin(pos Position, i int64) *Bin { return &Bin{integer{trivial{pos},i}} }
func MakeOct(pos Position, i int64) *Oct { return &Oct{integer{trivial{pos},i}} }
func MakeInt(pos Position, i int64) *Int { return &Int{integer{trivial{pos},i}} }
func MakeHex(pos Position, i int64) *Hex { return &Hex{integer{trivial{pos},i}} }
func MakeFloat(pos Position, f float64) *Float { return &Float{trivial{pos},f} }
func MakeDate(pos Position, s time.Time) *Date { return &Date{DateTime{trivial{pos},s}} }
func MakeTime(pos Position, t time.Time) *Time { return &Time{DateTime{trivial{pos},t}} }
func MakeString(pos Position, s string) *String { return &String{trivial{pos},s} }
func MakeURL(pos Position, s *url.URL) *URL {
        var host, port string
        v := strings.Split(s.Host, ":")
        if len(v) == 1 { host = v[0] }
        if len(v) == 2 { host, port = v[0], v[1] }
        var password Value
        if t, ok := s.User.Password(); ok {password = &String{trivial{pos},t}}
        return &URL{ // FIXME: calculate component positions
                trivial: trivial{pos},
                Scheme: &String{trivial{pos},s.Scheme},
                Username: &String{trivial{pos},s.User.Username()},
                Password: password,
                Host: &String{trivial{pos},host},
                Port: &String{trivial{pos},port},
                Path: &String{trivial{pos},s.Path},
                Query: &String{trivial{pos},s.RawQuery},
                Fragment: &String{trivial{pos},s.Fragment},
        }
}
func MakeBarecomp(pos Position, elems... Value) *Barecomp { return &Barecomp{trivial{pos},elements{elems}} }
func MakeCompound(pos Position, elems... Value) *Compound { return &Compound{trivial{pos},elements{elems}} }
func MakeList(pos Position, elems... Value) *List { return &List{elements{elems}} }
func MakeGroup(pos Position, elems... Value) (v *Group) { return &Group{trivial{pos},List{elements{elems}}} }
func MakeGlobMeta(pos Position, tok token.Token) *GlobMeta { return &GlobMeta{trivial{pos},tok} }
func MakeGlobRange(pos Position, v Value) *GlobRange { return &GlobRange{trivial{pos},v} }
func MakePath(pos Position, segments... Value) (v *Path) { return &Path{trivial{pos},elements{segments}/*, nil*/} }
func MakePathSeg(pos Position, ch rune) *PathSeg { return &PathSeg{trivial{pos},ch} }
func MakePathStr(pos Position, str string) (v *Path) {
        var segments []Value
        for _, s := range strings.Split(str, PathSep) {
                // TODO: calculate position of each segment
                segments = append(segments, &Bareword{trivial{pos},s})
        }
        return MakePath(pos, segments...)
}
func MakePair(pos Position, k, v Value) (p *Pair) {
        p = &Pair{trivial{pos},nil,nil}
        p.SetKey(k)
        p.SetValue(v)
        return
}
func MakePercPattern(pos Position, prefix, suffix Value) Pattern {
        if prefix == nil { prefix = &None{} }
        if suffix == nil { suffix = &None{} }
        return &PercPattern{
                trivial: trivial{pos},
                Prefix: prefix,
                Suffix: suffix,
        }
}
func MakeGlobPattern(pos Position, components... Value) Pattern {
        return &GlobPattern{trivial:trivial{pos},Components:components}
}
func MakeDelegate(pos Position, tok token.Token, obj Value, args... Value) Value {
        return &delegate{trivial{pos},closuredelegate{tok,obj,args}}
}
func MakeClosure(pos Position, tok token.Token, obj Value, args... Value) Value {
        if obj == nil { panic("closure of nil") }
        return &closure{trivial{pos},closuredelegate{tok,obj,args}}
}
func MakeListOrScalar(pos Position, elems []Value) (res Value) {
        if x := len(elems); x > 1 {
                res = &List{elements{elems}}
        } else if x == 1 {
                res = elems[0]
        } else {
                res = &None{trivial{/*pos*/}}
        }
        return
}

func Make(pos Position, in interface{}) (out Value) {
        switch v := in.(type) {
        case int:       out = MakeInt(pos,int64(v))
        case int32:     out = MakeInt(pos,int64(v))
        case int64:     out = MakeInt(pos,v)
        case float32:   out = MakeFloat(pos,float64(v))
        case float64:   out = MakeFloat(pos,v)
        case string:    out = &String{trivial{pos},v}
        case time.Time: out = &DateTime{trivial{pos},v} // FIXME: NewDate, NewTime
        case Value:     out = v
        default:        out = &Any{in} // TODO: position for any
        }
        return
}

func MakeAll(pos Position, in... interface{}) (out []Value) {
        for _, v := range in {
                // TODO: position for each element
                out = append(out, Make(pos,v))
        }
        return
}

func ParseBin(pos Position, s string) *Bin {
        if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
                s = s[2:]
        }
        if i, e := strconv.ParseInt(s, 2, 64); e == nil {
                return MakeBin(pos,i)
        } else {
                panic(e)
        }
}

func ParseOct(pos Position, s string) *Oct {
        if strings.HasPrefix(s, "0") {
                s = s[1:]
        }
        if i, e := strconv.ParseInt(s, 8, 64); e == nil {
                return MakeOct(pos,i)
        } else {
                panic(e)
        }
}

func ParseInt(pos Position, s string) *Int {
        if i, e := strconv.ParseInt(s, 10, 64); e == nil {
                return MakeInt(pos,i)
        } else {
                panic(e)
        }
}

func ParseHex(pos Position, s string) *Hex {
        if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
                s = s[2:]
        }
        if i, e := strconv.ParseInt(s, 16, 64); e == nil {
                return MakeHex(pos,i)
        } else {
                panic(e)
        }
}

func ParseFloat(pos Position, s string) *Float {
        if f, e := strconv.ParseFloat(strings.Replace(s, "_", "", -1), 64); e == nil {
                return MakeFloat(pos,f)
        } else {
                panic(e)
        }
}

func ParseDate(pos Position, s string) *Date {
        if t, e := time.Parse("2006-01-02", s); e == nil {
                return MakeDate(pos,t)
        } else {
                panic(e)
        }
}

func ParseTime(pos Position, s string) *Time {
        if t, e := time.Parse("15:04:05.999999999Z07:00", s); e == nil {
                return MakeTime(pos,t)
        } else {
                panic(e)
        }
}

func ParseURL(pos Position, s string) *URL {
        if u, e := url.Parse(s); e == nil {
                return MakeURL(pos,u)
        } else {
                panic(e)
        }
}

func get_filename(n int) string {
        var num int
        var filename string
        var lines = strings.Split(string(debug.Stack()), "\n")
        for _, line := range lines {
                if !strings.HasPrefix(line, "\t") { continue }
                if i := strings.Index(line, ":"); num == n && i > 0 {
                        filename = line[1:i]
                        break
                }
                num += 1
        }
        return filename
}
