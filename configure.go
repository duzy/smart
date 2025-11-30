//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "path/filepath"
    "io/ioutil"
    "strings"
    "reflect"
    "regexp"
    "bufio"
    "bytes"
    "sort"
    "time"
    "fmt"
    "os"
)

func _configurecontext(c Context) *configurecontext { return cast[*configurecontext](c) }
func is_configurecontext(ctx Context) bool { return _configurecontext(ctx) != nil }

type silent_configure struct {}

type configurecontext struct {
    Context
    current *project
    silent bool
    file *os.File
    writer *bufio.Writer
    defs map[string]struct{}
    done map[*def]struct{}
}
func (cc *configurecontext) inner() Context { return cc.Context }
func (cc *configurecontext) cast(t reflect.Type) Context { return icast(cc, t) }
func (cc *configurecontext) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case silent_configure: if cc.silent { return true }
    }
    return cc.Context.do(ctx, op)
}
func (cc *configurecontext) open_configuration_sm(ctx Context, p *project) (res *os.File) {
    if f := p.configuration_sm(ctx); f == nil {
        erro(ctx, "%p: nil configuration file", p).trace()
    } else if s := f.fullname(); s == "" {
        erro(ctx, "empty configuration file name: %v", f).trace()
    } else if e := os.MkdirAll(filepath.Dir(s), os.FileMode(0755)); e != nil {
        erro(ctx, "make path %s failed: %v", filepath.Dir(s), e).trace()
    } else {
        res, e = os.OpenFile(s, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0600))
        if e != nil { erro(ctx, "%v", e).trace() }
    }
    return
}
func (cc *configurecontext) execute(ctx *execution, e entry) {
    var d *def
    var s string
    var p = e.owner()

    if checkpoints {
        defer cc.execute_check(ctx, e, p, &s, &d)
    }

    if p != cc.current && p != nil {
        cc.defs = make(map[string]struct{}) // reset defs for p

        // NOTE: configuration.sm is created for every project
        var f = cc.open_configuration_sm(ctx, p)
        if f != nil {
            if cc.writer != nil {
                if e := cc.writer.Flush(); e != nil {
                    erro(ctx, "%v", e).trace()
                }
            }
            if cc.file != nil {
                if e := cc.file.Close(); e != nil {
                    erro(ctx, "%v", e).trace()
                }
            }
        }

        cc.file, cc.writer = f, bufio.NewWriter(f)
        fmt.Fprintf(cc.writer, "# %s (%s) configuration\n", p.spec, p.name)

        if !cc.silent {
            u := _universe(ctx)
            s := u.trimSpecPath(ctx, p.spec)
            prompt(ctx, "configure %s …… (%s)\n", p.name, s)
            if true { flush(ctx) }
        }

        cc.current = p
    }

    e.execute(ctx)

    s = __string(ctx, e.destiny())
    if _, y := cc.defs[s]; y { return }

    if d = cc.current.finddef(s); d == nil {
        erro(ctx, "%v: `%s` not configured", cc.current, s).trace()
        return
    }

    cc.defs[s] = struct{}{}

    if s := d.name; d.value == nil {
        // Set <nil> value with exec-assign ('!=') to a None value.
        fmt.Fprintf(cc.writer, "%v !=\n", s)
    } else {
        fmt.Fprintf(cc.writer, "%v = %v\n", s, d.value.String())
    }
    return
}
func (cc *configurecontext) close() {
    if cc.writer != nil { if e := cc.writer.Flush(); e != nil {} }
    if cc.file != nil   { if e := cc.file.Close();   e != nil {} }
}

func scanExitStatts(err error) (n, status int) {
    switch e := err.(type) {
    case *exitstatus: n, status = 1, e.int
    default: n, _ = fmt.Sscanf(err.Error(), fmtExitStatus, &status)
    }
    return
}

type filewalkFunc func(file *file, err error) error

func walkFileInfos(ctx Context, root string, pats []Value, fn filepath.WalkFunc) (err error) {
    return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if err != nil { return err }
    ForPats:
        for _, p := range pats {
            var matched bool
            if matched, _, _ = match(ctx, p, path); !matched {
                matched, _, _ = match(ctx, p, filepath.Base(path))
            }
            if matched {
                if err = fn(path, info, err); err != nil {
                    break ForPats
                }
            }
        }
        return err
    })
}

