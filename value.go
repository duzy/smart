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
    "io/fs"
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
    expandClosure   // &(...)            ->  $(...)
    expandSelection // foo->bar          -> ...
    expandArgs      // $(foo $(x),$(y))  -> $(foo ...,...)
    expandArgedArgs // foo($(args))      -> foo(...)
    expandDef       // foo=...           -> ...
    expandFullname  // foobar.c          -> /path/to/foobar.c
    expandPathStr   // "/path/to"/foo    -> /path/to/foo
    expandPairVal   // foo=$(bar)        -> foo=...
    expandPlainValue = expandClosure | expandDelegate | expandSelection | expandDef | expandPathStr | expandArgedArgs
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

// A Comment node represents a single #-style comment.
type Comment struct {
	Pos Position // position of "#" starting the comment
	Text  string // comment text (excluding '\n')
}

// A CommentGroup represents a sequence of comments
// with no other tokens and no empty lines between.
type CommentGroup struct {
	List []*Comment // len(List) > 0
}

func (g *CommentGroup) Position() Position { return g.List[0].Pos }

type statinfo struct {
    file *File
    next *statinfo
}
func (si *statinfo) mod() (res time.Time) {
    for p := si; p != nil; p = p.next {
        if p.file != nil && p.file.info != nil {
            if t := p.file.info.ModTime(); t.After(res) { res = t }
        }
    }
    return
}
func (si *statinfo) exists() (res existence) {
    res = existenceMatterless
ForStatInfos:
    for p := si; p != nil; p = p.next {
        if  p.file != nil { // matterless is nil file
            if p.file.exists() {
                res = existenceConfirmed
            } else {
                res = existenceNegated
                break ForStatInfos
            }
        }
    }
    return
}

// Value represents a value of a type.
type Value interface {
    Positioner // The position where the value appears (or NoPos).

    // Literal representations of the value.
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

    // whether this value can be used as a pattern
    patterned() bool

    // Match a Value or string, returned 's' is the matched string (or heading part).
    match(i interface{}) (full bool, s string, stems []string)

    // Stencil this value with stems.
    stencil(stems []string) (s string, rest []string)

    stat(t *traversal) (*statinfo)

    // Stamp the value if it's a file (aka. update FileInfo).
    stamp(t *traversal) ([]*File, error)

    // Recursively detecting whether this value references
    // the object (to avoid loop-delegation).
    refs(v Value) bool

    // Returns all defs of name `s` used in this value.
    defs(s string) []*Def

    // Test if this value is expandible for some bits.
    expandible(expandwhat) bool

    // &(...)        -> $(...)
    // $(...)        -> ......
    // $(...)=$(...) -> ...=$(...), ...=...
    // foo->bar      -> ...
    expand(expandwhat) (Value, error) // result is nil or identical to this value if no expansions

    traverse(t *traversal)
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

func refdef(val Value, origin Origin) (res bool) {
    var defs = val.defs("")
    for _, def := range defs {
        if     origin == defany { res = true; break }
        if def.origin == origin { res = true; break }
    }
    return
}

// traversal prepares prerequisites of targets.
type traversal struct {
    program *Program
    project *Project // program.project or caller.project (if (closure))
    closure *Scope // program.scope or caller.closure (if (closure))

    start time.Time // start time

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
    grepping bool

    updated []*updatedtarget // prerequisites newer than the target (from comparer) ($?)
    stems   []string // set by StemmedEntry

    traceLevel int

    breakers []*breaker
    interpreted []interpreter
    isConfigureExecution bool

    print bool // printing work directories (Entering/Leaving)
    debug bool
}

func (t *traversal) level(n int) { t.traceLevel += n }
func (t *traversal) trace(a ...interface{}) { printIndentDots(t.traceLevel, a...) }
func (t *traversal) tracef(s string, a ...interface{}) { printIndentDots(t.traceLevel, fmt.Sprintf(s, a...)) }

func (t *traversal) traceCallStack(pos Position, s string, a ...interface{}) (point *diagnostic) {
    point = diag.errorAt(pos, s, a...)
    point = diag.errorAt(t.program.position, "from here for %v", t.entry)
    for c, last := t, t.program.position; c != nil && c.program != nil; c = c.caller {
        if pos := c.program.position; !pos.SameLine(&last) {
            point = diag.errorAt(pos, "and here for %v", c.entry) //.debug(optionDebugErrors && c == nil)
            last = pos
        }
    }
    return
}

func (t *traversal) hasBreakers(what ...breakind) (res bool) {
    if len(what) == 0 { res = len(t.breakers) > 0 } else {
    ForBreakers:
        for _, brk := range t.breakers {
            for _, w := range what {
                if brk.what == w { res = true; break ForBreakers }
            }
        }
    }
    return
}
func (t *traversal) breakersNot(what ...breakind) (res []*breaker) {
ForBreakers:
    for _, brk := range t.breakers {
        for _, w := range what {
            if brk.what == w { continue ForBreakers }
        }
        res = append(res, brk)
    }
    return
}
func (t *traversal) breakersOf(what ...breakind) (res []*breaker) {
ForBreakers:
    for _, brk := range t.breakers {
        for _, w := range what {
            if brk.what == w {
                res = append(res, brk)
                continue ForBreakers
            }
        }
    }
    return
}
func (t *traversal) _break(pos Position, what breakind) *breaker {
    var brk = &breaker{ pos:pos, what:what }
    t.breakers = append(t.breakers, brk)
    return brk
}

func (t *traversal) _breakf(pos Position, what breakind, s string, a... interface{}) *breaker {
    var brk = t._break(pos, what)
    brk.message = fmt.Sprintf(s, a...)
    brk.scope = breakGroup
    return brk
}

func (t *traversal) add(target Value) {
    if isNil(target) || isNone(target) || t.targets == nil { return }
    if t.targets.value == target { return }
    if t.targets.value.cmp(target) == cmpEqual { return }
    for _, t := range merge(t.targets.value) {
        if t == target || t.cmp(target) == cmpEqual { return }
    }
    if t.targets.append(target); t.target0 != nil && t.target0.isEmpty() {
        t.target0.value = target
    }
}

func (t *traversal) getCurrentTargetValue() (res Value) {
    if t.def.target == nil {
        // top level, aka. via RuleEntry.Execute()
    } else if target := t.def.target.value; isNil(target) {
        if false { diag.errorOf(target, "target '%v' is nil", t.def.target) }
    } else if vals, err := ExpandAll(target); err != nil {
        diag.errorOf(target, "expand target '%v' failed: %v", target, err)
    } else if len(vals) == 1 { res = vals[0] } else {
        diag.errorOf(target, "target '%v' expaned to many: %v", target, res)
    }
    return
}

func (t *traversal) exists(v Value) bool {
    // FIXME: returns true if existenceMatterless ??
    return v != nil && v.stat(t).exists() == existenceConfirmed
}

func (t *traversal) depth() (res int) {
    for c := t.caller; c != nil; c = c.caller { res += 1 }
    return
}

func (t *traversal) calleeStart() {
    t.group.Add(1)
}

func (t *traversal) calleeDone(err error) {
    if err != nil {
        t.calleeErrsM.Lock()
        t.calleeErrs = append(t.calleeErrs, err)
        t.calleeErrsM.Unlock()
    }
    t.group.Done()
}

// DEPRECATED
func (t *traversal) any(i interface{}) {
    if optionEnableBenchmarks && false {  defer bench(mark(fmt.Sprintf("traversal.any(%s=%v)", typeof(i), i))) }

    var err error
    var pos = t.def.target.position
    if v := reflect.ValueOf(i); v.Kind() == reflect.Slice {
        for n := 0; err == nil && n < v.Len(); n++ {
            if optionEnableBenchmarks && false {
                i := v.Index(n).Interface()
                a, b := mark(fmt.Sprintf("%v: %s %v", n, typeof(i), i))
                t.any(i)
                bench(a, b)
            } else {
                t.any(v.Index(n).Interface())
            }
        }
    } else if i == nil {
        diag.errorAt(pos, "updating nil prerequisite")
    } else if value, ok := i.(Value); !ok {
        diag.errorAt(pos, "'%v' is invalid", value)
    } else if isNil(value) { // this could happen
        diag.errorAt(pos, "updating nil prerequisite")
    } else {
        value.traverse(t)
    }
    return
}

// DEPRECATED
func (t *traversal) filestub(p *Project, file *File, stub *filestub) (okay bool) {
    if optionEnableBenchspots { defer bench(spot("traversal.filestub")) }

    /// Searching entries from the most derived project.
    var ( entry *RuleEntry; err error )
    if entry, err = p.resolveEntry(stub.name, t.grepping); err != nil {
        diag.errorOf(stub.filemap.pattern, "resolve entry failed: %v", err)
        return
    } else if entry != nil {
        entry.traverse(t)
        return
    }

    /// Searching patterns from the most derived project.
    for _, entry := range p.resolvePatterns(stub) {
        entry.file(t, file)
        okay = !t.hasBreakers()
        break
    }
    return
}

func (t *traversal) closuredProjects() (projects []*Project) {
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

func (t *traversal) forClosuredProjects(f func(*Project) (bool, error)) (okay bool, err error) {
    var projects = t.closuredProjects()
    for _, proj := range projects {
        if okay, err = f(proj); okay || err != nil { break }
    }
    return
}

func (t *traversal) file(file *File) (okay bool) {
    if optionTraceTraversal   { t.tracef("traversal.file: %s", file) }
    if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("traversal.file(%v)", file))) }
    if optionEnableBenchspots { defer bench(spot("traversal.file")) }

    if strings.Contains(file.name, "ABIBreak.s") {
        diag.infoAt(file.position, "%v", file).
            debug(true, 1)
    }

    var (
        projects = t.closuredProjects()
        currentTargetValue = t.getCurrentTargetValue()
        concreteEntries = make(map[*RuleEntry]bool)
        concreteList []*RuleEntry
        err error
    )
    if isNil(currentTargetValue) {
        diag.errorAt(t.def.target.position, "target '%v' is nil", t.def.target).
            debug(optionDebugErrors, 1)
        return
    }
    defer func() {
        // Note that the file maybe not traversed yet at this point. But we
        // still have to check mod-time.
        var (
            a = currentTargetValue.stat(t).mod()
            b = file.stat(t).mod()
        )
        if !a.IsZero() && b.After(a) { // a.IsZero() indicates the target not exists
            t.appendUpdated(newUpdatedTarget(file))
        }

        // Add to the $^ or $| list
        t.add(file)
    } ()

    for _, project := range projects {
        var entry *RuleEntry
        if entry, err = project.resolveEntry(file.name, t.grepping); err != nil {
            diag.errorAt(file.position, "resolve entry '%v' failed: %v", file.name, err).
                debug(optionDebugErrors, 1)
            t.traceCallStack(file.position, "%v:", file.name)
            return
        } else if entry == nil {
            continue
        } else if _, ok := concreteEntries[entry]; !ok {
            concreteList = append(concreteList, entry)
            concreteEntries[entry] = true
        }
    }
    for _, entry := range concreteList {
        if entry != nil && currentTargetValue != entry {
            if entry.traverse(t); t.hasBreakers() {
                okay = t.hasBreakers(breakCase, breakDone)
                return
            }
        }
    }

    var (
        stemmedEntries = make(map[*RuleEntry]bool)
        stemmedList []*stemmed
    )
    for _, project := range projects {
        for _, ent := range project.resolvePatterns(file.name) {
            if _, ok := stemmedEntries[ent.RuleEntry]; !ok {
                stemmedList = append(stemmedList, ent)
                stemmedEntries[ent.RuleEntry] = true
            }
        }
    }
    for _, entry := range stemmedList {
        if entry.file(t, file); !t.hasBreakers() {
            if okay = file.exists(); okay { break }
        } else if brks := t.breakersOf(breakFail, breakErro); len(brks) > 0 {
            for _, brk := range brks {
                switch brk.what {
                case breakFail:
                    diag.errorAt(brk.pos, "broken traversal for stemmed file entry %v failed: %v", file, brk.message)
                case breakErro:
                    diag.errorAt(brk.pos, "broken traversal for stemmed file entry %v with error: %v", file, brk.error)
                default:
                    diag.errorAt(brk.pos, "broken traversal for stemmed file entry %v (%v)", file, brk.what)
                }
            }
            if t.breakers = t.breakersNot(breakFail, breakErro); len(t.breakers) > 0 {
                diag.errorAt(entry.position, "broken traversal for stemmed file entry %v in %v (more)",
                    entry, entry.OwnerProject()).debug(optionDebugErrors, 1)
            } else {
                diag.errorAt(entry.position, "broken traversal for stemmed file entry %v in %v", file, entry.OwnerProject()).
                    debug(optionDebugErrors, 1)
            }
            return
        } else if brks := t.breakersOf(breakCase, breakDone); len(brks) > 0 {
            if t.breakers = t.breakersNot(breakCase, breakDone); len(t.breakers) > 0 {
                diag.errorAt(entry.position, "broken traversal for stemmed file entry %v in %v", entry, entry.OwnerProject()).
                    debug(optionDebugErrors, 1)
            } else { okay = true }
            break
        } else if nxts := t.breakersOf(breakNext); len(nxts) > 0 {
            if t.breakers = t.breakersNot(breakNext); len(t.breakers) > 0 {
                for _, b := range t.breakers {
                    diag.errorAt(entry.position, "broken traversal for stemmed entry %v in %v: %v",
                        entry, entry.OwnerProject(), b.what)
                }
                diag.errorAt(entry.position, "broken traversal for stemmed entry %v in %v",
                    entry, entry.OwnerProject()).debug(optionDebugErrors, 1)
                return
            } else { continue }
        } else if t.hasBreakers() {
            for _, brk := range t.breakers {
                diag.errorAt(brk.pos, "broken traversal for stemmed file entry %v (%v)", file, brk.what)
            }
            diag.errorAt(entry.position, "broken traversal for stemmed file entry %v", entry).
                debug(optionDebugErrors, 1)
            t.traceCallStack(entry.position, "broken traversal for stemmed file %v:", file)
            return
        }
    }

    for _, project := range projects {
        if okay { break } else if file != nil {
            if okay = file.info != nil; okay { break }
            okay = file.searchInMatchedPaths(project)
        }
    }

    if err != nil {
        t.traceCallStack(file.position, "%v: file(%v): error: %v", t.project, file, err).
            debug(optionDebugErrors, 1)
    } else if !okay && len(t.stems) == 0 {
        diag.errorAt(file.position, "missing file %v", file)
        diag.errorAt(file.position, "concrete: %v", concreteList)
        diag.errorAt(file.position, "stemmed: %v", stemmedList)
        diag.errorAt(file.position, "internal stack:").
            debug(optionDebugErrors, 64)
        t.traceCallStack(file.position, "missing file %v required by %v (in %v)", file, currentTargetValue, t.project)
        t._break(file.position, breakErro).error = fileNotFoundError{ t.project, file }
    } else if !okay && len(t.stems) > 0 {
        if false { t._break(file.position, breakNext).scope = breakTrave }
    }
    return
}

func (t *traversal) string(pos Position, targetVal Value, target string) (okay bool) {
    if optionTraceTraversal   { t.tracef("traversal.target: %s", target) }
    if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("traversal.target(%v)", target))) }
    if optionEnableBenchspots { defer bench(spot("traversal.target")) }

    var (
        projects = t.closuredProjects()
        currentTargetValue = t.getCurrentTargetValue()
        concreteEntries = make(map[*RuleEntry]bool)
        concreteList []*RuleEntry
        file *File // if target is file
        err error
    )
    if isNil(currentTargetValue) {
        diag.errorAt(pos, "target '%v' is nil", t.def.target).
            debug(optionDebugErrors, 1)
        return
    }
    defer func() {
        // Note that the file maybe not traversed yet at this point. But we
        // still have to check mod-time.
        if file != nil {
            var (
                a = currentTargetValue.stat(t).mod()
                b = file.stat(t).mod()
            )
            if !a.IsZero() && b.After(a) { // a.IsZero() indicates the target not exists
                t.appendUpdated(newUpdatedTarget(file))
            }
            if !file.position.IsValid() {
                file.position = pos
            }

            // Add to the $^ or $| list
            t.add(file)
        } else if true {
            if targetVal == nil { targetVal = MakeString(pos, target) }
            t.add(targetVal)
        }
    } ()

    for _, project := range projects {
        var entry *RuleEntry
        if entry, err = project.resolveEntry(target, t.grepping); err != nil {
            diag.errorAt(pos, "resolve entry '%v' failed: %v", target, err).
                debug(optionDebugErrors, 1)
            t.traceCallStack(pos, "resolve entry '%v' failed: %v", target, err)
            return
        } else if entry == nil {
            continue
        } else if _, ok := concreteEntries[entry]; !ok {
            concreteList = append(concreteList, entry)
            concreteEntries[entry] = true
        }
    }
