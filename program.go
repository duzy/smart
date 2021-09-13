//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    //"strconv"
    "sync"
    "time"
    "fmt"
)

type dependPatternUnfit struct {}
func (*dependPatternUnfit) Error() string { return "pattern unfit" }

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
    mutex sync.Mutex
}

func (prog *Program) Position() Position { return prog.position }
func (prog *Program) Project() *Project { return prog.project }
func (prog *Program) Scope() *Scope { return prog.scope }
/*
func (t *traversal) auto(name string, value Value) (auto *Def, err error) {
    var alt Object
    assert(t.closure != t.program.scope, "traversal closure must not be program scope")
    if auto, alt = t.closure.define(t, DefAutoVal, name, value); alt != nil {
        var found = false
        if auto, found = alt.(*Def); found {
            auto.val(t, value)
        } else {
            err = fmt.Errorf("`%v` name already taken (%T)", name, alt)
        }
    }
    if enable_assertions && auto != nil {
        assert(auto.value == value, "wrong auto value")
    }
    return
}

func (t *traversal) seta(prog *Program, args []Value) (params []*Def, err error) {
    var argnum int // setup named/number parameters ($1, $2, etc.)
    var rest []Value
    assert(t.closure != t.program.scope, "traversal closure must not be program scope")
    for _, a := range args {
        var def *Def
        //<!IMPORTANT: Don't translate Flag, Flag values are valid
        //         regular arguments. Pair values are special.
        if l, ok := a.(*List); ok && l.Len() == 1 { a = l.Elems[0] }
        if p, ok := a.(*Pair); ok {
            var name string
            if name, err = p.Key.Strval(t); err != nil {
                diag.errorOf(p.Key, "strval '%v' failed: %v", p.Key, err).debug(1)
                return
            } else if o := t.closure.Lookup(name); isNil(o) {
                rest = append(rest, a)
            } else if def, ok = o.(*Def); !ok {
                diag.errorOf(o, "object is not a Def: %v", o).debug(1)
                return
            } else {
                params = append(params, def)
                def.set(t, DefArgVal, p.Value)
                argnum += 1
            }
        } else { rest = append(rest, a) }
    }
    for _, a := range rest {
        var def *Def
        if l, ok := a.(*List); ok && l.Len() == 1 { a = l.Elems[0] }
        if argnum < len(prog.params) {
            def = prog.params[argnum]
            params = append(params, def)
            def.set(t, DefArgVal, a)
        } else {
            name := strconv.Itoa(argnum+1)
            if def, err = t.auto(name, a); err != nil {
                diag.errorOf(a, "arg: %v", err).debug(1)
                return
            }
            params = append(params, def)
            def.origin = DefArgVal
        }
        argnum += 1
    }
    return
}
*/
func (prog *Program) interpret(t *traversal, i interpreter, params []Value) (err error) {
    if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("Program.interpret(%s)", typeof(i)))) }

    var pos = t.Position()
    if !pos.IsValid() { pos = prog.position }
    /*for _, e := range t.breakers {
        if e.what != breakCase { return }
    }*/

    // Wait for prerequisites before interpretion
    t.wait(t)

    var value Value
    if value, err = i.Evaluate(t, params...); err != nil {
        diag.errorAt(pos, "evaluation failed: %v", err).debug(1)
        return
    } else if isNil(value) {
        // disgard nil value
    } else if _, okay := t.Set("-", value); !okay {
        diag.errorAt(pos, "set buffer value failed: %v", value).debug(1)
        return
    }

    if _, _, err = t.updateRecipesHash(t); err != nil {
        diag.errorAt(pos, "update recipes hash failed: %v", err).debug(1)
    } else {
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
                diag.errorOf(m.name, "get modifier name '%v' failed: %v", m.name, e).debug(1)
                return
            } else if s == name {
                ms = append(ms, m)
            }
        }
    }
    return
}

