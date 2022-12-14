//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
  "extbit.io/smart/token"
  "path/filepath"
  "runtime"
  "strings"
  "plugin"
  "sync"
  "time"
  "fmt"
  "os"
)

const PathSep = string(filepath.Separator)

type FileMap struct {
  pattern Value
  Paths []Value
}

func (filemap *FileMap) String() string { return filemap.pattern.String() }
func (filemap *FileMap) Patterns() (pats []Value) {
  if filemap.pattern.closured() {
    var err error
    if pats, err = mergeresult(ExpandAll(filemap.pattern)); err != nil {
      fmt.Fprintf(stderr, "%v: %v\n", filemap.pattern.Position(), filemap.pattern)
    }
  } else {
    pats = append(pats, filemap.pattern)
  }
  return
}

func isRealPattern(pattern Value) (result bool) {
  if t, ok := pattern.(*Path); ok {
    result = t.isPattern()
  } else if _, ok := pattern.(Pattern); ok {
    result = true
  }
  return
}

// Match split filename into list and match each part with the pattern correspondingly.
func (filemap *FileMap) Match(filename string) (matched bool, pre string) {
  /*if filemap.Pattern.closured() {
    if pats, err := mergeresult(ExpandAll(filemap.Pattern)); err != nil {
      fmt.Fprintf(stderr, "%v: %v\n", filemap.Pattern.Position(), filemap.Pattern)
    } else {
      for _, pat := range pats {
        if matched, pre = filemap.match(pat, filename); matched {
          return
        }
      }
    }
  } else {
    matched, pre = filemap.match(filemap.Pattern, filename)
  }*/
  for _, pat := range filemap.Patterns() {
    if matched, pre = filemap.match(pat, filename); matched { break }
  }
  return
}

func (filemap *FileMap) match(pattern Value, filename string) (matched bool, pre string) {
  if matched, pre = globMatch(pattern, filename); matched { return }
  if false { // TODO: support percent (%, %%) and regex matching
    var ( s, t string ; e error )
    if t, e = pattern.Strval(); e != nil { return }
    for _, p := range filemap.Paths {
      if s, e = p.Strval(); e != nil { return }
      if filename == filepath.Join(s, t) {
        matched = true
      }
    }
  }
  return
}

