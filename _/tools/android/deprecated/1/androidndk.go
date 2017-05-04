//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "os"
        "os/exec"
        "path/filepath"
        "runtime"
        "strings"
        . "github.com/duzy/smart/build"
)

func init() {
        ndk := &_androidndk{ root:"", toolchainByAbi:make(map[string]string, 5) }
        RegisterToolset("android-ndk", ndk)

        if c, e := exec.LookPath("ndk-build"); e == nil {
                ndk.root = filepath.Dir(c)
        } else {
                if ndk.root = os.Getenv("ANDROIDNDK"); ndk.root == "" {
                        Message("cant locate Android NDK: %v", e)
                }
        }

        if ndk.root == "" {
                return
        }

        toolchainsDir := filepath.Join(ndk.root, "toolchains")

        names, err := ReadDirNames(toolchainsDir)
        if err != nil {
                Message("no toolchains in Android NDK `%v' (%v)", ndk.root, err)
                return
        }

        for _, name := range names {
                d := filepath.Join(toolchainsDir, name)
                if !ndk.addToolchain(d) {
                        //Message("bad toolchain `%v'", d)
                }
        }
}

type _androidndk struct {
        _gcc
        root string
        toolchainByAbi map[string]string
        modules map[string]*Module
}

func (ndk *_androidndk) addToolchain(d string) bool {
        toolchain := filepath.Base(d)
        if toolchain == "" {
                Message("error: toolchain: %v", d)
                return false
        }

        fn := filepath.Join(d, "config.mk")
        ctx, err := NewContextFromFile(fn, nil)
        if err != nil {
                //Message("error: toolchain: %v", err)
                return false
        }

        if s := strings.TrimSpace(ctx.Call("TOOLCHAIN_ABIS")); s != "" {
                abis := strings.Split(s, " ")
                for _, abi := range abis {
                        //Message("%v: %v, %v", fn, toolchain, abi)
                        ndk.toolchainByAbi[abi] = toolchain
                }
                return true
        }

        return false
}

func (ndk *_androidndk) toolchainDir(abi string) string {
        osname := ""

        switch runtime.GOOS {
        case "linux":
                osname = "linux-x86"
                osname = "linux-x86_64"
        default:
                print("TODO: choose Android NDK toolchain for `"+runtime.GOOS+"'\n")
        }

        if osname == "" || abi == "" {
                return ""
        }

        //toolchain := "arm-linux-androideabi-4.4.3"
        if toolchain, ok := ndk.toolchainByAbi[abi]; ok {
                return filepath.Join(ndk.root, "toolchains", toolchain, "prebuilt", osname)
        }
        return ""
}

func (ndk *_androidndk) ConfigModule(ctx *Context, m *Module, args []string, vars map[string]string) bool {
        if !ndk._gcc.ConfigModule(ctx, m, args, vars) {
                return false
        }

        if _, ok := ndk.modules[m.Name]; ok {
                //errorf(0, "module `%v' already defined in $ANDROIDNDK/sources", m.name)
        }

        var ld *gccCommand
        if c, ok := m.Action.Command.(*gccCommand); !ok {
                Errorf(0, "not a gcc command")
        } else {
                ld = c
        }

        var abi, platform string
        if s, ok := vars["ABI"]; ok { abi = s } else { abi = "armeabi" }
        if s, ok := vars["PLATFORM"]; ok { platform = s } else { platform = "android-9" }

        bin := filepath.Join(ndk.toolchainDir(abi), "bin")
        switch filepath.Base(ld.GetPath()) {
        case "ld": ld.SetPath(filepath.Join(bin, "arm-linux-androideabi-ld"))
        case "ar": ld.SetPath(filepath.Join(bin, "arm-linux-androideabi-ar"))
        }

        ld.SetIA32(IsIA32Command(ld.GetPath()))

        arch := "arch-"
        switch {
        case strings.HasSuffix(abi, "armeabi"): arch += "arm"
        case abi == "x86": arch += "x86"
        }

        includes := filepath.Join(ndk.root, "platforms", platform, arch, "usr/include")
        libdirs := filepath.Join(ndk.root, "platforms", platform, arch, "usr/lib")

        /*
        var v *define
        loc := ctx.l.location()
        v = ctx.set("me.abi", abi);           v.loc = *loc
        v = ctx.set("me.platform", platform); v.loc = *loc
        v = ctx.set("me.includes", includes); v.loc = *loc
        v = ctx.set("me.libdirs", libdirs);   v.loc = *loc */
        ctx.Set("me.abi", abi)
        ctx.Set("me.platform", platform)
        ctx.Set("me.includes", includes)
        ctx.Set("me.libdirs", libdirs)
        return true
}

