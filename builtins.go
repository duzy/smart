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
        "hash/crc64"
        "io/ioutil"
        "strings"
        "strconv"
	"unicode"
        "errors"
        "regexp"
        "bytes"
        "fmt"
        "os"
        "io"
)

type Position token.Position
type BuiltinFunc func(pos Position, args... Value) (Value, error)

var builtins = map[string]BuiltinFunc {
        `typeof`:       builtinTypeOf,

        `error`:        builtinError,
        `warning`:      builtinWarning,

        `assert-valid`: builtinAssertValid,

        `or`:           builtinLogicalOr,
        `and`:          builtinLogicalAnd,
        /*
        `xor`:          builtinLogicalXor,
        */
        `not`:          builtinLogicalNot,

        `match`:        builtinMatch,

        `if`:           builtinBranchIf,
        `ifeq`:         builtinBranchIfEq,
        `ifne`:         builtinBranchIfNE,

        `foreach`:      builtinForEach,

        `env`:          builtinEnv,
        `var`:          builtinValue,
        `value`:        builtinValue,
        
        `print`:        builtinPrint,
        `printl`:       builtinPrintl,
        `println`:      builtinPrintln,

        //`plus`:    builtinPlus,
        //`minus`:   builtinMinus,

        `quote`:        builtinQuote,
        `quote-join`:   builtinQuoteJoin,
        `split-string`: builtinSplitString,
        `split-quote`:  builtinSplitQuote,
        `split-quote-join`: builtinSplitQuoteJoin,
        `split-join-quote`: builtinSplitJoinQuote,
        `unique`:       builtinUnique,
        `join`:         builtinJoin, // concat
        `field`:        builtinField,
        `fields`:       builtinFields,

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

        `configure-file`: builtinConfigureFile,

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

func builtinTypeOf(pos Position, args... Value) (res Value, err error) {
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

func builtinLogicalOr(pos Position, args... Value) (res Value, err error) {
        for _, a := range args {
                if a.True() { res = a; break }
        }
        return
}

func builtinLogicalAnd(pos Position, args... Value) (res Value, err error) {
        for _, a := range args {
                if a.True() { res = a } else { res = nil; break }
        }
        return
}

// $(not x y z) -> (not (or x y z))
// $(not x,y,z) -> (and (not x) (not y) (not z))
func builtinLogicalNot(pos Position, args... Value) (res Value, err error) {
        for _, a := range args {
                if a.True() {
                        res = universalfalse;
                        break
                }
        }
        if res == nil {
                res = universaltrue;
        }
        return
}

func builtinMatch(pos Position, args... Value) (res Value, err error) {
        if n := len(args); n != 2 {
                err = scanner.Errorf(token.Position(pos), "wrong number of arguments ($(match <value-list>,<regexp-list>))", n)
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
                        if src, err = valSrc.Strval(); err != nil {
                                break ForMatchValues
                        } else if r.MatchString(src) {
                                res = universaltrue
                                break ForMatchValues
                        }
                }
        }
        return
}

func builtinBranchIf(pos Position, args... Value) (res Value, err error) {
        if n := len(args); n > 1 {
                var cond Value
                if cond, err = args[0].expand(expandAll); err != nil { return }
                if cond.True() { 
                        res = args[1]
                } else if n > 1 {
                        res = MakeListOrScalar(args[2:])
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
                        res = MakeListOrScalar(args[3:])
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
                        res = MakeListOrScalar(args[3:])
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
                        vals = append(vals, def.Value)
                        if i-1 < len(values) {
                                def.Value = values[i-1]
                        }
                }
                defer func() {
                        for i, def := range defs {
                                def.Value = vals[i]
                        }
                } ()

                var list []Value
                for _, a := range args[1:] {
                        if values, err = mergeresult(ExpandAll(a)); err != nil { return }
                        if len(values) == 0 {
                                list = append(list, universalnone)
                        } else if len(values) == 1 {
                                list = append(list, values[0])
                        } else {
                                list = append(list, &List{elements{values}})
                        }
                }
                res = MakeListOrScalar(list)
        }
        return
}

func builtinForEach(pos Position, args... Value) (res Value, err error) {
        if n := len(args); n < 2 {
                err = scanner.Errorf(token.Position(pos), "not enough arguments ($(foreach <list>,<template>))", n)
        } else {
                def := context.globe.scope.Lookup("_").(*Def)
                defer func(v Value) { def.Value = v } (def.Value)

                var values Value
                if values, err = args[0].expand(expandAll); err != nil { return }

                // FIXME: remove the second expandAll
                if values, err = values.expand(expandAll); err != nil { return }

                var resList []Value
                for _, val := range merge(values) {
                        def.Value = val // set value

                        var list []Value
                        for _, a := range args[1:] {
                                var v Value
                                if v, err = a.expand(expandAll); err != nil { return }
                                list = append(list, v)
                        }
                        if n = len(list); n == 0 {
                                resList = append(resList, universalnone)
                        } else if n == 1 {
                                resList = append(resList, list[0])
                        } else {
                                resList = append(resList, &List{elements{list}})
                        }
                }
                res = MakeListOrScalar(resList)
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
                                vals = append(vals, &String{os.Getenv(s)})
                        }
                } else {
                        return
                }
        }
        return MakeListOrScalar(vals), nil
}

