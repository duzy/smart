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
        //"unicode/utf8"
        //"unicode"
        //"hash/crc64"
        "io/ioutil"
        "strings"
        "regexp"
        "bufio"
        "bytes"
        "sort"
        "fmt"
        "os"
        //"io"
)

type packagetype uint8
const (
        packageUnknown packagetype = iota
        packageSmart // smart package
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

var configurationOps = map[string] func(pos Position, t *traversal, def *Def, fields map[string]Value, args ...Value) (result Value, err error) {
        "answer":  configureAnswer,
        "bool":    configureBool,
        "dump":    configureDump,
        "option":  configureOption,
        "package": configurePackage,
}

func do_configuration() {
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
                        // ...
                } else if e = os.Remove(s); e == nil {
                        fmt.Fprintf(stderr, "configure: Remove %s\n", s)
                } else if true {
                        fmt.Fprintf(stderr, "configure: %s\n", e)
                }
        }

        var defs = make(map[string]*Def)
        for _, entry := range configuration.entries {
                if p := entry.OwnerProject(); p != project && p != nil {
                  defs = make(map[string]*Def) // reset defs for p
                        var f, e = openConfigurationFile(p)
                        if e != nil { diag.errorAt(entry.position, "%v", e); return } else
                        if f != nil {
                                if writer != nil {
                                        if e = writer.Flush(); e != nil {
                                                diag.errorAt(entry.position, "%v", e)
                                                return
                                        }
                                }
                                if file != nil {
                                        if e = file.Close(); e != nil {
                                                diag.errorAt(entry.position, "%v", e)
                                                return
                                        }
                                }
                        }

                        file, writer = f, bufio.NewWriter(f)
                        fmt.Fprintf(writer, "# %s (%s) configuration\n", p.spec, p.relPath)

                        fmt.Fprintf(stderr, "configure: Project %s …… (%s)\n", p.spec, p.relPath)
                        project = p
                }
                if _, e := entry.Execute(entry.position); e != nil {
                        diag.errorAt(entry.position, "%v", e)
                } else if s, e := entry.target.Strval(); e != nil {
                        diag.errorAt(entry.position, "%v", e)
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
                                vs := elementString(def, def.value, elemNoBrace)
                                fmt.Fprintf(writer, "%v = %v\n", def.name, vs)
                        }
                } else {
                        diag.errorAt(entry.position, "`%s` unconfigured", s)
                }
        }
        if err != nil { return }

        printLeavingDirectory()
        return
}

func openConfigurationFile(p *Project) (file *os.File, err error) {
        defer setclosure(setclosure(cloctx.unshift(p.scope)))
        if s, e := configurationFileName(p); e != nil {
                err = e
        } else if e = os.MkdirAll(filepath.Dir(s), os.FileMode(0755)); e != nil {
                err = e
        } else if f, e := os.OpenFile(s, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0600)); e == nil {
                file = f
        } else {
                err = e
        }
        return
}

func configurationFileName(p *Project) (s string, err error) {
        const name = "configuration.sm"
        var pos Position // TODO: find the position
        if f := p.matchTempFile(pos, name); f != nil {
                s, err = f.Strval()
        } else {
                fmt.Fprintf(stderr, "%v: no file for configuration.sm\n", p)
        }
        return
}

func configPrintf(pos Position, str string, args... interface{}) {
        fmt.Fprintf(stderr, str, args...)
}

func configMessageDone(pos Position, str string, args... interface{}) {
        if !strings.HasSuffix(str, "\n") { str += "\n" }
        configPrintf(pos, str, args...)
}

// -dump
func configureDump(pos Position, t *traversal, def *Def, fields map[string]Value, params ...Value) (result Value, err error) {
        result = def.value
        return
}

func configureBoolValue(pos Position, t *traversal, def *Def) (result bool, err error) {
        var value = def.Call(pos)
        if !isNil(value) {
                var res Value
                res, err = value.expand(expandAll)
                if err == nil && res != value { value = res }
        }

        for i, v := range merge(value) {
                var t bool
                if v == nil { continue }
                if t, err = v.True(); err != nil { return }
                if i == 0 {
                        result = t
                } else {
                        result = result && t
                }
                if !result { break }
        }
        return
}

