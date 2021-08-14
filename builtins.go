//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/token"
        "encoding/base64"
        "path/filepath"
        //"hash/crc64"
        "io/ioutil"
        "net/http"
        "os/exec"
        goctx "context"
        "reflect"
        "strings"
        "strconv"
        "unicode"
        "unsafe"
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

func (pos *Position) Equals(other *Position) bool {
        return (*token.Position)(pos).Equals((*token.Position)(other))
}

func (pos *Position) SameLine(other *Position) bool {
        return (*token.Position)(pos).SameLine((*token.Position)(other))
}

type BuiltinFunc func(pos Position, args... Value) (Value)

var builtins = map[string]BuiltinFunc {
        `typeof`:       builtinTypeOf,
        `defined`:      builtinDefined,

        `position`:     builtinPosition,
        `date`:         builtinDate,

        `error`:        builtinError,
        //`warning`:      builtinWarning,

        //`assert`: builtinAssert,

        `defor`:        builtinDefor,
        `or`:           builtinOr,
        `and`:          builtinAnd,
        /*
        `xor`:          builtinXor,
        */
        `not`:          builtinNot,

        `not-equal`:    builtinNotEqual,
        `equal`:        builtinEqual,
        `equals`:       builtinEqual,
        `match`:        builtinMatch,

        `greater`:      builtinGreater,
        `less`:         builtinLess,

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

        // `print`:        builtinPrint,
        // `printl`:       builtinPrintl,
        // `println`:      builtinPrintln,

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
        `trim`:         builtinTrim,
        `trim-space`:   builtinTrimSpace,
        `trim-left`:    builtinTrimLeft,
        `trim-right`:   builtinTrimRight,
        `trim-prefix`:  builtinTrimPrefix,
        `trim-suffix`:  builtinTrimSuffix,
        `trim-ext`:     builtinTrimExt,

        `uppercase`:    builtinUpperCase,
        `lowercase`:    builtinLowerCase,
        `title`:        builtinTitle,

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

        // Fullname of a file or identical to the input
        `fullname`: builtinFullname,

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
        // `write-file`: builtinWriteFile, // io/ioutil/ioutil.go
        // `touch-file`: builtinTouchFile,

        `grep`:       builtinGrep,

        // `return`:     builtinReturn,
}

var commands = map[string]BuiltinFunc {
        `print`:        builtinPrint,
        `printl`:       builtinPrintl,
        `println`:      builtinPrintln,

        `assert`:       builtinAssert,

        //`error`:        builtinError,
        `warning`:      builtinWarning,

        `append`:       builtinAppend,

        //`read-dir`:     builtinReadDir,   // io/ioutil/ioutil.go
        //`read-file`:    builtinReadFile,  // io/ioutil/ioutil.go
        `write-file`:   builtinWriteFile, // io/ioutil/ioutil.go
        `touch-file`:   builtinTouchFile,

        `return`:       builtinReturn,
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

/*
func parseFlags(args []Value, opts []string, opt func(ru rune, v Value)) (va []Value, err error) {
ForArgs:
        for i, v := range args {
                if i == 0 && false {
                        diag.warnOf(v, "FIXME: parseFlags is deprecated by parseOpts").
                                debug(optionDebugErrors)
                }
                var ( runes []rune ; names []string )
                switch a := v.(type) {
                case *Flag:
                        if runes, names, err = a.opts(false, opts...); err != nil { diag.errorOf(a, "%v", err); return }
                case *Pair:
                        var flag, ok = a.Key.(*Flag)
                        if !ok { va = append(va, a); continue ForArgs }
                        if runes, names, err = flag.opts(false, opts...); err != nil { diag.errorOf(a.Key, "%v", err); return }
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
                        if runes, names, err = a.opts(true, opts...); err != nil { diag.errorOf(a, "%v", err); return }
                case *Pair:
                        var flag, ok = a.Key.(*Flag)
                        if !ok { va = append(va, a); continue ForArgs }
                        if runes, names, err = flag.opts(true, opts...); err != nil { diag.errorOf(a.Key, "%v", err); return }
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
}*/

type optFullname struct {
        string
        value Value
}

func parseOpt(pos Position, tag reflect.StructTag, field reflect.Value, args... Value) (rest []Value, err error) {
        var (
                val = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
                opt = string(tag)
                short, long []string // short and long opt name
                s, l string
                ok bool
        )
        if tag == "" { return args, nil }
        if s, ok = tag.Lookup("s"    ); ok { short = append(short, s) }
        if l, ok = tag.Lookup("l"    ); ok { long  = append(long , l) }
        if s, ok = tag.Lookup("short"); ok { short = append(short, s) }
        if l, ok = tag.Lookup("long" ); ok { long  = append(long , l) }
        if len(short) == 0 && len(long) == 0 {
                var t = opt[:]
                for i := strings.IndexRune(t, ','); i >= 0; {
                        if j := strings.IndexAny(t[i+1:], "; "); j == 0 {
                                diag.errorAt(pos, "illform option tag: %s", t).
                                        debug(optionDebugErrors)
                                return
                        } else if j > 0 {
                                s, l = t[:i], t[i+1:i+1+j]
                                short, long = append(short, s), append(long, l)
                                t = t[i+1+j+1:]
                                i = strings.IndexRune(t, ',')
                        } else {
                                s, l = t[:i], t[i+1:]
                                short, long = append(short, s), append(long, l)
                                break
                        }
                }
                if len(short) != len(long) || len(short) == 0 || len(long) == 0 {
                        diag.errorAt(pos, "illform option tag: %s", tag).
                                debug(optionDebugErrors)
                        return
                }
        }
        if false { diag.infoAt(pos, "%v -> %v %v\n", tag, short, long).debug(true,1) }
        if len(short) != len(long) {
                diag.errorAt(pos, "short and long option names not matching: %v, %v", short, long).
                        debug(/*optionDebugErrors*/true)
                return
        }

        var set func(reflect.Value, Value)
        set = func(val reflect.Value, v Value) {
                switch val.Kind() {
                case reflect.Bool:
                        if t, e := v.True(); e == nil { val.SetBool(t) } else {
                                diag.errorOf(v, "truthify '%v' failed: %v", v, e).debug(optionDebugErrors,1)
                        }
                case reflect.Float32, reflect.Float64:
                        if t, e := v.Float(); e == nil { val.SetFloat(t) } else {
                                diag.errorOf(v, "floatify '%v' failed: %v", v, e).debug(optionDebugErrors,1)
                        }
                case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
                        if t, e := v.Integer(); e == nil { val.SetInt(t) } else {
                                diag.errorOf(v, "integify '%v' failed: %v", v, e).debug(optionDebugErrors,1)
                        }
                case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
                        if t, e := v.Integer(); e == nil { val.SetUint(uint64(t)) } else {
                                diag.errorOf(v, "integify '%v' failed: %v", v, e).debug(optionDebugErrors,1)
                        }
                 case reflect.String:
                        if t, e := v.Strval(); e == nil { val.SetString(t) } else {
                                diag.errorOf(v, "stringify '%v' failed: %v", v, e).debug(optionDebugErrors,1)
                        }
                case reflect.Slice:
                        if tp := reflect.New(val.Type().Elem()); tp.Kind() == reflect.Ptr {
                                var tv = tp.Elem()
                                set(tv, v)
                                val.Set(reflect.Append(val, tv))
                        }
                case reflect.Interface: switch val.Type().String() {
                case "smart.Value": val.Set(reflect.ValueOf(v))
                default:
                        diag.errorOf(v, "option type unsupported: %T %v -> %v, %v", v, v, val.Kind(), val.Type()).
                                debug(optionDebugErrors,1)
                }
                case reflect.Ptr: switch val.Type().Elem().String() {
                case "smart.optFullname":
                        if x, e := v.expand(expandAll); e != nil {
                                diag.errorOf(v, "expand option '%v' failed: %v", v, e).
                                        debug(optionDebugErrors,1)
                        } else if isNil(x) || isNone(x) {
                                diag.errorOf(v, "expecting file value: %T %v", v, v).
                                        debug(optionDebugErrors,1)
                        } else /*if s, ok := fullname(x); ok {
                                val.Set(reflect.ValueOf(&optFullname{ s, x }))
                        } else if s, e = x.Strval(); e != nil {
                                diag.errorOf(x, "strval '%v' failed: %v", x, e).
                                        debug(optionDebugErrors,1)
                        } else if filepath.IsAbs(s) {
                                val.Set(reflect.ValueOf(&optFullname{ s, x }))
                        } else if proj := current(); proj == nil {
                                diag.errorOf(x, "no current project to find file '%v'", s).
                                        debug(optionDebugErrors,1)
                        } else if file := proj.FindFile(s); file != nil {
                                val.Set(reflect.ValueOf(&optFullname{ file.fullname(), x }))
                        */if _, s, ok, e = asOptFullname(nil, x); e != nil {
                                diag.errorOf(x, "fullname '%v' failed: %v", x, e).
                                        debug(optionDebugErrors,1)
                        } else if ok && s != "" {
                                val.Set(reflect.ValueOf(&optFullname{ s, x }))
                        } else {
                                diag.errorOf(v, "'%s' is not a file: %v", s, x).
                                        debug(optionDebugErrors,1)
                        }
                        if false {
                                vi := val.Interface().(*optFullname)
                                diag.warnOf(v, "%v %v %v", current(), v, vi.string).debug(true,1)
                        }
                case "smart.File":
                        if x, e := v.expand(expandAll); e != nil {
                                diag.errorOf(v, "expand option '%v' failed: %v", v, e).
                                        debug(optionDebugErrors,1)
                        } else if isNil(x) || isNone(x) {
                                diag.errorOf(v, "expecting file value: %T %v", v, v).
                                        debug(optionDebugErrors,1)
                        } else if file, ok := x.(*File); ok {
                                val.Set(reflect.ValueOf(file))
                        } else if s, e := x.Strval(); e != nil {
                                diag.errorOf(x, "strval '%v' failed: %v", x, e).
                                        debug(optionDebugErrors,1)
                        } else if proj := current(); proj == nil {
                                diag.errorOf(x, "no current project to find file '%v'", s).
                                        debug(optionDebugErrors,1)
                        } else if file = proj.FindFile(s); file != nil {
                                val.Set(reflect.ValueOf(file))
                        } else {
                                diag.errorOf(v, "'%s' is not a file", s).
                                        debug(optionDebugErrors,1)
                        }
                case "regexp.Regexp":
                        if s, e := v.Strval(); e != nil {
                                diag.errorOf(v, "stringify '%v' failed: %v", v, e).debug(optionDebugErrors,1)
                        } else if rx, e := regexp.Compile(s); e != nil {
                                diag.errorOf(v, "compile regexp '%v' failed: %v", v, e).debug(optionDebugErrors,1)
                        } else {
                                val.Set(reflect.ValueOf(rx))
                        }
                default:
                        diag.errorOf(v, "option type unsupported: %T %v -> %v, %v", v, v,
                                val.Elem().Kind(), val.Type().Elem()).
                                debug(optionDebugErrors,1)
                }
                default: switch val.Type().String() {
                case "fs.FileMode": // aka. reflect.Uint32
                        if t, e := v.Integer(); e == nil { val.SetUint(uint64(t)) } else {
                                diag.errorOf(v, "integify '%v' failed: %v", v, e).debug(optionDebugErrors,1)
                        }
                case "regex.Regex": // aka. reflect.Ptr
                        diag.errorOf(v, "TODO: regexp: %T %v -> %v, %v",
                                v, v, val.Kind(), val.Type()).debug(optionDebugErrors,1)
                default:
                        diag.errorOf(v, "option type unsupported: %T %v -> %v, %v", v, v, val.Kind(), val.Type()).
                                debug(optionDebugErrors,1)
                }}
        }
ForArgs:
        for _, arg := range args {
                var (
                        okay bool
                        flag *Flag
                        value Value
                )
                if flag, okay = arg.(*Flag); okay {
                        value = MakeBoolean(flag.position, true)
                } else if pair, ok := arg.(*Pair); ok {
                        if flag, okay = pair.Key.(*Flag); okay { value = pair.Value }
                }
                if !okay || flag == nil {
                        rest = append(rest, arg)
                        continue ForArgs
                }
                for i := 0; i < len(short) && i < len(long); i += 1 {
                        if _, match := flag.opt(short[i], long[i]); match {
                                set(val, value)
                                continue ForArgs
                        }
                }
                rest = append(rest, arg)
                continue ForArgs
        }
        if false && len(args) > 0 {
                diag.infoAt(pos, "%v,%v: %v %v %v", short, long, field.Kind(), field, rest)
        }
        return
}

func parseOpts(pos Position, iOpts interface{}, args... Value) (rest []Value, err error) {
        rest = args // NOTE: set the returning args first of all!
        if opts := reflect.ValueOf(iOpts); opts.Kind() != reflect.Ptr {
                diag.errorAt(pos, "opts must be ptr: %v", opts.Kind()).
                        debug(optionDebugErrors)
        } else if opts = opts.Elem(); opts.Kind() == reflect.Struct {
                var otyp = opts.Type()
                if false { diag.infoAt(pos, "opts: %v, %v", opts.Kind(), otyp) }
                for i := 0; i < otyp.NumField(); i += 1 {
                        var ft = otyp.Field(i)
                        var fv = opts.Field(i)
                        rest, err = parseOpt(pos, ft.Tag, fv, rest...)
                }
        } else {
                diag.errorAt(pos, "opts is not ptr of struct: %v", opts.Kind()).
                        debug(optionDebugErrors)
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
                s = strings.TrimPrefix(s, "*")
                s = strings.TrimPrefix(s, "smart.")
                s = strings.TrimPrefix(s, "ast.")
        }
        return
}

func builtinTypeOf(pos Position, args... Value) (res Value) {
        var ( elems []Value; s string )
        for _, arg := range args {
                // Arguments are passed in a list:
                //   $(fun abc)                 args: (abc)
                //   $(fun a,b,c)               args: (a),(b),(c)
                //   $(fun a b c,1 2 3)         args: (a b c),(1 2 3)
                s = typeof(arg)
                elems = append(elems, MakeString(pos, s))
        }
        return MakeListOrScalar(pos, elems)
}

func builtinDefined(pos Position, args... Value) (res Value) {
        var ( elems []Value; err error )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
        for _, arg := range args {
                var _, unresolved = arg.(*unresolvedobject)
                elems = append(elems, MakeBoolean(pos, !unresolved))
                if false { diag.infoAt(pos, "defined %v -> %v", arg, unresolved) }
        }
        return MakeListOrScalar(pos, elems)
}

type builtinPositionOpts struct {
        filename bool `f,filename`
        filenameQuoted bool `q,quote-filename;qf,quoted-filename`
        line bool `l,line`
        column bool `c,column`
        addLine int `a,add;al,add-line`
        addColumn int `ac,add-column`
}
func builtinPosition(pos Position, args... Value) (res Value) {
        var (
                opts builtinPositionOpts
                vals []Value
                err error
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "position: %v", err); return
        } else if args, err = parseOpts(pos, &opts, args...); err != nil {
                diag.errorAt(pos, "position: %v", err); return
        }

        if opts.filename {
                vals = append(vals, MakeString(pos, pos.Filename))
        } else if opts.filenameQuoted {
                var s = pos.Filename //strconv.Quote(pos.Filename)
                vals = append(vals, MakeString(pos, "\""+s+"\""))
        }

        if opts.line   { vals = append(vals, MakeInt(pos, int64(pos.Line + opts.addLine))) }
        if opts.column { vals = append(vals, MakeInt(pos, int64(pos.Column + opts.addColumn))) }
        /* case 'a':
                        if len(vals) == 0 { break }
                        var last, okay = Scalar(vals[len(vals)-1]).(*Int)
                        if okay {
                                var n int64
                                if n, err = int64Val(val, 0); err != nil {
                                        diag.errorAt(pos, "position: %v", err)
                                        return
                                }
                                last.int64 += n
                        }
        */

        if len(vals) > 0 {
                res = MakeListOrScalar(pos, vals)
        } else {
                res = MakeString(pos, pos.String())
        }
        return
}

