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
    entries []*RuleEntry // order list
    clean []string
}{
    fset: token.NewFileSet(),
    libraries: make(map[string]*libraryinfo),
    packages: make(map[string]*packageinfo),
    done: make(map[*Def]bool),
}

var configurationOps = map[string] func(*traversal, map[string]Value, ...Value) (Value, error) {
    "answer":  configureAnswer,
    "bool":    configureBool,
    "dump":    configureDump,
    "option":  configureOption,
    "package": configurePackage,
}

func do_configuration(ctx Context) {
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
            if false { diag.prompt("%v\n", e).debug(1) }
        } else if e = os.Remove(s); e == nil {
            diag.prompt("Remove %s\n", s)
        } else if true {
            diag.prompt("Remove: %s\n", e).debug(1)
        }
    }

    var defs = make(map[string]*Def)
    for _, entry := range configuration.entries {
        var entryCtx = contextAt(entry.position, ctx)
        if p := entry.OwnerProject(); p != project && p != nil {
            defs = make(map[string]*Def) // reset defs for p
            var f, e = openConfigurationFile(entryCtx, p)
            if e != nil {
                diag.errorAt(entry.position, "%v", e).debug(1)
                return
            } else if f != nil {
                if writer != nil {
                    if e = writer.Flush(); e != nil {
                        diag.errorAt(entry.position, "%v", e).debug(1)
                        return
                    }
                }
                if file != nil {
                    if e = file.Close(); e != nil {
                        diag.errorAt(entry.position, "%v", e).debug(1)
                        return
                    }
                }
            }

            file, writer = f, bufio.NewWriter(f)
            fmt.Fprintf(writer, "# %s (%s) configuration\n", p.spec, p.relPath)

            diag.prompt("Project %s …… (%s)\n", p.spec, p.relPath)
            project = p
        }

        if val, brks := entry.Execute(entryCtx); len(brks) > 0 {
            for _, brk := range brks {
                if brk.what == breakErro {
                    diag.errorAt(entry.position, "execute '%v' failed: %v", entry, brk.error).debug(1)
                }
            }
        } else if entry.String() == "-check-file" {
            diag.warnAt(entry.position, "configure %v: %v", entry, val).debug(true, 1)
        }
        if s, e := entry.target.Strval(entryCtx); e != nil {
            diag.errorAt(entry.position, "strval '%v' fail: %v", entry, e).debug(1)
        } else if def := project.scope.FindDef(s); def != nil {
            if d, ok := defs[s]; ok && d != nil {
                /*if d.value.cmp(def.value) != cmpEqual {
                    diag.errorAt(entry.position, "'%s' already configured: %v", d.name, d.value)
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
            diag.errorAt(entry.position, "`%s` unconfigured", s).debug(1)
        }
    }
    if err != nil {
        diag.prompt("configure: %v\n", err).debug(1)
        return
    }

    printLeavingDirectory()
    return
}

func openConfigurationFile(ctx Context, p *Project) (file *os.File, err error) {
    defer setclosure(setclosure(cloctx.unshift(p.scope)))
    if f := p.configurationFile(ctx); f == nil {
        diag.errorAt(p.position, "nil configuration file for %v", p).debug(1)
    } else if s := f.fullname(); s == "" {
        diag.errorAt(p.position, "empty configuration file name: %v", f).debug(1)
    } else if err = os.MkdirAll(filepath.Dir(s), os.FileMode(0755)); err != nil {
        diag.errorAt(p.position, "make path %s failed: %v", filepath.Dir(s), err).debug(1)
    } else if file, err = os.OpenFile(s, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0600)); err != nil {
        diag.errorAt(p.position, "open configuration %s failed: %v", s, err).debug(1)
    }
    return
}

func configPrintf(ctx Context, str string, args... interface{}) {
    diag.prompt(str, args...) //diag.prompt( str, args...)
}

func configMessageDone(ctx Context, str string, args... interface{}) {
    if !strings.HasSuffix(str, "\n") { str += "\n" }
    configPrintf(ctx, str, args...)
}

// -dump
func configureDump(t *traversal, fields map[string]Value, params ...Value) (result Value, err error) {
    result, _ = t.Get("-")
    return
}

func configureBoolValue(t *traversal) (result bool, err error) {
    var (
        pos = t.Position()
        value, okay = t.Get("-")
        res Value
    )
    if !okay || isNil(value) {
        return
    } else if res, err = value.expand(t, expandPlainValue); err != nil {
        diag.errorAt(pos, "expand value failed: %v", err).debug(1)
        return
    } else if !isNil(res) && res != value {
        value = res
    }
    for i, v := range merge(value) {
        var a bool
        if v == nil {
            continue
        } else if a, err = v.True(t); err != nil {
            diag.errorAt(pos, "%v", err).debug(1)
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
func configureBool(t *traversal, fields map[string]Value, params ...Value) (result Value, err error) {
    var ( pos = t.Position(); val bool )
    if val, err = configureBoolValue(t); err != nil {
        diag.errorAt(pos, "configure bool value failed: %v", err).debug(1)
    } else {
        result = MakeBoolean(pos, val)
    }
    return
}

// -answer
// -answer('message...')
func configureAnswer(t *traversal, fields map[string]Value, params ...Value) (result Value, err error) {
    var ( pos = t.Position(); val bool )
    if val, err = configureBoolValue(t); err != nil {
        diag.errorAt(pos, "configure bool value failed: %v", err).debug(1)
    } else {
        result = MakeAnswer(pos, val)
    }
    return
}

// -option
// -option('message...')
func configureOption(t *traversal, fields map[string]Value, args ...Value) (result Value, err error) {
    var ( pos = t.Position(); okay bool )
    if result, okay = t.Get("-"); okay && !isNil(result) {
        var res Value
        if res, err = result.expand(t, expandPlainValue); err != nil {
            diag.errorAt(pos, "expand configure option failed: %v", err).debug(1)
        } else if !isNil(res) && res != result {
            result = res
        }
    } else {
        result = MakeAnswer(pos, false)
    }
    return
}

// -package finds system package in a way similar to cmake.find_package
func configurePackage(t *traversal, fields map[string]Value, args ...Value) (result Value, err error) {
    var pos = t.Position()
    var names []string
    var optType packagetype = packageSmart
    for _, arg := range args {
        switch a := arg.(type) {
        case *Pair:
            var key, val string
            if key, err = a.Key.Strval(t);   err != nil { return }
            if val, err = a.Value.Strval(t); err != nil { return }
            switch key {
            case "type":
                switch val {
                case "", "smart": optType = packageSmart
                case "pkgconfig": optType = packageConfig
                default:      optType = packageUnknown
                    diag.errorAt(pos, "package: unknown type %v", val)
                    return
                }
            default:
                diag.prompt("%v: package: `%v` unknown option", key)
            }
        default:
            var name string
            if name, err = a.Strval(t); err != nil { return }
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
                diag.prompt("%v: package `%v`: unknown type\n", name)
            }
            if info != nil {
                configuration.packages[name] = info
                result = MakeAnswer(pos, true)
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

func configureExec(t *traversal, opts *modifierConfigureOpts, s string, target Value, paramsOrig ...Value) (configured bool, result Value, err error) {
    if optionTraceConfig { defer un(trace(t_config, fmt.Sprintf("configureExec(%s %v)", s, t.entry.target))) }

    var pos = t.Position()
    var projectConfigure = t.program.project.configure
    if  projectConfigure == nil {
        diag.errorAt(pos, "no .configure provided").debug(1)
        return
    }

    var entry *RuleEntry
    if entry, err = projectConfigure.resolveEntry(t, "-"+s, false); err != nil {
        diag.errorAt(pos, "resolve '%v' failed: %v", s, err).debug(1)
        return
    } else if entry == nil {
        diag.errorAt(pos, "unknown configuration action `%v`, no such entry", s).debug(1)
        return
    }

    if false { defer setclosure(setclosure(cloctx.unshift(t.program.scope))) }
    if false { diag.infoAt(pos, "configureExec(%v %v): %v, %v", entry, t.entry, paramsOrig, cloctx).debug(true, 1)}

    var buffer, _ = t.Get("-")
    var silent bool
    var params []Value
    var prog = entry.programs[0]
    for _, par := range prog.params {
        switch par.name {
        case "LANG":   params = append(params, MakePair(pos, MakeBareword(pos, "LANG"),   MakeString(pos, t.program.language)))
        case "TARGET": params = append(params, MakePair(pos, MakeBareword(pos, "TARGET"), target))
        case "VALUE":  params = append(params, MakePair(pos, MakeBareword(pos, "VALUE"),  buffer))
        }
        for _, a := range paramsOrig {
            if f, ok := a.(*Flag); ok {
                if s, _ := f.name.Strval(t); s == "s" { silent = true }
            } else if ap, ok := a.(*Pair); ok {
                s, e := ap.Key.Strval(t)
                if e != nil {
                    diag.errorOf(ap.Key, "stringify key '%v' failed: %v", ap.Key, e).debug(1)
                    return
                }
                if par.name == s {
                       params = append(params, a)
                } else if par.name == strings.ToUpper(s) {
                       params = append(params, MakePair(pos, MakeBareword(pos, par.name), ap.Value))
                } else if false {
                    diag.warnOf(a, "unknown parameter: %v", a).debug(1)
                    return
                }
            } else {
                diag.errorOf(a, "unsupported parameter %v (%T)", a, a).debug(1)
                return
            }
        }
    }

    // Turn on verbose mode if no silent flag was set
    var optVerbose = opts.verbose
    if !optVerbose && !silent { optVerbose = true }

    defer func(v bool) { t.isConfigureExecution = false } (t.isConfigureExecution)
    t.isConfigureExecution = true

    var brks breakers
    result, brks = prog.execute(t, entry, params)
    if false && (isNil(result) || isNone(result)) {
        var target, _ = t.Get("@")
        diag.infoAt(pos, "%v: %v = %v, %v, params = %v",
            entry, target, result, target, params).debug(true,1)
    }
    if brks = brks.not(breakDone); brks.has() {
        for i, brk := range brks {
            switch brk.what {
            case breakUnkn: diag.errorAt(pos, "broken configuration %v for unknown reason", entry).debug(16)
            case breakErro: diag.errorAt(pos, "%d: %v", i, brk.error).debug(1)
            case breakFail: diag.errorAt(pos, "%d: %v", i, brk.message).debug(1)
            default: diag.errorAt(pos, "%d: %v", i, brk.what).debug(16)
            }
        }
    } else if optVerbose {
        var res bool
        if isNil(result) || isNone(result) { res = true } else {
            if res, err = result.True(t); err != nil {
                diag.errorAt(pos, "truthify '%v' failed: %v", result, err).debug(1)
            }
        }
        if n := diag.numErrors(); /*!res || */n > 0 && false {
            var t, _ = target.Strval(t)
            diag.errorAt(pos, "s=%v target=%v result=%v res=%v", s, t, result, res).debug(1)
        }
        if n := diag.checkErrors(true); n > 0 {
            diag.warnAt(pos, "got %d error(s)", n).debug(1)
        }
    }

    configured = true
    return
}

func configureDo(t *traversal, opts *modifierConfigureOpts, target Value, def, name Value, args []Value) (configured bool, result Value, err error) {
    if optionTraceConfig { defer un(trace(t_config, "configureDo")) }

    var (
        pos = t.Position()
        strName string
        params []Value
        infos []Value
    )
    if strName, err = name.Strval(t); err != nil {
        diag.errorAt(pos, "stringify '%v' failed: %v", name, err).debug(1)
        return
    } else if strName == "" {
        diag.errorAt(pos, "empty configure name: %v (%T)", name, name).debug(1)
        return
    }

ForArgs:
    for _, arg := range args {
        var elems []Value
        if elems, err = mergeresult2(expandall2(t, expandPlainValue, arg)); err != nil {
            diag.errorOf(arg, "merge list elements '%v' failed: %v", arg, err).debug(1)
            return
        }
        for _, elem := range elems {
            switch t := elem.(type) {
            case *None, *Nil: continue
            case *Pair:
                params = append(params, t)
                continue ForArgs
            case *Raw, *String, *Compound:
                var ap = t.Position()
                params = append(params, MakePair(ap, MakeBareword(ap, "INFO"), t))
                infos = append(infos, t)
                continue ForArgs
            default:
                diag.errorOf(arg, "parameter '%v' of %T is unsupported", t, t).debug(1)
                return
            }
        }
    }

    defer func() {
        if err != nil {
            if e, ok := err.(*scanner.Error); ok {
                configMessageDone(t, "… (%v)", e.Brief())
            } else {
                configMessageDone(t, "… (%v)", err)
            }
        } else if isNil(result) {
            configMessageDone(t, "… <nil>")
        } else if isNone(result) {
            configMessageDone(t, "… <none>")
        } else if  s, e := result.Strval(t); e != nil {
            configMessageDone(t, "… (%v)", e)
            diag.errorAt(pos, "stringify configure result '%v' failed: %v", result, e).debug(1)
        } else {
            if s == "" { s = fmt.Sprintf("? (%s)", result) }
            configMessageDone(t, "… %v", s)
        }
    } ()

    if len(infos) == 0 {
        configPrintf(t, "%v %v …", target, args)
    } else {
        var msg string
        for _, info := range infos {
            if s, e := info.Strval(t); e == nil { msg += s } else {
                diag.errorAt(pos, "strval configure message failed: %v", e).debug(1)
                return
            }
        }
        if msg != "" { configPrintf(t, "%s …", msg) }
    }

    // Process configurations like:
    //   -bool
    //   -option
    //   -package
    //   ...
    if config, ok := configurationOps[strName]; ok {
        params = append(params, MakePair(pos, MakeBareword(pos, "TARGET"), target))
        if result, err = config(t, nil, params...); err != nil {
            diag.errorAt(pos, "configure '%s' failed: %v", strName, err).debug(1)
        } else {
            if optionTraceConfig {
                t_config.tracef("configured: %v, result = %v (%s)", configured, result, typeof(result))
            }
            configured = true
        }
    } else if configured, result, err = configureExec(t, opts, strName, target, params...); err != nil {
        diag.errorAt(pos, "configure exec '%v' failed: %v", name, err).debug(1)
    }
    if configured && optionTraceConfig {
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
func modifierConfigure(t *traversal, args ...Value) (result Value, _ breakers) {
    if optionTraceConfig { defer un(trace(t_config, fmt.Sprintf("modifierConfigure(%v) (reconfig=%v)", t.entry.target, options.reconfigure))) }

    var ( pos = t.Position(); opts modifierConfigureOpts; err error )
    if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
        diag.errorAt(pos, "merge configure args failed: %v", err).debug(1)
        return
    } else if args, err = parseOpts(t, &opts, args...); err != nil {
        diag.errorAt(pos, "parse configure opts failed: %v", err).debug(1)
        return
    }

    if t.program.project.configure == nil {
        if t.program.project.name == "configure" {
            if o := t.program.project.scope.Lookup(dotConfigure); !isNil(o) {
                if d, ok := o.(*Def); ok && !isNil(d.value) && !isNone(d.value) {
                    if val, err := d.value.True(t); err != nil {
                        diag.errorAt(pos, "truthify '%v' failed: %v", d.value, err)
                        diag.errorOf(d.value, "value '%v' from here", d.value)
                        diag.errorOf(d, "define for '%s' here", d.name).debug(1)
                    } else if val {
                        t.program.project.configure = t.program.project
                        if opts.verbose {
                            diag.infoAt(pos, "self-configure project enabled: %v", t.project).debug(1)
                        }
                    }
                }
            }
        }
        if t.program.project.configure == nil {
            diag.errorAt(pos, "%v: .configure not provided", t.program.project).debug(1)
            return
        }
    }

    var target, _ = t.Get("@")
    if isNil(target) || isNone(target) {
        diag.errorAt(pos, "target is nil for entry '%s'", t.entry.target).debug(1)
        return
    }

    var name string
    if name, err = target.Strval(t); err != nil {
        diag.errorOf(target, "stringify target '%v' failed: %v", target, err).debug(1)
        return
    }
    if len(t.program.project.bases) == 0 {
        diag.warnOf(target, "%v: %v %v", name, t.program.project.bases, cloctx).debug(1)
    }

    var def, alt = t.program.project.scope.define(t, DefConfig, name, nil)
    if alt != nil { def, _ = alt.(*Def) }
    if def != nil { result = def } else {
        diag.errorAt(pos, "cannot define configuration `%s`", name).debug(1)
        return
    }
    if optionTraceConfig {
        t_config.tracef("%s: %v (%T)", def.name, def.value, def.value)
        defer func() { t_config.tracef("%s: %v (%T)", def.name, def.value, def.value) } ()
    }
    if !isNil(def.value) { // Check if it's already configured?
        if !options.reconfigure { return } // return if not reconfigure
        if done, found := configuration.done[def]; done && found { return }
    }

    var value Value
    if len(args) == 0 { // Empty configuration: (configure)
        if value, _ = t.Get("-"); value == nil {
            diag.errorAt(pos, "`%v` not configured (%v)", target, value).debug(1)
            return
        } else if value == def || value.refs(t, def) {
            return
        }
        switch v := value.(type) {
        default: err = def.set(t, DefConfig, value)
        case *ExecResult:
            var s string
            if v.wg.Wait(); v.Status == 0 && v.Stdout.Buf != nil {
                s = v.Stdout.Buf.String()
            } else if v.Stderr.Buf != nil {
                s = v.Stderr.Buf.String()
            }
            err = def.set(t, DefConfig, MakeString(pos, s))
        }
        if err != nil {
            diag.errorOf(def, "set config '%s' value failed: %v", def.name, err).debug(1)
        }
        return
    } else if err = def.set(t, DefConfig, nil); err != nil {
        diag.errorOf(def, "set config '%s' value failed: %v", def.name, err).debug(1)
        return
    }

    var configured bool
ForConfig:
    for i, a := range args {
        if def.value == nil && i > 0 { break ForConfig }

        var ( name Value ; para []Value )
        switch arg := a.(type) {
        case *Argumented:
            if flag, okay := arg.value.(*Flag); !okay {
                diag.errorOf(a, "`%v` is unsupported value (%T)", arg.value, arg.value).debug(1)
                return
            } else {
                name, para = flag.name, arg.args
            }
        case *Flag:
            if isNil(arg.name) || isNone(arg.name) {
                diag.errorOf(a, "`%v` is unsupported flag (%T)", arg.name, arg.name).debug(1)
                return
            } else {
                name = arg.name
            }
        default:
            diag.errorOf(a, "`%v` is unsupported (%T)", a, a).debug(1)
            return
        }
        if name == nil {
            diag.errorOf(a, "unknown configure `%v` (%T)", a, a).debug(1)
            return
        }

        configured, value, err = configureDo(t, &opts, target, def, name, para)
        if err != nil {
            diag.errorAt(pos, "configure error: %v", err).debug(1)
            return
        } else if !configured {
            diag.errorAt(pos, "%s not configured for %s", name, target).debug(1)
            return
        } else if v := value; v == nil {
            value = MakeNil(a.Position())
        } else if isNil(v) || isNone(v) || isUndef(v) {
            // noop
        } else if v, err = value.expand(t, expandPlainValue); err != nil {
            diag.errorOf(a, "configured with value error: %v", err).debug(1)
            return
        } else if !isNil(v) && v != value {
            value = v
        }

        if value == def || (!isNil(value) && value.refs(t, def)) {
            // Value is the Def, does nothing!
        } else if opts.accumulate {
            if err = def.append(t, value); err != nil {
                diag.errorOf(a, "value accumulate error: %v", err).debug(1)
                return
            }
        } else if err = def.set(t, DefConfig, value); err != nil {
            diag.errorOf(a, "set config value error: %v", err).debug(1)
            return
        }

        if def == nil && err == nil { configuration.done[def] = true }
        if optionTraceConfig {
            t_config.tracef("configured: %v (%s) (%v)", value, typeof(value), def.origin)
        }
    }
    if !configured && err == nil {
        diag.errorAt(pos, "`%v` not configured", target).debug(1)
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
func modifierConfigureFile(t *traversal, args ...Value) (result Value, _ breakers) {
    var (
        opts = modifierConfigureFileOpts{ mode: os.FileMode(0600) }
        pos = t.Position()
        project *Project
        filename string
        file *File
        err error
    )
    if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
        diag.errorAt(pos, "merge configure-file args failed: %v", err).debug(1)
        return
    } else if args, err = parseOpts(t, &opts, args...); err != nil {
        diag.errorAt(pos, "parse configure-file opts failed: %v", err).debug(1)
        return
    }

    if target, okay := t.Get("@"); !okay || isNil(target) {
        diag.errorAt(pos, "target '@' is not defined").debug(1)
        return
    } else if file, _ = target.(*File); file == nil {
        var ( s string; okay bool )
        if s, err = target.Strval(t); err != nil {
            diag.errorAt(pos, "strval target '%v' failed: %v", target, err).debug(1)
            return
        }

        okay, err = t.forClosuredProjects(func(p *Project) (ok bool, err error) {
            if file = p.FindFile(t, s); file != nil { project, ok = p, true }
            if opts.verbose && opts.debug && file != nil {
                diag.infoAt(pos, "%v: file %v\n", p, file).debug(1)
            }
            return
        })

        if err != nil {
            diag.errorAt(pos, "find file '%s' failed: ", s, err).debug(1)
            return
        } else if !okay {
            diag.errorAt(pos, "target '%s' is not a file", s).debug(1)
            return
        }
    }

    if file == nil {
        diag.errorAt(pos, "no file target").debug(1)
        return
    } else if filename, _ = fullname(t, file); filename == "" {
        diag.errorAt(pos, "`%v` has empty filename", file).debug(1)
        return
    } else if !filepath.IsAbs(filename) {
        // FIXES: match file map to have the full filename.
        t.forClosuredProjects(func(p *Project) (ok bool, err error) {
            if f := p.FindFile(t, filename); f != nil {
                ok, file, filename, project = true, f, f.fullname(), p
                if _, okay := t.Set("@", file); !okay { // reset target file
                    diag.errorAt(pos, "set '@' failed: %v", file).debug(1)
                }
                if opts.debug {
                    diag.infoAt(pos, "configure-file: %v: %s->%s", p, f, filename).debug(1)
                }
            }
            return
        })
        if err != nil {
            diag.errorAt(pos, "locate file '%v' failed: %v", filename, err).debug(1)
            return
        }
    }
    if file.info == nil { if f := stat(t, filename, "", ""); f != nil { file.info = f.info }}
    if project == nil { project = t.project }
    if opts.debug && file != nil {
        var target, _ = t.Get("@")
        diag.infoAt(pos, "configure-file: %v: %v (%s) (%v, %v) (%v)",
            project, target, file.fullname(), t.project, t.closure.comment, cloctx)
    }

    // Check previously configured files, we only configure once unless
    // optReconfig is true.
    var closure *Scope
    if configuredFiles != nil {
        var okay bool
        closure, okay = configuredFiles[filename]
        if okay && closure != nil && !opts.reconfig { return }
    }
    if closure == nil { closure = t.closure }
    defer func(s string, c *Scope) {
        if err == nil { configuredFiles[s] = c } else { diag.errorAt(pos, "%v", err) }
    } (filename, closure)

    var data bytes.Buffer
    if buffer, okay := t.Get("-"); okay && !isNil(buffer) {
        args = append(args, buffer)
    }
    for _, arg := range args {
        var str string
        if str, err = arg.Strval(t); err != nil {
            diag.errorAt(pos, "%v", err).debug(1)
            return
        }
        if str == "" { continue }
        if err = configure(t, &data, closure.project, str); err != nil {
            diag.errorAt(pos, "%v", err).debug(1)
            return
        }
    }
    if data.Len() == 0 {
        diag.errorAt(pos, "no input data").debug(1)
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
            printEnteringDirectory()
            diag.prompt("update %v …… %s (in %v)\n", trimPromptString(filename), status, time.Now().Sub(st)).debug(opts.debug, 6)
        } (time.Now())
    }
    if file.info != nil {
        if same, err = crc64CheckFileModeContent(filename, data.Bytes(), opts.mode); err != nil {
            diag.errorAt(pos, "crc64 checksum failed: %v", err).debug(1)
            return
        } else if same {
            var tt = file.info.ModTime()
            for _, d := range merge(t.targets...) {
                if f, ok := d.(*File); !ok { continue } else
                if dt := f.info.ModTime(); dt.After(tt) { tt = dt }
            }
            if tt.After(file.info.ModTime()) { err = touch(t, file, 0, false, tt) }
            result = file
            return
        }
    } else if dir := filepath.Dir(filename); opts.makePath && dir != "." && dir != PathSep {
        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
            diag.errorAt(pos, "%v", err).debug(1)
            return
        }
    }

    if err = ioutil.WriteFile(filename, data.Bytes(), opts.mode); err != nil {
        diag.errorAt(pos, "%v", err).debug(1)
        return
    } else if file.info != nil { result = file } else {
        if file.info, err = os.Stat(filename); err == nil {
            t.Stamp(filename, file.info.ModTime())
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
func modifierExtractConfiguration(t *traversal, args ...Value) (result Value, _ breakers) {
    var (
        pos = t.Position()
        opts = modifierExtractConfigurationOpts{ mode:os.FileMode(0640) } // sys default 0666
        err error
    )
    if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
        diag.errorAt(pos, "merge args failed: %v", err).debug(1)
        return
    } else if args, err = parseOpts(t, &opts, args...); err != nil {
        diag.errorAt(pos, "parse opts failed: %v", err).debug(1)
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
    if target, okay := t.Get("@"); !okay || isNil(target) {
        diag.errorAt(pos, "target '@' is undefined").debug(1)
        return
    }  else if outFile, err = target.Strval(t); err != nil {
        diag.errorAt(pos, "strval '%v' failed: %v", target, err).debug(1)
        return
    }
    if opts.makePath {
        if err = os.MkdirAll(filepath.Dir(outFile), os.FileMode(0755)); err != nil {
            diag.errorAt(pos, "make path failed: %v", err).debug(1)
            return
        }
    }

    var ( fil *os.File; out *bufio.Writer )
    if fil, err = os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, opts.mode); err != nil {
        diag.errorAt(pos, "open file failed: %v", err).debug(1)
        return
    } else { out = bufio.NewWriter(fil) }
    defer func() {
        out.Flush()
        fil.Close()
    }()

    var depends []Value
    if value, okay := t.Get("^"); !okay || isNil(value) {
        // ...
    } else if depends, err = mergeresult2(expandall2(t, expandPlainValue, value)); err != nil {
        diag.errorAt(pos, "merge depends failed: %v", err).debug(1)
        return
    }

    var filterOpts builtinFilterOpts
    var sources []Value
    for _, depend := range depends {
        var a []Value
        switch d := depend.(type) {
        case *File:
            if a, err = filterValues(t, pats, filterOpts, false, d); err != nil {
                diag.errorAt(pos, "filter values failed: %v", err).debug(1)
            } else { sources = append(sources, a...) }
        case *Path:
            var s string
            if s, err = d.Strval(t); err != nil {
                diag.errorAt(pos, "strval '%v' failed: %v", d, err).debug(1)
                return
            }
            err = walkFiles(t, s, pats, func(file *File, err error) error {
                if err == nil { sources = append(sources, file) }
                return err
            })
        default:
            var s string
            if s, err = d.Strval(t); err != nil {
                diag.errorAt(pos, "strval '%v' failed: %v", d, err).debug(1)
                return
            }

            dir := filepath.Dir(s)
            name := filepath.Base(s)
            file := stat(t, name, "", dir)
            if file == nil {
                diag.errorAt(pos, "extract-configuration: `%s` file not found", name).debug(1)
                return
            } else if file.info.IsDir() {
                err = walkFiles(t, s, pats, func(file *File, err error) error {
                    if err == nil { sources = append(sources, file) }
                    return err
                })
            } else if a, err = filterValues(t, pats, filterOpts, false, d); err != nil {
                diag.errorAt(pos, "filter values failed: %v", err).debug(1)
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
            if s, err = v.Strval(t); err != nil {
                break ForSources
            }
        }
        if f, err = os.Open(s); err != nil {
            diag.prompt("%v: (configure) %v: %v\n", pos, source, err)
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
