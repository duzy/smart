//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
  "bufio"
  "bytes"
  "extbit.io/smart/scanner"
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

  //rxCompilationDefaultDirectory = rx(`\-\*\- mode: compilation; default\-directory: "(.+?)" \-\*\-`)
  rxNotTTYDevice = rx(`the input device is not a TTY`)
  rxNoContainer = rx(`Error.*: No such container: (.*)`)
  rxNoNetwork = rx(`Error.*: network (.*) not found\.`)
  rxDockerDaemonNotRunning = rx(`Cannot connect to the Docker daemon at (.*?)\. Is the docker daemon running\?`)
  rxContainerNotRunning = rx(`Error response from daemon: Container (.*?) is not running`)
  rxCompilationError = rx(`(.+?):(\d+):(\d+): error: (.+)(?: {2,}\n(.+))?`)
  rxCompilationWarning = rx(`(.+?):(\d+):(\d+): warning: (.+)`)
  rxIncludedFrom2 = rx(`In file included from (.+?):(\d+):`)
  rxIncludedFrom3 = rx(`In file included from (.+?):(\d+):(\d+):`)
  rxProtoImportNotFound = rx(`^(.+?\.proto):(\d+):(\d+): Import "(.+?)" was not found or had errors.`)
  rxProtoNameNotDefined = rx(`^(.+?\.proto):(\d+):(\d+): "(.+?)" is not defined.`)
  rxProtoFileNotFound = rx(`^(.+?\.proto): File not found\.`)
  rxFatalErrorFileNotFound = rx(`(.+?):(\d+):(\d+): fatal error: '(.+?)' file not found`)
  rxArNoSuchFile = rx(`ar: (.+?): No such file or directory`)
  rxArNoArchiveMembers = rx(`ar: no archive members specified`)
  rxBashNoSuchFile = rx(`bash: line ([0-9]+?): (.+?): No such file or directory`)
  rxClangNoSuchFile = rx(`clang(?:-(.+?))?: error: no such file or directory: '(.+?)'`)
  //XXX: rxClangError = rx(`clang(?:-(.+?))?: error: (.+)(?: \(.+\))?`)
  rxCmdError = rx(`(ld\.lld|ld64\.lld|lld-link|wasm-ld|ld|clang)(?:-(.+?))?: error: (.+)`)
  rxCmdWarning = rx(`(ld\.lld|ld64\.lld|lld-link|wasm-ld|ld|clang): warning: (.+)`)
  rxLdLibNotFound = rx(`(ld\.lld|ld64\.lld|lld-link|wasm-ld|ld): library not found for (.+)`)
  rxCouldnotParseObj = rx(`(ld\.lld|ld64\.lld|lld-link|wasm-ld|ld): could not parse object file (.+?): '(.+)', using libLTO version '(.+?)' file '(.+?)' for architecture (.+)`)
  rxTooManyPosArgs = rx(`(.+?): Too many positional arguments specified!`)
  rxUndefinedReference = rx(`  +"(.+?)", referenced from:`)
  rxShellCmdNotFound = rx(`(.+?): (.+?):( command)? not found`)
  rxExitStatus = rx(`exit status (\-?[0-9]+)`)
  rxIgnoringNonExistentDirectory = rx(`ignoring nonexistent directory "(.*?)"`)
  rxIgnoringDuplicateDirectory = rx(`ignoring duplicate directory "(.*?)"`)

  workingMutex = new(sync.Mutex)
  working atomic.Value // number of working executions

  stdout = &stdWriter{ std:os.Stdout }
  stderr = &stdWriter{ std:os.Stderr }
  udots = []byte("…")
)

const (
  maxRetries = 1
  maxWorkers = 3
)

func init() {
  working.Store(0)
}

func checkForWork() (good bool, num int) {
  if false { workingMutex.Lock(); defer workingMutex.Unlock()}
  if num = working.Load().(int); num < maxWorkers {
    working.Store(num + 1)
    good = true
  }
  return
}

func waitForWork() (num int) {
  var good = false
  for {
    if good, num = checkForWork(); good { break }
    time.Sleep(50*time.Millisecond)
  }
  return
}

