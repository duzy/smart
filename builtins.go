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

func makePosition(filename string, line, column int) (pos Position) {
        pos.Filename = filename
        pos.Line     = line
        pos.Column   = column
        return
}
func convPosition(filename, line, column string) (pos Position) {
        pos.Filename  = filename
        pos.Line, _   = strconv.Atoi(line)
        pos.Column, _ = strconv.Atoi(column)
        return
}

type BuiltinFunc func(ctx Context, args... Value) (Value)

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
        `fullname`: builtinFullName,

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

        `file`:       builtinFile,
        `stat`:       builtinStat,// stat (deprecates file-exists)
        `glob`:       builtinGlob,
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

func EscapedString(ctx Context, v Value) (s string, e error) {
        if p, ok := v.(*String); ok {
                if s, e = p.Strval(ctx); e == nil {
                        s = strings.Replace(s, "\\'", "'", -1)
                }
        } else {
                s, e = v.Strval(ctx)
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

type optFullname struct {
        string
        value Value
}

func parseOpt(ctx Context, tag reflect.StructTag, field reflect.Value, args... Value) (rest []Value, err error) {
        var (
                val = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
                pos Position = ctx.Position()
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
                                ctx.error("illform option tag: %s", t).at(pos).debug(1)
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
                        ctx.error("illform option tag: %s", tag).at(pos).debug(1)
                        return
                }
        }
        if false { ctx.info("%v -> %v %v\n", tag, short, long).at(pos).debug(1) }
        if len(short) != len(long) {
                ctx.error("short and long option names not matching: %v, %v", short, long).at(pos).debug(1)
                return
        }

        var set func(reflect.Value, Value)
        set = func(val reflect.Value, v Value) {
                switch val.Kind() {
                case reflect.Bool:
                        if t, e := v.True(ctx); e == nil { val.SetBool(t) } else {
                                ctx.error("truthify '%v' failed: %v", v, e).of(v).debug(1)
                        }
                case reflect.Float32, reflect.Float64:
                        if t, e := v.Float(ctx); e == nil { val.SetFloat(t) } else {
                                ctx.error("floatify '%v' failed: %v", v, e).of(v).debug(1)
                        }
                case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
                        if t, e := v.Integer(ctx); e == nil { val.SetInt(t) } else {
                                ctx.error("integify '%v' failed: %v", v, e).of(v).debug(1)
                        }
                case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
                        if t, e := v.Integer(ctx); e == nil { val.SetUint(uint64(t)) } else {
                                ctx.error("integify '%v' failed: %v", v, e).of(v).debug(1)
                        }
                 case reflect.String:
                        if t, e := v.Strval(ctx); e == nil { val.SetString(t) } else {
                                ctx.error("stringify '%v' failed: %v", v, e).of(v).debug(1)
                        }
                case reflect.Slice:
                        if tp := reflect.New(val.Type().Elem()); tp.Kind() == reflect.Ptr {
                                var tv = tp.Elem()
                                set(tv, v)
                                val.Set(reflect.Append(val, tv))
                        }
                case reflect.Interface: switch val.Type().String() {
                case "smart.Value": val.Set(reflect.ValueOf(v))
                default: ctx.error("option type unsupported: %T %v -> %v, %v", v, v, val.Kind(), val.Type()).
                        of(v).debug(1)
                }
                case reflect.Ptr: switch val.Type().Elem().String() {
                case "smart.optFullname":
                        var x Value
                        if x, err = v.expand(ctx, expandPlainValue|expandFullName); err != nil {
                                ctx.error("expand option '%v' failed: %v", v, err).of(v).debug(1)
                                return
                        } else if isNil(x) { x = v } else if isNone(x) {
                                ctx.error("expecting file value: %T %v", v, v).of(v).debug(1)
                                return
                        }
                        if _, s, ok, err = asOptFullname(ctx, nil, x); err != nil {
                                ctx.error("fullname '%v' failed: %v", x, err).of(x).debug(1)
                        } else if ok && s != "" {
                                val.Set(reflect.ValueOf(&optFullname{ s, x }))
                        } else {
                                var tv, _ = ctx.autoGet("@")
                                ctx.error("not a file: %v -> %v -> %s (%T, @=%v)", v, x, s, ctx, tv).of(v).debug(16)
                        }
                        if false {
                                vi := val.Interface().(*optFullname)
                                ctx.warn("%v %v %v", /*current().of(v)*/ctx.Project(), v, vi.string).debug(true,1)
                        }
                case "smart.File":
                        var x Value
                        if x, err = v.expand(ctx, expandPlainValue); err != nil {
                                ctx.error("expand option '%v' failed: %v", v, err).of(v).debug(1)
                                return
                        } else if isNil(x) { x = v } else if isNone(x) {
                                ctx.error("expecting file value: %T %v", v, v).of(v).debug(1)
                                return
                        }
                        if file, ok := x.(*File); ok {
                                val.Set(reflect.ValueOf(file))
                        } else if s, e := x.Strval(ctx); e != nil {
                                ctx.error("strval '%v' failed: %v", x, e).of(x).debug(1)
                        } else if proj := /*current()*/ctx.Project(); proj == nil {
                                ctx.error("no current project to find file '%v'", s).of(x).debug(1)
                        } else if file = proj.FindFile(ctx, s); file != nil {
                                val.Set(reflect.ValueOf(file))
                        } else {
                                ctx.error("'%s' is not a file", s).of(v).debug(1)
                        }
                case "regexp.Regexp":
                        if s, e := v.Strval(ctx); e != nil {
                                ctx.error("stringify '%v' failed: %v", v, e).of(v).debug(1)
                        } else if rx, e := regexp.Compile(s); e != nil {
                                ctx.error("compile regexp '%v' failed: %v", v, e).of(v).debug(1)
                        } else {
                                val.Set(reflect.ValueOf(rx))
                        }
                default:
                        ctx.error("option type unsupported: %T %v -> %v, %v", v, v, val.Elem().Kind(), val.Type().Elem()).
                                of(v).debug(1)
                }
                default: switch val.Type().String() {
                case "fs.FileMode": // aka. reflect.Uint32
                        if t, e := v.Integer(ctx); e == nil { val.SetUint(uint64(t)) } else {
                                ctx.error("integify '%v' failed: %v", v, e).of(v).debug(1)
                        }
                case "regex.Regex": // aka. reflect.Ptr
                        ctx.error("TODO: regexp: %T %v -> %v, %v", v, v, val.Kind(), val.Type()).of(v).debug(1)
                default:
                        ctx.error("option type unsupported: %T %v -> %v, %v", v, v, val.Kind(), val.Type()).of(v).debug(1)
                }}
        }
ForArgs:
        for _, arg := range args {
                var (
                        okay bool
                        flag *Flag
                        value Value
                )
                if arg.patterned(ctx) {
                        // don't parse patterns, e.g. -I%
                } else if flag, okay = arg.(*Flag); okay {
                        value = MakeBoolean(flag.position, true)
                } else if pair, ok := arg.(*Pair); ok {
                        if flag, okay = pair.Key.(*Flag); okay { value = pair.Value }
                }
                if !okay || flag == nil {
                        rest = append(rest, arg)
                        continue ForArgs
                }
                for i := 0; i < len(short) && i < len(long); i += 1 {
                        if _, match := flag.opt(ctx, short[i], long[i]); match {
                                set(val, value)
                                continue ForArgs
                        }
                }
                rest = append(rest, arg)
        }
        if false && len(args) > 0 {
                ctx.info("%v,%v: %v %v %v", short, long, field.Kind(), field, rest).at(pos)
        }
        return
}

func parseOpts(ctx Context, iOpts interface{}, args... Value) (rest []Value, err error) {
        var pos = ctx.Position()
        rest = args // NOTE: set the returning args first of all!
        if opts := reflect.ValueOf(iOpts); opts.Kind() != reflect.Ptr {
                ctx.error("opts must be ptr: %v", opts.Kind()).at(pos).debug(1)
        } else if opts = opts.Elem(); opts.Kind() == reflect.Struct {
                var otyp = opts.Type()
                if false { ctx.info("opts: %v, %v", opts.Kind(), otyp).at(pos) }
                for i := 0; i < otyp.NumField(); i += 1 {
                        var ft = otyp.Field(i)
                        var fv = opts.Field(i)
                        rest, err = parseOpt(ctx, ft.Tag, fv, rest...)
                }
        } else {
                ctx.error("opts is not ptr of struct: %v", opts.Kind()).at(pos).debug(1)
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

func builtinTypeOf(ctx Context, args... Value) (res Value) {
        var ( pos = ctx.Position(); elems []Value; s string )
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

func builtinDefined(ctx Context, args... Value) (res Value) {
        var ( pos = ctx.Position(); elems []Value; err error )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).at(pos).debug(1)
                return
        }
        for _, arg := range args {
                var _, unresolved = arg.(*unresolvedobject)
                elems = append(elems, MakeBoolean(pos, !unresolved))
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
func builtinPosition(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts builtinPositionOpts
                vals []Value
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("position: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("position: %v", err).debug(1)
                return
        }

        if opts.filename {
                vals = append(vals, MakeString(pos, pos.Filename))
        } else if opts.filenameQuoted {
                var s = pos.Filename //strconv.Quote(pos.Filename)
                vals = append(vals, MakeString(pos, "\""+s+"\""))
        }

        if opts.line   { vals = append(vals, MakeInt(pos, int64(pos.Line + opts.addLine))) }
        if opts.column { vals = append(vals, MakeInt(pos, int64(pos.Column + opts.addColumn))) }

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
func builtinDate(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts = builtinDateOpts{ today:true }
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }

        if t := time.Now(); opts.now {
                res = MakeTime(pos, t)
        } else if opts.today {
                res = MakeDate(pos, t)
        }
        return
}

func builtinError(ctx Context, args... Value) (res Value) {
        var (
                s bytes.Buffer
                v string
                err error
        )
        for i, a := range args {
                if i > 0 { fmt.Fprintf(&s, " ") }
                if v, err = a.Strval(ctx); err == nil {
                        fmt.Fprintf(&s, "%s", v)
                } else {
                        ctx.error("error: %v: %v", a, err).of(a).debug(1)
                        return
                }
        }
        ctx.error("%s", s).debug(1)
        return
}

func builtinWarning(ctx Context, args... Value) (res Value) {
        var (
                s bytes.Buffer
                v string
                err error
        )
        for i, a := range args {
                if i > 0 { fmt.Fprintf(&s, " ") }
                if v, err = a.Strval(ctx); err == nil {
                        fmt.Fprintf(&s, "%s", v)
                } else {
                        ctx.error("warning: %v: %v", a, err).of(a).debug(1)
                        return
                }
        }
        ctx.warn("%s", s).debug(1)
        return
}

func builtinAssert(ctx Context, args... Value) Value {
        var vals []Value
        for _, a := range args {
                if g, ok := a.(*Group); ok {
                        vals = append(vals, g.Elems...)
                }
        }
        for _, a := range vals {
                if v, e := a.True(ctx); e != nil {
                        ctx.error("assert: error: %v", e).of(a).debug(1)
                } else if !v {
                        ctx.error("assertion failed: %v", a).of(a).debug(1)
                }
        }
        return nil
}

// $(defor $(x),$(y),$(z)) is identical to $(if $(defined $(x)),$(x),...)
func builtinDefor(ctx Context, args... Value) (res Value) {
        var (
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
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

func builtinOr(ctx Context, args... Value) (res Value) {
        for _, a := range args {
                if false { if t := ctx.traversal(); t != nil && t.entry != nil && t.entry.String() == "HAVE_TERMINFO" {
                        v, _ := a.expand(ctx, expandPlainValue)
                        b, _ := a.True(ctx)
                        i, _ := v.True(ctx)
                        ctx.info("or: %T %v -> %T %v -> %v, %v", a, a, v, v, b, i).of(a).debug(1)
                }}
                if v, e := a.True(ctx); e != nil {
                        ctx.error("or: error: %v", e).of(a).debug(1)
                        break
                } else if v {
                        res = a
                        break
                }
        }
        return
}

func builtinAnd(ctx Context, args... Value) (res Value) {
        for _, a := range args {
                if v, e := a.True(ctx); e != nil {
                        ctx.error("and: error: %v", e).of(a).debug(1)
                        break
                } else if v {
                        res = a
                } else {
                        res = nil; break
                }
        }
        return
}

// $(not x y z) -> (not (or x y z))
// $(not x,y,z) -> (and (not x) (not y) (not z))
func builtinNot(ctx Context, args... Value) (res Value) {
        var (
                t bool
                e error
        )
        for _, a := range args {
                if t, e = a.True(ctx); e != nil {
                        ctx.error("not: error: %v", e)
                        return
                } else if t {
                        res = MakeBoolean(ctx.Position(), false)
                        return
                }
        }
        if e == nil { res = MakeBoolean(ctx.Position(), true) }
        return
}

func builtinNotEqual(ctx Context, args... Value) (res Value) {
        if n := len(args); n != 2 {
                ctx.error("wrong number of arguments, try: $(not-equal <value-list>,<regexp-list>)")
        } else if args[0].cmp(ctx, args[1]) != cmpEqual {
                res = MakeBoolean(ctx.Position(), true)
        }
        return
}

func builtinEqual(ctx Context, args... Value) (res Value) {
        if n := len(args); n != 2 {
                ctx.error("wrong number of arguments, try: $(equal <value-list>,<regexp-list>)")
        } else if cmp := args[0].cmp(ctx, args[1]); cmp == cmpEqual {
                res = MakeBoolean(ctx.Position(), true)
        }
        return
}

func builtinGreater(ctx Context, args... Value) (res Value) {
        if n := len(args); n != 2 {
                ctx.error("wrong number of arguments, try: $(greater <value-list>,<regexp-list>)")
        } else if cmp := args[0].cmp(ctx, args[1]); cmp == cmpGreater {
                res = MakeBoolean(ctx.Position(), true)
        }
        return
}

func builtinLess(ctx Context, args... Value) (res Value) {
        if n := len(args); n != 2 {
                ctx.error("wrong number of arguments, try: $(less <value-list>,<regexp-list>)")
        } else if cmp := args[0].cmp(ctx, args[1]); cmp == cmpSmaller {
                res = MakeBoolean(ctx.Position(), true)
        }
        return
}

type builtinMatchOpts struct {
        regexps []*regexp.Regexp `r,reg;rx,regex;re,regexp`
}
// $(match rx1 rx2 rx3, a b c d...)
func builtinMatch(ctx Context, args... Value) (res Value) {
        var (
                patList, valList []Value
                opts builtinMatchOpts
                err error
        )
        if n := len(args); n < 2 {
                ctx.error("wrong arguments, try: $(match <regexp-list>,<value-list>,...)").debug(1)
                return
        } else if patList, err = expandmerge2(ctx, expandPlainValue, args[0]); err != nil {
                ctx.error("expand '%v' failed: %v", args[0], err).debug(1)
                return
        } else if patList, err = parseOpts(ctx, &opts, patList...); err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        } else if valList, err = expandmerge2(ctx, expandPlainValue, args[1:]...); err != nil {
                ctx.error("expand value list failed: %v", err).debug(1)
                return
        }

        var pos = ctx.Position()
ForValList:
        for _, val := range valList {
                if isNil(val) || isUndef(val) || isNone(val) {
                        continue ForValList
                }
                var str string
                if str, err = val.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", val, err).of(val).debug(1)
                        return
                }
                for _, rx := range opts.regexps {
                        if rx.MatchString(str) {
                                res = MakeBoolean(pos, true)
                                break ForValList
                        }
                }
                for _, pat := range patList {
                        if matched, _, _ := pat.match(ctx, str); matched {
                                res = MakeBoolean(pos, true)
                                break ForValList
                        }
                }
        }
        return
}

// $(if cond, true-value, else-value, ...)
func builtinBranchIf(ctx Context, args... Value) (res Value) {
        if n := len(args); n > 1 {
                var (
                        err error
                        t bool
                )
                if t, err = args[0].True(ctx); err != nil {
                        ctx.error("truthify if condition failed: %v", err).debug(1)
                } else if t { 
                        res = args[1]
                } else if n > 1 {
                        res = MakeListOrScalar(ctx.Position(), args[2:])
                }
        }
        return
}

func builtinBranchIfEq(ctx Context, args... Value) (res Value) {
        if n := len(args); n > 2 {
                var (
                        s1, s2 string
                        a, b Value
                        err error
                )
                if a, err = args[0].expand(ctx, expandPlainValue); err != nil {
                        ctx.error("expand '%v' failed: %v", args[0], err).debug(1)
                        return
                } else if isNil(a) { a = args[0] }
                if b, err = args[1].expand(ctx, expandDelegate); err != nil {
                        ctx.error("expand '%v' failed: %v", args[1], err).debug(1)
                        return
                } else if isNil(b) { b = args[1] }

                if s1, err = a.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", a, err).debug(1)
                        return
                }
                if s2, err = b.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", b, err).debug(1)
                        return
                }
                if s1 == s2 {
                        res = args[2]
                } else if n > 3 {
                        res = MakeListOrScalar(ctx.Position(), args[3:])
                }
        }
        return
}

func builtinBranchIfNE(ctx Context, args... Value) (res Value) {
        if n := len(args); n > 2 {
                var (
                        s1, s2 string
                        a, b Value
                        err error
                )
                if a, err = args[0].expand(ctx, expandPlainValue); err != nil {
                        ctx.error("expand '%v' failed: %v", args[0], err).debug(1)
                        return
                } else if isNil(a) { a = args[0] }
                if b, err = args[1].expand(ctx, expandDelegate); err != nil {
                        ctx.error("expand '%v' failed: %v", args[1], err).debug(1)
                        return
                } else if isNil(b) { b = args[1] }

                if s1, err = a.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", a, err).debug(1)
                        return
                }
                if s2, err = b.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", b, err).debug(1)
                        return
                }
                if s1 != s2 {
                        res = args[2]
                } else if n > 3 {
                        res = MakeListOrScalar(ctx.Position(), args[3:])
                }
        }
        return
}

