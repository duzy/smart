//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "path/filepath"
        "os/exec"
        "strings"
        "strconv"
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

type executor struct {
        cmd, opt string
        bare bool
}

func docksFindObj(docks []*Project, name string) (obj Object) {
        for _, dock := range docks {
                if obj, _ = dock.resolveObject(name); obj != nil {
                        break
                }
        }
        return
}

func docksFindEnt(docks []*Project, name string) (entry *RuleEntry) {
        for _, dock := range docks {
                if entry, _ = dock.resolveEntry(name); entry != nil {
                        break
                }
        }
        return
}

func (p *executor) runContainer(prog *Program, docks []*Project) (err error) {
        if run := docksFindEnt(docks, "run"); run != nil {
                _, err = run.Execute(prog.Position()/*, &String{`sh -c "while sleep 3600; do :; done"`}*/)
        } else {
                err = fmt.Errorf("dock=>run undefined")
        }
        return
}

func (p *executor) ensureContainerRunning(prog *Program, docks []*Project, container string) (err error) {
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
        } (stderrR)

        if err = cmd.Run(); err == nil {
                if foundID == "" {
                        if err = p.runContainer(prog, docks); err == nil {
                                time.Sleep(time.Second)
                        }
                }
        }
        return
}

func (p *executor) Evaluate(prog *Program, args []Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        var prompt, verbout, verberr, saveout, saveerr, stdin, silent, nocd bool
        var cmd, promstr = p.cmd, ""
        var aa []string
        var opts = []string{
                "o,stdout",
                "e,stderr",
                "v,verbout",
                "w,verberr",
                "p,prompt", // --verbose-shell
                "i,stdin",
                "s,silent",
                "d,dump", // verbout, verberr
                "nocd",
        }
ForArgs:
        for i, v := range args {
                if !p.bare && i == 0 {
                        var s string
                        if s, err = v.Strval(); err != nil { return }
                        if s == "shell" { cmd = defaultShell }
                        continue ForArgs
                }

                var ( runes []rune ; names []string ; s string )
                switch t := v.(type) {
                case *Pair:
                        if flag, _ := t.Key.(*Flag); flag != nil {
                                if runes, names, err = flag.opts(opts...); err != nil { return } else {
                                        v = t.Value
                                }
                        } else {
                                err = fmt.Errorf("`%v` unsupported", t)
                                return
                        }
                case *Flag:
                        if runes, names, err = t.opts(opts...); err != nil { return }
                        v = nil // no flag value
                default:
                        if s, err = v.Strval(); err != nil { return } else {
                                aa = append(aa, s)
                        }
                        continue ForArgs
                }

                for i, ru := range runes {
                        switch ru {
                        case 'i': stdin   = true
                        case 'o': saveout = true
                        case 'e': saveerr = true
                        case 'v': verbout = true
                        case 'w': verberr = true
                        case 's': silent  = true
                        case 'p':
                                if v == nil {
                                        prompt = true
                                } else if s, err = v.Strval(); err == nil {
                                        prompt, promstr = true, s
                                } else {
                                        return
                                }
                        case 'd': // -dump=xxx or -d=xxx
                                if v == nil {
                                        verbout, verberr = true, true
                                } else if s, err = v.Strval(); err == nil {
                                        switch s {
                                        case "stdout": verbout = true
                                        case "stderr": verberr = true
                                        case "all":
                                                verbout = true
                                                verberr = true
                                        }
                                } else {
                                        return
                                }
                        case 0:
                                switch names[i] {
                                case "nocd": nocd = true
                                }
                        }
                }
        }

        var recipes []Value
        if recipes, err = mergeresult(ExpandAll(prog.recipes...)); err != nil { return }

        var docks []*Project
        if !p.bare {
                if prog.Project().Name() == "dock" {
                        docks = append(docks, prog.Project())
                } else {
                        for _, scope := range cloctx {
                                if _, sym := scope.Find("dock"); sym != nil {
                                        if p, ok := sym.(*ProjectName); ok && p != nil {
                                                docks = append(docks, p.NamedProject())
                                        }
                                }
                        }
                        if docks == nil {
                                if _, dockSym := prog.Project().Scope().Find("dock"); dockSym != nil {
                                        if pn, _ := dockSym.(*ProjectName); pn != nil {
                                                docks = append(docks, pn.NamedProject())
                                        }
                                }
                        }
                }

                if docks == nil {
                        err = fmt.Errorf("docking unavailable (in %s)", prog.Project().Name())
                        return
                }

                defer setclosure(scoping(docks...))

                var strval = func(name string) (str string, err error) {
                        if obj := docksFindObj(docks, name); obj != nil {
                                if def, _ := obj.(*Def); def != nil {
                                        var v Value
                                        if v, err = def.DiscloseValue(); err == nil && v != nil {
                                                if str, err = v.Strval(); str == "-" {
                                                        /*if v, err = def.DiscloseValue(docks); err == nil && v != nil {
                                                        if str, err = v.Strval(); str == "" { str = "-" }
                                                        fmt.Printf("%v: %v (%v)\n", name, str, def)
                                                }*/
                                                }
                                        }
                                }
                        }
                        return
                }

                var container, image string
                if container, err = strval("dock-container"); err != nil { return }
                if container == "" { err = fmt.Errorf("dock-container undefined"); return }
                if image, err = strval("dock-image"); err != nil { return }
                if image == "" { err = fmt.Errorf("dock-image undefined"); return }
                if container != "-" && image != "-" {
                        aa = append(aa, "exec", container, cmd)
                        cmd = "docker"
                }

                if false {
                        if err = p.ensureContainerRunning(prog, docks, container); err != nil {
                                return
                        }
                }
        }

        var source, str string
        var exeres = new(ExecResult)
        if saveout { exeres.Stdout.Buf = new(bytes.Buffer) }
        if saveerr { exeres.Stderr.Buf = new(bytes.Buffer) }
        if verbout { exeres.Stdout.Tie = os.Stdout }
        if verberr { exeres.Stderr.Tie = os.Stderr }
        exeres.Stderr.Line = rxKnownErrors

        printEnteringDirectory()
        if prompt {
                var target = prog.scope.Lookup("@").(*Def).Value
                var targetName string
                if targetName, err = target.Strval(); err != nil {
                        return
                }

                var proj = mostDerived()
                var trims = []Value{
                        prog.scope.FindDef("CWD").Value,
                        prog.scope.FindDef("CTD").Value,
                        proj.scope.FindDef("CWD").Value,
                        proj.scope.FindDef("CTD").Value,
                }
                for _, v := range trims {
                        var s string
                        if s, err = v.Strval(); err != nil { return }
                        if strings.HasPrefix(targetName, s) {
                                targetName = strings.TrimPrefix(targetName, s)
                                targetName = strings.TrimPrefix(targetName, PathSep)
                        }
                }
                
                if promstr == "" {
                        fmt.Printf("smart: gen %s\n", targetName)
                } else {
                        fmt.Printf("%s: %s\n", promstr, targetName)
                }
        }
        ForRecipes: for _, recipe := range recipes {
                if str, err = recipe.Strval(); err != nil { return }
                if source += str; strings.HasSuffix(source, "\\") {
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
                } else if !prompt {
                        var s = source
                        s = strings.Replace(s, "\n", "\\n", -1)
                        s = strings.Replace(s, "\\\\n", "\\\n", -1)
                        fmt.Printf("%s\n", s)
                }

                var src = source
                var envars []Value // disclosed values
                if def, _ := prog.Scope().Lookup(TheShellEnvarsDef).(*Def); def != nil {
                        if l, _ := def.Value.(*List); l != nil {
                                for _, v := range l.Elems {
                                        if v, err = v.expand(expandClosure); err != nil {
                                                return
                                        } else {
                                                envars = append(envars, v)
                                        }
                                }
                        }
                }

                var str string
                var wd = prog.scope.Lookup("CWD").(*Def).Value //Call(pos)
                if str, err = wd.Strval(); err != nil { return }
                if str != "" || nocd {
                        if false {
                                fmt.Printf("dock.evaluate: %s\n", str)
                        }
                        if t := strings.TrimSpace(source); t == "" {
                                src = fmt.Sprintf("cd '%s'", str)
                        } else if strings.HasPrefix(t, "#") {
                                src = fmt.Sprintf("cd '%s' %s", str, t)
                        } else {
                                // Insert a "\n" before the right paren ')' to ensure that
                                // it's working with something like "true #comment...".
                                src = fmt.Sprintf("cd '%s' && (%s\n)", str, t)
                        }
                        if str = ""; len(envars) > 0 {
                                for i, env := range envars {
                                        var k, v string
                                        var p = env.(*Pair)
                                        if k, err = p.Key.Strval(); err != nil { return }
                                        if v, err = p.Value.Strval(); err != nil { return }
                                        if i > 0 { str += " && " }
                                        str += fmt.Sprintf(`%s="%s"`, k, strconv.Quote(v))
                                }
                                src = fmt.Sprintf("%s && %s", str, src)
                        }
                }

                var skips = make(map[string]bool)
                var sh = exec.Command(cmd, aa...)
                sh.Stdout, sh.Stderr = &exeres.Stdout, &exeres.Stderr
                if stdin {
                        sh.Stdin = os.Stdin
                        sh.Args = append(sh.Args, "-ti")
                }
                sh.Args = append(sh.Args, p.opt, src)
                sh.Env = os.Environ()
                for _, v := range envars {
                        if v, err = v.expand(expandClosure); err != nil {
                                return
                        } else if str, err = v.Strval(); err == nil {
                                sh.Env = append(sh.Env, str)
                        } else {
                                return
                        }
                }

        RunCommand:
                var num = 0
                exeres.Stderr.Subm = nil
                if err, num = sh.Run(), num+1; err == nil {
                        exeres.Status, source = 0, ""
                        continue ForRecipes
                }

                // Parse errors of execution
                if n, e := fmt.Sscanf(err.Error(), "exit status %v", &exeres.Status); n == 1 && e == nil {
                        if exeres.Stderr.Subm != nil {
                                var errstr = string(exeres.Stderr.Subm[0][0][0])
                                if errstr == errNotTTYDevice {
                                        if num > 2 { break } // only retry once
                                        fmt.Printf("smart: good to retry (%s)\n", source)
                                        c := exec.Command(sh.Path, sh.Args...)
                                        c.Stdout, c.Stderr, c.Stdin, c.Env = sh.Stdout, sh.Stderr, sh.Stdin, sh.Env
                                        sh = c; goto RunCommand // retry the command
                                } else if m := rxCompilation.FindAllStringSubmatch(errstr, -1); m != nil {
                                        err = fmt.Errorf("%s", m[0][4])
                                } else if m := rxFileNotFound.FindAllStringSubmatch(errstr, -1); m != nil {
                                        err = fmt.Errorf("`%v` file not found, required by `%s`", m[0][4], filepath.Base(m[0][1]))
                                } else if matched, _ := regexp.MatchString(errNoNetwork, errstr); matched {
                                        // TODO: dealing with network not found error
                                } else if false && docks != nil {
                                        // retry the command
                                        var name = string(exeres.Stderr.Subm[0][0][1])
                                        if v, ok := skips[name]; !ok && !v {
                                                skips[name] = true
                                                if err = p.runContainer(prog, docks); err == nil {
                                                        //fmt.Printf("smart: started %s (name=%s)\n", container, name) // name
                                                        fmt.Printf("smart: started %s\n", name) // name
                                                        c := exec.Command(sh.Path, sh.Args...)
                                                        c.Stdout, c.Stderr, c.Stdin, c.Env = sh.Stdout, sh.Stderr, sh.Stdin, sh.Env
                                                        sh = c; goto RunCommand
                                                }
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
        
        result = exeres
        return
}
