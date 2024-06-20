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

type filemap struct { *_filemap ; pattern Value }

func (p *_filemap) String() (s string) {
  if n := len(p.patts); n == 1 {
    s = p.patts[0].String()
  } else if n > 1 {
    s = fmt.Sprintf("%s", p.patts)
  }
  return
}

func (p *filemap) ts(t string) string {
  return fmt.Sprintf("{=%s %v}", t, ts(p.pattern))
}
func (p *filemap) String() (s string) {
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
    // NOTE it may preserve closure patterns after this expand
    if pat := pattern.expand(ctx); isFinalValue(ctx, pat) {
      pats = append(pats, merge(pat)...)
    } else {
      erro(at(ctx,pat), "indeterminate pattern: %v", ts(pat))
      erro(at(ctx,pat), "%v", ts(ctx)).debug()
      trace(ctx)
    }
  }
  return
}

// match split filename into list and match each part with the pattern correspondingly.
func (p *filemap) match(ctx Context, val any) (_ bool, _ Value, _ string) {
  // TODO: escape file matching for 'String' and "compound" values
  for _, pat := range p.primePatterns(ctx) {
    if matched, name := p._match(ctx, pat, val); matched {
      return matched, pat, name
    }
  }
  return
}

func (p *filemap) _match(ctx Context, pat Value, val any) (matched bool, name string) {
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
    for _, p := range p.paths { // FIXME: performance, operate on p.(*path) instead
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
    erro(ctx, "unexpected result: %v", ts(res)).debug()
    trace(ctx)
  }
  return // NOTE: also `globMatchFile(ctx, pat, str, true)`
}

func (p *filemap) stat(ctx Context, name string) (file *File) {
  var patts = p.patts
  if len(patts) == 0 {
    erro(ctx, "no map patterns: %v", p).debug()
    trace(ctx)
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
    errostack(ctx, 5, "%s -> %v", name, p.patts).debug()
    trace(ctx)
  }

  var pos = patts[0].Position()
  for _, path := range p.paths {
    if isNull(path) {
      erro(at(ctx,pos), "nil path: name=%s",  name)
      erro(at(ctx,pos), "nil path: %v", p).debug()
      trace(ctx)
    } else if isNone(path) {
      warn(at(ctx,pos), "nil path: name=%s",  name)
      warn(at(ctx,pos), "nil path: %v", p).debug(32)
      continue
    }

    var dir, sub string

    if sub = path.string(ctx); sub == "" {
      if true {
        erro(at(ctx,path), "filemap path '%v' is empty (%T)", path, path)
        erro(at(ctx,pos), "filemap path '%v' is empty (pattern=%v)", path, patts)
        erro(ctx, "filemap path '%v' is empty (project=%v)", path, _project(ctx))//.at(pos)
        erro(ctx, "filemap path '%v' is empty in %v", path, ctx).debug()
        trace(ctx)
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
  } else if x, y := v.(*list); y && len(x.elems) == 1 {
    res = p.project.cmp(ctx, x.elems[0])
  }
  if checkpoints {
    switch v.(type) { case project_box, *project, *bareword, *barecomp: return }
    if res != cmpEqual && p.String() == v.String() {
      erro(ctx, "%v != %v, %v", ts(p), ts(v), res).debug()
      trace(ctx)
    }
  }
  return
}

type project_ext struct { *plugin.Plugin }
type project struct {
  *scope

  position Position
  keyword  token // aka kind: project, package, module

  bases []*project

  use *uselist

  configuration *File // default file, decided at load time
  configure *project // .configure
  configured bool

	absPath string
  tmpPath string
	name    string
	rel     string // path segment relative to the workBaseDir
	spec    string // relative to search-paths as a specification

  filemap valcache
  entries valcache

  patterns   []*rule // order is important
  configs    []entry // configure entries
  defaultEntry entry

  ext project_ext
  opt project_opt
}
func (_ *project) kind() Kind { return KindObject|KindKnownObject|KindProject }
func (_ *project) int(Context) (int64, error) { return 0, nil }
func (_ *project) float(Context) (float64, error) { return .0, nil }
func (_ *project) updated(Context) bool { return false }
func (_ *project) updatedDeps(Context, ...Value) []Value { return nil }
func (_ *project) stamp(ctx Context) (_ []*File, _ error) { return }
func (_ *project) delete(Context) (_ []*File, _ error) { return }
func (_ *project) defs(Context, ...string) (_ []*def) { return }
func (_ *project) refs(Context, Value) (_ bool) { return }
func (_ *project) patterned(Context) bool { return false }
func (_ *project) expandable(Context) bool { return false }
func (p *project) expand(Context) Value { return project_box{p} }
func (p *project) evoke(ctx *evocation) Value { return project_box{p} }
func (p *project) stencil(ctx Context, stems []string) (Value, []string) { return p, stems }
func (p *project) match(ctx Context, i any) (bool, any, []string) { return stringMatch(ctx, p, i) }
func (p *project) Position() Position { return p.position }
func (p *project) String() string { return p.name }
func (p *project) string(Context) string { return p.name }
func (p *project) ident(Context) string { return p.name }
func (p *project) true(Context) bool { return p.name != "" }
func (p *project) owner() *project { return p.scope.project }
func (p *project) sel(ctx Context, s string) any { return p.resolve(ctx, s) }
func (p *project) traverse(ctx Context) {
    if t := p.defaultEntry; t != nil {
        switch t.destiny().(type) { case flag: return }
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
          erro(ctx, "%v != %v, %v", ts(p), ts(v), res).debug()
          trace(ctx)
        }
    }
    return
}

type self struct { *project }
func (_ self) ident(Context) string { return ".self" }
func (_ self) srclit(Context, Object) string { return "$(.self)" }
func (p self) String() string { return p.srclit(nil, nil) }
func (p self) kind() Kind { return p.project.kind()|KindSelf }
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
          erro(ctx, "%v != %v, %v", ts(p), ts(v), res).debug()
          trace(ctx)
        }
    }
    return
}

