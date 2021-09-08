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
  project *Project
  pattern Value
  Paths []Value
}

func (filemap *FileMap) String() string { return filemap.pattern.String() }
func (filemap *FileMap) Patterns(ctx Context) (pats []Value) {
  if filemap.pattern.expandible(ctx, expandClosure) {
    var err error
    if pats, err = mergeresult2(expandall2(ctx, expandPlainValue, filemap.pattern)); err != nil {
      diag.errorOf(filemap.pattern, "merge pattern '%v' failed: %v", filemap.pattern, err)
    } else if pats, _, err = expandall2(ctx, expandPlainValue, pats...); err != nil {
      // Do a second expand to ensure converted closures are expanded
      diag.errorOf(filemap.pattern, "sencond expand patterns '%v' failed: %v", pats, err)
    }
  } else {
    pats = append(pats, filemap.pattern)
  }
  return merge(pats...)
}

// Match split filename into list and match each part with the pattern correspondingly.
func (filemap *FileMap) Match(ctx Context, filename string) (matched bool, pre string) {
  /*if filemap.Pattern.expandible(ctx, expandClosure) {
    if pats, err := mergeresult2(expandall2(ctx, expandPlainValue, filemap.Pattern)); err != nil {
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
  for _, pat := range filemap.Patterns(ctx) {
    if matched, pre = filemap.match(ctx, pat, filename); matched { break }
  }
  return
}

func (filemap *FileMap) match(ctx Context, pattern Value, filename string) (matched bool, pre string) {
  if matched, pre = globMatch(ctx, pattern, filename); matched { return }
  return
}

func (filemap *FileMap) stat(ctx Context, base, pre, name string) (file *File) {
  var pos = filemap.pattern.Position()
  if filemap.Paths == nil {
    // Check file in the filesystem (no paths).
    file = stat(ctx, name, "", base, nil)
    return
  }
  base = filepath.Clean(base)
  pre  = filepath.Clean(pre)
  for _, path := range filemap.Paths {
    if path == nil {
      diag.errorAt(pos, "mapping nil path (base=%s, pre=%s, name=%s)", base, pre, name).debug(32)
      panic("internal error")
    }

    var ( dir, sub string ; err error )
    if sub, err = path.Strval(ctx); err != nil {
      diag.errorAt(pos, "strval '%v' failed: %v", path, err).debug(1)
      return
    } else {
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
    if file = stat(ctx, name, sub, dir, nil); file != nil {
      break
    }

    if filepath.IsAbs(sub) {
      if pre == "" { // Fullmatch!
        // For example of:
        //   xxx.c  <->  (*.c => /path/to/source)
        // Become:
        //   /path/to/source  ""  xxx.c
        file = stat(ctx, name, "", sub, nil)
      } else if strings.HasSuffix(sub, PathSep+pre) {
        // For example of:
        //   foo/bar/xxx.c  <->  (*.c => /path/to/source/foo/bar)
        // Become:
        //   /path/to/source  foo/bar  xxx.c
        s := strings.TrimSuffix(sub, PathSep+pre)
        n := strings.TrimPrefix(name, pre+PathSep)
        file = stat(ctx, n, pre, s, nil)
      } else if false { // This is wrong, only base name matched!!
        // For example of:
        //   foo/bar/xxx.c  <->  (*.c => /path/to/source)
        // Become:
        //   /path/to/source  foo/bar  xxx.c
        n := strings.TrimPrefix(name, pre+PathSep)
        file = stat(ctx, n, pre, sub, nil)
      }
    } else {
      if pre == "" { // Fullmatch!
        // For example of:
        //   xxx.c  <->  (*.c => source)
        // Become:
        //   <p.absPath>  source  xxx.c
        file = stat(ctx, name, sub, dir, nil)
      } else if sub == pre {
        // For example of:
        //   foo/bar/xxx.c  <->  (*.c => foo/bar)
        // Become:
        //   <dir>  foo/bar  xxx.c
        n := strings.TrimPrefix(name, pre+PathSep)
        file = stat(ctx, n, sub, dir, nil)
      } else if strings.HasSuffix(sub, PathSep+pre) {
        // For example of:
        //   foo/bar/xxx.c  <->  (*.c => source/foo/bar)
        // Become:
        //   <dir>  source/foo/bar  xxx.c
        s := strings.TrimSuffix(sub, PathSep+pre)
        n := strings.TrimPrefix(name, pre+PathSep)
        file = stat(ctx, n, pre, s, nil)
      } else if false { // This is wrong, only base name matched!!
        // For example of:
        //   foo/bar/xxx.c  <->  (*.c => source)
        // Become:
        //   <dir>  source/foo/bar  xxx.c
        s := filepath.Join(sub, pre)
        n := strings.TrimPrefix(name, pre+PathSep)
        file = stat(ctx, n, s, dir, nil)
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
func globMatch(ctx Context, patval Value, filename string) (matched bool, pre string) {
  switch patval.(type) {
  default: // good to go!
  case *List:
    diag.errorOf(patval, "invalid glob matching pattern: %v (%T)", patval, patval)
    return
  }

  pattern, err := patval.Strval(ctx)
  if err != nil { return false, "" }

  list0 := strings.Split(filepath.Clean(pattern ), PathSep)
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
  mutex sync.Mutex
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

  opts projectDeclOpts
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
  p._filemap_ = append(p._filemap_, &FileMap{ p, pat, paths })
}
func (p *Project) myFilemaps(ctx Context) (filemap []*FileMap) {
  if true { return p._filemap_ }

  var _filemap_ []*FileMap
  if len(_filemap_) == 0 && len(p._files_) > 0 {
    var mapfile = func (pat Value, paths []Value) {
      // List order is significant, duplication is acceptable.
      _filemap_ = append(_filemap_, &FileMap{ p, pat, paths })
    }
    for _, spec := range p._files_ {
      switch v := spec.(type) {
      case *Pair:
        var pats, paths []Value
        switch k := v.Key.(type) {
        case *Group: pats = k.Elems
        default:     pats = append(pats, v.Key)
        }
        if a, err := mergeresult2(expandall2(ctx, expandPlainValue, pats...)); err != nil {
          diag.errorAt(v.Position(), "error expanding '%v': %v", v, err)
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
        paths = []Value{&String{valbase{v.Position()},p.absPath}}
        switch g := v.(type) {
        default: pats = append(pats, v)
        case *Group: pats = g.Elems
        }
        for _, k := range pats { mapfile(k, paths) }
      default:
        diag.errorOf(v, "invalid file spec: %v", v)
      }
    }
  }
  return _filemap_
}

func (p *Project) filemaps(ctx Context, using bool) (filemaps []*FileMap) {
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
          diag.warnOf(m.pattern,  "                     with: %v (paths=%v)" , m, m.Paths)
          diag.warnOf(a.Paths[0], "          differred paths: %v", a.Paths[0])
          diag.warnOf(m.Paths[0], "                      and: %v", m.Paths[0])
          numDuplicated += 1
        }
      }
    }
    if numDuplicated > 0 { diag.errorOf(a.pattern, "duplicated files: %v", a.pattern) }
    filemaps = append(filemaps, a)
  }

  for _, m := range p.myFilemaps(ctx) { uniqueAppend(m) }
  for _, base := range p.bases {
    for _, m := range base.filemaps(ctx, /*using*/false) { uniqueAppend(m) }
  }
  if using {
    // takes a big longer time to map usee filemaps, but acceptable
    var appendUsingList func(*Project)
    appendUsingList = func(p *Project) {
      for _, m := range p.myFilemaps(ctx) { uniqueAppend(m) }
      for _, u := range p.using.list {
        if u.opts.noFiles { appendUsingList(u.project) }
      }
    }
    appendUsingList(p)
  }
  return
}

