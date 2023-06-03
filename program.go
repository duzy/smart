//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "path/filepath"
    "hash/maphash"
    "io/fs"
    "strings"
    "sync"
    "time"
    "fmt"
    "os"
)

type dependPatternUnfit struct {}
func (*dependPatternUnfit) Error() string { return "pattern unfit" }

type dirtyOpts struct {
    generalOpts
    verboseUpdated  bool "vu,verbose-updated"
    verboseOutdated bool "vo,verbose-outdated"
    checksum bool "c,cs,crc,checksum"
    silent   bool "s,silent"
}
type programContext struct {
    autoContext
    sync.Mutex
    sync.WaitGroup
    by    modifierSetDirtyPatsOpts
    projs []*Project
    prog *Program
    params []string // $0, $1, $2, ...
    dirt string // reason of outdated

    start time.Time // start time

    execRec map[Value]int

    calleeErrs []error
    calleeErrsM sync.Mutex

    targets []Value // all targets def
    grepped []Value
    grepping bool

    traceLevel int
    traves travestates

    interpreted []interpreter

    debug_traverse int

    print bool // printing work directories (Entering/Leaving)
}
func (pc *programContext) caller() *programContext { return pc.Context.programContext() }
func (pc *programContext) inner() Context { return &pc.autoContext }
func (pc *programContext) aquireLock() (unlock func()) {
    pc.Lock() ; return func() { pc.Unlock() }
}
func (pc *programContext) wait() { pc.WaitGroup.Wait() }
//XXX: func (pc *programContext) stems() []string { return nil }
func (pc *programContext) String() string {
    if fullContextStringer {
        var s = strings.TrimPrefix(pc.prog.scope.comment, "rule ")
        return fmt.Sprintf("program{%s,%s}", s, pc.autoContext.String())
    } else {
        return pc.autoContext.String()
    }
}
func (pc *programContext) programContext() *programContext { return pc }
func (pc *programContext) program() *Program { return pc.prog }
func (pc *programContext) projects(ctx Context, projects ...*Project) []*Project {
    if len(pc.projs) == 0 { pc.projs = closureProjects(ctx) }
    if len(projects) > 0 {
        ForProjects: for i, proj := range projects {
            if i == 0 && proj == nil {
                pc.projs = nil // reset projects
                continue
            }
            for _, p := range pc.projs { if proj == p { continue ForProjects }}
            pc.projs = append(pc.projs, proj)
        }
    }
    return pc.projs
}
func (pc *programContext) Position() Position {
    if pc.prog != nil { return pc.prog.position }
    return pc.autoContext.Position()
}
func (pc *programContext) Project() *Project {
    if cc, ok := pc.Context.(*closureContext); ok && true {
        return cc.Project()
    } else if pc.prog != nil {
        return pc.prog.project
    }
    return pc.autoContext.Project()
}
func (pc *programContext) Scope() *Scope {
    if pc.prog != nil { return pc.prog.scope }
    return pc.autoContext.Scope()
}
func (pc *programContext) appendCallerUpdated() bool { return true }
func (pc *programContext) mustExists() bool { return false }
func (pc *programContext) closureScopes() (scopes []*Scope) {
    if cc, ok := pc.Context.(*closureContext); ok {
        if true {
            scopes = cc.closureScopes()
        } else if up := cc.programContext(); up != nil {
            scopes = up.closureScopes()
        }
    } else if true {
        // fallthrough
    } else if cc = pc.closure(); cc != nil {
        scopes = cc.closureScopes()
    }
    if pc.prog != nil { scopes = append(scopes, pc.prog.scope) }
    return
}

func (pc *programContext) dirtyOpts() *modifierSetDirtyPatsOpts { return &pc.by }
func (pc *programContext) dirtyMark(vals ...Value) {
    const (
        enableDirtyMark = true
        perUpdatedDep = true
    )
    if !enableDirtyMark {
        // does nothing
    } else if targets := merge(autoGet(pc, "@")); len(targets) == 0 {
        // should not happen, but safely ignoring..
    } else if len(vals) == 0 {
        vals = append(vals, targets...)
    } else if true {
        vals = merge(vals...)

        var (
            mat, dup bool
            opts = pc.dirtyOpts()
        )
        for _, t := range targets {
        ForVals:
            for _, val := range vals {
                if eq(pc, val, t) {
                    dup = true; continue ForVals
                }
                for _, pat := range opts.pats {
                    if mat, _, _ = pat.match(pc, val); mat {
                        if perUpdatedDep { t.updatedDeps(pc, val) }
                        break ForVals
                    }
                }
            }
            if !perUpdatedDep && mat { t.updatedDeps(pc, vals...) }
            if !dup { // vals = append(vals, merge(targets)...)
                vals = append(t.updatedDeps(pc), vals...)
                vals = append(merge(t), vals...)
            }
            if false { warn(pc, "dirtyMark: %T %v; %v, %v, %v, %v", t, t, vals, dup, t.updated(pc), t.updatedDeps(pc)).debug(0) }
            if false { warn(pc, "dirtyMark: %T %v; %v, %v, %v, %v", t, t, vals, dup, t.updated(pc), t.updatedDeps(pc)).debug(18) }
        }
    }
    if enableDirtyMark { pc.Context.dirtyMark(vals...) }
}

func isDirty(ctx Context, target Value, a ...Value) (dirty bool) {
    var opts = ctx.dirtyOpts()
    if len(target.updatedDeps(ctx)) > 0 { return true }
    if v := autoGet(ctx, "^"); v != nil { a = append(a, v) }
    for _, dep := range mergex(ctx, plain, a...) {
        var mat bool = len(opts.pats) == 0
        if !mat { for _, pat := range opts.pats { if mat, _, _ = pat.match(ctx, dep); mat { break }}}
        if mat && (dep.updated(ctx) || dep.stat(ctx).mod().After(target.stat(ctx).mod())) {
            return true
        }
    }
    return
}

func isDirtyAfter(ctx Context, target Value, t time.Time) (res bool) {
    for _, dep := range target.updatedDeps(ctx) {
        if ds := dep.stat(ctx); ds != nil {
            res = ds.mod().After(t) || isDirtyAfter(ctx, dep, t)
            if res { break }
        }
    }
    return
}

