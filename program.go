//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "path/filepath"
    "io/fs"
    // "strconv"
    "strings"
    "sync"
    "time"
    "fmt"
    "os"
)

type dependPatternUnfit struct {}
func (*dependPatternUnfit) Error() string { return "pattern unfit" }

type programContext struct {
    autoContext
    mutex sync.Mutex
    by    modifierSetDirtyPatsOpts
    projs []*Project
    prog *Program
    params []string // $0, $1, $2, ...
}

func (pc *programContext) inner() Context { return &pc.autoContext }
func (pc *programContext) caller() *programContext { return pc.Context.programContext() }
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
    return &programContext{
        autoContext{ Context: ctx, defs: pc.defs.clone() },
        sync.Mutex{}, modifierSetDirtyPatsOpts{}, //travestates{},
        pc.projs, pc.prog, pc.params,
    }
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
    dirt string // reason of outdated
    configure bool
}

func (prog *Program) Position() Position { return prog.position }
func (prog *Program) Project() *Project { return prog.project }
func (prog *Program) Scope() *Scope { return prog.scope }
func (prog *Program) interpret(ctx Context, i interpreter, params []Value) (err error) {
    if pos := ctx.Position(); !pos.IsValid() && prog.position.IsValid() {
        ctx = positional(ctx, prog.position)
    }

    var (
        target Value
        value Value
    )

    // wait for prerequisites before interpretion
    if target, _, _, err = wait(ctx); err != nil {
        erro(ctx, "waiting traversal failed: %v", err).debug(1)
        return
    }

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

    if _, _, err = updateRecipesHash(ctx, target); err != nil {
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
            if m.name.Strval(ctx) == name {
                ms = append(ms, m)
            }
        }
    }
    return
}

