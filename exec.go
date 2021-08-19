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
  rxIncludedFrom_i
  rxFileNotFound_i
  rxArNoSuchFile_i
  rxBashNoSuchFile_i
  rxClangNoSuchFile_i
  rxClangError_i
  rxLLDError_i
  rxLLDWarning_i
  rxCouldnotParseObj_i
  rxTooManyPosArgs_i
  rxUndefinedReference_i
  rxShellCmdNotFound_i
  rxExitStatus_i
)
var (
  defaultShell = "bash"

  errErrorPreprocess = `#error (.+)`

  errNotTTYDevice = `the input device is not a TTY`
  errNoContainer = `Error.*: No such container: (.*)`
  errNoNetwork = `Error.*: network (.*) not found\.`
  errDockerDaemonNotRunning = `Cannot connect to the Docker daemon at (.*?)\. Is the docker daemon running\?`
  errContainerNotRunning = `Error response from daemon: Container (.*?) is not running`

  errCompilation = `(.+?):(\d+):(\d+): error: (.+)`
  errIncludedFrom = `In file included from (.+?):(\d+):(\d+):`
  errFileNotFound = `(.+?):(\d+):(\d+): fatal error: '(.+?)' file not found`
  errArNoSuchFile = `ar: (.+?): No such file or directory`
  errBashNoSuchFile = `bash: (.+?): No such file or directory`
  errClangNoSuchFile = `clang-(.+?): error: no such file or directory: '(.+?)'`
  errClangError = `clang-(.+?): error: (.+)`
  errLLDError = `(ld\.lld|ld64\.lld|lld-link|wasm-ld|ld): error: (.+)`
  errLLDWarning = `(ld\.lld|ld64\.lld|lld-link|wasm-ld|ld): warning: (.+)`
  errCouldnotParseObj = `(ld\.lld|ld64\.lld|lld-link|wasm-ld|ld): could not parse object file (.+?): '(.+)', using libLTO version '(.+?)' file '(.+?)' for architecture (.+)`
  errTooManyPosArgs = `(.+?): Too many positional arguments specified!`
  errUndefinedReference = `  +"(.+?)", referenced from:`
  errShellCmdNotFound = `(.+?): (.+?):( command)? not found`
  errExitStatus = `exit status (\-?[0-9]+)`

  rxNotTTYDevice = regexp.MustCompile(errNotTTYDevice)
  rxNoContainer = regexp.MustCompile(errNoContainer)
  rxNoNetwork = regexp.MustCompile(errNoNetwork)
  rxDockerDaemonNotRunning = regexp.MustCompile(errDockerDaemonNotRunning)
  rxContainerNotRunning = regexp.MustCompile(errContainerNotRunning)
  rxCompilation = regexp.MustCompile(errCompilation)
  rxIncludedFrom = regexp.MustCompile(errIncludedFrom)
  rxFileNotFound = regexp.MustCompile(errFileNotFound)
  rxArNoSuchFile = regexp.MustCompile(errArNoSuchFile)
  rxBashNoSuchFile = regexp.MustCompile(errBashNoSuchFile)
  rxClangNoSuchFile = regexp.MustCompile(errClangNoSuchFile)
  rxClangError = regexp.MustCompile(errClangError)
  rxLLDError = regexp.MustCompile(errLLDError)
  rxLLDWarning = regexp.MustCompile(errLLDWarning)
  rxCouldnotParseObj = regexp.MustCompile(errCouldnotParseObj)
  rxTooManyPosArgs = regexp.MustCompile(errTooManyPosArgs)
  rxUndefinedReference = regexp.MustCompile(errUndefinedReference)
  rxShellCmdNotFound = regexp.MustCompile(errShellCmdNotFound)
  rxExitStatus = regexp.MustCompile(errExitStatus)

  knownerrors = []*regexp.Regexp{
    rxNotTTYDevice_i:           rxNotTTYDevice,
    rxNoContainer_i:            rxNoContainer,
    rxNoNetwork_i:              rxNoNetwork,
    rxCompilation_i:            rxCompilation,
    rxIncludedFrom_i:           rxIncludedFrom,
    rxFileNotFound_i:           rxFileNotFound,
    rxArNoSuchFile_i:           rxArNoSuchFile,
    rxBashNoSuchFile_i:         rxBashNoSuchFile,
    rxClangNoSuchFile_i:        rxClangNoSuchFile,
    rxClangError_i:             rxClangError,
    rxLLDError_i:               rxLLDError,
    rxLLDWarning_i:             rxLLDWarning,
    rxDockerDaemonNotRunning_i: rxDockerDaemonNotRunning,
    rxContainerNotRunning_i:    rxContainerNotRunning,
    rxCouldnotParseObj_i:       rxCouldnotParseObj,
    rxTooManyPosArgs_i:         rxTooManyPosArgs,
    rxUndefinedReference_i:     rxUndefinedReference,
    rxShellCmdNotFound_i:       rxShellCmdNotFound,
    rxExitStatus_i:             rxExitStatus,
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

type ExecBuffer struct {
  Tie io.Writer
  Buf *bytes.Buffer
  log *ExecLog
  scanerr bool
  line bytes.Buffer
  matches []knownMatch
  filters []string
  wrote uint64
  retried map[string]bool
  report bool
}

func (p *ExecBuffer) filter(s string) { p.filters = append(p.filters, s) }
func (p *ExecBuffer) Write(b []byte) (n int, err error) {
  for _, s := range p.filters {
    if bytes.Equal(b, []byte(s)) { // string(b) == s
      return len(b), nil
    }
  }

  var l int
  if p.log != nil {
    l = p.log.lines // get lines before writing new bytes
    if _, err = p.log.Write(b); err != nil {
      return
    }
  }
  if p.Buf != nil {
    if n, err = p.Buf.Write(b); err != nil {
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

  if !p.scanerr { return }

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
          p.matches = append(p.matches, knownMatch{ i, l, a })
        }
      }

      p.line.Reset()
    }
  }
  return
}

