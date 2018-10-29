//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "os/exec"
        "strings"
        "strconv"
        "regexp"
        "bytes"
        "bufio"
        "time"
        "fmt"
        "io"
        "os"
)

const (
        DockImageVarName = "dock->image" // docker image
        DockExecVarName = "dock->exec" // docker exec image
        DockDefaultContainer = "smart-dock-container"
)

var (
        errNotTTYDevice = `the input device is not a TTY`
        errNoContainer = `Error.*: No such container: (.*)`
        errNoNetwork = `Error.*: network (.*) not found\.`
        rxKnownErrors = regexp.MustCompile(strings.Join([]string{
                errNotTTYDevice,
                errNoContainer,
                errNoNetwork,
        }, "|"))
        ensureSkips = make(map[string]bool)
)

type dialectDock struct {}

func docksFindObj(docks []*Project, name string) (obj Object) {
        for _, dock := range docks {
                if obj, _ = dock.resolveObject(name); obj != nil {
                        break
                }
        }
        return
}

func docksFindEnt(docks []*Project, name string) (entry *RuleEntry) {
        for _, dock := range docks {
                if entry, _ = dock.resolveEntry(name); entry != nil {
                        break
                }
        }
        return
}

func (s *dialectDock) runContainer(prog *Program, docks []*Project) (err error) {
        if run := docksFindEnt(docks, "run"); run != nil {
                _, err = run.Execute(prog.Position()/*, &String{`sh -c "while sleep 3600; do :; done"`}*/)
        } else {
                err = fmt.Errorf("dock=>run undefined")
        }
        return
}

func (s *dialectDock) ensureContainerRunning(prog *Program, docks []*Project, container string) (err error) {
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
                        fmt.Printf("%s", s)
                }
        }(stderrR)

        if err = cmd.Run(); err == nil {
                if foundID == "" {
                        if err = s.runContainer(prog, docks); err == nil {
                                time.Sleep(time.Second)
                        }
                }
        }
        return
}