func builtinValue(pos Position, args... Value) (res Value, err error) {
        var scope *Scope
        switch {
        case len(execstack) > 0: scope = execstack[0].scope
        case context.loader != nil: scope = context.loader.scope
        }
        var vals []Value
        for _, a := range args {
                var s string
                if s, err = a.Strval(); err != nil { return }
                if def := scope.FindDef(s); def != nil {
                        vals = append(vals, def.Value)
                } else {
                        vals = append(vals, universalnone)
                }
        }
        return MakeListOrScalar(vals), nil
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
                        fmt.Fprintf(stderr, "%s:(print) %v", pos, err)
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
        return &Int{integer{num}}, nil
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
        return &Int{integer{num}}, nil
}

func builtinUnique(pos Position, args... Value) (res Value, err error) {
        var optReverse bool
        if len(args) > 0 {
                var head []Value
                for _, a := range merge(args[0]) {
                        var opt bool
                        switch t := a.(type) {
                        case *Flag:
                                if opt, err = t.is(0, "reverse"); err != nil { return }
                                if opt { optReverse = true }
                        case *Pair:
                                if opt, err = t.isFlag(0, "reverse"); err != nil { return }
                                if opt { optReverse = t.Value.True() }
                        default:
                                head = append(head, a)
                        }
                }
                args = append(head, args[1:]...)
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
        res = MakeListOrScalar(list)
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
                res = &String{strings.Join(fields, sep)}
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
                res = &String{strconv.Quote(strings.Join(fields, " "))}
        } else {
                res = universalnone
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
                res = &String{strconv.Quote(strings.Join(fields, sep))}
        } else {
                res = universalnone
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
                        if s != "" { fields = append(fields, &String{s}) }
                }

                res = &List{elements{fields}}
        } else {
                res = universalnone
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
                res = &String{strings.Join(strs, sep)}
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
                                res = &String{strconv.Quote(s)}
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
                        res = &String{s}
                }
        } else {
                res = universalnone
        }
        return
}

func builtinFields(pos Position, args... Value) (Value, error) {
        // TODO: ...
        return nil, nil
}

func builtinString(pos Position, args... Value) (result Value, err error) {
        var s bytes.Buffer
        for i, a := range args {
                var v string
                if i > 0 { s.WriteString(" ") }
                if v, err = a.Strval(); err != nil { return }
                s.WriteString(v)
        }
        result = &String{s.String()}
        return
}

