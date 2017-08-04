//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        "encoding/json"
        "strings"
        "io"
        //"fmt"
)

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
        )
        jd := json.NewDecoder(strings.NewReader(source))
loop:
        for {
                if t, err = jd.Token(); err != nil {
                        break
                }
                //fmt.Printf("%T: %v\n", t, t)
                x := len(stack)
        switch1:
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
                                break switch1
                        case '{':
                                nn := values.Group(values.Bareword(JsonObject))
                                if x == 0 {
                                        nodes = append(nodes, nn)
                                } else {
                                        node, value = stack[x-1], nn
                                }
                                stack = append(stack, nn) // APPEND
                                break switch1
                        case '}':
                                if x == 0 {
                                        err = ErrorIllJson; break loop
                                }
                                if k := stack[x-1].Get(0); k == nil || k.Strval() != JsonObject {
                                        err = ErrorIllJson; break loop
                                }
                                stack = stack[0:x-1] // POP
                                continue loop
                        case ']':
                                if x == 0 {
                                        err = ErrorIllJson; break loop
                                }
                                if k := stack[x-1].Get(0); k == nil || k.Strval() != JsonArray {
                                        err = ErrorIllJson; break loop
                                }
                                stack = stack[0:x-1] // POP
                                continue loop
                        default:
                                err = ErrorIllJson; break loop
                        }
                case string:
                        s := values.String(d)
                        if x == 0 {
                                nodes = append(nodes, s)
                                break
                        }
                        
                        node = stack[x-1]
                        if k := node.Get(0); k != nil {
                                if kind := k.Strval(); kind == JsonArray {
                                        node.Append(s); continue
                                } else if kind != JsonObject {
                                        err = ErrorIllJson; break loop
                                }
                        }

                        // Get value token 
                        if !jd.More() {
                                err = ErrorIllJson; break loop
                        } else if v, err = jd.Token(); err != nil {
                                break loop
                        }
                        
                        switch vd := v.(type) {
                        case json.Delim:
                                var vn *types.Group
                                switch vd {
                                case '[': vn = values.Group(values.Bareword(JsonArray))
                                case '{': vn = values.Group(values.Bareword(JsonObject))
                                default: err = ErrorIllJson; break loop
                                }
                                stack = append(stack, vn)
                                node.Append(values.Pair(s, vn))
                        case string:
                                node.Append(values.Pair(s, values.String(vd)))
                        case float64:
                                node.Append(values.Pair(s, values.Float(vd)))
                        case nil: // null
                                node.Append(values.Pair(s, values.Bareword("null")))
                        default:
                                err = ErrorIllJson; break loop
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
                        err = ErrorIllJson; break loop
                }
                if node != nil && value != nil {
                        if k := node.Get(0); k != nil && k.Strval() != JsonArray {
                                err = ErrorIllJson; break loop
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
        polyInterpreter
}

func (t *dialectJson) dialect() string { return "json" }
func (t *dialectJson) evaluate(prog *Program, args []types.Value, recipes []types.Value) (result types.Value, err error) {
        var source = joinRecipesString(recipes...)
        if result, err = DecodeJSON(source); err == nil {
                result = values.Group(targetJsonKind, result, values.None)
        } else {
                result = values.Group(targetJsonKind, 
                        values.None, values.String(err.Error()))
        }
        return
}
