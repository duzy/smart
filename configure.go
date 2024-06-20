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

var testConfigurationDiverged bool
var testPromptConfiguration bool

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

type configurecontext struct {
    Context
    current *project
    silent bool
    file *os.File
    writer *bufio.Writer
    defs map[string]struct{}
    done map[*def]struct{}
}
func (ctx *configurecontext) cast(t reflect.Type) Context { return implcast(ctx, t) }
func (ctx *configurecontext) openConfigurationFile(p *project) (file *os.File) {
    if f := p.configuration; f == nil {
        erro(ctx, "%p: nil configuration file", p).debug()
        trace(ctx)
    } else if s := f.fullname(); s == "" {
        erro(ctx, "empty configuration file name: %v", f).debug()
        trace(ctx)
    } else if err := os.MkdirAll(filepath.Dir(s), os.FileMode(0755)); err != nil {
        erro(ctx, "make path %s failed: %v", filepath.Dir(s), err).debug()
        trace(ctx)
    } else if file, err = os.OpenFile(s, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0600)); err != nil {
        erro(ctx, "open configuration %s failed: %v", s, err).debug()
        trace(ctx)
    } else if testPromptConfiguration {
        prompt(ctx, "%s:1: %v\n", s, p)
        note(ctx, "%v", p).debug(16)
    } else if testConfigurationDiverged || true {
        return
    } else if t := p._configuration(ctx); t != nil && t != f && t.fullname() != f.fullname() {
        erro(ctx, "%v: diverged configuration file (%v)", p, _project(ctx))
        prompt(ctx, "%v:1: <--- at load-time\n", t.fullname())
        prompt(ctx, "%v:1: <--- at configure-time\n", f.fullname()).debug()
        trace(ctx)
    }

    return
}
func (ctx *configurecontext) execute(entry entry) {
    if p := entry.owner(); p != ctx.current && p != nil {
        if p.configured { return } // already configured

        ctx.defs = make(map[string]struct{}) // reset defs for p

        // NOTE: configuration.sm is created for every project
        var f = ctx.openConfigurationFile(p)
        if f != nil {
            if ctx.writer != nil {
                if e := ctx.writer.Flush(); e != nil {
                    erro(ctx, "%v", e).debug()
                    trace(ctx)
                }
            }
            if ctx.file != nil {
                if e := ctx.file.Close(); e != nil {
                    erro(ctx, "%v", e).debug()
                    trace(ctx)
                }
            }
        }

        ctx.file, ctx.writer = f, bufio.NewWriter(f)
        fmt.Fprintf(ctx.writer, "# %s (%s) configuration\n", p.spec, p.rel)

        if !ctx.silent {
            if false && p.name == "lib.c++.inc" { note(ctx, "%v", p.spec).debug(16) }
            prompt(ctx, "configure %s …… (%s)\n", p.name, p.spec)
        }

        ctx.current = p
    }

    entry.execute(ctx)

    var s = entry.destiny().string(ctx)
    if _, y := ctx.defs[s]; y { return }

    var d = ctx.current.findDef(s)
    if d == nil {
        erro(ctx, "%v: `%s` not configured", ctx.current, s).debug()
        return
    }

    ctx.defs[s] = struct{}{}

    if d.value == nil {
        // Set <nil> value with exec-assign ('!=') to a None value.
        fmt.Fprintf(ctx.writer, "%v !=\n", d.ident(ctx))
    } else {
        fmt.Fprintf(ctx.writer, "%v = %v\n", d.ident(ctx), d.value.String())
    }
    return
}
func (ctx *configurecontext) close() {
    if ctx.writer != nil { if err := ctx.writer.Flush(); err != nil {} }
    if ctx.file != nil   { if err := ctx.file.Close();   err != nil {} }
}