func (pc *programContext) dirty(ctx Context, aa ...Value) (outdated bool) {
    var target as
    if val, /*files*/_, /*execRes*/_, err := wait(pc); err != nil {
        errostack(ctx, 5, "%v", err).debug(10)
        return
    } else {
        target.Value = val
    }

    var reason string
    var targetFile *File
    var targetFull string
    if false { if s := target.Strval(ctx); s != "" && (
        // s == "tablegen-min" ||
        false) {
        defer func() {
            for _, s := range pc.traves { prompt(ctx, "%v: %v\n", target.Value, s) }
            prompt(ctx, "%v: %s\n", target.Value, targetFull)
            prompt(ctx, "%v: outdated=%v, %s\n", target.Value, outdated, reason)
            warnstack(ctx, 5).debug(32)
        } ()
    }}

    var opts, args = _opts[dirtyOpts](ctx, plain, aa...)

    var y bool
    if targetFile, targetFull, y = target.fullname(ctx); !y {
        targetFull = target.Strval(ctx)
    } else if n := targetFile.traversed; n > 1 {
        if false { warnstack(ctx, 5, "%v, %v, %d", targetFile, targetFull, n).debug(10) }
        return
    }

    var verb = opts.debug>0 || opts.verbose
    var ts = trimPromptString(targetFull)

    if s := target.stat(ctx); s == nil || s.exists() != existenceConfirmed {
        outdated, reason = true, fmt.Sprintf("not exists: %s %v", typeof(target), target)
    } else if isDirty(ctx, target, args...) && isDirtyAfter(ctx, target, s.mod()) {
        outdated, reason = true, "prerequisites updated"
    }

    if outdated {
        assert(reason != "", "needs outdated reason")
    } else if y, e := isRecipesChanged(ctx, target); e != nil {
        erro(ctx, "recipes changed: %v", e).debug(1)
        return
    } else if outdated = y; y {
        reason = "recipes changed"
    } else if !opts.checksum {
        // does nothing
    } else {
        erro(ctx, "FIXME: check prerequisites against the saved checksums").debug(1)
        return
    }

    verb = verb ||
        (opts.verboseOutdated && outdated) ||
        (opts.verboseUpdated && !outdated)

    if verb || outdated {
        if d := ctx.gap(true); d > 0 {
            var s = "gap " + d.String()
            if reason != "" { reason += ", " } ; reason += s
            if false && d > 10*time.Second {
                warnstack(ctx, 10, "%v: %v", target.Value, d).debug(64)
            }
        }
    }

    if verb {
        var m string
        var s = time.Now().Sub(pc.start).String()
        if reason != "" { s += "; " + strings.TrimSpace(strings.TrimPrefix(reason, "outdated:")) }
        if outdated { m = "outdated" } else { m = "updated" }

        var n = len(pc.targets) + len(pc.grepped)
        if db := opts.debug>0; db && !opts.verbose {
            warn(ctx, "%s (%T) (%s) …… %s (%d files in %s, debug=%d)", ts, target, targetFull, m, n, s, opts.debug).debug(opts.debug * 2)
        } else {
            prompt(ctx, "%s …… %s (%d files in %s)\n", ts, m, n, s).debug(db, 6)
        }
    }

    if outdated && pc.dirt != "" { reason = pc.dirt + "; " + reason }
    if !opts.silent && reason != "" { pc.dirt = reason }
    return
}

func probPrereqValue(ctx Context, projects []*Project, val Value) (prereqValue, prereqPattern Value, prereqStrval string, prereqFile *File, prereqObj Object) {
    prereqValue = val

    var mapPrereqFile = func() {
        for _, project := range projects {
            if prereqFile = project.file(ctx, prereqStrval); prereqFile != nil {
                prereqValue = prereqFile
                break
            }
        }

        if prereqFile != nil {
            // okay
            // } else if prereqFile = file(ctx, prereqStrval); prereqFile != nil {
            //     prereqValue = prereqFile
        } else if prereqValue != nil {
            if f, y := toFile(prereqValue); y { prereqFile = f }
            if _, y := prereqValue.(*Path); y {
                if f := stat(ctx, prereqStrval, "", ""); f != nil { prereqFile, prereqValue = f, f }
            }
        }
    }

    if prereqValue == nil {
        if prereqStrval == "" {
            errostack(ctx, 3, "prerequisite is nothing").debug(8)
            return
        } else if mapPrereqFile(); prereqValue == nil {
            prereqValue = MakeString(ctx.Position(), prereqStrval)
            mapPrereqFile = nil // only do it once
        }
    } else if prereqObj, _ = prereqValue.(Object); prereqObj != nil {
        if false { info(ctx, "%v: %T %v %s", prereqObj, prereqObj, prereqObj, prereqStrval).debug(1) }
    } else if prereqValue.patterned(ctx) {
        var stems = ctx.stems()
        if len(stems) == 0 {
            errostack(ctx, 3, "%v: no stems", prereqValue).debug(8)
            return
        }

        var rest []string
        prereqPattern = prereqValue
        prereqValue, rest = prereqPattern.stencil(ctx, stems)
        if isTrivial(prereqValue) {
            errostack(ctx, 3, "%v: empty stencil with %v", prereqPattern, stems).debug(8)
            return
        } else if len(rest) > 0 {
            errostack(ctx, 3, "%v: partial stencil with %v, rest=%v", prereqPattern, stems, rest).debug(8)
            return
        } else if prereqStrval == "" {
            prereqStrval = prereqValue.Strval(ctx);
        }

        if prereqStrval != "" { mapPrereqFile() } else {
            errostack(ctx, 3, "%v: empty prerequisite, stems=%v", prereqValue, stems).debug(8)
            return
        }
     } else {
        if prereqStrval = prereqValue.Strval(ctx); prereqStrval == "" { // just reject empty strval
            errostack(ctx, 3, "%v: %v: empty prerequisite, stems=%v", prereqValue, ctx.stems()).debug(8)
            return
        }

        switch prereqValue.(type) {
        case *String, *Compound: // skip checking files for performance
        default: mapPrereqFile()
        }
    }

    return
}

const traverseCacheOn = false
var traverseCache = make(map[uint64]int, 128)

func (pc *programContext) deferTrave(ctx Context, targetValue, prereqValue, prereqPattern Value, prereqFile *File) {
    var (
        av = targetValue
        bv = prereqValue
    )
    if prereqFile == nil {
        ctx.traversed(prereqValue) // set $< $> $^ or $|
    } else if targetValue != prereqFile {
        ctx.traversed(prereqFile) // set $< $> $^ or $|
        bv = prereqFile
    } else if t := pc.traves.of(traveFile); t.has() {
        for _, s := range t {
            if d := s.depend; d != bv && !isTrivial(d) { bv = d }
        }
    }

    if !isTrivial(av) && !isTrivial(bv) {
        var (
            a = av.stat(ctx).mod()
            b = bv.stat(ctx).mod()
        )
        if (!a.IsZero() && b.After(a)) || bv.updated(ctx) || bv.updatedDeps(ctx) != nil {
            av.updatedDeps(ctx, bv)
        }
    }

    if prereqFile != nil {
        if false && !prereqFile.exists() { prereqFile.stat(ctx) }

        if prereqFile.exists() { prereqFile.traversed += 1
            if pc.traves.has(traveNext) { pc.traves = pc.traves.not(traveNext) }
        }

        for _, s := range pc.traves.of(traveFile) {
            if f, y := s.depend.(*File); y {
                if t := f == prereqFile || f.fullname() == prereqFile.fullname(); t { return }
            }
        }

        trave := pc.traves.add(ctx, traveFile, targetValue)
        trave.dependPat = prereqPattern
        trave.depend = prereqFile
    }
}

