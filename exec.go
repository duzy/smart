//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
  "bufio"
  "bytes"
  "fmt"
  "io"
  "io/fs"
  "os"
  "os/exec"
  "path/filepath"
  "regexp"
  "reflect"
  "strconv"
  "strings"
  "sync"
  "sync/atomic"
  "time"
  "unicode"
)

// Note that it's is also used with Sscanf.
const (
  fmtExitStatus = "exit status %d"
  maxPromptStr = 48
  maxWorkers = 3
  maxRetries = 1
)

type exitstatus struct { code int }
func (e *exitstatus) Error() string { return fmt.Sprintf(fmtExitStatus, e.code) }

type knownerror struct {
  string // capture
  int // column
}

var (
  defaultShell = "bash"
  udots = []byte("…")

  workingMutex = new(sync.Mutex)
  working atomic.Value // number of working executions

  stdout = &stdWriter{std:os.Stdout}
  stderr = &stdWriter{std:os.Stderr}

  testCheckExecRecipe func(Context, string, Value)
  testCheckExecOutput func(Context, string, int)

  rxFatalErrorFileNotFound = regexp.MustCompile(`(.+?):(\d+):(\d+): fatal error: '(.+?)' file not found`)

  knownerrors = map[*regexp.Regexp]func(*execBuffer, int, []knownerror) { // `(?P<first>\d+)\.(\d+).(?P<second>\d+)`
    regexp.MustCompile(`exit status (\-?[0-9]+)`): func(p *execBuffer, l int, g []knownerror) {
      if g[1].string != "0" { p.scanned(diagError, l, g[0].int, g[0].string) }
    },

    regexp.MustCompile(`(.+?):(\d+):(\d+): error: (.+)(?: {2,}\n(.+))?`): func(p *execBuffer, l int, g []knownerror) {
      if g[5].string != "" {
        p.scanned(diagError, l, g[4].int, fmt.Sprintf("%s: %s", g[4].string, g[5].string))
      } else {
        p.scanned(diagError, l, g[4].int, fmt.Sprintf("%s", g[4].string))
      }
      if false && !p.reportIncludedFrom() { erro(at(p,p.lpos(l, g[4].int)), "…reported here").debug() }
    },
    regexp.MustCompile(`(.+?):(\d+):(\d+): warning: (.+)`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagWarn, l, g[0].int, g[4].string)
      p.scanned(diagWarn, l, g[0].int, "warning").position = p.pos(g[1].string, g[2].string, g[3].string)
      if false && !p.reportIncludedFrom() { erro(at(p,p.lpos(l, g[4].int)), "…reported here").debug() }
    },

    regexp.MustCompile(`In file included from (.+?):(\d+):`): func(p *execBuffer, l int, g []knownerror) {
      p.includedFrom.pos1 = p.pos(g[1].string, g[2].string, "1")
      p.includedFrom.pos2 = p.lpos(l, g[2].int)
    },
    regexp.MustCompile(`In file included from (.+?):(\d+):(\d+):`): func(p *execBuffer, l int, g []knownerror) {
      p.includedFrom.pos1 = p.pos(g[1].string, g[2].string, g[3].string)
      p.includedFrom.pos2 = p.lpos(l, g[3].int)
    },

    regexp.MustCompile(`ar: (.+?): No such file or directory`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[0].int, fmt.Sprintf("'%v' file not found", filepath.Base(g[1].string)))
    },
    regexp.MustCompile(`ar: no archive members specified`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[0].int, g[0].string)
    },

    regexp.MustCompile(`bash: line ([0-9]+?): (.+?): No such file or directory`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[2].int, fmt.Sprintf("no such command '%v'", g[2].string))
    },
    regexp.MustCompile(`(.+?): (.+?):( command)? not found`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[2].int, fmt.Sprintf("%s: command not found", g[2].string))
    },

    regexp.MustCompile(`((?:clang|(?:[^\.]+\.)?l?ld|wasm)(?:\-.+?)?): error: (.+)`): func(p *execBuffer, l int, g []knownerror) {
      var vs string // TODO: fetch version string
      p.scanned(diagError, l, g[2].int, fmt.Sprintf("%s%s: %s", g[1].string, vs, g[2].string))
    },
    regexp.MustCompile(`((?:clang|(?:[^\.]+\.)?l?ld|wasm)(?:\-.+?)?): warning: (.+)`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagWarn, l, g[2].int, fmt.Sprintf("%s: %s", g[1].string, g[2].string))
    },
    regexp.MustCompile(`((?:clang|(?:[^\.]+\.)?l?ld|wasm)(?:\-.+?)?): could not parse object file (.+?): '(.+)', using libLTO version '(.+?)' file '(.+?)' for architecture (.+)`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[2].int, g[2].string)
    },
    regexp.MustCompile(`((?:clang|(?:[^\.]+\.)?l?ld|wasm)(?:\-.+?)?): library not found for (.+)`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[2].int, g[0].string)
    },

    regexp.MustCompile(`(.+?): Too many positional arguments specified!`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[0].int, fmt.Sprintf("%s: too many positional arguments", g[1].string))
    },
    regexp.MustCompile(`  +"([^"]+?)", referenced from:`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[1].int, fmt.Sprintf("undefined reference '%s'", g[1].string))
    },
    regexp.MustCompile(`undef: *(.+)`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[1].int, fmt.Sprintf("undefined reference '%s'", g[1].string))
    },

    regexp.MustCompile(`ignoring (duplicate|nonexistent) directory "(.*?)"`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagInfo, l, g[2].int, fmt.Sprintf(`ignoring nonexistent directory "%v"`, g[2].string))
    },

    regexp.MustCompile(`^(.+?\.proto): File not found\.`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[0].int, g[0].string)
    },
    regexp.MustCompile(`^(.+?\.proto):(\d+):(\d+): Import "(.+?)" was not found or had errors.`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[0].int, fmt.Sprintf(`Import "%v" not found or errors`, g[4].string))
      p.scanned(diagError, l, g[0].int, "error").position = p.pos(g[1].string, g[2].string, g[3].string)
      if false && !p.reportIncludedFrom() { erro(at(p,p.lpos(l, g[4].int)), "…reported here").debug() }
    },
    regexp.MustCompile(`^(.+?\.proto):(\d+):(\d+): "(.+?)" is not defined.`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[0].int, fmt.Sprintf(`"%v" is not defined`, g[4].string))
      p.scanned(diagError, l, g[0].int, "error").position = p.pos(g[1].string, g[2].string, g[3].string)
      if false && !p.reportIncludedFrom() { erro(at(p,p.lpos(l, g[4].int)), "…reported here").debug() }
    },
    rxFatalErrorFileNotFound: func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[0].int, fmt.Sprintf(`"%v" file not found`, g[4].string))
      p.scanned(diagError, l, g[0].int, "error").position = p.pos(g[1].string, g[2].string, g[3].string)
      if false && !p.reportIncludedFrom() { erro(at(p,p.lpos(l, g[4].int)), "…reported here").debug() }
    },

    // NOTE: python standard errors
    regexp.MustCompile(`^\s*File "(.+?)", line (\d+), in (.+)`): func(p *execBuffer, l int, g []knownerror) {
      var c = g[3].int
      p.scanned(diagError, l, c, fmt.Sprintf(`in %v`, g[3].string))
      p.scanned(diagError, l, c, "error").position = p.pos(g[1].string, g[2].string, "")
    },
    regexp.MustCompile(`ModuleNotFoundError: No module named '(.*?)'`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[1].int, fmt.Sprintf(`no python module named "%v"`, g[1].string))
    },
    regexp.MustCompile(`FileNotFoundError: \[Errno (\d+)\] No such file or directory: '(.*?)'`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[2].int, fmt.Sprintf(`no such file or directory "%v"`, g[2].string))
    },

    regexp.MustCompile(`Cannot connect to the Docker daemon at (.*?)\. Is the docker daemon running\?`): func(p *execBuffer, l int, g []knownerror) {
      var pos = p.lpos(l, g[0].int)
      if e := p.startDockerDaemon(pos, at(p, pos), p.container, g[1].string); e != nil {
        p.scanned(diagError, l, g[0].int, fmt.Sprintf("start container failed: %v", e))
      }
    },
    regexp.MustCompile(`Error response from daemon: (Container (.+?) is not running)`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[0].int, g[1].string)
    },
    regexp.MustCompile(`Error.*: No such container: (.*)`): func(p *execBuffer, l int, g []knownerror) {
      if name := g[1].string; p.skips(name) {
        p.scanned(diagError, l, g[0].int, fmt.Sprintf("container not running: %v", name))
      } else {
        p.containerToRun = name
      }
    },
    regexp.MustCompile(`Error.*: (network (.*) not found)\.`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[0].int, g[1].string)
    },
    regexp.MustCompile(`the input device is not a TTY`): func(p *execBuffer, l int, g []knownerror) {
      p.scanned(diagError, l, g[0].int, g[0].string)
    },
  }
)

