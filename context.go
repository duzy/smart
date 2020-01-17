//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/token"
	"path/filepath"
        "strings"
        "bufio"
        "time"
        "fmt"
        "os"
        "io"
)

var (
        optionHelp = false
        optionClean = false
        optionConfigure = false
        optionReconfig = false
        optionAlwaysBuildPlugins = false
        optionVerbose = false
        optionVerboseUsing = false
        optionVerboseParsing = false
        optionVerboseLoading = false
        optionVerboseImport = false
        optionVerboseChecks = false
        optionBenchImport = false
        optionBenchSlow = false
        optionBenchBuiltin = false
        optionPrintStack = false
)
const (
        optionTraceParsing = false
        optionTraceTraversal = false
        optionTraceTraversalNestIndent = true
        optionTraceExecutor = false
        optionTraceExec = false
        optionTraceEntering = optionTraceTraversal && false

        optionExecuteUseRulesRecursively = false
        optionExecuteUseRuleMultiTimes = false
        optionExecuteUseLightly = false
        optionExecuteUseBases = false

        optionSearchImportedFiles = false // time consuming

        optionNoDeprecatedFeatures = true

        // Return error if wildcard files not found.
        optionWildcardMissingError = false

        optionSaveGrepSourceName = false
)

type Context struct {
        workdir string
        preargs string // pre-loading command arguments (evaluated on the first project declaration)
        prefix  string // FIXME: prefix for distribution
        globe   *Globe
        loader  *loader
        goals   *Def
}

var context Context

func current() (proj *Project) {
        switch {
        case context.loader != nil: // at load time
                proj = context.loader.project
        case len(execstack) > 0: // at runtime
                proj = execstack[0].project
        }
        return
}

func mostDerived() (proj *Project) {
        // Check cloctx first, then execstack and context.loader
        if len(cloctx) > 0 && cloctx[0].project != nil {
                return cloctx[0].project
        }

        if l := len(execstack); l == 1 {
                proj = execstack[0].project
        } else if l > 1 {
                var p = execstack[0].project
                for i, prog := range execstack[1:] {
                        // If the next (n=i+1) project is derived from 'p'...
                        if n := i+1; n < l {
                                if p == prog.project { continue } else {
                                        var next = execstack[n].project
                                        if next.isa(p) { p = next; continue }
                                }
                        }
                        break
                }
                proj = p
        } else {
                proj = current()
        }
        return
}

func (ctx *Context) run() (result []Value, err error) {
        var main = ctx.globe.main
        if main == nil {
                err = fmt.Errorf("no targets to update `%v`", ctx.goals)
                return
        }

        var goals []Value
        for _, goal := range merge(ctx.goals.value) {
                switch t := goal.(type) {
                case *None: // just ignore
                case *Bareword:
                        if entry, err := main.resolveEntry(t.string); err != nil {
                                fmt.Fprintf(stderr, "%s\n", err)
                        } else if entry == nil {
                                fmt.Fprintf(stderr, "no such entry `%s`\n", t)
                        } else {
                                goals = append(goals, entry)
                        }
                case *delegate:
                        if s, err := t.Strval(); err != nil {
                                fmt.Fprintf(stderr, "%s\n", err)
                        } else if entry, err := main.resolveEntry(s); err != nil {
                                fmt.Fprintf(stderr, "%s\n", err)
                        } else if entry == nil {
                                fmt.Fprintf(stderr, "no such entry `%s` (via `%v`)\n", s, t)
                        } else {
                                goals = append(goals, entry)
                        }
                default:
                        fmt.Fprintf(stderr, "unknown target `%v` (%T)\n", goal, goal)
                }
        }

        var updated int
        if len(goals) == 0 {
                if entry := main.DefaultEntry(); entry != nil {
                        if result, err = entry.Execute(entry.position); err == nil {
                                updated += 1
                        }
                }
        } else {
                defer setclosure(setclosure(cloctx.unshift(main.scope)))
                for _, goal := range goals {
                        var ( entry *RuleEntry; ok bool )
                        if entry, ok = goal.(*RuleEntry); !ok || entry == nil {
                                fmt.Fprintf(stderr, "`%v` is not an entry", goal)
                                break
                        }

                        var v []Value
                        /*for _, a := range args {
                                v = append(v, &String{a})
                        }*/

                        // The the base project scope as execution context. For
                        // example of 'base.test', the entry 'test' can resolve
                        // '&(FOO)', '&(BAR)', etc.
                        if result, err = entry.Execute(entry.position, v...); err == nil {
                                updated += 1
                        } else {
                                break
                        }
                }
        }
        return
}

