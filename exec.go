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
)

type exitstatus struct { code int }
func (e *exitstatus) Error() string { return fmt.Sprintf(fmtExitStatus, e.code) }

func rx(s string) (res *regexp.Regexp) {
  res = regexp.MustCompile(s)
  knownerrors = append(knownerrors, res)
  return
}

var (
  defaultShell = "bash"

  strErrorPreprocess = `#error (.+)`

  knownerrors []*regexp.Regexp

  rxNotTTYDevice = rx(`the input device is not a TTY`)
  rxNoContainer  = rx(`Error.*: No such container: (.*)`)
  rxNoNetwork    = rx(`Error.*: network (.*) not found\.`)
  rxDockerDaemonNotRunning = rx(`Cannot connect to the Docker daemon at (.*?)\. Is the docker daemon running\?`)
  rxContainerNotRunning    = rx(`Error response from daemon: Container (.*?) is not running`)
  rxCompilationError       = rx(`(.+?):(\d+):(\d+): error: (.+)(?: {2,}\n(.+))?`)
  rxCompilationWarning     = rx(`(.+?):(\d+):(\d+): warning: (.+)`)
  rxIncludedFrom2          = rx(`In file included from (.+?):(\d+):`)
  rxIncludedFrom3          = rx(`In file included from (.+?):(\d+):(\d+):`)
  rxProtoImportNotFound    = rx(`^(.+?\.proto):(\d+):(\d+): Import "(.+?)" was not found or had errors.`)
  rxProtoNameNotDefined    = rx(`^(.+?\.proto):(\d+):(\d+): "(.+?)" is not defined.`)
  rxProtoFileNotFound      = rx(`^(.+?\.proto): File not found\.`)
  rxFatalErrorFileNotFound = rx(`(.+?):(\d+):(\d+): fatal error: '(.+?)' file not found`)
  rxArNoSuchFile       = rx(`ar: (.+?): No such file or directory`)
  rxArNoArchiveMembers = rx(`ar: no archive members specified`)
  rxBashNoSuchFile     = rx(`bash: line ([0-9]+?): (.+?): No such file or directory`)
  rxCmdError           = rx(`((?:clang|(?:[^\.]+\.)?l?ld|wasm)(?:\-.+?)?): error: (.+)`)
  rxCmdWarning         = rx(`((?:clang|(?:[^\.]+\.)?l?ld|wasm)(?:\-.+?)?): warning: (.+)`)
  rxCouldnotParseObj   = rx(`((?:clang|(?:[^\.]+\.)?l?ld|wasm)(?:\-.+?)?): could not parse object file (.+?): '(.+)', using libLTO version '(.+?)' file '(.+?)' for architecture (.+)`)
  rxLdLibNotFound      = rx(`((?:clang|(?:[^\.]+\.)?l?ld|wasm)(?:\-.+?)?): library not found for (.+)`)
  rxTooManyPosArgs     = rx(`(.+?): Too many positional arguments specified!`)
  rxUndefinedReference = rx(`  +"([^"]+?)", referenced from:`)
  rxUndefReference     = rx(`undef: *(.+)`)
  rxShellCmdNotFound   = rx(`(.+?): (.+?):( command)? not found`)
  rxIgnoringDirectory  = rx(`ignoring (duplicate|nonexistent) directory "(.*?)"`)
  rxExitStatus         = rx(`exit status (\-?[0-9]+)`)

  // NOTE: python standard errors
  rxPyErrorTrace          = rx(`^\s*File "(.+?)", line (\d+), in (.+)`)
  rxPyModuleNotFoundError = rx(`ModuleNotFoundError: No module named '(.*?)'`)
  rxPyFileNotFoundError   = rx(`FileNotFoundError: \[Errno (\d+)\] No such file or directory: '(.*?)'`)

  workingMutex = new(sync.Mutex)
  working atomic.Value // number of working executions

  stdout = &stdWriter{std:os.Stdout}
  stderr = &stdWriter{std:os.Stderr}
  udots = []byte("…")

  testCheckExecRecipe func(Context, string, Value)
  testCheckExecOutput func(Context, string, int)
)

const (
  maxRetries = 1
  maxWorkers = 3
)

func init() { working.Store(0) }

func trimPromptString(str string) string { return trimPromptStringX(str, maxPromptStr) }
func trimPromptStringX(str string, x int) (s string) {
  var segs = strings.Split(str, PathSep)
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
  std io.Writer
  mux sync.Mutex
  suffixDots bool
}

