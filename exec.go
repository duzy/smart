//
//  Copyright (C) 2012-2024, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "bufio"
    "bytes"
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "unicode"
    "regexp"
    "reflect"
    "strings"
    "strconv"
    "sync"
    "sync/atomic"
    "time"
)

// Note that it's is also used with Sscanf.
const (
    fmtExitStatus = "exit status %d"
    maxPromptStr = 48
    maxWorkers = 3
    maxRetries = 1
)

type exec_opts struct {
    general_opts
    logname *fullname "log"
    forRecipe Value `forrecipe,forrecipes,for-recipe,for-recipes`
    forStdout Value `forstdout,for-stdout,for-out`
    forStderr Value `forstderr,for-stderr,for-err`
    result    Value `result,return`
    removeOnFail bool `drop-fail,drop-failure,remove-failure,remove-on-fail`
    zeroStatusErrors bool `zero-status-errors`
    zeroErrs    bool `no-error,no-errors,zero-errors` // require zero error scaned from STDERR
    report      bool `report,report-stamp,verbose-stamp`
    silent      bool `silent,silent-errors` // silent errors
    stdin       bool `stdin,input`
    stdoutBuf   bool `stdout`
    stderrBuf   bool `stderr`
    stdoutTie   bool `tie-out,tie-stdout` // tied with log
    stderrTie   bool `tie-err,tie-stderr` // tied with log
    scanStdout  bool `scan-stdout,scan-out`
    scanStderr  bool `scan-stderr,scan-err`
    scanInfos   bool `scan-infos`
    parallel    bool `parallel,no-order`
    path        bool `path`
    prompt      bool `prompt,msg`
    promptSrc   bool `prompt-src,prompt-source,verbose-source`
    note        bool `note`
    cmd         string `cmd`
    tie         string `tie` // all, both, stdout, stderr, out, err
    _workdir    string `cd,dir,workdir,work-dir,work-directory`
}

type exitstatus struct { int }
func (p *exitstatus) Error() string { return fmt.Sprintf(fmtExitStatus, p.int) }