func (filemap *FileMap) stat(base, pre, name string) (file *File) {
  var pos = filemap.pattern.Position()
  if filemap.Paths == nil {
    // Check file in the filesystem (no paths).
    file = stat(pos, name, "", base, nil)
    return
  }
  base = filepath.Clean(base)
  pre  = filepath.Clean(pre)
  for _, path := range filemap.Paths {
    if path == nil {
      diag.errorAt(pos, "mapping nil path (base=%s, pre=%s, name=%s)", base, pre, name)
      panic("internal error")
    }

    var ( dir, sub string ; err error )
    if sub, err = path.Strval(); err != nil { return } else {
      // Clean the search path.
      sub = filepath.Clean(sub)
    }

    // Absolute path or using the base.
    if filepath.IsAbs(sub) {
      dir = sub
      sub = ""
    } else {
      dir = base //filepath.Join(base, sub)
    }

    /*if filepath.IsAbs(name) && !strings.HasPrefix(name, dir+PathSep) {
                        continue
                }*/

    // Check file in the filesystem.
    if file = stat(pos, name, sub, dir, nil); file != nil {
      break
    }

    if filepath.IsAbs(sub) {
      if pre == "" { // Fullmatch!
        // For example of:
        //   xxx.c  <->  (*.c => /path/to/source)
        // Become:
        //   /path/to/source  ""  xxx.c
        file = stat(pos, name, "", sub, nil)
      } else if strings.HasSuffix(sub, PathSep+pre) {
        // For example of:
        //   foo/bar/xxx.c  <->  (*.c => /path/to/source/foo/bar)
        // Become:
        //   /path/to/source  foo/bar  xxx.c
        s := strings.TrimSuffix(sub, PathSep+pre)
        n := strings.TrimPrefix(name, pre+PathSep)
        file = stat(pos, n, pre, s, nil)
      } else if false { // This is wrong, only base name matched!!
        // For example of:
        //   foo/bar/xxx.c  <->  (*.c => /path/to/source)
        // Become:
        //   /path/to/source  foo/bar  xxx.c
        n := strings.TrimPrefix(name, pre+PathSep)
        file = stat(pos, n, pre, sub, nil)
      }
    } else {
      if pre == "" { // Fullmatch!
        // For example of:
        //   xxx.c  <->  (*.c => source)
        // Become:
        //   <p.absPath>  source  xxx.c
        file = stat(pos, name, sub, dir, nil)
      } else if sub == pre {
        // For example of:
        //   foo/bar/xxx.c  <->  (*.c => foo/bar)
        // Become:
        //   <dir>  foo/bar  xxx.c
        n := strings.TrimPrefix(name, pre+PathSep)
        file = stat(pos, n, sub, dir, nil)
      } else if strings.HasSuffix(sub, PathSep+pre) {
        // For example of:
        //   foo/bar/xxx.c  <->  (*.c => source/foo/bar)
        // Become:
        //   <dir>  source/foo/bar  xxx.c
        s := strings.TrimSuffix(sub, PathSep+pre)
        n := strings.TrimPrefix(name, pre+PathSep)
        file = stat(pos, n, pre, s, nil)
      } else if false { // This is wrong, only base name matched!!
        // For example of:
        //   foo/bar/xxx.c  <->  (*.c => source)
        // Become:
        //   <dir>  source/foo/bar  xxx.c
        s := filepath.Join(sub, pre)
        n := strings.TrimPrefix(name, pre+PathSep)
        file = stat(pos, n, s, dir, nil)
      }
    }
  }
  return
}

