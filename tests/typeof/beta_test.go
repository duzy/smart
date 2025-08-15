//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
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

func TestCheckLog1(t *testing.T) {
        if bv, err := ioutil.ReadFile("1.log"); err != nil { t.Error(err) } else {
                if s, a, b, h, f := extractTextOutput(bv); a != nil && bytes.Equal(a, b) {
                        var v = []byte(wd)
                        if !bytes.Equal(v, a) { t.Errorf("bad path:\n%s\n%s", v, a) }

                        v = []byte(fmt.Sprintf(`glob
word
int
int
int
float
float
oct
hex
hex
bin
bin
date
datetime
datetime
list
time
word
string
strcomp
path /path/to/somewhere
path ./subdir/in/somewhere
barefile
barefile
compound
compound
group
pair
pair
pair
flag
word
word
word
word
word
word
word
word
barefile
barefile
barefile
compound
barefile word word group
barefile list list
compound -I/path/to/include
compound -I%s/include
list
`, wd))
                        if !bytes.Equal(v, h) { t.Errorf("bad header:\n%s\n%s", v, h) }

                        v = []byte(fmt.Sprintf(``))
                        if !bytes.Equal(v, s) { t.Errorf("bad output:\n%s\n%s", v, s) }

                        v = []byte(fmt.Sprintf(``))
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