var (
    defaultShell = "bash"
    udots = []byte("…")

    workingMutex = new(sync.Mutex)
    working atomic.Value // number of working executions

    stdout = &std_writer{io:os.Stdout}
    stderr = &std_writer{io:os.Stderr}

    rxExitStatus        = regexp.MustCompile(`^exit status (\-?[0-9]+)$`)
    rxFileNotFound      = regexp.MustCompile(`'(.+?)' file not found$`)
    rxCodeLinePanic     = regexp.MustCompile(`^([^:]+?):(\d+):(\d+): *((?:fatal )?error|warning): *(.+)$`)
    rxIgnoringDirectory = regexp.MustCompile(`^(ignoring (?:duplicate|nonexistent) directory) "(.*?)"`)
    rxLdManyMinVersions = regexp.MustCompile(`^(?:[^:]+?: )+(passed two min versions \((.+?)\) for platform macOS\. Using (.+)\.)`)

    rxArNoMembers     = regexp.MustCompile(`ar: no archive members specified`)
    rxArNoSuchFileDir = regexp.MustCompile(`ar: (.+?): No such file or directory`)

    rxShellNoSuchFileDir = regexp.MustCompile(`^bash:(?: line ([0-9]+?):)? (.+?): No such file or directory`)

    rxGitNotRepo = regexp.MustCompile(`^fatal: (not a git repository): '(.+?)'`)

    rxDockerCannotConnect   = regexp.MustCompile(`Cannot connect to the Docker daemon at (.*?)\. Is the docker daemon running\?`)
    rxDockerConNotRunning   = regexp.MustCompile(`Error response from daemon: (Container (.+?) is not running)`)
    rxDockerNoSuchContainer = regexp.MustCompile(`Error.*: No such container: (.*)`)
    rxDockerNetworkNotFound = regexp.MustCompile(`Error.*: (network (.*) not found)\.`)

    rxIncludedFrom     = regexp.MustCompile(`In file included from (.+?):(\d+):(?:(\d+):)?`)
    rxPyFileLineIn     = regexp.MustCompile(`^\s*File "(.+?)", line (\d+), in (.+)`)
    rxPyFileNotFound   = regexp.MustCompile(`FileNotFoundError: \[Errno (\d+)\] No such file or directory: '(.*?)'`)
    rxPyModuleNotFound = regexp.MustCompile(`ModuleNotFoundError: No module named '(.*?)'`)

    // ld: warning: passed two min versions (15.0, 23.2) for platform macOS. Using 23.2.
    rxNoticeLines = []*regexp.Regexp{
        regexp.MustCompile(`ld: library '[^']+' not found`),
    }

    rxZeroStatusErrors = map[*regexp.Regexp]struct{}{
        rxShellNoSuchFileDir:nv,
    }

    matchcontexts = map[*regexp.Regexp]func(*exec_buffer, []byte, [][]byte)Context{
        rxCodeLinePanic: func(p *exec_buffer, line []byte, sm [][]byte) Context {
            return p.sc(sm[1], sm[2], sm[3], 0) // TODO: column(line, sm[4])
        },
    }

    commonerrors = map[*regexp.Regexp]func(Context, []byte, [][]byte){
        rxExitStatus: func(c Context, line []byte, sm [][]byte) {
            if string(sm[1]) != "0" { errostack(c, 5, "%s", sm[0]).debug() }
        },
        rxShellNoSuchFileDir: func(c Context, line []byte, sm [][]byte) {
            errostack(c, 5, "no such command '%s'", sm[2]).debug()
        },
        regexp.MustCompile(`(.+?): (.+?):( command)? not found`): func(c Context, line []byte, sm [][]byte) {
            erro(c, "%s: command not found", sm[2]).debug()
        },
        regexp.MustCompile(`the input device is not a TTY`): func(c Context, line []byte, sm [][]byte) {
            errostack(c, 5, "%s", sm[0]).debug()
        },
    }

    // `(?P<first>\d+)\.(\d+).(?P<second>\d+)`
    knownerrors = map[*regexp.Regexp]map[*regexp.Regexp]func(Context, []byte, [][]byte){
        regexp.MustCompile(`^(?:.*/)?clang`):map[*regexp.Regexp]func(Context, []byte, [][]byte){
            rxCodeLinePanic: func(c Context, line []byte, sm [][]byte) {
                t := string(sm[4])
                s := string(sm[5])
                switch t {
                case "warning":
                    warnstack(c, 5, "%s", s).debug(3)
                default:
                    errostack(c, 5, "%s", s).debug() // "error", "fatal error"
                }
                if m := rxFileNotFound.FindStringSubmatch(s); m != nil {
                    do(c, missing_file{m[1]})
                }
            },

            rxIgnoringDirectory: func(c Context, line []byte, sm [][]byte) {
                notestack(pc(c,sm[2]), 5, "%s", sm[1]).debug()
            },

            rxLdManyMinVersions: func(c Context, line []byte, sm [][]byte) {
                notestack(c, 5, "%s", sm[1]).debug()
            },

            regexp.MustCompile(`  +"([^"]+?)", referenced from:`): func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },
            regexp.MustCompile(`undef: *(.+)`): func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },

            regexp.MustCompile(`((?:clang|wasm|(?:[^\.]+\.)?l?ld)(?:\-.+?)?): (error|warning): *(.+)`): func(c Context, line []byte, sm [][]byte) {
                if truly(c, is_configure{}) && string(sm[2]) == "warning" { return }
                errostack(c, 5, "%s", sm[0]).debug()
            },
            regexp.MustCompile(`((?:clang|wasm|(?:[^\.]+\.)?l?ld)(?:\-.+?)?): could not parse object file (.+?): '(.+)', using libLTO version '(.+?)' file '(.+?)' for architecture (.+)`): func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },
            regexp.MustCompile(`((?:clang|wasm|(?:[^\.]+\.)?l?ld)(?:\-.+?)?): library not found for (.+)`): func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },

            regexp.MustCompile(`(.+?): Too many positional arguments specified!`): func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },
        },
        regexp.MustCompile(`^(?:.*/)?ar`):map[*regexp.Regexp]func(Context, []byte, [][]byte){
            rxArNoSuchFileDir: func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "'%s' file not found", filepath.Base(string(sm[1]))).debug()
            },
            rxArNoMembers: func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },
        },
        regexp.MustCompile(`^(?:.*?bash -c|.*?)git`):map[*regexp.Regexp]func(Context, []byte, [][]byte){
            rxGitNotRepo: func(c Context, line []byte, sm [][]byte) {
                errostack(pc(c,sm[2]), 5, "%s", sm[1]).debug()
            },
        },
        regexp.MustCompile(`^(?:.*/)?python`):map[*regexp.Regexp]func(Context, []byte, [][]byte){
            rxIncludedFrom: func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },
            rxPyFileLineIn: func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },
            rxPyFileNotFound: func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },
            rxPyModuleNotFound: func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },
        },
        regexp.MustCompile(`^(?:.*/)?docker`):map[*regexp.Regexp]func(Context, []byte, [][]byte){
            rxDockerCannotConnect: func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },
            rxDockerConNotRunning: func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },
            rxDockerNoSuchContainer: func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },
            rxDockerNetworkNotFound: func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },
        },
        regexp.MustCompile(`^(?:.*/)?protoc`):map[*regexp.Regexp]func(Context, []byte, [][]byte){
            regexp.MustCompile(`^(.+?\.proto): File not found\.`): func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },
            regexp.MustCompile(`^(.+?\.proto):(\d+):(\d+): Import "(.+?)" was not found or had errors.`): func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },
            regexp.MustCompile(`^(.+?\.proto):(\d+):(\d+): "(.+?)" is not defined.`): func(c Context, line []byte, sm [][]byte) {
                errostack(c, 5, "%s", sm[0]).debug()
            },
        },
        regexp.MustCompile(`^(?:.*/)?echo`):map[*regexp.Regexp]func(Context, []byte, [][]byte){
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

type std_writer struct {
    sync.Mutex
    io io.Writer
    suffixDots bool
}

func (w *std_writer) Write(p []byte) (n int, err error) {
    w.Lock(); defer w.Unlock()
    if w.suffixDots {
      if !bytes.HasPrefix(p, udots) {
        w.io.Write([]byte("\n"))
      }
      w.suffixDots = false
    }
    if n, err = w.io.Write(p); bytes.HasSuffix(p, udots) {
      w.suffixDots = true
    }
    return
}

type exec_log struct {
    sync.Mutex
    writer *bufio.Writer
    filename string
    lines int
}
func (p *exec_log) Write(b []byte) (n int, err error) {
    p.Lock(); defer p.Unlock()
    p.lines += bytes.Count(b, []byte("\n"))
    n, err = p.writer.Write(b)
    return
}
func (p *exec_log) createWriter(file *os.File, dir, cmd string) {
    p.writer = bufio.NewWriter(file)
    fmt.Fprintf(p, "-*- mode: compilation; default-directory: \"%s\" -*-\n", dir)
    fmt.Fprintf(p, "Compilation started at %v\n\n", time.Now())
    fmt.Fprintf(p, "%s\n", cmd)
}

func is_notice_line(s string) (_ bool) {
    for _, x := range rxNoticeLines {
        if x.MatchString(s) { return true }
    }
    return
}

type exec_buffer struct {
    *exec_ctx

    Tie  io.Writer
    Buf *bytes.Buffer
    line bytes.Buffer // works done line by line
    lnum int // line number

    wrote uint64

    forLine Value
}
func (p *exec_buffer) Write(b []byte) (n int, err error) {
    var expandForLine = p.forLine != nil && !isTrivial(p.forLine)

    if p.Buf != nil {
        if n, err = p.Buf.Write(b); err != nil { return }
    }
    if p.log != nil {
        if _, err = p.log.Write(b); err != nil { return }
    }
    if p.Tie != nil {
        if n, err = p.Tie.Write(b); err != nil { return }
    }
    if err == nil && n == 0 {
        // Returns the number of bytes to avoid "short write" errors.
        // The real bytes written is discarded.
        n = len(b)
    }

    p.wrote += uint64(n)

    scanLine := expandForLine ||
        (p.scanStdout && p == &p.Stdout) ||
        (p.scanStderr && p == &p.Stderr)

    if !scanLine { return }
    if false && truly(p, is_rule{rxConfigRuleHeaders}) {
        note(p, "%s %s", do(p, execution_lang{}), p.sh).debug()
    }

    for slice := b[:]; len(slice) > 0; {
        var i = bytes.Index(slice, []byte("\n"))
        if i == -1 {
            p.line.Write(slice)
            slice = nil
        } else {
            p.lnum += 1
            p.line.Write(slice[:i+1])
            slice = slice[i+1:]

            var line = p.line.Bytes()

            if checkpoints && truly(p, is_test_mode{}) {
                p.check_line(string(line), p.lnum)
            }

            if expandForLine {
                c := p.exec_ctx
                c.line.s = string(line)
                c.lino.int64 = int64(p.lnum)
                v := p.forLine.expand(_final(p.Context))
                if !isNull(v) && is_notice_line(c.line.s) {
                    note(p, "%v : %d. %s → %v", p.forLine, line, c.line.s, ts(v)).debug()
                }
            }

            k := func(rx *regexp.Regexp, f func(Context, []byte, [][]byte)) {
                if sm := rx.FindSubmatch(line); sm != nil {
                    if p.zeroStatusErrors && rxZeroStatusErrors != nil {
                        if _, y := rxZeroStatusErrors[rx]; y {
                            p.resetStatusZero = true
                        }
                    }
                    c := Context(p)
                    if x, y := matchcontexts[rx]; y {
                        c = x(p, line, sm)
                    } else {
                        c = p.pc(0)
                    }
                    if !truly(c, is_configure_ignore{rx, sm}) {
                        f(c, line, sm)
                    }
                }
            }
            for rx, f := range commonerrors { k(rx, f) }
            for rx, f := range p.known { k(rx, f) }

            p.line.Reset()
        }
    }
    return
}
func (p *exec_buffer) startDockerDaemon(pos Position, ctx Context, container *project, sock string) (err error) {
    var c = exec.Command("dockerd") //c.Stdout, c.Stderr = stdout, stderr
    if err = c.Run(); err != nil {
      if p.report {
        erro(ctx, "dokcer daemon not running (at %s)", sock).trace()
      }
    } else {
      // TODO: start docker daemon
    }
    return
}
func (p *exec_buffer) filepath(s string) string {
    if p._workdir != "" && !filepath.IsAbs(s) { s = filepath.Join(p._workdir, s) }
    return s
}
func (p *exec_buffer) covpos(s1, s2, s3 string) Position {
    return convPosition(p.filepath(s1), s2, s3)
}
func (p *exec_buffer) lpos(column int) Position {
    var pos = p.position
    if p.log != nil {
        pos.Filename, pos.Line, pos.Column = p.log.filename, p.lnum, column
    }
    return pos
}
func (p *exec_buffer) pc(column int) Context {
    return pc(p, p.lpos(column))
}
func (p *exec_buffer) sc(b1, b2, b3 []byte, column int) Context {
    s1, s2, s3 := string(b1), string(b2), string(b3)
    return pc(pc(p,s1,atoi(s2),atoi(s3)), p.lpos(column))
}

type exec_result struct {
    valbase
    values []Value
    Stdout exec_buffer
    Stderr exec_buffer
    Status int // aka. exit code
}
func (p *exec_result) hash(ctx Context) uint64 {
    var a []any
    for _, v := range p.values { a = append(a, v) }
    return fnv1(ctx, p, a...)
}
func (p *exec_result) expand(_ Context) Value { return p }
func (p *exec_result) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*exec_result); ok {
      assert(ok, "value is not exec_result")
      if p.Status == a.Status { res = cmpEqual }
    }
    return
}
func (p *exec_result) true(ctx Context) (res bool) {
    res = p.Status == 0 && p.Stderr.Buf != nil && p.Stderr.Buf.Len() == 0 /* && p.Stdout.Buf.Len() > 0 */
    return
}
func (p *exec_result) int(ctx Context) (i int64) { return int64(p.Status) }
func (p *exec_result) float(ctx Context) (f float64) { return float64(p.Status) }
func (p *exec_result) string(ctx Context) (s string) {
    if p.Stdout.Buf != nil { return p.Stdout.Buf.String() }
    if p.Stderr.Buf != nil { return p.Stderr.Buf.String() }
    return strconv.Itoa(p.Status)
}
func (p *exec_result) String() string {
    var s bytes.Buffer
    fmt.Fprintf(&s, "{=exec {=status %d}", p.Status)
    if p.Stdout.Buf != nil { fmt.Fprintf(&s, " {=stdout %v}", p.Stdout.Buf) }
    if p.Stderr.Buf != nil { fmt.Fprintf(&s, " {=stderr %v}", p.Stderr.Buf) }
    fmt.Fprintf(&s, "}")
    return s.String()
}

