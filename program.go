//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
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
        return append(progs, execstack...)
}

var printstack []string
var printcd = true

type cdinfo struct {
        workdir, chdir string
        print bool
}

type cdrecord struct {
        chdir string
        n int
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
        params  []string
        depends []Value
        recipes []Value
        pipline []*modifier
        cdinfos []*cdinfo
        subcdrs []string // sub-program cd print records
        position token.Position
}

func (prog *Program) Position() token.Position { return prog.position }
func (prog *Program) Project() *Project { return prog.project }
func (prog *Program) Scope() *Scope { return prog.scope }

func (prog *Program) auto(name string, value Value) (auto *Def) {
        var alt Object
        if auto, alt = prog.scope.Def(prog.project, name, value); alt != nil {
                var found = false
                if auto, found = alt.(*Def); found {
                        auto.Assign(value)
                } else {
                        Fail("Name '%v' already taken, not auto (%T)", name, alt)
                }
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

func (prog *Program) interpret(i Interpreter, out *Def, params []Value) (err error) {
        var (
                recipes []Value
                target, value Value
        )
        if recipes, err = DiscloseAll(prog.recipes...); err != nil {
                return
        }
        if value, err = i.Evaluate(prog, params, recipes); err == nil {
                if value != nil {
                        out.Assign(value)
                }
                def := prog.scope.Lookup("@").(*Def)
                if target, err = def.Call(prog.position); err == nil {
                        var strings []string
                        for _, recipe := range recipes {
                                // Avoids calling recipe.Strval() twice, so that it won't be
                                // evaluated more than once.
                                strings = append(strings, recipe.String())
                        }
                        _, _, err = prog.project.UpdateCmdHash(target, strings)
                }
        }
        return
}

func (prog *Program) modify(m *modifier, out *Def) (interpreted string, err error) {
        // TODO: using rules in a different project to implement modifiers, e.g.
        //       [ foo.check-preprequisites ]
        //       [ foo.baaaar ]
        var name string
        if name, err = m.name.Strval(); err != nil {
                return
        } else if f, ok := modifiers[name]; ok {
                var (
                        value = out.Value
                        args []Value
                )
                if args, err = DiscloseAll(m.args...); err != nil {
                        return
                }
                if value, err = f(prog.position, prog, value, args...); err == nil && value !=  nil {
                        out.Assign(value)
                }
        } else if i, _ := dialects[name]; i != nil {
                err = prog.interpret(i, out, m.args)
                interpreted = name // return dialect name
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

func (prog *Program) cd(chdir string, print, exec bool) (err error) {
        var main = prog.globe.Main()
        if trace_workdir {
                fmt.Printf("entering: %v (init: %v)\n", chdir, main.absPath)
        }

        var caller *Program
        if len(execstack) > 1 {
                caller = execstack[1]
        }

        var cd = &cdinfo{ "", chdir, false }
        if cd.workdir, err = os.Getwd(); err == nil {
                if cd.workdir == cd.chdir || context.workdir == cd.chdir {
                        cd.print = false
                } else if err = os.Chdir(cd.chdir); err != nil {
                        return
                } else if cd.print = true; cd.print && exec {
                        if m := prog.getModifier("cd"); m != nil && len(m.args) > 0 {
                                var s string
                                if s, err = m.args[0].Strval(); err != nil {
                                        return
                                } else if s == "-" {
                                        cd.print = false
                                } else /*if s != ""*/ {
                                        cd.print = false
                                }
                        }
                }
                if cd.print && !(print && printcd) { cd.print = false }
                if cd.print && len(printstack) > 0 && printstack[0] == cd.chdir {
                        cd.print = false
                }
                if cd.print && caller != nil {
                        // check the caller program's subcdrs
                        if len(caller.subcdrs) > 0 && caller.subcdrs[0] == cd.chdir {
                                cd.print = false
                        } else {
                                caller.subcdrs = append([]string{cd.chdir}, caller.subcdrs...)
                        }
                }
                if cd.print {
                        fmt.Printf("smart: Entering directory '%s'\n", cd.chdir)
                        printstack = append([]string{cd.chdir}, printstack...)
                }
                prog.auto("CWD", &String{cd.chdir})
                prog.cdinfos = append([]*cdinfo{ cd }, prog.cdinfos...)
        }
        return
}

func (prog *Program) uncd() (err error) {
        for _, cd := range prog.cdinfos {
                if cd.print && false {
                        fmt.Printf("smart:  Leaving directory '%s'\n", cd.chdir)
                }
                if err = os.Chdir(cd.workdir); err != nil {
                        break
                }
        }
        prog.cdinfos = prog.cdinfos[:0] // clear all
        return
}

func (prog *Program) Execute(entry *RuleEntry, args []Value) (result Value, err error) {
        if trace_prepare {
                fmt.Printf("program.Execute: %v (%v) (%v) (%v)\n", entry.target, prog.depends, entry.class, prog.project.absPath)
        }

        var print = prog.getModifier("cd") == nil

        // cd before setting execstack, because cd reads execstack
        // before changes.
        if err = prog.cd(prog.project.absPath, print, true); err != nil { return }

        // Have to set execstack after cd.
        defer setexecstack(setexecstack(execstack.unshift(prog))) // build the call stack
        defer setclosure(setclosure(cloctx.unshift(entry.DeclScope())))////defer setclosure(setclosure(cloctx.append(entry.DeclScope())))

        // uncd after setting execstack to meet the FIFO order of execstack
        defer func() { if err == nil { err = prog.uncd() } } ()

        // set $@ before pre-modifiers.
        switch t := entry.target.(type) {
        case *File: prog.auto("@", t)
        default:
                var s string
                if s, err = entry.target.Strval(); err != nil {
                        return
                } else if file := prog.project.file(s); file == nil {
                        prog.auto("@", entry.target)
                } else {
                        prog.auto("@", file)
                }
        }

        // set up arguments ($1, $2, ..., $9, etc.)
        var argn = 0
        for _, a := range args {
                switch t := a.(type) {
                case *Pair:
                        var s string
                        if s, err = t.Key.Strval(); err != nil { return }
                        prog.auto(s, t.Value)
                default:
                        prog.auto(strconv.Itoa(argn+1), a)
                        if argn < len(prog.params) {
                                prog.auto(prog.params[argn], a)
                        }
                        argn += 1
                }
        }

        // modifier buffer
        var modifyBuf = prog.auto("-", universalnone)
        defer func() { result = modifyBuf.Value }()

        // Split modifiers by '|', if no '|', all goes postModifiers.
        var preModifiers, postModifiers []*modifier
        for i, m := range prog.pipline {
                if m.name == modifierbar {
                        preModifiers = prog.pipline[:i]
                        postModifiers = prog.pipline[i+1:]
                        break
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

        // Pre-modifiers
        var preInterpreted string
        PrePipeline: for _, m := range preModifiers {
                var lang string
                if m.name == modifierbar {
                        continue
                } else if lang, err = prog.modify(m, modifyBuf); err != nil {
                        if p, ok := err.(*breaker); ok && p != nil && p.good {
                                // Discard err and change dialect to avoid
                                // default interpreter being called.
                                err, preInterpreted = nil, "--"
                        }
                        if err != nil {
                                fmt.Fprintf(os.Stdout, "%s: %v\n", m.position, err)
                        }
                        break PrePipeline
                } else if lang != "" && preInterpreted == "" {
                        preInterpreted = lang
                }
        }

        // Expending all dependencies after pre-modifiers.
        var depends []Value
        if depends, err = mergeresult(ExpendAll(prog.depends...)); err != nil {
                return
        }
        for i, depend := range depends {
                switch depend.(type) {
                case *GlobPattern, *RegexpPattern: // break
                case *usinglist: // break
                case *Group: // break
                case *File: // break
                default:
                        var s string
                        if s, err = depend.Strval(); err != nil {
                                return
                        } else if file := prog.project.file(s); file != nil {
                                depends[i] = file
                        }
                }
        }

        // Calculate and prepare depends and files.
        var pc = &preparer{ prog, nil, new(List), prog.stem }
        if err = pc.updateall(depends); err != nil {
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
                prog.auto("<", pc.targets.Elems[0])
                prog.auto("^", pc.targets)
        }
        for _, rec := range prog.subcdrs {
                fmt.Printf("smart:  Leaving directory '%s'\n", rec)
                if len(printstack) > 0 && printstack[0] == rec {
                        printstack = printstack[1:]
                }
        }
        if s := prog.project.absPath; len(printstack) > 0 && printstack[0] == s {
                fmt.Printf("smart:  Leaving directory '%s'\n", s)
                printstack = printstack[1:]
        }
        if len(prog.subcdrs) > 0 {
                prog.subcdrs = prog.subcdrs[:0] // clear subcdrs
        }

        // TODO: define modifiers in a project, e.g.
        // 
        //      some-modifier : - :
        //              smart statments going here...
        //              
        var postInterpreted string
        PostPipeline: for _, m := range postModifiers {
                var lang string
                if m.name == modifierbar {
                        continue
                } else if lang, err = prog.modify(m, modifyBuf); err != nil {
                        if p, ok := err.(*breaker); ok && p != nil && p.good {
                                // Discard err and change dialect to
                                // avoid default interpreter being
                                // called.
                                err, postInterpreted = nil, "--"
                        }
                        if err != nil {
                                fmt.Fprintf(os.Stdout, "%s: %v\n", m.position, err)
                        }
                        break PostPipeline
                } else if lang != "" && postInterpreted == "" {
                        postInterpreted = lang
                }
        }
        if err == nil && preInterpreted == "" && postInterpreted == "" {
                // Using the default statements interpreter.
                if i, ok := dialects[""]; ok && i != nil {
                        err = prog.interpret(i, modifyBuf, nil)
                } else {
                        err = fmt.Errorf("no default dialect")
                }
        }
        return
}

func (prog *Program) AddModifier(position token.Position, operation Value) (err error) {
        switch g := operation.(type) {
        case *Group:
                prog.pipline = append(prog.pipline, &modifier{
                        position, g.Get(0), g.Slice(1),
                })
        case *ModifierBar:
                prog.pipline = append(prog.pipline, &modifier{
                        position, g, nil,
                })
        default:
                err = fmt.Errorf("unknown modifier (%T `%v`)", operation, operation)
                fmt.Fprintf(os.Stderr, "%s: %v\n", prog.position, err)
        }
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