// traverse - traverse the prerrequiste for the current target $@
func (pc *programContext) traverse(ctx Context, prereqValue Value) (result travestates) {
    var (
        db = false
        verb = options.verbose || options.verboseBreaks

        projects = ctx.projects(ctx)

        targetValue Value

        prereqPattern Value
        prereqStrval string
        prereqFile *File
        prereqObj Object

        concreteList []Entry
        stemmedList []*stemmed

        t1, t2 time.Time
    )
    defer func(t0 time.Time) {
        if false && (db || strings.HasSuffix(prereqStrval, "include/__cxxabi_config.h")) {
            var s = targetValue.Strval(ctx)
            var b bool ; if prereqFile != nil { b = prereqFile.exists() }
            for i, concrete := range concreteList { info(at(ctx,concrete.Position()), "%v : concrete: %d. %v (%d programs)", targetValue, i, concrete, len(concrete.Programs())) }
            for i, stemmed  := range stemmedList { info(at(ctx,stemmed.position), "%v : stemmed: %d. %v (%d programs)", targetValue, i, stemmed, len(stemmed.Programs())) }
            for i, t := range pc.traves { info(at(ctx, t.pos), "%v: %d. %v", s, i, t) }
            info(ctx, "%v: %v (%v, %v)\n", s, prereqStrval, prereqFile.fullname(), b)
            info(ctx, "%v: %v (%T)", s, prereqValue, prereqValue)
            warnstack(ctx, 3).debug(10)
        }

        pc.deferTrave(ctx, targetValue, prereqValue, prereqPattern, prereqFile)
        result = pc.traves

        if prereqFile != nil && !prereqFile.exists() {
            erro(of(ctx, prereqValue), "%v: missing %v %v", targetValue, prereqValue, prereqFile.fullname())
            for i, concrete := range concreteList { warn(at(ctx,concrete.Position()), "%v: concrete: %d. %v (%d programs)", targetValue, i, concrete, len(concrete.Programs())) }
            for i, stemmed  := range stemmedList  { warn(at(ctx,stemmed.position), "%v: stemmed: %d. %v", targetValue, i, stemmed) }
            for i, s := range pc.traves { warn(at(ctx,s.pos), "%v: %d. %v", targetValue, i, s) }
            errostack(ctx, 5, "%v, filemap=%v", projects, prereqFile.filemap).debug(10)
        }

        if d := time.Now().Sub(t0); d > 60*time.Second {
            if false {
                var ac = ctx.auto()
                for ac != nil {
                    if ac.autoGet("@") == targetValue {
                        warn(ctx, "%v", targetValue).debug(1)
                    }
                    ac = ac.Context.auto()
                }
            }
            for i, c := range concreteList { warn(at(ctx,c.Position()), "%v : C#%d %v", targetValue, i, c) }
            for i, s := range stemmedList  { warn(at(ctx,s.position), "%v : S#%d %v", targetValue, i, s) }
            warnstack(ctx, 5, "slow: %v: %v: %v, %v", targetValue, prereqValue, d, ctx.gap()).debug(10)
            ctx.flushDiags(true)
        }
    } (time.Now())

    if targetValue = getTargetValue(ctx); targetValue == nil {
        prompt(of(ctx,prereqValue), "%s: target is nil\n", prereqStrval)
        errostack(ctx, 3, "").debug(6)
        return
    } else if isTrivial(targetValue) {
        prompt(of(ctx,prereqValue), "%s: target is trivial (%T)\n", prereqStrval, targetValue)
        errostack(ctx, 3, "").debug(6)
        return
    }

    if len(projects) == 0 {
        erro(ctx, "%v: no projects to traverse (%s)", prereqValue, prereqStrval)
        erro(ctx, "%v: closure %v", prereqStrval, len(ctx.closureScopes()))
        errostack(ctx, 3, "").debug(8)
        return
    } else {
        prereqValue, prereqPattern, prereqStrval, prereqFile, prereqObj =
            probPrereqValue(ctx, projects, prereqValue)
    }

    // NOTE: Don't delete, keep it for future debugging.
    if false && ((
        strings.HasPrefix(prereqStrval, "") ||
            false) && (
        strings.Contains(prereqStrval, "/") ||
            false) && (
        strings.HasSuffix(prereqStrval, ".o") ||
            false)) {
        var s, _, _ = entryIndicator(ctx, targetValue)
        prompt(ctx, "%v : %T %v\n", s, prereqValue, prereqStrval)
        warn(ctx, "@: %T %v : %v", targetValue, targetValue, ctx.program().depends)
        if f := file(ctx, prereqStrval); f == nil {
            var p = ctx.Project()
            var a = ctx.universe().unmap(ctx, prereqStrval)
            var b = files(ctx, prereqStrval, p)
            warn(ctx, ">: %T %v -> file: %v", prereqValue, prereqValue, a)
            warn(ctx, ">: %T %v -> file: %v", prereqValue, prereqValue, b)
            warn(ctx, ">: %T %v -> file: %v", prereqValue, prereqValue, p.selectFiles(ctx, a))
            warn(ctx, ">: %T %v -> file: %v", prereqValue, prereqValue, p.selectFiles(ctx, b))
            warn(ctx, ">: %T %v -> file: %v", prereqValue, prereqValue, p.selectFile(ctx, a))
            warn(ctx, ">: %T %v -> file: %v", prereqValue, prereqValue, p.selectFile(ctx, b))
            for i, m := range a { warn(at(ctx, f.position), "%v: %d. %v: %v %v", p, i, m.project, m.name, m.pattern) }
            for i, m := range b { warn(at(ctx, f.position), "%v: %d. %v: %v %v", p, i, m.project, m.name, m.pattern) }
            if f, y := prereqValue.(*File); y {
                warn(at(ctx, f.position), "%v: %v", p, f.filestub)
                warn(at(ctx, f.position), "%v: %v %v", p, f.fullname(), f.exists())
            }
            warnstack(ctx, 5, "%v: %v", p, ctx.entry()).debug(10)
        } else {
            warn(ctx, ">: %T %v -> file: %v", prereqValue, prereqValue, f)
            warn(at(ctx, f.position), "%v", f.filestub)
            warn(at(ctx, f.position), "%v", f.fullname())
            if f, y := prereqValue.(*File); y {
                warn(at(ctx, f.position), "%v", f.filestub)
                warn(at(ctx, f.position), "%v", f.fullname())
            }
            warnstack(ctx, 5, "%v", ctx.entry()).debug(10)
        }
        db = true
    }

    if traverseCacheOn && traverseCache != nil && prereqFile != nil {
        var h maphash.Hash
        if false { h.WriteString(ctx.Project().absPath) }
        if false { h.WriteString(targetValue.Strval(ctx)) }
        if false { h.WriteString(prereqStrval) }
        if true  { h.WriteString(prereqFile.fullname()) }

        var hs = h.Sum64()
        if i, y := traverseCache[hs]; y && i > 0 {
            info(ctx, "traverse: %v: %v: %v, %d", targetValue, prereqValue, hs, i).debug(1)
            return
        }

        traverseCache[hs] += 1
    }

    // Recursion detection -- simply return to break it if looped.
    if traverseDetectLoops {
        if eq(ctx, targetValue, prereqValue) {
            prompt(ctx, "%v: %v: self dependency, consider using [(once)] to avoid\n", targetValue, prereqValue)
            warn(of(ctx,prereqValue), "recursion: %T %v", prereqValue, prereqValue)
            warn(of(ctx,targetValue), "recursion: %T %v", targetValue, targetValue)
            warn(ctx, "recursion: %v : %v ; in %v", targetValue, prereqFile, projects)
            if false {
                warnstack(ctx, 16, "").debug(32)
            } else {
                errostack(ctx, 16, "").debug(32)
            }
            return
        }
        for c := pc; c != nil; c = c.caller() {
            if val := autoGet(c, "@"); val != nil && eq(c, val, prereqValue) {
                if traverseLoopBreakState != traveUnkn {
                    var s = pc.traves.add(ctx, traverseLoopBreakState, targetValue)
                    if s.dependPat = prereqPattern; prereqFile == nil {
                        s.depend = prereqValue
                    } else {
                        s.depend = prereqFile
                    }
                }

                var f = as{targetValue}.file(ctx, projects...)
                if true && f == nil {
                    prompt(ctx, "%v: %v: recursion detected, consider using [(once)] to avoid\n", targetValue, prereqValue)
                    warn(ctx, "recursion: %T %v", prereqValue, prereqValue)//.of(prereqValue)
                    warn(ctx, "recursion: %T %v", targetValue, targetValue)//.of(targetValue)
                    warn(ctx, "recursion: %v : %v ; in %v", targetValue, prereqFile, projects)
                    warnstack(ctx, 3, "").debug(16)
                }
                return
            }
        }
    }

    var travedPrereqFile func (*travestate) *File
    if prereqFile != nil {
        if n := prereqFile.traversed; n > 0 {
            if false && n > 1 {
                warn(ctx, "traversed: %v: %v %T", targetValue, prereqValue, prereqValue)
                warn(ctx, "traversed: %v: %v", targetValue, prereqFile.fullname())
                warnstack(ctx, 3, "traversed: %d", prereqFile.traversed).debug(10)
            }
            return
        }

        travedPrereqFile = func (s *travestate) (res *File) { return }
    } else {
        // If the prereqValue is not a *File, for example a (*String) or (*Compound)
        // %.h <-> 'llvm/PassSupport.h' <-> [
        //   file@llvm/PassSupport.h>/Volumes/workspace/external/llvm-project/llvm/include/llvm/PassSupport.h
        //   file@llvm/PassSupport.h>/Volumes/workspace/external/llvm-project/llvm/include/llvm/PassSupport.h
        //   file@llvm/PassSupport.h>/Volumes/workspace/external/llvm-project/llvm/include/llvm/PassSupport.h
        //   done@llvm/PassSupport.h
        //   done@llvm/PassSupport.h
        // ]
        travedPrereqFile = func (s *travestate) (res *File) {
            // and the trave target is a *File with the name matched
            if f, y := toFile(s.target); y && f.name == prereqStrval { res = f }
            return
        }
    }

    type traveResT int
    const (
        traveResContinue traveResT = iota
        traveResBreak
        traveResReturn
    )
    var traverseEntry = func(project *Project, entry Entry, pattern bool) (result traveResT) {
        if false && db { defer func() { if result == traveResReturn {
            for i, s := range pc.traves { info(at(ctx,s.pos), "%v : %d. %v", entry, i, s) }
            warn(ctx, "%v, %T %v ; %v", entry, prereqValue, prereqValue, result).debug(6)
        }}()}

        pc.traves.add(ctx, traveRule, targetValue).depend = entry
        entry.traverse(ctx)

        var t = pc.traves
        if !t.has() { return traveResContinue }

        // NOTE: collect travestates from t according to each trave type

        if tt := t.of(traveFail); tt.has() {
            if !pattern || verb {
                var stems = ctx.stems()
                prompt(ctx, "%v: traverse entry failed (%v)\n", entry, project)
                warn(of(ctx,entry), "%v: %v: %v (stems=%v)", entry, targetValue, prereqValue, stems)
                for i, s := range pc.traves {
                    warn(at(ctx,s.pos), "%v: %v: %d. %v", targetValue, entry, i, s)
                }
                if n, m := 5, 16; len(stems) == 0 {
                    errostack(ctx, n, "#>", prereqValue).debug(m)
                } else if true {
                    warnstack(ctx, n, "#>", prereqValue).debug(m)
                }
            }
            for _, s := range tt {
                if g := s.target; g != nil && eq(ctx, prereqValue, g) && true {
                    warn(of(ctx,entry), "%v (%T) (by %v, in %v)", entry, entry.Target(), targetValue, entry.OwnerProject()).debug(1)
                    return traveResContinue
                }
            }
            return traveResReturn
        }

        if tt := t.of(traveCase); tt.has() {
            for _, s := range tt {
                if f := travedPrereqFile(s); f != nil {
                    prereqFile = f
                    return traveResReturn
                }
                if g := s.target; g != nil && eq(ctx, prereqValue, g) && true {
                    return traveResReturn
                }
            }
        }

        if tt := t.of(traveDone); tt.has() {
            for _, s := range tt {
                if f := travedPrereqFile(s); f != nil {
                    prereqFile = f
                    return traveResReturn
                }
                if g := s.target; g != nil && eq(ctx, prereqValue, g) {
                    if prereqFile != nil && prereqFile.exists() {
                        return traveResReturn
                    } else if prereqFile != nil {
                        var a = pc.traves
                        for i, s := range a { info(at(ctx,s.pos), "a: %v : %d. %v", entry, i, s) }
                        for i, s := range t { info(at(ctx,s.pos), "t: %v : %d. %v", entry, i, s) }
                        erro(at(ctx,s.pos), "%v: missing file %v", targetValue, prereqValue)
                        erro(at(ctx,s.pos), "%v: %v", targetValue, prereqFile.fullname())
                        erro(at(ctx,s.pos), "%v: %v", targetValue, entry)
                        errostack(at(ctx,s.pos), 3, "").debug(10)
                    } else if true && db {
                        info(at(ctx,s.pos), "%v %v %v %v, %v %v",
                            targetValue, entry, prereqValue, prereqFile, g, s).debug(10)
                    }
                }
            }
        }

        // NOTE: foo foo.o [next@foo.o>foo.c]
        // NOTE: foo.pdf foo.tex [next@foo.tex:%%.org>foo.org]
        if tt := t.of(traveNext); tt.has() {
            var stemmedThisTarget bool
            if s := ctx.stemmed(); s != nil && s.target != nil {
                stemmedThisTarget = eq(ctx, s.target, targetValue)
            }
            for _, s := range tt {
                if g := s.target; g == nil {
                    prompt(ctx, "%v: %v\n", targetValue.Strval(ctx), t)
                    erro(ctx, "%v: %v %v %v\n", targetValue.Strval(ctx), prereqPattern, prereqValue, pattern)
                    errostack(ctx, 5, "").debug(1)
                } else if true && eq(ctx, targetValue, g) {
                    if /*pattern && */stemmedThisTarget {
                        // pc.traves = append(pc.traves, s) // collect the state
                    }
                    return traveResContinue // try the next pattern
                } else if true && eq(ctx, prereqValue, g) {
                    if pattern && stemmedThisTarget {
                        // IMPORTANT NOTE: traveNext state should be remained
                        //   This traveNext state muse be returned, so that the
                        //   (*Program).traverse func can break it's loop properly.
                        // pc.traves = append(pc.traves, s) // collect the state
                    }
                    return traveResContinue // try the next pattern
                }
            }
        }

        // NOTE: *smart.File memory.o <-> [file@memory.o>memory.cpp file@memory.o>...]
        if tt := t.of(traveFile); tt.has() {
            for _, s := range tt {
                if f := travedPrereqFile(s); f != nil {
                    prereqFile = f
                    return traveResReturn // file processed
                } else if g := s.target; g != nil && eq(ctx, prereqValue, g) && true {
                    var op = traveResReturn // assuming file processed
                    if !entry.hasRecipes() { op = traveResContinue }
                    return op
                } else if _, ok := prereqValue.(*String); false && ok {
                    if prereqFile == nil { prompt(ctx, "nonfile: %T %v: %T %v : %T %v ; %v\n",
                        targetValue, targetValue, prereqValue, prereqValue, s.target, s.target, s).debug(1) }
                }
            }
        }

        return
    }

    t1 = time.Now()

ForProjectsConcretes:
    for _, project := range projects {
        var entries = project.resolveEntries(ctx, prereqStrval, pc.grepping, false)
        if entries != nil && len(entries.all) > 0 {
            concreteList = append(concreteList, entries.all...)
        } else {
            continue ForProjectsConcretes
        }
    ForEntries:
        for _, entry := range entries.all {
            if !isNil(entry) && targetValue == entry { continue ForEntries }
            if w, k := targetValue.(*bareword); k && w.string == prereqStrval {
                continue ForEntries // target resolve to itself, does nothing
            }
            switch traverseEntry(project, entry, false) {
            case traveResBreak : break ForEntries
            case traveResReturn: return //if ok { return } else { break ForEntries }
            }
        }
    }
    if d := time.Now().Sub(t1); d > 60*time.Second {
        for _, concrete := range concreteList { prompt(ctx, "%v: slow: %v", concrete.Position(), concrete) }
        prompt(ctx, "%v: slow: %v: %v %v (%d concretes)", ctx.Position(), targetValue, prereqValue, d, len(concreteList)).debug(1)
    }

    if prereqFile != nil && prereqFile.exists() { goto CheckPrereqResult }

    t2 = time.Now()

ForProjectsPatterns:
    for _, project := range projects {
        var patterns = project.resolvePatterns(ctx, prereqValue, prereqStrval)
        if len(patterns) > 0 {
            stemmedList = append(stemmedList, patterns...)
        } else {
            continue ForProjectsPatterns
        }
    ForPatterns:
        for _, entry := range patterns {
            switch traverseEntry(project, entry, true) {
            case traveResBreak : break ForPatterns
            case traveResReturn: return //if ok { return } else { break ForPatterns }
            }
        }
    }
    if d := time.Now().Sub(t2); d > 60*time.Second {
        for _, stemmed  := range stemmedList { prompt(ctx, "%v: slow: %v", stemmed.position, stemmed) }
        prompt(ctx, "%v: slow: %v: %v %v (%d stemmed)", ctx.Position(), targetValue, prereqValue, d, len(stemmedList)).debug(1)
    }

CheckPrereqResult:
    if false && prereqFile == nil { prereqFile, _ = toFile(prereqValue) }

    var p = prereqFile
    if p == nil {/* fallthrough */} else
    if p.exists() {
        trave := pc.traves.add(ctx, traveFile, targetValue)
        trave.dependPat = prereqPattern
        trave.depend = p
        return
    } else if m := p.filemap; m != nil {
        if p.info == nil { p.stat(ctx) }
        if true && p.info == nil && p.name == "tablegen-min" {
            var f = m.stat(ctx, p.name)
            var pos = ctx.Position()
            for _, loc := range m.locs { prompt(ctx, "%v: %v ⇒ %v\n", pos, m, loc) }
            prompt(ctx, "%v: {%v %v %v}\n", pos, p.dir, p.sub, p.name)
            prompt(ctx, "%v: {%v %v %v}\n", pos, f.dir, f.sub, f.name)
            prompt(ctx, "%v: %v, patts=%v\n", pos, m.project, m.patts).debug(1)
        }
        if p.info != nil {
            assert(p.exists(), "file must exists at this point")
            trave := pc.traves.add(ctx, traveFile, targetValue)
            trave.dependPat = prereqPattern
            trave.depend = p
            return
        }
    }

    if p != nil && p.exists() {
        if !pc.traves.has(traveFail) { return }
        if !pc.traves.has() { return }
    }
    if p == nil && prereqObj == nil && !pc.traves.has() {
        if p = stat(ctx, prereqStrval, "", ""); p != nil {
            trave := pc.traves.add(ctx, traveFile, targetValue)
            trave.dependPat = prereqPattern
            trave.depend = p
            prereqValue = p
            return
        }
    }

    if pc.traves.has(traveDone) { return }

    if t := pc.traves.of(traveRule); t.has() && t[0].depend != nil {
        if _, y := t[0].depend.(*stemmed); y {
            s := pc.traves.add(ctx, traveNext, targetValue)
            s.dependPat = prereqPattern
            s.depend = prereqValue
            return
        }
        if t[0].depend.(Entry).Name(ctx) == prereqStrval { return }
    }

    if t := pc.traves.of(traveObj); t.has() && t[0].depend != nil {
        if t[0].depend.(Object).Name(ctx) == prereqStrval { return }
    }

    if prereqPattern != nil {
        s := pc.traves.add(ctx, traveNext, targetValue)
        s.dependPat = prereqPattern
        s.depend = prereqValue
        return
    }

    if ctx.isConfiguration() { return }

    if len(ctx.stems()) == 0 || ctx.mustExists() {
        if s := pc.traves.add(ctx, traveFail, targetValue); prereqFile != nil {
            s.error = fileNotFoundError{ctx.Project(), prereqFile}
            ctx = at(ctx, prereqFile.position)
        } else {
            s.error = targetNotFoundError{ctx.Project(), prereqStrval}
            if prereqValue != nil { ctx = at(ctx, prereqValue.Position()) }
        }

        if prereqFile != nil && prereqValue != prereqFile {
            prompt(ctx, "%v:(%T): %T %v; file=%v projects=%v\n", targetValue, targetValue, prereqValue, prereqValue, prereqFile, projects).debug(1)
        } else if prereqFile != nil {
            prompt(ctx, "%v:(%T): %T %v; path=%s projects=%v\n", targetValue, targetValue, prereqValue, prereqValue, prereqFile.fullname(), projects).debug(1)
        } else if prereqObj != nil {
            prompt(ctx, "%v:(%T): %T %v; obj=%v projects=%v\n", targetValue, targetValue, prereqValue, prereqValue, prereqObj, projects).debug(1)
        } else {
            prompt(ctx, "%v:(%T): %T %v; projects=%v\n", targetValue, targetValue, prereqValue, prereqValue, projects).debug(1)
        }

        if fil := prereqFile;  fil != nil { erro(at(ctx,fil.Position()), "file: %v", fil) }
        if obj := prereqObj;   obj != nil { erro(at(ctx,obj.Position()), "object: %T %v", obj, obj) }
        if val := prereqValue; val != nil { erro(at(ctx,val.Position()), "value: %T %v", val, val) }
        for i, s := range pc.traves { erro(at(ctx,s.pos), "%d. %v: %v: %v", i, targetValue, prereqValue, s) }
        for i, c := range ctx.closureScopes() { erro(ctx, "%d. closure %v", i, c) }
        for i, concrete := range concreteList { erro(at(ctx,concrete.Position()), "concrete: %d. %v (%d programs)", i, concrete, len(concrete.Programs())) }
        for i, stemmed  := range stemmedList  { erro(at(ctx,stemmed.position), "stemmed: %d. %v", i, stemmed) }
        errostack(of(ctx, prereqValue), 6, "").debug(512)
        return
    } else {
        return // no operation
    }
}

