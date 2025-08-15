//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
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
    for name, _ := range _universe(ctx).globe.flagEntries {
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
        for name, entries := range _universe(ctx).globe.flagEntries {
                if len(entries) == 0 || name == "" { continue }
                prompt(ctx, `
   -%s`, name)
        }
        prompt(ctx, "\n\n")
}

func print_flag_trace(ctx Context) {
        for name, entries := range _universe(ctx).globe.flagEntries {
                if name == "" { continue }
                for _, entry := range entries {
                        prompt(ctx, "%s: %v\n", entry.Position(), entry)
                }
        }
}

func print_help_entries(ctx Context) {
}

func print_options(ctx Context) {
    type opt struct { entry entry; infos []Value }

    var opts []opt

    // _universe(ctx).config(func(proj *project, entry entry) {
    //     var infos = ruleOptionInfos(ctx, entry)
    //     if infos != nil { opts = append(opts, opt{entry, infos}) }
    // }, nil, nil)

    if len(opts) == 0 { return }

    prompt(ctx, "Configure:\n\n")
    for _, opt := range opts {
        prompt(ctx, "    %v:\n", opt.entry)
        for _, info := range opt.infos {
            prompt(ctx, "        %s\n", info.string(ctx))
        }
    }
}

func print_configuration(ctx Context) {
    prompt(ctx, `Configuration:
`)

    var configs = make(map[*project][]entry)

    // _universe(ctx).config(func(proj *project, entry entry) {
    //     entries, _ := configs[proj]
    //     entries = append(entries, entry)
    //     configs[proj] = entries
    // }, nil, nil)

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

func ruleOptionInfos(ctx Context, e entry) (infos []Value) {
    for _, p := range e.programs() {
        for _, depend := range p.depends {
            g, ok := depend.(*modification)
            if!ok { continue }
            for _, m := range g.list {
                if m.elems[0].string(ctx) != "configure" { continue }
                for _, arg := range m.elems[1:] {
                    a, ok := arg.(*argumented)
                    if!ok { continue }
                    f, ok := a.Value.(flag)
                    if!ok { continue }
                    if f.Value.string(ctx) != "option" { continue }
                    for _, v := range a.args {
                        if p, ok := v.(*pair); ok {
                            if p.key.string(ctx) != "info" { continue }
                            v = p.val
                        }
                        infos = append(infos, v)
                    }
                    return
                }
            }
        }
    }
    return
}
