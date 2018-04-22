//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package types

import (
        "github.com/duzy/smart/token"
        "encoding/base64"
        "path/filepath"
        "io/ioutil"
        "strings"
        "strconv"
	"unicode"
        "errors"
        "bytes"
        "fmt"
        "os"
)

type BuiltinFunc func(pos token.Position, context *Scope, args... Value) (Value, error)

var builtins = map[string]BuiltinFunc {
        `typeof`: builtinTypeOf,

        `error`:  builtinError,

        `assert-valid`: builtinAssertValid,

        `or`:    builtinLogicalOr,
        /* TODO:
        `and`:   builtinLogicalAnd,
        `xor`:   builtinLogicalXor,
        `not`:   builtinLogicalNot, */

        `if`:    builtinBranchIf,
        `ifeq`:  builtinBranchIfEq,
        `ifne`:  builtinBranchIfNE,

        `env`:     builtinEnv,
        
        `print`:   builtinPrint,
        `printl`:  builtinPrintl,
        `println`: builtinPrintln,

        `plus`:    builtinPlus,
        `minus`:   builtinMinus,

        `joint-quote`: builtinJointQuote,
        `join`:    builtinJoin,
        `field`:   builtinField,
        `fields`:  builtinFields,

        `string`:  builtinString,
        `strip`:   builtinStrip,
        `title`:       builtinTitle,
        `trim`:        builtinTrim,
        `trim-space`:  builtinTrimSpace,
        `trim-left`:   builtinTrimLeft,
        `trim-right`:  builtinTrimRight,
        `trim-prefix`: builtinTrimPrefix,
        `trim-suffix`: builtinTrimSuffix,
        `trim-ext`:    builtinTrimExt,

        `indent`:      builtinIndent,

        // https://www.gnu.org/software/make/manual/html_node/Text-Functions.html
        `subst`:      builtinSubst,
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

        // TODO: move these into builtin package `path', `filepath'
        `base`:       builtinBase,
        `dir`:        builtinDir,
        `dirs`:       builtinDirs, // do `dir` n times
        `dirdir`:     builtinDirDir,

        `relative-dir`: builtinRelativeDir,

        // TODO: move these into builtin package `os'
        `mkdir`:      builtinMkdir,     // os/file.go
        `mkdir-all`:  builtinMkdirAll,  // os/path.go
        `chdir`:      builtinChdir,     // os/file.go
        `rename`:     builtinRename,    // os/file.go
        `remove`:     builtinRemove,    // os/file_*.go
        `remove-all`: builtinRemoveAll, // os/path.go
        `truncate`:   builtinTruncate,  // os/file_*.go
        `link`:       builtinLink,      // os/file_*.go
        `symlink`:    builtinSymlink,   // os/file_*.go

        // TODO: move these into builtin package 'io/ioutil'
        `read-dir`:   builtinReadDir,   // io/ioutil/ioutil.go
        `read-file`:  builtinReadFile,  // io/ioutil/ioutil.go
        `write-file`: builtinWriteFile, // io/ioutil/ioutil.go

        `return`:     builtinReturn,
}

func GetBuiltinNames() (a []string) {
        for s, _ := range builtins {
                a = append(a, s)
        }
        return
}

func EscapedString(v Value) (s string, e error) {
        if v.Type() == StringType {
                var sv string
                if sv, e = v.Strval(); e == nil {
                        s = strings.Replace(sv, "\\'", "'", -1)
                }
        } else {
                s, e = v.Strval()
        }
        return
}

func builtinTypeOf(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var ( elems []Value; s string )
        for _, arg := range args {
                // Arguments are passed in a list:
                //   $(fun abc)                 args: (abc)
                //   $(fun a,b,c)               args: (a),(b),(c)
                //   $(fun a b c,1 2 3)         args: (a b c),(1 2 3)
                switch a := arg.(type) {
                case *List:
                        if n := len(a.Elems); n == 1 {
                                switch v := a.Elems[0].(type) {
                                case *delegate: // FIXME: recursively undelegate types
                                        if d, _ := v.o.(*Def); d != nil {
                                                s = d.Value.Type().String()
                                        } else {
                                                s = "unknown"
                                        }
                                default:
                                        s = v.Type().String()
                                }
                        } else if n > 1 {
                                s = ListType.name
                        } else {
                                s = NoneType.name
                        }
                default:
                        // FIXME: this should be an exception (panic).
                        s = a.Type().String()
                }
                elems = append(elems, &String{s})
        }
        return MakeListOrScalar(elems), nil
}

