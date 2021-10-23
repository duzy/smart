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

type programContext struct {
    autoContext
}

func (pc *programContext) String() string { return fmt.Sprintf("program{%s}", pc.autoContext.String()) }

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
    if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("Program.interpret(%s)", typeof(i)))) }

    if pos := ctx.Position(); !pos.IsValid() && prog.position.IsValid() {
        ctx = positional(ctx, prog.position)
    }

    // Wait for prerequisites before interpretion
    wait(ctx)

    var value Value
    if value, err = i.Evaluate(ctx, params...); err != nil {
        ctx.error("evaluation failed: %v", err).debug(6)
        return
    } else if isNil(value) {
        // disgard nil value
    } else if prev, ok := ctx.autoSet("-", value); !ok {
        ctx.error("set buffer value failed: %v -> %v", prev, value)
        ctx.error("set buffer value failed: %v", ctx).debug(1)
        return
    }

    if _, _, err = updateRecipesHash(ctx); err != nil {
        ctx.error("update recipes hash failed: %v", err).debug(1)
    } else {
        var t = ctx.traversal()
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
                ctx.error("get modifier name '%v' failed: %v", m.name, e).of(m.name).debug(1)
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
        ctx.error("expand modifier name '%v' failed: %v", m.name, err).of(m.name).debug(1)
        return
    } else if n := len(args); n == 0 {
        ctx.error("modifier name '%v' is empty", m.name).of(m.name).debug(1)
        return
    } else if name, err = args[0].Strval(ctx); err != nil {
        ctx.error("strval '%v' failed: %v", args[0], err).of(args[0]).debug(1)
        return
    } else {
        args = append(args[1:], m.args...)
    }

    var t = ctx.traversal()
    if f, ok := modifiers[name]; ok {
        var value Value //, _ = ctx.autoGet("-")
        // Special modifier processing (implicit interpretation) before (configure)
        if len(t.interpreted) == 0 && len(prog.recipes) > 0 && name == "configure" /*&& (isNil(value) || isNone(value))*/ {
            // Evaluate for configure modifier
            if i, ok := dialects["eval"]; ok && i != nil {
                if err = prog.interpret(ctx, i, args); err != nil {
                    ctx.error("interpret failed: %v", err).at(m.position).debug(1)
                    return
                }
                if false && t.entry.String() == "HAVE_TERMINFO" {
                    var v, _ = ctx.autoGet("-")
                    ctx.info("eval %v -> %T %v", t.entry, v, v).debug(6)
                }
            }
        }
        if value, brks = f(positional(ctx, m.position), args...); brks.has() {
            if tb := brks.not(breakCase, breakNext, breakDone); tb.has() {
                for _, brk := range brks {
                    switch brk.what {
                    case breakFail: ctx.error("broken modifier %v with failure: %v", name, brk.message).at(brk.pos)
                    case breakErro: ctx.error("broken modifier %v with error: %v", name, brk.error).at(brk.pos)
                    default: ctx.error("broken modifier %v (%v)", name, brk.what).at(brk.pos)
                    }
                }
                ctx.error("borken modifier %s %v", name, args).at(m.position).debug(6)
            }
        } else if hyphen, found := ctx.autoGet("-"); !found || isNil(value) || value == hyphen {
            // does nothing
        } else if ctx.autoSet("-", value); false {
            ctx.error("setting buffer value failed: %v", value).at(m.position).debug(1)
        }
    } else if i, _ := dialects[name]; i != nil {
        if err = prog.interpret(ctx, i, args); err != nil {
            ctx.error("interpret '%s' failed: %v", name, err).at(m.position).debug(1)
        }
    } else {
        ctx.error("unknown modifier '%s'", name).at(m.position).debug(1)
    }
    return
}

const maxRecursion  = 16 //32 //64

