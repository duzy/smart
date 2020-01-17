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
const exitstatusFmt = "exit status %d"

type exitstatus struct { code int }

func (e *exitstatus) Error() string {
        return fmt.Sprintf(exitstatusFmt, e.code)
}

const (
        rxNotTTYDevice_i int = iota
        rxNoContainer_i
        rxNoNetwork_i
        rxCompilation_i
        rxIncludedFrom_i
        rxFileNotFound_i
        rxArNoSuchFile_i
        rxBashNoSuchFile_i
)
var (
        defaultShell = "bash"

        errNotTTYDevice = `the input device is not a TTY`
        errNoContainer = `Error.*: No such container: (.*)`
        errNoNetwork = `Error.*: network (.*) not found\.`

        errCompilation = `(.+?):(\d+):(\d+): error: (.+)`
        errIncludedFrom = `In file included from (.+?):(\d+):(\d+):`
        errFileNotFound = `(.+?):(\d+):(\d+): fatal error: '(.+?)' file not found`
        errArNoSuchFile = `ar: (.+?): No such file or directory`
        errBashNoSuchFile = `bash: (.+?): No such file or directory`

        rxNotTTYDevice = regexp.MustCompile(errNotTTYDevice)
        rxNoContainer = regexp.MustCompile(errNoContainer)
        rxNoNetwork = regexp.MustCompile(errNoNetwork)
        rxCompilation = regexp.MustCompile(errCompilation)
        rxIncludedFrom = regexp.MustCompile(errIncludedFrom)
        rxFileNotFound = regexp.MustCompile(errFileNotFound)
        rxArNoSuchFile = regexp.MustCompile(errArNoSuchFile)
        rxBashNoSuchFile = regexp.MustCompile(errBashNoSuchFile)

        knownerrors = []*regexp.Regexp{
                rxNotTTYDevice_i:   rxNotTTYDevice,
                rxNoContainer_i:    rxNoContainer,
                rxNoNetwork_i:      rxNoNetwork,
                rxCompilation_i:    rxCompilation,
                rxIncludedFrom_i:   rxIncludedFrom,
                rxFileNotFound_i:   rxFileNotFound,
                rxArNoSuchFile_i:   rxArNoSuchFile,
                rxBashNoSuchFile_i: rxBashNoSuchFile,
        }

        workingMutex = new(sync.Mutex)
        working atomic.Value // number of working executions

        stdmux = &sync.Mutex{}
        stdout = &stdWriter{ std:os.Stdout }
        stderr = &stdWriter{ std:os.Stderr }
        dots = []byte("…")
)

const (
        maxRetries = 2
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

type stdWriter struct {
        std io.Writer
        suffixDots bool
}

func (w *stdWriter) Write(p []byte) (n int, err error) {
        stdmux.Lock(); defer stdmux.Unlock()
        if w.suffixDots {
                if !bytes.HasPrefix(p, dots) {
                        w.std.Write([]byte("\n"))
                }
                w.suffixDots = false
        }
        n, err = w.std.Write(p)
        if bytes.HasSuffix(p, dots) {
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
        i int
        v [][]string // groups of captures
}

type ExecBuffer struct {
        Tie io.Writer
        Buf *bytes.Buffer
        log *ExecLog
        scanerr bool
        matches []knownMatch
        line bytes.Buffer
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
                                        p.matches = append(p.matches, knownMatch{ i, a })
                                }
                        }

                        p.line.Reset()
                }
        }
        return
}

func (p *ExecBuffer) skips(tag string) (result bool) {
        if p.retried == nil {
                p.retried = make(map[string]bool)
        } else {
                a, b := p.retried[tag]
                result = a && b
        }
        return
}

