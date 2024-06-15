//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "path/filepath"
    "reflect"
    "io/fs"
    "sync"
    "time"
    "fmt"
    "os"
)

type dirtyOpts struct {
    generalOpts
    verboseUpdated  bool "verbose-updated"  // vu,
    verboseOutdated bool "verbose-outdated" // vo,
    checksum bool "checksum" // cs,crc,
    silent   bool "silent" // s,
    pats []Value
}

func _execution(c Context) *execution { return cast[*execution](c) }

type execution struct {
    automatic
    sync.Mutex
    sync.WaitGroup

    by dirtyOpts

    prog *program
    projs []*project

    defaultVal Value
    defers   []Value
    values   []Value

    _env []*pair
    changedWD string

    dirt string // reason of outdated
    start time.Time // start time

    execRec map[Value]int

    calleeErrs []error
    calleeErrsM sync.Mutex

    grepping bool
    grepped []Value
    targets []Value // all targets def

    countFiles int

    traceLevel int
    traves travestates

    interpreted []interpreter
}
func (pc *execution) inner() Context { return &pc.automatic }
func (pc *execution) caller() *execution { return _execution(pc.Context) }
func (pc *execution) cast(t reflect.Type) Context {
    if reflect.TypeOf(pc) == t { return pc }
    if reflect.TypeOf((*argumented_context)(nil)) == t { return nil }
    if reflect.TypeOf((*stemmed_context)(nil)) == t { return nil }
    return pc.automatic.cast(t)
}
func (pc *execution) do(ctx Context, op any) (res any) {
    switch t := op.(type) {
    case get_position:
        if pc.prog != nil { return pc.prog.position }
    case get_project:
        if pc.prog.project != nil { return pc.prog.project }
    case get_scope:
        if pc.prog.project != nil { return pc.prog.project.scope }
    case get_closure:
        if x, y := pc.Context.(*terminal); y { return do(x, op) }

    case act_dirty_mark:        pc.dirtyMark(t.a...)
    case act_dirt:     return pc.dirty(ctx,t.a...)
    case act_traverse:  return pc.traverse(ctx,t.v)
    case act_traversed: return pc.traversed(ctx,t.v)

    case get_param_name:
        if t.i < len(pc.prog.params) {
            res = pc.prog.params[t.i].name
        }
        return
    case property:
        if t&propDirtyOpts != 0 { return &pc.by }
        if t&propParameters != 0 {
            var params map[string]*auto
            for _, param := range pc.prog.params {
                if params == nil { params = make(map[string]*auto, len(pc.prog.params)) }
                params[param.name] = param
            }
            return params
        }
    }
    return pc.automatic.do(ctx, op)
}

func (pc *execution) ts(t string) string {
	return fmt.Sprintf("{=%s %v}", t, ts(pc.Context)) // NOTE: hides {=automatic}
}

func (pc *execution) aquire() func() { pc.Lock() ; return func(){ pc.Unlock() }}

func (pc *execution) depth() (res int) {
    for c := pc.caller(); c != nil; c = c.caller() { res += 1 }
    return
}
func (pc *execution) calleeError(err error) {
    if err != nil {
        pc.calleeErrsM.Lock()
        pc.calleeErrs = append(pc.calleeErrs, err)
        pc.calleeErrsM.Unlock()
    }
}

// traverse_context is a single thread traverse context, for traversing in a new goroutine,
// a spawned traversal must be used and then merge.
func (pc *execution) level(n int) { pc.traceLevel += n }
func (pc *execution) trace(a ...interface{}) { printIndentDots(pc.traceLevel, a...) }
func (pc *execution) tracef(s string, a ...interface{}) { printIndentDots(pc.traceLevel, fmt.Sprintf(s, a...)) }
func (pc *execution) traversed(ctx Context, target Value) []Value {
    if !isTrivial(target) {
        pc.targets = append(pc.targets, target)

        if false { if cc, y := pc.Context.(*terminal); y {
            pc.targets, _ = do(cc, act_traversed{target}).([]Value)
        } }

        if _, y := toFile(target); y { pc.addFilesCount(1) }
    }
    return pc.targets
}

func (pc *execution) addFilesCount(n int) {
    pc.countFiles += n
    if c := pc.caller(); c != nil { c.addFilesCount(1) }
}

func (pc *execution) env(ctx Context) (env []string, osi int) {
    env = os.Environ()
    osi = len(env)
    for _, p := range pc._env {
        var k, v = p.key.string(ctx), p.val.string(ctx)
        env = append(env, fmt.Sprintf("%s=%s", k, v)) // strconv.Quote(v)
    }
    return
}