// copy of filepath.hasMeta
func hasGlobMeta(path string) bool {
	magicChars := `*?[`
	if runtime.GOOS != "windows" {
		magicChars = `*?[\`
	}
	return strings.ContainsAny(path, magicChars)
}

// globMatch - Glob matching each component of the filename against the
// glob value. It checks in two different ways. If the filename and the
// glob pattern has the some number of components (splitted by PathSep),
// all components are compared. If the pattern has only one component,
// the last filename component is compared with the pattern, and the prefix
// components are returned in 'pre'.
func globMatch(patval Value, filename string) (matched bool, pre string) {
  pattern, err := patval.Strval()
  if err != nil { return false, "" }

  list0 := strings.Split(filepath.Clean(pattern), PathSep)
  list1 := strings.Split(filepath.Clean(filename), PathSep)
  if len(list0) == 0 {
    // FIXME: match any?
  } else if len(list0) == len(list1) { // foo/*.o  <->  src/foo.o
    // Matching all components
    for i, pat := range list0 {
      if true /*hasGlobMeta(pat)*/ {
        matched, _ = filepath.Match(pat, list1[i])
        if !matched { return }
      } else {
        matched = (pat == list1[i])
      }
    }
  } else if len(list0) == 1 && len(list1) > 1 { // *.o|foo.o  <->  src/foo.o
    // Matching the last component of filename and returns
    // the prefix if matched.
    list1_tail := list1[len(list1)-1]
    if true /*hasGlobMeta(list0[0])*/ {
      matched, _ = filepath.Match(list0[0], list1_tail)
    } else {
      matched = (list0[0] == list1_tail)
    }
    if matched {
      pre = filepath.Join(list1[:len(list1)-1]...)
    }
  }
  return
}

type enterec struct {
  wd, dir string
  print, silent bool
  num int
}

func (rec *enterec) String() string { return rec.dir }

var cd = &struct{
  stack []*enterec // entered directories
  enters map[string]*enterec // enters
}{
  enters: make(map[string]*enterec),
}

type useRuleEntry struct {
  RuleEntry
  post bool
}

type Project struct {
  position Position
  keyword  token.Token // project, package, module

  self *ProjectName // $:self:
  configure *Project // .configure

  changedWD string
	absPath string
	relPath string
  tmpPath string
	spec    string
	name    string
  scope   *Scope
  bases []*Project
  loads []*Project
  using   *usinglist

  // List order is significant, duplication is acceptable.
  _files_ []Value
  _filemap_ []*FileMap

  // Rule Registry (orderred)
  userules []*useRuleEntry // the 'use' rule
  concrete []*RuleEntry
  patterns []*PatternEntry

  filescopes []*Scope

  // TODO: printEntering() ...
  // TODO: printLeaving() ...

  plugin *plugin.Plugin
  pluginScope *Scope

  multiUseAllowed bool // this project is used multiple times
  breakUseLoop bool // don't recursively use this project
}

func (p *Project) String() string { return p.name }

func (p *Project) NewScope(pos Position, comment string) *Scope {
  return NewScope(pos, p.scope, p, comment)
}

func (p *Project) AbsPath() string { return p.absPath }
func (p *Project) RelPath() string { return p.relPath }
func (p *Project) Spec() string { return p.spec }
func (p *Project) Name() string { return p.name }
func (p *Project) Scope() *Scope { return p.scope }
func (p *Project) Bases() []*Project { return p.bases }
func (p *Project) Chain(bases ...*Project) {
  for _, base := range bases { p.bases = append(p.bases, base) }
}

func (p *Project) mapfile(pat Value, paths []Value) {
  // List order is significant, duplication is acceptable.
  p._filemap_ = append(p._filemap_, &FileMap{ pat, paths })
}
func (p *Project) getFilemap() (filemap []*FileMap) {
  if true { return p._filemap_ }

  var _filemap_ []*FileMap
  if len(_filemap_) == 0 && len(p._files_) > 0 {
    var mapfile = func (pat Value, paths []Value) {
      // List order is significant, duplication is acceptable.
      _filemap_ = append(_filemap_, &FileMap{ pat, paths })
    }
    for _, spec := range p._files_ {
      switch v := spec.(type) {
      case *Pair:
        var pats, paths []Value
        switch k := v.Key.(type) {
        case *Group: pats = k.Elems
        default: pats = append(pats, v.Key)
        }
        if a, err := mergeresult(ExpandAll(pats...)); err != nil {
          fmt.Fprintf(stderr, "%s: expand error: %s\n", v.Position(), v)
          fmt.Fprintf(stderr, "%s\n", err)
        } else {
          pats = a 
        }
        switch vv := v.Value.(type) {
        case *Group: paths = vv.Elems
        default: paths = append(paths, vv)
        }
        for _, k := range pats { mapfile(k, paths) }
      case Value:
        var pats, paths []Value
        paths = []Value{&String{trivial{v.Position()},p.absPath}}
        switch g := v.(type) {
        default: pats = append(pats, v)
        case *Group: pats = g.Elems
        }
        for _, k := range pats { mapfile(k, paths) }
      default:
        fmt.Fprintf(stderr, "%s: invalid file spec: %v\n", v.Position(), v)
      }
    }
  }
  return _filemap_
}

func (p *Project) filemaps(using bool) (filemaps []*FileMap) {
  if optionEnableBenchmarks && false { defer bench(mark("Project.filemaps")) }

  var uniqueAppend = func(a *FileMap) {
    var numDuplicated int
    for _, m := range filemaps {
      if a == m || (a.pattern == m.pattern && &a.Paths == &m.Paths) { return } else
      if a.pattern == m.pattern && len(a.Paths) == len(m.Paths) {
        var same = true // initially assumes all paths are identical
        for i, ap := range a.Paths {
          if ap != m.Paths[i] { same = false; break }
        }
        if same { return } else {
          diag.warnOf(a.pattern,  "files might be duplicated: %v (paths=%v),", a, a.Paths)
          diag.warnOf(m.pattern,  "                     with: %v (paths=%v)", m, m.Paths)
          diag.warnOf(a.Paths[0], "          differred paths: %v", a.Paths[0])
          diag.warnOf(m.Paths[0], "                      and: %v", m.Paths[0])
          numDuplicated += 1
        }
      }
    }
    if numDuplicated > 0 { diag.errorOf(a.pattern, "duplicated files: %v", a.pattern) }
    filemaps = append(filemaps, a)
  }
  for _, m := range p.getFilemap() { uniqueAppend(m) }
  if using {
    if true {
      // takes a big longer time to map usee filemaps, but acceptable
      var appendUsingList func(*Project)
      appendUsingList = func(p *Project) {
        for _, m := range p.getFilemap() {
          uniqueAppend(m)
        }
        for _, u := range p.using.list {
          appendUsingList(u.project)
        }
      }
      appendUsingList(p)
    } else {
      for _, u := range p.using.list {
        if false {
          // low performance doing recursivly file mapping
          for _, m := range u.project.filemaps(/*using*/false) {
            uniqueAppend(m) //app(u.project.filemaps(loads))
          }
        } else {
          for _, m := range u.project.getFilemap() {
            uniqueAppend(m)
          }
        }
      }
    }
  }
  for _, base := range p.bases {
    for _, m := range base.filemaps(using) { uniqueAppend(m) }
  }
  return
}

func (p *Project) wildcard(pos Position, wo wildcardOpts, patterns ...Value) (files []*File, err error) {
  var filemaps = p.filemaps(false)
ForPats:
  for _, pat := range patterns {
    var ( patStr string; matched, breakAbsRel bool )
    if patStr, err = pat.Strval(); err != nil { diag.errorAt(pos, "wildcard: %v", err); break ForPats }
    // The 'patStr' could be GlobPattern or just regular file/path names. PercPattern is not supported yet.
  ForFilemaps:
    for _, fm := range filemaps {
      for _, pattern := range fm.Patterns() {
        var pre string // <pre>/*.xxx
        var str = patStr
        if matched, pre = globMatch(pattern, patStr); !matched {
          // Flip glob matching order.
          if _, yes := pat.(*GlobPattern); !yes {
            continue ForFilemaps
          } else if str, err = pattern.Strval(); err != nil {
            break ForPats
          } else if matched, pre = globMatch(pat, str); !matched {
            continue ForFilemaps
          } else {
            // using the arg glob
            breakAbsRel = true
          }
        }

        if pre != "" { /* FIXME: ... */ }

        var names []string

        // Absolute or relative files are not related to the paths.
        if filepath.IsAbs(str) || strings.HasPrefix(str, "./") || strings.HasPrefix(str, "../") {
          if names, err = filepath.Glob(str); err != nil { break ForPats }
          for _, s := range names {
            file := stat(pos, filepath.Base(s), "", filepath.Dir(s))
            files = append(files, file)
            if enable_assertions {
              assert(file != nil, "`%s` missing", s)
            }
          }
          if breakAbsRel {
            continue ForPats
          } else {
            continue ForFilemaps
          }
        }

        // Check against paths for non-abs/rel patterns.
        for _, path := range fm.Paths {
          var sub string
          if sub, err = path.Strval(); err != nil {
            break ForPats
          }

          subfile := filepath.Join(sub, str)
          if names, err = filepath.Glob(subfile); err != nil {
            break ForPats
          }
          // Chop off path 'sub' prefix to have shorter names
          // Aka. trim prefix 'file.Sub+PathSep'
          prefix := strings.TrimSuffix(subfile, str)
          if len(names) > 0 {
            for _, s := range names {
              name := strings.TrimPrefix(s, prefix)
              file := stat(pos, name, sub, prefix)
              files = append(files, file)
              if enable_assertions {
                assert(file != nil, "`%s` missing (%s)", s, name)
              }
            }
          } else if ok := isRealPattern(pattern); !ok && wo.optIncludeMissing {
            // If the filemap is not a pattern (e.g. foobar.cpp), we include it in the returning files
            var name string
            name, err = pattern.Strval()
            if err != nil { break ForPats }

            // Append this non-existed/missing file.
            file := stat(pos, name, sub, prefix, nil)
            files = append(files, file)

            if false { fmt.Fprintf(stderr, "%s: %s -> %s\n", pos, pat, file) }
          } else if ok && len(fm.Paths) == 1 {
            // Just report that the pattern matches no files in the
            // file system (if only one path specified).
            var pp1 = pattern.Position()
            var pp2 =     pat.Position()
            fmt.Fprintf(stderr, "%s: %s: %s: '%v' not found in '%v'\n", pp1, p.name, pat, fm, sub)
            fmt.Fprintf(stderr, "%s: %s: wildcard: %v (try using flag -m, aka -include-missing)\n", pp2, p.name, pat)
          } else if optionWildcardMissingError {
            err = fmt.Errorf("files like '%v' not found", fm)
            break ForPats
          }
        }
      }
    }
  }
  return
}

func (p *Project) matchFile(name string) (file *File) {
  if optionEnableBenchmarks && false { defer bench(mark("Project.matchFile")) }
  if optionEnableBenchspots { defer bench(spot("Project.matchFile")) }

  //[optional]: defer setclosure(setclosure(cloctx.unshift(p.scope)))

  var first *File
ForFilemaps:
  for _, filemap := range p.filemaps(true) {
    // Match the represented file name.
    var matched, pre = filemap.Match(name)
    if !matched { continue ForFilemaps }
    if p.changedWD != "" { file = filemap.stat(p.changedWD, pre, name) }
    if file == nil { file = filemap.stat(p.absPath, pre, name) }
    if false { fmt.Fprintf(stderr, "%s: %s (file=%v (aka. file.name), exists=%v, cwd=%s) (project.matchFile)\n", p, name, file, exists(file), p.changedWD) }
    if file != nil {
      if file.match == nil { file.match = filemap }
      if pre != "" { /* FIXME: file.change(...pre) */ }
      if exists(file) { break ForFilemaps }
      if first == nil { first = file }
    }
    // If the filemap entry is defined by the project itself,
    // we have to break the matching loop. So that the current
    // project have a chance to define it's own file. This is
    // usefull when the bases (or imported projects) have also
    // matched files. The current project have the highest
    // priority to match.
    for _, fm := range p.getFilemap() {
      if filemap == fm { break ForFilemaps }
    }
  }
  if first != file && !exists(file) { file = first }
  return
}

func (p *Project) matchTempFile(pos Position, name string) (file *File) {
  if file = p.matchFile(name); file != nil {
    // good
  } else if ctd := p.scope.FindDef("CTD"); ctd == nil {
    unreachable()
  } else if s, err := ctd.Strval(); err == nil {
    // stat temp file (maybe not existed)
    file = stat(pos, filepath.Join(s, name), "", "", nil)
  } else {
    fmt.Fprintf(stderr, "%v: %v\n", p, err)
  }
  return
}

func (p *Project) isFileName(s string) (res bool) {
  if len(s) > 0 {
    for _, filemap := range p.filemaps(true) {
      if res, _ = filemap.Match(s); res { break }
    }
  }
  return
}

func (p *Project) DefaultEntry() (entry *RuleEntry) {
  if len(p.concrete) > 0 {
    entry = p.concrete[0]
  }
  return
}

func (p *Project) resolveObject(s string) (obj Object, err error) {
  if _, obj = p.scope.Find(s); obj == nil {
    if p.pluginScope != nil {
      if obj = p.pluginScope.Lookup(s); obj != nil {
        return
      }
    }
    for _, base := range p.bases {
      obj, err = base.resolveObject(s)
      if err != nil || obj != nil {
        break
      }
    }
  }
  return
}

func (p *Project) resolveEntry(s string) (entry *RuleEntry, err error) {
  if optionEnableBenchmarks && false { defer bench(mark("Project.resolveEntry")) }
  if optionEnableBenchspots { defer bench(spot("Project.resolveEntry")) }
  for _, rec := range p.concrete {
    switch target := rec.target.(type) {
    case *File:
      if target.name == s { return rec, nil }
    default:
      var sv string
      if sv, err = target.Strval(); err != nil { return }
      if sv == s { return rec, nil }
    }
  }
  for _, base := range p.bases {
    entry, err = base.resolveEntry(s)
    if err != nil || entry != nil { break }
  }
  if err == nil && entry == nil {
    if true { /* FAST */ } else { /* SLOW */
      for _, using := range p.using.list {
        entry, err = using.project.resolveEntry(s)
        if err != nil || entry != nil { break }
      }
    }
  }
  return
}

func (p *Project) resolvePatterns(i interface{}) (res []*StemmedEntry, err error) {
  if optionEnableBenchmarks && false { defer bench(mark("Project.resolvePatterns")) }
  if optionEnableBenchspots { defer bench(spot("Project.resolvePatterns")) }
  var v []*StemmedEntry
  if res, err = p._resolvePatterns1(i); err != nil { return }
  if v, err = p._resolvePatterns2(i); err != nil { return } else {
    res = append(res, v...)
  }
  if true { /* FAST */ } else /* SLOW */
  if v, err = p._resolvePatterns3(i); err != nil { return } else {
    res = append(res, v...)
  }
  return
}

func (p *Project) _resolvePatterns1(i interface{}) (res []*StemmedEntry, err error) {
  if optionEnableBenchspots { defer bench(spot("Project._resolvePatterns1")) }
  for _, pat := range p.patterns {
    var ( s string ; stems []string )
    if s, stems, err = pat.Pattern.match(i); err != nil {
      return
    } else if s != "" && stems != nil {
      res = append(res, &StemmedEntry{pat, stems})
    }
  }
  return
}

func (p *Project) _resolvePatterns2(i interface{}) (res []*StemmedEntry, err error) {
  if optionEnableBenchspots { defer bench(spot("Project._resolvePatterns2")) }
  for _, base := range p.bases {
    var ses []*StemmedEntry
    ses, err = base.resolvePatterns(i)
    if err != nil { return }
    res = append(res, ses...)
  }
  return
}

func (p *Project) _resolvePatterns3(i interface{}) (res []*StemmedEntry, err error) {
  if optionEnableBenchspots { defer bench(spot("Project._resolvePatterns3")) }
  for _, using := range p.using.list {
    var ses []*StemmedEntry
    ses, err = using.project.resolvePatterns(i)
    if err != nil { return }
    res = append(res, ses...)
  }
  return
}

func (p *Project) entry(special specialRule, options []Value, target Value, prog *Program) (entry *RuleEntry, err error) {
  defer func() {
    if entry != nil && err == nil {
      entry.programs = append(entry.programs, prog)
    }
  } ()

  var strval string
  if strval, err = target.Strval(); err != nil {
    return
  }

  // The 'use' rule entries.
  var closured = target.closured()
  if special == specialRuleUse && !closured {
    var optPostExecute bool
    if _, err = parseFlags(options, []string{
      "p,post",
    }, func(ru rune, v Value) {
      switch ru {
      case 'p': if optPostExecute, err = trueVal(v, false); err != nil { return }
      }
    }); err != nil { return }
    var userule = &useRuleEntry{
      RuleEntry{ class:UseRuleEntry, target:target },
      optPostExecute, // post-execute use rule?
    }
    p.userules = append(p.userules, userule)
    entry = &userule.RuleEntry
    return
  }

  var name string
  if name, err = target.Strval(); err != nil {
    return
  } else if name == "" {
    err = fmt.Errorf("name '%v' already taken as `%T'", name)
    return
  }

  // Looking for pattern rule entries.
  switch t := target.(type) {
  case *PercPattern:
    assert(t != nil, "nil PercPattern")
    entry = &RuleEntry{
      class: PercRuleEntry,
      target: target,
    }
    p.patterns = append(p.patterns, &PatternEntry{ t, entry })
    return
  case *GlobPattern:
    assert(t != nil, "nil GlobPattern")
    entry = &RuleEntry{
      class: GlobRuleEntry,
      target: target,
    }
    panic("TODO: GlobPattern target")
  case *RegexpPattern:
    assert(t != nil, "nil RegexpRuleEntry")
    entry = &RuleEntry{
      class: RegexpRuleEntry,
      target: target,
    }
    panic("TODO: RegexpPattern target")
  case *Path:
    var isPathPattern bool
  ForPathElements:
    for _, elem := range t.Elems {
      switch elem.(type) {
      case *PercPattern:
        isPathPattern = true
        break ForPathElements
      case *GlobPattern:
        panic("TODO: GlobPattern path target")
      case *RegexpPattern:
        panic("TODO: RegexpPattern path target")
      }
    }
    if isPathPattern {
      entry = &RuleEntry{
        class: PathPattRuleEntry,
        target: target,
      }
      p.patterns = append(p.patterns, &PatternEntry{ t, entry })
      return
    }
  }

  // Looking for concrete rule entries.
  for _, rec := range p.concrete {
    var sv string
    if closured && rec.String() == name {
      entry = rec; break
    } else if sv, err = rec.Strval(); err != nil {
      return
    } else if sv == strval {
      entry = rec; break
    }
  }
  if entry == nil {
    entry = &RuleEntry{
      class: GeneralRuleEntry,
      target: target,
    }
    p.concrete = append(p.concrete, entry)
  }
  return
}

