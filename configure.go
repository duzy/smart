//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
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

var configureOps = map[string] func(*modifier_configure, Value, ...Value) (Value) {
    "a":       (*modifier_configure)._answer,
    "answer":  (*modifier_configure)._answer,
    "b":       (*modifier_configure)._bool,
    "bool":    (*modifier_configure)._bool,
    "boolean": (*modifier_configure)._bool,
    "val":     (*modifier_configure)._value,
    "value":   (*modifier_configure)._value,
    "o":       (*modifier_configure)._option,
    "opt":     (*modifier_configure)._option,
    "option":  (*modifier_configure)._option,
    "pkg":     (*modifier_configure)._package,
    "package": (*modifier_configure)._package,
}

func _configurecontext(c Context) *configurecontext { return cast[*configurecontext](c) }
func isConfigure(ctx Context) bool { return _configurecontext(ctx) != nil }

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
func (cc *configurecontext) cast(t reflect.Type) Context { return implcast(cc, t) }
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
func (cc *configurecontext) execute(ctx Context, e entry) {
    var d *def
    var s string
    var p = e.owner()

    if checkpoints && truly(ctx, is_test_mode{}) {
        defer cc.execute_check(ctx, e, p, &s, &d)
    }

    if p != cc.current && p != nil {
        if p.configured { return } // already configured

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

    s = e.destiny().string(ctx)
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

func (u *universe) config(cal func(*project, entry), pre func(*project) func(), inf func(*project)) {
    var m = make(map[*project]struct{})
    var f func(*project)

    f = func(p *project) {
        if _, y := m[p]; y { return } else { m[p] = struct{}{} }

        if pre != nil { if f := pre(p); f != nil { defer f() }}

        for _, base := range p.bases { f(base) }

        if inf != nil { inf(p) }

        for _, t := range p.use.list { f(t.project) }

        if cal != nil { for _, e := range p.configs { cal(p, e) } }
    }

    f(u.globe.main)
}

func promptEnteringDirectory(ctx Context, s string) diagtracer {
    return prompt(ctx, "smart: Entering directory '%s'\n", s)
}

func promptLeavingDirectory(ctx Context, s string) (res diagtracer) {
    res = prompt(ctx, "smart: Leaving directory '%s'\n", s)
    if true { flush(ctx) }
    return
}

type configure_silent struct{}

func configure(ctx Context, ii ...any) {
    var cc = configurecontext{Context:ctx, done:make(map[*def]struct{},8)}

    defer cc.close()

    for _, i := range ii {
        switch i.(type) {
        case configure_silent: cc.silent = true
        }
    }

    u := _universe(ctx)

    // Remove existing configuration.sm files
    u.config(nil, nil, func(p *project) {
        if f := p.configuration_sm(ctx); f != nil { os.Remove(f.fullname()) }
    })

    u.config(func(p *project, entry entry) {
        cc.execute(ctx, entry)
    }, func(p *project) (f func()) {
        if !cc.silent && p.configure != nil && !p.configured && len(p.configs) > 0 {
            /********/ { promptEnteringDirectory(ctx, p.absPath) }
            f = func() { promptLeavingDirectory(ctx, p.absPath) }
        }
        return
    }, func(p *project) {
        if !p.configured && p.configure != nil && p.configure.defaultEntry != nil {
            p.configure.defaultEntry.execute(ctx)
        }
    })
    return
}

// configure - configures a variable, example usage:
//     (configure -answer)
//     (configure -option(info='...'))
//     (configure -package(xxx))
//     (configure -include('xxx.h'))
//     (configure -function(function,include='<xxx.h>'))
//     (configure -library(lib,function))
//     (configure -library(lib,function,include='<xxx.h>'))
//     (configure -symbol(symbol,include='<xxx.h>'))
//     (configure -compiles(info="..."))
type modifier_configure struct { modifier_
    accumulate bool `add,acc,accumulate`
}

func (ctx *modifier_configure) _param(name string, i any) *pair {
    var val Value
    var pos = _position(ctx)
    switch t := i.(type) {
    case Value: val = t
    case string: val = _strlit(pos, t)
    }
    return makePair(makeWord(pos, name), val)
}

func (ctx *modifier_configure) _value(_ Value, _ ...Value) (result Value) {
    return auto_get(ctx,"-")
}

func (ctx *modifier_configure) _boolvalue() (result bool) {
    var d Value
    if d = auto_get(ctx, "-"); d == nil { return }
    for i, v := range merge(d.expand(ctx)) {
        if v == nil { continue } else {
            result = (i == 0 || result) && v.true(ctx)
        }
        if !result { break }
    }
    return
}

// -bool
// -bool('message...')
func (ctx *modifier_configure) _bool(_ Value, params ...Value) Value {
    return makeBoolean(_position(ctx), ctx._boolvalue())
}

// -answer
// -answer('message...')
func (ctx *modifier_configure) _answer(_ Value, params ...Value) (result Value) {
    return makeAnswer(_position(ctx), ctx._boolvalue())
}

// -option
// -option('message...')
func (ctx *modifier_configure) _option(_ Value, args ...Value) (result Value) {
    if d := auto_get(ctx, "-"); d != nil {
        result = d.expand(ctx)
    } else {
        result = makeAnswer(_position(ctx), false)
    }
    return
}

// -package finds system package in a way similar to cmake.find_package
func (ctx *modifier_configure) _package(_ Value, args ...Value) (result Value) {
    var names []string
    var t packagetype = packageSmart
    for _, arg := range args {
        switch a := arg.(type) {
        case *pair:
            switch key, val := a.key.string(ctx), a.val.string(ctx); key {
            case "type":
                switch val {
                case "", "smart": t = packageSmart
                case "pkgconfig": t = packageConfig
                default:          t = packageUnknown
                    erro(ctx, "package: unknown type %v", val)
                    return
                }
            default:
                prompt(ctx, "%v: package: `%v` unknown option", key)
            }
        default:
            names = append(names, a.string(ctx))
        }
    }

    var u = _universe(ctx)
    if  u.packages == nil  {
        u.packages = make(map[string]packageinfo, 4)
    }

    for _, name := range names {
        if x, y := u.packages[name]; !y {
            var e error
            switch t {
            case packageSmart:  // TODO: x, e = loadPackageSmartInfo(pos, name)
            case packageConfig: // TODO: x, e = loadPackageConfigInfo(pos, name)
            case packageUnknown:
                prompt(ctx, "%v: package `%v`: unknown type\n", name)
            }
            if e != nil {
                return
            }
            if x.project != nil {
                u.packages[name] = x
                result = makeAnswer(_position(ctx), true)
                break
            }
        }
    }
    return
}

func scanExitStatts(err error) (n, status int) {
    switch e := err.(type) {
    case *exitstatus: n, status = 1, e.int
    default: n, _ = fmt.Sscanf(err.Error(), fmtExitStatus, &status)
    }
    return
}

type commonConfigureOpts struct {
    silent bool `silent`
    noResetHyphen bool `reset` // reset hyphen value, aka. "-"
}
func (ctx *modifier_configure) execute_entry(rule_name any, target Value, _params ...Value) (configured bool, result Value) {
    if _universe(ctx).traceConfig { defer un(l_trace(l_config, fmt.Sprintf("configure.execute_entry(%s %v)", rule_name, ctx))) }

    var entries []entry

    if p := _program(ctx); p == nil {
        errostack(ctx, 3, "needs program context to configure: %v", ctx).trace()
        return
    } else if c := p.project.configure; c == nil {
        errostack(ctx, 3, "%v: .configure not provided for %v (%s)", p.project.name, target, rule_name).trace()
        return
    } else if entries = c._entries(ctx, rule_name, false); entries == nil {
        errostack(pc(ctx,rule_name), 3, "%v: unknown configuration action : %v %v", rule_name, c.entries.ks(true), c.bases[0].entries.ks(true)).trace()
        return
    }

    var hyphen = auto_get(ctx,"-")
    var verbose = ctx.verbose
    var commopts commonConfigureOpts

    _params = parseOpts(ctx, &commopts, _params...)

    // Reset the result/output def '-'?
    // NOTE: have to reset hyphen to ensure configured value is saved
    if !commopts.noResetHyphen { auto_set(ctx, defVoid, "-", nil) }

    // verbose mode is on if silent flag was not set
    if !verbose && !commopts.silent { verbose = !commopts.silent }

    var params []Value
    var prog = entries[0].programs()[0]
    for _, par := range prog.params {
        switch par.ident(ctx) {
        case "LANG":   params = append(params, ctx._param("LANG",   _program(ctx).language))
        case "TARGET": params = append(params, ctx._param("TARGET", target))
        case "VALUE":  params = append(params, ctx._param("VALUE",  hyphen))
            if hyphen == nil { warn(ctx, "nil hyphen def").debug() }
        }
    }

paramsloop:
    for _, a := range _params {
        var p, y = a.(*pair)
        if !y {
            erro(pc(ctx,a), "unsupported parameter %s", tv(a)).trace()
            return
        }

        var key, value = p.key.string(ctx), p.val
        if _, y = value.(*strcomp); y {
            value = _strlit(_position(ctx), value.string(ctx))
        } else if value != nil {
            value = value.expand(ctx)
        }

        for _, par := range prog.params {
            if s := par.ident(ctx); s == key || s == strings.ToUpper(key) {
                params = append(params, ctx._param(par.ident(ctx), value))
                continue paramsloop
            }
        }

        if key == "INFO" {
            if false && verbose { prompt(ctx, "%s", p.val) }
        } else if true {
            var ps []string
            for _, p := range prog.params { ps = append(ps, p.ident(ctx)) }

            var t = auto_get(ctx,"@")
            warn(ctx, "ignored param: {=%s %v}; target: {=%s %v}", typeof(a), a, typeof(t), t)
            warn(ctx, "%v params = %v", t, ps).debug(16)
            return
        }
    }

    for _, e := range entries {
        if t := e.execute(ctx, params...); 0 < len(t) { result = t[0] }
    }

    configured = true
    return
}

func (ctx *modifier_configure) execute(target, name Value, args []Value) (configured bool, result Value) {
    if _universe(ctx).traceConfig { defer un(l_trace(l_config, "configure.execute")) }

    var opName string
    if x, y := name.(flag); y {
        opName = x.Value.string(ctx)
    } else {
        opName = name.string(ctx)
    }
    if opName == "" {
        erro(pc(ctx,name), "empty configure name: %v", ts(name)).trace()
    }

    if checkpoints && truly(ctx, is_test_mode{}) {
        defer ctx.execute_check(target, name, opName, args, &configured, &result)
    }

    var cc Context = ctx
    if ctx.final { cc = final{ctx} } // TODO: expandPathStr

    var params, infos []Value
    for _, arg := range xmerge(cc, args...) {
        switch t := arg.(type) {
        case *raw, *strlit, *strcomp:
            params = append(params, ctx._param("INFO", t))
            infos = append(infos, t)
        case *pair:
            params = append(params, t)
        default:
            if !isTrivial(arg) {
                erro(pc(ctx,arg), "unsupported configure parameter: %v", tv(arg)).trace()
            }
        }
    }

    if count_diag(ctx, diagError) > 0 { return }

    var silent = truly(ctx, silent_configure{})
    if !silent {
        if len(infos) == 0 {
            var a any = opName
            if len(args) > 0 { a = args }
            prompt(ctx, "%v %v …", target, a)
        } else {
            var s string
            for _, info := range infos { s += info.string(ctx) }
            if s != "" { prompt(ctx, "%s …", s) }
        }
    }

    if x, y := configureOps[opName]; y {
        configured, result = (x != nil), x(ctx, target, params...)
    } else {
        configured, result = ctx.execute_entry(name, target, params...)
    }

    if !silent {
        if count_diag(ctx, diagInfo, diagWarn, diagError) > 0 {
            return
        } else if result == nil {
            prompt(ctx, "… <nil>\n")
        } else if isNull(result) {
            prompt(ctx, "… <null>\n")
        } else if isNone(result) {
            prompt(ctx, "… <none>\n")
        } else if s := result.string(ctx); s == "" {
            prompt(ctx, "… ? (%v(%v))\n", typeof(result), result)
        } else {
            prompt(ctx, "… %v\n", s)
        }
    }
    return
}

func (ctx *modifier_configure) x(ops ...Value) (result any) {
    var u = _universe(ctx)
    if u.traceConfig {
        defer un(l_trace(l_config, fmt.Sprintf("modifierConfigure(%v) (reconfig=%v)", ctx, u.reconfigure)))
    }

    var proj = _project(ctx)
    var prog = _program(ctx)
    if proj == nil {
        erro(ctx, "no project to configure: %v", ctx).trace()
    }
    if prog == nil {
        erro(ctx, "no program to configure: %v", ctx).trace()
    }

    if proj.configure == nil {
        if proj.name == "configure" {
            if o := proj.Lookup(dot_configure); o != nil {
                if d, y := o.(*def); y && d.value != nil && !isTrivial(d.value) {
                    if val := d.value.true(ctx); val {
                        if proj.configure = proj; ctx.verbose {
                            info(ctx, "self-configure project enabled: %v", proj).debug()
                        }
                    }
                }
            }
        }
        if proj.configure == nil {
            erro(ctx, "%v: .configure not provided", proj).trace()
        }
    }

    var target = auto_get(ctx, "@")
    if target == nil {
        erro(ctx, "target is trivial: %s", ctx).trace()
    }

    var name = target.string(ctx)
    if name == "" {
        erro(ctx, "target is empty: %v: %v", typeof(target), target).trace()
    }

    var d *def
    if d = proj.finddef(name); d == nil {
        d, _ = proj.set(ctx, name, defConfig)
    }
    if d == nil {
        erro(ctx, "cannot define configuration `%s`", name).trace()
    }

    if checkpoints && truly(ctx, is_test_mode{}) {
        defer ctx.x_check(proj, d, &result)
    }

    result = d

    if !isNull(d.value) { // Check if it's already configured?
        if !u.reconfigure { return } // return if not reconfigure
        if p := _configurecontext(ctx); p != nil {
            if _, y := p.done[d]; y { return }
        }
    }

    var value Value
    if len(ops) == 0 { // Empty configuration: (configure)
        if value = auto_get(ctx,"-"); value == nil || value == d || value.refs(ctx, d) {
            return
        }

        switch v := value.(type) {
        case *exec_result:
            var s string
            if /*v.wg.Wait()*/; v.Status == 0 && v.Stdout.Buf != nil {
                s = v.Stdout.Buf.String()
            } else if v.Stderr.Buf != nil {
                s = v.Stderr.Buf.String()
            }
            value = _strlit(_position(ctx), s)
        }

        d.set(ctx, defConfig, value)
        return
    } else {
        d.set(ctx, defConfig, nil)
    }

    var configured bool
    var cc = _configurecontext(ctx)

    for i, a := range ops {
        if d.value == nil && i > 0 { break }

        var name Value
        var para []Value
        switch arg := a.(type) {
        case flag: name = arg.Value
        case *argumented:
            if _, y := arg.Value.(flag); !y {
                erro(ctx, "unsupported value: %v", ts(arg.Value)).trace()
            }
            name, para = arg.Value, arg.args
        default:
            erro(ctx, "unsupported: %v", ts(a)).trace()
        }

        if name == nil {
            erro(ctx, "unknown configure `%v`", ts(a)).trace()
        }

        if d := ctx.debug; d > 0 { note(ctx, "%v: %v: %v", target, name, para).debug(d) }

        if configured, value = ctx.execute(target, name, para); !configured {
            erro(ctx, "%v: not configured with: %v", target, ts(name)).trace()
        } else if v := value; v == nil {
            value = makeNull(a.Position())
        } else if isNull(v) || isNone(v) || isUndef(v) {
            // noop
        } else if v = value.expand(ctx); v != nil && v != value {
            value = v
        }

        if value == d || (value != nil && value.refs(ctx, d)) {
            // Value is the Def, does nothing!
        } else if ctx.accumulate {
            d.append(ctx, value)
        } else {
            d.set(ctx, defConfig, value)
        }

        if d == nil { cc.done[d] = struct{}{} }
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
            if matched, _, _ = p.match(ctx, path); !matched {
                matched, _, _ = p.match(ctx, filepath.Base(path))
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
        return fn(stat(ctx, rel, stat_dir{root}, stat_fileinfo{info}), err)
    })
}

var configuredFiles = make(map[string]*scope, 8)

type (
    configureconvertArgs func([]Value, *bytes.Buffer) []Value
    configureconvertFunc func(string, *bytes.Buffer)
    configureconvertOpts struct {
        generalOpts
        mode     os.FileMode `mode`
        makePath bool `path`
        mustConf bool `mustconfig,mustconf,must-conf,must-config,nc,needsconfig,needs-config`
        reconfig bool `reconfig`
        update   bool `update`
    }
)
func configureconvert(ctx Context, dealArgs configureconvertArgs, dealData configureconvertFunc, opts *configureconvertOpts, args ...Value) (_ Value) {
    var (
        closured = closure_projects(ctx)
        project = _project(ctx)
        filename string
        f *file
        target as
    )

    args = parseOpts(ctx, opts, args...)

    if target.Value = auto_get(ctx, "@"); isTrivial(target.Value) {
        erro(ctx, "'@' is not defined").trace()
    } else if f, filename, _ = target.fullnameFile(ctx, closured...); f == nil {
        if depend := auto_get(ctx,">"); !isTrivial(depend) {
            panic(traveTargetNotDefinedFile)
        } else if true {
            prompt(ctx, "%v: not defined as file\n", target.string(ctx))
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

    if f.info == nil { if f := stat(ctx, filename); f != nil { f.info = f.info }}
    if f != nil && 0 < opts.debug {
        info(ctx, "configure-file: %v: %v (%s) (%v)", auto_get(ctx,"@"), f.fullname(), closured).debug(opts.debug)
    }

    if len(project.configs) == 0 {
        // no need to check configuration
    } else if f := project.configuration_sm(ctx); f == nil || !f.exists() {
        prompt(ctx, "%v\n", filename)
        if opts.mustConf {
            var d = opts.debug ; if d == 0 { d = 1 }
            errostack(ctx, opts.stack, "no configuration (%v), try -conf first, in %v", f, project).trace()
        } else if true {
            warnstack(ctx, opts.stack, "no configuration (%v), try -conf first, in %v", f, project).debug(opts.debug)
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
            if str := arg.string(ctx); str == "" {
                continue
            } else {
                dealData(str, &data)
            }
        }
    }

    if data.Len() == 0 {
        prompt(ctx, "%v: %v %v\n", filename, auto_get(ctx,"@"), auto_get(ctx,">"))
        errostack(ctx, 5, "empty configuration data").trace()
    } else if f := _project(ctx).configuration_sm(ctx); (f == nil || !f.exists()) && opts.debug>0 {
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
            for _, d := range merge(_execution(ctx).targets...) {
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
            var name = a.string(ctx)
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

    return configureconvert(ctx, dealArgs, nil, &opts, args...)
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
    return configureconvert(ctx, nil, convert, &opts, args...)
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
        outFile = target.string(ctx)
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
            var s = d.string(ctx)
            err = walkFiles(ctx, s, pats, func(file *file, err error) error {
                if err == nil { sources = append(sources, file) }
                return err
            })
        default:
            var s = d.string(ctx)
            var dir = filepath.Dir(s)
            var name = filepath.Base(s)
            var f = stat(ctx, name, stat_dir{dir})
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
        default:    s = t.string(ctx)
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
