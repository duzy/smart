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
        ErrorIllName    = errors.New("illegal name")
)

func assert(p bool) {
	if !p {
		panic("assertion failed")
	}
}

func unreachable() {
	panic("unreachable")
}
