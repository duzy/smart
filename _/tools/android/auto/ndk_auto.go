//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
//  Building open-source autotool projects using Android NDK's standalone toolchain.
//  
package smart

import (
        "fmt"
        . "github.com/duzy/smart/build"
)

var hc = MustHookup(
        `# Build autotool projects using Android NDK's standalone toolchain.
template android/auto

post

$(me.dir)/Makefile: $(me.dir)/configure

$(me.dir)/configure: $(me.dir)/autogen.sh

commit
`, HooksMap{
        "android/auto": HookTable{
                "test": hook_test,
                // TODO: ...
        },
})

func hook_test(ctx *Context, args Items) (loadings Items) {
        fmt.Printf("test\n")
        return
}
