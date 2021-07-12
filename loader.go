//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "extbit.io/smart/token"
    "extbit.io/smart/scanner"
    "bytes"
    "io/ioutil"
    "io"
    "unicode/utf8"
    "path/filepath"
    "strings"
    "plugin"
    "errors"
    "time"
    "flag"
    "fmt"
    "os/exec"
    "os"
)

const optSortErrors = false

type ResolveBits int
const (
    // If many bits are set, resolve in the listed priority.
    FromGlobe ResolveBits = 1<<iota
    FromBase
    FromProject
    FindDef
    FindRule

    FromHere

    // This is the default be
    anywhere = FromHere
    global = FromGlobe
    local = FromProject
    nonlocal = FromGlobe | FromBase | FromProject
)

type EvalBits int
const (
    KeepClosures EvalBits = 1<<iota
    KeepDelegates

    // Wants value for rule depends.
    DependValue

    // Wants v.Strval(), expands delegates and closures,
    // turn off KeepClosures, KeepDelegates.
    StringValue = 0
)

// A Mode value is a set of flags (or 0).
// They control the amount of source code parsed and other optional
// parser functionality.
//
type Mode uint

const (
    ModuleClauseOnly Mode = 1 << iota // stop parsing after project or module clause
    ImportsOnly               // stop parsing after import declarations
    ParseComments             // parse comments and add them to AST
    Flat                  // parsing in flat mode (donot create a new module)
    //Trace                 // print a trace of parsed productions
    DeclarationErrors         // report declaration errors
    SpuriousErrors            // same as AllErrors, for backward-compatibility
    //AllErrors = SpuriousErrors    // report all errors (not just the first 10 on different lines)
    parsingDir
)

const (
    dotContainer = ".container"
    dotConfigure = ".configure"
)

var parseMode = DeclarationErrors //|Trace

type searchlist []string

func (sl *searchlist) String() string {
    return fmt.Sprint(*sl)
}

func (sl *searchlist) Set(value string) error {
    *sl = append(*sl, strings.Split(value, ",")...)
    return nil
}

var globalPaths searchlist

func init() {
    flag.Var(&globalPaths, "search", "comma-separated list of search paths")
}

type declare struct {
    project *Project
    backproj *Project
    backscope *Scope
    useesExecuted []*Project
}

type loadinfo struct {
    absDir string // absPath = filepath.Join(absDir, baseName)
    baseName string
    specName string
    useesExecuted []*Project
    loader *Project
    loadee *Project // the current loading project
    scope *Scope
    declares map[string]*declare // all projects declared in the load dir
}

func (li *loadinfo) absPath() string {
    return filepath.Join(li.absDir, li.baseName)
}

func (li *loadinfo) breakUseLoop() (result bool) {
    var first bool = true
    for _, decl := range li.declares {
        if first || result {
            result = decl.project.opts.breakUseLoop
        }
        if !result { break }
        first = false
    }
    return
}

type loaderScope struct {
    cc closurecontext
    scope *Scope
}

type loader struct {
    *Context
    *parser
    mode Mode // parsing mode
    fset     *token.FileSet
    paths    searchlist
    loadArgs []Value
    loads    []*loadinfo
    loaded   map[string]*Project // loaded projects
    loadStack []*Project // load path
    useStack []*Project // use path
    useesExecuted []*Project // all executed usees
    project  *Project // the current project
    scope    *Scope   // the current scope
    vs string // verbose prefix
    isLoadingBases bool
}

func (l *loader) error(pos token.Pos, f string, a... interface{}) {
    var pp Position
    if l.parser != nil {
        pp = Position(l.parser.file.Position(pos))
    }
    diag.errorAt(pp, f, a...)
}

func (i *loader) AddSearchPaths(paths... string) (err error) {
    for _, s := range paths {
        if s, err = filepath.Abs(s); err != nil {
            break
        }
        if fi, _ := os.Stat(s); fi != nil && fi.IsDir() {
            i.paths = append(i.paths, s)
        } else {
            return errors.New(fmt.Sprintf("path '%s' is not dir", s))
        }
    }
    return nil
}

func restoreLoadingInfo(l *loader) {
    var (
        last = len(l.loads)-1
        linfo = l.loads[last]
    )

    l.loads = l.loads[0:last]
    l.useesExecuted = linfo.useesExecuted
    l.project = linfo.loader
    l.scope = linfo.scope //l.SetScope(linfo.scope)

    /*var names []string
    for _, declare := range linfo.declares {
        names = append(names, declare.project.Name())
    }

    if loader := linfo.loader; loader != nil {
        fmt.Fprintf(stderr, "exit: %v from '%s' → %v\n", names, loader.Name(), linfo.scope)
    } else {
        fmt.Fprintf(stderr, "exit: %v → %v\n", names, linfo.scope)
    } */
}

func saveLoadingInfo(l *loader, specName, absDir, baseName string) *loader {
    l.loads = append(l.loads, &loadinfo{
        absDir: absDir,
        baseName: baseName,
        specName: filepath.Clean(specName),
        useesExecuted: l.useesExecuted,
        loader:   l.project,
        scope:    l.scope, //Scope(),
        declares: make(map[string]*declare),
    })
    return l
}

func (l *loader) searchSpecPath(linfo *loadinfo, specName string) (absPath string, isDir bool, err error) {
    var fi os.FileInfo
    if specName == "." {
        err = fmt.Errorf("Not possible to chain itself.")
    } else if abs := filepath.IsAbs(specName); abs ||
        specName == "~" || specName == ".." ||
        strings.HasPrefix(specName, "~\\") ||
        strings.HasPrefix(specName, "~/") ||
        strings.HasPrefix(specName, "./") ||
        strings.HasPrefix(specName, "../") ||
        strings.HasPrefix(specName, ".\\") ||
        strings.HasPrefix(specName, "..\\") {
        var (
            s = specName
            sx string
        )
        if !abs && linfo.absDir != "" {
            sx = filepath.Join(linfo.absDir, s)
            if a, e := filepath.Abs(sx); e == nil {
                s = a
            } else {
                err = e
                return
            }
        }
        if fi, err = os.Stat(s); err != nil {
            sx = s + ".smart"
            if fi, er := os.Stat(sx); fi != nil {
                isDir, absPath, err = fi.IsDir(), sx, er
                return
            }
            sx = s + ".sm"
            if fi, er := os.Stat(sx); fi != nil {
                isDir, absPath, err = fi.IsDir(), sx, er
                return
            }
        } else {
            isDir, absPath = fi.IsDir(), s
        }
    } else {
        for _, base := range l.paths {
            var s string
            if filepath.IsAbs(base) {
                s = filepath.Join(base, specName)
            } else {
                s = filepath.Join(l.workdir, base, specName)
            }
            if fi, err = os.Stat(s); err == nil && fi != nil {
                isDir, absPath = fi.IsDir(), s
                return
            }
        }
    }
    return
}

type genericoptions struct {
    keyword token.Token
    verbose bool // TODO: verbose operation
    dontOperate bool
    options []Value
}

type useoptions struct {
    allowReuse bool
    noFiles bool
}

type importoptions struct {
    useoptions
}

type importspecoptions struct {
    unuse bool
    reuse bool
    noFiles bool
}

