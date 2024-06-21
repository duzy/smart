//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "path/filepath"
    "reflect"
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

type act_exe_value struct{ v Value }
type get_program struct{}
type default_value struct{ v Value }

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

    recs map[Value]int

    calleeErrs []error
    calleeErrsM sync.Mutex

    grepping bool
    grepped []Value
    targets []Value // all targets def
    ordered []Value
    order bool

    countFiles int

    traceLevel int

    interpreted []interpreter
}
func (p *execution) inner() Context { return &p.automatic }
func (p *execution) caller() *execution { return _execution(p.Context) }
func (p *execution) cast(t reflect.Type) Context {
    if reflect.TypeOf(p) == t { return p }
    if reflect.TypeOf((*argumented_context)(nil)) == t { return nil }
    if reflect.TypeOf((*stemmed_context)(nil)) == t { return nil }
    return p.automatic.cast(t)
}
func (p *execution) do(ctx Context, op any) (res any) {
    switch t := op.(type) {
    case get_program: return p.prog
    case get_position: if p.prog != nil { return p.prog.position }
    case get_project: if p.prog.project != nil { return p.prog.project }
    case get_scope: if p.prog.project != nil { return p.prog.project.scope }

    case default_value: p.defaultVal = t.v
    case act_exe_value: p.values = append(p.values, t.v)
    case act_dirty_mark:       p.dirty_mark(t.a...)
    case act_dirt:      return p.dirty(ctx,t.a...)
    case act_traversed: return p.traversed(ctx,t.v)
    case act_traverse: p.traverse(ctx, t.v); return

    case get_param_name:
        if t.i < len(p.prog.params) {
            res = p.prog.params[t.i].name
        }
        return

    case property:
        if t&propExAuto != 0 { return true } //_, y = rule_autos[x.(*auto).name]
        if t&propDirtyOpts != 0 { return &p.by }
        if t&propParameters != 0 {
            var params map[string]*auto // TODO: store map[string]*auto in program
            for _, param := range p.prog.params {
                if params == nil { params = make(map[string]*auto, len(p.prog.params)) }
                params[param.name] = param
            }
            return params
        }
    }
    return p.automatic.do(ctx, op)
}

func (p *execution) ts(t string) string {
	return fmt.Sprintf("{=%s %v}", t, ts(p.Context)) // NOTE: hides {=automatic}
}

func (p *execution) aquire() func() { p.Lock() ; return func(){ p.Unlock() }}

func (p *execution) depth() (res int) {
    for c := p.caller(); c != nil; c = c.caller() { res += 1 }
    return
}
func (p *execution) calleeError(err error) {
    if err != nil {
        p.calleeErrsM.Lock()
        p.calleeErrs = append(p.calleeErrs, err)
        p.calleeErrsM.Unlock()
    }
}

// traverse_context is a single thread traverse context, for traversing in a new goroutine,
// a spawned traversal must be used and then merge.
func (p *execution) level(n int) { p.traceLevel += n }
func (p *execution) trace(a ...interface{}) { printIndentDots(p.traceLevel, a...) }
func (p *execution) tracef(s string, a ...interface{}) { printIndentDots(p.traceLevel, fmt.Sprintf(s, a...)) }

func (p *execution) traversed(ctx Context, target Value) []Value {
    if !isTrivial(target) {
        if _, y := toFile(target); y { p.addFilesCount(1) }
        if p.order {
            p.ordered = append(p.ordered, target)
            auto_set(ctx, "|", makeList(p.ordered...))
        } else {
            p.targets = append(p.targets, target)
            auto_set(ctx, "^", makeList(p.targets...))
            auto_set(ctx, "<", p.targets[0])
            auto_set(ctx, ">", p.targets[len(p.targets)-1])
        }
    }
    return p.targets
}

func (p *execution) addFilesCount(n int) {
    p.countFiles += n
    if c := p.caller(); c != nil { c.addFilesCount(1) }
}

