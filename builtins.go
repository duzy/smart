//
//  Copyright (C) 2012-2024, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "encoding/base64"
    "path/filepath"
    "io/ioutil"
    "net/http"
    "os/exec"
    "context"
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

type builtincalls struct{}
func _builtincalls(ctx Context) (_ string) {
    if s, y := do(ctx, builtincalls{}).(string); y {
        return strings.Replace(s, "(%s)", "", -1)
    }
    return
}

type builtinbase struct{ *evocation ; general_opts }
func (c *builtinbase) inner() Context { return c.evocation }
func (c *builtinbase) cast(t reflect.Type) Context {
    if reflect.TypeOf((*builtinbase)(nil)) == t { return c }
    if reflect.TypeOf(c) == t { return c }
    return c.evocation.cast(t)
}
func (c *builtinbase) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case builtincalls:
        var s = c.x.String()+"(%s)"
        if x, y := c.evocation.do(ctx, op).(string); y && x != "" {
            s = fmt.Sprintf(x, s)
        }
        return s
    }
    return c.evocation.do(ctx, op)
}
func (c *builtinbase) ts(string) string {
    if true {
        var s = c.defs.String()
        if s != "" { s += " " }
        return "{="+c.x.String()+" "+s+ts(c.Context)+"}"
    } else if false {
        return "{="+c.x.String()+" "+ts(c.evocation)+"}"
    } else {
        return "{="+c.x.String()+" "+ts(c.Context)+"}"
    }
}
func (c *builtinbase) _a(force bool) (skip bool) {
    c.evocation.a = expand(c, c.evocation.a...)
    return
}

type builtin_x interface{ x() any }
var builtin_x_t = reflect.TypeOf((*builtin_x)(nil)).Elem()
var builtins = map[string]reflect.Type {
    `typeof`:    reflect.TypeOf((*__typeof)(nil)).Elem(),
    `origin`:    reflect.TypeOf((*__origin)(nil)).Elem(),
    `defined`:   reflect.TypeOf((*__defined)(nil)).Elem(),

    `position`:  reflect.TypeOf((*__position)(nil)).Elem(),
    `date`:      reflect.TypeOf((*__date)(nil)).Elem(),

    `debug`:     reflect.TypeOf((*__debug)(nil)).Elem(),
    `error`:     reflect.TypeOf((*__error)(nil)).Elem(),
    `warning`:   reflect.TypeOf((*__warning)(nil)).Elem(),
    `assert`:    reflect.TypeOf((*__assert)(nil)).Elem(),
    `sure`:      reflect.TypeOf((*__sure)(nil)).Elem(),

    `defor`:     reflect.TypeOf((*__defor)(nil)).Elem(), // $(defor $(x),$(y),$(z))  <=>  $(ifdef x,$(y),$(z))
    `or`:        reflect.TypeOf((*__or)(nil)).Elem(),
    `and`:       reflect.TypeOf((*__and)(nil)).Elem(),
    `not`:       reflect.TypeOf((*__not)(nil)).Elem(),
    `xor`:       reflect.TypeOf((*__xor)(nil)).Elem(),

    `equal`:     reflect.TypeOf((*__equal)(nil)).Elem(),
    `ne`:        reflect.TypeOf((*__unequal)(nil)).Elem(),
    `not-equal`: reflect.TypeOf((*__unequal)(nil)).Elem(),
    `match`:     reflect.TypeOf((*__match)(nil)).Elem(),

    `greater`:   reflect.TypeOf((*__greater)(nil)).Elem(),
    `less`:      reflect.TypeOf((*__less)(nil)).Elem(),

    `case`:      reflect.TypeOf((*__case)(nil)).Elem(),
    `if`:        reflect.TypeOf((*__if)(nil)).Elem(),
    `ifeq`:      reflect.TypeOf((*__ifeq)(nil)).Elem(),
    `ifne`:      reflect.TypeOf((*__ifne)(nil)).Elem(),
    `ifarg`:     reflect.TypeOf((*__ifarg)(nil)).Elem(),
    `ifdef`:     reflect.TypeOf((*__ifdef)(nil)).Elem(),

    `for`:       reflect.TypeOf((*__for)(nil)).Elem(),
    `foreach`:   reflect.TypeOf((*__foreach)(nil)).Elem(),
    `count`:     reflect.TypeOf((*__count)(nil)).Elem(),

    `auto`:      reflect.TypeOf((*__auto)(nil)).Elem(),
    `var`:       reflect.TypeOf((*__var)(nil)).Elem(),

    `call`:      reflect.TypeOf((*__call)(nil)).Elem(),
    `closure`:   reflect.TypeOf((*__closure)(nil)).Elem(),
    `delegate`:  reflect.TypeOf((*__delegate)(nil)).Elem(),
    `defs`:      reflect.TypeOf((*__defs)(nil)).Elem(),

    `value`:     reflect.TypeOf((*__value)(nil)).Elem(),
    `list`:      reflect.TypeOf((*__list)(nil)).Elem(),
    `env`:       reflect.TypeOf((*__env)(nil)).Elem(),

    `shell`:     reflect.TypeOf((*__shell)(nil)).Elem(),
    `which`:     reflect.TypeOf((*__which)(nil)).Elem(),

    `plus`:      reflect.TypeOf((*__plus)(nil)).Elem(),
    `minus`:     reflect.TypeOf((*__minus)(nil)).Elem(),
    `multiply`:  reflect.TypeOf((*__multiply)(nil)).Elem(),
    `mul`:       reflect.TypeOf((*__multiply)(nil)).Elem(),
    `divide`:    reflect.TypeOf((*__divide)(nil)).Elem(),
    `div`:       reflect.TypeOf((*__divide)(nil)).Elem(),

    `join`:       reflect.TypeOf((*__join)(nil)).Elem(),
    `compose`:    reflect.TypeOf((*__compose)(nil)).Elem(), // concat
    `quote`:      reflect.TypeOf((*__quote)(nil)).Elem(),
    `unique`:     reflect.TypeOf((*__unique)(nil)).Elem(),

    `split`:            reflect.TypeOf((*__splitstring)(nil)).Elem(),
    `split-string`:     reflect.TypeOf((*__splitstring)(nil)).Elem(), // TODO: remove it?
    `split-quote`:      reflect.TypeOf((*__splitquote)(nil)).Elem(),
    `split-quote-join`: reflect.TypeOf((*__splitquotejoin)(nil)).Elem(),
    `split-join-quote`: reflect.TypeOf((*__splitjoinquote)(nil)).Elem(),

    `field`:        reflect.TypeOf((*__field)(nil)).Elem(),
    `fields`:       reflect.TypeOf((*__fields)(nil)).Elem(),

    `uses`:         reflect.TypeOf((*__uses)(nil)).Elem(),

    `bare`:         reflect.TypeOf((*__bare)(nil)).Elem(),
    `path`:         reflect.TypeOf((*__path)(nil)).Elem(),
    `word`:         reflect.TypeOf((*__word)(nil)).Elem(),
    `finalize`:     reflect.TypeOf((*__finalize)(nil)).Elem(),
    `resolve`:      reflect.TypeOf((*__resolve)(nil)).Elem(),
    `strip`:        reflect.TypeOf((*__strip)(nil)).Elem(),
    `trim`:         reflect.TypeOf((*__trim)(nil)).Elem(),
    // `trim-space`:   reflect.TypeOf((*__trimspace)(nil)).Elem(),
    `trim-left`:    reflect.TypeOf((*__trimleft)(nil)).Elem(),
    `trim-right`:   reflect.TypeOf((*__trimright)(nil)).Elem(),
    `trim-prefix`:  reflect.TypeOf((*__trimprefix)(nil)).Elem(),
    `trim-suffix`:  reflect.TypeOf((*__trimsuffix)(nil)).Elem(),
    `trim-ext`:     reflect.TypeOf((*__trimext)(nil)).Elem(),

    `gitdir`:       reflect.TypeOf((*__gitdir)(nil)).Elem(),

    `addprefix`:    reflect.TypeOf((*__addprefix)(nil)).Elem(),
    `addsuffix`:    reflect.TypeOf((*__addsuffix)(nil)).Elem(),

    `title`:        reflect.TypeOf((*__title)(nil)).Elem(),
    `indent`:       reflect.TypeOf((*__indent)(nil)).Elem(),
    `substring`:    reflect.TypeOf((*__substring)(nil)).Elem(),
    `uppercase`:    reflect.TypeOf((*__uppercase)(nil)).Elem(),
    `lowercase`:    reflect.TypeOf((*__lowercase)(nil)).Elem(),

    // https://www.gnu.org/software/make/manual/html_node/Text-Functions.html
    `subst`:        reflect.TypeOf((*__subst)(nil)).Elem(),
    `patsubst`:     reflect.TypeOf((*__patsubst)(nil)).Elem(),

    `contains`:     reflect.TypeOf((*__contains)(nil)).Elem(),
    `filter`:       reflect.TypeOf((*__filter)(nil)).Elem(),
    `filter-out`:   reflect.TypeOf((*__filterout)(nil)).Elem(),

    `decode-base64`: reflect.TypeOf((*__decodebase64)(nil)).Elem(),
    `encode-base64`: reflect.TypeOf((*__encodebase64)(nil)).Elem(),
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

    `ext`:        reflect.TypeOf((*__ext)(nil)).Elem(),

    `base` :      reflect.TypeOf((*__base1)(nil)).Elem(),
    `base2`:      reflect.TypeOf((*__base2)(nil)).Elem(),
    `base3`:      reflect.TypeOf((*__base3)(nil)).Elem(),
    `base4`:      reflect.TypeOf((*__base4)(nil)).Elem(),
    `base5`:      reflect.TypeOf((*__base5)(nil)).Elem(),
    `base6`:      reflect.TypeOf((*__base6)(nil)).Elem(),
    `base7`:      reflect.TypeOf((*__base7)(nil)).Elem(),
    `base8`:      reflect.TypeOf((*__base8)(nil)).Elem(),
    `base9`:      reflect.TypeOf((*__base9)(nil)).Elem(),
    `bases`:      reflect.TypeOf((*__bases)(nil)).Elem(),

    `chopdir`:    reflect.TypeOf((*__chopdir)(nil)).Elem(),

    `dir`:        reflect.TypeOf((*__dir)(nil)).Elem(),
    `dir2`:       reflect.TypeOf((*__dir2)(nil)).Elem(),
    `dir3`:       reflect.TypeOf((*__dir3)(nil)).Elem(),
    `dir4`:       reflect.TypeOf((*__dir4)(nil)).Elem(),
    `dir5`:       reflect.TypeOf((*__dir5)(nil)).Elem(),
    `dir6`:       reflect.TypeOf((*__dir6)(nil)).Elem(),
    `dir7`:       reflect.TypeOf((*__dir7)(nil)).Elem(),
    `dir8`:       reflect.TypeOf((*__dir8)(nil)).Elem(),
    `dir9`:       reflect.TypeOf((*__dir9)(nil)).Elem(),
    `dirs`:       reflect.TypeOf((*__dirs)(nil)).Elem(),

    `undir`:      reflect.TypeOf((*__undir1)(nil)).Elem(),
    `undir2`:     reflect.TypeOf((*__undir2)(nil)).Elem(),
    `undir3`:     reflect.TypeOf((*__undir3)(nil)).Elem(),
    `undir4`:     reflect.TypeOf((*__undir4)(nil)).Elem(),
    `undir5`:     reflect.TypeOf((*__undir5)(nil)).Elem(),
    `undir6`:     reflect.TypeOf((*__undir6)(nil)).Elem(),
    `undir7`:     reflect.TypeOf((*__undir7)(nil)).Elem(),
    `undir8`:     reflect.TypeOf((*__undir8)(nil)).Elem(),
    `undir9`:     reflect.TypeOf((*__undir9)(nil)).Elem(),
    `undirs`:     reflect.TypeOf((*__undirs)(nil)).Elem(),

    `reldir`:       reflect.TypeOf((*__reldir)(nil)).Elem(),
    `relative-dir`: reflect.TypeOf((*__reldir)(nil)).Elem(),

    `file`:         reflect.TypeOf((*__file)(nil)).Elem(),
    `stat`:         reflect.TypeOf((*__stat)(nil)).Elem(),// stat (deprecates file-exists)
    `glob`:         reflect.TypeOf((*__glob)(nil)).Elem(),
    `wildcard`:     reflect.TypeOf((*__wildcard)(nil)).Elem(),

    `read-dir`:     reflect.TypeOf((*__readdir)(nil)).Elem(),  // io/ioutil/ioutil.go
    `read-file`:    reflect.TypeOf((*__readfile)(nil)).Elem(), // io/ioutil/ioutil.go

    `grep`:         reflect.TypeOf((*__grep)(nil)).Elem(),

    `untraversed`:  reflect.TypeOf((*__untraversed)(nil)).Elem(),

    // commands ------------------------------------------------------------------
    `print`:        reflect.TypeOf((*__print)(nil)).Elem(),
    `printf`:       reflect.TypeOf((*__printf)(nil)).Elem(),

    `plain`:        reflect.TypeOf((*__plain)(nil)).Elem(),

    `append`:       reflect.TypeOf((*__append)(nil)).Elem(),
    // `pop`:          reflect.TypeOf((*__pop)(nil)).Elem(),

    `write-file`:   reflect.TypeOf((*__writefile)(nil)).Elem(), // io/ioutil/ioutil.go
    `touch-file`:   reflect.TypeOf((*__readfile)(nil)).Elem(),  // io/ioutil/ioutil.go

    `mkdir`:        reflect.TypeOf((*__mkdir)(nil)).Elem(),     // os/file.go
    `chdir`:        reflect.TypeOf((*__chdir)(nil)).Elem(),     // os/file.go
    `rename`:       reflect.TypeOf((*__rename)(nil)).Elem(),    // os/file.go
    `remove`:       reflect.TypeOf((*__remove)(nil)).Elem(),    // os/file_*.go
    `truncate`:     reflect.TypeOf((*__truncate)(nil)).Elem(),  // os/file_*.go
    `link`:         reflect.TypeOf((*__link)(nil)).Elem(),      // os/file_*.go
    `symlink`:      reflect.TypeOf((*__symlink)(nil)).Elem(),   // os/file_*.go

    `serve-http`:   reflect.TypeOf((*__servehttp)(nil)).Elem(),

    `return`:       reflect.TypeOf((*__return)(nil)).Elem(),
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
        val.SetBool(v.true(ctx))
    case reflect.Float32, reflect.Float64:
        val.SetFloat(v.float(ctx))
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        val.SetInt(v.int(ctx))
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        val.SetUint(uint64(v.int(ctx)))
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
            errostack(pc(ctx,v), 16, "option type unsupported: %v → %v, %v", ts(v), val.Kind(), val.Type()).trace()
        }
    case reflect.Ptr:
        switch val.Type().Elem().String() {
        case "smart.fullname":
            if t := (as{v}.fullname(fullfile_ctx{ctx})); t.Value != nil {
                val.Set(reflect.ValueOf(&t))
            } else {
                note(ctx, "%v → %v", v, (as{v}.file(ctx)))
                errostack(pc(ctx,v), 16, "not a file: %v → %s", ts(v), ts(v.expand(ctx))).trace()
            }
        case "smart.file":
            if t := (as{v}.file(ctx)); t != nil {
                val.Set(reflect.ValueOf(t))
            } else {
                errostack(pc(ctx,v), 16, "not a file: %v → %s", ts(v), ts(v.expand(ctx))).trace()
            }
        case "regexp.Regexp":
            if rx, e := regexp.Compile(v.string(ctx)); e == nil {
                val.Set(reflect.ValueOf(rx))
            } else {
                errostack(pc(ctx,v), 16, "wrong regexp: %v: %v", ts(v), e).trace()
            }
        default:
            errostack(pc(ctx,v), 16, "option type unsupported: %v → %v, %v", ts(v), val.Elem().Kind(), val.Type().Elem()).trace()
        }
    default:
        switch val.Type().String() {
        case "fs.FileMode", "os.FileMode": // aka. reflect.Uint32
            var t = v.int(ctx)
            if t == 0 { warn(pc(ctx,v), "zero file mode").debug() }
            val.SetUint(uint64(t))
        case "regexp.Regexp": // aka. reflect.Ptr
            errostack(pc(ctx,v), 16, "TODO: regexp: %v → %v, %v", ts(v), val.Kind(), val.Type()).trace()
        default:
            errostack(pc(ctx,v), 16, "option type unsupported: %v → %v, %v", ts(v), val.Kind(), val.Type()).trace()
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
            case        flag: f, value = t, _boolean(t.Position(), true)
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
    if args == nil { return }

    if opts.Kind() != reflect.Ptr {
        erro(ctx, "opts must be ptr: %v", opts.Kind()).trace()
    } else if opts = opts.Elem(); opts.Kind() != reflect.Struct {
        erro(ctx, "opts is not ptr of struct: %v", opts.Kind()).trace()
    }

    rest = merge(args...)

    var builtin, general, modifier, dots reflect.Value
    var ot = opts.Type()
    for i := 0; i < ot.NumField(); i += 1 {
        var ft, fv = ot.Field(i), opts.Field(i)
        if ft.Tag == "..." {
            dots = fv
        } else if t := fv.Type(); fv.Kind() != reflect.Struct {
            if ft.Anonymous && ft.Name == "Context" {
                if t.String() == "smart.Context" {
                    continue
                }
            }
            rest = _opt(ctx, ft.Tag, fv, rest...)
        } else if !ft.Anonymous {
            continue
        } else if ft.Name == "general_opts" {
            general = fv.Addr()
        } else if strings.HasPrefix(ft.Name, "__") {
            if builtin.IsValid() { note(ctx, "embedded multiple builtins: %v", ft).debug(3) }
            builtin = fv.Addr()
        } else if strings.HasPrefix(ft.Name, "modifier_") {
            if modifier.IsValid() { note(ctx, "embedded multiple modifiers: %v", ft).debug(3) }
            modifier = fv.Addr()
        }
    }
    if  general.IsValid() { rest = _opts(ctx,  general, rest) }
    if  builtin.IsValid() { rest = _opts(ctx,  builtin, rest) }
    if modifier.IsValid() { rest = _opts(ctx, modifier, rest) }
    if dots.IsValid() && rest != nil {
        _set(ctx, dots, ease(ctx, rest))
        rest = nil
    }
    return
}
func parse_opts(ctx Context, store any, args ...Value) (rest []Value) {
    return _opts(ctx, reflect.ValueOf(store), args)
}

