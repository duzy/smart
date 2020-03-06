//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/scanner"
        "path/filepath"
        "sync/atomic"
        "os/exec"
        "strings"
        "strconv"
        "regexp"
        "bytes"
        "bufio"
        "sync"
        "time"
        "fmt"
        "io"
        "os"
)

// Note that it's is also used with Sscanf.
const (
        exitstatusFmt = "exit status %d"
        maxPromptStr = 48
)

type exitstatus struct { code int }
func (e *exitstatus) Error() string {
        return fmt.Sprintf(exitstatusFmt, e.code)
}

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
)
var (
        defaultShell = "bash"

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
        }

        workingMutex = new(sync.Mutex)
        working atomic.Value // number of working executions

        stdout = &stdWriter{ std:os.Stdout }
        stderr = &stdWriter{ std:os.Stderr }
        udots = []byte("…")
)

const (
        maxRetries = 1
        maxWorkers = 10
)

func init() {
        working.Store(0)
}

func checkForWork() (good bool) {
        workingMutex.Lock()
        defer workingMutex.Unlock()

        var num = working.Load().(int)
        if num < maxWorkers {
                working.Store(num + 1)
                good = true
        }
        return
}

func waitForWork() {
        for {
                if checkForWork() { break }
                time.Sleep(5*time.Millisecond)
        }
}