func init() { working.Store(0) }

func trimPromptString(str string) string { return trimPromptStringX(str, maxPromptStr) }
func trimPromptStringX(str string, x int) (s string) {
  var segs = strings.Split(str, pathSep)
  if len(segs) <= 1 {
    if n, m := len(str), maxPromptStr; n > m {
      s = "…" + str[n-m:]
    } else {
      s = str
    }
    return
  }

  var i, n int
  for i = len(segs)-1; i >= 0; i -= 1 {
    n += len(segs[i]) + 1
    if n > x {
      var j = i - 1
      if j < 0 { j = i }
      segs[j] = "…"
      s = filepath.Join(segs[j:]...)
      return
    }
  }

  s = str
  return
}

type stdWriter struct {
  sync.Mutex
  std io.Writer
  suffixDots bool
}

func (w *stdWriter) Write(p []byte) (n int, err error) {
  w.Lock(); defer w.Unlock()
  if w.suffixDots {
    if !bytes.HasPrefix(p, udots) {
      w.std.Write([]byte("\n"))
    }
    w.suffixDots = false
  }
  if n, err = w.std.Write(p); bytes.HasSuffix(p, udots) {
    w.suffixDots = true
  }
  return
}

type execLog struct {
  sync.Mutex
  writer *bufio.Writer
  filename string
  lines int
}

func (p *execLog) Write(b []byte) (n int, err error) {
  p.Lock(); defer p.Unlock()
  p.lines += bytes.Count(b, []byte("\n"))
  n, err = p.writer.Write(b)
  return
}

func (p *execLog) createWriter(file *os.File, dir, cmd string) {
  p.writer = bufio.NewWriter(file)
  fmt.Fprintf(p, "-*- mode: compilation; default-directory: \"%s\" -*-\n", dir)
  fmt.Fprintf(p, "Compilation started at %v\n\n", time.Now())
  fmt.Fprintf(p, "%s\n", cmd)
}

type execDiag struct {
  position Position
  dt diagType
  msg string
  num int
}

