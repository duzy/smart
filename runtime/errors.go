//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        //"github.com/duzy/smart/token"
        "errors"
        "fmt"
)

var (
        ErrorUpdated   = errors.New("target updated")
        ErrorNilExec   = errors.New("execute nil program")
        ErrorNoEntry   = errors.New("no matched rule")
        ErrorIllXml    = errors.New("illegal xml format")
        ErrorIllJson   = errors.New("illegal json format")
)

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
