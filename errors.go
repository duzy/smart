//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/scanner"
        "runtime"
        "errors"
        "fmt"
        "os"
)

var (
        ErrorIllImport  = errors.New("illegal import spec")
        ErrorIllName    = errors.New("illegal name")

        ErrorUpdated   = errors.New("target updated")
        ErrorNilExec   = errors.New("execute nil program")
        ErrorNoEntry   = errors.New("no matched rule")
        ErrorIllXml    = errors.New("illegal xml format")
        ErrorIllJson   = errors.New("illegal json format")
)

type AssertionFailed string

func (s AssertionFailed) Error() string { return string(s) }

func assert(cond bool, s string, a ...interface{}) {
	if !cond { panic(AssertionFailed(fmt.Sprintf(s, a...))) }
}

type UnreachablePoint string

func (s UnreachablePoint) Error() string { return string(s) }

func unreachable(a ...interface{}) {
	panic(UnreachablePoint(fmt.Sprint(a...)))
}

type Returner struct {
        Value Value
}

func (p *Returner) Error() string {
        return "evaluation returned"
}

type RuntimeError struct {
        stack string
}

func (p *RuntimeError) Error() string {
        return p.stack
}

func GoFault(e interface{}) *RuntimeError {
        if _, ok := e.(runtime.Error); ok {
                s := fmt.Sprintf("%s\n\n%s", e, ParseStack(false))
                return &RuntimeError{ s }
        }
        return nil
}

func ParseStack(all bool) (s string) {
        // Alternative: debug.PrintStack
        // 1. runtime/stack.go
        // 2. runtime/debug/stack.go
        ba := make([]byte, 1<<16)
        ss := runtime.Stack(ba, all)
        s = string(ba[:ss])
        // TODO: trim off some sys frames
        return
}

// patternPrepareError indicates an error occurred in preparing a pattern.
type patternPrepareError error
type targetNotFoundError struct { target string }
type pathNotFoundError struct { path *Path }
type fileNotFoundError struct { file *File }

func (e targetNotFoundError) Error() string {
        return fmt.Sprintf("`%v` target not found", e.target) 
}

func (e pathNotFoundError) Error() string {
        return fmt.Sprintf("`%v` path not found", e.path)
}

func (e fileNotFoundError) Error() string {
        return fmt.Sprintf("`%v` file not found (%v)", e.file.name, e.file)
}

func report(err error) error {
        if err != nil {
                scanner.PrintError(os.Stderr, err)
        }
        return err
}
