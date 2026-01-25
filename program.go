//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "path/filepath"
    "reflect"
    "strings"
    "sync"
    "time"
    "fmt"
    "os"
)

type dirtyOpts struct {
    general_opts
    verboseUpdated  bool "verbose-updated"  // vu,
    verboseOutdated bool "verbose-outdated" // vo,
    checksum bool "checksum" // cs,crc,
    silent   bool "silent" // s,
    pats []Value
}

type default_value struct{ v Value }
type exe_res       struct{ v Value }
type get_program   struct{}

type is_ordered_prereq struct{}
type is_prerequisite   struct{}

func _execution(c Context) *execution { return cast[*execution](c) }

type execution_lang struct{}
type missing_file struct{ file string }

type interpret struct { name string ; args []Value }
type execution struct {
    automatic
    sync.Mutex
    sync.WaitGroup

    by dirtyOpts

    proj *project
    prog *program
    recipes []Value
    language string

    defval   Value
    defers []Value
    values []Value

    prerequisite Value
    _ordered bool

    _env []*pair
    changedWD string

    dirt string // reason of outdated
    start time.Time // start time

    recs map[Value]int

    calleeErrs []error
    calleeErrsM sync.Mutex

    missing []string

    grepping bool
    grepped []Value
    ordered []Value
    targets []Value // all targets def

    countFiles int
    traceLevel int

    interpreted []interpreter
}
func (p *execution) inner() Context { return &p.automatic }
func (p *execution) caller() *execution { return _execution(p.Context) }
func (p *execution) cast(t reflect.Type) Context {
    if t == reflect.TypeOf(p) { return p }
    if t == reflect.TypeOf((*argumented_ctx)(nil)) { return nil }
    if t == reflect.TypeOf((*stemmed_ctx)(nil)) { return nil }
    return p.automatic.cast(t)
}
func (p *execution) do(ctx Context, op any) (res any) {
    switch t := op.(type) {
    case ex_closure: return true
    case get_position:
        if p.prerequisite != nil { return p.prerequisite.Position() }
        if len(p.recipes) > 0 { return p.recipes[0].Position() }

    case get_program : return    p.prog
    case get_project : if nil != p.proj { return p.proj }
    case get_scope   : if nil != p.proj { return p.proj.scope }

    case is_ordered_prereq : return p._ordered && p.prerequisite != nil
    case is_prerequisite, evoke_loop_null: return p.prerequisite != nil

    case default_value:  p.defval = t.v; return
    case exe_res:        p.values = append(p.values, t.v); return
    case mark_dirty:     p.dirty_mark(t.a...)//; return
    case act_traverse:   p.traverse(ctx, t.v); return
    case act_traversed:  return p.traversed(ctx,t.v)
    case act_dirt:       return p.dirty(ctx,t.a...)

    case param_name:
        if p.prog != nil {
            if t.i < len(p.prog.params) {
                res = p.prog.params[t.i].name
            }
        }
        return

    case execution_lang:
        return p.language

    case interpret:
        return p.interp(ctx, t.name, t.args)

    case missing_file:
        p.missing = append(p.missing, t.file)

    case property:
        if t&propDirtyOpts != 0 { return &p.by }
    }
    return p.automatic.do(ctx, op)
}

func (p *execution) ts(t string) (s string) {
    s = "{=" + t
    if v := p.prerequisite; v != nil {
        s += " " + v.String()
    }
    s += " " + ts(p.Context) + "}"
	return
}

func (p *execution) aquire() func() { p.Lock() ; return p.Unlock }

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
func (p *execution) trace(a ...any) { printIndentDots(p.traceLevel, a...) }
func (p *execution) tracef(s string, a ...any) { printIndentDots(p.traceLevel, fmt.Sprintf(s, a...)) }