func (l *loader) loadUseSpecName(opts importoptions, specVal Value, specName string, specOpts *importspecoptions, params []Value) {
    var (
        linfo = l.loads[len(l.loads)-1]
        err error
    )

    var position = specVal.Position()
    var ( absPath string; isDir bool )
    if absPath, isDir, err = l.searchSpecPath(linfo, specName); err != nil {
        diag.errorOf(specVal, "no such package `%v`", specName)
        return
    } else if absPath == "" {
        diag.errorOf(specVal, "missing `%s` (in %v)", specName, l.paths)
        return
    }

    var loaded, loadedValid = l.loaded[absPath]

    // Checking circular loads. See also Project.loopImportPath()!
    var breakUseLoop bool
    for i, load := range l.loads {
        if load.absDir == absPath {
            var s string
            var loop, loopBreakers []*loadinfo
            for n := i; n < len(l.loads); n += 1 {
                var load = l.loads[n]
                loop = append(loop, load)
                if load.breakUseLoop() {
                    loopBreakers = append(loopBreakers, load)
                    s += "<" + load.specName + "> → "
                } else {
                    s += load.specName + " → "
                }
            }
            if loadedValid && loaded.opts.breakUseLoop {
                s += "<" + specName + ">"
            } else {
                s += specName
            }

            breakUseLoop = (loopBreakers != nil)
            if !breakUseLoop {
                diag.errorOf(specVal, "loop detected: %s", s)
            } else if optionVerboseImport || optionVerboseUsing || optionVerboseLoading {
                fmt.Fprintf(stderr, "%s: loop detected: %v\n", l.project, s)
            }
        }
    }

    /*if breakUseLoop {
        if loadedValid {
            _, a := l.project.scope.ProjectName(l.project, loaded.Name(), loaded)
            if a != nil {
                if v, ok := a.(*ProjectName); !ok || v == nil {
                    err = fmt.Errorf("`%s' name already taken (%T).", loaded.Name(), a)
                }
            }
        }
        return
    }*/

    defer func(a []*Project) { l.loadStack = a } (l.loadStack)
    l.loadStack = append(l.loadStack, l.project) // build the load path

    // https://unicode-table.com/en/sets/arrows-symbols/
    // ┌────────────────────────────────┐
    // ├────────────────────────────────┼───┬──⇢·
    // ├──────────────────────┬────→┬←──┤   │    ⇡
    // ├┬─→───────────────────┼─────┴───┘   ├────┼⇢
    // │├┬───→         ↑  └──┬──┐       │    ⇣
    // ││└──→    ·     │     │  ├─⇥     ↓
    // │└──→───⇥─┴─⇤────┬──┴──┬──┘  │
    // └──→       ⇠─┘     ↓     └─→ ⇒ …
    if optionVerboseImport {
        if len(l.loadStack) > 1 {
            defer func(s string) { l.vs = s } (l.vs)
            l.vs += "│"
        }
        if specOpts.reuse {
            fmt.Fprintf(stderr, "%s├┬→\"%s\" (reuse, %s)\n", l.vs, specName, absPath)
        } else {
            fmt.Fprintf(stderr, "%s├┬→\"%s\" (%s)\n", l.vs, specName, absPath)
        }
        defer func(t time.Time) {
            var name string
            var d = time.Now().Sub(t)//*time.Millisecond // µs, ms, s
            var ds = fmt.Sprintf("(%s)", d)
            if d>=1*time.Second { ds = fmt.Sprintf("▶%s◀",ds) }
            if loaded != nil { name = loaded.name }
            fmt.Fprintf(stderr, "%s├┴─\"%s\" ⇢ %s %s\n", l.vs, specName, name, ds)
        } (time.Now())
    }

    if loadedValid && !specOpts.reuse {
        var ( proj *Project ; res, isb bool )
        if proj, res, isb, err = l.project.hasLoaded(loaded, breakUseLoop); err != nil {
            diag.errorOf(specVal, "`%s`: %s", specName, err)
            return
        } else if isb {
            diag.errorOf(specVal, "`%s` is a base (%s)", specName, proj.name)
            return
        } else if res {
            diag.errorOf(specVal, "'%s' already imported by '%s'", specName, proj.name)
            return
        }
    }

    if /*!loadedValid || loaded == nil*/true {
        var okay bool
        if isDir {
            okay = l.loadDir(position, specName, absPath, nil)
        } else {
            okay = l.load(specName, absPath, nil)
        }
        if !okay {
            diag.errorOf(specVal, "failed loading `%v` (%v)", specName, absPath)
            return
        }

        if loaded != nil {
            // already loaded previously
        } else if loaded, loadedValid = l.loaded[absPath]; loadedValid {
            // successfully loaded (first)
        } else {
            diag.errorOf(specVal, "'%s' not loaded (%s)", specName, absPath)
        }

        if loaded == nil {
            diag.errorOf(specVal, "'%s' not smart project", specName)
            return
        }
    }
    if breakUseLoop { /*return*/ }

    // Check against the current load list before appending loaded.
    for _, lp := range l.project.loads {
        var ( proj *Project ; res, isb bool )

        if loaded == lp {
            diag.errorOf(specVal, "using `%s` multiple times", specName)
            return
        }

        if proj, res, isb, err = loaded.hasLoaded(lp, breakUseLoop); err != nil {
            diag.errorOf(specVal, "%s: %s", specName, err)
            return
        } else if isb {
            if l.project.hasBase(lp) {
                // common bases are fine
            } else {
                diag.errorOf(specVal, "`%s` is already a base", specName)
            }
        } else if res && !lp.opts.multiUseAllowed {
            diag.warnAt(position, "`%s` has already imported `%s` (from %s)", loaded, lp, proj)
        }

        if proj, res, isb, err = lp.hasLoaded(loaded, breakUseLoop); err != nil {
            diag.errorOf(specVal, "%s: %s", specName, err)
            return
        } else if isb {
            diag.warnAt(position, "`%s` is already base of `%s` (%s)", loaded, lp, proj)
        } else if res && !loaded.opts.multiUseAllowed {
            diag.warnAt(position, "`%s` has already been imported by `%s` (from %s)", loaded, lp, proj)
        }
    }

    if specOpts.unuse || breakUseLoop { return } else {
        // The project load list is different from using list.
        l.project.loads = append(l.project.loads, loaded)
    }

    name, _ := l.project.scope.Lookup(loaded.name).(*ProjectName)
    if name == nil {
        diag.errorOf(specVal, "%v (%v,dir=%v) not in %v", specName, absPath, isDir, l.project.scope.comment)
        return
    }

    var useopts = opts.useoptions
    if specOpts.reuse {
        // override reuse option
        useopts.allowReuse = true
    }
    if specOpts.noFiles {
        useopts.noFiles = true
    }

    if optionVerboseImport {
        defer func(t time.Time) {
            var d = time.Now().Sub(t)//*time.Millisecond // µs, ms, s ┼
            fmt.Fprintf(stderr, "%s├┤ %s:import(%s) (%s)\n", l.vs, l.project, specName, d)
        } (time.Now())
    }

    if err = l.useProject(position, loaded, params, useopts); err == nil && !specOpts.unuse {
        var ( using Value; names []string )
        if o, e := l.project.resolveObject("using.*"); e == nil && !isNil(o) {
            if def, ok := o.(*Def); ok && !isNil(def) { using = def.value }
        }
        if !(isNil(using) || isNone(using)) {
            if s, e := using.Strval(); e == nil {
                names = strings.Fields(s)
            }
        }
        for _, nameprops := range names {
            var name, _, _, _ = parseUsingNameProps(nameprops)
            var usingVarName = fmt.Sprintf("using.%s", name)
            var def, alt = l.project.scope.define(l.project, name, MakeNone(position))
            if def != nil && alt == nil { // if it's new Def (first time being defined)
                for _, base := range l.project.bases {
                    if obj, err := base.resolveObject(name); err == nil && !(isNil(obj) || isNone(obj)) {
                        def.append(obj) // append all base Defs (act like '+=')
                    }
                }
            }
            if def == nil && alt != nil { def, _ = alt.(*Def) }
            if o := loaded.scope.Lookup(usingVarName); !isNil(o) {
                def.append(o)
            }
        }
    }
    return
}

const pluginDifferentVersionError = `plugin was built with a different version of package`
var numUpdatedPlugins = 0

