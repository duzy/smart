//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "bytes"
        "os/exec"
        . "github.com/duzy/smart/build"
)

var hc = MustHookup(
        `# Execute Shell Command
template shell

start:!: $(me.depends) $(me.using)
	@$(me.command) $(me.args)

commit
`, HooksMap{
        "shell": HookTable{
                "exec": hookExec,
        },
})


func hookExec(ctx *Context, args Items) (res Items) {
        cmd := exec.Command("sh", "-c", args.Expand(ctx))
        if cmd != nil {
                stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
                cmd.Stdout, cmd.Stderr = stdout, stderr
                if err := cmd.Run(); err != nil {
                        // TODO: report errors
                        return
                }
                res = append(res, StringItem(stdout.String()))
        } else {
                // TODO: report errors
        }
        return
}