// see https://go.dev/doc/tutorial/generics
func _opts_[Opts any](ctx Context, args ...Value) (opts Opts, res []Value) {
    res = parse_opts(ctx, &opts, args...)
    return
}

func _parseHeadArgs(ctx Context, store any, args ...Value) (head, rest []Value) {
    if len(args) == 0 {
        // zero args
    } else if head = parse_opts(ctx, store, args[0]); len(head) > 0 {
        rest = args[1:] //xmerge(ctx, args[1:]...)
    } else if len(args) == 1 {
        // done
    } else if head = xmerge(ctx, args[1]); len(args) > 2 {
        rest = args[2:] //xmerge(ctx, args[2:]...)
    }
    return
}

func _parseHeadArgsMerge(ctx Context, store any, args ...Value) (res []Value) {
    var head, rest = _parseHeadArgs(ctx, store, args...)
    res = append(head, rest...)
    return
}

func _parseHeadArgsRequired(ctx Context, store any, args ...Value) (head, rest []Value) {
    head, rest = _parseHeadArgs(ctx, store, args...)
    if len(head) == 0 || len(rest) == 0 {
        erro(ctx, "insufficient number of arguments").trace()
    }
    return
}

type __noop struct { builtinbase }
func (ctx *__noop) inner() Context { return &ctx.builtinbase }
func (ctx *__noop) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__noop) x() (_ any) { return }

type __typeof struct { builtinbase }
func (ctx *__typeof) inner() Context { return &ctx.builtinbase }
func (ctx *__typeof) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__typeof) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var vals []Value
    for _, a := range ctx.a {
        // Arguments are passed in a list:
        //   $(fun abc)             args: (abc)
        //   $(fun a,b,c)           args: (a),(b),(c)
        //   $(fun a b c,1 2 3)     args: (a b c),(1 2 3)
        vals = append(vals, _word(a.Position(), typeof(a)))
    }
    return vals
}

type __origin struct { builtinbase }
func (ctx *__origin) inner() Context { return &ctx.builtinbase }
func (ctx *__origin) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__origin) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var vals []Value
    var scope = _scope(ctx)
    for _, a := range ctx.a {
        if s := a.string(ctx); s == "" {
            vals = append(vals, _null(a.Position()))
        } else if d := scope.finddef(s); d != nil {
            vals = append(vals, _word(a.Position(), d.o.String()))
        } else {
            vals = append(vals, _null(a.Position()))
        }
    }
    return vals
}

type __defined struct { builtinbase }
func (ctx *__defined) inner() Context { return &ctx.builtinbase }
func (ctx *__defined) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__defined) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var scope = _scope(ctx)
    for _, arg := range ctx.a {
        if d := scope.finddef(arg.string(ctx)); d != nil && !isTrivial(d.value) {
            return true
        }
    }
    return
}

type __position struct { builtinbase
    filename bool `filename`
    filenameQuoted bool `quote-filename,quoted-filename`
    line bool `ln,line`
    column bool `col,column`
    addLine int `add,add-line`
    addColumn int `add-column`
}
func (ctx *__position) inner() Context { return &ctx.builtinbase }
func (ctx *__position) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__position) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var vals []Value
    var pos = _position(ctx)
    if ctx.filename {
        vals = append(vals, _strlit(pos, pos.Filename))
    } else if ctx.filenameQuoted {
        var s = pos.Filename //strconv.Quote(pos.Filename)
        vals = append(vals, _strlit(pos, "\""+s+"\""))
    }

    if ctx.line   { vals = append(vals, _decimal(pos, int64(pos.Line + ctx.addLine))) }
    if ctx.column { vals = append(vals, _decimal(pos, int64(pos.Column + ctx.addColumn))) }

    if len(vals) == 0 { return _strlit(pos, pos.String()) }
    if len(vals) == 1 { return vals[0] }
    return vals
}

type __date struct { builtinbase
    time bool `time,now`
}
func (ctx *__date) inner() Context { return &ctx.builtinbase }
func (ctx *__date) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__date) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    if t := time.Now(); len(ctx.a) > 0 {
        var vals []Value
        for _, a := range ctx.a {
            var s string
            if s = a.string(ctx); s == "" {
                s = t.String()
            } else if s = t.Format(s); s == "" {
                s = fmt.Sprintf("%v", t)
            }
            vals = append(vals, _strlit(a.Position(), s))
        }
        return vals
    } else if ctx.time {
        res = makeTime(_position(ctx), t)
    } else {
        res = makeDate(_position(ctx), t)
    }
    return
}

type __debug struct { builtinbase
    s int `stack`
    n int `num`
}
func (ctx *__debug) inner() Context { return &ctx.builtinbase }
func (ctx *__debug) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__debug) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var s bytes.Buffer
    for i, a := range ctx.a {
        if i > 0 { fmt.Fprintf(&s, " ") }
        fmt.Fprintf(&s, "%s", a.string(ctx))
    }
    if hook := _universe(ctx).hooks.debug; hook != nil {
        hook(ctx, s.String(), ctx.a)
    } else {
        warnstack(ctx, ctx.s, "%s", s.String()).debug(ctx.n)
    }
    return
}

type __error struct { builtinbase }
func (ctx *__error) inner() Context { return &ctx.builtinbase }
func (ctx *__error) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__error) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var s bytes.Buffer
    for i, a := range ctx.a {
        if i > 0 { fmt.Fprintf(&s, " ") }
        fmt.Fprintf(&s, "%s", a.string(ctx))
    }

    errostack(ctx, 5, "%s", s.String()).trace()
    return
}

type __warning struct { builtinbase }
func (ctx *__warning) inner() Context { return &ctx.builtinbase }
func (ctx *__warning) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__warning) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var s bytes.Buffer
    for i, a := range ctx.a {
        if i > 0 { fmt.Fprintf(&s, " ") }
        fmt.Fprintf(&s, "%s", a.string(ctx))
    }
    warn(ctx, "%s", s).debug()
    return
}

type __assert struct { builtinbase ; msg string `msg,message` }
func (ctx *__assert) inner() Context { return &ctx.builtinbase }
func (ctx *__assert) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__assert) x() (res any) {
    var d = ctx.debug ; if d < 1 { d = 1 }
    var s = ctx.stack ; if s < 1 { s = 1 }
    var t = diagError ; if ctx.warning { t = diagWarn }
    var hook = _universe(ctx).hooks.assert

    if false && !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if ctx.a == nil && hook != nil && !hook(ctx, nil, false) {
        prompt(ctx, "assert: %v\n", ctx.a)
        diagstack(ctx, s, t).debug(d)
    }

    for _, a := range ctx.a {
        if a == nil {
            erro(ctx, "nil argument").trace()
            continue
        }

        var ctx = pc(ctx, a)
        var yes = a.true(ctx)
        if hook != nil && hook(ctx, a, yes) || yes {
            continue
        }

        if false {
            var v = a.expand(_final(ctx))
            prompt(ctx, "assert: %v ⇒ %v: %v\n", ts(a), ts(v))
            diagstack(ctx, s, t, "%v → %v ⇒ '%s'", ts(a), ts(v), v.string(ctx)).debug(d)
        } else if true {
            diagstack(ctx, s, t, "%v ⇒ '%s'", ts(a), a.string(ctx)).debug(d)
        } else {
            diagstack(ctx, s, t, "%v", ts(a)).debug(d)
        }
    }

    if ctx.fail { panic(_failure(ctx)) }
    return
}

type __sure struct { builtinbase }
func (ctx *__sure) inner() Context { return &ctx.builtinbase }
func (ctx *__sure) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__sure) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    for _, a := range ctx.a {
        if !a.true(ctx) {
            erro(ctx, "assert: %v", ts(a)).trace()
        }
    }
    return ctx.a
}

// $(defor $(x),$(y),$(z)) is identical to $(if $(defined $(x)),$(x),...)
type __defor struct { builtinbase } // aka. defined-or
func (ctx *__defor) inner() Context { return &ctx.builtinbase }
func (ctx *__defor) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__defor) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    for _, a := range merge(ctx.a...) {
        erro(ctx, "TODO: %v", ts(a)).trace()

        var unres bool
        if unres {
            continue
        } else {
            res = a
            break
        }
    }
    return
}

type __or struct { builtinbase }
func (ctx *__or) inner() Context { return &ctx.builtinbase }
func (ctx *__or) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__or) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    for _, a := range merge(ctx.a...) {
        if a.true(ctx) { return a }
    }
    return
}

type __and struct { builtinbase }
func (ctx *__and) inner() Context { return &ctx.builtinbase }
func (ctx *__and) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__and) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    for _, a := range merge(ctx.a...) {
        if a.true(ctx) { res = a } else { return nil }
    }
    return
}

// $(not x y z) ⇒ (not (or x y z))
// $(not x,y,z) ⇒ (and (not x) (not y) (not z))
type __not struct { builtinbase }
func (ctx *__not) inner() Context { return &ctx.builtinbase }
func (ctx *__not) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__not) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    var t bool
    for _, a := range ctx.a { if t = a.true(ctx); t { break } }
    return !t
}

type __xor struct { builtinbase }
func (ctx *__xor) inner() Context { return &ctx.builtinbase }
func (ctx *__xor) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__xor) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if vals := merge(ctx.a...); len(vals) > 1 {
        var t = vals[0].true(ctx)
        for _, a := range vals[1:] {
            if a.true(ctx) != t {
                return _boolean(a.Position(), true)
            }
        }
    }
    return
}

type __unequal struct { builtinbase }
func (ctx *__unequal) inner() Context { return &ctx.builtinbase }
func (ctx *__unequal) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__unequal) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if len(ctx.a) != 2 {
        erro(ctx, "unequal: wrong number of arguments: %v", ctx.a)
        erro(ctx, "try: $(unequal <value-list>,<value-list>)").trace()
    }

    var a = ctx.a[0].expand(_final(ctx))
    var b = ctx.a[1].expand(_final(ctx))
    var t = a.cmp(ctx, b) != cmpEqual

    if t {
        return _boolean(_position(ctx), true)
    } else if n := ctx.debug; n>0 {
        if l, y := a.(*list); y {
            var v = l.elems[0]
            warn(ctx, "unequal: a: %T(len=%d), %T %v", a, len(l.elems), v, v)
        } else {
            warn(ctx, "unequal: a: %T %v", a, a)
        }
        if l, y := b.(*list); y {
            var v = l.elems[0]
            warn(ctx, "unequal: b: %T(len=%d), %T %v", b, len(l.elems), v, v)
        } else {
            warn(ctx, "unequal: b: %T %v", b, b)
        }
        warnstack(ctx, n, "unequal: %v", t).debug(n)
    } else if len(ctx.a)>2 {
        warnstack(ctx, 1, "unequal: extra args specified: %v", ctx.a[2]).debug()
    }
    return
}

type __equal struct { builtinbase; str bool `str,string` }
func (ctx *__equal) inner() Context { return &ctx.builtinbase }
func (ctx *__equal) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__equal) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if len(ctx.a) != 2 {
        erro(ctx, "wrong number of arguments: %v", ctx.a)
        note(ctx, "try: $(equal <value-list>,<value-list>)").trace()
    }

    a, b := ctx.a[0], ctx.a[1]

    if ctx.str {
        if a.string(ctx) == b.string(ctx) { return true }
    } else {
        if a.cmp(ctx, b) == cmpEqual { return true }
    }
    return
}

type __greater struct { builtinbase; str bool `str,string` }
func (ctx *__greater) inner() Context { return &ctx.builtinbase }
func (ctx *__greater) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__greater) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if len(ctx.a) != 2 {
        erro(ctx, "wrong number of arguments: %v", ctx.a)
        note(ctx, "try: $(greater <value-list>,<value-list>)").trace()
    }

    a, b := ctx.a[0], ctx.a[1]

    if ctx.str {
        if a.string(ctx) > b.string(ctx) { return true }
    } else {
        if a.cmp(ctx, b) == cmpGreater { return true }
    }
    return
}

type __less struct { builtinbase; str bool `str,string` }
func (ctx *__less) inner() Context { return &ctx.builtinbase }
func (ctx *__less) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__less) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if len(ctx.a) != 2 {
        erro(ctx, "wrong number of arguments: %v", ctx.a)
        note(ctx, "try: $(greater <value-list>,<value-list>)").trace()
    }

    a, b := ctx.a[0], ctx.a[1]

    if ctx.str {
        if a.string(ctx) < b.string(ctx) { return true }
    } else {
        if a.cmp(ctx, b) == cmpSmaller { return true }
    }
    return
}

// $(match val1 val2 val3, a b c d...)
// $(match -rx=r1 -rx=r2 -rx=r3, a b c d...)
type __match struct { builtinbase
    regexps []*regexp.Regexp //`re,rx,reg,regex,regexp`
    negated bool `ne,neg,negated,negative,not`
    all bool `all`
}
func (ctx *__match) inner() Context { return &ctx.builtinbase }
func (ctx *__match) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__match) x() (result any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if n := len(ctx.a); n < 2 {
        erro(ctx, "wrong arguments, try: $(match <regexp-list>,<value-list-1>,...)").trace()
    }

    var leftList, rightList []Value

    if true {
        leftList, rightList = xmerge(ctx, ctx.a[0]), xmerge(ctx, ctx.a[1:]...)
    } else {
        leftList, rightList = merge(ctx.a[0]), merge(ctx.a[1:]...)
    }

    var res *boolean

    if ctx.negated {
        defer func() {
            if res != nil {
                res.bool = !res.bool
            } else {
                result = _boolean(_position(ctx), true)
            }
        } ()
    }

    for _, left := range leftList {
        for _, right := range rightList {
            var matched bool
            if !left.patterned(ctx) && right.patterned(ctx) {
                matched, _, _ = right.match(ctx, left)
            } else {
                matched, _, _ = left.match(ctx, right)
            }
            if matched {
                if res == nil { res = _boolean(_position(ctx), true) }
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
func (ctx *__match) _x() (res any) {
    var patList, valList []Value
    if n := len(ctx.a); n < 1 {
        erro(ctx, "wrong arguments, try: $(match <regexp-list>,<value-list>,...)").trace()
    }

    if len(ctx.a) > 1 {
        patList = merge(ctx.a[0])
        valList = merge(ctx.a[1:]...)
    } else {
        valList = merge(ctx.a[0])
    }
    if ctx.debug > 0 {
        var ( n = len(ctx.a) ; d = ctx.debug )
        note(ctx, "match: %v %v %v, %d", ctx.regexps, patList, valList, n).debug(d)
    }

    var pos = _position(ctx)
ForValList:
    for _, val := range valList {
        if isTrivial(val) { continue ForValList }

        var str = val.string(ctx)
        for _, rx := range ctx.regexps {
            var matched = rx.MatchString(str);
            if ctx.negated { matched = !matched }
            if matched {
                if ctx.all {
                    if res == nil { res = _boolean(pos, true) }
                } else {
                    return _boolean(pos, true)
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
                    if res == nil { res = _boolean(pos, true) }
                } else {
                    return _boolean(pos, true)
                }
            } else if ctx.all {
                return nil
            }
        }

        if ctx.debug > 0 {
            note(ctx, "match: %v", str)
            note(ctx, "match: %v %T", val, val).debug()
        }
    }
    return
}

// 1: $(case     (a 'xxx') (b 'yyy') (c 'zzz') (yes 'else'))
// 2: $(case val (a 'xxx') (b 'yyy') (c 'zzz') ('if none or nil'))
// 3: $(case val (a 'xxx') (b 'yyy') (c 'zzz') (- 'if none or nil'))
// 4: $(case val (a 'xxx') (b 'yyy') (c -) (- -))
type __case struct { builtinbase }
func (ctx *__case) inner() Context { return &ctx.builtinbase }
func (ctx *__case) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__case) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var val Value
    var args = merge(ctx.a...)
    if len(args) == 0 {
        return
    }
    if _, y := args[0].(*group); !y {
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
        for _, v := range g.elems[1:] {
            if f, y := v.(flag); !y || isNull(f.Value) {
                vals = append(vals, v)
            }
        }
        return vals
    } else {
        errostack(pc(ctx,arg), 3, "unexpected case: %v", tv(arg)).trace()
    }}
    return
}

type __if struct { builtinbase }
func (ctx *__if) inner() Context { return &ctx.builtinbase }
func (ctx *__if) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__if) ts(string) string {
    if n := len(ctx.a); n > 0 {
        var s = ctx.a[0].String()
        if n > 1 { s += ","+ctx.a[1].String() }
        if n > 2 { s += ","+ctx.a[2].String() }
        if n > 3 { s += ","+ctx.a[3].String() }
        if s != "" { s += " " }
        return "{=if "+s+ts(ctx.Context)+"}"
    } else {
        return "{=if "+ts(ctx.Context)+"}"
    }
}
func (ctx *__if) x() (_ any) {
    if 1 < len(ctx.a) {
        ctx.a[0] = ctx.a[0].expand(ctx)
        if ctx.a[0].true(ctx) {
            return ctx.a[1].expand(ctx)
        } else {
            return expand(ctx, ctx.a[2:]...)
        }
    }
    return
}

