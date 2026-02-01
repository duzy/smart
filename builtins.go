//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
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
	"unicode/utf8"
    "unicode"
    "unsafe"
    "regexp"
    "bytes"
    "bufio"
    "time"
    "sync"
	"slices"
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
    c.evocation.a = expands(c, c.evocation.a...)
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
    `trace`:     reflect.TypeOf((*__trace)(nil)).Elem(),

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
    `conjunct`:    reflect.TypeOf((*__conjunct)(nil)).Elem(), // concat
    `quote`:      reflect.TypeOf((*__quote)(nil)).Elem(),
    `unique`:     reflect.TypeOf((*__unique)(nil)).Elem(),

    `split`:            reflect.TypeOf((*__split)(nil)).Elem(),
    `split-quote`:      reflect.TypeOf((*__splitquote)(nil)).Elem(),
    `split-quote-join`: reflect.TypeOf((*__splitquotejoin)(nil)).Elem(),
    `split-join-quote`: reflect.TypeOf((*__splitjoinquote)(nil)).Elem(),

    `fields`:       reflect.TypeOf((*__fields)(nil)).Elem(),

    // `usee`:         reflect.TypeOf((*__usee)(nil)).Elem(),
    `uses`:         reflect.TypeOf((*__uses)(nil)).Elem(),

    `bare`:         reflect.TypeOf((*__bare)(nil)).Elem(),
    `path`:         reflect.TypeOf((*__path)(nil)).Elem(),
    `word`:         reflect.TypeOf((*__word)(nil)).Elem(),
    `finalize`:     reflect.TypeOf((*__finalize)(nil)).Elem(),
    `resolve`:      reflect.TypeOf((*__resolve)(nil)).Elem(),
    `strip`:        reflect.TypeOf((*__trim)(nil)).Elem(),
    `trim`:         reflect.TypeOf((*__trim)(nil)).Elem(),
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

    // commands ------------------------------------------------------------------
    `print`:        reflect.TypeOf((*__print)(nil)).Elem(),
    `printf`:       reflect.TypeOf((*__printf)(nil)).Elem(),

    `plain`:        reflect.TypeOf((*__plain)(nil)).Elem(),

    `append`:       reflect.TypeOf((*__append)(nil)).Elem(),
    // `unshift`:      reflect.TypeOf((*__unshift)(nil)).Elem(),
    // `pop`:          reflect.TypeOf((*__pop)(nil)).Elem(),

    `write-file`:   reflect.TypeOf((*__writefile)(nil)).Elem(), // io/ioutil/ioutil.go
    `touch-file`:   reflect.TypeOf((*__readfile)(nil)).Elem(),  // io/ioutil/ioutil.go

    `mkdir`:        reflect.TypeOf((*__mkdir)(nil)).Elem(),     // os/file.go
    `chdir`:        reflect.TypeOf((*__chdir)(nil)).Elem(),     // os/file.go
    `rename`:       reflect.TypeOf((*__rename)(nil)).Elem(),    // os/file.go
    `remove`:       reflect.TypeOf((*__remove)(nil)).Elem(),    // os/file_*.go
    `link`:         reflect.TypeOf((*__link)(nil)).Elem(),      // os/file_*.go
    `symlink`:      reflect.TypeOf((*__symlink)(nil)).Elem(),   // os/file_*.go
    `truncate`:     reflect.TypeOf((*__truncate)(nil)).Elem(),  // os/file_*.go

    `return`:       reflect.TypeOf((*__return)(nil)).Elem(),
    `serve-http`:   reflect.TypeOf((*__servehttp)(nil)).Elem(),
}