func (prog *Program) modify(ctx Context, m *modifier) (traves travestates) {
    // TODO: using rules in a different project to implement modifiers, e.g.
    //       [ foo.check-preprequisites ]
    //       [ foo.baaaar ]
    var (
        name string
        args = mergeExpand(ctx, expandPlainValue, m.name)
    )
    if n := len(args); n == 0 {
        erro(ctx, "modifier name '%v' is empty", m.name).of(m.name).debug(1)
        return
    } else {
        name = args[0].Strval(ctx)
        args = append(args[1:], m.args...)
    }

    var proj = ctx.Project()
    if f, ok := modifiers[name]; ok {
        if false { warn(ctx, "%T %v ; %s", m.name, m.name, name).debug(1) }
        var value Value //, _ = ctx.autoGet("-")
        // Special modifier processing (implicit interpretation) before (configure)
        if len(ctx.traversal().interpreted) == 0 && len(prog.recipes) > 0 && name == "configure" {
            // Evaluate for configure modifier
            if i, ok := dialects["eval"]; ok && i != nil {
                if err := prog.interpret(ctx, i, args); err != nil {
                    var _, ent, _ = entryStr(ctx, ctx.entry())
                    prompt(ctx, "%v: %s failed for %s\n", ent, name, proj)
                    erro(ctx, "interpret failed: %v", err)
                    errostack(ctx, 3, "%v", ctx).debug(1)
                    return
                }
            }
        }
        if value, traves = f(positional(ctx, m.position), args...); traves.has() {
            if t := traves.not(traveCase, traveNext, traveDone); false && t.has() {
                if options.verbose || options.verboseBreaks {
                    var _, ent, _ = entryStr(ctx, ctx.entry())
                    prompt(ctx, "%v: %s failed for %s\n", ent, name, proj)
                    for _, s := range t { warn(ctx, "%v: %s: %v", proj, name, s).at(s.pos) }
                    warnstack(ctx, 5, "").debug(16)
                }
            }
            return
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
        if false { warn(ctx, "%T %v ; %s", m.name, m.name, name).debug(1) }
        if err := prog.interpret(ctx, i, args); err != nil {
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

const maxCallRecursion  = 32 //64

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

func (prog *Program) workDir(ctx Context) (workDir string) {
    if prog.changedWD == "" {
        var o Object
        if _, o = prog.scope.Find("CWD"); isTrivial(o) {
            if _, o = prog.scope.Find("/"); isTrivial(o) {
                erro(ctx, "both $(CWD) and $/ are trivial").debug(1)
                return
            }
        }
        if def, ok := o.(*Def); ok {
            if v := def.Call(ctx); !isTrivial(v) {
                workDir = v.Strval(ctx)
            } else {
                erro(ctx, "%v is trivial", def.name).debug(1)
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
      var envars []*Pair // disclosed values
    if def, _ := prog.scope.Lookup(TheShellEnvarsDef).(*Def); def != nil {
        if l, _ := def.value.(*List); l != nil {
            for _, v := range l.Elems {
                var t Value
                if t = v.expand(ctx, expandClosure); isNil(t) { t = v }
                if p, ok := t.(*Pair); ok {
                    envars = append(envars, p)
                } else {
                    erro(ctx, "env expecting pairs: %T", t).of(t).debug(1)
                    return
                }
            }
        }
    }

    env = os.Environ()
    osi = len(env)
    for _, p := range envars {
        var (
            k = p.Key.Strval(ctx)
            v = p.Value.Strval(ctx)
        )
        // if i > 0 { envstr += " && " }
        // envstr += fmt.Sprintf(`%s=%s`, k, strconv.Quote(v))
        env = append(env, fmt.Sprintf("%s=%s", k, v))
    }
    return
}

func (prog *Program) execute(cc Context) (result Value, traves travestates) {
    var (
        args  = cc.arguments()
        entry = cc.entry()
        pos   = cc.Position()
    )
    if !pos.IsValid() { pos = entry.Position() }
    if cc != nil && cc.checkErrors(true) > 0 {
        var errs = cc.totalErrors()
        var s string; if errs > 1 { s = "s" }
        prompt(cc, "%v: canceled execution (%d error%s), project %s\n",
            entry, errs,s, prog.project)
        warn(cc, `cancel "%v"`, entry)
        warnstack(cc, 5, `%v`, cc).debug(16)
        if options.failOnErrors { fail(pos, "fail by %d error%s", errs, s) }
        return
    }

    assert(prog.project == prog.scope.project, "mismatched scope/project")
    prog.dirt = "" // reset "dirt" -- the 'dirty' reasons

    var t = traverseContext{
        Context: cc,
        execRec: make(map[Value]int),
        start: time.Now(),
        print: true,
    }
    var pc = programContext{
        autoContext: autoContext{ Context:&t, defs:make(autoDefMap) },
        prog: prog,
    }

    var (
        ctx Context = &pc
        target Value
        err error
    )
    defer func() {
        var (
            targets, _ = ctx.autoGet("@")
            depends, _ = ctx.autoGet("^")
            tb = traves.not(traveDone, traveNext)
        )
        if isTrivial(targets) { targets = entry.Target() }
        if true || tb.has() {
            // breaked
        } else if !isTrivial(targets) && !isTrivial(depends) {
            for _, target := range merge(targets) {
                for _, dep := range merge(depends) {
                    var u = dep.updated(ctx)
                    if s := target.Strval(ctx); strings.HasPrefix(s, "ui/ui.h") {
                        warn(ctx, "%T %v: %T %v; updated = %v", target, target, dep, dep, u).debug(1)
                    }
                    if u { /* TODO: ... */ }
                }
            } 
        }
        if ctx.checkErrors(true) > 0 {
            var (
                str, ent, tar = entryStr(ctx, entry)
                errs = ctx.totalErrors()
            )
            if !ctx.configuration() && cc != nil {
                s := traves.add(ctx, traveFail, target)
                if errs == 1 {
                    s.error = fmt.Errorf("execution yields an error for %v", str)
                } else {
                    s.error = fmt.Errorf("execution yields %d errors for %v", errs, str)
                }
            }
            if tar != "" {
                prompt(ctx, "%v: %s, execution failed with %d errors, project %s\n", ent, tar, errs, prog.project)
            } else {
                prompt(ctx, "%v: execution failed with %d errors, project %s\n", ent, errs, prog.project)
            }
            warn(ctx, `%d errors in execution "%s"`, errs, str)
            warnstack(ctx, 8, "(%T): %v", ctx, prog.project).debug(32)
            if options.failOnErrors { fail(prog.position, "fail by %d errors", errs) }
        }
    } ()

    if cc != nil {
        var depth int
        if false {
            for c := cc.traversal(); c != nil; c = c.caller() {
                if c.program() == prog {
                    if depth += 1; depth == maxCallRecursion { break }
                }
            }
        } else {
            for c := cc.programContext(); c != nil; c = c.caller() {
                if c.program() == prog {
                    if depth += 1; depth == maxCallRecursion { break }
                }
            }
        }
        if depth < maxCallRecursion {
            // continues
        } else if c := /*cc.traversal()*/cc.programContext(); c != nil {
            var tt, _ = c.autoGet("@")
            prompt(ctx, "%v: max recursion call (%d)\n", fullnameOrStrval(ctx, tt), depth)
            warn(ctx, "max recursion call (%d)\n", depth).of(tt).debug(1)

            const collapse = false
            for ; c != nil; c = c.caller() {
                var n int
                if collapse { for next := c.caller(); next != nil; next = next.caller() {
                    if t, _ := next.autoGet("@"); t != nil && t.cmp(ctx, tt) == cmpEqual {
                        n += 1;  continue
                    }
                    if next.program() == c.program() { n += 1; c = next } else { break }
                }}

                var t, _ = c.autoGet("@")
                if prog := c.program(); prog == nil {
                    erro(ctx, "%v (@=%v)", entry, tt).at(entry.Position())
                    break
                } else if pos := prog.position; n > 0 {
                    erro(ctx, "%v (repeated %d times)", t, n).at(pos)
                } else if !collapse {
                    var d, _ = c.autoGet(">")
                    erro(ctx, "%v : %v", t, d).at(pos)
                } else if depth -= 1; maxCallRecursion - depth > 5 {
                    erro(ctx, "%v ... (%d)", t, maxCallRecursion - depth).at(pos)
                    break
                } else {
                    var d, _ = c.autoGet(">")
                    erro(ctx, "%v : %v", t, d).at(pos)
                }

                ctx.checkErrors(true) // dump immediately
            }
            errostack(ctx, depth, "#>", entry).debug(512)
            if false { fail(prog.position, "max call depth") }
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
        if isNil(a) {
            erro(ctx, "%v: nil entry target", target)
            errostack(ctx, 8, "%v", ctx).debug(20)
            return
        } else if file := prog.project.FindFile(ctx, a.Strval(ctx)); file != nil {
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
        } else if result, _ = ctx.autoGet("-"); !isNil(result) {
            // good!
        } else if !isNil(defaultVal) {
            result = defaultVal
        }

        if cc != nil { if p := cc.program(); p != nil && !isNil(result) {
            p.defaultVal = result
        }}
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
    traves = append(traves, prog.traverse(&normalTraverseContext{ ctx }, prog.depends)...)
    if errs := ctx.checkErrors(true); errs > 0 {
        s := traves.add(positional(ctx, prog.position), traveFail, nil)
        s.error = fmt.Errorf("%d errors counted", errs)
        prompt(ctx, "%v: execute failed, project %s; traves=%v\n", entry, proj, traves)
        warn(ctx, "%d errors while traversing prerequisites for %v", errs, target)
        if warnstack(ctx, 6, "").debug(8); true && options.failOnErrors {
            fail(prog.position, "fail by %d errors", ctx.totalErrors())
        }
        return
    } else if traves.has(traveCase, traveDone, traveFail, traveNext) {
        return
    }

    // Update order-only prerequisites
    ctx.autoSet("|", nil)
    traves = append(traves, prog.traverse(&orderTraverseContext{ ctx }, prog.ordered)...)
    if errs := ctx.checkErrors(true); errs > 0 {
        s := traves.add(positional(ctx, prog.position), traveFail, nil)
        s.error = fmt.Errorf("%d errors counted", errs)
        prompt(ctx, "%v: execute failed, project %s; traves=%v\n", entry, proj, traves)
        warn(ctx, "%d errors while traversing prerequisites for %v", errs, target)
        if warnstack(ctx, 6, "").debug(8); true && options.failOnErrors {
            fail(prog.position, "fail by %d errors", ctx.totalErrors())
        }
        return
    } else if traves.has(traveCase, traveDone, traveFail, traveNext) {
        return
    }

    if prog.language != "" || len(t.interpreted) > 0 || len(prog.recipes) == 0 {
        // does nothing
    } else if h, ok := ctx.autoGet("-"); !ok || isNil(h) {
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

func (prog *Program) traverse(ctx Context, prerequisites []Value) (traves travestates) {
    var asyncUnsafe bool = true
    if !asyncUnsafe { for _, prerequisite := range prerequisites {
        if _, ok := prerequisite.(*modifiergroup); !ok {
            asyncUnsafe = true; break
        }
    }}

    // const dbg = true
    const dbg = false
    var (
        stems = ctx.stems()
        g,  _ = ctx.autoGet("@")
        verb  = options.verbose || options.verboseBreaks
        mu sync.Mutex
        wg sync.WaitGroup
    )
    defer func() {
        wg.Wait()

        // NOTE: warnings if travestates go too large
        if true {
            // does nothing
        } else if tt := traves.of(traveObj, traveFile); len(tt) > 100 {
            prompt(ctx, "%v: traves=%d\n", g, len(traves))
            for i, s := range traves {
                prompt(ctx, "%v: %d. %v\n", g, i, s)
                if i > 10 { break }
            }
            warnstack(ctx, 5, "%d errors", ctx.checkErrors(true)).debug(16)
        }

        // FIXME: optimization: the traves may grow into large number of traveFile
    } ()

    if asyncUnsafe {
        var depends valueList
        ForPrerequisites: for _, prerequisite := range prerequisites {
            if _, ok := prerequisite.(*modifiergroup); ok {
                wg.Wait()
            } else {
                // wg.Add(1)
            }

            var t = prerequisite.traverse(ctx)
            if !t.has() { continue } else {
                // NOTE: the program should collect all travestates
                traves = append(traves, t...)
            }

            // NOTE: Must fetch $@ after every prerequisite traverse, because
            // NOTE: a prerequisite modifier may had changed it.
            var (
                target, _ = ctx.autoGet("@")
                depend, _ = ctx.autoGet(">")
            )
            if dbg || false && strings.Contains(prerequisite.Strval(ctx), "%.h.cmake") {
                warn(ctx, "%v", t).of(prerequisite)
                warn(ctx, "%T %v\n", depend, depend).of(depend)
                warn(ctx, "%T %v\n", prerequisite, prerequisite).of(prerequisite)
                warn(ctx, "%T %v\n", target, target).of(target).debug(1)
                defer func(v Value) { warn(ctx, "%v", t).of(v).debug(20) } (prerequisite)
            }

            depends.add(depend)

            if tt := t.of(traveFail); tt.has() {
                var (
                    m = ctx.stemmed()
                    isPatternContextForMyTarget = m != nil && m.target != nil &&
                        m.target.cmp(ctx, target) == cmpEqual
                )
                if isPatternContextForMyTarget {
                    // TODO: convert traveFail into traveNext for stemmed entries ?
                    const dbgInfoStates = false
                    for _, s := range tt {
                        var dbgFail2Next bool
                        var dependMine = depends.contains(s.depend)
                        if dbgInfoStates || false &&
                            strings.Contains(target.Strval(ctx), "polly/Config/config.h") {
                            var f = ctx.Project().FindFile(ctx , "polly/Config/config.h")
                            prompt(ctx, "%v: found in %v\n", f, ctx.Project())
                            prompt(ctx, "%v\n", t)
                            prompt(ctx, "%v\n", traves)
                            info(ctx, "  %v", s).at(s.pos)
                            info(ctx, "1. %T %v %s", target, target, target.Strval(ctx)).of(target)
                            info(ctx, "2. %T %v %s", depend, depend, depend.Strval(ctx)).of(depend)
                            info(ctx, "3. %T %v %v", s.depend, s.depend, dependMine).of(s.depend)
                            info(ctx, "4. %T %v", prerequisite, prerequisite).of(prerequisite)
                            infostack(ctx, 5, "#>", s.target/*, s.depend*/).debug(128)
                            dbgFail2Next = true
                        }
                        if s.error == traveTargetNotDefinedFile && dependMine {
                            // add traveNext to try the next pattern
                            // trave := traves.remove(s).add(ctx, traveNext, target)
                            // trave.depend = s.depend
                            traves.remove(s).add(ctx, traveNext, target)
                            if dbgFail2Next {
                                warn(ctx, "%v", tt)
                                warn(ctx, "%v", traves).debug(1)
                            }
                        } else {
                            erro(ctx, "  %v", s).at(s.pos)
                            erro(ctx, "1. %T %v %s", target, target, target.Strval(ctx)).of(target)
                            erro(ctx, "2. %T %v %s", depend, depend, depend.Strval(ctx)).of(depend)
                            erro(ctx, "3. %T %v %v", s.depend, s.depend, dependMine).of(s.depend)
                            erro(ctx, "4. %T %v", prerequisite, prerequisite).of(prerequisite)
                            errostack(ctx, 5, "#>").debug(10)
                        }
                        return
                    }
                }
                if true || dbg || verb {
                    var p = ctx.Project()
                    prompt(ctx, "%v:(%T): %T %v ; project=%s, stems=%v ; %v\n",
                        t, target, prerequisite, prerequisite, p, stems, m)

                    var a []interface{}
                    for _, s := range tt {
                        if pe, ok := s.error.(*fs.PathError); ok { // NOTE: pe.Path == s.target
                            warn(ctx, "%v: %v: %v", t, s.target, pe.Err).at(s.pos)
                        } else {
                            warn(ctx, "%v: %v: %v (%T)", t,
                                s.target, s.error, s.error).at(s.pos)
                        }
                        a = append(a, s.target) // if !s.target.Position().Same(&s.pos)
                    }
                    warnstack(ctx, 5, "#>", a...).debug(16)
                }
                return //break ForPrerequisites
            }

            if tt := t.of(traveDone); tt.has() {
                for _, s := range tt {
                    if g := s.target; g != nil &&
                        (target == g || target.cmp(ctx, g) == cmpEqual) {
                        return //break ForPrerequisites
                    }
                }
                // traves = traves.not(traveDone)
            }

            if tt := t.of(traveCase); tt.has() {
                for _, s := range tt {
                    if g := s.target; g != nil &&
                        (target == g || target.cmp(ctx, g) == cmpEqual) {
                        return //break ForPrerequisites
                    }
                }
                // traves = traves.not(traveCase)
            }

            if tt := t.of(traveNext); tt.has() {
                for _, s := range tt {
                    if g := s.target; g != nil &&
                        (target == g || target.cmp(ctx, g) == cmpEqual) {
                        // info(ctx, "%v", s).of(prerequisite).debug(16)
                        return //break ForPrerequisites
                    } else if d := s.depend; d != nil &&
                        (target == d || target.cmp(ctx, d) == cmpEqual) {
                        info(ctx, "%v", s).of(prerequisite).debug(16)
                        continue ForPrerequisites
                    }
                }
                if false { info(ctx, "%v", traves).of(prerequisite).debug(6) }
                // traves = traves.not(traveNext)
            }

            if tt := t.of(traveFile); tt.has() {
                // traves = traves.not(traveFile)
                continue
            }

            if tt := t.of(traveObj); tt.has() {
                // traves = traves.not(traveObj)
                continue
            }

            if tt := t.not(traveCase, traveDone, traveNext); tt.has() {
                var str string
                if g == target || g.cmp(ctx, target) == cmpEqual {
                    str = g.String()
                } else {
                    str = fmt.Sprintf("%v(%v)", g, target)
                }
                prompt(ctx, "%s: %v ; traves=%d\n", str, prerequisite, len(t))
                for i, s := range t { prompt(ctx, "%s: %d. %v\n", str, i, s) }
                errostack(ctx, 5, "").debug(16)
                return //break ForPrerequisites
            }
        }
        return
    } else if num := len(prerequisites); num > 0 {
        for _, prerequisite := range prerequisites {
            var target, _ = ctx.autoGet("@")
            warn(ctx, "%v: %T %v %v %v\n",
                target, prerequisite, prerequisite,
                ctx.entry().Target(), ctx.stems()).debug(1)
            wg.Add(1)
            go func (ctx Context) {
                defer checkPanicsErrors(ctx)
                defer wg.Done()
                if t := prerequisite.traverse(ctx); t.has() {
                    mu.Lock(); defer mu.Unlock()
                    traves = append(traves, t...)

                    var proj = ctx.Project()
                    if t = t.not(traveNext, traveCase, traveDone); t.has() {
                        for _, s := range t { warn(ctx, "%v: %v", prerequisite, s).at(s.pos) }
                        warnstack(ctx, 5, "%v: %v: %v", proj, ctx.entry(), prerequisite).debug(36)
                    }
                }
            } (ctx.spawn())
        }
        return
    } else {
        return
    }
}
