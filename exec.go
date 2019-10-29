//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/scanner"
        "extbit.io/smart/token"
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
const errCommandFailedFmt = "exit status %d"
const (
        rxNotTTYDevice_i int = iota
        rxNoContainer_i
        rxNoNetwork_i
        rxCompilation_i
        rxIncludedFrom_i
        rxFileNotFound_i
        rxArNoSuchFile_i
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

        rxNotTTYDevice = regexp.MustCompile(errNotTTYDevice)
        rxNoContainer = regexp.MustCompile(errNoContainer)
        rxNoNetwork = regexp.MustCompile(errNoNetwork)
        rxCompilation = regexp.MustCompile(errCompilation)
        rxIncludedFrom = regexp.MustCompile(errIncludedFrom)
        rxFileNotFound = regexp.MustCompile(errFileNotFound)
        rxArNoSuchFile = regexp.MustCompile(errArNoSuchFile)

        knownerrors = []*regexp.Regexp{
                rxNotTTYDevice_i: rxNotTTYDevice,
                rxNoContainer_i: rxNoContainer,
                rxNoNetwork_i: rxNoNetwork,
                rxCompilation_i: rxCompilation,
                rxIncludedFrom_i: rxIncludedFrom,
                rxFileNotFound_i: rxFileNotFound,
                rxArNoSuchFile_i: rxArNoSuchFile,
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
        lines int
}

func (p *ExecLog) Write(b []byte) (n int, err error) {
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

func (p *ExecBuffer) processKnownErrors(prog *Program, dock *Project, sh *exec.Cmd, x *executor, status, num int) (err error) {
        var pos = prog.position
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
                                fmt.Fprintf(stderr, "%s:%s:%s: included here\n", v[1], v[2], v[3])
                        case rxFileNotFound_i:
                                fmt.Fprintf(stderr, "%s:%s:%s: exec: `%s` file not found\n", v[1], v[2], v[3], v[4])
                                if err == nil {
                                        err = scanner.Errorf(token.Position(pos), "`%v` file not found, required by `%s` (exec)", v[4], filepath.Base(string(v[1])))
                                }
                        case rxArNoSuchFile_i:
                                fmt.Fprintf(stderr, "exec: (ar): '%s' not found (as '%s')", filepath.Base(string(v[1])), v[1])
                                if err == nil {
                                        err = scanner.Errorf(token.Position(pos), "`%v` file not found", filepath.Base(string(v[1])))
                                }
                        }
                }
        }
        if err != nil { return }
        if !p.scanerr || p.matches == nil {
                //err = fmt.Errorf(...)
        }

        if err == nil && retry && num < maxRetries {
                fmt.Fprintf(stderr, "smart: good to retry (num = %d)\n", num)
                c := exec.Command(sh.Path, sh.Args...)
                c.Stdout, c.Stderr, c.Stdin, c.Env = sh.Stdout, sh.Stderr, sh.Stdin, sh.Env
                p.runAndProcessKnownErrors(prog, dock, c, x, num+1) // retry
        } else if err != nil {
                // ends with error
        } else if tag == "" {
                err = fmt.Errorf(errCommandFailedFmt, status) //, targetName
        } else if skip := p.skips(tag); !skip && dock != nil && num < maxRetries {
                p.retried[tag] = true // save it to skip next time
                if err = x.runContainer(prog, dock); err == nil {
                        fmt.Fprintf(stderr, "smart: started %s\n", tag)
                        c := exec.Command(sh.Path, sh.Args...)
                        c.Stdout, c.Stderr, c.Stdin, c.Env = sh.Stdout, sh.Stderr, sh.Stdin, sh.Env
                        p.runAndProcessKnownErrors(prog, dock, c, x, num+1) // retry
                } else {
                        //err = scanner.Errorf(token.Position(prog.position), "`%s` no such container", tag)
                }
        }
        return
}

