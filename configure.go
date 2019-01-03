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
        entires []*RuleEntry // order list
}{
        fset: token.NewFileSet(),
        libraries: make(map[string]*libraryinfo),
        packages: make(map[string]*packageinfo),
        done: make(map[*Def]bool),
}

var configurationOps = map[string] func(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        "include":      configureInclude,
        "option":       configureOption,
        "package":      configurePackage,
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
        /*for _, entry := range configuration.entires {
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

func do_configuration() error {
        var ( errs scanner.Errors; project *Project; num int )
        var reportConfiguredNum = func() {
                if project != nil {
                        fmt.Printf("configure: Project %v configured %v items.\n", project.name, num)
                }
        }

        var ( file *os.File; writer *bufio.Writer )
        defer func() {
                reportConfiguredNum()
                if writer != nil { if err := writer.Flush(); err != nil {}}
                if file != nil { if err := file.Close(); err != nil {}}
        } ()

        for _, entry := range configuration.entires {
                if p := entry.OwnerProject(); project != p {
                        if ctd := p.scope.FindDef("CTD"); ctd == nil {
                                unreachable()
                        } else if s, err := ctd.Strval(); err != nil {
                                return err
                        } else if f, err := os.OpenFile(filepath.Join(s, "configuration.sm"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0600)); err == nil {
                                if writer != nil { if err = writer.Flush(); err != nil { return err }}
                                if file != nil { if err = file.Close(); err != nil { return err }}
                                file, writer = f, bufio.NewWriter(f)
                                fmt.Fprintf(writer, "#configuration %v\n", p.name)
                        } else {
                                return err
                        }
                        reportConfiguredNum()
                        fmt.Printf("configure: Project %v …… (%v)\n", p.name, p.relPath)
                        project, num = p, 0
                }

                var pos = entry.Position
                if _, err := entry.Execute(pos); err != nil {
                        switch e := err.(type) {
                        case *scanner.Error: errs = append(errs, e)
                        case scanner.Errors: errs = append(errs, e...)
                        default: errs = append(errs, &scanner.Error{ pos, e })
                        }
                } else if s, err := entry.target.Strval(); err != nil {
                        e := scanner.Errorf(pos, "%v (%v)", err, entry.target)
                        errs = append(errs, e.(*scanner.Error))
                } else if def := project.scope.FindDef(s); def != nil {
                        if def.Value == nil {
                                // Set <nil> value with exec-assigning ('!=')
                                // to a None value.
                                fmt.Fprintf(writer, "%v !=\n", def.name)
                                /*
                        } else if d, ok := def.Value.(*Def); ok {
                                if p := d.OwnerProject(); p == def.OwnerProject() {
                                        fmt.Fprintf(writer, "%s = $(%s)\n", def.name, d.name)
                                } else {
                                        fmt.Fprintf(writer, "%s = $(%s->%s)\n", def.name, p.name, d.name)
                                }
                        } else if true {
                                fmt.Fprintf(writer, "%s = %s\n", def.name, def.Value)
                                */
                        } else {
                                fmt.Fprintf(writer, "%v = %v\n", def.name, elementString(def, def.Value))
                        }
                        num += 1
                } else {
                        e := scanner.Errorf(pos, "`%s` not configured", s)
                        errs = append(errs, e.(*scanner.Error))
                }
        }
        return errs
}

func configinfo(pos token.Position, str string, args... interface{}) {
        var debug bool
        if o := configuration.scope.Lookup("DEBUG"); o != nil { debug = o.True() }
        if debug { str = fmt.Sprintf("%v:info: %s", pos, str) }
        fmt.Printf(str, args...)
}

func configinfon(pos token.Position, str string, args... interface{}) {
        if !strings.HasSuffix(str, "\n") { str += "\n" }
        configinfo(pos, str, args...)
}

func configinfox(pos token.Position, fields map[string]Value, args... Value) {
        var str string
        var ints []interface{}
        if msg, ok := fields["info"]; ok {
                str = "configure: "
                if s, err := msg.Strval(); err == nil && len(s) > 0 {
                        r, size := utf8.DecodeRuneInString(s)
                        if size > 0 && unicode.IsUpper(r) {
                                str += s
                        } else {
                                str += "Checking " + s
                        }
                }
        } else if name, ok := fields["name"]; ok {
                str = "configure: Checking"
                if s, err := name.Strval(); err == nil {
                        str += " " + s
                        if len(args) > 1 { str += "s" }
                }
        }
        for _, a := range args {
                s, _ := a.Strval()
                ints = append(ints, s)
                str += " %v"
        }
        str += " …"
        configinfo(pos, str, ints...)
}