func walkFiles(ctx Context, root string, pats []Value, fn filewalkFunc) error {
    return walkFileInfos(ctx, root, pats, func(path string, info os.FileInfo, err error) error {
        if err != nil { return err }
        var rel string
        if rel, err = filepath.Rel(root, path); err != nil {
            return err
        }
        return fn(_stat(ctx, rel, stat_dir{root}, stat_fileinfo{info}), err)
    })
}

var configuredFiles = make(map[string]*scope, 8)

type (
    configureconvertArgs func([]Value, *bytes.Buffer) []Value
    configureconvertFunc func(string, *bytes.Buffer)
    configureconvertOpts struct {
        general_opts
        mode     os.FileMode `mode`
        makePath bool `path`
        mustConf bool `mustconfig,mustconf,must-conf,must-config,nc,needsconfig,needs-config`
        reconfig bool `reconfig`
        update   bool `update`
    }
)
func configureconvert(ctx *execution, dealArgs configureconvertArgs, dealData configureconvertFunc, opts *configureconvertOpts, args ...Value) (_ Value) {
    var (
        closured = closure_projects(ctx)
        filename string
        f *file
        target as
    )

    args = parse_opts(ctx, opts, args...)

    if target.Value = auto_get(ctx, "@"); isTrivial(target.Value) {
        erro(ctx, "'@' is not defined").trace()
    } else if f, filename, _ = target.file_fullname(ctx, closured...); f == nil {
        if depend := auto_get(ctx,">"); !isTrivial(depend) {
            panic(traveTargetNotDefinedFile)
        } else if true {
            prompt(ctx, "%v: not defined as file\n", __string(ctx, target))
            erro(ctx, "%v", ts(target.Value))
            errostack(ctx, 8).trace()
        }
        return
    } else if filename == "" {
        errostack(ctx, 3, "%v: empty fullname", target.Value).trace()
    }

    if _, prev := auto_set(ctx, defVoid, "@", f); opts.debug>0 {
        info(ctx, "configure-file: %s->%s (%v -> %v)", f, filename, ts(prev), ts(f)).debug(opts.debug)
    }

    if f.info == nil { if f := _stat(ctx, filename); f != nil { f.info = f.info }}
    if f != nil && 0 < opts.debug {
        info(ctx, "configure-file: %v: %v (%s) (%v)", auto_get(ctx,"@"), f.fullname(), closured).debug(opts.debug)
    }

    if len(ctx.proj.configs) == 0 {
        // no need to check configuration
    } else if f := ctx.proj.configuration_sm(ctx); f == nil || !f.exists() {
        prompt(ctx, "%v\n", filename)
        if opts.mustConf {
            var d = opts.debug ; if d == 0 { d = 1 }
            errostack(ctx, opts.stack, "no configuration (%v), try -conf first, in %v", f, ctx.proj).trace()
        } else if true {
            warnstack(ctx, opts.stack, "no configuration (%v), try -conf first, in %v", f, ctx.proj).debug(opts.debug)
        }
    }

    // Check previously configured files, we only configure once unless
    // optReconfig is true.
    var closure *scope
    if configuredFiles != nil {
        var okay bool
        closure, okay = configuredFiles[filename]
        if okay && closure != nil && !opts.reconfig { return }
    }

    //if closure == nil { closure = ctx.closcop }
    defer func(s string, c *scope) { configuredFiles[s] = c } (filename, closure)

    var data bytes.Buffer
    if h := auto_get(ctx,"-"); !isNull(h) { args = append(args, h) }
    if dealArgs != nil { args = dealArgs(args, &data) }
    if dealData != nil {
        for _, arg := range args {
            if str := __string(ctx, arg); str == "" {
                continue
            } else {
                dealData(str, &data)
            }
        }
    }

    if data.Len() == 0 {
        prompt(ctx, "%v: %v %v\n", filename, auto_get(ctx,"@"), auto_get(ctx,">"))
        errostack(ctx, 5, "empty configuration data").trace()
    } else if f := ctx.proj.configuration_sm(ctx); (f == nil || !f.exists()) && opts.debug>0 {
        // NOTE: TrimSpace to ease emacs *compilation* parse errors
        prompt(ctx, "%v: %v\n%s\n", filename, auto_get(ctx,"@"), strings.TrimSpace(data.String())).debug()
    }

    var (
        e error
        same bool
        status string
    )
    if opts.verbose { defer func(st time.Time) {
        if same {
            if true { return } else { status = "unchanged" }
        } else if status == "" {
            status = fmt.Sprintf("outdated (%s)", filename)
        }

        var d = time.Now().Sub(st)
        prompt(ctx, "update %v …… %s (in %v)\n", trimPromptString(filename), status, d)
        if d := opts.debug; d>0 { infostack(ctx, opts.stack, "%v (%v)", auto_get(ctx, "@"), d).debug(d) }
    }(time.Now())}

    if f.info != nil {
        if same, e = crc64CheckFileModeContent(ctx, filename, data.Bytes(), opts.mode); e != nil {
            erro(ctx, " crc64 checksum failed: %v", e).trace()
            return
        }
        if same {
            var tt = f.info.ModTime()
            for _, d := range merge(ctx.targets...) {
                if f, y := to_file(d); !y { continue } else
                if dt := f.info.ModTime(); dt.After(tt) { tt = dt }
            }
            if tt.After(f.info.ModTime()) { e = touch(ctx, f, 0, false, tt) }
            return f
        }
    } else if dir := filepath.Dir(filename); opts.makePath && dir != "." && dir != pathSep {
        if e = os.MkdirAll(dir, os.FileMode(0755)); e != nil {
            erro(ctx, " %v", e).trace()
        }
    }

    if e = ioutil.WriteFile(filename, data.Bytes(), opts.mode); e != nil {
        erro(ctx, " %v", e).trace()
    }

    if f.info == nil {
        if f.info, e = os.Stat(filename) ; e == nil {
            erro(ctx, " %v", e).trace()
        }
    }

    if 0 < opts.debug {
        status = fmt.Sprintf("configured (%s, %d bytes)", filename, data.Len())
    } else {
        status = fmt.Sprintf("configured (%d bytes)", data.Len())
    }
    return f
}

