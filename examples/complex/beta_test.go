//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smartest

import (
        "io/ioutil"
        "testing"
        "bytes"
        "fmt"
        "os"
)

var wd, _ = os.Getwd()

func TestCheckFiles(t *testing.T) {
        if _, err := os.Stat("many/hello"); err != nil { t.Error(err) }
        if _, err := os.Stat("many/hello.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("many/hello.cpp"); err != nil { t.Error(err) }
        if _, err := os.Stat("many/greeting/libgreeting.a"); err != nil { t.Error(err) }
        if _, err := os.Stat("many/greeting/obj/greeting.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("many/greeting/src/greeting.c"); err != nil { t.Error(err) }
        if _, err := os.Stat("many/greeting/include/greeting.h"); err != nil { t.Error(err) }
        if _, err := os.Stat("hello"); err == nil { t.Error("unexpected file") }
        if _, err := os.Stat("hello.o"); err == nil { t.Error("unexpected file") }
        if _, err := os.Stat("hello.cpp"); err == nil { t.Error("unexpected file") }
        if _, err := os.Stat("greeting.o"); err == nil { t.Error("unexpected file") }
        if _, err := os.Stat("greeting.c"); err == nil { t.Error("unexpected file") }
        if _, err := os.Stat("greeting.h"); err == nil { t.Error("unexpected file") }
        if _, err := os.Stat("libgreeting.a"); err == nil { t.Error("unexpected file") }
        if _, err := os.Stat("many/greeting.o"); err == nil { t.Error("unexpected file") }
        if _, err := os.Stat("many/greeting.c"); err == nil { t.Error("unexpected file") }
        if _, err := os.Stat("many/greeting.h"); err == nil { t.Error("unexpected file") }
        if _, err := os.Stat("many/libgreeting.a"); err == nil { t.Error("unexpected file") }
        if _, err := os.Stat("many/greeting/greeting.h"); err == nil { t.Error("unexpected file") }
        if _, err := os.Stat("many/greeting/greeting.c"); err == nil { t.Error("unexpected file") }
        if _, err := os.Stat("many/greeting/greeting.o"); err == nil { t.Error("unexpected file") }
}

func TestCheckLog1(t *testing.T) {
        if bv, err := ioutil.ReadFile("1.log"); err != nil { t.Error(err) } else {
                if s, a, b, h, f := extractTextOutput(bv); a != nil && bytes.Equal(a, b) {
                        var v = []byte(wd)
                        if !bytes.Equal(v, a) { t.Errorf("bad path:\n%s\n%s", v, a) }

                        v = []byte(fmt.Sprintf(``))
                        if !bytes.Equal(v, h) { t.Errorf("bad header:\n%s\n%s", v, h) }

                        v = []byte(fmt.Sprintf(`smart: Entering directory '%s/many'
smart: Entering directory '%s/many/greeting'
update file 'include/greeting.h' ... (ok)
test "%s/many/greeting" = "%s/many/greeting"
test -f include/greeting.h
gcc -DXXX -Iinclude -std=c++1z -c greeting.cpp -o greeting.o
ar rvs libgreeting.a greeting.o
a greeting.o
smart:  Leaving directory '%s/many/greeting'
update file 'hello.cpp' ... (ok)
g++ -DXXX -Iinclude -I%s/many/greeting/include -DTEST=1 -std=c++1z -c hello.cpp -o hello.o
g++ -DTEST=1 -std=c++1z hello.o   -o hello
./hello
Hello World!
smart:  Leaving directory '%s/many'
`, wd, wd, wd, wd, wd, wd))
                        if !bytes.Equal(v, s) { t.Errorf("bad output:\n%s\n%s", v, s) }

                        v = []byte(``)
                        if !bytes.Equal(v, f) { t.Errorf("bad footer:\n%s\n%s", v, f) }
                } else { t.Errorf("path: '%s' != '%s'", a, b) }
        }
}

func TestCheckLog2(t *testing.T) {
        if bv, err := ioutil.ReadFile("2.log"); err != nil { t.Error(err) } else {
                if !bytes.Equal(bv, nil) { t.Errorf("bad output:\n%s", bv) }
        }
}
