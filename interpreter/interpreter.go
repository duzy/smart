//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package interpreter

import (
        "github.com/duzy/smart/token"
        //"github.com/duzy/smart/types"
        //"github.com/duzy/smart/values"
        "github.com/duzy/smart/runtime"
        //"errors"
        //"fmt"
)

type Interpreter struct {
        runtime.Context
        fset     *token.FileSet
        loading  []*loadingInfo
        paths    []string
}

type loadingInfo struct {
        dir, file string
}

// Create and initialize a new interpreter.
func New() *Interpreter {
        return &Interpreter{
                Context:  runtime.MakeContext("interpreter"),
                fset:     token.NewFileSet(), 
        }
}

func (i *Interpreter) AddSearchPaths(paths... string) {
        i.paths = append(i.paths, paths...)
}

func (i *Interpreter) Run(targets... string) (err error) {
        var updated = 0
        if len(targets) == 0 {
                if entry := i.GetDefaultEntry(); entry != nil {
                        if err = i.RunEntry(entry); err == nil {
                                updated += 1
                        }
                }
        } else {
                for _, target := range targets {
                        if err = i.RunEntryByName(target); err == nil {
                                updated += 1
                        } else {
                                break
                        }
                }
        }
        //fmt.Printf("updated %v targets\n", updated)
        //return errors.New("TODO: run entry rules of projects")
        return
}
