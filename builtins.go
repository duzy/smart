//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/scanner"
        "extbit.io/smart/token"
        "encoding/base64"
        "path/filepath"
        //"hash/crc64"
        "io/ioutil"
        "net/http"
        "os/exec"
        goctx "context"
        "strings"
        "strconv"
	"unicode"
        "errors"
        "regexp"
        "bytes"
        "bufio"
        "time"
        "fmt"
        "os"
        "io"
)

type Position token.Position
type BuiltinFunc func(pos Position, args... Value) (Value, error)

var builtins = map[string]BuiltinFunc {
        `typeof`:       builtinTypeOf,

        `position`:     builtinPosition,

        `error`:        builtinError,
        `warning`:      builtinWarning,

        `assert-valid`: builtinAssertValid,

        `or`:           builtinOr,
        `and`:          builtinAnd,
        /*
        `xor`:          builtinXor,
        */
        `not`:          builtinNot,

        `not-equal`:    builtinNotEqual,
        `equal`:        builtinEqual,
        `match`:        builtinMatch,

        `if`:           builtinBranchIf,
        `ifeq`:         builtinBranchIfEq,
        `ifne`:         builtinBranchIfNE,

        `foreach`:      builtinForEach,

        `env`:          builtinEnv,
        `var`:          builtinValue,
        `value`:        builtinValue,
        `list`:         builtinList,

        `shell`:        builtinShell,

        `serve-http`:   builtinServeHttp,
        `serve-https`:  builtinServeHttps,
        
        `print`:        builtinPrint,
        `printl`:       builtinPrintl,
        `println`:      builtinPrintln,

        //`plus`:    builtinPlus,
        //`minus`:   builtinMinus,

        `quote`:                builtinQuote,
        `quote-join`:           builtinQuoteJoin,
        `split-string`:         builtinSplitString,
        `split-quote`:          builtinSplitQuote,
        `split-quote-join`:     builtinSplitQuoteJoin,
        `split-join-quote`:     builtinSplitJoinQuote,
        `unique`:               builtinUnique,
        `join`:                 builtinJoin, // concat
        `field`:                builtinField,
        `fields`:               builtinFields,

        //`usee`:       builtinUsee,
        
        `path`:         builtinPath,
        `string`:       builtinString,
        `strip`:        builtinStrip,
        `title`:        builtinTitle,
        `trim`:         builtinTrim,
        `trim-space`:   builtinTrimSpace,
        `trim-left`:    builtinTrimLeft,
        `trim-right`:   builtinTrimRight,
        `trim-prefix`:  builtinTrimPrefix,
        `trim-suffix`:  builtinTrimSuffix,
        `trim-ext`:     builtinTrimExt,

        `indent`:       builtinIndent,

        `substring`:    builtinSubstring,

        // https://www.gnu.org/software/make/manual/html_node/Text-Functions.html
        `subst`:        builtinSubst,
        `patsubst`:     builtinPatsubst,

        `contains`:     builtinContains,
        `filter`:       builtinFilter,
        `filter-out`:   builtinFilterOut,

        `encode-base64`:builtinEncodeBase64,
        `decode-base64`:builtinDecodeBase64,

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
        `dir2`:       builtinDir2,
        `dir3`:       builtinDir3,
        `dir4`:       builtinDir4,
        `dir5`:       builtinDir5,
        `dir6`:       builtinDir6,
        `dir7`:       builtinDir7,
        `dir8`:       builtinDir8,
        `dir9`:       builtinDir9,
        `dirs`:       builtinDirs, // do `dir` n times

        `undir`:      builtinUndir,
        `undir2`:     builtinUndir2,
        `undir3`:     builtinUndir3,
        `undir4`:     builtinUndir4,
        `undir5`:     builtinUndir5,
        `undir6`:     builtinUndir6,
        `undir7`:     builtinUndir7,
        `undir8`:     builtinUndir8,
        `undir9`:     builtinUndir9,
        `undirs`:     builtinUndirs, // do `undir` n times

        `dir-chop`:   builtinDirChop,

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

        `file-exists`:builtinFileExists,// stat
        `file-source`:builtinFileSource,
        `file`:       builtinFile,
        `wildcard`:   builtinWildcard,

        // TODO: move these into builtin package 'io/ioutil'
        `read-dir`:   builtinReadDir,   // io/ioutil/ioutil.go
        `read-file`:  builtinReadFile,  // io/ioutil/ioutil.go
        `write-file`: builtinWriteFile, // io/ioutil/ioutil.go
        `touch-file`: builtinTouchFile,

        `grep`:       builtinGrep,

        `return`:     builtinReturn,
}

func RegisterBuiltins(m map[string]BuiltinFunc) (err error) {
        for s, f := range m {
                if _, existed := builtins[s]; existed {
                        err = fmt.Errorf("Builtin '%s' already existed", s)
                        break
                } else {
                        builtins[s] = f
                }
        }
        return
}

func EscapedString(v Value) (s string, e error) {
        if p, ok := v.(*String); ok {
                if s, e = p.Strval(); e == nil {
                        s = strings.Replace(s, "\\'", "'", -1)
                }
        } else {
                s, e = v.Strval()
        }
        return
}

func isNotSpace(r rune) bool {
        return !unicode.IsSpace(r)
}

func isRelPath(filename string) (res bool) {
        // This implementation replaces:
        //      strings.HasPrefix(filename, "."+PathSep)
        //      strings.HasPrefix(filename, ".."+PathSep)
        var ( s = "."+PathSep ; n = len(filename) )
        if n > 1 && filename[0] == s[0] {
                if filename[1] == s[0] && n > 2 {
                        res = filename[2] == s[1]
                } else if filename[1] == s[1] {
                        res = true
                }
        }
        return
}

func isAbsOrRel(filename string) bool {
        return filepath.IsAbs(filename) || isRelPath(filename)
}

func trimLeftSpaces(s string) string {
        return strings.TrimLeftFunc(s, unicode.IsSpace)
}

func trimRightSpaces(s string) string {
        return strings.TrimRightFunc(s, unicode.IsSpace)
}

func parseFlags(args []Value, opts []string, opt func(ru rune, v Value)) (va []Value, err error) {
ForArgs:
        for _, v := range args {
                var ( runes []rune ; names []string )
                switch a := v.(type) {
                case *Flag:
                        if runes, names, err = a.opts(false, opts...); err != nil { return }
                case *Pair:
                        var flag, ok = a.Key.(*Flag)
                        if !ok { va = append(va, a); continue ForArgs }
                        if runes, names, err = flag.opts(false, opts...); err != nil { return }
                        v = a.Value // use flag value
                default:
                        va = append(va, a)
                        continue ForArgs
                }
                if enable_assertions { assert(len(runes) == len(names), "Flag.opts(...) error") }
                for _, ru := range runes { opt(ru, v) }
        }
        return
}

func tryParseFlags(args []Value, opts []string, opt func(ru rune, v Value)) (va []Value, err error) {
ForArgs:
        for _, v := range args {
                var ( runes []rune ; names []string )
                switch a := v.(type) {
                case *Flag:
                        if runes, names, err = a.opts(true, opts...); err != nil { return }
                case *Pair:
                        var flag, ok = a.Key.(*Flag)
                        if !ok { va = append(va, a); continue ForArgs }
                        if runes, names, err = flag.opts(true, opts...); err != nil { return }
                        if len(runes) > 0 { v = a.Value } // use flag value
                default:
                        va = append(va, a)
                        continue ForArgs
                }
                if enable_assertions { assert(len(runes) == len(names), "Flag.opts(...) error") }
                if len(runes) > 0 { for _, ru := range runes { opt(ru, v) }
                } else { va = append(va, v) }
        }
        return
}

func typeof(arg interface{}) (s string) {
        switch a := arg.(type) {
        case *List:
                if n := len(a.Elems); n == 1 {
                        switch v := a.Elems[0].(type) {
                        case *delegate: // FIXME: recursively undelegate types
                                if d, _ := v.x.(*Def); d != nil {
                                        s = fmt.Sprintf("%T", d.value) //s = d.value.Type().String()
                                        s = strings.ReplaceAll(strings.TrimPrefix(s, "*"), "smart.", "")
                                } else {
                                        s = "unknown"
                                }
                        default:
                                s = fmt.Sprintf("%T", v) //s = v.Type().String()
                        }
                } else if n > 1 {
                        s = "List" //ListType.name
                } else {
                        s = "None" //NoneType.name
                }
        default:
                // FIXME: this should be an exception (panic).
                s = fmt.Sprintf("%T", a) //s = a.Type().String()
                s = strings.TrimPrefix(strings.TrimPrefix(s, "*"), "smart.")
        }
        return
}

func builtinTypeOf(pos Position, args... Value) (res Value, err error) {
        var ( elems []Value; s string )
        for _, arg := range args {
                // Arguments are passed in a list:
                //   $(fun abc)                 args: (abc)
                //   $(fun a,b,c)               args: (a),(b),(c)
                //   $(fun a b c,1 2 3)         args: (a b c),(1 2 3)
                s = typeof(arg)
                elems = append(elems, &String{trivial{pos},s})
        }
        return MakeListOrScalar(pos, elems), nil
}