func builtinFor(ctx Context, args... Value) (res Value) {
        if n := len(args); n < 2 {
                ctx.error("not enough arguments, try: $(foreach <list>,<template>)")
        } else {
                var ( defs []*Def ; vals, values []Value; err error )
                if values, err = expandmerge2(ctx, expandPlainValue, args[0]); err != nil {
                        ctx.error("merge '%v' failed: %v", args[0], err).debug(1)
                        return
                }

                var scope = ctx.Globe().scope
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

                var pos = ctx.Position()
                var list []Value
                for _, a := range args[1:] {
                        if values, err = expandmerge2(ctx, expandPlainValue, a); err != nil {
                                ctx.error("merge '%v' failed: %v", a, err)
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

func builtinForEach(ctx Context, args... Value) (res Value) {
        if n := len(args); n < 2 {
                ctx.error("not enough arguments ($(foreach <list>,<template>)): %v", n).debug(1)
                return
        }

        var (
                cc = autoContext{ Context:ctx, defs:make(autoDefMap) }
                resList []Value
                values []Value
                err error
        )
        if values, err = expandmerge2(ctx, expandPlainValue, args[0]); err != nil {
                ctx.error("merge arg0 failed: %v", err).debug(1)
                return
        }

        var pos = ctx.Position()
        for _, val := range values {
                if isNil(val) || isUndef(val) || isNone(val) {
                        continue // ignore
                } else if s, ok := val.(*String); ok && s.string == "" {
                        continue // ignore
                } else { cc.autoSet("_", val) }

                var list []Value
                for _, a := range args[1:] {
                        var v Value
                        if v, err = a.expand(&cc, expandPlainValue|expandPairVal); err != nil {
                                ctx.error("expand '%v' failed: %v", a, err).of(a).debug(1)
                                return
                        } else if isNil(v) { v = a }
                        if true && len(v.defs(&cc, "_")) > 0 {
                                ctx.error("'_' in '%v' not expanded: %v", a, v).of(a).debug(true, 1)
                                return
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

func builtinEnv(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                vals []Value
                val Value
                v string
                err error
        )
        for _, a := range args {
                if val, err = a.expand(ctx, expandDelegate); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
                if val == nil {
                        // discard
                } else if v, err = val.Strval(ctx); err == nil {
                        if s := strings.TrimSpace(v); s != "" {
                                vals = append(vals, MakeString(pos, os.Getenv(s)))
                        }
                } else {
                        ctx.error("%v", err).debug(1)
                        return
                }
        }
        return MakeListOrScalar(pos, vals)
}

type builtinValueOpts struct {
        closure bool `c,closure`
}
func builtinValue(ctx Context, args... Value) (res Value) {
        /*
        var (
                scope *Scope
                vals []Value
                err error
        )
        if len(cloctx) > 0 { scope = cloctx[0] } else {
                scope = ctx.LoaderScope()
        }

        for _, a := range args {
                var s string
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", a, err).debug(1)
                        return
                }
                if def := scope.FindDef(s); def != nil {
                        vals = append(vals, def.value)
                } else {
                        vals = append(vals, MakeNone(pos))
                }
        }*/
        var (
                opts builtinValueOpts
                vals []Value
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }
        for _, a := range args {
                var ( name string; val Value/*; scope *Scope*/ )
                if name, err = a.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", a, err).at(a.Position()).debug(1)
                        return
                } else if opts.closure {
                        val = ctx.closureGet(name)
                } /*else if scope = ctx.closureScope(); scope == nil {
                        scope = ctx.loaderScope()
                }
                if scope != nil {
                        if def := scope.FindDef(name); def != nil {
                                val = def.Call(ctx)
                        }
                }*/
                if isNil(val) { val, _ = ctx.autoGet(name) }
                if isNil(val) { val = MakeNone(a.Position()) }
                vals = append(vals, val)
        }
        return MakeListOrScalar(ctx.Position(), vals)
}

func builtinList(ctx Context, args... Value) (res Value) {
        res = MakeListOrScalar(ctx.Position(), args)
        return
}

func builtinShell(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                vals []Value
                err error
        )
        for _, a := range args {
                var ( bufout, buferr bytes.Buffer; s string )
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
                sh := exec.Command("sh", "-c", s)
                sh.Stdout, sh.Stderr = &bufout, &buferr
                if err = sh.Run(); err != nil {
                        s = strings.TrimSpace(buferr.String())
                        ctx.error("%s", err).debug(1)
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
func builtinServeHttp(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts = builtinServeHttpOpts{ port:80 }
                va []Value
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        } else if va, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("%v", err).debug(1)
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
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
                fmt.Fprintf(stderr, "%s: serving files %v ...\n", pos, s)
                http.Handle("/", http.FileServer(http.Dir(s)))
        }

        if err = server.ListenAndServe(); err == http.ErrServerClosed {
                if false { ctx.info("http server closed") }// Requested /quit
        } else if err != nil {
                ctx.error("%s", err).debug(1)
        }
        return
}

func builtinServeHttps(ctx Context, args... Value) (res Value) {
        ctx.error("'serve-https' is unimplemented yet").at(ctx.Position()).debug(1)
        return
}

func builtinPrint(ctx Context, args... Value) (res Value) {
        var (
                x = len(args)
                err error
        )
        for i, a := range args {
                var s string
                if 0 < i && i < x { fmt.Printf(" ") }
                if a == nil {
                        continue
                } else if s, err = EscapedString(ctx, a); err == nil {
                        if s != "" { fmt.Printf("%s", s) }
                } else {
                        ctx.error("%s", err).debug(1)
                        break
                }
        }
        return
}

func builtinPrintl(ctx Context, args... Value) (res Value) {
        var (
                x = len(args)
                err error
        )
        for i, a := range args {
                var s string
                if 0 < i && i < x { fmt.Printf(" ") }
                if s, err = EscapedString(ctx, a); err != nil {
                        ctx.error("%s", err)
                        return
                }
                fmt.Printf("%s", s)
                if i == x && !strings.HasSuffix(s, "\n") {
                        fmt.Printf("\n")
                }
        }
        return
}

func builtinPrintln(ctx Context, args... Value) (res Value) {
        builtinPrint(ctx, args...)
        fmt.Printf("\n")
        return
}

type builtinAppendOpts struct {
        verbose bool `v,verbose`
        closure bool `c,closure`
        string bool `s,str;s,string`
}
func builtinAppend(ctx Context, args... Value) (result Value) {
        var (
                opts builtinAppendOpts
                vars []Value
                list []Value
                err error
        )
        if len(args) < 2 {
                ctx.error("insufficient number of arguments: %v", args).debug(1)
                return
        } else if vars, err = expandmerge2(ctx, expandPlainValue, args[0]); err != nil {
                ctx.error("%s", err).of(args[0]).debug(1)
                return
        } else if vars, err = parseOpts(ctx, &opts, vars...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        } else if list, err = expandmerge2(ctx, expandPlainValue, args[1:]...); err != nil {
                ctx.error("%s", err).of(args[1]).debug(1)
                return
        } else if len(list) == 0 {
                ctx.warn("append no values").debug(1)
                return
        }

        var pos = ctx.Position()
        for _, a := range vars {
                var name string
                if name, err = a.Strval(ctx); err != nil {
                        ctx.error("%s", err).of(a).debug(1)
                        break
                } else if name == "" {
                        ctx.error("name '%v' is empty", a).of(a).debug(1)
                        break
                }
                /*
                var def *Def
                if def == nil {
                        var obj Object
                        if obj, err = cloctx[0].project.resolveObject(ctx, name); err != nil {
                                ctx.error("%v", err).of(a).debug(1)
                                break
                        } else if def, _ = obj.(*Def); def == nil {
                        }
                }
                if def == nil {
                        for _, scope := range cloctx {
                                if def = scope.FindDef(name); def != nil { break }
                        }
                }
                if def == nil {
                        ctx.error("'%s' (%v) is undefined (%v)", name, a, cloctx).debug(1)
                        break
                } else if err = def.append(ctx, list...); err != nil {
                        ctx.error("%s", err).debug(1)
                        break
                } */
                if val := ctx.closureGet(name); !isNil(val) {
                        list = append(merge(val), list...)
                }
                if val := MakeListOrScalar(pos, list); opts.closure {
                        ctx.closureSet(name, val)
                } else {
                        ctx.autoSet(name, val)
                }
        }
        return
}

func builtinPlus(ctx Context, args... Value) (result Value) {
        var (
                num, v int64
                err error
        )
        for _, a := range args {
                if v, err = a.Integer(ctx); err != nil {
                        ctx.error("%s", err).of(a).debug(1)
                        return
                }
                num += v
        } 
        return MakeInt(ctx.Position(), num)
}

func builtinMinus(ctx Context, args... Value) (result Value) {
        var (
                num, v int64
                err error
        )
        for i, a := range args {
                if v, err = a.Integer(ctx); err != nil {
                        ctx.error("%s", err).of(a).debug(1)
                        return
                } else if i == 0 {
                        num = v
                } else {
                        num -= v
                }
        }
        return MakeInt(ctx.Position(), num)
}

type builtinUniqueOpts struct {
        reverse bool `r,reverse`
}
func builtinUnique(ctx Context, args... Value) (res Value) {
        var (
                opts builtinUniqueOpts
                err error
        )
        if options.benchBuiltins {
                defer func(t time.Time) {
                        ctx.info("(%8s) unique", time.Now().Sub(t))
                } (time.Now())
        }
        if len(args) > 0 {
                var a []Value
                if a, err = parseOpts(ctx, &opts, merge(args[0])...); err != nil {
                        ctx.error("%v", err).of(args[0]).debug(1)
                        return
                }
                args = append(a, args[1:]...)
        }
        if false {
                args = merge(args...)
        } else if true {
                if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
        } else {
                var x = expandDelegate | expandPathStr | expandPairVal
                if args, err = expandmerge2(ctx, x, args...); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
        }

        var list []Value
ForArgs:
        for i, a := range args {
                var tmp []Value
                if opts.reverse { tmp = args[i+1:] } else { tmp = list }
                for _, v := range tmp {
                        if a == v || a.cmp(ctx, v) == cmpEqual {
                                continue ForArgs
                        }
                }

                if false {
                        var s1, s2 string
                        if s1, err = a.Strval(ctx); err != nil {
                                ctx.error("%v", err).of(a).debug(1)
                                return
                        }
                        for _, v := range list {
                                if s2, err = v.Strval(ctx); err != nil {
                                        ctx.error("%v", err).of(v).debug(1)
                                        return
                                }
                                if s1 == s2 { continue ForArgs }
                        }
                }

                list = append(list, a)
        }
        res = MakeListOrScalar(ctx.Position(), list)
        return
}

func builtinJoin(ctx Context, args... Value) (res Value) {
        if l := len(args); l > 0 {
                var (
                        fields []string
                        vals []Value
                        sep string
                        err error
                )
                if l < 2 {
                        if vals, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                                ctx.error("%v", err).debug(1)
                                return
                        }
                } else {
                        if vals, err = expandmerge2(ctx, expandPlainValue, args[:l-1]...); err != nil {
                                ctx.error("%v", err).debug(1)
                                return
                        } else if sep, err = args[l-1].Strval(ctx); err != nil {
                                ctx.error("%v", err).debug(1)
                                return
                        }
                }
                for _, a := range vals {
                        var v string
                        if v, err = a.Strval(ctx); err != nil {
                                ctx.error("%v", err).debug(1)
                                return
                        }
                        if v != "" { fields = append(fields, v) }
                }
                res = MakeString(ctx.Position(), strings.Join(fields, sep))
        }
        return
}

func builtinQuote(ctx Context, args... Value) (res Value) {
        var err error
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }
        if l := len(args); l > 0 {
                var (
                        fields []string
                        v string
                )
                for _, a := range args {
                        if v, err = a.Strval(ctx); err != nil {
                                ctx.error("%v", err)
                                return
                        } else if v != "" { fields = append(fields, v) }
                }
                res = MakeString(ctx.Position(), strconv.Quote(strings.Join(fields, " ")))
        } else {
                res = MakeNone(ctx.Position())
        }
        return
}

func builtinQuoteJoin(ctx Context, args... Value) (res Value) {
        var (
                sep string
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }

        if l := len(args); l > 1 {
                if sep, err = args[l-1].Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
                args = args[:l-1]
        }
        if l := len(args); l > 0 {
                var fields []string
                var v string
                for _, a := range args {
                        if v, err = a.Strval(ctx); err != nil {
                                ctx.error("%v", err).debug(1)
                                return
                        } else if v != "" { fields = append(fields, v) }
                }
                res = MakeString(ctx.Position(), strconv.Quote(strings.Join(fields, sep)))
        } else {
                res = MakeNone(ctx.Position())
        }
        return
}

func builtinSplitString(ctx Context, args... Value) (res Value) {
        var (
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        } else if l := len(args); l > 0 {
                var fields []Value
                for _, a := range args {
                        var s string
                        if s, err = a.Strval(ctx); err != nil {
                                ctx.error("%v", err).debug(1)
                                return
                        } else if s != "" {
                                fields = append(fields, MakeString(a.Position(), s))
                        }
                }
                res = MakeList(ctx.Position(), fields...)
        } else {
                res = MakeNone(ctx.Position())
        }
        return
}

func quotestrings(value Value) {
        switch v := value.(type) {
        case *String: v.string = strconv.Quote(v.string)
        case *List:
                for _, elem := range v.Elems {
                        quotestrings(elem)
                }
        }
        return
}

func joinstrings(ctx Context, value Value, sep string) (res Value, err error) {
        if sep == "" { sep = " " }
ValueType:
        switch v := value.(type) {
        case *String: res = value
        case *List:
                var strs []string
                for _, elem := range v.Elems {
                        var ( v Value; s string )
                        if v, err = joinstrings(ctx, elem, sep); err != nil { break ValueType }
                        if s, err = v.Strval(ctx); err != nil { break ValueType }
                        if s != "" { strs = append(strs, s) }
                }
                res = MakeString(value.Position(), strings.Join(strs, sep))
        }
        return
}

// TODO: deprecate this and add -quote to builtinSplitString
func builtinSplitQuote(ctx Context, args... Value) (res Value) {
        if res = builtinSplitString(ctx, args...); !isNil(res) {
                quotestrings(res)
        }
        return
}

// TODO: deprecate this and add -quote to builtinSplitString
func builtinSplitQuoteJoin(ctx Context, args... Value) (res Value) {
        var (
                sep string
                err error
        )
        if l := len(args); l > 1 {
                if sep, err = args[l-1].Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
                args = args[:l-1]
        }
        if res = builtinSplitQuote(ctx, args...); !isNil(res) {
                if res, err = joinstrings(ctx, res, sep); err != nil {
                        ctx.error("%v", err).debug(1)
                }
        } else {
                ctx.error("%v", err).debug(1)
        }
        return
}

func builtinSplitJoinQuote(ctx Context, args... Value) (res Value) {
        var (
                sep string
                err error
        )
        if l := len(args); l > 1 {
                if sep, err = args[l-1].Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
                args = args[:l-1]
        }
        var v Value
        if v = builtinSplitString(ctx, args...); !isNil(v) {
                if v, err = joinstrings(ctx, v, sep); err == nil {
                        var s string
                        if s, err = v.Strval(ctx); err == nil {
                                res = MakeString(ctx.Position(), strconv.Quote(s))
                        }
                }
        }
        if err != nil { ctx.error("%v", err).debug(1) }
        return
}

func builtinField(ctx Context, args... Value) (res Value) {
        var pos = ctx.Position()
        if l := len(args); l >= 2 {
                var (
                        fields []string
                        i int64
                        s string
                        err error
                )
                if i, err = args[0].Integer(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
                if s, err = args[1].Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
                if l > 2 {
                        var v string
                        if v, err = args[2].Strval(ctx); err != nil {
                                ctx.error("%v", err).debug(1)
                                return
                        }
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

func builtinFields(ctx Context, args... Value) (res Value) {
        // TODO: ...
        return
}

func builtinUsee(ctx Context, args... Value) (result Value) {
        var (
                proj = ctx.Project() //current()
                list []Value
                err error
        )
        if proj == nil {
                ctx.error("unknown current context").debug(1)
                return
        }

        for _, arg := range args {
                var ( s string; v Value )
                if s, err = arg.Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                } else if v, err = proj.using.Get(ctx, s); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                } else {
                        list = append(list, v)
                }
        }
        if err == nil {
                result = MakeListOrScalar(ctx.Position(), list)
        }
        return
}

func builtinPath(ctx Context, args... Value) (result Value) {
        var (
                pos = ctx.Position()
                list []Value
                err error
        )
        for _, a := range args {
                var s string
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
                list = append(list, MakePathStr(pos, s))
        }
        result = MakeListOrScalar(pos, list)
        return
}

func builtinString(ctx Context, args... Value) (result Value) {
        var (
                s bytes.Buffer
                err error
        )
        for i, a := range args {
                var v string
                if i > 0 { s.WriteString(" ") }
                if v, err = a.Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
                s.WriteString(v)
        }
        result = MakeString(ctx.Position(), s.String())
        return
}

type builtinFilterOpts struct {
        stem bool `s,stem;us,use-stem`
}
func filterValues(ctx Context, pats []Value, opts builtinFilterOpts, neg bool, values... Value) (result []Value, err error) {
        const info = false
        var filter = func(v Value) Value {
                for _, pat := range pats {
                        if info { if full, s, stems := pat.match(ctx, v); full || s != "" {
                                ctx.warn("pat=%v (%T) value=%v (%T) => full=%v result=%v stems=%v", pat, pat, v, v, full, s, stems).of(pat).debug(true, 1)
                        }}
                        if full, s, stems := pat.match(ctx, v); full {
                                if neg { v = nil } else if opts.stem {
                                        if len(stems) > 0 { s = stems[0] }
                                        v = MakeString(v.Position(), s)
                                }
                                return v
                        }
                }
                if neg { return v } else { return nil }
        }
        if values, err = mergeresult(Reveal(ctx, values...)); err != nil {
                ctx.error("%v", err).of(values[0]).debug(1)
                return
        }
        for _, v := range values {
                if t := filter(v); err != nil { break } else if t != nil {
                        result = append(result, t)
                }
        }
        return
}

func filterValues1(ctx Context, neg bool, args... Value) (res Value) {
        var ( pos = ctx.Position(); err error )
        if len(args) > 1 {
                var (
                        opts builtinFilterOpts
                        pats []Value
                        vals []Value
                )
                if pats, err = expandmerge2(ctx, expandPlainValue, args[0]); err != nil {
                        ctx.error("merge patterns '%v' failed: %v", args[0], err).debug(1)
                        return
                } else if pats, err = parseOpts(ctx, &opts, pats...); err != nil {
                        ctx.error("parse opts failed: %v", err).debug(1)
                        return
                } else if vals, err = expandmerge2(ctx, expandPlainValue, args[1:]...); err != nil {
                        ctx.error("merge values failed: %v", err).debug(1)
                        return
                }
                if vals, err = filterValues(ctx, pats, opts, neg, vals...); err == nil {
                        res = MakeListOrScalar(pos, vals)
                }
        }
        if res == nil && err == nil { res = MakeNone(pos) }
        return
}

func builtinFilter(ctx Context, args... Value) (res Value) {
        // $(filter pattern…,text)
        res = filterValues1(ctx, false, args...)
        return
}

func builtinFilterOut(ctx Context, args... Value) (res Value) {
        // $(filter-out pattern…,text)
        res = filterValues1(ctx, true, args...)
        return
}

func builtinSubstring(ctx Context, args... Value) (res Value) {
        var ( pos = ctx.Position(); err error )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        }

        var list []Value
        if n := len(args); n > 1 {
                var ( i1, i2 int )
                if i1, err = intVal(ctx, args[0], -1); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                } else {
                        args = args[1:]
                }
                if i2, err = intVal(ctx, args[0], -1); err != nil {
                        if _, ok := err.(*strconv.NumError); ok {
                                err = nil // ignore
                        } else {
                                ctx.error("%v", err).of(args[0]).debug(1)
                                return
                        }
                } else { args = args[1:] }

                if i1 < -1 && i2 < -1 {
                        ctx.error("wrong indices (%d, %d)", i1, i2).debug(1)
                        return
                } else if i1 > i2 { t := i1; i1 = i2; i2 = t } // swap the wrong order
                
                var a, b = int(i1), int(i2)
                if a == -1 { a = b }
                if a == -1 { return }

                for _, arg := range args {
                        var s string
                        if s, err = arg.Strval(ctx); err != nil {
                                ctx.error("strval '%v' failed: %v", arg, err).debug(1)
                                return
                        }
                        if i := len(s); i <= a { s = "" } else
                        if b == -1 || i <= b { s = s[a:b] } else { s = s[a:] }
                        list = append(list, MakeString(pos, s))
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

// $(subst from,to,text)
func builtinSubst(ctx Context, args... Value) (res Value) {
        var ( pos = ctx.Position(); list []Value; err error )
        if nargs := len(args); nargs > 2 {
                var ( a []Value; s, s1, s2 string )
                if s1, err = args[0].Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
                if s2, err = args[1].Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
                if a, err = expandmerge2(ctx, expandDelegate, args[2:]...); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
                for _, arg := range a {
                        if s, err = arg.Strval(ctx); err != nil {
                                ctx.error("%v", err).debug(1)
                                return
                        }
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
func builtinPatsubst(ctx Context, args... Value) (res Value) {
        var (
                proj = ctx.Project() //current()
                opts builtinPatsubstOpts
                list []Value
                arg0 []Value
                err error
        )
        if proj == nil {
                ctx.error("unknown current context").debug(1)
                return
        } else if len(args) < 3 {
                ctx.error("not enough arguments").debug(1)
                return
        } else if arg0, err = expandmerge2(ctx, expandPlainValue, args[0]); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if arg0, err = parseOpts(ctx, &opts, arg0...) ; err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }

        const infos = false
        //var infos = proj.name == "headers"

        // TODO: support flags -name and -full for name-only and full-name-only matching
        var srcPats, dstPats, sources []Value
        if len(arg0) > 0 {
                srcPats = arg0
                if dstPats, err = expandmerge2(ctx, expandPlainValue, args[1])    ; err != nil { ctx.error("%v", err); return }
                if sources, err = expandmerge2(ctx, expandPlainValue, args[2:]...); err != nil { ctx.error("%v", err); return }
                if infos {
                        ctx.info("src: %v", srcPats)
                        ctx.info("dst: %v", dstPats)
                        ctx.info("%v", sources).debug(1)
                }
        } else {
                if srcPats, err = expandmerge2(ctx, expandPlainValue, args[1])    ; err != nil { ctx.error("%v", err); return }
                if dstPats, err = expandmerge2(ctx, expandPlainValue, args[2])    ; err != nil { ctx.error("%v", err); return }
                if sources, err = expandmerge2(ctx, expandPlainValue, args[3:]...); err != nil { ctx.error("%v", err); return }
                if infos {
                        ctx.info("src: %v", srcPats)
                        ctx.info("dst: %v", dstPats)
                        ctx.info("%v", sources).debug(1)
                }
        }

        // Using the most derived context for correct &(...)
        //defer setclosure(setclosure(cloctx.unshift(proj.scope)))

        var filemaps []*FileMap
        if !opts.noFileMap { filemaps = proj.filemaps(ctx, false) }

ForSources:
        for _, src := range sources {
                var source interface{} = src
                if opts.files || opts.fullfiles {
                        var s string
                        if file, ok := src.(*File); ok {
                                source = file
                        } else if s, err = src.Strval(ctx); err != nil {
                                ctx.error("strval '%v' failed: %v", src, err).of(src)
                                ctx.error("called from here", src).debug(1)
                                return
                        } else if file = proj.FindFile(ctx, s); file != nil {
                                if (opts.full || opts.fullfiles) && !filepath.IsAbs(file.name) {
                                        if !file.change("", "", file.fullname()) {
                                                ctx.warn("changing fullname failed: %v", file).debug(1)
                                        }
                                }
                                source = file
                        }
                } else if opts.full {
                        var ( s string; ok bool )
                        if _, s, ok, err = asOptFullname(ctx, proj, src); err != nil {
                                ctx.error("fullname '%v' failed: %v", src, err).of(src)
                                ctx.error("called from here", src).debug(1)
                                return
                        } else if s == "" {
                                ctx.error("fullname '%v' is empty", src).of(src)
                                ctx.error("called from here", src).debug(1)
                                return
                        } else if !ok {
                                ctx.error("fullname '%v' failed", src).of(src)
                                ctx.error("called from here", src).debug(1)
                                return
                        } else {
                                source = s
                        }
                }

                var ( matched bool; str string; stems []string )
        ForSrcPats:
                for _, elem := range srcPats {
                        if matched, str, stems = elem.match(ctx, source); matched {
                                break ForSrcPats
                        } else if infos {
                                ctx.info("source=%v (%T) elem=%v (%T) str=%s stems=%v",
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
                        var name, rest = dst.stencil(ctx, stems)
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
                                        if ok, s := m.Match(ctx, name); ok {
                                                match, pre = m, s
                                                break
                                        }
                                }

                                var file *File
                                if match != nil {
                                        if file = match.stat(ctx, t.dir, pre, name); file != nil {
                                                assert(file.name == name, fmt.Sprintf("invalid file name: %s != %s (t.dir=%s, pre=%s)", file.name, name, t.dir, pre))
                                        } else if file = match.stat(ctx, proj.absPath, pre, name); file != nil {
                                                assert(file.name == name, fmt.Sprintf("invalid file name: %s != %s (proj.absPath=%s, pre=%s)", file.name, name, proj.absPath, pre))
                                        } /* else if match.Paths != nil {
                                                var ( path = match.Paths[0] ; sub string )
                                                if sub, err = path.Strval(); err != nil { ctx.error("%v", err); return }
                                                if filepath.IsAbs(sub) {
                                                        file = stat(name, "", sub, nil)
                                                } else {
                                                        file = stat(name, sub, t.dir, nil)
                                                }
                                        } */
                                }
                                if file == nil {
                                        file = stat(ctx, name, t.sub, t.dir, nil/* okay missing */)
                                }

                                list = append(list, file)
                                continue ForDstPats

                        default:
                                list = append(list, MakeString(pos, name))
                                continue ForDstPats
                        }
                }
        }

        res = MakeListOrScalar(ctx.Position(), list)
        return
}

func builtinStrip(ctx Context, args... Value) (res Value) {
        return builtinTrimSpace(ctx, args...)
}

func builtinTrimSpace(ctx Context, args... Value) (res Value) {
        return builtinTrim(ctx, append([]Value{MakeNone(ctx.Position())}, args...)...)
}

func builtinTitle(ctx Context, args... Value) (res Value) {
        var ( pos = ctx.Position(); err error )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        }

        var (
                list []Value
                s string
        )
        for _, a := range args {
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("stringify '%v' failed: %v", a, err).of(a).debug(1)
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

func builtinUpperCase(ctx Context, args... Value) (res Value) {
        var (
                list []Value
                s string
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        }

        for _, a := range args {
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("stringify '%v' failed: %v", a, err).of(a).debug(1)
                        return
                } else if s != "" {
                        list = append(list, MakeString(a.Position(), strings.ToUpper(s)))
                }
        }
        if err == nil {
                res = MakeListOrScalar(ctx.Position(), list)
        }
        return
}

func builtinLowerCase(ctx Context, args... Value) (res Value) {
        var (
                list []Value
                s string
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        }

        for _, a := range args {
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("stringify '%v' failed: %v", a, err).of(a).debug(1)
                        return
                } else if s != "" {
                        list = append(list, MakeString(a.Position(), strings.ToLower(s)))
                }
        }
        if err == nil {
                res = MakeListOrScalar(ctx.Position(), list)
        }
        return
}

func builtinTrim(ctx Context, args... Value) (res Value) {
        var (
                cutset, s string
                list []Value
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }

        var pos = ctx.Position()
        for i, a := range args {
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
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

func builtinTrimLeft(ctx Context, args... Value) (res Value) {
        var (
                cutset, s string
                list []Value
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }

        for i, a := range args {
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("%v", err).at(a.Position()).debug(1)
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
                res = MakeListOrScalar(ctx.Position(), list)
        }
        return
}

func builtinTrimRight(ctx Context, args... Value) (res Value) {
        var (
                cutset, s string
                list []Value
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }

        for i, a := range args {
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
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
                res = MakeListOrScalar(ctx.Position(), list)
        }
        return
}

// $(trim-prefix foo%, fooxxx foo123)
// $(trim-prefix %/foo, xxx/foo/a/b/c)
// $(trim-prefix %%/foo, xxx/yyy/zzz/foo/a/b/c)
func builtinTrimPrefix(ctx Context, args... Value) (res Value) {
        const info = false
        var (
                prefixs, values, list []Value
                err error
        )
        if len(args) == 0 { return } else
        if prefixs, err = expandmerge2(ctx, expandPlainValue, args[0]); err != nil {
                ctx.error("merge args '%v' failed: %v", args[0], err).of(args[0]).debug(1)
                return
        }
        if len(args) == 1 {
                if len(prefixs) > 1 { values = prefixs[1:] }
        } else if values, err = expandmerge2(ctx, expandPlainValue, args[1:]...); err != nil {
                ctx.error("merge args '%v' failed: %v", args[1:], err).of(args[1]).debug(1)
                return
        }
        if len(values) == 0 { return } else if len(prefixs) == 0 {
                res = MakeListOrScalar(ctx.Position(), values)
                return
        }
        for _, value := range values {
                var (
                        pos = value.Position()
                        s string
                )
                if s, err = value.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", value, err).of(value).debug(1)
                        return
                }
        ForPrefix:
                for _, prefix := range prefixs {
                        var full, cutset, stems = prefix.match(ctx, value)
                        if info { ctx.warn("prefix=%v (%T); value=%v (%T) -> full=%v cutset=%v stems=%v", prefix, prefix, value, value, full, cutset, stems).of(prefix).debug(true, 1) }
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
                if info { ctx.warn("list=%v trimmed=%v", list, s).debug(true, 1) }
                if s != "" { list = append(list, MakeString(pos, s)) }
        }
        if err == nil { res = MakeListOrScalar(ctx.Position(), list) }
        return
}

func builtinTrimSuffix(ctx Context, args... Value) (res Value) {
        var (
                cutset, s string
                list []Value
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }

        for i, a := range args {
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
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
                res = MakeListOrScalar(ctx.Position(), list)
        }
        return
}

func builtinTrimExt(ctx Context, args... Value) (res Value) {
        var (
                list []Value
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }

        for i, a := range args {
                var ext, s string
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
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
                res = MakeListOrScalar(ctx.Position(), list)
        }
        return
}

func builtinIndent(ctx Context, args... Value) (res Value) {
        var (
                l []Value
                s string // indent
                err error
        )
        if x := len(args); x > 0 {
                if v, ok := Scalar(args[0]).(*Int); ok {
                        args, s = args[1:], strings.Repeat(" ", int(v.int64))
                } else {
                        ctx.error("requires integer argument (first|last)")
                        return
                }
        }
        for _, a := range args {
                var (
                        lines []string
                        v string
                )
                if v, err = a.Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
                for _, line := range strings.Split(v, "\n") {
                        lines = append(lines, s + line)
                }
                l = append(l, MakeString(a.Position(), strings.Join(lines, "\n")))
        }
        res = MakeListOrScalar(ctx.Position(), l)
        return
}

func builtinFindstring(ctx Context, args... Value) (res Value) {
        // TODO: $(findstring find,text)
        return
}

// $(contains a b c, v1 v2 …)
// $(contains a b c1 -or c2, v1 v2 …)
// $(contains a b c1 -or c2 -or c3, v1 v2 …)
// $(contains a b -or=(c1 c2 c3), v1 v2 …)
type builtinContainsOpts struct {
        debug bool `d,debug`
        verbose bool `v,verbose`
        string bool `s,string`
}
func builtinContains(ctx Context, args... Value) (res Value) {
        var (
                opts builtinContainsOpts
                vals []Value
                list []Value
                err error
        )
        if len(args) < 2 {
                ctx.error("unexpected number of arguments, try $(contains a b c1 -or c2, v1 v2 …)").debug(1)
                return
        }

        if vals, err = expandmerge2(ctx, expandPlainValue, args[0]); err != nil {
                ctx.error("expand args failed: %v", err).debug(1)
                return
        } else if vals, err = parseOpts(ctx, &opts, vals...); err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }
        if list, err = expandmerge2(ctx, expandPlainValue, args[1:]...); err != nil {
                ctx.error("expand args failed: %v", err).debug(1)
                return
        }

        var ( n = 0; x = len(vals); va []Value )
        for _, val := range vals {
                var s string
                switch v := val.(type) {
                default: va = []Value{ val }
                case *Flag:
                        if s, err = v.name.Strval(ctx); err != nil {
                                ctx.error("strval '%v' failed: %v", v.name, err).debug(1)
                                return
                        } else if s == "or" { va, x = append(va, val), x-1; continue }
                case *Pair: // FIXME: -or=(c1 c2 c3)
                        if f, ok := v.Key.(*Flag); !ok {va = []Value{ val }} else {
                                if s, err = f.name.Strval(ctx); err != nil {
                                        ctx.error("strval '%v' failed: %v", f.name, err).debug(1)
                                        return
                                } else if s == "or" { va, x = append(va, v.Value), x-1; continue }
                        }
                }

                if len(va) == 0 { continue }
                ForList:for _, v := range list {
                        for _, a := range va {
                                if opts.string {
                                        var r string
                                        if r, err = v.Strval(ctx); err != nil {
                                                ctx.error("strval '%v' failed: %v", v, err).debug(1)
                                                return
                                        }
                                        if s, err = a.Strval(ctx); err != nil {
                                                ctx.error("strval '%v' failed: %v", a, err).debug(1)
                                                return
                                        }
                                        if r != s { continue ForList }
                                } else if a.cmp(ctx, v) != cmpEqual { continue ForList }
                        }
                        n += 1 // one matched
                }
                va = nil
        }
        if opts.verbose {
                ctx.info("%v contains %v: %v (%v, %v)\n", list, vals, (n==x), n, x).debug(opts.debug)
        }
        res = MakeBoolean(ctx.Position(), (n == x))
        return
}

func builtinSort(ctx Context, args... Value) (res Value) {
        // TODO: $(sort list)
        return
}

func builtinWord(ctx Context, args... Value) (res Value) {
        // TODO: $(word n,text)
        return
}

func builtinWordList(ctx Context, args... Value) (res Value) {
        // TODO: $(wordlist s,e,text)
        return
}

func builtinWords(ctx Context, args... Value) (res Value) {
        // TODO: $(words n,text)
        return
}

func builtinFirstWord(ctx Context, args... Value) (res Value) {
        // TODO: $(firstword names...)
        return
}

func builtinLastWord(ctx Context, args... Value) (res Value) {
        // TODO: $(lastword names...)
        return
}

func builtinEncodeBase64(ctx Context, args... Value) (res Value) {
        if len(args) > 0 {
                pos := ctx.Position()
                buf := new(bytes.Buffer)
                enc := base64.NewEncoder(base64.StdEncoding, buf)
                for _, a := range args {
                        var ( s string; err error )
                        if s, err = a.Strval(ctx); err != nil {
                                ctx.error("strval '%v' failed: %v", a, err).debug(1)
                                return
                        }
                        enc.Write([]byte(s))
                }
                enc.Close()
                res = MakeString(pos, buf.String())
        }
        return
}

func builtinDecodeBase64(ctx Context, args... Value) (res Value) {
        if len(args) > 0 {
                var list []Value
                for _, a := range args {
                        var (
                                dat []byte
                                s string
                                err error
                        )
                        if s, err = a.Strval(ctx); err != nil {
                                ctx.error("strval '%v' failed: %v", a, err).debug(1)
                                return
                        } else if dat, err = base64.StdEncoding.DecodeString(s); err != nil {
                                ctx.error("decode '%s' failed: %v", s, err).debug(1)
                                return
                        } else {
                                list = append(list, MakeString(a.Position(), string(dat)))
                        }
                }
                res = MakeListOrScalar(ctx.Position(), list)
        }
        return
}

func asFile(a Value) (f *File) {
        switch t := a.(type) {
        case *File     : f = t
        case *Barefile : f = t.File
        case *Def      : if t.value != nil    { return asFile(t.value   ) }
        case *List     : if len(t.Elems) == 1 { return asFile(t.Elems[0]) }
        case *RuleEntry:                        return asFile(t.target  )
        }
        return
}

func fullname(ctx Context, a Value) (s string, ok bool) {
        if v, err := a.expand(ctx, expandFullName); err != nil {
                ctx.error("expand fullname failed: %v", err).of(a).debug(1)
        } else {
                if isNil(v) { v = a }
                s, ok = fullname1(v)
        }
        return
}

func fullname1(a Value) (s string, ok bool) {
        if f := asFile(a); f == nil {
                // no fullname
        } else if s = f.fullname(); filepath.IsAbs(s) {
                ok = true
        } else {
                s = ""
        }
        return
}

func fullnameOrStrval(ctx Context, a Value) (s string, err error) {
        var ok bool
        if s, ok = fullname(ctx, a); !ok {
                s, err = a.Strval(ctx)
        }
        return
}

// see optFullname and parseOpt
func asOptFullname(ctx Context, proj *Project, val Value) (rp *Project, s string, ok bool, e error) {
        if proj == nil { proj = /*current()*/ctx.Project() }
        if s, ok = fullname(ctx, val); ok {
                // done
        } else if proj == nil {
                ctx.error("no current project to find file '%v'", val).of(val).debug(1)
        } else if s, e = val.Strval(ctx); e != nil {
                ctx.error("strval '%v' failed: %v", val, e).of(val).debug(1)
        } else if filepath.IsAbs(s) {
                ok = true
        } else if file := proj.FindFile(ctx, s); file != nil {
                s, ok = file.fullname(), true
        }
        rp = proj
        return
}

type builtinFullNameOpts struct {
        debug int `d,debug`
}
func builtinFullName(ctx Context, args... Value) (res Value) {
        var (
                opts builtinFullNameOpts
                proj *Project
                l []Value
                err error
                s string
                ok bool
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }

        for _, a := range args {
                if opts.debug > 0 {
                        if f, ok := a.(*File); ok {
                                ctx.warn("dir=%v sub=%v name=%v", f.dir, f.sub, f.name).debug(opts.debug)
                        } else {
                                ctx.warn("%T %v", a, a).debug(opts.debug,1)
                        }
                }
                if proj, s, ok, err = asOptFullname(ctx, proj, a); err != nil {
                        ctx.error("fullname '%v' failed: %v", a, err).debug(1)
                        break
                } else if ok || s != "" {
                        l = append(l, MakeString(a.Position(), s))
                } else {
                        l = append(l, a)
                }
        }
        res = MakeListOrScalar(ctx.Position(), l)
        return
}

type builtinBaseOpts struct {
        debug int `d,debug`
        fullname bool `f,full;fn,fullname` // unused
}
func builtinBase(ctx Context, args... Value) (res Value) {
        var (
                opts builtinBaseOpts
                l []Value
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }

        var pos = ctx.Position()
        for _, a := range args {
                var s string
                if s, err = fullnameOrStrval(ctx, a); err != nil {
                        ctx.error("fullname '%v' failed : %v", a, err).debug(1)
                        return
                }
                l = append(l, MakeString(pos, filepath.Base(s)))
        }
        res = MakeListOrScalar(pos, l)
        return
}

func dirx(ctx Context, n int, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                l []Value
                s string
                err error
        )
        for _, a := range args {
                if s, err = fullnameOrStrval(ctx, a); err != nil {
                        ctx.error("fullname '%v' failed : %v", a, err).debug(1)
                        return
                }
                s = filepath.Dir(s)
                for i := n-1; 0 < i; i -= 1 {
                        s = filepath.Dir(s)
                }
                l = append(l, MakePathStr(pos, s))
        }
        res = MakeListOrScalar(pos, l)
        return
}

func undirx(ctx Context, n int, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                l []Value
                s string
                err error
        )
        for _, a := range args {
                if s, err = fullnameOrStrval(ctx, a); err != nil {
                        ctx.error("fullname '%v' failed : %v", a, err).debug(1)
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

type builtinDirOpts struct {
        fullname bool `f,full;fn,fullname`
}
func builtinDir(ctx Context, args... Value) (res Value) {
        var (
                opts builtinDirOpts
                proj *Project
                l []Value
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }

        var pos = ctx.Position()
        for _, a := range args {
                var s string
                if opts.fullname {
                        if proj, s, _, err = asOptFullname(ctx, proj, a); err != nil {
                                ctx.error("fullname '%v' failed: %v", a, err).debug(1)
                                break
                        }
                }
                if !opts.fullname || s == "" {
                        if s, err = a.Strval(ctx); err != nil {
                                ctx.error("strval '%v' failed: %v", a, err).debug(1)
                                return
                        }
                }
                l = append(l, MakePathStr(pos,filepath.Dir(s)))
        }
        res = MakeListOrScalar(pos, l)
        return
}

func builtinDir2(ctx Context, args... Value) (res Value) {
        return dirx(ctx, 2, args...)
}

func builtinDir3(ctx Context, args... Value) (res Value) {
        return dirx(ctx, 3, args...)
}

func builtinDir4(ctx Context, args... Value) (res Value) {
        return dirx(ctx, 4, args...)
}

func builtinDir5(ctx Context, args... Value) (res Value) {
        return dirx(ctx, 5, args...)
}

func builtinDir6(ctx Context, args... Value) (res Value) {
        return dirx(ctx, 6, args...)
}

func builtinDir7(ctx Context, args... Value) (res Value) {
        return dirx(ctx, 7, args...)
}

func builtinDir8(ctx Context, args... Value) (res Value) {
        return dirx(ctx, 8, args...)
}

func builtinDir9(ctx Context, args... Value) (res Value) {
        return dirx(ctx, 9, args...)
}

func builtinDirs(ctx Context, args... Value) (res Value) {
        var n int
        if x := len(args); x > 0 {
                if v, ok := Scalar(args[0]).(*Int); ok {
                        args, n = args[1:], int(v.int64)
                } else if v, ok := Scalar(args[x-1]).(*Int); ok {
                        args, n = args[:x-1], int(v.int64)
                } else {
                        ctx.error("require (first/last) integer argument (first=%T, last=%T)", args[0], args[x-1]).debug(1)
                        return
                }
        }
        res = dirx(ctx, n, args...)
        return
}

func builtinUndir(ctx Context, args... Value) (res Value) {
        return undirx(ctx, 1, args...)
}

func builtinUndir2(ctx Context, args... Value) (res Value) {
        return undirx(ctx, 2, args...)
}

func builtinUndir3(ctx Context, args... Value) (res Value) {
        return undirx(ctx, 3, args...)
}

func builtinUndir4(ctx Context, args... Value) (res Value) {
        return undirx(ctx, 4, args...)
}

func builtinUndir5(ctx Context, args... Value) (res Value) {
        return undirx(ctx, 5, args...)
}

func builtinUndir6(ctx Context, args... Value) (res Value) {
        return undirx(ctx, 6, args...)
}

func builtinUndir7(ctx Context, args... Value) (res Value) {
        return undirx(ctx, 7, args...)
}

func builtinUndir8(ctx Context, args... Value) (res Value) {
        return undirx(ctx, 8, args...)
}

func builtinUndir9(ctx Context, args... Value) (res Value) {
        return undirx(ctx, 9, args...)
}

func builtinUndirs(ctx Context, args... Value) (res Value) {
        var n = 0
        if x := len(args); x > 0 {
                if v, ok := Scalar(args[0]).(*Int); ok {
                        args, n = args[1:], int(v.int64)
                } else if v, ok := Scalar(args[x-1]).(*Int); ok {
                        args, n = args[:x-1], int(v.int64)
                } else {
                        ctx.error("require (first/last) integer argument (first=%T, last=%T)", args[0], args[x-1]).debug(1)
                        return
                }
        }
        return undirx(ctx, n, args...)
}

func builtinDirChop(ctx Context, args... Value) (res Value) {
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
                        ctx.error("require (first/last) integer argument (first=%T, last=%T)", args[0], args[x-1]).debug(1)
                        return

                }
        }
        for _, a := range args {
                var s string
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", err).of(a).debug(1)
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
                l = append(l, MakeString(a.Position(), filepath.Join(v...)))
        }
        res = MakeListOrScalar(ctx.Position(), l)
        return
}

func builtinRelativeDir(ctx Context, args... Value) (res Value) {
        var (
                err error
                l []Value
                t, s string
        )
        for i, a := range args {
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("%v", err)
                        return
                }
                if i == 0 {
                        t = s
                } else if s, err = filepath.Rel(t, s); err == nil {
                        l = append(l, MakeString(a.Position(), s))
                } else {
                        ctx.error("%v", err)
                        return
                }
        }
        res = MakeListOrScalar(ctx.Position(), l)
        return
}

func builtinMkdir(ctx Context, args... Value) (res Value) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name string
                        perm os.FileMode
                        err error
                )
                switch t := a.(type) {
                case *Pair: // mkdir name => perm name => perm
                        if name, err = t.Key.Strval(ctx); err != nil { ctx.error("%v", err); return }
                        if perm, err = permVal(ctx, t.Value,0600); err != nil { ctx.error("%v", err); return }
                case *Group: // mkdir (name perm) (name perm)
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(ctx); err != nil { ctx.error("%v", err); return }
                                if perm, err = permVal(ctx, t.Get(1),0600); err != nil { ctx.error("%v", err); return }
                        } else {
                                ctx.error("Wrong size of list `%v'", t)
                                break
                        }
                case *List: // mkdir name perm, name perm, ...
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(ctx); err != nil { ctx.error("%v", err); return }
                                if perm, err = permVal(ctx, t.Get(1),0600); err != nil { ctx.error("%v", err); return }
                        } else {
                                ctx.error("Wrong size of list `%v'", t)
                                break
                        }
                default: // mkdir name perm, name perm, ...
                        if name, err = args[i].Strval(ctx); err != nil { ctx.error("%v", err); return }
                        if i+1 < nargs {
                                if perm, err = permVal(ctx, args[i+1],0600); err != nil { ctx.error("%v", err); return }
                                i += 1
                        }
                }
                if err = os.Mkdir(name, perm); err != nil { ctx.error("%v", err); break }
        }
        return
}

func builtinMkdirAll(ctx Context, args... Value) (res Value) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name string
                        perm os.FileMode
                        err error
                )
                switch t := a.(type) {
                case *Pair: // mkdir name => perm name => perm
                        if name, err = t.Key.Strval(ctx); err != nil { ctx.error("%v", err).of(t.Key); return }
                        if perm, err = permVal(ctx, t.Value,0600); err != nil { ctx.error("%v", err).of(t.Value); return }
                case *Group: // mkdir (name perm) (name perm)
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(ctx); err != nil { ctx.error("%v", err).of(t.Get(0)); return }
                                if perm, err = permVal(ctx, t.Get(1),0600); err != nil { ctx.error("%v", err).of(t.Get(1)); return }
                        } else {
                                ctx.error("Wrong size of list `%v'", t);
                                break
                        }
                case *List: // mkdir name perm, name perm, ...
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(ctx); err != nil { ctx.error("%v", err).of(t.Get(0)); return }
                                if perm, err = permVal(ctx, t.Get(1),0600); err != nil { ctx.error("%v", err).of(t.Get(1)); return }
                        } else {
                                ctx.error("Wrong size of list `%v'", t);
                                break
                        }
                default: // mkdir name perm, name perm, ...
                        if name, err = args[i].Strval(ctx); err != nil { ctx.error("%v", err).of(args[i]); return }
                        if i+1 < nargs {
                                if perm, err = permVal(ctx, args[i+1],0600); err != nil { ctx.error("%v", err).of(args[i+1]); return }
                                i += 1
                        }
                }
                if err = os.MkdirAll(name, perm); err != nil { ctx.error("%v", err); break }
        }
        return
}

