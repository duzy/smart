//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "fmt"
)

func do_helpscreen() {
        fmt.Fprintf(stderr, `Build your projects the smart way!

Usage:

    smart -help[(arguments)]
    smart -configure[(arguments)]
    smart -reconfigure[(arguments)]
`)
        for name, _ := range context.flagEntries {
                if name == "" { continue }
                fmt.Fprintf(stderr, `
    smart -%s[(arguments)]`, name)
        }

        fmt.Fprintf(stderr, `

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

        print_flag_entries()
        print_help_entries()
        print_options()

        fmt.Fprintf(stderr, `
Issues:

    * https://github.com/extbit/smart/issues
    * https://bugs.extbit.io/smart/report (not ready yet)

`)
}

func print_flag_entries() {
        fmt.Fprintf(stderr, "Defined:\n")
        for name, entries := range context.flagEntries {
                if len(entries) == 0 || name == "" { continue }
                fmt.Fprintf(stderr, `
   -%s`, name)
        }
        fmt.Fprintf(stderr, "\n\n")
}

func print_help_entries() {
}

func print_options() {
        type opt struct { entry *RuleEntry; infos []Value }
        var opts []opt
        for _, entry := range configuration.entries {
                okay, infos := entry.option()
                if okay { opts = append(opts, opt{entry, infos}) }
        }

        if len(opts) == 0 { return }

        fmt.Fprintf(stderr, "Configure:\n\n")
        for _, opt := range opts {
                fmt.Fprintf(stderr, "    %v:\n", opt.entry)
                for _, info := range opt.infos {
                        s, _ := info.Strval()
                        fmt.Fprintf(stderr, "        %s\n", s)
                }
        }
}

func print_configuration() {
        fmt.Fprintf(stderr, `Configuration:
`)

        var configs = make(map[*Project][]*RuleEntry)
        for _, entry := range configuration.entries {
                project := entry.OwnerProject()
                entries, _ := configs[project]
                entries = append(entries, entry)
                configs[project] = entries
        }

        for project, entries := range configs {
                fmt.Fprintf(stderr, `
    %s`, project.spec)
                for _, entry := range entries {
                        fmt.Fprintf(stderr, `
        %s`, entry)
                }
        }

        fmt.Fprintf(stderr, "\n")
}