func (p *Project) isa(proj *Project) (res bool) {
  for _, base := range p.bases {
    if base == proj { res = true; break }
  }
  return
}

func (p *Project) hasBase(proj *Project) (res bool) {
  for _, base := range p.bases {
    if res = base == proj; res { break }
    if res = base.hasBase(proj); res { break }
  }
  return
}

func (p *Project) hasLoaded(proj *Project, breakUseLoop bool) (rp *Project, res, isb bool, err error) {
  return p.hasLoadedRecur(p, proj, breakUseLoop)
}

func (p *Project) hasLoadedRecur(top, proj *Project, breakUseLoop bool) (rp *Project, res, isb bool, err error) {
  for _, base := range p.bases {
    if isb = base == proj; isb { return }
    if rp, res, isb, err = base.hasLoadedRecur(top, proj, breakUseLoop); err != nil {
      return
    } else if res || isb { rp = base ; return }
  }
  for _, imp := range p.loads {
    if imp == top && !breakUseLoop {
      s := top.loopLoadPath()
      err = fmt.Errorf("loop `%v`", s)
      return
    }
    if res = imp == proj; res { rp = imp; return }
    if rp, res, res, err = imp.hasLoadedRecur(top, proj, breakUseLoop); err != nil {
      return
    } else if res { rp = imp; return }
  }
  rp = p
  return
}