type is_exec struct{}
type exec_ctx struct {
    Context

    exec_opts
    exec_result

    line strlit
    lino decimal

    log *exec_log
    logPos Position

    target as
    targetName string

    retried map[string]bool // work with containerToRun
    containerToRun string   // work with retried
    container *project

    num int

    sh *exec.Cmd
    args []string

    known map[*regexp.Regexp]func(Context, []byte, [][]byte)

    start time.Time

    resetStatusZero bool
}
func (p *exec_ctx) inner() Context { return p.Context }
func (p *exec_ctx) cast(t reflect.Type) Context { return icast(p,t) }
// func (p *exec_ctx) String() string { return p.Context.String() }
func (p *exec_ctx) ts(t string) (s string) {
    s = "{=" + t
    if p.sh != nil {
      s += " " + filepath.Base(p.sh.Path)
    }
    s += " " + ts(p.Context) + "}"
    return
}
func (p *exec_ctx) do(ctx Context, op any) any {
    switch op.(type) {
    case is_exec: return true
    case wants_fullfile: return p.fullname
    }
    return p.Context.do(ctx, op)
}

func (p *exec_ctx) runContainerAndRetry(exe *execution) (err error) {
    if p.container == nil {
      erro(p.Context, "no container").trace()
    } else if maxRetries < p.num {
      fmt.Fprintf(p.sh.Stderr, "\n---- Retried %d times\n", p.num)
      return
    }

    var (
      name = p.containerToRun
      sh = p.sh
    )

    fmt.Fprintf(sh.Stderr, "\n---- Run container '%s'\n", name)
    if entries := p.container._entries(p.Context, "run", false); entries != nil {
      for _, run := range entries {
        run.execute(p.Context, nil)
      }
    } else {
      erro(p.Context, "%s⇒run undefined", p.container).trace()
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
    if err = p.run(exe); err != nil {
      fmt.Fprintf(sh.Stderr, "\n---- Retry failed: %s\n", err)
    }
    return
}

// DEPRECATED
func (p *exec_ctx) DEPRECATED_ensureContainerRunning(containerName string) (err error) {
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
      if entries := p.container._entries(p.Context, "run", false); entries != nil {
        for _, run := range entries {
          run.execute(p.Context, nil)
        }
      } else {
        erro(p.Context, "%s⇒run undefined", p.container).trace()
      }
    } else if err != nil {
      erro(p.Context, "%v", err).trace()
    }
    return
}

