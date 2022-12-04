//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
  "extbit.io/smart/token"
  "path/filepath"
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
  patts []Value
  paths []Value
  public bool
}

func (filemap *FileMap) String() (s string) {
  if n := len(filemap.patts); n == 1 {
    s = filemap.patts[0].String()
  } else if n > 1 {
    s = fmt.Sprintf("%s", filemap.patts)
  }
  return
}
func (filemap *FileMap) Patterns(ctx Context) (pats []Value) {
  for _, pattern := range filemap.patts {
    if pattern.expandible(ctx, expandClosure) {
      if false && !options.allowClosureFilemap { // -closure-files
        warnstack(ctx, 8, "closure filemap pattern may cause recursive file resolving: %v", pattern).
          of(pattern).debug(32)
        ctx.checkErrors(true) // check here to report warnings immediately
      }

      // FIXME+TODO: this could be time consuming to expand clousre in the filemap
      /*if pats, err = mergex(ctx, plain, pattern); err != nil {
        erro(ctx, "merge pattern '%v' failed: %v", pattern, err).of(pattern)
      } else*/
      var unexpanded int
      if pats, unexpanded, _ = expand(ctx, plain, pattern); unexpanded>0 {
        errostack(ctx, 3, "unexpanded file pattern: %v", pats).of(pattern).debug(15)
        ctx.checkErrors(true) // check here to report warnings immediately
      }
      pats = mergex(ctx, plain, pats...)
    } else {
      pats = append(pats, pattern)
    }
  }
  return merge(pats...)
}

// Match split filename into list and match each part with the pattern correspondingly.
func (filemap *FileMap) Match(ctx Context, str string) (matched bool, pattern Value, pre string) {
  // TODO: escape file matching for 'String' and "Compound" values
  for _, pat := range filemap.Patterns(ctx) {
    if matched, pre = filemap.match(ctx, pat, str); matched { pattern = pat; break }
  }
  return
}

func (filemap *FileMap) match(ctx Context, pat Value, str string) (matched bool, pre string) {
  // TODO: escape file matching for 'String' and "Compound" values
  if false && pat.String() == "$(name).tex" {
    matched, pattern, pre := pat.match(ctx, str)
    warn(ctx, "%T %v %s ; %v -> %v %v '%v'", pat, pat, pat.Strval(ctx), str, matched, pattern, pre).debug(1)
  }
  if matched, _, _ = pat.match(ctx, str); !matched && !(isNone(pat) || isNil(pat)) {
    if n := strings.Index(str, PathSep); n < 0 { return }
    // NOTE: Dealing with these files:
    //     files (
    //         (foo.c) => $(srcdir)/sub/dir
    //         (sub/dir/foo.c) => $(srcdir)
    //     )
    for _, p := range filemap.paths { // FIXME: performance, operate on p.(*Path) instead
      if _, ok := p.(*Path); !ok { continue } // NOTE: only work with paths to improve performance
      var ps = p.Strval(ctx)
      for i := strings.LastIndex(ps, PathSep); -1 <= i; {
        var ( prefix = ps[i+1:]; l = len(prefix) ) // NOTE: -1 <= i < len(ps)
        if has := strings.HasPrefix(str, prefix) && str[l] == '/'; has {
          if matched, _, _ = pat.match(ctx, str[len(prefix)+1:]); matched { break }
        }
        if 0 < i { i = strings.LastIndex(ps[:i], PathSep) } else { break }
      }
    }
  }
  return // NOTE: also `globMatchFile(ctx, pat, str, true)`
}

