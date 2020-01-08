//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/scanner"
        "extbit.io/smart/token"
        "runtime/debug"
        "runtime"
        "errors"
        "fmt"
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
        Values []Value
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

type patternCompareError struct { error }
type patternTraverseError struct {
        position Position
        error
}

type targetNotFoundError struct { project *Project; target string }
type pathNotFoundError struct { project *Project; path *Path }
type fileNotFoundError struct { project *Project; file *File }

func (e targetNotFoundError) Error() string {
        if false { debug.PrintStack() }
        return fmt.Sprintf("%s: `%v` target not found", e.project.name, e.target)
}

func (e pathNotFoundError) Error() string {
        if false { debug.PrintStack() }
        return fmt.Sprintf("%s: `%v` path not found", e.project.name, e.path)
}

func (e fileNotFoundError) Error() string {
        if false { debug.PrintStack() }
        return fmt.Sprintf("%s: `%v` file not found (as `%s`)", e.project.name, e.file.name, e.file.fullname())
}

func report(err error) error {
        if err != nil {
                if false { debug.PrintStack() }
                scanner.PrintError(stderr, err)
        }
        return err
}

func errorf(pos Position, s string, args... interface{}) (err error) {
        return scanner.Errorf(token.Position(pos), s, args...)
}

func wrap(pos Position, errs ...error) (err error) {
        return scanner.WrapErrors(token.Position(pos), errs...)
}
