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
    _context "context"
    "reflect"
    "strings"
    "strconv"
    "unicode"
    "unsafe"
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
    builtinCallable uint = 0
    builtinCommand       = 1<<(iota-1)
    builtinForce
)

type builtin_ struct {
    *evocation
    generalOpts
}
func (c builtin_) cast(t reflect.Type) Context {
    if reflect.TypeOf(c) == t { return c }
    if reflect.TypeOf((*builtin_)(nil)) == t { return &c }
    return c.evocation.cast(t) // implcast(c, t)
}

type builtin_a interface{ a() bool }
type builtin_c interface{ c() interface{} }
type builtin_x interface{ x() interface{} }

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
    `or`:        reflect.TypeOf((*builtin_or)(nil)).Elem(),
    `and`:       reflect.TypeOf((*builtin_and)(nil)).Elem(),
    `not`:       reflect.TypeOf((*builtin_not)(nil)).Elem(),
    `xor`:       reflect.TypeOf((*builtin_xor)(nil)).Elem(),

    `equal`:     reflect.TypeOf((*builtin_equal)(nil)).Elem(),
    `equals`:    reflect.TypeOf((*builtin_equal)(nil)).Elem(),
    `ne`:        reflect.TypeOf((*builtin_unequal)(nil)).Elem(),
    `not-equal`: reflect.TypeOf((*builtin_unequal)(nil)).Elem(),
    `match`:     reflect.TypeOf((*builtin_match)(nil)).Elem(),

    `greater`:   reflect.TypeOf((*builtin_greater)(nil)).Elem(),
    `less`:      reflect.TypeOf((*builtin_less)(nil)).Elem(),

    `case`:      reflect.TypeOf((*builtin_case)(nil)).Elem(),
    `if`:        reflect.TypeOf((*builtin_if)(nil)).Elem(),
    `ifeq`:      reflect.TypeOf((*builtin_ifeq)(nil)).Elem(),
    `ifne`:      reflect.TypeOf((*builtin_ifne)(nil)).Elem(),

    `for`:       reflect.TypeOf((*builtin_for)(nil)).Elem(),
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
    `join`:       reflect.TypeOf((*builtin_join)(nil)).Elem(),
    `compose`:    reflect.TypeOf((*builtin_compose)(nil)).Elem(), // concat
    `quote`:      reflect.TypeOf((*builtin_quote)(nil)).Elem(),
    `quote-join`: reflect.TypeOf((*builtin_quotejoin)(nil)).Elem(),

    `split`:             reflect.TypeOf((*builtin_splitstring)(nil)).Elem(),
    `split-string`:      reflect.TypeOf((*builtin_splitstring)(nil)).Elem(), // TODO: remove it?
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
    `finalize`:     reflect.TypeOf((*builtin_finalize)(nil)).Elem(),
    `stringify`:    reflect.TypeOf((*builtin_finalize)(nil)).Elem(), // alias
    `string`:       reflect.TypeOf((*builtin_string)(nil)).Elem(),
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
    `ext`:        reflect.TypeOf((*builtin_ext)(nil)).Elem(),

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

    `dir`:        reflect.TypeOf((*builtin_dir)(nil)).Elem(),
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

    `reldir`:       reflect.TypeOf((*builtin_reldir)(nil)).Elem(),
    `relative-dir`: reflect.TypeOf((*builtin_reldir)(nil)).Elem(),

    `file`:         reflect.TypeOf((*builtin_file)(nil)).Elem(),
    `stat`:         reflect.TypeOf((*builtin_stat)(nil)).Elem(),// stat (deprecates file-exists)
    `glob`:         reflect.TypeOf((*builtin_glob)(nil)).Elem(),
    `wildcard`:     reflect.TypeOf((*builtin_wildcard)(nil)).Elem(),

    `read-dir`:     reflect.TypeOf((*builtin_readdir)(nil)).Elem(),  // io/ioutil/ioutil.go
    `read-file`:    reflect.TypeOf((*builtin_readfile)(nil)).Elem(),  // io/ioutil/ioutil.go

    `grep`:         reflect.TypeOf((*builtin_grep)(nil)).Elem(),

    `untraversed`:  reflect.TypeOf((*builtin_untraversed)(nil)).Elem(),

    // commands ------------------------------------------------------------------
    `print`:        reflect.TypeOf((*builtin_print)(nil)).Elem(),
    `printf`:       reflect.TypeOf((*builtin_printf)(nil)).Elem(),
    `printl`:       reflect.TypeOf((*builtin_printl)(nil)).Elem(),
    `println`:      reflect.TypeOf((*builtin_println)(nil)).Elem(),

    `plain`:        reflect.TypeOf((*builtin_plain)(nil)).Elem(),

    `append`:       reflect.TypeOf((*builtin_append)(nil)).Elem(),
    // `pop`:          reflect.TypeOf((*builtin_pop)(nil)).Elem(),

    `write-file`:   reflect.TypeOf((*builtin_writefile)(nil)).Elem(),  // io/ioutil/ioutil.go
    `touch-file`:   reflect.TypeOf((*builtin_readfile)(nil)).Elem(),  // io/ioutil/ioutil.go

    `push-context`: reflect.TypeOf((*builtin_pushcontext)(nil)).Elem(),
    `pop-context`:  reflect.TypeOf((*builtin_popcontext)(nil)).Elem(),

    `mkdir`:        reflect.TypeOf((*builtin_mkdir)(nil)).Elem(),     // os/file.go
    `chdir`:        reflect.TypeOf((*builtin_chdir)(nil)).Elem(),     // os/file.go
    `rename`:       reflect.TypeOf((*builtin_rename)(nil)).Elem(),    // os/file.go
    `remove`:       reflect.TypeOf((*builtin_remove)(nil)).Elem(),    // os/file_*.go
    `truncate`:     reflect.TypeOf((*builtin_truncate)(nil)).Elem(),  // os/file_*.go
    `link`:         reflect.TypeOf((*builtin_link)(nil)).Elem(),      // os/file_*.go
    `symlink`:      reflect.TypeOf((*builtin_symlink)(nil)).Elem(),   // os/file_*.go

    `serve-http`:   reflect.TypeOf((*builtin_servehttp)(nil)).Elem(),

    `return`:       reflect.TypeOf((*builtin_return)(nil)).Elem(),
}

func escapedString(ctx Context, v Value) (s string) {
    if p, ok := v.(*strlit); ok {
        s = strings.Replace(p.string(ctx), "\\'", "'", -1)
    } else {
        s = v.string(ctx)
    }
    return
}

func isNotSpace(r rune) bool {
    return !unicode.IsSpace(r)
}

func isRelPath(filename string) (res bool) {
    // This implementation replaces:
    //      strings.HasPrefix(filename, "."+pathSep)
    //      strings.HasPrefix(filename, ".."+pathSep)
    var ( s = "."+pathSep ; n = len(filename) )
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
        if t, e := v.float(ctx); e == nil {
            val.SetFloat(t)
        } else {
            erro(ctx, "%v: %v", v, e).debug(10)
        }
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        if t, e := v.int(ctx); e == nil {
            val.SetInt(t)
        } else {
            erro(ctx, "%v: %v", v, e).debug(10)
        }
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        if t, e := v.int(ctx); e == nil {
            val.SetUint(uint64(t))
        } else {
            erro(ctx, "%v: %v", v, t).debug(10)
        }
    case reflect.String:
        val.SetString(v.string(ctx))
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
            erro(at(ctx,v), "option type unsupported: %T %v → %v, %v", v, v, val.Kind(), val.Type()).debug(1)
        }
    case reflect.Ptr:
        switch val.Type().Elem().String() {
        case "smart.fullname":
            if x := v.expand(expandFullFile{ctx}); isTrivial(x) {
                erro(at(ctx, v), "expecting file value: %v{%v}", typeof(v), v).debug(1)
            } else if o, y := (as{x}.fullname(ctx)); y && o.Value != nil {
                val.Set(reflect.ValueOf(&o))
            } else {
                erro(at(ctx,v), "%v: not a file: %v → %v{%v}", ctx.project(), v, typeof(x), x)
                errostack(ctx, 5).debug(32)
            }
        case "smart.File":
            if x := v.expand(final{ctx}); isNone(x) {
                erro(at(ctx,v), "expecting file value: %v{%v}", typeof(x), v).debug(1)
            } else if file, y := toFile(x); y {
                val.Set(reflect.ValueOf(file))
            } else if proj := ctx.project(); proj == nil {
                erro(at(ctx,x), "no current project to find file '%v'", x).debug(1)
            } else if file = proj.file(ctx, x.string(ctx)); file != nil {
                val.Set(reflect.ValueOf(file))
            } else {
                erro(at(ctx,v), "'%v' is not a file", x).debug(1)
            }
        case "regexp.Regexp":
            if rx, e := regexp.Compile(v.string(ctx)); e != nil {
                erro(at(ctx,v), "compile regexp '%v' failed: %v", v, e).debug(1)
            } else {
                val.Set(reflect.ValueOf(rx))
            }
        default:
            erro(at(ctx,v), "option type unsupported: %v{%v} → %v, %v", typeof(v), v, val.Elem().Kind(), val.Type().Elem()).debug(1)
        }
    default:
        switch val.Type().String() {
        case "fs.FileMode", "os.FileMode": // aka. reflect.Uint32
            if t, e := v.int(ctx); e == nil {
                if t == 0 { warn(at(ctx,v), "zero file mode").debug(1) }
                val.SetUint(uint64(t))
            } else {
                erro(ctx, "%v: %v", v, t).debug(1)
            }
        case "regex.Regex": // aka. reflect.Ptr
            erro(at(ctx,v), "TODO: regexp: %T %v → %v, %v", v, v, val.Kind(), val.Type()).debug(1)
        default:
            erro(at(ctx,v), "option type unsupported: %T %v → %v, %v", v, v, val.Kind(), val.Type()).debug(1)
        }
    }
}

func _opt(ctx Context, tag reflect.StructTag, field reflect.Value, args ...Value) (rest []Value) {
    var val = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
    var opts []string // opt names

    if tag == "" { return args }
    if t := string(tag)[:]; t != "" {
        for {
            if i := strings.IndexAny(t, ";, "); 0 <= i {
                opts = append(opts, t[:i])
                t = t[i+1:]
            } else {
                opts = append(opts, t)
                break
            }
        }
    }

outer:
    for _, arg := range args {
        var y bool
        var f flag
        var value Value
        var a = arg

        // skip parsing patterns, e.g. -I%
        if !a.patterned(ctx) {
            switch t := a.(type) {
            case        flag: f, value = t, makeBoolean(t.Position(), true)
            case       *pair: if f, y = t.key.(flag);   y { value = t.val }
            case *argumented: if f, y = t.Value.(flag); y { value = ease(ctx, t.args) }
            }
        }

        if f.Value == nil {
            rest = append(rest, arg)
            continue outer
        }

        for i := 0; i < len(opts); i += 1 {
            if _, y = f.opt(ctx, opts[i]); y {
                _set(ctx, val, value)
                continue outer
            }
        }

        rest = append(rest, arg)
    }

    switch val.Type().String() { // os.FileMode(0640)
    case "fs.FileMode", "os.FileMode": if val.Uint() == 0 { val.SetUint(0640) }
    }
    return
}

func _opts(ctx Context, opts reflect.Value, args []Value) (rest []Value) {
    if opts.Kind() != reflect.Ptr {
        erro(ctx, "opts must be ptr: %v", opts.Kind()).debug(10)
        return
    } else if opts = opts.Elem(); opts.Kind() != reflect.Struct {
        erro(ctx, "opts is not ptr of struct: %v", opts.Kind()).debug(1)
        return
    }

    rest = merge(args...)

    var builtin, general, modifier reflect.Value
    var ot = opts.Type()
    for i := 0; i < ot.NumField(); i += 1 {
        var ft, fv = ot.Field(i), opts.Field(i)
        if t := fv.Type(); fv.Kind() != reflect.Struct {
            if ft.Anonymous && ft.Name == "Context" && t.String() == "smart.Context" {
                continue
            }
            rest = _opt(ctx, ft.Tag, fv, rest...)
        } else if !ft.Anonymous {
            continue
        } else if ft.Name == "generalOpts" {
            general = fv.Addr()
        } else if strings.HasPrefix(ft.Name, "builtin_") {
            if builtin.IsValid() { note(ctx, "embedded multiple builtins: %v", ft).debug(3) }
            builtin = fv.Addr()
        } else if strings.HasPrefix(ft.Name, "modifier_") {
            if modifier.IsValid() { note(ctx, "embedded multiple modifiers: %v", ft).debug(3) }
            modifier = fv.Addr()
        }
    }
    if  builtin.IsValid() { rest = _opts(ctx,  builtin, rest) }
    if  general.IsValid() { rest = _opts(ctx,  general, rest) }
    if modifier.IsValid() { rest = _opts(ctx, modifier, rest) }
    return
}
func parseOpts(ctx Context, store interface{}, args ...Value) (rest []Value) {
    return _opts(ctx, reflect.ValueOf(store), args)
}

// see https://go.dev/doc/tutorial/generics
func _opts_[Opts interface{}](ctx Context, args ...Value) (opts Opts, res []Value) {
    res = parseOpts(ctx, &opts, args...)
    return
}

func _parseHeadArgs(ctx Context, store interface{}, args ...Value) (head, rest []Value) {
    if len(args) == 0 {
        // zero args
    } else if head = parseOpts(ctx, store, args[0]); len(head) > 0 {
        rest = args[1:] //xmerge(ctx, args[1:]...)
    } else if len(args) == 1 {
        // done
    } else if head = xmerge(ctx, args[1]); len(args) > 2 {
        rest = args[2:] //xmerge(ctx, args[2:]...)
    }
    return
}

func _parseHeadArgsMerge(ctx Context, store interface{}, args ...Value) (res []Value) {
    var head, rest = _parseHeadArgs(ctx, store, args...)
    res = append(head, rest...)
    return
}

func _parseHeadArgsRequired(ctx Context, store interface{}, args ...Value) (head, rest []Value) {
    head, rest = _parseHeadArgs(ctx, store, args...)
    if len(head) == 0 || len(rest) == 0 {
        erro(ctx, "insufficient number of arguments").debug(6)
    }
    return
}

func argstring(ctx Context, arg Value) (s string) {
    if isFinalValue(ctx, arg) {
        s = arg.string(ctx)
    }
    return
}

type builtin_noop struct { builtin_ }
func (ctx *builtin_noop) c() interface{} { return nil }
func (ctx *builtin_noop) x() interface{} { return nil }

type builtin_typeof struct { builtin_
    expand bool `expand`
}
func (ctx *builtin_typeof) a() (skip bool) { return }
func (ctx *builtin_typeof) x() (res interface{}) {
    var elems []Value
    for _, arg := range ctx.evocation.a {
        if ctx.expand { arg = arg.expand(ctx) }
        // Arguments are passed in a list:
        //   $(fun abc)             args: (abc)
        //   $(fun a,b,c)           args: (a),(b),(c)
        //   $(fun a b c,1 2 3)     args: (a b c),(1 2 3)
        elems = append(elems, makeBareword(arg.Position(), typeof(arg)))
    }
    return elems
}

type builtin_origin struct { builtin_ }
func (ctx *builtin_origin) x() (res interface{}) {
    var elems []Value
    var scope = ctx.Scope()
    for _, arg := range ctx.evocation.a {
        if s := argstring(ctx, arg); s == "" {
            elems = append(elems, makeNull(arg.Position()))
        } else if d := scope.FindDef(s); d != nil {
            elems = append(elems, makeStrlit(arg.Position(), d.origin.String()))
        } else {
            elems = append(elems, makeNull(arg.Position()))
        }
    }
    return elems
}

type builtin_defined struct { builtin_ }
func (ctx *builtin_defined) x() (res interface{}) {
    var elems []Value
    for _, arg := range ctx.evocation.a {
        var unresolved bool
        erro(ctx, "TODO: %v", us(arg)).debug(1)
        elems = append(elems, makeBoolean(arg.Position(), !unresolved))
    }
    return elems
}