func (p *exec_ctx) skips(tag string) bool {
    if p.retried == nil { p.retried = make(map[string]bool) }
    var a, b = p.retried[tag]
    return a && b
}

func (p *exec_ctx) run(exe *execution) (err error) {
    if p.containerToRun != "" {
      p.retried[p.containerToRun] = true // mark it to skip next time
      err = p.runContainerAndRetry(exe)
      p.containerToRun = ""
      return
    }

    if checkpoints && truly(p, is_test_mode{}) { defer p.run_check(exe) }

    exe.Add(1)
    p.num += 1

    run := func(c *exec.Cmd) {
        defer exe.Done()

        err = c.Run();

        if err == nil {
            err = p.check()
        } else if x, y := err.(*exec.ExitError); y {
            if p.Status = x.ExitCode(); p.Status == 0 { err = p.check() } // success!
            if p.resetStatusZero { p.Status = 0 }
        } else {
            erro(p.Context, "exec failed: %v", err).trace()
            return
        }
    }

    if true { run(p.sh) } else { go run(p.sh) }
    if p.note {
        prompt(p, "%s\n", p.sh)
        if buf := p.Stdout.Buf; buf != nil { prompt(p, "%s", buf) }
        notestack(pc(p,auto_get(p,"@")), 3, "status=%v", p.Status).debug(32)
    }
    return
}

