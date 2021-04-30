//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        //"runtime/debug" // debug.PrintStack()
        "strconv"
        //"strings"
        "sync"
        //"time"
        "fmt"
        //"os"
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
        position Position
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
                        auto.setval(value)
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
        if optionEnableBenchmarks {
                s := fmt.Sprintf("Program.interpret(%s)", typeof(i))
                defer bench(mark(s))
        }

        for _, e := range t.breakers {
                if e.what != breakCase { return }
        }

        t.wait(prog.position) // wait for prerequisites

        var value Value
        if value, err = i.Evaluate(pos, t, params...); err == nil {
                if value != nil { t.def.buffer.setval(value) }
                _, _, err = t.updateRecipesHash()
        }

        t.interpreted = append(t.interpreted, i)
        return
}

func (prog *Program) modify(t *traversal, m *modifier) (err error) {
        // TODO: using rules in a different project to implement modifiers, e.g.
        //       [ foo.check-preprequisites ]
        //       [ foo.baaaar ]
        var name string
        var v []Value
        if v, err = mergeresult(ExpandAll(m.name)); err != nil { diag.errorOf(m.name, "%v", err); return } else
        if name, err = v[0].Strval(); err != nil { diag.errorOf(v[0], "%v", err); return } else {
                v = append(v[1:], m.args...)
        }

        if f, ok := modifiers[name]; ok {
                // Special modifier processing (implicit interpretation)
                if name == "configure" && t.interpreted == nil {
                        // Evaluate for configure modifier
                        if i, ok := dialects["eval"]; ok && i != nil {
                                if err = prog.interpret(m.Position(), t, i, v); err != nil {
                                        diag.errorAt(m.Position(), "%v", err); return
                                }
                        }
                }

                var value Value
                if value, err = f(m.position, t, v...); err == nil && value != nil {
                        if value != t.def.buffer && value != t.def.buffer.value {
                                err = t.def.buffer.setval(value)
                        }
                }
        } else if i, _ := dialects[name]; i != nil {
                if err = prog.interpret(m.Position(), t, i, v); err != nil {
                        diag.errorAt(m.Position(), "%v", err)
                }
        } else {
                diag.errorAt(m.position, "unknown modifier '%s'", name)
        }
        return
}

func (prog *Program) prerequisites(t *traversal, args []Value) (result []Value, err error) {
        if optionEnableBenchmarks && false { defer bench(mark("Program.prerequisites")) }
        if optionEnableBenchspots { defer bench(spot("Program.prerequisites")) }
        // IMPORTANT: don't expand the args here. The prerequisites like
        // '$(or &@,...)' have to be expanded when it's used (e.g. compare).
        for _, arg := range args {
                switch a := arg.(type) {
                case Pattern:
                        // Double checks for path pattern.
                        if p, ok := arg.(*Path); ok && !p.isPattern() {
                                result = append(result, p)
                                break
                        }

                        var pos = arg.Position()
                        var ( s string ; rest []string )
                        if s, rest, err = a.stencil(t.stems); err != nil {
                                diag.errorAt(prog.position, "%v", err)
                                return
                        }
                        if len(rest) > 0 {
                                fmt.Fprintf(stderr, "%v: unhandled stems: %v, %v, %v, %v\n", pos, arg, s, rest, t.stems)
                                panic(s)
                        }

                        if file := t.project.matchFile(s); file != nil {
                                file.position = pos
                                result = append(result, file)
                                break
                        }

                        if true {
                                result = append(result, a)
                        } else if false {
                                result = append(result, &String{trivial{pos},s})
                        } else {
                                diag.errorAt(pos, "`%s` unknown target (via %s)", s, a)
                        }
                default:
                        result = append(result, a)
                }
        }
        return
}