type Program struct {
    position Position
    project *Project
    scope   *Scope
    _env     []*Pair
    params   []*def
    depends  []Value // normal
    ordered  []Value // order-only
    recipes  []Value
    defaultVal Value
    language  string
    changedWD string
    configure bool
}
func (prog *Program) Position() Position { return prog.position }
func (prog *Program) Project() *Project { return prog.project }
func (prog *Program) Scope() *Scope { return prog.scope }
func (prog *Program) interpret(ctx Context, i interpreter, params []Value) (err error) {
    if pos := ctx.Position(); !pos.IsValid() && prog.position.IsValid() {
        ctx = at(ctx, prog.position)
    }

    var target Value
    if target, _, _, err = wait(ctx); err != nil { // wait for prerequisites
        erro(ctx, "waiting traversal failed: %v", err).debug(1)
        return
    }

    var pc = ctx.programContext()
    if f, y := target.(*File); y && pc != nil && !ctx.dirty(ctx) {
        pc.traves.add(ctx, traveDone, nil) // NOTE: modifier.predictDirty
        if false { info(ctx, "interpret: %v %s", f, f.fullname()) }
        return
    }

    var value Value
    if value, err = i.Evaluate(ctx, params...); err != nil {
        var (
            _, ent, _ = entryIndicator(ctx, ctx.entry())
            nam = intername(i)
        )
        prompt(ctx, "%v: interpret '%s' recipes failed\n", ent, nam)
        erro(ctx, "%s: %v", nam, err)
        errostack(ctx, 3, "%v", ctx).debug(1)
        return
    } else if value == nil {
        // disgard nil value
    } else if def, prev := ctx.autoSet("-", value); def == nil {
        var (
            _, ent, _ = entryIndicator(ctx, ctx.entry())
            nam = intername(i)
        )
        prompt(ctx, "%v: %s\n", ent, nam)
        erro(ctx, "set buffer value failed: %v -> %v", prev, value)
        errostack(ctx, 3, "%v", ctx).debug(1)
        return
    }

    if _, _, err = updateRecipesHash(ctx, target); err != nil {
        var (
            _, ent, _ = entryIndicator(ctx, ctx.entry())
            nam = intername(i)
        )
        prompt(ctx, "%v: %s\n", ent, nam)
        erro(ctx, "update recipes hash failed: %v", err)
        errostack(ctx, 3, "%v", ctx).debug(1)
    } else if pc != nil {
        pc.interpreted = append(pc.interpreted, i)
    }
    return
}

