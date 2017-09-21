//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package loader

import (
        "testing"
        "fmt"
)

func TestLoad(t *testing.T) {
        i := New()
        if err := i.Load(`testdata/example.smart`, nil); err != nil {
                fmt.Printf("%v\n", err)
                t.Fatalf("Load: testdata/example.smart")
        }
        if err := i.Run(); err != nil {
                t.Fatalf("Load: %v", err)
        }
}

func TestBuildExample(t *testing.T) {
        i := New()
        i.AddSearchPaths(`../modules`)
        if err := i.Load(`testdata/example-build.smart`, nil); err != nil {
                fmt.Printf("%v\n", err)
                t.Fatalf("Load: testdata/example-build.smart")
        }
        if err := i.Run(); err != nil {
                t.Fatalf("Run: %v", err)
        }
}
