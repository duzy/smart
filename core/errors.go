//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "errors"
        "fmt"
        "runtime"
)

var (
        ErrorIllImport  = errors.New("illegal import spec")
        ErrorIllName    = errors.New("illegal name")
        ErrorUnreachable = errors.New("unreachable")
        ErrorAssertion   = errors.New("assertion failed")

        ErrorUpdated   = errors.New("target updated")
        ErrorNilExec   = errors.New("execute nil program")
        ErrorNoEntry   = errors.New("no matched rule")
        ErrorIllXml    = errors.New("illegal xml format")
        ErrorIllJson   = errors.New("illegal json format")
)

func assert(p bool) {
	if !p {
		panic(ErrorAssertion)
	}
}

func precondition(cond bool, msg string) {
	if !cond {
		panic("parser internal error: " + msg)
	}
}

func unreachable() {
	panic(ErrorUnreachable)
}

type Returner struct {
        Value Value
}

func (p *Returner) Error() string {
        return "evaluation returned"
}

type Failure struct {
        msg string
}

func (f *Failure) Error() string { return f.msg }

func Fail(s string, a... interface{}) {
        if len(a) > 0 {
                s = fmt.Sprintf(s, a...)
        }
        panic(&Failure{s})
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
        return fmt.Sprintf("unknown target `%v`", e.target) 
}

func (e pathNotFoundError) Error() string {
        return fmt.Sprintf("unknown path `%v`", e.path)
}

func (e fileNotFoundError) Error() string {
        return fmt.Sprintf("unknown file `%v` (%v)", e.file.Name, e.file)
}