func (w *stdWriter) Write(p []byte) (n int, err error) {
  w.mux.Lock(); defer w.mux.Unlock()
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

type ExecLog struct {
  filename string
  writer *bufio.Writer
  wrimux sync.Mutex
  lines int
}

func (p *ExecLog) Write(b []byte) (n int, err error) {
  p.wrimux.Lock(); defer p.wrimux.Unlock()
  p.lines += bytes.Count(b, []byte("\n"))
  n, err = p.writer.Write(b)
  return
}

func (p *ExecLog) createWriter(file *os.File, dir, cmd string) {
  p.writer = bufio.NewWriter(file)
  fmt.Fprintf(p, "-*- mode: compilation; default-directory: \"%s\" -*-\n", dir)
  fmt.Fprintf(p, "Compilation started at %v\n\n", time.Now())
  fmt.Fprintf(p, "%s\n", cmd)
}

type knownMatchCap struct {
  string
  col int
}
type knownMatch struct {
  rx *regexp.Regexp
  l int
  v [][]knownMatchCap // groups of captures
}

type scannedExecDiag struct {
  position Position
  dt diagType
  msg string
  num int
}

type ExecBuffer struct {
  *execContext

  Buf *bytes.Buffer
  Tie  io.Writer
  line bytes.Buffer
  wrote uint64

  includedFrom struct { pos1, pos2 Position }
}
func (p *ExecBuffer) Write(b []byte) (n int, err error) {
  if p.wrote == 0 { p.onFirstWrote() }

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

  if !((p.scanStdout && p == &p.Stdout) || (p.scanStderr && p == &p.Stderr)) {
    return
  }

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

        if false && ctx.dia().error() {
          noted(p.execContext, "%s", line).debug(1)
        }
      }

      for _, rx := range knownerrors {
        if rx == nil { continue }
        if all := rx.FindAllSubmatch(line, -1); all != nil {
          // var result = make(map[string]string) // `(?P<first>\d+)\.(\d+).(?P<second>\d+)`
          // for i, name := range rx.SubexpNames() {
          //   if i != 0 && name != "" {
          //     result[name] = match[i]
          //   }
          // }
          var ( a [][]knownMatchCap; c int )
          for _, m := range all { // [][][]byte
            var v []knownMatchCap // captures
            for i, cap := range m {
              if true  { if i > 0 { c = bytes.Index(line, cap) }}
              v = append(v, knownMatchCap{string(cap),c})
              if false { if i > 0 { c += len(cap) }}
            }
            a = append(a, v)
          }
          p.scan(p.position, &knownMatch{ rx, l, a })
        }
      }

      p.line.Reset()
    }
  }
  return
}

