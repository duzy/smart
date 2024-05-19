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
  "os"
)

const configuration_sm = "configuration.sm"
const pathSepByte = filepath.Separator
const pathSep = string(pathSepByte)

type _filemap struct {
  project *project
  patts []Value
  paths []Value
}

type filemap struct {
  *_filemap
  pattern Value
}

func (p *_filemap) String() (s string) {
  if n := len(p.patts); n == 1 {
    s = p.patts[0].String()
  } else if n > 1 {
    s = fmt.Sprintf("%s", p.patts)
  }
  return
}

func (p filemap) String() (s string) {
  if p.pattern == nil {
    s = p._filemap.String()
  } else {
    s = p.pattern.String()
  }
  return
}

func (p *filemap) primePatterns(ctx Context) (pats []Value) {
  var patts = []Value{ p.pattern }
  if patts[0] == nil { patts = p.patts }

  for _, pattern := range patts {
    // NOTE it may preserve closure patterns after this expand:
    var pat = pattern.expand(ctx)
    if isFinalValue(ctx, pat) {
      pats = append(pats, merge(pat)...)
    } else {
      erro(at(ctx,pattern), "unexpanded file pattern: %v", us(pat))
      errostack(at(ctx,pattern), 3, "%v", us(ctx)).debug(15)
      return nil
    }
  }
  return
}

// match split filename into list and match each part with the pattern correspondingly.
func (filemap *filemap) match(ctx Context, val interface{}) (_ bool, _ Value, _ string) {
  // TODO: escape file matching for 'String' and "compound" values
  for _, pat := range filemap.primePatterns(ctx) {
    if matched, name := filemap._match(ctx, pat, val); matched {
      return matched, pat, name
    }
  }
  return
}

func (filemap *filemap) _match(ctx Context, pat Value, val interface{}) (matched bool, name string) {
  // TODO: escape file matching for 'String' and "compound" values
  var res interface{}
  matched, res, _ = pat.match(ctx, val)

  if false && !matched && !(isNone(pat) || isNull(pat)) {
    var str string // NOOP
    if n := strings.Index(str, pathSep); n < 0 { return }

    // NOTE: Dealing with these files:
    //     files (
    //         (foo.c) => $(srcdir)/sub/dir
    //         (sub/dir/foo.c) => $(srcdir)
    //     )
    for _, p := range filemap.paths { // FIXME: performance, operate on p.(*path) instead
      if _, ok := p.(*path); !ok { continue } // NOTE: only work with paths to improve performance
      var ps = p.string(ctx)
      for i := strings.LastIndex(ps, pathSep); -1 <= i; {
        var ( prefix = ps[i+1:]; l = len(prefix) ) // NOTE: -1 <= i < len(ps)
        if has := strings.HasPrefix(str, prefix) && str[l] == '/'; has {
          if matched, _, _ = pat.match(ctx, str[len(prefix)+1:]); matched { break }
        }
        if 0 < i { i = strings.LastIndex(ps[:i], pathSep) } else { break }
      }
    }
  }

  if res == nil {
    // okay
  } else if s, y := res.(string); y {
    name = s
  } else if a, y := res.([]string); y {
    name = joinPath(a...)
  } else {
    erro(ctx, "unexpected result: %T %v", res, res).debug(1)
  }
  return // NOTE: also `globMatchFile(ctx, pat, str, true)`
}

