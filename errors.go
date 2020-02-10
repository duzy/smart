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
        "strings"
        "errors"
        "fmt"
)

var (
        ErrorIllImport  = errors.New("illegal import spec")
        ErrorIllJson    = errors.New("illegal json format")
        ErrorIllName    = errors.New("illegal name")
        ErrorIllXml     = errors.New("illegal xml format")
        ErrorNilExec    = errors.New("execute nil program")
        ErrorNoEntry    = errors.New("no matched rule")
        ErrorUpdated    = errors.New("target updated")
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
        return fmt.Sprintf("%s: `%v` file not found (%s)", e.project.name, e.file.name, e.file.fullname())
}

func extractTargetNotFoundError(err error) (res *targetNotFoundError) {
        if err != nil {
                switch t := err.(type) {
                case *scanner.Error:
                        for _, e := range t.Errs {
                                res = extractTargetNotFoundError(e)
                                if res != nil { break }
                        }
                case targetNotFoundError:
                        res = &t
                }
        }
        return
}

func extractPathNotFoundError(err error) (res *pathNotFoundError) {
        if err != nil {
                switch t := err.(type) {
                case *scanner.Error:
                        for _, e := range t.Errs {
                                res = extractPathNotFoundError(e)
                                if res != nil { break }
                        }
                case pathNotFoundError:
                        res = &t
                }
        }
        return
}

func extractFileNotFoundError(err error) (res *fileNotFoundError) {
        if err != nil {
                switch t := err.(type) {
                case *scanner.Error:
                        for _, e := range t.Errs {
                                res = extractFileNotFoundError(e)
                                if res != nil { break }
                        }
                case fileNotFoundError:
                        res = &t
                }
        }
        return
}

func report(err error) error {
        if err != nil {
                if false { debug.PrintStack() }
                fmt.Fprintf(stderr, "%s\n", strings.TrimSpace(err.Error()))
        }
        return err
}

func errorf(pos Position, s string, args... interface{}) (err error) {
        return scanner.Errorf(token.Position(pos), s, args...)
}

func wrap(pos Position, errs ...error) (err error) {
        return scanner.WrapErrors(token.Position(pos), errs...)
}
