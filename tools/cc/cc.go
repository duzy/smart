//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
//  CC compatible toolchain -- gcc, clang
//
//      type
//      tool
//      defines
//      include_dirs
//      lib_dirs
//      libs
//      compile_flags
//      gen_flags
//      sources
//      
//      
package smart

import (
        "os"
        "fmt"
        "runtime"
        "strings"
        "path/filepath"
        . "github.com/duzy/smart/build"
)

var hc = MustHookup(`
template cc

me.compile_flags := -std=c++1y

post

$(me.dir)/$(cc:libname): $(cc:objects)
	$(cc:gen $@) $^ $(cc:loadings)

%.o: %.c   ; $(cc:compile $<) -o $@
%.o: %.cpp ; $(cc:compile $<) -o $@

commit

template cc-system-library

me.type := system-library
me.compile_flags :=
me.include_dirs :=
me.lib_dirs :=
me.libname = $(me.name)
me.libs = $(me.libname)

post

$(me.dir):!:
$(me.dir):?:
	pkg-config --exists $(me.name)

commit

template cc-system-library-pkg-config

me.type := system-library
me.compile_flags := $(pkg-config --cflags-only-other $(me.name))
me.include_dirs := $(pkg-config --cflags-only-I $(me.name))
me.lib_dirs := $(pkg-config --libs-only-L $(me.name))
me.libs := $(pkg-config --libs-only-l $(me.name))

post

$(me.dir):!:
$(me.dir):?:
	@pkg-config --exists $(me.name)

commit
`, HooksMap{
        "cc": HookTable{
                "sources": hook_sources,
                "objects": hook_objects,
                "libname": hook_libname,
                "loadings": hook_loadings,
                "compile": hook_compile,
                "gen": hook_gen,
        },
})

var langs = SourceLangMap{
        "c":   []string{ ".c" },
        "c++": []string{ ".cc", ".cpp", ".cxx", ".c++" },
}

func isCxxExtension(s string) bool {
        return langs.ExtLang(s) == "c++"
}

func getGenTypeString(ctx *Context) string {
        return strings.ToLower(ctx.Call("me.type").Expand(ctx))
}

func getToolchainCommand(ctx *Context, lang string) (cmd string) {
        tool := ctx.Call("me.tool").Expand(ctx)
        switch tool {
        case "clang", "clang++":
                switch lang {
                case "c":       cmd = "clang"
                case "c++":     cmd = "clang++"
                }
        case "gcc", "g++":
                switch lang {
                case "c":       cmd = "gcc"
                case "c++":     cmd = "g++"
                }
        case "":
                switch lang {
                case "c":       cmd = "cc"
                case "c++":     cmd = "c++"
                }
        }
        if cmd == "" {
                cmd = "false"
        }
        return
}

func expandFormatList(ctx *Context, name, format string) (list []string) {
        var prefix, suffix string
        if i := strings.Index(format, "%s"); 0 <= i {
                prefix = format[0:i]
                suffix = format[i+2:]
        }
        for _, s := range Split(ctx.Call(name).Expand(ctx)) {
                s = strings.TrimPrefix(s, prefix)
                s = strings.TrimSuffix(s, suffix)
                list = append(list, fmt.Sprintf(format, s))
        }
        return
}

func hook_sources(ctx *Context, args Items) (sources Items) {
        dir := ctx.Call("me.dir").Expand(ctx)
        for _, a := range Split(args.Expand(ctx)) {
                d := filepath.Join(dir, a)
                filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
                        if langs.ExtLang(filepath.Ext(path)) != "" {
                                //path = strings.TrimPrefix(path, dir)
                                //path = strings.TrimPrefix(path, "/")
                                sources.AppendString(path)
                        }
                        return nil
                })
        }
        return
}

func hook_objects(ctx *Context, args Items) (objects Items) {
        if args.IsEmpty(ctx) {
                args = ctx.Call("me.sources")
        }
        for _, a := range args {
                for _, src := range strings.Fields(a.Expand(ctx)) {
                        ext := filepath.Ext(src)
                        obj := strings.TrimSuffix(src, ext) + ".o"
                        objects = append(objects, StringItem(obj))
                }
        }
        return
}

