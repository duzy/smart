//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "bytes"
        "fmt"
        "os"
        "os/exec"
        "strings"
        "testing"
        "path/filepath"
        . "github.com/duzy/smart/build"
)

func ComputeTestRunParams() (vars map[string]string, cmds []string) {
        vars = map[string]string{}
        for _, arg := range os.Args[1:] {
                if arg[0] == '-' { continue }
                if i := strings.Index(arg, "="); 0 < i /* false at '=foo' */ {
                        vars[arg[0:i]] = arg[i+1:]
                        continue
                }
                cmds = append(cmds, arg)
        }
        SetFlagVV(true)
        SetFlagV(true)
        SetFlagL(false)
        return
}

func Runcmd(name string, args ...string) (string, error) {
        var buf bytes.Buffer
        cmd := exec.Command(name, args...)
        cmd.Stdout, cmd.Stderr = &buf, &buf
        if err := cmd.Run(); err != nil {
                //fmt.Printf("test: (run: %v) %v\n", name, err)
                return "", err
        }
        return buf.String(), nil
}

func RunToolsetTestCase(t *testing.T, tp, tn string, tf func(t *testing.T)) {
        tc := filepath.Join(tp, "test", tn)

        if fi, _ := os.Stat(tc); fi != nil && fi.IsDir() {
                fmt.Printf("test: no test `%v' (%v)\n", tc, tn)
        }

        wd, e := os.Getwd()
        if e != nil { t.Errorf("Getwd: %v", e); return }
        if e := os.Chdir(tc); e != nil { t.Errorf("Chdir: %v", e); return }
        fmt.Printf("test: Entering directory `%v'\n", tc)

        tf(t)

        fmt.Printf("test: Leaving directory `%v'\n", tc)
        if e := os.Chdir(wd); e != nil { t.Errorf("Chdir: %v", e); return }
}