func (prog *Program) getModifiers(ctx Context, name string) (ms []*modification) {
    for _, d := range prog.depends {
        var g, ok = d.(*modifications)
        if !ok { continue }
        for _, m := range g.list {
            if m.name.Strval(ctx) == name {
                ms = append(ms, m)
            }
        }
    }
    return
}

const maxCallRecursion  = 32 //64

type normalTraverseContext struct { Context }
type orderTraverseContext struct { Context }
func (t normalTraverseContext) traversed(target Value) (targets []Value) {
    if targets = t.Context.traversed(target); len(targets) > 0 {
        t.autoSet("^", MakeList(t.Position(), targets...))
        t.autoSet("<", targets[0])
        t.autoSet(">", targets[len(targets)-1])
        var c = at(t.Context, target.Position())
        if false && strings.HasSuffix(target.Strval(c), "patchlevel.c") {
            warn(c, "%T %p %v\n", t.Context, t.Context, t.autoGet("^")).debug(32)
        }
    }
    return
}
func (t orderTraverseContext) traversed(target Value) (targets []Value) {
    if targets = t.Context.traversed(target); len(targets) > 0 {
        t.autoSet("|", MakeList(t.Position(), targets...))
    }
    return
}

func (prog *Program) workDir(ctx Context) (workDir string) {
    if prog.changedWD == "" {
        var o Object
        if _, o = prog.scope.Find("CWD"); isTrivial(o) {
            if _, o = prog.scope.Find("/"); isTrivial(o) {
                erro(ctx, "both $(CWD) and $/ are trivial").debug(1)
                return
            }
        }
        if d, y := o.(*def); y {
            if v := d.Call(ctx, nil); !isTrivial(v) {
                workDir = v.Strval(ctx)
            } else {
                errostack(ctx, 3, "trivial %v: %v", d.origin, d).debug(32)
            }
        }
    } else if filepath.IsAbs(prog.changedWD) {
        workDir = prog.changedWD
    } else {
        workDir = filepath.Join(prog.project.absPath, prog.changedWD)
    }
    return
}

