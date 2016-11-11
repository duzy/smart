//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
//  The implementation of smart/token package is highly referencing to go/token.
//  
package token

import (
        got "go/token"
)

const NoPos Pos = Pos(got.NoPos)

type Position got.Position
type Pos got.Pos
type File got.File
type FileSet got.FileSet
