//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	golang "go/token"
	"strconv"
	// "fmt"
)

/*
  Struct Position:
	Filename string  -- filename, if any
	Offset   int     -- offset, starting at 0
	Line     int     -- line number, starting at 1
	Column   int     -- column number, starting at 1 (byte count)
*/
type Position struct { golang.Position }
func (p *Position) _valid() bool { return p.Filename != "" && p.Line > 0 }
func (p *Position) IsValid() bool { return p._valid() && p.Column > 0 && p.Offset >= 0 }
func (p *Position) SameLine(o *Position) bool {
	return p == o || (p.Filename == o.Filename && p.Line == o.Line)
}
func (p *Position) Same(o *Position) bool {
	return p == o ||
		p.Filename == o.Filename && p.Line == o.Line &&
		p.Column == o.Column && p.Offset == o.Offset
}

func makePosition(filename string, line, column int) (pos Position) {
	pos.Filename = filename
	pos.Line     = line
	pos.Column   = column
	return
}

func convPosition(filename, line, column string) (pos Position) {
	pos.Filename  = filename
	pos.Line, _   = strconv.Atoi(line)
	pos.Column, _ = strconv.Atoi(column)
	return
}

const NoPos Pos = Pos(golang.NoPos)

type Pos golang.Pos

func (p Pos) IsValid() bool {
	return golang.Pos(p).IsValid()
}

type TokFile struct {
	*golang.File
}

func (f *TokFile) string() string {
	return f.Name() //fmt.Sprintf("{%s}", f.Name())
}

func (f *TokFile) Offset(p Pos) int {
	return f.File.Offset(golang.Pos(p))
}

func (f *TokFile) Line(p Pos) int {
	return f.File.Line(golang.Pos(p))
}

func (f *TokFile) Pos(offset int) Pos {
	return Pos(f.File.Pos(offset))
}

func (f *TokFile) PositionFor(p Pos, adjusted bool) (pos Position) {
	return Position{ f.File.PositionFor(golang.Pos(p), adjusted) }
}

func (f *TokFile) Position(p Pos) (pos Position) {
	return Position{ f.File.Position(golang.Pos(p)) }
}

type FileSet struct {
	*golang.FileSet
}

// NewFileSet creates a new file set.
func NewFileSet() *FileSet {
	return &FileSet{ golang.NewFileSet() }
}

func (s *FileSet) AddFile(filename string, base, size int) *TokFile {
	return &TokFile{ s.FileSet.AddFile(filename, base, size) }
}

func (s *FileSet) Iterate(f func(*TokFile) bool) {
	s.FileSet.Iterate(func(file *golang.File) bool {
		return f(&TokFile{ file })
	})
}
