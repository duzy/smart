//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "strconv"
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
}

func (prog *Program) Position() Position { return prog.position }
func (prog *Program) Project() *Project { return prog.project }
func (prog *Program) Scope() *Scope { return prog.scope }

func (prog *Program) auto(name string, value Value) (auto *Def, err error) {
    var alt Object
    if auto, alt = prog.scope.define(prog.project, name, value); alt != nil {
        var found = false
        if auto, found = alt.(*Def); found {
            auto.val(value)
        } else {
            err = fmt.Errorf("`%v` name already taken (%T)", name, alt)
        }
    }
    if enable_assertions {
        assert(auto.value == value, "wrong auto value")
    }
    return
}

func (prog *Program) interpret(pos Position, t *traversal, i interpreter, params []Value) (err error) {
    if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("Program.interpret(%s)", typeof(i)))) }

    var value Value
    for _, e := range t.breakers {
        if e.what != breakCase { return }
    }

    // Wait for prerequisites before interpretion
    t.wait(prog.position)

    if value, err = i.Evaluate(pos, t, params...); err != nil {
        diag.errorAt(pos, "evaluation failed: %v", err).
            debug(options.debugErrors, 1)
        return
    } else if !isNil(value) {
        if err = t.def.buffer.val(value); err != nil {
            diag.errorAt(pos, "set modify buffer value failed: %v", err).
                debug(options.debugErrors, 1)
            return
        }
    }

    if _, _, err = t.updateRecipesHash(); err != nil {
        diag.errorAt(pos, "update recipes hash failed: %v", err).
            debug(options.debugErrors, 1)
    } else {
        t.interpreted = append(t.interpreted, i)
    }
    return
}

func (prog *Program) getModifiers(name string) (ms []*modifier) {
    for _, d := range prog.depends {
        var g, ok = d.(*modifiergroup)
        if !ok { continue }
        for _, m := range g.modifiers {
            if s, e := m.name.Strval(); e != nil {
                diag.errorOf(m.name, "get modifier name '%v' failed: %v", m.name, e).
                    debug(options.debugErrors, 1)
                return
            } else if s == name {
                ms = append(ms, m)
            }
        }
    }
    return
}

func (prog *Program) modify(t *traversal, m *modifier) (err error) {
    // TODO: using rules in a different project to implement modifiers, e.g.
    //       [ foo.check-preprequisites ]
    //       [ foo.baaaar ]
    var ( name string; args []Value )
    if args, err = mergeresult2(expandall2(expandPlainValue, m.name)); err != nil {
        diag.errorOf(m.name, "expand modifier name '%v' failed: %v", m.name, err).
            debug(options.debugErrors,1)
        return
    } else if name, err = args[0].Strval(); err != nil {
        diag.errorOf(args[0], "strval '%v' failed: %v", args[0], err).
            debug(options.debugErrors,1)
        return
    } else {
        args = append(args[1:], m.args...)
    }

    if f, ok := modifiers[name]; ok {
        var value = t.def.buffer.value
        // Special modifier processing (implicit interpretation) before (configure)
        if name == "configure" && len(t.interpreted) == 0 && len(prog.recipes) > 0 /*&& (isNil(value) || isNone(value))*/ {
            // Evaluate for configure modifier
            if i, ok := dialects["eval"]; ok && i != nil {
                if err = prog.interpret(m.position, t, i, args); err != nil {
                    diag.errorAt(m.position, "interpret failed: %v", err).
                        debug(options.debugErrors,1)
                    return
                }
            }
        }
        if value, err = f(m.position, t, args...); err != nil {
            diag.errorAt(m.position, "%s: %v", name, err).
                debug(options.debugErrors, 1)
        } else if isNil(value) || (value == t.def.buffer && value != t.def.buffer.value) {
            // does nothing
        } else if err = t.def.buffer.val(value); err != nil {
            diag.errorAt(m.position, "setting modifier buffer value failed: %v", err).
                debug(options.debugErrors, 1)
        }
    } else if i, _ := dialects[name]; i != nil {
        if err = prog.interpret(m.position, t, i, args); err != nil {
            diag.errorAt(m.position, "interpret '%s' failed: %v", name, err).
                debug(options.debugErrors, 1)
        }
    } else {
        diag.errorAt(m.position, "unknown modifier '%s'", name).
            debug(options.debugErrors, 1)
    }
    return
}

