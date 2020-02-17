//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        //"extbit.io/smart/scanner"
        //"extbit.io/smart/token"
        "runtime/debug" // debug.PrintStack()
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
        mutex *sync.Mutex // execution mutex
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

        if err = t.wait(prog.position); err != nil { return } else
        if false { debug.PrintStack() }

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
        if v, err = mergeresult(ExpandAll(m.name)); err != nil { return } else
        if name, err = v[0].Strval(); err != nil { return } else {
                v = append(v[1:], m.args...)
        }

        if f, ok := modifiers[name]; ok {
                // Special modifier processing (implicit interpretation)
                if name == "configure" && t.interpreted == nil {
                        // Evaluate for configure modifier
                        if i, ok := dialects["eval"]; ok && i != nil {
                                if err = prog.interpret(m.Position(), t, i, v); err != nil {
                                        return
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
                err = prog.interpret(m.Position(), t, i, v)
        } else {
                err = errorf(m.position, "unknown modifier '%s'", name)
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
                                err = wrap(prog.position, err)
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
                                err = errorf(pos, "`%s` unknown target (via %s)", s, a)
                        }
                default:
                        result = append(result, a)
                }
        }
        return
}

func (prog *Program) args(args []Value) (params []*Def, err error) {
        var argnum int // setup named/number parameters ($1, $2, etc.)
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
                                        def.set(DefArg, t.Value)
                                        params = append(params, def)
                                } else {
                                        err = errorf(prog.position, "`%s` no such named parameter", s)
                                        return
                                }
                        }
                default:
                        if argnum < len(prog.params) {
                                def = prog.params[argnum]
                                def.set(DefArg, a)
                                params = append(params, def)
                        } else {
                                name := strconv.Itoa(argnum+1)
                                if def, err = prog.auto(name, a); err == nil {
                                        params = append(params, def)
                                        def.origin = DefArg
                                }
                        }
                        argnum += 1
                }
                if err != nil {
                        err = wrap(prog.position, err)
                        return
                }
        }
        return
}

const maxRecursion  = 16 //32 //64

func (prog *Program) execute(caller *traversal, entry *RuleEntry, args []Value) (result Value, err error) {
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("Program.execute(%s)", entry.target))) }
        if optionEnableBenchspots { defer bench(spot("Program.execute")) }
        if false {
                // Execution can be nested, a program.mutex.lock may
                // cause dead-lock in such case.
                prog.mutex.Lock()
                defer prog.mutex.Unlock()
        }

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
                err = errorf(pos, "too many recursion (%d) (%v) (from %v)",
                        recursion, entry.target, caller.def.target.value)
                return
        }

        // The program scope must be protected!
        for _, o := range prog.scope.elems {
                if def, okay := o.(*Def); okay {
                        defer func(def *Def, v Value) {
                                def.value = v
                        } (def, def.value)
                }
        }

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
        var ( none = &None{trivial{pos}} ; stem Value = none )
        if t.caller != nil {
                if optionTraceTraversalNestIndent { t.traceLevel = t.caller.traceLevel }
                if t.stems = t.caller.stems; t.stems != nil {
                        stem = &String{trivial{pos}, t.stems[0]}
                }
        }
        if t.def.stem,    err = prog.auto("*", stem); err != nil { return }
        if t.def.target,  err = prog.auto("@", none); err != nil { return }
        if t.def.depend0, err = prog.auto("<", none); err != nil { return }
        if t.def.depends, err = prog.auto("^", none); err != nil { return }
        if t.def.ordered, err = prog.auto("|", none); err != nil { return }
        if t.def.grepped, err = prog.auto("~", none); err != nil { return }
        if t.def.updated, err = prog.auto("?", none); err != nil { return }
        if t.def.buffer,  err = prog.auto("-", none); err != nil { return }
        if t.def.params,  err = prog.args(args); err != nil { return }
        // Flag targets (-foo) turn off printing automatically
        if _, ok := t.entry.target.(*Flag); ok { t.print = false }
        if t.print && t.entry.class == UseRuleEntry { t.print = false }
        if t.print && prog.configure { t.print = false }

        // cd before setting cloctx
        var enterStop *enterec
        if len(cd.stack) > 0 { enterStop = cd.stack[0] }
        if err = enter(prog, prog.project.absPath); err != nil {
                err = wrap(prog.position, err)
                return
        }
        cd.stack[0].silent = !t.print

        // must set cloctx after cd (enter)
        defer setclosure(setclosure(cloctx.unshift(prog.scope)))
        defer func(s string) { // leaving after setting cloctx to meet the FIFO order
                if e := leave(prog, enterStop); e != nil {
                        // NOTE: err could be breakCase, breakDone, etc.
                        if err == nil { err = e } else {
                                fmt.Fprintf(stderr, "%s: leaving: %s\n", t.entry.Position, e)
                        }
                }
                prog.project.changedWD = s
        } (prog.project.changedWD)

        var fileTarget *File

        // Select the right target value before setting parameters,
        // because the target could be overrided by parameters.
        switch a := t.entry.target.(type) {
        case *File:
                t.def.target.setval(a)
                fileTarget = a
        default:
                var name string
                var target = t.entry.target
                if name, err = target.Strval(); err != nil {
                        err = wrap(prog.position, err)
                        return
                }
                if file := prog.project.matchFile(name); file != nil {
                        fileTarget = file
                        target = file
                }
                t.def.target.setval(target)
        }
        if fileTarget != nil && fileTarget.info != nil && fileTarget.updated {
                if optionVerbose {
                        fmt.Fprintf(stderr, "smart: Already updated %v\n", fileTarget)
                }
                return
        }
        defer func() {
                if err != nil { return }

                // Set fileTarget.updated in stamp() (from exec.go).
                if fileTarget != nil && fileTarget.info != nil && !fileTarget.updated {
                        fileTarget.updated = true
                }

                result, err = t.def.buffer.Call(prog.position)
                if err != nil { err = wrap(prog.position, err) }
        } ()
        return t.exec(prog)
}

