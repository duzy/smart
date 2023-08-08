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
    "regexp"
    "bufio"
    "bytes"
    "sort"
    "time"
    "fmt"
    "os"
)

var configureOps = map[string] func(Context, Value, ...Value) (Value) {
    "a":       configureAnswer,
    "answer":  configureAnswer,
    "b":       configureBool,
    "bool":    configureBool,
    "boolean": configureBool,
    "v":       configureValue,
    "val":     configureValue,
    "value":   configureValue,
    "o":       configureOption,
    "opt":     configureOption,
    "option":  configureOption,
    "pkg":     configurePackage,
    "package": configurePackage,
}

var testPromptConfiguration bool
var testConfigurationDiverged bool

type configureExecutor struct {
    Context
    file *os.File
    writer *bufio.Writer
    defs map[string]struct{}
}
func (ctx *configureExecutor) openConfigurationFile(p *Project) (file *os.File) {
    defer ctx.dia().trace(ctx, "configuration-file")

    if f := p.configurationFile; f == nil {
        erro(ctx, "%p: nil configuration file", p).debug(1)
        return
    } else if s := f.fullname(); s == "" {
        erro(ctx, "empty configuration file name: %v", f).debug(1)
        return
    } else if err := os.MkdirAll(filepath.Dir(s), os.FileMode(0755)); err != nil {
        erro(ctx, "make path %s failed: %v", filepath.Dir(s), err).debug(1)
        return
    } else if file, err = os.OpenFile(s, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0600)); err != nil {
        erro(ctx, "open configuration %s failed: %v", s, err).debug(1)
        return
    } else if testPromptConfiguration {
        prompt(ctx, "%s:1: %v\n", s, p)
        noted(ctx, "%v", p).debug(16)
    } else if testConfigurationDiverged || true {
        return
    } else if t := p.configuration(ctx); t != nil && t != f && t.fullname() != f.fullname() {
        erro(ctx, "%v: diverged configuration file (%v)", p, ctx.Project())
        prompt(ctx, "%v:1: <--- at load-time\n", t.fullname())
        prompt(ctx, "%v:1: <--- at configure-time\n", f.fullname()).debug(1)
    }
    return
}
func (ctx *configureExecutor) execute(project *Project, entry Entry) (result *Project, okay bool) {
    defer ctx.dia().trace(ctx, "execute")

    if p := entry.OwnerProject(); p != project && p != nil {
        if p.configured { return nil, true } // already configured

        if false {
            var ss = []*Scope{ p.scope }
            if c := p.configure; c != nil { ss = append(ss, c.scope) }
            ctx.Context = closureWith(ctx.Context, ss...)
        }

        ctx.defs = make(map[string]struct{}) // reset defs for p

        // NOTE: configuration.sm is created for every project
        var f = ctx.openConfigurationFile(p)
        if f != nil {
            if ctx.writer != nil { if e := ctx.writer.Flush(); e != nil {
                erro(ctx, "%v", e).debug(1)
                return
            }}
            if ctx.file != nil { if e := ctx.file.Close(); e != nil {
                erro(ctx, "%v", e).debug(1)
                return
            }}
        }

        ctx.file, ctx.writer = f, bufio.NewWriter(f)
        fmt.Fprintf(ctx.writer, "# %s (%s) configuration\n", p.spec, p.relPath)

        if !ctx.universe().configuration.silent {
            prompt(ctx, "Configure project %s …… (%s)\n", p.spec, p.relPath)
        }

        project = p
    }

    result = project

    if val, traves := entry.execute(ctx); traves.has(traveFail) {
        for _, s := range traves { if s.what == traveFail {
            erro(at(ctx, s.pos), "execute '%v' failed: %v", entry, s)
        } else {
            info(ctx, "%v: %v", entry, s)
        }}
        erro(ctx, "%v: %v", entry, val).debug(1)
    }

    var s = entry.Target().strval(ctx)
    if d := project.scope.FindDef(s); d != nil { okay = true // good!
        if true && testPromptConfiguration { noted(of(ctx,d), "%v: %p %v", project, d, d).debug(1) }
        if _, y := ctx.defs[s]; y { return } else { ctx.defs[s] = struct{}{} }
        if d.value == nil {
            // Set <nil> value with exec-assign ('!=') to a None value.
            fmt.Fprintf(ctx.writer, "%v !=\n", d.name(ctx))
        } else {
            var s = elemstr(ctx, d, d.value, elemNoBrace)
            fmt.Fprintf(ctx.writer, "%v = %v\n", d.name(ctx), s)
        }
    } else {
        erro(ctx, "`%s` unconfigured", s).debug(1)
    }
    return
}
func (ctx *configureExecutor) close() {
    if ctx.writer != nil { if err := ctx.writer.Flush(); err != nil {} }
    if ctx.file != nil   { if err := ctx.file.Close();   err != nil {} }
}