func (p *ExecBuffer) startDockerDaemon(pos Position, ctx Context, container *Project, sock string) (err error) {
  var c = exec.Command("dockerd") //c.Stdout, c.Stderr = stdout, stderr
  if err = c.Run(); err != nil {
    if p.report { erro(at(ctx,pos), "dokcer daemon not running (at %s)", sock).debug(1) }
  } else {
    // TODO: start docker daemon
  }
  return
}
func (p *ExecBuffer) filepath(s string) string {
  if p.workDir != "" && !filepath.IsAbs(s) { s = filepath.Join(p.workDir, s) }
  return s
}
func (p *ExecBuffer) pos(s1, s2, s3 string) Position { return convPosition(p.filepath(s1), s2, s3) }
func (p *ExecBuffer) scan(pos Position, m *knownMatch) {
  var ctx = p.Context
  if p == nil {
    erro(at(ctx,pos), "nil exec buffer").debug(1)
    return
  }

  if !p.scanErrors() { return }

  var (
    lpos Position = pos
    container *Project = p.container
    reportIncludedFrom = func() (res bool) {
      if p.includedFrom.pos1.IsValid() && p.includedFrom.pos2.IsValid() {
        erro(at(ctx,p.includedFrom.pos1), "… included from here")
        erro(at(ctx,p.includedFrom.pos2), "… reported here").debug(4)
        p.includedFrom.pos1 = Position{}
        p.includedFrom.pos2 = Position{}
        res = true
      }
      return
    }
    scanned = func(dt diagType, msg string) (res *scannedExecDiag) {
      for _, rec := range p.scannedDiags {
        if rec.msg == "error" || rec.msg == "warning" { continue }
        if rec.msg == msg { rec.num += 1 ; return rec }
      }

      res = &scannedExecDiag{ lpos, dt, msg, 1 }
      p.scannedDiags = append(p.scannedDiags, res)
      return
    }
  )

  if p.log != nil { lpos.Filename = p.log.filename }
  if     m != nil { lpos.Line, lpos.Column = m.l, 0 }
  for _, v := range m.v { // captures
    if len(v) > 1 { lpos.Column = v[1].col }
    switch m.rx {
    case rxExitStatus:
      if s := v[1].string; s != "0" /*&& p.report*/ {
        // FIXME: the 'exit status' report is not working
        scanned(diagError, fmt.Sprintf("abnormal exist status %s", s))
      }
    case rxDockerDaemonNotRunning:
      if err := p.startDockerDaemon(lpos, ctx, container, v[1].string); err != nil {
        scanned(diagError, fmt.Sprintf("start container failed: %v", err))
      }
    case rxContainerNotRunning:
      scanned(diagError, fmt.Sprintf("Container not running (%v)", v[1].string))
    case rxNotTTYDevice:
      scanned(diagError, fmt.Sprintf(`needs TTY (input device)`))
    case rxNoContainer:
      if name := v[1].string; p.skips(name) {
        scanned(diagError, fmt.Sprintf("container not running: %v", name))
      } else {
        p.containerToRun = name
      }
    case rxNoNetwork:
      scanned(diagError, fmt.Sprintf("Network not found (%v)", v[1].string))
    case rxIncludedFrom2:
      lpos.Column = v[2].col + 1
      p.includedFrom.pos1 = p.pos(v[1].string, v[2].string, "1")
      p.includedFrom.pos2 = lpos
    case rxIncludedFrom3:
      lpos.Column = v[3].col + 1
      p.includedFrom.pos1 = p.pos(v[1].string, v[2].string, v[3].string)
      p.includedFrom.pos2 = lpos
    case rxCompilationError:
      // p.errorPos = p.pos(v[1].string, v[2].string, v[3].string)
      lpos.Column = v[4].col + 1
      if s := v[5].string; s != "" {
        scanned(diagError, fmt.Sprintf("%s: %s", v[4].string, s))
      } else {
        scanned(diagError, fmt.Sprintf("%s", v[4].string))
      }
      if false && !reportIncludedFrom() { erro(at(ctx,lpos), "…reported here").debug(1) }
    case rxCompilationWarning:
      lpos.Column = v[4].col + 1
      scanned(diagWarn, fmt.Sprintf("%s", v[4].string))
      scanned(diagWarn, "warning").position = p.pos(v[1].string, v[2].string, v[3].string)
      if false && !reportIncludedFrom() { warn(at(ctx,lpos), "…reported here").debug(1) }
    case rxProtoFileNotFound:
      lpos.Column = v[1].col
      scanned(diagError, fmt.Sprintf(`"%v" file not found`, v[1].string))
    case rxProtoImportNotFound:
      lpos.Column = v[4].col
      scanned(diagError, fmt.Sprintf(`Import "%v" not found or errors`, v[4].string))
      scanned(diagError, "error").position = p.pos(v[1].string, v[2].string, v[3].string)
      if false && !reportIncludedFrom() { erro(at(ctx,lpos), "…reported here").debug(1) }
    case rxProtoNameNotDefined:
      lpos.Column = v[4].col
      scanned(diagError, fmt.Sprintf(`"%v" is not defined`, v[4].string))
      scanned(diagError, "error").position = p.pos(v[1].string, v[2].string, v[3].string)
      if false && !reportIncludedFrom() { erro(at(ctx,lpos), "…reported here").debug(1) }
    case rxFatalErrorFileNotFound:
      lpos.Column = v[4].col
      scanned(diagError, fmt.Sprintf(`"%v" file not found`, v[4].string))
      scanned(diagError, "error").position = p.pos(v[1].string, v[2].string, v[3].string)
      if false && !reportIncludedFrom() { erro(at(ctx,lpos), "…reported here").debug(1) }
    case rxArNoSuchFile:
      var s = v[1].string
      scanned(diagError, fmt.Sprintf("'%v' file not found", filepath.Base(s)))
    case rxArNoArchiveMembers:
      if false {
        var obj = closureResolveObject(ctx, "objects")
        erro(at(ctx,lpos), "%s", v[0].string)
        erro(at(ctx,lpos), "%s", obj)
        if !isNull(obj) {
          if val := obj.expand(ctx.closure().pc(), plain); !isNull(val) {
            erro(at(ctx,lpos), "%s -> %v", obj.name(ctx), val)
          }
        }
        erro(ctx, "%v", ctx).debug(16)
      } else {
        scanned(diagError, v[0].string)
      }
    case rxBashNoSuchFile:
      var s = fmt.Sprintf("no such command '%v'", v[2].string)
      lpos.Column = v[2].col + 1
      scanned(diagError, s)
    case rxCmdError:
      var cs, vs string; cs = v[1].string
      lpos.Column = v[2].col + 1
      scanned(diagError, fmt.Sprintf("%s%s: %s", cs, vs, v[2].string))
    case rxCmdWarning:
      lpos.Column = v[2].col + 1
      scanned(diagWarn, fmt.Sprintf("%s: %s", v[1].string, v[2].string))
    case rxCouldnotParseObj:
      lpos.Column = v[2].col
      scanned(diagError, v[2].string)
    case rxLdLibNotFound:
      lpos.Column = v[2].col + 1
      scanned(diagError, v[0].string)
    case rxTooManyPosArgs:
      scanned(diagError, fmt.Sprintf("%s: too many positional arguments", v[1].string))
    case rxUndefinedReference:
      scanned(diagError, fmt.Sprintf("Undefined reference '%s'", v[1].string))
    case rxUndefReference:
      scanned(diagError, fmt.Sprintf("Undefined reference '%s'", v[1].string))
    case rxShellCmdNotFound:
      lpos.Column = v[2].col
      scanned(diagError, fmt.Sprintf("%s: command not found", v[2].string))
    case rxIgnoringDirectory:
      var dir = v[2].string;  lpos.Column = v[2].col + 1
      scanned(diagInfo, fmt.Sprintf(`ignoring nonexistent directory "%v"`, dir))
    case rxPyErrorTrace:
      lpos.Column = v[3].col
      scanned(diagError, fmt.Sprintf(`in %v`, v[3].string))
      scanned(diagError, "error").position = p.pos(v[1].string, v[2].string, "")
    case rxPyModuleNotFoundError:
      var name = v[1].string;  lpos.Column = v[1].col + 1
      scanned(diagError, fmt.Sprintf(`no python module named "%v"`, name))
    case rxPyFileNotFoundError:
      var name = v[2].string;  lpos.Column = v[2].col + 1
      scanned(diagError, fmt.Sprintf(`no such file or directory "%v"`, name))
    }
  }
  return
}

