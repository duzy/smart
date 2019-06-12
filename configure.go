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
        "strings"
        "regexp"
        "bufio"
        "sort"
        "fmt"
        "os"
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
        configs []*RuleEntry // -configure entry
        entries []*RuleEntry // order list
        //assert *RuleEntry // -assert entry
}{
        fset: token.NewFileSet(),
        libraries: make(map[string]*libraryinfo),
        packages: make(map[string]*packageinfo),
        done: make(map[*Def]bool),
}

var configurationOps = map[string] func(pos Position, prog *Program, def *Def, args... Value) (result Value, err error) {
        "answer":  configureAnswer,
        "bool":    configureBool,
        "dump":    configureDump,
        "option":  configureOption,
        "package": configurePackage,
}

func init_configuration(paths searchlist) (err error) {
        configuration.scope = NewScope(context.globe.scope, nil, "configuration")
        configuration.paths = paths
        
        var l = &loader{
                Context: &context,
                scope:    configuration.scope,
                fset:     configuration.fset,
                paths:    configuration.paths,
                loaded:   make(map[string]*Project),
                ruleParseFunc: parseRuleClause,
        }
        var filename = filepath.Join(context.workdir, "~.smart")
        if err = l.loadFile(filename, configurationInitFile); err != nil {
                return
        } else if project, ok := l.loaded[filename]; ok {
                configuration.project = project
        } else {
                err = fmt.Errorf("configuration: `%v` not loaded\n", filename)
        }

        if configuration.project == nil {
                panic("configuration.project still nil")
        }

        // Define configuration entries.
        /*for _, entry := range configuration.entries {
                var name string
                if name, err = entry.target.Strval(); err != nil { return }
                var project = entry.OwnerProject()
                def, alt := project.scope.Def(project, name, nil) // unconfigured
                if alt != nil {
                        err = fmt.Errorf("configure: `%v` already existed", name)
                        return
                } else if def == nil {
                        unreachable()
                }
        }*/
        return
}

func do_configuration() (err error) {
        var ( project *Project; num int )
        var reportConfiguredNum = func() {
                if project != nil {
                        fmt.Fprintf(stderr, "configure: Project %v configured %v items.\n", project.name, num)
                }
        }

        var ( file *os.File; writer *bufio.Writer )
        defer func() {
                reportConfiguredNum()
                if writer != nil { if err := writer.Flush(); err != nil {}}
                if file != nil { if err := file.Close(); err != nil {}}
        } ()

        var defs = make(map[string]Value)
        for _, entry := range configuration.entries {
                var pos = token.Position(entry.Position)
                if p := entry.OwnerProject(); project != p {
                        if ctd := p.scope.FindDef("CTD"); ctd == nil {
                                unreachable()
                        } else if s, e := ctd.Strval(); e != nil {
                                err = scanner.WrapErrors(pos, e, err)
                                return
                        } else if e = os.MkdirAll(s, os.FileMode(0755)); e != nil {
                                err = scanner.WrapErrors(pos, e, err)
                                return
                        } else if f, e := os.OpenFile(filepath.Join(s, "configuration.sm"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0600)); e == nil {
                                if writer != nil {
                                        if e = writer.Flush(); e != nil {
                                                err = scanner.WrapErrors(pos, e, err)
                                                return
                                        }
                                }
                                if file != nil {
                                        if e = file.Close(); e != nil {
                                                err = scanner.WrapErrors(pos, e, err)
                                                return
                                        }
                                }
                                file, writer = f, bufio.NewWriter(f)
                                fmt.Fprintf(writer, "# %s (%s) configuration\n", p.name, p.relPath)
                        } else {
                                err = scanner.WrapErrors(pos, e, err)
                                return
                        }
                        reportConfiguredNum()
                        fmt.Fprintf(stderr, "configure: Project %s …… (%s)\n", p.name, p.relPath)
                        project, num = p, 0
                }

                if _, e := entry.Execute(entry.Position); e != nil {
                        err = scanner.WrapErrors(pos, e, err)
                } else if s, e := entry.target.Strval(); e != nil {
                        err = scanner.WrapErrors(pos, e, err)
                } else if def := project.scope.FindDef(s); def != nil {
                        if _, ok := defs[s]; ok {
                                // already defined
                                continue
                        }
                        if def.Value == nil {
                                // Set <nil> value with exec-assigning ('!=')
                                // to a None value.
                                fmt.Fprintf(writer, "%v !=\n", def.name)
                        } else {
                                vs := elementString(def, def.Value, elemNoBrace)
                                fmt.Fprintf(writer, "%v = %v\n", def.name, vs)
                        }
                        defs[s] = def.Value
                        num += 1
                } else {
                        //e := scanner.Errorf(token.Position(pos), "`%s` unconfigured", s)
                        e := fmt.Errorf("`%s` unconfigured", s)
                        err = scanner.WrapErrors(pos, e, err)
                }
        }
        if err != nil { return }

        var executed = make(map[*RuleEntry]bool)
        for _, entry := range configuration.configs {
                if a, b := executed[entry]; a && b {
                        continue
                } else {
                        executed[entry] = true
                }
                if _, e := entry.Execute(entry.Position); e != nil {
                        pos := token.Position(entry.Position)
                        err = scanner.WrapErrors(pos, e, err)
                }
        }
        executed = nil

        printLeavingDirectory()
        return
}

