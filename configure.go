//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
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

var configurationOps = map[string] func(Context, map[string]Value, ...Value) (Value, error) {
    "answer":  configureAnswer,
    "bool":    configureBool,
    "dump":    configureDump,
    "option":  configureOption,
    "package": configurePackage,
}

func (ctx *defaultContext) configure() {
    var (
        project *Project
        file *os.File
        writer *bufio.Writer
        err error
    )
    defer func() {
        if writer != nil { if err := writer.Flush(); err != nil {} }
        if file != nil   { if err := file.Close();   err != nil {} }
    } ()

    // Remove all existing configuration.sm files
    for _, s := range configuration.clean {
        if _, e := os.Stat(s); e != nil {
            if false { ctx.prompt("%v\n", e).debug(1) }
        } else if e = os.Remove(s); e == nil {
            ctx.prompt("Remove %s\n", s)
        } else if true {
            ctx.prompt("Remove: %s\n", e).debug(1)
        }
    }

    var defs = make(map[string]*Def)
    for _, entry := range configuration.entries {
        var ctx = positional(ctx, entry.Position()) // redefine ctx
        if n := ctx.checkErrors(true); n > 0 {
            return
        } else if p := entry.OwnerProject(); p != project && p != nil {
            defs = make(map[string]*Def) // reset defs for p
            var f, e = p.openConfiguration(ctx)
            if e != nil {
                ctx.error("%v", e).debug(1)
                return
            } else if f != nil {
                if writer != nil {
                    if e = writer.Flush(); e != nil {
                        ctx.error("%v", e).debug(1)
                        return
                    }
                }
                if file != nil {
                    if e = file.Close(); e != nil {
                        ctx.error("%v", e).debug(1)
                        return
                    }
                }
            }

            file, writer = f, bufio.NewWriter(f)
            fmt.Fprintf(writer, "# %s (%s) configuration\n", p.spec, p.relPath)

            ctx.prompt("Project %s …… (%s)\n", p.spec, p.relPath)
            project = p
        }

        if val, brks := entry.Execute(ctx); len(brks) > 0 {
            for _, brk := range brks {
                if brk.what == breakErro {
                    ctx.error("execute '%v' failed: %v", entry, brk.error).debug(1)
                }
            }
        } else if entry.String() == "-check-file" {
            ctx.warn("configure %v: %v", entry, val).debug(1)
        }

        if s, e := entry.Target().Strval(ctx); e != nil {
            ctx.error("strval '%v' fail: %v", entry, e).debug(1)
        } else if def := project.scope.FindDef(s); def != nil {
            if d, ok := defs[s]; ok && d != nil {
                /*if d.value.cmp(def.value) != cmpEqual {
                    ctx.error("'%s' already configured: %v", d.name, d.value).at(entry.Position())
                    return
                }*/
              continue
            } else { defs[s] = def }
            if def.value == nil {
                // Set <nil> value with exec-assigning ('!=')
                // to a None value.
                fmt.Fprintf(writer, "%v !=\n", def.name)
            } else {
                vs := elementString(ctx, def, def.value, elemNoBrace)
                fmt.Fprintf(writer, "%v = %v\n", def.name, vs)
            }
        } else {
            ctx.error("`%s` unconfigured", s).debug(1)
        }
    }
    if err != nil {
        ctx.prompt("configure failed: %v\n", err).debug(1)
        return
    }
    if n := ctx.checkErrors(true); n > 0 {
        ctx.warn("configuration got %d errors", ctx.totalErrors()).debug(1)
        if options.failOnErrors { fail(ctx.Position(), "fail by %d errors", ctx.totalErrors()) }
        //return
    }
    printLeavingDirectory(ctx)
    return
}

func (p *Project) openConfiguration(ctx Context, ) (file *os.File, err error) {
    // defer setclosure(setclosure(cloctx.unshift(p.scope)))
    if f := p.configuration(ctx); f == nil {
        ctx.error("nil configuration file for %v", p).at(p.position).debug(1)
    } else if s := f.fullname(); s == "" {
        ctx.error("empty configuration file name: %v", f).at(p.position).debug(1)
    } else if err = os.MkdirAll(filepath.Dir(s), os.FileMode(0755)); err != nil {
        ctx.error("make path %s failed: %v", filepath.Dir(s), err).at(p.position).debug(1)
    } else if file, err = os.OpenFile(s, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0600)); err != nil {
        ctx.error("open configuration %s failed: %v", s, err).at(p.position).debug(1)
    }
    return
}