func (prog *Program) execute(cc Context, entry Entry, args []Value) (result Value, brks breakers) {
    if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("Program.execute(%s)", entry.Target()))) }
    if optionEnableBenchspots { defer bench(spot("Program.execute")) }

    var pos = cc.Position()
    if !pos.IsValid() { pos = entry.Position() }
    if cc != nil && cc.checkErrors(true) > 0 {
        var errs = cc.totalErrors()
        var s string; if errs > 1 { s = "(s)" }
        cc.warn(`cancel execution for "%v" due to %d error%s`, entry, errs, s).debug(16)
        if options.failOnErrors { fail(pos, "fail by %d error%s", errs, s) }
        return
    }

    assert(prog.project == prog.scope.project, "mismatched scope/project")

    var (
        _t_ = traverseContext{
            Context: cc,
            configuration: prog.configure || (cc != nil && cc.traversal().configuration),
            execRec: make(map[Value]int),
            start: time.Now(),
            program: prog,
            entry: entry,
            print: true,
        }
        pc = programContext{autoContext{ Context:&_t_, defs:make(autoDefMap) }}
        ctx Context = &pc
        err error
    )
    defer func() {
        if ctx.checkErrors(true) > 0  {
            var errs = ctx.totalErrors()
            if !_t_.configuration && cc != nil {
                if errs == 1 {
                    err = fmt.Errorf("execution yields an error for %v", entry)
                } else {
                    err = fmt.Errorf("execution yields %d errors for %v", errs, entry)
                }
                brks.add(pos, breakErro).error = err
            }
            ctx.warn("execution got %d errors", errs).debug(1)
            if options.failOnErrors { fail(ctx.Position(), "fail by %d errors", errs) }
        }
    } ()
    if cc != nil {
        var recursion int
        for c := cc.traversal(); c != nil; c = c.caller() { if c.program == prog { recursion += 1 }}
        if recursion >= maxRecursion {
            ctx.error("exceeds max recursion: %v", entry.Target()).debug(1)
            for c := cc.traversal(); c != nil; c = c.caller() {
                var n int
                for next := c.caller(); next != nil; next = next.caller() {
                    if next.program == c.program { n += 1; c = next } else { break }
                }
                var ct, _ = c.autoGet("@")
                if n > 0 {
                    ctx.error("%v (repeats %d times)", ct, n).at(c.program.position)
                } else {
                    ctx.error("%v", ct).at(c.program.position)
                }
            }
            ctx.error("too many recursion (%d) (%v)", recursion, entry.Target()).debug(1)
            return
        }
        if options.traceTraversalNestIndent { _t_.traceLevel = cc.traversal().traceLevel }
        if _t_.stems = cc.traversal().stems; _t_.stems != nil { ctx.autoSet("*", MakeString(pos, _t_.stems[0])) }
    }
    if _t_.params, err = ctx.autoArgs(_t_.program.params, args); err != nil {
        ctx.error("auto args failed: %v", err).debug(1)
        return
    }

    // Note: must enter work directory (cd) before setting cloctx
    var (
        alreadyUpdated bool
        enterBack *enterec
    )
    if len(cd.stack) > 0 { enterBack = cd.stack[0] }
    if err = enter(ctx, prog.project.absPath); err != nil {
        ctx.error("enter project '%v' failed: %v", prog.project, err).debug(1)
        return
    }
    defer func(swd string) {
        if e := leave(ctx, prog, enterBack); e != nil {
            // NOTE: err could be breakCase, breakDone, etc.
            if err == nil { err = e } else {
                ctx.error("leave project '%v' failed: %v", prog.project, err).debug(1)
            }
        }
        if prog.project.changedWD = swd; err != nil {
            ctx.error("execution failed: %v", err).debug(6)
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

        if cc != nil && cc.Program() != nil && !isNil(result) {
            cc.Program().defaultVal = result
        }
    } (prog.project.changedWD)

    // Select the right target value before setting parameters,
    // because the target could be overrided by parameters.
    switch a := _t_.entry.Target().(type) {
    case *Flag:
        ctx.autoSet("@", a)
        ctx.traversal().print = false // Flag target (-foo) turns off printing automatically
    case *File:
        ctx.autoSet("@", a)
        //alreadyUpdated = a.info != nil && a.updated
    default:
        var name string
        if name, err = a.Strval(ctx); err != nil {
            ctx.error("strval '%v' failed: %v", a, err).debug(1)
            return
        } else if file := prog.project.FindFile(ctx, name); file != nil {
            //alreadyUpdated = file.info != nil && file.updated
            a = file
        }
        ctx.autoSet("@", a)
    }
    if alreadyUpdated {
        var target, _ = ctx.autoGet("@")
        if options.traceTraversal { _t_.tracef("Program.execute: '%v' already updated (%v)", target, _t_.targets) }
        if options.verbose { ctx.info("'%v' already updated", target) }
        if false { ctx.warn("'%v' already updated", target).debug(1) }
        if false { return }
    }

    if _t_.print && _t_.entry.Class() == UseRuleEntry { _t_.print = false }
    if _t_.print && prog.configure { _t_.print = false }
    cd.stack[0].silent = !_t_.print
    return prog.exec(ctx)
}