func (p *ExecBuffer) skips(tag string) bool {
  if p.retried == nil { p.retried = make(map[string]bool) }
  var a, b = p.retried[tag]
  return a && b
}

func (p *ExecBuffer) startDockerDaemon(pos Position, t *traversal, container *Project, sock string) (err error) {
  var c = exec.Command("dockerd")
  //c.Stdout, c.Stderr = stdout, stderr
  if err = c.Run(); err != nil {
    if p.report { diag.errorAt(pos, "dokcer daemon not running (at %s)", sock).debug(optionDebugErrors,1) }
  } else {
    // TODO: start docker daemon
  }
  return
}

func (p *ExecBuffer) runContainerAndRetry(pos Position, t *traversal, container *Project, name string, sh *exec.Cmd, x *executor, num int) (status int, err error) {
  if container != nil && num <= maxRetries {
    fmt.Fprintf(sh.Stderr, "\n---- Run the container: %s\n", name)
    if x.runContainer(t, container); t.hasBreakers() {
      if p.report { diag.errorAt(pos, "container not running: %v", name).debug(optionDebugErrors,1) }
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

    c := exec.Command(sh.Path, sh.Args[1:]...) // must ignore Args[0]
    c.Stdout, c.Stderr, c.Stdin, c.Env = sh.Stdout, sh.Stderr, sh.Stdin, sh.Env
    if false {
      fmt.Fprintf(sh.Stderr, "\n  %s", sh)
      fmt.Fprintf(sh.Stderr, "\n  %s", c)
      fmt.Fprintf(sh.Stderr, "\n----\n")
    }

    status, err = p.runWithErrorsFilter(pos, t, container, c, x, num+1)
    if status != 0 && err == nil { err = &exitstatus{status} }
    if err != nil { fmt.Fprintf(sh.Stderr, "\n---- Retry failed: %s\n", err) }
  }
  return
}

func (p *ExecBuffer) knownError(pos Position, t *traversal, container *Project, sh *exec.Cmd, x *executor, num int, m *knownMatch) (status int, err error) {
  if p == nil {
    diag.errorAt(pos, "nil exec buffer").
      debug(optionDebugErrors,1)
    return
  }
  var lpos Position = pos
  if p.log != nil { lpos.Filename = p.log.filename }
  if     m != nil { lpos.Line, lpos.Column = m.l, 0 }
  for _, v := range m.v { // captures
    if len(v) > 1 { lpos.Column = v[1].col }
    switch m.i {
    case rxNotTTYDevice_i:
      if p.report {
        diag.errorAt(lpos, "Needs TTY (input device)").
          debug(optionDebugErrors,1)
      }
    case rxDockerDaemonNotRunning_i:
      err = p.startDockerDaemon(lpos, t, container, v[1].string)
      if err != nil {
        diag.errorAt(pos, "start container failed: %v", err).
          debug(optionDebugErrors,1)
      }
    case rxNoContainer_i:
      if name := v[1].string; p.skips(name) {
        if p.report {
          diag.errorAt(lpos, "container not running: %v", name).
            debug(optionDebugErrors,1)
        }
      } else if status, err = p.runContainerAndRetry(lpos, t, container, name, sh, x, num); err == nil {
        p.retried[name] = true // save it to skip next time
        break // discard the rest errors
      }
    case rxContainerNotRunning_i:
      if p.report {
        diag.errorAt(lpos, "Container not running (%v)", v[1].string).
          debug(optionDebugErrors,1)
      }
    case rxNoNetwork_i:
      if p.report {
        diag.errorAt(lpos, "Network not found (%v)", v[1].string).
          debug(optionDebugErrors,1)
      }
    case rxCompilation_i:
      if p.report {
        var pos Position = convPosition(v[1].string, v[2].string, v[3].string)
        lpos.Column = v[4].col
        diag.errorAt(pos, "%s", v[4].string)
        diag.errorAt(lpos, "…from here").
          debug(optionDebugErrors,1)
      }
    case rxIncludedFrom_i:
      if p.report {
        var pos Position = convPosition(v[1].string, v[2].string, v[3].string)
        lpos.Column = v[3].col + 1
        diag.errorAt(pos, "included error")
        diag.errorAt(lpos, "…from here").
          debug(optionDebugErrors,1)
      }
    case rxFileNotFound_i:
      if p.report {
        var pos Position = convPosition(v[1].string, v[2].string, v[3].string)
        lpos.Column = v[4].col
        diag.errorAt(pos, "'%s' file not found", v[4].string)
        diag.errorAt(lpos, "…from here").
          debug(optionDebugErrors,1)
      }
    case rxArNoSuchFile_i:
      if p.report {
        diag.errorAt(lpos, "'%v' file not found (as '%s')", filepath.Base(v[1].string), v[1]).
          debug(optionDebugErrors,1)
      }
    case rxBashNoSuchFile_i:
      if p.report {
        diag.errorAt(lpos, "%v: no such command", v[1].string).
          debug(optionDebugErrors,1)
      }
    case rxClangNoSuchFile_i:
      if p.report {
        lpos.Column = v[2].col
        diag.errorAt(lpos, "clang-%s: no such source file: %s", v[1].string, v[2].string).
          debug(optionDebugErrors,1)
      }
    case rxClangError_i:
      if p.report {
        lpos.Column = v[2].col
        diag.errorAt(lpos, "clang-%s: %s", v[1].string, v[2].string).
          debug(optionDebugErrors,1)
      }
    case rxLLDError_i:
      if p.report {
        lpos.Column = v[2].col
        diag.errorAt(lpos, "%s", v[2].string).
          debug(optionDebugErrors,1)
      }
    case rxLLDWarning_i:
      if p.report {
        lpos.Column = v[2].col
        diag.warnAt(pos, "%s", v[2].string)
        diag.warnAt(lpos, "…from here").
          debug(optionDebugErrors,1)
      }
    case rxCouldnotParseObj_i:
      if p.report {
        lpos.Column = v[3].col
        diag.errorAt(lpos, "%s", v[3].string).
          debug(optionDebugErrors,1)
      }
    case rxTooManyPosArgs_i:
      if p.report {
        diag.errorAt(lpos, "%s: too many positional arguments", v[1].string).
          debug(optionDebugErrors,1)
      }
    case rxUndefinedReference_i:
      if p.report {
        diag.errorAt(lpos, "Undefined reference '%s'", v[1].string).
          debug(optionDebugErrors,1)
      }
    case rxShellCmdNotFound_i:
      if p.report {
        lpos.Column = v[2].col
        diag.errorAt(lpos, "%s: command not found", v[2].string).
          debug(optionDebugErrors,1)
      }
    case rxExitStatus_i:
      if s := v[1].string; s != "0" /*&& p.report*/ {
        // FIXME: the 'exit status' report is not working
        diag.errorAt(lpos, "abnormal exist status %s", s).
          debug(optionDebugErrors,1)
      }
    }
    if err != nil { break }
  }
  return
}

func (p *ExecBuffer) knownErrors(pos Position, t *traversal, container *Project, sh *exec.Cmd, x *executor, num int) (status int, err error) {
  for _, m := range p.matches {
    if status, err = p.knownError(pos, t, container, sh, x, num, &m); err != nil {
      diag.errorAt(pos, "%v (status=%d)", err, status).
        debug(optionDebugErrors, 1)
      break
    }
  }
  if err == nil && status != 0 { err = &exitstatus{ status }}
  return
}

func (p *ExecBuffer) runWithErrorsFilter(pos Position, t *traversal, dock *Project, sh *exec.Cmd, x *executor, num int) (status int, err error) {
  defer func(m []knownMatch) { p.matches = m } (p.matches)
  p.matches = nil // clear previous matches
  if err = sh.Run(); err == nil { return } else
  if n, e := fmt.Sscanf(err.Error(), exitstatusFmt, &status); n == 1 && e == nil {
    es := &exitstatus{ status } // convert to exitstatus
    err = es

    if p.log != nil && p.log.writer != nil {
      fmt.Fprintf(p.log, "\n%s\n", err)
   }

    p.retried = nil
    status, e = p.knownErrors(pos, t, dock, sh, x, num)
    if p.retried != nil && len(p.retried) > 0 {
      if e != nil {
        diag.errorAt(pos, "process known errors failed: %v", e).
          debug(optionDebugErrors,1)
      } else if status == 0 {
        err = nil
      } else {
        es.code = status
      }
    } else { status = es.code }

    if p.report {
      var pos Position
      pos.Filename = p.log.filename
      pos.Line = p.log.lines
      pos.Column = 0
      pos.Offset = 0 // FIXME: what should be the offset?
      diag.errorAt(pos, "%v", es)
    }
  } else {
    if status == 0 { status = -1 }
    if e != nil { err = e }
  }
  return
}

type ExecResult struct {
  valbase
  wg *sync.WaitGroup
  Stdout ExecBuffer
  Stderr ExecBuffer
  Status int
}
func (p *ExecResult) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *ExecResult) cmp(v Value) (res cmpres) {
  if a, ok := v.(*ExecResult); ok {
    assert(ok, "value is not ExecResult")
    if p.Status == a.Status { res = cmpEqual }
  }
  return
}
func (p *ExecResult) True() (res bool, err error) {
  res = p.Status == 0 && p.Stderr.Buf != nil && p.Stderr.Buf.Len() == 0 /* && p.Stdout.Buf.Len() > 0 */
  return
}
func (p *ExecResult) Integer() (int64, error) { return int64(p.Status), nil }
func (p *ExecResult) Float() (float64, error) { return float64(p.Status), nil }
func (p *ExecResult) Strval() (s string, err error) {
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

type executor struct {
  cmd, opt string
  contained bool
}

func (p *executor) runContainer(t *traversal, container *Project) {
  if run, _ := container.resolveEntry("run"); run != nil && len(run.programs) > 0 {
    defer setclosure(setclosure(cloctx.unshift(container.scope)))
    if run.programs[0].execute(t, run, nil); t.hasBreakers() {
      diag.errorAt(t.program.position, "%v", t.breakers)
    } else { t.group.Wait() }
  } else {
    diag.errorAt(t.program.position, "%s⇒run undefined", container)
  }
  return
}

func (p *executor) ensureContainerRunning(t *traversal, container *Project, containerName string) (err error) {
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
  }(stdoutR)

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
    if p.runContainer(t, container); t.hasBreakers() {
      time.Sleep(time.Second)
    }
  } else if err != nil {
    diag.errorAt(container.position, "%v", err)
  }
  return
}