func hook_compile(ctx *Context, args Items) (compile Items) {
        if 0 < len(args) {
                source := args[0].Expand(ctx)
                if isCxxExtension(filepath.Ext(source)) {
                        compile.AppendString(getToolchainCommand(ctx, "c++"))
                } else {
                        compile.AppendString(getToolchainCommand(ctx, "c"))
                }

                for _, u := range ctx.Call("me.using") {
                        name := u.Expand(ctx)
                        switch strings.TrimSpace(ctx.Call(name+".type").Expand(ctx)) {
                        case "shared":
                                // TODO: ...
                        case "static":
                                // TODO: ...
                        case "system-library":
                                compile.Append(ctx.Call(name+".compile_flags")...)
                                compile.AppendString(expandFormatList(ctx, name+".defines", `-D%s`)...)
                                compile.AppendString(expandFormatList(ctx, name+".include_dirs", `-I%s`)...)
                        }
                }
                
                compile.Append(ctx.Call("me.compile_flags")...)

                for _, d := range Split(ctx.Call("me.defines").Expand(ctx)) {
                        compile.AppendString(fmt.Sprintf("-D%s", d))
                }
                
                for _, d := range Split(ctx.Call("me.include_dirs").Expand(ctx)) {
                        compile.AppendString(fmt.Sprintf("-I%s", d))
                }
                
                compile.AppendString("-c", source)
        } else {
                compile.AppendString("false")
        }
        return
}

func hook_gen(ctx *Context, args Items) (gen Items) {
        var (
                flags Items
                t = getGenTypeString(ctx)
                cmd = getToolchainCommand(ctx, "c")
        )

source_loop:
        for _, s := range ctx.Call("me.sources") {
                if isCxxExtension(filepath.Ext(s.Expand(ctx))) {
                        cmd = getToolchainCommand(ctx, "c++")
                        break source_loop
                }
        }
        
        switch t {
        case "shared":
                gen = Items{ StringItem(cmd) }
                if !args.IsEmpty(ctx) {
                        flags.AppendString("-o")
                        //flags.AppendString(fmt.Sprintf("lib%s.so", args.Expand(ctx)))
                        flags.AppendString(args.Expand(ctx))
                }
                flags.AppendString("-shared")
        case "static":
                gen = Items{ StringItem("ar") }
                if !args.IsEmpty(ctx) {
                        flags.AppendString("crs")
                        //flags.AppendString(fmt.Sprintf("lib%s.a", args.Expand(ctx)))
                        flags.AppendString(args.Expand(ctx))
                }
        case "exe": fallthrough
        default:
                gen = Items{ StringItem(cmd) }
                if !args.IsEmpty(ctx) {
                        flags.AppendString("-o")
                        flags.Append(args...)
                }
        }

        gen.Append(flags...)
        gen.Append(ctx.Call("me.gen_flags")...)
        return 
}

func hook_libname(ctx *Context, args Items) (libname Items) {
        var (
                t = getGenTypeString(ctx)
                s = ctx.Call("me.name").Expand(ctx)
        )
        switch t {
        case "shared": s = "lib" + strings.TrimPrefix(s, "lib") + ".so"
        case "static": s = "lib" + strings.TrimPrefix(s, "lib") + ".a"
        default:
                switch runtime.GOOS {
                case "windows": s += ".exe"
                }
        }
        libname.AppendString(s)
        return
}

func hook_loadings(ctx *Context, args Items) (loadings Items) {
        var (
                using = ctx.Call("me.using")
                t = getGenTypeString(ctx)
                rpaths, dirs, libs []string
        )
        switch t {
        case "static": return
        case "shared":
                // TODO: special handling regarding --no-undefined
                /*
                if --no-undefined {
                        
                } else {
                        return
                } */
        }
        
        for _, u := range using {
                name := u.Expand(ctx)
                wd := ctx.Call(name+".workdir").Expand(ctx)
                t := ctx.Call(name+".type").Expand(ctx)
                switch t {
                case "shared":
                        // Add runtime shared library search paths.
                        rpaths = append(rpaths, fmt.Sprintf(`-Wl,-rpath="%s"`, wd))
                        fallthrough
                case "static":
                        dirs = append(dirs, fmt.Sprintf(`-L"%s"`, wd))
                        libs = append(libs, fmt.Sprintf(`-l"%s"`, strings.TrimPrefix(name, "lib")))
                case "system-library":
                        //dirs = append(dirs, expandFormatList(ctx, name+".lib_dirs", "-L", "", `-L"%s"`)...)
                        //libs = append(libs, expandFormatList(ctx, name+".libs", "-l", "", `-l"%s"`)...)
                        libs = append(libs, ctx.Call(name+".libs").Expand(ctx))
                }
        }

        loadings.AppendString(rpaths...)
        loadings.AppendString(dirs...)
        loadings.AppendString(libs...)
        loadings.Append(ctx.Call("me.libs")...)
        return
}