func builtinError(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                s bytes.Buffer
                v string
        )
        for i, a := range args {
                if i > 0 { fmt.Fprintf(&s, " ") }
                if v, err = a.Strval(); err == nil {
                        fmt.Fprintf(&s, "%s", v)
                } else {
                        fmt.Fprintf(os.Stderr, "%s: %v\n", pos, err)
                        return
                }
        }
        fmt.Fprintf(os.Stderr, "%s: %v\n", pos, s.String())
        err = fmt.Errorf("%v", s.String())
        return
}

func builtinAssertValid(pos token.Position, context *Scope, args... Value) (Value, error) {
        for _, a := range args {
                if s, e := a.Strval(); e != nil {
                        return nil, e
                } else if s == "" {
                        return nil, fmt.Errorf("invalid value")
                }
        }
        return nil, nil
}

func builtinLogicalOr(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        for _, a := range args {
                if a, err = Reveal(a); err != nil { return }
                if a == nil { continue } // discard nil
                var s string
                if s, err = a.Strval(); err != nil { return }
                if strings.TrimSpace(s) != "" { 
                        res = a; break
                }
        }
        return
}

func builtinBranchIf(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        if n := len(args); n > 1 {
                var (
                        cond Value
                        s string
                )
                if cond, err = Reveal(args[0]); err != nil { return }
                if s, err = cond.Strval(); err != nil { return }
                if strings.TrimSpace(s) != "" { 
                        res = args[1]
                } else if n > 1 {
                        res = MakeListOrScalar(args[2:])
                }
        }
        return
}

func builtinBranchIfEq(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        if n := len(args); n > 2 {
                var (
                        a, b Value
                        s1, s2 string
                )
                if a, err = Reveal(args[0]); err != nil { return }
                if b, err = Reveal(args[1]); err != nil { return }
                if s1, err = a.Strval(); err != nil { return }
                if s2, err = b.Strval(); err != nil { return }
                if s1 == s2 { 
                        res = args[2]
                } else if n > 3 {
                        res = MakeListOrScalar(args[3:])
                }
        }
        return
}

func builtinBranchIfNE(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        if n := len(args); n > 2 {
                var (
                        a, b Value
                        s1, s2 string
                )
                if a, err = Reveal(args[0]); err != nil { return }
                if b, err = Reveal(args[1]); err != nil { return }
                if s1, err = a.Strval(); err != nil { return }
                if s2, err = b.Strval(); err != nil { return }
                if s1 != s2 { 
                        res = args[2]
                } else if n > 3 {
                        res = MakeListOrScalar(args[3:])
                }
        }
        return
}

func builtinEnv(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                vals []Value
                val Value
                v string
        )
        for _, a := range args {
                if val, err = Reveal(a); err != nil { return }
                if val == nil {
                        // discard
                } else if v, err = val.Strval(); err == nil {
                        if s := strings.TrimSpace(v); s != "" {
                                vals = append(vals, &String{os.Getenv(s)})
                        }
                } else {
                        return
                }
        }
        return MakeListOrScalar(vals), nil
}

func builtinPrint(pos token.Position, context *Scope, args... Value) (Value, error) {
        var x = len(args)
        for i, a := range args {
                if 0 < i && i < x {
                        fmt.Printf(" ")
                }
                if s, e := EscapedString(a); e == nil {
                        fmt.Printf("%s", s)
                } else {
                        fmt.Fprintf(os.Stderr, "%s: %s", pos, e)
                }
        }
        return nil, nil
}

func builtinPrintl(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                x = len(args)
                s string
        )
        for i, a := range args {
                if 0 < i && i < x {
                        fmt.Printf(" ")
                }
                if s, err = EscapedString(a); err != nil {
                        return
                }
                fmt.Printf("%s", s)
                if i == x && !strings.HasSuffix(s, "\n") {
                        fmt.Printf("\n")
                }
        }
        return nil, nil
}

func builtinPrintln(pos token.Position, context *Scope, args... Value) (Value, error) {
        builtinPrint(pos, context, args...)
        fmt.Printf("\n")
        return nil, nil
}

