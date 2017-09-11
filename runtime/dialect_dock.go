//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        //"github.com/duzy/smart/token"
        "github.com/duzy/smart/types"
        //"github.com/duzy/smart/values"
        "os/exec"
        "strings"
        "errors"
        //"bytes"
        "fmt"
        "os"
)

type dialectDock struct {
        monoInterpreter
}
func (s *dialectDock) dialect() string { return "dock" }
func (s *dialectDock) evaluate(prog *Program, context *types.Scope, args []types.Value, recipes []types.Value) (result types.Value, err error) {
        var (
                envarsOpt, _ = prog.scope.Lookup("shell-envars").(*types.Def)
                stdoutOpt, _ = prog.scope.Lookup("shell-stdout").(*types.Def)
                stderrOpt, _ = prog.scope.Lookup("shell-stderr").(*types.Def)
                stdinOpt,  _ = prog.scope.Lookup("shell-stdin").(*types.Def)
                wd = prog.Getwd(context)
                exeres = new(types.ExecResult)
                //stdout bytes.Buffer
                //stderr bytes.Buffer
                //status types.Value
                source string
                shi = "sh"
        )

        if len(args) > 0 {
                shi = args[0].Strval()
        }

        for _, recipe := range recipes {
                source += recipe.Strval()
                if strings.HasSuffix(source, "\\") {
                        source += "\n" // give back the line feed
                        continue
                }

                // Escape '$$' sequences.
                source = strings.Replace(source, "$$", "$", -1)

                // Remove tabs in line breakings.
                source = strings.Replace(source, "\\\n\t", "\\\n", -1)

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
                        dxi = "default-dock-image"
                        dxiName = "docker-exec-image"
                        _, obj = context.Find(dxiName)
                        envars []types.Value // disclosed values
                )
                if obj == nil { _, obj = prog.scope.Find(dxiName) }
                if obj != nil {
                        if v, e := obj.(types.Caller).Call(); e != nil {
                                err = e; return
                        } else if v == nil {
                                // nothing changed
                        } else if s := strings.TrimSpace(v.Strval()); s != "" {
                                dxi = s
                        }
                }
                if envarsOpt != nil {
                        if l, _ := envarsOpt.Value.(*types.List); l != nil {
                                for _, v := range l.Elems {
                                        if v, err = types.Disclose(context, v); err != nil {
                                                return
                                        } else {
                                                envars = append(envars, v)
                                        }
                                }
                        }
                }
                if s := wd; s != "" {
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

                var (
                        args []string
                        stdin = stdinOpt != nil && stdinOpt.Value.Strval() == "on"
                )
                if stdin {
                        args = []string{ "exec", "-ti", dxi, shi, "-c", src }
                } else {
                        args = []string{ "exec", dxi, shi, "-c", src }
                }

                var sh *exec.Cmd
                sh = exec.Command("docker", args...)
                sh.Stdout, sh.Stderr = &exeres.Stdout, &exeres.Stderr
                for _, env := range envars {
                        sh.Env = append(sh.Env, env.Strval())
                }
                if stdoutOpt != nil && stdoutOpt.Value.Strval() == "on" {
                        sh.Stdout = os.Stdout
                }
                if stderrOpt != nil && stderrOpt.Value.Strval() == "on" {
                        sh.Stderr = os.Stderr
                }
                if stdin {
                        sh.Stdin = os.Stdin
                }
                err = sh.Run()
                if err == nil {
                        exeres.Status, source = 0, ""
                } else {
                        var s = err.Error()
                        if n, e := fmt.Sscanf(s, "exit status %v", &exeres.Status); n == 1 && e == nil {
                                err = errors.New(fmt.Sprintf("%v (%s)", err, source))
                        } else {
                                exeres.Status = -1 //values.String(s)
                        }
                        source = ""
                        break
                }
        }
        
        if /* TODO: using `--verbose-shell` to control this */false {
                fmt.Printf("%v", exeres.Stdout.String())
        }
        
        /*result = values.Group(targetShellKind, status,
                values.String(stdout.String()),
                values.String(stderr.String()))*/
        result = exeres
        return
}