type execResult struct {
  valbase
  vals []Value
  Stdout ExecBuffer
  Stderr ExecBuffer
  Status int // aka. exit code
}
func (p *execResult) expand(_ Context, _ facet) Value { return p }
func (p *execResult) cmp(ctx Context, v Value) (res cmpres) {
  if a, ok := v.(*execResult); ok {
    assert(ok, "value is not execResult")
    if p.Status == a.Status { res = cmpEqual }
  }
  return
}
func (_ *execResult) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *execResult) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *execResult) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "cache unsupported").debug(32)
    return
}
func (p *execResult) true(ctx Context) (res bool) {
  res = p.Status == 0 && p.Stderr.Buf != nil && p.Stderr.Buf.Len() == 0 /* && p.Stdout.Buf.Len() > 0 */
  return
}
func (p *execResult) int(ctx Context) (i int64, _ error) { return int64(p.Status), nil }
func (p *execResult) float(ctx Context) (f float64, _ error) { return float64(p.Status), nil }
func (p *execResult) string(ctx Context) (s string) {
  if p.Stdout.Buf != nil { s = p.Stdout.Buf.String() }
  return
}
func (p *execResult) String() string {
  var s bytes.Buffer
  fmt.Fprintf(&s, "(execResult status=%d", p.Status)
  if p.Stdout.Buf != nil { fmt.Fprintf(&s, " stdout=%v", p.Stdout.Buf) }
  if p.Stderr.Buf != nil { fmt.Fprintf(&s, " stderr=%v", p.Stderr.Buf) }
  fmt.Fprintf(&s, ")")
  return s.String()
}

type execOpts struct {
  generalOpts
  logFileName *fullnameOpt "l,log"
  forRecipe Value `forrecipe,forrecipes,for-recipe,for-recipes`
  // checkRecipe bool `checkrecipe,checkrecipes,check-recipe,check-recipes`
  correction  bool `correction,correct-flags,correct-command-flags`
  warnCorrection bool `correction-warning,warn-correction`
  deprecated  bool `dump,deprecate`
  dropFailed  bool `df,drop,drop-fail,drop-failure,fail-drop,remove-on-fail`
  infos       bool `sci,scan-infos`
  silentErrs  bool `silent,silent-errors` // silent errors
  zeroErrs    bool `ze,zero-errors` // require zero error scaned from STDERR
  tieStdout   bool `to,tie-out,tie-stdout` // tied with log
  tieStderr   bool `te,tie-err,tie-stderr` // tied with log
  bufStdout   bool `o,stdout;bo,buffer-stdout;so,save-stdout`
  bufStderr   bool `e,stderr;be,buffer-stderr;se,save-stderr`
  stdin       bool `i,stdin;in,input`
  stamp       bool `stamp,stamp-file`
  noStamp     bool `nostamp,no-stamp,no-stamp-file`
  waitRes     bool `wr,wait,waitres,wait-res,waitresult,wait-result` // wait for execution finished
  report      bool `r,rs,report,report-stamp;vs,verbose-stamp`
  retStdout   bool `ro,return-stdout,result-stdout,stdout`
  retStderr   bool `re,return-stderr,result-stderr,stderr`
  retStatus   bool `rs,return-status,result-status,status` // may work with zero-errors
  scanStdout  bool `so,scan-stdout,scan-out`
  scanStderr  bool `se,scan-stderr,scan-err`
  parallel    bool `par,parallel,no-order`
  path        bool `p,path`
  noCD        bool `n,nocd`
  prompt      bool `pm,prompt;m,msg`
  promptSrc   bool `ps,prompt-src,prompt-source;vs,verbose-source`
  promStr     string `c,cmd;m,msg`
  workDir     string `cd,change-dir,wd,workdir,work-dir,work-directory`
  tie         string `t,tie` // all, both, stdout, stderr, out, err
}