type execBuffer struct {
  *execContext

  Tie  io.Writer
  Buf *bytes.Buffer
  line bytes.Buffer // works done line by line

  wrote uint64

  forLine Value

  includedFrom struct { pos1, pos2 Position }
}
func (p *execBuffer) Write(b []byte) (n int, err error) {
  var expandForLine = p.forLine != nil && !isTrivial(p.forLine)

  // Diagnostics assured only when expanding forLine.
  if expandForLine { defer trace(p) }

  var l int
  if p.Buf != nil {
    if n, err = p.Buf.Write(b); err != nil {
      return
    }
  }
  if p.log != nil {
    l = p.log.lines // get lines before writing new bytes
    if _, err = p.log.Write(b); err != nil {
      return
    }
  }
  if p.Tie != nil {
    if n, err = p.Tie.Write(b); err != nil {
      return
    }
  }
  if err == nil && n == 0 {
    // Returns the number of bytes to avoid "short write" errors.
    // The real bytes written is discarded.
    n = len(b)
  }

  p.wrote += uint64(n)

  scanLine := p.scanErrors() || expandForLine ||
    (p.scanStdout && p == &p.Stdout) ||
    (p.scanStderr && p == &p.Stderr) ||
    testCheckExecOutput != nil
  if !scanLine { return }

  for slice := b[:]; len(slice) > 0; {
    var i = bytes.Index(slice, []byte("\n"))
    if i == -1 {
      p.line.Write(slice)
      slice = nil
    } else {
      p.line.Write(slice[:i+1])
      slice = slice[i+1:]
      l += 1

      var line = p.line.Bytes()

      if testCheckExecOutput != nil {
        var ctx Context = p.execContext
        if p.log != nil && !p.logPos.IsValid() {
          var pos Position
          pos.Filename, pos.Line = p.log.filename, l
          ctx = at(p.execContext, pos)
        }
        testCheckExecOutput(ctx, string(line), l)
      }

      if expandForLine {
        c := p.execContext
        c.line.s, c.lino.int64 = string(line), int64(l)
        v := p.forLine.expand(final{p.Context})
        if true { if t := p.forLine; t.String() == "${.test.for $1,$2}" {
          note(p, "%v → %v{%v} ; %v", t, typeof(v), v, auto_get(p, "1")).debug()
        } else if v != nil {
          if s := strings.TrimSpace(string(line)); true ||
            strings.HasPrefix(s, "test one\n") ||
            strings.HasPrefix(s, "test two\n") || (
            strings.HasPrefix(s, "ld: library '") && strings.HasSuffix(s, "' not found")) {
            note(p, "%v: %s → %v{%v} ; %v", p.forLine, s, typeof(v), v, auto_get(p.Context, "1")).debug()
          }
        }}
      }

      if p.scanErrors() {
        for rx, f := range knownerrors {
          for _, submatches := range rx.FindAllSubmatch(line, -1) { // range [][][]byte
            var column int
            var v []knownerror // known captures
            for i, capture := range submatches { // range [][]byte
              if true  { if i > 0 { column += bytes.Index(line[column:], capture) }}
              v = append(v, knownerror{string(capture), column + 1})
              if false { if i > 0 { column += len(capture) }}
            }
            f(p/*, rx*/, l, v)
          }
        }
      }

      p.line.Reset()
    }
  }
  return
}

func (p *execBuffer) startDockerDaemon(pos Position, ctx Context, container *project, sock string) (err error) {
  var c = exec.Command("dockerd") //c.Stdout, c.Stderr = stdout, stderr
  if err = c.Run(); err != nil {
    if p.report { erro(at(ctx,pos), "dokcer daemon not running (at %s)", sock).debug() }
  } else {
    // TODO: start docker daemon
  }
  return
}
func (p *execBuffer) filepath(s string) string {
  if p._workdir != "" && !filepath.IsAbs(s) { s = filepath.Join(p._workdir, s) }
  return s
}
func (p *execBuffer)  pos(s1, s2, s3 string) Position { return convPosition(p.filepath(s1), s2, s3) }
func (p *execBuffer) lpos(line, column int) Position {
  var pos = p.position
  if p.log != nil {
    pos.Filename, pos.Line, pos.Column = p.log.filename, line, column
  }
  return pos
}
func (p *execBuffer) reportIncludedFrom() (res bool) {
  if p.includedFrom.pos1.IsValid() && p.includedFrom.pos2.IsValid() {
    erro(at(p,p.includedFrom.pos1), "… included from here")
    erro(at(p,p.includedFrom.pos2), "… reported here").debug(4)
    p.includedFrom.pos1 = Position{}
    p.includedFrom.pos2 = Position{}
    res = true
  }
  return
}
func (p *execBuffer) scanned(dt diagType, line, column int, msg string) (res *execDiag) {
  for _, rec := range p.scannedDiags {
    if rec.msg == "error" || rec.msg == "warning" { continue }
    if rec.msg == msg { rec.num += 1 ; return rec }
  }

  res = &execDiag{ p.lpos(line, column), dt, msg, 1 }
  p.scannedDiags = append(p.scannedDiags, res)
  return
}