ForConcreteList:
    for _, entry := range concreteList {
        if project, _ := concreteEntries[entry]; entry != nil && currentTargetValue != entry {
            if w, ok := currentTargetValue.(*Bareword); ok && w.string == target {
                // target resolve to itself, does nothing
            } else if entry.traverse(t); t.hasBreakers() {
                for _, brk := range t.breakers {
                    switch brk.what {
                    case breakNext:
                        continue ForConcreteList
                    case breakFail:
                        diag.errorAt(pos, "broken traversal for concrete entry '%v' failed: %v", entry, brk.message)
                    case breakErro:
                        diag.errorAt(pos, "broken traversal for concrete entry '%v' with error: %v", entry, brk.error)
                    default:
                        diag.errorAt(pos, "broken traversal for concrete entry '%v' in %v (%v)", entry, project, brk.what)
                    }
                }
                diag.errorAt(pos, "broken traversal for concrete entry '%v' in %v", entry, project).
                    debug(optionDebugErrors, 1)
                return
            } else {
                file, _ = entry.target.(*File)
                okay = true
                return
            }
        }
    }

    for _, project := range projects {
        var obj Object
        if obj, err = project.resolveObject(target); err != nil {
            diag.errorAt(pos, "resolve object '%s' failed: %v", target, err).
                debug(optionDebugErrors, 1)
            return
        } else if isNil(obj) || isUndef(obj) || isNone(obj) {
            // does nothing here and keep trying FindFile
        } else if obj.traverse(t); t.hasBreakers() {
            diag.errorAt(pos, "broken traversal '%v' (%T) (project=%v)", obj, obj, project).
                debug(optionDebugErrors, 1)
            return
        } else if _, ok := obj.(*ProjectName); ok {
            // solved, no need to check against patterns
            return
        }
    }

    var (
        stemmedEntries = make(map[*RuleEntry]bool)
        stemmedList []*stemmed
    )
    for _, project := range projects {
        for _, ent := range project.resolvePatterns(target) {
            if _, ok := stemmedEntries[ent.RuleEntry]; !ok {
                stemmedList = append(stemmedList, ent)
                stemmedEntries[ent.RuleEntry] = true
            }
        }
    }
    for x, entry := range stemmedList {
        entry.string(t, targetVal, target)
        if strings.Contains(target, "ABIBreak") {
            for _, b := range t.breakers { diag.infoAt(pos, "%v %v", targetVal, b.what) }
            diag.infoAt(pos, "%v %v %v %v %v %v", targetVal, entry, x, len(stemmedList), len(t.breakers), entry.programs[0].depends).
                debug(true, 1)
        }
        if ; !t.hasBreakers() {
            okay = true; break // continue
        } else if brks := t.breakersOf(breakFail, breakErro); len(brks) > 0 {
            t.breakers = t.breakersNot(breakFail, breakErro);
            for _, brk := range brks {
                switch brk.what {
                case breakFail: diag.errorAt(entry.position, "traverse %v failed: %v", file, brk.message).
                    debug(optionDebugErrors, 1)
                case breakErro: diag.errorAt(entry.position, "traverse %v error: %v", file, brk.error).
                    debug(optionDebugErrors, 1)
                }
            }
            diag.errorAt(pos, "broken traversal for stemmed entry '%v' in %v", entry, entry.OwnerProject()).
                debug(optionDebugErrors, 1)
            return
        } else if brks := t.breakersOf(breakCase, breakDone); len(brks) > 0 {
            t.breakers, okay = t.breakersNot(breakCase, breakDone), true // reset breakers
            break
        } else if nxts := t.breakersOf(breakNext); len(nxts) > 0 {
            if t.breakers = t.breakersNot(breakNext); len(t.breakers) > 0 {
                diag.errorAt(entry.position, "next with broken traversal for %v (pattern=%v, next=%v)",
                    target, entry.Pattern, nxts[0].value).
                    debug(optionDebugErrors, 1)
            }
            continue
        } else {
            diag.errorAt(entry.position, "unknown breakers for target %v (%v)", target, t.breakers[0].what).
                debug(optionDebugErrors, 1)
            t.traceCallStack(entry.position, "unknown breakers for target %v (%v)", target, t.breakers[0].what)
            return
        }
    }

    for _, project := range projects {
        if file = project.FindFile(target); file != nil {
            file.position = pos // Change the position for tracing
            if false {
                var names = make(map[string]bool)
                for stub := file.filestub; true; stub = stub.other {
                    names[stub.name] = true // mark to avoid trying many times
                    if okay = t.filestub(project, file, stub); t.hasBreakers() {
                        return
                    }
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
                    var stub = &filestub{ file.dir, sub, name, file.filemap, file.filestub.other }
                    file.filestub.other = stub

                    if okay = t.filestub(project, file, stub); t.hasBreakers() { return }
                    if okay { file.filestub = stub; break }
                }
            } else if okay = t.file(file); t.hasBreakers() {
                return
            }

            // Check file existance
            if okay { break } else if file.info != nil {
                okay = true // it's good
            } else if file != nil {
                okay = file.searchInMatchedPaths(project)
            }
            if !okay && file.name != target {
                var alt = file //project.FindFile(file.name)
                if alt != nil { okay = alt.sub == "-" || alt.exists() }
            }
            if okay { return } // Done!
        }
    }

    if err != nil {
        t.traceCallStack(pos, "%v: target(%v), file=%v: error: %v", t.project, target, file, err).
            debug(optionDebugErrors,1)
    } else if !okay && !t.isConfigureExecution && len(t.stems) == 0 {
        if file != nil {
            diag.errorAt(file.position, "missing file %v", file)
            diag.errorAt(file.position, "concrete: %v", concreteList)
            diag.errorAt(file.position, "stemmed: %v", stemmedList)
            diag.errorAt(file.position, "internal stack:").
                debug(optionDebugErrors, 64)
            t.traceCallStack(file.position, "traverse missing target file '%v' for %v", file, t.project).
                debug(optionDebugErrors,1)
            t._break(file.position, breakErro).error = fileNotFoundError{t.project, file}
        } else {
            diag.errorAt(pos, "missing target %v", target)
            diag.errorAt(pos, "concrete: %v", concreteList)
            diag.errorAt(pos, "stemmed: %v", stemmedList)
            diag.errorAt(pos, "internal stack:").
                debug(optionDebugErrors, 64)
            t.traceCallStack(pos, "traverse missing target '%v' for %v", target, t.project).
                debug(optionDebugErrors,1)
            t._break(pos, breakErro).error = targetNotFoundError{t.project, target}
        }
    } else if !okay && len(t.stems) > 0 {
        t._break(pos, breakNext).scope = breakTrave
    }
    return
}

func (t *traversal) pattern(pat Value) {
    var (
        pos = pat.Position()
        rest []string
        okay bool
        s string
    )
    if s, rest = pat.stencil(t.stems); s == "" {
        diag.errorAt(pos, "empty stencil: %v %v", pat, t.stems).
            debug(optionDebugErrors, 1)
        return
    } else if len(rest) > 0 {
        diag.errorAt(pos, "partial stencil: %v, %v, %v, %v", pat, s, rest, t.stems).
            debug(optionDebugErrors, 1)
        panic(s)
    } else if file := t.project.FindFile(s); file != nil {
        file.position = pos
        okay = t.file(file)
    } else {
        okay = t.string(pos, pat, s)
    }

    if !okay && len(t.stems) > 0 {
        b := t._break(pos, breakNext)
        b.scope = breakTrave
        b.value = pat
    }
}

func (t *traversal) appendUpdated(updated *updatedtarget) {
    var currentTargetValue Value = t.getCurrentTargetValue()
    if isNil(currentTargetValue) {
        if false { diag.warnAt(t.def.target.position, "target '%v' is nil", t.def.target).
            debug(optionDebugErrors, 1) }
    } else {
        if currentTargetValue == updated.target { return }
        if currentTargetValue.cmp(updated.target) == cmpEqual { return }
    }
    for _, u := range t.updated { // check if already added
        if u.target == updated.target { return }
        if u.target.cmp(updated.target) == cmpEqual { return }
    }
    t.updated = append(t.updated, updated)
    for c := t.caller; c != nil; c = c.caller { // clear update loop
        if c.def.target != nil && c.def.target.value == updated.target { return }
    }
    if c := t.caller; c != nil {
        c.appendUpdated(newUpdatedTarget(currentTargetValue, updated))
    }
}

func (t *traversal) removeUpdated(target Value) (removed []*updatedtarget) {
    var currentTargetValue = t.getCurrentTargetValue()
    if isNil(currentTargetValue) {
        diag.errorAt(t.def.target.position, "target '%v' is nil", t.def.target)
        return
    }
    for i, u := range t.updated {
        if u.target == target || u.target.cmp(target) == cmpEqual {
            removed = append(removed, u)
            t.updated = append(t.updated[:i], t.updated[i+1:]...)
            if t.caller != nil && len(t.updated) == 0 {
                t.caller.removeUpdated(currentTargetValue)
            }
        }
    }
    return
}

func (t *traversal) removeCallerUpdated(target Value) {
    if t.caller != nil {
        for _, u := range t.caller.removeUpdated(target) {
            for _, uu := range u.prerequisites {
                t.removeUpdated(uu.target)
            }
        }
    }
}

func (t *traversal) hashDir(k []byte) string {
    dir := t.program.project.tmpPath
    h := fmt.Sprintf("%x", k[:2]) // HEX of the first two bytes
    return filepath.Join(dir, ".hash", h[0:1], h[1:2], h[2:3], h[3:])
}

func (t *traversal) cmdHash(values ...Value) (k, v HashBytes, err error) {
    var currentTargetValue = t.getCurrentTargetValue()
    if isNil(currentTargetValue) {
        diag.errorAt(t.def.target.position, "target '%v' is nil", t.def.target)
        return
    }

    var (
        key = sha256.New()
        val = sha256.New()
        str string
    )
    if str, err = fullnameOrStrval(currentTargetValue); err != nil { return }
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
        diag.errorAt(t.program.position, "hashing recipes failed: %v", err).
            debug(optionDebugErrors, 1)
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
        diag.errorAt(t.program.position, "compute recipes hash failed: %v", err).
            debug(optionDebugErrors, 1)
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

func (t *traversal) wait(pos Position, opts ...bool) (target Value, files []*File, execRes *ExecResult, err error) {
    if optionEnableBenchmarks && false { defer bench(mark("traversal.wait")) }

    // Waiting for prerequisites
    t.group.Wait()
    t.calleeErrsM.Lock()
    var errs = t.calleeErrs
    t.calleeErrs = nil
    t.calleeErrsM.Unlock()

    if target = t.getCurrentTargetValue(); isNil(target) {
        diag.errorAt(t.def.target.position, "target '%v' is nil", t.def.target).
            debug(optionDebugErrors,1)
        return
    }

    if n := len(errs); n > 0 /*&& t.stems == nil*/ {
        var (
            targetPos = t.def.target.Position()
            numRealErrs = 0
        )
        for _, err := range errs {
            diag.errorAt(pos, "%v: %v", target, err).
                debug(optionDebugErrors, 1)
            numRealErrs += 1
        }
        if numRealErrs == 0 {
            return // simply return if no real errors
        }
        if !pos.Equals(&targetPos) {
            var s string
            if n > 1 { s = "s" }
            diag.errorAt(targetPos, "%d error%s while waiting prerequisites for '%v'",
                n, s, target).debug(optionDebugErrors, 1)
        }
        var (
            v = target
            targetValuePos = v.Position()
        )
        if l, ok := v.(*List); ok && l.Len() == 1 { v = l.Elems[0] }
        if targetValuePos.IsValid() && !targetValuePos.Equals(&targetPos) {
            if f, ok := v.(*File); ok && f.filemap != nil {
                diag.errorAt(targetValuePos, "waiting for '%v'", target)
                diag.errorOf(f.filemap.pattern, "via pattern '%v' (of %v)", v, f.filemap.project).
                    debug(optionDebugErrors && target == v && t.closure == nil, 1)
            } else {
                diag.errorAt(targetValuePos, "waiting for '%v'", target).
                    debug(optionDebugErrors && target == v && t.closure == nil, 1)
            }
        }
        if def, ok := v.(*Def); ok && target != v && target != def.value {
            // trace source Def in diagnostics
            diag.errorOf(def.value, "waiting for def '%v': %v", def.name, def.value).
                debug(optionDebugErrors && t.closure == nil, 1)
        }
        if c := t.closure; c != nil {
            diag.errorAt(c.position, "waiting closured from %v", c.comment).
                debug(optionDebugErrors, 1)
        }
        if t.isConfigureExecution {
            //diag.errorOf(t., "%v: %v = %v", s, t, result)
        }
    }

    var (
        optReportFileUpdates = len(opts) > 0 && opts[0]
        waitForExecResult    = len(opts) > 1 && opts[1]
        stampCurrentTarget   = len(opts) > 2 && opts[2]
    )
    if waitForExecResult {
        // Waiting for command (shell/python/etc.) exec result
        if v, ok := t.def.buffer.value, false; v != nil {
            if execRes, ok = v.(*ExecResult); ok {
                execRes.wg.Wait()
            }
        }
    }
    if !stampCurrentTarget {
        // done!
    } else if files, err = target.stamp(t); err != nil {
        var p = target.Position()
        if !p.IsValid() { p = pos }
        diag.errorAt(pos, "%v", err).
            debug(optionDebugErrors,1)
        return
    } else if optReportFileUpdates {
        reportFileUpdates(pos, t.start, files)
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

type valbase struct { position Position }
func (_ *valbase) refs(_ Value) (res bool) { return }
func (_ *valbase) defs(s string) (res []*Def) { return }
func (_ *valbase) expandible(_ expandwhat) bool { return false }
func (_ *valbase) expand(_ expandwhat) (v Value, err error) { return }
func (_ *valbase) cmp(_ Value) (res cmpres) { return }
func (_ *valbase) patterned() bool { return false }
func (_ *valbase) match(i interface{}) (full bool, s string, stems []string) { return }
func (_ *valbase) stencil(stems []string) (s string, rest []string) { return }
func (_ *valbase) stat(t *traversal) (si *statinfo) { return }
func (_ *valbase) stamp(t *traversal) (file []*File, err error) { return }
func (t *valbase) Position() (res Position) { return t.position }
func (_ *valbase) True() (res bool, err error) { return }
func (_ *valbase) Integer() (i int64, err error) { return }
func (_ *valbase) Float() (f float64, err error) { return }
func (_ *valbase) String() (s string) { return }
func (_ *valbase) Strval() (s string, err error) { return }
func (_ *valbase) traverse(t *traversal) { }
func (_ *valbase) _match(p Value, i interface{}) (full bool, s string, stems []string) {
    var ( v string; e error )
    if v, e = p.Strval(); e == nil {
        var is string
        switch t := i.(type) {
        case string: is = t
        case Value:
            if is, e = t.Strval(); e != nil {
                diag.errorOf(t, "strval '%v' error: %v", t, e)
                return
            }
        }
        if strings.HasPrefix(is, v) {
            s, full = v, (len(v) == len(is))
        }
    } else {
        diag.errorOf(p, "strval '%v' error: %v", p, e)
    }
    return
}
func (_ *valbase) _stencil(p Value, stems []string) (s string, rest []string) {
    var ( v string; e error )
    if v, e = p.Strval(); e == nil {
        s, rest = v, stems
    } else {
        diag.errorOf(p, "strval '%v' error: %v", p, e)
    }
    return
}

type returner struct {
    valbase
    Values []Value
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
func (p *Argumented) defs(s string) (res []*Def) {
    res = p.value.defs(s)
    for _, a := range p.args {
        res = append(res, a.defs(s)...)
    }
    return
}
func (p *Argumented) expandible(w expandwhat) (res bool) {
    if res = p.value.expandible(w); !res && w&expandArgedArgs != 0 {
        for _, a := range p.args {
            if res = a.expandible(w); res { break }
        }
    }
    return
}
func (p *Argumented) expand(w expandwhat) (res Value, err error) {
    var (
        val Value
        args []Value
        num int
    )
    if val, err = p.value.expand(w); err != nil {
        diag.errorOf(p.value, "expand '%v' failed: %v", p.value, err).
            debug(optionDebugErrors, 1)
        return
    } else if isNil(val) { val = p.value }
    if w&expandArgedArgs != 0 {
        if args, num, err = expandall1(w, p.args...); err != nil {
            diag.errorOf(p.value, "expand args '%v' failed: %v", p.args, err).
                debug(optionDebugErrors, 1)
            return
        }
    }
    if val != p.value || num > 0 { res = &Argumented{ val, args }}
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
func (p *Argumented) patterned() bool { return p.value.patterned() }
func (p *Argumented) match(i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p.value.match(i)
    return
}
func (p *Argumented) stencil(stems []string) (s string, rest []string) {
    s, rest = p.value.stencil(stems)
    return
}

func (p *Argumented) stamp(t *traversal) ([]*File, error) { return p.value.stamp(t) }
func (p *Argumented) stat(t *traversal) (si *statinfo) {
    // FIXME: p.value might be not the real target (depending on the arguments)
    return p.value.stat(t)
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
func (p *Argumented) traverse(t *traversal) {
    if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
    //!< IMPORTANT! - Don't merge-expand arguments here!
    //!< Arguments should be passed to program.execute as it is.
    defer func(a []Value) { t.arguments = a } (t.arguments)
    t.arguments = p.args
    p.value.traverse(t)
}

type None struct { valbase }
func (_ *None) cmp(v Value) (res cmpres) {
    if _, ok := v.(*None); ok { res = cmpEqual }
    return
}

type Nil struct { valbase }
func (p *Nil) cmp(v Value) (res cmpres) {
    if _, ok := v.(*Nil); ok { res = cmpEqual }
    return
}

func isUndef(v Value) (t bool) { _, t = v.(*unresolvedobject); return }
func isNone(v Value) (t bool) { _, t = v.(*None); return }
func isNil(v Value) (t bool) {
    if v == nil {
        t = true
    } else if _, t = v.(*Nil); t {
        // true
    } else if vv := reflect.ValueOf(v); vv.Kind() == reflect.Ptr && vv.IsNil() {
        t = true
    }
    return
}

// Any is used to box an arbitrary value
type Any struct { value interface{} }
func (p *Any) cmp(v Value) (res cmpres) {
    switch a := v.(type) {
    case *Any:
        if p.value == a.value {
            res = cmpEqual
        } else if v1, ok := p.value.(Value); ok {
            if v2, ok := a.value.(Value); ok {
                res = v1.cmp(v2)
            }
        }
    case Value:
        if p.value == a {
            res = cmpEqual
        } else if v1, ok := p.value.(Value); ok {
            res = v1.cmp(a)
        }
    }
    return
}
func (p *Any) patterned() (res bool) {
    if p.value == nil {
        // does nothing
    } else if v, ok := p.value.(Value); ok {
       res = v.patterned()
    }
    return
}
func (p *Any) match(i interface{}) (full bool, s string, stems []string) {
    if p.value == nil {
        // does nothing
    } else if v, ok := p.value.(Value); ok {
        full, s, stems = v.match(i)
    }
    return
}
func (p *Any) stencil(stems []string) (s string, rest []string) {
    if p.value == nil {
        // does nothing
    } else if v, ok := p.value.(Value); ok {
        s, rest = v.stencil(stems)
    }
    return
}
func (p *Any) stamp(t *traversal) (files []*File, err error) {
    if a, ok := p.value.(Value); ok { files, err = a.stamp(t) }
    return
}
func (p *Any) stat(t *traversal) (si *statinfo) {
    if v, ok := p.value.(Value); ok && v != nil { si = v.stat(t) }
    return
}
func (p *Any) expand(w expandwhat) (res Value, err error) {
    if val, ok := p.value.(Value); ok && !isNil(val) {
        if res, err = val.expand(w); err != nil {
            diag.errorOf(p, "expand '%v' failed: %v", val, err).
                debug(optionDebugErrors,1)
        }
    }
    return
}
func (p *Any) refs(o Value) (res bool) {
    if v, ok := p.value.(Value); ok { res = v.refs(o) }
    return
}
func (p *Any) defs(s string) (res []*Def) {
    if v, ok := p.value.(Value); ok { res = v.defs(s) }
    return
}
func (p *Any) expandible(w expandwhat) (res bool) {
    if v, ok := p.value.(Value); ok { res = v.expandible(w) }
    return
}
func (p *Any) Position() (res Position) {
    if v, ok := p.value.(Positioner); ok { res = v.Position() }
    return
}
func (p *Any) True() (t bool, err error) {
    switch v := p.value.(type) {
    case Value:     t, err = v.True()
    case float32:   t      = math.Abs(float64(v))-0 >= FloatEpsilon
    case float64:   t      = math.Abs(v)-0 >= FloatEpsilon
    case int64:     t      = v != 0
    case int:       t      = v != 0
    case bool:      t      = v
    }
    return
}
func (p *Any) Float() (res float64, err error) {
    switch v := p.value.(type) {
    case Value:       res, err = v.Float()
    case float32:     res      = float64(v)
    case float64:     res      = v
    case int:         res      = float64(v)
    case int64:       res      = float64(v)
    case bool: if v { res      = FloatEpsilon }
    }
    return
}
func (p *Any) Integer() (res int64, err error) {
    switch v := p.value.(type) {
    case Value:       res, err = v.Integer()
    case float32:     res      = int64(v)
    case float64:     res      = int64(v)
    case int:         res      = int64(v)
    case int64:       res      = v
    case bool: if v { res      = 1 }
    }
    return
}
func (p *Any) Strval() (s string, err error) {
    switch v := p.value.(type) {
    case Value:       s, err = v.Strval()
    case float32:     s      = strconv.FormatFloat(float64(v),'g', -1, 32)
    case float64:     s      = strconv.FormatFloat(float64(v),'g', -1, 64)
    case int:         s      = strconv.FormatInt(int64(v),10)
    case int64:       s      = strconv.FormatInt(int64(v),10)
    case bool: if v { s      = "true" } else { s = "false" }
    default: s = fmt.Sprintf("%s", p.value)
    }
    return
}
func (p *Any) String() string { return fmt.Sprintf("<%v>", p.value) }
func (p *Any) traverse(t *traversal) {
    if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
    if v, ok := p.value.(Value); ok { v.traverse(t) }
}

type negative struct { valbase; x Value }
func (p *negative) refs(o Value) bool { return p.x.refs(o) }
func (p *negative) defs(s string) []*Def { return p.x.defs(s) }
func (p *negative) expandible(w expandwhat) bool { return p.x.expandible(w) }
func (p *negative) expand(w expandwhat) (res Value, err error) {
    var val Value
    if val, err = p.x.expand(w); err != nil {
        diag.errorOf(p.x, "expand '%v' failed: %v", p.x, err).
            debug(optionDebugErrors, 1)
    } else if !isNil(val) && val != p.x {
        res = &negative{p.valbase, val}
    }
    return
}
func (p *negative) cmp(v Value) (res cmpres) {
    if a, ok := v.(*negative); ok { res = p.x.cmp(a.x) }
    return
}
func (p *negative) True() (res bool, err error) {
    if p.x != nil { if res, err = p.x.True(); err == nil { res = !res }}
    return
}
func (p *negative) elemstr(o Object, k elemkind) string { return `!`+elementString(o, p.x, k) }
func (p *negative) String() (s string) { return p.elemstr(nil, 0) }
func (p *negative) Strval() (s string, err error) {
    var t bool
    if t, err = p.x.True(); err != nil {
        diag.errorAt(p.position, "truthify '%v' failed: %v", p.x, err).
            debug(optionDebugErrors, 1)
    } else {
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
func (p *negative) traverse(t *traversal) {
    if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
    if p.x != nil { p.x.traverse(t) }
}

func Negative(val Value) *negative { return &negative{valbase{val.Position()},val} }

type boolean struct { valbase; bool }
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
func (p *boolean) match(i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(p, i)
    return
}
func (p *boolean) stencil(stems []string) (s string, rest []string) {
    s, rest = p._stencil(p, stems)
    return
}

type answer struct { valbase; bool }
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
func (p *answer) match(i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(p, i)
    return
}
func (p *answer) stencil(stems []string) (s string, rest []string) {
    s, rest = p._stencil(p, stems)
    return
}

type option struct { valbase; bool }
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
func (p *option) match(i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(p, i)
    return
}
func (p *option) stencil(stems []string) (s string, rest []string) {
    s, rest = p._stencil(p, stems)
    return
}

type prediction struct {
    boolean
    reason string
}

type integer struct {
    valbase
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
func (p *Bin) String() string { return fmt.Sprintf("0b%s", strconv.FormatInt(int64(p.int64),2)) }
func (p *Bin) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),2), nil }
func (p *Bin) match(i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(p, i)
    return
}
func (p *Bin) stencil(stems []string) (s string, rest []string) {
    s, rest = p._stencil(p, stems)
    return
}

type Oct struct { integer }
func (p *Oct) String() string { return fmt.Sprintf("0%s", strconv.FormatInt(int64(p.int64),8)) }
func (p *Oct) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),8), nil }
func (p *Oct) match(i interface{}) (full bool, s string, stems []string) { return p._match(p, i) }
func (p *Oct) stencil(stems []string) (s string, rest []string) { return p._stencil(p, stems) }

type Int struct { integer }
func (p *Int) String() string { return strconv.FormatInt(int64(p.int64),10) }
func (p *Int) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),10), nil }
func (p *Int) match(i interface{}) (full bool, s string, stems []string) { return p._match(p, i) }
func (p *Int) stencil(stems []string) (s string, rest []string) { return p._stencil(p, stems) }

