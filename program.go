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

type dependPatternUnfit struct {
}

func (*dependPatternUnfit) Error() string { return "pattern unfit" }

type Program struct {
    project *Project
    scope   *Scope
    params  []*Def
    depends []Value // normal
    ordered []Value // order-only
    recipes []Value
    defaultVal Value
    position Position
    language string
    changedWD string
    configure bool
}

func (prog *Program) Position() Position { return prog.position }
func (prog *Program) Project() *Project { return prog.project }
func (prog *Program) Scope() *Scope { return prog.scope }

func (prog *Program) setDefaultValue(val Value) { prog.defaultVal = val }

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
        diag.errorAt(pos, "evaluate failed: %v", err).
            debug(optionDebugErrors, 1)
        return
    } else if !isNil(value) {
        if err = t.def.buffer.val(value); err != nil {
            diag.errorAt(pos, "set modify buffer value failed: %v", err).
                debug(optionDebugErrors, 1)
            return
        }
    }

    if _, _, err = t.updateRecipesHash(); err != nil {
        diag.errorAt(pos, "update recipes hash failed: %v", err).
            debug(optionDebugErrors, 1)
    } else {
        t.interpreted = append(t.interpreted, i)
    }
    return
}

func (prog *Program) getModifies(name string) (ms []*modifier) {
    for _, d := range prog.depends {
        var g, ok = d.(*modifiergroup)
        if !ok { continue }
        for _, m := range g.modifiers {
            if s, e := m.name.Strval(); e != nil {
                diag.errorOf(m.name, "get modifier name '%v' failed: %v", m.name, e)
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
    var name string
    var v []Value
    if v, err = mergeresult(ExpandAll(m.name)); err != nil {
        diag.errorOf(m.name, "expand modifier name '%v' failed: %v", m.name, err).
            debug(optionDebugErrors,1)
        return
    } else if name, err = v[0].Strval(); err != nil {
        diag.errorOf(v[0], "strval '%v' failed: %v", v[0], err).
            debug(optionDebugErrors,1)
        return
    } else {
        v = append(v[1:], m.args...)
    }

    var isConfigure = name == "configure"
    if f, ok := modifiers[name]; ok {
        var value Value
        // Special modifier processing (implicit interpretation) before (configure)
        if isConfigure && len(t.interpreted) == 0 {
            // Evaluate for configure modifier
            if i, ok := dialects["eval"]; ok && i != nil {
                if err = prog.interpret(m.Position(), t, i, v); err != nil {
                    diag.errorAt(m.Position(), "interpret failed: %v", err).
                        debug(optionDebugErrors,1)
                    return
                }
            }
        }
        if value, err = f(m.position, t, v...); err == nil && value != nil {
            if value != t.def.buffer && value != t.def.buffer.value {
                if err = t.def.buffer.val(value); err != nil {
                    diag.errorAt(m.position, "setting modifier buffer value failed: %v", err).
                        debug(optionDebugErrors, 1)
                }
            }
        }
    } else if i, _ := dialects[name]; i != nil {
        if err = prog.interpret(m.Position(), t, i, v); err != nil {
            diag.errorAt(m.Position(), "interpret '%s' failed: %v", name, err).
                debug(optionDebugErrors,1)
        }
    } else {
        diag.errorAt(m.position, "unknown modifier '%s'", name).
            debug(optionDebugErrors,1)
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
                diag.errorOf(t.Key, "%v", err)
                return
            } else if o := prog.scope.Lookup(s); isNil(o) {
                rest = append(rest, a)
            } else if def, ok = o.(*Def); !ok {
                diag.errorOf(o, "object is not a Def: %v", o)
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
                diag.errorOf(a, "arg: %v", err)
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

    var isConfigureExecution bool = prog.configure
    if !isConfigureExecution && caller != nil {
        isConfigureExecution =  caller.isConfigureExecution
    }
    /*if n := diag.checkErrors(true); n > 0 && !isConfigureExecution {
        diag.errorAt(prog.position, "%v: %d errors, discard execution", entry, n)
        return
    }*/
    defer func() {
        if n := diag.checkErrors(true); n > 0 && !isConfigureExecution {
            // create a new error point for next checking
            var pos = prog.position
            if !pos.IsValid() { pos = entry.Position() }
            if !pos.IsValid() { pos = entry.position }
            if n == 1 {
                diag.warnAt(pos, "execution yields an error for %v", entry).
                    debug(optionDebugErrors, 1)
            } else {
                diag.warnAt(pos, "execution yields %d errors for %v", n, entry).
                    debug(optionDebugErrors, 1)
            }
            if caller != nil {
                brk := caller._break(prog.position, breakErro)
                brk.error = fmt.Errorf("%v: got %d errors", entry, n)
                caller.breakers = append(caller.breakers, brk)
            }
        }
    } ()

    var (
        pos = prog.position
        recursion int
    )
    for c := caller; c != nil; c = c.caller {
        if c.program == prog { recursion += 1 }
    }
    if recursion >= maxRecursion {
        diag.errorAt(pos, "exceeds max recursion: %v", entry.target).
            debug(optionDebugErrors,1)
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
            debug(optionDebugErrors, 1)
        return
    }

    // The program scope must be protected!
    for _, o := range prog.scope.elems { if d, okay := o.(*Def); okay {
        defer func(d *Def, v Value) { d.value = v } (d, d.value)
    }}

    var t = &traversal{
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
        isConfigureExecution: isConfigureExecution,
    }
    defer func() {
        // Pass breakers to the caller for handling breakNext, breakCase, breakDone, etc.
        if caller != nil && t.hasBreakers() {
            caller.breakers = append(caller.breakers, t.breakers...)
        }
    } ()

    var ( none = MakeNone(pos) ; stem Value = none; f func() ; err error )
    if t.caller != nil {
        if optionTraceTraversalNestIndent { t.traceLevel = t.caller.traceLevel }
        if t.stems = t.caller.stems; t.stems != nil { stem = MakeString(pos, t.stems[0]) }
    }
    if t.def.stem,    err = prog.auto("*", stem); err != nil { diag.errorAt(pos, "%v", err); return }
    if t.def.target,  err = prog.auto("@", none); err != nil { diag.errorAt(pos, "%v", err); return }
    if t.def.depend0, err = prog.auto("<", none); err != nil { diag.errorAt(pos, "%v", err); return }
    if t.def.depends, err = prog.auto("^", none); err != nil { diag.errorAt(pos, "%v", err); return }
    if t.def.ordered, err = prog.auto("|", none); err != nil { diag.errorAt(pos, "%v", err); return }
    if t.def.grepped, err = prog.auto("~", none); err != nil { diag.errorAt(pos, "%v", err); return }
    if t.def.updated, err = prog.auto("?", none); err != nil { diag.errorAt(pos, "%v", err); return }
    if t.def.buffer,  err = prog.auto("-", none); err != nil { diag.errorAt(pos, "%v", err); return }
    if t.def.params,f,err = prog.args(args)     ; err != nil { diag.errorAt(pos, "%v", err); return } else {
        defer f()
    }

    // Note: must enter work directory (cd) before setting cloctx
    var alreadyUpdated bool
    var enterBack *enterec
    if len(cd.stack) > 0 { enterBack = cd.stack[0] }
    if err = enter(prog, prog.project.absPath); err != nil {
        diag.errorAt(pos, "%v", err)
        return
    }
    defer func(scc closurecontext, swd string) {
        setclosure(scc) // restore closure context

        if e := leave(prog, enterBack); e != nil {
            // NOTE: err could be breakCase, breakDone, etc.
            if err == nil { err = e } else {
                fmt.Fprintf(stderr, "%s: leaving: %s\n", t.entry.Position, e)
            }
        }
        prog.project.changedWD = swd

        if err != nil { return }

        var target = t.def.target.value
        if file, okay := target.(*File); okay && file.info != nil && !file.updated {
            file.updated = true
        }

        if !isNil(t.def.buffer.value) && !isNone(t.def.buffer.value) {
            result = t.def.buffer.Call(prog.position)
        } else if !isNil(prog.defaultVal) {
            result = prog.defaultVal
        }
        if caller != nil && !isNil(result) {
            caller.program.setDefaultValue(result)
        }
        prog.setDefaultValue(nil)

        if !(isNil(target) || isNone(target)) && t.caller != nil {
            t.caller.addTarget(target)
        }
    } (setclosure(cloctx.unshift(prog.scope)), prog.project.changedWD)

    if t.project.name == "-" { optionTraceTraversal = true }

    // Select the right target value before setting parameters,
    // because the target could be overrided by parameters.
    switch a := t.entry.target.(type) {
    case *Flag:
        t.def.target.val(a)
        // Flag target (-foo) turns off printing automatically
        t.print = false
    case *File:
        alreadyUpdated = a.info != nil && a.updated
        t.def.target.val(a)
    default:
        var name string
        var target = t.entry.target
        if name, err = target.Strval(); err != nil {
            diag.errorAt(pos, "strval '%v' failed: %v", target, err).
                debug(optionDebugErrors, 1)
            return
        }
        if file := prog.project.FindFile(name); file != nil {
            alreadyUpdated = file.info != nil && file.updated
            target = file
        }
        t.def.target.val(target)
    }
    if alreadyUpdated {
        if optionTraceTraversal { t.tracef("Program.execute: '%v' already updated (%v)", t.def.target.value, t.targets) }
        if options.verbose { fmt.Fprintf(stderr, "smart: '%v' already updated\n", t.def.target.value) }
        return
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
    if t.visited[t.def.target.value] > 1 {
        if optionTraceExec { t_exec.trace(fmt.Sprintf("visited: %v", t.def.target.value)) }
        if false { return }
    }

    var pos = prog.position

    // Update normal prerequisites
    if t.traverseNormalPrerequisites(pos);    t.hasBreakers() { return }
    if n := diag.checkErrors(true); n > 0 {
        diag.warnAt(pos, "%d errors while traversing prerequisites for %v", n, t.def.target.value).
            debug(optionDebugErrors, 1)
        return
    }

    // Update order-only prerequisites
    if t.traverseOrderOnlyPrerequisites(pos); t.hasBreakers() { return }
    if n := diag.checkErrors(true); n > 0 {
        diag.warnAt(pos, "%d errors while traversing prerequisites for %v", n, t.def.target.value).
            debug(optionDebugErrors, 1)
        return
    }

    // Update grapped files
    if t.traverseGreppedFiles(pos);           t.hasBreakers() { return }
    if n := diag.checkErrors(true); n > 0 {
        diag.warnAt(pos, "%d errors while traversing prerequisites for %v", n, t.def.target.value).
            debug(optionDebugErrors, 1)
        return
    }

    if len(t.interpreted) == 0 {
        // Using the default statements interpreter.
        if i, ok := dialects["eval"]; ok && i != nil {
            if err := prog.interpret(pos, t, i, nil); err != nil {
                diag.errorAt(pos, "%v", err).
                    debug(optionDebugErrors,1)
            }
        } else {
            diag.errorAt(pos, "no default dialect").
                    debug(optionDebugErrors,1)
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

func (t *traversal) prerequisite(pos Position, prerequisite Value) {
    if _, ok := prerequisite.(*Path); !ok && prerequisite.patterned() {
        var pos = prerequisite.Position()
        var ( s string ; rest []string; okay bool )
        if s, rest = prerequisite.stencil(t.stems); s == "" {
            diag.errorAt(pos, "empty prerequisite stencil: %v %v", prerequisite, t.stems).
                debug(optionDebugErrors, 1)
            return
        } else if len(rest) > 0 {
            diag.errorAt(pos, "partial prerequisite stencil: %v, %v, %v, %v", prerequisite, s, rest, t.stems).
                debug(optionDebugErrors, 1)
            panic(s)
        }

        if file := t.project.FindFile(s); file != nil {
            file.position = pos
            okay = t.file(file)
        } else {
            okay = t.target(pos, s)
        }

        if !okay && t.stems != nil {
            b := t._break(pos, breakNext)
            b.scope = breakTrave
            b.value = prerequisite
        }

        if !okay && false {
            diag.warnAt(pos, "missing file %v required by %v", s, t.def.target.value).
                debug(optionDebugErrors, 1)
        }
    } else {
        if false && len(t.stems) > 0 {
            diag.warnAt(pos, "%v -> %v", t.def.target.value, prerequisite).
                debug(optionDebugErrors, 1)
        }
        t.dispatch(prerequisite)
    }
}

func (t *traversal) prerequisites(pos Position, prerequisites []Value) {
    // IMPORTANT: don't expand the args here. The prerequisites like
    // '$(or &@,...)' have to be expanded when it's used (e.g. compare).
    for _, prerequisite := range prerequisites {
        if t.prerequisite(pos, prerequisite); t.hasBreakers() {
            var brks = t.breakersNot(breakNext, breakCase, breakDone)
            if len(brks) > 0 && t.stems != nil && false {
                diag.warnAt(pos, "broken traversal: %v (target = %v, stems = %v)", brks[0].what,
                    t.def.target.value, t.stems).debug(optionDebugErrors)
            }
            break
        }
    }
}

func (t *traversal) traverseNormalPrerequisites(pos Position) {
    if optionTraceExec        { defer un(trace(t_exec, t.def.depends.name)) }
    if optionEnableBenchmarks { defer bench(mark("traversal.traverseNormalPrerequisites")) }
    defer func(t0, ta *Def) {
        if t.target0, t.targets = t0, ta; len(t.updated) > 0 {
            t.def.updated.value = t.updated[0].target // $?
            for _, u := range t.updated[1:] {
                t.def.updated.append(u.target)
            }
        }
    } (t.target0, t.targets)
    t.target0 = t.def.depend0
    t.targets = t.def.depends
    t.prerequisites(pos, t.program.depends)
}

func (t *traversal) traverseOrderOnlyPrerequisites(pos Position) {
    if optionTraceExec        { defer un(trace(t_exec, t.def.ordered.name)) }
    if optionEnableBenchmarks { defer bench(mark("traversal.traverseOrderOnlyPrerequisites")) }
    defer func(t0, ta *Def) {
        t.target0, t.targets = t0, ta
    } (t.target0, t.targets)
    t.target0 = nil
    t.targets = t.def.ordered
    t.prerequisites(pos, t.program.ordered)
}

func (t *traversal) traverseGreppedFiles(pos Position) {
    if optionTraceExec        { defer un(trace(t_exec, t.def.grepped.name)) }
    if optionEnableBenchmarks { defer bench(mark("traversal.traverseGreppedFiles")) }
    defer func(t0, ta *Def) {
        t.target0, t.targets = t0, ta
    } (t.target0, t.targets)
    t.target0 = nil
    t.targets = t.def.grepped
    t.prerequisites(pos, t.grepped)
}