func (p *execution) env(ctx Context) (env []string, osi int) {
    env = os.Environ()
    osi = len(env)
    for _, p := range p._env {
        var k, v = p.key.string(ctx), p.val.string(ctx)
        env = append(env, fmt.Sprintf("%s=%s", k, v)) // strconv.Quote(v)
    }
    return
}

func (p *execution) dirty_mark(vals ...Value) {
    const (
        enableDirtyMark = true
        perUpdatedDep = true
    )
    if !enableDirtyMark {
        // does nothing
    } else if targets := merge(auto_get(p, "@")); len(targets) == 0 {
        // should not happen, but safely ignoring..
    } else if len(vals) == 0 {
        vals = append(vals, targets...)
    } else if true {
        vals = merge(vals...)

        var mat, dup bool
        var opts = &p.by
        for _, t := range targets {
        ForVals:
            for _, val := range vals {
                if eq(p, val, t) {
                    dup = true; continue ForVals
                }
                for _, pat := range opts.pats {
                    if mat, _, _ = pat.match(p, val); mat {
                        if perUpdatedDep { t.updatedDeps(p, val) }
                        break ForVals
                    }
                }
            }
            if !perUpdatedDep && mat { t.updatedDeps(p, vals...) }
            if !dup { // vals = append(vals, merge(targets)...)
                vals = append(t.updatedDeps(p), vals...)
                vals = append(merge(t), vals...)
            }
        }
    }
    if enableDirtyMark { p.dirty_mark(vals...) }
}

func (p *execution) interpret(ctx Context, i interpreter, params []Value) (res Value) {
    var target, _, _ = wait(ctx, waitopts{
        ReportUpdates: false,
        ExecResults: false,
        StampCurrentTarget: false,
    })

    if !isConfigure(ctx) {
        if _, y := target.(*File); y && p != nil && !p.dirty(ctx) {
            // p.traves.add(ctx, traveDone, nil) // NOTE: modifier.predictDirty
            return
        }
    }

    if res = i.evaluate(ctx, params...); res != nil {
        if d, prev := auto_set(ctx, "-", res); d == nil {
            var _, ent, _ = entryIndicator(ctx, _entry(ctx))
            prompt(ctx, "%v: %s\n", ent, intername(i))
            erro(ctx, "set buffer value failed: %v -> %v", prev, res)
            errostack(ctx, 3).trace()
        }
    }

    p.interpreted = append(p.interpreted, i)

    if _, _, err := updateRecipesHash(ctx, target); err != nil {
        var _, ent, _ = entryIndicator(ctx, _entry(ctx))
        prompt(ctx, "%v: %s\n", ent, intername(i))
        erro(ctx, "update recipes hash failed: %v", err)
        errostack(ctx, 3).trace()
    }
    return
}

