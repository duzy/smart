//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
  "path/filepath"
  "strings"
  "plugin"
  "sync"
  "time"
  "fmt"
  "io/fs"
  "os"
)

const configuration_sm = "configuration.sm"
const PathSepByte = filepath.Separator
const PathSep = string(PathSepByte)

type filemap struct {
  project *Project
  patts []Value
  locs []Value
}

type FileMap struct {
  *filemap
  pattern Value
}

func (p *filemap) String() (s string) {
  if n := len(p.patts); n == 1 {
    s = p.patts[0].String()
  } else if n > 1 {
    s = fmt.Sprintf("%s", p.patts)
  }
  return
}

func (p FileMap) String() (s string) {
  if p.pattern == nil {
    s = p.filemap.String()
  } else {
    s = p.pattern.String()
  }
  return
}

func (p *FileMap) primePatterns(ctx Context) (pats []Value) {
  var patts = []Value{ p.pattern }
  if patts[0] == nil { patts = p.patts }

  for _, pattern := range patts {
    if pattern.expandable(ctx, expandClosure) {
      var u int
      // FIXME+TODO: this could be time consuming to expand clousre in the filemap
      /*if pats, err = xmerge(ctx, plain, pattern); err != nil {
        erro(of(ctx,pattern), "merge pattern '%v' failed: %v", pattern, err)
      } else*/ if pats, u, _ = plain.expand(ctx, pattern); u == 0 {
        if pats != nil { pats = xmerge(ctx, plain, pats...) }
      } else {
        errostack(of(ctx,pattern), 3, "unexpanded file pattern: %v", pats).debug(15)
      }
    } else {
      pats = append(pats, pattern)
    }
  }
  return merge(pats...)
}

// Match split filename into list and match each part with the pattern correspondingly.
func (filemap *FileMap) Match(ctx Context, val interface{}) (matched bool, pattern Value, name string) {
  // TODO: escape file matching for 'String' and "Compound" values
  for _, pat := range filemap.primePatterns(ctx) {
    if matched, name = filemap.match(ctx, pat, val); matched { pattern = pat; break }
  }
  return
}

