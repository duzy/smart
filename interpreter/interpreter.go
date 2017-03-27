//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package interpreter

import (
        "github.com/duzy/smart/token"
        //"github.com/duzy/smart/values"
        "github.com/duzy/smart/runtime"
        "path/filepath"
        "fmt"
        "os"
)

type Interpreter struct {
        *runtime.Context
        fset     *token.FileSet
        loads    []*loadInfo
        paths    []string
}

type loadInfo struct {
        dir, file string
}

// Create and initialize a new interpreter.
func New() *Interpreter {
        return &Interpreter{
                Context:  runtime.NewContext("interpreter"),
                fset:     token.NewFileSet(), 
        }
}

func (i *Interpreter) AddSearchPaths(paths... string) (err error) {
        for _, s := range paths {
                if s, err = filepath.Abs(s); err != nil {
                        break
                }
                if fi, _ := os.Stat(s); fi != nil && fi.IsDir() {
                        i.paths = append(i.paths, s)
                } else {
                        return ErrorSearchPath 
                }
        }
        return nil
}

func CommandLine() {
        i := New()
        if err := i.Load("build.smart", nil); err != nil {
                fmt.Printf("%v\n", err)
                return
        } else if err = i.Run(); err != nil {
                fmt.Printf("%v\n", err)
                return
        }
}