func (prog *Program) args(args []Value) (params []*Def, restore func(), err error) {
    var argnum int // setup named/number parameters ($1, $2, etc.)
    var rest, values []Value // save values to restore after execution
    for _, d := range prog.params { values = append(values, d.value) }
    for _, a := range args {
        var def *Def
        //<!IMPORTANT: Don't translate Flag, Flag values are valid
        //         regular arguments. Pair values are special.
        if l, ok := a.(*List); ok && l.Len() == 1 { a = l.Elems[0] }
        if t, ok := a.(*Pair); ok {
            var s string
            if s, err = t.Key.Strval(); err != nil {
                diag.errorOf(t.Key, "strval '%v' failed: %v", t.Key, err).
                    debug(options.debugErrors, 1)
                return
            } else if o := prog.scope.Lookup(s); isNil(o) {
                rest = append(rest, a)
            } else if def, ok = o.(*Def); !ok {
                diag.errorOf(o, "object is not a Def: %v", o).
                    debug(options.debugErrors, 1)
                return
            } else {
                values = append(values, def.value)
                params = append(params, def)
                def.set(DefArg, t.Value)
                argnum += 1
            }
        } else { rest = append(rest, a) }
    }
    for _, a := range rest {
        var def *Def
        if l, ok := a.(*List); ok && l.Len() == 1 { a = l.Elems[0] }
        if argnum < len(prog.params) {
            def = prog.params[argnum]
            values = append(values, def.value)
            params = append(params, def)
            def.set(DefArg, a)
        } else {
            name := strconv.Itoa(argnum+1)
            if def, err = prog.auto(name, a); err != nil {
                diag.errorOf(a, "arg: %v", err).
                    debug(options.debugErrors, 1)
                return
            }
            values = append(values, def.value)
            params = append(params, def)
            def.origin = DefArg
        }
        argnum += 1
    }
    restore = func() {
        var nlen  = len(prog.params)
        for i, d := range prog.params { d.value = values[i] }
        for i, d := range params { d.value = values[nlen+i] }
    }
    return
}

const maxRecursion  = 16 //32 //64