func (p *exec_ctx) check() (err error) {
    if (!p.silent || p.debug>0) && (/* len(p.scannedDiags) > 0 || */ p.Status != 0 || err != nil) {
        if p.silent /* || p.retStatus */ {
            err = nil
        } else if p.Status != 0 {
            err = &exitstatus{ p.Status } // set or convert error
        }

        var en, wn, in int
        // for _, rec := range p.scannedDiags {
        //     switch rec.dt {
        //     case diagError: en += rec.num
        //     case diagWarn:  wn += rec.num
        //     case diagInfo:  in += rec.num
        //     }
        // }

        var ctx = p.Context
        if en > 0 || p.Status != 0 || err != nil {
            prompt(ctx, "exec: failure (status=%d; err=%v); target=%s\n", p.Status, err, p.targetName)
        } else if wn > 0 {
            prompt(ctx, "%v: %d warnings\n", p.targetName, wn)
        }

        // for i, rec := range p.scannedDiags {
        //     if !p.scanInfos && rec.dt == diagInfo { continue }
        //     if !p.logPos.IsValid() { p.logPos = rec.position }
        //     if i == 0 && !rec.position.same(&rec.position) {
        //         diag(ctx, rec.dt, rec.msg)//.debug()
        //     }
        //     if rec.num > 1 {
        //         diag(ctx, rec.dt, `%s (%d)`, rec.msg, rec.num)//.debug()
        //     } else {
        //         diag(ctx, rec.dt, rec.msg)//.debug()
        //     }
        //     if n := (en+wn+in)-(i+1); i == 8 && 0 < n {
        //         diag(ctx, rec.dt, "%d more...", n)//.debug()
        //         break
        //     }
        // }

        var pos = _position(ctx)
        if !p.logPos.IsValid() && p.log != nil {
            p.logPos.Filename = p.log.filename
            p.logPos.Line = p.Stderr.log.lines + 1
        } else {
            p.logPos = pos
        }

        var diffLogPos = !p.logPos.sameLine(&pos)
        var str, _, _ = entryIndicator(ctx, _entry(ctx))
        if (/* !p.retStatus && */ p.Status != 0) || en > 0 {
            if p.removeOnFail {
                if e := os.RemoveAll(p.targetName); e != nil {
                    warn(ctx, "remove: %v", e).debug()
                }
            }

            if diffLogPos && en > 0 { erro(ctx, "%v: %d known errors", str, en) }
            erro(p, "%v: exit status %d", str, p.Status).trace()
        } else if wn > 0 {
            if diffLogPos { warn(ctx, "%v: %d known warnings", str, wn) }
            warn(p, "%v: exit status %d", str, p.Status)
            warn(ctx, "%v: %d known warnings", str, wn)
            warnstack(ctx, 3).debug()
        } else if in > 0 && p.scanInfos {
            if diffLogPos { info(ctx, "%v: %d known messages", str, in) }
            info(p, "%v: exit status %d", str, p.Status)
            info(ctx, "%v: %d known messages", str, in)
            infostack(ctx, 8).debug()
        }

        // if p.retStatus {
        //   if p.zeroErrs && en == 0 && err == nil {
        //     p.vals = append(p.vals, _decimal(p.logPos, int64(p.Status)))
        //   } else {
        //     p.vals = append(p.vals, _none(p.logPos))
        //   }
        // } else if p.Status != 0 || err != nil {
        //   // break
        // }
    }
    return
}

