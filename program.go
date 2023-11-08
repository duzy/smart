//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "path/filepath"
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

    pats []Value
}
type programContext struct {
    autoContext
    sync.Mutex
    sync.WaitGroup

    by dirtyOpts

    prog *program
    projs []*Project
    params []string // $0, $1, $2, ...
    defaultVal Value
    values []Value
    defers []Value

    _env []*pair
    changedWD string

    start time.Time // start time
    dirt string // reason of outdated

    execRec map[Value]int

    calleeErrs []error
    calleeErrsM sync.Mutex

    targets []Value // all targets def
    grepped []Value
    grepping bool

    countFiles int

    traceLevel int
    traves travestates

    interpreted []interpreter

    debug_traverse int

    print bool // printing work directories (Entering/Leaving)
}
func (pc *programContext) inner() Context { return &pc.autoContext }
func (pc *programContext) caller() *programContext { return pc.Context.pc() }
func (pc *programContext) aquireLock() func() { pc.Lock() ; return func(){ pc.Unlock() }}
//XXX: func (pc *programContext) stems() []string { return nil }
func (pc *programContext) String() string {
    if fullContextStringer {
        var s = strings.TrimPrefix(pc.prog.scope.comment, "rule ")
        return fmt.Sprintf("program{%s,%s}", s, pc.autoContext.String())
    } else {
        return pc.autoContext.String()
    }
}

func (pc *programContext) initializeArgs() {
    if a := pc.Context.argumented(); a != nil {
        pc.params = pc.args(pc.Context, pc.prog.params, a.args)
    }
}

// func (pc *programContext) spawn(ctx Context) Context {
//     return &traverseContext{
//         Context: pc.Context.spawn(ctx),
//         print:   pc.print,
//         execRec: make(map[Value]int),
//         start:   time.Now(),
//     }
// }

func (pc *programContext) argumented() *argumentedContext { return nil }
func (pc *programContext) depth() (res int) {
    for c := pc.caller(); c != nil; c = c.caller() { res += 1 }
    return
}
func (pc *programContext) calleeError(err error) {
    if err != nil {
        pc.calleeErrsM.Lock()
        pc.calleeErrs = append(pc.calleeErrs, err)
        pc.calleeErrsM.Unlock()
    }
}

// traverseContext is a single thread traverse context, for traversing in a new goroutine,
// a spawned traversal must be used and then merge.
func (pc *programContext) level(n int) { pc.traceLevel += n }
func (pc *programContext) trace(a ...interface{}) { printIndentDots(pc.traceLevel, a...) }
func (pc *programContext) tracef(s string, a ...interface{}) { printIndentDots(pc.traceLevel, fmt.Sprintf(s, a...)) }
func (pc *programContext) traversed(ctx Context, target Value) []Value {
    if !isTrivial(target) {
        pc.targets = append(pc.targets, target)

        if false { if cc, y := pc.Context.(*closurecontext); y {
            pc.targets = cc.traversed(ctx, target)
        } }

        if _, y := toFile(target); y { pc.addFilesCount(1) }
    }
    return pc.targets
}

func (pc *programContext) addFilesCount(n int) {
    pc.countFiles += n
    if c := pc.caller(); c != nil { c.addFilesCount(1) }
}

func (pc *programContext) env(ctx Context) (env []string, osi int) {
    env = os.Environ()
    osi = len(env)
    for _, p := range pc._env {
        var k, v = p.Key.string(ctx), p.Value.string(ctx)
        env = append(env, fmt.Sprintf("%s=%s", k, v)) // strconv.Quote(v)
    }
    return
}