type __ifarg struct { builtinbase }
func (ctx *__ifarg) inner() Context { return &ctx.builtinbase }
func (ctx *__ifarg) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__ifarg) x() (_ any) {
    if 1 < len(ctx.a) {
        ctx.a[0] = ctx.a[0].expand(ctx)
        var s = ctx.a[0].string(ctx)
        if d := auto_find(ctx, s); d != nil && !isTrivial(d.value) {
            return ctx.a[1].expand(ctx)
        } else {
            return expand(ctx, ctx.a[2:]...)
        }
    }
    return
}

type __ifdef struct { builtinbase }
func (ctx *__ifdef) inner() Context { return &ctx.builtinbase }
func (ctx *__ifdef) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__ifdef) x() (_ any) {
    if 1 < len(ctx.a) {
        ctx.a[0] = ctx.a[0].expand(ctx)
        var s = ctx.a[0].string(ctx)
        if d := _scope(ctx).finddef(s); d != nil && !isTrivial(d.value) {
            return ctx.a[1].expand(ctx)
        } else {
            return expand(ctx, ctx.a[2:]...)
        }
    }
    return
}

type __ifeq struct { builtinbase }
func (ctx *__ifeq) inner() Context { return &ctx.builtinbase }
func (ctx *__ifeq) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__ifeq) x() (_ any) {
    if 2 < len(ctx.a) {
        ctx.a[0] = ctx.a[0].expand(ctx)
        ctx.a[1] = ctx.a[1].expand(ctx)
        if a, b := ctx.a[0], ctx.a[1]; equal(ctx, a, b) {
            return ctx.a[2].expand(ctx)
        } else {
            return expand(ctx, ctx.a[3:]...)
        }
    }
    return
}

type __ifne struct { builtinbase }
func (ctx *__ifne) inner() Context { return &ctx.builtinbase }
func (ctx *__ifne) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__ifne) x() (_ any) {
    if 2 < len(ctx.a) {
        ctx.a[0] = ctx.a[0].expand(ctx)
        ctx.a[1] = ctx.a[1].expand(ctx)
        if a, b := ctx.a[0], ctx.a[1]; !equal(ctx, a, b) {
            return ctx.a[2].expand(ctx)
        } else {
            return expand(ctx, ctx.a[3:]...)
        }
    }
    return
}

type __for struct { builtinbase ; empty bool `allow-empty,empty` }
func (ctx *__for) inner() Context { return &ctx.builtinbase }
func (ctx *__for) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__for) x() (res any) {
    erro(ctx, "TODO: $(for): %v", ts(ctx.a)).trace()
    return
}

type __foreach struct { builtinbase
    empty  bool `allow-empty,empty`
    unique bool `unique`
}
func (ctx *__foreach) inner() Context { return &ctx.builtinbase }
func (ctx *__foreach) cast(t reflect.Type) Context {
    if reflect.TypeOf(partial{}) == t { return nil }
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__foreach) x() (res any) {
    if len(ctx.a) == 0 { return }

    ctx.a[0] = ctx.a[0].expand(ctx)

    var vals []Value
    var args = merge(ctx.a[0])
    if len(args) == 0 { return }
    if checkpoints && truly(ctx, is_test_mode{}) {
        defer ctx.check(&args, &vals)
    }

    var um map[uint64]Value
    if ctx.unique { um = make(map[uint64]Value) }

    for _, val := range args {
        if isEmpty(val) {
            if !ctx.empty { continue }
        }

        if ctx.unique {
            t := val.hash(ctx)
            if x, y := um[t]; y {
                if checkpoints && !equal(ctx, x, val) {
                    erro(ctx, "%v != %v", ts(x), ts(val)).debug()
                }
                continue
            } else {
                um[t] = val
            }
        }

        // NOTE: don't use defStatic (it's for codeblock auto)
        ctx.set(ctx, defVoid, "_", redis(val))

        for _, v := range decoupleCompoundList(xmerge(ctx, ctx.a[1:]...)...) {
            if !ctx.empty && isEmpty(v) { continue }
            if v == nil { v = _null(v.Position()) }
            vals = append(vals, v)
        }
    }
    return vals
}

type __count struct { builtinbase ; vals []Value `value` }
func (ctx *__count) inner() Context { return &ctx.builtinbase }
func (ctx *__count) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__count) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var num int64
    var vals = valvec(ctx.vals)
    for _, a := range ctx.a {
        if a.true(ctx) || vals.has2(ctx, a) { num += 1 }
    }
    return num
}

type __env struct { builtinbase }
func (ctx *__env) inner() Context { return &ctx.builtinbase }
func (ctx *__env) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__env) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var vals []Value
    for _, a := range ctx.a {
        if val := a.expand(ctx); isTrivial(val) {
            continue
        } else if s := strings.TrimSpace(val.string(ctx)); s != "" {
            vals = append(vals, _strlit(a.Position(), os.Getenv(s)))
        }
    }
    return vals
}

type __auto struct { builtinbase }
func (ctx *__auto) inner() Context { return &ctx.builtinbase }
func (ctx *__auto) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__auto) x() (_ any) {
    if !ctx.originalArgs {
        ctx.o = expand(ctx, ctx.o...)
    }
    if 0 < len(ctx.a) {
        for _, a := range merge(ctx.o...) {
            switch t := a.(type) {
            case *pair:
                if k := t.key.string(ctx); k == "" {
                    erro(pc(ctx,a), "empty name: %s : %s", t.key, ts(t.key)).trace()
                } else {
                    ctx.set(ctx, defVoid, k, t.val) // NOTE: don't use defStatic (it's codeblock auto)
                }
            default:
                erro(pc(ctx,a), "wrong auto def: %s : %s", a, ts(a)).trace()
            }
        }

        var vals = expand(ctx, ctx.a...)
        if !ctx.originalArgs { ctx.a = vals }
        if checkpoints && truly(ctx, is_test_mode{}) { ctx.check_res(vals) }
        return vals
    } else {
        return
    }
}

type __var struct { builtinbase }
func (ctx *__var) inner() Context { return &ctx.builtinbase }
func (ctx *__var) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__var) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    return
}

type __closure struct { builtinbase ; closure bool `closure` }
func (ctx *__closure) inner() Context { return &ctx.builtinbase }
func (ctx *__closure) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__closure) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    var vals []Value
    for _, a := range merge(ctx.a[0]) {
        if s := a.string(ctx); s != "" {
            if x := closure_resolve(ctx, s); x != nil {
                v := makeClosure(a.Position(), LPAREN, x, nil, ctx.a[1:]...)
                vals = append(vals, v)
                continue
            }
        }
        v := makeClosure(a.Position(), LPAREN, a, nil, ctx.a[1:]...)
        vals = append(vals, v)
    }
    return vals
}

type __delegate struct { builtinbase ; closure bool `closure` }
func (ctx *__delegate) inner() Context { return &ctx.builtinbase }
func (ctx *__delegate) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__delegate) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    var vals []Value
    var p = _project(ctx)
    for _, a := range merge(ctx.a[0]) {
        if s := a.string(ctx); s != "" {
            if x := p.resolve(ctx, s); x != nil {
                v := makeDelegate(a.Position(), LPAREN, x, nil, ctx.a[1:]...)
                vals = append(vals, v)
                continue
            }
        }
        v := makeDelegate(a.Position(), LPAREN, unresolved{a}, nil, ctx.a[1:]...)
        vals = append(vals, v)
    }
    return vals
}

type __call struct { builtinbase ; closure bool `closure` }
func (ctx *__call) inner() Context { return &ctx.builtinbase }
func (ctx *__call) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__call) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    var vals []Value
    for _, a := range merge(ctx.a[0]) {
        var x Value
        var s = a.string(ctx)
        if s == "" {
            erro(ctx, "empty string: %v : %v", a, ts(a)).trace()
        } else if ctx.closure {
            x = closure_resolve(ctx, s)
        } else {
            x = project_resolve(ctx, s)
        }
        if x == nil { x = auto_get(ctx, s) }
        if x != nil {
            if v, _, _ := evoke(ctx, x, nil, ctx.a[1:]); v != nil {
                vals = append(vals, v)
            }
        }
    }
    return vals
}

type __value struct { builtinbase ; closure bool `closure` }
func (ctx *__value) inner() Context { return &ctx.builtinbase }
func (ctx *__value) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__value) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    var vals []Value
    var p = _project(ctx)
    for _, a := range merge(ctx.a...) {
        var s = a.string(ctx)
        if s != "" {
            var x Value
            if ctx.closure {
                x = closure_resolve(ctx, s)
            } else {
                x = p.resolve(ctx, s)
            }
            if x == nil { x = auto_get(ctx, s) }
            if x != nil {
                if d, y := x.(*def); y {
                    vals = append(vals, d.value)
                }
            }
        }
    }
    return vals
}

type __defs struct { builtinbase
    n int `num,number`
    r int `capture`
}
func (ctx *__defs) inner() Context { return &ctx.builtinbase }
func (ctx *__defs) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__defs) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var names []bare
    var pats []Value
    for _, val := range merge(ctx.a...) {
        pats = append(pats, val)
    }
defsloop:
    for name, _ := range _project(ctx).elems {
        var str string
        for _, pat := range pats {
            var neg bool
            if x, y := pat.(negative); y { pat, neg = x.Value, y }

            var a, _, c = pat.match(ctx, name)
            if a && neg { continue defsloop }
            if a || neg {
                if ctx.r <= 0 || 0 == len(c) {
                    str = name
                } else if ctx.r <= len(c) {
                    str = c[ctx.r-1]
                }
            }
        }
        if str != "" {
            names = append(names, bare(str))
        }
    }
    return names
}

type __list struct { builtinbase }
func (ctx *__list) inner() Context { return &ctx.builtinbase }
func (ctx *__list) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__list) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    return ctx.a
}

type __plain struct { builtinbase
    scope_ bool `findscope,find-scope,scope`
}
func (ctx *__plain) inner() Context { return &ctx.builtinbase }
func (ctx *__plain) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__plain) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var scope = _scope(ctx)
    for _, a := range ctx.a {
        var ( o object ; s = a.string(ctx) )
        if ctx.scope_ { _, o = scope.find(s) } else { o = project_resolve(ctx, s) }
        if o == nil {
            erro(ctx, "no such symbol: %s", s).trace()
        } else if d, y := o.(*def); !y {
            erro(ctx, "not a def: %s: %v", s, typeof(o)).trace()
        } else if d.value != nil {
            d.value = d.value.expand(ctx/* , plain */)
        }
    }
    return
}

type __shell struct { builtinbase }
func (ctx *__shell) inner() Context { return &ctx.builtinbase }
func (ctx *__shell) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__shell) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var err error
    var vals []Value
    var pos = _position(ctx)
    for _, a := range ctx.a {
        var bufout, buferr bytes.Buffer
        var s = a.string(ctx)
        sh := exec.Command("sh", "-c", s)
        sh.Stdout, sh.Stderr = &bufout, &buferr
        if err = sh.Run(); err != nil {
            s = strings.TrimSpace(buferr.String())
            if !strings.HasPrefix(s, ":") { s = ":\n" + s }
            prompt(ctx, "%s%s\n", a.string(ctx), s)
            errostack(ctx, 3, "%s", err).trace()
            return
        }
        val := _strlit(pos, strings.TrimSpace(bufout.String()))
        vals = append(vals, val)
        bufout.Reset()
        buferr.Reset()
    }
    return vals
}

type __which struct { builtinbase }
func (ctx *__which) inner() Context { return &ctx.builtinbase }
func (ctx *__which) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__which) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var vals []Value
    for _, a := range ctx.a {
        if s, err := exec.LookPath(a.string(ctx)); err != nil {
            erro(ctx, "%v", err).trace()
        } else if s != "" {
            vals = append(vals, _strlit(_position(ctx), s))
        }
    }
    return vals
}

type __servehttp struct { builtinbase
    ssl bool `ssl`
    host string `host`
    port int `port`
}
func (ctx *__servehttp) inner() Context { return &ctx.builtinbase }
func (ctx *__servehttp) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__servehttp) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    if ctx.port == 0 { ctx.port = 80 }
    if ctx.ssl {
        erro(ctx, "'serve-http(-ssl)' is unimplemented yet").trace()
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
            server.Shutdown(context.Background())
        } ()
    }

    http.HandleFunc("/-/end",  quit)
    http.HandleFunc("/-/quit", quit)
    http.HandleFunc("/-/shut", quit)

    if ctx.a == nil {
        http.Handle("/", http.FileServer(http.Dir(_workdir(ctx))))
    } else {
        for _, a := range ctx.a {
            var s = a.string(ctx)
            info(ctx, "serving files %v ...", s)
            http.Handle("/", http.FileServer(http.Dir(s)))
        }
    }

    flush(ctx)

    var err = server.ListenAndServe()
    if err != nil && err != http.ErrServerClosed {
        erro(ctx, "%s", err).trace()
    }
    return
}

type __append struct { builtinbase
    auto    bool `auto`
    closure bool `closure`
}
func (ctx *__append) inner() Context { return &ctx.builtinbase }
func (ctx *__append) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__append) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if len(ctx.a) < 2 {
        erro(ctx, "insufficient number of arguments: %v", ctx.a).trace()
    }

    var names []Value
    if names = merge(ctx.a[0]); len(names) == 0 {
        warn(ctx, "append to nowhere: %v", tv(ctx.a[0])).debug()
        return
    }

    var vals []Value
    for _, a := range names {
        var s = a.string(ctx)
        var d *def
        if s == "" {
            erro(ctx, "'%v' is empty for name", a).trace()
        } else if ctx.auto {
            d = auto_find(ctx, s)
        } else if ctx.closure {
            d = closure_finddef(ctx, s)
        } else if o := project_resolve(ctx, s); o != nil {
            d, _ = o.(*def)
        }
        if d == nil {
            erro(ctx, "%v → %s is undefined", a, s)
            erro(ctx, "%v", ts(ctx)).trace()
        } else {
            if vals == nil {
                if vals = merge(ctx.a[1:]...); len(vals) == 0 {
                    warn(ctx, "append no values: %v", ctx.a[1:]).debug()
                    return
                }
            }
            d.append(ctx, vals...)
        }
    }
    return
}

type __plus struct { builtinbase
    int bool `int,integer`
}
func (ctx *__plus) inner() Context { return &ctx.builtinbase }
func (ctx *__plus) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__plus) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    if ctx.int {
        var num int64
        for n, a := range ctx.a {
            var i = a.int(ctx)
            if n == 0 { num = i } else { num += i }
        }
        return _decimal(_position(ctx), num)
    } else {
        var num float64
        for n, a := range ctx.a {
            var f = a.float(ctx)
            if n == 0 { num = f } else { num += f }
        }
        return _float(_position(ctx), num)
    }
}

type __minus struct { builtinbase
    int bool `int,integer`
}
func (ctx *__minus) inner() Context { return &ctx.builtinbase }
func (ctx *__minus) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__minus) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    if ctx.int {
        var num int64
        for n, a := range ctx.a {
            var i = a.int(ctx)
            if n == 0 { num = i } else { num -= i }
        }
        return _decimal(_position(ctx), num)
    } else {
        var num float64
        for n, a := range ctx.a {
            var f = a.float(ctx)
            if n == 0 { num = f } else { num -= f }
        }
        return _float(_position(ctx), num)
    }
}

type __multiply struct { builtinbase
    int bool `int,integer`
}
func (ctx *__multiply) inner() Context { return &ctx.builtinbase }
func (ctx *__multiply) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__multiply) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    if ctx.int {
        var num int64
        for n, a := range ctx.a {
            var i = a.int(ctx)
            if n == 0 { num = i } else { num *= i }
        }
        return num
    } else {
        var num float64
        for n, a := range ctx.a {
            var f = a.float(ctx)
            if n == 0 { num = f } else { num *= f }
        }
        return num
    }
}

