//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "os/exec"
        "strings"
        "unicode"
        "regexp"
        "bytes"
        "fmt"
        "os"
        "io"
)

type ExecBuffer struct {
        Tie io.Writer
        Buf *bytes.Buffer
        Line *regexp.Regexp
        Subm [][][][]byte
        line []byte
        filters []string
}

func (p *ExecBuffer) filter(s string) {
        p.filters = append(p.filters, s)
}

func (p *ExecBuffer) Write(b []byte) (n int, err error) {
        if p.Line != nil {
                i := bytes.Index(b, []byte("\n"))
                if i == -1 {
                        p.line = append(p.line, b...)
                } else {
                        p.line = append(p.line, b[:i]...)
                }
                if m := p.Line.FindAllSubmatch(p.line, -1); m != nil {
                        p.Subm = append(p.Subm, m)
                }
                if i != -1 {
                        p.line = b[i+1:]
                }
        }
        for _, s := range p.filters {
                if string(b) == s {
                        return len(b), nil
                }
        }
        if p.Tie != nil {
                if n, err = p.Tie.Write(b); err != nil {
                        return
                }
        }
        if p.Buf != nil {
                if n, err = p.Buf.Write(b); err != nil {
                        return
                }
        }
        if err == nil && n == 0 {
                // Returns the number of bytes to avoid "short write" errors.
                // The real bytes written is discarded.
                n = len(b)
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

func trimLeftSpaces(s string) string {
        return strings.TrimLeftFunc(s, unicode.IsSpace)
}

func trimRightSpaces(s string) string {
        return strings.TrimRightFunc(s, unicode.IsSpace)
}

type executor struct {
        cmd string // shell command
        opt string // execute option: -c (sh, python), -e (perl)
}

func (s *executor) Evaluate(prog *Program, args []Value) (result Value, err error) {
        var recipes []Value
        if recipes, err = DiscloseAll(prog.recipes...); err != nil {
                return
        }

        //if args, err = ExpandAll(Merge(args...)...); err != nil {
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        }

        var (
                // TODO: parsing envars and status flags from `args'
                envarsDef, _ = prog.Scope().Lookup(TheShellEnvarsDef).(*Def)
                exeres = new(ExecResult)
                envars []Value // disclosed values
                source, str string
        )
        if envarsDef != nil {
                if l, _ := envarsDef.Value.(*List); l != nil {
                        for _, v := range l.Elems {
                                //if v, err = Disclose(prog.Scope(), v); err != nil {
                                if v, err = v.expand(expandClosure); err != nil {
                                        return
                                } else {
                                        envars = append(envars, v)
                                }
                        }
                }
        }
        for _, recipe := range recipes {
                if str, err = recipe.Strval(); err != nil { return }
                source += str // trimRightSpaces(str)
                if strings.HasSuffix(source, "\\") {
                        source += "\n" // give back the line feed
                        continue
                }

                // Escape '$$' sequences.
                source = strings.Replace(source, "$$", "$", -1)

                // Remove tabs in line breakings.
                source = strings.Replace(source, "\\\n\t", "\\\n", -1)

                // Duplicates all %
                //source = strings.Replace(source, "%", "%%", -1)

                if strings.HasPrefix(source, "@") {
                        source = source[1:]
                } else {
                        // TODO: using `--verbose-shell` to control this
                        var src = source
                        src = strings.Replace(src, "\n", "\\n", -1)
                        src = strings.Replace(src, "\\\\n", "\\\n", -1)
                        fmt.Printf("%v\n", src)
                }

                /*if s := ""; len(envars) > 0 {
                        for i, env := range envars {
                                if i > 0 { s += " && " }
                                p := env.(*Pair)
                                s += "export "
                                s += p.Key.Strval() + "=\""
                                s += p.Value.Strval() + "\""
                        }
                        source = fmt.Sprintf("%s && %s", s, source)
                }*/

                var sh *exec.Cmd
                var verbout, verberr, saveout, saveerr, stdin, silent bool
                if len(args) == 0 {
                        sh = exec.Command(s.cmd, s.opt, source)
                } else {
                        var key, value string
                        var a []string
                        ForArgs: for _, v := range args {
                                switch t := v.(type) {
                                case *Pair:
                                        if f, _ := t.Key.(*Flag); f != nil {
                                                if key, err = f.Name.Strval(); err != nil { return }
                                                if value, err = t.Value.Strval(); err != nil { return }
                                                switch key {
                                                case "dump": // -dump=xxx
                                                        switch value {
                                                        case "stdout": verbout = true
                                                        case "stderr": verberr = true
                                                        }
                                                        continue ForArgs
                                                }
                                        }
                                case *Flag:
                                        if str, err = t.Name.Strval(); err != nil { return }
                                        if verbout = strings.ContainsRune(str, 'v'); verbout { exeres.Stdout.Tie = os.Stdout }
                                        if verberr = strings.ContainsRune(str, 'w'); verberr { exeres.Stderr.Tie = os.Stderr }
                                        if silent  = strings.ContainsRune(str, 's'); silent  { }
                                        if saveout = strings.ContainsRune(str, 'o'); saveout { exeres.Stdout.Buf = new(bytes.Buffer) }
                                        if saveerr = strings.ContainsRune(str, 'e'); saveerr { exeres.Stderr.Buf = new(bytes.Buffer) }
                                        if stdin   = strings.ContainsRune(str, 'i'); stdin   { a = append(a, "-ti") }
                                        continue ForArgs
                                }
                                if str, err = v.Strval(); err != nil { return } else {
                                        a = append(a, str)
                                }
                        }
                        a = append(a, s.opt, source)
                        sh = exec.Command(s.cmd, a...)
                }
                sh.Stdout, sh.Stderr, sh.Env = &exeres.Stdout, &exeres.Stderr, os.Environ()
                if stdin { sh.Stdin = os.Stdin }
                for _, v := range envars {
                        if v, err = v.expand(expandClosure); err != nil {
                                return
                        } else if str, err = v.Strval(); err == nil {
                                sh.Env = append(sh.Env, str)
                        } else {
                                return
                        }
                }

                exeres.Stderr.filter("bash: no job control in this shell\n")
                //exeres.Stderr.Line = rxKnownErrors
                //exeres.Stderr.Subm = nil

                if err = sh.Run(); err == nil {
                        exeres.Status, source = 0, ""
                } else {
                        var s = err.Error()
                        if n, e := fmt.Sscanf(s, "exit status %v", &exeres.Status); n == 1 && e == nil {
                                if silent {
                                        err = nil
                                } else {
                                        err = fmt.Errorf("%v", err) // , source
                                }
                        } else {
                                exeres.Status = -1
                        }
                        source = ""
                        break
                }
        }

        result = exeres
        return
}

func init() {
        RegisterDialect("shell", &executor{ "bash", "-c" })
        RegisterDialect("python", &executor{ "python", "-c" })
        RegisterDialect("perl", &executor{ "perl", "-e" })
}
