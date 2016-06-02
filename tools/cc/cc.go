//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
//  CC compatible toolchain -- gcc, clang
//  
package smart

import (
        "fmt"
        "strings"
        "path/filepath"
        . "github.com/duzy/smart/build"
)

var hc = MustHookup(
        HooksMap{
                "cc": HookTable{
                        "objects": hook_objects,
                        "loadings": hook_loadings,
                        "compile": hook_compile,
                        "gen": hook_gen,
                        //"c++-source-p": hookIsCxxSource,
                        //"c-source-p": hookIsCSource,
                },
        },
        `# Build GCC Projects
template cc

post

$(me.name): $(cc:objects)
	$(cc:gen $@) $^ $(cc:loadings)

%.o: %.c   ; $(cc:compile $<) -o $@
%.o: %.cpp ; $(cc:compile $<) -o $@

commit
`)

func isCxxExtension(s string) bool {
        switch strings.ToLower(s) {
        case ".cpp": fallthrough
        case ".cxx": fallthrough
        case ".c++": fallthrough
        case ".cc":  return true
        }
        return false
}

func getGenTypeString(ctx *Context) string {
        return strings.ToLower(ctx.Call("me.type").Expand(ctx))
}

func getToolchainCommand(ctx *Context, lang string) (cmd string) {
        tool := ctx.Call("me.tool").Expand(ctx)
        switch tool {
        case "clang":
                switch lang {
                case "c":       cmd = "clang"
                case "c++":     cmd = "clang++"
                }
        case "gcc":
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

func hook_objects(ctx *Context, args Items) (objects Items) {
        if args.IsEmpty(ctx) {
                args = ctx.Call("me.sources")
        }
        for _, a := range args {
                src := a.Expand(ctx) // FIXME: split 'a'
                ext := filepath.Ext(src)
                obj := strings.TrimSuffix(src, ext) + ".o"
                objects = append(objects, StringItem(obj))
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
                        flags.AppendString(fmt.Sprintf("lib%s.so", args.Expand(ctx)))
                }
                flags.AppendString("-shared")
        case "static":
                gen = Items{ StringItem("ar") }
                if !args.IsEmpty(ctx) {
                        flags.AppendString("crs")
                        flags.AppendString(fmt.Sprintf("lib%s.a", args.Expand(ctx)))
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
                        libs = append(libs, fmt.Sprintf(`-l"%s"`, name))
                }
        }

        loadings.AppendString(rpaths...)
        loadings.AppendString(dirs...)
        loadings.AppendString(libs...)
        return
}
