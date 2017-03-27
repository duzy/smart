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
        "unicode"
        "bytes"
        "fmt"
        //"os"
)

var defaultShellInterpreter = "sh"

func trimLeftSpaces(s string) string {
        return strings.TrimLeftFunc(s, unicode.IsSpace)
}

func trimRightSpaces(s string) string {
        return strings.TrimRightFunc(s, unicode.IsSpace)
}

type dialectShell struct {
        monoInterpreter
        interpreter string // shell interpreter
        xopt string // execute option: -c (sh, python), -e (perl)
}

func (s *dialectShell) dialect() string { return "shell" }
func (s *dialectShell) evaluate(prog *Program, recipes... types.Value) (result types.Value, err error) {
        var (
                stdout bytes.Buffer
                stderr bytes.Buffer
                status types.Value
                source string
        )
        for _, recipe := range recipes {
                source += recipe.String() // trimRightSpaces(recipe.String())
                if strings.HasSuffix(source, "\\") {
                        source += "\n" // give back the line feed
                        continue
                }

                if /* TODO: using `--verbose-shell` to control this */true {
                        var s = source
                        s = strings.Replace(s, "\n", "\\n", -1)
                        s = strings.Replace(s, "\\\\n", "\\\n", -1)
                        fmt.Printf("%v\n", s)
                }
                
                sh := exec.Command(s.interpreter, s.xopt, source)
                sh.Stdout, sh.Stderr = &stdout, &stderr
                err = sh.Run(); source = ""
                if err == nil {
                        status = values.Int(0) //values.None
                } else {
                        var (
                                s = err.Error()
                                code int64
                        )
                        if n, e := fmt.Sscanf(s, "exit status %v", &code); n == 1 && e == nil {
                                status, err = values.Int(code), nil
                        } else {
                                status = values.String(s)
                        }
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