func builtinPlus(pos token.Position, context *Scope, args... Value) (result Value, err error) {
        var num, v int64
        for _, a := range args {
                if v, err = a.Integer(); err != nil {
                        return
                }
                num += v
        } 
        return &Int{integer{num}}, nil
}

func builtinMinus(pos token.Position, context *Scope, args... Value) (result Value, err error) {
        var num, v int64
        for i, a := range args {
                if v, err = a.Integer(); err != nil {
                        return
                }
                if i == 0 {
                        num = v
                } else {
                        num -= v
                }
        }
        return &Int{integer{num}}, nil
}

func builtinJoin(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        if args, err = JoinEval(ClosureContext{context}, args...); err != nil { return }
        if l := len(args); l >= 2 {
                var (
                        fields []string
                        v string
                )
                for _, a := range args[:l-1] {
                        if v, err = a.Strval(); err != nil { return }
                        if v != "" { fields = append(fields, v) }
                }
                if v, err = args[l-1].Strval(); err != nil { return }
                res = &String{strings.Join(fields, v)}
        }
        return
}

func builtinJointQuote(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        if args, err = JoinEval(ClosureContext{context}, args...); err != nil { return }
        if l := len(args); l >= 3 {
                var (
                        fields []string
                        v string
                )
                for _, a := range args[:l-1] {
                        if v, err = a.Strval(); err != nil { return }
                        if v != "" { fields = append(fields, strconv.Quote(v)) }
                }
                if v, err = args[l-1].Strval(); err != nil { return }
                if v == "" { v = " " }
                res = &String{strings.Join(fields, v)}
        }
        return
}

func builtinField(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        if l := len(args); l >= 2 {
                var (
                        i int64
                        s string
                        fields []string
                )
                if i, err = args[0].Integer(); err != nil { return }
                if s, err = args[1].Strval(); err != nil { return }
                if l > 2 {
                        var v string
                        if v, err = args[2].Strval(); err != nil { return }
                        fields = strings.Split(s, v)
                } else {
                        fields = strings.Fields(s)
                }
                if n := int(i)-1; 0 <= n && n < len(fields) {
                        s = strings.TrimSpace(fields[n])
                        res = &String{s}
                }
        }
        return
}

func builtinFields(pos token.Position, context *Scope, args... Value) (Value, error) {
        // TODO: ...
        return nil, nil
}

func builtinString(pos token.Position, context *Scope, args... Value) (result Value, err error) {
        var (
                s bytes.Buffer
                v string
        )
        for i, a := range args {
                if i > 0 { s.WriteString(" ") }
                if v, err = a.Strval(); err != nil {
                        return
                }
                s.WriteString(v)
        }
        result = &String{s.String()}
        return
}

func builtinFilterValues(pos token.Position, context *Scope, neg bool, args... Value) (res Value, err error) {
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
                                case *GlobPattern:
                                        var (
                                                str, s string
                                                m bool
                                        )
                                        if str, err = v.Strval(); err == nil {
                                                if m, s, err = p.Match(str); err == nil && m && s != "" {
                                                        //fmt.Printf("match: %v: %v (%v)\n", m, s, v)
                                                        return true
                                                }
                                        }
                                        if err != nil { break }
                                default:
                                        fmt.Printf("todo: %v (%T) (%v)\n", pat, pat, v)
                                }
                        }
                        return false
                }
                if len(pats) > 0 {
                        var elems, a []Value
                        if a, err = JoinReveal(args[1:]...); err != nil { return }
                        for _, v := range a {
                                var okay = f(v)
                                if err != nil { return }
                                if neg { okay = !okay }
                                if okay { elems = append(elems, v) }
                        }
                        res = MakeListOrScalar(elems)
                }
        }
        if res == nil {
                res = UniversalNone
        }
        return
}

// $(subst from,to,text)
func builtinSubst(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var list []Value
        if nargs := len(args); nargs > 2 {
                var s, s1, s2 string
                if s1, err = args[0].Strval(); err != nil { return }
                if s2, err = args[1].Strval(); err != nil { return }
                var a []Value
                if a, err = JoinReveal(args[2:]...); err != nil { return }
                for _, arg := range a {
                        if s, err = arg.Strval(); err != nil { return }
                        list = append(list, &String{ strings.Replace(s, s1, s2, -1) })
                }
        }
        res = MakeListOrScalar(list)
        return
}

