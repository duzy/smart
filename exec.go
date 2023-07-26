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
  rxUndefinedReference = rx(`  +"([^"]+?)", referenced from:`)
  rxUndefReference     = rx(`undef: *(.+)`)
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
  *execContext

  Buf *bytes.Buffer
  Tie  io.Writer
  line bytes.Buffer
  filters []string
  wrote uint64

  scanKnownErrors bool
  errorPos Position
  errors []error
  defaultDirectory string
  includedFrom struct { pos1, pos2 Position }
}
func (p *ExecBuffer) filter(s string) { p.filters = append(p.filters, s) }
func (p *ExecBuffer) Write(b []byte) (n int, err error) {
  if p.wrote == 0 { p.onFirstWrote() }

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
          if _, e := p.scan(p.position, &knownMatch{ rx, l, a }); e != nil {
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
  var ctx = p.Context
  if p == nil {
    erro(at(ctx,pos), "nil exec buffer").debug(1)
    return
  }

  var (
    container *Project = p.container
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
      for _, rec := range p.scannedDiags {
        if rec.msg == msg { rec.num += 1; done = true }
      }
      if !done {
        var e = &scannedExecDiag{ pos, lpos, dt, msg, 1 }
        p.scannedDiags = append(p.scannedDiags, e)
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
      if name := v[1].string; p.skips(name) {
        if p.report {
          addScannedDiag(diagError, lpos, fmt.Sprintf("container not running: %v", name))
        }
      } else {
        p.containerToRun = name
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
        var s = v[1].string
        addScannedDiag(diagError, lpos, fmt.Sprintf("'%v' file not found", filepath.Base(s)))
      }
    case rxArNoArchiveMembers:
      if p.report {
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
    //     p.errs += 1
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
    case rxUndefReference:
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

type execOpts struct {
  generalOpts
  logFileName *fullnameOpt "l,log"
  deprecated  bool `dump,deprecate`
  dropFailed  bool `df,drop,drop-fail,drop-failure,fail-drop,remove-on-fail`
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
func (p *execResult) strval(ctx Context) (s string) {
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

type execContext struct {
  Context

  execOpts
  execResult

  positions []Position
  sources []string

  log *ExecLog
  logPos Position
  srcPos Position

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

func (p *execContext) onFirstWrote() {
  if p.printEnteringOnFirstWrote {
    printEnteringDirectory(p.Context)

    // Call diagFlush to ensure printEnteringDirectory works immediately
    if errs := p.Context.dia().flush(); errs > 0 {
      warn(p.Context, "exec: encountered %d errors", errs).debug(1)
    }
  }
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

    for i, rec := range p.scannedDiags {
      if !p.infos && rec.dt == diagInfo { continue }
      if !p.logPos.IsValid() { p.logPos = rec.lpos }
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
      erro(at(ctx,p.srcPos), "%v: exit status %d", str, p.Status)
      errostack(ctx, 16, "").debug(32)
    } else if wn > 0 {
      if diffLogPos { warn(at(ctx,p.logPos), "%v: %d known warnings", str, wn) }
      warn(at(ctx,p.srcPos), "%v: exit status %d", str, p.Status)

      warn(ctx, "%v: %d known warnings", str, wn)
      warnstack(ctx, 3, "").debug(1)
    } else if in > 0 && p.infos {
      if diffLogPos { info(at(ctx,p.logPos), "%v: %d known messages", str, in) }
      info(at(ctx,p.srcPos), "%v: exit status %d", str, p.Status)
      info(ctx, "%v: %d known messages", str, in)
      infostack(ctx, 8, "").debug(1)
    }

    if p.retStatus {
      if p.zeroErrs && en == 0 && err == nil {
        p.vals = append(p.vals, MakeInt(p.logPos, int64(p.Status)))
      } else {
        p.vals = append(p.vals, MakeNone(p.logPos))
      }
    } else if p.Status != 0 || err != nil {
      // break
    }
  }

  return
}

func (exe *execContext) exec(cmd, opt string, err error) {
  var (
    ctx = exe.Context
    uni = ctx.universe()
    pc = ctx.pc()
    env, envSep = pc.env(ctx)
    program = ctx.program()
    envstr string
    logFile *os.File
  )
  for i, s := range env[envSep:] {
    if i > 0 { envstr += " && " }
    if k := strings.Index(s, "="); k > 0 {
      envstr += fmt.Sprintf(`%s%s`, s[:k+1], strconv.Quote(s[k+2:]))
    }
  }

  if ctx.dia().flush() > 0 { var total = ctx.dia().totalErrors()
    if str := trimPromptString(exe.targetName); filepath.IsAbs(exe.targetName) {
      var pos Position; pos.Filename, pos.Line = exe.targetName, 1
      warn(at(ctx,pos), "cancel execution for %s, got %d error(s)", str, total).debug(1)
    } else {
      warn(ctx, "got %d error(s), cancel execution for %s", total, str).debug(1)
    }
    if uni.failOnErrors { panic(failure{"fail by %d errors",ia(ctx.Position(), total)}) }
    return
  }

  exe.start = time.Now()

  defer func() {
    var caller = pc.caller()
    if exe.log != nil && exe.log.writer != nil { exe.log.writer.Flush() }
    if logFile != nil { logFile.Close() }
    if exe.log != nil && exe.log.filename != "" &&
      exe.Stdout.wrote == 0 && exe.Stderr.wrote == 0 {
      if false { os.Remove(exe.log.filename) }
    }
    if !exe.silentErrs && caller != nil && err != nil { caller.calleeError(err) }
    exe.Stdout.execContext = nil
    exe.Stderr.execContext = nil
    exe.container = nil
    exe.Context = nil
    exe.sh = nil
    exe.x = nil

    // Stamp the target file.
    if !exe.stamp || ctx.isConfigure() {
      // no stamp for target files
    } else if err != nil {
      var files, e = exe.target.delete(ctx)
      if e != nil { erro(ctx, `%v: delete: %v`, exe.target, e) }

      if exe.log != nil {
        var s, l = exe.log.filename, exe.log.lines
        prompt(ctx, "%v:%d: %v: %v\n", s, l, exe.target, err)
      } else {
        prompt(ctx, "%v: %v (deleted %d files)\n", exe.target, err, len(files))
      }

      for _, file := range files {
        if s, fn := file.String(), file.fullname(); s == fn {
          erro(ctx, `%v: deleted`, s)
        } else {
          erro(ctx, `%v: deleted, %v`, s, fn)
        }
      }
      errostack(ctx, 10, "%v: %v", exe.target, err).debug(16)
      return
    } else if files, e := exe.target.stamp(ctx); e != nil {
      if pe, ok := e.(*fs.PathError); ok { err = fmt.Errorf(`"%v" not found`, exe.target)
        prompt(ctx, "%v: target not found, stamp \"%v\"\n", pe.Path, exe.target)
      } else {
        prompt(ctx, "%v: target not found, \"%v\"\n", pe.Path, e)
      }
      if exe.logFileName != nil && !exe.logPos.IsValid() {
        prompt(ctx, "%v:1: see logs for \"%s\"\n", exe.logFileName.strval(ctx), exe.target)
      }
      errostack(ctx, 6, `stamp "%v" failed`, exe.target).debug(10)
      return
    } else if !exe.prompt && exe.report {
      reportFileUpdates(ctx, files)
    }

    if err == nil {
      // Good!
    } else if ctx.isConfigure() {
      err = nil
    } else {
      erro(ctx, "shell: %v", err).debug(1)
      return
    }

    if exe.prompt {
      var ps = exe.promStr
      if ps += trimPromptString(exe.targetName); caller == nil {
        if ps += " …… "; err != nil { ps += err.Error() } else { ps += "ok" }
      }
      if ps != "" {
        var s = time.Now().Sub(exe.start).String()
        if n := exe.Stdout.wrote; n > 0 { s += fmt.Sprintf(", stdout=%d bytes", n) }
        if n := exe.Stderr.wrote; n > 0 { s += fmt.Sprintf(", stderr=%d bytes", n) }
        if t := pc.dirt; t != "" { s += "; " + t }
        prompt(ctx, "%s (exec %s)\n", ps, s)
      }
    }
  } ()

  if exe.logFileName != nil { exe.log = &ExecLog{ filename: exe.logFileName.strval(ctx) } }
  if exe.bufStdout || exe.retStdout { exe.Stdout.Buf = new(bytes.Buffer) }
  if exe.bufStderr || exe.retStderr { exe.Stderr.Buf = new(bytes.Buffer) }
  if exe.tieStdout { exe.Stdout.Tie = stdout }
  if exe.tieStderr { exe.Stderr.Tie = stderr }
  if exe.log == nil || exe.log.filename == "" {
    // no log required
  } else if err = os.MkdirAll(filepath.Dir(exe.log.filename), os.FileMode(0755)); err != nil {
    erro(at(ctx,program.position), "%v", err).debug(1)
    return
  } else if logFile, err = os.Create(exe.log.filename); err != nil {
    erro(at(ctx,program.position), "%v", err).debug(1)
    return
  } else {
    cmdline := strings.Join(exe.sources, "\n")
    exe.log.createWriter(logFile, exe.workDir, cmdline)
  }
  exe.Stdout.scanKnownErrors = exe.scanStdout
  exe.Stderr.scanKnownErrors = exe.scanStderr
  exe.Stdout.defaultDirectory = exe.workDir
  exe.Stderr.defaultDirectory = exe.workDir
  exe.Stdout.execContext = exe
  exe.Stderr.execContext = exe

  for i, src := range exe.sources {
    var ctx = at(ctx, exe.positions[i])
    exe.srcPos = ctx.Position()

    if a := "@"; strings.HasPrefix(src, a) {
      src = strings.TrimPrefix(src, a)
    } else if exe.promptSrc && !exe.prompt {
      var s string = src
      s = strings.Replace(s, "\n", "\\n", -1)
      s = strings.Replace(s, "\\\\n", "\\\n", -1)
      prompt(ctx, "%s\n", s)//.debug(1)
    }

    if src = strings.TrimSpace(src); src == "" { continue }

    if false && !exe.noCD && exe.workDir != "" {
      if strings.HasPrefix(src, "#") {
        src = fmt.Sprintf("cd '%s' %s", exe.workDir, src)
      } else {
        // Insert a "\n" before the right paren ')' to ensure that
        // it's working with comments like "true #comment...".
        src = fmt.Sprintf("cd '%s' && (%s\n)", exe.workDir, src)
      }
    }

    if cmd == "docker" && len(envstr) > 0 {
      src = fmt.Sprintf("%s && %s", envstr, src)
    }

    if uni.noExec { continue }

    if !exe.silentErrs || exe.prompt || exe.promptSrc {
      exe.printEnteringOnFirstWrote = true
    }

    exe.sh = exec.Command(cmd, exe.args...)
    exe.sh.Dir = exe.workDir // always set command work directory
    exe.sh.Env = env
    exe.sh.Stdout = &exe.Stdout
    exe.sh.Stderr = &exe.Stderr
    if exe.stdin {
      exe.sh.Stdin = os.Stdin
      exe.sh.Args = append(exe.sh.Args, "-ti")
    }
    if opt != "" { exe.sh.Args = append(exe.sh.Args, opt) }
    if src != "" { exe.sh.Args = append(exe.sh.Args, src) }
    if exe.debug > 0 {
      warn(at(ctx,program.position), "%v: %v", ctx.entry(), exe.target)
      warn(ctx, "context: %v", ctx.pc())
      warn(ctx, "exec:\n%v", exe.sh).debug(exe.debug*2)
    }

    exe.Stdout.report = !exe.silentErrs
    exe.Stderr.report = !exe.silentErrs
    if err = exe.run(); exe.Status != 0 || err != nil { break }
  }
}

type executor struct {
  cmd, opt string
  contained bool
}
func (p *executor) Evaluate(ctx Context, args ...Value) (result Value, err error) {
  var uni = ctx.universe()
  if uni.traceExecutor {
    var t = autoVal(ctx, "@")
    defer un(trace(t_exec, fmt.Sprintf("executor(%s %v)", typeof(t), t)))
  }

  var (
    pos = ctx.Position()
    exe = &execContext{ Context:ctx, x:p }
    cmd = p.cmd
  )

  exe.position = pos
  exe.scanStderr = true
  args = parseOpts(ctx, &exe.execOpts, plain, args...)

  if exe.deprecated {
    erro(ctx, "deprecated args: -v (-to), -w (-te), -a (-se), -d (-t)").debug(1)
    return
  } else if d := exe.debug; d>0 { defer func() {
    warnstack(ctx, d, "%v: %v (%v)", ctx.entry(), exe.target.Value, result).debug(d)
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
  } else if exe.targetName, _ = exe.target.fullnameOrStrval(ctx); ctx.isConfigure() {
    // does nothing
  } else if exe.waitRes {
    // good to work without (stamp) or (wait) with the -wait flag
  } else if m := program.getModifiers(ctx, "wait"); len(m) > 0 {
    // should be good to work
  } else if t := exe.target.Value; !(exe.stamp || exe.noStamp || exe.silentErrs) {
    warn(ctx, "add -stamp to (shell); target=%v (%T)", t, t).debug(1)
  }

  if (exe.retStdout && exe.retStatus) || (exe.retStderr && exe.retStatus) {
    erro(ctx, "cannot have both status and stdout|stderr at the same time (try -so or -se)").debug(1)
    return
  }

  for i, v := range args {
    var s string
    if p.contained && i == 0 {
      if s = v.strval(ctx); s == "shell" {
        cmd = defaultShell
      }
    } else if s = strings.TrimSpace(v.strval(ctx)); s != "" {
      exe.args = append(exe.args, s)
    }
  }

  if p.contained {
    if program.project.name == dotContainer {
      exe.container = program.project
    } else if _, containerSym := program.project.scope.Find(dotContainer); containerSym != nil {
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
      if obj := exe.container.resolveObject(ctx, name); obj != nil {
        if d, _ := obj.(*def); d != nil {
          if v := d.invoke(ctx, plain, nil, nil); v != nil {
            if str = v.strval(ctx); str == "-" {
              /*if v, err = def.DiscloseValue(exe.container); err == nil && v != nil {
                  if str, err = v.strval(ctx); str == "" { str = "-" }
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

  // Fixes work directory conflicts. It happens
  // sometimes even the 'sh.Dir' is set to cwd.
  // Because the current work directory is not
  // thread safe.
  if exe.workDir != "" {
    // good
  } else if exe.workDir = program.workDir(ctx); exe.workDir == "" {
    erro(ctx, "CWD is empty").debug(1)
    return
  }

  if exe.path {
    var s string
    if s = filepath.Dir(exe.targetName); s != "" && s != "." && s != "/" {
      if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
        erro(of(ctx,exe.target), "make path '%s' for target failed: %v", s, err).debug(1)
        return
      }
    }
  }

  var (
    recipePos Position
    recipes []Value
    source string
    w = plain
  )

  if exe.fullname { w |= expandFullName }
  recipes = xmerge(ctx, w, program.recipes...)

  for _, recipe := range recipes {
    if !recipePos.IsValid() { recipePos = recipe.Position() }

    var str = recipe.strval(ctx)
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

    exe.positions = append(exe.positions, recipePos) ; recipePos = Position{}
    exe.sources = append(exe.sources, source)
    source = ""
  }
  if len(recipes) > 0 && len(exe.sources) == 0 {
    erro(ctx, "empty recipes: %v", recipes).debug(1)
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
    if exe.Stdout.Buf != nil {
      s = exe.Stdout.Buf.String()
    }
    exe.vals = append(exe.vals, MakeString(pos, s))
  }

  // Add stderr result
  if exe.retStderr {
    var s string
    if exe.Stderr.Buf != nil {
      s = exe.Stderr.Buf.String()
    }
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
