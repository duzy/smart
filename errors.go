//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "errors"
    "fmt"
)

var (
    ErrorIllImport = errors.New("illegal import spec")
    ErrorIllJson   = errors.New("illegal json format")
    ErrorIllName   = errors.New("illegal name")
    ErrorIllXml    = errors.New("illegal xml format")
    ErrorNilExec   = errors.New("execute nil program")
    ErrorNoEntry   = errors.New("no matched rule")
    ErrorUpdated   = errors.New("target updated")
)

type (
    failure struct {
        position Position
        metainfo interface{}
    }

    FailedAssertion string
    Unreachable string

    targetNotFoundError struct { project *Project; target string }
    pathNotFoundError   struct { project *Project; path *Path }
    fileNotFoundError   struct { project *Project; file *File }
)

func assert(cond bool, s string, a ...interface{}) {
    if !cond { panic(FailedAssertion(fmt.Sprintf(s, a...))) }
}

func unreachable(a ...interface{}) {
    panic(Unreachable(fmt.Sprint(a...)))
}

func fail(pos Position, s string, a ...interface{}) {
    panic(failure{pos,fmt.Sprintf(s,a...)})
}

func (s FailedAssertion) Error() string { return string(s) }
func (s Unreachable) Error() string { return string(s) }

func (e targetNotFoundError) Error() string {
    return fmt.Sprintf("%s: %v target not found", e.project.name, e.target)
}

func (e pathNotFoundError) Error() string {
    return fmt.Sprintf("%s: %v path not found", e.project.name, e.path)
}

func (e fileNotFoundError) Error() string {
    if s := e.file.fullname(); e.file.name == s { // e.project.name
        return fmt.Sprintf(`"%v" not found`, e.file.name)
    } else {
        return fmt.Sprintf(`"%v" not found at %s`, e.file.name, trimPromptString(s))
    }
}
