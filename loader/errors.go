//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package loader

import (
        //"github.com/duzy/smart/runtime"
        "errors"
)

var (
        ErrorIllImport  = errors.New("illegal import spec")
        ErrorIllName    = errors.New("illegal name")
        ErrorUnreachable = errors.New("unreachable")
        ErrorAssertion   = errors.New("assertion failed")
)

func assert(p bool) {
	if !p {
		panic(ErrorAssertion)
	}
}

func unreachable() {
	panic(ErrorUnreachable)
}