func (prog *Program) modify(t *traversal, m *modifier) (brks breakers) {
    // TODO: using rules in a different project to implement modifiers, e.g.
    //       [ foo.check-preprequisites ]
    //       [ foo.baaaar ]
    var (
        name string
        args []Value
        err error
    )
    if args, err = mergeresult2(expandall2(t, expandPlainValue, m.name)); err != nil {
        diag.errorOf(m.name, "expand modifier name '%v' failed: %v", m.name, err).debug(1)
        return
    } else if n := len(args); n == 0 {
        diag.errorOf(m.name, "modifier name '%v' is empty", m.name).debug(1)
        return
    } else if name, err = args[0].Strval(t); err != nil {
        diag.errorOf(args[0], "strval '%v' failed: %v", args[0], err).debug(1)
        return
    } else {
        args = append(args[1:], m.args...)
    }

    //var ctx = contextAt(m.position, t)
    if f, ok := modifiers[name]; ok {
        var value, _ = t.Get("-")
        // Special modifier processing (implicit interpretation) before (configure)
        if name == "configure" && len(t.interpreted) == 0 && len(prog.recipes) > 0 /*&& (isNil(value) || isNone(value))*/ {
            // Evaluate for configure modifier
            if i, ok := dialects["eval"]; ok && i != nil {
                if err = prog.interpret(t, i, args); err != nil {
                    diag.errorAt(m.position, "interpret failed: %v", err).debug(1)
                    return
                }
            }
        }
        if value, brks = f(t, args...); brks.has() {
            if tb := brks.not(breakCase, breakDone); tb.has() {
                t.batch(func() {
                    for _, brk := range tb {
                        switch brk.what {
                        case breakFail: diag.errorAt(brk.pos, "broken modifier %v with failure: %v", name, brk.message)
                        case breakErro: diag.errorAt(brk.pos, "broken modifier %v with error: %v", name, brk.error)
                        default: diag.errorAt(brk.pos, "broken modifier %v (%v)", name, brk.what)
                        }
                    }
                    diag.errorAt(m.position, "borken modifier %s %v", name, args).debug(6)
                })
            }
        } else if bv, _ := t.Get("-"); isNil(value) || value == bv {
            // does nothing
        } else if _, okay := t.Set("-", value); !okay {
            diag.errorAt(m.position, "setting buffer value failed: %v", value).debug(1)
        }
    } else if i, _ := dialects[name]; i != nil {
        if err = prog.interpret(t, i, args); err != nil {
            diag.errorAt(m.position, "interpret '%s' failed: %v", name, err).debug(1)
        }
    } else {
        diag.errorAt(m.position, "unknown modifier '%s'", name).debug(1)
    }
    return
}

const maxRecursion  = 16 //32 //64