// -bool
// -bool('message...')
func configureBool(pos Position, t *traversal, def *Def, fields map[string]Value, params ...Value) (result Value, err error) {
        var val bool
        if val, err = configureBoolValue(pos, t, def); err == nil {
                result = MakeBoolean(pos, val)
        }
        return
}

// -answer
// -answer('message...')
func configureAnswer(pos Position, t *traversal, def *Def, fields map[string]Value, params ...Value) (result Value, err error) {
        var val bool
        if val, err = configureBoolValue(pos, t, def); err == nil {
                result = MakeAnswer(pos, val)
        }
        return
}

// -option
// -option('message...')
func configureOption(pos Position, t *traversal, def *Def, fields map[string]Value, args ...Value) (result Value, err error) {
        if result = def.Call(pos); isNil(result) { result = &answer{trivial{pos},false} }
        if result != nil {
                var res Value
                if res, err = result.expand(expandAll); err == nil && res != result {
                        result = res
                } else if err != nil {
                        diag.errorAt(pos, "%v", err)
                }
        }
        return
}

// -package finds system package in a way similar to cmake.find_package
func configurePackage(pos Position, t *traversal, def *Def, fields map[string]Value, args ...Value) (result Value, err error) {
        var names []string
        var optType packagetype = packageSmart
        for _, arg := range args {
                switch a := arg.(type) {
                case *Pair:
                        var key, val string
                        if key, err = a.Key.Strval();   err != nil { return }
                        if val, err = a.Value.Strval(); err != nil { return }
                        switch key {
                        case "type":
                                switch val {
                                case "", "smart": optType = packageSmart
                                case "pkgconfig": optType = packageConfig
                                default: optType = packageUnknown
                                        err = scanner.Errorf(token.Position(pos), "package: `%v` unknown type\n", val)
                                        return
                                }
                        default:
                                fmt.Fprintf(stderr, "%v: package: `%v` unknown option\n", key)
                        }
                default:
                        var name string
                        if name, err = a.Strval(); err != nil { return }
                        names = append(names, name)
                }
        }
        for _, name := range names {
                var info, cached = configuration.packages[name]
                if !cached {
                        switch optType {
                        // case packageSmart:
                        //         if info, err = loadPackageSmartInfo(pos, name); err != nil { return }
                        // case packageConfig:
                        //         if info, err = loadPackageConfigInfo(pos, name); err != nil { return }
                        case packageUnknown:
                                fmt.Fprintf(stderr, "%v: package `%v`: unknown type\n", name)
                        }
                        if info != nil {
                                configuration.packages[name] = info
                                result = &answer{trivial{pos},true}
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

func configureExec(pos Position, t *traversal, s string, target Value, paramsOrig ...Value) (configured bool, result Value, err error) {
        if optionTraceConfig { defer un(trace(t_config, fmt.Sprintf("configureExec(%s %v)", s, t.entry.target))) }

        var projectConfigure = t.program.project.configure
        if  projectConfigure == nil {
                diag.errorAt(pos, "no .configure provided")
                return
        }

        var entry *RuleEntry
        if entry, err = projectConfigure.resolveEntry("-"+s); err != nil {
                diag.errorAt(pos, "resolve %v: %v", s, err)
                return
        } else if entry == nil {
                diag.errorAt(pos, "unknown configuration `%v` (no such entry)", s)
                return
        }

        if false { defer setclosure(setclosure(cloctx.unshift(t.program.scope))) }
        if false { fmt.Fprintf(stderr, "%v: configureExec(%v %v): %v, %v\n", pos, entry, t.entry, paramsOrig, cloctx) }

        var params []Value
        var prog = entry.programs[0]
        for _, par := range prog.params {
                switch par.name {
                case "TARGET": params = append(params, MakePair(pos, MakeBareword(pos, "TARGET"), target))
                case "LANG":   params = append(params, MakePair(pos, MakeBareword(pos, "LANG"), MakeString(pos, t.program.language)))
                case "VALUE":  params = append(params, MakePair(pos, MakeBareword(pos, "VALUE"), t.def.buffer))
                }
                for _, a := range paramsOrig {
                        if ap, ok := a.(*Pair); ok {
                                s, e := ap.Key.Strval()
                                if e != nil { diag.errorOf(ap.Key, "%v", e); return }
                                if par.name == s {
                                       params = append(params, a)
                                } else if par.name == strings.ToUpper(s) {
                                       params = append(params, MakePair(pos, MakeBareword(pos, par.name), ap.Value))
                                } else if false {
                                        diag.warnOf(a, "unknown parameter: %v", a)
                                        return
                                }
                        } else {
                                diag.errorOf(a, "unsupported parameter: %v (%T)", a, a)
                                return
                        }
                }
        }

        defer func(v bool) { t.isConfigureExecution = false } (t.isConfigureExecution)
        t.isConfigureExecution = true

        var breakers []*breaker
        if result, breakers = prog.execute(t, entry, params); len(breakers) > 0 {
                for i, brk := range breakers {
                        if brk.what == breakErro {
                                fmt.Fprintf(stderr, "%s: %d: %v\n", pos, i, brk.error)
                        } else {
                                fmt.Fprintf(stderr, "%s: %d: %v\n", pos, i, brk.what)
                        }
                }
        } else if false {
                fmt.Fprintf(stderr, "%s: %s: %v (%T)\n", pos, entry, result, result)
        }

        configured = true
        return
}

func configureDo(pos Position, t *traversal, target Value, def, name Value, args []Value) (configured bool, result Value, err error) {
        if optionTraceConfig { defer un(trace(t_config, "configureDo")) }

        var pipe = t.def.buffer
        var strName string
        if  strName, err = name.Strval(); err != nil {
                diag.errorAt(pos, "%v: %v", name, err)
                return
        } else if strName == "" {
                diag.errorAt(pos, "`%v` empty configuration (%T)", name, name)
                return
        }

        var info Value
        var params []Value
ForArgs:
        for _, arg := range args {
                var list, ok = arg.(*List)
                if ok && list != nil && len(list.Elems) == 1 {
                        switch t := list.Elems[0].(type) {
                        case *Pair:
                                params = append(params, t)
                                continue ForArgs
                        case *String, *Compound:
                                var ap = t.Position()
                                arg = MakePair(ap, MakeBareword(ap, "INFO"), t)
                                params = append(params, arg)
                                info = t
                                continue ForArgs
                        }
                }
                diag.errorOf(arg, "unsupported parameter: %v (%T)", arg, arg)
                return
        }

        var defsChanged = make(map[*Def]Value)
        defer func() {
                for def, val := range defsChanged { def.value = val }
                if err != nil {
                        if e, ok := err.(*scanner.Error); ok {
                                configMessageDone(pos, "… (%v).", e.Brief())
                        } else {
                                configMessageDone(pos, "… (%v).", err)
                        }
                        return
                } else if result == nil {
                        configMessageDone(pos, "… <nil>")
                } else {
                        var s string
                        if  s, _ = result.Strval(); s == "" { s = "?" }
                        configMessageDone(pos, "… %v", s)
                }
         } ()

        if isNil(info) {
                configPrintf(pos, "configure: %v %v …", target, args)
        } else if s, e := info.Strval(); e == nil {
                configPrintf(pos, "configure: %s …", s)
        } else {
                diag.errorAt(pos, "%v", e)
        }

        // Process configurations like:
        //   -bool
        //   -option
        //   -package
        //   ...
        if config, ok := configurationOps[strName]; ok {
                params = append(params, MakePair(pos, MakeBareword(pos, "TARGET"), target))
                if result, err = config(pos, t, pipe, nil, params...); err != nil {
                        diag.errorAt(pos, "config: %v", err)
                } else {
                        configured = true
                        if optionTraceConfig {
                                t_config.tracef("configured: %v, result = %v (%s)", configured, result, typeof(result))
                        }
                        return
                }
        }

        if configured, result, err = configureExec(name.Position(), t, strName, target, params...); err != nil {
                diag.errorAt(pos, "%v", err)
        } else if configured {
                if optionTraceConfig {
                        t_config.tracef("configured: %v, result = %v (%s)", configured, result, typeof(result))
                }
                return
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
func modifierConfigure(pos Position, t *traversal, args ...Value) (result Value, err error) {
        if optionTraceConfig { defer un(trace(t_config, fmt.Sprintf("modifierConfigure(%v) (reconfig=%v)", t.entry.target, optionReconfig))) }
        if t.program.project.configure == nil {
                diag.errorAt(pos, "%v: .configure not provided", t.program.project)
                return
        }

        var optAccumulate bool
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return } else
        if args, err = tryParseFlags(args, []string{
                "a,accumulate",
        }, func(ru rune, v Value) {
                switch ru {
                case 'a': if optAccumulate, err = trueVal(v, true); err != nil { diag.errorOf(v, "%v", err); return }
                }
        }); err != nil { diag.errorAt(pos, "%v", err); return }

        var target = t.def.target.value
        if isNil(target) || isNone(target) {
                diag.errorAt(pos, "target is nil for entry '%s'", t.entry.target)
                return
        }

        var name string
        if name, err = target.Strval(); err != nil { diag.errorOf(target, "%v", err); return }

        var def, alt = t.program.project.scope.define(t.program.project, name, nil)
        if alt != nil { def, _ = alt.(*Def) }
        if def != nil { result = def } else {
                diag.errorAt(pos, "cannot define configuration `%s`", name)
                return
        }
        if optionTraceConfig {
                t_config.tracef("%s: %v (%T)", def.name, def.value, def.value)
                defer func() { t_config.tracef("%s: %v (%T)", def.name, def.value, def.value) } ()
        }
        if !isNil(def.value) { // Check if it's already configured?
                if !optionReconfig { return } // return if not reconfigure
                if done, found := configuration.done[def]; done && found { return }
        }

        var value Value
        if len(args) == 0 { // Empty configuration: (configure)
                if value = t.def.buffer.Call(pos); value == nil {
                        diag.errorAt(pos, "`%v` not configured (%v)", target, value)
                        return
                } else if value == def || value.refs(def) {
                        return
                }
                switch v := value.(type) {
                default: err = def.set(DefConfig, value)
                case *ExecResult:
                        var s string
                        if v.wg.Wait(); v.Status == 0 && v.Stdout.Buf != nil {
                                s = v.Stdout.Buf.String()
                        } else if v.Stderr.Buf != nil {
                                s = v.Stderr.Buf.String()
                        }
                        err = def.set(DefConfig, MakeString(pos, s))
                }
                if err != nil { diag.errorAt(pos, "%v", err) }
                return
        } else if err = def.set(DefConfig, nil); err != nil {
                diag.errorOf(def, "%v", err)
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
                                diag.errorOf(a, "`%v` is unsupported value (%T)\n", arg.value, arg.value)
                                return
                        } else {
                                name, para = flag.name, arg.args
                        }
                case *Flag:
                        if isNil(arg.name) || isNone(arg.name) {
                                diag.errorOf(a, "`%v` is unsupported flag (%T)\n", arg.name, arg.name)
                                return
                        } else {
                                name = arg.name
                        }
                default:
                        diag.errorOf(a, "`%v` is unsupported (%T)\n", a, a)
                        return
                }
                if name == nil {
                        diag.errorOf(a, "unknown configure `%v` (%T)\n", a, a)
                        return
                }

                configured, value, err = configureDo(pos, t, target, def, name, para)
                if err != nil {
                        diag.errorAt(pos, "%v", err); return
                } else if !configured {
                        diag.warnAt(pos, "%s not configured", name)
                        return
                } else if value == nil {
                        value = MakeNil(a.Position())
                } else if value, err = value.expand(expandAll); err != nil {
                        diag.errorOf(a, "%v", err)
                } else if value == def || value.refs(def) {
                        // Value is the Def, does nothing!
                } else if optAccumulate {
                        if err = def.append(value);         err != nil { diag.errorOf(a, "%v", err) }
                } else {
                        if err = def.set(DefConfig, value); err != nil { diag.errorOf(a, "%v", err) }
                }

                if err == nil { configuration.done[def] = true }
                if optionTraceConfig {
                        t_config.tracef("configured: %v (%s) (%v)", value, typeof(value), def.origin)
                }
        }
        if !configured && err == nil {
                diag.errorAt(pos, "`%v` not configured", target)
        }
        return
}

type filewalkFunc func(file *File, err error) error

func walkFileInfos(root string, pats []Value, fn filepath.WalkFunc) (err error) {
        return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
                if err != nil { return err }
        ForPats:
                for _, p := range pats {
                        switch pat := p.(type) {
                        case Pattern: //*PercPattern, *GlobPattern, *RegexpPattern, *Path
                                var ( s string ; ss []string )
                                if s, ss, err = pat.match(path); err != nil {
                                        break ForPats
                                }
                                if s == "" || ss == nil {
                                        var s = filepath.Base(path)
                                        if s, ss, err = pat.match(s); err != nil {
                                                break ForPats
                                        }
                                }
                                if s != "" && ss != nil {
                                        if err = fn(path, info, err); err != nil {
                                                break ForPats
                                        }
                                }
                        default:
                                var s string
                                if s, err = p.Strval(); err != nil { break ForPats }
                                if path == s || filepath.Base(path) == s {
                                        if err = fn(path, info, err); err != nil { break ForPats }
                                }
                        }
                }
                return err
        })
}

func walkFiles(pos Position, root string, pats []Value, fn filewalkFunc) error {
        return walkFileInfos(root, pats, func(path string, info os.FileInfo, err error) error {
                if err != nil { return err }
                var rel string
                if rel, err = filepath.Rel(root, path); err != nil {
                        return err
                }
                file := stat(pos, rel, "", root, info)
                if enable_assertions {
                        assert(file != nil, "`%s` file is nil", rel)
                }
                return fn(file, err)
        })
}

var configuredFiles = make(map[string]*Scope,8)

// configure-file modifier (see also builtinConfigureFile), example usage:
// 
//     config.h: config.h.in [(configure-file)]
//     
func modifierConfigureFile(pos Position, t *traversal, args ...Value) (result Value, err error) {
        var (
                optPath = false
                optDebug = false
                optVerbose = false
                optReconfig = false
                optMode = os.FileMode(0600)
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "m,mode",
                "p,path",
                "r,reconfig",
                "v,verbose",
                "d,debug",
        }, func(ru rune, v Value) {
                switch ru {
                case 'd': if optDebug, err = trueVal(v, true); err != nil { return }
                case 'p': if optPath, err = trueVal(v, true); err != nil { return }
                case 'v': if optVerbose, err = trueVal(v, true); err != nil { return }
                case 'r': if optReconfig, err = trueVal(v, true); err != nil { return }
                case 'm': if v != nil {
                        var num int64
                        if num, err = v.Integer(); err != nil { return } else {
                                optMode = os.FileMode(num & 0777)
                        }
                }}
        }); err != nil { diag.errorAt(pos, "%v", err); return }

        var project *Project
        var filename string
        var file *File
        if file, _ = t.def.target.value.(*File); file == nil {
                var s string
                if s, err = t.def.target.value.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }

                var okay bool
                okay, err = t.forClosureProject(func(p *Project) (ok bool, err error) {
                        if file = p.matchFile(s); file != nil { project, ok = p, true }
                        if optDebug && file != nil { fmt.Fprintf(stderr, "%s: %v: file %v\n", pos, p, file) }
                        return
                })
                if err != nil { return } else if !okay { diag.errorAt(pos, "'%s' is not a file", s); return }
        }
        if file == nil { diag.errorAt(pos, "no file target"); return }
        if filename, err = file.Strval(); err != nil { diag.errorAt(pos, "%v", err); return } else
        if filename == "" { diag.errorAt(pos, "`%v` has empty filename", file); return } else
        if!filepath.IsAbs(filename) {
                // FIXES: match file map to have the full filename.
                t.forClosureProject(func(p *Project) (ok bool, err error) {
                        if f := p.matchFile(filename); f != nil {
                                var s string
                                ok, file = true, f
                                s, err = f.Strval()
                                t.def.target.value = file // reset target file
                                project, filename = p, s // using full filename instead
                                if optDebug { fmt.Fprintf(stderr, "%s: configure-file: %v: %s->%s\n", pos, p, f, s) }
                        }
                        return
                })
                if err != nil { diag.errorAt(pos, "%v", err); return }
        }
        if file.info == nil { if f := stat(pos, filename, "", ""); f != nil { file.info = f.info }}
        if project == nil { project = t.project }
        if optDebug && file != nil {
                var s, _ = file.Strval()
                var target = t.def.target.value
                fmt.Fprintf(stderr, "%s: configure-file: %v: %v (%s) (%v, %v) (%v)\n", pos,
                        project, target, s, t.project, t.closure.comment, cloctx)
        }

        // Check previously configured files, we only configure once unless
        // optReconfig is true.
        var closure *Scope
        if configuredFiles != nil {
                var okay bool
                closure, okay = configuredFiles[filename]
                if okay && closure != nil && !optReconfig { return }
        }
        if closure == nil { closure = t.closure }
        defer func(s string, c *Scope) {
                if err == nil { configuredFiles[s] = c } else {
                        diag.errorAt(pos, "%v", err)
                }
        } (filename, closure)

        var data bytes.Buffer
        for _, arg := range append(args, t.def.buffer.value) {
                var str string
                if str, err = arg.Strval(); err != nil { return }
                if str == "" { continue }
                if err = configure(pos, &data, closure.project, str); err != nil { return }
        }
        if data.Len() == 0 { diag.errorAt(pos, "no input data"); return }

        if optVerbose { fmt.Fprintf(stderr, "smart: Checking %v …", file) }
        if file.info != nil {
                if same, e := crc64CheckFileModeContent(filename, data.Bytes(), optMode); e != nil {
                        if optVerbose { fmt.Fprintf(stderr, "… (error: %s)\n", e) }
                        diag.errorAt(pos, "%v", e); return
                } else if same {
                        var tt = file.info.ModTime()
                        for _, d := range merge(t.targets.value) {
                                if f, ok := d.(*File); !ok { continue } else
                                if dt := f.info.ModTime(); dt.After(tt) { tt = dt }
                        }
                        if tt.After(file.info.ModTime()) { err = touch(file, 0, false, tt) }
                        if optVerbose { fmt.Fprintf(stderr, "… Good\n") }
                        result = file
                        return
                }
        } else if dir := filepath.Dir(filename); optPath && dir != "." && dir != PathSep {
                if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                        diag.errorAt(pos, "%v", err)
                        return
                }
        }
        if optVerbose { fmt.Fprintf(stderr, "… Outdated (%s)\n", filename) }

        var status string
        if optVerbose {
                printEnteringDirectory()
                fmt.Fprintf(stderr, "smart: Updating %v …", file)
                defer func() {
                        if err != nil { status = "error!" } else
                        if status == "" { status = "done." }
                        fmt.Fprintf(stderr, "… %s\n", status)
                } ()
        }

        if err = ioutil.WriteFile(filename, data.Bytes(), optMode); err != nil {
                diag.errorAt(pos, "%v", err)
                return
        }
        if file.info != nil { result = file } else {
                if file.info, err = os.Stat(filename); err == nil {
                        context.globe.stamp(filename, file.info.ModTime())
                        result = file
                }
        }
        status = fmt.Sprintf("Updated (%s, %d bytes)", filename, data.Len())
        return
}

