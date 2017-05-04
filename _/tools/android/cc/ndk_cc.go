//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
//  Building C/C++ using Android NDK standalone toolchain (gcc, clang).
//  
package smart

import (
        . "github.com/duzy/smart/build"
)

var hc = MustHookup(
        `# Build Android NDK Projects
template android-cc

post

use android-toolchain

$(info $(me.dir))

# TODO: ...

commit
`, HooksMap{
        "android-cc": HookTable{
                // TODO: ...
        },
})