func (prog *Program) env(ctx Context) (env []string, osi int) {
    env = os.Environ()
    osi = len(env)
    for _, p := range prog._env {
        var k, v = p.Key.Strval(ctx), p.Value.Strval(ctx)
        env = append(env, fmt.Sprintf("%s=%s", k, v)) // strconv.Quote(v)
    }
    return
}

func (prog *Program) execute(cc Context) (result Value, _traves travestates) {
    var (
        ctx Context = cc
        entry = cc.entry()
        args  = cc.arguments()
        pos   = cc.Position()
    )
    if cc != nil && cc.flushDiags(true) > 0 {
        var errs = cc.totalErrors()
        var s string; if errs > 1 { s = "s" }
        prompt(cc, "%v: canceled execution (%d error%s), project %s\n", entry, errs,s, prog.project)
        warn(cc, `cancel "%v"`, entry)
        warnstack(cc, 5, `%v`, cc).debug(16)
        if options.failOnErrors { fail(pos, "fail by %d error%s", errs, s) }
        return
    }

    var pc = programContext{
        autoContext: autoContext{ Context:cc, defs:make(autoDefMap) },
        execRec: make(map[Value]int),
        start: time.Now(),
        print: true,
        prog: prog,
    }

    defer func() {
        var (
            targets = autoGet(ctx, "@")
            // depends = autoGet(ctx, "^")
            // tb = traves.not(traveDone, traveNext)
        )
        if isTrivial(targets) { targets = entry.Target() }
        if ctx.flushDiags(true) > 0 {
            var (
                str, ent, tar = entryIndicator(ctx, entry)
                errs = ctx.totalErrors()
            )
            if !ctx.isConfiguration() && cc != nil {
                s := pc.traves.add(ctx, traveFail, targets)
                if errs == 1 {
                    s.error = fmt.Errorf("execution yields an error for %v", str)
                } else {
                    s.error = fmt.Errorf("execution yields %d errors for %v", errs, str)
                }
            }
            if tar != "" && tar != ent {
                erro(ctx, "%s: %s: execution yields %d errors", ent, tar, errs)
            } else {
                erro(ctx, "%s: execution yields %d errors", ent, errs)
            }
            errostack(ctx, 8, "").debug(32)
            if options.failOnErrors { fail(prog.position, "fail by %d errors", errs) }
        }

        _traves = pc.traves
    } ()

    ctx = &pc

    assert(prog.project == prog.scope.project, "mismatched scope/project")
    if options.verbose { info(ctx, "%v: %v", entry, args).debug(1) }
    if cc != nil {
        var depth, loop int = 0, -1
        var a = []Value{ autoGet(cc, "@") }

    ForPC:
        for c := cc.programContext(); c != nil; c = c.caller() {
            if c.program() == prog {
                if depth += 1; depth == maxCallRecursion { break ForPC }
                var t = autoGet(c, "@")
                if /* 1 < depth */true {
                    for i, v := range a { if eq(cc, t, v) { loop = i; break ForPC } }
                }
                if loop < 0 { a = append(a, t) }
            }
        }

        if /* 1 < depth && */ 0 <= loop {
            var t = autoGet(cc, "@")
            if o := cc.closure(); o != nil {
                if v := autoGet(o, "@"); v != nil && eq(cc, v, t) {
                    if true { warnstack(ctx, 3, "skip closure loop: %v %v", o, t).debug(1) }
                    // FIXES: skip execution as it's closure, for example:
                    //
                    //   %.h($(headers)): $(srcinc)/%.h update-file
                    //
                    // where the 'update-file' is like:
                    //
                    //   update-file: [((in)) (closure) (set @=&@)] $(in) \
                    //       [(read-file $>) (update-file -p)]
                    //
                    // see also Rule.traverse for the same skip.
                    return
                }
            }

            prompt(ctx, "%v: %v: %v, %v\n", a[0], autoGet(cc.closure(), "@"), cc, cc.closure())
            for i, t := range a { erro(of(ctx,t), "loop: %v: %v", i, t) }
            errostack(at(ctx,prog.position), 128, "loop, (depth=%d, %v, %v)\n", depth, a[loop], a).debug(6)
            return
        }

        if depth < maxCallRecursion {
            // continues
        } else if c := cc.programContext(); c != nil {
            if /*options.traceTraversalNestIndent*/true { pc.traceLevel = c.traceLevel }

            var tt = as{autoGet(c, "@")}
            var s, _ = tt.fullnameOrStrval(ctx)
            prompt(ctx, "%v: max recursion call (%d)\n", s, depth)
            warn(of(ctx,tt), "max recursion call (%d)\n", depth).debug(1)

            const collapse = false
            for ; c != nil; c = c.caller() {
                var n int
                if collapse { for next := c.caller(); next != nil; next = next.caller() {
                    if d := autoGet(next, "@"); d == nil { continue } else
                    if t := d; t != nil && eq(ctx, t, tt) {
                        n += 1;  continue
                    }
                    if next.program() == c.program() { n += 1; c = next } else { break }
                }}

                var t = autoGet(c, "@")
                if prog := c.program(); prog == nil {
                    erro(at(ctx,entry.Position()), "%v (@=%v)", entry, tt)
                    break
                } else if pos := prog.position; n > 0 {
                    erro(at(ctx,pos), "%v (repeated %d times)", t, n)
                } else if !collapse {
                    erro(at(ctx,pos), "%v : %v", t, autoGet(c, ">"))
                } else if depth -= 1; maxCallRecursion - depth > 5 {
                    erro(at(ctx,pos), "%v ... (%d)", t, maxCallRecursion - depth)
                    break
                } else {
                    erro(at(ctx,pos), "%v : %v", t, autoGet(c, ">"))
                }

                ctx.flushDiags(true) // dump immediately
            }
            errostack(ctx, depth, "#>", entry).debug(512)
            if false { fail(prog.position, "max call depth") }
            return
        }

        if stems := cc.stems(); stems != nil {
            ctx.autoSet("*", MakeString(pos, stems[0]))
        }
    }

    var alreadyUpdated bool

    // NOTE: set "@" before autoArgs
    // Select the right target value before setting parameters,
    // because the target could be overrided by parameters.
    if target := entry.Target(); target == nil {
        erro(ctx, "%v: nil entry target", target)
        errostack(ctx, 8, "").debug(20)
        return
    } else {
        switch a := target.(type) {
        case *Flag: pc.print = false // Flag target (-foo) turns off printing automatically
        case *File: // alreadyUpdated = a.info != nil && a.updated
        case *String, *Compound: // NOTE: escape 'String' and "Compound" values from file searching
        default:
            if file := prog.project.file(ctx, a.Strval(ctx)); file != nil {
                // alreadyUpdated = file.info != nil && file.updated
                target = file
            }
        }

        if pc.execRec[target] += 1; false { if pc.execRec[target] > 1 {
            if options.traceExec { t_exec.trace(fmt.Sprintf("exec: %v", target)) }
            return
        }}

        if options.verbose { info(ctx, "%v: %v", target, args).debug(1) }

        ctx.autoSet("@", target)
    }

    var err error
    if pc.params, err = pc.autoArgs(prog.params, args); err != nil {
        erro(ctx, "auto args failed: %v", err).debug(1)
        return
    }

    // Note: must enter work directory (cd) before setting cloctx
    var enterBack *enterec
    if len(cd.stack) > 0 { enterBack = cd.stack[0] }
    if err = enter(ctx, prog.project.absPath); err != nil {
        erro(ctx, "enter project '%v' failed: %v", prog.project, err).debug(1)
        return
    }

    defer func(swd string) {
        if e := leave(ctx, prog, enterBack); e != nil {
            // NOTE: err could be traveCase, traveDone, etc.
            if err == nil { err = e } else {
                erro(ctx, "leave project '%v' failed: %v", prog.project, err).debug(1)
            }
        }
        if prog.project.changedWD = swd; err != nil {
            erro(ctx, "execution failed: %v", err).debug(6)
            return
        }

        var defaultVal = prog.defaultVal
        prog.defaultVal = nil

        if /*!isNil(result) && !isNone(result)*/!isTrivial(result) {
            // good!
        } else if d := autoGet(ctx, "-"); d != nil {
            result = d
        } else if !isNil(defaultVal) {
            result = defaultVal
        }

        if cc != nil { if p := cc.program(); p != nil && !isNil(result) {
            p.defaultVal = result
        }}
    } (prog.project.changedWD)

    if alreadyUpdated {
        if options.verbose {
            info(ctx, "'%v' already updated", autoGet(ctx, "@"))
        }
        if false { return }
    }

    // if pc.print && entry.Class() == UseRule { pc.print = false }
    if pc.print && prog.configure { pc.print = false }
    cd.stack[0].silent = !pc.print

    if options.traceExec {
        var d = pc.depth()
        var t = autoGet(ctx, "@")
        var s = fmt.Sprintf("%s: %v (%p, exec.depth=%d)", typeof(t), t, t, d)
        defer un(trace(t_exec, s))
    }

    var proj = ctx.Project()
    ctx.autoSet("^", nil)
    ctx.autoSet("<", nil)
    ctx.autoSet(">", nil)
    ctx.autoSet("|", nil)

    // Update normal prerequisites
    prog.traverse(normalTraverseContext{ctx}, prog.depends)
    if errs := ctx.flushDiags(true); errs > 0 {
        s := pc.traves.add(at(ctx, prog.position), traveFail, nil)
        s.error = fmt.Errorf("%d errors counted", errs)
        prompt(ctx, "%v: execute failed, project %s; traves=%v\n", entry, proj, pc.traves).debug(1)
        warn(ctx, "%d errors while traversing prerequisites for %v", errs, autoGet(ctx,"@"))
        if warnstack(ctx, 6, "").debug(8); true && options.failOnErrors {
            fail(prog.position, "fail by %d errors", ctx.totalErrors())
        }
        return
    } else if pc.traves.has(traveCase, traveDone, traveFail, traveNext) {
        return
    }

    // Update order-only prerequisites
    prog.traverse(orderTraverseContext{ctx}, prog.ordered)
    if errs := ctx.flushDiags(true); errs > 0 {
        s := pc.traves.add(at(ctx, prog.position), traveFail, nil)
        s.error = fmt.Errorf("%d errors counted", errs)
        prompt(ctx, "%v: execute failed, project %s; traves=%v\n", entry, proj, pc.traves).debug(1)
        warn(ctx, "%d errors while traversing prerequisites for %v", errs, autoGet(ctx,"@"))
        if warnstack(ctx, 6, "").debug(8); true && options.failOnErrors {
            fail(prog.position, "fail by %d errors", ctx.totalErrors())
        }
        return
    } else if pc.traves.has(traveCase, traveDone, traveFail, traveNext) {
        return
    }

    if prog.language != "" || len(prog.recipes) == 0 {
        // does nothing
    } else if d := autoGet(ctx,"-"); d == nil || len(pc.interpreted) == 0 {
        // Using the default statements interpreter (aka. evaluation).
        if i, ok := dialects["eval"]; ok && i != nil {
            if err := prog.interpret(ctx, i, nil); err != nil {
                erro(ctx, "%v", err).debug(1)
            }
        } else {
            erro(ctx, "no default dialect").debug(1)
        }
    }
    return
}