func buildPlugin(s, src string) (err error) {
    if optionTraceLaunch { defer un(trace(t_launch, "loader.buildPlugin")) }

    fmt.Fprintf(stderr, "smart: Build %v …", src)
    dir, _ := filepath.Split(src)
    o := &bytes.Buffer{}
    c := exec.Command("go", "build", "-buildmode=plugin", "-o", s)
    c.Stdout, c.Stderr, c.Dir = o, o, dir
    if err = c.Run(); err == nil {
        numUpdatedPlugins += 1
        fmt.Fprintf(stderr, "… ok\n")
        fmt.Fprintf(stderr, "smart: Plugin updated, please relaunch.\n")
        os.Exit(0)
    } else {
        fmt.Fprintf(stderr, "… error\n")
        fmt.Fprintf(stderr, "%s", o)
    }
    return
}

func (l *loader) loadPlugin(pos Position) (err error) {
    if optionTraceLaunch { defer un(trace(t_launch, "loader.loadPlugin")) }

    g := stat(pos, "smart.go", "", l.project.absPath)
    if g == nil { return /* smart.go was not presented */ }

    var src string
    if src, err = g.Strval(); err != nil { return }

    s := strings.Replace(l.project.relPath, "..", "_", -1)
    s = filepath.Join(filepath.Dir(joinTmpPath("", "")), "plugins", s)

    var build = true

    so := stat(pos, /*l.project.name*/"plugin", "", s, nil)
    if s, err = so.Strval(); err != nil { return }
    if exists(so) && !optionAlwaysBuildPlugins {
        if so.info.ModTime().After(g.info.ModTime()) {
            build = false // Plugin already updated.
        }
    }
    if build { err = buildPlugin(s, src) }
    if err != nil { return }

    // Once plugin is opened, there's no need/way to close it.
    if l.project.plugin, err = plugin.Open(s); err == nil {
        var p plugin.Symbol
        if p, err = l.project.plugin.Lookup("Init"); err != nil {
            return
        } else if p == nil {
            // no initialization (optional)
        } else if f, ok := p.(func(Position, *Project) (*Scope, error)); ok {
            l.project.pluginScope, err = f(pos, l.project)
        } else if f, ok := p.(func(*Project) (*Scope, error)); ok {
            l.project.pluginScope, err = f(l.project)
        }
    } else if es := err.Error(); strings.Contains(es, pluginDifferentVersionError) {
        err = buildPlugin(s, src)
    }
    return
}

func (l *loader) convertBarefiles(targets []Value) []Value {
    for i, target := range targets {
        var pos = target.Position()
        switch t := target.(type) {
        case *Bareword:
            if file := l.project.FindFile(t.string); file != nil {
                targets[i] = &Barefile{ Name:target, File:file }
                file.position = pos
            }
        case *Barecomp:
            if t.closured() || t.refdef(DefArg) { break }
            if s, err := t.Strval(); err != nil {
                diag.errorAt(pos, "stringify '%v' failed: %v", t, err)
            } else if file := l.project.FindFile(s); file != nil {
                targets[i] = &Barefile{ Name:target, File:file }
                file.position = pos
            }
        case *Argumented:
            vals := l.convertBarefiles(append([]Value{t.value}, t.args...))
            t.value, t.args = vals[0], vals[1:]
        }
    }
    return targets
}

func (l *loader) useProject(position Position, usee *Project, params []Value, opts useoptions) (err error) {
    if optionVerboseUsing && optionVerboseImport && optionBenchImport {
        defer func(t time.Time) {
            var d = time.Now().Sub(t)
            fmt.Fprintf(stderr, "%s││ using(%8s) %s ⇒ %v\n", l.vs, d, l.project, l.project.using)
        } (time.Now())
    } else if optionVerboseUsing {
        defer func(t time.Time) {
            var d = time.Now().Sub(t)
            fmt.Fprintf(stderr, "using(%8s) %s ⇒ %v\n", d, l.project, l.project.using)
        } (time.Now())
    }
    if err = l.useProject2(position, usee, params, opts); err != nil {
        if p, ok := err.(*scanner.Error); ok {
            diag.errorAt(position, "%v", p.Brief())
        } else {
            diag.errorAt(position, "%v", err)
        }
    }
    return
}

func (l *loader) usePath() (s string) {
    for i, u := range l.useStack {
        if i > 0 { s += "," }
        s += u.name
    }
    return
}

func (l *loader) useProject2(position Position, usee *Project, params []Value, opts useoptions) (err error) {
    if usee == l.project {
        diag.errorAt(position, "'%v' use loop (%s)", usee.name, l.usePath())
        return
    } else if false {
        for _, using := range l.project.using.list {
            if using.project == usee { return }
        }
    } else if l.project.isUsingDirectly(usee) {
        return
    }

    // clocks:🕐🕑🕒🕓🕔🕕🕖🕗🕘🕙🕚🕛🕜🕝🕞🕟🕠🕡🕢🕣🕤🕥🕦🕧
    defer func(t time.Time) {
        var d = time.Now().Sub(t)
        if optionVerboseImport {
            if optionBenchImport /*&& d > 1*time.Millisecond*/ {
                var s = l.usePath()
                fmt.Fprintf(stderr, "%s││ %s:use(%s) … (%s) (%s)\n", l.vs, l.project.name, usee.name, d, s)
            }
        } else if optionBenchSlow && d > 500*time.Millisecond { // ⌚ ⌛
            fmt.Fprintf(stderr, "smart: %s: slow ▶use(%s)◀ … (%s)\n", l.project.name, usee.name, d)
        }
    } (time.Now())

    defer func(a []*Project) { l.useStack = a } (l.useStack)
    l.useStack = append(l.useStack, usee) // build the use path

    // Add to the project using list, so that the use path is correct.
    l.project.using.append(position, usee, params, opts)

    return // :user: rules are deprecated!
}

func (l *loader) determine(position Position, tok token.Token, identifier, value Value) (def *Def) {
    var dbg bool
    var alt Object
    switch t := identifier.(type) {
    case *selection:
        var v, err = t.value()
        if err != nil {
            diag.errorAt(position, "determine `%v`: %v", t, err)
            return
        } else if d, ok := v.(*Def); ok {
            def = d
        } else {
            diag.errorAt(position, "`%v` is not a def (%T)", t, v)
            return
        }

    case *Bareword, *Barecomp, *Qualiword:
        var name, err = t.Strval()
        if err != nil {
            diag.errorAt(position, "determine `%v`: %v", t, err)
            return
        } else if _, ok := builtins[name]; ok {
            diag.errorAt(position, "`%v` (%v) is builtin name", identifier, name)
            return
        }

        // Resolve base value to derive.
        var prev Object
        prev, err = l.project.resolveObject(name)
        if err != nil { diag.errorAt(position, "resolve '%s' failed: %v", name, err) }
        if def, alt = l.def(position, name); alt == nil {
            // does nothing...
        } else if alt != nil && (tok == token.ASSIGN || tok == token.EXC_ASSIGN) {
            var ( okay bool; ad *Def )
            if ad, okay = alt.(*Def); !okay {
                diag.errorAt(position, "`%v` already defined (%T) (%v,%v)", identifier, alt, alt.OwnerProject(), l.project).
                    debug(optionDebugErrors && optionPrintStack)
                return
            } else if ad.owner == l.project && ad.origin != DefConfRef {
                diag.errorAt(position, "`%v` already defined (%T) (%v)", identifier, alt, l.project).
                    debug(optionDebugErrors && optionPrintStack)
                return
            } else {
                def = ad
            }
        } else if alt != nil {
            def = alt.(*Def)
        }

        if prev == nil {
            // no derived value
        } else if prev.OwnerProject() == l.project {
            // not derivable def if in the same project
        } else if derived, okay := prev.(*Def); !okay {
            // not a def
        } else if derived == nil {
            diag.errorAt(position, "def '%s' is nil", name)
        } else if derived == def || def.value.refs(derived) {
            // same def
        } else if tok == token.ADD_ASSIGN {
            // Unshift the delegation to derive value.
            err := def.append(MakeDelegate(position, token.LPAREN, derived))
            if err != nil {
                diag.errorAt(position, "append def '%s' failed: %v", def.name, err)
            }
        }
    }

    if def == nil {
        diag.errorAt(position, "identifier `%v' is nil", identifier)
        return
    }

    // Ensures that all immediate assignments are in the current
    // project context.
    defer setclosure(setclosure(cloctx.unshift(l.scope)))

    def.position = position
    if err := l.assign(tok, def, alt, value); err != nil {
        diag.errorAt(position, "assign '%v' failed: %v", def.name, err)
    } else if dbg {
        s, _ := def.value.Strval()
        diag.infoAt(position, "%v: %v->%v: %v -> %s (%v)", l.project, def.owner, def.name, def.value, s, value).
            debug(dbg)
    }
    return
}