func (u *universe)  forConfigs(cal func(*project, entry)) { u._forConfigs(cal, nil, nil) }
func (u *universe) _forConfigs(cal func(*project, entry), pre func(*project) func(), inf func(*project)) {
    var m = make(map[*project]struct{}, 4)
    var f func(*project)

    f = func(p *project) {
        if _, y := m[p]; y { return } else { m[p] = struct{}{} }

        if pre != nil { if f := pre(p); f != nil { defer f() }}

        for _, base := range p.bases { f(base) }

        if inf != nil { inf(p) }

        for _, u := range p.use.list { f(u.project) }

        if cal != nil { for _, e := range p.configs { cal(p, e) } }
    }

    f(u.globe.main)
}

func promptEnteringDirectory(ctx Context, s string) *diagpoint {
    return prompt(ctx, "smart: Entering directory '%s'\n", s)
}

func promptLeavingDirectory(ctx Context, s string) *diagpoint {
    return prompt(ctx, "smart: Leaving directory '%s'\n", s)
}

type configure_silent struct{}

func configure(ctx Context, ii ...interface{}) {
    var u = _universe(ctx)
    var c = configurecontext{
        Context: ctx, done: make(map[*def]struct{}, 8),
    }

    defer c.close()

    for _, i := range ii {
        switch i.(type) {
        case configure_silent: c.silent = true
        }
    }

    // Remove existing configuration.sm files
    u._forConfigs(nil, nil, func(p *project) {
        if f := p._configuration(ctx); f != nil { os.Remove(f.fullname()) }
    })

    u._forConfigs(func(p *project, entry entry) {
        c.execute(entry)
    }, func(p *project) (f func()) {
        if !c.silent && p.configure != nil && !p.configured && len(p.configs) > 0 {
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

func (ctx *modifier_configure) _param(name string, i interface{}) *pair {
    var val Value
    var pos = _position(ctx)
    switch t := i.(type) {
    case Value: val = t
    case string: val = makeStrlit(pos, t)
    }
    return makePair(makeBareword(pos, name), val)
}

// -value
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
    for _, arg := range args { switch a := arg.(type) {
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
    }}

    var u = _universe(ctx)
    if  u.packages == nil {
        u.packages = make(map[string]packageinfo, 4)
    }
    for _, name := range names { if info, y := u.packages[name]; !y {
        var err error
        switch t {
        case packageSmart: // TODO: info, err = loadPackageSmartInfo(pos, name)
        case packageConfig: // TODO: info, err = loadPackageConfigInfo(pos, name)
        case packageUnknown:
            prompt(ctx, "%v: package `%v`: unknown type\n", name)
        }
        if err != nil { return } else if info.project != nil {
            _universe(ctx).packages[name] = info
            result = makeAnswer(_position(ctx), true)
            break
        }
    }}
    return
}

func scanExitStatts(err error) (n, status int) {
    switch e := err.(type) {
    case *exitstatus: n, status = 1, e.code
    // case *scanner.Error:
    //     for _, t := range e.Errs {
    //         if n, status = scanExitStatts(t); n == 1 { return }
    //     }
    default:
        n, _ = fmt.Sscanf(err.Error(), fmtExitStatus, &status)
    }
    return
}

type commonConfigureOpts struct {
    silent bool `silent`
    noResetHyphen bool `reset` // reset hyphen value, aka. "-"
}
func (ctx *modifier_configure) executeEntry(entryName interface{}, target Value, paramsOrig ...Value) (configured bool, result Value) {
    if _universe(ctx).traceConfig { defer un(l_trace(l_config, fmt.Sprintf("configureExecuteEntry(%s %v)", entryName, ctx))) }

    var entries []entry
    if program := _program(ctx); program == nil {
        errostack(ctx, 3, "needs program context to configure: %v", ctx).debug(16)
        return
    } else if program.project.configure == nil {
        errostack(ctx, 3, "%v: .configure not provided for %v (%s)", program.project, target, entryName).debug(16)
        return
    } else if entries = program.project.configure.resolveEntries(ctx, entryName, false); entries == nil {
        errostack(ctx, 3, "%T %v: unknown configuration action", entryName, entryName).debug(16)
        return
    }

    var (
        hyphen = auto_get(ctx,"-")
        verbose = ctx.verbose

        params []Value
        commOpts commonConfigureOpts
    )

    paramsOrig = parseOpts(ctx, &commOpts, paramsOrig...)

    // Reset the result/output def '-'?
    // NOTE: have to reset hyphen to ensure configured value is saved
    if !commOpts.noResetHyphen { auto_set(ctx, "-", nil) }

    // verbose mode is on if silent flag was not set
    if !verbose && !commOpts.silent { verbose = !commOpts.silent }

    var prog = entries[0].programs()[0]
    for _, par := range prog.params {
        switch par.ident(ctx) {
        case "LANG":   params = append(params, ctx._param("LANG",   _program(ctx).language))
        case "TARGET": params = append(params, ctx._param("TARGET", target))
        case "VALUE":  params = append(params, ctx._param("VALUE",  hyphen))
            if hyphen == nil { warn(ctx, "nil hyphen def").debug() }
        }
    }

ForInParams:
    for _, a := range paramsOrig {
        var p, y = a.(*pair)
        if !y {
            erro(at(ctx,a), "unsupported parameter %v (%T)", a, a).debug()
            return
        }

        var key, value = p.key.string(ctx), p.val
        if _, y = value.(*compound); y {
            value = makeStrlit(_position(ctx), value.string(ctx))
        } else if value != nil {
            value = value.expand(ctx)
        }

        for _, par := range prog.params { if s := par.ident(ctx); s == key || s == strings.ToUpper(key) {
            params = append(params, ctx._param(par.ident(ctx), value))
            continue ForInParams
        }}

        if key == "INFO" {
            if false && verbose { prompt(ctx, "%s", p.val) }
        } else if true {
            var params []string
            for _, p := range prog.params { params = append(params, p.ident(ctx)) }

            var t = auto_get(ctx,"@")
            ctx.Context = at(ctx.Context, a)
            warn(ctx, "ignored param: %T %v; target: %T %v", a, a, t, t)
            warn(at(ctx,prog.position), "%v params = %v", t, params).debug(16)
            return
        }
    }

    for _, entry := range entries {
        var reses = entry.execute(ctx, params...)
        if len(reses) > 0 { result = reses[0] }
    }

    configured = true
    return
}

func (ctx *modifier_configure) execute(target, name Value, args []Value) (configured bool, result Value) {
    if _universe(ctx).traceConfig { defer un(l_trace(l_config, "configureExecute")) }

    var opName string
    if f, y := name.(flag); y {
        opName = f.Value.string(ctx)
    } else {
        opName = name.string(ctx)
    }
    if opName == "" {
        erro(ctx, "empty configure name: %v", ts(name)).debug()
        trace(ctx)
    }

    var params, infos []Value

    if d := ctx.debug; d > 0 { defer func() {
        note(ctx, "%v: %v -> %v %v", ts(target), args, infos, params).debug(1+d)
    }()}

    var cc Context = ctx
    if ctx.force { cc = final{ctx} } // TODO: expandPathStr

    for _, arg := range xmerge(cc, args...) {
        if !isTrivial(arg) { switch t := arg.(type) {
        case *raw, *strlit, *compound:
            params, infos = append(params, ctx._param("INFO", t)), append(infos, t)
        case *pair:
            params = append(params, t)
        default:
            erro(at(ctx,arg), " unsupported parameter: %v{%v}, %v{%v}", typeof(t), t, typeof(arg), arg).debug()
            trace(ctx)
        }}
    }

    var silent bool
    if p := _configurecontext(ctx); p != nil { silent = p.silent }

    if silent {
        // silent
    } else if len(infos) == 0 {
        var a interface{} = opName; if len(args) > 0 { a = args }
        prompt(ctx, "%v %v …", target, a)
    } else {
        var s string
        for _, info := range infos { s += info.string(ctx) }
        if s != "" { prompt(ctx, "%s …", s) }
    }

    var dia = _diagnostic(ctx)
    if dia.error() { return }

    defer func() {
        if silent {
            return
        } else if dia.count(diagInfo, diagWarn, diagError) > 0 {
            return
        } else if false && dia.points != nil { t := true
            for _, d := range dia.points {
                if strings.HasSuffix(d.message, "…") { t = false; break }
            }
            if t { return }
        }
        if result == nil {
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
    } ()

    if configureOp, y := configureOps[opName]; y {
        configured, result = true, configureOp(ctx, target, params...)
    } else {
        configured, result = ctx.executeEntry(name, target, params...)
    }
    return
}

func (ctx *modifier_configure) x(ops ...Value) (result interface{}) {
    var u = _universe(ctx)
    if u.traceConfig { defer un(l_trace(l_config, fmt.Sprintf("modifierConfigure(%v) (reconfig=%v)", ctx, u.reconfigure))) }

    var project = _project(ctx)
    var program = _program(ctx)
    if project == nil {
        erro(ctx, " no project to configure: %v", ctx).debug()
        trace(ctx)
    }
    if program == nil {
        erro(ctx, " no program to configure: %v", ctx).debug()
        trace(ctx)
    }

    if project.configure == nil {
        if project.name == "configure" {
            if o := project.Lookup(dotConfigure); o != nil {
                if d, y := o.(*def); y && d.value != nil && !isTrivial(d.value) {
                    if val := d.value.true(ctx); val {
                        if project.configure = project; ctx.verbose {
                            info(ctx, "self-configure project enabled: %v", _project(ctx)).debug()
                        }
                    }
                }
            }
        }
        if project.configure == nil {
            erro(ctx, " %v: .configure not provided", project).debug()
            trace(ctx)
        }
    }

    var target = auto_get(ctx, "@")
    if target == nil {
        erro(ctx, " target is trivial: %s", ctx).debug()
        trace(ctx)
    }

    var name = target.string(ctx)
    if name == "" {
        erro(ctx, " target is empty: %v: %v", typeof(target), target).debug()
        trace(ctx)
    }

    var d *def
    if d = project.findDef(name); d == nil {
        d, _ = project.set(ctx, name, defConfig)
    }
    if d == nil {
        erro(ctx, " cannot define configuration `%s`", name).debug()
        trace(ctx)
    }

    if result = d; !isNull(d.value) { // Check if it's already configured?
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
        case *execResult:
            var s string
            if /*v.wg.Wait()*/; v.Status == 0 && v.Stdout.Buf != nil {
                s = v.Stdout.Buf.String()
            } else if v.Stderr.Buf != nil {
                s = v.Stderr.Buf.String()
            }
            value = makeStrlit(_position(ctx), s)
        }

        d.set(ctx, defConfig, value)
        return
    } else {
        d.set(ctx, defConfig, nil)
    }

    var configured bool
    var ce = _configurecontext(ctx)

ForConfig:
    for i, a := range ops {
        if d.value == nil && i > 0 { break ForConfig }

        var name Value
        var para []Value
        switch arg := a.(type) {
        case flag: name = arg.Value
        case *argumented:
            if _, y := arg.Value.(flag); !y {
                erro(at(ctx,a), "unsupported value: %v", ts(arg.Value)).debug()
                trace(ctx)
            }
            name, para = arg.Value, arg.args
        default:
            erro(at(ctx,a), "unsupported: %v", ts(a)).debug()
            trace(ctx)
        }

        if name == nil {
            erro(at(ctx,a), "unknown configure `%v`", ts(a)).debug()
            trace(ctx)
        }

        if d := ctx.debug; d > 0 { note(ctx, "%v: %v: %v", target, name, para).debug(d) }

        if configured, value = ctx.execute(target, name, para); !configured {
            erro(ctx, "%v: not configured with: %v", target, ts(name)).debug()
            trace(ctx)
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

        if d == nil { ce.done[d] = struct{}{} }
    }
    return
}

type filewalkFunc func(file *File, err error) error

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
        mode     os.FileMode `m,mode`
        makePath bool `p,path`
        mustConf bool `mc,mustconfig,mustconf,must-conf,must-config,nc,needsconfig,needs-config`
        reconfig bool `r,reconfig`
        update   bool `u,update`
    }
)
func configureconvert(ctx Context, dealArgs configureconvertArgs, dealData configureconvertFunc, opts *configureconvertOpts, args ...Value) (_ Value) {
    var (
        closured = closure_projects(ctx)
        project = _project(ctx)
        filename string
        f *File
        target as
    )

    args = parseOpts(ctx, opts, args...)

    if target.Value = auto_get(ctx, "@"); isTrivial(target.Value) {
        erro(ctx, "'@' is not defined").debug()
        trace(ctx)
    } else if f, filename, _ = target.fullnameFile(ctx, closured...); f == nil {
        if depend := auto_get(ctx,">"); !isTrivial(depend) {
            panic(traveTargetNotDefinedFile)
        } else if true {
            prompt(ctx, "%v: not defined as file\n", target.string(ctx))
            erro(ctx, "%v", ts(target.Value))
            errostack(ctx, 8).debug()
            trace(ctx)
        }
        return
    } else if filename == "" {
        errostack(ctx, 3, "%v: empty fullname: `%v`", target.Value, file).debug()
        trace(ctx)
    }

    if _, prev := auto_set(ctx, "@", f); opts.debug>0 {
        info(ctx, "configure-file: %s->%s (%v -> %v)", f, filename, ts(prev), ts(f)).debug(opts.debug)
    }

    if f.info == nil { if f := stat(ctx, filename); f != nil { f.info = f.info }}
    if f != nil && 0 < opts.debug {
        info(ctx, "configure-file: %v: %v (%s) (%v)", auto_get(ctx,"@"), f.fullname(), closured).debug(opts.debug)
    }

    if len(project.configs) == 0 {
        // no need to check configuration
    } else if f := project._configuration(ctx); f == nil || !f.exists() {
        prompt(ctx, "%v: %v\n", filename, file)
        if opts.mustConf {
            var d = opts.debug ; if d == 0 { d = 1 }
            errostack(ctx, opts.stack, "no configuration (%v), try -conf first, in %v", f, project).debug(d)
            trace(ctx)
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
        errostack(ctx, 5, "empty configuration data").debug()
        trace(ctx)
    } else if f := _project(ctx)._configuration(ctx); (f == nil || !f.exists()) && opts.debug>0 {
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
            erro(ctx, " crc64 checksum failed: %v", e).debug()
            return
        }
        if same {
            var tt = f.info.ModTime()
            for _, d := range merge(_execution(ctx).targets...) {
                if f, y := toFile(d); !y { continue } else
                if dt := f.info.ModTime(); dt.After(tt) { tt = dt }
            }
            if tt.After(f.info.ModTime()) { e = touch(ctx, f, 0, false, tt) }
            return f
        }
    } else if dir := filepath.Dir(filename); opts.makePath && dir != "." && dir != pathSep {
        if e = os.MkdirAll(dir, os.FileMode(0755)); e != nil {
            erro(ctx, " %v", e).debug()
            trace(ctx)
        }
    }

    if e = ioutil.WriteFile(filename, data.Bytes(), opts.mode); e != nil {
        erro(ctx, " %v", e).debug()
        trace(ctx)
    }

    if f.info == nil {
        if f.info, e = os.Stat(filename) ; e == nil {
            erro(ctx, " %v", e).debug()
            trace(ctx)
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
func (ctx *modifier_configureinput) x(args ...Value) (result interface{}) {
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
                erro(ctx, "undefined %v", name).debug()
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
func (ctx *modifier_configurefile) x(args ...Value) (result interface{}) {
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
    mode os.FileMode "m,mode"
    makePath bool "p,path"
    target string "t,target"
    rxs []*regexp.Regexp "r,rx,regex" // regexp.Compile(s)
}
func (ctx *modifier_extractconfiguration) x(args ...Value) (result interface{}) {
    var pos = _position(ctx)
    var pats []Value
    for _, arg := range args {
        switch a := arg.(type) {
        case *group: pats = append(pats, a.elems...)
        default:     pats = append(pats, a)
        }
    }
    if len(pats) == 0 {
        erro(ctx, "extract-configuration: missing file names (patterns)").debug()
        return
    }
    if len(ctx.rxs) == 0 {
        erro(ctx, "extract-configuration: missing -rx=... flags").debug()
        return
    }
    if ctx.target == "" {
        ctx.target = "configuration"
    }

    var outFile string
    if target := auto_get(ctx,"@"); isNull(target) {
        erro(ctx, " target '@' is undefined").debug()
        return
    } else {
        outFile = target.string(ctx)
    }

    if ctx.makePath {
        if err := os.MkdirAll(filepath.Dir(outFile), os.FileMode(0755)); err != nil {
            erro(ctx, " make path failed: %v", err).debug()
            return
        }
    }

    var (
        err error
        fil *os.File
        out *bufio.Writer
    )
    if fil, err = os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, ctx.mode); err != nil {
        erro(ctx, " open file failed: %v", err).debug()
        return
    } else {
        out = bufio.NewWriter(fil)
    }
    defer func() {
        out.Flush()
        fil.Close()
    }()

    var depends, sources []Value
    if d := auto_get(ctx, "^"); !isTrivial(d) {
        depends = xmerge(ctx, d)
    }

    var patsVal = ease(ctx, pats)
    for _, depend := range depends {
        var a []Value
        switch d := depend.(type) {
        case *File:
            if a = merge(call(ctx, "filter", nil, patsVal, d)); a != nil {
                sources = append(sources, a...)
            }
        case *path:
            var s = d.string(ctx)
            err = walkFiles(ctx, s, pats, func(file *File, err error) error {
                if err == nil { sources = append(sources, file) }
                return err
            })
        default:
            var s = d.string(ctx)
            dir := filepath.Dir(s)
            name := filepath.Base(s)
            file := stat(ctx, name, stat_dir{dir})
            if file == nil {
                erro(ctx, " extract-configuration: `%s` file not found", name).debug()
                return
            } else if file.info.IsDir() {
                err = walkFiles(ctx, s, pats, func(file *File, err error) error {
                    if err == nil { sources = append(sources, file) }
                    return err
                })
            } else if a = merge(call(ctx, "filter", nil, patsVal, d)); a != nil {
                sources = append(sources, a...)
            }
        }
    }

    var exprs = make(map[string]int)

ForSources:
    for _, source := range sources {
        var (s string; f *os.File)
        switch v := source.(type) {
        case *File: s = v.fullname()
        default: s = v.string(ctx)
        }
        if f, err = os.Open(s); err != nil {
            prompt(ctx, "%v: (configure) %v: %v\n", pos, source, err)
            continue ForSources
        }

        scanner := bufio.NewScanner(f)
        scanner.Split(bufio.ScanLines)
        for scanner.Scan() { s := scanner.Text()
        ForOpts:
            for _, x := range ctx.rxs {
                sm := x.FindStringSubmatch(s)
                if sm == nil { continue }
                exprs[sm[1]] += 1
                break ForOpts
            }
        }
        f.Close()
    }

    var keys []string
    for x, n := range exprs {
        if n == 0 { continue }
        keys = append(keys, x)
    }

    sort.Strings(keys)

    for _, k := range keys {
        fmt.Fprintf(out, "#%s :[(configure)]\n", k)
    }

    fmt.Fprintf(out, "\n")
    fmt.Fprintf(out, "%s:[(configure -check)]:\\\n", ctx.target)
    for _, k := range keys { fmt.Fprintf(out, "  %s \\\n", k) }
    fmt.Fprintf(out, "\n")
    return
}