func (p *filemap) stat(ctx Context, name string) (file *File) {
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

  if len(p.paths) == 0 {
    for _, pat := range p.patts {
      if f, y := pat.(*File); y && f.ident(ctx) == name { return f }
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
  for _, path := range p.paths {
    if isNull(path) {
      erro(at(ctx,pos), "nil path: name=%s",  name)
      erro(at(ctx,pos), "nil path: %v", p).debug(32)
      panic(_failure(ctx))
    } else if isNone(path) {
      warn(at(ctx,pos), "nil path: name=%s",  name)
      warn(at(ctx,pos), "nil path: %v", p).debug(32)
      continue
    }

    var dir, sub string

    if sub = path.string(ctx); sub == "" {
      if true {
        erro(at(ctx,path.Position()), "filemap path '%v' is empty (%T)", path, path)
        erro(at(ctx,pos), "filemap path '%v' is empty (pattern=%v)", path, patts)
        erro(ctx, "filemap path '%v' is empty (project=%v)", path, ctx.project())//.at(pos)
        erro(ctx, "filemap path '%v' is empty in %v", path, ctx).debug(64)
      }
      return
    } else if s := filepath.Clean(sub); sub != s {
      sub = s
    }

    if filepath.IsAbs(sub) {    // 'sub' is abs
      if filepath.IsAbs(name) { // 'name' is abs too
        if s := sub+pathSep; strings.HasPrefix(name, s) { // 'name' should have 'sub' prefix
          name = strings.TrimPrefix(name, s)
        } else {
          continue
        }
      }
    } else if !filepath.IsAbs(name) {
      // dir = filepath.Clean(base)
    }

    if file = stat(ctx, name, stat_sub{sub}, stat_dir{dir}, stat_nonexist{true}); file != nil { break }

    var pre string // Not used!
    if filepath.IsAbs(sub) {
      if pre == "" { // Fullmatch!
        // For example of:
        //   xxx.c  <->  (*.c => /path/to/source)
        // Become:
        //   /path/to/source  ""  xxx.c
        file = stat(ctx, name, stat_dir{sub}, stat_nonexist{true})
      } else if strings.HasSuffix(sub, pathSep+pre) {
        // For example of:
        //   foo/bar/xxx.c  <->  (*.c => /path/to/source/foo/bar)
        // Become:
        //   /path/to/source  foo/bar  xxx.c
        s := strings.TrimSuffix(sub, pathSep+pre)
        n := strings.TrimPrefix(name, pre+pathSep)
        file = stat(ctx, n, stat_sub{pre}, stat_dir{s}, stat_nonexist{true})
      } else if false { // This is wrong, only base name matched!!
        // For example of:
        //   foo/bar/xxx.c  <->  (*.c => /path/to/source)
        // Become:
        //   /path/to/source  foo/bar  xxx.c
        n := strings.TrimPrefix(name, pre+pathSep)
        file = stat(ctx, n, stat_sub{pre}, stat_dir{sub}, stat_nonexist{true})
      }
    } else {
      if pre == "" { // Fullmatch!
        // For example of:
        //   xxx.c  <->  (*.c => source)
        // Become:
        //   <p.absPath>  source  xxx.c
        file = stat(ctx, name, stat_sub{sub}, stat_dir{dir}, stat_nonexist{true})
      } else if sub == pre {
        // For example of:
        //   foo/bar/xxx.c  <->  (*.c => foo/bar)
        // Become:
        //   <dir>  foo/bar  xxx.c
        n := strings.TrimPrefix(name, pre+pathSep)
        file = stat(ctx, n, stat_sub{sub}, stat_dir{dir}, stat_nonexist{true})
      } else if strings.HasSuffix(sub, pathSep+pre) {
        // For example of:
        //   foo/bar/xxx.c  <->  (*.c => source/foo/bar)
        // Become:
        //   <dir>  source/foo/bar  xxx.c
        s := strings.TrimSuffix(sub, pathSep+pre)
        n := strings.TrimPrefix(name, pre+pathSep)
        file = stat(ctx, n, stat_sub{pre}, stat_dir{s}, stat_nonexist{true})
      } else if false { // This is wrong, only base name matched!!
        // For example of:
        //   foo/bar/xxx.c  <->  (*.c => source)
        // Become:
        //   <dir>  source/foo/bar  xxx.c
        s := filepath.Join(sub, pre)
        n := strings.TrimPrefix(name, pre+pathSep)
        file = stat(ctx, n, stat_sub{s}, stat_dir{dir}, stat_nonexist{true})
      }
    }
  }
  return
}

type project_box struct { *project }
func (p project_box) cmp(ctx Context, v Value) (res cmpres) {
    if x, y := v.(project_box); y {
        if x.project == p.project { res = cmpEqual }
    } else if x, y := v.(condval); y {
        res = p.project.cmp(ctx, x.Value)
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        res = p.project.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        switch v.(type) { case project_box, *project, *bareword, *barecomp: return }
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v != %v, %v", us(p), us(v), res).debug(5)
            panic(_failure(ctx))
        }
    }
    return
}

type project struct {
  position Position
  keyword  token // aka kind: project, package, module

  configurationFile *File // default file, decided at load time
  configure *project // .configure
  configured bool

	absPath string
	relPath string
  tmpPath string
	name    string
	spec    string

  scope   *Scope

  bases []*project
  use     *uselist

  filemapx []*_DEPRECATED_vcache_kv // closure cache
  filemap     _DEPRECATED_vcache
  entries     _DEPRECATED_vcache
  patterns []*rule // order is important
  configs []entry // configure entries
  defaultEntry entry

  plugin *plugin.Plugin
  pluginScope *Scope

  opts *projectDeclOpts
}
func (_ *project) kind() Kind { return KindObject|KindKnownObject|Kindproject }
func (_ *project) int(Context) (int64, error) { return 0, nil }
func (_ *project) float(Context) (float64, error) { return .0, nil }
func (_ *project) updated(Context) bool { return false }
func (_ *project) updatedDeps(Context, ...Value) []Value { return nil }
func (_ *project) stamp(ctx Context) (files []*File, err error) { return }
func (_ *project) delete(Context) (files []*File, err error) { return }
func (_ *project) defs(Context, ...string) (res []*def) { return }
func (_ *project) refs(Context, Value) (res bool) { return }
func (_ *project) patterned(Context) bool { return false }
func (_ *project) expandable(Context) bool { return false }
func (p *project) expand(Context) Value { return project_box{p} }
func (p *project) evoke(ctx *evocation) (res Value) { return project_box{p} }
func (p *project) stencil(ctx Context, stems []string) (Value, []string) { return p, stems }
func (p *project) match(ctx Context, i interface{}) (bool, interface{}, []string) { return stringMatch(ctx, p, i) }
func (p *project) Position() Position { return p.position }
func (p *project) String() string { return p.name }
func (p *project) string(Context) string { return p.name }
func (p *project) ident(Context) string { return p.name }
func (p *project) true(Context) bool { return p.name != "" }
func (p *project) declScope() *Scope { return p.scope }
func (p *project) owner() *project { return p.scope.project }
func (p *project) get(ctx Context, s string) Value { return p.resolve(ctx, s) }
func (p *project) traverse(ctx Context) {
    if t := p.defaultEntry; t != nil {
        switch t.Target().(type) { case flag: return }
        t.traverse(ctx)
    }
}
func (p *project) stat(ctx Context) (si *statinfo) {
    if t := p.defaultEntry; t != nil { si = t.stat(ctx) }
    return
}
func (p *project) cmp(ctx Context, v Value) (res cmpres) {
    if x, y := v.(*project); y {
        if x == p { res = cmpEqual }
    } else if x, y := v.(condval); y {
        res = p.cmp(ctx, x.Value)
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        res = p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        switch v.(type) { case project_box, *project, *bareword, *barecomp: return }
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v != %v, %v", us(p), us(v), res).debug(5)
            panic(_failure(ctx))
        }
    }
    return
}

