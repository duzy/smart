//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
//  Building Android NDK modules using it's ndk-build command.
//
//      me.api          - APP_API
//      me.optim        - APP_OPTIM
//      me.platform     - APP_PLATFORM
//      me.stl          - APP_STL
//      me.scripts      - Android.mk files
//      
package smart

import (
        . "github.com/duzy/smart/build"
        "path/filepath"
        "strings"
        "fmt"
        "os"
)

var hc = MustHookup(`
template android-ndk

me.abi = armeabi
me.optim = debug
me.platform = android-9
me.stl = system
me.boot_script = $(me.dir)/out/RootAndroid.mk
me.modules_dump_script = $(me.dir)/out/DumpModules.mk

post

build-ndk-modules:!: $(me.boot_script) ndk-build
	ndk-build NDK_PROJECT_PATH="$(me.dir)" NDK_OUT="$(me.dir)/out" APP_BUILD_SCRIPT="$(me.boot_script)" APP_ABI="$(me.abi)" APP_PLATFORM="$(me.platform)" APP_STL="$(me.stl)" APP_OPTIM="$(me.optim)"

$(me.dir)/out/ModulesDatabase-%.mk: $(me.modules_dump_script)
	@echo "android-ndk: export modules database ($*)..."
	ndk-build NDK_PROJECT_PATH="$(me.dir)" NDK_OUT="$(me.dir)/out" APP_BUILD_SCRIPT="$(me.modules_dump_script)" APP_ABI="$*" APP_PLATFORM="$(me.platform)" APP_STL="$(me.stl)" APP_OPTIM="$(me.optim)" smart-dump-android-modules-database

$(me.boot_script): $(~:scripts)
	@mkdir -p $(@D) && (for s in $^ ; do printf "include %s\n" $$s ; done) > "$@"

$(me.modules_dump_script): $(~:scripts)
	mkdir -p $(@D) && $(~:generate-android-modules-export-script)

ndk-build:!:
ndk-build:?:
	@which $@ > /dev/null || ( echo "ndk-build is not found" && false )

# $(me.dir)/out/ModulesDatabase-armeabi.mk
# $(me.dir)/out/ModulesDatabase-armeabi-v7a.mk
# $(me.dir)/out/ModulesDatabase-armeabi-v7a-hard.mk
$(~:load-android-modules-database)

commit
`, HooksMap{
        "android-ndk": HookTable{
                "scripts": hook_scripts,
                "generate-android-modules-export-script": hook_gen_ames,
        },
})

func hook_scripts(ctx *Context, args Items) (scripts Items) {
        // TODO: using $(me.scripts) and $(me.script)

        dir := ctx.Call("me.dir").Expand(ctx)
        filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
                if info != nil && info.Name() == `Android.mk` {
                        scripts.AppendString(path) //(filepath.Join(dir, path))
                }
                return nil
        })
        return
}

func hook_gen_ames(ctx *Context, args Items) (cmd Items) {
        target := ctx.Call("@").Expand(ctx)
        f, e := os.Create(target)
        if e != nil {
                cmd.AppendString(fmt.Sprintf(`echo "%s" && false`, e))
                return
        }

        defer f.Close()

        scripts := strings.Fields(ctx.Call("^").Expand(ctx))
        for _, s := range scripts {
                fmt.Fprintf(f, "include %s\n", s)
        }
        fmt.Fprintf(f, `# $(info $(modules-dump-database))
 ~:= smart-dump-android-modules-database
$~: DUMMY_TARGET_OUT := $(TARGET_OUT)
$~: DUMMY_TARGET_OBJS := $(TARGET_OBJS)
$~: DUMMY_TARGET_GDB_SETUP := $(TARGET_GDB_SETUP)
$~: DUMMY_TARGET_GDB_SERVER := $(TARGET_GDBSERVER)
$~: DUMMY_MODULES := $(modules-get-list)
$~:
	@echo "NDK_ROOT := $(NDK_ROOT)"
	@echo "TARGET_OUT := $(DUMMY_TARGET_OUT)"
	@echo "TARGET_OBJS := $(DUMMY_TARGET_OBJS)"
	@echo "TARGET_GDB_SETUP := $(DUMMY_TARGET_GDB_SETUP)"
	@echo "TARGET_GDB_SERVER := $(DUMMY_TARGET_GDB_SERVER)"
	@echo "MODULES := $(DUMMY_MODULES)"
	@$(foreach s,$(DUMMY_MODULES),\
echo "$(s)_NAME := $(__ndk_modules.$s.MODULE)" &&\
echo "$(s)_FILENAME := $(__ndk_modules.$s.MODULE_FILENAME)" &&\
echo "$(s)_PATH := $(__ndk_modules.$s.PATH)" &&\
echo "$(s)_SOURCES := $(__ndk_modules.$s.SRC_FILES)" &&\
echo "$(s)_SCRIPT := $(__ndk_modules.$s.MAKEFILE)" &&\
echo "$(s)_OBJS_DIR := $(__ndk_modules.$s.OBJS_DIR)" &&\
echo "$(s)_BUILT := $(__ndk_modules.$s.BUILT_MODULE)" &&\
echo "$(s)_INSTALLED := $(__ndk_modules.$s.INSTALLED)" &&\
echo "$(s)_CLASS := $(__ndk_modules.$s.MODULE_CLASS)" &&\
) true
	@echo
`)
                
        cmd.AppendString(fmt.Sprintf("test %s", target))
        return
}
