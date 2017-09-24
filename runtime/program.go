//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        "path/filepath"
        "strconv"
        "strings"
        "errors"
        "fmt"
        "os"
)

type dependPatternUnfit struct {
}

func (*dependPatternUnfit) Error() string { return "pattern unfit" }

type workinfo struct {
        dir, backdir string
        print bool
        num int
}

var workstack []*workinfo

func enterWorkdir(dir string, print bool) (wi *workinfo) {
        if wd, _ := os.Getwd(); dir != filepath.Clean(wd) {
                if nws := len(workstack); nws > 0 {
                        if p := workstack[nws-1]; p.dir == dir {
                                //fmt.Printf("renter: %s (%v, %v)\n", p.dir, p.num, print)
                                fmt.Printf("renter: %s (%v %s)\n", wd, p.num, p.backdir)
                                if p.num <= 0 {
                                        workstack, print = workstack[0:nws-1], false
                                } else {
                                        p.backdir = wd
                                        p.num += 1
                                }
                        }
                }
                if print {
                        fmt.Printf("smart: Entering directory '%s'\n", dir)
                }
                if wi != nil {
                        return
                }
                if err := os.Chdir(dir); err == nil {
                        if wi == nil {
                                wi = &workinfo{
                                        dir: dir, backdir: wd, 
                                        print: print, 
                                        num: 1,
                                }
                        }
                        workstack = append(workstack, wi)
                } else {
                        // TODO: error...
                }
        }
        return
}

func leaveWorkdir(wi *workinfo) {
        if /*nws := len(workstack); nws > 0 &&*/ wi != nil {
                //assert(wi == workstack[nws-1])
                if wi.print {
                        fmt.Printf("smart:  Leaving directory '%s'\n", wi.dir)
                }
                if err := os.Chdir(wi.backdir); err == nil {
                        // ...
                } else {
                        // TODO: error...
                }
                wi.num -= 1
        }
}

// Program (TODO: moving program into `types` package)
type Program struct {
        context *Context
        project  *types.Project
        scope   *types.Scope
        params  []string // named parameters
        depends []types.Value // *types.RuleEntry, *types.Barefile
        recipes []types.Value
        pipline []types.Value
}

func (prog *Program) Params() []string { return prog.params }
func (prog *Program) Project() *types.Project { return prog.project }
func (prog *Program) Depends() []types.Value { return prog.depends }
func (prog *Program) Recipes() []types.Value { return prog.recipes }
func (prog *Program) Pipeline() []types.Value { return prog.pipline }
func (prog *Program) Scope() *types.Scope { return prog.scope }

func (prog *Program) auto(name string, value interface{}) (auto *types.Def) {
        var alt types.Object
        if auto, alt = prog.scope.InsertDef(prog.project, name, values.Make(value)); alt != nil {
                //fmt.Printf("auto: %p %T %v\n", prog, sym, sym.Name())
                var found = false
                if auto, found = alt.(*types.Def); found {
                        auto.Assign(values.Make(value))
                } else {
                        Fail("Name '%v' already taken, not auto (%T)", name, alt)
                }
        }
        return
}

func (prog *Program) discloseRecipes(context *types.Scope) (recipes []types.Value, err error) {
        for _, recipe := range prog.recipes {
                if v, e := types.Disclose(context, recipe); e != nil {
                        return nil, e
                } else if v != nil {
                        recipe = v
                }
                recipes = append(recipes, recipe) // types.EvalElems
        }
        return
}

func (prog *Program) interpret(context *types.Scope, i interpreter, out *types.Def, params []types.Value) (err error) {
        var (
                recipes []types.Value
                target, value types.Value
        )
        if recipes, err = prog.discloseRecipes(context); err != nil {
                return
        }
        if value, err = i.evaluate(prog, context, params, recipes); err == nil {
                if value != nil {
                        out.Assign(value)
                }
                def := prog.scope.Lookup("@").(*types.Def)
                if target, err = def.Call(); err == nil {
                        _, _, err = prog.project.UpdateCmdHash(target, recipes)
                }
        }
        return
}