func (p *Project) loopLoadPath() (s string) { return p.loopLoadRecur(p) }
func (p *Project) loopLoadRecur(top *Project) (s string) {
  for _, imp := range p.loads {
    if imp == top {
      if p != top { s = "???" }
      s += fmt.Sprintf("(%s)???(%s)", p.spec, imp.spec)
      break
    }
    if t := imp.loopLoadRecur(top); t != "" {
      if p != top { s = "???" }
      s += fmt.Sprintf("(%s)%s", p.spec, t)
      break
    }
  }
  return
}

func (p *Project) isUsingProject(usee *Project) (res bool) {
  for _, using := range p.using.list {
    if res = using.project == usee; res { break  }
    if res = using.project.isUsingProject(usee); res { break }
  }
  return
}

func (p *Project) isUsingDirectly(proj *Project) (res bool) {
  for _, u := range p.using.list {
    if res = u.project == proj; res { break }
  }
  return
}

func (p *Project) usees(post bool) (res []*Project) {
  if p.breakUseLoop { return }
  for _, u := range p.using.list {
    if !post { res = append(res, u.project) }
    for _, u := range u.project.usees(post) {
      if !p.isUsingDirectly(u) { res = append(res, u) }
    }
    if post { res = append(res, u.project) }
  }
  return
}