type __divide  struct { builtinbase
    int bool `int,integer`
}
func (ctx *__divide) inner() Context { return &ctx.builtinbase }
func (ctx *__divide) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__divide) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    if ctx.int {
        var num int64
        for n, a := range ctx.a {
            var i = a.int(ctx)
            if n == 0 { num = i } else { num /= i } // FIXME: NaN
        }
        return num
    } else {
        var num float64
        for n, a := range ctx.a {
            var f = a.float(ctx)
            if n == 0 { num = f } else { num /= f } // FIXME: NaN
        }
        return num
    }
}

type __unique struct { builtinbase
    reverse  bool `reverse`
    keepAuto bool `auto,keepauto,keep-auto`
    unexpand bool `unexpand,noexpand,no-expand`
}
func (ctx *__unique) inner() Context { return &ctx.builtinbase }
func (ctx *__unique) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__unique) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var args = ctx.a
    var t1, t2 time.Time

    if false { defer func() {
        t3 := time.Now()
        d0 := t3.Sub(t1)
        d1 := t2.Sub(t1)
        d2 := t3.Sub(t2)
        if d0 > 1*time.Second {
            for _, a := range args { a.string(ctx) }
            t4 := time.Now()
            d3 := t4.Sub(t3)
            for i, a := range args { if i > 0 { a.cmp(ctx, args[i-1]) } }
            t5 := time.Now()
            d4 := t5.Sub(t4)
            // for i, a := range args { if i > 0 { eq(ctx, a, args[i-1]) } }
            for i, a := range args { if i > 0 { equal(ctx, a, args[i-1]) } }
            t6 := time.Now()
            d5 := t6.Sub(t5)
            var args2 []Value
            var seen = make(map[uint64]struct{})
            for _, a := range args {
                c := a.hash(ctx)
                if _, y := seen[c]; y {
                    note(ctx, "%v")
                } else {
                    seen[c] = struct{}{}
                }
                var t = true
                for _, b := range args2 {
                    if equal(ctx, a, b) { t = false ; break }
                }
                if t { args2 = append(args2, a) }
            }
            t7 := time.Now()
            d6 := t7.Sub(t6)
            note(ctx, "%v %v %v (%v, %v, %v, %v, %d %d)", d0, d1, d2, d3, d4, d5, d6, len(args), len(args2))//.debug(2)
            t7 = time.Now()
            unique(ctx, args...)
            d6 = t7.Sub(t6)
            erro(ctx, "unique: %v", d6).trace()
        }
    } () }

    t1 = time.Now()

    if ctx.unexpand {
        args =  merge(args...)
    } else {
        args = xmerge(ctx, args...)
    }

    t2 = time.Now()

    if ctx.reverse {
        return reverse_unique(ctx, args...)
    } else {
        return         unique(ctx, args...)
    }
}

type __join struct { builtinbase }
func (ctx *__join) inner() Context { return &ctx.builtinbase }
func (ctx *__join) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__join) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if l := len(ctx.a); 0 < l {
        var t = &compound{}
        if l < 2 {
            t.app(merge(ctx.a...)...)
        } else {
            var sep = scalarize(ctx.a[l-1])
            for i, v := range merge(ctx.a[:l-1]...) {
                if 0 < i {
                    t.app(sep, v)
                } else {
                    t.app(v)
                }
            }
        }
        return t
    }
    return
}

type __compose struct { builtinbase }
func (ctx *__compose) inner() Context { return &ctx.builtinbase }
func (ctx *__compose) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__compose) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if l := len(ctx.a); 0 < l {
        var con conjunction
        if l < 2 {
            con.list = _list(merge(ctx.a...)...)
        } else {
            con.list = _list(merge(ctx.a[:l-1]...)...)
            con.sep  = ctx.a[l-1]
        }
        return con
    }
    return
}

type __quote struct { builtinbase }
func (ctx *__quote) inner() Context { return &ctx.builtinbase }
func (ctx *__quote) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__quote) x() any {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    return &quoted{list{elements{ctx.a}}}
}

type __quotejoin struct { builtinbase }
func (ctx *__quotejoin) inner() Context { return &ctx.builtinbase }
func (ctx *__quotejoin) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__quotejoin) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var sep string
    var args = merge(ctx.a...)
    if l := len(args); l > 1 {
        sep = args[l-1].string(ctx)
        args = args[:l-1]
    }
    if l := len(args); l > 0 {
        var fields []string
        for _, a := range args[1:] {
            if v := a.string(ctx); v != "" { fields = append(fields, v) }
        }
        res = _strlit(_position(ctx), strconv.Quote(strings.Join(fields, sep)))
    } else {
        res = _none(_position(ctx))
    }
    return
}

// $(split-string .,1.2.3)
type __splitstring struct { builtinbase
    sep string `sep,separator`
}
func (ctx *__splitstring) inner() Context { return &ctx.builtinbase }
func (ctx *__splitstring) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__splitstring) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if 0 < len(ctx.a) {
        var fields []Value
        var sep = ctx.sep
        if sep == "" { sep = ctx.a[0].string(ctx) }
        for _, a := range ctx.a[1:] {
            for _, s := range strings.Split(a.string(ctx), sep) {
                fields = append(fields, _strlit(a.Position(), s))
            }
        }
        return fields
    }
    return
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
        res = _strlit(value.Position(), strings.Join(strs, sep))
    }
    return
}

// TODO: deprecate this and add -quote to __splitstring
type __splitquote struct { __splitstring }
func (ctx *__splitquote) inner() Context { return &ctx.builtinbase }
func (ctx *__splitquote) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__splitquote) x() (res any) {
    res = ctx.__splitstring.x()
    if v, y := res.(Value); y && v != nil { quotestrings(v) }
    return
}

// TODO: deprecate this and add -quote to __splitstring
type __splitquotejoin struct { __splitstring }
func (ctx *__splitquotejoin) inner() Context { return &ctx.builtinbase }
func (ctx *__splitquotejoin) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__splitquotejoin) x() (res any) {
    res = ctx.__splitstring.x()
    if val, y := res.(Value); y && val != nil {
        var err error
        var sep string
        if l := len(ctx.a); l > 1 {
            sep = ctx.a[l-1].string(ctx)
            ctx.a = ctx.a[:l-1]
        }
        if res, err = joinstrings(ctx, val, sep); err != nil {
            erro(ctx, "%v", err).trace()
        }
    }
    return
}

type __splitjoinquote struct { __splitstring }
func (ctx *__splitjoinquote) inner() Context { return &ctx.builtinbase }
func (ctx *__splitjoinquote) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__splitjoinquote) x() (res any) {
    res = ctx.__splitstring.x()
    if val, y := res.(Value); y && val != nil {
        var err error
        var sep string
        if l := len(ctx.a); l > 1 {
            sep = ctx.a[l-1].string(ctx)
            ctx.a = ctx.a[:l-1]
        }

        var v Value
        if v, err = joinstrings(ctx, val, sep); err != nil {
            erro(ctx, "%v", err).trace()
        } else {
            res = _strlit(_position(ctx), strconv.Quote(v.string(ctx)))
        }
    }
    return
}

type __field struct { builtinbase }
func (ctx *__field) inner() Context { return &ctx.builtinbase }
func (ctx *__field) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__field) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if l := len(ctx.a); l >= 2 {
        var fields []string
        var s string = ctx.a[1].string(ctx)
        var i int64 = ctx.a[0].int(ctx)
        if l > 2 {
            fields = strings.Split(s, ctx.a[2].string(ctx))
        } else {
            fields = strings.Fields(s)
        }
        if n := int(i)-1; 0 <= n && n < len(fields) {
            s = strings.TrimSpace(fields[n])
        }
        return fields
    }
    return
}

type __fields struct { builtinbase }
func (ctx *__fields) inner() Context { return &ctx.builtinbase }
func (ctx *__fields) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__fields) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    // TODO: ...
    return
}

type __usee struct { builtinbase }
func (ctx *__usee) inner() Context { return &ctx.builtinbase }
func (ctx *__usee) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__usee) x() (res any) {
    var proj = _project(ctx)
    if proj == nil {
        erro(ctx, "unknown current context").trace()
    }

    var vals []Value
    for _, a := range ctx.a {
        v := proj.use.sel(ctx, a.string(ctx))
        if v != nil { vals = append(vals, v.(Value)) }
    }
    if vals == nil { res = vals }
    return
}

type __uses struct { builtinbase }
func (ctx *__uses) inner() Context { return &ctx.builtinbase }
func (ctx *__uses) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__uses) x() (res any) {
    var proj = _project(ctx)
    if proj == nil {
        erro(ctx, "unknown current context").trace()
    }

    var found bool

outer:
    for _, a := range ctx.a {
        var s = a.string(ctx)
        for _, u := range proj.use.list {
            found = u.project.name == s
            if found { break outer }
        }
    }

    if found { res = found }
    return
}

type __path struct { builtinbase }
func (ctx *__path) inner() Context { return &ctx.builtinbase }
func (ctx *__path) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__path) x() any {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    var res []Value
    for _, a := range ctx.a {
        if x, y := a.(*path); y {
            res = append(res, x)
        } else {
            res = append(res, _pathstr(ctx, a.string(ctx)))
        }
    }
    return res
}

type __bare struct { builtinbase
    name bool `name,filename,file-name,non-full,not-full`
}
func (ctx *__bare) inner() Context { return &ctx.builtinbase }
func (ctx *__bare) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__bare) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var vals []Value
    for _, a := range ctx.a {
        switch p := a.Position(); t := a.(type) {
        case *strlit, *strcomp:
            a = _word(p, a.string(ctx))
        case *file:
            a = _word(p, t.ident(ctx))
        case fullfile:
            if ctx.name {
                a = _word(p, t.ident(ctx))
            } else {
                a = _word(p, t.string(ctx))
            }
        }
        vals = append(vals, a)
    }
    return vals
}

type __word struct { builtinbase }
func (ctx *__word) inner() Context { return &ctx.builtinbase }
func (ctx *__word) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__word) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    var vals []Value
    for _, a := range ctx.a {
        if _, y := a.(*word); !y {
            a = _word(a.Position(), a.string(ctx))
        }
        vals = append(vals, a)
    }
    return vals
}

type __resolve struct { builtinbase
    closure bool `closure`
    // expand bool `expand`
}
func (ctx *__resolve) inner() Context { return &ctx.builtinbase }
func (ctx *__resolve) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__resolve) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if 0 < len(ctx.a) {
        var resolve func(Context, string) object
        if ctx.closure {
            resolve = closure_resolve
        } else {
            resolve = project_resolve
        }

        var vals []Value
        for _, a := range merge(ctx.a...) {
            var name = a.string(ctx)
            if o := resolve(ctx, name); o == nil {
                erro(ctx, "%v is nil : %v", a, ts(a)).trace()
            } else if x, y := o.(*def); !y {
                erro(ctx, "%v is not def : %v : %v", a, o, ts(o)).trace()
            } else if x.value != nil {
                vals = append(vals, merge(x.value)...)
            }
        }
        return vals
    }
    return
}

type __string struct { builtinbase
    expand bool `expand`
    name   bool `name,file-name,non-full`
    con  bool `conjunct,conjunction`
    dis  bool `disjunct,disjunction`
    closure  []string `closure`
    def  []string `def,var`
    join []string `join`
}
func (ctx *__string) inner() Context { return &ctx.builtinbase }
func (ctx *__string) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__string) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if 0 < len(ctx.a)+len(ctx.closure)+len(ctx.def) {
        var vals []Value
        for _, name := range ctx.closure {
            if o := closure_resolve(ctx, name); o != nil {
                if d, y := o.(*def); y && d != nil && d.value != nil {
                    vals = append(vals, d.value)
                }
            }
        }
        for _, name := range ctx.def {
            if o := project_resolve(ctx, name); o != nil {
                if d, y := o.(*def); y && d != nil && d.value != nil {
                    vals = append(vals, d.value)
                }
            }
        }

        vals = merge(append(vals, ctx.a...)...)

        if /* expandable(_final(ctx), vals...) */true {
            return &strval{valbase{_position(ctx)},vals}
        } else if 0 < len(ctx.join) {
            var s bytes.Buffer
            for i, v := range vals {
                if t := v.string(ctx); t != "" {
                    if 0 < i { s.WriteString(ctx.join[i % len(ctx.join)]) }
                    s.WriteString(t)
                }
            }
            return &strlit{valbase{_position(ctx)},s.String()}
        } else if ctx.con || !ctx.dis { // conjunction (default)
            var s bytes.Buffer
            for i, v := range vals {
                if t := v.string(ctx); t != "" {
                    if 0 < i { s.WriteString(" ") }
                    s.WriteString(t)
                }
            }
            return &strlit{valbase{_position(ctx)},s.String()}
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

type __finalize struct { builtinbase }
func (ctx *__finalize) inner() Context { return &ctx.builtinbase }
func (ctx *__finalize) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__finalize) x() any {
    ctx.a = expand(_final(ctx), ctx.a...)
    return ctx.a
}

type __filter struct { builtinbase
    stem bool `stem`
    neg bool
}
func (ctx *__filter) inner() Context { return &ctx.builtinbase }
func (ctx *__filter) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__filter) _x(pats []Value, values... Value) (result []Value) {
    defer func(t0 time.Time) {
        if d := time.Now().Sub(t0); d > 1*time.Second {
            pos := _position(ctx)
            prompt(ctx, "%v: slow: %d result, %v\n", pos, len(result), result)
            prompt(ctx, "%v: slow: %d pats, %v\n", pos, len(pats), pats)
            prompt(ctx, "%v: slow: %v\n", pos, d).debug(4)
        }
    } (time.Now())

    var f = func(v Value) Value {
        for _, pat := range pats {
            if full, res, stems := pat.match(ctx, v); full {
                if ctx.neg {
                    v = nil
                } else if ctx.stem {
                    v = ease(ctx, stems)
                } else if false {
                    if t, r := pat.stencil(ctx, stems); t != nil && len(r) == 0 {
                        v = t
                    } else {
                        v = ease(ctx, res)
                    }
                }
                return v
            }
        }
        if ctx.neg { return v } else { return nil }
    }

    for _, v := range values {
        if t := f(v); t != nil {
            result = append(result, t)
        }
    }
    return
}
func (ctx *__filter) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if len(ctx.a) > 1 {
        var i int
        var vals []Value
        var pats = merge(ctx.a[0])
        if len(pats) > 0 {
            i = 1 // good
        } else if pats = merge(ctx.a[1]); len(pats) == 0 {
            erro(ctx, "no patterns: %v", ctx.a).trace()
        } else {
            i = 2
        }

        if len(ctx.a) < i {
            erro(ctx, "out of index: %d > %d, %v", i, len(ctx.a), ctx.a).trace()
        }

        vals = merge(ctx.a[i:]...)
        vals = ctx._x(pats, vals...)
        if len(vals) > 0 { res = vals }
    }
    return
}

// $(filter-out pattern…,text)
type __filterout struct { __filter }
func (ctx *__filterout) inner() Context { return &ctx.builtinbase }
func (ctx *__filterout) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__filterout) x() any {
    ctx.neg = true ; return ctx.__filter.x()
}

type __substring struct { builtinbase }
func (ctx *__substring) inner() Context { return &ctx.builtinbase }
func (ctx *__substring) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__substring) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    var res []Value
    if n := len(ctx.a); n > 1 {
        var v1, v2 = ctx.a[0], ctx.a[1]
        var a, b = intVal(ctx, v1, -1), intVal(ctx, v2, -1)
        if ctx.a = ctx.a[2:]; a < -1 && b < -1 {
            erro(ctx, "wrong indices (%v, %v)", v1, v2).trace()
        }
        if a > b { t := a; a = b; b = t } // swap the wrong order
        if a == -1 { a = b }
        if a == -1 { return }

        for _, arg := range ctx.a {
            var s = arg.string(ctx)
            if i := len(s); i <= a { s = "" } else
            if b == -1 || i <= b { s = s[a:b] } else { s = s[a:] }
            res = append(res, _strlit(arg.Position(), s))
        }
    }
    return res
}

// $(subst from,to,text)
type __subst struct { builtinbase }
func (ctx *__subst) inner() Context { return &ctx.builtinbase }
func (ctx *__subst) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__subst) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    var res []Value
    if len(ctx.a) > 2 {
        var s1 = ctx.a[0].string(ctx)
        var s2 = ctx.a[1].string(ctx)
        for _, arg := range merge(ctx.a[2:]...) {
            s := strings.Replace(arg.string(ctx), s1, s2, -1)
            res = append(res, _strlit(arg.Position(), s))
        }
    }
    return res
}

