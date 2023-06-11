//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        // "extbit.io/smart"
        "testing"
)

var testtrace = true

func TestParseFile(t *testing.T) {
        // mode := DeclarationErrors
        // if testtrace {
        //         mode |= Trace
        // }
        // files := []string{
        //         `testdata/defines.smart`,
        //         `testdata/simple.smart`,
        //         `testdata/dialect.smart`,
        // }
	// for i, filename := range files {
	// 	_, err := ParseFile(NewFileSet(), filename, nil, mode)
	// 	if err != nil {
	// 		t.Fatalf("ParseFile: #%d: %v", i, err)
	// 	}
	// }
}

func TestParseDir(t *testing.T) {
        // fset, dir := NewFileSet(), "testdata"
        // _, err := ParseDir(fset, dir, nil, DeclarationErrors)
        // if err != nil {
        //         t.Fatalf("ParseDir(%s): %v", dir, err)
        // }
}