func (s *dialectDock) Evaluate(prog *Program, args []Value, recipes []Value) (result Value, err error) {
        var docks []*Project

        if prog.Project().Name() == "dock" {
                docks = append(docks, prog.Project())
        } else {
                for _, scope := range Closure {
                        if _, sym := scope.Find("dock"); sym != nil {
                                if p, ok := sym.(*ProjectName); ok && p != nil {
                                        docks = append(docks, p.NamedProject())
                                }
                        }
                }
                if docks == nil {
                        if _, dockSym := prog.Project().Scope().Find("dock"); dockSym != nil {
                                if pn, _ := dockSym.(*ProjectName); pn != nil {
                                        docks = append(docks, pn.NamedProject())
                                }
                        }
                }
        }

        if docks == nil {
                err = fmt.Errorf("docking unavailable (in %s)\n", prog.Project().Name())
                return
        }

        defer setclosure(scoping(docks...))

        var strval = func(name string) (str string, err error) {
                if obj := docksFindObj(docks, name); obj != nil {
                        if def, _ := obj.(*Def); def != nil {
                                var v Value
                                if v, err = def.DiscloseValue(); err == nil && v != nil {
                                        if str, err = v.Strval(); str == "-" {
                                                //fmt.Printf("dock: %v %v\n", docks, cc)
                                                /*if v, err = def.DiscloseValue(docks); err == nil && v != nil {
                                                        if str, err = v.Strval(); str == "" { str = "-" }
                                                        fmt.Printf("%v: %v (%v)\n", name, str, def)
                                                }*/
                                        }
                                }
                        }
                }
                return
        }

        var container, image string
        if container, err = strval("dock-container"); err != nil { return }
        if container == "" { err = fmt.Errorf("dock-container undefined"); return }
        if image, err = strval("dock-image"); err != nil { return }
        if image == "" { err = fmt.Errorf("dock-image undefined"); return }
        if args, err = ExpendAll(Join(args...)...); err != nil { return }

        if false {
                if err = s.ensureContainerRunning(prog, docks, container); err != nil {
                        return
                }
        }

        var (
                exeres = new(ExecResult)
                source, str string
                shi = "sh" // interpreter
        )
        ForRecipes: for _, recipe := range recipes {
                if str, err = recipe.Strval(); err != nil { return }
                if source += str; strings.HasSuffix(source, "\\") {
                        source += "\n" // give back the line feed
                        continue ForRecipes
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
                        var s = source
                        s = strings.Replace(s, "\n", "\\n", -1)
                        s = strings.Replace(s, "\\\\n", "\\\n", -1)
                        fmt.Printf("%v\n", s)
                }

                var (
                        src = source
                        envars []Value // disclosed values
                )
                if envarsDef, _ := prog.Scope().Lookup(TheShellEnvarsDef).(*Def); envarsDef != nil {
                        if l, _ := envarsDef.Value.(*List); l != nil {
                                for _, v := range l.Elems {
                                        if v, err = Disclose(v); err != nil {
                                                return
                                        } else {
                                                envars = append(envars, v)
                                        }
                                }
                        }
                }

                var (
                        verbout, verberr, saveout, saveerr, stdin, silent bool
                        nocd bool
                        cmd, str string
                        a = []string{ "exec" }
                )
                ForArgs: for _, v := range args {
                        switch t := v.(type) {
                        case *Pair:
                                if f, _ := t.Key.(*Flag); f != nil {
                                        var name, value string
                                        if name, err = f.Name.Strval(); err != nil { return }
                                        switch name {
                                        case "dump": // -dump=xxx
                                                if value, err = t.Value.Strval(); err != nil { return }
                                                switch value {
                                                case "stdout": verbout = true
                                                case "stderr": verberr = true
                                                }
                                                continue ForArgs
                                        }
                                }
                        case *Flag:
                                var name string
                                if name, err = t.Name.Strval(); err != nil { return }
                                switch name {
                                case "s" : silent = true;  continue ForArgs
                                case "so": saveout = true; continue ForArgs
                                case "se": saveerr = true; continue ForArgs
                                case "soe", "seo":
                                        saveout, saveerr = true, true
                                        continue ForArgs
                                case "vo": verbout = true; continue ForArgs
                                case "ve": verberr = true; continue ForArgs
                                case "veo", "voe", "eo", "oe":
                                        verbout, verberr = true, true
                                        continue ForArgs
                                case "i":
                                        stdin, a = true, append(a, "-ti")
                                        continue ForArgs
                                case "nocd": nocd = true; continue ForArgs
                                }
                        default:
                                if shi, err = args[0].Strval(); err != nil { return }
                                continue ForArgs
                        }
                        if str, err = v.Strval(); err == nil {
                                a = append(a, str)
                        } else {
                                return
                        }
                }

                wd := prog.Scope().Lookup(TheCurrWorkDirDef).(*Def)
                if str, err = wd.Value.Strval(); err != nil { return }
                if str != "" || nocd {
                        if false {
                                fmt.Printf("dialectDock.evaluate: %s\n", str)
                        }
                        if t := strings.TrimSpace(source); t == "" {
                                src = fmt.Sprintf("cd '%s'", str)
                        } else if strings.HasPrefix(t, "#") {
                                src = fmt.Sprintf("cd '%s' %s", str, t)
                        } else {
                                // Insert a "\n" before the right paren ')' to ensure that
                                // it's working with something like "true #comment...".
                                src = fmt.Sprintf("cd '%s' && (%s\n)", str, t)
                        }
                        if str = ""; len(envars) > 0 {
                                for i, env := range envars {
                                        var k, v string
                                        var p = env.(*Pair)
                                        if k, err = p.Key.Strval(); err != nil { return }
                                        if v, err = p.Value.Strval(); err != nil { return }
                                        if i > 0 { str += " && " }
                                        str += fmt.Sprintf(`%s="%s"`, k, strconv.Quote(v))
                                }
                                src = fmt.Sprintf("%s && %s", str, src)
                        }
                }

                if shi == "shell" {
                        shi = "bash" //defaultShellInterpreter
                }

                if container == "-" && image == "-" {
                        cmd, a = shi, []string{ "-c", src }
                } else {
                        cmd, a = "docker", append(a, container, shi, "-c", src)
                }
                
                var (
                        sh = exec.Command(cmd, a...)
                        num = 0
                )
                sh.Stdout, sh.Stderr, sh.Env = &exeres.Stdout, &exeres.Stderr, os.Environ()
                for _, v := range envars {
                        if v, err = Disclose(v); err != nil {
                                return
                        } else if str, err = v.Strval(); err == nil {
                                sh.Env = append(sh.Env, str)
                        } else {
                                return
                        }
                }
                if verbout { exeres.Stdout.Tie = os.Stdout }
                if verberr { exeres.Stderr.Tie = os.Stderr }
                if saveout { exeres.Stdout.Buf = new(bytes.Buffer) }
                if saveerr { exeres.Stderr.Buf = new(bytes.Buffer) }
                if stdin   { sh.Stdin = os.Stdin }
                exeres.Stderr.Line = rxKnownErrors
                RunCommand: exeres.Stderr.Subm = nil
                if err, num = sh.Run(), num+1; err == nil {
                        exeres.Status, source = 0, ""
                } else {
                        var str = err.Error()
                        if n, e := fmt.Sscanf(str, "exit status %v", &exeres.Status); n == 1 && e == nil {
                                if exeres.Stderr.Subm != nil {
                                        if errstr := string(exeres.Stderr.Subm[0][0][0]); errstr == errNotTTYDevice {
                                                if num > 2 { break } // only retry once
                                                fmt.Printf("smart: good to retry (%s)\n", source)
                                                c := exec.Command(cmd, a...)
                                                c.Stdout, c.Stderr, c.Env = sh.Stdout, sh.Stderr, sh.Env
                                                sh = c; goto RunCommand // retry the command
                                        } else if errstr == errNoNetwork {
                                                // TODO: dealing with network not found error
                                        }

                                        var (
                                                name = string(exeres.Stderr.Subm[0][0][1])
                                                skip, _ = ensureSkips[name]
                                        )
                                        if !skip {
                                                ensureSkips[name] = true
                                                if err = s.runContainer(prog, docks); err == nil {
                                                        fmt.Printf("smart: started %s (name=%s)\n", container, name) // name
                                                        c := exec.Command(cmd, a...)
                                                        c.Stdout, c.Stderr, c.Env = sh.Stdout, sh.Stderr, sh.Env
                                                        sh = c; goto RunCommand
                                                }
                                        }
                                }
                                if silent {
                                        err = nil
                                } else {
                                        err = fmt.Errorf("%v", err) // , source
                                }
                        } else {
                                exeres.Status = -1 //values.String(s)
                        }
                        source = ""
                        break
                }
        }
        
        result = exeres
        return
}

func init() {
        RegisterDialect("dock", new(dialectDock))
}