// $(patsubst pattern,replacement,text)
// TODO: supports: $(var:pattern=replacement)
// TODO: supports: $(var:suffix=replacement)
// TODO: support flags -name and -full for name-only and full-name-only matching
type __patsubst struct { builtinbase
    findFiles bool `find,find-file`
    fullFiles bool `ff,fullfile,fullfiles`
    cleanPath bool `c,clean,cleanpath`
    nofilemap bool `nomap,no-map,nofile,nofiles,no-files,no-filemap`
    erroDstNomap bool `err-dst-nomap,error-dst-nomap`
    warnDstNomap bool `warn-dst-nomap`
}
func (ctx *__patsubst) inner() Context { return &ctx.builtinbase }
func (ctx *__patsubst) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__patsubst) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if len(ctx.a) < 3 {
        erro(ctx, "not enough arguments").trace()
    }

    var (
        proj = _project(ctx)
        closured = closure_projects(ctx)
        srcPats = merge(ctx.a[0])
        dstPats, sources, res []Value
        t1 time.Time
    )
    defer func(t0 time.Time) {
        var t2 = time.Now()
        if d := t2.Sub(t0); d > 1*time.Second {
            var d1 = t1.Sub(t0)
            var d2 = t2.Sub(t1)
            var pos = _position(ctx)
            prompt(ctx, "%v: slow: src %d %v\n", pos, len(srcPats), srcPats)
            prompt(ctx, "%v: slow: dst %d %v\n", pos, len(dstPats), dstPats)
            prompt(ctx, "%v: slow: sources %d %v\n", pos, len(sources), sources)
            prompt(ctx, "%v: slow: list %d %v\n", pos, len(res), res)
            prompt(ctx, "%v: slow: %v⇒%v+%v\n", pos, d, d1, d2).debug(4)
        }
    } (time.Now())

    if len(srcPats) == 0 {
        if len(ctx.a) < 4 {
            erro(ctx, "not enough arguments").trace()
        }
        srcPats = merge(ctx.a[1])
        dstPats = merge(ctx.a[2])
        sources = merge(ctx.a[3:]...)
    } else {
        dstPats = merge(ctx.a[1])
        sources = merge(ctx.a[2:]...)
    }

    t1 = time.Now()

ForSources:
    for _, src := range sources {
        var (
            source any = src
            srcFile *file
            srcPat Value
            stems []string
            ok bool
        )
        if srcFile, ok = to_file(src); ok {
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
        } else if o := (as{src}.fullname(ctx, closured...)); o.Value != nil {
            source = o.string(ctx)
        } else {
            erro(ctx, "fullname '%v' failed", src)
            erro(ctx, "called from here", src).trace()
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
                erro(ctx, "nil stencil: %T %v (stems=%v, ramnant=%v)", dstPat, dstPat, stems, ramnant).trace()
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
                var dstFile *file
                if !ctx.nofilemap { dstFile = proj.file(ctx, nameStr) }
                if dstFile == nil {
                    a := []any{
                        "%v: %v (%v): unmapped destination, aka files (...)",
                        proj, nameStr, dstPat,
                    }
                    if t := unmap_files(ctx, nameVal); ctx.erroDstNomap {
                        erro(ctx, "%v: patsubst: %v (%v) ⇒ %v (%v) ⇒ %v", proj, srcFile, srcPat, nameVal, dstPat, t)
                        errostack(ctx, 16, a...).trace()
                    } else if ctx.warnDstNomap {
                        warn(ctx, "%v: patsubst: %v (%v) ⇒ %v (%v) ⇒ %v", proj, srcFile, srcPat, nameVal, dstPat, t)
                        warnstack(ctx, 16, a...).debug(5)
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
            case *file, fullfile:
            case *strlit, *strcomp:
                res = append(res, _strlit(pos, nameStr))
                continue stencilTargetPats
            case *path:
                res = append(res, _pathstr(ctx, nameStr))
                continue stencilTargetPats
            case *word, *compound:
                if strings.Contains(nameStr, pathSep) {
                    res = append(res, _pathstr(ctx, nameStr))
                } else {
                    res = append(res, _word(pos, nameStr))
                }
                continue stencilTargetPats
            default:
                if strings.Contains(nameStr, pathSep) {
                    res = append(res, _pathstr(ctx, nameStr))
                } else if true {
                    res = append(res, _word(pos, nameStr))
                } else {
                    res = append(res, _strlit(pos, nameStr))
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
        warnstack(ctx, 3, "%v", ts(ctx)).debug(ctx.debug)
    }
    return res
}

type __title struct { builtinbase }
func (ctx *__title) inner() Context { return &ctx.builtinbase }
func (ctx *__title) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__title) x() any {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    var res []Value
    for _, a := range ctx.a {
        switch t := a.(type) {
        case interface{ change(func(string) string) Value }:
            a = t.change(strings.Title)
        default:
            a = _raw(a.Position(), strings.Title(a.string(ctx)))
        }
    }
    return res
}

type __uppercase struct { builtinbase }
func (ctx *__uppercase) inner() Context { return &ctx.builtinbase }
func (ctx *__uppercase) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__uppercase) x() any {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    var res []Value
    for _, a := range ctx.a {
        switch t := a.(type) {
        case interface{ change(func(string) string) Value }:
            a = t.change(strings.ToUpper)
        default:
            a = _raw(a.Position(), strings.ToUpper(a.string(ctx)))
        }
        res = append(res, a)
    }
    return res
}

type __lowercase struct { builtinbase }
func (ctx *__lowercase) inner() Context { return &ctx.builtinbase }
func (ctx *__lowercase) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__lowercase) x() any {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    var res []Value
    for _, a := range ctx.a {
        switch t := a.(type) {
        case interface{ change(func(string) string) Value }:
            a = t.change(strings.ToLower)
        default:
            a = _raw(a.Position(), strings.ToLower(a.string(ctx)))
        }
        res = append(res, a)
    }
    return res
}

func (ctx *builtinbase) trim(f func(_, _ Value, _ string) (Value, string), ss ...string) (_ any) {
    if len(ctx.a) < 1 {
        var s = strings.Join(ss, "")
        erro(ctx, "try $(trim%s <...%s>, <...value>)", s, s).trace()
    }

    var prefix = merge(ctx.a[0])
    var values = merge(ctx.a[1:]...)

    if len(values) == 0 { return }
    if false && len(prefix) == 0 { return ease(ctx, values) }

    var res []Value
    for _, val := range values {
        var v Value
        var s string
        for _, pre := range prefix {
            if v, s = f(val, pre, s); v != nil { break }
        }
        if v != nil { res = append(res, v) }
    }
    return res
}

type __strip struct { __trimspace }

type __trim struct { builtinbase }
func (ctx *__trim) inner() Context { return &ctx.builtinbase }
func (ctx *__trim) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trim) x() any {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var res []Value
    var cutset string
    var f func(string) string
    if cutset == "" {
        f = strings.TrimSpace
    } else {
        f = func(s string) string { return strings.Trim(s, cutset) }
    }
    for _, a := range merge(ctx.a...) {
        switch t := a.(type) {
        case interface{ change(func(string) string) Value }:
            a = t.change(f)
        default:
            a = _raw(a.Position(), f(a.string(ctx)))
        }
        res = append(res, a)
    }
    return res
}

type __trimspace struct { __trim }
func (ctx *__trimspace) inner() Context { return &ctx.builtinbase }
func (ctx *__trimspace) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}

type __trimleft struct { builtinbase }
func (ctx *__trimleft) inner() Context { return &ctx.builtinbase }
func (ctx *__trimleft) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trimleft) x() any {
    var res []Value
    var cutset string
    var f func(string) string
    if cutset == "" {
        f = func(s string) string { return strings.TrimLeftFunc(s, unicode.IsSpace) }
    } else {
        f = func(s string) string { return strings.TrimLeft(s, cutset) }
    }
    for _, a := range ctx.a {
        switch t := a.(type) {
        case interface{ change(func(string) string) Value }:
            a = t.change(f)
        default:
            a = _raw(a.Position(), f(a.string(ctx)))
        }
        res = append(res, a)
    }
    return res
}

type __trimright struct { builtinbase }
func (ctx *__trimright) inner() Context { return &ctx.builtinbase }
func (ctx *__trimright) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trimright) x() any {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var res []Value
    var cutset string
    var f func(string) string
    if cutset == "" {
        f = func(s string) string { return strings.TrimRightFunc(s, unicode.IsSpace) }
    } else {
        f = func(s string) string { return strings.TrimRight(s, cutset) }
    }
    for _, a := range ctx.a {
        switch t := a.(type) {
        case interface{ change(func(string) string) Value }:
            a = t.change(f)
        default:
            a = _raw(a.Position(), f(a.string(ctx)))
        }
        res = append(res, a)
    }
    return res
}

// $(trim-prefix foo%, fooxxx foo123)
// $(trim-prefix %/foo, xxx/foo/a/b/c)
// $(trim-prefix %%/foo, xxx/yyy/zzz/foo/a/b/c)
type __trimprefix struct { builtinbase }
func (ctx *__trimprefix) inner() Context { return &ctx.builtinbase }
func (ctx *__trimprefix) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trimprefix) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    return ctx.trim(func(val, prefix Value, _s string) (res Value, s string) {
        var t string
        if prefix.patterned(ctx) {
            var f, r, m = prefix.match(ctx, val)
            if checkpoints && truly(ctx, is_test_mode{}) { ctx.check_match(val, prefix, f, r, m) }
            if f { return } // trim all for prefix
            t = _joinpath(ctx, r)
        } else {
            t = prefix.string(ctx)
        }

        if s = _s ; s == "" { s = val.string(ctx) }

        if strings.HasPrefix(s, t) {
            _s = strings.TrimPrefix(s, t)
        } else if true {
            _s = strings.TrimLeftFunc(s, unicode.IsSpace)
        } else {
            _s = s
        }

        switch val.(type) {
        case *path: res = _pathstr(ctx, _s)
        default: res = _strlit(val.Position(), _s)
        }

        if checkpoints && truly(ctx, is_test_mode{}) { ctx.check(prefix, val, res) }
        return
    })
}

type __trimsuffix struct { builtinbase }
func (ctx *__trimsuffix) inner() Context { return &ctx.builtinbase }
func (ctx *__trimsuffix) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trimsuffix) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    return ctx.trim(func(val, suffix Value, _s string) (res Value, s string) {
        var t string
        if suffix.patterned(ctx) {
            var f, r, _s = suffix.match(reversal{ctx}, val)
            if checkpoints {
                if suffix.String() == "/testdata/**" {
                    if f != false {
                        erro(ctx, "%v : %v %v %v", tv(suffix), f, r, _s).trace()
                    }
                    if x, y := r.([]string); !y || joinpath(x...) != "/testdata/builtins/trimsuffix" {
                        erro(ctx, "%v : %v %v %v", tv(suffix), f, r, _s).trace()
                    }
                    if len(_s) != 1 {
                        erro(ctx, "%v : %v %v %v", tv(suffix), f, r, _s).trace()
                    } else if _s[0] != "builtins/trimsuffix" {
                        erro(ctx, "%v : %v %v %v", tv(suffix), f, r, _s).trace()
                    }
                }
            }
            if f { return }

            t = _joinpath(ctx, r)
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
        default: res = _strlit(val.Position(), _s)
        }
        return
    })
}

type __trimext struct { __trim
    all bool `all`
    ext []string `ext`
}
func (ctx *__trimext) inner() Context { return &ctx.builtinbase }
func (ctx *__trimext) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trimext) x() any {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var ext string
    var res []Value
    for i, a := range ctx.a {
        if s := a.string(ctx); s != "" {
            if i == 0 && len(ctx.a) > 1 {
                ext = s
            } else if ext == "" {
                for ext = filepath.Ext(s); ext != ""; {
                    s = strings.TrimSuffix(s, ext)
                    if ctx.all { ext = filepath.Ext(s) } else { break }
                }
                res = append(res, _strlit(a.Position(), s))
            } else if ext == filepath.Ext(s) {
                res = append(res, _strlit(a.Position(), strings.TrimRight(s, ext)))
            }
        }
    }
    return res
}

type __gitdir struct { builtinbase }
func (ctx *__gitdir) inner() Context { return &ctx.builtinbase }
func (ctx *__gitdir) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__gitdir) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var vals []Value
    for _, a := range merge(ctx.a...) {
        var s = a.string(ctx)
        if !strings.HasSuffix(s, "/.git") {
            s = filepath.Join(s, ".git")
        }
        if i, e := os.Stat(s); e != nil {
            a = _pathstr(ctx, s)
        } else if m := i.Mode(); m.IsDir() {
            a = _pathstr(ctx, s)
        } else if m.IsRegular() {
            if b, e := ioutil.ReadFile(s); e != nil {
                errostack(ctx, 5, "%v", e).trace()
            } else if !bytes.HasPrefix(b, []byte("gitdir:")) {
                errostack(ctx, 5, "%s", b).trace()
            } else {
                t := string(bytes.TrimSpace(b[7:]))
                s = filepath.Join(filepath.Dir(s), t)
                a = _pathstr(ctx, s)
            }
        } else {
            erro(pc(ctx,a), "%v", s).trace()
        }
        vals = append(vals, a)
    }
    return vals
}

type __add___fix struct { builtinbase; dis Value }
func (ctx *__add___fix) do(c Context, op any) (_ any) {
    switch t := op.(type) {
    case dis: ctx.dis = t.Value
    }
    return ctx.builtinbase.do(c, op)
}
func (ctx *__add___fix) x(f func(_, _ Value) Value) (_ any) {
    if len(ctx.a) < 1 {
        erro(ctx, "not enough args, try $(addprefix prefix, ...)").trace()
    }
    if !ctx.originalArgs {
        ctx.a[0] = ctx.a[0].expand(ctx)
    }

    var res []Value
    for _, fix := range merge(ctx.a[0]) {
        fix = redis(fix)
        for i, val := range ctx.a[1:] {
            var _v = val.expand(ctx)
            if !ctx.originalArgs { ctx.a[1+i] = _v }
            if fix != nil && _v != nil {
                for _, v := range merge(_v) {
                    res = append(res, f(fix, redis(v)))
                }
            }
        }
    }
    return res
}

type __addprefix struct { __add___fix }
func (ctx *__addprefix) inner() Context { return &ctx.builtinbase }
func (ctx *__addprefix) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__addprefix) x() (_ any) {
    return ctx.__add___fix.x(func(x, y Value) Value {
        return prefix(ctx, x, y)
    })
}

type __addsuffix struct { __add___fix }
func (ctx *__addsuffix) inner() Context { return &ctx.builtinbase }
func (ctx *__addsuffix) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__addsuffix) x() (_ any) {
    return ctx.__add___fix.x(func(x, y Value) Value {
        return suffix(ctx, x, y)
    })
}

type __print struct{ builtinbase
    noErrs bool `noerrs,noerrors,no-errs,no-errors`
    noWarn bool `nowarn,nowarns,no-warn,no-warns`
    f string `...`
}
func (ctx *__print) inner() Context { return &ctx.builtinbase }
func (ctx *__print) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__print) x() (_ any) {
    if ctx.noErrs && 0 < count_diag(ctx, diagError) { return }
    if ctx.noWarn && 0 < count_diag(ctx, diagWarn)  { return }
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var sb bytes.Buffer
    var x = len(ctx.a)
    for i, a := range ctx.a {
        if a == nil { continue }
        if 0 < i && i < x { fmt.Fprintf(&sb, " ") }
        fmt.Fprintf(&sb, "%s", escapedString(ctx, a))
    }
    prompt(ctx, sb.String())
    return
}

