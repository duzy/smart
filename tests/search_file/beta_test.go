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

func TestCheckCheckFiles(t *testing.T) {
        if _, err := os.Stat("hello"); err != nil { t.Error(err) }
        if _, err := os.Stat("obj/hello.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("src/hello.cpp"); err != nil { t.Error(err) }
        if _, err := os.Stat("hello.o"); err == nil { t.Error("unexpected file") }
        if _, err := os.Stat("hello.cpp"); err == nil { t.Error("unexpected file") }
}

func TestCheckLog1(t *testing.T) {
        if bv, err := ioutil.ReadFile("1.log"); err != nil { t.Error(err) } else {
                if s, a, b, h, f := extractTextOutput(bv); a != nil && bytes.Equal(a, b) {
                        var v = []byte(wd)
                        if !bytes.Equal(v, a) { t.Errorf("bad path:\n%s\n%s", v, a) }

                        v = []byte(fmt.Sprintf(`/opt/foobar
./foo/bar
`))
                        if !bytes.Equal(v, h) { t.Errorf("bad header:\n%s\n%s", v, h) }

                        v = []byte(fmt.Sprintf(`g++ -std=c++1z -c src/hello.cpp -o obj/hello.o
g++ -std=c++1z obj/hello.o -o hello
./hello
Hello World!
`))
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
