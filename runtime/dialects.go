//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        "fmt"
)

type interpreter interface {
        dialect() string
        evaluate(recipe types.Value) types.Value
}

type dialectTrivial struct {
}

func (t *dialectTrivial) evaluate(recipe types.Value) types.Value {
        fmt.Printf("trivial: %v\n", recipe)
        return values.None
}

var trivialDialect = new(dialectTrivial)

type dialectShell struct {
}

func (s *dialectShell) evaluate(recipe types.Value) (result types.Value) {
        fmt.Printf("shell: %v\n", recipe)
        return
}

type dialectXml struct {
}

func (t *dialectXml) evaluate(recipe types.Value) (result types.Value) {
        fmt.Printf("xml: %v\n", recipe)
        return
}

func (*dialectTrivial) dialect() string { return "trivial" }
func (*dialectShell) dialect() string { return "shell" }
func (*dialectXml) dialect() string { return "xml" }