// extract-configuration extracts configuration from C/C++ files, example usage:
//
//      config.h.in:[(extract-configuration)]: $(wildcard *.cpp)
//
func modifierExtractConfiguration(pos Position, pc *traversal, args ...Value) (result Value, err error) {
        var pats []Value
        var rxs []*regexp.Regexp
        var optTarget string
        var optPath bool
        var optPerm = os.FileMode(0640) // sys default 0666
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                "p,path",
                "r,rx",
                "r,regex",
                "g,grep",
                "m,mode",
                "t,target",
        }, func(ru rune, v Value) {
                switch ru {
                case 'p': optPath = true
                case 'r':
                        var (s string; x *regexp.Regexp)
                        if s, err = v.Strval(); err != nil {
                                return
                        } else if x, err = regexp.Compile(s); err != nil {
                                return
                        } else {
                                rxs = append(rxs, x)
                        }
                case 't':
                        if optTarget, err = v.Strval(); err != nil {
                                return
                        }
                case 'm': if v != nil {
                        var num int64
                        if num, err = v.Integer(); err != nil { return } else {
                                optPerm = os.FileMode(num & 0777)
                        }
                }}
        }); err != nil { return }
        for _, arg := range args {
                switch a := arg.(type) {
                default:
                        pats = append(pats, a)
                case *Group:
                        pats = append(pats, a.Elems...)
                }
        }
        if len(pats) == 0 {
                err = fmt.Errorf("extract-configuration: missing file names (patterns)")
                return
        }
        if len(rxs) == 0 {
                err = fmt.Errorf("extract-configuration: missing -rx=... flags")
                return
        }
        if optTarget == "" {
                optTarget = "configuration"
        }

        var outFile string
        if outFile, err = pc.def.target.value.Strval(); err != nil { return }
        if optPath {
                if err = os.MkdirAll(filepath.Dir(outFile), os.FileMode(0755)); err != nil {
                        return
                }
        }

        var ( fil *os.File; out *bufio.Writer )
        fil, err = os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, optPerm)
        if err != nil { return } else { out = bufio.NewWriter(fil) }
        defer func() {
                out.Flush()
                fil.Close()
        }()

        var depends []Value
        if depends, err = mergeresult(ExpandAll(pc.def.depends.value)); err != nil { return }

        var sources []Value
        for _, depend := range depends {
                var a []Value
                switch d := depend.(type) {
                case *File:
                        if a, err = filterValues(pats, false, d); err == nil {
                                sources = append(sources, a...)
                        }
                case *Path:
                        var s string
                        if s, err = d.Strval(); err == nil {
                                err = walkFiles(pos, s, pats, func(file *File, err error) error {
                                        if err == nil {
                                                sources = append(sources, file)
                                        }
                                        return err
                                })
                        }
                default:
                        var s string
                        if s, err = d.Strval(); err != nil {
                                return
                        }

                        dir := filepath.Dir(s)
                        name := filepath.Base(s)
                        file := stat(pos, name, "", dir)
                        if file == nil {
                                err = scanner.Errorf(token.Position(pos), "`%s` file not found (configure)", name)
                                return
                        } else if file.info.IsDir() {
                                err = walkFiles(pos, s, pats, func(file *File, err error) error {
                                        if err == nil { sources = append(sources, file) }
                                        return err
                                })
                        } else if a, err = filterValues(pats, false, d); err == nil {
                                sources = append(sources, a...)
                        }
                }
        }

        var exprs = make(map[string]int)
ForSources:
        for _, source := range sources {
                var (s string; f *os.File)
                switch t := source.(type) {
                case *File: s = t.fullname()
                default:
                        if s, err = t.Strval(); err != nil {
                                break ForSources
                        }
                }
                if f, err = os.Open(s); err != nil {
                        fmt.Fprintf(stderr, "%v: (configure) %v: %v\n", pos, source, err)
                        continue ForSources
                }
                scanner := bufio.NewScanner(f)
                scanner.Split(bufio.ScanLines)
                for scanner.Scan() {
                        s := scanner.Text()
                        ForOpts: for _, x := range rxs {
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
        fmt.Fprintf(out, "%s:[(configure -check)]:\\\n", optTarget)
        for _, k := range keys { fmt.Fprintf(out, "  %s \\\n", k) }
        fmt.Fprintf(out, "\n")
        return
}
