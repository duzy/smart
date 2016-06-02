//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
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
                "gcc": HookTable{
                        "objects": hook_objects,
                        "gen": hook_gen,
                        "loadings": hook_loadings,
                        //"c++-source-p": hookIsCxxSource,
                        //"c-source-p": hookIsCSource,
                },
        },
        `# Build GCC Projects
template gcc

post

$(me.name): $(gcc:objects)
	$(gcc:gen $@) $^ $(gcc:loadings)

%.o: %.c   ; gcc -c -o $@ $<
%.o: %.cpp ; g++ -c -o $@ $<

commit
`)

func getGenTypeString(ctx *Context) string {
        return strings.ToLower(ctx.Call("me.type").Expand(ctx))
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

func hook_gen(ctx *Context, args Items) (gen Items) {
        var (
                flags Items
                t = getGenTypeString(ctx)
        )

        for _, a := range ctx.Call("me.sources") {
                src := a.Expand(ctx)
                ext := strings.ToLower(filepath.Ext(src))
                switch ext {
                case ".cpp": fallthrough
                case ".cxx": fallthrough
                case ".c++": fallthrough
                case ".cc":
                        gen = Items{ StringItem("g++") }
                        return
                }
        }
        if gen.IsEmpty(ctx) {
                gen = Items{ StringItem("gcc") }
        }

        switch t {
        case "shared":
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