func filterValues(pats []Value, neg bool, values... Value) (result []Value, err error) {
        var f = func(v Value) bool {
                for _, pat := range pats {
                        //fmt.Fprintf(stderr, "filter: (%T %v) (%T %v)\n", pat, pat, v, v)
                        switch p := pat.(type) {
                        case *PercPattern:
                                var ( s string ; m bool )
                                if m, s, err = p.match(v); err != nil { break }
                                if m && s != "" { return true }
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
                        res = MakeListOrScalar(elems)
                }
        }
        if res == nil && err == nil {
                res = universalnone
        }
        return
}

func builtinSubstring(pos Position, args... Value) (res Value, err error) {
        var list []Value
        if n := len(args); n > 1 {
                var ( v Value ; i1, i2 int64 )

                if v, err = args[0].expand(expandAll); err != nil { return }
                if t := Scalar(v, IntType); t != nil {
                        args = args[1:] // remove the first element
                        if i1, err = t.Integer(); err != nil { return }
                        if l, ok := t.(*List); ok && len(l.Elems) > 1 && l.Elems[1].Type() == IntType {
                                if i2, err = l.Elems[1].Integer(); err != nil { return }
                                goto CheckRange
                        }
                } else {
                        err = scanner.Errorf(token.Position(pos), "'%v' is not integer", args[0])
                        return
                }

                if v, err = args[0].expand(expandAll); err != nil { return }
                if v = Scalar(v, IntType); v != nil {
                        args = args[1:] // remove the first element again
                        if i2, err = v.Integer(); err != nil { return }
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
                        list = append(list, &String{s})
                }
        }
        res = MakeListOrScalar(list)
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
                        list = append(list, &String{ strings.Replace(s, s1, s2, -1) })
                }
        }
        res = MakeListOrScalar(list)
        return
}

// TODO:
//   $(var:pattern=replacement)
//   $(var:suffix=replacement)
func builtinPatsubst(pos Position, args... Value) (res Value, err error) {
        // $(patsubst pattern,replacement,text)
        var ( list []Value ; nargs = len(args) )
        if nargs < 3 { return }

        var srcPats, dstPats, sources []Value
        if srcPats, err = mergeresult(Disclose(args[0])); err != nil { return }
        if dstPats, err = mergeresult(Disclose(args[1])); err != nil { return }
        if sources, err = mergeresult(ExpandAll(args[2:]...)); err != nil { return }

        var proj = mostDerived()
        if proj == nil {
                err = fmt.Errorf("unknown most derived context")
                return
        }

        // Using the most derived context for correct &(...)
        defer setclosure(setclosure(cloctx.unshift(proj.scope)))

        var filemaps = proj.filemaps(false)

ForSources:
        for _, src := range sources {
                var ( matched bool ; stem = new(String) )

        ForSrcPats:
                for _, elem := range srcPats {
                        switch pat := elem.(type) {
                        case *PercPattern:
                                if matched, stem.string, err = pat.match(src); err != nil { break ForSources }
                                if matched && stem.string != "" { break ForSrcPats }
                        }
                }

                if !matched || stem.string == "" {
                        // Just return what the src is if not matched.
                        if src.Type().Kind() != NoneKind {
                                list = append(list, src)
                        }
                        continue ForSources
                }

                // Compose the matched results with stem value.
        ForDstPats:
                for _, dst := range dstPats {
                        var prefix, suffix Value
                        switch pat := dst.(type) {
                        case *PercPattern:
                                prefix, suffix = pat.Prefix, pat.Suffix
                        default:
                                // FIXME: *GlobPattern, *RegexpPattern, etc.
                                prefix, suffix = universalnone, universalnone
                        }

                        // Just compose the regular value
                        var comp = new(Barecomp)
                        comp.Append(prefix, stem, suffix)

                        // Deal with special source value
                        switch t := src.(type) {
                        case *File:
                                var name string
                                if name, err = comp.Strval(); err != nil { break ForSources }

                                var match *FileMap
                                for _, m := range filemaps {
                                        if ok, _ := m.Match(name); ok {
                                                match = m
                                                break
                                        }
                                }

                                var file *File
                                if match != nil {
                                        if file = match.stat(t.dir, name); file != nil {
                                                assert(file.name == name, "invalid file name")
                                        } else if file = match.stat(proj.absPath, name); file != nil {
                                                assert(file.name == name, "invalid file name")
                                        } else if match.Paths != nil {
                                                var ( path = match.Paths[0] ; sub string )
                                                if sub, err = path.Strval(); err != nil { return }
                                                if filepath.IsAbs(sub) {
                                                        file = stat(name, "", sub, nil)
                                                } else {
                                                        file = stat(name, sub, t.dir, nil)
                                                }
                                        }
                                }
                                if file == nil {
                                        file = stat(name, t.sub, t.dir, nil/* okay missing */)
                                }

                                list = append(list, file)
                                continue ForDstPats

                        default:
                                list = append(list, comp)
                                continue ForDstPats
                        }
                }
        }

        res = MakeListOrScalar(list)
        return
}