func (prog *Program) modify(context *types.Scope, g *types.Group, out *types.Def) (err error) {
        // TODO: using rules in a different project to implement modifiers, e.g.
        //       [ foo.check-preprequisites ]
        //       [ foo.baaaar ]
        var name = g.Get(0).Strval()
        if name == "cd" {
                // does nothing
        } else if f, ok := modifiers[name]; ok {
                var value = out.Value
                if value, err = f(prog, context, value, g.Slice(1)...); err == nil && value !=  nil {
                        out.Assign(value)
                }
        } else if i, _ := interpreters[name]; i != nil {
                err = prog.interpret(context, i, out, g.Slice(1))
        } else {
                err = errors.New(fmt.Sprintf("no modifier or dialect '%s'", name))
        }
        return
}

func (prog *Program) prepare(context *types.Scope, entry *types.RuleEntry, dependList *types.List) (err error) {
        for _, depend := range prog.depends {
                if v, e := types.Disclose(context, depend); e != nil {
                        return e
                } else if v != nil {
                        depend = v
                }
                for _, d := range types.JoinReveal(depend) { // types.Join(depend)
                        //fmt.Printf("Program.prepare: %T `%v' -> %T `%v'\n", depend, depend, d, d)
                        if err = prog.prepareDepend(prog.project, context, entry, false, d, nil, dependList); err != nil {
                                return
                        }
                }
        }
        return
}

func (prog *Program) prepareDepend(project *types.Project, context *types.Scope, entry *types.RuleEntry, isFile bool, depend types.Value, args []types.Value, dependList *types.List) (err error) {
        //fmt.Printf("Program.prepareDepend: %s: %T `%v'\n", entry.Name(), depend, depend)
        switch d := depend.(type) {
        case *types.Argumented:
                fmt.Printf("Program.prepareDepend: argumented: %d %v\n", len(d.Args), d.Args)
                err = prog.prepareDepend(project, context, entry, isFile, d.Value, d.Args, dependList)
        case *types.Bareword:
                err = prog.prepareHandleName(project, context, entry, isFile, d.Strval(), depend, args, dependList)
        case *types.Barefile, *types.File, *types.Path:
                err = prog.prepareHandleName(project, context, entry, true, d.Strval(), depend, args, dependList)
        case *types.ProjectName:
                if depent := d.Project().DefaultEntry(); depent != nil {
                        err = prog.prepareDepend(project, context, entry, isFile, depent, args, dependList)
                }
        case *types.PercentPattern:
                if stem := entry.Stem(); stem != "" {
                        err = prog.prepareHandleName(project, context, entry, isFile, d.MakeString(stem), depend, args, dependList)
                } else {
                        err = errors.New(fmt.Sprintf("Empty stem (%s, dependency %v)", entry, d))
                }
        case *types.RuleEntry:
                err = prog.prepareHandleEntry(project, context, entry, isFile, d, args, dependList)
        case *types.None:
                // ...
        default:
                err = errors.New(fmt.Sprintf("Unknown depend `%T' (%v) (by `%s').", d, d, entry.Name()))
        }
        return
}

