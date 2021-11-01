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
  exitstatusFmt = "exit status %d"
  maxPromptStr = 48
)

type exitstatus struct { code int }
func (e *exitstatus) Error() string { return fmt.Sprintf(exitstatusFmt, e.code) }

const (
  rxNotTTYDevice_i int = iota
  rxNoContainer_i
  rxNoNetwork_i
  rxDockerDaemonNotRunning_i
  rxContainerNotRunning_i
  rxCompilation_i
  rxIncludedFrom2_i
  rxIncludedFrom3_i
  rxFileNotFound_i
  rxArNoSuchFile_i
  rxArNoArchiveMembers_i
  rxBashNoSuchFile_i
  rxClangNoSuchFile_i
  rxClangError_i
  rxCmdError_i
  rxCmdWarning_i
  rxLdLibNotFound_i
  rxCouldnotParseObj_i
  rxTooManyPosArgs_i
  rxUndefinedReference_i
  rxShellCmdNotFound_i
  rxExitStatus_i
  rxIgnoringNonExistentDirectory_i
  rxIgnoringDuplicateDirectory_i
)
var (
  defaultShell = "bash"

  strErrorPreprocess = `#error (.+)`

  strNotTTYDevice = `the input device is not a TTY`
  strNoContainer = `Error.*: No such container: (.*)`
  strNoNetwork = `Error.*: network (.*) not found\.`
  strDockerDaemonNotRunning = `Cannot connect to the Docker daemon at (.*?)\. Is the docker daemon running\?`
  strContainerNotRunning = `Error response from daemon: Container (.*?) is not running`

  strCompilation = `(.+?):(\d+):(\d+): error: (.+)(?: {2,}\n(.+))?`
  strIncludedFrom2 = `In file included from (.+?):(\d+):`
  strIncludedFrom3 = `In file included from (.+?):(\d+):(\d+):`
  strFileNotFound = `(.+?):(\d+):(\d+): fatal error: '(.+?)' file not found`
  strArNoSuchFile = `ar: (.+?): No such file or directory`
  strArNoArchiveMembers = `ar: no archive members specified`
  strBashNoSuchFile = `bash: (.+?): No such file or directory`
  strClangNoSuchFile = `clang(?:-(.+?))?: error: no such file or directory: '(.+?)'`
  strClangError = `clang(?:-(.+?))?: error: (.+)(?: \(.+\))?`
  strCmdError = `(ld\.lld|ld64\.lld|lld-link|wasm-ld|ld|clang): error: (.+)`
  strCmdWarning = `(ld\.lld|ld64\.lld|lld-link|wasm-ld|ld|clang): warning: (.+)`
  strLdLibNotFound = `(ld\.lld|ld64\.lld|lld-link|wasm-ld|ld): library not found for (.+)`
  strCouldnotParseObj = `(ld\.lld|ld64\.lld|lld-link|wasm-ld|ld): could not parse object file (.+?): '(.+)', using libLTO version '(.+?)' file '(.+?)' for architecture (.+)`
  strTooManyPosArgs = `(.+?): Too many positional arguments specified!`
  strUndefinedReference = `  +"(.+?)", referenced from:`
  strShellCmdNotFound = `(.+?): (.+?):( command)? not found`
  strExitStatus = `exit status (\-?[0-9]+)`
  strIgnoringNonExistentDirectory = `ignoring nonexistent directory "(.*?)"`
  strIgnoringDuplicateDirectory = `ignoring duplicate directory "(.*?)"`

  rxNotTTYDevice = regexp.MustCompile(strNotTTYDevice)
  rxNoContainer = regexp.MustCompile(strNoContainer)
  rxNoNetwork = regexp.MustCompile(strNoNetwork)
  rxDockerDaemonNotRunning = regexp.MustCompile(strDockerDaemonNotRunning)
  rxContainerNotRunning = regexp.MustCompile(strContainerNotRunning)
  rxCompilation = regexp.MustCompile(strCompilation)
  rxIncludedFrom2 = regexp.MustCompile(strIncludedFrom2)
  rxIncludedFrom3 = regexp.MustCompile(strIncludedFrom3)
  rxFileNotFound = regexp.MustCompile(strFileNotFound)
  rxArNoSuchFile = regexp.MustCompile(strArNoSuchFile)
  rxArNoArchiveMembers = regexp.MustCompile(strArNoArchiveMembers)
  rxBashNoSuchFile = regexp.MustCompile(strBashNoSuchFile)
  rxClangNoSuchFile = regexp.MustCompile(strClangNoSuchFile)
  rxClangError = regexp.MustCompile(strClangError)
  rxCmdError = regexp.MustCompile(strCmdError)
  rxCmdWarning = regexp.MustCompile(strCmdWarning)
  rxLdLibNotFound = regexp.MustCompile(strLdLibNotFound)
  rxCouldnotParseObj = regexp.MustCompile(strCouldnotParseObj)
  rxTooManyPosArgs = regexp.MustCompile(strTooManyPosArgs)
  rxUndefinedReference = regexp.MustCompile(strUndefinedReference)
  rxShellCmdNotFound = regexp.MustCompile(strShellCmdNotFound)
  rxExitStatus = regexp.MustCompile(strExitStatus)
  rxIgnoringNonExistentDirectory = regexp.MustCompile(strIgnoringNonExistentDirectory)
  rxIgnoringDuplicateDirectory = regexp.MustCompile(strIgnoringDuplicateDirectory)

  knownerrors = []*regexp.Regexp{
    rxNotTTYDevice_i:           rxNotTTYDevice,
    rxNoContainer_i:            rxNoContainer,
    rxNoNetwork_i:              rxNoNetwork,
    rxCompilation_i:            rxCompilation,
    rxIncludedFrom2_i:          rxIncludedFrom2,
    rxIncludedFrom3_i:          rxIncludedFrom3,
    rxFileNotFound_i:           rxFileNotFound,
    rxArNoSuchFile_i:           rxArNoSuchFile,
    rxArNoArchiveMembers_i:     rxArNoArchiveMembers,
    rxBashNoSuchFile_i:         rxBashNoSuchFile,
    rxClangNoSuchFile_i:        rxClangNoSuchFile,
    rxClangError_i:             rxClangError,
    rxCmdError_i:               rxCmdError,
    rxCmdWarning_i:             rxCmdWarning,
    rxLdLibNotFound_i:          rxLdLibNotFound,
    rxDockerDaemonNotRunning_i: rxDockerDaemonNotRunning,
    rxContainerNotRunning_i:    rxContainerNotRunning,
    rxCouldnotParseObj_i:       rxCouldnotParseObj,
    rxTooManyPosArgs_i:         rxTooManyPosArgs,
    rxUndefinedReference_i:     rxUndefinedReference,
    rxShellCmdNotFound_i:       rxShellCmdNotFound,
    rxExitStatus_i:             rxExitStatus,
    rxIgnoringNonExistentDirectory_i: rxIgnoringNonExistentDirectory,
    rxIgnoringDuplicateDirectory_i: rxIgnoringDuplicateDirectory,
  }

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

func trimPromptString(str string) (s string) {
  var segs = strings.Split(str, PathSep)
  if len(segs) == 0 {
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
    if n > maxPromptStr {
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
  i, l int
  v [][]knownMatchCap // groups of captures
}

type ignoringDuplicateDirectoryCounter struct {
  position Position
  dir string
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
  includedFrom struct { pos1, pos2 Position }
}
func (p *ExecBuffer) filter(s string) { p.filters = append(p.filters, s) }
func (p *ExecBuffer) Write(b []byte) (n int, err error) {
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
      for i, rx := range knownerrors {
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
          if _, e := p.scan(p.res.position, &knownMatch{ i, l, a }); e != nil {
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
  )
  if p.log != nil { lpos.Filename = p.log.filename }
  if     m != nil { lpos.Line, lpos.Column = m.l, 0 }
  for _, v := range m.v { // captures
    if len(v) > 1 { lpos.Column = v[1].col }
    switch m.i {
    case rxNotTTYDevice_i:
      if p.report {
        erro(ctx, "Needs TTY (input device)").at(lpos).debug(1)
        p.res.errs += 1
      }
    case rxDockerDaemonNotRunning_i:
      if err = p.startDockerDaemon(lpos, ctx, container, v[1].string); err != nil {
        erro(ctx, "start container failed: %v", err).at(pos).debug(1)
        p.res.errs += 1
      }
    case rxNoContainer_i:
      if name := v[1].string; p.res.skips(name) {
        if p.report {
          erro(ctx, "container not running: %v", name).at(lpos).debug(1)
          p.res.errs += 1
        }
      } else {
        p.res.containerToRun = name
      }
    case rxContainerNotRunning_i:
      if p.report {
        erro(ctx, "Container not running (%v)", v[1].string).at(lpos).debug(1)
        p.res.errs += 1
      }
    case rxNoNetwork_i:
      if p.report {
        erro(ctx, "Network not found (%v)", v[1].string).at(lpos).debug(1)
        p.res.errs += 1
      }
    case rxIncludedFrom2_i:
      if p.report {
        lpos.Column = v[2].col + 1
        p.includedFrom.pos1 = convPosition(v[1].string, v[2].string, "1")
        p.includedFrom.pos2 = lpos
      }
    case rxIncludedFrom3_i:
      if p.report {
        lpos.Column = v[3].col + 1
        p.includedFrom.pos1 = convPosition(v[1].string, v[2].string, v[3].string)
        p.includedFrom.pos2 = lpos
      }
    case rxCompilation_i:
      if p.report {
        p.errorPos = convPosition(v[1].string, v[2].string, v[3].string)
        lpos.Column = v[4].col
        if s := v[5].string; s != "" {
          erro(ctx, "%s: %s", v[4].string, s).at(p.errorPos)
        } else {
          erro(ctx, "%s", v[4].string).at(p.errorPos)
        }
        if !reportIncludedFrom() { erro(ctx, "…reported here").at(lpos).debug(1) }
        p.res.errs += 1
      }
    case rxFileNotFound_i:
      if p.report {
        p.errorPos = convPosition(v[1].string, v[2].string, v[3].string)
        lpos.Column = v[4].col
        erro(ctx, "'%s' file not found", v[4].string).at(p.errorPos)
        if !reportIncludedFrom() { erro(ctx, "…reported here").at(lpos).debug(1) }
        p.res.errs += 1
      }
    case rxArNoSuchFile_i:
      if p.report {
        erro(ctx, "'%v' file not found (as '%s')", filepath.Base(v[1].string), v[1]).at(lpos).debug(1)
        p.res.errs += 1
      }
    case rxArNoArchiveMembers_i:
      if p.report {
        if true {
          var obj = closureResolveObject(ctx, lpos, "objects")
          erro(ctx, "%s", v[0].string).at(lpos)
          erro(ctx, "%s", obj).at(lpos)
          if !isNil(obj) {
            if val, _ := obj.expand(ctx.closure().programCtx(), expandPlainValue); !isNil(val) {
              erro(ctx, "%s -> %v", obj.Name(), val).at(lpos)
            }
          }
          erro(ctx, "%v", ctx).debug(16)
        } else {
          erro(ctx, "%s", v[0].string).at(lpos).debug(1)
        }
        p.res.errs += 1
      }
    case rxBashNoSuchFile_i:
      if p.report {
        erro(ctx, "%v: no such command", v[1].string).at(lpos).debug(1)
        p.res.errs += 1
      }
    case rxClangNoSuchFile_i:
      if p.report {
        var vs string
        if s := v[1].string; s != "" { vs = "-" + s }
        lpos.Column = v[2].col + 1
        erro(ctx, "clang%s: no such source file: %s", vs, v[2].string).at(lpos).debug(1)
        p.res.errs += 1
      }
    case rxClangError_i:
      if p.report {
        var vs string
        if s := v[1].string; s != "" { vs = "-" + s }
        lpos.Column = v[2].col + 1
        erro(ctx, "clang%s: %s", vs, v[2].string).at(lpos).debug(1)
        p.res.errs += 1
      }
    case rxCmdError_i:
      if p.report {
        lpos.Column = v[2].col + 1
        erro(ctx, "%s", v[2].string).at(lpos).debug(1)
        p.res.errs += 1
      }
    case rxCmdWarning_i:
      if p.report {
        lpos.Column = v[2].col + 1
        warn(ctx, "%s: %s", v[1].string, v[2].string).at(lpos).debug(1)
        p.res.warns += 1
      }
    case rxLdLibNotFound_i:
      if p.report {
        lpos.Column = v[2].col + 1
        if false {
          erro(ctx, "%s: library not found: %s", v[1].string, v[2].string).at(lpos).debug(1)
        } else {
          erro(ctx, "%s", v[0].string).at(lpos).debug(1)
        }
        p.res.errs += 1
      }
    case rxCouldnotParseObj_i:
      if p.report {
        lpos.Column = v[3].col
        erro(ctx, "%s", v[3].string).at(lpos).debug(1)
        p.res.errs += 1
      }
    case rxTooManyPosArgs_i:
      if p.report {
        erro(ctx, "%s: too many positional arguments", v[1].string).at(lpos).debug(1)
        p.res.errs += 1
      }
    case rxUndefinedReference_i:
      if p.report {
        erro(ctx, "Undefined reference '%s'", v[1].string).at(lpos).debug(1)
        p.res.errs += 1
      }
    case rxShellCmdNotFound_i:
      if p.report {
        lpos.Column = v[2].col
        erro(ctx, "%s: command not found", v[2].string).at(lpos).debug(1)
        p.res.errs += 1
      }
    case rxIgnoringNonExistentDirectory_i:
      if p.report {
        var dir = v[1].string
        lpos.Column = v[1].col + 1
        if true { info(ctx, "ignoring nonexistent directory: %s", dir).at(lpos).debug(1) }
      }
    case rxIgnoringDuplicateDirectory_i:
      if p.report {
        var ( dir = v[1].string; done bool )
        lpos.Column = v[1].col + 1
        if false { info(ctx, "ignoring duplicate directory: %s", dir).at(lpos).debug(1) }
        for _, rec := range p.res.ignoringDuplicateDirectory {
          if rec.dir == dir { rec.num += 1; done = true }
        }
        if !done {
          var rec = &ignoringDuplicateDirectoryCounter{ lpos, dir, 1 }
          p.res.ignoringDuplicateDirectory = append(p.res.ignoringDuplicateDirectory, rec)
        }
      }
    case rxExitStatus_i:
      if s := v[1].string; s != "0" /*&& p.report*/ {
        // FIXME: the 'exit status' report is not working
        erro(ctx, "abnormal exist status %s", s).at(lpos).debug(1)
        p.res.errs += 1
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

  errs, warns int

  num int
  ctx Context
  x *executor
  sh *exec.Cmd
  container *Project

  ignoringDuplicateDirectory []*ignoringDuplicateDirectoryCounter
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
  if run, _ := p.container.resolveEntry(ctx, "run", false); run != nil {
    if _, brks := run.execute(p.ctx); brks.has() {
      erro(ctx, "%d breakers", len(brks)).at(p.position).debug(1)
      return
    } //else { p.t.group.Wait() }
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
    if run, _ := p.container.resolveEntry(ctx, "run", false); run != nil {
      if _, brks := run.execute(p.ctx); brks.has() {
        erro(ctx, "%d breakers", len(brks)).at(p.position).debug(1)
        return
      } //else { p.t.group.Wait() }
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
    debug  bool "d,debug"
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
    erro(ctx, "parse opts failed: %v", err).debug(1)
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
  if targetName, err = fullnameOrStrval(ctx, target); err != nil {
    erro(ctx, "stringify target '%v' failed: %v", target, err).of(target).debug(1)
    return
  } else if t.configuration {
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
    warn(ctx, "add -stamp to (shell)").debug(1)
  }

  var start = time.Now()
  var exeres = &ExecResult{valbase:valbase{pos}, ctx:ctx, x:p}
  var aa []string
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

  ///////////
  if false {
    defer func() {
      info(ctx, "%v, status=%v", target, exeres.Status)
      info(ctx, "%v: %v", target, recipes)
      info(ctx, "%v: %v", target, sources).debug(6)
    } ()
  }

  var caller = ctx.traversal().caller()
  defer func() {
    if log.writer != nil { log.writer.Flush() }
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
    if opts.stamp && !t.configuration {
      var files []*File
      if files, err = target.stamp(t); err != nil {
        if pe, ok := err.(*fs.PathError); ok {
          erro(ctx, "stamp %v: not found", trimPromptString(pe.Path)).debug(6)
        } else {
          erro(ctx, "%v", err).debug(6)
        }
        return
      } else if opts.report {
        reportFileUpdates(ctx, t.start, files)
      }
    }

    if err == nil {
      // Good!
    } else if t.configuration {
      if false { info(ctx, "configure failed: %v", err).debug(1) }
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
      if false && ps == "" { ps = "exec: " }
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

    //if err = lockCD(dir, 25*time.Millisecond); err != nil { erro(ctx, "%v", err); return }
    //if s, e := os.Getwd(); e == nil { assert(s == dir, "wrong work directory (%s != %s)", s, dir) }
    for {
      if err = lockCD(dir, 25*time.Millisecond); err != nil {
        erro(ctx, "%v", err).debug(1)
        return
      } else if s, _ := os.Getwd(); s == dir { break }
    }
    if !opts.silent || opts.prompt || opts.verboseSrc {
      printEnteringDirectory(t)
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
    if exeres.Status, err = exeres.run(positional(ctx, pos)); err != nil {
      if !opts.silent || opts.debug {
        if exeres.Stderr.log != nil {
          var lpos Position
          lpos.Filename = log.filename
          lpos.Line = exeres.Stderr.log.lines

          erro(ctx, "%v: %s", target, err).at(lpos)
          erro(ctx, "%v", ctx.Project()).at(ctx.Project().position)
          erro(ctx, "%v: %T %s", target, target, targetName)
          if caller := ctx.traversal().caller(); caller != nil {
            var targetStr, _ = fullnameOrStrval(caller, target)
            erro(ctx, "%v: %T %s", target, target, targetStr)
            erro(ctx, "%v", target).of(target)
            erro(ctx, "%v: %s", target, ctx)
            erro(caller, "%v: %s", target, caller).debug(42)
          } else {
            var targetStr, _ = fullnameOrStrval(ctx.closure(), target)
            erro(ctx, "%v: %T %s", target, target, targetStr)
            erro(ctx, "%v", target).of(target)
            erro(ctx, "%v: %s", target, ctx).debug(24)
          }
        }
        if exeres.errs > 0 { exeres.errs += 1 // err != nil
          if cc, _ := t.inner().(*closureContext); cc != nil {
            if exeres.warns > 0 {
              erro(ctx, "exec: got %d errors and %d warnings", exeres.errs, exeres.warns)
            } else {
              erro(ctx, "exec: got %d errors", exeres.errs)
            }
            erro(ctx, "%v", t).at(t.Position())
            if caller != nil {
              erro(ctx, "%v", cc).at(cc.Position())
              erro(ctx, "%v", caller).at(caller.Position()).debug(16)
            } else {
              erro(ctx, "%v", cc).at(cc.Position()).debug(16)
            }
          } else {
            erro(ctx, "… requested here").debug(16)
          }
        } else if exeres.warns > 0 {
          if cc, _ := t.inner().(*closureContext); cc != nil {
            warn(t, "exec: got %d warnings", exeres.warns)
            warn(t, "… requested here").at(cc.Position()).debug(16)
          } else {
            warn(t, "… requested here").debug(16)
          }
        } else if cc, _ := t.inner().(*closureContext); cc != nil {
          erro(ctx, "%v: %v (%T, status=%v)", target, err, err, exeres.Status)
          erro(ctx, "… requested here").at(cc.Position()).debug(16)
        } else if caller != nil {
          erro(ctx, "%v: %v (%T, status=%v)", target, err, err, exeres.Status)
          erro(ctx, "… requested here").at(caller.Position()).debug(16)
        } else {
          erro(ctx, "%v: %v (%T, status=%v)", target, err, err, exeres.Status).debug(16)
        }
      }
    }
    if len(exeres.ignoringDuplicateDirectory) > 0 {
      var idds int
      for _, rec := range exeres.ignoringDuplicateDirectory {                idds += rec.num
        info(ctx, `ignoring duplicate directory "%v" (%d)`, rec.dir, rec.num).at(rec.position)
      }
      if idds > 0 && !opts.silent {
        if true {
          info(ctx, "%v: ignoring duplicate directories: %v", target, idds).debug(1)
        } else {
          var t, _ = ctx.autoGet("@")
          info(ctx, "%v: ignoring duplicate directories: %v", target, idds)
          info(ctx, "%v: %v", target, t)
          info(ctx, "%v: %v", target, ctx).debug(16)
        }
      }
      if opts.silent { err = nil }
    }
    if exeres.Status != 0 && (!opts.silent || opts.debug) {
      erro(ctx, "abnormal exec exit status %d", exeres.Status).debug(1)
      err = &exitstatus{ exeres.Status } // convert to exitstatus
      break
    }
  }

  // The execution is performed asynchronously, the result can't
  // be fetched immediately. Caller should do a t.wait(...) or
  // exeres.wait() before using the result.
  result = exeres
  return
}