type modifier_configureinput struct { modifier_ }
func (ctx *modifier_configureinput) x(args ...Value) (result any) {
    var opts = configureconvertOpts{ mode: os.FileMode(0600) }
    var dealArgs = func(args []Value, out *bytes.Buffer) []Value {
        var p = _project(ctx)

        if x, y := p.Lookup("configure.names").(*def); y {
            args = append(args, xmerge(ctx, x.value)...)
        }

        var configs = make(map[string]*def)
        for _, a := range args {
            var name = __string(ctx, a)
            if _, ok := configs[name]; ok {
                continue
            } else if obj := p.resolve(ctx, name); obj == nil {
                erro(ctx, "undefined %v", name).trace()
                return nil
            } else if def, ok := obj.(*def); ok {
                configs[name] = def
            }
        }
        for _, c := range p.configs {
            var name = c.ident(ctx)
            if def, ok := p.Lookup(name).(*def); ok {
                configs[name] = def
            }
        }
        for _, def := range configs {
            fmt.Fprintf(out, "#undef %s\n", def.ident(ctx))
        }
        return args
    }
    return configureconvert(_execution(ctx), dealArgs, nil, &opts, args...)
}

// configure-file modifier (see also builtinConfigureFile), example usage:
//
//     config.h: config.h.in [(configure-file)]
//
type modifier_configurefile struct { modifier_ }
func (ctx *modifier_configurefile) x(args ...Value) (result any) {
    var opts = configureconvertOpts{ mode: os.FileMode(0600) }
    var convert = func(str string, out *bytes.Buffer) {
        configurestring(ctx, out, _project(ctx), str)
    }
    return configureconvert(_execution(ctx), nil, convert, &opts, args...)
}

