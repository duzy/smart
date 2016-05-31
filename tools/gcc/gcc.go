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
                        "objects": hookObjects,
                        "gen": hookGen,
                        //"c++-source-p": hookIsCxxSource,
                        //"c-source-p": hookIsCSource,
                },
        },
        `# Build GCC Projects
template gcc

post

$(me.name): $(gcc:objects)
	$(gcc:gen $@) $^ $(gcc:libs)

%.o: %.c   ; gcc -c -o $@ $<
%.o: %.cpp ; g++ -c -o $@ $<

commit
`)

func hookObjects(ctx *Context, args Items) (objects Items) {
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

func hookGen(ctx *Context, args Items) (cmd Items) {
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
                        cmd = Items{ StringItem("g++") }
                        return
                }
        }
        if cmd.IsEmpty(ctx) {
                cmd = Items{ StringItem("gcc") }
        }

        switch t {
        case "exe":
                if !args.IsEmpty(ctx) {
                        flags.AppendString("-o")
                        flags.Append(args...)
                }
        case "shared":
                if !args.IsEmpty(ctx) {
                        flags.AppendString("-o")
                        flags.Append(args...)
                }
                flags = append(flags, StringItem("-shared"))
        case "static":
                cmd = Items{ StringItem("ar") }
                if !args.IsEmpty(ctx) {
                        flags.AppendString("crs")
                        flags.AppendString(fmt.Sprintf("lib%s.a", args.Expand(ctx)))
                }
        }

        cmd.Append(flags...)
        cmd.Append(ctx.Call("me.gen_flags")...)
        return 
}

func getGenTypeString(ctx *Context) string {
        return strings.ToLower(ctx.Call("me.type").Expand(ctx))
}