func (uni *universe) configure(ctx Context) {
    var project *Project

    defer ctx.dia().trace(ctx, "configure")

    // Remove all existing configuration.sm files
    if uni.cleanConf { for _, s := range uni.configuration.clean {
        if _, e := os.Stat(s); e != nil {
            if false { prompt(ctx, "%v\n", e).debug(1) }
        } else if e = os.Remove(s); e == nil {
            prompt(ctx, "Remove %s\n", s)
        } else if true {
            prompt(ctx, "Remove: %s\n", e).debug(1)
        }
    }}

    var configureInits = make(map[Entry]struct{})
    for _, entry := range uni.configuration.entries { var p = entry.OwnerProject().configure
        if p != nil && p.defaultEntry != nil { configureInits[p.defaultEntry] = struct{}{} }
    }
    for entry, _ := range configureInits {
        var /* vals */_, traves = entry.execute(ctx)
        for _, brk := range traves { if brk.what == traveFail {
            erro(of(ctx,entry), "execute '%v' failed: %v", entry, brk).debug(1)
        }}
    }

    var ce = configureExecutor{Context:ctx} ; defer ce.close()
    for _, entry := range uni.configuration.entries { var okay bool
        if project, okay = ce.execute(project, entry); !okay {
            erro(ctx, "configure '%v' failed", entry).debug(1)
            break
        }
    }

    printLeavingDirectory(ctx)
    return
}

func configureParam(ctx Context, name string, i interface{}) *pair {
    var val Value
    var pos = ctx.Position()
    switch t := i.(type) {
    case Value: val = t
    case string: val = MakeString(pos, t)
    }
    return MakePair(pos, MakeBareword(pos, name), val)
}

// -value
func configureValue(ctx Context, _ Value, _ ...Value) (result Value) {
    return autoVal(ctx,"-")
}

func configureBoolValue(ctx Context) (result bool) {
    var d Value
    if d = autoVal(ctx, "-"); d == nil { return }
    for i, v := range merge(d.expand(ctx, plain)) {
        if v == nil { continue } else {
            result = (i == 0 || result) && v.true(ctx)
        }
        if !result { break }
    }
    return
}

// -bool
// -bool('message...')
func configureBool(ctx Context, _ Value, params ...Value) Value {
    return MakeBoolean(ctx.Position(), configureBoolValue(ctx))
}

// -answer
// -answer('message...')
func configureAnswer(ctx Context, _ Value, params ...Value) (result Value) {
    return makeAnswer(ctx.Position(), configureBoolValue(ctx))
}

// -option
// -option('message...')
func configureOption(ctx Context, _ Value, args ...Value) (result Value) {
    if d := autoVal(ctx, "-"); d != nil {
        result = d.expand(ctx, plain)
    } else {
        result = makeAnswer(ctx.Position(), false)
    }
    return
}

