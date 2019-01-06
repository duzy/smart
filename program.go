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
        globe   *Globe
        project *Project
        scope   *Scope
        stem    string // set by preparer, for pattern rules
        level   int // execution level/depth
        params  []string
        depends []Value
        recipes []Value
        pipline []*modifier
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

func (prog *Program) interpret(i interpreter, print bool, out *Def, params []Value) (err error) {
        var target, value Value
        if value, err = i.Evaluate(prog, params); err == nil {
                if value != nil {
                        out.set(DefDefault, value)
                }
                def := prog.scope.Lookup("@").(*Def)
                if target, err = def.Call(prog.position); err == nil {
                        var strings []string
                        for _, recipe := range prog.recipes {
                                // Avoids calling recipe.Strval() twice, so that it won't be
                                // evaluated more than once.
                                strings = append(strings, recipe.String())
                        }
                        _, _, err = prog.project.UpdateCmdHash(target, strings)
                }
        }
        return
}

func (prog *Program) modify(m *modifier, post, print bool, interpreted *string, out *Def) (err error) {
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
                switch name {
                case "configure": if post && *interpreted == "" {
                        if i, ok := dialects["eval"]; ok && i != nil {
                                err = prog.interpret(i, print, out, v)
                                *interpreted = "eval"
                        }
                }}
                if err == nil {
                        var value Value
                        if value, err = f(prog.position, prog, v...); err == nil && value != nil {
                                out.set(DefDefault, value)
                        }
                }
        } else if i, _ := dialects[name]; i != nil {
                err = prog.interpret(i, print, out, v)
                *interpreted = name // return dialect name
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

func (prog *Program) Execute(entry *RuleEntry, args []Value) (result Value, err error) {
        var print = true // printing work directories (Entering/Leaving)

        // Flag targets (-foo) turn off printing
        if _, ok := entry.target.(*Flag); ok { print = false }
        if print {
                if t, ok := entry.target.(*Bareword); ok {
                        if t.string == "use" { print = false }
                }
        }
        //if print && prog.getModifier("cd") != nil { print = false }
        if print && prog.getModifier("configure") != nil { print = false }

        // cd before setting execstack, because cd reads execstack
        // before changes.
        var lenEnters = len(cd.stack)
        if err = enter(prog, prog.project.absPath); err != nil { return }
        cd.stack[0].silent = !print

        // must set execstack after entering project
        defer setexecstack(setexecstack(execstack.unshift(prog))) // build the call stack
        defer setclosure(setclosure(cloctx.unshift(prog.scope))) // entry.DeclScope()
        defer func() { // leaving after setting execstack to meet the FIFO order of execstack
                e := leave(prog, lenEnters)
                if err == nil { err = e } else if e != nil {
                        fmt.Fprintf(os.Stderr, "%s: leaving: %s\n", entry.Position, e)
                }
        } ()

        // select the right target value
        var target Value = universalnone
        switch t := entry.target.(type) {
        case *File: target = t
        default:
                var s string
                if s, err = entry.target.Strval(); err != nil { return }
                if file := prog.project.file(s); file == nil {
                        target = entry.target
                } else {
                        target = file
                }
        }

        // set $@, $^, $<, etc before pre-modifiers.
        var targDef, preqDef, pre0Def *Def
        if targDef, err = prog.auto("@", target); err != nil {
                return
        } else if targDef == nil {
                err = scanner.Errorf(prog.position, "undefined target automatic")
                return
        }
        if preqDef, err = prog.auto("^", universalnone); err != nil {
                return
        }
        if pre0Def, err = prog.auto("<", universalnone); err != nil {
                return
        }

        var params []*Def
        for i, param := range prog.params {
                var def *Def
                if def, err = prog.auto(param, universalnone); err != nil { return }
                prog.scope.replace(strconv.Itoa(i+1), def)
                params = append(params, def)
        }

        // setup named/number parameters ($1, $2, ..., $9, etc.)
        var argnum int
        for _, a := range args {
                var def *Def
                switch t := a.(type) {
                case *Flag:
                        // TODO: parsing flags
                case *Pair:
                        var s string
                        if s, err = t.Key.Strval(); err == nil {
                                if o := prog.scope.Lookup(s); o != nil {
                                        def = o.(*Def)
                                        def.set(DefDefault, t.Value)
                                } else {
                                        err = scanner.Errorf(prog.position, "`%s` no such named parameter", s)
                                }
                        }
                default:
                        if argnum < len(params) {
                                def = params[argnum]
                        } else {
                                name := strconv.Itoa(argnum+1)
                                if def, err = prog.auto(name, a); err == nil {
                                        params = append(params, def)
                                }
                        }
                        argnum += 1
                }
                if err != nil {
                        return
                }
        }

        // Modifier buffer.
        var modifyBuf *Def
        if modifyBuf, err = prog.auto("-", universalnone); err != nil { return }
        defer func() {
                for _, param := range params {
                        param.set(DefDefault, universalnone)
                }
                result = modifyBuf.Value
        } ()


        // Pre-interpreted & post-interpreted dialect name.
        var preInterpreted, postInterpreted string

        // Split modifiers by '|', if no '|', all goes postModifiers.
        var preModifiers, postModifiers []*modifier
        for i, m := range prog.pipline {
                if m.name == modifierbar {
                        preModifiers = prog.pipline[:i]
                        postModifiers = prog.pipline[i+1:]
                        goto PrePipe // Process the pipeline immediately
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

        // Pre-modifiers could change $@, $^, $<, etc.
PrePipe:
        for _, m := range preModifiers {
                if m.name == modifierbar { continue }
                if err = prog.modify(m, false, print, &preInterpreted, modifyBuf); err != nil {
                        if br, ok := err.(*breaker); ok && br.good {
                                // Discard err and change dialect to avoid
                                // default interpreter being called.
                                err, preInterpreted = nil, "--"
                        }
                        break PrePipe
                }
        }

        // Expanding all dependencies after pre-modifiers.
        var depends []Value
        if depends, err = mergeresult(ExpandAll(prog.depends...)); err != nil { return }
        preqDef.append(depends...)

        // Calculate and prepare depends and files.
        var pc = &preparer{
                entry, prog, nil, new(List),
                prog.stem, prog.level,
        }
        if err = pc.updateall(preqDef); err != nil {
                if false {
                        fmt.Fprintf(os.Stdout, "%s: %s\n", entry.Position, err)
                }
                return
        } else if pc.targets.Len() > 0 {
                var elems = pc.targets.Elems[:]
                for i := 0; i < len(elems); i += 1 {
                        for j := i + 1; j < len(elems); j += 1 {
                                if dependEquals(elems[i], elems[j]) {
                                        elems = append(elems[:j], elems[j+1:]...)
                                        j -= 1
                                }
                        }
                }
                pc.targets.Elems = elems
                preqDef.set(DefDefault, pc.targets)
                pre0Def.set(DefDefault, pc.targets.Elems[0])
        }

PostPipe:
        for _, m := range postModifiers {
                if m.name == modifierbar { continue }
                if err = prog.modify(m, true, print, &postInterpreted, modifyBuf); err != nil {
                        if br, ok := err.(*breaker); ok && br.good {
                                // Discard err and change dialect to
                                // avoid default interpreter being
                                // called.
                                err, postInterpreted = nil, "--"
                        }
                        break PostPipe
                }
        }

        if err == nil && preInterpreted == "" && postInterpreted == "" {
                // Using the default statements interpreter.
                if i, ok := dialects["eval"]; ok && i != nil {
                        err = prog.interpret(i, print, modifyBuf, nil)
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

func dependEquals(a, b Value) bool {
        if a == b {
                return true
        }

        // TODO: more advanced checking "the same depend"

        var (
                sa, sb string
                err error
        )
        if sa, err = a.Strval(); err != nil {
                return false
        }
        if sb, err = b.Strval(); err != nil {
                return false
        }
        return sa == sb
}
