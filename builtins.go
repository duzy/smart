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
        gc "context"
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

type BuiltinFunc struct {
        f func(ctx Context, w facet, args... Value) (Value)
        command bool
        n int // first n args to apply m
        m, o facet
}

var builtins = map[string]BuiltinFunc {
        `typeof`:       BuiltinFunc{builtinTypeof, false, 0, expandZero, expandZero},
        `origin`:       BuiltinFunc{builtinOrigin, false, 0, expandZero, expandZero},
        `defined`:      BuiltinFunc{builtinDefined, false, 0, expandZero, expandZero},

        `position`:     BuiltinFunc{builtinPosition, false, 0, expandZero, expandZero},
        `date`:         BuiltinFunc{builtinDate, false, 0, expandZero, expandZero},

        `debug`:        BuiltinFunc{builtinDebug, false, 0, expandZero, expandZero},
        `error`:        BuiltinFunc{builtinError, false, 0, expandZero, expandZero},
        //`warning`:      BuiltinFunc{builtinWarning, false, 0, expandZero, expandZero},

        //`assert`: BuiltinFunc{builtinAssert, false, 0, expandZero, expandZero},

        // $(defor) (aka. defined-or)
        `defor`:        BuiltinFunc{builtinDefOr, false, 0, expandZero, expandZero}, // $(defor $(x),$(y),$(z))  <=>  $(if $(defined $(x)),$(x),...)
        `or`:           BuiltinFunc{builtinOr, false, 0, expandZero, expandZero},
        `and`:          BuiltinFunc{builtinAnd, false, 0, expandZero, expandZero},
        //`xor`:          BuiltinFunc{builtinXor, false, 0, expandZero, expandZero},
        `not`:          BuiltinFunc{builtinNot, false, 0, expandZero, expandZero},

        `not-equal`:    BuiltinFunc{builtinNotEqual, false, 0, expandZero, expandZero},
        `equal`:        BuiltinFunc{builtinEqual, false, 0, expandZero, expandZero},
        `equals`:       BuiltinFunc{builtinEqual, false, 0, expandZero, expandZero},
        `match`:        BuiltinFunc{builtinMatch, false, 0, expandZero, expandZero},

        `greater`:      BuiltinFunc{builtinGreater, false, 0, expandZero, expandZero},
        `less`:         BuiltinFunc{builtinLess, false, 0, expandZero, expandZero},

        `case`:         BuiltinFunc{builtinCase, false, 0, expandZero, expandZero},
        `if`:           BuiltinFunc{builtinIf, false, 0, expandZero, expandZero},
        `ifeq`:         BuiltinFunc{builtinIfEq, false, 0, expandZero, expandZero},
        `ifne`:         BuiltinFunc{builtinIfNE, false, 0, expandZero, expandZero},

        `foreach`:      BuiltinFunc{builtinForEach, false, 1, expandPlaceholders, expandUnPlaceholders},
        `count`:        BuiltinFunc{builtinCount, false, 0, expandZero, expandZero},

        `env`:          BuiltinFunc{builtinEnv, false, 0, expandZero, expandZero},
        `closure`:      BuiltinFunc{builtinClosure, false, 0, expandZero, expandZero},
        `defs`:         BuiltinFunc{builtinDefList, false, 0, expandZero, expandZero},
        `call`:         BuiltinFunc{nil, false, 0, expandZero, expandZero},
        `auto`:         BuiltinFunc{nil, false, 0, expandZero, expandZero},
        `var`:          BuiltinFunc{nil, false, 0, expandZero, expandZero},
        `value`:        BuiltinFunc{builtinValue, false, 0, expandZero, expandZero},
        `list`:         BuiltinFunc{builtinList, false, 0, expandZero, expandZero},
        `sure-value`:   BuiltinFunc{builtinSureValue, false, 0, expandZero, expandZero},

        `shell`:        BuiltinFunc{builtinShell, false, 0, expandZero, expandZero},
        `which`:        BuiltinFunc{builtinWhich, false, 0, expandZero, expandZero},

        `serve-http`:   BuiltinFunc{builtinServeHttp, false, 0, expandZero, expandZero},
        `serve-https`:  BuiltinFunc{builtinServeHttps, false, 0, expandZero, expandZero},

        `plus`:     BuiltinFunc{builtinPlus, false, 0, expandZero, expandZero},
        `minus`:    BuiltinFunc{builtinMinus, false, 0, expandZero, expandZero},
        `multiply`: BuiltinFunc{builtinMultiply, false, 0, expandZero, expandZero},
        `mul`:      BuiltinFunc{builtinMultiply, false, 0, expandZero, expandZero},
        `divide`:   BuiltinFunc{builtinDivide, false, 0, expandZero, expandZero},
        `div`:      BuiltinFunc{builtinDivide, false, 0, expandZero, expandZero},

        `quote`:                BuiltinFunc{builtinQuote, false, 0, expandZero, expandZero},
        `quote-join`:           BuiltinFunc{builtinQuoteJoin, false, 0, expandZero, expandZero},
        `split-string`:         BuiltinFunc{builtinSplitString, false, 0, expandZero, expandZero},
        `split-quote`:          BuiltinFunc{builtinSplitQuote, false, 0, expandZero, expandZero},
        `split-quote-join`:     BuiltinFunc{builtinSplitQuoteJoin, false, 0, expandZero, expandZero},
        `split-join-quote`:     BuiltinFunc{builtinSplitJoinQuote, false, 0, expandZero, expandZero},
        `unique`:               BuiltinFunc{builtinUnique, false, 0, expandZero, expandZero},
        `join`:                 BuiltinFunc{builtinJoin, false, 0, expandZero, expandZero}, // concat
        `field`:                BuiltinFunc{builtinField, false, 0, expandZero, expandZero},
        `fields`:               BuiltinFunc{builtinFields, false, 0, expandZero, expandZero},

        //`usee`:       BuiltinFunc{builtinUsee, false, 0, expandZero, expandZero},
        `uses`:         BuiltinFunc{builtinUses, false, 0, expandZero, expandZero},

        `path`:         BuiltinFunc{builtinPath, false, 0, expandZero, expandZero},
        `bare`:         BuiltinFunc{builtinBare, false, 0, expandZero, expandZero}, // different from builtinBareword, for files, etc.
        `bareword`:     BuiltinFunc{builtinBareword, false, 0, expandZero, expandZero},
        `string`:       BuiltinFunc{builtinString, false, 0, expandZero, expandZero},
        `strval`:       BuiltinFunc{builtinStrval, false, 0, expandZero, expandZero},
        `strip`:        BuiltinFunc{builtinStrip, false, 0, expandZero, expandZero},
        `trim`:         BuiltinFunc{builtinTrim, false, 0, expandZero, expandZero},
        `trim-space`:   BuiltinFunc{builtinTrimSpace, false, 0, expandZero, expandZero},
        `trim-left`:    BuiltinFunc{builtinTrimLeft, false, 0, expandZero, expandZero},
        `trim-right`:   BuiltinFunc{builtinTrimRight, false, 0, expandZero, expandZero},
        `trim-prefix`:  BuiltinFunc{builtinTrimPrefix, false, 0, expandZero, expandZero},
        `trim-suffix`:  BuiltinFunc{builtinTrimSuffix, false, 0, expandZero, expandZero},
        `trim-ext`:     BuiltinFunc{builtinTrimExt, false, 0, expandZero, expandZero},

        `ext`:          BuiltinFunc{builtinExt, false, 0, expandZero, expandZero},

        `addprefix`:    BuiltinFunc{builtinAddPrefix, false, 0, expandZero, expandZero},
        `addsuffix`:    BuiltinFunc{builtinAddSuffix, false, 0, expandZero, expandZero},

        `printf`:       BuiltinFunc{builtinPrintf, false, 0, expandZero, expandZero},

        `uppercase`:    BuiltinFunc{builtinUpperCase, false, 0, expandZero, expandZero},
        `lowercase`:    BuiltinFunc{builtinLowerCase, false, 0, expandZero, expandZero},
        `title`:        BuiltinFunc{builtinTitle, false, 0, expandZero, expandZero},

        `indent`:       BuiltinFunc{builtinIndent, false, 0, expandZero, expandZero},

        `substring`:    BuiltinFunc{builtinSubstring, false, 0, expandZero, expandZero},

        // https://www.gnu.org/software/make/manual/html_node/Text-Functions.html
        `subst`:        BuiltinFunc{builtinSubst, false, 0, expandZero, expandZero},
        `patsubst`:     BuiltinFunc{builtinPatsubst, false, 0, expandZero, expandZero},

        `contains`:     BuiltinFunc{builtinContains, false, 0, expandZero, expandZero},
        `filter`:       BuiltinFunc{builtinFilter, false, 0, expandZero, expandZero},
        `filter-out`:   BuiltinFunc{builtinFilterOut, false, 0, expandZero, expandZero},

        `encode-base64`:BuiltinFunc{builtinEncodeBase64, false, 0, expandZero, expandZero},
        `decode-base64`:BuiltinFunc{builtinDecodeBase64, false, 0, expandZero, expandZero},

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
        `fullname`:   BuiltinFunc{builtinFullName, false, 0, expandZero, expandZero},

        `base`:       BuiltinFunc{builtinBase, false, 0, expandZero, expandZero},
        `base2`:      BuiltinFunc{builtinBase2, false, 0, expandZero, expandZero},
        `base3`:      BuiltinFunc{builtinBase3, false, 0, expandZero, expandZero},
        `base4`:      BuiltinFunc{builtinBase4, false, 0, expandZero, expandZero},
        `base5`:      BuiltinFunc{builtinBase5, false, 0, expandZero, expandZero},
        `base6`:      BuiltinFunc{builtinBase6, false, 0, expandZero, expandZero},
        `base7`:      BuiltinFunc{builtinBase7, false, 0, expandZero, expandZero},
        `base8`:      BuiltinFunc{builtinBase8, false, 0, expandZero, expandZero},
        `base9`:      BuiltinFunc{builtinBase9, false, 0, expandZero, expandZero},

        `dir`:        BuiltinFunc{builtinDir, false, 0, expandZero, expandZero},
        `dir2`:       BuiltinFunc{builtinDir2, false, 0, expandZero, expandZero},
        `dir3`:       BuiltinFunc{builtinDir3, false, 0, expandZero, expandZero},
        `dir4`:       BuiltinFunc{builtinDir4, false, 0, expandZero, expandZero},
        `dir5`:       BuiltinFunc{builtinDir5, false, 0, expandZero, expandZero},
        `dir6`:       BuiltinFunc{builtinDir6, false, 0, expandZero, expandZero},
        `dir7`:       BuiltinFunc{builtinDir7, false, 0, expandZero, expandZero},
        `dir8`:       BuiltinFunc{builtinDir8, false, 0, expandZero, expandZero},
        `dir9`:       BuiltinFunc{builtinDir9, false, 0, expandZero, expandZero},
        `dirs`:       BuiltinFunc{builtinDirs, false, 0, expandZero, expandZero}, // do `dir` n times

        `undir`:      BuiltinFunc{builtinUndir, false, 0, expandZero, expandZero},
        `undir2`:     BuiltinFunc{builtinUndir2, false, 0, expandZero, expandZero},
        `undir3`:     BuiltinFunc{builtinUndir3, false, 0, expandZero, expandZero},
        `undir4`:     BuiltinFunc{builtinUndir4, false, 0, expandZero, expandZero},
        `undir5`:     BuiltinFunc{builtinUndir5, false, 0, expandZero, expandZero},
        `undir6`:     BuiltinFunc{builtinUndir6, false, 0, expandZero, expandZero},
        `undir7`:     BuiltinFunc{builtinUndir7, false, 0, expandZero, expandZero},
        `undir8`:     BuiltinFunc{builtinUndir8, false, 0, expandZero, expandZero},
        `undir9`:     BuiltinFunc{builtinUndir9, false, 0, expandZero, expandZero},
        `undirs`:     BuiltinFunc{builtinUndirs, false, 0, expandZero, expandZero}, // do `undir` n times

        `dir-chop`:   BuiltinFunc{builtinDirChop, false, 0, expandZero, expandZero},

        `relative-dir`: BuiltinFunc{builtinRelativeDir, false, 0, expandZero, expandZero},

        // TODO: move these into builtin package `os'
        // `mkdir`:      BuiltinFunc{builtinMkdir, false, 0, expandZero, expandZero},     // os/file.go
        // `mkdir-all`:  BuiltinFunc{builtinMkdirAll, false, 0, expandZero, expandZero},  // os/path.go
        // `chdir`:      BuiltinFunc{builtinChdir, false, 0, expandZero, expandZero},     // os/file.go
        // `rename`:     BuiltinFunc{builtinRename, false, 0, expandZero, expandZero},    // os/file.go
        // `remove`:     BuiltinFunc{builtinRemove, false, 0, expandZero, expandZero},    // os/file_*.go
        // `remove-all`: BuiltinFunc{builtinRemoveAll, false, 0, expandZero, expandZero}, // os/path.go
        // `truncate`:   BuiltinFunc{builtinTruncate, false, 0, expandZero, expandZero},  // os/file_*.go
        // `link`:       BuiltinFunc{builtinLink, false, 0, expandZero, expandZero},      // os/file_*.go
        // `symlink`:    BuiltinFunc{builtinSymlink, false, 0, expandZero, expandZero},   // os/file_*.go

        `file`:       BuiltinFunc{builtinFile, false, 0, expandZero, expandZero},
        `stat`:       BuiltinFunc{builtinStat, false, 0, expandZero, expandZero},// stat (deprecates file-exists)
        `glob`:       BuiltinFunc{builtinGlob, false, 0, expandZero, expandZero},
        `wildcard`:   BuiltinFunc{builtinWildcard, false, 0, expandZero, expandZero},

        // TODO: move these into builtin package 'io/ioutil'
        `read-dir`:   BuiltinFunc{builtinReadDir, false, 0, expandZero, expandZero},   // io/ioutil/ioutil.go
        `read-file`:  BuiltinFunc{builtinReadFile, false, 0, expandZero, expandZero},  // io/ioutil/ioutil.go
        // `write-file`: BuiltinFunc{builtinWriteFile, false, 0, expandZero, expandZero}, // io/ioutil/ioutil.go
        // `touch-file`: BuiltinFunc{builtinTouchFile, false, 0, expandZero, expandZero},

        `grep`:       BuiltinFunc{builtinGrep, false, 1, expandDigits, expandUnDigits},

        `untraversed`: BuiltinFunc{builtinUntraversed, false, 1, expandZero, expandZero},

        // `return`:     BuiltinFunc{builtinReturn, false, 0, expandZero, expandZero},
}

