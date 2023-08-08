//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
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
    "sync"
    "fmt"
    "os"
    "io"
)

const (
    builtinCallable int = 0
    builtinCommand      = 1<<(iota-1)
)

type builtin_ struct {
    Context
    generalOpts
}
type builtin_a interface{ a(*invocation,facet) bool }
type builtin_c interface{ c(*invocation,facet) interface{} }
type builtin_x interface{ x(*invocation,facet) interface{} }
type builtin_1 interface{ x(*invocation) interface{} }
type builtin_0 interface{ x() interface{} }

var builtin_a_t = reflect.TypeOf((*builtin_a)(nil)).Elem()
var builtin_c_t = reflect.TypeOf((*builtin_c)(nil)).Elem()
var builtin_x_t = reflect.TypeOf((*builtin_x)(nil)).Elem()

var builtins = map[string]reflect.Type {
    `typeof`:    reflect.TypeOf((*builtin_typeof)(nil)).Elem(),
    `origin`:    reflect.TypeOf((*builtin_origin)(nil)).Elem(),
    `defined`:   reflect.TypeOf((*builtin_defined)(nil)).Elem(),

    `position`:  reflect.TypeOf((*builtin_position)(nil)).Elem(),
    `date`:      reflect.TypeOf((*builtin_date)(nil)).Elem(),

    `debug`:     reflect.TypeOf((*builtin_debug)(nil)).Elem(),
    `error`:     reflect.TypeOf((*builtin_error)(nil)).Elem(),
    `warning`:   reflect.TypeOf((*builtin_warning)(nil)).Elem(),
    `assert`:    reflect.TypeOf((*builtin_assert)(nil)).Elem(),
    `sure`:      reflect.TypeOf((*builtin_sure)(nil)).Elem(),

    // $(defor) (aka. defined-or)
    `defor`:     reflect.TypeOf((*builtin_defor)(nil)).Elem(), // $(defor $(x),$(y),$(z))  <=>  $(if $(defined $(x)),$(x),...)
    `or`:    reflect.TypeOf((*builtin_or)(nil)).Elem(),
    `and`:       reflect.TypeOf((*builtin_and)(nil)).Elem(),
    `not`:       reflect.TypeOf((*builtin_not)(nil)).Elem(),
    //`xor`:       reflect.TypeOf((*builtin_xor)(nil)).Elem(),

    `equal`:     reflect.TypeOf((*builtin_equal)(nil)).Elem(),
    `equals`:    reflect.TypeOf((*builtin_equal)(nil)).Elem(),
    `ne`:    reflect.TypeOf((*builtin_unequal)(nil)).Elem(),
    `not-equal`: reflect.TypeOf((*builtin_unequal)(nil)).Elem(),
    `match`:     reflect.TypeOf((*builtin_match)(nil)).Elem(),

    `greater`:   reflect.TypeOf((*builtin_greater)(nil)).Elem(),
    `less`:      reflect.TypeOf((*builtin_less)(nil)).Elem(),

    `case`:      reflect.TypeOf((*builtin_case)(nil)).Elem(),
    `if`:    reflect.TypeOf((*builtin_if)(nil)).Elem(),
    `ifeq`:      reflect.TypeOf((*builtin_ifeq)(nil)).Elem(),
    `ifne`:      reflect.TypeOf((*builtin_ifne)(nil)).Elem(),

    `foreach`:   reflect.TypeOf((*builtin_foreach)(nil)).Elem(),
    `count`:     reflect.TypeOf((*builtin_count)(nil)).Elem(),

    `auto`:      reflect.TypeOf((*builtin_auto)(nil)).Elem(),
    `var`:       reflect.TypeOf((*builtin_var)(nil)).Elem(),

    `call`:      reflect.TypeOf((*builtin_call)(nil)).Elem(),
    `closure`:   reflect.TypeOf((*builtin_closure)(nil)).Elem(),
    `defs`:      reflect.TypeOf((*builtin_defs)(nil)).Elem(),

    `env`:       reflect.TypeOf((*builtin_env)(nil)).Elem(),
    `value`:     reflect.TypeOf((*builtin_value)(nil)).Elem(),
    `list`:      reflect.TypeOf((*builtin_list)(nil)).Elem(),

    `shell`:     reflect.TypeOf((*builtin_shell)(nil)).Elem(),
    `which`:     reflect.TypeOf((*builtin_which)(nil)).Elem(),

    `plus`:      reflect.TypeOf((*builtin_plus)(nil)).Elem(),
    `minus`:     reflect.TypeOf((*builtin_minus)(nil)).Elem(),
    `multiply`:  reflect.TypeOf((*builtin_multiply)(nil)).Elem(),
    `mul`:       reflect.TypeOf((*builtin_multiply)(nil)).Elem(),
    `divide`:    reflect.TypeOf((*builtin_divide)(nil)).Elem(),
    `div`:       reflect.TypeOf((*builtin_divide)(nil)).Elem(),

    `unique`:     reflect.TypeOf((*builtin_unique)(nil)).Elem(),
    `join`:       reflect.TypeOf((*builtin_join)(nil)).Elem(), // concat
    `quote`:      reflect.TypeOf((*builtin_quote)(nil)).Elem(),
    `quote-join`: reflect.TypeOf((*builtin_quotejoin)(nil)).Elem(),

    `split-string`:      reflect.TypeOf((*builtin_splitstring)(nil)).Elem(),
    `split-quote`:       reflect.TypeOf((*builtin_splitquote)(nil)).Elem(),
    `split-quote-join`:  reflect.TypeOf((*builtin_splitquotejoin)(nil)).Elem(),
    `split-join-quote`:  reflect.TypeOf((*builtin_splitjoinquote)(nil)).Elem(),
    `field`:         reflect.TypeOf((*builtin_field)(nil)).Elem(),
    `fields`:        reflect.TypeOf((*builtin_fields)(nil)).Elem(),

    // `usee`:      reflect.TypeOf((*builtin_usee)(nil)).Elem(),
    `uses`:         reflect.TypeOf((*builtin_uses)(nil)).Elem(),

    `path`:         reflect.TypeOf((*builtin_path)(nil)).Elem(),
    `bare`:         reflect.TypeOf((*builtin_bare)(nil)).Elem(),
    `bareword`:     reflect.TypeOf((*builtin_bareword)(nil)).Elem(),
    `str`:          reflect.TypeOf((*builtin_str)(nil)).Elem(),
    `string`:       reflect.TypeOf((*builtin_string)(nil)).Elem(),
    `strval`:       reflect.TypeOf((*builtin_strval)(nil)).Elem(),
    `strip`:        reflect.TypeOf((*builtin_strip)(nil)).Elem(),
    `trim`:         reflect.TypeOf((*builtin_trim)(nil)).Elem(),
    `trim-space`:   reflect.TypeOf((*builtin_trimspace)(nil)).Elem(),
    `trim-left`:    reflect.TypeOf((*builtin_trimleft)(nil)).Elem(),
    `trim-right`:   reflect.TypeOf((*builtin_trimright)(nil)).Elem(),
    `trim-prefix`:  reflect.TypeOf((*builtin_trimprefix)(nil)).Elem(),
    `trim-suffix`:  reflect.TypeOf((*builtin_trimsuffix)(nil)).Elem(),
    `trim-ext`:     reflect.TypeOf((*builtin_trimext)(nil)).Elem(),

    `addprefix`:    reflect.TypeOf((*builtin_addprefix)(nil)).Elem(),
    `addsuffix`:    reflect.TypeOf((*builtin_addsuffix)(nil)).Elem(),

    `uppercase`:    reflect.TypeOf((*builtin_uppercase)(nil)).Elem(),
    `lowercase`:    reflect.TypeOf((*builtin_lowercase)(nil)).Elem(),
    `title`:        reflect.TypeOf((*builtin_title)(nil)).Elem(),
    `indent`:       reflect.TypeOf((*builtin_indent)(nil)).Elem(),
    `substring`:    reflect.TypeOf((*builtin_substring)(nil)).Elem(),

    // https://www.gnu.org/software/make/manual/html_node/Text-Functions.html
    `subst`:        reflect.TypeOf((*builtin_subst)(nil)).Elem(),
    `patsubst`:     reflect.TypeOf((*builtin_patsubst)(nil)).Elem(),

    `contains`:     reflect.TypeOf((*builtin_contains)(nil)).Elem(),
    `filter`:       reflect.TypeOf((*builtin_filter)(nil)).Elem(),
    `filter-out`:   reflect.TypeOf((*builtin_filterout)(nil)).Elem(),

    `decode-base64`:reflect.TypeOf((*builtin_decodebase64)(nil)).Elem(),
    `encode-base64`:reflect.TypeOf((*builtin_encodebase64)(nil)).Elem(),

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

    `fullname`:   reflect.TypeOf((*builtin_fullname)(nil)).Elem(),
    `ext`:    reflect.TypeOf((*builtin_ext)(nil)).Elem(),

    `base`:       reflect.TypeOf((*builtin_base)(nil)).Elem(),
    `bases`:      reflect.TypeOf((*builtin_bases)(nil)).Elem(),
    `base2`:      reflect.TypeOf((*builtin_base2)(nil)).Elem(),
    `base3`:      reflect.TypeOf((*builtin_base3)(nil)).Elem(),
    `base4`:      reflect.TypeOf((*builtin_base4)(nil)).Elem(),
    `base5`:      reflect.TypeOf((*builtin_base5)(nil)).Elem(),
    `base6`:      reflect.TypeOf((*builtin_base6)(nil)).Elem(),
    `base7`:      reflect.TypeOf((*builtin_base7)(nil)).Elem(),
    `base8`:      reflect.TypeOf((*builtin_base8)(nil)).Elem(),
    `base9`:      reflect.TypeOf((*builtin_base9)(nil)).Elem(),

    `chopdir`:    reflect.TypeOf((*builtin_chopdir)(nil)).Elem(),

    `dir`:    reflect.TypeOf((*builtin_dir)(nil)).Elem(),
    `dirs`:       reflect.TypeOf((*builtin_dirs)(nil)).Elem(),
    `dir2`:       reflect.TypeOf((*builtin_dir2)(nil)).Elem(),
    `dir3`:       reflect.TypeOf((*builtin_dir3)(nil)).Elem(),
    `dir4`:       reflect.TypeOf((*builtin_dir4)(nil)).Elem(),
    `dir5`:       reflect.TypeOf((*builtin_dir5)(nil)).Elem(),
    `dir6`:       reflect.TypeOf((*builtin_dir6)(nil)).Elem(),
    `dir7`:       reflect.TypeOf((*builtin_dir7)(nil)).Elem(),
    `dir8`:       reflect.TypeOf((*builtin_dir8)(nil)).Elem(),
    `dir9`:       reflect.TypeOf((*builtin_dir9)(nil)).Elem(),

    `undir`:      reflect.TypeOf((*builtin_undir)(nil)).Elem(),
    `undirs`:     reflect.TypeOf((*builtin_undirs)(nil)).Elem(),
    `undir2`:     reflect.TypeOf((*builtin_undir2)(nil)).Elem(),
    `undir3`:     reflect.TypeOf((*builtin_undir3)(nil)).Elem(),
    `undir4`:     reflect.TypeOf((*builtin_undir4)(nil)).Elem(),
    `undir5`:     reflect.TypeOf((*builtin_undir5)(nil)).Elem(),
    `undir6`:     reflect.TypeOf((*builtin_undir6)(nil)).Elem(),
    `undir7`:     reflect.TypeOf((*builtin_undir7)(nil)).Elem(),
    `undir8`:     reflect.TypeOf((*builtin_undir8)(nil)).Elem(),
    `undir9`:     reflect.TypeOf((*builtin_undir9)(nil)).Elem(),

    `reldir`: reflect.TypeOf((*builtin_reldir)(nil)).Elem(),
    `relative-dir`: reflect.TypeOf((*builtin_reldir)(nil)).Elem(),

    `file`:       reflect.TypeOf((*builtin_file)(nil)).Elem(),
    `stat`:       reflect.TypeOf((*builtin_stat)(nil)).Elem(),// stat (deprecates file-exists)
    `glob`:       reflect.TypeOf((*builtin_glob)(nil)).Elem(),
    `wildcard`:   reflect.TypeOf((*builtin_wildcard)(nil)).Elem(),

    `read-dir`:   reflect.TypeOf((*builtin_readdir)(nil)).Elem(),  // io/ioutil/ioutil.go
    `read-file`:  reflect.TypeOf((*builtin_readfile)(nil)).Elem(),  // io/ioutil/ioutil.go

    `grep`:       reflect.TypeOf((*builtin_grep)(nil)).Elem(),

    `untraversed`: reflect.TypeOf((*builtin_untraversed)(nil)).Elem(),

    // commands ------------------------------------------------------------------
    `print`:    reflect.TypeOf((*builtin_print)(nil)).Elem(),
    `printf`:       reflect.TypeOf((*builtin_printf)(nil)).Elem(),
    `printl`:       reflect.TypeOf((*builtin_printl)(nil)).Elem(),
    `println`:      reflect.TypeOf((*builtin_println)(nil)).Elem(),

    `plain`:    reflect.TypeOf((*builtin_plain)(nil)).Elem(),

    `append`:       reflect.TypeOf((*builtin_append)(nil)).Elem(),
    // `pop`:      reflect.TypeOf((*builtin_pop)(nil)).Elem(),

    `write-file`:   reflect.TypeOf((*builtin_writefile)(nil)).Elem(),  // io/ioutil/ioutil.go
    `touch-file`:   reflect.TypeOf((*builtin_readfile)(nil)).Elem(),  // io/ioutil/ioutil.go

    `push-context`: reflect.TypeOf((*builtin_pushcontext)(nil)).Elem(),
    `pop-context`:  reflect.TypeOf((*builtin_popcontext)(nil)).Elem(),

    `mkdir`:    reflect.TypeOf((*builtin_mkdir)(nil)).Elem(),     // os/file.go
    `chdir`:    reflect.TypeOf((*builtin_chdir)(nil)).Elem(),     // os/file.go
    `rename`:       reflect.TypeOf((*builtin_rename)(nil)).Elem(),    // os/file.go
    `remove`:       reflect.TypeOf((*builtin_remove)(nil)).Elem(),    // os/file_*.go
    `truncate`:     reflect.TypeOf((*builtin_truncate)(nil)).Elem(),  // os/file_*.go
    `link`:     reflect.TypeOf((*builtin_link)(nil)).Elem(),      // os/file_*.go
    `symlink`:      reflect.TypeOf((*builtin_symlink)(nil)).Elem(),   // os/file_*.go

    `serve-http`:   reflect.TypeOf((*builtin_servehttp)(nil)).Elem(),

    `return`:       reflect.TypeOf((*builtin_return)(nil)).Elem(),
}