//var cdUnlocked = make(chan bool, 1)
// Note: this is okay not using an atomic value, because
// chdirMutex can serve to protect the whole timeframe.
//var cdUnlockTime atomic.Value
var chdirMutex = new(sync.Mutex)

func lockCD(dir string, dura time.Duration) error {
  // Protect the work directory, `chdirMutex` ensures that
  // there's only one timer being counting to avoid work
  // directory being changed before the deadline.
  chdirMutex.Lock()
  go func() {
    if dura > 0 { time.Sleep(dura) }
    chdirMutex.Unlock()
  } ()
  return os.Chdir(dir)
}

func enter(prog *Program, dir string) (err error) {
  if optionTraceEntering {
    fmt.Fprintf(stderr, "entering: %v (%v)\n", dir, prog.project.name)
  }

  var wd string
  if wd, err = os.Getwd(); err != nil { return }
  if err = lockCD(dir, 0); err != nil { return }
  if !filepath.IsAbs(dir) { dir = filepath.Join(wd, dir) }
  prog.auto("CWD", &String{trivial{prog.position},dir})

  var ( enter *enterec ; ok bool )
  if enter, ok = cd.enters[dir]; !ok {
    enter = &enterec{ wd:wd, dir:dir }
    cd.enters[dir] = enter
  }
  enter.num += 1
  cd.stack = append([]*enterec{enter}, cd.stack...)
  return
}