type executorEvaluateOpts struct {
  debug bool "d,debug"
  prompt bool
  promStr string "c,cmd;m,prompt"
  verbout bool "v,verbout"
  verberr bool "w,verberr"
  buffOut bool "o,stdout"
  buffErr bool "e,stderr"
  stdin bool "i,stdin"
  silent bool "s,silent"
  stamp bool `st,stamp;sf,stamp-file`
  report bool `r,report;rs,report-stamp`
  noCD bool "n,nocd"
  path bool "p,path"
  scanStderr bool "a,scan"
  dump string "d,dump"
  logFileName *optFullname "l,log"
}
func (p *executor) Evaluate(pos Position, t *traversal, args ...Value) (result Value, err error) {
  if optionTraceExecutor {
    var t = t.def.target.value
    defer un(trace(t_exec, fmt.Sprintf("executor(%s %v)", typeof(t), t)))
  }

  var cmd = p.cmd
  var opts = executorEvaluateOpts{ scanStderr: true }
  if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
  if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }
  if opts.promStr != "" { opts.prompt = true }
  switch opts.dump {
  case "stdout": opts.verbout = true
  case "stderr": opts.verberr = true
  case "all":
    opts.verbout = true
    opts.verberr = true
  }

  var aa []string
  for i, v := range args {
    var s string
    if p.contained && i == 0 {
      if s, err = v.Strval(); err != nil { diag.errorOf(v, "%v", err); return }
      if s == "shell" { cmd = defaultShell }
      continue
    }
    if s, err = v.Strval(); err != nil { diag.errorOf(v, "%v", err); return } else
    if s = strings.TrimSpace(s); s != "" { aa = append(aa, s) }
  }

  var container *Project
  if p.contained {
    if t.program.project.name == dotContainer {
      container = t.program.project
    } else if false {
      for _, scope := range cloctx {
        if _, sym := scope.Find(dotContainer); sym != nil {
          if p, ok := sym.(*ProjectName); ok && p != nil {
            container = p.NamedProject()
            break
          }
        }
      }
      if container == nil {
        if _, containerSym := t.program.project.scope.Find(dotContainer); containerSym != nil {
          if pn, _ := containerSym.(*ProjectName); pn != nil {
            container = pn.NamedProject()
          }
        }
      }
    } else if _, containerSym := t.program.project.scope.Find(dotContainer); containerSym != nil {
      if pn, _ := containerSym.(*ProjectName); pn != nil {
        container = pn.NamedProject()
      }
    }

    if container == nil {
      diag.errorAt(pos, "container unavailable (in %s)", t.program.Project().Name())
      return
    }

    var strval = func(name string) (str string, err error) {
      if false {
        defer setclosure(scoping(container))
      } else {
        defer setclosure(cloctx)
        cloctx = append(closurecontext{container.Scope()}, cloctx...)
      }
      if obj, _ := container.resolveObject(name); obj != nil {
        if def, _ := obj.(*Def); def != nil {
          var v Value
          if v, err = def.DiscloseValue(); err == nil && v != nil {
            if str, err = v.Strval(); str == "-" {
              /*if v, err = def.DiscloseValue(container); err == nil && v != nil {
                                                        if str, err = v.Strval(); str == "" { str = "-" }
                                                        fmt.Fprintf(stderr, "%v: %v (%v)\n", name, str, def)
                                                        }*/
            }
          }
        }
      }
      return
    }

    var containerName, containerImage string
    if containerName , err = strval("container"); err != nil { diag.errorAt(pos, "%v", err); return }
    if containerName  == "" { diag.errorAt(pos, ".container.name undefined") ; return }
    if containerImage, err = strval("image")    ; err != nil { diag.errorAt(pos, "%v", err); return }
    if containerImage == "" { diag.errorAt(pos, ".container.image undefined"); return }
    if options.verbose { fmt.Fprintf(stderr, "%v: container=%v, image=%v\n", container, containerName, containerImage) }

    aa = append(aa, "exec", containerName, cmd)
    cmd = "docker"
  }

  var cwd string
  if v := t.program.scope.Lookup("CWD").(*Def).Call(t.program.position); v != nil { if cwd, err = v.Strval(); err != nil { return }} else
  if v := t.program.scope.Lookup("/"  ).(*Def).Call(t.program.position); v != nil { if cwd, err = v.Strval(); err != nil { return }}

  // Fixes work directory conflicts. It happens
  // sometimes even the 'sh.Dir' is set to cwd.
  // Because the current work directory is not
  // thread safe.
  var dir = cwd
  if t.program.changedWD != "" {
    if filepath.IsAbs(t.program.changedWD) {
      dir = t.program.changedWD
    } else {
      dir = filepath.Join(t.program.project.absPath, t.program.changedWD)
    }
  }

  var targetName string
  var target = t.getCurrentTargetValue() //t.def.target.value
  if targetName, err = fullnameOrStrval(target); err != nil {
    diag.errorOf(target, "stringify target '%v' failed: %v", target, err).
      debug(optionDebugErrors, 1)
    return
  }
  if opts.path {
    var s string
    if s = filepath.Dir(targetName); s != "" && s != "." && s != "/" {
      if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
        diag.errorOf(target, "make path '%s' for target failed: %v", s, err).
          debug(optionDebugErrors, 1)
        return
      }
    }
  }

  if t.isConfigureExecution {
    // does nothing
  } else if ms := t.program.getModifies("stamp"); len(ms) > 0 {
    switch target.(type) {
    case *Barefile, *File, *Path:
      diag.warnAt(ms[0].position, "use (shell -stamp) instead of stamp modifier (%T %v)", target, target).
        debug(optionDebugErrors, 1)
    }
  } else if !opts.stamp && !opts.silent {
    diag.warnAt(pos, "add -stamp to (shell)").
      debug(optionDebugErrors, 1)
  }

  var envars []*Pair // disclosed values
  if def, _ := t.program.scope.Lookup(TheShellEnvarsDef).(*Def); def != nil {
    if l, _ := def.value.(*List); l != nil {
      for _, v := range l.Elems {
        var t Value
        if t, err = v.expand(expandClosure); err != nil {
          diag.errorOf(v, "expand value '%v' failed: %v", v, err).
            debug(optionDebugErrors, 1)
          return
        } else if isNil(t) { t = v }
        if p, ok := t.(*Pair); ok {
          envars = append(envars, p)
        } else {
          diag.errorOf(t, "env expecting pairs: %T", t).
            debug(optionDebugErrors, 1)
          return
        }
      }
    }
  }

  var (
    recipes []Value
    source, str string
    sources []string
    positions []Position
    rp Position
  )
  if recipes, err = mergeresult(ExpandAll(t.program.recipes...)); err != nil {
    diag.errorAt(pos, "%v", err)
    return
  }
  for _, recipe := range recipes {
    if !rp.IsValid() { rp = recipe.Position() }
    if str, err = recipe.Strval(); err != nil { diag.errorOf(recipe, "%v", err); return }
    if source += str; strings.HasSuffix(source, "\\") {
      source += "\n" // append the line feed
      continue
    }

    // Escape '$$' sequences.
    source = strings.Replace(source, "$$", "$", -1)

    // Remove tabs in line breakings.
    source = strings.Replace(source, "\\\n\t", "\\\n", -1)

    // Duplicates all %
    //source = strings.Replace(source, "%", "%%", -1)

    positions = append(positions, rp)
    sources = append(sources, source)
    source = ""
    rp = Position{}
  }

  var envstr string
  var envs []string = os.Environ()
  for i, p := range envars {
    var k, v string
    if k, err = p.Key.Strval()  ; err != nil { diag.errorOf(p.Key  , "%v", err); return }
    if v, err = p.Value.Strval(); err != nil { diag.errorOf(p.Value, "%v", err); return }
    if i > 0 { envstr += " && " }
    envstr += fmt.Sprintf(`%s=%s`, k, strconv.Quote(v))
    envs = append(envs, fmt.Sprintf("%s=%s", k, v))
  }

  var logFile *os.File
  var log = &ExecLog{ filename: opts.logFileName.string }
  var exeres = &ExecResult{valbase:valbase{pos}, wg:new(sync.WaitGroup)}
  if opts.buffOut { exeres.Stdout.Buf = new(bytes.Buffer) }
  if opts.buffErr { exeres.Stderr.Buf = new(bytes.Buffer) }
  if opts.verbout { exeres.Stdout.Tie = stdout }
  if opts.verberr { exeres.Stderr.Tie = stderr }
  if log.filename == "" {
    // no log required
  } else if err = os.MkdirAll(filepath.Dir(log.filename), os.FileMode(0755)); err != nil {
    diag.errorAt(t.program.position, "%v", err)
    return
  } else if logFile, err = os.Create(log.filename); err != nil {
    diag.errorAt(t.program.position, "%v", err)
    return
  } else {
    cmdline := strings.Join(sources, "\n")
    log.createWriter(logFile, dir, cmdline)
    exeres.Stdout.log = log
    exeres.Stderr.log = log
  }

  exeres.Stderr.scanerr = opts.scanStderr

  var run = func() {
    var targetStr string

    defer func() {
      if log.writer != nil {
        if false && exeres.Stdout.wrote == 0 && exeres.Stderr.wrote == 0 {
          // Discard empty log buffer.
          logFile.Close()
          os.Remove(log.filename)
        } else {
          log.writer.Flush()
          logFile.Close()
        }
      }
      if t.caller != nil { t.caller.calleeDone(err) }
      exeres.wg.Done()
    } ()

    defer func() {
      if opts.stamp && !t.isConfigureExecution {
        var files []*File
        if files, err = target.stamp(t); err != nil {
          var p = target.Position()
          if !p.IsValid() { p = pos }
          if pe, ok := err.(*fs.PathError); ok {
            diag.errorAt(pos, "stamp %v: not found", trimPromptString(pe.Path)).
              debug(optionDebugErrors,1)
          } else {
            diag.errorAt(pos, "%v", err).
              debug(optionDebugErrors,1)
          }
          return
        } else if opts.report {
          reportFileUpdates(pos, t.start, files)
        }
      }
      if t.isConfigureExecution && err != nil {
        if false { diag.infoAt(pos, "configure exec failed: %v", err) }
        err = nil
      } else if err != nil {
        diag.errorAt(pos, "shell: %v", err).
          debug(optionDebugErrors,1)
        return
      }
      if opts.prompt {
        if t.caller == nil {
          if err == nil {
            fmt.Fprintf(stderr, "… ok\n")
          } else if _, ok := err.(*scanner.Error); ok {
            fmt.Fprintf(stderr, " error:\n%v\n", err)
          } else {
            fmt.Fprintf(stderr, " error: %v\n", err)
          }
        } else if false {
          if !strings.HasSuffix(opts.promStr, ":") { opts.promStr += ": " }
          if err == nil {
            if false { fmt.Fprintf(stderr, "%s%s, okay.\n", opts.promStr, targetStr) }
          } else if _, ok := err.(*scanner.Error); ok {
            fmt.Fprintf(stderr, "%s%s, error:\n%v\n", opts.promStr, targetStr, err)
          } else {
            fmt.Fprintf(stderr, "%s%s, error: %v\n", opts.promStr, targetStr, err)
          }
        }
      }
    } ()

    if n := diag.checkErrors(true); n > 0 {
      diag.warnAt(pos, "got %d error(s), cancel execution for %s",
        n, trimPromptString(targetName)).debug(optionDebugErrors, 1)
      return
    }

    if opts.prompt {
      targetStr = trimPromptString(targetName)
      if opts.promStr == "" {
        opts.promStr = "smart: gen "
      } else {
        opts.promStr += ": "
      }
      if t.caller == nil {
        fmt.Fprintf(stderr, "%s%s …\n", opts.promStr, targetStr)
      } else { // ……
        fmt.Fprintf(stderr, "%s%s\n", opts.promStr, targetStr)
      }
    }
    for i, src := range sources {
      var pos = positions[i]
      if false { fmt.Fprintf(stderr, "%s: %v\n", pos, src) }
      if strings.HasPrefix(src, "@") {
        src = src[1:]
      } else if !opts.prompt {
        var s string
        s = strings.Replace(src, "\n", "\\n", -1)
        s = strings.Replace(s, "\\\\n", "\\\n", -1)
        fmt.Fprintf(stderr, "%s\n", s)
      }
      if src = strings.TrimSpace(src); src == "" { continue } else
      if dir != "" && !opts.noCD /*&& t.program.changedWD == ""*/ {
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

      //if err = lockCD(dir, 25*time.Millisecond); err != nil { diag.errorAt(pos, "%v", err); return }
      //if s, e := os.Getwd(); e == nil { assert(s == dir, "wrong work directory (%s != %s)", s, dir) }
      for {
        if err = lockCD(dir, 25*time.Millisecond); err != nil {
          diag.errorAt(pos, "%v", err).debug(optionDebugErrors,1)
          return
        }
        if s, _ := os.Getwd(); s == dir { break }
      }

      var sh = exec.Command(cmd, aa...)
      sh.Dir = dir // always set command work directory
      sh.Env = envs
      sh.Stdout = &exeres.Stdout
      sh.Stderr = &exeres.Stderr
      if opts.stdin {
        sh.Stdin = os.Stdin
        sh.Args = append(sh.Args, "-ti")
      }
      if p.opt != "" { sh.Args = append(sh.Args, p.opt) }
      if src   != "" { sh.Args = append(sh.Args, src) }

      if opts.debug { diag.warnAt(pos, "%v", sh).debug(optionDebugErrors, 1) }

      exeres.Stderr.report = !opts.silent
      exeres.Status, err = exeres.Stderr.runWithErrorsFilter(pos, t, container, sh, p, 1)
      if err != nil {
        if !opts.silent || opts.debug {
          diag.errorAt(pos, "exec error: %v", err).
            debug(optionDebugErrors, 1)
        }
        if opts.silent { err = nil }
      } else if exeres.Status != 0 && (!opts.silent || opts.debug) {
        diag.errorAt(pos, "abnormal exec exit status %d", exeres.Status).
          debug(optionDebugErrors, 1)
      }
    }
  }

  if !opts.silent { printEnteringDirectory() }
  if t.caller != nil { t.caller.calleeStart() }
  if true {
    exeres.wg.Add(1); go run()
    if t.caller == nil || opts.stamp/*FIXME: it's a temporary solution */ { exeres.wg.Wait() }
  } else {
    run()
  }

  // The execution is performed asynchronously, the result can't
  // be fetched immediately. Caller should do a t.wait(...) or
  // exeres.wait() before using the result.
  result = exeres
  return
}