func (l *loader) rule(clause *parsedRuleData) (entries []*RuleEntry) {
    defer setclosure(setclosure(cloctx.unshift(l.project.scope)))

	var ( a = t_traverse.elapsed(); b = a )
	if xxx_debug { defer un(tracef(t_traverse, "rule(%s)", clause.targets)) }

    var (
        params  []*Def
        depends []Value
        ordered []Value
        progScope *Scope = l.scope
        configure = clause.config
        recipes = clause.recipes
    )
    for _, name := range clause.params {
        def := progScope.Lookup(name).(*Def)
        params = append(params, def)
    }
    for _, depend := range clause.depends {
        switch dep := depend.(type) {
        case *List: depends = append(depends, dep.Elems...)
        default:    depends = append(depends, dep)
        }
    }
    for _, depend := range clause.ordered {
        switch dep := depend.(type) {
        case *List: ordered = append(ordered, dep.Elems...)
        default:    ordered = append(ordered, dep)
        }
    }

    var prog = &Program{
        project:  l.project,
        scope:    progScope,
        params:   params,
        depends:  depends,
        ordered:  ordered,
        recipes:  recipes,
        configure: configure,
        position: clause.position,
    }

	if b = t_traverse.elapsed(); xxx_debug { t_traverse.tracef("%v %v", b, (b-a)) }
    for _, target := range clause.targets {
        if target == nil {
            diag.errorOf(target, "nil target (%T)", target)
            return
        }
        var ( name string ; entry *RuleEntry ; err error )
        if name, err = target.Strval(); err != nil {
            diag.errorOf(target, "stringify target '%v' failed: %v", target, err).
                debug(optionDebugErrors)
        }
        if true {// it should work too if not checking against files
            switch target.(type) {
            case *File, *Path, Pattern:
            default:
                var file = l.project.FindFile(name)
                if file != nil {
                    file.position = target.Position()
                    target = file
                }
            }
        }

        //if b = t_traverse.elapsed(); xxx_debug { t_traverse.tracef("%v %v", b, (b-a)) }
        entry, err = l.project.entry(clause.special, clause.options, target, prog)
        //if b = t_traverse.elapsed(); xxx_debug { t_traverse.tracef("%v %v", b, (b-a)) }
        if err != nil {
            diag.errorOf(target, "creating entry '%v' failed: %v", target, err)
            return
        } else /*if entry != nil*/ {
            entry.position = target.Position()
            entries = append(entries, entry)
        }
        if t, okay := entry.target.(*Flag); okay && t != nil {
            var s string
            if s, err = t.name.Strval(); err != nil {
                diag.errorOf(target, "stringify flag target name '%v' failed: %v", t.name, err)
            } else if l.project.name != "~" {
                flags, _ := context.flagEntries[s]
                flags = append(flags, entry)
                context.flagEntries[s] = flags
            }
            //if s == "configure" { configuration.configs = append(configuration.configs, entry) }
        } else if configure {
            configuration.entries = append(configuration.entries, entry)
        }
    }
    return
}

func (l *loader) includeFile(pos Position, spec Value) {
    var (
        linfo = l.loads[len(l.loads)-1]
        specName, fullname string
        err error
    )

    // Execute the rule entry to update include source.
    if entry, ok := spec.(*RuleEntry); ok && entry != nil {
        var ( result []Value; breakers []*breaker )
        if result, breakers = entry.Execute(entry.position); len(breakers) > 0 {
            diag.errorAt(pos, "include error occurred (entry %v)", entry)
            return
        } else if result != nil && optionVerbose {
            diag.infoAt(pos, "include %v: %v", entry, result)
        }
        spec = entry.target
    }

    switch t := spec.(type) {
    case *File:
        if t.info == nil {
            diag.errorAt(pos, "`%v` no source file", t)
            return
        }
        fullname = t.fullname() //filepath.Join(t.dir, t.Name)
        specName = t.name
    default:
        if specName, err = spec.Strval(); err != nil {
            diag.errorAt(pos, "include error occurred (spec %v)", spec)
            return
        }
        if filepath.IsAbs(specName) {
            fullname = specName
        } else {
            fullname = filepath.Join(linfo.absDir, specName)
        }
    }

    if specName == "" {
        diag.errorAt(pos, "`%v` is empty string", spec)
        return
    }

    if false { defer un(trace(t_traverse, "includeFile.ParseFile")) } // xxx_debug

    var absDir, baseName = filepath.Split(fullname)
    defer func(mode Mode) { l.mode = mode } (l.mode) // Must restore parse mode!
    defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, baseName))
    if _, err = l.ParseFile(fullname, nil, parseMode|Flat); err != nil {
        diag.errorAt(pos, "include error occurred (from %v)", fullname)
    }

    _ = diag.checkErrors(true)
    return
}

func (l *loader) openScope(comment string) loaderScope {
    if false && optionTraceLaunch { defer un(trace(t_launch, "loader.openScope")) }
    var pos Position
    if l.parser != nil { pos = l.position() }
    l.scope = NewScope(pos, l.scope, l.project, comment)
    cc := setclosure(cloctx.unshift(l.scope))
    return loaderScope{ cc, l.scope }
}

func (l *loader) closeScope(ls loaderScope) {
    if false && optionTraceLaunch { defer un(trace(t_launch, "loader.closeScope")) }
    if ls.scope != nil {
        l.scope = ls.scope.outer
        if ls.cc != nil { setclosure(ls.cc) }

        // Must change the outer of dir scope to globe to avoid Finding symbols
        // into the wrong context.
        if s := ls.scope.Comment(); strings.HasPrefix(s, "dir ") {
            l.globe.SetScopeOuter(ls.scope)
        }
    }
    return
}

func (l *loader) setArgs(args []Value) (oldArgs []Value) {
    oldArgs = l.loadArgs
    l.loadArgs = args
    return
}

// project example (base(var=value))
func (l *loader) loadBases(position Position, linfo *loadinfo, params ...Value) (result bool) {
    if optionTraceLaunch { defer un(trace(t_launch, "loader.loadBases")) }

    var isDir bool
    var absPath, specName string

    absPath = filepath.Join(l.project.absPath, ".base")
    if fi, e := os.Stat(absPath); e == nil && fi.IsDir() {
        base := MakePathStr(l.project.position, "./.base")
        params = append([]Value{base}, params...)
        absPath = ""
    }

    // For &(foobar) set from loadArgs
    defer setclosure(setclosure(cloctx.unshift(l.project.scope)))

ParamsLoop:
    for _, elem := range params {
        var ( args []Value; err error )
        if list, ok := elem.(*List); ok && len(list.Elems) == 1 { elem = list.Elems[0] }
        if a, ok := elem.(*Argumented); ok { elem, args = a.value, a.args }
        if p, ok := elem.(*Pair); ok {
            var identifier = p.Key
            var position = identifier.Position()
            var name string
            if name, err = p.Key.Strval(); err != nil { diag.errorAt(position, "%v", err); return }
            if len(name) > 0 && name[0] == '.' { identifier = MakeBarecomp(position, MakeBareword(position, "project"), p.Key) }
            var def = l.determine(position, token.ASSIGN, identifier, p.Value)
            if isNil(def) {/* FIXME: ... */}
            continue ParamsLoop
        }

        if specName, err = elem.Strval(); err != nil { diag.errorOf(elem, "%v", err); return }
        if specName == "" {
            diag.errorOf(elem, "%v: empty base name `%v` (%T)", l.project, elem, elem)
            break ParamsLoop
        }
        if absPath, isDir, err = l.searchSpecPath(linfo, specName); err != nil {
            break ParamsLoop
        }

        for _, base := range l.project.bases {
            if base.absPath == absPath {
                //diag.errorOf(elem, "duplicated base: %v (in %v)", elem, l.project.bases)
                continue ParamsLoop
            }
        }

        var okay bool
        if isDir {
            okay = l.loadDirWithArgs(position, specName, absPath, args, nil)
        } else {
            okay = l.loadWithArgs(position, specName, absPath, args, nil)
        }
        if !okay {
            diag.errorOf(elem, "loadBases: '%s' not loaded (%s)", specName, absPath)
            break ParamsLoop
        } else if loaded, yes := l.loaded[absPath]; yes && loaded != nil {
            // chain loaded base project, note that err might not be nil
            l.project.Chain(loaded)
        } else {
            diag.errorOf(elem, "project `%v`(%T: %s) not loaded (%s)", elem, elem, specName, absPath)
            break ParamsLoop
        }
    }
    return true
}

