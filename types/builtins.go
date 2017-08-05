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

type BuiltinFunc func(context *Scope, args... Value) (Value, error)

var builtins = map[string]BuiltinFunc {
        /* TODO:
        `or`:    builtinLogicalOr,
        `and`:   builtinLogicalAnd,
        `xor`:   builtinLogicalXor,
        `not`:   builtinLogicalNot,

        `if`:    builtinBranchIf, */
        
        `print`:   builtinPrint,
        `printl`:  builtinPrintl,
        `println`: builtinPrintln,

        `string`:  builtinString,

        // https://www.gnu.org/software/make/manual/html_node/Text-Functions.html
        //`subst`:      builtinSubst,
        `patsubst`:   builtinPatsubst,

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

func EscapedString(v Value) (s string) {
        if v.Type() == StringType {
                s = strings.Replace(v.Strval(), "\\'", "'", -1)
        } else {
                s = v.Strval()
        }
        return
}

func builtinPrint(context *Scope, args... Value) (Value, error) {
        var x = len(args)
        for i, a := range args {
                if 0 < i && i < x {
                        fmt.Printf(" ")
                }
                fmt.Printf("%s", EscapedString(a))
        }
        return nil, nil
}

func builtinPrintl(context *Scope, args... Value) (Value, error) {
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

func builtinPrintln(context *Scope, args... Value) (Value, error) {
        builtinPrint(context, args...)
        fmt.Printf("\n")
        return nil, nil
}

func builtinString(context *Scope, args... Value) (result Value, err error) {
        s := new(bytes.Buffer)
        for i, a := range args {
                if i > 0 { s.WriteString(" ") }
                s.WriteString(a.String())
        }
        return &String{s.String()}, nil
}

func builtinFilterValues(context *Scope, neg bool, args... Value) (res Value, err error) {
        if len(args) > 1 {
                var pats []Value
                switch pat := args[0].(type) {
                case *List:
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
                                        if m, s := p.Match(v.Strval()); m && s != "" {
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
                                res = &List{Elements{elems}}
                        }
                }
        }
        if res == nil {
                res = UniversalNone
        }
        return
}

func builtinSubst(context *Scope, args... Value) (res Value, err error) {
        // $(subst from,to,text)
        return
}

// TODO:
//   $(var:pattern=replacement)
//   $(var:suffix=replacement)
func builtinPatsubst(context *Scope, args... Value) (res Value, err error) {
        // $(patsubst pattern,replacement,text)
        var list []Value
        if nargs := len(args); nargs > 2 {
                for _, arg := range EvalElems(args[2:]...) {
                        var (
                                s string // stemp
                                m bool // matched
                        )
                        if pat, _ := args[0].(*PercentPattern); pat != nil {
                                m, s = pat.Match(arg.Strval())
                        } else if l, _ := args[0].(*List); l != nil {
                                for _, elem := range l.Elems {
                                        if pat, _ := elem.(*PercentPattern); pat != nil {
                                                m, s = pat.Match(arg.Strval())
                                                if m && s != "" {
                                                        break
                                                }
                                        }
                                }
                        }

                        if m && s != "" {
                                if rep, ok := args[1].(*PercentPattern); ok {
                                        s = rep.Prefix.Strval() + s + rep.Suffix.Strval()
                                } else if l, _ := args[1].(*List); l != nil {
                                        var str = s
                                        for _, elem := range l.Elems {
                                                if rep, _ := elem.(*PercentPattern); rep != nil {
                                                        str = rep.Prefix.Strval() + str + rep.Suffix.Strval()
                                                        break
                                                }
                                        }
                                        s = str
                                } else {
                                        s = args[1].Strval()
                                }
                                
                                switch arg.(type) {
                                case *Barefile:
                                        ext := filepath.Ext(s)
                                        if len(ext) > 0 {
                                                s = s[:len(s)-len(ext)]
                                                ext = ext[1:]
                                        }
                                        list = append(list, &Barefile{
                                                Name: &Bareword{s},
                                                Ext: ext,
                                        })
                                default:
                                        list = append(list, &String{s})
                                }
                        } else if arg.Type().Kind() != NoneKind {
                                fmt.Printf("%T %p %v %v\n", arg, arg, m, s)
                                list = append(list, arg)
                        }
                }
        }
        res = &List{Elements{list}}
        return
}

func builtinStrip(context *Scope, args... Value) (res Value, err error) {
        // $(strip string)
        return
}

func builtinFindstring(context *Scope, args... Value) (res Value, err error) {
        // $(findstring find,text)
        return
}

func builtinFilter(context *Scope, args... Value) (res Value, err error) {
        // $(filter pattern…,text)
        res, err = builtinFilterValues(context, false, args...)
        return
}

func builtinFilterOut(context *Scope, args... Value) (res Value, err error) {
        // $(filter-out pattern…,text)
        res, err = builtinFilterValues(context, true, args...)
        return
}

func builtinSort(context *Scope, args... Value) (res Value, err error) {
        // $(sort list)
        return
}

func builtinWord(context *Scope, args... Value) (res Value, err error) {
        // $(word n,text)
        return
}

func builtinWordList(context *Scope, args... Value) (res Value, err error) {
        // $(wordlist s,e,text)
        return
}

func builtinWords(context *Scope, args... Value) (res Value, err error) {
        // $(words n,text)
        return
}

func builtinFirstWord(context *Scope, args... Value) (res Value, err error) {
        // $(firstword names...)
        return
}

func builtinLastWord(context *Scope, args... Value) (res Value, err error) {
        // $(lastword names...)
        return
}

func builtinEncodeBase64(context *Scope, args... Value) (res Value, err error) {
        if len(args) > 0 {
                buf := new(bytes.Buffer)
                enc := base64.NewEncoder(base64.StdEncoding, buf)
                for _, a := range args {
                        enc.Write([]byte(a.Strval()))
                }
                enc.Close()
                res = &String{buf.String()}
        }
        return
}

func builtinDecodeBase64(context *Scope, args... Value) (res Value, err error) {
        if len(args) > 0 {
                var list []Value
                for _, a := range args {
                        var dat []byte
                        dat, err = base64.StdEncoding.DecodeString(a.Strval())
                        if err == nil {
                                list = append(list, &String{string(dat)})
                        } else {
                                return
                        }
                }
                if x := len(list); x == 0 {
                        res = UniversalNone
                } else if x == 1 {
                        res = list[0]
                } else {
                        res = &List{Elements{list}}
                }
        }
        return
}

func builtinBase(context *Scope, args... Value) (Value, error) {
        var (
                l []Value
                s string
        )
        for _, a := range args {
                s = filepath.Base(a.Strval())
                l = append(l, &String{s})
        }
        if len(l) == 1 {
                return l[0], nil
        } else {
                return &List{Elements{l}}, nil
        }
}

func builtinDirDir(context *Scope, args... Value) (Value, error) {
        var (
                l []Value
                s string
        )
        for _, a := range args {
                s = filepath.Dir(filepath.Dir(a.Strval()))
                l = append(l, &String{s})
        }
        if len(l) == 1 {
                return l[0], nil
        } else {
                return &List{Elements{l}}, nil
        }
}

func builtinDir(context *Scope, args... Value) (Value, error) {
        var (
                l []Value
                s string
        )
        for _, a := range args {
                s = filepath.Dir(a.Strval())
                l = append(l, &String{s})
        }
        if len(l) == 1 {
                return l[0], nil
        } else {
                return &List{Elements{l}}, nil
        }
}

func builtinNDir(context *Scope, args... Value) (Value, error) {
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
                s = filepath.Dir(a.Strval())
                for i := n-1; 0 < i; i -= 1 {
                        s = filepath.Dir(s)
                }
                l = append(l, &String{s})
        }
        if len(l) == 1 {
                return l[0], nil
        } else {
                return &List{Elements{l}}, nil
        }
}

func builtinReadFile(context *Scope, args... Value) (res Value, err error) {
        var l []Value
        for _, a := range args {
                var s []byte
                if s, err = ioutil.ReadFile(a.Strval()); err == nil {
                        l = append(l, &String{string(s)})
                } else {
                        l = append(l, UniversalNone)
                }
        }
        if x := len(l); x == 0 {
                res = UniversalNone
        } else if x == 1 {
                res = l[0]
        } else {
                res = &List{Elements{l}}
        }
        return
}