type self struct { *project }
func (p self) kind() Kind { return p.project.kind()|KindSelf }
func (_ self) ident(Context) string { return ".self" }
func (p self) String() string { return p.srclit(nil, nil) }
func (_ self) srclit(Context, Object) string { return "$(.self)" }
func (p self) expand(Context) Value { return p }
func (p self) cmp(ctx Context, v Value) (res cmpres) {
    if x, y := v.(self); y {
        res = p.project.cmp(ctx, x.project)
    } else if x, y := v.(*project); y && false {
        if p.project == x { res = cmpEqual }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        res = p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        switch v.(type) { case project_box, *project, *bareword, *barecomp: return }
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v != %v, %v", us(p), us(v), res).debug(5)
            panic(_failure(ctx))
        }
    }
    return
}

func (p *project) AbsPath() string { return p.absPath }
func (p *project) RelPath() string { return p.relPath }
func (p *project) Scope() *Scope { return p.scope }
func (p *project) Bases() []*project { return p.bases }
func (p *project) newScope(pos Position, comment string) *Scope {
  return newScope(pos, p.scope, p, comment)
}

func file(ctx Context, s string, projects ...*project) (res *File) {
  if len(projects) == 0 {
    projects = append(projects, ctx.project())
  }
  for _, p := range projects {
    if res = p.file(ctx, s); res != nil { break }
  }
  return
}