type execResult struct {
  valbase
  vals []Value
  Stdout execBuffer
  Stderr execBuffer
  Status int // aka. exit code
}
func (p *execResult) expand(_ Context) Value { return p }
func (p *execResult) cmp(ctx Context, v Value) (res cmpres) {
  if a, ok := v.(*execResult); ok {
    assert(ok, "value is not execResult")
    if p.Status == a.Status { res = cmpEqual }
  }
  return
}
func (p *execResult) true(ctx Context) (res bool) {
  res = p.Status == 0 && p.Stderr.Buf != nil && p.Stderr.Buf.Len() == 0 /* && p.Stdout.Buf.Len() > 0 */
  return
}
func (p *execResult) int(ctx Context) (i int64, _ error) { return int64(p.Status), nil }
func (p *execResult) float(ctx Context) (f float64, _ error) { return float64(p.Status), nil }
func (p *execResult) string(ctx Context) (s string) {
  if p.Stdout.Buf != nil { return p.Stdout.Buf.String() }
  if p.Stderr.Buf != nil { return p.Stderr.Buf.String() }
  return strconv.Itoa(p.Status)
}
func (p *execResult) String() string {
  var s bytes.Buffer
  fmt.Fprintf(&s, "exec{status=%d", p.Status)
  if p.Stdout.Buf != nil { fmt.Fprintf(&s, " stdout=%v", p.Stdout.Buf) }
  if p.Stderr.Buf != nil { fmt.Fprintf(&s, " stderr=%v", p.Stderr.Buf) }
  fmt.Fprintf(&s, "}")
  return s.String()
}

type execOpts struct {
  generalOpts
  logFileName *fullname "log"
  forRecipe Value `forrecipe,forrecipes,for-recipe,for-recipes`
  forStdout Value `forstdout,for-stdout,for-out`
  forStderr Value `forstderr,for-stderr,for-err`
  correction  bool `correction,correct-flags,correct-command-flags`
  warnCorrection bool `correction-warning,warn-correction`
  deprecated  bool `dump,deprecate`
  dropFailed  bool `drop,drop-fail,drop-failure,fail-drop,remove-on-fail`
  infos       bool `scan-infos`
  silentErrs  bool `silent,silent-errors` // silent errors
  zeroErrs    bool `no-errors,zero-errors` // require zero error scaned from STDERR
  tieStdout   bool `tie-out,tie-stdout` // tied with log
  tieStderr   bool `tie-err,tie-stderr` // tied with log
  bufStdout   bool `stdout,save-stdout`
  bufStderr   bool `stderr,save-stderr`
  stdin       bool `stdin,in,input`
  stamp       bool `stamp,stamp-file`
  noStamp     bool `nostamp,no-stamp,no-stamp-file`
  waitRes     bool `wait,waitresult,wait-result,waitres,wait-res` // wait for execution finished
  report      bool `report,report-stamp,verbose-stamp`
  retStdout   bool `return-stdout,result-stdout,stdout`
  retStderr   bool `return-stderr,result-stderr,stderr`
  retStatus   bool `return-status,result-status,status` // may work with zero-errors
  scanStdout  bool `scan-stdout,scan-out`
  scanStderr  bool `scan-stderr,scan-err`
  parallel    bool `par,parallel,no-order`
  path        bool `path`
  noCD        bool `nocd`
  prompt      bool `prompt,msg`
  promptSrc   bool `prompt-src,prompt-source,verbose-source`
  promStr     string `cmd`
  _workdir    string `cd,change-dir,wd,workdir,work-dir,work-directory`
  tie         string `tie` // all, both, stdout, stderr, out, err
}

type execContext struct {
  Context

  execOpts
  execResult

  sources []*raw
  current int

  line strlit
  lino decimal

  log *execLog
  logPos Position

  target as
  targetName string

  retried map[string]bool // work with containerToRun
  containerToRun string   // work with retried
  container *project

  num int
  x  *executor
  sh *exec.Cmd
  args []string

  start time.Time
  scannedDiags []*execDiag
}

func (p *execContext) cast(t reflect.Type) Context { return implcast(p,t) }
// func (p *execContext) String() string { return p.Context.String() }
func (p *execContext) Position() Position {
  if p.current < 0 { return _program(p).position }
  return p.sources[p.current].position
}

func (p *execContext) scanErrors() bool {
  return (p.debug > 0 || p.report) && p.silentErrs == false
}

func (p *execContext) runContainerAndRetry() (err error) {
  if p.container == nil {
    erro(p.Context, "no container").debug()
    return
  } else if maxRetries < p.num {
    fmt.Fprintf(p.sh.Stderr, "\n---- Retried %d times\n", p.num)
    return
  }

  var (
    name = p.containerToRun
    sh = p.sh
  )

  fmt.Fprintf(sh.Stderr, "\n---- Run container '%s'\n", name)
  if entries := p.container.resolveEntries(p.Context, "run", false); entries != nil {
    for _, run := range entries {
      run.execute(p.Context, nil)
    }
  } else {
    erro(p.Context, "%s⇒run undefined", p.container).debug()
    return
  }

  fmt.Fprintf(sh.Stderr, "\n---- Retry the command in %s:", name)
  if false {
    fmt.Fprintf(sh.Stderr, "\n%s:\n    %v", sh.Path, strings.Join(sh.Args, "\n    "))
    fmt.Fprintf(sh.Stderr, "\n\naka:\n    %s", sh)
    fmt.Fprintf(sh.Stderr, "\n----\n")
  } else {
    fmt.Fprintf(sh.Stderr, "\n")
  }

  p.sh = exec.Command(sh.Path, sh.Args[1:]...) // must ignore Args[0]
  p.sh.Stdout, p.sh.Stderr, p.sh.Stdin = sh.Stdout, sh.Stderr, sh.Stdin
  p.sh.Dir, p.sh.Env = sh.Dir, sh.Env
  if err = p.run(); err != nil {
    fmt.Fprintf(sh.Stderr, "\n---- Retry failed: %s\n", err)
  }
  return
}