func releaseWork(num int) {
  if false { workingMutex.Lock(); defer workingMutex.Unlock() }
  working.Store(num - 1)
}

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
  position, lpos Position
  dt diagType
  msg string
  num int
}
type ExecBuffer struct {
  res *ExecResult
  log *ExecLog
  Buf *bytes.Buffer
  Tie  io.Writer
  line bytes.Buffer
  filters []string
  wrote uint64
  report bool
  scanKnownErrors bool
  errorPos Position
  errors []error
  defaultDirectory string
  includedFrom struct { pos1, pos2 Position }
}
func (p *ExecBuffer) filter(s string) { p.filters = append(p.filters, s) }
func (p *ExecBuffer) Write(b []byte) (n int, err error) {
  if p.wrote == 0 { p.res.onFirstWrote() }

  for _, s := range p.filters {
    if bytes.Equal(b, []byte(s)) { // string(b) == s
      return len(b), nil
    }
  }

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

  if !p.scanKnownErrors { return }

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
      for _, rx := range knownerrors {
        if rx == nil { continue }
        if all := rx.FindAllSubmatch(line, -1); all != nil {
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
          if _, e := p.scan(p.res.position, &knownMatch{ rx, l, a }); e != nil {
            p.errors = append(p.errors, e)
          }
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
    if p.report { erro(ctx, "dokcer daemon not running (at %s)", sock).at(pos).debug(1) }
  } else {
    // TODO: start docker daemon
  }
  return
}
func (p *ExecBuffer) filepath(s string) string {
  if p.defaultDirectory != "" && !filepath.IsAbs(s) {
    s = filepath.Join(p.defaultDirectory, s)
  }
  return s
}
func (p *ExecBuffer) convPos(s1, s2, s3 string) Position {
  return convPosition(p.filepath(s1), s2, s3)
}
func (p *ExecBuffer) scan(pos Position, m *knownMatch) (status int, err error) {
  var ctx = p.res.ctx
  if p == nil {
    erro(ctx, "nil exec buffer").at(pos).debug(1)
    return
  }
  var (
    container *Project = p.res.container
    lpos Position = pos
    reportIncludedFrom = func() (res bool) {
      if p.includedFrom.pos1.IsValid() && p.includedFrom.pos2.IsValid() {
        erro(ctx, "… included from here").at(p.includedFrom.pos1)
        erro(ctx, "… reported here").at(p.includedFrom.pos2).debug(4)
        p.includedFrom.pos1 = Position{}
        p.includedFrom.pos2 = Position{}
        res = true
      }
      return
    }
    addScannedDiag = func(dt diagType, pos Position, msg string) {
      var done bool
      for _, rec := range p.res.scannedDiags {
        if rec.msg == msg { rec.num += 1; done = true }
      }
      if !done {
        var e = &scannedExecDiag{ pos, lpos, dt, msg, 1 }
        p.res.scannedDiags = append(p.res.scannedDiags, e)
      }
    }
  )

  if p.log != nil { lpos.Filename = p.log.filename }
  if     m != nil { lpos.Line, lpos.Column = m.l, 0 }
  for _, v := range m.v { // captures
    if len(v) > 1 { lpos.Column = v[1].col }
    switch m.rx {
    //case rxCompilationDefaultDirectory: p.defaultDirectory = v[1].string
    case rxNotTTYDevice:
      if p.report {
        addScannedDiag(diagError, lpos, fmt.Sprintf(`Needs TTY (input device)`))
      }
    case rxDockerDaemonNotRunning:
      if err = p.startDockerDaemon(lpos, ctx, container, v[1].string); err != nil {
        addScannedDiag(diagError, lpos, fmt.Sprintf("start container failed: %v", err))
      }
    case rxNoContainer:
      if name := v[1].string; p.res.skips(name) {
        if p.report {
          addScannedDiag(diagError, lpos, fmt.Sprintf("container not running: %v", name))
        }
      } else {
        p.res.containerToRun = name
      }
    case rxContainerNotRunning:
      if p.report {
        addScannedDiag(diagError, lpos, fmt.Sprintf("Container not running (%v)", v[1].string))
      }
    case rxNoNetwork:
      if p.report {
        addScannedDiag(diagError, lpos, fmt.Sprintf("Network not found (%v)", v[1].string))
      }
    case rxIncludedFrom2:
      if p.report {
        lpos.Column = v[2].col + 1
        p.includedFrom.pos1 = p.convPos(v[1].string, v[2].string, "1")
        p.includedFrom.pos2 = lpos
      }
    case rxIncludedFrom3:
      if p.report {
        lpos.Column = v[3].col + 1
        p.includedFrom.pos1 = p.convPos(v[1].string, v[2].string, v[3].string)
        p.includedFrom.pos2 = lpos
      }
    case rxCompilationError:
      if p.report {
        p.errorPos = p.convPos(v[1].string, v[2].string, v[3].string)
        lpos.Column = v[4].col + 1
        if s := v[5].string; s != "" {
          addScannedDiag(diagError, lpos, fmt.Sprintf("%s: %s", v[4].string, s))
        } else {
          addScannedDiag(diagError, lpos, fmt.Sprintf("%s", v[4].string))
        }
        if false && !reportIncludedFrom() { erro(ctx, "…reported here").at(lpos).debug(1) }
      }
    case rxCompilationWarning:
      if p.report {
        var wpos = p.convPos(v[1].string, v[2].string, v[3].string)
        lpos.Column = v[4].col + 1
        addScannedDiag(diagWarn, lpos, fmt.Sprintf("%s", v[4].string))
        addScannedDiag(diagWarn, wpos, "warning from here")
        if false && !reportIncludedFrom() { warn(ctx, "…reported here").at(lpos).debug(1) }
      }
    case rxProtoFileNotFound:
      if p.report {
        var pos = lpos
        lpos.Column = v[1].col
        addScannedDiag(diagError, pos, fmt.Sprintf(`"%v" file not found`, v[1].string))
      }
    case rxProtoImportNotFound:
      if p.report {
        lpos.Column = v[4].col
        var pos = p.convPos(v[1].string, v[2].string, v[3].string)
        addScannedDiag(diagError, pos, fmt.Sprintf(`Import "%v" not found or errors`, v[4].string))
        if false && !reportIncludedFrom() { erro(ctx, "…reported here").at(lpos).debug(1) }
      }
    case rxProtoNameNotDefined:
      if p.report {
        lpos.Column = v[4].col
        var pos = p.convPos(v[1].string, v[2].string, v[3].string)
        addScannedDiag(diagError, pos, fmt.Sprintf(`"%v" is not defined`, v[4].string))
        if false && !reportIncludedFrom() { erro(ctx, "…reported here").at(lpos).debug(1) }
      }
    case rxFatalErrorFileNotFound:
      if p.report {
        lpos.Column = v[4].col
        var pos = p.convPos(v[1].string, v[2].string, v[3].string)
        addScannedDiag(diagError, pos, fmt.Sprintf(`"%v" file not found`, v[4].string))
        if false && !reportIncludedFrom() { erro(ctx, "…reported here").at(lpos).debug(1) }
      }
    case rxArNoSuchFile:
      if p.report {
        addScannedDiag(diagError, lpos, fmt.Sprintf("'%v' file not found (as '%v')", filepath.Base(v[1].string), v[1].string))
      }
    case rxArNoArchiveMembers:
      if p.report {
        if false {
          var obj = closureResolveObject(ctx, lpos, "objects")
          erro(ctx, "%s", v[0].string).at(lpos)
          erro(ctx, "%s", obj).at(lpos)
          if !isNil(obj) {
            if val, _ := obj.expand(ctx.closure().programCtx(), expandPlainValue); !isNil(val) {
              erro(ctx, "%s -> %v", obj.Name(ctx), val).at(lpos)
            }
          }
          erro(ctx, "%v", ctx).debug(16)
        } else {
          addScannedDiag(diagError, lpos, v[0].string)
        }
      }
    case rxBashNoSuchFile:
      if p.report {
        lpos.Column = v[2].col + 1
        addScannedDiag(diagError, lpos, fmt.Sprintf("no such command '%v'", v[2].string))
      }
    case rxClangNoSuchFile:
      if p.report {
        var vs string
        if s := v[1].string; s != "" { vs = "-" + s }
        lpos.Column = v[2].col + 1
        addScannedDiag(diagError, lpos, fmt.Sprintf("clang%s: no such source file: %s", vs, v[2].string))
      }
      /*
    case rxClangError:
      if p.report {
        var vs string
        if s := v[1].string; s != "" { vs = "-" + s }
        lpos.Column = v[2].col + 1
        if false {
          erro(ctx, "clang%s: %s", vs, v[2].string).at(lpos).debug(1)
        } else {
          addScannedDiag(diagError, lpos, fmt.Sprintf("clang%s: %s", vs, v[2].string))
        }
        p.res.errs += 1
      }*/
    case rxCmdError:
      if p.report {
        var cs, vs string; cs = v[1].string
        if s := v[2].string; s != "" { vs = "-" + s }
        lpos.Column = v[3].col + 1
        addScannedDiag(diagError, lpos, fmt.Sprintf("%s%s: %s", cs, vs, v[3].string))
      }
    case rxCmdWarning:
      if p.report {
        lpos.Column = v[2].col + 1
        addScannedDiag(diagWarn, lpos, fmt.Sprintf("%s: %s", v[1].string, v[2].string))
      }
    case rxLdLibNotFound:
      if p.report {
        lpos.Column = v[2].col + 1
        addScannedDiag(diagError, lpos, v[0].string)
      }
    case rxCouldnotParseObj:
      if p.report {
        lpos.Column = v[3].col
        addScannedDiag(diagError, lpos, v[3].string)
      }
    case rxTooManyPosArgs:
      if p.report {
        addScannedDiag(diagError, lpos, fmt.Sprintf("%s: too many positional arguments", v[1].string))
      }
    case rxUndefinedReference:
      if p.report {
        addScannedDiag(diagError, lpos, fmt.Sprintf("Undefined reference '%s'", v[1].string))
      }
    case rxShellCmdNotFound:
      if p.report {
        lpos.Column = v[2].col
        addScannedDiag(diagError, lpos, fmt.Sprintf("%s: command not found", v[2].string))
      }
    case rxIgnoringNonExistentDirectory:
      if p.report {
        var dir = v[1].string;  lpos.Column = v[1].col + 1
        addScannedDiag(diagInfo, lpos, fmt.Sprintf(`ignoring nonexistent directory "%v"`, dir))
      }
    case rxIgnoringDuplicateDirectory:
      if p.report {
        var dir = v[1].string;  lpos.Column = v[1].col + 1
        addScannedDiag(diagInfo, lpos, fmt.Sprintf(`ignoring duplicate directory "%v"`, dir))
      }
    case rxExitStatus:
      if s := v[1].string; s != "0" /*&& p.report*/ {
        // FIXME: the 'exit status' report is not working
        addScannedDiag(diagError, lpos, fmt.Sprintf("abnormal exist status %s", s))
      }
    }
    if err != nil { break }
  }
  return
}

type ExecResult struct {
  valbase

  Stdout ExecBuffer
  Stderr ExecBuffer
  Status int // aka. exit code

  retried map[string]bool // work with containerToRun
  containerToRun string   // work with retried

  printEnteringOnFirstWrote bool

  num int
  ctx Context
  x *executor
  sh *exec.Cmd
  container *Project

  scannedDiags []*scannedExecDiag
}
func (p *ExecResult) cmp(ctx Context, v Value) (res cmpres) {
  if a, ok := v.(*ExecResult); ok {
    assert(ok, "value is not ExecResult")
    if p.Status == a.Status { res = cmpEqual }
  }
  return
}
func (p *ExecResult) True(ctx Context) (res bool, err error) {
  res = p.Status == 0 && p.Stderr.Buf != nil && p.Stderr.Buf.Len() == 0 /* && p.Stdout.Buf.Len() > 0 */
  return
}
func (p *ExecResult) Integer(ctx Context) (int64, error) {
  return int64(p.Status), nil
}
func (p *ExecResult) Float(ctx Context) (float64, error) {
  return float64(p.Status), nil
}
func (p *ExecResult) Strval(ctx Context) (s string, err error) {
  if p.Stdout.Buf != nil { s = p.Stdout.Buf.String() }
  return
}
func (p *ExecResult) String() string {
  var s bytes.Buffer
  fmt.Fprintf(&s, "(ExecResult status=%d", p.Status)
  if p.Stdout.Buf != nil { fmt.Fprintf(&s, " stdout=%S", p.Stdout.Buf) }
  if p.Stderr.Buf != nil { fmt.Fprintf(&s, " stdout=%S", p.Stderr.Buf) }
  fmt.Fprintf(&s, ")")
  return s.String()
}

func (p *ExecResult) onFirstWrote() {
  if p.printEnteringOnFirstWrote {
    printEnteringDirectory(p.ctx)

    // Call checkErrors to ensure printEnteringDirectory works immediately
    if errs := p.ctx.checkErrors(true); errs > 0 {
      warn(p.ctx, "exec: encountered %d errors", errs).debug(1)
    }
  }
}

func (p *ExecResult) runContainerAndRetry(ctx Context) (status int, err error) {
  if p.container == nil {
    erro(ctx, "no container").at(p.position).debug(1)
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
  if entries, _ := p.container.resolveEntries(ctx, "run", false, false); entries != nil {
    for _, run := range entries.all {
      if _, brks := run.execute(p.ctx); brks.has() {
        erro(ctx, "%d breakers", len(brks)).at(p.position).debug(1)
        return
      } //else { p.t.group.Wait() }
    }
  } else {
    erro(ctx, "%s⇒run undefined", p.container).debug(1)
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
  if status, err = p.run(ctx); err != nil {
    fmt.Fprintf(sh.Stderr, "\n---- Retry failed: %s\n", err)
  }
  return
}

// DEPRECATED
func (p *ExecResult) ensureContainerRunning(ctx Context, containerName string) (err error) {
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
      fmt.Fprintf(stderr, "%s", s)
    }
  } (stderrR)

  if err = cmd.Run(); err == nil && foundID == "" {
    if entries, _ := p.container.resolveEntries(ctx, "run", false, false); entries != nil {
      for _, run := range entries.all {
        if _, brks := run.execute(p.ctx); brks.has() {
          erro(ctx, "%d breakers", len(brks)).at(p.position).debug(1)
          return
        } //else { p.t.group.Wait() }
      }
    } else {
      erro(ctx, "%s⇒run undefined", p.container).debug(1)
      return
    }
  } else if err != nil {
    erro(ctx, "%v", err).at(p.container.position).debug(1)
  }
  return
}

func (p *ExecResult) skips(tag string) bool {
  if p.retried == nil { p.retried = make(map[string]bool) }
  var a, b = p.retried[tag]
  return a && b
}

func (p *ExecResult) run(ctx Context) (status int, err error) {
  p.num += 1
  if err = p.sh.Run(); err == nil {
    return
  } else if ee, ok := err.(*exec.ExitError); !ok {
    erro(ctx, "exec failed: %v", err).debug(1)
    return
  } else if status = ee.ExitCode(); status == 0 {
    return // success!
  } else if p.containerToRun != "" {
    p.retried[p.containerToRun] = true // mark it to skip next time
    status, err = p.runContainerAndRetry(ctx)
    p.containerToRun = ""
  }
  return
}

type (
  executorOpts struct {
    deprecated bool `v,vo;w,ve;a,a;d,dump`
    debug  bool `d,debug`
    infos  bool `sci,scan-infos`
    prompt bool `pm,prompt;m,msg`
    promStr string "c,cmd;m,msg"
    silent bool "s,silent" // silent errors
    verboseSrc bool `vs,verbose-source`
    tieStdout  bool "to,tie-out;to,tie-stdout" // tied with log
    tieStderr  bool "te,tie-err;te,tie-stderr" // tied with log
    tie string `t,tie` // all, both, stdout, stderr, out, err
    bufStdout bool "o,stdout;bo,buffer-stdout;so,save-stdout"
    bufStderr bool "e,stderr;be,buffer-stderr;se,save-stderr"
    stdin  bool "i,stdin;in,input"
    stamp  bool `st,stamp;sf,stamp-file`
    wait   bool `w,wait;wr,wait-result` // wait for execution finished
    report bool `r,report;rs,report-stamp;vs,verbose-stamp`
    retStdout bool `ro,return-stdout;ro,result-stdout`
    retStderr bool `ro,return-stderr;ro,result-stderr`
    retStatus bool `ro,return-status;ro,result-status`
    fullname   bool `f,full;fn,fullname` // expand fullname
    scanStdout bool `so,scan-stdout;so,scan-out`
    scanStderr bool `se,scan-stderr;se,scan-err`
    parallel   bool `par,parallel;no,no-order`
    path bool "p,path"
    noCD bool "n,nocd"
    logFileName *optFullname "l,log"
  }
  executor struct {
    cmd, opt string
    contained bool
  }
)
func (p *executor) Evaluate(ctx Context, args ...Value) (result Value, err error) {
  if options.traceExecutor {
    var t, _ = ctx.autoGet("@")
    defer un(trace(t_exec, fmt.Sprintf("executor(%s %v)", typeof(t), t)))
  }

  var (
    opts = executorOpts{ scanStderr: true }
    pos = ctx.Position()
    cmd = p.cmd
  )
  if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
    erro(ctx, "merge args failed: %v", err).debug(1)
    return
  } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
    erro(ctx, "parse opts failed: %v", err)
    errostack(ctx, 5, "%v", ctx).debug(1)
    return
  } else if opts.deprecated {
    erro(ctx, "deprecated args: -v (-to), -w (-te), -a (-se), -d (-t)").debug(1)
    return
  } else if !opts.prompt {
    opts.prompt = opts.promStr != ""
  }
  switch opts.tie {
  case "stdout", "out" : opts.tieStdout = true
  case "stderr", "err" : opts.tieStderr = true
  case "all"   , "both": opts.tieStdout, opts.tieStderr = true, true
  }

  var (
    t = ctx.traversal()
    program = ctx.program()
    target = getTargetValue(ctx)
    targetName string
  )
  if program == nil {
    erro(ctx, "needs program context to exec: %v", ctx).debug(16)
    return
  }
  if _, ok := target.(*Flag); ok {
    // no stamp required for Flags
  } else if _, ok = target.(*File); !ok {
    // no stamp required for non-file targets
  } else if targetName, err = fullnameOrStrval(ctx, target); err != nil {
    erro(ctx, "stringify target '%v' failed: %v", target, err).of(target).debug(1)
    return
  } else if ctx.configuration() {
    // does nothing
  } else if opts.wait {
    // good to work without (stamp) or (wait) with the -wait flag
  } else if ms := program.getModifiers(ctx, "stamp"); len(ms) > 0 {
    switch target.(type) {
    case *Barefile, *File, *Path:
      warn(ctx, "use (shell -stamp) instead of stamp modifier (%T %v)", target, target).at(ms[0].position).debug(1)
    }
  } else if ms := program.getModifiers(ctx, "wait"); len(ms) > 0 {
    // should be good to work
  } else if !opts.stamp && !opts.silent {
    warn(ctx, "add -stamp to (shell); target=%v (%T)", target, target).debug(1)
  }
  if strings.HasSuffix(targetName, "external.google.tensorflow.prototext") {
    defer func() { warn(ctx, "%v", target).debug(10) } ()
  }

  var (
    start = time.Now()
    exeres = &ExecResult{valbase:valbase{pos}, ctx:ctx, x:p}
    aa []string
  )
  for i, v := range args {
    var s string
    if p.contained && i == 0 {
      if s, err = v.Strval(ctx); err != nil {
        erro(ctx, "strval '%v' failed: %v", v, err).of(v).debug(1)
        return
      } else if s == "shell" {
        cmd = defaultShell
      }
    } else if s, err = v.Strval(ctx); err != nil {
      erro(ctx, "strval '%v' failed: %v", v, err).of(v).debug(1)
      return
    } else if s = strings.TrimSpace(s); s != "" {
      aa = append(aa, s)
    }
  }

  var container *Project
  if p.contained {
    if program.project.name == dotContainer {
      container = program.project
    } else if _, containerSym := program.project.scope.Find(dotContainer); containerSym != nil {
      if pn, _ := containerSym.(*ProjectName); pn != nil {
        container = pn.NamedProject()
      }
    }

    if container == nil {
      erro(ctx, "container unavailable (in %s)", program.project.name).debug(1)
      return
    }

    var strval = func(name string) (str string, err error) {
      var ctx = closureWith(ctx, pos, container.Scope())
      if obj, _ := container.resolveObject(ctx, name); obj != nil {
        if def, _ := obj.(*Def); def != nil {
          var v Value
          if v, err = def.DiscloseValue(ctx); err == nil && v != nil {
            if str, err = v.Strval(ctx); str == "-" {
              /*if v, err = def.DiscloseValue(container); err == nil && v != nil {
                  if str, err = v.Strval(ctx); str == "" { str = "-" }
                  fmt.Fprintf(stderr, "%v: %v (%v)\n", name, str, def)
                }*/
            }
          }
        }
      }
      return
    }

    var containerName string
    if containerName , err = strval("container"); err != nil {
      erro(ctx, "strval .container.container failed: %v", err).debug(1)
      return
    } else if containerName == "" {
      erro(ctx, ".container.name undefined").debug(1)
      return
    }

    var containerImage string
    if containerImage, err = strval("image"); err != nil {
      erro(ctx, "strval .container.image failed: %v", err).debug(1)
      return
    } else if containerImage == "" {
      erro(ctx, ".container.image undefined").debug(1)
      return
    }

    if options.verbose {
      prompt(ctx, "%v: container=%v, image=%v\n", container, containerName, containerImage)
    }

    aa = append(aa, "exec", containerName, cmd)
    cmd = "docker"
  }

  var cwd string
  {
    var (
      cc = positional(ctx, program.position)
      o Object
      v Value
    )
    if _, o = program.scope.Find("CWD"); isNil(o) {
      if _, o = program.scope.Find("/"); isNil(o) {
        erro(ctx, "'CWD' and '/' is undefined").debug(1)
        return
      }
    }
    if v = o.(*Def).Call(cc); isNil(v) || isNone(v) {
      erro(ctx, "CWD is <nil>").debug(1)
      return
    } else if cwd, err = v.Strval(ctx); err != nil {
      erro(ctx, "strval '%v' failed: %v", v, err).debug(1)
      return
    } else if cwd == "" {
      erro(ctx, "CWD is empty").debug(1)
      return
    }
  }

  // Fixes work directory conflicts. It happens
  // sometimes even the 'sh.Dir' is set to cwd.
  // Because the current work directory is not
  // thread safe.
  var dir = cwd
  if program.changedWD != "" {
    if filepath.IsAbs(program.changedWD) {
      dir = program.changedWD
    } else {
      dir = filepath.Join(program.project.absPath, program.changedWD)
    }
  }

  if opts.path {
    var s string
    if s = filepath.Dir(targetName); s != "" && s != "." && s != "/" {
      if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
        erro(ctx, "make path '%s' for target failed: %v", s, err).of(target).debug(1)
        return
      }
    }
  }

  var envars []*Pair // disclosed values
  if def, _ := program.scope.Lookup(TheShellEnvarsDef).(*Def); def != nil {
    if l, _ := def.value.(*List); l != nil {
      for _, v := range l.Elems {
        var t Value
        if t, err = v.expand(ctx, expandClosure); err != nil {
          erro(ctx, "expand value '%v' failed: %v", v, err).of(v).debug(1)
          return
        } else if isNil(t) { t = v }
        if p, ok := t.(*Pair); ok {
          envars = append(envars, p)
        } else {
          erro(ctx, "env expecting pairs: %T", t).of(t).debug(1)
          return
        }
      }
    }
  }

  var (
    positions []Position
    recipePos Position
    recipes []Value
    sources []string
    source string
    w = expandPlainValue
  )
  if opts.fullname { w |= expandFullName }
  if recipes, err = expandmerge2(ctx, w, program.recipes...); err != nil {
    erro(ctx, "merge recipes failed: %v", err).debug(1)
    return
  }
  for _, recipe := range recipes {
    var str string
    if !recipePos.IsValid() { recipePos = recipe.Position() }
    if str, err = recipe.Strval(ctx); err != nil {
      erro(ctx, "strval recipe failed: %v", err).of(recipe).debug(1)
      return
    } else if str = strings.TrimRightFunc(str, unicode.IsSpace); str == "" {
      source += "\n" // an empty line
      continue
    } else if source += str; strings.HasSuffix(source, "\\") {
      source += "\n" // append the line feed
      continue
    }

    // Escape '$$' sequences.
    source = strings.Replace(source, "$$", "$", -1)

    // Remove tabs in line breakings.
    source = strings.Replace(source, "\\\n\t", "\\\n", -1)

    // Duplicate all %
    //source = strings.Replace(source, "%", "%%", -1)

    positions = append(positions, recipePos); recipePos = Position{}
    sources = append(sources, source)
    source = ""
  }
  if len(recipes) > 0 && len(sources) == 0 {
    erro(ctx, "empty recipes: %v", recipes).debug(1)
    return;
  }

  var envstr string
  var envs []string = os.Environ()
  for i, p := range envars {
    var k, v string
    if k, err = p.Key.Strval(ctx); err != nil {
      erro(ctx, "strval '%v' failed: %v", p.Key, err).of(p.Key).debug(1)
      return
    }
    if v, err = p.Value.Strval(ctx); err != nil {
      erro(ctx, "strval '%v' failed: %v", p.Value, err).of(p.Value).debug(1)
      return
    }
    if i > 0 { envstr += " && " }
    envstr += fmt.Sprintf(`%s=%s`, k, strconv.Quote(v))
    envs = append(envs, fmt.Sprintf("%s=%s", k, v))
  }

  var log *ExecLog
  var logFile *os.File
  if opts.logFileName != nil { log = &ExecLog{ filename: opts.logFileName.string } }
  if opts.bufStdout { exeres.Stdout.Buf = new(bytes.Buffer) }
  if opts.bufStderr { exeres.Stderr.Buf = new(bytes.Buffer) }
  if opts.tieStdout { exeres.Stdout.Tie = stdout }
  if opts.tieStderr { exeres.Stderr.Tie = stderr }
  if log == nil || log.filename == "" {
    // no log required
  } else if err = os.MkdirAll(filepath.Dir(log.filename), os.FileMode(0755)); err != nil {
    erro(ctx, "%v", err).at(program.position).debug(1)
    return
  } else if logFile, err = os.Create(log.filename); err != nil {
    erro(ctx, "%v", err).at(program.position).debug(1)
    return
  } else {
    cmdline := strings.Join(sources, "\n")
    log.createWriter(logFile, dir, cmdline)
    exeres.Stdout.log = log
    exeres.Stderr.log = log
  }
  exeres.Stdout.scanKnownErrors = opts.scanStdout
  exeres.Stderr.scanKnownErrors = opts.scanStderr
  exeres.Stdout.defaultDirectory = dir
  exeres.Stderr.defaultDirectory = dir
  exeres.Stdout.res = exeres
  exeres.Stderr.res = exeres

  if ctx.checkErrors(true) > 0 {
    if str := trimPromptString(targetName); filepath.IsAbs(targetName) {
      var pos Position; pos.Filename, pos.Line = targetName, 1
      warn(ctx, "got %d error(s)", ctx.totalErrors())
      warn(ctx, "cancel execution for %s", str).at(pos).debug(1)
    } else {
      warn(ctx, "got %d error(s), cancel execution for %s", ctx.totalErrors(), str).debug(1)
    }
    if options.failOnErrors { fail(ctx.Position(), "fail by %d errors", ctx.totalErrors()) }
    return
  }

  var caller = ctx.traversal().caller()
  defer func() {
    if log != nil && log.writer != nil { log.writer.Flush() }
    if logFile != nil { logFile.Close() }
    if false && log.filename != "" && exeres.Stdout.wrote == 0 && exeres.Stderr.wrote == 0 {
      os.Remove(log.filename)
    }
    if caller != nil && err != nil { caller.calleeError(err) }
    exeres.Stdout.res = nil
    exeres.Stderr.res = nil
    exeres.container = nil
    exeres.ctx = nil
    exeres.sh = nil
    exeres.x = nil
  } ()

  defer func() {
    if !opts.stamp || ctx.configuration() {
      // no stamp for target files
    } else if err != nil {
      var files, _ = target.delete(ctx)
      prompt(ctx, "%v: %v (won't stamp)\n", target, err)
      for _, file := range files {
        var fullname = file.fullname()
        if s := file.String(); s == fullname {
          erro(ctx, `%v: deleted`, s)
        } else {
          erro(ctx, `%v: deleted "%v"`, s, fullname)
        }
      }
      errostack(ctx, 3, `%v: (%T)`, target, ctx).debug(6)
      if /*opts.fail*/true { fail(target.Position(), `"%v" deleted`, target) }
      return
    } else if files, err := target.stamp(t); err != nil {
      if pe, ok := err.(*fs.PathError); ok { err = fmt.Errorf(`"%v" not found`, target)
        prompt(ctx, "%v: not found, stamp \"%v\"\n", pe.Path, target)
      } else {
        prompt(ctx, "%v: not found, \"%v\"\n", pe.Path, err)
      }
      erro(ctx, `stamp "%v" failed`, target)
      errostack(ctx, 6, `%v: %v`, target, ctx).debug(10)
      if /*opts.fail*/true { fail(target.Position(), `"%v" not generated`, target) }
      return
    } else if opts.report {
      reportFileUpdates(ctx, t.start, files)
    }

    if err == nil {
      // Good!
    } else if ctx.configuration() {
      err = nil
    } else {
      erro(ctx, "shell: %v", err).debug(1)
      return
    }

    if opts.prompt {
      var (
        ps = opts.promStr
        st = trimPromptString(targetName)
      )
      if caller == nil {
        if st += " …… "; err == nil {
          st += "ok"
        } else if _, ok := err.(*scanner.Error); ok {
          st += "scan error" // fmt.Fprintf(stderr, "%v\n", err)
        } else {
          st += err.Error()
        }
      }
      prompt(ctx, "%s%s (%v, stdout=%d bytes, stderr=%d bytes)\n", ps, st, time.Now().Sub(start),
        exeres.Stdout.wrote, exeres.Stderr.wrote)
    }
  } ()

  for i, src := range sources {
    var pos = positions[i]
    if strings.HasPrefix(src, "@") {
      src = src[1:]
    } else if opts.verboseSrc && !opts.prompt {
      var s string = src
      s = strings.Replace(s, "\n", "\\n", -1)
      s = strings.Replace(s, "\\\\n", "\\\n", -1)
      prompt(ctx, "%s\n", s)//.debug(1)
    }
    if src = strings.TrimSpace(src); src == "" { continue }
    if dir != "" && !opts.noCD /*&& program.changedWD == ""*/ {
      if strings.HasPrefix(src, "#") {
        src = fmt.Sprintf("cd '%s' %s", dir, src)
      } else {
        // Insert a "\n" before the right paren ')' to ensure that
        // it's working with comments like "true #comment...".
        src = fmt.Sprintf("cd '%s' && (%s\n)", dir, src)
      }
    }
    if cmd == "docker" && len(envstr) > 0 {
      src = fmt.Sprintf("%s && %s", envstr, src)
    }

    if options.noExec { continue }
    if false {
      // Restricts the number of workers.
      ///fmt.Fprintf(stderr, "run.1: %v\n", targetName)
      var num = waitForWork(); defer releaseWork(num)
      ///fmt.Fprintf(stderr, "run.2: %v\n", targetName)
    }

    for {
      if err = lockCD(dir, 25*time.Millisecond); err != nil {
        erro(ctx, "%v", err).debug(1)
        return
      } else if s, _ := os.Getwd(); s == dir { break }
    }
    if !opts.silent || opts.prompt || opts.verboseSrc {
      exeres.printEnteringOnFirstWrote = true
    }

    exeres.container = container
    exeres.sh = exec.Command(cmd, aa...)
    exeres.sh.Dir = dir
    exeres.sh.Env = envs // always set command work directory
    exeres.sh.Stdout = &exeres.Stdout
    exeres.sh.Stderr = &exeres.Stderr
    if opts.stdin {
      exeres.sh.Stdin = os.Stdin
      exeres.sh.Args = append(exeres.sh.Args, "-ti")
    }
    if p.opt != "" { exeres.sh.Args = append(exeres.sh.Args, p.opt) }
    if src   != "" { exeres.sh.Args = append(exeres.sh.Args, src) }
    if opts.debug {
      warn(ctx, "%v: %v", ctx.entry(), target).at(program.position)
      warn(ctx, "context: %v", t)
      warn(ctx, "exec:\n%v", exeres.sh).debug(1)
    }

    exeres.Stdout.report = !opts.silent
    exeres.Stderr.report = !opts.silent
    exeres.Status, err = exeres.run(positional(ctx, pos))
    if (!opts.silent || opts.debug) && (len(exeres.scannedDiags) > 0 || exeres.Status != 0 || err != nil) {
      var ( lpos Position; en, wn, in int )
      for _, rec := range exeres.scannedDiags {
        switch rec.dt {
        case diagError: en += rec.num
        case diagWarn:  wn += rec.num
        case diagInfo:  in += rec.num
        }
      }
      if en > 0 || exeres.Status != 0 || err != nil {
        prompt(ctx, "%v: exec failed (status=%d; err=%v)\n", targetName, exeres.Status, err)
      } else if wn > 0 {
        prompt(ctx, "%v: %d warnings\n", targetName, wn)
      }
      for i, rec := range exeres.scannedDiags {
        if !opts.infos && rec.dt == diagInfo { continue }
        if !lpos.IsValid() { lpos = rec.lpos }
        if i == 0 && !rec.position.Equals(&rec.lpos) {
          diag(ctx, rec.dt, rec.msg).at(rec.lpos)//.debug(1)
        }
        if rec.num > 1 {
          diag(ctx, rec.dt, `%s (%d)`, rec.msg, rec.num).at(rec.position)//.debug(1)
        } else {
          diag(ctx, rec.dt, rec.msg).at(rec.position)//.debug(1)
        }
        if n := (en+wn+in)-(i+1); i == 8 && 0 < n {
          diag(ctx, rec.dt, "%d more...", n).at(rec.lpos)//.debug(1)
          break
        }
      }
      if !lpos.IsValid() && log != nil {
        lpos.Filename = log.filename
        lpos.Line = exeres.Stderr.log.lines + 1
      } else {
        lpos = ctx.Position()
      }
      var str, _, _ = entryStr(ctx, ctx.entry())
      if exeres.Status != 0 { err = &exitstatus{ exeres.Status } // set or convert error
        erro(ctx, "%v: abnormal exit status %d", str, exeres.Status).at(lpos)
      } else if err != nil { if opts.silent { err = nil }
        erro(ctx, "%v: %s", str, err).at(lpos)
      }
      if en > 0 {
        erro(ctx, "%v: scanned %d known errors", str, en).at(lpos)
        erro(ctx, "%v: execute failed (%d errors)", str, en)
        errostack(ctx, 32, "%v: (%T)", str, ctx).debug(6)
      } else if wn > 0 {
        if false {
          warn(ctx, "%v: scanned %d known warnings", str, wn).at(lpos)
          warn(ctx, "%v: execute has %d warnings", str, wn)
          warnstack(ctx, 3, "%v: %v", str, ctx).debug(1)
        } else if pos := ctx.Position(); lpos.Equals(&pos) {
          warn(ctx, "%v: scanned %d known warnings", str, wn)
          warnstack(ctx, 3, "%v: %v", str, ctx).debug(1)
        } else {
          warn(ctx, "%v: scanned %d known warnings", str, wn).at(lpos)
          warn(ctx, "%v: execute has %d warnings", str, wn)
          warnstack(ctx, 3, "%v: %v", str, ctx).debug(1)
        }
      } else if in > 0 && opts.infos {
        info(ctx, "%v: scanned %d known messages", str, in).at(lpos)
        info(ctx, "%v: execute has %d messages", str, in)
        infostack(ctx, 8, "%v: %v", str, ctx).debug(1)
      } else if err != nil || exeres.Status != 0 {
        errostack(ctx, 20, "%v: %v:", str, ctx).debug(6)
      }
      if exeres.Status != 0 || err != nil { break }
    }
  }

  // The execution is performed asynchronously, the result can't be fetched immediately.
  // Caller should do a t.wait(...) or exeres.wait() before using the result.
  var res []Value
  // TODO: if opts.retStdout { res = append(res, ) }
  // TODO: if opts.retStderr { res = append(res, ) }
  // TODO: if opts.retStatus { res = append(res, ) }
  if len(res) == 0 {
    result = exeres
  } else {
    result = MakeListOrScalar(pos, res)
  }
  return
}