type builtinDateOpts struct {
        now bool `n,now`
        today bool `t,today`
}
func builtinDate(pos Position, args... Value) (res Value) {
        var (
                opts = builtinDateOpts{ today:true }
                err error
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }
        if t := time.Now(); opts.now {
                res = MakeTime(pos, t)
        } else if opts.today {
                res = MakeDate(pos, t)
        }
        return
}

func builtinError(pos Position, args... Value) (res Value) {
        var (
                s bytes.Buffer
                v string
                err error
        )
        for i, a := range args {
                if i > 0 { fmt.Fprintf(&s, " ") }
                if v, err = a.Strval(); err == nil {
                        fmt.Fprintf(&s, "%s", v)
                } else {
                        diag.errorOf(a, "error: %v: %v", a, err)
                        return
                }
        }
        diag.errorAt(pos, "%s", s)
        return
}

func builtinWarning(pos Position, args... Value) (res Value) {
        var (
                s bytes.Buffer
                v string
                err error
        )
        for i, a := range args {
                if i > 0 { fmt.Fprintf(&s, " ") }
                if v, err = a.Strval(); err == nil {
                        fmt.Fprintf(&s, "%s", v)
                } else {
                        diag.errorOf(a, "warning: %v: %v", a, err)
                        return
                }
        }
        diag.warnAt(pos, "%s", s)
        return
}

func builtinAssert(pos Position, args... Value) Value {
        var vals []Value
        for _, a := range args {
                if g, ok := a.(*Group); ok {
                        vals = append(vals, g.Elems...)
                }
        }
        for _, a := range vals {
                if v, e := a.True(); e != nil {
                        diag.errorOf(a, "assert: error: %v", e).
                                debug(optionDebugErrors, 1)
                } else if !v {
                        diag.errorOf(a, "assertion failed: %v", a).
                                debug(optionDebugErrors, 1)
                }
        }
        return nil
}

// $(defor $(x),$(y),$(z)) is identical to $(if $(defined $(x)),$(x),...)
func builtinDefor(pos Position, args... Value) (res Value) {
        var err error
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err)
                return
        }
        for _, a := range args {
                var _, unresolved = a.(*unresolvedobject)
                if unresolved { continue } else {
                        res = a
                        break
                }
        }
        return
}

func builtinOr(pos Position, args... Value) (res Value) {
        var ( t bool; e error )
        for _, a := range args {
                if t, e = a.True(); e != nil {
                        diag.errorOf(a, "or: error: %v", e)
                        break
                } else if t {
                        res = a
                        break
                }
        }
        return
}

func builtinAnd(pos Position, args... Value) (res Value) {
        var ( t bool; e error )
        for _, a := range args {
                if t, e = a.True(); e != nil {
                        diag.errorOf(a, "and: error: %v", e)
                        break
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
func builtinNot(pos Position, args... Value) (res Value) {
        var ( t bool; e error )
        for _, a := range args {
                if t, e = a.True(); e != nil { diag.errorAt(pos, "not: error: %v", e); return } else
                if t {
                        res = &boolean{valbase{pos},false}
                        return
                }
        }
        if e == nil {res = &boolean{valbase{pos},true}}
        return
}

func builtinNotEqual(pos Position, args... Value) (res Value) {
        if n := len(args); n != 2 {
                diag.errorAt(pos, "wrong number of arguments, try: $(not-equal <value-list>,<regexp-list>)")
        } else if args[0].cmp(args[1]) != cmpEqual {
                res = &boolean{valbase{pos},true}
        }
        return
}

func builtinEqual(pos Position, args... Value) (res Value) {
        if n := len(args); n != 2 {
                diag.errorAt(pos, "wrong number of arguments, try: $(equal <value-list>,<regexp-list>)")
        } else if cmp := args[0].cmp(args[1]); cmp == cmpEqual {
                res = &boolean{valbase{pos},true}
        }
        return
}

func builtinGreater(pos Position, args... Value) (res Value) {
        if n := len(args); n != 2 {
                diag.errorAt(pos, "wrong number of arguments, try: $(greater <value-list>,<regexp-list>)")
        } else if cmp := args[0].cmp(args[1]); cmp == cmpGreater {
                res = &boolean{valbase{pos},true}
        }
        return
}

func builtinLess(pos Position, args... Value) (res Value) {
        if n := len(args); n != 2 {
                diag.errorAt(pos, "wrong number of arguments, try: $(less <value-list>,<regexp-list>)")
        } else if cmp := args[0].cmp(args[1]); cmp == cmpSmaller {
                res = &boolean{valbase{pos},true}
        }
        return
}

type builtinMatchOpts struct {
        regexps []*regexp.Regexp `r,reg;rx,regex;re,regexp`
}
// $(match rx1 rx2 rx3, a b c d...)
func builtinMatch(pos Position, args... Value) (res Value) {
        var (
                patList, valList []Value
                opts builtinMatchOpts
                err error
        )
        if n := len(args); n < 2 {
                diag.errorAt(pos, "wrong arguments, try: $(match <regexp-list>,<value-list>,...)").
                        debug(optionDebugErrors, 1)
                return
        } else if patList, err = mergeresult2(expandall2(expandAll, args[0])); err != nil {
                diag.errorAt(pos, "expand '%v' failed: %v", args[0], err).
                        debug(optionDebugErrors, 1)
                return
        } else if patList, err = parseOpts(pos, &opts, patList...); err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err).
                        debug(optionDebugErrors, 1)
                return
        } else if valList, err = mergeresult2(expandall2(expandAll, args[1:]...)); err != nil {
                diag.errorAt(pos, "expand value list failed: %v", err).
                        debug(optionDebugErrors, 1)
                return
        }
        /*
ForPatList:
        for _, pat := range patList {
                var ( r *regexp.Regexp ; s string )
                if s, err = pat.Strval(); err != nil {
                        diag.errorOf(pat, "strval '%v' failed: %v", pat, err).
                                debug(optionDebugErrors, 1)
                        return
                } else if r, err = regexp.Compile(s); err != nil {
                        diag.errorOf(pat, "compile regexp '%s' failed: %v", s, err).
                                debug(optionDebugErrors, 1)
                        return
                }
        ForValList:
                for _, val := range valList {
                        var str string
                        if isNil(val) || isUndef(val) || isNone(val) {
                                continue ForValList
                        } else if str, err = val.Strval(); err != nil {
                                diag.errorOf(val, "strval '%v' failed: %v", val, err).
                                        debug(optionDebugErrors, 1)
                                return
                        } else if r.MatchString(str) {
                                res = MakeBoolean(pos, true)
                                break ForPatList
                        }
                }
        }*/
ForValList:
        for _, val := range valList {
                if isNil(val) || isUndef(val) || isNone(val) {
                        continue ForValList
                }
                var str string
                if str, err = val.Strval(); err != nil {
                        diag.errorOf(val, "strval '%v' failed: %v", val, err).
                                debug(optionDebugErrors, 1)
                        return
                }
                for _, rx := range opts.regexps {
                        if rx.MatchString(str) {
                                res = MakeBoolean(pos, true)
                                break ForValList
                        }
                }
                for _, pat := range patList {
                        if matched, _, _ := pat.match(str); matched {
                                res = MakeBoolean(pos, true)
                                break ForValList
                        }
                }
        }
        return
}

// $(if cond, true-value, else-value, ...)
func builtinBranchIf(pos Position, args... Value) (res Value) {
        var err error
        if n := len(args); n > 1 {
                var t bool
                if t, err = args[0].True(); err != nil {
                        diag.errorAt(pos, "truthify if condition failed: %v", err)
                } else if t { 
                        res = args[1]
                } else if n > 1 {
                        res = MakeListOrScalar(pos, args[2:])
                }
        }
        return
}

func builtinBranchIfEq(pos Position, args... Value) (res Value) {
        if n := len(args); n > 2 {
                var (
                        a, b Value
                        s1, s2 string
                        err error
                )
                if a, err = args[0].expand(expandAll); err != nil { diag.errorAt(pos, "%v", err); return }
                if b, err = args[1].expand(expandDelegate); err != nil { diag.errorAt(pos, "%v", err); return }
                if s1, err = a.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                if s2, err = b.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                if s1 == s2 { 
                        res = args[2]
                } else if n > 3 {
                        res = MakeListOrScalar(pos, args[3:])
                }
        }
        return
}

func builtinBranchIfNE(pos Position, args... Value) (res Value) {
        if n := len(args); n > 2 {
                var (
                        a, b Value
                        s1, s2 string
                        err error
                )
                if a, err = args[0].expand(expandDelegate); err != nil { diag.errorAt(pos, "%v", err); return }
                if b, err = args[1].expand(expandDelegate); err != nil { diag.errorAt(pos, "%v", err); return }
                if s1, err = a.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                if s2, err = b.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                if s1 != s2 { 
                        res = args[2]
                } else if n > 3 {
                        res = MakeListOrScalar(pos, args[3:])
                }
        }
        return
}

func builtinFor(pos Position, args... Value) (res Value) {
        if n := len(args); n < 2 {
                diag.errorAt(pos, "not enough arguments, try: $(foreach <list>,<template>)")
        } else {
                var ( defs []*Def ; vals, values []Value; err error )
                if values, err = mergeresult(ExpandAll(args[0])); err != nil {
                        diag.errorAt(pos, "merge '%v' failed: %v", args[0], err)
                        return
                }

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
                        if values, err = mergeresult(ExpandAll(a)); err != nil {
                                diag.errorAt(pos, "merge '%v' failed: %v", a, err)
                                return
                        }
                        if len(values) == 0 {
                                list = append(list, MakeNone(pos))
                        } else if len(values) == 1 {
                                list = append(list, values[0])
                        } else {
                                list = append(list, MakeList(a.Position(), values...))
                        }
                }
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinForEach(pos Position, args... Value) (res Value) {
        if n := len(args); n < 2 {
                diag.errorAt(pos, "not enough arguments ($(foreach <list>,<template>)): %v", n)
                return
        }

        var ( values []Value; err error )
        if values, err = mergeresult(ExpandAll(args[0])); err != nil {
                diag.errorAt(pos, "merge arg0 failed: %v", err)
                return
        }

        var def *Def // = context.globe.scope.Lookup("_").(*Def)
        for _, a := range args[1:] {
                for _, d := range a.defs("_") {
                        if def == nil { def = d } else if d != def {
                                diag.errorAt(d.position  , "'_' resolves to different defs: %v", d)
                                diag.errorAt(def.position, "'_' resolves to different defs: %v", def)
                                diag.errorAt(a.Position(), "'_' is used here")
                                diag.errorAt(pos         , "'_' is used here")
                                return
                        }
                }
        }

        if def != nil { defer func(v Value) { def.value = v } (def.value) }
        if false { diag.infoAt(pos, "%v; %v; %v", def, values, args).debug(true, 1) }

        var resList []Value
        for _, val := range values {
                if isNil(val) || isUndef(val) || isNone(val) {
                        continue // ignore
                } else if s, ok := val.(*String); ok && s.string == "" {
                        continue // ignore
                } else if def != nil { def.value = val } // set "$_" value

                var list []Value
                for _, a := range args[1:] {
                        var v Value
                        if v, err = a.expand(expandAll|expandPairVal); err != nil {
                                diag.errorOf(a, "expand '%v' failed: %v", a, err).
                                        debug(optionDebugErrors, 1)
                                return
                        } else if true && len(v.defs("_")) > 0 {
                                diag.errorOf(a, "'_' in '%v' not expanded: %v", a, v).debug(true, 1)
                        }
                        if false && a.String() == "include=$_" {
                                diag.infoOf(a, "%v; %v; %v => %v", def, val, a, v).debug(true, 1)
                        }
                        if isNil(v) || isUndef(v) || isNone(v) {
                                // ignore
                        } else if s, ok := v.(*String); ok && s.string == "" {
                                // ignore
                        } else {
                                list = append(list, v)
                        }
                }
                if n := len(list); n == 0 {
                        resList = append(resList, MakeNone(pos))
                } else if n == 1 {
                        resList = append(resList, list[0])
                } else {
                        resList = append(resList, MakeList(list[0].Position(), list...))
                }
        }
        res = MakeListOrScalar(pos, resList)
        return
}

func builtinEnv(pos Position, args... Value) (res Value) {
        var (
                vals []Value
                val Value
                v string
                err error
        )
        for _, a := range args {
                if val, err = a.expand(expandDelegate); err != nil { diag.errorAt(pos, "%v", err); return }
                if val == nil {
                        // discard
                } else if v, err = val.Strval(); err == nil {
                        if s := strings.TrimSpace(v); s != "" {
                                vals = append(vals, MakeString(pos, os.Getenv(s)))
                        }
                } else {
                        diag.errorAt(pos, "%v", err)
                        return
                }
        }
        return MakeListOrScalar(pos, vals)
}

func builtinValue(pos Position, args... Value) (res Value) {
        var scope *Scope
        if len(cloctx) > 0 { scope = cloctx[0] } else
        if context.loader != nil { scope = context.loader.scope }

        var err error
        var vals []Value
        for _, a := range args {
                var s string
                if s, err = a.Strval(); err != nil {
                        diag.errorAt(pos, "strval '%v' failed: %v", a, err)
                        return
                }
                if def := scope.FindDef(s); def != nil {
                        vals = append(vals, def.value)
                } else {
                        vals = append(vals, MakeNone(pos))
                }
        }
        return MakeListOrScalar(pos, vals)
}

func builtinList(pos Position, args... Value) (res Value) {
        res = MakeListOrScalar(pos, args)
        return
}

func builtinShell(pos Position, args... Value) (res Value) {
        var ( vals []Value; err error )
        for _, a := range args {
                var ( bufout, buferr bytes.Buffer; s string )
                if s, err = a.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                sh := exec.Command("sh", "-c", s)
                sh.Stdout, sh.Stderr = &bufout, &buferr
                if err = sh.Run(); err != nil {
                        s = strings.TrimSpace(buferr.String())
                        diag.errorAt(pos, "%s", err)
                        return
                }
                val := MakeString(pos, strings.TrimSpace(bufout.String()))
                vals = append(vals, val)
                bufout.Reset()
                buferr.Reset()
        }
        return MakeListOrScalar(pos, vals)
}

type builtinServeHttpOpts struct {
        host string `h,host`
        port int `p,port`
}
func builtinServeHttp(pos Position, args... Value) (res Value) {
        var (
                opts = builtinServeHttpOpts{ port:80 }
                va []Value
                err error
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "%v", err)
                return
        } else if va, err = parseOpts(pos, &opts, args...); err != nil {
                diag.errorAt(pos, "%v", err)
                return
        }

        var server = &http.Server{}
        server.Addr = fmt.Sprintf("%s:%d", opts.host, opts.port)
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
                if s, err = a.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                fmt.Fprintf(stderr, "%s: serving files %v ...\n", pos, s)
                http.Handle("/", http.FileServer(http.Dir(s)))
        }

        if err = server.ListenAndServe(); err == http.ErrServerClosed {
                if false { diag.infoAt(pos, "http server closed") }// Requested /quit
        } else if err != nil {
                diag.errorAt(pos, "%s", err)
        }
        return
}