// DEPRECATED
func (p *execContext) ensureContainerRunning(containerName string) (err error) {
  var (
    stdoutR, stdoutW = io.Pipe()
    stderrR, stderrW = io.Pipe()
    enviro = os.Environ()
    cmd = exec.Command(`docker`, `ps`,
      `--filter`, `status=running`,
      //`--filter`, fmt.Sprintf(`ancestor=%s`, image),
      `--filter`, fmt.Sprintf(`name=%s`, containerName),
      `--format`, `{{.ID}}\t{{.Image}}\t{{.Names}}`,
    )
    foundID, foundImage string
  )
  cmd.Stdout, cmd.Stderr, cmd.Env = stdoutW, stderrW, enviro
  defer stdoutW.Close()
  defer stderrW.Close()

  go func(r io.Reader) {
    var buf = bufio.NewReader(r)
    for {
      s, e := buf.ReadString('\n')
      if e != nil {
        break
      }
      if fields := strings.Split(s, "\t"); len(fields) == 3 {
        if names := strings.Split(fields[2], ","); len(names) > 0 {
          foundID, foundImage = fields[0], fields[1]
          if foundImage == "" { /* FIXME: unused */ }
        }
      }
    }
  } (stdoutR)

  go func(r io.Reader) {
    var buf = bufio.NewReader(r)
    for {
      s, e := buf.ReadString('\n')
      if e != nil {
        break
      }
      prompt(p.Context, "%s", s)
    }
  } (stderrR)

  if err = cmd.Run(); err == nil && foundID == "" {
    if entries := p.container.resolveEntries(p.Context, "run", false); entries != nil {
      for _, run := range entries {
        run.execute(p.Context, nil)
      }
    } else {
      erro(p.Context, "%s⇒run undefined", p.container).debug()
      return
    }
  } else if err != nil {
    erro(at(p.Context,p.container.position), "%v", err).debug()
  }
  return
}

func (p *execContext) skips(tag string) bool {
  if p.retried == nil { p.retried = make(map[string]bool) }
  var a, b = p.retried[tag]
  return a && b
}

func (p *execContext) run() (err error) {
  if p.containerToRun != "" {
    p.retried[p.containerToRun] = true // mark it to skip next time
    err = p.runContainerAndRetry()
    p.containerToRun = ""
    return
  }

  var pc = _execution(p.Context)

  pc.Add(1)
  p.num += 1

  var run = func(c *exec.Cmd) {
    defer pc.Done()

    if err = c.Run(); err == nil {
      err = p.check()
    } else if ee, ok := err.(*exec.ExitError); !ok {
      erro(p.Context, "exec failed: %v", err).debug()
      return
    } else if p.Status = ee.ExitCode(); p.Status == 0 {
      err = p.check() // success!
    }
  }

  if true { run(p.sh) } else { go run(p.sh) }
  return
}

func (p *execContext) check() (err error) {
  if (!p.silentErrs || p.debug>0) && (len(p.scannedDiags) > 0 || p.Status != 0 || err != nil) {
    if p.silentErrs || p.retStatus {
      err = nil
    } else if p.Status != 0 {
      err = &exitstatus{ p.Status } // set or convert error
    }

    var en, wn, in int
    for _, rec := range p.scannedDiags {
      switch rec.dt {
      case diagError: en += rec.num
      case diagWarn:  wn += rec.num
      case diagInfo:  in += rec.num
      }
    }

    var ctx = p.Context
    if en > 0 || p.Status != 0 || err != nil {
      prompt(ctx, "exec: failure (status=%d; err=%v); target=%s\n", p.Status, err, p.targetName)
    } else if wn > 0 {
      prompt(ctx, "%v: %d warnings\n", p.targetName, wn)
    }

    if p.scanErrors() { for i, rec := range p.scannedDiags {
      if !p.infos && rec.dt == diagInfo { continue }
      if !p.logPos.IsValid() { p.logPos = rec.position }
      if i == 0 && !rec.position.Same(&rec.position) {
        diag(at(ctx,rec.position), rec.dt, rec.msg)//.debug()
      }
      if rec.num > 1 {
        diag(at(ctx,rec.position), rec.dt, `%s (%d)`, rec.msg, rec.num)//.debug()
      } else {
        diag(at(ctx,rec.position), rec.dt, rec.msg)//.debug()
      }
      if n := (en+wn+in)-(i+1); i == 8 && 0 < n {
        diag(at(ctx,rec.position), rec.dt, "%d more...", n)//.debug()
        break
      }
    }}

    var pos = _position(ctx)
    if !p.logPos.IsValid() && p.log != nil {
      p.logPos.Filename = p.log.filename
      p.logPos.Line = p.Stderr.log.lines + 1
    } else {
      p.logPos = pos
    }

    var diffLogPos = !p.logPos.SameLine(&pos)
    var str, _, _ = entryIndicator(ctx, _entry(ctx))
    if (!p.retStatus && p.Status != 0) || en > 0 {
      if p.dropFailed {
        if e := os.RemoveAll(p.targetName); e != nil {
          warn(ctx, "remove: %v", e).debug()
        }
      }

      if diffLogPos && en > 0 { erro(at(ctx,p.logPos), "%v: %d known errors", str, en) }
      erro(p, "%v: exit status %d", str, p.Status).debug()
    } else if wn > 0 {
      if diffLogPos { warn(at(ctx,p.logPos), "%v: %d known warnings", str, wn) }
      warn(p, "%v: exit status %d", str, p.Status)
      warn(ctx, "%v: %d known warnings", str, wn)
      warnstack(ctx, 3).debug()
    } else if in > 0 && p.infos {
      if diffLogPos { info(at(ctx,p.logPos), "%v: %d known messages", str, in) }
      info(p, "%v: exit status %d", str, p.Status)
      info(ctx, "%v: %d known messages", str, in)
      infostack(ctx, 8).debug()
    }

    if p.retStatus {
      if p.zeroErrs && en == 0 && err == nil {
        p.vals = append(p.vals, makeDecimal(p.logPos, int64(p.Status)))
      } else {
        p.vals = append(p.vals, makeNone(p.logPos))
      }
    } else if p.Status != 0 || err != nil {
      // break
    }
  }
  return
}

