//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "extbit.io/smart/scanner"
    "extbit.io/smart/token"
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

type packagetype uint8

const (
    packageUnknown packagetype = iota
    packageSmart  // smart package
    packageConfig // pkgconfig
)

type packageinfo struct {
    *Project
    ty packagetype // smart, pkgconfig, cmake, etc.
}

type libraryinfo struct {
    name string // lib[name].a, lib[name].so, [name].lib, etc.
    dir string
}

var configuration = &struct{
    paths searchlist
    fset *token.FileSet
    libraries map[string]*libraryinfo
    packages map[string]*packageinfo
    done map[*def]bool
    entries []Entry // order list
    clean []string
}{
    fset: token.NewFileSet(),
    libraries: make(map[string]*libraryinfo),
    packages: make(map[string]*packageinfo),
    done: make(map[*def]bool),
}

var configurationOps = map[string] func(Context, map[string]Value, ...Value) (Value) {
    "answer":  configureAnswer,
    "bool":    configureBool,
    "dump":    configureDump,
    "o":       configureOption,
    "opt":     configureOption,
    "option":  configureOption,
    "pkg":     configurePackage,
    "package": configurePackage,
}

type configureExecutor struct {
    file *os.File
    writer *bufio.Writer
    defs map[string]*def
}

func (ce *configureExecutor) execute(ctx Context, project *Project, entry Entry) (result *Project, okay bool) {
    if n := ctx.checkErrors(true); n > 0 {
        return
    } else if ctx = at(ctx, entry.Position()); ctx == nil {
        erro(ctx, "%v: nil positional context", project).debug(1)
        return
    } else if p := entry.OwnerProject(); p != project && p != nil {
        if p.configured { return nil, true } // already configured

        ce.defs = make(map[string]*def) // reset defs for p
        var f, e = p.openConfiguration(ctx)
        if e != nil {
            erro(ctx, "%v", e).debug(1)
            return
        } else if f != nil {
            if ce.writer != nil {
                if e = ce.writer.Flush(); e != nil {
                    erro(ctx, "%v", e).debug(1)
                    return
                }
            }
            if ce.file != nil {
                if e = ce.file.Close(); e != nil {
                    erro(ctx, "%v", e).debug(1)
                    return
                }
            }
        }

        ce.file, ce.writer = f, bufio.NewWriter(f)
        fmt.Fprintf(ce.writer, "# %s (%s) configuration\n", p.spec, p.relPath)

        prompt(ctx, "Project %s …… (%s)\n", p.spec, p.relPath)
        project = p
    }

    result = project

    if val, traves := entry.Execute(ctx); len(traves) > 0 {
        for _, brk := range traves {
            if brk.what == traveFail {
                erro(ctx, "execute '%v' failed: %v", entry, brk).debug(1)
            }
        }
    } else if entry.String() == "-check-file" {
        warn(ctx, "configure %v: %v", entry, val).debug(1)
    }

    var s = entry.Target().Strval(ctx)
    if def := project.scope.FindDef(s); def != nil {
        okay = true // good!
        if d, ok := ce.defs[s]; ok && d != nil {
            /*if d.value.cmp(def.value) != cmpEqual {
                    erro(ctx, "'%s' already configured: %v", d.name, d.value).at(entry.Position())
                    return
            }*/
            return //continue
        } else {
            ce.defs[s] = def
        }
        if def.value == nil {
            // Set <nil> value with exec-assigning ('!=') to a None value.
            fmt.Fprintf(ce.writer, "%v !=\n", def.name)
        } else {
            fmt.Fprintf(ce.writer, "%v = %v\n", def.name,
                elementString(ctx, def, def.value, elemNoBrace))
        }
    } else {
        erro(ctx, "`%s` unconfigured", s).debug(1)
    }
    return
}
func (ce *configureExecutor) close() {
    if ce.writer != nil { if err := ce.writer.Flush(); err != nil {} }
    if ce.file != nil   { if err := ce.file.Close();   err != nil {} }
}