// TODO:
//   $(var:pattern=replacement)
//   $(var:suffix=replacement)
func builtinPatsubst(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        // $(patsubst pattern,replacement,text)
        var list []Value
        if nargs := len(args); nargs > 2 {
                var a []Value
                if a, err = JoinReveal(args[2:]...); err != nil { return }
                for _, arg := range a {
                        var (
                                a, s string // stemp
                                m bool // matched
                        )
                        if pat, _ := args[0].(*GlobPattern); pat != nil {
                                if a, err = arg.Strval(); err != nil {
                                        return
                                }
                                if m, s, err = pat.Match(a); err != nil {
                                        return
                                }
                        } else if l, _ := args[0].(*List); l != nil {
                                for _, elem := range l.Elems {
                                        if pat, _ := elem.(*GlobPattern); pat != nil {
                                                if a, err = arg.Strval(); err != nil {
                                                        return
                                                }
                                                if m, s, err = pat.Match(a); err != nil {
                                                        return
                                                } else if m && s != "" {
                                                        break
                                                }
                                        }
                                }
                        }

                        if m && s != "" {
                                var s1, s2 string
                                if rep, ok := args[1].(*GlobPattern); ok {
                                        if s1, err = rep.Prefix.Strval(); err != nil {
                                                return
                                        }
                                        if s2, err = rep.Suffix.Strval(); err != nil {
                                                return
                                        }
                                        s = s1 + s + s2
                                } else if l, _ := args[1].(*List); l != nil {
                                        var str = s
                                        for _, elem := range l.Elems {
                                                if rep, _ := elem.(*GlobPattern); rep != nil {
                                                        if s1, err = rep.Prefix.Strval(); err != nil {
                                                                return
                                                        }
                                                        if s2, err = rep.Suffix.Strval(); err != nil {
                                                                return
                                                        }
                                                        str = s1 + str + s2
                                                        break
                                                }
                                        }
                                        s = str
                                } else if s, err = args[1].Strval(); err != nil {
                                        return
                                }
                                
                                switch t := arg.(type) {
                                case *Barefile:
                                        /*ext := filepath.Ext(s)
                                        if len(ext) > 0 {
                                                s = s[:len(s)-len(ext)]
                                                ext = ext[1:]
                                        }
                                        list = append(list, &Barefile{ &Bareword{s} })*/
                                        list = append(list, t)
                                default:
                                        list = append(list, &String{s})
                                }
                        } else if arg.Type().Kind() != NoneKind {
                                list = append(list, arg)
                        }
                }
        }
        res = MakeListOrScalar(list)
        return
}

func builtinStrip(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        return builtinTrimSpace(pos, context, args...)
}

func builtinTrimSpace(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        return builtinTrim(pos, context, append([]Value{ UniversalNone }, args...)...)
}

func builtinTitle(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                list []Value
                s string
        )
        for _, a := range args {
                if a, err = Reveal(a); err != nil { return }
                if a == nil {
                        // discard
                } else if s, err = a.Strval(); err != nil {
                        return
                } else if s != "" {
                        list = append(list, strval(strings.Title(s)))
                }
        }
        if err == nil {
                res = MakeListOrScalar(list)
        }
        return
}

func builtinTrim(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                list []Value
                cutset, s string
        )
        for i, a := range args {
                if a, err = Reveal(a); err != nil { return }
                if a == nil {
                        // discard
                } else if s, err = a.Strval(); err != nil {
                        return
                } else if s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, strval(strings.TrimSpace(s)))
                        } else {
                                list = append(list, strval(strings.Trim(s, cutset)))
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(list)
        }
        return
}

func builtinTrimLeft(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                list []Value
                cutset, s string
        )
        for i, a := range args {
                if a, err = Reveal(a); err != nil { return }
                if a == nil {
                        // discard
                } else if s, err = a.Strval(); err != nil {
                        return
                } else if s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, strval(strings.TrimLeftFunc(s, unicode.IsSpace)))
                        } else {
                                list = append(list, strval(strings.TrimLeft(s, cutset)))
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(list)
        }
        return
}

