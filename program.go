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
        "fmt"
        "os"
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

type modifier struct {
        position token.Position
        name Value
        args []Value
}

func (m *modifier) String() (s string) {
        s = "(" + m.name.String()
        for _, a := range m.args {
                s += " " + a.String()
        }
        s += ")"
        return
}

type Program struct {
        pc *preparer // current prepare context
        globe   *Globe
        project *Project
        scope   *Scope
        params  []string
        depends []Value
        ordered []Value
        recipes []Value
        pipline []*modifier
        callers []*preparecontext
        position token.Position
}

func (prog *Program) Position() token.Position { return prog.position }
func (prog *Program) Project() *Project { return prog.project }
func (prog *Program) Scope() *Scope { return prog.scope }

func (prog *Program) auto(name string, value Value) (auto *Def, err error) {
        var alt Object
        if auto, alt = prog.scope.Def(prog.project, name, value); alt != nil {
                var found = false
                if auto, found = alt.(*Def); found {
                        auto.set(DefDefault, value)
                } else {
                        err = fmt.Errorf("`%v` name already taken (%T)", name, alt)
                }
        }
        if enable_assertions {
                assert(auto.Value == value, "wrong auto value")
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

func (prog *Program) interpret(pc *preparer, i interpreter, params []Value) (err error) {
        var mode = prog.pc.mode
        if prog.scope.comment != usecomment {
                if len(prog.depends) == 0 {
                        // If the program has no prerequisites and not
                        // updated yet, we force to update it, e.g.:
                        //
                        //      foobar.cpp:; println "name: $@"
                        //
                        // As it's supposed to be invoked alone.
                } else if mode != updateMode {
                        return
                }
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

func (prog *Program) modify(pc *preparer, m *modifier, post bool) (err error) {
        // TODO: using rules in a different project to implement modifiers, e.g.
        //       [ foo.check-preprequisites ]
        //       [ foo.baaaar ]
        var v []Value
        var name string
        if v, err = mergeresult(ExpandAll(m.name)); err != nil {
                return
        } else if name, err = v[0].Strval(); err != nil {
                return
        } else {
                v = append(v[1:], m.args...)
        }

        if f, ok := modifiers[name]; ok {
                // Special modifier processing (implicit interpretation)
                if post && name == "configure" && pc.interpreted == nil {
                        // Evaluate for configure modifier
                        if i, ok := dialects["eval"]; ok && i != nil {
                                err = prog.interpret(pc, i, v)
                        }
                }
                if err == nil {
                        var value Value
                        if value, err = f(prog.position, prog, v...); err == nil && value != nil {
                                pc.modifyBuf.set(DefDefault, value)
                        }
                }
        } else if i, _ := dialects[name]; i != nil {
                err = prog.interpret(pc, i, v)
        } else {
                err = fmt.Errorf("no modifier or dialect '%s'", name)
        }
        return
}

func (prog *Program) getModifier(name string) (res *modifier) {
        for _, m := range prog.pipline {
                if s, err := m.name.Strval(); err != nil {
                        return
                } else if name == s {
                        res = m
                }
        }
        return
}

func (prog *Program) hasCDDash() (res bool) {
        if m := prog.getModifier("cd"); m != nil && len(m.args) > 0 {
                var (
                        s string
                        e error
                )
                if s, e = m.args[0].Strval(); e != nil {
                        // TODO: error...
                } else if s == "-" {
                        res = true
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
                case *PercPattern:
                        var s string
                        if s, err = a.MakeString(prog.pc.stem); err != nil { return }
                        if file := prog.pc.derived.matchFile(s); file != nil {
                                result = append(result, file)
                                break
                        }
                        if true {
                                result = append(result, a)
                        } else if false {
                                result = append(result, &String{s})
                        } else {
                                err = scanner.Errorf(prog.position, "`%s` unknown target (via %s)", s, a)
                        }
                case *GlobPattern:
                        unreachable("`%s` glob pattern unsupported", a)
                default:
                        result = append(result, a)
                }
        }
        return
}

// Split modifiers by '|', if no '|', all goes postModifiers.
func (prog *Program) modifiers() (preModifiers, postModifiers []*modifier) {
        for i, m := range prog.pipline {
                if m.name == modifierbar {
                        preModifiers = prog.pipline[:i]
                        postModifiers = prog.pipline[i+1:]
                        return
                }
        }
        if len(postModifiers) == 0 {
                if n := len(preModifiers); n > 0 {
                        postModifiers = preModifiers
                        preModifiers = nil
                } else if n == 0 {
                        postModifiers = prog.pipline
                }
        }
        return
}

func (prog *Program) Execute(entry *RuleEntry, args []Value) (result Value, err error) {
        ctx := preparecontext{entry:entry, args:args, derived:mostDerived()}
        if len(prog.callers) > 0 {
                var caller = prog.callers[0]
                ctx.level = caller.level
                //ctx.mode = caller.mode
                ctx.stem = caller.stem
                if caller.mode == updateMode {
                        // Only update if the caller found it updated.
                        if !caller.isUpdatedTarget(entry.target) {
                                ctx.visitInsteadUpdate = true
                        }
                }
        }

        // Build related project list from derived.
        if owner := entry.OwnerProject(); ctx.derived == nil {
                ctx.related = append(ctx.related, owner)
        } else if ctx.derived.isa(owner) {
                ctx.related = append(ctx.related, ctx.derived)
        } else {
                ctx.related = append(ctx.related, owner, ctx.derived)
        }

        defer func(pc *preparer) { prog.pc = pc } (prog.pc)
        prog.pc = &preparer{program:prog, preparecontext:ctx, print:true}

        // Flag targets (-foo) turn off printing
        if _, ok := prog.pc.entry.target.(*Flag); ok { prog.pc.print = false }
        if prog.pc.print {
                if t, ok := prog.pc.entry.target.(*Bareword); ok {
                        if t.string == "use" { prog.pc.print = false }
                }
        }
        if prog.pc.print && prog.getModifier("configure") != nil { prog.pc.print = false }

        // cd before setting execstack, because cd reads execstack
        // before changes.
        var lenEnters = len(cd.stack)
        if err = enter(prog, prog.project.absPath); err != nil { return }
        cd.stack[0].silent = !prog.pc.print

        // must set execstack after entering project
        defer setexecstack(setexecstack(execstack.unshift(prog))) // build the call stack
        defer setclosure(setclosure(cloctx.unshift(prog.scope))) // entry.DeclScope()
        defer func() { // leaving after setting execstack to meet the FIFO order of execstack
                e := leave(prog, lenEnters)
                if err == nil { err = e } else if e != nil {
                        fmt.Fprintf(os.Stderr, "%s: leaving: %s\n", prog.pc.entry.Position, e)
                }
        } ()

        // set $@, $^, $<, $|, $~, $?, etc
        if prog.pc.targetDef,  err = prog.auto("@", universalnone); err != nil { return }
        if prog.pc.dependsDef, err = prog.auto("^", universalnone); err != nil { return }
        if prog.pc.depend0Def, err = prog.auto("<", universalnone); err != nil { return }
        if prog.pc.orderedDef, err = prog.auto("|", universalnone); err != nil { return }
        if prog.pc.greppedDef, err = prog.auto("~", universalnone); err != nil { return }
        if prog.pc.updatedDef, err = prog.auto("?", universalnone); err != nil { return }
        if prog.pc.stemDef,    err = prog.auto("*", universalnone); err != nil { return }
        if prog.pc.modifyBuf,  err = prog.auto("-", universalnone); err != nil { return }

        // Select the right target value before setting parameters,
        // because the target could be overrided by parameters.
        switch t := prog.pc.entry.target.(type) {
        case *File: prog.pc.targetDef.set(DefDefault, t)
        default:
                var name string
                var target = prog.pc.entry.target
                if name, err = target.Strval(); err != nil { return }
                if file := prog.project.searchFile(name); file != nil {
                        target = file
                }
                prog.pc.targetDef.set(DefDefault, target)
        }

        defer func() {
                for _, param := range prog.pc.params {
                        param.set(DefDefault, universalnone)
                }
                prog.pc.params = nil
        } ()
        for i, param := range prog.params {
                var def *Def
                if def, err = prog.auto(param, universalnone); err != nil { return }
                prog.scope.replace(strconv.Itoa(i+1), def)
                prog.pc.params = append(prog.pc.params, def)
        }
        var argnum int // setup named/number parameters ($1, $2, etc.)
        for _, a := range prog.pc.args {
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
                                        err = scanner.Errorf(prog.position, "`%s` no such named parameter", s)
                                }
                        }
                default:
                        var def *Def
                        if argnum < len(prog.pc.params) {
                                def = prog.pc.params[argnum]
                                def.set(DefDefault, a)
                        } else {
                                name := strconv.Itoa(argnum+1)
                                if def, err = prog.auto(name, a); err == nil {
                                        prog.pc.params = append(prog.pc.params, def)
                                }
                        }
                        argnum += 1
                }
                if err != nil { return }
        }

        // Expanding all dependencies after pre-modifiers.
        var depends, ordered []Value
        if depends, err = prog.prerequisites(prog.depends); err != nil { return }
        if ordered, err = prog.prerequisites(prog.ordered); err != nil { return }
        if len(depends) > 0 {
                prog.pc.dependsDef.append(depends...)
                prog.pc.depend0Def.append(depends[0])
        }
        if len(ordered) > 0 {
                prog.pc.orderedDef.append(ordered...)
        }

        if prog.pc.stem != "" {
                prog.pc.stemDef.set(DefDefault, &String{prog.pc.stem})
        }

        defer func() { result = prog.pc.modifyBuf.Value } ()
        prog.pc.preModifiers, prog.pc.postModifiers = prog.modifiers()
        return prog.pc.exec(prog)
}

func (pc *preparer) checkUpdates(src error) (err error) {
        if src != nil {
                var br, ok = src.(*breaker)
                if ok && br.what == breakUpdates {
                        pc.updated = append(pc.updated, br.updated...)
                        for _, updated := range br.updated {
                                pc.updatedDef.append(updated.target)
                        }

                        if len(pc.updated) > 0 {
                                // switch into update mode
                                pc.mode = updateMode
                        } else {
                                err = pc.checkTargetMode()
                        }
                } else {
                        err = src
                }
        }
        return
}

func (pc *preparer) checkTargetMode() (err error) {
        // Check (file) target existence
        var s string
        if file, ok := pc.targetDef.Value.(*File); ok && !file.exists() {
                pc.mode = updateMode // switch into update mode
                //if len(prog.callers) > 0 {
                //        var caller = prog.callers[0]
                //        caller.updated = append(...)
                //}
        } else if s, err = pc.targetDef.Value.Strval(); err != nil {
                return
        } else if file := pc.derived.matchFile(s); file != nil && !file.exists() {
                pc.mode = updateMode // switch into update mode
                pc.targetDef.Value = file
        }
        return
}

func (pc *preparer) checkMode4Breaker(tag string, name Value, br *breaker) (done bool, err error) {
        switch tag = fmt.Sprintf("(%s) %s:", tag, name); br.what {
        case breakBad:
                if trace_prepare { pc.trace(tag, "(bad)", br.message) }
                err = scanner.Errorf(br.pos, br.message)
        case breakGood:
                //if trace_prepare { pc.trace(tag, "(good)") }
                err = pc.checkTargetMode()
        case breakUpdates:
                if trace_prepare { pc.trace(tag, "(updates)", br.updated) }

                // Collect updates, so that the updated targets could be
                // returned to the caller.
                pc.updated = append(pc.updated, br.updated...)
                for _, updated := range br.updated {
                        pc.updatedDef.append(updated.target)
                }

                if len(br.updated) > 0 {
                        pc.mode = updateMode // switch into update mode
                } else {
                        err = pc.checkTargetMode()
                }
        }
        return
}

func (pc *preparer) preModify(prog *Program) (done bool, err error) {
        for _, m := range pc.preModifiers {
                if m.name == modifierbar { continue }
                if err = prog.modify(pc, m, false); err == nil { continue }
                if br, ok := err.(*breaker); ok /*&& pc.mode == defaultMode*/ {
                        done, err = pc.checkMode4Breaker("pre", m.name, br)
                }
                if err != nil { break }
        }
        return
}

func (pc *preparer) postModify(prog *Program) (done bool, err error) {
        for _, m := range pc.postModifiers {
                if m.name == modifierbar { continue }
                if err = prog.modify(pc, m, true); err == nil { continue }
                if br, ok := err.(*breaker); ok /*&& pc.mode == defaultMode*/ {
                        done, err = pc.checkMode4Breaker("post", m.name, br)
                }
                if err != nil { break }
        }
        return
}

func (pc *preparer) exec(prog *Program) (result Value, err error) {
        pc.updatedDef.set(DefDefault, universalnone)

        // Defers to collect all updates.
        defer func() {
                if err == nil && pc.updated != nil {
                        // Return the updates to caller program.
                        err = break_updates(prog.position, pc.updated...)
                }
        } ()

        var done bool

        // Pre-modifying could change $@, $^, $<, $|, etc.
        if done, err = pc.preModify(prog); err != nil || done { return }
        if pc.visitInsteadUpdate && pc.mode == updateMode {
                // Work in visit mode to ensure all it's dependencies will
                // be updated.
                //pc.mode = visitMode
        }

        // Updating $^
        pc.targets = nil // clear the target list
        if err = pc.traverseAll(pc.dependsDef); err != nil { return }
        if n := len(pc.targets); n == 0 {
                pc.dependsDef.set(DefDefault, universalnone)
                pc.depend0Def.set(DefDefault, universalnone)
        } else if n == 1 {
                if trace_prepare { pc.tracef("$^: %v", pc.targets[0]) }
                pc.dependsDef.set(DefDefault, pc.targets[0])
                pc.depend0Def.set(DefDefault, pc.targets[0])
        } else if n > 1 {
                if trace_prepare { pc.tracef("$^: (%d) %v", n, pc.targets) }
                pc.dependsDef.set(DefDefault, MakeList(pc.targets...))
                pc.depend0Def.set(DefDefault, pc.targets[0])
        }

        // Updating $|
        pc.targets = nil // clear the target list
        if err = pc.traverseAll(pc.orderedDef); err != nil { return }
        if n := len(pc.targets); n == 0 {
                pc.orderedDef.set(DefDefault, universalnone)
        } else {
                if trace_prepare { pc.tracef("$|: (%d) %v", n, pc.targets) }
                pc.orderedDef.set(DefDefault, MakeList(pc.targets...))
        }

        // Updating $~
        pc.targets = nil // clear the target list
        if err = pc.traverseAll(pc.greppedDef); err != nil { return }
        if n := len(pc.targets); n == 0 {
                pc.greppedDef.set(DefDefault, universalnone)
        } else {
                if trace_prepare { pc.tracef("$~: (%d) %v", n, pc.targets) }
                pc.greppedDef.set(DefDefault, MakeList(pc.targets...))
        }

        pc.targets = nil // clear the target list

        // Post modifying
        if done, err = pc.postModify(prog); err != nil || done { return }
        if pc.interpreted == nil {
                // Using the default statements interpreter.
                if i, ok := dialects["eval"]; ok && i != nil {
                        err = prog.interpret(pc, i, nil)
                } else {
                        err = fmt.Errorf("no default dialect")
                }
        }
        return
}

func (prog *Program) pipe(position token.Position, operation Value) (m *modifier, err error) {
        switch g := operation.(type) {
        case *Group:
                m = &modifier{ position, g.Get(0), g.Slice(1) }
        case *ModifierBar:
                m = &modifier{ position, g, nil }
        default:
                err = fmt.Errorf("unknown modifier (%T `%v`)", operation, operation)
                //fmt.Fprintf(os.Stderr, "%s: %v\n", prog.position, err)
        }
        if m != nil && err == nil {
                prog.pipline = append(prog.pipline, m)
        }
        return
}

func (prog *Program) passExecution(position token.Position, entry *RuleEntry, args... Value) (result []Value, err error) {
        result, err = Executer(entry).Execute(position, args...)
        return
}