func (ctx *universe) configure() {
    var (
        project *Project
        err error
    )

    // Remove all existing configuration.sm files
    if options.cleanConf { for _, s := range configuration.clean {
        if _, e := os.Stat(s); e != nil {
            if false { prompt(ctx, "%v\n", e).debug(1) }
        } else if e = os.Remove(s); e == nil {
            prompt(ctx, "Remove %s\n", s)
        } else if true {
            prompt(ctx, "Remove: %s\n", e).debug(1)
        }
    }}

    var configureInits = make(map[Entry]int)
    for _, entry := range configuration.entries {
        var project = entry.OwnerProject()
        if defent := project.configure.defaultEntry; defent != nil {
            configureInits[defent] += 1
        }
    }
    for entry, _ := range configureInits {
        var vals, traves = entry.Execute(ctx)
        if traves.has() {
            for _, brk := range traves {
                if brk.what == traveFail {
                    erro(of(ctx,entry), "execute '%v' failed: %v", entry, brk).debug(1)
                }
            }
        }
        if false && len(vals) > 0 {
            var n int
            for _, val := range vals {
                if !isTrivial(val) {
                    info(ctx, "%v: %v", entry, val)
                    n += 1
                }
            }
            if n > 0 { info(ctx, "%v (%d results)", entry, n).debug(1) }
        }
    }

    var ce = &configureExecutor{ defs:make(map[string]*def) }
    defer ce.close()
    for _, entry := range configuration.entries {
        var okay bool
        if project, okay = ce.execute(ctx, project, entry); !okay {
            erro(ctx, "configure '%v' failed", entry).debug(1)
            break
        }
    }
    if err != nil {
        prompt(ctx, "configure failed: %v\n", err).debug(1)
        return
    }
    if n := ctx.checkErrors(true); n > 0 {
        warn(ctx, "configuration got %d errors", ctx.totalErrors()).debug(1)
        if options.failOnErrors { fail(ctx.Position(), "fail by %d errors", ctx.totalErrors()) }
        //return
    }
    printLeavingDirectory(ctx)
    return
}

func (p *Project) openConfiguration(ctx Context) (file *os.File, err error) {
    // defer setclosure(setclosure(cloctx.unshift(p.scope)))
    if f := p.configuration(ctx); f == nil {
        erro(ctx, "nil configuration file for %v", p).debug(1)
        return
    } else if s := f.fullname(); s == "" {
        erro(ctx, "empty configuration file name: %v", f).debug(1)
        return
    } else if err = os.MkdirAll(filepath.Dir(s), os.FileMode(0755)); err != nil {
        erro(ctx, "make path %s failed: %v", filepath.Dir(s), err).debug(1)
        return
    } else if file, err = os.OpenFile(s, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0600)); err != nil {
        erro(ctx, "open configuration %s failed: %v", s, err).debug(1)
        return
    } else {
        return
    }
}

func configPrintf(ctx Context, str string, args... interface{}) {
    prompt(ctx, str, args...) //prompt(ctx,  str, args...)
}

func configMessageDone(ctx Context, str string, args... interface{}) {
    if !strings.HasSuffix(str, "\n") { str += "\n" }
    configPrintf(ctx, str, args...)
}

// -dump
func configureDump(ctx Context, fields map[string]Value, params ...Value) (result Value) {
    return autoGet(ctx,"-")
}

func configureBoolValue(ctx Context) (result bool) {
    var d = autoGet(ctx, "-")
    if d = autoGet(ctx, "-"); d == nil { return }
    for i, v := range merge(d.expand(ctx, plain)) {
        if v == nil { continue } else {
            result = (i == 0 || result) && v.True(ctx)
        }
        if !result { break }
    }
    return
}

// -bool
// -bool('message...')
func configureBool(ctx Context, fields map[string]Value, params ...Value) Value {
    return MakeBoolean(ctx.Position(), configureBoolValue(ctx))
}

// -answer
// -answer('message...')
func configureAnswer(ctx Context, fields map[string]Value, params ...Value) (result Value) {
    return MakeAnswer(ctx.Position(), configureBoolValue(ctx))
}