func walkSmartBaseDirs(cwd string, vis func(string)bool) (s string) {
        s = cwd
        for s != "" {
                file := stat(".smart", "", s)
                if file != nil && file.info.IsDir() && !vis(s) { break }
                if up := filepath.Dir(s); up == s {
                        break
                } else {
                        s = up
                }
        }
        if s == "" {
                s = cwd
        }
        return
}

// baseTmpPath is the base tmp path initialized only once.
var baseTmpPath string

func joinTmpPath(base, rel string) string {
        if baseTmpPath == "" {
                var s = walkSmartBaseDirs(base, func(d string) bool {
                        return false // return the first found
                })
                if s == "" {
                        // FIXME: Windows system temporary path.
                        s = filepath.Join("/", "tmp")
                }
                baseTmpPath = s
        }
        if s := filepath.Dir(rel); s != "" {
                if strings.HasSuffix(base, s) {
                        // In case like '/foo/bar/a/b/c/x'+'a/b/c/x', we set
                        // rel to 'x' to produce 'foo/bar/.smart/tmp/a/b/c/x'.
                        rel = filepath.Base(rel)
                } else if t, _ := filepath.Rel(baseTmpPath, base); strings.HasPrefix(t, ".smart"+PathSep) {
                        // In case like '/foo/bar/.smart/a/b/x'+'a/e/f/x', we set
                        // base to '/foo/bar/.smart' to produce 'foo/bar/.smart/tmp/a/e/f/x'.
                        v1 := strings.Split(t, PathSep)
                        v2 := strings.Split(s, PathSep)
                        for i := len(v1)-1; i >= 0; i -= 1 {
                                if v1[i] == v2[0] {
                                        base = filepath.Join(v1[i-1:]...)
                                        break
                                }
                        }
                }
        }
        if s, err := filepath.Rel(baseTmpPath, filepath.Join(base, rel)); err == nil {
                rel = s
        }
        if s := ".smart"+PathSep; strings.HasPrefix(rel, s) { // .smart/
                rel = strings.TrimPrefix(rel, s)
                if s = "modules"+PathSep; strings.HasPrefix(rel, s) { // modules/
                        rel = strings.TrimPrefix(rel, s)
                }
        }
        rel = strings.Replace(rel, "..", "_", -1)
        if strings.HasPrefix(rel, "tmp"+PathSep) {
                return filepath.Join(baseTmpPath, ".smart", rel)
        }
        return filepath.Join(baseTmpPath, ".smart", "tmp", rel)
}

func processCommandOption(args... Value) (err error) {
        return
}

func (ctx *Context) loadCommandArguments(text string) (err error) {
        var args = ctx.loader.loadText("@", text)
        if args, err = parseFlags(args, []string{
                "h,help",
                "b,build-plugins",
                "n,bench-import",
                "v,verbose",
                "i,verbose-import",
                "k,verbose-checks",
                "r,reconfigure",
                "c,configure",
        }, func(ru rune, v Value) {
                switch ru {
                case 'h': optionHelp = trueVal(v, true)
                case 'b': optionAlwaysBuildPlugins = trueVal(v, true)
                case 'n': optionBenchImport = trueVal(v, true)
                case 'v': optionVerbose = trueVal(v, true)
                case 'i': optionVerboseImport = trueVal(v, true)
                case 'k': optionVerboseChecks = trueVal(v, true)
                case 'c': optionConfigure = trueVal(v, true)
                case 'r':
                        optionReconfig = trueVal(v, true)
                        optionConfigure = optionReconfig
                }
        }); err != nil { return }
        for _, target := range args {
                switch t := target.(type) {
                case *Pair:
                        switch k := t.Key.(type) {
                        case *Bareword:
                                if proj := ctx.loader.project; proj != nil {
                                        def, alt := ctx.loader.def(k.string)
                                        if def == nil && alt != nil {
                                                def = alt.(*Def)
                                        }
                                        def.set(DefDefault, t.Value)
                                }
                        default:
                                fmt.Fprintf(stderr, "unknown target `%v` (%v)\n", t, ctx.loader.project)
                        }
                case *Bareword, *delegate:
                        ctx.goals.append(t)
                default:
                        fmt.Fprintf(stderr, "unknown target `%s` (of %T)\n", target, target)
                }
        }
        return
}