func builtinPosition(pos Position, args... Value) (res Value, err error) {
        var vals []Value
        var opts = []string{
                "f,filename",
                "q,quote-filename",
                "l,line",
                "c,column",
                "a,add", // add value to the last argument
        }
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if _, err = parseFlags(args, opts, func(ru rune, val Value) {
                switch ru {
                case 'f': vals = append(vals, &String{trivial{pos},pos.Filename})
                case 'q': vals = append(vals, &String{trivial{pos},"\""+pos.Filename+"\""})
                case 'l': vals = append(vals, &Int{integer{trivial{pos},int64(pos.Line)}})
                case 'c': vals = append(vals, &Int{integer{trivial{pos},int64(pos.Column)}})
                case 'a':
                        if len(vals) == 0 { break }
                        var last, okay = Scalar(vals[len(vals)-1]).(*Int)
                        if okay {
                                var n int64
                                if n, err = int64Val(val, 0); err != nil { return }
                                last.int64 += n
                        }
                }
        }); err != nil { return }
        if len(vals) > 0 {
                res = MakeListOrScalar(pos, vals)
        } else {
                res = &String{trivial{pos},pos.String()}
        }
        return
}

func builtinError(pos Position, args... Value) (res Value, err error) {
        var (
                s bytes.Buffer
                v string
        )
        for i, a := range args {
                if i > 0 { fmt.Fprintf(&s, " ") }
                if v, err = a.Strval(); err == nil {
                        fmt.Fprintf(&s, "%s", v)
                } else {
                        fmt.Fprintf(stderr, "%s: %v\n", pos, err)
                        return
                }
        }
        err = fmt.Errorf("%v", s.String())
        fmt.Fprintf(stderr, "%s: %v\n", pos, s.String())
        return
}

func builtinWarning(pos Position, args... Value) (res Value, err error) {
        var (
                s bytes.Buffer
                v string
        )
        for i, a := range args {
                if i > 0 { fmt.Fprintf(&s, " ") }
                if v, err = a.Strval(); err == nil {
                        fmt.Fprintf(&s, "%s", v)
                } else {
                        fmt.Fprintf(stderr, "%s: %v\n", pos, err)
                        return
                }
        }
        fmt.Fprintf(stderr, "%s: %v\n", pos, s.String())
        return
}

func builtinAssertValid(pos Position, args... Value) (Value, error) {
        for _, a := range args {
                if s, e := a.Strval(); e != nil {
                        return nil, e
                } else if s == "" {
                        return nil, fmt.Errorf("invalid value")
                }
        }
        return nil, nil
}

func builtinOr(pos Position, args... Value) (res Value, err error) {
        var t bool
        for _, a := range args {
                if t, err = a.True(); err != nil {
                        break
                } else if t {
                        res = a
                        break
                }
        }
        return
}

func builtinAnd(pos Position, args... Value) (res Value, err error) {
        var t bool
        for _, a := range args {
                if t, err = a.True(); err != nil {
                        res = nil; break
                } else if t {
                        res = a
                } else {
                        res = nil; break
                }
        }
        return
}

// $(not x y z) -> (not (or x y z))
// $(not x,y,z) -> (and (not x) (not y) (not z))
func builtinNot(pos Position, args... Value) (res Value, err error) {
        var t bool
        for _, a := range args {
                if t, err = a.True(); err != nil { return } else
                if t {
                        res = &boolean{trivial{pos},false}
                        return
                }
        }
        if err == nil {res = &boolean{trivial{pos},true}}
        return
}

func builtinNotEqual(pos Position, args... Value) (res Value, err error) {
        if n := len(args); n != 2 {
                err = errorf(pos, "wrong number of arguments ($(match <value-list>,<regexp-list>))", n)
        } else if args[0].cmp(args[1]) != cmpEqual {
                res = &boolean{trivial{pos},true}
        }
        return
}

func builtinEqual(pos Position, args... Value) (res Value, err error) {
        if n := len(args); n != 2 {
                err = errorf(pos, "wrong number of arguments ($(match <value-list>,<regexp-list>))", n)
        } else if cmp := args[0].cmp(args[1]); cmp == cmpEqual {
                res = &boolean{trivial{pos},true}
        }
        return
}

// $(match rx1 rx2 rx3, a b c d...)
func builtinMatch(pos Position, args... Value) (res Value, err error) {
        if n := len(args); n != 2 {
                err = errorf(pos, "wrong number of arguments ($(match <value-list>,<regexp-list>))", n)
                return
        }
        var rexList = merge(args[0])
        var srcList = merge(args[1])
ForMatchValues:
        for _, valRex := range rexList {
                var ( r *regexp.Regexp ; s string )
                if s, err = valRex.Strval(); err != nil { return }
                if r, err = regexp.Compile(s); err != nil { return }
                for _, valSrc := range srcList {
                        var src string
                        if valSrc == nil {
                                break ForMatchValues
                        } else if src, err = valSrc.Strval(); err != nil {
                                break ForMatchValues
                        } else if r.MatchString(src) {
                                res = &boolean{trivial{pos},true}
                                break ForMatchValues
                        }
                }
        }
        return
}

func builtinBranchIf(pos Position, args... Value) (res Value, err error) {
        if n := len(args); n > 1 {
                var t bool
                if t, err = args[0].True(); err != nil {
                        // oops
                } else if t { 
                        res = args[1]
                } else if n > 1 {
                        res = MakeListOrScalar(pos, args[2:])
                }
        }
        return
}

func builtinBranchIfEq(pos Position, args... Value) (res Value, err error) {
        if n := len(args); n > 2 {
                var (
                        a, b Value
                        s1, s2 string
                )
                if a, err = args[0].expand(expandAll); err != nil { return }
                if b, err = args[1].expand(expandDelegate); err != nil { return }
                if s1, err = a.Strval(); err != nil { return }
                if s2, err = b.Strval(); err != nil { return }
                if s1 == s2 { 
                        res = args[2]
                } else if n > 3 {
                        res = MakeListOrScalar(pos, args[3:])
                }
        }
        return
}

func builtinBranchIfNE(pos Position, args... Value) (res Value, err error) {
        if n := len(args); n > 2 {
                var (
                        a, b Value
                        s1, s2 string
                )
                if a, err = args[0].expand(expandDelegate); err != nil { return }
                if b, err = args[1].expand(expandDelegate); err != nil { return }
                if s1, err = a.Strval(); err != nil { return }
                if s2, err = b.Strval(); err != nil { return }
                if s1 != s2 { 
                        res = args[2]
                } else if n > 3 {
                        res = MakeListOrScalar(pos, args[3:])
                }
        }
        return
}