func (prog *Program) traverse(ctx Context, prerequisites []Value) {
    var (
        verb  = options.verbose || options.verboseBreaks
        stemd = ctx.stemmed()
        stems = ctx.stems()
        pc    = ctx.programContext()
        ent   = autoGet(ctx, "@")
        depends valvec
        by Value
    )
    defer func() {
        pc.Wait()

        var target Value = autoGet(ctx, "@")

        // NOTE: warnings if travestates go too large
        if true {
            // does nothing
        } else if tt := pc.traves.of(traveObj, traveRule, traveFile); len(tt) > 100 {
            prompt(ctx, "%v: traves=%d\n", target, len(pc.traves))
            for i, s := range pc.traves {
                prompt(ctx, "%v: %d. %v\n", target, i, s)
                if i > 10 { break }
            }
            warnstack(ctx, 5, "%d errors", ctx.flushDiags(true)).debug(16)
        }

        if f, y := target.(*File); false && y && !f.exists() {
            for i, s := range pc.traves {
                if s.what == traveFile {
                    var d = s.depend.(*File)
                    prompt(ctx, "%v: %d. %v ; %v, exists=%v\n", target, i, s, d, d.exists())
                } else {
                    prompt(ctx, "%v: %d. %v\n", target, i, s)
                }
            }
            erro(of(ctx, by), "%v: at %v", f.name, by)
            erro(ctx, "%v: target not exists (%s, %v)", f.name, f.fullname(), f.stat(ctx))
            errostack(ctx, 5).debug(16)
        }

        // FIXME: optimization: the pc.traves may grow into large number of traveFile
    } ()

ForPrerequisites:
    for _, prerequisite := range prerequisites {
        var (
            pos = prerequisite.Position()
            ctx = at(ctx, pos)
            k = len(pc.traves)
            t0 = time.Now()
        )
        switch by = prerequisite; u := prerequisite.(type) {
        case unexpanded : warn(ctx, "%v: unexpanded %v" , ent, u.Value).debug(1) ; continue
        case untraversed: warn(ctx, "%v: untraversed %v", ent, u.Value).debug(1) ; continue
        default: prerequisite.traverse(ctx)
        }

        var traves = pc.traves.slice(k)
        var target = autoGet(ctx, "@") // fetch updated $@
        var depend = autoGet(ctx, ">") // fetch updated $>
        var isPatternStemmedForTarget = stemd != nil && stemd.target != nil &&
            eq(ctx, stemd.target, target)

        if depend != nil { depends.add(depend) }

        if d := time.Now().Sub(t0); d > 30*time.Second {
            var p = prerequisite
            var a = target
            var t = autoGet(ctx, "^")
            var c = autoGet(ctx, "<")
            var q = depend
            prompt(ctx, "%v: Program.traverse %v: @: %v\n", pos, p, a)
            prompt(ctx, "%v: Program.traverse %v: ^: %v\n", pos, p, t)
            prompt(ctx, "%v: Program.traverse %v: <: %v\n", pos, p, c)
            prompt(ctx, "%v: Program.traverse %v: >: %v\n", pos, p, q)
            prompt(ctx, "%v: Program.traverse %v\n", pos, d).debug(1)
        }

        if false && !prog.configure {
            prompt(ctx, "%v: %s %v ; %s %v\n", target, typeof(prerequisite), prerequisite, typeof(depend), depend)
            for _, s := range traves {
                if s.what == traveFile {
                    var f = s.depend.(*File)
                    prompt(ctx, "%v: %v: %v ; exists=%v\n", target, prerequisite, s, f.exists())
                } else {
                    prompt(ctx, "%v: %v: %v\n", target, prerequisite, s)
                }
            }
            if false { prompt(ctx, "%v: %v\n", target, prog.configure).debug(6) }
            if true { infostack(ctx, 3, "%v, configure=%v", target, prog.configure).debug(16) }
        }

        if pc.debug_traverse > 0 { if true { pc.debug_traverse -= 1 }
            for i, a := range pc.traves { info(of(ctx,prerequisite), "%v: %d. %v %v", ent, i, a.what, a) }
            warnstack(of(ctx,prerequisite), 12, "%v: %v: %v %v", prog.project, ent, prerequisite, autoGet(ctx, ">")).debug(10)
        }

        if tt := traves.of(traveFail); tt.has() {
            if /* isPatternStemmedForTarget */true {
                for _, s := range tt {
                    var dependMine = depends.contains(s.depend)
                    if s.error == traveTargetNotDefinedFile && dependMine {
                        // add traveNext to try the next pattern
                        pc.traves.remove(s).add(ctx, traveNext, target)
                    } else if depend == nil {
                        prompt(ctx, "%v: %v\n", target, s).debug(1)
                        erro(at(ctx,s.pos), "%v", s).debug(1)
                        erro(of(ctx,target), "1. %T %v %s", target, target, target.Strval(ctx))
                        erro(of(ctx,s.depend), "2. %T %v mine=%v", s.depend, s.depend, dependMine)
                        erro(of(ctx,prerequisite), "3. %T %v", prerequisite, prerequisite)
                        errostack(ctx, 5, "#>").debug(10)
                    } else {
                        prompt(ctx, "%v: %v: %v\n", target, depend, s).debug(1)
                        erro(at(ctx,s.pos), "%v", s).debug(1)
                        erro(of(ctx,target), "1. %T %v %s", target, target, target.Strval(ctx))
                        erro(of(ctx,depend), "2. %T %v %s", depend, depend, depend.Strval(ctx))
                        if s.depend == nil { erro(ctx, "3. mine=%v", dependMine) } else {
                            erro(of(ctx,s.depend), "3. %T %v mine=%v", s.depend, s.depend, dependMine)
                        }
                        erro(of(ctx,prerequisite), "4. %T %v", prerequisite, prerequisite)
                        errostack(ctx, 5, "#>").debug(10)
                    }
                    return
                }
            }

            if verb {
                var p = ctx.Project()
                prompt(ctx, "%v:(%T): %T %v ; project=%s, stems=%v ; %v\n",
                    pc.traves, target, prerequisite, prerequisite, p, stems, stemd)

                var a = []interface{}{ "#>" }
                for _, s := range tt {
                    if pe, ok := s.error.(*fs.PathError); ok { // NOTE: pe.Path == s.target
                        warn(at(ctx,s.pos), "%v: %v: %v", pc.traves, s.target, pe.Err)
                    } else {
                        warn(at(ctx,s.pos), "%v: %v: %v (%T)", pc.traves, s.target, s.error, s.error)
                    }
                    a = append(a, s.target) // if !s.target.Position().Same(&s.pos)
                }
                warnstack(ctx, 5, a...).debug(16)
            }
            return // fail
        }

        if tt := traves.my(ctx, target, traveNext); tt.has() {
            var deps []Value
            for _, s := range tt {
                deps = append(deps, s.depend)

                if s.dependPat != nil && eq(ctx, s.depend, depend) {
                    if false { info(of(ctx,prerequisite), "%v: %T %v ; %v %v",
                        target, prerequisite, prerequisite, s.dependPat, s.depend).debug(1) }
                    return // end this pattern entry to let trying next one
                }
            }

            var _, isPP = prerequisite.(*PercPattern)
            if isPP && isPatternStemmedForTarget && len(deps) == 1 {
                if deps[0] == nil           { return } // %.h : %.h.cmake configure-file($>,$@)
                if eq(ctx, depend, deps[0]) { return } // %.o : %.cpp
            } else {
                continue ForPrerequisites
            }
        }

        if tt := traves.my(ctx, target, traveCase, traveDone); tt.has() {
            return
        }

        if tt := traves.my(ctx, target, traveFile, traveRule, traveObj); tt.has() {
            continue ForPrerequisites
        }

        if tt := traves.not(traveCase, traveDone, traveNext); tt.has() {
            var str string
            if eq(ctx, ent, target) {
                str = ent.String()
            } else {
                str = fmt.Sprintf("%v(%v)", ent, target)
            }
            prompt(ctx, "%s: %v ; traves=%d\n", str, prerequisite, len(pc.traves))
            for i, s := range pc.traves { prompt(ctx, "%s: %d. %v\n", str, i, s) }
            errostack(ctx, 5, "").debug(16)
            return
        }
    }
    return
}