func (pc *programContext) pc() *programContext { return pc }
func (pc *programContext) program() *program { return pc.prog }
func (pc *programContext) projects(ctx Context, projects ...*Project) []*Project {
    if len(pc.projs) == 0 { pc.projs = closureProjects(ctx) }
    if len(projects) > 0 { outer: for i, proj := range projects {
        if i == 0 && proj == nil { pc.projs = nil ; continue }
        for _, p := range pc.projs { if proj == p { continue outer }}
        pc.projs = append(pc.projs, proj)
    }}
    return pc.projs
}
func (pc *programContext) Position() Position {
    if pc.prog != nil { return pc.prog.position }
    return pc.autoContext.Position()
}
func (pc *programContext) Project() *Project {
    if cc, ok := pc.Context.(*closurecontext); ok && true {
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
    if cc, ok := pc.Context.(*closurecontext); ok {
        if true {
            scopes = cc.closureScopes()
        } else if up := cc.pc(); up != nil {
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

func (pc *programContext) dirtyOpts() *dirtyOpts { return &pc.by }
func (pc *programContext) dirtyMark(vals ...Value) {
    const (
        enableDirtyMark = true
        perUpdatedDep = true
    )
    if !enableDirtyMark {
        // does nothing
    } else if targets := merge(autoVal(pc, "@")); len(targets) == 0 {
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
func (pc *programContext) interpret(ctx Context, i interpreter, params []Value) {
    if pos := ctx.Position(); !pos.IsValid() && pc.prog.position.IsValid() {
        ctx = at(ctx, pc.prog.position)
    }

    if false && i == dialects["shell"] { defer func() {
        warnstack(ctx, 3, "shell: %v %v", params, ctx.isConfigure()).debug(10)
    }()}

    var err error
    var target Value
    if target, _, _, err = wait(ctx, waitOpts{
        ReportUpdates: false,
        ExecResults: false,
        StampCurrentTarget: false,
    }); err != nil { // wait for prerequisites
        erro(ctx, "waiting traversal failed: %v", err).debug(1)
        return
    }

    if ctx.isConfigure() {/* no dirty-checks for configure */} else
    if f, y := target.(*File); y && pc != nil && !ctx.dirty(ctx) {
        pc.traves.add(ctx, traveDone, nil) // NOTE: modifier.predictDirty

        if false { if e := ctx.entry(); e != nil { if r, y := e.(*rule); y {
            if t, y := r.target.(flag); y {
                warnstack(ctx, 3, "interpret: %v", t, f).debug(1)
            }
        }}}
        return
    }

    var value Value
    if value, err = i.evaluate(ctx, params...); err != nil {
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
    } else if def, prev := autoSet(ctx, "-", value); def == nil {
        var _, ent, _ = entryIndicator(ctx, ctx.entry())
        prompt(ctx, "%v: %s\n", ent, intername(i))
        erro(ctx, "set buffer value failed: %v -> %v", prev, value)
        errostack(ctx, 3, "%v", ctx).debug(1)
        return
    }

    if _, _, err = updateRecipesHash(ctx, target); err != nil {
        var _, ent, _ = entryIndicator(ctx, ctx.entry())
        prompt(ctx, "%v: %s\n", ent, intername(i))
        erro(ctx, "update recipes hash failed: %v", err)
        errostack(ctx, 3, "%v", ctx).debug(1)
    } else if pc != nil {
        pc.interpreted = append(pc.interpreted, i)
    }
    return
}

func isDirty(ctx Context, target Value, a ...Value) (dirty bool) {
    var opts = ctx.dirtyOpts()
    if len(target.updatedDeps(ctx)) > 0 { return true }
    if v := autoVal(ctx, "^"); v != nil { a = append(a, v) }
    for _, dep := range xmerge(ctx, plain, a...) {
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
    if val, /*files*/_, /*execRes*/_, err := wait(pc, waitOpts{
        ReportUpdates: false,
        ExecResults: false,
        StampCurrentTarget: false,
    }); err != nil {
        errostack(ctx, 5, "%v", err).debug(10)
        return
    } else {
        target.Value = val
    }

    var reason string
    var targetFile *File
    var targetFull string
    if false { if s := target.string(ctx); s != "" && (
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
    if targetFile, targetFull, y = target.fullnameFile(ctx); !y {
        targetFull = target.string(ctx)
    } else if n := targetFile._traved; n > 1 {
        if false { warnstack(ctx, 5, "%v, %v, %d", targetFile, targetFull, n).debug(10) }
        return
    }

    var verb = opts.debug>0 || opts.verbose
    var ts = trimPromptString(targetFull)

    if s := target.stat(ctx); s == nil || s.exists() != existenceConfirmed {
        outdated, reason = true, "not exists" //fmt.Sprintf("not exists: %s %v", typeof(target), target)
    } else if isDirty(ctx, target, args...) && isDirtyAfter(ctx, target, s.mod()) {
        outdated, reason = true, "prerequisites updated"
    }

    if outdated {
        assert(reason != "", "needs outdated reason")
    } else if y, e := isRecipesChanged(ctx, target); e != nil {
        erro(ctx, "recipes changed: %v", e).debug(1)
        return
    } else if y {
        outdated, reason = true, "recipes changed"
    } else if !opts.checksum {
        // does nothing
    } else {
        erro(ctx, "FIXME: check prerequisites against the saved checksums").debug(1)
        return
    }

    verb = verb ||
        (opts.verboseOutdated && outdated) ||
        (opts.verboseUpdated && !outdated)

    if verb {
        var d = time.Now().Sub(pc.start)
        if false && d > 10*time.Second {
            warnstack(ctx, 10, "%v: %v", target.Value, d).debug(64)
        }

        var m string
        if outdated { m = "outdated" } else { m = "updated" }

        var s = d.String()
        if targetFile != nil {
            if targetFile._travin > 1 {
                s += fmt.Sprintf(", travin %d", targetFile._travin)
            }
            if targetFile._traved > 1 {
                s += fmt.Sprintf(", traved %d", targetFile._traved)
            }
            if true && targetFile._travin < 2 && targetFile._traved < 2 {
                if targetFile.name(ctx) == "libllvm.Demangle.a" {
                    warn(ctx, "%p: %d, %d, %d, %v, %v, %s", targetFile, targetFile._travin, targetFile._traved, targetFile._dirty, targetFile._updated, targetFile._updatedDeps, targetFile.fullname())
                    warnstack(ctx, 64, "with %v: %v %v", target.Value, with(ctx, target.Value), with(ctx, targetFile)).debug(64)
                }
            }
            targetFile._dirty += 1
        }

        var n = pc.countFiles + len(pc.grepped)
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
    var mapPrereqFile = func(name interface{}) {
        var maps = unmap(ctx, name)
        if maps != nil { defer func() { if prereqFile == nil {
            for _, m := range maps { warn(of(ctx, m.pattern), "%v, skipped %v", name, m) }
            warnstack(ctx, 3, "skipped %d, projects %v", len(maps), projects).debug(8)

            var en int
            for _, p := range projects {
                var c *valcache
                if v, y := name.(Value); y {
                    c = p.filemap.slot(ctx, v, cacheMatchPatts)
                } else if s, y := name.(string); y {
                    c = p.filemap.strx(ctx, s, cacheMatchPatts)
                } else {
                    erro(of(ctx, v), "%v: skipped match: %s, %v (%T)", p, s, v, v).debug(1)
                    break
                }

                if c == nil || c._val == nil {
                    erro(ctx, "%v: %v: %v", p, name, name).debug(1)
                    break
                }

                noted(ctx, "%T %v %v", name, name, c).debug(1)
            }

            if en > 0 { errostack(ctx, 3).debug(8) }
        }}() }

        for _, project := range projects {
            if prereqFile = project.selectFile(ctx, maps); prereqFile != nil {
                prereqValue = prereqFile
                return
            }
        }

        if prereqValue == nil {
            prereqValue = makeStrlit(ctx.Position(), prereqStrval)
        } else if f, y := toFile(prereqValue); y {
            prereqFile = f
        } else if _, y := prereqValue.(*Path); y {
            if f := stat(ctx, prereqStrval); f != nil { prereqFile, prereqValue = f, f }
        }
    }

    if prereqValue = val; prereqValue == nil {
        if prereqStrval == "" {
            errostack(ctx, 3, "prerequisite is nothing").debug(8)
            return
        }

        mapPrereqFile(prereqStrval)
        return
    } else if o, y := prereqValue.(Object); y {
        prereqObj = o
        return
    }

    if !prereqValue.patterned(ctx) {
        prereqStrval = prereqValue.string(ctx)

        if prereqStrval == "" { // just reject empty strval
            errostack(ctx, 3, "%v: %v: empty prerequisite, stems=%v", prereqValue, ctx.stems()).debug(8)
            return
        }

        switch prereqValue.(type) {
        case flag, *strlit, *compound:
            return // skip checking files for performance
        }
        for _, p := range ctx.entry().programs() { if p.configure {
            return
        }}

        mapPrereqFile(prereqValue)
        return
    }

    var stems = ctx.stems()
    if len(stems) == 0 {
        if false { errostack(ctx, 3, "%v: no stems, %v", prereqValue, ctx).debug(8) }
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
    }

    if prereqStrval == "" { prereqStrval = prereqValue.string(ctx); }
    if prereqStrval == "" {
        errostack(ctx, 3, "%v: empty prerequisite, stems=%v", prereqValue, stems).debug(8)
        return
    }

    mapPrereqFile(prereqValue)
    return
}

func (pc *programContext) deferTrave(ctx Context, targetValue, prereqValue, prereqPattern Value, prereqFile *File) {
    var (
        av = targetValue
        bv = prereqValue
    )
    if prereqFile == nil {
        ctx.traversed(ctx, prereqValue) // set $< $> $^ or $|
    } else if targetValue != prereqFile {
        ctx.traversed(ctx, prereqFile) // set $< $> $^ or $|
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

        if prereqFile.exists() { prereqFile._traved += 1
            /* if pc.traves.has(traveNext) */ { pc.traves = pc.traves.not(traveNext) }
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

func with(ctx Context, target Value) (res bool) {
    if a := ctx.ac(); a != nil { if d := a.get(ctx, "@"); target == d || eq(ctx, target, d) {
        res = true
    } else {
        res = with(a.inner(), target)
    }}
    return
}

// traverse - traverse the prerrequiste for the current target $@
func (pc *programContext) traverse(ctx Context, prereqValue Value) (result travestates) {
    defer dtrace(ctx, "traverse")

    var (
        uni = cast[*universe](ctx)
        verb = uni.verbose || uni.verboseBreaks

        projects = ctx.projects(ctx)

        targetValue Value

        prereqPattern Value
        prereqStrval string
        prereqFile *File
        prereqObj Object

        concreteList []Entry
        stemmedList []*stemmed

        t1, t2 time.Time

        db = false
    )
    defer func(t0 time.Time) {
        pc.deferTrave(ctx, targetValue, prereqValue, prereqPattern, prereqFile)

        if result = pc.traves; prereqFile != nil && prereqFile.stat(ctx) == nil {
            prompt(of(ctx, prereqValue), "%v:0: <- missing file\n", prereqFile.fullname())

            if m := prereqFile.filemap; m != nil {
                noted(of(ctx, m.pattern), "%v ⇒ %v ⇒ %v", targetValue, m, prereqFile).debug(1)
            }

            for i, s := range pc.traves { ctx := at(ctx, s.pos)
                noted(ctx, "%v → traves[%d] ⇒ %v", targetValue, i, s).debug(1)
            }
            for i, concrete := range concreteList { ctx := at(ctx, concrete.Position())
                noted(ctx, "%v → concrete[%d] ⇒ %v", targetValue, i, concrete).debug(1)
            }
            for i, stemmed := range stemmedList  { ctx := at(ctx, stemmed.position)
                noted(ctx, "%v → stemmed[%d] ⇒ %v", targetValue, i, stemmed).debug(1)
            }

            erro(of(ctx, prereqValue), "%v ⇒ %v", targetValue, prereqValue).debug(2)
        }

        if d := time.Now().Sub(t0); d > 60*time.Second {
            if false {
                var ac = ctx.ac()
                for ac != nil {
                    if d := ac.get(ctx, "@"); d != nil && d.value == targetValue {
                        warn(ctx, "%v", targetValue).debug(1)
                    }
                    ac = ac.Context.ac()
                }
            }
            for i, c := range concreteList { warn(at(ctx,c.Position()), "%v : C#%d %v", targetValue, i, c) }
            for i, s := range stemmedList  { warn(at(ctx,s.position), "%v : S#%d %v", targetValue, i, s) }
            warnstack(ctx, 5, "slow: %v: %v: %v", targetValue, prereqValue, d).debug(10)
            ctx.dia().flush()
        }
    } (time.Now())

    if targetValue = getTargetValue(ctx); targetValue == nil {
        erro(of(ctx,prereqValue), "%s: target is nil\n", prereqStrval).debug(1)
        return
    } else if isTrivial(targetValue) {
        erro(of(ctx,prereqValue), "%s: target is trivial (%T)\n", prereqStrval, targetValue).debug(1)
        return
    } else if len(projects) == 0 {
        erro(of(ctx,prereqValue), "%v: no projects: %v", prereqStrval, prereqValue).debug(1)
        return
    }

    prereqValue, prereqPattern, prereqStrval, prereqFile, prereqObj =
        probPrereqValue(ctx, projects, prereqValue)

    if uni.db("traverse") || (true && (
        // strings.HasPrefix(targetValue.string(ctx), "HAVE_PTHREAD_IN_LIBC") ||
        // strings.HasPrefix(targetValue.string(ctx), "HAVE_LIBPTHREAD") ||
        false)) {

        if f, y := targetValue.(*File); y {
            prompt(ctx, "%v:0: @\n", f.fullname()).debug(1)
        } else {
            var s, _, _ = entryIndicator(ctx, targetValue)
            prompt(ctx, "%v : %v(%v)\n", s, typeof(prereqValue), prereqStrval).debug(1)
        }

        noted(of(ctx,targetValue), "@: %v(%v) %v(%v)",
            typeof(targetValue), targetValue,
            typeof(prereqValue), prereqValue).debug(1)

        if prereqFile != nil { if false { s := prereqFile.fullname()
            noted(ctx, ">: %T %v ⇒ %v", prereqValue, prereqValue, s).debug(1)
        }} else if f, y := prereqValue.(*File); y {
            noted(at(ctx, f.position), "%v %v", f, f.exists()).debug(1)
        } else if f := file(ctx, prereqStrval); f == nil {
            var a = unmap(ctx, prereqStrval)
            var b = files(ctx, prereqStrval, ctx.Project())
            noted(ctx, ">: %T %v ⇒ file: %v", prereqValue, prereqValue, a)
            noted(ctx, ">: %T %v ⇒ file: %v", prereqValue, prereqValue, b).debug(1)
            if p := ctx.Project(); false {
                warn(ctx, ">: %T %v ⇒ file: %v", prereqValue, prereqValue, p.selectFiles(ctx, a))
                warn(ctx, ">: %T %v ⇒ file: %v", prereqValue, prereqValue, p.selectFiles(ctx, b))
                warn(ctx, ">: %T %v ⇒ file: %v", prereqValue, prereqValue, p.selectFile(ctx, a))
                warn(ctx, ">: %T %v ⇒ file: %v", prereqValue, prereqValue, p.selectFile(ctx, b))
                for i, m := range a { warn(at(ctx, f.position), "%v: %d. %v: %v %v", p, i, m.project, m.name, m.pattern) }
                for i, m := range b { warn(at(ctx, f.position), "%v: %d. %v: %v %v", p, i, m.project, m.name, m.pattern) }
            }
        } else {
            noted(ctx, ">: %T %v ⇒ %v", prereqValue, prereqValue, f).debug(1)
        }

        defer func() {
            var s = targetValue.string(ctx)
            for i, concrete := range concreteList { noted(at(ctx,concrete.Position()), "%v : concrete: %d. %v", targetValue, i, concrete).debug(1) }
            for i, stemmed := range stemmedList { noted(at(ctx,stemmed.position), "%v : stemmed: %d. %v", targetValue, i, stemmed).debug(1) }
            for i, t := range pc.traves { noted(at(ctx, t.pos), "%v: %d. %v", s, i, t).debug(1) }
            notestack(ctx, 5).debug(2)
        } ()

        db = true
    }

    if f := prereqFile; f != nil { if f._travin += 1; f._travin > 1 { return }}

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
            if val := autoVal(c, "@"); val != nil && eq(c, val, prereqValue) {
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
        if n := prereqFile._traved; n > 0 {
            if false && n > 1 {
                warn(ctx, "traversed: %v: %v %T", targetValue, prereqValue, prereqValue)
                warn(ctx, "traversed: %v: %v", targetValue, prereqFile.fullname())
                warnstack(ctx, 3, "traversed: %d", prereqFile._traved).debug(10)
            }
            return
        }

        travedPrereqFile = func (s *travestate) (res *File) { return }
    } else {
        // If the prereqValue is not a *File, for example a (*strlit) or (*compound)
        // %.h <-> 'llvm/PassSupport.h' <-> [
        //   file@llvm/PassSupport.h>/Volumes/workspace/external/llvm-project/llvm/include/llvm/PassSupport.h
        //   file@llvm/PassSupport.h>/Volumes/workspace/external/llvm-project/llvm/include/llvm/PassSupport.h
        //   file@llvm/PassSupport.h>/Volumes/workspace/external/llvm-project/llvm/include/llvm/PassSupport.h
        //   done@llvm/PassSupport.h
        //   done@llvm/PassSupport.h
        // ]
        travedPrereqFile = func (s *travestate) (res *File) {
            // and the trave target is a *File with the name matched
            if f, y := toFile(s.target); y && f.name(ctx) == prereqStrval { res = f }
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
        if false && db { defer func() { noted(ctx, "%v: %v", entry, pc.traves).debug(2) } () }

        entry.traverse(ctx) // NOTE: this adds traveRule to pc.traves

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
            if sc := ctx.sc(); sc != nil && sc.stem != nil && sc.stem.target != nil {
                stemmedThisTarget = eq(ctx, sc.stem.target, targetValue)
            }
            for _, s := range tt {
                if g := s.target; g == nil {
                    prompt(ctx, "%v: %v\n", targetValue.string(ctx), t)
                    erro(ctx, "%v: %v %v %v\n", targetValue.string(ctx), prereqPattern, prereqValue, pattern)
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
                        //   (*program).traverse func can break it's loop properly.
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
                } else if _, y := prereqValue.(*strlit); false && y {
                    if prereqFile == nil { prompt(ctx, "nonfile: %T %v: %T %v : %T %v ; %v\n",
                        targetValue, targetValue, prereqValue, prereqValue, s.target, s.target, s).debug(1) }
                }
            }
        }
        return
    }

    t1 = time.Now()

    for _, project := range projects {
        var entries = project.resolveEntries(ctx, prereqValue, false)
        if false && db {
            noted(ctx, "entries: %T %v ⇒ %v (%v)", prereqValue, prereqValue, entries, project).debug(1)
        }
        if entries == nil || len(entries.all) == 0 { continue }
        concreteList = append(concreteList, entries.all...)
    ForEntries:
        for _, entry := range entries.all {
            if !isNull(entry) && targetValue == entry { continue ForEntries }
            if w, k := targetValue.(*bareword); k && w.s == prereqStrval {
                continue ForEntries // target resolve to itself, does nothing
            }
            switch traverseEntry(project, entry, false) {
            case traveResBreak : break ForEntries
            case traveResReturn: return //if ok { return } else { break ForEntries }
            }
        }
    }
    if d := time.Now().Sub(t1); d > 60*time.Second {
        for _, concrete := range concreteList { prompt(ctx, "%v: slow: %v %v\n", concrete.Position(), concrete, targetValue) }
        prompt(ctx, "%v: slow: %v: %v %v (%d concretes)\n", ctx.Position(), targetValue, prereqValue, d, len(concreteList)).debug(1)
    }

    if prereqFile != nil && prereqFile.exists() { goto CheckPrereqResult }

    t2 = time.Now()

    for _, project := range projects {
        if false { for i, p := range project.patterns { t := p.target
            if s := t.string(ctx); s == ".configure/library/*.c" || strings.HasPrefix(s, ".configure/library/HAVE_") {
                a, b, c := p.match(ctx, prereqStrval)
                noted(ctx, "stemmed: %s ⇒ %d %v %v ⇒ %v %v %v %v", prereqStrval, i, typeof(t), p, s, a, b, c).debug(10)
            }
        }}
        for _, p := range project.patterns { assert(p.target.patterned(ctx), "not pattern") }
        var patterns = project.resolvePatterns(ctx, prereqValue, prereqStrval)
        if false && db {
            noted(ctx, "stemmed: %T %v ⇒ %v (%v)", prereqValue, prereqValue, patterns, project).debug(1)
        }
        if len(patterns) == 0 { continue }
        stemmedList = append(stemmedList, patterns...)
    ForPatterns:
        for _, entry := range patterns {
            switch traverseEntry(project, entry, true) {
            case traveResBreak : break ForPatterns
            case traveResReturn: return //if ok { return } else { break ForPatterns }
            }
        }
    }
    if d := time.Now().Sub(t2); d > 60*time.Second {
        for _, stemmed  := range stemmedList { prompt(ctx, "%v: slow: %v\n", stemmed.position, stemmed) }
        prompt(ctx, "%v: slow: %v: %v %v (%d stemmed)\n", ctx.Position(), targetValue, prereqValue, d, len(stemmedList)).debug(1)
    }

CheckPrereqResult:
    if false && prereqFile == nil { prereqFile, _ = toFile(prereqValue) }

    var p = prereqFile
    if p == nil {
        // fallthrough
    } else if p.exists() {
        trave := pc.traves.add(ctx, traveFile, targetValue)
        trave.dependPat = prereqPattern
        trave.depend = p
        return
    } else if m := p.filemap; m != nil {
        if p.info == nil { p.stat(ctx) }
        if p.info != nil {
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
        if p = stat(ctx, prereqStrval); p != nil {
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
        if t[0].depend.(Entry).name(ctx) == prereqStrval { return }
    }

    if t := pc.traves.of(traveObj); t.has() && t[0].depend != nil {
        if t[0].depend.(Object).name(ctx) == prereqStrval { return }
    }

    if prereqPattern != nil {
        s := pc.traves.add(ctx, traveNext, targetValue)
        s.dependPat = prereqPattern
        s.depend = prereqValue
        return
    }

    if ctx.isConfigure() { return }

    if len(ctx.stems()) == 0 || ctx.mustExists() {
        if s := pc.traves.add(ctx, traveFail, targetValue); prereqFile != nil {
            s.error = fileNotFoundError{ctx.Project(), prereqFile}
            ctx = at(ctx, prereqFile.position)
        } else {
            s.error = targetNotFoundError{ctx.Project(), prereqStrval}
            if prereqValue != nil { ctx = at(ctx, prereqValue.Position()) }
        }

        if prereqFile != nil && prereqValue != prereqFile {
            noted(ctx, "%v(%v): %v(%v); file=%v\n", typeof(targetValue), targetValue, typeof(prereqValue), prereqValue, prereqFile).debug(1)
        } else if prereqFile != nil {
            noted(ctx, "%v(%v): %v(%v); path=%s\n", typeof(targetValue), targetValue, typeof(prereqValue), prereqValue, prereqFile.fullname()).debug(1)
        } else if prereqObj != nil {
            noted(ctx, "%v(%v): %v(%v); obj=%v\n", typeof(targetValue), targetValue, typeof(prereqValue), prereqValue, prereqObj).debug(1)
        } else {
            noted(ctx, "%v(%v): %v(%v)\n", typeof(targetValue), targetValue, typeof(prereqValue), prereqValue).debug(1)
        }

        if val := prereqValue; val != nil { erro(at(ctx,val.Position()), "value: %T %v ; %s %v", val, val, prereqStrval, files(ctx, prereqStrval)) }
        if fil := prereqFile;  fil != nil { erro(at(ctx,fil.Position()), "file: %v, exists=%v", fil, fil.exists()) }
        if obj := prereqObj;   obj != nil { erro(at(ctx,obj.Position()), "object: %T %v", obj, obj) }
        for i, s := range pc.traves { erro(at(ctx,s.pos), "trave.%d: %v: %v: %v", i, targetValue, prereqValue, s) }
        for i, c := range ctx.closureScopes() { erro(ctx, "closure.%d: %v", i, c) }
        for i, concrete := range concreteList { erro(at(ctx,concrete.Position()), "concrete: %d. %v (%d programs)", i, concrete, len(concrete.programs())) }
        for i, stemmed  := range stemmedList  { erro(at(ctx,stemmed.position), "stemmed: %d. %v", i, stemmed) }
        errostack(of(ctx, prereqValue), 6).debug(512)
        return
    }

    return // no operation
}

func (pc *programContext) prerequisite(ctx Context, prerequisites []Value) {
    var (
        uni = cast[*universe](ctx)
        verb = uni.verbose || uni.verboseBreaks
        ent  = autoVal(ctx, "@")
        stem *stemmed
        depends valvec
        by Value
    )
    defer func() {
        pc.Wait()

        var target Value = autoVal(ctx, "@")

        // NOTE: warnings if travestates go too large
        if true {
            // does nothing
        } else if tt := pc.traves.of(traveObj, traveRule, traveFile); len(tt) > 100 {
            prompt(ctx, "%v: traves=%d\n", target, len(pc.traves))
            for i, s := range pc.traves {
                prompt(ctx, "%v: %d. %v\n", target, i, s)
                if i > 10 { break }
            }
            warnstack(ctx, 5, "%d errors", ctx.dia().flush()).debug(16)
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

    if sc := ctx.sc(); sc != nil { stem = sc.stem }

ForPrerequisites:
    for _, prerequisite := range prerequisites {
        var (
            ctx = at(ctx, prerequisite.Position())
            k = len(pc.traves)
            t0 = time.Now()
        )
        switch by = prerequisite; u := prerequisite.(type) {
        case unexpanded : warn(ctx, "%v: unexpanded %v" , ent, u.Value).debug(1) ; continue
        case untraversed: warn(ctx, "%v: untraversed %v", ent, u.Value).debug(1) ; continue
        default: prerequisite.traverse(ctx)
        }

        var traves = pc.traves.slice(k)
        var target = autoVal(ctx, "@") // fetch updated $@
        var depend = autoVal(ctx, ">") // fetch updated $>
        var isPatternStemmedForTarget = stem != nil && stem.target != nil &&
            eq(ctx, stem.target, target)

        if false {
            var p = prerequisite
            if _, y := p.(*modification); !y {
                v := p.expand(ctx, strval|expandUnexpandedForth)
                noted(ctx, "%v %v(%v) → %v(%v)", target, typeof(p), p, typeof(v), v).debug(1)
            } else {
                noted(ctx, "%v %v(%v) ⇒ %v", target, typeof(p), p, depend).debug(1)
            }
        }

        if depend != nil {
            if u, y := depend.(unexpanded); y { v := u.Value
                erro(ctx, "%v: %v(%v) ⇒ %v(%v)", target, typeof(prerequisite), prerequisite, typeof(v), v).debug(1)
            } else {
                depends.add(depend)
            }
        } else if _, y := prerequisite.(*modification); y {
            // noop
        } else if p, y := prerequisite.(*delegate); y && p.x.kind() == KindUseList {
            // noop
        } else if p := prerequisite.expand(ctx, strval|expandUnexpandedForth); p == nil {
            // noop
        } else if u, y := p.(unexpanded); y {
            noted(ctx, "%v: %v(%v) → %v(%v), %v(%v)",
                target, typeof(prerequisite), prerequisite, typeof(u.Value), u.Value).debug(1)
        }

        if d, f := time.Now().Sub(t0), time.Duration(pc.countFiles); d > 15*time.Second &&
            (f == 0 || (d/f > 10*time.Millisecond)) {
            var p = prerequisite
            var a = target
            var t = autoVal(ctx, "^")
            var c = autoVal(ctx, "<")
            var q = depend
            var pos = ctx.Position()
            if f != 0 {
                prompt(ctx, "%v: program.traverse slow: %v %v %d\n", pos, d, d/f, pc.countFiles)
            } else {
                prompt(ctx, "%v: program.traverse slow: %v %d\n", pos, d, pc.countFiles)
            }
            prompt(ctx, "%v: program.traverse %v: @: %v\n", pos, p, a)
            prompt(ctx, "%v: program.traverse %v: ^: %v\n", pos, p, t)
            prompt(ctx, "%v: program.traverse %v: <: %v\n", pos, p, c)
            prompt(ctx, "%v: program.traverse %v: >: %v\n", pos, p, q)
            infostack(ctx, 3).debug(1)
        }

        if false && !pc.prog.configure {
            prompt(ctx, "%v: %s %v ; %s %v\n", target, typeof(prerequisite), prerequisite, typeof(depend), depend)
            for _, s := range traves {
                if s.what == traveFile {
                    var f = s.depend.(*File)
                    prompt(ctx, "%v: %v: %v ; exists=%v\n", target, prerequisite, s, f.exists())
                } else {
                    prompt(ctx, "%v: %v: %v\n", target, prerequisite, s)
                }
            }
            if false { prompt(ctx, "%v: %v\n", target, pc.prog.configure).debug(6) }
            if true { infostack(ctx, 3, "%v, configure=%v", target, pc.prog.configure).debug(16) }
        }

        if pc.debug_traverse > 0 { if true { pc.debug_traverse -= 1 }
            for i, a := range pc.traves { info(of(ctx,prerequisite), "%v: %d. %v %v", ent, i, a.what, a) }
            warnstack(of(ctx,prerequisite), 12, "%v: %v: %v %v", pc.prog.project, ent, prerequisite, autoVal(ctx, ">")).debug(10)
        }

        if tt := traves.of(traveFail); tt.has() {
            if /* isPatternStemmedForTarget */true {
                for _, s := range tt {
                    var dependMine = depends.has(s.depend)
                    if s.error == traveTargetNotDefinedFile && dependMine {
                        // add traveNext to try the next pattern
                        pc.traves.remove(s).add(ctx, traveNext, target)
                    } else if depend == nil {
                        prompt(ctx, "%v: %v\n", target, s).debug(1)
                        erro(at(ctx,s.pos), "%v", s)
                        erro(of(ctx,target), "1. %T %v %s", target, target, target.string(ctx))
                        erro(of(ctx,s.depend), "2. %T %v mine=%v", s.depend, s.depend, dependMine)
                        erro(of(ctx,prerequisite), "3. %T %v", prerequisite, prerequisite)
                        errostack(ctx, 5, "#>").debug(10)
                    } else {
                        prompt(ctx, "%v: %v: %v\n", target, depend, s).debug(1)
                        erro(at(ctx,s.pos), "%v", s)
                        erro(of(ctx,target), "1. %T %v %s", target, target, target.string(ctx))
                        erro(of(ctx,depend), "2. %T %v %s", depend, depend, depend.string(ctx))
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
                    pc.traves, target, prerequisite, prerequisite, p, ctx.stems(), stem)

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
            var str = ent.String()
            if !eq(ctx, ent, target) { str = fmt.Sprintf("%v(%v)", str, target) }

            prompt(ctx, "%v: %s: %v\n", prerequisite.Position(), str, prerequisite)
            for i, s := range pc.traves { prompt(ctx, "%v: %s: %d. %v\n", s.pos, str, i, s) }
            errostack(ctx, 5).debug(16)
            return
        }
    }
    return
}

type program struct {
    position Position
    project *Project
    scope   *Scope
    params   []*auto
    depends  []Value // normal
    ordered  []Value // order-only
    recipes  []Value
    language  string
    configure bool
}

func (prog *program) getModifiers(ctx Context, name string) (ms []*modifier) {
    for _, d := range prog.depends {
        if g, y := d.(*modification); y { for _, m := range g.list {
            if m.Elems[0].string(ctx) == name { ms = append(ms, m) }
        }}
    }
    return
}

const maxCallRecursion  = 32 //64

type normalTraverseContext struct { Context }
type orderTraverseContext struct { Context }
func (t normalTraverseContext) traversed(ctx Context, target Value) (targets []Value) {
    if targets = t.Context.traversed(ctx, target); len(targets) > 0 {
        autoSet(ctx, "^", makeList(t.Position(), targets...))
        autoSet(ctx, "<", targets[0])
        autoSet(ctx, ">", targets[len(targets)-1])
    }
    if false { noted(ctx, "%v %v", target, targets).debug(1) }
    return
}
func (t orderTraverseContext) traversed(ctx Context, target Value) (targets []Value) {
    if targets = t.Context.traversed(ctx, target); len(targets) > 0 {
        autoSet(ctx, "|", makeList(t.Position(), targets...))
    }
    if false { noted(ctx, "%v %v", target, targets).debug(1) }
    return
}

func (prog *program) workDir(ctx Context) (workDir string) {
    if pc := ctx.pc(); pc == nil {
        workDir = prog.project.absPath
    } else if pc.changedWD == "" {
        var o Object
        if _, o = prog.scope.find("CWD"); isTrivial(o) {
            if _, o = prog.scope.find("/"); isTrivial(o) {
                erro(ctx, "both $(CWD) and $/ are trivial").debug(1)
                return
            }
        }
        if x, y := o.(invoker); y {
            if v := x.invoke(ctx, plain, nil, nil); v != nil {
                workDir = v.string(ctx)
            } else {
                erro(ctx, "trivial %T %v", x, x).debug(1)
            }
        } else if v := o.expand(ctx, strval); v != nil {
            workDir = v.string(ctx)
        } else {
            erro(ctx, "trivial %T %v", x, x).debug(1)
        }
    } else if filepath.IsAbs(pc.changedWD) {
        workDir = pc.changedWD
    } else {
        workDir = filepath.Join(prog.project.absPath, pc.changedWD)
    }
    return
}

func (prog *program) execute(ctx Context) (result Value, _traves travestates) {
    var entry = ctx.entry()
    var dia = ctx.dia()

    defer dtrace(ctx, "execute "+entry.String())

    if t := dia.countErrors(); t > 0 {
        erro(ctx, "%v: got %d errors, canceled execution (%v)", entry, t, prog.project).debug(1)
        return
    }

    assert(prog.project == prog.scope.project, "mismatched scope/project")

    pc := programContext{
        autoContext: autoContext{ Context:ctx, defs:make(autoDefMap) },
        execRec: make(map[Value]int), start: time.Now(), prog: prog, print: true,
    }

    ctx = &pc

    defer func() {
        var targets = autoVal(ctx, "@") // depends = autoVal(ctx, "^")
        if isTrivial(targets) { targets = entry.Target() }

        if errs := dia.countErrors(); errs > 0 {
            var str, ent, tar = entryIndicator(ctx, entry)
            if false { for _, s := range pc.traves {
                erro(at(ctx,s.pos), "%v: %v", ent, s).debug(1)
            }}

            if tar != "" && tar != ent {
                noted(ctx, "%s: %s: got %d errors", ent, tar, errs).debug(1)
            } else {
                noted(ctx, "%s: got %d errors", ent, errs).debug(1)
            }

            if false && !ctx.isConfigure() {
                pc.traves.add(ctx, traveFail, targets).
                    error = fmt.Errorf("got %d error(s) for %v", errs, str)
            }
        } else { for _, a := range pc.defers { if g, y := a.(*group); y {
            modify(ctx, g, true)
        } else {
            erro(of(ctx, a), "defer: not a modifier: %v: %v", typeof(a), a).debug(1)
        }}}

        if result == nil { if result = autoVal(ctx, "-"); result == nil {
            result = pc.defaultVal
        }}
        if result != nil { if caller := pc.caller(); caller != nil {
            caller.defaultVal = result
        }}

        pc.defaultVal = nil
        _traves = pc.traves
    } ()

    if true {
        var cc = pc.Context
        var a = []Value{ autoVal(cc, "@") }
        var depth, loop int = 0, -1

    ForPC:
        for c := pc.caller(); c != nil; c = c.caller() { if c.program() == prog {
            if depth += 1; depth == maxCallRecursion { break ForPC }
            var t = autoVal(c, "@")
            for i, v := range a { if eq(ctx, t, v) { loop = i; break ForPC } }
            if loop < 0 { a = append(a, t) }
        }}

        if 0 <= loop { var t = autoVal(cc, "@")
            if o := cc.closure(); o != nil { if v := autoVal(o, "@"); v != nil && eq(cc, v, t) {
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
            }}

            prompt(ctx, "%v: %v: %v, %v\n", a[0], autoVal(cc.closure(), "@"), cc, cc.closure())
            for i, t := range a { erro(of(ctx,t), "loop: %v: %v", i, t) }
            errostack(at(ctx,prog.position), 128, "loop, (depth=%d, %v, %v)\n", depth, a[loop], a).debug(6)
            return
        }

        if depth < maxCallRecursion {
            // continues
        } else if c := pc.caller(); c != nil { pc.traceLevel = c.traceLevel
            var tt = as{autoVal(c, "@")}
            var s, _ = tt.fullnameOrStrval(ctx)
            prompt(ctx, "%v: max recursion call (%d)\n", s, depth)
            warn(of(ctx,tt), "max recursion call (%d)\n", depth).debug(1)

            const collapse = false
            for ; c != nil; c = c.caller() { var n int
                if collapse { for next := c.caller(); next != nil; next = next.caller() {
                    if d := autoVal(next, "@"); d == nil { continue } else
                    if t := d; t != nil && eq(ctx, t, tt) { n += 1;  continue }
                    if next.program() == c.program() { n += 1; c = next } else { break }
                }}

                if prog, t := c.program(), autoVal(c, "@"); prog == nil {
                    erro(at(ctx,entry.Position()), "%v (@=%v)", entry, tt)
                    break
                } else if pos := prog.position; n > 0 {
                    erro(at(ctx,pos), "%v (repeated %d times)", t, n)
                } else if !collapse {
                    erro(at(ctx,pos), "%v : %v", t, autoVal(c, ">"))
                } else if depth -= 1; maxCallRecursion - depth > 5 {
                    erro(at(ctx,pos), "%v ... (%d)", t, maxCallRecursion - depth)
                    break
                } else {
                    erro(at(ctx,pos), "%v : %v", t, autoVal(c, ">"))
                }

                dia.flush() // dump immediately
            }

            errostack(ctx, depth, "#>", entry).debug(512)
            if false { panic(failure{"max call depth",ia(prog.position)}) }
            return
        }

        if t := ctx.stems(); t != nil { autoSet(ctx, "*", ease(ctx, t)) }
    }

    var uni = cast[*universe](ctx)

    // NOTE: set "@" before setting auto args
    // Select the right target value before setting parameters,
    // because the target could be overrided by parameters.
    if target := entry.Target(); target == nil {
        erro(ctx, "%v: nil entry target", target).debug(1)
        return
    } else {
        switch a := target.(type) {
        case *strlit, *compound: // NOTE: skip strings to optimize speed from searching
        case fullfile: if a._traved > 1 { return } // alreadyUpdated = a.info != nil && a.updated
        case    *File: if a._traved > 1 { return } // alreadyUpdated = a.info != nil && a.updated
        case     flag: pc.print = false // Flag target (-foo) turns off printing automatically
        default:
            if file := prog.project.file(ctx, a.string(ctx)); file != nil {
                if file._traved > 1 { return } else { target = file }
            }
        }

        if pc.execRec[target] += 1; false { if pc.execRec[target] > 1 {
            if uni.traceExec { t_exec.trace(fmt.Sprintf("exec: %v", target)) }
            return
        }}

        autoSet(ctx, "@", target)
    }

    pc.initializeArgs()

    if uni.traceExec {
        var d = pc.depth()
        var t = autoVal(ctx, "@")
        var s = fmt.Sprintf("%s: %v (%p, exec.depth=%d)", typeof(t), t, t, d)
        defer un(trace(t_exec, s))
    }

    autoSet(ctx, "^", nil)
    autoSet(ctx, "<", nil)
    autoSet(ctx, ">", nil)
    autoSet(ctx, "|", nil)

    // Update normal prerequisites
    pc.prerequisite(normalTraverseContext{ctx}, prog.depends)
    if errs := dia.countErrors(); errs > 0 {
        s := pc.traves.add(at(ctx, prog.position), traveFail, nil)
        s.error = fmt.Errorf("%d errors counted", errs)
        return
    } else if pc.traves.has(traveCase, traveDone, traveFail, traveNext) {
        return
    }

    // Update order-only prerequisites
    pc.prerequisite(orderTraverseContext{ctx}, prog.ordered)
    if errs := dia.countErrors(); errs > 0 {
        s := pc.traves.add(at(ctx, prog.position), traveFail, nil)
        s.error = fmt.Errorf("%d errors counted", errs)
        return
    } else if pc.traves.has(traveCase, traveDone, traveFail, traveNext) {
        return
    }

    if prog.language != "" || len(prog.recipes) == 0 {
        return
    }

    if h := autoVal(ctx,"-"); h == nil || len(pc.interpreted) == 0 {
        if i, y := dialects["eval"]; y && i != nil {
            pc.interpret(ctx, i, nil)
        } else {
            erro(ctx, "no default dialect").debug(1)
        }
    }
    return
}