// -option
// -option('message...')
func configureOption(ctx Context, fields map[string]Value, args ...Value) (result Value) {
    if d := autoGet(ctx,"-"); d != nil {
        result = d.expand(ctx, plain)
    } else {
        result = MakeAnswer(ctx.Position(), false)
    }
    return
}

// -package finds system package in a way similar to cmake.find_package
func configurePackage(ctx Context, fields map[string]Value, args ...Value) (result Value) {
    var names []string
    var optType packagetype = packageSmart
    for _, arg := range args {
        switch a := arg.(type) {
        case *Pair:
            var (
                key = a.Key.Strval(ctx)
                val = a.Value.Strval(ctx)
            )
            switch key {
            case "type":
                switch val {
                case "", "smart": optType = packageSmart
                case "pkgconfig": optType = packageConfig
                default:      optType = packageUnknown
                    erro(ctx, "package: unknown type %v", val)
                    return
                }
            default:
                prompt(ctx, "%v: package: `%v` unknown option", key)
            }
        default:
            names = append(names, a.Strval(ctx))
        }
    }
    for _, name := range names {
        var info, cached = configuration.packages[name]
        if !cached {
            switch optType {
            // case packageSmart:
            //     if info, err = loadPackageSmartInfo(pos, name); err != nil { return }
            // case packageConfig:
            //     if info, err = loadPackageConfigInfo(pos, name); err != nil { return }
            case packageUnknown:
                prompt(ctx, "%v: package `%v`: unknown type\n", name)
            }
            if info != nil {
                configuration.packages[name] = info
                result = MakeAnswer(ctx.Position(), true)
                break
            }
        }
    }
    return
}