func releaseWork() {
        workingMutex.Lock()
        defer workingMutex.Unlock()
        var num = working.Load().(int)
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
        n, err = w.std.Write(p)
        if bytes.HasSuffix(p, udots) {
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
        p.wrimux.Lock()
        defer p.wrimux.Unlock()

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

type knownMatch struct {
        i, l int
        v [][]string // groups of captures
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

func (p *ExecBuffer) filter(s string) {
        p.filters = append(p.filters, s)
}

func (p *ExecBuffer) Write(b []byte) (n int, err error) {
        for _, s := range p.filters {
                if bytes.Equal(b, []byte(s)) { // string(b) == s
                        return len(b), nil
                }
        }
        if p.log != nil {
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
        var l int
        if p.log != nil { l = p.log.lines }
        for slice := b[:]; len(slice) > 0; {
                var i = bytes.Index(slice, []byte("\n"))
                if i == -1 {
                        p.line.Write(slice)
                        slice = nil
                } else {
                        p.line.Write(slice[:i+1])
                        slice = slice[i+1:]

                        var line = p.line.Bytes()
                        for i, rx := range knownerrors {
                                if rx == nil { continue }
                                if all := rx.FindAllSubmatch(line, -1); all != nil {
                                        var a [][]string
                                        for _, m := range all { // [][][]byte
                                                var v []string // captures
                                                for _, cap := range m {
                                                        v = append(v, string(cap))
                                                }
                                                a = append(a, v)
                                        }
                                        p.matches = append(p.matches, knownMatch{ i, l, a })
                                }
                        }

                        p.line.Reset()
                        l += 1
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
                err = wrap(pos, fmt.Errorf("dokcer daemon not running (at %s)", sock), err)
        } else {
                // TODO: start docker daemon
        }
        return
}

func (p *ExecBuffer) runContainerAndRetry(pos Position, t *traversal, container *Project, name string, sh *exec.Cmd, x *executor, num int) (err error) {
        if container != nil && num <= maxRetries {
                fmt.Fprintf(sh.Stderr, "\n---- Run the container: %s", name)
                if err = x.runContainer(t, container); err != nil {
                        err = wrap(pos, errorf(pos, "container not running: %v", name), err)
                        return
                }

                fmt.Fprintf(sh.Stderr, "\n---- Retry the command:")
                fmt.Fprintf(sh.Stderr, "\n---- %v\n", sh)
                fmt.Fprintf(sh.Stderr, "\n----\n")
                c := exec.Command("docker", sh.Args...)
                c.Stdout, c.Stderr, c.Stdin, c.Env = sh.Stdout, sh.Stderr, sh.Stdin, sh.Env
                _, err = p.runAndProcessKnownErrors(pos, t, container, c, x, num+1)
        }
        return
}

func (p *ExecBuffer) processKnownError(pos Position, t *traversal, container *Project, sh *exec.Cmd, x *executor, status, num int, m *knownMatch) (err error) {
        var lpos Position
        lpos.Filename, lpos.Line = p.log.filename, m.l
        for _, v := range m.v { // captures
                switch m.i {
                case rxNotTTYDevice_i:
                        err = errorf(lpos, "Needs TTY (input device)")
                case rxDockerDaemonNotRunning_i:
                        err = p.startDockerDaemon(lpos, t, container, string(v[1]))
                        if err != nil { err = wrap(pos, err) }
                case rxNoContainer_i:
                        if name := string(v[1]); p.skips(name) {
                                err = errorf(lpos, "container not running: %v", name)
                        } else if err = p.runContainerAndRetry(lpos, t, container, name, sh, x, num); err == nil {
                                p.retried[name] = true // save it to skip next time
                        }
                case rxContainerNotRunning_i:
                        err = errorf(lpos, "Container not running (%v)", string(v[1]))
                case rxNoNetwork_i:
                        err = errorf(lpos, "Network not found (%v)", string(v[1]))
                case rxCompilation_i:
                        var pos Position
                        pos.Filename = string(v[1])
                        pos.Line, _ = strconv.Atoi(string(v[2]))
                        pos.Column, _ = strconv.Atoi(string(v[3]))
                        err = wrap(pos, errorf(lpos, "%s", string(v[4])))
                case rxIncludedFrom_i:
                        if p.report { fmt.Fprintf(stderr, "%s:%s:%s: included here\n", v[1], v[2], v[3]) }
                case rxFileNotFound_i:
                        err = errorf(lpos, "`%v` file not found, required by `%s` (exec)", v[4], filepath.Base(string(v[1])))
                        if p.report { fmt.Fprintf(stderr, "%s:%s:%s: exec: `%s` file not found\n", v[1], v[2], v[3], v[4]) }
                case rxArNoSuchFile_i:
                        err = errorf(lpos, "`%v` file not found", filepath.Base(string(v[1])))
                        if p.report { fmt.Fprintf(stderr, "exec: (ar): '%s' not found (as '%s')", filepath.Base(string(v[1])), v[1]) }
                case rxBashNoSuchFile_i:
                        err = errorf(lpos, "%v: no such command", string(v[1]))
                case rxClangNoSuchFile_i:
                        err = errorf(lpos, "clang-%s: no such source file: %s", string(v[1]), string(v[2]))
                case rxClangError_i:
                        err = errorf(lpos, "clang-%s: %s", string(v[1]), string(v[2]))
                case rxLLDError_i:
                        err = errorf(lpos, "%s", string(v[2]))
                case rxCouldnotParseObj_i:
                        err = errorf(lpos, "%s", string(v[3]))
                case rxTooManyPosArgs_i:
                        err = errorf(lpos, "%s: too many positional arguments", string(v[1]))
                case rxLLDWarning_i:
                        if p.report {
                                fmt.Fprintf(stderr, "%s: warning: %s\n", lpos, string(v[2]))
                                fmt.Fprintf(stderr, "%s: warning: …from here\n", pos)
                        }
                }
                if err != nil { break }
        }
        return
}

func (p *ExecBuffer) processKnownErrors(pos Position, t *traversal, container *Project, sh *exec.Cmd, x *executor, status, num int) (err error) {
        for _, m := range p.matches {
                err = p.processKnownError(pos, t, container, sh, x, status, num, &m)
                if err != nil { err = wrap(pos, err); break }
        }
        if err == nil && status != 0 { err = &exitstatus{ status }}
        return
}

func (p *ExecBuffer) runAndProcessKnownErrors(pos Position, t *traversal, dock *Project, sh *exec.Cmd, x *executor, num int) (status int, err error) {
        p.matches = nil
        if err = sh.Run(); err == nil {
                // It's good!
        } else if n, e := fmt.Sscanf(err.Error(), exitstatusFmt, &status); n == 1 && e == nil {
                err = &exitstatus{ status } // convert to exitstatus

                if p.log != nil && p.log.writer != nil {
                        fmt.Fprintf(p.log, "\n%s\n", err)

                        var pos Position
                        pos.Filename = p.log.filename
                        pos.Offset = 0 // FIXME: what should be the offset?
                        pos.Line = p.log.lines
                        pos.Column = 0
                        err = wrap(pos, err)
                }

                if e := p.processKnownErrors(pos, t, dock, sh, x, status, num); e != nil {
                        err = wrap(pos, e, err)
                }
        } else {
                if status == 0 { status = -1 }
                if e != nil { err = e }
        }
        return
}

type ExecResult struct {
        trivial
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
func (p *ExecResult) True() (bool, error) { return p.Status == 0 && p.Stderr.Buf.Len() == 0 /* && p.Stdout.Buf.Len() > 0 */, nil }
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

func (p *executor) runContainer(t *traversal, container *Project) (err error) {
        if run, _ := container.resolveEntry("run"); run != nil && len(run.programs) > 0 {
                defer setclosure(setclosure(cloctx.unshift(container.scope)))
                if _, err = run.programs[0].execute(t, run, nil); err != nil {
                        err = wrap(t.program.position, err)
                } else { t.group.Wait() }
        } else {
                err = errorf(t.program.position, "%s⇒run undefined", container)
        }
        return
}

func (p *executor) ensureContainerRunning(t *traversal, dock *Project, container string) (err error) {
        var (
                stdoutR, stdoutW = io.Pipe()
                stderrR, stderrW = io.Pipe()
                enviro = os.Environ()
                cmd = exec.Command(`docker`, `ps`,
                        `--filter`, `status=running`,
                        //`--filter`, fmt.Sprintf(`ancestor=%s`, image),
                        `--filter`, fmt.Sprintf(`name=%s`, container),
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
                if err = p.runContainer(t, dock); err == nil {
                        time.Sleep(time.Second)
                }
        }
        return
}

func (p *executor) Evaluate(pos Position, t *traversal, args ...Value) (result Value, err error) {
        if optionTraceExecutor {
                var t = t.def.target.value
                defer un(trace(t_exec, fmt.Sprintf("executor(%s %v)", typeof(t), t)))
        }

        var (
                optPrompt, optVerbout, optVerberr, optDebug bool
                optBuffOut, optBuffErr, optStdin bool
                optSilent, optNoCD, optPath bool
                optScanStderr bool = true
                promStr, logFileName string
                cmd = p.cmd
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "c,cmd", // replaces -p, -prompt
                "d,dump", // verbout, verberr
                "g,debug",
                "o,stdout",
                "e,stderr",
                "i,stdin",
                "l,log",
                "n,nocd",
                "p,path",
                "s,silent", // report nothing, discard errors
                "v,verbout",
                "w,verberr",
        }, func(ru rune, v Value) {
                var s string
                switch ru {
                case 'i': if optStdin   , err = trueVal(v, true); err != nil { return }
                case 'o': if optBuffOut , err = trueVal(v, true); err != nil { return }
                case 'e': if optBuffErr , err = trueVal(v, true); err != nil { return }
                case 'v': if optVerbout , err = trueVal(v, true); err != nil { return }
                case 'w': if optVerberr , err = trueVal(v, true); err != nil { return }
                case 's': if optSilent  , err = trueVal(v, true); err != nil { return }
                case 'g': if optDebug   , err = trueVal(v, true); err != nil { return }
                case 'p': if optPath    , err = trueVal(v, true); err != nil { return }
                        if p, ok := v.(*Pair); ok {
                                fmt.Printf("%s: -p=xxx has been replaced with -c (-cmd), -p is no -path", p.Value.Position())
                        }
                case 'c':
                        if v == nil {
                                optPrompt = true
                        } else if s, err = v.Strval(); err == nil {
                                optPrompt, promStr = true, s
                        } else {
                                return
                        }
                case 'l': // logFileName
                        if v == nil {
                                logFileName = ""
                        } else if s, err = v.Strval(); err == nil {
                                logFileName = s
                        } else {
                                return
                        }
                case 'd': // -dump=xxx or -d=xxx
                        if v == nil {
                                optVerbout, optVerberr = true, true
                        } else if s, err = v.Strval(); err == nil {
                                switch s {
                                case "stdout": optVerbout = true
                                case "stderr": optVerberr = true
                                case "all":
                                        optVerbout = true
                                        optVerberr = true
                                }
                        } else {
                                return
                        }
                case 'n':
                        optNoCD = true
                }
        }); err != nil { return }

        var aa []string
        for i, v := range args {
                var s string
                if p.contained && i == 0 {
                        if s, err = v.Strval(); err != nil { return }
                        if s == "shell" { cmd = defaultShell }
                        continue
                }
                if s, err = v.Strval(); err != nil { return } else {
                        aa = append(aa, s)
                }
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
                } else {
                        if _, containerSym := t.program.project.scope.Find(dotContainer); containerSym != nil {
                                if pn, _ := containerSym.(*ProjectName); pn != nil {
                                        container = pn.NamedProject()
                                }
                        }
                }

                if container == nil {
                        err = fmt.Errorf("container unavailable (in %s)", t.program.Project().Name())
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
                if containerName, err = strval("container"); err != nil { return }
                if containerName == "" { err = fmt.Errorf(".container.name undefined"); return }
                if containerImage, err = strval("image"); err != nil { return }
                if containerImage == "" { err = fmt.Errorf(".container.image undefined"); return }

                fmt.Fprintf(stderr, "%v: %v (%v)\n", container, containerName, containerImage)

                aa = append(aa, "exec", containerName, cmd)
                cmd = "docker"

                if false {
                        if err = p.ensureContainerRunning(t, container, containerName); err != nil {
                                return
                        }
                }
        }

        var cwd string
        if v, e := t.program.scope.Lookup("CWD").(*Def).Call(t.program.position); e != nil { err = e; return } else
        if v != nil { if cwd, err = v.Strval(); err != nil { return }} else
        if v, e := t.program.scope.Lookup("/").(*Def).Call(t.program.position); e != nil { err = e; return } else
        if v != nil { if cwd, err = v.Strval(); err != nil { return }}

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
        var target = t.def.target.value
        if targetName, err = target.Strval(); err != nil { return }
        if optPath {
                var s string
                if s = filepath.Dir(targetName); s != "" && s != "." && s != "/" {
                        err = os.MkdirAll(s, os.FileMode(0755))
                        if err != nil { return }
                }
        }

        var envars []*Pair // disclosed values
        if def, _ := t.program.Scope().Lookup(TheShellEnvarsDef).(*Def); def != nil {
                if l, _ := def.value.(*List); l != nil {
                        for _, v := range l.Elems {
                                if v, err = v.expand(expandClosure); err != nil {
                                        return
                                } else if p, ok := v.(*Pair); ok {
                                        envars = append(envars, p)
                                } else {
                                        err = fmt.Errorf("env expecting pairs (%T)", v)
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
        if recipes, err = mergeresult(ExpandAll(t.program.recipes...)); err != nil { return }
        for _, recipe := range recipes {
                if !rp.IsValid() { rp = recipe.Position() }
                if str, err = recipe.Strval(); err != nil { return }
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
                if k, err = p.Key.Strval(); err != nil { return }
                if v, err = p.Value.Strval(); err != nil { return }
                if i > 0 { envstr += " && " }
                envstr += fmt.Sprintf(`%s=%s`, k, strconv.Quote(v))
                envs = append(envs, fmt.Sprintf("%s=%s", k, v))
        }

        var log ExecLog
        var logfile *os.File
        var exeres = &ExecResult{trivial:trivial{pos},wg:new(sync.WaitGroup)}
        if optBuffOut { exeres.Stdout.Buf = new(bytes.Buffer) }
        if optBuffErr { exeres.Stderr.Buf = new(bytes.Buffer) }
        if optVerbout { exeres.Stdout.Tie = stdout }
        if optVerberr { exeres.Stderr.Tie = stderr }
        if logFileName == "" {
                // no log required
        } else if err = os.MkdirAll(filepath.Dir(logFileName), os.FileMode(0755)); err != nil {
                err = wrap(t.program.position, err)
                return // FIXME: err for outer func
        } else if logfile, err = os.Create(logFileName); err != nil {
                err = wrap(t.program.position, err)
                return // FIXME: err for outer func
        } else {
                cmdline := strings.Join(sources, "\n")
                log.createWriter(logfile, dir, cmdline)
                exeres.Stdout.log = &log
                exeres.Stderr.log = &log
        }

        exeres.Stderr.scanerr = optScanStderr
        log.filename = logFileName

        var run = func() {
                var targetStr string
                defer func(start time.Time) {
                        if err == nil { err = stamp(t, target, start, optPrompt) }
                        if log.writer != nil {
                                if false && exeres.Stdout.wrote == 0 && exeres.Stderr.wrote == 0 {
                                        // Discard log buffer.
                                        logfile.Close()
                                        os.Remove(logFileName)
                                } else {
                                        log.writer.Flush()
                                        logfile.Close()
                                }
                        }
                        if c := t.caller; c != nil { c.calleeDone(err) }
                        if optPrompt {
                                if t.caller == nil {
                                        if err == nil {
                                                fmt.Fprintf(stderr, "… ok\n")
                                        } else if _, ok := err.(*scanner.Error); ok {
                                                fmt.Fprintf(stderr, " error:\n%v\n", err)
                                        } else {
                                                fmt.Fprintf(stderr, " error: %v\n", err)
                                        }
                                } else {
                                        if err == nil {
                                                if false { fmt.Fprintf(stderr, "%s%s, okay.\n", promStr, targetStr) }
                                        } else if _, ok := err.(*scanner.Error); ok {
                                                fmt.Fprintf(stderr, "%s%s, error:\n%v\n", promStr, targetStr, err)
                                        } else {
                                                fmt.Fprintf(stderr, "%s%s, error: %v\n", promStr, targetStr, err)
                                        }
                                }
                        }
                        exeres.wg.Done()
                } (time.Now())
                if optPrompt {
                        targetStr = trimPromptString(targetName)
                        if promStr == "" {
                                promStr = "smart: gen "
                        } else {
                                promStr += ": "
                        }
                        if t.caller == nil {
                                fmt.Fprintf(stderr, "%s%s …\n", promStr, targetStr)
                        } else { // ……
                                fmt.Fprintf(stderr, "%s%s\n", promStr, targetStr)
                        }
                }
                if optDebug { fmt.Fprintf(stderr, "%s: %v (%v)\n", pos, cmd, t.def.target.value) }
                for i, src := range sources {
                        var pos = positions[i]
                        if false { fmt.Fprintf(stderr, "%s: %v\n", pos, src) }
                        if strings.HasPrefix(src, "@") {
                                src = src[1:]
                        } else if !optPrompt {
                                var s string
                                s = strings.Replace(src, "\n", "\\n", -1)
                                s = strings.Replace(s, "\\\\n", "\\\n", -1)
                                fmt.Fprintf(stderr, "%s\n", s)
                        }
                        if src = strings.TrimSpace(src); src == "" {
                                continue
                        } else if dir != "" && !optNoCD /*&& t.program.changedWD == ""*/ {
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

                        if optionNoExec { continue }

                        // Restricts the number of workers.
                        waitForWork(); defer releaseWork()

                        lockCD(dir, 25*time.Millisecond)
                        if s, _ := os.Getwd(); s != dir {
                                assert(s == dir, "wrong work directory (%s != %s)", s, dir)
                        }

                        var sh = exec.Command(cmd, aa...)
                        sh.Dir = dir // always set command work directory
                        sh.Env = envs
                        sh.Stdout = &exeres.Stdout
                        sh.Stderr = &exeres.Stderr
                        if optStdin {
                                sh.Stdin = os.Stdin
                                sh.Args = append(sh.Args, "-ti")
                        }
                        sh.Args = append(sh.Args, p.opt, src)

                        if optDebug { fmt.Fprintf(stderr, "%s: %v\n", pos, sh) }

                        exeres.Stderr.report = !optSilent
                        exeres.Status, err = exeres.Stderr.runAndProcessKnownErrors(pos, t, container, sh, p, 1)
                        if err != nil {
                                if false { fmt.Fprintf(stderr, "%v\n", wrap(pos, err)) }
                                if optSilent { err = nil } else {
                                        err = wrap(pos, err)
                                        return
                                }
                        }
                }
        }

        if !optSilent { printEnteringDirectory() }
        if t.caller != nil { t.caller.calleeStart() }
        exeres.wg.Add(1); go run()
        if t.caller == nil { exeres.wg.Wait() }

        // The execution is performed asynchronously, the result can't
        // be fetched immediately. Caller should t.wait(...) or
        // exeres.wait() before using the result.
        result = exeres
        return
}

func stamp(t *traversal, target Value, start time.Time, verb bool) (err error) {
        var v Value
        var files []*File
        if v, err = target.expand(expandAll); err != nil { return } else
        if files, err = v.stamp(t); err == nil && verb {
                for _, file := range files {
                        d := file.info.ModTime().Sub(start);
                        fmt.Printf("smart: Updated %v (%v)\n", file, d)
                }
        }
        return
}