func (t *traversal) exec(prog *Program) (result Value, err error) {
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
        if err = t.traverseNormalPrerequisites(pos); err != nil { return }

        // Update order-only prerequisites
        if err = t.traverseOrderOnlyPrerequisites(pos); err != nil { return }

        // Update grapped files
        if err = t.traverseGreppedFiles(pos); err != nil { return }

        if len(t.interpreted) == 0 {
                // Using the default statements interpreter.
                if i, ok := dialects["eval"]; ok && i != nil {
                        if err = prog.interpret(pos, t, i, nil); err != nil { err = wrap(pos, err) }
                } else {
                        err = errorf(pos, "no default dialect")
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

func (t *traversal) traverseNormalPrerequisites(pos Position) (err error) {
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

        var depends []Value
        if depends, err = t.program.prerequisites(t, t.program.depends); err != nil {
                err = wrap(pos, err)
        } else if err = t.dispatch(depends); err != nil {
                err = wrap(pos, t.wait(pos), err)
        } else if err = t.wait(pos); err != nil {
                // ...
        }
        return
}

func (t *traversal) traverseOrderOnlyPrerequisites(pos Position) (err error) {
        if optionTraceExec { defer un(trace(t_exec, t.def.ordered.name)) }
        if optionEnableBenchmarks { defer bench(mark("traversal.traverseOrderOnlyPrerequisites")) }

        t.target0 = nil
        t.targets = t.def.ordered
        defer func() {
                t.target0, t.targets = nil, nil
        } ()

        var ordered []Value
        if ordered, err = t.program.prerequisites(t, t.program.ordered); err != nil {
                err = wrap(pos, err)
        } else if err = t.dispatch(ordered); err != nil {
                err = wrap(pos, t.wait(pos), err)
        } else if err = t.wait(pos); err != nil {
                // ...
        }
        return
}

func (t *traversal) traverseGreppedFiles(pos Position) (err error) {
        if optionTraceExec { defer un(trace(t_exec, t.def.grepped.name)) }
        if optionEnableBenchmarks { defer bench(mark("traversal.traverseGreppedFiles")) }

        t.target0 = nil
        t.targets = t.def.grepped
        defer func() {
                t.target0, t.targets = nil, nil
        } ()
        
        var grepped []Value
        if grepped, err = t.program.prerequisites(t, t.grepped); err != nil {
                err = wrap(pos, err)
        } else if err = t.dispatch(grepped); err != nil {
                err = wrap(pos, t.wait(pos), err)
        } else if err = t.wait(pos); err != nil {
                // ...
        }
        return
}