func (p *ExecBuffer) processKnownErrors(pos Position, pc *traversal, dock *Project, sh *exec.Cmd, x *executor, status, num int) (err error) {
        var retry bool
        var tag string
        for _, m := range p.matches {
                for _, v := range m.v { // captures
                        switch m.i {
                        case rxNotTTYDevice_i:
                                retry = true
                        case rxNoContainer_i:
                                tag = string(v[1])
                        case rxNoNetwork_i:
                                // TODO: ...
                        case rxCompilation_i:
                                // TODO: ...
                        case rxIncludedFrom_i:
                                if p.report { fmt.Fprintf(stderr, "%s:%s:%s: included here\n", v[1], v[2], v[3]) }
                        case rxFileNotFound_i:
                                if p.report { fmt.Fprintf(stderr, "%s:%s:%s: exec: `%s` file not found\n", v[1], v[2], v[3], v[4]) }
                                err = wrap(pos, fmt.Errorf("`%v` file not found, required by `%s` (exec)", v[4], filepath.Base(string(v[1]))), err)
                        case rxArNoSuchFile_i:
                                if p.report { fmt.Fprintf(stderr, "exec: (ar): '%s' not found (as '%s')", filepath.Base(string(v[1])), v[1]) }
                                err = wrap(pos, fmt.Errorf("`%v` file not found", filepath.Base(string(v[1]))), err)
                        case rxBashNoSuchFile_i:
                                err = wrap(pos, fmt.Errorf("%v: no such command", string(v[1])), err)
                        }
                }
        }
        if err != nil { return }
        if !p.scanerr || p.matches == nil {
                //err = fmt.Errorf(...)
        }

        if err == nil && retry && num < maxRetries {
                if p.report { fmt.Fprintf(stderr, "smart: good to retry (num = %d)\n", num) }
                c := exec.Command(sh.Path, sh.Args...)
                c.Stdout, c.Stderr, c.Stdin, c.Env = sh.Stdout, sh.Stderr, sh.Stdin, sh.Env
                _, err = p.runAndProcessKnownErrors(pos, pc, dock, c, x, num+1) // retry
        } else if err != nil {
                // ends with error
        } else if tag == "" {
                err = &exitstatus{ status }
        } else if skip := p.skips(tag); !skip && dock != nil && num < maxRetries {
                p.retried[tag] = true // save it to skip next time
                if err = x.runContainer(pc, dock); err == nil {
                        if p.report { fmt.Fprintf(stderr, "smart: started %s\n", tag) }
                        c := exec.Command(sh.Path, sh.Args...)
                        c.Stdout, c.Stderr, c.Stdin, c.Env = sh.Stdout, sh.Stderr, sh.Stdin, sh.Env
                        _, err = p.runAndProcessKnownErrors(pos, pc, dock, c, x, num+1) // retry
                } else {
                        //err = errorf(pos, "`%s` no such container", tag)
                }
        }
        return
}

func (p *ExecBuffer) runAndProcessKnownErrors(pos Position, pc *traversal, dock *Project, sh *exec.Cmd, x *executor, num int) (status int, err error) {
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
                if e := p.processKnownErrors(pos, pc, dock, sh, x, status, num); e != nil {
                        err = wrap(pos, e, err)
                }
                //if p.report { fmt.Fprintf(stderr, "%v\n", err) }
        } else {
                if status == 0 { status = -1 }
                if e != nil { err = e }
        }
        return
}

type ExecResult struct {
        trivial
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
        if p.Stdout.Buf != nil {
                s = p.Stdout.Buf.String()
        }
        return
}
func (p *ExecResult) String() string {
        var s bytes.Buffer
        fmt.Fprintf(&s, "(ExecResult status=%d", p.Status)
        if p.Stdout.Buf != nil {
                fmt.Fprintf(&s, " stdout=%S", p.Stdout.Buf)
        }
        if p.Stderr.Buf != nil {
                fmt.Fprintf(&s, " stdout=%S", p.Stderr.Buf)
        }
        fmt.Fprintf(&s, ")")
        return s.String()
}

type executor struct {
        cmd, opt string
        bare bool
}

