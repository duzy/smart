//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package interpreter

import (
        "errors"
)

var (
        ErrorIllImport  = errors.New("illegal import spec")
        ErrorNoModule   = errors.New("missing import module")
        ErrorSearchPath = errors.New("bad search path")
        ErrorIllName    = errors.New("illegal name")
        ErrorNotModuleScope = errors.New("not in a module scope")
)

func assert(p bool) {
	if !p {
		panic("assertion failed")
	}
}

func unreachable() {
	panic("unreachable")
}
