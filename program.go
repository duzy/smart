//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    //"strconv"
    "strings"
    "sync"
    "time"
    "fmt"
)

type dependPatternUnfit struct {}
func (*dependPatternUnfit) Error() string { return "pattern unfit" }

type spawnProgramContext struct { programContext }
func (sc *spawnProgramContext) String() string { return fmt.Sprintf("spawn-program{%s}", sc.Context) }

type programContext struct {
    autoContext
    prog *Program
    params []string // $0, $1, $2, ...
    mutex sync.Mutex
    _dirtyOpts modifierSetDirtyPatsOpts
}

func (pc *programContext) inner() Context { return &pc.autoContext }
func (pc *programContext) caller() *programContext { return pc.Context.programCtx() }
//XXX: func (pc *programContext) stems() []string { return nil }
func (pc *programContext) String() string {
    if fullContextStringer {
        var s = strings.TrimPrefix(pc.prog.scope.comment, "rule ")
        return fmt.Sprintf("program{%s,%s}", s, pc.autoContext.String())
    } else {
        return pc.autoContext.String()
    }
}
func (pc *programContext) programCtx() *programContext { return pc }
func (pc *programContext) program() *Program { return pc.prog }
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
func (pc *programContext) Position() Position {
    if pc.prog != nil { return pc.prog.position }
    return pc.autoContext.Position()
}
func (pc *programContext) spawn() Context {
    pc.mutex.Lock(); defer pc.mutex.Unlock()

    var ctx = pc.Context
    switch t := ctx.(type) {
    case *closureContext, *traverseContext: ctx = t.spawn()
    default: erro(pc, "program needs to spawn %v", ctx).debug(1)
    }
    return &spawnProgramContext{programContext{autoContext{
        Context: ctx, defs: pc.defs.clone() }, pc.prog, pc.params, sync.Mutex{},
        modifierSetDirtyPatsOpts{},
    }}
}
func (pc *programContext) appendCallerUpdated() bool { return true }
func (pc *programContext) mustExists() bool { return false }
func (pc *programContext) closureScopes() (scopes []*Scope) {
    if cc, ok := pc.Context.(*closureContext); ok {
        if true {
            scopes = cc.closureScopes()
        } else if up := cc.programCtx(); up != nil {
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

func (pc *programContext) dirtyOpts() *modifierSetDirtyPatsOpts { return &pc._dirtyOpts }
func (pc *programContext) dirtyMark(vals ...Value) {
    const enableDirtyMark = true
    if !enableDirtyMark {
        // does nothing
    } else if t, _ := pc.autoGet("@"); isTrivial(t) {
        // should not happen, but safely ignoring..
    } else if tt := merge(t); len(tt) == 0 {
        // should not happen, but safely ignoring..
    } else if len(vals) == 0 {
        vals = append(vals, tt...)
    } else if /*last := vals[len(vals)-1]; last != tt*/true {
        const perUpdatedDep = true
        var (
            mat, dup bool
            opts = pc.dirtyOpts()
        )
        vals = merge(vals...)
        for _, t := range tt {
            ForVals: for _, val := range vals {
                if val == t /*|| val.cmp(pc,t) == cmpEqual*/ {
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
            if !dup { // vals = append(vals, merge(tt)...)
                vals = append(t.updatedDeps(pc), vals...)
                vals = append(merge(t), vals...)
            }
            if false { warn(pc, "dirtyMark: %T %v; %v, %v, %v, %v", t, t, vals, dup,
                t.updated(pc), t.updatedDeps(pc)).debug(0) }
            if false { warn(pc, "dirtyMark: %T %v; %v, %v, %v, %v", t, t, vals, dup,
                t.updated(pc), t.updatedDeps(pc)).debug(18) }
         }
    }
    if enableDirtyMark { pc.Context.dirtyMark(vals...) }
}

type Program struct {
    position Position
    project *Project
    scope   *Scope
    params  []*Def
    depends []Value // normal
    ordered []Value // order-only
    recipes []Value
    defaultVal Value
    language string
    changedWD string
    configure bool
}

func (prog *Program) Position() Position { return prog.position }
func (prog *Program) Project() *Project { return prog.project }
func (prog *Program) Scope() *Scope { return prog.scope }
func (prog *Program) interpret(ctx Context, i interpreter, params []Value) (err error) {
    if pos := ctx.Position(); !pos.IsValid() && prog.position.IsValid() {
        ctx = positional(ctx, prog.position)
    }

    wait(ctx) // wait for prerequisites before interpretion

    var value Value
    if value, err = i.Evaluate(ctx, params...); err != nil {
        var (
            _, ent, _ = entryStr(ctx, ctx.entry())
            nam = intername(i)
        )
        prompt(ctx, "%v: interpret '%s' recipes failed\n", ent, nam)
        erro(ctx, "%s: %v", nam, err)
        errostack(ctx, 3, "%v", ctx).debug(1)
        return
    } else if isNil(value) {
        // disgard nil value
    } else if prev, ok := ctx.autoSet("-", value); !ok {
        var (
            _, ent, _ = entryStr(ctx, ctx.entry())
            nam = intername(i)
        )
        prompt(ctx, "%v: %s\n", ent, nam)
        erro(ctx, "set buffer value failed: %v -> %v", prev, value)
        errostack(ctx, 3, "%v", ctx).debug(1)
        return
    }

    if _, _, err = updateRecipesHash(ctx); err != nil {
        var (
            _, ent, _ = entryStr(ctx, ctx.entry())
            nam = intername(i)
        )
        prompt(ctx, "%v: %s\n", ent, nam)
        erro(ctx, "update recipes hash failed: %v", err)
        errostack(ctx, 3, "%v", ctx).debug(1)
    } else if t := ctx.traversal(); t != nil {
        t.interpreted = append(t.interpreted, i)
    }
    return
}

func (prog *Program) getModifiers(ctx Context, name string) (ms []*modifier) {
    for _, d := range prog.depends {
        var g, ok = d.(*modifiergroup)
        if !ok { continue }
        for _, m := range g.modifiers {
            if s, e := m.name.Strval(ctx); e != nil {
                erro(ctx, "get modifier name '%v' failed: %v", m.name, e).of(m.name).debug(1)
                return
            } else if s == name {
                ms = append(ms, m)
            }
        }
    }
    return
}

func (prog *Program) modify(ctx Context, m *modifier) (brks breakers) {
    // TODO: using rules in a different project to implement modifiers, e.g.
    //       [ foo.check-preprequisites ]
    //       [ foo.baaaar ]
    var (
        name string
        args []Value
        err error
    )
    if args, err = expandmerge2(ctx, expandPlainValue, m.name); err != nil {
        erro(ctx, "expand modifier name '%v' failed: %v", m.name, err).of(m.name).debug(1)
        return
    } else if n := len(args); n == 0 {
        erro(ctx, "modifier name '%v' is empty", m.name).of(m.name).debug(1)
        return
    } else if name, err = args[0].Strval(ctx); err != nil {
        erro(ctx, "strval '%v' failed: %v", args[0], err).of(args[0]).debug(1)
        return
    } else {
        args = append(args[1:], m.args...)
    }

    var proj = ctx.Project()
    if f, ok := modifiers[name]; ok {
        var t = ctx.traversal()
        var value Value //, _ = ctx.autoGet("-")
        // Special modifier processing (implicit interpretation) before (configure)
        if len(t.interpreted) == 0 && len(prog.recipes) > 0 && name == "configure" /*&& (isNil(value) || isNone(value))*/ {
            // Evaluate for configure modifier
            if i, ok := dialects["eval"]; ok && i != nil {
                if err = prog.interpret(ctx, i, args); err != nil {
                    var _, ent, _ = entryStr(ctx, ctx.entry())
                    prompt(ctx, "%v: %s failed for %s\n", ent, name, proj)
                    erro(ctx, "interpret failed: %v", err)
                    errostack(ctx, 3, "%v", ctx).debug(1)
                    return
                }
            }
        }
        if value, brks = f(positional(ctx, m.position), args...); brks.has() {
            if tb := brks.not(breakCase, breakNext, breakDone); false && tb.has() {
                var _, ent, _ = entryStr(ctx, ctx.entry())
                prompt(ctx, "%v: %s failed for %s\n", ent, name, proj)
                for _, brk := range tb {
                    switch brk.what {
                    case breakFail: erro(ctx, "%v: %s: %v", proj, name, brk.message).at(brk.pos)
                    case breakErro: erro(ctx, "%v: %s: %v", proj, name, brk.error).at(brk.pos)
                    default: erro(ctx, "%v: %s: %v", proj, name, brk.what).at(brk.pos)
                    }
                }
                errostack(ctx, 3, "(%T):", ctx).debug(6)
                fail(m.Position(), "%s failed for project %s", name, proj)
            } else {
                return;
            }
        } else if hyphen, found := ctx.autoGet("-"); !found || isNil(value) || value == hyphen {
            // does nothing
        } else if ctx.autoSet("-", value); false {
            var _, ent, _ = entryStr(ctx, ctx.entry())
            prompt(ctx, "%v: %s failed for %s\n", ent, name, proj)
            erro(ctx, "setting buffer value failed: %v", value)
            errostack(ctx, 3, "(%T):", ctx).debug(1)
            fail(m.Position(), "%s failed for project %s", name, proj)
        }
    } else if i, _ := dialects[name]; i != nil {
        if err = prog.interpret(ctx, i, args); err != nil {
            var _, ent, _ = entryStr(ctx, ctx.entry())
            prompt(ctx, "%v: %s failed for %s\n", ent, name, proj)
            erro(ctx, "%s: %v", name, err)
            errostack(ctx, 3, "(%T):", ctx).debug(1)
            fail(m.Position(), "%s failed for project %s", name, proj)
        }
    } else {
        var _, ent, _ = entryStr(ctx, ctx.entry())
        prompt(ctx, "%v: %s failed for %s\n", ent, name, proj)
        erro(ctx, "unknown modifier '%s'", name)
        errostack(ctx, 3, "%v", ctx).debug(1)
    }
    return
}

const maxRecursion  = 32 //64

type normalTraverseContext struct { Context }
type orderTraverseContext struct { Context }
func (t *normalTraverseContext) traversed(target Value) (targets []Value) {
    if targets = t.Context.traversed(target); len(targets) > 0 {
        t.autoSet("^", MakeList(t.Position(), targets...))
        t.autoSet("<", targets[0])
        t.autoSet(">", targets[len(targets)-1])
    }
    return
}
func (t *orderTraverseContext) traversed(target Value) (targets []Value) {
    if targets = t.Context.traversed(target); len(targets) > 0 {
        t.autoSet("|", MakeList(t.Position(), targets...))
    }
    return
}

func (prog *Program) execute(cc Context) (result Value, brks breakers) {
    var (
        args  = cc.arguments()
        entry = cc.entry()
        pos   = cc.Position()
    )
    if !pos.IsValid() { pos = entry.Position() }
    if cc != nil && cc.checkErrors(true) > 0 {
        var errs = cc.totalErrors()
        var s string; if errs > 1 { s = "s" }
        prompt(cc, "%v: execution canceled with %d errors for '%s'\n", entry, errs, prog.project)
        warn(cc, `cancel execution for "%v" due to %d error%s`, entry, errs, s)
        warnstack(cc, 5, `%v`, cc).debug(16)
        if options.failOnErrors { fail(pos, "fail by %d error%s", errs, s) }
        return
    }

    assert(prog.project == prog.scope.project, "mismatched scope/project")

    var (
        t = traverseContext{
            Context: cc,
            execRec: make(map[Value]int),
            start: time.Now(),
            print: true,
        }
        pc = programContext{autoContext:autoContext{Context:&t, defs:make(autoDefMap)}, prog:prog}
        ctx Context = &pc
        target Value
        err error
    )

    defer func() {
        var (
            targets, _ = ctx.autoGet("@")
            depends, _ = ctx.autoGet("^")
            tb = brks.not(breakDone, breakNext)
        )
        if isTrivial(targets) { targets = entry.Target() }
        if true || tb.has() {
            // breaked
        } else if !isTrivial(targets) && !isTrivial(depends) {
            for _, target := range merge(targets) {
                for _, dep := range merge(depends) {
                    var u = dep.updated(ctx)
                    if s, _ := target.Strval(ctx); strings.HasPrefix(s, "ui/ui.h") {
                        warn(ctx, "%T %v: %T %v; updated = %v", target, target, dep, dep, u).debug(1)
                    }
                    if u { /* TODO: ... */ }
                }
            } 
        }
        if ctx.checkErrors(true) > 0  {
            var (
                str, ent, tar = entryStr(ctx, entry)
                errs = ctx.totalErrors()
            )
            if !ctx.configuration() && cc != nil {
                if errs == 1 {
                    err = fmt.Errorf("execution yields an error for %v", str)
                } else {
                    err = fmt.Errorf("execution yields %d errors for %v", errs, str)
                }
                brks.add(ctx, breakErro).error = err
            }
            if tar != "" {
                prompt(ctx, "%v: %s, execution failed with %d errors, project %s\n", ent, tar, errs, prog.project)
            } else {
                prompt(ctx, "%v: execution failed with %d errors, project %s\n", ent, errs, prog.project)
            }
            warn(ctx, `%d errors in execution "%s"`, errs, str)
            warnstack(ctx, 8, "(%T): %v", ctx, prog.project).debug(10)
            if options.failOnErrors { fail(prog.position, "fail by %d errors", errs) }
        }
    } ()

    if cc != nil {
        var recursion int
        for c := cc.traversal(); c != nil; c = c.caller() { if c.program() == prog { recursion += 1 }}
        if recursion >= maxRecursion {
            errostack(ctx, recursion, "max recursion: %v", entry.Target()).debug(16)
            for c := cc.traversal(); c != nil; c = c.caller() {
                var n int
                for next := c.caller(); next != nil; next = next.caller() {
                    if next.program() == c.program() { n += 1; c = next } else { break }
                }
                if ct, _ := c.autoGet("@"); n > 0 {
                    erro(ctx, "%v (repeats %d times)", ct, n).at(c.program().position)
                } else {
                    erro(ctx, "%v", ct).at(c.program().position)
                }
            }
            erro(ctx, "too many recursion (%d) (%v)", recursion, entry.Target()).debug(1)
            fail(prog.position, "max recursion")
            return
        }
        if /*options.traceTraversalNestIndent*/true { t.traceLevel = cc.traversal().traceLevel }
        if stems := cc.stems(); stems != nil { ctx.autoSet("*", MakeString(pos, stems[0])) }
    }

    if pc.params, err = pc.autoArgs(prog.params, args); err != nil {
        erro(ctx, "auto args failed: %v", err).debug(1)
        return
    }

    // Select the right target value before setting parameters,
    // because the target could be overrided by parameters.
    switch target = entry.Target(); a := target.(type) {
    case *Flag: t.print = false // Flag target (-foo) turns off printing automatically
    case *File: //alreadyUpdated = a.info != nil && a.updated
    default:
        var s string
        if isNil(a) {
            erro(ctx, "%v: nil entry target", target)
            errostack(ctx, 8, "%v", ctx).debug(20)
            return
        } else if s, err = a.Strval(ctx); err != nil {
            erro(ctx, "strval '%v' failed: %v", a, err).debug(1)
            return
        } else if file := prog.project.FindFile(ctx, s); file != nil {
            //alreadyUpdated = file.info != nil && file.updated
            target = file
        }
    }
    ctx.autoSet("@", target)

    // Note: must enter work directory (cd) before setting cloctx
    var (
        alreadyUpdated bool
        enterBack *enterec
    )
    if len(cd.stack) > 0 { enterBack = cd.stack[0] }
    if err = enter(ctx, prog.project.absPath); err != nil {
        erro(ctx, "enter project '%v' failed: %v", prog.project, err).debug(1)
        return
    }
    defer func(swd string) {
        if e := leave(ctx, prog, enterBack); e != nil {
            // NOTE: err could be breakCase, breakDone, etc.
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

        if !isNil(result) && !isNone(result) {
            // good!
        } else if result, _ = ctx.autoGet("-"); !isNil(result) {
            // good!
        } else if !isNil(defaultVal) {
            result = defaultVal
        }

        if cc != nil { if p := cc.program(); p != nil && !isNil(result) { p.defaultVal = result }}
    } (prog.project.changedWD)

    if alreadyUpdated {
        if options.verbose { info(ctx, "'%v' already updated", target) }
        if false { warn(ctx, "'%v' already updated", target).debug(1) }
        if false { return }
    }

    if t.print && entry.Class() == UseRuleEntry { t.print = false }
    if t.print && prog.configure { t.print = false }
    cd.stack[0].silent = !t.print

    if options.traceExec {
        var d = t.depth()
        var s = fmt.Sprintf("%s: %v (%p, exec.depth=%d)", typeof(target), target, target, d)
        defer un(trace(t_exec, s))
    }

    if t.execRec[target] += 1; false { if t.execRec[target] > 1 {
        if options.traceExec { t_exec.trace(fmt.Sprintf("exec: %v", target)) }
        return
    }}

    var proj = ctx.Project()

    // Update normal prerequisites
    ctx.autoSet("^", nil)
    ctx.autoSet("<", nil)
    ctx.autoSet(">", nil)
    if brks = prog.traverse(&normalTraverseContext{ ctx }, prog.depends); brks.has() {
        if t := brks.not(breakCase, breakNext, breakDone); false && t.has() {
            var target, _ = ctx.autoGet("@")
            prompt(ctx, "%v: traverse program failed (target=%s; project=%s)\n",
                entry, target, proj)
            for _, brk := range t {
                switch brk.what {
                case breakErro: warn(ctx, `%v: %s: %v`, proj, target, brk.error).at(brk.pos)
                case breakFail: warn(ctx, `%v: %s: %v`, proj, target, brk.message).at(brk.pos)
                default:        warn(ctx, `%v: %s: %v (%v)`, proj, target, brk.message, brk.what).at(brk.pos)
                }
            }
            warnstack(ctx, 5, "%v: %v", proj, ctx).debug(16)
        }
        return
    } else if errs := ctx.checkErrors(true); errs > 0 {
        brks.add(positional(ctx, prog.position), breakFail).message = fmt.Sprintf("traverse prerequisites failed (%d errors)", ctx.totalErrors())
        prompt(ctx, "%v: execute failed, project %s\n", /*entryStr1(ctx, entry)*/entry, proj)
        warn(ctx, "%d errors while traversing prerequisites for %v", errs, target)
        if warnstack(ctx, 6, "call stack for %v:", target).debug(8); options.failOnErrors {
            fail(prog.position, "fail by %d errors", ctx.totalErrors())
        }
        return
    }

    // Update order-only prerequisites
    ctx.autoSet("|", nil)
    if brks = prog.traverse(&orderTraverseContext{ ctx }, prog.ordered); brks.has() {
        if t := brks.not(breakCase, breakNext, breakDone); false && t.has() {
            var target, _ = ctx.autoGet("@")
            prompt(ctx, "%v: traverse program failed (target=%s; project=%s)\n",
                entry, target, proj)
            for _, brk := range t {
                switch brk.what {
                case breakErro: warn(ctx, `%v: %s: %v`, proj, target, brk.error).at(brk.pos)
                case breakFail: warn(ctx, `%v: %s: %v`, proj, target, brk.message).at(brk.pos)
                default:        warn(ctx, `%v: %s: %v (%v)`, proj, target, brk.message, brk.what).at(brk.pos)
                }
            }
            warnstack(ctx, 5, "%v: %v", proj, ctx).debug(16)
        }
        return
    } else if errs := ctx.checkErrors(true); errs > 0 {
        brks.add(positional(ctx, prog.position), breakFail).message = fmt.Sprintf("traverse prerequisites failed (%d errors)", errs)
        prompt(ctx, "%v: execute failed, project %s\n", /*entryStr1(ctx, entry)*/entry, proj)
        warn(ctx, "%d errors while traversing prerequisites for %v", errs, target)
        if warnstack(ctx, -1, "call stack for %v:", target).debug(8); options.failOnErrors {
            fail(prog.position, "fail by %d errors", ctx.totalErrors())
        }
        return
    }

    if h, ok := ctx.autoGet("-"); len(t.interpreted) == 0 && len(prog.recipes) > 0 && (!ok || isNil(h) || isNone(h)) {
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

func (prog *Program) traverse(ctx Context, prerequisites []Value) (brks breakers) {
    // IMPORTANT: don't expand the args here. The prerequisites like
    // '$(or &@,...)' have to be expanded when it's used (e.g. compare).
    if true {
        for _, prerequisite := range prerequisites {
            if brks = prerequisite.traverse(ctx); !brks.has() {
                continue
            } else if t := brks.not(breakNext, breakCase, breakDone); false && t.has() {
                var proj = ctx.Project()
                for _, brk := range t {
                    switch brk.what {
                    case breakErro: warn(ctx, "%v: %v", proj, brk.error  ).at(brk.pos)
                    case breakFail: warn(ctx, "%v: %v", proj, brk.message).at(brk.pos)
                    default:        warn(ctx, "%v: %v", proj, brk.what   ).at(brk.pos)
                    }
                }
                warnstack(ctx, 5, "%v: %v: %v", proj, ctx.entry(), prerequisite).debug(36)
            }
            if brks.has(breakCase) {
                continue
            } else {
                break
            }
        }
    } else if num := len(prerequisites); num > 0 {
        var (
            mu sync.Mutex
            wg sync.WaitGroup
        )
        for _, prerequisite := range prerequisites {
            wg.Add(1); go func (ctx Context) {
                defer checkPanicsErrors(ctx)
                defer wg.Done() // minus 1
                if t := prerequisite.traverse(ctx); t.has() {
                    mu.Lock(); defer mu.Unlock()
                    brks = append(brks, t...)
                }
            } (ctx.spawn())
        }
        if wg.Wait(); brks.has() {
            var t = brks.not(breakNext, breakCase, breakDone)
            if len(t) > 0 && len(ctx.stems()) > 0 && false {
                var target, _ = ctx.autoGet("@")
                warn(ctx, "broken traversal: %v (target = %v, stems = %v)", t[0].what, target, ctx.stems()).debug(1)
            }
        }
    }
    return
}