func (p *execution) traversed(ctx Context, target Value) []Value {
    if !isTrivial(target) {
        if _, y := to_file(target); y {
            p.addFilesCount(1)
        }
        if truly(ctx, is_ordered_prereq{}) {
            p.ordered = append(p.ordered, target)
            auto_set(ctx, defVoid, "|", _list(p.ordered...))
        } else {
            p.targets = append(p.targets, target)
            auto_set(ctx, defVoid, "^", _list(p.targets...))
            auto_set(ctx, defVoid, "<", p.targets[0])
            auto_set(ctx, defVoid, ">", p.targets[len(p.targets)-1])
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
        var k, v = __string(ctx, p.key), __string(ctx, p.val)
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
                    if mat, _, _ = match(p, pat, val); mat {
                        if perUpdatedDep { updatedDeps(p, t, val) }
                        break ForVals
                    }
                }
            }
            if !perUpdatedDep && mat { updatedDeps(p, t, vals...) }
            if !dup { // vals = append(vals, merge(targets)...)
                vals = append(updatedDeps(p, t), vals...)
                vals = append(merge(t), vals...)
            }
        }
    }
    if false && enableDirtyMark { p.dirty_mark(vals...) }
}

func (p *execution) interp(ctx Context, name string, args []Value) (res bool) {
    if len(p.interpreted) == 0 && 0 < len(p.recipes) && name == "configure" {
        if x, y := dialects["eval"]; y && x != nil {
            p.interpret(ctx, x, args)
        }
    } else if x, y := dialects[name]; y && x != nil {
        p.interpret(ctx, x, args)
        return
    }
    return true
}

func (p *execution) interpret(ctx Context, i interpreter, args []Value) (res Value) {
    target, _, _ := wait(ctx, waitopts{
        ExecResults: false,
        ReportUpdates: false,
        StampCurrentTarget: false,
    })

    if false && truly(ctx, is_test_univ{}) {
        if x, y := target.(*file); y && strings.HasSuffix(x.name, ".log") {
            defer func() {
                var cc = pc(pc(ctx,auto_get(ctx,">")),x.fullname())
                debug(cc, "%v %v %v", p.language, args, res)
            } ()
        }
    }
    if _, y := target.(*file); y && !truly(ctx, is_configure{}) && !p.dirty(ctx) {
        // p.traves.add(ctx, traveDone, nil) // NOTE: modifier.predictDirty
        return
    }

    res = i.evaluate(ctx, args...)

    if checkpoints {
        p.evaluate_check(ctx, i, args, res)
    }

    if res != nil {
        if d, prev := auto_set(ctx, defVoid, "-", res); d == nil {
            _, ent, _ := entryIndicator(ctx, _entry(ctx))
            prompt(ctx, "%v: %s\n", ent, intername(i))
            debug(ctx, "set buffer value failed: %v → %v", prev, res, trace{})
        }
    }

    p.interpreted = append(p.interpreted, i)

    if _, _, e := p.updateRecipesHash(ctx, target); e != nil {
        _, ent, _ := entryIndicator(ctx, _entry(ctx))
        prompt(ctx, "%v: %s\n", ent, intername(i))
        debug(ctx, "update recipes hash failed: %v", e, trace{})
    }
    return
}

func isDirty(ctx Context, target Value, a ...Value) (dirty bool) {
    var opts, y = do(ctx, propDirtyOpts).(*dirtyOpts)
    if !y {
        debug(ctx, "nil dirtyopts : %v", ts(ctx), trace{})
        return
    }
    if len(updatedDeps(ctx, target)) > 0 { return true }
    if v := auto_get(ctx, "^"); v != nil { a = append(a, v) }
    for _, dep := range xmerge(ctx, a...) {
        var mat bool = len(opts.pats) == 0
        if !mat {
            for _, pat := range opts.pats {
                if mat, _, _ = match(ctx, pat, dep); mat { break }
            }
        }
        if mat && (updated(ctx, dep) || statFile(ctx, dep).mod().After(statFile(ctx, target).mod())) {
            return true
        }
    }
    return
}

func isDirtyAfter(ctx Context, target Value, t time.Time) (res bool) {
    for _, dep := range updatedDeps(ctx, target) {
        if ds := statFile(ctx, dep); ds != nil {
            res = ds.mod().After(t) || isDirtyAfter(ctx, dep, t)
            if res { break }
        }
    }
    return
}