func (p *executor) runContainer(pc *traversal, dock *Project) (err error) {
        if run, _ := dock.resolveEntry("run"); run != nil {
                if run.OwnerProject() != dock {
                        err = fmt.Errorf("'%v' must have 'run' rule", dock)
                        fmt.Fprintf(stderr, "%v: %v\n", dock.absPath, err)
                } else {
                        _, err = run.Execute(pc.program.position)
                }
                if err != nil {
                        fmt.Fprintf(stderr, "%v: %v\n", pc.program.position, err)
                }

        } else {
                err = fmt.Errorf("dock⇒run undefined")
        }
        return
}

func (p *executor) ensureContainerRunning(pc *traversal, dock *Project, container string) (err error) {
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
                if err = p.runContainer(pc, dock); err == nil {
                        time.Sleep(time.Second)
                }
        }
        return
}

func (p *executor) Evaluate(pc *traversal, args []Value) (result Value, err error) {
        if optionTraceExecutor {
                var t = pc.def.target.value
                defer un(trace(t_executor, fmt.Sprintf("%s: %v (depth=%d)", typeof(t), t, pc.depth())))
        }

        var recursion int
        for c := pc.caller; c != nil; c = c.caller {
                if c.def.target.value == pc.def.target.value {
                        recursion += 1
                }
        }
        if recursion > 1 {
                var pos = pc.caller.program.Position()
                err = errorf(pos, "%v: too many recursion (%d)", pc.def.target.value, recursion)
                return
        }

        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        var prompt, verbout, verberr, buffout, bufferr, stdin, silent, nocd bool
        var cmd, promStr, logFileName = p.cmd, "", ""
        var optPath bool
        var aa []string
        if args, err = parseFlags(args, []string{
                "c,cmd", // replaces -p, -prompt
                "d,dump", // verbout, verberr
                "e,stderr",
                "i,stdin",
                "l,log",
                "n,nocd",
                "o,stdout",
                "p,path",
                "s,silent",
                "v,verbout",
                "w,verberr",
        }, func(ru rune, v Value) {
                var s string
                switch ru {
                case 'i': stdin   = true
                case 'o': buffout = true
                case 'e': bufferr = true
                case 'v': verbout = true
                case 'w': verberr = true
                case 's': silent  = true
                case 'p': optPath = trueVal(v, false)
                        if p, ok := v.(*Pair); ok {
                                fmt.Printf("%s: -p=xxx has been replaced with -c (-cmd), -p is no -path", p.Value.Position())
                        }
                case 'c':
                        if v == nil {
                                prompt = true
                        } else if s, err = v.Strval(); err == nil {
                                prompt, promStr = true, s
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
                                verbout, verberr = true, true
                        } else if s, err = v.Strval(); err == nil {
                                switch s {
                                case "stdout": verbout = true
                                case "stderr": verberr = true
                                case "all":
                                        verbout = true
                                        verberr = true
                                }
                        } else {
                                return
                        }
                case 'n':
                        nocd = true
                }
        }); err != nil { return }

        for i, v := range args {
                var s string
                if !p.bare && i == 0 {
                        if s, err = v.Strval(); err != nil { return }
                        if s == "shell" { cmd = defaultShell }
                        continue
                }
                if s, err = v.Strval(); err != nil { return } else {
                        aa = append(aa, s)
                }
        }

        var dock *Project
        if !p.bare {
                if pc.program.project.name == ".dock" {
                        dock = pc.program.project
                } else if false {
                        for _, scope := range cloctx {
                                if _, sym := scope.Find(".dock"); sym != nil {
                                        if p, ok := sym.(*ProjectName); ok && p != nil {
                                                dock = p.NamedProject()
                                                break
                                        }
                                }
                        }
                        if dock == nil {
                                if _, dockSym := pc.program.project.scope.Find(".dock"); dockSym != nil {
                                        if pn, _ := dockSym.(*ProjectName); pn != nil {
                                                dock = pn.NamedProject()
                                        }
                                }
                        }
                } else {
                        if _, dockSym := pc.program.project.scope.Find(".dock"); dockSym != nil {
                                if pn, _ := dockSym.(*ProjectName); pn != nil {
                                        dock = pn.NamedProject()
                                }
                        }
                }

                // fmt.Fprintf(stderr, "%v: %v\n", dock, dock.absPath)

                if dock == nil {
                        err = fmt.Errorf("docking unavailable (in %s)", pc.program.Project().Name())
                        return
                }

                var strval = func(name string) (str string, err error) {
                        if false {
                                defer setclosure(scoping(dock)) //(scoping(docks...))
                        } else {
                                defer setclosure(cloctx)
                                cloctx = append(closurecontext{dock.Scope()}, cloctx...)
                        }
                        if obj, _ := dock.resolveObject(name); obj != nil {
                                if def, _ := obj.(*Def); def != nil {
                                        var v Value
                                        if v, err = def.DiscloseValue(); err == nil && v != nil {
                                                if str, err = v.Strval(); str == "-" {
                                                        /*if v, err = def.DiscloseValue(dock); err == nil && v != nil {
                                                        if str, err = v.Strval(); str == "" { str = "-" }
                                                        fmt.Fprintf(stderr, "%v: %v (%v)\n", name, str, def)
                                                        }*/
                                                }
                                        }
                                }
                        }
                        return
                }

                var container, image string
                if container, err = strval("container"); err != nil { return }
                if container == "" { err = fmt.Errorf("container undefined"); return }
                if image, err = strval("image"); err != nil { return }
                if image == "" { err = fmt.Errorf("image undefined"); return }

                fmt.Fprintf(stderr, "%v: %v (%v)\n", dock, container, image)

                aa = append(aa, "exec", container, cmd)
                cmd = "docker"

                if false {
                        if err = p.ensureContainerRunning(pc, dock, container); err != nil {
                                return
                        }
                }
        }

        var cwd string
        if v, e := pc.program.scope.Lookup("CWD").(*Def).Call(pc.program.position); e != nil {
                err = e; return
        } else if v != nil {
                if cwd, err = v.Strval(); err != nil { return }
        } else if v, e := pc.program.scope.Lookup("/").(*Def).Call(pc.program.position); e != nil {
                err = e; return
        } else if v != nil {
                if cwd, err = v.Strval(); err != nil { return }
        }

        // Fixes work directory conflicts. It happens
        // sometimes even the 'sh.Dir' is set to cwd.
        // Because the current work directory is not
        // thread safe.
        var dir = cwd
        if pc.program.changedWD != "" {
                if filepath.IsAbs(pc.program.changedWD) {
                        dir = pc.program.changedWD
                } else {
                        dir = filepath.Join(pc.program.project.absPath, pc.program.changedWD)
                }
        }

        var targetName string
        var target = pc.def.target.value
        if targetName, err = target.Strval(); err != nil {
                return
        }

        if optPath {
                var s string
                if s, err = pc.def.target.value.Strval(); err != nil { return }
                if s = filepath.Dir(s); s != "" && s != "." && s != "/" {
                        err = os.MkdirAll(s, os.FileMode(0755))
                        if err != nil { return }
                }
        }

        var envars []*Pair // disclosed values
        if def, _ := pc.program.Scope().Lookup(TheShellEnvarsDef).(*Def); def != nil {
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

        var recipes []Value
        if recipes, err = mergeresult(ExpandAll(pc.program.recipes...)); err != nil { return }

        var pos Position
        var source, str string
        var sources []string
        var positions []Position
        for _, recipe := range recipes {
                if !pos.IsValid() { pos = recipe.Position() }
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

                positions = append(positions, pos)
                sources = append(sources, source)
                source = ""
                pos = Position{}
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
        var exeres = new(ExecResult)
        exeres.position = pc.program.position

        if buffout { exeres.Stdout.Buf = new(bytes.Buffer) }
        if bufferr { exeres.Stderr.Buf = new(bytes.Buffer) }
        if verbout { exeres.Stdout.Tie = stdout }
        if verberr { exeres.Stderr.Tie = stderr }
        if logFileName == "" {
                // no log required
        } else if err = os.MkdirAll(filepath.Dir(logFileName), os.FileMode(0755)); err != nil {
                err = wrap(pc.program.position, err)
                return // FIXME: err for outer func
        } else if logfile, err = os.Create(logFileName); err != nil {
                err = wrap(pc.program.position, err)
                return // FIXME: err for outer func
        } else {
                cmdline := strings.Join(sources, "\n")
                log.createWriter(logfile, dir, cmdline)
                exeres.Stdout.log = &log
                exeres.Stderr.log = &log
        }

        //exeres.Stderr.Line = rxKnownErrors // the line filter
        exeres.Stderr.scanerr = true
        log.filename = logFileName

        var run = func() {
                var targetStr string
                defer func(start time.Time) {
                        if err == nil {
                                err = stamp(pc, target, start, /*!prompt*/true)
                        }
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
                        if pc.caller != nil {
                                pc.caller.calleeDone(err)
                        }
                        if prompt {
                                if pc.caller == nil {
                                        if err == nil {
                                                fmt.Fprintf(stderr, "… ok\n")
                                        } else if _, ok := err.(*scanner.Error); ok {
                                                fmt.Fprintf(stderr, "\n%v\n", err)
                                        } else {
                                                fmt.Fprintf(stderr, "%v\n", err)
                                        }
                                } else {
                                        if err == nil {
                                                //fmt.Fprintf(stderr, "%s%s …… ok\n", promStr, targetStr)
                                        } else if _, ok := err.(*scanner.Error); ok {
                                                fmt.Fprintf(stderr, "%s%s ……\n%v\n", promStr, targetStr, err)
                                        } else {
                                                fmt.Fprintf(stderr, "%s%s …… %v\n", promStr, targetStr, err)
                                        }
                                }
                        }
                } (time.Now())
                if prompt {
                        if a := strings.Split(targetName, PathSep); len(a) > 3 {
                                targetStr = filepath.Join(a[len(a)-3:]...)
                                targetStr = filepath.Join("…", targetStr)
                        } else {
                                targetStr = targetName
                        }
                        if promStr == "" {
                                promStr = "smart: gen "
                        } else {
                                promStr += ": "
                        }
                        if pc.caller == nil {
                                fmt.Fprintf(stderr, "%s%s …\n", promStr, targetStr)
                        } else {
                                fmt.Fprintf(stderr, "%s%s ……\n", promStr, targetStr)
                        }
                }
                for i, src := range sources {
                        var pos = positions[i]
                        if strings.HasPrefix(src, "@") {
                                src = src[1:]
                        } else if !prompt {
                                var s string
                                s = strings.Replace(src, "\n", "\\n", -1)
                                s = strings.Replace(s, "\\\\n", "\\\n", -1)
                                fmt.Fprintf(stderr, "%s\n", s)
                        }
                        if src = strings.TrimSpace(src); src == "" {
                                continue
                        } else if dir != "" && !nocd /*&& pc.program.changedWD == ""*/ {
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
                        if stdin {
                                sh.Stdin = os.Stdin
                                sh.Args = append(sh.Args, "-ti")
                        }
                        sh.Args = append(sh.Args, p.opt, src)

                        exeres.Stderr.report = !silent
                        exeres.Status, err = exeres.Stderr.runAndProcessKnownErrors(pos, pc, dock, sh, p, 1)
                        if err != nil { err = wrap(pc.program.position, err)
                                if !silent { fmt.Fprintf(stderr, "%v\n", err) }
                                return
                        }
                }
        }

        printEnteringDirectory()

        if pc.caller != nil {
                pc.caller.calleeStart()
                go run()
        } else {
                run()
        }

        // The execution is performed asynchronously, the result can't
        // be obtained at this point.
        result = exeres
        return
}

func stamp(pc *traversal, target Value, start time.Time, verb bool) (err error) {
        var t Value
        if t, err = target.expand(expandAll); err != nil {
                return
        }

        var files []*File
        if files, err = t.stamp(pc); err == nil && verb {
                for _, file := range files {
                        d := file.info.ModTime().Sub(start);
                        fmt.Printf("smart: Updated %v (%v)\n", file, d)
                }
        }
        return
}