func (ctx *execContext) exec(cmd, opt string) {
  var (
    pc = _execution(ctx)
    env, sep = pc.env(ctx)
    envstr string
    logFile *os.File
  )

  for i, s := range env[sep:] {
    if i > 0 { envstr += " && " }
    if k := strings.Index(s, "="); k > 0 {
      envstr += fmt.Sprintf(`%s%s`, s[:k+1], strconv.Quote(s[k+2:]))
    }
  }

  defer func() {
    if ctx.log != nil && ctx.log.writer != nil { ctx.log.writer.Flush() }
    if logFile != nil { logFile.Close() }
    if ctx.log != nil && ctx.log.filename != "" && ctx.Stdout.wrote == 0 && ctx.Stderr.wrote == 0 {
      if false { os.Remove(ctx.log.filename) }
    }

    // var c = pc.caller()
    // if !ctx.silentErrs && c != nil && err != nil { c.calleeError(err) }

    ctx.Stdout.execContext = nil
    ctx.Stderr.execContext = nil
    ctx.container = nil
    ctx.sh = nil
    ctx.x = nil

    // Stamp the target file.
    if !ctx.stamp || isConfigure(ctx) {
      // no stamp for target files
    } else if files, e := ctx.target.stamp(ctx); e != nil {
      if pe, ok := e.(*fs.PathError); ok {
        prompt(ctx, "%v: target not found, stamp \"%v\"\n", pe.Path, ctx.target)
        erro(ctx, `"%v" not found`, ctx.target).debug()
        trace(ctx)
      } else {
        prompt(ctx, "%v: target not found, \"%v\"\n", pe.Path, e)
      }
      if ctx.logFileName != nil && !ctx.logPos.IsValid() {
        prompt(ctx, "%v:1: see logs for \"%s\"\n", ctx.logFileName.string(ctx), ctx.target)
      }
      erro(ctx, `stamp "%v" failed`, ctx.target).debug()
      trace(ctx)
    } else if !ctx.prompt && ctx.report {
      reportFileUpdates(ctx, files)
    }

    if ctx.prompt {
      var ps = ctx.promStr
      if ps += trimPromptString(ctx.targetName); pc.caller() == nil {
        ps += " …… ok"
      }
      if ps != "" {
        var s = time.Now().Sub(ctx.start).String()
        if n := ctx.Stdout.wrote; n > 0 { s += fmt.Sprintf(", stdout=%d bytes", n) }
        if n := ctx.Stderr.wrote; n > 0 { s += fmt.Sprintf(", stderr=%d bytes", n) }
        if t := pc.dirt; t != "" { s += "; " + t }
        prompt(ctx, "%s (exec %s)\n", ps, s)
      }
    }
  } ()

  ctx.Stdout.forLine = ctx.forStdout
  ctx.Stderr.forLine = ctx.forStderr
  if ctx.forStdout != nil || ctx.forStderr != nil {
    ac := &automatic{ Context:ctx.Context, defs:make(auto_defs) }
    ac.args(ac.Context, []Value{&ctx.line, &ctx.lino})
    if d, y := ac.defs["1"]; y {
      ac.Lock()
      ac.defs["_"] = d // alias
      ac.Unlock()
    } else {
      erro(ctx, "wrong args: %v", ac.defs).debug()
      trace(ctx)
    }
    ctx.Context = ac
  }

  if ctx.bufStdout || ctx.retStdout { ctx.Stdout.Buf = new(bytes.Buffer) }
  if ctx.bufStderr || ctx.retStderr { ctx.Stderr.Buf = new(bytes.Buffer) }
  if ctx.tieStdout { ctx.Stdout.Tie = stdout }
  if ctx.tieStderr { ctx.Stderr.Tie = stderr }
  if ctx.logFileName != nil { ctx.log = &execLog{ filename: ctx.logFileName.string(ctx) } }
  if ctx.log == nil || ctx.log.filename == "" {
    // no log required
  } else if err := os.MkdirAll(filepath.Dir(ctx.log.filename), os.FileMode(0755)); err != nil {
    erro(ctx, "%v", err).debug()
    trace(ctx)
  } else if logFile, err = os.Create(ctx.log.filename); err != nil {
    erro(ctx, "%v", err).debug()
    trace(ctx)
  } else {
    cmdline := joinRaws("\n", ctx.sources...)
    ctx.log.createWriter(logFile, ctx._workdir, cmdline)
  }
  ctx.Stdout.execContext = ctx
  ctx.Stderr.execContext = ctx
  ctx.start = time.Now()

  var _ctx = ctx.Context
  var u = _universe(ctx)
  for i, src := range ctx.sources {
    ctx.Context = at(_ctx, src)
    ctx.current = i

    if a := "@"; strings.HasPrefix(src.s, a) {
      src.s = strings.TrimPrefix(src.s, a)
    } else if ctx.promptSrc && !ctx.prompt {
      var s string = src.s
      s = strings.Replace(s, "\n", "\\n", -1)
      s = strings.Replace(s, "\\\\n", "\\\n", -1)
      prompt(ctx, "%s\n", s)//.debug()
    }

    if src.s = strings.TrimSpace(src.s); src.s == "" { continue }

    if cmd == "docker" && len(envstr) > 0 {
      src.s = fmt.Sprintf("%s && %s", envstr, src.s)
    }

    if u.noExec { continue }

    ctx.sh = exec.Command(cmd, ctx.args...)
    ctx.sh.Dir = ctx._workdir // always set command work directory
    ctx.sh.Env = env
    ctx.sh.Stdout = &ctx.Stdout
    ctx.sh.Stderr = &ctx.Stderr
    if ctx.stdin {
      ctx.sh.Stdin = os.Stdin
      ctx.sh.Args = append(ctx.sh.Args, "-ti")
    }
    if   opt != "" { ctx.sh.Args = append(ctx.sh.Args, opt) }
    if src.s != "" { ctx.sh.Args = append(ctx.sh.Args, src.s) }

    var err = ctx.run()
    if ctx.Status != 0 || err != nil { break }
  }
}

