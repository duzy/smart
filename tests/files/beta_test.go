//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smartest

import (
        "testing"
        "io/ioutil"
        "bytes"
        "fmt"
        "os"
)

var wd, _ = os.Getwd()

func TestCheckFiles(t *testing.T) {
        if _, err := os.Stat("1.log"); err != nil { t.Error(err) }
        if _, err := os.Stat("2.log"); err != nil { t.Error(err) }
        if _, err := os.Stat("hello"); err != nil { t.Error(err) }
        if _, err := os.Stat("alt/foo.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("alt/other.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("obj/1.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("obj/2.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("obj/3.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("obj/foo.1.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("obj/fo.o.2.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("obj/f.o.o.3.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("obj/f.o.o.4.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("obj/foo.5.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("obj/foo.6.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("obj/main.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("src/baz1.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("src/baz2.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("src/baz3.o"); err != nil { t.Error(err) }
        if _, err := os.Stat("src/baz4.o"); err != nil { t.Error(err) }
}

func TestCheckLog1(t *testing.T) {
        if bv, err := ioutil.ReadFile("1.log"); err != nil { t.Error(err) } else {
                if s, a, b, h, f := extractTextOutput(bv); a != nil && bytes.Equal(a, b) {
                        var v = []byte(wd)
                        if !bytes.Equal(v, a) { t.Errorf("bad path:\n%s\n%s", v, a) }

                        v = []byte(fmt.Sprintf(``))
                        if !bytes.Equal(v, h) { t.Errorf("bad header:\n%s\n%s", v, h) }

                        v = []byte(fmt.Sprintf(`g++ -std=c++1z -c alt/main.cc -o obj/main.o
g++ -std=c++1z -c alt/other.c++ -o alt/other.o
gcc -c -o obj/foo.1.o foo.1.c
g++ -std=c++1z -c fo.o.2.cc -o obj/fo.o.2.o
g++ -std=c++1z -c f.o.o.3.C -o obj/f.o.o.3.o
g++ -std=c++1z -c f.o.o.4.c++ -o obj/f.o.o.4.o
g++ -std=c++1z -c foo.5.cxx -o obj/foo.5.o
g++ -std=c++1z -c foo.6.cpp -o obj/foo.6.o
g++ -std=c++1z -c bar/1.c++ -o obj/1.o
g++ -std=c++1z -c bar/2.c++ -o obj/2.o
g++ -std=c++1z -c bar/3.c++ -o obj/3.o
gcc -c -o alt/foo.o alt/foo.c
gcc -c -o src/baz1.o src/baz1.c
g++ -std=c++1z -c src/baz2.cpp -o src/baz2.o
g++ -std=c++1z -c src/baz3.cc -o src/baz3.o
g++ -std=c++1z -c src/baz4.C -o src/baz4.o
g++ -std=c++1z obj/main.o alt/other.o obj/foo.1.o obj/fo.o.2.o obj/f.o.o.3.o obj/f.o.o.4.o obj/foo.5.o obj/foo.6.o obj/1.o obj/2.o obj/3.o alt/foo.o src/baz1.o src/baz2.o src/baz3.o src/baz4.o   -o hello
./hello
foo
foo_1
foo_2
foo_3
foo_4
foo_5
foo_6
baz_1
`))
                        if !bytes.Equal(v, s) { t.Errorf("bad output:\n%s\n%s", v, s) }

                        v = []byte(``)
                        if !bytes.Equal(v, f) { t.Errorf("bad footer:\n%s\n%s", v, f) }
                } else { t.Errorf("path: '%s' != '%s'", a, b) }
        }
}

func TestCheckLog2(t *testing.T) {
        if v, err := ioutil.ReadFile("2.log"); err != nil { t.Error(err) } else {
                s := []byte(``)
                if !bytes.Equal(v, s) { t.Errorf("bad output:\n%s\n%s", v, s) }
        }
}