// extract-configuration extracts configuration from C/C++ files, example usage:
//
//      config.h.in:[(extract-configuration)]: $(wildcard *.cpp)
//
type modifier_extractconfiguration struct { modifier_
    mode os.FileMode "mode"
    makePath bool "path"
    target string "target"
    rxs []*regexp.Regexp "rx,regex" // regexp.Compile(s)
}
func (ctx *modifier_extractconfiguration) x(args ...Value) (result any) {
    var pats []Value
    var pos = _position(ctx)
    for _, arg := range args {
        switch a := arg.(type) {
        case *group: pats = append(pats, a.elems...)
        default:     pats = append(pats, a)
        }
    }

    if len(pats) == 0 {
        erro(ctx, "extract-configuration: missing file names (patterns)").trace()
        return
    }

    if len(ctx.rxs) == 0 {
        erro(ctx, "extract-configuration: missing -rx=... flags").trace()
        return
    }

    if ctx.target == "" { ctx.target = "configuration" }

    var outFile string
    if target := auto_get(ctx,"@"); isNull(target) {
        erro(ctx, " target '@' is undefined").trace()
        return
    } else {
        outFile = __string(ctx, target)
    }

    if ctx.makePath {
        if err := os.MkdirAll(filepath.Dir(outFile), os.FileMode(0755)); err != nil {
            erro(ctx, " make path failed: %v", err).trace()
            return
        }
    }

    var fil, err = os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, ctx.mode)
    if err != nil {
        erro(ctx, " open file failed: %v", err).trace()
        return
    }

    var out = bufio.NewWriter(fil)
    defer func() { out.Flush() ; fil.Close() } ()

    var depends, sources []Value
    if d := auto_get(ctx, "^"); !isTrivial(d) { depends = xmerge(ctx, d) }

    var patsVal = ease(ctx, pats)
    for _, depend := range depends {
        var a []Value
        switch d := depend.(type) {
        case *file:
            if a = merge(call(ctx, "filter", nil, patsVal, d)); a != nil {
                sources = append(sources, a...)
            }
        case *path:
            var s = __string(ctx, d)
            err = walkFiles(ctx, s, pats, func(file *file, err error) error {
                if err == nil { sources = append(sources, file) }
                return err
            })
        default:
            var s = __string(ctx, d)
            var dir = filepath.Dir(s)
            var name = filepath.Base(s)
            var f = _stat(ctx, name, stat_dir{dir})
            if f == nil {
                erro(ctx, " extract-configuration: `%s` file not found", name).trace()
                return
            } else if f.info.IsDir() {
                err = walkFiles(ctx, s, pats, func(f *file, err error) error {
                    if err == nil { sources = append(sources, f) }
                    return err
                })
            } else if a = merge(call(ctx, "filter", nil, patsVal, d)); a != nil {
                sources = append(sources, a...)
            }
        }
    }

    var exprs = make(map[string]struct{})

sourceloop:
    for _, source := range sources {
        var s string
        switch t := source.(type) {
        case *file: s = t.fullname()
        default:    s = __string(ctx, t)
        }

        var f *os.File
        if f, err = os.Open(s); err != nil {
            prompt(ctx, "%v: (configure) %v: %v\n", pos, source, err)
            continue sourceloop
        }

        scanner := bufio.NewScanner(f)
        scanner.Split(bufio.ScanLines)
    scanloop:
        for scanner.Scan() {
            var s = scanner.Text()
            for _, x := range ctx.rxs {
                if sm := x.FindStringSubmatch(s); sm == nil {
                    continue
                } else {
                    exprs[sm[1]] = struct{}{}
                    break scanloop
                }
            }
        }

        f.Close()
    }

    var keys []string
    for x, _ := range exprs { keys = append(keys, x) }
    sort.Strings(keys)

    for _, k := range keys { fmt.Fprintf(out, "#%s :{(configure)}\n", k) }

    fmt.Fprintf(out, "\n")
    fmt.Fprintf(out, "%s:{(configure -check)}\\\n", ctx.target)
    for _, k := range keys { fmt.Fprintf(out, "  %s \\\n", k) }
    fmt.Fprintf(out, "\n")
    return
}