func (prog *Program) prepareHandleEntry(project *types.Project, context *types.Scope, entry *types.RuleEntry, isFile bool, depend *types.RuleEntry, args []types.Value, dependList *types.List) (err error) {
        var (
                isDependPatternUnfit = false
                isDependUpdated = false
        )
        if depend.Programs() == nil {
                switch depend.Class() {
                case types.FileRuleEntry:
                        //fmt.Printf("file: %s (%v)\n", project.Name(), d)
                        err = prog.prepareHandleName(project, context, entry, true, depend.Strval(), depend, args, dependList)
                case types.PatternFileRuleEntry:
                        // A pattern entry without program can't
                        // help to update the file.
                        err = errors.New(fmt.Sprintf("No rule to make file `%v'", depend))
                default:
                        err = errors.New(fmt.Sprintf("%v: `%v' requies update actions (%v)\n", entry, depend, depend.Class()))
                }
                return
        }
        ProgramsLoop: for _, deprog := range depend.Programs() {
                if deprog == prog {
                        return errors.New(fmt.Sprintf("Depends on itself (%v).", entry))
                }
                if err = prog.prepareExecuteDeprog(project, context, entry, isFile, depend, deprog, args, dependList); err == nil {
                        isDependUpdated = true
                        break ProgramsLoop
                } else if _, ok := err.(*dependPatternUnfit); ok {
                        //fmt.Printf("%s: %v (pattern not fit)\n", entry.Name(), depend)
                        isDependPatternUnfit = true
                        continue ProgramsLoop
                } else if true {
                        var s = err.Error()
                        if strings.HasPrefix(s, "Updating ") {
                                s = "->" + strings.TrimPrefix(s, "Updating ")
                        } else {
                                s = ", " + s
                        }
                        err = errors.New(fmt.Sprintf("Updating %v%v", entry.Name(), s))
                } else {
                        err = errors.New(fmt.Sprintf("Updating `%v' failed (%v)", entry.Name(), prog.project.AbsPath()))
                }
        }
        if isDependPatternUnfit && !isDependUpdated {
                if args != nil {
                        err = errors.New(fmt.Sprintf("No rule for `%v' (required by `%v%v')", depend.Name(), entry.Name(), args))
                } else {
                        err = errors.New(fmt.Sprintf("No rule for `%v' (required by `%v')", depend.Name(), entry.Name()))
                }
        }
        return
}

func (prog *Program) prepareHandleName(project *types.Project, context *types.Scope, entry *types.RuleEntry, isFile bool, name string, depend types.Value, args []types.Value, dependList *types.List) (err error) {
        FindEntry: if _, obj := project.Scope().Find(name); obj != nil {
                if depent, _ := obj.(*types.RuleEntry); entry != nil {
                        return prog.prepareHandleEntry(project, context, entry, isFile, depent, args, dependList)
                }
        }

        // Find patterns.
        if pss := project.FindPatterns(name); pss != nil {
                return prog.prepareHandlePatterns(project, context, entry, isFile, name, pss, depend, args, dependList)
        }

        // Find entry in the context project.
        if proj := context.FindProject(); proj != nil {
                if proj != project {
                        //fmt.Printf("prepareHandleName: %s -> %s\n", project.Name(), proj.Name())
                        project = proj; goto FindEntry
                } else if isFile || proj.IsFile(name) {
                        // Search file in context's project.
                        return prog.searchFile(proj, context, entry, name, depend, args, dependList)
                }
        }

        return errors.New(fmt.Sprintf("No such rule `%v' (required by `%s' via `%v').", name, entry.Name(), depend))
}

func (prog *Program) prepareHandlePatterns(project *types.Project, context *types.Scope, entry *types.RuleEntry, isFile bool, name string, pss []*types.PatternStem, depend types.Value, args []types.Value, dependList *types.List) (err error) {
        if isFile {
                return prog.prepareHandleFilePatterns(project, context, entry, isFile, name, pss, depend, args, dependList)
        }

        var num int
        // Update all found PatternStems if it's not a file.
        for _, ps := range pss {
                if depent, e := ps.MakeConcreteEntry(); e != nil {
                        err = e; return
                } else if depent != nil {
                        //isFile = project.IsFile(depent.Strval())
                        if err = prog.prepareHandleEntry(project, context, entry, isFile, depent, args, dependList); err == nil {
                                num += 1
                        } else {
                                return
                        }
                }
        }
        return
}

