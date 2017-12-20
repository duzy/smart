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
                // TODO: parsing envars and status flags from `args'
                envarsDef, _ = prog.scope.Lookup(theShellEnvarsDef).(*types.Def)
                exeres = new(types.ExecResult)
                envars []types.Value // disclosed values
                source string
                silent bool
        )
        if envarsDef != nil {
                if l, _ := envarsDef.Value.(*types.List); l != nil {
                        for _, v := range l.Elems {
                                if v, err = types.Disclose(context, v); err != nil {
                                        return
                                } else {
                                        envars = append(envars, v)
                                }
                        }
                }
        }
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

                /*if s := ""; len(envars) > 0 {
                        for i, env := range envars {
                                if i > 0 { s += " && " }
                                p := env.(*types.Pair)
                                s += "export "
                                s += p.Key.Strval() + "=\""
                                s += p.Value.Strval() + "\""
                        }
                        source = fmt.Sprintf("%s && %s", s, source)
                }*/

                var (
                        verbout, verberr, stdin bool
                        sh *exec.Cmd
                )
                if len(args) == 0 {
                        sh = exec.Command(s.interpreter, s.xopt, source)
                } else {
                        var a []string
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
                                        case "s": silent = true; continue ForArgs
                                        case "i": stdin = true; continue ForArgs
                                        case "do": verbout = true; continue ForArgs
                                        case "de": verberr = true; continue ForArgs
                                        case "eo", "oe", "deo", "doe":
                                                verbout, verberr = true, true
                                                continue ForArgs
                                        }
                                }
                                a = append(a, v.Strval())
                        }
                        a = append(a, s.xopt, source)
                        sh = exec.Command(s.interpreter, a...)
                }
                sh.Stdout, sh.Stderr, sh.Env = &exeres.Stdout, &exeres.Stderr, os.Environ()
                for _, v := range envars {
                        if v, err = types.Disclose(context, v); err != nil {
                                return
                        } else {
                                sh.Env = append(sh.Env, v.Strval())
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
                                if statusDef := prog.auto(theShellStatusDef, values.Int(int64(exeres.Status))); statusDef == nil {
                                        // FIXME: error
                                }
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
        
        if /* TODO: using `--verbose-shell` to control this */false {
                fmt.Printf("%v", exeres.Stdout.String())
        }
        
        result = exeres
        return
}
