//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/scanner"
        "extbit.io/smart/token"
        "runtime/debug" // debug.PrintStack()
        "strconv"
        //"strings"
        "sync"
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
        params  []string
        depends []Value // normal
        ordered []Value // order-only
        recipes []Value
        position Position
        changedWD string
}

func (prog *Program) Position() Position { return prog.position }
func (prog *Program) Project() *Project { return prog.project }
func (prog *Program) Scope() *Scope { return prog.scope }

func (prog *Program) auto(name string, value Value) (auto *Def, err error) {
        var alt Object
        if auto, alt = prog.scope.define(prog.project, name, value); alt != nil {
                var found = false
                if auto, found = alt.(*Def); found {
                        auto.set(DefDefault, value)
                } else {
                        err = fmt.Errorf("`%v` name already taken (%T)", name, alt)
                }
        }
        if enable_assertions {
                assert(auto.value == value, "wrong auto value")
        }
        return
}

func (prog *Program) setUser(proj *Project) (saved *Project) {
        if obj := prog.scope.Lookup(userproj); obj != nil {
                if name, ok := obj.(*ProjectName); ok && name != nil {
                        saved = name.project
                        name.project = proj
                }
        }
        return
}

func (prog *Program) interpret(pc *traversal, i interpreter, params []Value) (err error) {
        if len(pc.breakers) > 0 { return }
        if err = pc.wait(prog.position); err != nil {
                return
        } else if false { debug.PrintStack() }

        var value Value
        if value, err = i.Evaluate(pc, params); err == nil {
                if value != nil { pc.def.modbuff.set(DefDefault, value) }
                _, _, err = pc.updateRecipesHash()
        }

        pc.interpreted = append(pc.interpreted, i)
        return
}

func (prog *Program) modify(pc *traversal, m *modifier) (err error) {
        // TODO: using rules in a different project to implement modifiers, e.g.
        //       [ foo.check-preprequisites ]
        //       [ foo.baaaar ]
        var name string
        var v []Value
        if v, err = mergeresult(ExpandAll(m.name)); err != nil {
                return
        } else if name, err = v[0].Strval(); err != nil {
                return
        } else {
                v = append(v[1:], m.args...)
        }

        if f, ok := modifiers[name]; ok {
                // Special modifier processing (implicit interpretation)
                if name == "configure" && pc.interpreted == nil {
                        // Evaluate for configure modifier
                        if i, ok := dialects["eval"]; ok && i != nil {
                                err = prog.interpret(pc, i, v)
                        }
                }
                if err == nil {
                        var value Value
                        if value, err = f(m.position, pc, v...); err == nil && value != nil {
                                pc.def.modbuff.set(DefDefault, value)
                        }
                }
        } else if i, _ := dialects[name]; i != nil {
                err = prog.interpret(pc, i, v)
        } else {
                err = fmt.Errorf("unknown modifier '%s'", name)
        }
        return
}

func (prog *Program) getModifier(name string) (res *modifier) {
        for _, d := range prog.depends {
                var g, ok = d.(*modifiergroup)
                if !ok { continue }
                for _, m := range g.modifiers {
                        if s, err := m.name.Strval(); err != nil {
                                break
                        } else if name == s {
                                res = m
                        }
                }
        }
        return
}

func (prog *Program) prerequisites(pc *traversal, args []Value) (result []Value, err error) {
        // IMPORTANT: don't expand the args here. The prerequisites like
        // '$(or &@,...)' have to be expanded when it's used (e.g. compare).
        for _, arg := range args {
                switch a := arg.(type) {
                case Pattern:
                        var s string
                        var rest []string
                        if s, rest, err = a.stencil(pc.stems); err != nil {
                                err = wrap(prog.position, err)
                                return
                        }
                        if len(rest) > 0 {
                                panic(fmt.Sprintf("FIXME: unhandled stems: %v (%v, %v) (%v)", arg, s, rest, pc.stems))
                        }
                        if file := pc.project.matchFile(s); file != nil {
                                file.position = arg.Position()
                                result = append(result, file)
                                break
                        }
                        if true {
                                result = append(result, a)
                        } else if false {
                                result = append(result, &String{trivial{prog.position},s})
                        } else {
                                err = scanner.Errorf(token.Position(prog.position), "`%s` unknown target (via %s)", s, a)
                        }
                default:
                        result = append(result, a)
                }
        }
        return
}