func file(ctx Context, s string, projects ...*project) (_ *File) {
  if len(projects) == 0 {
    projects = append(projects, _project(ctx))
  }

  for _, p := range projects {
    if f := p.file(ctx, s); f != nil {
      return f
    }
  }
  return
}

func files(ctx Context, iname interface{}, projects ...*project) (res []filemap_name) {
  if len(projects) == 0 {
    projects = append(projects, _project(ctx))
  }

  var a, b, c, d []filemap_name // four sections
  var ms = unmap_files(ctx, iname)

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

  res = append(a, b...)

  if true  && len(res) == 0 { res = c }
  if false && len(res) == 0 { res = d }
  return
}

func (p *project) selectFiles(ctx Context, v []filemap_name) (res []*File) {
  for _, m := range v {
    if m.project == p {
      // mine
    } else if p.hasBase(m.project) {
      // base files
    } else if t := p.configure; t != nil && (m.project == t || t.hasBase(m.project)) {
      // configure files
    } else {
      continue
    }

    if f := m.stat(ctx, m.name) ; f != nil {
      f.filemap = &m.filemap
      res = append(res, f)
    }
  }
  return
}

func (p *project) selectFile(ctx Context, v []filemap_name) (res *File) {
  if a := p.selectFiles(ctx, v); 0 < len(a) {
    if res = a[0]; !res.exists() {
      for _, f := range a {
        if f.exists() { return f }
      }
    }
  }
  return
}

func (p *project) file(ctx Context, iname interface{}) (file *File) {
  return p.selectFile(ctx, unmap_files(ctx, iname))
}

func (p *project) tempFile(ctx Context, name string) (file *File) {
  if file = p.file(ctx, name); file != nil {
    return
  }

  if d := p.resolveDef(ctx, "CTD"); d == nil {
    erro(ctx, "%v: $(CTD) is not defined: %v", p, name).debug()
    trace(ctx)
  } else if ctd := d.value; isTrivial(ctd) {
    erro(ctx, "%v: $(CTD) is trivial for %v", p, name).debug()
    trace(ctx)
  } else if file = stat(ctx, name, stat_dir{ctd.string(ctx)}, stat_nonexist{true}); file == nil {
    erro(ctx, "%v: not a file: %v (%v)", p, name, ctd.string(ctx)).debug()
    trace(ctx)
  }
  return
}

