//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func do_helpscreen(ctx Context) {
        prompt(ctx, `Build your projects the smart way!

Usage:

    smart -help[(arguments)]
    smart -configure[(arguments)]
    smart -reconfigure[(arguments)]
`)
        for name, _ := range ctx.Globe().flagEntries {
                if name == "" { continue }
                prompt(ctx, `
    smart -%s[(arguments)]`, name)
        }

        prompt(ctx, `

Basic:

   -h
   -help
    Display this help screen.

   -c
   -configure
    Configure all projects underneath the work directory.

   -r
   -reconfigure
    Reconfigures all projects underneath the work directory.

`)

        print_flag_entries(ctx)
        print_help_entries(ctx)
        print_options(ctx)

        prompt(ctx, `
Issues:

    * https://github.com/extbit/smart/issues
    * https://bugs.extbit.io/smart/report (not ready yet)

`)
}

func print_flag_entries(ctx Context) {
        prompt(ctx, "Defined:\n")
        for name, entries := range ctx.Globe().flagEntries {
                if len(entries) == 0 || name == "" { continue }
                prompt(ctx, `
   -%s`, name)
        }
        prompt(ctx, "\n\n")
}

func print_flag_trace(ctx Context) {
        for name, entries := range ctx.Globe().flagEntries {
                if name == "" { continue }
                for _, entry := range entries {
                        prompt(ctx, "%s: %v\n", entry.Position(), entry)
                }
        }
}

func print_help_entries(ctx Context) {
}

func print_options(ctx Context) {
        type opt struct { entry Entry; infos []Value }
        var opts []opt
        for _, entry := range configuration.entries {
                okay, infos := entry.option(ctx)
                if okay { opts = append(opts, opt{entry, infos}) }
        }

        if len(opts) == 0 { return }

        prompt(ctx, "Configure:\n\n")
        for _, opt := range opts {
                prompt(ctx, "    %v:\n", opt.entry)
                for _, info := range opt.infos {
                        prompt(ctx, "        %s\n", info.Strval(ctx))
                }
        }
}

func print_configuration(ctx Context) {
        prompt(ctx, `Configuration:
`)

        var configs = make(map[*Project][]Entry)
        for _, entry := range configuration.entries {
                project := entry.OwnerProject()
                entries, _ := configs[project]
                entries = append(entries, entry)
                configs[project] = entries
        }

        for project, entries := range configs {
                prompt(ctx, `
    %s`, project.spec)
                for _, entry := range entries {
                        prompt(ctx, `
        %s`, entry)
                }
        }

        prompt(ctx, "\n")
}