func builtinTrimRight(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                list []Value
                cutset, s string
        )
        for i, a := range args {
                if a, err = Reveal(a); err != nil { return }
                if a == nil {
                        // discard
                } else if s, err = a.Strval(); err != nil {
                        return
                } else if s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, strval(strings.TrimRightFunc(s, unicode.IsSpace)))
                        } else {
                                list = append(list, strval(strings.TrimRight(s, cutset)))
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(list)
        }
        return
}

func builtinTrimPrefix(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                list []Value
                cutset, s string
        )
        for i, a := range args {
                if a, err = Reveal(a); err != nil { return }
                if a == nil {
                        // discard
                } else if s, err = a.Strval(); err != nil {
                        return
                } else if s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, strval(strings.TrimLeftFunc(s, unicode.IsSpace)))
                        } else {
                                list = append(list, strval(strings.TrimPrefix(s, cutset)))
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(list)
        }
        return
}

func builtinTrimSuffix(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                list []Value
                cutset, s string
        )
        for i, a := range args {
                if a, err = Reveal(a); err != nil { return }
                if a == nil {
                        // discard
                } else if s, err = a.Strval(); err != nil {
                        return
                } else if s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, strval(strings.TrimRightFunc(s, unicode.IsSpace)))
                        } else {
                                list = append(list, strval(strings.TrimSuffix(s, cutset)))
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(list)
        }
        return
}

func builtinTrimExt(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                list []Value
                ext, s string
        )
        for i, a := range args {
                if a, err = Reveal(a); err != nil { return }
                if a == nil {
                        // discard
                } else if s, err = a.Strval(); err != nil {
                        return
                } else if s != "" {
                        if i == 0 && len(args) > 1 {
                                ext = s
                        } else if ext == "" {
                                list = append(list, strval(strings.TrimSuffix(s, filepath.Ext(s))))
                        } else if ext == filepath.Ext(s) {
                                list = append(list, strval(strings.TrimRight(s, ext)))
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(list)
        }
        return
}

func builtinIndent(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                l []Value
                s string // indent
        )
        if x := len(args); x > 0 {
                if v := Scalar(args[0], IntType); v != nil {
                        var i int64
                        if i, err = v.Integer(); err != nil {
                                return
                        }
                        args, s = args[1:], strings.Repeat(" ", int(i))
                } else {
                        return nil, errors.New("requires integer argument (first|last)")
                }
        }
        for _, a := range args {
                var (
                        lines []string
                        v string
                )
                if v, err = a.Strval(); err != nil {
                        return
                }
                for _, line := range strings.Split(v, "\n") {
                        lines = append(lines, s + line)
                }
                l = append(l, &String{strings.Join(lines, "\n")})
        }
        res = MakeListOrScalar(l)
        return
}

func builtinFindstring(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        // $(findstring find,text)
        return
}

func builtinFilter(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        // $(filter pattern…,text)
        res, err = builtinFilterValues(pos, context, false, args...)
        return
}

func builtinFilterOut(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        // $(filter-out pattern…,text)
        res, err = builtinFilterValues(pos, context, true, args...)
        return
}

func builtinSort(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        // $(sort list)
        return
}

func builtinWord(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        // $(word n,text)
        return
}

func builtinWordList(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        // $(wordlist s,e,text)
        return
}

func builtinWords(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        // $(words n,text)
        return
}

func builtinFirstWord(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        // $(firstword names...)
        return
}

func builtinLastWord(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        // $(lastword names...)
        return
}

func builtinEncodeBase64(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        if len(args) > 0 {
                buf := new(bytes.Buffer)
                enc := base64.NewEncoder(base64.StdEncoding, buf)
                for _, a := range args {
                        var s string
                        if s, err = a.Strval(); err != nil {
                                return
                        }
                        enc.Write([]byte(s))
                }
                enc.Close()
                res = &String{buf.String()}
        }
        return
}

func builtinDecodeBase64(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        if len(args) > 0 {
                var list []Value
                for _, a := range args {
                        var (
                                dat []byte
                                s string
                        )
                        if s, err = a.Strval(); err != nil {
                                return
                        }
                        dat, err = base64.StdEncoding.DecodeString(s)
                        if err == nil {
                                list = append(list, &String{string(dat)})
                        } else {
                                return
                        }
                }
                res = MakeListOrScalar(list)
        }
        return
}

