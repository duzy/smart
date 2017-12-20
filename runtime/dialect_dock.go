//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        "os/exec"
        "strings"
        //"bytes"
        "fmt"
        "os"
)

const (
        DockImageVarName = "dock->image" // docker image
        DockExecVarName = "dock->exec" // docker exec image
)

type dialectDock struct {
        monoInterpreter
}
func (s *dialectDock) dialect() string { return "dock" }
func (s *dialectDock) evaluate(prog *Program, context *types.Scope, args []types.Value, recipes []types.Value) (result types.Value, err error) {
        if args, err = types.JoinEval(context, args...); err != nil {
                return
        }

        var (
                wd = prog.Getwd(context)
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
                        dxi = "smart-dock-image" // context.Find(DockImageVarName)
                        _, obj = context.Find(DockExecVarName)
                        envars []types.Value // disclosed values
                )
                if obj == nil { _, obj = prog.scope.Find(DockExecVarName) }
                if obj != nil {
                        if v, e := obj.(types.Caller).Call(); e != nil {
                                err = e; return
                        } else if v == nil {
                                // nothing changed
                        } else if v, err = types.Disclose(context, v); err != nil {
                                return
                        } else if s := strings.TrimSpace(v.Strval()); s != "" {
                                dxi = s
                        }
                }
                
                if envarsDef, _ := prog.scope.Lookup(theShellEnvarsDef).(*types.Def); envarsDef != nil {
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
                                case "do": verbout = true; continue ForArgs
                                case "de": verberr = true; continue ForArgs
                                case "eo", "oe", "deo", "doe":
                                        verbout, verberr = true, true
                                        continue ForArgs
                                case "i": 
                                        stdin, a = true, append(a, "-ti")
                                        continue ForArgs
                                }
                        default:
                                shi = args[0].Strval()
                                continue ForArgs
                        }
                        a = append(a, v.Strval())
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

                var sh = exec.Command(cmd, a...)
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
                                err = fmt.Errorf("%v", err) // , source
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