func builtinStrip(pos Position, args... Value) (res Value, err error) {
        return builtinTrimSpace(pos, args...)
}

func builtinTrimSpace(pos Position, args... Value) (res Value, err error) {
        return builtinTrim(pos, append([]Value{ universalnone }, args...)...)
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
                        list = append(list, &String{strings.Title(s)})
                }
        }
        if err == nil {
                res = MakeListOrScalar(list)
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
                                list = append(list, &String{strings.TrimSpace(s)})
                        } else {
                                list = append(list, &String{strings.Trim(s, cutset)})
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(list)
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
                                list = append(list, &String{strings.TrimLeftFunc(s, unicode.IsSpace)})
                        } else {
                                list = append(list, &String{strings.TrimLeft(s, cutset)})
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(list)
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
                                list = append(list, &String{strings.TrimRightFunc(s, unicode.IsSpace)})
                        } else {
                                list = append(list, &String{strings.TrimRight(s, cutset)})
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(list)
        }
        return
}

func builtinTrimPrefix(pos Position, args... Value) (res Value, err error) {
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
                                list = append(list, &String{strings.TrimLeftFunc(s, unicode.IsSpace)})
                        } else {
                                list = append(list, &String{strings.TrimPrefix(s, cutset)})
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(list)
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
                                list = append(list, &String{strings.TrimRightFunc(s, unicode.IsSpace)})
                        } else {
                                list = append(list, &String{strings.TrimSuffix(s, cutset)})
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(list)
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
                                list = append(list, &String{strings.TrimSuffix(s, filepath.Ext(s))})
                        } else if ext == filepath.Ext(s) {
                                list = append(list, &String{strings.TrimRight(s, ext)})
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(list)
        }
        return
}

func builtinIndent(pos Position, args... Value) (res Value, err error) {
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

func builtinFindstring(pos Position, args... Value) (res Value, err error) {
        // $(findstring find,text)
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
                res = &String{buf.String()}
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
                                list = append(list, &String{string(dat)})
                        } else {
                                return
                        }
                }
                res = MakeListOrScalar(list)
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
                l = append(l, &String{s})
        }
        res = MakeListOrScalar(l)
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
                l = append(l, MakePathStr(s))
        }
        res = MakeListOrScalar(l)
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
                l = append(l, MakePathStr(filepath.Join(v...)))
        }
        res = MakeListOrScalar(l)
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
                l = append(l, MakePathStr(s))
        }
        res = MakeListOrScalar(l)
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
        var (
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
        var (
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
                var i int64
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
                l = append(l, &String{filepath.Join(v...)})
        }
        res = MakeListOrScalar(l)
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
                        l = append(l, &String{s})
                } else {
                        return
                }
        }
        res = MakeListOrScalar(l)
        return
}