func (p *execution) dirty(ctx Context, aa ...Value) (outdated bool) {
    var target as

    target.Value, _, _ = wait(p, waitopts{
        ReportUpdates: false,
        ExecResults: false,
        StampCurrentTarget: false,
    })

    var y bool
    var reason string
    var targetFile *file
    var targetFull string
    var opts, args = _opts_[dirtyOpts](ctx, aa...)

    if targetFile, targetFull, y = target.fullname_file(ctx); !y {
        targetFull = __string(ctx, target)
    } else if n := targetFile._traved; n > 1 {
        if false { debug(ctx, 5, "%v, %v, %d", targetFile, targetFull, n) }
        return
    }

    var verb = opts.debug>0 || opts.verbose
    var ts = trimPromptString(targetFull)

    if s := statFile(ctx, target); s == nil || s.exists() != existenceConfirmed {
        outdated, reason = true, "not exists" //fmt.Sprintf("not exists: %s %v", typeof(target), target)
    } else if isDirty(ctx, target, args...) && isDirtyAfter(ctx, target, s.mod()) {
        outdated, reason = true, "prerequisites updated"
    }

    if outdated {
        assert(reason != "", "needs outdated reason")
    } else if y, e := p.isRecipesChanged(ctx, target); e != nil {
        debug(ctx, "recipes changed: %v", e, trace{})
        return
    } else if y {
        outdated, reason = true, "recipes changed"
    } else if !opts.checksum {
        // does nothing
    } else {
        debug(ctx, "FIXME: check prerequisites against the saved checksums", trace{})
        return
    }

    verb = verb ||
        (opts.verboseOutdated && outdated) ||
        (opts.verboseUpdated && !outdated)

    if verb {
        var d = time.Now().Sub(p.start)
        if false && d > 10*time.Second {
            debug(ctx, "%v: %v", target.Value, d)
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
            targetFile._dirty += 1
        }

        var n = p.countFiles + len(p.grepped)
        if db := opts.debug>0; db && !opts.verbose {
            debug(ctx, "%s (%T) (%s) …… %s (%d files in %s, debug=%d)", ts, target, targetFull, m, n, s, opts.debug)
        } else {
            prompt(ctx, "%s …… %s (%d files in %s)\n", ts, m, n, s)
			debug(ctx, "%d %d", db, 6)
        }
    }

    if outdated && p.dirt != "" { reason = p.dirt + "; " + reason }
    if !opts.silent && reason != "" { p.dirt = reason }
    return
}

func probPrereqValue(ctx Context, projects []*project, val Value) (prereqValue, prereqPattern Value, prereqFinal string, prereqFile *file) {
    var mapPrereqFile = func(name any) {
        var ms = unmap_files(unmap_uncheck_ctx{ctx}, _project(ctx), name, nil)
        if ms != nil {
            defer func() {
                if prereqFile != nil { return }
                for _, m := range ms { warn(ctx, "%v, skipped %v", name, m) }
                debug(ctx, "skipped %d, projects %v", len(ms), projects)
            }()
        }

        if prereqFile = select_file(ctx, ms); prereqFile != nil {
            prereqValue = prereqFile
            return
        }

        if prereqValue == nil {
            prereqValue = _strlit(_position(ctx), prereqFinal)
        } else if f, y := to_file(prereqValue); y {
            prereqFile = f
        } else if _, y := prereqValue.(*path); y {
            if f := _stat(ctx, prereqFinal); f != nil { prereqFile, prereqValue = f, f }
        }
    }

    if prereqValue = val; prereqValue == nil {
        if prereqFinal == "" {
            debug(ctx, "prerequisite is nothing", trace{})
        }

        mapPrereqFile(prereqFinal)
        return
    }

    if _, y := prereqValue.(object); y { return }

    if !patterned(ctx,prereqValue) {
        prereqFinal = __string(ctx, prereqValue)

        if prereqFinal == "" { // just reject empty final
            debug(ctx, "%v: %v: empty prerequisite, stems=%v", prereqValue, _stems(ctx), trace{})
        }

        switch prereqValue.(type) {
        case flag, *strlit, *strcomp:
            return // skip checking files for performance
        }

        mapPrereqFile(prereqValue)
        return
    }

    var stems = _stems(ctx)
    if len(stems) == 0 {
        if false { debug(ctx, "%v: no stems, %v", prereqValue, ctx, trace{}) }
        return
    }

    var rest []string
    prereqPattern = prereqValue
    prereqValue, rest = stencil(ctx, prereqPattern, stems)
    if isTrivial(prereqValue) {
        errostack(ctx, 3, "%v: empty stencil with %v", prereqPattern, stems, trace{})
    } else if len(rest) > 0 {
        errostack(ctx, 3, "%v: partial stencil with %v, rest=%v", prereqPattern, stems, rest, trace{})
    }

    if prereqFinal == "" { prereqFinal = __string(ctx, prereqValue); }
    if prereqFinal == "" {
        errostack(ctx, 3, "%v: empty prerequisite, stems=%v", prereqValue, stems, trace{})
    }

    mapPrereqFile(prereqValue)
    return
}