func builtinServeHttps(pos Position, args... Value) (res Value) {
        diag.errorAt(pos, "'serve-https' is unimplemented yet")
        return
}

func builtinPrint(pos Position, args... Value) (res Value) {
        var err error
        var x = len(args)
        for i, a := range args {
                var s string
                if 0 < i && i < x { fmt.Printf(" ") }
                if a == nil {
                        continue
                } else if s, err = EscapedString(a); err == nil {
                        if s != "" { fmt.Printf("%s", s) }
                } else {
                        diag.errorAt(pos, "%s", err)
                        break
                }
        }
        return
}

func builtinPrintl(pos Position, args... Value) (res Value) {
        var err error
        var x = len(args)
        for i, a := range args {
                var s string
                if 0 < i && i < x { fmt.Printf(" ") }
                if s, err = EscapedString(a); err != nil {
                        diag.errorAt(pos, "%s", err)
                        return
                }
                fmt.Printf("%s", s)
                if i == x && !strings.HasSuffix(s, "\n") {
                        fmt.Printf("\n")
                }
        }
        return
}

func builtinPrintln(pos Position, args... Value) (res Value) {
        builtinPrint(pos, args...)
        fmt.Printf("\n")
        return
}

type builtinAppendOpts struct {
        string bool `s,str;s,string`
        verbose bool `v,verbose`
}
func builtinAppend(pos Position, args... Value) (result Value) {
        if len(args) < 2 {
                diag.errorAt(pos, "insufficient number of arguments: %v", args)
                return
        }

        var (
                opts builtinAppendOpts
                vars []Value
                list []Value
                err error
        )
        if vars, err = mergeresult(ExpandAll(args[0])); err != nil { diag.errorOf(args[0], "%s", err); return } else
        if vars, err = parseOpts(pos, &opts, vars...); err != nil { diag.errorAt(pos, "%v", err); return }
        if list, err = mergeresult(ExpandAll(args[1:]...)); err != nil { diag.errorOf(args[1], "%s", err); return }
        if len(list) == 0 { diag.warnAt(pos, "append no values"); return }

        for _, a := range vars {
                var name string
                if name, err = a.Strval(); err != nil { diag.errorOf(a, "%s", err); break }
                if name == "" { diag.errorOf(a, "name '%v' is empty", a); break }

                var def *Def
                if def == nil {
                        var obj Object
                        obj, err = cloctx[0].project.resolveObject(name)
                        if err != nil { diag.errorOf(a, "%v", err); break } else
                        if def, _ = obj.(*Def); def == nil { /*...*/ }
                }
                if def == nil {
                        for _, scope := range cloctx {
                                if def = scope.FindDef(name); def != nil { break }
                        }
                }
                if def == nil { diag.errorAt(pos, "'%s' (%v) is undefined (%v)", name, a, cloctx); break }
                if err = def.append(list...); err != nil { diag.errorAt(pos, "%s", err); break }
        }
        return
}

func builtinPlus(pos Position, args... Value) (result Value) {
        var err error
        var num, v int64
        for _, a := range args {
                if v, err = a.Integer(); err != nil {
                        diag.errorOf(a, "%s", err)
                        return
                }
                num += v
        } 
        return &Int{integer{valbase{pos},num}}
}

func builtinMinus(pos Position, args... Value) (result Value) {
        var err error
        var num, v int64
        for i, a := range args {
                if v, err = a.Integer(); err != nil {
                        diag.errorOf(a, "%s", err)
                        return
                }
                if i == 0 {
                        num = v
                } else {
                        num -= v
                }
        }
        return &Int{integer{valbase{pos},num}}
}

type builtinUniqueOpts struct {
        reverse bool `r,reverse`
}
func builtinUnique(pos Position, args... Value) (res Value) {
        if options.benchBuiltins {
                defer func(t time.Time) {
                        var d = time.Now().Sub(t)
                        fmt.Fprintf(stderr, "%s:(%8s) unique\n", pos, d)
                } (time.Now())
        }
        var (
                opts builtinUniqueOpts
                err error
        )
        if len(args) > 0 {
                var a []Value
                if a, err = parseOpts(pos, &opts, merge(args[0])...); err != nil {
                        diag.errorOf(args[0], "%v", err)
                        return
                }
                args = append(a, args[1:]...)
        }
        if false {
                args = merge(args...)
        } else if true {
                if args, err = mergeresult(ExpandAll(args...)); err != nil {
                        diag.errorAt(pos, "%v", err); return
                }
        } else {
                var x = expandDelegate | expandPathStr | expandPairVal
                if args, err = mergeresult2(expandall2(x, args...)); err != nil {
                        diag.errorAt(pos, "%v", err); return
                }
        }

        var list []Value
ForArgs:
        for i, a := range args {
                var tmp []Value
                if opts.reverse { tmp = args[i+1:] } else { tmp = list }
                for _, v := range tmp {
                        if a == v || a.cmp(v) == cmpEqual {
                                continue ForArgs
                        }
                }

                if false {
                        var s1, s2 string
                        if s1, err = a.Strval(); err != nil { diag.errorOf(a, "%v", err); return }
                        for _, v := range list {
                                if s2, err = v.Strval(); err != nil { diag.errorOf(v, "%v", err); return }
                                if s1 == s2 { continue ForArgs }
                        }
                }

                list = append(list, a)
        }
        res = MakeListOrScalar(pos, list)
        return
}

func builtinJoin(pos Position, args... Value) (res Value) {
        if l := len(args); l > 0 {
                var ( vals []Value ; fields []string ; sep string; err error )
                if l < 2 {
                        if vals, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return }
                } else {
                        if vals, err = mergeresult(ExpandAll(args[:l-1]...)); err != nil { diag.errorAt(pos, "%v", err); return }
                        if sep, err = args[l-1].Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                }
                for _, a := range vals {
                        var v string
                        if v, err = a.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        if v != "" { fields = append(fields, v) }
                }
                res = MakeString(pos, strings.Join(fields, sep))
        }
        return
}

func builtinQuote(pos Position, args... Value) (res Value) {
        var err error
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return }
        if l := len(args); l > 0 {
                var fields []string
                var v string
                for _, a := range args {
                        if v, err = a.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        if v != "" { fields = append(fields, v) }
                }
                res = MakeString(pos, strconv.Quote(strings.Join(fields, " ")))
        } else {
                res = MakeNone(pos)
        }
        return
}

func builtinQuoteJoin(pos Position, args... Value) (res Value) {
        var err error
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return }

        var sep string
        if l := len(args); l > 1 {
                if sep, err = args[l-1].Strval(); err != nil {
                        diag.errorAt(pos, "%v", err)
                        return
                }
                args = args[:l-1]
        }
        if l := len(args); l > 0 {
                var fields []string
                var v string
                for _, a := range args {
                        if v, err = a.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        if v != "" { fields = append(fields, v) }
                }
                res = MakeString(pos, strconv.Quote(strings.Join(fields, sep)))
        } else {
                res = MakeNone(pos)
        }
        return
}

func builtinSplitString(pos Position, args... Value) (res Value) {
        var err error
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return }
        if l := len(args); l > 0 {
                var fields []Value
                for _, a := range args {
                        var s string
                        if s, err = a.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        if s != "" { fields = append(fields, MakeString(a.Position(), s)) }
                }
                res = MakeList(pos, fields...)
        } else {
                res = MakeNone(pos)
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
                res = MakeString(value.Position(), strings.Join(strs, sep))
        }
        return
}

func builtinSplitQuote(pos Position, args... Value) (res Value) {
        if res = builtinSplitString(pos, args...); !isNil(res) {
                quotestrings(res)
        }
        return
}

func builtinSplitQuoteJoin(pos Position, args... Value) (res Value) {
        var ( sep string; err error )
        if l := len(args); l > 1 {
                if sep, err = args[l-1].Strval(); err != nil {
                        diag.errorAt(pos, "%v", err); return
                }
                args = args[:l-1]
        }
        if res = builtinSplitQuote(pos, args...); !isNil(res) {
                if res, err = joinstrings(res, sep); err != nil {
                        diag.errorAt(pos, "%v", err)
                }
        } else {
                diag.errorAt(pos, "%v", err)
        }
        return
}