type executor struct {
  cmd, opt string
  contained bool
}
func (p *executor) evaluate(ctx Context, args ...Value) (result Value) {
  var uni = _universe(ctx)
  if uni.traceExecutor {
    var t = auto_get(ctx, "@")
    defer un(l_trace(l_exec, fmt.Sprintf("executor(%v)", ts(t))))
  }

  defer trace(ctx)

  var (
    pos = _position(ctx)
    exe = &execContext{Context:ctx, current:-1, x:p}
    cmd = p.cmd
  )
  defer func() {
    exe.Stdout.execContext = nil
    exe.Stderr.execContext = nil
  }()

  exe.scanStderr = true
  exe.execResult.position = pos
  args = parseOpts(ctx, &exe.execOpts, args...)

  if exe.deprecated {
    erro(ctx, "deprecated args: -v (-to), -w (-te), -a (-se), -d (-t)").debug()
    trace(ctx)
  } else if d := exe.debug; false && d>0 { defer func() {
    note(ctx, "%v: %v (%v)", _entry(ctx), exe.target.Value, result).debug(d)
  }()}

  if !exe.prompt { exe.prompt = exe.promStr != "" }

  switch exe.tie {
  case "stdout", "out" : exe.tieStdout = true
  case "stderr", "err" : exe.tieStderr = true
  case "all"   , "both": exe.tieStdout, exe.tieStderr = true, true
  }

  var pc = _execution(ctx)
  var program = _program(pc)
  if exe.target.Value = getTargetValue(ctx); program == nil {
    erro(ctx, "needs program context to exec: %v", ctx).debug(16)
    trace(ctx)
  } else if exe.stamp && exe.target.patterned(ctx) {
    errostack(ctx, 5, "target is pattern: %v", exe.target).debug(64)
    trace(ctx)
  } else if _, ok := exe.target.Value.(flag); ok {
    // no stamp required for Flags
  } else if _, ok = toFile(exe.target.Value); !ok {
    // no stamp required for non-file targets
  } else if exe.targetName, _ = exe.target.fullnameOrFinal(ctx); isConfigure(exe) {
    // does nothing
  } else if exe.waitRes {
    // good to work without (stamp) or (wait) with the -wait flag
  } else if m := program.getModifiers(ctx, "wait"); len(m) > 0 {
    // should be good to work
  } else if t := exe.target.Value; !(exe.stamp || exe.noStamp || exe.silentErrs) {
    warn(ctx, "add -stamp or -nostamp to (shell); target=%v(%v)", typeof(t), t).debug()
  }

  if (exe.retStdout && exe.retStatus) || (exe.retStderr && exe.retStatus) {
    erro(ctx, "cannot have both status and stdout|stderr at the same time (try -so or -se)").debug()
    trace(ctx)
  }

  for i, v := range args { var s string
    if p.contained && i == 0 {
      if s = v.string(ctx); s == "shell" { cmd = defaultShell }
    } else if s = strings.TrimSpace(v.string(ctx)); s != "" {
      exe.args = append(exe.args, s)
    }
  }

  if p.contained {
    if program.project.name == dotContainer {
      exe.container = program.project
    } else if _, sym := program.project.find(dotContainer); sym != nil {
      exe.container = sym.(*project)
    }

    if exe.container == nil {
      erro(ctx, "container unavailable (in %s)", program.project.name).debug()
      trace(ctx)
    }

    var stringify = func(name string) (str string) {
      var ctx = closure_with(ctx, exe.container.scope)
      if obj := exe.container.resolve(ctx, name); obj != nil {
        if d, _ := obj.(*def); d != nil {
          if v := d.invoke(ctx, nil, nil); v != nil {
            if str = v.string(ctx); str == "-" {
              /*if v, err = def.DiscloseValue(exe.container); err == nil && v != nil {
                  if str, err = v.string(ctx); str == "" { str = "-" }
                  prompt(ctx, "%v: %v (%v)\n", name, str, def)
                }*/
            }
          }
        }
      }
      return
    }

    var containerName string
    if containerName = stringify("container"); containerName == "" {
      erro(ctx, ".container.name undefined").debug()
      trace(ctx)
    }

    var containerImage string
    if containerImage = stringify("image"); containerImage == "" {
      erro(ctx, ".container.image undefined").debug()
      trace(ctx)
    }

    if uni.verbose {
      prompt(ctx, "%v: container=%v, image=%v\n", exe.container, containerName, containerImage)
    }

    exe.args = append(exe.args, "exec", containerName, cmd)
    cmd = "docker"
  }

  // FIXME: work directory conflicts sometimes even the 'sh.Dir' is set to cwd.
  // Because the current work directory is not thread safe.
  if exe._workdir == "" {
    if exe._workdir = program.workdir(ctx); exe._workdir == "" {
      erro(ctx, "CWD is empty").debug()
      trace(ctx)
    }
  }

  if exe.path { var s string
    if s = filepath.Dir(exe.targetName); s != "" && s != "." && s != "/" {
      if err := os.MkdirAll(s, os.FileMode(0755)); err != nil {
        erro(at(ctx,exe.target), "make path '%s' for target failed: %v", s, err).debug()
        trace(ctx)
      }
    }
  }

  var ac *automatic
  var a1 *strlit
  var a2 *decimal
  if exe.forRecipe != nil {
    a1, a2 = &strlit{}, &decimal{}
    ac = &automatic{ Context:ctx, defs:make(auto_defs) }
    ac.args(ac.Context, []Value{a1, a2})
  }

  var source string
  var recipePos Position
  for i, recipe := range program.recipes {
    if recipe = recipe.expand(ctx); !fixEvokedFullnames && exe.fullname {
      // NOTE: do a second expand for fullname because delegate to file skipped
      //       fullname expansion (FIXME: fixEvokedFullnames)
      recipe = recipe.expand(expandFullFile{ctx})
    }

    if !recipePos.IsValid() { recipePos = recipe.Position() }

    if s := strings.TrimRightFunc(recipe.string(ctx), unicode.IsSpace); s == "" {
      source += "\n" // an empty line
      continue
    } else {
      if false { if y, _ := regexp.MatchString(" <[^>]+>PIC ", s); y {
        // builtin_foreach_d = true
        x := recipe.string(ctx)
        // builtin_foreach_d = false

        note(at(ctx, recipe), "%v: %v", typeof(recipe), recipe)
        note(at(ctx, recipe), "%v: %v", typeof(recipe), x).debug()
      }}

      // Escape '$$' sequences.
      s = strings.Replace(s, "$$", "$", -1)

      // Duplicate all %
      //s = strings.Replace(s, "%", "%%", -1)

      source += s
    }

    if strings.HasSuffix(source, "\\") {
      source += "\n" // append the line feed
      if i < len(program.recipes) { continue }
    }

    // Remove tabs in line breakings.
    source = strings.Replace(source, "\\\n\t", "\\\n", -1)

    if exe.correction {
      source = correctCommandFlags(ctx, source, exe.warnCorrection)
    }

    if exe.forRecipe != nil {
      a1.position, a1.s = recipePos, source
      a2.position, a2.int64 = recipePos, int64(len(exe.sources)+1)
      ac.Context = at(ctx, recipePos)
      if val := exe.forRecipe.expand(final{ac}); false && val != nil {
        for i := 0; indeterminate(ac, val); i += 1 {
          if i < max_evoke { val = val.expand(final{ac}) } else {
            erro(at(ctx, exe.forRecipe), "%v → %v", exe.forRecipe, val).debug()
            trace(ctx)
            break
          }
        }
        if false { note(at(ctx,exe.forRecipe), "%v → %v", exe.forRecipe, val).debug() }
      }
    }

    if testCheckExecRecipe != nil { testCheckExecRecipe(ctx, source, recipe) }

    exe.sources = append(exe.sources, &raw{valbase{recipePos}, source})
    recipePos, source = Position{}, ""
  }

  if len(program.recipes) > 0 && len(exe.sources) == 0 {
    erro(ctx, "empty recipes: %v", program.recipes).debug()
    trace(ctx)
  }

  if true {
    exe.exec(cmd, p.opt)
  } else {
    go exe.exec(cmd, p.opt)
  }

  // Add stdout result
  if exe.retStdout {
    var s string
    if exe.Stdout.Buf != nil { s = exe.Stdout.Buf.String() }
    exe.vals = append(exe.vals, makeStrlit(pos, s))
  }

  // Add stderr result
  if exe.retStderr {
    var s string
    if exe.Stderr.Buf != nil { s = exe.Stderr.Buf.String() }
    exe.vals = append(exe.vals, makeStrlit(pos, s))
  }

  // The execution is performed asynchronously, the result can't be fetched immediately.
  // Caller should do a t.wait(...) or exe.wait() before using the result.
  if exe.vals == nil {
    result = &exe.execResult
  } else {
    result = ease(ctx, exe.vals)
  }
  return
}