func builtinFor(pos Position, args... Value) (res Value, err error) {
        if n := len(args); n < 2 {
                err = scanner.Errorf(token.Position(pos), "not enough arguments ($(foreach <list>,<template>))", n)
        } else {
                var ( defs []*Def ; vals, values []Value )
                if values, err = mergeresult(ExpandAll(args[0])); err != nil { return }

                scope := context.globe.scope
                for i := 1; i <= maxNumVarVal; i += 1 {
                        def := scope.Lookup(strconv.Itoa(i)).(*Def)
                        defs = append(defs, def)
                        vals = append(vals, def.value)
                        if i-1 < len(values) {
                                def.value = values[i-1]
                        }
                }
                defer func() {
                        for i, def := range defs {
                                def.value = vals[i]
                        }
                } ()

                var list []Value
                for _, a := range args[1:] {
                        if values, err = mergeresult(ExpandAll(a)); err != nil { return }
                        if len(values) == 0 {
                                list = append(list, &None{trivial{pos}})
                        } else if len(values) == 1 {
                                list = append(list, values[0])
                        } else {
                                list = append(list, &List{elements{values}})
                        }
                }
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinForEach(pos Position, args... Value) (res Value, err error) {
        if n := len(args); n < 2 {
                err = scanner.Errorf(token.Position(pos), "not enough arguments ($(foreach <list>,<template>))", n)
        } else {
                def := context.globe.scope.Lookup("_").(*Def)
                defer func(v Value) { def.value = v } (def.value)

                var values Value
                if values, err = args[0].expand(expandAll); err != nil { return }

                // FIXME: remove the second expandAll
                if values, err = values.expand(expandAll); err != nil { return }

                var resList []Value
                for _, val := range merge(values) {
                        if isNil(val) || isNone(val) {
                                continue // ignore
                        } else if s, ok := val.(*String); ok && s.string == "" {
                                continue // ignore
                        }

                        def.value = val // set "$_" value

                        var list []Value
                        for _, a := range args[1:] {
                                var v Value
                                if v, err = a.expand(expandAll); err != nil { return }
                                if isNil(v) || isNone(v) {
                                        // ignore
                                } else if s, ok := v.(*String); ok && s.string == "" {
                                        // ignore
                                } else {
                                        list = append(list, v)
                                }
                        }
                        if n = len(list); n == 0 {
                                resList = append(resList, &None{trivial{pos}})
                        } else if n == 1 {
                                resList = append(resList, list[0])
                        } else {
                                resList = append(resList, &List{elements{list}})
                        }
                }
                res = MakeListOrScalar(pos, resList)
        }
        return
}

func builtinEnv(pos Position, args... Value) (res Value, err error) {
        var (
                vals []Value
                val Value
                v string
        )
        for _, a := range args {
                if val, err = a.expand(expandDelegate); err != nil { return }
                if val == nil {
                        // discard
                } else if v, err = val.Strval(); err == nil {
                        if s := strings.TrimSpace(v); s != "" {
                                vals = append(vals, &String{trivial{pos},os.Getenv(s)})
                        }
                } else {
                        return
                }
        }
        return MakeListOrScalar(pos, vals), nil
}

func builtinValue(pos Position, args... Value) (res Value, err error) {
        var scope *Scope
        if len(cloctx) > 0 { scope = cloctx[0] } else
        if context.loader != nil { scope = context.loader.scope }

        var vals []Value
        for _, a := range args {
                var s string
                if s, err = a.Strval(); err != nil { return }
                if def := scope.FindDef(s); def != nil {
                        vals = append(vals, def.value)
                } else {
                        vals = append(vals, &None{trivial{pos}})
                }
        }
        return MakeListOrScalar(pos, vals), nil
}

func builtinList(pos Position, args... Value) (res Value, err error) {
        res = MakeListOrScalar(pos, args)
        return
}

func builtinShell(pos Position, args... Value) (res Value, err error) {
        var vals []Value
        for _, a := range args {
                var ( bufout, buferr bytes.Buffer; s string )
                if s, err = a.Strval(); err != nil { return }
                sh := exec.Command("sh", "-c", s)
                sh.Stdout, sh.Stderr = &bufout, &buferr
                if err = sh.Run(); err != nil {
                        s = strings.TrimSpace(buferr.String())
                        err = wrap(pos, fmt.Errorf("%s", s), err)
                        return
                }
                val := &String{trivial{pos},strings.TrimSpace(bufout.String())}
                vals = append(vals, val)
                bufout.Reset()
                buferr.Reset()
        }
        return MakeListOrScalar(pos, vals), nil
}

func builtinServeHttp(pos Position, args... Value) (res Value, err error) {
        var va []Value
        var opts = []string{
                "h,host",
                "p,port",
        }
        var optHost string
        var optPort = 80
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if va, err = parseFlags(args, opts, func(ru rune, v Value) {
                switch ru {
                case 'p': if optPort, err = intVal(v, optPort); err != nil { return }
                case 'h': if optHost, err = v.Strval(); err != nil { return }
                }
        }); err != nil { return }

        var server = &http.Server{}
        server.Addr = fmt.Sprintf("%s:%d", optHost, optPort)
        fmt.Fprintf(stderr, "%s: serving http at %v\n", pos, server.Addr)
        
        http.HandleFunc("/quit", func(w http.ResponseWriter, r *http.Request) {
                io.WriteString(w, "<font color=red>Server will close in 1sec ...</font>")
                go func() {
                        time.Sleep(1 * time.Second)
                        server.Shutdown(goctx.Background())
                } ()
        })

        for _, a := range va {
                var s string
                if s, err = a.Strval(); err != nil { return }
                fmt.Fprintf(stderr, "%s: serving files %v ...\n", pos, s)
                http.Handle("/", http.FileServer(http.Dir(s)))
        }

        if err = server.ListenAndServe(); err == http.ErrServerClosed {
                // Requested /quit
        } else if err != nil {
                err = wrap(pos, err)
        }
        return
}

func builtinServeHttps(pos Position, args... Value) (res Value, err error) {
        err = scanner.Errorf(token.Position(pos), "'serve-https' is unimplemented yet")
        return
}

func builtinPrint(pos Position, args... Value) (res Value, err error) {
        var x = len(args)
        for i, a := range args {
                var s string
                if 0 < i && i < x { fmt.Printf(" ") }
                if a == nil {
                        continue
                } else if s, err = EscapedString(a); err == nil {
                        if s != "" { fmt.Printf("%s", s) }
                } else {
                        err = wrap(pos, err)
                        break
                }
        }
        return
}

func builtinPrintl(pos Position, args... Value) (res Value, err error) {
        var x = len(args)
        for i, a := range args {
                var s string
                if 0 < i && i < x { fmt.Printf(" ") }
                if s, err = EscapedString(a); err != nil { return }
                fmt.Printf("%s", s)
                if i == x && !strings.HasSuffix(s, "\n") {
                        fmt.Printf("\n")
                }
        }
        return nil, nil
}

func builtinPrintln(pos Position, args... Value) (Value, error) {
        builtinPrint(pos, args...)
        fmt.Printf("\n")
        return nil, nil
}

func builtinPlus(pos Position, args... Value) (result Value, err error) {
        var num, v int64
        for _, a := range args {
                if v, err = a.Integer(); err != nil {
                        return
                }
                num += v
        } 
        return &Int{integer{trivial{pos},num}}, nil
}

func builtinMinus(pos Position, args... Value) (result Value, err error) {
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
        return &Int{integer{trivial{pos},num}}, nil
}

func builtinUnique(pos Position, args... Value) (res Value, err error) {
        if optionBenchBuiltin {
                defer func(t time.Time) {
                        var d = time.Now().Sub(t)
                        fmt.Fprintf(stderr, "%s:(%8s) unique\n", pos, d)
                } (time.Now())
        }
        var optReverse bool
        if len(args) > 0 {
                var a []Value
                if a, err = tryParseFlags(merge(args[0]), []string{
                        "r,reverse",
                }, func(ru rune, v Value) {
                        switch ru {
                        case 'r': if optReverse, err = trueVal(v,true); err != nil { return }
                        }
                }); err != nil { return }
                args = append(a, args[1:]...)
        }
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var list []Value
ForArgs:
        for i, a := range args {
                var tmp []Value
                if optReverse { tmp = args[i+1:] } else { tmp = list }
                for _, v := range tmp {
                        if a == v || a.cmp(v) == cmpEqual {
                                continue ForArgs
                        }
                }

                if false {
                        var s1, s2 string
                        if s1, err = a.Strval(); err != nil { return }
                        for _, v := range list {
                                if s2, err = v.Strval(); err != nil { return }
                                if s1 == s2 { continue ForArgs }
                        }
                }

                list = append(list, a)
        }
        res = MakeListOrScalar(pos, list)
        return
}

func builtinJoin(pos Position, args... Value) (res Value, err error) {
        if l := len(args); l > 0 {
                var ( vals []Value ; fields []string ; sep string )
                if l < 2 {
                        if vals, err = mergeresult(ExpandAll(args...)); err != nil { return }
                } else {
                        if vals, err = mergeresult(ExpandAll(args[:l-1]...)); err != nil { return }
                        if sep, err = args[l-1].Strval(); err != nil { return }
                }
                for _, a := range vals {
                        var v string
                        if v, err = a.Strval(); err != nil { return }
                        if v != "" { fields = append(fields, v) }
                }
                res = &String{trivial{pos},strings.Join(fields, sep)}
        }
        return
}

func builtinQuote(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        if l := len(args); l > 0 {
                var fields []string
                var v string
                for _, a := range args {
                        if v, err = a.Strval(); err != nil { return }
                        if v != "" { fields = append(fields, v) }
                }
                res = &String{trivial{pos},strconv.Quote(strings.Join(fields, " "))}
        } else {
                res = &None{trivial{pos}}
        }
        return
}

func builtinQuoteJoin(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var sep string
        if l := len(args); l > 1 {
                if sep, err = args[l-1].Strval(); err != nil {
                        return
                }
                args = args[:l-1]
        }
        if l := len(args); l > 0 {
                var fields []string
                var v string
                for _, a := range args {
                        if v, err = a.Strval(); err != nil { return }
                        if v != "" { fields = append(fields, v) }
                }
                res = &String{trivial{pos},strconv.Quote(strings.Join(fields, sep))}
        } else {
                res = &None{trivial{pos}}
        }
        return
}

func builtinSplitString(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        if l := len(args); l > 0 {
                var fields []Value
                for _, a := range args {
                        var s string
                        if s, err = a.Strval(); err != nil { return }
                        if s != "" { fields = append(fields, &String{trivial{a.Position()},s}) }
                }

                res = &List{elements{fields}}
        } else {
                res = &None{trivial{pos}}
        }
        return
}

func quotestrings(value Value) {
        switch v := value.(type) {
        case *String:
                v.string = strconv.Quote(v.string)
        case *List:
                for _, elem := range v.Elems {
                        quotestrings(elem)
                }
        }
        return
}

func joinstrings(value Value, sep string) (res Value, err error) {
        if sep == "" { sep = " " }
        ValueType: switch v := value.(type) {
        case *String: res = value
        case *List:
                var strs []string
                for _, elem := range v.Elems {
                        var ( v Value; s string )
                        if v, err = joinstrings(elem, sep); err != nil { break ValueType }
                        if s, err = v.Strval(); err != nil { break ValueType }
                        if s != "" { strs = append(strs, s) }
                }
                res = &String{trivial{value.Position()},strings.Join(strs, sep)}
        }
        return
}

func builtinSplitQuote(pos Position, args... Value) (res Value, err error) {
        if res, err = builtinSplitString(pos, args...); err == nil {
                quotestrings(res)
        }
        return
}

func builtinSplitQuoteJoin(pos Position, args... Value) (res Value, err error) {
        var sep string
        if l := len(args); l > 1 {
                if sep, err = args[l-1].Strval(); err != nil {
                        return
                }
                args = args[:l-1]
        }
        if res, err = builtinSplitQuote(pos, args...); err == nil {
                res, err = joinstrings(res, sep)
        }
        return
}

func builtinSplitJoinQuote(pos Position, args... Value) (res Value, err error) {
        var sep string
        if l := len(args); l > 1 {
                if sep, err = args[l-1].Strval(); err != nil {
                        return
                }
                args = args[:l-1]
        }
        var v Value
        if v, err = builtinSplitString(pos, args...); err == nil {
                if v, err = joinstrings(v, sep); err == nil {
                        var s string
                        if s, err = v.Strval(); err == nil {
                                res = &String{trivial{pos},strconv.Quote(s)}
                        }
                }
        }
        return
}

func builtinField(pos Position, args... Value) (res Value, err error) {
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
                        res = &String{trivial{pos},s}
                }
        } else {
                res = &None{trivial{pos}}
        }
        return
}