type __printf struct{ builtinbase }
func (ctx *__printf) inner() Context { return &ctx.builtinbase }
func (ctx *__printf) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__printf) x() (_ any) {
    if len(ctx.a) < 1 {
        erro(ctx, "not enough args, try $(printf 'format', ...)").trace()
    }
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var vals = merge(ctx.a[0])
    if len(vals) != 1 {
        erro(ctx, "not enough args, try $(printf 'format', ...)").trace()
    }

    var i int
    var a []any
    var f = vals[0].string(ctx)

outer:
    for _, v := range merge(ctx.a[1:]...) {
    fmtloop:
        for i < len(f) {
            if f[i] != '%' { i += 1; continue }
            for i += 1; i < len(f); i += 1 {
                switch f[i] {
                case '%': continue fmtloop
                case '+', '-', '#', ' ', '.', '0', '1', '2', '3',
                    '4', '5', '6', '7', '8', '9': continue
                case 'c', 'd', 'o', 'O', 'q', 'U':
                    a = append(a, v.int(ctx))
                    continue outer
                case 'e', 'E', 'f', 'F', 'g', 'G':
                    a = append(a, v.float(ctx))
                    continue outer
                case 'b', 'x', 'X':
                    switch k := v.kind(); {
                    case k&KindInteger != 0:
                        a = append(a, v.int(ctx))
                        continue outer
                    case k&KindFloat != 0:
                        a = append(a, v.float(ctx))
                        continue outer
                    default:
                        if t, e := strconv.Atoi(v.string(ctx)) ; e == nil { a = append(a, t) } else {
                            erro(ctx, "%v: %v", v, e).trace()
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

type __indent struct { builtinbase }
func (ctx *__indent) inner() Context { return &ctx.builtinbase }
func (ctx *__indent) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__indent) x() (res any) {
    var l []Value
    var s string // indent
    if x := len(ctx.a); x > 0 {
        if v, ok := scalarize(ctx.a[0]).(*decimal); ok {
            ctx.a, s = ctx.a[1:], strings.Repeat(" ", int(v.int64))
        } else {
            erro(ctx, "requires integer argument (first|last)").trace()
        }
    }
    for _, a := range ctx.a {
        var lines []string
        for _, line := range strings.Split(a.string(ctx), "\n") {
            lines = append(lines, s + line)
        }
        l = append(l, _strlit(a.Position(), strings.Join(lines, "\n")))
    }
    return l
}

type __findstring struct { builtinbase }
func (ctx *__findstring) inner() Context { return &ctx.builtinbase }
func (ctx *__findstring) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__findstring) x() (res any) {
    // TODO: $(findstring find,text)
    return
}

// $(contains a b c, v1 v2 …)
// $(contains a b c1 -or c2, v1 v2 …)          -- xx
// $(contains a b c1 -or c2 -or c3, v1 v2 …)   -- xx
// $(contains a b -or=(c1 c2 c3), v1 v2 …)     -- xx
type __contains struct { builtinbase
    match  bool `match,pat,pattern`
    string bool `str,string`
}
func (ctx *__contains) inner() Context { return &ctx.builtinbase }
func (ctx *__contains) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__contains) x() (_ any) {
    if len(ctx.a) < 2 {
        erro(ctx, "unexpected number of arguments, try $(contains a b c1 c2, v1 v2 …)").trace()
    }
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var n int
    var vals = merge(ctx.a[0])
    var list = merge(ctx.a[1:]...)
    if len(vals) == 0 || len(list) == 0 {
        erro(ctx, "insufficient number of arguments: %v ⇒ %v %v", ctx.a, vals, list).trace()
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
                t = elem.string(ctx) == s
            } else {
                t = val.cmp(ctx, elem) == cmpEqual
            }
            if t { n += 1; continue outer }
        }
    }

    if n == len(vals) {
        return _boolean(_position(ctx), true)
    } else {
        return
    }
}

type __sort struct { builtinbase }
func (ctx *__sort) inner() Context { return &ctx.builtinbase }
func (ctx *__sort) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__sort) x() (res any) {
    erro(ctx, "TODO: $(sort ...)").trace()
    return
}

type __wordlist struct { builtinbase }
func (ctx *__wordlist) inner() Context { return &ctx.builtinbase }
func (ctx *__wordlist) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__wordlist) x() (res any) {
    erro(ctx, "TODO: $(wordlist ...)").trace()
    return
}

type __words struct { builtinbase }
func (ctx *__words) inner() Context { return &ctx.builtinbase }
func (ctx *__words) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__words) x() (res any) {
    erro(ctx, "TODO: $(words ...)").trace()
    return
}

type __firstword struct { builtinbase }
func (ctx *__firstword) inner() Context { return &ctx.builtinbase }
func (ctx *__firstword) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__firstword) x() (res any) {
    erro(ctx, "TODO: $(firstword ...)").trace()
    return
}

type __lastword struct { builtinbase }
func (ctx *__lastword) inner() Context { return &ctx.builtinbase }
func (ctx *__lastword) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__lastword) x() (res any) {
    erro(ctx, "TODO: $(lastword ...)").trace()
    return
}

type __encodebase64 struct { builtinbase }
func (ctx *__encodebase64) inner() Context { return &ctx.builtinbase }
func (ctx *__encodebase64) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__encodebase64) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if 0 < len(ctx.a) {
        pos := _position(ctx)
        buf := new(bytes.Buffer)
        enc := base64.NewEncoder(base64.StdEncoding, buf)
        for _, a := range ctx.a { enc.Write([]byte(a.string(ctx))) }
        enc.Close()
        res = _strlit(pos, buf.String())
    }
    return
}

type __decodebase64 struct { builtinbase }
func (ctx *__decodebase64) inner() Context { return &ctx.builtinbase }
func (ctx *__decodebase64) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__decodebase64) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if 0 < len(ctx.a) {
        var res []Value
        for _, a := range ctx.a {
            var s = a.string(ctx)
            if dat, err := base64.StdEncoding.DecodeString(s); err != nil {
                erro(ctx, "decode '%s' failed: %v", s, err).trace()
            } else {
                res = append(res, _strlit(a.Position(), string(dat)))
            }
        }
        return ease(ctx, res)
    }
    return
}

type __ext struct { builtinbase }
func (ctx *__ext) inner() Context { return &ctx.builtinbase }
func (ctx *__ext) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__ext) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    var res []Value
    for _, a := range merge(ctx.a...) {
        res = append(res, _strlit(a.Position(), filepath.Ext(a.string(ctx))))
    }
    if 0 < len(res) { return res }
    return
}

func bases(s string, n int, t ...bool) (res string) {
    _, res = _bases(s, n, t...)
    return
}
func _bases(s string, n int, t ...bool) (d, b string) {
    d = filepath.Dir(s)
    b = filepath.Base(s)
    for i := n-1; 0 < i; i -= 1 {
        b = filepath.Join(filepath.Base(d), b)
        d = filepath.Dir(d)
    }
    if t != nil && t[0] && d != "" && d != "." {
        b = filepath.Join("…", b)
    }
    return
}

type __bases struct { builtinbase ; n int `num,size,count` }
func (ctx *__bases) inner() Context { return &ctx.builtinbase }
func (ctx *__bases) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__bases) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    var l []Value
    for _, a := range ctx.a {
        var s string
        if ctx.fullname {
            s, _ = as{a}.fullname_string(ctx)
        } else {
            s = a.string(ctx)
        }

        _, s = _bases(s, ctx.n)
        l = append(l, _strlit(a.Position(), s))
    }
    return l
}

type __base1 struct { __bases }
func (ctx *__base1) inner() Context { return &ctx.__bases }
func (ctx *__base1) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base1) x() any { ctx.n = 1; return ctx.__bases.x() }

type __base2 struct { __bases }
func (ctx *__base2) inner() Context { return &ctx.__bases }
func (ctx *__base2) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base2) x() any { ctx.n = 2; return ctx.__bases.x() }

type __base3 struct { __bases }
func (ctx *__base3) inner() Context { return &ctx.__bases }
func (ctx *__base3) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base3) x() any { ctx.n = 3; return ctx.__bases.x() }

type __base4 struct { __bases }
func (ctx *__base4) inner() Context { return &ctx.__bases }
func (ctx *__base4) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base4) x() any { ctx.n = 4; return ctx.__bases.x() }

type __base5 struct { __bases }
func (ctx *__base5) inner() Context { return &ctx.__bases }
func (ctx *__base5) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base5) x() any { ctx.n = 5; return ctx.__bases.x() }

type __base6 struct { __bases }
func (ctx *__base6) inner() Context { return &ctx.__bases }
func (ctx *__base6) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base6) x() any { ctx.n = 6; return ctx.__bases.x() }

type __base7 struct { __bases }
func (ctx *__base7) inner() Context { return &ctx.__bases }
func (ctx *__base7) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base7) x() any { ctx.n = 7; return ctx.__bases.x() }

type __base8 struct { __bases }
func (ctx *__base8) inner() Context { return &ctx.__bases }
func (ctx *__base8) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base8) x() any { ctx.n = 8; return ctx.__bases.x() }

type __base9 struct { __bases }
func (ctx *__base9) inner() Context { return &ctx.__bases }
func (ctx *__base9) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__bases.cast(t)
}
func (ctx *__base9) x() any { ctx.n = 9; return ctx.__bases.x() }

type __dir struct { __dirs ; sub Value `has,contain,contains` }
func (ctx *__dir) inner() Context { return &ctx.__dirs }
func (ctx *__dir) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var sub string
    if ctx.sub != nil {
        sub = ctx.sub.string(ctx)
    }
    if sub == "" {
        ctx.n = 1
        return ctx.__dirs.x()
    }

    var l []Value
    for _, a := range merge(ctx.a...) {
        var s string
        if ctx.fullname {
            s, _ = as{a}.fullname_string(ctx)
        } else {
            s = a.string(ctx)
        }
        for {
            var d = filepath.Dir(s)
            if d == "" || d == s { break } else { s = d }
            if _, e := os.Stat(filepath.Join(d,sub)); e == nil {
                l = append(l, _pathstr(ctx, d))
                break
            }
        }
    }
    return l
}

func dirs(n int, s string) (_ string) {
    for n > 0 {
        s = filepath.Dir(s)
        n -= 1
    }
    return s
}

type __dirs struct { builtinbase ; n int `num,size,count` }
func (ctx *__dirs) inner() Context { return &ctx.builtinbase }
func (ctx *__dirs) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__dirs) x() any {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var l []Value
    for _, a := range merge(ctx.a...) {
        var s string
        if ctx.fullname {
            s, _ = as{a}.fullname_string(ctx)
        } else {
            s = a.string(ctx)
        }

        s = dirs(ctx.n, s)

        if f, y := a.(*file); y {
            if ctx.fullname {
                f = stat(ctx, s, stat_nonexist{true})
            } else {
                f = stat(ctx, s, stat_nonexist{true}, stat_sub{f.sub}, stat_dir{f.dir})
            }
            l = append(l, f)
        } else if s != "" {
            l = append(l, _pathstr(ctx, s))
        } else {
            continue
        }
    }
    return l
}

type __dir1 struct { __dirs }
func (ctx *__dir1) inner() Context { return &ctx.__dirs }
func (ctx *__dir1) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir1) x() any { ctx.n = 1; return ctx.__dirs.x() }

type __dir2 struct { __dirs }
func (ctx *__dir2) inner() Context { return &ctx.__dirs }
func (ctx *__dir2) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir2) x() any { ctx.n = 2; return ctx.__dirs.x() }

type __dir3 struct { __dirs }
func (ctx *__dir3) inner() Context { return &ctx.__dirs }
func (ctx *__dir3) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir3) x() any { ctx.n = 3; return ctx.__dirs.x() }

type __dir4 struct { __dirs }
func (ctx *__dir4) inner() Context { return &ctx.__dirs }
func (ctx *__dir4) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir4) x() any { ctx.n = 4; return ctx.__dirs.x() }

type __dir5 struct { __dirs }
func (ctx *__dir5) inner() Context { return &ctx.__dirs }
func (ctx *__dir5) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir5) x() any { ctx.n = 5; return ctx.__dirs.x() }

type __dir6 struct { __dirs }
func (ctx *__dir6) inner() Context { return &ctx.__dirs }
func (ctx *__dir6) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir6) x() any { ctx.n = 6; return ctx.__dirs.x() }

type __dir7 struct { __dirs }
func (ctx *__dir7) inner() Context { return &ctx.__dirs }
func (ctx *__dir7) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir7) x() any { ctx.n = 7; return ctx.__dirs.x() }

type __dir8 struct { __dirs }
func (ctx *__dir8) inner() Context { return &ctx.__dirs }
func (ctx *__dir8) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir8) x() any { ctx.n = 8; return ctx.__dirs.x() }

type __dir9 struct { __dirs }
func (ctx *__dir9) inner() Context { return &ctx.__dirs }
func (ctx *__dir9) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__dirs.cast(t)
}
func (ctx *__dir9) x() any { ctx.n = 9; return ctx.__dirs.x() }

type __undirs struct { builtinbase ; n int `num,size,count` }
func (ctx *__undirs) inner() Context { return &ctx.builtinbase }
func (ctx *__undirs) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__undirs) x() any {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var l []Value
    for _, a := range ctx.a {
        var s string
        if ctx.fullname {
            s, _ = as{a}.fullname_string(ctx)
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
        l = append(l, _pathstr(ctx, filepath.Join(v...)))
    }
    return l
}

type __undir1 struct { __undirs }
func (ctx *__undir1) inner() Context { return &ctx.__undirs }
func (ctx *__undir1) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir1) x() any { ctx.n = 1; return ctx.__undirs.x() }

type __undir2 struct { __undirs }
func (ctx *__undir2) inner() Context { return &ctx.__undirs }
func (ctx *__undir2) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir2) x() any { ctx.n = 2; return ctx.__undirs.x() }

type __undir3 struct { __undirs }
func (ctx *__undir3) inner() Context { return &ctx.__undirs }
func (ctx *__undir3) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir3) x() any { ctx.n = 3; return ctx.__undirs.x() }

type __undir4 struct { __undirs }
func (ctx *__undir4) inner() Context { return &ctx.__undirs }
func (ctx *__undir4) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir4) x() any { ctx.n = 4; return ctx.__undirs.x() }

type __undir5 struct { __undirs }
func (ctx *__undir5) inner() Context { return &ctx.__undirs }
func (ctx *__undir5) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir5) x() any { ctx.n = 5; return ctx.__undirs.x() }

type __undir6 struct { __undirs }
func (ctx *__undir6) inner() Context { return &ctx.__undirs }
func (ctx *__undir6) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir6) x() any { ctx.n = 6; return ctx.__undirs.x() }

type __undir7 struct { __undirs }
func (ctx *__undir7) inner() Context { return &ctx.__undirs }
func (ctx *__undir7) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir7) x() any { ctx.n = 7; return ctx.__undirs.x() }

type __undir8 struct { __undirs }
func (ctx *__undir8) inner() Context { return &ctx.__undirs }
func (ctx *__undir8) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir8) x() any { ctx.n = 8; return ctx.__undirs.x() }

type __undir9 struct { __undirs }
func (ctx *__undir9) inner() Context { return &ctx.__undirs }
func (ctx *__undir9) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.__undirs.cast(t)
}
func (ctx *__undir9) x() any { ctx.n = 9; return ctx.__undirs.x() }

type __chopdir struct { builtinbase }
func (ctx *__chopdir) inner() Context { return &ctx.builtinbase }
func (ctx *__chopdir) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__chopdir) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var l []Value
    var n = 0
    if x := len(ctx.a); x > 0 {
        if v, ok := scalarize(ctx.a[0]).(*decimal); ok {
            ctx.a, n = ctx.a[1:], int(v.int64)
        } else if v, ok := scalarize(ctx.a[x-1]).(*decimal); ok {
            ctx.a, n = ctx.a[:x-1], int(v.int64)
        } else {
            erro(ctx, "require (first/last) integer argument (first=%T, last=%T)", ctx.a[0], ctx.a[x-1]).trace()
            return

        }
    }
    for _, a := range ctx.a {
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
        l = append(l, _strlit(a.Position(), filepath.Join(v...)))
    }
    return l
}

type __reldir struct { builtinbase }
func (ctx *__reldir) inner() Context { return &ctx.builtinbase }
func (ctx *__reldir) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__reldir) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var err error
    var l []Value
    var t string
    for i, a := range ctx.a {
        if s := a.string(ctx); i == 0 {
            t = s
        } else if s, err = filepath.Rel(t, s); err == nil {
            l = append(l, _strlit(a.Position(), s))
        } else {
            erro(ctx, "%v", err)
        }
    }
    return l
}

type __mkdir struct { builtinbase
    all bool `all,p,path`
}
func (ctx *__mkdir) inner() Context { return &ctx.builtinbase }
func (ctx *__mkdir) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__mkdir) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    for i, nargs := 0, len(ctx.a); i < nargs; i += 1 {
        var (
            a = ctx.a[i]
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
                erro(ctx, "Wrong size of list `%v'", t).trace()
            }
        case *list: // mkdir name perm, name perm, ...
            if t.len() == 2 {
                name = t.at(0).string(ctx)
                perm = filePerm(ctx, t.at(1), uint32(perm))
            } else {
                erro(ctx, "Wrong size of list `%v'", t).trace()
            }
        default: // mkdir name perm, name perm, ...
            name = ctx.a[i].string(ctx)
            if i+1 < nargs {
                perm = filePerm(ctx, ctx.a[i+1], uint32(perm))
                i += 1
            }
        }
        if ctx.all {
            if err := os.MkdirAll(name, perm); err != nil {
                erro(ctx, "%v", err).trace()
            }
        } else {
            if err := os.Mkdir(name, perm); err != nil {
                erro(ctx, "%v", err).trace()
            }
        }
    }
    return
}

type __chdir struct { builtinbase }
func (ctx *__chdir) inner() Context { return &ctx.builtinbase }
func (ctx *__chdir) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__chdir) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if len(ctx.a) == 1 {
        var str = ctx.a[0].string(ctx)
        if err := lockCD(str, 0); err != nil {
            erro(ctx, "%v", err).trace()
        }
    } else {
        erro(ctx, "wrong number of arguments: %v", len(ctx.a))
    }
    return
}

type __rename struct { builtinbase }
func (ctx *__rename) inner() Context { return &ctx.builtinbase }
func (ctx *__rename) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__rename) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    for i, nargs := 0, len(ctx.a); i < nargs; i += 1 {
        var (
            a = ctx.a[i]
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
                erro(ctx, "wrong size of group `%v'", t).trace()
            }
        case *list: // rename oldname newname, old new, ...
            if t.len() == 2 {
                oldname = t.at(0).string(ctx)
                newname = t.at(1).string(ctx)
            } else {
                erro(ctx, "wrong size of list `%v'", t).trace()
            }
        default: // rename newname oldname  newname oldname ...
            if i+1 < nargs {
                oldname = ctx.a[i+0].string(ctx)
                newname = ctx.a[i+1].string(ctx)
                i += 1
            } else {
                erro(ctx, "Wrong arguments `%v'", ctx.a).trace()
            }
        }
        if err := os.Rename(oldname, newname); err != nil {
            erro(ctx, "%v", err).trace()
        }
    }
    return
}