func (prog *Program) execute(caller *traversal, entry *RuleEntry, args []Value) (result Value, brks breakers) {
    if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("Program.execute(%s)", entry.target))) }
    if optionEnableBenchspots { defer bench(spot("Program.execute")) }

    var pos = prog.position
    if !pos.IsValid() { pos = entry.Position() }
    if !pos.IsValid() { pos = entry.position }

    var isConfigureExecution bool = prog.configure
    if !isConfigureExecution && caller != nil {
        isConfigureExecution =  caller.isConfigureExecution
    }

    // only one thread allowed to enter this section
    if false { prog.mutex.Lock(); defer prog.mutex.Unlock() }

    defer func() {
        if n := diag.checkErrors(true); n > 0 && !isConfigureExecution {
            if caller != nil {
                var err error
                if n == 1 {
                    err = fmt.Errorf("execution yields an error for %v", entry)
                } else {
                    err = fmt.Errorf("execution yields %d errors for %v", n, entry)
                }
                //caller._break(pos, breakErro).error = err
                //if false { diag.warnAt(pos, "break: %v", err).debug(1) }
                brks.add(pos, breakErro).error = err
            }
        }
    } ()

    var recursion int
    for c := caller; c != nil; c = c.caller() {
        if c.program == prog { recursion += 1 }
    }
    if recursion >= maxRecursion {
        diag.errorAt(pos, "exceeds max recursion: %v", entry.target).debug(1)
        for c := caller; c != nil; c = c.caller() {
            var n int
            for next := c.caller(); next != nil; next = next.caller() {
                if next.program == c.program { n += 1; c = next } else { break }
            }
            var ct, _ = c.Get("@")
            if n > 0 {
                diag.errorAt(c.program.position, "%v (repeats %d times)", ct, n)
            } else {
                diag.errorAt(c.program.position, "%v", ct)
            }
        }
        diag.errorAt(pos, "too many recursion (%d) (%v)", recursion, entry.target).debug(1)
        return
    }

    // The program scope must be protected!
    /*for _, o := range prog.scope.elems { if d, okay := o.(*Def); okay {
        defer func(d *Def, v Value) { d.value = v } (d, d.value)
    }}*/

    var t = &traversal{
        //Context: caller/*.Context*/,
        //caller: caller,
        isConfigureExecution: isConfigureExecution,
        callContext: callContext{ caller, make(callContextDefs) },
        execRec: make(map[Value]int),
        closure: prog.scope, //NewScope(pos, prog.scope, prog.project, "exec "+prog.scope.comment),
        project: prog.project,
        program: prog,
        start: time.Now(),
        entry: entry,
        print: true,
        args: args,
    }

    /*
    var (
        none = MakeNone(pos)
        stem Value = none
        err error
    )
    if caller != nil {
        if optionTraceTraversalNestIndent { t.traceLevel = caller.traceLevel }
        if t.stems = caller.stems; t.stems != nil { stem = MakeString(pos, t.stems[0]) }
    }
    if t.def.target,  err = t.auto("@", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.depend0, err = t.auto("<", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.dependx, err = t.auto(">", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.depends, err = t.auto("^", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.ordered, err = t.auto("|", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.grepped, err = t.auto("~", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.updated, err = t.auto("?", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.buffer,  err = t.auto("-", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.stem,    err = t.auto("*", stem); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.params,  err = t.seta(prog.params, args); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    */
    var err error
    if caller != nil {
        if optionTraceTraversalNestIndent { t.traceLevel = caller.traceLevel }
        if t.stems = caller.stems; t.stems != nil {
            t.Set("*", MakeString(pos, t.stems[0]))
        }
    }
    if t.params, err = t.setArgs(prog.params, args); err != nil {
        diag.errorAt(pos, "%v", err).debug(1)
        return
    }

    // Note: must enter work directory (cd) before setting cloctx
    var (
        ctx = contextAt(prog.position, t)
        alreadyUpdated bool
        enterBack *enterec
    )
    if len(cd.stack) > 0 { enterBack = cd.stack[0] }
    if err = enter(t, prog.project.absPath); err != nil {
        diag.errorAt(pos, "enter project '%v' failed: %v", prog.project, err).debug(1)
        return
    }
    defer func(scc closurecontext, swd string) {
        setclosure(scc) // restore closure context

        if e := leave(t, prog, enterBack); e != nil {
            // NOTE: err could be breakCase, breakDone, etc.
            if err == nil { err = e } else {
                diag.errorAt(pos, "leave project '%v' failed: %v", prog.project, err).debug(1)
            }
        }
        prog.project.changedWD = swd

        if err != nil {
            diag.errorAt(pos, "execution failed: %v", err).debug(6)
            return
        }

        var defaultVal = prog.defaultVal
        prog.defaultVal = nil

        if !isNil(result) && !isNone(result) {
            // good!
        } else if result, _ = t.Get("-"); !isNil(result) {
            // good!
        } else if !isNil(defaultVal) {
            result = defaultVal
        }

        if caller != nil && caller.program != nil && !isNil(result) {
            caller.program.defaultVal = result
        }
    } (setclosure(cloctx.unshift(prog.scope)), prog.project.changedWD)

    // Select the right target value before setting parameters,
    // because the target could be overrided by parameters.
    switch a := t.entry.target.(type) {
    case *Flag:
        t.Set("@", a)
        // Flag target (-foo) turns off printing automatically
        t.print = false
    case *File:
        t.Set("@", a)
        //alreadyUpdated = a.info != nil && a.updated
    default:
        var name string
        var target = t.entry.target
        if name, err = target.Strval(ctx); err != nil {
            diag.errorAt(pos, "strval '%v' failed: %v", target, err).debug(1)
            return
        }
        if file := prog.project.FindFile(ctx, name); file != nil {
            //alreadyUpdated = file.info != nil && file.updated
            target = file
        }
        t.Set("@", target)
    }
    if alreadyUpdated {
        var target, _ = t.Get("@")
        if optionTraceTraversal { t.tracef("Program.execute: '%v' already updated (%v)", target, t.targets) }
        if options.verbose { diag.infoAt(pos, "'%v' already updated", target) }
        if false { diag.warnAt(pos, "'%v' already updated", target).debug(true, 1) }
        if false { return }
    }

    if t.print && t.entry.class == UseRuleEntry { t.print = false }
    if t.print && prog.configure { t.print = false }
    cd.stack[0].silent = !t.print
    return prog.exec(t)
}