func (prog *Program) prepareHandleFilePatterns(project *types.Project, context *types.Scope, entry *types.RuleEntry, isFile bool, name string, pss []*types.PatternStem, depend types.Value, args []types.Value, dependList *types.List) (err error) {
        //fmt.Printf("%v: %v\n", name, pss)
        for _, ps := range pss {
                depent, e := ps.MakeConcreteEntry()
                if e != nil {
                        err = e; return
                } else if depent == nil {
                        err = errors.New(fmt.Sprintf("No concrete entry for `%v' (%v)", ps.Patent, name))
                        return
                }
                if false {
                        fmt.Printf("%v: %v -> %v -> %v\n", name, ps.Patent.RuleEntry.Strval(), depent.Strval(), depent.Depends())
                }

                // Only execute existing file depends.
                LoopPrograms: for _, depentProg := range depent.Programs() {
                        for _, depentDep := range depentProg.Depends() {
                                switch dd := depentDep.(type) {
                                case *types.PercentPattern:
                                        ddName := dd.MakeString(ps.Stem)
                                        if project.IsFile(ddName) {
                                                if fi, e := os.Stat(ddName); fi == nil || e != nil {
                                                        continue LoopPrograms // Discard the program if file depend is not statible.
                                                }
                                        }
                                }
                        }
                        if false {
                                fmt.Printf("%v: %v -> %v -> %v\n", name, ps.Patent.RuleEntry.Strval(), depent.Strval(), depentProg.Depends())
                        }
                        if err = prog.prepareExecuteDeprog(project, context, entry, true, depent, depentProg, args, dependList); err == nil {
                                //v := dependList.Elems[len(dependList.Elems)-1]
                                //fmt.Printf("%s: %v -> %v (executed)\n", name, depentProg.Depends(), v.Strval())
                                return
                        }
                }
        }
        return
}

func (prog *Program) prepareExecuteDeprog(project *types.Project, context *types.Scope, entry *types.RuleEntry, isFileDepend bool, depend *types.RuleEntry, deprog types.Program, args []types.Value, dependList *types.List) (err error) {
        scope := context
        if entry.Project() != depend.Project() {
                scope = entry.Project().Scope()
        }

        var res types.Value
        if res, err = deprog.Execute(scope, depend, args); err == nil {
                dd, _ := deprog.Scope().Lookup("@").(*types.Def).Call()
                if isFileDepend {
                        dependList.Append(values.File(dd, dd.Strval()))
                } else {
                        switch depend.Class() {
                        case types.FileRuleEntry, types.PatternFileRuleEntry:
                                dependList.Append(values.File(dd, dd.Strval()))
                        default:
                                if res != nil && res.Type() != types.NoneType {
                                        dependList.Append(res); return
                                } else {
                                        dependList.Append(depend)
                                }
                        }
                }
                if res != nil && res.Type() != types.NoneType {
                        for _, elem := range types.Join(res) {
                                switch elem.(type) {
                                case *types.File: dependList.Append(elem)
                                }
                        }
                }
        }
        return
}

func (prog *Program) searchFile(project *types.Project, context *types.Scope, entry *types.RuleEntry, file string, depend types.Value, args []types.Value, dependList *types.List) (err error) {
        // Search file.
        var fv = values.File(depend, file)
        if fv = project.SearchFile(context, fv); fv.Info == nil {
                fv = prog.project.SearchFile(context, fv)
        }
        if fv.Info != nil {
                dependList.Append(fv)
        } else if depend.Type() == types.PatternType {
                return new(dependPatternUnfit)
        } else {
                if true /*verbose*/ {
                        fmt.Fprintf(os.Stderr, "%v: No such file `%v'\n", entry.Name(), file)
                }
                return errors.New(fmt.Sprintf("No such file `%v' (required by `%v')", fv, entry.Name()))
        }
        return
}