func configPrintf(pos Position, str string, args... interface{}) {
        var debug bool
        if o := configuration.scope.Lookup("DEBUG"); o != nil { debug = o.True() }
        if debug { str = fmt.Sprintf("%v:info: %s", pos, str) }
        fmt.Fprintf(stderr, str, args...)
}

func configMessageDone(pos Position, str string, args... interface{}) {
        if !strings.HasSuffix(str, "\n") { str += "\n" }
        configPrintf(pos, str, args...)
}

func configPrintMessageHead(pos Position, fields map[string]Value, args... Value) {
        var str string
        var ints []interface{}
        /*if msg, ok := fields["info"]; ok {
                str = "configure: "
                if s, err := msg.Strval(); err == nil && len(s) > 0 {
                        r, size := utf8.DecodeRuneInString(s)
                        if size > 0 && unicode.IsUpper(r) {
                                str += s
                        } else {
                                str += "Checking " + s
                        }
                }
        } else */if name, ok := fields["name"]; ok {
                str = "configure: Checking"
                if s, err := name.Strval(); err == nil {
                        str += " " + s
                        if len(args) > 1 { str += "s" }
                }
        }
        for i, a := range args {
                s, _ := a.Strval()
                ints = append(ints, s)
                if i == 0 {
                        str += " (%v"
                } else {
                        str += " %v"
                }
                if i+1 == len(args) { str += ")" }
        }
        str += " …"
        configPrintf(pos, str, ints...)
}

