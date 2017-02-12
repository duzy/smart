//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package parser

import (
        "github.com/duzy/smart/token"
        "testing"
)

func TestParseFile(t *testing.T) {
        mode := DeclarationErrors | Trace
        files := []string{
                `testdata/defines.smart`,
                `testdata/simple.smart`,
        }
	for i, filename := range files {
		_, err := ParseFile(token.NewFileSet(), filename, nil, mode)
		if err != nil {
			t.Fatalf("ParseFile: #%d: %v", i, err)
		}
	}
}

func TestParseDir(t *testing.T) {
        fset, dir := token.NewFileSet(), "testdata"
        _, err := ParseDir(fset, dir, nil, DeclarationErrors)
        if err != nil {
                t.Fatalf("ParseDir(%s): %v", dir, err)
        }
}