func (ctx *exec_ctx) sources(recipes []Value) (sources []*raw) {
      var a1 *strlit
      var a2 *decimal
      var ac *automatic
      if ctx.forRecipe != nil {
          a1, a2 = &strlit{}, &decimal{}
          ac = &automatic{Context:ctx, defs:make(defs_map)}
          ac.args(ac.Context, []Value{a1, a2})
      }

      var pos Position
      var source string
      for i, recipe := range recipes {
          if !pos.IsValid() { pos = recipe.Position() }

          var cc Context = _final(pc(ctx, pos))
          var s = recipe.string(cc)

          if checkpoints && truly(ctx, is_test_mode{}) {
              ctx.sources_check(cc, i, recipe, s)
          }

          if s = strings.TrimRightFunc(s, unicode.IsSpace); s == "" {
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
              if i < len(recipes) { continue }
          }

          // Remove tabs in line breakings.
          source = strings.Replace(source, "\\\n\t", "\\\n", -1)
          sources = append(sources, &raw{valbase{pos}, source})

          if ctx.forRecipe != nil {
              a1.position, a1.s     = pos, source
              a2.position, a2.int64 = pos, int64(len(sources)+1)
              ac.Context = ctx
              if v := ctx.forRecipe.expand(_final(ac)); false && v != nil {
                  for i := 0; indeterminate(ac, v); i += 1 {
                      if i < max_evoke {
                          v = v.expand(_final(ac))
                      } else {
                          erro(ctx, "%v → %v", ctx.forRecipe, v).trace()
                      }
                  }
              }
          }

          pos, source = Position{}, ""
      }

      if len(sources) == 0 && 0 < len(recipes) {
          erro(ctx, "empty recipes: %v", recipes).trace()
      }
      return
}

