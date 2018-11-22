//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/scanner"
        "extbit.io/smart/token"
	"path/filepath"
        "strings"
        "flag"
        "fmt"
        "os"
)

type Context struct {
        workdir string
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

func (ctx *Context) run(targets... Value) (err error) {
        var (
                result []Value
                updated int
        )

        if ctx.globe.main == nil {
                Fail("no targets to update")
        }

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
        if err == nil && result != nil && len(result) > 0 {
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
        rel = strings.Replace(rel, "..", "_", -1)
        return filepath.Join(baseTmpPath, ".smart", ".tmp", rel)
}

// loadwork loads smart files, making it as individual func to avoid being
// abused by loaders.
func (ctx *Context) loadwork() (targets []Value) {
        defer func(l *loader) { ctx.loader = l } (ctx.loader)

        ctx.loader = &loader{
                Context:  ctx,
                fset:     token.NewFileSet(), 
                paths:    []string(globalPaths),
                loaded:   make(map[string]*Project),
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

        //absDir, baseName := filepath.Split(at.absPath)
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
                var (
                        s1 = filepath.Join(ab, "@.smart")
                        s2 = filepath.Join(ab, "@")
                )
                if fi, err := os.Stat(s1); err == nil {
                        if m := fi.Mode(); m.IsRegular() {
                                defS.Assign(MakeString(ab))
                                defD.Assign(MakeString(ab))
                                if err = ctx.loader.loadFile(s1, nil); err != nil {
                                        scanner.PrintError(os.Stderr, err)
                                        return
                                } else {
                                        break AtLookupLoop
                                }
                        } else {
                                fmt.Fprintf(os.Stderr, "@.smart is not a regular")
                        }
                } else if fi, err = os.Stat(s2); err == nil {
                        if m := fi.Mode(); m.IsDir() {
                                defS.Assign(MakeString(ab))
                                defD.Assign(MakeString(ab))
                                if err = ctx.loader.loadPath(s2, nil); err != nil {
                                        scanner.PrintError(os.Stderr, err)
                                        return
                                } else {
                                        break AtLookupLoop
                                }
                        } else {
                                fmt.Fprintf(os.Stderr, "@ is not a directory")
                        }
                }
                if ab == "/" {
                        break
                }
                if ab = filepath.Dir(ab); ab == "." {
                        break
                }
        }

        restoreLoadingInfo(ctx.loader)

        if err := ctx.loader.loadPath(base, nil); err != nil {
                scanner.PrintError(os.Stderr, err)
                return
        }

        text := strings.Join(flag.Args(), " ")
        for _, target := range ctx.loader.loadText("@", text) {
                switch t := target.(type) {
                case *Bareword:
                        if entry, err := ctx.loader.project.resolveEntry(t.Value); err != nil {
                                fmt.Fprintf(os.Stderr, "%s", err)
                        } else if entry == nil {
                                fmt.Fprintf(os.Stderr, "no such entry `%s`", t)
                        } else {
                                targets = append(targets, entry)
                        }
                case *delegate:
                        if s, err := t.Strval(); err != nil {
                                fmt.Fprintf(os.Stderr, "%s", err)
                        } else if entry, err := ctx.loader.project.resolveEntry(s); err != nil {
                                fmt.Fprintf(os.Stderr, "%s", err)
                        } else if entry == nil {
                                fmt.Fprintf(os.Stderr, "no such entry `%s` (via `%v`)", s, t)
                        } else {
                                targets = append(targets, entry)
                        }
                default:
                        fmt.Fprintf(os.Stderr, "unknown target `%s` (of %T)", target, target)
                }
        }
        return
}

func CommandLine() {
        defer func() {
		if e := recover(); e != nil {
			// resume same panic if it's not a Failure
			if failure, ok := e.(*Failure); !ok {
				panic(e)
			} else {
                                scanner.PrintError(os.Stderr, failure)
                        }
		}
        }()

        if s, err := os.Getwd(); err == nil {
                context.workdir = s
        } else {
                return
        }

        var smartDirs searchlist
        walkSmartBaseDirs(context.workdir, func(s string) bool {
                if baseTmpPath == "" { baseTmpPath = s }
                smartDirs = append(smartDirs, filepath.Join(s, ".smart"))
                return true
        })

        if !flag.Parsed() {
                flag.Parse()
        }

        // make sure that .smart dirs have higher priority.
        globalPaths = append(smartDirs, globalPaths...)

        context.globe = NewGlobe("smart")
        if err := context.run(context.loadwork()...); err != nil {
                scanner.PrintError(os.Stderr, err)
                return
        }
}