func (p *project) _configuration(ctx Context) (f *File) {
  if f = p.tempFile(closure_with(ctx, p.scope, p.configure.scope), configuration_sm); f == nil {
    erro(ctx, "%v: no file configuration.sm", p).debug()
    trace(ctx)
  }
  return
}

type cacher struct { generalOpts }

func (opts *cacher) cache(ctx Context, patts, paths []Value) {
  for _, m := range map_files(ctx, patts, paths) {
    if t := m.pattern; false { note(at(ctx,t), "%v %v → %v", t, ts(t), ts(m)).debug(2) }
  }
}

func project_entry(c Context, s string, a ...bool) entry { return _project(c).resolveEntry(c, s, a...) }
func project_resolve(c Context, s string) Object { return _project(c).resolve(c, s) }

func (p *project) resolveDef(ctx Context, name string) (res *def) {
  if o := p.resolve(ctx, name); o != nil { res, _ = o.(*def) }
  return
}

func (p *project) resolve(ctx Context, name string) (obj Object) {
  if _, obj = p.find(name); obj != nil {
    return
  }

  if p.ext.Plugin != nil {
    if sym, e := p.ext.Lookup(name); e == nil && sym != nil {
      erro(ctx, "TODO: convert ext symbol: %v: %T", name, sym).debug()
      trace(ctx)
    }
  }

  for _, base := range p.bases {
    if obj = base.resolve(ctx, name); obj != nil {
      return
    }
  }

  if p.configure != nil && p.configure != p {
    return p.configure.resolve(ctx, name)
  }
  return
}

func (p *project) resolveEntries(ctx Context, name any, _b ...bool) (entries []entry) {
  entries = append(entries, p.unmap_entries(ctx, name)...)

  if p.configure != nil && isConfigure(ctx) {
    if t := p.configure.resolveEntries(ctx, name, true); t != nil {
      entries = append(entries, t...)
    }
  }

  var alwaysResolveBases bool
  if n := len(_b); n > 0 { alwaysResolveBases = _b[n-1] }

  if alwaysResolveBases || entries == nil {
    for _, base := range p.bases {
      if t := base.resolveEntries(ctx, name, alwaysResolveBases); t != nil {
        entries = append(entries, t...)
        break
      }
    }
  }

  if true {
    /* FAST */
  } else if entries == nil { /* SLOW */
    for _, use := range p.use.list {
      if t := use.project.resolveEntries(ctx, name, alwaysResolveBases); t != nil {
        entries = append(entries, t...)
        break
      }
    }
  }
  return
}

func (p *project) resolveEntry(c Context, name any, a ...bool) (_ entry) {
  var entries = p.resolveEntries(c, name, a...)
  if n := len(entries) ; 0 < n {
    if 1 < n {
      erro(c, "%d entries: %v", n, name).debug()
      trace(c)
    }
    return entries[0]
  }
  return
}