func (l *loader) loadDotContainer(ident *Bareword, file *File) (result bool) {
    if optionTraceLaunch { defer un(trace(t_launch, "loader.loadDotContainer")) }
    var position = ident.Position()
    if !l.loadDir(position, dotContainer, file.fullname(), nil) {
        diag.errorOf(ident, "%s: load '%v' failed (%s)", ident, dotContainer, file.fullname())
    } else if loaded, yes := l.loaded[file.fullname()]; yes && loaded != nil {
        name, _ := l.project.scope.Lookup(loaded.Name()).(*ProjectName)
        if name == nil {
            diag.errorOf(ident, "%v: %v: `dock` is not a project", l.project.name, file)
        } else {
            if optionVerboseLoading { fmt.Fprintf(stderr, "smart: %v (%v)\n", name, file.fullname()) }

            var opts useoptions
            // TODO: parse the useoptions
            l.useProject(position, loaded, nil, opts)

            result = true
        }
    }
    return
}

func (l *loader) loadDotConfigure(ident *Bareword, file *File) (result bool) {
    if optionTraceLaunch { defer un(trace(t_launch, "loader.loadDotConfigure")) }
    var position = ident.Position()
    if !l.loadDir(position, dotConfigure, file.fullname(), nil) {
        diag.errorAt(position, "%s: load %v failed  (%s)", ident, dotConfigure, file.fullname())
    } else if loaded, yes := l.loaded[file.fullname()]; yes && loaded != nil {
        name, _ := l.project.scope.Lookup(loaded.Name()).(*ProjectName)
        if name == nil {
           diag.errorAt(position, "%v: %v: `.configure` is not a project", l.project.name, file)
        } else {
            if optionVerboseLoading { fmt.Fprintf(stderr, "smart: %v (%v)\n", name, file.fullname()) }
            if conf := l.project.configure; conf != nil {
                if conf == loaded { return }
                diag.errorAt(position, ".configure already specified")
            }
            l.project.configure = loaded
            result = true
        }
    }
    return
}

type declareOpts struct {
    breakUseLoop bool `b,break;l,loop`  // don't recursively use this project
    multiUseAllowed bool `m,multi`  // this project is used multiple times
}

func (l *loader) declare(keyword token.Token, ident *Bareword, options []Value) (result bool) {
    var pos = ident.Position()
    if ident.string == "@" {
        var (
            linfo = l.loads[0]
            dec, ok = linfo.declares[ident.string]
            at, _ = l.globe.scope.Lookup(ident.string).(*ProjectName)
        )
        if !ok {
            dec = &declare{ project: at.NamedProject() }
            linfo.declares[ident.string] = dec
        }
        dec.backproj = l.project
        dec.backscope = l.scope
        l.useesExecuted = nil
        l.project = at.NamedProject()
        l.scope = l.project.scope
        return true
    } else if _, o := l.scope.Find(ident.string); o != nil {
        if _, ok := o.(*Builtin); ok {
            diag.errorAt(pos, "project name '%s' is a builtin name", ident.string)
            return
        }
    }

    var (
        name = ident.string
        linfo = l.loads[len(l.loads)-1]
        dec, declared = linfo.declares[name]
    )
    if !declared {
        var (
            outer = l.scope
            absDir = linfo.absDir
            relPath, tmpPath string
        )
        if !filepath.IsAbs(absDir) {
            //absDir = filepath.Join(l.workdir, absDir)
            absDir, _ = filepath.Abs(absDir)
        }
        relPath, _ = filepath.Rel(l.workdir, absDir)
        tmpPath = joinTmpPath(l.workdir, relPath)

        // Avoid nesting project scopes!
        for strings.HasPrefix(outer.Comment(), "project \"") {
            outer = outer.outer
        }

        dec = &declare{ project: l.globe.project(pos,
            outer, absDir, relPath, tmpPath, linfo.specName, name) }
        l.loaded[linfo.absPath()] = dec.project
        linfo.declares[name] = dec
    }
    if loader := linfo.loader; loader != nil {
        if !strings.HasPrefix(loader.scope.comment, "project \"") {
            diag.warnAt(pos, "'%s' not loaded from project scope", name)
        }
        _, a := loader.scope.ProjectName(loader, name, dec.project)
        if a != nil {
            if v, ok := a.(*ProjectName); !ok || v == nil {
                diag.errorOf(a, "`%s` name already taken (%T).", name, a)
                return
            }
        }
    }

    if _, err := parseOpts(pos, &dec.project.opts, options...); err != nil {
        diag.errorAt(pos, "parse declare opts failed: %v", err)
        return
    }

    dec.backproj = l.project
    dec.backscope = l.scope
    l.useesExecuted = nil
    l.project = dec.project
    l.scope = l.project.scope
    if l.globe.main != nil && l.globe.main == l.project && l.project.name != "~" {
        for _, t := range context.pairs {
            switch k := t.Key.(type) {
            case *Bareword, *Barecomp:
                var ( name string; err error )
                if name, err = k.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                //if name[0] == '.' { name = "project" + name }
                var def, alt = l.def(l.position(), name)
                if def == nil && alt != nil { def = alt.(*Def) }
                def.set(DefDecl, t.Value)
            default:
                diag.errorAt(pos, "`%v` unknown target from command line (%v)\n", t, l.project)
                return
            }
        }
    }

    for _, arg := range merge(l.loadArgs...) {
        switch t := arg.(type) {
        case *Pair:
            var ( name string; err error )
            name, err = t.Key.Strval()
            if err != nil { diag.errorAt(pos, "%v", err); return }

            var def, alt = l.def(l.position(), name)
            if alt != nil {
                var ok bool
                def, ok = alt.(*Def)
                if !ok {
                    diag.errorAt(pos, "'%v' is not a Def (%T)", alt, alt)
                    return
                }
            }
            if def != nil { def.val(t.Value) }
        }
    }

    if err := l.loadPlugin(pos); err != nil {
        diag.errorAt(pos, "loadPlugin: %v", err)
        return
    }

   return true
}

