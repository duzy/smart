//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/token"
        "encoding/base64"
        "path/filepath"
        // "hash/crc64"
        "io/ioutil"
        "net/http"
        "os/exec"
        goctx "context"
        "reflect"
        "strings"
        "strconv"
        "unicode"
        "unsafe"
        // "errors"
        "regexp"
        "bytes"
        "bufio"
        "time"
        "fmt"
        "os"
        "io"
)

type Position token.Position

func (pos *Position) Same(other *Position) bool {
        return (*token.Position)(pos).Same((*token.Position)(other))
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

        `debug`:        builtinDebug,
        `error`:        builtinError,
        //`warning`:      builtinWarning,

        //`assert`: builtinAssert,

        `defor`:        builtinDefor,
        `or`:           builtinOr,
        `and`:          builtinAnd,
        //`xor`:          builtinXor,
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
        // DEPRECATED: `call`:         builtinCall,
        //`auto`:         builtinAuto,
        `closure`:      builtinClosure,
        `value`:        builtinValue,
        `var`:          builtinValue,
        `list`:         builtinList,
        `sure-value`:   builtinSureValue,

        `shell`:        builtinShell,
        `which`:        builtinWhich,

        `serve-http`:   builtinServeHttp,
        `serve-https`:  builtinServeHttps,

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
        `strings`:      builtinStrings,
        `strip`:        builtinStrip,
        `trim`:         builtinTrim,
        `trim-space`:   builtinTrimSpace,
        `trim-left`:    builtinTrimLeft,
        `trim-right`:   builtinTrimRight,
        `trim-prefix`:  builtinTrimPrefix,
        `trim-suffix`:  builtinTrimSuffix,
        `trim-ext`:     builtinTrimExt,

        `printf`:       builtinPrintf,

        `ext`:          builtinExt,

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
        `fullname`:   builtinFullName,

        `base`:       builtinBase,
        `base2`:      builtinBase2,
        `base3`:      builtinBase3,
        `base4`:      builtinBase4,
        `base5`:      builtinBase5,
        `base6`:      builtinBase6,
        `base7`:      builtinBase7,
        `base8`:      builtinBase8,
        `base9`:      builtinBase9,

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
        // `mkdir`:      builtinMkdir,     // os/file.go
        // `mkdir-all`:  builtinMkdirAll,  // os/path.go
        // `chdir`:      builtinChdir,     // os/file.go
        // `rename`:     builtinRename,    // os/file.go
        // `remove`:     builtinRemove,    // os/file_*.go
        // `remove-all`: builtinRemoveAll, // os/path.go
        // `truncate`:   builtinTruncate,  // os/file_*.go
        // `link`:       builtinLink,      // os/file_*.go
        // `symlink`:    builtinSymlink,   // os/file_*.go

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
        //`printf`:       builtinPrintf,

        `assert`:       builtinAssert,

        //`error`:        builtinError,
        `warning`:      builtinWarning,

        `append`:       builtinAppend,

        //`read-dir`:     builtinReadDir,   // io/ioutil/ioutil.go
        //`read-file`:    builtinReadFile,  // io/ioutil/ioutil.go
        `write-file`:   builtinWriteFile, // io/ioutil/ioutil.go
        `touch-file`:   builtinTouchFile,

        `mkdir`:        builtinMkdir,     // os/file.go
        `mkdir-all`:    builtinMkdirAll,  // os/path.go
        `chdir`:        builtinChdir,     // os/file.go
        `rename`:       builtinRename,    // os/file.go
        `remove`:       builtinRemove,    // os/file_*.go
        `remove-all`:   builtinRemoveAll, // os/path.go
        `truncate`:     builtinTruncate,  // os/file_*.go
        `link`:         builtinLink,      // os/file_*.go
        `symlink`:      builtinSymlink,   // os/file_*.go

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

func EscapedString(ctx Context, v Value) (s string) {
        if p, ok := v.(*String); ok {
                s = strings.Replace(p.Strval(ctx), "\\'", "'", -1)
        } else {
                s = v.Strval(ctx)
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

func parseOpt(ctx Context, tag reflect.StructTag, field reflect.Value, args... Value) (rest []Value) {
        var (
                val = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
                opts []string // opt names
                s string
                ok bool
        )
        if tag == "" { return args }
        if t := string(tag)[:]; t != "" {
                const seps = ",; "
                for {
                        if i := strings.IndexAny(t, seps); i >= 0 {
                                opts = append(opts, t[:i])
                                t = t[i+1:]
                        } else {
                                opts = append(opts, t)
                                break
                        }
                }
        }

        var set func(reflect.Value, Value)
        set = func(val reflect.Value, v Value) {
                switch val.Kind() {
                case reflect.Bool:
                        val.SetBool(v.True(ctx))
                case reflect.Float32, reflect.Float64:
                        if t, e := v.Float(ctx); e == nil { val.SetFloat(t) } else {
                                erro(ctx, "%v: %v", v, e).debug(1)
                        }
                case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
                        if t, e := v.Integer(ctx); e == nil { val.SetInt(t) } else {
                                erro(ctx, "%v: %v", v, e).debug(1)
                        }
                case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
                        if t, e := v.Integer(ctx); e == nil { val.SetUint(uint64(t)) } else {
                                erro(ctx, "%v: %v", v, t).debug(1)
                        }
                 case reflect.String:
                        val.SetString(v.Strval(ctx))
                case reflect.Slice:
                        if tp := reflect.New(val.Type().Elem()); tp.Kind() == reflect.Ptr {
                                var tv = tp.Elem()
                                set(tv, v)
                                val.Set(reflect.Append(val, tv))
                        }
                case reflect.Interface: switch val.Type().String() {
                case "smart.Value": val.Set(reflect.ValueOf(v))
                default: erro(ctx, "option type unsupported: %T %v -> %v, %v", v, v, val.Kind(), val.Type()).
                        of(v).debug(1)
                }
                case reflect.Ptr: switch val.Type().Elem().String() {
                case "smart.optFullname":
                        var x Value
                        if x = v.expand(ctx, expandPlainValue|expandFullName); isNil(x) {
                                x = v
                        } else if isNone(x) {
                                erro(ctx, "expecting file value: %T %v", v, v).of(v).debug(1)
                                return
                        }

                        if _, s, ok = asOptFullname(ctx, x/*, projects...*/); ok && s != "" {
                                val.Set(reflect.ValueOf(&optFullname{ s, x }))
                        } else {
                                var tv, _ = ctx.autoGet("@")
                                erro(ctx, "not a file: %v -> %v -> %s (@=%T %v)", v, x, s, tv, tv).of(v)
                                errostack(ctx, 5, "%v", ctx).debug(16)
                        }

                        if false {
                                vi := val.Interface().(*optFullname)
                                warn(ctx, "%v %v %v", /*current().of(v)*/ctx.Project(), v, vi.string).debug(true,1)
                        }
                case "smart.File":
                        var x Value
                        if x = v.expand(ctx, expandPlainValue); isNil(x) {
                                x = v
                        } else if isNone(x) {
                                erro(ctx, "expecting file value: %T %v", v, v).of(v).debug(1)
                                return
                        }

                        if file, ok := x.(*File); ok {
                                val.Set(reflect.ValueOf(file))
                        } else if proj := /*current()*/ctx.Project(); proj == nil {
                                erro(ctx, "no current project to find file '%v'", s).of(x).debug(1)
                        } else if file = proj.FindFile(ctx, x.Strval(ctx)); file != nil {
                                val.Set(reflect.ValueOf(file))
                        } else {
                                erro(ctx, "'%s' is not a file", s).of(v).debug(1)
                        }
                case "regexp.Regexp":
                        if rx, e := regexp.Compile(v.Strval(ctx)); e != nil {
                                erro(ctx, "compile regexp '%v' failed: %v", v, e).of(v).debug(1)
                        } else {
                                val.Set(reflect.ValueOf(rx))
                        }
                default:
                        erro(ctx, "option type unsupported: %T %v -> %v, %v", v, v, val.Elem().Kind(), val.Type().Elem()).
                                of(v).debug(1)
                }
                default: switch val.Type().String() {
                case "fs.FileMode": // aka. reflect.Uint32
                        if t, e := v.Integer(ctx); e == nil { val.SetUint(uint64(t)) } else {
                                erro(ctx, "%v: %v", v, t).debug(1)
                        }
                case "regex.Regex": // aka. reflect.Ptr
                        erro(ctx, "TODO: regexp: %T %v -> %v, %v", v, v, val.Kind(), val.Type()).of(v).debug(1)
                default:
                        erro(ctx, "option type unsupported: %T %v -> %v, %v", v, v, val.Kind(), val.Type()).of(v).debug(1)
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
                } else if aa, ok := arg.(*Argumented); ok {
                        if flag, okay = aa.value.(*Flag); okay {
                                value = MakeListOrScalar(aa.Position(), aa.args)
                        }
                }
                if !okay || flag == nil {
                        rest = append(rest, arg)
                        continue ForArgs
                }
                for i := 0; i < len(opts); i += 1 {
                        if _, match := flag.opt(ctx, opts[i]); match {
                                set(val, value)
                                continue ForArgs
                        }
                }
                rest = append(rest, arg)
        }
        if false && len(args) > 0 {
                info(ctx, "%v: %v %v %v", opts, field.Kind(), field, rest)
        }
        return
}

func parseOpts(ctx Context, iOpts interface{}, w expandwhat, args... Value) (rest []Value) {
        if w == expandNone {
                rest = args // NOTE: set the returning args first of all!
        } else {
                rest = mergeExpand(ctx, w, args...)
        }

        if opts := reflect.ValueOf(iOpts); opts.Kind() != reflect.Ptr {
                erro(ctx, "opts must be ptr: %v", opts.Kind()).debug(1)
        } else if opts = opts.Elem(); opts.Kind() == reflect.Struct {
                var (
                        otyp = opts.Type()
                        gen *generalOpts
                )
                if false { info(ctx, "opts: %v, %v", opts.Kind(), otyp) }
                for i := 0; i < otyp.NumField(); i += 1 {
                        var ft = otyp.Field(i)
                        var fv = opts.Field(i)
                        if ft.Name == "generalOpts" && fv.Kind() == reflect.Struct &&
                                fv.Type().String() == "smart.generalOpts" {
                                gen = (*generalOpts)(unsafe.Pointer(fv.UnsafeAddr()))
                                rest = parseOpts(ctx, gen, w, rest...)
                                if false { prompt(ctx, "%v: %v ; %v -> %v", ft.Name, *gen, args, rest).debug(1) }
                        } else {
                                rest = parseOpt(ctx, ft.Tag, fv, rest...)
                        }
                }
                if gen == nil { return }
                if gen.fullname { rest = mergeExpand(ctx, expandFullName, rest...) }
        } else {
                erro(ctx, "opts is not ptr of struct: %v", opts.Kind()).debug(1)
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
        var ( pos = ctx.Position(); elems []Value )
        for _, arg := range mergeExpand(ctx, expandPlainValue, args...) {
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
        )
        args = parseOpts(ctx, &opts, expandPlainValue, args...)

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
        time bool `t,tm,time,n,now`
}
func builtinDate(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts = builtinDateOpts{ }
        )
        args = parseOpts(ctx, &opts, expandPlainValue, args...)

        if t := time.Now(); len(args) > 0 {
                var vals []Value
                for _, a := range args {
                        var s string
                        if s = a.Strval(ctx); s == "" {
                                s = t.String()
                        } else if s = t.Format(s); s == "" {
                                s = fmt.Sprintf("%v", t)
                        }
                        vals = append(vals, MakeString(a.Position(), s))
                }
                res = MakeListOrScalar(pos, vals)
        } else if opts.time {
                res = MakeTime(pos, t)
        } else {
                res = MakeDate(pos, t)
        }
        return
}

type builtinDebugOpts struct {
        s int `s,stack`
        n int `n,num`
}
func builtinDebug(ctx Context, args... Value) (res Value) {
        var opts builtinDebugOpts
        args = parseOpts(ctx, &opts, expandPlainValue, args...)

        var s bytes.Buffer
        for i, a := range args {
                if i > 0 { fmt.Fprintf(&s, " ") }
                fmt.Fprintf(&s, "%s", a.Strval(ctx))
        }
        warnstack(ctx, opts.s, "%s", s.String()).debug(opts.n)
        return
}

func builtinError(ctx Context, args... Value) (res Value) {
        var s bytes.Buffer
        for i, a := range args {
                if i > 0 { fmt.Fprintf(&s, " ") }
                fmt.Fprintf(&s, "%s", a.Strval(ctx))
        }
        if false {
                erro(ctx, "%s", s.String()).debug(1)
        } else {
                errostack(ctx, 5, "%s", s.String()).debug(6)
        }
        return
}

func builtinWarning(ctx Context, args... Value) (res Value) {
        var s bytes.Buffer
        for i, a := range args {
                if i > 0 { fmt.Fprintf(&s, " ") }
                fmt.Fprintf(&s, "%s", a.Strval(ctx))
        }
        warn(ctx, "%s", s).debug(1)
        return
}

func builtinAssert(ctx Context, args... Value) Value {
        var vals []Value
        for _, a := range args {
                if g, ok := a.(*Group); ok {
                        vals = append(vals, g.Elems...)
                }
        }
        for _, a := range vals { if !a.True(ctx) {
                erro(ctx, "assertion failed: %v", a).of(a).debug(1)
        }}
        return nil
}

func builtinSureValue(ctx Context, args... Value) Value {
        for _, a := range args { if !a.True(ctx) {
                erro(ctx, "assertion failed: %v", a).of(a).debug(1)
        }}
        return MakeListOrScalar(ctx.Position(), args)
}

// $(defor $(x),$(y),$(z)) is identical to $(if $(defined $(x)),$(x),...)
func builtinDefor(ctx Context, args... Value) (res Value) {
        for _, a := range mergeExpand(ctx, expandPlainValue, args...) {
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
                if v := a.True(ctx); v {
                        res = a
                        break
                }
        }
        return
}

func builtinAnd(ctx Context, args... Value) (res Value) {
        for _, a := range args {
                if v := a.True(ctx); v {
                        res = a
                } else {
                        res = nil; break
                }
        }
        return
}

// $(not x y z) => (not (or x y z))
// $(not x,y,z) => (and (not x) (not y) (not z))
func builtinNot(ctx Context, args... Value) (res Value) {
        var t bool
        for _, a := range args { if t = a.True(ctx); t { break }}
        res = MakeBoolean(ctx.Position(), !t)
        return
}

func builtinNotEqual(ctx Context, args... Value) (res Value) {
        if n := len(args); n != 2 {
                erro(ctx, "wrong number of arguments, try: $(not-equal <value-list>,<regexp-list>)")
        } else if args[0].cmp(ctx, args[1]) != cmpEqual {
                res = MakeBoolean(ctx.Position(), true)
        }
        return
}

func builtinEqual(ctx Context, args... Value) (res Value) {
        if n := len(args); n != 2 {
                erro(ctx, "wrong number of arguments, try: $(equal <value-list>,<regexp-list>)")
        } else if cmp := args[0].cmp(ctx, args[1]); cmp == cmpEqual {
                res = MakeBoolean(ctx.Position(), true)
        }
        return
}

func builtinGreater(ctx Context, args... Value) (res Value) {
        if n := len(args); n != 2 {
                erro(ctx, "wrong number of arguments, try: $(greater <value-list>,<regexp-list>)")
        } else if cmp := args[0].cmp(ctx, args[1]); cmp == cmpGreater {
                res = MakeBoolean(ctx.Position(), true)
        }
        return
}

func builtinLess(ctx Context, args... Value) (res Value) {
        if n := len(args); n != 2 {
                erro(ctx, "wrong number of arguments, try: $(less <value-list>,<regexp-list>)")
        } else if cmp := args[0].cmp(ctx, args[1]); cmp == cmpSmaller {
                res = MakeBoolean(ctx.Position(), true)
        }
        return
}

type builtinMatchOpts struct {
        generalOpts
        regexps []*regexp.Regexp `r,re,rx,reg,regex,regexp`
        negated bool `n,ne,neg,negated,negative`
}
// $(match rx1 rx2 rx3, a b c d...)
func builtinMatch(ctx Context, args... Value) (res Value) {
        var (
                patList, valList []Value
                opts builtinMatchOpts
        )
        if n := len(args); n < 2 {
                erro(ctx, "wrong arguments, try: $(match <regexp-list>,<value-list>,...)").debug(1)
                return
        }

        patList = parseOpts(ctx, &opts, expandPlainValue, args[0])
        valList = mergeExpand(ctx, expandPlainValue, args[1:]...)

        var pos = ctx.Position()
ForValList:
        for _, val := range valList {
                if isTrivial(val) { continue ForValList }

                var str = val.Strval(ctx)
                for _, rx := range opts.regexps {
                        var matched = rx.MatchString(str);
                        if opts.negated { matched = !matched }
                        if matched {
                                res = MakeBoolean(pos, true)
                                break ForValList
                        }
                }
                for _, pat := range patList {
                        var matched, s, _ = pat.match(ctx, str)
                        if !matched { matched = !opts.fullname && s != "" }
                        if opts.negated { matched = !matched }
                        if matched {
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
                if t := args[0].True(ctx); t {
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
                )
                if a = args[0].expand(ctx, expandPlainValue); isNil(a) { a = args[0] }
                if b = args[1].expand(ctx, expandDelegate  ); isNil(b) { b = args[1] }
                if s1, s2 = a.Strval(ctx), b.Strval(ctx); s1 == s2 {
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
                )
                if a = args[0].expand(ctx, expandPlainValue); isNil(a) { a = args[0] }
                if b = args[1].expand(ctx, expandDelegate  ); isNil(b) { b = args[1] }
                if s1, s2 = a.Strval(ctx), b.Strval(ctx); s1 != s2 {
                        res = args[2]
                } else if n > 3 {
                        res = MakeListOrScalar(ctx.Position(), args[3:])
                }
        }
        return
}

func builtinFor(ctx Context, args... Value) (res Value) {
        if n := len(args); n < 2 {
                erro(ctx, "not enough arguments, try: $(foreach <list>,<template>)")
                return
        }

        var (
                defs []*Def
                vals []Value
                values = mergeExpand(ctx, expandPlainValue, args[0])
        )

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

        var list []Value
        var pos = ctx.Position()
        for _, a := range args[1:] {
                if values = mergeExpand(ctx, expandPlainValue, a); len(values) == 0 {
                        list = append(list, MakeNone(pos))
                } else if len(values) == 1 {
                        list = append(list, values[0])
                } else {
                        list = append(list, MakeList(a.Position(), values...))
                }
        }

        res = MakeListOrScalar(pos, list)
        return
}

func builtinForEach(ctx Context, args... Value) (res Value) {
        if n := len(args); n < 2 {
                erro(ctx, "not enough arguments ($(foreach <list>,<template>)): %v", n).debug(1)
                return
        }

        var (
                pos = ctx.Position()
                cc = autoContext{ Context:ctx, defs:make(autoDefMap) }
                values = mergeExpand(ctx, expandPlainValue, args[0])
                resList []Value
        )
        for _, val := range values {
                if isTrivial(val) {
                        continue // ignore
                } else if s, ok := val.(*String); ok && s.string == "" {
                        continue // ignore
                } else {
                        cc.autoSet("_", val)
                }

                var list []Value
                for _, a := range args[1:] {
                        var v Value
                        if v = a.expand(&cc, expandPlainValue|expandPairVal); isNil(v) { v = a }
                        if true && len(v.defs(&cc, "_")) > 0 {
                                erro(ctx, "'_' in '%v' not expanded: %v", a, v).of(a).debug(true, 1)
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
                s string
        )
        for _, a := range args {
                if val = a.expand(ctx, expandDelegate); isNil(val) {
                        val = a
                }
                if isTrivial(val) {
                        continue
                } else if s = strings.TrimSpace(val.Strval(ctx)); s != "" {
                        vals = append(vals, MakeString(pos, os.Getenv(s)))
                }
        }
        return MakeListOrScalar(pos, vals)
}

type builtinAutoOpts struct {
        //closure bool `c,closure`
}
func builtinAuto(ctx Context, args... Value) (res Value) {
        var (
                opts builtinAutoOpts
                vals []Value
        )
        for _, a := range parseOpts(ctx, &opts, expandPlainValue, args...) {
                var ( name string; val Value )
                name = a.Strval(ctx)
                for c := ctx; c != nil; c = c.inner() {
                        //warn(ctx, "%v %T", name, c)
                        if _, ok := c.(*defaultContext); ok {
                                val = MakeNone(a.Position())
                                break
                        } else if val, _ = c.autoGet(name); !isNil(val) {
                                warn(ctx, "%v %T %v", a, c, val)
                                //break
                        }
                }
                warn(ctx, "%v %v", name, ctx)
                warn(ctx, "%v %T %v", name, val, val).debug(1)
                vals = append(vals, val)
        }
        return MakeListOrScalar(ctx.Position(), vals)
}

type builtinValueOpts struct {
        closure bool `c,closure`
}
func builtinValue(ctx Context, args... Value) (res Value) {
        var (
                opts builtinValueOpts
                vals []Value
        )
        for _, a := range parseOpts(ctx, &opts, expandPlainValue, args...) {
                var (
                        name string
                        val Value
                )
                if name = a.Strval(ctx); opts.closure {
                        val = closureGet(ctx, name)
                } else if scope := ctx.Scope(); scope != nil {
                        if def := scope.FindDef(name); def != nil {
                                val = def.Call(ctx)
                        }
                }
                if isNil(val) { val, _ = ctx.autoGet(name) }
                if isNil(val) { val = MakeNone(a.Position()) }
                vals = append(vals, val)
        }
        return MakeListOrScalar(ctx.Position(), vals)
}

type builtinCallOpts struct {
        closure bool `c,closure`
}
func builtinCall_failure(ctx Context, args... Value) (res Value) {
        var (
                opts builtinCallOpts
                vals []Value
        )
        if args = parseOpts(ctx, &opts, expandPlainValue, args...); len(args) > 0 {
                var ( name string; val Value )
                if name = args[0].Strval(ctx); opts.closure {
                        for _, scope := range ctx.closureScopes() {
                                if def := scope.FindDef(name); def != nil && !isTrivial(def.value) {
                                        val = def.Call(ctx, args[1:]...)
                                        break
                                }
                        }
                } else if def := ctx.Scope().FindDef(name); def != nil {
                        val = def.Call(ctx, args[1:]...)
                }
                if isNil(val) { val = MakeNone(args[0].Position()) }
                vals = append(vals, val)
        }
        return MakeListOrScalar(ctx.Position(), vals)
}

type builtinClosureOpts struct {
        required bool `required,require-def,require-defs`
}
func builtinClosure(ctx Context, args... Value) (res Value) {
        var (
                opts builtinClosureOpts
                vals, names []Value
        )
        if len(args) < 1 {
                erro(ctx, "insufficient args: %v", args).debug(1)
                return
        }

        names = parseOpts(ctx, &opts, expandPlainValue, args[0])
        if len(names) < 1 {
                erro(ctx, "no names: %v", args[0]).debug(1)
                return
        }

        for _, nameVal := range names {
                var ( def *Def; name string )
                if name = nameVal.Strval(ctx); /*opts.closure*/true {
                        for _, scope := range ctx.closureScopes() {
                                if def = scope.FindDef(name); def != nil {
                                        break
                                }
                        }
                } else if def != nil {
                        def = ctx.Scope().FindDef(name)
                }
                if def == nil {
                        if opts.required {
                                erro(ctx, "no def '%v' (%v)", name, nameVal).of(nameVal).debug(1)
                        }
                } else {
                        vals = append(vals, def.Call(ctx, args[1:]...))
                }
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
                var bufout, buferr bytes.Buffer
                var s = a.Strval(ctx)
                sh := exec.Command("sh", "-c", s)
                sh.Stdout, sh.Stderr = &bufout, &buferr
                if err = sh.Run(); err != nil {
                        s = strings.TrimSpace(buferr.String())
                        erro(ctx, "%s", err).debug(1)
                        return
                }
                val := MakeString(pos, strings.TrimSpace(bufout.String()))
                vals = append(vals, val)
                bufout.Reset()
                buferr.Reset()
        }
        return MakeListOrScalar(pos, vals)
}

type builtinWhichOpts struct {
}
func builtinWhich(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts builtinWhichOpts
                vals []Value
        )
        for _, a := range parseOpts(ctx, &opts, expandPlainValue, args...) {
                if s, err := exec.LookPath(a.Strval(ctx)); err != nil {
                        erro(ctx, "%v", err).debug(1)
                        return
                } else if s != "" {
                        vals = append(vals, MakeString(pos, s))
                }
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
        )

        args = parseOpts(ctx, &opts, expandPlainValue, args...)

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

        for _, a := range args {
                var s = a.Strval(ctx)
                fmt.Fprintf(stderr, "%s: serving files %v ...\n", pos, s)
                http.Handle("/", http.FileServer(http.Dir(s)))
        }

        if err := server.ListenAndServe(); err == http.ErrServerClosed {
                if false { info(ctx, "http server closed") }// Requested /quit
        } else if err != nil {
                erro(ctx, "%s", err).debug(1)
        }
        return
}

func builtinServeHttps(ctx Context, args... Value) (res Value) {
        erro(ctx, "'serve-https' is unimplemented yet").at(ctx.Position()).debug(1)
        return
}

func builtinPrint(ctx Context, args... Value) (res Value) {
        var (
                x = len(args)
                sb bytes.Buffer
        )
        for i, a := range args {
                var s string
                if 0 < i && i < x { fmt.Fprintf(&sb, " ") }
                if a == nil {
                        continue
                } else if s = EscapedString(ctx, a); s != "" {
                        fmt.Fprintf(&sb, "%s", s)
                }
        }
        prompt(ctx, sb.String())
        return
}

func builtinPrintl(ctx Context, args... Value) (res Value) {
        var (
                x = len(args)
                sb bytes.Buffer
        )
        for i, a := range args {
                var s string
                if 0 < i && i < x { fmt.Fprintf(&sb, " ") }
                s = EscapedString(ctx, a)
                fmt.Fprintf(&sb, "%s", s)
                if i == x && !strings.HasSuffix(s, "\n") {
                        fmt.Fprintf(&sb, "\n")
                }
        }
        prompt(ctx, sb.String())
        return
}

func builtinPrintln(ctx Context, args... Value) (res Value) {
        var (
                x = len(args)
                sb bytes.Buffer
        )
        for i, a := range args {
                var s string
                if 0 < i && i < x { fmt.Fprintf(&sb, " ") }
                if a == nil {
                        continue
                } else if s = EscapedString(ctx, a); s != "" {
                        fmt.Fprintf(&sb, "%s", s)
                }
        }
        fmt.Fprintf(&sb, "\n")
        prompt(ctx, sb.String())
        return
}

type builtinAppendOpts struct {
        auto bool `a,auto`
        closure bool `c,closure`
        string bool `s,str;s,string`
        verbose bool `v,verbose`
}
func builtinAppend(ctx Context, args... Value) (result Value) {
        var (
                opts builtinAppendOpts
                vars []Value
                list []Value
        )
        if len(args) < 2 {
                erro(ctx, "insufficient number of arguments: %v", args).debug(1)
                return
        }

        vars = parseOpts(ctx, &opts, expandPlainValue, args[0])
        if list = mergeExpand(ctx, expandPlainValue, args[1:]...); len(list) == 0 {
                warn(ctx, "append no values").debug(1)
                return
        }

        var pos = ctx.Position()
        for _, a := range vars {
                var name string
                if name = a.Strval(ctx); name == "" {
                        erro(ctx, "name '%v' is empty", a).of(a).debug(1)
                        break
                }
                if opts.closure {
                        if val := closureGet(ctx, name); !isTrivial(val) {
                                list = append(merge(val), list...)
                        }
                        closureSet(ctx, name, MakeListOrScalar(pos, list))
                } else if opts.auto {
                        if val, found := ctx.autoGet(name); found && !isTrivial(val) {
                                list = append(merge(val), list...)
                        }
                        ctx.autoSet(name, MakeListOrScalar(pos, list))
                } else if proj := ctx.Project(); proj != nil {
                        var def *Def
                        if obj := proj.resolveObject(ctx, name); obj != nil {
                                def, _ = obj.(*Def)
                        }
                        if def == nil {
                                erro(ctx, "'%s' (%v) is undefined (%T)", name, a, ctx).debug(1)
                                break
                        } else {
                                def.append(ctx, list...)
                        }
                } else {
                        erro(ctx, "%s", ctx).debug(1)
                        break
                }
        }
        return
}

func builtinPlus(ctx Context, args... Value) (result Value) {
        var num int64
        for _, a := range args {
                if i, e := a.Integer(ctx); e == nil { num += i } else {
                        erro(ctx, "%v: %v", a, e).debug(1)
                }
        }
        return MakeInt(ctx.Position(), num)
}

func builtinMinus(ctx Context, args... Value) (result Value) {
        var num int64
        for i, a := range args {
                if v, e := a.Integer(ctx); e != nil {
                        erro(ctx, "%v: %v", a, e).debug(1)
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
	reverse bool `r,rev,reverse`
	keepAuto bool `a,auto,keepauto,keep-auto`
        unexpand bool `un,ue,unexpand,ne,noexpand,no-expand`
        plain bool `pl,pla,plain,pv,plainvalue,plain-value`
}
func builtinUnique(ctx Context, args... Value) (res Value) {
        var opts builtinUniqueOpts
        if len(args) > 0 {
                args = append(parseOpts(ctx, &opts, 0, merge(args[0])...), args[1:]...)
        }
        if opts.unexpand {
                args = merge(args...)
        } else if opts.plain {
                var x = expandPlainValue
                if opts.keepAuto { x &= ^expandAuto }
                args = mergeExpand(ctx, x, args...)
        } else {
                var x = expandDelegate | expandPathStr | expandPairVal
                if opts.keepAuto { x &= ^expandAuto }
                args = mergeExpand(ctx, x, args...)
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
                        var s = a.Strval(ctx)
                        for _, v := range list {
                                if s == v.Strval(ctx) { continue ForArgs }
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
                )
                if l < 2 {
                        vals = mergeExpand(ctx, expandPlainValue, args...)
                } else {
                        vals = mergeExpand(ctx, expandPlainValue, args[:l-1]...)
                        sep = args[l-1].Strval(ctx)
                }
                for _, a := range vals {
                        if v := a.Strval(ctx); v != "" { fields = append(fields, v) }
                }
                res = MakeString(ctx.Position(), strings.Join(fields, sep))
        }
        return
}

func builtinQuote(ctx Context, args... Value) (res Value) {
        args = mergeExpand(ctx, expandPlainValue, args...)
        if l := len(args); l > 0 {
                var fields []string
                for _, a := range args {
                        if v := a.Strval(ctx); v != "" { fields = append(fields, v) }
                }
                res = MakeString(ctx.Position(), strconv.Quote(strings.Join(fields, " ")))
        } else {
                res = MakeNone(ctx.Position())
        }
        return
}

func builtinQuoteJoin(ctx Context, args... Value) (res Value) {
        var sep string
        args = mergeExpand(ctx, expandPlainValue, args...)

        if l := len(args); l > 1 {
                sep = args[l-1].Strval(ctx)
                args = args[:l-1]
        }
        if l := len(args); l > 0 {
                var fields []string
                for _, a := range args {
                        if v := a.Strval(ctx); v != "" { fields = append(fields, v) }
                }
                res = MakeString(ctx.Position(), strconv.Quote(strings.Join(fields, sep)))
        } else {
                res = MakeNone(ctx.Position())
        }
        return
}

func builtinSplitString(ctx Context, args... Value) (res Value) {
        args = mergeExpand(ctx, expandPlainValue, args...)
        if l := len(args); l > 0 {
                var fields []Value
                for _, a := range args {
                        if s := a.Strval(ctx); s != "" {
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
                        if s = v.Strval(ctx); s != "" { strs = append(strs, s) }
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
        var sep string
        if l := len(args); l > 1 {
                sep = args[l-1].Strval(ctx)
                args = args[:l-1]
        }

        var err error
        if res = builtinSplitQuote(ctx, args...); !isNil(res) {
                if res, err = joinstrings(ctx, res, sep); err != nil {
                        erro(ctx, "%v", err).debug(1)
                }
        } else {
                erro(ctx, "%v", err).debug(1)
        }
        return
}

func builtinSplitJoinQuote(ctx Context, args... Value) (res Value) {
        var sep string
        if l := len(args); l > 1 {
                sep = args[l-1].Strval(ctx)
                args = args[:l-1]
        }

        var (
                v Value
                err error
        )
        if v = builtinSplitString(ctx, args...); !isNil(v) {
                if v, err = joinstrings(ctx, v, sep); err == nil {
                        res = MakeString(ctx.Position(), strconv.Quote(v.Strval(ctx)))
                }
        }
        if err != nil { erro(ctx, "%v", err).debug(1) }
        return
}

func builtinField(ctx Context, args... Value) (res Value) {
        var pos = ctx.Position()
        if l := len(args); l >= 2 {
                var (
                        fields []string
                        s string = args[1].Strval(ctx)
                        i int64
                )
                if n, e := args[0].Integer(ctx); e != nil {
                        erro(ctx, "%v: %v", args[0], e).debug(1)
                        return
                } else { i = n }

                if l > 2 {
                        fields = strings.Split(s, args[2].Strval(ctx))
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
                erro(ctx, "unknown current context").debug(1)
                return
        }

        for _, arg := range args {
                var v Value
                if v, err = proj.using.Get(ctx, arg.Strval(ctx)); err != nil {
                        erro(ctx, "%v", err).debug(1)
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
        )
        for _, a := range args {
                list = append(list, MakePathStr(pos, a.Strval(ctx)))
        }
        result = MakeListOrScalar(pos, list)
        return
}

func builtinString(ctx Context, args... Value) (result Value) {
        var s bytes.Buffer
        for i, a := range args {
                if i > 0 { s.WriteString(" ") }
                s.WriteString(a.Strval(ctx))
        }
        result = MakeString(ctx.Position(), s.String())
        return
}

func builtinStrings(ctx Context, args... Value) (result Value) {
        var strs []Value
        for _, a := range mergeExpand(ctx, expandPlainValue, args...) {
                strs = append(strs, MakeString(a.Position(), a.Strval(ctx)))
        }
        result = MakeListOrScalar(ctx.Position(), strs)
        return
}

type builtinFilterOpts struct {
        stem bool `s,stem;us,use-stem`
}
func filterValues(ctx Context, pats []Value, opts builtinFilterOpts, neg bool, values... Value) (result []Value, err error) {
        var filter = func(v Value) Value {
                if strings.HasPrefix(v.String(), "-lcrypto") {
                        warn(ctx, "pats=%v value=%v (%T); %v", pats, v, v, values).of(v).debug(1)
                }
                for _, pat := range pats {
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
        for _, v := range merge(Reveal(ctx, values...)...) {
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
                        vals, pats []Value
                        i int
                )
                if pats = parseOpts(ctx, &opts, expandPlainValue, args[0]); len(pats) > 0 {
                        i = 1 // good
                } else if pats = mergeExpand(ctx, expandPlainValue, args[1]); len(pats) == 0 {
                        erro(ctx, "no patterns: %v", args).debug(1)
                        return
                } else {
                        i = 2
                }

                if len(args) <= i {
                        erro(ctx, "out of index: %d %v", i, args).debug(1)
                        return
                }

                vals = mergeExpand(ctx, expandPlainValue, args[i:]...)
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
        var pos = ctx.Position()
        args = mergeExpand(ctx, expandPlainValue, args...)

        var list []Value
        if n := len(args); n > 1 {
                var ( i1, i2 int; e error )
                if i1, e = intVal(ctx, args[0], -1); e != nil {
                      erro(ctx, "%v", e).debug(1)
                } else {
                        args = args[1:]
                }
                if i2, e = intVal(ctx, args[0], -1); e != nil {
                        if _, ok := e.(*strconv.NumError); !ok {
                                erro(ctx, "%v", e).of(args[0]).debug(1)
                                return
                        }
                } else {
                        args = args[1:]
                }

                if i1 < -1 && i2 < -1 {
                        erro(ctx, "wrong indices (%d, %d)", i1, i2).debug(1)
                        return
                } else if i1 > i2 { t := i1; i1 = i2; i2 = t } // swap the wrong order
                
                var a, b = int(i1), int(i2)
                if a == -1 { a = b }
                if a == -1 { return }

                for _, arg := range args {
                        var s = arg.Strval(ctx)
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
        var ( pos = ctx.Position(); list []Value )
        if nargs := len(args); nargs > 2 {
                var (
                        s1 = args[0].Strval(ctx)
                        s2 = args[1].Strval(ctx)
                )
                for _, arg := range mergeExpand(ctx, expandDelegate, args[2:]...) {
                        var s = strings.Replace(arg.Strval(ctx), s1, s2, -1)
                        list = append(list, MakeString(pos, s))
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

type builtinPatsubstOpts struct {
        generalOpts
        fullfiles bool `ff,fullfile;ff,fullfiles`
        files bool `f,file;fs,files`
        cleanPath bool `c,clean;c,cleanpath`
        baseFiles bool `b,base;b,bases;bf,base-files`
        usedFiles bool `u,used;u,using;uf,used-files`
        noFileMap bool `n,no-filemap`
}

// $(patsubst pattern,replacement,text)
// TODO:
//   $(var:pattern=replacement)
//   $(var:suffix=replacement)
func builtinPatsubst(ctx Context, args... Value) (res Value) {
        var (
                opts builtinPatsubstOpts
                list []Value
                arg0 []Value
        )
        if len(args) < 3 {
                erro(ctx, "not enough arguments").debug(1)
                return
        }

        const infos = false

        arg0 = parseOpts(ctx, &opts, expandPlainValue, args[0])

        // TODO: support flags -name and -full for name-only and full-name-only matching
        var srcPats, dstPats, sources []Value
        if len(arg0) > 0 {
                srcPats = arg0
                dstPats = mergeExpand(ctx, expandPlainValue, args[1])
                sources = mergeExpand(ctx, expandPlainValue, args[2:]...)
                if infos {
                        info(ctx, "src: %v", srcPats)
                        info(ctx, "dst: %v", dstPats)
                        info(ctx, "%v", sources).debug(1)
                }
        } else {
                srcPats = mergeExpand(ctx, expandPlainValue, args[1])
                dstPats = mergeExpand(ctx, expandPlainValue, args[2])
                sources = mergeExpand(ctx, expandPlainValue, args[3:]...)
                if infos {
                        info(ctx, "src: %v", srcPats)
                        info(ctx, "dst: %v", dstPats)
                        info(ctx, "%v", sources).debug(1)
                }
        }

        var proj = ctx.Project()
        var closured = closureProjects(ctx)
        // Using the most derived context for correct &(...)
        //defer setclosure(setclosure(cloctx.unshift(proj.scope)))

        var filemaps []*FileMap
        if !opts.noFileMap { filemaps = proj.filemaps(ctx, opts.baseFiles, opts.usedFiles) }

ForSources:
        for _, src := range sources {
                var source interface{} = src
                if opts.files || opts.fullfiles {
                        if file, ok := src.(*File); ok {
                                source = file
                        } else if file = proj.FindFile(ctx, src.Strval(ctx)); file != nil {
                                if (opts.fullname || opts.fullfiles) && !filepath.IsAbs(file.name) {
                                        if !file.change("", "", file.fullname()) {
                                                warn(ctx, "changing fullname failed: %v", file).debug(1)
                                        }
                                }
                                source = file
                        }
                } else if opts.fullname {
                        var ( s string; ok bool )
                        if _, s, ok = asOptFullname(ctx, src, closured...); s == "" {
                                erro(ctx, "fullname '%v' is empty", src).of(src)
                                erro(ctx, "called from here", src).debug(1)
                                return
                        } else if !ok {
                                erro(ctx, "fullname '%v' failed", src).of(src)
                                erro(ctx, "called from here", src).debug(1)
                                return
                        } else {
                                source = s
                        }
                }

                var srcPat Value
                var ( matched bool; str string; stems []string )
        ForSrcPats:
                for _, srcPat = range srcPats {
                        if matched, str, stems = srcPat.match(ctx, source); matched {
                                break ForSrcPats
                        } else if infos {
                                info(ctx, "source=%v (%T) srcPat=%v (%T) str=%s stems=%v",
                                        source, source, srcPat, srcPat, str, stems).debug(true,1)
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
                        var nameVal, /*rest*/_ = dst.stencil(ctx, stems)
                        if isNil(nameVal) {
                                erro(ctx, "nil stencil: %T %v (stems=%v)", dst, dst, stems).debug(1)
                                nameVal = dst
                        }

                        var name string
                        if name = nameVal.Strval(ctx); name == "" /*|| len(rest) > 0*/ {
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
                                        if ok, _, s := m.Match(ctx, name); ok {
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
                                                if sub, err = path.Strval(); err != nil { erro(ctx, "%v", err); return }
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

                                file.position = srcPat.Position()
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
        var (
                pos = ctx.Position()
                list []Value
        )
        for i, a := range mergeExpand(ctx, expandPlainValue, args...) {
                if i == 0 { pos = a.Position() }
                if s := a.Strval(ctx); s != "" {
                        list = append(list, MakeString(a.Position(), strings.Title(s)))
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

func builtinUpperCase(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                list []Value
        )
        for i, a := range mergeExpand(ctx, expandPlainValue, args...) {
                if i == 0 { pos = a.Position() }
                if s := a.Strval(ctx); s != "" {
                        list = append(list, MakeString(a.Position(), strings.ToUpper(s)))
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

func builtinLowerCase(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                list []Value
        )
        for _, a := range mergeExpand(ctx, expandPlainValue, args...) {
                if s := a.Strval(ctx); s != "" {
                        list = append(list, MakeString(a.Position(), strings.ToLower(s)))
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

func builtinTrim(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                cutset string
                list []Value
        )
        for i, a := range mergeExpand(ctx, expandPlainValue, args...) {
                if i == 0 { pos = a.Position() }
                if s := a.Strval(ctx); s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, MakeString(pos, strings.TrimSpace(s)))
                        } else {
                                list = append(list, MakeString(pos, strings.Trim(s, cutset)))
                        }
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

func builtinTrimLeft(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                cutset string
                list []Value
        )
        for i, a := range mergeExpand(ctx, expandPlainValue, args...) {
                if i == 0 { pos = a.Position() }
                if s := a.Strval(ctx); s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, MakeString(a.Position(), strings.TrimLeftFunc(s, unicode.IsSpace)))
                        } else {
                                list = append(list, MakeString(a.Position(), strings.TrimLeft(s, cutset)))
                        }
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

func builtinTrimRight(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                cutset string
                list []Value
        )
        for i, a := range mergeExpand(ctx, expandPlainValue, args...) {
                if i == 0 { pos = a.Position() }
                if s := a.Strval(ctx); s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, MakeString(a.Position(), strings.TrimRightFunc(s, unicode.IsSpace)))
                        } else {
                                list = append(list, MakeString(a.Position(), strings.TrimRight(s, cutset)))
                        }
                }
        }
        res = MakeListOrScalar(pos, list)
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
        if len(args) == 0 { return }
        prefixs = mergeExpand(ctx, expandPlainValue, args[0])

        if len(args) == 1 {
                if len(prefixs) > 1 { values = prefixs[1:] }
        } else {
                values = mergeExpand(ctx, expandPlainValue, args[1:]...)
        }

        if len(values) == 0 {
                return
        } else if len(prefixs) == 0 {
                res = MakeListOrScalar(ctx.Position(), values)
                return
        }

        if info { warn(ctx, "%v %v", prefixs, values) }
        for _, value := range values {
                var (
                        pos = value.Position()
                        p, s string
                )
                if s = value.Strval(ctx); s == "" { continue }
        ForPrefix:
                for _, prefix := range prefixs {
                        if prefix.patterned(ctx) {
                                // fallthrough
                        } else if p = prefix.Strval(ctx); p == "" {
                                continue
                        } else if strings.HasPrefix(s, p) {
                                s = strings.TrimPrefix(s, p)
                                pos = prefix.Position()
                                break ForPrefix
                        }

                        var full, cutset, stems = prefix.match(ctx, value)
                        if info /*|| (strings.Contains(s, "/.smart/modules/") && prefix.String() == "%%/.smart/modules/")*/ {
                                warn(ctx, "prefix = %T %v", prefix, prefix)
                                warn(ctx, "value  = %T %v", value, value)
                                warn(ctx, "trim   = %v", strings.TrimPrefix(s, cutset))
                                warn(ctx, "full=%v cutset=%v stems=%v", full, cutset, stems).debug(1)
                        }
                        if !full && s == cutset {
                                continue
                        } else if len(s) > len(cutset) && strings.HasPrefix(s, cutset) {
                                s = strings.TrimPrefix(s, cutset)
                        } else {
                                s = strings.TrimLeftFunc(s, unicode.IsSpace)
                        }
                        pos = prefix.Position()
                        break ForPrefix
                }
                if info { warn(ctx, "list=%v trimmed=%v", list, s).debug(true, 1) }
                if s != "" { list = append(list, MakeString(pos, s)) }
        }
        if err == nil { res = MakeListOrScalar(ctx.Position(), list) }
        return
}

func builtinTrimSuffix(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                cutset, s string
                list []Value
        )
        for i, a := range mergeExpand(ctx, expandPlainValue, args...) {
                if i == 0 { pos = a.Position() }
                if s = a.Strval(ctx); s != "" {
                        if i == 0 {
                                cutset = s
                        } else if cutset == "" {
                                list = append(list, MakeString(a.Position(), strings.TrimRightFunc(s, unicode.IsSpace)))
                        } else {
                                list = append(list, MakeString(a.Position(), strings.TrimSuffix(s, cutset)))
                        }
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

type builtinTrimExtOpts struct {
        all bool `a,all`
        ext []string `e,ext`
}
func builtinTrimExt(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts builtinTrimExtOpts
                list []Value
                ext string
        )
        for i, a := range parseOpts(ctx, &opts, expandPlainValue, args...) {
                if i == 0 { pos = a.Position() }
                if s := a.Strval(ctx); s != "" {
                        if i == 0 && len(args) > 1 {
                                ext = s
                        } else if ext == "" {
                                for ext = filepath.Ext(s); ext != ""; {
                                        s = strings.TrimSuffix(s, ext)
                                        if opts.all { ext = filepath.Ext(s) } else { break }
                                }
                                list = append(list, MakeString(a.Position(), s))
                        } else if ext == filepath.Ext(s) {
                                list = append(list, MakeString(a.Position(), strings.TrimRight(s, ext)))
                        }
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

type builtinExtOpts struct {

}
func builtinExt(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts builtinExtOpts
                list []Value
        )
        for _, a := range parseOpts(ctx, &opts, expandPlainValue, args...) {
                list = append(list, MakeString(a.Position(), filepath.Ext(a.Strval(ctx))))
        }
        res = MakeListOrScalar(pos, list)
        return
}

type builtinPrintfOpts struct {

}
func builtinPrintf(ctx Context, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts builtinPrintfOpts
                vals []Value
                f string
        )
        if len(args) < 1 {
                erro(ctx, "not enough args, try $(printf 'format', ...)").debug(1)
                return
        } else if vals = parseOpts(ctx, &opts, expandPlainValue, args[0]); len(vals) != 1 {
                erro(ctx, "not enough args, try $(printf 'format', ...)").debug(1)
                return
        } else {
                f = vals[0].Strval(ctx)
        }

        var a []interface{}
ForArgs:
        for i, v := range mergeExpand(ctx, expandPlainValue, args[1:]...) {
                if i == 0 { pos = v.Position() }
                for ; i < len(f); {
                        if f[i] != '%' {
                                if i += 1; i < len(f) {
                                        continue
                                } else {
                                        continue ForArgs
                                }
                        }
                        for i += 1; i < len(f); i += 1 {
                                switch f[i] {
                                case '%': continue ForArgs
                                case '+', '-', '#', ' ', '.', '0', '1', '2', '3',
                                        '4', '5', '6', '7', '8', '9': continue
                                case 'c', 'd', 'o', 'O', 'q', 'U':
                                        if t, e := v.Integer(ctx); e == nil { a = append(a, t) } else {
                                                erro(ctx, "%v: %v", v, e).debug(1)
                                        }
                                        continue ForArgs
                                case 'e', 'E', 'f', 'F', 'g', 'G':
                                        if t, e := v.Float(ctx); e == nil { a = append(a, t) } else {
                                                erro(ctx, "%v: %v", v, e).debug(1)
                                        }
                                        continue ForArgs
                                case 'b', 'x', 'X':
                                        switch k := v.kind(); k {
                                        case valInteger:
                                                if t, e := v.Integer(ctx); e == nil { a = append(a, t) } else {
                                                        erro(ctx, "%v: %v", v, e).debug(1)
                                                }
                                                continue ForArgs
                                        case valFloat:
                                                if t, e := v.Float(ctx); e == nil { a = append(a, t) } else {
                                                        erro(ctx, "%v: %v", v, e).debug(1)
                                                }
                                                continue ForArgs
                                        case valOther:
                                                a = append(a, v.Strval(ctx))
                                                continue ForArgs
                                        }
                                case 'v':
                                        a = append(a, v.Strval(ctx))
                                        continue ForArgs
                                case 't', 'T':
                                        a = append(a, v)
                                        continue ForArgs
                                }
                        }
                }
        }
        res = MakeString(pos, fmt.Sprintf(f, a...))
        return
}

func builtinIndent(ctx Context, args... Value) (res Value) {
        var (
                l []Value
                s string // indent
        )
        if x := len(args); x > 0 {
                if v, ok := Scalar(args[0]).(*Int); ok {
                        args, s = args[1:], strings.Repeat(" ", int(v.int64))
                } else {
                        erro(ctx, "requires integer argument (first|last)").debug(1)
                        return
                }
        }
        for _, a := range args {
                var lines []string
                for _, line := range strings.Split(a.Strval(ctx), "\n") {
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
        verbose bool `v,verb,verbose`
        string bool `s,str,string`
        match bool `m,mat,match,p,pat,pattern`
}
func builtinContains(ctx Context, args... Value) (res Value) {
        var (
                opts builtinContainsOpts
                vals []Value
                list []Value
        )
        if len(args) < 2 {
                erro(ctx, "unexpected number of arguments, try $(contains a b c1 -or c2, v1 v2 …)").debug(1)
                return
        }

        vals = parseOpts(ctx, &opts, expandPlainValue, args[0])
        list = mergeExpand(ctx, expandPlainValue, args[1:]...)

        var ( n = 0; x = len(vals); va []Value )
        for _, val := range vals {
                var s string
                switch v := val.(type) {
                default: va = []Value{ val }
                case *Flag:
                        if s = v.name.Strval(ctx); s == "or" {
                                va, x = append(va, val), x-1
                                continue
                        }
                case *Pair: // FIXME: -or=(c1 c2 c3)
                        if f, ok := v.Key.(*Flag); !ok {
                                va = []Value{ val }
                        } else {
                                if s = f.name.Strval(ctx); s == "or" {
                                        va, x = append(va, v.Value), x-1
                                        continue
                                }
                        }
                }

                if len(va) == 0 { continue }

        ForList:
                for _, v := range list {
                        for _, a := range va {
                                if opts.string {
                                        var r string = v.Strval(ctx)
                                        if s = a.Strval(ctx); r != s { continue ForList }
                                } else if opts.match {
                                        var full, r, s = a.match(ctx, v)
                                        if false { warn(ctx, "%v %v; %v %v %v; %v, %v", a, v, full, r, s, n, x).debug(1) }
                                        if !full { continue ForList }
                                } else if a.cmp(ctx, v) != cmpEqual {
                                        continue ForList
                                }
                        }
                        if n += 1/* matched one */; n == x { break }
                }
                va = nil
        }
        if opts.verbose {
                info(ctx, "%v contains %v: %v (%v, %v)\n", list, vals, (n==x), n, x).debug(opts.debug)
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
                for _, a := range args { enc.Write([]byte(a.Strval(ctx))) }
                enc.Close()
                res = MakeString(pos, buf.String())
        }
        return
}

func builtinDecodeBase64(ctx Context, args... Value) (res Value) {
        if len(args) > 0 {
                var list []Value
                for _, a := range args {
                        var s string = a.Strval(ctx)
                        if dat, err := base64.StdEncoding.DecodeString(s); err != nil {
                                erro(ctx, "decode '%s' failed: %v", s, err).debug(1)
                                return
                        } else {
                                list = append(list, MakeString(a.Position(), string(dat)))
                        }
                }
                res = MakeListOrScalar(ctx.Position(), list)
        }
        return
}

func asFile(ctx Context, a Value, projects ...*Project) (f *File) {
        switch t := a.(type) {
        case *File     : f = t
        case *Barefile : f = t.File
        case *Def      : if !isTrivial(t.value) { return asFile(ctx, t.value   ) }
        case *List     : if len(t.Elems) == 1   { return asFile(ctx, t.Elems[0]) }
        case *RuleEntry:                          return asFile(ctx, t.target  )
        case *String: // NOTE: Finding string here is slow! It's acceptable to keep string.
        case *Bareword, *Barecomp, *Path:
                if len(projects) == 0 { projects = closureProjects(ctx) }
                for _, proj := range projects {
                        if f = proj.FindFile(ctx, t.Strval(ctx)); f != nil { break }
                }
        }
        return
}

func fullname(ctx Context, a Value, projects ...*Project) (f *File, s string, ok bool) {
        if f = asFile(ctx, a, projects...); f == nil {
                // no fullname
        } else if s = f.fullname(); filepath.IsAbs(s) {
                ok = true
        } else {
                // s = ""
        }
        return
}

func fullnameOrStrval(ctx Context, a Value, projects ...*Project) (s string) {
        var ok bool
        if _, s, ok = fullname(ctx, a, projects...); !ok {
                s = a.Strval(ctx)
        }
        return
}

// see optFullname and parseOpt
func asOptFullname(ctx Context, val Value, projects ...*Project) (file *File, s string, ok bool) {
        if file, s, ok = fullname(ctx, val, projects...); ok {
                // done
        } else if s = val.Strval(ctx); s == "" {
                // ...
        } else if filepath.IsAbs(s) {
                file = stat(ctx, s, "", "")
                ok = true
        }
        return
}

func asOptFullname2(ctx Context, val Value, projects ...*Project) (file *File, s string, ok bool) {
        if file, s, ok = asOptFullname(ctx, val, projects...);
        file == nil && s != "" && !filepath.IsAbs(s) {
                for _, proj := range projects {
                        if file = proj.FindFile(ctx, s); file != nil {
                                s = file.fullname()
                                ok = filepath.IsAbs(s)
                                break
                        }
                }
        }
        return
}

type builtinFullNameOpts struct {
        generalOpts
}
func builtinFullName(ctx Context, args... Value) (res Value) {
        var (
                closured = closureProjects(ctx)
                opts builtinFullNameOpts
                l []Value
                s string
                ok bool
        )
        for _, a := range parseOpts(ctx, &opts, expandPlainValue, args...) {
                if opts.debug > 0 {
                        if f, ok := a.(*File); ok {
                                warn(ctx, "dir=%v sub=%v name=%v", f.dir, f.sub, f.name).debug(opts.debug)
                        } else {
                                warn(ctx, "%T %v", a, a).debug(opts.debug,1)
                        }
                }
                if _, s, ok = asOptFullname2(ctx, a, closured...); ok || s != "" {
                        l = append(l, MakeString(a.Position(), s))
                } else {
                        l = append(l, a)
                }
        }
        res = MakeListOrScalar(ctx.Position(), l)
        return
}

type builtinBaseOpts struct {
        generalOpts
}
func basex(ctx Context, n int, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts builtinBaseOpts
                l []Value
        )
        for _, a := range parseOpts(ctx, &opts, expandPlainValue, args...) {
                s := fullnameOrStrval(ctx, a)
                d := filepath.Dir(s)
                s = filepath.Base(s)
                for i := n-1; 0 < i; i -= 1 {
                        s = filepath.Join(filepath.Base(d), s)
                        d = filepath.Dir(d)
                }
                l = append(l, MakeString(pos, s))
        }
        res = MakeListOrScalar(pos, l)
        return
}
func builtinBase (ctx Context, args... Value) Value { return basex(ctx, 1, args...) }
func builtinBase2(ctx Context, args... Value) Value { return basex(ctx, 2, args...) }
func builtinBase3(ctx Context, args... Value) Value { return basex(ctx, 3, args...) }
func builtinBase4(ctx Context, args... Value) Value { return basex(ctx, 4, args...) }
func builtinBase5(ctx Context, args... Value) Value { return basex(ctx, 5, args...) }
func builtinBase6(ctx Context, args... Value) Value { return basex(ctx, 6, args...) }
func builtinBase7(ctx Context, args... Value) Value { return basex(ctx, 7, args...) }
func builtinBase8(ctx Context, args... Value) Value { return basex(ctx, 8, args...) }
func builtinBase9(ctx Context, args... Value) Value { return basex(ctx, 9, args...) }

type builtinDirOpts struct {
        generalOpts
}
func dirx(ctx Context, n int, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts builtinDirOpts
                l []Value
                s string
        )
        for _, a := range parseOpts(ctx, &opts, expandPlainValue, args...) {
                if opts.fullname {
                        s = fullnameOrStrval(ctx, a)
                } else {
                        s = a.Strval(ctx)
                }
                s = filepath.Dir(s)
                for i := n-1; 0 < i; i -= 1 { s = filepath.Dir(s) }
                l = append(l, MakePathStr(pos, s))
        }
        res = MakeListOrScalar(pos, l)
        return
}

func undirx(ctx Context, n int, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts builtinDirOpts
                l []Value
                s string
        )
        for _, a := range parseOpts(ctx, &opts, expandPlainValue, args...) {
                if opts.fullname {
                        s = fullnameOrStrval(ctx, a)
                } else {
                        s = a.Strval(ctx)
                }
                var v = strings.Split(s, PathSep)
                if i := len(v); i == 0 {
                        // v is empty
                } else if n < i {
                        v = v[n:]
                } else {
                        v = v[i-1:] // empty
                }
                l = append(l, MakePathStr(pos, filepath.Join(v...)))
        }
        res = MakeListOrScalar(pos, l)
        return
}

func builtinDir (ctx Context, args... Value) (res Value) { return dirx(ctx, 1, args...) }
func builtinDir2(ctx Context, args... Value) (res Value) { return dirx(ctx, 2, args...) }
func builtinDir3(ctx Context, args... Value) (res Value) { return dirx(ctx, 3, args...) }
func builtinDir4(ctx Context, args... Value) (res Value) { return dirx(ctx, 4, args...) }
func builtinDir5(ctx Context, args... Value) (res Value) { return dirx(ctx, 5, args...) }
func builtinDir6(ctx Context, args... Value) (res Value) { return dirx(ctx, 6, args...) }
func builtinDir7(ctx Context, args... Value) (res Value) { return dirx(ctx, 7, args...) }
func builtinDir8(ctx Context, args... Value) (res Value) { return dirx(ctx, 8, args...) }
func builtinDir9(ctx Context, args... Value) (res Value) { return dirx(ctx, 9, args...) }
func builtinDirs(ctx Context, args... Value) (res Value) {
        var n int
        if x := len(args); x > 0 {
                if v, ok := Scalar(args[0]).(*Int); ok {
                        args, n = args[1:], int(v.int64)
                } else if v, ok := Scalar(args[x-1]).(*Int); ok {
                        args, n = args[:x-1], int(v.int64)
                } else {
                        erro(ctx, "require (first/last) integer argument (first=%T, last=%T)", args[0], args[x-1]).debug(1)
                        return
                }
        }
        res = dirx(ctx, n, args...)
        return
}

func builtinUndir (ctx Context, args... Value) (res Value) { return undirx(ctx, 1, args...) }
func builtinUndir2(ctx Context, args... Value) (res Value) { return undirx(ctx, 2, args...) }
func builtinUndir3(ctx Context, args... Value) (res Value) { return undirx(ctx, 3, args...) }
func builtinUndir4(ctx Context, args... Value) (res Value) { return undirx(ctx, 4, args...) }
func builtinUndir5(ctx Context, args... Value) (res Value) { return undirx(ctx, 5, args...) }
func builtinUndir6(ctx Context, args... Value) (res Value) { return undirx(ctx, 6, args...) }
func builtinUndir7(ctx Context, args... Value) (res Value) { return undirx(ctx, 7, args...) }
func builtinUndir8(ctx Context, args... Value) (res Value) { return undirx(ctx, 8, args...) }
func builtinUndir9(ctx Context, args... Value) (res Value) { return undirx(ctx, 9, args...) }
func builtinUndirs(ctx Context, args... Value) (res Value) {
        var n = 0
        if x := len(args); x > 0 {
                if v, ok := Scalar(args[0]).(*Int); ok {
                        args, n = args[1:], int(v.int64)
                } else if v, ok := Scalar(args[x-1]).(*Int); ok {
                        args, n = args[:x-1], int(v.int64)
                } else {
                        erro(ctx, "require (first/last) integer argument (first=%T, last=%T)", args[0], args[x-1]).debug(1)
                        return
                }
        }
        return undirx(ctx, n, args...)
}

func builtinDirChop(ctx Context, args... Value) (res Value) {
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
                        erro(ctx, "require (first/last) integer argument (first=%T, last=%T)", args[0], args[x-1]).debug(1)
                        return

                }
        }
        for _, a := range args {
                var v = strings.Split(a.Strval(ctx), PathSep)
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
                t string
        )
        for i, a := range args {
                if s := a.Strval(ctx); i == 0 {
                        t = s
                } else if s, err = filepath.Rel(t, s); err == nil {
                        l = append(l, MakeString(a.Position(), s))
                } else {
                        erro(ctx, "%v", err)
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
                )
                switch t := a.(type) {
                case *Pair: // mkdir name => perm name => perm
                        name = t.Key.Strval(ctx)
                        perm = permVal(ctx, t.Value,0600)
                case *Group: // mkdir (name perm) (name perm)
                        if t.Len() == 2 {
                                name = t.Get(0).Strval(ctx)
                                perm = permVal(ctx, t.Get(1),0600)
                        } else {
                                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                                break
                        }
                case *List: // mkdir name perm, name perm, ...
                        if t.Len() == 2 {
                                name = t.Get(0).Strval(ctx)
                                perm = permVal(ctx, t.Get(1),0600)
                        } else {
                                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                                break
                        }
                default: // mkdir name perm, name perm, ...
                        name = args[i].Strval(ctx)
                        if i+1 < nargs {
                                perm = permVal(ctx, args[i+1],0600)
                                i += 1
                        }
                }
                if err := os.Mkdir(name, perm); err != nil {
                        erro(ctx, "%v", err).debug(1)
                        break
                }
        }
        return
}

func builtinMkdirAll(ctx Context, args... Value) (res Value) {
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        a = args[i]
                        name string
                        perm os.FileMode
                )
                switch t := a.(type) {
                case *Pair: // mkdir name => perm name => perm
                        name = t.Key.Strval(ctx)
                        perm = permVal(ctx, t.Value,0600)
                case *Group: // mkdir (name perm) (name perm)
                        if t.Len() == 2 {
                                name = t.Get(0).Strval(ctx)
                                perm = permVal(ctx, t.Get(1),0600)
                        } else {
                                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                                break
                        }
                case *List: // mkdir name perm, name perm, ...
                        if t.Len() == 2 {
                                name = t.Get(0).Strval(ctx)
                                perm = permVal(ctx, t.Get(1),0600)
                        } else {
                                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                                break
                        }
                default: // mkdir name perm, name perm, ...
                        if name = args[i].Strval(ctx); i+1 < nargs {
                                perm = permVal(ctx, args[i+1],0600)
                                i += 1
                        }
                }
                if err := os.MkdirAll(name, perm); err != nil {
                        erro(ctx, "%v", err).debug(1)
                        break
                }
        }
        return
}

func builtinChdir(ctx Context, args... Value) (res Value) {
        if len(args) == 1 {
                var str = args[0].Strval(ctx)
                if err := lockCD(str, 0); err != nil {
                        erro(ctx, "%v", err).debug(1)
                }
        } else {
                erro(ctx, "wrong number of arguments: %v", len(args))
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
                )
                switch t := a.(type) {
                case *Pair: // rename oldname=newname
                        oldname = t.Key.Strval(ctx)
                        newname = t.Value.Strval(ctx)
                case *Group: // rename (oldname newname) (old new)
                        if t.Len() == 2 {
                                oldname = t.Get(0).Strval(ctx)
                                newname = t.Get(1).Strval(ctx)
                        } else {
                                erro(ctx, "wrong size of group `%v'", t).of(t).debug(1)
                                break
                        }
                case *List: // rename oldname newname, old new, ...
                        if t.Len() == 2 {
                                oldname = t.Get(0).Strval(ctx)
                                newname = t.Get(1).Strval(ctx)
                        } else {
                                erro(ctx, "wrong size of list `%v'", t).of(t).debug(1)
                                break
                        }
                default: // rename newname oldname  newname oldname ...
                        if i+1 < nargs {
                                oldname = args[i+0].Strval(ctx)
                                newname = args[i+1].Strval(ctx)
                                i += 1
                        } else {
                                erro(ctx, "Wrong arguments `%v'", args).of(t).debug(1)
                                break
                        }
                }
                if err := os.Rename(oldname, newname); err != nil {
                        erro(ctx, "%v", err).at(ctx.Position()).debug(1)
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
                closured = closureProjects(ctx)
                opts builtinRemoveOpts
                names []string
                str string
                ok bool
        )
        for _, a := range parseOpts(ctx, &opts, expandPlainValue, args...) {
                var (
                        ctx = positional(ctx, a.Position())
                        file *File
                        err error
                )
                if isTrivial(a) {
                        // ignore
                } else if a.patterned(ctx) {
                        if names, err = filepath.Glob(a.Strval(ctx)); err != nil {
                                erro(ctx, "%v", err).debug(1)
                                return
                        }
                        for _, s := range names {
                                if opts.verbose { prompt(ctx, "remove %s\n", s) }
                                if opts.debug   { info(ctx, "remove %s", s).debug(1) }
                                if opts.all {
                                        err = os.RemoveAll(s)
                                } else {
                                        err = os.Remove(s)
                                }
                                if err != nil {
                                        erro(ctx, "remove failed: %v", err)
                                        return
                                }
                        }
                        continue
                } else if file, str, ok = asOptFullname2(ctx, a, closured...); !ok || str == "" {
                        if file != nil { ok = true } else
                        if opts.all {
                                warn(ctx, "not a file: %v (%T)", a, a)
                                warn(ctx, "not a file: %s (%v)", str, file)
                                warn(ctx, "%in %v", closured)
                                warnstack(ctx, 3, "").debug(8)
                        } else {
                                erro(ctx, "not a file: %v (%T)", a, a)
                                erro(ctx, "not a file: %v (%v)", str, file)
                                erro(ctx, "in %v", closured)
                                errostack(ctx, 3, "").debug(16)
                                break
                        }
                } else {
                        if opts.verbose { prompt(ctx, "remove %s\n", str) }
                        if opts.debug   { warn(ctx, "remove %s", str).debug(1) }
                        if opts.all {
                                err = os.RemoveAll(str)
                        } else {
                                err = os.Remove(str)
                        }
                        if err != nil {
                                erro(ctx, "%v", err)
                                erro(ctx, "source: %v (%T)", a, a)
                                erro(ctx, "source: %v", str).debug(1)
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
                closured = closureProjects(ctx)
                opts builtinRemoveAllOpts
                names []string
                str string
                ok bool
        )
        for _, a := range parseOpts(ctx, &opts, expandPlainValue, args...) {
                var ctx = positional(ctx, a.Position())
                if a.patterned(ctx) {
                        var err error
                        if names, err = filepath.Glob(a.Strval(ctx)); err != nil {
                                erro(ctx, "%v", err).debug(1)
                                return
                        }
                        for _, s := range names {
                                if opts.verbose { info(ctx, "remove %s", s).at(a.Position()) }
                                if err = os.RemoveAll(s); err != nil {
                                        erro(ctx, "%v", err).debug(1)
                                        return
                                }
                        }
                } else if _, str, ok = asOptFullname2(ctx, a, closured...); !ok || str == "" {
                        erro(ctx, "%v is not a file", a).debug(1)
                        break
                } else {
                        if opts.verbose { info(ctx, "remove %s", str) }
                        if opts.debug   { info(ctx, "remove %s", str).debug(1) }
                        if err := os.RemoveAll(str); err != nil {
                                erro(ctx, "remove failed: %v", err).debug(1)
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
                        e error
                )
                switch t := a.(type) {
                case *Pair: // truncate name => size old => new
                        name = t.Key.Strval(ctx)
                        if size, e = t.Value.Integer(ctx); e != nil {
                                erro(ctx, "%v: %v", t.Value, e).debug(1)
                        }
                case *Group: // truncate (name size) (old new)
                        if t.Len() == 2 {
                                name = t.Get(0).Strval(ctx)
                                if size, e = t.Get(1).Integer(ctx); e != nil {
                                        erro(ctx, "%v: %v", t.Get(1), e).debug(1)
                                }
                        } else {
                                erro(ctx, "Wrong size of group `%v'", t).debug(1)
                                break
                        }
                case *List: // truncate name size, old new, ...
                        if t.Len() == 2 {
                                name = t.Get(0).Strval(ctx)
                                if size, e = t.Get(1).Integer(ctx); e != nil {
                                        erro(ctx, "%v: %v", t.Get(1), e).debug(1)
                                }
                        } else {
                                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                                break
                        }
                default: // truncate name size  name size ...
                        if i+1 < nargs {
                                name = args[i+0].Strval(ctx)
                                if size, e = args[i+1].Integer(ctx); e != nil {
                                        erro(ctx, "%v: %v", args[i+1], e).debug(1)
                                }
                                i += 1
                        } else {
                                erro(ctx, "Wrong arguments `%v'", args).debug(1)
                                break
                        }
                }
                if err := os.Truncate(name, size); err != nil {
                        erro(ctx, "%v", err).debug(1)
                        break
                }
        }
        return
}

type builtinLinkOpts struct {
        // TODO: ...
}
func builtinLink(ctx Context, args... Value) (res Value) {
        var opts builtinLinkOpts
        args = parseOpts(ctx, &opts, expandPlainValue, args...)
        for i, nargs := 0, len(args); i < nargs; i += 1 {
                var (
                        oldname, newname string
                        a = args[i]
                )
                switch t := a.(type) {
                case *Pair: // link oldname => newname old => new
                        oldname = t.Key.Strval(ctx)
                        newname = t.Value.Strval(ctx)
                case *Group: // link (oldname newname) (old new)
                        if t.Len() == 2 {
                                oldname = t.Get(0).Strval(ctx)
                                newname = t.Get(1).Strval(ctx)
                        } else {
                                erro(ctx, "Wrong size of group `%v'", t).debug(1)
                                break
                        }
                case *List: // link oldname newname, old new, ...
                        if t.Len() == 2 {
                                oldname = t.Get(0).Strval(ctx)
                                newname = t.Get(1).Strval(ctx)
                        } else {
                                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                                break
                        }
                default: // link oldname newname  oldname newname ...
                        if i+1 < nargs {
                                oldname = args[i+0].Strval(ctx)
                                newname = args[i+1].Strval(ctx)
                                i += 1
                        } else {
                                erro(ctx, "Wrong arguments `%v'", args).debug(1)
                                break
                        }
                }
                if err := os.Link(oldname, newname); err != nil {
                        erro(ctx, "%v", err).debug(1)
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
        generalOpts
        path bool `p,path`
        force bool `f,force`
        update bool `u,update`
        relative bool `r,relative;l,rel`
}
func builtinSymlink(ctx Context, args... Value) (res Value) {
        var opts builtinSymlinkOpts
        args = parseOpts(ctx, &opts, expandPlainValue, args...)
ForArgs:
        for i, na := 0, len(args); i < na; i += 1 {
                var (
                        opts = opts // make a copy
                        oldNameVal, newNameVal Value
                        oldName   , newName    string
                        aa []Value
                )
                switch t := args[i].(type) {
                case *Pair: // symlink oldName=newName oldName=>newName...
                        oldNameVal, newNameVal = t.Key, t.Value
                case *Group: // symlink (-u oldName newName) (-v oldName newName)...
                        if aa = parseOpts(ctx, &opts, expandPlainValue, t.Elems...); len(aa) != 2 {
                                erro(ctx, "expects two values for group").of(t).debug(1)
                                return
                        } else {
                                oldNameVal, newNameVal = aa[0], aa[1]
                        }
                case *List: // XXX: symlink old new, old new, ...
                        if aa = parseOpts(ctx, &opts, expandPlainValue, t.Elems...); len(aa) != 2 {
                                erro(ctx, "expects two values for list").of(t).debug(1)
                                return
                        } else {
                                oldNameVal, newNameVal = aa[0], aa[1]
                        }
                default:// Multiple pairs of names:
                        // symlink  new old, new old ...
                        // symlink  new old  new old ...
                        if i+1 < na {
                                oldNameVal = args[i+0]
                                newNameVal = args[i+1]
                                i += 1
                        } else {
                                var a, _ = ctx.autoGet("@")
                                var l, _ = ctx.autoGet("<")
                                var r, _ = ctx.autoGet(">")
                                prompt(ctx, "symlink: args=%v -> %v\n", args, t)
                                prompt(ctx, "symlink: %v, %v, %v\n", a, l, r)
                                errostack(ctx, 5, "expects pair of names (%T %v)", t, t).of(t).debug(6)
                                return
                        }
                }

                if oldName = oldNameVal.Strval(ctx); oldName == "" {
                        prompt(ctx, "symlink: args=%v\n", args)
                        prompt(ctx, "symlink: old=%v\n", oldNameVal)
                        errostack(ctx, 5, "empty old filename (%T)", oldNameVal).of(oldNameVal).debug(6)
                        return
                }
                if newName = newNameVal.Strval(ctx); newName == "" {
                        prompt(ctx, "symlink: args=%v\n", args)
                        prompt(ctx, "symlink: new=%v\n", newNameVal)
                        errostack(ctx, 5, "empty new filename (%T)", newNameVal).of(newNameVal).debug(6)
                        return
                }

                if opts.force {
                        if err := os.Remove(newName); err != nil {
                                erro(ctx, "%v", err).of(newNameVal).debug(1)
                        }
                } else if opts.update {
                        if s, err := os.Readlink(newName); err != nil {
                                if false {
                                        prompt(ctx, "%v: readlink failed (%T)\n", newName, err)
                                        erro(ctx, "%v", err).of(newNameVal)
                                        errostack(ctx, 6, "%v", ctx).of(newNameVal).debug(8)
                                }
                        } else if s == newName {
                                continue ForArgs
                        } else if err = os.Remove(newName); err != nil {
                                if true {
                                        prompt(ctx, "%v: remove old symlink failed (%T)\n", newName, err)
                                        erro(ctx, "%v", err).of(newNameVal)
                                        errostack(ctx, 6, "%v", ctx).of(newNameVal).debug(8)
                                }
                        }
                }

                if opts.relative && filepath.IsAbs(oldName) {
                        var (
                                dir = filepath.Dir(newName)
                                s = oldName
                                err error
                        )
                        if oldName, err = filepath.Rel(dir, oldName); err != nil {
                                prompt(ctx, "%s: symlink: rel(%s, %s)\n", newName, dir, s)
                                erro(ctx, "%v", err).of(newNameVal)
                                errostack(ctx, 8, "%v", ctx).of(newNameVal).debug(10)
                                return
                        }
                }

                if dir := filepath.Dir(newName); opts.path && dir != "." && dir != PathSep {
                        if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                                erro(ctx, "%v", err).of(newNameVal).debug(1)
                                return
                        }
                }

                if err := os.Symlink(oldName, newName); err != nil {
                        if opts.verbose { prompt(ctx, "… %s\n", err) }
                        break
                } else if opts.verbose {
                        var d = trimPromptString(newName)
                        var s = filepath.Base(oldName)
                        prompt(ctx, "%s -> %s …… ok\n", d, s)
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
        )
        if proj == nil {
                erro(ctx, "unknown current context").debug(1)
                return
        }

        args = parseOpts(ctx, &opts, expandPlainValue, args...)

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
                if s = a.Strval(ctx); filepath.IsAbs(s) {
                        file = stat(ctx, s, "", "")
                } else {
                        file = stat(ctx, s, "", proj.absPath)
                }
                if file == nil { file = proj.FindFile(ctx, s) }
                if file != nil { check(file) }
        }

        for _, a := range args {
                switch t := a.(type) {
                case *File: check(t)
                case *Path: checkstat(a)
                default:    checkstat(a)
                }
        }

        res = MakeListOrScalar(pos, reses)
        return
}

/*func builtinFileSource(ctx Context, args... Value) (res Value) {
        var ( pos = ctx.Position(); err error )
        if args, err = mergeExpand(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "expand args failed: %v", err).debug(1)
                return
        }

        var proj = ctx.Project()//current()
        if proj == nil {
                erro(ctx, "unknown current context").debug(1)
                return
        }

        var l []Value
        for _, a := range args {
                var str string
                if str, err = a.Strval(ctx); err != nil {
                        erro(ctx, "strval '%v' failed: %v", err).debug(1)
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
                proj *Project
                list []Value
        )

        args = parseOpts(ctx, &opts, expandPlainValue, args...)

        if opts.caller && false {
                // program -> closure -> traversal -> ...
                if false {
                        proj = ctx.closure().Project()
                } else {
                        proj = ctx.traversal().Project()
                }
        } else {
                proj = ctx.Project()
        }
        for _, a := range args {
                var ctx = positional(ctx, a.Position())
                if file, ok := a.(*File); ok {
                        list = append(list, file)
                        if file.exists() { continue }
                        if opts.report { info(ctx, "%v is no such file", a).debug(1) }
                } else if file = proj.FindFile(ctx, a.Strval(ctx)); file != nil {
                        list = append(list, file)
                        if opts.report { info(ctx, "%v is no such file", a).debug(1) }
                } else {
                        erro(ctx, `%v: "%v" is not a file (%T)`, proj, a, a)
                        errostack(ctx, 3, "(%T): %v", ctx, proj).debug(16)
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
        )

        args = parseOpts(ctx, &opts, expandPlainValue, args...)

        var cwd string // TODO: get current work directory
        if proj = ctx.Project(); proj == nil {
                erro(ctx, "unknown current cntext").debug(1)
                return
        }

        var pos = ctx.Position()
        var list []Value
        for _, a := range args {
                var ( str string; names []string )
                if str = a.Strval(ctx); !filepath.IsAbs(str) {
                        str = filepath.Join(cwd, str)
                }

                var err error
                if names, err = filepath.Glob(str); err != nil {
                        erro(ctx, "glob '%v' failed: %v", str, err).debug(1)
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

func wildcardPathPatsInDir1(ctx Context, opts *wildcardOpts, pats ...Value) (files []*File) {
        var dir *os.File
        if fi, err := os.Stat(opts.dir); err != nil {
                erro(ctx, "%v", err).debug(1)
                return
        } else if !fi.IsDir() {
                erro(ctx, "not dir: %v", opts.dir).debug(1)
                return
        } else if dir, err = os.Open(opts.dir); err != nil {
                erro(ctx, "not dir: %v", opts.dir).debug(1)
                return
        }
        var dbg = false //strings.HasSuffix(opts.dir, "/llvm-project/llvm/include")
        var names, err = dir.Readdirnames(-1); dir.Close()
        if err != nil {
                erro(ctx, "readdir: %v", err).debug(1)
                return
        }
LoopNames:
        for _, name := range names {
                for _, x := range opts.exclude {
                        if ok, _, _ := x.match(ctx, name); ok { continue LoopNames }
                }
                for _, pat := range pats {
                        if p, ok := pat.(*Path); ok {
                                var full, s, stems = p.Elems[0].match(ctx, name)
                                if dbg { warn(ctx, "%v %v; %v %v %v", pat, name, full, s, stems) }
                                if !full { continue }

                                var (
                                        sub wildcardOpts = *opts
                                        subFiles []*File
                                )
                                if sub.dir = filepath.Join(opts.dir, name); len(p.Elems) == 2 {
                                        subFiles = wildcardPathPatsInDir1(ctx, &sub, p.Elems[1])
                                } else {
                                        p = MakePath(p.Elems[1].Position(), p.Elems[1:]...)
                                        subFiles = wildcardPathPatsInDir1(ctx, &sub, p)
                                }
                                for _, f := range subFiles {
                                        if true { assert(filepath.Base(f.dir) == name, "file.dir: %s != %s", f.dir, name) }
                                        if false {
                                                f.name = filepath.Join(name, f.name)
                                                f.dir = filepath.Dir(f.dir)
                                        } else if !f.change(filepath.Dir(f.dir), "", filepath.Join(name, f.name)) {
                                                prompt(ctx, "%v: %v: can't change file into %s/%s\n", opts.dir, f, name, f.name)
                                                errostack(ctx, 6, "can't change into: %v/%v", name, f.name).debug(12)
                                                return
                                        }
                                }
                                if dbg { warn(ctx, "%v %v %v", pat, name, subFiles) }
                                files = append(files, subFiles...)
                        } else if full, _, _ := pat.match(ctx, name); full {
                                if dbg { warn(ctx, "%v %v", pat, name) }
                                file := stat(ctx, name, "", opts.dir)
                                files = append(files, file)
                                break
                        }
                }
        }
        if dbg {
                prompt(ctx, "%v: has %d files\n", opts.dir, len(files))
                warn(ctx, "pats: %v", pats).debug(24)
        }
        return
}

func readDirNames(ctx Context, inDir string) (names []string) {
        var dir *os.File
        if fi, err := os.Stat(inDir); err != nil {
                erro(ctx, "%v", err).debug(1)
                return
        } else if !fi.IsDir() {
                erro(ctx, "not dir: %v", inDir).debug(1)
                return
        } else if dir, err = os.Open(inDir); err != nil {
                erro(ctx, "not dir: %v", inDir).debug(1)
                return
        }

        var _names, err = dir.Readdirnames(-1); dir.Close()
        if err != nil {
                erro(ctx, "readdir: %v", err).debug(1)
                return
        } else { names = _names }
        return
}

func wildcardPathSubDir(ctx Context, parentOpts *wildcardOpts, name string, p *Path, dbg bool) (subFiles []*File) {
        var full, s, stems = p.Elems[0].match(ctx, name)
        if dbg { warn(ctx, "%v %v; %v %v %v", p, name, full, s, stems) }
        if !full { return }

        var ( opts wildcardOpts = *parentOpts; inDir = opts.dir )
        if opts.dir = filepath.Join(inDir, name); len(p.Elems) == 2 {
                subFiles = wildcardPathPatsInDir(ctx, &opts, p.Elems[1])
        } else {
                p = MakePath(p.Elems[1].Position(), p.Elems[1:]...)
                subFiles = wildcardPathPatsInDir(ctx, &opts, p)
        }

        for _, f := range subFiles {
                if true { assert(filepath.Base(f.dir) == name, "file.dir: %s != %s", f.dir, name) }
                if false {
                        f.name = filepath.Join(name, f.name)
                        f.dir = filepath.Dir(f.dir)
                } else if !f.change(filepath.Dir(f.dir), "", filepath.Join(name, f.name)) {
                        prompt(ctx, "%v: %v: can't change file into %s/%s\n", inDir, f, name, f.name)
                        errostack(ctx, 6, "can't change into: %v/%v", name, f.name).debug(12)
                        return
                }
        }
        return
}

func wildcardPathPatsInDir2(ctx Context, opts *wildcardOpts, pats ...Value) (files []*File) {
        var dbg = strings.HasSuffix(opts.dir, "/external/llvm-project/llvm/include")
        var names []string
        for _, pat := range pats {
                if p, ok := pat.(*Path); ok {
                        if p.Elems[0].patterned(ctx) {
                                if names == nil { names = readDirNames(ctx, opts.dir) }
                                for i := 0; i < len(names); i += 1 {
                                        subFiles := wildcardPathSubDir(ctx, opts, names[i], p, dbg)
                                        if subFiles != nil { files = append(files, subFiles...) }
                                }
                        }

                        var s = p.Elems[0].Strval(ctx)
                        if fi, e := os.Stat(filepath.Join(opts.dir, s)); e == nil && fi.IsDir() {
                                subFiles := wildcardPathSubDir(ctx, opts, s, p, dbg)
                                if subFiles != nil { files = append(files, subFiles...) }
                        }
                } else {
                        if names == nil { names = readDirNames(ctx, opts.dir) }
                        for i := 0; i < len(names); i += 1 {
                                var name = names[i]
                                if full, _, _ := pat.match(ctx, name); full {
                                        file := stat(ctx, name, "", opts.dir)
                                        files = append(files, file)
                                        break
                                }
                        }
                }
        }
        return
}

func wildcardPathPatsInDir3(ctx Context, opts *wildcardOpts, pats ...Value) (files []*File) {
        var dbg = false //strings.Contains(opts.dir, "/external/llvm-project/llvm/include")
        var names = readDirNames(ctx, opts.dir)
        forNames: for _, name := range names {
                for _, x := range opts.exclude {
                        if full, _, _ := x.match(ctx, name); full { continue forNames }
                }
                for _, pat := range pats {
                        var ctx = positional(ctx, pat.Position())
                        var p, ok = pat.(*Path)
                        if !ok || len(p.Elems) <= 1 {
                                var full, s, stems = pat.match(ctx, name)
                                if dbg { warn(ctx, "%T %v %v; %v %v %v; %v %s", pat, pat, name, full, s, stems, p.Elems, opts.dir) }
                                if full {
                                        files = append(files, stat(ctx, name, "", opts.dir))
                                        continue forNames
                                } else {
                                        continue
                                }
                        } else if full, s, stems := p.Elems[0].match(ctx, name); !full {
                                continue
                        } else if dbg {
                                warn(ctx, "%T %v %v; %v %v", p.Elems[0], p.Elems[0], name, s, stems)
                        }

                        var subOpts wildcardOpts = *opts
                        subOpts.dir = filepath.Join(opts.dir, name)
                        if fi, err := os.Stat(subOpts.dir); err != nil {
                                erro(ctx, "%v", err).debug(1)
                                return
                        } else if !fi.IsDir() {
                                continue
                        }

                        var subPat = Path{ valbase:valbase{ p.Elems[1].Position() } }
                        subPat.Elems = p.Elems[1:]
                        if dbg { warn(ctx, "%T %v -> %v %v", pat, pat, &subPat, subOpts.dir) }

                        var subs = wildcardPathPatsInDir3(ctx, &subOpts, &subPat)
                        for _, f := range subs {
                                if false {
                                        f.name = filepath.Join(name, f.name)
                                        f.dir = filepath.Dir(f.dir)
                                } else if !f.change(filepath.Dir(f.dir), "", filepath.Join(name, f.name)) {
                                        prompt(ctx, "%v: %v: can't change file into %s/%s\n", opts.dir, f, name, f.name)
                                        errostack(ctx, 6, "can't change into: %v/%v", name, f.name).debug(12)
                                        return
                                }
                        }
                        files = append(files, subs...)
                }
        }
        return
}

func wildcardPathPatsInDir(ctx Context, opts *wildcardOpts, pats ...Value) (files []*File) {
        if true {
                files = wildcardPathPatsInDir3(ctx, opts, pats...)
        } else if false {
                files = wildcardPathPatsInDir2(ctx, opts, pats...) // FIXME: incorrect
        } else {
                files = wildcardPathPatsInDir1(ctx, opts, pats...)
        }
        if opts.filetype != "" {
               var res []*File
                for _, file := range files {
                        switch opts.filetype {
                        case "d", "dir" : if file.info.IsDir() { res = append(res, file) }
                        case "f", "file": if!file.info.IsDir() { res = append(res, file) }
                        default: erro(ctx, "unknown -filetype option: %s (file=%v)", opts.filetype, file).debug(1)
                        }
                }
                files = res
        }
        return
}

type wildcardOpts struct {
        generalOpts
        includeMissing bool `im,includemissing;m,include-missing`
        errorMissing bool `em,errormissing;e,error-missing`
        baseFiles bool `b,base,bases;bf,base-files`
        usedFiles bool `u,used;u,using;uf,used-files`
        name bool `s,str,string;n,name`
        exclude []Value `x,ex,excl,exclude,except,no,not`
        filetype string `ft,filetype,file-type` // dir, file, etc.
        dir string `di,dir,directory`
}
func builtinWildcard(ctx Context, args... Value) (res Value) {
        var (
                proj = ctx.Project()
                opts wildcardOpts
                files []*File
                err error
        )
        if proj == nil {
                erro(ctx, "unknown most derived context").debug(1)
                return
        }
        if args = parseOpts(ctx, &opts, expandPlainValue, args...); len(opts.exclude) > 0 {
                opts.exclude = mergeExpand(ctx, expandPlainValue, opts.exclude...)
        }

        if opts.timing {
                defer func(t time.Time) {
                        info(ctx, "wildcard time: %v", time.Now().Sub(t)).debug(1)
                } (time.Now())
        }

        if opts.dir != "" {
                files = wildcardPathPatsInDir(ctx, &opts, args...)
        } else if files, err = proj.wildcard(ctx, opts, args...); err != nil {
                erro(ctx, "wildcard failed: %v", err).debug(1)
                return
        }

        var vals []Value
        if opts.name {
        LoopFiles1:
                for _, file := range files {
                        for _, x := range opts.exclude {
                                if ok, _, _ := x.match(ctx, file); ok {
                                        continue LoopFiles1
                                }
                        }
                        vals = append(vals, MakeString(file.position, file.name))
                }
        } else {
        LoopFiles2:
                for _, file := range files {
                        for _, x := range opts.exclude {
                                if ok, _, _ := x.match(ctx, file); ok {
                                        continue LoopFiles2
                                }
                        }
                        vals = append(vals, file)
                }
        }
        res = MakeListOrScalar(ctx.Position(), vals)
        return
}

type builtinReadDirOpts struct {
        generalOpts
}
func builtinReadDir(ctx Context, args... Value) (res Value) {
        var l []Value
        for _, a := range args {
                if fis, err := ioutil.ReadDir(a.Strval(ctx)); err == nil {
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
        generalOpts
        trim bool `t,trim;ta,trim-all`
        trimLeft bool `tl,trim-left`
        trimRight bool `tr,trim-right`
}
func builtinReadFile(ctx Context, args... Value) (res Value) {
        var (
                closured = closureProjects(ctx)
                pos = ctx.Position()
                opts builtinReadFileOpts
                l []Value
        )
        for _, a := range parseOpts(ctx, &opts, expandPlainValue, args...) {
                var (
                        apos = a.Position()
                        str string
                        err error
                        s []byte
                        ok bool
                )
                if !apos.IsValid() { apos = pos }
                if _, str, ok = asOptFullname2(ctx, a, closured...); !ok || str == "" {
                        erro(ctx, "%v is not a file", a).at(apos).debug(1)
                        break
                } else if s, err = ioutil.ReadFile(str); err != nil {
                        erro(ctx, "read file failed: %v", err).at(apos).debug(1)
                        break
                } else {
                        if opts.trim      { s = bytes.TrimFunc     (s, unicode.IsSpace) } else
                        if opts.trimLeft  { s = bytes.TrimLeftFunc (s, unicode.IsSpace) } else
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
        generalOpts
        path bool `p,path`
}
func builtinWriteFile(ctx Context, args... Value) (res Value) {
        // $(write-file filename,content)
        // $(write-file -p filename,content)
        var opts builtinWriteFileOpts
        if len(args) > 0 {
                var va = parseOpts(ctx, &opts, expandPlainValue, args[1])
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
                        name = t.Key  .Strval(ctx)
                        data = t.Value.Strval(ctx)
                case *Group: // write-file (name text) (name text 0660)
                        if n := t.Len(); n < 4 && n > 0 {
                                name = t.Get(0).Strval(ctx)
                                if n > 1 { data = t.Get(1).Strval(ctx) }
                                if n > 2 { perm = permVal(ctx, t.Get(2),0600) }
                        } else {
                                erro(ctx, "Wrong size of group `%v'", t).debug(1)
                                break
                        }
                case *List: // write-file name text, name text 0660, ...
                        if n := t.Len(); n < 4 && n > 0 {
                                name = t.Get(0).Strval(ctx)
                                if n > 1 { data = t.Get(1).Strval(ctx) }
                                if n > 2 { perm = permVal(ctx, t.Get(2),0600) }
                        } else {
                                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                                break
                        }
                default: // write-file name text 0660  name text 0660 ...
                        name = args[i].Strval(ctx)
                        if i+1 < len(args) {
                                data = args[i+1].Strval(ctx)
                                i += 1
                        }
                        if i+1 < len(args) {
                                perm = permVal(ctx, args[i+1],0600)
                                i += 1
                        }
                }
                if name == "" {
                        continue ForArgs
                } else if dir := filepath.Dir(name); opts.path && dir != "." && dir != PathSep {
                        if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                                erro(ctx, "%v", err).debug(1)
                                return
                        }
                }
                if err := ioutil.WriteFile(name, []byte(data), perm); err != nil {
                        erro(ctx, "%v", err).debug(1)
                        break
                }
        }
        return
}

func touch(ctx Context, file Value, optMode uint32, optPath bool, ts ...time.Time) (err error) {
        var _, filename, _ = fullname(ctx, file)
        if  filename == "" {
                erro(ctx, "touch: file fullname of '%v' is empty", file, err).of(file).debug(1)
                return
        }

        if dir := filepath.Dir(filename); optPath && dir != "." && dir != PathSep {
                if err = os.MkdirAll(dir, os.FileMode(optMode|0733)); err != nil {
                        erro(ctx, "touch: %v", err).of(file).debug(1)
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
                        erro(ctx, "touch: %v", err).of(file).debug(1)
                } else if err = f.Close(); err != nil {
                        erro(ctx, "touch: %v", err).of(file).debug(1)
                }
        }
        if err == nil {
                if err = os.Chtimes(filename, at, mt); err != nil {
                        erro(ctx, "touch: %v", err).of(file).debug(1)
                }
        }
        if err == nil && mode != 0 && m != 0 && mode != m {
                if err = os.Chmod(filename, mode); err != nil {
                        erro(ctx, "touch: %v", err).of(file).debug(1)
                }
        }
        return
}

type builtinTouchFileOpts struct {
        generalOpts
        mode os.FileMode `m,mode;fm,filemode;fm,file-mode`
        path bool `p,path`
}
func builtinTouchFile(ctx Context, args... Value) (res Value) {
        // $(touch-file filename)
        // $(touch-file -p filename)
        var opts = builtinTouchFileOpts{ mode: os.FileMode(0600) }
        args = parseOpts(ctx, &opts, expandPlainValue, args...)
        for i := 0; i < len(args); i += 1 {
                if err := touch(ctx, args[i], uint32(opts.mode), opts.path); err != nil {
                        erro(ctx, "%v", err).debug(1)
                        break
                }
        }
        return
}

// $(grep 'status=1',$@)
// $(grep 'status=([0-9]+)',$1,$@)
type builtinGrepOpts struct {
        generalOpts
        //val Value `c,cap,capture,v,val,value,r,res,result`
}
func builtinGrep(ctx Context, args... Value) (res Value) {
        var (
                opts builtinGrepOpts
                vals, list []Value
                rxs []*regexp.Regexp
                result Value
                nargs int
                err error

        )
        if nargs = len(args); !(nargs == 2 || nargs == 3) {
                erro(ctx, "wants exactly 2 args, e.g. $(grep -1 '^example$',$(file))").debug(1)
                return
        }

        if vals = parseOpts(ctx, &opts, expandPlainValue, args[0]); nargs == 2 {
                args = args[1:]
        } else if nargs == 3 {
                result = args[1]
                args = args[2:]
        }
        for _, a := range vals {
                if s := a.Strval(ctx); s == "" {
                        erro(ctx, "empty regexp").of(a).debug(1)
                        return
                } else if r, e := regexp.Compile(s); e != nil {
                        erro(ctx, "%v", e).of(a).debug(1)
                        return
                } else {
                        rxs = append(rxs, r)
                }
        }

        vals = mergeExpand(ctx, expandPlainValue, args...)

        var pos = ctx.Position()
        var cc = autoContext{ Context:ctx, defs:make(autoDefMap) }
        var greped = func(line int, match []string) (done bool) {
                var vals []Value
                for i, s := range match {
                        if v, ok := cc.autoSet(fmt.Sprintf("%d",i), MakeString(pos, s)); !ok {
                                erro(ctx, "set $%d to '%s' failed", i, s).debug(1)
                                return
                        } else { vals = append(vals, v) }
                }
                defer func() {
                        for i, v := range vals {
                                if v, ok := cc.autoSet(fmt.Sprintf("%d",i), v); !ok {
                                        erro(ctx, "restore $%d to '%s' failed", i, v).debug(1)
                                }
                        }
                } ()
                list = append(list, result.expand(&cc, expandPlainValue))
                return
        }

        for _, a := range vals {
                var file *os.File
                var filename = a.Strval(ctx)
                if filename == "" {
                        errostack(ctx, 5, "empty filename: %v (%T) (%v -> %v)",
                                a, a, args, vals).of(a).debug(64)
                        return
                }
                if file, err = os.Open(filename); err != nil {
                        errostack(ctx, 5, "%v", err).of(a).debug(128)
                        return
                }
                defer file.Close()

                var (
                        line int // line number
                        scanner = bufio.NewScanner(file)
                )
                scanner.Split(bufio.ScanLines)
                ScanLines: for scanner.Scan() {
                        var text = scanner.Text()
                        line += 1 // starting from #1
                        for _, rx := range rxs {
                                if sm := rx.FindStringSubmatch(text); len(sm) > 0 {
                                        if greped(line, sm) { break ScanLines }
                                }
                        }
                }
        }
        if len(list) > 0 { res = MakeListOrScalar(pos, list) }
        return
}

var (
        rsAutoconf  = `AC_(CHECK_(FILES?|FUNCS?|HEADERS?|PROG|SIZEOF|TOOL)|DEFINE)\(([^\)]*?)\)`
        rsConfigRef = `[$%]\{([^\s\}]+)\}|@([^\s\@]+)@`
        rsConfigure = `^[\t ]*#[\t ]*(define|undef|smartdefine|smartdefine01|cmakedefine|cmakedefine01)[\t ]+([A-Za-z0-9_]+)(?:[\t ]+([^\n]*))?$`
        rxAutoconf  = regexp.MustCompile(rsAutoconf)
        rxConfigure = regexp.MustCompile(fmt.Sprintf(`(?m:%s)`, rsConfigure)) // m: multilines
        rxConfigRef = regexp.MustCompile(rsConfigRef)
)

func (project *Project) config(ctx Context, name string) (def *Def) {
        var obj Object
        if obj = project.resolveObject(ctx, name); !isNil(obj) { def, _ = obj.(*Def) }
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

                var (
                        def *Def
                        val Value
                )
                if def = project.config(ctx, name); def == nil {
                        if true { warnstack(ctx, 3, "%v undefined", name).debug(1) }
                        continue
                } else if val = def.Call(ctx); isNil(val) {
                        if true { warn(ctx, "%v is nil (%T)", name, val).of(def).debug(1) }
                        if cf := project.configuration(ctx); cf == nil {
                                erro(ctx, "%v: configuration file not defined", name, cf).of(def).debug(1)
                                return
                        } else if !cf.exists() {
                                prompt(ctx, "%s: file not exists (for %v)\n", cf.fullname(), name)
                                erro(ctx, "%v: configuration file not exists, try -conf first", name).of(def).debug(1)
                                return
                        }
                        continue
                }

                switch t := val.(type) {
                case *Plain: fmt.Fprintf(res, "%s", t.Value)
                case *answer, *boolean:
                        if i, e := t.Integer(ctx); e == nil {
                                fmt.Fprintf(res, "%d", i)
                        } else {
                                erro(ctx, "%: %v", t, i).debug(1)
                        }

                case *Group:
                        fmt.Fprintf(res, "%s", parseGroupValue(ctx, t).Strval(ctx))
                default:
                        fmt.Fprintf(res, "%s", val.Strval(ctx))
                }
        }
        if index < len(s) { fmt.Fprint(res, s[index:]) }
        result = res.String()
        return
}

// https://www.gnu.org/software/autoconf/manual/autoconf-2.67/autoconf.html
func autoconf(ctx Context, out *bytes.Buffer, project *Project, str string) (err error) {
        var num int
        for _, m := range rxAutoconf.FindAllStringSubmatch(str, -1) {
                info(ctx, "TODO: %v", m)
                num += 1
        }
        warn(ctx, "TODO: %d", num).debug(1)
        return
}

func configure(ctx Context, out *bytes.Buffer, project *Project, str string) (err error) {
        var index = 0
        if str, err = project.configExpand(ctx, str); err != nil {
                erro(ctx, "%v", err).debug(1)
                return
        }
        for _, m := range rxConfigure.FindAllStringSubmatchIndex(str, -1) {
                if _, err = out.WriteString(str[index:m[0]]); err != nil {
                        erro(ctx, "%v", err).debug(1)
                        return
                }
                index = m[1] // reset index immediately to keep forward

                var t bool
                var s string
                var verb = str[m[2]:m[3]]
                var name = str[m[4]:m[5]]
                var hasv = m[6] > m[0] && m[7] > m[6]
                var def *Def
                if def = project.config(ctx, name); def != nil {
                        t = def.True(ctx);
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
                        } else if va, _ = expandall(ctx, expandPlainValue, def.value); len(va) == 1 {
                                switch v := va[0].(type) {
                                case *answer, *boolean:
                                        if b := v.True(ctx); b {
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

                if _, err = out.WriteString(s); err != nil { erro(ctx, "%v", err); return }
        }
        if index < len(str) { _, err = out.WriteString(str[index:]) }
        return
}

func builtinReturn(ctx Context, args... Value) Value {
        return &returner{valbase{ctx.Position()}, args }
}
