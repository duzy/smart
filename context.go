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
        "fmt"
        "os"
)

var (
        optionReconfig = false
        optionConfigures = false
        usageConfigures = ``
)

type Context struct {
        workdir string
        prefix  string // FIXME: prefix for distribution
        globe   *Globe
        loader  *loader
}

var context Context

func current() (proj *Project) {
        switch {
        case context.loader != nil: // at load time
                proj = context.loader.project
        case len(execstack) > 0: // at run time
                proj = execstack[0].project
        }
        return
}

func (ctx *Context) run(targets... Value) (result []Value, err error) {
        if ctx.globe.main == nil {
                err = fmt.Errorf("no targets to update `%v`", targets)
                return
        }

        var updated int
        if len(targets) == 0 {
                if entry := ctx.globe.main.DefaultEntry(); entry != nil {
                        if result, err = entry.Execute(entry.Position); err == nil {
                                updated += 1
                        }
                }
        } else {
                defer setclosure(setclosure(cloctx.unshift(ctx.globe.main.scope)))
                for _, target := range targets {
                        var ( entry *RuleEntry; ok bool )
                        if entry, ok = target.(*RuleEntry); !ok || entry == nil {
                                fmt.Fprintf(os.Stderr, "`%v` is not an entry", target)
                                break
                        }

                        var v []Value
                        /*for _, a := range args {
                                v = append(v, MakeString(a))
                        }*/

                        // The the base project scope as execution context. For
                        // example of 'base.test', the entry 'test' can resolve
                        // '&(FOO)', '&(BAR)', etc.
                        if result, err = entry.Execute(entry.Position, v...); err == nil {
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
                if fi, err := os.Stat(filepath.Join(s, ".smart")); err == nil && fi != nil && !vis(s) {
                        break
                }
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

func processCommandOption(flag *Flag, args... Value) (err error) {
        var opt bool
        if opt, err = flag.is('c', "configure"); err != nil { return } else if opt {
                optionConfigures = true; return
        }
        if opt, err = flag.is('r', "reconfigure"); err != nil { return } else if opt {
                optionConfigures, optionReconfig = true, true; return
        }
        err = fmt.Errorf("`%v` unknown command option", flag.Name)
        return
}

// loadwork loads smart files, making it as individual func to avoid being
// abused by loaders.
func (ctx *Context) loadwork() (targets []Value, err error) {
        defer func(l *loader) { ctx.loader = l } (ctx.loader)
        ctx.loader = &loader{
                Context:  ctx,
                fset:     token.NewFileSet(), 
                paths:    []string(globalPaths),
                loaded:   make(map[string]*Project),
                ruleParseFunc: parseRuleClause,
                includeFunc: includespec,
                usefunc:  useProject,
                scope:    ctx.globe.scope,
        }

        var (
                base, _ = os.Getwd()
                rel, _ = filepath.Rel(base, base)
                tmp = joinTmpPath(base, rel)
                sp = filepath.Join(base, ".smart", "modules")

                at = ctx.loader.globe.project(nil, base, rel, tmp, ".", "@")
                as = at.Scope()
        )

        if _, obj := as.Find("SMART"); obj != nil {
                def := obj.(*Def)
                for _, s := range globalPaths {
                        def.Append(MakeString("-search"))
                        def.Append(MakeString(s))
                }
        }

        if _, e := os.Stat(sp); e == nil {
                ctx.loader.AddSearchPaths(sp)
        }

        saveLoadingInfo(ctx.loader, at.Spec(), at.absPath, "")
        linfo := ctx.loader.loads[len(ctx.loader.loads)-1]
        linfo.declares[at.Name()] = &declare{ project: at }

        ctx.loader.globe.scope.ProjectName(nil, at.Name(), at)

        var (
                ab = base
                defCTD, _ = as.Def(at, "CTD", MakeString(tmp))
                defCWD, _ = as.Def(at, "CWD", MakeString(at.absPath))
                defS, _ = as.Def(at, "/", MakeString(at.absPath))
                defD, _ = as.Def(at, ".", universalnone)
        )
        if defCTD == nil { /* ... */ }
        if defCWD == nil { /* ... */ }
        AtLookupLoop: for {
                var s1 = filepath.Join(ab, "@.smart")
                var s2 = filepath.Join(ab, "@")
                if fi, _ := os.Stat(s1); fi != nil {
                        if m := fi.Mode(); m.IsRegular() {
                                defS.Assign(MakeString(ab))
                                defD.Assign(MakeString(ab))
                                if err = ctx.loader.loadFile(s1, nil); err != nil {
                                        return
                                } else {
                                        break AtLookupLoop
                                }
                        } else {
                                fmt.Fprintf(os.Stderr, "@.smart is not a regular")
                        }
                } else if fi, _ = os.Stat(s2); fi != nil {
                        if m := fi.Mode(); m.IsDir() {
                                defS.Assign(MakeString(ab))
                                defD.Assign(MakeString(ab))
                                if err = ctx.loader.loadPath(s2, nil); err != nil {
                                        return
                                } else {
                                        break AtLookupLoop
                                }
                        } else {
                                fmt.Fprintf(os.Stderr, "@ is not a directory")
                        }
                }
                if ab == "/" { break }
                if ab = filepath.Dir(ab); ab == "." { break }
        }

        restoreLoadingInfo(ctx.loader)

        if err = ctx.loader.loadPath(base, nil); err != nil { return }

        text := strings.Join(os.Args[1:], " ")
        for _, target := range ctx.loader.loadText("@", text) {
                switch t := target.(type) {
                case *Flag:
                        if err = processCommandOption(t); err != nil {
                                fmt.Fprintf(os.Stderr, "%s\n", err)
                        }
                case *Pair:
                        switch k := t.Key.(type) {
                        case *Flag:
                                if err = processCommandOption(k, t.Value); err != nil {
                                        fmt.Fprintf(os.Stderr, "%s\n", err)
                                }
                        default:
                                fmt.Fprintf(os.Stderr, "unknown target `%v`\n", t)
                        }
                case *Bareword:
                        if entry, err := ctx.loader.project.resolveEntry(t.string); err != nil {
                                fmt.Fprintf(os.Stderr, "%s\n", err)
                        } else if entry == nil {
                                fmt.Fprintf(os.Stderr, "no such entry `%s`\n", t)
                        } else {
                                targets = append(targets, entry)
                        }
                case *delegate:
                        if s, err := t.Strval(); err != nil {
                                fmt.Fprintf(os.Stderr, "%s\n", err)
                        } else if entry, err := ctx.loader.project.resolveEntry(s); err != nil {
                                fmt.Fprintf(os.Stderr, "%s\n", err)
                        } else if entry == nil {
                                fmt.Fprintf(os.Stderr, "no such entry `%s` (via `%v`)\n", s, t)
                        } else {
                                targets = append(targets, entry)
                        }
                default:
                        fmt.Fprintf(os.Stderr, "unknown target `%s` (of %T)\n", target, target)
                }
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

        defer func(globe *Globe) { context.globe = globe } (context.globe)
        context.globe = NewGlobe("smart")
        if err := init_configuration(packagePaths); err != nil {
                report(err)
        } else if works, err := context.loadwork(); err != nil {
                report(err)
        } else if optionConfigures {
                report(do_configuration())
        } else if result, err := context.run(works...); err != nil {
                report(err)
        } else if result != nil {
                for _, v := range result {
                        var s string
                        if s, err = v.Strval(); err != nil {
                                fmt.Printf("%s [%s]", v, err)
                        } else {
                                fmt.Printf("%s", s)
                        }
                }
                fmt.Printf("\n")
        }
}