func builtinMkdir(pos Position, args... Value) (res Value, err error) {
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

func builtinMkdirAll(pos Position, args... Value) (res Value, err error) {
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

func builtinChdir(pos Position, args... Value) (res Value, err error) {
        if len(args) == 1 {
                var str string
                if str, err = args[0].Strval(); err != nil { return }
                err = os.Chdir(str)
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

func builtinSymlink(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var va []Value
        var optForce, optUpdate, optVerbose bool
        var opts = []string{
                "f,force",
                "u,update",
                "v,verbose",
        }
ForArgs:
        for _, v := range args {
                var ( runes []rune ; names []string )
                switch a := v.(type) {
                case *Flag:
                        if runes, names, err = a.opts(opts...); err != nil { return }
                        a = nil
                case *Pair:
                        if flag, ok := a.Key.(*Flag); ok && flag != nil {
                                if runes, names, err = flag.opts(opts...); err != nil { return }
                                v = a.Value // use flag value
                        } else {
                                va = append(va, a)
                                continue ForArgs
                        }
                default:
                        va = append(va, a)
                        continue ForArgs
                }
                if enable_assertions {
                        assert(len(runes) == len(names), "Flag.opts(...) error")
                }
                for _, ru := range runes {
                        switch ru {
                        case 'f': optForce = trueVal(v, true)
                        case 'u': optUpdate = trueVal(v, true)
                        case 'v': optVerbose = trueVal(v, true)
                        }
                }
        }
ForVals:
        for i, na := 0, len(va); i < na; i += 1 {
                var oldname, newname string
                switch t := va[i].(type) {
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
                        if i+1 < na {
                                if oldname, err = va[i+0].Strval(); err != nil { return }
                                if newname, err = va[i+1].Strval(); err != nil { return }
                                i += 1
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong arguments `%v'", va))
                                break
                        }
                }

                if optForce {
                        if err = os.Remove(newname); err != nil {
                                err = nil //return
                        }
                } else if optUpdate {
                        var s string
                        if s, err = os.Readlink(newname); err != nil {
                                err = nil //continue ForVals
                        } else if s == newname {
                                continue ForVals
                        } else if err = os.Remove(newname); err != nil {
                                err = nil //return
                        }
                }
                if optVerbose {
                        fmt.Fprintf(stderr, "smart: Symlink %s -> %s …", newname, oldname)
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
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        // TODO: $(file-exists -f filename)
        // TODO: $(file-exists -d dirname)

        var proj = current()
        if proj == nil {
                err = fmt.Errorf("unknown current context")
                return
        }

        var l []Value
        for _, a := range args {
                var (str string)
                if str, err = a.Strval(); err != nil { return }
                if file := proj.searchFile(str); file != nil {
                        if file.exists() {
                                l = append(l, file)
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(l)
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
                if file := proj.searchFile(str); file != nil {
                        l = append(l, &String{file.sub})
                }
        }
        if err == nil {
                res = MakeListOrScalar(l)
        }
        return
}

func builtinFile(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var va []Value
        var optCallerContext bool
        var optReportMissing bool
        var opts = []string{
                "c,caller", // in the caller context
                "e,report", // report if not exists
        }
ForArgs:
        for _, arg := range args {
                var ( runes []rune ; names []string ; v Value )
                switch a := arg.(type) {
                case *Flag:
                        if runes, names, err = a.opts(opts...); err != nil { return }
                        v = nil // no flag value
                case *Pair:
                        if flag, ok := a.Key.(*Flag); ok && flag != nil {
                                if runes, names, err = flag.opts(opts...); err != nil { return }
                                v = a.Value // got flag value
                        } else {
                                err = fmt.Errorf("`%v` unknown argument", a)
                                return
                        }
                default:
                        va = append(va, a)
                        continue ForArgs
                }
                if enable_assertions {
                        assert(len(runes) == len(names), "Flag.opts(...) error")
                }
                for _, ru := range runes {
                        switch ru {
                        case 'c': optCallerContext = trueVal(v, true)
                        case 'e': optReportMissing = trueVal(v, true)
                        }
                }
        }

        var proj *Project
        if optCallerContext {
                proj = cloctx[0].project
        } else if proj = current(); proj == nil {
                err = fmt.Errorf("unknown current context")
                return
        }

        var list []Value
        for _, a := range va {
                var str string
                if file, ok := a.(*File); ok {
                        if !file.exists() && optReportMissing {
                                fmt.Fprintf(stderr, "%s: `%v` no such file\n", pos, a)
                        }
                } else if str, err = a.Strval(); err != nil {
                        fmt.Fprintf(stderr, "%s: %v", pos, err)
                        return
                } else if file = proj.matchFile(str); file != nil {
                        list = append(list, file)
                        if optReportMissing {
                                fmt.Fprintf(stderr, "%s: `%v` no such file\n", pos, a)
                        }
                } else {
                        fmt.Fprintf(stderr, "%s: `%v` is not a file (%v)\n", pos, a, proj)
                }
        }
        res = MakeListOrScalar(list)
        return
}

func builtinWildcard(pos Position, args... Value) (res Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var proj = mostDerived()
        if proj == nil {
                err = fmt.Errorf("unknown most derived context")
                return
        }

        var files []*File
        if files, err = proj.wildcard(args...); err == nil {
                var list []Value
                for _, f := range files {
                        list = append(list, f)
                }
                res = MakeListOrScalar(list)
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
                                v.Append(&String{fi.Name()})
                        }
                        l = append(l, v)
                } else {
                        break //l = append(l, universalnone)
                }
        }
        if err == nil {
                res = MakeListOrScalar(l)
        }
        return
}

func builtinReadFile(pos Position, args... Value) (res Value, err error) {
        var l []Value
        for _, a := range args {
                var (
                        s []byte
                        str string
                )
                if str, err = a.Strval(); err != nil { return }
                if str == "" {
                        err = scanner.Errorf(token.Position(pos), "`%v` empty file name", a)
                        break
                }
                if s, err = ioutil.ReadFile(str); err == nil {
                        l = append(l, &String{string(s)})
                } else {
                        break //l = append(l, universalnone)
                }
        }
        if err == nil {
                res = MakeListOrScalar(l)
        }
        return
}

func builtinWriteFile(pos Position, args... Value) (res Value, err error) {
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

                var name string
                switch {
                case m[2] > m[0] && m[3] > m[2]: // ${VAR}
                        name = s[m[2]:m[3]]
                case m[4] > m[0] && m[5] > m[4]: // @VAR@
                        name = s[m[4]:m[5]]
                }

                if def := scope.FindDef(name); def != nil {
                        val, err := def.Call(pos)
                        if err != nil {
                                fmt.Fprintf(stderr, "%s: %v", pos, err)
                        } else if val == nil {
                                continue
                        }
                        switch t := val.(type) {
                        case *answer, *boolean:
                                if v, e := t.Integer(); e == nil {
                                        fmt.Fprintf(res, "%d", v)
                                }
                        default:
                                if v, e := val.Strval(); e == nil {
                                        fmt.Fprintf(res, "%s", v)
                                }
                        }
                }

                index = m[1]
        }
        if index < len(s) {
                fmt.Fprint(res, s[index:])
        }
        return res.String()
}

func configure(pos Position, out *bytes.Buffer, scope *Scope, str string) (err error) {
        var index = 0
        str = scope.configExpand(pos, str)
        for _, m := range rxConfigure.FindAllStringSubmatchIndex(str, -1) {
                if _, err = out.WriteString(str[index:m[0]]); err != nil { return }

                var s string
                var verb = str[m[2]:m[3]]
                var name = str[m[4]:m[5]]
                var hasv = m[6] > m[0] && m[7] > m[6]
                switch def := scope.FindDef(name); verb {
                case "define":
                        if hasv && !(def == nil || def.Value == nil) {
                                v := str[m[6]:m[7]] //scope.expand(str[m[6]:m[7]])
                                s = fmt.Sprintf("#define %s %s", name, v)
                        } else {
                                s = fmt.Sprintf("#define %s", name)
                        }
                case "smartdefine", "cmakedefine":
                        if def == nil || def.Value == nil || !def.Value.True() {
                                s = fmt.Sprintf("/* #undef %s */", name)
                        } else if hasv {
                                v := str[m[6]:m[7]] //scope.expand(str[m[6]:m[7]])
                                s = fmt.Sprintf("#define %s %s", name, v)
                        } else {
                                s = fmt.Sprintf("#define %s", name)
                        }
                case "smartdefine01", "cmakedefine01":
                        if def == nil || def.Value == nil || !def.Value.True() {
                                s = fmt.Sprintf("#define %s 0", name)
                        } else if hasv {
                                v := str[m[6]:m[7]] //scope.expand(str[m[6]:m[7]])
                                s = fmt.Sprintf("#define %s 1 /* %s */", name, v)
                        } else {
                                s = fmt.Sprintf("#define %s 1", name)
                        }
                }

                if _, err = out.WriteString(s); err != nil { return } else {
                        index = m[1]
                }
        }
        if index < len(str) {
                _, err = out.WriteString(str[index:])
        }
        return
}

// configure-file builtin (see also modifierConfigureFile), example usage:
// 
//      config.h:[(compare)]: config.h.in
//      	configure-file -p -m=0600 $@ $(read-file $<)
//     
func builtinConfigureFile(pos Position, args... Value) (res Value, err error) {
        var optVerbose = true
        var optPath = false
        var optMode = os.FileMode(0600)
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if len(args) > 0 {
                var altargs []Value
                for _, arg := range args {
                        var opt bool
                        switch a := arg.(type) {
                        default: altargs = append(altargs, a)
                        case *None: // ignores
                        case *Flag:
                                if opt, err = a.is('p', "path"); err != nil { return } else if opt { optPath = opt }
                                if opt, err = a.is('s', "silent"); err != nil { return } else if opt { optVerbose = false }
                        case *Pair:
                                if opt, err = a.isFlag('m', "mode"); err != nil { return } else if opt {
                                        var num int64
                                        if num, err = a.Value.Integer(); err != nil { return } else {
                                                optMode = os.FileMode(num & 0777)
                                        }
                                }
                        }
                }
                args = altargs
        }
        if len(args) < 1 { return }

        var scope *Scope
        switch {
        case len(execstack) > 0: scope = execstack[0].scope
        case context.loader != nil: scope = context.loader.scope
        }
        if scope == nil {
                err = fmt.Errorf("unknown configure scope")
                return
        }

        var data bytes.Buffer
        for _, arg := range args[1:] {
                var str string
                if str, err = arg.Strval(); err != nil { return }
                if str == "" { continue }
                if err = configure(pos, &data, scope, str); err != nil {
                        return
                }
        }
        if data.Len() == 0 && len(args) <= 1 {
                // no content produced
                return
        }

        var file *File
        var filename string
        if file, _ = args[0].(*File); file == nil {
                if filename, err = args[0].Strval(); err != nil {
                        return
                }
                var dir = filepath.Dir(filename)
                var name = filepath.Base(filename)
                file = stat(name, "", dir)
        } else if filename, err = file.Strval(); err != nil {
                unreachable()
        } else if filename == "" {
                err = fmt.Errorf("invalid file `%v`", file)
                return
        }
        if file.info == nil {
                if f := stat(filename, "", ""); f != nil {
                        file.info = f.info
                }
        }
        if file.info != nil {
                var f *os.File
                if optVerbose { fmt.Fprintf(stderr, "smart: Checking %v …", file) }
                if f, err = os.Open(filename); err == nil && f != nil {
                        defer f.Close()
                        if st, _ := f.Stat(); st.Mode().Perm() != optMode {
                                if err = f.Chmod(optMode); err != nil {
                                        fmt.Fprintf(stderr, "… (error: %s)\n", err)
                                        return
                                }
                        }
                        w1 := crc64.New(crc64Table)
                        w2 := crc64.New(crc64Table)
                        if _, err = io.Copy(w1, f); err != nil {
                                fmt.Fprintf(stderr, "… (error: %s)\n", err)
                                return
                        }
                        if _, err = w2.Write(data.Bytes()); err != nil {
                                fmt.Fprintf(stderr, "… (error: %s)\n", err)
                                return
                        }
                        if s1, s2 := w1.Sum64(), w2.Sum64(); s1 == s2 {
                                if optVerbose { fmt.Fprintf(stderr, "… Good\n") }
                                res = file
                                return
                        }
                }
                if optVerbose { fmt.Fprintf(stderr, "… Outdated\n") }
        } else if dir := filepath.Dir(filename); optPath && dir != "." && dir != PathSep {
                if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil { return }
        }

        var status string
        if optVerbose {
                printEnteringDirectory()
                fmt.Fprintf(stderr, "smart: Updating %v …", file)
                defer func() {
                        if err != nil { status = "error!" } else {
                                if status == "" { status = "done." }
                        }
                        fmt.Fprintf(stderr, "… %s\n", status)
                } ()
        }

        if err = ioutil.WriteFile(filename, data.Bytes(), optMode); err == nil {
                if file.info != nil { res = file } else {
                        if file.info, err = os.Stat(filename); err == nil {
                                context.globe.stamp(filename, file.info.ModTime())
                                res = file
                        }
                }
                status = fmt.Sprintf("updated (%d bytes).", data.Len())
        }
        return
}

func builtinReturn(pos Position, args... Value) (res Value, err error) {
        //if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        return nil, &Returner{ args }
}