func (l *loader) loadProjectConfiguration(ident *Bareword, declared bool) (result bool) {
    // FIXES: set cloctx immediately to ensure the right configuration is matched!
    defer setclosure(setclosure(cloctx.unshift(l.scope)))
    if false { defer un(tracef(t_traverse, "loadProjectConfiguration(%v)", ident)) }

    var pos = ident.Position()
    // Get configuration file name for the project and include it in flat mode.
    if s, err := configurationFileName(l.project); err != nil {
        diag.errorAt(pos, "%v: failed getting configuration file name: %v", ident, err)
        return
    } else if declared || optionConfigure {
        var exists bool
        for _, v := range configuration.clean { if s == v { exists = true; break }}
        if !exists { configuration.clean = append(configuration.clean, s) }
    } else if file := stat(pos, filepath.Base(s), "", filepath.Dir(s)); file != nil {
        if optionVerboseImport || optionVerboseLoading {
            full, _ := file.Strval()
            fmt.Fprintf(stderr, "Configuration for %s (%s) ⇒ %s\n", l.project, l.project.spec, full)
        } else if optionVerbose || true {
            fmt.Fprintf(stderr, "Configuration for %s (%s)\n", l.project, l.project.spec)
        }
        l.isIncludingConf = true
        l.includeFile(pos, file)
        l.isIncludingConf = false
    }

    if l.project.name != dotConfigure {
        // Looking for project specific .configure module
        if file := stat(pos, dotConfigure, "", l.project.absPath); exists(file) {
            if ident.string == dotConfigure {
                diag.errorAt(pos, "provided .configure for a .configure project")
            } else if !l.loadDotConfigure(ident, file) {
                //diag.errorAt(pos, "declare %s: %s/.configure", name, l.project.absPath)
            }
        }
    }
    return true
}

func (l *loader) loadProjectContainer(ident *Bareword) (result bool) {
    // FIXES: set cloctx immediately to ensure the right configuration is matched!
    defer setclosure(setclosure(cloctx.unshift(l.scope)))

    var pos = ident.Position()
    if l.project.name != dotContainer {
        if _, e := os.Stat(".dock"); e == nil {
            diag.errorAt(pos, "Must rename .dock into .container !")
            return
        }

        // Looking for project specific .container module
        if file := stat(pos, dotContainer, "", l.project.absPath); exists(file) {
            if !l.loadDotContainer(ident, file) {
                //diag.errorAt(pos, "declare %s: %s/.container", name, l.project.absPath)
            }
            return
        }

        // Looking for .smart/.container
        walkSmartBaseDirs(l.project.absPath, func(s string) bool {
            var file = stat(pos, dotContainer, "", filepath.Join(s, ".smart"))
            if !exists(file) {
                // no docking enabled
            } else if !l.loadDotContainer(ident, file) {
                //diag.errorAt(pos, "%v", err)
            }
            return false
        })

        result = true
    }
    return
}

func (l *loader) closeCurrent(ident *Bareword) (err error) {
    if ident.string == "@" {
        if dec, ok := l.loads[0].declares[ident.string]; ok {
            l.scope = dec.backscope
            l.project = dec.backproj
            l.useesExecuted = dec.useesExecuted
            dec.backproj = nil
            dec.backscope = nil
            dec.useesExecuted = nil
        }
        return nil
    }

    var linfo = l.loads[len(l.loads)-1]
    var dec, ok = linfo.declares[ident.string]
    if dec == nil || !ok {
        return fmt.Errorf("no loaded project %s", ident.string)
    }
    if l.project == nil {
        return fmt.Errorf("no current project")
    } else if s := l.project.Name(); s != ident.string {
        return fmt.Errorf("current project is %s but %s", s, ident.string)
    } else if l.project != dec.project {
        return fmt.Errorf("project conflicts (%s, %s)", l.project.Name(), dec.project.Name())
    }

    l.scope = dec.backscope
    l.project = dec.backproj
    l.useesExecuted = dec.useesExecuted
    return
}

func (l *loader) OpenNamedScope(name, comment string) (loaderScope, error) {
    if l.scope == nil {
        return loaderScope{}, fmt.Errorf("no parent scope (%v)", comment)
    }

    var outer = l.scope
    var pos = Position(l.parser.file.Position(l.parser.pos))
    var scope = NewScope(pos, outer, l.project, comment)
    if strings.HasPrefix(outer.Comment(), "dir ") {
        outer = outer.outer // discard dir scope
    }

    outer.ScopeName(l.project, name, scope)

    ls := loaderScope{ setclosure(cloctx.unshift(scope)), scope }
    l.scope = ls.scope
    return ls, nil
}

func (l *loader) resolve(value Value) (result Value, err error) {
    if sel, ok := value.(*selection); ok {
        result = sel
        return
    }

    var name string
    if name, err = value.Strval(); err == nil {
        if l.scope != nil { _, result = l.scope.Find(name) }
        if isNil(result) && l.project != nil {
            result, err = l.project.resolveObject(name)
        }
    }
    return
}

func (l *loader) find(target Value) (obj Object, err error) {
    var name string
    if name, err = target.Strval(); err != nil { return }

    var entry *RuleEntry
    if entry, err = l.project.resolveEntry(name); err != nil {
        return
    } else if entry != nil {
        obj = entry
    }
    return
}

func (l *loader) def(position Position, name string) (def *Def, alt Object) {
    var scope = l.scope
    if strings.HasPrefix(scope.comment, "file ") && l.mode&Flat != 0 {
        // use project scope if defining in flat file (aka. include)
        // to ensure that the symbol is valid in the project
        scope = l.project.scope
    }
    def, alt = scope.define(l.project, name, MakeNone(position))
    if def != nil { def.position = position }
    return
}

func (l *loader) assign(tok token.Token, def *Def, alt Object, value Value) (err error) {
    switch tok {
    case token.ASSIGN: // =
        err = def.set(DefDefault, value)
    case token.SCO_ASSIGN: // :=
        err = def.set(DefExpand1, value)
    case token.DCO_ASSIGN: // ::=
        err = def.set(DefExpand2, value)
    case token.EXC_ASSIGN: // !=
        err = def.set(DefExecute, value)
    case token.QUE_ASSIGN: // ?=
        if isNil(alt) {
            err = def.set(DefDefault, value)
        }
    case token.ADD_ASSIGN: // +=
        if isNil(value) || isNone(value) {
            // NOOP
        } else if isNil(def.value) || !def.value.refs(value) {
            err = def.append(value)
        } else {
            err = fmt.Errorf("can't append value '%v' to: %v", value, def)
        }
    case token.SHI_ASSIGN: // =+
        if !def.value.refs(value) {
            var tail = def.value
            if err = def.val(value); err == nil {
                err = def.append(merge(tail)...)
            }
        }
    case token.SUB_ASSIGN: // -=
        if isNil(def.value) || isNone(def.value) {
            // ...
        } else {
            var (
                vals []Value
                sub = merge(value)
            )
            for _, val := range merge(def.value) {
                var b bool
                for _, v := range sub {
                    if b = val.cmp(v) == cmpEqual; b { break }
                }
                if !b { vals = append(vals, val) }
            }
            def.value = MakeList(def.position, vals...)
        }
    case token.SAD_ASSIGN: // -+=
        var vals []Value
        if isNil(def.value) || isNone(def.value) {
            // ...
        } else {
            var sub = merge(value)
            for _, val := range merge(def.value) {
                var b bool
                for _, v := range sub {
                    if b = val.cmp(v) == cmpEqual; b { break }
                }
                if !b { vals = append(vals, val) }
            }
            vals = append(vals, sub...)
        }
        def.value = MakeList(def.position, vals...)
    case token.SSH_ASSIGN: // -=+
        var vals []Value
        if isNil(def.value) || isNone(def.value) {
            // ...
        } else {
            var sub = merge(value)
            for _, val := range merge(def.value) {
                var b bool
                for _, v := range sub {
                    if b = val.cmp(v) == cmpEqual; b { break }
                }
                if !b { vals = append(vals, val) }
            }
            vals = append(sub, vals...)
        }
        def.value = MakeList(def.position, vals...)
    }
    return
}

// If src != nil, readSource converts src to a []byte if possible;
// otherwise it returns an error. If src == nil, readSource returns
// the result of reading the file specified by filename.
//
func readSource(filename string, src interface{}) ([]byte, error) {
    if src != nil {
        switch s := src.(type) {
        case string:
            return []byte(s), nil
        case []byte:
            return s, nil
        case *bytes.Buffer:
            // is io.Reader, but src is already available in []byte form
            if s != nil {
                return s.Bytes(), nil
            }
        case io.Reader:
            var buf bytes.Buffer
            if _, err := io.Copy(&buf, s); err != nil {
                return nil, err
            }
            return buf.Bytes(), nil
        }
        return nil, fmt.Errorf("invalid source")
    }
    return ioutil.ReadFile(filename)
}

