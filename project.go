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
  paths []Value
  public bool
}

func (filemap *FileMap) String() string { return filemap.pattern.String() }
func (filemap *FileMap) Patterns(ctx Context) (pats []Value) {
  if filemap.pattern.expandible(ctx, expandClosure) {
    var err error
    if pats, err = expandmerge2(ctx, expandPlainValue, filemap.pattern); err != nil {
      erro(ctx, "merge pattern '%v' failed: %v", filemap.pattern, err).of(filemap.pattern)
    } else if pats, _, err = expandall2(ctx, expandPlainValue, pats...); err != nil {
      // NOTE: do a second expand to ensure converted closures are expanded
      erro(ctx, "sencond expand patterns '%v' failed: %v", pats, err).of(filemap.pattern)
    }
  } else {
    pats = append(pats, filemap.pattern)
  }
  return merge(pats...)
}

// Match split filename into list and match each part with the pattern correspondingly.
func (filemap *FileMap) Match(ctx Context, str string) (matched bool, pre string) {
  for _, pat := range filemap.Patterns(ctx) {
    if matched, pre = filemap.match(ctx, pat, str); matched { break }
  }
  return
}

func (filemap *FileMap) match(ctx Context, pat Value, str string) (matched bool, pre string) {
  if matched, _, _ = pat.match(ctx, str); !matched && !(isNone(pat) || isNil(pat)) {
    if n := strings.Index(str, PathSep); n < 0 { return }

    // NOTE: Dealing with these files:
    //     files (
    //         (foo.c) => $(srcdir)/sub/dir
    //         (sub/dir/foo.c) => $(srcdir)
    //     )
    for _, p := range filemap.paths { // FIXME: performance, operate on p.(*Path) instead
      if _, ok := p.(*Path); !ok { continue } // NOTE: only work with paths to improve performance
      var ps, err = p.Strval(ctx)
      if err != nil {
        erro(ctx, "%v: strval `%v` failed: %v", filemap, p, err).debug(1)
        return
      }
      for i := strings.LastIndex(ps, PathSep); -1 <= i; {
        var ( prefix = ps[i+1:]; l = len(prefix) ) // NOTE: -1 <= i < len(ps)
        if has := strings.HasPrefix(str, prefix) && str[l] == '/'; has {
          if false && strings.HasSuffix(ps, "/clang/AST") && str == "clang/AST/CommentCommandList.inc" {
            v, s, t := pat.match(ctx, str[len(prefix)+1:])
            warnstack(ctx, 8, "%v, %v: %v: %v: %v, %v %v %v", p, filemap, str, prefix, has, v, s, t).debug(1)
          }
          if matched, _, _ = pat.match(ctx, str[len(prefix)+1:]); matched { break }
        }
        if 0 < i { i = strings.LastIndex(ps[:i], PathSep) } else { break }
      }
    }
  }
  return // NOTE: also `globMatchFile(ctx, pat, str, true)`
}