func (prog *Program) setParams(args []Value) (err error, restore func()) {
        var params []*Def
        for i, param := range prog.params {
                var def *Def
                if def, err = prog.auto(param, &None{}); err != nil {
                        err = wrap(prog.position, err)
                        return
                }
                prog.scope.replace(strconv.Itoa(i+1), def)
                params = append(params, def)
        }
        var argnum int // setup named/number parameters ($1, $2, etc.)
        for _, a := range args {
                //<!IMPORTANT: Don't translate Flag, Flag values are valid
                //             regular arguments. Don't Pair values are
                //             special.
                switch t := a.(type) {
                case *Pair:
                        var s string
                        if s, err = t.Key.Strval(); err == nil {
                                if o := prog.scope.Lookup(s); o != nil {
                                        o.(*Def).set(DefDefault, t.Value)
                                } else {
                                        err = scanner.Errorf(token.Position(prog.position), "`%s` no such named parameter", s)
                                }
                        }
                default:
                        var def *Def
                        if argnum < len(params) {
                                def = params[argnum]
                                def.set(DefDefault, a)
                        } else {
                                name := strconv.Itoa(argnum+1)
                                if def, err = prog.auto(name, a); err == nil {
                                        params = append(params, def)
                                }
                        }
                        argnum += 1
                }
                if err != nil {
                        err = wrap(prog.position, err)
                        return
                }
        }
        restore = func() {
                for _, param := range params {
                        param.set(DefDefault, &None{})
                }
        }
        return
}

const maxRecursion  = 16 //32 //64

func (prog *Program) execute(caller *traversal, entry *RuleEntry, args []Value) (result Value, err error) {
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
                err = errorf(pos, "too many recursion (%d) (%v) (from %v)", recursion, entry.target, caller.def.target.value)
                return
        }

        var none = &None{trivial{pos}}
        var pc = &traversal{
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
        if pc.caller != nil {
                pc.stems = pc.caller.stems
                if optionTraceTraversalNestIndent {
                        pc.traceLevel = pc.caller.traceLevel
                }
        }

        // Flag targets (-foo) turn off printing
        if _, ok := pc.entry.target.(*Flag); ok { pc.print = false }
        if pc.print && pc.entry.class == UseRuleEntry {
                pc.print = false
        }
        if pc.print && prog.getModifier("configure") != nil { pc.print = false }

        // cd before setting cloctx
        var enterStop *enterec
        if len(cd.stack) > 0 { enterStop = cd.stack[0] }
        if err = enter(prog, prog.project.absPath); err != nil {
                err = wrap(prog.position, err)
                return
        }
        cd.stack[0].silent = !pc.print

        // must set cloctx after cd (enter)
        defer setclosure(setclosure(cloctx.unshift(prog.scope)))
        defer func(s string) { prog.project.changedWD = s } (prog.project.changedWD)
        defer func() { // leaving after setting cloctx to meet the FIFO order
                if e := leave(prog, enterStop); e != nil {
                        // NOTE: err could be breakCase, breakDone, etc.
                        if err == nil { err = e } else {
                                fmt.Fprintf(stderr, "%s: leaving: %s\n", pc.entry.Position, e)
                        }
                }
        } ()

        // set $@, $^, $<, $|, $~, $?, etc
        if pc.def.target,  err = prog.auto("@", none); err != nil { return }
        if pc.def.depend0, err = prog.auto("<", none); err != nil { return }
        if pc.def.depends, err = prog.auto("^", none); err != nil { return }
        if pc.def.ordered, err = prog.auto("|", none); err != nil { return }
        if pc.def.grepped, err = prog.auto("~", none); err != nil { return }
        if pc.def.updated, err = prog.auto("?", none); err != nil { return }
        if pc.def.stem,    err = prog.auto("*", none); err != nil { return }
        if pc.def.modbuff, err = prog.auto("-", none); err != nil { return }
        var set = func(def *Def, val Value) { def.value = val }
        for _, def := range []*Def{
                pc.def.target,  // $@
                pc.def.depends, // $^
                pc.def.depend0, // $<
                pc.def.ordered, // $|
                pc.def.grepped, // $~
                pc.def.updated, // $?
                pc.def.stem,    // $*
                pc.def.modbuff, // $-
        } { defer set(def, def.value) }

        var fileTarget *File

        // Select the right target value before setting parameters,
        // because the target could be overrided by parameters.
        switch t := pc.entry.target.(type) {
        case *File:
                pc.def.target.set(DefDefault, t)
                fileTarget = t
        default:
                var name string
                var target = pc.entry.target
                if name, err = target.Strval(); err != nil {
                        err = wrap(prog.position, err)
                        return
                }
                if file := prog.project./*searchFile*/matchFile(name); file != nil {
                        fileTarget = file
                        target = file
                }
                pc.def.target.set(DefDefault, target)
        }

        if e, clearParams := prog.setParams(args); e != nil {
                err = e; return
        } else {
                defer func() {
                        clearParams()
                        pc.def.params = nil
                } ()
        }

        // Expanding all dependencies after pre-modifiers.
        var depends, ordered []Value
        if depends, err = prog.prerequisites(pc, prog.depends); err != nil {
                err = wrap(prog.position, err)
                return
        }
        if ordered, err = prog.prerequisites(pc, prog.ordered); err != nil {
                err = wrap(prog.position, err)
                return
        }

        if len(depends) > 0 { pc.def.depends.set(DefDefault, MakeList(prog.position, depends...)) }
        if len(ordered) > 0 { pc.def.ordered.set(DefDefault, MakeList(prog.position, ordered...)) }

        if pc.stems != nil {
                pc.def.stem.set(DefDefault, &String{trivial{prog.position},pc.stems[0]})
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

                result, err = pc.def.modbuff.Call(prog.position)
                if err != nil {
                        // NOTE: err could be breakCase, breakDone, etc.
                        err = wrap(prog.position, err)
                }
        } ()
        return pc.exec(prog)
}

