//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//

/*
        Package smart builds complex project faster in the simple way.


*/
package smart

import (
        "flag"
        "fmt"
        "strings"
)

var (
        flagA = flag.Bool("a", false, "auto mode")
        flagM = flag.Bool("m", false, "searching module for targets")
        flagG = flag.Bool("g", false, "searching global targets")
        flagGG = flag.Bool("G", true, "ignore names like \".git\", \".svn\", etc.")
        flagJ = flag.Int("j", 3, "Allow N jobs at once.")
        flagO = flag.String("o", "", "output directory")
        flagC = flag.String("C", "", "change directory")
        flagT = flag.String("T", "", "traverse")
        flagV = flag.Bool("v", false, "prompt command")
        flagVV= flag.Bool("V", false, "print command verbosely")
        flagW = flag.Bool("w", false, "warn undefined symbols")
        flagL = flag.Bool("l", false, "warn undefined symbols")
        flagCD = flag.Bool("cd", true, "chdir when building")
)

var hc = MustHookup(`
template smart
post
commit
`, HooksMap{
        "smart": HookTable{
                //"configure": hook_configure,
        },
})

func GetFlagA() bool    { return *flagA }
func GetFlagM() bool    { return *flagM }
func GetFlagG() bool    { return *flagG }
func GetFlagGG() bool   { return *flagGG }
func GetFlagJ() int     { return *flagJ }
func GetFlagO() string  { return *flagO }
func GetFlagC() string  { return *flagC }
func GetFlagT() string  { return *flagT }
func GetFlagL() bool    { return *flagL }
func GetFlagV() bool    { return *flagV }
func GetFlagVV() bool   { return *flagVV }
func GetFlagCD() bool   { return *flagCD }

func SetFlagA(v bool)   { *flagA = v }
func SetFlagM(v bool)   { *flagM = v }
func SetFlagG(v bool)   { *flagG = v }
func SetFlagGG(v bool)  { *flagGG = v }
func SetFlagJ(v int)    { *flagJ = v }
func SetFlagO(v string) { *flagO = v }
func SetFlagC(v string) { *flagC = v }
func SetFlagT(v string) { *flagT = v }
func SetFlagL(v bool)   { *flagL = v }
func SetFlagV(v bool)   { *flagV = v }
func SetFlagVV(v bool)  { *flagVV = v }
func SetFlagCD(v bool)  { *flagCD = v }

type smarterror struct {
        message string
}

func (e *smarterror) String() string {
        return e.message
}

func Fatal(f string, a ...interface{}) {
        errorf(f, a...)
}

func Message(s string, a ...interface{}) {
        message(s, a...)
}

func Verbose(s string, a ...interface{}) {
        verbose(s, a...)
}

// errorf throw a panic message
func errorf(f string, a ...interface{}) {
        panic(&smarterror{ fmt.Sprintf(f, a...) })
}

// verbose prints a message if `V' flag is enabled
func verbose(s string, a ...interface{}) {
        if *flagVV {
                message(s, a...)
        }
}

// message prints a message
func message(s string, a ...interface{}) {
        if !strings.HasPrefix(s, "smart:") {
                s = "smart: " + s
        }
        if !strings.HasSuffix(s, "\n") {
                s = s + "\n"
        }
        fmt.Printf(s, a...)
}

// Split splits a string by space or tab
func Split(str string) (items []string) {
        /*
        a := strings.Split(str, " ")
        for _, s := range a {
                if strings.TrimSpace(s) != "" {
                        items = append(items, s)
                }
        } */
        items = strings.Fields(str)
        return
}

// splitVarArgs split arguments in the form of "NAME=value" with the others.
func splitVarArgs(args []string) (vars map[string]string, rest []string) {
        vars = make(map[string]string, 10)

        for _, arg := range args {
                if i := strings.Index(arg, "="); 0 < i /* false at '=foo' */ {
                        vars[strings.TrimSpace(arg[0:i])] = strings.TrimSpace(arg[i+1:])
                } else {
                        rest = append(rest, arg)
                }
        }

        return
}

// Main starts build from the command line.
func Main() {
        flag.Parse()

        vars, cmds := splitVarArgs(flag.Args())

        Build(vars, cmds...)
}