func (ndk *_androidndk) CreateActions(ctx *Context, m *Module, args []string) bool {
        if !ndk._gcc.CreateActions(ctx, m, args) {
                return false
        }

        platform := strings.TrimSpace(ctx.Call("me.platform"))
        if platform == "" {
                Errorf(0, "unkown platform for `%v'", m.Name)
        }

        bin := filepath.Join(ndk.toolchainDir(ctx.Call("me.abi")), "bin")
        binAs := filepath.Join(bin, "arm-linux-androideabi-as")
        binGcc := filepath.Join(bin, "arm-linux-androideabi-gcc")
        binGxx := filepath.Join(bin, "arm-linux-androideabi-g++")

        var setCommands func(a *Action)
        setCommands = func(a *Action) {
                for _, pre := range a.Prequisites {
                        if pre.Command == nil { continue }
                        if c, ok := pre.Command.(*gccCommand); !ok {
                                Message("%v: wrong command `%v'", ctx.CurrentLocation(), pre.Command)
                                continue
                        } else {
                                switch filepath.Base(c.GetPath()) {
                                case "as":  c.SetPath(binAs)
                                case "gcc": c.SetPath(binGcc)
                                case "g++": c.SetPath(binGxx)
                                default: Errorf(0, "unknown command %v", c.GetPath())
                                }

                                c.SetIA32(IsIA32Command(c.GetPath()))
                        }
                        setCommands(pre)
                }
        }
        setCommands(m.Action)
        return true
}

func (ndk *_androidndk) loadModule(fn, ndksrc, subdir string) (ok bool) {
        ctx, err := NewContextFromFile(fn, map[string]string{
                "my-dir": filepath.Join(ndksrc, subdir),
        })

        if err != nil {
                Errorf(0, "failed to load module resident in $ANDROIDNDK/sources/%v", subdir)
        }

        Message("ndk: %v, %v", subdir, ctx.Call("LOCAL_PATH"))
        return false
}

func (ndk *_androidndk) loadModules() (ok bool) {
        return true

        ndksrc := filepath.Join(ndk.root, "sources")
        err := Traverse(ndksrc, func(fn string, fi os.FileInfo) bool {
                if !fi.IsDir() && fi.Name() == "Android.mk" {
                        if ok = ndk.loadModule(fn, ndksrc, filepath.Dir(fn[len(ndksrc)+1:])); !ok {
                                //return false
                        }
                }
                return true
        })
        if err != nil {
                return false
        }
        return ok
}

func (ndk *_androidndk) UseModule(ctx *Context, m, o *Module) bool {
        if !(m.Toolset == nil && m.Kind == "") {
                //errorf(0, "no toolset for `%v'", ctx.module.name)
                return false
        }
        if !(o.Toolset == nil && o.Kind == "") {
                //errorf(0, "no toolset for `%v'", ctx.module.name)
                return false
        }

        if ndk.modules == nil && !ndk.loadModules() {
                Errorf(0, "failed to load modules resident in $ANDROIDNDK/sources")
        }

        Message("use: %v by %v", m.Name, o.Name)

        return false
}
