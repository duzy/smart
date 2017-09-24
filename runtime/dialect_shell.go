//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        //"github.com/duzy/smart/values"
        "os/exec"
        "strings"
        "errors"
        "unicode"
        //"bytes"
        "fmt"
        "os"
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

func isTrueValue(s string) (res bool) {
        switch strings.ToLower(s) {
        case "on", "yes", "y", "1":
                res = true
        }
        return
}

func (s *dialectShell) dialect() string { return "shell" }
func (s *dialectShell) evaluate(prog *Program, context *types.Scope, args []types.Value, recipes []types.Value) (result types.Value, err error) {
        if args, err = types.JoinEval(context, args...); err != nil {
                return
        }

        var (
                envarsOpt, _ = prog.scope.Lookup("shell-envars").(*types.Def)
                statusOpt, _ = prog.scope.Lookup("shell-status").(*types.Def)
                //stdoutOpt, _ = prog.scope.Lookup("shell-stdout").(*types.Def)
                //stderrOpt, _ = prog.scope.Lookup("shell-stderr").(*types.Def)
                //stdinOpt, _ = prog.scope.Lookup("shell-stdin").(*types.Def)
                exeres = new(types.ExecResult)
                source string
        )
        for _, recipe := range recipes {
                source += recipe.Strval() // trimRightSpaces(recipe.Strval())
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
                        var src = source
                        src = strings.Replace(src, "\n", "\\n", -1)
                        src = strings.Replace(src, "\\\\n", "\\\n", -1)
                        fmt.Printf("%v\n", src)
                }

                var (
                        verbout, verberr, stdin bool
                        sh *exec.Cmd
                )
                if len(args) == 0 {
                        sh = exec.Command(s.interpreter, s.xopt, source)
                } else {
                        var a []string
                        LoopArgs: for _, v := range args {
                                switch t := v.(type) {
                                case *types.Pair:
                                        if f, _ := t.Key.(*types.Flag); f != nil {
                                                switch f.Name.Strval() {
                                                case "dump": // -dump=xxx
                                                        switch t.Value.Strval() {
                                                        case "stdout": verbout = true
                                                        case "stderr": verberr = true
                                                        }
                                                        continue LoopArgs
                                                }
                                        }
                                case *types.Flag:
                                        switch t.Name.Strval() {
                                        case "i": stdin = true; continue LoopArgs
                                        case "do": verbout = true; continue LoopArgs
                                        case "de": verberr = true; continue LoopArgs
                                        case "eo", "oe", "deo", "doe":
                                                verbout, verberr = true, true
                                                continue LoopArgs
                                        }
                                }
                                a = append(a, v.Strval())
                        }
                        a = append(a, s.xopt, source)
                        sh = exec.Command(s.interpreter, a...)
                }
                sh.Stdout, sh.Stderr = &exeres.Stdout, &exeres.Stderr
                if envarsOpt != nil {
                        if l, _ := envarsOpt.Value.(*types.List); l != nil {
                                for _, v := range l.Elems {
                                        if v, err = types.Disclose(context, v); err != nil {
                                                return
                                        } else {
                                                sh.Env = append(sh.Env, v.Strval())
                                        }
                                }
                        }
                }
                // TODO: ExecResult.VerboseStdout
                // TODO: ExecResult.VerboseStderr
                if verbout { sh.Stdout = os.Stdout }
                if verberr { sh.Stderr = os.Stderr }
                if stdin   { sh.Stdin = os.Stdin }
                if err = sh.Run(); err == nil {
                        exeres.Status, source = 0, ""
                } else {
                        var s = err.Error()
                        if n, e := fmt.Sscanf(s, "exit status %v", &exeres.Status); n == 1 && e == nil {
                                if statusOpt != nil && statusOpt.Value.Strval() == "on" {
                                        err = nil
                                } else {
                                        err = errors.New(fmt.Sprintf("%v (%s)", err, source))
                                }
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
