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
                source += strings.TrimRightFunc(recipe.String(), unicode.IsSpace)
                if strings.HasSuffix(source, "\\") {
                        continue
                }

                if /* TODO: using `--verbose-shell` to control this */true {
                        fmt.Printf("%v\n", strings.Replace(source, "\n", "\\n", -1))
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
