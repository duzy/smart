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
        globe   *Globe
        workdir string
}

func NewContext(name string) *Context {
        var (
                workdir, _ = os.Getwd()
                context = &Context{
                        globe:    NewGlobe(name),
                        workdir:  workdir,
                }
        )
        return context
}

func (ctx *Context) run(targets... Value) (err error) {
        var (
                result []Value
                updated int
                mm = ctx.globe.main
        )

        if mm == nil {
                Fail("no targets to update")
        }

        if len(targets) == 0 {
                if entry := mm.DefaultEntry(); entry != nil {
                        if result, err = entry.Execute(entry.Position); err == nil {
                                updated += 1
                        }
                }
        } else {
                for _, target := range targets {
                        var ( entry *RuleEntry; ok bool )
                        if entry, ok = target.(*RuleEntry); !ok || entry == nil {
                                fmt.Fprintf(os.Stderr, "`%v` is not an entry", target)
                                break
                        }

                        closure := closurecontext{ mm.scope }

                        /*if names := strings.Split(target, "->"); len(names) > 1 {
                                for _, s := range names[0:len(names)-1] {
                                        var _, obj = m.Scope().Find(s)
                                        switch t := obj.(type) {
                                        case *ProjectName:
                                                m = t.NamedProject()
                                        case nil:
                                                fmt.Printf("'%s' is not defined in %v", s, m.Scope())
                                                return
                                        default:
                                                fmt.Printf("object '%s' is not project (%T)", s, t)
                                                return
                                        }
                                        if m == nil {
                                                fmt.Printf("project '%s' not imported %v", s)
                                                return
                                        }
                                        closure = append(closure, m.Scope())
                                }
                                target = names[len(names)-1]
                        }*/

                        defer setclosure(setclosure(append(closure, Closure...)))

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

// loadwork loads smart files, making it as individual func to avoid being
// abused by loaders.
func loadwork(ctx *Context) (targets []Value) {
        l := &loader{
                Context:  ctx,
                fset:     token.NewFileSet(), 
                paths:    []string(globalPaths),
                loaded:   make(map[string]*Project),
                scope:    ctx.globe.scope,
        }

        var (
                base, _ = os.Getwd()
                rel, _ = filepath.Rel(base, base)
                sp = filepath.Join(base, ".smart", "modules")

                at = l.globe.NewProject(nil, base, rel, ".", "@")
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
                l.AddSearchPaths(sp)
        }

        //absDir, baseName := filepath.Split(at.AbsPath())
        saveLoadingInfo(l, at.Spec(), at.AbsPath(), "")
        linfo := l.loads[len(l.loads)-1]
        linfo.declares[at.Name()] = &declare{ project: at }

        l.globe.scope.ProjectName(nil, at.Name(), at)

        var (
                ab = base
                defS, _ = as.Def(at, "/", MakeString(at.AbsPath()))
                defD, _ = as.Def(at, ".", UniversalNone)
        )
        AtLookupLoop: for {
                var (
                        s1 = filepath.Join(ab, "@.smart")
                        s2 = filepath.Join(ab, "@")
                )
                if fi, err := os.Stat(s1); err == nil {
                        if m := fi.Mode(); m.IsRegular() {
                                defS.Assign(MakeString(ab))
                                defD.Assign(MakeString(ab))
                                if err = l.loadFile(s1, nil); err != nil {
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
                                if err = l.loadPath(s2, nil); err != nil {
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

        restoreLoadingInfo(l)

        if err := l.loadPath(base, nil); err != nil {
                scanner.PrintError(os.Stderr, err)
                return
        }

        text := strings.Join(flag.Args(), " ")
        for _, target := range l.loadText("@", text) {
                switch t := target.(type) {
                case *Bareword:
                        if entry, err := l.project.resolveEntry(t.Value); err != nil {
                                fmt.Fprintf(os.Stderr, "%s", err)
                        } else if entry == nil {
                                fmt.Fprintf(os.Stderr, "no such entry `%s`", t)
                        } else {
                                targets = append(targets, entry)
                        }
                case *delegate:
                        if s, err := t.Strval(); err != nil {
                                fmt.Fprintf(os.Stderr, "%s", err)
                        } else if entry, err := l.project.resolveEntry(s); err != nil {
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

        if !flag.Parsed() {
                flag.Parse()
        }

        ctx := NewContext("smart")
        if err := ctx.run(loadwork(ctx)...); err != nil {
                scanner.PrintError(os.Stderr, err)
                return
        }
}