var execCommandClang = regexp.MustCompile(`^@?(?:/(?:[^/]*/)+)?(clang(?:\+{2})?)$`)
var execExistFlagPath = map[*regexp.Regexp][]*regexp.Regexp{
  execCommandClang: []*regexp.Regexp{
    regexp.MustCompile(`^-([IL]|include|(?:i(?:(?:framework)?with)|-)?sysroot|(?:cxx-|stdlib(?:\+\+)?)?isystem(?:-after)?|iframework)=?([[:alnum:]_\-/]+)?$`),
  },
}

func correctCommandFlags(ctx Context, source string, w bool) string {
  var flags []string
  var fields = strings.Fields(source)
  if len(fields) > 0 { flags = fields[:1] }

forFields:
  for i := 1; i < len(fields); i += 1 {
    var field = fields[i]

    for rx, rxs := range execExistFlagPath {
      if rx.MatchString(fields[0]) { for _, rx := range rxs {
        var m = rx.FindStringSubmatch(field)
        if len(m) == 0 { continue }

        var f bool
        var s = m[2]
        if s == "" {
          if i += 1; i == len(fields) { break forFields }
          s, f = fields[i], true
        }

        if _, e := os.Stat(s); e != nil {
          if w { warn(ctx, "ignoring nonexistent path: %v", s).debug() }
          continue forFields // skip nonexistent path flags
        } else if f {
          flags = append(flags, field)
          field = s
        }
      }}
    }

    flags = append(flags, field)
  }
  return strings.Join(flags, " ")
}
