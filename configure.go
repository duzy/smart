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
        "unicode/utf8"
        "unicode"
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
        scope *Scope
        project *Project
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

var confinitSource, confinitFilename = configurationInitFile()
func init_configuration(paths searchlist) (err error) {
        if optionTraceLaunch { defer un(trace(t_launch, "init_configuration")) }

        var pos Position
        pos.Filename = confinitFilename
        configuration.scope = NewScope(pos, context.globe.scope, nil, "configuration")
        configuration.paths = paths

        var l = &loader{
                Context: &context,
                scope:    configuration.scope,
                fset:     configuration.fset,
                paths:    configuration.paths,
                loaded:   make(map[string]*Project),
        }

        // Restore cloctx (remove "~.smart" from cloctx)
        defer setclosure(cloctx) // FIXME: "~.smart" should not be in cloctx

        if err = l.loadFile(confinitFilename, confinitSource); err != nil {
                return
        } else if project, ok := l.loaded[confinitFilename]; ok {
                configuration.project = project
        } else {
                err = errorf(pos, "configuration: `%v` not loaded\n", confinitFilename)
        }

        if configuration.project == nil {
                panic("configuration.project is nil")
        }
        return
}

func do_configuration() (err error) {
        var (
                project *Project
                file *os.File
                writer *bufio.Writer
        )
        defer func() {
                if writer != nil { if err := writer.Flush(); err != nil {} }
                if file != nil { if err := file.Close(); err != nil {} }
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
                        if e != nil { err = wrap(entry.position, e, err); return } else
                        if f != nil {
                                if writer != nil {
                                        if e = writer.Flush(); e != nil {
                                                err = wrap(entry.position, e, err)
                                                return
                                        }
                                }
                                if file != nil {
                                        if e = file.Close(); e != nil {
                                                err = wrap(entry.position, e, err)
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
                        err = wrap(entry.position, e, err)
                } else if s, e := entry.target.Strval(); e != nil {
                        err = wrap(entry.position, e, err)
                } else if def := project.scope.FindDef(s); def != nil {
                        if d, ok := defs[s]; ok && d != nil {
                                /*if d.value.cmp(def.value) != cmpEqual {
                                        err = errorf(entry.position, "'%s' already configured: %v", d.name, d.value)
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
                        e := fmt.Errorf("`%s` unconfigured", s)
                        err = wrap(entry.position, e, err)
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
        var debug bool
        if o := configuration.scope.Lookup("DEBUG"); o != nil {
                debug, _ = o.True()
        }
        if debug { str = fmt.Sprintf("%v:info: %s", pos, str) }
        fmt.Fprintf(stderr, str, args...)
}

func configMessageDone(pos Position, str string, args... interface{}) {
        if !strings.HasSuffix(str, "\n") { str += "\n" }
        configPrintf(pos, str, args...)
}

func configPrintMessageHead(pos Position, fields map[string]Value, args ...Value) (str string, err error) {
        var s string
        if name, ok := fields["name"]; ok {
                str = "Checking"
                if s, err = name.Strval(); err != nil { return }
                if str += " " + s; len(args) > 1 { str += "s" }
        }
        for i, a := range args {
                if s, err = a.Strval(); err != nil { return }
                str += " "
                if i == 0 { str += "(" }
                str += s
                if i == len(args)-1 { str += ")" }
        }
        str += " …"
        return
}

var configMessageHeadPrinters = map[string] func(Position, map[string]Value, ...Value) (string, error) {
        "if":       configPrintMessageHead,
        "option":   configPrintMessageHead,
        "compiles": configPrintMessageHead,
        "library": func(pos Position, fields map[string]Value, args ...Value) (str string, err error) {
                // Examples:
                //   -library(foo,func,include='<foo.h>')
                //   -library(foo,func)
                //   -library(foobar)
                // TODO: using this form instead:
                //   -library(foo [,function=func] [,include='<foo.h>']...)
                var libs = args[1:]
                var s string
                s, err = libs[0].Strval()
                if err != nil { return }
                str += "Checking library " + s
                if len(libs) > 1 {
                        s, err = libs[1].Strval()
                        if err != nil { return }
                        str += " (" + s
                        for _, lib := range libs[2:] {
                                if p, ok := lib.(*Pair); ok {
                                        s, err = p.Key.Strval()
                                        if err != nil { return }
                                        if s == "include" {
                                                s, err = p.Value.Strval()
                                                if err != nil { return }
                                                str += ", #include " + s
                                        }
                                }
                        }
                        str += ")"
                }
                str += " …"
                return
        },
        "include": func(pos Position, fields map[string]Value, args ...Value) (str string, err error) {
                var s string
                var incs = args[1:]
                if len(incs) > 0 {
                        s, err = incs[0].Strval()
                        if err != nil { return }
                } else {
                        s, err = args[0].Strval()
                        if err != nil { return }
                }
                str += "Checking include " + s
                str += " …"
                return
        },
        "symbol": func(pos Position, fields map[string]Value, args ...Value) (str string, err error) {
                var syms = args[1:]
                var s string
                s, err = syms[0].Strval()
                if err != nil { return }
                str += "Checking symbol " + s
                if len(syms) > 1 {
                        s, err = syms[1].Strval()
                        if err != nil { return }
                        str += " (" + s
                        for _, lib := range syms[2:] {
                                if p, ok := lib.(*Pair); ok {
                                        s, err = p.Key.Strval()
                                        if err != nil { return }
                                        switch s {
                                        case "include":
                                                s, err = p.Value.Strval()
                                                if err != nil { return }
                                                str += ", #include " + s
                                        case "library":
                                                s, err = p.Value.Strval()
                                                if err != nil { return }
                                                str += ", -l" + s
                                        }
                                }
                        }
                        str += ")"
                }
                str += " …"
                return
        },
        "function": func(pos Position, fields map[string]Value, args ...Value) (str string, err error) {
                var funcs = args[1:]
                var s string
                s, err = funcs[0].Strval()
                if err != nil { return }
                str += "Checking function " + s
                if len(funcs) > 1 {
                        s, err = funcs[1].Strval()
                        if err != nil { return }
                        str += " (" + s
                        for _, lib := range funcs[2:] {
                                if p, ok := lib.(*Pair); ok {
                                        s, err = p.Key.Strval()
                                        if err != nil { return }
                                        switch s {
                                        case "include":
                                                s, err = p.Value.Strval()
                                                if err != nil { return }
                                                str += ", #include " + s
                                        case "library":
                                                s, err = p.Value.Strval()
                                                if err != nil { return }
                                                str += ", -l" + s
                                        }
                                }
                        }
                        str += ")"
                }
                str += " …"
                return
        },
        "struct-member": func(pos Position, fields map[string]Value, args ...Value) (str string, err error) {
                // Examples:
                //   -struct-member('struct stat',st_mtimespec.tv_nsec,include=('<sys/types.h>','<sys/stat.h>'))
                var mems = args[1:]
                var s string
                if len(mems) > 1 {
                        s, err = mems[0].Strval()
                        if err != nil { return }
                        str += "Checking (" + s + ") " // struct member

                        s, err = mems[1].Strval()
                        if err != nil { return }
                        str += s
                }
                if len(mems) > 2 {
                        str += " ("
                        for _, val := range mems[3:] {
                                if p, ok := val.(*Pair); ok {
                                        s, err = p.Key.Strval()
                                        if err != nil { return }
                                        switch s {
                                        case "include":
                                                s, err = p.Value.Strval()
                                                if err != nil { return }
                                                str += ", #include " + s
                                        case "library":
                                                s, err = p.Value.Strval()
                                                if err != nil { return }
                                                str += ", -l" + s
                                        }
                                }
                        }
                        str += ")"
                }
                str += " …"
                return
        },
        "type": func(pos Position, fields map[string]Value, args ...Value) (str string, err error) {
                var typs = args[1:]
                var s string
                s, err = typs[0].Strval()
                if err != nil { return }
                str += "Checking type " + s
                if len(typs) > 1 {
                        s, err = typs[1].Strval()
                        if err != nil { return }
                        str += " (" + s
                        for _, lib := range typs[2:] {
                                if p, ok := lib.(*Pair); ok {
                                        s, err = p.Key.Strval()
                                        if err != nil { return }
                                        switch s {
                                        case "include":
                                                s, err = p.Value.Strval()
                                                if err != nil { return }
                                                str += ", #include " + s
                                        }
                                }
                        }
                        str += ")"
                }
                str += " …"
                return
        },
        "sizeof": func(pos Position, fields map[string]Value, args ...Value) (str string, err error) {
                var syms = args[1:]
                var s string
                str += "Checking size of "
                for _, sym := range syms {
                        s, err = sym.Strval()
                        if err != nil { return }
                        str += s
                }
                str += " …"
                return
        },
        "package": func(pos Position, fields map[string]Value, args ...Value) (str string, err error) {
                if len(args) > 2 {
                        return configPrintMessageHead(pos, fields, args[2])
                } else {
                        return configPrintMessageHead(pos, fields, args[0])
                }
        },
}

func configMessageHead(pos Position, op string, fields map[string]Value, params ...Value) (err error) {
        var ( s string; str = "configure: " )
        if v, ok := fields["info"]; ok {
                if l, ok := v.(*List); ok && len(l.Elems) > 0 { v = l.Elems[0] }
                if s, err = v.Strval(); err == nil && len(s) > 0 {
                        r, size := utf8.DecodeRuneInString(s)
                        if size > 0 && unicode.IsUpper(r) {
                                str += s
                        } else {
                                str += "Checking " + s
                        }
                        str += " …"
                        configPrintf(pos, str)
                }
        } else {
                f, ok := configMessageHeadPrinters[op]
                if!ok { f = configPrintMessageHead }
                if s, err = f(pos, fields, params...); err != nil { return }
                str += s
                configPrintf(pos, str)
        }
        return
}

// -dump
func configureDump(pos Position, t *traversal, def *Def, fields map[string]Value, params ...Value) (result Value, err error) {
        result = def.value
        return
}

func configureBoolValue(pos Position, t *traversal, def *Def) (result bool, err error) {
        var value Value
        if value, err = def.Call(pos); err != nil { return } else {
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
        if len(params) > 1 { fmt.Fprintf(stderr, "%s: useless args %v for -bool\n", pos, params) }
        var val bool
        if val, err = configureBoolValue(pos, t, def); err == nil {
                result = &boolean{trivial{pos},val}
        }
        return
}

// -answer
// -answer('message...')
func configureAnswer(pos Position, t *traversal, def *Def, fields map[string]Value, params ...Value) (result Value, err error) {
        if len(params) > 1 { fmt.Fprintf(stderr, "%s: useless args %v for -answer\n", pos, params) }
        var val bool
        if val, err = configureBoolValue(pos, t, def); err == nil {
                result = &answer{trivial{pos},val}
        }
        return
}

// -option
// -option('message...')
func configureOption(pos Position, t *traversal, def *Def, fields map[string]Value, args ...Value) (result Value, err error) {
        if result, err = def.Call(pos); err == nil {
                if result == nil { result = &answer{trivial{pos},false} }
        } else if result != nil {
                var res Value
                if res, err = result.expand(expandAll); err == nil && res != result {
                        result = res
                }
        }
        return
}

func loadPackageSmartInfo(pos Position, name string) (info *packageinfo, err error) {
        var file *File
        for _, path := range configuration.paths {
                file = stat(pos, name+".smart", "", path)
                if file != nil { break }
        }
        if file == nil { return }

        var l = &loader{
                Context: &context,
                fset:     configuration.fset,
                scope:    configuration.scope,
                paths:    configuration.paths,
                loaded:   make(map[string]*Project),
        }

        var filename = file.fullname()
        if err = l.loadFile(filename, nil); err != nil { return }
        if project, _ := l.loaded[filename]; project == nil {
                err = scanner.Errorf(token.Position(pos), "unloaded package %v (%v)\n", name, file)
        } else if project.name != name {
                err = scanner.Errorf(token.Position(pos), "%v: conflicted package name %v (!= %v)\n", file, project.name, name)
        } else {
                info = &packageinfo{ project, packageSmart }
        }
        return
}

func loadPackageConfigInfo(pos Position, name string) (info *packageinfo, err error) {
        return
}

// -library finds system library in a way similar to cmake.find_library
//func configureLibrary(pos Position, prog *Program, args ...Value) (result Value, err error) {
//        return
//}

// -package finds system package in a way similar to cmake.find_package
func configurePackage(pos Position, t *traversal, def *Def, fields map[string]Value, args ...Value) (result Value, err error) {
        var names []string
        var optType packagetype = packageSmart
        for _, arg := range args {
                switch a := arg.(type) {
                case *Pair:
                        var key, val string
                        if key, err = a.Key.Strval(); err != nil { return }
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
                        //case packageSmart:
                        //        if info, err = loadPackageSmartInfo(pos, name); err != nil { return }
                        case packageConfig:
                                if info, err = loadPackageConfigInfo(pos, name); err != nil { return }
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

func configureExec(pos Position, t *traversal, s string, params ...Value) (configured bool, result Value, err error) {
        if optionTraceConfig { defer un(trace(t_config, fmt.Sprintf("configureExec(%s %v)", s, t.entry.target))) }

        var entry *RuleEntry
        if entry, err = configuration.project.resolveEntry("-"+s); err != nil {
                err = errorf(pos, "resolve %v: %v", s, err)
                return
        } else if entry == nil {
                err = errorf(pos, "unknown configuration `%v` (no such entry)", s)
                return
        }

        if false { defer setclosure(setclosure(cloctx.unshift(t.program.scope))) }
        if false { fmt.Fprintf(stderr, "%v: configureExec(%v %v): %v, %v\n", pos, entry, t.entry, params, cloctx) }
        if result, err = entry.programs[0].execute(t, entry, params); err == nil { err = t.wait(pos) }
        if false { fmt.Fprintf(stderr, "%v: configureExec(%v %v): %v (%T), %v\n", pos, entry, t.entry, result, result, err) }

        var status int
        if err == nil { configured = true } else {
                var n int
                if n, status = scanExitStatus(err); n == 1 {
                        configured = true
                }
        }

        if optionTraceConfig {
                t_config.tracef("result=%v, status=%d, configured=%v, error=%v", result, status, configured, err!=nil)
                if err != nil { fmt.Fprintf(stderr, "%v\n", err) }
        }

        if status == 0 { err = nil } else
        if err != nil { err = wrap(pos, err) }
        return
}

func configureDo(pos Position, t *traversal, target Value, def, pipe *Def, name Value, args []Value) (configured bool, result Value, err error) {
        if optionTraceConfig { defer un(trace(t_config, "configureDo")) }

        var strName string
        if strName, err = name.Strval(); err != nil { return }
        if strName == "" {
                err = errorf(pos, "`%v` empty configuration (%T)", name, name)
                return
        }

        var none = &None{trivial{pos}}
        var params = []Value{ target }
        var fields = map[string]Value{ "name":name }
        ForArgs: for _, arg := range args {
                var list, ok = arg.(*List)
                if ok && list != nil && len(list.Elems) > 0 {
                        switch t := list.Elems[0].(type) {
                        case *Pair:
                                var key string
                                if key, err = t.Key.Strval(); err == nil { key = strings.ToLower(key) } else {
                                        err = wrap(pos, err); return }
                                if v, ok := fields[key]; !ok { fields[key] = t.Value } else {
                                        fields[key] = &List{elements{merge(v, t.Value)}}
                                }
                                continue ForArgs
                        case *String, *Compound:
                                switch strName {
                                case "answer","bool","option":
                                        if v, ok := fields["info"]; !ok { fields["info"] = t } else {
                                                fields["info"] = &List{elements{merge(v, t)}}
                                        }
                                        continue ForArgs
                                }
                        }
                }
                params = append(params, arg)
        }

        var defsChanged = make(map[*Def]Value)
        defer func() {
                for def, val := range defsChanged { def.value = val }

                var t bool
                if err == nil && result != nil { t, err = result.True() }
                if err == nil && t && configured && !isNil(pipe.value) /*&& !isNone(pipe.value)*/ {
                        /*switch strName { case "program-stdout", "program-stderr":
                                if false { fmt.Fprintf(stderr, "%s: %v %v\n", pos, result, pipe.value) }
                                result = pipe.value
                        }*/
                }
                if err == nil {
                        if result == nil {
                                configMessageDone(pos, "… <nil>")
                        } else if false {
                                s := elementString(nil, result, elemExpand)
                                configMessageDone(pos, "… %v", s)
                        } else {
                                s, _ := result.Strval()
                                configMessageDone(pos, "… %v", s)
                        }
                        return
                }
                switch e := err.(type) {
                case *scanner.Error: configMessageDone(pos, "… (%v)", e.Brief())
                default: configMessageDone(pos, "… (%v).", err)
                }
        } ()

        // Process configurations like:
        //   -bool
        //   -option
        //   -package
        //   ...
        if config, ok := configurationOps[strName]; ok {
                err = configMessageHead(pos, strName, fields, params...)
                if err == nil {
                        result, err = config(pos, t, pipe, fields, params...)
                        if err == nil { configured = true }
                        if optionTraceConfig {
                                t_config.tracef("configured: %v, result = %v (%s)", configured, result, typeof(result))
                        }
                }
                return
        }

        if value, ok := fields["lang"]; ok {
                var lang = configuration.project.scope.Lookup("LANG").(*Def)
                defsChanged[lang] = lang.value
                if err = lang.set(DefSimple, value); err != nil { return }
        }
        if value, ok := fields["cflags"]; ok {
                var lang = configuration.project.scope.Lookup("CFLAGS").(*Def)
                defsChanged[lang] = lang.value
                if err = lang.set(DefSimple, value); err != nil { return }
        }

        // Process configurations like:
        //   -include(foo,lib=xxx)
        //   -symbol(foo,include=xxx,lib=yyy)
        //   -function(foo,include=xxx,lib=yyy)
        //   -variable(foo,include=xxx,lib=yyy)
        //   -library(foo,include=xxx,lib=yyy)
        //   ...

        var value = configuration.project.scope.Lookup("_VALUE_").(*Def)
        defsChanged[value] = value.value
        if err = value.set(DefSimple, none); err != nil { return }
        if pipe.value != nil {
                if _, ok := pipe.value.(*None); !ok {
                        value.value = pipe.value
                }
        }

        var includesValues []Value
        var includes = configuration.project.scope.Lookup("_INCLUDES_").(*Def)
        defsChanged[includes] = includes.value
        if err = includes.set(DefSimple, none); err != nil { return }
        if strName == "include" && len(params) > 1 {
                // -include('<xxx.h>',...)
                for _, value := range params[1:] {
                        includesValues = append(includesValues, merge(value)...)
                }
        }
        if value, ok := fields["include"]; ok { includesValues = append(includesValues, value) }
        if value, ok := fields["includes"]; ok { includesValues = append(includesValues, value) }
        for _, value := range includesValues {
                var ( elems []Value; lines []string )
                if value != nil {
                        switch v := value.(type) {
                        default: elems = []Value{ v }
                        case *Group: elems = v.Elems
                        case *List:  elems = v.Elems
                        }
                }
                for _, elem := range merge(elems...) {
                        var s string
                        switch v := elem.(type) {
                        case *String: s = v.string
                        case *Bareword: s = v.string
                        case *Compound:
                                if s, err = elem.Strval(); err != nil { return }
                                if !strings.HasPrefix(s, `<`) { s = `"`+s+`"` }
                        default:
                                if s, err = elem.Strval(); err != nil { return }
                        }
                        if !(strings.HasPrefix(s, `<`) || strings.HasPrefix(s, `"`)) {
                                s = `"`+s+`"`
                        }
                        lines = append(lines, `#include `+s)
                }
                value = &String{trivial{pos},strings.Join(lines, "\n")}
                if err = includes.set(DefExpand, value); err != nil { return }
        }

        var loadlibsValues []Value
        var loadlibs = configuration.project.scope.Lookup("_LOADLIBES_").(*Def)
        defsChanged[loadlibs] = loadlibs.value
        if err = loadlibs.set(DefSimple, none); err != nil { return }
        if value, ok := fields["load"]; ok { loadlibsValues = append(loadlibsValues, value) }
        if value, ok := fields["loadlib"]; ok { loadlibsValues = append(loadlibsValues, value) }
        if value, ok := fields["loadlibs"]; ok { loadlibsValues = append(loadlibsValues, value) }
        for _, value := range loadlibsValues {
                var ( elems []Value; lines []string )
                if value != nil {
                        switch v := value.(type) {
                        default: elems = []Value{ v }
                        case *Group: elems = v.Elems
                        case *List:  elems = v.Elems
                        }
                }
                for _, elem := range merge(elems...) {
                        var s string
                        if s, err = elem.Strval(); err != nil { return }
                        if !strings.HasPrefix(s, "lib") { s = "lib"+s+".a" }
                        lines = append(lines, s)
                }
                value = &String{trivial{pos},strings.Join(lines, " ")}
                if err = loadlibs.set(DefExpand, value); err != nil { return }
        }

        var libsValues []Value
        var libs = configuration.project.scope.Lookup("_LIBS_").(*Def)
        defsChanged[libs] = libs.value
        if err = libs.set(DefSimple, none); err != nil { return }
        if value, ok := fields["lib"]; ok { libsValues = append(libsValues, value) }
        if value, ok := fields["libs"]; ok { libsValues = append(libsValues, value) }
        for _, value := range libsValues {
                var ( elems []Value; lines []string )
                if value != nil {
                        switch v := value.(type) {
                        default: elems = []Value{ v }
                        case *Group: elems = v.Elems
                        case *List:  elems = v.Elems
                        }
                }
                for _, elem := range merge(elems...) {
                        var s string
                        if s, err = elem.Strval(); err != nil { return }
                        if !strings.HasPrefix(s, "-l") { s = "-l" + s }
                        lines = append(lines, s)
                }
                value = &String{trivial{pos},strings.Join(lines, " ")}
                if err = libs.set(DefExpand, value); err != nil { return }
        }
        if err = configMessageHead(pos, strName, fields, params...); err == nil {
                configured, result, err = configureExec(pos, t, strName, params...)
        }

        delete(fields, "include")
        delete(fields, "includes")
        delete(fields, "load")
        delete(fields, "loadlib")
        delete(fields, "loadlibs")
        delete(fields, "lib")
        delete(fields, "libs")
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
        }); err != nil { err = wrap(pos, err); return }

        var project *Project
        var filename string
        var file *File
        if file, _ = t.def.target.value.(*File); file == nil {
                var s string
                if s, err = t.def.target.value.Strval(); err != nil { err = wrap(pos, err); return }

                var okay bool
                okay, err = t.forClosureProject(func(p *Project) (ok bool, err error) {
                        if file = p.matchFile(s); file != nil { project, ok = p, true }
                        if optDebug && file != nil { fmt.Fprintf(stderr, "%s: %v: file %v\n", pos, p, file) }
                        return
                })
                if err != nil { return } else if !okay { err = errorf(pos, "'%s' is not a file", s); return }
        }
        if file == nil { err = errorf(pos, "no file target"); return }
        if filename, err = file.Strval(); err != nil { err = wrap(pos, err); return } else
        if filename == "" { err = errorf(pos, "`%v` has empty filename", file); return } else
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
                if err != nil { err = wrap(pos, err); return }
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
                        err = wrap(pos, err)
                }
        } (filename, closure)

        var data bytes.Buffer
        for _, arg := range append(args, t.def.buffer.value) {
                var str string
                if str, err = arg.Strval(); err != nil { return }
                if str == "" { continue }
                if err = configure(pos, &data, closure.project, str); err != nil { return }
        }
        if data.Len() == 0 { err = errorf(pos, "no input data"); return }

        if optVerbose { fmt.Fprintf(stderr, "smart: Checking %v …", file) }
        if file.info != nil {
                if same, e := crc64CheckFileModeContent(filename, data.Bytes(), optMode); e != nil {
                        if optVerbose { fmt.Fprintf(stderr, "… (error: %s)\n", e) }
                        err = wrap(pos, e, err); return
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
                        err = wrap(pos, err)
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
                err = wrap(pos, err)
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

        var optAccumulate bool
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = tryParseFlags(args, []string{
                "a,accumulate",
        }, func(ru rune, v Value) {
                switch ru {
                case 'a': if optAccumulate, err = trueVal(v, true); err != nil { return }
                }
        }); err != nil { return }

        var target = t.def.target.value
        if isNil(target) || isNone(target) {
                err = errorf(pos, "target is nil for entry '%s'", t.entry.target)
                return
        }

        var name string
        if name, err = target.Strval(); err != nil { return }

        var def, alt = t.program.project.scope.define(t.program.project, name, nil)
        if alt != nil { def, _ = alt.(*Def) }
        if def == nil {
                err = errorf(pos, "cannot define configuration `%s`", name)
                return
        } else { result = def } // Set result above all!

        if optionTraceConfig {
                t_config.tracef("%s: %v (%T)", def.name, def.value, def.value)
                defer func() { t_config.tracef("%s: %v (%T)", def.name, def.value, def.value) } ()
        }

        if !isNil(def.value) { // Check if it's already configured?
                // reconfigure the def or return it
                if !optionReconfig { return }
                if done, found := configuration.done[def]; done && found {
                  return // already executed (re)configuration
                }
        }

        var value Value
        if len(args) == 0 { // Empty configuration: (configure)
                if value, err = t.def.buffer.Call(pos); err != nil {
                        err = wrap(pos, err)
                        return
                } else if value == nil {
                        err = errorf(pos, "`%v` not configured (%v)", target, value)
                        return
                } else if value == def || value.refs(def) { return }
                switch v := value.(type) {
                default: err = def.set(DefConfig, value)
                case *ExecResult:
                        var s string
                        if v.wg.Wait(); v.Status == 0 && v.Stdout.Buf != nil {
                                s = v.Stdout.Buf.String()
                        } else if v.Stderr.Buf != nil {
                                s = v.Stderr.Buf.String()
                        }
                        err = def.set(DefConfig, &String{trivial{pos},s})
                }
                if err != nil { err = wrap(pos, err) }
                return
        } else if err = def.set(DefConfig, nil); err != nil { return }

        var configured bool
        ForConfig: for i, a := range args {
                if def.value == nil && i > 0 { break ForConfig }

                var ( name Value ; para []Value )
                switch arg := a.(type) {
                case *Argumented:
                        if flag, okay := arg.value.(*Flag); !okay {
                                err = wrap(pos, errorf(a.Position(), "`%v` is unsupported\n", arg.value), err)
                                return
                        } else {
                                name, para = flag.name, arg.args
                        }
                case *Flag:
                        if isNil(arg.name) || isNone(arg.name) {
                                err = wrap(pos, errorf(a.Position(), "`%v` is unsupported\n", arg.name), err)
                                return
                        } else {
                                name = arg.name
                        }
                default:
                        err = wrap(pos, errorf(a.Position(), "`%v` is unsupported\n", a), err)
                        return
                }
                if name == nil {
                        err = wrap(pos, errorf(a.Position(), "unknown configure `%v` (%T)\n", a, a))
                        return
                }

                configured, value, err = configureDo(pos, t, target, def, t.def.buffer, name, para)
                if err != nil { err = wrap(pos, err); return } else if !configured {
                        //err = errorf(pos, "%s not configured", name)
                        return
                }
                if optionTraceConfig { t_config.tracef("configured: %v (%s) (%v)", value, typeof(value), def.origin) }
                if value == nil { value = &Nil{trivial{a.Position()}} } else
                if v, e := value.expand(expandAll); e == nil { value = v } else { err = wrap(a.Position(), e); return }
                if value == def || value.refs(def) {
                        // Value is the Def, does nothing!
                } else if optAccumulate {
                        err = def.append(value)
                } else {
                        err = def.set(DefConfig, value)
                }
                if err != nil { err = wrap(pos, err); return }
                // Marks done (needed for reconfiguring)!
                configuration.done[def] = true
        }
        if !configured && err == nil {
                err = errorf(pos, "`%v` not configured", target)
        }
        return
}