func configmessage(pos token.Position, s string, fields map[string]Value, params... Value) {
        if _, ok := fields["info"]; ok {
                configinfox(pos, fields)
                return
        }
        switch n := len(params); s {
        case "if":
                configinfox(pos, fields, params[0])
        case "option":
                configinfox(pos, fields, params[0])
        case "compiles":
                configinfox(pos, fields, params[0])
        case "library":
                if n == 2 {
                        s := fmt.Sprintf("%v(%v)", params[0], params[1])
                        configinfox(pos, fields, &String{s})
                } else {
                        configinfox(pos, fields, params...)
                }
        case "include", "symbol", "function", "package":
                if n > 2 {
                        configinfox(pos, fields, params[2])
                } else {
                        configinfox(pos, fields, params[0])
                }
        default:
                configinfox(pos, fields, params...)
        }
}

func configurePair(pos token.Position, prog *Program, key, val Value) (result Value, err error) {
        switch k := key.(type) {
        case *Bareword:
                def, alt := prog.project.scope.Def(prog.project, k.string, universalnone)
                if alt != nil {
                        if p, _ := alt.(*Def); p != nil && p != def { def = p }
                }
                err = def.set(DefSimple, val)
        default:
                err = scanner.Errorf(pos, "unknown configuration `%v = %v` (%T %T)\n", key, key, val)
        }
        return
}

func configureInclude(pos token.Position, prog *Program, params... Value) (result Value, err error) {
        var includes = configuration.project.scope.Lookup("INCLUDES").(*Def)
        for _, value := range params[2:] {
                var s string
                if list, ok := value.(*List); ok {
                        var elems []Value
                        for _, elem := range list.Elems {
                                if s, err = elem.Strval(); err != nil { return }
                                elem = &String{fmt.Sprintf("#include %s\n", s)}
                                elems = append(elems, elem)
                        }
                        value = &List{elements{elems}}
                } else if s, err = value.Strval(); err == nil {
                        value = &String{fmt.Sprintf("#include %s\n", s)}
                } else {
                        return
                }
                if err = includes.append(value); err != nil { return }
        }
        _, result, err = configureEntry(pos, prog, "include", params[:2]...)
        return
}

func configureOption(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        if result, err = prog.scope.Lookup("-").(*Def).Call(pos); err == nil {
                if result == nil { result = universalno }
        }
        return
}

func loadPackageSmartInfo(pos token.Position, name string) (info *packageinfo, err error) {
        var found string
        for _, path := range configuration.paths {
                s := filepath.Join(path, name + ".smart")
                if fi, er := stat(s); er == nil && fi != nil {
                        found = s; break
                }
        }
        if found == "" { return }
        var l = &loader{
                Context: &context,
                fset:     configuration.fset,
                scope:    configuration.scope,
                paths:    configuration.paths,
                loaded:   make(map[string]*Project),
        }
        if err = l.loadFile(found, nil); err != nil {
                /*if p, ok := err.(scanner.Errors); ok {
                        fmt.Fprintf(os.Stderr, "%v\n", p)
                        fmt.Fprintf(os.Stderr, "%v: `%v` package not loaded\n", pos, name)
                } else {
                        fmt.Fprintf(os.Stderr, "%v: package %v: %v\n", pos, name, err)
                }*/
                return
        }
        if project, _ := l.loaded[found]; project == nil {
                err = scanner.Errorf(pos, "unloaded package %v (%v)\n", name, found)
        } else if project.name != name {
                err = scanner.Errorf(pos, "%v: conflicted package name %v (!= %v)\n", found, project.name, name)
        } else {
                info = &packageinfo{ project, packageSmart }
        }
        return
}

func loadPackageConfigInfo(pos token.Position, name string) (info *packageinfo, err error) {
        return
}

// -library finds system library in a way similar to cmake.find_library
//func configureLibrary(pos token.Position, prog *Program, args... Value) (result Value, err error) {
//        return
//}

// -package finds system package in a way similar to cmake.find_package
func configurePackage(pos token.Position, prog *Program, args... Value) (result Value, err error) {
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
                                        err = scanner.Errorf(pos, "package: `%v` unknown type\n", val)
                                        return
                                }
                        default:
                                fmt.Fprintf(os.Stderr, "%v: package: `%v` unknown option\n", key)
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
                                fmt.Fprintf(os.Stderr, "%v: package `%v`: unknown type\n", name)
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