func builtinChdir(ctx Context, args... Value) (res Value) {
        if len(args) == 1 {
                var ( str string; err error )
                if str, err = args[0].Strval(ctx); err != nil { ctx.error("%v", err).of(args[0]); return }
                if err = lockCD(str, 0); err != nil { ctx.error("%v", err) }
        } else {
                ctx.error("wrong number of arguments: %v", len(args))
        }
        return
}

type builtinRenameOpts struct {
        // TODO: ...
}
func builtinRename(ctx Context, args... Value) (res Value) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        oldname, newname string
                        err error
                )
                switch t := a.(type) {
                case *Pair: // rename oldname=newname
                        if oldname, err = t.Key.Strval(ctx);   err != nil { ctx.error("%v", err).of(t.Key); return }
                        if newname, err = t.Value.Strval(ctx); err != nil { ctx.error("%v", err).of(t.Value); return }
                case *Group: // rename (oldname newname) (old new)
                        if t.Len() == 2 {
                                if oldname, err = t.Get(0).Strval(ctx); err != nil { ctx.error("%v", err).of(t.Get(0)); return }
                                if newname, err = t.Get(1).Strval(ctx); err != nil { ctx.error("%v", err).of(t.Get(1)); return }
                        } else {
                                ctx.error("wrong size of group `%v'", t).of(t)
                                break
                        }
                case *List: // rename oldname newname, old new, ...
                        if t.Len() == 2 {
                                if oldname, err = t.Get(0).Strval(ctx); err != nil { ctx.error("%v", err).of(t.Get(0)); return }
                                if newname, err = t.Get(1).Strval(ctx); err != nil { ctx.error("%v", err).of(t.Get(1)); return }
                        } else {
                                ctx.error("wrong size of list `%v'", t).of(t)
                                break
                        }
                default: // rename newname oldname  newname oldname ...
                        if i+1 < nargs {
                                if oldname, err = args[i+0].Strval(ctx); err != nil { ctx.error("%v", err).of(args[i+0]); return }
                                if newname, err = args[i+1].Strval(ctx); err != nil { ctx.error("%v", err).of(args[i+1]); return }
                                i += 1
                        } else {
                                ctx.error("Wrong arguments `%v'", args).of(t)
                                break
                        }
                }
                if err = os.Rename(oldname, newname); err != nil {
                        ctx.error("%v", err).at(ctx.Position()).debug(1)
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
func builtinRemove(ctx Context, args... Value) (res Value) {
        var (
                opts builtinRemoveOpts
                names []string
                proj *Project
                str string
                err error
                ok bool
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err)
                return
        }

        for _, a := range args {
                if isNil(a) || isNone(a) {
                        // ignore
                } else if a.patterned(ctx) {
                        if str, err = a.Strval(ctx); err != nil { ctx.error("%v", err).of(a).debug(true, 1); return }
                        if names, err = filepath.Glob(str); err != nil { ctx.error("%v", err).of(a).debug(true, 1); return }
                        for _, s := range names {
                                if opts.verbose { ctx.prompt("remove %s\n", s) }
                                if opts.debug   { ctx.info("remove %s", s).debug(1) }
                                if opts.all {
                                        err = os.RemoveAll(s)
                                } else {
                                        err = os.Remove(s)
                                }
                                if err != nil {
                                        ctx.error("remove failed: %v", err).of(a)
                                        return
                                }
                        }
                } else if proj, str, ok, err = asOptFullname(ctx, proj, a); err != nil {
                        ctx.error("fullname '%v' failed: %v", a, err).of(a)
                        ctx.error("internal stack:").debug(16)
                        return
                } else if !ok || str == "" {
                        ctx.error("remove failed: %v (%T)", a, a).of(a)
                        ctx.error("remove failed: %v", str).of(a)
                        ctx.error("internal stack:").debug(16)
                        break
                } else {
                        if opts.verbose { ctx.prompt("remove %s\n", str) }
                        if opts.debug   { ctx.info("remove %s", str).debug(1) }
                        if opts.all {
                                err = os.RemoveAll(str)
                        } else {
                                err = os.Remove(str)
                        }
                        if err != nil {
                                ctx.error("%v", err)
                                ctx.error("source: %v (%T)", a, a).of(a)
                                ctx.error("source: %v", str).of(a).debug(1)
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
func builtinRemoveAll(ctx Context, args... Value) (res Value) {
        var (
                opts builtinRemoveAllOpts
                names []string
                proj *Project
                str string
                err error
                ok bool
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }

        for _, a := range args {
                if a.patterned(ctx) {
                        if str, err = a.Strval(ctx); err != nil {
                                ctx.error("%v", err).of(a).debug(1)
                                return
                        } else if names, err = filepath.Glob(str); err != nil {
                                ctx.error("%v", err).of(a).debug(1)
                                return
                        }
                        for _, s := range names {
                                if opts.verbose { ctx.info("remove %s", s).at(a.Position()) }
                                if err = os.RemoveAll(s); err != nil {
                                        ctx.error("%v", err).of(a).debug(1)
                                        return
                                }
                        }
                } else if proj, str, ok, err = asOptFullname(ctx, proj, a); err != nil {
                        ctx.error("remove failed: %v", err).of(a).debug(1)
                        return
                } else if !ok || str == "" {
                        ctx.error("%v is not a file", a).of(a).debug(1)
                        break
                } else {
                        if opts.verbose { ctx.info("remove %s", str) }
                        if opts.debug   { ctx.info("remove %s", str).debug(1) }
                        if err = os.RemoveAll(str); err != nil {
                                ctx.error("remove failed: %v", err).of(a).debug(1)
                                return
                        }
                }
        }
        return
}

func builtinTruncate(ctx Context, args... Value) (res Value) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name string
                        size int64
                        err error
                )
                switch t := a.(type) {
                case *Pair: // truncate name => size old => new
                        if name, err = t.Key.Strval(ctx); err != nil {
                                ctx.error("%v", err).debug(1)
                                return
                        } else if size, err = t.Value.Integer(ctx); err != nil {
                                ctx.error("%v", err).debug(1)
                                return
                        }
                case *Group: // truncate (name size) (old new)
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(ctx); err != nil { ctx.error("%v", err).debug(1); return }
                                if size, err = t.Get(1).Integer(ctx); err != nil { ctx.error("%v", err).debug(1); return }
                        } else {
                                ctx.error("Wrong size of group `%v'", t).debug(1)
                                break
                        }
                case *List: // truncate name size, old new, ...
                        if t.Len() == 2 {
                                if name, err = t.Get(0).Strval(ctx); err != nil { ctx.error("%v", err).debug(1); return }
                                if size, err = t.Get(1).Integer(ctx); err != nil { ctx.error("%v", err).debug(1); return }
                        } else {
                                ctx.error("Wrong size of list `%v'", t).debug(1)
                                break
                        }
                default: // truncate name size  name size ...
                        if i+1 < nargs {
                                if name, err = args[i+0].Strval(ctx); err != nil { ctx.error("%v", err).debug(1); return }
                                if size, err = args[i+1].Integer(ctx); err != nil { ctx.error("%v", err).debug(1); return }
                                i += 1
                        } else {
                                ctx.error("Wrong arguments `%v'", args).debug(1)
                                break
                        }
                }
                if err = os.Truncate(name, size); err != nil {
                        ctx.error("%v", err).debug(1)
                        break
                }
        }
        return
}