// -package finds system package in a way similar to cmake.find_package
func configurePackage(ctx Context, _ Value, args ...Value) (result Value) {
    var names []string
    var uni = ctx.universe()
    var t packagetype = packageSmart
    for _, arg := range args { switch a := arg.(type) {
    case *pair:
        switch key, val := a.Key.strval(ctx), a.Value.strval(ctx); key {
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
        names = append(names, a.strval(ctx))
    }}
    for _, name := range names { if info, y := uni.configuration.packages[name]; !y {
        var err error
        switch t {
        case packageSmart: // TODO: info, err = loadPackageSmartInfo(pos, name)
        case packageConfig: // TODO: info, err = loadPackageConfigInfo(pos, name)
        case packageUnknown:
            prompt(ctx, "%v: package `%v`: unknown type\n", name)
        }
        if err != nil { return } else if info.Project != nil {
            uni.configuration.packages[name] = info
            result = makeAnswer(ctx.Position(), true)
            break
        }
    }}
    return
}

func scanExitStatus(err error) (n, status int) {
    switch e := err.(type) {
    case *exitstatus: n, status = 1, e.code
    // case *scanner.Error:
    //     for _, t := range e.Errs {
    //         if n, status = scanExitStatus(t); n == 1 { return }
    //     }
    default:
        n, _ = fmt.Sscanf(err.Error(), fmtExitStatus, &status)
    }
    return
}

type configureContext struct {
    Context
}
func (cc *configureContext) String() string { return fmt.Sprintf("configure{%s}", cc.Context) }
func (cc *configureContext) inner() Context { return cc.Context }
func (cc *configureContext) isConfigure() bool { return true }

type commonConfigureOpts struct {
    silent bool `s,silent`
    noResetHyphen bool `r,reset` // reset hyphen value, aka. "-"
}
type modifierConfigureOpts struct {
    generalOpts
    accumulate bool `add,acc,accumulate`
}
func configureExecuteEntry(ctx Context, opts *modifierConfigureOpts, entryName interface{}, target Value, paramsOrig ...Value) (configured bool, result Value) {
    var uni = ctx.universe()
    if uni.traceConfig { defer un(trace(t_config, fmt.Sprintf("configureExecuteEntry(%s %v)", entryName, ctx))) }

    var entries *resolvedEntries
    if program := ctx.program(); program == nil {
        errostack(ctx, 3, "needs program context to configure: %v", ctx).debug(16)
        return
    } else if program.project.configure == nil {
        errostack(ctx, 3, "%v: .configure not provided for %v (%s)", program.project, target, entryName).debug(16)
        return
    } else if entries = program.project.configure.resolveEntries(ctx, entryName, false); entries == nil {
        var t = &program.project.configure.entries
        if t.fast != nil { t, _ = t.fast["-"] }
        errostack(ctx, 3, "%T %v: unknown configuration action, %v", entryName, entryName, t).debug(16)
        return
    }

    var (
        hyphen = autoVal(ctx,"-")
        verbose = opts.verbose

        programs = entries.programs()
        prog = programs[0]

        params []Value
        commOpts commonConfigureOpts
    )
    paramsOrig = parseOpts(ctx, &commOpts, plain, paramsOrig...)

    // Reset the result/output def '-'?
    // NOTE: have to reset hyphen to ensure configured value is saved
    if !commOpts.noResetHyphen { autoSet(ctx, "-", nil) }

    // verbose mode is on if silent flag was not set
    if !verbose && !commOpts.silent { verbose = !commOpts.silent }

    for _, par := range prog.params {
        switch par.name(ctx) {
        case "LANG":   params = append(params, configureParam(ctx, "LANG",   ctx.program().language))
        case "TARGET": params = append(params, configureParam(ctx, "TARGET", target))
        case "VALUE":  params = append(params, configureParam(ctx, "VALUE",  hyphen))
            if hyphen == nil { warn(ctx, "nil hyphen def").debug(1) }
        }
    }

ForInParams:
    for _, a := range paramsOrig {
        var p, y = a.(*pair)
        if !y {
            erro(of(ctx,a), "unsupported parameter %v (%T)", a, a).debug(1)
            return
        }

        var key, value = p.Key.strval(ctx), p.Value
        if _, y = value.(*compound); y {
            value = MakeString(ctx.Position(), value.strval(ctx))
        } else if value != nil {
            value = value.expand(ctx, plain)
        }

        for _, par := range prog.params { if s := par.name(ctx); s == key || s == strings.ToUpper(key) {
            params = append(params, configureParam(ctx, par.name(ctx), value))
            continue ForInParams
        }}

        if key == "INFO" {
            if false && verbose { prompt(ctx, "%s", p.Value) }
        } else if true {
            var params []string
            for _, p := range prog.params { params = append(params, p.name(ctx)) }

            var t = autoVal(ctx,"@")
            ctx = at(ctx, a.Position())
            warn(ctx, "ignored param: %T %v; target: %T %v", a, a, t, t)
            warn(at(ctx,prog.position), "%v params = %v", t, params).debug(16)
            return
        }
    }

    ctx = &configureContext{ ctx }

    var reses []Value
    var traves travestates
    for _, entry := range entries.all {
        if false { if entry.String() == "-library-c" {
            noted(ctx, "%v: %v", entry, params).debug(1)
        }}

        reses, traves = entry.execute(ctx, params...)

        if ctx.dia().error() { return }
        if traves = traves.not(traveDone, traveRule, traveFile); traves.has() {
            for i, s := range traves { erro(ctx, "%v: %d. %v", entry, i, s) }
            erro(ctx, "%v: %d trave states", entry, len(traves)).debug(16)
            return
        }

        if n := len(reses); n == 0 {
            if false { warn(at(ctx,entry.Position()), "%v", entry).debug(1) }
        } else if result = reses[0]; result != nil && result == hyphen {
            warn(at(ctx,entry.Position()), "%v", entry)
            warn(ctx, `%v: configure yields value the same as input will be ignored: %v`, entry, result).debug(1)
            result = nil // simply discard the result as it's the same as the input (hyphen) value
        }

        if result != nil { if d, y := result.(*def); y /* && d.name == "@" */ {
            var h = autoVal(ctx, "-")
            var x = at(ctx, entry.Position())
            errostack(x, 3, "%v: invalid result: %v (%v)", entry, d, h).debug(10)
        }}
    }

    configured = true
    return
}

func configureExecute(ctx Context, opts *modifierConfigureOpts, target Value, name Value, args []Value) (configured bool, result Value) {
    var uni = ctx.universe()
    if uni.traceConfig { defer un(trace(t_config, "configureDo")) }

    var opName string
    if f, y := name.(flag); y {
        opName = f.Value.strval(ctx)
    } else {
        opName = name.strval(ctx)
    }
    if opName == "" {
        erro(ctx, "empty configure name: %v (%T)", name, name).debug(1)
        return
    }

    var params, infos []Value
    for _, arg := range xmerge(ctx, plain, args...) {
        if isTrivial(arg) { continue }
        switch t := arg.(type) {
        case *pair: params = append(params, t)
        case *raw, *String, *compound:
            params = append(params, configureParam(ctx, "INFO", t))
            infos = append(infos, t)
        default:
            erro(of(ctx,arg), " unsupported parameter: %T %v", t, t).debug(1)
            return
        }
    }

    if uni.configuration.silent {
        // silent
    } else if len(infos) == 0 {
        var a interface{} = opName; if len(args) > 0 { a = args }
        prompt(ctx, "%v %v …", target, a)
    } else {
        var s string
        for _, info := range infos { s += info.strval(ctx) }
        if s != "" { prompt(ctx, "%s …", s) }
    }

    defer func() {
        if dia := ctx.dia(); uni.configuration.silent {
            return
        } else if dia.count(diagInfo, diagWarn, diagError) > 0 {
            return
        } else if false && dia.points != nil { t := true
            for _, d := range dia.points {
                if strings.HasSuffix(d.message, "…") { t = false; break }
            }
            if t { return }
        }
        if isNull(result) {
            prompt(ctx, "… <nil>\n")
        } else if isNone(result) {
            prompt(ctx, "… <none>\n")
        } else if s := result.strval(ctx); s == "" {
            prompt(ctx, "… ? (%s %v)\n", typeof(result), result)
        } else {
            prompt(ctx, "… %v\n", s)
        }
    } ()

    if config, y := configureOps[opName]; y {
        configured, result = true, config(ctx, target, params...)
    } else {
        configured, result = configureExecuteEntry(ctx, opts, name, target, params...)
    }
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
type modifier_configure struct { modifier_ }
func (ctx *modifier_configure) x(aa ...Value) (result interface{}) {
    var uni = ctx.universe()
    if uni.traceConfig { defer un(trace(t_config, fmt.Sprintf("modifierConfigure(%v) (reconfig=%v)", ctx, uni.reconfigure))) }

    var program = ctx.program()
    if program == nil {
        erro(ctx, " needs traversal context to configure: %v", ctx).debug(1)
        return
    }

    var opts, args = _opts[modifierConfigureOpts](ctx.Context, plain, aa...)

    if program.project.configure == nil {
        if program.project.name == "configure" {
            if o := program.project.scope.Lookup(dotConfigure); !isNull(o) {
                if d, ok := o.(*def); ok && !isNull(d.value) && !isNone(d.value) {
                    if val := d.value.true(ctx); val {
                        program.project.configure = program.project
                        if opts.verbose {
                            info(ctx, "self-configure project enabled: %v", ctx.Project()).debug(1)
                        }
                    }
                }
            }
        }
        if program.project.configure == nil {
            erro(ctx, " %v: .configure not provided", program.project).debug(1)
            return
        }
    }

    var target = autoVal(ctx,"@")
    if isNull(target) {
        erro(ctx, " target is trivial: %s", ctx).debug(1)
        return
    }

    var d *def
    var name = target.strval(ctx)
    if d = program.scope.FindDef(name); d == nil {
        var alt Object
        d, alt = program.project.scope.define(ctx, DefConfig, name, nil)
        if d == nil && alt != nil { d, _ = alt.(*def) }
    }
    if d == nil {
        erro(ctx, " cannot define configuration `%s`", name).debug(1)
        return
    } else {
        result = d
    }

    if !isNull(d.value) { // Check if it's already configured?
        if !uni.reconfigure { return } // return if not reconfigure
        if done, found := uni.configuration.done[d]; done && found { return }
    }

    var value Value
    if len(args) == 0 { // Empty configuration: (configure)
        if value = autoVal(ctx,"-"); value == nil || value == d || value.refs(ctx, d) {
            return
        }

        switch v := value.(type) {
        default: d.set(ctx, DefConfig, value)
        case *execResult:
            var s string
            if /*v.wg.Wait()*/; v.Status == 0 && v.Stdout.Buf != nil {
                s = v.Stdout.Buf.String()
            } else if v.Stderr.Buf != nil {
                s = v.Stderr.Buf.String()
            }
            d.set(ctx, DefConfig, MakeString(ctx.Position(), s))
        }
        return
    } else {
        d.set(ctx, DefConfig, nil)
    }

    var configured bool
ForConfig:
    for i, a := range args {
        if d.value == nil && i > 0 { break ForConfig }

        var ( name Value ; para []Value )
        switch arg := a.(type) {
        case flag: name = arg.Value
        case *argumented:
            if flag, okay := arg.Value.(flag); !okay {
                erro(of(ctx,a), " `%v` is unsupported value (%T)", arg.Value, arg.Value).debug(1)
                return
            } else {
                name, para = flag, arg.args
            }
        default:
            erro(of(ctx,a), " `%v` is unsupported (%T)", a, a).debug(1)
            return
        }

        if name == nil {
            erro(of(ctx,a), " unknown configure `%v` (%T)", a, a).debug(1)
            return
        }

        if configured, value = configureExecute(ctx, &opts, target, name, para); !configured {
            erro(ctx, "%v: not configured with: %T %v", target, name, name).debug(1)
            return
        } else if v := value; v == nil {
            value = makeNull(a.Position())
        } else if isNull(v) || isNone(v) || isUndef(v) {
            // noop
        } else if v = value.expand(ctx, plain); v != nil && v != value {
            value = v
        }

        if value == d || (value != nil && value.refs(ctx, d)) {
            // Value is the Def, does nothing!
        } else if opts.accumulate {
            d.append(ctx, value)
        } else {
            d.set(ctx, DefConfig, value)
        }

        if d == nil { uni.configuration.done[d] = true }
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
        file := stat(ctx, rel, "", root, info)
        if enable_assertions {
            assert(file != nil, "`%s` file is nil", rel)
        }
        return fn(file, err)
    })
}

var configuredFiles = make(map[string]*Scope,8)

type (
    configureConvertArgs func(args []Value, out *bytes.Buffer) []Value
    configureConvertFunc func(str string, out *bytes.Buffer) error
    configureConvertOpts struct {
        generalOpts
        mode     os.FileMode `m,mode`
        makePath bool `p,path`
        mustConf bool `mc,mustconfig,mustconf,must-conf,must-config,nc,needsconfig,needs-config`
        reconfig bool `r,reconfig`
        update   bool `u,update`
    }
)
func configureConvert(ctx Context, dealArgs configureConvertArgs, dealData configureConvertFunc, opts *configureConvertOpts, args ...Value) (result Value) {
    var (
        project = ctx.Project()
        pc = ctx.pc()
        closured = closureProjects(ctx)
        filename string
        file *File
        target as
    )

    args = parseOpts(ctx, opts, plain, args...)

    if target.Value = autoVal(ctx, "@"); isTrivial(target.Value) {
        erro(ctx, "'@' is not defined").debug(1)
        return
    } else if file, filename, _ = target.fullname(ctx, closured...); file == nil {
        if depend := autoVal(ctx,">"); !isTrivial(depend) {
            s := pc.traves.add(ctx, traveFail, target.Value)
            s.error = traveTargetNotDefinedFile
            s.depend = depend
        } else if true {
            prompt(ctx, "%v: not defined as file\n", target.strval(ctx))
            erro(ctx, "(%T) %v", target.Value, target.Value)
            errostack(ctx, 8, "").debug(64)
        }
        return
    } else if filename == "" {
        errostack(ctx, 3, "%v: empty fullname: `%v`", target.Value, file).debug(1)
        return
    }

    if _, prev := autoSet(ctx, "@", file); opts.debug>0 {
        info(ctx, "configure-file: %s->%s (%T %v -> %T %v)",
            file, filename, prev, prev, file, file).debug(opts.debug)
    }

    if file.info == nil { if f := stat(ctx, filename, "", ""); f != nil { file.info = f.info }}
    if opts.debug>0 && file != nil {
        info(ctx, "configure-file: %v: %v (%s) (%v)",
            autoVal(ctx,"@"), file.fullname(), closured).debug(opts.debug)
    }

    if len(project.configs) == 0 {
        // no need to check configuration
    } else if f := project.configuration(ctx); f == nil || !f.exists() {
        prompt(ctx, "%v: %v\n", filename, file)
        if opts.mustConf {
            var d = opts.debug ; if d == 0 { d = 1 }
            errostack(ctx, opts.stack, "no configuration (%v), try -conf first, in %v",
                f, project).debug(d)
            return
        } else if true {
            warnstack(ctx, opts.stack, "no configuration (%v), try -conf first, in %v",
                f, project).debug(opts.debug)
        }
    }

    // Check previously configured files, we only configure once unless
    // optReconfig is true.
    var closure *Scope
    if configuredFiles != nil {
        var okay bool
        closure, okay = configuredFiles[filename]
        if okay && closure != nil && !opts.reconfig { return }
    }

    //if closure == nil { closure = ctx.closcop }
    defer func(s string, c *Scope) { configuredFiles[s] = c } (filename, closure)

    var data bytes.Buffer
    if h := autoVal(ctx,"-"); !isNull(h) { args = append(args, h) }
    if dealArgs != nil { args = dealArgs(args, &data) }
    if dealData != nil { for _, arg := range args {
        if str := arg.strval(ctx); str == "" {
            continue
        } else if err := dealData(str, &data); err != nil {
            erro(ctx, "convert: %v", err).debug(1)
            return
        }
    }}
    if data.Len() == 0 {
        prompt(ctx, "%v: %v %v\n", filename, autoVal(ctx,"@"), autoVal(ctx,">"))
        errostack(ctx, 5, "empty configuration data").debug(6)
        return
    } else if f := ctx.Project().configuration(ctx); (f == nil || !f.exists()) && opts.debug>0 {
        // NOTE: TrimSpace to ease emacs *compilation* parse errors
        prompt(ctx, "%v: %v\n%s\n", filename, autoVal(ctx,"@"), strings.TrimSpace(data.String())).debug(1)
    }

    var ( status string; same bool )
    if opts.verbose { defer func(st time.Time) {
        if same {
            if true { return } else { status = "unchanged" }
        } else if status == "" {
            status = fmt.Sprintf("outdated (%s)", filename)
        }

        var d = time.Now().Sub(st)
        printEnteringDirectory(ctx)
        prompt(ctx, "update %v …… %s (in %v)\n", trimPromptString(filename), status, d)
        if d := opts.debug; d>0 { infostack(ctx, opts.stack, "%v (%v)", autoVal(ctx, "@"), d).debug(d) }
    }(time.Now())}

    if file.info != nil {
        var err error
        if same, err = crc64CheckFileModeContent(ctx, filename, data.Bytes(), opts.mode); err != nil {
            erro(ctx, " crc64 checksum failed: %v", err).debug(1)
            return
        } else if same {
            var tt = file.info.ModTime()
            for _, d := range merge(ctx.pc().targets...) {
                if f, ok := toFile(d); !ok { continue } else
                if dt := f.info.ModTime(); dt.After(tt) { tt = dt }
            }
            if tt.After(file.info.ModTime()) { err = touch(ctx, file, 0, false, tt) }
            result = file
            return
        }
    } else if dir := filepath.Dir(filename); opts.makePath && dir != "." && dir != PathSep {
        if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
            erro(ctx, " %v", err).debug(1)
            return
        }
    }

    if err := ioutil.WriteFile(filename, data.Bytes(), opts.mode); err != nil {
        erro(ctx, " %v", err).debug(1)
        return
    } else if file.info != nil { result = file } else {
        if file.info, err = os.Stat(filename); err == nil {
            //ctx.Globe().stamp(filename, file.info.ModTime())
            result = file
        }
    }

    if opts.debug>0 {
        status = fmt.Sprintf("configured (%s, %d bytes)", filename, data.Len())
    } else {
        status = fmt.Sprintf("configured (%d bytes)", data.Len())
    }
    return
}

