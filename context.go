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
        "io/ioutil"
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
        optionTraceLaunch = false
        optionTraceParsing = false
        optionTraceTraversal = false
        optionTraceTraversalNestIndent = true
        optionTraceExecutor = false
        optionTraceExec = false
        optionTraceEntering = optionTraceTraversal && false

        // Return error if wildcard files not found.
        optionWildcardMissingError = false

        optionSaveGrepSourceName = false
)

type Context struct {
        workdir string
        prefix  string // FIXME: prefix for distribution
        globe   *Globe
        goals   *Def
        pairs []*Pair
        loader  *loader
}

var context Context

func current() (proj *Project) {
        if len(cloctx) > 0 && cloctx[0].project != nil {
                proj = cloctx[0].project
        } else if context.loader != nil { // for load time
                proj = context.loader.project
        }
        return
}

func (ctx *Context) run() (result []Value, err error) {
        if optionTraceLaunch { defer un(trace(t_launch, "Context.run")) }

        var main = ctx.globe.main
        if main == nil {
                err = fmt.Errorf("no targets to update `%v`", ctx.goals)
                return
        }

        defer setclosure(setclosure(cloctx.unshift(main.scope)))

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
                file := stat(Position{}, ".smart", "", s)
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

// loadwork loads smart files, making it as individual func to avoid being
// abused by loaders.
func (ctx *Context) loadwork() (err error) {
        if optionTraceLaunch { defer un(trace(t_launch, "Context.loadwork")) }

        var pos Position // FIXME: find a useful position
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
                DefDefault, &None{trivial{pos}},
        }

        if optionVerbose || optionBenchImport {
                defer func(t time.Time) {
                        var d = time.Now().Sub(t)
                        fmt.Fprintf(stderr, "smart: Goals %v (%s)\n", ctx.goals, d)
                } (time.Now())
        }

        var (
                base, _ = os.Getwd()
                sp = filepath.Join(base, ".smart", "modules")
        )
        if _, e := os.Stat(sp); e == nil {
                ctx.loader.AddSearchPaths(sp)
        }

        var args []Value
        if text := strings.Join(os.Args[1:], " "); text == "" {
                // Relax!
        } else if args = ctx.loader.loadText("@", text); err != nil {
                // ...
        } else if args, err = parseFlags(args, []string{
                "h,help",
                "b,build-plugins",
                "n,bench-import",
                "e,bench-builtins",
                "v,verbose",
                "i,verbose-import",
                "c,verbose-checks",
                "l,verbose-loading",
                "p,verbose-parsing",
                "u,verbose-using",
                "r,reconfigure",
                "g,configure",
                "d,debug",
        }, func(ru rune, v Value) {
                switch ru {
                case 'h': optionHelp = trueVal(v, true)
                case 'b': optionAlwaysBuildPlugins = trueVal(v, true)
                case 'd': optionPrintStack = trueVal(v, true)
                case 'n': optionBenchImport = trueVal(v, true)
                case 'e': optionBenchBuiltin = trueVal(v, true)
                case 'v': optionVerbose = trueVal(v, true)
                case 'i': optionVerboseImport = trueVal(v, true)
                case 'c': optionVerboseChecks = trueVal(v, true)
                case 'p': optionVerboseParsing = trueVal(v, true)
                case 'l': optionVerboseLoading = trueVal(v, true)
                case 'u': optionVerboseUsing = trueVal(v, true)
                case 'g': optionConfigure = trueVal(v, true)
                case 'r':
                        optionReconfig = trueVal(v, true)
                        optionConfigure = optionReconfig
                }
        }); err != nil { return }
        for _, target := range args {
                switch t := target.(type) {
                case *Pair: ctx.pairs = append(ctx.pairs, t)
                default:    ctx.goals.append(t)
                }
        }

        defer func(t time.Time) {
                var d = time.Now().Sub(t)
                if optionVerboseImport {
                        var name string
                        if p := ctx.loader.project; p != nil { name = p.name }
                        fmt.Fprintf(stderr, "└·%s … (%s)\n", name, d)
                } else if d > 5000*time.Millisecond {
                        fmt.Fprintf(stderr, "smart: Long load time: %s !\n", d)
                }
        } (time.Now())
        if optionVerboseImport { fmt.Fprintf(stderr, "┌→%s\n", base) }

        if err = ctx.loader.loadPath(base, nil); err != nil { return }
        if ctx.loader.globe.main == nil {
                fmt.Fprintf(stderr, "no projects loaded\n")
        }
        return
}

func CommandLine() {
        if s, err := os.Getwd(); err != nil { return } else {
                context.workdir = s
        }

        if optionTraceLaunch { defer un(trace(t_launch, "CommandLine")) }
        if optionEnableBenchmarks {
                var w *bufio.Writer
                var d = filepath.Join(context.workdir, "benchmarks")
                if err := os.MkdirAll(d, os.FileMode(0777)); err != nil {
                        fmt.Fprintf(stderr, "MkdirAll: %s\n", err)
                        return
                } else if f, err := ioutil.TempFile(d, "*.log"); err != nil {
                        fmt.Fprintf(stderr, "TempFile: %s\n", err)
                        return
                } else {
                        w = bufio.NewWriter(f)
                        benchmark.start = time.Now()
                        benchmark.spot = benchmark.start
                        defer func(t time.Time) {
                                benchspot_report(w)
                                w.WriteString("--------\n")
                                benchmark.spent = time.Now().Sub(t)
                                benchmark.summary(w)
                                benchmark.report(w, 0, nil)
                                w.Flush()
                                f.Close()
                        } (benchmark.spot)
                }
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

                var brks, errs = extractBreakers(err)
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
