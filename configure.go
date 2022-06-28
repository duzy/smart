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
    done map[*Def]bool
    entries []Entry // order list
    clean []string
}{
    fset: token.NewFileSet(),
    libraries: make(map[string]*libraryinfo),
    packages: make(map[string]*packageinfo),
    done: make(map[*Def]bool),
}

var configurationOps = map[string] func(Context, map[string]Value, ...Value) (Value) {
    "answer":  configureAnswer,
    "bool":    configureBool,
    "dump":    configureDump,
    "option":  configureOption,
    "package": configurePackage,
}

type configureExecutor struct {
    file *os.File
    writer *bufio.Writer
    defs map[string]*Def
}

func (ce *configureExecutor) execute(ctx Context, project *Project, entry Entry) (result *Project, okay bool) {
    if n := ctx.checkErrors(true); n > 0 {
        return
    } else if ctx = positional(ctx, entry.Position()); ctx == nil {
        erro(ctx, "%v: nil positional context", project).debug(1)
        return
    } else if p := entry.OwnerProject(); p != project && p != nil {
        if p.configured { return nil, true } // already configured

        ce.defs = make(map[string]*Def) // reset defs for p
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
        } else { ce.defs[s] = def }
        if def.value == nil {
            // Set <nil> value with exec-assigning ('!=')
            // to a None value.
            fmt.Fprintf(ce.writer, "%v !=\n", def.name)
        } else {
            vs := elementString(ctx, def, def.value, elemNoBrace)
            fmt.Fprintf(ce.writer, "%v = %v\n", def.name, vs)
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

func (ctx *defaultContext) configure() {
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
        if defent := project.configure.DefaultEntry(); defent != nil {
            configureInits[defent] += 1
        }
    }
    for entry, _ := range configureInits {
        var vals, traves = entry.Execute(ctx)
        if traves.has() {
            for _, brk := range traves {
                if brk.what == traveFail {
                    erro(ctx, "execute '%v' failed: %v", entry, brk).of(entry).debug(1)
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

    var ce = &configureExecutor{ defs:make(map[string]*Def) }
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

func (p *Project) openConfiguration(ctx Context, ) (file *os.File, err error) {
    // defer setclosure(setclosure(cloctx.unshift(p.scope)))
    if f := p.configuration(ctx); f == nil {
        erro(ctx, "nil configuration file for %v", p).at(p.position).debug(1)
    } else if s := f.fullname(); s == "" {
        erro(ctx, "empty configuration file name: %v", f).at(p.position).debug(1)
    } else if err = os.MkdirAll(filepath.Dir(s), os.FileMode(0755)); err != nil {
        erro(ctx, "make path %s failed: %v", filepath.Dir(s), err).at(p.position).debug(1)
    } else if file, err = os.OpenFile(s, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0600)); err != nil {
        erro(ctx, "open configuration %s failed: %v", s, err).at(p.position).debug(1)
    }
    return
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
    result, _ = ctx.autoGet("-")
    return
}

func configureBoolValue(ctx Context) (result bool) {
    var (
        value, _ = ctx.autoGet("-")
        res Value
    )
    if isNil(value) {
        return
    } else if res = value.expand(ctx, expandPlainValue); !isNil(res) && res != value {
        value = res
    }
    for i, v := range merge(value) {
        if v == nil {
            continue
        } else {
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
    if result, _ = ctx.autoGet("-"); !isNil(result) {
        var res Value
        if res = result.expand(ctx, expandPlainValue); !isNil(res) && res != result {
            result = res
        }
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
func (cc *configureContext) configuration() bool { return true }

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
        commonOpts commonConfigureOpts
        params []Value
        pos = ctx.Position()
        hyphenVal, /*hyphenFound*/_ = ctx.autoGet("-")
        verbose = opts.verbose
    )

    paramsOrig = parseOpts(ctx, &commonOpts, paramsOrig...)

    // Reset the result/output def '-'?
    // NOTE: have to reset hyphen to ensure configured value is saved
    if !commonOpts.noResetHyphen { ctx.autoSet("-", nil) }

    // verbose mode is on if silent flag was not set
    if !verbose && !commonOpts.silent { verbose = !commonOpts.silent }

    var (
        programs = entries.Programs()
        prog = programs[0]
    )
    for _, par := range prog.params {
        switch par.name {
        case "LANG":   params = append(params, MakePair(pos, MakeBareword(pos, "LANG"),   MakeString(pos, ctx.program().language)))
        case "TARGET": params = append(params, MakePair(pos, MakeBareword(pos, "TARGET"), target))
        case "VALUE":  params = append(params, MakePair(pos, MakeBareword(pos, "VALUE"),  hyphenVal))
        }
    }
ForInParams:
    for _, a := range paramsOrig {
        var (
            pair *Pair
            ok bool
        )
        if pair, ok = a.(*Pair); !ok {
            erro(ctx, " unsupported parameter %v (%T)", a, a).of(a).debug(1)
            return
        }

        var key = pair.Key.Strval(ctx)
        for _, par := range prog.params {
            if par.name == key {
                params = append(params, a)
                continue ForInParams
            } else if par.name == strings.ToUpper(key) {
                params = append(params, MakePair(pos, MakeBareword(pos, par.name), pair.Value))
                continue ForInParams
            }
        }
        if key == "INFO" {
            if false && verbose { prompt(ctx, "%s", pair.Value) }
        } else if true {
            var params []string
            for _, p := range prog.params { params = append(params, p.name) }

            var at, _ = ctx.autoGet("@")
            ctx = positional(ctx, a.Position())
            warn(ctx, "ignored param: %T %v; target: %T %v", a, a, at, at)
            warn(ctx, "%v params = %v", at, params).at(prog.position).debug(16)
            return
        }
    }

    ctx = &configureContext{ ctx }

    var (
        reses []Value
        traves travestates
    )
    for _, entry := range entries.all {
        if reses, traves = entry.execute(ctx, params...); ctx.checkErrors(true) > 0 {
            warn(ctx, "%v", entry).at(entry.Position())
            warnstack(ctx, 5, `configure '%s' got %d error(s)`,
                entryName, ctx.totalErrors()).debug(1)
            if options.failOnErrors { fail(pos, "fail by %d errors", ctx.totalErrors()) }
        } else if n := len(reses); n != 1 {
            if true { // just bypass, no configuration results - <nil>
                if false { warn(ctx, "%v", entry).at(entry.Position()).debug(1) }
            } else if erro(ctx, "%v", entry).at(entry.Position()); n == 0 {
                errostack(ctx, 5, `configure "%s" has no results`, entryName).debug(32)
            } else {
                errostack(ctx, 5, `configure "%s" has multiple results (%d)`, entryName, n).debug(32)
            }
        } else if result = reses[0]; !isNil(result) && result == hyphenVal {
            warn(ctx, "%v", entry).at(entry.Position())
            warn(ctx, `%v: configure yields value the same as input will be ignored: %v`, entry, result).debug(1)
            result = nil // simply discard the result as it's the same as the input (hyphen) value
        }
        if traves = traves.not(traveDone); traves.has() {
            for i, brk := range traves {
                erro(ctx, " %d: %v", i, brk).debug(16)
            }
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

ForArgs:
    for _, arg := range args {
        for _, elem := range mergeExpand(ctx, expandPlainValue, arg) {
            switch tv := elem.(type) {
            case *None, *Nil: continue
            case *Pair:
                params = append(params, tv)
                continue ForArgs
            case *Raw, *String, *Compound:
                params = append(params, MakePair(pos, MakeBareword(pos, "INFO"), tv))
                infos = append(infos, tv)
                continue ForArgs
            default:
                erro(ctx, " parameter '%v' of %T is unsupported", tv, tv).of(arg).debug(1)
                return
            }
        }
    }

    defer func() {
        // if err != nil {
        //     if e, ok := err.(*scanner.Error); ok {
        //         configMessageDone(ctx, "… (%v)", e.Brief())
        //     } else {
        //         configMessageDone(ctx, "… (%v)", err)
        //     }
        // } else
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
    accumulate bool `a,accumulate;a,add`
    verbose bool `v,verbose`
}
func modifierConfigure(ctx Context, args ...Value) (result Value, _ travestates) {
    if options.traceConfig { defer un(trace(t_config, fmt.Sprintf("modifierConfigure(%v) (reconfig=%v)", ctx, options.reconfigure))) }

    var program = ctx.program()
    if program == nil {
        erro(ctx, " needs traversal context to configure: %v", ctx).debug(1)
        return
    }

    var pos = ctx.Position()
    var opts modifierConfigureOpts
    args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

    if program.project.configure == nil {
        if program.project.name == "configure" {
            if o := program.project.scope.Lookup(dotConfigure); !isNil(o) {
                if d, ok := o.(*Def); ok && !isNil(d.value) && !isNone(d.value) {
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

    var target, found = ctx.autoGet("@")
    if !found || isTrivial(target) {
        erro(ctx, " target is trivial: %s", ctx).debug(1)
        return
    }

    var name = target.Strval(ctx)
    if len(program.project.bases) == 0 {
        warn(ctx, "%v: project has no bases (should have at least .configure)", name).of(target).debug(1)
    }

    var def *Def
    if def = program.scope.FindDef(name); def == nil {
        var alt Object
        def, alt = program.project.scope.define(ctx, DefConfig, name, nil)
        if def == nil && alt != nil { def, _ = alt.(*Def) }
    }
    if def == nil {
        erro(ctx, " cannot define configuration `%s`", name).debug(1)
        return
    } else {
        result = def
    }

    if options.traceConfig {
        t_config.tracef("%s: %v (%T)", def.name, def.value, def.value)
        defer func() { t_config.tracef("%s: %v (%T)", def.name, def.value, def.value) } ()
    }
    if !isNil(def.value) { // Check if it's already configured?
        if !options.reconfigure { return } // return if not reconfigure
        if done, found := configuration.done[def]; done && found { return }
    }

    var value Value
    if len(args) == 0 { // Empty configuration: (configure)
        if value, _ = ctx.autoGet("-"); value == nil {
            erro(ctx, " `%v` not configured (%v)", target, value).debug(1)
            return
        } else if value == def || value.refs(ctx, def) {
            return
        }
        switch v := value.(type) {
        default: def.set(ctx, DefConfig, value)
        case *ExecResult:
            var s string
            if /*v.wg.Wait()*/; v.Status == 0 && v.Stdout.Buf != nil {
                s = v.Stdout.Buf.String()
            } else if v.Stderr.Buf != nil {
                s = v.Stderr.Buf.String()
            }
            def.set(ctx, DefConfig, MakeString(pos, s))
        }
        return
    } else {
        def.set(ctx, DefConfig, nil)
    }

    var configured bool
ForConfig:
    for i, a := range args {
        if def.value == nil && i > 0 { break ForConfig }

        var ( name Value ; para []Value )
        switch arg := a.(type) {
        case *Argumented:
            if flag, okay := arg.value.(*Flag); !okay {
                erro(ctx, " `%v` is unsupported value (%T)", arg.value, arg.value).of(a).debug(1)
                return
            } else {
                name, para = flag.name, arg.args
            }
        case *Flag:
            if isNil(arg.name) || isNone(arg.name) {
                erro(ctx, " `%v` is unsupported flag (%T)", arg.name, arg.name).of(a).debug(1)
                return
            } else {
                name = arg.name
            }
        default:
            erro(ctx, " `%v` is unsupported (%T)", a, a).of(a).debug(1)
            return
        }
        if name == nil {
            erro(ctx, " unknown configure `%v` (%T)", a, a).of(a).debug(1)
            return
        }

        if configured, value = configureDo(ctx, &opts, target, name, para); !configured {
            erro(ctx, " %s not configured for %v", name, target).debug(1)
            return
        } else if v := value; v == nil {
            value = MakeNil(a.Position())
        } else if isNil(v) || isNone(v) || isUndef(v) {
            // noop
        } else if v = value.expand(ctx, expandPlainValue); !isNil(v) && v != value {
            value = v
        }

        if value == def || (!isNil(value) && value.refs(ctx, def)) {
            // Value is the Def, does nothing!
        } else if opts.accumulate {
            def.append(ctx, value)
        } else {
            def.set(ctx, DefConfig, value)
        }

        if def == nil { configuration.done[def] = true }
        if options.traceConfig {
            t_config.tracef("configured: %v (%s) (%v)", value, typeof(value), def.origin)
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

type configureConvertOpts struct {
    mode os.FileMode `m,mode`
    makePath bool `p,path`
    reconfig bool `r,reconfig`
    verbose bool `v,verbose`
    update bool `u,update`
    debug bool `d,debug`
}

type configureConvertArgs func(args []Value, out *bytes.Buffer) []Value
type configureConvertFunc func(str string, out *bytes.Buffer) error
func configureConvert(ctx Context, dealArgs configureConvertArgs, dealData configureConvertFunc, opts *configureConvertOpts, args ...Value) (result Value, _ travestates) {
    var (
        closured []*Project
        project *Project
        filename string
        file *File
    )
    args = parseOpts(ctx, opts, mergeExpand(ctx, expandPlainValue, args...)...)
    if target, found := ctx.autoGet("@"); !found || isTrivial(target) {
        erro(ctx, " target '@' is not defined").debug(1)
        return
    } else if file, _ = target.(*File); file == nil {
        var s string = target.Strval(ctx)

        if closured == nil { closured = closureProjects(ctx) }
        for _, p := range closured {
            if file = p.FindFile(ctx, s); file != nil { project = p
                if opts.verbose && opts.debug {
                    info(ctx, "%v: file %v\n", p, file).debug(1)
                }
                break
            }
        }

        if file == nil {
            prompt(ctx, "%v: configure-file failed\n", s)
            erro(ctx, "%v: %v", s, closured)
            errostack(ctx, 8, "target '%s' is not a file", s).debug(16)
            return
        }
    }

    if file == nil {
        erro(ctx, " no file target").debug(1)
        return
    } else if filename, _ = fullname(ctx, file); filename == "" {
        erro(ctx, " `%v` has empty filename", file).debug(1)
        return
    } else if !filepath.IsAbs(filename) {
        // fix: find the the full filename and set file target
        if closured == nil { closured = closureProjects(ctx) }
        for _, p := range closured {
            if f := p.FindFile(ctx, filename); f != nil {
                var prev Value
                file, filename, project = f, f.fullname(), p
                if prev, _ = ctx.autoSet("@", file); opts.debug {
                    info(ctx, "configure-file: %v: %s->%s (prev=%v)", p, f, filename, prev).debug(1)
                }
                break
            }
        }
    }
    if file.info == nil { if f := stat(ctx, filename, "", ""); f != nil { file.info = f.info }}
    if project == nil { project = ctx.Project() }
    if opts.debug && file != nil {
        var tv, _ = ctx.autoGet("@")
        info(ctx, "configure-file: %v: %v (%s) (%v, %v)", project, tv, file.fullname(), ctx.Project(), ctx).debug(1)
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
    defer func(s string, c *Scope) {
        configuredFiles[s] = c
    } (filename, closure)

    var data bytes.Buffer
    if buffer, _ := ctx.autoGet("-"); !isNil(buffer) {
        args = append(args, buffer)
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
        erro(ctx, "no input data").debug(6)
        return
    }

    var ( status string; same bool )
    if opts.verbose {
        defer func(st time.Time) {
            if same {
                if true { return } else { status = "unchanged" }
            } else if status == "" {
                status = fmt.Sprintf("outdated (%s)", filename)
            }
            printEnteringDirectory(ctx)
            prompt(ctx, "update %v …… %s (in %v)\n", trimPromptString(filename), status, time.Now().Sub(st)).debug(opts.debug, 6)
        } (time.Now())
    }

    if file.info != nil {
        var err error
        if same, err = crc64CheckFileModeContent(ctx, filename, data.Bytes(), opts.mode); err != nil {
            erro(ctx, " crc64 checksum failed: %v", err).debug(1)
            return
        } else if same {
            var tt = file.info.ModTime()
            for _, d := range merge(ctx.traversal().targets...) {
                if f, ok := d.(*File); !ok { continue } else
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

    if opts.debug {
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
    args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)
    if target, found := ctx.autoGet("@"); !found || isTrivial(target) {
        erro(ctx, " target '@' is not defined").debug(1)
        return
    }

    if def, ok := project.scope.Lookup("configure.names").(*Def); ok {
        args = append(args, mergeExpand(ctx, expandPlainValue, def.value)...)
    }

    var configs = make(map[string]*Def)
    for _, a := range args {
        var name = a.Strval(ctx)
        if _, ok := configs[name]; ok {
            continue
        } else if obj := project.resolveObject(ctx, name); obj == nil {
            erro(ctx, "undefined %v", name).debug(1)
            return
        } else if def, ok := obj.(*Def); ok {
            configs[name] = def
        }
    }
    for _, c := range project.configs {
        var name = c.Name(ctx)
        if def, ok := project.scope.Lookup(name).(*Def); ok {
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

func modifierConfigureInput(ctx Context, args ...Value) (result Value, _ travestates) {
    var opts = configureConvertOpts{ mode: os.FileMode(0600) }
    var dealArgs = func(args []Value, out *bytes.Buffer) []Value {
        var project = ctx.Project()
        if def, ok := project.scope.Lookup("configure.names").(*Def); ok {
            args = append(args, mergeExpand(ctx, expandPlainValue, def.value)...)
        }

        var configs = make(map[string]*Def)
        for _, a := range args {
            var name = a.Strval(ctx)
            if _, ok := configs[name]; ok {
                continue
            } else if obj := project.resolveObject(ctx, name); obj == nil {
                erro(ctx, "undefined %v", name).debug(1)
                return nil
            } else if def, ok := obj.(*Def); ok {
                configs[name] = def
            }
        }
        for _, c := range project.configs {
            var name = c.Name(ctx)
            if def, ok := project.scope.Lookup(name).(*Def); ok {
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
func modifierConfigureFile(ctx Context, args ...Value) (result Value, _ travestates) {
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
func modifierExtractConfiguration(ctx Context, args ...Value) (result Value, _ travestates) {
    var (
        pos = ctx.Position()
        opts = modifierExtractConfigurationOpts{ mode:os.FileMode(0640) } // sys default 0666
        pats []Value
    )
    for _, arg := range parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...) {
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
    if target, _ := ctx.autoGet("@"); isNil(target) {
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
    if value, _ := ctx.autoGet("^"); !isTrivial(value) {
        depends = mergeExpand(ctx, expandPlainValue, value)
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