func (p *ExecBuffer) runAndProcessKnownErrors(prog *Program, dock *Project, sh *exec.Cmd, x *executor, num int) (status int, err error) {
        p.matches = nil
        if err = sh.Run(); err == nil {
                // It's good!
        } else if n, e := fmt.Sscanf(err.Error(), "exit status %v", &status); n == 1 && e == nil {
                if p.log != nil && p.log.writer != nil {
                        fmt.Fprintf(p.log, "\n"+errCommandFailedFmt+"\n", status)
                        fmt.Fprintf(stderr, "%s:%d: %v\n", p.log.filename, p.log.lines, err)
                }
                err = p.processKnownErrors(prog, dock, sh, x, status, num)
        } else {
                if status == 0 { status = -1 }
                if e != nil { err = e }
        }
        return
}

func (p *ExecBuffer) parseKnownErrors(pos Position, target string, report bool) (err error, tag string, retry bool) {
        if true/*p.Subm == nil*/ {
                return
        } else if str := string(""/*p.Subm[0][0][0]*/); str == errNotTTYDevice {
                retry = true
        } else if m := rxNoContainer.FindAllStringSubmatch(str, -1); m != nil {
                tag = m[0][1] // tag the container name
        } else if m := rxCompilation.FindAllStringSubmatch(str, -1); m != nil {
                err = scanner.Errorf(token.Position(pos), "%s", m[0][4])
        } else if m := rxFileNotFound.FindAllStringSubmatch(str, -1); m != nil {
                fmt.Printf("=%v\n", str)
                fmt.Printf("=%v\n", m)
                err = scanner.Errorf(token.Position(pos), "`%v` file not found, required by `%s` (exec)", m[0][4], filepath.Base(m[0][1]))
                if report { fmt.Fprintf(stderr, "%s:%s:%s: exec: `%s` file not found\n", m[0][1], m[0][2], m[0][3], m[0][4]) }
        } else if m := rxArNoSuchFile.FindAllStringSubmatch(str, -1); m != nil {
                err = scanner.Errorf(token.Position(pos), "`%v` file not found, required by `%s` (exec)", filepath.Base(m[0][1]), filepath.Base(target))
                if report { fmt.Fprintf(stderr, "exec: (ar): '%s' not found (as '%s')", filepath.Base(m[0][1]), m[0][1]) }
        } else if matched, _ := regexp.MatchString(errNoNetwork, str); matched {
                // TODO: dealing with network not found error
        } else if false {
                // retry the command
                tag, retry = ""/*string(p.Subm[0][0][1])*/, true
        } else {
                err = fmt.Errorf(str)
        }
        return
}

type ExecResult struct {
        Stdout ExecBuffer
        Stderr ExecBuffer
        Status int
}
func (p *ExecResult) refs(_ Value) bool { return false }
func (p *ExecResult) closured() bool { return false }
func (p *ExecResult) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *ExecResult) cmp(v Value) (res cmpres) {
        if v.Type() == ExecResultType {
                a, ok := v.(*ExecResult)
                assert(ok, "value is not ExecResult")
                if p.Status == a.Status { res = cmpEqual }
        }
        return
}
func (p *ExecResult) Type() Type { return ExecResultType }
func (p *ExecResult) True() bool { return p.Status == 0 && p.Stderr.Buf.Len() == 0 /* && p.Stdout.Buf.Len() > 0 */ }
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

func (p *executor) runContainer(prog *Program, dock *Project) (err error) {
        if run, _ := dock.resolveEntry("run"); run != nil {
                _, err = run.Execute(prog.position/*, &String{`sh -c "while sleep 3600; do :; done"`}*/)
                if err != nil {
                        fmt.Fprintf(stderr, "%v: %v\n", prog.position, err)
                }
        } else {
                err = fmt.Errorf("dock⇒run undefined")
        }
        return
}

func (p *executor) ensureContainerRunning(prog *Program, dock *Project, container string) (err error) {
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
                if err = p.runContainer(prog, dock); err == nil {
                        time.Sleep(time.Second)
                }
        }
        return
}