func (prog *Program) args(args []Value) (params []*Def, restore func(), err error) {
        var argnum int // setup named/number parameters ($1, $2, etc.)
        var values []Value
        for _, d := range prog.params { values = append(values, d.value) }
        for _, a := range args {
                var def *Def
                //<!IMPORTANT: Don't translate Flag, Flag values are valid
                //             regular arguments. Pair values are special.
                switch t := a.(type) {
                case *Pair:
                        var s string
                        if s, err = t.Key.Strval(); err == nil {
                                if o := prog.scope.Lookup(s); o != nil {
                                        def = o.(*Def)
                                        values = append(values, def.value)
                                        params = append(params, def)
                                        def.set(DefArg, t.Value)
                                } else {
                                        diag.errorAt(prog.position, "`%s` no such named parameter", s)
                                        return
                                }
                        }
                default:
                        if argnum < len(prog.params) {
                                def = prog.params[argnum]
                                values = append(values, def.value)
                                params = append(params, def)
                                def.set(DefArg, a)
                        } else {
                                name := strconv.Itoa(argnum+1)
                                if def, err = prog.auto(name, a); err == nil {
                                        values = append(values, def.value)
                                        params = append(params, def)
                                        def.origin = DefArg
                                }
                        }
                        argnum += 1
                }
                if err != nil {
                        diag.errorAt(prog.position, "%v", err)
                        return
                }
        }
        restore = func() { 
                var nlen = len(prog.params)
                for i, d := range prog.params { d.value = values[i] }
                for i, d := range params { d.value = values[nlen+i] }
        }
        return
}

const maxRecursion  = 16 //32 //64

func (prog *Program) execute(caller *traversal, entry *RuleEntry, args []Value) (result Value, brks []*breaker) {
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("Program.execute(%s)", entry.target))) }
        if optionEnableBenchspots { defer bench(spot("Program.execute")) }
        defer func() {
                if diag.checkErrors(true) > 0 {
                        brks = append(brks, &breaker{
                                pos: prog.position, what:breakErro,
                                error: fmt.Errorf("%v: too many errors", entry),
                        })
                        if false { panic(fmt.Errorf("%v: too many errors", entry)) }
                }
        } ()

        var recursion int
        var pos = prog.position
        for c := caller; c != nil; c = c.caller {
                if c.program == prog { recursion += 1 }
        }
        if recursion >= maxRecursion {
                fmt.Fprintf(stderr, "%v: max recursion: %v\n", pos, entry.target)
                for c := caller; c != nil; c = c.caller {
                        fmt.Fprintf(stderr, "    %v: %v\n", c.program.position, c.def.target)
                }
                if false { fmt.Fprintf(stderr, "\n") }
                diag.errorAt(pos, "too many recursion (%d) (%v) (from %v)",
                        recursion, entry.target, caller.def.target.value)
                return
        }

        // The program scope must be protected!
        for _, o := range prog.scope.elems { if d, okay := o.(*Def); okay {
                defer func(d *Def, v Value) { d.value = v } (d, d.value)
        }}

        var t = &traversal{
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
        var ( none = &None{trivial{pos}} ; stem Value = none; f func() ; err error )
        if t.caller != nil {
                if optionTraceTraversalNestIndent { t.traceLevel = t.caller.traceLevel }
                if t.stems = t.caller.stems; t.stems != nil { stem = &String{trivial{pos}, t.stems[0]} }
        }
        if t.def.stem,    err = prog.auto("*", stem); err != nil { diag.errorAt(pos, "%v", err); return }
        if t.def.target,  err = prog.auto("@", none); err != nil { diag.errorAt(pos, "%v", err); return }
        if t.def.depend0, err = prog.auto("<", none); err != nil { diag.errorAt(pos, "%v", err); return }
        if t.def.depends, err = prog.auto("^", none); err != nil { diag.errorAt(pos, "%v", err); return }
        if t.def.ordered, err = prog.auto("|", none); err != nil { diag.errorAt(pos, "%v", err); return }
        if t.def.grepped, err = prog.auto("~", none); err != nil { diag.errorAt(pos, "%v", err); return }
        if t.def.updated, err = prog.auto("?", none); err != nil { diag.errorAt(pos, "%v", err); return }
        if t.def.buffer,  err = prog.auto("-", none); err != nil { diag.errorAt(pos, "%v", err); return }
        if t.def.params,f,err = prog.args(args); err != nil { diag.errorAt(pos, "%v", err); return } else {
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

                result, err = t.def.buffer.Call(prog.position)
                if err != nil { diag.errorAt(prog.position, "%v", err) } else
                if !(isNil(target) || isNone(target)) && t.caller != nil {
                        //if s, _ := target.Strval(); strings.Contains(s, "isl_srcdir.") { t.tracef("%v (%v) (%v)", s, t.target0, t.targets) }
                        t.caller.addNewTarget(target)
                }
        } (setclosure(cloctx.unshift(prog.scope)), prog.project.changedWD)

        if t.project.name == "-" { optionTraceTraversal = true }

        // Select the right target value before setting parameters,
        // because the target could be overrided by parameters.
        switch a := t.entry.target.(type) {
        case *Flag:
                t.def.target.setval(a)
                // Flag target (-foo) turns off printing automatically
                t.print = false
        case *File:
                alreadyUpdated = a.info != nil && a.updated
                t.def.target.setval(a)
        default:
                var name string
                var target = t.entry.target
                if name, err = target.Strval(); err != nil { diag.errorAt(pos, "%v", err);  return }
                if file := prog.project.matchFile(name); file != nil {
                        alreadyUpdated = file.info != nil && file.updated
                        target = file
                }
                t.def.target.setval(target)
        }
        if alreadyUpdated {
                if optionTraceTraversal { t.tracef("Program.execute: '%v' already updated (%v)", t.def.target.value, t.targets) }
                if optionVerbose { fmt.Fprintf(stderr, "smart: '%v' already updated\n", t.def.target.value) }
                return
        }

        if t.print && t.entry.class == UseRuleEntry { t.print = false }
        if t.print && prog.configure { t.print = false }
        cd.stack[0].silent = !t.print
        return t.exec(prog)
}