var strStructMember = &String{ "struct member" }
func configMessageHead(pos Position, op string, fields map[string]Value, params... Value) (err error) {
        var str = "configure: "
        if v, ok := fields["info"]; ok {
                var s string
                s, err = v.Strval()
                if err == nil && len(s) > 0 {
                        r, size := utf8.DecodeRuneInString(s)
                        if size > 0 && unicode.IsUpper(r) {
                                str += s
                        } else {
                                str += "Checking " + s
                        }
                        str += " …"
                        configPrintf(pos, str)
                }
                return
        }
        switch n := len(params); op {
        case "if":
                configPrintMessageHead(pos, fields, params[0])
        case "option":
                configPrintMessageHead(pos, fields, params[0])
        case "compiles":
                configPrintMessageHead(pos, fields, params[0])
        case "library":
                // Examples:
                //   -library(foo,func,include='<foo.h>')
                //   -library(foo,func)
                //   -library(foobar)
                // TODO: using this form instead:
                //   -library(foo [,function=func] [,include='<foo.h>']...)
                var libs = params[1:]
                /*str += "Checking "
                if len(libs) == 1 {
                        str += "library "
                } else if len(libs) > 1 {
                        str += "libraries "
                } else {
                        str += "library (but none)"
                }
                for i, param := range libs {
                        var s string
                        s, err = param.Strval()
                        if err != nil { return }
                        if i > 0 { str += "," }
                        str += s
                }*/
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
                configPrintf(pos, str)
        case "include":
                var incs = params[1:]
                var s string
                s, err = incs[0].Strval()
                if err != nil { return }
                str += "Checking include " + s
                str += " …"
                configPrintf(pos, str)
        case "symbol":
                var syms = params[1:]
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
                configPrintf(pos, str)
        case "function":
                var funcs = params[1:]
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
                configPrintf(pos, str)
        case "struct-member":
                // Examples:
                //   -struct-member('struct stat',st_mtimespec.tv_nsec,include=('<sys/types.h>','<sys/stat.h>'))
                var mems = params[1:]
                var s string
                if len(mems) > 1 {
                        s, err = mems[0].Strval()
                        if err != nil { return }
                        str += "Checking struct member (" + s + ") "

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
                configPrintf(pos, str)
        case "type":
                var typs = params[1:]
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
                configPrintf(pos, str)
        case "package":
                if n > 2 {
                        configPrintMessageHead(pos, fields, params[2])
                } else {
                        configPrintMessageHead(pos, fields, params[0])
                }
        default:
                configPrintMessageHead(pos, fields, params...)
        }
        return
}

func configureDump(pos Position, prog *Program, def *Def, params... Value) (result Value, err error) {
        result = def.Value
        return
}

// (configure -bool)
// (configure -bool(opt_true,opt_false))
func configureBool(pos Position, prog *Program, def *Def, params... Value) (result Value, err error) {
        var positive bool
        var previous Value
        if previous, err = def.Call(pos); err != nil {
                return
        } else {
                var res Value
                res, err = previous.expand(expandAll)
                if err == nil && res != previous {
                        previous = res
                }
        }

        for i, v := range merge(previous) {
                if v == nil { continue }
                if i == 0 {
                        positive = v.True()
                } else {
                        positive = positive && v.True()
                }
                if !positive { break }
        }
        if positive {
                if len(params) > 1 { // [NAME 1 0]
                        result = params[1]
                } else {
                        result = universaltrue
                }
        } else {
                if len(params) > 2 { // [NAME 1 0]
                        result = params[2]
                } else {
                        result = universalfalse
                }
        }
        return
}

// (configure -answer)
// (configure -answer(opt_true,opt_false))
func configureAnswer(pos Position, prog *Program, def *Def, params... Value) (result Value, err error) {
        return configureBool(pos, prog, def, params[0], universalyes, universalno)
}

func configureOption(pos Position, prog *Program, def *Def, args... Value) (result Value, err error) {
        if result, err = def.Call(pos); err == nil {
                if result == nil { result = universalno }
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
                file = stat(name+".smart", "", path)
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

        var filename = file.FullName()
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
//func configureLibrary(pos Position, prog *Program, args... Value) (result Value, err error) {
//        return
//}

// -package finds system package in a way similar to cmake.find_package
func configurePackage(pos Position, prog *Program, def *Def, args... Value) (result Value, err error) {
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
                        case packageSmart:
                                if info, err = loadPackageSmartInfo(pos, name); err != nil { return }
                        case packageConfig:
                                if info, err = loadPackageConfigInfo(pos, name); err != nil { return }
                        case packageUnknown:
                                fmt.Fprintf(stderr, "%v: package `%v`: unknown type\n", name)
                        }
                        if info != nil {
                                configuration.packages[name] = info
                                result = universalyes
                                break
                        }
                }
        }
        return
}

func scanCommandFailedError(err error) (n, en int, tag string) {
        switch e := err.(type) {
        case scanner.Errors:
                for _, p := range e {
                        n, en, tag = scanCommandFailedError(p.Err)
                        if n == 2 { return }
                }
        case *scanner.Error:
                n, en, tag = scanCommandFailedError(e.Err)
        default:
                var s = err.Error()
                n, _ = fmt.Sscanf(s, errCommandFailedFmt, &tag, &en)
        }
        return
}

func configureEntry(pos Position, prog *Program, s string, params... Value) (configured bool, result Value, err error) {
        var res []Value
        var entry *RuleEntry
        if entry, err = configuration.project.resolveEntry("-"+s); err != nil {
                err = scanner.Errorf(token.Position(pos), "resolve %v: %v", s, err)
        } else if entry == nil {
                err = scanner.Errorf(token.Position(pos), "unknown configuration `%v` (no such entry)", s)
        } else if res, err = prog.passExecution(pos, entry, params...); err != nil {
                n, _, _ := scanCommandFailedError(err)
                if n == 2 {
                        configured, err = true, nil
                } else {
                        err = scanner.WrapErrors(token.Position(pos), err)
                }
        } else {
                if res != nil { result = MakeListOrScalar(res) }
                configured = true
        }
        return
}

