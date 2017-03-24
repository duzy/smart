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
        source string
}

func (s *dialectShell) dialect() string { return "shell" }
func (s *dialectShell) evaluate(recipes... types.Value) (result types.Value, err error) {
        s.source += strings.TrimRightFunc(recipes[0].String(), unicode.IsSpace)
        if strings.HasSuffix(s.source, "\\") {
                return
        }

        var (
                stdout bytes.Buffer
                stderr bytes.Buffer
                status types.Value
        )
        sh := exec.Command(s.interpreter, s.xopt, s.source)
        sh.Stdout, sh.Stderr = &stdout, &stderr
        err = sh.Run(); s.source = ""
        if err == nil {
                /*
                result = values.String(stdout.String())
                es := strings.TrimRightFunc(stderr.String(), unicode.IsSpace)
                if es != "" {
                        fmt.Printf("%v", es)
                } */
                status = values.None
        } else {
                /*
                fmt.Fprintf(os.Stderr, "%v", s.source)
                fmt.Fprintf(os.Stderr, "%s", stderr.String()) */
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
        result = values.Group(targetShellKind, status,
                values.String(stdout.String()),
                values.String(stderr.String()))
        return
}