func (p *project) resolvePatterns(ctx Context, v Value, s string) (res []*stemmed) {
  var t1, t2 time.Time

  defer func(t0 time.Time) {
    var t = time.Now()
    if d := t.Sub(t0); d > 1*time.Second {
      var ( d1 = t1.Sub(t0) ; d2 = t2.Sub(t1) ; d3 = t.Sub(t2) ; n int )
      var a = auto_get(ctx, "@")
      for sc := _stemmed_context(ctx); sc != nil; n += 1 {
        if c := inner(sc); c != nil { sc = _stemmed_context(c) } else { break }
      }

      var pos = _position(ctx)
      prompt(ctx, "%v: slow: %v: %v, %v %v %v", pos, p, d, d1, d2, d3)
      prompt(ctx, "%v: slow: %v: %v: %v %v, %d nests", pos, p, a, v, p.patterns, n).debug(4)

      for _, pat := range p.patterns {
        var pt = pat.target
        var pa = pat.arged
        var full, r, stems = pt.match(ctx, s)
        var m = _path(ctx, r)
        prompt(ctx, "%v: slow: %v%v: %v: %v %v %v, %v ; %v", pos, pt, pa, s, full, r, stems, m)
      }
      warnstack(ctx, 3).debug(6)
    }
  } (time.Now())

  if res, t1, t2 = p.resolvePatterns123(ctx, v, s); false && len(res) > 0 {
    for _, t := range res {
      if f, _ := toFile(t.target); f != nil {
        f.position = t.Position()
      } else if f = p.file(ctx, s); f != nil {
        f.position = t.Position()
        t.target = f
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
      prompt(ctx, "%v: slow: %v %v", _position(ctx), val, d).debug()
    }
  } (time.Now())

ForPatterns:
  for _, pat := range p.patterns {
    if full, r, stems := pat.target.match(ctx, s); full {
      var m = _path(ctx, r)

      if true { for sc := _stemmed_context(ctx); sc != nil; { // pattern loop detection
        if s := sc.stem.target.string(ctx); s == m { continue ForPatterns }
        if c := inner(sc); c != nil { sc = _stemmed_context(c) } else { break }
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
          var ( p = _position(ctx) ; pt = pat.target )
          prompt(ctx, "%v: slow: %v, %v→%d; %v⇒%v+%v", p, pt, pa, len(av), d, d2, d3).debug()
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

func (p *project) entry(ctx Context, options []Value, target Value, prog *program) (entry entry) {
  var name string
  if name = target.string(ctx); name == "" {
    erro(at(ctx, target), "empty target name: %v", target).debug()
    trace(ctx)
  }

  var patterned = target.patterned(ctx)
  if !patterned {
    // NOTE: it should work too if not checking against files
    switch target.(type) {
    case *File, *path, *barefile, *percpat, *globpat, *regexpat:
    default:
      if f := p.file(ctx, name); f != nil {
        f.position = target.Position()
        target = f
      }
    }
  }

  defer func() {
    if entry != nil {
      entry.programs(append(entry.programs(), prog)...)
    }
  } ()

  var arged []Value // e.g. for pattern filtering
  switch t := target.(type) {
  case *group:
    erro(ctx, "group target not supported: %v", t).debug()
    trace(ctx)
  case *argumented:
    target, arged = t.Value, merge(t.args...)
  }

  var c, _ = p.entries.hit(cache{ctx}, target)
  if c == nil {
    erro(ctx, "no cache for target: %v", target).debug()
    trace(ctx)
  }

  if len(c.a) == 0 {
    var rule = &rule{ target:target, arged:arged }

    if patterned {
      p.patterns = append(p.patterns, rule)
    }

    entry = rule
    c.a = append(c.a, rule)
  } else if p, y := c.a[0].(*rule); y {
    entry = p
  } else {
    errostack(ctx, 3, "wrong cache: %v", c).debug()
    trace(ctx)
  }

  if entry != nil && p.defaultEntry == nil { p.defaultEntry = entry }
  return
}

func (p *project) family() (res []*project) {
  res = append(res, p)
  for _, base := range p.bases {
    res = append(res, base.family()...)
  }
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

func (p *project) hasLoaded(ctx Context, proj *project, traveUseLoop bool) (rp *project, res, isb bool) {
  var uni = _universe(ctx)
  if uni.checkLoadGraph || !uni.fastMode {
    rp, res, isb, _ = p.hasLoadedRecur(ctx, p, proj, 1, traveUseLoop)
  }
  return
}

func (p *project) hasLoadedRecur(ctx Context, top, proj *project, depth int, traveUseLoop bool) (rp *project, res, isb bool, err error) {
  if depth > 1 && top == p && true {
    err = fmt.Errorf("loop '%v' (depth=%d)", p.loopLoadPath(), depth)
    erro(at(ctx,p.position), "%v: %v", p, err).debug()
    trace(ctx)
  } else if depth > 128 {
    err = fmt.Errorf("exceeds maximum base depth (%d) (start=%v, target=%v)", depth, top, proj)
    erro(at(ctx,p.position), "%v: %v", p, err)
    erro(at(ctx,top.position), "start: %v", top)
    erro(at(ctx,proj.position), "target: %v", proj).debug()
    trace(ctx)
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
      erro(at(ctx,p.position), "%v: %v", p, err).debug()
      trace(ctx)
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

func (p *project) isUsing(usee *project) (res bool) {
  for _, use := range p.use.list {
    if res = use.project == usee; res { break  }
    if res = use.project.isUsing(usee); res { break }
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
  if p.opt.traveUseLoop { return }
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