func builtinBase(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                l []Value
                s string
        )
        for _, a := range args {
                if s, err = a.Strval(); err != nil {
                        return
                }
                s = filepath.Base(s)
                l = append(l, &String{s})
        }
        res = MakeListOrScalar(l)
        return
}

func builtinDirDir(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                l []Value
                s string
        )
        for _, a := range args {
                if s, err = a.Strval(); err != nil {
                        return
                }
                s = filepath.Dir(filepath.Dir(s))
                l = append(l, &String{s})
        }
        res = MakeListOrScalar(l)
        return
}

func builtinDir(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                l []Value
                s string
        )
        for _, a := range args {
                if s, err = a.Strval(); err != nil {
                        return
                }
                s = filepath.Dir(s)
                l = append(l, &String{s})
        }
        res = MakeListOrScalar(l)
        return
}

func builtinDirs(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                l []Value
                s string
                i int64
                n = 0
        )
        if x := len(args); x > 0 {
                if v := Scalar(args[0], IntType); v != nil {
                        if i, err = v.Integer(); err != nil {
                                return
                        }
                        args, n = args[1:], int(i)
                } else if v := Scalar(args[x-1], IntType); v != nil {
                        if i, err = v.Integer(); err != nil {
                                return
                        }
                        args, n = args[:x-1], int(i)
                } else {
                        return nil, errors.New("Require (first/last) integer argument")
                }
        }
        for _, a := range args {
                if s, err = a.Strval(); err != nil {
                        return
                }
                s = filepath.Dir(s)
                for i := n-1; 0 < i; i -= 1 {
                        s = filepath.Dir(s)
                }
                l = append(l, &String{s})
        }
        res = MakeListOrScalar(l)
        return
}

func builtinRelativeDir(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var (
                l []Value
                t, s string
        )
        for i, a := range args {
                if s, err = a.Strval(); err != nil {
                        return
                }
                if i == 0 {
                        t = s
                } else if s, err = filepath.Rel(t, s); err == nil {
                        l = append(l, &String{s})
                } else {
                        return
                }
        }
        res = MakeListOrScalar(l)
        return
}

func builtinMkdir(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name string
                        perm os.FileMode
                        num int64
                )
                switch t := a.(type) {
                case *Pair: // mkdir name => perm name => perm
                        if name, err = t.Key.Strval(); err != nil { return }
                        if num, err = t.Value.Integer(); err != nil { return }
                        perm = os.FileMode(num & 0777)
                case *Group: // mkdir (name perm) (name perm)
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { return }
                                if num, err = t.Get(1).Integer(); err != nil { return }
                                perm = os.FileMode(num & 0777)
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // mkdir name perm, name perm, ...
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { return }
                                if num, err = t.Get(1).Integer(); err != nil { return }
                                perm = os.FileMode(num & 0777)
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // mkdir name perm, name perm, ...
                        if name, err = args[i].Strval(); err != nil { return }
                        if i+1 < nargs {
                                if num, err = args[i+1].Integer(); err != nil { return }
                                perm = os.FileMode(num & 0777)
                                i += 1
                        }
                }
                if err = os.Mkdir(name, perm); err != nil {
                        break
                }
        }
        return
}

func builtinMkdirAll(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name string
                        perm os.FileMode
                        num int64
                )
                switch t := a.(type) {
                case *Pair: // mkdir name => perm name => perm
                        if name, err = t.Key.Strval(); err != nil { return }
                        if num, err = t.Value.Integer(); err != nil { return }
                        perm = os.FileMode(num & 0777)
                case *Group: // mkdir (name perm) (name perm)
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { return }
                                if num, err = t.Get(1).Integer(); err != nil { return }
                                perm = os.FileMode(num & 0777)
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // mkdir name perm, name perm, ...
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { return }
                                if num, err = t.Get(1).Integer(); err != nil { return }
                                perm = os.FileMode(num & 0777)
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // mkdir name perm, name perm, ...
                        if name, err = args[i].Strval(); err != nil { return }
                        if i+1 < nargs {
                                if num, err = args[i+1].Integer(); err != nil { return }
                                perm = os.FileMode(num & 0777)
                                i += 1
                        }
                }
                if err = os.MkdirAll(name, perm); err != nil {
                        break
                }
        }
        return
}

func builtinChdir(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        if len(args) == 1 {
                var str string
                if str, err = args[0].Strval(); err != nil { return }
                err = os.Chdir(str)
        } else {
                err = errors.New("Wrong number of arguments.")
        }
        return
}