func scanExitStatus(err error) (n, status int) {
    switch e := err.(type) {
    case *exitstatus: n, status = 1, e.code
    case *scanner.Error:
        for _, t := range e.Errs {
            if n, status = scanExitStatus(t); n == 1 { return }
        }
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
func (cc *configureContext) isConfiguration() bool { return true }

type commonConfigureOpts struct {
    silent bool `s,silent`
    noResetHyphen bool `r,reset` // reset hyphen value, aka. "-"
}
func executeConfigureEntry(ctx Context, opts *modifierConfigureOpts, entryName string, target Value, paramsOrig ...Value) (configured bool, result Value) {
    if options.traceConfig { defer un(trace(t_config, fmt.Sprintf("executeConfigureEntry(%s %v)", entryName, ctx))) }

    var entries *ResolveEntries
    if program := ctx.program(); program == nil {
        erro(ctx, "needs program context to configure: %v", ctx).debug(1)
        return
    } else if program.project.configure == nil {
        erro(ctx, "%v: .configure not provided for %v (%s)", program.project, target, entryName).debug(1)
        return
    } else if entries = program.project.configure.resolveEntries(ctx, "-"+entryName, false, false); entries == nil {
        erro(ctx, "unknown configuration action `%v`, no such entry", entryName).debug(1)
        return
    }

    var (
        commOpts commonConfigureOpts
        params []Value
        pos = ctx.Position()
        hyphen = autoGet(ctx,"-")
        verbose = opts.verbose
    )

    paramsOrig = parseOpts(ctx, &commOpts, 0, paramsOrig...)

    // Reset the result/output def '-'?
    // NOTE: have to reset hyphen to ensure configured value is saved
    if !commOpts.noResetHyphen { ctx.autoSet("-", nil) }

    // verbose mode is on if silent flag was not set
    if !verbose && !commOpts.silent { verbose = !commOpts.silent }

    var (
        programs = entries.Programs()
        prog = programs[0]
    )
    for _, par := range prog.params {
        switch par.name {
        case "LANG":   params = append(params, MakePair(pos, MakeBareword(pos, "LANG"),   MakeString(pos, ctx.program().language)))
        case "TARGET": params = append(params, MakePair(pos, MakeBareword(pos, "TARGET"), target))
        case "VALUE":  params = append(params, MakePair(pos, MakeBareword(pos, "VALUE"),  hyphen))
            if hyphen == nil { warn(ctx, "nil hyphen def").debug(1) }
        }
    }
ForInParams:
    for _, a := range paramsOrig {
        var (
            pair *Pair
            ok bool
        )
        if pair, ok = a.(*Pair); !ok {
            erro(of(ctx,a), " unsupported parameter %v (%T)", a, a).debug(1)
            return
        }

        var (
            key = pair.Key.Strval(ctx)
            value = pair.Value
        )
        if _, ok := value.(*Compound); ok {
            value = MakeString(pos, value.Strval(ctx))
        } else if value != nil {
            value = value.expand(ctx, plain)
        }

        for _, par := range prog.params {
            if par.name == key || par.name == strings.ToUpper(key) {
                params = append(params, MakePair(pos, MakeBareword(pos, par.name), value))
                continue ForInParams
            }
        }

        if key == "INFO" {
            if false && verbose { prompt(ctx, "%s", pair.Value) }
        } else if true {
            var params []string
            for _, p := range prog.params { params = append(params, p.name) }

            var t = autoGet(ctx,"@")
            ctx = at(ctx, a.Position())
            warn(ctx, "ignored param: %T %v; target: %T %v", a, a, t, t)
            warn(at(ctx,prog.position), "%v params = %v", t, params).debug(16)
            return
        }
    }
    if false { if s := target.Strval(ctx); s == "HAVE_FUN_SENDFILE" {
        warn(ctx, "%v: %v", s, paramsOrig)
        warn(ctx, "%v: %v", s, params).debug(1)
    }}

    ctx = &configureContext{ ctx }

    var (
        reses []Value
        traves travestates
    )
    for _, entry := range entries.all {
        if reses, traves = entry.execute(ctx, params...); ctx.checkErrors(true) > 0 {
            warn(at(ctx,entry.Position()), "%v", entry)
            warnstack(ctx, 5, `configure '%s' got %d error(s)`,
                entryName, ctx.totalErrors()).debug(1)
            if options.failOnErrors { fail(pos, "fail by %d errors", ctx.totalErrors()) }
        } else if n := len(reses); n != 1 {
            if true { // just bypass, no configuration results - <nil>
                if false { warn(at(ctx,entry.Position()), "%v", entry).debug(1) }
            } else if erro(at(ctx,entry.Position()), "%v", entry); n == 0 {
                errostack(ctx, 5, `configure "%s" has no results`, entryName).debug(32)
            } else {
                errostack(ctx, 5, `configure "%s" has multiple results (%d)`, entryName, n).debug(32)
            }
        } else if result = reses[0]; !isNil(result) && result == hyphen {
            warn(at(ctx,entry.Position()), "%v", entry)
            warn(ctx, `%v: configure yields value the same as input will be ignored: %v`, entry, result).debug(1)
            result = nil // simply discard the result as it's the same as the input (hyphen) value
        }
        if traves = traves.not(traveDone,traveRule,traveFile); traves.has() {
            for i, s := range traves { erro(ctx, "%v: %d. %v", entry, i, s) }
            erro(ctx, "%v: %d trave states", entry, len(traves)).debug(16)
        }
    }
    configured = true
    return
}

func configureDo(ctx Context, opts *modifierConfigureOpts, target Value, name Value, args []Value) (configured bool, result Value) {
    if options.traceConfig { defer un(trace(t_config, "configureDo")) }

    var (
        pos = ctx.Position()
        strName = name.Strval(ctx)
        params []Value
        infos []Value
    )
    if strName == "" {
        erro(ctx, " empty configure name: %v (%T)", name, name).debug(1)
        return
    }

    for _, arg := range mergex(ctx, plain, args...) {
        if isTrivial(arg) { continue }
        switch t := arg.(type) {
        case *Pair: params = append(params, t)
        case *Raw, *String, *Compound:
            params = append(params, MakePair(pos, MakeBareword(pos, "INFO"), t))
            infos = append(infos, t)
        default:
            erro(of(ctx,arg), " unsupported parameter: $T %v", t, t).debug(1)
            return
        }
    }
    if false { if s := target.Strval(ctx); s == "HAVE_FUN_SENDFILE" {
        warn(ctx, "%v: %v", s, args)
        warn(ctx, "%v: %v", s, mergex(ctx, plain, args...))
        warn(ctx, "%v: %v", s, params).debug(1)
    }}

    defer func() {
        if isNil(result) {
            configMessageDone(ctx, "… <nil>")
        } else if isNone(result) {
            configMessageDone(ctx, "… <none>")
        } else {
            var s = result.Strval(ctx)
            if s == "" { s = fmt.Sprintf("? (%s)", result) }
            configMessageDone(ctx, "… %v", s)
        }
    } ()

    if len(infos) == 0 {
        configPrintf(ctx, "%v %v …", target, args)
    } else {
        var msg string
        for _, info := range infos {
            msg += info.Strval(ctx)
        }
        if msg != "" { configPrintf(ctx, "%s …", msg) }
    }

    // Process configurations like:
    //   -bool
    //   -option
    //   -package
    //   ...
    if config, ok := configurationOps[strName]; ok {
        params = append(params, MakePair(pos, MakeBareword(pos, "TARGET"), target))
        result = config(ctx, nil, params...)
        if options.traceConfig {
            t_config.tracef("configured: %v, result = %v (%s)", configured, result, typeof(result))
        }
        configured = true
    } else {
        configured, result = executeConfigureEntry(ctx, opts, strName, target, params...)
    }
    if configured && options.traceConfig {
        t_config.tracef("configured: %v, result = %v (%s)", configured, result, typeof(result))
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
type modifierConfigureOpts struct {
    generalOpts
    accumulate bool `a,accumulate;a,add`
}
func (ctx modifier) Configure(args ...Value) (result Value, _ travestates) {
    if options.traceConfig { defer un(trace(t_config, fmt.Sprintf("modifierConfigure(%v) (reconfig=%v)", ctx, options.reconfigure))) }

    var program = ctx.program()
    if program == nil {
        erro(ctx, " needs traversal context to configure: %v", ctx).debug(1)
        return
    }

    var pos = ctx.Position()
    var opts modifierConfigureOpts
    args = parseOpts(ctx, &opts, plain, args...)

    if program.project.configure == nil {
        if program.project.name == "configure" {
            if o := program.project.scope.Lookup(dotConfigure); !isNil(o) {
                if d, ok := o.(*def); ok && !isNil(d.value) && !isNone(d.value) {
                    if val := d.value.True(ctx); val {
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

    var target = autoGet(ctx,"@")
    if isNil(target) {
        erro(ctx, " target is trivial: %s", ctx).debug(1)
        return
    }

    var name = target.Strval(ctx)
    if len(program.project.bases) == 0 {
        warn(of(ctx,target), "%v: project has no bases (should have at least .configure)", name).debug(1)
    }

    var d *def
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

    if options.traceConfig {
        t_config.tracef("%s: %v (%T)", d.name, d.value, d.value)
        defer func() { t_config.tracef("%s: %v (%T)", d.name, d.value, d.value) } ()
    }
    if !isNil(d.value) { // Check if it's already configured?
        if !options.reconfigure { return } // return if not reconfigure
        if done, found := configuration.done[d]; done && found { return }
    }

    var value Value
    if len(args) == 0 { // Empty configuration: (configure)
        if value = autoGet(ctx,"-"); value == nil || value == d || value.refs(ctx, d) {
            return
        }

        switch v := value.(type) {
        default: d.set(ctx, DefConfig, value)
        case *ExecResult:
            var s string
            if /*v.wg.Wait()*/; v.Status == 0 && v.Stdout.Buf != nil {
                s = v.Stdout.Buf.String()
            } else if v.Stderr.Buf != nil {
                s = v.Stderr.Buf.String()
            }
            d.set(ctx, DefConfig, MakeString(pos, s))
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
        case *Argumented:
            if flag, okay := arg.value.(*Flag); !okay {
                erro(of(ctx,a), " `%v` is unsupported value (%T)", arg.value, arg.value).debug(1)
                return
            } else {
                name, para = flag.name, arg.args
            }
        case *Flag:
            if isNil(arg.name) || isNone(arg.name) {
                erro(of(ctx,a), " `%v` is unsupported flag (%T)", arg.name, arg.name).debug(1)
                return
            } else {
                name = arg.name
            }
        default:
            erro(of(ctx,a), " `%v` is unsupported (%T)", a, a).debug(1)
            return
        }
        if name == nil {
            erro(of(ctx,a), " unknown configure `%v` (%T)", a, a).debug(1)
            return
        }

        if configured, value = configureDo(ctx, &opts, target, name, para); !configured {
            erro(ctx, " %s not configured for %v", name, target).debug(1)
            return
        } else if v := value; v == nil {
            value = MakeNil(a.Position())
        } else if isNil(v) || isNone(v) || isUndef(v) {
            // noop
        } else if v = value.expand(ctx, plain); v != nil && v != value {
            value = v
        }

        if value == d || (!isNil(value) && value.refs(ctx, d)) {
            // Value is the Def, does nothing!
        } else if opts.accumulate {
            d.append(ctx, value)
        } else {
            d.set(ctx, DefConfig, value)
        }

        if d == nil { configuration.done[d] = true }
        if options.traceConfig {
            t_config.tracef("configured: %v (%s) (%v)", value, typeof(value), d.origin)
        }
    }
    if !configured { erro(ctx, " `%v` not configured", target).debug(1) }
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
func configureConvert(ctx Context, dealArgs configureConvertArgs, dealData configureConvertFunc, opts *configureConvertOpts, args ...Value) (result Value, traves travestates) {
    var (
        project = ctx.Project()
        closured = closureProjects(ctx)
        filename string
        file *File
        target as
    )

    args = parseOpts(ctx, opts, plain, args...)

    if target.Value = autoGet(ctx, "@"); isTrivial(target.Value) {
        erro(ctx, "'@' is not defined").debug(1)
        return
    } else if file, filename, _ = target.fullname(ctx, closured...); file == nil {
        if depend := autoGet(ctx,">"); !isTrivial(depend) {
            s := traves.add(ctx, traveFail, target.Value)
            s.error = traveTargetNotDefinedFile
            s.depend = depend
        } else if true {
            prompt(ctx, "%v: not defined as file\n", target.Strval(ctx))
            erro(ctx, "(%T) %v", target.Value, target.Value)
            errostack(ctx, 8, "").debug(64)
        }
        return
    } else if filename == "" {
        errostack(ctx, 3, "%v: empty fullname: `%v`", target.Value, file).debug(1)
        return
    }

    if _, prev := ctx.autoSet("@", file); opts.debug>0 {
        info(ctx, "configure-file: %s->%s (%T %v -> %T %v)",
            file, filename, prev, prev, file, file).debug(opts.debug)
    }

    if file.info == nil { if f := stat(ctx, filename, "", ""); f != nil { file.info = f.info }}
    if opts.debug>0 && file != nil {
        info(ctx, "configure-file: %v: %v (%s) (%v)",
            autoGet(ctx,"@"), file.fullname(), closured).debug(opts.debug)
    }

    if len(project.configs) == 0 {
        // no need to check configuration
    } else if f := project.configuration(ctx); f == nil || !f.exists() {
        prompt(ctx, "%v: %v\n", filename, file)
        if opts.mustConf {
            var d = opts.debug ; if d == 0 { d = 1 }
            errostack(ctx, opts.stackNum, "no configuration (%v), try -conf first, in %v",
                f, project).debug(d)
            return
        } else if true {
            warnstack(ctx, opts.stackNum, "no configuration (%v), try -conf first, in %v",
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
    if h := autoGet(ctx,"-"); !isNil(h) {
        args = append(args, h)
    }
    if dealArgs != nil { args = dealArgs(args, &data) }
    if dealData != nil { for _, arg := range args {
        if str := arg.Strval(ctx); str == "" {
            continue
        } else if err := dealData(str, &data); err != nil {
            erro(ctx, "convert: %v", err).debug(1)
            return
        }
    }}
    if data.Len() == 0 {
        prompt(ctx, "%v: %v\n", filename, autoGet(ctx,"@"))
        erro(ctx, "no configuration data").debug(6)
        return
    } else if f := ctx.Project().configuration(ctx); (f == nil || !f.exists()) && opts.debug>0 {
        // NOTE: TrimSpace to ease emacs *compilation* parse errors
        prompt(ctx, "%v: %v\n%s\n",
            filename, autoGet(ctx,"@"), strings.TrimSpace(data.String())).debug(1)
    }

    var ( status string; same bool )
    if opts.verbose {
        defer func(st time.Time) {
            if same {
                if true { return } else { status = "unchanged" }
            } else if status == "" {
                status = fmt.Sprintf("outdated (%s)", filename)
            }

            var d = time.Now().Sub(st)
            printEnteringDirectory(ctx)
            prompt(ctx, "update %v …… %s (in %v)\n",
                trimPromptString(filename), status, d)
            if opts.debug>0 { infostack(ctx, opts.stackNum, "%v (%v)",
                autoGet(ctx, "@"), d).debug(opts.debug) }
        } (time.Now())
    }

    if file.info != nil {
        var err error
        if same, err = crc64CheckFileModeContent(ctx, filename, data.Bytes(), opts.mode); err != nil {
            erro(ctx, " crc64 checksum failed: %v", err).debug(1)
            return
        } else if same {
            var tt = file.info.ModTime()
            for _, d := range merge(ctx.programContext().targets...) {
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

// configure-input generate configure input file from a .ac file, example usage:
// 
//     config.h.in: configure.ac [(configure-input)]
//     
func _modifierConfigureInput(ctx Context, args ...Value) (result Value, _ travestates) {
    var opts = configureConvertOpts{ mode: os.FileMode(0600) }
    var convert = func(str string, out *bytes.Buffer) (err error) {
        return autoconf(ctx, out, ctx.Project(), str)
    }
    return configureConvert(ctx, nil, convert, &opts, args...)
}

type modifierConfigureInputOpts struct {
    mode os.FileMode "m,mode"
    makePath bool "p,path"
    verbose bool "v,verb,verbose"
    update bool "u,up,update"
    debug bool "d,db,debug"
}
func __modifierConfigureInput(ctx Context, args ...Value) (result Value, _ travestates) {
    var (
        opts = modifierConfigureInputOpts{ mode:os.FileMode(0640) }
        project = ctx.Project()
    )
    args = parseOpts(ctx, &opts, plain, args...)
    if target := autoGet(ctx,"@"); isTrivial(target) {
        erro(ctx, " target '@' is not defined").debug(1)
        return
    }

    if def, ok := project.scope.Lookup("configure.names").(*def); ok {
        args = append(args, mergex(ctx, plain, def.value)...)
    }

    var configs = make(map[string]*def)
    for _, a := range args {
        var name = a.Strval(ctx)
        if _, ok := configs[name]; ok {
            continue
        } else if obj := project.resolveObject(ctx, name); obj == nil {
            erro(ctx, "undefined %v", name).debug(1)
            return
        } else if def, ok := obj.(*def); ok {
            configs[name] = def
        }
    }
    for _, c := range project.configs {
        var name = c.Name(ctx)
        if def, ok := project.scope.Lookup(name).(*def); ok {
            configs[name] = def
        }
    }

    var data bytes.Buffer
    for _, def := range configs {
        fmt.Fprintf(&data, "#undef %s\n", def.name)
    }
    warn(ctx, "%v", data)
    warn(ctx, "%v: %v", project, args).debug(1)
    return
}

func (ctx modifier) ConfigureInput(args ...Value) (result Value, _ travestates) {
    var opts = configureConvertOpts{ mode: os.FileMode(0600) }
    var dealArgs = func(args []Value, out *bytes.Buffer) []Value {
        var project = ctx.Project()
        if def, ok := project.scope.Lookup("configure.names").(*def); ok {
            args = append(args, mergex(ctx, plain, def.value)...)
        }

        var configs = make(map[string]*def)
        for _, a := range args {
            var name = a.Strval(ctx)
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
            var name = c.Name(ctx)
            if def, ok := project.scope.Lookup(name).(*def); ok {
                configs[name] = def
            }
        }
        for _, def := range configs {
            fmt.Fprintf(out, "#undef %s\n", def.name)
        }
        return args
    }
    return configureConvert(ctx, dealArgs, nil, &opts, args...)
}

// configure-file modifier (see also builtinConfigureFile), example usage:
//
//     config.h: config.h.in [(configure-file)]
//
func (ctx modifier) ConfigureFile(args ...Value) (result Value, _ travestates) {
    var opts = configureConvertOpts{ mode: os.FileMode(0600) }
    var convert = func(str string, out *bytes.Buffer) (err error) {
        return configure(ctx, out, ctx.Project(), str)
    }
    return configureConvert(ctx, nil, convert, &opts, args...)
}

type modifierExtractConfigurationOpts struct {
    mode os.FileMode "m,mode"
    makePath bool "p,path"
    target string "t,target"
    rxs []*regexp.Regexp "r,regex;r,rx" // regexp.Compile(s)
}
// extract-configuration extracts configuration from C/C++ files, example usage:
//
//      config.h.in:[(extract-configuration)]: $(wildcard *.cpp)
//
func (ctx modifier) ExtractConfiguration(args ...Value) (result Value, _ travestates) {
    var (
        pos = ctx.Position()
        opts = modifierExtractConfigurationOpts{ mode:os.FileMode(0640) } // sys default 0666
        pats []Value
    )
    for _, arg := range parseOpts(ctx, &opts, plain, args...) {
        switch a := arg.(type) {
        case *Group: pats = append(pats, a.Elems...)
        default:     pats = append(pats, a)
        }
    }
    if len(pats) == 0 {
        erro(ctx, "extract-configuration: missing file names (patterns)").debug(1)
        return
    }
    if len(opts.rxs) == 0 {
        erro(ctx, "extract-configuration: missing -rx=... flags").debug(1)
        return
    }
    if opts.target == "" {
        opts.target = "configuration"
    }

    var outFile string
    if target := autoGet(ctx,"@"); isNil(target) {
        erro(ctx, " target '@' is undefined").debug(1)
        return
    } else {
        outFile = target.Strval(ctx)
    }

    if opts.makePath {
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
    if fil, err = os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, opts.mode); err != nil {
        erro(ctx, " open file failed: %v", err).debug(1)
        return
    } else {
        out = bufio.NewWriter(fil)
    }
    defer func() {
        out.Flush()
        fil.Close()
    }()

    var (
        filterOpts builtinFilterOpts
        depends, sources []Value
    )
    if d := autoGet(ctx, "^"); !isTrivial(d) {
        depends = mergex(ctx, plain, d)
    }
    for _, depend := range depends {
        var a []Value
        switch d := depend.(type) {
        case *File:
            if a, err = filterValues(ctx, pats, filterOpts, false, d); err != nil {
                erro(ctx, " filter values failed: %v", err).debug(1)
            } else { sources = append(sources, a...) }
        case *Path:
            var s = d.Strval(ctx)
            err = walkFiles(ctx, s, pats, func(file *File, err error) error {
                if err == nil { sources = append(sources, file) }
                return err
            })
        default:
            var s = d.Strval(ctx)
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
            } else if a, err = filterValues(ctx, pats, filterOpts, false, d); err != nil {
                erro(ctx, " filter values failed: %v", err).debug(1)
                return
            } else {
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
        default: s = v.Strval(ctx)
        }
        if f, err = os.Open(s); err != nil {
            prompt(ctx, "%v: (configure) %v: %v\n", pos, source, err)
            continue ForSources
        }
        scanner := bufio.NewScanner(f)
        scanner.Split(bufio.ScanLines)
        for scanner.Scan() {
            s := scanner.Text()
            ForOpts: for _, x := range opts.rxs {
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
    fmt.Fprintf(out, "%s:[(configure -check)]:\\\n", opts.target)
    for _, k := range keys { fmt.Fprintf(out, "  %s \\\n", k) }
    fmt.Fprintf(out, "\n")
    return
}