func escapedString(ctx Context, v Value) (s string) {
    if p, ok := v.(*strlit); ok {
        s = strings.Replace(__string(ctx, p), "\\'", "'", -1)
    } else {
        s = __string(ctx, v)
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
        val.SetBool(__true(ctx, v))
    case reflect.Float32, reflect.Float64:
        val.SetFloat(__float(ctx, v))
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        val.SetInt(__int(ctx, v))
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        val.SetUint(uint64(__int(ctx, v)))
    case reflect.String:
        val.SetString(__string(ctx, v))
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
            debug(pc(ctx,v), "option type unsupported: %v → %v, %v", ts(v), val.Kind(), val.Type(), trace{})
        }
    case reflect.Ptr:
        switch val.Type().Elem().String() {
        case "smart.fullname":
            if t := (as{v}.fullname(fullfile_ctx{ctx})); t.Value != nil {
                val.Set(reflect.ValueOf(&t))
            } else {
                debug(pc(ctx,v), _f("%v → %v", v, (as{v}.file(ctx))),
					_f("not a file: %v → %s", ts(v), ts(expand(ctx, v))), trace{})
            }
        case "smart.file":
            if t := (as{v}.file(ctx)); t != nil {
                val.Set(reflect.ValueOf(t))
            } else {
                debug(pc(ctx,v), _f("not a file: %v → %s", ts(v), ts(expand(ctx, v))), trace{})
            }
        case "regexp.Regexp":
            if rx, e := regexp.Compile(__string(ctx, v)); e == nil {
                val.Set(reflect.ValueOf(rx))
            } else {
                debug(pc(ctx,v), "wrong regexp: %v: %v", ts(v), e, trace{})
            }
        default:
            debug(pc(ctx,v), "option type unsupported: %v → %v, %v", ts(v), val.Elem().Kind(), val.Type().Elem(), trace{})
        }
    default:
        switch val.Type().String() {
        case "fs.FileMode", "os.FileMode": // aka. reflect.Uint32
            var t = __int(ctx, v)
            if t == 0 { debug(pc(ctx,v), "zero file mode") }
            val.SetUint(uint64(t))
        case "regexp.Regexp": // aka. reflect.Ptr
            debug(pc(ctx,v), "TODO: regexp: %v → %v, %v", ts(v), val.Kind(), val.Type(), trace{})
        default:
            debug(pc(ctx,v), "option type unsupported: %v → %v, %v", ts(v), val.Kind(), val.Type(), trace{})
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

        // skip parsing patterns, e.g. -I%
        if !patterned(ctx, arg) {
            switch t := arg.(type) {
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

    switch val.Type().String() {
    case "fs.FileMode", "os.FileMode":
		if val.Uint() == 0 { val.SetUint(0640) }
    }
    return
}

func _opts(ctx Context, opts reflect.Value, args []Value) (rest []Value) {
    if args == nil { return }

    if opts.Kind() != reflect.Ptr {
        debug(ctx, "opts must be ptr: %v", opts.Kind(), trace{})
    } else if opts = opts.Elem(); opts.Kind() != reflect.Struct {
        debug(ctx, "opts is not ptr of struct: %v", opts.Kind(), trace{})
    }

    rest = merge(args...)

    var builtin, general, modifier, dots reflect.Value
    var ot = opts.Type()
    for i := 0; i < ot.NumField(); i += 1 {
        var ft, fv = ot.Field(i), opts.Field(i)
        if ft.Tag == "..." {
            dots = fv
        } else if t := fv.Type(); fv.Kind() != reflect.Struct {
            if ft.Anonymous && ft.Name == "Context" && t.String() == "smart.Context" {
				continue
            }
            rest = _opt(ctx, ft.Tag, fv, rest...)
        } else if !ft.Anonymous {
            continue
        } else if ft.Name == "general_opts" {
            general = fv.Addr()
        } else if strings.HasPrefix(ft.Name, "__") {
            if builtin.IsValid() { debug(ctx, "embedded multiple builtins: %v", ft) }
            builtin = fv.Addr()
        } else if strings.HasPrefix(ft.Name, "modifier_") {
            if modifier.IsValid() { debug(ctx, "embedded multiple modifiers: %v", ft) }
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
func parse_opts(ctx Context, store any, vals ...Value) []Value {
    return _opts(ctx, reflect.ValueOf(store), vals)
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
        debug(ctx, "insufficient number of arguments", trace{})
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
    var vals []Value
    var scope = _scope(ctx)
    for _, a := range ctx.a {
        if s := __string(ctx, a); s == "" {
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
    var scope = _scope(ctx)
    for _, arg := range ctx.a {
        if d := scope.finddef(__string(ctx, arg)); d != nil && !isTrivial(d.value) {
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
    if t := time.Now(); len(ctx.a) > 0 {
        var vals []Value
        for _, a := range ctx.a {
            var s string
            if s = __string(ctx, a); s == "" {
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
    var s bytes.Buffer
    for i, a := range ctx.a {
        if i > 0 { fmt.Fprintf(&s, " ") }
        fmt.Fprintf(&s, "%s", __string(ctx, a))
    }
    if hook := _universe(ctx).hooks.debug; hook != nil {
        hook(ctx, s.String(), ctx.a)
    } else {
        debug(ctx, "%s", s.String(), callstack{num:ctx.n})
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
    var s bytes.Buffer
    for i, a := range ctx.a {
        if i > 0 { fmt.Fprintf(&s, " ") }
        fmt.Fprintf(&s, "%s", __string(ctx, a))
    }

    debug(ctx, "%s", s.String(), trace{})
    return
}

type __warning struct { builtinbase }
func (ctx *__warning) inner() Context { return &ctx.builtinbase }
func (ctx *__warning) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__warning) x() (res any) {
    var s bytes.Buffer
    for i, a := range ctx.a {
        if i > 0 { fmt.Fprintf(&s, " ") }
        fmt.Fprintf(&s, "%s", __string(ctx, a))
    }
    debug(ctx, "%s", s.String())
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

    if ctx.a == nil && hook != nil && !hook(ctx, nil, false) {
        prompt(ctx, "assert: %v\n", ctx.a)
        debug(ctx, s, t, callstack{num:d})
    }

    for _, a := range expands(ctx, ctx.a...) {
        if a == nil {
            debug(ctx, "nil argument", trace{})
            continue
        }

        var c = pc(ctx, a)
        var y = __true(c, a)
        if hook != nil && hook(c, a, y) || y {
            continue
        }

        debug(c, s, t, "%v ⇒ '%s'", ts(a), __string(c, a), callstack{num:d})
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
    for _, a := range ctx.a {
        if !__true(ctx, a) {
            debug(ctx, "assert: %v", ts(a), trace{})
        }
    }
    return ctx.a
}

type __trace struct { builtinbase }
func (ctx *__trace) inner() Context { return &ctx.builtinbase }
func (ctx *__trace) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trace) x() (res any) {
    for _, a := range ctx.a {
        note(ctx, "%v", ts(a), trace{})
    }
    return
}

// $(defor $(x),$(y),$(z)) is identical to $(if $(defined $(x)),$(x),...)
type __defor struct { builtinbase } // aka. defined-or
func (ctx *__defor) inner() Context { return &ctx.builtinbase }
func (ctx *__defor) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__defor) x() (res any) {
    for _, a := range merge(ctx.a...) {
        debug(ctx, "TODO: %v", ts(a), trace{})

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
	for _, a := range merge(ctx.a...) {
		if a = expand(ctx, a); __true(ctx, a) { return a }
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
    for _, a := range merge(ctx.a...) {
        if a = expand(ctx, a); __true(ctx, a) { res = a } else { return nil }
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
    var t bool
    for _, a := range ctx.a { if t = __true(ctx, a); t { break } }
    return !t
}

type __xor struct { builtinbase }
func (ctx *__xor) inner() Context { return &ctx.builtinbase }
func (ctx *__xor) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__xor) x() (res any) {
    if vals := merge(ctx.a...); len(vals) > 1 {
        var t = __true(ctx, vals[0])
        for _, a := range vals[1:] {
            if __true(ctx, a) != t {
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
    if len(ctx.a) != 2 {
        debug(ctx, _f("unequal: wrong number of arguments: %v", ctx.a),
			_f("try: $(unequal <value-list>,<value-list>)"), trace{})
    }

    var a = expand(_final(ctx), ctx.a[0])
    var b = expand(_final(ctx), ctx.a[1])
    var t = cmp(ctx, a, b) != cmpEqual

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
        debug(ctx, "unequal: %v", t, callstack{num:n})
    } else if len(ctx.a)>2 {
        debug(ctx, "unequal: extra args specified: %v", ctx.a[2])
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
    if len(ctx.a) != 2 {
        debug(ctx, "wrong number of arguments: %v", ctx.a)
        note(ctx, "try: $(equal <value-list>,<value-list>)", trace{})
    }

    args := expands(ctx, ctx.a...)

    if a, b := args[0], args[1]; ctx.str {
        return __string(ctx, a) == __string(ctx, b)
    } else {
        return cmp(ctx, a, b) == cmpEqual
    }
}

type __greater struct { builtinbase; str bool `str,string` }
func (ctx *__greater) inner() Context { return &ctx.builtinbase }
func (ctx *__greater) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__greater) x() (res any) {
    if len(ctx.a) != 2 {
        debug(ctx, "wrong number of arguments: %v", ctx.a)
        note(ctx, "try: $(greater <value-list>,<value-list>)", trace{})
    }

    args := expands(ctx, ctx.a...)

    if a, b := args[0], args[1]; ctx.str {
        if __string(ctx, a) > __string(ctx, b) { return true }
    } else {
        if cmp(ctx, a, b) == cmpGreater { return true }
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
    if len(ctx.a) != 2 {
        debug(ctx, "wrong number of arguments: %v", ctx.a)
        note(ctx, "try: $(greater <value-list>,<value-list>)", trace{})
    }

    args := expands(ctx, ctx.a...)

    if a, b := args[0], args[1]; ctx.str {
        if __string(ctx, a) < __string(ctx, b) { return true }
    } else {
        if cmp(ctx, a, b) == cmpSmaller { return true }
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
    if n := len(ctx.a); n < 2 {
        debug(ctx, "wrong arguments, try: $(match <regexp-list>,<value-list-1>,...)", trace{})
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
            if !patterned(ctx, left) && patterned(ctx, right) {
                matched, _, _ = match(ctx, right, left)
            } else {
                matched, _, _ = match(ctx, left, right)
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
        debug(ctx, "wrong arguments, try: $(match <regexp-list>,<value-list>,...)", trace{})
    }

    if len(ctx.a) > 1 {
        patList = merge(ctx.a[0])
        valList = merge(ctx.a[1:]...)
    } else {
        valList = merge(ctx.a[0])
    }
    if ctx.debug > 0 {
        var ( n = len(ctx.a) ; d = ctx.debug )
        debug(ctx, "match: %v %v %v, %d", ctx.regexps, patList, valList, n, callstack{num:d})
    }

    var pos = _position(ctx)
ForValList:
    for _, val := range valList {
        if isTrivial(val) { continue ForValList }

        var str = __string(ctx, val)
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
            var matched, _, _ = match(ctx, pat, str)
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
            debug(ctx, _f("match: %v", str), _f("match: %v %T", val, val))
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
    var val Value
    var args = merge(ctx.a...)
    if len(args) == 0 {
        return
    }
    if _, y := args[0].(*group); !y {
        val = expand(ctx, args[0])
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
        var v = expand(ctx, g.elems[0])
        if val == nil && v != nil && __true(ctx, v) {
            collect = true
        } else if val != nil && isTrivial(val) {
            if isTrivial(v) {
                collect = true
            } else if f, y := v.(flag); y && isNull(f.Value) {
                collect = true
            }
        } else if val != nil && cmp(ctx, val, v) == cmpEqual {
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
        debug(pc(ctx,arg), "unexpected case: %v", tv(arg), trace{})
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
func (ctx *__if) x() (res any) {
	if 1 < len(ctx.a) {
		if __true(ctx, ctx.a[0]) {
			return expand(ctx, ctx.a[1])
		} else {
			return expands(ctx, ctx.a[2:]...)
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
        if d := auto_find(ctx, __string(ctx, ctx.a[0])); d != nil && !isTrivial(d.value) {
            return ctx.a[1]
        } else {
            return ease(ctx, ctx.a[2:])
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
        if d := _scope(ctx).finddef(__string(ctx, ctx.a[0])); d != nil && !isTrivial(d.value) {
            return ctx.a[1]
        } else {
            return ease(ctx, ctx.a[2:])
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
        if equal(ctx, ctx.a[0], ctx.a[1]) {
            return ctx.a[2]
        } else {
            return ease(ctx, ctx.a[3:])
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
        if !equal(ctx, ctx.a[0], ctx.a[1]) {
            return ctx.a[2]
        } else {
            return ease(ctx, ctx.a[3:])
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
    debug(ctx, "TODO: $(for): %v", ts(ctx.a), trace{})
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

    var vals []Value
    var um map[uint64]Value
    if ctx.unique { um = make(map[uint64]Value) }

    for _, val := range merge(expand(ctx, ctx.a[0])) {
        if !ctx.empty && isEmpty(val) {
			continue
		} else if ctx.unique {
			var t = hash(ctx, val)
			if x, y := um[t]; y {
				if checkpoints && !equal(ctx, x, val) {
					debug(ctx, "%v != %v", ts(x), ts(val), trace{})
				}
				continue
			} else { um[t] = val }
		}

        // NOTE: don't use defStatic (it's for codeblock auto)
        ctx.set(ctx, defVoid, "_", redis(val))

		for _, v := range merge(expands(ctx, ctx.a[1:]...)...) {
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
    var num int64
    var vals = valvec(ctx.vals)
    for _, a := range ctx.a {
        if __true(ctx, a) || vals.has2(ctx, a) { num += 1 }
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
    var vals []Value
    for _, a := range ctx.a {
        if val := expand(ctx, a); isTrivial(val) {
            continue
        } else if s := strings.TrimSpace(__string(ctx, val)); s != "" {
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
    if 0 < len(ctx.a) {
        for _, a := range merge(ctx.o...) {
            switch t := a.(type) {
            case *pair:
                if k := __string(ctx, t.key); k == "" {
                    debug(pc(ctx,a), "empty name: %v : %s", t.key, ts(t.key), trace{})
                } else {
                    ctx.set(ctx, defVoid, k, t.val)
                }
            default:
                debug(pc(ctx,a), "wrong auto def: %s : %s", a, ts(a), trace{})
            }
        }
        return expands(ctx, ctx.a...)
    }
    return
}

type __var struct { builtinbase }
func (ctx *__var) inner() Context { return &ctx.builtinbase }
func (ctx *__var) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__var) x() (res any) {
    return
}

type __closure struct { builtinbase ; closure bool `closure` }
func (ctx *__closure) inner() Context { return &ctx.builtinbase }
func (ctx *__closure) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__closure) x() (res any) {
    var vals []Value
    var pos = _position(ctx)
    for _, a := range merge(ctx.a[0]) {
        vals = append(vals, makeClosure(pos, LPAREN, a, nil, ctx.a[1:]...))
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
    var vals []Value
    var pos = _position(ctx)
    for _, a := range merge(ctx.a[0]) {
        vals = append(vals, makeDelegate(pos, LPAREN, a, nil, ctx.a[1:]...))
    }
    return vals
}

type __call struct { builtinbase ; closure bool `closure` }
func (ctx *__call) inner() Context { return &ctx.builtinbase }
func (ctx *__call) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__call) _x() (res any) {
    var vals []Value
    for _, a := range merge(ctx.a[0]) {
        var x Value
        var s = __string(ctx, a)
        if s == "" {
            debug(ctx, "empty string: %v : %v", a, ts(a), trace{})
        } else if ctx.closure {
            x = closure_resolve(ctx, s)
        } else {
            x = project_resolve(ctx, s)
        }
        if x == nil { x = auto_get(ctx, s) }
        if x != nil {
            if v := evoke(ctx, x, nil, ctx.a[1:]); v != nil {
                vals = append(vals, v)
            }
        }
    }
    return vals
}
func (ctx *__call) x() (_ any) {
    var s = ctx.a[0].String()
    for _, v := range ctx.a[1:] { s += " " + v.String() }
    debug(ctx, "deprecated $(call %s), use $(%s)", s, s, trace{})
    return
}

type __value struct { builtinbase ; closure bool `closure` }
func (ctx *__value) inner() Context { return &ctx.builtinbase }
func (ctx *__value) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__value) x() (res any) {
    var vals []Value
    var p = _project(ctx)
    for _, a := range merge(ctx.a...) {
        var s = __string(ctx, a)
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
	sort bool `sort`
}
func (ctx *__defs) inner() Context { return &ctx.builtinbase }
func (ctx *__defs) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__defs) x() (_ any) {
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

            var a, _, c = match(ctx, pat, name)
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
	if ctx.sort { slices.Sort(names) }
    return names
}

type __list struct { builtinbase }
func (ctx *__list) inner() Context { return &ctx.builtinbase }
func (ctx *__list) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__list) x() (res any) {
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
    var scope = _scope(ctx)
    for _, a := range ctx.a {
        var ( o object ; s = __string(ctx, a) )
        if ctx.scope_ { _, o = scope.find(s) } else { o = project_resolve(ctx, s) }
        if o == nil {
            debug(ctx, "no such symbol: %s", s, trace{})
        } else if d, y := o.(*def); !y {
            debug(ctx, "not a def: %s: %v", s, typeof(o), trace{})
        } else if d.value != nil {
            d.value = expand(ctx, d.value)
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
    var err error
    var vals []Value
    var pos = _position(ctx)
    for _, a := range ctx.a {
        var bufout, buferr bytes.Buffer
        var s = __string(ctx, a)
        sh := exec.Command("sh", "-c", s)
        sh.Stdout, sh.Stderr = &bufout, &buferr
        if err = sh.Run(); err != nil {
            s = strings.TrimSpace(buferr.String())
            if !strings.HasPrefix(s, ":") { s = ":\n" + s }
            prompt(ctx, "%s%s\n", __string(ctx, a), s)
            debug(ctx, "%s", err, trace{})
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
    var vals []Value
    for _, a := range ctx.a {
        if s, err := exec.LookPath(__string(ctx, a)); err != nil {
            debug(ctx, "%v", err, trace{})
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
    if ctx.port == 0 { ctx.port = 80 }
    if ctx.ssl {
        debug(ctx, "'serve-http(-ssl)' is unimplemented yet", trace{})
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
            var s = __string(ctx, a)
            info(ctx, "serving files %v ...", s)
            http.Handle("/", http.FileServer(http.Dir(s)))
        }
    }

    flush(ctx)

    var err = server.ListenAndServe()
    if err != nil && err != http.ErrServerClosed {
        debug(ctx, "%s", err, trace{})
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
    if len(ctx.a) < 2 {
        debug(ctx, "insufficient number of arguments: %v", ctx.a, trace{})
    }

    var names []Value
    if names = merge(ctx.a[0]); len(names) == 0 {
        debug(ctx, "append to nowhere: %v", tv(ctx.a[0]))
        return
    }

    var vals []Value
    for _, a := range names {
        var s = __string(ctx, a)
        var d *def
        if s == "" {
            debug(ctx, "'%v' is empty for name", a, trace{})
        } else if ctx.auto {
            d = auto_find(ctx, s)
        } else if ctx.closure {
            d = closure_finddef(ctx, s)
        } else if o := project_resolve(ctx, s); o != nil {
            d, _ = o.(*def)
        }
        if d == nil {
            debug(ctx, "%v → %s is undefined", a, s, trace{})
        } else {
            if vals == nil {
                if vals = merge(ctx.a[1:]...); len(vals) == 0 {
                    debug(ctx, "append no values: %v", ctx.a[1:])
                    return
                }
            }
            d.append(ctx, vals...)
        }
    }
    return
}

type __plus struct { builtinbase ; int bool `int,integer` }
func (ctx *__plus) inner() Context { return &ctx.builtinbase }
func (ctx *__plus) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__plus) x() (res any) {
    if ctx.int {
        var num int64
        for n, a := range ctx.a {
            var i = __int(ctx, a)
            if n == 0 { num = i } else { num += i }
        }
        return _decimal(_position(ctx), num)
    } else {
        var num float64
        for n, a := range ctx.a {
            var f = __float(ctx, a)
            if n == 0 { num = f } else { num += f }
        }
        return _float(_position(ctx), num)
    }
}

type __minus struct { builtinbase ; int bool `int,integer` }
func (ctx *__minus) inner() Context { return &ctx.builtinbase }
func (ctx *__minus) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__minus) x() (res any) {
    if ctx.int {
        var num int64
        for n, a := range ctx.a {
            var i = __int(ctx, a)
            if n == 0 { num = i } else { num -= i }
        }
        return _decimal(_position(ctx), num)
    } else {
        var num float64
        for n, a := range ctx.a {
            var f = __float(ctx, a)
            if n == 0 { num = f } else { num -= f }
        }
        return _float(_position(ctx), num)
    }
}

type __multiply struct { builtinbase ; int bool `int,integer` }
func (ctx *__multiply) inner() Context { return &ctx.builtinbase }
func (ctx *__multiply) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__multiply) x() (res any) {
    if ctx.int {
        var num int64
        for n, a := range ctx.a {
            var i = __int(ctx, a)
            if n == 0 { num = i } else { num *= i }
        }
        return num
    } else {
        var num float64
        for n, a := range ctx.a {
            var f = __float(ctx, a)
            if n == 0 { num = f } else { num *= f }
        }
        return num
    }
}

type __divide  struct { builtinbase ; int bool `int,integer` }
func (ctx *__divide) inner() Context { return &ctx.builtinbase }
func (ctx *__divide) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__divide) x() (res any) {
    if ctx.int {
        var num int64
        for n, a := range ctx.a {
            var i = __int(ctx, a)
            if n == 0 { num = i } else { num /= i } // FIXME: NaN
        }
        return num
    } else {
        var num float64
        for n, a := range ctx.a {
            var f = __float(ctx, a)
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
    var args = ctx.a
    var t1, t2 time.Time

    if false { defer func() {
        t3 := time.Now()
        d0 := t3.Sub(t1)
        d1 := t2.Sub(t1)
        d2 := t3.Sub(t2)
        if d0 > 1*time.Second {
            for _, a := range args { __string(ctx, a) }
            t4 := time.Now()
            d3 := t4.Sub(t3)
            for i, a := range args { if i > 0 { cmp(ctx, a, args[i-1]) } }
            t5 := time.Now()
            d4 := t5.Sub(t4)
            // for i, a := range args { if i > 0 { eq(ctx, a, args[i-1]) } }
            for i, a := range args { if i > 0 { equal(ctx, a, args[i-1]) } }
            t6 := time.Now()
            d5 := t6.Sub(t5)
            var args2 []Value
            var seen = make(map[uint64]struct{})
            for _, a := range args {
                c := hash(ctx, a)
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
            debug(ctx, "%v %v %v (%v, %v, %v, %v, %d %d)", d0, d1, d2, d3, d4, d5, d6, len(args), len(args2))
            t7 = time.Now()
            unique(ctx, args...)
            d6 = t7.Sub(t6)
            debug(ctx, "unique: %v", d6, trace{})
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

type __conjunct struct { builtinbase }
func (ctx *__conjunct) inner() Context { return &ctx.builtinbase }
func (ctx *__conjunct) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__conjunct) x() (_ any) {
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
    return &quote{list{elements{ctx.a}}}
}

type __quotejoin struct { builtinbase }
func (ctx *__quotejoin) inner() Context { return &ctx.builtinbase }
func (ctx *__quotejoin) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__quotejoin) x() (res any) {
    var sep string
    var args = merge(ctx.a...)
    if l := len(args); l > 1 {
        sep = __string(ctx, args[l-1])
        args = args[:l-1]
    }
    if l := len(args); l > 0 {
        var fields []string
        for _, a := range args[1:] {
            if v := __string(ctx, a); v != "" { fields = append(fields, v) }
        }
        res = _strlit(_position(ctx), strconv.Quote(strings.Join(fields, sep)))
    } else {
        res = _none(_position(ctx))
    }
    return
}

// $(split .,1.2.3)
type __split struct { builtinbase
    sep string `sep,separator`
}
func (ctx *__split) inner() Context { return &ctx.builtinbase }
func (ctx *__split) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__split) x() (res any) {
    if 0 < len(ctx.a) {
        var fields []Value
        var sep = ctx.sep
        if sep == "" { sep = __string(ctx, ctx.a[0]) }
        for _, a := range ctx.a[1:] {
            for _, s := range strings.Split(__string(ctx, a), sep) {
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
            if s = __string(ctx, v); s != "" { strs = append(strs, s) }
        }
        res = _strlit(value.Position(), strings.Join(strs, sep))
    }
    return
}

// TODO: deprecate this and add -quote to __split
type __splitquote struct { __split }
func (ctx *__splitquote) inner() Context { return &ctx.builtinbase }
func (ctx *__splitquote) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__splitquote) x() (res any) {
    res = ctx.__split.x()
    if v, y := res.(Value); y && v != nil { quotestrings(v) }
    return
}

// TODO: deprecate this and add -quote to __split
type __splitquotejoin struct { __split }
func (ctx *__splitquotejoin) inner() Context { return &ctx.builtinbase }
func (ctx *__splitquotejoin) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__splitquotejoin) x() (res any) {
    res = ctx.__split.x()
    if val, y := res.(Value); y && val != nil {
        var err error
        var sep string
        if l := len(ctx.a); l > 1 {
            sep = __string(ctx, ctx.a[l-1])
            ctx.a = ctx.a[:l-1]
        }
        if res, err = joinstrings(ctx, val, sep); err != nil {
            debug(ctx, "%v", err, trace{})
        }
    }
    return
}

type __splitjoinquote struct { __split }
func (ctx *__splitjoinquote) inner() Context { return &ctx.builtinbase }
func (ctx *__splitjoinquote) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__splitjoinquote) x() (res any) {
    res = ctx.__split.x()
    if val, y := res.(Value); y && val != nil {
        var err error
        var sep string
        if l := len(ctx.a); l > 1 {
            sep = __string(ctx, ctx.a[l-1])
            ctx.a = ctx.a[:l-1]
        }

        var v Value
        if v, err = joinstrings(ctx, val, sep); err != nil {
            debug(ctx, "%v", err, trace{})
        } else {
            res = _strlit(_position(ctx), strconv.Quote(__string(ctx, v)))
        }
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
    if l := len(ctx.a); l >= 2 {
        var fields []string
        var s string = __string(ctx, ctx.a[1])
        var i int64 = __int(ctx, ctx.a[0])
        if l > 2 {
            fields = strings.Split(s, __string(ctx, ctx.a[2]))
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

type __usee struct { builtinbase }
func (ctx *__usee) inner() Context { return &ctx.builtinbase }
func (ctx *__usee) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__usee) x() (res any) {
    var proj = _project(ctx)
    if proj == nil {
        debug(ctx, "unknown current context", trace{})
    }

    var vals []Value
    for _, a := range ctx.a {
        v := sel(ctx, proj.use, __string(ctx, a))
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
        debug(ctx, "unknown current context", trace{})
    }

    var found bool

outer:
    for _, a := range ctx.a {
        var s = __string(ctx, a)
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
    var res []Value
    for _, a := range ctx.a {
        if x, y := a.(*path); y {
            res = append(res, x)
        } else {
            res = append(res, _pathStr(ctx, __string(ctx, a)))
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
    var vals []Value
    for _, a := range ctx.a {
        switch p := a.Position(); t := a.(type) {
        case *strlit, *strcomp:
            a = _word(p, __string(ctx, a))
        case *file:
            a = _word(p, ident(ctx,t))
        case fullfile:
            if ctx.name {
                a = _word(p, ident(ctx,t))
            } else {
                a = _word(p, __string(ctx,t))
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
    var vals []Value
    for _, a := range ctx.a {
        if _, y := a.(*word); !y {
            a = _word(a.Position(), __string(ctx, a))
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
    if 0 < len(ctx.a) {
        var resolve func(Context, string) object
        if ctx.closure {
            resolve = closure_resolve
        } else {
            resolve = project_resolve
        }

        var vals []Value
        for _, a := range merge(ctx.a...) {
            var name = __string(ctx, a)
            if o := resolve(ctx, name); o == nil {
                debug(ctx, "%v is nil : %v", a, ts(a), trace{})
            } else if x, y := o.(*def); !y {
                debug(ctx, "%v is not def : %v : %v", a, o, ts(o), trace{})
            } else if x.value != nil {
                vals = append(vals, merge(x.value)...)
            }
        }
        return vals
    }
    return
}

type __string_bin struct { builtinbase
    expand bool `expand`
    name   bool `name,file-name,non-full`
    con  bool `conjunct,conjunction`
    dis  bool `disjunct,disjunction`
    closure  []string `closure`
    def  []string `def,var`
    join []string `join`
}
func (ctx *__string_bin) inner() Context { return &ctx.builtinbase }
func (ctx *__string_bin) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__string_bin) x() (res any) {
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
                if t := __string(ctx, v); t != "" {
                    if 0 < i { s.WriteString(ctx.join[i % len(ctx.join)]) }
                    s.WriteString(t)
                }
            }
            return &strlit{valbase{_position(ctx)},s.String()}
        } else if ctx.con || !ctx.dis { // conjunction (default)
            var s bytes.Buffer
            for i, v := range vals {
                if t := __string(ctx, v); t != "" {
                    if 0 < i { s.WriteString(" ") }
                    s.WriteString(t)
                }
            }
            return &strlit{valbase{_position(ctx)},s.String()}
        } else { // disjunction
            var a []Value
            for _, v := range vals {
                if t := __string(ctx, v); t != "" {
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
    return expands(_final(ctx), ctx.a...)
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
            debug(ctx, "%v: slow: %v\n", pos, d)
        }
    } (time.Now())

    var f = func(v Value) Value {
        for _, pat := range pats {
            if full, res, stems := match(ctx, pat, v); full {
                if ctx.neg {
                    v = nil
                } else if ctx.stem {
                    v = ease(ctx, stems)
                } else if false {
                    if t, r := stencil(ctx, pat, stems); t != nil && len(r) == 0 {
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
    if len(ctx.a) > 1 {
        var i int
        var vals []Value
        var pats = merge(ctx.a[0])
        if len(pats) > 0 {
            i = 1 // good
        } else if pats = merge(ctx.a[1]); len(pats) == 0 {
            debug(ctx, "no patterns: %v", ctx.a, trace{})
        } else {
            i = 2
        }

        if len(ctx.a) < i {
            debug(ctx, "out of index: %d > %d, %v", i, len(ctx.a), ctx.a, trace{})
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
    var res []Value
    if n := len(ctx.a); n > 1 {
        var v1, v2 = ctx.a[0], ctx.a[1]
        var a, b = intVal(ctx, v1, -1), intVal(ctx, v2, -1)
        if ctx.a = ctx.a[2:]; a < -1 && b < -1 {
            debug(ctx, "wrong indices (%v, %v)", v1, v2, trace{})
        }
        if a > b { t := a; a = b; b = t } // swap the wrong order
        if a == -1 { a = b }
        if a == -1 { return }

        for _, arg := range ctx.a {
            var s = __string(ctx, arg)
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
    var res []Value
    if len(ctx.a) > 2 {
        var s1 = __string(ctx, ctx.a[0])
        var s2 = __string(ctx, ctx.a[1])
        for _, arg := range merge(ctx.a[2:]...) {
            s := strings.Replace(__string(ctx, arg), s1, s2, -1)
            res = append(res, _strlit(arg.Position(), s))
        }
    }
    return res
}

func coupleval(ctx Context, v Value, str string) (_ Value) {
	switch v.(type) {
	case *file, fullfile:
	case *strlit, *strcomp:
		return _strlit(_position(ctx), str)
	case *path:
		return _pathStr(ctx, str)
	default:
		if strings.Contains(str, pathSep) { return _pathStr(ctx, str) }
		return _word(_position(ctx), str)
	}
	return
}

// $(patsubst pattern,replacement,text)
// TODO: supports: $(var:pattern=replacement)
// TODO: supports: $(var:suffix=replacement)
// TODO: support flags -name and -full for name-only and full-name-only matching
type __patsubst struct { builtinbase
    mapfiles bool `map,mapfiles,map-files`
    fullfiles bool `fullfile,fullfiles`
    nofilemap bool `nomap,no-map,nofile,nofiles,no-files,no-filemap`
}
func (ctx *__patsubst) inner() Context { return &ctx.builtinbase }
func (ctx *__patsubst) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__patsubst) matchPats(pats []Value, a any) (ok bool, pat Value, stems []string) {
	for _, pat = range pats { if ok, _, stems = match(ctx, pat, a); ok { break } }
	return
}
func (ctx *__patsubst) srcFile(proj *project, src Value) (srcFile *file, source any, full bool) {
	var ok bool
	if srcFile, ok = to_file(src); ok {
		source = srcFile
	} else if ctx.mapfiles {
		var s = __string(ctx, src)
		if srcFile = proj.file(ctx, s); srcFile != nil {
			source = srcFile
		} else {
			source = s
		}
	} else if true {
		source = src
	} else {
		debug(ctx, "unexpected %s", ts(src), trace{})
	}
	if full = ctx.fullfiles; !full { _, full = src.(fullfile) }
	return
}
func (ctx *__patsubst) x() (_ any) {
    var (
        srcPats, dstPats, sources, res []Value
        proj = _project(ctx)
    )
	if nil != ctx.a {
		var l = len(ctx.a)
		if 0 < l { srcPats = xmerge(ctx, ctx.a[0]) }
		if 1 < l { dstPats = xmerge(ctx, ctx.a[1]) }
		if 2 < l { sources = xmerge(ctx, ctx.a[2:]...) }
	}
    for _, src := range sources {
		var srcFile, source, full = ctx.srcFile(proj, src)
        var ok, srcPat, stems = ctx.matchPats(srcPats, source)
		if !ok {
			if !isTrivial(src) { res = append(res, src) }
			continue // just append src to the list
		}

        for _, dstPat := range dstPats {
            var val, ramnant = stencil(ctx, dstPat, stems)
            if isNull(val) {
                debug(ctx, "nil stencil: %s: %v (stems=%v, ramnant=%v)", typeof(dstPat), dstPat, stems, ramnant, trace{})
            } else if 0 < ctx.debug {
                debug(ctx, "patsubst: %v: %v → %v → %v %v → %v %v", srcPat, src, source, stems, dstPat, val, ramnant)
            }

            var str string
            if str = __string(ctx, val); str == "" { continue }
            if srcFile != nil {
                var dst *file
                if ctx.nofilemap {
                    dst = _stat(ctx, str, stat_sub{srcFile.sub}, stat_dir{srcFile.dir}, stat_nonexist{true})
				} else {
					dst = proj.file(ctx, val)
				}
				if dst == nil {
					debug(ctx, "%v %v", srcPat, srcFile)
				} else if dst.position = srcPat.Position(); full {
                    res = append(res, fullfile{dst})
                } else {
                    res = append(res, dst)
                }
                continue
            }

			res = append(res, coupleval(pc(ctx, dstPat), src, str))
        }
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
    var res []Value
    for _, a := range ctx.a {
        switch t := a.(type) {
        case interface{ change(func(string) string) Value }:
            a = t.change(strings.Title)
        default:
            a = _raw(a.Position(), strings.Title(__string(ctx, a)))
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
    var res []Value
    for _, a := range ctx.a {
        switch t := a.(type) {
        case interface{ change(func(string) string) Value }:
            a = t.change(strings.ToUpper)
        default:
            a = _raw(a.Position(), strings.ToUpper(__string(ctx, a)))
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
    var res []Value
    for _, a := range ctx.a {
        switch t := a.(type) {
        case interface{ change(func(string) string) Value }:
            a = t.change(strings.ToLower)
        default:
            a = _raw(a.Position(), strings.ToLower(__string(ctx, a)))
        }
        res = append(res, a)
    }
    return res
}

type __trim struct { builtinbase }
func (ctx *__trim) inner() Context { return &ctx.builtinbase }
func (ctx *__trim) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trim) x() any {
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
            a = _raw(a.Position(), f(__string(ctx, a)))
        }
        res = append(res, a)
    }
    return res
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
            a = _raw(a.Position(), f(__string(ctx, a)))
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
            a = _raw(a.Position(), f(__string(ctx, a)))
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
func (ctx *__trimprefix) x() any {
	var res []Value
	var prefix = merge(expand(ctx, ctx.a[0]))
	for _, val := range merge(expands(ctx, ctx.a[1:]...)...) {
		var s = __string(ctx, val)
		for _, prefix := range prefix {
			full, r, _ := match(ctx, prefix, s)
			if full { s = "" ; break } // trim all for full prefix
			if t := joinp(ctx, r); strings.HasPrefix(s, t) {
				s = strings.TrimPrefix(s, t)
			} else {
				s = strings.TrimLeftFunc(s, unicode.IsSpace)
			}
		}
		if s == "" {
			res = append(res, _null(val.Position()))
		} else if strings.Contains(s, pathSep) {
			res = append(res, _pathStr(ctx, s))
		} else {
			res = append(res, _word(val.Position(), s))
		}
	}
	return res
}

type __trimsuffix struct { builtinbase }
func (ctx *__trimsuffix) inner() Context { return &ctx.builtinbase }
func (ctx *__trimsuffix) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__trimsuffix) x() any {
	var res []Value
	var prefix = merge(expand(ctx, ctx.a[0]))
	for _, val := range merge(expands(ctx, ctx.a[1:]...)...) {
		var s = __string(ctx, val)
		for _, prefix := range prefix {
			full, r, _ := match(reversal{ctx}, prefix, s)
			if full { s = "" ; break } // trim all for full prefix
			if t := joinp(ctx, r); strings.HasSuffix(s, t) {
				s = strings.TrimSuffix(s, t)
			} else {
				s = strings.TrimRightFunc(s, unicode.IsSpace)
			}
		}
		if s == "" {
			res = append(res, _null(val.Position()))
		} else if strings.Contains(s, pathSep) {
			res = append(res, _pathStr(ctx, s))
		} else {
			res = append(res, _word(val.Position(), s))
		}
	}
	return res
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
    var ext string
    var res []Value
    for i, a := range ctx.a {
        if s := __string(ctx, a); s != "" {
            if i == 0 && len(ctx.a) > 1 {
                ext = s
            } else if ext == "" {
                for ext = filepath.Ext(s); ext != ""; {
                    s = strings.TrimSuffix(s, ext)
                    if ctx.all { ext = filepath.Ext(s) } else { break }
                }
                res = append(res, _word(a.Position(), s))
            } else if ext == filepath.Ext(s) {
                res = append(res, _word(a.Position(), strings.TrimRight(s, ext)))
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
    var vals []Value
    for _, a := range merge(ctx.a...) {
        var s = __string(ctx, a)
        if !strings.HasSuffix(s, "/.git") {
            s = filepath.Join(s, ".git")
        }
        if i, e := os.Stat(s); e != nil {
            a = _pathStr(ctx, s)
        } else if m := i.Mode(); m.IsDir() {
            a = _pathStr(ctx, s)
        } else if m.IsRegular() {
            if b, e := ioutil.ReadFile(s); e != nil {
                debug(ctx, "%v", e, trace{})
            } else if !bytes.HasPrefix(b, []byte("gitdir:")) {
                debug(ctx, "%s", b, trace{})
            } else {
                t := string(bytes.TrimSpace(b[7:]))
                s = filepath.Join(filepath.Dir(s), t)
                a = _pathStr(ctx, s)
            }
        } else {
            debug(pc(ctx,a), "%v", s, trace{})
        }
        vals = append(vals, a)
    }
    return vals
}

type __add___fix struct { builtinbase; dis Value }
func (ctx *__add___fix) x(f func(_ Context, _, _ Value) Value) (_ any) {
    if len(ctx.a) < 1 {
        debug(ctx, "wrong args, try $(addprefix fix,...)", trace{})
    }

    var res []Value

    for _, fix := range xmerge(ctx,ctx.a[0]) {
        if !isTrivial(fix) {
            fix = redis(fix)
            for _, v := range xmerge(ctx, ctx.a[1:]...) {
                if !isTrivial(v) { res = append(res, f(ctx, fix, redis(v))) }
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
    return ctx.__add___fix.x(prefix)
}

type __addsuffix struct { __add___fix }
func (ctx *__addsuffix) inner() Context { return &ctx.builtinbase }
func (ctx *__addsuffix) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__addsuffix) x() (_ any) {
    return ctx.__add___fix.x(func (c Context, x, y Value) Value { return prefix(c, y, x) })
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
    if ctx.noErrs && 0 < diagCount(ctx, diagError) { return }
    if ctx.noWarn && 0 < diagCount(ctx, diagWarn)  { return }

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
        debug(ctx, "not enough args, try $(printf 'format', ...)", trace{})
    }

    var vals = merge(ctx.a[0])
    if len(vals) != 1 {
        debug(ctx, "not enough args, try $(printf 'format', ...)", trace{})
    }

    var i int
    var a []any
    var f = __string(ctx, vals[0])

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
                    a = append(a, __int(ctx, v))
                    continue outer
                case 'e', 'E', 'f', 'F', 'g', 'G':
                    a = append(a, __float(ctx, v))
                    continue outer
                case 'b', 'x', 'X':
                    switch k := v.kind(); {
                    case k&KindInteger != 0:
                        a = append(a, __int(ctx, v))
                        continue outer
                    case k&KindFloat != 0:
                        a = append(a, __float(ctx, v))
                        continue outer
                    default:
                        if t, e := strconv.Atoi(__string(ctx, v)) ; e == nil { a = append(a, t) } else {
                            debug(ctx, "%v: %v", v, e, trace{})
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
            debug(ctx, "requires integer argument (first|last)", trace{})
        }
    }
    for _, a := range ctx.a {
        var lines []string
        for _, line := range strings.Split(__string(ctx, a), "\n") {
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
        debug(ctx, "unexpected number of arguments, try $(contains a b c1 c2, v1 v2 …)", trace{})
    }

    var n int
    var vals = merge(ctx.a[0])
    var list = merge(ctx.a[1:]...)
    if len(vals) == 0 || len(list) == 0 {
        debug(ctx, "insufficient number of arguments: %v ⇒ %v %v", ctx.a, vals, list, trace{})
    }

outer:
    for _, val := range vals {
        var s string
        if ctx.string { s = __string(ctx, val) }

        for _, elem := range list {
            var t bool
            if ctx.match || patterned(ctx,val) {
                t, _, _ = match(ctx, val, elem)
            } else if ctx.string {
                t = __string(ctx, elem) == s
            } else {
                t = cmp(ctx, val, elem) == cmpEqual
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
    debug(ctx, "TODO: $(sort ...)", trace{})
    return
}

type __wordlist struct { builtinbase }
func (ctx *__wordlist) inner() Context { return &ctx.builtinbase }
func (ctx *__wordlist) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__wordlist) x() (res any) {
    debug(ctx, "TODO: $(wordlist ...)", trace{})
    return
}

type __words struct { builtinbase }
func (ctx *__words) inner() Context { return &ctx.builtinbase }
func (ctx *__words) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__words) x() (res any) {
    debug(ctx, "TODO: $(words ...)", trace{})
    return
}

type __firstword struct { builtinbase }
func (ctx *__firstword) inner() Context { return &ctx.builtinbase }
func (ctx *__firstword) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__firstword) x() (res any) {
    debug(ctx, "TODO: $(firstword ...)", trace{})
    return
}

type __lastword struct { builtinbase }
func (ctx *__lastword) inner() Context { return &ctx.builtinbase }
func (ctx *__lastword) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__lastword) x() (res any) {
    debug(ctx, "TODO: $(lastword ...)", trace{})
    return
}

type __encodebase64 struct { builtinbase }
func (ctx *__encodebase64) inner() Context { return &ctx.builtinbase }
func (ctx *__encodebase64) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__encodebase64) x() (res any) {
    if 0 < len(ctx.a) {
        pos := _position(ctx)
        buf := new(bytes.Buffer)
        enc := base64.NewEncoder(base64.StdEncoding, buf)
        for _, a := range ctx.a { enc.Write([]byte(__string(ctx, a))) }
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
    if 0 < len(ctx.a) {
        var res []Value
        for _, a := range ctx.a {
            var s = __string(ctx, a)
            if dat, err := base64.StdEncoding.DecodeString(s); err != nil {
                debug(ctx, "decode '%s' failed: %v", s, err, trace{})
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
    var res []Value
    for _, a := range merge(ctx.a...) {
        res = append(res, _strlit(a.Position(), filepath.Ext(__string(ctx, a))))
    }
    if 0 < len(res) { return res }
    return
}

func _bases(s string, a ...any) (d, b string) {
    if d, b = filepath.Dir(s), filepath.Base(s); a != nil && d != "" && d != "." {
        var k = 1
        for _, a := range a {
            switch t := a.(type) {
            case bool:
                if filepath.IsAbs(d) {
                    b = filepath.Join(d, b)
                } else if t {
                    b = filepath.Join("…", b)
                }
            case int:
                for i := t-k; 0 < i; i -= 1 {
                    b = filepath.Join(filepath.Base(d), b)
                    d = filepath.Dir(d)
                }
            case string:
                for d != "/" && d != "." && d != "" && len(d)+len(b)<len(s) {
                    if true { k += 1 }
                    if s := filepath.Base(d); s == t {
                        d = filepath.Dir(d)
                        break
                    } else {
                        b = filepath.Join(s, b)
                        d = filepath.Dir(d)
                    }
                }
            }
        }
    }
    return
}
func bases(s string, a ...any) (b string) {
    _, b = _bases(s, a...)
    return
}

type __bases struct { builtinbase ; n int `num,size,count` }
func (ctx *__bases) inner() Context { return &ctx.builtinbase }
func (ctx *__bases) cast(t reflect.Type) Context {
	if reflect.TypeOf(ctx) == t { return ctx }
	return ctx.builtinbase.cast(t)
}
func (ctx *__bases) x() any {
	var vals []Value
	for _, a := range ctx.a {
		var s string
		if ctx.fullname {
			s, _ = as{a}.fullname_string(ctx)
		} else {
			s = __string(ctx, a)
		}
		if s != "" {
			_, s = _bases(s, ctx.n)
			switch t := strings.Split(s, pathSep); len(t) {
			case 0:
			case 1: vals = append(vals, _word(a.Position(), s))
			default:
				var p = new(path)
				for _, t := range t {
					p.elems = append(p.elems, _word(a.Position(), t))
				}
				vals = append(vals, p)
			}
		}
	}
	return vals
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
    var sub string
    if ctx.sub != nil {
        sub = __string(ctx, ctx.sub)
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
            s = __string(ctx, a)
        }
        for {
            var d = filepath.Dir(s)
            if d == "" || d == s { break } else { s = d }
            if _, e := os.Stat(filepath.Join(d,sub)); e == nil {
                l = append(l, _pathStr(ctx, d))
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
    var l []Value
    for _, a := range merge(ctx.a...) {
        var s string
        if ctx.fullname {
            s, _ = as{a}.fullname_string(ctx)
        } else {
            s = __string(ctx, a)
        }

        s = dirs(ctx.n, s)

        if f, y := a.(*file); y {
            if ctx.fullname {
                f = _stat(ctx, s, stat_nonexist{true})
            } else {
                f = _stat(ctx, s, stat_nonexist{true}, stat_sub{f.sub}, stat_dir{f.dir})
            }
            l = append(l, f)
        } else if s != "" {
            l = append(l, _pathStr(ctx, s))
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
    var l []Value
    for _, a := range ctx.a {
        var s string
        if ctx.fullname {
            s, _ = as{a}.fullname_string(ctx)
        } else {
            s = __string(ctx, a)
        }
        var v = strings.Split(s, pathSep)
        if i := len(v); i == 0 {
            // v is empty
        } else if ctx.n < i {
            v = v[ctx.n:]
        } else {
            v = v[i-1:] // empty
        }
        l = append(l, _pathStr(ctx, filepath.Join(v...)))
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
    var l []Value
    var n = 0
    if x := len(ctx.a); x > 0 {
        if v, ok := scalarize(ctx.a[0]).(*decimal); ok {
            ctx.a, n = ctx.a[1:], int(v.int64)
        } else if v, ok := scalarize(ctx.a[x-1]).(*decimal); ok {
            ctx.a, n = ctx.a[:x-1], int(v.int64)
        } else {
            debug(ctx, "require (first/last) integer argument (first=%T, last=%T)", ctx.a[0], ctx.a[x-1], trace{})
            return

        }
    }
    for _, a := range ctx.a {
        var v = strings.Split(__string(ctx, a), pathSep)
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
    var err error
    var l []Value
    var t string
    for i, a := range ctx.a {
        if s := __string(ctx, a); i == 0 {
            t = s
        } else if s, err = filepath.Rel(t, s); err == nil {
            l = append(l, _strlit(a.Position(), s))
        } else {
            debug(ctx, "%v", err)
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
    for i, nargs := 0, len(ctx.a); i < nargs; i += 1 {
        var (
            a = ctx.a[i]
            perm = os.FileMode(0755)
            name string
        )
        switch t := a.(type) {
        case *pair: // mkdir name ⇒ perm name ⇒ perm
            name = __string(ctx, t.key)
            perm = filePerm(ctx, t.val, uint32(perm))
        case *group: // mkdir (name perm) (name perm)
            if t.len() == 2 {
                name = __string(ctx, t.at(0))
                perm = filePerm(ctx, t.at(1), uint32(perm))
            } else {
                debug(ctx, "Wrong size of list `%v'", t, trace{})
            }
        case *list: // mkdir name perm, name perm, ...
            if t.len() == 2 {
                name = __string(ctx, t.at(0))
                perm = filePerm(ctx, t.at(1), uint32(perm))
            } else {
                debug(ctx, "Wrong size of list `%v'", t, trace{})
            }
        default: // mkdir name perm, name perm, ...
            name = __string(ctx, ctx.a[i])
            if i+1 < nargs {
                perm = filePerm(ctx, ctx.a[i+1], uint32(perm))
                i += 1
            }
        }
        if ctx.all {
            if err := os.MkdirAll(name, perm); err != nil {
                debug(ctx, "%v", err, trace{})
            }
        } else {
            if err := os.Mkdir(name, perm); err != nil {
                debug(ctx, "%v", err, trace{})
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
    if len(ctx.a) == 1 {
        var str = __string(ctx, ctx.a[0])
        if err := lockCD(str, 0); err != nil {
            debug(ctx, "%v", err, trace{})
        }
    } else {
        debug(ctx, "wrong number of arguments: %v", len(ctx.a))
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
    for i, nargs := 0, len(ctx.a); i < nargs; i += 1 {
        var (
            a = ctx.a[i]
            oldname, newname string
        )
        switch t := a.(type) {
        case *pair: // rename oldname=newname
            oldname = __string(ctx, t.key)
            newname = __string(ctx, t.val)
        case *group: // rename (oldname newname) (old new)
            if t.len() == 2 {
                oldname = __string(ctx, t.at(0))
                newname = __string(ctx, t.at(1))
            } else {
                debug(ctx, "wrong size of group `%v'", t, trace{})
            }
        case *list: // rename oldname newname, old new, ...
            if t.len() == 2 {
                oldname = __string(ctx, t.at(0))
                newname = __string(ctx, t.at(1))
            } else {
                debug(ctx, "wrong size of list `%v'", t, trace{})
            }
        default: // rename newname oldname  newname oldname ...
            if i+1 < nargs {
                oldname = __string(ctx, ctx.a[i+0])
                newname = __string(ctx, ctx.a[i+1])
                i += 1
            } else {
                debug(ctx, "Wrong arguments `%v'", ctx.a, trace{})
            }
        }
        if err := os.Rename(oldname, newname); err != nil {
            debug(ctx, "%v", err, trace{})
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
    var opts = ctx
    var remove func(Context, Value)
    var removeFile = func(ctx Context, f *file) {
        var err error
        var s = f.fullname()
        if opts.skip != "" {
            if strings.HasPrefix(s, opts.skip) { return } else
            if strings.HasPrefix(ident(ctx,f), opts.skip) { return }
        }
        if opts.all { err = os.RemoveAll(s) } else { err = os.Remove(s) }
        if err != nil {
            debug(ctx, _f("remove: %v", err), _f("remove: %v → %s", f, s), trace{})
            return
        }
        if d := opts.debug; d>0 { debug(ctx, "remove %s (%s)", f, s, callstack{num:d}) }
        if opts.verbose { prompt(ctx, "removed %s\n", f) }
    }
    var removePath = func(ctx Context, p *path) {
        var err error
        var s = __string(ctx, p)
        if opts.skip != "" {
            if strings.HasPrefix(s, opts.skip) { return }
        }
        if opts.all { err = os.RemoveAll(s) } else {
            debug(ctx, "remove path: %v", p, trace{})
            return
        }
        if err != nil {
            debug(ctx, _f("remove: %v", err), _f("remove: %v", p), trace{})
            return
        }
        if d := opts.debug; d>0 { debug(ctx, "remove %s", s, callstack{num:d}) }
        if opts.verbose { prompt(ctx, "removed %s\n", s) }
    }
    var removePat = func(ctx Context, pat Value) {
        // var val = (&__wildcard{__:__{evocation:?}})._do(pat)
        // debug(ctx, "TODO: remove: %v → %v", pat, val, trace{})
        debug(ctx, "TODO: remove: %v", ts(pat), trace{})
    }

    remove = func(ctx Context, v Value) {
        if _, y := v.(*none); y {
            return
        } else if isTrivial(v) {
            debug(ctx, "triviality: %v (%T)", v, v)
        } else if l, y := v.(*list); y {
            for _, v := range l.elems { remove(ctx, v) }
        } else if d, y := v.(*delegate); y {
            debug(ctx, "delegate: %v (%T, %v, %v)", d.x, d.x, d.o, d.a)
        } else if patterned(ctx,v) {
            removePat(ctx, v)
        } else if f, y := v.(*file); y {
            removeFile(ctx, f)
        } else if f = findfile(ctx, __string(ctx, v)); f != nil {
            removeFile(ctx, f)
        } else if p, y := v.(*path); y {
            removePath(ctx, p)
        } else if !opts.ignoreMissing {
            debug(ctx, "not file: %v (%T)", v, v, trace{})
        }
    }
    for _, a := range ctx.a {
        ctx := ctx.Context
        remove(ctx, expand(ctx, a))
    }

    if opts.debug > 0 { debug(ctx, "%v", ctx.a) }
    if opts.debug > 0 && flush(ctx) > 0 {
        debug(ctx, "remove errors", trace{})
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
    for i, nargs := 0, len(ctx.a); i < nargs; i += 1 {
        var (
            a = ctx.a[i]
            name string
            size int64
        )
        switch t := a.(type) {
        case *pair: // truncate name ⇒ size old ⇒ new
            name = __string(ctx, t.key)
            size = __int(ctx, t.val)
        case *group: // truncate (name size) (old new)
            if t.len() == 2 {
                name = __string(ctx, t.at(0))
                size = __int(ctx, t.at(1))
            } else {
                debug(ctx, "Wrong size of group `%v'", t, trace{})
                break
            }
        case *list: // truncate name size, old new, ...
            if t.len() == 2 {
                name = __string(ctx, t.at(0))
                size = __int(ctx, t.at(1))
            } else {
                debug(ctx, "Wrong size of list `%v'", t, trace{})
                break
            }
        default: // truncate name size  name size ...
            if i+1 < nargs {
                name = __string(ctx, ctx.a[i+0])
                size = __int(ctx, ctx.a[i+1])
                i += 1
            } else {
                debug(ctx, "Wrong arguments `%v'", ctx.a, trace{})
                break
            }
        }
        if err := os.Truncate(name, size); err != nil {
            debug(ctx, "%v", err, trace{})
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
    for i, nargs := 0, len(ctx.a); i < nargs; i += 1 {
		var (
			a = ctx.a[i]
			oldname, newname string
		)
        switch t := a.(type) {
        case *pair: // link oldname ⇒ newname old ⇒ new
            oldname = __string(ctx, t.key)
            newname = __string(ctx, t.val)
        case *group: // link (oldname newname) (old new)
            if t.len() == 2 {
                oldname = __string(ctx, t.at(0))
                newname = __string(ctx, t.at(1))
            } else {
                debug(ctx, "Wrong size of group `%v'", t, trace{})
                break
            }
        case *list: // link oldname newname, old new, ...
            if t.len() == 2 {
                oldname = __string(ctx, t.at(0))
                newname = __string(ctx, t.at(1))
            } else {
                debug(ctx, "Wrong size of list `%v'", t, trace{})
                break
            }
        default: // link oldname newname  oldname newname ...
            if i+1 < nargs {
                oldname = __string(ctx, ctx.a[i+0])
                newname = __string(ctx, ctx.a[i+1])
                i += 1
            } else {
                debug(ctx, "Wrong arguments `%v'", ctx.a, trace{})
                break
            }
        }
        if err := os.Link(oldname, newname); err != nil {
            debug(ctx, "%v", err, trace{})
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
            debug(ctx, "%v", e, trace{})
            return
        }

        var rel = !filepath.IsAbs(linkname)
        if rel {
            linkname = filepath.Join(linkpath, linkname)
            linkpath = filepath.Dir(linkname)
        }

        if d, e = os.Lstat(linkname); e != nil {
            prompt(ctx, "%s: lstat %s\n", fn, linkname)
            debug(ctx, "%v", e, trace{})
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
                debug(ctx, "expects two values for group", trace{})
                return
            } else {
                srcNameVal, dstNameVal = aa[0], aa[1]
            }
        case *list: // XXX: symlink old new, old new, ...
            if aa = parse_opts(ctx, &opts, t.elems...); len(aa) != 2 {
                debug(ctx, "expects two values for list", trace{})
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
                debug(ctx, "expects pair of names (%T %v)", t, t, trace{})
                return
            }
        }

        if srcDir, srcName = splitFileName(ctx, srcNameVal); srcName == "" {
            prompt(ctx, "symlink: args=%v\n", ctx.a)
            prompt(ctx, "symlink: src=%v\n", srcNameVal)
            debug(ctx, "empty src filename (%T)", srcNameVal, trace{})
            return
        }
        if dstDir, dstName = splitFileName(ctx, dstNameVal); dstName == "" {
            prompt(ctx, "symlink: args=%v\n", ctx.a)
            prompt(ctx, "symlink: dest=%v\n", dstNameVal)
            debug(ctx, "empty dest filename (%T)", dstNameVal, trace{})
            return
        }

        var src = srcName
        var dst = dstName
        if !filepath.IsAbs(src) { src = filepath.Join(srcDir, srcName) }
        if !filepath.IsAbs(dst) { dst = filepath.Join(dstDir, dstName) }
        if _, err := os.Stat(src); err != nil {
            prompt(ctx, "symlink: %v: %v\n", srcName, err)
            debug(ctx, "%v does not exist", srcName, trace{})
            return
        }

        if !opts.relative {/* no rel required */} else
        if s, e := filepath.Rel(filepath.Dir(dst), src); e != nil {
            prompt(ctx, "symlink: %s: rel(%s, %s)\n", dstName, dst, src)
            debug(ctx, "%v", e, trace{})
            return
        } else {
            if false {
                debug(ctx,
					_f("%v %v\t%s", srcDir, srcName, src),
					_f("%v %v\t%s", dstDir, dstName, dst),
					_f("%v", s))
            }
            src = s
        }

        if !opts.path {/* no mkdir */} else
        if dstDir == "" || dstDir == "." || dstDir == pathSep {
            // no need to mkdir: . or /
        } else if err := os.MkdirAll(dstDir, os.FileMode(0755)); err != nil {
            debug(ctx, "%v", err, trace{})
            return
        }

        var rm bool
        if rm = opts.force; rm {
            // overwrite...
        } else if s, e := os.Readlink(dst); e != nil {
            if false {
                prompt(ctx, "%v: readlink failed (%T)\n", dstName, e)
                debug(ctx, "%v", e, trace{})
            }
        } else if rm = s != src; !rm {
            continue outer
        }

        if rm { if e := os.Remove(dst); e != nil {
            prompt(ctx, "%v: remove old symlink failed (%T)\n", dstName, e)
            debug(ctx, "%v", e, trace{})
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

    var proj = _project(ctx)
    if proj == nil {
        debug(ctx, "unknown project", trace{})
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
        if s = __string(ctx, a); filepath.IsAbs(s) {
            f = _stat(ctx, s)
        } else {
            f = _stat(ctx, s, proj) // aka stat_dir{proj.absPath}
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
    var res []Value
    for _, a := range merge(ctx.a...) {
        if x, y := to_file(a); y {
            if !ctx.exists || x.exists() /* || x.stat(ctx) != nil */ {
                res = append(res, try_fullfile(ctx, x))
            } else if ctx.report {
                debug(ctx, "no such file {%v %v %v}", x.dir, x.sub, x.name)
            }
            continue
        }
		var proj = _project(ctx)
        for _, f := range select_files(ctx, unmap_files(ctx, proj, a, nil)) {
            if !ctx.exists || f.exists() {
                res = append(res, try_fullfile(ctx, f))
            } else if ctx.ignore {
                if ctx.verbose { debug(ctx, "%v → %v", tv(a), f) }
            } else if ctx.exists {
                debug(ctx, `not a file: %v : %s ; %s`, a, ts(a), ts(res), trace{})
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
    var cwd string // TODO: get current work directory
    var proj *project
    if proj = _project(ctx); proj == nil {
        debug(ctx, "unknown current cntext", trace{})
    }

    var res []Value
    for _, a := range ctx.a {
        var ( str string; names []string )
        if str = __string(ctx, a); !filepath.IsAbs(str) {
            str = filepath.Join(cwd, str)
        }

        var err error
        if names, err = filepath.Glob(str); err != nil {
            debug(ctx, "glob '%v' failed: %v", str, err, trace{})
        }
        for _, name := range names {
            // TODO: ctx.dir, ctx.file, ctx.symbol
            res = append(res, _pathStr(ctx, name))
        }
    }
    return res
}

func readDirNames(ctx Context, sd string, errorMissing bool) (names []string) {
    if f, err := os.Stat(sd); err != nil {
        if errorMissing {
            debug(ctx, "%v", err, trace{})
        }
        return
    } else if !f.IsDir() {
        debug(ctx, "not dir: %v", sd, trace{})
    } else if dir, err := os.Open(sd); err != nil {
        debug(ctx, "not dir: %v", sd, trace{})
    } else if names, err = dir.Readdirnames(-1); err != nil { // NOTE: see also filepath.Glob(...)
        if errorMissing { debug(ctx, "readdir: %v", err, trace{}) }
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
	sort bool `sort`
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
	var collect_file = func(f *file) {
		if ctx.sort {
			i, found := slices.BinarySearchFunc(files, f, func(a, b *file) (i int) {
				switch {
				case a.name < b.name : i = -1
				case a.name > b.name : i =  1
				}
				return
			})
			if !found { files = slices.Insert(files, i, f) }
		} else {
			files = append(files, f)
		}
	}
    var collect = func(name string) {
        var ne = ctx.includeMissing && !ctx.ignoreMissing
        var f = _stat(ctx, name, stat_dir{topDir}, stat_nonexist{ne})
        if true { assert(f != nil, "stat %s %s", name, topDir) }

        top.Lock()
        switch d := f.info.IsDir(); strings.ToLower(ctx.filetype) {
        case "f", "file": if!d { collect_file(f) }
        case "d", "dir" : if d { collect_file(f) }
        case "":                 collect_file(f)
        default:
            debug(ctx, "unknown -filetype: %s (%v)", ctx.filetype, f, trace{})
        }
        top.Unlock()
        top.Done()
    }
    var subcard = func(sub *subr, pat Value) {
        defer sub.Done()

        // if t, y := pat.(compositePattern); y { pat = t.Value }
        if t, y := pat.(*list); y {
            debug(ctx, "pattern is a list: %T %v %v", pat, pat, t.elems)
            if len(t.elems) == 1 { pat = t.elems[0] }
        }

        var ctx = ctx
        if p, y := pat.(*path); !y {
            // fallthrough
        } else if nElems := len(p.elems); nElems == 0 {
            debug(ctx, "empty path: %v", pat, trace{})
        } else if y, _, _ = match(ctx, p.elems[0], sub.n); y && nElems == 1 {
            debug(ctx, "%v %v: invalid path: %v, %v, %v", topDir, sub.dn, pat, sub.n, nElems, trace{})
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
            if y { debug(ctx, "%T %v %v", pat, pat, sub) }
            return
        }

        if gp, y := pat.(*globpat); !y {
            // fallthrough
        } else if len(gp.elems) == 0 {
            debug(ctx, "empty glob: %v (%s)", pat, sub.dn, trace{})
        } else if m, y := gp.elems[0].(*globmeta); !y {
            // fallthrough
        } else if m.token == DAST { // aka **
            y, _, _ = match(ctx, gp, sub.dn)
            if sub.isDir { subed(sub, pat) }
            if y { top.Add(1) ; go collect(sub.dn) ; return }
            return
        }

        y, _, _ := match(ctx, pat, sub.n)
        if y { top.Add(1) ; go collect(sub.dn) ; return }
        return
    }
    var subwork = func(subdir, name string, pats []Value) {
        defer top.Done()

        var sub = &subr{ d:subdir, n:name, dn:filepath.Join(subdir,name) }

        for _, x := range ctx.exclude {
            if y, _, _ := match(ctx, x, sub.dn); y { return }
        }

        if fi, err := os.Stat(filepath.Join(topDir, sub.dn)); err == nil {
            sub.isDir = fi.IsDir()
        } else {
            debug(ctx, _f("%p: %v %v → %v", sub, sub.d, sub.n, sub.dn),
				_f("%v", err), trace{})
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
        for _, a := range unmap_files(ctx, p, pat, nil) {
            for _, loc := range a.paths {
                var dir = __string(ctx, loc)
                debug(ctx, "%v %v %v %v", pat, a.pattern, loc, dir)
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
            for _, a := range unmap_files(ctx, p, pat, nil) {
                debug(ctx, "%v %v %v", pat, a.filemap.pattern, a.filemap.paths)
            }
        } ()
    }
    g.Wait()
    return
}
func (ctx *__wildcard) _project(p *project, pats ...Value) (files []*file) {
	if false { defer func(t0 time.Time) {
		if d := time.Now().Sub(t0); d > 1*time.Second {
			var pos = _position(ctx)
			prompt(ctx, "%v: slow: %d patterns, %v\n", pos, len(pats), pats)
			prompt(ctx, "%v: slow: %d files\n", pos, len(files))
			debug(ctx, "%v: slow: %v\n", pos, d)
		}
	}(time.Now())}

    var m sync.Mutex
    var g sync.WaitGroup
	var collect = func(t ...*file) {
		m.Lock()
		if ctx.sort {
			for _, f := range t {
				i, found := slices.BinarySearchFunc(files, f, func(a, b *file) (i int) {
					switch {
					case a.name < b.name : i = -1
					case a.name > b.name : i =  1
					}
					return
				})
				if !found { files = slices.Insert(files, i, f) }
			}
		} else {
			files = append(files, t...)
		}
		m.Unlock()
		g.Done()
	}

    var ne = ctx.includeMissing && !ctx.ignoreMissing
    var st = func(dir string, val Value) {
        if f := _stat(ctx, __string(ctx, val), stat_dir{dir}, stat_nonexist{ne}); f != nil {
            g.Add(1); go collect(f)
        }
    }

	var do_paths = func(isPat bool, pat Value, paths []string) { defer g.Done()
		for _, dir := range paths {
			if isPat {
				g.Add(1); go collect(ctx._directory(dir, pat)...)
			} else {
				st(dir, pat)
			}
		}
	}

    var f1 = func(lVal, rVal Value, fm *filemap) { defer g.Done()
		var paths []string
		for _, v := range merge(expands(_final(ctx), fm.paths...)...) {
			paths = append(paths, __string(ctx, v))
		}
		var lValIsPattern = patterned(ctx, lVal)
		var rValIsPattern = patterned(ctx, rVal)
		switch {
		case lValIsPattern:
			if rValIsPattern {
				ok1, _, _ := match(ctx, lVal, rVal)
				ok2, _, _ := match(ctx, rVal, lVal)
				switch {
				case ok1 && ok2:
					switch t := cmp(ctx, lVal, rVal); t {
					case cmpEqual  : g.Add(1); go do_paths(true, lVal, paths)
					case cmpSmaller: g.Add(1); go do_paths(true, lVal, paths)
					case cmpGreater: g.Add(1); go do_paths(true, rVal, paths)
					default:
						debug(ctx, "cmp(%v, %v) => %v", lVal, rVal, t, trace{})
					}
				case ok1 && !ok2:
					g.Add(1); go do_paths(true, rVal, paths)
				case !ok1 && ok2:
					g.Add(1); go do_paths(true, lVal, paths)
				case !ok1 && !ok2:
					debug(ctx, "%v %v", lVal, rVal, trace{})
				default:
					debug(ctx, "%v %v, %v %v", lVal, rVal, ok1, ok2, trace{})
				}
			} else {
				g.Add(1); go do_paths(false, rVal, paths)
			}
		default:
			if !rValIsPattern && !equal(ctx, lVal, rVal) {
				debug(ctx, "%v %v", lVal, rVal, trace{})
			}
			g.Add(1); go do_paths(false, lVal, paths)
		}
    }

    var f2 = func(pat Value, a filemap) { defer g.Done()
        for _, v := range a.patterns(ctx) { g.Add(1); go f1(pat, v, &a) }
    }

    var f3 = func(pat Value) { defer g.Done()
        for _, a := range unmap_files(ctx, p, pat, nil) { g.Add(1); go f2(pat, a.filemap) }
    }

    for _, pat := range pats { g.Add(1); go f3(pat) }

    g.Wait()

	if false && ctx.sort { slices.SortFunc(files, func(a, b *file) (i int) {
		switch {
		case a.name < b.name : i = -1
		case a.name > b.name : i =  1
		}
		return
	})}
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
    if len(ctx.exclude) > 0 {
        ctx.exclude = xmerge(_final(ctx.Context), ctx.exclude...)
    }

    var vals []Value
    for _, f := range ctx._do(merge(ctx.a...)...) {
        if f == nil {
            debug(ctx, "nil file: %v", ctx.a, trace{})
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
    var l []Value
    for _, a := range ctx.a {
        if fis, err := ioutil.ReadDir(__string(ctx, a)); err == nil {
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
    var l []Value
    var closured = closure_projects(ctx)
    for _, v := range ctx.a {
        if o := (as{v}.fullname(ctx, closured...)); o.Value == nil {
            debug(ctx, "%v is not a file", v, trace{})
        } else if s, e := ioutil.ReadFile(__string(ctx,o)); e != nil {
            debug(ctx, "read file failed: %v", e, trace{})
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
            name = __string(ctx, t.key)
            data = __string(ctx, t.val)
        case *group: // write-file (name text) (name text 0660)
            if n := t.len(); n < 4 && n > 0 {
                name = __string(ctx, t.at(0))
                if n > 1 { data = __string(ctx, t.at(1)) }
                if n > 2 { perm = filePerm(ctx, t.at(2),0600) }
            } else {
                debug(ctx, "Wrong size of group `%v'", t, trace{})
            }
        case *list: // write-file name text, name text 0660, ...
            if n := t.len(); n < 4 && n > 0 {
                name = __string(ctx, t.at(0))
                if n > 1 { data = __string(ctx, t.at(1)) }
                if n > 2 { perm = filePerm(ctx, t.at(2),0600) }
            } else {
                debug(ctx, "Wrong size of list `%v'", t, trace{})
            }
        default: // write-file name text 0660  name text 0660 ...
            name = __string(ctx, ctx.a[i])
            if i+1 < len(ctx.a) {
                data = __string(ctx, ctx.a[i+1])
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
                debug(ctx, "%v", err, trace{})
            }
        }
        if err := ioutil.WriteFile(name, []byte(data), perm); err != nil {
            debug(ctx, "%v", err, trace{})
        }
    }
    return
}

func touch(ctx Context, file Value, optMode uint32, optPath bool, ts ...time.Time) (err error) {
    var a, filename, c = as{file}.fullname_file(ctx)

    if filename == "" {
        debug(ctx, "touch: empty file name: %v (%v, %v, %v)", file, typeof(file), a, c, trace{})
    } else if d := filepath.Dir(filename); optPath && d != "." && d != pathSep {
        if err = os.MkdirAll(d, os.FileMode(optMode|0733)); err != nil {
            debug(ctx, "touch: %v", err, trace{})
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
            debug(ctx, "touch: %v", err, trace{})
        } else if err = f.Close(); err != nil {
            debug(ctx, "touch: %v", err, trace{})
        }
    }
    if err == nil {
        if err = os.Chtimes(filename, ta, tm); err != nil {
            debug(ctx, "touch: %v", err, trace{})
        }
    }
    if err == nil && mode != 0 && m != 0 && mode != m {
        if err = os.Chmod(filename, mode); err != nil {
            debug(ctx, "touch: %v", err, trace{})
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
    // $(touch-file filename)
    // $(touch-file -p filename)
    for i := 0; i < len(ctx.a); i += 1 {
        if err := touch(ctx, ctx.a[i], uint32(ctx.mode), ctx.path); err != nil {
            debug(ctx, "%v", err, trace{})
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
    var args = ctx.a
    var nargs = len(args)
    if !(nargs == 2 || nargs == 3) {
        debug(ctx, "wrong args, try $(grep {=regex '^example$'},$0,$(file))", trace{})
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
        } else if s := __string(ctx, a); s == "" {
            debug(ctx, "empty regexp", trace{})
            return
        } else if r, e := regexp.Compile(s); e != nil {
            debug(ctx, "%v", e, trace{})
            return
        } else {
            rxs = append(rxs, r)
        }
    }

    var p Position
    var res []Value
    for _, a := range merge(args...) {
        var c = pc(ctx, a)
        if x, y := a.(*file); y {
            p.Filename = x.fullname()
        } else {
            p.Filename = __string(ctx, a)
        }

        var e error
        var f *os.File
        if p.Filename == "" {
            debug(c, "empty filename: %v", ts(a), trace{})
            return
        } else if f, e = os.Open(p.Filename); e != nil {
            debug(c, "%s ; %s", e, ts(a), trace{})
            return
        } else {
            defer f.Close()
        }

        s := bufio.NewScanner(f)
        s.Split(bufio.ScanLines)
        p.Line, p.Column = 0, 0
        for s.Scan() {
            text := s.Text()
            p.Line += 1

            for _, rx := range rxs {
                // := rx.FindStringSubmatch(text)
                si := rx.FindStringSubmatchIndex(text)
                if si == nil { continue }

                var val Value

                ctx.defs = make(def_map) // ensure a clear defs map
                for i, n := range rx.SubexpNames() {
                    if n == "" { n = strconv.Itoa(i) }

                    var t string
                    var a, b = si[2*i], si[2*i+1]
                    if 0 <= a && 0 < b { p.Column, t = 1+a, text[a:b] }

                    var v = _raw(p, t)
                    ctx.set(pc(c,p), defVoid, n, v)

                    if i == 0 && result == nil { val = v } else
                    if 0 < i && a < 0 { p.Column += utf8.RuneCountInString(t) }
                }
                if result != nil { val = expand(_final(c), result) }
                res = append(res, val)

                if checkpoints { ctx.check(rx, text, result, val) }
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
        if d = p.resolveDef(ctx, name); d == nil {
            if true {
                prompt(ctx, "%v: %v undefined\n", pos, name)
                debug(ctx, "in %v", p)
            }
            continue
        } else if val = evoke(ctx, d, nil, nil); isNull(val) {
            if f := p.configuration_sm(ctx); f == nil {
                debug(ctx, "%v: configuration file not defined", name, f, trace{})
                return
            } else if !f.exists() {
                prompt(ctx, "%s: file not exists (for %v)\n", f.fullname(), name)
                debug(ctx, "%v: configuration file not exists, try -conf first", name, trace{})
                return
            }
            continue
        }

        switch t := val.(type) {
        case *undef, undef: // FIXME: fmt.Fprintf(res, "#undef")
        case *answer, *boolean:
            fmt.Fprintf(res, "%d", __int(ctx, t))
        case *group:
            fmt.Fprintf(res, "%s", __string(ctx, parseGroupValue(ctx, t)))
        case *plain:
            fmt.Fprintf(res, "%s", t.String())
        default:
            fmt.Fprintf(res, "%s", __string(ctx, val))
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
    debug(ctx, "TODO: %d", num)
    return
}

func configurestring(ctx Context, out *bytes.Buffer, p *project, str string) {
    if s, e := p.strExpandConfig(ctx, str); e != nil {
        debug(ctx, "%v : %v", str, e, trace{})
    } else {
        str = s
    }

    var index = 0

    for _, ii := range rxConfigure.FindAllStringSubmatchIndex(str, -1) {
        if _, e := out.WriteString(str[index:ii[0]]); e != nil {
            debug(ctx, "%v", e, trace{})
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
        if d = p.resolveDef(ctx, name); d != nil {
            if v := evoke(ctx, d, nil, nil); v == nil {
                // noop, TODO: or #undef?
            } else if _, t := v.(*undef); t {
                _, e := out.WriteString(fmt.Sprintf("#undef /* %s */", name))
                if e != nil {
                    debug(ctx, "%v", e, trace{})
                } else {
                    continue
                }
            } else {
                t = __true(ctx, v)
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
            if d == nil {
                s = fmt.Sprintf("#undef %s", name)
            } else if isNull(d.value) || isNone(d.value) {
                s = fmt.Sprintf("#undef %s /* %v */", name, d.value)
            } else if va := expand(ctx, d.value); va != nil {
                switch v := va.(type) {
                case *answer, *boolean:
                    if b := __true(ctx, v); b {
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
            debug(ctx, "%v", e, trace{})
        }
    }

    if len(str) <= index {
        return
    }

    if _, e := out.WriteString(str[index:]); e != nil {
        debug(ctx, "%v", e, trace{})
    }
    return
}

type __return struct { builtinbase }
func (ctx *__return) inner() Context { return &ctx.builtinbase }
func (ctx *__return) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.builtinbase.cast(t)
}
func (ctx *__return) x() (res any) {
    return &returner{valbase{_position(ctx)}, ctx.a}
}
