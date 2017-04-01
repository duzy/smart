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
        "path/filepath"
        "errors"
        "fmt"
        "os"
)

var globalPaths []string

type Interpreter struct {
        *runtime.Context
        pc       *parser.Context
        fset     *token.FileSet
        loads    []*loadInfo
        paths    []string

}

type loadInfo struct {
        dir, file string
}

// Create and initialize a new interpreter.
func New() *Interpreter {
        pc := parser.NewContext()
        for s, _ := range types.GetBuiltins() {
                pc.Builtin(s, nil)
        }
        for _, s := range runtime.GetBuiltinNames() {
                pc.Builtin(s, nil)
        }
        for _, s := range runtime.GetDialectNames() {
                pc.Dialect(s, nil)
        }
        for _, s := range runtime.GetModifierNames() {
                pc.Modifier(s, nil)
        }
        return &Interpreter{
                Context: runtime.NewContext("interpreter"),
                fset:    token.NewFileSet(), 
                paths:   globalPaths,
                pc:      pc,
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
                        }
		}
        }()
        
        i := New()
        if err := i.Load("build.smart", nil); err != nil {
                fmt.Printf("%v\n", err)
                return
        } else if err = i.Run(); err != nil {
                fmt.Printf("%v\n", err)
                return
        }
}