func (p *execution) traved(ctx Context, targetValue, prereqValue, prereqPattern Value, prereqFile *file) {
    var av = targetValue
    var bv = prereqValue
    if prereqFile == nil {
        do(ctx, act_traversed{prereqValue}) // set $< $> $^ or $|
    } else if targetValue != prereqFile {
        do(ctx, act_traversed{prereqFile}) // set $< $> $^ or $|
        bv = prereqFile
    }

    if !isTrivial(av) && !isTrivial(bv) {
        var a = statFile(ctx, av).mod()
        var b = statFile(ctx, bv).mod()
        if (!a.IsZero() && b.After(a)) || updated(ctx, bv) || updatedDeps(ctx, bv) != nil {
            updatedDeps(ctx, av, bv)
        }
    }
}

func with(ctx Context, target Value) (res bool) {
    if t := auto_find(ctx, "@"); t == nil || t.value == nil {
        return
    } else if res = equal(ctx, target, t.value); res {
        return
    } else {
        return with(inner(ctx), target)
    }
}

func (p *execution) traverse(ctx Context, prereqValue Value) {
    var (
        targetValue Value

        prereqPattern Value
        prereqFinal string
        prereqFile *file

        concreteList []entry
        stemmedList []*stemmed_rule
    )

    if targetValue = auto_target_value(ctx); targetValue == nil {
        erro(ctx, "%s: target is nil\n", prereqFinal, trace{})
    } else if isTrivial(targetValue) {
        erro(ctx, "%s: target is trivial (%T)\n", prereqFinal, targetValue, trace{})
    }

    var projs = []*project{ p.proj }

    if len(projs) == 0 {
        note(ctx, "%v", closure_projects(ctx))
        erro(ctx, "%v: %v → %v: no projects", p.proj, targetValue, prereqValue, trace{})
    }

    prereqValue, prereqPattern, prereqFinal, prereqFile = probPrereqValue(ctx, projs, prereqValue)

    if f := prereqFile; f != nil {
        if f._travin += 1; f._travin > 1 { return }
    }

    // Recursion detection -- simply return to break it if looped.
    if traverseDetectLoops {
        if eq(ctx, targetValue, prereqValue) {
            prompt(ctx, "%v: %v: self dependency, consider using [(once)] to avoid\n", targetValue, prereqValue)
            warn(ctx, "recursion: %T %v", prereqValue, prereqValue)
            warn(ctx, "recursion: %T %v", targetValue, targetValue)
            debug(ctx, "recursion: %v : %v ; in %v", targetValue, prereqFile, projs, trace{})
        }
        for c := p; c != nil; c = c.caller() {
            if val := auto_get(c, "@"); val != nil && eq(c, val, prereqValue) {
                var f = as{targetValue}.file(ctx, projs...)
                if true && f == nil {
                    prompt(ctx, "%v: %v: recursion detected, consider using [(once)] to avoid\n", targetValue, prereqValue)
                    warn(ctx, "recursion: %T %v", prereqValue, prereqValue)//.of(prereqValue)
                    warn(ctx, "recursion: %T %v", targetValue, targetValue)//.of(targetValue)
                    debug(ctx, "recursion: %v : %v ; in %v", targetValue, prereqFile, projs)
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

    t1 := time.Now()

    for _, proj := range projs {
        var entries = proj._entries(unmap_uncheck_ctx{ctx}, prereqValue, false)
        if len(entries) == 0 { continue }
        concreteList = append(concreteList, entries...)
        for _, entry := range entries {
            if !isNull(entry) && targetValue == entry { continue }
            if w, k := targetValue.(*word); k && w.s == prereqFinal {
                continue // target resolve to itself, does nothing
            }
            traverse(ctx, entry)
        }
    }

    if d := time.Now().Sub(t1); 60*time.Second < d {
        for _, concrete := range concreteList {
            prompt(ctx, "%v: slow: %v %v\n", concrete.Position(), concrete, targetValue)
        }
        debug(ctx, "%v: slow: %v: %v %v (%d concretes)\n", _position(ctx), targetValue, prereqValue, d, len(concreteList))
    }

    if prereqFile != nil && prereqFile.exists() {
        p.traved(ctx, targetValue, prereqValue, prereqPattern, prereqFile)
        return
    }

    t2 := time.Now()

    for _, proj := range projs {
        for _, p := range proj.patterns { assert(patterned(ctx,p.target), "not pattern") }

        var patterns = proj.resolvePatterns(ctx, prereqValue, prereqFinal)
        if len(patterns) == 0 { continue }

        stemmedList = append(stemmedList, patterns...)

        for _, entry := range patterns { traverse(ctx, entry) }
    }

    if d := time.Now().Sub(t2); 60*time.Second < d {
        for _, stemmed  := range stemmedList { prompt(ctx, "%v: slow: %v\n", stemmed.Position(), stemmed) }
        debug(ctx, "%v: slow: %v: %v %v (%d stemmed)\n", _position(ctx), targetValue, prereqValue, d, len(stemmedList))
    }

    p.traved(ctx, targetValue, prereqValue, prereqPattern, prereqFile)
    return // no operation
}

func (p *execution) prerequisites(va []Value, ordered bool) {
    defer p.Wait()
    p._ordered = ordered
    for _, p.prerequisite = range va { traverse(p, p.prerequisite) }
    p.prerequisite = nil
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
}

func (prog *program) getModifiers(ctx Context, name string) (ms []*modifier) {
    for _, d := range prog.depends {
        if g, y := d.(*modification); y {
            for _, m := range g.list {
                if __string(ctx, m.elems[0]) == name { ms = append(ms, m) }
            }
        }
    }
    return
}

const maxCallRecursion  = 32 //64

func (prog *program) workdir(ctx Context) (workdir string) {
    var proj = prog.project
    if p := _execution(ctx); p == nil {
        workdir = proj.absPath
    } else if p.changedWD == "" {
        var o object
        if o = proj.resolve(ctx, "CWD"); isTrivial(o) {
            if o = proj.resolve(ctx, "/"); isTrivial(o) {
                erro(ctx, "both $(CWD) and $/ are trivial", trace{})
            }
        }
        if v := expand(_final(ctx),o); v == nil {
            erro(ctx, "trivial %v", ts(o), trace{})
        } else {
            workdir = __string(ctx, v)
        }
    } else if filepath.IsAbs(p.changedWD) {
        workdir = p.changedWD
    } else {
        workdir = filepath.Join(proj.absPath, p.changedWD)
    }
    return
}

func (prog *program) target(ctx *execution) (target Value) {
    if target = _entry(ctx).destiny(); target == nil {
        erro(ctx, "%v: nil entry target", target, trace{})
    }

    switch a := target.(type) {
    case *strlit, *strcomp: // NOTE: skip strings to optimize speed from searching
    case *file: if a._traved > 1 { return } // alreadyUpdated = a.info != nil && a.updated
    case fullfile: if a._traved > 1 { return } // alreadyUpdated = a.info != nil && a.updated
    default:
        if _, y := a.(flag); y {
            if s := prog.project.name; s == "configure" || s == "configure.base" {
                break
            }
        }
        if f := prog.project.file(ctx, a); f != nil {
            if f._traved > 1 { return } else { target = f }
        }
    }

    if ctx.recs == nil { ctx.recs = make(map[Value]int) }
    ctx.recs[target] += 1
    return
}

func (prog *program) check_exe(ctx *execution) {
    var cc = ctx.Context
    var a = []Value{ auto_get(cc, "@") }
    var depth, loop int = 0, -1

callerloop:
    for c := ctx.caller(); c != nil; c = c.caller() {
        if _program(c) == prog {
            if depth += 1; depth == maxCallRecursion { break callerloop }
            var t = auto_get(c, "@")
            for i, v := range a {
                if eq(ctx, t, v) { loop = i; break callerloop }
            }
            if loop < 0 { a = append(a, t) }
        }
    }

    if 0 <= loop {
        var o = cast[*term](cc)
        var v = auto_get(o, "@")
        var t = auto_get(cc, "@")
        if o != nil {
            if v != nil && eq(cc, v, t) {
                if true { debug(ctx, "skip closure loop: %v %v", o, t) }
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

        prompt(ctx, "%v: %v: %v, %v\n", a[0], v, cc, o)
        for i, t := range a { erro(ctx, "loop: %v: %v", i, t) }
        debug(ctx, "loop, (depth=%d, %v, %v)\n", depth, a[loop], a, trace{})
    }

    if depth < maxCallRecursion {
        // continues
    } else if c := ctx.caller(); c != nil {
        ctx.traceLevel = c.traceLevel

        var tt = as{auto_get(c, "@")}
        var s, _ = tt.fullname_string(ctx)
        prompt(ctx, "%v: max recursion call (%d)\n", s, depth)
        debug(ctx, "max recursion call (%d)\n", depth)

        const collapse = false

        for ; c != nil; c = c.caller() {
            var n int

            if collapse {
                for next := c.caller(); next != nil; next = next.caller() {
                    if d := auto_get(next, "@"); d == nil { continue } else
                    if t := d; t != nil && eq(ctx, t, tt) { n += 1;  continue }
                    if _program(next) == _program(c) { n += 1; c = next } else { break }
                }
            }

            if prog, t := _program(c), auto_get(c, "@"); prog == nil {
                erro(ctx, "%v (@=%v)", _entry(ctx), tt)
                break
            } else if n > 0 {
                erro(ctx, "%v (repeated %d times)", t, n)
            } else if !collapse {
                erro(ctx, "%v : %v", t, auto_get(c, ">"))
            } else if depth -= 1; maxCallRecursion - depth > 5 {
                erro(ctx, "%v ... (%d)", t, maxCallRecursion - depth)
                break
            } else {
                erro(ctx, "%v : %v", t, auto_get(c, ">"))
            }

            flush(ctx) // dump immediately
        }

        debug(ctx, depth, "#>", _entry(ctx), trace{})
    }
}

func (prog *program) result_or_default_interpret(ctx *execution) (res Value) {
    if res = auto_get(ctx, "-"); res != nil {
        return
    }
    if len(ctx.interpreted) == 0 {
        if x, y := dialects[""]; y && x != nil {
            return ctx.interpret(ctx, x, nil)
        }
        debug(ctx, "no default dialect", trace{})
    }
    return
}

func (prog *program) execute(_ctx Context) (res Value) {
    var exe = &execution{
        automatic:automatic{Context:_ctx, defs:make(def_map)},
        recs:make(map[Value]int), start:time.Now(), prog:prog,
        proj:prog.project, recipes:prog.recipes, language:prog.language,
    }

    if checkpoints {
        defer prog.execute_check(exe, &res)
    }

    prog.check_exe(exe)

    defer func() {
        for _, a := range exe.defers {
            if x, y := a.(*group); y {
                modify(exe, x, true)
            } else {
                debug(exe, "defer: not a modifier: %s", ts(a), trace{})
            }
        }

        if res == nil { res = auto_get(exe, "-") }
        if res == nil { res = exe.defval }
        if res != nil { do(exe.Context, default_value{res}) }

        exe.defval = nil
    } ()

    for _, param := range prog.params {
        if  exe.params == nil  {
            exe.params = make(map[string]*auto, len(prog.params))
        }
        exe.params[param.name] = param
    }

    if checkpoints {
        prog.execute_check_0(exe)
    }

    // NOTE: set "@" before setting auto args
    // Select the right target value before setting parameters,
    // because the target could be overrided by parameters.
    exe.set(exe, defVoid, "@", prog.target(exe))

    if t := _stems(exe); t != nil {
        exe.set(exe, defVoid, "*", ease(exe, t))
    }

    exe.do(exe, init_args{&exe.automatic})
    exe.prerequisites(prog.depends, false)
    exe.prerequisites(prog.ordered, true)

    if checkpoints {
        prog.execute_check_1(exe)
    }

    if len(prog.recipes) == 0  { return }
    return prog.result_or_default_interpret(exe)
}