func builtinFields(pos Position, args... Value) (Value, error) {
        // TODO: ...
        return nil, nil
}

func builtinUsee(pos Position, args... Value) (result Value, err error) {
        var proj = current()
        if proj == nil {
                err = fmt.Errorf("unknown current context")
                return
        }

        var list []Value
        for _, arg := range args {
                var ( s string; v Value )
                if s, err = arg.Strval(); err != nil {
                        return
                } else if v, err = proj.using.Get(s); err != nil {
                        return
                } else {
                        list = append(list, v)
                }
        }
        if err == nil {
                result = MakeListOrScalar(pos, list)
        }
        return
}

func builtinPath(pos Position, args... Value) (result Value, err error) {
        var list []Value
        for _, a := range args {
                var s string
                if s, err = a.Strval(); err != nil { return }
                list = append(list, MakePathStr(pos,s))
        }
        result = MakeListOrScalar(pos, list)
        return
}

func builtinString(pos Position, args... Value) (result Value, err error) {
        var s bytes.Buffer
        for i, a := range args {
                var v string
                if i > 0 { s.WriteString(" ") }
                if v, err = a.Strval(); err != nil { return }
                s.WriteString(v)
        }
        result = &String{trivial{pos},s.String()}
        return
}

func filterValues(pats []Value, neg bool, values... Value) (result []Value, err error) {
        var f = func(v Value) bool {
                for _, pat := range pats {
                        //fmt.Fprintf(stderr, "filter: (%T %v) (%T %v)\n", pat, pat, v, v)
                        switch p := pat.(type) {
                        case *PercPattern:
                                var ( s string ; stems []string )
                                if s, stems, err = p.match(v); err != nil { break }
                                if s != "" && stems != nil {
                                        return true
                                }
                        default:
                                if pat.cmp(v) == cmpEqual {
                                        return true
                                }
                                switch p := v.(type) {
                                case *File:
                                        var s string
                                        if s, err = pat.Strval(); err != nil { break }
                                        if p.name == s { return true }
                                }                        
                        }
                }
                return false
        }
        if values, err = mergeresult(Reveal(values...)); err != nil { return }
        for _, v := range values {
                var okay = f(v)
                if err != nil { break }
                if neg { okay = !okay }
                if okay { result = append(result, v) }
        }
        return
}

func builtinFilterValues(pos Position, neg bool, args... Value) (res Value, err error) {
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
                var elems []Value
                if elems, err = filterValues(pats, neg, args[1:]...); err == nil {
                        res = MakeListOrScalar(pos, elems)
                }
        }
        if res == nil && err == nil {
                res = &None{trivial{pos}}
        }
        return
}