type builtin_pushcontext struct { builtin_ }
func (ctx *builtin_pushcontext) c() (res interface{}) {
    var (
        scope = ctx.Scope()
        uc = _universe(ctx)
        m map[string]*def
    )
    for _, arg := range ctx.evocation.a {
        var s = arg.string(ctx)
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
    rules []Value `rule,rules`
}
func (ctx *builtin_popcontext) c() (res interface{}) {
    for _, arg := range ctx.evocation.a {
        warn(ctx, "unused argument: %T %v", arg, arg).debug(1)
        break
    }

    var rules []Value
    for _, r := range ctx.rules { if v, y := r.(*group); !y {
        rules = append(rules, v)
    } else {
        rules = append(rules, v.elems...)
    }}

    var scope = ctx.Scope()
    var uc = _universe(ctx)
    var l = len(uc.globe.stack)
    if l == 0 { return }
    for s, d := range uc.globe.stack[l-1] { if d == nil { if s == "" { continue }
        scope.mutex.Lock()
        delete(scope.elems, s)
        scope.mutex.Unlock()
    } else if o := scope.Lookup(d.name); o != nil { if t, ok := o.(*def); ok {
        *t = *d
    }}}
    uc.globe.stack = uc.globe.stack[0:l-1]
    return
}

type builtin_position struct { builtin_
    filename bool `filename`
    filenameQuoted bool `quote-filename,quoted-filename`
    line bool `ln,line`
    column bool `col,column`
    addLine int `add,add-line`
    addColumn int `add-column`
}
func (ctx *builtin_position) x() (res interface{}) {
    var vals []Value
    var pos = ctx.Position()
    if ctx.filename {
        vals = append(vals, makeStrlit(pos, pos.Filename))
    } else if ctx.filenameQuoted {
        var s = pos.Filename //strconv.Quote(pos.Filename)
        vals = append(vals, makeStrlit(pos, "\""+s+"\""))
    }

    if ctx.line   { vals = append(vals, makeDecimal(pos, int64(pos.Line + ctx.addLine))) }
    if ctx.column { vals = append(vals, makeDecimal(pos, int64(pos.Column + ctx.addColumn))) }

    if len(vals) == 0 { return makeStrlit(pos, pos.String()) }
    if len(vals) == 1 { return vals[0] }
    return vals
}

type builtin_date struct { builtin_
    time bool `tm,time,now`
}
func (ctx *builtin_date) x() (res interface{}) {
    if t := time.Now(); len(ctx.evocation.a) > 0 {
        var vals []Value
        for _, a := range ctx.evocation.a {
            var s string
            if s = a.string(ctx); s == "" {
                s = t.String()
            } else if s = t.Format(s); s == "" {
                s = fmt.Sprintf("%v", t)
            }
            vals = append(vals, makeStrlit(a.Position(), s))
        }
        return vals
    } else if ctx.time {
        res = makeTime(ctx.Position(), t)
    } else {
        res = makeDate(ctx.Position(), t)
    }
    return
}

type builtin_debug struct { builtin_
    s int `stack`
    n int `num`
}
func (ctx *builtin_debug) x() (res interface{}) {
    var s bytes.Buffer
    for i, a := range ctx.evocation.a {
        if i > 0 { fmt.Fprintf(&s, " ") }
        fmt.Fprintf(&s, "%s", a.string(ctx))
    }
    if hook := _universe(ctx).hooks.debug; hook != nil {
        hook(ctx, s.String(), ctx.evocation.a)
    } else {
        warnstack(ctx, ctx.s, "%s", s.String()).debug(ctx.n)
    }
    return
}

type builtin_error struct { builtin_ }
func (ctx *builtin_error) x() (res interface{}) {
    defer trace(ctx)

    var s bytes.Buffer
    for i, a := range ctx.evocation.a {
        if i > 0 { fmt.Fprintf(&s, " ") }
        fmt.Fprintf(&s, "%s", a.string(ctx))
    }

    errostack(ctx, 5, "%s", s.String()).debug(1)
    return
}

type builtin_warning struct { builtin_ }
func (ctx *builtin_warning) x() (res interface{}) {
    var s bytes.Buffer
    for i, a := range ctx.evocation.a {
        if i > 0 { fmt.Fprintf(&s, " ") }
        fmt.Fprintf(&s, "%s", a.string(ctx))
    }
    warn(ctx, "%s", s).debug(1)
    return
}

type builtin_assert struct { builtin_ ; msg string `msg,message` }
func (ctx *builtin_assert) a() (skip bool) { return }
func (ctx *builtin_assert) c() (res interface{}) { return ctx.x() }
func (ctx *builtin_assert) x() (res interface{}) {
    if false { defer trace(ctx) }

    var d = ctx.debug ; if d < 1 { d = 1 }
    var s = ctx.stack ; if s < 1 { s = 1 }
    var t = diagError ; if ctx.warn { t = diagWarn }

    var hook = _universe(ctx).hooks.assert
    if ctx.evocation.a == nil && hook != nil && !hook(ctx, nil, false) {
        prompt(ctx, "assert: %v\n", ctx.evocation.a)
        diagstack(ctx, s, t).debug(d)
    }

    var cc = ctx.Context
    for _, a := range ctx.evocation.a {
        ctx.Context = at(cc, a.Position())

        var okay = a.true(ctx)
        if hook != nil && hook(ctx, a, okay) || okay { continue }
        if false {
            var v = a.expand(final{ctx})
            prompt(ctx, "assert: %v ⇒ %v: %v\n", us(a), us(v))
            diagstack(ctx, s, t, "%v ⇒ '%s'", us(a), a.string(ctx)).debug(d)
        } else if true {
            diagstack(ctx, s, t, "%v ⇒ '%s'", us(a), a.string(ctx)).debug(d)
        } else {
            diagstack(ctx, s, t, "%v", us(a)).debug(d)
        }
    }

    if ctx.fail { panic(_failure(ctx)) }
    return
}

type builtin_sure struct { builtin_ }
func (ctx *builtin_sure) x() (res interface{}) {
    defer trace(ctx)

    for _, a := range ctx.evocation.a { if !a.true(ctx) {
        erro(at(ctx,a), "assert: %v", us(a)).debug(1)
    }}

    return ctx.evocation.a
}

// $(defor $(x),$(y),$(z)) is identical to $(if $(defined $(x)),$(x),...)
type builtin_defor struct { builtin_ } // aka. defined-or
func (ctx *builtin_defor) x() (res interface{}) {
    for _, a := range merge(ctx.evocation.a...) {
        var unresolved bool
        erro(ctx, "TODO: %v", us(a)).debug(1)
        if unresolved {
            continue
        } else {
            res = a
            break
        }
    }
    return
}

type builtin_or struct { builtin_ }
func (ctx *builtin_or) a() (skip bool) {
    ctx.evocation.a = expand(ctx, ctx.evocation.a...)
    return !_exFinal(ctx) && expandable(final{ctx}, ctx.evocation.a...)
}
func (ctx *builtin_or) x() (res interface{}) {
    for _, a := range merge(ctx.evocation.a...) { if a.true(ctx) { return a } }
    return
}

type builtin_and struct { builtin_ }
func (ctx *builtin_and) a() (skip bool) {
    ctx.evocation.a = expand(ctx, ctx.evocation.a...)
    return !_exFinal(ctx) && expandable(final{ctx}, ctx.evocation.a...)
}
func (ctx *builtin_and) x() (res interface{}) {
    for _, a := range merge(ctx.evocation.a...) {
        if a.true(ctx) { res = a } else { return nil }
    }
    return
}

// $(not x y z) ⇒ (not (or x y z))
// $(not x,y,z) ⇒ (and (not x) (not y) (not z))
type builtin_not struct { builtin_ }
func (ctx *builtin_not) x() (res interface{}) {
    var t bool
    for _, a := range ctx.evocation.a { if t = a.true(ctx); t { break } }
    return !t
}

type builtin_xor struct { builtin_ }
func (ctx *builtin_xor) x() (res interface{}) {
    if vals := merge(ctx.evocation.a...); len(vals) > 1 {
        var t = vals[0].true(ctx)
        for _, a := range vals[1:] { if a.true(ctx) != t {
            return makeBoolean(a.Position(), true)
        }}
    }
    return
}

type builtin_unequal struct { builtin_
    final bool `final`
}
func (ctx *builtin_unequal) x() (res interface{}) {
    if ctx.trace { trace(ctx, "unequal") }

    if len(ctx.evocation.a) != 2 {
        erro(ctx, "unequal: wrong number of arguments: %v", ctx.evocation.a)
        erro(ctx, "try: $(unequal <value-list>,<value-list>)").debug(1)
        return
    }

    var t bool
    var a = ctx.evocation.a[0].expand(final{ctx})
    var b = ctx.evocation.a[1].expand(final{ctx})
    if ctx.final {
        t = a.string(ctx) != b.string(ctx)
    } else {
        t = a.cmp(ctx, b) != cmpEqual
    }

    if t {
        res = makeBoolean(ctx.Position(), true)
    } else if n := ctx.debug; n>0 {
        if l, y := a.(*list); y {
            var v = l.elems[0]
            warn(at(ctx,a), "unequal: a: %T(len=%d), %T %v", a, len(l.elems), v, v)
        } else {
            warn(at(ctx,a), "unequal: a: %T %v", a, a)
        }
        if l, y := b.(*list); y {
            var v = l.elems[0]
            warn(at(ctx,b), "unequal: b: %T(len=%d), %T %v", b, len(l.elems), v, v)
        } else {
            warn(at(ctx,b), "unequal: b: %T %v", b, b)
        }
        warnstack(ctx, n, "unequal: %v", t).debug(n)
    } else if len(ctx.evocation.a)>2 {
        warnstack(at(ctx,ctx.evocation.a[2]), 1, "unequal: extra args specified: %v", ctx.evocation.a[2]).debug(1)
    }
    return
}

type builtin_equal struct { builtin_
    final bool `final`
}
func (ctx *builtin_equal) x() (res interface{}) {
    if ctx.trace { trace(ctx, "equal") }

    if len(ctx.evocation.a) > 0 {
        if a := merge(ctx.evocation.a[0]); len(a) == 1 {
            ctx.evocation.a[0] = a[0]
        } else {
            ctx.evocation.a[0] = makeList(a...)
        }
    }

    if len(ctx.evocation.a) != 2 {
        erro(ctx, "equal: wrong number of arguments: %v", ctx.evocation.a)
        erro(ctx, "try: $(equal <value-list>,<value-list>)").debug(1)
        return
    }

    var a = ctx.evocation.a[0].expand(final{ctx})
    var b = ctx.evocation.a[1].expand(final{ctx})
    var t bool
    if ctx.final {
        t = a.string(ctx) == b.string(ctx)
    } else {
        t = a.cmp(ctx, b) == cmpEqual
    }
    // if ctx.evocation.a[0].String() == "$(foo)" && ctx.evocation.a[1].String() == "foo" {
    //     note(ctx, "%v %v %v", us(a), us(b), t).debug(1)
    // }

    if t {
        res = makeBoolean(ctx.Position(), true)
    } else if n := ctx.debug; n>0 {
        if l, y := a.(*list); y { var v = l.elems[0]
            note(at(ctx,a), "equal: a: %v{%v} (len=%d)", typeof(v), v, len(l.elems))
        } else {
            note(at(ctx,a), "equal: a: %v{%v} (%s)", typeof(a), a, a.string(ctx))
        }
        if l, y := b.(*list); y { var v = l.elems[0]
            note(at(ctx,b), "equal: b: %v{%v} (len=%d)", typeof(v), v, len(l.elems))
        } else {
            note(at(ctx,b), "equal: b: %v{%v} (%s)", typeof(b), b, b.string(ctx))
        }
        notestack(ctx, n).debug(n)
    } else if len(ctx.evocation.a)>2 {
        warnstack(at(ctx,ctx.evocation.a[2]), 1, "equal: extra args specified: %v", ctx.evocation.a[2]).debug(1)
    }
    return
}

type builtin_greater struct { builtin_ }
func (ctx *builtin_greater) x() (res interface{}) {
    if n := len(ctx.evocation.a); n != 2 {
        erro(ctx, "wrong number of arguments, try: $(greater <value-list>,<value-list>)")
    } else if cmp := ctx.evocation.a[0].cmp(ctx, ctx.evocation.a[1]); cmp == cmpGreater {
        res = makeBoolean(ctx.Position(), true)
    }
    return
}

type builtin_less struct { builtin_ }
func (ctx *builtin_less) x() (res interface{}) {
    if n := len(ctx.evocation.a); n != 2 {
        erro(ctx, "wrong number of arguments, try: $(less <value-list>,<value-list>)")
    } else if cmp := ctx.evocation.a[0].cmp(ctx, ctx.evocation.a[1]); cmp == cmpSmaller {
        res = makeBoolean(ctx.Position(), true)
    }
    return
}

// $(match val1 val2 val3, a b c d...)
// $(match -rx=r1 -rx=r2 -rx=r3, a b c d...)
type builtin_match struct { builtin_
    regexps []*regexp.Regexp //`re,rx,reg,regex,regexp`
    negated bool `ne,neg,negated,negative,not`
    all bool `all`
}
func (ctx *builtin_match) x() (result interface{}) {
    var leftList, rightList []Value
    if n := len(ctx.evocation.a); n < 2 {
        erro(ctx, "wrong arguments, try: $(match <regexp-list>,<value-list-1>,...)").debug(1)
        return
    }

    if true {
        leftList, rightList = xmerge(ctx, ctx.evocation.a[0]), xmerge(ctx, ctx.evocation.a[1:]...)
    } else {
        leftList, rightList = merge(ctx.evocation.a[0]), merge(ctx.evocation.a[1:]...)
    }

    var res *boolean

    if ctx.negated { defer func() {
        if res != nil {
            res.bool = !res.bool
        } else {
            result = makeBoolean(ctx.Position(), true)
        }
    }()}

    for _, left := range leftList {
        for _, right := range rightList {
            var matched bool
            if !left.patterned(ctx) && right.patterned(ctx) {
                matched, _, _ = right.match(ctx, left)
            } else {
                matched, _, _ = left.match(ctx, right)
            }
            if matched {
                if res == nil { res = makeBoolean(ctx.Position(), true) }
                if !ctx.all { return res }
            } else if ctx.all {
                res = nil
                return res
            }
        }
    }

    if res != nil { result = res }
    return
}
func (ctx *builtin_match) _x() (res interface{}) {
    var patList, valList []Value
    if n := len(ctx.evocation.a); n < 1 {
        erro(ctx, "wrong arguments, try: $(match <regexp-list>,<value-list>,...)").debug(1)
        return
    }

    if len(ctx.evocation.a) > 1 {
        patList = merge(ctx.evocation.a[0])
        valList = merge(ctx.evocation.a[1:]...)
    } else {
        valList = merge(ctx.evocation.a[0])
    }
    if ctx.debug > 0 {
        var ( n = len(ctx.evocation.a) ; d = ctx.debug )
        note(ctx, "match: %v %v %v, %d", ctx.regexps, patList, valList, n).debug(d)
    }

    var pos = ctx.Position()
ForValList:
    for _, val := range valList {
        if isTrivial(val) { continue ForValList }

        var str = val.string(ctx)
        for _, rx := range ctx.regexps {
            var matched = rx.MatchString(str);
            if ctx.negated { matched = !matched }
            if matched {
                if ctx.all {
                    if res == nil { res = makeBoolean(pos, true) }
                } else {
                    return makeBoolean(pos, true)
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
                    if res == nil { res = makeBoolean(pos, true) }
                } else {
                    return makeBoolean(pos, true)
                }
            } else if ctx.all {
                return nil
            }
        }

        if ctx.debug > 0 {
            note(ctx, "match: %v", str)
            note(ctx, "match: %v %T", val, val).debug(1)
        }
    }
    return
}

// 1: $(case     (a 'xxx') (b 'yyy') (c 'zzz') (yes 'else'))
// 2: $(case val (a 'xxx') (b 'yyy') (c 'zzz') ('if none or nil'))
// 3: $(case val (a 'xxx') (b 'yyy') (c 'zzz') (- 'if none or nil'))
// 4: $(case val (a 'xxx') (b 'yyy') (c -) (- -))
type builtin_case struct { builtin_ }
func (ctx *builtin_case) a() (skip bool) { return }
func (ctx *builtin_case) x() (res interface{}) {
    var val Value
    var args = merge(ctx.evocation.a...)
    if len(args) == 0 { return } else
    if _, ok := args[0].(*group); !ok {
        val = args[0].expand(ctx)
        args = args[1:]
    }

    var def []Value
    for _, arg := range args { if g, y := arg.(*group); y && len(g.elems)>0 {
        if n := len(g.elems); val != nil && isNone(val) && n == 1 {
            return g.elems[0]
        } else if n == 1 {
            def = append(def, g.elems[0])
            continue
        }

        var collect bool
        var v = g.elems[0].expand(ctx)
        if val == nil && v != nil && v.true(ctx) {
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
        for _, v := range g.elems[1:] { if f, y := v.(flag); !y || isNull(f.Value) {
            vals = append(vals, v)
        }}
        return vals
    } else {
        erro(at(ctx,arg), "unexpected case: %T %v", arg, arg).debug(1)
        return
    }}
    return
}
// $(if cond, true-value, else-value, ...)
type builtin_if struct { builtin_ }
func (ctx *builtin_if) a() (skip bool) {
    for i, a := range ctx.evocation.a {
        if a = a.expand(ctx); i == 0 {
            skip = indeterminate(ctx, a)
        }
        ctx.evocation.a[i] = a
    }
    return
}
func (ctx *builtin_if) x() (res interface{}) {
    if n := len(ctx.evocation.a); n > 1 {
        if checkpoints {
            if !_exFinal(ctx) && indeterminate(ctx, ctx.evocation.a[0]) {
                erro(ctx, "should skip: %v; %v", ctx.evocation.a[0], us(ctx)).debug(3)
            }
        }
        if ctx.evocation.a[0].true(ctx) {
            return ctx.evocation.a[1].expand(ctx)
        } else if n > 2 {
            return expand(ctx, ctx.evocation.a[2:]...)
        }
    }
    return
}

type builtin_ifeq struct { builtin_
    final bool `final`
}
func (ctx *builtin_ifeq) a() (skip bool) {
    for i, a := range ctx.evocation.a {
        if a = a.expand(ctx); i == 0 {
            skip = indeterminate(ctx, a)
        }
        ctx.evocation.a[i] = a
    }
    return
}
func (ctx *builtin_ifeq) x() (res interface{}) {
    if n := len(ctx.evocation.a); n > 2 {
        if a, b := ctx.evocation.a[0], ctx.evocation.a[1]; equal(ctx, a, b, ctx.final) {
            res = ctx.evocation.a[2]
        } else if n > 3 {
            res = ctx.evocation.a[3:]
        }
    }
    return
}

type builtin_ifne struct { builtin_
    final bool `final`
}
func (ctx *builtin_ifne) a() (skip bool) {
    for i, a := range ctx.evocation.a {
        if a = a.expand(ctx); i == 0 {
            skip = indeterminate(ctx, a)
        }
        ctx.evocation.a[i] = a
    }
    return
}
func (ctx *builtin_ifne) x() (res interface{}) {
    if n := len(ctx.evocation.a); n > 2 {
        if a, b := ctx.evocation.a[0], ctx.evocation.a[1]; !equal(ctx, a, b, ctx.final) {
            res = ctx.evocation.a[2]
        } else if n > 3 {
            res = ctx.evocation.a[3:]
        }
    }
    return
}

// $(for x=(a b c),$(x))
type builtin_for struct { builtin_
    empty bool `empty,allow-empty`
}
func (ctx *builtin_for) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtin_.cast(t)
}
func (ctx *builtin_for) x() (res interface{}) {
    erro(ctx, "TODO: $(for): %v", us(ctx.evocation.a)).debug(1)
    return
}

// $(foreach a b c,$_)
type builtin_foreach struct { builtin_
    empty     bool `empty,allow-empty`
    unique    bool `unique`
    x_closure bool `x-closure`
    x_values  bool `x-values`
}
func (ctx *builtin_foreach) cast(t reflect.Type) Context {
    if reflect.TypeOf(partial{}) == t { return nil }
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtin_.cast(t)
}
func (ctx *builtin_foreach) a() (skip bool) {
    for i, a := range ctx.evocation.a {
        if i == 0 {
            if b := a.expand(ctx); equal(ctx, a, b) {
                skip = indeterminate(ctx, b)
            } else {
                a = b
            }
        } else {
            a = a.expand(partial{ctx, digitalPart})
        }
        ctx.evocation.a[i] = a
    }
    return
}
func (ctx *builtin_foreach) x() (res interface{}) {
    if len(ctx.evocation.a) < 2 { return }

    var values []Value
    if a := ctx.evocation.a[0]; ctx.x_values {
        values = xmerge(ctx, a)
    } else {
        values = merge(a)
    }
    if len(values) == 0 { return }
    if ctx.unique { values = unique(ctx, values...) }

    var cc = automatic{ Context:ctx, defs:make(autodefs),
        suppress:func(s string) bool { return s == "_" }}

    var vals []Value
    for _, val := range values {
        if isEmpty(val) {
            if !ctx.empty { continue }
        } else if indeterminate(ctx, val) {
            val = disjunction{val}
        }

        cc.set(ctx, "_", val)

        for _, v := range detachBarecompList(xmerge(&cc, ctx.evocation.a[1:]...)...) {
            if isEmpty(v) {
                if ctx.empty {
                    if v == nil { v = makeNull(v.Position()) }
                    vals = append(vals, v)
                }
            } else {
                if !cond(v) && indeterminate(ctx, v) { v = condish(ctx, v) }
                vals = append(vals, v)
            }
        }
    }
    return vals
}

type builtin_count struct { builtin_
    vals []Value `value`
}
func (ctx *builtin_count) x() (res interface{}) {
    var num int64
    var vals = valvec(ctx.vals)
    for _, a := range ctx.evocation.a { if a.true(ctx) || vals.has2(ctx, a) {
        num += 1
    }}
    return num
}

type builtin_env struct { builtin_ }
func (ctx *builtin_env) x() (res interface{}) {
    var vals []Value
    for _, a := range ctx.evocation.a { if val := a.expand(ctx); isTrivial(val) {
        continue
    } else if s := strings.TrimSpace(val.string(ctx)); s != "" {
        vals = append(vals, makeStrlit(a.Position(), os.Getenv(s)))
    }}
    return vals
}

type builtin_auto struct { builtin_ }
func (ctx *builtin_auto) a() (skip bool) {
    if len(ctx.evocation.a) == 0 {
        // noop
    } else if false {
        ctx.evocation.a = expand(ctx, ctx.evocation.a...)
    } else if x := (negate{ctx, propExPairVal}); false {
        ctx.evocation.a = append(expand(x, ctx.evocation.a[0]), expand(ctx, ctx.evocation.a[1:]...)...)
    } else if ctx.evocation.a[0] != nil {
        ctx.evocation.a[0] = ctx.evocation.a[0].expand(x)
    }
    return false // never skips
}
func (ctx *builtin_auto) x() (res interface{}) {
    if 1 < len(ctx.evocation.a) {
        var ac = automatic{ Context:ctx, defs:make(autodefs),
            suppress: func(string) bool { return false } }

        for _, a := range merge(ctx.evocation.a[0]) {
            switch t := a.(type) {
            case *pair:
                if s := t.key.string(&ac); s == "" {
                    erro(at(ctx,t.key), "%v is empty for name", us(t.key)).debug(1)
                } else {
                    ac.set(ctx, s, t.val)
                }
            default:
                erro(at(ctx,a), "%v is unsupported for auto", us(a)).debug(1)
            }
        }

        res = expand(&ac, ctx.evocation.a[1:]...)
    }
    return
}

type builtin_var struct { builtin_ }
func (ctx *builtin_var) a() (skip bool) {
    erro(ctx, "%v", ctx.evocation.a).debug(6)
    return
}
func (ctx *builtin_var) x() (res interface{}) {
    erro(ctx, "%v", ctx.evocation.a).debug(6)
    return
}

// $(value <name>,...)
type builtin_value struct { builtin_ }
func (ctx *builtin_value) x() (res interface{}) {
    var vals []Value
    var p = ctx.project()
    for _, a := range merge(ctx.evocation.a...) {
        var v Value

        if s := argstring(ctx, a); s != "" {
            if d := p.resolveDef(ctx, s); d != nil { v = d.value }
            if v == nil { v = autoVal(ctx, s) }
        }

        if v == nil { v = makeDelegate(ctx.Position(), LPAREN, a, nil) }
        if v != nil { vals = append(vals, v) }
    }
    return vals
}

// $(closure <name>,...)
type builtin_closure struct { builtin_ }
func (ctx *builtin_closure) x() (res interface{}) {
    var vals []Value
    for _, a := range merge(ctx.evocation.a...) {
        var v Value

        if s := argstring(ctx, a); s != "" {
            if d := closureGet(ctx, s); d != nil { v = d.value }
        }

        if v == nil { v = makeClosure(a.Position(), LPAREN, a, nil) }
        if v != nil { vals = append(vals, v) }
    }
    return vals
}

// $(call <name>, <arg>,...)
type builtin_call struct { builtin_ ; _closure bool `closure` }
func (ctx *builtin_call) a() (skip bool) {
    if 0 < len(ctx.evocation.a) {
        var a = expand(ctx, ctx.evocation.a...)
        skip = indeterminate(ctx, a[0])
        ctx.evocation.a = a
    }
    return
}
func (ctx *builtin_call) x() (res interface{}) {
    if 0 < len(ctx.evocation.a) {
        var o Object
        if s := ctx.evocation.a[0].string(ctx); s == "" {
            erro(ctx, "%v is empty for name", us(ctx.evocation.a[0])).debug(1)
            return
        } else if ctx._closure {
            o = closureResolve(ctx, s)
        } else {
            o = resolve(ctx, s)
        }
        if a := ctx.evocation.a[1:]; o == nil {
            return skip{}
        } else {
            if res, _ = evoke(ctx, o, nil, a); res == nil {
                return
            }
            if r, y := res.(Value); y && equal(ctx, r, o) {
                return skip{}
            }
            return
        }
    }
    return
}

type builtin_defs struct { builtin_
    n int `num,number`
    r int `capture`
}
func (ctx *builtin_defs) x() (res interface{}) {
    var find = func(pat Value) (res []bare) {
        var neg bool
        if x, y := pat.(negative); y { pat, neg = x.Value, y }
        for name, _ := range ctx.project().scope.elems {
            var a, _, c = pat.match(ctx, name)
            if a {
                if neg {
                    continue
                } else if ctx.r <= 0 {
                    res = append(res, bare(name))
                } else if ctx.r <= len(c) {
                    res = append(res, bare(c[ctx.r-1]))
                }
            } else if neg {
                // NOTE: regexs match always yields nil for `c`
                if ctx.r <= 0 || 0 == len(c) {
                    res = append(res, bare(name))
                } else if ctx.r <= len(c) {
                    res = append(res, bare(c[ctx.r-1]))
                }
            }
        }
        return
    }

    var names []bare
    for _, val := range merge(ctx.evocation.a...) {
        if indeterminate(ctx, val) {
            erro(ctx, "indeterminate name pattern: %v", us(val)).debug(1)
        } else {
            names = append(names, find(val)...)
        }
    }
    return names
}

type builtin_list struct { builtin_ }
func (ctx *builtin_list) x() (res interface{}) {
    return ctx.evocation.a
}

type builtin_plain struct { builtin_
    scope bool `findscope,find-scope,scope`
}
func (ctx *builtin_plain) c() (res interface{}) {
    var scope = ctx.Scope()
    for _, a := range ctx.evocation.a {
        var ( o Object ; s = a.string(ctx) )
        if ctx.scope { _, o = scope.find(s) } else { o = resolve(ctx, s) }
        if o == nil {
            erro(at(ctx,a), "no such symbol: %s", s).debug(1)
        } else if d, y := o.(*def); !y {
            erro(at(ctx,a), "not a def: %s: %v", s, typeof(o)).debug(1)
        } else if d.value != nil {
            d.value = d.value.expand(ctx/* , plain */)
        }
    }
    return
}

type builtin_shell struct { builtin_ }
func (ctx *builtin_shell) x() (res interface{}) {
    var (
        pos = ctx.Position()
        vals []Value
        err error
    )
    for _, a := range ctx.evocation.a {
        var bufout, buferr bytes.Buffer
        var s = a.string(ctx)
        sh := exec.Command("sh", "-c", s)
        sh.Stdout, sh.Stderr = &bufout, &buferr
        if err = sh.Run(); err != nil {
            s = strings.TrimSpace(buferr.String())
            if !strings.HasPrefix(s, ":") { s = ":\n" + s }
            prompt(ctx, "%s%s\n", a.string(ctx), s)
            errostack(ctx, 3, "%s", err).debug(10)
            panic(_failure(ctx, "%v", err))
            return
        }
        val := makeStrlit(pos, strings.TrimSpace(bufout.String()))
        vals = append(vals, val)
        bufout.Reset()
        buferr.Reset()
    }
    return vals
}

type builtin_which struct { builtin_ }
func (ctx *builtin_which) x() (res interface{}) {
    var vals []Value
    for _, a := range ctx.evocation.a {
        if s, err := exec.LookPath(a.string(ctx)); err != nil {
            erro(ctx, "%v", err).debug(1)
            return
        } else if s != "" {
            vals = append(vals, makeStrlit(ctx.Position(), s))
        }
    }
    return vals
}

type builtin_servehttp struct { builtin_
    ssl bool `ssl`
    host string `host`
    port int `port`
}
func (ctx *builtin_servehttp) c() (res interface{}) {
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
            server.Shutdown(_context.Background())
        } ()
    }

    http.HandleFunc("/-/end",  quit)
    http.HandleFunc("/-/quit", quit)
    http.HandleFunc("/-/shut", quit)

    if ctx.evocation.a == nil {
        http.Handle("/", http.FileServer(http.Dir(_workdir(ctx))))
    } else {
        for _, a := range ctx.evocation.a {
            var s = a.string(ctx)
            info(ctx, "serving files %v ...", s)
            http.Handle("/", http.FileServer(http.Dir(s)))
        }
    }

    flush(ctx)

    var err = server.ListenAndServe()
    if err != nil && err != http.ErrServerClosed { erro(ctx, "%s", err).debug(1) }
    return
}