func (p *executor) Evaluate(prog *Program, args []Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        var prompt, verbout, verberr, buffout, bufferr, stdin, silent, nocd bool
        var cmd, promStr, logFileName = p.cmd, "", ""
        var aa []string
        var opts = []string{
                "o,stdout",
                "e,stderr",
                "v,verbout",
                "w,verberr",
                "p,prompt", // --verbose-shell
                "i,stdin",
                "s,silent",
                "d,dump", // verbout, verberr
                "l,log",
                "nocd",
        }
ForArgs:
        for i, v := range args {
                if !p.bare && i == 0 {
                        var s string
                        if s, err = v.Strval(); err != nil { return }
                        if s == "shell" { cmd = defaultShell }
                        continue ForArgs
                }

                var ( runes []rune ; names []string ; s string )
                switch t := v.(type) {
                case *Pair:
                        if flag, _ := t.Key.(*Flag); flag != nil {
                                if runes, names, err = flag.opts(opts...); err != nil { return } else {
                                        v = t.Value
                                }
                        } else {
                                err = fmt.Errorf("`%v` unsupported", t)
                                return
                        }
                case *Flag:
                        if runes, names, err = t.opts(opts...); err != nil { return }
                        v = nil // no flag value
                default:
                        if s, err = v.Strval(); err != nil { return } else {
                                aa = append(aa, s)
                        }
                        continue ForArgs
                }

                for i, ru := range runes {
                        switch ru {
                        case 'i': stdin   = true
                        case 'o': buffout = true
                        case 'e': bufferr = true
                        case 'v': verbout = true
                        case 'w': verberr = true
                        case 's': silent  = true
                        case 'p':
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
                        case 0:
                                switch names[i] {
                                case "nocd": nocd = true
                                }
                        }
                }
        }

        var dock *Project
        if !p.bare {
                if prog.project.name == ".dock" {
                        dock = prog.project
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
                                if _, dockSym := prog.project.scope.Find(".dock"); dockSym != nil {
                                        if pn, _ := dockSym.(*ProjectName); pn != nil {
                                                dock = pn.NamedProject()
                                        }
                                }
                        }
                } else {
                        if _, dockSym := prog.project.scope.Find(".dock"); dockSym != nil {
                                if pn, _ := dockSym.(*ProjectName); pn != nil {
                                        dock = pn.NamedProject()
                                }
                        }
                }

                fmt.Fprintf(stderr, "%v: %v\n", dock, dock.absPath)

                if dock == nil {
                        err = fmt.Errorf("docking unavailable (in %s)", prog.Project().Name())
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
                        if err = p.ensureContainerRunning(prog, dock, container); err != nil {
                                return
                        }
                }
        }

        var cwd string
        if v, e := prog.scope.Lookup("CWD").(*Def).Call(prog.position); e != nil {
                err = e; return
        } else if v != nil {
                if cwd, err = v.Strval(); err != nil { return }
        } else if v, e := prog.scope.Lookup("/").(*Def).Call(prog.position); e != nil {
                err = e; return
        } else if v != nil {
                if cwd, err = v.Strval(); err != nil { return }
        }

        // Fixes work directory conflicts. It happens
        // sometimes even the 'sh.Dir' is set to cwd.
        // Because the current work directory is not
        // thread safe.
        var dir = cwd
        if prog.changedWD != "" {
                if filepath.IsAbs(prog.changedWD) {
                        dir = prog.changedWD
                } else {
                        dir = filepath.Join(prog.project.absPath, prog.changedWD)
                }
        }

        var targetName string
        var target = prog.pc.targetDef.Value
        if targetName, err = target.Strval(); err != nil {
                return
        }

        var source, str string
        var sources []string
        var envars []*Pair // disclosed values
        if def, _ := prog.Scope().Lookup(TheShellEnvarsDef).(*Def); def != nil {
                if l, _ := def.Value.(*List); l != nil {
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
        if recipes, err = mergeresult(ExpandAll(prog.recipes...)); err != nil { return }
        for _, recipe := range recipes {
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

                sources = append(sources, source)
                source = ""
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
        if buffout { exeres.Stdout.Buf = new(bytes.Buffer) }
        if bufferr { exeres.Stderr.Buf = new(bytes.Buffer) }
        if verbout { exeres.Stdout.Tie = stdout }
        if verberr { exeres.Stderr.Tie = stderr }
        if logFileName == "" {
                // no log required
        } else if err = os.MkdirAll(filepath.Dir(logFileName), os.FileMode(0755)); err != nil {
                fmt.Printf("MkdirAll: %v\n", err)
                return // FIXME: err for outer func
        } else if logfile, err = os.Create(logFileName); err != nil {
                fmt.Printf("Create: %v\n", err)
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

        var caller *traversecontext
        var run = func() {
                var targetStr string
                defer func(start time.Time) {
                        if err == nil {
                                err = stamp(target, start, /*!prompt*/true)
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
                        if caller != nil {
                                caller.group.Done()
                                //caller.calleeReses = append(caller.calleeReses, exeres)
                                if err != nil {
                                        caller.calleeErrors = append(caller.calleeErrors, err)
                                }
                        }
                        if prompt {
                                if caller == nil {
                                        if err == nil {
                                                fmt.Fprintf(stderr, "… ok\n")
                                        } else if _, ok := err.(*scanner.Error); ok {
                                                fmt.Fprintf(stderr, "\n%v\n", err)
                                        } else if _, ok := err.(*scanner.Errors); ok {
                                                fmt.Fprintf(stderr, "\n%v\n", err)
                                        } else {
                                                fmt.Fprintf(stderr, "%v\n", err)
                                        }
                                } else {
                                        if err == nil {
                                                //fmt.Fprintf(stderr, "%s%s …… ok\n", promStr, targetStr)
                                        } else if _, ok := err.(*scanner.Error); ok {
                                                fmt.Fprintf(stderr, "%s%s ……\n%v\n", promStr, targetStr, err)
                                        } else if _, ok := err.(*scanner.Errors); ok {
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
                        if caller == nil {
                                fmt.Fprintf(stderr, "%s%s …\n", promStr, targetStr)
                        } else {
                                fmt.Fprintf(stderr, "%s%s ……\n", promStr, targetStr)
                        }
                }
                for _, src := range sources {
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
                        } else if dir != "" && !nocd /*&& prog.changedWD == ""*/ {
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

                        lockCD(dir, 5*time.Millisecond)
                        if s, _ := os.Getwd(); s != dir {
                                assert(s == dir, "wrong work directory (%s != %s)", s, dir)
                                if false {
                                        fmt.Printf("exec: %v %v (%v %v)\n", dir, s, cwd, prog.changedWD)
                                }
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

                        exeres.Status, err = exeres.Stderr.runAndProcessKnownErrors(prog, dock, sh, p, 1)
                        if err == nil {
                                // good
                        } else {
                                // Return immediately once error occured. The
                                // rest commands won't be executed.
                                if silent { err = nil }
                                return
                        }
                }
        }

        printEnteringDirectory()

        if len(prog.callers) > 0 {
                caller = prog.callers[0]
                caller.group.Add(1)
                go run()
        } else {
                run()
        }

        // The execution is performed asynchronously, the result can't
        // be obtained at this point.
        result = exeres
        return
}

func stamp(target Value, start time.Time, verb bool) (err error) {
        var t Value
        if t, err = target.expand(expandAll); err != nil {
                return
        }
        switch t := t.(type) {
        case *Bareword, *Flag:
                // does nothing...
        case *File:
                fullname := t.FullName()
                t.info, err = os.Stat(fullname)
                context.globe.stamp(fullname, t.info.ModTime())
                if verb {
                        d := t.info.ModTime().Sub(start);
                        fmt.Printf("smart: Updated %v (%v)\n", target, d)
                }
        case *Path:
                //if t.File == nil { break }
                //fullname := t.File.FullName()
                //t.File.info, err = os.Stat(fullname)
                //context.globe.stamp(fullname, t.File.info.ModTime())
                var fullname string
                if fullname, err = t.Strval(); err != nil { break }
                var info os.FileInfo
                if info, err = os.Stat(fullname); err != nil { break }
                context.globe.stamp(fullname, info.ModTime())
                if verb {
                        d := info.ModTime().Sub(start);
                        fmt.Printf("smart: Updated %v (%v)\n", target, d)
                }
        case *List:
                for _, elem := range t.Elems {
                        if err = stamp(elem, start, verb); err != nil {
                                return
                        }
                }
        default:
                if verb {
                        fmt.Printf("smart: Updated %v (stamp %T)\n", target, target)
                }
        }
        return
}