// ParseFile parses the source code of a single source file and returns
// the corresponding ast.File node. The source code may be provided via
// the filename of the source file, or via the src parameter.
func (l *loader) ParseFile(filename string, src interface{}, mode Mode) (f *parsedFile, err error) {
    if optionTraceLaunch { defer un(trace(t_launch, "loader.ParseFile")) }

    var text []byte
    if text, err = readSource(filename, src); err != nil {
        diag.errorAt(l.position(), "read source file failed: %v", err)
        return
    }

    //if optionTraceParsing { mode |= Trace }
    defer func(saved *parser, m Mode) {
        var panics int
        var pos Position
        if l.parser != nil && l.parser.file != nil {
            pos = l.parser.position()
        } else {
            pos.Filename = filename
        }
        for e := recover(); e != nil; e = recover() {
            if _, ok := e.(bailout); !ok {
                diag.errorAt(pos, "panic %v", e)
                panics += 1
            }
        }
        if panics > 0 {
            diag.errorAt(pos, "got %d panics", panics).debug(optionDebugErrors)
        }
        if err != nil {
            diag.errorAt(pos, "parse file failed: %v", err).debug(optionDebugErrors)
        }
        l.parser.loader = nil
        l.parser = saved
        l.mode = m
        //l.tracing.all = l.mode&AllErrors != 0
        //l.tracing.enabled = l.mode&Trace != 0
    } (l.parser, l.mode)

    // set the current parser
    l.parser = new(parser)
    l.parser.init(l, filename, text)
    l.mode = mode
    //l.tracing.all = l.mode&AllErrors != 0
    //l.tracing.enabled = l.mode&Trace != 0

    // set result values
    if f = l.parser.parseFile(); f == nil {
        // Source is not a valid source file, returnning a valid but empty parsedFile
        s := l.openScope(fmt.Sprintf("file %s", filename))
        f = &parsedFile{ scope:s.scope }
        f.position.Filename = filename
        // TODO: validate basename as a valid identifier
        f.name = MakeBareword(f.position, filepath.Base(filepath.Dir(filename)))
        l.closeScope(s)
    }
    return
}

// ParseConfigDir parses a configuration directory, where
//     * pathname - is the original pathname (symlink or 'configure' smart file)
//     * linked - is the destination directory pathname to be really iterated
func (l *loader) ParseConfigDir(pathname, linked string) (err error) {
    var fd *os.File // Directory of the destination.
    if fd, err = os.Open(linked); err != nil { return }
    defer fd.Close()

    var list []os.FileInfo
    if list, err = fd.Readdir(-1); err != nil || len(list) == 0 { return }

    var ident = filepath.Base(pathname)
    if ident == "_" {
        err = fmt.Errorf("invalid package name %s", ident)
        return
    }

    ls, err := l.OpenNamedScope(ident, fmt.Sprintf("config %s", pathname))
    if err != nil { return }
    defer l.closeScope(ls)

    var def *Def
ListLoop:
    for _, d := range list {
        var name = d.Name()
        if strings.HasPrefix(name, "~") ||
            strings.HasSuffix(name, ".#") ||
            strings.HasSuffix(name, ".smart") ||
            strings.HasSuffix(name, ".sm") { continue ListLoop }

        var fullname = filepath.Join(linked, name)
        if d.Mode()&os.ModeSymlink != 0 {
            var ( l string; t os.FileInfo )
            if l, err = os.Readlink(fullname); err != nil { continue ListLoop }
            if !filepath.IsAbs(l) { l = filepath.Join(linked, l) }
            if t, err = os.Stat(l); err != nil { continue ListLoop }
            if t.IsDir() { continue ListLoop }
        }

        var pos = Position(l.parser.file.Position(l.pos))
        if d.IsDir() {
            if err = l.ParseConfigDir(filepath.Join(pathname, name), fullname); err != nil { break ListLoop }
            if diag.checkErrors(true) > 0 { return }
        } else if s, a := l.def(l.position(), name); a != nil {
            diag.errorAt(pos, "declare project: %v", err)
            break ListLoop
        } else if def = s; def != nil {
            var ( v []byte; s string )
            if v, err = ioutil.ReadFile(fullname); err != nil { break ListLoop }
            if s = string(v); !utf8.ValidString(s) {
                diag.errorAt(pos, "%s: invalid UTF8 content", fullname)
                break ListLoop
            }
            def.set(DefConfDir, &String{valbase{pos},s})
        } else if s != nil {
            diag.errorAt(pos, "Name `%s' already taken, not def (%T).", name, s)
            break ListLoop
        }
    }
    return
}

// ParseDir calls ParseFile for all files with names ending in ".go" in the
// directory specified by path and returns a map of package name -> package
// AST with all the packages found.
//
// If filter != nil, only the files with os.FileInfo entries passing through
// the filter (and ending in ".go") are considered. The mode bits are passed
// to ParseFile unchanged. Position information is recorded in fset.
//
// If the directory couldn't be read, a nil map and the respective error are
// returned. If a parse error occurred, a non-nil but incomplete map and the
// first error encountered are returned.
//
func (l *loader) ParseDir(pos Position, path string, filter func(os.FileInfo) bool, mode Mode) (mods map[string]*Project) {
    defer func(t time.Time) {
        var d = time.Now().Sub(t)
        if optionVerboseParsing /*&& d > 50*time.Millisecond*/ {
            fmt.Fprintf(stderr, "parse(%15s) %s ⇒ %s\n", d, l.project, path)
        } else if optionBenchSlow && l.project == nil && d>5000*time.Millisecond {
            fmt.Fprintf(stderr, "smart: slow ▶parse(%s)◀ … (%s)\n", path, d)
        } else if optionBenchSlow && l.project != nil && d>2500*time.Millisecond {
            fmt.Fprintf(stderr, "smart: %s: slow ▶parse(%s)◀ … (%s)\n", l.project, path, d)
        }
    } (time.Now())

    var fd, err = os.Open(path)
    if err != nil {
        diag.errorAt(pos, "open(%s): %v", path, err)
        return
    }
    defer fd.Close()

    list, err := fd.Readdir(-1)
    if err != nil {
        diag.errorAt(pos, "readdir: %v", err)
        return
    } else if len(list) == 0 {
        diag.errorAt(pos, "no files underneath: %s", path)
        return
    }
    for i, a := range list {
        if i > 0 && a.Name() == "build.smart" {
            first := list[0]
            list[0] = a
            list[i] = first
        }
    }

    ls := l.openScope(fmt.Sprintf("dir %s", path))
    defer l.closeScope(ls)

    // FIXES: use 'globe' scope as outer to avoid chaining scopes to other unrelated
    // projects which are in consequence load order. Setting dir scope outer to such
    // project scopes will cause resolving objects to the wrong ones.
    ls.scope.outer = context.globe.scope

    mods = make(map[string]*Project)
ListLoop:
    for _, d := range list {
        var (
            filename, mo = filepath.Join(path, d.Name()), d.Mode()
            linked, linkPath = "", path
        )
        for fn := filename; mo&os.ModeSymlink != 0; {
            if s, err := os.Readlink(fn); err != nil {
                diag.errorAt(pos, "readlink: %v", err)
                continue ListLoop
            } else {
                rel := !filepath.IsAbs(s)
                if rel { s = filepath.Join(linkPath, s) }
                if fi, err := os.Lstat(s); err != nil {
                    diag.errorAt(pos, "lstat: %v", err)
                    continue ListLoop
                } else {
                    if rel { linkPath = filepath.Dir(s) }
                    mo, fn = fi.Mode(), s
                    linked = fn
                }
            }
        }

        var name = d.Name()
        if name != "" {
            var skip = strings.HasPrefix(name, ".#")
            skip = skip || !(strings.HasSuffix(name, ".smart") || strings.HasSuffix(name, ".sm"))
            if skip { continue ListLoop }
        }
        /*if (name == "configure.smart" || name == "configure.sm") && (linked != "" || mo.IsDir()) {
            //hasConfDir = true // TODO: remove ConfigDir feature
            if err := l.ParseConfigDir(filepath.Dir(filename), linked); err != nil {
                if first == nil {
                    first = err
                }
                return
            }
            continue ListLoop
        }*/
        if linked != "" { }

        if mo.IsRegular() && (filter == nil || filter(d)) {
            var src, err = l.ParseFile(filename, nil, mode|parsingDir)
            if diag.checkErrors(true) > 0 { return }
            if err == nil {
                var position Position
                if l.parser != nil && l.parser.file != nil {
                    position = Position(l.parser.file.Position(l.pos))
                } else {
                    position.Filename = filename
                }
                if src == nil {
                    diag.errorAt(position, "'%v' not loaded", filename)
                    return
                } else if src.name == nil {
                    diag.errorAt(position, "module '%v' has no name", filename)
                    return
                }

                name := src.name.string
                mod, found := mods[name]
                if !found {
                    mod = &Project{
                        name:  name,
                        scope: ls.scope,
                    }
                    mods[name] = mod
                }
                //mod.Files[filename] = src
            } else {
                diag.errorAt(pos, "ParseFile: %v", err)
            }
        }
    }
    return
}