type builtin_append struct { builtin_
    _auto    bool `auto`
    _closure bool `closure`
}
func (ctx *builtin_append) x() (_ interface{}) {
    if len(ctx.evocation.a) < 2 {
        erro(ctx, "insufficient number of arguments: %v", ctx.evocation.a).debug(1)
        return
    }

    var names []Value
    if names = merge(ctx.evocation.a[0]); len(names) == 0 {
        warn(ctx, "append to nowhere: %v", tv(ctx.evocation.a[0])).debug(1)
        return
    }

    var vals []Value
    for _, a := range names {
        var s = a.string(ctx)
        var d *def
        if s == "" {
            erro(at(ctx,a), "'%v' is empty for name", a).debug(1)
        } else if ctx._auto {
            d = autoDef(ctx, s)
        } else if ctx._closure {
            d = closureGet(ctx, s)
        } else if o := resolve(ctx, s); o != nil {
            d, _ = o.(*def)
        }
        if d == nil {
            erro(at(ctx,a), "%v → %s is undefined", a, s)
            erro(ctx, "%v", us(ctx)).debug(1)
        } else {
            if vals == nil {
                if vals = merge(ctx.evocation.a[1:]...); len(vals) == 0 {
                    warn(ctx, "append no values: %v", ctx.evocation.a[1:]).debug(1)
                    return
                }
            }
            d.append(ctx, vals...)
        }
    }
    return
}

type builtin_plus struct { builtin_
    int bool `int,integer`
}
func (ctx *builtin_plus) x() (res interface{}) {
    if ctx.int {
        var num int64
        for n, a := range ctx.evocation.a {
            if i, e := a.int(ctx); e == nil {
                if n == 0 { num = i } else { num += i }
            } else {
                erro(ctx, "%v: %v", a, e).debug(1)
            }
        }
        return makeDecimal(ctx.Position(), num)
    } else {
        var num float64
        for n, a := range ctx.evocation.a {
            if f, e := a.float(ctx); e == nil {
                if n == 0 { num = f } else { num += f }
            } else {
                erro(ctx, "%v: %v", a, e).debug(1)
            }
        }
        return makeFloat(ctx.Position(), num)
    }
}

type builtin_minus struct { builtin_
    int bool `int,integer`
}
func (ctx *builtin_minus) x() (res interface{}) {
    if ctx.int {
        var num int64
        for n, a := range ctx.evocation.a {
            if i, e := a.int(ctx); e == nil {
                if n == 0 { num = i } else { num -= i }
            } else {
                erro(ctx, "%v: %v", a, e).debug(1)
            }
        }
        return makeDecimal(ctx.Position(), num)
    } else {
        var num float64
        for n, a := range ctx.evocation.a {
            if f, e := a.float(ctx); e == nil {
                if n == 0 { num = f } else { num -= f }
            } else {
                erro(ctx, "%v: %v", a, e).debug(1)
            }
        }
        return makeFloat(ctx.Position(), num)
    }
}