func (p *Project) wildcard(ctx Context, opts wildcardOpts, patterns ...Value) (files []*File, err error) {
  var pos = ctx.Position()
  var filemaps = p.filemaps(ctx, false)
ForPatterns:
  for _, pat := range patterns {
    var (
      breakAbsRel bool
      matched bool
      patStr string
    )
    if patStr, err = pat.Strval(ctx); err != nil {
      diag.errorAt(pos, "strval '%v' failed: %v", pat, err).debug(1)
      break ForPatterns
    }
    // The 'patStr' could be GlobPattern or just regular file/path names. PercPattern is not supported yet.
  ForFilemaps:
    for _, fm := range filemaps {
      for _, pattern := range fm.Patterns(ctx) {
        var pre string // <pre>/*.xxx
        var str = patStr
        if matched, pre = globMatch(ctx, pattern, patStr); !matched {
          // Flip glob matching order.
          if _, yes := pat.(*GlobPattern); !yes {
            continue ForFilemaps
          } else if str, err = pattern.Strval(ctx); err != nil {
            break ForPatterns
          } else if matched, pre = globMatch(ctx, pat, str); !matched {
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
          if names, err = filepath.Glob(str); err != nil { break ForPatterns }
          for _, s := range names {
            file := stat(ctx, filepath.Base(s), "", filepath.Dir(s))
            files = append(files, file)
            if enable_assertions {
              assert(file != nil, "`%s` missing", s)
            }
          }
          if breakAbsRel {
            continue ForPatterns
          } else {
            continue ForFilemaps
          }
        }

        // Check against paths for non-abs/rel patterns.
        for _, path := range fm.Paths {
          var sub string
          if sub, err = path.Strval(ctx); err != nil {
            break ForPatterns
          }

          subfile := filepath.Join(sub, str)
          if names, err = filepath.Glob(subfile); err != nil {
            break ForPatterns
          }
          // Chop off path 'sub' prefix to have shorter names
          // Aka. trim prefix 'file.Sub+PathSep'
          prefix := strings.TrimSuffix(subfile, str)
          if len(names) > 0 {
            for _, s := range names {
              if false && s == "..." {
                diag.warnAt(pos, "sub = %s", sub)
                diag.warnAt(pos, "pat = %s", pat)
                diag.warnAt(pos, "pattern = %v", pattern)
                diag.warnAt(pos, "pre = %s", pre)
                diag.warnAt(pos, "str = %s", str)
                diag.warnAt(pos, "subfile = %s", subfile)
                diag.warnAt(pos, "prefix = %s", prefix)
                diag.warnAt(pos, "name = %s", s).debug(16)
              }
              name := strings.TrimPrefix(s, prefix)
              file := stat(ctx, name, sub, prefix)
              files = append(files, file)
              if enable_assertions {
                assert(file != nil, "`%s` missing (%s)", s, name)
              }
            }
          } else if ok := pattern.patterned(ctx); !ok && opts.includeMissing {
            // If the filemap is not a pattern (e.g. foobar.cpp), we include it in the returning files
            var name string
            name, err = pattern.Strval(ctx)
            if err != nil { break ForPatterns }

            // Append this non-existed/missing file.
            file := stat(ctx, name, sub, prefix, nil)
            files = append(files, file)
          } else if ok && !path.expandible(ctx, expandClosure) && len(fm.Paths) == 1 {
            // Just report that the pattern matches no files in the
            // file system (if only one path specified).
            if false {
              diag.warnOf(pattern, "%s: %v matches no files in '%v'", p.name, fm, sub)
              diag.warnOf(    pat, "%s: here is %v (try using flag -m, aka -include-missing)", p.name, pat).
                debug(1)
            }
          } else if opts.errorMissing {
            err = fmt.Errorf("missing files like '%v'", fm)
            break ForPatterns
          }
        }
      }
    }
  }
  return
}

func (p *Project) matchFile(ctx Context, name string) (file *File) {
  if optionEnableBenchmarks && false { defer bench(mark("Project.FindFile")) }
  if optionEnableBenchspots { defer bench(spot("Project.FindFile")) }

  var first *File

ForFilemaps:
  for _, filemap := range p.filemaps(ctx, true) {
    // Match the represented file name.
    var matched, pre = filemap.Match(ctx, name)
    if !matched { continue ForFilemaps }
    if p.changedWD != "" { file = filemap.stat(ctx, p.changedWD, pre, name) }
    if file == nil       { file = filemap.stat(ctx, p.absPath,   pre, name) }
    if false {
      var s1, s2 string
      var pos = filemap.pattern.Position()
      if file  != nil { s1 = file.fullname() }
      if first != nil { s2 = first.fullname() }
      diag.errorAt(pos, "%s: name=%s (file=%v, exists=%v, first=%v, cwd=%s, filemap=%v, patterns=%v, pre=%v)\n",
        p, name, s1, file.exists(), s2, p.changedWD, filemap.pattern, filemap.Patterns(ctx), pre).
        debug(1)
    }
    if file != nil {
      if file.filemap == nil { file.filemap = filemap }
      if pre != "" { /* FIXME: file.change(...pre) */ }
      if file.exists() { break ForFilemaps }
      if first == nil { first = file }
      file = nil // reset for the next match
    }
    // If the filemap entry is defined by the project itself,
    // we have to break the matching loop. So that the current
    // project have a chance to define it's own file. This is
    // usefull when the bases (or imported projects) have also
    // matched files. The current project have the highest
    // priority to match.
    for _, fm := range p.myFilemaps(ctx) {
      if filemap == fm { break ForFilemaps }
    }
  }
  if first != file && !file.exists() { file = first }
  return
}

func (p *Project) matchTempFile(ctx Context, name string) (file *File) {
  var pos = ctx.Position()
  if file = p.matchFile(ctx, name); file != nil {
    // good
  } else if ctd := p.scope.FindDef("CTD"); ctd == nil {
    diag.errorAt(pos, "%v: CTD is not defined for temp file: %v", p, name).debug(1)
  } else if s, err := ctd.Strval(ctx); err != nil {
    diag.errorAt(pos, "%v: stringify temp directory failed: %v", p, err).debug(1)
  } else if file = stat(ctx, filepath.Join(s, name), "", "", nil); file == nil {
    diag.errorAt(pos, "%v: nil stat %v %v", p, s, name).debug(1)
  } else if false {
    if !pos.IsValid() { pos = p.position }
    diag.warnAt(pos, "using default temp file: %v/%v", s, name)
    diag.warnAt(p.position, "suggesting define files rule for '%s' in %v", name, p).debug(12)
  }
  return // NOTE: temp file may not exists
}

func (p *Project) configurationFile(ctx Context) (file *File) {
    if file = p.matchTempFile(ctx, "configuration.sm"); file == nil {
        diag.errorAt(p.position, "%v: no file configuration.sm", p).
            debug(1)
    }
    return
}

func (p *Project) FindFile(ctx Context, name string) (file *File) { return p.matchFile(ctx, name) }

func (p *Project) isFileName(ctx Context, s string) (res bool) {
  if len(s) > 0 {
    for _, filemap := range p.filemaps(ctx, true) {
      if res, _ = filemap.Match(ctx, s); res { break }
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

func (p *Project) resolveObject(ctx Context, s string) (obj Object, err error) {
  if _, obj = p.scope.Find(s); obj == nil {
    if p.pluginScope != nil {
      if obj = p.pluginScope.Lookup(s); obj != nil {
        return
      }
    }
    for _, base := range p.bases {
      obj, err = base.resolveObject(ctx, s)
      if err != nil || obj != nil {
        break
      }
    }
  } else if obj != nil && false {
    var s string = p.scope.comment
    for o := p.scope.outer; o != nil; o = o.outer {
      s += " -> " + o.comment
    }
    diag.infoOf(obj, "%v => %v, %v, %v", p, obj, obj.OwnerProject(), s)
  }
  return
}

func (p *Project) resolveEntry(ctx Context, s string, matchFullSuffix bool) (entry *RuleEntry, err error) {
  if optionEnableBenchmarks && false { defer bench(mark("Project.resolveEntry")) }
  if optionEnableBenchspots { defer bench(spot("Project.resolveEntry")) }
  for _, rec := range p.concrete {
    switch target := rec.target.(type) {
    case *File:
      if false && s == "llvm/IR/Attributes.inc" && target.name == "Attributes.inc" {
        var ok = (!filepath.IsAbs(s) && strings.HasSuffix(target.fullname(), filepath.Clean(s)))
        var full, res, stems = target.match(ctx, s)
        diag.warnAt(rec.Position(), "%v: %s <-> %v => %v, %v, %v, %v", p, s, target, full, res, stems, ok).
          debug(1)
      }
      if target.name == s {
        return rec, nil
      } else if filepath.IsAbs(s) && s == target.fullname() {
        return rec, nil // fullname matched
      } else if !matchFullSuffix {
        // not matching
      } else if strings.HasSuffix(target.fullname(), PathSep+filepath.Clean(s)) {
        if false { diag.warnAt(rec.Position(), "TODO: %v: %s <-> %v %v", p, s, target, target.fullname()).
          debug(8) }
        return rec, nil
      }
    //case *Path:
    default:
      var sv string
      if sv, err = target.Strval(ctx); err != nil {
        diag.errorAt(target.Position(), "strval '%v' failed: %v", target, err).debug(1)
        return
      } else if sv == s { return rec, nil }
    }
  }
  for _, base := range p.bases {
    if entry, err = base.resolveEntry(ctx, s, matchFullSuffix); entry != nil || err != nil { break }
  }
  if err == nil && entry == nil {
    if true { /* FAST */ } else { /* SLOW */
      for _, using := range p.using.list {
        entry, err = using.project.resolveEntry(ctx, s, matchFullSuffix)
        if err != nil || entry != nil { break }
      }
    }
  }
  return
}

func (p *Project) resolvePatterns(ctx Context, i interface{}) (res []*stemmed) {
  if optionEnableBenchmarks && false { defer bench(mark("Project.resolvePatterns")) }
  if optionEnableBenchspots { defer bench(spot("Project.resolvePatterns")) }
  res   = p._resolvePatterns1(ctx, i)
  if v := p._resolvePatterns2(ctx, i); len(v) > 0 { res = append(res, v...) }
  if true { /* FAST */ } else /* SLOW */
  if v := p._resolvePatterns3(ctx, i); len(v) > 0 { res = append(res, v...) }
  return
}

func (p *Project) _resolvePatterns1(ctx Context, i interface{}) (res []*stemmed) {
  if optionEnableBenchspots { defer bench(spot("Project._resolvePatterns1")) }
  for _, pat := range p.patterns {
    if full, _, stems := pat.Pattern.match(ctx, i); full {
      res = append(res, &stemmed{pat, stems})
    }
  }
  return
}

func (p *Project) _resolvePatterns2(ctx Context, i interface{}) (res []*stemmed) {
  if optionEnableBenchspots { defer bench(spot("Project._resolvePatterns2")) }
  for _, base := range p.bases {
    res = append(res, base.resolvePatterns(ctx, i)...)
  }
  return
}

func (p *Project) _resolvePatterns3(ctx Context, i interface{}) (res []*stemmed) {
  if optionEnableBenchspots { defer bench(spot("Project._resolvePatterns3")) }
  for _, using := range p.using.list {
    res = append(res, using.project.resolvePatterns(ctx, i)...)
  }
  return
}

type entryOpts struct {
  postExec bool `p,post;pe,post-execute;pe,post-exec`
}
func (p *Project) entry(ctx Context, special specialRule, options []Value, target Value, prog *Program) (entry *RuleEntry, err error) {
  defer func() {
    if entry != nil && err == nil {
      entry.programs = append(entry.programs, prog)
    }
  } ()

  var strval string
  if strval, err = fullnameOrStrval(ctx, target); err != nil {
    return
  }

  // The 'use' rule entries.
  var closured = target.expandible(ctx, expandClosure)
  if special == specialRuleUse && !closured {
    var opts entryOpts
    if len(options) > 0 {
      var pos = options[0].Position()
      if _, err = parseOpts(ctx, &opts, options...); err != nil {
        diag.errorAt(pos, "parse opts failed: %v", err)
        return
      }
    }
    var userule = &useRuleEntry{
      RuleEntry{ class:UseRuleEntry, target:target },
      opts.postExec, // post-execute use rule?
    }
    p.userules = append(p.userules, userule)
    entry = &userule.RuleEntry
    return
  }

  var name string
  if name, err = fullnameOrStrval(ctx, target); err != nil {
    return
  } else if name == "" {
    err = fmt.Errorf("name '%v' already taken as `%T'", name)
    return
  }

  // Looking for pattern rule entries.
  switch t := target.(type) {
  case *PercPattern:
    assert(t != nil, "nil PercPattern")
    if false {
      for _, pe := range p.patterns {
        if pe.Pattern.cmp(ctx, t) == cmpEqual {
          entry = pe.RuleEntry
          return
        }
      }
    }
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
    }
    if sv, err = fullnameOrStrval(ctx, rec); err != nil {
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
  return p.hasLoadedRecur(p, proj, 1, breakUseLoop)
}

func (p *Project) hasLoadedRecur(top, proj *Project, depth int, breakUseLoop bool) (rp *Project, res, isb bool, err error) {
  if depth > 1 && top == p && true {
    err = fmt.Errorf("loop '%v' (depth=%d)", p.loopLoadPath(), depth)
    diag.errorAt(p.position, "%v: %v", p, err).debug(128)
    return
  } else if depth > 128 {
    err = fmt.Errorf("exceeds maximum base depth (%d) (start=%v, target=%v)", depth, top, proj)
    diag.errorAt(p.position, "%v: %v", p, err)
    diag.errorAt(top.position, "start: %v", top)
    diag.errorAt(proj.position, "target: %v", proj).debug(200)
    return
  }
  for _, base := range p.bases {
    if isb = base == proj; isb { return }
    if rp, res, isb, err = base.hasLoadedRecur(top, proj, depth+1, breakUseLoop); err != nil {
      return
    } else if res || isb { rp = base ; return }
  }
  for _, imp := range p.loads {
    if imp == top && !breakUseLoop {
      s := top.loopLoadPath()
      err = fmt.Errorf("loop `%v`", s)
      diag.errorAt(top.position, "start: %v", top)
      diag.errorAt(proj.position, "stop: %v", proj)
      diag.errorAt(p.position, "%v: %v", p, err).
        debug(128)
      return
    }
    if res = imp == proj; res { rp = imp; return }
    if rp, res, res, err = imp.hasLoadedRecur(top, proj, depth+1, breakUseLoop); err != nil {
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
      if p != top { s = "⇢" }
      s += fmt.Sprintf("(%s)⇢(%s)", p.spec, imp.spec)
      break
    }
    if t := imp.loopLoadRecur(top); t != "" {
      if p != top { s = "⇢" }
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
  if p.opts.breakUseLoop { return }
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

func enter(t *traversal, dir string) (err error) {
  cd.mutex.Lock(); defer cd.mutex.Unlock()

  if optionTraceEntering {
    fmt.Fprintf(stderr, "entering: %v (%v)\n", dir, t.project.name)
  }

  var wd string
  if wd, err = os.Getwd(); err != nil { return }
  if err = lockCD(dir, 0); err != nil { return }
  if !filepath.IsAbs(dir) { dir = filepath.Join(wd, dir) }
  t.auto("CWD", &String{valbase{t.program.position},dir})

  var ( enter *enterec ; ok bool )
  if enter, ok = cd.enters[dir]; !ok {
    enter = &enterec{ wd:wd, dir:dir }
    cd.enters[dir] = enter
  }
  enter.num += 1
  cd.stack = append([]*enterec{enter}, cd.stack...)
  return
}

func leave(ctx Context, prog *Program, stop *enterec) (err error) {
  cd.mutex.Lock(); defer cd.mutex.Unlock()

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
        diag.prompt("smart:  Leaving directory '%s'\n", enter.dir)
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
  cd.mutex.Lock(); defer cd.mutex.Unlock()
  if size := len(cd.stack); size > 0 {
    var enter = cd.stack[0]
    if enter.silent { return }
    for _, p := range cd.stack {
      if p.print && p != enter {
        p.print = false
        diag.prompt("smart:  Leaving directory '%s'\n", p.dir)
      }
    }
    if !enter.print {
      enter.print = true
      diag.prompt("smart: Entering directory '%s'\n", enter.dir)
    }
  }
}

func printLeavingDirectory() {
  cd.mutex.Lock(); defer cd.mutex.Unlock()
  if size := len(cd.stack); size > 0 {
    for _, enter := range cd.stack {
      if enter.print {
        enter.print = false
        diag.prompt("smart:  Leaving directory '%s'\n", enter.dir)
      }
    }
  }
}