func (pc *execution) dirtyMark(vals ...Value) {
    const (
        enableDirtyMark = true
        perUpdatedDep = true
    )
    if !enableDirtyMark {
        // does nothing
    } else if targets := merge(auto_get(pc, "@")); len(targets) == 0 {
        // should not happen, but safely ignoring..
    } else if len(vals) == 0 {
        vals = append(vals, targets...)
    } else if true {
        vals = merge(vals...)

        var mat, dup bool
        var opts = &pc.by
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
    if enableDirtyMark { pc.dirtyMark(vals...) }
}
func (pc *execution) interpret(ctx Context, i interpreter, params []Value) {
    if pos := _position(ctx); !pos.IsValid() && pc.prog.position.IsValid() {
        ctx = at(ctx, pc.prog.position)
    }

    target, _, _, err := wait(ctx, waitopts{
        ReportUpdates: false,
        ExecResults: false,
        StampCurrentTarget: false,
    })
    if err != nil { // wait for prerequisites
        erro(ctx, "waiting traversal failed: %v", err).debug()
        return
    }

    if isConfigure(ctx) {/* no dirty-checks for configure */} else
    if _, y := target.(*File); y && pc != nil && !pc.dirty(ctx) {
        pc.traves.add(ctx, traveDone, nil) // NOTE: modifier.predictDirty
        return
    }

    var value Value
    if value, err = i.evaluate(ctx, params...); err != nil {
        var _, ent, _ = entryIndicator(ctx, _entry(ctx))
        var nam = intername(i)
        prompt(ctx, "%v: interpret '%s' recipes failed\n", ent, nam)
        erro(ctx, "%s: %v", nam, err)
        errostack(ctx, 3).debug()
        return
    } else if value == nil {
        // disgard nil value
    } else if def, prev := auto_set(ctx, "-", value); def == nil {
        var _, ent, _ = entryIndicator(ctx, _entry(ctx))
        prompt(ctx, "%v: %s\n", ent, intername(i))
        erro(ctx, "set buffer value failed: %v -> %v", prev, value)
        errostack(ctx, 3).debug()
        return
    }

    if _, _, err = updateRecipesHash(ctx, target); err != nil {
        var _, ent, _ = entryIndicator(ctx, _entry(ctx))
        prompt(ctx, "%v: %s\n", ent, intername(i))
        erro(ctx, "update recipes hash failed: %v", err)
        errostack(ctx, 3).debug()
    } else if pc != nil {
        pc.interpreted = append(pc.interpreted, i)
    }
    return
}

func isDirty(ctx Context, target Value, a ...Value) (dirty bool) {
    var opts, y = do(ctx, propDirtyOpts).(*dirtyOpts)
    if !y {
        erro(ctx, "nil dirtyopts : %v", ts(ctx)).debug()
        return
    }
    if len(target.updatedDeps(ctx)) > 0 { return true }
    if v := auto_get(ctx, "^"); v != nil { a = append(a, v) }
    for _, dep := range xmerge(ctx, a...) {
        var mat bool = len(opts.pats) == 0
        if !mat {
            for _, pat := range opts.pats {
                if mat, _, _ = pat.match(ctx, dep); mat { break }
            }
        }
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

func (pc *execution) dirty(ctx Context, aa ...Value) (outdated bool) {
    var target as
    if val, /*files*/_, /*execRes*/_, err := wait(pc, waitopts{
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

    var opts, args = _opts_[dirtyOpts](ctx, aa...)

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
        erro(ctx, "recipes changed: %v", e).debug()
        return
    } else if y {
        outdated, reason = true, "recipes changed"
    } else if !opts.checksum {
        // does nothing
    } else {
        erro(ctx, "FIXME: check prerequisites against the saved checksums").debug()
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
                if targetFile.ident(ctx) == "libllvm.Demangle.a" {
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

func probPrereqValue(ctx Context, projects []*project, val Value) (prereqValue, prereqPattern Value, prereqFinal string, prereqFile *File, prereqObj Object) {
    var mapPrereqFile = func(name interface{}) {
        var maps = unmap_files(ctx, name)
        if maps != nil {
            defer func() {
                if prereqFile != nil { return }

                for _, m := range maps { warn(at(ctx, m.pattern), "%v, skipped %v", name, m) }
                warnstack(ctx, 3, "skipped %d, projects %v", len(maps), projects).debug(8)
            }()
        }

        for _, project := range projects {
            if prereqFile = project.selectFile(ctx, maps); prereqFile != nil {
                prereqValue = prereqFile
                return
            }
        }

        if prereqValue == nil {
            prereqValue = makeStrlit(_position(ctx), prereqFinal)
        } else if f, y := toFile(prereqValue); y {
            prereqFile = f
        } else if _, y := prereqValue.(*path); y {
            if f := stat(ctx, prereqFinal); f != nil { prereqFile, prereqValue = f, f }
        }
    }

    if prereqValue = val; prereqValue == nil {
        if prereqFinal == "" {
            errostack(ctx, 3, "prerequisite is nothing").debug(8)
            return
        }

        mapPrereqFile(prereqFinal)
        return
    } else if o, y := prereqValue.(Object); y {
        prereqObj = o
        return
    }

    if !prereqValue.patterned(ctx) {
        prereqFinal = prereqValue.string(ctx)

        if prereqFinal == "" { // just reject empty final
            errostack(ctx, 3, "%v: %v: empty prerequisite, stems=%v", prereqValue, _stems(ctx)).debug(8)
            return
        }

        switch prereqValue.(type) {
        case flag, *strlit, *compound:
            return // skip checking files for performance
        }
        for _, p := range _entry(ctx).programs() { if p.configure {
            return
        }}

        mapPrereqFile(prereqValue)
        return
    }

    var stems = _stems(ctx)
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

    if prereqFinal == "" { prereqFinal = prereqValue.string(ctx); }
    if prereqFinal == "" {
        errostack(ctx, 3, "%v: empty prerequisite, stems=%v", prereqValue, stems).debug(8)
        return
    }

    mapPrereqFile(prereqValue)
    return
}

func (pc *execution) deferTrave(ctx Context, targetValue, prereqValue, prereqPattern Value, prereqFile *File) {
    var av = targetValue
    var bv = prereqValue
    if prereqFile == nil {
        do(ctx, act_traversed{prereqValue}) // set $< $> $^ or $|
    } else if targetValue != prereqFile {
        do(ctx, act_traversed{prereqFile}) // set $< $> $^ or $|
        bv = prereqFile
    } else if t := pc.traves.of(traveFile); t.has() {
        for _, s := range t {
            if d := s.depend; d != bv && !isTrivial(d) { bv = d }
        }
    }

    if !isTrivial(av) && !isTrivial(bv) {
        var a = av.stat(ctx).mod()
        var b = bv.stat(ctx).mod()
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
    if ac := _automatic(ctx); ac == nil {
        return
    } else if t := ac.search(ctx, "@"); t == nil || t.value == nil {
        return
    } else if res = equal(ctx, target, t.value); res {
        return
    } else {
        return with(inner(ac), target)
    }
}

// traverse - traverse the prerrequiste for the current target $@
func (pc *execution) traverse(ctx Context, prereqValue Value) (result travestates) {
    var (
        targetValue Value

        prereqPattern Value
        prereqFinal string
        prereqFile *File
        prereqObj Object

        concreteList []entry
        stemmedList []*stemmed

        t1, t2 time.Time

        _db = false
    )

    defer trace(ctx)
    defer func(t0 time.Time) {
        pc.deferTrave(ctx, targetValue, prereqValue, prereqPattern, prereqFile)

        if result = pc.traves; prereqFile != nil && prereqFile.stat(ctx) == nil {
            prompt(at(ctx, prereqValue), "%v:0: <- missing file\n", prereqFile.fullname())

            if m := prereqFile.filemap; m != nil {
                note(at(ctx, m.pattern), "%v ⇒ %v ⇒ %v", targetValue, m, prereqFile).debug()
            }

            for i, s := range pc.traves { ctx := at(ctx, s.pos)
                note(ctx, "%v → traves[%d] ⇒ %v", targetValue, i, s).debug()
            }
            for i, concrete := range concreteList { ctx := at(ctx, concrete)
                note(ctx, "%v → concrete[%d] ⇒ %v", targetValue, i, concrete).debug()
            }
            for i, stemmed := range stemmedList  { ctx := at(ctx, stemmed)
                note(ctx, "%v → stemmed[%d] ⇒ %v", targetValue, i, stemmed).debug()
            }

            erro(at(ctx, prereqValue), "%v ⇒ %v", targetValue, prereqValue).debug(2)
        }

        if d := time.Now().Sub(t0); d > 60*time.Second {
            for i, c := range concreteList { warn(at(ctx,c), "%v : C#%d %v", targetValue, i, c) }
            for i, s := range stemmedList  { warn(at(ctx,s), "%v : S#%d %v", targetValue, i, s) }
            warnstack(ctx, 5, "slow: %v: %v: %v", targetValue, prereqValue, d).debug(10)
            flush(ctx)
        }
    } (time.Now())

    if targetValue = getTargetValue(ctx); targetValue == nil {
        erro(at(ctx,prereqValue), "%s: target is nil\n", prereqFinal).debug()
        return
    } else if isTrivial(targetValue) {
        erro(at(ctx,prereqValue), "%s: target is trivial (%T)\n", prereqFinal, targetValue).debug()
        return
    }

    var projs = []*project{ pc.prog.project } //ctx.projects(ctx)

    if len(projs) == 0 {
        note(at(ctx,targetValue), "%v", closure_projects(ctx))
        erro(at(ctx,targetValue), "%v: %v → %v: no projects", pc.prog.project, targetValue, prereqValue).debug()
        return
    }

    prereqValue, prereqPattern, prereqFinal, prereqFile, prereqObj =
        probPrereqValue(ctx, projs, prereqValue)

    if db(ctx, "traverse") || (true && (
        // strings.HasPrefix(targetValue.string(ctx), "HAVE_PTHREAD_IN_LIBC") ||
        // strings.HasPrefix(targetValue.string(ctx), "HAVE_LIBPTHREAD") ||
        false)) {

        if f, y := targetValue.(*File); y {
            prompt(ctx, "%v:0: @\n", f.fullname()).debug()
        } else {
            var s, _, _ = entryIndicator(ctx, targetValue)
            prompt(ctx, "%v : %v(%v)\n", s, typeof(prereqValue), prereqFinal).debug()
        }

        note(at(ctx,targetValue), "@: %v(%v) %v(%v)",
            typeof(targetValue), targetValue,
            typeof(prereqValue), prereqValue).debug()

        if prereqFile != nil { if false { s := prereqFile.fullname()
            note(ctx, ">: %T %v ⇒ %v", prereqValue, prereqValue, s).debug()
        }} else if f, y := prereqValue.(*File); y {
            note(at(ctx, f.position), "%v %v", f, f.exists()).debug()
        } else if f := file(ctx, prereqFinal); f == nil {
            var a = unmap_files(ctx, prereqFinal)
            var b = files(ctx, prereqFinal, _project(ctx))
            note(ctx, ">: %T %v ⇒ file: %v", prereqValue, prereqValue, a)
            note(ctx, ">: %T %v ⇒ file: %v", prereqValue, prereqValue, b).debug()
            if p := _project(ctx); false {
                warn(ctx, ">: %T %v ⇒ file: %v", prereqValue, prereqValue, p.selectFiles(ctx, a))
                warn(ctx, ">: %T %v ⇒ file: %v", prereqValue, prereqValue, p.selectFiles(ctx, b))
                warn(ctx, ">: %T %v ⇒ file: %v", prereqValue, prereqValue, p.selectFile(ctx, a))
                warn(ctx, ">: %T %v ⇒ file: %v", prereqValue, prereqValue, p.selectFile(ctx, b))
                for i, m := range a { warn(at(ctx, f.position), "%v: %d. %v: %v %v", p, i, m.project, m.name, m.pattern) }
                for i, m := range b { warn(at(ctx, f.position), "%v: %d. %v: %v %v", p, i, m.project, m.name, m.pattern) }
            }
        } else {
            note(ctx, ">: %T %v ⇒ %v", prereqValue, prereqValue, f).debug()
        }

        defer func() {
            var s = targetValue.string(ctx)
            for i, concrete := range concreteList { note(at(ctx,concrete), "%v : concrete: %d. %v", targetValue, i, concrete).debug() }
            for i, stemmed := range stemmedList { note(at(ctx,stemmed), "%v : stemmed: %d. %v", targetValue, i, stemmed).debug() }
            for i, t := range pc.traves { note(at(ctx, t.pos), "%v: %d. %v", s, i, t).debug() }
            notestack(ctx, 5).debug(2)
        } ()

        _db = true
    }

    if f := prereqFile; f != nil {
        if f._travin += 1; f._travin > 1 { return }
    }

    // Recursion detection -- simply return to break it if looped.
    if traverseDetectLoops {
        if eq(ctx, targetValue, prereqValue) {
            prompt(ctx, "%v: %v: self dependency, consider using [(once)] to avoid\n", targetValue, prereqValue)
            warn(at(ctx,prereqValue), "recursion: %T %v", prereqValue, prereqValue)
            warn(at(ctx,targetValue), "recursion: %T %v", targetValue, targetValue)
            warn(ctx, "recursion: %v : %v ; in %v", targetValue, prereqFile, projs)
            if false {
                warnstack(ctx, 16, "").debug(32)
            } else {
                errostack(ctx, 16, "").debug(32)
            }
            return
        }
        for c := pc; c != nil; c = c.caller() {
            if val := auto_get(c, "@"); val != nil && eq(c, val, prereqValue) {
                if traverseLoopBreakState != traveUnkn {
                    var s = pc.traves.add(ctx, traverseLoopBreakState, targetValue)
                    if s.dependPat = prereqPattern; prereqFile == nil {
                        s.depend = prereqValue
                    } else {
                        s.depend = prereqFile
                    }
                }

                var f = as{targetValue}.file(ctx, projs...)
                if true && f == nil {
                    prompt(ctx, "%v: %v: recursion detected, consider using [(once)] to avoid\n", targetValue, prereqValue)
                    warn(ctx, "recursion: %T %v", prereqValue, prereqValue)//.of(prereqValue)
                    warn(ctx, "recursion: %T %v", targetValue, targetValue)//.of(targetValue)
                    warn(ctx, "recursion: %v : %v ; in %v", targetValue, prereqFile, projs)
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
            if f, y := toFile(s.target); y && f.ident(ctx) == prereqFinal { res = f }
            return
        }
    }

    type traveResT int
    const (
        traveResContinue traveResT = iota
        traveResBreak
        traveResReturn
    )
    var traverseEntry = func(project *project, entry entry, pattern bool) (result traveResT) {
        if false && _db { defer func() { note(ctx, "%v: %v", entry, pc.traves).debug(2) } () }

        entry.traverse(ctx) // NOTE: this adds traveRule to pc.traves

        var t = pc.traves
        if !t.has() { return traveResContinue }

        // NOTE: collect travestates from t according to each trave type

        if tt := t.of(traveFail); tt.has() {
            if !pattern {
                var stems = _stems(ctx)
                prompt(ctx, "%v: traverse entry failed (%v)\n", entry, project)
                warn(at(ctx,entry), "%v: %v: %v (stems=%v)", entry, targetValue, prereqValue, stems)
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
                    warn(at(ctx,entry), "%v (%T) (by %v, in %v)", entry, entry, targetValue, entry.owner()).debug()
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
                    } else if true && _db {
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
            if sc := _stemmed_context(ctx); sc != nil && sc.stem != nil && sc.stem.target != nil {
                stemmedThisTarget = eq(ctx, sc.stem.target, targetValue)
            }
            for _, s := range tt {
                if g := s.target; g == nil {
                    prompt(ctx, "%v: %v\n", targetValue.string(ctx), t)
                    erro(ctx, "%v: %v %v %v\n", targetValue.string(ctx), prereqPattern, prereqValue, pattern)
                    errostack(ctx, 5, "").debug()
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
                    if !hasRecipes(entry) { op = traveResContinue }
                    return op
                } else if _, y := prereqValue.(*strlit); false && y {
                    if prereqFile == nil { prompt(ctx, "nonfile: %T %v: %T %v : %T %v ; %v\n",
                        targetValue, targetValue, prereqValue, prereqValue, s.target, s.target, s).debug() }
                }
            }
        }
        return
    }

    t1 = time.Now()

    for _, proj := range projs {
        var entries = proj.resolveEntries(ctx, prereqValue, false)
        if false && _db {
            note(ctx, "entries: %T %v ⇒ %v (%v)", prereqValue, prereqValue, entries, proj).debug()
        }
        if len(entries) == 0 { continue }
        concreteList = append(concreteList, entries...)
    ForEntries:
        for _, entry := range entries {
            if !isNull(entry) && targetValue == entry { continue ForEntries }
            if w, k := targetValue.(*bareword); k && w.s == prereqFinal {
                continue ForEntries // target resolve to itself, does nothing
            }
            switch traverseEntry(proj, entry, false) {
            case traveResBreak : break ForEntries
            case traveResReturn: return //if ok { return } else { break ForEntries }
            }
        }
    }
    if d := time.Now().Sub(t1); d > 60*time.Second {
        for _, concrete := range concreteList { prompt(ctx, "%v: slow: %v %v\n", concrete.Position(), concrete, targetValue) }
        prompt(ctx, "%v: slow: %v: %v %v (%d concretes)\n", _position(ctx), targetValue, prereqValue, d, len(concreteList)).debug()
    }

    if prereqFile != nil && prereqFile.exists() { goto CheckPrereqResult }

    t2 = time.Now()

    for _, proj := range projs {
        for _, p := range proj.patterns { assert(p.target.patterned(ctx), "not pattern") }
        var patterns = proj.resolvePatterns(ctx, prereqValue, prereqFinal)
        if len(patterns) == 0 { continue }
        stemmedList = append(stemmedList, patterns...)
    ForPatterns:
        for _, entry := range patterns {
            switch traverseEntry(proj, entry, true) {
            case traveResBreak : break ForPatterns
            case traveResReturn: return //if ok { return } else { break ForPatterns }
            }
        }
    }
    if d := time.Now().Sub(t2); d > 60*time.Second {
        for _, stemmed  := range stemmedList { prompt(ctx, "%v: slow: %v\n", stemmed.Position(), stemmed) }
        prompt(ctx, "%v: slow: %v: %v %v (%d stemmed)\n", _position(ctx), targetValue, prereqValue, d, len(stemmedList)).debug()
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
        if p = stat(ctx, prereqFinal); p != nil {
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
        if t[0].depend.(entry).ident(ctx) == prereqFinal { return }
    }

    if t := pc.traves.of(traveObj); t.has() && t[0].depend != nil {
        if t[0].depend.(Object).ident(ctx) == prereqFinal { return }
    }

    if prereqPattern != nil {
        s := pc.traves.add(ctx, traveNext, targetValue)
        s.dependPat = prereqPattern
        s.depend = prereqValue
        return
    }

    if isConfigure(ctx) { return }

    if len(_stems(ctx)) == 0 || cast[*modifier_deps](ctx) != nil {
        if s := pc.traves.add(ctx, traveFail, targetValue); prereqFile != nil {
            s.error = failureFileNotFound{_project(ctx), prereqFile}
            ctx = at(ctx, prereqFile.position)
        } else {
            s.error = failureTargetNotFound{_project(ctx), prereqFinal}
            if prereqValue != nil { ctx = at(ctx, prereqValue) }
        }

        if prereqFile != nil && prereqValue != prereqFile {
            note(ctx, "%v: %v; file=%v\n", ts(targetValue), ts(prereqValue), prereqFile).debug()
        } else if prereqFile != nil {
            note(ctx, "%v: %v; path=%s\n", ts(targetValue), ts(prereqValue), prereqFile.fullname()).debug()
        } else if prereqObj != nil {
            note(ctx, "%v: %v; obj=%v\n", ts(targetValue), ts(prereqValue), prereqObj).debug()
        } else {
            note(ctx, "%v: %v\n", ts(targetValue), ts(prereqValue)).debug()
        }

        if val := prereqValue; val != nil { erro(at(ctx,val), "value: %v ; %s %v", ts(val), prereqFinal, files(ctx, prereqFinal)) }
        if fil := prereqFile;  fil != nil { erro(at(ctx,fil), "file: %v, exists=%v", fil, fil.exists()) }
        if obj := prereqObj;   obj != nil { erro(at(ctx,obj), "object: %v", ts(obj)) }
        for i, s := range pc.traves { erro(at(ctx,s.pos), "trave.%d: %v: %v: %v", i, targetValue, prereqValue, s) }
        for i, c := range closure_scopes(ctx) { erro(ctx, "closure.%d: %v", i, c) }
        for i, concrete := range concreteList { erro(at(ctx,concrete), "concrete: %d. %v (%d programs)", i, concrete, len(concrete.programs())) }
        for i, stemmed  := range stemmedList  { erro(at(ctx,stemmed), "stemmed: %d. %v", i, stemmed) }
        errostack(at(ctx, prereqValue), 6).debug(512)
        return
    }

    return // no operation
}

func (pc *execution) prerequisite(ctx Context, prerequisites []Value) {
    var (
        uni = _universe(ctx)
        verb = uni.verbose || uni.verboseBreaks
        ent  = auto_get(ctx, "@")
        stem *stemmed
        depends valvec
        by Value
    )
    defer func() {
        pc.Wait()

        var target Value = auto_get(ctx, "@")

        // NOTE: warnings if travestates go too large
        if true {
            // does nothing
        } else if tt := pc.traves.of(traveObj, traveRule, traveFile); len(tt) > 100 {
            prompt(ctx, "%v: traves=%d\n", target, len(pc.traves))
            for i, s := range pc.traves {
                prompt(ctx, "%v: %d. %v\n", target, i, s)
                if i > 10 { break }
            }
            warnstack(ctx, 5, "%d errors", flush(ctx)).debug(16)
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
            erro(at(ctx, by), "%v: at %v", f.name, by)
            erro(ctx, "%v: target not exists (%s, %v)", f.name, f.fullname(), f.stat(ctx))
            errostack(ctx, 5).debug(16)
        }

        // FIXME: optimization: the pc.traves may grow into large number of traveFile
    } ()

    if sc := _stemmed_context(ctx); sc != nil { stem = sc.stem }

ForPrerequisites:
    for _, prerequisite := range prerequisites {
        var (
            ctx = at(ctx, prerequisite)
            k = len(pc.traves)
            t0 = time.Now()
        )
        switch by = prerequisite; u := prerequisite.(type) {
        case untraversed: note(ctx, "%v: %v", ent, ts(u)).debug() ; continue
        default: prerequisite.traverse(ctx)
        }

        var traves = pc.traves.slice(k)
        var target = auto_get(ctx, "@") // fetch updated $@
        var depend = auto_get(ctx, ">") // fetch updated $>
        var isPatternStemmedForTarget = stem != nil && stem.target != nil &&
            eq(ctx, stem.target, target)

        if depend != nil {
            depends.add(depend)
        } else if _, y := prerequisite.(*modification); y {
            // noop
        } else if p, y := prerequisite.(*delegate); y && is(p.x, KindUse) {
            // noop
        } else if p := prerequisite.expand(ctx); p == nil {//, final/* |expandUnexpanded */
            // noop
        }

        if d, f := time.Now().Sub(t0), time.Duration(pc.countFiles); d > 15*time.Second &&
            (f == 0 || (d/f > 10*time.Millisecond)) {
            var p = prerequisite
            var a = target
            var t = auto_get(ctx, "^")
            var c = auto_get(ctx, "<")
            var q = depend
            var pos = _position(ctx)
            if f != 0 {
                prompt(ctx, "%v: program.traverse slow: %v %v %d\n", pos, d, d/f, pc.countFiles)
            } else {
                prompt(ctx, "%v: program.traverse slow: %v %d\n", pos, d, pc.countFiles)
            }
            prompt(ctx, "%v: program.traverse %v: @: %v\n", pos, p, a)
            prompt(ctx, "%v: program.traverse %v: ^: %v\n", pos, p, t)
            prompt(ctx, "%v: program.traverse %v: <: %v\n", pos, p, c)
            prompt(ctx, "%v: program.traverse %v: >: %v\n", pos, p, q)
            infostack(ctx, 3).debug()
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

        if tt := traves.of(traveFail); tt.has() {
            if /* isPatternStemmedForTarget */true {
                for _, s := range tt {
                    var dependMine = depends.has(s.depend)
                    if s.error == traveTargetNotDefinedFile && dependMine {
                        // add traveNext to try the next pattern
                        pc.traves.remove(s).add(ctx, traveNext, target)
                    } else if depend == nil {
                        prompt(ctx, "%v: %v\n", target, s).debug()
                        erro(at(ctx,s.pos), "%v", s)
                        erro(at(ctx,target), "1. %T %v %s", target, target, target.string(ctx))
                        erro(at(ctx,s.depend), "2. %T %v mine=%v", s.depend, s.depend, dependMine)
                        erro(at(ctx,prerequisite), "3. %T %v", prerequisite, prerequisite)
                        errostack(ctx, 5, "#>").debug(10)
                    } else {
                        prompt(ctx, "%v: %v: %v\n", target, depend, s).debug()
                        erro(at(ctx,s.pos), "%v", s)
                        erro(at(ctx,target), "1. %T %v %s", target, target, target.string(ctx))
                        erro(at(ctx,depend), "2. %T %v %s", depend, depend, depend.string(ctx))
                        if s.depend == nil { erro(ctx, "3. mine=%v", dependMine) } else {
                            erro(at(ctx,s.depend), "3. %T %v mine=%v", s.depend, s.depend, dependMine)
                        }
                        erro(at(ctx,prerequisite), "4. %T %v", prerequisite, prerequisite)
                        errostack(ctx, 5, "#>").debug(10)
                    }
                    return
                }
            }

            if verb {
                var p = _project(ctx)
                prompt(ctx, "%v:(%T): %T %v ; project=%s, stems=%v ; %v\n",
                    pc.traves, target, prerequisite, prerequisite, p, _stems(ctx), stem)

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
                    if false { info(at(ctx,prerequisite), "%v: %T %v ; %v %v",
                        target, prerequisite, prerequisite, s.dependPat, s.depend).debug() }
                    return // end this pattern entry to let trying next one
                }
            }

            var _, isPP = prerequisite.(*percpat)
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

func _program(ctx Context) (_ *program) {
    if p := _execution(ctx); p != nil {
        return p.prog
    }
    return
}

type program struct {
    position Position
    project *project
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
            if m.elems[0].string(ctx) == name { ms = append(ms, m) }
        }}
    }
    return
}

const maxCallRecursion  = 32 //64

type traverse_context struct { Context }
type order_traverse_context struct { Context }
func (t traverse_context) traversed(ctx Context, target Value) (targets []Value) {
    if targets, _ = do(t.Context, act_traversed{target}).([]Value); len(targets) > 0 {
        auto_set(ctx, "^", makeList(targets...))
        auto_set(ctx, "<", targets[0])
        auto_set(ctx, ">", targets[len(targets)-1])
    }
    if false { note(ctx, "%v %v", target, targets).debug() }
    return
}
func (t order_traverse_context) traversed(ctx Context, target Value) (targets []Value) {
    if targets, _ = do(t.Context, act_traversed{target}).([]Value); len(targets) > 0 {
        auto_set(ctx, "|", makeList(targets...))
    }
    if false { note(ctx, "%v %v", target, targets).debug() }
    return
}

func (prog *program) workdir(ctx Context) (workdir string) {
    if pc := _execution(ctx); pc == nil {
        workdir = prog.project.absPath
    } else if pc.changedWD == "" {
        var o Object
        if o = prog.project.resolve(ctx, "CWD"); isTrivial(o) {
            if o = prog.project.resolve(ctx, "/"); isTrivial(o) {
                erro(ctx, "both $(CWD) and $/ are trivial").debug()
                return
            }
        }
        if v := o.expand(ctx); v != nil {//, final
            workdir = v.string(ctx)
        } else {
            erro(ctx, "trivial %v", ts(o)).debug()
        }
    } else if filepath.IsAbs(pc.changedWD) {
        workdir = pc.changedWD
    } else {
        workdir = filepath.Join(prog.project.absPath, pc.changedWD)
    }
    return
}

func (prog *program) execute(ctx Context) (result Value, _traves travestates) {
    if true && checkpoints && truly(ctx, is_test_mode{}) { defer func(){
        if result == nil {
            note(ctx, "nil result; %v", ts(ctx)).debug(2)
        } else {
            note(ctx, "%v → %v, %v %v %v", result, result.expand(final{ctx}), auto_find(ctx, "@"), auto_find(ctx, "<"), auto_find(ctx, ">"))
            note(ctx, "%v", ts(ctx)).debug(2)
        }
    }()}

    var ent = _entry(ctx)

    if 0 < count_error(ctx) {
        erro(ctx, "%v: got errors, execution canceled", ent).debug()
        return
    }

    defer trace(ctx)

    var pc = execution{
        automatic: automatic{ Context:ctx, defs:make(auto_defs) },
        execRec: make(map[Value]int), start: time.Now(), prog: prog,
    }

    ctx = &pc

    defer func() {
        var targets = auto_get(ctx, "@") // depends = auto_get(ctx, "^")
        if isTrivial(targets) { targets = ent.destiny() }

        if 0 < count_error(ctx) {
            return
        }

        for _, a := range pc.defers {
            if g, y := a.(*group); y {
                modify(ctx, g, true)
            } else {
                erro(at(ctx, a), "defer: not a modifier: {=%s %v}", typeof(a), a).debug()
                return
            }
        }

        if result == nil {
            if result = auto_get(ctx, "-"); result == nil {
                result = pc.defaultVal
            }
        }
        if result != nil {
            if t := pc.caller(); t != nil {
                t.defaultVal = result
            }
        }

        pc.defaultVal = nil
        _traves = pc.traves
    } ()

    if true {
        var cc = pc.Context
        var a = []Value{ auto_get(cc, "@") }
        var depth, loop int = 0, -1

    ForPC:
        for c := pc.caller(); c != nil; c = c.caller() { if _program(c) == prog {
            if depth += 1; depth == maxCallRecursion { break ForPC }
            var t = auto_get(c, "@")
            for i, v := range a { if eq(ctx, t, v) { loop = i; break ForPC } }
            if loop < 0 { a = append(a, t) }
        }}

        if 0 <= loop { var t = auto_get(cc, "@")
            if o := cast[*terminal](cc); o != nil { if v := auto_get(o, "@"); v != nil && eq(cc, v, t) {
                if true { warnstack(ctx, 3, "skip closure loop: %v %v", o, t).debug() }
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

            prompt(ctx, "%v: %v: %v, %v\n", a[0], auto_get(cast[*terminal](cc), "@"), cc, cast[*terminal](cc))
            for i, t := range a { erro(at(ctx,t), "loop: %v: %v", i, t) }
            errostack(at(ctx,prog.position), 128, "loop, (depth=%d, %v, %v)\n", depth, a[loop], a).debug(6)
            return
        }

        if depth < maxCallRecursion {
            // continues
        } else if c := pc.caller(); c != nil { pc.traceLevel = c.traceLevel
            var tt = as{auto_get(c, "@")}
            var s, _ = tt.fullnameOrFinal(ctx)
            prompt(ctx, "%v: max recursion call (%d)\n", s, depth)
            warn(at(ctx,tt), "max recursion call (%d)\n", depth).debug()

            const collapse = false
            for ; c != nil; c = c.caller() { var n int
                if collapse { for next := c.caller(); next != nil; next = next.caller() {
                    if d := auto_get(next, "@"); d == nil { continue } else
                    if t := d; t != nil && eq(ctx, t, tt) { n += 1;  continue }
                    if _program(next) == _program(c) { n += 1; c = next } else { break }
                }}

                if prog, t := _program(c), auto_get(c, "@"); prog == nil {
                    erro(at(ctx,ent), "%v (@=%v)", ent, tt)
                    break
                } else if pos := prog.position; n > 0 {
                    erro(at(ctx,pos), "%v (repeated %d times)", t, n)
                } else if !collapse {
                    erro(at(ctx,pos), "%v : %v", t, auto_get(c, ">"))
                } else if depth -= 1; maxCallRecursion - depth > 5 {
                    erro(at(ctx,pos), "%v ... (%d)", t, maxCallRecursion - depth)
                    break
                } else {
                    erro(at(ctx,pos), "%v : %v", t, auto_get(c, ">"))
                }

                flush(ctx) // dump immediately
            }

            errostack(ctx, depth, "#>", ent).debug(512)
            if false { panic(_failure(ctx, "max call depth")) }
            return
        }

        if t := _stems(ctx); t != nil { auto_set(ctx, "*", ease(ctx, t)) }
    }

    var u = _universe(ctx)

    // NOTE: set "@" before setting auto args
    // Select the right target value before setting parameters,
    // because the target could be overrided by parameters.
    if target := ent.destiny(); target == nil {
        erro(ctx, "%v: nil entry target", target).debug()
        return
    } else {
        switch a := target.(type) {
        case *strlit, *compound: // NOTE: skip strings to optimize speed from searching
        case fullfile: if a._traved > 1 { return } // alreadyUpdated = a.info != nil && a.updated
        case    *File: if a._traved > 1 { return } // alreadyUpdated = a.info != nil && a.updated
        default:
            if file := prog.project.file(ctx, a.string(ctx)); file != nil {
                if file._traved > 1 { return } else { target = file }
            }
        }

        if pc.execRec[target] += 1; false { if pc.execRec[target] > 1 {
            if u.traceExec { l_exec.trace(fmt.Sprintf("exec: %v", target)) }
            return
        }}

        auto_set(ctx, "@", target)
    }

    do(ctx, act_arguments{})

    if u.traceExec {
        var d = pc.depth()
        var t = auto_get(ctx, "@")
        var s = fmt.Sprintf("%s: %v (%p, exec.depth=%d)", typeof(t), t, t, d)
        defer un(l_trace(l_exec, s))
    }

    auto_set(ctx, "^", nil)
    auto_set(ctx, "<", nil)
    auto_set(ctx, ">", nil)
    auto_set(ctx, "|", nil)

    // Update normal prerequisites
    pc.prerequisite(traverse_context{ctx}, prog.depends)
    if errs := count_error(ctx); errs > 0 {
        s := pc.traves.add(at(ctx, prog.position), traveFail, nil)
        s.error = fmt.Errorf("%d errors counted", errs)
        return
    } else if pc.traves.has(traveCase, traveDone, traveFail, traveNext) {
        return
    }

    // Update order-only prerequisites
    pc.prerequisite(order_traverse_context{ctx}, prog.ordered)
    if errs := count_error(ctx); errs > 0 {
        s := pc.traves.add(at(ctx, prog.position), traveFail, nil)
        s.error = fmt.Errorf("%d errors counted", errs)
        return
    } else if pc.traves.has(traveCase, traveDone, traveFail, traveNext) {
        return
    }

    if prog.language != "" || len(prog.recipes) == 0 {
        return
    }

    if h := auto_get(ctx,"-"); h == nil || len(pc.interpreted) == 0 {
        if i, y := dialects["eval"]; y && i != nil {
            pc.interpret(ctx, i, nil)
        } else {
            erro(ctx, "no default dialect").debug()
            return
        }
    }
    return
}
