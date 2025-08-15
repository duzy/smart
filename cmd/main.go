//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//
package main

import (
    "extbit.io/smart"
    "path/filepath"
    "os"
)

func main() {
    var w string
    for _, s := range []string{`/Volumes`, `/media`, `/`, os.Getenv("HOME")}{
        s = filepath.Join(s, "workspace")
        if x, y := os.Stat(s); y == nil && x.IsDir() { w = s }
    }

    var modules = `extbit.io/smart/modules`
    var paths []string
    if w != "" {
        paths = []string{
            filepath.Join(w, "smart"),
            filepath.Join(w, "go", modules),
        }
    }
    for _, s := range filepath.SplitList(os.Getenv("GOPATH")) {
        s = filepath.Join(s, `src`, modules)
        if x, y := os.Stat(s); x != nil && y == nil { paths = append(paths, s) }
    }
    smart.AddPaths(paths...)
    smart.Main()
}