func (prog *Program) exec(t *traversal) (result Value, brks breakers) {
    if optionEnableBenchmarks { defer bench(mark("traversal.exec")) }
    if optionEnableBenchspots { defer bench(spot("traversal.exec")) }
    if optionTraceExec {
        var d = t.depth()
        var t, _ = t.Get("@")
        var s = fmt.Sprintf("%s: %v (%p, exec.depth=%d)", typeof(t), t, t, d)
        defer un(trace(t_exec, s))
    }

    var target, _ = t.Get("@")
    t.execRec[target] += 1
    if false { if t.execRec[target] > 1 {
        if optionTraceExec { t_exec.trace(fmt.Sprintf("exec: %v", target)) }
        return
    }}

    var pos = prog.position
    var ctx = contextAt(pos, t)

    // Update normal prerequisites
    if brks = t.normalPrerequisites(ctx); brks.has() {
        return
    } else if n := diag.checkErrors(true); n > 0 {
        brks.add(pos, breakFail).message = fmt.Sprintf("traverse prerequisites failed (%d errors)", n)
        t.batch(func() {
            diag.warnAt(pos, "%d errors while traversing prerequisites for %v", n, target).debug(8)
            t.traceCallStack(pos, -1, "call stack for %v:", target)
        })
        return
    }

    // Update order-only prerequisites
    if brks = t.orderOnlyPrerequisites(ctx); brks.has() {
        return
    } else if n := diag.checkErrors(true); n > 0 {
        brks.add(pos, breakFail).message = fmt.Sprintf("traverse prerequisites failed (%d errors)", n)
        t.batch(func() {
            diag.warnAt(pos, "%d errors while traversing prerequisites for %v", n, target).debug(8)
            t.traceCallStack(pos, -1, "call stack for %v:", target)
        })
        return
    }

    // Update grapped files
    /* modifierGrepFiles already did the traverse()
    if t.greppedFiles(pos);           t.hasBreakers() { return }
    if n := diag.checkErrors(true); n > 0 {
        diag.warnAt(pos, "%d errors while traversing prerequisites for %v", n, target).debug(1)
        return
    }
    */

    var value, _ = t.Get("-")
    if len(t.interpreted) == 0 && len(prog.recipes) > 0 && (isNil(value) || isNone(value)) {
        // Using the default statements interpreter.
        if i, ok := dialects["eval"]; ok && i != nil {
            if err := prog.interpret(t, i, nil); err != nil {
                diag.errorAt(pos, "%v", err).debug(1)
            }
        } else {
            diag.errorAt(pos, "no default dialect").debug(1)
        }
    }
    return
}