func EscapedString(ctx Context, v Value) (s string) {
    if p, ok := v.(*String); ok {
        s = strings.Replace(p.strval(ctx), "\\'", "'", -1)
    } else {
        s = v.strval(ctx)
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

func _set(ctx Context, val reflect.Value, v Value) {
    switch val.Kind() {
    case reflect.Bool:
        if t := v == nil || v.true(ctx); true { val.SetBool(t) }
    case reflect.Float32, reflect.Float64:
        if t, e := v.float(ctx); e == nil { val.SetFloat(t) } else {
            erro(ctx, "%v: %v", v, e).debug(10)
        }
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        if t, e := v.int(ctx); e == nil { val.SetInt(t) } else {
            erro(ctx, "%v: %v", v, e).debug(10)
        }
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        if t, e := v.int(ctx); e == nil { val.SetUint(uint64(t)) } else {
            erro(ctx, "%v: %v", v, t).debug(10)
        }
    case reflect.String:
        val.SetString(v.strval(ctx))
    case reflect.Slice:
        if p := reflect.New(val.Type().Elem()); p.Kind() == reflect.Ptr {
            var t = p.Elem()
            _set(ctx, t, v)
            val.Set(reflect.Append(val, t))
        }
    case reflect.Interface:
        switch val.Type().String() {
        case "smart.Value":
            val.Set(reflect.ValueOf(v))
        default:
            erro(of(ctx,v), "option type unsupported: %T %v -> %v, %v", v, v, val.Kind(), val.Type()).debug(1)
        }
    case reflect.Ptr:
        switch val.Type().Elem().String() {
        case "smart.fullnameOpt":
            if x := v.expand(ctx, plain|expandFullName); isTrivial(x) {
                erro(of(ctx, v), "expecting file value: %T %v", v, v).debug(1)
            } else if o, y := (as{x}.fullnameOpt(ctx)); y && o.Value != nil {
                val.Set(reflect.ValueOf(&o))
            } else {
                erro(of(ctx,v), "%v: not a file: %v → %T %v", ctx.Project(), v, x, x)
                errostack(ctx, 5).debug(32)
            }
        case "smart.File":
            if x := v.expand(ctx, plain); isNone(x) {
                erro(of(ctx,v), "expecting file value: %T %v", v, v).debug(1)
            } else if file, y := toFile(x); y {
                val.Set(reflect.ValueOf(file))
            } else if proj := ctx.Project(); proj == nil {
                erro(of(ctx,x), "no current project to find file '%v'", x).debug(1)
            } else if file = proj.file(ctx, x.strval(ctx)); file != nil {
                val.Set(reflect.ValueOf(file))
            } else {
                erro(of(ctx,v), "'%v' is not a file", x).debug(1)
            }
        case "regexp.Regexp":
            if rx, e := regexp.Compile(v.strval(ctx)); e != nil {
                erro(of(ctx,v), "compile regexp '%v' failed: %v", v, e).debug(1)
            } else {
                val.Set(reflect.ValueOf(rx))
            }
        default:
            erro(of(ctx,v), "option type unsupported: %T %v -> %v, %v", v, v, val.Elem().Kind(), val.Type().Elem()).debug(1)
        }
    default:
        switch val.Type().String() {
        case "fs.FileMode", "os.FileMode": // aka. reflect.Uint32
            if t, e := v.int(ctx); e == nil {
                if t == 0 { warn(of(ctx,v), "zero file mode").debug(1) }
                val.SetUint(uint64(t))
            } else {
                erro(ctx, "%v: %v", v, t).debug(1)
            }
        case "regex.Regex": // aka. reflect.Ptr
            erro(of(ctx,v), "TODO: regexp: %T %v -> %v, %v", v, v, val.Kind(), val.Type()).debug(1)
        default:
            erro(of(ctx,v), "option type unsupported: %T %v -> %v, %v", v, v, val.Kind(), val.Type()).debug(1)
        }
    }
}

func _parseOpt(ctx Context, tag reflect.StructTag, field reflect.Value, args ...Value) (rest []Value) {
    var (
        val = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
        opts []string // opt names
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

ForArgs:
    for _, arg := range args {
        var (
            y bool
            f flag
            a, value Value
        )
        if u, y := arg.(unexpanded); y {
            a = u.Value
        } else {
            a = arg
        }

        // don't parse patterns, e.g. -I%
        if !a.patterned(ctx) { switch t := a.(type) {
        case  flag: f, value = t, MakeBoolean(t.Position(), true)
        case *pair: if f, y = t.Key.(flag); y { value = t.Value }
        case *argumented: if f, y = t.Value.(flag); y { value = ease(ctx, t.args) }
        }}

        if f.Value == nil {
            rest = append(rest, arg)
            continue ForArgs
        }

        for i := 0; i < len(opts); i += 1 { if _, y = f.opt(ctx, opts[i]); y {
            _set(ctx, val, value)
            continue ForArgs
        }}

        rest = append(rest, arg)
    }

    switch val.Type().String() { // os.FileMode(0640)
    case "fs.FileMode", "os.FileMode": if val.Uint() == 0 { val.SetUint(0640) }
    }
    return
}

func _parseOpts(ctx Context, opts reflect.Value, w facet, args []Value) (rest []Value) {
    if w == 0 {
        rest = merge(args...) // NOTE: set the returning args first of all!
    } else if false {
        if false && // FIXME: the args[1].expand(ctx, w) causes 'd.x == nil'
            w == 0 && len(args)>1 && args[0].String() == "-plain" &&
            args[1].String() == "$(configure~$(target.sys).features)" {
            var d = args[1].(*delegate)
            var t = args[1].expand(ctx, w)
            warn(ctx, "%v ; w=%016b", args, w)
            warn(ctx, "%T %v %p -> %T %v %p %p", d.x, args[1], d,
                t.(*delegate).x, t, t, t.(*delegate)).debug(1)
        }
        rest = xmerge(ctx, w, args...)
    } else {
        rest = umerge(true, args...)
    }

    if opts.Kind() != reflect.Ptr {
        erro(ctx, "opts must be ptr: %v", opts.Kind()).debug(10)
        return
    } else if opts = opts.Elem(); opts.Kind() != reflect.Struct {
        erro(ctx, "opts is not ptr of struct: %v", opts.Kind()).debug(1)
        return
    }

    var (
        otyp = opts.Type()
        builtin, general, modifier reflect.Value
    )
    if false { info(ctx, "opts: %v, %v", opts.Kind(), otyp) }
    for i := 0; i < otyp.NumField(); i += 1 {
        var ft = otyp.Field(i)
        var fv = opts.Field(i)
        if t := fv.Type(); fv.Kind() != reflect.Struct {
            if ft.Anonymous && ft.Name == "Context" && t.String() == "smart.Context" {
                if false { info(ctx, "%v %v %v %v", ft.Name, ft.Tag, t, rest) }
                continue
            } else {
                rest = _parseOpt(ctx, ft.Tag, fv, rest...)
            }
        } else if !ft.Anonymous {
            continue
        } else if ft.Name == "generalOpts" {
            // genOpts = (*generalOpts)(unsafe.Pointer(fv.UnsafeAddr()))
            general = fv.Addr()
        } else if strings.HasPrefix(ft.Name, "builtin_") {
            if builtin.IsValid() { warn(ctx, "embedded multiple builtins: %v", ft).debug(3) }
            builtin = fv.Addr()
        } else if strings.HasPrefix(ft.Name, "modifier_") {
            if modifier.IsValid() { warn(ctx, "embedded multiple modifiers: %v", ft).debug(3) }
            modifier = fv.Addr()
        }
    }
    if  builtin.IsValid() { rest = _parseOpts(ctx,  builtin, w, rest) }
    if  general.IsValid() { rest = _parseOpts(ctx,  general, w, rest) }
    if modifier.IsValid() { rest = _parseOpts(ctx, modifier, w, rest) }
    return
}
func parseOpts(ctx Context, store interface{}, w facet, args ...Value) (rest []Value) {
    return _parseOpts(ctx, reflect.ValueOf(store), w, args)
}

// see https://go.dev/doc/tutorial/generics
func _opts[Opts interface{}](ctx Context, w facet, args ...Value) (opts Opts, res []Value) {
    res = parseOpts(ctx, &opts, w, args...)
    return
}

func _parseHeadArgs(ctx Context, store interface{}, w facet, args ...Value) (head, rest []Value) {
    if len(args) == 0 {
        // zero args
    } else if head = parseOpts(ctx, store, w, args[0]); len(head) > 0 {
        rest = args[1:] //xmerge(ctx, w, args[1:]...)
    } else if len(args) == 1 {
        // done
    } else if head = xmerge(ctx, w, args[1]); len(args) > 2 {
        rest = args[2:] //xmerge(ctx, w, args[2:]...)
    }
    return
}

func _parseHeadArgsMerge(ctx Context, store interface{}, w facet, args ...Value) (res []Value) {
    var head, rest = _parseHeadArgs(ctx, store, w, args...)
    res = append(head, rest...)
    return
}

func _parseHeadArgsRequired(ctx Context, store interface{}, w facet, args ...Value) (head, rest []Value) {
    head, rest = _parseHeadArgs(ctx, store, w, args...)
    if len(head) == 0 || len(rest) == 0 {
        erro(ctx, "insufficient number of arguments").debug(6)
    }
    return
}

type builtin_noop struct { builtin_ }
func (ctx *builtin_noop) c(ic *invocation, w facet) (res interface{}) { return }
func (ctx *builtin_noop) x(ic *invocation, w facet) (res interface{}) { return }

func typeof(arg interface{}) (s string) {
    switch a := arg.(type) {
    case *List:
        if n := len(a.Elems); n == 1 {
            switch v := a.Elems[0].(type) {
            case *delegate: // FIXME: recursively undelegate types
                if d, _ := v.x.(*def); d != nil {
                    s = fmt.Sprintf("%T", d.value) //s = d.value.Type().String()
                } else {
                    s = "unknown"
                }
            default:
                s = fmt.Sprintf("%T", v) //s = v.Type().String()
            }
        } else if n > 1 {
            s = "List"
        } else if false {
            s = "None"
        }
    default:
        // FIXME: this should be an exception (panic).
        s = fmt.Sprintf("%T", a) //s = a.Type().String()
    }
    if s != "" {
        s = strings.TrimPrefix(s, "*")
        s = strings.TrimPrefix(s, "smart.")
        if false { s = strings.TrimPrefix(s, "ast.") }
        // s = strings.ReplaceAll(strings.TrimPrefix(s, "*"), "smart.", "")
    }
    return
}

type builtin_typeof struct { builtin_
    expand bool `x,e,ex,exp,expand`
}
func (ctx *builtin_typeof) a(ic *invocation, w facet) (skip bool) { return }
func (ctx *builtin_typeof) x(ic *invocation, w facet) (res interface{}) {
    var elems []Value
    for _, arg := range ic.a {
        if ctx.expand { arg = arg.expand(ctx, w) }
        // Arguments are passed in a list:
        //   $(fun abc)             args: (abc)
        //   $(fun a,b,c)           args: (a),(b),(c)
        //   $(fun a b c,1 2 3)     args: (a b c),(1 2 3)
        elems = append(elems, MakeBareword(arg.Position(), typeof(arg)))
    }
    return elems
}

type builtin_origin struct { builtin_ }
func (ctx *builtin_origin) x(ic *invocation, w facet) (res interface{}) {
    var elems []Value
    var scope = ctx.Scope()
    for _, arg := range ic.a { if s := arg.strval(ctx); s == "" {
        elems = append(elems, makeNull(arg.Position()))
    } else if d := scope.FindDef(s); d != nil {
        elems = append(elems, MakeString(arg.Position(), d.origin.String()))
    } else {
        elems = append(elems, makeNull(arg.Position()))
    }}
    return elems
}

type builtin_defined struct { builtin_ }
func (ctx *builtin_defined) x(ic *invocation, w facet) (res interface{}) {
    var elems []Value
    for _, arg := range ic.a {
        var _, unresolved = arg.(unresolved)
        elems = append(elems, MakeBoolean(arg.Position(), !unresolved))
    }
    return elems
}

type builtin_pushcontext struct { builtin_ }
func (ctx *builtin_pushcontext) c(ic *invocation, w facet) (res interface{}) {
    var (
        scope = ctx.Scope()
        uc = ctx.universe()
        m map[string]*def
    )
    for _, arg := range ic.a {
        var s = arg.strval(ctx)
        if s == "" { continue }
        if m == nil { m = make(map[string]*def) }

        var t *def
        if o := scope.Lookup(s); o != nil { if d, y := o.(*def); y {
            t = new(def) ; *t = *d
        }}
        m[s] = t
    }
    uc.globe.stack = append(uc.globe.stack, m)
    return
}

type builtin_popcontext struct { builtin_
    rules []Value `r,rule,rules`
}
func (ctx *builtin_popcontext) c(ic *invocation, w facet) (res interface{}) {
    for _, arg := range ic.a {
        warn(ctx, "unused argument: %T %v", arg, arg).debug(1)
        break
    }

    var rules []Value
    for _, r := range ctx.rules { if v, y := r.(*group); !y {
        rules = append(rules, v)
    } else {
        rules = append(rules, v.Elems...)
    }}

    var scope = ctx.Scope()
    var uc = ctx.universe()
    var l = len(uc.globe.stack)
    if l == 0 { return }
    for s, d := range uc.globe.stack[l-1] { if d == nil { if s == "" { continue }
        scope.mutex.Lock()
        delete(scope.elems, s)
        scope.mutex.Unlock()
    } else if o := scope.Lookup(d.name_); o != nil { if t, ok := o.(*def); ok {
        *t = *d
    }}}
    uc.globe.stack = uc.globe.stack[0:l-1]
    return
}

type builtin_position struct { builtin_
    filename bool `f,filename`
    filenameQuoted bool `q,quote-filename;qf,quoted-filename`
    line bool `l,line`
    column bool `c,column`
    addLine int `a,add;al,add-line`
    addColumn int `ac,add-column`
}
func (ctx *builtin_position) x(ic *invocation, w facet) (res interface{}) {
    var vals []Value
    var pos = ctx.Position()
    if ctx.filename {
        vals = append(vals, MakeString(pos, pos.Filename))
    } else if ctx.filenameQuoted {
        var s = pos.Filename //strconv.Quote(pos.Filename)
        vals = append(vals, MakeString(pos, "\""+s+"\""))
    }

    if ctx.line   { vals = append(vals, MakeInt(pos, int64(pos.Line + ctx.addLine))) }
    if ctx.column { vals = append(vals, MakeInt(pos, int64(pos.Column + ctx.addColumn))) }

    if len(vals) == 0 { return MakeString(pos, pos.String()) }
    if len(vals) == 1 { return vals[0] }
    return vals
}

type builtin_date struct { builtin_
    time bool `t,tm,time,n,now`
}
func (ctx *builtin_date) x(ic *invocation, w facet) (res interface{}) {
    if t := time.Now(); len(ic.a) > 0 {
        var vals []Value
        for _, a := range ic.a {
            var s string
            if s = a.strval(ctx); s == "" {
                s = t.String()
            } else if s = t.Format(s); s == "" {
                s = fmt.Sprintf("%v", t)
            }
            vals = append(vals, MakeString(a.Position(), s))
        }
        return vals
    } else if ctx.time {
        res = MakeTime(ctx.Position(), t)
    } else {
        res = MakeDate(ctx.Position(), t)
    }
    return
}

type builtin_debug struct { builtin_
    s int `s,stack`
    n int `n,num`
}
func (ctx *builtin_debug) x(ic *invocation, w facet) (res interface{}) {
    var s bytes.Buffer
    for i, a := range ic.a {
        if i > 0 { fmt.Fprintf(&s, " ") }
        fmt.Fprintf(&s, "%s", a.strval(ctx))
    }
    warnstack(ctx, ctx.s, "%s", s.String()).debug(ctx.n)
    return
}

type builtin_error struct { builtin_ }
func (ctx *builtin_error) x(ic *invocation, w facet) (res interface{}) {
    defer ctx.dia().trace(ctx, "builtin_error")

    var s bytes.Buffer
    for i, a := range ic.a {
        if i > 0 { fmt.Fprintf(&s, " ") }
        fmt.Fprintf(&s, "%s", a.strval(ctx))
    }

    errostack(ctx, 5, "%s", s.String()).debug(1)
    return
}

type builtin_warning struct { builtin_ }
func (ctx *builtin_warning) x(ic *invocation, w facet) (res interface{}) {
    var s bytes.Buffer
    for i, a := range ic.a {
        if i > 0 { fmt.Fprintf(&s, " ") }
        fmt.Fprintf(&s, "%s", a.strval(ctx))
    }
    warn(ctx, "%s", s).debug(1)
    return
}

type builtin_assert struct { builtin_
    msg string `m,msg,message`
}
func (ctx *builtin_assert) a(ic *invocation, w facet) (skip bool) { return }
func (ctx *builtin_assert) c(ic *invocation, w facet) (res interface{}) { return ctx.x(ic, w) }
func (ctx *builtin_assert) x(ic *invocation, w facet) (res interface{}) {
    defer ctx.dia().trace(ctx, "builtin_assert")

    const sn = 1
    var t = diagError ; if ctx.warn { t = diagWarn }
    var d = ctx.debug ; if d < 1 { d = 1 }

    var hook = ctx.universe().hooks.assert
    if ic.a == nil && hook != nil && !hook(ctx, nil, false) {
        prompt(ctx, "assert: %v\n", ic.a)
        diagstack(ctx, sn, t).debug(d)
    }

    var cc = ctx.Context
    for _, a := range ic.a { ctx.Context = at(cc, a.Position()) ; var okay = a.true(ctx)
        if hook != nil && hook(ctx, a, okay) { continue }
        if okay { continue }
        if false {
            var v = a.expand(ctx, strval)
            prompt(ctx, "assert: %v: %v ⇒ %v: %v\n", typeof(a), a, typeof(v), v)
            diagstack(ctx, sn, t, "%v: %v ⇒ %s", typeof(a), a, a.strval(ctx)).debug(d)
        } else if false {
            diagstack(ctx, sn, t, "%v: %v ⇒ %s", typeof(a), a, a.strval(ctx)).debug(d)
        } else {
            diagstack(ctx, sn, t, "%v: %v", typeof(a), a).debug(d)
        }
    }

    if ctx.fail { panic(failure{"%v: failed assertion",ia(cc.Position())}) }
    return
}

type builtin_sure struct { builtin_ }
func (ctx *builtin_sure) x(ic *invocation, w facet) (res interface{}) {
    defer ctx.dia().trace(ctx, "builtin_sure")

    for _, a := range ic.a { if !a.true(ctx) {
        erro(of(ctx,a), "assert: %T %v", a, a).debug(1)
    }}
    return ic.a
}

// $(defor $(x),$(y),$(z)) is identical to $(if $(defined $(x)),$(x),...)
type builtin_defor struct { builtin_ }
func (ctx *builtin_defor) x(ic *invocation, w facet) (res interface{}) {
    for _, a := range umerge(true, ic.a...) { if _, y := a.(unresolved); y {
        continue
    } else {
        res = a
        break
    }}
    return
}

type builtin_or struct { builtin_ }
func (ctx *builtin_or) _a(ic *invocation, w facet) (skip bool) {
    for i, a := range ic.a {
        if false && !ctx.forth {
            if _, y := a.(unexpanded); y { skip = true }
        }
        if false && ctx.universe().db("or") {
            noted(ctx, "or: %v %d. %v ; %v %v %v", ic.a, i, a.expand(ctx, w), autoDef(ctx, "1"), autoDef(ctx, "2"), autoDef(ctx, "3"))
        }
        ic.a[i] = a.expand(ctx, w)
    }
    return
}
func (ctx *builtin_or) x(ic *invocation, w facet) (res interface{}) {
    for _, a := range umerge(true, ic.a...) {
        if a.expandable(ctx, w) { a = a.expand(ctx, w) }
        if !ctx.forth { if u, y := a.(unexpanded); y {
            if false && ctx.universe().db("or") { v := u.Value
                noted(ctx, "or: %T %v %030b", v, v, w&expandDefAssign).debug(1)
            }
            return skip{}
        }}
        if a.true(ctx) { return a }
    }
    return
}

type builtin_and struct { builtin_ }
func (ctx *builtin_and) _a(ic *invocation, w facet) (skip bool) {
    for i, a := range ic.a {
        if false && !ctx.forth {
            if _, y := a.(unexpanded); y { skip = true }
        }
        ic.a[i] = a.expand(ctx, w)
    }
    return
}
func (ctx *builtin_and) x(ic *invocation, w facet) (res interface{}) {
    for _, a := range umerge(true, ic.a...) {
        if !ctx.forth {
            if _, y := a.(unexpanded); y { return skip{} }
        }
        if a.true(ctx) { res = a } else { return nil }
    }
    return
}

// $(not x y z) => (not (or x y z))
// $(not x,y,z) => (and (not x) (not y) (not z))
type builtin_not struct { builtin_ }
func (ctx *builtin_not) x(ic *invocation, w facet) (res interface{}) {
    var t bool
    for _, a := range ic.a { if t = a.true(ctx); t { break } }
    if n := ctx.debug; n>0 { warnstack(ctx, 3).debug(n) }
    return !t
}

type builtin_unequal struct { builtin_
    strval bool `s,sv,strval`
}
func (ctx *builtin_unequal) x(ic *invocation, w facet) (res interface{}) {
    if ctx.trace { ctx.dia().trace(ctx, "unequal") }

    if len(ic.a) != 2 {
        erro(ctx, "unequal: wrong number of arguments: %v", ic.a)
        erro(ctx, "try: $(unequal <value-list>,<value-list>)").debug(1)
        return
    }

    var t bool
    var a = ic.a[0].expand(ctx, strval)
    var b = ic.a[1].expand(ctx, strval)
    if ctx.strval {
        t = a.strval(ctx) != b.strval(ctx)
    } else {
        t = a.cmp(ctx, b) != cmpEqual
    }

    if t {
        res = MakeBoolean(ctx.Position(), true)
    } else if n := ctx.debug; n>0 {
        if u, y := a.(unexpanded); y {
            warn(of(ctx,a), "unequal: a: %T %v (unexpanded)", u.Value, a)
        } else if l, y := a.(*List); y {
            var v = l.Elems[0]
            warn(of(ctx,a), "unequal: a: %T(len=%d), %T %v", a, len(l.Elems), v, v)
        } else {
            warn(of(ctx,a), "unequal: a: %T %v", a, a)
        }
        if u, y := b.(unexpanded); y {
            warn(of(ctx,a), "unequal: b: %T %v (unexpanded)", u.Value, b)
        } else if l, y := b.(*List); y {
            var v = l.Elems[0]
            warn(of(ctx,b), "unequal: b: %T(len=%d), %T %v", b, len(l.Elems), v, v)
        } else {
            warn(of(ctx,b), "unequal: b: %T %v", b, b)
        }
        warnstack(ctx, n, "unequal: %v", t).debug(n)
    } else if len(ic.a)>2 {
        warnstack(of(ctx,ic.a[2]), 1, "unequal: extra args specified: %v", ic.a[2]).debug(1)
    }
    return
}

type builtin_equal struct { builtin_
    strval bool `s,sv,strval`
}
func (ctx *builtin_equal) x(ic *invocation, w facet) (res interface{}) {
    if ctx.trace { ctx.dia().trace(ctx, "equal") }

    if len(ic.a) > 0 {
        if a := umerge(true, ic.a[0]); len(a) == 1 {
            ic.a[0] = a[0]
        } else {
            ic.a[0] = MakeList(ic.a[0].Position(), a...)
        }
    }

    if len(ic.a) != 2 {
        erro(ctx, "equal: wrong number of arguments: %v", ic.a)
        erro(ctx, "try: $(equal <value-list>,<value-list>)").debug(1)
        return
    }

    var t bool
    var a = ic.a[0].expand(ctx, strval)
    var b = ic.a[1].expand(ctx, strval)
    if ctx.strval {
        t = a.strval(ctx) == b.strval(ctx)
    } else {
        t = a.cmp(ctx, b) == cmpEqual
    }

    if t {
        res = MakeBoolean(ctx.Position(), true)
    } else if n := ctx.debug; n>0 {
        if u, y := a.(unexpanded); y {
            warn(of(ctx,a), "equal: a: %T %v (unexpanded, %s)", u.Value, a, a.strval(ctx))
        } else if l, y := a.(*List); y { var v = l.Elems[0]
            warn(of(ctx,a), "equal: a: %T(len=%d), %T %v", a, len(l.Elems), v, v)
        } else {
            warn(of(ctx,a), "equal: a: %T %v (%s)", a, a, a.strval(ctx))
        }
        if u, y := b.(unexpanded); y {
            warn(of(ctx,a), "equal: b: %T %v (unexpanded, %s)", u.Value, b, b.strval(ctx))
        } else if l, y := b.(*List); y { var v = l.Elems[0]
            warn(of(ctx,b), "equal: b: %T(len=%d), %T %v", b, len(l.Elems), v, v)
        } else {
            warn(of(ctx,b), "equal: b: %T %v (%s)", b, b, b.strval(ctx))
        }
        warnstack(ctx, n).debug(n)
    } else if len(ic.a)>2 {
        warnstack(of(ctx,ic.a[2]), 1, "equal: extra args specified: %v", ic.a[2]).debug(1)
    }
    return
}

type builtin_greater struct { builtin_ }
func (ctx *builtin_greater) x(ic *invocation, w facet) (res interface{}) {
    if n := len(ic.a); n != 2 {
        erro(ctx, "wrong number of arguments, try: $(greater <value-list>,<value-list>)")
    } else if cmp := ic.a[0].cmp(ctx, ic.a[1]); cmp == cmpGreater {
        res = MakeBoolean(ctx.Position(), true)
    }
    return
}

type builtin_less struct { builtin_ }
func (ctx *builtin_less) x(ic *invocation, w facet) (res interface{}) {
    if n := len(ic.a); n != 2 {
        erro(ctx, "wrong number of arguments, try: $(less <value-list>,<value-list>)")
    } else if cmp := ic.a[0].cmp(ctx, ic.a[1]); cmp == cmpSmaller {
        res = MakeBoolean(ctx.Position(), true)
    }
    return
}

// $(match val1 val2 val3, a b c d...)
// $(match -rx=r1 -rx=r2 -rx=r3, a b c d...)
type builtin_match struct { builtin_
    regexps []*regexp.Regexp `r,re,rx,reg,regex,regexp`
    negated bool `n,ne,neg,negated,negative,not`
    all bool `a,all`
}
func (ctx *builtin_match) x(ic *invocation, w facet) (res interface{}) {
    var patList, valList []Value
    if n := len(ic.a); n < 1 {
        erro(ctx, "wrong arguments, try: $(match <regexp-list>,<value-list>,...)").debug(1)
        return
    }

    if len(ic.a) > 1 {
        patList = umerge(true, ic.a[0])
        valList = umerge(true, ic.a[1:]...)
    } else {
        valList = umerge(true, ic.a[0])
    }
    if ctx.debug > 0 {
        var ( n = len(ic.a) ; d = ctx.debug )
        warn(ctx, "match: %v %v %v, %d", ctx.regexps, patList, valList, n).debug(d)
    }

    var pos = ctx.Position()
ForValList:
    for _, val := range valList {
        if isTrivial(val) { continue ForValList }

        var str = val.strval(ctx)
        for _, rx := range ctx.regexps {
            var matched = rx.MatchString(str);
            if ctx.negated { matched = !matched }
            if matched {
                if ctx.all {
                    if res == nil { res = MakeBoolean(pos, true) }
                } else {
                    return MakeBoolean(pos, true)
                }
            } else if ctx.all {
                return nil
            }
        }
        for _, pat := range patList {
            var matched, _, _ = pat.match(ctx, str)
            if ctx.negated { matched = !matched }
            if matched {
                if ctx.all {
                    if res == nil { res = MakeBoolean(pos, true) }
                } else {
                    return MakeBoolean(pos, true)
                }
            } else if ctx.all {
                return nil
            }
        }

        if ctx.debug > 0 {
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
type builtin_case struct { builtin_ }
func (ctx *builtin_case) a(ic *invocation, w facet) (skip bool) { return }
func (ctx *builtin_case) x(ic *invocation, w facet) (res interface{}) {
    var val Value
    var args = umerge(true, ic.a...)
    if len(args) == 0 { return } else
    if _, ok := args[0].(*group); !ok {
        val = args[0].expand(ctx, w)
        args = args[1:]
    }

    var def []Value
    for _, arg := range args { if g, y := arg.(*group); y && len(g.Elems)>0 {
        if n := len(g.Elems); val != nil && isNone(val) && n == 1 {
            return g.Elems[0]
        } else if n == 1 {
            def = append(def, g.Elems[0])
            continue
        }

        var collect bool
        var v = g.Elems[0].expand(ctx, w)
        if val == nil && v.true(ctx) {
            collect = true
        } else if val != nil && isTrivial(val) {
            if isTrivial(v) {
                collect = true
            } else if f, y := v.(flag); y && isNull(f.Value) {
                collect = true
            }
        } else if val != nil && val.cmp(ctx, v) == cmpEqual {
            collect = true
        }
        if !collect { continue }

        var vals []Value
        for _, v := range g.Elems[1:] { if f, y := v.(flag); !y || isNull(f.Value) {
            vals = append(vals, v)
        }}
        return vals
    } else {
        erro(of(ctx,arg), "unexpected case: %T %v", arg, arg).debug(1)
        return
    }}
    return
}

// $(if cond, true-value, else-value, ...)
type builtin_if struct { builtin_
    def bool `def,defined`
}
func (ctx *builtin_if) a(ic *invocation, w facet) (skip bool) { // NOTE: optional
    for i, v := range ic.a { v = v.expand(ctx, w)
        if !ctx.def && w&expandTraverse == 0 && i == 0 {
            _, skip = v.(unexpanded)
        }
        ic.a[i] = v
    }
    return
}
func (ctx *builtin_if) x(ic *invocation, w facet) (res interface{}) {
    if n := len(ic.a); n > 1 {
        var t = ic.a[0]//.expand(ctx, strval)

        if !ctx.def && w&expandTraverse == 0 { if _, y := t.(unexpanded); y {
            return skip{}
        }}

        if t.true(ctx) {
            res = ic.a[1]
        } else if n > 2 {
            res = ic.a[2:]
        }
    }
    return
}

type builtin_ifeq struct { builtin_
    strval bool `s,sv,str,strval`
}
func (ctx *builtin_ifeq) a(ic *invocation, w facet) (skip bool) {
    for i, v := range ic.a { v = v.expand(ctx, w)
        if i < 2 && !skip && w&expandTraverse == 0 {
            _, skip = v.(unexpanded)
        }
        ic.a[i] = v
    }
    return
}
func (ctx *builtin_ifeq) x(ic *invocation, w facet) (res interface{}) {
    if n := len(ic.a); n > 2 {
        var (
            a = ic.a[0]//.expand(ctx, plain)
            b = ic.a[1]//.expand(ctx, expandDelegate)
        )
        if w&expandTraverse == 0 {
            if _, y := a.(unexpanded); y { return skip{} }
            if _, y := b.(unexpanded); y { return skip{} }
        }

        var equal bool
        if ctx.strval {
            equal = a.strval(ctx) == b.strval(ctx)
        } else {
            equal = a.cmp(ctx, b) == cmpEqual
        }
        if equal {
            res = ic.a[2]
        } else if n > 3 {
            res = ic.a[3:]
        }
    }
    return
}

type builtin_ifne struct { builtin_
    strval bool `s,sv,str,strval`
}
func (ctx *builtin_ifne) a(ic *invocation, w facet) (skip bool) {
    for i, v := range ic.a { v = v.expand(ctx, w)
        if i < 2 && !skip && w&expandTraverse == 0 {
            _, skip = v.(unexpanded)
        }
        ic.a[i] = v
    }
    return
}
func (ctx *builtin_ifne) x(ic *invocation, w facet) (res interface{}) {
    if n := len(ic.a); n > 2 {
        var (
            a = ic.a[0]//.expand(ctx, plain)
            b = ic.a[1]//.expand(ctx, expandDelegate)
        )
        if w&expandTraverse == 0 {
            if _, y := a.(unexpanded); y { return skip{} }
            if _, y := b.(unexpanded); y { return skip{} }
        }

        var equal bool
        if ctx.strval {
            equal = a.strval(ctx) == b.strval(ctx)
        } else {
            equal = a.cmp(ctx, b) == cmpEqual
        }
        if !equal {
            res = ic.a[2]
        } else if n > 3 {
            res = ic.a[3:]
        }
    }
    return
}

// $(for x=(a b c),$(x))
// type builtin_for struct {
//     Context
//     generalOpts
// }
// func (ctx *builtin_for) x(ic *invocation, w facet) (res interface{}) {
//     return
// }

type builtin_foreach struct { builtin_
    empty bool `empty,allow-empty`
    unique bool `u,uni,unique`
}
func (ctx *builtin_foreach) String() string {
    if true || fullContextStringer {
        return fmt.Sprintf("foreach{%s}", ctx.Context)
    } else {
        return ctx.Context.String()
    }
}
func (ctx *builtin_foreach) a1(ic *invocation, w facet) {
    w &= ^(expandPlaceholder|expandInvoke)
    for i, a := range ic.a { if i > 0 { ic.a[i] = a.expand(ctx, w) }}
}
func (ctx *builtin_foreach) a(ic *invocation, w facet) (skip bool) {
    if n := len(ic.a); n < 2 {
        errostack(ctx, 3, "insurficient arguments (%d); $(foreach <list>,<template>): %v", n, ic.a).debug(32)
        return true
    }

    // NOTE: only expand the first arg with placeholder bit
    ic.a[0] = ic.a[0].expand(ctx, w|expandPlaceholder)

    if !ctx.forth {
        if _, skip = ic.a[0].(unexpanded); skip { ctx.a1(ic, w) }
    }
    return
}
func (ctx *builtin_foreach) x(ic *invocation, w facet) (res interface{}) {
    var ( values, temps []Value ; db bool )
    if false { if w&expandDebug != 0 && ic.a != nil { defer func() {
        w.noted(ctx, ic.a[0], ic.a[1:])
        noted(ctx, "%v %v -> %v", typeof(ic.a[0]), values, res).debug(16)
    }(); db = true }}

    if values, temps = umerge(true, ic.a[0]), ic.a[1:]; len(values) == 0 {
        var d = ctx.debug ; if d < 1 { d = 1 }
        errostack(ctx, 3, "$(foreach <list>,<templates>): insurficient arguments: %v", ic.a).debug(d)
        return
    } else if ctx.unique {
        t := call(ctx, "unique", w, nil, values...)
        if !isTrivial(t) { values = merge(t) }
    }

    var list []Value
    var d = ctx.debug
    var vw = (w|expandPairVal|expandPlaceholder)&^expandPlaceholderKept
    var cc = &autoContext{ Context:ctx, defs:make(autoDefMap) }
    for _, val := range values {
        if !ctx.empty && xEmpty(ctx, val) { continue }

        cc.set(ctx, "_", val)

        var l []Value
        for _, a := range temps { v := scalarize(scalarize(a).expand(cc, vw))
            if ctx.empty || !isTrivial(v) && !isEmpty(v) { l = append(l, v) }
            if db { noted(ctx, "%T %v -> %v %v", a, a, typeof(v), v).debug(1) }
        }
        if l == nil { if ctx.empty {
            list = append(list, makeNone(ctx.Position()))
        }} else if false {
            list = append(list, ease(ctx, l))
        } else {
            list = append(list, l...)
        }
        if d>0 { warnstack(ctx, 3, "foreach: %v %v -> %v -> %v", typeof(val), val, temps, l).debug(d) }
        if db { noted(ctx, "%v %v => %v -> %v", typeof(val), val, temps, l).debug(1) }
    }
    return list
}

type builtin_count struct { builtin_
    vals []Value `v,val,value`
    // incs []string `add,inc,increase`
    // num int64 `n,num`
}
func (ctx *builtin_count) x(ic *invocation, w facet) (res interface{}) {
    var num int64
    var vals = valvec(ctx.vals)
    for _, a := range ic.a { if a.true(ctx) || vals.has2(ctx, a) {
        num += 1
    }}
    return num
}

type builtin_env struct { builtin_ }
func (ctx *builtin_env) x(ic *invocation, w facet) (res interface{}) {
    var vals []Value
    for _, a := range ic.a { if val := a.expand(ctx, expandDelegate); isTrivial(val) {
        continue
    } else if s := strings.TrimSpace(val.strval(ctx)); s != "" {
        vals = append(vals, MakeString(a.Position(), os.Getenv(s)))
    }}
    return vals
}

type builtin_auto struct { builtin_ }
func (ctx *builtin_auto) a(ic *invocation, w facet) (skip bool) {
    if n := len(ic.a); n < 2 {
        errostack(ctx, 3, "insurficient arguments (%d); $(foreach <list>,<template>): %v", n, ic.a).debug(32)
        return true
    }

    ic.a[0] = ic.a[0].expand(ctx, w)
    return
}
func (ctx *builtin_auto) x(ic *invocation, w facet) (res interface{}) {
    if len(ic.a) == 0 { return }

    var ac = &autoContext{ Context:ctx, defs:make(autoDefMap) }
    for _, a := range umerge(true, ic.a[0]) {
        if p, y := a.(*pair); y { if s := p.Key.strval(ctx); s != "" {
            ac.set(ctx, s, p.Value)
        } else { erro(ctx, "empty auto name: %T %v", p.Key, p.Key).debug(1) }}
    }

    var vals []Value
    for _, v := range ic.a[1:] { vals = append(vals, v.expand(ac, w|expandAuto)) }
    return vals
}

type builtin_var struct { builtin_ }
func (ctx *builtin_var) a(ic *invocation, w facet) (skip bool) {
    noted(ctx, "%030b %v", w, ic.a).debug(6)
    return
}
func (ctx *builtin_var) x(ic *invocation, w facet) (res interface{}) {
    noted(ctx, "%030b %v", w, ic.a).debug(6)
    return
}

// $(value <name1>,<name2>...)  -- this is specially useful when <name> is a closure.
type builtin_value struct { builtin_
    clo bool `c,clo,closure`
    unexp bool `ux,unexpand,unexpanded`
    undef bool `u,un,undef`
}
func (ctx *builtin_value) x(ic *invocation, w facet) (res interface{}) {
    var vals []Value
    if ctx.undef {
        vals = append(vals, &undef{&none{valbase{ctx.Position()}, nil}})
    }
    for _, a := range ic.a {
        var (
            closure = ctx.clo || a.expandable(ctx, expandClosure)
            name string
            val Value
        )
        if name = a.strval(ctx); name == "" {
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
        if !closure && val == nil { val = autoVal(ctx,name) }
        if d := ctx.debug; d>0 { warnstack(ctx, 3, "value: %v ; %v -> %v -> %v (closure=%v)",
            ic.a, a, name, val, closure).debug(2*d) }
        if val != nil {
            if ctx.unexp { val = unexpanded{val} }
        } else if closure {
            val = MakeClosure(ctx.Position(), LPAREN, unresolved{a, ctx.Project()}, nil)
        } else if false {
            val = makeNone(a.Position())
        } else {
            val = makeNull(a.Position())
        }
        vals = append(vals, val)
    }
    if len(vals) > 0 {
        res = ease(ctx, vals)
    }
    return
}

type builtin_call struct { builtin_
    _closure bool `c,clo,closure`
}
func (ctx *builtin_call) x(ic *invocation, w facet) (res interface{}) {
    if len(ic.a) == 0 { return }
    var obj Object
    var name = ic.a[0]
    if _, y := name.(unexpanded); y {
        return skip{}//ctx
    } else if s := name.strval(ctx); ctx._closure {
        obj = closureResolveObject(ctx, s)
    } else {
        obj = resolveObject(ctx, s)
    }
    if obj == nil {
        // var a, u, n = (w|expandAuto|expandArgs).expand(ctx, ic.a...)
        // if true { noted(ctx, "%v ⇒ %v", a, ic.a).debug(1) }
        // if u > 0 || n > 0 { ic.a = a }
        return skip{}//ctx
    }
    return invoke(ctx, obj, w, nil, ic.a[1:])
}

type builtin_closure struct { builtin_
    required bool `required,require-def,require-defs`
}
func (ctx *builtin_closure) x(ic *invocation, w facet) (res interface{}) {
    if len(ic.a) < 1 { return }

    var vals []Value
    outer: for _, val := range umerge(true, ic.a[0]) {
        var name = val.strval(ctx)
        for _, scope := range ctx.closureScopes() {
            if d := scope.FindDef(name); d != nil {
                vals = append(vals, d.invoke(ctx, w, nil, ic.a[1:]))
                continue outer
            }
        }
        if ctx.required {
            erro(of(ctx,val), "no def '%v' (%v)", name, val).debug(1)
        }
    }
    return vals
}

type builtin_defs struct { builtin_
    rxs []*regexp.Regexp `r,re,rx,reg,regex,regexp`
    not   *regexp.Regexp `nr,neg,not,ex,except,exclude`
    n int `n,num,g`
    rn int `rn`
}
func (ctx *builtin_defs) x(ic *invocation, w facet) (res interface{}) {
    var names []bare
outer:
    for name, _ := range ctx.Project().scope.elems {
        if len(ctx.rxs) == 0 {
            names = append(names, bare{name})
            if ctx.n>0 && len(names) == ctx.n {
                break
            } else {
                continue
            }
        }
        if ctx.not != nil && ctx.not.MatchString(name) {
            continue
        }
        for _, rx := range ctx.rxs {
            var sm = rx.FindStringSubmatch(name)
            if len(sm)>0 && ctx.rn<len(sm) {
                names = append(names, bare{sm[ctx.rn]})
                if ctx.n>0 && len(names) == ctx.n {
                    break outer
                } else {
                    continue outer
                }
            }
        }
    }
    if false /* && names != nil */ { noted(ctx, "%v %v", ctx.rxs, names).debug(1) }
    return names
}

type builtin_list struct { builtin_ }
func (ctx *builtin_list) x(ic *invocation, w facet) (res interface{}) {
    return ic.a
}

type builtin_plain struct { builtin_
    scope bool `findscope,find-scope,scope`
}
func (ctx *builtin_plain) c(ic *invocation, w facet) (res interface{}) {
    var scope = ctx.Scope()
    for _, a := range ic.a {
        var ( o Object ; s = a.strval(ctx) )
        if ctx.scope { _, o = scope.Find(s) } else { o = resolveObject(ctx, s) }
        if o == nil {
            erro(of(ctx,a), "no such symbol: %s", s).debug(1)
        } else if d, y := o.(*def); !y {
            erro(of(ctx,a), "not a def: %s: %v", s, typeof(o)).debug(1)
        } else if d.value != nil {
            d.value = d.value.expand(ctx, plain)
        }
    }
    return
}

type builtin_shell struct { builtin_ }
func (ctx *builtin_shell) x(ic *invocation, w facet) (res interface{}) {
    var (
        pos = ctx.Position()
        vals []Value
        err error
    )
    for _, a := range ic.a {
        var bufout, buferr bytes.Buffer
        var s = a.strval(ctx)
        sh := exec.Command("sh", "-c", s)
        sh.Stdout, sh.Stderr = &bufout, &buferr
        if err = sh.Run(); err != nil {
            s = strings.TrimSpace(buferr.String())
            if !strings.HasPrefix(s, ":") { s = ":\n" + s }
            prompt(ctx, "%s%s\n", a.strval(ctx), s)
            errostack(ctx, 3, "%s", err).debug(10)
            panic(failure{"%v",ia(ctx.Position(), err)})
            return
        }
        val := MakeString(pos, strings.TrimSpace(bufout.String()))
        vals = append(vals, val)
        bufout.Reset()
        buferr.Reset()
    }
    return vals
}

type builtin_which struct { builtin_ }
func (ctx *builtin_which) x(ic *invocation, w facet) (res interface{}) {
    var vals []Value
    for _, a := range ic.a {
        if s, err := exec.LookPath(a.strval(ctx)); err != nil {
            erro(ctx, "%v", err).debug(1)
            return
        } else if s != "" {
            vals = append(vals, MakeString(ctx.Position(), s))
        }
    }
    return vals
}

type builtin_servehttp struct { builtin_
    ssl bool `s,ss,ssl`
    host string `h,host`
    port int `p,port`
}
func (ctx *builtin_servehttp) c(ic *invocation, w facet) (res interface{}) {
    if ctx.port == 0 { ctx.port = 80 }
    if ctx.ssl {
        erro(ctx, "'serve-http(-ssl)' is unimplemented yet").debug(1)
        return
    }

    var server = http.Server{}
    server.Addr = fmt.Sprintf("%s:%d", ctx.host, ctx.port)
    info(ctx, "serving http at %v ...", server.Addr)

    var root string
    var quit = func(w http.ResponseWriter, r *http.Request) {
        var s = "<font color=red>stop serving '%s' close in a second ...</font>"
        io.WriteString(w, fmt.Sprintf(s, root))
        go func() {
            time.Sleep(1 * time.Second)
            server.Shutdown(gc.Background())
        } ()
    }

    http.HandleFunc("/-/end",  quit)
    http.HandleFunc("/-/quit", quit)
    http.HandleFunc("/-/shut", quit)

    if ic.a == nil {
        var s = ctx.WorkDir()
        http.Handle("/", http.FileServer(http.Dir(s)))
    } else {
        for _, a := range ic.a {
            var s = a.strval(ctx)
            info(ctx, "serving files %v ...", s)
            http.Handle("/", http.FileServer(http.Dir(s)))
        }
    }

    ctx.dia().flush() // flush

    var err = server.ListenAndServe()
    if err != nil && err != http.ErrServerClosed { erro(ctx, "%s", err).debug(1) }
    return
}

type builtin_append struct { builtin_
    _auto bool `a,auto`
    _closure bool `c,closure`
    // _string bool `s,str,string`
}
func (ctx *builtin_append) x(ic *invocation, w facet) (res interface{}) {
    if len(ic.a) < 2 {
        erro(ctx, "insufficient number of arguments: %v", ic.a).debug(1)
        return
    }

    var names, list []Value
    if names = merge(ic.a[0]); len(names) == 0 {
        warn(ctx, "append to nowhere: %T %v", ic.a[0], ic.a[0]).debug(1)
        return
    }
    if list = merge(ic.a[1:]...); len(list) == 0 {
        warn(ctx, "append no values: %v", ic.a[1:]).debug(1)
        return
    }

    for _, a := range names {
        var d *def
        if name := a.strval(ctx); name == "" {
            erro(of(ctx,a), "name '%v' is empty", a).debug(1)
        } else if ctx._closure { if d = closureGet(ctx, name); d == nil {
            erro(ctx, "'%s' (%v) is undefined (%T)", name, a, ctx).debug(1)
        }} else if ctx._auto { if d = autoDef(ctx, name); d == nil {
            erro(ctx, "'%s' (%v) is undefined (%T)", name, a, ctx).debug(1)
        }} else if o := resolveObject(ctx, name); o != nil { if d, _ = o.(*def); d == nil {
            erro(ctx, "'%s' (%v) is undefined (%T)", name, a, ctx).debug(1)
        }} else {
            erro(ctx, "%T %v", a, a).debug(1)
        }
        if d != nil { d.append(ctx, list...) }
    }
    return
}

type builtin_plus struct { builtin_
    int bool `i,int,integer`
}
func (ctx *builtin_plus) x(ic *invocation, w facet) (res interface{}) {
    if ctx.int {
        var num int64
        for n, a := range ic.a {
            if i, e := a.int(ctx); e == nil {
                if n == 0 { num = i } else { num += i }
            } else {
                erro(ctx, "%v: %v", a, e).debug(1)
            }
        }
        return MakeInt(ctx.Position(), num)
    } else {
        var num float64
        for n, a := range ic.a {
            if f, e := a.float(ctx); e == nil {
                if n == 0 { num = f } else { num += f }
            } else {
                erro(ctx, "%v: %v", a, e).debug(1)
            }
        }
        return MakeFloat(ctx.Position(), num)
    }
}

type builtin_minus struct { builtin_
    int bool `i,int,integer`
}
func (ctx *builtin_minus) x(ic *invocation, w facet) (res interface{}) {
    if ctx.int {
        var num int64
        for n, a := range ic.a {
            if i, e := a.int(ctx); e == nil {
                if n == 0 { num = i } else { num -= i }
            } else {
                erro(ctx, "%v: %v", a, e).debug(1)
            }
        }
        return MakeInt(ctx.Position(), num)
    } else {
        var num float64
        for n, a := range ic.a {
            if f, e := a.float(ctx); e == nil {
                if n == 0 { num = f } else { num -= f }
            } else {
                erro(ctx, "%v: %v", a, e).debug(1)
            }
        }
        return MakeFloat(ctx.Position(), num)
    }
}

type builtin_multiply struct { builtin_
    int bool `i,int,integer`
}
func (ctx *builtin_multiply) x(ic *invocation, w facet) (res interface{}) {
    if ctx.int {
        var num int64
        for n, a := range ic.a {
            if i, e := a.int(ctx); e == nil {
                if n == 0 { num = i } else { num *= i }
            } else {
                erro(ctx, "%v: %v", a, e).debug(1)
            }
        }
        return num
    } else {
        var num float64
        for n, a := range ic.a {
            if f, e := a.float(ctx); e == nil {
                if n == 0 { num = f } else { num *= f }
            } else {
                erro(ctx, "%v: %v", a, e).debug(1)
            }
        }
        return num
    }
}

type builtin_divide  struct { builtin_
    int bool `i,int,integer`
}
func (ctx *builtin_divide) x(ic *invocation, w facet) (res interface{}) {
    if ctx.int {
        var num int64
        for n, a := range ic.a {
            if i, e := a.int(ctx); e == nil {
                if n == 0 { num = i } else { num /= i } // FIXME: NaN
            } else {
                erro(ctx, "%v: %v", a, e).debug(1)
            }
        }
        return num
    } else {
        var num float64
        for n, a := range ic.a {
            if f, e := a.float(ctx); e == nil {
                if n == 0 { num = f } else { num /= f } // FIXME: NaN
            } else {
                erro(ctx, "%v: %v", a, e).debug(1)
            }
        }
        return num
    }
}

type builtin_unique struct { builtin_
    reverse bool `r,rev,reverse`
    keepAuto bool `a,auto,keepauto,keep-auto`
    unexpand bool `un,ue,unexpand,ne,noexpand,no-expand`
    plain bool `pl,pla,plain,pv,plainvalue,plain-value`
}
func (ctx *builtin_unique) x(ic *invocation, w facet) (res interface{}) {
    var args = ic.a
    if ctx.unexpand {
        args = merge(args...)
    } else if ctx.plain {
        var x = plain
        if ctx.keepAuto { x &= ^expandAuto }
        args = xmerge(ctx, x, args...)
    } else {
        var x = expandDelegate | expandPathStr | expandPairVal
        if ctx.keepAuto { x &= ^expandAuto }
        args = xmerge(ctx, x, args...)
    }

    var list []Value
ForArgs:
    for i, a := range args {
        var tmp []Value
        if ctx.reverse { tmp = args[i+1:] } else { tmp = list }
        for _, v := range tmp { if a == v || a.cmp(ctx, v) == cmpEqual {
            continue ForArgs
        }}

        if false {
            var s = a.strval(ctx)
            for _, v := range list {
                if s == v.strval(ctx) { continue ForArgs }
            }
        }
        list = append(list, a)
    }
    return list
}

type builtin_join struct { builtin_
    comp bool `comp,compose,compose-value`
}
func (ctx *builtin_join) a(ic *invocation, w facet) (skip bool) { var u int
    ic.a, u, _ = w.expand(ctx, ic.a...)
    return !ctx.comp && u > 0
}
func (ctx *builtin_join) x(ic *invocation, w facet) (res interface{}) {
    if l := len(ic.a); l > 0 {
        var (
            fields []string
            vals []Value
            sep Value
        )
        if l < 2 {
            vals = umerge(true, ic.a...)
        } else {
            vals = umerge(true, ic.a[:l-1]...)
            sep = scalarize(ic.a[l-1])
        }
        if len(vals) == 0 { return }
        if ctx.comp {
            var comp = MakeBarecomp(ctx.Position())
            for i, a := range vals {
                if i > 0 && !isTrivial(sep) { a = rearcomp{sep,a} }
                comp.Elems = append(comp.Elems, a)
            }
            res = comp
        } else {
            var s string; if sep != nil { s = sep.strval(ctx) }
            for _, a := range vals {
                if v := a.strval(ctx); v != "" { fields = append(fields, v) }
            }
            res = MakeString(ctx.Position(), strings.Join(fields, s))
        }
    }
    return
}

type builtin_quote struct { builtin_ }
func (ctx *builtin_quote) x(ic *invocation, w facet) (res interface{}) {
    var args = merge(ic.a...)
    if l := len(args); l > 0 {
        var fields []string
        for _, a := range args {
            if v := a.strval(ctx); v != "" { fields = append(fields, v) }
        }
        res = MakeString(ctx.Position(), strconv.Quote(strings.Join(fields, " ")))
    } else {
        res = makeNone(ctx.Position())
    }
    return
}

type builtin_quotejoin struct { builtin_ }
func (ctx *builtin_quotejoin) x(ic *invocation, w facet) (res interface{}) {
    var sep string
    var args = merge(ic.a...)
    if l := len(args); l > 1 {
        sep = args[l-1].strval(ctx)
        args = args[:l-1]
    }
    if l := len(args); l > 0 {
        var fields []string
        for _, a := range args[1:] {
            if v := a.strval(ctx); v != "" { fields = append(fields, v) }
        }
        res = MakeString(ctx.Position(), strconv.Quote(strings.Join(fields, sep)))
    } else {
        res = makeNone(ctx.Position())
    }
    return
}

// $(split-string .,1.2.3)
type builtin_splitstring struct { builtin_ }
func (ctx *builtin_splitstring) x(ic *invocation, w facet) (res interface{}) {
    var fields []Value
    if len(ic.a) > 0 {
        var sep = ic.a[0].strval(ctx)
        for _, a := range ic.a[1:] { for _, s := range strings.Split(a.strval(ctx), sep) {
            fields = append(fields, MakeString(a.Position(), s))
        }}
    }
    return fields
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
            if s = v.strval(ctx); s != "" { strs = append(strs, s) }
        }
        res = MakeString(value.Position(), strings.Join(strs, sep))
    }
    return
}

// TODO: deprecate this and add -quote to _builtin.SplitString
type builtin_splitquote struct { builtin_splitstring }
func (ctx *builtin_splitquote) x(ic *invocation, w facet) (res interface{}) {
    res = ctx.builtin_splitstring.x(ic, w)
    if val, y := res.(Value); y && val != nil { quotestrings(val) }
    return
}

// TODO: deprecate this and add -quote to _builtin.SplitString
type builtin_splitquotejoin struct { builtin_splitstring }
func (ctx *builtin_splitquotejoin) x(ic *invocation, w facet) (res interface{}) {
    res = ctx.builtin_splitstring.x(ic, w)
    if val, y := res.(Value); y && val != nil {
        var err error
        var sep string
        if l := len(ic.a); l > 1 {
            sep = ic.a[l-1].strval(ctx)
            ic.a = ic.a[:l-1]
        }
        if res, err = joinstrings(ctx, val, sep); err != nil {
            erro(ctx, "%v", err).debug(1)
        }
    }
    return
}

type builtin_splitjoinquote struct { builtin_splitstring }
func (ctx *builtin_splitjoinquote) x(ic *invocation, w facet) (res interface{}) {
    res = ctx.builtin_splitstring.x(ic, w)
    if val, y := res.(Value); y && val != nil {
        var err error
        var sep string
        if l := len(ic.a); l > 1 {
            sep = ic.a[l-1].strval(ctx)
            ic.a = ic.a[:l-1]
        }

        var v Value
        if v, err = joinstrings(ctx, val, sep); err != nil {
            erro(ctx, "%v", err).debug(1)
        } else {
            res = MakeString(ctx.Position(), strconv.Quote(v.strval(ctx)))
        }
    }
    return
}

type builtin_field struct { builtin_ }
func (ctx *builtin_field) x(ic *invocation, w facet) (res interface{}) {
    var fields []string
    if l := len(ic.a); l >= 2 {
        var (
            s string = ic.a[1].strval(ctx)
            i int64
        )
        if n, e := ic.a[0].int(ctx); e != nil {
            erro(ctx, "%v: %v", ic.a[0], e).debug(1)
            return
        } else { i = n }

        if l > 2 {
            fields = strings.Split(s, ic.a[2].strval(ctx))
        } else {
            fields = strings.Fields(s)
        }
        if n := int(i)-1; 0 <= n && n < len(fields) {
            s = strings.TrimSpace(fields[n])
        }
    }
    return fields
}

type builtin_fields struct { builtin_ }
func (ctx *builtin_fields) x(ic *invocation, w facet) (res interface{}) {
    // TODO: ...
    return
}

type builtin_usee struct { builtin_ }
func (ctx *builtin_usee) x(ic *invocation, w facet) (res interface{}) {
    var (
        proj = ctx.Project() //current()
        list []Value
        err error
    )
    if proj == nil {
        erro(ctx, "unknown current context").debug(1)
        return
    }

    for _, arg := range ic.a {
        var v Value
        if v, err = proj.use.Get(ctx, arg.strval(ctx)); err != nil {
            erro(ctx, "%v", err).debug(1)
            return
        } else {
            list = append(list, v)
        }
    }
    if err == nil { res = list }
    return
}

type builtin_uses struct { builtin_ }
func (ctx *builtin_uses) x(ic *invocation, w facet) (res interface{}) {
    var proj = ctx.Project() //current()
    if proj == nil {
        erro(ctx, "unknown current context").debug(1)
        return
    }

    var found bool

ForArgs:
    for _, arg := range ic.a {
        var s = arg.strval(ctx)
        for _, u := range proj.use.list {
            if found = u.project.name == s; found {
                break ForArgs
            }
        }
    }
    if found { res = found }
    return
}

type builtin_path struct { builtin_ }
func (ctx *builtin_path) x(ic *invocation, w facet) (res interface{}) {
    var list []Value
    for _, a := range ic.a {
        list = append(list, pathStr(ctx, a.Position(), a.strval(ctx)))
    }
    return list
}

type builtin_bare struct { builtin_
    name bool `n,name,file-name,non-full`
}
func (ctx *builtin_bare) x(ic *invocation, w facet) (res interface{}) {
    var vals []Value
    for _, a := range ic.a {
        var val Value
        switch t := a.(type) {
        case *String, *compound:
            val = MakeBareword(a.Position(), a.strval(ctx));
        case *File:
            val = MakeBareword(a.Position(), t.name(ctx));
        case fullfile:
            if ctx.name {
                val = MakeBareword(a.Position(), t.name(ctx));
            } else {
                val = MakeBareword(a.Position(), t.strval(ctx));
            }
        default: val = a
        }
        vals = append(vals, val)
    }
    return vals
}

type builtin_bareword struct { builtin_ }
func (ctx *builtin_bareword) x(ic *invocation, w facet) (res interface{}) {
    var vals []Value
    for _, a := range ic.a {
        var val Value
        switch a.(type) {
        case *bareword: val = a
        default: val = MakeBareword(a.Position(), a.strval(ctx));
        }
        vals = append(vals, val)
    }
    return vals
}

type builtin_str struct { builtin_
    strval bool `sv,strval`
    expand bool `x,e,ex,exp,expand`
    merge  bool `m,merge` // TODO: implement this merge opt
    name   bool `n,name,file-name,non-full`
    join []string `j,join`
    clo  []string `clo,closure`
    def  []string `def,var`
}
func (ctx *builtin_str) x(ic *invocation, w facet) (res interface{}) {
    if len(ic.a)+len(ctx.clo)+len(ctx.def) > 0 {
        var defs []*def
        for _, name := range ctx.clo {
            if o := closureResolveObject(ctx, name); o == nil { } else
            if d, y := o.(*def); y && d != nil { defs = append(defs, d) }
        }
        for _, name := range ctx.def {
            if _, o := ctx.Scope().Find(name); o == nil { } else
            if d, y := o.(*def); y && d != nil { defs = append(defs, d) }
        }

        var strs []string
        for _, d := range defs {
            var t string
            var v = d.value
            if f, y := v.(fullfile); y && ctx.name { v = f.File }
            if ctx.expand && v != nil { v = v.expand(ctx, w) }
            if v == nil { t = "<nil>" } else
            if ctx.strval { t = v.strval(ctx) } else { t = v.String() }
            if ctx.debug>0 { warn(ctx, "%T %v -> %v", d.value, d.value, t) }
            strs = append(strs, t)
        }
        for _, a := range ic.a {
            var t string
            if f, y := a.(fullfile); y && ctx.name { a = f.File }
            if ctx.expand { a = a.expand(ctx, w) }
            if ctx.strval { t = a.strval(ctx) } else { t = a.String() }
            if ctx.debug>0 { warn(ctx, "%T %v -> %v", a, a, t) }
            strs = append(strs, t)
        }

        if len(ctx.join)>0 {
            var s bytes.Buffer
            for i, t := range strs {
                if i > 0 { s.WriteString(ctx.join[i % len(ctx.join)]) }
                s.WriteString(t)
                i += 1
            }
            res = s.String()
        } else {
            var pos = ctx.Position()
            var vals []Value
            for _, t := range strs {
                vals = append(vals, MakeString(pos, t))
            }
            res = vals
        }

        if n := ctx.debug; n>0 { warnstack(ctx, n, "%T %v", res, res).debug(n) }
    }
    return
}

type builtin_string struct { builtin_str }
func (ctx *builtin_string) x(ic *invocation, w facet) (res interface{}) { ctx.strval = false
    return ctx.builtin_str.x(ic, w)
}

type builtin_strval struct { builtin_str }
func (ctx *builtin_strval) x(ic *invocation, w facet) (res interface{}) { ctx.strval = true
    return ctx.builtin_str.x(ic, w)
}

type builtin_filter struct { builtin_
    stem bool `s,stem,us,use-stem`
    neg bool
}
func (ctx *builtin_filter) do(pats []Value, values... Value) (result []Value) {
    defer func(t0 time.Time) { if d := time.Now().Sub(t0); d > 1*time.Second {
        pos := ctx.Position()
        prompt(ctx, "%v: slow: %d result, %v\n", pos, len(result), result)
        prompt(ctx, "%v: slow: %d pats, %v\n", pos, len(pats), pats)
        prompt(ctx, "%v: slow: %v\n", pos, d).debug(4)
    }} (time.Now())

    var f = func(v Value) Value {
        for _, pat := range pats { if u, y := pat.(unexpanded); false && y {
            erro(ctx, "unexpanded pattern: %v %v : %v %v", typeof(u.Value), u.Value, typeof(v), v).debug(5)
        } else if full, res, stems := pat.match(ctx, v); full {
            if ctx.neg { v = nil } else if ctx.stem {
                var vals []Value
                for _, s := range stems {
                    vals = append(vals, MakeString(v.Position(), s))
                }
                v = ease(ctx, vals)
            } else if true {
                // 'v' is just good enough
            } else if t, r := pat.stencil(ctx, stems); t != nil && len(r) == 0 {
                v = t
            } else if s, y := res.(string); y {
                v = MakeString(v.Position(), s)
            } else if a, y := res.([]string); y {
                var vals []Value
                for _, s := range a {
                    vals = append(vals, MakeString(v.Position(), s))
                }
                v = ease(ctx, vals)
            }
            return v
        }}
        if ctx.neg { return v } else { return nil }
    }

    for _, v := range values { if t := f(v); t != nil { result = append(result, t) }}
    return
}
func (ctx *builtin_filter) x(ic *invocation, w facet) (res interface{}) {
    if len(ic.a) > 1 {
        var i int
        var vals []Value
        var pats = umerge(true, ic.a[0])
        if len(pats) > 0 {
            i = 1 // good
        } else if pats = umerge(true, ic.a[1]); len(pats) == 0 {
            erro(ctx, "no patterns: %v", ic.a).debug(1)
            return
        } else {
            i = 2
        }

        if len(ic.a) <= i {
            erro(ctx, "out of index: %d %v", i, ic.a).debug(1)
            return
        }

        vals = umerge(true, ic.a[i:]...)
        vals = ctx.do(pats, vals...)
        if len(vals) > 0 { res = vals }
        if false {
            w.noted(ctx, pats[0], ic.a)
            noted(ctx, "%v %v -> %v", pats, ic.a[i:], vals).debug(10)
        }
    }
    return
}

// $(filter-out pattern…,text)
type builtin_filterout struct { builtin_filter }
func (ctx *builtin_filterout) do(pats []Value, values... Value) (result []Value) { ctx.neg = true
    return ctx.builtin_filter.do(pats, values...)
}
func (ctx *builtin_filterout) x(ic *invocation, w facet) (res interface{}) { ctx.neg = true
    return ctx.builtin_filter.x(ic, w)
}

type builtin_substring struct { builtin_ }
func (ctx *builtin_substring) x(ic *invocation, w facet) (res interface{}) {
    var list []Value
    if n := len(ic.a); n > 1 {
        var ( i1, i2 int; e error )
        if i1, e = intVal(ctx, ic.a[0], -1); e != nil {
              erro(ctx, "%v", e).debug(1)
        } else {
            ic.a = ic.a[1:]
        }
        if i2, e = intVal(ctx, ic.a[0], -1); e != nil {
            if _, ok := e.(*strconv.NumError); !ok {
                erro(of(ctx,ic.a[0]), "%v", e).debug(1)
                return
            }
        } else {
            ic.a = ic.a[1:]
        }

        if i1 < -1 && i2 < -1 {
            erro(ctx, "wrong indices (%d, %d)", i1, i2).debug(1)
            return
        } else if i1 > i2 { t := i1; i1 = i2; i2 = t } // swap the wrong order

        var a, b = int(i1), int(i2)
        if a == -1 { a = b }
        if a == -1 { return }

        for _, arg := range ic.a {
            var s = arg.strval(ctx)
            if i := len(s); i <= a { s = "" } else
            if b == -1 || i <= b { s = s[a:b] } else { s = s[a:] }
            list = append(list, MakeString(arg.Position(), s))
        }
    }
    return list
}

// $(subst from,to,text)
type builtin_subst struct { builtin_ }
func (ctx *builtin_subst) x(ic *invocation, w facet) (res interface{}) {
    var list []Value
    if nargs := len(ic.a); nargs > 2 {
        var (
            s1 = ic.a[0].strval(ctx)
            s2 = ic.a[1].strval(ctx)
        )
        for _, arg := range xmerge(ctx, expandDelegate, ic.a[2:]...) {
            var s = strings.Replace(arg.strval(ctx), s1, s2, -1)
            list = append(list, MakeString(arg.Position(), s))
        }
    }
    return list
}

// $(patsubst pattern,replacement,text)
// TODO: supports: $(var:pattern=replacement)
// TODO: supports: $(var:suffix=replacement)
// TODO: support flags -name and -full for name-only and full-name-only matching
type builtin_patsubst struct { builtin_
    findFiles bool `find,find-file`
    fullFiles bool `ff,fullfile,fullfiles`
    cleanPath bool `c,clean,cleanpath`
    noFileMap bool `nomap,no-map,nofile,nofiles,no-files,no-filemap`
    erroDstNomap bool `err-dst-nomap,error-dst-nomap`
    warnDstNomap bool `warn-dst-nomap`
}
func (ctx *builtin_patsubst) x(ic *invocation, w facet) (res interface{}) {
    if len(ic.a) < 3 {
        erro(ctx, "not enough arguments").debug(1)
        return
    }

    var (
        proj = ctx.Project()
        closured = closureProjects(ctx)
        srcPats = umerge(true, ic.a[0])
        dstPats, sources, list []Value
        t1 time.Time
    )
    defer func(t0 time.Time) { t2 := time.Now(); if d := t2.Sub(t0); d > 1*time.Second {
        var ( d1 = t1.Sub(t0) ; d2 = t2.Sub(t1) ; pos = ctx.Position() )
        prompt(ctx, "%v: slow: src %d %v\n", pos, len(srcPats), srcPats)
        prompt(ctx, "%v: slow: dst %d %v\n", pos, len(dstPats), dstPats)
        prompt(ctx, "%v: slow: sources %d %v\n", pos, len(sources), sources)
        prompt(ctx, "%v: slow: list %d %v\n", pos, len(list), list)
        prompt(ctx, "%v: slow: %v⇒%v+%v\n", pos, d, d1, d2).debug(4)
    }} (time.Now())

    if len(srcPats) == 0 {
        if len(ic.a) < 4 {
            erro(ctx, "not enough arguments").debug(1)
            return
        }
        srcPats = umerge(true, ic.a[1])
        dstPats = umerge(true, ic.a[2])
        sources = umerge(true, ic.a[3:]...)
    } else {
        dstPats = umerge(true, ic.a[1])
        sources = umerge(true, ic.a[2:]...)
    }

    t1 = time.Now()

ForSources:
    for _, src := range sources {
        var (
            source interface{} = src
            srcFile *File
            srcPat Value
            stems []string
            ok bool
        )
        if srcFile, ok = toFile(src); ok {
            source = srcFile
        } else if ctx.findFiles {
            var s = src.strval(ctx)
            if srcFile = proj.file(ctx, s); srcFile != nil {
                source = srcFile
            } else {
                source = s
            }
        } else if !ctx.fullname {
            source = src
        } else if o, y := (as{src}.fullnameOpt(ctx, closured...)); y {
            source = o.strval(ctx)
        } else {
            erro(of(ctx,src), "fullname '%v' failed", src)
            erro(ctx, "called from here", src).debug(1)
            return
        }

        var full = ctx.fullFiles
        if !full { _, full = src.(fullfile) }

        for _, srcPat = range srcPats {
            if ok, _, stems = srcPat.match(ctx, source); ok {
                goto stencilTargetPats
            }
        }

        if !isTrivial(src) { list = append(list, src) }
        continue ForSources // just append src to the list

        // Compose the matched results with stem value.
    stencilTargetPats:
        for _, dstPat := range dstPats {
            var nameVal, ramnant = dstPat.stencil(ctx, stems)
            if isNull(nameVal) {
                erro(ctx, "nil stencil: %T %v (stems=%v, ramnant=%v)", dstPat, dstPat, stems, ramnant).debug(1)
                return
            } else if ctx.debug>0 {
                warnstack(ctx, ctx.debug, "patsubst: %v: %v -> %v -> %v %v -> %v %v",
                    srcPat, src, source, stems, dstPat, nameVal, ramnant).debug(ctx.debug)
            }

            var nameStr string
            if nameStr = nameVal.strval(ctx); nameStr == "" {
                continue stencilTargetPats
            } else if ctx.cleanPath {
                nameStr = filepath.Clean(nameStr)
            }

            if srcFile != nil {
                var dstFile *File
                if !ctx.noFileMap { dstFile = proj.file(ctx, nameStr) }
                if dstFile == nil {
                    a := []interface{}{
                        "%v: %v (%v): unmapped destination, aka files (...)",
                        proj, nameStr, dstPat,
                    }
                    if t := files(ctx, nameVal, proj); ctx.erroDstNomap {
                        erro(of(ctx,srcPat), "%v: patsubst: %v (%v) ⇒ %v (%v) ⇒ %v", proj, srcFile, srcPat, nameVal, dstPat, t)
                        errostack(of(ctx,dstPat), 16, a...).debug(16)
                    } else if ctx.warnDstNomap {
                        warn(of(ctx,srcPat), "%v: patsubst: %v (%v) ⇒ %v (%v) ⇒ %v", proj, srcFile, srcPat, nameVal, dstPat, t)
                        warnstack(of(ctx,dstPat), 16, a...).debug(5)
                    }
                    dstFile = stat(ctx, nameStr, srcFile.sub, srcFile.dir, nil)
                }
                if dstFile.position = srcPat.Position(); full {
                    list = append(list, fullfile{dstFile})
                } else {
                    list = append(list, dstFile)
                }
                continue stencilTargetPats
            }

            // Deal with source value types
            switch pos := dstPat.Position(); src.(type) {
            case *File, fullfile:
            case *String, *compound:
                list = append(list, MakeString(pos, nameStr))
                continue stencilTargetPats
            case *Path:
                list = append(list, pathStr(ctx, pos, nameStr))
                continue stencilTargetPats
            case *bareword, *barecomp:
                if strings.Contains(nameStr, PathSep) {
                    list = append(list, pathStr(ctx, pos, nameStr))
                } else {
                    list = append(list, MakeBareword(pos, nameStr))
                }
                continue stencilTargetPats
            default:
                if strings.Contains(nameStr, PathSep) {
                    list = append(list, pathStr(ctx, pos, nameStr))
                } else if true {
                    list = append(list, MakeBareword(pos, nameStr))
                } else {
                    list = append(list, MakeString(pos, nameStr))
                }
                continue stencilTargetPats
            }
        }
    }

    if ctx.debug>0 && len(list) == 0 {
        warn(ctx, "src: %v", srcPats)
        warn(ctx, "dst: %v", dstPats)
        warn(ctx, "val: %v", sources)
        warn(ctx, "res: %v", list)
        warnstack(ctx, 3, "").debug(ctx.debug)
    }
    return list
}

type builtin_title struct { builtin_ }
func (ctx *builtin_title) x(ic *invocation, w facet) (res interface{}) {
    var list []Value
    for _, a := range ic.a { if s := a.strval(ctx); s != "" {
        list = append(list, MakeString(a.Position(), strings.Title(s)))
    }}
    return list
}

type builtin_uppercase struct { builtin_ }
func (ctx *builtin_uppercase) x(ic *invocation, w facet) (res interface{}) {
    var list []Value
    for _, a := range ic.a { if s := a.strval(ctx); s != "" {
        list = append(list, MakeString(a.Position(), strings.ToUpper(s)))
    }}
    return list
}

type builtin_lowercase struct { builtin_ }
func (ctx *builtin_lowercase) x(ic *invocation, w facet) (res interface{}) {
    var list []Value
    for _, a := range ic.a { if s := a.strval(ctx); s != "" {
        list = append(list, MakeString(a.Position(), strings.ToLower(s)))
    }}
    return list
}

type builtin_strip struct { builtin_trimspace }
func (ctx *builtin_strip) x(ic *invocation, w facet) (res interface{}) {
    return ctx.builtin_trimspace.x(ic, w)
}

type builtin_trimspace struct { builtin_trim }
func (ctx *builtin_trimspace) a(ic *invocation, w facet) (skip bool) {
    a, _, _ := w.expand(ctx, ic.a...)
    ic.a = append([]Value{makeNone(ctx.Position())}, a...)
    return
}
func (ctx *builtin_trimspace) x(ic *invocation, w facet) (res interface{}) {
    return ctx.builtin_trim.x(ic, w)
}

type builtin_trim struct { builtin_ }
func (ctx *builtin_trim) x(ic *invocation, w facet) (res interface{}) {
    var cutset string
    var list []Value
    for i, a := range ic.a { if s := a.strval(ctx); s != "" { if i == 0 {
        cutset = s
    } else if cutset == "" {
        list = append(list, MakeString(a.Position(), strings.TrimSpace(s)))
    } else {
        list = append(list, MakeString(a.Position(), strings.Trim(s, cutset)))
    }}}
    return list
}

type builtin_trimleft struct { builtin_ }
func (ctx *builtin_trimleft) x(ic *invocation, w facet) (res interface{}) {
    var (
        cutset string
        list []Value
    )
    for i, a := range ic.a { if s := a.strval(ctx); s != "" { if i == 0 {
        cutset = s
    } else if cutset == "" {
        list = append(list, MakeString(a.Position(), strings.TrimLeftFunc(s, unicode.IsSpace)))
    } else {
        list = append(list, MakeString(a.Position(), strings.TrimLeft(s, cutset)))
    }}}
    return list
}

type builtin_trimright struct { builtin_ }
func (ctx *builtin_trimright) x(ic *invocation, w facet) (res interface{}) {
    var (
        cutset string
        list []Value
    )
    for i, a := range ic.a { if s := a.strval(ctx); s != "" { if i == 0 {
        cutset = s
    } else if cutset == "" {
        list = append(list, MakeString(a.Position(), strings.TrimRightFunc(s, unicode.IsSpace)))
    } else {
        list = append(list, MakeString(a.Position(), strings.TrimRight(s, cutset)))
    }}}
    return list
}

// $(trim-prefix foo%, fooxxx foo123)
// $(trim-prefix %/foo, xxx/foo/a/b/c)
// $(trim-prefix %%/foo, xxx/yyy/zzz/foo/a/b/c)
type builtin_trimprefix struct { builtin_ }
func (ctx *builtin_trimprefix) x(ic *invocation, w facet) (res interface{}) {
    if false { if ctx.verbose = ctx.Project().name == "testllvmconfig"; ctx.verbose { defer func() {
        noted(ctx, "%v: %v %v", ctx.Project(), ic.a, res).debug(2)
    }()}}

    if len(ic.a) == 0 { return }

    var values, list []Value
    var prefixs = merge(ic.a[0])
    if len(ic.a) == 1 {
        if len(prefixs) > 1 { values = prefixs[1:] }
    } else {
        values = umerge(true, ic.a[1:]...)
    }

    if len(values) == 0 { return }
    if len(prefixs) == 0 { return ease(ctx, values) }
    if ctx.verbose {
        warn(ctx, "prefix = %v", prefixs)
        warn(ctx, "values = %v", values)
    }

ForValues:
    for _, value := range values { var s string
        if s = value.strval(ctx); s == "" { continue }

        var pos = value.Position()

    ForPrefix:
        for _, prefix := range prefixs { var p string
            if p = prefix.strval(ctx); p == "" { continue }

            // FIXME: matched cutset is empty: %-xxx- and *-xxx-
            var full, r, stems = prefix.match(ctx, value)
            var cutset = joinMatchRes(ctx, r)
            if ctx.verbose {
                warn(ctx, "full   = %v", full)
                warn(ctx, "prefix = %v (%v)", prefix, typeof(prefix))
                warn(ctx, "value  = %v (%v)", value, typeof(value))
                warn(ctx, "cutset = %v", cutset)
                warn(ctx, "stems  = %v", stems)
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
    return list
}

type builtin_trimsuffix struct { builtin_ }
func (ctx *builtin_trimsuffix) x(ic *invocation, w facet) (res interface{}) {
    var (
        cutset, s string
        list []Value
    )
    for i, a := range ic.a { if s = a.strval(ctx); s != "" { if i == 0 {
        cutset = s
    } else if cutset == "" {
        list = append(list, MakeString(a.Position(), strings.TrimRightFunc(s, unicode.IsSpace)))
    } else {
        list = append(list, MakeString(a.Position(), strings.TrimSuffix(s, cutset)))
    }}}
    return list
}

type builtin_trimext struct { builtin_
    all bool `a,all`
    ext []string `e,ext`
}
func (ctx *builtin_trimext) x(ic *invocation, w facet) (res interface{}) {
    var ext string
    var list []Value
    for i, a := range ic.a { if s := a.strval(ctx); s != "" {
        if i == 0 && len(ic.a) > 1 {
            ext = s
        } else if ext == "" {
            for ext = filepath.Ext(s); ext != ""; {
                s = strings.TrimSuffix(s, ext)
                if ctx.all { ext = filepath.Ext(s) } else { break }
            }
            list = append(list, MakeString(a.Position(), s))
        } else if ext == filepath.Ext(s) {
            list = append(list, MakeString(a.Position(), strings.TrimRight(s, ext)))
        }
    }}
    return list
}

type builtin_addprefix struct { builtin_ }
func (ctx *builtin_addprefix) a(ic *invocation, w facet) (skip bool) {
    ic.a, _, _ = w.expand(ctx, ic.a...) // NOTE: discard unexpanded-number
    return
}
func (ctx *builtin_addprefix) x(ic *invocation, w facet) (res interface{}) {
    if len(ic.a) < 1 {
        erro(ctx, "not enough args, try $(addprefix 'prefix', ...)").debug(1)
        return
    }

    var prefixs = umerge(true, ic.a[0])
    if len(prefixs) != 1 {
        erro(ctx, "not enough args, try $(addprefix 'prefix', ...)").debug(1)
        return
    }

    var list []Value
    var vals = umerge(true, ic.a[1:]...)
    var tw = expandClosure|expandDelegate
    for _, prefix := range prefixs { if isTrivial(prefix) { continue }; p, y := prefix.(*pair)
        for _, val := range vals { if isTrivial(val) { continue }; pos := val.Position()
            if y && !isTrivial(p.Value) {
                c := MakeBarecomp(pos, p.Value)
                c.comp(ctx, val)
                val = c
            }
            if a, b := y && p.Key.expandable(ctx, tw), val.expandable(ctx, tw); a||b {
                if b { val = unexpanded{val}}
                if y { key := p.Key
                    if a { key = unexpanded{key}}
                    val = paircomp{MakePair(pos, key, val)}
                } else {
                    val = precomp{prefix, val}
                }
            } else if y {
                val = MakePair(pos, p.Key, val)
            } else {
                c := MakeBarecomp(pos, prefix)
                c.comp(ctx, val)
                val = c
            }
            list = append(list, val)
        }
    }
    return list
}

type builtin_addsuffix struct { builtin_ }
func (ctx *builtin_addsuffix) a(ic *invocation, w facet) (skip bool) {
    ic.a, _, _ = w.expand(ctx, ic.a...) // NOTE: discard unexpanded-number
    return
}
func (ctx *builtin_addsuffix) x(ic *invocation, w facet) (res interface{}) {
    if len(ic.a) < 1 {
        erro(ctx, "not enough args, try $(addsuffix 'suffix', ...)").debug(1)
        return
    }

    var suffixs = merge(ic.a[0])
    if len(suffixs) != 1 {
        erro(ctx, "not enough args, try $(addsuffix 'suffix', ...)").debug(1)
        return
    }

    var list []Value
    var vals = xmerge(ctx.Context, plain, ic.a[1:]...)

    for _, suffix := range suffixs {
        if !suffix.true(ctx) { continue }
        for _, val := range vals {
            if /* false && !val.true(ctx) */isTrivial(val) {
                continue
            }
            var pos = val.Position()
            var p, y = val.(*pair)
            if y && !isTrivial(p.Value) {
                val = MakeBarecomp(p.Key.Position(), val, p.Key)
            }
            if val.expandable(ctx, expandDelegate|expandClosure) {
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

    res = ease(ctx, list)
    return
}

type builtin_printf struct{ builtin_ }
func (ctx *builtin_printf) c(ic *invocation, w facet) (res interface{}) { return ctx.x(ic, w) }
func (ctx *builtin_printf) x(ic *invocation, w facet) (res interface{}) {
    if len(ic.a) < 1 {
        erro(ctx, "not enough args, try $(printf 'format', ...)").debug(1)
        return
    }

    var f string
    var vals = merge(ic.a[0])
    if len(vals) != 1 {
        erro(ctx, "not enough args, try $(printf 'format', ...)").debug(1)
        return
    } else {
        f = vals[0].strval(ctx)
    }

    var i int
    var a []interface{}
ForArgs:
    for _, v := range merge(ic.a[1:]...) {
    ForFmt:
        for i < len(f) {
            if f[i] != '%' { i += 1; continue }
            for i += 1; i < len(f); i += 1 {
                switch f[i] {
                case '%': continue ForFmt
                case '+', '-', '#', ' ', '.', '0', '1', '2', '3',
                    '4', '5', '6', '7', '8', '9': continue
                case 'c', 'd', 'o', 'O', 'q', 'U':
                    if t, e := v.int(ctx); e == nil { a = append(a, t) } else {
                        erro(ctx, "%v: %v", v, e).debug(1)
                    }
                    continue ForArgs
                case 'e', 'E', 'f', 'F', 'g', 'G':
                    if t, e := v.float(ctx); e == nil { a = append(a, t) } else {
                        erro(ctx, "%v: %v", v, e).debug(1)
                    }
                    continue ForArgs
                case 'b', 'x', 'X':
                    switch k := v.kind(); {
                    case k&KindInteger != 0:
                        if t, e := v.int(ctx); e == nil { a = append(a, t) } else {
                            erro(ctx, "%v: %v", v, e).debug(1)
                        }
                        continue ForArgs
                    case k&KindFloat != 0:
                        if t, e := v.float(ctx); e == nil { a = append(a, t) } else {
                            erro(ctx, "%v: %v", v, e).debug(1)
                        }
                        continue ForArgs
                    default:
                        if t, e := strconv.Atoi(v.strval(ctx)) ; e == nil { a = append(a, t) } else {
                            erro(ctx, "%v: %v", v, e).debug(1)
                        }
                        continue ForArgs
                    }
                case 'v':
                    a = append(a, v/* .strval(ctx) */)
                    continue ForArgs
                case 't', 'T':
                    a = append(a, v)
                    continue ForArgs
                }
            }
        }
    }
    return fmt.Sprintf(f, a...)
}

type builtin_print struct{ builtin_
    noErrs bool `noerrs,noerrors,no-errs,no-errors`
    noWarn bool `nowarn,nowarns,no-warn,no-warns`
}
func (ctx *builtin_print) c(ic *invocation, w facet) (res interface{}) { return ctx.x(ic, w) }
func (ctx *builtin_print) x(ic *invocation, w facet) (res interface{}) {
    var diag = ctx.dia()
    if ctx.noErrs && diag.count(diagError) > 0 { return }
    if ctx.noWarn && diag.count(diagWarn) > 0 { return }

    var (
        x = len(ic.a)
        sb bytes.Buffer
    )
    for i, a := range ic.a {
        if a == nil { continue } else
        if 0 < i && i < x { fmt.Fprintf(&sb, " ") }
        fmt.Fprintf(&sb, "%s", EscapedString(ctx, a))
    }
    prompt(ctx, sb.String())
    return
}

type builtin_printl struct{ builtin_
    noErrs bool `noerrs,noerrors,no-errs,no-errors`
    noWarn bool `nowarn,nowarns,no-warn,no-warns`
}
func (ctx *builtin_printl) c(ic *invocation, w facet) (res interface{}) { return ctx.x(ic, w) }
func (ctx *builtin_printl) x(ic *invocation, w facet) (res interface{}) {
    var diag = ctx.dia()
    if ctx.noErrs && diag.count(diagError) > 0 { return }
    if ctx.noWarn && diag.count(diagWarn) > 0 { return }

    var (
        x = len(ic.a)
        sb bytes.Buffer
    )
    for i, a := range ic.a {
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

type builtin_println struct{ builtin_
    noErrs bool `noerrs,noerrors,no-errs,no-errors`
    noWarn bool `nowarn,nowarns,no-warn,no-warns`
}
func (ctx *builtin_println) c(ic *invocation, w facet) (res interface{}) { return ctx.x(ic, w) }
func (ctx *builtin_println) x(ic *invocation, w facet) (res interface{}) {
    var dia = ctx.dia()
    if ctx.noErrs && dia.count(diagError) > 0 { return }
    if ctx.noWarn && dia.count(diagWarn) > 0 { return }

    var (
        x = len(ic.a)
        sb bytes.Buffer
    )
    for i, a := range ic.a {
        if a == nil { continue } else
        if 0 < i && i < x { fmt.Fprintf(&sb, " ") }
        fmt.Fprintf(&sb, "%s", EscapedString(ctx, a))
    }
    fmt.Fprintf(&sb, "\n")
    prompt(ctx, sb.String())
    return
}

type builtin_indent struct { builtin_ }
func (ctx *builtin_indent) x(ic *invocation, w facet) (res interface{}) {
    var (
        l []Value
        s string // indent
    )
    if x := len(ic.a); x > 0 {
        if v, ok := scalarize(ic.a[0]).(*Int); ok {
            ic.a, s = ic.a[1:], strings.Repeat(" ", int(v.int64))
        } else {
            erro(ctx, "requires integer argument (first|last)").debug(1)
            return
        }
    }
    for _, a := range ic.a {
        var lines []string
        for _, line := range strings.Split(a.strval(ctx), "\n") {
            lines = append(lines, s + line)
        }
        l = append(l, MakeString(a.Position(), strings.Join(lines, "\n")))
    }
    return l
}

type builtin_findstring struct { builtin_ }
func (ctx *builtin_findstring) x(ic *invocation, w facet) (res interface{}) {
    // TODO: $(findstring find,text)
    return
}

// $(contains a b c, v1 v2 …)
// $(contains a b c1 -or c2, v1 v2 …)      -- xx
// $(contains a b c1 -or c2 -or c3, v1 v2 …)   -- xx
// $(contains a b -or=(c1 c2 c3), v1 v2 …)     -- xx
type builtin_contains struct { builtin_
    match  bool `m,mat,match,p,pat,pattern`
    string bool `s,str,string`
}
func (ctx *builtin_contains) a(ic *invocation, w facet) (skip bool) { return }
func (ctx *builtin_contains) x(ic *invocation, w facet) (res interface{}) {
    if len(ic.a) < 2 {
        erro(ctx, "unexpected number of arguments, try $(contains a b c1 c2, v1 v2 …)").debug(1)
        return
    }

    w |= expandPairVal|expandUnexpandedMerge

    var vals = xmerge(ctx, w, ic.a[0])
    var list = xmerge(ctx, w, ic.a[1:]...)
    if len(vals) == 0 || len(list) == 0 {
        erro(ctx, "insufficient number of arguments: %v ⇒ %v %v", ic.a, vals, list).debug(6)
        return
    }

    var ddd = ctx.universe().ddd == "contains"
    var n int
outer:
    for i, val := range vals { var s string ; if ctx.string { s = val.strval(ctx) }
        for j, elem := range list {
            if ctx.string { if elem.strval(ctx) == s {
                n += 1; continue outer
            }} else if ctx.match { if t, _, _ := val.match(ctx, elem); t {
                n += 1; continue outer
            }} else if val.cmp(ctx, elem) == cmpEqual {
                n += 1; continue outer
            }
            if ctx.debug>0 && ddd { warn(of(ctx,val), "%d. %T %v <-> %d. %T %v", i, val, val, j, elem, elem) }
            if ctx.debug>0 && !ctx.string && elem != nil { if a, b := val.strval(ctx), elem.strval(ctx); a == b {
                warn(of(ctx,val), "wrong: %T %v <-> %T %v ; '%s', '%s'", val, val, elem, elem, a, b)
            }}
        }
    }

    var y = (n == len(vals))
    if ctx.debug>0 && !y { warn(ctx, "found %d/%d: %v ; %v", n, len(vals), list, ic.a).debug(ctx.debug) }
    return MakeBoolean(ctx.Position(), y)
}

type builtin_sort struct { builtin_ }
func (ctx builtin_sort) x(ic *invocation, w facet) (res interface{}) {
    // TODO: $(sort list)
    return
}

type builtin_word struct { builtin_ }
func (ctx builtin_word) x(ic *invocation, w facet) (res interface{}) {
    // TODO: $(word n,text)
    return
}

type builtin_wordlist struct { builtin_ }
func (ctx builtin_wordlist) x(ic *invocation, w facet) (res interface{}) {
    // TODO: $(wordlist s,e,text)
    return
}

type builtin_words struct { builtin_ }
func (ctx builtin_words) x(ic *invocation, w facet) (res interface{}) {
    // TODO: $(words n,text)
    return
}

type builtin_firstword struct { builtin_ }
func (ctx builtin_firstword) x(ic *invocation, w facet) (res interface{}) {
    // TODO: $(firstword names...)
    return
}

type builtin_lastword struct { builtin_ }
func (ctx builtin_lastword) x(ic *invocation, w facet) (res interface{}) {
    // TODO: $(lastword names...)
    return
}

type builtin_encodebase64 struct { builtin_ }
func (ctx *builtin_encodebase64) x(ic *invocation, w facet) (res interface{}) {
    if len(ic.a) > 0 {
        pos := ctx.Position()
        buf := new(bytes.Buffer)
        enc := base64.NewEncoder(base64.StdEncoding, buf)
        for _, a := range ic.a { enc.Write([]byte(a.strval(ctx))) }
        enc.Close()
        res = MakeString(pos, buf.String())
    }
    return
}

type builtin_decodebase64 struct { builtin_ }
func (ctx *builtin_decodebase64) x(ic *invocation, w facet) (res interface{}) {
    if len(ic.a) > 0 {
        var list []Value
        for _, a := range ic.a {
            var s string = a.strval(ctx)
            if dat, err := base64.StdEncoding.DecodeString(s); err != nil {
                erro(ctx, "decode '%s' failed: %v", s, err).debug(1)
                return
            } else {
                list = append(list, MakeString(a.Position(), string(dat)))
            }
        }
        res = ease(ctx, list)
    }
    return
}

type builtin_fullname struct { builtin_ }
func (ctx *builtin_fullname) x(ic *invocation, w facet) (res interface{}) {
    var ( l []Value ; p = /* closureProjects(ctx) */[]*Project{ctx.Project()} )
    for _, a := range umerge(true, ic.a...) {
        if ctx.debug > 0 { if f, y := toFile(a); y {
            warn(ctx, "dir=%v sub=%v name=%v", f.dir, f.sub, f.name).debug(ctx.debug)
        } else {
            warn(ctx, "%T %v", a, a).debug(ctx.debug,1)
        }}

        if o, y := (as{a}.fullnameOpt(ctx, p...)); y { a = o }
        l = append(l, a)
    }
    return l
}

type builtin_ext struct { builtin_ }
func (ctx *builtin_ext) x(ic *invocation, w facet) (res interface{}) {
    var list []Value
    for _, a := range ic.a {
        list = append(list, MakeString(a.Position(), filepath.Ext(a.strval(ctx))))
    }
    return list
}

type builtin_bases struct { builtin_
    n int `n,num,size,count`
}
func (ctx *builtin_bases) x(ic *invocation, w facet) (res interface{}) {
    var l []Value
    for _, a := range ic.a {
        var s string
        if ctx.fullname {
            s, _ = as{a}.fullnameOrStrval(ctx)
        } else {
            s = a.strval(ctx)
        }

        d := filepath.Dir(s)
        s  = filepath.Base(s)
        for i := ctx.n-1; 0 < i; i -= 1 {
            s = filepath.Join(filepath.Base(d), s)
            d = filepath.Dir(d)
        }
        l = append(l, MakeString(a.Position(), s))
    }
    return l
}

type builtin_base struct { builtin_bases }
func (ctx *builtin_base) x(ic *invocation, w facet) (res interface{}) { ctx.n = 1
    return ctx.builtin_bases.x(ic, w)
}

type builtin_base2 struct { builtin_bases }
func (ctx *builtin_base2) x(ic *invocation, w facet) (res interface{}) { ctx.n = 2
    return ctx.builtin_bases.x(ic, w)
}

type builtin_base3 struct { builtin_bases }
func (ctx *builtin_base3) x(ic *invocation, w facet) (res interface{}) { ctx.n = 3
    return ctx.builtin_bases.x(ic, w)
}

type builtin_base4 struct { builtin_bases }
func (ctx *builtin_base4) x(ic *invocation, w facet) (res interface{}) { ctx.n = 4
    return ctx.builtin_bases.x(ic, w)
}

type builtin_base5 struct { builtin_bases }
func (ctx *builtin_base5) x(ic *invocation, w facet) (res interface{}) { ctx.n = 5
    return ctx.builtin_bases.x(ic, w)
}

type builtin_base6 struct { builtin_bases }
func (ctx *builtin_base6) x(ic *invocation, w facet) (res interface{}) { ctx.n = 6
    return ctx.builtin_bases.x(ic, w)
}

type builtin_base7 struct { builtin_bases }
func (ctx *builtin_base7) x(ic *invocation, w facet) (res interface{}) { ctx.n = 7
    return ctx.builtin_bases.x(ic, w)
}

type builtin_base8 struct { builtin_bases }
func (ctx *builtin_base8) x(ic *invocation, w facet) (res interface{}) { ctx.n = 8
    return ctx.builtin_bases.x(ic, w)
}

type builtin_base9 struct { builtin_bases }
func (ctx *builtin_base9) x(ic *invocation, w facet) (res interface{}) { ctx.n = 9
    return ctx.builtin_bases.x(ic, w)
}

type builtin_dirs struct { builtin_
    n int `n,num,size,count`
}
func (ctx *builtin_dirs) x(ic *invocation, w facet) (res interface{}) {
    var l []Value
    for _, a := range merge(ic.a...) {
        var s string
        if ctx.fullname {
            s, _ = as{a}.fullnameOrStrval(ctx)
        } else {
            s = a.strval(ctx)
        }
        s = filepath.Dir(s)
        for i := ctx.n-1; 0 < i; i -= 1 { s = filepath.Dir(s) }

        var v Value
        var d = ctx.debug
        if f, y := a.(*File); y {
            if ctx.fullname {
                f = stat(ctx, s, "", "", nil)
            } else {
                f = stat(ctx, s, f.sub, f.dir, nil)
            }
            if d>0 { noted(ctx, "%T %v ⇒ %v %v", a, a, f, f.fullname()).debug(d) }
            v = f
        } else if s != "" {
            if d>0 { noted(ctx, "%T %v ⇒ %v", a, a, s).debug(d) }
            v = pathStr(ctx, a.Position(), s)
        } else {
            continue
        }
        l = append(l, v)
    }
    return l
}

type builtin_dir struct { builtin_dirs }
func (ctx *builtin_dir) x(ic *invocation, w facet) (res interface{}) { ctx.n = 1
    return ctx.builtin_dirs.x(ic, w)
}

type builtin_dir2 struct { builtin_dirs }
func (ctx *builtin_dir2) x(ic *invocation, w facet) (res interface{}) { ctx.n = 2
    return ctx.builtin_dirs.x(ic, w)
}

type builtin_dir3 struct { builtin_dirs }
func (ctx *builtin_dir3) x(ic *invocation, w facet) (res interface{}) { ctx.n = 3
    return ctx.builtin_dirs.x(ic, w)
}

type builtin_dir4 struct { builtin_dirs }
func (ctx *builtin_dir4) x(ic *invocation, w facet) (res interface{}) { ctx.n = 4
    return ctx.builtin_dirs.x(ic, w)
}

type builtin_dir5 struct { builtin_dirs }
func (ctx *builtin_dir5) x(ic *invocation, w facet) (res interface{}) { ctx.n = 5
    return ctx.builtin_dirs.x(ic, w)
}

type builtin_dir6 struct { builtin_dirs }
func (ctx *builtin_dir6) x(ic *invocation, w facet) (res interface{}) { ctx.n = 6
    return ctx.builtin_dirs.x(ic, w)
}

type builtin_dir7 struct { builtin_dirs }
func (ctx *builtin_dir7) x(ic *invocation, w facet) (res interface{}) { ctx.n = 7
    return ctx.builtin_dirs.x(ic, w)
}

type builtin_dir8 struct { builtin_dirs }
func (ctx *builtin_dir8) x(ic *invocation, w facet) (res interface{}) { ctx.n = 8
    return ctx.builtin_dirs.x(ic, w)
}

type builtin_dir9 struct { builtin_dirs }
func (ctx *builtin_dir9) x(ic *invocation, w facet) (res interface{}) { ctx.n = 9
    return ctx.builtin_dirs.x(ic, w)
}

type builtin_undirs struct { builtin_
    n int `n,num,size,count`
}
func (ctx *builtin_undirs) x(ic *invocation, w facet) (res interface{}) {
    var l []Value
    for _, a := range ic.a {
        var s string
        if ctx.fullname {
            s, _ = as{a}.fullnameOrStrval(ctx)
        } else {
            s = a.strval(ctx)
        }
        var v = strings.Split(s, PathSep)
        if i := len(v); i == 0 {
            // v is empty
        } else if ctx.n < i {
            v = v[ctx.n:]
        } else {
            v = v[i-1:] // empty
        }
        l = append(l, pathStr(ctx, a.Position(), filepath.Join(v...)))
    }
    return l
}

type builtin_undir struct { builtin_undirs }
func (ctx *builtin_undir) x(ic *invocation, w facet) (res interface{}) { ctx.n = 1
    return ctx.builtin_undirs.x(ic, w)
}

type builtin_undir2 struct { builtin_undirs }
func (ctx *builtin_undir2) x(ic *invocation, w facet) (res interface{}) { ctx.n = 2
    return ctx.builtin_undirs.x(ic, w)
}

type builtin_undir3 struct { builtin_undirs }
func (ctx *builtin_undir3) x(ic *invocation, w facet) (res interface{}) { ctx.n = 3
    return ctx.builtin_undirs.x(ic, w)
}

type builtin_undir4 struct { builtin_undirs }
func (ctx *builtin_undir4) x(ic *invocation, w facet) (res interface{}) { ctx.n = 4
    return ctx.builtin_undirs.x(ic, w)
}

type builtin_undir5 struct { builtin_undirs }
func (ctx *builtin_undir5) x(ic *invocation, w facet) (res interface{}) { ctx.n = 5
    return ctx.builtin_undirs.x(ic, w)
}

type builtin_undir6 struct { builtin_undirs }
func (ctx *builtin_undir6) x(ic *invocation, w facet) (res interface{}) { ctx.n = 6
    return ctx.builtin_undirs.x(ic, w)
}

type builtin_undir7 struct { builtin_undirs }
func (ctx *builtin_undir7) x(ic *invocation, w facet) (res interface{}) { ctx.n = 7
    return ctx.builtin_undirs.x(ic, w)
}

type builtin_undir8 struct { builtin_undirs }
func (ctx *builtin_undir8) x(ic *invocation, w facet) (res interface{}) { ctx.n = 8
    return ctx.builtin_undirs.x(ic, w)
}

type builtin_undir9 struct { builtin_undirs }
func (ctx *builtin_undir9) x(ic *invocation, w facet) (res interface{}) { ctx.n = 9
    return ctx.builtin_undirs.x(ic, w)
}

type builtin_chopdir struct { builtin_ }
func (ctx *builtin_chopdir) x(ic *invocation, w facet) (res interface{}) {
    var l []Value
    var n = 0
    if x := len(ic.a); x > 0 {
        if v, ok := scalarize(ic.a[0]).(*Int); ok {
            ic.a, n = ic.a[1:], int(v.int64)
        } else if v, ok := scalarize(ic.a[x-1]).(*Int); ok {
            ic.a, n = ic.a[:x-1], int(v.int64)
        } else {
            erro(ctx, "require (first/last) integer argument (first=%T, last=%T)", ic.a[0], ic.a[x-1]).debug(1)
            return

        }
    }
    for _, a := range ic.a {
        var v = strings.Split(a.strval(ctx), PathSep)
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
    return l
}

type builtin_reldir struct { builtin_ }
func (ctx *builtin_reldir) x(ic *invocation, w facet) (res interface{}) {
    var (
        err error
        l []Value
        t string
    )
    for i, a := range ic.a {
        if s := a.strval(ctx); i == 0 {
            t = s
        } else if s, err = filepath.Rel(t, s); err == nil {
            l = append(l, MakeString(a.Position(), s))
        } else {
            erro(ctx, "%v", err)
            return
        }
    }
    return l
}

type builtin_mkdir struct { builtin_
    all bool `a,all,p,path`
}
func (ctx *builtin_mkdir) c(ic *invocation, w facet) (res interface{}) {
    for i, nargs := 0, len(ic.a); i < nargs; i += 1 {
        var (
            a = ic.a[i]
            perm = os.FileMode(0755)
            name string
        )
        switch t := a.(type) {
        case *pair: // mkdir name => perm name => perm
            name = t.Key.strval(ctx)
            perm = permVal(ctx, t.Value, uint32(perm))
        case *group: // mkdir (name perm) (name perm)
            if t.Len() == 2 {
                name = t.Get(0).strval(ctx)
                perm = permVal(ctx, t.Get(1), uint32(perm))
            } else {
                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                break
            }
        case *List: // mkdir name perm, name perm, ...
            if t.Len() == 2 {
                name = t.Get(0).strval(ctx)
                perm = permVal(ctx, t.Get(1), uint32(perm))
            } else {
                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                break
            }
        default: // mkdir name perm, name perm, ...
            name = ic.a[i].strval(ctx)
            if i+1 < nargs {
                perm = permVal(ctx, ic.a[i+1], uint32(perm))
                i += 1
            }
        }
        if ctx.all {
            if err := os.MkdirAll(name, perm); err != nil {
                erro(ctx, "%v", err).debug(1)
                break
            }
        } else {
            if err := os.Mkdir(name, perm); err != nil {
                erro(ctx, "%v", err).debug(1)
                break
            }
        }
    }
    return
}

type builtin_chdir struct { builtin_ }
func (ctx *builtin_chdir) c(ic *invocation, w facet) (res interface{}) {
    if len(ic.a) == 1 {
        var str = ic.a[0].strval(ctx)
        if err := lockCD(str, 0); err != nil {
            erro(ctx, "%v", err).debug(1)
        }
    } else {
        erro(ctx, "wrong number of arguments: %v", len(ic.a))
    }
    return
}

type builtin_rename struct { builtin_ }
func (ctx *builtin_rename) c(ic *invocation, w facet) (res interface{}) {
    for i, nargs := 0, len(ic.a); i < nargs; i += 1 {
        var (
            a = ic.a[i]
            oldname, newname string
        )
        switch t := a.(type) {
        case *pair: // rename oldname=newname
            oldname = t.Key.strval(ctx)
            newname = t.Value.strval(ctx)
        case *group: // rename (oldname newname) (old new)
            if t.Len() == 2 {
                oldname = t.Get(0).strval(ctx)
                newname = t.Get(1).strval(ctx)
            } else {
                erro(of(ctx,t), "wrong size of group `%v'", t).debug(1)
                break
            }
        case *List: // rename oldname newname, old new, ...
            if t.Len() == 2 {
                oldname = t.Get(0).strval(ctx)
                newname = t.Get(1).strval(ctx)
            } else {
                erro(of(ctx,t), "wrong size of list `%v'", t).debug(1)
                break
            }
        default: // rename newname oldname  newname oldname ...
            if i+1 < nargs {
                oldname = ic.a[i+0].strval(ctx)
                newname = ic.a[i+1].strval(ctx)
                i += 1
            } else {
                erro(of(ctx,t), "Wrong arguments `%v'", ic.a).debug(1)
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

type builtin_remove struct { builtin_
    skip string `skip`
    ignoreMissing bool `i,ig,ignore,ignore-missing`
    warnNotFile bool `warn-not-file`
    all bool `a,all,r,recursive`
}
func (ctx *builtin_remove) c(ic *invocation, w facet) (res interface{}) {
    var opts = ctx
    var remove func(Context, Value)
    var removeFile = func(ctx Context, f *File) {
        var err error
        var s = f.fullname()
        if opts.skip != "" {
            if strings.HasPrefix(s, opts.skip) { return } else
            if strings.HasPrefix(f.name(ctx), opts.skip) { return }
        }
        if opts.all { err = os.RemoveAll(s) } else { err = os.Remove(s) }
        if err != nil {
            erro(ctx, "remove: %v", err)
            erro(ctx, "remove: %v -> %s", f, s).debug(6)
            return
        }
        if d := opts.debug; d>0 { warn(ctx, "remove %s (%s)", f, s).debug(d) }
        if opts.verbose { prompt(ctx, "removed %s\n", f) }
    }
    var removePath = func(ctx Context, p *Path) {
        var err error
        var s = p.strval(ctx)
        if opts.skip != "" {
            if strings.HasPrefix(s, opts.skip) { return }
        }
        if opts.all { err = os.RemoveAll(s) } else {
            erro(ctx, "remove path: %v", p).debug(6)
            return
        }
        if err != nil {
            erro(ctx, "remove: %v", err)
            erro(ctx, "remove: %v", p).debug(6)
            return
        }
        if d := opts.debug; d>0 { warn(ctx, "remove %s", s).debug(d) }
        if opts.verbose { prompt(ctx, "removed %s\n", s) }
    }
    var removePat = func(ctx Context, pat Value) {
        var val = (&builtin_wildcard{builtin_:builtin_{Context:ctx}}).do(pat)
        erro(ctx, "TODO: remove: %T %v -> %T %v", pat, pat, val, val).debug(1)
    }

    remove = func(ctx Context, v Value) {
        if _, y := v.(*none); y {
            return
        } else if isTrivial(v) {
            warnstack(ctx, 5, "triviality: %v (%T)", v, v).debug(6)
        } else if u, y := v.(unexpanded); y {
            if !opts.verbose && opts.debug == 0 { return }
            warnstack(ctx, 5, "unexpended: %v (%T)", u.Value, u.Value).debug(6)
        } else if l, y := v.(*List); y {
            for _, v := range l.Elems { remove(ctx, v) }
        } else if d, y := v.(*delegate); y {
            if u, y := d.x.(unresolved); y {
                if opts.debug > 0 {
                    warn(ctx, "unresolved: %v: %v (%T, %v, %v)", d, u.Value, u.Value, d.o, d.a).debug(6)
                }
                return
            }
            warnstack(ctx, 5, "delegate: %v (%T, %v, %v)", d.x, d.x, d.o, d.a).debug(6)
        } else if v.patterned(ctx) {
            removePat(ctx, v)
        } else if f, y := v.(*File); y {
            removeFile(ctx, f)
        } else if f = file(ctx, v.strval(ctx)); f != nil {
            removeFile(ctx, f)
        } else if p, y := v.(*Path); y {
            removePath(ctx, p)
        } else if !opts.ignoreMissing {
            errostack(ctx, 5, "not file: %v (%T)", v, v).debug(6)
        }
    }
    for _, a := range ic.a { ctx := at(ctx.Context, a.Position())
        remove(ctx, a.expand(ctx, plain))
    }

    if opts.debug > 0 { warn(ctx, "%v", ic.a).debug(1) }
    if opts.debug > 0 && ctx.dia().flush() > 0 {
        errostack(ctx, 3, "remove errors").debug(1)
    }
    return
}

type builtin_truncate struct { builtin_ }
func (ctx *builtin_truncate) c(ic *invocation, w facet) (res interface{}) {
    for i, nargs := 0, len(ic.a); i < nargs; i += 1 {
        var (
            a = ic.a[i]
            name string
            size int64
            e error
        )
        switch t := a.(type) {
        case *pair: // truncate name => size old => new
            name = t.Key.strval(ctx)
            if size, e = t.Value.int(ctx); e != nil {
                erro(ctx, "%v: %v", t.Value, e).debug(1)
            }
        case *group: // truncate (name size) (old new)
            if t.Len() == 2 {
                name = t.Get(0).strval(ctx)
                if size, e = t.Get(1).int(ctx); e != nil {
                    erro(ctx, "%v: %v", t.Get(1), e).debug(1)
                }
            } else {
                erro(ctx, "Wrong size of group `%v'", t).debug(1)
                break
            }
        case *List: // truncate name size, old new, ...
            if t.Len() == 2 {
                name = t.Get(0).strval(ctx)
                if size, e = t.Get(1).int(ctx); e != nil {
                    erro(ctx, "%v: %v", t.Get(1), e).debug(1)
                }
            } else {
                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                break
            }
        default: // truncate name size  name size ...
            if i+1 < nargs {
                name = ic.a[i+0].strval(ctx)
                if size, e = ic.a[i+1].int(ctx); e != nil {
                    erro(ctx, "%v: %v", ic.a[i+1], e).debug(1)
                }
                i += 1
            } else {
                erro(ctx, "Wrong arguments `%v'", ic.a).debug(1)
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

type builtin_link struct { builtin_ }
func (ctx *builtin_link) c(ic *invocation, w facet) (res interface{}) {
    for i, nargs := 0, len(ic.a); i < nargs; i += 1 {
        var (
            oldname, newname string
            a = ic.a[i]
        )
        switch t := a.(type) {
        case *pair: // link oldname => newname old => new
            oldname = t.Key.strval(ctx)
            newname = t.Value.strval(ctx)
        case *group: // link (oldname newname) (old new)
            if t.Len() == 2 {
                oldname = t.Get(0).strval(ctx)
                newname = t.Get(1).strval(ctx)
            } else {
                erro(ctx, "Wrong size of group `%v'", t).debug(1)
                break
            }
        case *List: // link oldname newname, old new, ...
            if t.Len() == 2 {
                oldname = t.Get(0).strval(ctx)
                newname = t.Get(1).strval(ctx)
            } else {
                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                break
            }
        default: // link oldname newname  oldname newname ...
            if i+1 < nargs {
                oldname = ic.a[i+0].strval(ctx)
                newname = ic.a[i+1].strval(ctx)
                i += 1
            } else {
                erro(ctx, "Wrong arguments `%v'", ic.a).debug(1)
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
type builtin_symlink struct { builtin_
    path     bool `p,path`
    force    bool `force;ow,overwrite`
    update   bool `u,update`
    relative bool `r,rel,relative;l`
}
func (ctx *builtin_symlink) c(ic *invocation, w facet) (res interface{}) {
ForArgs:
    for i, na := 0, len(ic.a); i < na; i += 1 {
        var (
            opts = *ctx // make a copy
            srcNameVal, dstNameVal Value
            srcName   , dstName    string
            srcDir    , dstDir     string
            aa []Value
        )
        switch t := ic.a[i].(type) {
        case *pair: // symlink srcName=dstName srcName=>dstName...
            srcNameVal, dstNameVal = t.Key, t.Value
        case *group: // symlink (-u srcName dstName) (-v srcName dstName)...
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
                srcNameVal = ic.a[i+0]
                dstNameVal = ic.a[i+1]
                i += 1
            } else {
                var a = autoVal(ctx,"@")
                var l = autoVal(ctx,"<")
                var r = autoVal(ctx,">")
                prompt(ctx, "symlink: args=%v -> %v\n", ic.a, t)
                prompt(ctx, "symlink: %v, %v, %v\n", a, l, r)
                errostack(of(ctx,t), 5, "expects pair of names (%T %v)", t, t).debug(6)
                return
            }
        }

        if srcDir, srcName = splitFileName(ctx, srcNameVal); srcName == "" {
            prompt(ctx, "symlink: args=%v\n", ic.a)
            prompt(ctx, "symlink: src=%v\n", srcNameVal)
            errostack(of(ctx,srcNameVal), 5, "empty src filename (%T)", srcNameVal).debug(6)
            return
        }
        if dstDir, dstName = splitFileName(ctx, dstNameVal); dstName == "" {
            prompt(ctx, "symlink: args=%v\n", ic.a)
            prompt(ctx, "symlink: dest=%v\n", dstNameVal)
            errostack(of(ctx,dstNameVal), 6, "empty dest filename (%T)", dstNameVal).debug(12)
            return
        }

        var src = srcName
        var dst = dstName
        if !filepath.IsAbs(src) { src = filepath.Join(srcDir, srcName) }
        if !filepath.IsAbs(dst) { dst = filepath.Join(dstDir, dstName) }
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

type builtin_stat struct { builtin_
    dir bool `di,dr,dir`
    file bool `fi,file`
    symbol bool `s,sym,symlink,symbol,l,link`
}
func (ctx *builtin_stat) x(ic *invocation, w facet) (res interface{}) {
    if len(ic.a) == 0 { return }

    var proj = ctx.Project()
    if proj == nil {
        erro(ctx, "unknown current context").debug(1)
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
        } else if mode := file.info.Mode(); ctx.dir && mode&os.ModeDir != 0 { // IsDir()
            vals = append(vals, valT)//file
        } else if ctx.symbol && mode&os.ModeSymlink != 0 {
            vals = append(vals, valT)//file
        } else if ctx.file && mode&os.ModeType != 0 { // IsRegular()
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
        if s = a.strval(ctx); filepath.IsAbs(s) {
            file = stat(ctx, s, "", "")
        } else {
            file = stat(ctx, s, "", proj.absPath)
        }
        if file == nil { file = proj.file(ctx, s) }
        if file != nil { check(file) }
    }

    for _, a := range ic.a {
        switch t := a.(type) {
        case *File: check(t)
        case *Path: checkstat(a)
        default:    checkstat(a)
        }
    }

    return vals
}

type builtin_file struct { builtin_
    caller bool `c,cc,caller,callercontext,caller-context`
    exists bool `e,ex,exist,exists,me,existsist,must-exist,must,required`
    report bool `r,report,reportmissing;rm,report-missing;er,err,error`
    ignore bool `i,ig,ignore,ignore-missing`
}
func (ctx *builtin_file) x(ic *invocation, w facet) (res interface{}) {
    var proj *Project = ctx.Project()
    if false { if ctx.caller {
        // program -> closure -> traversal -> ...
        if false {
            proj = ctx.closure().Project()
        } else {
            proj = ctx.pc().Project()
        }
    }}
    return ctx.do([]*Project{proj}, ic.a...)
}
func (ctx *builtin_file) do(projs []*Project, args ...Value) (list []Value) {
    var en int
    var cc = ctx.Context
    var fil = func(a Value) { ctx.Context = at(cc, a.Position())
        if false { if strings.HasPrefix(a.strval(ctx), ".configure/") { defer func() {
            if list != nil { if t, y := list[len(list)-1].(*File); y {
                noted(ctx, "%T %v ⇒ %v %s", a, a, t, t.fullname()).debug(10)
            }}
        }()}}

        var fs []*File
        var am []matchedFileMap
        if f, y := toFile(a); y {
            if !ctx.exists || f.exists() /* || f.stat(ctx) != nil */ {
                list = append(list, f)
            } else if ctx.report {
                info(ctx, "no such file {%v %v %v}", f.dir, f.sub, f.name).debug(1)
            }
            return
        }

        if am = files(ctx, a, projs...); am == nil { if s := a.strval(ctx); s != "" { const w = false
            if am = files(ctx, s, projs...); am != nil {
                if w { warnstack(ctx, 3, "%v: incorrect files(%T %v) (%v)", projs, a, a, ctx.Project()).debug(6) }
            } else if f := file(ctx, s); f != nil {
                if w { warnstack(ctx, 3, "%v: incorrect files(%T %v) (%v)", projs, a, a, ctx.Project()).debug(6) }
                list = append(list, f)
                return
            } else {
                return
            }
        }}

        for _, p := range projs { fs = append(fs, p.selectFiles(ctx, am)...) }
        for _, f := range fs {
            if !ctx.exists || f.exists() {
                list = append(list, f)
            } else if ctx.ignore {
                if ctx.verbose { info(ctx, "%s(%v) → %v", typeof(a), a, f).debug(1) }
            } else if ctx.exists {
                en += 1
            }
        }

        if en > 0 { for i, m := range am {
            info(of(ctx,m.pattern), "found %d. %s → %s(%s) → %v", i, m.name, typeof(m.pattern), m.pattern, m.locs)
        }}
    }

    for _, a := range umerge(true, args...) { if fil(a); en > 0 {
        erro(ctx, `%v: %s(%v) is not a file (%v)`, projs, typeof(a), a, list)
        errostack(ctx, 5).debug(16)
        break
    }}
    return list
}

type builtin_glob struct { builtin_
    dir bool `di,dir,directory`
    file bool `fi,file`
    symbol bool `s,sym,symlink,symbol,symbolic`
}
func (ctx *builtin_glob) x(ic *invocation, w facet) (res interface{}) {
    var cwd string // TODO: get current work directory
    var proj *Project
    if proj = ctx.Project(); proj == nil {
        erro(ctx, "unknown current cntext").debug(1)
        return
    }

    var list []Value
    var pos = ctx.Position()
    for _, a := range ic.a {
        var ( str string; names []string )
        if str = a.strval(ctx); !filepath.IsAbs(str) {
            str = filepath.Join(cwd, str)
        }

        var err error
        if names, err = filepath.Glob(str); err != nil {
            erro(ctx, "glob '%v' failed: %v", str, err).debug(1)
            return
        }
        for _, name := range names {
            //var fi, _ = os.Stat(name)
            // TODO: ctx.dir, ctx.file, ctx.symbol
            list = append(list, pathStr(ctx, pos, name))
        }
    }
    return list
}

func readDirNames(ctx Context, sd string, errorMissing bool) (names []string) {
    var dir *os.File
    if fi, err := os.Stat(sd); err != nil {
        if errorMissing { erro(ctx, "%v", err).debug(1) }
        return
    } else if !fi.IsDir() {
        erro(ctx, "not dir: %v", sd).debug(1)
        return
    } else if dir, err = os.Open(sd); err != nil {
        erro(ctx, "not dir: %v", sd).debug(1)
        return
    }

    // NOTE: see alsl filepath.Glob(...)
    var _names, err = dir.Readdirnames(-1); dir.Close()
    if err != nil {
        if errorMissing { erro(ctx, "readdir: %v", err).debug(1) }
        return
    } else { names = _names }
    return
}

type builtin_wildcard struct { builtin_
    includeMissing bool `im,includemissing,include-missing,m,missing`
    ignoreMissing bool `gm,ignoremissing,ignore-missing`
    errorMissing bool `em,errormissing,e,err,error-missing,no-missing`
    names bool `bare,n,name,names`
    strs bool `s,str,strs,string,strings`
    exclude []Value `x,ex,exc,excl,exclude,except,no,not`
    filetype string `ft,filetype,file-type` // dir, file, etc.
    dir string `di,dir,directory`
}
func (ctx *builtin_wildcard) _do(pats ...Value) (files []*File) {
    var db = false //ctx.debug == 1000

    //strings.HasSuffix(ctx.dir, "/testdata/wildcard")
    if false { if pats != nil { defer func() { if files == nil {
        noted(ctx, "%v %v -> %v", ctx.dir, pats, files).debug(10)
    }}(); db = pats[0].String() == "[foobar/config/*.def.am, **.def.am]" }}

    type subr struct {
        d, n, dn string
        isDir bool
        pat chan Value
        ss []*subr
        sync.WaitGroup
        sync.Mutex
    }

    var work func(sub *subr)
    var top = subr{ pat: make(chan Value, 1) }
    var subsub = func(sub *subr) (ss *subr) {
        if sub.ss != nil { for _, s := range sub.ss {
            if s != nil && sub != nil && s.d == sub.dn { return s }
        }}
        return
    }
    var subed = func(sub *subr, pat Value) {
        var ss *subr = subsub(sub)
        if ss == nil {
            sub.Lock()
            if ss = subsub(sub); ss == nil {
                if false { info(ctx, "%p: %v %v %v", sub, pat, sub.d, sub.n) }
                ss = &subr{ d: sub.dn, pat: make(chan Value, 1) }
                sub.ss = append(sub.ss, ss)
                top.Add(1) ; go work(ss)
            }
            sub.Unlock()
        }
        ss.pat <- pat
    }
    var missing = ctx.includeMissing && !ctx.ignoreMissing
    var collect = func(name string) {
        var a []os.FileInfo ; if missing { a = append(a, nil) }
        var f = stat(ctx, name, "", ctx.dir, a...)
        if true { assert(f != nil, "stat %s %s", name, ctx.dir) }

        top.Lock()
        switch d := f.info.IsDir(); ctx.filetype {
        case "f", "file": if!d { files = append(files, f) }
        case "d", "dir" : if d { files = append(files, f) }
        case "":         files = append(files, f)
        default: erro(ctx, "unknown -filetype: %s (%v)", ctx.filetype, f).debug(1)
        }
        top.Unlock()
        top.Done()
    }
    var subcard = func(sub *subr, pat Value) {
        defer sub.Done()

        if db { defer func(){ noted(ctx, "subcard: %T %v, %v, %v", pat, pat, sub.dn, files).debug(3) }() }

        if t, y := pat.(*compositePattern); y { pat = t.Value }
        if t, y := pat.(*List); y {
            warn(ctx, "pattern is a list: %T %v %v", pat, pat, t.Elems).debug(1)
            if len(t.Elems) == 1 { pat = t.Elems[0] }
        }

        var dir = ctx.dir
        var ctx = at(ctx, pat.Position())
        if p, y := pat.(*Path); !y {
            // fallthrough
        } else if nElems := len(p.Elems); nElems == 0 {
            errostack(ctx, 3, "empty path: %v", pat).debug(3)
            return
        } else if y, _, _ = p.Elems[0].match(ctx, sub.n); y && nElems == 1 {
            errostack(ctx, 3, "%v %v: invalid path: %v, %v, %v", dir, sub.dn, pat, sub.n, nElems).debug(1)
            return
        } else if y && sub.isDir && nElems > 1 {
            val := p.Elems[1]
            if nElems > 2 {
                var v = &Path{}
                v.position = val.Position()
                v.Elems = p.Elems[1:]
                val = v
            }
            subed(sub, val)
            return
        } else if false && sub.d == "" {
            if y { warn(ctx, "bad: %T %v %v", pat, pat, sub).debug(16) }
            return
        } else if true && sub.d == "" {
            if y { warn(ctx, "%T %v %v", pat, pat, sub).debug(1) }
            return
        }

        const infos = false

        if gp, y := pat.(*GlobPattern); !y {
            // fallthrough
        } else if len(gp.components) == 0 {
            errostack(ctx, 3, "empty glob: %v (%s)", pat, sub.dn).debug(3)
            return
        } else if m, y := gp.components[0].(*GlobMeta); !y {
            // fallthrough
        } else if m.Token == DAST { // aka **
            if y, _, _ = gp.match(ctx, sub.dn); infos {
                info(ctx, "_wildcard: %v %v (%v %v, %v)", gp, sub.dn, sub.d, sub.n, y)
            }
            if sub.isDir { subed(sub, pat) }
            if y { top.Add(1) ; go collect(sub.dn) ; return }
            return
        }

        var y bool
        if y, _, _ = pat.match(ctx, sub.n); infos {
            info(ctx, "_wildcard: %s %v, %v (%v %v, %v)", typeof(pat), pat, sub.dn, sub.d, sub.n, y)
        }
        if y { top.Add(1) ; go collect(sub.dn) ; return }
        return
    }
    var subwork = func(subdir, name string, pats []Value) {
        defer top.Done()

        var sub = &subr{ d:subdir, n:name, dn:filepath.Join(subdir,name) }

        if db { defer func(){ noted(ctx, "subwork: %v %s", pats, sub.dn).debug(3) }() }

        for _, x := range ctx.exclude {
            if y, _, _ := x.match(ctx, sub.dn); y { return }
        }

        if fi, err := os.Stat(filepath.Join(ctx.dir, sub.dn)); err == nil {
            sub.isDir = fi.IsDir()
        } else if true {
            erro(ctx, "%p: %v %v → %v", sub, sub.d, sub.n, sub.dn)
            errostack(ctx, 3, "%v", err).debug(16)
            return
        } else {
            warn(ctx, "%p: %v %v → %v", sub, sub.d, sub.n, sub.dn)
            warn(ctx, "%v", err).debug(16)
            return
        }

        for _, pat := range pats { sub.Add(1) ; go subcard(sub, pat) }
        top.Add(1) ; go func() { sub.Wait()
            for _, s := range sub.ss { if s.pat != nil { close(s.pat) }}
            if db { info(ctx, "subwork: %v %v %v", pats, sub.dn, files).debug(1) }
            top.Done()
        } ()
    }

    work = func(sub *subr) {
        names := readDirNames(ctx, filepath.Join(ctx.dir, sub.d), ctx.errorMissing)

        if db { noted(ctx, "work: dir=%v, names=%v", sub.d, names).debug(3) }

        var pats []Value
        for v := range sub.pat { pats = append(pats, v) }
        for _, name := range names { top.Add(1) ; go subwork(sub.d, name, pats) }
        top.Done()
    }

    // Merge pats to make sure no patterns are concealed in List values, this will
    // fail matchings in subcard routine.
    pats = merge(pats...)

    top.Add(1) ; go work(&top)
    for _, v := range pats { top.pat <- v }; close(top.pat)
    top.Wait()
    return
}
func (ctx *builtin_wildcard) do(pats ...Value) (files []*File) {
    // strings.HasSuffix(ctx.dir, "/testdata/wildcard")
    if false { if ctx.dir == "" && pats != nil { defer func() { if files == nil {
        noted(ctx, "%v %v -> %v", ctx.dir, pats, files).debug(5)
    }}()}}

    if ctx.dir != "" {
        return ctx._do(pats...)
    } else {
        return ctx.Project().wildcard(ctx, pats...)
    }
}
func (ctx *builtin_wildcard) x(ic *invocation, w facet) (res interface{}) {
    var vals []Value // strings.HasSuffix(ctx.dir, "/testdata/wildcard")
    if false { if ctx.dir == "" && ic.a != nil { defer func() { if vals == nil {
        noted(ctx, "%v %v %v -> %v", ctx.dir, ic.o, ic.a, res).debug(10)
    }}()}}

    if len(ctx.exclude) > 0 { ctx.exclude = umerge(true, ctx.exclude...) }

    for _, file := range ctx.do(merge(ic.a...)...) {
        if file == nil {
            errostack(ctx, 3, "nil file: %v", ic.a).debug(3)
        } else if !(ctx.names || ctx.strs) {
            vals = append(vals, file)
        } else if ctx.strs {
            vals = append(vals, MakeString(file.position, file.name(ctx)))
        } else if strings.Contains(file.name(ctx), PathSep) {
            vals = append(vals, pathStr(ctx, file.position, file.name(ctx)))
        } else {
            vals = append(vals, MakeBareword(file.position, file.name(ctx)))
        }
    }
    return vals
}

type builtin_readdir struct { builtin_ }
func (ctx *builtin_readdir) x(ic *invocation, w facet) (res interface{}) {
    var l []Value
    for _, a := range ic.a {
        if fis, err := ioutil.ReadDir(a.strval(ctx)); err == nil {
            v := new(List)
            for _, fi := range fis {
                v.Append(MakeString(a.Position(), fi.Name()))
            }
            l = append(l, v)
        } else {
            break //l = append(l, makeNone(pos))
        }
    }
    return l
}

type builtin_readfile struct { builtin_
    trim      bool `ta,trim,trim-all`
    trimLeft  bool `tl,trim-left`
    trimRight bool `tr,trim-right`
}
func (ctx *builtin_readfile) x(ic *invocation, w facet) (res interface{}) {
    var l []Value
    var closured = closureProjects(ctx)
    for _, v := range ic.a {
        if o, y := (as{v}.fullnameOpt(ctx, closured...)); !y {
            errostack(of(ctx,v), 5, "%v is not a file", v).debug(1)
            break
        } else if s, e := ioutil.ReadFile(o.strval(ctx)); e != nil {
            errostack(of(ctx,v), 5, "read file failed: %v", e).debug(1)
            break
        } else {
            if ctx.trim      { s = bytes.TrimFunc     (s, unicode.IsSpace) } else
            if ctx.trimLeft  { s = bytes.TrimLeftFunc (s, unicode.IsSpace) } else
            if ctx.trimRight { s = bytes.TrimRightFunc(s, unicode.IsSpace) }
            l = append(l, MakeString(v.Position(), string(s)))
        }
    }
    return l
}

type builtin_writefile struct { builtin_
    path bool `p,path`
}
func (ctx *builtin_writefile) x(ic *invocation, w facet) (res interface{}) {
    // $(write-file filename,content)
    // $(write-file -p filename,content)
ForArgs:
    for i := 0; i < len(ic.a); i += 1 {
        var (
            a = ic.a[i]
            name, data string
            perm = os.FileMode(0600)
        )
        switch t := a.(type) {
        case *pair: // write-file name=text name=text
            name = t.Key  .strval(ctx)
            data = t.Value.strval(ctx)
        case *group: // write-file (name text) (name text 0660)
            if n := t.Len(); n < 4 && n > 0 {
                name = t.Get(0).strval(ctx)
                if n > 1 { data = t.Get(1).strval(ctx) }
                if n > 2 { perm = permVal(ctx, t.Get(2),0600) }
            } else {
                erro(ctx, "Wrong size of group `%v'", t).debug(1)
                break
            }
        case *List: // write-file name text, name text 0660, ...
            if n := t.Len(); n < 4 && n > 0 {
                name = t.Get(0).strval(ctx)
                if n > 1 { data = t.Get(1).strval(ctx) }
                if n > 2 { perm = permVal(ctx, t.Get(2),0600) }
            } else {
                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                break
            }
        default: // write-file name text 0660  name text 0660 ...
            name = ic.a[i].strval(ctx)
            if i+1 < len(ic.a) {
                data = ic.a[i+1].strval(ctx)
                i += 1
            }
            if i+1 < len(ic.a) {
                perm = permVal(ctx, ic.a[i+1],0600)
                i += 1
            }
        }
        if name == "" {
            continue ForArgs
        } else if dir := filepath.Dir(name); ctx.path && dir != "." && dir != PathSep {
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
    var a, filename, c = as{file}.fullname(ctx)
    if filename == "" {
        errostack(of(ctx,file), 3, "touch: empty file name: %v (%v, %v, %v)", file, typeof(file), a, c).debug(24)
        return
    } else if d := filepath.Dir(filename); optPath && d != "." && d != PathSep {
        if err = os.MkdirAll(d, os.FileMode(optMode|0733)); err != nil {
            erro(of(ctx,file), "touch: %v", err).debug(1)
            return
        }
    }

    var (
        mode = os.FileMode(optMode)
        at, mt time.Time
        m os.FileMode
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

type builtin_touchfile struct { builtin_
    mode os.FileMode `m,mode;fm,filemode,file-mode`
    path bool `p,path`
}
func (ctx *builtin_touchfile) x(ic *invocation, w facet) (res interface{}) {
    // $(touch-file filename)
    // $(touch-file -p filename)
    for i := 0; i < len(ic.a); i += 1 {
        if err := touch(ctx, ic.a[i], uint32(ctx.mode), ctx.path); err != nil {
            erro(ctx, "%v", err).debug(1)
            break
        }
    }
    return
}

// $(grep 'status=1',$@)
// $(grep 'status=([0-9]+)',$1,$@)
type builtin_grep struct { builtin_ }
func (ctx *builtin_grep) x(ic *invocation, w facet) (res interface{}) {
    var (
        args = ic.a
        nargs = len(args)
        list []Value
        rxs []*regexp.Regexp // TODO: move it into builtinGrepOpts
        result Value
        err error

    )
    if !(nargs == 2 || nargs == 3) {
        erro(ctx, "wants exactly 2 args, e.g. $(grep -1 '^example$',$(file))").debug(1)
        return
    }

    var rvs = merge(args[0])
    switch nargs {
    case 2:   args = args[1:]
    case 3: result = args[1] ; args = args[2:]
    }
    for _, a := range rvs {
        if s := a.strval(ctx); s == "" {
            erro(of(ctx,a), "empty regexp").debug(1)
            return
        } else if r, e := regexp.Compile(s); e != nil {
            erro(of(ctx,a), "%v", e).debug(1)
            return
        } else {
            rxs = append(rxs, r)
        }
    }

    var pos = ctx.Position()
    var cc = &autoContext{ Context:ctx, defs:make(autoDefMap) }
    var greped = func(line int, match []string) (done bool) {
        var vals []Value
        for i, s := range match {
            if d, v := cc.set(ctx, fmt.Sprintf("%d",i), MakeString(pos, s)); d == nil {
                erro(ctx, "set $%d to '%s' failed", i, s).debug(1)
                return
            } else { vals = append(vals, v) }
        }
        defer func() {
            for i, v := range vals {
                if d, v := cc.set(ctx, fmt.Sprintf("%d",i), v); d == nil {
                    erro(ctx, "restore $%d to '%s' failed", i, v).debug(1)
                }
            }
        } ()
        list = append(list, result.expand(cc, expandDigits|plain))
        return
    }

    for _, a := range umerge(true, args...) {
        var file *os.File
        var filename string

        if f, y := a.(*File); y {
            filename = f.fullname()
        } else {
            filename = a.strval(ctx)
        }

        if c := of(ctx, a); filename == "" {
            var pc = ctx.pc()
            erro(c, "empty filename: %T %v", a, a)
            erro(c, "%v %v", rvs, args)
            errostack(c, 5, "%p %v", pc, pc.get(ctx, "^")).debug(64)
            return
        } else if file, err = os.Open(filename); err != nil {
            erro(c, "%v", err)
            errostack(c, 5, "%v (%T)", a.strval(ctx), a).debug(128)
            return
        }
        defer file.Close()

        var line int // line number
        var scanner = bufio.NewScanner(file)
        scanner.Split(bufio.ScanLines)
        outer: for scanner.Scan() {
            var text = scanner.Text()
            line += 1 // starting from #1
            for _, rx := range rxs {
                var sm = rx.FindStringSubmatch(text)
                if len(sm) > 0 && greped(line, sm) { break outer }
            }
        }
    }
    return ease(ctx, list)
}

var (
    rsAutoconf  = `AC_(CHECK_(FILES?|FUNCS?|HEADERS?|PROG|SIZEOF|TOOL)|DEFINE)\(([^\)]*?)\)`
    rsConfigRef = `[$%]\{([^\s\}]+)\}|@([^\s\@]+)@`
    rsConfigure = `^[\t ]*#[\t ]*(define|undef|smartdefine|smartdefine01|cmakedefine|cmakedefine01)[\t ]+([A-Za-z0-9_]+)(?:[\t ]+([^\n]*))?$`
    rxAutoconf  = regexp.MustCompile(rsAutoconf)
    rxConfigure = regexp.MustCompile(fmt.Sprintf(`(?m:%s)`, rsConfigure)) // m: multilines
    rxConfigRef = regexp.MustCompile(rsConfigRef)
)

func (project *Project) strExpandConfig(ctx Context, s string) (result string, err error) {
    var (
        pos Position
        res = new(bytes.Buffer)
        index, line = 0, 0
    )
    if d := autoVal(ctx, "-file"); d != nil {
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
        } else if val = def.invoke(ctx, plain, nil, nil); isNull(val) {
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
        case *Plain: fmt.Fprintf(res, "%s", t.raw.String())
        case *answer, *boolean:
            if i, e := t.int(ctx); e == nil {
                fmt.Fprintf(res, "%d", i)
            } else {
                erro(ctx, "%: %v", t, i).debug(1)
            }
        case *group:
            fmt.Fprintf(res, "%s", parseGroupValue(ctx, t).strval(ctx))
        default:
            fmt.Fprintf(res, "%s", val.strval(ctx))
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
        if def = project.resolveDef(ctx, name); def != nil { // t = def.true(ctx);
            if val := def.invoke(ctx, plain, nil, nil); val == nil {
                // noop, TODO: or #undef?
            } else if _, undef := val.(*undef); undef {
                _, err = out.WriteString(fmt.Sprintf("#undef /* %s */", name))
                if err != nil { erro(ctx, "%v", err); return }
                continue
            } else {
                   t = val.true(ctx)
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
            } else if isNull(def.value) || isNone(def.value) {
                s = fmt.Sprintf("#undef %s /* %v */", name, def.value)
            } else if va, _, _ = plain.expand(ctx, def.value); len(va) == 1 {
                switch v := va[0].(type) {
                case *answer, *boolean:
                    if b := v.true(ctx); b {
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

type builtin_untraversed struct { builtin_ }
func (ctx *builtin_untraversed) x(ic *invocation, w facet) (res interface{}) {
    return untraversed{ease(ctx, ic.a)}
}

type builtin_return struct { builtin_ }
func (ctx *builtin_return) x(ic *invocation, w facet) (res interface{}) {
    return &returner{valbase{ctx.Position()}, ic.a }
}