func files(ctx Context, iname interface{}, projects ...*project) (maps []matched_filemap) {
  var a, b, c, d []matched_filemap // four sections
  var ms = unmap_files(ctx, iname)

  if len(projects) == 0 {
    projects = append(projects, ctx.project())
  }

outer:
  for _, m := range ms { for _, p := range projects {
    if m.project == p {
      a = append(a, m) ; continue outer
    } else if p.hasBase(m.project) {
      b = append(b, m) ; continue outer
    } else if t := p.configure; t != nil && (m.project == t || t.hasBase(m.project)) {
      c = append(c, m) ; continue outer
    } else {
      d = append(d, m) ; continue outer
    }
  }}

  maps = append(a, b...)
  if true  && len(maps) == 0 { maps = c }
  if false && len(maps) == 0 { maps = d }
  return
}

func (p *project) selectFiles(ctx Context, maps []matched_filemap) (files []*File) {
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

    var f = m.stat(ctx, m.name)
    if f != nil {
      f.filemap = &m.filemap
      files = append(files, f)
    }

    if false { if strings.HasPrefix(m.name, ".configure/") && strings.HasSuffix(m.name, ".log") {
      note(ctx, "%v: %v %v → %v %v\n", p, m._filemap, m.name, f, files).debug(16)
    }}
  }
  return
}

func (p *project) selectFile(ctx Context, maps []matched_filemap) (file *File) {
  if a := p.selectFiles(ctx, maps); len(a) > 0 { if file = a[0]; !file.exists() {
    for _, f := range a { if f.exists() { return f } }
  }}
  return
}

func (p *project) file(ctx Context, iname interface{}) (file *File) {
  return p.selectFile(ctx, unmap_files(ctx, iname))
}

func (p *project) tempFile(ctx Context, name string) (file *File) {
  if false { ctx = closureWith(ctx, p.scope) }

  if file = p.file(ctx, name); file != nil {
    return
  }

  if d := p.resolveDef(ctx, "CTD"); d == nil {
    erro(ctx, "%v: $(CTD) is not defined: %v", p, name).debug(1)
  } else if ctd := d.value; isTrivial(ctd) {
    erro(ctx, "%v: $(CTD) is trivial for %v", p, name).debug(1)
  } else if file = stat(ctx, name, stat_dir{ctd.string(ctx)}, stat_nonexist{true}); file == nil {
    erro(ctx, "%v: not a file: %v (%v)", p, name, ctd.string(ctx)).debug(1)
  }
  return
}

func (p *project) configuration(ctx Context) (file *File) {
  var s = []*Scope{ p.scope }
  if p.configure != nil { s = append(s, p.configure.scope) }
  if file = p.tempFile(closureWith(ctx, s...), configuration_sm); file == nil {
    erro(ctx, "%v: no file configuration.sm", p).debug(1)
  }
  return
}

