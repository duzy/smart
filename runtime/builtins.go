//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        //"github.com/duzy/smart/token"
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        "io/ioutil"
        "bytes"
        "encoding/base64"
        //"fmt"
)

type builtin func(ctx *Context, args... types.Value) (types.Value, error)

var (
        builtins = map[string]builtin {
                `return`:       builtinReturn,
                `lit`:          builtinLit,
                //`run`:          builtinRun,

                `read-file`:    builtinReadFile,

                `encode-base64`:  builtinEncodeBase64,
                `decode-base64`:  builtinDecodeBase64,

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
        }
)

func GetBuiltinNames() (a []string) {
        for s, _ := range builtins {
                a = append(a, s)
        }
        return
}

type returner struct {
        value types.Value
}

func (p *returner) Error() string {
        return "evaluation return"
}

func builtinReturn(ctx *Context, args... types.Value) (result types.Value, err error) {
        var value types.Value
        if x := len(args); x == 0 {
                value = args[x]
        } else {
                value = values.List(args...)
        }
        return nil, &returner{ value }
}

func builtinLit(ctx *Context, args... types.Value) (result types.Value, err error) {
        var s string
        for _, a := range args {
                s += a.Lit()
        }
        return values.String(s), nil
}

/*
func builtinRun(ctx *Context, args... types.Value) (result types.Value, err error) {
        if len(args) > 0 {
                var (
                        err error
                        name = args[0]
                        //rest = args[1:]
                        m = ctx.CurrentModule()
                        entry = m.Lookup(name.String())
                )
                if entry != nil {
                        if result, err = entry.Call(args...); err != nil {
                                //...
                        }
                }
        }
        return
} */

func builtinReadFile(ctx *Context, args... types.Value) (res types.Value, err error) {
        var l []types.Value
        for _, a := range args {
                var s []byte
                if s, err = ioutil.ReadFile(a.String()); err == nil {
                        l = append(l, values.String(string(s)))
                } else {
                        l = append(l, values.None)
                }
        }
        if x := len(l); x == 0 {
                res = values.None
        } else if x == 1 {
                res = l[0]
        } else {
                res = values.List(l...)
        }
        return
}

func builtinEncodeBase64(ctx *Context, args... types.Value) (res types.Value, err error) {
        if len(args) > 0 {
                buf := new(bytes.Buffer)
                enc := base64.NewEncoder(base64.StdEncoding, buf)
                for _, a := range args {
                        enc.Write([]byte(a.String()))
                }
                enc.Close()
                res = values.String(buf.String())
        }
        return
}

func builtinDecodeBase64(ctx *Context, args... types.Value) (res types.Value, err error) {
        if len(args) > 0 {
                var list []types.Value
                for _, a := range args {
                        var dat []byte
                        dat, err = base64.StdEncoding.DecodeString(a.String())
                        if err == nil {
                                list = append(list, values.String(string(dat)))
                        } else {
                                return
                        }
                }
                if x := len(list); x == 0 {
                        res = values.None
                } else if x == 1 {
                        res = list[0]
                } else {
                        res = values.List(list...)
                }
        }
        return
}