func builtinSplitJoinQuote(pos Position, args... Value) (res Value) {
        var ( sep string; err error )
        if l := len(args); l > 1 {
                if sep, err = args[l-1].Strval(); err != nil {
                        diag.errorAt(pos, "%v", err); return
                }
                args = args[:l-1]
        }
        var v Value
        if v = builtinSplitString(pos, args...); !isNil(v) {
                if v, err = joinstrings(v, sep); err == nil {
                        var s string
                        if s, err = v.Strval(); err == nil {
                                res = MakeString(pos, strconv.Quote(s))
                        }
                }
        }
        if err != nil { diag.errorAt(pos, "%v", err) }
        return
}

func builtinField(pos Position, args... Value) (res Value) {
        if l := len(args); l >= 2 {
                var (
                        i int64
                        s string
                        fields []string
                        err error
                )
                if i, err = args[0].Integer(); err != nil { diag.errorAt(pos, "%v", err); return }
                if s, err = args[1].Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                if l > 2 {
                        var v string
                        if v, err = args[2].Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        fields = strings.Split(s, v)
                } else {
                        fields = strings.Fields(s)
                }
                if n := int(i)-1; 0 <= n && n < len(fields) {
                        s = strings.TrimSpace(fields[n])
                        res = MakeString(pos, s)
                }
        } else {
                res = MakeNone(pos)
        }
        return
}

func builtinFields(pos Position, args... Value) (res Value) {
        // TODO: ...
        return
}