var commands = map[string]BuiltinFunc {
        `print`:        BuiltinFunc{builtinPrint, true, 0, expandZero, expandZero},
        `printl`:       BuiltinFunc{builtinPrintl, true, 0, expandZero, expandZero},
        `println`:      BuiltinFunc{builtinPrintln, true, 0, expandZero, expandZero},
        //`printf`:       BuiltinFunc{builtinPrintf, true, 0, expandZero, expandZero},

        `assert`:       BuiltinFunc{builtinAssert, true, 0, expandZero, expandZero},

        //`error`:        BuiltinFunc{builtinError, true, 0, expandZero, expandZero},
        `warning`:      BuiltinFunc{builtinWarning, true, 0, expandZero, expandZero},

        `push-context`: BuiltinFunc{builtinPushContext, true, 0, expandZero, expandZero},
        `pop-context`:  BuiltinFunc{builtinPopContext, true, 0, expandZero, expandZero},

        `append`:       BuiltinFunc{builtinAppend, true, 0, expandZero, expandZero},

        //`read-dir`:     BuiltinFunc{builtinReadDir, true, 0, expandZero, expandZero},   // io/ioutil/ioutil.go
        //`read-file`:    BuiltinFunc{builtinReadFile, true, 0, expandZero, expandZero},  // io/ioutil/ioutil.go
        `write-file`:   BuiltinFunc{builtinWriteFile, true, 0, expandZero, expandZero}, // io/ioutil/ioutil.go
        `touch-file`:   BuiltinFunc{builtinTouchFile, true, 0, expandZero, expandZero},

        `mkdir`:        BuiltinFunc{builtinMkdir, true, 0, expandZero, expandZero},     // os/file.go
        `mkdir-all`:    BuiltinFunc{builtinMkdirAll, true, 0, expandZero, expandZero},  // os/path.go
        `chdir`:        BuiltinFunc{builtinChdir, true, 0, expandZero, expandZero},     // os/file.go
        `rename`:       BuiltinFunc{builtinRename, true, 0, expandZero, expandZero},    // os/file.go
        `remove`:       BuiltinFunc{builtinRemove, true, 0, expandZero, expandZero},    // os/file_*.go
        `remove-all`:   BuiltinFunc{builtinRemoveAll, true, 0, expandZero, expandZero}, // os/path.go
        `truncate`:     BuiltinFunc{builtinTruncate, true, 0, expandZero, expandZero},  // os/file_*.go
        `link`:         BuiltinFunc{builtinLink, true, 0, expandZero, expandZero},      // os/file_*.go
        `symlink`:      BuiltinFunc{builtinSymlink, true, 0, expandZero, expandZero},   // os/file_*.go

        `return`:       BuiltinFunc{builtinReturn, true, 0, expandZero, expandZero},
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
                        if t := v == nil || v.True(ctx); true { val.SetBool(t) }
                case reflect.Float32, reflect.Float64:
                        if t, e := v.Float(ctx); e == nil { val.SetFloat(t) } else {
                                erro(ctx, "%v: %v", v, e).debug(10)
                        }
                case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
                        if t, e := v.Integer(ctx); e == nil { val.SetInt(t) } else {
                                erro(ctx, "%v: %v", v, e).debug(10)
                        }
                case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
                        if t, e := v.Integer(ctx); e == nil { val.SetUint(uint64(t)) } else {
                                erro(ctx, "%v: %v", v, t).debug(10)
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
                default: erro(of(ctx,v), "option type unsupported: %T %v -> %v, %v", v, v, val.Kind(), val.Type()).debug(1)
                }
                case reflect.Ptr: switch val.Type().Elem().String() {
                case "smart.optFullname":
                        var x Value
                        if x = v.expand(ctx, plain|expandFullName); isNone(x) {
                                erro(of(ctx,v), "expecting file value: %T %v", v, v).debug(1)
                                return
                        }

                        if _, s, ok = asOptFullname(ctx, x/*, projects...*/); ok && s != "" {
                                val.Set(reflect.ValueOf(&optFullname{ s, x }))
                        } else {
                                var tv = autoGet(ctx,"@")
                                var f = matchFile(ctx, s)
                                erro(of(ctx,v), "not a file: %v -> %v -> %s (@=%T %v %v)", v, x, s, tv, tv, f)
                                errostack(ctx, 5, "%v", ctx).debug(16)
                        }

                        if false {
                                vi := val.Interface().(*optFullname)
                                warn(of(ctx,v), "%v %v %v", /*current()*/ctx.Project(), v, vi.string).debug(true,1)
                        }
                case "smart.File":
                        var x Value
                        if x = v.expand(ctx, plain); isNone(x) {
                                erro(of(ctx,v), "expecting file value: %T %v", v, v).debug(1)
                                return
                        }

                        if file, y := toFile(x); y {
                                val.Set(reflect.ValueOf(file))
                        } else if proj := /*current()*/ctx.Project(); proj == nil {
                                erro(of(ctx,x), "no current project to find file '%v'", s).debug(1)
                        } else if file = proj.file(ctx, x.Strval(ctx)); file != nil {
                                val.Set(reflect.ValueOf(file))
                        } else {
                                erro(of(ctx,v), "'%s' is not a file", s).debug(1)
                        }
                case "regexp.Regexp":
                        if rx, e := regexp.Compile(v.Strval(ctx)); e != nil {
                                erro(of(ctx,v), "compile regexp '%v' failed: %v", v, e).debug(1)
                        } else {
                                val.Set(reflect.ValueOf(rx))
                        }
                default:
                        erro(of(ctx,v), "option type unsupported: %T %v -> %v, %v", v, v, val.Elem().Kind(), val.Type().Elem()).debug(1)
                }
                default: switch val.Type().String() {
                case "fs.FileMode": // aka. reflect.Uint32
                        if t, e := v.Integer(ctx); e == nil { val.SetUint(uint64(t)) } else {
                                erro(ctx, "%v: %v", v, t).debug(1)
                        }
                case "regex.Regex": // aka. reflect.Ptr
                        erro(of(ctx,v), "TODO: regexp: %T %v -> %v, %v", v, v, val.Kind(), val.Type()).debug(1)
                default:
                        erro(of(ctx,v), "option type unsupported: %T %v -> %v, %v", v, v, val.Kind(), val.Type()).debug(1)
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
                } else if pair, y := arg.(*Pair); y {
                        if flag, okay = pair.Key.(*Flag); okay { value = pair.Value }
                } else if aa, y := arg.(*Argumented); y {
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

func parseOpts(ctx Context, iOpts interface{}, w facet, args... Value) (rest []Value) {
        if w&^expandNone == 0 {
                rest = merge(args...) // NOTE: set the returning args first of all!
        } else {
                if false && // FIXME: the args[1].expand(ctx, w) causes 'd.x == nil'
                        w == 0 && len(args)>1 && args[0].String() == "-plain" &&
                        args[1].String() == "$(configure~$(target.sys).features)" {
                        var d = args[1].(*delegate)
                        var t = args[1].expand(ctx, w)
                        warn(ctx, "%v ; w=%016b", args, w)
                        warn(ctx, "%T %v %p -> %T %v %p %p", d.x, args[1], d,
                                t.(*delegate).x, t, t, t.(*delegate)).debug(1)
                }
                rest = mergex(ctx, w, args...)
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
                                if rest = parseOpts(ctx, gen, w, rest...); gen.debug>0 {
                                        gen.debug = gen.debug * 2
                                }
                                if false { prompt(ctx, "%v: %v ; %v -> %v", ft.Name, *gen, args, rest).debug(1) }
                        } else {
                                rest = parseOpt(ctx, ft.Tag, fv, rest...)
                        }
                }
                if gen == nil { return }

                // var a = mergex(ctx, expandFullName, rest...)
                // for _, v := range rest {
                //         if strings.Contains(v.String(), ".configure/library") {
                //                 var t = v.expand(ctx, expandFullName)
                //                 warn(of(ctx,v), "%v: %v; %v", v, t, a).debug(1)
                //         }
                // }
                if gen.fullname { rest = mergex(ctx, expandFullName, rest...) }
        } else {
                erro(ctx, "opts is not ptr of struct: %v", opts.Kind()).debug(1)
        }
        return
}

func parseHeadArgs(ctx Context, iOpts interface{}, w facet, args... Value) (head, rest []Value) {
        if len(args) == 0 {
                // zero args
        } else if head = parseOpts(ctx, iOpts, w, args[0]); len(head) > 0 {
                rest = args[1:] //mergex(ctx, w, args[1:]...)
        } else if len(args) == 1 {
                // done
        } else if head = mergex(ctx, w, args[1]); len(args) > 2 {
                rest = args[2:] //mergex(ctx, w, args[2:]...)
        }
        return
}

func parseHeadArgsMerge(ctx Context, iOpts interface{}, w facet, args... Value) (res []Value) {
        var head, rest = parseHeadArgs(ctx, iOpts, w, args...)
        res = append(head, rest...)
        return
}

func parseHeadArgsRequired(ctx Context, iOpts interface{}, w facet, args... Value) (head, rest []Value) {
        head, rest = parseHeadArgs(ctx, iOpts, w, args...)
        if len(head) == 0 || len(rest) == 0 {
                erro(ctx, "insufficient number of arguments").debug(6)
        }
        return
}