func configPrintf(ctx Context, str string, args... interface{}) {
    ctx.prompt(str, args...) //ctx.prompt( str, args...)
}

func configMessageDone(ctx Context, str string, args... interface{}) {
    if !strings.HasSuffix(str, "\n") { str += "\n" }
    configPrintf(ctx, str, args...)
}

// -dump
func configureDump(ctx Context, fields map[string]Value, params ...Value) (result Value, err error) {
    result, _ = ctx.autoGet("-")
    return
}

func configureBoolValue(ctx Context) (result bool, err error) {
    var (
        value, _ = ctx.autoGet("-")
        res Value
    )
    if isNil(value) {
        return
    } else if res, err = value.expand(ctx, expandPlainValue); err != nil {
        ctx.error("expand value failed: %v", err).debug(1)
        return
    } else if !isNil(res) && res != value {
        value = res
    }
    for i, v := range merge(value) {
        var a bool
        if v == nil {
            continue
        } else if a, err = v.True(ctx); err != nil {
            ctx.error("%v", err).debug(1)
            return
        } else if i == 0 {
            result = a
        } else {
            result = result && a
        }
        if !result { break }
    }
    return
}

// -bool
// -bool('message...')
func configureBool(ctx Context, fields map[string]Value, params ...Value) (result Value, err error) {
    var val bool
    if val, err = configureBoolValue(ctx); err != nil {
        ctx.error("configure bool value failed: %v", err).debug(1)
    } else {
        result = MakeBoolean(ctx.Position(), val)
    }
    return
}

// -answer
// -answer('message...')
func configureAnswer(ctx Context, fields map[string]Value, params ...Value) (result Value, err error) {
    var val bool
    if val, err = configureBoolValue(ctx); err != nil {
        ctx.error("configure bool value failed: %v", err).debug(1)
    } else {
        result = MakeAnswer(ctx.Position(), val)
    }
    return
}

// -option
// -option('message...')
func configureOption(ctx Context, fields map[string]Value, args ...Value) (result Value, err error) {
    if false { if target, _ := ctx.autoGet("@"); strings.HasPrefix(target.String(), "HAVE_TERMINFO") {
        defer func() {
            var v, _ = ctx.autoGet("-")
            ctx.info("%v: hyphen = %v, result = %v", target, v, result)
            ctx.info("%v", ctx).debug(6)
        } ()
    }}
    if result, _ = ctx.autoGet("-"); !isNil(result) {
        var res Value
        if res, err = result.expand(ctx, expandPlainValue); err != nil {
            ctx.error("expand configure option failed: %v", err).debug(1)
        } else if !isNil(res) && res != result {
            result = res
        }
    } else {
        result = MakeAnswer(ctx.Position(), false)
    }
    return
}