func (prog *Program) execute(caller *traversal, entry *RuleEntry, args []Value) (result Value) {
    if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("Program.execute(%s)", entry.target))) }
    if optionEnableBenchspots { defer bench(spot("Program.execute")) }

    var pos = prog.position
    if !pos.IsValid() { pos = entry.Position() }
    if !pos.IsValid() { pos = entry.position }

    var isConfigureExecution bool = prog.configure
    if !isConfigureExecution && caller != nil {
        isConfigureExecution =  caller.isConfigureExecution
    }
    defer func() {
        if n := diag.checkErrors(true); n > 0 && !isConfigureExecution {
            if caller != nil {
                var err error
                if n == 1 {
                    err = fmt.Errorf("execution yields an error for %v", entry)
                } else {
                    err = fmt.Errorf("execution yields %d errors for %v", n, entry)
                }
                caller._break(pos, breakErro).error = err
                if false { diag.warnAt(pos, "break: %v", err).debug(1) }
            }
        }
    } ()

    var recursion int
    for c := caller; c != nil; c = c.caller {
        if c.program == prog { recursion += 1 }
    }
    if recursion >= maxRecursion {
        diag.errorAt(pos, "exceeds max recursion: %v", entry.target).
            debug(1)
        for c := caller; c != nil; c = c.caller {
            var n int
            for next := c.caller; next != nil; next = next.caller {
                if next.program == c.program { n += 1; c = next } else { break }
            }
            if n > 0 {
                diag.errorAt(c.program.position, "%v (repeats %d times)", c.def.target.value, n)
            } else {
                diag.errorAt(c.program.position, "%v", c.def.target.value)
            }
        }
        diag.errorAt(pos, "too many recursion (%d) (%v) (from %v)",
            recursion, entry.target, caller.def.target.value).
            debug(1)
        return
    }

    // The program scope must be protected!
    for _, o := range prog.scope.elems { if d, okay := o.(*Def); okay {
        defer func(d *Def, v Value) { d.value = v } (d, d.value)
    }}

    var t = &traversal{
        isConfigureExecution: isConfigureExecution,
        start: time.Now(),
        program: prog,
        project: prog.project,
        closure: prog.scope,
        visited: make(map[Value]int),
        group: new(sync.WaitGroup),
        entry: entry,
        args: args,
        caller: caller,
        print: true,
    }
    if caller != nil {
        defer func() {
            // Pass breakers to the caller for handling breakNext, breakCase, breakDone, etc.
            caller.breakers = append(caller.breakers, t.breakers...)
            for _, b := range t.breakers {
                switch b.what {
                case breakNext, breakCase, breakDone:
                case breakFail:
                    diag.errorAt(pos, "broken execution for %v (%v): %v", entry, t.def.target, b.message).
                        debug(1)
                case breakErro:
                    diag.errorAt(pos, "broken execution for %v (%v): %v", entry, t.def.target, b.error).
                        debug(1)
                default:
                    diag.errorAt(pos, "broken execution for %v (%v): %v", entry, t.def.target, b.what).
                        debug(16)
                }
            }
        } ()
    }

    var ( none = MakeNone(pos) ; stem Value = none; f func() ; err error )
    if t.caller != nil {
        if optionTraceTraversalNestIndent { t.traceLevel = t.caller.traceLevel }
        if t.stems = t.caller.stems; t.stems != nil { stem = MakeString(pos, t.stems[0]) }
    }
    if t.def.stem,    err = prog.auto("*", stem); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.target,  err = prog.auto("@", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.depend0, err = prog.auto("<", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.dependx, err = prog.auto(">", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.depends, err = prog.auto("^", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.ordered, err = prog.auto("|", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.grepped, err = prog.auto("~", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.updated, err = prog.auto("?", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.buffer,  err = prog.auto("-", none); err != nil { diag.errorAt(pos, "%v", err).debug(1); return }
    if t.def.params,f,err = prog.args(args)     ; err != nil { diag.errorAt(pos, "%v", err).debug(1); return } else {
        defer f()
    }

    // Note: must enter work directory (cd) before setting cloctx
    var (
        alreadyUpdated bool
        enterBack *enterec
    )
    if len(cd.stack) > 0 { enterBack = cd.stack[0] }
    if err = enter(prog, prog.project.absPath); err != nil {
        diag.errorAt(pos, "enter project '%v' failed: %v", prog.project, err).
            debug(1)
        return
    }
    defer func(scc closurecontext, swd string) {
        setclosure(scc) // restore closure context

        if e := leave(prog, enterBack); e != nil {
            // NOTE: err could be breakCase, breakDone, etc.
            if err == nil { err = e } else {
                diag.errorAt(pos, "leave project '%v' failed: %v", prog.project, err).
                    debug(1)
            }
        }
        prog.project.changedWD = swd

        if err != nil {
            diag.errorAt(pos, "execution failed: %v", err).
                debug(6)
            return
        }

        var defaultVal = prog.defaultVal
        prog.defaultVal = nil

        if !isNil(result) && !isNone(result) {
            // good!
        } else if !isNil(t.def.buffer.value) {
            result = t.def.buffer.Call(prog.position)
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
        t.def.target.val(a)
        // Flag target (-foo) turns off printing automatically
        t.print = false
    case *File:
        //alreadyUpdated = a.info != nil && a.updated
        t.def.target.val(a)
    default:
        var name string
        var target = t.entry.target
        if name, err = target.Strval(); err != nil {
            diag.errorAt(pos, "strval '%v' failed: %v", target, err).
                debug(1)
            return
        }
        if file := prog.project.FindFile(name); file != nil {
            //alreadyUpdated = file.info != nil && file.updated
            target = file
        }
        t.def.target.val(target)
    }
    if alreadyUpdated {
        if optionTraceTraversal { t.tracef("Program.execute: '%v' already updated (%v)", t.def.target.value, t.targets) }
        if options.verbose { diag.infoAt(pos, "'%v' already updated", t.def.target.value) }
        if false { diag.warnAt(pos, "'%v' already updated", t.def.target.value).debug(true, 1) }
        if false { return }
    }

    if t.print && t.entry.class == UseRuleEntry { t.print = false }
    if t.print && prog.configure { t.print = false }
    cd.stack[0].silent = !t.print
    return t.exec(prog)
}

func (t *traversal) exec(prog *Program) (result Value) {
    if optionEnableBenchmarks { defer bench(mark("traversal.exec")) }
    if optionEnableBenchspots { defer bench(spot("traversal.exec")) }
    if optionTraceExec {
        var d = t.depth()
        var t = t.def.target.value
        var s = fmt.Sprintf("%s: %v (%p, exec.depth=%d)", typeof(t), t, t, d)
        defer un(trace(t_exec, s))
    }

    t.visited[t.def.target.value] += 1
    if false { if t.visited[t.def.target.value] > 1 {
        if optionTraceExec { t_exec.trace(fmt.Sprintf("visited: %v", t.def.target.value)) }
        return
    }}

    var pos = prog.position

    // Update normal prerequisites
    if t.normalPrerequisites(pos); t.hasBreakers() { return }
    if n := diag.checkErrors(true); n > 0 {
        diag.warnAt(pos, "%d errors while traversing prerequisites for %v", n, t.def.target.value).debug(1)
        t.traceCallStack(pos, "call stack for %v:", t.def.target.value)
        t._break(pos, breakFail).message = fmt.Sprintf("traverse prerequisites failed (%d errors)", n)
        return
    }

    // Update order-only prerequisites
    if t.orderOnlyPrerequisites(pos); t.hasBreakers() { return }
    if n := diag.checkErrors(true); n > 0 {
        diag.warnAt(pos, "%d errors while traversing prerequisites for %v", n, t.def.target.value).debug(1)
        t.traceCallStack(pos, "call stack for %v:", t.def.target.value)
        t._break(pos, breakFail).message = fmt.Sprintf("traverse prerequisites failed (%d errors)", n)
        return
    }

    // Update grapped files
    /* modifierGrepFiles already did the traverse()
    if t.greppedFiles(pos);           t.hasBreakers() { return }
    if n := diag.checkErrors(true); n > 0 {
        diag.warnAt(pos, "%d errors while traversing prerequisites for %v", n, t.def.target.value).
            debug(1)
        return
    }
    */

    var value = t.def.buffer.value
    if len(t.interpreted) == 0 && len(prog.recipes) > 0 && (isNil(value) || isNone(value)) {
        // Using the default statements interpreter.
        if i, ok := dialects["eval"]; ok && i != nil {
            if err := prog.interpret(pos, t, i, nil); err != nil {
                diag.errorAt(pos, "%v", err).debug(1)
            }
        } else {
            diag.errorAt(pos, "no default dialect").debug(1)
        }
    }

    if optionTraceExec {
        t_exec.trace(t.def.stem)
        t_exec.trace(t.def.target)
        t_exec.trace(t.def.depend0)
        t_exec.trace(t.def.depends)
        t_exec.trace(t.def.ordered)
        t_exec.trace(t.def.grepped)
        t_exec.trace(t.def.updated)
        t_exec.trace(t.def.buffer)
    }
    return
}

func (t *traversal) prerequisites(pos Position, prerequisites []Value) {
    // IMPORTANT: don't expand the args here. The prerequisites like
    // '$(or &@,...)' have to be expanded when it's used (e.g. compare).
    for _, prerequisite := range prerequisites {
        if prerequisite.traverse(t); t.hasBreakers() {
            var brks = t.breakersNot(breakNext, breakCase, breakDone)
            if len(brks) > 0 && len(t.stems) > 0 && false {
                diag.warnAt(pos, "broken traversal: %v (target = %v, stems = %v)", brks[0].what,
                    t.def.target.value, t.stems).debug(1)
            }
            break
        }
    }
}

func (t *traversal) normalPrerequisites(pos Position) {
    if optionTraceExec        { defer un(trace(t_exec, t.def.depends.name)) }
    if optionEnableBenchmarks { defer bench(mark("traversal.normalPrerequisites")) }
    defer func(t0, tx, ta *Def) {
        if t.target0, t.targetx, t.targets = t0, tx, ta; len(t.updated) > 0 {
            t.def.updated.value = t.updated[0].target // $?
            for _, u := range t.updated[1:] {
                t.def.updated.append(u.target)
            }
        }
    } (t.target0, t.targetx, t.targets)
    t.targets, t.target0, t.targetx = t.def.depends, t.def.depend0, t.def.dependx
    t.prerequisites(pos, t.program.depends)
}

func (t *traversal) orderOnlyPrerequisites(pos Position) {
    if optionTraceExec        { defer un(trace(t_exec, t.def.ordered.name)) }
    if optionEnableBenchmarks { defer bench(mark("traversal.orderOnlyPrerequisites")) }
    defer func(t0, tx, ta *Def) {
        t.target0, t.targetx, t.targets = t0, tx, ta
    } (t.target0, t.targetx, t.targets)
    t.targets, t.target0, t.targetx = t.def.ordered, nil, nil
    t.prerequisites(pos, t.program.ordered)
}

func (t *traversal) greppedFiles(pos Position) {
    if optionTraceExec        { defer un(trace(t_exec, t.def.grepped.name)) }
    if optionEnableBenchmarks { defer bench(mark("traversal.greppedFiles")) }
    defer func(t0, tx, ta *Def) {
        t.target0, t.targetx, t.targets = t0, tx, ta
    } (t.target0, t.targetx, t.targets)
    t.targets, t.target0, t.targetx = t.def.grepped, nil, nil
    t.prerequisites(pos, t.grepped)
}