type execContext struct {
  Context

  execOpts
  execResult

  sources []*raw
  current int

  log *ExecLog
  logPos Position

  target as
  targetName string

  retried map[string]bool // work with containerToRun
  containerToRun string   // work with retried
  container *Project

  printEnteringOnFirstWrote bool

  num int
  x  *executor
  sh *exec.Cmd
  args []string

  start time.Time
  scannedDiags []*scannedExecDiag
}

func (p *execContext) String() string { return p.Context.String() }
func (p *execContext) Position() Position {
  if p.current < 0 { return p.program().position }
  return p.sources[p.current].position
}
func (p *execContext) onFirstWrote() {
  if p.printEnteringOnFirstWrote {
    promptEnteringDirectory(p.Context)

    // Call diagFlush to ensure promptEnteringDirectory works immediately
    if errs := p.Context.dia().flush(); errs > 0 {
      warn(p.Context, "exec: encountered %d errors", errs).debug(1)
    }
  }
}

func (p *execContext) scanErrors() bool {
  return (p.debug > 0 || p.report) && p.silentErrs == false
}

func (p *execContext) runContainerAndRetry() (err error) {
  if p.container == nil {
    erro(p.Context, "no container").debug(1)
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
    for _, run := range entries.all {
      if _, traves := run.execute(p.Context, nil); traves.has() {
        erro(p.Context, "%d travestates", len(traves)).debug(1)
        return
      } //else { p.t.group.Wait() }
    }
  } else {
    erro(p.Context, "%s⇒run undefined", p.container).debug(1)
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
      for _, run := range entries.all {
        if _, traves := run.execute(p.Context, nil); traves.has() {
          erro(p.Context, "%d travestates", len(traves)).debug(1)
          return
        } //else { p.t.group.Wait() }
      }
    } else {
      erro(p.Context, "%s⇒run undefined", p.container).debug(1)
      return
    }
  } else if err != nil {
    erro(at(p.Context,p.container.position), "%v", err).debug(1)
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

  var pc = p.Context.pc()

  pc.Add(1)
  p.num += 1

  var run = func(c *exec.Cmd) {
    defer pc.Done()

    if err = c.Run(); err == nil {
      err = p.check()
    } else if ee, ok := err.(*exec.ExitError); !ok {
      erro(p.Context, "exec failed: %v", err).debug(1)
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
        diag(at(ctx,rec.position), rec.dt, rec.msg)//.debug(1)
      }
      if rec.num > 1 {
        diag(at(ctx,rec.position), rec.dt, `%s (%d)`, rec.msg, rec.num)//.debug(1)
      } else {
        diag(at(ctx,rec.position), rec.dt, rec.msg)//.debug(1)
      }
      if n := (en+wn+in)-(i+1); i == 8 && 0 < n {
        diag(at(ctx,rec.position), rec.dt, "%d more...", n)//.debug(1)
        break
      }
    }}

    var pos = ctx.Position()
    if !p.logPos.IsValid() && p.log != nil {
      p.logPos.Filename = p.log.filename
      p.logPos.Line = p.Stderr.log.lines + 1
    } else {
      p.logPos = pos
    }

    var diffLogPos = !p.logPos.SameLine(&pos)
    var str, _, _ = entryIndicator(ctx, ctx.entry())
    if (!p.retStatus && p.Status != 0) || en > 0 {
      if p.dropFailed {
        if e := os.RemoveAll(p.targetName); e != nil {
          warn(ctx, "remove: %v", e).debug(1)
        }
      }

      if diffLogPos && en > 0 { erro(at(ctx,p.logPos), "%v: %d known errors", str, en) }
      erro(p, "%v: exit status %d", str, p.Status).debug(1)
    } else if wn > 0 {
      if diffLogPos { warn(at(ctx,p.logPos), "%v: %d known warnings", str, wn) }
      warn(p, "%v: exit status %d", str, p.Status)
      warn(ctx, "%v: %d known warnings", str, wn)
      warnstack(ctx, 3).debug(1)
    } else if in > 0 && p.infos {
      if diffLogPos { info(at(ctx,p.logPos), "%v: %d known messages", str, in) }
      info(p, "%v: exit status %d", str, p.Status)
      info(ctx, "%v: %d known messages", str, in)
      infostack(ctx, 8).debug(1)
    }

    if p.retStatus {
      if p.zeroErrs && en == 0 && err == nil {
        p.vals = append(p.vals, MakeInt(p.logPos, int64(p.Status)))
      } else {
        p.vals = append(p.vals, makeNone(p.logPos))
      }
    } else if p.Status != 0 || err != nil {
      // break
    }
  }
  return
}