func builtinUsee(pos Position, args... Value) (result Value) {
        var proj = current()
        if proj == nil {
                diag.errorAt(pos, "unknown current context")
                return
        }

        var err error
        var list []Value
        for _, arg := range args {
                var ( s string; v Value )
                if s, err = arg.Strval(); err != nil {
                        diag.errorAt(pos, "%v", err)
                        return
                } else if v, err = proj.using.Get(s); err != nil {
                        diag.errorAt(pos, "%v", err)
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

func builtinPath(pos Position, args... Value) (result Value) {
        var err error
        var list []Value
        for _, a := range args {
                var s string
                if s, err = a.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                list = append(list, MakePathStr(pos,s))
        }
        result = MakeListOrScalar(pos, list)
        return
}

func builtinString(pos Position, args... Value) (result Value) {
        var err error
        var s bytes.Buffer
        for i, a := range args {
                var v string
                if i > 0 { s.WriteString(" ") }
                if v, err = a.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                s.WriteString(v)
        }
        result = MakeString(pos, s.String())
        return
}

func filterValues(pats []Value, neg bool, values... Value) (result []Value, err error) {
        const info = false
        var f = func(v Value) bool {
                for _, pat := range pats {
                        if info { if full, s, stems := pat.match(v); full || s != "" {
                                diag.warnOf(pat, "pat=%v (%T) value=%v (%T) => full=%v result=%v stems=%v",
                                        pat, pat, v, v, full, s, stems).
                                        debug(true, 1)
                        }}
                        if ok, _, _ := pat.match(v); ok { return true }
                }
                return false
        }
        if values, err = mergeresult(Reveal(values...)); err != nil {
                diag.errorOf(values[0], "%v", err)
                return
        }
        for _, v := range values {
                var okay = f(v)
                if err != nil { break }
                if neg { okay = !okay }
                if okay { result = append(result, v) }
        }
        return
}

func builtinFilterValues(pos Position, neg bool, args... Value) (res Value) {
        var err error
        if len(args) > 1 {
                var ( pats []Value; vals []Value )
                if pats, err = mergeresult(ExpandAll(args[0]))    ; err != nil { diag.errorAt(pos, "%v", err); return }
                if vals, err = mergeresult(ExpandAll(args[1:]...)); err != nil { diag.errorAt(pos, "%v", err); return }
                if vals, err = filterValues(pats, neg, vals...); err == nil { res = MakeListOrScalar(pos, vals) }
        }
        if res == nil && err == nil { res = MakeNone(pos) }
        return
}

func builtinSubstring(pos Position, args... Value) (res Value) {
        var err error
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "%v", err)
                return
        }

        var list []Value
        if n := len(args); n > 1 {
                var ( i1, i2 int )
                if i1, err = intVal(args[0], -1); err != nil {
                        diag.errorAt(pos, "%v", err)
                        return
                } else {
                        args = args[1:]
                }
                if i2, err = intVal(args[0], -1); err != nil {
                        if _, ok := err.(*strconv.NumError); ok {
                                err = nil // ignore
                        } else {
                                diag.errorOf(args[0], "%v", err)
                                return
                        }
                } else { args = args[1:] }

                if i1 < -1 && i2 < -1 {
                        diag.errorAt(pos, "wrong indices (%d, %d)", i1, i2)
                        return
                } else if i1 > i2 { t := i1; i1 = i2; i2 = t } // swap the wrong order
                
                var a, b = int(i1), int(i2)
                if a == -1 { a = b }
                if a == -1 { return }

                for _, arg := range args {
                        var s string
                        if s, err = arg.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        if i := len(s); i <= a { s = "" } else
                        if b == -1 || i <= b { s = s[a:b] } else { s = s[a:] }
                        list = append(list, MakeString(pos, s))
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

// $(subst from,to,text)
func builtinSubst(pos Position, args... Value) (res Value) {
        var err error
        var list []Value
        if nargs := len(args); nargs > 2 {
                var s, s1, s2 string
                if s1, err = args[0].Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                if s2, err = args[1].Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                var a []Value
                if a, err = mergeresult(Reveal(args[2:]...)); err != nil { diag.errorAt(pos, "%v", err); return }
                for _, arg := range a {
                        if s, err = arg.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        list = append(list, MakeString(pos, strings.Replace(s, s1, s2, -1)))
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

type builtinPatsubstOpts struct {
        full bool `full,fullname`
        fullfiles bool `ff,fullfile;ff,fullfiles`
        files bool `f,file;fs,files`
        cleanPath bool `c,clean;c,cleanpath`
        noFileMap bool `n,no-filemap`
}

// $(patsubst pattern,replacement,text)
// TODO:
//   $(var:pattern=replacement)
//   $(var:suffix=replacement)
func builtinPatsubst(pos Position, args... Value) (res Value) {
        var list []Value
        if len(args) < 3 { return }

        var (
                proj = current()
                opts builtinPatsubstOpts
                arg0 []Value
                err error
        )
        if proj == nil {
                diag.errorAt(pos, "unknown current context").
                        debug(optionDebugErrors,1)
                return
        }
        if arg0, err = mergeresult(ExpandAll(args[0])); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).
                        debug(optionDebugErrors,1)
                return
        }
        if arg0, err = parseOpts(pos, &opts, arg0...) ; err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err).
                        debug(optionDebugErrors,1)
                return
        }

        const infos = false
        //var infos = proj.name == "headers"

        // TODO: support flags -name and -full for name-only and full-name-only matching
        var srcPats, dstPats, sources []Value
        if len(arg0) > 0 {
                srcPats = arg0
                if dstPats, err = mergeresult(ExpandAll(args[1]))    ; err != nil { diag.errorAt(pos, "%v", err); return }
                if sources, err = mergeresult(ExpandAll(args[2:]...)); err != nil { diag.errorAt(pos, "%v", err); return }
                if infos {
                        diag.infoAt(pos, "src: %v", srcPats)
                        diag.infoAt(pos, "dst: %v", dstPats)
                        diag.infoAt(pos, "%v", sources).
                                debug(optionDebugErrors,1)
                }
        } else {
                if srcPats, err = mergeresult(ExpandAll(args[1]))    ; err != nil { diag.errorAt(pos, "%v", err); return }
                if dstPats, err = mergeresult(ExpandAll(args[2]))    ; err != nil { diag.errorAt(pos, "%v", err); return }
                if sources, err = mergeresult(ExpandAll(args[3:]...)); err != nil { diag.errorAt(pos, "%v", err); return }
                if infos {
                        diag.infoAt(pos, "src: %v", srcPats)
                        diag.infoAt(pos, "dst: %v", dstPats)
                        diag.infoAt(pos, "%v", sources).
                                debug(optionDebugErrors,1)
                }
        }

        // Using the most derived context for correct &(...)
        defer setclosure(setclosure(cloctx.unshift(proj.scope)))

        var filemaps []*FileMap
        if !opts.noFileMap { filemaps = proj.filemaps(false) }

ForSources:
        for _, src := range sources {
                var source interface{} = src
                if opts.files || opts.fullfiles {
                        var s string
                        if file, ok := src.(*File); ok {
                                source = file
                        } else if s, err = src.Strval(); err != nil {
                                diag.errorOf(src, "strval '%v' failed: %v", src, err)
                                diag.errorAt(pos, "called from here", src).debug(optionDebugErrors, 1)
                                return
                        } else if file = proj.FindFile(s); file != nil {
                                if (opts.full || opts.fullfiles) && !filepath.IsAbs(file.name) {
                                        if !file.change("", "", file.fullname()) {
                                                diag.warnAt(pos, "changing fullname failed: %v", file).
                                                        debug(optionDebugErrors, 1)
                                        }
                                }
                                source = file
                        }
                } else if opts.full {
                        var ( s string; ok bool )
                        if _, s, ok, err = asOptFullname(proj, src); err != nil {
                                diag.errorOf(src, "fullname '%v' failed: %v", src, err)
                                diag.errorAt(pos, "called from here", src).debug(optionDebugErrors, 1)
                                return
                        } else if s == "" {
                                diag.errorOf(src, "fullname '%v' is empty", src)
                                diag.errorAt(pos, "called from here", src).debug(optionDebugErrors, 1)
                                return
                        } else if !ok {
                                diag.errorOf(src, "fullname '%v' failed", src)
                                diag.errorAt(pos, "called from here", src).debug(optionDebugErrors, 1)
                                return
                        } else {
                                source = s
                        }
                }

                var ( matched bool; str string; stems []string )
        ForSrcPats:
                for _, elem := range srcPats {
                        if matched, str, stems = elem.match(source); matched {
                                break ForSrcPats
                        } else if infos {
                                diag.infoAt(pos, "source=%v (%T) elem=%v (%T) str=%s stems=%v",
                                        source, source, elem, elem, str, stems).debug(true,1)
                        }
                }
                if !matched {
                        // Just return the src if no matching.
                        if !(isNil(src) || isNone(src)) { list = append(list, src) }
                        continue ForSources
                }

                // Compose the matched results with stem value.
        ForDstPats:
                for _, dst := range dstPats {
                        var name, rest = dst.stencil(stems)
                        if name == "" || len(rest) > 0 {
                                continue ForDstPats
                        } else if opts.cleanPath {
                                name = filepath.Clean(name)
                        }

                        // Deal with special source value
                        var pos = dst.Position()
                        switch t := src.(type) {
                        case *File:
                                var (
                                        pre string
                                        match *FileMap
                                )
                                for _, m := range filemaps {
                                        if ok, s := m.Match(name); ok {
                                                match, pre = m, s
                                                break
                                        }
                                }

                                var file *File
                                if match != nil {
                                        if file = match.stat(t.dir, pre, name); file != nil {
                                                assert(file.name == name, fmt.Sprintf("invalid file name: %s != %s (t.dir=%s, pre=%s)", file.name, name, t.dir, pre))
                                        } else if file = match.stat(proj.absPath, pre, name); file != nil {
                                                assert(file.name == name, fmt.Sprintf("invalid file name: %s != %s (proj.absPath=%s, pre=%s)", file.name, name, proj.absPath, pre))
                                        } /* else if match.Paths != nil {
                                                var ( path = match.Paths[0] ; sub string )
                                                if sub, err = path.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                                if filepath.IsAbs(sub) {
                                                        file = stat(name, "", sub, nil)
                                                } else {
                                                        file = stat(name, sub, t.dir, nil)
                                                }
                                        } */
                                }
                                if file == nil {
                                        file = stat(pos, name, t.sub, t.dir, nil/* okay missing */)
                                }

                                list = append(list, file)
                                continue ForDstPats

                        default:
                                list = append(list, MakeString(pos, name))
                                continue ForDstPats
                        }
                }
        }

        res = MakeListOrScalar(pos, list)
        return
}

func builtinStrip(pos Position, args... Value) (res Value) {
        return builtinTrimSpace(pos, args...)
}

func builtinTrimSpace(pos Position, args... Value) (res Value) {
        return builtinTrim(pos, append([]Value{MakeNone(pos)}, args...)...)
}

func builtinTitle(pos Position, args... Value) (res Value) {
        var err error
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err)
                return
        }

        var (
                list []Value
                s string
        )
        for _, a := range args {
                if s, err = a.Strval(); err != nil {
                        diag.errorOf(a, "stringify '%v' failed: %v", a, err)
                        return
                } else if s != "" {
                        list = append(list, MakeString(a.Position(), strings.Title(s)))
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinUpperCase(pos Position, args... Value) (res Value) {
        var err error
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err)
                return
        }

        var (
                list []Value
                s string
        )
        for _, a := range args {
                if s, err = a.Strval(); err != nil {
                        diag.errorOf(a, "stringify '%v' failed: %v", a, err)
                        return
                } else if s != "" {
                        list = append(list, MakeString(a.Position(), strings.ToUpper(s)))
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinLowerCase(pos Position, args... Value) (res Value) {
        var err error
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err)
                return
        }

        var (
                list []Value
                s string
        )
        for _, a := range args {
                if s, err = a.Strval(); err != nil {
                        diag.errorOf(a, "stringify '%v' failed: %v", a, err)
                        return
                } else if s != "" {
                        list = append(list, MakeString(a.Position(), strings.ToLower(s)))
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinTrim(pos Position, args... Value) (res Value) {
        var err error
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return }

        var (
                list []Value
                cutset, s string
        )
        for i, a := range args {
                if s, err = a.Strval(); err != nil {
                        diag.errorAt(pos, "%v", err); return
                } else if s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, MakeString(pos, strings.TrimSpace(s)))
                        } else {
                                list = append(list, MakeString(pos, strings.Trim(s, cutset)))
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinTrimLeft(pos Position, args... Value) (res Value) {
        var err error
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return }

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
                                list = append(list, MakeString(a.Position(), strings.TrimLeftFunc(s, unicode.IsSpace)))
                        } else {
                                list = append(list, MakeString(a.Position(), strings.TrimLeft(s, cutset)))
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinTrimRight(pos Position, args... Value) (res Value) {
        var err error
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return }

        var (
                list []Value
                cutset, s string
        )
        for i, a := range args {
                if s, err = a.Strval(); err != nil {
                        diag.errorAt(pos, "%v", err); return
                } else if s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, MakeString(a.Position(), strings.TrimRightFunc(s, unicode.IsSpace)))
                        } else {
                                list = append(list, MakeString(a.Position(), strings.TrimRight(s, cutset)))
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
func builtinTrimPrefix(pos Position, args... Value) (res Value) {
        const info = false
        var (
                prefixs, values, list []Value
                err error
        )
        if len(args) == 0 { return } else
        if prefixs, err = mergeresult(ExpandAll(args[0])); err != nil {
                diag.errorOf(args[0], "merge args '%v' failed: %v", args[0], err)
                return
        }
        if len(args) == 1 {
                if len(prefixs) > 1 { values = prefixs[1:] }
        } else if values, err = mergeresult(ExpandAll(args[1:]...)); err != nil {
                diag.errorOf(args[1], "merge args '%v' failed: %v", args[1:], err)
                return
        }
        if len(values) == 0 { return } else if len(prefixs) == 0 {
                res = MakeListOrScalar(pos, values)
                return
        }
        for _, value := range values {
                var (
                        pos = value.Position()
                        s string
                )
                if s, err = value.Strval(); err != nil {
                        diag.errorOf(value, "strval '%v' failed: %v", value, err)
                        return
                }
        ForPrefix:
                for _, prefix := range prefixs {
                        var full, cutset, stems = prefix.match(value)
                        if info { diag.warnOf(prefix, "prefix=%v (%T); value=%v (%T) -> full=%v cutset=%v stems=%v",
                                prefix, prefix, value, value, full, cutset, stems).debug(true, 1) }
                        if s != "" && (cutset == "" || strings.HasPrefix(s, cutset)) {
                                if cutset == "" {
                                        s = strings.TrimLeftFunc(s, unicode.IsSpace)
                                } else {
                                        s = strings.TrimPrefix(s, cutset)
                                }
                                pos = prefix.Position()
                                break ForPrefix
                        }
                }
                if info { diag.warnAt(pos, "list=%v trimmed=%v", list, s).debug(true, 1) }
                if s != "" { list = append(list, MakeString(pos, s)) }
        }
        if err == nil { res = MakeListOrScalar(pos, list) }
        return
}

func builtinTrimSuffix(pos Position, args... Value) (res Value) {
        var err error
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return }

        var (
                list []Value
                cutset, s string
        )
        for i, a := range args {
                if s, err = a.Strval(); err != nil {
                        diag.errorAt(pos, "%v", err); return
                } else if s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, MakeString(a.Position(), strings.TrimRightFunc(s, unicode.IsSpace)))
                        } else {
                                list = append(list, MakeString(a.Position(), strings.TrimSuffix(s, cutset)))
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinTrimExt(pos Position, args... Value) (res Value) {
        var err error
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return }

        var list []Value
        for i, a := range args {
                /*switch _ := a.(type) {
                case *File:
                        fmt.Fprintf(stderr, "todo: trim-ext File{%v %v %v}\n", t.dir, t.sub, t.name)
                }*/
                var ext, s string
                if s, err = a.Strval(); err != nil {
                        diag.errorAt(pos, "%v", err); return
                } else if s != "" {
                        if i == 0 && len(args) > 1 {
                                ext = s
                        } else if ext == "" {
                                list = append(list, MakeString(a.Position(), strings.TrimSuffix(s, filepath.Ext(s))))
                        } else if ext == filepath.Ext(s) {
                                list = append(list, MakeString(a.Position(), strings.TrimRight(s, ext)))
                        }
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, list)
        }
        return
}

func builtinIndent(pos Position, args... Value) (res Value) {
        var (
                l []Value
                s string // indent
                err error
        )
        if x := len(args); x > 0 {
                if v, ok := Scalar(args[0]).(*Int); ok {
                        args, s = args[1:], strings.Repeat(" ", int(v.int64))
                } else {
                        diag.errorAt(pos, "requires integer argument (first|last)")
                        return
                }
        }
        for _, a := range args {
                var (
                        lines []string
                        v string
                )
                if v, err = a.Strval(); err != nil {
                        diag.errorAt(pos, "%v", err); return
                }
                for _, line := range strings.Split(v, "\n") {
                        lines = append(lines, s + line)
                }
                l = append(l, MakeString(a.Position(), strings.Join(lines, "\n")))
        }
        res = MakeListOrScalar(pos, l)
        return
}

func builtinFindstring(pos Position, args... Value) (res Value) {
        // TODO: $(findstring find,text)
        return
}

// $(contains a b c, v1 v2 …)
// $(contains a b c1 -or c2, v1 v2 …)
// $(contains a b c1 -or c2 -or c3, v1 v2 …)
// $(contains a b -or=(c1 c2 c3), v1 v2 …)
type builtinContainsOpts struct {
        string bool `s,string`
        verbose bool `v,verbose`
}
func builtinContains(pos Position, args... Value) (res Value) {
        if len(args) < 2 {
                diag.errorAt(pos, "unexpected number of arguments, try $(contains a b c1 -or c2, v1 v2 …)")
                return
        }

        var (
                opts builtinContainsOpts
                vals []Value
                list []Value
                err error
        )
        if vals, err = mergeresult(ExpandAll(args[0])); err != nil { diag.errorAt(pos, "%v", err); return }
        if vals, err = parseOpts(pos, &opts, vals...); err != nil { diag.errorAt(pos, "%v", err); return }
        if list, err = mergeresult(ExpandAll(args[1:]...)); err != nil { diag.errorAt(pos, "%v", err); return }

        var ( n = 0; x = len(vals); va []Value )
        for _, val := range vals {
                var s string
                switch v := val.(type) {
                default: va = []Value{ val }
                case *Flag:
                        if s, err = v.name.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        if s == "or" { va, x = append(va, val), x-1; continue }
                case *Pair: // FIXME: -or=(c1 c2 c3)
                        if f, ok := v.Key.(*Flag); !ok {va = []Value{ val }} else {
                                if s, err = f.name.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                if s == "or" { va, x = append(va, v.Value), x-1; continue }
                        }
                }

                if len(va) == 0 { continue }
                ForList:for _, v := range list {
                        for _, a := range va {
                                if opts.string {
                                        var r string
                                        if r, err = v.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                        if s, err = a.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                        if r != s { continue ForList }
                                } else if a.cmp(v) != cmpEqual { continue ForList }
                        }
                        n += 1 // one matched
                }
                va = nil
        }
        if opts.verbose {
                diag.infoAt(pos, "%v contains %v: %v (%v, %v)\n", list, vals, (n==x), n, x)
        }
        res = &boolean{valbase{pos},(n == x)}
        return
}

func builtinFilter(pos Position, args... Value) (res Value) {
        // $(filter pattern…,text)
        res = builtinFilterValues(pos, false, args...)
        return
}

func builtinFilterOut(pos Position, args... Value) (res Value) {
        // $(filter-out pattern…,text)
        res = builtinFilterValues(pos, true, args...)
        return
}

func builtinSort(pos Position, args... Value) (res Value) {
        // TODO: $(sort list)
        return
}

func builtinWord(pos Position, args... Value) (res Value) {
        // TODO: $(word n,text)
        return
}

func builtinWordList(pos Position, args... Value) (res Value) {
        // TODO: $(wordlist s,e,text)
        return
}

func builtinWords(pos Position, args... Value) (res Value) {
        // TODO: $(words n,text)
        return
}

func builtinFirstWord(pos Position, args... Value) (res Value) {
        // TODO: $(firstword names...)
        return
}

func builtinLastWord(pos Position, args... Value) (res Value) {
        // TODO: $(lastword names...)
        return
}

func builtinEncodeBase64(pos Position, args... Value) (res Value) {
        if len(args) > 0 {
                buf := new(bytes.Buffer)
                enc := base64.NewEncoder(base64.StdEncoding, buf)
                for _, a := range args {
                        var ( s string; err error )
                        if s, err = a.Strval(); err != nil {
                                diag.errorAt(pos, "%v", err); return
                        }
                        enc.Write([]byte(s))
                }
                enc.Close()
                res = MakeString(pos, buf.String())
        }
        return
}

func builtinDecodeBase64(pos Position, args... Value) (res Value) {
        if len(args) > 0 {
                var list []Value
                for _, a := range args {
                        var (
                                dat []byte
                                s string
                                err error
                        )
                        if s, err = a.Strval(); err != nil {
                                diag.errorAt(pos, "%v", err); return
                        }
                        dat, err = base64.StdEncoding.DecodeString(s)
                        if err == nil {
                                list = append(list, MakeString(a.Position(), string(dat)))
                        } else {
                                diag.errorAt(pos, "%v", err); return
                        }
                }
                res = MakeListOrScalar(pos, list)
        }
        return
}

func fullname(a Value) (s string, ok bool) {
        var f *File
        switch t := a.(type) {
        case *File     : f = t
        case *Barefile : f = t.File
        case *RuleEntry:               return fullname(t.target)
        case *Def: if t.value != nil { return fullname(t.value ) }
        }
        if f != nil && (f.dir != "" || f.sub != "") {
                s, ok = f.fullname(), true
        }
        return
}

func fullnameOrStrval(a Value) (s string, err error) {
        var ok bool
        if s, ok = fullname(a); !ok {
                s, err = a.Strval()
        }
        return
}

// see optFullname and parseOpt
func asOptFullname(proj *Project, val Value) (rp *Project, s string, ok bool, e error) {
        if proj == nil { proj = current() }
        if s, ok = fullname(val); ok {
                // done
        } else if proj == nil {
                diag.errorOf(val, "no current project to find file '%v'", val).
                        debug(optionDebugErrors,1)
        } else if s, e = val.Strval(); e != nil {
                diag.errorOf(val, "no current project to find file '%v'", val).
                        debug(optionDebugErrors,1)
        } else if filepath.IsAbs(s) {
                ok = true
        } else if file := proj.FindFile(s); file != nil {
                s, ok = file.fullname(), true
        }
        rp = proj
        return
}

type builtinFullnameOpts struct {
        debug int `d,debug`
}
func builtinFullname(pos Position, args... Value) (res Value) {
        var (
                opts builtinFullnameOpts
                proj *Project
                l []Value
                err error
                s string
                ok bool
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err)
                return
        } else if args, err = parseOpts(pos, &opts, args...) ; err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err)
                return
        }

        for _, a := range args {
                if opts.debug > 0 {
                        if f, ok := a.(*File); ok {
                                diag.warnAt(pos, "dir=%v sub=%v name=%v", f.dir, f.sub, f.name).
                                        debug(optionDebugErrors, opts.debug)
                        } else {
                                diag.warnAt(pos, "%T %v", a, a).
                                        debug(optionDebugErrors, opts.debug)
                        }
                }
                if proj, s, ok, err = asOptFullname(proj, a); err != nil {
                        diag.errorAt(pos, "fullname '%v' failed: %v", a, err)
                        break
                } else if ok || s != "" {
                        l = append(l, MakeString(a.Position(), s))
                } else {
                        l = append(l, a)
                }
        }
        res = MakeListOrScalar(pos, l)
        return
}

type builtinBaseOpts struct {
        debug int `d,debug`
        fullname bool `f,full;fn,fullname` // unused
}
func builtinBase(pos Position, args... Value) (res Value) {
        var (
                opts builtinBaseOpts
                err error
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err)
                return
        } else if args, err = parseOpts(pos, &opts, args...) ; err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err)
                return
        }

        var l []Value
        for _, a := range args {
                var s string
                if s, err = fullnameOrStrval(a); err != nil {
                        diag.errorAt(pos, "%v", err); return
                }
                l = append(l, MakeString(pos, filepath.Base(s)))
        }
        res = MakeListOrScalar(pos, l)
        return
}