func isDirty(ctx Context, target Value, a ...Value) (dirty bool) {
    var opts, y = do(ctx, propDirtyOpts).(*dirtyOpts)
    if !y {
        erro(ctx, "nil dirtyopts : %v", ts(ctx)).trace()
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

func (p *execution) dirty(ctx Context, aa ...Value) (outdated bool) {
    var target as

    target.Value, /*files*/_, /*execRes*/_ = wait(p, waitopts{
        ReportUpdates: false,
        ExecResults: false,
        StampCurrentTarget: false,
    })

    var reason string
    var targetFile *File
    var targetFull string
    if false { if s := target.string(ctx); s != "" && (
        // s == "tablegen-min" ||
        false) {
        defer func() {
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
        erro(ctx, "recipes changed: %v", e).trace()
        return
    } else if y {
        outdated, reason = true, "recipes changed"
    } else if !opts.checksum {
        // does nothing
    } else {
        erro(ctx, "FIXME: check prerequisites against the saved checksums").trace()
        return
    }

    verb = verb ||
        (opts.verboseOutdated && outdated) ||
        (opts.verboseUpdated && !outdated)

    if verb {
        var d = time.Now().Sub(p.start)
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

        var n = p.countFiles + len(p.grepped)
        if db := opts.debug>0; db && !opts.verbose {
            warn(ctx, "%s (%T) (%s) …… %s (%d files in %s, debug=%d)", ts, target, targetFull, m, n, s, opts.debug).debug(opts.debug * 2)
        } else {
            prompt(ctx, "%s …… %s (%d files in %s)\n", ts, m, n, s).debug(db, 6)
        }
    }

    if outdated && p.dirt != "" { reason = p.dirt + "; " + reason }
    if !opts.silent && reason != "" { p.dirt = reason }
    return
}

func probPrereqValue(ctx Context, projects []*project, val Value) (prereqValue, prereqPattern Value, prereqFinal string, prereqFile *File) {
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
            errostack(ctx, 3, "prerequisite is nothing").trace()
        }

        mapPrereqFile(prereqFinal)
        return
    }

    if _, y := prereqValue.(Object); y { return }

    if !prereqValue.patterned(ctx) {
        prereqFinal = prereqValue.string(ctx)

        if prereqFinal == "" { // just reject empty final
            errostack(ctx, 3, "%v: %v: empty prerequisite, stems=%v", prereqValue, _stems(ctx)).trace()
        }

        switch prereqValue.(type) {
        case flag, *strlit, *compound:
            return // skip checking files for performance
        }
        for _, p := range _entry(ctx).programs() {
            if p.configure { return }
        }

        mapPrereqFile(prereqValue)
        return
    }

    var stems = _stems(ctx)
    if len(stems) == 0 {
        if false { errostack(ctx, 3, "%v: no stems, %v", prereqValue, ctx).trace() }
        return
    }

    var rest []string
    prereqPattern = prereqValue
    prereqValue, rest = prereqPattern.stencil(ctx, stems)
    if isTrivial(prereqValue) {
        errostack(ctx, 3, "%v: empty stencil with %v", prereqPattern, stems).trace()
    } else if len(rest) > 0 {
        errostack(ctx, 3, "%v: partial stencil with %v, rest=%v", prereqPattern, stems, rest).trace()
    }

    if prereqFinal == "" { prereqFinal = prereqValue.string(ctx); }
    if prereqFinal == "" {
        errostack(ctx, 3, "%v: empty prerequisite, stems=%v", prereqValue, stems).trace()
    }

    mapPrereqFile(prereqValue)
    return
}

func (p *execution) defertrave(ctx Context, targetValue, prereqValue, prereqPattern Value, prereqFile *File) {
    var av = targetValue
    var bv = prereqValue
    if prereqFile == nil {
        do(ctx, act_traversed{prereqValue}) // set $< $> $^ or $|
    } else if targetValue != prereqFile {
        do(ctx, act_traversed{prereqFile}) // set $< $> $^ or $|
        bv = prereqFile
    }

    if !isTrivial(av) && !isTrivial(bv) {
        var a = av.stat(ctx).mod()
        var b = bv.stat(ctx).mod()
        if (!a.IsZero() && b.After(a)) || bv.updated(ctx) || bv.updatedDeps(ctx) != nil {
            av.updatedDeps(ctx, bv)
        }
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

func (p *execution) traverse(ctx Context, prereqValue Value) {
    var (
        targetValue Value

        prereqPattern Value
        prereqFinal string
        prereqFile *File

        concreteList []entry
        stemmedList []*stemmed

        t1, t2 time.Time
    )

    defer p.defertrave(ctx, targetValue, prereqValue, prereqPattern, prereqFile)

    if targetValue = getTargetValue(ctx); targetValue == nil {
        erro(at(ctx,prereqValue), "%s: target is nil\n", prereqFinal).trace()
    } else if isTrivial(targetValue) {
        erro(at(ctx,prereqValue), "%s: target is trivial (%T)\n", prereqFinal, targetValue).trace()
    }

    var projs = []*project{ p.prog.project } //ctx.projects(ctx)

    if len(projs) == 0 {
        note(at(ctx,targetValue), "%v", closure_projects(ctx))
        erro(at(ctx,targetValue), "%v: %v → %v: no projects", p.prog.project, targetValue, prereqValue).trace()
    }

    prereqValue, prereqPattern, prereqFinal, prereqFile = probPrereqValue(ctx, projs, prereqValue)

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
            errostack(ctx, 16).trace()
        }
        for c := p; c != nil; c = c.caller() {
            if val := auto_get(c, "@"); val != nil && eq(c, val, prereqValue) {
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

    if prereqFile != nil {
        if n := prereqFile._traved; n > 0 {
            return
        }
    }

    t1 = time.Now()

    for _, proj := range projs {
        var entries = proj.resolveEntries(ctx, prereqValue, false)
        if len(entries) == 0 { continue }
        concreteList = append(concreteList, entries...)
        for _, entry := range entries {
            if !isNull(entry) && targetValue == entry { continue }
            if w, k := targetValue.(*bareword); k && w.s == prereqFinal {
                continue // target resolve to itself, does nothing
            }
            entry.traverse(ctx)
        }
    }

    if d := time.Now().Sub(t1); d > 60*time.Second {
        for _, concrete := range concreteList {
            prompt(ctx, "%v: slow: %v %v\n", concrete.Position(), concrete, targetValue)
        }
        prompt(ctx, "%v: slow: %v: %v %v (%d concretes)\n", _position(ctx), targetValue, prereqValue, d, len(concreteList)).debug()
    }

    if prereqFile != nil && prereqFile.exists() {
        return
    }

    t2 = time.Now()

    for _, proj := range projs {
        for _, p := range proj.patterns { assert(p.target.patterned(ctx), "not pattern") }
        var patterns = proj.resolvePatterns(ctx, prereqValue, prereqFinal)
        if len(patterns) == 0 { continue }
        stemmedList = append(stemmedList, patterns...)
        for _, entry := range patterns {
            entry.traverse(ctx)
        }
    }

    if d := time.Now().Sub(t2); d > 60*time.Second {
        for _, stemmed  := range stemmedList { prompt(ctx, "%v: slow: %v\n", stemmed.Position(), stemmed) }
        prompt(ctx, "%v: slow: %v: %v %v (%d stemmed)\n", _position(ctx), targetValue, prereqValue, d, len(stemmedList)).debug()
    }

    return // no operation
}

func (p *execution) prerequisites(ctx Context, prerequisites []Value) {
    defer p.Wait()
    for _, prerequisite := range prerequisites {
        prerequisite.traverse(at(ctx, prerequisite))
    }
    return
}

func _program(ctx Context) (p *program) {
    p, _ = do(ctx, get_program{}).(*program)
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
        if g, y := d.(*modification); y {
            for _, m := range g.list {
                if m.elems[0].string(ctx) == name { ms = append(ms, m) }
            }
        }
    }
    return
}

const maxCallRecursion  = 32 //64

func (prog *program) workdir(ctx Context) (workdir string) {
    if p := _execution(ctx); p == nil {
        workdir = prog.project.absPath
    } else if p.changedWD == "" {
        var o Object
        if o = prog.project.resolve(ctx, "CWD"); isTrivial(o) {
            if o = prog.project.resolve(ctx, "/"); isTrivial(o) {
                erro(ctx, "both $(CWD) and $/ are trivial").trace()
            }
        }
        if v := o.expand(ctx); v != nil {//, final
            workdir = v.string(ctx)
        } else {
            erro(ctx, "trivial %v", ts(o)).trace()
        }
    } else if filepath.IsAbs(p.changedWD) {
        workdir = p.changedWD
    } else {
        workdir = filepath.Join(prog.project.absPath, p.changedWD)
    }
    return
}

func (prog *program) target(ctx Context, exe *execution) (target Value) {
    if target = _entry(ctx).destiny(); target == nil {
        erro(ctx, "%v: nil entry target", target).trace()
    }

    switch a := target.(type) {
    case *strlit, *compound: // NOTE: skip strings to optimize speed from searching
    case fullfile: if a._traved > 1 { return } // alreadyUpdated = a.info != nil && a.updated
    case    *File: if a._traved > 1 { return } // alreadyUpdated = a.info != nil && a.updated
    default:
        if f := prog.project.file(ctx, a.string(ctx)); f != nil {
            if f._traved > 1 { return } else { target = f }
        }
    }

    if exe.recs[target] += 1; false {
        if exe.recs[target] > 1 {
            if _universe(ctx).traceExec { l_exec.trace(fmt.Sprintf("exec: %v", target)) }
            return
        }
    }
    return
}

func (prog *program) check_exe(ctx Context, exe *execution) {
    var cc = exe.Context
    var a = []Value{ auto_get(cc, "@") }
    var depth, loop int = 0, -1

ForPC:
    for c := exe.caller(); c != nil; c = c.caller() {
        if _program(c) == prog {
            if depth += 1; depth == maxCallRecursion { break ForPC }
            var t = auto_get(c, "@")
            for i, v := range a {
                if eq(ctx, t, v) { loop = i; break ForPC }
            }
            if loop < 0 { a = append(a, t) }
        }
    }

    if 0 <= loop {
        var t = auto_get(cc, "@")
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
        errostack(at(ctx,prog.position), 128, "loop, (depth=%d, %v, %v)\n", depth, a[loop], a).trace()
    }

    if depth < maxCallRecursion {
        // continues
    } else if c := exe.caller(); c != nil {
        exe.traceLevel = c.traceLevel

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
                erro(ctx, "%v (@=%v)", _entry(ctx), tt)
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

        errostack(ctx, depth, "#>", _entry(ctx)).trace()
    }
}

func (prog *program) result_or_default_interpret(ctx *execution) (result Value) {
    if result = auto_get(ctx, "-"); result != nil {
        return
    }
    if len(ctx.interpreted) == 0 {
        if x, y := dialects[""]; y && x != nil {
            return ctx.interpret(ctx, x, nil)
        }
        erro(ctx, "no default dialect").trace()
    }
    return
}

func (prog *program) execute(ctx Context) (result Value) {
    if 0 < count_error(ctx) {
        erro(ctx, "%v: got errors, execution canceled", _entry(ctx)).trace()
    }

    var exe = execution{
        automatic:automatic{Context:ctx, defs:make(auto_defs)},
        prog:prog, recs:make(map[Value]int), start:time.Now(),
    }

    ctx = &exe

    if checkpoints && truly(ctx, is_test_mode{}) {
        if x := prog.checks(); x != nil { defer x(ctx, &result) }
    }

    defer func() {
        for _, a := range exe.defers {
            if x, y := a.(*group); y {
                modify(ctx, x, true)
            } else {
                erro(at(ctx,a), "defer: not a modifier: %s", ts(a)).trace()
            }
        }

        if result == nil { result = auto_get(ctx, "-") }
        if result == nil { result = exe.defaultVal }
        if result != nil { do(exe.Context, default_value{result}) }

        exe.defaultVal = nil
    } ()

    if true { prog.check_exe(ctx, &exe) }

    // NOTE: set "@" before setting auto args
    // Select the right target value before setting parameters,
    // because the target could be overrided by parameters.
    auto_set(ctx, "@", prog.target(ctx, &exe))

    if t := _stems(ctx); t != nil {
        auto_set(ctx, "*", ease(ctx, t))
    }

    do(ctx, act_arguments{})

    exe.order = false
    exe.prerequisites(ctx, prog.depends)

    exe.order = true
    exe.prerequisites(ctx, prog.ordered)

    if prog.language != "" { return }
    if len(prog.recipes) == 0 { return }
    return prog.result_or_default_interpret(&exe)
}