type cacher struct { generalOpts }

func (opts *cacher) cache(ctx Context, patts, paths []Value) {
  defer trace(ctx)

  var p = ctx.project()
  if p == nil {
    erro(ctx, "nil project").debug(1)
    return
  }

  var bits = cacheStore // cacheMatchPatts
  for mi, m := range _universe(ctx).filemap(ctx, p, patts, paths) {
    var ctx = at(ctx, m.pattern)
    for i, pat := range xmerge(ctx, m.pattern) {
      if pat.expandable(ctx) {
        p.filemapx = append(p.filemapx, &_DEPRECATED_vcache_kv{ pat, m })
      } else if c := p.filemap.slot(ctx, pat, bits|cacheKey); c == nil {
        erro(ctx, "valcache slot: %v: %v", us(pat), us(m)).debug()
      } else if c._val == nil {
        c._val = m
      } else {
        if t, y := c._val.(filemap); y {
          if t._filemap == m._filemap && eq(ctx, t.pattern, pat) {
            if opts.silent {/* silent, simply ignore duplications */} else
            if foundDup := -1; /* (opts.debug>0 || opts.verbose) && */true {
              for i, t := range patts {
                if eq(ctx, pat, t) {
                  if foundDup < 0 && i > 0 && i-foundDup>1 { info(ctx, "pats[%d...] ...", i) }
                  info(at(ctx, t), "patts[%d]: %v, %v", i, us(t), paths)
                  foundDup = i
                }
                if 0 <= foundDup && i-foundDup == 3 {
                  info(ctx, "patts[%d...%d] ... (%v %v)", i, len(patts), pat, t)
                }
              }
              d := warn(at(ctx,t.pattern), "%d. duplication: %v (%T, in %d patts)", mi, c._key, t.pattern, len(patts))
              if true { warnstack(at(ctx,t.pattern), 3).debug(10) } else { d.debug(1) }
            }

            continue // duplications are okay to go
          }

          erro(at(ctx,t.pattern), "valcache conflict: %v: t=%v %p=%v", t.project, t, t._filemap, t._filemap)
        } else {
          erro(ctx, "valcache conflict: %T %v", c._val, c._val)
        }
        erro(at(ctx,pat), "valcache conflict: %v: m=%v %p=%v", m.project, m, m._filemap, m._filemap)
        errostack(ctx, 3, "valcache duplicated in %d patts", len(patts)).debug(1)
      }
    }
  }
}

func resolveTempFile(c Context, s string) *File { return c.project().tempFile(c, s) }
func resolve(c Context, s string) Object { return c.project().resolve(c, s) }
func resolveEntries(c Context, s string, a ...bool) entryArray { return c.project().resolveEntries(c, s, a...) }
func resolvePatterns(c Context, v Value, s string) []*stemmed { return c.project().resolvePatterns(c, v, s) }

func (p *project) resolveDef(ctx Context, name string) (res *def) {
  if o := p.resolve(ctx, name); o != nil { res, _ = o.(*def) }
  return
}

func (p *project) resolve(ctx Context, s string) (obj Object) {
  if p != nil && p.scope != nil { if _, obj = p.scope.find(s); obj == nil {
    if p.pluginScope != nil { if obj = p.pluginScope.Lookup(s); obj != nil {
        return
    }}
    for _, base := range p.bases {
      if obj = base.resolve(ctx, s); obj != nil {
        break
      }
    }
    if obj == nil && p.configure != nil && p.configure != p {
      obj = p.configure.resolve(ctx, s) // isConfigure(ctx)
    }
  }}
  return
}

var testResolveEntries bool