// (configure -xxx)
// (configure -xxx=yyy)
// (configure -xxx(...))
func configureAction(pos Position, prog *Program, target Value, def, pipe *Def, name Value, args []Value) (configured bool, result Value, err error) {
        var strName string
        if strName, err = name.Strval(); err != nil { return }
        if strName == "" {
                err = fmt.Errorf("`%v` empty configuration (%T)", name, name)
                return
        }

        var params = []Value{ target }
        var fields = map[string]Value{ "name": name }
ForArgs:
        for _, arg := range args {
                if list := arg.(*List); list != nil && list.Len() > 0 {
                        var key string
                        switch t := list.Elems[0].(type) {
                        case *Pair:
                                if key, err = t.Key.Strval(); err != nil { return }
                                key = strings.ToLower(key)
                                if v, ok := fields[key]; ok {
                                        fields[key] = &List{elements{merge(v, t.Value)}}
                                } else {
                                        fields[key] = t.Value
                                }
                                continue ForArgs
                        case *String:if strName == "option" {
                                key = "info"
                                if v, ok := fields[key]; ok {
                                        fields[key] = &List{elements{merge(v, t)}}
                                } else {
                                        fields[key] = t
                                }
                                continue ForArgs
                        }}
                }
                params = append(params, arg)
        }

        defer func() {
                if configured && err == nil && result != nil && result.True() && strName != "compiles" {
                        if v := pipe.Value; v != nil && v.Type() != NoneType { result = v }
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
                case scanner.Errors:
                        if n := len(e); n == 1 {
                                configMessageDone(pos, "… (%v, and %d errors)", e[0].Err, len(e))
                        } else if n == 2 {
                                configMessageDone(pos, "… (%v, and 1 more error)", e[0].Err)
                        } else if n > 2 {
                                configMessageDone(pos, "… (%v, and %d more errors)", e[0].Err, len(e)-1)
                        }
                case *scanner.Error: configMessageDone(pos, "… (%v)", e.Err)
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
                        result, err = config(pos, prog, pipe, params...)
                        if err == nil { configured = true }
                }
                return
        }

        // Process configurations like:
        //   -include(foo,lib=xxx)
        //   -symbol(foo,include=xxx,lib=yyy)
        //   -function(foo,include=xxx,lib=yyy)
        //   -variable(foo,include=xxx,lib=yyy)
        //   -library(foo,include=xxx,lib=yyy)
        //   ...

        var value = configuration.project.scope.Lookup("_VALUE_").(*Def)
        if err = value.set(DefSimple, universalnone); err != nil { return }
        if pipe.Value != nil && pipe.Value.Type() != NoneType {
                value.Value = pipe.Value
        }

        var includesValues []Value
        var includes = configuration.project.scope.Lookup("_INCLUDES_").(*Def)
        if err = includes.set(DefSimple, universalnone); err != nil { return }
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
                        if strings.HasPrefix(s, `<`) || strings.HasPrefix(s, `"`) {
                                s = fmt.Sprintf(`#include %s`, s)
                        } else {
                                s = fmt.Sprintf(`#include "%s"`, s)
                        }
                        lines = append(lines, s)
                }
                value = &String{strings.Join(lines, "\n")}
                if err = includes.set(DefExpand, value); err != nil { return }
        }

        var loadlibsValues []Value
        var loadlibs = configuration.project.scope.Lookup("_LOADLIBES_").(*Def)
        if err = loadlibs.set(DefSimple, universalnone); err != nil { return }
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
                        if !strings.HasPrefix(s, "lib") {
                                s = fmt.Sprintf("lib%s.a", s)
                        }
                        lines = append(lines, s)
                }
                value = &String{strings.Join(lines, " ")}
                if err = loadlibs.set(DefExpand, value); err != nil { return }
        }

        var libsValues []Value
        var libs = configuration.project.scope.Lookup("_LIBS_").(*Def)
        if err = libs.set(DefSimple, universalnone); err != nil { return }
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
                        if !strings.HasPrefix(s, "-l") {
                                s = fmt.Sprintf("-l%s", s)
                        }
                        lines = append(lines, s)
                }
                value = &String{strings.Join(lines, " ")}
                if err = libs.set(DefExpand, value); err != nil { return }
        }
        /*
        delete(fields, "include")
        delete(fields, "includes")
        delete(fields, "load")
        delete(fields, "loadlib")
        delete(fields, "loadlibs")
        delete(fields, "lib")
        delete(fields, "libs")
        */
        if err = configMessageHead(pos, strName, fields, params...); err == nil {
                configured, result, err = configureEntry(pos, prog, strName, params...)
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

func walkFiles(root string, pats []Value, fn filewalkFunc) error {
        return walkFileInfos(root, pats, func(path string, info os.FileInfo, err error) error {
                if err != nil { return err }
                var rel string
                if rel, err = filepath.Rel(root, path); err != nil {
                        return err
                }
                file := stat(rel, "", root, info)
                if enable_assertions {
                        assert(file != nil, "`%s` file is nil", rel)
                }
                return fn(file, err)
        })
}

// configure-file modifier (see also builtinConfigureFile), example usage:
// 
//     config.h:[(compare) (configure-file)]: config.h.in
//     
func modifierConfigureFile(pos Position, prog *Program, args... Value) (result Value, err error) {
        // Only configure file in update mode.
        if prog.pc.mode != updateMode {
                // Return to not overriding the configured file. 
                return
        }

        var target Value
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        for _, arg := range args {
                switch a := arg.(type) {
                case *None, *Flag, *Pair:
                default:
                        if target == nil {
                                target = a
                        } else {
                                err = fmt.Errorf("too many configure files")
                                return
                        }
                }
        }

        if target == nil {
                if target, err = prog.scope.Lookup("@").(*Def).Call(pos); err != nil {
                        return
                } else if target == nil {
                        err = fmt.Errorf("unknown configure file")
                        return
                } else {
                        args = append(args, target)
                }
        }

        var value Value
        if value, err = prog.scope.Lookup("-").(*Def).Call(pos); err != nil {
                return
        }

        args = append(args, value)
        result, err = builtinConfigureFile(pos, args...)
        return
}

// extract-configuration extracts configuration from C/C++ files, example usage:
//
//      config.h.in:[(extract-configuration)]: $(wildcard *.cpp)
//
func modifierExtractConfiguration(pos Position, prog *Program, args... Value) (result Value, err error) {
        // Only generate configure file in update mode
        if m := prog.pc.mode; m != updateMode {
                return
        }

        var target Value
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        if target, err = prog.scope.Lookup("@").(*Def).Call(pos); err != nil { return }

        var (depends []Value; val Value)
        if val, err = prog.scope.Lookup("^").(*Def).Call(pos); err != nil { return }
        if depends, err = mergeresult(ExpandAll(val)); err != nil { return }

        val = nil // clear

        var optPath bool
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var pats []Value
        var rxs []*regexp.Regexp
        var optTarget string
        var optPerm = os.FileMode(0640) // sys default 0666
        for _, arg := range args {
                var opt bool
                var name string
                switch a := arg.(type) {
                default:
                        pats = append(pats, a)
                case *Group:
                        pats = append(pats, a.Elems...)
                case *Flag:
                        if opt, err = a.is('p', "path"); err != nil { return } else if opt { optPath = opt }
                case *Pair:
                        if name, err = a.Key.Strval(); err != nil { return }
                        switch name {
                        case "rx", "-rx", "grep", "-grep", "-g":
                                var (s string; x *regexp.Regexp)
                                if s, err = a.Value.Strval(); err != nil {
                                        return
                                } else if x, err = regexp.Compile(s); err != nil {
                                        return
                                } else {
                                        rxs = append(rxs, x)
                                }
                        case "-m", "-mode", "mode":
                                var num int64
                                if num, err = a.Integer(); err != nil { return }
                                optPerm = os.FileMode(num & 0777)
                        case "-t", "-target", "target":
                                if optTarget, err = a.Value.Strval(); err != nil {
                                        return
                                }
                        }
                }
        }
        if len(pats) == 0 {
                err = fmt.Errorf("missing file names (patterns)")
                return
        }
        if len(rxs) == 0 {
                err = fmt.Errorf("missing -rx=... flags")
                return
        }
        if optTarget == "" {
                optTarget = "configuration"
        }

        var outFile string
        if outFile, err = target.Strval(); err != nil { return }
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
                                err = walkFiles(s, pats, func(file *File, err error) error {
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
                        file := stat(name, "", dir)
                        if file == nil {
                                err = scanner.Errorf(token.Position(pos), "`%s` file not found (configure)", name)
                                return
                        } else if file.info.IsDir() {
                                err = walkFiles(s, pats, func(file *File, err error) error {
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
                case *File: s = t.FullName()
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
func modifierConfigure(pos Position, prog *Program, args... Value) (result Value, err error) {
        // don't configure in compare mode
        if m := prog.pc.mode; m == compareMode { return }

        var (
                opts = []string{
                        "a,accumulate",
                }
                optAccumulate bool
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else {
                var va []Value
        ForArgs:
                for _, arg := range args {
                        var ( v Value ; runes []rune ; names []string )
                        v, runes, names, err = parseOpts(arg, opts)
                        if err != nil {
                                err = scanner.WrapErrors(token.Position(pos), err)
                                return
                        }
                        if v == nil && runes == nil && names == nil {
                                va = append(va, arg)
                                continue ForArgs
                        }
                        for _, ru := range runes {
                                switch ru {
                                case 'a': optAccumulate = true
                                }
                        }                        
                }
                args = va // reset args
        }

        var ( target Value; name string )
        if target, err = prog.scope.Lookup("@").(*Def).Call(pos); err != nil { return }
        if name, err = target.Strval(); err != nil { return }

        var def, alt = prog.project.scope.Def(prog.project, name, nil)
        if alt != nil { def, _ = alt.(*Def) }
        if def == nil {
                err = scanner.Errorf(token.Position(pos), "cannot define configuration `%s`", name)
                return
        }

        result = def // Set result above all.

        if def.Value != nil { // if it's already configured
                // reconfigure the def or return it
                if !optionReconfig { return }
                if done, found := configuration.done[def]; done && found {
                        return // already executed (re)configuration
                }
        }

        var pipe = prog.scope.Lookup("-").(*Def)
        if len(args) == 0 { // zero configuration: (configure)
                var value Value
                value, err = pipe.Call(pos)
                if err != nil { return }
                if value != nil {
                        switch value.Type() {
                        case NoneType: err = def.set(DefExecute, nil)
                        default: err = def.set(DefExpand, value)
                        }
                } else {
                        err = fmt.Errorf("`%v` not configured (%v)", target, value)
                        err = scanner.WrapErrors(token.Position(pos), err)
                }
                return
        }

        // Reset configuration value to nil
        if err = def.set(DefExecute, nil); err != nil { return }

        var ( value Value; configured bool )
ForConfig:
        for i, a := range args {
                var ( name Value ; para []Value )
                if def.Value == nil && i > 0 { break ForConfig }
                switch arg := a.(type) {
                case *Pair: // Set def
                        /*switch k := arg.Key.(type) {
                        case *Bareword:
                                def, alt := prog.project.scope.Def(prog.project, k.string, universalnone)
                                if alt != nil {
                                        if p, _ := alt.(*Def); p != nil && p != def { def = p }
                                }
                                err = def.set(DefSimple, arg.Value)
                                if err == nil { continue ForConfig }
                        }*/
                        err = scanner.Errorf(token.Position(pos), "`%v` is useless for assignment\n", a)
                case *Argumented:
                        if flag, okay := arg.Val.(*Flag); okay {
                                name = flag.Name
                                para = arg.Args
                        }
                case *Flag:
                        if arg.Name != nil && arg.Name.Type() != NoneType {
                                name = arg.Name
                        }
                }
                if err == nil && name != nil {
                        configured, value, err = configureAction(pos, prog, target, def, pipe, name, para)
                        if err != nil {
                                err = scanner.WrapErrors(token.Position(pos), err)
                        }
                } else if err == nil {
                        err = scanner.Errorf(token.Position(pos), ") unknown configure action `%v` (%T)\n", a, a)
                }
                if err != nil { break ForConfig }
                if configured {
                        if value == nil { value = universalnil }
                        if optAccumulate {
                                err = def.append(value)
                        } else {
                                err = def.set(DefSimple, value)
                        }
                        if err != nil {
                                err = scanner.WrapErrors(token.Position(pos), err)
                                return
                        }
                        // marking it done (needed for reconfiguring)
                        configuration.done[def] = true
                }
        }
        if !configured {
                if err == nil {
                        err = fmt.Errorf("`%v` not configured", target)
                }
                err = scanner.WrapErrors(token.Position(pos), err)
        }
        return
}
