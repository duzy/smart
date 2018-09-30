//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "extbit.io/smart/types"
        "extbit.io/smart/values"
        "encoding/json"
        "strings"
        "bytes"
        "fmt"
        "io"
)

func joinRecipesString(recipes... types.Value) (res string, err error) {
        var (
                x = len(recipes)-1
                s = new(bytes.Buffer)
                r string
        )
        for n, recipe := range recipes {
                if r, err = recipe.Strval(); err != nil { return }
                if fmt.Fprint(s, r); n < x {
                        fmt.Fprint(s, "\n")
                }
        }
        res = s.String()
        return
}

type jsonDecodeState struct {
        dec *json.Decoder
        stack []*types.Group
        nodes []*types.Group
}

func (ds *jsonDecodeState) decode() {
}

const (
        JsonArray  = "array"
        JsonObject = "object"
)

func DecodeJSON(source string) (result types.Value, err error) {
        //fmt.Printf("json: %v\n", source)
        var (
                stack []*types.Group
                nodes []types.Value
                node *types.Group
                value types.Value
                t, v json.Token
                s string
        )
        jd := json.NewDecoder(strings.NewReader(source))
        LoopJSON: for {
                if t, err = jd.Token(); err != nil { break }
                x := len(stack)
                //fmt.Printf("%T: %v\n", t, t)
        SwitchNodeType:
                switch node, value = nil, nil; d := t.(type) {
                case json.Delim:
                        switch d {
                        case '[':
                                nn := values.Group(values.Bareword(JsonArray))
                                if x == 0 {
                                        nodes = append(nodes, nn)
                                } else {
                                        node, value = stack[x-1], nn
                                }
                                stack = append(stack, nn) // APPEND
                                break SwitchNodeType
                        case '{':
                                nn := values.Group(values.Bareword(JsonObject))
                                if x == 0 {
                                        nodes = append(nodes, nn)
                                } else {
                                        node, value = stack[x-1], nn
                                }
                                stack = append(stack, nn) // APPEND
                                break SwitchNodeType
                        case '}':
                                if x == 0 {
                                        err = types.ErrorIllJson; break LoopJSON
                                }
                                if k := stack[x-1].Get(0); k == nil {
                                        if s, err = k.Strval(); err != nil { return } else if s != JsonObject {
                                                err = types.ErrorIllJson; break LoopJSON
                                        }
                                }
                                stack = stack[0:x-1] // POP
                                continue LoopJSON
                        case ']':
                                if x == 0 {
                                        err = types.ErrorIllJson; break LoopJSON
                                }
                                if k := stack[x-1].Get(0); k == nil {
                                        if s, err = k.Strval(); err != nil { return } else if s != JsonArray {
                                                err = types.ErrorIllJson; break LoopJSON
                                        }
                                }
                                stack = stack[0:x-1] // POP
                                continue LoopJSON
                        default:
                                err = types.ErrorIllJson; break LoopJSON
                        }
                case string:
                        var sv = values.String(d)
                        if x == 0 {
                                nodes = append(nodes, sv)
                                break
                        }
                        
                        node = stack[x-1]
                        if k := node.Get(0); k != nil {
                                var kind string
                                if kind, err = k.Strval(); err != nil { return } else if kind == JsonArray {
                                        node.Append(sv); continue
                                } else if kind != JsonObject {
                                        err = types.ErrorIllJson; break LoopJSON
                                }
                        }

                        // Get value token 
                        if !jd.More() {
                                err = types.ErrorIllJson; break LoopJSON
                        } else if v, err = jd.Token(); err != nil {
                                break LoopJSON
                        }
                        
                        switch vd := v.(type) {
                        case json.Delim:
                                var vn *types.Group
                                switch vd {
                                case '[': vn = values.Group(values.Bareword(JsonArray))
                                case '{': vn = values.Group(values.Bareword(JsonObject))
                                default: err = types.ErrorIllJson; break LoopJSON
                                }
                                stack = append(stack, vn)
                                node.Append(values.Pair(sv, vn))
                        case string:
                                node.Append(values.Pair(sv, values.String(vd)))
                        case float64:
                                node.Append(values.Pair(sv, values.Float(vd)))
                        case nil: // null
                                node.Append(values.Pair(sv, values.Bareword("null")))
                        default:
                                err = types.ErrorIllJson; break LoopJSON
                        }
                        //fmt.Printf("node: %v\n", node)
                case float64:
                        if v := values.Float(d); x == 0 {
                                nodes = append(nodes, v)
                        } else {
                                node, value = stack[x-1], v
                        }
                case nil: // null
                        if v := values.Bareword("null"); x == 0 {
                                nodes = append(nodes, v)
                        } else {
                                node, value = stack[x-1], v
                        }
                default:
                        err = types.ErrorIllJson; break LoopJSON
                }
                if node != nil && value != nil {
                        if k := node.Get(0); k != nil {
                                if s, err = k.Strval(); err != nil { return } else if s != JsonArray {
                                        err = types.ErrorIllJson; break LoopJSON
                                }
                        }
                        node.Append(value)
                }
        }
        if err == io.EOF { 
                err = nil
                // TODO: check completeness
        }
        if x := len(nodes); x == 1 {
                result = nodes[0]
        } else {
                g := values.Group()
                for _, v := range nodes {
                        g.Append(v)
                }
                result = g
        }
        return
}

type dialectJson struct {
}

func (t *dialectJson) Evaluate(prog *types.Program, args []types.Value, recipes []types.Value) (result types.Value, err error) {
        var source string
        if source, err = joinRecipesString(recipes...); err != nil { return }
        if result, err = DecodeJSON(source); err == nil {
                result = &types.JSON{ result }
        } else {
                result = &types.JSON{ values.None }
        }
        return
}

func init() {
        types.RegisterDialect("json", new(dialectJson))
}
