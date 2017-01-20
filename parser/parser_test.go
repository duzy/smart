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
        files := []string{
                `testdata/defines.smart`,
        }
	for _, filename := range files {
		_, err := ParseFile(token.NewFileSet(), filename, nil, DeclarationErrors)
		if err != nil {
			t.Fatalf("ParseFile(%s): %v", filename, err)
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
