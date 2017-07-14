//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package types

import (
        //"github.com/duzy/smart/token"
        "path/filepath"
        "encoding/base64"
        "io/ioutil"
        "strings"
        "bytes"
        "fmt"
)

type BuiltinFunc func(args... Value) (Value, error)

var builtins = map[string]BuiltinFunc {
        `print`:   builtinPrint,
        `printl`:  builtinPrintl,
        `println`: builtinPrintln,

        `lit`:        builtinLit,
        `filter`:     builtinFilter,
        `filter-out`: builtinFilterOut,
        
        `encode-base64`:  builtinEncodeBase64,
        `decode-base64`:  builtinDecodeBase64,

        /* TODO:
        `encode-base32`
        `decode-base32`
        `encode-json`
        `decode-json`
        `encode-xml`
        `decode-xml`
        `encode-hex`
        `decode-hex`
        `encode-csv`
        `decode-csv` */
        
        `base`:    builtinBase,
        `dir`:     builtinDir,
        `dirdir`:  builtinDirDir,
        `ndir`:    builtinNDir,

        `read-file`: builtinReadFile,
}

func GetBuiltins() map[string]BuiltinFunc {
        return builtins
}

func EscapedString(v Value) (s string) {
        if v.Type() == String {
                s = strings.Replace(v.String(), "\\'", "'", -1)
        } else {
                s = v.String()
        }
        return
}

func builtinPrint(args... Value) (Value, error) {
        var x = len(args)
        for i, a := range args {
                if 0 < i && i < x {
                        fmt.Printf(" ")
                }
                fmt.Printf("%s", EscapedString(a))
        }
        return nil, nil
}

func builtinPrintl(args... Value) (Value, error) {
        var x = len(args)
        for i, a := range args {
                if 0 < i && i < x {
                        fmt.Printf(" ")
                }
                s := EscapedString(a)
                fmt.Printf("%s", s)
                if i == x && !strings.HasSuffix(s, "\n") {
                        fmt.Printf("\n")
                }
        }
        return nil, nil
}

func builtinPrintln(args... Value) (Value, error) {
        builtinPrint(args...)
        fmt.Printf("\n")
        return nil, nil
}

func builtinLit(args... Value) (result Value, err error) {
        var s string
        for _, a := range args {
                s += a.Lit()
        }
        return &StringValue{s}, nil
}

func builtinFilterValues(neg bool, args... Value) (res Value, err error) {
        if len(args) > 1 {
                var pats []Value
                switch pat := args[0].(type) {
                case *ListValue:
                        for _, elem := range pat.Elems {
                                pats = append(pats, elem)
                        }
                default:
                        pats = append(pats, pat)
                }
                f := func(v Value) bool {
                        for _, pat := range pats {
                                switch p := pat.(type) {
                                case *PercentPattern:
                                        if m, s := p.Match(v.String()); m && s != "" {
                                                //fmt.Printf("match: %v: %v (%v)\n", m, s, v)
                                                return true
                                        }
                                default:
                                        fmt.Printf("todo: %v (%T) (%v)\n", pat, pat, v)
                                }
                        }
                        return false
                }
                if len(pats) > 0 {
                        var elems []Value
                        for _, v := range EvalElems(args[1:]...) {
                                var okay = f(v)
                                if neg { okay = !okay }
                                if okay { elems = append(elems, v) }
                        }
                        if len(elems) > 0 {
                                res = &ListValue{Elements{elems}}
                        }
                }
        }
        if res == nil {
                res = UniversalNone
        }
        return
}

func builtinFilter(args... Value) (res Value, err error) {
        res, err = builtinFilterValues(false, args...)
        return
}

func builtinFilterOut(args... Value) (res Value, err error) {
        res, err = builtinFilterValues(true, args...)
        return
}

func builtinEncodeBase64(args... Value) (res Value, err error) {
        if len(args) > 0 {
                buf := new(bytes.Buffer)
                enc := base64.NewEncoder(base64.StdEncoding, buf)
                for _, a := range args {
                        enc.Write([]byte(a.String()))
                }
                enc.Close()
                res = &StringValue{buf.String()}
        }
        return
}

func builtinDecodeBase64(args... Value) (res Value, err error) {
        if len(args) > 0 {
                var list []Value
                for _, a := range args {
                        var dat []byte
                        dat, err = base64.StdEncoding.DecodeString(a.String())
                        if err == nil {
                                list = append(list, &StringValue{string(dat)})
                        } else {
                                return
                        }
                }
                if x := len(list); x == 0 {
                        res = UniversalNone
                } else if x == 1 {
                        res = list[0]
                } else {
                        res = &ListValue{Elements{list}}
                }
        }
        return
}

func builtinBase(args... Value) (Value, error) {
        var (
                l []Value
                s string
        )
        for _, a := range args {
                s = filepath.Base(a.String())
                l = append(l, &StringValue{s})
        }
        if len(l) == 1 {
                return l[0], nil
        } else {
                return &ListValue{Elements{l}}, nil
        }
}

func builtinDirDir(args... Value) (Value, error) {
        var (
                l []Value
                s string
        )
        for _, a := range args {
                s = filepath.Dir(filepath.Dir(a.String()))
                l = append(l, &StringValue{s})
        }
        if len(l) == 1 {
                return l[0], nil
        } else {
                return &ListValue{Elements{l}}, nil
        }
}

func builtinDir(args... Value) (Value, error) {
        var (
                l []Value
                s string
        )
        for _, a := range args {
                s = filepath.Dir(a.String())
                l = append(l, &StringValue{s})
        }
        if len(l) == 1 {
                return l[0], nil
        } else {
                return &ListValue{Elements{l}}, nil
        }
}

func builtinNDir(args... Value) (Value, error) {
        var (
                l []Value
                s string
                n = 0
        )
        if len(args) > 0 {
                n = int(args[0].Integer())
                args = args[1:]
        }
        for _, a := range args {
                s = filepath.Dir(a.String())
                for i := n-1; 0 < i; i -= 1 {
                        s = filepath.Dir(s)
                }
                l = append(l, &StringValue{s})
        }
        if len(l) == 1 {
                return l[0], nil
        } else {
                return &ListValue{Elements{l}}, nil
        }
}

func builtinReadFile(args... Value) (res Value, err error) {
        var l []Value
        for _, a := range args {
                var s []byte
                if s, err = ioutil.ReadFile(a.String()); err == nil {
                        l = append(l, &StringValue{string(s)})
                } else {
                        l = append(l, UniversalNone)
                }
        }
        if x := len(l); x == 0 {
                res = UniversalNone
        } else if x == 1 {
                res = l[0]
        } else {
                res = &ListValue{Elements{l}}
        }
        return
}