// loader.Load loads script from a file or source code (string, []byte).
func (l *loader) load(specName, absPath string, source interface{}) (result bool) {
    if optionTraceLaunch { defer un(trace(t_launch, "loader.load")) }
    defer func(t time.Time) {
        var d = time.Now().Sub(t)
        if optionVerboseLoading /*&& d > 50*time.Millisecond*/ {
            loaded, _ := l.loaded[absPath]
            if l.project == nil {
                fmt.Fprintf(stderr, "load (%15s) ⇒ %s (%s)\n", d, loaded, specName)
            } else {
                fmt.Fprintf(stderr, "load (%15s) %s ⇒ %s (%s)\n", d, l.project.name, loaded, specName)
            }
        } else if optionBenchSlow && d > 100*time.Millisecond {
            fmt.Fprintf(stderr, "smart: %s: slow ▶load(%s) … (%s)◀\n", l.project.name, specName, d)
        }
    } (time.Now())

    if absPath == "" {
        l.error(l.pos, "no such module `%s' (in paths %v)", specName, l.paths)
        return
    } else if !filepath.IsAbs(absPath) {
        l.error(l.pos, "invalid abs name `%s' (%s)", absPath, specName)
        return
    }

    // Check already project.
    if loaded, yes := l.loaded[absPath]; yes {
        _, a := l.project.scope.ProjectName(l.project, loaded.Name(), loaded)
        if a != nil {
            if v, ok := a.(*ProjectName); !ok || v == nil {
                l.error(l.pos, "`%v` name already taken (%T).", loaded, a)
            }
        }
        return
    }

    var absDir, baseName = filepath.Split(absPath)
    defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, baseName))

    var doc, err = l.ParseFile(absPath, source, parseMode)
    if diag.checkErrors(true) > 0 { return } else if err != nil {
        l.error(l.pos, "load: %v", err)
    } else if doc == nil {
        l.error(l.pos, "load: doc is nil (%s)", absPath)
    } else {
        result = true
    }
    return
}

func (l *loader) loadDir(pos Position, specName, absDir string, filter func(os.FileInfo) bool) (result bool) {
    if optionTraceLaunch { defer un(trace(t_launch, "loader.loadDir")) }
    defer func(t time.Time) {
        var d = time.Now().Sub(t)
        if optionVerboseLoading /*&& d > 50*time.Millisecond*/ {
            loaded, _ := l.loaded[absDir]
            if l.project == nil {
                fmt.Fprintf(stderr, "load (%15s) ⇒ %s (%s)\n", d, loaded, specName)
            } else {
                fmt.Fprintf(stderr, "load (%15s) %s ⇒ %s (%s)\n", d, l.project.name, loaded, specName)
            }
        } else if optionBenchSlow && l.project == nil && d>5000*time.Millisecond {
            fmt.Fprintf(stderr, "smart: slow ▶load(%s)◀ … (%s)\n", specName, d)
        } else if optionBenchSlow && l.project != nil && d>2500*time.Millisecond {
            fmt.Fprintf(stderr, "smart: %s: slow ▶load(%s)◀ … (%s)\n", l.project.name, specName, d)
        }
    } (time.Now())

    if !filepath.IsAbs(absDir) {
        diag.errorAt(pos, "needs absolute dir `%s' (%s)", absDir, specName)
        return
    }

    var loaded *Project
    // Check already loaded project.
    if loaded, result = l.loaded[absDir]; result {
        _, a := l.project.scope.ProjectName(l.project, loaded.Name(), loaded)
        if a != nil {
            if v, ok := a.(*ProjectName); !ok || v == nil {
                diag.errorAt(pos, "name `%s' already taken (%T).", loaded.Name(), a)
            }
        }
        return
    }

    defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, ""))

    var mods = l.ParseDir(pos, absDir, filter, parseMode)
    if diag.checkErrors(true) > 0 { return }
    // FIXME: loading failed if different 'project' found in
    // the same dir, for example:
    //      project Foo # file build.smart
    //      project # file config.smart
    if len(mods) == 0 && filepath.Base(specName) != "@" {
        diag.errorAt(pos, "parse failed: %s (%s)", specName, absDir)
    }
    if loaded, result = l.loaded[absDir]; result && loaded != nil { // Good!
        if false {
            a := l.project.scope.Lookup(loaded.Name())
            fmt.Fprintf(stderr, "%s: %v %s\n", l.project, loaded, a)
        }
    } else if filepath.Base(specName) != "@" {
        diag.errorAt(pos, "loadDir: '%s' not loaded (%s)", specName, absDir)
    }
    return
}

func (l *loader) loadWithArgs(position Position, specName, absPath string, args []Value, source interface{}) bool {
    if optionTraceLaunch { defer un(trace(t_launch, "loader.loadWithArgs")) }
    defer l.setArgs(l.setArgs(args))
    return l.load(specName, absPath, source)
}

func (l *loader) loadDirWithArgs(position Position, specName, absPath string, args []Value, filter func(os.FileInfo) bool) bool {
    if optionTraceLaunch { defer un(trace(t_launch, "loader.loadDirWithArgs")) }
    defer l.setArgs(l.setArgs(args))
    return l.loadDir(position, specName, absPath, filter)
}

func (l *loader) loadFile(filename string, source interface{}) bool {
    if optionTraceLaunch { defer un(trace(t_launch, "loader.loadFile")) }
    s, _ := filepath.Split(filename)
    s, _  = filepath.Rel(l.workdir, s)
    return l.load(s, filename, source)
}

func (l *loader) loadPath(path string, filter func(os.FileInfo) bool) bool {
    if optionTraceLaunch { defer un(trace(t_launch, "loader.loadPath")) }
    s, _ := filepath.Rel(l.workdir, path)
    var position Position
    position.Filename = s
    return l.loadDir(position, s, path, filter)
}

func (l *loader) loadText(filename string, text string) []Value {
    if optionTraceLaunch { defer un(trace(t_launch, "loader.loadText")) }

    defer func(saved *parser) {
        l.parser.loader = nil
        l.parser = saved
    } (l.parser)

    if l.globe.main == nil {
        l.project = l.globe.os
    } else {
        l.project = l.globe.main
    }
    l.scope = l.project.scope
    l.useesExecuted = nil

    l.parser = new(parser)
    l.parser.init(l, filename, []byte(text))
    return l.parser.parseText()
}

func AddSearchPaths(paths... string) (err error) {
    for _, s := range paths {
        if s, err = filepath.Abs(s); err != nil {
            break
        }
        if fi, _ := os.Stat(s); fi != nil && fi.IsDir() {
            globalPaths = append(globalPaths, s)
        }
    }
    return
}