// -package finds system package in a way similar to cmake.find_package
func configurePackage(ctx Context, fields map[string]Value, args ...Value) (result Value, err error) {
    var names []string
    var optType packagetype = packageSmart
    for _, arg := range args {
        switch a := arg.(type) {
        case *Pair:
            var key, val string
            if key, err = a.Key.Strval(ctx);   err != nil { return }
            if val, err = a.Value.Strval(ctx); err != nil { return }
            switch key {
            case "type":
                switch val {
                case "", "smart": optType = packageSmart
                case "pkgconfig": optType = packageConfig
                default:      optType = packageUnknown
                    ctx.error("package: unknown type %v", val)
                    return
                }
            default:
                ctx.prompt("%v: package: `%v` unknown option", key)
            }
        default:
            var name string
            if name, err = a.Strval(ctx); err != nil { return }
            names = append(names, name)
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
                ctx.prompt("%v: package `%v`: unknown type\n", name)
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
        n, _ = fmt.Sscanf(err.Error(), exitstatusFmt, &status)
    }
    return
}

type commonConfigureOpts struct {
    silent bool `s,silent`
    noResetHyphen bool `r,reset` // reset hyphen value, aka. "-"
}
func executeConfigureEntry(ctx Context, opts *modifierConfigureOpts, entryName string, target Value, paramsOrig ...Value) (configured bool, result Value, err error) {
    if options.traceConfig { defer un(trace(t_config, fmt.Sprintf("executeConfigureEntry(%s %v)", entryName, ctx))) }

    var entry Entry
    if t := ctx.traversal(); t == nil {
        ctx.error("needs traversal context to configure: %v", ctx).debug(1)
        return
    } else if t.program.project.configure == nil {
        ctx.error("%v: .configure not provided for %v (%s)", t.program.project, target, entryName).debug(1)
        return
    } else if entry, err = t.program.project.configure.resolveEntry(ctx, "-"+entryName, false); err != nil {
        ctx.error("resolve entry '%v' failed: %v", entryName, err).debug(1)
        return
    } else if entry == nil {
        ctx.error("unknown configuration action `%v`, no such entry", entryName).debug(1)
        return
    }

    var (
        commonOpts commonConfigureOpts
        params []Value
        pos = ctx.Position()
        programs = entry.Programs()
        prog = programs[0]
        hyphenVal, /*hyphenFound*/_ = ctx.autoGet("-")
        verbose = opts.verbose
    )
    if paramsOrig, err = parseOpts(ctx, &commonOpts, paramsOrig...); err != nil {
        ctx.error("parse opts failed: %v", err).debug(1)
        return
    }
    if false && entryName == "library-c" && strings.HasPrefix(target.String(), "HAVE_TERMINFO") {
        defer func() {
            ctx.info("%s: target = %v, result = %v", entryName, target, result)
            ctx.info("%s: %v", entryName, ctx).debug(6)
        } ()
    }

    // Reset the result/output def '-'?
    // NOTE: have to reset hyphen to ensure configured value is saved
    if !commonOpts.noResetHyphen { ctx.autoSet("-", nil) }

    // verbose mode is on if silent flag was not set
    if !verbose && !commonOpts.silent { verbose = !commonOpts.silent }

    for _, par := range prog.params {
        switch par.name {
        case "LANG":   params = append(params, MakePair(pos, MakeBareword(pos, "LANG"),   MakeString(pos, ctx.traversal().program.language)))
        case "TARGET": params = append(params, MakePair(pos, MakeBareword(pos, "TARGET"), target))
        case "VALUE":  params = append(params, MakePair(pos, MakeBareword(pos, "VALUE"),  hyphenVal))
        }
    }
ForInParams:
    for _, a := range paramsOrig {
        var (
            pair *Pair
            key string
            ok bool
        )
        if pair, ok = a.(*Pair); !ok {
            ctx.error("unsupported parameter %v (%T)", a, a).of(a).debug(1)
            return
        } else if key, err = pair.Key.Strval(ctx); err != nil {
            ctx.error("stringify key '%v' failed: %v", pair.Key, err).of(pair.Key).debug(1)
            return
        }
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
            if false && verbose { ctx.prompt("%s", pair.Value) }
        } else if true {
            var params []string
            for _, p := range prog.params { params = append(params, p.name) }

            var at, _ = ctx.autoGet("@")
            ctx = positional(ctx, a.Position())
            ctx.warn("ignored param: %T %v; target: %T %v", a, a, at, at)
            ctx.warn("%v params = %v", at, params).at(prog.position).debug(16)
            return
        }
    }

    if t := ctx.traversal(); true {
        defer func(v bool) { t.configuration = v } (t.configuration)
        t.configuration = true
    }

    var brks breakers
    if result, brks = prog._execute(ctx, entry, params); ctx.checkErrors(true) > 0 {
        ctx.warn(`configure '%s' got %d error(s)`, entryName, ctx.totalErrors()).debug(1)
        if options.failOnErrors { fail(pos, "fail by %d errors", ctx.totalErrors()) }
    } else if !isNil(result) && result == hyphenVal {
        ctx.warn(`%v: configure yields value the same as input will be ignored: %v`, entry, result).debug(1)
        result = nil // simply discard the result as it's the same as the input (hyphen) value
    }
    if false && entryName == "library-c" && strings.HasPrefix(target.String(), "HAVE_TERMINFO") {
        ctx.info("%v: config = %s", entry, entryName).at(prog.Position())
        ctx.info("%v: config = %s", entry, entryName).at(entry.Position())
        ctx.info("%v: target = %v", entry, target)
        ctx.info("%v: - = %v", entry, hyphenVal)
        ctx.info("%v: params = %v", entry, prog.params)
        ctx.info("%v: params = %v", entry, paramsOrig)
        ctx.info("%v: params = %v", entry, params)
        ctx.info("%v: result = %v", entry, result)
        ctx.info("%v: %v", entry, ctx).debug(16)
    }
    if brks = brks.not(breakDone); brks.has() {
        for i, brk := range brks {
            switch brk.what {
            case breakUnkn: ctx.error("broken configuration %v for unknown reason", entry).debug(16)
            case breakErro: ctx.error("%d: %v", i, brk.error).debug(1)
            case breakFail: ctx.error("%d: %v", i, brk.message).debug(1)
            default: ctx.error("%d: %v", i, brk.what).debug(16)
            }
        }
    }
    configured = true
    return
}