// loadwork loads smart files, making it as individual func to avoid being
// abused by loaders.
func (ctx *Context) loadwork() (err error) {
        var pos Position
        defer func(l *loader) { ctx.loader = l } (ctx.loader)
        ctx.loader = &loader{
                Context:  ctx,
                fset:     token.NewFileSet(), 
                paths:    []string(globalPaths),
                loaded:   make(map[string]*Project),
                scope:    ctx.globe.scope,
        }
        ctx.goals = &Def{
                knownobject{
                        trivialobject{
                                scope: ctx.globe.scope,
                                owner: nil,
                        }, ":goals:",
                },
                DefDefault,
                &None{trivial{pos}},
        }

        if optionVerbose || optionBenchImport {
                defer func(t time.Time) {
                        var d = time.Now().Sub(t)
                        fmt.Fprintf(stderr, "smart: Goals %v (%s)\n", ctx.goals, d)
                } (time.Now())
        }

        var (
                base, _ = os.Getwd()
                rel, _ = filepath.Rel(base, base)
                tmp = joinTmpPath(base, rel)
                sp = filepath.Join(base, ".smart", "modules")

                at = ctx.loader.globe.project(nil, base, rel, tmp, ".", "@")
                as = at.Scope()
        )

        if def := as.FindDef("SMART"); def != nil {
                def.set(DefSimple, nil)
                for _, s := range globalPaths {
                        def.append(&String{trivial{pos},"-search"})
                        def.append(&String{trivial{pos},s})
                }
        }

        if _, e := os.Stat(sp); e == nil {
                ctx.loader.AddSearchPaths(sp)
        }

        saveLoadingInfo(ctx.loader, at.Spec(), at.absPath, "")
        linfo := ctx.loader.loads[len(ctx.loader.loads)-1]
        linfo.declares[at.Name()] = &declare{ project: at }

        ctx.loader.globe.scope.ProjectName(nil, at.Name(), at)

        /*
        var (
                ab = base
                defCTD, _ = as.define(at, "CTD", &String{pos,tmp})
                defCWD, _ = as.define(at, "CWD", &String{pos,at.absPath})
                defS, _ = as.define(at, "/", &String{pos,at.absPath})
                defD, _ = as.define(at, ".", &None{trivial{pos}})
        )
AtLookupLoop:
        for {
                var s1 = filepath.Join(ab, "@.smart")
                var s2 = filepath.Join(ab, "@")
                if fi, _ := os.Stat(s1); fi != nil {
                        if m := fi.Mode(); m.IsRegular() {
                                defS.set(DefExpand, &String{ab})
                                defD.set(DefExpand, &String{ab})
                                if optionVerboseImport { fmt.Fprintf(stderr, "┌→%s\n", s1) }
                                if err = ctx.loader.loadFile(s1, nil); err != nil {
                                        return
                                } else {
                                        break AtLookupLoop
                                }
                        } else {
                                fmt.Fprintf(stderr, "@.smart is not a regular")
                        }
                } else if fi, _ = os.Stat(s2); fi != nil {
                        if m := fi.Mode(); m.IsDir() {
                                defS.set(DefExpand, &String{ab})
                                defD.set(DefExpand, &String{ab})
                                if optionVerboseImport { fmt.Fprintf(stderr, "┌→%s\n", s2) }
                                if err = ctx.loader.loadPath(s2, nil); err != nil {
                                        return
                                } else {
                                        break AtLookupLoop
                                }
                        } else {
                                fmt.Fprintf(stderr, "@ is not a directory")
                        }
                }
                if ab == "/" { break }
                if ab = filepath.Dir(ab); ab == "." { break }
        } */

        restoreLoadingInfo(ctx.loader)

        var args []string
        var commandText string
        for _, a := range os.Args[1:] {
                switch a {
                case "-b", "-build-plugins": // TODO: -build=plugins
                        optionAlwaysBuildPlugins = true
                case "-bi", "-bench-import": // TODO: -bench=import
                        optionBenchImport = true
                case "-bb", "-bench-builtins": // TODO: -bench=builtins
                        optionBenchBuiltin = true
                case "-v", "-verbose":
                        optionVerbose = true
                case "-vp", "-verbose-parsing": // TODO: -verbose=parsing
                        optionVerboseParsing = true
                case "-vl", "-verbose-loading": // TODO: -verbose=loading
                        optionVerboseLoading = true
                case "-vu", "-verbose-using": // TODO: -verbose=using
                        optionVerboseUsing = true
                case "-vi", "-verbose-import": // TODO: -verbose=import
                        optionVerboseImport = true
                case "-vc", "-verbose-checks":  // TODO: -verbose=checks
                        optionVerboseChecks = true
                default:
                        args = append(args, a)
                }
        }
        for i, s := range args {
                if s == "-" {
                        ctx.preargs = strings.Join(args[:i], " ")
                        commandText = strings.Join(args[i:], " ")
                        break
                }
        }
        if ctx.preargs == "" && commandText == "" && len(args) > 0 {
                ctx.preargs = strings.Join(args, " ")
        }

        defer func(t time.Time) {
                var name string
                if ctx.loader.project != nil {
                        name = ctx.loader.project.name
                }
                var d = time.Now().Sub(t)
                if optionVerboseImport {
                        fmt.Fprintf(stderr, "└·%s … (%s)\n", name, d)
                } else if d > 5000*time.Millisecond {
                        fmt.Fprintf(stderr, "smart: %s … (%s)\n", name, d)
                }
        } (time.Now())
        if optionVerboseImport { fmt.Fprintf(stderr, "┌→%s\n", base) }

        if err = ctx.loader.loadPath(base, nil); err != nil { return }
        if ctx.loader.globe.main == nil {
                fmt.Fprintf(stderr, "no projects loaded\n")
                return
        }

        if commandText != "" {
                err = ctx.loadCommandArguments(commandText)
        }
        return
}