func (filemap *FileMap) stat(ctx Context, base, pre, name string) (file *File) {
  var pos = filemap.patts[0].Position(); if false { ctx = positional(ctx, pos) }
  if base = filepath.Clean(base); len(filemap.paths) == 0 {
    file = stat(ctx, name, "", base, nil) // simply stat file name if no paths
    return
  } else if pre != "" {
    pre = filepath.Clean(pre)
  }
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

    var sub string
    if sub = path.Strval(ctx); sub == "" {
      if true {
        erro(ctx, "filemap path '%v' is empty (%T)", path, path).at(path.Position())
        erro(ctx, "filemap path '%v' is empty (pattern=%v)", path, filemap.patts).at(pos)
        erro(ctx, "filemap path '%v' is empty (project=%v)", path, ctx.Project())//.at(pos)
        erro(ctx, "filemap path '%v' is empty in %v", path, ctx).debug(64)
      }
      return
    } else if s := filepath.Clean(sub); sub != s {
      if false {
        erro(ctx, "filemap path '%v' is not clean (sub=%s)", path, sub).at(path.Position())
        erro(ctx, "filemap path '%v' is not clean (pattern=%v)", path, filemap.patts).at(pos)
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
        if s := sub+PathSep; strings.HasPrefix(name, s) { // 'name' should have 'sub' prefix
          if false {
            warn(ctx, "sub  = %v", s).at(pos)
            warn(ctx, "name = %v", name).at(pos)
            warn(ctx, "%v", ctx).debug(6)
          }
          name = strings.TrimPrefix(name, s)
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
  configs []Entry // configure entries
  configured bool

  changedWD string
	absPath string
	relPath string
  tmpPath string
	spec    string
	name    string
  scope   *Scope
  bases []*Project
  use     *uselist

  // List order is significant, duplication is acceptable.
  _filemap_ []*FileMap

  // Rule Registry (orderred)
  //userules []*useRuleEntry // the 'use' rule
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

func (p *Project) mapfile(ctx Context, opts filesOpts, patts, paths []Value) {
  // NOTE: List order is significant, duplications are acceptable.
  p._filemap_ = append(p._filemap_, &FileMap{ p, patts, paths, opts.public })
}

func (p *Project) myFilemaps(ctx Context) (filemap []*FileMap) {
  return p._filemap_
}

func uniqueAppendFilemap(ctx Context, filemaps []*FileMap, a *FileMap) (result []*FileMap) {
  if false {
    var numDuplicated int
    for _, m := range filemaps {
      if a == m || (&a.patts == &m.patts && &a.paths == &m.paths) {
        result = filemaps
        return
      } else if len(a.patts) == len(m.patts) && len(a.paths) == len(m.paths) {
        var same = true // initially assumes all paths are identical
        for i, ap := range a.paths {
          if ap != m.paths[i] { same = false; break }
        }
        if same {
          result = filemaps
          return
        } else {
          warn(ctx, "files might be duplicated: %v (paths=%v),", a, a.paths).of(a.patts[0])
          warn(ctx, "                     with: %v (paths=%v)" , m, m.paths).of(m.patts[0])
          warn(ctx, "          differred paths: %v", a.paths[0]).of(a.paths[0])
          warn(ctx, "                      and: %v", m.paths[0]).of(m.paths[0])
          numDuplicated += 1
        }
      }
    }
    if numDuplicated > 0 { erro(ctx, "duplicated files: %v", a.patts).of(a.patts[0]) }
  }
  result = append(filemaps, a)
  return
}

func (p *Project) filemaps(ctx Context, baseFiles, usedFiles bool) (filemaps []*FileMap) {
  var appendUnique = func(a *FileMap) {
    filemaps = uniqueAppendFilemap(ctx, filemaps, a)
  }

  for _, m := range p.myFilemaps(ctx) { appendUnique(m) }
  if baseFiles { for _, base := range p.bases {
    for _, m := range base.filemaps(ctx, true, usedFiles) { appendUnique(m) }
  }}
  if p.configure != nil && ctx.configuration() {
    for _, m := range p.configure.filemaps(ctx, true, usedFiles) { appendUnique(m) }
  }
  if usedFiles && false/* FIXME: performance */ {
    // takes a big longer time to map usee filemaps, but acceptable
    var (
      appendUselist func(*Project)
      appendUsedFiles func(*Project)
    )
    appendUselist = func(p *Project) {
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
      for _, u := range p.use.list {
        appendUselist(u.project)
      }
    }
    appendUsedFiles(p)
  }
  return
}

func (p *Project) wildcard(ctx Context, opts wildcardOpts, patterns ...Value) (files []*File, err error) {
  var filemaps = p.filemaps(ctx, opts.baseFiles, opts.usedFiles)
ForPatterns:
  for _, inPat := range patterns {
    var (
      inPatPatterned = inPat.patterned(ctx)
      breakAbsRel bool
      matched bool
    )
  ForFilemaps:
    for _, filemap := range filemaps {
      for _, pattern := range filemap.Patterns(ctx) {
        if matched, _, _ = pattern.match(ctx, inPat); !matched {
          // flip matching patterns
          if !inPatPatterned {
            // unmatched pattern
          } else if matched, _, _ = inPat.match(ctx, pattern); matched {
            breakAbsRel = true // using the arg glob
            goto afterMatchedPattern
          }
          continue /*ForFilemaps -- FIXES: break too early */; afterMatchedPattern:
        }

        // glob returned file names
        var (
          names []string
          str string
        )
        if file, y := toFile(pattern); y && len(filemap.paths) == 0 {
          files = append(files, file)
          if n := opts.debug; n>0 /* && p.name == "lib.unwind" */ { warn(ctx, "%v -> %v (exists=%v)",
            pattern, file, file.exists()).debug(n) }
          continue
        } else if str = pattern.Strval(ctx); str == "" {
          erro(ctx, "empty pattern: %v", pattern).debug(1)
          return
        }

        // Absolute or relative files are not related to the paths.
        if filepath.IsAbs(str) || strings.HasPrefix(str, "./") || strings.HasPrefix(str, "../") {
          if names, err = filepath.Glob(str); err != nil { break ForPatterns }
          for _, s := range names {
            file := stat(ctx, filepath.Base(s), "", filepath.Dir(s))
            files = append(files, file)
            if enable_assertions { assert(file != nil, "`%s` missing", s) }
            if n := opts.debug; n>0 { warn(ctx, "%v -> %v (exists=%v)",
              pattern, file, file.exists()).debug(n) }
          }
          if breakAbsRel {
            continue ForPatterns
          } else {
            continue ForFilemaps
          }
        }

        var patterned = pattern.patterned(ctx)

        // Check against paths for non-abs/rel patterns.
        for _, path := range filemap.paths {
          var sub = path.Strval(ctx)
          var subfile = filepath.Join(sub, str)
          if names, err = filepath.Glob(subfile); err != nil {
            erro(ctx, "%v: %v: %v", p, subfile, err).of(path)
            errostack(ctx, 6, "").of(path).debug(12)
            break ForPatterns
          }

          // Chop off path 'sub' prefix to have shorter names
          // Aka. trim prefix 'file.Sub+PathSep'
          if prefix := strings.TrimSuffix(subfile, str); len(names) > 0 {
            for _, s := range names {
              var name = strings.TrimPrefix(s, prefix)
              if file := stat(ctx, name, sub, prefix, nil); file == nil {
                if n := opts.debug; n>0 { warn(ctx, "%v -> %v %v (nil)",
                  pattern, sub, prefix).debug(n) }
                erro(ctx, "%v: '%v' not found in %v", p, name, path).of(filemap.patts[0])
                errostack(ctx, 6, "").of(path).debug(12)
              } else if file.exists() || opts.includeMissing {
                files = append(files, file)
                if n := opts.debug; n>0 /* && p.name == "lib.unwind" */ { warnstack(ctx, n, "%v -> %v %v+%v (exists=%v)",
                  pattern, sub, prefix, file, file.exists()).debug(n) }
              } else if opts.errorMissing {
                erro(ctx, "%v: '%v' not found in %v", p, name, path).of(filemap.patts[0])
                errostack(ctx, 6, "").of(path).debug(12)
                if true { fail(path.Position(), "missing %v", path) }
              }
              if false && strings.HasSuffix(s, "libunwind.cpp") {
                warnstack(ctx, 32, "wildcard: %v -> %v", str, s).debug(32)
              }
            }
          } else if !patterned && opts.includeMissing {
            // If the filemap is not a pattern (e.g. foobar.cpp), we include it in the returning files
            // Append this non-existed/missing file.
            file := stat(ctx, pattern.Strval(ctx), sub, prefix, nil)
            files = append(files, file)
            if n := opts.debug; n>0 /* && p.name == "lib.unwind" */ { warn(ctx, "%v -> %v %v+%v",
              pattern, sub, prefix, file).debug(n) }
          } else if patterned && !path.expandible(ctx, expandClosure) && len(filemap.paths) == 1 {
            if false {
              // Just report that the pattern matches no files in the
              // file system (if only one path specified).
              warn(ctx, "%s: %v matches no files in '%v'", p.name, filemap, sub).of(pattern)
              warn(ctx, "%s: here is %v (try using flag -m, aka -include-missing)", p.name, inPat).of(inPat).debug(1)
            }
          } else if opts.errorMissing {
            erro(ctx, "%v: '%v' not found in %v", p, pattern, path).of(filemap.patts[0])
            errostack(ctx, 6, "(%T):", ctx).of(path).debug(12)
            if true { fail(path.Position(), "missing %v", path) }
            break ForPatterns
          }
        }
      }
    }
  }
  return
}

func (p *Project) matchFile(ctx Context, name string, baseFiles bool) (file *File) {
  var first *File

ForFilemaps:
  for _, filemap := range p.filemaps(ctx, /*true*/baseFiles, false) {
    // Match the represented file name.
    var matched, pattern, pre = filemap.Match(ctx, name) // TODO: performance
    // warn(ctx, "%T %v ; %v -> %v %v '%v'", filemap, filemap, name, matched, pattern, pre).debug(1)
    if !matched { continue ForFilemaps }
    if f, y := toFile(pattern); y {
      file = f; break ForFilemaps
    }

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
  } else if file = stat(ctx, filepath.Join(ctd.Strval(ctx), name), "", "", nil); file == nil {
    erro(ctx, "%v: nil stat %v %v", p, ctd.Strval(ctx), name).at(pos).debug(1)
  } else if false {
    if !pos.IsValid() { pos = p.position }
    warn(ctx, "using default temp file: %v/%v", ctd.Strval(ctx), name).at(pos)
    warn(ctx, "suggesting define files rule for '%s' in %v", name, p).at(p.position).debug(12)
  }
  return // NOTE: temp file may not exists
}

func (p *Project) configuration(ctx Context) (file *File) {
  if file = p.matchTempFile(closureWith(ctx, p.scope), "configuration.sm"); file == nil {
    erro(ctx, "%v: no file configuration.sm", p).debug(1)
  }
  return
}

func (p *Project) FindFile(ctx Context, name string) (file *File) {
  return p.matchFile(ctx, name, true)
}

// func (p *Project) isFileName(ctx Context, s string) (res bool) {
//   if len(s) > 0 {
//     for _, filemap := range p.filemaps(ctx, true, true) {
//       if res, _, _ = filemap.Match(ctx, s); res { break }
//     }
//   }
//   return
// }

func (p *Project) DefaultEntry() (entry Entry) {
  if len(p.concrete) > 0 { entry = p.concrete[0] }
  return
}

func (p *Project) resolveObject(ctx Context, s string) (obj Object) {
  if _, obj = p.scope.Find(s); isNil(obj) {
    if p.pluginScope != nil {
      if obj = p.pluginScope.Lookup(s); obj != nil {
        return
      }
    }
    for _, base := range p.bases {
      if obj = base.resolveObject(ctx, s); obj != nil {
        break
      }
    }
    if isNil(obj) && p.configure != nil && ctx.configuration() {
      obj = p.configure.resolveObject(ctx, s)
    }
  }
  return
}

func (p *Project) resolveEntries(ctx Context, s string, matchingFullSuffix, alwaysResolveBases bool) (entries *ResolveEntries) {
  var add = func(a ...Entry) {
    if len(a) > 0 {
      if entries == nil { entries = new(ResolveEntries) }
      entries.add(a[0])
      entries.all = append(entries.all, a[1:]...)
    }
  }

  var match = func(entry Entry, name string) (res bool) {
    var target = entry.Target()
    if file, ok := toFile(target); ok {
      if res = file.name == name; !res {
        var full = file.fullname()
        if res = filepath.IsAbs(name) && name == full; !res && matchingFullSuffix {
          res = strings.HasSuffix(full, PathSep+filepath.Clean(name))
        }
      }
    } else {
      res = target.Strval(ctx) == name
    }
    return
  }

  var found Entry
  var t1 = autoGet(ctx, "@")
  ForConcretes: for _, entry := range p.concrete {
    if match(entry, s) { found = entry } else { continue ForConcretes }

    for pc := ctx.programContext(); pc != nil; { // loop detection
      if pc.entry() == entry {
        var t2 = autoGet(pc, "@")
        if t1 == t2 || t1.cmp(ctx, t2) == cmpEqual || t1.Strval(ctx) == t2.Strval(ctx) {
          if false {
            warn(ctx, "%v: %p %v %T", entry, t1, t1, t1).of(t1)
            warn(ctx, "%v: %p %v %T", entry, t2, t2, t2).of(t1)
            warnstack(ctx, 3, "%v: %v, %v %v (same: %v, %v, %v)",
              entry, s, t1, t2, (t1 == t2), t1.cmp(ctx, t2), t2.cmp(ctx, t1)).debug(1)
          }
          continue ForConcretes // break the loop
        }
      }
      if c := pc.inner(); c != nil {
        pc = c.programContext()
      } else {
        break
      }
    }

    add(entry)
  }
  if entries == nil && found != nil { add(found) }

  if alwaysResolveBases || entries == nil {
    for _, base := range p.bases {
      if ents := base.resolveEntries(ctx, s, matchingFullSuffix, alwaysResolveBases); ents != nil {
        add(ents.all...)
        break
      }
    }
  }
  if p.configure != nil && ctx.configuration() {
    if ents := p.configure.resolveEntries(ctx, s, matchingFullSuffix, true); ents != nil {
      add(ents.all...)
    }
  }
  if true {
    /* FAST */
  } else if entries == nil { /* SLOW */
    for _, use := range p.use.list {
      ents := use.project.resolveEntries(ctx, s, matchingFullSuffix, alwaysResolveBases)
      if ents != nil {
        add(ents.all...)
        break
      }
    }
  }
  return
}

func (p *Project) resolvePatterns(ctx Context, v Value, s string) (res []*stemmed) {
  if res = p.resolvePatterns123(ctx, v, s); false && len(res) > 0 {
    for _, t := range res {
      if file, _ := toFile(t.target); file != nil {
        file.position = t.position
      } else if file = p.FindFile(ctx, s); file != nil {
        file.position = t.position
        t.target = file
      }
    }
  }
  return
}

func (p *Project) resolvePatterns123(ctx Context, v Value, s string) (res []*stemmed) {
  if true  { res = append(res, p.resolvePatterns1(ctx, v, s)...) }
  if true  { res = append(res, p.resolvePatterns2(ctx, v, s)...) }
  if false { res = append(res, p.resolvePatterns3(ctx, v, s)...)/* heavy work, VERY SLOW! */ }
  return
}

func (p *Project) resolvePatterns1(ctx Context, val Value, s string) (res []*stemmed) {
  ForPatterns: for _, pat := range p.patterns {
    if full, m, stems := pat.target.match(ctx, s); full {
      for sc := ctx.stemmedContext(); sc != nil; { // pattern loop detection
        if s := sc.stem.target.Strval(ctx); s == m {
          if false && strings.Contains(m, "isl/stdint.h") {
            var t = sc.stem.target
            warn(sc, "%v: %T %v; %v; %v; %v, %v %v",
              s, t, t, sc.stem.Stems, pat, s == m, m, stems).debug(1)
          }
          continue ForPatterns // break the loop
        }
        if c := sc.inner(); c != nil { sc = c.stemmedContext() } else { break }
      }

      if ok := false; len(pat.argumented) > 0 {
        for _, a := range mergex(ctx, plain, pat.argumented...) {
          if ok, _, _ = a.match(ctx, s); ok { break }
        }
        if !ok { continue ForPatterns }
      }

      res = append(res, &stemmed{pat, val, stems})
    }
  }
  return
}

func (p *Project) resolvePatterns2(ctx Context, val Value, s string) (res []*stemmed) {
  for _, base := range p.bases {
    res = append(res, base.resolvePatterns123(ctx, val, s)...)
  }
  if p.configure != nil && ctx.configuration() {
    res = append(res, p.configure.resolvePatterns123(ctx, val, s)...)
  }
  return
}

func (p *Project) resolvePatterns3(ctx Context, val Value, s string) (res []*stemmed) {
  for _, use := range p.use.list {
    res = append(res, use.project.resolvePatterns123(ctx, val, s)...)
  }
  return
}

type entryOpts struct {
  postExec bool `p,post;pe,post-execute;pe,post-exec`
}
func (p *Project) entry(ctx Context, special specialRule, options []Value, patterned bool, target Value, arged []Value, prog *Program) (entry Entry, err error) {
  defer func() {
    if entry != nil && err == nil {
      entry.setPrograms(append(entry.Programs(), prog))
    }
  } ()

  // The 'use' rule entries.
  var closured = target.expandible(ctx, expandClosure)
  if special == specialRuleUse && !closured {
    var opts entryOpts
    parseOpts(ctx, &opts, 0, options...)

    /*var userule = &useRuleEntry{
      RuleEntry{
        position: target.Position(),

        class:UseRuleEntry,
        target:target,
      },
      opts.postExec, // post-execute use rule?
    }
    p.userules = append(p.userules, userule)
    entry = userule //&userule.RuleEntry*/
    panic(":use: rule entry is deprecated")
    return
  }

  var name string
  if patterned {
    var class = PatternRuleEntry
    if _, ok := target.(*Path); ok { class = PathPattRuleEntry }
    var pattern = &PatternEntry{RuleEntry{
      position: target.Position(), class: class, target: target, argumented: arged,
    }}
    p.patterns = append(p.patterns, pattern)
    entry = pattern
    return
  } else if name = fullnameOrStrval(ctx, target); name == "" {
    erro(ctx, "empty target name: %v", target).of(target).debug(1)
    return
  }

  // Looking for concrete rule entries.
  for _, rec := range p.concrete {
    var sv string
    if closured && rec.String() == name { entry = rec; break }
    if sv = fullnameOrStrval(ctx, rec); sv == name { entry = rec; break }
  }
  if entry == nil {
    entry = &RuleEntry{
      position: target.Position(), class: GeneralRuleEntry, target: target, argumented: arged,
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
    if res = base == proj || base.hasBase(proj); res { break }
  }
  return
}

func (p *Project) hasLoaded(ctx Context, proj *Project, traveUseLoop bool) (rp *Project, res, isb bool, err error) {
  if options.checkLoadGraph || !options.fastMode {
    rp, res, isb, err = p.hasLoadedRecur(ctx, p, proj, 1, traveUseLoop)
  }
  return
}

func (p *Project) hasLoadedRecur(ctx Context, top, proj *Project, depth int, traveUseLoop bool) (rp *Project, res, isb bool, err error) {
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
    if rp, res, isb, err = base.hasLoadedRecur(ctx, top, proj, depth+1, traveUseLoop); err != nil {
      return
    } else if res || isb { rp = base ; return }
  }
  for _, use := range /*p.loads*/p.use.list {
    var imp = use.project
    if imp == top && !traveUseLoop {
      s := top.loopLoadPath()
      err = fmt.Errorf("loop `%v`", s)
      erro(ctx, "start: %v", top).at(top.position)
      erro(ctx, "stop: %v", proj).at(proj.position)
      erro(ctx, "%v: %v", p, err).at(p.position).debug(128)
      return
    }
    if res = imp == proj; res { rp = imp; return }
    if rp, res, res, err = imp.hasLoadedRecur(ctx, top, proj, depth+1, traveUseLoop); err != nil {
      return
    } else if res { rp = imp; return }
  }
  rp = p
  return
}

func (p *Project) loopLoadPath() (s string) { return p.loopLoadRecur(p) }
func (p *Project) loopLoadRecur(top *Project) (s string) {
  for _, use := range /*p.loads*/p.use.list {
    var imp = use.project
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
  for _, use := range p.use.list {
    if res = use.project == usee; res { break  }
    if res = use.project.isUsingProject(usee); res { break }
  }
  return
}

func (p *Project) isUsingDirectly(proj *Project) (res bool) {
  for _, u := range p.use.list {
    if res = u.project == proj; res { break }
  }
  return
}

func (p *Project) usees(bases, basesRecur, useeRecur, pre bool) (res []*Project) {
  if p.opts.traveUseLoop { return }
  if bases {
    for _, base := range p.bases {
      res = append(res, base.usees(basesRecur, basesRecur, useeRecur, pre)...)
    }
  }
  for _, u := range p.use.list {
    if pre { res = append(res, u.project) }
    if useeRecur {
      for _, u := range u.project.usees(bases && basesRecur, basesRecur, true, pre) {
        if !p.isUsingDirectly(u) { res = append(res, u) }
      }
    }
    if !pre { res = append(res, u.project) }
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
