//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
//  Making Android NDK standalone toolchain.
//  
package smart

import (
        . "github.com/duzy/smart/build"
        "path/filepath"
        "os/exec"
)

var hc = MustHookup(`
template android-toolchain

~.ndk_dir := $(~:ndk-dir)
~.make-standalone-toolchain = $(~.ndk_dir)/build/tools/make-standalone-toolchain.sh

post

$(me.dir):!:
	SHELL=$$(which bash) $(~.make-standalone-toolchain) --arch=$(me.arch) --toolchain=$(me.toolchain) --platform=$(me.platform) --install-dir="$(@D)"
$(me.dir):?:
	@test -d $(me.dir)/bin
	@test -d $(me.dir)/lib
	@test -d $(me.dir)/libexec
	@test -d $(me.dir)/include
	@test -d $(me.dir)/include/c++
	@test -d $(me.dir)/sysroot
	@test -f $(me.dir)/bin/*-as
	@test -f $(me.dir)/bin/*-ld
	@test -f $(me.dir)/bin/*-gcc-ar
	@test -f $(me.dir)/bin/*-gcc-nm
	@test -f $(me.dir)/bin/*-gcc-ranlib
	@test -f $(me.dir)/bin/*-gcc
	@test -f $(me.dir)/bin/*-g++
	@test -f $(me.dir)/bin/*-strings
	@test -f $(me.dir)/bin/*-strip
	@test -f $(me.dir)/bin/*-size

commit
`, HooksMap{
        "android-toolchain": HookTable{
                "ndk-dir": hook_ndk_dir,
        },
})

func hook_ndk_dir(ctx *Context, args Items) (dir Items) {
        if c, e := exec.LookPath("ndk-build"); e == nil {
                dir.AppendString(filepath.Dir(c))
        } else {
                dir.AppendString("/opt/android/ndk")
        }
        return
}
