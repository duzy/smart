//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package interpreter

import (
        "github.com/duzy/smart/token"
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/parser"
        "github.com/duzy/smart/runtime"
        "github.com/duzy/smart/values"
        "path/filepath"
        "strings"
        "errors"
        "flag"
        "fmt"
        "os"
)

type searchlist []string

func (sl *searchlist) String() string {
        return fmt.Sprint(*sl)
}

func (sl *searchlist) Set(value string) error {
        *sl = append(*sl, strings.Split(value, ",")...)
        return nil
}

var globalPaths searchlist

func init() {
        flag.Var(&globalPaths, "search", "comma-separated list of search paths")
}

type declare struct {
        project *types.Project
        backscope *types.Scope
}

type loadinfo struct {
        specPath, absPath, baseName string
        loader *types.Project
        scope *types.Scope
        declares map[string]*declare // all project declares in the loaded dir
}

type Interpreter struct {
        *runtime.Context
        pc       *parser.Context
        fset     *token.FileSet
        paths    searchlist
        loads    []*loadinfo
        loaded   map[string]*types.Project
        project  *types.Project // the current project
        scope    *types.Scope   // the current scope
}

type parseContext struct {
        *Interpreter
}

// Create and initialize a new interpreter.
func New() (interpreter *Interpreter) {
        interpreter = &Interpreter{
                Context:  runtime.NewContext("interpreter"),
                fset:     token.NewFileSet(), 
                paths:    []string(globalPaths),
                loaded:   make(map[string]*types.Project),
        }
        interpreter.scope = interpreter.Globe().Scope()
        interpreter.pc = parser.NewContext(&parseContext{ interpreter })
        for s, _ := range types.GetBuiltins() {
                interpreter.pc.Builtin(s, nil)
        }
        for _, s := range runtime.GetBuiltinNames() {
                interpreter.pc.Builtin(s, nil)
        }
        return
}

func (i *Interpreter) AddSearchPaths(paths... string) (err error) {
        for _, s := range paths {
                if s, err = filepath.Abs(s); err != nil {
                        break
                }
                if fi, _ := os.Stat(s); fi != nil && fi.IsDir() {
                        i.paths = append(i.paths, s)
                } else {
                        return errors.New(fmt.Sprintf("path '%s' is not dir", s))
                }
        }
        return nil
}

func AddSearchPaths(paths... string) (err error) {
        for _, s := range paths {
                if s, err = filepath.Abs(s); err != nil {
                        break
                }
                if fi, _ := os.Stat(s); fi != nil && fi.IsDir() {
                        globalPaths = append(globalPaths, s)
                } else {
                        return errors.New(fmt.Sprintf("path '%s' is not dir", s))
                }
        }
        return nil
}

func CommandLine() {
        defer func() {
		if e := recover(); e != nil {
			// resume same panic if it's not a Failure
			if failure, ok := e.(*runtime.Failure); !ok {
				panic(e)
			} else {
                                fmt.Printf("%s\n", failure)
                                os.Exit(-1) // exit abnormally
                        }
		}
        }()

        flag.Parse()

        var (
                base, _ = os.Getwd()
                rel, _ = filepath.Rel(base, base)

                i = New()
                at = i.Globe().NewProject(base, rel, ".", "@")
                as = at.Scope()

                targets []string
        )

        saveLoadingInfo(i, at.Spec(), at.AbsPath(), at.Name())

        linfo := i.loads[len(i.loads)-1]
        linfo.declares[at.Name()] = &declare{ project: at }

        for _, a := range flag.Args() {
                if i := strings.Index(a, "="); 0 <= i {
                        var (
                                name = strings.TrimSpace(a[0:i])
                                v = strings.TrimSpace(a[1+1:])
                        )
                        if name == "" {
                                fmt.Printf("ERROR: bad argument '%v'\n", a)
                                return
                        }
                        as.InsertNewDef(at, name, values.String(v))
                } else {
                        targets = append(targets, a)
                }
        }

        i.Globe().Scope().InsertNewProjectName(nil, at.Name(), at)

        var (
                s1 = filepath.Join(base, "@.smart")
                s2 = filepath.Join(base, "@")
        )
        if fi, err := os.Stat(s1); err == nil {
                if m := fi.Mode(); m.IsRegular() {
                        if err = i.Load(s1, nil); err != nil {
                                fmt.Printf("%v\n", err)
                                return
                        }
                } else {
                        fmt.Fprintf(os.Stderr, "@.smart is not a regular")
                }
        } else if fi, err = os.Stat(s2); err == nil {
                if m := fi.Mode(); m.IsDir() {
                        if err = i.LoadDir(s2, nil); err != nil {
                                fmt.Printf("%v\n", err)
                                return
                        }
                } else {
                        fmt.Fprintf(os.Stderr, "@ is not a directory")
                }
        }

        if err := i.Load("build.smart", nil); err != nil {
                fmt.Printf("%v\n", err)
                return
        } else if err = i.Run(targets...); err != nil {
                fmt.Printf("%v\n", err)
                return
        }
}