func (prog *Program) exec(ctx Context) (result Value, brks breakers) {
    if optionEnableBenchmarks { defer bench(mark("traversal.exec")) }
    if optionEnableBenchspots { defer bench(spot("traversal.exec")) }

    var t = ctx.traversal()
    var target, _ = ctx.autoGet("@")
    if options.traceExec {
        var d = t.depth()
        var s = fmt.Sprintf("%s: %v (%p, exec.depth=%d)", typeof(target), target, target, d)
        defer un(trace(t_exec, s))
    }

    t.execRec[target] += 1
    if false { if t.execRec[target] > 1 {
        if options.traceExec { t_exec.trace(fmt.Sprintf("exec: %v", target)) }
        return
    }}

    var pos = prog.position

    // Update normal prerequisites
    if brks = traverseNormal(ctx); brks.has() {
        return
    } else if errs := ctx.checkErrors(true); errs > 0 {
        brks.add(pos, breakFail).message = fmt.Sprintf("traverse prerequisites failed (%d errors)", ctx.totalErrors())
        ctx.warn("%d errors while traversing prerequisites for %v", errs, target).debug(8)
        callstack(ctx, -1, "call stack for %v:", target)
        if options.failOnErrors { fail(pos, "fail by %d errors", ctx.totalErrors()) }
        return
    }

    // Update order-only prerequisites
    if brks = traverseOrderOnly(ctx); brks.has() {
        return
    } else if errs := ctx.checkErrors(true); errs > 0 {
        brks.add(pos, breakFail).message = fmt.Sprintf("traverse prerequisites failed (%d errors)", errs)
        ctx.warn("%d errors while traversing prerequisites for %v", errs, target).debug(8)
        callstack(ctx, -1, "call stack for %v:", target)
        if options.failOnErrors { fail(ctx.Position(), "fail by %d errors", ctx.totalErrors()) }
        return
    }

    var value, _ = ctx.autoGet("-")
    if len(t.interpreted) == 0 && len(prog.recipes) > 0 && (isNil(value) || isNone(value)) {
        // Using the default statements interpreter.
        if i, ok := dialects["eval"]; ok && i != nil {
            if err := prog.interpret(ctx, i, nil); err != nil {
                ctx.error("%v", err).debug(1)
            }
        } else {
            ctx.error("no default dialect").debug(1)
        }
    }
    return
}