func (ctx *execContext) exec(cmd, opt string, err error) {
  if ctx.dia().error() { return }

  defer dtrace(ctx, "exec")

  var (
    pc = ctx.pc()
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

    var caller = pc.caller()
    if !ctx.silentErrs && caller != nil && err != nil { caller.calleeError(err) }

    ctx.Stdout.execContext = nil
    ctx.Stderr.execContext = nil
    ctx.container = nil
    ctx.sh = nil
    ctx.x = nil

    // Stamp the target file.
    if !ctx.stamp || ctx.isConfigure() {
      // no stamp for target files
    } else if err != nil {
      var files, e = ctx.target.delete(ctx)
      if e != nil { erro(ctx, `%v: delete: %v`, ctx.target, e) }

      if ctx.log != nil {
        var s, l = ctx.log.filename, ctx.log.lines
        prompt(ctx, "%v:%d: %v: %v\n", s, l, ctx.target, err)
      } else {
        prompt(ctx, "%v: %v (deleted %d files)\n", ctx.target, err, len(files))
      }

      for _, file := range files {
        if s, fn := file.String(), file.fullname(); s == fn {
          erro(ctx, `%v: deleted`, s)
        } else {
          erro(ctx, `%v: deleted, %v`, s, fn)
        }
      }
      erro(ctx, "%v: %v", ctx.target, err).debug(1)
      return
    } else if files, e := ctx.target.stamp(ctx); e != nil {
      if pe, ok := e.(*fs.PathError); ok { err = fmt.Errorf(`"%v" not found`, ctx.target)
        prompt(ctx, "%v: target not found, stamp \"%v\"\n", pe.Path, ctx.target)
      } else {
        prompt(ctx, "%v: target not found, \"%v\"\n", pe.Path, e)
      }
      if ctx.logFileName != nil && !ctx.logPos.IsValid() {
        prompt(ctx, "%v:1: see logs for \"%s\"\n", ctx.logFileName.string(ctx), ctx.target)
      }
      erro(ctx, `stamp "%v" failed`, ctx.target).debug(1)
      return
    } else if !ctx.prompt && ctx.report {
      reportFileUpdates(ctx, files)
    }

    if err != nil && ctx.isConfigure() { err = nil }
    if err != nil {
      erro(ctx, "shell: %v", err).debug(1)
      return
    }

    if ctx.prompt { var ps = ctx.promStr
      if ps += trimPromptString(ctx.targetName); caller == nil {
        if ps += " …… "; err != nil { ps += err.Error() } else { ps += "ok" }
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

  if ctx.logFileName != nil { ctx.log = &ExecLog{ filename: ctx.logFileName.string(ctx) } }
  if ctx.bufStdout || ctx.retStdout { ctx.Stdout.Buf = new(bytes.Buffer) }
  if ctx.bufStderr || ctx.retStderr { ctx.Stderr.Buf = new(bytes.Buffer) }
  if ctx.tieStdout { ctx.Stdout.Tie = stdout }
  if ctx.tieStderr { ctx.Stderr.Tie = stderr }
  if ctx.log == nil || ctx.log.filename == "" {
    // no log required
  } else if err = os.MkdirAll(filepath.Dir(ctx.log.filename), os.FileMode(0755)); err != nil {
    erro(ctx, "%v", err).debug(1)
    return
  } else if logFile, err = os.Create(ctx.log.filename); err != nil {
    erro(ctx, "%v", err).debug(1)
    return
  } else {
    cmdline := joinRaws("\n", ctx.sources...)
    ctx.log.createWriter(logFile, ctx.workDir, cmdline)
  }
  ctx.Stdout.execContext = ctx
  ctx.Stderr.execContext = ctx
  ctx.start = time.Now()

  var _ctx = ctx.Context
  var uni = ctx.universe()
  for i, src := range ctx.sources {
    ctx.Context = at(_ctx, src.Position())
    ctx.current = i

    if a := "@"; strings.HasPrefix(src.s, a) {
      src.s = strings.TrimPrefix(src.s, a)
    } else if ctx.promptSrc && !ctx.prompt {
      var s string = src.s
      s = strings.Replace(s, "\n", "\\n", -1)
      s = strings.Replace(s, "\\\\n", "\\\n", -1)
      prompt(ctx, "%s\n", s)//.debug(1)
    }

    if src.s = strings.TrimSpace(src.s); src.s == "" { continue }

    if false && !ctx.noCD && ctx.workDir != "" {
      if strings.HasPrefix(src.s, "#") {
        src.s = fmt.Sprintf("cd '%s' %s", ctx.workDir, src.s)
      } else {
        // Insert a "\n" before the right paren ')' to ensure that
        // it's working with comments like "true #comment...".
        src.s = fmt.Sprintf("cd '%s' && (%s\n)", ctx.workDir, src.s)
      }
    }

    if cmd == "docker" && len(envstr) > 0 {
      src.s = fmt.Sprintf("%s && %s", envstr, src.s)
    }

    if uni.noExec { continue }

    if !ctx.silentErrs || ctx.prompt || ctx.promptSrc {
      ctx.printEnteringOnFirstWrote = true
    }

    ctx.sh = exec.Command(cmd, ctx.args...)
    ctx.sh.Dir = ctx.workDir // always set command work directory
    ctx.sh.Env = env
    ctx.sh.Stdout = &ctx.Stdout
    ctx.sh.Stderr = &ctx.Stderr
    if ctx.stdin {
      ctx.sh.Stdin = os.Stdin
      ctx.sh.Args = append(ctx.sh.Args, "-ti")
    }
    if opt != "" { ctx.sh.Args = append(ctx.sh.Args, opt) }
    if src.s != "" { ctx.sh.Args = append(ctx.sh.Args, src.s) }

    err = ctx.run()

    d := ctx.debug
    if d > 0 {
      entry := ctx.entry()
      prompt(ctx, "%v\n", ctx.sh)
      if _, s, y := ctx.target.fullname(ctx); y {
        prompt(ctx, "%v:1: %v: %v\n", s, entry, err)
      } else {
        noted(ctx, "%v: %v ; status=%v", entry, ctx.target, ctx.Status)
      }
      noted(ctx, "%v: status=%v", entry, ctx.Status).debug(d)

      uni.configuration.silent = true
    }

    if ctx.Status != 0 || err != nil { break }
  }
}

type executor struct {
  cmd, opt string
  contained bool
}
func (p *executor) evaluate(ctx Context, args ...Value) (result Value, err error) {
  var uni = ctx.universe()
  if uni.traceExecutor {
    var t = autoVal(ctx, "@")
    defer un(trace(t_exec, fmt.Sprintf("executor(%s %v)", typeof(t), t)))
  }

  defer dtrace(ctx, "executor")

  var (
    pos = ctx.Position()
    exe = &execContext{Context:ctx, current:-1, x:p}
    cmd = p.cmd
  )
  defer func() {
    exe.Stdout.execContext = nil
    exe.Stderr.execContext = nil
  }()

  exe.scanStderr = true
  exe.execResult.position = pos
  args = parseOpts(ctx, &exe.execOpts, plain, args...)

  if exe.deprecated {
    erro(ctx, "deprecated args: -v (-to), -w (-te), -a (-se), -d (-t)").debug(1)
    return
  } else if d := exe.debug; false && d>0 { defer func() {
    noted(ctx, "%v: %v (%v)", ctx.entry(), exe.target.Value, result).debug(d)
  }()}

  if !exe.prompt { exe.prompt = exe.promStr != "" }

  switch exe.tie {
  case "stdout", "out" : exe.tieStdout = true
  case "stderr", "err" : exe.tieStderr = true
  case "all"   , "both": exe.tieStdout, exe.tieStderr = true, true
  }

  var pc = ctx.pc()
  var program = pc.program()
  if exe.target.Value = getTargetValue(ctx); program == nil {
    erro(ctx, "needs program context to exec: %v", ctx).debug(16)
    return
  } else if exe.stamp && exe.target.patterned(ctx) {
    errostack(ctx, 5, "target is pattern: %v", exe.target).debug(64)
    return
  } else if _, ok := exe.target.Value.(flag); ok {
    // no stamp required for Flags
  } else if _, ok = toFile(exe.target.Value); !ok {
    // no stamp required for non-file targets
  } else if exe.targetName, _ = exe.target.fullnameOrStrval(ctx); exe.isConfigure() {
    // does nothing
  } else if exe.waitRes {
    // good to work without (stamp) or (wait) with the -wait flag
  } else if m := program.getModifiers(ctx, "wait"); len(m) > 0 {
    // should be good to work
  } else if t := exe.target.Value; !(exe.stamp || exe.noStamp || exe.silentErrs) {
    warn(ctx, "add -stamp or -nostamp to (shell); target=%v(%v)", typeof(t), t).debug(1)
  }

  if (exe.retStdout && exe.retStatus) || (exe.retStderr && exe.retStatus) {
    erro(ctx, "cannot have both status and stdout|stderr at the same time (try -so or -se)").debug(1)
    return
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
    } else if _, containerSym := program.project.scope.find(dotContainer); containerSym != nil {
      if pn, _ := containerSym.(*projectname); pn != nil {
        exe.container = pn.Project
      }
    }

    if exe.container == nil {
      erro(ctx, "container unavailable (in %s)", program.project.name).debug(1)
      return
    }

    var strval = func(name string) (str string) {
      var ctx = closureWith(ctx, exe.container.Scope())
      if obj := exe.container.resolve(ctx, name); obj != nil {
        if d, _ := obj.(*def); d != nil {
          if v := d.invoke(ctx, plain, nil, nil); v != nil {
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
    if containerName = strval("container"); containerName == "" {
      erro(ctx, ".container.name undefined").debug(1)
      return
    }

    var containerImage string
    if containerImage = strval("image"); containerImage == "" {
      erro(ctx, ".container.image undefined").debug(1)
      return
    }

    if uni.verbose {
      prompt(ctx, "%v: container=%v, image=%v\n", exe.container, containerName, containerImage)
    }

    exe.args = append(exe.args, "exec", containerName, cmd)
    cmd = "docker"
  }

  // FIXME: work directory conflicts sometimes even the 'sh.Dir' is set to cwd.
  // Because the current work directory is not thread safe.
  if exe.workDir != "" {
    // good
  } else if exe.workDir = program.workDir(ctx); exe.workDir == "" {
    erro(ctx, "CWD is empty").debug(1)
    return
  }

  if exe.path { var s string
    if s = filepath.Dir(exe.targetName); s != "" && s != "." && s != "/" {
      if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
        erro(of(ctx,exe.target), "make path '%s' for target failed: %v", s, err).debug(1)
        return
      }
    }
  }

  var w = strval
  if exe.fullname { w |= expandFullName }

  var ( ac *autoContext; a1 *String; a2 *Int )
  if exe.forRecipe != nil {
    a1, a2 = &String{}, &Int{}
    ac = &autoContext{ Context:ctx, defs:make(autoDefMap) }
    ac.args(ac.Context, nil, []Value{a1, a2})
  }

  var source string
  var recipePos Position
  for i, recipe := range program.recipes {
    if recipe = recipe.expand(ctx, w); !fixEvokedFullnames && exe.fullname {
      // NOTE: do a second expand for fullname because delegate to file
      //       skipped fullname expansion (FIXME: fixEvokedFullnames)
      recipe = recipe.expand(ctx, expandFullName)
    }

    if !recipePos.IsValid() { recipePos = recipe.Position() }

    if s := strings.TrimRightFunc(recipe.string(ctx), unicode.IsSpace); s == "" {
      source += "\n" // an empty line
      continue
    } else {
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
      val := exe.forRecipe.expand(ac, strval) // aka. xauto(...)
      for i := 0; val != nil && val.expandable(ac, strval); i += 1 {
        if i < max_evoke { val = val.expand(ac, strval) } else {
          erro(of(ctx, exe.forRecipe), "%v → %v", exe.forRecipe, val).debug(1)
          break
        }
      }
      if false { noted(of(ctx,exe.forRecipe), "%v → %v", exe.forRecipe, val).debug(1) }
    }

    if testCheckExecRecipe != nil { testCheckExecRecipe(ctx, source, recipe) }

    exe.sources = append(exe.sources, &raw{valbase{recipePos}, source})
    recipePos, source = Position{}, ""
  }

  if len(program.recipes) > 0 && len(exe.sources) == 0 {
    erro(ctx, "empty recipes: %v", program.recipes).debug(1)
    return;
  }

  if true {
    exe.exec(cmd, p.opt, err)
  } else {
    go exe.exec(cmd, p.opt, err)
  }

  // Add stdout result
  if exe.retStdout {
    var s string
    if exe.Stdout.Buf != nil { s = exe.Stdout.Buf.String() }
    exe.vals = append(exe.vals, MakeString(pos, s))
  }

  // Add stderr result
  if exe.retStderr {
    var s string
    if exe.Stderr.Buf != nil { s = exe.Stderr.Buf.String() }
    exe.vals = append(exe.vals, MakeString(pos, s))
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
    regexp.MustCompile(`^-([IL]|include|(?:cxx-|stdlib(?:\+\+)?)?isystem(?:-after)?)=?([[:alnum:]_\-/]+)?$`),
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
          if w { warn(ctx, "ignoring nonexistent path: %v", s).debug(1) }
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