func CommandLine() {
        if s, err := os.Getwd(); err != nil { return } else {
                context.workdir = s
        }

        var modulesPaths, packagePaths searchlist
        walkSmartBaseDirs(context.workdir, func(s string) bool {
                if baseTmpPath == "" { baseTmpPath = s }
                packagePaths = append(packagePaths, filepath.Join(s, ".smart", "packages"))
                modulesPaths = append(modulesPaths, filepath.Join(s, ".smart", "modules"))
                return true
        })
        packagePaths = append(packagePaths, filepath.Join(context.prefix, "user", "lib", "smart", "packages"))
        modulesPaths = append(modulesPaths, filepath.Join(context.prefix, "user", "lib", "smart", "modules"))

        // make sure that .smart dirs have higher priority.
        globalPaths = append(modulesPaths, globalPaths...)
        for _, s := range modulesPaths {
                searchFile := filepath.Join(s, ".search")
                if fi, _ := os.Stat(searchFile); fi == nil {
                        continue
                }
                file, err := os.Open(searchFile)
                if err != nil { report(err); return }
                defer file.Close()
                r := bufio.NewReader(file)
                for err == nil {
                        var line string
                        if line, err = r.ReadString('\n'); err != nil {
                                if err != io.EOF { report(err) }
                                break
                        } else {
                                line = strings.TrimSpace(line)
                        }
                        if strings.HasPrefix(line, "#") {
                                continue
                        }
                        line = filepath.Clean(filepath.Join(s, line))
                        if fi, err := os.Stat(line); err == nil && fi.IsDir() {
                                globalPaths = append(globalPaths, line)
                        }
                }
        }

        //loadGrepCache()
        defer func(globe *Globe) {
                saveGrepCache()
                context.globe = globe
        } (context.globe)

        context.globe = NewGlobe("smart")
        if err := init_configuration(packagePaths); err != nil {
                report(err)
        } else if err = context.loadwork(); err != nil {
                report(err)
        } else if optionHelp {
                do_helpscreen()
        } else if numUpdatedPlugins > 0 { // see buildPlugin
                fmt.Fprintf(stderr, "smart: Plugin updated, please relaunch.\n")
                //os.Exit(0)
        } else if optionConfigure {
                report(do_configuration())
        } else if result, err := context.run(); err != nil {
                defer printLeavingDirectory()
                var brks, errs = breakers(err)
                for _, e := range brks {
                        switch e.what {
                        default: report(e)
                        case breakDone, breakCase:
                                // just relax
                        }
                }
                for _, e := range errs { report(e) }
        } else if result != nil {
                for _, v := range result {
                        var s string
                        if s, err = v.Strval(); err != nil {
                                fmt.Fprintf(stderr, "%s [%s]", v, err)
                        } else {
                                fmt.Fprintf(stderr, "%s", s)
                        }
                }
                fmt.Fprintf(stderr, "\n")
                printLeavingDirectory()
        }
}