type builtinLinkOpts struct {
        // TODO: ...
}
func builtinLink(ctx Context, args... Value) (res Value) {
        var (
                opts builtinLinkOpts
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        oldname, newname string
                        a = args[i]
                )
                switch t := a.(type) {
                case *Pair: // link oldname => newname old => new
                        if oldname, err = t.Key.Strval(ctx); err != nil { ctx.error("%v", err); return }
                        if newname, err = t.Value.Strval(ctx); err != nil { ctx.error("%v", err); return }
                case *Group: // link (oldname newname) (old new)
                        if t.Len() == 2 {
                                if oldname, err = t.Get(0).Strval(ctx); err != nil { ctx.error("%v", err); return }
                                if newname, err = t.Get(1).Strval(ctx); err != nil { ctx.error("%v", err); return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // link oldname newname, old new, ...
                        if t.Len() == 2 {
                                if oldname, err = t.Get(0).Strval(ctx); err != nil { ctx.error("%v", err); return }
                                if newname, err = t.Get(1).Strval(ctx); err != nil { ctx.error("%v", err); return }
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // link oldname newname  oldname newname ...
                        if i+1 < nargs {
                                if oldname, err = args[i+0].Strval(ctx); err != nil { ctx.error("%v", err); return }
                                if newname, err = args[i+1].Strval(ctx); err != nil { ctx.error("%v", err); return }
                                i += 1
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong arguments `%v'", args))
                                break
                        }
                }
                if err = os.Link(oldname, newname); err != nil {
                        ctx.error("%v", err).debug(1)
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
func builtinSymlink(ctx Context, args... Value) (res Value) {
        var (
                opts builtinSymlinkOpts
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }
ForArgs:
        for i, na := 0, len(args); i < na; i += 1 {
                var oldNameVal, newNameVal Value
                switch t := args[i].(type) {
                case *Pair: // symlink oldname=newname oldname=>newname...
                        oldNameVal, newNameVal = t.Key, t.Value
                case *Group: // symlink (oldname newname) (oldname newname)...
                        if t.Len() != 2 {
                                ctx.error("expects two values of group").of(t).debug(1)
                                return
                        }
                        oldNameVal, newNameVal = t.Get(0), t.Get(1)
                case *List: // symlink oldname newname, old new, ...
                        if t.Len() != 2 {
                                ctx.error("expects two values of list").of(t).debug(1)
                                return
                        }
                        oldNameVal, newNameVal = t.Get(0), t.Get(1)
                default:// Multiple pairs of names:
                        // symlink  newname oldname  newname oldname ...
                        if i+1 < na {
                                oldNameVal, newNameVal = args[i+0], args[i+1]
                                i += 1
                        } else {
                                ctx.error("expects pair of names (%v)", args[i]).of(args[i]).debug(1)
                                return
                        }
                }

                var oldname, newname string
                if oldname, err = oldNameVal.Strval(ctx); err != nil {
                        ctx.error("%v", err).of(oldNameVal).debug(1)
                        return
                }
                if newname, err = newNameVal.Strval(ctx); err != nil {
                        ctx.error("%v", err).of(newNameVal).debug(1)
                        return
                }

                if newname == "" {
                        ctx.error("empty new filename").debug(1)
                        return
                }
                if oldname == "" {
                        ctx.error("empty old filename (%v)", ).debug(1)
                        return
                }

                if opts.force {
                        if err = os.Remove(newname); err != nil {
                                ctx.error("%v", err).debug(1)
                                err = nil //return
                        }
                } else if opts.update {
                        var s string
                        if s, err = os.Readlink(newname); err != nil {
                                ctx.error("%v", err).debug(1)
                                err = nil //continue ForArgs
                        } else if s == newname {
                                continue ForArgs
                        } else if err = os.Remove(newname); err != nil {
                                ctx.error("%v", err).debug(1)
                                err = nil //return
                        }
                }
                if opts.verbose {
                        var d = filepath.Base(newname)
                        var s = filepath.Base(oldname)
                        ctx.prompt("Symlink %s -> %s …", d, s)
                }
                if opts.relative {
                        var dir = filepath.Dir(newname)
                        oldname, err = filepath.Rel(dir, oldname)
                        if err != nil {
                                if opts.verbose { ctx.prompt("symlink: %s\n", err) }
                                ctx.error("%v", err).debug(1)
                                return
                        }
                }
                if dir := filepath.Dir(newname); opts.path && dir != "." && dir != PathSep {
                        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                                ctx.error("%v", err).debug(1)
                                return
                        }
                }
                if err = os.Symlink(oldname, newname); err != nil {
                        if opts.verbose { ctx.prompt("… %s\n", err) }
                        break
                } else if opts.verbose {
                        ctx.prompt("… ok\n")
                }
        }
        return
}

type builtinStatOpts struct {
        dir bool `d,dir`
        file bool `f,file`
        symbol bool `s,symlink;sym,symbol`
}
func builtinStat(ctx Context, args... Value) (res Value) {
        var (
                proj = ctx.Project()
                opts builtinStatOpts
                err error
        )
        if proj == nil {
                ctx.error("unknown current context").debug(1)
                return
        } else if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }

        var (
                pos = ctx.Position()
                valF = MakeBoolean(pos, false)
                valT = MakeBoolean(pos, true)
                reses []Value
        )
        var check = func(file *File) {
                if file == nil || file.info == nil {
                        reses = append(reses, valF)
                } else if mode := file.info.Mode(); opts.dir && mode&os.ModeDir != 0 { // IsDir()
                        reses = append(reses, valT)//file
                } else if opts.symbol && mode&os.ModeSymlink != 0 {
                        reses = append(reses, valT)//file
                } else if opts.file && mode&os.ModeType != 0 { // IsRegular()
                        reses = append(reses, valT)//file
                } else {
                        reses = append(reses, valT)//file
                }
        }

        var checkstat = func(a Value) {
                var (
                        file *File
                        s string
                )
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", a, err).debug(1)
                        return
                }
                if filepath.IsAbs(s) {
                        file = stat(ctx, s, "", "")
                } else {
                        file = stat(ctx, s, "", proj.absPath)
                }
                if file == nil { file = proj.FindFile(ctx, s) }
                if file != nil { check(file) }
                if false && strings.Contains(s, "polly") {
                        ctx.warn("%v: %v", proj, file)
                        ctx.warn("%v: %v (%T)", proj, a, a)
                        ctx.warn("%v: %v", proj, ctx).debug(1)
                }
        }

        for _, a := range args {
                switch t := a.(type) {
                case *File: check(t)
                case *Path: checkstat(a)
                default:    checkstat(a)
                }
        }

        if err == nil {
                res = MakeListOrScalar(pos, reses)
        }
        return
}

/*func builtinFileSource(ctx Context, args... Value) (res Value) {
        var ( pos = ctx.Position(); err error )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("expand args failed: %v", err).debug(1)
                return
        }

        var proj = ctx.Project()//current()
        if proj == nil {
                ctx.error("unknown current context").debug(1)
                return
        }

        var l []Value
        for _, a := range args {
                var str string
                if str, err = a.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", err).debug(1)
                        return
                }
                if file := proj.FindFile(ctx, str); file != nil {
                        l = append(l, MakeString(a.Position(), file.sub))
                }
        }
        if err == nil {
                res = MakeListOrScalar(ctx.Position(), l)
        }
        return
}*/

type builtinFileOpts struct {
        caller bool `c,caller;cc,callercontext;cc,caller-context`
        report bool `r,report;r,reportmissing;rm,report-missing;e,error`
}
func builtinFile(ctx Context, args... Value) (res Value) {
        var (
                opts builtinFileOpts
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("expand args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }

        var (
                proj = ctx.Project()
                list []Value
        )
        for _, a := range args {
                var str string
                if file, ok := a.(*File); ok {
                        list = append(list, file)
                        if file.exists() { continue }
                        if opts.report { ctx.info("%v is no such file", a).debug(1) }
                } else if str, err = a.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", err).debug(1)
                        return
                } else if file = proj.FindFile(ctx, str); file != nil {
                        list = append(list, file)
                        if opts.report { ctx.info("%v is no such file", a).debug(1) }
                } else {
                        ctx.error("%v is not a file in %v (%v)", a, proj, args).debug(1)
                }
        }

        res = MakeListOrScalar(ctx.Position(), list)
        return
}

type builtinGlobOpts struct {
        dir bool `d,dir;d,directory`
        file bool `f,file`
        symbol bool `s,symlink;sym,symbol;sym,symbolic`
}
func builtinGlob(ctx Context, args... Value) (res Value) {
        var (
                opts builtinGlobOpts
                proj *Project
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("expand args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }

        var cwd string // TODO: get current work directory
        if proj = ctx.Project(); proj == nil {
                ctx.error("unknown current cntext").debug(1)
                return
        }

        var pos = ctx.Position()
        var list []Value
        for _, a := range args {
                var ( str string; names []string )
                if str, err = a.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", err).debug(1)
                        return
                } else if !filepath.IsAbs(str) {
                        str = filepath.Join(cwd, str)
                } 
                if names, err = filepath.Glob(str); err != nil {
                        ctx.error("glob '%v' failed: %v", str, err).debug(1)
                        return
                }
                for _, name := range names {
                        //var fi, _ = os.Stat(name)
                        // TODO: opts.dir, opts.file, opts.symbol
                        list = append(list, MakePathStr(pos, name))
                }
        }
        return MakeListOrScalar(pos, list)
}

type wildcardOpts struct {
        includeMissing bool `im,includemissing;m,include-missing`
        errorMissing bool `em,errormissing;e,error-missing`
        verbose bool `v,verbose`
}
func builtinWildcard(ctx Context, args... Value) (res Value) {
        var (
                proj = /*current()*/ctx.Project()
                opts wildcardOpts
                files []*File
                err error
        )
        if proj == nil {
                ctx.error("unknown most derived context").debug(1)
                return
        }

        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        } else if files, err = proj.wildcard(ctx, opts, args...); err == nil {
                res = MakeListOrScalar(ctx.Position(), values(files))
        } else {
                ctx.error("wildcard failed: %v", err).debug(1)
        }
        return
}

func builtinReadDir(ctx Context, args... Value) (res Value) {
        var l []Value
        for _, a := range args {
                var (
                        fis []os.FileInfo
                        str string
                        err error
                )
                if str, err = a.Strval(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
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
                res = MakeListOrScalar(ctx.Position(), l)
        }
        return
}

type builtinReadFileOpts struct {
        trim bool `t,trim;ta,trim-all`
        trimLeft bool `tl,trim-left`
        trimRight bool `tr,trim-right`
}
func builtinReadFile(ctx Context, args... Value) (res Value) {
        var (
                opts builtinReadFileOpts
                proj *Project
                err error
                l []Value
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        }
        if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }
        var pos = ctx.Position()
        for _, a := range args {
                var (
                        apos = a.Position()
                        str string
                        err error
                        s []byte
                        ok bool
                )
                if !apos.IsValid() { apos = pos }
                if proj, str, ok, err = asOptFullname(ctx, proj, a); err != nil {
                        ctx.error("fullname '%v' failed: %v", a, err).debug(1)
                        break
                } else if !ok || str == "" {
                        ctx.error("%v is not a file", a).at(apos).debug(1)
                        break
                } else if s, err = ioutil.ReadFile(str); err != nil {
                        ctx.error("read file failed: %v", err).at(apos).debug(1)
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
func builtinWriteFile(ctx Context, args... Value) (res Value) {
        // $(write-file filename,content)
        // $(write-file -p filename,content)
        var (
                opts builtinWriteFileOpts
                va []Value
                err error
        )
        if len(args) > 0 {
                if va, err = expandmerge2(ctx, expandPlainValue, args[1]); err != nil {
                        ctx.error("merge args failed: %v", err)
                        return
                } else if va, err = parseOpts(ctx, &opts, va...) ; err != nil {
                        ctx.error("parse opts failed: %v", err)
                        return
                }
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
                        if name, err = t.Key.Strval(ctx); err != nil { ctx.error("%v", err); return }
                        if data, err = t.Value.Strval(ctx); err != nil { ctx.error("%v", err); return }
                case *Group: // write-file (name text) (name text 0660)
                        if n := t.Len(); n < 4 && n > 0 {
                                if name, err = t.Get(0).Strval(ctx); err != nil { ctx.error("%v", err); return }
                                if n > 1 { if data, err = t.Get(1).Strval(ctx); err != nil { ctx.error("%v", err); return }}
                                if n > 2 { if perm, err = permVal(ctx, t.Get(2),0600); err != nil { ctx.error("%v", err); return }}
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of group `%v'", t))
                                break
                        }
                case *List: // write-file name text, name text 0660, ...
                        if n := t.Len(); n < 4 && n > 0 {
                                if name, err = t.Get(0).Strval(ctx); err != nil { ctx.error("%v", err); return }
                                if n > 1 { if data, err = t.Get(1).Strval(ctx); err != nil { ctx.error("%v", err); return }}
                                if n > 2 { if perm, err = permVal(ctx, t.Get(2),0600); err != nil { ctx.error("%v", err); return }}
                        } else {
                                err = errors.New(fmt.Sprintf("Wrong size of list `%v'", t))
                                break
                        }
                default: // write-file name text 0660  name text 0660 ...
                        if name, err = args[i].Strval(ctx); err != nil { ctx.error("%v", err); return }
                        if i+1 < len(args) {
                                if data, err = args[i+1].Strval(ctx); err != nil { ctx.error("%v", err); return }
                                i += 1
                        }
                        if i+1 < len(args) {
                                if perm, err = permVal(ctx, args[i+1],0600); err != nil { ctx.error("%v", err); return }
                                i += 1
                        }
                }
                if name == "" {
                        continue ForArgs
                } else if dir := filepath.Dir(name); opts.path && dir != "." && dir != PathSep {
                        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil { ctx.error("%v", err); return }
                }
                if err = ioutil.WriteFile(name, []byte(data), perm); err != nil {
                        ctx.error("%v", err); break
                }
        }
        return
}

func touch(ctx Context, file Value, optMode uint32, optPath bool, ts ...time.Time) (err error) {
        var filename, _ = fullname(ctx, file)
        if  filename == "" {
                ctx.error("touch: file fullname of '%v' is empty", file, err).of(file).debug(1)
                return
        }

        if dir := filepath.Dir(filename); optPath && dir != "." && dir != PathSep {
                if err = os.MkdirAll(dir, os.FileMode(optMode|0733)); err != nil {
                        ctx.error("touch: %v", err).of(file).debug(1)
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
                        ctx.error("touch: %v", err).of(file).debug(1)
                } else if err = f.Close(); err != nil {
                        ctx.error("touch: %v", err).of(file).debug(1)
                }
        }
        if err == nil {
                if err = os.Chtimes(filename, at, mt); err != nil {
                        ctx.error("touch: %v", err).of(file).debug(1)
                }
        }
        if err == nil && mode != 0 && m != 0 && mode != m {
                if err = os.Chmod(filename, mode); err != nil {
                        ctx.error("touch: %v", err).of(file).debug(1)
                }
        }
        return
}

type builtinTouchFileOpts struct {
        path bool `p,path`
        mode os.FileMode `m,mode;fm,filemode;fm,file-mode`
}
func builtinTouchFile(ctx Context, args... Value) (res Value) {
        // $(touch-file filename)
        // $(touch-file -p filename)
        var (
                opts = builtinTouchFileOpts{ mode: os.FileMode(0600) }
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }
        for i := 0; i < len(args); i += 1 {
                if err = touch(ctx, args[i], uint32(opts.mode), opts.path); err != nil {
                        ctx.error("%v", err).debug(1)
                        break
                }
        }
        return
}

// $(grep 'status=1',$@)
// $(grep -1 'status=1',$@)
func builtinGrep(ctx Context, args... Value) (res Value) {
        var (
                vals, list []Value
                linesPos, linesNeg []int
                rxs []*regexp.Regexp
                err error
        )
        if len(args) != 2 {
                ctx.error("wants exactly 2 args, e.g. $(grep -1 '^example$',$(file))").debug(1)
                return
        } else if vals, err = expandmerge2(ctx, expandPlainValue, args[0]); err != nil {
                 ctx.error("%v", err).debug(1)
                 return
        }
        for _, a := range vals {
                if i, ok := a.(*Int); ok {
                        if i.int64 > 0 {
                                linesPos = append(linesPos, int(i.int64))
                        } else if i.int64 < 0{
                                linesNeg = append(linesNeg, int(i.int64))
                        } else {
                                ctx.error("zero line number").of(a).debug(1)
                                return
                        }
                } else if s, e := a.Strval(ctx); e != nil {
                        ctx.error("%v", e).of(a); return
                } else if s == "" {
                        ctx.error("empty regexp").of(a); return
                } else if r, e := regexp.Compile(s); e != nil {
                        ctx.error("%v", e).of(a); return
                } else {
                        rxs = append(rxs, r)
                }
        }

        if vals, err = expandmerge2(ctx, expandPlainValue, args[1:]...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }

        var pos = ctx.Position()
        for _, a := range vals {
                var file *os.File
                var filename string
                if filename, err = a.Strval(ctx); err != nil {
                        ctx.error("%v", err).of(a).debug(1)
                        return
                }
                if file, err = os.Open(filename); err != nil {
                        ctx.error("%v", err).of(a).debug(1)
                        return
                }
                defer file.Close()

                var (
                        line int // line number
                        greps = make(map[int][]string,2)
                        scanner = bufio.NewScanner(file)
                )
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

func (project *Project) config(ctx Context, name string) (def *Def, err error) {
        var obj Object
        if obj, err = project.resolveObject(ctx, name); err == nil && !isNil(obj) { def, _ = obj.(*Def) }
        if false && def != nil { fmt.Fprintf(stderr, "%s: %s: %v\n", project, def.position, def) }
        return
}

func (project *Project) configExpand(ctx Context, s string) (result string, err error) {
        var (
                res = new(bytes.Buffer)
                index = 0
        )
        for _, m := range rxConfigRef.FindAllStringSubmatchIndex(s, -1) {
                fmt.Fprint(res, s[index:m[0]])
                index = m[1] // reset index immediately to keep forward

                var name string
                switch {
                case m[2] > m[0] && m[3] > m[2]: name = s[m[2]:m[3]] // ${VAR}
                case m[4] > m[0] && m[5] > m[4]: name = s[m[4]:m[5]] // @VAR@
                }

                var def *Def
                if def, err = project.config(ctx, name); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                } else if def != nil {
                        var val = def.Call(ctx)
                        if isNil(val) || isNone(val) { continue }
                        switch t := val.(type) {
                        case *Plain: fmt.Fprintf(res, "%s", t.Value)
                        case *answer, *boolean:
                                var v int64
                                if v, err = t.Integer(ctx); err != nil {
                                        ctx.error("%v", err).debug(1)
                                        return
                                }
                                fmt.Fprintf(res, "%d", v)
                        case *Group:
                                var v string
                                if v, err = parseGroupValue(ctx, t).Strval(ctx); err != nil {
                                        ctx.error("%v", err).debug(1)
                                        return
                                }
                                fmt.Fprintf(res, "%s", v)
                        default:
                                var v string
                                if v, err = val.Strval(ctx); err != nil {
                                        ctx.error("%v", err).debug(1)
                                        return
                                }
                                fmt.Fprintf(res, "%s", v)
                        }
                }
        }
        if index < len(s) { fmt.Fprint(res, s[index:]) }
        result = res.String()
        return
}

func configure(ctx Context, out *bytes.Buffer, project *Project, str string) (err error) {
        var index = 0
        if str, err = project.configExpand(ctx, str); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }
        for _, m := range rxConfigure.FindAllStringSubmatchIndex(str, -1) {
                if _, err = out.WriteString(str[index:m[0]]); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                }
                index = m[1] // reset index immediately to keep forward

                var t bool
                var s string
                var verb = str[m[2]:m[3]]
                var name = str[m[4]:m[5]]
                var hasv = m[6] > m[0] && m[7] > m[6]
                var def *Def
                if def, err = project.config(ctx, name); err != nil {
                        ctx.error("config '%s' failed: %v", name, err).debug(1)
                        return
                } else if def == nil {
                        // ...
                } else if t, err = def.True(ctx); err != nil {
                        ctx.error("truthify '%v failed: %v", def, err).debug(1)
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
                        } else if va, _, err = expandall2(ctx, expandPlainValue, def.value); err != nil {
                                ctx.error("expand '%v' failed: %v", def.value, err).of(def.value)
                                return
                        } else if len(va) == 1 {
                                switch v := va[0].(type) {
                                case *answer, *boolean:
                                        if b, e := v.True(ctx); e != nil {
                                                ctx.error("truthify '%v' failed: %v", v, e).of(def.value)
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

                if _, err = out.WriteString(s); err != nil { ctx.error("%v", err); return }
        }
        if index < len(str) { _, err = out.WriteString(str[index:]) }
        return
}

func builtinReturn(ctx Context, args... Value) Value {
        return &returner{valbase{ctx.Position()}, args }
}