func (filemap *FileMap) stat(ctx Context, base, pre, name string) (file *File) {
  var pos = filemap.pattern.Position(); if false { ctx = positional(ctx, pos) }
  if base = filepath.Clean(base); len(filemap.paths) == 0 {
    file = stat(ctx, name, "", base, nil) // simply stat file name if no paths
    return
  } else if pre != "" { pre = filepath.Clean(pre) }
  for _, path := range filemap.paths {
    if isNil(path) {
      erro(ctx, "nil path: base=%s)", base).at(pos)
      erro(ctx, "nil path: pre=%s",   pre).at(pos)
      erro(ctx, "nil path: name=%s",  name).at(pos)
      erro(ctx, "nil path: %v", filemap).at(pos).debug(32)
      fail(pos, "file mapping nil path: %v", filemap)
    } else if isNone(path) {
      warn(ctx, "nil path: base=%s)", base).at(pos)
      warn(ctx, "nil path: pre=%s",   pre).at(pos)
      warn(ctx, "nil path: name=%s",  name).at(pos)
      warn(ctx, "nil path: %v", filemap).at(pos).debug(32)
      continue
    }

    var (
      sub string
      err error
    )
    if sub, err = path.Strval(ctx); err != nil {
      erro(ctx, "strval path failed: %v", err).at(path.Position())
      erro(ctx, "strval '%v' failed: %v", path, err).at(pos)
      erro(ctx, "strval '%v' failed (project=%s)", path, ctx.Project())//.at(pos)
      erro(ctx, "strval '%v' failed in %v", ctx).debug(32)
      return
    } else if sub == "" {
      erro(ctx, "filemap path '%v' is empty", path).at(path.Position())
      erro(ctx, "filemap path '%v' is empty (pattern=%v)", path, filemap.pattern).at(pos)
      erro(ctx, "filemap path '%v' is empty (project=%v)", path, ctx.Project())//.at(pos)
      erro(ctx, "filemap path '%v' is empty in %v", path, ctx).debug(64)
      return
    } else if s := filepath.Clean(sub); sub != s {
      if false {
        erro(ctx, "filemap path '%v' is not clean (sub=%s)", path, sub).at(path.Position())
        erro(ctx, "filemap path '%v' is not clean (pattern=%v)", path, filemap.pattern).at(pos)
        erro(ctx, "filemap path '%v' is not clean (project=%v)", path, ctx.Project())//.at(pos)
        erro(ctx, "filemap path '%v' is not clean in %v", ctx).debug(16)
        return
      } else {
        sub = s
      }
    }

    var dir string
    if filepath.IsAbs(sub) {    // 'sub' is abs
      if filepath.IsAbs(name) { // 'name' is abs too
        if strings.HasPrefix(name, sub) { // 'name' should have 'sub' prefix
          if true {
            warn(ctx, "sub  = %v", sub).at(pos)
            warn(ctx, "name = %v", name).at(pos)
            warn(ctx, "%v", ctx).debug(6)
            name = strings.TrimPrefix(name, sub)
          }
        } else {
          if false {
            warn(ctx, "sub  = %v", sub).at(pos)
            warn(ctx, "name = %v", name).at(pos)
            warn(ctx, "%v", ctx).debug(6)
          }
          continue
        }
      }
    } else if !filepath.IsAbs(name) { dir = base }

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

// globMatchFile - Glob matching each component of the filename against the
// glob value. It checks in two different ways. If the filename and the
// glob pattern has the some number of components (splitted by PathSep),
// all components are compared. If the pattern has only one component,
// the last filename component is compared with the pattern, and the prefix
// components are returned in 'pre'.
func globMatchFile(ctx Context, patVal Value, filename string, tailMatch bool) (matched bool, pre string) {
  switch patVal.(type) {
  default: // good to go!
  case *List:
    erro(ctx, "invalid glob matching pattern: %v", patVal).of(patVal).debug(8)
    return
  }

  var patStr, err = patVal.Strval(ctx)
  if err != nil { return false, "" }
  if false && strings.HasSuffix(filename, "...") {
    defer func() {
      if matched {
        prompt(ctx, "%v: %v (%v)\n", filename, patVal, ctx.Project())
        warnstack(ctx, 6, "%T %v %v; %v", patVal, patVal, filename, pre).of(patVal).debug(24)
      }
    } ()
  }

  var patList = strings.Split(filepath.Clean(patStr), PathSep)
  if len(patList) == 0 { return } // FIXME: match any?

  var srcList = strings.Split(filepath.Clean(filename), PathSep)
  if len(patList) == len(srcList) { // src/*.o  <->  src/foo.o
    for i, pat := range patList { // Matching all components
      if matched, _ = filepath.Match(pat, srcList[i]); !matched { return }
    }
  } else if len(patList) == 1 && len(srcList) > 1 { // *.o|foo.o  <->  src/foo.o
    // NOTE: partially matching only the last part is logically incorrect!
    //       for example of this wrong match: stdint.h <-> isl/stdint.h
    if tailMatch && true {
      for i, j := len(patList)-1, len(srcList)-1; -1 < i && -1 < j; i, j = i-1, j-1 {
        if v, _ := filepath.Match(patList[i], srcList[j]); !v {
          pre = filepath.Join(srcList[:j]...)
          return
        } else {
          matched = true
        }
      }
    } else if tailMatch && false {
      if matched, _ = filepath.Match(patList[0], srcList[len(srcList)-1]); matched {
        pre = filepath.Join(srcList[:len(srcList)-1]...)
        return
      }
    } else if matched, _ = filepath.Match(patList[0], filename); matched {
      return
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
  //loads []*Project
  using   *usinglist

  // List order is significant, duplication is acceptable.
  _files_ []Value
  _filemap_ []*FileMap

  // Rule Registry (orderred)
  userules []*useRuleEntry // the 'use' rule
  concrete []Entry //*RuleEntry
  patterns []*PatternEntry

  filescopes []*Scope

  // TODO: printEntering() ...
  // TODO: printLeaving() ...

  plugin *plugin.Plugin
  pluginScope *Scope

  opts projectDeclOpts
}

func (p *Project) AbsPath() string { return p.absPath }
func (p *Project) RelPath() string { return p.relPath }
func (p *Project) Spec() string { return p.spec }
func (p *Project) String() string { return p.name }
func (p *Project) Name() string { return p.name }
func (p *Project) Scope() *Scope { return p.scope }
func (p *Project) Bases() []*Project { return p.bases }

func (p *Project) NewScope(pos Position, comment string) *Scope {
  return NewScope(pos, p.scope, p, comment)
}

func (p *Project) mapfile(ctx Context, opts filesOpts, pat Value, paths []Value) {
  // List order is significant, duplication is acceptable.
  p._filemap_ = append(p._filemap_, &FileMap{ p, pat, paths, opts.public })
}

func (p *Project) myFilemaps(ctx Context) (filemap []*FileMap) {
  return p._filemap_

  var _filemap_ []*FileMap
  if len(_filemap_) == 0 && len(p._files_) > 0 {
    var mapfile = func (pat Value, paths []Value) {
      // List order is significant, duplication is acceptable.
      _filemap_ = append(_filemap_, &FileMap{ p, pat, paths, false })
    }
    for _, spec := range p._files_ {
      switch v := spec.(type) {
      case *Pair:
        var pats, paths []Value
        switch k := v.Key.(type) {
        case *Group: pats = k.Elems
        default:     pats = append(pats, v.Key)
        }
        if a, err := expandmerge2(ctx, expandPlainValue, pats...); err != nil {
          erro(ctx, "error expanding '%v': %v", v, err).at(v.Position())
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
        erro(ctx, "invalid file spec: %v", v).of(v)
      }
    }
  }
  return _filemap_
}

func appendFilemapUniquely(ctx Context, filemaps []*FileMap, a *FileMap) (result []*FileMap) {
  var numDuplicated int
  for _, m := range filemaps {
    if a == m || (a.pattern == m.pattern && &a.paths == &m.paths) {
      result = filemaps
      return
    } else if a.pattern == m.pattern && len(a.paths) == len(m.paths) {
      var same = true // initially assumes all paths are identical
      for i, ap := range a.paths {
        if ap != m.paths[i] { same = false; break }
      }
      if same {
        result = filemaps
        return
      } else {
        warn(ctx, "files might be duplicated: %v (paths=%v),", a, a.paths).of(a.pattern)
        warn(ctx, "                     with: %v (paths=%v)" , m, m.paths).of(m.pattern)
        warn(ctx, "          differred paths: %v", a.paths[0]).of(a.paths[0])
        warn(ctx, "                      and: %v", m.paths[0]).of(m.paths[0])
        numDuplicated += 1
      }
    }
  }
  if numDuplicated > 0 { erro(ctx, "duplicated files: %v", a.pattern).of(a.pattern) }
  result = append(filemaps, a)
  return
}

func (p *Project) filemaps(ctx Context, baseFiles, usedFiles bool) (filemaps []*FileMap) {
  if optionEnableBenchmarks && false { defer bench(mark("Project.filemaps")) }

  var appendUnique = func(a *FileMap) {
    filemaps = appendFilemapUniquely(ctx, filemaps, a)
  }

  for _, m := range p.myFilemaps(ctx) { appendUnique(m) }
  if baseFiles { for _, base := range p.bases {
    for _, m := range base.filemaps(ctx, true, usedFiles) { appendUnique(m) }
  }}
  if usedFiles && false/* FIXME: performance */ {
    // takes a big longer time to map usee filemaps, but acceptable
    var (
      appendUsingList func(*Project)
      appendUsedFiles func(*Project)
    )
    appendUsingList = func(p *Project) {
      var fms []*FileMap
      if true {
        fms = p.myFilemaps(ctx)
      } else {
        // FIXME: this is the expensive way, really slow!
        fms = p.filemaps(ctx, baseFiles, usedFiles)
      }
      for _, m := range fms { appendUnique(m) }
      appendUsedFiles(p)
    }
    appendUsedFiles = func(p *Project) {
      for _, u := range p.using.list {
        appendUsingList(u.project)
      }
    }
    appendUsedFiles(p)
  }
  return
}

func (p *Project) wildcard(ctx Context, opts wildcardOpts, patterns ...Value) (files []*File, err error) {
  var filemaps = p.filemaps(ctx, opts.baseFiles, opts.usedFiles)
ForPatterns:
  for _, pat := range patterns {
    var (
      patPatterned = pat.patterned(ctx)
      breakAbsRel bool
      matched bool
    )
  ForFilemaps:
    for _, fm := range filemaps {
      for _, pattern := range fm.Patterns(ctx) {
        /*if !pattern.patterned(ctx) {
          // mapping regular file path
        } else */if matched, _, _ = pattern.match(ctx, pat); !matched {
          // Flip glob matching order.
          if patPatterned { if matched, _, _ = pat.match(ctx, pattern); matched {
            breakAbsRel = true // using the arg glob
            goto afterMatchedPattern
          }}
          continue ForFilemaps; afterMatchedPattern:
        }

        var str string
        var names []string // glob returned file names
        if str, err = pattern.Strval(ctx); err != nil {
          erro(ctx, "strval '%v' failed: %v", pattern, err).debug(1)
          return
        } else if str == "" {
          erro(ctx, "empty pattern: %v", pattern).debug(1)
          return
        }

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
        for _, path := range fm.paths {
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
            if name, err = pattern.Strval(ctx); err != nil { break ForPatterns }

            // Append this non-existed/missing file.
            file := stat(ctx, name, sub, prefix, nil)
            files = append(files, file)
          } else if ok && !path.expandible(ctx, expandClosure) && len(fm.paths) == 1 {
            // Just report that the pattern matches no files in the
            // file system (if only one path specified).
            if false {
              warn(ctx, "%s: %v matches no files in '%v'", p.name, fm, sub).of(pattern)
              warn(ctx, "%s: here is %v (try using flag -m, aka -include-missing)", p.name, pat).of(pat).debug(1)
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

func (p *Project) matchFile(ctx Context, name string, baseFiles bool) (file *File) {
  if optionEnableBenchmarks && false { defer bench(mark("Project.FindFile")) }
  if optionEnableBenchspots { defer bench(spot("Project.FindFile")) }

  var first *File

ForFilemaps:
  for _, filemap := range p.filemaps(ctx, /*true*/baseFiles, false) {
    // Match the represented file name.
    var matched, pre = filemap.Match(ctx, name) // TODO: performance
    if !matched { continue ForFilemaps }

    var proj = filemap.project
    if proj.changedWD != "" { file = filemap.stat(ctx, proj.changedWD, pre, name) }
    if file == nil          { file = filemap.stat(ctx, proj.absPath,   pre, name) }
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
      if fm.project == p && filemap == fm { break ForFilemaps }
    }
  }
  if first != file && !file.exists() { file = first }
  return
}

func (p *Project) matchTempFile(ctx Context, name string) (file *File) {
  var pos = ctx.Position()
  if file = p.matchFile(ctx, name, true); file != nil {
    // good
  } else if ctd := p.scope.FindDef("CTD"); ctd == nil {
    erro(ctx, "%v: CTD is not defined for temp file: %v", p, name).at(pos).debug(1)
  } else if s, err := ctd.Strval(ctx); err != nil {
    erro(ctx, "%v: stringify temp directory failed: %v", p, err).at(pos).debug(1)
  } else if file = stat(ctx, filepath.Join(s, name), "", "", nil); file == nil {
    erro(ctx, "%v: nil stat %v %v", p, s, name).at(pos).debug(1)
  } else if false {
    if !pos.IsValid() { pos = p.position }
    warn(ctx, "using default temp file: %v/%v", s, name).at(pos)
    warn(ctx, "suggesting define files rule for '%s' in %v", name, p).at(p.position).debug(12)
  }
  return // NOTE: temp file may not exists
}

func (p *Project) configuration(ctx Context) (file *File) {
  ctx = closureWith(ctx, p.scope.position, p.scope)
  if file = p.matchTempFile(ctx, "configuration.sm"); file == nil {
    erro(ctx, "%v: no file configuration.sm", p).at(p.position).debug(1)
  }
  return
}

func (p *Project) FindFile(ctx Context, name string) (file *File) {
  return p.matchFile(ctx, name, true)
}

func (p *Project) isFileName(ctx Context, s string) (res bool) {
  if len(s) > 0 {
    for _, filemap := range p.filemaps(ctx, true, true) {
      if res, _ = filemap.Match(ctx, s); res { break }
    }
  }
  return
}

func (p *Project) DefaultEntry() (entry Entry) {
  if len(p.concrete) > 0 { entry = p.concrete[0] }
  return
}

func (p *Project) resolveObject(ctx Context, s string) (obj Object, err error) {
  if _, obj = p.scope.Find(s); isNil(obj) {
    if p.pluginScope != nil {
      if obj = p.pluginScope.Lookup(s); obj != nil {
        return
      }
    }
    for _, base := range p.bases {
      if obj, err = base.resolveObject(ctx, s); err != nil {
        erro(ctx, "resolve object '%v' failed: %v", s, err).at(base.position).debug(1)
        break
      } else if obj != nil {
        break
      }
    }
  }
  return
}

func (p *Project) resolveEntries(ctx Context, s string, matchFullSuffix, alwaysResolveBases bool) (entries *ResolveEntries, err error) {
  if optionEnableBenchmarks && false { defer bench(mark("Project.resolveEntry")) }
  if optionEnableBenchspots { defer bench(spot("Project.resolveEntry")) }
  var add = func(a ...Entry) {
    if len(a) > 0 {
      if entries == nil { entries = new(ResolveEntries) }
      entries.add(a[0])
      entries.all = append(entries.all, a[1:]...)
    }
  }
  for _, entry := range p.concrete {
    switch target := entry.Target().(type) {
    case *File:
      if target.name == s {
        add(entry)
      } else if fullname := target.fullname(); filepath.IsAbs(s) && s == fullname {
        add(entry)
      } else if !matchFullSuffix {
        // not matching
      } else if strings.HasSuffix(fullname, PathSep+filepath.Clean(s)) {
        add(entry)
      }
    default:
      var sv string
      if sv, err = target.Strval(ctx); err != nil {
        erro(ctx, "strval '%v' failed: %v", target, err).at(target.Position()).debug(1)
        return
      } else if sv == s {
        add(entry)
      }
    }
  }
  if alwaysResolveBases || entries == nil {
    for _, base := range p.bases {
      var ents *ResolveEntries
      if ents, err = base.resolveEntries(ctx, s, matchFullSuffix, alwaysResolveBases); err != nil {
        erro(ctx, "resolve entry '%v' failed: %v", s, err).at(base.position)
        erro(ctx, "resolve entry '%v' failed (%s)", s, ctx).debug(1)
        return
      } else if ents != nil {
        add(ents.all...)
        break
      }
    }
  }
  if true { /* FAST */ } else if err == nil && entries == nil { /* SLOW */
    for _, using := range p.using.list {
      var ents *ResolveEntries
      if ents, err = using.project.resolveEntries(ctx, s, matchFullSuffix, alwaysResolveBases); err != nil {
        erro(ctx, "resolve entry '%v' failed: %v", s, err).at(using.position)
        erro(ctx, "resolve entry '%v' failed (%s)", s, ctx).debug(1)
        break
      } else if ents != nil {
        add(ents.all...)
        break
      }
    }
  }
  return
}

func (p *Project) resolvePatterns(ctx Context, i interface{}) (res []*stemmed) {
  if optionEnableBenchmarks && false { defer bench(mark("Project.resolvePatterns")) }
  if optionEnableBenchspots { defer bench(spot("Project.resolvePatterns")) }
  if true  { res = append(res, p._resolvePatterns1(ctx, i)...) }
  if true  { res = append(res, p._resolvePatterns2(ctx, i)...) }
  if false { res = append(res, p._resolvePatterns3(ctx, i)...)/* heavy work, VERY SLOW! */ }
  return
}

func (p *Project) _resolvePatterns1(ctx Context, i interface{}) (res []*stemmed) {
  if optionEnableBenchspots { defer bench(spot("Project._resolvePatterns1")) }
  for _, pat := range p.patterns {
    if full, _, stems := pat.target.match(ctx, i); full {
      res = append(res, &stemmed{*pat, stems})
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
func (p *Project) entry(ctx Context, special specialRule, options []Value, target Value, prog *Program) (entry Entry, err error) {
  defer func() {
    if entry != nil && err == nil {
      entry.SetPrograms(append(entry.Programs(), prog))
    }
  } ()

  // The 'use' rule entries.
  var closured = target.expandible(ctx, expandClosure)
  if special == specialRuleUse && !closured {
    var opts entryOpts
    if len(options) > 0 {
      var pos = options[0].Position()
      if _, err = parseOpts(ctx, &opts, options...); err != nil {
        erro(ctx, "parse opts failed: %v", err).at(pos)
        return
      }
    }
    var userule = &useRuleEntry{
      RuleEntry{
        position: target.Position(),
        class:UseRuleEntry,
        target:target,
      },
      opts.postExec, // post-execute use rule?
    }
    p.userules = append(p.userules, userule)
    entry = userule //&userule.RuleEntry
    return
  }

  var name string
  if target.patterned(ctx) {
    var class = PatternRuleEntry
    if _, ok := target.(*Path); ok {
      class = PathPattRuleEntry
    }
    var pattern = &PatternEntry{RuleEntry{
      position: target.Position(),
      class: class,
      target: target,
    }}
    p.patterns = append(p.patterns, pattern)
    entry = pattern //&pattern.RuleEntry
    return
  } else if name, err = fullnameOrStrval(ctx, target); err != nil {
    erro(ctx, "fullname/strval failed: %v", err).of(target).debug(1)
    return
  } else if name == "" {
    erro(ctx, "empty target name: %v", target).of(target).debug(1)
    return
  }

  // Looking for concrete rule entries.
  for _, rec := range p.concrete {
    var sv string
    if closured && rec.String() == name {
      entry = rec; break
    }
    if sv, err = fullnameOrStrval(ctx, rec); err != nil {
      return
    } else if sv == name {
      entry = rec; break
    }
  }
  if entry == nil {
    entry = &RuleEntry{
      position: target.Position(),
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

func (p *Project) hasLoaded(ctx Context, proj *Project, breakUseLoop bool) (rp *Project, res, isb bool, err error) {
  return p.hasLoadedRecur(ctx, p, proj, 1, breakUseLoop)
}

func (p *Project) hasLoadedRecur(ctx Context, top, proj *Project, depth int, breakUseLoop bool) (rp *Project, res, isb bool, err error) {
  if depth > 1 && top == p && true {
    err = fmt.Errorf("loop '%v' (depth=%d)", p.loopLoadPath(), depth)
    erro(ctx, "%v: %v", p, err).at(p.position).debug(128)
    return
  } else if depth > 128 {
    err = fmt.Errorf("exceeds maximum base depth (%d) (start=%v, target=%v)", depth, top, proj)
    erro(ctx, "%v: %v", p, err).at(p.position)
    erro(ctx, "start: %v", top).at(top.position)
    erro(ctx, "target: %v", proj).at(proj.position).debug(200)
    return
  }
  for _, base := range p.bases {
    if isb = base == proj; isb { return }
    if rp, res, isb, err = base.hasLoadedRecur(ctx, top, proj, depth+1, breakUseLoop); err != nil {
      return
    } else if res || isb { rp = base ; return }
  }
  for _, using := range /*p.loads*/p.using.list {
    var imp = using.project
    if imp == top && !breakUseLoop {
      s := top.loopLoadPath()
      err = fmt.Errorf("loop `%v`", s)
      erro(ctx, "start: %v", top).at(top.position)
      erro(ctx, "stop: %v", proj).at(proj.position)
      erro(ctx, "%v: %v", p, err).at(p.position).debug(128)
      return
    }
    if res = imp == proj; res { rp = imp; return }
    if rp, res, res, err = imp.hasLoadedRecur(ctx, top, proj, depth+1, breakUseLoop); err != nil {
      return
    } else if res { rp = imp; return }
  }
  rp = p
  return
}

func (p *Project) loopLoadPath() (s string) { return p.loopLoadRecur(p) }
func (p *Project) loopLoadRecur(top *Project) (s string) {
  for _, using := range /*p.loads*/p.using.list {
    var imp = using.project
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

func enter(ctx Context, dir string) (err error) {
  cd.mutex.Lock(); defer cd.mutex.Unlock()

  if options.traceEntering {
    fmt.Fprintf(stderr, "entering: %v (%v)\n", dir, ctx.Project().name)
  }

  var wd string
  if wd, err = os.Getwd(); err != nil { return }
  if err = lockCD(dir, 0); err != nil { return }
  if !filepath.IsAbs(dir) { dir = filepath.Join(wd, dir) }
  ctx.autoSet("CWD", MakeString(ctx.program().position, dir))

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
  if options.traceEntering {
    fmt.Fprintf(stderr, "leaving: %v (%v %v %v)\n", stop.dir, prog.project.name, stop.num, size)
  }

  for _, enter := range cd.stack {
    if enter.num == 0 { continue } else {
      enter.num -= 1
    }
    if enter == stop {
      if enter.print && false {
        enter.print = false
        prompt(ctx, "smart:  Leaving directory '%s'\n", enter.dir)
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

func printEnteringDirectory(ctx Context) {
  cd.mutex.Lock(); defer cd.mutex.Unlock()
  if size := len(cd.stack); size > 0 {
    var enter = cd.stack[0]
    if enter.silent { return }
    for _, p := range cd.stack {
      if p.print && p != enter {
        p.print = false
        prompt(ctx, "smart:  Leaving directory '%s'\n", p.dir)
      }
    }
    if !enter.print {
      enter.print = true
      prompt(ctx, "smart: Entering directory '%s'\n", enter.dir)
    }
  }
}

func printLeavingDirectory(ctx Context) {
  cd.mutex.Lock(); defer cd.mutex.Unlock()
  if size := len(cd.stack); size > 0 {
    for _, enter := range cd.stack {
      if enter.print {
        enter.print = false
        prompt(ctx, "smart:  Leaving directory '%s'\n", enter.dir)
      }
    }
  }
}