func configureEntry(pos token.Position, prog *Program, s string, params... Value) (configured bool, result Value, err error) {
        var res []Value
        var entry *RuleEntry
        if entry, err = configuration.project.resolveEntry("-"+s); err != nil {
                err = scanner.Errorf(pos, "resolve %v: %v", s, err)
        } else if entry == nil {
                err = scanner.Errorf(pos, "unknown configuration `%v` (no such entry)", s)
        } else if res, err = prog.passExecution(pos, entry, params...); err != nil {
                err = scanner.Errorf(pos, "execute %v: %v", s, err)
        } else {
                if res != nil { result = MakeListOrScalar(res) }
                configured = true
        }
        return
}

func configureArgumented(pos token.Position, prog *Program, target Value, arged *Argumented) (configured bool, result Value, err error) {
        var name Value
        switch val := arged.Val.(type) {
        case *Flag: name = val.Name
        default:
                err = scanner.Errorf(pos, "unknown argumented configuration `%v` (%T)\n", val, val)
                return
        }

        var strName string
        if strName, err = name.Strval(); err != nil { return } else if strName == "" {
                err = fmt.Errorf("`%v` empty configuration (%T)", name, name)
                return
        }

        var params = []Value{ target }
        var fields = map[string]Value{ "name": name }
ForArgs:
        for _, arg := range arged.Args {
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

        var pipe = prog.scope.Lookup("-").(*Def)
        var value = configuration.project.scope.Lookup("VALUE").(*Def)
        if err = value.set(DefSimple, universalnone); err != nil { return }
        if pipe.Value != nil && pipe.Value.Type() != NoneType {
                value.Value = pipe.Value
        }

        var includes = configuration.project.scope.Lookup("INCLUDES").(*Def)
        if err = includes.set(DefSimple, universalnone); err != nil { return }
        if value, ok := fields["include"]; ok || strName == "include" {
                var ( elems []Value; lines []string )
                if strName == "include" && len(params) > 1 {
                        // -include('<xxx.h>')
                        elems = append(elems, params[1])
                }
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

        defer func() {
                if err == nil { configinfon(pos, "… %v", result) } else {
                        switch e := err.(type) {
                        case scanner.Errors: configinfon(pos, "… (%d errors)", len(e))
                        case *scanner.Error: configinfon(pos, "… (error: %v)", e.Err)
                        default: configinfon(pos, "… (error: %v)", err)
                        }
                }
        } ()

        configmessage(pos, strName, fields, arged.Args...)

        if config, ok := configurationOps[strName]; ok {
                if result, err = config(pos, prog, params...); err == nil { configured = true }
        } else {
                configured, result, err = configureEntry(pos, prog, strName, params...)
        }

        if configured && err == nil && result != nil && result.True() && strName != "compiles" {
                if v := pipe.Value; v != nil && v.Type() != NoneType { result = v }
        }
        return 
}

type filewalkFunc func(file *File, err error) error

func walkFileInfos(root string, pats []Value, fn filepath.WalkFunc) (err error) {
        return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
                if err != nil { return err }
        ForPats:
                for _, p := range pats {
                        var matched bool
                        switch pat := p.(type) {
                        case Pattern: //*PercPattern, *GlobPattern, *RegexpPattern
                                if matched, _, err = pat.match(path); err != nil { break ForPats }
                                if !matched {
                                        var s = filepath.Base(path)
                                        if matched, _, err = pat.match(s); err != nil { break ForPats }
                                }
                                if matched {
                                        if err = fn(path, info, err); err != nil { break ForPats }
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
                return fn(&File{
                        Name: rel, //filepath.Base(path),
                        Dir: root, //filepath.Dir(path),
                        Info: info,
                        //Sub: ...
                }, err)
        })
}

// configure-file modifier (see also builtinConfigureFile), example usage:
// 
//     config.h:[(compare) (configure-file)]: config.h.in
//     
func modifierConfigureFile(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = Disclose(args...); err != nil { return }

        var target Value
        for _, arg := range args {
                switch a := arg.(type) {
                case *None, *Flag, *Pair:
                default: if target == nil {
                        target = a
                } else {
                        err = fmt.Errorf("too many configure files")
                        return
                }}
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
func modifierExtractConfiguration(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = Disclose(args...); err != nil { return }

        var target Value
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
                        var (s string; fi os.FileInfo)
                        if s, err = d.Strval(); err != nil {
                                return
                        } else if fi, err = stat(s); err != nil {
                                return
                        } else if fi.IsDir() {
                                err = walkFiles(s, pats, func(file *File, err error) error {
                                        if err == nil {
                                                sources = append(sources, file)
                                        }
                                        return err
                                })
                        } else if a, err = filterValues(pats, false, d); err == nil {
                                sources = append(sources, a...)
                        }
                }
        }

        var exprs = make(map[string]int)
        ForSources: for _, source := range sources {
                var (s string; f *os.File)
                switch t := source.(type) {
                case *File: s = filepath.Join(t.Dir, t.Name)
                default:
                        if s, err = t.Strval(); err != nil {
                                break ForSources
                        }
                }
                if f, err = os.Open(s); err != nil {
                        fmt.Fprintf(os.Stderr, "%v: %v: %v\n", pos, source, err)
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
func modifierConfigure(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = Disclose(args...); err != nil { return }

        var ( target Value; name string )
        if target, err = prog.scope.Lookup("@").(*Def).Call(pos); err != nil { return }
        if name, err = target.Strval(); err != nil { return }

        var def, alt = prog.project.scope.Def(prog.project, name, nil)
        if alt != nil {
                if def, _ = alt.(*Def); def != nil {
                        if def.Value != nil { // if it's already configured
                                // reconfigure the def or return it
                                if optionReconfig {
                                        def.set(DefSimple, universalnone)
                                } else {
                                        return def, nil
                                }
                        }
                }
        }
        if def != nil { result = def } else {
                err = fmt.Errorf("`%s` configuration undefined", name)
                return
        }

        if done, found := configuration.done[def]; done && found { return }
        if err == nil && len(args) == 0 {
                var value Value
                if value, err = prog.scope.Lookup("-").(*Def).Call(pos); err == nil && value != nil {
                        if value.Type() == NoneType {
                                err = def.set(DefExecute, nil)
                        } else {
                                err = def.set(DefExpand, value)
                        }
                }
                return
        }

        if err = def.set(DefExecute, nil); err != nil { return }

        ForConfig: for _, arg := range args {
                switch a := arg.(type) {
                case *Argumented:
                        var ( value Value; configured bool )
                        if configured, value, err = configureArgumented(pos, prog, target, a); err != nil { break ForConfig } else if configured {
                                // marking done (needed for reconfiguring)
                                configuration.done[def] = true
                                if value == nil {
                                        // FIXME: should use `append`, rather then `set`
                                        if err = def.set(DefExecute, nil); err != nil { return }
                                } else {
                                        if err = def.append(value); err != nil { return }
                                }
                        }
                case *Pair:
                        var value Value
                        if value, err = configurePair(pos, prog, a.Key, a.Value); err != nil { break ForConfig } else if value != nil {
                                //def.Append(value)
                        }
                case *Flag:
                        var s string
                        if s, err = a.Name.Strval(); err != nil { break ForConfig }
                        switch s {
                        case "check":
                                
                        default:
                                err = scanner.Errorf(pos, "unknown configuration `-%v`\n", a.Name)
                                break ForConfig
                        }
                default:
                        err = scanner.Errorf(pos, ") unknown configuration `%v` (%T)\n", a, a)
                        break ForConfig
                }
        }
        return
}

const configurationInitFile = `project ~ (-nodock -final)
SHELL := shell -s
CC := gcc
CFLAGS :=
LDFLAGS :=
LOADLIBES :=
LIBS :=
LANG := c++
INCLUDES :=
VALUE :=
-include:[((TARGET)) (unclose) (cd -s &/) | ($(SHELL)) (check -a status=0)] : &(CTD)/check/$(TARGET).$(LANG).include
	@$(CC) -x$(LANG) $(CFLAGS) $(LDFLAGS) $< $(LOADLIBES) $(LIBS) -o &(CTD)/check.out
-symbol:[((TARGET SYMBOL)) (unclose) (cd -s &/) | ($(SHELL)) (check -a status=0)] : &(CTD)/check/$(TARGET).symbol($(SYMBOL))
	@$(CC) -x$(LANG) $(CFLAGS) $(LDFLAGS) $< $(LOADLIBES) $(LIBS) -o &(CTD)/check.out
-function:[((TARGET FUNCTION)) (unclose) (cd -s &/) | ($(SHELL)) (check -a status=0)] : &(CTD)/check/$(TARGET).function($(FUNCTION))
	@$(CC) -x$(LANG) $(CFLAGS) $(LDFLAGS) $< $(LOADLIBES) $(LIBS) -o &(CTD)/check.out
-library:[((TARGET LIBRARY FUNCTION)) (unclose) (cd -s &/) | ($(SHELL)) (check -a status=0)] : &(CTD)/check/$(TARGET).function($(FUNCTION))
	@$(CC) -x$(LANG) $(CFLAGS) $(LDFLAGS) $< $(LOADLIBES) $(LIBS) -l$(LIBRARY) -o &(CTD)/check.out
-struct-member:[((TARGET STRUCT MEMBER)) (unclose) (cd -s &/) | ($(SHELL)) (check -a status=0)] : &(CTD)/check/$(TARGET).structmember($(STRUCT),$(MEMBER))
	@$(CC) -x$(LANG) $(CFLAGS) $(LDFLAGS) $< $(LOADLIBES) $(LIBS) -o &(CTD)/check.out
-sizeof:[((TARGET TYPE)) (unclose) (cd -s &/) | ($(SHELL)) (check -a status=0)] : &(CTD)/check/$(TARGET).sizeof($(TYPE))
	@$(CC) -x$(LANG) $(CFLAGS) $(LDFLAGS) $< $(LOADLIBES) $(LIBS) -o &(CTD)/check.out
-compiles:[((TARGET)) (unclose) (cd -s &/) | ($(SHELL)) (check -a status=0)] : &(CTD)/check/$(TARGET).$(LANG)
	@$(CC) -x$(LANG) $(CFLAGS) $(LDFLAGS) $< $(LOADLIBES) $(LIBS) -o &(CTD)/check.out

%.c.include:[(unclose) (cd -s &/) | (plain c) (update-file -sp)]
	$(INCLUDES)
	#ifdef __CLASSIC_C__
	int main() { return 0; }
	#else
	int main(void) { return 0; }
	#endif
	
%.c++.include:[(unclose) (cd -s &/) | (plain c++) (update-file -sp)]
	$(INCLUDES)
	int main() { return 0; }
	
%.symbol:[((SYMBOL)) (unclose) (cd -s &/) | (plain text) (update-file -sp)]
	$(INCLUDES)
	int main(int argc, char** argv)
	{
	  (void)argv;
	#ifndef $(SYMBOL)
	  return ((int*)(\&$(SYMBOL)))[argc];
	#else
	  (void)argc;
	  return 0;
	#endif
	}
	
%.variable:[((VARIABLE)) (unclose) (cd -s &/) | (plain text) (update-file -sp)]
	$(INCLUDES)
	extern int $(VARIABLE)
	#ifdef __CLASSIC_C__
	int main()
	#else
	int main(int argc, char** argv)
	#endif
	{ (void)argv; return $(VARIABLE); }
	
%.function:[((FUNCTION)) (unclose) (cd -s &/) | (plain text) (update-file -sp)]
	$(INCLUDES)
	#ifdef __cplusplus
	extern "C"
	#endif
	char $(FUNCTION)(void);
	#ifdef __CLASSIC_C__
	int main()
	#else
	int main(int ac, char* av[])
	#endif
	{ $(FUNCTION)(); return 0; }
	
%.structmember:[((STRUCT MEMBER)) (unclose) (cd -s &/) | (plain text) (update-file -sp)]
	$(INCLUDES)
	int main() { (void)sizeof((($(STRUCT) *)0)->$(MEMBER)); return 0; }
	
%.sizeof:[((TYPE)) (unclose) (cd -s &/) | (plain text) (update-file -sp)]
	#undef ARCH
	#if defined(__i386)
	#   define ARCH "__i386"
	#elif defined(__x86_64)
	#   define ARCH "__x86_64"
	#elif defined(__ppc__)
	#   define ARCH "__ppc__"
	#elif defined(__ppc64__)
	#   define ARCH "__ppc64__"
	#elif defined(__aarch64__)
	#   define ARCH "__aarch64__"
	#elif defined(__ARM_ARCH_7A__)
	#   define ARCH "__ARM_ARCH_7A__"
	#elif defined(__ARM_ARCH_7S__)
	#   define ARCH "__ARM_ARCH_7S__"
	#endif
	#define SIZE (sizeof($(TYPE)))
	#ifdef __CLASSIC_C__
	int main(argc, argv) int argc; char *argv[];
	#else
	int main(int argc, char *argv[])
	#endif
	{ (void)argv; return SIZE; }
	
&(CTD)/check/pthreads.c:[(unclose) (cd -s &/) | (plain c) (update-file -sp)]
	#include <pthread.h>
	void* routine(void* args) { return args; }
	int main(void) {
	  pthread_t t;
	  pthread_create(\&t, routine, 0);
	  pthread_join(t, 0);
	  return 0;
	}
	
%.c:[(unclose) (cd -s &/) | (plain c) (update-file -sp)]
	$(VALUE)
	
%.c++:[(unclose) (cd -s &/) | (plain c++) (update-file -sp)]
	$(VALUE)
	
`