func (t *traversal) prerequisites(ctx Context, prerequisites []Value) (brks breakers) {
    // IMPORTANT: don't expand the args here. The prerequisites like
    // '$(or &@,...)' have to be expanded when it's used (e.g. compare).
    var pos = ctx.Position()
    var target, _ = t.Get("@")
    if true {
        for _, prerequisite := range prerequisites {
            if brks = prerequisite.traverse(t); brks.has() {
                var tb = brks.not(breakNext, breakCase, breakDone)
                if len(tb) > 0 && len(t.stems) > 0 && false {
                    diag.warnAt(pos, "broken traversal: %v (target = %v, stems = %v)", tb[0].what,
                        target, t.stems).debug(1)
                }
                break
            }
        }
    } else if num := len(prerequisites); num > 0 {
        var (
            mu sync.Mutex
            wg sync.WaitGroup
        )
        wg.Add(num)
        for _, prerequisite := range prerequisites {
            go func () {
                defer recoverPanics(ctx)
                defer wg.Done() // minus 1
                if tb := prerequisite.traverse(t); tb.has() {
                    mu.Lock(); defer mu.Unlock()
                    brks = append(brks, tb...)
                }
            } ()
        }
        if wg.Wait(); brks.has() {
            var tb = brks.not(breakNext, breakCase, breakDone)
            if len(tb) > 0 && len(t.stems) > 0 && false {
                diag.warnAt(pos, "broken traversal: %v (target = %v, stems = %v)", tb[0].what,
                    target, t.stems).debug(1)
            }
        }
    }
    return
}

func (t *traversal) normalPrerequisites(ctx Context) (brks breakers) {
    if optionTraceExec        { defer un(trace(t_exec, "^")) }
    if optionEnableBenchmarks { defer bench(mark("traversal.normalPrerequisites")) }
    /*defer func(t0, tx, ta *Def) {
        if t.target0, t.targetx, t.targets = t0, tx, ta; len(t.updated) > 0 {
            t.def.updated.value = t.updated[0].target // $?
            for _, u := range t.updated[1:] {
                t.def.updated.append(ctx, u.target)
            }
        }
    } (t.target0, t.targetx, t.targets)*/
    t.target0, t.targetN, t.targetX = "<", ">", "^"
    return t.prerequisites(ctx, t.program.depends)
    /*if n := len(t.targets); n > 0 {
        t.Set("<", t.targets[0])
        t.Set(">", t.targets[n-1])
        if n == 1 {
            t.Set("^", t.targets[0])
        } else {
            t.Set("^", MakeList(t.targets[0].Position(), t.targets))
        }
    }*/
}

func (t *traversal) orderOnlyPrerequisites(ctx Context) (brks breakers) {
    if optionTraceExec        { defer un(trace(t_exec, "|")) }
    if optionEnableBenchmarks { defer bench(mark("traversal.orderOnlyPrerequisites")) }
    /*defer func(t0, tx, ta *Def) {
        t.target0, t.targetx, t.targets = t0, tx, ta
    } (t.target0, t.targetx, t.targets)*/
    t.target0, t.targetN, t.targetX = "", "", "|"
    return t.prerequisites(ctx, t.program.ordered)
}

// DEPRECATED
func (t *traversal) greppedFiles(ctx Context) (brks breakers) {
    if optionTraceExec        { defer un(trace(t_exec, "~")) }
    if optionEnableBenchmarks { defer bench(mark("traversal.greppedFiles")) }
    /*defer func(t0, tx, ta *Def) {
        t.target0, t.targetx, t.targets = t0, tx, ta
    } (t.target0, t.targetx, t.targets)*/
    t.target0, t.targetN, t.targetX = "", "", "~"
    return t.prerequisites(ctx, t.grepped)
}
