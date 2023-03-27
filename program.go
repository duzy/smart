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
    sync.Mutex
    sync.WaitGroup
    by    modifierSetDirtyPatsOpts
    projs []*Project
    prog *Program
    params []string // $0, $1, $2, ...
    dirt string // reason of outdated

    /// traverseContext

    start time.Time // start time

    execRec map[Value]int

    calleeErrs []error
    calleeErrsM sync.Mutex

    targets []Value // all targets def
    grepped []Value
    grepping bool

    traceLevel int

    interpreted []interpreter

    print bool // printing work directories (Entering/Leaving)
}
func (pc *programContext) caller() *programContext { return pc.Context.programContext() }
func (pc *programContext) inner() Context { return &pc.autoContext }
func (pc *programContext) wait() { pc.WaitGroup.Wait() }
func (pc *programContext) aquireLock() (unlock func()) {
    pc.Lock() ; return func() { pc.Unlock() }
}
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
    const enableDirtyMark = true
    if !enableDirtyMark {
        // does nothing
    } else if tt := merge(autoGet(pc, "@")); len(tt) == 0 {
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
    params   []*def
    depends  []Value // normal
    ordered  []Value // order-only
    recipes  []Value
    defaultVal Value
    language  string
    changedWD string
    configure bool
    debug_traverse int
}
func (prog *Program) Position() Position { return prog.position }
func (prog *Program) Project() *Project { return prog.project }
func (prog *Program) Scope() *Scope { return prog.scope }
func (prog *Program) interpret(ctx Context, i interpreter, params []Value) (err error) {
    if pos := ctx.Position(); !pos.IsValid() && prog.position.IsValid() {
        ctx = at(ctx, prog.position)
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
    } else if def, prev := ctx.autoSet("-", value); def == nil {
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
    } else if t := ctx.programContext(); t != nil {
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
        args = mergex(ctx, plain, m.name)
    )
    if n := len(args); n == 0 {
        erro(of(ctx,m.name), "modifier name '%v' is empty", m.name).debug(1)
        return
    } else {
        name = args[0].Strval(ctx)
        args = append(args[1:], m.args...)
    }

    var proj = ctx.Project()
    if f, ok := modifiers[name]; ok {
        var value Value //= autoGet(ctx, "-")
        // Special modifier processing (implicit interpretation) before (configure)
        if len(ctx.programContext().interpreted) == 0 && len(prog.recipes) > 0 && name == "configure" {
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
        if value, traves = f(at(ctx, m.position), args...); traves.has() {
            if t := traves.not(traveCase, traveNext, traveDone); false && t.has() {
                if options.verbose || options.verboseBreaks {
                    var _, ent, _ = entryStr(ctx, ctx.entry())
                    prompt(ctx, "%v: %s failed for %s\n", ent, name, proj)
                    for _, s := range t { warn(at(ctx,s.pos), "%v: %s: %v", proj, name, s) }
                    warnstack(ctx, 5, "").debug(16)
                }
            }
            return
        } else if h := autoGet(ctx,"-"); h == nil || isNil(value) || value == h {
            // does nothing
        } else if ctx.autoSet("-", value); false {
            var _, ent, _ = entryStr(ctx, ctx.entry())
            prompt(ctx, "%v: %s failed for %s\n", ent, name, proj)
            erro(ctx, "setting buffer value failed: %v", value)
            errostack(ctx, 3, "(%T):", ctx).debug(1)
            fail(m.Position(), "%s failed for project %s", name, proj)
        }
    } else if i, _ := dialects[name]; i != nil {
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
func (t normalTraverseContext) traversed(target Value) (targets []Value) {
    if targets = t.Context.traversed(target); len(targets) > 0 {
        t.autoSet("^", MakeList(t.Position(), targets...))
        t.autoSet("<", targets[0])
        t.autoSet(">", targets[len(targets)-1])
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
            if v := d.Call(ctx); !isTrivial(v) {
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
      var envars []*Pair // disclosed values
    if def, _ := prog.scope.Lookup(TheShellEnvarsDef).(*def); def != nil {
        if l, _ := def.value.(*List); l != nil {
            for _, v := range l.Elems {
                var t Value
                if t = v.expand(ctx, expandClosure); isNil(t) { t = v }
                if p, ok := t.(*Pair); ok {
                    envars = append(envars, p)
                } else {
                    erro(of(ctx,t), "env expecting pairs: %T", t).debug(1)
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
        ctx Context = cc
        entry = cc.entry()
        args  = cc.arguments()
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
    if options.verbose { info(ctx, "%v: %v", entry, args).debug(1) }

    var pc = programContext{
        autoContext: autoContext{ Context:cc, defs:make(autoDefMap) },
        prog: prog,

        execRec: make(map[Value]int),
        start: time.Now(),
        print: true,
    }
    ctx = &pc

    defer func() {
        var (
            targets = autoGet(ctx, "@")
            depends = autoGet(ctx, "^")
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
            if !ctx.isConfiguration() && cc != nil {
                s := traves.add(ctx, traveFail, targets)
                if errs == 1 {
                    s.error = fmt.Errorf("execution yields an error for %v", str)
                } else {
                    s.error = fmt.Errorf("execution yields %d errors for %v", errs, str)
                }
            }
            if tar != "" && tar != ent {
                prompt(ctx, "%s: %s: execution with %d errors, project %s\n",
                    ent, tar, errs, prog.project).debug(1)
            } else {
                prompt(ctx, "%s: execution with %d errors, project %s\n",
                    ent, errs, prog.project).debug(1)
            }
            warn(ctx, `%v: %d errors in execution "%s"`, prog.project, errs, str)
            warnstack(ctx, 8, "").debug(32)
            if options.failOnErrors { fail(prog.position, "fail by %d errors", errs) }
        }
    } ()

    if cc != nil {
        var depth, loop int = 0, -1
        var a = []Value{ autoGet(cc, "@") }
        ForPC: for c := cc.programContext(); c != nil; c = c.caller() {
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
                    // see also RuleEntry.traverse for the same skip.
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

                ctx.checkErrors(true) // dump immediately
            }
            errostack(ctx, depth, "#>", entry).debug(512)
            if false { fail(prog.position, "max call depth") }
            return
        }
        if stems := cc.stems(); stems != nil { ctx.autoSet("*", MakeString(pos, stems[0])) }
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

    if pc.print && entry.Class() == UseRuleEntry { pc.print = false }
    if pc.print && prog.configure { pc.print = false }
    cd.stack[0].silent = !pc.print

    if options.traceExec {
        var d = pc.depth()
        var t = autoGet(ctx, "@")
        var s = fmt.Sprintf("%s: %v (%p, exec.depth=%d)", typeof(t), t, t, d)
        defer un(trace(t_exec, s))
    }

    var proj = ctx.Project()

    // Update normal prerequisites
    ctx.autoSet("^", nil)
    ctx.autoSet("<", nil)
    ctx.autoSet(">", nil)
    traves = append(traves, prog.traverse(normalTraverseContext{ctx}, prog.depends)...)
    if errs := ctx.checkErrors(true); errs > 0 {
        s := traves.add(at(ctx, prog.position), traveFail, nil)
        s.error = fmt.Errorf("%d errors counted", errs)
        prompt(ctx, "%v: execute failed, project %s; traves=%v\n",
            entry, proj, traves).debug(1)
        warn(ctx, "%d errors while traversing prerequisites for %v", errs, autoGet(ctx,"@"))
        if warnstack(ctx, 6, "").debug(8); true && options.failOnErrors {
            fail(prog.position, "fail by %d errors", ctx.totalErrors())
        }
        return
    } else if traves.has(traveCase, traveDone, traveFail, traveNext) {
        return
    }

    // Update order-only prerequisites
    ctx.autoSet("|", nil)
    traves = append(traves, prog.traverse(orderTraverseContext{ctx}, prog.ordered)...)
    if errs := ctx.checkErrors(true); errs > 0 {
        s := traves.add(at(ctx, prog.position), traveFail, nil)
        s.error = fmt.Errorf("%d errors counted", errs)
        prompt(ctx, "%v: execute failed, project %s; traves=%v\n",
            entry, proj, traves).debug(1)
        warn(ctx, "%d errors while traversing prerequisites for %v", errs, autoGet(ctx,"@"))
        if warnstack(ctx, 6, "").debug(8); true && options.failOnErrors {
            fail(prog.position, "fail by %d errors", ctx.totalErrors())
        }
        return
    } else if traves.has(traveCase, traveDone, traveFail, traveNext) {
        return
    }

    if prog.language != "" || len(pc.interpreted) > 0 || len(prog.recipes) == 0 {
        // does nothing
    } else if d := autoGet(ctx,"-"); d == nil {
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
    const dbg = false

    var (
        // FIXME:NOTE: parallel prerequisite traversal does not work here, as they
        //             must be go in ordered. See List.traverse for parallel instead.
        parallel = false && options.parallel
        verb  = options.verbose || options.verboseBreaks
        num   = len(prerequisites)
        pc    = ctx.programContext()
        stemd = ctx.stemmed()
        stems = ctx.stems()
        entry = ctx.entry()
        // patEnt, isPatEnt = entry.(*PatternEntry)
        depends valueList
    )
    defer func() {
        pc.Wait()

        // NOTE: warnings if travestates go too large
        if true {
            // does nothing
        } else if tt := traves.of(traveObj, traveRule, traveFile); len(tt) > 100 {
            var g Value = autoGet(ctx, "@")
            prompt(ctx, "%v: traves=%d\n", g, len(traves))
            for i, s := range traves {
                prompt(ctx, "%v: %d. %v\n", g, i, s)
                if i > 10 { break }
            }
            warnstack(ctx, 5, "%d errors", ctx.checkErrors(true)).debug(16)
        }

        // FIXME: optimization: the traves may grow into large number of traveFile
    } ()

    if ent := autoGet(ctx, "@"); !parallel {
        ForPrerequisites: for i, prerequisite := range prerequisites {
            var ctx = at(ctx, prerequisite.Position())
            var _, g = prerequisite.(*modifiergroup)
            /****/ if u, y := prerequisite.(untraversed); y {
                warn(ctx, "%v: untraversed %v", ent, u.Value).debug(1)
                continue
            } else if u, y := prerequisite.(unexpanded); y {
                warn(ctx, "%v: unexpanded %v", ent, u.Value).debug(1)
                continue
            } else if g {
                pc.Wait()
            } else {
                if i == 0 { ctx.autoSet("<", prerequisite) }
                if false  { ctx.autoSet(">", prerequisite) }
            }

            var t = prerequisite.traverse(ctx)
            if false { if prog.project.name == "llvm.ADT" {
                warn(of(ctx,prerequisite), "%v: %v: %v", prog.project, entry, prerequisite).debug(1)
                for i, a := range t { info(of(ctx,prerequisite), "%v. %v %v", i, a.what, a) }
                infostack(of(ctx,prerequisite), 12, "%v: %v: %v %v", prog.project, ent, prerequisite, autoGet(ctx, ">")).debug(10)
                defer func(pre, v Value) {
                    infostack(of(ctx,pre), 12, "%v → %v → %v", pre, v, prerequisite).debug(20)
                } (prerequisite, autoGet(ctx, "^"))
            }}
            if false { if a := autoGet(ctx, "@"); strings.Contains(a.String(), "dlfcn_simple.") {
                var s = a.Strval(ctx)
                if f, y := a.(*File); y { s = f.fullname() }
                warn(of(ctx,prerequisite), "%v: %v: %s, %v %T ; %v", entry, a, s, prerequisite, prerequisite, stemd).debug(1)
                for i, a := range t { info(of(ctx,prerequisite), "%v. %v %v", i, a.what, a) }
                infostack(of(ctx,prerequisite), 12, "%v: %v: %v %v", prog.project, ent, prerequisite, autoGet(ctx, ">")).debug(10)
                defer func(pre, l, r, v Value) {
                    infostack(of(ctx,pre), 12, "%v → %v %v %v → %v", pre, l, r, v, prerequisite).debug(20)
                } (prerequisite, autoGet(ctx, "<"), autoGet(ctx, ">"), autoGet(ctx, "^"))
            }}
            if false { if a := autoGet(ctx, "@"); a.String() == "llvm-tools-driver" {
                warn(of(ctx,prerequisite), "%v: %v: %v %T", entry, a, prerequisite, prerequisite).debug(1)
                for i, a := range t { info(of(ctx,prerequisite), "%v. %v %v", i, a.what, a) }
                info(of(ctx,prerequisite), "%v: %v %v", a, prerequisite, autoGet(ctx, ">")).debug(1)
                defer func(s Value) { info(of(ctx,prerequisite), "%v → %v", prerequisite, s).debug(10) } (autoGet(ctx, "^"))
            }}

            if prog.debug_traverse > 0 {
                if true { prog.debug_traverse -= 1 }
                for i, a := range t { info(of(ctx,prerequisite), "%v. %v %v", i, a.what, a) }
                infostack(of(ctx,prerequisite), 12, "%v: %v: %v %v", prog.project, ent, prerequisite, autoGet(ctx, ">")).debug(10)
            }
            if !t.has() { continue } else {
                traves = append(traves, t.not(traveNext)...)
            }

            var (
                target = autoGet(ctx, "@") // fetch updated $@
                depend = autoGet(ctx, ">") // fetch updated $>
                isPatternStemmedForTarget = stemd != nil && stemd.target != nil &&
                    eq(ctx, stemd.target, target)
            )
            if depend != nil { depends.add(depend) }
            if tt := t.of(traveFail); tt.has() {
                if isPatternStemmedForTarget {
                    // TODO: convert traveFail into traveNext for stemmed entries ?
                    for _, s := range tt {
                        var dbgFail2Next bool
                        var dependMine = depends.contains(s.depend)
                        if s.error == traveTargetNotDefinedFile && dependMine {
                            // add traveNext to try the next pattern
                            // trave := traves.remove(s).add(ctx, traveNext, target)
                            // trave.depend = s.depend
                            traves.remove(s).add(ctx, traveNext, target)
                            if dbgFail2Next {
                                warn(ctx, "%v", tt)
                                warn(ctx, "%v", traves).debug(1)
                            }
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
                if true || dbg || verb {
                    var p = ctx.Project()
                    prompt(ctx, "%v:(%T): %T %v ; project=%s, stems=%v ; %v\n",
                        t, target, prerequisite, prerequisite, p, stems, stemd)

                    var a []interface{}
                    for _, s := range tt {
                        if pe, ok := s.error.(*fs.PathError); ok { // NOTE: pe.Path == s.target
                            warn(at(ctx,s.pos), "%v: %v: %v", t, s.target, pe.Err)
                        } else {
                            warn(at(ctx,s.pos), "%v: %v: %v (%T)", t, s.target, s.error, s.error)
                        }
                        a = append(a, s.target) // if !s.target.Position().Same(&s.pos)
                    }
                    warnstack(ctx, 5, "#>", a...).debug(16)
                }
                return // fail
            }

            if tt := t.of(traveNext); tt.has() {
                var deps []Value
                for _, s := range tt {
                    if eq(ctx, target, s.target) { deps = append(deps, s.depend) } else
                    if eq(ctx, target, s.depend) {
                        info(of(ctx,prerequisite), "%v %v ; %v", target, depend, prerequisite)
                        info(of(ctx,prerequisite), "%v %v ; %d", s.target, s.depend,  len(tt)).
                            debug(1)
                        continue ForPrerequisites
                    }
                }

                var _, isPP = prerequisite.(*PercPattern)
                if isPP && isPatternStemmedForTarget && len(deps) == 1 {
                    if deps[0] == nil           { return } // %.h : %.h.cmake configure-file($>,$@)
                    if eq(ctx, depend, deps[0]) { return } // %.o : %.cpp
                } else {
                    if false {
                        for i, dep := range deps { info(of(ctx,prerequisite), "%d. %v %v", i, target, dep) }
                        info(of(ctx,prerequisite), "%v %v", target, prerequisite).debug(1)
                    }
                    continue ForPrerequisites
                }
            }

            if tt := t.of(traveDone); tt.has() {
                for _, s := range tt {
                    if eq(ctx, target, s.target) { return }
                }
            }

            if tt := t.of(traveCase); tt.has() {
                for _, s := range tt {
                    if eq(ctx, target, s.target) { return }
                }
            }

            if tt := t.of(traveFile); tt.has() {
                continue ForPrerequisites
            }

            if tt := t.of(traveRule); tt.has() {
                continue ForPrerequisites
            }

            if tt := t.of(traveObj); tt.has() {
                continue ForPrerequisites
            }

            if tt := t.not(traveCase, traveDone, traveNext); tt.has() {
                var str string
                if eq(ctx, ent, target) {
                    str = ent.String()
                } else {
                    str = fmt.Sprintf("%v(%v)", ent, target)
                }
                prompt(ctx, "%s: %v ; traves=%d\n", str, prerequisite, len(t))
                for i, s := range t { prompt(ctx, "%s: %d. %v\n", str, i, s) }
                errostack(ctx, 5, "").debug(16)
                return //break ForPrerequisites
            }
        }
        return
    } else if num > 0 {
        const (
            deferWait = false
            stateCont int = 0
            stateBrek     = 1
        )
        var state int
        for i, prerequisite := range prerequisites {
            var _, g = prerequisite.(*modifiergroup)
            /****/ if u, y := prerequisite.(untraversed); y {
                warn(ctx, "%v: untraversed %v", ent, u.Value).debug(1)
                continue
            } else if u, y := prerequisite.(unexpanded); y {
                warn(ctx, "%v: unexpanded %v", ent, u.Value).debug(1)
                continue
            } else if g {
                if !deferWait { pc.Wait() }
            } else {
                if i == 0 { ctx.autoSet("<", prerequisite) }
                if false  { ctx.autoSet(">", prerequisite) }
            }

            pc.Add(1)
            go func (ctx Context, i int, prerequisite Value) {
                defer func() {
                    pc.Done()
                    checkFailure(ctx)
                } ()

                var set = func(s int) {
                    var unlock = ctx.aquireLock()
                    state = s
                    if unlock != nil { unlock() }
                }

                ctx.autoSet(">", prerequisite)

                var t = prerequisite.traverse(ctx)
                if false { if prerequisite.String() == ".test.fxxbar" {
                    for i, a := range t { info(of(ctx,prerequisite), "%v. %v %v", i, a.what, a) }
                    infostack(of(ctx,prerequisite), 12, "%v: %v: %v %v", prog.project, ent, prerequisite, autoGet(ctx, ">")).debug(10)
                    defer func(pre, v Value) {
                        infostack(of(ctx,pre), 12, "%v → %v → %v", pre, v, prerequisite).debug(20)
                    } (prerequisite, autoGet(ctx, "^"))
                }}
                if false { if a := autoGet(ctx, "@"); a.String() == "llvm-tools-driver" {
                    for i, a := range t { info(of(ctx,prerequisite), "%v. %v %v", i, a.what, a) }
                    info(of(ctx,prerequisite), "%v: %v %v", a, prerequisite, autoGet(ctx, ">")).debug(1)
                    defer func(s Value) { info(of(ctx,prerequisite), "%v -> %v", prerequisite, s).debug(4) } (autoGet(ctx, "^"))
                }}

                if !t.has() { return }

                var brek bool
                var target, depend Value
                var unlock = ctx.aquireLock()
                if state == stateBrek { brek = true } else {
                    traves = append(traves, t.not(traveNext)...)
                    target = autoGet(ctx, "@") // fetch updated $@
                    depend = autoGet(ctx, ">") // fetch updated $>
                    if depend != nil { depends.add(depend) }
                }
                if unlock != nil { unlock() }

                if brek { return } else
                if tt := t.of(traveFail); tt.has() {
                    var (
                        m = ctx.stemmed()
                        isPatternStemmedForTarget = m != nil && m.target != nil &&
                            eq(ctx, m.target, target)
                    )
                    if isPatternStemmedForTarget {
                        // TODO: convert traveFail into traveNext for stemmed entries ?
                        const dbgInfoStates = false
                        for _, s := range tt {
                            var dbgFail2Next bool
                            var dependMine = depends.contains(s.depend)
                            if s.error == traveTargetNotDefinedFile && dependMine {
                                // add traveNext to try the next pattern
                                // trave := traves.remove(s).add(ctx, traveNext, target)
                                // trave.depend = s.depend
                                traves.remove(s).add(ctx, traveNext, target)
                                if dbgFail2Next {
                                    warn(ctx, "%v", tt)
                                    warn(ctx, "%v", traves).debug(1)
                                }
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
                                erro(of(ctx,s.depend), "3. %T %v mine=%v", s.depend, s.depend, dependMine)
                                erro(of(ctx,prerequisite), "4. %T %v", prerequisite, prerequisite)
                                errostack(ctx, 5, "#>").debug(10)
                            }

                            set(stateBrek)
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
                                warn(at(ctx,s.pos), "%v: %v: %v", t, s.target, pe.Err)
                            } else {
                                warn(at(ctx,s.pos), "%v: %v: %v (%T)", t, s.target, s.error, s.error)
                            }
                            a = append(a, s.target) // if !s.target.Position().Same(&s.pos)
                        }
                        warnstack(ctx, 5, "#>", a...).debug(16)
                    }

                    set(stateBrek)
                    return
                }

                if tt := t.of(traveDone); tt.has() {
                    for _, s := range tt {
                        if eq(ctx, target, s.target) {
                            set(stateBrek)
                            return
                        }
                    }
                }

                if tt := t.of(traveCase); tt.has() {
                    for _, s := range tt {
                        if eq(ctx, target, s.target) {
                            set(stateBrek)
                            return
                        }
                    }
                }

                if tt := t.of(traveNext); tt.has() {
                    var deps []Value
                    for _, s := range tt {
                        if eq(ctx, target, s.target) { deps = append(deps, s.depend) } else
                        if eq(ctx, target, s.depend) {
                            info(of(ctx,prerequisite), "%v %v ; %v", target, depend, prerequisite)
                            info(of(ctx,prerequisite), "%v %v ; %d", s.target, s.depend,  len(tt)).debug(1)
                            set(stateCont)
                            return
                        }
                    }
                    if len(deps) == 1 {
                        // %.h : %.h.cmake configure-file($>,$@)
                        // %.o : %.cpp
                        if deps[0] == nil || eq(ctx, depend, deps[0]) {
                            set(stateBrek)
                            return
                        }
                    } else {
                        if false {
                            for i, dep := range deps { info(of(ctx,prerequisite), "%d. %v %v", i, target, dep) }
                            info(of(ctx,prerequisite), "%v %v", target, prerequisite).debug(1)
                        }
                        set(stateCont)
                        return
                    }
                }

                if tt := t.of(traveFile); tt.has() {
                    set(stateCont)
                    return
                }

                if tt := t.of(traveRule); tt.has() {
                    set(stateCont)
                    return
                }

                if tt := t.of(traveObj); tt.has() {
                    set(stateCont)
                    return
                }

                if tt := t.not(traveCase, traveDone, traveNext); tt.has() {
                    var str string
                    if eq(ctx, ent, target) {
                        str = ent.String()
                    } else {
                        str = fmt.Sprintf("%v(%v)", ent, target)
                    }
                    prompt(ctx, "%s: %v ; traves=%d\n", str, prerequisite, len(t))
                    for i, s := range t { prompt(ctx, "%s: %d. %v\n", str, i, s) }
                    errostack(ctx, 5, "").debug(16)

                    set(stateBrek)
                    return
                }
            } (ctx.spawn(ctx), i, prerequisite)
            if g && deferWait { pc.Wait() }
        }
        return
    } else {
        return
    }
}
