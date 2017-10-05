//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package types

import (
        //"github.com/duzy/smart/token"
        "encoding/base64"
        "path/filepath"
        "io/ioutil"
        "strings"
	"unicode"
        "errors"
        "bytes"
        "fmt"
        "os"
)

type BuiltinFunc func(context *Scope, args... Value) (Value, error)

var builtins = map[string]BuiltinFunc {
        `or`:    builtinLogicalOr,
        /* TODO:
        `and`:   builtinLogicalAnd,
        `xor`:   builtinLogicalXor,
        `not`:   builtinLogicalNot,

        `if`:    builtinBranchIf, */
        
        `print`:   builtinPrint,
        `printl`:  builtinPrintl,
        `println`: builtinPrintln,

        `plus`:    builtinPlus,
        `minus`:   builtinMinus,

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

        // TODO: move these into builtin package `path', `filepath'
        `base`:       builtinBase,
        `dir`:        builtinDir,
        `dirdir`:     builtinDirDir,
        `ndir`:       builtinNDir,

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
}

func EscapedString(v Value) (s string) {
        if v.Type() == StringType {
                s = strings.Replace(v.Strval(), "\\'", "'", -1)
        } else {
                s = v.Strval()
        }
        return
}

func builtinLogicalOr(context *Scope, args... Value) (Value, error) {
        for _, a := range args {
                if val, err := Disclose(context, Reveal(a)); err != nil {
                        return nil, err
                } else if val != nil && val.String() != "" {
                        return val, nil
                }
        }
        return nil, nil
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

func builtinPlus(context *Scope, args... Value) (result Value, err error) {
        var num int64
        for _, a := range args {
                num += a.Integer()
        } 
        return &Int{integer{num}}, nil
}

func builtinMinus(context *Scope, args... Value) (result Value, err error) {
        var num int64
        for i, a := range args {
                if i == 0 {
                        num = a.Integer()
                } else {
                        num -= a.Integer()
                }
        }
        return &Int{integer{num}}, nil
}

func builtinString(context *Scope, args... Value) (result Value, err error) {
        var s bytes.Buffer
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
                        for _, v := range JoinReveal(args[1:]...) {
                                var okay = f(v)
                                if neg { okay = !okay }
                                if okay { elems = append(elems, v) }
                        }
                        res = toListOrValue(elems)
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
                for _, arg := range JoinReveal(args[2:]...) {
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
                                list = append(list, arg)
                        }
                }
        }
        res = toListOrValue(list)
        return
}

func builtinStrip(context *Scope, args... Value) (res Value, err error) {
        return builtinTrimSpace(context, args...)
}

func builtinTrimSpace(context *Scope, args... Value) (res Value, err error) {
        return builtinTrim(context, append([]Value{ UniversalNone }, args...)...)
}

func builtinTitle(context *Scope, args... Value) (res Value, err error) {
        var list []Value
        for _, a := range args {
                if val, err := Disclose(context, Reveal(a)); err != nil {
                        return nil, err
                } else if val == nil {
                        // discard
                } else if s := val.String(); s != "" {
                        list = append(list, strval(strings.Title(s)))
                }
        }
        if err == nil {
                res = toListOrValue(list)
        }
        return
}