type builtin_multiply struct { builtin_
    int bool `int,integer`
}
func (ctx *builtin_multiply) x() (res interface{}) {
    if ctx.int {
        var num int64
        for n, a := range ctx.evocation.a {
            if i, e := a.int(ctx); e == nil {
                if n == 0 { num = i } else { num *= i }
            } else {
                erro(ctx, "%v: %v", a, e).debug(1)
            }
        }
        return num
    } else {
        var num float64
        for n, a := range ctx.evocation.a {
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
    int bool `int,integer`
}
func (ctx *builtin_divide) x() (res interface{}) {
    if ctx.int {
        var num int64
        for n, a := range ctx.evocation.a {
            if i, e := a.int(ctx); e == nil {
                if n == 0 { num = i } else { num /= i } // FIXME: NaN
            } else {
                erro(ctx, "%v: %v", a, e).debug(1)
            }
        }
        return num
    } else {
        var num float64
        for n, a := range ctx.evocation.a {
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
    reverse bool `rev,reverse`
    keepAuto bool `auto,keepauto,keep-auto`
    unexpand bool `un,ue,unexpand,ne,noexpand,no-expand`
    plain bool `pl,pla,plain,pv,plainvalue,plain-value`
}
func (ctx *builtin_unique) x() (res interface{}) {
    var args = ctx.evocation.a
    if ctx.unexpand {
        args = merge(args...)
    } else if ctx.plain {
        args = xmerge(final{ctx}, args...)
    } else {
        args = xmerge(ctx, args...)
    }
    if ctx.reverse {
        return reverse_unique(ctx, args...)
    } else {
        return unique(ctx, args...)
    }
}

type builtin_join struct { builtin_ }
func (ctx *builtin_join) a() (skip bool) {
    var a = expand(ctx, ctx.evocation.a...)
    skip = expandable(final{ctx}, a...)
    ctx.evocation.a = a
    return
}
func (ctx *builtin_join) x() (res interface{}) {
    if l := len(ctx.evocation.a); 0 < l {
        var fields []string
        var vals []Value
        var sep Value
        if l < 2 {
            vals = merge(ctx.evocation.a...)
        } else {
            vals = merge(ctx.evocation.a[:l-1]...)
            sep = scalarize(ctx.evocation.a[l-1])
        }
        if len(vals) == 0 { return }

        var s string
        if sep != nil { s = sep.string(ctx) }
        for _, a := range vals {
            if v := a.string(ctx); v != "" { fields = append(fields, v) }
        }

        res = makeStrlit(ctx.Position(), strings.Join(fields, s))
    }
    return
}

type builtin_compose struct { builtin_ }
func (ctx *builtin_compose) a() (skip bool) {
    var a = expand(ctx, ctx.evocation.a...)
    // skip = !ctx.compose && expandable(final{ctx}, a...)
    ctx.evocation.a = a
    return
}
func (ctx *builtin_compose) x() (res interface{}) {
    if l := len(ctx.evocation.a); 0 < l {
        var con conjunction
        if l < 2 {
            con.list = makeList(merge(ctx.evocation.a...)...)
        } else {
            con.list = makeList(merge(ctx.evocation.a[:l-1]...)...)
            con.sep  = ctx.evocation.a[l-1]
        }
        return con
    }
    return
}

type builtin_quote struct { builtin_ }
func (ctx *builtin_quote) x() (res interface{}) {
    var args = merge(ctx.evocation.a...)
    if l := len(args); l > 0 {
        var fields []string
        for _, a := range args {
            if v := a.string(ctx); v != "" { fields = append(fields, v) }
        }
        res = makeStrlit(ctx.Position(), strconv.Quote(strings.Join(fields, " ")))
    } else {
        res = makeNone(ctx.Position())
    }
    return
}

type builtin_quotejoin struct { builtin_ }
func (ctx *builtin_quotejoin) x() (res interface{}) {
    var sep string
    var args = merge(ctx.evocation.a...)
    if l := len(args); l > 1 {
        sep = args[l-1].string(ctx)
        args = args[:l-1]
    }
    if l := len(args); l > 0 {
        var fields []string
        for _, a := range args[1:] {
            if v := a.string(ctx); v != "" { fields = append(fields, v) }
        }
        res = makeStrlit(ctx.Position(), strconv.Quote(strings.Join(fields, sep)))
    } else {
        res = makeNone(ctx.Position())
    }
    return
}

// $(split-string .,1.2.3)
type builtin_splitstring struct { builtin_
    sep string `sep,separator`
}
func (ctx *builtin_splitstring) x() (res interface{}) {
    var fields []Value
    if len(ctx.evocation.a) > 0 {
        var sep = ctx.sep
        if sep == "" { sep = ctx.evocation.a[0].string(ctx) }
        for _, a := range ctx.evocation.a[1:] { for _, s := range strings.Split(a.string(ctx), sep) {
            fields = append(fields, makeStrlit(a.Position(), s))
        }}
    }
    return fields
}

func quotestrings(value Value) {
    switch v := value.(type) {
    case *strlit: v.s = strconv.Quote(v.s)
    case *list:
        for _, elem := range v.elems {
            quotestrings(elem)
        }
    }
    return
}

func joinstrings(ctx Context, value Value, sep string) (res Value, err error) {
    if sep == "" { sep = " " }
ValueType:
    switch v := value.(type) {
    case *strlit: res = value
    case *list:
        var strs []string
        for _, elem := range v.elems {
            var ( v Value; s string )
            if v, err = joinstrings(ctx, elem, sep); err != nil { break ValueType }
            if s = v.string(ctx); s != "" { strs = append(strs, s) }
        }
        res = makeStrlit(value.Position(), strings.Join(strs, sep))
    }
    return
}

// TODO: deprecate this and add -quote to builtin_splitstring
type builtin_splitquote struct { builtin_splitstring }
func (ctx *builtin_splitquote) x() (res interface{}) {
    res = ctx.builtin_splitstring.x()
    if v, y := res.(Value); y && v != nil { quotestrings(v) }
    return
}

// TODO: deprecate this and add -quote to builtin_splitstring
type builtin_splitquotejoin struct { builtin_splitstring }
func (ctx *builtin_splitquotejoin) x() (res interface{}) {
    res = ctx.builtin_splitstring.x()
    if val, y := res.(Value); y && val != nil {
        var err error
        var sep string
        if l := len(ctx.evocation.a); l > 1 {
            sep = ctx.evocation.a[l-1].string(ctx)
            ctx.evocation.a = ctx.evocation.a[:l-1]
        }
        if res, err = joinstrings(ctx, val, sep); err != nil {
            erro(ctx, "%v", err).debug(1)
        }
    }
    return
}

type builtin_splitjoinquote struct { builtin_splitstring }
func (ctx *builtin_splitjoinquote) x() (res interface{}) {
    res = ctx.builtin_splitstring.x()
    if val, y := res.(Value); y && val != nil {
        var err error
        var sep string
        if l := len(ctx.evocation.a); l > 1 {
            sep = ctx.evocation.a[l-1].string(ctx)
            ctx.evocation.a = ctx.evocation.a[:l-1]
        }

        var v Value
        if v, err = joinstrings(ctx, val, sep); err != nil {
            erro(ctx, "%v", err).debug(1)
        } else {
            res = makeStrlit(ctx.Position(), strconv.Quote(v.string(ctx)))
        }
    }
    return
}

type builtin_field struct { builtin_ }
func (ctx *builtin_field) x() (res interface{}) {
    var fields []string
    if l := len(ctx.evocation.a); l >= 2 {
        var (
            s string = ctx.evocation.a[1].string(ctx)
            i int64
        )
        if n, e := ctx.evocation.a[0].int(ctx); e != nil {
            erro(ctx, "%v: %v", ctx.evocation.a[0], e).debug(1)
            return
        } else { i = n }

        if l > 2 {
            fields = strings.Split(s, ctx.evocation.a[2].string(ctx))
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
func (ctx *builtin_fields) x() (res interface{}) {
    // TODO: ...
    return
}

type builtin_usee struct { builtin_ }
func (ctx *builtin_usee) x() (res interface{}) {
    var proj = ctx.project()
    if proj == nil {
        erro(ctx, "unknown current context").debug(1)
        return
    }

    var vals []Value
    for _, a := range ctx.evocation.a {
        v := proj.use.get(ctx, a.string(ctx))
        if v != nil { vals = append(vals, v) }
    }
    if vals == nil { res = vals }
    return
}

type builtin_uses struct { builtin_ }
func (ctx *builtin_uses) x() (res interface{}) {
    var proj = ctx.project()
    if proj == nil {
        erro(ctx, "unknown current context").debug(1)
        return
    }

    var found bool

outer:
    for _, a := range ctx.evocation.a {
        var s = a.string(ctx)
        for _, u := range proj.use.list {
            found = u.project.name == s
            if found { break outer }
        }
    }

    if found { res = found }
    return
}

type builtin_path struct { builtin_ }
func (ctx *builtin_path) x() interface{} {
    var res []Value
    for _, a := range ctx.evocation.a {
        if x, y := a.(*path); y {
            res = append(res, x)
        } else {
            res = append(res, _pathstr(ctx, a.string(ctx)))
        }
    }
    return res
}

type builtin_bare struct { builtin_
    name bool `name,filename,file-name,non-full,not-full`
}
func (ctx *builtin_bare) x() (res interface{}) {
    var vals []Value
    for _, a := range ctx.evocation.a {
        switch p := a.Position(); t := a.(type) {
        case *strlit, *compound:
            a = makeBareword(p, a.string(ctx))
        case *File:
            a = makeBareword(p, t.ident(ctx))
        case fullfile:
            if ctx.name {
                a = makeBareword(p, t.ident(ctx))
            } else {
                a = makeBareword(p, t.string(ctx))
            }
        }
        vals = append(vals, a)
    }
    return vals
}

type builtin_bareword struct { builtin_ }
func (ctx *builtin_bareword) x() (res interface{}) {
    var vals []Value
    for _, a := range ctx.evocation.a {
        if _, y := a.(*bareword); !y {
            a = makeBareword(a.Position(), a.string(ctx))
        }
        vals = append(vals, a)
    }
    return vals
}

type builtin_string struct { builtin_
    finalize bool //`final` // not returning strval
    expand bool `ex,exp,expand`
    name   bool `name,file-name,non-full`
    dis    bool `disjunct,disjunction`
    con    bool `conjunct,conjunction` // default
    clo  []string `clo,closure`
    def  []string `def,var`
    join []string `join`
}
func (ctx *builtin_string) a() (skip bool) {
    ctx.evocation.a = expand(ctx, ctx.evocation.a...)
    return
}
func (ctx *builtin_string) x() (res interface{}) {
    if 0 < len(ctx.evocation.a)+len(ctx.clo)+len(ctx.def) {
        var vals []Value
        for _, name := range ctx.clo {
            if o := closureResolve(ctx, name); o != nil {
                if d, y := o.(*def); y && d != nil && d.value != nil {
                    vals = append(vals, d.value)
                }
            }
        }
        for _, name := range ctx.def {
            if o := resolve(ctx, name); o != nil {
                if d, y := o.(*def); y && d != nil && d.value != nil {
                    vals = append(vals, d.value)
                }
            }
        }

        vals = merge(append(vals, ctx.evocation.a...)...)

        if ctx.finalize {
            return expand(final{ctx}, vals...)
        } else if expandable(final{ctx}, vals...) {
            return &strval{valbase{ctx.Position()},vals}
        } else if 0 < len(ctx.join) {
            var s bytes.Buffer
            for i, v := range vals {
                if t := v.string(ctx); t != "" {
                    if 0 < i { s.WriteString(ctx.join[i % len(ctx.join)]) }
                    s.WriteString(t)
                }
            }
            return &strlit{valbase{ctx.Position()},s.String()}
        } else if ctx.con || !ctx.dis { // conjunction (default)
            var s bytes.Buffer
            for i, v := range vals {
                if t := v.string(ctx); t != "" {
                    if 0 < i { s.WriteString(" ") }
                    s.WriteString(t)
                }
            }
            return &strlit{valbase{ctx.Position()},s.String()}
        } else { // disjunction
            var a []Value
            for _, v := range vals {
                if t := v.string(ctx); t != "" {
                    a = append(a, &strlit{valbase{v.Position()},t})
                }
            }
            return a
        }
    }
    return
}

type builtin_finalize struct { builtin_string }
func (ctx *builtin_finalize) x() interface{} {
    ctx.finalize = true ; return ctx.builtin_string.x()
}

type builtin_filter struct { builtin_
    stem bool `stem`
    neg bool
}
func (ctx *builtin_filter) _do(pats []Value, values... Value) (result []Value) {
    defer func(t0 time.Time) { if d := time.Now().Sub(t0); d > 1*time.Second {
        pos := ctx.Position()
        prompt(ctx, "%v: slow: %d result, %v\n", pos, len(result), result)
        prompt(ctx, "%v: slow: %d pats, %v\n", pos, len(pats), pats)
        prompt(ctx, "%v: slow: %v\n", pos, d).debug(4)
    }} (time.Now())

    var f = func(v Value) Value {
        for _, pat := range pats { if full, res, stems := pat.match(ctx, v); full {
            if ctx.neg { v = nil } else if ctx.stem {
                var vals []Value
                for _, s := range stems {
                    vals = append(vals, makeStrlit(v.Position(), s))
                }
                v = ease(ctx, vals)
            } else if true {
                // 'v' is just good enough
            } else if t, r := pat.stencil(ctx, stems); t != nil && len(r) == 0 {
                v = t
            } else if s, y := res.(string); y {
                v = makeStrlit(v.Position(), s)
            } else if a, y := res.([]string); y {
                var vals []Value
                for _, s := range a {
                    vals = append(vals, makeStrlit(v.Position(), s))
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
func (ctx *builtin_filter) x() (res interface{}) {
    if len(ctx.evocation.a) > 1 {
        var i int
        var vals []Value
        var pats = merge(ctx.evocation.a[0])
        if len(pats) > 0 {
            i = 1 // good
        } else if pats = merge(ctx.evocation.a[1]); len(pats) == 0 {
            erro(ctx, "no patterns: %v", ctx.evocation.a).debug(1)
            return
        } else {
            i = 2
        }

        if len(ctx.evocation.a) <= i {
            erro(ctx, "out of index: %d %v", i, ctx.evocation.a).debug(1)
            return
        }

        vals = merge(ctx.evocation.a[i:]...)
        vals = ctx._do(pats, vals...)
        if len(vals) > 0 { res = vals }
    }
    return
}

// $(filter-out pattern…,text)
type builtin_filterout struct { builtin_filter }
func (ctx *builtin_filterout) _do(pats []Value, values... Value) (result []Value) { ctx.neg = true
    return ctx.builtin_filter._do(pats, values...)
}
func (ctx *builtin_filterout) x() (res interface{}) { ctx.neg = true
    return ctx.builtin_filter.x()
}

type builtin_substring struct { builtin_ }
func (ctx *builtin_substring) x() (_ interface{}) {
    var res []Value
    if n := len(ctx.evocation.a); n > 1 {
        var a = intVal(ctx, ctx.evocation.a[0], -1)
        var b = intVal(ctx, ctx.evocation.a[1], -1)
        if ctx.evocation.a = ctx.evocation.a[2:]; a < -1 && b < -1 {
            erro(ctx, "wrong indices (%v, %v)", ctx.evocation.a[0], ctx.evocation.a[1]).debug(1)
            return
        }
        if a > b { t := a; a = b; b = t } // swap the wrong order
        if a == -1 { a = b }
        if a == -1 { return }

        for _, arg := range ctx.evocation.a {
            var s = arg.string(ctx)
            if i := len(s); i <= a { s = "" } else
            if b == -1 || i <= b { s = s[a:b] } else { s = s[a:] }
            res = append(res, makeStrlit(arg.Position(), s))
        }
    }
    return res
}

// $(subst from,to,text)
type builtin_subst struct { builtin_ }
func (ctx *builtin_subst) x() (_ interface{}) {
    var res []Value
    if len(ctx.evocation.a) > 2 {
        var s1 = ctx.evocation.a[0].string(ctx)
        var s2 = ctx.evocation.a[1].string(ctx)
        for _, arg := range merge(ctx.evocation.a[2:]...) {
            s := strings.Replace(arg.string(ctx), s1, s2, -1)
            res = append(res, makeStrlit(arg.Position(), s))
        }
    }
    return res
}

// $(patsubst pattern,replacement,text)
// TODO: supports: $(var:pattern=replacement)
// TODO: supports: $(var:suffix=replacement)
// TODO: support flags -name and -full for name-only and full-name-only matching
type builtin_patsubst struct { builtin_
    findFiles bool `find,find-file`
    fullFiles bool `ff,fullfile,fullfiles`
    cleanPath bool `c,clean,cleanpath`
    nofilemap bool `nomap,no-map,nofile,nofiles,no-files,no-filemap`
    erroDstNomap bool `err-dst-nomap,error-dst-nomap`
    warnDstNomap bool `warn-dst-nomap`
}
func (ctx *builtin_patsubst) x() (_ interface{}) {
    if len(ctx.evocation.a) < 3 {
        erro(ctx, "not enough arguments").debug(1)
        return
    }

    var (
        proj = ctx.project()
        closured = closureprojects(ctx)
        srcPats = merge(ctx.evocation.a[0])
        dstPats, sources, res []Value
        t1 time.Time
    )
    defer func(t0 time.Time) {
        var t2 = time.Now()
        if d := t2.Sub(t0); d > 1*time.Second {
            var d1 = t1.Sub(t0)
            var d2 = t2.Sub(t1)
            var pos = ctx.Position()
            prompt(ctx, "%v: slow: src %d %v\n", pos, len(srcPats), srcPats)
            prompt(ctx, "%v: slow: dst %d %v\n", pos, len(dstPats), dstPats)
            prompt(ctx, "%v: slow: sources %d %v\n", pos, len(sources), sources)
            prompt(ctx, "%v: slow: list %d %v\n", pos, len(res), res)
            prompt(ctx, "%v: slow: %v⇒%v+%v\n", pos, d, d1, d2).debug(4)
        }
    } (time.Now())

    if len(srcPats) == 0 {
        if len(ctx.evocation.a) < 4 {
            erro(ctx, "not enough arguments").debug(1)
            return
        }
        srcPats = merge(ctx.evocation.a[1])
        dstPats = merge(ctx.evocation.a[2])
        sources = merge(ctx.evocation.a[3:]...)
    } else {
        dstPats = merge(ctx.evocation.a[1])
        sources = merge(ctx.evocation.a[2:]...)
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
            var s = src.string(ctx)
            if srcFile = proj.file(ctx, s); srcFile != nil {
                source = srcFile
            } else {
                source = s
            }
        } else if !ctx.fullname {
            source = src
        } else if o, y := (as{src}.fullname(ctx, closured...)); y {
            source = o.string(ctx)
        } else {
            erro(at(ctx,src), "fullname '%v' failed", src)
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

        if !isTrivial(src) { res = append(res, src) }

        continue ForSources // just append src to the list

        // Compose the matched results with stem value.
    stencilTargetPats:
        for _, dstPat := range dstPats {
            var nameVal, ramnant = dstPat.stencil(ctx, stems)
            if isNull(nameVal) {
                erro(ctx, "nil stencil: %T %v (stems=%v, ramnant=%v)", dstPat, dstPat, stems, ramnant).debug(1)
                return
            } else if ctx.debug>0 {
                warnstack(ctx, ctx.debug, "patsubst: %v: %v → %v → %v %v → %v %v",
                    srcPat, src, source, stems, dstPat, nameVal, ramnant).debug(ctx.debug)
            }

            var nameStr string
            if nameStr = nameVal.string(ctx); nameStr == "" {
                continue stencilTargetPats
            } else if ctx.cleanPath {
                nameStr = filepath.Clean(nameStr)
            }

            if srcFile != nil {
                var dstFile *File
                if !ctx.nofilemap { dstFile = proj.file(ctx, nameStr) }
                if dstFile == nil {
                    a := []interface{}{
                        "%v: %v (%v): unmapped destination, aka files (...)",
                        proj, nameStr, dstPat,
                    }
                    if t := files(ctx, nameVal, proj); ctx.erroDstNomap {
                        erro(at(ctx,srcPat), "%v: patsubst: %v (%v) ⇒ %v (%v) ⇒ %v", proj, srcFile, srcPat, nameVal, dstPat, t)
                        errostack(at(ctx,dstPat), 16, a...).debug(16)
                    } else if ctx.warnDstNomap {
                        warn(at(ctx,srcPat), "%v: patsubst: %v (%v) ⇒ %v (%v) ⇒ %v", proj, srcFile, srcPat, nameVal, dstPat, t)
                        warnstack(at(ctx,dstPat), 16, a...).debug(5)
                    }
                    dstFile = stat(ctx, nameStr, stat_sub{srcFile.sub}, stat_dir{srcFile.dir}, stat_nonexist{true})
                }
                if dstFile.position = srcPat.Position(); full {
                    res = append(res, fullfile{dstFile})
                } else {
                    res = append(res, dstFile)
                }
                continue stencilTargetPats
            }

            // Deal with source value types
            switch pos := dstPat.Position(); src.(type) {
            case *File, fullfile:
            case *strlit, *compound:
                res = append(res, makeStrlit(pos, nameStr))
                continue stencilTargetPats
            case *path:
                res = append(res, _pathstr(ctx, nameStr))
                continue stencilTargetPats
            case *bareword, *barecomp:
                if strings.Contains(nameStr, pathSep) {
                    res = append(res, _pathstr(ctx, nameStr))
                } else {
                    res = append(res, makeBareword(pos, nameStr))
                }
                continue stencilTargetPats
            default:
                if strings.Contains(nameStr, pathSep) {
                    res = append(res, _pathstr(ctx, nameStr))
                } else if true {
                    res = append(res, makeBareword(pos, nameStr))
                } else {
                    res = append(res, makeStrlit(pos, nameStr))
                }
                continue stencilTargetPats
            }
        }
    }

    if 0 < ctx.debug && len(res) == 0 {
        warn(ctx, "src: %v", srcPats)
        warn(ctx, "dst: %v", dstPats)
        warn(ctx, "val: %v", sources)
        warn(ctx, "res: %v", res)
        warnstack(ctx, 3, "%v", us(ctx)).debug(ctx.debug)
    }
    return res
}

type builtin_title struct { builtin_ }
func (ctx *builtin_title) x() interface{} {
    var res []Value
    for _, a := range ctx.evocation.a {
        if s := a.string(ctx); s != "" {
            res = append(res, makeStrlit(a.Position(), strings.Title(s)))
        }
    }
    return res
}

type builtin_uppercase struct { builtin_ }
func (ctx *builtin_uppercase) x() interface{} {
    var res []Value
    for _, a := range ctx.evocation.a {
        if s := a.string(ctx); s != "" {
            res = append(res, makeStrlit(a.Position(), strings.ToUpper(s)))
        }
    }
    return res
}

type builtin_lowercase struct { builtin_ }
func (ctx *builtin_lowercase) x() interface{} {
    var res []Value
    for _, a := range ctx.evocation.a {
        if s := a.string(ctx); s != "" {
            res = append(res, makeStrlit(a.Position(), strings.ToLower(s)))
        }
    }
    return res
}

func (ctx *builtin_) trim(f func(_, _ Value, _ string) (Value, string), ss ...string) (_ interface{}) {
    defer trace(ctx)

    if len(ctx.evocation.a) < 1 {
        var s = strings.Join(ss, "")
        erro(ctx, "try $(trim%s <...%s>, <...value>)", s, s).debug(1)
        return
    }

    var prefix = merge(ctx.evocation.a[0])
    var values = merge(ctx.evocation.a[1:]...)

    if len(values) == 0 { return }
    if false && len(prefix) == 0 { return ease(ctx, values) }

    var res []Value
    for _, val := range values {
        if indeterminate(ctx, val) {
            erro(ctx, "indeterminate value: %v", tv(val)).debug(1)
            continue
        }

        var v Value
        var s string
        for _, pre := range prefix {
            if v, s = f(val, pre, s); v != nil { break }
        }
        if v != nil { res = append(res, v) }
    }
    return res
}

type builtin_strip struct { builtin_trimspace }

type builtin_trim struct { builtin_ }
func (ctx *builtin_trim) x() interface{} {
    var res []Value
    var cutset string
    for i, a := range merge(ctx.evocation.a...) {
        if s := a.string(ctx); s != "" {
            if i == 0 {
                cutset = s
            } else if cutset == "" {
                res = append(res, makeStrlit(a.Position(), strings.TrimSpace(s)))
            } else {
                res = append(res, makeStrlit(a.Position(), strings.Trim(s, cutset)))
            }
        }
    }
    return res
}

type builtin_trimspace struct { builtin_trim }
func (ctx *builtin_trimspace) a() (skip bool) {
    var a = expand(ctx, ctx.evocation.a...)
    ctx.evocation.a = append([]Value{_null(ctx)}, a...)
    return
}

type builtin_trimleft struct { builtin_ }
func (ctx *builtin_trimleft) x_0() interface{} {
    var res []Value
    var cutset string
    for i, a := range ctx.evocation.a {
        if s := a.string(ctx); s != "" {
            if i == 0 {
                cutset = s
            } else if cutset == "" {
                res = append(res, makeStrlit(a.Position(), strings.TrimLeftFunc(s, unicode.IsSpace)))
            } else {
                res = append(res, makeStrlit(a.Position(), strings.TrimLeft(s, cutset)))
            }
        }
    }
    return res
}
func (ctx *builtin_trimleft) x() (_ interface{}) {
    return ctx.trim(func(a, b Value, _s string) (res Value, s string) {
        return
    })
}

type builtin_trimright struct { builtin_ }
func (ctx *builtin_trimright) x_0() interface{} {
    var res []Value
    var cutset string
    for i, a := range ctx.evocation.a {
        if s := a.string(ctx); s != "" {
            if i == 0 {
                cutset = s
            } else if cutset == "" {
                res = append(res, makeStrlit(a.Position(), strings.TrimRightFunc(s, unicode.IsSpace)))
            } else {
                res = append(res, makeStrlit(a.Position(), strings.TrimRight(s, cutset)))
            }
        }
    }
    return res
}
func (ctx *builtin_trimright) x() (_ interface{}) {
    return ctx.trim(func(a, b Value, _s string) (res Value, s string) {
        return
    })
}

// $(trim-prefix foo%, fooxxx foo123)
// $(trim-prefix %/foo, xxx/foo/a/b/c)
// $(trim-prefix %%/foo, xxx/yyy/zzz/foo/a/b/c)
type builtin_trimprefix struct { builtin_ }
func (ctx *builtin_trimprefix) x_0() interface{} {
    var res []Value
    var cutset string
    for i, a := range ctx.evocation.a {
        if s := a.string(ctx); s != "" {
            if i == 0 {
                cutset = s
            } else if cutset == "" {
                res = append(res, makeStrlit(a.Position(), strings.TrimLeftFunc(s, unicode.IsSpace)))
            } else {
                res = append(res, makeStrlit(a.Position(), strings.TrimPrefix(s, cutset)))
            }
        }
    }
    return res
}
func (ctx *builtin_trimprefix) x() (_ interface{}) {
    return ctx.trim(func(val, prefix Value, _s string) (res Value, s string) {
        if indeterminate(ctx, prefix) {
            erro(ctx, "indeterminate prefix: %v", tv(prefix)).debug(1)
            return
        }

        var t string
        if prefix.patterned(ctx) {
            var f, r, m = prefix.match(ctx, val)
            if checkpoints {
                if prefix.String() == "/**/testdata/" {
                    var v = val.string(ctx)
                    if f != false {
                        erro(ctx, "%v : %v %v %v", tv(prefix), f, r, m).debug(1)
                    }
                    if x, y := r.([]string); !y || strings.TrimPrefix(v, joinPath(x...)) != "builtins/trimprefix" {
                        erro(ctx, "%v : %v %v %v", tv(prefix), f, r, m).debug(1)
                    }
                    if len(m) != 1 {
                        erro(ctx, "%v : %v %v %v", tv(prefix), f, r, m).debug(1)
                    } else if strings.TrimPrefix(v, "/"+m[0]) != "/testdata/builtins/trimprefix" {
                        note(ctx, "/%v", m[0])
                        note(ctx, "%v", val)
                        erro(ctx, "%v : %v %v %v", tv(prefix), f, r, m).debug(1)
                    }
                }
            }
            if f { return }

            t = _path(ctx, r)
        } else {
            t = prefix.string(ctx)
        }

        if s = _s; s == "" { s = val.string(ctx) }
        if strings.HasPrefix(s, t) {
            _s = strings.TrimPrefix(s, t)
        } else if false {
            _s = strings.TrimLeftFunc(s, unicode.IsSpace)
        }

        switch val.(type) {
        case *path: res = _pathstr(ctx, _s)
        default: res = makeStrlit(val.Position(), _s)
        }
        return
    })
}

type builtin_trimsuffix struct { builtin_ }
func (ctx *builtin_trimsuffix) x_0() interface{} {
    var res []Value
    var cutset string
    for i, a := range ctx.evocation.a {
        if s := a.string(ctx); s != "" {
            if i == 0 {
                cutset = s
            } else if cutset == "" {
                res = append(res, makeStrlit(a.Position(), strings.TrimRightFunc(s, unicode.IsSpace)))
            } else {
                res = append(res, makeStrlit(a.Position(), strings.TrimSuffix(s, cutset)))
            }
        }
    }
    return res
}
func (ctx *builtin_trimsuffix) x() (_ interface{}) {
    return ctx.trim(func(val, suffix Value, _s string) (res Value, s string) {
        if indeterminate(ctx, suffix) {
            erro(ctx, "indeterminate suffix: %v", tv(suffix)).debug(1)
            return
        }

        var t string
        if suffix.patterned(ctx) {
            var f, r, _s = suffix.match(reversal{ctx}, val)
            if checkpoints {
                if suffix.String() == "/testdata/**" {
                    if f != false {
                        erro(ctx, "%v : %v %v %v", tv(suffix), f, r, _s).debug(1)
                    }
                    if x, y := r.([]string); !y || joinPath(x...) != "/testdata/builtins/trimsuffix" {
                        erro(ctx, "%v : %v %v %v", tv(suffix), f, r, _s).debug(1)
                    }
                    if len(_s) != 1 {
                        erro(ctx, "%v : %v %v %v", tv(suffix), f, r, _s).debug(1)
                    } else if _s[0] != "builtins/trimsuffix" {
                        erro(ctx, "%v : %v %v %v", tv(suffix), f, r, _s).debug(1)
                    }
                }
            }
            if f { return }

            t = _path(ctx, r)
        } else {
            t = suffix.string(ctx)
        }

        if s = _s; s == "" { s = val.string(ctx) }
        if strings.HasSuffix(s, t) {
            _s = strings.TrimSuffix(s, t)
        } else if false {
            _s = strings.TrimRightFunc(s, unicode.IsSpace)
        }

        switch val.(type) {
        case *path: res = _pathstr(ctx, _s)
        default: res = makeStrlit(val.Position(), _s)
        }
        return
    })
}

type builtin_trimext struct { builtin_trim
    all bool `all`
    ext []string `ext`
}
func (ctx *builtin_trimext) x() interface{} {
    var ext string
    var res []Value
    for i, a := range ctx.evocation.a {
        if s := a.string(ctx); s != "" {
            if i == 0 && len(ctx.evocation.a) > 1 {
                ext = s
            } else if ext == "" {
                for ext = filepath.Ext(s); ext != ""; {
                    s = strings.TrimSuffix(s, ext)
                    if ctx.all { ext = filepath.Ext(s) } else { break }
                }
                res = append(res, makeStrlit(a.Position(), s))
            } else if ext == filepath.Ext(s) {
                res = append(res, makeStrlit(a.Position(), strings.TrimRight(s, ext)))
            }
        }
    }
    return res
}

type builtin_addprefix struct { builtin_ }
func (ctx *builtin_addprefix) a() (skip bool) {
    for i, a := range ctx.evocation.a {
        if a = a.expand(ctx); i == 0 {
            skip = indeterminate(ctx, a)
        }
        ctx.evocation.a[i] = a
    }
    return
}
func (ctx *builtin_addprefix) x() (_ interface{}) {
    if len(ctx.evocation.a) < 1 {
        erro(ctx, "not enough args, try $(addprefix prefix, ...)").debug(1)
        return
    }

    var res []Value
    var fixs = merge(ctx.evocation.a[0])
    var vals = merge(ctx.evocation.a[1:]...)
    for _, fix := range fixs {
        for _, val := range vals {
            if indeterminate(ctx, val) {
                val = condish(ctx, compose(condless{ctx}, fix, disjunction{val}))
            } else if indeterminate(ctx, fix) {
                val = condish(ctx, compose(condless{ctx}, fix, val))
            } else {
                val = compose(ctx, fix, val)
            }
            res = append(res, val)
        }
    }
    return res
}

type builtin_addsuffix struct { builtin_ }
func (ctx *builtin_addsuffix) a() (skip bool) {
    for i, a := range ctx.evocation.a {
        if a = a.expand(ctx); i == 0 {
            skip = indeterminate(ctx, a)
        }
        ctx.evocation.a[i] = a
    }
    return
}
func (ctx *builtin_addsuffix) x() (_ interface{}) {
    if len(ctx.evocation.a) < 1 {
        erro(ctx, "not enough args, try $(addsuffix suffix, ...)").debug(1)
        return
    }

    var res []Value
    var fixs = merge(ctx.evocation.a[0])
    var vals = merge(ctx.evocation.a[1:]...)
    for _, fix := range fixs {
        for _, val := range vals {
            if indeterminate(ctx, val) {
                val = condish(ctx, compose(condless{ctx}, disjunction{val}, fix))
            } else if indeterminate(ctx, fix) {
                val = condish(ctx, compose(condless{ctx}, val, fix))
            } else {
                val = compose(ctx, val, fix)
            }
            res = append(res, val)
        }
    }
    return res
}

type builtin_printf struct{ builtin_ }
func (ctx *builtin_printf) c() (_ interface{}) { return ctx.x() }
func (ctx *builtin_printf) x() (_ interface{}) {
    if len(ctx.evocation.a) < 1 {
        erro(ctx, "not enough args, try $(printf 'format', ...)").debug(1)
        return
    }

    var vals = merge(ctx.evocation.a[0])
    if len(vals) != 1 {
        erro(ctx, "not enough args, try $(printf 'format', ...)").debug(1)
        return
    }

    var i int
    var a []interface{}
    var f = vals[0].string(ctx)

outer:
    for _, v := range merge(ctx.evocation.a[1:]...) {
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
                    continue outer
                case 'e', 'E', 'f', 'F', 'g', 'G':
                    if t, e := v.float(ctx); e == nil { a = append(a, t) } else {
                        erro(ctx, "%v: %v", v, e).debug(1)
                    }
                    continue outer
                case 'b', 'x', 'X':
                    switch k := v.kind(); {
                    case k&KindInteger != 0:
                        if t, e := v.int(ctx); e == nil { a = append(a, t) } else {
                            erro(ctx, "%v: %v", v, e).debug(1)
                        }
                        continue outer
                    case k&KindFloat != 0:
                        if t, e := v.float(ctx); e == nil { a = append(a, t) } else {
                            erro(ctx, "%v: %v", v, e).debug(1)
                        }
                        continue outer
                    default:
                        if t, e := strconv.Atoi(v.string(ctx)) ; e == nil { a = append(a, t) } else {
                            erro(ctx, "%v: %v", v, e).debug(1)
                        }
                        continue outer
                    }
                case 'v':
                    a = append(a, v/* .string(ctx) */)
                    continue outer
                case 't', 'T':
                    a = append(a, v)
                    continue outer
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
func (ctx *builtin_print) c() (_ interface{}) { return ctx.x() }
func (ctx *builtin_print) x() (_ interface{}) {
    var diag = _diagnostic(ctx)
    if ctx.noErrs && diag.count(diagError) > 0 { return }
    if ctx.noWarn && diag.count(diagWarn) > 0 { return }

    var sb bytes.Buffer
    var x = len(ctx.evocation.a)
    for i, a := range ctx.evocation.a {
        if a == nil {
            continue
        } else if 0 < i && i < x {
            fmt.Fprintf(&sb, " ")
        }
        fmt.Fprintf(&sb, "%s", escapedString(ctx, a))
    }
    prompt(ctx, sb.String())
    return
}

type builtin_printl struct{ builtin_
    noErrs bool `noerrs,noerrors,no-errs,no-errors`
    noWarn bool `nowarn,nowarns,no-warn,no-warns`
}
func (ctx *builtin_printl) c() (_ interface{}) { return ctx.x() }
func (ctx *builtin_printl) x() (_ interface{}) {
    var diag = _diagnostic(ctx)
    if ctx.noErrs && diag.count(diagError) > 0 { return }
    if ctx.noWarn && diag.count(diagWarn) > 0 { return }

    var sb bytes.Buffer
    var x = len(ctx.evocation.a)
    for i, a := range ctx.evocation.a {
        if 0 < i && i < x { fmt.Fprintf(&sb, " ") }
        var s = escapedString(ctx, a)
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
func (ctx *builtin_println) c() (_ interface{}) { return ctx.x() }
func (ctx *builtin_println) x() (_ interface{}) {
    var dia = _diagnostic(ctx)
    if ctx.noErrs && dia.count(diagError) > 0 { return }
    if ctx.noWarn && dia.count(diagWarn) > 0 { return }

    var x = len(ctx.evocation.a)
    var sb bytes.Buffer
    for i, a := range ctx.evocation.a {
        if a == nil {
            continue
        } else if 0 < i && i < x {
            fmt.Fprintf(&sb, " ")
        }
        fmt.Fprintf(&sb, "%s", escapedString(ctx, a))
    }
    fmt.Fprintf(&sb, "\n")
    prompt(ctx, sb.String())
    return
}

type builtin_indent struct { builtin_ }
func (ctx *builtin_indent) x() (res interface{}) {
    var l []Value
    var s string // indent
    if x := len(ctx.evocation.a); x > 0 {
        if v, ok := scalarize(ctx.evocation.a[0]).(*decimal); ok {
            ctx.evocation.a, s = ctx.evocation.a[1:], strings.Repeat(" ", int(v.int64))
        } else {
            erro(ctx, "requires integer argument (first|last)").debug(1)
            return
        }
    }
    for _, a := range ctx.evocation.a {
        var lines []string
        for _, line := range strings.Split(a.string(ctx), "\n") {
            lines = append(lines, s + line)
        }
        l = append(l, makeStrlit(a.Position(), strings.Join(lines, "\n")))
    }
    return l
}

type builtin_findstring struct { builtin_ }
func (ctx *builtin_findstring) x() (res interface{}) {
    // TODO: $(findstring find,text)
    return
}

// $(contains a b c, v1 v2 …)
// $(contains a b c1 -or c2, v1 v2 …)          -- xx
// $(contains a b c1 -or c2 -or c3, v1 v2 …)   -- xx
// $(contains a b -or=(c1 c2 c3), v1 v2 …)     -- xx
type builtin_contains struct { builtin_
    match  bool `mat,match,pat,pattern`
    string bool `str,string`
}
func (ctx *builtin_contains) x() (_ interface{}) {
    if len(ctx.evocation.a) < 2 {
        erro(ctx, "unexpected number of arguments, try $(contains a b c1 c2, v1 v2 …)").debug(1)
        return
    }

    var n int
    var vals = merge(ctx.evocation.a[0])
    var list = merge(ctx.evocation.a[1:]...)
    if len(vals) == 0 || len(list) == 0 {
        erro(ctx, "insufficient number of arguments: %v ⇒ %v %v", ctx.evocation.a, vals, list).debug(6)
        return
    }

outer:
    for _, val := range vals {
        var s string

        if ctx.string { s = val.string(ctx) }

        for _, elem := range list {
            var t bool
            if ctx.match || val.patterned(ctx) {
                t, _, _ = val.match(ctx, elem)
            } else if ctx.string {
                t = s == elem.string(ctx)
            } else {
                t = val.cmp(ctx, elem) == cmpEqual
            }
            if t { n += 1; continue outer }
        }
    }

    if n == len(vals) {
        return makeBoolean(ctx.Position(), true)
    }
    return
}

type builtin_sort struct { builtin_ }
func (ctx builtin_sort) x() (res interface{}) {
    erro(ctx, "TODO: $(sort ...)").debug(5)
    return
}

type builtin_word struct { builtin_ }
func (ctx builtin_word) x() (res interface{}) {
    erro(ctx, "TODO: $(word ...)").debug(5)
    return
}

type builtin_wordlist struct { builtin_ }
func (ctx builtin_wordlist) x() (res interface{}) {
    erro(ctx, "TODO: $(wordlist ...)").debug(5)
    return
}

type builtin_words struct { builtin_ }
func (ctx builtin_words) x() (res interface{}) {
    erro(ctx, "TODO: $(words ...)").debug(5)
    return
}

type builtin_firstword struct { builtin_ }
func (ctx builtin_firstword) x() (res interface{}) {
    erro(ctx, "TODO: $(firstword ...)").debug(5)
    return
}

type builtin_lastword struct { builtin_ }
func (ctx builtin_lastword) x() (res interface{}) {
    erro(ctx, "TODO: $(lastword ...)").debug(5)
    return
}

type builtin_encodebase64 struct { builtin_ }
func (ctx *builtin_encodebase64) x() (res interface{}) {
    if len(ctx.evocation.a) > 0 {
        pos := ctx.Position()
        buf := new(bytes.Buffer)
        enc := base64.NewEncoder(base64.StdEncoding, buf)
        for _, a := range ctx.evocation.a { enc.Write([]byte(a.string(ctx))) }
        enc.Close()
        res = makeStrlit(pos, buf.String())
    }
    return
}

type builtin_decodebase64 struct { builtin_ }
func (ctx *builtin_decodebase64) x() (_ interface{}) {
    if len(ctx.evocation.a) > 0 {
        var res []Value
        for _, a := range ctx.evocation.a {
            var s = a.string(ctx)
            if dat, err := base64.StdEncoding.DecodeString(s); err != nil {
                erro(ctx, "decode '%s' failed: %v", s, err).debug(1)
                return
            } else {
                res = append(res, makeStrlit(a.Position(), string(dat)))
            }
        }
        return ease(ctx, res)
    }
    return
}

type builtin_fullname struct { builtin_ }
func (ctx *builtin_fullname) x() (_ interface{}) {
    var res []Value
    var p = []*project{ctx.project()} // closureprojects(ctx)
    for _, a := range merge(ctx.evocation.a...) {
        if x, y := (as{a}.fullname(ctx, p...)); y { a = x }
        res = append(res, a)
    }
    if 0 < len(res) { return res }
    return
}

type builtin_ext struct { builtin_ }
func (ctx *builtin_ext) x() (_ interface{}) {
    var res []Value
    for _, a := range merge(ctx.evocation.a...) {
        res = append(res, makeStrlit(a.Position(), filepath.Ext(a.string(ctx))))
    }
    if 0 < len(res) { return res }
    return
}

func bases(ctx Context, n int, s string) (d, b string) {
    d, b = filepath.Dir(s), filepath.Base(s)
    for i := n-1; 0 < i; i -= 1 {
        b = filepath.Join(filepath.Base(d), b)
        d = filepath.Dir(d)
    }
    return
}

type builtin_bases struct { builtin_ ; n int `num,size,count` }
func (ctx *builtin_bases) x() (res interface{}) {
    var l []Value
    for _, a := range ctx.evocation.a {
        var s string
        if ctx.fullname {
            s, _ = as{a}.fullnameOrFinal(ctx)
        } else {
            s = a.string(ctx)
        }

        _, s = bases(ctx, ctx.n, s)
        l = append(l, makeStrlit(a.Position(), s))
    }
    return l
}

type builtin_base struct { builtin_bases }
func (ctx *builtin_base) x() (res interface{}) { ctx.n = 1
    return ctx.builtin_bases.x()
}

type builtin_base2 struct { builtin_bases }
func (ctx *builtin_base2) x() (res interface{}) { ctx.n = 2
    return ctx.builtin_bases.x()
}

type builtin_base3 struct { builtin_bases }
func (ctx *builtin_base3) x() (res interface{}) { ctx.n = 3
    return ctx.builtin_bases.x()
}

type builtin_base4 struct { builtin_bases }
func (ctx *builtin_base4) x() (res interface{}) { ctx.n = 4
    return ctx.builtin_bases.x()
}

type builtin_base5 struct { builtin_bases }
func (ctx *builtin_base5) x() (res interface{}) { ctx.n = 5
    return ctx.builtin_bases.x()
}

type builtin_base6 struct { builtin_bases }
func (ctx *builtin_base6) x() (res interface{}) { ctx.n = 6
    return ctx.builtin_bases.x()
}

type builtin_base7 struct { builtin_bases }
func (ctx *builtin_base7) x() (res interface{}) { ctx.n = 7
    return ctx.builtin_bases.x()
}

type builtin_base8 struct { builtin_bases }
func (ctx *builtin_base8) x() (res interface{}) { ctx.n = 8
    return ctx.builtin_bases.x()
}

type builtin_base9 struct { builtin_bases }
func (ctx *builtin_base9) x() (res interface{}) { ctx.n = 9
    return ctx.builtin_bases.x()
}

type builtin_dirs struct { builtin_
    n int `num,size,count`
}
func (ctx *builtin_dirs) x() (res interface{}) {
    var l []Value
    for _, a := range merge(ctx.evocation.a...) {
        var s string
        if ctx.fullname {
            s, _ = as{a}.fullnameOrFinal(ctx)
        } else {
            s = a.string(ctx)
        }
        s = filepath.Dir(s)
        for i := ctx.n-1; 0 < i; i -= 1 { s = filepath.Dir(s) }

        var v Value
        var d = ctx.debug
        if f, y := a.(*File); y {
            if ctx.fullname {
                f = stat(ctx, s, stat_nonexist{true})
            } else {
                f = stat(ctx, s, stat_sub{f.sub}, stat_dir{f.dir}, stat_nonexist{true})
            }
            if d>0 { note(ctx, "%T %v ⇒ %v %v", a, a, f, f.fullname()).debug(d) }
            v = f
        } else if s != "" {
            if d>0 { note(ctx, "%T %v ⇒ %v", a, a, s).debug(d) }
            v = _pathstr(at(ctx, a), s)
        } else {
            continue
        }
        l = append(l, v)
    }
    return l
}

type builtin_dir struct { builtin_dirs }
func (ctx *builtin_dir) x() (res interface{}) { ctx.n = 1
    return ctx.builtin_dirs.x()
}

type builtin_dir2 struct { builtin_dirs }
func (ctx *builtin_dir2) x() (res interface{}) { ctx.n = 2
    return ctx.builtin_dirs.x()
}

type builtin_dir3 struct { builtin_dirs }
func (ctx *builtin_dir3) x() (res interface{}) { ctx.n = 3
    return ctx.builtin_dirs.x()
}

type builtin_dir4 struct { builtin_dirs }
func (ctx *builtin_dir4) x() (res interface{}) { ctx.n = 4
    return ctx.builtin_dirs.x()
}

type builtin_dir5 struct { builtin_dirs }
func (ctx *builtin_dir5) x() (res interface{}) { ctx.n = 5
    return ctx.builtin_dirs.x()
}

type builtin_dir6 struct { builtin_dirs }
func (ctx *builtin_dir6) x() (res interface{}) { ctx.n = 6
    return ctx.builtin_dirs.x()
}

type builtin_dir7 struct { builtin_dirs }
func (ctx *builtin_dir7) x() (res interface{}) { ctx.n = 7
    return ctx.builtin_dirs.x()
}

type builtin_dir8 struct { builtin_dirs }
func (ctx *builtin_dir8) x() (res interface{}) { ctx.n = 8
    return ctx.builtin_dirs.x()
}

type builtin_dir9 struct { builtin_dirs }
func (ctx *builtin_dir9) x() (res interface{}) { ctx.n = 9
    return ctx.builtin_dirs.x()
}

type builtin_undirs struct { builtin_
    n int `num,size,count`
}
func (ctx *builtin_undirs) x() (res interface{}) {
    var l []Value
    for _, a := range ctx.evocation.a {
        var s string
        if ctx.fullname {
            s, _ = as{a}.fullnameOrFinal(ctx)
        } else {
            s = a.string(ctx)
        }
        var v = strings.Split(s, pathSep)
        if i := len(v); i == 0 {
            // v is empty
        } else if ctx.n < i {
            v = v[ctx.n:]
        } else {
            v = v[i-1:] // empty
        }
        l = append(l, _pathstr(at(ctx, a), filepath.Join(v...)))
    }
    return l
}

type builtin_undir struct { builtin_undirs }
func (ctx *builtin_undir) x() (res interface{}) { ctx.n = 1
    return ctx.builtin_undirs.x()
}

type builtin_undir2 struct { builtin_undirs }
func (ctx *builtin_undir2) x() (res interface{}) { ctx.n = 2
    return ctx.builtin_undirs.x()
}

type builtin_undir3 struct { builtin_undirs }
func (ctx *builtin_undir3) x() (res interface{}) { ctx.n = 3
    return ctx.builtin_undirs.x()
}

type builtin_undir4 struct { builtin_undirs }
func (ctx *builtin_undir4) x() (res interface{}) { ctx.n = 4
    return ctx.builtin_undirs.x()
}

type builtin_undir5 struct { builtin_undirs }
func (ctx *builtin_undir5) x() (res interface{}) { ctx.n = 5
    return ctx.builtin_undirs.x()
}

type builtin_undir6 struct { builtin_undirs }
func (ctx *builtin_undir6) x() (res interface{}) { ctx.n = 6
    return ctx.builtin_undirs.x()
}

type builtin_undir7 struct { builtin_undirs }
func (ctx *builtin_undir7) x() (res interface{}) { ctx.n = 7
    return ctx.builtin_undirs.x()
}

type builtin_undir8 struct { builtin_undirs }
func (ctx *builtin_undir8) x() (res interface{}) { ctx.n = 8
    return ctx.builtin_undirs.x()
}

type builtin_undir9 struct { builtin_undirs }
func (ctx *builtin_undir9) x() (res interface{}) { ctx.n = 9
    return ctx.builtin_undirs.x()
}

type builtin_chopdir struct { builtin_ }
func (ctx *builtin_chopdir) x() (res interface{}) {
    var l []Value
    var n = 0
    if x := len(ctx.evocation.a); x > 0 {
        if v, ok := scalarize(ctx.evocation.a[0]).(*decimal); ok {
            ctx.evocation.a, n = ctx.evocation.a[1:], int(v.int64)
        } else if v, ok := scalarize(ctx.evocation.a[x-1]).(*decimal); ok {
            ctx.evocation.a, n = ctx.evocation.a[:x-1], int(v.int64)
        } else {
            erro(ctx, "require (first/last) integer argument (first=%T, last=%T)", ctx.evocation.a[0], ctx.evocation.a[x-1]).debug(1)
            return

        }
    }
    for _, a := range ctx.evocation.a {
        var v = strings.Split(a.string(ctx), pathSep)
        if i := len(v); 0 < i {
            if n < 0 { n = i + n }
            if 0 <= n && n+1 < i {
                v = append(v[0:n], v[n+1:]...)
            } else {
                v = append(v[0:n])
            }
            if len(v) > 0 && v[0] == "" {
                v[0] = pathSep // for absolute paths
            }
        }
        l = append(l, makeStrlit(a.Position(), filepath.Join(v...)))
    }
    return l
}

type builtin_reldir struct { builtin_ }
func (ctx *builtin_reldir) x() (res interface{}) {
    var err error
    var l []Value
    var t string
    for i, a := range ctx.evocation.a {
        if s := a.string(ctx); i == 0 {
            t = s
        } else if s, err = filepath.Rel(t, s); err == nil {
            l = append(l, makeStrlit(a.Position(), s))
        } else {
            erro(ctx, "%v", err)
            return
        }
    }
    return l
}

type builtin_mkdir struct { builtin_
    all bool `all,p,path`
}
func (ctx *builtin_mkdir) c() (res interface{}) {
    for i, nargs := 0, len(ctx.evocation.a); i < nargs; i += 1 {
        var (
            a = ctx.evocation.a[i]
            perm = os.FileMode(0755)
            name string
        )
        switch t := a.(type) {
        case *pair: // mkdir name ⇒ perm name ⇒ perm
            name = t.key.string(ctx)
            perm = filePerm(ctx, t.val, uint32(perm))
        case *group: // mkdir (name perm) (name perm)
            if t.len() == 2 {
                name = t.at(0).string(ctx)
                perm = filePerm(ctx, t.at(1), uint32(perm))
            } else {
                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                break
            }
        case *list: // mkdir name perm, name perm, ...
            if t.len() == 2 {
                name = t.at(0).string(ctx)
                perm = filePerm(ctx, t.at(1), uint32(perm))
            } else {
                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                break
            }
        default: // mkdir name perm, name perm, ...
            name = ctx.evocation.a[i].string(ctx)
            if i+1 < nargs {
                perm = filePerm(ctx, ctx.evocation.a[i+1], uint32(perm))
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
func (ctx *builtin_chdir) c() (res interface{}) {
    if len(ctx.evocation.a) == 1 {
        var str = ctx.evocation.a[0].string(ctx)
        if err := lockCD(str, 0); err != nil {
            erro(ctx, "%v", err).debug(1)
        }
    } else {
        erro(ctx, "wrong number of arguments: %v", len(ctx.evocation.a))
    }
    return
}

type builtin_rename struct { builtin_ }
func (ctx *builtin_rename) c() (res interface{}) {
    for i, nargs := 0, len(ctx.evocation.a); i < nargs; i += 1 {
        var (
            a = ctx.evocation.a[i]
            oldname, newname string
        )
        switch t := a.(type) {
        case *pair: // rename oldname=newname
            oldname = t.key.string(ctx)
            newname = t.val.string(ctx)
        case *group: // rename (oldname newname) (old new)
            if t.len() == 2 {
                oldname = t.at(0).string(ctx)
                newname = t.at(1).string(ctx)
            } else {
                erro(at(ctx,t), "wrong size of group `%v'", t).debug(1)
                break
            }
        case *list: // rename oldname newname, old new, ...
            if t.len() == 2 {
                oldname = t.at(0).string(ctx)
                newname = t.at(1).string(ctx)
            } else {
                erro(at(ctx,t), "wrong size of list `%v'", t).debug(1)
                break
            }
        default: // rename newname oldname  newname oldname ...
            if i+1 < nargs {
                oldname = ctx.evocation.a[i+0].string(ctx)
                newname = ctx.evocation.a[i+1].string(ctx)
                i += 1
            } else {
                erro(at(ctx,t), "Wrong arguments `%v'", ctx.evocation.a).debug(1)
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
    ignoreMissing bool `ig,ignore,ignore-missing,ignore-not-found`
    warnNotFile bool `warn-not-file`
    all bool `all,recursive`
}
func (ctx *builtin_remove) c() (res interface{}) {
    var opts = ctx
    var remove func(Context, Value)
    var removeFile = func(ctx Context, f *File) {
        var err error
        var s = f.fullname()
        if opts.skip != "" {
            if strings.HasPrefix(s, opts.skip) { return } else
            if strings.HasPrefix(f.ident(ctx), opts.skip) { return }
        }
        if opts.all { err = os.RemoveAll(s) } else { err = os.Remove(s) }
        if err != nil {
            erro(ctx, "remove: %v", err)
            erro(ctx, "remove: %v → %s", f, s).debug(6)
            return
        }
        if d := opts.debug; d>0 { warn(ctx, "remove %s (%s)", f, s).debug(d) }
        if opts.verbose { prompt(ctx, "removed %s\n", f) }
    }
    var removePath = func(ctx Context, p *path) {
        var err error
        var s = p.string(ctx)
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
        // var val = (&builtin_wildcard{builtin_:builtin_{evocation:?}})._do(pat)
        // erro(ctx, "TODO: remove: %v → %v", pat, val).debug(1)
        erro(ctx, "TODO: remove: %v", us(pat)).debug(1)
    }

    remove = func(ctx Context, v Value) {
        if _, y := v.(*none); y {
            return
        } else if isTrivial(v) {
            notestack(ctx, 5, "triviality: %v (%T)", v, v).debug(6)
        } else if l, y := v.(*list); y {
            for _, v := range l.elems { remove(ctx, v) }
        } else if d, y := v.(*delegate); y {
            notestack(ctx, 5, "delegate: %v (%T, %v, %v)", d.x, d.x, d.o, d.a).debug(6)
        } else if v.patterned(ctx) {
            removePat(ctx, v)
        } else if f, y := v.(*File); y {
            removeFile(ctx, f)
        } else if f = file(ctx, v.string(ctx)); f != nil {
            removeFile(ctx, f)
        } else if p, y := v.(*path); y {
            removePath(ctx, p)
        } else if !opts.ignoreMissing {
            errostack(ctx, 5, "not file: %v (%T)", v, v).debug(6)
        }
    }
    for _, a := range ctx.evocation.a {
        ctx := at(ctx.Context, a.Position())
        remove(ctx, a.expand(ctx))
    }

    if opts.debug > 0 { warn(ctx, "%v", ctx.evocation.a).debug(1) }
    if opts.debug > 0 && flush(ctx) > 0 {
        errostack(ctx, 3, "remove errors").debug(1)
    }
    return
}

type builtin_truncate struct { builtin_ }
func (ctx *builtin_truncate) c() (res interface{}) {
    for i, nargs := 0, len(ctx.evocation.a); i < nargs; i += 1 {
        var (
            a = ctx.evocation.a[i]
            name string
            size int64
            e error
        )
        switch t := a.(type) {
        case *pair: // truncate name ⇒ size old ⇒ new
            name = t.key.string(ctx)
            if size, e = t.val.int(ctx); e != nil {
                erro(ctx, "%v: %v", t.val, e).debug(1)
            }
        case *group: // truncate (name size) (old new)
            if t.len() == 2 {
                name = t.at(0).string(ctx)
                if size, e = t.at(1).int(ctx); e != nil {
                    erro(ctx, "%v: %v", t.at(1), e).debug(1)
                }
            } else {
                erro(ctx, "Wrong size of group `%v'", t).debug(1)
                break
            }
        case *list: // truncate name size, old new, ...
            if t.len() == 2 {
                name = t.at(0).string(ctx)
                if size, e = t.at(1).int(ctx); e != nil {
                    erro(ctx, "%v: %v", t.at(1), e).debug(1)
                }
            } else {
                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                break
            }
        default: // truncate name size  name size ...
            if i+1 < nargs {
                name = ctx.evocation.a[i+0].string(ctx)
                if size, e = ctx.evocation.a[i+1].int(ctx); e != nil {
                    erro(ctx, "%v: %v", ctx.evocation.a[i+1], e).debug(1)
                }
                i += 1
            } else {
                erro(ctx, "Wrong arguments `%v'", ctx.evocation.a).debug(1)
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
func (ctx *builtin_link) c() (res interface{}) {
    for i, nargs := 0, len(ctx.evocation.a); i < nargs; i += 1 {
        var (
            oldname, newname string
            a = ctx.evocation.a[i]
        )
        switch t := a.(type) {
        case *pair: // link oldname ⇒ newname old ⇒ new
            oldname = t.key.string(ctx)
            newname = t.val.string(ctx)
        case *group: // link (oldname newname) (old new)
            if t.len() == 2 {
                oldname = t.at(0).string(ctx)
                newname = t.at(1).string(ctx)
            } else {
                erro(ctx, "Wrong size of group `%v'", t).debug(1)
                break
            }
        case *list: // link oldname newname, old new, ...
            if t.len() == 2 {
                oldname = t.at(0).string(ctx)
                newname = t.at(1).string(ctx)
            } else {
                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                break
            }
        default: // link oldname newname  oldname newname ...
            if i+1 < nargs {
                oldname = ctx.evocation.a[i+0].string(ctx)
                newname = ctx.evocation.a[i+1].string(ctx)
                i += 1
            } else {
                erro(ctx, "Wrong arguments `%v'", ctx.evocation.a).debug(1)
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
    path     bool `path`
    force    bool `force,overwrite`
    update   bool `update`
    relative bool `rel,relative`
}
func (ctx *builtin_symlink) c() (res interface{}) {
outer:
    for i, na := 0, len(ctx.evocation.a); i < na; i += 1 {
        var (
            opts = *ctx // make a copy
            srcNameVal, dstNameVal Value
            srcName   , dstName    string
            srcDir    , dstDir     string
            aa []Value
        )
        switch t := ctx.evocation.a[i].(type) {
        case *pair: // symlink srcName=dstName srcName=>dstName...
            srcNameVal, dstNameVal = t.key, t.val
        case *group: // symlink (-u srcName dstName) (-v srcName dstName)...
            if aa = parseOpts(ctx, &opts, t.elems...); len(aa) != 2 {
                erro(at(ctx,t), "expects two values for group").debug(1)
                return
            } else {
                srcNameVal, dstNameVal = aa[0], aa[1]
            }
        case *list: // XXX: symlink old new, old new, ...
            if aa = parseOpts(ctx, &opts, t.elems...); len(aa) != 2 {
                erro(at(ctx,t), "expects two values for list").debug(1)
                return
            } else {
                srcNameVal, dstNameVal = aa[0], aa[1]
            }
        default:// Multiple pairs of names:
            // symlink  new old, new old ...
            // symlink  new old  new old ...
            if i+1 < na {
                srcNameVal = ctx.evocation.a[i+0]
                dstNameVal = ctx.evocation.a[i+1]
                i += 1
            } else {
                var a = autoVal(ctx,"@")
                var l = autoVal(ctx,"<")
                var r = autoVal(ctx,">")
                prompt(ctx, "symlink: args=%v → %v\n", ctx.evocation.a, t)
                prompt(ctx, "symlink: %v, %v, %v\n", a, l, r)
                errostack(at(ctx,t), 5, "expects pair of names (%T %v)", t, t).debug(6)
                return
            }
        }

        if srcDir, srcName = splitFileName(ctx, srcNameVal); srcName == "" {
            prompt(ctx, "symlink: args=%v\n", ctx.evocation.a)
            prompt(ctx, "symlink: src=%v\n", srcNameVal)
            errostack(at(ctx,srcNameVal), 5, "empty src filename (%T)", srcNameVal).debug(6)
            return
        }
        if dstDir, dstName = splitFileName(ctx, dstNameVal); dstName == "" {
            prompt(ctx, "symlink: args=%v\n", ctx.evocation.a)
            prompt(ctx, "symlink: dest=%v\n", dstNameVal)
            errostack(at(ctx,dstNameVal), 6, "empty dest filename (%T)", dstNameVal).debug(12)
            return
        }

        var src = srcName
        var dst = dstName
        if !filepath.IsAbs(src) { src = filepath.Join(srcDir, srcName) }
        if !filepath.IsAbs(dst) { dst = filepath.Join(dstDir, dstName) }
        if _, err := os.Stat(src); err != nil {
            prompt(ctx, "symlink: %v: %v\n", srcName, err)
            errostack(at(ctx,srcNameVal), 6, "%v does not exist", srcName).debug(8)
            return
        }

        if !opts.relative {/* no rel required */} else
        if s, e := filepath.Rel(filepath.Dir(dst), src); e != nil {
            prompt(ctx, "symlink: %s: rel(%s, %s)\n", dstName, dst, src)
            errostack(at(ctx,dstNameVal), 8, "%v", e).debug(10)
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
        if dstDir == "" || dstDir == "." || dstDir == pathSep {
            // no need to mkdir: . or /
        } else if err := os.MkdirAll(dstDir, os.FileMode(0755)); err != nil {
            erro(at(ctx,dstNameVal), "%v", err).debug(1)
            return
        }

        var rm bool
        if rm = opts.force; rm {
            // overwrite...
        } else if s, e := os.Readlink(dst); e != nil {
            if false {
                prompt(ctx, "%v: readlink failed (%T)\n", dstName, e)
                errostack(at(ctx,dstNameVal), 6, "%v", e).debug(8)
            }
        } else if rm = s != src; !rm {
            continue outer
        }

        if rm { if e := os.Remove(dst); e != nil {
            prompt(ctx, "%v: remove old symlink failed (%T)\n", dstName, e)
            errostack(at(ctx,dstNameVal), 6, "%v", e).debug(8)
            return
        }}
        if err := os.Symlink(src, dst); err != nil {
            if opts.verbose { prompt(ctx, "… %s\n", err) }
            break
        } else if opts.verbose {
            var d = trimPromptString(dstName)
            var s = filepath.Base(srcName)
            prompt(ctx, "%s → %s …… ok\n", d, s)
        }
    }
    return
}

type builtin_stat struct { builtin_
    symbol bool `sym,symbol,symlink,link`
    file   bool `file`
    dir    bool `dir`
}
func (ctx *builtin_stat) x() (res interface{}) {
    if len(ctx.evocation.a) == 0 { return }

    var proj = ctx.project()
    if proj == nil {
        erro(ctx, "unknown project").debug(1)
        return
    }

    var vals []Value

    var check = func(f *File) {
        if f != nil && f.info != nil {
            var mode = f.info.Mode()
            if  (ctx.dir    && mode&os.ModeDir     != 0) ||
                (ctx.file   && mode&os.ModeType    != 0) ||
                (ctx.symbol && mode&os.ModeSymlink != 0) ||
                (!ctx.dir && !ctx.file && !ctx.symbol) { vals = append(vals, f) }
        }
    }

    var checkstat = func(a Value) {
        var file *File
        var s string
        if s = a.string(ctx); filepath.IsAbs(s) {
            file = stat(ctx, s)
        } else {
            file = stat(ctx, s, proj) // aka stat_dir{proj.absPath}
        }
        if file == nil { file = proj.file(ctx, s) }
        if file != nil { check(file) }
    }

    for _, a := range merge(ctx.evocation.a...) {
        switch t := a.(type) {
        case *File: check(t)
        case *path: checkstat(a)
        default:    checkstat(a)
        }
    }

    return vals
}

type builtin_file struct { builtin_
    exists bool `exist,exists,must,must-exist,required`
    report bool `report,reportmissing,report-missing`
    ignore bool `ignore,ignore-missing,missing,nonexist,non-exist`
    mapped bool `map,maps,mapped`
}
func (ctx *builtin_file) x() (res interface{}) {
    return ctx.z([]*project{ctx.project()}, ctx.evocation.a...)
}
func (ctx *builtin_file) z(projs []*project, args ...Value) (res []Value) {
    defer trace(ctx)

    var en int
    var cc = ctx.Context
    var f = func(a Value) {
        ctx.Context = at(cc, a)

        var fs []*File
        var am []matched_filemap
        if f, y := toFile(a); y {
            if !ctx.exists || f.exists() /* || f.stat(ctx) != nil */ {
                res = append(res, f)
            } else if ctx.report {
                info(ctx, "no such file {%v %v %v}", f.dir, f.sub, f.name).debug(1)
            }
            return
        }

        if am = files(ctx, a, projs...); am == nil && false {
            var s = a.string(ctx)
            if s == "" {
                erro(ctx, "%v is empty for searching file", us(a)).debug(3)
                return
            }
            if am = files(ctx, s, projs...); am == nil {
                if f := file(ctx, s, projs...); f != nil {
                    res = append(res, f)
                    return
                } else {
                    if ctx.mapped {
                        var t = unmap_files(ctx, a)
                        erro(ctx, "not a file ; %v → %v", tv(a), t).debug(1)
                    }
                    return
                }
            }
        }

        for _, p := range projs {
            fs = append(fs, p.selectFiles(ctx, am)...)
        }

        for _, f := range fs {
            if !ctx.exists || f.exists() {
                res = append(res, f)
            } else if ctx.ignore {
                if ctx.verbose { info(ctx, "%v → %v", tv(a), f).debug(1) }
            } else if ctx.exists {
                en += 1
            }
        }

        if en > 0 { for i, m := range am {
            info(at(ctx,m.pattern), "found %d. %s → %s(%s) → %v", i, m.name, typeof(m.pattern), m.pattern, m.paths)
        }}
    }

    for _, a := range merge(args...) {
        if f(a); en > 0 {
            erro(ctx, `%v: %v is not a file (%v)`, projs, us(a), res)
            errostack(ctx, 5).debug(16)
            return
        }
    }
    return
}

type builtin_glob struct { builtin_
    symbol bool `sym,symlink,symbol`
    dir bool `dir,directory`
    file bool `file`
}
func (ctx *builtin_glob) x() (_ interface{}) {
    var cwd string // TODO: get current work directory
    var proj *project
    if proj = ctx.project(); proj == nil {
        erro(ctx, "unknown current cntext").debug(1)
        return
    }

    var res []Value
    for _, a := range ctx.evocation.a {
        var ( str string; names []string )
        if str = a.string(ctx); !filepath.IsAbs(str) {
            str = filepath.Join(cwd, str)
        }

        var err error
        if names, err = filepath.Glob(str); err != nil {
            erro(ctx, "glob '%v' failed: %v", str, err).debug(1)
            return
        }
        for _, name := range names {
            // TODO: ctx.dir, ctx.file, ctx.symbol
            res = append(res, _pathstr(ctx, name))
        }
    }
    return res
}

func readDirNames(ctx Context, sd string, errorMissing bool) (names []string) {
    if f, err := os.Stat(sd); err != nil {
        if errorMissing { erro(ctx, "%v", err).debug(1) }
        return
    } else if !f.IsDir() {
        erro(ctx, "not dir: %v", sd).debug(1)
        return
    } else if dir, err := os.Open(sd); err != nil {
        erro(ctx, "not dir: %v", sd).debug(1)
        return
    } else if names, err = dir.Readdirnames(-1); err != nil { // NOTE: see also filepath.Glob(...)
        if errorMissing { erro(ctx, "readdir: %v", err).debug(1) }
        return
    } else {
        dir.Close()
        return
    }
}

type builtin_wildcard struct { builtin_
    includeMissing bool `includemissing,include-missing,missing`
    ignoreMissing bool `ignoremissing,ignore-missing`
    errorMissing bool `err,errormissing,error-missing,no-missing`
    names bool `name,names,nameonly`
    strs bool `str,strs,string,strings`
    exclude []Value `excl,exclude,except,no,not`
    filetype string `type,filetype,file-type` // dir, file, etc.
    dir string `dir,directory`
}
func (ctx *builtin_wildcard) _directory(topDir string, pats ...Value) (files []*File) {
    if checkpoints {
        if s := _workdir(ctx); s == "" {
            erro(ctx, "empty workdir").debug(16)
        } else if !strings.HasPrefix(topDir, s) {
            erro(ctx, "%s", s)
            erro(ctx, "%s", topDir)
            erro(ctx, "%v", us(ctx)).debug(16)
        }
    }

    type subr struct {
        d, n, dn string // dir, name, dir+name
        pat chan Value
        isDir bool
        ss []*subr
        sync.WaitGroup
        sync.Mutex
    }

    var work func(sub *subr)
    var top = subr{ pat: make(chan Value, 4) }
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
                ss = &subr{ d: sub.dn, pat: make(chan Value, 1) }
                sub.ss = append(sub.ss, ss)
                top.Add(1) ; go work(ss)
            }
            sub.Unlock()
        }
        ss.pat <- pat
    }
    var collect = func(name string) {
        var ne = ctx.includeMissing && !ctx.ignoreMissing
        var f = stat(ctx, name, stat_dir{topDir}, stat_nonexist{ne})
        if true { assert(f != nil, "stat %s %s", name, topDir) }

        top.Lock()
        switch d := f.info.IsDir(); strings.ToLower(ctx.filetype) {
        case "f", "file": if!d { files = append(files, f) }
        case "d", "dir" : if d { files = append(files, f) }
        case "":                 files = append(files, f)
        default: erro(ctx, "unknown -filetype: %s (%v)", ctx.filetype, f).debug(1)
        }
        top.Unlock()
        top.Done()
    }
    var subcard = func(sub *subr, pat Value) {
        defer sub.Done()

        if t, y := pat.(compositePattern); y { pat = t.Value }
        if t, y := pat.(*list); y {
            warn(ctx, "pattern is a list: %T %v %v", pat, pat, t.elems).debug(1)
            if len(t.elems) == 1 { pat = t.elems[0] }
        }

        var ctx = at(ctx, pat.Position())
        if p, y := pat.(*path); !y {
            // fallthrough
        } else if nElems := len(p.elems); nElems == 0 {
            errostack(ctx, 3, "empty path: %v", pat).debug(3)
            return
        } else if y, _, _ = p.elems[0].match(ctx, sub.n); y && nElems == 1 {
            errostack(ctx, 3, "%v %v: invalid path: %v, %v, %v", topDir, sub.dn, pat, sub.n, nElems).debug(1)
            return
        } else if y && sub.isDir && nElems > 1 {
            val := p.elems[1]
            if nElems > 2 {
                var v = &path{}
                v.elems = p.elems[1:]
                val = v
            }
            subed(sub, val)
            return
        } else if sub.d == "" {
            if y { warn(ctx, "%T %v %v", pat, pat, sub).debug(1) }
            return
        }

        if gp, y := pat.(*globpat); !y {
            // fallthrough
        } else if len(gp.elems) == 0 {
            errostack(ctx, 3, "empty glob: %v (%s)", pat, sub.dn).debug(3)
            return
        } else if m, y := gp.elems[0].(*globmeta); !y {
            // fallthrough
        } else if m.token == DAST { // aka **
            y, _, _ = gp.match(ctx, sub.dn)
            if sub.isDir { subed(sub, pat) }
            if y { top.Add(1) ; go collect(sub.dn) ; return }
            return
        }

        y, _, _ := pat.match(ctx, sub.n)
        if y { top.Add(1) ; go collect(sub.dn) ; return }
        return
    }
    var subwork = func(subdir, name string, pats []Value) {
        defer top.Done()

        var sub = &subr{ d:subdir, n:name, dn:filepath.Join(subdir,name) }

        for _, x := range ctx.exclude {
            if y, _, _ := x.match(ctx, sub.dn); y { return }
        }

        if fi, err := os.Stat(filepath.Join(topDir, sub.dn)); err == nil {
            sub.isDir = fi.IsDir()
        } else {
            erro(ctx, "%p: %v %v → %v", sub, sub.d, sub.n, sub.dn)
            errostack(ctx, 3, "%v", err).debug(16)
            return
        }

        for _, pat := range pats { sub.Add(1) ; go subcard(sub, pat) }
        top.Add(1) ; go func() { sub.Wait()
            for _, s := range sub.ss { if s.pat != nil { close(s.pat) }}
            top.Done()
        } ()
    }

    work = func(sub *subr) {
        names := readDirNames(ctx, filepath.Join(topDir, sub.d), ctx.errorMissing)

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
func (ctx *builtin_wildcard) _project(p *project, pats ...Value) (files []*File) {
    if false { defer func(t0 time.Time) {
        if d := time.Now().Sub(t0); d > 1*time.Second {
            var pos = ctx.Position()
            prompt(ctx, "%v: slow: %d patterns, %v\n", pos, len(pats), pats)
            prompt(ctx, "%v: slow: %d files\n", pos, len(files))
            prompt(ctx, "%v: slow: %v\n", pos, d).debug(4)
        }
    }(time.Now())}

    var m sync.Mutex
    var g sync.WaitGroup
    var collect = func(t ...*File) {
        m.Lock()
        files = append(files, t...)
        m.Unlock()
        g.Done()
    }

    var st = func(dir string, val Value) {
        var ne = ctx.includeMissing && !ctx.ignoreMissing
        if f := stat(ctx, val.string(ctx), stat_dir{dir}, stat_nonexist{ne}); f != nil {
            g.Add(1) ; go collect(f)
        } else if false {
            erro(ctx, "nil: %v (%s)", us(val), dir).debug(1)
        }
    }

    var dofilemap = func(lVal, rVal Value, lPat, rPat bool, fm *filemap) {
        defer g.Done()
        for _, loc := range fm.paths {
            if dir := loc.string(ctx); lPat && rPat {
                var pat Value
                if lVal.cmp(ctx, rVal) == cmpEqual {
                    pat = lVal
                } else {
                    pat = compositePattern{lVal, []Value{rVal}}
                }
                g.Add(1) ; go collect(ctx._directory(dir, pat)...)
            } else if lPat && !rPat {
                st(dir, rVal)
            } else if !lPat && rPat {
                st(dir, lVal)
            } else {
                note(ctx, "TODO: wildcard: 3. %v %v %s", lVal, rVal, dir)
            }
        }
    }

    var f1 = func(inVal, mapVal Value, inPat, mapPat bool, fm *filemap) {
        defer g.Done()
        if y, _, _ := inVal.match(ctx, mapVal); y { // e.g. inVal=**.am <-> mapVal=foo/bar/*.am
            g.Add(1) ; go dofilemap(inVal, mapVal, inPat, mapPat, fm)
        } else if y, _, _ = mapVal.match(ctx, inVal); y { // e.g. mapVal=**.am <-> inVal=foo/bar/*.am
            if g.Add(1) ; true {
                go dofilemap(inVal, mapVal, inPat, mapPat, fm)
            } else {
                go dofilemap(mapVal, inVal, mapPat, inPat, fm)
            }
        } else {
            warn(ctx, "TODO: wildcard: %v %v", mapVal, inVal).debug(1)
        }
    }

    var f2 = func(inVal Value, inPat bool, c *_DEPRECATED_vcache) {
        defer g.Done()
        var fm, y = c._val.(filemap)
        if y && fm._filemap != nil {
            for _, mapVal := range fm.primePatterns(ctx) {
                g.Add(1) ; go f1(inVal, mapVal, inPat, mapVal.patterned(ctx), &fm)
            }
        } else {
            erro(ctx, "not filemap: %v", us(c._val)).debug(1)
        }
    }

    var f3 = func(inVal Value) {
        defer g.Done()
        var inPat = inVal.patterned(ctx)
        for _, c := range _DEPR_collect(ctx, inVal, &p.filemap, cacheMatchPatts) {
            g.Add(1) ; go f2(inVal, inPat, c)
        }
    }

    for _, pat := range pats { g.Add(1) ; go f3(pat) }
    g.Wait()
    return
}
func (ctx *builtin_wildcard) _do(pats ...Value) []*File {
    if ctx.dir == "" {
        return ctx._project(ctx.project(), pats...)
    } else {
        return ctx._directory(ctx.dir, pats...)
    }
}
func (ctx *builtin_wildcard) x() interface{} {
    var vals []Value
    if len(ctx.exclude) > 0 { ctx.exclude = merge(ctx.exclude...) }
    for _, file := range ctx._do(merge(ctx.evocation.a...)...) {
        if file == nil {
            errostack(ctx, 3, "nil file: %v", ctx.evocation.a).debug(3)
        } else if !(ctx.names || ctx.strs) {
            vals = append(vals, file)
        } else if ctx.strs {
            vals = append(vals, makeStrlit(file.position, file.ident(ctx)))
        } else if strings.Contains(file.ident(ctx), pathSep) {
            vals = append(vals, _pathstr(ctx, file.ident(ctx)))
        } else {
            vals = append(vals, makeBareword(file.position, file.ident(ctx)))
        }
    }
    return vals
}

type builtin_readdir struct { builtin_ }
func (ctx *builtin_readdir) x() (res interface{}) {
    var l []Value
    for _, a := range ctx.evocation.a {
        if fis, err := ioutil.ReadDir(a.string(ctx)); err == nil {
            v := new(list)
            for _, fi := range fis {
                v.append(makeStrlit(a.Position(), fi.Name()))
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
func (ctx *builtin_readfile) x() (res interface{}) {
    var l []Value
    var closured = closureprojects(ctx)
    for _, v := range ctx.evocation.a {
        if o, y := (as{v}.fullname(ctx, closured...)); !y {
            errostack(at(ctx,v), 5, "%v is not a file", v).debug(1)
            break
        } else if s, e := ioutil.ReadFile(o.string(ctx)); e != nil {
            errostack(at(ctx,v), 5, "read file failed: %v", e).debug(1)
            break
        } else {
            if ctx.trim      { s = bytes.TrimFunc     (s, unicode.IsSpace) } else
            if ctx.trimLeft  { s = bytes.TrimLeftFunc (s, unicode.IsSpace) } else
            if ctx.trimRight { s = bytes.TrimRightFunc(s, unicode.IsSpace) }
            l = append(l, makeStrlit(v.Position(), string(s)))
        }
    }
    return l
}

type builtin_writefile struct { builtin_
    path bool `path`
}
func (ctx *builtin_writefile) x() (res interface{}) {
    // $(write-file filename,content)
    // $(write-file -p filename,content)
outer:
    for i := 0; i < len(ctx.evocation.a); i += 1 {
        var (
            a = ctx.evocation.a[i]
            name, data string
            perm = os.FileMode(0600)
        )
        switch t := a.(type) {
        case *pair: // write-file name=text name=text
            name = t.key.string(ctx)
            data = t.val.string(ctx)
        case *group: // write-file (name text) (name text 0660)
            if n := t.len(); n < 4 && n > 0 {
                name = t.at(0).string(ctx)
                if n > 1 { data = t.at(1).string(ctx) }
                if n > 2 { perm = filePerm(ctx, t.at(2),0600) }
            } else {
                erro(ctx, "Wrong size of group `%v'", t).debug(1)
                break
            }
        case *list: // write-file name text, name text 0660, ...
            if n := t.len(); n < 4 && n > 0 {
                name = t.at(0).string(ctx)
                if n > 1 { data = t.at(1).string(ctx) }
                if n > 2 { perm = filePerm(ctx, t.at(2),0600) }
            } else {
                erro(ctx, "Wrong size of list `%v'", t).debug(1)
                break
            }
        default: // write-file name text 0660  name text 0660 ...
            name = ctx.evocation.a[i].string(ctx)
            if i+1 < len(ctx.evocation.a) {
                data = ctx.evocation.a[i+1].string(ctx)
                i += 1
            }
            if i+1 < len(ctx.evocation.a) {
                perm = filePerm(ctx, ctx.evocation.a[i+1],0600)
                i += 1
            }
        }
        if name == "" {
            continue outer
        } else if dir := filepath.Dir(name); ctx.path && dir != "." && dir != pathSep {
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
    var a, filename, c = as{file}.fullnameFile(ctx)
    if filename == "" {
        errostack(at(ctx,file), 3, "touch: empty file name: %v (%v, %v, %v)", file, typeof(file), a, c).debug(24)
        return
    } else if d := filepath.Dir(filename); optPath && d != "." && d != pathSep {
        if err = os.MkdirAll(d, os.FileMode(optMode|0733)); err != nil {
            erro(at(ctx,file), "touch: %v", err).debug(1)
            return
        }
    }

    var (
        mode = os.FileMode(optMode)
        ta, tm time.Time
        m os.FileMode
    )
    if len(ts) > 0 { ta = ts[0] } else { ta = time.Now() }
    if len(ts) > 1 { tm = ts[1] } else { tm = time.Now() }
    if fi, k := toFile(file); k && fi.info != nil {
        m = fi.info.Mode()
    } else if fi, e := os.Stat(filename); e == nil && fi != nil {
        m = fi.Mode()
    } else {
        var f *os.File
        if m = mode; m == 0 { m = os.FileMode(0600); mode = m }
        if f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, m&os.ModePerm); err != nil {
            erro(at(ctx,file), "touch: %v", err).debug(1)
        } else if err = f.Close(); err != nil {
            erro(at(ctx,file), "touch: %v", err).debug(1)
        }
    }
    if err == nil {
        if err = os.Chtimes(filename, ta, tm); err != nil {
            erro(at(ctx,file), "touch: %v", err).debug(1)
        }
    }
    if err == nil && mode != 0 && m != 0 && mode != m {
        if err = os.Chmod(filename, mode); err != nil {
            erro(at(ctx,file), "touch: %v", err).debug(1)
        }
    }
    return
}

type builtin_touchfile struct { builtin_
    mode os.FileMode `mode`
    path bool `path`
}
func (ctx *builtin_touchfile) x() (res interface{}) {
    // $(touch-file filename)
    // $(touch-file -p filename)
    for i := 0; i < len(ctx.evocation.a); i += 1 {
        if err := touch(ctx, ctx.evocation.a[i], uint32(ctx.mode), ctx.path); err != nil {
            erro(ctx, "%v", err).debug(1)
            break
        }
    }
    return
}

// $(grep 'status=1',$@)
// $(grep 'status=([0-9]+)',$1,$@)
type builtin_grep struct { builtin_ }
func (ctx *builtin_grep) x() (_ interface{}) {
    var (
        args = ctx.evocation.a
        nargs = len(args)
        res []Value
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
        if s := a.string(ctx); s == "" {
            erro(at(ctx,a), "empty regexp").debug(1)
            return
        } else if r, e := regexp.Compile(s); e != nil {
            erro(at(ctx,a), "%v", e).debug(1)
            return
        } else {
            rxs = append(rxs, r)
        }
    }

    var pos = ctx.Position()
    var cc = automatic{ Context:ctx, defs:make(autodefs),
        suppress:func(s string) bool { return _isDigits(s) }}
    var greped = func(line int, match []string) (done bool) {
        var vals []Value
        for i, s := range match {
            if d, v := cc.set(ctx, fmt.Sprintf("%d",i), makeStrlit(pos, s)); d == nil {
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
        res = append(res, result.expand(&cc))
        return
    }

    for _, a := range merge(args...) {
        var file *os.File
        var filename string

        if f, y := a.(*File); y {
            filename = f.fullname()
        } else {
            filename = a.string(ctx)
        }

        if c := at(ctx, a); filename == "" {
            var pc = cast[*programContext](ctx)
            erro(c, "empty filename: %v", us(a))
            erro(c, "%v %v", rvs, args)
            errostack(c, 5, "%p %v", pc, pc.search(ctx, "^")).debug(64)
            return
        } else if file, err = os.Open(filename); err != nil {
            erro(c, "%v", err)
            errostack(c, 5, "%v (%T)", a.string(ctx), a).debug(128)
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
    return ease(ctx, res)
}

var (
    rsAutoconf  = `AC_(CHECK_(FILES?|FUNCS?|HEADERS?|PROG|SIZEOF|TOOL)|DEFINE)\(([^\)]*?)\)`
    rsConfigRef = `[$%]\{([^\s\}]+)\}|@([^\s\@]+)@`
    rsConfigure = `^[\t ]*#[\t ]*(define|undef|smartdefine|smartdefine01|cmakedefine|cmakedefine01)[\t ]+([A-Za-z0-9_]+)(?:[\t ]+([^\n]*))?$`
    rxAutoconf  = regexp.MustCompile(rsAutoconf)
    rxConfigure = regexp.MustCompile(fmt.Sprintf(`(?m:%s)`, rsConfigure)) // m: multilines
    rxConfigRef = regexp.MustCompile(rsConfigRef)
)

func (project *project) strExpandConfig(ctx Context, s string) (result string, err error) {
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
        } else if val = def.invoke(ctx, nil, nil); isNull(val) {
            if cf := project.configuration(ctx); cf == nil {
                erro(at(ctx,def), "%v: configuration file not defined", name, cf).debug(1)
                return
            } else if !cf.exists() {
                prompt(ctx, "%s: file not exists (for %v)\n", cf.fullname(), name)
                erro(at(ctx,def), "%v: configuration file not exists, try -conf first", name).debug(1)
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
            fmt.Fprintf(res, "%s", parseGroupValue(ctx, t).string(ctx))
        default:
            fmt.Fprintf(res, "%s", val.string(ctx))
        }
    }
    if index < len(s) { fmt.Fprint(res, s[index:]) }
    result = res.String()
    return
}

// https://www.gnu.org/software/autoconf/manual/autoconf-2.67/autoconf.html
func autoconf(ctx Context, out *bytes.Buffer, project *project, str string) (err error) {
    var num int
    for _, m := range rxAutoconf.FindAllStringSubmatch(str, -1) {
        info(ctx, "TODO: %v", m)
        num += 1
    }
    warn(ctx, "TODO: %d", num).debug(1)
    return
}

func configureString(ctx Context, out *bytes.Buffer, project *project, str string) (err error) {
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
            if val := def.invoke(ctx, nil, nil); val == nil {
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
            } else if va = expand(ctx, def.value); len(va) == 1 {
                switch v := va[0].(type) {
                case *answer, *boolean:
                    if b := v.true(ctx); b {
                        s = fmt.Sprintf("#define %s 1 /* %T %v */", name, v, v)
                    } else {
                        s = fmt.Sprintf("#undef %s /* %T %v */", name, v, v)
                    }
                case *strlit:
                    s = strings.Replace(v.s, "\"", "\\\"", -1)
                    s = fmt.Sprintf("#define %s \"%s\"", name, v.s)
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
func (ctx *builtin_untraversed) x() (res interface{}) {
    return untraversed{ease(ctx, ctx.evocation.a)}
}

type builtin_return struct { builtin_ }
func (ctx *builtin_return) x() (res interface{}) {
    return &returner{valbase{ctx.Position()}, ctx.evocation.a }
}
