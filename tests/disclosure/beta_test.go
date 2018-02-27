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
        if _, err := os.Stat("test/foobar"); err == nil { t.Error("unexpected file") }
        if _, err := os.Stat("default/foobar"); err != nil { t.Error(err) }
}

func TestCheckLog1(t *testing.T) {
        if bv, err := ioutil.ReadFile("1.log"); err != nil { t.Error(err) } else {
                if s, a, b, h, f := extractTextOutput(bv); a != nil && bytes.Equal(a, b) {
                        var v = []byte(wd)
                        if !bytes.Equal(v, a) { t.Errorf("bad path:\n%s\n%s", v, a) }

                        v = []byte(fmt.Sprintf(`test
%s/test
test
%s/test
none: 
`, wd, wd))
                        if !bytes.Equal(v, h) { t.Errorf("bad header:\n%s\n%s", v, h) }

                        v = []byte(fmt.Sprintf(`%s
%s
default/foobar
default
.
%s
smart: Entering directory '%s/test'
smart: Entering directory '%s/test/plains'
smart:  Leaving directory '%s/test/plains'
smart: Entering directory '%s/test/plains'
smart:  Leaving directory '%s/test/plains'
smart:  Leaving directory '%s/test'
disclosure: plain-foo-1 default; disclosure: plain-foo-2 default; disclosure: plain-foo-3 default; plains: disclosure: plain-foo-1 default disclosure: plain-foo-2 default 'plain-foo-3'
test-plains-foo
smart: Entering directory '%s/test'
smart: Entering directory '%s/test/plains'
smart:  Leaving directory '%s/test/plains'
smart: Entering directory '%s/test/plains'
smart:  Leaving directory '%s/test/plains'
smart:  Leaving directory '%s/test'
`, wd, wd, wd, wd, wd, wd, wd, wd, wd, wd, wd, wd, wd, wd, wd))
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
