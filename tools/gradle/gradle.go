//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
//  Building gradle backed projects.
//  
package smart

import (
        "fmt"
        . "github.com/duzy/smart/build"
)

var hc = MustHookup(`
template gradle

post

# TODO: ...

commit
`, HooksMap{
        "gradle": HookTable{
                "test": hook_test,
                // TODO: ...
        },
})

func hook_test(ctx *Context, args Items) (loadings Items) {
        fmt.Printf("test\n")
        return
}
