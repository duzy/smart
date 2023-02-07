//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
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
  // rxClangNoSuchFile = rx(`clang(?:-(.+?))?: error: no such file or directory: '(.+?)'`)
  // rxClangError      = rx(`clang(?:-(.+?))?: error: (.+)(?: \(.+\))?`)
  rxCmdError           = rx(`((?:clang|(?:[^\.]+\.)?l?ld|wasm)(?:\-.+?)?): error: (.+)`)
  rxCmdWarning         = rx(`((?:clang|(?:[^\.]+\.)?l?ld|wasm)(?:\-.+?)?): warning: (.+)`)
  rxCouldnotParseObj   = rx(`((?:clang|(?:[^\.]+\.)?l?ld|wasm)(?:\-.+?)?): could not parse object file (.+?): '(.+)', using libLTO version '(.+?)' file '(.+?)' for architecture (.+)`)
  rxLdLibNotFound      = rx(`((?:clang|(?:[^\.]+\.)?l?ld|wasm)(?:\-.+?)?): library not found for (.+)`)
  rxTooManyPosArgs     = rx(`(.+?): Too many positional arguments specified!`)
  rxUndefinedReference = rx(`  +"(.+?)", referenced from:`)
  rxShellCmdNotFound   = rx(`(.+?): (.+?):( command)? not found`)
  rxIgnoringDirectory  = rx(`ignoring (duplicate|nonexistent) directory "(.*?)"`)
  rxExitStatus = rx(`exit status (\-?[0-9]+)`)

  // NOTE: python standard errors
  rxPyErrorTrace = rx(`^\s*File "(.+?)", line (\d+), in (.+)`)
  rxPyModuleNotFoundError = rx(`ModuleNotFoundError: No module named '(.*?)'`)
  rxPyFileNotFoundError = rx(`FileNotFoundError: \[Errno (\d+)\] No such file or directory: '(.*?)'`)

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
    if p.report { erro(at(ctx,pos), "dokcer daemon not running (at %s)", sock).debug(1) }
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
    erro(at(ctx,pos), "nil exec buffer").debug(1)
    return
  }
  var (
    container *Project = p.res.container
    lpos Position = pos
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
        if false && !reportIncludedFrom() { erro(at(ctx,lpos), "…reported here").debug(1) }
      }
    case rxCompilationWarning:
      if p.report {
        var pos = p.convPos(v[1].string, v[2].string, v[3].string)
        var s = fmt.Sprintf("%s", v[4].string)
        lpos.Column = v[4].col + 1
        addScannedDiag(diagWarn, lpos, s)
        addScannedDiag(diagWarn,  pos, s)
        if false && !reportIncludedFrom() { warn(at(ctx,lpos), "…reported here").debug(1) }
      }
    case rxProtoFileNotFound:
      if p.report {
        var pos = lpos
        lpos.Column = v[1].col
        addScannedDiag(diagError, pos, fmt.Sprintf(`"%v" file not found`, v[1].string))
      }
    case rxProtoImportNotFound:
      if p.report {
        var pos = p.convPos(v[1].string, v[2].string, v[3].string)
        var s = fmt.Sprintf(`Import "%v" not found or errors`, v[4].string)
        lpos.Column = v[4].col
        addScannedDiag(diagError, lpos, s)
        addScannedDiag(diagError,  pos, s)
        if false && !reportIncludedFrom() { erro(at(ctx,lpos), "…reported here").debug(1) }
      }
    case rxProtoNameNotDefined:
      if p.report {
        var pos = p.convPos(v[1].string, v[2].string, v[3].string)
        var s = fmt.Sprintf(`"%v" is not defined`, v[4].string)
        lpos.Column = v[4].col
        addScannedDiag(diagError, lpos, s)
        addScannedDiag(diagError,  pos, s)
        if false && !reportIncludedFrom() { erro(at(ctx,lpos), "…reported here").debug(1) }
      }
    case rxFatalErrorFileNotFound:
      if p.report {
        var pos = p.convPos(v[1].string, v[2].string, v[3].string)
        var s = fmt.Sprintf(`"%v" file not found`, v[4].string)
        lpos.Column = v[4].col
        addScannedDiag(diagError, lpos, s)
        addScannedDiag(diagError,  pos, s)
        if false && !reportIncludedFrom() { erro(at(ctx,lpos), "…reported here").debug(1) }
      }
    case rxArNoSuchFile:
      if p.report {
        addScannedDiag(diagError, lpos, fmt.Sprintf("'%v' file not found (as '%v')", filepath.Base(v[1].string), v[1].string))
      }
    case rxArNoArchiveMembers:
      if p.report {
        if false {
          var obj = closureResolveObject(ctx, "objects")
          erro(at(ctx,lpos), "%s", v[0].string)
          erro(at(ctx,lpos), "%s", obj)
          if !isNil(obj) {
            if val := obj.expand(ctx.closure().programContext(), plain); !isNil(val) {
              erro(at(ctx,lpos), "%s -> %v", obj.Name(ctx), val)
            }
          }
          erro(ctx, "%v", ctx).debug(16)
        } else {
          addScannedDiag(diagError, lpos, v[0].string)
        }
      }
    case rxBashNoSuchFile:
      if p.report {
        var s = fmt.Sprintf("no such command '%v'", v[2].string)
        lpos.Column = v[2].col + 1
        addScannedDiag(diagError, lpos, s)
      }
    // case rxClangNoSuchFile:
    //   if p.report {
    //     var vs string
    //     if s := v[1].string; s != "" { vs = "-" + s }
    //     var s = fmt.Sprintf("clang%s: no such source file: %s", vs, v[2].string)
    //     lpos.Column = v[2].col + 1
    //     addScannedDiag(diagError, lpos, s)
    //   }
    // case rxClangError:
    //   if p.report {
    //     var vs string
    //     if s := v[1].string; s != "" { vs = "-" + s }
    //     lpos.Column = v[2].col + 1
    //     if false {
    //       erro(at(ctx,lpos), "clang%s: %s", vs, v[2].string).debug(1)
    //     } else {
    //       addScannedDiag(diagError, lpos, fmt.Sprintf("clang%s: %s", vs, v[2].string))
    //     }
    //     p.res.errs += 1
    //   }
    case rxCmdError:
      if p.report {
        var cs, vs string; cs = v[1].string
        lpos.Column = v[2].col + 1
        addScannedDiag(diagError, lpos, fmt.Sprintf("%s%s: %s", cs, vs, v[2].string))
      }
    case rxCmdWarning:
      if p.report {
        lpos.Column = v[2].col + 1
        addScannedDiag(diagWarn, lpos, fmt.Sprintf("%s: %s", v[1].string, v[2].string))
      }
    case rxCouldnotParseObj:
      if p.report {
        lpos.Column = v[2].col
        addScannedDiag(diagError, lpos, v[2].string)
      }
    case rxLdLibNotFound:
      if p.report {
        lpos.Column = v[2].col + 1
        addScannedDiag(diagError, lpos, v[0].string)
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
    case rxIgnoringDirectory:
      if p.report {
        var dir = v[2].string;  lpos.Column = v[2].col + 1
        addScannedDiag(diagInfo, lpos, fmt.Sprintf(`ignoring nonexistent directory "%v"`, dir))
      }
    case rxExitStatus:
      if s := v[1].string; s != "0" /*&& p.report*/ {
        // FIXME: the 'exit status' report is not working
        addScannedDiag(diagError, lpos, fmt.Sprintf("abnormal exist status %s", s))
      }
    case rxPyErrorTrace:
      if p.report {
        var pos = p.convPos(v[1].string, v[2].string, "")
        var s = fmt.Sprintf(`in %v`, v[3].string)
        lpos.Column = v[3].col
        addScannedDiag(diagError, lpos, s)
        addScannedDiag(diagError, pos, s)
      }
    case rxPyModuleNotFoundError:
      if p.report {
        var name = v[1].string;  lpos.Column = v[1].col + 1
        addScannedDiag(diagError, lpos, fmt.Sprintf(`no python module named "%v"`, name))
      }
    case rxPyFileNotFoundError:
      if p.report {
        var name = v[2].string;  lpos.Column = v[2].col + 1
        addScannedDiag(diagError, lpos, fmt.Sprintf(`no such file or directory "%v"`, name))
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
  x  *executor
  sh *exec.Cmd
  container *Project

  scannedDiags []*scannedExecDiag
}
func (p *ExecResult) expand(_ Context, _ facet) Value { return p }
func (p *ExecResult) cmp(ctx Context, v Value) (res cmpres) {
  if a, ok := v.(*ExecResult); ok {
    assert(ok, "value is not ExecResult")
    if p.Status == a.Status { res = cmpEqual }
  }
  return
}
func (p *ExecResult) True(ctx Context) (res bool) {
  res = p.Status == 0 && p.Stderr.Buf != nil && p.Stderr.Buf.Len() == 0 /* && p.Stdout.Buf.Len() > 0 */
  return
}
func (p *ExecResult) Integer(ctx Context) (i int64, _ error) { return int64(p.Status), nil }
func (p *ExecResult) Float(ctx Context) (f float64, _ error) { return float64(p.Status), nil }
func (p *ExecResult) Strval(ctx Context) (s string) {
  if p.Stdout.Buf != nil { s = p.Stdout.Buf.String() }
  return
}
func (p *ExecResult) String() string {
  var s bytes.Buffer
  fmt.Fprintf(&s, "(ExecResult status=%d", p.Status)
  if p.Stdout.Buf != nil { fmt.Fprintf(&s, " stdout=%S", p.Stdout.Buf) }
  if p.Stderr.Buf != nil { fmt.Fprintf(&s, " stderr=%S", p.Stderr.Buf) }
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
    erro(at(ctx,p.position), "no container").debug(1)
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
  if entries := p.container.resolveEntries(ctx, "run", false, false); entries != nil {
    for _, run := range entries.all {
      if _, traves := run.execute(p.ctx, nil); traves.has() {
        erro(at(ctx,p.position), "%d travestates", len(traves)).debug(1)
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
      fmt.Fprintf(stderr, "%s", s)
    }
  } (stderrR)

  if err = cmd.Run(); err == nil && foundID == "" {
    if entries := p.container.resolveEntries(ctx, "run", false, false); entries != nil {
      for _, run := range entries.all {
        if _, traves := run.execute(p.ctx, nil); traves.has() {
          erro(at(ctx,p.position), "%d travestates", len(traves)).debug(1)
          return
        } //else { p.t.group.Wait() }
      }
    } else {
      erro(ctx, "%s⇒run undefined", p.container).debug(1)
      return
    }
  } else if err != nil {
    erro(at(ctx,p.container.position), "%v", err).debug(1)
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
    generalOpts
    deprecated  bool `dump,deprecate`
    dropFailed  bool `df,drop-fail`
    infos       bool `sci,scan-infos`
    silentErrs  bool `s,silent,silent-errors` // silent errors
    zeroErrs    bool `ze,zero-errors` // require zero error scaned from STDERR
    tieStdout   bool `to,tie-out,tie-stdout` // tied with log
    tieStderr   bool `te,tie-err,tie-stderr` // tied with log
    bufStdout   bool `o,stdout;bo,buffer-stdout;so,save-stdout`
    bufStderr   bool `e,stderr;be,buffer-stderr;se,save-stderr`
    stdin       bool `i,stdin;in,input`
    stamp       bool `st,stamp;sf,stamp-file`
    noStamp     bool `ns,nostamp,no-stamp,no-stamp-file`
    wait        bool `wr,wait,waitres,wait-res,waitresult,wait-result` // wait for execution finished
    report      bool `r,report;rs,report-stamp;vs,verbose-stamp`
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
    tie         string `t,tie` // all, both, stdout, stderr, out, err
    workDir     string `cd,change-dir,wd,workdir,work-dir,work-directory`
    logFileName *optFullname "l,log"
  }
  executor struct {
    cmd, opt string
    contained bool
  }
)
func (p *executor) Evaluate(ctx Context, args ...Value) (result Value, err error) {
  if options.traceExecutor {
    var t = autoGet(ctx, "@")
    defer un(trace(t_exec, fmt.Sprintf("executor(%s %v)", typeof(t), t)))
  }

  var (
    opts = executorOpts{ scanStderr: true }
    pos = ctx.Position()
    cmd = p.cmd
  )
  if args = parseOpts(ctx, &opts, plain, args...); opts.deprecated {
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

  if false { defer func() { warn(ctx, "%v", cmd).debug(8) } () }

  var (
    programCtx = ctx.programContext()
    program = programCtx.program()
    target = getTargetValue(ctx)
    targetName string
  )
  if program == nil {
    erro(ctx, "needs program context to exec: %v", ctx).debug(16)
    return
  } else if opts.stamp && target.patterned(ctx) {
    errostack(ctx, 5, "target is pattern: %v", target).debug(64)
    return
  } else if _, ok := target.(*Flag); ok {
    // no stamp required for Flags
  } else if _, ok = toFile(target); !ok {
    // no stamp required for non-file targets
  } else if targetName = fullnameOrStrval(ctx, target); ctx.configuration() {
    // does nothing
  } else if opts.wait {
    // good to work without (stamp) or (wait) with the -wait flag
  } else if ms := program.getModifiers(ctx, "stamp"); len(ms) > 0 {
    switch target.(type) {
    case *Barefile, *File, *Path:
      warn(at(ctx,ms[0].position), "use (shell -stamp) instead of stamp modifier (%T %v)", target, target).debug(1)
    default:
      warn(at(ctx,ms[0].position), "no need to use (shell -stamp) here", target, target).debug(1)
    }
  } else if ms := program.getModifiers(ctx, "wait"); len(ms) > 0 {
    // should be good to work
  } else if !(opts.stamp || opts.noStamp || opts.silentErrs) {
    warn(ctx, "add -stamp to (shell); target=%v (%T)", target, target).debug(1)
  }

  if (opts.retStdout && opts.retStatus) || (opts.retStderr && opts.retStatus) {
    erro(ctx, "cannot have both status and stdout|stderr at the same time (try -so or -se)").debug(1)
    return
  }

  var (
    start = time.Now()
    exeres = &ExecResult{valbase:valbase{pos}, ctx:ctx, x:p}
    aa []string
  )
  for i, v := range args {
    var s string
    if p.contained && i == 0 {
      if s = v.Strval(ctx); s == "shell" {
        cmd = defaultShell
      }
    } else if s = strings.TrimSpace(v.Strval(ctx)); s != "" {
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

    var strval = func(name string) (str string) {
      var ctx = closureWith(ctx, container.Scope())
      if obj := container.resolveObject(ctx, name); obj != nil {
        if d, _ := obj.(*def); d != nil {
          if v := d.Call(ctx); v != nil {
            if str = v.Strval(ctx); str == "-" {
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
    if containerName = strval("container"); containerName == "" {
      erro(ctx, ".container.name undefined").debug(1)
      return
    }

    var containerImage string
    if containerImage = strval("image"); containerImage == "" {
      erro(ctx, ".container.image undefined").debug(1)
      return
    }

    if options.verbose {
      prompt(ctx, "%v: container=%v, image=%v\n", container, containerName, containerImage)
    }

    aa = append(aa, "exec", containerName, cmd)
    cmd = "docker"
  }

  // Fixes work directory conflicts. It happens
  // sometimes even the 'sh.Dir' is set to cwd.
  // Because the current work directory is not
  // thread safe.
  var workDir string
  if opts.workDir != "" {
    workDir = opts.workDir
  } else if workDir = program.workDir(ctx); workDir == "" {
    erro(ctx, "CWD is empty").debug(1)
    return
  }

  if opts.path {
    var s string
    if s = filepath.Dir(targetName); s != "" && s != "." && s != "/" {
      if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
        erro(of(ctx,target), "make path '%s' for target failed: %v", s, err).debug(1)
        return
      }
    }
  }

  var (
    positions []Position
    recipePos Position
    recipes []Value
    sources []string
    source string
    w = plain
  )
  if opts.fullname { w |= expandFullName }
  recipes = mergex(ctx, w, program.recipes...)

  for i, recipe := range recipes {
    if !recipePos.IsValid() { recipePos = recipe.Position() }

    var str = recipe.Strval(ctx)
    if false && strings.Contains(str, "llvm-driver-objcopy.cpp") {
      var vals = mergex(ctx, w, recipe.(*Compound).Elems...)
      warn(ctx, "%v %T", program.recipes[i], program.recipes[i])
      warn(ctx, "%v %T", recipe, recipe)
      warn(ctx, "%v", vals)
      warn(ctx, "%v", str)
      for _, val := range vals {
        if true {
          warn(ctx, "%T %v", val, val)
        } else if val.Strval(ctx) == "llvm-driver-objcopy.cpp" {
          warn(ctx, "%T %v", val, val)
          if c, ok := val.(*Barecomp); ok {
            for _, val := range c.Elems {
              warn(ctx, "%T %v", val, val)
            }
          }
        }
      }
      warnstack(ctx, 3, "").debug(1)
    }

    if str = strings.TrimRightFunc(str, unicode.IsSpace); str == "" {
      source += "\n" // an empty line
      continue
    } else if source += str; strings.HasSuffix(source, "\\") {
      source += "\n" // append the line feed
      // continue
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

  var (
    env, envSep = program.env(ctx)
    envstr string
  )
  for i, s := range env[envSep:] {
    if i > 0 { envstr += " && " }
    if k := strings.Index(s, "="); k > 0 {
      envstr += fmt.Sprintf(`%s%s`, s[:k+1], strconv.Quote(s[k+2:]))
    }
  }

  var (
    logPos Position
    logFile *os.File
    log *ExecLog
  )
  if opts.logFileName != nil { log = &ExecLog{ filename: opts.logFileName.string } }
  if opts.bufStdout || opts.retStdout { exeres.Stdout.Buf = new(bytes.Buffer) }
  if opts.bufStderr || opts.retStderr { exeres.Stderr.Buf = new(bytes.Buffer) }
  if opts.tieStdout { exeres.Stdout.Tie = stdout }
  if opts.tieStderr { exeres.Stderr.Tie = stderr }
  if log == nil || log.filename == "" {
    // no log required
  } else if err = os.MkdirAll(filepath.Dir(log.filename), os.FileMode(0755)); err != nil {
    erro(at(ctx,program.position), "%v", err).debug(1)
    return
  } else if logFile, err = os.Create(log.filename); err != nil {
    erro(at(ctx,program.position), "%v", err).debug(1)
    return
  } else {
    cmdline := strings.Join(sources, "\n")
    log.createWriter(logFile, workDir, cmdline)
    exeres.Stdout.log = log
    exeres.Stderr.log = log
  }
  exeres.Stdout.scanKnownErrors = opts.scanStdout
  exeres.Stderr.scanKnownErrors = opts.scanStderr
  exeres.Stdout.defaultDirectory = workDir
  exeres.Stderr.defaultDirectory = workDir
  exeres.Stdout.res = exeres
  exeres.Stderr.res = exeres

  if ctx.checkErrors(true) > 0 {
    if str := trimPromptString(targetName); filepath.IsAbs(targetName) {
      var pos Position; pos.Filename, pos.Line = targetName, 1
      warn(ctx, "got %d error(s)", ctx.totalErrors())
      warn(at(ctx,pos), "cancel execution for %s", str).debug(1)
    } else {
      warn(ctx, "got %d error(s), cancel execution for %s", ctx.totalErrors(), str).debug(1)
    }
    if options.failOnErrors { fail(ctx.Position(), "fail by %d errors", ctx.totalErrors()) }
    return
  }

  var caller = ctx.programContext().caller()
  defer func() {
    if log != nil && log.writer != nil { log.writer.Flush() }
    if logFile != nil { logFile.Close() }
    if log != nil && log.filename != "" &&
      exeres.Stdout.wrote == 0 && exeres.Stderr.wrote == 0 {
      if false { os.Remove(log.filename) }}
    if !opts.silentErrs && caller != nil && err != nil { caller.calleeError(err) }
    exeres.Stdout.res = nil
    exeres.Stderr.res = nil
    exeres.container = nil
    exeres.ctx = nil
    exeres.sh = nil
    exeres.x = nil

    // Stamp the target file.
    if !opts.stamp || ctx.configuration() {
      // no stamp for target files
    } else if err != nil {
      var files, e = target.delete(ctx)
      prompt(ctx, "%v: %v (deleted %d files)\n", target, err, len(files))
      if e != nil { erro(ctx, `%v: delete: %v`, target, e) }
      for _, file := range files {
        var fullname = file.fullname()
        if s := file.String(); s == fullname {
          erro(ctx, `%v: deleted`, s)
        } else {
          erro(ctx, `%v: deleted: %v`, s, fullname)
        }
      }
      errostack(ctx, 3, ``).debug(6)
      return
    } else if files, err := target.stamp(ctx); err != nil {
      if pe, ok := err.(*fs.PathError); ok { err = fmt.Errorf(`"%v" not found`, target)
        prompt(ctx, "%v: target not found, stamp \"%v\"\n", pe.Path, target)
      } else {
        prompt(ctx, "%v: target not found, \"%v\"\n", pe.Path, err)
      }
      if opts.logFileName != nil && !logPos.IsValid() {
        prompt(ctx, "%v:1: see logs for \"%s\"\n", opts.logFileName.string, target)
      }
      erro(ctx, `stamp "%v" failed`, target)
      errostack(ctx, 6, ``).debug(10)
      return
    } else if opts.report {
      var t = ctx.programContext()
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
        s string
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
      if n := exeres.Stdout.wrote; n > 0 { s += fmt.Sprintf(", stdout=%d bytes", n) }
      if n := exeres.Stderr.wrote; n > 0 { s += fmt.Sprintf(", stderr=%d bytes", n) }
      if t := programCtx.dirt; t != "" { s += "; " + t }
      prompt(ctx, "%s%s (%v%s)\n", ps, st, time.Now().Sub(start), s)
    }
  } ()

  var res []Value
  for i, src := range sources {
    var pos = positions[i]
    if strings.HasPrefix(src, "@") {
      src = src[1:]
    } else if opts.promptSrc && !opts.prompt {
      var s string = src
      s = strings.Replace(s, "\n", "\\n", -1)
      s = strings.Replace(s, "\\\\n", "\\\n", -1)
      prompt(ctx, "%s\n", s)//.debug(1)
    }
    if src = strings.TrimSpace(src); src == "" { continue }
    if false && !opts.noCD && workDir != "" {
      if strings.HasPrefix(src, "#") {
        src = fmt.Sprintf("cd '%s' %s", workDir, src)
      } else {
        // Insert a "\n" before the right paren ')' to ensure that
        // it's working with comments like "true #comment...".
        src = fmt.Sprintf("cd '%s' && (%s\n)", workDir, src)
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

    if false { for {
      if err = lockCD(workDir, 25*time.Millisecond); err != nil {
        erro(ctx, "%v", err).debug(1)
        return
      } else if s, _ := os.Getwd(); s == workDir { break }
    }}
    if !opts.silentErrs || opts.prompt || opts.promptSrc {
      exeres.printEnteringOnFirstWrote = true
    }

    exeres.container = container
    exeres.sh = exec.Command(cmd, aa...)
    exeres.sh.Dir = workDir // always set command work directory
    exeres.sh.Env = env
    exeres.sh.Stdout = &exeres.Stdout
    exeres.sh.Stderr = &exeres.Stderr
    if opts.stdin {
      exeres.sh.Stdin = os.Stdin
      exeres.sh.Args = append(exeres.sh.Args, "-ti")
    }
    if p.opt != "" { exeres.sh.Args = append(exeres.sh.Args, p.opt) }
    if src   != "" { exeres.sh.Args = append(exeres.sh.Args, src) }
    if opts.debug > 0 {
      warn(at(ctx,program.position), "%v: %v", ctx.entry(), target)
      warn(ctx, "context: %v", ctx.programContext())
      warn(ctx, "exec:\n%v", exeres.sh).debug(opts.debug*2)
    }

    exeres.Stdout.report = !opts.silentErrs
    exeres.Stderr.report = !opts.silentErrs
    exeres.Status, err = exeres.run(at(ctx, pos))
    if false { warn(ctx, "%v: %v --> %v", cmd, src, exeres.Status).debug(1) }
    if (!opts.silentErrs || opts.debug>0) && (len(exeres.scannedDiags) > 0 || exeres.Status != 0 || err != nil) {
      if opts.silentErrs || opts.retStatus {
        err = nil
      } else if exeres.Status != 0 {
        err = &exitstatus{ exeres.Status } // set or convert error
      }

      var en, wn, in int
      for _, rec := range exeres.scannedDiags {
        switch rec.dt {
        case diagError: en += rec.num
        case diagWarn:  wn += rec.num
        case diagInfo:  in += rec.num
        }
      }
      if en > 0 || exeres.Status != 0 || err != nil {
        prompt(ctx, "exec: failure (status=%d; err=%v); target=%s\n", exeres.Status, err, targetName)
      } else if wn > 0 {
        prompt(ctx, "%v: %d warnings\n", targetName, wn)
      }

      for i, rec := range exeres.scannedDiags {
        if !opts.infos && rec.dt == diagInfo { continue }
        if !logPos.IsValid() { logPos = rec.lpos }
        if i == 0 && !rec.position.Same(&rec.lpos) {
          diag(at(ctx,rec.lpos), rec.dt, rec.msg)//.debug(1)
        }
        if rec.num > 1 {
          diag(at(ctx,rec.position), rec.dt, `%s (%d)`, rec.msg, rec.num)//.debug(1)
        } else {
          diag(at(ctx,rec.position), rec.dt, rec.msg)//.debug(1)
        }
        if n := (en+wn+in)-(i+1); i == 8 && 0 < n {
          diag(at(ctx,rec.lpos), rec.dt, "%d more...", n)//.debug(1)
          break
        }
      }

      var pos = ctx.Position()
      if !logPos.IsValid() && log != nil {
        logPos.Filename = log.filename
        logPos.Line = exeres.Stderr.log.lines + 1
      } else {
        logPos = pos
      }

      var diffLogPos = !logPos.SameLine(&pos)
      var str, _, _ = entryStr(ctx, ctx.entry())
      if (!opts.retStatus && exeres.Status != 0) || en > 0 {
        if opts.dropFailed {
          if e := os.RemoveAll(targetName); e != nil {
            warn(ctx, "remove: %v", e).debug(1)
          }
        }
        if diffLogPos { erro(at(ctx,logPos), "%v: %d known errors", str, en) }
        erro(at(ctx,positions[i]), "%v: exit status %d (%d known errors)", str, exeres.Status, en)
        errostack(ctx, 32, "").debug(32)
      } else if wn > 0 {
        if diffLogPos { warn(at(ctx,logPos), "%v: %d known warnings", str, wn) }
        warn(at(ctx,positions[i]), "%v: exit status %d", str, exeres.Status)

        warn(ctx, "%v: %d known warnings", str, wn)
        warnstack(ctx, 3, "").debug(1)
      } else if in > 0 && opts.infos {
        if diffLogPos { info(at(ctx,logPos), "%v: %d known messages", str, in) }
        info(at(ctx,positions[i]), "%v: exit status %d", str, exeres.Status)
        info(ctx, "%v: %d known messages", str, in)
        infostack(ctx, 8, "").debug(1)
      }

      if opts.retStatus {
        if opts.zeroErrs && en == 0 && err == nil {
          res = append(res, MakeInt(logPos, int64(exeres.Status)))
        } else {
          res = append(res, MakeNone(logPos))
        }
      } else if exeres.Status != 0 || err != nil {
        break
      }
    }
  }

  // Add stdout result
  if opts.retStdout {
    var s string
    if exeres.Stdout.Buf != nil {
      s = exeres.Stdout.Buf.String()
    }
    res = append(res, MakeString(pos, s))
  }

  // Add stderr result
  if opts.retStderr {
    var s string
    if exeres.Stderr.Buf != nil {
      s = exeres.Stderr.Buf.String()
    }
    res = append(res, MakeString(pos, s))
  }

  // The execution is performed asynchronously, the result can't be fetched immediately.
  // Caller should do a t.wait(...) or exeres.wait() before using the result.
  if len(res) == 0 {
    result = exeres
  } else {
    result = MakeListOrScalar(pos, res)
  }
  return
}