func leave(prog *Program, stop *enterec) (err error) {
  var size = len(cd.stack)
  if optionTraceEntering {
    fmt.Fprintf(stderr, "leaving: %v (%v %v %v)\n", stop.dir, prog.project.name, stop.num, size)
  }

  for _, enter := range cd.stack {
    if enter.num == 0 { continue } else {
      enter.num -= 1
    }
    if enter == stop {
      if enter.print && false {
        enter.print = false
        fmt.Fprintf(stderr, "smart:  Leaving directory '%s'\n", enter.dir)
      }
      err = lockCD(enter.wd, 0)
      break
    }
  }

  // Erase 'zero' and unprint records, the first record is always kept.
  // So that the right entering/leaving pairs are printed.
  if size > 1 {
    var stack = []*enterec{ cd.stack[0] }
    for i := 1; i < size; i += 1 {
      var rec = cd.stack[i]
      if rec.num > 0 || rec.print {
        stack = append(stack, rec)
      }
    }
    cd.stack = stack
  }
  return
}

func printEnteringDirectory() {
  if size := len(cd.stack); size > 0 {
    var enter = cd.stack[0]
    if enter.silent { return }
    for _, p := range cd.stack {
      if p.print && p != enter {
        p.print = false
        fmt.Fprintf(stderr, "smart:  Leaving directory '%s'\n", p.dir)
      }
    }
    if !enter.print {
      enter.print = true
      fmt.Fprintf(stderr, "smart: Entering directory '%s'\n", enter.dir)
    }
  }
}

func printLeavingDirectory() {
  if size := len(cd.stack); size > 0 {
    for _, enter := range cd.stack {
      if enter.print {
        enter.print = false
        fmt.Fprintf(stderr, "smart:  Leaving directory '%s'\n", enter.dir)
      }
    }
  }
}