type Hex struct { integer }
func (p *Hex) String() string { return fmt.Sprintf("0x%s", strconv.FormatInt(int64(p.int64),16)) }
func (p *Hex) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),16), nil }
func (p *Hex) match(i interface{}) (full bool, s string, stems []string) { return p._match(p, i) }
func (p *Hex) stencil(stems []string) (s string, rest []string) { return p._stencil(p, stems) }

const FloatEpsilon = 1e-15 /* 1e-16 */
type Float struct { valbase; float64 } // IEEE-754 64-bit binary floating-point
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
func (p *Float) match(i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(p, i)
    return
}
func (p *Float) stencil(stems []string) (s string, rest []string) {
    s, rest = p._stencil(p, stems)
    return
}

type DateTime struct {
    valbase
    t time.Time
}
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
func (p *DateTime) match(i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(p, i)
    return
}
func (p *DateTime) stencil(stems []string) (s string, rest []string) {
    s, rest = p._stencil(p, stems)
    return
}

func ParseDateTime(pos Position, s string) *DateTime {
    // time.RFC3339Nano
    if t, e := time.Parse("2006-01-02T15:04:05.999999999Z07:00", s); e == nil {
        return &DateTime{valbase{pos},t}
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
func (p *Date) match(i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(p, i)
    return
}
func (p *Date) stencil(stems []string) (s string, rest []string) {
    s, rest = p._stencil(p, stems)
    return
}

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
func (p *Time) match(i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(p, i)
    return
}
func (p *Time) stencil(stems []string) (s string, rest []string) {
    s, rest = p._stencil(p, stems)
    return
}

// ie. https://en.wikipedia.org/wiki/URL
// ▶▶─<scheme>─(:)┬──────────────────────────────────────┬<path>┬───────────┬┬──────────────┬─▶◀
//        └(//)┬──────────────┬<host>┬──────────┬┘      └(?)─<query>┘└(#)─<fragment>┘
//             └<userinfo>─(@)┘      └(:)─<port>┘
type URL struct {
    valbase
    Scheme Value
    Username Value
    Password Value
    Host Value
    Port Value
    Path Value
    Query Value
    Fragment Value
}
func (p *URL) True() (t bool, e error) {
    if p.Scheme != nil { if t, e = p.Scheme.True(); t { return }}
    if p.Host   != nil { if t, e = p.Host  .True(); t { return }}
    if p.Path   != nil { if t, e = p.Path  .True(); t { return }}
    return //p.String() != "", nil
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
func (p *URL) match(i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(p, i)
    return
}
func (p *URL) stencil(stems []string) (s string, rest []string) {
    s, rest = p._stencil(p, stems)
    return
}
func (p *URL) Validate() (res *url.URL) {
    panic(fmt.Sprintf("validate %s", p))
    return
}

type Raw struct { valbase; string }
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
func (p *Raw) match(i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(p, i)
    return
}
func (p *Raw) stencil(stems []string) (s string, rest []string) {
    s, rest = p._stencil(p, stems)
    return
}

type String struct { valbase; string }
func (p *String) True() (bool, error) { return p.string != "", nil }
func (p *String) String() string { return p.elemstr(nil, 0) }
func (p *String) Strval() (string, error) { return strings.Replace(p.string, "\\\"", "\"", -1), nil }
func (p *String) Integer() (int64, error) { return strconv.ParseInt(p.string, 10, 64) }
func (p *String) Float() (float64, error) { return strconv.ParseFloat(p.string, 64) }
func (p *String) elemstr(o Object, k elemkind) (s string) {
    if k&elemNoQuote == 0 { s = `'`+p.string+`'` } else { s = p.string }
    return
}
func (p *String) traverse(t *traversal) {
    if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
    t.string(p.position, p, p.string)
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
func (p *String) match(i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(p, i)
    return
}
func (p *String) stencil(stems []string) (s string, rest []string) {
    s, rest = p._stencil(p, stems)
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

type Bareword struct { valbase; string }
func (p *Bareword) True() (bool, error) { return isTrueString(p.string), nil }
func (p *Bareword) String() string { return p.string }
func (p *Bareword) Strval() (string, error) { return p.string, nil }
func (p *Bareword) Integer() (int64, error) { return strconv.ParseInt(p.string, 10, 64) }
func (p *Bareword) Float() (float64, error) { return strconv.ParseFloat(p.string, 64) }
func (p *Bareword) traverse(t *traversal) {
    if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
    t.string(p.position, p, p.string)
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
func (p *Bareword) match(i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(p, i)
    return
}
func (p *Bareword) stencil(stems []string) (s string, rest []string) {
    s, rest = p._stencil(p, stems)
    return
}

type Qualiword struct { valbase; words []string } // foo.bar.zar, foo.&(bar).zar
func (p *Qualiword) True() (bool, error) { return len(p.words)!=0, nil }
func (p *Qualiword) String() string { return strings.Join(p.words,".") }
func (p *Qualiword) Strval() (string, error) { return p.String(), nil }
func (p *Qualiword) Integer() (int64, error) { return int64(len(p.words)), nil }
func (p *Qualiword) Float() (float64, error) { return float64(len(p.words)), nil }
func (p *Qualiword) traverse(t *traversal) {
    if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
    t.string(p.position, p, p.String())
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
func (p *Qualiword) match(i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(p, i)
    return
}
func (p *Qualiword) stencil(stems []string) (s string, rest []string) {
    s, rest = p._stencil(p, stems)
    return
}

type elements struct { Elems []Value }
func (p *elements) Len() int                { return len(p.Elems) }
func (p *elements) Append(v... Value)       { p.Elems = append(p.Elems, v...) }
func (p *elements) Get(n int) (v Value)     { if n>=0 && n<len(p.Elems) { v = p.Elems[n] }; return }
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
func (p *elements) ToBarecomp(pos Position) *Barecomp { return &Barecomp{valbase{pos},*p} }
func (p *elements) ToCompound(pos Position) *Compound { return &Compound{valbase{pos},*p} }
func (p *elements) ToList(pos Position) *List { return &List{pos, *p} }
func (p *elements) True() (t bool, err error) { // (or elems...)
    for _, elem := range p.Elems {
        if isNil(elem) {
            continue
        } else if t, err = elem.True(); err != nil {
            diag.errorOf(elem, "truthify '%v' failed: %v", elem, err).
                debug(optionDebugErrors, 1)
            break
        } else if t { break }
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
func (p *elements) defs(s string) (res []*Def) {
    for _, elem := range p.Elems {
        res = append(res, elem.defs(s)...)
    }
    return
}
func (p *elements) expandible(w expandwhat) (res bool) {
    for _, elem := range p.Elems {
        if elem.expandible(w) { return true }
    }
    return
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

type Barecomp struct { valbase ; elements }
func (p *Barecomp) Strval() (s string, e error) {
    for _, elem := range p.Elems {
        var v string
        if elem == nil { continue } else
        if v, e = elem.Strval(); e == nil { s += v } else { break }
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
func (p *Barecomp) refs(v Value) bool { return p.elements.refs(v) }
func (p *Barecomp) defs(s string) []*Def { return p.elements.defs(s) }
func (p *Barecomp) elemstr(o Object, k elemkind) (s string) {
    for _, elem := range p.Elems {
        s += elementString(o, elem, k)
    }
    return
}
func (p *Barecomp) expandible(w expandwhat) bool { return p.elements.expandible(w) }
func (p *Barecomp) expand(w expandwhat) (res Value, err error) {
    var ( elems []Value; num int )
    if elems, num, err = expandall1(w, p.Elems...); err != nil {
        diag.errorOf(p, "expand '%v' failed: %v", p, err).
            debug(optionDebugErrors, 1)
    } else if num > 0 {
        res = &Barecomp{p.valbase, elements{elems}}
    }
    return
}
func (p *Barecomp) traverse(t *traversal) {
    if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
    if target, err := p.Strval(); err == nil {
        if false { diag.warnAt(p.position, "%v (%s)", p, target).debug(true, 1) }
        t.string(p.position, p, target)
    } else {
        diag.errorOf(p, "strval '%v' error: %v", p, err)
    }
}
func (p *Barecomp) cmp(v Value) (res cmpres) {
    if a, ok := v.(*Barecomp); ok { res = p.cmpElems(a.Elems) }
    return
}
func (p *Barecomp) patterned() (res bool) {
    for _, elem := range p.Elems {
        if res = elem.patterned(); res { break }
    }
    return
}
func (p *Barecomp) match(i interface{}) (full bool, s string, stems []string) {
    if strings.Contains(p.String(), "/Volumes/workspace/external/google/tensorflow/%%") {
        diag.warnOf(p, "%v %v", p, p.patterned()).debug(true,1)
        for _, elem := range p.Elems {
            diag.warnOf(elem, "%T %v", elem, elem)
        }
    }
    full, s, stems = p._match(p, i)
    return
}
func (p *Barecomp) stencil(stems []string) (s string, rest []string) {
    s, rest = p._stencil(p, stems)
    return
}
func (p *Barecomp) Combine(x Value) {
    if o, ok := x.(*Barecomp); ok {
        for _, elem := range o.Elems {
            p.Combine(elem)
        }
    } else {
        p.Elems = append(p.Elems, x)
    }
}

// Barefile works like an alias of a File, the Strval() is identical to File.
type Barefile struct {
    valbase
    Name Value
    File *File
}
func (p *Barefile) True() (t bool, err error) {
    if p.File != nil { t, err = p.File.True() }
    return
}
func (p *Barefile) String() string { return p.elemstr(nil, 0) }
func (p *Barefile) Strval() (string, error) {
    if p.File != nil {
        return p.File.Strval()
    } else {
        return p.Name.Strval()
    }
}
func (p *Barefile) Integer() (res int64, err error) {
    if p.File.exists() { res = p.File.info.Size() }
    return
}
func (p *Barefile) Float() (float64, error) {
    i, e := p.Integer()
    return float64(i), e
}
func (p *Barefile) refs(v Value) bool { return p.Name.refs(v) }
func (p *Barefile) defs(s string) []*Def { return p.Name.defs(s) }
func (p *Barefile) elemstr(o Object, k elemkind) (s string) { return elementString(o, p.Name, k) }
func (p *Barefile) expandible(w expandwhat) bool { return p.Name.expandible(w) }
func (p *Barefile) expand(w expandwhat) (res Value, err error) {
    var name Value
    if name, err = p.Name.expand(w); err != nil {
        diag.errorOf(p.Name, "expand '%v' failed: %v", p.Name, err).
            debug(optionDebugErrors, 1)
    } else if !isNil(name) && name != p.Name {
        res = &Barefile{p.valbase, name, p.File}
    }
    return
}
func (p *Barefile) traverse(t *traversal) {
    if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
    if p.File == nil { // it happens if p.Name refers argument
        var ( target string; err error )
        if target, err = p.Strval(); err != nil {
            diag.errorOf(p, "strval '%v' failed: %v", p, err).
                debug(optionDebugErrors, 1)
            return
        }

        var okay bool
        okay, err = t.forClosuredProjects(func(project *Project) (bool, error) {
            p.File = project.FindFile(target)
            return p.File != nil, nil
        })
        if !okay || p.File == nil {
            diag.errorAt(p.position, "barefile '%s' not found", target).
                debug(optionDebugErrors, 1)
            return
        }
    }
    if p.File != nil { p.File.traverse(t) } else {
        diag.errorAt(p.position, "barefile '%s' is nil", p).
            debug(optionDebugErrors, 1)
    }
}
func (p *Barefile) stamp(t *traversal) (files []*File, err error) {
    if p.File != nil { files, err = p.File.stamp(t) }
    return
}
func (p *Barefile) stat(t *traversal) (si *statinfo) {
    if p.File != nil { si = p.File.stat(t) }
    return
}
func (p *Barefile) cmp(v Value) (res cmpres) {
    if a, ok := v.(*Barefile); ok { res = p.Name.cmp(a.Name) }
    return
}
func (p *Barefile) match(i interface{}) (full bool, s string, stems []string) {
    if p.File != nil {
        full, s, stems = p.File.match(i)
    } else {
        full, s, stems = p.Name.match(i)
    }
    return
}
func (p *Barefile) stencil(stems []string) (s string, rest []string) {
    if p.File != nil {
        s, rest = p.File.stencil(stems)
    } else {
        s, rest = p.Name.stencil(stems)
    }
    return
}

type GlobMeta struct { valbase ; token.Token }
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
type GlobRange struct { valbase ; Chars Value }
func (p *GlobRange) String() (s string) { return p.elemstr(nil, 0) }
func (p *GlobRange) Strval() (s string, err error) {
    var chars string
    if chars, err = p.Chars.Strval(); err == nil {
        s = fmt.Sprintf("[%s]", chars)
    }
    return
}
func (p *GlobRange) refs(v Value) bool { return p.Chars.refs(v) }
func (p *GlobRange) defs(s string) []*Def { return p.Chars.defs(s) }
func (p *GlobRange) expandible(w expandwhat) bool { return p.Chars.expandible(w) }
func (p *GlobRange) expand(w expandwhat) (res Value, err error) {
    var val Value
    if val, err = p.Chars.expand(w); err != nil {
        diag.errorOf(p.Chars, "expand '%v' failed: %v", p.Chars, err).
            debug(optionDebugErrors, 1)
    } else if !isNil(val) && val != p.Chars {
        res = &GlobRange{p.valbase, val}
    }
    return
}
func (p *GlobRange) elemstr(o Object, k elemkind) (s string) {
    return fmt.Sprintf("[%s]", elementString(o, p.Chars, k))
}
func (p *GlobRange) cmp(v Value) (res cmpres) {
    if a, ok := v.(*GlobRange); ok { res = p.Chars.cmp(a.Chars) }
    return
}

// Path is addressing a file (dynamically), the real located file varies
// base on 'elements' and the context.
type Path struct { valbase ; elements }
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
    for _, elem := range p.Elems {
        if t, err = elem.True(); err != nil {
            diag.errorAt(p.position, "truthify path element '%v' failed: %v", elem, err).
                debug(optionDebugErrors, 1)
        } else if t { break }
    }
    return
}
func (p *Path) refs(v Value) (res bool) { return p.elements.refs(v) }
func (p *Path) defs(s string) (res []*Def) { return p.elements.defs(s) }
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
func (p *Path) expandible(w expandwhat) bool { return p.elements.expandible(w) }
func (p *Path) expand(w expandwhat) (res Value, err error) {
    var ( elems []Value; num int )
    if elems, num, err = expandPathElems(p.position, w, p.Elems...); err != nil {
        diag.errorAt(p.position, "expand path elems failed: %v", err).
            debug(optionDebugErrors, 1)
    } else if num > 0 {
        res = &Path{p.valbase, elements{elems}}
    }
    return
}
func (p *Path) pathname(stems []string) (pathname string, err error) {// the addressed file target
    var rest []string // unmatched path segmants
    if len(stems) == 0 {
        if pathname, err = p.Strval(); err != nil {
            //diag.errorAt(p.position, "strval '%v' failed: %v", p, e)
        }
    } else if pathname, rest = p.stencil(stems); len(rest) > 0 {
        //err = errorf(p.position, "partial match: %v", rest)
    }
    return
}
func (p *Path) stamp(t *traversal) (files []*File, err error) {
    var pathname string
    if pathname, err = p.Strval(); err == nil {
        if pathname == "" {
            diag.errorOf(p, "no pathname for `%s`", p)
        } else if file := stat(p.position,pathname,"","",nil); file != nil {
            if files, err = file.stamp(t); err != nil {
                diag.errorOf(p, "stamp: %v (%v)", err, file)
            }
        }
    }
    return
}
func (p *Path) stat(t *traversal) (si *statinfo) {
    var (
        pathname string // the addressed file target
        err error
    )
    if pathname, err = p.pathname(t.stems); err != nil {
        diag.errorAt(p.position, "pathname error: %v", err)
    } else if pathname == "" {
        diag.errorAt(p.position, "pathname is empty: %v", p)
    } else if file := stat(p.position, pathname, "", "", nil); file != nil {
        si = &statinfo{ file: file }
    }
    return
}
func (p *Path) traverse(t *traversal) {
    if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }

    var (
        pathname string
        err error
    )
    if p.patterned() && len(t.stems) == 0 {
        diag.errorAt(p.position, "empty stems to traverse pattern: %v", p).
            debug(optionDebugErrors,8)
        return
    } else if pathname, err = p.pathname(t.stems); err == nil && pathname == "" {
        diag.errorAt(p.position, "path matches no target: %v", p).
            debug(optionDebugErrors,1)
        return
    } else if err != nil {
        diag.errorAt(p.position, "compute pathname failed: %v", err).
            debug(optionDebugErrors,1)
        return
    }

    // Stat the file by pathname.
    if file := stat(p.position, pathname, "", ""/*, nil*/); file != nil {
        file.traverse(t)
    } else if okay := t.string(p.position, p, pathname); !okay && len(t.stems) > 0 {
        if false { t._break(p.position, breakNext).scope = breakTrave }
    }
}
func (p *Path) patterned() (result bool) {
    for _, seg := range p.Elems {
        if result = seg.patterned(); result { break }
    }
    return
}
func (p *Path) cmp(v Value) (res cmpres) {
    if a, ok := v.(*Path); ok { res = p.cmpElems(a.Elems) }
    return
}

func expandPathElems(pos Position, w expandwhat, elems ...Value) (res []Value, num int, err error) {
    var xelems []Value
    if xelems, num, err = expandall1(w, elems...); err != nil {
        diag.errorAt(pos, "expand path elems failed: %v", err).
            debug(optionDebugErrors, 1)
        return
    }
    for _, elem := range xelems {
        if p, ok := elem.(*Path); ok {
            var ( ev []Value; n int )
            if ev, n, err = expandPathElems(pos, w, p.Elems...); err != nil {
                diag.errorOf(elem, "expand sub path '%v' failed: %v", elem, err).
                    debug(optionDebugErrors, 1)
                return
            }
            res = append(res, ev...)
            num += n
        } else {
            res = append(res, elem)
        }
    }
    if w&expandPathStr != 0 {
        var vals []Value
        for _, elem := range res {
            var s string
            switch v := elem.(type) {
            case *String:
                if v.string != "" {
                    vals = append(vals, splitPathStr(v.position, v.string)...)
                }
                num += 1
            case *Compound:
                if s, err = v.Strval(); err != nil {
                    diag.errorAt(v.position, "strval '%v' failed: %v", v, err).
                        debug(optionDebugErrors, 1)
                    return
                } else if s != "" {
                    vals = append(vals, splitPathStr(v.position, s)...)
                }
                num += 1
            default:
                vals = append(vals, elem)
            }
        }
        res = vals
    }
    return
}

func (p *Path) match1(str string) (full bool, result string, stems []string) {
    var (
        srcs []string
        segs []Value
        err error
    )
    if srcs = strings.Split(str, PathSep); len(srcs) == 0 {
        //diag.errorAt(p.position, "empty: %v", str)
        return
    }
    if segs, _, err = expandPathElems(p.position, expandPlainValue, p.Elems...); err != nil {
        diag.errorAt(p.position, "failed to expand path '%v': %v", p, segs)
        return
    }

    const info = false
    //var info = strings.Contains(p.String(), "...") && strings.Contains(str, "...")
    //var ps, _ = p.Strval()
    //var info = strings.Contains(ps, "...") || strings.Contains(str, "...")
    //var info = strings.Contains(ps, "%") && strings.Contains(str, "%")
    //var info = strings.Contains(str, "/Volumes/workspace/external/google/tensorflow/tensorflow/core/util")
    //var info = strings.Contains(p.String(), "/Volumes/workspace/external/google/tensorflow/%%util")
    //var info = false ||
    //    strings.HasPrefix(p.String(), "/Volumes/workspace/.out/x86_64-apple-darwin-extbit/bootstrap/lib/python") ||
    //    strings.HasPrefix(str, "/Volumes/workspace/.out/x86_64-apple-darwin-extbit/bootstrap/lib/python")
    //var info = strings.HasSuffix(p.String(), "/%%.py") && strings.HasSuffix(str, "__future__.py")
    var (
        lenSegs = len(segs)
        lenSrcs = len(srcs)
        lastSuf Value
        res []string
    )
SegsLoop:
    for n, m := 0, 0; n < lenSegs && m < lenSrcs; {
        var si, seg, src = n, segs[n], srcs[m]
        if cs := correctPathSegForMatch(seg); cs != nil { seg = cs } else {
            diag.errorOf(seg, "invalid path seg: %v (%T)", seg, seg).
                debug(optionDebugErrors,1)
            break SegsLoop
        }
        var (
            pp, pre, suf = percperc(seg)
            ss []string
            s    string
            f    bool
        )
        if pp {
            if info { diag.infoOf(p, "%d: path=%v seg=%v (%T) src=%v pp=%v pre=%v suf=%v res=%v stems=%v srcs[%d]=%s lenSegs=%d",
                si, p, seg, seg, src, pp, pre, suf, res, stems, m, src, lenSegs).debug(true,1) }
            if !isNil(lastSuf) && !isNil(pre) && !isNone(pre) {
                diag.errorOf(seg, "the continual %%/%% makes no sense")
                break SegsLoop
            }
            if /*ps, ok := seg.(*PathSeg);*/ src == "" /*&& ok && (ps.rune == 0 || ps.rune == '/')*/ { // for root path seg '/'
                // NOTE: seg could also be % or %% here
                res = append(res, "") // for '/'
                stems = append(stems, "")
                //diag.infoOf(p, "%v %v; %v %v; %v %v", p, seg, res, stems, ps, ok)
            }
            lastSuf = suf
            n += 1 // move forward to the next seg
            m += 1 // move forward to the next src
        } else if f, s, ss = seg.match(src); f || s == src {
            if info { diag.infoOf(p, "%d: path=%v seg=%v (%T); str=%v srcs[%d]=%v -> f=%v s=%v ss=%v => res=%v stems=%v",
                si, p, seg, seg, str, m, src, f, s, ss, res, stems).debug(true,1) }
            // NOTE: `s` could be empty string, e.g. when `str` is absolute path
            res   = append(res  , s)
            stems = append(stems, ss...)
            lastSuf = nil
            n += 1 // move forward to the next seg
            m += 1 // move forward to the next src
            if f { continue SegsLoop } else { break SegsLoop }
        } else {
            if ps, ok := seg.(*PathSeg); (s == "" && ok && ps.rune == 0) || s != "" {
                res = append(res, s)
            } else {
                res, stems = nil, nil
            }
            if info { diag.infoOf(p, "%d: path=%v seg=%v (%T) res=%v stems=%v f=%v s=%s ss=%v str=%s src=%s lenSegs=%d",
                si, p, seg, seg, res, stems, f, s, ss, str, src, lenSegs).debug(true,1) }
            if str == "%%.py" {
                diag.warnOf(p, "path=%v, str=%v, res=%v, stems=%v", p, str, res, stems).
                    debug(true)
            }
            break SegsLoop
        }

        var prefix string
        if !(isNil(pre) || isNone(pre)) {
            if prefix, err = pre.Strval(); err != nil {
                diag.errorOf(pre, "strval prefix '%v' failed: %v", pre, err)
                return
            } else if !strings.HasPrefix(src, prefix) {
                if info { diag.infoOf(p, "%d: seg=%v (%T) pp=%v pre=%v suf=%v res=%v stems=%v src=%s",
                    si, seg, seg, pp, pre, suf, res, stems, src).debug(true,1) }
                break SegsLoop
            }
        }

        // Iterate segs for %%, e.g. bar, baz in foo/%%/bar/baz
        var stem []string
        if prefix != "" { stem = append(stem, strings.TrimPrefix(src, prefix)) }
        if !(isNil(suf) || isNone(suf)) {
            var suffix string
            if suffix, err = suf.Strval(); err != nil {
                diag.errorOf(suf, "strval suffix '%v' failed: %v", suf, err)
                break SegsLoop
            }
            res = append(res, src)
            if m < lenSrcs {
                stem = append(stem, src)
                for ; m < lenSrcs; m += 1 {
                    src = srcs[m]
                    res = append(res, src)
                    if strings.HasSuffix(src, suffix) {
                        stem = append(stem, strings.TrimSuffix(src, suffix))
                        stems = append(stems, strings.Join(stem, PathSep))
                        if info { diag.infoOf(p, "%d: path=%v seg=%v (%T) res=%v stems=%v suffix=%v src=%s lenSegs=%d",
                            si, p, seg, seg, res, stems, suffix, src, lenSegs).debug(true,1) }
                        n += 1 // continue for next seg
                        continue SegsLoop
                    } else {
                        stem = append(stem, src)
                    }
                }
            } else {
                stem = append(stem, strings.TrimSuffix(src, suffix))
            }
            if len(stem) > 0 {
                stems = append(stems, strings.Join(stem, PathSep))
                if info { diag.infoOf(p, "%d: path=%v seg=%v (%T) res=%v stems=%v suffix=%v src=%s lenSegs=%d lenSrcs=%d m=%d",
                    si, p, seg, seg, res, stems, suffix, src, lenSegs, lenSrcs, m).debug(true,1) }
            }
        } else if n < lenSegs && !(isNil(segs[n]) || isNone(segs[n])) {
            for seg = segs[n]; m < lenSrcs; m += 1 {
                src = srcs[m]
                if matched, s, ss := seg.match(src); matched || s == src {
                    res = append(res, s)
                    if s == "" && len(ss) == 0 {
                        stem = append(stem, s)
                    } else if false {
                        stem = append(stem, ss...)
                    }
                    if si == 0 && len(stems) == 1 && stems[0] == "" { // heading %% matched root '/'
                        stem = append(stems, stem...) // for the root '/'
                        stems[0] = strings.Join(stem, PathSep)
                    } else {
                        stems = append(stems, strings.Join(stem, PathSep))
                    }
                    if info { diag.infoOf(p, "%d: path=%v seg=%v (%T) res=%v stems=%v matched=%v s=%s ss=%v src=%s lenSegs=%d",
                        si, p, seg, seg, res, stems, matched, s, ss, src, lenSegs).debug(true,1) }
                    n += 1 // continue for next seg
                    continue SegsLoop
                } else {
                    res = append(res, src)
                    stem = append(stem, src)
                }
            }
            if len(stem) > 0 {
                stems = append(stems, strings.Join(stem, PathSep))
                if info { diag.infoOf(p, "%d: path=%v seg=%v (%T) res=%v stems=%v s=%s ss=%v src=%s lenSegs=%d",
                    si, p, seg, seg, res, stems, s, ss, src, lenSegs).debug(true,1) }
            }
        } else { // the tailing %%
            if /*m < lenSrcs*/true {
                var rest = srcs[m-1:]
                res = append(res, rest...)
                stem = append(stem, rest...)
                stems = append(stems, strings.Join(stem, PathSep))
            }
            if info { diag.infoOf(p, "%d: seg=%v pre=%v suf=%v res=%v stems=%v src=%s",
                si, seg, pre, suf, res, stems, src).debug(true,1) }
            break SegsLoop
        }
        if info && n == lenSegs {
            diag.infoOf(p, "%d: path=%v seg=%v (%T) str=%v src=%v -> f=%v s=%v ss=%v -> res=%v stems=%v stem=%v m=%d/%d 3.lengSegs=%d",
                si, p, seg, seg, str, src, f, s, ss, res, stems, stem, m, lenSrcs, lenSegs).debug(true, 1)
        }
    }
    if lenRes := len(res); lenRes > 0 { // full or partial matched
        result = strings.Join(res, PathSep) // NOTE: don NOT use `filepath.Join(res...)` here
        full = lenRes == lenSrcs && result == str
        if info { if false {
            diag.warnOf(p, "Path.match: path=%v str=%v res=%v stems=%v -> full=%v result=%v lens=%d,%d",
                p, str, res, stems, full, result, lenRes, lenSrcs).debug(true, 1)
        } else {
            diag.warnOf(p, "Path.match: path=%v res=%v stems=%v lenRes=%d", p, res, stems, lenRes)
            diag.warnOf(p, "Path.match: str=%v full=%v result=%v lenSrcs=%d", str, full, result, lenSrcs).
                debug(true, 1)
        }}
        if correct := (!full && strings.HasPrefix(str, result)) || (full && str == result); false {
            assert(correct, "incorrect result: res=%v result=%v full=%v stems=%v str=%s",
                res, result, full, stems, str)
        } else if !correct {
            diag.errorAt(p.position, "incorrect result: str=%s res=%v stems=%v full=%v result=%v",
                str, res, stems, full, result).debug(true,1)
        }
        if p.patterned() && full && len(stems) == 0 {
            diag.errorAt(p.position, "incorrect result: path=%v, str=%s, res=%v result=%v",
                p, str, res, result).debug(true,1)
        }
    }
    return
}
func (p *Path) match(i interface{}) (full bool, result string, stems []string) {
    switch t := i.(type) {
    case  string  : return p.match1(t)
    case *filestub:
        if full, result, stems = p.match1(t.name); full || result != "" {
            return // NOTE: done if path fully or partially matched
        } else if t.dir != "" || t.sub != "" {
            return p.match1(filepath.Join(t.dir, t.sub, t.name))  // NOTE: matching the fullname form
        }
    case *File:
        if false {
            for stub := t.filestub; true; stub = stub.other {
                if full, result, stems = p.match1(stub.name); full || result != "" {
                    return
                } else if stub.other == t.filestub { break }
            }
            var s = t.name
            if t.sub != "" {
                s = filepath.Join(t.sub, t.name)
                if full, result, stems = p.match1(s); full || result != "" {
                    return
                }
            }
            if t.dir != "" {
                s = filepath.Join(t.dir, s)
                if full, result, stems = p.match1(s); full || result != "" {
                    return
                }
            }
        } else if full, result, stems = p.match1(t.name); full || result != "" {
            return // NOTE: done if path fully or partially matched
        } else if t.dir != "" || t.sub != "" {
            return p.match1(t.fullname()) // NOTE: matching the fullname form
        }
    case Value :
        if str, err := t.Strval(); err != nil {
            diag.errorOf(t, "strval '%v' failed: %v", t, err).
                debug(optionDebugErrors, 1)
        } else if str != "" {
            return p.match1(str)
        }
    default:
        diag.errorAt(p.position, "matching unsupport value: %T %v", i, i).
            debug(optionDebugErrors, 8)
    }
    return
}

func (p *Path) stencil(stems []string) (result string, rest []string) {
    var (
        strs []string
        segs []Value
        err error
    )
    if segs, err = ExpandAll(p.Elems...); err != nil {
        diag.errorOf(p, "expand path '%v' failed: %v", p, err)
        return
    }

    ts := stems

ForPathSegs:
    for _, seg := range segs {
        var s string
        if s, stems = seg.stencil(stems); s != "" {
            strs = append(strs, s)
            continue ForPathSegs
        } else if s, err = seg.Strval(); err != nil {
            diag.errorOf(seg, "strval seg '%v' failed: %v", seg, err)
            break ForPathSegs
        } else {
            strs = append(strs, s)
        }
    }
    result = strings.Join(strs, PathSep)
    rest = stems // the rest stems
    if false && strings.Contains(p.String(), "$(srcdir)/%%.py") {
        diag.warnOf(p, "%v %v %v", p, ts, result).debug(true,1)
    }
    return
}

func (p *Path) Combine(val Value) {
    var (
        ti = len(p.Elems)-1
        tail = p.Elems[ti]
        comp *Barecomp
    )
    if isNil(tail) {
        diag.errorOf(p, "path tail is nil: %v", p)
        return
    } else if seg, ok := tail.(*PathSeg); ok {
        switch comp = MakeBarecomp(tail.Position()); seg.rune {
        case 0, '/': break // discard
        default: comp.Combine(tail)
        }
        p.Elems[ti] = comp
    } else if comp, ok = tail.(*Barecomp); !ok || comp == nil {
        comp = MakeBarecomp(tail.Position(), tail)
        p.Elems[ti] = comp
    } else {
        diag.errorAt(p.position, "FIXME: join %v (%T), %v (%T)", tail, tail, val, val).
            debug(optionDebugErrors, 1)
        return
    }

    if vp, ok := val.(*Path); ok {
        var head = vp.Elems[0]
        if seg, ok := head.(*PathSeg); ok {
            switch seg.rune {
            case 0, '/': break // discard
            default: comp.Combine(head)
            }
        } else { comp.Combine(head) }
        p.Elems = append(p.Elems, vp.Elems[1:]...)
    } else { comp.Combine(val) }

    if false { diag.warnAt(p.position, "%v  %v", p, val).debug(true, 1) }
}

type PathSeg struct { valbase; rune }
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
    case 0  : s = "" // empty segment after the last '/', e.g. /foo/bar/
    default : e = fmt.Errorf("unknown pathseg (%s)", p.rune)
    }
    return
}
func (p *PathSeg) cmp(v Value) (res cmpres) {
    if a, ok := v.(*PathSeg); ok && p.rune == a.rune { res = cmpEqual }
    return
}
func (p *PathSeg) match(i interface{}) (full bool, result string, stems []string) {
    var s string
    switch t := i.(type) {
    case string: s = t
    case Value:
        var e error
        if s, e = t.Strval(); e != nil {
            diag.errorAt(p.position, "strval '%v' failed: %v", t, e)
            return
        }
    }
    switch p.rune {
    case '/': if s == ""   { result, full = s, true }
    case '~': if s == "~"  { result, full = s, true }
    case '.': if s == "."  { result, full = s, true }
    case '^': if s == ".." { result, full = s, true }
    case 0  : if s == ""   { result, full = s, true }
    }
    return
}
func (p *PathSeg) stencil(stems []string) (result string, rest []string) {
    var e error
    if result, e = p.Strval(); e != nil {
        diag.errorAt(p.position, "strval '%v' failed: %v", p, e)
    }
    return
}

type filestub struct {
    dir  string      // full directory where the file was or should be found
    sub  string      // matched sub path (see Project.search), may be Dir (absoletep path)
    name string      // constant represented name (e.g. relative filename)
    filemap *FileMap // matched pattern (see 'files' directive)
    other *filestub  // pointed to another stub (in a different project) of the same file
}

type filebase struct {
    stub filestub    // cycled-list of file stubs of different projects
    info os.FileInfo // file info if exists
    //updated bool // true if this file has been updated by a program
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
func (p *filebase) exists() (res bool) { return p.info != nil }

func stat(pos Position, name, sub, dir string, infos ...os.FileInfo) (file *File) {
    var ( base *filebase ; stub *filestub ; fullname string )

    statmutex.Lock(); defer statmutex.Unlock()

    // Trims / suffix
    if dir != "" { dir = filepath.Clean(dir) }
    if sub != "" { sub = filepath.Clean(sub) }
    if false {
        t := strings.HasPrefix(name, "./")
        if name!= "" { name = filepath.Clean(name) }
        if t { name = "./" + name }
    }

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
                if options.debug || true { debug.PrintStack() }
                diag.errorAt(pos, "dir name conflicts: %s <-> %s (sub=%v)", dir, name, sub)
                unreachable("path error")
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
        dir = filepath.Join(context.workdir, dir)
        fullname = filepath.Join(dir, sub, name)
    }

    if false { fullname = filepath.Clean(fullname) }
    if enable_assertions {
        assert(filepath.IsAbs(fullname), "`%s` is not abs {%s %s %s}", fullname, name, sub, dir)
        if sub != "-" && filepath.IsAbs(name) {
          assert(dir == "", "`%s` invalid file (dir=%s, sub=%s)", fullname, dir, sub)
          assert(sub == "", "`%s` invalid file (dir=%s, sub=%s)", fullname, dir, sub)
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
        unreachable("too many input file infos")
    }

    var okay bool
    if base, okay = filecache[fullname]; okay {
        if base.info == nil {
            if info == nil { info, _ = os.Stat(fullname) }
            if info == nil && !addNotExisted { return nil } // file not exists
            base.info = info
        }

        var head = &base.stub
        if enable_assertions {
            for stub = head; stub != nil ; stub = stub.other {
                s := filepath.Join(stub.dir, stub.sub, stub.name)
                assert(fullname == s, "(%s %s %s) fullname conflicted: %s (%s %s %s)",
                    stub.dir, stub.sub, stub.name, fullname, dir, sub, name)
                if stub.other == head { break }
            }
        }
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

        base = &filebase{ filestub{ dir, sub, name, nil, nil }, info/*, false*/ }
        base.stub.other = &base.stub
        stub = &base.stub
        filecache[fullname] = base
    }
GotFile:
    file = &File{valbase{pos},base,stub} // FIXME: needs position information
    if enable_assertions {
        if !addNotExisted { assert(file.exists(), "`%s` file not existed", fullname) }
        assert(file.name == name, "(%s %s %s).name != %s", file.name, file.sub, file.dir, name)
        assert(file.sub == sub, "(%s %s %s).sub != %s", file.name, file.sub, file.dir, sub)
        if file.dir != dir {
            var head = &base.stub
            for stub := head; stub != nil; stub = stub.other {
                diag.infoAt(pos, "stat: %s %s %s", stub.dir, stub.sub, stub.name).debug(true,1)
                if stub.other == head { break }
            }
        }
        assert(file.dir == dir, "{%s %s %s} dir incorrect: %s", file.dir, file.sub, file.name, dir)
        //assert(file.dir != "", "(%s %s %s) empty dir", file.name, file.sub, file.dir)
        if file.exists() {
            if s := filepath.Join(file.dir, file.sub, file.name); false {
                assert(file.info != nil, "{%s %s %s} file info is nil", file.dir, file.sub, file.name)
                assert(file.info.Name() == filepath.Base(file.name), "{dir=%s sub=%s name=%s} name incorrect: %s",
                    file.dir, file.sub, file.name, file.info.Name())
                assert(file.fullname() == s, "{dir=%s sub=%s name=%s} fullname incorrect: %s",
                    file.dir, file.sub, file.name, s)
            } else {
                if file.info == nil {
                    diag.errorAt(pos, "{%s %s %s} file info is nil", file.dir, file.sub, file.name).
                        debug(true,24)
                }
                if file.fullname() != s {
                    diag.errorAt(pos, "{dir=%s sub=%s name=%s} fullname incorrect: %s", file.dir, file.sub, file.name, s).
                        debug(true,24)
                }
                if file.info.Name() != filepath.Base(s) { // NOTE: file.name might be "" here
                    diag.errorAt(pos, "{dir=%s sub=%s name=%s} name incorrect: %s",
                        file.dir, file.sub, file.name, file.info.Name()).
                        debug(true,24)
                }
            }
        }
    }
    return
}

type File struct {
    valbase
    *filebase
    *filestub
}
func (p *File) True() (t bool, err error) {
    if p.name != "" {
        t = true // p.exists() == existenceConfirmed
    }
    return
}
func (p *File) String() string { return p.name }
func (p *File) Strval() (s string, err error) {
    if false { diag.warnOf(p, "use file.fullname() instead").debug(true, 8) }
    s = p.name; return }
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
    if p.filemap != nil {
        var pre string
        // FIXME: File should keep both 'match' and 'pre',
        // or just remove searchInMatchedPaths
        f := p.filemap.stat(proj.absPath, pre, p.name)
        if f.info != nil { p.info, res = f.info, true }
    }
    return
}
func (p *File) stamp(t *traversal) (files []*File, err error) {
    var fullname string
    if fullname = p.fullname(); fullname == "" {
        diag.errorOf(p, "file `%s` has no fullname", p).
            debug(optionDebugErrors, 1)
        return
    }

    var currentTargetValue = t.getCurrentTargetValue()
    if isNil(currentTargetValue) {
        diag.errorAt(t.def.target.position, "target '%v' is nil", t.def.target).
            debug(optionDebugErrors, 1)
        return
    }

    //var oldModTime time.Time
    //if p.info != nil { oldModTime = p.info.ModTime() }
    if p.info, err = os.Stat(fullname); err != nil {
        if false { diag.errorAt(p.position, "%v", err).
            debug(optionDebugErrors, 1) }
    } else if p.info != nil {
        var newModTime = p.info.ModTime()
        context.globe.stamp(fullname, newModTime)
        // p.updated = newModTime.After(oldModTime)
        files = append(files, p)

        var target = currentTargetValue
        var cmp = target.cmp(p)
        if cmp == cmpEqual && t.caller != nil {
            // Add to caller context
            t.caller.appendUpdated(newUpdatedTarget(p))
            target = t.caller.def.target.value
        } else {
            t.appendUpdated(newUpdatedTarget(p))
        }
    }
    return
}
func (p *File) exists() (res bool) {
    if p != nil && p.filebase != nil {
        res = p.filebase.exists()
    }
    return
}
func (p *File) stat(t *traversal) (si *statinfo) {
    var err error
    if p.info == nil {
        if p.info, err = os.Stat(p.fullname()); err == nil {
            // good
        } else if pe, ok := err.(*fs.PathError); ok {
            if false {
                diag.errorAt(p.position, "File.stat %v: %v", trimPromptString(pe.Path), pe.Err).
                    debug(optionDebugErrors, 1)
            }
            return
        } else {
            diag.errorAt(p.position, "File.stat failed: %v", err).
                debug(optionDebugErrors, 1)
        }
    }
    if err == nil { si = &statinfo{ file: p } }
    return
}
func (p *File) isSysFile() (res bool) {
    if p.filemap != nil && len(p.filemap.Paths) == 1 {
        // system files defined by:
        //     files (
        //     (foo.xxx) ⇒ -
        //     )
        if f, ok := p.filemap.Paths[0].(*Flag); ok {
            res = isNone(f.name) || isNil(f.name)
            //fmt.Fprintf(stderr, "sys: %v %v %v\n", p, res, p.match)
        }
    }
    return
}
func (p *File) traverse(t *traversal) {
    if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
    if optionTraceExec { defer un(trace(t_exec, fmt.Sprintf("File %v", p))) }
    if p.isSysFile() { if optionTraceTraversal { t.tracef("SysFile: true") }
        return
    }

    var currentTargetValue = t.getCurrentTargetValue()
    if isNil(currentTargetValue) {
        diag.errorAt(p.position, "target '%v' is nil", t.def.target).
            debug(optionDebugErrors, 1)
        return
    }

    // FIXES: checks none-File file target
    switch a := currentTargetValue.(type) {
    case *Barecomp: // convert barecomp path into a real Path
        var v = a.Elems[0]
        if p, ok := v.(*Path); ok {
            a.Elems = append(p.Elems[len(p.Elems)-1:], a.Elems[1:]...)
            p.Elems[len(p.Elems)-1] = a
            currentTargetValue = p
            t.def.target.value = p
            if optionTraceTraversal { t.tracef("FIX: barecomp path: %v", p) }
        } else {
            var s string
            var err error
            if s, err = a.Strval(); err != nil {
                diag.errorOf(a, "strval '%v' failed: %v", a, err).
                    debug(optionDebugErrors, 1)
                return
            }
            if file := t.project.FindFile(s); file != nil {
                currentTargetValue = file
                t.def.target.value = file
                if optionTraceTraversal { t.tracef("FIX: barecomp file: %v", p) }
            }
        }
    }

    if t.file(p); t.hasBreakers() {
        diag.errorAt(p.position, "broken traversal for file '%v' (at %s)", p, p.fullname()).
            debug(optionDebugErrors, 1)
    } else if p.info == nil {
        t._break(p.position, breakErro).error = fileNotFoundError{ t.project, p }
        diag.errorAt(p.position, "break: missing file %v (at %s)", p, p.fullname())
        diag.errorAt(t.project.position, "from project %v (for %v)", t.project, p).
            debug(optionDebugErrors, 1)
    }
}

func (p *File) cmp(v Value) (res cmpres) {
    if isNil(v) || isNone(v) {
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

func (p *File) patterned() bool { return false }
func (p *File) match1(v string) (full bool, s string, stems []string) {
    if name := p.name; name == v {
        s, full = name, true
    } else if name = filepath.Join(p.sub, p.name); name == v {
        s, full = name, true
    }
    return
}
func (p *File) match(i interface{}) (full bool, s string, stems []string) {
    switch t := i.(type) {
    case string: return p.match1(t)
    case Value:
        if !(isNil(t) || isNone(t)) {
            var ( v string; e error )
            if v, e = t.Strval(); e != nil {
                diag.errorOf(t, "strval '%v' failed: %v", t, e).
                    debug(optionDebugErrors,1)
            } else { return p.match1(v) }
        }
    default:
        diag.errorAt(p.position, "matching file '%v' with unknown input: %v", p, i).
            debug(optionDebugErrors, 1)
    }
    return
}
func (p *File) stencil(stems []string) (s string, rest []string) {
    s = p.name
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

type Flag struct { valbase ; name Value }
func (p *Flag) True() (t bool, err error) { return p.name.True() }
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
func (p *Flag) refs(v Value) bool { return p.name.refs(v) }
func (p *Flag) defs(s string) []*Def { return p.name.defs(s) }
func (p *Flag) elemstr(o Object, k elemkind) (s string) { return "-" + elementString(o, p.name, k) }
func (p *Flag) match(i interface{}) (full bool, s string, stems []string) {
    switch t := i.(type) {
    case *None, *Nil, *unresolvedobject:
    case *Flag:
        full, s, stems = p.name.match(t.name)
        s = "-" + s
    case Value:
        if v, e := t.Strval(); e != nil {
            diag.errorOf(t, "strval '%v' failed: %v", t, e).
                debug(optionDebugErrors, 1)
        } else if strings.HasPrefix(v, "-") {
            full, s, stems = p.name.match(v[1:])
            s = "-" + s
        }
    case string:
        if strings.HasPrefix(t, "-") {
            full, s, stems = p.name.match(t[1:])
            s = "-" + s
        }
    default:
        diag.warnAt(p.position, "-%v <-> %T %v", p.name, i, i).
            debug(true, 16)
    }
    return
}
func (p *Flag) expandible(w expandwhat) bool { return p.name.expandible(w) }
func (p *Flag) expand(w expandwhat) (res Value, err error) {
    var name Value
    if name, err = p.name.expand(w); err != nil {
        diag.errorOf(p.name, "expand '%v' failed: %v", p.name, err).
            debug(optionDebugErrors, 1)
    } else if !isNil(name) && name != p.name {
        res = &Flag{p.valbase, name}
    }
    return
}
func (p *Flag) opt(short, long string) (res string, match bool) {
        if isNil(p.name) {
                diag.errorOf(p, "flag name is nil")
        } else if f, ok := p.name.(*Flag); ok {
                res, match = f.opt(short, long)
        } else if s, err := p.name.Strval(); err != nil {
                diag.errorOf(p.name, "strval '%v' failed: %v", p.name, err)
        } else if s == short {
                res, match = short, true
        } else if s == long {
                res, match = long, true
        }
        return
}
// DEPRECATED
func (p *Flag) opts(try bool, opts ...string) (runes []rune, names []string, err error) {
    switch t := p.name.(type) {
    case *Flag:
        runes, names, err = t.opts(try, opts...)
    case *String:
        for _, opt := range opts {
            if t.string == opt { names = append(names, opt) }
        }
        if !try && len(names) == 0 {
            diag.errorOf(p, "unknown flag (known: %s)", strings.Join(opts, ", "))
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
            diag.errorOf(p, "unknown flag (known: %s)", strings.Join(opts, ", "))
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
func (p *Flag) traverse(t *traversal) {
    if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
    if s, err := p.Strval(); err != nil {
        diag.errorOf(p, "strval '%v' failed: %v", p, err).
            debug(optionDebugErrors, 1)
    } else {
        t.string(p.position, p, s)
    }
}

const escapedChars = "\"\r\n"

type Compound struct { valbase ; elements } // "compound string"
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
func (p *Compound) defs(s string) []*Def { return p.elements.defs(s) }
func (p *Compound) expandible(w expandwhat) bool { return p.elements.expandible(w) }
func (p *Compound) expand(w expandwhat) (res Value, err error) {
    var ( elems []Value; num int )
    if elems, num, err = expandall1(w, p.Elems...); err != nil {
        diag.errorAt(p.position, "expand compound elems failed: %v", err).
            debug(optionDebugErrors, 1)
    } else if num > 0 {
        res = &Compound{p.valbase, elements{elems}}
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
            diag.errorOf(p, "%v", err)
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
            diag.errorOf(p, "%v", err)
			return
        }
        i = strings.IndexAny(s, escapedChars)
    }
    if _, err = buf.WriteString(s); err != nil {
        diag.errorOf(p, "%v", err)
    }
    return
}
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

type List struct {
        position Position
        elements
}
func (p *List) Position() (pos Position) {
        /*if len(p.Elems) > 0 {
                pos = p.Elems[0].Position()
        }*/
        return p.position
}
func (p *List) Float() (float64, error) {
    i, e := p.Integer()
    return float64(i), e
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
func (p *List) elemstr(o Object, k elemkind) (s string) {
    var strs []string
    for _, elem := range p.Elems {
        strs = append(strs, elementString(o, elem, k))
    }
    return strings.Join(strs, " ")
}
func (p *List) expand(w expandwhat) (res Value, err error) {
    var ( elems []Value; num int )
    if elems, num, err = expandall1(w, p.Elems...); err != nil {
        diag.errorAt(p.position, "expand list elems failed: %v", err).
            debug(optionDebugErrors, 1)
    } else if num > 0 {
        res = &List{p.position, elements{elems}}
    }
    return
}
func (p *List) traverse(t *traversal) {
    if len(p.Elems) == 0 { return }
    if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
    for _, v := range p.Elems {
        if v.traverse(t); t.hasBreakers() { break }
    }
    return
}
func (p *List) stat(t *traversal) (si *statinfo) {
    if len(p.Elems) > 0 {
        for _, elem := range p.Elems {
            if ei := elem.stat(t); ei == nil {
                // FIXME: insert new statinfo or just discard it ??
            } else if si == nil {
                si = ei
            } else {
                si.next = ei
            }
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

func (p *List) cmp(v Value) (res cmpres) {
    if a, ok := v.(*List); ok { res = p.cmpElems(a.Elems) }
    return
}

func (p *List) patterned() bool {
    // TODO: apply to each element:
    /*for _, elem := range p.Elems {
        if elem.patterned() { return true }
    }*/
    return false
}

func (p *List) match(i interface{}) (full bool, s string, stems []string) {
    // TODO: match each element
    return
}

func (p *List) stencil(stems []string) (s string, rest []string) {
    // TODO: stencil each element
    return
}

type Group struct { valbase ; elements }
func (p *Group) Position() Position { return p.valbase.Position() }
func (p *Group) Float() (float64, error) { return p.valbase.Float() }
func (p *Group) Integer() (int64, error) { return p.valbase.Integer() }
func (p *Group) True() (t bool, err error) {
    t = len(p.Elems) > 0
    return
}
func (p *Group) String() string { return p.elemstr(nil, 0) }
func (p *Group) Strval() (s string, err error) {
    if s, err = p.Strval(); err == nil {
        s = "(" + s + ")"
    }
    return
}
func (p *Group) refs(v Value) bool { return p.elements.refs(v) }
func (p *Group) defs(s string) []*Def { return p.elements.defs(s) }
func (p *Group) stat(t *traversal) (si *statinfo) { return }
func (p *Group) stamp(t *traversal) (files []*File, err error) { return }
func (p *Group) elemstr(o Object, k elemkind) string {
    var strs []string
    for _, elem := range p.Elems {
        strs = append(strs, elementString(o, elem, k))
    }
    return fmt.Sprintf("(%s)", strings.Join(strs, " "))
}
func (p *Group) expandible(w expandwhat) bool { return p.elements.expandible(w) }
func (p *Group) expand(w expandwhat) (res Value, err error) {
    var ( elems []Value; num int )
    if elems, num, err = expandall1(w, p.Elems...); err != nil {
        diag.errorAt(p.position, "expand group elems failed: %v", err).
            debug(optionDebugErrors, 1)
    } else if num > 0 {
        res = &Group{p.valbase, elements{elems}}
    }
    return
}
func (p *Group) traverse(t *traversal) {
    diag.warnAt(p.position, "traversing group: %v", p).
        debug(optionDebugErrors, 32)
}
func (p *Group) cmp(v Value) (res cmpres) {
    if a, ok := v.(*Group); ok { res = p.cmpElems(a.Elems) }
    return
}
func (p *Group) patterned() bool { return false }
func (p *Group) match(i interface{}) (full bool, s string, stems []string) { return }
func (p *Group) stencil(stems []string) (s string, rest []string) { return }

func parseGroupValue(g *Group) (result Value) {
    if len(g.Elems) == 0 { return g } else {
        var word *Bareword
        switch kind := g.Elems[0].(type) {
        case *Bareword: word = kind
        case *Group: if len(kind.Elems) > 0 {
            var ( name = kind.Elems[0]; ok bool )
            if word, ok = name.(*Bareword); !ok {
                diag.errorOf(name, "unsupported name type: %T %v", name, name).
                    debug(optionDebugErrors, 1)
            }
        }}
        if word != nil {
            switch word.string {
            case "plain", "json", "yaml", "xml":
                result = MakeList(g.Elems[1].Position(), g.Elems[1:]...)
            }
        }
        if isNil(result) { result = g }
    }
    return
}

type Pair struct { // key=value
    valbase
    Key Value
    Value Value
}
func (p *Pair) True() (t bool, err error) {
    if t, err = p.Key.True(); err != nil {
        diag.errorOf(p.Key, "truthify '%v' failed: %v", p.Key, err).
            debug(optionDebugErrors, 1)
    } else if t || isNil(p.Value) {
        // done
    } else if t, err = p.Value.True(); err != nil {
        diag.errorOf(p.Key, "truthify '%v' failed: %v", p.Value, err).
            debug(optionDebugErrors, 1)
    }
    return
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
func (p *Pair) refs(v Value) bool { return p.Key.refs(v) || p.Value.refs(v) }
func (p *Pair) defs(s string) []*Def { return append(p.Key.defs(s), p.Value.defs(s)...) }
func (p *Pair) elemstr(o Object, k elemkind) string {
    return elementString(o, p.Key, k)+`=`+elementString(o, p.Value, k)
}
func (p *Pair) traverse(t *traversal) {
    diag.errorAt(p.position, "traversing pair '%v' is undefined", p).
        debug(optionDebugErrors, 16)
    t.traceCallStack(p.position, "pair is not traversible: %v", p)
}
func (p *Pair) expandible(w expandwhat) bool {
    if p.Key.expandible(w) { return true }
    return w&expandPairVal != 0 && p.Value.expandible(w)
}
func (p *Pair) expand(w expandwhat) (res Value, err error) {
    var k, v Value
    if k, err = p.Key.expand(w); err != nil {
        diag.errorOf(p.Key, "expand '%v' failed: %v", p.Key, err).
            debug(optionDebugErrors, 1)
        return
    } else if isNil(k) { k = p.Key }

    // Note: donot expand the p.Value! It's used as template
    // in arguments (see copy-file for example).
    if w&expandPairVal != 0 {
        if v, err = p.Value.expand(w); err != nil {
            diag.errorOf(p.Value, "expand '%v' failed: %v").
                debug(optionDebugErrors, 1)
        } else if (!isNil(k) && k != p.Key) || (!isNil(v) && v != p.Value) {
            if isNil(v) { v = p.Value }
            res = &Pair{p.valbase, k, v}
        }
    } else if !isNil(k) && k != p.Key {
        res = &Pair{p.valbase, k, p.Value}
    }
    return
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

// Delegate wraps '$(foo a,b,c)' into Valuer
type delegate struct {
    valbase
    l token.Token
    x Value
    a []Value
}
func (p *delegate) True() (t bool, err error) {
    var v Value
    if v, err = p.expand(expandPlainValue); err != nil {
        diag.errorAt(p.position, "expand '%v' failed: %v", p, err).
            debug(optionDebugErrors, 1)
    } else if isNil(v) {
        diag.errorAt(p.position, "expand '%v' to nil", p).
            debug(optionDebugErrors, 1)
    } else if t, err = v.True(); err != nil {
        diag.errorAt(p.position, "truthify '%v' failed: %v", p, err).
            debug(optionDebugErrors, 1)
    } else if false {
        diag.infoAt(p.position, "%v -> %T %v -> %v", p, v, v, t).
            debug(optionDebugErrors, 8)
    }
    return
}
func (p *delegate) String() (s string) { return p.elemstr(nil, 0) }
func (p *delegate) Strval() (s string, err error) {
    var v Value
    if v, err = p.value(); err != nil {
        diag.errorOf(p, "delegate '%v' value failed: %v", p, err).
            debug(optionDebugErrors, 1)
    } else if isNil(v) {
        diag.errorOf(p, "delegate value is nil: %v", p).
            debug(optionDebugErrors, 1)
    } else if s, err = v.Strval(); err != nil {
        diag.errorOf(v, "strval '%v' failed: %v", v, err).
            debug(optionDebugErrors, 1)
    }
    return
}
func (p *delegate) Integer() (i int64, err error) {
    var v Value
    if v, err = p.value(); err != nil {
        diag.errorOf(p, "delegate '%v' value failed: %v", p, err).
            debug(optionDebugErrors, 1)
    } else if isNil(v) {
        diag.errorOf(p, "delegate value is nil: %v", p).
            debug(optionDebugErrors, 1)
    } else if i, err = v.Integer(); err != nil {
        diag.errorOf(v, "integify '%v' failed: %v", v, err).
            debug(optionDebugErrors, 1)
    }
    return
}
func (p *delegate) Float() (f float64, err error) {
    var v Value
    if v, err = p.value(); err != nil {
        diag.errorOf(p, "delegate '%v' value failed: %v", p, err).
            debug(optionDebugErrors, 1)
    } else if isNil(v) {
        diag.errorOf(p, "nil delegate value: %v", p).
            debug(optionDebugErrors, 1)
    } else if f, err = v.Float(); err != nil {
        diag.errorOf(v, "floatify '%v' failed: %v", v, err).
            debug(optionDebugErrors, 1)
    }
    return
}
func (p *delegate) isValidToken() (res bool) {
    switch p.l {
    case token.LCOLON, token.LPAREN, token.LBRACE, token.STRING, token.COMPOUND, token.ILLEGAL:
        res = true
    default:
        // for $. $/ $1 ... &. &/ &1 ... etc.
        res = p.l.IsClosure() || p.l.IsDelegate()
    }
    return
}
func (p *delegate) string(o Object, k elemkind) (s string) { // source representation
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
    case token.LPAREN:
        s = fmt.Sprintf("(%s%s)", name, s)
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
        if p.l.IsClosure() || p.l.IsDelegate() {
            s = p.l.String()
        } else {
            s = fmt.Sprintf("[%s%s]!(%v)", name, s, p.l)
        }
    }
    return
}
func (p *delegate) refs(v Value) (res bool) {
    if isNil(p.x) {
        diag.errorOf(p, "delegation of nil (v=%v)", v).
            debug(optionDebugErrors, 1)
        return
    } else if p.x == v || p.x.refs(v) {
        return true
    } else {
        for _, a := range p.a {
            if a.refs(v) { return true }
        }
    }
    return
}
func (p *delegate) defs(s string) (res []*Def) {
    if isNil(p.x) {
        diag.errorOf(p, "delegation of nil (s=%v)", p, s).
            debug(optionDebugErrors, 1)
        return
    } else if d, ok := p.x.(*Def); ok && (s == "" || d.name == s) {
        res = append(res, d)
    } else {
        res = p.x.defs(s)
    }
    for _, a := range p.a {
        res = append(res, a.defs(s)...)
    }
    return
}
func (p *delegate) traverse(t *traversal) {
    if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
    if val, err := p.expand(expandPlainValue); err != nil {
        diag.errorAt(p.position, "expand '%v' failed: %v", p, err).
            debug(optionDebugErrors, 1)
    } else if isNil(val) {
        diag.warnAt(p.position, "delegate '%v' expands to nil", p).
            debug(optionDebugErrors, 1)
    } else if isNone(val) {
        diag.warnAt(p.position, "delegate '%v' expands to none", p).
            debug(optionDebugErrors, 1)
    } else {
        val.traverse(t)
    }
}
func (p *delegate) elemstr(o Object, k elemkind) (s string) {
    if k&elemExpand == 0 {
        if s = p.string(o, k); !(token.DELEGATE < p.l && p.l <= token.DELEGATE__) {
            s = "$" + s
        }
    } else if v, e := p.expand(expandDelegate); e != nil {
        diag.errorAt(p.position, "expand failed: %v", e).
            debug(optionDebugErrors, 1)
    } else {
        s = elementString(o, v, k)
    }
    return
}
func (p *delegate) value() (v Value, err error) {
    if v, err = p.expand(expandDelegate); err != nil {
        diag.errorAt(p.position, "expand '%v' failed: %v", p, err).
            debug(optionDebugErrors, 1)
    } else if v == p { // d, ok := v.(*delegate); ok && d == p
        diag.errorOf(p, "self delegation: %v", p).
            debug(optionDebugErrors, 1)
    }
    return
}
func (p *delegate) args(w expandwhat) (args []Value, num int, err error) {
    if w&expandArgs != 0 {
        if args, num, err = expandall1(w, p.a...); err != nil {
            diag.errorAt(p.position, "expand args %v failed: %v", p.a, err).
                debug(optionDebugErrors, 1)
            return
        } else if len(args) == 0 && num == 0 && len(p.a) > 0 { args = p.a }
    } else if len(p.a) > 0 { args = p.a }
    return
}
func (p *delegate) expandible(w expandwhat) (res bool) {
    if res = w&expandDelegate != 0; !res {
        if res = p.x.expandible(w); !res && w&expandArgs != 0 {
            for _, a := range p.a {
                if res = a.expandible(w); res { break }
            }
        }
    }
    return
}
func (p *delegate) expand(w expandwhat) (res Value, err error) {
    if isNil(p.x) {
        diag.errorAt(p.position, "expand nil delegation (w=%016b)", w).
            debug(optionDebugErrors, 32)
        return
    }

    var v Value
    if w&expandDelegate == 0 {
        var x Value
        if x, err = p.x.expand(w); err != nil {
            diag.errorOf(p.x, "expand '%v' failed: %v", p.x, err).
                debug(optionDebugErrors, 1)
            return
        } else if isNil(x) { x = p.x }

        var ( args []Value; num int )
        if args, num, err = p.args(w); err != nil {
            diag.errorAt(p.position, "expand args failed: %v", err).
                debug(optionDebugErrors, 1)
            return
        }

        if (!isNil(x) && x != p.x) || num > 0 {
            if num == 0 { args = p.a }
            res = &delegate{p.valbase, p.l, x, args}
        }
    } else if res, err = p.reveal(w); err != nil {
        diag.errorOf(p, "reveal '%v' failed: %v", p, err).
            debug(optionDebugErrors, 1)
    } else if isNil(res) {
        res = MakeNone(p.position)
    } else if v, err = res.expand(w); err != nil {
        diag.errorOf(p, "expand '%v' failed: %v", res, err).
            debug(optionDebugErrors, 1)
    } else if !isNil(v) && v != res {
        res = v
    }
    return
}
func (p *delegate) reveal(w expandwhat) (res Value, err error) {
    var x Object
    if t, ok := p.x.(Object); ok && t != nil {
        x = t
    } else if t, ok := p.x.(*selection); ok && t != nil && !isNil(t.o) {
        if isNil(t.o) {
            diag.errorAt(p.position, "%v: selection object is nil (w=%016b)", p, w).
                debug(optionDebugErrors, 16)
            return
        } else if n, ok := t.o.(*ProjectName); ok && n != nil && n.project != nil {
            defer setclosure(setclosure(cloctx.unshift(n.project.scope)))
        }
        if v, e := t.value(); e != nil {
            diag.errorAt(p.position, "select value '%v' failed: %v", p.x, e).
                debug(optionDebugErrors, 1)
            err = e; return
        } else if isNil(v) {
            diag.errorAt(p.position, "%v: selected value is nil (%T %v) (w=%016b)", p, t.o, t.o, w).
                debug(optionDebugErrors, 16)
            return
        } else if t, ok := v.(Object); ok {
            x = t
        }
    } else {
        diag.errorOf(p, "delegate unsupport value '%v'", p.x).
            debug(optionDebugErrors, 1)
        return
    }

    var args []Value
    if args, _, err = p.args(w); err != nil { return }

    switch t := x.(type) {
    case Caller:
        if res = t.Call(p.position, args...); isNil(res) {
            if d, ok := x.(*Def); ok && !isNil(d.value) {
                diag.errorAt(p.position, "calling Def '%v' (%v) returned incorrect value (value=%v (%T))",
                    d.Name(), d.origin, d.value, d.value).debug(optionDebugErrors, 1)
            }
        }
    case Executer:
        if vals, brks := t.Execute(p.position, args...); len(brks) > 0 {
            for _, brk := range brks {
                var s string
                if brk.message != "" { s = brk.message }
                if brk.error != nil { s += fmt.Sprintf(" (error: %s)", brk.error) }
                diag.errorAt(brk.pos, "broken '%v': (%s) %s", x, brk.what, s).
                    debug(optionDebugErrors, 1)
            }
        } else if len(vals) > 0 {
            res = MakeList(Position{}, vals...)
        }
    default:
        diag.errorAt(p.position, "unknown delegation: %v (%T) -> %v %T", p.x, p.x, t, t).
            debug(optionDebugErrors, 32)
    }
    return
}
func (p *delegate) stat(t *traversal) (si *statinfo) {
    diag.errorAt(p.position, "cant stat delegate %v, must expand it first", p).
        debug(optionDebugErrors, 16)
    return
}
func (p *delegate) stamp(t *traversal) (file []*File, err error) {
    diag.errorAt(p.position, "cant stamp delegate %v, must expand it first", p).
        debug(optionDebugErrors, 16)
    return
}
func (p *delegate) cmp(v Value) (res cmpres) {
    if a, ok := v.(*delegate); ok {
        // NOTE: don't compare the expanded value!!
        if res = p.x.cmp(a.x); res == cmpEqual && len(p.a) == len(a.a) {
            for i, t := range p.a {
                if res = t.cmp(a.a[i]); res != cmpEqual { return }
            }
        }
    } else if d, ok := p.x.(*Def); ok && len(p.a) == 0 {
        res = d.value.cmp(v)
    }
    return
}

type closure struct { delegate }
func (p *closure) True() (t bool, err error) {
    var v Value
    if v, err = p.expand(expandPlainValue); err != nil {
        diag.errorAt(p.position, "expand '%v' failed: %v", p, err).
            debug(optionDebugErrors, 1)
    } else if isNil(v) {
        // does nothing
    } else if t, err = v.True(); err != nil {
        diag.errorAt(p.position, "truthify '%v' failed: %v", p, err).
            debug(optionDebugErrors, 1)
    } else if false {
        diag.infoAt(p.position, "%v -> %T %v -> %v", p, v, v, t).
            debug(optionDebugErrors, 8)
    }
    return
}
func (p *closure) Strval() (s string, err error) {
    var v Value

    if !p.isValidToken() {
        err = fmt.Errorf("invalid closure token: %v", p.l)
        diag.errorAt(p.Position(), err.Error()).
            debug(optionDebugErrors,1)
        return
    }

    if v, err = p.expand(expandDelegate|expandClosure); err != nil {
        diag.errorOf(p, "expand '%v' failed: %v", p, err).
            debug(optionDebugErrors, 1)
    } else if isNil(v) {
        if false { diag.warnOf(p, "expand '%v' to nil", p).debug(optionDebugErrors, 1) }
    } else if s, err = v.Strval(); err != nil {
        diag.errorOf(p, "strval '%v' failed: %v", p, err).
            debug(optionDebugErrors, 1)
    }
    return
}
func (p *closure) String() (s string) { return p.elemstr(nil, 0) }
func (p *closure) elemstr(o Object, k elemkind) (s string) {
    if k&elemExpand == 0 {
        if s = p.string(o, k); !(token.CLOSURE < p.l && p.l <= token.CLOSURE__) {
            s = "&" + s
        }
    } else if v, e := p.expand(expandDelegate/*|expandClosure*/); e != nil {
        diag.errorAt(p.position, "expand '%v' failed: %v", p, e).
            debug(optionDebugErrors, 1)
        return
    } else {
        if isNil(v) { v = p }
        s = elementString(o, v, k)
    }
    return
}
func (p *closure) refs(v Value) bool {
    if p.x == v { return true }
    for _, a := range p.a { if a.refs(v) { return true }}
    return false
}
func (p *closure) defs(s string) (res []*Def) {
    res = append(res, p.x.defs(s)...)
    for _, a := range p.a {
        res = append(res, a.defs(s)...)
    }
    return
}
func (p *closure) expandible(w expandwhat) (res bool) {
    if res = w&expandClosure != 0; !res {
        if res = p.x.expandible(w); !res && w&expandArgs != 0 {
            for _, a := range p.a {
                if res = a.expandible(w); res { break }
            }
        }
    }
    return
}
func (p *closure) expand(w expandwhat) (res Value, err error) {
    if isNil(p.x) {
        diag.errorAt(p.position, "expand nil closure: %v (%d)", p, w).
            debug(optionDebugErrors, 1)
        return
    }

    var v Value
    if w&expandClosure == 0 {
        var x Value
        if x, err = p.x.expand(w); err != nil {
            diag.errorOf(p.x, "expand '%v' failed: %v", p.x, err).
                debug(optionDebugErrors, 1)
            return
        } else if isNil(x) { x = p.x }

        var ( args []Value; num int )
        if args, num, err = p.args(w); err != nil { return }

        if (!isNil(x) && x != p.x) || num > 0 {
            if num == 0 { args = p.a }
            res = &closure{delegate{p.valbase, p.l, x, args}}
        }
    } else if res, err = p.disclose(w); err != nil {
        diag.errorOf(p, "disclose '%v' failed: %v", p, err).
            debug(optionDebugErrors, 1)
    } else if isNil(res) {
        diag.errorOf(p, "disclose '%v' to nil (%s '%v')", p, typeof(p.x), p.x).
            debug(optionDebugErrors, 16)
    } else if w&^expandClosure == 0 {
        // done, no more expand
    } else if v, err = res.expand(w); err != nil {
        diag.errorOf(p, "expand '%v' failed: %v", res, err).
            debug(optionDebugErrors, 1)
    } else if !isNil(v) && v != res {
        if false { diag.warnOf(p, "%v -> %T %v -> %T %v (w=%016b)", p, res, res, v, v, w).debug(true,16) }
        if false && isNone(v) { if d, ok := res.(*delegate); ok && !isNone(d.x) { if dd, ok := d.x.(*Def); ok && !isNone(dd.value) {
            diag.errorOf(p, "expand '%v -> %v' incorrectly: %s %v (%T) (w=%016b)",
                p, res, dd.name, dd.value, dd.value, w).debug(optionDebugErrors, 48)
        }}}
        res = v
    }
    return
}
func (p *closure) disclose(w expandwhat) (res Value, err error) {
    var x Object
    if t, ok := p.x.(Object); ok && t != nil {
        x = t
    } else if t, ok := p.x.(*selection); ok && t != nil && !isNil(t.o) {
        if n, ok := t.o.(*ProjectName); ok && n != nil && n.project != nil {
            defer setclosure(setclosure(cloctx.unshift(n.project.scope)))
        }
        if v, e := t.value(); e != nil {
            diag.errorOf(p.x, "select value '%v' failed: %v", p.x, e).
                debug(optionDebugErrors, 1)
            err = e; return
        } else if t, ok := v.(Object); ok {
            x = t
        }
    }

    var name string
    if isNil(x) {
        diag.errorOf(p.x, "closure non-object: %v (%T)", p.x, p.x).
            debug(optionDebugErrors, 1)
        return
    } else if name = x.Name(); name == "" {
        diag.errorOf(p.x, "empty closure name: %T %v -> %T %v", p.x, p.x, x, x).
            debug(optionDebugErrors, 1)
        return
    }

ClosureTok:
    switch p.l {
    case token.LBRACE, token.STRING, token.COMPOUND:
        for _, scope := range cloctx {
            var entry *RuleEntry
            if scope.project == nil {
                // continue
            } else if entry, err = scope.project.resolveEntry(name, false); err != nil {
                diag.errorAt(p.position, "resolve entry '%s' in '%s' failed: %v", name, scope.project, err).
                    debug(optionDebugErrors, 1)
                return
            } else if entry == nil {
                // continue
            } else if p.l == token.LBRACE {
                x = entry
                break ClosureTok
            } else { // token.STRING, token.COMPOUND
                // &'xxx' and &"xxx" are fetching entry in the closure context
                res = entry
                return
            }
        }
    default: //case token.LPAREN, token.ILLEGAL:
        for _, scope := range cloctx {
            var s Object
            if scope.project == nil {
                continue
            } else if scope != scope.project.scope {
                if _, s = scope.Find(name); !isNil(s) {
                    x = s; break ClosureTok
                }
            }
            if s, err = scope.project.resolveObject(name); err != nil {
                diag.errorAt(p.position, "resolve object '%s' in '%s' failed: %v", name, scope.project, err).
                    debug(optionDebugErrors, 1)
                return
            } else if !isNil(s) {
                x = s; break ClosureTok
            }
        }
    }

    var ( args []Value; num int )
    if args, num, err = p.args(w); err != nil { return }

    if isNil(x) {
        diag.errorAt(p.position, "closure object is nil: %v", p.x).
            debug(optionDebugErrors, 1)
    } else if p.x != x || num > 0 {
        if _, ok := x.(*unresolvedobject); false && ok {
            diag.warnAt(p.position, "object '%s' (%v) is undefined: %v", name, p.x, cloctx).
                debug(optionDebugErrors, 8)
        }
        res = &delegate{p.valbase, p.l, x, args}
    } else {
        res = &p.delegate
    }

    if false {
        diag.warnOf(p, "%v -> %T %v -> %T %v (same=%v); args=%v", p, p.x, p.x, x, x, (p.x==x), args)
        diag.warnOf(p, "%v -> %v (same=%v, x=%v)", p, res, (p==res), x).debug(true, 16)
    }
    return
}
func (p *closure) traverse(t *traversal) {
    if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
    if val, err := p.expand(expandClosure); err != nil {
        diag.errorAt(p.position, "expand '%v' failed: %v", p, err).
            debug(optionDebugErrors,1)
    } else if isNil(val) {
        diag.warnAt(p.position, "closure '%v' expands to nil", p).
            debug(optionDebugErrors,1)
    } else if isNone(val) {
        diag.warnAt(p.position, "closure '%v' expands to none", p).
            debug(optionDebugErrors,1)
    } else {
        val.traverse(t)
    }
}
func (p *closure) stat(t *traversal) (si *statinfo) {
    diag.errorAt(p.position, "cant stat closure %v, must expand it first", p).
        debug(optionDebugErrors, 16)
    return
}
func (p *closure) stamp(t *traversal) (file []*File, err error) {
    diag.errorAt(p.position, "cant stamp closure %v, must expand it first", p).
        debug(optionDebugErrors, 16)
    return
}
func (p *closure) cmp(v Value) (res cmpres) {
    if a, ok := v.(*closure); ok {
        // NOTE: don't compare the expanded value!!
        if res = p.x.cmp(a.x); res == cmpEqual && len(p.a) == len(a.a) {
            for i, t := range p.a {
                if res = t.cmp(a.a[i]); res != cmpEqual { return }
            }
        }
    }
    return
}

type selection struct {
    valbase
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
func (p *selection) String() string { return p.elemstr(nil, 0) }
func (p *selection) Strval() (s string, err error) {
    if n, ok := p.o.(*ProjectName); ok && n != nil {
        defer setclosure(setclosure(cloctx.unshift(n.project.scope)))
        if false && options.debug {
            fmt.Fprintf(stderr, "%s: %v %v\n", p.position, p, cloctx)
            debug.PrintStack()
        }
    }

    var v Value
    if v, err = p.value(); err != nil {
        diag.errorAt(p.position, "%v", err)
    } else if v != nil {
        if s, err = v.Strval(); err != nil { diag.errorAt(p.position, "%v", err) }
        if false && options.debug {
            fmt.Fprintf(stderr, "%s: %v → %v\n", p.position, v, s)
            debug.PrintStack()
        }
    } else if false {
        diag.errorAt(p.position, "selection.strval: `%s` is nil", p.String())
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
func (p *selection) defs(s string) []*Def { return append(p.o.defs(s), p.s.defs(s)...) }
func (p *selection) elemstr(o Object, k elemkind) (s string) {
    if _, ok := p.o.(*usinglist); ok { s = "usee" } else {
        s = elementString(o, p.o, k)
    }
    s += p.t.String() + elementString(o, p.s, k)
    return
}
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
            diag.errorAt(p.position, "selection.object: `%s` is nil", s.String())
        }
    } else if o, ok = p.o.(Object); !ok {
        diag.errorAt(p.position, "selection.object: '%v' is not object (but %s)", p.o, typeof(p.o))
    }
    return
}
func (p *selection) value() (v Value, err error) {
    var o Object
    if isNil(p.s) {
        diag.errorAt(p.position, "selection prop is nil: %s", p.String()).
            debug(optionDebugErrors, 1)
    } else if o, err = p.object(); err != nil {
        diag.errorAt(p.position, "get selection object failed: %v", err).
            debug(optionDebugErrors, 1)
    } else if s := ""; o != nil {
        /*if n, ok := o.(*ProjectName); ok && n != nil && n.project != nil {
            defer setclosure(setclosure(cloctx.unshift(n.project.scope)))
        }*/
        if s, err = p.s.Strval(); err == nil {
            if pn, ok := o.(*ProjectName); ok && (p.t == token.SELECT_PROG1 || p.t == token.SELECT_PROG2) {
                var entry *RuleEntry
                if entry, err = pn.project.resolveEntry(s, false); err != nil {
                    return
                } else if entry == nil {
                    diag.errorAt(p.position, "selection.value: no entry `%s` (%+v)", s, p.String())
                } else {
                    v = entry
                }
            } else if v, err = o.Get(s); err != nil {
                diag.errorAt(p.position, "%v", err)
                if false && options.debug {
                    fmt.Fprintf(stderr, "%s: %v %v\n", p.position, p, cloctx)
                    debug.PrintStack()
                }
            }
        }
    } else /*if o == nil*/ {
        diag.errorAt(p.position, "selection.value: nil object `%s`", p.String())
    }
    return
}
func (p *selection) expandible(w expandwhat) (res bool) {
    if res = w&expandSelection != 0; !res {
        res = p.o.expandible(w) || p.s.expandible(w)
    }
    return
}
func (p *selection) expand(w expandwhat) (res Value, err error) {
    if w&expandSelection != 0 {
        if res, err = p.value(); err != nil {
            diag.errorAt(p.position, "selection '%v' failed: %v", p, err).
                debug(optionDebugErrors, 1)
        }
    } else if isNil(p.o) {
        return // nil object
    } else if isNil(p.s) {
        return // nil prop
    }

    var o, s Value
    if o, err = p.o.expand(w); err != nil {
        diag.errorOf(p.o, "expand '%v' failed: %v", p.o, err).
            debug(optionDebugErrors, 1)
        return
    } else if isNil(o) { o = p.o }
    if s, err = p.s.expand(w); err != nil {
        diag.errorOf(p.s, "expand '%v' failed: %v", p.s, err).
            debug(optionDebugErrors, 1)
        return
    } else if isNil(s) { s = p.s }

    if o != p.o || s != p.s { res = &selection{p.valbase,p.t,o,s}}
    return
}
func (p *selection) traverse(t *traversal) {
    if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
    if val, err := p.value(); err != nil {
        diag.errorAt(p.position, "select value '%v' failed: %v", p, err).
            debug(optionDebugErrors, 1)
    } else if isNil(val) {
        diag.warnAt(p.position, "selected value '%v' is nil", p).
            debug(optionDebugErrors, 1)
    } else if isNone(val) {
        diag.warnAt(p.position, "selected value '%v' is none", p).
            debug(optionDebugErrors, 1)
    } else {
        val.traverse(t)
    }
}
func (p *selection) stat(t *traversal) (si *statinfo) {
    diag.errorAt(p.position, "cant stat selection %v, must expand it first", p).
        debug(optionDebugErrors, 1)
    return
}
func (p *selection) stamp(t *traversal) (file []*File, err error) {
    diag.errorAt(p.position, "cant stamp selection %v, must expand it first", p).
        debug(optionDebugErrors, 1)
    return
}
func (p *selection) cmp(v Value) (res cmpres) {
    if a, ok := v.(*selection); ok && p.t == a.t {
        if res = p.o.cmp(a.o); res == cmpEqual {
            if res = p.s.cmp(a.s); res == cmpEqual {
                // if p.t == a.t { res = cmpEqual }
            }
        }
    }
    return
}

/*
type partialMatcher interface {
    partialMatch(i interface{}) (result string, rest, stems []string, err error)
}

// TODO: endingMatcher is not implemented (e.g. $(trim-suffix .%, a.xxx b.xxx))
type endingMatcher interface {
    endingMatch(i interface{}) (result string, rest, stems []string, err error)
}
*/

// Pattern
/*type Pattern interface {
    Value
    match(i interface{}) (s string, stems []string, err error)
    stencil(stems []string) (s string, rest []string, err error)
}*/

// PercPattern represents percent pattern expressions (e.g. '%.o')
type PercPattern struct {
    valbase // TODO: supporting multiple %: foo%bar%xxx
    Prefix Value
    Suffix Value
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
func (p *PercPattern) refs(v Value) bool { return p.Prefix.refs(v) || p.Suffix.refs(v) }
func (p *PercPattern) defs(s string) []*Def { return append(p.Prefix.defs(s), p.Suffix.defs(s)...) }
func (p *PercPattern) expandible(w expandwhat) bool { return p.Prefix.expandible(w) || p.Suffix.expandible(w) }
func (p *PercPattern) elemstr(o Object, k elemkind) (s string) {
    s  = elementString(o, p.Prefix, k) + `%`
    s += elementString(o, p.Suffix, k)
    return
}
func (p *PercPattern) patterned() bool { return true }
func (p *PercPattern) match1(rep string) (full bool, result string, stems []string) {
    if optionEnableBenchspots { defer bench(spot("PercPattern.match")) }

    var ( prefix string; err error )
    if !(isNil(p.Prefix) || isNone(p.Prefix)) {
        // FIXME: the prefix could be Glob, Regexp, etc.
        if prefix, err = p.Prefix.Strval(); err != nil {
            diag.errorOf(p.Prefix, "prefix strval '%v' failed: %v", p.Prefix, err)
            return
        } else if strings.HasPrefix(rep, prefix) {
            result = prefix
        } else {
            return
        }
    }

    var a, b = len(prefix), len(rep)
    if isNil(p.Suffix) || isNone(p.Suffix) {
        if a < b { stems, result, full = append(stems, rep[a:]), rep, true }
    } else if pp, ok := p.Suffix.(*PercPattern); a < b && ok {
        // fooxxbaryybaz -> foo%bar%baz => (foo xx bar yy baz) [xx yy]
        // fooxxx -> foo%% => (foo xxx) [xxx]
        // fooxxxbar -> foo%%bar => (foo xxx bar) [xxx]
        for ok {
            if isNil(pp.Prefix) || isNone(pp.Prefix) {
                // does nothing
            } else if s, e := pp.Prefix.Strval(); e != nil {
                diag.errorOf(pp.Prefix, "strval '%v' failed: %v", pp.Prefix, e)
                return
            } else if s != "" {
                if n := strings.Index(rep[a:], s); n < 0 {
                    break
                } else {
                    var v = rep[a:a+n]
                    stems = append(stems, v)
                    result += v + s
                    a += n + len(s)
                }
            }
            var pp2 *PercPattern
            if isNil(pp.Suffix) || isNone(pp.Suffix) {
                var s = rep[a:] // let %% matches everything else
                full, stems = true, append(stems, s)
                result += s
                break
            } else if pp2, ok = pp.Suffix.(*PercPattern); ok {
                pp = pp2
                continue
            } else if s, e := pp.Suffix.Strval(); e != nil {
                diag.errorOf(pp.Prefix, "strval '%v' failed: %v", pp.Suffix, e)
                return
            } else if s != "" && strings.HasSuffix(rep[a:], s) {
                if b -= len(s); a < b {
                    stems = append(stems, rep[a:b])
                    result += rep[a:]
                    full = true
                }
                break
            }
        }
    } else if a < b && p.Suffix.patterned() {
        if false {
            diag.warnOf(p.Suffix, "mixing % pattern might have performance impact: %v", p).
                debug(optionDebugErrors, 1)
        }
        for n := b-1; a < n; n -= 1 {
            if f, s, ss := p.Suffix.match(rep[n:]); f && s != "" {
                stems = append(append(stems, rep[a:n]), ss...)
                result += s // rep[a:]
                full = f
                break
            }
        }
   } else if a <= b {
        var s string
        if s, err = p.Suffix.Strval(); err != nil {
            diag.errorOf(p.Suffix, "strval '%v' failed: %v", p.Suffix, err)
        } else if strings.HasSuffix(rep[a:], s) {
            if b -= len(s); a < b {
                stems = append(stems, rep[a:b])
                result = rep
                full = true
            }
        }
    } else {
        // does nothing
    }
    return
}
func (p *PercPattern) match(i interface{}) (full bool, result string, stems []string) {
    if optionEnableBenchspots { defer bench(spot("PercPattern.match")) }

    switch t := i.(type) {
    case string: return p.match1(t)
    case *filestub:
        if full, result, stems = p.match1(t.name); full || result != "" {
            return // NOTE: done if path fully or partially matched
        } else if t.dir != "" || t.sub != "" {
            return p.match1(filepath.Join(t.dir, t.sub, t.name))  // NOTE: matching the fullname form
        }
    case *File:
        if full, result, stems = p.match1(t.name); full || result != "" {
            return // NOTE: done if path fully or partially matched
        } else if t.dir != "" || t.sub != "" {
            return p.match1(t.fullname()) // NOTE: matching the fullname form
        }
    case Value:
        if rep, err := t.Strval(); err != nil {
            diag.errorOf(t, "strval '%v' failed: %v", t, err)
        } else { return p.match1(rep) }
    default:
        unreachable(fmt.Sprintf("perc.match: %T %v", i, i))
    }
    return
}
func (p *PercPattern) stencil(stems []string) (s string, rest []string) {
    if optionEnableBenchmarks && false { defer bench(mark(fmt.Sprintf("PercPattern.stencil(%v)", p))) }
    if optionEnableBenchspots { defer bench(spot("PercPattern.stencil")) }

    var err error
    if !(isNil(p.Prefix) || isNone(p.Prefix)) {
        // FIXME: the prefix could be Glob, Regexp, etc.
        if s, err = p.Prefix.Strval(); err != nil {
            diag.errorOf(p.Suffix, "strval prefix '%v' failed: %v", p.Prefix, err).
                debug(optionDebugErrors, 1)
            return
        }
    }

    if len(stems) > 0 {
        s += stems[0]
        rest = stems[1:]
    } else if s != "" {
        // return
    }

    var v string
    if isNil(p.Suffix) || isNone(p.Suffix) {
        // done
    } else if p.Suffix.patterned() {
        // FIXME: patterns like '%%...' should use only one stem,
        // FIXME: patterns like '%xxx%...' use multiple stems.
        if v, rest = p.Suffix.stencil(rest); v != "" { s += v }
    } else if v, err = p.Suffix.Strval(); err != nil {
        diag.errorOf(p.Suffix, "strval suffix '%v' failed: %v", p.Suffix, err).
            debug(optionDebugErrors, 1)
        return
    } else if v != "" {
        s += v
    }
    return
}
func (p *PercPattern) traverse(t *traversal) { t.pattern(p) }
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
func percperc(p Value) (t bool, prefix, suffix Value) {
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

func correctPathSegForMatch(seg Value) Value {
    if bc, ok := seg.(*Barecomp); ok {
        for _, elem := range bc.Elems {
            if _, t := elem.(*Path); t { seg = nil; break }
        }
    }
    return seg
}

// GlobPattern represents glob pattern expressions (e.g. '*.o', '[a-z].o', 'a?a.o')
// 
// The pattern syntax is:
//
//	pattern:
//		{ term }
//	term:
//		'*'     matches any sequence of non-Separator characters
//		'?'     matches any single non-Separator character
//		'[' [ '^' ] { character-range } ']'
//		        character class (must be non-empty)
//		c       matches character c (c != '*', '?', '\\', '[')
//		'\\' c      matches character c
//
//	character-range:
//		c       matches character c (c != '\\', '-', ']')
//		'\\' c      matches character c
//		lo '-' hi   matches character c for lo <= c <= hi
type GlobPattern struct {
    valbase
    Components []Value
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
func (p *GlobPattern) refs(v Value) (res bool) {
    for _, comp := range p.Components {
        if res = comp.refs(v); res { break }
    }
    return
}
func (p *GlobPattern) defs(s string) (res []*Def) {
    for _, comp := range p.Components {
        res = append(res, comp.defs(s)...)
    }
    return
}
func (p *GlobPattern) expandible(w expandwhat) (res bool) {
    for _, comp := range p.Components {
        if res = comp.expandible(w); res { break }
    }
    return
}
func (p *GlobPattern) expand(w expandwhat) (res Value, err error) {
    var ( components []Value; num int )
    if components, num, err = expandall1(w, p.Components...); err != nil {
        diag.errorOf(p, "expand glob components failed: %v (w=%016b)", err, w).
            debug(optionDebugErrors, 1)
    } else if num > 0 {
        res = &GlobPattern{p.valbase, components}
    }
    return
}
func (p *GlobPattern) elemstr(o Object, k elemkind) (s string) {
    for _, comp := range p.Components {
        s += elementString(o, comp, k)
    }
    return
}
func (p *GlobPattern) patterned() bool { return true }
func (p *GlobPattern) match(i interface{}) (full bool, result string, stems []string) {
    if optionEnableBenchspots { defer bench(spot("GlobPattern.match")) }
    var ( pat, s string; e error )
    switch t := i.(type) {
    case string: s = t
    case *File: s = t.name
    case *filestub: s = t.name
    case Value:
        if s, e = t.Strval(); e != nil {
            diag.errorOf(t, "strval '%v' failed: %v", t, e)
            return
        }
    default:
        unreachable("glob.match: %T %v", i, i)
    }
    if pat, e = p.Strval(); e != nil {
        diag.errorAt(p.position, "strval '%v' failed: %v", p, e)
    } else if full, e = filepath.Match(pat, s); e != nil {
        diag.errorAt(p.position, "glob match '%s' failed: %v", pat, e)
    } else if full {
        result = s
        // FIXME: calculate stems from matching
    }
    return
}
func (p *GlobPattern) stencil(stems []string) (s string, rest []string) {
    unreachable(fmt.Sprintf("Unimplemented GlobPattern stencil %v (stems=%v)", p, stems))
    return
}
func (p *GlobPattern) traverse(t *traversal) { t.pattern(p) }
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
type RegexpPattern struct { valbase }
func (p *RegexpPattern) String() string { return "{RegexpPattern}" }
func (p *RegexpPattern) Strval() (s string, err error) { return "", nil }
func (p *RegexpPattern) patterned() bool { return true }
func (p *RegexpPattern) match(i interface{}) (full bool, result string, stems []string) {
    if optionEnableBenchspots { defer bench(spot("RegexpPattern.match")) }
    unreachable("regexp.match: %T %v", i, i)
    return
}
func (p *RegexpPattern) stencil(stems []string) (s string, rest []string) {
    unreachable("regexp.stencil: %v", stems)
    return
}
func (p *RegexpPattern) traverse(t *traversal) { t.pattern(p) }
func (p *RegexpPattern) cmp(v Value) (res cmpres) {
    if a, ok := v.(*RegexpPattern); ok {
        if a != nil { /* FIXME: ... */ }
    }
    return
}

func NewRegexpPattern(pos Position) Value {
    return &RegexpPattern{valbase{pos}} // TODO: RegexpPattern implementation
}

type Valuer interface {
    Value() Value
}

type Caller interface {
    Call(pos Position, args... Value) (result Value)
}

type Executer interface {
    Execute(pos Position, a... Value) (result []Value, breakers []*breaker)
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
        var t Value
        //if v, err = Reveal(v); err != nil { break }
        if t, err = v.expand(expandDelegate); err != nil {
            diag.errorOf(v, "expand '%v' failed: %v", v, err).
                debug(optionDebugErrors, 1)
            break
        } else if isNil(t) { t = v }
        res = append(res, t)
    }
    return
}

// Disclose expands closures to normal value recursively.
func Disclose(values ...Value) (res []Value, err error) {
    for _, v := range values {
        var t Value
        if t, err = v.expand(expandClosure); err != nil {
            diag.errorOf(v, "expand '%v' failed: %v", v, err).
                debug(optionDebugErrors, 1)
            break
        } else if isNil(t) { t = v }
        res = append(res, t)
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

// Merge lists recursively into a single list. Previously called Join.
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

// example: mergeresult(expandall(...))
func mergeresult(res []Value, err error) ([]Value, error) {
    if err == nil { res = merge(res...) }
    return res, err
}

// example: mergeresult2(expandall2(...))
func mergeresult2(res []Value, _ int, err error) ([]Value, error) {
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
func expandall1(w expandwhat, values ...Value) (elems []Value, num int, err error) {
    defer func(i int64) { expanddepth = i } (expanddepth)
    if expanddepth += 1; expanddepth > 128 {
        err = fmt.Errorf("exceeds maximum expand depth")
        return
    }
    for _, elem := range values {
        var val Value
        if isNil(elem) {
            // TODO: report nil expand ??
        } else if val, err = elem.expand(w); err != nil {
            diag.errorOf(elem, "expand '%v' failed: %v", elem, err).
                debug(optionDebugErrors, 1)
            break
        }
        if isNil(val) || val == elem {
            elems = append(elems, elem)
        } else {
            elems = append(elems, val)
            num += 1
        }
    }
    return
}

func expandall2(w expandwhat, values ...Value) (res []Value, num int, err error) {
    if res, num, err = expandall1(w, values...); err == nil && w != 0 {
        for i, v := range res {
            if v.expandible(w) {
                t, _ := v.expand(w)
                diag.warnOf(values[i], "expand incomplete: %T %v -> %T %v (equal=%v) -> %v (w=%016b)",
                    values[i], values[i], v, v, (values[i]==v), t, w).debug(true,16)
            }
        }
    }
    return
}

func ExpandAll(values ...Value) (res []Value, err error) {
    res, _, err = expandall2(expandPlainValue, values...)
    return
}

func splitPathStr(pos Position, str string) (segments []Value) {
    for _, s := range strings.Split(str, PathSep) {
        // TODO: calculate position of each segment
        segments = append(segments, MakeBareword(pos, s))
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

func MakeNil(pos Position) Value { return &Nil{valbase{pos}} }
func MakeNone(pos Position) Value { return &None{valbase{pos}} }
func MakeSelection(pos Position, tok token.Token, lhs, rhs Value) Value { return &selection{valbase{pos}, tok, lhs, rhs} }
func MakeAnswer(pos Position, v bool) (res Value) {
    if v {
        res = &answer{valbase{pos},true}
    } else {
        res = &answer{valbase{pos},false}
    }
    return
}
func MakeBoolean(pos Position, v bool) (res Value) {
    if v {
        res = &boolean{valbase{pos},true}
    } else {
        res = &boolean{valbase{pos},false}
    }
    return
}
func MakeBin(pos Position, i int64) *Bin { return &Bin{integer{valbase{pos},i}} }
func MakeOct(pos Position, i int64) *Oct { return &Oct{integer{valbase{pos},i}} }
func MakeInt(pos Position, i int64) *Int { return &Int{integer{valbase{pos},i}} }
func MakeHex(pos Position, i int64) *Hex { return &Hex{integer{valbase{pos},i}} }
func MakeFloat(pos Position, f float64) *Float { return &Float{valbase{pos},f} }
func MakeDate(pos Position, s time.Time) *Date { return &Date{DateTime{valbase{pos},s}} }
func MakeTime(pos Position, t time.Time) *Time { return &Time{DateTime{valbase{pos},t}} }
func MakeRaw(pos Position, s string) *Raw { return &Raw{valbase{pos},s} }
func MakeString(pos Position, s string) *String { return &String{valbase{pos},s} }
func MakeFlag(pos Position, s string) *Flag { return &Flag{valbase{pos}, &Bareword{valbase{pos},s}} }
func MakeFlagValue(pos Position, v Value) *Flag { return &Flag{valbase{pos}, v} }
func MakeURL(pos Position, s *url.URL) *URL {
    var host, port string
    v := strings.Split(s.Host, ":")
    if len(v) == 1 { host = v[0] }
    if len(v) == 2 { host, port = v[0], v[1] }
    var password Value
    if t, ok := s.User.Password(); ok {password = MakeString(pos, t)}
    return &URL{ // FIXME: calculate component positions
        valbase: valbase{pos},
        Scheme: MakeString(pos, s.Scheme),
        Username: MakeString(pos, s.User.Username()),
        Password: password,
        Host: MakeString(pos, host),
        Port: MakeString(pos, port),
        Path: MakeString(pos, s.Path),
        Query: MakeString(pos, s.RawQuery),
        Fragment: MakeString(pos, s.Fragment),
    }
}
func MakeBareword(pos Position, word string) *Bareword { return &Bareword{valbase{pos},word} }
func MakeBarecomp(pos Position, elems... Value) *Barecomp { return &Barecomp{valbase{pos},elements{elems}} }
func MakeCompound(pos Position, elems... Value) *Compound { return &Compound{valbase{pos},elements{elems}} }
func MakeArgumented(val Value, args... Value) *Argumented { return &Argumented{val, args} }
func MakeList(pos Position, elems... Value) *List {
    if !pos.IsValid() && len(elems) > 0 {
        pos = elems[0].Position()
    }
    return &List{pos,elements{elems}}
}
func MakeGroup(pos Position, elems... Value) (v *Group) { return &Group{valbase{pos},elements{elems}} }
func MakeGlobMeta(pos Position, tok token.Token) *GlobMeta { return &GlobMeta{valbase{pos},tok} }
func MakeGlobRange(pos Position, v Value) *GlobRange { return &GlobRange{valbase{pos},v} }
func MakePath(pos Position, segments... Value) (v *Path) { return &Path{valbase{pos},elements{segments}/*, nil*/} }
func MakePathSeg(pos Position, ch rune) *PathSeg { return &PathSeg{valbase{pos},ch} }
func MakePathStr(pos Position, str string) *Path { return MakePath(pos, splitPathStr(pos, str)...) }
func MakePair(pos Position, k, v Value) (p *Pair) {
    p = &Pair{valbase{pos},nil,nil}
    p.SetKey(k)
    p.SetValue(v)
    return
}
func MakePercPattern(pos Position, prefix, suffix Value) *PercPattern {
    if prefix == nil { prefix = MakeNone(pos) }
    if suffix == nil { suffix = MakeNone(pos) }
    return &PercPattern{
        valbase: valbase{pos},
        Prefix: prefix,
        Suffix: suffix,
    }
}
func MakeGlobPattern(pos Position, components... Value) Value {
    return &GlobPattern{valbase:valbase{pos},Components:components}
}
func MakeDelegate(pos Position, tok token.Token, obj Value, args... Value) Value {
    return &delegate{valbase{pos}, tok, obj, args}
}
func MakeClosure(pos Position, tok token.Token, obj Value, args... Value) Value {
    if obj == nil { panic("making closure to nil object") }
    return &closure{delegate{valbase{pos}, tok, obj, args}}
}
func MakeListOrScalar(pos Position, elems []Value) (res Value) {
    if x := len(elems); x > 1 {
        res = MakeList(elems[0].Position(), elems...)
    } else if x == 1 {
        res = elems[0]
    } else {
        res = MakeNone(pos)
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
    case string:    out = MakeString(pos, v)
    case time.Time: out = &DateTime{valbase{pos},v} // FIXME: NewDate, NewTime
    case Value:     out = v
    default:    out = &Any{in} // TODO: position for any
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