func configureDo(ctx Context, opts *modifierConfigureOpts, target Value, name Value, args []Value) (configured bool, result Value, err error) {
    if options.traceConfig { defer un(trace(t_config, "configureDo")) }

    var (
        pos = ctx.Position()
        strName string
        params []Value
        infos []Value
    )
    if strName, err = name.Strval(ctx); err != nil {
        ctx.error("stringify '%v' failed: %v", name, err).debug(1)
        return
    } else if strName == "" {
        ctx.error("empty configure name: %v (%T)", name, name).debug(1)
        return
    }

ForArgs:
    for _, arg := range args {
        var elems []Value
        if elems, err = expandmerge2(ctx, expandPlainValue, arg); err != nil {
            ctx.error("merge list elements '%v' failed: %v", arg, err).of(arg).debug(1)
            return
        }
        for _, elem := range elems {
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
                ctx.error("parameter '%v' of %T is unsupported", tv, tv).of(arg).debug(1)
                return
            }
        }
    }

    defer func() {
        if err != nil {
            if e, ok := err.(*scanner.Error); ok {
                configMessageDone(ctx, "… (%v)", e.Brief())
            } else {
                configMessageDone(ctx, "… (%v)", err)
            }
        } else if isNil(result) {
            configMessageDone(ctx, "… <nil>")
        } else if isNone(result) {
            configMessageDone(ctx, "… <none>")
        } else if  s, e := result.Strval(ctx); e != nil {
            configMessageDone(ctx, "… (%v)", e)
            ctx.error("stringify configure result '%v' failed: %v", result, e).debug(1)
        } else {
            if s == "" { s = fmt.Sprintf("? (%s)", result) }
            configMessageDone(ctx, "… %v", s)
        }
    } ()

    if len(infos) == 0 {
        configPrintf(ctx, "%v %v …", target, args)
    } else {
        var msg string
        for _, info := range infos {
            if s, e := info.Strval(ctx); e == nil { msg += s } else {
                ctx.error("strval configure message failed: %v", e).debug(1)
                return
            }
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
        if result, err = config(ctx, nil, params...); err != nil {
            ctx.error("configure '%s' failed: %v", strName, err).debug(1)
        } else {
            if options.traceConfig {
                t_config.tracef("configured: %v, result = %v (%s)", configured, result, typeof(result))
            }
            configured = true
        }
    } else if configured, result, err = executeConfigureEntry(ctx, opts, strName, target, params...); err != nil {
        ctx.error("configure exec '%v' failed: %v", name, err).debug(1)
    }
    if configured && options.traceConfig {
        t_config.tracef("configured: %v, result = %v (%s)", configured, result, typeof(result))
    }
    return
}

