//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/scanner"
        "extbit.io/smart/token"
        "strconv"
        //"strings"
        "sync"
        "fmt"
        //"os"
)

type dependPatternUnfit struct {
}

func (*dependPatternUnfit) Error() string { return "pattern unfit" }

type executestack []*Program

var execstack executestack // latest on top

func setexecstack(v []*Program) (saved []*Program) {
        saved = execstack; execstack = v; return
}

func (xs executestack) unshift(progs... *Program) executestack {
        /*var res executestack
        ForProgs: for _, prog := range progs {
                for _, x := range xs {
                        if prog.project == x.project { continue ForProgs }
                }
                res = append(res, prog)
        }
        return append(res, execstack...)*/
        return append(progs, execstack...)
}

func (xs executestack) projects(first *Project) (res []*Project) {
        if first != nil { res = append(res, first) }
ForXS:
        for _, x := range xs {
                for _, p := range res {
                        if x.project == p { continue ForXS }
                }
                res = append(res, x.project)
        }
        return
}

func (xs executestack) String() (s string) {
        for i, x := range xs {
                if i > 0 { s += " " }
                s += x.project.name
        }
        return fmt.Sprintf("[%s]", s)
}

type Program struct {
        mutex *sync.Mutex // execution mutex
        pc *traversal // current prepare context
        project *Project
        scope   *Scope
        params  []string
        depends []Value
        ordered []Value
        recipes []Value
        callers []*traversecontext
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

func (prog *Program) waitForPrerequisites() (err error) {
        prog.pc.group.Wait()
        if prog.pc.calleeErrors != nil {
                // TODO: combine the callee errors
                err = prog.pc.calleeErrors[0]
        }
        return
}

func (prog *Program) interpret(pc *traversal, i interpreter, params []Value) (err error) {
        //if pc.breaker != nil || (exists(pc.targetDef.value) && len(pc.updated) == 0) {
        //        return
        //}
        if pc.breaker != nil { return }
        if err = prog.waitForPrerequisites(); err != nil {
                return
        }

        var value Value
        if value, err = i.Evaluate(prog, params); err == nil {
                if value != nil { pc.modifyBuf.set(DefDefault, value) }
                if value, err = prog.pc.targetDef.Call(prog.position); err == nil {
                        var strings []string
                        for _, recipe := range prog.recipes {
                                // Avoids calling recipe.Strval() twice, so that it won't be
                                // evaluated more than once.
                                strings = append(strings, recipe.String())
                        }
                        _, _, err = prog.project.UpdateCmdHash(value, strings)
                }
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
                                pc.modifyBuf.set(DefDefault, value)
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

func (prog *Program) prerequisites(args []Value) (result []Value, err error) {
        // IMPORTANT: don't expand the args here. The prerequisites like
        // '$(or &@,...)' have to be expanded when it's used (e.g. compare).
        //      xxx: mergeresult(ExpandAll(args...))
        for _, arg := range args {
                switch a := arg.(type) {
                case Pattern: //*PercPattern:
                        var s string
                        var rest []string
                        if s, rest, err = a.stencil(prog.pc.stems); err != nil {
                                err = wrap(prog.position, err)
                                return
                        }
                        if len(rest) > 0 {
                                panic(fmt.Sprintf("FIXME: unhandled stems: %v (%v, %v) (%v)", arg, s, rest, prog.pc.stems))
                        }
                        if prog.pc.derived == nil {
                                // FIXME: prog.project.matchFile(s) ???
                        } else if file := prog.pc.derived.matchFile(s); file != nil {
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
                case *GlobPattern:
                        unreachable("`%s` glob pattern unsupported", a)
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

func (prog *Program) Execute(entry *RuleEntry, args []Value) (result Value, err error) {
        if false {
                // Execution can be nested, a program.mutex.lock may
                // cause dead-lock in such case.
                prog.mutex.Lock()
                defer prog.mutex.Unlock()
        }

        defer func(s string) {
                prog.project.changedWD = s
        } (prog.project.changedWD)

        var none = &None{trivial{prog.position}}
        var ctx = traversecontext{
                derived: mostDerived(),
                group: new(sync.WaitGroup),
                entry: entry,
                args: args,
        }
        if len(prog.callers) > 0 {
                var caller = prog.callers[0]
                ctx.stems = caller.stems
                if optionTraceTraversalNestIndent {
                        ctx.traceLevel = caller.traceLevel
                }
        }

        var nestedExecution bool
        if pc := prog.pc; pc != nil {
                setVal := func(def *Def, val Value) { def.value = val }
                for _, def := range []*Def{
                        pc.targetDef,
                        pc.dependsDef,
                        pc.depend0Def,
                        pc.orderedDef,
                        pc.greppedDef,
                        pc.updatedDef,
                        pc.stemDef,
                        pc.modifyBuf,
                } { defer setVal(def, def.value) }
                nestedExecution = true
        }
        defer func(pc *traversal) { prog.pc = pc } (prog.pc)
        prog.pc = &traversal{
                program: prog,
                traversecontext: ctx,
                print: true,
        }

        // Flag targets (-foo) turn off printing
        if _, ok := prog.pc.entry.target.(*Flag); ok { prog.pc.print = false }
        if prog.pc.print && prog.pc.entry.class == UseRuleEntry {
                prog.pc.print = false
        }
        if prog.pc.print && prog.getModifier("configure") != nil { prog.pc.print = false }

        // cd before setting execstack, because cd reads execstack
        // before changes.
        var enterStop *enterec
        if len(cd.stack) > 0 { enterStop = cd.stack[0] }
        if err = enter(prog, prog.project.absPath); err != nil {
                err = wrap(prog.position, err)
                return
        }
        cd.stack[0].silent = !prog.pc.print

        // must set execstack after entering project
        defer setexecstack(setexecstack(execstack.unshift(prog))) // build the call stack
        defer setclosure(setclosure(cloctx.unshift(prog.scope))) // entry.DeclScope()
        defer func() { // leaving after setting execstack to meet the FIFO order of execstack
                if e := leave(prog, enterStop); e != nil {
                        // NOTE: err could be breakCase, breakDone, etc.
                        if err == nil { err = e } else {
                                fmt.Fprintf(stderr, "%s: leaving: %s\n", prog.pc.entry.Position, e)
                        }
                }
        } ()

        // set $@, $^, $<, $|, $~, $?, etc
        if prog.pc.targetDef,  err = prog.auto("@", none); err != nil { return }
        if prog.pc.depend0Def, err = prog.auto("<", none); err != nil { return }
        if prog.pc.dependsDef, err = prog.auto("^", none); err != nil { return }
        if prog.pc.orderedDef, err = prog.auto("|", none); err != nil { return }
        if prog.pc.greppedDef, err = prog.auto("~", none); err != nil { return }
        if prog.pc.updatedDef, err = prog.auto("?", none); err != nil { return }
        if prog.pc.stemDef,    err = prog.auto("*", none); err != nil { return }
        if prog.pc.modifyBuf,  err = prog.auto("-", none); err != nil { return }

        var fileTarget *File

        // Select the right target value before setting parameters,
        // because the target could be overrided by parameters.
        switch t := prog.pc.entry.target.(type) {
        case *File:
                prog.pc.targetDef.set(DefDefault, t)
                fileTarget = t
        default:
                var name string
                var target = prog.pc.entry.target
                if name, err = target.Strval(); err != nil {
                        err = wrap(prog.position, err)
                        return
                }
                if file := prog.project.searchFile(name); file != nil {
                        fileTarget = file
                        target = file
                }
                prog.pc.targetDef.set(DefDefault, target)
        }

        if e, clearParams := prog.setParams(args); e != nil {
                err = e; return
        } else {
                defer func() {
                        clearParams()
                        prog.pc.params = nil
                } ()
        }

        // Expanding all dependencies after pre-modifiers.
        var depends, ordered []Value
        if depends, err = prog.prerequisites(prog.depends); err != nil {
                err = wrap(prog.position, err)
                return
        }
        if ordered, err = prog.prerequisites(prog.ordered); err != nil {
                err = wrap(prog.position, err)
                return
        }

        prog.pc.dependsDef.set(DefDefault, MakeList(prog.position, depends...))
        if len(depends) > 0 {
                prog.pc.depend0Def.set(DefDefault, depends[0])
        }
        if len(ordered) > 0 {
                prog.pc.orderedDef.set(DefDefault, MakeList(prog.position, ordered...))
        }

        if prog.pc.stems != nil {
                prog.pc.stemDef.set(DefDefault, &String{trivial{prog.position},prog.pc.stems[0]})
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

                result, err = prog.pc.modifyBuf.Call(prog.position)
                if err != nil {
                        // NOTE: err could be breakCase, breakDone, etc.
                        err = wrap(prog.position, err)
                }
        } ()
        return prog.pc.exec(prog, nestedExecution)
}

func (pc *traversal) checkTargetMode() (err error) {
        // Check (file) target existence
        var s string
        if file, ok := pc.targetDef.value.(*File); ok && !exists(file) {
                //if len(prog.callers) > 0 {
                //        var caller = prog.callers[0]
                //        caller.updated = append(...)
                //}
        } else if s, err = pc.targetDef.value.Strval(); err != nil {
                return
        } else if file := pc.derived.matchFile(s); !exists(file) {
                pc.targetDef.value = file
        }
        return
}

func (pc *traversal) exec(prog *Program, nested bool) (result Value, err error) {
        var (
                none = &None{trivial{prog.position}}
                depends = pc.dependsDef.value
                ordered = pc.orderedDef.value
                grepped = pc.greppedDef.value
        )

        pc.updatedDef.set(DefDefault, none)

        if err = pc.traverseAll([]Value{depends,ordered,grepped}, nested); err != nil {
                return
        } else if optionTraceTraversal && false {
                pc.tracef("%v", pc.targets)
                pc.tracef("%v", pc.targetDef)
                pc.tracef("%v", pc.depend0Def)
                pc.tracef("%v", pc.dependsDef)
                pc.tracef("%v", pc.orderedDef)
                pc.tracef("%v", pc.greppedDef)
                pc.tracef("%v", pc.updatedDef)
                pc.tracef("%v", pc.stemDef)
                pc.tracef("%v", pc.modifyBuf)
        }

        if len(pc.interpreted) == 0 {
                // Using the default statements interpreter.
                if i, ok := dialects["eval"]; ok && i != nil {
                        if err = prog.interpret(pc, i, nil); err != nil {
                                err = wrap(prog.position, err)
                        }
                } else {
                        err = errorf(prog.position, "no default dialect")
                }
        }
        return
}

func (prog *Program) passExecution(position Position, entry *RuleEntry, args... Value) (result []Value, err error) {
        result, err = Executer(entry).Execute(position, args...)
        return
}
