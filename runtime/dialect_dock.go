//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        "os/exec"
        "strings"
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
        rxNoSuchContainer = regexp.MustCompile(`Error response from daemon: No such container: (.*)`)
        ensureSkips = make(map[string]bool)
)

type dialectDock struct {
}

func (s *dialectDock) runContainer(prog *types.Program, container string) (err error) {
        var _, dockSym = prog.Scope().Find("dock")
        if pn, ok := dockSym.(*types.ProjectName); ok && pn != nil {
                var (
                        scope = pn.Project().Scope()
                        _, symContainer = scope.Find("container")
                        _, symImage = scope.Find("image")
                        _, symStart = scope.Find("start")
                        _, symRun = scope.Find("run")
                )
                if defContainer, _ := symContainer.(*types.Def); defContainer != nil {
                        if s := defContainer.Value.Strval(); s != container {
                                return fmt.Errorf("dock %s, but %s", container, s)
                        }
                }
                if start, _ := symStart.(*types.RuleEntry); start != nil {
                        _, err = start.Execute(prog.Position())
                } else if run, _ := symRun.(*types.RuleEntry); run != nil {
                        _, err = run.Execute(prog.Position(), values.String("sh -i"))
                } else if defImage, _ := symImage.(*types.Def); defImage != nil {
                        err = fmt.Errorf("todo: run container: %T %v %v\n", symImage, symImage, container)
                }
        } else {
                err = fmt.Errorf("no docking for %v\n", container)
        }
        return
}

func (s *dialectDock) ensureContainerRunning(prog *types.Program, container string) (err error) {
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
                        if err = s.runContainer(prog, container); err == nil {
                                time.Sleep(time.Second)
                        }
                }
        }
        return
}

func (s *dialectDock) Evaluate(prog *types.Program, args []types.Value, recipes []types.Value) (result types.Value, err error) {
        if args, err = types.JoinEval(prog.Scope(), args...); err != nil {
                return
        }

        var (
                exeres = new(types.ExecResult)
                source string
                shi = "sh" // interpreter
        )

        ForRecipes: for _, recipe := range recipes {
                if source += recipe.Strval(); strings.HasSuffix(source, "\\") {
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
                        dxi = DockDefaultContainer
                        _, obj = prog.Scope().Find(DockExecVarName)
                        envars []types.Value // disclosed values
                )
                if obj == nil { _, obj = prog.Scope().Find(DockExecVarName) }
                if obj != nil {
                        if v, e := obj.(types.Caller).Call(prog.Position()); e != nil {
                                err = e; return
                        } else if v == nil {
                                // nothing changed
                        } else if v, err = types.Disclose(prog.Scope(), v); err != nil {
                                return
                        } else if s := strings.TrimSpace(v.Strval()); s != "" {
                                dxi = s
                        }
                }
                
                if envarsDef, _ := prog.Scope().Lookup(types.TheShellEnvarsDef).(*types.Def); envarsDef != nil {
                        if l, _ := envarsDef.Value.(*types.List); l != nil {
                                for _, v := range l.Elems {
                                        if v, err = types.Disclose(prog.Scope(), v); err != nil {
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
                        a = []string{ "exec" }
                        cmd string
                )
                ForArgs: for _, v := range args {
                        switch t := v.(type) {
                        case *types.Pair:
                                if f, _ := t.Key.(*types.Flag); f != nil {
                                        switch f.Name.Strval() {
                                        case "dump": // -dump=xxx
                                                switch t.Value.Strval() {
                                                case "stdout": verbout = true
                                                case "stderr": verberr = true
                                                }
                                                continue ForArgs
                                        }
                                }
                        case *types.Flag:
                                switch t.Name.Strval() {
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
                                shi = args[0].Strval()
                                continue ForArgs
                        }
                        a = append(a, v.Strval())
                }

                wd := prog.Scope().Lookup(types.TheCurrWorkDirDef).(*types.Def)
                if s := wd.Value.Strval(); s != "" || nocd {
                        if false {
                                fmt.Printf("dialectDock.evaluate: %s\n", s)
                        }
                        if t := strings.TrimSpace(source); t == "" {
                                src = fmt.Sprintf("cd '%s'", s)
                        } else if strings.HasPrefix(t, "#") {
                                src = fmt.Sprintf("cd '%s' %s", s, t)
                        } else {
                                // Insert a "\n" before the right paren ')' to ensure that
                                // it's working with something like "true #comment...".
                                src = fmt.Sprintf("cd '%s' && (%s\n)", s, t)
                        }
                        if s = ""; len(envars) > 0 {
                                for i, env := range envars {
                                        if i > 0 { s += " && " }
                                        p := env.(*types.Pair)
                                        s += "export "
                                        s += p.Key.Strval() + "=\""
                                        s += p.Value.Strval() + "\""
                                }
                                src = fmt.Sprintf("%s && %s", s, src)
                        }
                }

                if shi == "shell" {
                        shi = defaultShellInterpreter
                }

                if dxi == "" {
                        return nil, fmt.Errorf("unknown script interpreter")
                } else if dxi == "-" {
                        cmd, a = shi, []string{ "-c", src }
                } else {
                        cmd, a = "docker", append(a, dxi, shi, "-c", src)
                }

                if false {
                        if err = s.ensureContainerRunning(prog, dxi); err != nil {
                                return
                        }
                }

                var sh = exec.Command(cmd, a...)
                sh.Stdout, sh.Stderr, sh.Env = &exeres.Stdout, &exeres.Stderr, os.Environ()
                for _, v := range envars {
                        if v, err = types.Disclose(prog.Scope(), v); err != nil {
                                return
                        } else {
                                sh.Env = append(sh.Env, v.Strval())
                        }
                }
                if verbout { exeres.Stdout.Tie = os.Stdout }
                if verberr { exeres.Stderr.Tie = os.Stderr }
                if saveout { exeres.Stdout.Buf = new(bytes.Buffer) }
                if saveerr { exeres.Stderr.Buf = new(bytes.Buffer) }
                if stdin   { sh.Stdin = os.Stdin }
                exeres.Stderr.Line = rxNoSuchContainer
                RunCommand: exeres.Stderr.Subm = nil
                if err = sh.Run(); err == nil {
                        exeres.Status, source = 0, ""
                } else {
                        var str = err.Error()
                        if n, e := fmt.Sscanf(str, "exit status %v", &exeres.Status); n == 1 && e == nil {
                                if exeres.Stderr.Subm != nil {
                                        var (
                                                name = string(exeres.Stderr.Subm[0][0][1])
                                                skip, _ = ensureSkips[name]
                                        )
                                        if !skip {
                                                ensureSkips[name] = true
                                                if err = s.runContainer(prog, name); err == nil {
                                                        fmt.Printf("smart: %s started\n", name)
                                                        c := exec.Command(cmd, a...)
                                                        c.Stdout, c.Stderr, c.Env = sh.Stdout, sh.Stderr, sh.Env
                                                        sh = c
                                                        goto RunCommand
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
        types.RegisterDialect("dock", new(dialectDock))
}
