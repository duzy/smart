//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        //"fmt"
        "strings"
        "path/filepath"
        . "github.com/duzy/smart/build"
)

var hc = MustHookup(
        HooksMap{
                "gcc": HookTable{
                        "objects": hookObjects,
                },
        },
        `# Build GCC Projects
template gcc

post

$(me.name): $(gcc:objects)
	@echo "gcc: todo: $^ -> $(me.name) ($(me.workdir))"

%.o: %.c   ; gcc -c -o $@ $< ; pwd ; ls -l *.o
%.o: %.cpp ; g++ -c -o $@ $< ; pwd ; ls -l *.o

commit
`)

func hookObjects(ctx *Context, args Items) (objects Items) {
        if len(args) == 0 {
                args = ctx.Call("me.sources")
        }
        for _, a := range args {
                // FIXME: split 'a'
                src := a.Expand(ctx)
                ext := filepath.Ext(src)
                obj := strings.TrimSuffix(src, ext) + ".o"
                objects = append(objects, StringItem(obj))
        }
        return
}