func (ctx *exec_ctx) exec(cmd, opt string) {
    var exe = _execution(ctx)
    var env, sep = exe.env(ctx)
    var envs string
    var logFile *os.File

    for i, s := range env[sep:] {
        if i > 0 { envs += " && " }
        if k := strings.Index(s, "="); k > 0 {
            envs += fmt.Sprintf(`%s%s`, s[:k+1], strconv.Quote(s[k+2:]))
        }
    }

    defer func() {
        if ctx.log != nil && ctx.log.writer != nil { ctx.log.writer.Flush() }
        if logFile != nil { logFile.Close() }
        if false && ctx.log != nil && ctx.log.filename != "" {
            if ctx.Stdout.wrote == 0 && ctx.Stderr.wrote == 0 {
                os.Remove(ctx.log.filename)
            }
        }

        ctx.Stdout.exec_ctx = nil
        ctx.Stderr.exec_ctx = nil
        ctx.container = nil
        ctx.sh = nil

        if !ctx.silent && !is_configurecontext(ctx) && ctx.target.Value != nil {
            var files = ctx.target.stamp(must_files_stamp{ctx})
            if !ctx.prompt && ctx.report { reportFileUpdates(ctx, files) }
        }

        if !ctx.silent && ctx.prompt {
            var ps = ctx.cmd + trimPromptString(ctx.targetName)
            if _execution(exe.Context) == nil { ps += " …… ok" }
            if ps != "" {
                var s = time.Now().Sub(ctx.start).String()
                if n := ctx.Stdout.wrote; n > 0 { s += fmt.Sprintf(", stdout=%d bytes", n) }
                if n := ctx.Stderr.wrote; n > 0 { s += fmt.Sprintf(", stderr=%d bytes", n) }
                if t := exe.dirt; t != "" { s += "; " + t }
                prompt(ctx, "%s (exec %s)\n", ps, s)
            }
        }
    } ()

    ctx.Stdout.forLine = ctx.forStdout
    ctx.Stderr.forLine = ctx.forStderr

    if ctx.forStdout != nil || ctx.forStderr != nil {
        ac := automatic{Context:ctx.Context, defs:make(defs_map)}
        ac.args(ac.Context, []Value{&ctx.line, &ctx.lino})
        if x, y := ac.defs["1"]; y {
            ac.defs["_"] = x // alias
        } else {
            erro(ctx, "wrong args: %v", ac.defs).trace()
        }
        ctx.Context = &ac
    }

    if ctx.stdoutBuf { ctx.Stdout.Buf = new(bytes.Buffer) }
    if ctx.stderrBuf { ctx.Stderr.Buf = new(bytes.Buffer) }
    if ctx.stdoutTie { ctx.Stdout.Tie = stdout }
    if ctx.stderrTie { ctx.Stderr.Tie = stderr }
    if ctx.logname != nil {
        ctx.log = &exec_log{ filename: ctx.logname.string(ctx) }
    }

    var srcs = ctx.sources(exe.recipes)

    if ctx.log == nil || ctx.log.filename == "" {
        // no log required
    } else if err := os.MkdirAll(filepath.Dir(ctx.log.filename), os.FileMode(0755)); err != nil {
        erro(ctx, "%v", err).trace()
    } else if logFile, err = os.Create(ctx.log.filename); err != nil {
        erro(ctx, "%v", err).trace()
    } else {
        cmdline := joinraws("\n", srcs...)
        ctx.log.createWriter(logFile, ctx._workdir, cmdline)
    }

    ctx.Stdout.exec_ctx = ctx
    ctx.Stderr.exec_ctx = ctx
    ctx.start = time.Now()

    var noExec = truly(ctx, no_exec{})
    for _, src := range srcs {
        if src.trim("@"); src.s == "" { continue }
        if ctx.promptSrc && !ctx.prompt {
            s := src.s
            s = strings.Replace(s, "\n", "\\n", -1)
            s = strings.Replace(s, "\\\\n", "\\\n", -1)
            prompt(ctx, "%s\n", s)
        }

        if checkpoints && truly(ctx, is_test_mode{}) { ctx.exec_check(exe, src) }
        if cmd == "docker" && len(envs) > 0 { src.s = envs+" && "+src.s }
        if noExec { continue }

        ctx.known = nil
        for rx, m := range knownerrors {
            if rx.MatchString(src.s) { ctx.known = m }
        }
        if ctx.known == nil {
            note(ctx, "unknown: %s", src).debug()
        }

        ctx.sh = exec.Command(cmd, ctx.args...)
        ctx.sh.Env = env
        ctx.sh.Dir = ctx._workdir // always set command work directory
        ctx.sh.Stdout = &ctx.Stdout
        ctx.sh.Stderr = &ctx.Stderr
        if ctx.stdin {
            ctx.sh.Stdin = os.Stdin
            ctx.sh.Args = append(ctx.sh.Args, "-ti")
        }
        if   opt != "" { ctx.sh.Args = append(ctx.sh.Args, opt) }
        if src.s != "" { ctx.sh.Args = append(ctx.sh.Args, src.s) }
        if e := ctx.run(exe); ctx.Status != 0 || e != nil { break }
    }
}