func (pc *traversal) exec(prog *Program) (result Value, err error) {
        if optionTraceExec {
                var d = pc.depth()
                var t = pc.def.target.value
                var s = fmt.Sprintf("%s: %v (%p, exec.depth=%d)", typeof(t), t, t, d)
                defer un(trace(t_exec, s))
        }

        var pos = prog.position
        if n, ok := pc.visited[pc.def.target.value]; ok && n > 0 {
                if optionTraceExec { t_exec.trace(fmt.Sprintf("visited: %v", pc.def.target.value)) }
                if false { return }
        }
        pc.visited[pc.def.target.value] += 1

        // Update normal prerequisites
        if err = pc.traverseNormalPrerequisites(pos); err != nil {
                return
        }

        // Update order-only prerequisites
        if err = pc.traverseOrderOnlyPrerequisites(pos); err != nil {
                return
        }

        // Update grapped files
        if err = pc.traverseGrepFiles(pos); err != nil {
                return
        }

        if len(pc.interpreted) == 0 {
                // Using the default statements interpreter.
                if i, ok := dialects["eval"]; ok && i != nil {
                        if err = prog.interpret(pc, i, nil); err != nil {
                                err = wrap(pos, err)
                        }
                } else {
                        err = errorf(pos, "no default dialect")
                }
        }

        if optionTraceExec {
                t_exec.trace(pc.def.stem)
                t_exec.trace(pc.def.target)
                t_exec.trace(pc.def.depend0)
                t_exec.trace(pc.def.depends)
                t_exec.trace(pc.def.ordered)
                t_exec.trace(pc.def.grepped)
                t_exec.trace(pc.def.updated)
                t_exec.trace(pc.def.modbuff)
        }
        return
}

func (pc *traversal) traverseNormalPrerequisites(pos Position) (err error) {
        if optionTraceExec { defer un(trace(t_exec, pc.def.depends.name)) }
        if err = pc.traverseAll([]Value{pc.def.depends.value}); err != nil {
                err = wrap(pos, pc.wait(pos), err)
        } else if err = pc.wait(pos); err != nil {  }
        if len(pc.targets) > 0 {
                pc.def.depend0.value = pc.targets[0] // $<
                pc.def.depends.value = pc.targets[0] // $^
                for _, t := range pc.targets[1:] {
                        pc.def.depends.append(t)
                }
                pc.targets = nil
        }
        if len(pc.updated) > 0 {
                pc.def.updated.value = pc.updated[0].target // $?
                for _, t := range pc.updated[1:] {
                        pc.def.updated.append(t.target)
                }
        }
        return
}

func (pc *traversal) traverseOrderOnlyPrerequisites(pos Position) (err error) {
        if optionTraceExec { defer un(trace(t_exec, pc.def.ordered.name)) }
        if err = pc.traverseAll([]Value{pc.def.ordered.value}); err != nil {
                err = wrap(pos, pc.wait(pos), err)
        } else if err = pc.wait(pos); err != nil {  }
        if len(pc.targets) > 0 {
                pc.def.ordered.value = pc.targets[0] // $|
                for _, t := range pc.targets[1:] {
                        pc.def.ordered.append(t)
                }
                pc.targets = nil
        }
        return
}

func (pc *traversal) traverseGrepFiles(pos Position) (err error) {
        if optionTraceExec { defer un(trace(t_exec, pc.def.grepped.name)) }
        if err = pc.traverseAll([]Value{pc.def.grepped.value}); err != nil {
                err = wrap(pos, pc.wait(pos), err)
        } else if err = pc.wait(pos); err != nil {  }
        if len(pc.targets) > 0 {
                pc.def.grepped.value = pc.targets[0]
                for _, t := range pc.targets[1:] {
                        pc.def.grepped.append(t)
                }
                pc.targets = nil
        }
        return
}

func (prog *Program) passExecution(position Position, entry *RuleEntry, args... Value) (result []Value, err error) {
        result, err = Executer(entry).Execute(position, args...)
        return
}
