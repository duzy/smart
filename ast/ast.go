//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package ast

import (
        "github.com/duzy/smart/token"
)

type Node interface {
        Pos() token.Pos
        End() token.Pos
}