func builtinRename(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        oldname, newname string
                )
                switch t := a.(type) {
                case *Pair: // rename oldname => newname old => new
                        if oldname, err = t.Key.Strval(); err != nil { return }
                        if newname, err = t.Value.Strval(); err != nil { return }
                case *Group: // rename (oldname newname) (old new)
                        if t.Len() == 2 {
                                if oldname, err = t.Get(0).Strval(); err != nil { return }
                                if newname, err = t.Get(1).Strval(); err != nil { return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // rename oldname newname, old new, ...
                        if t.Len() == 2 {
                                if oldname, err = t.Get(0).Strval(); err != nil { return }
                                if newname, err = t.Get(1).Strval(); err != nil { return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // rename newname oldname  newname oldname ...
                        if i+1 < nargs {
                                if oldname, err = args[i+0].Strval(); err != nil { return }
                                if newname, err = args[i+1].Strval(); err != nil { return }
                                i += 1
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong arguments `%v'", args))
                                break
                        }
                }
                if err = os.Rename(oldname, newname); err != nil {
                        break
                }
        }
        return
}

func builtinRemove(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        if args, err = JoinEval(ClosureContext{context}, args...); err != nil {
                return
        }
        var (
                names []string
                str string
        )
        ArgsLoop: for _, a := range args {
                if str, err = a.Strval(); err != nil {
                        return
                }
                if names, err = filepath.Glob(str); err != nil {
                        fmt.Fprintf(os.Stderr, "error: remove: %s\n", err)
                        break
                } else {
                        for _, s := range names {
                                //fmt.Printf("remove %s\n", s)
                                if err = os.Remove(s); err != nil {
                                        break ArgsLoop
                                }
                        }
                }
        }
        return
}

func builtinRemoveAll(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        if args, err = JoinEval(ClosureContext{context}, args...); err != nil {
                return
        }
        /*for _, a := range args {
                if err = os.RemoveAll(a.Strval()); err != nil {
                        break
                }
        }*/
        var (
                names []string
                str string
        )
        ArgsLoop: for _, a := range args {
                if str, err = a.Strval(); err != nil {
                        return
                }
                if names, err = filepath.Glob(str); err != nil {
                        fmt.Fprintf(os.Stderr, "error: remove-all: %s\n", err)
                        break
                } else {
                        for _, s := range names {
                                //fmt.Printf("remove-all %s\n", s)
                                if err = os.RemoveAll(s); err != nil {
                                        break ArgsLoop
                                }
                        }
                }
        }
        return
}

func builtinTruncate(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name string
                        size int64
                )
                switch t := a.(type) {
                case *Pair: // truncate name => size old => new
                        if name, err = t.Key.Strval(); err != nil { return }
                        if size, err = t.Value.Integer(); err != nil { return }
                case *Group: // truncate (name size) (old new)
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { return }
                                if size, err = t.Get(1).Integer(); err != nil { return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // truncate name size, old new, ...
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { return }
                                if size, err = t.Get(1).Integer(); err != nil { return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // truncate name size  name size ...
                        if i+1 < nargs {
                                if name, err = args[i+0].Strval(); err != nil { return }
                                if size, err = args[i+1].Integer(); err != nil { return }
                                i += 1
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong arguments `%v'", args))
                                break
                        }
                }
                if err = os.Truncate(name, size); err != nil {
                        break
                }
        }
        return
}

func builtinLink(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        oldname, newname string
                )
                switch t := a.(type) {
                case *Pair: // link oldname => newname old => new
                        if oldname, err = t.Key.Strval(); err != nil { return }
                        if newname, err = t.Value.Strval(); err != nil { return }
                case *Group: // link (oldname newname) (old new)
                        if t.Len() == 2 {
                                if oldname, err = t.Get(0).Strval(); err != nil { return }
                                if newname, err = t.Get(1).Strval(); err != nil { return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // link oldname newname, old new, ...
                        if t.Len() == 2 {
                                if oldname, err = t.Get(0).Strval(); err != nil { return }
                                if newname, err = t.Get(1).Strval(); err != nil { return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // link oldname newname  oldname newname ...
                        if i+1 < nargs {
                                if oldname, err = args[i+0].Strval(); err != nil { return }
                                if newname, err = args[i+1].Strval(); err != nil { return }
                                i += 1
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong arguments `%v'", args))
                                break
                        }
                }
                if err = os.Link(oldname, newname); err != nil {
                        break
                }
        }
        return
}

func builtinSymlink(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        oldname, newname string
                )
                switch t := a.(type) {
                case *Pair: // symlink oldname => newname old => new
                        if oldname, err = t.Key.Strval(); err != nil { return }
                        if newname, err = t.Value.Strval(); err != nil { return }
                case *Group: // symlink (oldname newname) (old new)
                        if t.Len() == 2 {
                                if oldname, err = t.Get(0).Strval(); err != nil { return }
                                if newname, err = t.Get(1).Strval(); err != nil { return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // symlink oldname newname, old new, ...
                        if t.Len() == 2 {
                                if oldname, err = t.Get(0).Strval(); err != nil { return }
                                if newname, err = t.Get(1).Strval(); err != nil { return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // symlink newname oldname  newname oldname ...
                        if i+1 < nargs {
                                if oldname, err = args[i+0].Strval(); err != nil { return }
                                if newname, err = args[i+1].Strval(); err != nil { return }
                                i += 1
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong arguments `%v'", args))
                                break
                        }
                }
                if err = os.Symlink(oldname, newname); err != nil {
                        break
                }
        }
        return
}

func builtinReadDir(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var l []Value
        for _, a := range args {
                var (
                        fis []os.FileInfo
                        str string
                )
                if str, err = a.Strval(); err != nil { return }
                if fis, err = ioutil.ReadDir(str); err == nil {
                        v := new(List)
                        for _, fi := range fis {
                                v.Append(&String{fi.Name()})
                        }
                        l = append(l, v)
                } else {
                        break //l = append(l, UniversalNone)
                }
        }
        if err == nil {
                res = MakeListOrScalar(l)
        }
        return
}

func builtinReadFile(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var l []Value
        for _, a := range args {
                var (
                        s []byte
                        str string
                )
                if str, err = a.Strval(); err != nil { return }
                if s, err = ioutil.ReadFile(str); err == nil {
                        l = append(l, &String{string(s)})
                } else {
                        break //l = append(l, UniversalNone)
                }
        }
        if err == nil {
                res = MakeListOrScalar(l)
        }
        return
}

func builtinWriteFile(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name, data string
                        perm = os.FileMode(0600)
                        num int64
                )
                switch t := a.(type) {
                case *Pair: // write-file name => text name => text
                        if name, err = t.Key.Strval(); err != nil { return }
                        if data, err = t.Value.Strval(); err != nil { return }
                case *Group: // write-file (name text) (name text 0660)
                        if n := t.Len(); n < 4 && n > 0 {
                                if name, err = t.Get(0).Strval(); err != nil { return }
                                if n > 1 {
                                        if data, err = t.Get(1).Strval(); err != nil { return }
                                }
                                if n > 2 {
                                        if num, err = t.Get(2).Integer(); err != nil { return }
                                        perm = os.FileMode(num & 0777)
                                }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // write-file name text, name text 0660, ...
                        if n := t.Len(); n < 4 && n > 0 {
                                if name, err = t.Get(0).Strval(); err != nil { return }
                                if n > 1 {
                                        if data, err = t.Get(1).Strval(); err != nil { return }
                                }
                                if n > 2 {
                                        if num, err = t.Get(2).Integer(); err != nil { return }
                                        perm = os.FileMode(num & 0777)
                                }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // write-file name text 0660  name text 0660 ...
                        if name, err = args[i].Strval(); err != nil { return }
                        if i+1 < nargs {
                                if data, err = args[i+1].Strval(); err != nil { return }
                                i += 1
                        }
                        if i+1 < nargs {
                                if num, err = args[i+1].Integer(); err != nil { return }
                                perm = os.FileMode(num & 0777)
                                i += 1
                        }
                }
                if err = ioutil.WriteFile(name, []byte(data), perm); err != nil {
                        break
                }
        }
        return
}

func builtinReturn(pos token.Position, context *Scope, args... Value) (res Value, err error) {
        var value Value
        if x := len(args); x == 0 {
                value = args[x]
        } else {
                value = &List{Elements{args}}
        }
        return nil, &Returner{ value }
}