func dirx(pos Position, n int, args... Value) (res Value) {
        var (
                l []Value
                s string
                err error
        )
        for _, a := range args {
                if s, err = fullnameOrStrval(a); err != nil {
                        diag.errorAt(pos, "%v", err); return
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

func undirx(pos Position, n int, args... Value) (res Value) {
        var (
                l []Value
                s string
                err error
        )
        for _, a := range args {
                if s, err = fullnameOrStrval(a); err != nil {
                        diag.errorAt(pos, "%v", err); return
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

type builtinDirOpts struct {
        fullname bool `f,full;fn,fullname`
}
func builtinDir(pos Position, args... Value) (res Value) {
        var (
                opts builtinDirOpts
                proj *Project
                l []Value
                err error
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err)
                return
        } else if args, err = parseOpts(pos, &opts, args...) ; err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err)
                return
        }
        for _, a := range args {
                var s string
                if opts.fullname {
                        if proj, s, _, err = asOptFullname(proj, a); err != nil {
                                diag.errorAt(pos, "fullname '%v' failed: %v", a, err)
                                break
                        }
                }
                if !opts.fullname || s == "" {
                        if s, err = a.Strval(); err != nil {
                                diag.errorAt(pos, "strval '%v' failed: %v", a, err)
                                return
                        }
                }
                l = append(l, MakePathStr(pos,filepath.Dir(s)))
        }
        res = MakeListOrScalar(pos, l)
        return
}

func builtinDir2(pos Position, args... Value) (res Value) {
        return dirx(pos, 2, args...)
}

func builtinDir3(pos Position, args... Value) (res Value) {
        return dirx(pos, 3, args...)
}

func builtinDir4(pos Position, args... Value) (res Value) {
        return dirx(pos, 4, args...)
}

func builtinDir5(pos Position, args... Value) (res Value) {
        return dirx(pos, 5, args...)
}

func builtinDir6(pos Position, args... Value) (res Value) {
        return dirx(pos, 6, args...)
}

func builtinDir7(pos Position, args... Value) (res Value) {
        return dirx(pos, 7, args...)
}

func builtinDir8(pos Position, args... Value) (res Value) {
        return dirx(pos, 8, args...)
}

func builtinDir9(pos Position, args... Value) (res Value) {
        return dirx(pos, 9, args...)
}

func builtinDirs(pos Position, args... Value) (res Value) {
        var n int
        if x := len(args); x > 0 {
                if v, ok := Scalar(args[0]).(*Int); ok {
                        args, n = args[1:], int(v.int64)
                } else if v, ok := Scalar(args[x-1]).(*Int); ok {
                        args, n = args[:x-1], int(v.int64)
                } else {
                        diag.errorAt(pos, "require (first/last) integer argument (first=%T, last=%T)", args[0], args[x-1])
                        return
                }
        }
        res = dirx(pos, n, args...)
        return
}

func builtinUndir(pos Position, args... Value) (res Value) {
        return undirx(pos, 1, args...)
}

func builtinUndir2(pos Position, args... Value) (res Value) {
        return undirx(pos, 2, args...)
}

func builtinUndir3(pos Position, args... Value) (res Value) {
        return undirx(pos, 3, args...)
}

func builtinUndir4(pos Position, args... Value) (res Value) {
        return undirx(pos, 4, args...)
}

func builtinUndir5(pos Position, args... Value) (res Value) {
        return undirx(pos, 5, args...)
}

func builtinUndir6(pos Position, args... Value) (res Value) {
        return undirx(pos, 6, args...)
}

func builtinUndir7(pos Position, args... Value) (res Value) {
        return undirx(pos, 7, args...)
}

func builtinUndir8(pos Position, args... Value) (res Value) {
        return undirx(pos, 8, args...)
}

func builtinUndir9(pos Position, args... Value) (res Value) {
        return undirx(pos, 9, args...)
}

func builtinUndirs(pos Position, args... Value) (res Value) {
        var n = 0
        if x := len(args); x > 0 {
                if v, ok := Scalar(args[0]).(*Int); ok {
                        args, n = args[1:], int(v.int64)
                } else if v, ok := Scalar(args[x-1]).(*Int); ok {
                        args, n = args[:x-1], int(v.int64)
                } else {
                        diag.errorAt(pos, "require (first/last) integer argument (first=%T, last=%T)", args[0], args[x-1])
                        return
                }
        }
        return undirx(pos, n, args...)
}

func builtinDirChop(pos Position, args... Value) (res Value) {
        var (
                err error
                l []Value
                n = 0
        )
        if x := len(args); x > 0 {
                if v, ok := Scalar(args[0]).(*Int); ok {
                        args, n = args[1:], int(v.int64)
                } else if v, ok := Scalar(args[x-1]).(*Int); ok {
                        args, n = args[:x-1], int(v.int64)
                } else {
                        diag.errorAt(pos, "require (first/last) integer argument (first=%T, last=%T)", args[0], args[x-1])
                        return

                }
        }
        for _, a := range args {
                var s string
                if s, err = a.Strval(); err != nil {
                        diag.errorAt(pos, "%v", err); return
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
                l = append(l, MakeString(a.Position(), filepath.Join(v...)))
        }
        res = MakeListOrScalar(pos, l)
        return
}

func builtinRelativeDir(pos Position, args... Value) (res Value) {
        var (
                err error
                l []Value
                t, s string
        )
        for i, a := range args {
                if s, err = a.Strval(); err != nil {
                        diag.errorAt(pos, "%v", err)
                        return
                }
                if i == 0 {
                        t = s
                } else if s, err = filepath.Rel(t, s); err == nil {
                        l = append(l, MakeString(a.Position(), s))
                } else {
                        diag.errorAt(pos, "%v", err)
                        return
                }
        }
        res = MakeListOrScalar(pos, l)
        return
}

func builtinMkdir(pos Position, args... Value) (res Value) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name string
                        perm os.FileMode
                        err error
                )
                switch t := a.(type) {
                case *Pair: // mkdir name => perm name => perm
                        if name, err = t.Key.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        if perm, err = permVal(t.Value,0600); err != nil { diag.errorAt(pos, "%v", err); return }
                case *Group: // mkdir (name perm) (name perm)
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                if perm, err = permVal(t.Get(1),0600); err != nil { diag.errorAt(pos, "%v", err); return }
                        } else {
                                diag.errorAt(pos, "Wrong size of list `%v'", t)
                                break
                        }
                case *List: // mkdir name perm, name perm, ...
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                if perm, err = permVal(t.Get(1),0600); err != nil { diag.errorAt(pos, "%v", err); return }
                        } else {
                                diag.errorAt(pos, "Wrong size of list `%v'", t)
                                break
                        }
                default: // mkdir name perm, name perm, ...
                        if name, err = args[i].Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        if i+1 < nargs {
                                if perm, err = permVal(args[i+1],0600); err != nil { diag.errorAt(pos, "%v", err); return }
                                i += 1
                        }
                }
                if err = os.Mkdir(name, perm); err != nil { diag.errorAt(pos, "%v", err); break }
        }
        return
}

func builtinMkdirAll(pos Position, args... Value) (res Value) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name string
                        perm os.FileMode
                        err error
                )
                switch t := a.(type) {
                case *Pair: // mkdir name => perm name => perm
                        if name, err = t.Key.Strval(); err != nil { diag.errorOf(t.Key, "%v", err); return }
                        if perm, err = permVal(t.Value,0600); err != nil { diag.errorOf(t.Value, "%v", err); return }
                case *Group: // mkdir (name perm) (name perm)
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { diag.errorOf(t.Get(0), "%v", err); return }
                                if perm, err = permVal(t.Get(1),0600); err != nil { diag.errorOf(t.Get(1), "%v", err); return }
                        } else {
                                diag.errorAt(pos, "Wrong size of list `%v'", t);
                                break
                        }
                case *List: // mkdir name perm, name perm, ...
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { diag.errorOf(t.Get(0), "%v", err); return }
                                if perm, err = permVal(t.Get(1),0600); err != nil { diag.errorOf(t.Get(1), "%v", err); return }
                        } else {
                                diag.errorAt(pos, "Wrong size of list `%v'", t);
                                break
                        }
                default: // mkdir name perm, name perm, ...
                        if name, err = args[i].Strval(); err != nil { diag.errorOf(args[i], "%v", err); return }
                        if i+1 < nargs {
                                if perm, err = permVal(args[i+1],0600); err != nil { diag.errorOf(args[i+1], "%v", err); return }
                                i += 1
                        }
                }
                if err = os.MkdirAll(name, perm); err != nil { diag.errorAt(pos, "%v", err); break }
        }
        return
}

func builtinChdir(pos Position, args... Value) (res Value) {
        if len(args) == 1 {
                var ( str string; err error )
                if str, err = args[0].Strval(); err != nil { diag.errorOf(args[0], "%v", err); return }
                if err = lockCD(str, 0); err != nil { diag.errorAt(pos, "%v", err) }
        } else {
                diag.errorAt(pos, "wrong number of arguments: %v", len(args))
        }
        return
}

type builtinRenameOpts struct {
        // TODO: ...
}
func builtinRename(pos Position, args... Value) (res Value) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        oldname, newname string
                        err error
                )
                switch t := a.(type) {
                case *Pair: // rename oldname=newname
                        if oldname, err = t.Key.Strval();   err != nil { diag.errorOf(t.Key, "%v", err); return }
                        if newname, err = t.Value.Strval(); err != nil { diag.errorOf(t.Value, "%v", err); return }
                case *Group: // rename (oldname newname) (old new)
                        if t.Len() == 2 {
                                if oldname, err = t.Get(0).Strval(); err != nil { diag.errorOf(t.Get(0), "%v", err); return }
                                if newname, err = t.Get(1).Strval(); err != nil { diag.errorOf(t.Get(1), "%v", err); return }
                        } else {
                                diag.errorOf(t, "wrong size of group `%v'", t)
                                break
                        }
                case *List: // rename oldname newname, old new, ...
                        if t.Len() == 2 {
                                if oldname, err = t.Get(0).Strval(); err != nil { diag.errorOf(t.Get(0), "%v", err); return }
                                if newname, err = t.Get(1).Strval(); err != nil { diag.errorOf(t.Get(1), "%v", err); return }
                        } else {
                                diag.errorOf(t, "wrong size of list `%v'", t)
                                break
                        }
                default: // rename newname oldname  newname oldname ...
                        if i+1 < nargs {
                                if oldname, err = args[i+0].Strval(); err != nil { diag.errorOf(args[i+0], "%v", err); return }
                                if newname, err = args[i+1].Strval(); err != nil { diag.errorOf(args[i+1], "%v", err); return }
                                i += 1
                        } else {
                                diag.errorOf(t, "Wrong arguments `%v'", args)
                                break
                        }
                }
                if err = os.Rename(oldname, newname); err != nil {
                        diag.errorAt(pos, "%v", err)
                        break
                }
        }
        return
}

type builtinRemoveOpts struct {
        all bool `a,all;r,recursive`
        debug bool `d,debug`
        verbose bool `v,verbose`
}
func builtinRemove(pos Position, args... Value) (res Value) {
        var (
                opts builtinRemoveOpts
                err error
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return } else
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }

        var (
                names []string
                proj *Project
                str string
                ok bool
        )
        for _, a := range args {
                if isNil(a) || isNone(a) {
                        // ignore
                } else if a.patterned() {
                        if str, err = a.Strval(); err != nil { diag.errorOf(a, "%v", err).debug(true, 1); return }
                        if names, err = filepath.Glob(str); err != nil { diag.errorOf(a, "%v", err).debug(true, 1); return }
                        for _, s := range names {
                                if opts.verbose { diag.prompt("remove %s\n", s) }
                                if opts.debug   { diag.infoAt(pos, "remove %s", s).debug(optionDebugErrors, 1) }
                                if opts.all {
                                        err = os.RemoveAll(s)
                                } else {
                                        err = os.Remove(s)
                                }
                                if err != nil {
                                        diag.errorOf(a, "remove failed: %v", err)
                                        return
                                }
                        }
                } else if proj, str, ok, err = asOptFullname(proj, a); err != nil {
                        diag.errorOf(a, "fullname '%v' failed: %v", a, err)
                        diag.errorAt(pos, "internal stack:").
                                debug(optionDebugErrors, 16)
                        return
                } else if !ok || str == "" {
                        diag.errorOf(a, "remove failed: %v (%T)", a, a)
                        diag.errorOf(a, "remove failed: %v", str)
                        diag.errorAt(pos, "internal stack:").
                                debug(optionDebugErrors, 16)
                        break
                } else {
                        if opts.verbose { diag.prompt("remove %s\n", str) }
                        if opts.debug   { diag.infoAt(pos, "remove %s", str).debug(optionDebugErrors, 1) }
                        if opts.all {
                                err = os.RemoveAll(str)
                        } else {
                                err = os.Remove(str)
                        }
                        if err != nil {
                                diag.errorAt(pos, "%v", err)
                                diag.errorOf(a, "source: %v (%T)", a, a)
                                diag.errorOf(a, "source: %v", str).debug(optionDebugErrors, 1)
                                return
                        }
                }
        }
        return
}

