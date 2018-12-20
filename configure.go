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
}{
        fset: token.NewFileSet(),
        libraries: make(map[string]*libraryinfo),
        packages: make(map[string]*packageinfo),
}

var configurationOps = map[string] func(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        "if":           configureIf,
        "include":      configureInclude,
        "option":       configureOption,
        "package":      configurePackage,
}

func init_configuration(paths searchlist) {
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
        if err := l.loadFile(filename, configurationInitFile); err != nil {
                if p, ok := err.(scanner.Errors); ok {
                        fmt.Fprintf(os.Stderr, "%v\n", p)
                } else {
                        fmt.Fprintf(os.Stderr, "configuration: %v\n", err)
                }
        } else if project, ok := l.loaded[filename]; ok {
                configuration.project = project
        } else {
                fmt.Fprintf(os.Stderr, "configuration: `%v` not loaded\n", filename)
        }

        if configuration.project == nil {
                panic("configuration.project still nil")
        }
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
                if n == 3 {
                        configinfox(pos, fields, params[2])
                } else if n == 4 {
                        s := fmt.Sprintf("%v(%v)", params[2], params[3])
                        configinfox(pos, fields, &String{s})
                } else {
                        configinfox(pos, fields, params[0])
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

func configurePair(pos token.Position, prog *Program, value, key, val Value) (result Value, err error) {
        switch k := key.(type) {
        case *Bareword:
                def, alt := prog.project.scope.Def(prog.project, k.string, universalnone)
                if alt != nil {
                        if p, _ := alt.(*Def); p != nil && p != def { def = p }
                }
                if val, err = val.expand(expandDelegate); err == nil {
                        def.Assign(val)
                }
        default:
                fmt.Fprintf(os.Stderr, "%v: unknown configuration `%v` (%T)\n", pos, key, key)
        }
        return
}

func configureIf(pos token.Position, prog *Program, args... Value) (result Value, err error) {
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
                        value = &List{Elements{elems}}
                } else if s, err = value.Strval(); err == nil {
                        value = &String{fmt.Sprintf("#include %s\n", s)}
                } else {
                        return
                }
                includes.Append(value)
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

func loadPackageSmartInfo(pos token.Position, name string) (info *packageinfo) {
        var found string
        for _, path := range configuration.paths {
                s := filepath.Join(path, name + ".smart")
                if fi, err := os.Stat(s); err == nil && fi != nil {
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
        if err := l.loadFile(found, nil); err != nil {
                if p, ok := err.(scanner.Errors); ok {
                        fmt.Fprintf(os.Stderr, "%v\n", p)
                        fmt.Fprintf(os.Stderr, "%v: `%v` package not loaded\n", pos, name)
                } else {
                        fmt.Fprintf(os.Stderr, "%v: package %v: %v\n", pos, name, err)
                }
                return
        }
        if project, _ := l.loaded[found]; project == nil {
                fmt.Fprintf(os.Stderr, "%v: unloaded package %v (%v)\n", pos, name, found)
        } else if project.name != name {
                fmt.Fprintf(os.Stderr, "%v:1: conflicted package name %v (!= %v)\n", found, project.name, name)
        } else {
                info = &packageinfo{ project, packageSmart }
        }
        return
}

func loadPackageConfigInfo(pos token.Position, name string) (info *packageinfo) {
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
                                        fmt.Fprintf(os.Stderr, "%v: package: `%v` unknown type\n", val)
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
                                info = loadPackageSmartInfo(pos, name)
                        case packageConfig:
                                info = loadPackageConfigInfo(pos, name)
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
                fmt.Fprintf(os.Stderr, "\n%v: %v: %v\n", pos, s, err)
        } else if entry == nil {
                fmt.Fprintf(os.Stderr, "\n%v: `%v` unknown configuration\n", pos, s)
        } else if res, err = prog.passExecution(pos, entry, params...); err != nil {
                fmt.Fprintf(os.Stderr, "\n%v: %v: %v\n", pos, s, err)
        } else {
                if res != nil { result = MakeListOrScalar(res) }
                configured = true
        }
        return
}

func configureArged(pos token.Position, prog *Program, target, value, name Value, args... Value) (configured bool, result Value, err error) {
        var fields = map[string]Value{ "name": name }

        var s string
        if s, err = name.Strval(); err != nil { return } else if s == "" {
                err = fmt.Errorf("`%v` empty configuration (%T)", name, name)
                return
        }

        var params = []Value{ target, value }
        ForArgs: for _, arg := range args {
                if list := arg.(*List); list != nil && list.Len() > 0 {
                        var key string
                        switch t := list.Elems[0].(type) {
                        case *Pair:
                                if key, err = t.Key.Strval(); err != nil { return }
                                key = strings.ToLower(key)
                                if v, ok := fields[key]; ok {
                                        fields[key] = &List{Elements{merge(v, t.Value)}}
                                } else {
                                        fields[key] = t.Value
                                }
                                continue ForArgs
                        case *String:if s == "option" {
                                key = "info"
                                if v, ok := fields[key]; ok {
                                        fields[key] = &List{Elements{merge(v, t)}}
                                } else {
                                        fields[key] = t
                                }
                                continue ForArgs
                        }}
                }
                params = append(params, arg)
        }

        var includes = configuration.project.scope.Lookup("INCLUDES").(*Def)
        if value, ok := fields["include"]; ok {
                var s string
                if list, ok := value.(*List); ok {
                        for i, elem := range list.Elems {
                                if s, err = elem.Strval(); err != nil { return }
                                list.Elems[i] = &String{fmt.Sprintf("#include %s\n", s)}
                        }
                } else if s, err = value.Strval(); err == nil {
                        value = &String{fmt.Sprintf("#include %s\n", s)}
                } else {
                        return
                }
                includes.Assign(value)
        } else {
                includes.Assign(universalnone)
        }

        defer func() { if err == nil { configinfon(pos, "… %v", result) }} ()
        configmessage(pos, s, fields, args...)

        if config, ok := configurationOps[s]; ok {
                result, err = config(pos, prog, params...)
                if err == nil { configured = true }
                return
        }

        var vals = append([]Value{params[0]}, params[1:]...)
        return configureEntry(pos, prog, s, vals...)
}

type filewalkFunc func(file *File, err error) error

func walkFileInfos(root string, pats []Value, fn filepath.WalkFunc) (err error) {
        return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
                if err != nil { return err }
                ForPats: for _, p := range pats {
                        var matched bool
                        switch pat := p.(type) {
                        case Pattern: //*PercPattern, *GlobPattern, *RegexpPattern
                                if matched, _, err = pat.match(path); err != nil {
                                        break ForPats
                                } else if !matched {
                                        var s = filepath.Base(path)
                                        if matched, _, err = pat.match(s); err != nil {
                                                break ForPats
                                        }
                                }
                                if matched {
                                        if err = fn(path, info, err); err != nil {
                                                break ForPats
                                        }
                                }
                        default:
                                var s string
                                if s, err = p.Strval(); err != nil {
                                        break ForPats
                                } else if path == s || filepath.Base(path) == s {
                                        if err = fn(path, info, err); err != nil {
                                                break ForPats
                                        }
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
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        }

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
        var target Value
        if target, err = prog.scope.Lookup("@").(*Def).Call(pos); err != nil {
                return
        }

        var (depends []Value; val Value)
        if val, err = prog.scope.Lookup("^").(*Def).Call(pos); err != nil {
                return
        } else if depends, err = mergeresult(ExpandAll(val)); err != nil {
                return
        }

        val = nil // clear

        var optPath bool
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        }

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

        var out *bufio.Writer
        var f *os.File
        f, err = os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, optPerm)
        if err != nil {
                return
        } else {
                out = bufio.NewWriter(f)
        }
        defer func() {
                out.Flush()
                f.Close()
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
                        } else if fi, err = os.Stat(s); err != nil {
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

        exprs := make(map[string]int)
        ForSource: for _, source := range sources {
                var (s string; f *os.File)
                switch t := source.(type) {
                case *File: s = filepath.Join(t.Dir, t.Name)
                default:
                        if s, err = t.Strval(); err != nil {
                                break ForSource
                        }
                }
                if f, err = os.Open(s); err != nil {
                        fmt.Fprintf(os.Stderr, "%v: %v: %v\n", pos, source, err)
                        continue ForSource
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
        fmt.Fprintf(out, "%s:\\\n", optTarget)
        for _, k := range keys {
                fmt.Fprintf(out, "  %s \\\n", k)
        }
        fmt.Fprintf(out, "\n")
        return
}

// configure - configures a variable, example usage:
func modifierConfigure(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        var (name string; target Value)
        if target, err = prog.scope.Lookup("@").(*Def).Call(pos); err != nil {
                return
        } else if name, err = target.Strval(); err != nil { return }

        var def, alt = prog.project.scope.Def(prog.project, name, universalnone)
        if alt != nil { // && p != def
                // it's already configured, just return the value
                if p, _ := alt.(*Def); p != nil { result = p.Value }
                return
        }

        var value Value
        if value, err = prog.scope.Lookup("-").(*Def).Call(pos); err != nil { return }

        var configured bool
        for _, arg := range args {
                switch a := arg.(type) {
                case *Argumented:
                        switch key := a.Val.(type) {
                        case *Flag:
                                if configured, result, err = configureArged(pos, prog, target, value, key.Name, a.Args...); err != nil { return }
                                if result == nil { result = value }
                                if configured { def.Append(result) }
                        default:
                                fmt.Fprintf(os.Stderr, "%v: unknown argumented configuration `%v` (%T)\n", pos, a.Val, a.Val)
                        }
                case *Pair:
                        if _, err = configurePair(pos, prog, value, a.Key, a.Value); err != nil { return }
                default:
                        fmt.Fprintf(os.Stderr, "%v: unknown configuration `%v` (%T)\n", pos, a, a)
                }
        }
        if err == nil {
                if !configured && value != nil {
                        def.Append(value)
                        result = def.Value
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
-include:[((TARGET VALUE)) (unclose) (cd &/) | ($(SHELL)) (check -a status=0)] : &(CTD)/check/$(TARGET).$(LANG).include($(VALUE))
	@$(CC) -x$(LANG) $(CFLAGS) $(LDFLAGS) $< $(LOADLIBES) $(LIBS) -o &(CTD)/check.out
-symbol:[((TARGET VALUE SYMBOL)) (unclose) (cd &/) | ($(SHELL)) (check -a status=0)] : &(CTD)/check/$(TARGET).symbol($(VALUE),$(SYMBOL))
	@$(CC) -x$(LANG) $(CFLAGS) $(LDFLAGS) $< $(LOADLIBES) $(LIBS) -o &(CTD)/check.out
-function:[((TARGET VALUE FUNCTION)) (unclose) (cd &/) | ($(SHELL)) (check -a status=0)] : &(CTD)/check/$(TARGET).function($(VALUE),$(FUNCTION))
	@$(CC) -x$(LANG) $(CFLAGS) $(LDFLAGS) $< $(LOADLIBES) $(LIBS) -o &(CTD)/check.out
-library:[((TARGET VALUE LIBRARY FUNCTION)) (unclose) (cd &/) | ($(SHELL)) (check -a status=0)] : &(CTD)/check/$(TARGET).function($(VALUE),$(FUNCTION))
	@$(CC) -x$(LANG) $(CFLAGS) $(LDFLAGS) $< $(LOADLIBES) $(LIBS) -l$(LIBRARY) -o &(CTD)/check.out
-struct-member:[((TARGET VALUE STRUCT MEMBER)) (unclose) (cd &/) | ($(SHELL)) (check -a status=0)] : &(CTD)/check/$(TARGET).structmember($(VALUE),$(STRUCT),$(MEMBER))
	@$(CC) -x$(LANG) $(CFLAGS) $(LDFLAGS) $< $(LOADLIBES) $(LIBS) -o &(CTD)/check.out
-sizeof:[((TARGET VALUE TYPE)) (unclose) (cd &/) | ($(SHELL)) (check -a status=0)] : &(CTD)/check/$(TARGET).sizeof($(VALUE),$(TYPE))
	@$(CC) -x$(LANG) $(CFLAGS) $(LDFLAGS) $< $(LOADLIBES) $(LIBS) -o &(CTD)/check.out
-compiles:[((TARGET VALUE)) (unclose) (cd &/) | ($(SHELL)) (check -a status=0)] : &(CTD)/check/$(TARGET).$(LANG)($(VALUE))
	@$(CC) -x$(LANG) $(CFLAGS) $(LDFLAGS) $< $(LOADLIBES) $(LIBS) -o &(CTD)/check.out

%.c.include:[((VALUE)) (unclose) (cd &/) | (plain c) (update-file -sp)]
	$(INCLUDES)
	#ifdef __CLASSIC_C__
	int main() { return 0; }
	#else
	int main(void) { return 0; }
	#endif
	
%.c++.include:[((VALUE)) (unclose) (cd &/) | (plain c++) (update-file -sp)]
	$(INCLUDES)
	int main() { return 0; }
	
%.symbol:[((VALUE SYMBOL)) (unclose) (cd &/) | (plain text) (update-file -sp)]
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
	
%.variable:[((VALUE VARIABLE)) (unclose) (cd &/) | (plain text) (update-file -sp)]
	$(INCLUDES)
	extern int $(VARIABLE)
	#ifdef __CLASSIC_C__
	int main()
	#else
	int main(int argc, char** argv)
	#endif
	{ (void)argv; return $(VARIABLE); }
	
%.function:[((VALUE FUNCTION)) (unclose) (cd &/) | (plain text) (update-file -sp)]
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
	
%.structmember:[((VALUE STRUCT MEMBER)) (unclose) (cd &/) | (plain text) (update-file -sp)]
	$(INCLUDES)
	int main() { (void)sizeof((($(STRUCT) *)0)->$(MEMBER)); return 0; }
	
%.sizeof:[((VALUE TYPE)) (unclose) (cd &/) | (plain text) (update-file -sp)]
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
	
&(CTD)/check/pthreads.c:[((VALUE)) (unclose) (cd &/) | (plain c) (update-file -sp)]
	#include <pthread.h>
	void* routine(void* args) { return args; }
	int main(void) {
	  pthread_t t;
	  pthread_create(\&t, routine, 0);
	  pthread_join(t, 0);
	  return 0;
	}
	
%.c:[((VALUE)) (unclose) (cd &/) | (plain c) (update-file -sp)]
	$(VALUE)
	
%.c++:[((VALUE)) (unclose) (cd &/) | (plain c++) (update-file -sp)]
	$(VALUE)
	
`