type modifierConfigureOpts struct {
    accumulate bool `a,accumulate;a,add`
    verbose bool `v,verbose`
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
func modifierConfigure(ctx Context, args ...Value) (result Value, _ breakers) {
    if options.traceConfig { defer un(trace(t_config, fmt.Sprintf("modifierConfigure(%v) (reconfig=%v)", ctx, options.reconfigure))) }

    var ( pos = ctx.Position(); opts modifierConfigureOpts; err error )
    if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
        ctx.error("merge configure args failed: %v", err).debug(1)
        return
    } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
        ctx.error("parse configure opts failed: %v", err).debug(1)
        return
    }

    var program = ctx.Program()
    if program == nil {
        ctx.error("needs traversal context to configure: %v", ctx).debug(1)
        return
    }

    if program.project.configure == nil {
        if program.project.name == "configure" {
            if o := program.project.scope.Lookup(dotConfigure); !isNil(o) {
                if d, ok := o.(*Def); ok && !isNil(d.value) && !isNone(d.value) {
                    if val, err := d.value.True(ctx); err != nil {
                        ctx.error("truthify '%v' failed: %v", d.value, err)
                        ctx.error("value '%v' from here", d.value).of(d.value)
                        ctx.error("define for '%s' here", d.name).of(d).debug(1)
                    } else if val {
                        program.project.configure = program.project
                        if opts.verbose {
                            ctx.info("self-configure project enabled: %v", ctx.Project()).debug(1)
                        }
                    }
                }
            }
        }
        if program.project.configure == nil {
            ctx.error("%v: .configure not provided", program.project).debug(1)
            return
        }
    }

    var target, found = ctx.autoGet("@")
    if !found || isNil(target) || isNone(target) {
        ctx.error("target is nil: %s", ctx).debug(1)
        return
    }

    var name string
    if name, err = target.Strval(ctx); err != nil {
        ctx.error("stringify target '%v' failed: %v", target, err).of(target).debug(1)
        return
    } else if len(program.project.bases) == 0 {
        ctx.warn("%v: project has no bases (should have at least .configure)", name).of(target).debug(1)
    }

    var def *Def
    if def = program.scope.FindDef(name); def == nil {
        var alt Object
        def, alt = program.project.scope.define(ctx, DefConfig, name, nil)
        if def == nil && alt != nil { def, _ = alt.(*Def) }
    }
    if def == nil {
        ctx.error("cannot define configuration `%s`", name).debug(1)
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
            ctx.error("`%v` not configured (%v)", target, value).debug(1)
            return
        } else if value == def || value.refs(ctx, def) {
            return
        }
        switch v := value.(type) {
        default: err = def.set(ctx, DefConfig, value)
        case *ExecResult:
            var s string
            if /*v.wg.Wait()*/; v.Status == 0 && v.Stdout.Buf != nil {
                s = v.Stdout.Buf.String()
            } else if v.Stderr.Buf != nil {
                s = v.Stderr.Buf.String()
            }
            err = def.set(ctx, DefConfig, MakeString(pos, s))
        }
        if err != nil {
            ctx.error("set config '%s' value failed: %v", def.name, err).of(def).debug(1)
        }
        return
    } else if err = def.set(ctx, DefConfig, nil); err != nil {
        ctx.error("set config '%s' value failed: %v", def.name, err).of(def).debug(1)
        return
    }

    if false && strings.HasPrefix(name, "HAVE_TERMINFO") {
        defer func() {
            ctx.warn("value = %s", value)
            ctx.warn("result = %s", result)
            ctx.warn("%v: %p, %v", def.OwnerProject(), def, def)
            ctx.warn("%v", ctx).debug(12)
        } ()
    }

    var configured bool