type builtinRemoveAllOpts struct {
        debug bool `d,debug`
        verbose bool `v,verbose`
}
func builtinRemoveAll(pos Position, args... Value) (res Value) {
        var (
                opts builtinRemoveAllOpts
                err error
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return }
        if args, err = parseOpts(pos, &opts, args...); err != nil { diag.errorAt(pos, "%v", err); return }

        var (
                names []string
                proj *Project
                str string
                ok bool
        )
        for _, a := range args {
                if a.patterned() {
                        if str, err = a.Strval(); err != nil { diag.errorOf(a, "%v", err); return }
                        if names, err = filepath.Glob(str); err != nil { diag.errorOf(a, "%v", err); return }
                        for _, s := range names {
                                if opts.verbose { diag.infoAt(a.Position(), "remove %s", s) }
                                if err = os.RemoveAll(s); err != nil { diag.errorOf(a, "%v", err); return }
                        }
                } else if proj, str, ok, err = asOptFullname(proj, a); err != nil {
                        diag.errorOf(a, "remove failed: %v", err)
                        return
                } else if !ok || str == "" {
                        diag.errorOf(a, "'%v' is not a file", a)
                        break
                } else {
                        if opts.verbose { diag.infoAt(pos, "remove %s", str) }
                        if opts.debug   { diag.infoAt(pos, "remove %s", str).debug(optionDebugErrors, 1) }
                        if err = os.RemoveAll(str); err != nil {
                                diag.errorOf(a, "remove failed: %v", err)
                                return
                        }
                }
        }
        return
}

func builtinTruncate(pos Position, args... Value) (res Value) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name string
                        size int64
                        err error
                )
                switch t := a.(type) {
                case *Pair: // truncate name => size old => new
                        if name, err = t.Key.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        if size, err = t.Value.Integer(); err != nil { diag.errorAt(pos, "%v", err); return }
                case *Group: // truncate (name size) (old new)
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                if size, err = t.Get(1).Integer(); err != nil { diag.errorAt(pos, "%v", err); return }
                        } else {
                                diag.errorAt(pos, "Wrong size of group `%v'", t)
                                break
                        }
                case *List: // truncate name size, old new, ...
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                if size, err = t.Get(1).Integer(); err != nil { diag.errorAt(pos, "%v", err); return }
                        } else {
                                diag.errorAt(pos, "Wrong size of list `%v'", t)
                                break
                        }
                default: // truncate name size  name size ...
                        if i+1 < nargs {
                                if name, err = args[i+0].Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                if size, err = args[i+1].Integer(); err != nil { diag.errorAt(pos, "%v", err); return }
                                i += 1
                        } else {
                                diag.errorAt(pos, "Wrong arguments `%v'", args)
                                break
                        }
                }
                if err = os.Truncate(name, size); err != nil {
                        diag.errorAt(pos, "%v", err); break
                }
        }
        return
}

type builtinLinkOpts struct {
        // TODO: ...
}
func builtinLink(pos Position, args... Value) (res Value) {
        var (
                opts builtinLinkOpts
                err error
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return }
        if args, err = parseOpts(pos, &opts, args...); err != nil { diag.errorAt(pos, "%v", err); return }
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        oldname, newname string
                )
                switch t := a.(type) {
                case *Pair: // link oldname => newname old => new
                        if oldname, err = t.Key.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        if newname, err = t.Value.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                case *Group: // link (oldname newname) (old new)
                        if t.Len() == 2 {
                                if oldname, err = t.Get(0).Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                if newname, err = t.Get(1).Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // link oldname newname, old new, ...
                        if t.Len() == 2 {
                                if oldname, err = t.Get(0).Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                if newname, err = t.Get(1).Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // link oldname newname  oldname newname ...
                        if i+1 < nargs {
                                if oldname, err = args[i+0].Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                if newname, err = args[i+1].Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
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
type builtinSymlinkOpts struct {
        path bool `p,path`
        force bool `f,force`
        update bool `u,update`
        relative bool `r,relative;l,rel`
        verbose bool `v,verbose`
}
func builtinSymlink(pos Position, args... Value) (res Value) {
        var (
                opts builtinSymlinkOpts
                err error
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "%v", err)
                return
        } else if args, err = parseOpts(pos, &opts, args...); err != nil {
                diag.errorAt(pos, "%v", err)
                return
        }
        if false { fmt.Printf("%v: %v\n", pos, args) }
ForArgs:
        for i, na := 0, len(args); i < na; i += 1 {
                var oldNameVal, newNameVal Value
                switch t := args[i].(type) {
                case *Pair: // symlink oldname=newname oldname=>newname...
                        oldNameVal, newNameVal = t.Key, t.Value
                case *Group: // symlink (oldname newname) (oldname newname)...
                        if t.Len() != 2 {
                                diag.errorOf(t, "expects two values of group")
                                return
                        }
                        oldNameVal, newNameVal = t.Get(0), t.Get(1)
                case *List: // symlink oldname newname, old new, ...
                        if t.Len() != 2 {
                                diag.errorOf(t, "expects two values of list")
                                return
                        }
                        oldNameVal, newNameVal = t.Get(0), t.Get(1)
                default:// Multiple pairs of names:
                        // symlink  newname oldname  newname oldname ...
                        if i+1 < na {
                                oldNameVal, newNameVal = args[i+0], args[i+1]
                                i += 1
                        } else {
                                diag.errorOf(args[i], "expects pair of names (%v)", args[i])
                                return
                        }
                }

                var oldname, newname string
                if oldname, err = oldNameVal.Strval(); err != nil {
                        diag.errorOf(oldNameVal, "%v", err)
                        return
                }
                if newname, err = newNameVal.Strval(); err != nil {
                        diag.errorOf(newNameVal, "%v", err)
                        return
                }

                if newname == "" {
                        diag.errorAt(pos, "empty new filename")
                        return
                }
                if oldname == "" {
                        diag.errorAt(pos, "empty old filename (%v)", )
                        return
                }

                if opts.force {
                        if err = os.Remove(newname); err != nil {
                                diag.errorAt(pos, "%v", err)
                                err = nil //return
                        }
                } else if opts.update {
                        var s string
                        if s, err = os.Readlink(newname); err != nil {
                                diag.errorAt(pos, "%v", err)
                                err = nil //continue ForArgs
                        } else if s == newname {
                                continue ForArgs
                        } else if err = os.Remove(newname); err != nil {
                                diag.errorAt(pos, "%v", err)
                                err = nil //return
                        }
                }
                if opts.verbose {
                        var d = filepath.Base(newname)
                        var s = filepath.Base(oldname)
                        fmt.Fprintf(stderr, "smart: Symlink %s -> %s …", d, s)
                }
                if opts.relative {
                        var dir = filepath.Dir(newname)
                        oldname, err = filepath.Rel(dir, oldname)
                        if err != nil {
                                if opts.verbose {
                                        fmt.Fprintf(stderr, "symlink: %s\n", err)
                                }
                                diag.errorAt(pos, "%v", err)
                                return
                        }
                }
                if dir := filepath.Dir(newname); opts.path && dir != "." && dir != PathSep {
                        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil { diag.errorAt(pos, "%v", err); return }
                }
                if err = os.Symlink(oldname, newname); err != nil {
                        if opts.verbose {
                                fmt.Fprintf(stderr, "… %s\n", err)
                        }
                        break
                } else if opts.verbose {
                        fmt.Fprintf(stderr, "… ok\n")
                }
        }
        return
}

type builtinFileExistsOpts struct {
        dir bool `d,dir`
        file bool `f,file`
        symbol bool `s,symlink;sym,symbol`
}
func builtinFileExists(pos Position, args... Value) (res Value) {
        var (
                opts builtinFileExistsOpts
                err error
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "%v", err)
                return
        } else if args, err = parseOpts(pos, &opts, args...); err != nil {
                diag.errorAt(pos, "%v", err)
                return
        }

        var proj = current()
        if proj == nil {
                diag.errorAt(pos, "unknown current context")
                return
        }

        var reses []Value
        var check = func(file *File) {
                if file.info == nil {
                        reses = append(reses, &boolean{valbase{pos},false})
                        return
                }
                var mode = file.info.Mode()
                 if opts.dir && mode&os.ModeDir != 0 { // IsDir()
                        reses = append(reses, &boolean{valbase{pos},true})//file
                        return
                }
                if opts.symbol && mode&os.ModeSymlink != 0 {
                        reses = append(reses, &boolean{valbase{pos},true})//file
                        return
                }
                if opts.file && mode&os.ModeType != 0 { // IsRegular()
                        reses = append(reses, &boolean{valbase{pos},true})//file
                        return
                }
                reses = append(reses, &boolean{valbase{pos},true})//file
                return
        }

        var checkstat = func(a Value) {
                var ( s string ; file *File )
                if s, err = a.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                if filepath.IsAbs(s) {
                        file = stat(pos, s, "", "")
                } else {
                        file = stat(pos, s, "", proj.absPath)
                }
                if file == nil { file = proj./*searchFile*/FindFile(s) }
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

func builtinFileSource(pos Position, args... Value) (res Value) {
        var err error
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return }

        var proj = current()
        if proj == nil {
                diag.errorAt(pos, "unknown current context")
                return
        }

        var l []Value
        for _, a := range args {
                var str string
                if str, err = a.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                if file := proj./*searchFile*/FindFile(str); file != nil {
                        l = append(l, MakeString(a.Position(), file.sub))
                }
        }
        if err == nil {
                res = MakeListOrScalar(pos, l)
        }
        return
}

type builtinFileOpts struct {
        caller bool `c,caller;cc,callercontext;cc,caller-context`
        report bool `r,report;r,reportmissing;rm,report-missing;e,error`
}
func builtinFile(pos Position, args... Value) (res Value) {
        var ( opts builtinFileOpts; err error )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return }
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "%v", err); return }

        var proj *Project
        if opts.caller {
                proj = cloctx[0].project
        } else if proj = current(); proj == nil {
                diag.errorAt(pos, "unknown current cntext")
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
                        if file.exists() { continue }
                        if opts.report { fmt.Fprintf(stderr, "%s: `%v` no such file\n", pos, a) }
                } else if str, err = a.Strval(); err != nil {
                        diag.errorAt(pos, "%v", err)
                        return
                } else if file = proj.FindFile(str); file != nil {
                        list = append(list, file)
                        if opts.report { fmt.Fprintf(stderr, "%s: `%v` no such file\n", pos, a) }
                } else {
                        diag.errorAt(pos, "`%v` is not a file", a)
                }
        }

        res = MakeListOrScalar(pos, list)
        return
}

type wildcardOpts struct {
        includeMissing bool `im,includemissing;m,include-missing`
        errorMissing bool `em,errormissing;e,error-missing`
        verbose bool `v,verbose`
}
func builtinWildcard(pos Position, args... Value) (res Value) {
        var proj = current()
        if proj == nil {
                diag.errorAt(pos, "unknown most derived context").debug(optionDebugErrors,1)
                return
        }

        var (
                opts wildcardOpts
                files []*File
                err error
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(optionDebugErrors,1)
                return
        } else if args, err = parseOpts(pos, &opts, args...); err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err).debug(optionDebugErrors,1)
                return
        } else if files, err = proj.wildcard(pos, opts, args...); err == nil {
                var list []Value
                for _, f := range files { list = append(list, f) }
                res = MakeListOrScalar(pos, list)
        } else {
                diag.errorAt(pos, "wildcard failed: %v", err).debug(optionDebugErrors,1)
        }
        return
}

func builtinReadDir(pos Position, args... Value) (res Value) {
        var l []Value
        for _, a := range args {
                var (
                        fis []os.FileInfo
                        str string
                        err error
                )
                if str, err = a.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                if fis, err = ioutil.ReadDir(str); err == nil {
                        v := new(List)
                        for _, fi := range fis {
                                v.Append(MakeString(a.Position(), fi.Name()))
                        }
                        l = append(l, v)
                } else {
                        break //l = append(l, MakeNone(pos))
                }
        }
        if len(l) > 0 {
                res = MakeListOrScalar(pos, l)
        }
        return
}

type builtinReadFileOpts struct {
        trim bool `t,trim;ta,trim-all`
        trimLeft bool `tl,trim-left`
        trimRight bool `tr,trim-right`
}
func builtinReadFile(pos Position, args... Value) (res Value) {
        var (
                opts builtinReadFileOpts
                proj *Project
                err error
                l []Value
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }
        for _, a := range args {
                var (
                        apos = a.Position()
                        str string
                        err error
                        s []byte
                        ok bool
                )
                if !apos.IsValid() { apos = pos }
                if proj, str, ok, err = asOptFullname(proj, a); err != nil {
                        diag.errorAt(pos, "fullname '%v' failed: %v", a, err)
                        break
                } else if !ok || str == "" {
                        diag.errorAt(apos, "'%v' is not a file", a)
                        break
                } else if s, err = ioutil.ReadFile(str); err != nil {
                        diag.errorAt(apos, "read file failed: %v", err)
                        break
                } else {
                        if opts.trim { s = bytes.TrimFunc(s, unicode.IsSpace) } else
                        if opts.trimLeft { s = bytes.TrimLeftFunc(s, unicode.IsSpace) } else
                        if opts.trimRight { s = bytes.TrimRightFunc(s, unicode.IsSpace) }
                        l = append(l, MakeString(pos, string(s)))
                }
        }
        if len(l) > 0 {
                res = MakeListOrScalar(pos, l)
        }
        return
}