func (filemap *FileMap) match(ctx Context, pat Value, val interface{}) (matched bool, name string) {
  // TODO: escape file matching for 'String' and "Compound" values
  var res interface{}
  matched, res, _ = pat.match(ctx, val)

  if false && !matched && !(isNone(pat) || isNull(pat)) {
    var str string // NOOP
    if n := strings.Index(str, PathSep); n < 0 { return }

    // NOTE: Dealing with these files:
    //     files (
    //         (foo.c) => $(srcdir)/sub/dir
    //         (sub/dir/foo.c) => $(srcdir)
    //     )
    for _, p := range filemap.locs { // FIXME: performance, operate on p.(*Path) instead
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

  if res == nil {
    // okay
  } else if s, y := res.(string); y {
    name = s
  } else if a, y := res.([]string); y {
    name = strings.Join(a, PathSep)
  } else {
    erro(ctx, "unexpected result: %T %v", res, res).debug(1)
  }
  return // NOTE: also `globMatchFile(ctx, pat, str, true)`
}

func (p *FileMap) stat(ctx Context, name string) (file *File) {
  if false && name == "der_dsa_gen.c" {
    defer func() {
      var ( d, s string ; e bool )
      if file != nil { d, s, e = file.dir, file.sub, file.exists() }
      warn(ctx, "%v: %v", name, ctx.projects(ctx))
      warnstack(ctx, 5, "%v: {%s %s %s} %v", file, d, s, name, e).debug(32)
    } ()
  }

  var patts = p.patts
  if len(patts) == 0 {
    errostack(ctx, 5, "no map patterns: %v", p).debug(16)
    return
  }

  if len(p.locs) == 0 {
    for _, pat := range p.patts {
      if f, y := pat.(*File); y && f.name == name { return f }
    }

    for i, pat := range p.patts {
      if f, y := pat.(*File); y {
        info(ctx, "pattern %d. %v %s (exists=%v)", i, f, f.fullname(), f.exists())
      } else {
        info(ctx, "pattern %d. %v (%T)", i, pat, pat)
      }
    }
    errostack(ctx, 5, "%s -> %v", name, p.patts).debug(32)
    return
  }

  var pos = patts[0].Position()
  for _, path := range p.locs {
    if isNull(path) {
      erro(at(ctx,pos), "nil path: name=%s",  name)
      erro(at(ctx,pos), "nil path: %v", p).debug(32)
      fail(pos, "file mapping nil path: %v", p)
    } else if isNone(path) {
      warn(at(ctx,pos), "nil path: name=%s",  name)
      warn(at(ctx,pos), "nil path: %v", p).debug(32)
      continue
    }

    var dir, sub string

    if sub = path.Strval(ctx); sub == "" {
      if true {
        erro(at(ctx,path.Position()), "filemap path '%v' is empty (%T)", path, path)
        erro(at(ctx,pos), "filemap path '%v' is empty (pattern=%v)", path, patts)
        erro(ctx, "filemap path '%v' is empty (project=%v)", path, ctx.Project())//.at(pos)
        erro(ctx, "filemap path '%v' is empty in %v", path, ctx).debug(64)
      }
      return
    } else if s := filepath.Clean(sub); sub != s {
      sub = s
    }

    if filepath.IsAbs(sub) {    // 'sub' is abs
      if filepath.IsAbs(name) { // 'name' is abs too
        if s := sub+PathSep; strings.HasPrefix(name, s) { // 'name' should have 'sub' prefix
          name = strings.TrimPrefix(name, s)
        } else {
          continue
        }
      }
    } else if !filepath.IsAbs(name) {
      // dir = filepath.Clean(base)
    }

    if file = stat(ctx, name, sub, dir, nil); file != nil { break }

    var pre string // Not used!
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

type Project struct {
  position Position
  keyword  Token // project, package, module

  configurationLoad *File // decided at load time
  configurationSave *File // decided at configure time
  configure *Project // .configure
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

  filemapx []*valcache_kv // closure cache
  filemap valcache
  entries valcache
  patterns []*rule // order is important
  configs []Entry // configure entries
  defaultEntry Entry

  // TODO: printEntering() ...
  // TODO: printLeaving() ...

  plugin *plugin.Plugin
  pluginScope *Scope

  opts *projectDeclOpts
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

func (p *Project) wildcard(ctx *builtin_wildcard, patterns ...Value) (files []*File) {
  defer func(t0 time.Time) {
    if d := time.Now().Sub(t0); d > 1*time.Second {
      var pos = ctx.Position()
      prompt(ctx, "%v: slow: %d patterns, %v\n", pos, len(patterns), patterns)
      prompt(ctx, "%v: slow: %d files\n", pos, len(files))
      prompt(ctx, "%v: slow: %v\n", pos, d).debug(4)
    }
  } (time.Now())

  var a []fs.FileInfo
  if ctx.includeMissing && !ctx.ignoreMissing {
    a = []fs.FileInfo{nil}
  }

  var m sync.Mutex
  var g sync.WaitGroup
  var collect = func(t ...*File) {
    m.Lock()
    files = append(files, t...)
    m.Unlock()
    g.Done()
  }
  var f0 = func(lVal, rVal Value, lPat, rPat bool, fm *FileMap) {
    var o = *ctx
    for _, loc := range fm.locs {
      if o.dir = loc.Strval(ctx); lPat && rPat {
        var pat Value
        if lVal.cmp(ctx, rVal) == cmpEqual { pat = lVal } else {
          pat = &compositePattern{lVal, []Value{rVal}}
        }
        g.Add(1) ; go collect(o._do(pat)...)
      } else if lPat && !rPat {
        if file := stat(ctx, rVal.Strval(ctx), "", o.dir, a...); file != nil {
          g.Add(1) ; go collect(file)
        } else if false {
          erro(ctx, "nil: %v: %T %v (%s, %v)", lVal, rVal, rVal, o.dir, a).debug(1)
        }
      } else if !lPat && rPat {
        if file := stat(ctx, lVal.Strval(ctx), o.dir, "", a...); file != nil {
          g.Add(1) ; go collect(file)
        } else if false {
          erro(ctx, "nil: %v: %T %v (%s, %v)", rVal, lVal, lVal, o.dir, a).debug(1)
        }
      } else {
        warn(ctx, "TODO: wildcard: 3. %v %v %s", lVal, rVal, o.dir)
      }
    }
    g.Done()
  }

  var f1 = func(inVal, mapVal Value, inPat, mapPat bool, fm *FileMap) {
    if y, _, _ := inVal.match(ctx, mapVal); y { // e.g. inVal=**.am <-> mapVal=foo/bar/*.am
      g.Add(1) ; go f0(inVal, mapVal, inPat, mapPat, fm)
    } else if y, _, _ = mapVal.match(ctx, inVal); y { // e.g. mapVal=**.am <-> inVal=foo/bar/*.am
      if g.Add(1) ; true {
        go f0(inVal, mapVal, inPat, mapPat, fm)
      } else {
        go f0(mapVal, inVal, mapPat, inPat, fm)
      }
    } else {
      warn(ctx, "TODO: wildcard: %v %v", mapVal, inVal).debug(1)
    }
    g.Done()
  }

  var f2 = func(inVal Value, inPat bool, c *valcache) {
    var fm, y = c._val.(FileMap)
    if y && fm.filemap != nil {
      for _, mapVal := range fm.primePatterns(ctx) {
        g.Add(1) ; go f1(inVal, mapVal, inPat, mapVal.patterned(ctx), &fm)
      }
    } else {
      erro(ctx, "Not FileMap: %T %v", c._val, c._val).debug(1)
    }
    g.Done()
  }

  var f3 = func(inVal Value) {
    var inPat = inVal.patterned(ctx)
    for _, c := range inVal.collect(ctx, &p.filemap, cacheMatchPatts) {
      g.Add(1) ; go f2(inVal, inPat, c)
    }
    g.Done()
  }

  for _, inVal := range patterns { g.Add(1) ; go f3(inVal) }; g.Wait()
  return
}

func files(ctx Context, iname interface{}, projects ...*Project) (maps []matchedFileMap) {
  var a, b, c, d []matchedFileMap // four sections
  var ms = ctx.unmap(ctx, iname)

  if len(projects) == 0 {
    projects = append(projects, ctx.Project())
  }

outer:
  for _, m := range ms {
    for _, p := range projects {
      if m.project == p {
        a = append(a, m) ; continue outer
      } else if p.hasBase(m.project) {
        b = append(b, m) ; continue outer
      } else if t := p.configure; t != nil && (m.project == t || t.hasBase(m.project)) {
        c = append(c, m) ; continue outer
      } else {
        d = append(d, m) ; continue outer
      }
    }
  }

  maps = append(a, b...)
  if true  && len(maps) == 0 { maps = c }
  if false && len(maps) == 0 { maps = d }
  return
}

func (p *Project) selectFiles(ctx Context, maps []matchedFileMap) (files []*File) {
  for _, m := range maps {
    if m.project == p {
      // mine
    } else if p.hasBase(m.project) {
      // base files
    } else if t := p.configure; t != nil && (m.project == t || t.hasBase(m.project)) {
      // configure files
    } else {
      continue
    }

    var f *File //, _ = toFile(m.pattern)
    // if filepath.IsAbs(m.name) {
    //   f = m.stat(ctx, "", m.name)
    // } else {
    //   if m.project.changedWD != "" { f = m.stat(ctx, m.project.changedWD, m.name) }
    //   if f == nil { f = m.stat(ctx, m.project.absPath, m.name) }
    //   if f == nil {
    //     if p.changedWD != "" { f = m.stat(ctx, p.changedWD, m.name) }
    //     if f == nil { f = m.stat(ctx, p.absPath, m.name) }
    //   }
    // }

    f = m.stat(ctx, m.name)
    if f != nil {
      f.filemap = &m.FileMap
      files = append(files, f)
    }

    if false && strings.HasSuffix(m.name, ".log") {
      info(ctx, "%v: %v %v → %v %v\n", p, m.FileMap, m.name, f, files).debug(16)
    }
  }
  return
}

func (p *Project) selectFile(ctx Context, maps []matchedFileMap) (file *File) {
  if a := p.selectFiles(ctx, maps); len(a) > 0 { if file = a[0]; !file.exists() {
    for _, f := range a { if f.exists() { return f } }
  }}
  return
}

func (p *Project) file(ctx Context, iname interface{}) (file *File) {
  if file = p.selectFile(ctx, /*files(ctx, iname, p)*/ctx.unmap(ctx, iname)); false && file == nil {
    var s, d string // TODO: s = iname
    if s != "" {
      if !filepath.IsAbs(s) { d = p.absPath }
      file = stat(ctx, s, "", d)
    }
  }
  return
}

func (p *Project) tempFile(ctx Context, name string) (file *File) {
  if file = p.file(ctx, name); file != nil {
    // good
  } else if ctd := p.scope.FindDef("CTD"); ctd == nil {
    erro(ctx, "%v: CTD is not defined for temp file: %v", p, name).debug(1)
  } else if file = stat(ctx, filepath.Join(ctd.Strval(ctx), name), "", "", nil); file == nil {
    erro(ctx, "%v: nil stat %v %v", p, ctd.Strval(ctx), name).debug(1)
  } else if false {
    warn(of(ctx,ctd), "using default temp file: %v/%v", ctd.Strval(ctx), name)
    warn(at(ctx,p.position), "suggesting define files rule for '%s' in %v", name, p).debug(12)
  }
  return // NOTE: temp file may not exists
}

func (p *Project) configuration(ctx Context) (file *File) {
  var s = []*Scope{ p.scope }
  if p.configure != nil { s = append(s, p.configure.scope) }
  if file = p.tempFile(closureWith(ctx, s...), configuration_sm); file == nil {
    erro(ctx, "%v: no file configuration.sm", p).debug(1)
  }
  return
}

type cacher struct { generalOpts }

func (opts *cacher) cache(ctx Context, patts, paths []Value) {
  var p = ctx.Project()
  if p == nil { erro(ctx, "nil project").debug(1) ; return }

  var bits = cacheStore // cacheMatchPatts
  for mi, m := range ctx.universe().cache(ctx, p, patts, paths) {
    var ctx = of(ctx, m.pattern)
    for i, pat := range xmerge(ctx, plain, m.pattern) {
      if pat.expandable(ctx, plain) {
        p.filemapx = append(p.filemapx, &valcache_kv{ pat, m })
      } else if c := p.filemap.slot(ctx, pat, bits|cacheKey); c != nil && c._val == nil {
        c._val = m
      } else if c != nil && c._val != nil {
        if t, y := c._val.(FileMap); y {
          if t.filemap == m.filemap && eq(ctx, t.pattern, pat) {
            if opts.silent {/* silent, simply ignore duplications */} else
            if foundDup := -1; /* (opts.debug>0 || opts.verbose) && */true {
              for i, t := range patts {
                if eq(ctx, pat, t) {
                  if foundDup < 0 && i > 0 && i-foundDup>1 { info(ctx, "pats[%d...] ...", i) }
                  info(of(ctx, t), "patts[%d]: %T %v, %v", i, t, t, paths)
                  foundDup = i
                }
                if 0 <= foundDup && i-foundDup == 3 {
                  info(ctx, "patts[%d...%d] ... (%v %v)", i, len(patts), pat, t)
                }
              }
              d := warn(of(ctx,t.pattern), "%d. duplication: %v (%T, in %d patts)", mi, c._key, t.pattern, len(patts))
              if true { warnstack(of(ctx,t.pattern), 3).debug(10) } else { d.debug(1) }
            }

            continue // duplications are okay to go
          }

          erro(of(ctx,t.pattern), "valcache conflict: %v: t=%v %p=%v", t.project, t, t.filemap, t.filemap)
        } else {
          erro(ctx, "valcache conflict: %T %v", c._val, c._val)
        }
        erro(of(ctx,pat), "valcache conflict: %v: m=%v %p=%v", m.project, m, m.filemap, m.filemap)
        errostack(ctx, 3, "valcache duplicated in %d patts", len(patts)).debug(1)
      } else {
        erro(ctx, "valcache slot: %v: %d: %T %v", m.pattern, i, pat, pat).debug(1)
      }
    }
  }
}

func file(c Context, s string) *File { return c.Project().file(c, s) }
func resolveTempFile(c Context, s string) *File { return c.Project().tempFile(c, s) }
func resolveObject(c Context, s string) Object {
  if scope := c.Scope(); scope != nil { if o := scope.Resolve(s); o != nil { return o }}
  return c.Project().resolveObject(c, s)
}
func resolveEntries(c Context, s string, a bool) *resolvedEntries { return c.Project().resolveEntries(c, s, a) }
func resolvePatterns(c Context, v Value, s string) []*stemmed { return c.Project().resolvePatterns(c, v, s) }

func (p *Project) resolveObject(ctx Context, s string) (obj Object) {
  if p != nil && p.scope != nil { if _, obj = p.scope.Find(s); isNull(obj) {
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
    if isNull(obj) && p.configure != nil && ctx.isConfigure() {
      obj = p.configure.resolveObject(ctx, s)
    }
  }}
  return
}

func (p *Project) resolveEntries(ctx Context, name interface{}, alwaysResolveBases bool) (entries *resolvedEntries) {
  var add = func(a ...Entry) {
    if len(a) > 0 {
      if entries == nil { entries = new(resolvedEntries) }
      entries.add(a[0])
      entries.all = append(entries.all, a[1:]...)
    }
  }

  var cache *valcache
  var bits = cacheMatchPatts
  if s, y := name.(string); y {
    if cache = p.entries.strx(ctx, s, bits); cache != nil {
      // good
    } else if c := p.entries.strx(ctx, "''", bits); c != nil {
      if c = c.strx(ctx, s, bits); c != nil {
        errostack(ctx, 3, "%s: no such entry, do you mean '%s'?", s, s).debug(16)
        return
      }
    } else if c := p.entries.strx(ctx, "\"\"", bits); c != nil {
      if c = c.strx(ctx, s, bits); c != nil {
        errostack(ctx, 3, "%v: no such entry, do you mean \"%s\"?", s, s).debug(16)
        return
      }
    }
  } else if v, y := name.(Value); y {
    if cache = p.entries.slot(ctx, v, bits); cache != nil {
      // good
    } else if c := p.entries.strx(ctx, "''", bits); c != nil {
      var s = v.Strval(ctx)
      if c = c.strx(ctx, s, bits); c != nil {
        errostack(ctx, 3, "%T %v: no such entry, do you mean '%s'?", v, v, s).debug(16)
        return
      }
    } else if c := p.entries.strx(ctx, "\"\"", bits); c != nil {
      var s = v.Strval(ctx)
      if c = c.strx(ctx, s, bits); c != nil {
        errostack(ctx, 3, "%T %v: no such entry, do you mean \"%s\"?", v, v, s).debug(16)
        return
      }
    }
  } else {
    errostack(ctx, 3, "%s: no such entry, do you mean '%s'?", s, s).debug(16)
    return
  }

  if cache != nil && cache._val != nil {
    add(cache._val.(*rule))
  }

  if p.configure != nil && ctx.isConfigure() {
    var t = p.configure.resolveEntries(ctx, name, true)
    if t != nil && t.all != nil { add(t.all...) }
  }

  if alwaysResolveBases || entries == nil {
    for _, base := range p.bases {
      if t := base.resolveEntries(ctx, name, alwaysResolveBases); t != nil {
        add(t.all...)
        break
      }
    }
  }

  if true {
    /* FAST */
  } else if entries == nil { /* SLOW */
    for _, use := range p.use.list {
      if t := use.project.resolveEntries(ctx, name, alwaysResolveBases); t != nil {
        add(t.all...)
        break
      }
    }
  }
  return
}

func (p *Project) resolvePatterns(ctx Context, v Value, s string) (res []*stemmed) {
  var t1, t2 time.Time

  defer func(t0 time.Time) {
    var t = time.Now()
    if d := t.Sub(t0); d > 1*time.Second {
      var ( d1 = t1.Sub(t0) ; d2 = t2.Sub(t1) ; d3 = t.Sub(t2) ; n int )
      var a = autoVal(ctx, "@")
      for sc := ctx.sc(); sc != nil; n += 1 {
        if c := sc.inner(); c != nil { sc = c.sc() } else { break }
      }

      var pos = ctx.Position()
      prompt(ctx, "%v: slow: %v: %v, %v %v %v", pos, p, d, d1, d2, d3)
      prompt(ctx, "%v: slow: %v: %v: %v %v, %d nests", pos, p, a, v, p.patterns, n).debug(4)

      for _, pat := range p.patterns {
        var ( pt = pat.target ; pa = pat.arged )
        var full, r, stems = pt.match(ctx, s)
        var m = joinMatchRes(ctx, r)
        prompt(ctx, "%v: slow: %v%v: %v: %v %v %v, %v ; %v", pos, pt, pa, s, full, r, stems, m)
      }
      warnstack(ctx, 3).debug(6)
    }
  } (time.Now())

  if res, t1, t2 = p.resolvePatterns123(ctx, v, s); false && len(res) > 0 {
    for _, t := range res {
      if file, _ := toFile(t.target); file != nil {
        file.position = t.position
      } else if file = p.file(ctx, s); file != nil {
        file.position = t.position
        t.target = file
      }
    }
  }
  return
}

func (p *Project) resolvePatterns123(ctx Context, v Value, s string) (res []*stemmed, t1, t2 time.Time) {
  if true  { res = append(res, p.resolvePatterns1(ctx, v, s)...) } ; t1 = time.Now()
  if true  { res = append(res, p.resolvePatterns2(ctx, v, s)...) } ; t2 = time.Now()
  if false { res = append(res, p.resolvePatterns3(ctx, v, s)...)/* heavy work, VERY SLOW! */ }
  return
}

func (p *Project) resolvePatterns1(ctx Context, val Value, s string) (res []*stemmed) {
  defer func(t0 time.Time) {
    var t = time.Now()
    if d := t.Sub(t0); d > 1*time.Second {
      var pos = ctx.Position()
      prompt(ctx, "%v: slow: %v %v", pos, val, d).debug(1)
    }
  } (time.Now())

ForPatterns:
  for _, pat := range p.patterns {
    if full, r, stems := pat.target.match(ctx, s); full {
      var m = joinMatchRes(ctx, r)

      if true {
        for sc := ctx.sc(); sc != nil; { // pattern loop detection
          if s := sc.stem.target.Strval(ctx); s == m {
            continue ForPatterns // break the loop
          }
          if c := sc.inner(); c != nil { sc = c.sc() } else { break }
        }
      }

      if pa := pat.arged; len(pa) > 0 {
        var y bool
        var t1 = time.Now()
        var av = xmerge(ctx, plain, pa...)
        var t2 = time.Now()
        for _, a := range av { if y, _, _ = a.match(ctx, s); y { break } }

        var t3 = time.Now()
        if d := t3.Sub(t1); d > 1*time.Second {
          var ( d2 = t2.Sub(t1) ; d3 = t3.Sub(t2) )
          var ( p = ctx.Position() ; pt = pat.target )
          prompt(ctx, "%v: slow: %v, %v→%d; %v⇒%v+%v", p, pt, pa, len(av), d, d2, d3).debug(1)
        }

        if !y { continue ForPatterns }
      }

      res = append(res, &stemmed{pat, val, stems})
    }
  }
  return
}

func (p *Project) resolvePatterns2(ctx Context, val Value, s string) (res []*stemmed) {
  for _, base := range p.bases {
    var a, _, _ = base.resolvePatterns123(ctx, val, s)
    res = append(res, a...)
  }
  if p.configure != nil && ctx.isConfigure() {
    var a, _, _ = p.configure.resolvePatterns123(ctx, val, s)
    res = append(res, a...)
  }
  return
}

func (p *Project) resolvePatterns3(ctx Context, val Value, s string) (res []*stemmed) {
  for _, use := range p.use.list {
    var a, _, _ = use.project.resolvePatterns123(ctx, val, s)
    res = append(res, a...)
  }
  return
}

func (p *Project) entry(ctx Context, special specialRule, options []Value, target Value, prog *program) (entry Entry, err error) {
  var name string
  if name = target.Strval(ctx); name == "" {
    erro(of(ctx, target), "empty target name: %v", target).debug(1)
    return
  }

  var patterned = target.patterned(ctx)
  if true && !patterned {
    // NOTE: it should work too if not checking against files
    switch target.(type) {
    case *File, *Path, *barefile, *PercPattern, *GlobPattern, *RegexpPattern:
    default:
      if file := p.file(ctx, name); file != nil {
        file.position = target.Position()
        target = file
      }
    }
  }

  defer func() {
    if entry != nil && err == nil {
      entry.setPrograms(append(entry.programs(), prog))
    }
  } ()

  // The 'use' rule entries.
  var closured = target.expandable(ctx, expandClosure)
  if special == specialRuleUse && !closured {
    var _, _ = _opts[struct{
      postExec bool `p,post;pe,post-execute;pe,post-exec`
    }](ctx, expandZero, options...)
    panic(":use: rule entry is deprecated")
    return
  }

  var arged []Value // e.g. for pattern filtering
  switch t := target.(type) {
  case *group:
    erro(ctx, "group target not supported: %v", t).debug(1)
    return
  case *argumented:
    target, arged = t.Value, merge(t.args...)
  }

  var cache = p.entries.slot(ctx, target, cacheKey|cacheStore|cacheNoConflict)
  if cache == nil {
    erro(ctx, "no cache for target: %v", target).debug(1)
    return
  }

  if cache._val == nil {
    var rule = &rule{
      position: target.Position(), class: GeneralRule, target: target, arged: arged,
    }

    if patterned {
      if _, y := target.(*Path); y {
        rule.class = PathPattRule
      } else {
        rule.class = PatternRule
      }
      p.patterns = append(p.patterns, rule)
    }

    entry = rule
    cache._val = rule
  } else if p, y := cache._val.(*rule); y { entry = p } else {
    errostack(ctx, 3, "wrong cache: %T %v", cache._val, cache._val).debug(1)
    return
  }

  if entry != nil && p.defaultEntry == nil { p.defaultEntry = entry }
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
    erro(at(ctx,p.position), "%v: %v", p, err).debug(128)
    return
  } else if depth > 128 {
    err = fmt.Errorf("exceeds maximum base depth (%d) (start=%v, target=%v)", depth, top, proj)
    erro(at(ctx,p.position), "%v: %v", p, err)
    erro(at(ctx,top.position), "start: %v", top)
    erro(at(ctx,proj.position), "target: %v", proj).debug(200)
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
      erro(at(ctx,top.position), "start: %v", top)
      erro(at(ctx,proj.position), "stop: %v", proj)
      erro(at(ctx,p.position), "%v: %v", p, err).debug(128)
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
    prompt(ctx, "entering: %v (%v)\n", dir, ctx.Project().name)
  }

  var wd string
  if wd, err = os.Getwd(); err != nil { return }
  if err = lockCD(dir, 0); err != nil { return }
  if !filepath.IsAbs(dir) { dir = filepath.Join(wd, dir) }
  autoSet(ctx, "CWD", MakeString(ctx.program().position, dir))

  var ( enter *enterec ; ok bool )
  if enter, ok = cd.enters[dir]; !ok {
    enter = &enterec{ wd:wd, dir:dir }
    cd.enters[dir] = enter
  }
  enter.num += 1
  cd.stack = append([]*enterec{enter}, cd.stack...)
  return
}

func leave(ctx Context, prog *program, stop *enterec) (err error) {
  cd.mutex.Lock(); defer cd.mutex.Unlock()

  var size = len(cd.stack)
  if options.traceEntering {
    prompt(ctx, "leaving: %v (%v %v %v)\n", stop.dir, prog.project.name, stop.num, size)
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
  cd.mutex.Lock() ; defer cd.mutex.Unlock()
  if size := len(cd.stack); size > 0 {
    var enter = cd.stack[0]
    if enter.silent { return }
    for _, p := range cd.stack {
      if p.print && p != enter {
        p.print = false
        diag(ctx, diagPromptNL, "smart:  Leaving directory '%s'", p.dir)
      }
    }
    if !enter.print {
      enter.print = true
      diag(ctx, diagPromptNL, "smart: Entering directory '%s'", enter.dir)
    }
  }
}

func printLeavingDirectory(ctx Context) {
  cd.mutex.Lock() ; defer cd.mutex.Unlock()
  if size := len(cd.stack); size > 0 {
    for _, enter := range cd.stack {
      if enter.print {
        enter.print = false
        diag(ctx, diagPromptNL, "smart:  Leaving directory '%s'", enter.dir)
      }
    }
  }
}