func typeof(arg interface{}) (s string) {
        switch a := arg.(type) {
        case *List:
                if n := len(a.Elems); n == 1 {
                        switch v := a.Elems[0].(type) {
                        case *delegate: // FIXME: recursively undelegate types
                                if d, _ := v.x.(*def); d != nil {
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

type builtinTypeofOpts struct {
        generalOpts
        expand bool `x,e,ex,exp,expand`
}
func builtinTypeof(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts builtinTypeofOpts
                elems []Value
        )
        for _, arg := range parseHeadArgsMerge(ctx, &opts, 0, args...) {
                if opts.expand { arg = arg.expand(ctx, plain) }
                // Arguments are passed in a list:
                //   $(fun abc)                 args: (abc)
                //   $(fun a,b,c)               args: (a),(b),(c)
                //   $(fun a b c,1 2 3)         args: (a b c),(1 2 3)
                elems = append(elems, MakeString(pos, typeof(arg)))
        }
        return MakeListOrScalar(pos, elems)
}

type builtinOriginOpts struct {
        generalOpts
}
func builtinOrigin(ctx Context, w facet, args... Value) (res Value) {
        var (
                scope = ctx.Scope()
                opts builtinOriginOpts
                elems []Value
        )
        for _, arg := range parseHeadArgsMerge(ctx, &opts, 0, args...) {
                var pos = arg.Position()
                if name := arg.Strval(ctx); name == "" {
                        elems = append(elems, MakeNil(pos))
                } else if def := scope.FindDef(name); def != nil {
                        elems = append(elems, MakeString(pos, def.origin.String()))
                } else {
                        elems = append(elems, MakeNil(pos))
                }
        }
        return MakeListOrScalar(ctx.Position(), elems)
}

type builtinDefinedOpts struct {
        generalOpts
}
func builtinDefined(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts builtinDefinedOpts
                elems []Value
        )
        for _, arg := range parseHeadArgsMerge(ctx, &opts, 0, args...) {
                var _, unresolved = arg.(unresolved)
                elems = append(elems, MakeBoolean(pos, !unresolved))
        }
        return MakeListOrScalar(pos, elems)
}

type builtinPushContextOpts struct {
        generalOpts
}
func builtinPushContext(ctx Context, w facet, args... Value) (res Value) {
        var (
                scope = ctx.Scope()
                dc = ctx.universe()
                opts builtinPushContextOpts
                m map[string]*def
        )
        for _, arg := range parseHeadArgsMerge(ctx, &opts, plain, args...) {
                var s = arg.Strval(ctx)
                if s == "" { continue }
                if m == nil { m = make(map[string]*def) }

                var t *def
                if o := scope.Lookup(s); o != nil { if d, ok := o.(*def); ok {
                        t = new(def) ; *t = *d
                }}
                m[s] = t
        }
        dc.globe.stack = append(dc.globe.stack, m)
        return
}

type builtinPopContextOpts struct {
        generalOpts
        rules []Value `r,rule,rules`
}
func builtinPopContext(ctx Context, w facet, args... Value) (res Value) {
        var opts builtinPopContextOpts
        for _, arg := range parseHeadArgsMerge(ctx, &opts, plain, args...) {
                warn(ctx, "unused argument: %T %v", arg, arg).debug(1)
                break
        }

        var rules []Value
        for _, r := range opts.rules {
                if v, y := r.(*Group); !y { rules = append(rules, v) } else {
                       rules = append(rules, v.Elems...)
                }
        }
        if proj := ctx.Project(); proj.entries != nil {
                for _, r := range rules {
                        delete(proj.entries, r.Strval(ctx))
                }
        }

        var scope = ctx.Scope()
        var dc = ctx.universe()
        var l = len(dc.globe.stack)
        if l == 0 { return }
        for s, d := range dc.globe.stack[l-1] {
                if d == nil { if s == "" { continue }
                        scope.mutex.Lock()
                        delete(scope.elems, s)
                        scope.mutex.Unlock()
                } else if o := scope.Lookup(d.name); o != nil { if t, ok := o.(*def); ok {
                        *t = *d
                }}
        }
        dc.globe.stack = dc.globe.stack[0:l-1]
        return
}

type builtinPositionOpts struct {
        filename bool `f,filename`
        filenameQuoted bool `q,quote-filename;qf,quoted-filename`
        line bool `l,line`
        column bool `c,column`
        addLine int `a,add;al,add-line`
        addColumn int `ac,add-column`
}
func builtinPosition(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts builtinPositionOpts
                vals []Value
        )
        args = parseOpts(ctx, &opts, plain, args...)

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
func builtinDate(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts = builtinDateOpts{ }
        )
        args = parseOpts(ctx, &opts, plain, args...)

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
func builtinDebug(ctx Context, w facet, args... Value) (res Value) {
        var opts builtinDebugOpts
        args = parseOpts(ctx, &opts, plain, args...)

        var s bytes.Buffer
        for i, a := range args {
                if i > 0 { fmt.Fprintf(&s, " ") }
                fmt.Fprintf(&s, "%s", a.Strval(ctx))
        }
        warnstack(ctx, opts.s, "%s", s.String()).debug(opts.n)
        return
}

func builtinError(ctx Context, w facet, args... Value) (res Value) {
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

func builtinWarning(ctx Context, w facet, args... Value) (res Value) {
        var s bytes.Buffer
        for i, a := range args {
                if i > 0 { fmt.Fprintf(&s, " ") }
                fmt.Fprintf(&s, "%s", a.Strval(ctx))
        }
        warn(ctx, "%s", s).debug(1)
        return
}

type builtinAssertOpts struct {
        generalOpts
}
func builtinAssert(ctx Context, w facet, args... Value) (res Value) {
        return assertion(ctx, generalOpts{}, w, args...)
}
func assertion(ctx Context, g generalOpts, w facet, args... Value) (res Value) {
        var opts = builtinAssertOpts{ g }
        if len(args) > 0 { args = append(
                parseOpts(ctx, &opts, expandZero, args[0]), args[1:]...) }

        var d = opts.debug ; if d < 1 { d = 1 + 3 }
        for _, a := range args {
                var ctx = at(ctx, a.Position())
                if v := a.expand(ctx, w); !v.True(ctx) {
                        prompt(ctx, "assertion: %T %v -> %T %v\n", a, a, v, v)
                        if opts.warn {
                                warnstack(ctx, d, "").debug(d)
                        } else {
                                errostack(ctx, d, "").debug(d)
                        }
                }
        }

        ctx.checkErrors(true)
        return
}

func builtinSureValue(ctx Context, w facet, args... Value) Value {
        for _, a := range args { if !a.True(ctx) {
                erro(of(ctx,a), "assertion failed: %v", a).debug(1)
        }}
        return MakeListOrScalar(ctx.Position(), args)
}

// $(defor $(x),$(y),$(z)) is identical to $(if $(defined $(x)),$(x),...)
func builtinDefOr(ctx Context, w facet, args... Value) (res Value) {
        for _, a := range mergex(ctx, plain, args...) {
                var _, unresolved = a.(unresolved)
                if unresolved { continue } else {
                        res = a
                        break
                }
        }
        return
}

func builtinOr(ctx Context, w facet, args... Value) (res Value) {
        for _, a := range args {
                if v := a.True(ctx); v {
                        res = a
                        break
                }
        }
        return
}

func builtinAnd(ctx Context, w facet, args... Value) (res Value) {
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
type builtinNotOpts struct {
        generalOpts
}
func builtinNot(ctx Context, w facet, args... Value) (res Value) {
        var (
                opts builtinNotOpts
                t bool
        )
        if len(args)>0 {
                if _, y := args[0].(unexpanded); y {
                        // NOTE: no opts from unexpanded value
                        // TODO: apply this rule to parseOpts
                } else if a := parseOpts(ctx, &opts, plain, args[0]); len(a)>0 {
                        args = append(a, args...)
                }
                if opts.debug>0 { for i, a := range args {
                        warn(ctx, "%v. %T %v", i, a, a) }}
                for _, a := range args { if t = a.True(ctx); t { break }}
                if n := opts.debug; n>0 { warnstack(ctx, 3, "").debug(n) }
        }
        res = MakeBoolean(ctx.Position(), !t)
        return
}

type builtinNotEqualOpts struct {
        generalOpts
}
func builtinNotEqual(ctx Context, w facet, args... Value) (res Value) {
        var opts builtinNotEqualOpts
        args = parseOpts(ctx, &opts, plain, args...)
        if n := len(args); n != 2 {
                erro(ctx, "wrong number of arguments, try: $(not-equal <value-list>,<value-list>)")
        } else if args[0].cmp(ctx, args[1]) != cmpEqual {
                res = MakeBoolean(ctx.Position(), true)
        }
        return
}

type builtinEqualOpts struct {
        generalOpts
}
func builtinEqual(ctx Context, w facet, args... Value) (res Value) {
        var opts builtinEqualOpts
        if len(args) > 0 {
                var a = parseOpts(ctx, &opts, /* NOTE: don't expand */0, args[0])
                if len(a) == 1 { args[0] = a[0] } else {
                        args[0] = MakeList(args[0].Position(), a...)
                }
        }
        if n := len(args); n != 2 {
                erro(ctx, "wrong number of arguments: %v", args)
                erro(ctx, "try: $(equal <value-list>,<value-list>)").debug(1)
                return
        }

        var (
                a = args[0].expand(ctx, plain)
                b = args[1].expand(ctx, plain)
        )
        if t := a.cmp(ctx, b); t == cmpEqual {
                res = MakeBoolean(ctx.Position(), true)
        } else if n := opts.debug; n>0 {
                warn(ctx, "equal: %v != %v ⇒ %v", a, b, t)
                if u, y := a.(unexpanded); y {
                        warn(of(ctx,a), "a: %T %v (unexpanded)", u.Value, a)
                } else {
                        warn(of(ctx,a), "a: %T %v", a, a)
                }
                if u, y := b.(unexpanded); y {
                        warn(of(ctx,a), "b: %T %v (unexpanded)", u.Value, b)
                } else {
                        warn(of(ctx,b), "b: %T %v", b, b)
                }
                warnstack(ctx, n, "").debug(n)
        } else if len(args)>2 {
                warnstack(of(ctx,args[2]), 1, "equal: extra args specified: %v", args[2]).debug(1)
        }
        return
}

type builtinGreaterOpts struct {
        generalOpts
}
func builtinGreater(ctx Context, w facet, args... Value) (res Value) {
        var opts builtinGreaterOpts
        args = parseOpts(ctx, &opts, plain, args...)
        if n := len(args); n != 2 {
                erro(ctx, "wrong number of arguments, try: $(greater <value-list>,<value-list>)")
        } else if cmp := args[0].cmp(ctx, args[1]); cmp == cmpGreater {
                res = MakeBoolean(ctx.Position(), true)
        }
        return
}

type builtinLessOpts struct {
        generalOpts
}
func builtinLess(ctx Context, w facet, args... Value) (res Value) {
        var opts builtinLessOpts
        args = parseOpts(ctx, &opts, plain, args...)
        if n := len(args); n != 2 {
                erro(ctx, "wrong number of arguments, try: $(less <value-list>,<value-list>)")
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
// $(match val1 val2 val3, a b c d...)
// $(match -rx=r1 -rx=r2 -rx=r3, a b c d...)
func builtinMatch(ctx Context, w facet, args... Value) (res Value) {
        var (
                patList, valList []Value
                opts builtinMatchOpts
        )
        if n := len(args); n < 2 {
                erro(ctx, "wrong arguments, try: $(match <regexp-list>,<value-list>,...)").debug(1)
                return
        }
        if opts.debug>0 {
                warn(ctx, "match: args=%v", args)
        }

        patList = parseOpts(ctx, &opts, plain, args[0])
        valList = mergex(ctx, plain, args[1:]...)

        var pos = ctx.Position()
        ForValList: for _, val := range valList {
                if isTrivial(val) { continue ForValList }

                var str = val.Strval(ctx)
                for _, rx := range opts.regexps {
                        var matched = rx.MatchString(str);
                        if opts.negated { matched = !matched }
                        if matched {
                                res = MakeBoolean(pos, true)
                                return
                        }
                }
                for _, pat := range patList {
                        var matched, s, _ = pat.match(ctx, str)
                        if !matched { matched = !opts.fullname && s != "" }
                        if opts.negated { matched = !matched }
                        if matched {
                                res = MakeBoolean(pos, true)
                                return
                        }
                }

                if opts.debug>0 {
                        warn(ctx, "match: %v", str)
                        warn(ctx, "match: %v %T", val, val).debug(1)
                }
        }
        return
}

// 1: $(case     (a 'xxx') (b 'yyy') (c 'zzz') (yes 'else'))
// 2: $(case val (a 'xxx') (b 'yyy') (c 'zzz') ('if none or nil'))
// 3: $(case val (a 'xxx') (b 'yyy') (c 'zzz') (- 'if none or nil'))
// 4: $(case val (a 'xxx') (b 'yyy') (c -) (- -))
func builtinCase(ctx Context, w facet, args... Value) (res Value) {
        var val Value
        if args = merge(args...); len(args) == 0 { return } else
        if _, ok := args[0].(*Group); !ok {
                val = args[0].expand(ctx, plain)
                args = args[1:]
        }

        var def []Value
        for _, arg := range args {
                if g, ok := arg.(*Group); ok && len(g.Elems)>0 {
                        if n := len(g.Elems); val != nil && isNone(val) && n == 1 {
                                res = g.Elems[0]
                                return
                        } else if n == 1 {
                                def = append(def, g.Elems[0])
                                continue
                        }

                        var collect bool
                        var v = g.Elems[0].expand(ctx, plain)
                        if val == nil && v.True(ctx) {
                                collect = true
                        } else if val != nil && isTrivial(val) {
                                if isTrivial(v) {
                                        collect = true
                                } else if f, ok := v.(*Flag); ok && isNil(f.name) {
                                        collect = true
                                }
                        } else if val != nil && val.cmp(ctx, v) == cmpEqual {
                                collect = true
                        } else if false && val != nil {
                                warn(ctx, "%v %v %v %v", val, v, g, val.cmp(ctx, v))
                        }
                        if !collect { continue }

                        var vals []Value
                        for _, v := range g.Elems[1:] {
                                if f, ok := v.(*Flag); ok && isNil(f.name) { continue }
                                vals = append(vals, v)
                        }
                        res = MakeListOrScalar(arg.Position(), vals)
                        return
                } else {
                        erro(of(ctx,arg), "unexpected case: %T %v", arg, arg).debug(1)
                        return
                }
        }

        if len(def) > 0 { res = MakeListOrScalar(ctx.Position(), def) }
        return
}

// $(if cond, true-value, else-value, ...)
func builtinIf(ctx Context, w facet, args... Value) (res Value) {
        if n := len(args); n > 1 {
                if false { if v := args[0]; v.String() == "&(.test.$_)" {
                        conds := mergex(ctx, plain, v)
                        t := conds[0].expand(ctx, plain)
                        s := v.Strval(ctx)
                        info(ctx, "%v -> %T %v -> %T %v -> %s", v, conds[0], conds[0], t, t, s)

                        info(ctx, "%v", autoGet(ctx,"_"))
                        infostack(ctx, 10, "").debug(64)
                }}
                if args[0].expand(ctx, plain).True(ctx) {
                        res = args[1]
                } else if n > 1 {
                        res = MakeListOrScalar(ctx.Position(), args[2:])
                }
        }
        return
}

func builtinIfEq(ctx Context, w facet, args... Value) (res Value) {
        if n := len(args); n > 2 {
                var (
                        a = args[0].expand(ctx, plain)
                        b = args[1].expand(ctx, expandDelegate  )
                        equal bool
                )
                if true {
                        equal = a.cmp(ctx, b) == cmpEqual
                } else {
                        equal = a.Strval(ctx) == b.Strval(ctx)
                }
                if equal {
                        res = args[2]
                } else if n > 3 {
                        res = MakeListOrScalar(ctx.Position(), args[3:])
                }
        }
        return
}

func builtinIfNE(ctx Context, w facet, args... Value) (res Value) {
        if n := len(args); n > 2 {
                var (
                        a = args[0].expand(ctx, plain)
                        b = args[1].expand(ctx, expandDelegate  )
                        equal bool
                )
                if true {
                        equal = a.cmp(ctx, b) == cmpEqual
                } else {
                        equal = a.Strval(ctx) == b.Strval(ctx)
                }
                if !equal {
                        res = args[2]
                } else if n > 3 {
                        res = MakeListOrScalar(ctx.Position(), args[3:])
                }
        }
        return
}

func builtinFor(ctx Context, w facet, args... Value) (res Value) {
        if n := len(args); n < 2 {
                erro(ctx, "not enough arguments, try: $(for <list>,<template>)")
                return
        }

        var (
                defs []*def
                vals []Value
                values = mergex(ctx, plain, args[0])
        )

        var scope = ctx.Globe().scope
        for i := 1; i <= maxNumVarVal; i += 1 {
                def := scope.Lookup(strconv.Itoa(i)).(*def)
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
                if values = mergex(ctx, plain, a); len(values) == 0 {
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

type builtinForEachOpts struct {
        // NOTE: disable all $(foreach) options to avoid messing with flag values.
        // generalOpts
        // empty bool `empty,allow-empty`
        debug int
        empty bool
}
func builtinForEach(ctx Context, w facet, args... Value) (res Value) {
        if n := len(args); n < 2 {
                errostack(ctx, 3, "insurficient arguments (%d); $(foreach <list>,<template>): %v", n, args).debug(32)
                return
        }

        var (
                opts builtinForEachOpts
                values = parseOpts(ctx, &opts, plain, args[0])
        )
        if len(values) == 0 {
                var d = opts.debug ; if d < 1 { d = 1 }
                errostack(ctx, 3, "insurficient arguments (%d); $(foreach <list>,<template>): %v", len(args), args).debug(d)
                return
        }

        var (
                resList []Value
                pos = ctx.Position()
                cc = autoContext{ Context:ctx, defs:make(autoDefMap) }
        )

        ctx = &cc

        for _, val := range values {
                if !opts.empty {
                        switch t := val.(type) {
                        case *Nil, *None, *delegate, *closure:
                                if opts.debug>0 { warn(ctx, "empty: %T %v", val, val).debug(1) }
                                continue
                        case *String:
                                if t.string == "" {
                                        if opts.debug>0 { warn(ctx, "empty: %T %v", val, val).debug(1) }
                                        continue
                                }
                        default:
                                if s := val.Strval(ctx); s == "" {
                                        if true || opts.debug>0 { warn(ctx, "empty: %T %v", val, val).debug(1) }
                                        continue
                                }
                        }
                }

                cc.autoSet("_", val)

                var list []Value
                var w = plain|expandPlaceholders|expandPairVal
                for _, a := range args[1:] {
                        if v := a.expand(ctx, w); isTrivial(v) {
                                // ignore
                        } else if s, ok := v.(*String); ok && s.string == "" {
                                // ignore
                        } else {
                                list = append(list, v)
                        }
                }

                if opts.debug>0 { warnstack(ctx, 3,
                        "foreach: %v -> %v ; %T %v -> %v -> %v",
                        args[0], values, val, val, args[1:], list).debug(2*opts.debug) }

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

type builtinCountOpts struct {
        generalOpts
        vals []Value `v,val,value`
}
func builtinCount(ctx Context, w facet, args... Value) (res Value) {
        var (
                opts builtinCountOpts
                num int64
        )
        args = parseOpts(ctx, &opts, plain, args...)

        var x = len(opts.vals)
        for i, a := range args {
                if (opts.vals == nil && a.True(ctx)) || (x>0 &&
                        cmpEqual == opts.vals[i % x].cmp(ctx, a)) {
                        num += 1
                }
        }
        res = MakeInt(ctx.Position(), num)
        return
}

type builtinEnvOpts struct {
        generalOpts
}
func builtinEnv(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                vals []Value
        )
        for _, a := range args {
                if val := a.expand(ctx, expandDelegate); isTrivial(val) {
                        continue
                } else if s := strings.TrimSpace(val.Strval(ctx)); s != "" {
                        vals = append(vals, MakeString(pos, os.Getenv(s)))
                }
        }
        return MakeListOrScalar(pos, vals)
}

type builtinAutoOpts struct {
        generalOpts
}
func builtinAuto(ctx Context, p *delegate, w facet, args... Value) (res Value) {
        if len(args) == 0 { return }

        var opts builtinAutoOpts
        var scope = ctx.Scope()
        var a = parseOpts(ctx, &opts, plain, args[0])
        if len(a) > 0 {
                var c = autoContext{ Context:ctx, defs:make(autoDefMap) }
                for _, t := range a {
                        if p, y := t.(*Pair); y {
                                if s := p.Key.Strval(ctx); s != "" {
                                        c.defs[s] = &def{
                                                origin: DefAuto, value: p.Value,
                                                knownobject: knownobject{
                                                        objbase{valbase{p.Position()}, scope, scope.project},
                                                        s,
                                                },
                                        }
                                }
                        }
                }
                for i, v := range args {
                        // TODO: expand with expandAuto, instead of w
                        args[i] = v.expand(&c, w)
                }
        }

        res = MakeListOrScalar(ctx.Position(), args[1:])
        return
}

// $(value <name1>,<name2>...)  -- this is specially useful when <name> is a closure.
type builtinValueOpts struct {
        generalOpts
        // def  []string `def,var`
        closure bool `c,clo,closure`
        unexp bool `ux,unexpand,unexpanded`
        undef bool `u,un,undef`
}
func builtinValue(ctx Context, w facet, args... Value) (res Value) {
        var (
                opts builtinValueOpts
                vals []Value
                closure bool
        )
        if args = parseOpts(ctx, &opts, plain, args...); opts.undef {
                vals = append(vals, &undef{&None{valbase{ctx.Position()}}})
        }
        closure = opts.closure
        for _, a := range args {
                var (
                        closure = closure || a.expandible(ctx, expandClosure)
                        name string
                        val Value
                )
                if name = a.Strval(ctx); name == "" {
                        // TODO: name is empty
                } else if closure {
                        if def := closureGet(ctx, name); def != nil {
                                val = def.value // NOTE: donot do 'def.Call(ctx)'
                        }
                } else if scope := ctx.Scope(); scope != nil {
                        if def := scope.FindDef(name); def != nil {
                                val = def.value // NOTE: donot do 'def.Call(ctx)'
                        }
                }
                if !closure && val == nil { val = autoGet(ctx,name) }
                if opts.debug>0 { warnstack(ctx, 3, "value: %v ; %v -> %v -> %v (closure=%v)",
                        args, a, name, val, closure).debug(2*opts.debug) }
                if val != nil {
                        if opts.unexp { val = unexpanded{val} }
                } else if closure {
                        val = MakeClosure(ctx.Position(), token.LPAREN, unresolved{a, ctx.Project()})
                } else if false {
                        val = MakeNone(a.Position())
                } else {
                        val = MakeNil(a.Position())
                }
                vals = append(vals, val)
        }
        if len(vals) > 0 {
                res = MakeListOrScalar(ctx.Position(), vals)
        }
        return
}

// $(call <name>,<args>...)  -- this is specially useful when <name> is a closure.
// NOTE: it's working differently from $(value <name>,<args>...), which is like calling
// without arguments, and the automatics $1, $2, ... will still be available (delegated)
// to the callee (<name>).
type builtinCallOpts struct {
        generalOpts
        closure bool `c,closure`
}
func builtinCall(ctx Context, p *delegate, w facet, args ...Value) (res Value) {
        var opts builtinCallOpts
        if l, ok := args[0].(*List); ok {
                l.Elems = parseOpts(ctx, &opts, /* don't expand */0, l.Elems...)
                if len(l.Elems) == 0 {
                        erro(ctx, "no callee name: %v", args[0]).debug(1)
                        return
                }
        } else if true {
                var a = parseOpts(ctx, &opts, /* don't expand */0, args[0])
                if len(a) != 1 {
                        erro(ctx, "invalid callee name: %v", a).debug(1)
                        return
                }
                args[0] = a[0] // overwrite the callee name
        }

        if len(args) == 0 {
                erro(ctx, "no name specified: %v", args[0]).debug(1)
                return
        }

        var (
                o Caller
                name string
                nameVal = args[0]
                closure bool = opts.closure ||
                        nameVal.expandible(ctx, expandClosure)
        )
        if name = nameVal.Strval(ctx); closure {
                for _, scope := range ctx.closureScopes() {
                        if def := scope.FindDef(name); def != nil && !isTrivial(def.value) {
                                o = def
                                break
                        }
                }
        } else if def := ctx.Scope().FindDef(name); def != nil {
                o = def
        }

        args = args[1:]

        if d, y := o.(*def); true && y {
                res, _ = p.call(ctx, w, d, args...)
        } else {
                res = o.Call(ctx, args...)
        }

        if res != nil {
                return
        } else if false {
                return MakeNone(nameVal.Position())
        } else {
                return MakeNil(nameVal.Position())
        }
}

type builtinClosureOpts struct {
        required bool `required,require-def,require-defs`
}
func builtinClosure(ctx Context, w facet, args... Value) (res Value) {
        var (
                opts builtinClosureOpts
                vals, names []Value
        )
        if len(args) < 1 {
                erro(ctx, "insufficient args: %v", args).debug(1)
                return
        }

        names = parseOpts(ctx, &opts, plain, args[0])
        if len(names) < 1 {
                erro(ctx, "no names: %v", args[0]).debug(1)
                return
        }

        for _, nameVal := range names {
                var ( def *def; name string )
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
                                erro(of(ctx,nameVal), "no def '%v' (%v)", name, nameVal).debug(1)
                        }
                } else {
                        vals = append(vals, def.Call(ctx, args[1:]...))
                }
        }
        return MakeListOrScalar(ctx.Position(), vals)
}

type builtinDefListOpts struct {
        rxs []*regexp.Regexp `r,re,rx,reg,regex,regexp`
        not   *regexp.Regexp `nr,neg,not,ex,except,exclude`
        n int `n,num,g`
        rn int `rn`
}
func builtinDefList(ctx Context, w facet, args... Value) (res Value) {
        var (
                opts builtinDefListOpts
                strs []string
                vals []Value
        )
        args = parseOpts(ctx, &opts, plain, args...)
        ForDefs: for name, _ := range ctx.Project().scope.elems {
                if len(opts.rxs) == 0 {
                        strs = append(strs, name)
                        if opts.n>0 && len(strs) == opts.n {
                                break
                        } else {
                                continue
                        }
                }
                if opts.not != nil && opts.not.MatchString(name) {
                        continue
                }
                for _, rx := range opts.rxs {
                        var sm = rx.FindStringSubmatch(name)
                        if len(sm)>0 && opts.rn<len(sm) {
                                strs = append(strs, sm[opts.rn])
                                if opts.n>0 && len(strs) == opts.n {
                                        break ForDefs
                                } else {
                                        continue ForDefs
                                }
                        }
                }
        }
        for _, str := range strs {
                vals = append(vals, MakeString(ctx.Position(), str))
        }
        return MakeListOrScalar(ctx.Position(), vals)
}

func builtinList(ctx Context, w facet, args... Value) (res Value) {
        res = MakeListOrScalar(ctx.Position(), args)
        return
}

func builtinShell(ctx Context, w facet, args... Value) (res Value) {
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
                        if !strings.HasPrefix(s, ":") { s = ":\n" + s }
                        prompt(ctx, "%s%s\n", a.Strval(ctx), s)
                        errostack(ctx, 3, "%s", err).debug(10)
                        if true { fail(ctx.Position(), "%v", err) }
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
func builtinWhich(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts builtinWhichOpts
                vals []Value
        )
        for _, a := range parseOpts(ctx, &opts, plain, args...) {
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
func builtinServeHttp(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts = builtinServeHttpOpts{ port:80 }
        )

        args = parseOpts(ctx, &opts, plain, args...)

        var server = &http.Server{}
        server.Addr = fmt.Sprintf("%s:%d", opts.host, opts.port)
        fmt.Fprintf(stderr, "%s: serving http at %v\n", pos, server.Addr)
        
        http.HandleFunc("/quit", func(w http.ResponseWriter, r *http.Request) {
                io.WriteString(w, "<font color=red>Server will close in 1sec ...</font>")
                go func() {
                        time.Sleep(1 * time.Second)
                        server.Shutdown(gc.Background())
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

func builtinServeHttps(ctx Context, w facet, args... Value) (res Value) {
        erro(ctx, "'serve-https' is unimplemented yet").debug(1)
        return
}

func builtinPrint(ctx Context, w facet, args... Value) (res Value) {
        var (
                x = len(args)
                sb bytes.Buffer
        )
        for i, a := range mergex(ctx, w, args...) {
                if a == nil { continue } else
                if 0 < i && i < x { fmt.Fprintf(&sb, " ") }
                fmt.Fprintf(&sb, "%s", EscapedString(ctx, a))
        }
        prompt(ctx, sb.String())
        return
}

func builtinPrintl(ctx Context, w facet, args... Value) (res Value) {
        var (
                x = len(args)
                sb bytes.Buffer
        )
        for i, a := range mergex(ctx, w, args...) {
                if 0 < i && i < x { fmt.Fprintf(&sb, " ") }
                var s = EscapedString(ctx, a)
                fmt.Fprintf(&sb, "%s", s)
                if i == x && !strings.HasSuffix(s, "\n") {
                        fmt.Fprintf(&sb, "\n")
                }
        }
        prompt(ctx, sb.String())
        return
}

func builtinPrintln(ctx Context, w facet, args... Value) (res Value) {
        var (
                x = len(args)
                sb bytes.Buffer
        )
        for i, a := range mergex(ctx, w, args...) {
                if a == nil { continue } else
                if 0 < i && i < x { fmt.Fprintf(&sb, " ") }
                fmt.Fprintf(&sb, "%s", EscapedString(ctx, a))
        }
        fmt.Fprintf(&sb, "\n")
        prompt(ctx, sb.String())
        return
}

type builtinAppendOpts struct {
        generalOpts
        auto bool `a,auto`
        closure bool `c,closure`
        // string bool `s,str,string`
}
func builtinAppend(ctx Context, w facet, args... Value) (result Value) {
        return __append(ctx, generalOpts{}, w, args...)
}
func __append(ctx Context, g generalOpts, w facet, args... Value) (result Value) {
        if len(args) < 2 {
                erro(ctx, "insufficient number of arguments: %v", args).debug(1)
                return
        }

        var opts = builtinAppendOpts{ generalOpts:g }
        var names, list []Value
        if names = parseOpts(ctx, &opts, plain, args[0]); len(names) == 0 {
                warn(ctx, "append to nowhere: %T %v", args[0], args[0]).debug(1)
                return
        }
        if list = mergex(ctx, plain, args[1:]...); len(list) == 0 {
                warn(ctx, "append no values: %v", args[1:]).debug(1)
                return
        }

        for _, a := range names {
                if name := a.Strval(ctx); name == "" {
                        erro(of(ctx,a), "name '%v' is empty", a).debug(1)
                } else if opts.closure {
                        if def := closureGet(ctx, name); def != nil {
                                def.append(ctx, list...)
                        } else {
                                erro(ctx, "'%s' (%v) is undefined (%T)", name, a, ctx).debug(1)
                        }
                } else if opts.auto {
                        if def := ctx.autoGet(name); def != nil {
                                def.append(ctx, list...)
                        } else {
                                erro(ctx, "'%s' (%v) is undefined (%T)", name, a, ctx).debug(1)
                        }
                } else if o := resolveObject(ctx, name); o != nil {
                        if d, y := o.(*def); y && d != nil { d.append(ctx, list...) } else {
                                erro(ctx, "'%s' (%v) is undefined (%T)", name, a, ctx).debug(1)
                        }
                } else {
                        erro(ctx, "%s", ctx).debug(1)
                }
        }
        return
}

type builtinMathOpts struct {
        generalOpts
        int bool `i,int,integer`
}
func builtinPlus(ctx Context, w facet, args... Value) (result Value) {
        var opts builtinMathOpts
        args = parseOpts(ctx, &opts, plain, args...)
        if opts.int {
                var num int64
                for n, a := range args {
                        if i, e := a.Integer(ctx); e == nil {
                                if n == 0 { num = i } else { num += i }
                        } else {
                                erro(ctx, "%v: %v", a, e).debug(1)
                        }
                }
                return MakeInt(ctx.Position(), num)
        } else {
                var num float64
                for n, a := range args {
                        if f, e := a.Float(ctx); e == nil {
                                if n == 0 { num = f } else { num += f }
                        } else {
                                erro(ctx, "%v: %v", a, e).debug(1)
                        }
                }
                return MakeFloat(ctx.Position(), num)
        }
}
func builtinMinus(ctx Context, w facet, args... Value) (result Value) {
        var opts builtinMathOpts
        args = parseOpts(ctx, &opts, plain, args...)
        if opts.int {
                var num int64
                for n, a := range args {
                        if i, e := a.Integer(ctx); e == nil {
                                if n == 0 { num = i } else { num -= i }
                        } else {
                                erro(ctx, "%v: %v", a, e).debug(1)
                        }
                }
                return MakeInt(ctx.Position(), num)
        } else {
                var num float64
                for n, a := range args {
                        if f, e := a.Float(ctx); e == nil {
                                if n == 0 { num = f } else { num -= f }
                        } else {
                                erro(ctx, "%v: %v", a, e).debug(1)
                        }
                }
                return MakeFloat(ctx.Position(), num)
        }
}
func builtinMultiply(ctx Context, w facet, args... Value) (result Value) {
        var opts builtinMathOpts
        args = parseOpts(ctx, &opts, plain, args...)
        if opts.int {
                var num int64
                for n, a := range args {
                        if i, e := a.Integer(ctx); e == nil {
                                if n == 0 { num = i } else { num *= i }
                        } else {
                                erro(ctx, "%v: %v", a, e).debug(1)
                        }
                }
                return MakeInt(ctx.Position(), num)
        } else {
                var num float64
                for n, a := range args {
                        if f, e := a.Float(ctx); e == nil {
                                if n == 0 { num = f } else { num *= f }
                        } else {
                                erro(ctx, "%v: %v", a, e).debug(1)
                        }
                }
                return MakeFloat(ctx.Position(), num)
        }
}
func builtinDivide(ctx Context, w facet, args... Value) (result Value) {
        var opts builtinMathOpts
        args = parseOpts(ctx, &opts, plain, args...)
        if opts.int {
                var num int64
                for n, a := range args {
                        if i, e := a.Integer(ctx); e == nil {
                                if n == 0 { num = i } else { num /= i } // FIXME: NaN
                        } else {
                                erro(ctx, "%v: %v", a, e).debug(1)
                        }
                }
                return MakeInt(ctx.Position(), num)
        } else {
                var num float64
                for n, a := range args {
                        if f, e := a.Float(ctx); e == nil {
                                if n == 0 { num = f } else { num /= f } // FIXME: NaN
                        } else {
                                erro(ctx, "%v: %v", a, e).debug(1)
                        }
                }
                return MakeFloat(ctx.Position(), num)
        }
}

type builtinUniqueOpts struct {
        generalOpts
	reverse bool `r,rev,reverse`
	keepAuto bool `a,auto,keepauto,keep-auto`
        unexpand bool `un,ue,unexpand,ne,noexpand,no-expand`
        plain bool `pl,pla,plain,pv,plainvalue,plain-value`
}
func builtinUnique(ctx Context, w facet, args... Value) (res Value) {
        var opts builtinUniqueOpts
        if len(args) > 0 {
                args = append(parseOpts(ctx, &opts, 0, merge(args[0])...),
                        args[1:]...)
        }
        if opts.unexpand {
                args = merge(args...)
        } else if opts.plain {
                var x = plain
                if opts.keepAuto { x &= ^expandAuto }
                args = mergex(ctx, x, args...)
        } else {
                var x = expandDelegate | expandPathStr | expandPairVal
                if opts.keepAuto { x &= ^expandAuto }
                args = mergex(ctx, x, args...)
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

func builtinJoin(ctx Context, w facet, args... Value) (res Value) {
        if l := len(args); l > 0 {
                var (
                        fields []string
                        vals []Value
                        sep string
                )
                if l < 2 {
                        vals = mergex(ctx, plain, args...)
                } else {
                        vals = mergex(ctx, plain, args[:l-1]...)
                        sep = args[l-1].Strval(ctx)
                }
                for _, a := range vals {
                        if v := a.Strval(ctx); v != "" { fields = append(fields, v) }
                }
                res = MakeString(ctx.Position(), strings.Join(fields, sep))
        }
        return
}

func builtinQuote(ctx Context, w facet, args... Value) (res Value) {
        args = mergex(ctx, plain, args...)
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

func builtinQuoteJoin(ctx Context, w facet, args... Value) (res Value) {
        var sep string
        args = mergex(ctx, plain, args...)

        if l := len(args); l > 1 {
                sep = args[l-1].Strval(ctx)
                args = args[:l-1]
        }
        if l := len(args); l > 0 {
                var fields []string
                for _, a := range args[1:] {
                        if v := a.Strval(ctx); v != "" { fields = append(fields, v) }
                }
                res = MakeString(ctx.Position(), strconv.Quote(strings.Join(fields, sep)))
        } else {
                res = MakeNone(ctx.Position())
        }
        return
}

// $(split-string .,1.2.3)
func builtinSplitString(ctx Context, w facet, args... Value) (res Value) {
        args = mergex(ctx, plain, args...)
        if l := len(args); l > 0 {
                var (
                        fields []Value
                        sep = args[0].Strval(ctx)
                )
                for _, a := range args[1:] {
                        for _, s := range strings.Split(a.Strval(ctx), sep) {
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
func builtinSplitQuote(ctx Context, w facet, args... Value) (res Value) {
        if res = builtinSplitString(ctx, w, args...); !isNil(res) {
                quotestrings(res)
        }
        return
}

// TODO: deprecate this and add -quote to builtinSplitString
func builtinSplitQuoteJoin(ctx Context, w facet, args... Value) (res Value) {
        var sep string
        if l := len(args); l > 1 {
                sep = args[l-1].Strval(ctx)
                args = args[:l-1]
        }

        var err error
        if res = builtinSplitQuote(ctx, w, args...); !isNil(res) {
                if res, err = joinstrings(ctx, res, sep); err != nil {
                        erro(ctx, "%v", err).debug(1)
                }
        } else {
                erro(ctx, "%v", err).debug(1)
        }
        return
}

func builtinSplitJoinQuote(ctx Context, w facet, args... Value) (res Value) {
        var sep string
        if l := len(args); l > 1 {
                sep = args[l-1].Strval(ctx)
                args = args[:l-1]
        }

        var (
                v Value
                err error
        )
        if v = builtinSplitString(ctx, w, args...); !isNil(v) {
                if v, err = joinstrings(ctx, v, sep); err == nil {
                        res = MakeString(ctx.Position(), strconv.Quote(v.Strval(ctx)))
                }
        }
        if err != nil { erro(ctx, "%v", err).debug(1) }
        return
}

func builtinField(ctx Context, w facet, args... Value) (res Value) {
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

func builtinFields(ctx Context, w facet, args... Value) (res Value) {
        // TODO: ...
        return
}

func builtinUsee(ctx Context, w facet, args... Value) (result Value) {
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
                if v, err = proj.use.Get(ctx, arg.Strval(ctx)); err != nil {
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

type builtinUsesOpts struct {
       generalOpts
}
func builtinUses(ctx Context, w facet, args... Value) (result Value) {
        var proj = ctx.Project() //current()
        if proj == nil {
                erro(ctx, "unknown current context").debug(1)
                return
        }

        var found bool
        var opts builtinUsesOpts
        args = parseOpts(ctx, &opts, plain, args...)

ForArgs:
        for _, arg := range args {
                var s = arg.Strval(ctx)
                for _, u := range proj.use.list {
                        if found = u.project.name == s; found {
                                break ForArgs
                        }
                }
        }
        if found {
                result = MakeBoolean(ctx.Position(), found)
        }
        return
}

func builtinPath(ctx Context, w facet, args... Value) (result Value) {
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

type builtinBareOpts struct {
        generalOpts
        name   bool `n,name,file-name,non-full`
}
func builtinBare(ctx Context, w facet, args... Value) (result Value) {
        var opts builtinBareOpts
        var vals []Value
        for _, a := range parseHeadArgsMerge(ctx, &opts, plain, args...) {
                var val Value
                switch t := a.(type) {
                case *String, *Compound:
                        val = MakeBareword(a.Position(), a.Strval(ctx));
                case *File:
                        val = MakeBareword(a.Position(), t.name);
                case fullfile:
                        if opts.name {
                                val = MakeBareword(a.Position(), t.name);
                        } else {
                                val = MakeBareword(a.Position(), t.Strval(ctx));
                        }
                default: val = a
                }
                vals = append(vals, val)
        }
        result = MakeListOrScalar(ctx.Position(), vals)
        return
}

func builtinBareword(ctx Context, w facet, args... Value) (result Value) {
        var vals []Value
        for _, a := range mergex(ctx, plain, args...) {
                var val Value
                switch a.(type) {
                case *Bareword: val = a
                default: val = MakeBareword(a.Position(), a.Strval(ctx));
                }
                vals = append(vals, val)
        }
        result = MakeListOrScalar(ctx.Position(), vals)
        return
}

type builtinStrOpts struct {
        generalOpts
        expand bool `x,e,ex,exp,expand`
        merge  bool `m,merge` // TODO: implement this merge opt
        name   bool `n,name,file-name,non-full`
        join []string `j,join`
        clo  []string `clo,closure`
        def  []string `def,var`
}
func builtinStr(ctx Context, strval bool, w facet, args... Value) (result Value) {
        var opts builtinStrOpts
        args = parseHeadArgsMerge(ctx, &opts, 0, args...)
        if len(args)+len(opts.clo)+len(opts.def) > 0 {
                var defs []*def
                for _, name := range opts.clo {
                        if o := closureResolveObject(ctx, name); o == nil { } else
                        if d, y := o.(*def); y && d != nil { defs = append(defs, d) }
                }
                for _, name := range opts.def {
                        if _, o := ctx.Scope().Find(name); o == nil { } else
                        if d, y := o.(*def); y && d != nil { defs = append(defs, d) }
                }

                var strs []string
                for _, d := range defs {
                        var t string
                        var v = d.value
                        if f, y := v.(fullfile); y && opts.name { v = f.File }
                        if opts.expand && v != nil { v = v.expand(ctx, w) }
                        if v == nil { t = "<nil>" } else
                        if strval { t = v.Strval(ctx) } else { t = v.String() }
                        strs = append(strs, t)
                        if opts.debug>0 { warn(ctx, "%T %v -> %v", d.value, d.value, t) }
                }
                for _, a := range args {
                        var t string
                        if f, y := a.(fullfile); y && opts.name { a = f.File }
                        if opts.expand { a = a.expand(ctx, w) }
                        if strval { t = a.Strval(ctx) } else { t = a.String() }
                        strs = append(strs, t)
                        if opts.debug>0 { warn(ctx, "%T %v -> %v", a, a, t) }
                }

                if len(opts.join)>0 {
                        var s bytes.Buffer
                        for i, t := range strs {
                                if i > 0 { s.WriteString(opts.join[i % len(opts.join)]) }
                                s.WriteString(t)
                                i += 1
                        }
                        result = MakeString(ctx.Position(), s.String())
                } else {
                        var pos = ctx.Position()
                        var vals []Value
                        for _, t := range strs {
                                vals = append(vals, MakeString(pos, t))
                        }
                        result = MakeListOrScalar(pos, vals)
                }
                if n := opts.debug; n>0 { warnstack(ctx, n, "%v", result).debug(n) }
        }
        return
}

func builtinStrval(ctx Context, w facet, args... Value) (result Value) {
        return builtinStr(ctx, true, w, args...)
}

func builtinString(ctx Context, w facet, args... Value) (result Value) {
        return builtinStr(ctx, false, w, args...)
}

type builtinFilterOpts struct {
        stem bool `s,stem;us,use-stem`
}
func filterValues(ctx Context, pats []Value, opts builtinFilterOpts, neg bool, values... Value) (result []Value, err error) {
        var filter = func(v Value) Value {
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
        for _, v := range mergex(ctx, plain, values...) {
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
                if pats = parseOpts(ctx, &opts, plain, args[0]); len(pats) > 0 {
                        i = 1 // good
                } else if pats = mergex(ctx, plain, args[1]); len(pats) == 0 {
                        erro(ctx, "no patterns: %v", args).debug(1)
                        return
                } else {
                        i = 2
                }

                if len(args) <= i {
                        erro(ctx, "out of index: %d %v", i, args).debug(1)
                        return
                }

                vals = mergex(ctx, plain, args[i:]...)
                if vals, err = filterValues(ctx, pats, opts, neg, vals...); err == nil {
                        res = MakeListOrScalar(pos, vals)
                }
        }
        if res == nil && err == nil { res = MakeNone(pos) }
        return
}

func builtinFilter(ctx Context, w facet, args... Value) (res Value) {
        // $(filter pattern…,text)
        res = filterValues1(ctx, false, args...)
        return
}

func builtinFilterOut(ctx Context, w facet, args... Value) (res Value) {
        // $(filter-out pattern…,text)
        res = filterValues1(ctx, true, args...)
        return
}

func builtinSubstring(ctx Context, w facet, args... Value) (res Value) {
        var pos = ctx.Position()
        args = mergex(ctx, plain, args...)

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
                                erro(of(ctx,args[0]), "%v", e).debug(1)
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
func builtinSubst(ctx Context, w facet, args... Value) (res Value) {
        var ( pos = ctx.Position(); list []Value )
        if nargs := len(args); nargs > 2 {
                var (
                        s1 = args[0].Strval(ctx)
                        s2 = args[1].Strval(ctx)
                )
                for _, arg := range mergex(ctx, expandDelegate, args[2:]...) {
                        var s = strings.Replace(arg.Strval(ctx), s1, s2, -1)
                        list = append(list, MakeString(pos, s))
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

// $(patsubst pattern,replacement,text)
// TODO: supports: $(var:pattern=replacement)
// TODO: supports: $(var:suffix=replacement)
type builtinPatsubstOpts struct {
        generalOpts
        files     bool `fi,file,fs,files`
        findFiles bool `find,find-file`
        fullFiles bool `ff,fullfile,fullfiles`
        cleanPath bool `c,clean,cleanpath`
        baseFiles bool `b,base,bases;bf,base-files,search-bases`
        useeFiles bool `u,used,using;uf,used-files,search-usees`
        noFileMap bool `nm,nomap,no-map,nofiles,no-files,no-filemap`
}
func builtinPatsubst(ctx Context, w facet, args... Value) (res Value) {
        var (
                opts builtinPatsubstOpts
                srcPats, dstPats, sources []Value
        )

        // TODO: support flags -name and -full for name-only and full-name-only matching
        if len(args) < 3 {
                erro(ctx, "not enough arguments").debug(1)
                return
        } else if srcPats = parseOpts(ctx, &opts, 0, args[0]); len(srcPats) == 0 {
                if len(args) < 4 {
                        erro(ctx, "not enough arguments").debug(1)
                        return
                }
                srcPats = mergex(ctx, plain, args[1])
                dstPats = mergex(ctx, plain, args[2])
                sources = mergex(ctx, plain, args[3:]...)
        } else {
                dstPats = mergex(ctx, plain, args[1])
                sources = mergex(ctx, plain, args[2:]...)
        }

        var (
                proj = ctx.Project()
                closured = closureProjects(ctx)
                filemaps []*filemap
                list []Value
        )
        if !opts.noFileMap {
                filemaps = proj.getFileMaps(ctx, opts.baseFiles, opts.useeFiles)
        }

ForSources:
        for _, src := range sources {
                var (
                        source interface{} = src
                        file *File
                        ok bool
                )
                if file, ok = toFile(src); ok {
                        source = file
                } else if opts.findFiles {
                        var s = src.Strval(ctx)
                        if file = proj.file(ctx, s); file != nil {
                                source = file
                        } else {
                                source = s
                        }
                } else if false && (opts.findFiles || opts.fullFiles) {
                        if file, ok = toFile(src); ok {
                                source = file
                        } else if file = proj.file(ctx, src.Strval(ctx)); file != nil {
                                if (opts.fullname || opts.fullFiles) && !filepath.IsAbs(file.name) {
                                        if !file.change("", "", file.fullname()) {
                                                warn(ctx, "changing fullname failed: %v", file).debug(1)
                                        }
                                }
                                source = file
                        }
                } else if opts.fullname {
                        var ( s string; ok bool )
                        if _, s, ok = asOptFullname(ctx, src, closured...); s == "" {
                                erro(of(ctx,src), "fullname '%v' is empty", src)
                                erro(ctx, "called from here", src).debug(1)
                                return
                        } else if !ok {
                                erro(of(ctx,src), "fullname '%v' failed", src)
                                erro(ctx, "called from here", src).debug(1)
                                return
                        } else {
                                source = s
                        }
                }

                var full = opts.fullFiles
                if !full { _, full = src.(fullfile) }

                var (
                        srcPat Value
                        stems []string
                )
                for _, srcPat = range srcPats {
                        if ok, _, stems = srcPat.match(ctx, source); ok { goto ForDstPats }
                }
                if !isTrivial(src) { list = append(list, src) }
                continue ForSources // just append src to the list

                // Compose the matched results with stem value.
        ForDstPats:
                for _, dst := range dstPats {
                        var nameVal, rest = dst.stencil(ctx, stems)
                        if isNil(nameVal) {
                                erro(ctx, "nil stencil: %T %v (stems=%v, rest=%v)",
                                        dst, dst, stems, rest).debug(1)
                                return
                        } else if opts.debug>0 {
                                warnstack(ctx, opts.debug, "patsubst: %v: %v -> %v -> %v %v -> %v %v",
                                        srcPat, src, source, stems, dst, nameVal, rest).debug(opts.debug)
                        }

                        var name string
                        if name = nameVal.Strval(ctx); name == "" {
                                continue ForDstPats
                        } else if opts.cleanPath {
                                name = filepath.Clean(name)
                        }

                        if t := file; t != nil {
                                var (
                                        match *FileMap
                                )
                                for _, m := range filemaps {
                                        var t = &FileMap{ filemap: m }
                                        if ok, _, _ := t.Match(ctx, name); ok {
                                                match = t
                                                break
                                        }
                                }

                                var file *File // dest file
                                if match != nil {
                                        if file = match.stat(ctx, t.dir, name); file != nil {
                                                assert(file.name == name, fmt.Sprintf("invalid file name: %s != %s (t.dir=%s)", file.name, name, t.dir))
                                        } else if file = match.stat(ctx, proj.absPath, name); file != nil {
                                                assert(file.name == name, fmt.Sprintf("invalid file name: %s != %s (proj.absPath=%s)", file.name, name, proj.absPath))
                                        }
                                }
                                if file == nil {
                                        file = stat(ctx, name, t.sub, t.dir, nil/* okay missing */)
                                }

                                if file.position = srcPat.Position(); full {
                                        list = append(list, fullfile{file})
                                } else {
                                        list = append(list, file)
                                }
                                continue ForDstPats
                        }

                        // Deal with source value types
                        // TODO: consider opts.files
                        var pos = dst.Position()
                        switch src.(type) {
                        case *File, fullfile:
                        case *String, *Compound:
                                list = append(list, MakeString(pos, name))
                                continue ForDstPats
                        case *Path:
                                list = append(list, MakePathStr(pos, name))
                                continue ForDstPats
                        case *Bareword, *Barecomp:
                                if strings.Contains(name, PathSep) {
                                        list = append(list, MakePathStr(pos, name))
                                } else {
                                        list = append(list, MakeBareword(pos, name))
                                }
                                continue ForDstPats
                        default:
                                if strings.Contains(name, PathSep) {
                                        list = append(list, MakePathStr(pos, name))
                                } else if true {
                                        list = append(list, MakeBareword(pos, name))
                                } else {
                                        list = append(list, MakeString(pos, name))
                                }
                                continue ForDstPats
                        }
                }
        }

        if opts.debug>0 && len(list) == 0 {
                warn(ctx, "src: %v", srcPats)
                warn(ctx, "dst: %v", dstPats)
                warn(ctx, "val: %v", sources)
                warn(ctx, "res: %v", list)
                warnstack(ctx, 3, "").debug(opts.debug)
        }

        res = MakeListOrScalar(ctx.Position(), list)
        return
}

func builtinStrip(ctx Context, w facet, args... Value) (res Value) {
        return builtinTrimSpace(ctx, w, args...)
}

func builtinTrimSpace(ctx Context, w facet, args... Value) (res Value) {
        return builtinTrim(ctx, w, append([]Value{MakeNone(ctx.Position())}, args...)...)
}

func builtinTitle(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                list []Value
        )
        for i, a := range mergex(ctx, plain, args...) {
                if i == 0 { pos = a.Position() }
                if s := a.Strval(ctx); s != "" {
                        list = append(list, MakeString(a.Position(), strings.Title(s)))
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

func builtinUpperCase(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                list []Value
        )
        for i, a := range mergex(ctx, plain, args...) {
                if i == 0 { pos = a.Position() }
                if s := a.Strval(ctx); s != "" {
                        list = append(list, MakeString(a.Position(), strings.ToUpper(s)))
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

func builtinLowerCase(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                list []Value
        )
        for _, a := range mergex(ctx, plain, args...) {
                if s := a.Strval(ctx); s != "" {
                        list = append(list, MakeString(a.Position(), strings.ToLower(s)))
                }
        }
        res = MakeListOrScalar(pos, list)
        return
}

func builtinTrim(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                cutset string
                list []Value
        )
        for i, a := range mergex(ctx, plain, args...) {
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

func builtinTrimLeft(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                cutset string
                list []Value
        )
        for i, a := range mergex(ctx, plain, args...) {
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

func builtinTrimRight(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                cutset string
                list []Value
        )
        for i, a := range mergex(ctx, plain, args...) {
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

type builtinTrimPrefixOpts struct {
        generalOpts
}
// $(trim-prefix foo%, fooxxx foo123)
// $(trim-prefix %/foo, xxx/foo/a/b/c)
// $(trim-prefix %%/foo, xxx/yyy/zzz/foo/a/b/c)
func builtinTrimPrefix(ctx Context, w facet, args... Value) (res Value) {
        var (
                opts builtinTrimPrefixOpts
                prefixs, values, list []Value
                err error
        )
        if len(args) == 0 { return } else {
                prefixs = parseOpts(ctx, &opts, plain, args[0])
        }
        if len(args) == 1 {
                if len(prefixs) > 1 { values = prefixs[1:] }
        } else {
                values = mergex(ctx, plain, args[1:]...)
        }

        if len(values) == 0 {
                return
        } else if len(prefixs) == 0 {
                res = MakeListOrScalar(ctx.Position(), values)
                return
        }

        if opts.verbose { warn(ctx, "prefix=%v values=%v", prefixs, values) }
        ForValues: for _, value := range values {
                var (
                        pos = value.Position()
                        p, s string
                )
                if s = value.Strval(ctx); s == "" { continue }
                ForPrefix: for _, prefix := range prefixs {
                        if p = prefix.Strval(ctx); p == "" { continue }

                        // FIXME: matched cutset is empty: %-xxx- and *-xxx-
                        var full, cutset, stems = prefix.match(ctx, value)
                        if opts.verbose /*|| (strings.Contains(s, "/.smart/modules/") && prefix.String() == "%%/.smart/modules/")*/ {
                                warn(ctx, "full=%v cutset=%v stems=%v", full, cutset, stems)
                                warn(ctx, "prefix = %T %v", prefix, prefix)
                                warn(ctx, "value  = %T %v", value, value)
                                warn(ctx, "trim   = %v", strings.TrimPrefix(s, cutset)).debug(1)
                        }

                        if full {
                                continue ForValues
                        } else if strings.HasPrefix(s, p) {
                                s = strings.TrimPrefix(s, p)
                                pos = prefix.Position()
                                break ForPrefix
                        } else if prefix.patterned(ctx) {
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
                }
                if s != "" { list = append(list, MakeString(pos, s)) }
        }
        if err == nil { res = MakeListOrScalar(ctx.Position(), list) }
        return
}

func builtinTrimSuffix(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                cutset, s string
                list []Value
        )
        for i, a := range mergex(ctx, plain, args...) {
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
func builtinTrimExt(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts builtinTrimExtOpts
                list []Value
                ext string
        )
        for i, a := range parseOpts(ctx, &opts, plain, args...) {
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
        generalOpts
}
func builtinExt(ctx Context, w facet, args... Value) (res Value) {
        var (
                opts builtinExtOpts
                list []Value
        )
        for _, a := range parseOpts(ctx, &opts, plain, args...) {
                list = append(list, MakeString(a.Position(), filepath.Ext(a.Strval(ctx))))
        }
        res = MakeListOrScalar(ctx.Position(), list)
        return
}

type builtinAddPrefixOpts struct {
        generalOpts
}
func builtinAddPrefix(ctx Context, w facet, args... Value) (res Value) {
        if len(args) < 1 {
                erro(ctx, "not enough args, try $(addprefix 'prefix', ...)").debug(1)
                return
        }

        var opts builtinAddPrefixOpts
        var prefixs = parseOpts(ctx, &opts, plain, args[0])
        if len(prefixs) != 1 {
                erro(ctx, "not enough args, try $(addprefix 'prefix', ...)").debug(1)
                return
        }

        var list []Value
        var vals = parseOpts(ctx, &opts, plain, args[1:]...)
        for _, prefix := range prefixs {
                if !prefix.True(ctx) { continue }
                var p, y = prefix.(*Pair)
                for _, val := range vals {
                        if /* false && !val.True(ctx) */isTrivial(val) { continue }
                        if y && !isTrivial(p.Value) {
                                val = MakeBarecomp(val.Position(), p.Value, val)
                        }
                        if val.expandible(ctx, expandDelegate|expandClosure) {
                                if y {
                                        val = paircomp{MakePair(p.Position(), p.Key, val)}
                                } else {
                                        val = precomp{prefix, val}
                                }
                        } else if y {
                                val = MakePair(p.Position(), p.Key, val)
                        } else {
                                val = MakeBarecomp(val.Position(), prefix, val)
                        }
                        list = append(list, val)
                }
        }

        res = MakeListOrScalar(ctx.Position(), list)
        return
}

type builtinAddSuffixOpts struct {
        generalOpts
        final bool `final`
}
func builtinAddSuffix(ctx Context, w facet, args... Value) (res Value) {
        if len(args) < 1 {
                erro(ctx, "not enough args, try $(addsuffix 'suffix', ...)").debug(1)
                return
        }

        var opts builtinAddSuffixOpts
        var suffixs = parseOpts(ctx, &opts, plain, args[0])
        if len(suffixs) != 1 {
                erro(ctx, "not enough args, try $(addsuffix 'suffix', ...)").debug(1)
                return
        }

        var list []Value
        var vals = parseOpts(ctx, &opts, plain, args[1:]...)
        for _, suffix := range suffixs {
                if !suffix.True(ctx) { continue }
                for _, val := range vals {
                        if /* false && !val.True(ctx) */isTrivial(val) {
                                continue
                        }
                        var pos = val.Position()
                        var p, y = val.(*Pair)
                        if y && !isTrivial(p.Value) {
                                val = MakeBarecomp(p.Key.Position(), val, p.Key)
                        }
                        if val.expandible(ctx, expandDelegate|expandClosure) {
                                if y {
                                        val = paircomp{MakePair(pos, val, p.Value)}
                                } else {
                                        val = rearcomp{val, suffix}
                                }
                        } else if y {
                                val = MakePair(pos, val, p.Value)
                        } else {
                                val = MakeBarecomp(pos, val, suffix)
                        }
                        list = append(list, val)
                }
        }

        res = MakeListOrScalar(ctx.Position(), list)
        return
}

type builtinPrintfOpts struct {
        generalOpts
}
func builtinPrintf(ctx Context, w facet, args... Value) (res Value) {
        var (
                pos = ctx.Position()
                opts builtinPrintfOpts
                vals []Value
                f string
        )
        if len(args) < 1 {
                erro(ctx, "not enough args, try $(printf 'format', ...)").debug(1)
                return
        } else if vals = parseOpts(ctx, &opts, plain, args[0]); len(vals) != 1 {
                erro(ctx, "not enough args, try $(printf 'format', ...)").debug(1)
                return
        } else {
                f = vals[0].Strval(ctx)
        }

        var i int
        var a []interface{}
        ForArgs: for n, v := range mergex(ctx, plain, args[1:]...) {
                if n == 0 { pos = v.Position() }
                ForFmt: for i < len(f) {
                        if f[i] != '%' { i += 1; continue }
                        for i += 1; i < len(f); i += 1 {
                                switch f[i] {
                                case '%': continue ForFmt
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
                                                if t, e := strconv.Atoi(v.Strval(ctx)) ; e == nil { a = append(a, t) } else {
                                                        erro(ctx, "%v: %v", v, e).debug(1)
                                                }
                                                continue ForArgs
                                        }
                                case 'v':
                                        a = append(a, v/* .Strval(ctx) */)
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

func builtinIndent(ctx Context, w facet, args... Value) (res Value) {
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

func builtinFindstring(ctx Context, w facet, args... Value) (res Value) {
        // TODO: $(findstring find,text)
        return
}

// $(contains a b c, v1 v2 …)
// $(contains a b c1 -or c2, v1 v2 …)          -- xx
// $(contains a b c1 -or c2 -or c3, v1 v2 …)   -- xx
// $(contains a b -or=(c1 c2 c3), v1 v2 …)     -- xx
type builtinContainsOpts struct {
        generalOpts
        match bool `m,mat,match,p,pat,pattern`
        string bool `s,str,string`
}
func builtinContains(ctx Context, w facet, args... Value) (res Value) {
        var (
                opts builtinContainsOpts
                vals []Value
                list []Value
        )
        if len(args) < 2 {
                erro(ctx, "unexpected number of arguments, try $(contains a b c1 -or c2, v1 v2 …)").debug(1)
                return
        }

        // const w = plain|expandPairVal
        vals, list = parseHeadArgsRequired(ctx, &opts, w|expandPairVal, args...)
        if len(vals) == 0 || len(list) == 0 { return }
        list = mergex(ctx, w, list...)

        var (
                y int
                s string
        )
        // NOTE: returns true if list contains all vals in it's presented order.
        ForVals: for _, val := range vals {
                if opts.string { s = val.Strval(ctx) }
                for _, elem := range list {
                        if opts.string {
                                if elem.Strval(ctx) == s {
                                        y += 1; continue ForVals
                                }
                        } else if opts.match {
                                if full, _, _ := val.match(ctx, elem); full {
                                        y += 1; continue ForVals
                                }
                        } else if elem.cmp(ctx, val) == cmpEqual {
                                y += 1; continue ForVals
                        }
                        if false && opts.debug>0 && !opts.string && !isNil(elem) &&
                                val.Strval(ctx) == elem.Strval(ctx) {
                                warn(of(ctx,val), "wrong: %T %v <-> %T %v", val, val, elem, elem)
                        }
                }
                if opts.debug>0 { warn(of(ctx,val), "found 0: %T %v", val, val) }
        }

        var b = (y == len(vals))
        if opts.debug>0 && !b {
                warn(ctx, "found %d/%d: %v", y, len(vals), list).debug(opts.debug)
        }
        res = MakeBoolean(ctx.Position(), b)
        return
}
func builtinContains_buggy(ctx Context, w facet, args... Value) (res Value) {
        var (
                opts builtinContainsOpts
                vals []Value
                list []Value
        )
        if len(args) < 2 {
                erro(ctx, "unexpected number of arguments, try $(contains a b c1 -or c2, v1 v2 …)").debug(1)
                return
        }

        vals = parseOpts(ctx, &opts, plain, args[0])
        list = mergex(ctx, plain, args[1:]...)

        var (
                n = 0
                x = len(vals)
                va []Value
        )
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
                                } else if t := a.cmp(ctx, v); t != cmpEqual {
                                        if opts.debug>0 {
                                                warn(ctx, "%v %v ; %v ; %v", val, v, a, t).debug(1)
                                        }
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

func builtinSort(ctx Context, w facet, args... Value) (res Value) {
        // TODO: $(sort list)
        return
}

func builtinWord(ctx Context, w facet, args... Value) (res Value) {
        // TODO: $(word n,text)
        return
}

func builtinWordList(ctx Context, w facet, args... Value) (res Value) {
        // TODO: $(wordlist s,e,text)
        return
}

func builtinWords(ctx Context, w facet, args... Value) (res Value) {
        // TODO: $(words n,text)
        return
}

func builtinFirstWord(ctx Context, w facet, args... Value) (res Value) {
        // TODO: $(firstword names...)
        return
}

func builtinLastWord(ctx Context, w facet, args... Value) (res Value) {
        // TODO: $(lastword names...)
        return
}

func builtinEncodeBase64(ctx Context, w facet, args... Value) (res Value) {
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

func builtinDecodeBase64(ctx Context, w facet, args... Value) (res Value) {
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
        case *def      : if !isTrivial(t.value) { return asFile(ctx, t.value   ) }
        case *List     : if len(t.Elems) == 1   { return asFile(ctx, t.Elems[0]) }
        case *RuleEntry:                          return asFile(ctx, t.target  )
        case *String, *Compound:
                // NOTE: escape 'string' and "compound" values from file parsing
                // NOTE: for optimization
        case *Bareword, *Barecomp, *Path:
                if len(projects) == 0 { projects = closureProjects(ctx) }
                for _, proj := range projects {
                        if f = proj.file(ctx, t.Strval(ctx)); f != nil { break }
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
                return
        } else if s = val.Strval(ctx); s == "" {
                // ...
        } else if filepath.IsAbs(s) {
                file = stat(ctx, s, "", "")
                ok = true
        }
        return
}

func asOptFullname2(ctx Context, val Value, projects ...*Project) (file *File, s string, ok bool) {
        file, s, ok = asOptFullname(ctx, val, projects...)
        if !ok && file == nil && s != "" && !filepath.IsAbs(s) {
                switch val.(type) {
                case *String, *Compound: return
                }
                for _, proj := range projects {
                        if file = proj.file(ctx, s); file != nil {
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
func builtinFullName(ctx Context, w facet, args... Value) (res Value) {
        var (
                closured = closureProjects(ctx)
                opts builtinFullNameOpts
                l []Value
                s string
                ok bool
        )
        for _, a := range parseOpts(ctx, &opts, plain, args...) {
                if opts.debug > 0 {
                        if f, ok := toFile(a); ok {
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
        for _, a := range parseOpts(ctx, &opts, plain, args...) {
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
func builtinBase (ctx Context, w facet, args... Value) Value { return basex(ctx, 1, args...) }
func builtinBase2(ctx Context, w facet, args... Value) Value { return basex(ctx, 2, args...) }
func builtinBase3(ctx Context, w facet, args... Value) Value { return basex(ctx, 3, args...) }
func builtinBase4(ctx Context, w facet, args... Value) Value { return basex(ctx, 4, args...) }
func builtinBase5(ctx Context, w facet, args... Value) Value { return basex(ctx, 5, args...) }
func builtinBase6(ctx Context, w facet, args... Value) Value { return basex(ctx, 6, args...) }
func builtinBase7(ctx Context, w facet, args... Value) Value { return basex(ctx, 7, args...) }
func builtinBase8(ctx Context, w facet, args... Value) Value { return basex(ctx, 8, args...) }
func builtinBase9(ctx Context, w facet, args... Value) Value { return basex(ctx, 9, args...) }

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
        for _, a := range parseOpts(ctx, &opts, plain, args...) {
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
        for _, a := range parseOpts(ctx, &opts, plain, args...) {
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

func builtinDir (ctx Context, w facet, args... Value) (res Value) { return dirx(ctx, 1, args...) }
func builtinDir2(ctx Context, w facet, args... Value) (res Value) { return dirx(ctx, 2, args...) }
func builtinDir3(ctx Context, w facet, args... Value) (res Value) { return dirx(ctx, 3, args...) }
func builtinDir4(ctx Context, w facet, args... Value) (res Value) { return dirx(ctx, 4, args...) }
func builtinDir5(ctx Context, w facet, args... Value) (res Value) { return dirx(ctx, 5, args...) }
func builtinDir6(ctx Context, w facet, args... Value) (res Value) { return dirx(ctx, 6, args...) }
func builtinDir7(ctx Context, w facet, args... Value) (res Value) { return dirx(ctx, 7, args...) }
func builtinDir8(ctx Context, w facet, args... Value) (res Value) { return dirx(ctx, 8, args...) }
func builtinDir9(ctx Context, w facet, args... Value) (res Value) { return dirx(ctx, 9, args...) }
func builtinDirs(ctx Context, w facet, args... Value) (res Value) {
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

func builtinUndir (ctx Context, w facet, args... Value) (res Value) { return undirx(ctx, 1, args...) }
func builtinUndir2(ctx Context, w facet, args... Value) (res Value) { return undirx(ctx, 2, args...) }
func builtinUndir3(ctx Context, w facet, args... Value) (res Value) { return undirx(ctx, 3, args...) }
func builtinUndir4(ctx Context, w facet, args... Value) (res Value) { return undirx(ctx, 4, args...) }
func builtinUndir5(ctx Context, w facet, args... Value) (res Value) { return undirx(ctx, 5, args...) }
func builtinUndir6(ctx Context, w facet, args... Value) (res Value) { return undirx(ctx, 6, args...) }
func builtinUndir7(ctx Context, w facet, args... Value) (res Value) { return undirx(ctx, 7, args...) }
func builtinUndir8(ctx Context, w facet, args... Value) (res Value) { return undirx(ctx, 8, args...) }
func builtinUndir9(ctx Context, w facet, args... Value) (res Value) { return undirx(ctx, 9, args...) }
func builtinUndirs(ctx Context, w facet, args... Value) (res Value) {
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

func builtinDirChop(ctx Context, w facet, args... Value) (res Value) {
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

func builtinRelativeDir(ctx Context, w facet, args... Value) (res Value) {
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

func builtinMkdir(ctx Context, w facet, args... Value) (res Value) {
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

func builtinMkdirAll(ctx Context, w facet, args... Value) (res Value) {
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

func builtinChdir(ctx Context, w facet, args... Value) (res Value) {
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
        generalOpts
}
func builtinRename(ctx Context, w facet, args... Value) (res Value) {
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
                                erro(of(ctx,t), "wrong size of group `%v'", t).debug(1)
                                break
                        }
                case *List: // rename oldname newname, old new, ...
                        if t.Len() == 2 {
                                oldname = t.Get(0).Strval(ctx)
                                newname = t.Get(1).Strval(ctx)
                        } else {
                                erro(of(ctx,t), "wrong size of list `%v'", t).debug(1)
                                break
                        }
                default: // rename newname oldname  newname oldname ...
                        if i+1 < nargs {
                                oldname = args[i+0].Strval(ctx)
                                newname = args[i+1].Strval(ctx)
                                i += 1
                        } else {
                                erro(of(ctx,t), "Wrong arguments `%v'", args).debug(1)
                                break
                        }
                }
                if err := os.Rename(oldname, newname); err != nil {
                        erro(ctx, "%v", err).debug(1)
                        break
                }
        }
        return
}

type builtinRemoveOpts struct {
        generalOpts
        skip string `save,skip`
        ignoreMissing bool `gm,ignoremissing,ignore-missing`
        warnNotFile bool `warn-not-file`
        all bool `a,all;r,recursive`
}
func builtinRemove(ctx Context, w facet, args... Value) (res Value) {
        var (
                closured = closureProjects(ctx)
                opts builtinRemoveOpts
                names []string
                str string
                ok bool
        )
        for _, a := range parseOpts(ctx, &opts, plain, args...) {
                var (
                        ctx = at(ctx, a.Position())
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
                                if opts.debug>0 { info(ctx, "remove %s", s).debug(opts.debug) }
                                if opts.all { err = os.RemoveAll(s) } else { err = os.Remove(s) }
                                if err == nil {
                                        if opts.verbose { prompt(ctx, "remove %s (%s)\n", s, typeof(a)) }
                                } else {
                                        erro(ctx, "remove failed: %v", err)
                                        return
                                }
                        }
                        continue
                } else if file, str, ok = asOptFullname2(ctx, a, closured...); !ok || str == "" {
                        if file != nil { ok = true } else
                        if opts.all { if opts.warnNotFile {
                                warn(ctx, "not a file: %v (%T)", a, a)
                                warn(ctx, "in %v", closured)
                                warnstack(ctx, 3, "").debug(32)
                        }} else {
                                erro(ctx, "not a file: %v (%T)", a, a)
                                erro(ctx, "in %v", closured)
                                errostack(ctx, 3, "").debug(32)
                                break
                        }
                } else if file != nil && file.exists() {
                        if false && strings.HasSuffix(str, "prov/bio.h") {
                                warn(ctx, "%T %v %s", a, a, str).debug(1)
                        }
                        if opts.skip != "" && strings.HasPrefix(str, opts.skip) {
                                prompt(ctx, "remove: skip %v\t-> %s\n", a, str)
                                continue
                        }
                        if opts.debug>0 { warn(ctx, "remove %s", str).debug(opts.debug) }
                        if opts.all { err = os.RemoveAll(str) } else { err = os.Remove(str) }
                        if err == nil {
                                if opts.verbose { prompt(ctx, "remove %s (%s)\n", str, typeof(a)) }
                        } else {
                                erro(ctx, "%v", err)
                                erro(ctx, "source: %v (%T)", a, a)
                                erro(ctx, "source: %v", str).debug(1)
                                return
                        }
                } else if opts.verbose && !opts.ignoreMissing {
                        if file != nil {
                                prompt(ctx, "remove: no such file %s (%s)\n", str, typeof(a))
                        } else {
                                prompt(ctx, "remove: no such %s (%s)\n", str, typeof(a))
                        }
                }
        }
        return
}

type builtinRemoveAllOpts struct {
        generalOpts
}
func builtinRemoveAll(ctx Context, w facet, args... Value) (res Value) {
        var (
                closured = closureProjects(ctx)
                opts builtinRemoveAllOpts
                names []string
                str string
                ok bool
        )
        for _, a := range parseOpts(ctx, &opts, plain, args...) {
                var ctx = at(ctx, a.Position())
                if a.patterned(ctx) {
                        var err error
                        if names, err = filepath.Glob(a.Strval(ctx)); err != nil {
                                erro(ctx, "%v", err).debug(1)
                                return
                        }
                        for _, s := range names {
                                if opts.verbose { info(of(ctx,a), "remove %s", s) }
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
                        if opts.debug>0 { info(ctx, "remove %s", str).debug(1) }
                        if err := os.RemoveAll(str); err != nil {
                                erro(ctx, "remove failed: %v", err).debug(1)
                                return
                        }
                }
        }
        return
}

func builtinTruncate(ctx Context, w facet, args... Value) (res Value) {
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
func builtinLink(ctx Context, w facet, args... Value) (res Value) {
        var opts builtinLinkOpts
        args = parseOpts(ctx, &opts, plain, args...)
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
        path     bool `p,path`
        force    bool `force;ow,overwrite`
        update   bool `u,update`
        relative bool `r,rel,relative;l`
}
func builtinSymlink(ctx Context, w facet, args... Value) (res Value) {
        var opts builtinSymlinkOpts
        args = parseOpts(ctx, &opts, plain, args...)
ForArgs:
        for i, na := 0, len(args); i < na; i += 1 {
                var (
                        opts = opts // make a copy
                        srcNameVal, dstNameVal Value
                        srcName   , dstName    string
                        srcDir    , dstDir     string
                        aa []Value
                )
                switch t := args[i].(type) {
                case *Pair: // symlink srcName=dstName srcName=>dstName...
                        srcNameVal, dstNameVal = t.Key, t.Value
                case *Group: // symlink (-u srcName dstName) (-v srcName dstName)...
                        if aa = parseOpts(ctx, &opts, plain, t.Elems...); len(aa) != 2 {
                                erro(of(ctx,t), "expects two values for group").debug(1)
                                return
                        } else {
                                srcNameVal, dstNameVal = aa[0], aa[1]
                        }
                case *List: // XXX: symlink old new, old new, ...
                        if aa = parseOpts(ctx, &opts, plain, t.Elems...); len(aa) != 2 {
                                erro(of(ctx,t), "expects two values for list").debug(1)
                                return
                        } else {
                                srcNameVal, dstNameVal = aa[0], aa[1]
                        }
                default:// Multiple pairs of names:
                        // symlink  new old, new old ...
                        // symlink  new old  new old ...
                        if i+1 < na {
                                srcNameVal = args[i+0]
                                dstNameVal = args[i+1]
                                i += 1
                        } else {
                                var a = autoGet(ctx,"@")
                                var l = autoGet(ctx,"<")
                                var r = autoGet(ctx,">")
                                prompt(ctx, "symlink: args=%v -> %v\n", args, t)
                                prompt(ctx, "symlink: %v, %v, %v\n", a, l, r)
                                errostack(of(ctx,t), 5, "expects pair of names (%T %v)", t, t).debug(6)
                                return
                        }
                }

                if srcDir, srcName = splitFileName(ctx, srcNameVal); srcName == "" {
                        prompt(ctx, "symlink: args=%v\n", args)
                        prompt(ctx, "symlink: src=%v\n", srcNameVal)
                        errostack(of(ctx,srcNameVal), 5, "empty src filename (%T)", srcNameVal).debug(6)
                        return
                }
                if dstDir, dstName = splitFileName(ctx, dstNameVal); dstName == "" {
                        prompt(ctx, "symlink: args=%v\n", args)
                        prompt(ctx, "symlink: dest=%v\n", dstNameVal)
                        errostack(of(ctx,dstNameVal), 6, "empty dest filename (%T)", dstNameVal).debug(12)
                        return
                }

                var src = filepath.Join(srcDir, srcName)
                var dst = filepath.Join(dstDir, dstName)
                if _, err := os.Stat(src); err != nil {
                        prompt(ctx, "symlink: %v: %v\n", srcName, err)
                        errostack(of(ctx,srcNameVal), 6, "%v does not exist", srcName).debug(8)
                        return
                }

                if !opts.relative {/* no rel required */} else
                if s, e := filepath.Rel(filepath.Dir(dst), src); e != nil {
                        prompt(ctx, "symlink: %s: rel(%s, %s)\n", dstName, dst, src)
                        errostack(of(ctx,dstNameVal), 8, "%v", e).debug(10)
                        return
                } else {
                        if false {
                                info(ctx, "%v %v\t%s", srcDir, srcName, src)
                                info(ctx, "%v %v\t%s", dstDir, dstName, dst)
                                info(ctx, "%v", s).debug(1)
                        }
                        src = s
                }

                if !opts.path {/* no mkdir */} else
                if dstDir == "" || dstDir == "." || dstDir == PathSep {
                        // no need to mkdir: . or /
                } else if err := os.MkdirAll(dstDir, os.FileMode(0755)); err != nil {
                        erro(of(ctx,dstNameVal), "%v", err).debug(1)
                        return
                }

                var rm bool
                if rm = opts.force; rm {
                        // overwrite...
                } else if s, e := os.Readlink(dst); e != nil {
                        if false {
                                prompt(ctx, "%v: readlink failed (%T)\n", dstName, e)
                                errostack(of(ctx,dstNameVal), 6, "%v", e).debug(8)
                        }
                } else if rm = s != src; !rm {
                        continue ForArgs
                }

                if rm { if e := os.Remove(dst); e != nil {
                        prompt(ctx, "%v: remove old symlink failed (%T)\n", dstName, e)
                        errostack(of(ctx,dstNameVal), 6, "%v", e).debug(8)
                        return
                }}
                if err := os.Symlink(src, dst); err != nil {
                        if opts.verbose { prompt(ctx, "… %s\n", err) }
                        break
                } else if opts.verbose {
                        var d = trimPromptString(dstName)
                        var s = filepath.Base(srcName)
                        prompt(ctx, "%s -> %s …… ok\n", d, s)
                }
        }
        return
}

type builtinStatOpts struct {
        generalOpts
        dir bool `di,dr,dir`
        file bool `fi,file`
        symbol bool `s,sym,symlink,symbol;l,link`
}
func builtinStat(ctx Context, w facet, args... Value) (res Value) {
        var (
                proj = ctx.Project()
                opts builtinStatOpts
                nams []Value
        )
        if proj == nil {
                erro(ctx, "unknown current context").debug(1)
                return
        }

        if nams = parseOpts(ctx, &opts, plain, args...); len(nams) == 0 {
                return
        }

        var (
                pos = ctx.Position()
                valF = MakeBoolean(pos, false)
                valT = MakeBoolean(pos, true)
                vals []Value
        )
        var check = func(file *File) {
                if file == nil || file.info == nil {
                        vals = append(vals, valF)
                } else if mode := file.info.Mode(); opts.dir && mode&os.ModeDir != 0 { // IsDir()
                        vals = append(vals, valT)//file
                } else if opts.symbol && mode&os.ModeSymlink != 0 {
                        vals = append(vals, valT)//file
                } else if opts.file && mode&os.ModeType != 0 { // IsRegular()
                        vals = append(vals, valT)//file
                } else {
                        vals = append(vals, valT)//file
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
                if file == nil { file = proj.file(ctx, s) }
                if file != nil { check(file) }
        }

        for _, a := range nams {
                switch t := a.(type) {
                case *File: check(t)
                case *Path: checkstat(a)
                default:    checkstat(a)
                }
        }

        res = MakeListOrScalar(pos, vals)
        return
}

type builtinFileOpts struct {
        generalOpts
        ignore bool `i,ig,ignore,ignore-missing`
        caller bool `c,cc,caller,callercontext,caller-context`
        report bool `r,report,reportmissing;rm,report-missing;e,error`
}
func builtinFile(ctx Context, w facet, args... Value) (res Value) {
        var (
                opts builtinFileOpts
                proj *Project
                list []Value
        )

        args = parseOpts(ctx, &opts, plain, args...)

        if opts.caller && false {
                // program -> closure -> traversal -> ...
                if false {
                        proj = ctx.closure().Project()
                } else {
                        proj = ctx.programContext().Project()
                }
        } else {
                proj = ctx.Project()
        }

        for _, a := range args {
                var ctx = at(ctx, a.Position())
                if file, y := toFile(a); y {
                        if list = append(list, file); !file.exists() { file.stat(ctx) }
                        if !file.exists() && opts.report {
                                info(ctx, "%v is no such file", a).debug(1)
                        }
                        continue
                } else if s := a.Strval(ctx); s == "" {
                        erro(ctx, `%v: %T "%v" is empty`, proj, a, a)
                        errostack(ctx, 3, "(%T): %v", ctx, proj).debug(6)
                        continue
                } else if file = proj.file(ctx, s); file != nil {
                        list = append(list, file)
                        if opts.report { info(ctx, "%v is no such file", a).debug(1) }
                        continue
                } else if filepath.IsAbs(s) {
                        if file = stat(ctx, s, "", ""); file != nil {
                                list = append(list, file)
                                if opts.report { info(ctx, "%v is no such file", a).debug(1) }
                                continue
                        }
                } else if file = stat(ctx, s, "", proj.absPath); file != nil {
                        list = append(list, file)
                        if opts.report { info(ctx, "%v is no such file", a).debug(1) }
                        continue
                }

                if opts.ignore {
                        if opts.verbose { info(ctx, "%v", a).debug(1) }
                } else {
                        erro(ctx, `%v: "%v" is not a file (%T)`, proj, a, a)
                        errostack(ctx, 3, "(%T): %v", ctx, proj).debug(16)
                }
        }

        res = MakeListOrScalar(ctx.Position(), list)
        return
}

type builtinGlobOpts struct {
        generalOpts
        dir bool `di,dir,directory`
        file bool `fi,file`
        symbol bool `s,sym,symlink,symbol,symbolic`
}
func builtinGlob(ctx Context, w facet, args... Value) (res Value) {
        var (
                opts builtinGlobOpts
                proj *Project
        )

        args = parseOpts(ctx, &opts, plain, args...)

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

func readDirNames(ctx Context, opts *wildcardOpts) (names []string) {
        var dir *os.File
        if fi, err := os.Stat(opts.dir); err != nil {
                if opts.errorMissing { erro(ctx, "%v", err).debug(1) }
                return
        } else if !fi.IsDir() {
                erro(ctx, "not dir: %v", opts.dir).debug(1)
                return
        } else if dir, err = os.Open(opts.dir); err != nil {
                erro(ctx, "not dir: %v", opts.dir).debug(1)
                return
        }

        // NOTE: see alsl filepath.Glob(...)
        var _names, err = dir.Readdirnames(-1); dir.Close()
        if err != nil {
                if opts.errorMissing { erro(ctx, "readdir: %v", err).debug(1) }
                return
        } else { names = _names }
        return
}

func wildcardPathPatsInDir3(ctx Context, opts *wildcardOpts, pats ...Value) (files []*File) {
        var dbg = false //strings.Contains(opts.dir, "/external/llvm-project/llvm/include")
        var names = readDirNames(ctx, opts)
        forNames: for _, name := range names {
                for _, x := range opts.exclude {
                        if full, _, _ := x.match(ctx, name); full { continue forNames }
                }
                for _, pat := range pats {
                        var ctx = at(ctx, pat.Position())
                        var p, ok = pat.(*Path)
                        if !ok || len(p.Elems) <= 1 {
                                var full, s, stems = pat.match(ctx, name)
                                if dbg { warn(ctx, "%T %v %v; %v %v %v; %v %s", pat, pat, name, full, s, stems, p.Elems, opts.dir) }
                                if full {
                                        files = append(files, stat(ctx, name, "", opts.dir))
                                        if opts.debug>0 {
                                                warn(ctx, "wildcard: %v -> %v -> %v",
                                                        name, pat, opts.dir).debug(opts.debug)
                                        }
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
                        if opts.debug>0 {
                                warn(ctx, "wildcard: %v -> %v -> %v",
                                        name, pat, subs).debug(opts.debug)
                        }
                }
        }
        return
}

func wildcardPathPatsInDir(ctx Context, opts *wildcardOpts, pats ...Value) (files []*File) {
        if files = wildcardPathPatsInDir3(ctx, opts, pats...); opts.filetype != "" {
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
        includeMissing bool `im,includemissing,include-missing,m,missing`
        ignoreMissing bool `gm,ignoremissing,ignore-missing`
        errorMissing bool `em,errormissing,e,err,error-missing,no-missing`
        baseFiles bool `b,base,bases;bf,base-files`
        useeFiles bool `u,used;u,using;uf,used-files`
        names bool `bare,n,name,names`
        strs bool `s,str,strs,string,strings`
        exclude []Value `x,ex,excl,exclude,except,no,not`
        filetype string `ft,filetype,file-type` // dir, file, etc.
        dir string `di,dir,directory`
}
func builtinWildcard(ctx Context, w facet, args... Value) (res Value) {
        var (
                opts wildcardOpts
                files []*File
                err error
        )
        if args = parseOpts(ctx, &opts, plain, args...); len(opts.exclude) > 0 {
                opts.exclude = mergex(ctx, plain, opts.exclude...)
        }

        if opts.timing {
                defer func(t time.Time) {
                        info(ctx, "wildcard time: %v", time.Now().Sub(t)).debug(1)
                } (time.Now())
        }

        if opts.dir != "" {
                files = wildcardPathPatsInDir(ctx, &opts, args...)
        } else if files, err = ctx.Project().wildcard(ctx, opts, args...); err != nil {
                erro(ctx, "wildcard failed: %v", err).debug(1)
                return
        }

        var vals []Value
        ForFiles: for _, file := range files {
                for _, x := range opts.exclude {
                        if ok, _, _ := x.match(ctx, file); ok {
                                continue ForFiles
                        }
                }
                if !(opts.names || opts.strs) {
                        vals = append(vals, file)
                } else if opts.strs {
                        vals = append(vals, MakeString(file.position, file.name))
                } else if strings.Contains(file.name, PathSep) {
                        vals = append(vals, MakePathStr(file.position, file.name))
                } else {
                        vals = append(vals, MakeBareword(file.position, file.name))
                }
        }
        res = MakeListOrScalar(ctx.Position(), vals)
        return
}

type builtinReadDirOpts struct {
        generalOpts
}
func builtinReadDir(ctx Context, w facet, args... Value) (res Value) {
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
        trim      bool `ta,trim,trim-all`
        trimLeft  bool `tl,trim-left`
        trimRight bool `tr,trim-right`
}
func builtinReadFile(ctx Context, w facet, args... Value) (res Value) {
        var (
                closured = closureProjects(ctx)
                pos = ctx.Position()
                opts builtinReadFileOpts
                l []Value
        )
        for _, a := range parseOpts(ctx, &opts, plain, args...) {
                var (
                        apos = a.Position()
                        str string
                        err error
                        s []byte
                        ok bool
                )
                if !apos.IsValid() { apos = pos }
                if _, str, ok = asOptFullname2(ctx, a, closured...); !ok || str == "" {
                        erro(at(ctx,apos), "%v is not a file", a).debug(1)
                        break
                } else if s, err = ioutil.ReadFile(str); err != nil {
                        erro(at(ctx,apos), "read file failed: %v", err).debug(1)
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
func builtinWriteFile(ctx Context, w facet, args... Value) (res Value) {
        // $(write-file filename,content)
        // $(write-file -p filename,content)
        var opts builtinWriteFileOpts
        if len(args) > 0 {
                var va = parseOpts(ctx, &opts, plain, args[1])
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
                erro(of(ctx,file), "touch: file fullname of '%v' is empty", file, err).debug(1)
                return
        }

        if dir := filepath.Dir(filename); optPath && dir != "." && dir != PathSep {
                if err = os.MkdirAll(dir, os.FileMode(optMode|0733)); err != nil {
                        erro(of(ctx,file), "touch: %v", err).debug(1)
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
        if fi, k := toFile(file); k && fi.info != nil {
                m = fi.info.Mode()
        } else if fi, e := os.Stat(filename); e == nil && fi != nil {
                m = fi.Mode()
        } else {
                var f *os.File
                if m = mode; m == 0 { m = os.FileMode(0600); mode = m }
                if f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, m&os.ModePerm); err != nil {
                        erro(of(ctx,file), "touch: %v", err).debug(1)
                } else if err = f.Close(); err != nil {
                        erro(of(ctx,file), "touch: %v", err).debug(1)
                }
        }
        if err == nil {
                if err = os.Chtimes(filename, at, mt); err != nil {
                        erro(of(ctx,file), "touch: %v", err).debug(1)
                }
        }
        if err == nil && mode != 0 && m != 0 && mode != m {
                if err = os.Chmod(filename, mode); err != nil {
                        erro(of(ctx,file), "touch: %v", err).debug(1)
                }
        }
        return
}

type builtinTouchFileOpts struct {
        generalOpts
        mode os.FileMode `m,mode;fm,filemode;fm,file-mode`
        path bool `p,path`
}
func builtinTouchFile(ctx Context, w facet, args... Value) (res Value) {
        // $(touch-file filename)
        // $(touch-file -p filename)
        var opts = builtinTouchFileOpts{ mode: os.FileMode(0600) }
        args = parseOpts(ctx, &opts, plain, args...)
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
func builtinGrep(ctx Context, w facet, args... Value) (res Value) {
        var (
                opts builtinGrepOpts
                vals, list []Value
                rxs []*regexp.Regexp // TODO: move it into builtinGrepOpts
                result Value
                nargs int
                err error

        )
        if nargs = len(args); !(nargs == 2 || nargs == 3) {
                erro(ctx, "wants exactly 2 args, e.g. $(grep -1 '^example$',$(file))").debug(1)
                return
        }

        if vals = parseOpts(ctx, &opts, plain, args[0]); nargs == 2 {
                args = args[1:]
        } else if nargs == 3 {
                result = args[1]
                args = args[2:]
        }
        for _, a := range vals {
                if s := a.Strval(ctx); s == "" {
                        erro(of(ctx,a), "empty regexp").debug(1)
                        return
                } else if r, e := regexp.Compile(s); e != nil {
                        erro(of(ctx,a), "%v", e).debug(1)
                        return
                } else {
                        rxs = append(rxs, r)
                }
        }

        vals = mergex(ctx, plain, args...)

        var pos = ctx.Position()
        var cc = autoContext{ Context:ctx, defs:make(autoDefMap) }
        var greped = func(line int, match []string) (done bool) {
                var vals []Value
                for i, s := range match {
                        if d, v := cc.autoSet(fmt.Sprintf("%d",i), MakeString(pos, s)); d == nil {
                                erro(ctx, "set $%d to '%s' failed", i, s).debug(1)
                                return
                        } else { vals = append(vals, v) }
                }
                defer func() {
                        for i, v := range vals {
                                if d, v := cc.autoSet(fmt.Sprintf("%d",i), v); d == nil {
                                        erro(ctx, "restore $%d to '%s' failed", i, v).debug(1)
                                }
                        }
                } ()
                list = append(list, result.expand(&cc, expandDigits|plain))
                return
        }

        for _, a := range vals {
                var file *os.File
                var filename = a.Strval(ctx)
                if filename == "" {
                        errostack(of(ctx,a), 5, "empty filename: %v (%T) (%v -> %v)",
                                a, a, args, vals).debug(64)
                        return
                }
                if file, err = os.Open(filename); err != nil {
                        errostack(of(ctx,a), 5, "%v", err).debug(128)
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

func (project *Project) resolveDef(ctx Context, name string) (res *def) {
        var obj Object
        if obj = project.resolveObject(ctx, name); !isNil(obj) { res, _ = obj.(*def) }
        return
}

func (project *Project) strExpandConfig(ctx Context, s string) (result string, err error) {
        var (
                pos Position
                res = new(bytes.Buffer)
                index, line = 0, 0
        )
        if d := autoGet(ctx, "-file"); d != nil {
                if f, y := toFile(d); y { pos.Filename = f.fullname() }
                // warn(ctx, "%T %v %v", v, v, pos)
        }
        for _, m := range rxConfigRef.FindAllStringSubmatchIndex(s, -1) {
                line += strings.Count(s[index:m[0]], "\n")
                pos.Line = 1 + line
                pos.Column = m[0] - index - strings.LastIndex(s[index:m[0]], "\n")

                fmt.Fprint(res, s[index:m[0]])
                index = m[1] // reset index immediately to keep forward

                var name string
                switch {
                case m[2] > m[0] && m[3] > m[2]: name = s[m[2]:m[3]] // ${VAR}
                case m[4] > m[0] && m[5] > m[4]: name = s[m[4]:m[5]] // @VAR@
                }

                var (
                        def *def
                        val Value
                )
                if def = project.resolveDef(ctx, name); def == nil {
                        if true {
                                prompt(ctx, "%v: %v undefined\n", pos, name)
                                warnstack(ctx, 10, "in %v", project).debug(6)
                        }
                        continue
                } else if val = def.Call(ctx); isNil(val) {
                        if false && (def.origin != DefExecute || def.value != nil) {
                                warn(of(ctx,def), "%v is nil (%T)", name, val).debug(1)
                        }
                        if cf := project.configuration(ctx); cf == nil {
                                erro(of(ctx,def), "%v: configuration file not defined", name, cf).debug(1)
                                return
                        } else if !cf.exists() {
                                prompt(ctx, "%s: file not exists (for %v)\n", cf.fullname(), name)
                                erro(of(ctx,def), "%v: configuration file not exists, try -conf first", name).debug(1)
                                return
                        }
                        continue
                }

                switch t := val.(type) {
                case *undef, undef: // FIXME: fmt.Fprintf(res, "#undef")
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
        if s, e := project.strExpandConfig(ctx, str); e != nil {
                erro(ctx, "%v: %v", str, err).debug(1)
                return e
        } else { str = s }

        var index = 0
        for _, m := range rxConfigure.FindAllStringSubmatchIndex(str, -1) {
                if _, err = out.WriteString(str[index:m[0]]); err != nil {
                        erro(ctx, "WriteString: %v", err).debug(1)
                        return
                } else { index = m[1] }

                var (
                        t bool
                        s string
                        def *def
                        verb = str[m[2]:m[3]]
                        name = str[m[4]:m[5]]
                        hasv = m[6] > m[0] && m[7] > m[6]
                )
                if def = project.resolveDef(ctx, name); def != nil { // t = def.True(ctx);
                        if val := def.Call(ctx); val == nil {
                                // noop, TODO: or #undef?
                        } else if _, undef := val.(*undef); undef {
                                _, err = out.WriteString(fmt.Sprintf("#undef /* %s */", name))
                                if err != nil { erro(ctx, "%v", err); return }
                                continue
                        } else {
                               t = val.True(ctx)
                        }
                }

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
                        } else if va, _, _ = plain.expand(ctx, def.value); len(va) == 1 {
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

func builtinUntraversed(ctx Context, w facet, args... Value) Value {
        var pos = ctx.Position()
        var vals = mergex(ctx, plain, args...)
        return untraversed{MakeListOrScalar(pos, vals)}
}

func builtinReturn(ctx Context, w facet, args... Value) Value {
        return &returner{valbase{ctx.Position()}, args }
}