type __remove struct { builtinbase
    skip string `skip`
    ignoreMissing bool `ig,ignore,ignore-missing,ignore-not-found`
    warnNotFile bool `warn-not-file`
    all bool `all,recursive`
}
func (ctx *__remove) inner() Context { return &ctx.builtinbase }
func (ctx *__remove) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__remove) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var opts = ctx
    var remove func(Context, Value)
    var removeFile = func(ctx Context, f *file) {
        var err error
        var s = f.fullname()
        if opts.skip != "" {
            if strings.HasPrefix(s, opts.skip) { return } else
            if strings.HasPrefix(f.ident(ctx), opts.skip) { return }
        }
        if opts.all { err = os.RemoveAll(s) } else { err = os.Remove(s) }
        if err != nil {
            erro(ctx, "remove: %v", err)
            erro(ctx, "remove: %v → %s", f, s).trace()
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
            erro(ctx, "remove path: %v", p).trace()
            return
        }
        if err != nil {
            erro(ctx, "remove: %v", err)
            erro(ctx, "remove: %v", p).trace()
            return
        }
        if d := opts.debug; d>0 { warn(ctx, "remove %s", s).debug(d) }
        if opts.verbose { prompt(ctx, "removed %s\n", s) }
    }
    var removePat = func(ctx Context, pat Value) {
        // var val = (&__wildcard{__:__{evocation:?}})._do(pat)
        // erro(ctx, "TODO: remove: %v → %v", pat, val).trace()
        erro(ctx, "TODO: remove: %v", ts(pat)).trace()
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
        } else if f, y := v.(*file); y {
            removeFile(ctx, f)
        } else if f = findfile(ctx, v.string(ctx)); f != nil {
            removeFile(ctx, f)
        } else if p, y := v.(*path); y {
            removePath(ctx, p)
        } else if !opts.ignoreMissing {
            errostack(ctx, 5, "not file: %v (%T)", v, v).trace()
        }
    }
    for _, a := range ctx.a {
        ctx := ctx.Context
        remove(ctx, a.expand(ctx))
    }

    if opts.debug > 0 { warn(ctx, "%v", ctx.a).debug() }
    if opts.debug > 0 && flush(ctx) > 0 {
        errostack(ctx, 3, "remove errors").trace()
    }
    return
}

type __truncate struct { builtinbase }
func (ctx *__truncate) inner() Context { return &ctx.builtinbase }
func (ctx *__truncate) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__truncate) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    for i, nargs := 0, len(ctx.a); i < nargs; i += 1 {
        var (
            a = ctx.a[i]
            name string
            size int64
        )
        switch t := a.(type) {
        case *pair: // truncate name ⇒ size old ⇒ new
            name = t.key.string(ctx)
            size = t.val.int(ctx)
        case *group: // truncate (name size) (old new)
            if t.len() == 2 {
                name = t.at(0).string(ctx)
                size = t.at(1).int(ctx)
            } else {
                erro(ctx, "Wrong size of group `%v'", t).trace()
                break
            }
        case *list: // truncate name size, old new, ...
            if t.len() == 2 {
                name = t.at(0).string(ctx)
                size = t.at(1).int(ctx)
            } else {
                erro(ctx, "Wrong size of list `%v'", t).trace()
                break
            }
        default: // truncate name size  name size ...
            if i+1 < nargs {
                name = ctx.a[i+0].string(ctx)
                size = ctx.a[i+1].int(ctx)
                i += 1
            } else {
                erro(ctx, "Wrong arguments `%v'", ctx.a).trace()
                break
            }
        }
        if err := os.Truncate(name, size); err != nil {
            erro(ctx, "%v", err).trace()
            break
        }
    }
    return
}

type __link struct { builtinbase }
func (ctx *__link) inner() Context { return &ctx.builtinbase }
func (ctx *__link) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__link) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    for i, nargs := 0, len(ctx.a); i < nargs; i += 1 {
        var (
            oldname, newname string
            a = ctx.a[i]
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
                erro(ctx, "Wrong size of group `%v'", t).trace()
                break
            }
        case *list: // link oldname newname, old new, ...
            if t.len() == 2 {
                oldname = t.at(0).string(ctx)
                newname = t.at(1).string(ctx)
            } else {
                erro(ctx, "Wrong size of list `%v'", t).trace()
                break
            }
        default: // link oldname newname  oldname newname ...
            if i+1 < nargs {
                oldname = ctx.a[i+0].string(ctx)
                newname = ctx.a[i+1].string(ctx)
                i += 1
            } else {
                erro(ctx, "Wrong arguments `%v'", ctx.a).trace()
                break
            }
        }
        if err := os.Link(oldname, newname); err != nil {
            erro(ctx, "%v", err).trace()
            break
        }
    }
    return
}

func _readlink(ctx Context, filename string, d os.FileInfo) (_ string, linked bool) {
    fn, linkpath := filename, filepath.Dir(filename)
    for d.Mode()&os.ModeSymlink != 0 {
        linkname, e := os.Readlink(fn)

        if e != nil {
            prompt(ctx, "%s: readlink failed\n", fn)
            note(ctx, "%v", e).debug()
            return
        }

        var rel = !filepath.IsAbs(linkname)
        if rel {
            linkname = filepath.Join(linkpath, linkname)
            linkpath = filepath.Dir(linkname)
        }

        if d, e = os.Lstat(linkname); e != nil {
            prompt(ctx, "%s: lstat %s\n", fn, linkname)
            note(ctx, "%v", e).debug()
            return
        }

        fn, linked = linkname, true
    }
    return fn, linked
}

func readlink(ctx Context, filename string) (_ string, _ bool) {
    if d, e := os.Stat(filename); e == nil {
        return _readlink(ctx, filename, d)
    }
    return
}

/* Example:
foo: foobar
	symlink -pluv $< $@
*/
type __symlink struct { builtinbase
    path     bool `path`
    force    bool `force,overwrite`
    update   bool `update`
    relative bool `rel,relative`
}
func (ctx *__symlink) inner() Context { return &ctx.builtinbase }
func (ctx *__symlink) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__symlink) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
outer:
    for i, na := 0, len(ctx.a); i < na; i += 1 {
        var (
            opts = *ctx // make a copy
            srcNameVal, dstNameVal Value
            srcName   , dstName    string
            srcDir    , dstDir     string
            aa []Value
        )
        switch t := ctx.a[i].(type) {
        case *pair: // symlink srcName=dstName srcName=>dstName...
            srcNameVal, dstNameVal = t.key, t.val
        case *group: // symlink (-u srcName dstName) (-v srcName dstName)...
            if aa = parse_opts(ctx, &opts, t.elems...); len(aa) != 2 {
                erro(ctx, "expects two values for group").trace()
                return
            } else {
                srcNameVal, dstNameVal = aa[0], aa[1]
            }
        case *list: // XXX: symlink old new, old new, ...
            if aa = parse_opts(ctx, &opts, t.elems...); len(aa) != 2 {
                erro(ctx, "expects two values for list").trace()
                return
            } else {
                srcNameVal, dstNameVal = aa[0], aa[1]
            }
        default:// Multiple pairs of names:
            // symlink  new old, new old ...
            // symlink  new old  new old ...
            if i+1 < na {
                srcNameVal = ctx.a[i+0]
                dstNameVal = ctx.a[i+1]
                i += 1
            } else {
                var a = auto_get(ctx,"@")
                var l = auto_get(ctx,"<")
                var r = auto_get(ctx,">")
                prompt(ctx, "symlink: args=%v → %v\n", ctx.a, t)
                prompt(ctx, "symlink: %v, %v, %v\n", a, l, r)
                errostack(ctx, 5, "expects pair of names (%T %v)", t, t).trace()
                return
            }
        }

        if srcDir, srcName = splitFileName(ctx, srcNameVal); srcName == "" {
            prompt(ctx, "symlink: args=%v\n", ctx.a)
            prompt(ctx, "symlink: src=%v\n", srcNameVal)
            errostack(ctx, 5, "empty src filename (%T)", srcNameVal).trace()
            return
        }
        if dstDir, dstName = splitFileName(ctx, dstNameVal); dstName == "" {
            prompt(ctx, "symlink: args=%v\n", ctx.a)
            prompt(ctx, "symlink: dest=%v\n", dstNameVal)
            errostack(ctx, 6, "empty dest filename (%T)", dstNameVal).trace()
            return
        }

        var src = srcName
        var dst = dstName
        if !filepath.IsAbs(src) { src = filepath.Join(srcDir, srcName) }
        if !filepath.IsAbs(dst) { dst = filepath.Join(dstDir, dstName) }
        if _, err := os.Stat(src); err != nil {
            prompt(ctx, "symlink: %v: %v\n", srcName, err)
            errostack(ctx, 6, "%v does not exist", srcName).trace()
            return
        }

        if !opts.relative {/* no rel required */} else
        if s, e := filepath.Rel(filepath.Dir(dst), src); e != nil {
            prompt(ctx, "symlink: %s: rel(%s, %s)\n", dstName, dst, src)
            errostack(ctx, 8, "%v", e).trace()
            return
        } else {
            if false {
                info(ctx, "%v %v\t%s", srcDir, srcName, src)
                info(ctx, "%v %v\t%s", dstDir, dstName, dst)
                info(ctx, "%v", s).debug()
            }
            src = s
        }

        if !opts.path {/* no mkdir */} else
        if dstDir == "" || dstDir == "." || dstDir == pathSep {
            // no need to mkdir: . or /
        } else if err := os.MkdirAll(dstDir, os.FileMode(0755)); err != nil {
            erro(ctx, "%v", err).trace()
            return
        }

        var rm bool
        if rm = opts.force; rm {
            // overwrite...
        } else if s, e := os.Readlink(dst); e != nil {
            if false {
                prompt(ctx, "%v: readlink failed (%T)\n", dstName, e)
                errostack(ctx, 6, "%v", e).trace()
            }
        } else if rm = s != src; !rm {
            continue outer
        }

        if rm { if e := os.Remove(dst); e != nil {
            prompt(ctx, "%v: remove old symlink failed (%T)\n", dstName, e)
            errostack(ctx, 6, "%v", e).trace()
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

type __stat struct { builtinbase
    symbol bool `sym,symbol,symlink,link`
    file   bool `file`
    dir    bool `dir`
}
func (ctx *__stat) inner() Context { return &ctx.builtinbase }
func (ctx *__stat) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__stat) x() (res any) {
    if len(ctx.a) == 0 { return }
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var proj = _project(ctx)
    if proj == nil {
        erro(ctx, "unknown project").trace()
        return
    }

    var vals []Value
    var check = func(f *file) {
        if f != nil && f.info != nil {
            var mode = f.info.Mode()
            if  (ctx.dir    && mode&os.ModeDir     != 0) ||
                (ctx.file   && mode&os.ModeType    != 0) ||
                (ctx.symbol && mode&os.ModeSymlink != 0) ||
                (!ctx.dir && !ctx.file && !ctx.symbol) { vals = append(vals, f) }
        }
    }

    var checkstat = func(a Value) {
        var f *file
        var s string
        if s = a.string(ctx); filepath.IsAbs(s) {
            f = stat(ctx, s)
        } else {
            f = stat(ctx, s, proj) // aka stat_dir{proj.absPath}
        }
        if f == nil { f = proj.file(ctx, s) }
        if f != nil { check(f) }
    }

    for _, a := range merge(ctx.a...) {
        switch t := a.(type) {
        case *file: check(t)
        case *path: checkstat(a)
        default:    checkstat(a)
        }
    }
    return vals
}

type __file struct { builtinbase
    exists bool `exist,exists,must,must-exist,required`
    report bool `report,reportmissing,report-missing`
    ignore bool `ignore,ignore-missing,missing,nonexist,non-exist`
}
func (ctx *__file) inner() Context { return &ctx.builtinbase }
func (ctx *__file) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__file) x() any {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    var res []Value
    for _, a := range merge(ctx.a...) {
        if x, y := to_file(a); y {
            if !ctx.exists || x.exists() /* || x.stat(ctx) != nil */ {
                res = append(res, try_fullfile(ctx, x))
            } else if ctx.report {
                info(ctx, "no such file {%v %v %v}", x.dir, x.sub, x.name).debug()
            }
            continue
        }
        for _, f := range select_files(ctx, unmap_files(ctx, a)) {
            if !ctx.exists || f.exists() {
                res = append(res, try_fullfile(ctx, f))
            } else if ctx.ignore {
                if ctx.verbose { info(ctx, "%v → %v", tv(a), f).debug() }
            } else if ctx.exists {
                errostack(ctx, 5, `not a file: %v : %s ; %s`, a, ts(a), ts(res)).trace()
            }
        }
    }
    return res
}

type __glob struct { builtinbase
    symbol bool `sym,symlink,symbol`
    dir bool `dir,directory`
    file bool `file`
}
func (ctx *__glob) inner() Context { return &ctx.builtinbase }
func (ctx *__glob) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__glob) x() (_ any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var cwd string // TODO: get current work directory
    var proj *project
    if proj = _project(ctx); proj == nil {
        erro(ctx, "unknown current cntext").trace()
    }

    var res []Value
    for _, a := range ctx.a {
        var ( str string; names []string )
        if str = a.string(ctx); !filepath.IsAbs(str) {
            str = filepath.Join(cwd, str)
        }

        var err error
        if names, err = filepath.Glob(str); err != nil {
            erro(ctx, "glob '%v' failed: %v", str, err).trace()
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
        if errorMissing {
            erro(ctx, "%v", err).trace()
        }
        return
    } else if !f.IsDir() {
        erro(ctx, "not dir: %v", sd).trace()
    } else if dir, err := os.Open(sd); err != nil {
        erro(ctx, "not dir: %v", sd).trace()
    } else if names, err = dir.Readdirnames(-1); err != nil { // NOTE: see also filepath.Glob(...)
        if errorMissing {
            erro(ctx, "readdir: %v", err).trace()
        }
        return
    } else {
        dir.Close()
    }
    return
}

type __wildcard struct { builtinbase
    includeMissing bool `include,includemissing,include-missing,missing,all`
    ignoreMissing bool `ignore,ignoremissing,ignore-missing`
    errorMissing bool `err,error,errormissing,error-missing,no-missing`
    exclude []Value `exclude,except,no,not`
    filetype string `type` // dir, file, etc.
    dir string `dir,directory`
}
func (ctx *__wildcard) inner() Context { return &ctx.builtinbase }
func (ctx *__wildcard) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__wildcard) _directory(topDir string, pats ...Value) (files []*file) {
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
        default:
            erro(ctx, "unknown -filetype: %s (%v)", ctx.filetype, f).trace()
        }
        top.Unlock()
        top.Done()
    }
    var subcard = func(sub *subr, pat Value) {
        defer sub.Done()

        if t, y := pat.(compositePattern); y { pat = t.Value }
        if t, y := pat.(*list); y {
            warn(ctx, "pattern is a list: %T %v %v", pat, pat, t.elems).debug()
            if len(t.elems) == 1 { pat = t.elems[0] }
        }

        var ctx = ctx
        if p, y := pat.(*path); !y {
            // fallthrough
        } else if nElems := len(p.elems); nElems == 0 {
            errostack(ctx, 3, "empty path: %v", pat).trace()
        } else if y, _, _ = p.elems[0].match(ctx, sub.n); y && nElems == 1 {
            errostack(ctx, 3, "%v %v: invalid path: %v, %v, %v", topDir, sub.dn, pat, sub.n, nElems).trace()
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
            if y { warn(ctx, "%T %v %v", pat, pat, sub).debug() }
            return
        }

        if gp, y := pat.(*globpat); !y {
            // fallthrough
        } else if len(gp.elems) == 0 {
            errostack(ctx, 3, "empty glob: %v (%s)", pat, sub.dn).trace()
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
            errostack(ctx, 3, "%v", err).trace()
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
func (ctx *__wildcard) _project_0(p *project, pats ...Value) (files []*file) {
    for _, pat := range pats {
        for _, a := range p.unmap_files(ctx, pat, nil) {
            for _, loc := range a.paths {
                var dir = loc.string(ctx)
                note(ctx, "%v %v %v %v", pat, a.pattern, loc, dir).debug()
            }
        }
    }
    return
}
func (ctx *__wildcard) _project_1(p *project, pats ...Value) (files []*file) {
    var g sync.WaitGroup
    for _, pat := range pats {
        g.Add(1)
        func() {
            defer g.Done()
            for _, a := range p.unmap_files(ctx, pat, nil) {
                note(ctx, "%v %v %v", pat, a.filemap.pattern, a.filemap.paths).debug()
            }
        } ()
    }
    g.Wait()
    return
}
func (ctx *__wildcard) _project(p *project, pats ...Value) (files []*file) {
    if false {
        defer func(t0 time.Time) {
            if d := time.Now().Sub(t0); d > 1*time.Second {
                var pos = _position(ctx)
                prompt(ctx, "%v: slow: %d patterns, %v\n", pos, len(pats), pats)
                prompt(ctx, "%v: slow: %d files\n", pos, len(files))
                prompt(ctx, "%v: slow: %v\n", pos, d).debug(4)
            }
        } (time.Now())
    }

    var m sync.Mutex
    var g sync.WaitGroup
    var collect = func(t ...*file) {
        m.Lock()
        files = append(files, t...)
        m.Unlock()
        g.Done()
    }

    var ne = ctx.includeMissing && !ctx.ignoreMissing
    var st = func(dir string, val Value) {
        if f := stat(ctx, val.string(ctx), stat_dir{dir}, stat_nonexist{ne}); f != nil {
            g.Add(1) ; go collect(f)
        } else if false {
            erro(ctx, "nil: %v : %s", ts(val), dir).trace()
        }
    }

    var dofilemap = func(lVal, rVal Value, fm filemap) {
        defer g.Done()
        var lPat = lVal.patterned(ctx)
        var rPat = rVal.patterned(ctx)
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
                note(ctx, "TODO: wildcard: 3. %v %v %s", lVal, rVal, dir).debug()
            }
        }
    }

    var f1 = func(lVal, rVal Value, fm filemap) {
        defer g.Done()
        if y, _, _ := lVal.match(ctx, rVal); y { // e.g. **.am <-> foo/bar/*.am
            g.Add(1) ; go dofilemap(lVal, rVal, fm)
        } else if y, _, _ = rVal.match(ctx, lVal); y {
            if g.Add(1) ; true {
                go dofilemap(lVal, rVal, fm)
            } else {
                go dofilemap(rVal, lVal, fm)
            }
        } else {
            erro(ctx, "TODO: wildcard: %v %v", rVal, lVal).trace()
        }
    }

    var f2 = func(pat Value, a filemap) {
        defer g.Done()
        for _, val := range a.primePatterns(ctx) {
            g.Add(1) ; go f1(pat, val, a)
        }
    }

    var f3 = func(pat Value) {
        defer g.Done()
        for _, a := range p.unmap_files(ctx, pat, nil) {
            g.Add(1) ; go f2(pat, a.filemap)
        }
    }

    for _, pat := range pats {
        g.Add(1) ; go f3(pat)
    }
    g.Wait()
    return
}
func (ctx *__wildcard) _do(pats ...Value) []*file {
    if ctx.dir == "" {
        return ctx._project(_project(ctx), pats...)
    } else {
        return ctx._directory(ctx.dir, pats...)
    }
}
func (ctx *__wildcard) x() any {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    if len(ctx.exclude) > 0 {
        ctx.exclude = xmerge(_final(ctx.Context), ctx.exclude...)
    }

    var vals []Value
    for _, f := range ctx._do(merge(ctx.a...)...) {
        if f == nil {
            errostack(ctx, 3, "nil file: %v", ctx.a).trace()
        } else {
            vals = append(vals, f)
        }
    }
    return vals
}

