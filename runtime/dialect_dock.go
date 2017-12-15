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

const DockExecVarName = "dock->exec" // docker exec image

type dialectDock struct {
        monoInterpreter
}
func (s *dialectDock) dialect() string { return "dock" }
func (s *dialectDock) evaluate(prog *Program, context *types.Scope, args []types.Value, recipes []types.Value) (result types.Value, err error) {
        if args, err = types.JoinEval(context, args...); err != nil {
                return
        }

        var (
                envarsOpt, _ = prog.scope.Lookup("shell-envars").(*types.Def)
                //stdoutOpt, _ = prog.scope.Lookup("shell-stdout").(*types.Def)
                //stderrOpt, _ = prog.scope.Lookup("shell-stderr").(*types.Def)
                //stdinOpt,  _ = prog.scope.Lookup("shell-stdin").(*types.Def)
                wd = prog.Getwd(context)
                exeres = new(types.ExecResult)
                source string
                shi = "sh"
        )

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
                        _, obj = context.Find(DockExecVarName)
                        envars []types.Value // disclosed values
                )
                if obj == nil { _, obj = prog.scope.Find(DockExecVarName) }
                if obj != nil {
                        if v, e := obj.(types.Caller).Call(); e != nil {
                                err = e; return
                        } else if v == nil {
                                // nothing changed
                        } else {
                                if v, err = types.Disclose(context, v); err != nil {
                                        return
                                } else if s := strings.TrimSpace(v.Strval()); s != "" {
                                        dxi = s
                                }
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
                        verbout, verberr, stdin bool
                        a = []string{ "exec" }
                )
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
                                case "do": verbout = true; continue LoopArgs
                                case "de": verberr = true; continue LoopArgs
                                case "eo", "oe", "deo", "doe":
                                        verbout, verberr = true, true
                                        continue LoopArgs
                                case "i": 
                                        a = append(a, "-ti")
                                        stdin = true; continue LoopArgs
                                }
                        default:
                                shi = args[0].Strval()
                                continue LoopArgs
                        }
                        a = append(a, v.Strval())
                }
                a = append(a, dxi, shi, "-c", src)

                var sh *exec.Cmd
                sh = exec.Command("docker", a...)
                sh.Stdout, sh.Stderr = &exeres.Stdout, &exeres.Stderr
                if len(envars) > 0 {
                        sh.Env = os.Environ()
                        for _, env := range envars {
                                sh.Env = append(sh.Env, env.Strval())
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
        
        result = exeres
        return
}