func (p *project) resolveEntries(ctx Context, name interface{}, _b ...bool) (entries entryArray) {
  var add = func(a ...entry) {
    if len(a) > 0 {
      // if entries == nil { entries = new(entryArray) }
      // entries.add(a[0])
      // entries = append(entries, a[1:]...)
      entries = append(entries, a...)
    }
  }

  var cache *_DEPRECATED_vcache
  var bits = cacheMatchPatts
  if s, y := name.(string); y {
    if cache = p.entries.strx(ctx, s, bits); cache != nil {
      // good
    } else if c := p.entries.strx(ctx, "''", bits); c != nil {
      if c = c.strx(ctx, s, bits); c != nil {
        if testResolveEntries { return }
        errostack(ctx, 3, "%s: no such entry, do you mean '%s'?", s, s).debug(16)
        return
      }
    } else if c := p.entries.strx(ctx, "\"\"", bits); c != nil {
      if c = c.strx(ctx, s, bits); c != nil {
        if testResolveEntries { return }
        errostack(ctx, 3, "%v: no such entry, do you mean \"%s\"?", s, s).debug(16)
        return
      }
    }
  } else if v, y := name.(Value); y {
    if cache = p.entries.slot(ctx, v, bits); cache != nil {
      // good
    } else if c := p.entries.strx(ctx, "''", bits); c != nil {
      var s = v.string(ctx)
      if c = c.strx(ctx, s, bits); c != nil {
        errostack(ctx, 3, "%T %v: no such entry, do you mean '%s'?", v, v, s).debug(16)
        return
      }
    } else if c := p.entries.strx(ctx, "\"\"", bits); c != nil {
      var s = v.string(ctx)
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

  if p.configure != nil && isConfigure(ctx) {
    var t = p.configure.resolveEntries(ctx, name, true)
    if t != nil { add(t...) }
  }

  var alwaysResolveBases bool
  if n := len(_b); n > 0 { alwaysResolveBases = _b[n-1] }
  if alwaysResolveBases || entries == nil {
    for _, base := range p.bases {
      if t := base.resolveEntries(ctx, name, alwaysResolveBases); t != nil {
        add(t...)
        break
      }
    }
  }

  if true {
    /* FAST */
  } else if entries == nil { /* SLOW */
    for _, use := range p.use.list {
      if t := use.project.resolveEntries(ctx, name, alwaysResolveBases); t != nil {
        add(t...)
        break
      }
    }
  }
  return
}

func (p *project) resolvePatterns(ctx Context, v Value, s string) (res []*stemmed) {
  var t1, t2 time.Time

  defer func(t0 time.Time) {
    var t = time.Now()
    if d := t.Sub(t0); d > 1*time.Second {
      var ( d1 = t1.Sub(t0) ; d2 = t2.Sub(t1) ; d3 = t.Sub(t2) ; n int )
      var a = autoVal(ctx, "@")
      for sc := cast[*stemmedContext](ctx); sc != nil; n += 1 {
        if c := inner(sc); c != nil { sc = cast[*stemmedContext](c) } else { break }
      }

      var pos = ctx.Position()
      prompt(ctx, "%v: slow: %v: %v, %v %v %v", pos, p, d, d1, d2, d3)
      prompt(ctx, "%v: slow: %v: %v: %v %v, %d nests", pos, p, a, v, p.patterns, n).debug(4)

      for _, pat := range p.patterns {
        var ( pt = pat.target ; pa = pat.arged )
        var full, r, stems = pt.match(ctx, s)
        var m = _path(ctx, r)
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

func (p *project) resolvePatterns123(ctx Context, v Value, s string) (res []*stemmed, t1, t2 time.Time) {
  if true  { res = append(res, p.resolvePatterns1(ctx, v, s)...) } ; t1 = time.Now()
  if true  { res = append(res, p.resolvePatterns2(ctx, v, s)...) } ; t2 = time.Now()
  if false { res = append(res, p.resolvePatterns3(ctx, v, s)...)/* heavy work, VERY SLOW! */ }
  return
}

func (p *project) resolvePatterns1(ctx Context, val Value, s string) (res []*stemmed) {
  defer func(t0 time.Time) {
    if d := time.Now().Sub(t0); d > 1*time.Second {
      prompt(ctx, "%v: slow: %v %v", ctx.Position(), val, d).debug(1)
    }
  } (time.Now())

ForPatterns:
  for _, pat := range p.patterns {
    if full, r, stems := pat.target.match(ctx, s); full {
      var m = _path(ctx, r)

      if true { for sc := cast[*stemmedContext](ctx); sc != nil; { // pattern loop detection
        if s := sc.stem.target.string(ctx); s == m { continue ForPatterns }
        if c := inner(sc); c != nil { sc = cast[*stemmedContext](c) } else { break }
      }}

      if pa := pat.arged; len(pa) > 0 {
        var y bool
        var t1 = time.Now()
        var av = xmerge(ctx, pa...)
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

func (p *project) resolvePatterns2(ctx Context, val Value, s string) (res []*stemmed) {
  for _, base := range p.bases {
    var a, _, _ = base.resolvePatterns123(ctx, val, s)
    res = append(res, a...)
  }
  if p.configure != nil && isConfigure(ctx) {
    var a, _, _ = p.configure.resolvePatterns123(ctx, val, s)
    res = append(res, a...)
  }
  return
}

func (p *project) resolvePatterns3(ctx Context, val Value, s string) (res []*stemmed) {
  for _, use := range p.use.list {
    var a, _, _ = use.project.resolvePatterns123(ctx, val, s)
    res = append(res, a...)
  }
  return
}

func (p *project) entry(ctx Context, special specialRule, options []Value, target Value, prog *program) (entry entry, err error) {
  var name string
  if name = target.string(ctx); name == "" {
    erro(at(ctx, target), "empty target name: %v", target).debug(1)
    return
  }

  var patterned = target.patterned(ctx)
  if true && !patterned {
    // NOTE: it should work too if not checking against files
    switch target.(type) {
    case *File, *path, *barefile, *percpat, *globpat, *regexpat:
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
  var closured = target.expandable(ctx)//, expandDef2
  if special == specialRuleUse && !closured {
    var _, _ = _opts_[struct{
      postExec bool `post,post-execute,post-exec`
    }](ctx, options...)
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
      position: target.Position(), class: generalRule, target: target, arged: arged,
    }

    if patterned {
      if _, y := target.(*path); y {
        rule.class = pathPatRule
      } else {
        rule.class = patternRule
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

func (p *project) isa(proj *project) (res bool) {
  for _, base := range p.bases {
    if base == proj { res = true; break }
  }
  return
}

func (p *project) hasBase(proj *project) (res bool) {
  for _, base := range p.bases {
    if res = base == proj || base.hasBase(proj); res { break }
  }
  return
}

func (p *project) hasLoaded(ctx Context, proj *project, traveUseLoop bool) (rp *project, res, isb bool, err error) {
  var uni = _universe(ctx)
  if uni.checkLoadGraph || !uni.fastMode {
    rp, res, isb, err = p.hasLoadedRecur(ctx, p, proj, 1, traveUseLoop)
  }
  return
}

func (p *project) hasLoadedRecur(ctx Context, top, proj *project, depth int, traveUseLoop bool) (rp *project, res, isb bool, err error) {
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

func (p *project) loopLoadPath() (s string) { return p.loopLoadRecur(p) }
func (p *project) loopLoadRecur(top *project) (s string) {
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

func (p *project) isUsingproject(usee *project) (res bool) {
  for _, use := range p.use.list {
    if res = use.project == usee; res { break  }
    if res = use.project.isUsingproject(usee); res { break }
  }
  return
}

func (p *project) isUsingDirectly(proj *project) (res bool) {
  for _, u := range p.use.list {
    if res = u.project == proj; res { break }
  }
  return
}

func (p *project) usees(bases, basesRecur, useeRecur, pre bool) (res []*project) {
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

// Note: this is okay not using an atomic value, because
// chdirMutex can serve to protect the whole timeframe.
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