type __readdir struct { builtinbase }
func (ctx *__readdir) inner() Context { return &ctx.builtinbase }
func (ctx *__readdir) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__readdir) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var l []Value
    for _, a := range ctx.a {
        if fis, err := ioutil.ReadDir(a.string(ctx)); err == nil {
            v := new(list)
            for _, fi := range fis {
                v.append(_strlit(a.Position(), fi.Name()))
            }
            l = append(l, v)
        } else {
            break //l = append(l, _none(pos))
        }
    }
    return l
}

type __readfile struct { builtinbase
    trim      bool `ta,trim,trim-all`
    trimLeft  bool `tl,trim-left`
    trimRight bool `tr,trim-right`
}
func (ctx *__readfile) inner() Context { return &ctx.builtinbase }
func (ctx *__readfile) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__readfile) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    var l []Value
    var closured = closure_projects(ctx)
    for _, v := range ctx.a {
        if o := (as{v}.fullname(ctx, closured...)); o.Value == nil {
            errostack(ctx, 5, "%v is not a file", v).trace()
        } else if s, e := ioutil.ReadFile(o.string(ctx)); e != nil {
            errostack(ctx, 5, "read file failed: %v", e).trace()
        } else {
            if ctx.trim      { s = bytes.TrimFunc     (s, unicode.IsSpace) } else
            if ctx.trimLeft  { s = bytes.TrimLeftFunc (s, unicode.IsSpace) } else
            if ctx.trimRight { s = bytes.TrimRightFunc(s, unicode.IsSpace) }
            l = append(l, _strlit(v.Position(), string(s)))
        }
    }
    return l
}

type __writefile struct { builtinbase
    path bool `path`
}
func (ctx *__writefile) inner() Context { return &ctx.builtinbase }
func (ctx *__writefile) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__writefile) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }

    // $(write-file filename,content)
    // $(write-file -p filename,content)
outer:
    for i := 0; i < len(ctx.a); i += 1 {
        var (
            a = ctx.a[i]
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
                erro(ctx, "Wrong size of group `%v'", t).trace()
            }
        case *list: // write-file name text, name text 0660, ...
            if n := t.len(); n < 4 && n > 0 {
                name = t.at(0).string(ctx)
                if n > 1 { data = t.at(1).string(ctx) }
                if n > 2 { perm = filePerm(ctx, t.at(2),0600) }
            } else {
                erro(ctx, "Wrong size of list `%v'", t).trace()
            }
        default: // write-file name text 0660  name text 0660 ...
            name = ctx.a[i].string(ctx)
            if i+1 < len(ctx.a) {
                data = ctx.a[i+1].string(ctx)
                i += 1
            }
            if i+1 < len(ctx.a) {
                perm = filePerm(ctx, ctx.a[i+1],0600)
                i += 1
            }
        }
        if name == "" {
            continue outer
        } else if dir := filepath.Dir(name); ctx.path && dir != "." && dir != pathSep {
            if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                erro(ctx, "%v", err).trace()
            }
        }
        if err := ioutil.WriteFile(name, []byte(data), perm); err != nil {
            erro(ctx, "%v", err).trace()
        }
    }
    return
}

func touch(ctx Context, file Value, optMode uint32, optPath bool, ts ...time.Time) (err error) {
    var a, filename, c = as{file}.file_fullname(ctx)

    if filename == "" {
        errostack(ctx, 3, "touch: empty file name: %v (%v, %v, %v)", file, typeof(file), a, c).trace()
    } else if d := filepath.Dir(filename); optPath && d != "." && d != pathSep {
        if err = os.MkdirAll(d, os.FileMode(optMode|0733)); err != nil {
            erro(ctx, "touch: %v", err).trace()
        }
    }

    var (
        mode = os.FileMode(optMode)
        ta, tm time.Time
        m os.FileMode
    )
    if len(ts) > 0 { ta = ts[0] } else { ta = time.Now() }
    if len(ts) > 1 { tm = ts[1] } else { tm = time.Now() }
    if fi, k := to_file(file); k && fi.info != nil {
        m = fi.info.Mode()
    } else if fi, e := os.Stat(filename); e == nil && fi != nil {
        m = fi.Mode()
    } else {
        var f *os.File
        if m = mode; m == 0 { m = os.FileMode(0600); mode = m }
        if f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, m&os.ModePerm); err != nil {
            erro(ctx, "touch: %v", err).trace()
        } else if err = f.Close(); err != nil {
            erro(ctx, "touch: %v", err).trace()
        }
    }
    if err == nil {
        if err = os.Chtimes(filename, ta, tm); err != nil {
            erro(ctx, "touch: %v", err).trace()
        }
    }
    if err == nil && mode != 0 && m != 0 && mode != m {
        if err = os.Chmod(filename, mode); err != nil {
            erro(ctx, "touch: %v", err).trace()
        }
    }
    return
}

type __touchfile struct { builtinbase
    mode os.FileMode `mode`
    path bool `path`
}
func (ctx *__touchfile) inner() Context { return &ctx.builtinbase }
func (ctx *__touchfile) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__touchfile) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    // $(touch-file filename)
    // $(touch-file -p filename)
    for i := 0; i < len(ctx.a); i += 1 {
        if err := touch(ctx, ctx.a[i], uint32(ctx.mode), ctx.path); err != nil {
            erro(ctx, "%v", err).trace()
            break
        }
    }
    return
}

type __grep struct { builtinbase }
func (ctx *__grep) inner() Context { return &ctx.builtinbase }
func (ctx *__grep) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__grep) x() (_ any) {
    if !ctx.originalArgs {
        for i, a := range ctx.a {
            if i == 1 { continue }
            ctx.a[i] = a.expand(ctx)
        }
    }

    var args = ctx.a
    var nargs = len(args)
    if !(nargs == 2 || nargs == 3) {
        erro(ctx, "wrong args, try $(grep {=regex '^example$'},$0,$(file))").trace()
        return
    }

    var result Value
    var rxs []*regexp.Regexp // TODO: move it into builtinGrepOpts
    var rvs = merge(args[0])
    switch nargs {
    case 2:   args = args[1:]
    case 3: result = args[1]; args = args[2:]
    }

    for _, a := range rvs {
        if x, y := a.(*regexpat); y {
            rxs = append(rxs, x.Regexp)
        } else if s := a.string(ctx); s == "" {
            erro(ctx, "empty regexp").trace()
            return
        } else if r, e := regexp.Compile(s); e != nil {
            erro(ctx, "%v", e).trace()
            return
        } else {
            rxs = append(rxs, r)
        }
    }

    var res []Value
    for _, a := range merge(args...) {
        var c = pc(ctx, a)
        var filename string
        if x, y := a.(*file); y {
            filename = x.fullname()
        } else {
            filename = a.string(ctx)
        }

        var e error
        var f *os.File
        if filename == "" {
            errostack(c, 5, "empty filename: %v", ts(a)).trace()
            return
        } else if f, e = os.Open(filename); e != nil {
            errostack(c, 5, "%s ; %s", e, ts(a)).trace()
            return
        } else {
            defer f.Close()
        }

        var line int // line number
        var s = bufio.NewScanner(f)
        s.Split(bufio.ScanLines)

        for s.Scan() {
            text := s.Text()
            line += 1 // starting from #1

            cc := pc(c, filename, line)
            for _, rx := range rxs {
                sm := rx.FindStringSubmatch(text)
                if sm == nil { continue }

                ctx.defs = make(defmap) // ensure a clear defs map
                for i, n := range rx.SubexpNames() {
                    if n == "" { n = strconv.Itoa(i) }
                    ctx.set(cc, defVoid, n, _raw(_position(cc), sm[i]))
                    if false { note(cc, "%40v %-2v %-2v %-32v %v", rx, i, n, sm[i], ctx.defs) }
                }

                var val Value
                if result == nil {
                    val = _raw(_position(cc), sm[0])
                } else {
                    val = result.expand(_final(cc))
                }
                if checkpoints && truly(cc, is_test_mode{}) {
                    ctx.check_res(rx, text, result, val)
                }
                res = append(res, val)
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

func (p *project) strExpandConfig(ctx Context, s string) (result string, err error) {
    var pos Position
    var index, line = 0, 0
    var res = new(bytes.Buffer)
    if v := auto_get(ctx, "-file"); v != nil {
        if x, y := to_file(v); y { pos.Filename = x.fullname() }
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
        case m[4] > m[0] && m[5] > m[4]: name = s[m[4]:m[5]] //  @VAR@
        }

        var d *def
        var val Value
        if d = p.def(ctx, name); d == nil {
            if true {
                prompt(ctx, "%v: %v undefined\n", pos, name)
                warnstack(ctx, 10, "in %v", p).debug(6)
            }
            continue
        } else if val, _, _ = evoke(ctx, d, nil, nil); isNull(val) {
            if f := p.configuration_sm(ctx); f == nil {
                erro(ctx, "%v: configuration file not defined", name, f).trace()
                return
            } else if !f.exists() {
                prompt(ctx, "%s: file not exists (for %v)\n", f.fullname(), name)
                erro(ctx, "%v: configuration file not exists, try -conf first", name).trace()
                return
            }
            continue
        }

        switch t := val.(type) {
        case *undef, undef: // FIXME: fmt.Fprintf(res, "#undef")
        case *answer, *boolean:
            fmt.Fprintf(res, "%d", t.int(ctx))
        case *group:
            fmt.Fprintf(res, "%s", parseGroupValue(ctx, t).string(ctx))
        case *plain:
            fmt.Fprintf(res, "%s", t.String())
        default:
            fmt.Fprintf(res, "%s", val.string(ctx))
        }
    }
    if index < len(s) { fmt.Fprint(res, s[index:]) }
    result = res.String()
    return
}

// https://www.gnu.org/software/autoconf/manual/autoconf-2.67/autoconf.html
func autoconf(ctx Context, out *bytes.Buffer, p *project, str string) (err error) {
    var num int
    for _, m := range rxAutoconf.FindAllStringSubmatch(str, -1) {
        info(ctx, "TODO: %v", m)
        num += 1
    }
    warn(ctx, "TODO: %d", num).debug()
    return
}

func configurestring(ctx Context, out *bytes.Buffer, p *project, str string) {
    if s, e := p.strExpandConfig(ctx, str); e != nil {
        erro(ctx, "%v : %v", str, e).trace()
    } else {
        str = s
    }

    var index = 0

    for _, ii := range rxConfigure.FindAllStringSubmatchIndex(str, -1) {
        if _, e := out.WriteString(str[index:ii[0]]); e != nil {
            erro(ctx, "%v", e).trace()
        }

        index = ii[1]

        var (
            d *def
            t bool
            s string
            verb = str[ii[2]:ii[3]]
            name = str[ii[4]:ii[5]]
            hasv = ii[6] > ii[0] && ii[7] > ii[6]
        )
        if d = p.def(ctx, name); d != nil {
            if v, _, _ := evoke(ctx, d, nil, nil); v == nil {
                // noop, TODO: or #undef?
            } else if _, t := v.(*undef); t {
                _, e := out.WriteString(fmt.Sprintf("#undef /* %s */", name))
                if e != nil {
                    erro(ctx, "%v", e).trace()
                } else {
                    continue
                }
            } else {
                t = v.true(ctx)
            }
        }

        switch verb {
        case "define":
            if hasv /*&& !(def == nil || d.value == nil)*/ {
                v := str[ii[6]:ii[7]]
                s = fmt.Sprintf("#define %s %s", name, v)
            } else {
                s = fmt.Sprintf("#define %s", name)
            }
        case "undef":
            var va []Value
            if d == nil {
                s = fmt.Sprintf("#undef %s", name)
            } else if isNull(d.value) || isNone(d.value) {
                s = fmt.Sprintf("#undef %s /* %v */", name, d.value)
            } else if va = expand(ctx, d.value); len(va) == 1 {
                switch v := va[0].(type) {
                case *answer, *boolean:
                    if b := v.true(ctx); b {
                        s = fmt.Sprintf("#define %s 1 /* %s %v */", name, typeof(v), v)
                    } else {
                        s = fmt.Sprintf("#undef %s /* %s %v */", name, typeof(v), v)
                    }
                case *strlit:
                    s = strings.Replace(v.s, "\"", "\\\"", -1)
                    s = fmt.Sprintf("#define %s \"%s\"", name, v.s)
                default:
                    s = fmt.Sprintf("#define %s %v /* %s */", name, v, typeof(v))
                }
            } else {
                var v = d.value
                s = fmt.Sprintf("#define %s %v /* %s %v */", name, typeof(v), v, va)
            }
        case "smartdefine", "cmakedefine":
            if !t {
                s = fmt.Sprintf("/* #undef %s */", name)
            } else if hasv {
                v := str[ii[6]:ii[7]]
                s = fmt.Sprintf("#define %s %s", name, v)
            } else {
                s = fmt.Sprintf("#define %s", name)
            }
        case "smartdefine01", "cmakedefine01":
            if !t {
                s = fmt.Sprintf("#define %s 0", name)
            } else if hasv {
                v := str[ii[6]:ii[7]]
                s = fmt.Sprintf("#define %s 1 /* %s */", name, v)
            } else {
                s = fmt.Sprintf("#define %s 1", name)
            }
        }

        if _, e := out.WriteString(s); e != nil {
            erro(ctx, "%v", e).trace()
        }
    }

    if len(str) <= index {
        return
    }

    if _, e := out.WriteString(str[index:]); e != nil {
        erro(ctx, "%v", e).trace()
    }
    return
}

type __untraversed struct { builtinbase }
func (ctx *__untraversed) inner() Context { return &ctx.builtinbase }
func (ctx *__untraversed) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__untraversed) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    return untraversed{ease(ctx, ctx.a)}
}

type __return struct { builtinbase }
func (ctx *__return) inner() Context { return &ctx.builtinbase }
func (ctx *__return) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__return) x() (res any) {
    if !ctx.originalArgs {
        ctx.a = expand(ctx, ctx.a...)
    }
    return &returner{valbase{_position(ctx)}, ctx.a}
}