type modifier_configureinput struct { modifier_ }
func (ctx *modifier_configureinput) x(args ...Value) (result interface{}) {
    var opts = configureConvertOpts{ mode: os.FileMode(0600) }
    var dealArgs = func(args []Value, out *bytes.Buffer) []Value {
        var project = ctx.Project()
        if def, ok := project.scope.Lookup("configure.names").(*def); ok {
            args = append(args, xmerge(ctx, plain, def.value)...)
        }

        var configs = make(map[string]*def)
        for _, a := range args {
            var name = a.strval(ctx)
            if _, ok := configs[name]; ok {
                continue
            } else if obj := project.resolveObject(ctx, name); obj == nil {
                erro(ctx, "undefined %v", name).debug(1)
                return nil
            } else if def, ok := obj.(*def); ok {
                configs[name] = def
            }
        }
        for _, c := range project.configs {
            var name = c.name(ctx)
            if def, ok := project.scope.Lookup(name).(*def); ok {
                configs[name] = def
            }
        }
        for _, def := range configs {
            fmt.Fprintf(out, "#undef %s\n", def.name(ctx))
        }
        return args
    }

    return configureConvert(ctx, dealArgs, nil, &opts, args...)
}

// configure-file modifier (see also builtinConfigureFile), example usage:
//
//     config.h: config.h.in [(configure-file)]
//
type modifier_configurefile struct { modifier_ }
func (ctx *modifier_configurefile) x(args ...Value) (result interface{}) {
    var opts = configureConvertOpts{ mode: os.FileMode(0600) }
    var convert = func(str string, out *bytes.Buffer) (err error) {
        return configure(ctx, out, ctx.Project(), str)
    }
    return configureConvert(ctx, nil, convert, &opts, args...)
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
    var pos = ctx.Position()
    var pats []Value
    for _, arg := range args {
        switch a := arg.(type) {
        case *group: pats = append(pats, a.Elems...)
        default:     pats = append(pats, a)
        }
    }
    if len(pats) == 0 {
        erro(ctx, "extract-configuration: missing file names (patterns)").debug(1)
        return
    }
    if len(ctx.rxs) == 0 {
        erro(ctx, "extract-configuration: missing -rx=... flags").debug(1)
        return
    }
    if ctx.target == "" {
        ctx.target = "configuration"
    }

    var outFile string
    if target := autoVal(ctx,"@"); isNull(target) {
        erro(ctx, " target '@' is undefined").debug(1)
        return
    } else {
        outFile = target.strval(ctx)
    }

    if ctx.makePath {
        if err := os.MkdirAll(filepath.Dir(outFile), os.FileMode(0755)); err != nil {
            erro(ctx, " make path failed: %v", err).debug(1)
            return
        }
    }

    var (
        err error
        fil *os.File
        out *bufio.Writer
    )
    if fil, err = os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, ctx.mode); err != nil {
        erro(ctx, " open file failed: %v", err).debug(1)
        return
    } else {
        out = bufio.NewWriter(fil)
    }
    defer func() {
        out.Flush()
        fil.Close()
    }()

    var depends, sources []Value
    if d := autoVal(ctx, "^"); !isTrivial(d) {
        depends = xmerge(ctx, plain, d)
    }

    var patsVal = ease(ctx, pats)
    for _, depend := range depends {
        var a []Value
        switch d := depend.(type) {
        case *File:
            if a = merge(call(ctx, "filter", plain, nil, patsVal, d)); a != nil {
                sources = append(sources, a...)
            }
        case *Path:
            var s = d.strval(ctx)
            err = walkFiles(ctx, s, pats, func(file *File, err error) error {
                if err == nil { sources = append(sources, file) }
                return err
            })
        default:
            var s = d.strval(ctx)
            dir := filepath.Dir(s)
            name := filepath.Base(s)
            file := stat(ctx, name, "", dir)
            if file == nil {
                erro(ctx, " extract-configuration: `%s` file not found", name).debug(1)
                return
            } else if file.info.IsDir() {
                err = walkFiles(ctx, s, pats, func(file *File, err error) error {
                    if err == nil { sources = append(sources, file) }
                    return err
                })
            } else if a = merge(call(ctx, "filter", plain, nil, patsVal, d)); a != nil {
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
        default: s = v.strval(ctx)
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