type executor struct {
    cmd, opt string
    contained bool
}
func (p *executor) evaluate(ctx Context, args ...Value) (result Value) {
    var prog = _program(ctx)
    if prog == nil {
        erro(ctx, "needs program context to exec: %v", ctx).trace()
    }

    if false && truly(ctx, is_test_univ{}) {
        defer func() {
            note(ctx, "%v %v %v", p.cmd, args, result).debug()
        } ()
    }

    var cmd = p.cmd
    var ec = &exec_ctx{Context:ctx}
    ec.exec_result.position = _position(ctx)
    ec.scanStderr = true

    args = parse_opts(ctx, &ec.exec_opts, args...)

    if !ec.prompt { ec.prompt = ec.cmd != "" }

    var resKind, resType string
    var resValue Value
    var trim = identity

    if r := ec.result; r != nil {
        for {
            var x, y = r.(*argumented)
            if !y { break }
            if len(x.args) != 1 {
                erro(pc(ctx,x), "wrong result spec: %v", x).trace()
            }

            switch s := x.Value.string(ctx); s {
            case "trim": trim = strings.TrimSpace
            default: resType = s
            }

            if l, y := x.args[0].(*list); y {
                if l.len() != 1 {
                    erro(pc(ctx,x), "wrong result spec: %v", x).trace()
                }
                r = l.elems[0]
            }
        }
        if x, y := r.(*pair); y {
            resKind = x.key.string(ctx)
            resValue = x.val
        } else {
            resKind = r.string(ctx)
        }
        switch resKind {
        case "stdout": ec.stdoutBuf = true
        case "stderr": ec.stderrBuf = true
        }
    }

    switch ec.tie {
    case "stdout", "out": ec.stdoutTie = true
    case "stderr", "err": ec.stderrTie = true
    case "all", "both":
        ec.stdoutTie = true
        ec.stderrTie = true
    }

    if t := auto_target_value(ctx); t.patterned(ctx) {
        errostack(ctx, 5, "target is pattern: %v", ec.target).trace()
    } else {
        ec.target.Value = t
        if _, y := t.(flag); !y {
            ec.targetName, _ = ec.target.fullname_string(ctx)
        }
    }

    for i, v := range args {
        var s string
        if i == 0 && p.contained {
            if s = v.string(ctx); s == "shell" { cmd = defaultShell }
        } else if s = strings.TrimSpace(v.string(ctx)); s != "" {
            ec.args = append(ec.args, s)
        }
    }

    if p.contained {
        var proj = _project(ctx)
        if proj == nil {
            erro(ctx, "nil project").trace()
        }

        if proj.name == dot_container {
            ec.container = proj
        } else if _, sym := proj.find(dot_container); sym != nil {
            ec.container = sym.(*project)
        }
        if ec.container == nil {
            erro(ctx, "%s: nil container", proj.name).trace()
        }

        var stringify = func(name string) (str string) {
            var ctx = closure_with(ctx, ec.container.scope)
            if obj := ec.container.resolve(ctx, name); obj != nil {
                if d, _ := obj.(*def); d != nil {
                    if v := d.invoke(ctx, nil, nil); v != nil {
                        if str = v.string(ctx); str == "-" {
                            // if v, err = def.DiscloseValue(ec.container); err == nil && v != nil {
                            //   if str, err = v.string(ctx); str == "" { str = "-" }
                            //   prompt(ctx, "%v: %v (%v)\n", name, str, def)
                            // }
                        }
                    }
                }
            }
            return
        }

        var containerName = stringify("container")
        if containerName == "" {
            erro(ctx, ".container.name undefined").trace()
        }

        var containerImage = stringify("image")
        if containerImage == "" {
            erro(ctx, ".container.image undefined").trace()
        }

        ec.args = append(ec.args, "exec", containerName, cmd)
        cmd = "docker"
    }

    if ec._workdir == "" {
        ec._workdir = prog.workdir(ctx)
        if ec._workdir == "" {
            erro(ctx, "workdir is empty").trace()
        }
    }

    if ec.path {
        if s := filepath.Dir(ec.targetName); s != "" && s != "." && s != "/" {
            if e := os.MkdirAll(s, os.FileMode(0755)); e != nil {
                erro(ctx, "make path '%s' for target failed: %v", s, e).trace()
            }
        }
    }

    ec.exec(cmd, p.opt)

    if ec.result != nil {
        if resValue != nil {
            var s string
            switch resKind {
            case "stdout": s = trim(ec.Stdout.Buf.String())
            case "stderr": s = trim(ec.Stderr.Buf.String())
            case "status": s = fmt.Sprintf("%v", ec.Status)
            }
            switch v := resValue.string(ctx) == s ; resType {
            case "answer": return _answer(_position(ctx), v)
            case "option": return _option(_position(ctx), v)
            default:       return _boolean(_position(ctx), v)
            }
        } else {
            switch resKind {
            case "stdout": return _raw(_position(ctx), trim(ec.Stdout.Buf.String()))
            case "stderr": return _raw(_position(ctx), trim(ec.Stderr.Buf.String()))
            case "status": return _decimal(_position(ctx), int64(ec.Status))
            }
        }
    }
    return &ec.exec_result
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

fieldsloop:
    for i := 1; i < len(fields); i += 1 {
      var field = fields[i]

      for rx, rxs := range execExistFlagPath {
        if rx.MatchString(fields[0]) {
          for _, rx := range rxs {
            var m = rx.FindStringSubmatch(field)
            if len(m) == 0 { continue }

            var f bool
            var s = m[2]
            if s == "" {
              if i += 1; i == len(fields) { break fieldsloop }
              s, f = fields[i], true
            }

            if _, e := os.Stat(s); e != nil {
              if w { warn(ctx, "ignoring nonexistent path: %v", s).debug() }
              continue fieldsloop // skip nonexistent path flags
            } else if f {
              flags = append(flags, field)
              field = s
            }
          }
        }
      }

      flags = append(flags, field)
    }
    return strings.Join(flags, " ")
}