func builtinTrim(context *Scope, args... Value) (res Value, err error) {
        var (
                list []Value
                cutset string
        )
        for i, a := range args {
                if val, err := Disclose(context, Reveal(a)); err != nil {
                        return nil, err
                } else if val == nil {
                        // discard
                } else if s := val.String(); s != "" {
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
                res = toListOrValue(list)
        }
        return
}

func builtinTrimLeft(context *Scope, args... Value) (res Value, err error) {
        var (
                list []Value
                cutset string
        )
        for i, a := range args {
                if val, err := Disclose(context, Reveal(a)); err != nil {
                        return nil, err
                } else if val == nil {
                        // discard
                } else if s := val.String(); s != "" {
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
                res = toListOrValue(list)
        }
        return
}

func builtinTrimRight(context *Scope, args... Value) (res Value, err error) {
        var (
                list []Value
                cutset string
        )
        for i, a := range args {
                if val, err := Disclose(context, Reveal(a)); err != nil {
                        return nil, err
                } else if val == nil {
                        // discard
                } else if s := val.String(); s != "" {
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
                res = toListOrValue(list)
        }
        return
}

func builtinTrimPrefix(context *Scope, args... Value) (res Value, err error) {
        var (
                list []Value
                cutset string
        )
        for i, a := range args {
                if val, err := Disclose(context, Reveal(a)); err != nil {
                        return nil, err
                } else if val == nil {
                        // discard
                } else if s := val.String(); s != "" {
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
                res = toListOrValue(list)
        }
        return
}

func builtinTrimSuffix(context *Scope, args... Value) (res Value, err error) {
        var (
                list []Value
                cutset string
        )
        for i, a := range args {
                if val, err := Disclose(context, Reveal(a)); err != nil {
                        return nil, err
                } else if val == nil {
                        // discard
                } else if s := val.String(); s != "" {
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
                res = toListOrValue(list)
        }
        return
}

func builtinTrimExt(context *Scope, args... Value) (res Value, err error) {
        var (
                list []Value
                ext string
        )
        for i, a := range args {
                if val, err := Disclose(context, Reveal(a)); err != nil {
                        return nil, err
                } else if val == nil {
                        // discard
                } else if s := val.String(); s != "" {
                        if i == 0 && len(args) > 1 {
                                ext = s
                        } else if ext == "" {
                                list = append(list, strval(strings.TrimSuffix(s, filepath.Ext(s))))
                        } else if ext == filepath.Ext(s) {
                                list = append(list, strval(strings.TrimSuffix(s, ext)))
                        }
                }
        }
        if err == nil {
                res = toListOrValue(list)
        }
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
                res = toListOrValue(list)
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
        return toListOrValue(l), nil
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
        return toListOrValue(l), nil
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
        return toListOrValue(l), nil
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
        return toListOrValue(l), nil
}

func builtinMkdir(context *Scope, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name string
                        perm os.FileMode
                )
                switch t := a.(type) {
                case *Pair: // mkdir name => perm name => perm
                        name = t.Key.Strval()
                        perm = os.FileMode(t.Value.Integer() & 0777)
                case *Group: // mkdir (name perm) (name perm)
                        if t.Len() == 2 {
                                name = t.Get(0).Strval()
                                perm = os.FileMode(t.Get(1).Integer() & 0777)
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // mkdir name perm, name perm, ...
                        if t.Len() == 2 {
                                name = t.Get(0).Strval()
                                perm = os.FileMode(t.Get(1).Integer() & 0777)
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // mkdir name perm, name perm, ...
                        name = args[i].Strval()
                        if i+1 < nargs {
                                perm = os.FileMode(args[i+1].Integer() & 0777)
                                i += 1
                        }
                }
                if err = os.Mkdir(name, perm); err != nil {
                        break
                }
        }
        return
}

func builtinMkdirAll(context *Scope, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name string
                        perm os.FileMode
                )
                switch t := a.(type) {
                case *Pair: // mkdir name => perm name => perm
                        name = t.Key.Strval()
                        perm = os.FileMode(t.Value.Integer() & 0777)
                case *Group: // mkdir (name perm) (name perm)
                        if t.Len() == 2 {
                                name = t.Get(0).Strval()
                                perm = os.FileMode(t.Get(1).Integer() & 0777)
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // mkdir name perm, name perm, ...
                        if t.Len() == 2 {
                                name = t.Get(0).Strval()
                                perm = os.FileMode(t.Get(1).Integer() & 0777)
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // mkdir name perm, name perm, ...
                        name = args[i].Strval()
                        if i+1 < nargs {
                                perm = os.FileMode(args[i+1].Integer() & 0777)
                                i += 1
                        }
                }
                if err = os.MkdirAll(name, perm); err != nil {
                        break
                }
        }
        return
}

func builtinChdir(context *Scope, args... Value) (res Value, err error) {
        if len(args) == 1 {
                err = os.Chdir(args[0].Strval())
        } else {
                err = errors.New("Wrong number of arguments.")
        }
        return
}

func builtinRename(context *Scope, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        oldname, newname string
                )
                switch t := a.(type) {
                case *Pair: // rename oldname => newname old => new
                        oldname = t.Key.Strval()
                        newname = t.Value.Strval()
                case *Group: // rename (oldname newname) (old new)
                        if t.Len() == 2 {
                                oldname = t.Get(0).Strval()
                                newname = t.Get(1).Strval()
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // rename oldname newname, old new, ...
                        if t.Len() == 2 {
                                oldname = t.Get(0).Strval()
                                newname = t.Get(1).Strval()
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // rename newname oldname  newname oldname ...
                        if i+1 < nargs {
                                oldname = args[i+0].Strval()
                                newname = args[i+1].Strval()
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

func builtinRemove(context *Scope, args... Value) (res Value, err error) {
        if args, err = JoinEval(context, args...); err != nil {
                return
        }
        var names []string
        ArgsLoop: for _, a := range args {
                if names, err = filepath.Glob(a.Strval()); err != nil {
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

func builtinRemoveAll(context *Scope, args... Value) (res Value, err error) {
        if args, err = JoinEval(context, args...); err != nil {
                return
        }
        /*for _, a := range args {
                if err = os.RemoveAll(a.Strval()); err != nil {
                        break
                }
        }*/
        var names []string
        ArgsLoop: for _, a := range args {
                if names, err = filepath.Glob(a.Strval()); err != nil {
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

func builtinTruncate(context *Scope, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name string
                        size int64
                )
                switch t := a.(type) {
                case *Pair: // truncate name => size old => new
                        name = t.Key.Strval()
                        size = t.Value.Integer()
                case *Group: // truncate (name size) (old new)
                        if t.Len() == 2 {
                                name = t.Get(0).Strval()
                                size = t.Get(1).Integer()
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // truncate name size, old new, ...
                        if t.Len() == 2 {
                                name = t.Get(0).Strval()
                                size = t.Get(1).Integer()
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // truncate name size  name size ...
                        if i+1 < nargs {
                                name = args[i+0].Strval()
                                size = args[i+1].Integer()
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

func builtinLink(context *Scope, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        oldname, newname string
                )
                switch t := a.(type) {
                case *Pair: // link oldname => newname old => new
                        oldname = t.Key.Strval()
                        newname = t.Value.Strval()
                case *Group: // link (oldname newname) (old new)
                        if t.Len() == 2 {
                                oldname = t.Get(0).Strval()
                                newname = t.Get(1).Strval()
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // link oldname newname, old new, ...
                        if t.Len() == 2 {
                                oldname = t.Get(0).Strval()
                                newname = t.Get(1).Strval()
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // link oldname newname  oldname newname ...
                        if i+1 < nargs {
                                oldname = args[i+0].Strval()
                                newname = args[i+1].Strval()
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

func builtinSymlink(context *Scope, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        oldname, newname string
                )
                switch t := a.(type) {
                case *Pair: // symlink oldname => newname old => new
                        oldname = t.Key.Strval()
                        newname = t.Value.Strval()
                case *Group: // symlink (oldname newname) (old new)
                        if t.Len() == 2 {
                                oldname = t.Get(0).Strval()
                                newname = t.Get(1).Strval()
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // symlink oldname newname, old new, ...
                        if t.Len() == 2 {
                                oldname = t.Get(0).Strval()
                                newname = t.Get(1).Strval()
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // symlink newname oldname  newname oldname ...
                        if i+1 < nargs {
                                oldname = args[i+0].Strval()
                                newname = args[i+1].Strval()
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

func builtinReadDir(context *Scope, args... Value) (res Value, err error) {
        var l []Value
        for _, a := range args {
                var fis []os.FileInfo
                if fis, err = ioutil.ReadDir(a.Strval()); err == nil {
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
                res = toListOrValue(l)
        }
        return
}

func builtinReadFile(context *Scope, args... Value) (res Value, err error) {
        var l []Value
        for _, a := range args {
                var s []byte
                if s, err = ioutil.ReadFile(a.Strval()); err == nil {
                        l = append(l, &String{string(s)})
                } else {
                        break //l = append(l, UniversalNone)
                }
        }
        if err == nil {
                res = toListOrValue(l)
        }
        return
}

func builtinWriteFile(context *Scope, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name, data string
                        perm = os.FileMode(0600)
                )
                switch t := a.(type) {
                case *Pair: // write-file name => text name => text
                        name = t.Key.Strval()
                        data = t.Value.Strval()
                case *Group: // write-file (name text) (name text 0660)
                        if n := t.Len(); n < 4 && n > 0 {
                                name = t.Get(0).Strval()
                                if n > 1 {
                                        data = t.Get(1).Strval()
                                }
                                if n > 2 {
                                        perm = os.FileMode(t.Get(2).Integer() & 0777)
                                }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // write-file name text, name text 0660, ...
                        if n := t.Len(); n < 4 && n > 0 {
                                name = t.Get(0).Strval()
                                if n > 1 {
                                        data = t.Get(1).Strval()
                                }
                                if n > 2 {
                                        perm = os.FileMode(t.Get(2).Integer() & 0777)
                                }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // write-file name text 0660  name text 0660 ...
                        name = args[i].Strval()
                        if i+1 < nargs {
                                data = args[i+1].Strval()
                                i += 1
                        }
                        if i+1 < nargs {
                                perm = os.FileMode(args[i+1].Integer() & 0777)
                                i += 1
                        }
                }
                if err = ioutil.WriteFile(name, []byte(data), perm); err != nil {
                        break
                }
        }
        return
}
