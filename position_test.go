//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	//"fmt"
	//"math/rand"
	//"sync"
	"path/filepath"
	"testing"
)

func testPositionExample(t *testing.T) {
        src := []byte(`
project foo
include modules/*.smart
`)
        filename := filepath.Join("test", "TestPositionExample")
        fs := NewFileSet()
        f := fs.AddFile(filename, fs.Base(), len(src))
        f.SetLinesForContent(src)
        if x := f.LineCount(); x < 2 {
                t.Errorf("LineCount: %v < 2", x)
        } else {
                
        }
}