type builtinWriteFileOpts struct {
        path bool `p,path`
}
func builtinWriteFile(pos Position, args... Value) (res Value) {
        // $(write-file filename,content)
        // $(write-file -p filename,content)
        var (
                opts builtinWriteFileOpts
                va []Value
                err error
        )
        if len(args) > 0 {
                if va, err = mergeresult(ExpandAll(args[1])); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
                if va, err = parseOpts(pos, &opts, va...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }
                args = append(va, args[1:]...)
        }
ForArgs:
        for i := 0; i < len(args); i += 1 {
                var (
                        a = args[i]
                        name, data string
                        perm = os.FileMode(0600)
                )
                switch t := a.(type) {
                case *Pair: // write-file name=text name=text
                        if name, err = t.Key.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        if data, err = t.Value.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                case *Group: // write-file (name text) (name text 0660)
                        if n := t.Len(); n < 4 && n > 0 {
                                if name, err = t.Get(0).Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                if n > 1 { if data, err = t.Get(1).Strval(); err != nil { diag.errorAt(pos, "%v", err); return }}
                                if n > 2 { if perm, err = permVal(t.Get(2),0600); err != nil { diag.errorAt(pos, "%v", err); return }}
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // write-file name text, name text 0660, ...
                        if n := t.Len(); n < 4 && n > 0 {
                                if name, err = t.Get(0).Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                if n > 1 { if data, err = t.Get(1).Strval(); err != nil { diag.errorAt(pos, "%v", err); return }}
                                if n > 2 { if perm, err = permVal(t.Get(2),0600); err != nil { diag.errorAt(pos, "%v", err); return }}
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // write-file name text 0660  name text 0660 ...
                        if name, err = args[i].Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        if i+1 < len(args) {
                                if data, err = args[i+1].Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                i += 1
                        }
                        if i+1 < len(args) {
                                if perm, err = permVal(args[i+1],0600); err != nil { diag.errorAt(pos, "%v", err); return }
                                i += 1
                        }
                }
                if name == "" {
                        continue ForArgs
                } else if dir := filepath.Dir(name); opts.path && dir != "." && dir != PathSep {
                        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil { diag.errorAt(pos, "%v", err); return }
                }
                if err = ioutil.WriteFile(name, []byte(data), perm); err != nil {
                        diag.errorAt(pos, "%v", err); break
                }
        }
        return
}

func touch(file Value, optMode uint32, optPath bool, ts ...time.Time) (err error) {
        var filename, _ = fullname(file)
        if  filename == "" {
                diag.errorOf(file, "touch: file fullname of '%v' is empty", file, err).
                        debug(optionDebugErrors, 1)
                return
        }

        if dir := filepath.Dir(filename); optPath && dir != "." && dir != PathSep {
                if err = os.MkdirAll(dir, os.FileMode(optMode|0733)); err != nil {
                        diag.errorOf(file, "touch: %v", err).
                                debug(optionDebugErrors, 1)
                        return
                }
        }

        var (
                mode = os.FileMode(optMode)
                m os.FileMode
                at, mt time.Time
        )
        if len(ts) > 0 { at = ts[0] } else { at = time.Now() }
        if len(ts) > 1 { mt = ts[1] } else { mt = time.Now() }
        if fi, k := file.(*File); k && fi.info != nil {
                m = fi.info.Mode()
        } else if fi, e := os.Stat(filename); e == nil && fi != nil {
                m = fi.Mode()
        } else {
                var f *os.File
                if m = mode; m == 0 { m = os.FileMode(0600); mode = m }
                if f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, m&os.ModePerm); err != nil {
                        diag.errorOf(file, "touch: %v", err).
                                debug(optionDebugErrors, 1)
                } else if err = f.Close(); err != nil {
                        diag.errorOf(file, "touch: %v", err).
                                debug(optionDebugErrors, 1)
                }
        }
        if err == nil {
                if err = os.Chtimes(filename, at, mt); err != nil {
                        diag.errorOf(file, "touch: %v", err).
                                debug(optionDebugErrors, 1)
                }
        }
        if err == nil && mode != 0 && m != 0 && mode != m {
                if err = os.Chmod(filename, mode); err != nil {
                        diag.errorOf(file, "touch: %v", err).
                                debug(optionDebugErrors, 1)
                }
        }
        return
}

type builtinTouchFileOpts struct {
        path bool `p,path`
        mode os.FileMode `m,mode;fm,filemode;fm,file-mode`
}
func builtinTouchFile(pos Position, args... Value) (res Value) {
        // $(touch-file filename)
        // $(touch-file -p filename)
        var (
                opts = builtinTouchFileOpts{ mode: os.FileMode(0600) }
                err error
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return }
        if args, err = parseOpts(pos, &opts, args...); err != nil { diag.errorAt(pos, "%v", err); return }
        for i := 0; i < len(args); i += 1 {
                err = touch(args[i], uint32(opts.mode), opts.path)
                if err != nil { diag.errorAt(pos, "%v", err); break }
        }
        return
}

// $(grep 'status=1',$@)
// $(grep -1 'status=1',$@)
func builtinGrep(pos Position, args... Value) (res Value) {
        if len(args) != 2 {
                diag.errorAt(pos, "wants exactly 2 args, e.g. $(grep -1 '^example$',$(file))")
                return
        }

        var err error
        var vals, list []Value
        var linesPos, linesNeg []int
        var rxs []*regexp.Regexp
        
        if vals, err = mergeresult(ExpandAll(args[0])); err != nil { diag.errorAt(pos, "%v", err); return }
        for _, a := range vals {
                if i, ok := a.(*Int); ok {
                        if i.int64 > 0 {
                                linesPos = append(linesPos, int(i.int64))
                        } else if i.int64 < 0{
                                linesNeg = append(linesNeg, int(i.int64))
                        } else {
                                diag.errorOf(a, "zero line number")
                                return
                        }
                } else if s, e := a.Strval(); e != nil {
                        diag.errorOf(a, "%v", e); return
                } else if s == "" {
                        diag.errorOf(a, "empty regexp"); return
                } else if r, e := regexp.Compile(s); e != nil {
                        diag.errorOf(a, "%v", e); return
                } else {
                        rxs = append(rxs, r)
                }
        }

        if vals, err = mergeresult(ExpandAll(args[1:]...)); err != nil { diag.errorAt(pos, "%v", err); return }
        for _, a := range vals {
                var file *os.File
                var filename string
                if filename, err = a.Strval(); err != nil {
                        diag.errorOf(a, "%v", err); return
                }
                if file, err = os.Open(filename); err != nil {
                        diag.errorOf(a, "%v", err); return
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
                                var elems = []Value{ MakeInt(pos, int64(line+n)) }
                                for _, s := range ss {
                                        elems = append(elems, MakeString(pos, s))
                                }
                                list = append(list, MakeGroup(pos, elems...))
                        }

                        line += 1 // go behind the last line 
                        for _, n := range linesNeg {
                                var ss, ok = greps[line+n]
                                if !ok || ss == nil { continue }
                                var elems = []Value{ MakeInt(pos, int64(line+n)) }
                                for _, s := range ss {
                                        elems = append(elems, MakeString(pos, s))
                                }
                                list = append(list, MakeGroup(pos, elems...))
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
        rsConfigure = `^[\t ]*#[\t ]*(define|undef|smartdefine|smartdefine01|cmakedefine|cmakedefine01)[\t ]+([A-Za-z0-9_]+)(?:[\t ]+([^\n]*))?$`
        rxConfigure = regexp.MustCompile(fmt.Sprintf(`(?m:%s)`, rsConfigure))
        rxConfigRef = regexp.MustCompile(rsConfigRef)
)

func (project *Project) config(name string) (def *Def, err error) {
        var obj Object
        if obj, err = project.resolveObject(name); err == nil && !isNil(obj) { def, _ = obj.(*Def) }
        if false && def != nil { fmt.Fprintf(stderr, "%s: %s: %v\n", project, def.position, def) }
        return
}

func (project *Project) configExpand(pos Position, s string) (result string, err error) {
        var index = 0
        var res = new(bytes.Buffer)
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

                var def *Def
                if def, err = project.config(name); err != nil { diag.errorAt(pos, "%v", err); return }
                if false && strings.Contains(name, "LLDB_") { fmt.Fprintf(stderr, "%s: %s: %s: %v\n", project, pos, name, def) }
                if def != nil {
                        var val = def.Call(pos)
                        if false { fmt.Fprintf(stderr, "%s: %s: %s = %v -> %v (%v)\n", project, pos, name, def.value, val, typeof(val)) }
                        if isNil(val) || isNone(val) { continue }
                        switch t := val.(type) {
                        case *Plain: fmt.Fprintf(res, "%s", t.Value)
                        case *answer, *boolean:
                                var v int64
                                if v, err = t.Integer(); err != nil { diag.errorAt(pos, "%v", err); return }
                                fmt.Fprintf(res, "%d", v)
                        case *Group:
                                var v string
                                if v, err = parseGroupValue(t).Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                fmt.Fprintf(res, "%s", v)
                        default:
                                var v string
                                if v, err = val.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                fmt.Fprintf(res, "%s", v)
                        }
                }
        }
        if index < len(s) { fmt.Fprint(res, s[index:]) }
        result = res.String()
        return
}

func configure(pos Position, out *bytes.Buffer, project *Project, str string) (err error) {
        var index = 0
        if str, err = project.configExpand(pos, str); err != nil { diag.errorAt(pos, "%v", err); return }
        for _, m := range rxConfigure.FindAllStringSubmatchIndex(str, -1) {
                if _, err = out.WriteString(str[index:m[0]]); err != nil { diag.errorAt(pos, "%v", err); return }
                index = m[1] // reset index immediately to keep forward

                var t bool
                var s string
                var verb = str[m[2]:m[3]]
                var name = str[m[4]:m[5]]
                var hasv = m[6] > m[0] && m[7] > m[6]
                var def *Def
                if def, err = project.config(name); err != nil {
                        diag.errorAt(pos, "config '%s' failed: %v", name, err)
                        return
                } else if def == nil {
                        // ...
                } else if t, err = def.True(); err != nil {
                        diag.errorAt(pos, "truthify '%v failed: %v", def, err)
                        return
                }
                //fmt.Fprintf(stderr, "%v: configure: %v %v %v\n", scope.comment, verb, name, def)
                switch verb {
                case "define":
                        if hasv /*&& !(def == nil || def.value == nil)*/ {
                                v := str[m[6]:m[7]]
                                s = fmt.Sprintf("#define %s %s", name, v)
                        } else {
                                s = fmt.Sprintf("#define %s", name)
                        }
                case "undef":
                        var va []Value
                        if def == nil {
                                s = fmt.Sprintf("#undef %s", name)
                        } else if isNil(def.value) || isNone(def.value) {
                                s = fmt.Sprintf("#undef %s /* %v */", name, def.value)
                        } else if va, err = ExpandAll(def.value); err != nil {
                                diag.errorOf(def.value, "expand '%v' failed: %v", def.value, err)
                                return
                        } else if len(va) == 1 {
                                switch v := va[0].(type) {
                                case *answer, *boolean:
                                        if b, e := v.True(); e != nil {
                                                diag.errorOf(def.value, "truthify '%v' failed: %v", v, e)
                                                return
                                        } else if b {
                                                s = fmt.Sprintf("#define %s 1 /* %T %v */", name, v, v)
                                        } else {
                                                s = fmt.Sprintf("#undef %s /* %T %v */", name, v, v)
                                        }
                                case *String:
                                        s = strings.Replace(v.string, "\"", "\\\"", -1)
                                        s = fmt.Sprintf("#define %s \"%s\"", name, v.string)
                                default:
                                        s = fmt.Sprintf("#define %s %v /* %T */", name, v, v)
                                }
                        } else {
                                var v = def.value
                                s = fmt.Sprintf("#define %s %v /* %T %v */", name, v, v, va)
                        }
                case "smartdefine", "cmakedefine":
                        if !t {
                                s = fmt.Sprintf("/* #undef %s */", name)
                        } else if hasv {
                                v := str[m[6]:m[7]]
                                s = fmt.Sprintf("#define %s %s", name, v)
                        } else {
                                s = fmt.Sprintf("#define %s", name)
                        }
                case "smartdefine01", "cmakedefine01":
                        if !t {
                                s = fmt.Sprintf("#define %s 0", name)
                        } else if hasv {
                                v := str[m[6]:m[7]]
                                s = fmt.Sprintf("#define %s 1 /* %s */", name, v)
                        } else {
                                s = fmt.Sprintf("#define %s 1", name)
                        }
                }

                if _, err = out.WriteString(s); err != nil { diag.errorAt(pos, "%v", err); return }
        }
        if index < len(str) { _, err = out.WriteString(str[index:]) }
        return
}

func builtinReturn(pos Position, args... Value) Value {
        //if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "%v", err); return }
        return &returner{ valbase{pos}, args }
}