func (t *traversal) exec(prog *Program) (result Value, breakers []*breaker) {
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
        if breakers = t.traverseNormalPrerequisites(pos); len(breakers) > 0 { return }

        // Update order-only prerequisites
        if breakers = t.traverseOrderOnlyPrerequisites(pos); len(breakers) > 0 { return }

        // Update grapped files
        if breakers = t.traverseGreppedFiles(pos); len(breakers) > 0 { return }

        if len(t.interpreted) == 0 {
                // Using the default statements interpreter.
                if i, ok := dialects["eval"]; ok && i != nil {
                        if err := prog.interpret(pos, t, i, nil); err != nil {
                                diag.errorAt(pos, "%v", err)
                        }
                } else {
                        diag.errorAt(pos, "no default dialect")
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

func (t *traversal) traverseNormalPrerequisites(pos Position) (breakers []*breaker) {
        if optionTraceExec { defer un(trace(t_exec, t.def.depends.name)) }
        if optionEnableBenchmarks { defer bench(mark("traversal.traverseNormalPrerequisites")) }

        t.target0 = t.def.depend0
        t.targets = t.def.depends
        defer func() {
                t.target0, t.targets = nil, nil
                if len(t.updated) > 0 {
                        t.def.updated.value = t.updated[0].target // $?
                        for _, u := range t.updated[1:] {
                                t.def.updated.append(u.target)
                        }
                }
        } ()

        var ( depends []Value; err error )
        if depends, err = t.program.prerequisites(t, t.program.depends); err != nil {
                diag.errorAt(pos, "prerequisites: %v", err)
        }
        if breakers = t.dispatch(depends); len(breakers) > 0 {  }
        t.wait(pos) // wait for prerequisites
        return
}

func (t *traversal) traverseOrderOnlyPrerequisites(pos Position) (breakers []*breaker) {
        if optionTraceExec        { defer un(trace(t_exec, t.def.ordered.name)) }
        if optionEnableBenchmarks { defer bench(mark("traversal.traverseOrderOnlyPrerequisites")) }

        t.target0 = nil
        t.targets = t.def.ordered
        defer func() {
                t.target0, t.targets = nil, nil
        } ()

        var ( ordered []Value; err error )
        if ordered, err = t.program.prerequisites(t, t.program.ordered); err != nil {
                diag.errorAt(pos, "%v", err)
        }
        if breakers = t.dispatch(ordered); len(breakers) > 0 {  }
        t.wait(pos) // wait for prerequisites
        return
}

func (t *traversal) traverseGreppedFiles(pos Position) (breakers []*breaker) {
        if optionTraceExec        { defer un(trace(t_exec, t.def.grepped.name)) }
        if optionEnableBenchmarks { defer bench(mark("traversal.traverseGreppedFiles")) }

        t.target0 = nil
        t.targets = t.def.grepped
        defer func() {
                t.target0, t.targets = nil, nil
        } ()
        
        var ( grepped []Value; err error )
        if grepped, err = t.program.prerequisites(t, t.grepped); err != nil {
                diag.errorAt(pos, "%v", err)
        }
        if breakers = t.dispatch(grepped); len(breakers) > 0 {  }
        t.wait(pos) // wait for prerequisites
        return
}