func (prog *Program) Getwd(context *types.Scope) string {
        for _, m := range prog.pipline {
                if g, ok := m.(*types.Group); ok && g != nil {
                        if n := len(g.Elems); n > 0 && g.Elems[0].Strval() == "cd" {
                                var s string
                                if n > 1 {
                                        if v, e := types.Disclose(context, g.Elems[1]); e != nil {
                                                // TODO: error...
                                        } else if v != nil {
                                                s = filepath.Clean(v.Strval())
                                                if s == "-" { s = "" }
                                        }
                                }
                                return s
                        }
                }
        }
        return filepath.Clean(prog.project.AbsPath())
}

func (prog *Program) Execute(context *types.Scope, entry *types.RuleEntry, args []types.Value) (result types.Value, err error) {
        /*if entry.Name() == "lib.a" {
                //fmt.Printf("Program.Execute: %v: %v %v\n", entry, args, prog.depends)
                //fmt.Printf("Program.Execute: %v: %v %v\n", entry, args, prog.pipline)
                //fmt.Printf("Program.Execute: %v: %v\n", entry.Name(), context)
                fmt.Printf("Program.Execute: %v: %v\n", entry.Name(), args)
        }*/
        if workdir := prog.Getwd(context); workdir != "" {
                var pcd = entry.Class() != types.UseRuleEntry
                defer leaveWorkdir(enterWorkdir(workdir, pcd))
        }
        
        for i, a := range args {
                // TODO: handle with Pair, map 'key => value' into
                // parameters.
                prog.auto(strconv.Itoa(i+1), a)
                if i < len(prog.params) {
                        name := prog.params[i]
                        prog.auto(name, a)
                }
        }

        if s := entry.Name(); prog.project.IsFile(s) {
                file := prog.project.SearchFile(context, values.File(entry, s))
                prog.auto("@", file)
        } else {
                prog.auto("@", entry)
        }

        dependList := values.List()
        prog.auto("^", dependList)

        // Calculate and prepare depends and files.
        if err = prog.prepare(context, entry, dependList); err != nil {
                return
        }

        var out = prog.auto("-", values.None)
        defer func() { result = out.Value }()

        // TODO: define modifiers in a project, e.g.
        // 
        //      some-modifier : - :
        //              smart statments going here...
        //              
        if len(prog.pipline) == 0 {
                // Using the default statements interpreter.
                if i, _ := interpreters[``]; i == nil {
                        err = errors.New("no default dialect")
                } else {
                        err = prog.interpret(context, i, out, nil)
                }
                return
        }
        LoopPipeline: for _, v := range prog.pipline {
                switch op := v.(type) {
                case *types.Group:
                        if err = prog.modify(context, op, out); err != nil {
                                if p, ok := err.(*breaker); ok {
                                        if p.okay {
                                                err = nil
                                        } else {
                                                fmt.Printf("%s, required by '%s' (from %v)\n", p.message, entry.Name(), prog.project.RelPath())
                                        }
                                }
                                break LoopPipeline
                        }
                case *types.Bareword:
                        if i, _ := interpreters[op.Strval()]; i == nil {
                                err = errors.New(fmt.Sprintf("no dialect '%s', required by '%s'", op, entry.Name()))
                                break LoopPipeline
                        } else if err = prog.interpret(context, i, out, nil); err != nil {
                                //fmt.Printf("interpret: %v\n", err)
                                break LoopPipeline
                        }
                default:
                        err = errors.New(fmt.Sprintf("unsupported modifier '%s'", v))
                        break LoopPipeline
                }
        }
        return
}

func (prog *Program) SetModifiers(modifiers... types.Value) (err error) {
        prog.pipline = modifiers
        return
}

func (context *Context) NewProgram(project *types.Project, params []string, scope *types.Scope, depends []types.Value, recipes... types.Value) *Program {
        return &Program{
                context:     context,
                project:     project,
                scope:       scope,
                params:      params,
                depends:     depends, // *types.RuleEntry, *types.Barefile
                recipes:     recipes,
        }
}