ForConfig:
    for i, a := range args {
        if def.value == nil && i > 0 { break ForConfig }

        var ( name Value ; para []Value )
        switch arg := a.(type) {
        case *Argumented:
            if flag, okay := arg.value.(*Flag); !okay {
                ctx.error("`%v` is unsupported value (%T)", arg.value, arg.value).of(a).debug(1)
                return
            } else {
                name, para = flag.name, arg.args
            }
        case *Flag:
            if isNil(arg.name) || isNone(arg.name) {
                ctx.error("`%v` is unsupported flag (%T)", arg.name, arg.name).of(a).debug(1)
                return
            } else {
                name = arg.name
            }
        default:
            ctx.error("`%v` is unsupported (%T)", a, a).of(a).debug(1)
            return
        }
        if name == nil {
            ctx.error("unknown configure `%v` (%T)", a, a).of(a).debug(1)
            return
        }

        configured, value, err = configureDo(ctx, &opts, target, name, para)
        if err != nil {
            ctx.error("configure error: %v", err).debug(1)
            return
        } else if !configured {
            ctx.error("%s not configured for %v", name, target).debug(1)
            return
        } else if v := value; v == nil {
            value = MakeNil(a.Position())
        } else if isNil(v) || isNone(v) || isUndef(v) {
            // noop
        } else if v, err = value.expand(ctx, expandPlainValue); err != nil {
            ctx.error("configured with value error: %v", err).of(a).debug(1)
            return
        } else if !isNil(v) && v != value {
            value = v
        }

        if value == def || (!isNil(value) && value.refs(ctx, def)) {
            // Value is the Def, does nothing!
        } else if opts.accumulate {
            if err = def.append(ctx, value); err != nil {
                ctx.error("value accumulate error: %v", err).of(a).debug(1)
                return
            }
        } else if err = def.set(ctx, DefConfig, value); err != nil {
            ctx.error("set config value error: %v", err).of(a).debug(1)
            return
        }

        if def == nil && err == nil { configuration.done[def] = true }
        if options.traceConfig {
            t_config.tracef("configured: %v (%s) (%v)", value, typeof(value), def.origin)
        }
    }
    if !configured && err == nil {
        ctx.error("`%v` not configured", target).debug(1)
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

type modifierConfigureFileOpts struct {
    mode os.FileMode `m,mode`
    makePath bool `p,path`
    reconfig bool `r,reconfig`
    verbose bool `v,verbose`
    debug bool `d,debug`
}
// configure-file modifier (see also builtinConfigureFile), example usage:
// 
//     config.h: config.h.in [(configure-file)]
//     
func modifierConfigureFile(ctx Context, args ...Value) (result Value, _ breakers) {
    var (
        opts = modifierConfigureFileOpts{ mode: os.FileMode(0600) }
        project *Project
        filename string
        file *File
        err error
    )
    if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
        ctx.error("merge configure-file args failed: %v", err).debug(1)
        return
    } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
        ctx.error("parse configure-file opts failed: %v", err).debug(1)
        return
    }

    if target, found := ctx.autoGet("@"); !found || isNil(target) {
        ctx.error("target '@' is not defined").debug(1)
        return
    } else if file, _ = target.(*File); file == nil {
        var ( s string; okay bool )
        if s, err = target.Strval(ctx); err != nil {
            ctx.error("strval target '%v' failed: %v", target, err).debug(1)
            return
        }

        for _, p := range ctx.closureProjects() {
            if file = p.FindFile(ctx, s); file != nil { project = p
                if opts.verbose && opts.debug {
                    ctx.info("%v: file %v\n", p, file).debug(1)
                }
                break
            }
        }

        if err != nil {
            ctx.error("find file '%s' failed: ", s, err).debug(1)
            return
        } else if !okay {
            ctx.error("target '%s' is not a file", s).debug(1)
            return
        }
    }

    if file == nil {
        ctx.error("no file target").debug(1)
        return
    } else if filename, _ = fullname(ctx, file); filename == "" {
        ctx.error("`%v` has empty filename", file).debug(1)
        return
    } else if !filepath.IsAbs(filename) {
        // FIXES: match file map to have the full filename.
        for _, p := range ctx.closureProjects() {
            if f := p.FindFile(ctx, filename); f != nil {
                var prev Value
                file, filename, project = f, f.fullname(), p
                if prev, _ = ctx.autoSet("@", file); opts.debug {
                    ctx.info("configure-file: %v: %s->%s (prev=%v)", p, f, filename, prev).debug(1)
                }
                break
            }
        }
        if err != nil {
            ctx.error("locate file '%v' failed: %v", filename, err).debug(1)
            return
        }
    }
    if file.info == nil { if f := stat(ctx, filename, "", ""); f != nil { file.info = f.info }}
    if project == nil { project = ctx.Project() }
    if opts.debug && file != nil {
        var tv, _ = ctx.autoGet("@")
        ctx.info("configure-file: %v: %v (%s) (%v, %v)", project, tv, file.fullname(), ctx.Project(), ctx).debug(1)
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
        if err == nil { configuredFiles[s] = c } else { ctx.error("%v", err) }
    } (filename, closure)

    var data bytes.Buffer
    if buffer, _ := ctx.autoGet("-"); !isNil(buffer) {
        args = append(args, buffer)
    }
    for _, arg := range args {
        var str string
        if str, err = arg.Strval(ctx); err != nil {
            ctx.error("%v", err).debug(1)
            return
        }
        if str == "" { continue }
        if err = configure(ctx, &data, ctx.Project(), str); err != nil {
            ctx.error("%v", err).debug(1)
            return
        }
    }
    if data.Len() == 0 {
        ctx.error("no input data").debug(1)
        return
    }

    var ( status string; same bool )
    if opts.verbose {
        defer func(st time.Time) {
            if err != nil { status = err.Error() } else if same {
                if true { return } else { status = "unchanged" }
            } else if status == "" {
                status = fmt.Sprintf("outdated (%s)", filename)
            }
            printEnteringDirectory(ctx)
            ctx.prompt("update %v …… %s (in %v)\n", trimPromptString(filename), status, time.Now().Sub(st)).debug(opts.debug, 6)
        } (time.Now())
    }
    if file.info != nil {
        if same, err = crc64CheckFileModeContent(ctx, filename, data.Bytes(), opts.mode); err != nil {
            ctx.error("crc64 checksum failed: %v", err).debug(1)
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
        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
            ctx.error("%v", err).debug(1)
            return
        }
    }

    if err = ioutil.WriteFile(filename, data.Bytes(), opts.mode); err != nil {
        ctx.error("%v", err).debug(1)
        return
    } else if file.info != nil { result = file } else {
        if file.info, err = os.Stat(filename); err == nil {
            ctx.Globe().stamp(filename, file.info.ModTime())
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
func modifierExtractConfiguration(ctx Context, args ...Value) (result Value, _ breakers) {
    var (
        pos = ctx.Position()
        opts = modifierExtractConfigurationOpts{ mode:os.FileMode(0640) } // sys default 0666
        err error
    )
    if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
        ctx.error("merge args failed: %v", err).debug(1)
        return
    } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
        ctx.error("parse opts failed: %v", err).debug(1)
        return
    }

    var pats []Value
    for _, arg := range args {
        switch a := arg.(type) {
        case *Group: pats = append(pats, a.Elems...)
        default:     pats = append(pats, a)
        }
    }
    if len(pats) == 0 {
        err = fmt.Errorf("extract-configuration: missing file names (patterns)")
        return
    }
    if len(opts.rxs) == 0 {
        err = fmt.Errorf("extract-configuration: missing -rx=... flags")
        return
    }
    if opts.target == "" {
        opts.target = "configuration"
    }

    var outFile string
    if target, _ := ctx.autoGet("@"); isNil(target) {
        ctx.error("target '@' is undefined").debug(1)
        return
    }  else if outFile, err = target.Strval(ctx); err != nil {
        ctx.error("strval '%v' failed: %v", target, err).debug(1)
        return
    }
    if opts.makePath {
        if err = os.MkdirAll(filepath.Dir(outFile), os.FileMode(0755)); err != nil {
            ctx.error("make path failed: %v", err).debug(1)
            return
        }
    }

    var ( fil *os.File; out *bufio.Writer )
    if fil, err = os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, opts.mode); err != nil {
        ctx.error("open file failed: %v", err).debug(1)
        return
    } else { out = bufio.NewWriter(fil) }
    defer func() {
        out.Flush()
        fil.Close()
    }()

    var depends []Value
    if value, _ := ctx.autoGet("^"); isNil(value) {
        // ...
    } else if depends, err = expandmerge2(ctx, expandPlainValue, value); err != nil {
        ctx.error("merge depends failed: %v", err).debug(1)
        return
    }

    var filterOpts builtinFilterOpts
    var sources []Value
    for _, depend := range depends {
        var a []Value
        switch d := depend.(type) {
        case *File:
            if a, err = filterValues(ctx, pats, filterOpts, false, d); err != nil {
                ctx.error("filter values failed: %v", err).debug(1)
            } else { sources = append(sources, a...) }
        case *Path:
            var s string
            if s, err = d.Strval(ctx); err != nil {
                ctx.error("strval '%v' failed: %v", d, err).debug(1)
                return
            }
            err = walkFiles(ctx, s, pats, func(file *File, err error) error {
                if err == nil { sources = append(sources, file) }
                return err
            })
        default:
            var s string
            if s, err = d.Strval(ctx); err != nil {
                ctx.error("strval '%v' failed: %v", d, err).debug(1)
                return
            }

            dir := filepath.Dir(s)
            name := filepath.Base(s)
            file := stat(ctx, name, "", dir)
            if file == nil {
                ctx.error("extract-configuration: `%s` file not found", name).debug(1)
                return
            } else if file.info.IsDir() {
                err = walkFiles(ctx, s, pats, func(file *File, err error) error {
                    if err == nil { sources = append(sources, file) }
                    return err
                })
            } else if a, err = filterValues(ctx, pats, filterOpts, false, d); err != nil {
                ctx.error("filter values failed: %v", err).debug(1)
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
        default:
            if s, err = v.Strval(ctx); err != nil {
                break ForSources
            }
        }
        if f, err = os.Open(s); err != nil {
            ctx.prompt("%v: (configure) %v: %v\n", pos, source, err)
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