func traversePrerequisites(ctx Context, prerequisites []Value) (brks breakers) {
    // IMPORTANT: don't expand the args here. The prerequisites like
    // '$(or &@,...)' have to be expanded when it's used (e.g. compare).
    var t = ctx.traversal()
    var target, _ = ctx.autoGet("@")
    if true {
        //var entryName = t.entry.Name()
        //var infos = t.configuration //entryName == "llvm-ar"
        for _, prerequisite := range prerequisites {
            /*var prestr = prerequisite.String()
            if infos || (entryName == "program" && prestr == "$(requirement)") {
                if d, _ := prerequisite.(*delegate); d != nil {
                    t.info("%v: %T %v, %v", entryName, d.x, d.x, d.x.(*Def).origin).at(ctx.Position())
                }
                var val, _ = prerequisite.expand(ctx, expandPlainValue)
                t.info("%v: %T %v -> %T %v", entryName, prerequisite, prerequisite, val, val).at(ctx.Position())
                t.info("%v: args = %v", entryName, t.args).at(ctx.Position())
                t.info("%v: %v", entryName, ctx).at(ctx.Position())
                t.info("%v: %v", entryName, t).at(ctx.Position()).debug(1)
            }*/
            if brks = prerequisite.traverse(ctx); brks.has() {
                var tb = brks.not(breakNext, breakCase, breakDone)
                if len(tb) > 0 && len(t.stems) > 0 && false {
                    t.warn("broken traversal: %v (target = %v, stems = %v)", tb[0].what, target, t.stems).debug(1)
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
                defer checkPanicsErrors(ctx)
                defer wg.Done() // minus 1
                if tb := prerequisite.traverse(ctx); tb.has() {
                    mu.Lock(); defer mu.Unlock()
                    brks = append(brks, tb...)
                }
            } ()
        }
        if wg.Wait(); brks.has() {
            var tb = brks.not(breakNext, breakCase, breakDone)
            if len(tb) > 0 && len(t.stems) > 0 && false {
                t.warn("broken traversal: %v (target = %v, stems = %v)", tb[0].what, target, t.stems).debug(1)
            }
        }
    }
    return
}

func traverseNormal(ctx Context) (brks breakers) {
    if options.traceExec      { defer un(trace(t_exec, "^")) }
    if optionEnableBenchmarks { defer bench(mark("traversal.normal")) }
    var t = ctx.traversal()
    /*defer func(t0, tx, ta *Def) {
        if t.target0, t.targetx, t.targets = t0, tx, ta; len(t.updated) > 0 {
            t.def.updated.value = t.updated[0].target // $?
            for _, u := range t.updated[1:] {
                t.def.updated.append(ctx, u.target)
            }
        }
    } (t.target0, t.targetx, t.targets)*/
    t.target0, t.targetN, t.targetX = "<", ">", "^"
    return traversePrerequisites(ctx, t.program.depends)
    /*if n := len(t.targets); n > 0 {
        t.autoSet("<", t.targets[0])
        t.autoSet(">", t.targets[n-1])
        if n == 1 {
            t.autoSet("^", t.targets[0])
        } else {
            t.autoSet("^", MakeList(t.targets[0].Position(), t.targets))
        }
    }*/
}

func traverseOrderOnly(ctx Context) (brks breakers) {
    if options.traceExec        { defer un(trace(t_exec, "|")) }
    if optionEnableBenchmarks { defer bench(mark("traversal.orderonly")) }
    var t = ctx.traversal()
    /*defer func(t0, tx, ta *Def) {
        t.target0, t.targetx, t.targets = t0, tx, ta
    } (t.target0, t.targetx, t.targets)*/
    t.target0, t.targetN, t.targetX = "", "", "|"
    return traversePrerequisites(ctx, t.program.ordered)
}

// DEPRECATED
func _traverseGrepped(ctx Context) (brks breakers) {
    if options.traceExec        { defer un(trace(t_exec, "~")) }
    if optionEnableBenchmarks { defer bench(mark("traversal.grepped")) }
    var t = ctx.traversal()
    /*defer func(t0, tx, ta *Def) {
        t.target0, t.targetx, t.targets = t0, tx, ta
    } (t.target0, t.targetx, t.targets)*/
    t.target0, t.targetN, t.targetX = "", "", "~"
    return traversePrerequisites(ctx, t.grepped)
}