func builtinSubstring(pos Position, args... Value) (res Value, err error) {
        var list []Value
        if n := len(args); n > 1 {
                var ( v Value ; i1, i2 int64 )

                if v, err = args[0].expand(expandAll); err != nil { return }
                if t, ok := Scalar(v).(*Int); ok {
                        // remove the first element
                        args, i1 = args[1:], t.int64
                        if l, ok := v.(*List); ok && len(l.Elems) > 1 {
                                if t, ok := Scalar(l.Elems[1]).(*Int); ok {
                                        i2 = t.int64
                                        goto CheckRange
                                }
                        }
                } else {
                        err = errorf(pos, "'%v' is not integer", args[0])
                        return
                }

                if v, err = args[0].expand(expandAll); err != nil { return }
                if t, ok := Scalar(v).(*Int); ok {
                        // remove the first element again
                        args, i2 = args[1:], t.int64
                } else {
                        i2 = i1; i1 = 0 // [:i2]
                }

        CheckRange:
                if i1 > i2 { t := i1; i1 = i2; i2 = t } // swap the wrong order
                
                var a, b = int(i1), int(i2)
                for _, arg := range args {
                        var s string
                        if s, err = arg.Strval(); err != nil { return }
                        if i := len(s); a < i {
                                if b < i {
                                        s = s[a:b]
                                } else {
                                        s = s[a:]
                                }
                        } else {
                                s = ""
                        }
                        list = append(list, &String{trivial{pos},s})
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

// $(subst from,to,text)
func builtinSubst(pos Position, args... Value) (res Value, err error) {
        var list []Value
        if nargs := len(args); nargs > 2 {
                var s, s1, s2 string
                if s1, err = args[0].Strval(); err != nil { return }
                if s2, err = args[1].Strval(); err != nil { return }
                var a []Value
                if a, err = mergeresult(Reveal(args[2:]...)); err != nil { return }
                for _, arg := range a {
                        if s, err = arg.Strval(); err != nil { return }
                        list = append(list, &String{trivial{pos},strings.Replace(s, s1, s2, -1)})
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

// TODO:
//   $(var:pattern=replacement)
//   $(var:suffix=replacement)
func builtinPatsubst(pos Position, args... Value) (res Value, err error) {
        // $(patsubst pattern,replacement,text)
        var list []Value
        if len(args) < 3 { return }

        var srcPats, dstPats, sources []Value
        if srcPats, err = mergeresult(ExpandAll(args[0])); err != nil { return }
        if dstPats, err = mergeresult(ExpandAll(args[1])); err != nil { return }
        if sources, err = mergeresult(ExpandAll(args[2:]...)); err != nil { return }

        var proj = current()
        if proj == nil {
                err = fmt.Errorf("unknown most derived context")
                return
        }

        // Using the most derived context for correct &(...)
        defer setclosure(setclosure(cloctx.unshift(proj.scope)))
        var filemaps = proj.filemaps()

ForSources:
        for _, src := range sources {
                var ( stems []string ; matched bool )

        ForSrcPats:
                for _, elem := range srcPats {
                        switch pat := elem.(type) {
                        case Pattern:
                                var s string
                                if s, stems, err = pat.match(src); err != nil {
                                        break ForSources
                                } else if s != "" && len(stems) > 0 {
                                        matched = true
                                        break ForSrcPats
                                }
                        }
                }

                if !matched {
                        // Just return what the src is if not matched.
                        if !isNone(src) {
                                list = append(list, src)
                        }
                        continue ForSources
                }

                // Compose the matched results with stem value.
        ForDstPats:
                for _, dst := range dstPats {
                        var name string
                        switch pat := dst.(type) {
                        case Pattern:
                                var rest []string
                                name, rest, err = pat.stencil(stems)
                                if err != nil { break ForSources }
                                if len(rest) > 0 {
                                        //err = fmt.Errorf("stems rest (%v)", rest)
                                        //return
                                        continue ForDstPats
                                }
                        default:
                                if name, err = pat.Strval(); err != nil {
                                        break ForSources
                                }
                        }

                        // Deal with special source value
                        switch t := src.(type) {
                        case *File:
                                var pre string
                                var match *FileMap
                                for _, m := range filemaps {
                                        if ok, s := m.Match(name); ok {
                                                match, pre = m, s
                                                break
                                        }
                                }

                                var file *File
                                if match != nil {
                                        if file = match.stat(t.dir, pre, name); file != nil {
                                                assert(file.name == name, "invalid file name")
                                        } else if file = match.stat(proj.absPath, pre, name); file != nil {
                                                assert(file.name == name, "invalid file name")
                                                /*
                                        } else if match.Paths != nil {
                                                var ( path = match.Paths[0] ; sub string )
                                                if sub, err = path.Strval(); err != nil { return }
                                                if filepath.IsAbs(sub) {
                                                        file = stat(name, "", sub, nil)
                                                } else {
                                                        file = stat(name, sub, t.dir, nil)
                                                }*/
                                        }
                                }
                                if file == nil {
                                        file = stat(pos, name, t.sub, t.dir, nil/* okay missing */)
                                }

                                list = append(list, file)
                                continue ForDstPats

                        default:
                                list = append(list, &String{trivial{pos},name})
                                continue ForDstPats
                        }
                }
        }

        res = MakeListOrScalar(pos, list)
        return
}

func builtinStrip(pos Position, args... Value) (res Value, err error) {
        return builtinTrimSpace(pos, args...)
}

func builtinTrimSpace(pos Position, args... Value) (res Value, err error) {
        return builtinTrim(pos, append([]Value{&None{trivial{pos}}}, args...)...)
}

func builtinTitle(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var (
                list []Value
                s string
        )
        for _, a := range args {
                if s, err = a.Strval(); err != nil {
                        return
                } else if s != "" {
                        list = append(list, &String{trivial{a.Position()},strings.Title(s)})
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinTrim(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var (
                list []Value
                cutset, s string
        )
        for i, a := range args {
                if s, err = a.Strval(); err != nil {
                        return
                } else if s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, &String{trivial{pos},strings.TrimSpace(s)})
                        } else {
                                list = append(list, &String{trivial{pos},strings.Trim(s, cutset)})
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinTrimLeft(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var (
                list []Value
                cutset, s string
        )
        for i, a := range args {
                if s, err = a.Strval(); err != nil {
                        return
                } else if s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, &String{trivial{a.Position()},strings.TrimLeftFunc(s, unicode.IsSpace)})
                        } else {
                                list = append(list, &String{trivial{a.Position()},strings.TrimLeft(s, cutset)})
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinTrimRight(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var (
                list []Value
                cutset, s string
        )
        for i, a := range args {
                if s, err = a.Strval(); err != nil {
                        return
                } else if s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, &String{trivial{a.Position()},strings.TrimRightFunc(s, unicode.IsSpace)})
                        } else {
                                list = append(list, &String{trivial{a.Position()},strings.TrimRight(s, cutset)})
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, list)
        }
        return
}

// $(trim-prefix foo%, fooxxx foo123)
// $(trim-prefix %/foo, xxx/foo/a/b/c)
// $(trim-prefix %%/foo, xxx/yyy/zzz/foo/a/b/c)
// FIXME: %%/foo is not working
func builtinTrimPrefix(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        if len(args) == 0 { return }

        var (
                prefix = args[0]
                list []Value
                cutset, s string
        )
        //fmt.Fprintf(stderr, "trim-prefix: %T %v : %v\n", prefix, prefix, args)
        //err = fmt.Errorf("debug"); return
        if pat, ok := prefix.(partialMatcher); ok {
                for _, a := range args[1:] {
                        var ( result string; rest, stems []string )
                        result, rest, stems, err = pat.partialMatch(a)
                        if result != "" && stems != nil && rest != nil {
                                s = filepath.Join(rest...)
                        } else if s, err = a.Strval(); err != nil {
                                return
                        } else if s == "" {
                                continue // ignore empty string
                        }

                        if s != "" {
                                list = append(list, &String{trivial{a.Position()},s})
                        }
                }
        } else if cutset, err = prefix.Strval(); err != nil {
                return
        } else {
                for _, a := range args[1:] {
                        if s, err = a.Strval(); err != nil {
                                return
                        } else if s != "" {
                                if cutset == "" {
                                        s = strings.TrimLeftFunc(s, unicode.IsSpace)
                                } else {
                                        s = strings.TrimPrefix(s, cutset)
                                }
                        }
                        if s != "" {
                                list = append(list, &String{trivial{a.Position()},s})
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinTrimSuffix(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var (
                list []Value
                cutset, s string
        )
        for i, a := range args {
                if s, err = a.Strval(); err != nil {
                        return
                } else if s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, &String{trivial{a.Position()},strings.TrimRightFunc(s, unicode.IsSpace)})
                        } else {
                                list = append(list, &String{trivial{a.Position()},strings.TrimSuffix(s, cutset)})
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinTrimExt(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var list []Value
        for i, a := range args {
                /*switch _ := a.(type) {
                case *File:
                        fmt.Fprintf(stderr, "todo: trim-ext File{%v %v %v}\n", t.dir, t.sub, t.name)
                }*/
                var ext, s string
                if s, err = a.Strval(); err != nil {
                        return
                } else if s != "" {
                        if i == 0 && len(args) > 1 {
                                ext = s
                        } else if ext == "" {
                                list = append(list, &String{trivial{a.Position()},strings.TrimSuffix(s, filepath.Ext(s))})
                        } else if ext == filepath.Ext(s) {
                                list = append(list, &String{trivial{a.Position()},strings.TrimRight(s, ext)})
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinIndent(pos Position, args... Value) (res Value, err error) {
        var (
                l []Value
                s string // indent
        )
        if x := len(args); x > 0 {
                if v, ok := Scalar(args[0]).(*Int); ok {
                        args, s = args[1:], strings.Repeat(" ", int(v.int64))
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
                l = append(l, &String{trivial{a.Position()},strings.Join(lines, "\n")})
        }
        res = MakeListOrScalar(pos, l)
        return
}

func builtinFindstring(pos Position, args... Value) (res Value, err error) {
        // $(findstring find,text)
        return
}

func builtinContains(pos Position, args... Value) (res Value, err error) {
        // $(contains v1 v2 v3,a b c … x y z)
        if len(args) < 2 { return }

        var vals []Value
        var list []Value
        if vals, err = mergeresult(ExpandAll(args[0])); err != nil { return }
        if list, err = mergeresult(ExpandAll(args[1:]...)); err != nil { return }
        // TODO: -and -or
        if true { // -or
                for _, val := range vals {
                        for _, v := range list {
                                if val == nil || v == nil {
                                        if false { fmt.Fprintf(stderr, "%s: nil values (%v,%v)\n", pos, val, v) }
                                        continue
                                }
                                if val.cmp(v) == cmpEqual {
                                        res = &boolean{trivial{pos},true}
                                        return
                                }
                        }
                }
        } else { // -and
                num := 0
                for _, val := range vals {
                        for _, v := range list {
                                if val.cmp(v) == cmpEqual {
                                        num += 1
                                        break
                                }
                        }
                }
                if num == len(vals) {
                        res = &boolean{trivial{pos},true}
                }
        }
        if res == nil {
                res = &boolean{trivial{pos},false}
        }
        return
}

func builtinFilter(pos Position, args... Value) (res Value, err error) {
        // $(filter pattern…,text)
        res, err = builtinFilterValues(pos, false, args...)
        return
}

func builtinFilterOut(pos Position, args... Value) (res Value, err error) {
        // $(filter-out pattern…,text)
        res, err = builtinFilterValues(pos, true, args...)
        return
}

func builtinSort(pos Position, args... Value) (res Value, err error) {
        // $(sort list)
        return
}

func builtinWord(pos Position, args... Value) (res Value, err error) {
        // $(word n,text)
        return
}

func builtinWordList(pos Position, args... Value) (res Value, err error) {
        // $(wordlist s,e,text)
        return
}

func builtinWords(pos Position, args... Value) (res Value, err error) {
        // $(words n,text)
        return
}

func builtinFirstWord(pos Position, args... Value) (res Value, err error) {
        // $(firstword names...)
        return
}

func builtinLastWord(pos Position, args... Value) (res Value, err error) {
        // $(lastword names...)
        return
}

func builtinEncodeBase64(pos Position, args... Value) (res Value, err error) {
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
                res = &String{trivial{pos},buf.String()}
        }
        return
}

func builtinDecodeBase64(pos Position, args... Value) (res Value, err error) {
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
                                list = append(list, &String{trivial{a.Position()},string(dat)})
                        } else {
                                return
                        }
                }
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinBase(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        var l []Value
        for _, a := range args {
                /*switch t := a.(type) {
                case *File:
                        fmt.Fprintf(stderr, "todo: base File{%v %v %v}\n", t.dir, t.sub, t.name)
                }*/
                var s string
                if s, err = a.Strval(); err != nil {
                        return
                }
                s = filepath.Base(s) // the last element of path
                l = append(l, &String{trivial{a.Position()},s})
        }
        res = MakeListOrScalar(pos, l)
        return
}

func dirx(pos Position, n int, args... Value) (res Value, err error) {
        var (
                l []Value
                s string
        )
        for _, a := range args {
                if s, err = a.Strval(); err != nil {
                        return
                }
                s = filepath.Dir(s)
                for i := n-1; 0 < i; i -= 1 {
                        s = filepath.Dir(s)
                }
                l = append(l, MakePathStr(pos,s))
        }
        res = MakeListOrScalar(pos, l)
        return
}

func undirx(pos Position, n int, args... Value) (res Value, err error) {
        var (
                l []Value
                s string
        )
        for _, a := range args {
                if s, err = a.Strval(); err != nil {
                        return
                }
                v := strings.Split(s, PathSep)
                if i := len(v); i == 0 {
                        // v is empty
                } else if n < i {
                        v = v[n:]
                } else {
                        v = v[i-1:] // empty
                }
                l = append(l, MakePathStr(pos,filepath.Join(v...)))
        }
        res = MakeListOrScalar(pos, l)
        return
}

func builtinDir(pos Position, args... Value) (res Value, err error) {
        var (
                l []Value
                s string
        )
        for _, a := range args {
                if s, err = a.Strval(); err != nil {
                        return
                }
                s = filepath.Dir(s)
                l = append(l, MakePathStr(pos,s))
        }
        res = MakeListOrScalar(pos, l)
        return
}

func builtinDir2(pos Position, args... Value) (res Value, err error) {
        return dirx(pos, 2, args...)
}

func builtinDir3(pos Position, args... Value) (res Value, err error) {
        return dirx(pos, 3, args...)
}

func builtinDir4(pos Position, args... Value) (res Value, err error) {
        return dirx(pos, 4, args...)
}

func builtinDir5(pos Position, args... Value) (res Value, err error) {
        return dirx(pos, 5, args...)
}

func builtinDir6(pos Position, args... Value) (res Value, err error) {
        return dirx(pos, 6, args...)
}

func builtinDir7(pos Position, args... Value) (res Value, err error) {
        return dirx(pos, 7, args...)
}

func builtinDir8(pos Position, args... Value) (res Value, err error) {
        return dirx(pos, 8, args...)
}

func builtinDir9(pos Position, args... Value) (res Value, err error) {
        return dirx(pos, 9, args...)
}

func builtinDirs(pos Position, args... Value) (res Value, err error) {
        var n int
        if x := len(args); x > 0 {
                if v, ok := Scalar(args[0]).(*Int); ok {
                        args, n = args[1:], int(v.int64)
                } else if v, ok := Scalar(args[x-1]).(*Int); ok {
                        args, n = args[:x-1], int(v.int64)
                } else {
                        return nil, fmt.Errorf("require (first/last) integer argument (first=%T, last=%T)", args[0], args[x-1])
                }
        }
        res, err = dirx(pos, n, args...)
        return
}

func builtinUndir(pos Position, args... Value) (res Value, err error) {
        return undirx(pos, 1, args...)
}

func builtinUndir2(pos Position, args... Value) (res Value, err error) {
        return undirx(pos, 2, args...)
}

func builtinUndir3(pos Position, args... Value) (res Value, err error) {
        return undirx(pos, 3, args...)
}

func builtinUndir4(pos Position, args... Value) (res Value, err error) {
        return undirx(pos, 4, args...)
}

func builtinUndir5(pos Position, args... Value) (res Value, err error) {
        return undirx(pos, 5, args...)
}

func builtinUndir6(pos Position, args... Value) (res Value, err error) {
        return undirx(pos, 6, args...)
}

func builtinUndir7(pos Position, args... Value) (res Value, err error) {
        return undirx(pos, 7, args...)
}

func builtinUndir8(pos Position, args... Value) (res Value, err error) {
        return undirx(pos, 8, args...)
}

func builtinUndir9(pos Position, args... Value) (res Value, err error) {
        return undirx(pos, 9, args...)
}

func builtinUndirs(pos Position, args... Value) (res Value, err error) {
        var n = 0
        if x := len(args); x > 0 {
                if v, ok := Scalar(args[0]).(*Int); ok {
                        args, n = args[1:], int(v.int64)
                } else if v, ok := Scalar(args[x-1]).(*Int); ok {
                        args, n = args[:x-1], int(v.int64)
                } else {
                        return nil, fmt.Errorf("require (first/last) integer argument (first=%T, last=%T)", args[0], args[x-1])
                }
        }
        return undirx(pos, n, args...)
}

func builtinDirChop(pos Position, args... Value) (res Value, err error) {
        var (
                l []Value
                n = 0
        )
        if x := len(args); x > 0 {
                if v, ok := Scalar(args[0]).(*Int); ok {
                        args, n = args[1:], int(v.int64)
                } else if v, ok := Scalar(args[x-1]).(*Int); ok {
                        args, n = args[:x-1], int(v.int64)
                } else {
                        return nil, fmt.Errorf("require (first/last) integer argument (first=%T, last=%T)", args[0], args[x-1])
                }
        }
        for _, a := range args {
                var s string
                if s, err = a.Strval(); err != nil {
                        return
                }
                var v = strings.Split(s, PathSep)
                if i := len(v); 0 < i {
                        if n < 0 { n = i + n }
                        if 0 <= n && n+1 < i {
                                v = append(v[0:n], v[n+1:]...)
                        } else {
                                v = append(v[0:n])
                        }
                        if len(v) > 0 && v[0] == "" {
                                v[0] = PathSep // for absolute paths
                        }
                }
                l = append(l, &String{trivial{a.Position()},filepath.Join(v...)})
        }
        res = MakeListOrScalar(pos, l)
        return
}

func builtinRelativeDir(pos Position, args... Value) (res Value, err error) {
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
                        l = append(l, &String{trivial{a.Position()},s})
                } else {
                        return
                }
        }
        res = MakeListOrScalar(pos, l)
        return
}

func builtinMkdir(pos Position, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name string
                        perm os.FileMode
                )
                switch t := a.(type) {
                case *Pair: // mkdir name => perm name => perm
                        if name, err = t.Key.Strval(); err != nil { return }
                        if perm, err = permVal(t.Value,0600); err != nil { return }
                case *Group: // mkdir (name perm) (name perm)
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { return }
                                if perm, err = permVal(t.Get(1),0600); err != nil { return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // mkdir name perm, name perm, ...
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { return }
                                if perm, err = permVal(t.Get(1),0600); err != nil { return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // mkdir name perm, name perm, ...
                        if name, err = args[i].Strval(); err != nil { return }
                        if i+1 < nargs {
                                if perm, err = permVal(args[i+1],0600); err != nil { return }
                                i += 1
                        }
                }
                if err = os.Mkdir(name, perm); err != nil { break }
        }
        return
}

func builtinMkdirAll(pos Position, args... Value) (res Value, err error) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name string
                        perm os.FileMode
                )
                switch t := a.(type) {
                case *Pair: // mkdir name => perm name => perm
                        if name, err = t.Key.Strval(); err != nil { return }
                        if perm, err = permVal(t.Value,0600); err != nil { return }
                case *Group: // mkdir (name perm) (name perm)
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { return }
                                if perm, err = permVal(t.Get(1),0600); err != nil { return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // mkdir name perm, name perm, ...
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { return }
                                if perm, err = permVal(t.Get(1),0600); err != nil { return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // mkdir name perm, name perm, ...
                        if name, err = args[i].Strval(); err != nil { return }
                        if i+1 < nargs {
                                if perm, err = permVal(args[i+1],0600); err != nil { return }
                                i += 1
                        }
                }
                if err = os.MkdirAll(name, perm); err != nil {
                        break
                }
        }
        return
}

func builtinChdir(pos Position, args... Value) (res Value, err error) {
        if len(args) == 1 {
                var str string
                if str, err = args[0].Strval(); err != nil { return }
                err = lockCD(str, 0)
        } else {
                err = errors.New("Wrong number of arguments.")
        }
        return
}

func builtinRename(pos Position, args... Value) (res Value, err error) {
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

func builtinRemove(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        // TODO: parse options like -r -v
        var (
                names []string
                str string
        )
        ArgsLoop: for _, a := range args {
                if str, err = a.Strval(); err != nil {
                        return
                }
                if names, err = filepath.Glob(str); err != nil {
                        fmt.Fprintf(stderr, "error: remove: %s\n", err)
                        break
                } else {
                        for _, s := range names {
                                if err = os.Remove(s); err != nil {
                                        break ArgsLoop
                                }
                        }
                }
        }
        return
}

func builtinRemoveAll(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

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
                        fmt.Fprintf(stderr, "error: remove-all: %s\n", err)
                        break
                } else {
                        for _, s := range names {
                                if err = os.RemoveAll(s); err != nil {
                                        break ArgsLoop
                                }
                        }
                }
        }
        return
}

func builtinTruncate(pos Position, args... Value) (res Value, err error) {
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

func builtinLink(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                // TODO: ...
        }, func(ru rune, v Value) {
                /*switch ru {
                // TODO: ...
                }*/
        }); err != nil { return }
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

/* Example:
foo: foobar
	symlink -pluv $< $@
*/
func builtinSymlink(pos Position, args... Value) (res Value, err error) {
        var optForce, optUpdate, optVerbose, optRel, optPath bool
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                "f,force",
                "u,update",
                "v,verbose",
                "l,rel", // relative
                "p,path",
        }, func(ru rune, v Value) {
                switch ru {
                case 'l': if optRel, err = trueVal(v, true); err != nil { return }
                case 'p': if optPath, err = trueVal(v, false); err != nil { return }
                case 'f': if optForce, err = trueVal(v, true); err != nil { return }
                case 'u': if optUpdate, err = trueVal(v, true); err != nil { return }
                case 'v': if optVerbose, err = trueVal(v, true); err != nil { return }
                }
        }); err != nil { return }
        if false { fmt.Printf("%v: %v\n", pos, args) }
ForArgs:
        for i, na := 0, len(args); i < na; i += 1 {
                var oldNameVal, newNameVal Value
                switch t := args[i].(type) {
                case *Pair: // symlink oldname=newname oldname=>newname...
                        oldNameVal, newNameVal = t.Key, t.Value
                case *Group: // symlink (oldname newname) (oldname newname)...
                        if t.Len() != 2 {
                                err = wrap(pos, errorf(t.Position(), "expects two values of group"))
                                return
                        }
                        oldNameVal, newNameVal = t.Get(0), t.Get(1)
                case *List: // symlink oldname newname, old new, ...
                        if t.Len() != 2 {
                                err = wrap(pos, errorf(t.Position(), "expects two values of list"))
                                return
                        }
                        oldNameVal, newNameVal = t.Get(0), t.Get(1)
                default:// Multiple pairs of names:
                        // symlink  newname oldname  newname oldname ...
                        if i+1 < na {
                                oldNameVal, newNameVal = args[i+0], args[i+1]
                                i += 1
                        } else {
                                err = wrap(pos, errorf(args[i].Position(), "expects pair of names (%v)", args[i]))
                                return
                        }
                }

                var oldname, newname string
                if oldname, err = oldNameVal.Strval(); err != nil {
                        err = wrap(oldNameVal.Position(), err)
                        err = wrap(pos, err)
                        return
                }
                if newname, err = newNameVal.Strval(); err != nil {
                        err = wrap(newNameVal.Position(), err)
                        err = wrap(pos, err)
                        return
                }

                if newname == "" {
                        err = errorf(pos, "empty new filename")
                        return
                }
                if oldname == "" {
                        err = errorf(pos, "empty old filename (%v)", )
                        return
                }

                if optForce {
                        if err = os.Remove(newname); err != nil {
                                err = nil //return
                        }
                } else if optUpdate {
                        var s string
                        if s, err = os.Readlink(newname); err != nil {
                                err = nil //continue ForArgs
                        } else if s == newname {
                                continue ForArgs
                        } else if err = os.Remove(newname); err != nil {
                                err = nil //return
                        }
                }
                if optVerbose {
                        var d = filepath.Base(newname)
                        var s = filepath.Base(oldname)
                        fmt.Fprintf(stderr, "smart: Symlink %s -> %s …", d, s)
                }
                if optRel {
                        var dir = filepath.Dir(newname)
                        oldname, err = filepath.Rel(dir, oldname)
                        if err != nil {
                                if optVerbose {
                                        fmt.Fprintf(stderr, "symlink: %s\n", err)
                                }
                                err = wrap(pos, err)
                                return
                        }
                }
                if dir := filepath.Dir(newname); optPath && dir != "." && dir != PathSep {
                        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil { return }
                }
                if err = os.Symlink(oldname, newname); err != nil {
                        if optVerbose {
                                fmt.Fprintf(stderr, "… %s\n", err)
                        }
                        break
                } else if optVerbose {
                        fmt.Fprintf(stderr, "… ok\n")
                }
        }
        return
}

func builtinFileExists(pos Position, args... Value) (res Value, err error) {
        var optKind rune
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                "d,dir", // check for directory
                "f,file", // check for regular file
                "s,symbol", // check for symbolic link
        }, func(ru rune, v Value) {
                switch ru {
                case 'f', 'd', 's':
                        if t, _ := v.True(); t { optKind = ru }
                }
        }); err != nil { return }

        var proj = current()
        if proj == nil {
                err = fmt.Errorf("unknown current context")
                return
        }

        var reses []Value
        var check = func(file *File) {
                if file.info == nil {
                        reses = append(reses, &boolean{trivial{pos},false})
                        return
                }
                var mode = file.info.Mode()
                switch optKind {
                case 'd': if mode&os.ModeDir != 0 { // IsDir()
                        reses = append(reses, &boolean{trivial{pos},true})//file
                        return
                }
                case 's': if mode&os.ModeSymlink != 0 {
                        reses = append(reses, &boolean{trivial{pos},true})//file
                        return
                }
                case 'f': if mode&os.ModeType != 0 { // IsRegular()
                        reses = append(reses, &boolean{trivial{pos},true})//file
                        return
                }
                default:
                        reses = append(reses, &boolean{trivial{pos},true})//file
                        return
                }
        }

        var checkstat = func(a Value) {
                var ( s string ; file *File )
                if s, err = a.Strval(); err != nil { return }
                if filepath.IsAbs(s) {
                        file = stat(pos, s, "", "")
                } else {
                        file = stat(pos, s, "", proj.absPath)
                }
                if file == nil { file = proj./*searchFile*/matchFile(s) }
                if file != nil { check(file) }
        }

        for _, a := range args {
                switch t := a.(type) {
                case *File: check(t)
                case *Path: checkstat(a)
                default:    checkstat(a)
                }
                //fmt.Printf("file-exists: %T %v %v\n", a, a, reses)
        }

        if err == nil {
                res = MakeListOrScalar(pos, reses)
        }
        return
}

func builtinFileSource(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var proj = current()
        if proj == nil {
                err = fmt.Errorf("unknown current context")
                return
        }

        var l []Value
        for _, a := range args {
                var str string
                if str, err = a.Strval(); err != nil { return }
                if file := proj./*searchFile*/matchFile(str); file != nil {
                        l = append(l, &String{trivial{a.Position()},file.sub})
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, l)
        }
        return
}

func builtinFile(pos Position, args... Value) (res Value, err error) {
        var optCallerContext bool
        var optReportMissing bool
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                "c,caller", // in the caller context
                "e,report", // report if not exists
        }, func(ru rune, v Value) {
                switch ru {
                case 'c': if optCallerContext, err = trueVal(v, true); err != nil { return }
                case 'e': if optReportMissing, err = trueVal(v, true); err != nil { return }
                }
        }); err != nil { return }

        var proj *Project
        if optCallerContext {
                proj = cloctx[0].project
        } else if proj = current(); proj == nil {
                err = errorf(pos, "unknown current context")
                return
        } else if false {
                // Ensure that we're in the right closure context
                defer setclosure(setclosure(cloctx.unshift(proj.scope)))
        }

        var list []Value
        for _, a := range args {
                var str string
                if file, ok := a.(*File); ok {
                        list = append(list, file)
                        if exists(file) { continue }
                        if optReportMissing {
                                fmt.Fprintf(stderr, "%s: `%v` no such file\n", pos, a)
                        }
                } else if str, err = a.Strval(); err != nil {
                        //fmt.Fprintf(stderr, "%s: %v", pos, err)
                        err = wrap(pos, err)
                        return
                } else if file = proj.matchFile(str); file != nil {
                        list = append(list, file)
                        if optReportMissing {
                                fmt.Fprintf(stderr, "%s: `%v` no such file\n", pos, a)
                        }
                } else {
                        err = errorf(pos, "`%v` is not a file", a)
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

type wildcardOpts struct {
        optIncludeMissing bool
        optVerbose bool
}

func builtinWildcard(pos Position, args... Value) (res Value, err error) {
        var wo wildcardOpts
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                "m,include-missing",
        }, func(ru rune, v Value) {
                switch ru {
                case 'm': if wo.optIncludeMissing, err = trueVal(v, true); err != nil { return }
                case 'v': if wo.optVerbose, err = trueVal(v, true); err != nil { return }
                }
        }); err != nil { return }

        var proj = current()
        if proj == nil {
                err = fmt.Errorf("unknown most derived context")
                return
        }

        var files []*File
        if files, err = proj.wildcard(pos, wo, args...); err == nil {
                var list []Value
                for _, f := range files {
                        list = append(list, f)
                }
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinReadDir(pos Position, args... Value) (res Value, err error) {
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
                                v.Append(&String{trivial{a.Position()},fi.Name()})
                        }
                        l = append(l, v)
                } else {
                        break //l = append(l, &None{trivial{pos}})
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, l)
        }
        return
}

func builtinReadFile(pos Position, args... Value) (res Value, err error) {
        var l []Value
        for _, a := range args {
                var (
                        s []byte
                        str string
                        apos = a.Position()
                )
                if !apos.IsValid() { apos = pos }
                if str, err = a.Strval(); err != nil { return }
                if str == "" {
                        err = errorf(apos, "`%v` is empty file name", a)
                        break
                }
                if s, err = ioutil.ReadFile(str); err == nil {
                        l = append(l, &String{trivial{pos},string(s)})
                } else {
                        break
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, l)
        }
        return
}

func builtinWriteFile(pos Position, args... Value) (res Value, err error) {
        // $(write-file filename,content)
        // $(write-file -p filename,content)
        var optPath = false
        if args, err = parseFlags(args, []string{
                "p,path",
        }, func(ru rune, v Value) {
                switch ru {
                case 'p': if optPath, err = trueVal(v, false); err != nil { return }
                }
        }); err != nil { return }
ForArgs:
        for i := 0; i < len(args); i += 1 {
                var (
                        a = args[i]
                        name, data string
                        perm = os.FileMode(0600)
                )
                switch t := a.(type) {
                case *Pair: // write-file name => text name => text
                        if name, err = t.Key.Strval(); err != nil { return }
                        if data, err = t.Value.Strval(); err != nil { return }
                case *Group: // write-file (name text) (name text 0660)
                        if n := t.Len(); n < 4 && n > 0 {
                                if name, err = t.Get(0).Strval(); err != nil { return }
                                if n > 1 { if data, err = t.Get(1).Strval(); err != nil { return }}
                                if n > 2 { if perm, err = permVal(t.Get(2),0600); err != nil { return }}
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // write-file name text, name text 0660, ...
                        if n := t.Len(); n < 4 && n > 0 {
                                if name, err = t.Get(0).Strval(); err != nil { return }
                                if n > 1 { if data, err = t.Get(1).Strval(); err != nil { return }}
                                if n > 2 { if perm, err = permVal(t.Get(2),0600); err != nil { return }}
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // write-file name text 0660  name text 0660 ...
                        if name, err = args[i].Strval(); err != nil { return }
                        if i+1 < len(args) {
                                if data, err = args[i+1].Strval(); err != nil { return }
                                i += 1
                        }
                        if i+1 < len(args) {
                                if perm, err = permVal(args[i+1],0600); err != nil { return }
                                i += 1
                        }
                }
                if name == "" {
                        continue ForArgs
                } else if dir := filepath.Dir(name); optPath && dir != "." && dir != PathSep {
                        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil { return }
                }
                if err = ioutil.WriteFile(name, []byte(data), perm); err != nil {
                        break
                }
        }
        return
}

func touch(file Value, optMode uint32, optPath bool, ts ...time.Time) (err error) {
        var name string
        if name, err = file.Strval(); err != nil || name == "" { return }
        if dir := filepath.Dir(name); optPath && dir != "." && dir != PathSep {
                if err = os.MkdirAll(dir, os.FileMode(optMode|0733)); err != nil { return }
        }

        var mode = os.FileMode(optMode)
        var m os.FileMode
        var at, mt time.Time
        if len(ts) > 0 { at = ts[0] } else { at = time.Now() }
        if len(ts) > 1 { mt = ts[1] } else { mt = time.Now() }
        if fi, k := file.(*File); k && fi.info != nil { m = fi.info.Mode() } else
        if fi, e := os.Stat(name); e == nil && fi != nil { m = fi.Mode() } else {
                var f *os.File
                f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_APPEND, mode&os.ModePerm)
                if err == nil { err = f.Close() }
        }
        if err == nil { err = os.Chtimes(name, at, mt) }
        if err == nil && mode != 0 && m != 0 && mode != m {
                err = os.Chmod(name, mode)
        }
        return
}

func builtinTouchFile(pos Position, args... Value) (res Value, err error) {
        // $(touch-file filename)
        // $(touch-file -p filename)
        var optPath = false
        var optMode = os.FileMode(0600)
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "p,path",
                "m,mode",
        }, func(ru rune, v Value) {
                switch ru {
                case 'p': if optPath, err = trueVal(v, true); err != nil { return }
                case 'm': if optMode, err = permVal(v, 0600); err != nil { return }
                }
        }); err != nil { return }
        for i := 0; i < len(args); i += 1 {
                err = touch(args[i], uint32(optMode), optPath)
                if err != nil { break }
        }
        return
}

// $(grep 'status=1',$@)
// $(grep -1 'status=1',$@)
func builtinGrep(pos Position, args... Value) (res Value, err error) {
        if len(args) != 2 {
                err = errorf(pos, "wants exactly 2 args, e.g. $(grep -1 '^example$',$(file))")
                return
        }

        var vals, list []Value
        var linesPos, linesNeg []int
        var rxs []*regexp.Regexp
        
        if vals, err = mergeresult(ExpandAll(args[0])); err != nil { return }
        for _, a := range vals {
                if i, ok := a.(*Int); ok {
                        if i.int64 > 0 {
                                linesPos = append(linesPos, int(i.int64))
                        } else if i.int64 < 0{
                                linesNeg = append(linesNeg, int(i.int64))
                        } else {
                                err = errorf(a.Position(), "zero line number")
                                return
                        }
                } else if s, e := a.Strval(); e != nil {
                        err = wrap(a.Position(), e); return
                } else if s == "" {
                        err = errorf(a.Position(), "empty regexp"); return
                } else if r, e := regexp.Compile(s); e != nil {
                        err = wrap(a.Position(), e); return
                } else {
                        rxs = append(rxs, r)
                }
        }

        if vals, err = mergeresult(ExpandAll(args[1:]...)); err != nil { return }
        for _, a := range vals {
                var file *os.File
                var filename string
                if filename, err = a.Strval(); err != nil {
                        err = wrap(pos, err)
                        return
                }
                if file, err = os.Open(filename); err != nil {
                        err = wrap(pos, err)
                        return
                }
                defer file.Close()

                var greps = make(map[int][]string,2)
                var line int // line number
                var scanner = bufio.NewScanner(file)
                scanner.Split(bufio.ScanLines)
                for scanner.Scan() {
                        var text = scanner.Text()
                        line += 1 // starting from #1
                        for _, rx := range rxs {
                                var sm = rx.FindStringSubmatch(text)
                                if len(sm) > 0 {
                                        greps[line] = append(greps[line], sm[0])
                                }
                        }
                }
                if linesPos == nil && linesNeg == nil {
                        for n, ss := range greps {
                                //list = append(list, s)
                                fmt.Printf("grep: %v %v\n", n, ss)
                        }
                } else {
                        for _, n := range linesPos {
                                var ss, ok = greps[n]
                                if !ok || ss == nil { continue }
                                var elems = []Value{&Int{integer{trivial{pos},int64(line+n)}}}
                                for _, s := range ss {
                                        elems = append(elems, &String{trivial{pos},s})
                                }
                                list = append(list, &Group{trivial{pos},List{elements{elems}}})
                        }

                        line += 1 // go behind the last line 
                        for _, n := range linesNeg {
                                var ss, ok = greps[line+n]
                                if !ok || ss == nil { continue }
                                var elems = []Value{&Int{integer{trivial{pos},int64(line+n)}}}
                                for _, s := range ss {
                                        elems = append(elems, &String{trivial{pos},s})
                                }
                                list = append(list, &Group{trivial{pos},List{elements{elems}}})
                        }
                }
                greps = nil
        }
        if err == nil {
                res = MakeListOrScalar(pos, list)
        }
        return
}

var (
        rsConfigRef = `\$\{([^\s\}]+)\}|@([^\s\@]+)@`
        rsConfigure = `^[\t ]*#[\t ]*(define|smartdefine|smartdefine01|cmakedefine|cmakedefine01)[\t ]+([A-Za-z0-9_]+)(?:[\t ]+([^\n]*))?$`
        rxConfigure = regexp.MustCompile(fmt.Sprintf(`(?m:%s)`, rsConfigure))
        rxConfigRef = regexp.MustCompile(rsConfigRef)
)

func (scope *Scope) configExpand(pos Position, s string) string {
        var res = new(bytes.Buffer)
        var index = 0
        for _, m := range rxConfigRef.FindAllStringSubmatchIndex(s, -1) {
                fmt.Fprint(res, s[index:m[0]])
                index = m[1] // reset index immediately to keep forward

                var name string
                switch {
                case m[2] > m[0] && m[3] > m[2]: // ${VAR}
                        name = s[m[2]:m[3]]
                case m[4] > m[0] && m[5] > m[4]: // @VAR@
                        name = s[m[4]:m[5]]
                }

                if def := scope.FindDef(name); def != nil {
                        var val, err = def.Call(pos)
                        if false { fmt.Printf("%s: %v: %s %v\n", pos, name, typeof(val), val) }
                        if err != nil { fmt.Fprintf(stderr, "%s: %v", pos, err) } else
                        if isNil(val) || isNone(val) { continue }
                        switch t := val.(type) {
                        case *Plain: fmt.Fprintf(res, "%s", t.Value)
                        case *answer, *boolean:
                                if v, e := t.Integer(); e == nil {
                                        fmt.Fprintf(res, "%d", v)
                                }
                        case *Group:
                                if v, e := parseGroupValue(t).Strval(); e == nil {
                                        fmt.Fprintf(res, "%s", v)
                                }
                        default:
                                if v, e := val.Strval(); e == nil {
                                        fmt.Fprintf(res, "%s", v)
                                }
                        }
                }
        }
        if index < len(s) { fmt.Fprint(res, s[index:]) }
        return res.String()
}

func configure(pos Position, out *bytes.Buffer, scope *Scope, str string) (err error) {
        var index = 0
        str = scope.configExpand(pos, str)
        for _, m := range rxConfigure.FindAllStringSubmatchIndex(str, -1) {
                if _, err = out.WriteString(str[index:m[0]]); err != nil { return }
                index = m[1] // reset index immediately to keep forward

                var t bool
                var s string
                var verb = str[m[2]:m[3]]
                var name = str[m[4]:m[5]]
                var hasv = m[6] > m[0] && m[7] > m[6]
                var def = scope.FindDef(name)
                //fmt.Fprintf(stderr, "%v: configure: %v %v %v\n", scope.comment, verb, name, def)
                switch verb {
                case "define":
                        if hasv && !(def == nil || def.value == nil) {
                                v := str[m[6]:m[7]] //scope.expand(str[m[6]:m[7]])
                                s = fmt.Sprintf("#define %s %s", name, v)
                        } else {
                                s = fmt.Sprintf("#define %s", name)
                        }
                case "smartdefine", "cmakedefine":
                        if def != nil {
                                if t, err = def.True(); err != nil {
                                        return
                                }
                        }
                        if !t {
                                s = fmt.Sprintf("/* #undef %s */", name)
                        } else if hasv {
                                v := str[m[6]:m[7]] //scope.expand(str[m[6]:m[7]])
                                s = fmt.Sprintf("#define %s %s", name, v)
                        } else {
                                s = fmt.Sprintf("#define %s", name)
                        }
                case "smartdefine01", "cmakedefine01":
                        if def != nil {
                                if t, err = def.True(); err != nil {
                                        return
                                }
                        }
                        if !t {
                                s = fmt.Sprintf("#define %s 0", name)
                        } else if hasv {
                                v := str[m[6]:m[7]] //scope.expand(str[m[6]:m[7]])
                                s = fmt.Sprintf("#define %s 1 /* %s */", name, v)
                        } else {
                                s = fmt.Sprintf("#define %s 1", name)
                        }
                }

                if _, err = out.WriteString(s); err != nil { return }
        }
        if index < len(str) {
                _, err = out.WriteString(str[index:])
        }
        return
}

func builtinReturn(pos Position, args... Value) (res Value, err error) {
        //if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        return nil, &Returner{ args }
}
