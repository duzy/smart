//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package token

import (
        got "go/token"
)

/*
  Struct Position:
	Filename string  -- filename, if any
	Offset   int     -- offset, starting at 0
	Line     int     -- line number, starting at 1
	Column   int     -- column number, starting at 1 (byte count)
*/
type Position struct {
        got.Position
}

const NoPos Pos = Pos(got.NoPos)

type Pos got.Pos

func (p Pos) IsValid() bool {
        return got.Pos(p).IsValid() 
}
        
type File struct {
        *got.File
}

func (f *File) Offset(p Pos) int {
        return f.File.Offset(got.Pos(p))
}

func (f *File) Line(p Pos) int {
	return f.File.Line(got.Pos(p))
}

func (f *File) Pos(offset int) Pos {
        return Pos(f.File.Pos(offset))
}

func (f *File) PositionFor(p Pos, adjusted bool) (pos Position) {
        return Position{ f.File.PositionFor(got.Pos(p), adjusted) }
}

func (f *File) Position(p Pos) (pos Position) {
        return Position{ f.File.Position(got.Pos(p)) }
}

type FileSet struct {
        *got.FileSet
}

// NewFileSet creates a new file set.
func NewFileSet() *FileSet {
	return &FileSet{ got.NewFileSet() }
}

func (s *FileSet) AddFile(filename string, base, size int) *File {
        return &File{ s.FileSet.AddFile(filename, base, size) }
}

func (s *FileSet) Iterate(f func(*File) bool) {
        s.FileSet.Iterate(func(file *got.File) bool {
                return f(&File{ file })
        })
}
