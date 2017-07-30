//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        //"github.com/duzy/smart/token"
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        "os/exec"
        "strings"
        "errors"
        "bytes"
        "fmt"
        "os"
)

type dialectDocksh struct {
        monoInterpreter
}

func (s *dialectDocksh) dialect() string { return "docksh" }
func (s *dialectDocksh) evaluate(prog *Program, args []types.Value, recipes []types.Value) (result types.Value, err error) {
        var (
                stdoutOpt, _ = prog.scope.Lookup("shell-stdout").(*types.Def)
                stderrOpt, _ = prog.scope.Lookup("shell-stderr").(*types.Def)
                stdinOpt,  _ = prog.scope.Lookup("shell-stdin").(*types.Def)
                symDxi = prog.scope.Find("docker-exec-image")
                symWd = prog.scope.Find("/") // "."
                stdout bytes.Buffer
                stderr bytes.Buffer
                status types.Value
                source string
        )

        for _, recipe := range recipes {
                source += recipe.String()
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
                        dxi = "default-image"
                        src = source
                )
                if symDxi != nil {
                        v, _ := symDxi.(types.Caller).Call()
                        dxi = v.String()
                }
                if symWd != nil {
                        v, _ := symWd.(types.Caller).Call()
                        if s := v.String(); s != "" {
                                src = fmt.Sprintf("cd '%s' && %s", s, source)
                        }
                }

                var (
                        args []string
                        stdin = stdinOpt != nil && stdinOpt.Value().String() == "on"
                )
                if stdin {
                        args = []string{ "exec", "-ti", dxi, "sh", "-c", src }
                } else {
                        args = []string{ "exec", dxi, "sh", "-c", src }
                }

                var sh *exec.Cmd
                sh = exec.Command("docker", args...)
                sh.Stdout, sh.Stderr = &stdout, &stderr
                if stdoutOpt != nil && stdoutOpt.Value().String() == "on" {
                        sh.Stdout = os.Stdout
                }
                if stderrOpt != nil && stderrOpt.Value().String() == "on" {
                        sh.Stderr = os.Stderr
                }
                if stdin {
                        sh.Stdin = os.Stdin
                }
                err = sh.Run()
                if err == nil {
                        status, source = values.Int(0), ""
                } else {
                        var (
                                s = err.Error()
                                code int64
                        )
                        if n, e := fmt.Sscanf(s, "exit status %v", &code); n == 1 && e == nil {
                                status = values.Int(code)
                                err = errors.New(fmt.Sprintf("%v (%s)", err, source))
                        } else {
                                status = values.String(s)
                        }
                        source = ""
                        break
                }
        }
        
        if /* TODO: using `--verbose-shell` to control this */false {
                fmt.Printf("%v", stdout.String())
        }
        
        result = values.Group(targetShellKind, status,
                values.String(stdout.String()),
                values.String(stderr.String()))
        return
}
