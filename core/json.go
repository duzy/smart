//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package core

import (
        "encoding/json"
        "strings"
        "bytes"
        "fmt"
        "io"
)

type JSON struct {
        Value Value
}
func (p *JSON) disclose() (Value, error) { return nil, nil }
func (p *JSON) referencing(_ Object) bool { return false }
func (p *JSON) Type() Type { return JSONType }
func (p *JSON) String() string { return "(json " + p.Value.String() + ")" }
func (p *JSON) Strval() (string, error) { return p.Value.Strval() }
func (p *JSON) Integer() (int64, error) { return 0, nil }
func (p *JSON) Float() (float64, error) { return 0, nil }

func joinRecipesString(recipes... Value) (res string, err error) {
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
        stack []*Group
        nodes []*Group
}

func (ds *jsonDecodeState) decode() {
}

const (
        JsonArray  = "array"
        JsonObject = "object"
)

func DecodeJSON(source string) (result Value, err error) {
        //fmt.Printf("json: %v\n", source)
        var (
                stack []*Group
                nodes []Value
                node *Group
                value Value
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
                                nn := &Group{List{Elements{[]Value{&Bareword{JsonArray}}}}}
                                if x == 0 {
                                        nodes = append(nodes, nn)
                                } else {
                                        node, value = stack[x-1], nn
                                }
                                stack = append(stack, nn) // APPEND
                                break SwitchNodeType
                        case '{':
                                nn := &Group{List{Elements{[]Value{&Bareword{JsonObject}}}}}
                                if x == 0 {
                                        nodes = append(nodes, nn)
                                } else {
                                        node, value = stack[x-1], nn
                                }
                                stack = append(stack, nn) // APPEND
                                break SwitchNodeType
                        case '}':
                                if x == 0 {
                                        err = ErrorIllJson; break LoopJSON
                                }
                                if k := stack[x-1].Get(0); k == nil {
                                        if s, err = k.Strval(); err != nil { return } else if s != JsonObject {
                                                err = ErrorIllJson; break LoopJSON
                                        }
                                }
                                stack = stack[0:x-1] // POP
                                continue LoopJSON
                        case ']':
                                if x == 0 {
                                        err = ErrorIllJson; break LoopJSON
                                }
                                if k := stack[x-1].Get(0); k == nil {
                                        if s, err = k.Strval(); err != nil { return } else if s != JsonArray {
                                                err = ErrorIllJson; break LoopJSON
                                        }
                                }
                                stack = stack[0:x-1] // POP
                                continue LoopJSON
                        default:
                                err = ErrorIllJson; break LoopJSON
                        }
                case string:
                        var sv = &String{d}
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
                                        err = ErrorIllJson; break LoopJSON
                                }
                        }

                        // Get value token 
                        if !jd.More() {
                                err = ErrorIllJson; break LoopJSON
                        } else if v, err = jd.Token(); err != nil {
                                break LoopJSON
                        }
                        
                        switch vd := v.(type) {
                        case json.Delim:
                                var vn *Group
                                switch vd {
                                case '[': vn = &Group{List{Elements{[]Value{&Bareword{JsonArray}}}}}
                                case '{': vn = &Group{List{Elements{[]Value{&Bareword{JsonObject}}}}}
                                default: err = ErrorIllJson; break LoopJSON
                                }
                                stack = append(stack, vn)
                                node.Append(&Pair{sv, vn})
                        case string:
                                node.Append(&Pair{sv, &String{vd}})
                        case float64:
                                node.Append(&Pair{sv, &Float{vd}})
                        case nil: // null
                                node.Append(&Pair{sv, &Bareword{"null"}})
                        default:
                                err = ErrorIllJson; break LoopJSON
                        }
                        //fmt.Printf("node: %v\n", node)
                case float64:
                        if v := Value(&Float{d}); x == 0 {
                                nodes = append(nodes, v)
                        } else {
                                node, value = stack[x-1], v
                        }
                case nil: // null
                        if v := Value(&Bareword{"null"}); x == 0 {
                                nodes = append(nodes, v)
                        } else {
                                node, value = stack[x-1], v
                        }
                default:
                        err = ErrorIllJson; break LoopJSON
                }
                if node != nil && value != nil {
                        if k := node.Get(0); k != nil {
                                if s, err = k.Strval(); err != nil { return } else if s != JsonArray {
                                        err = ErrorIllJson; break LoopJSON
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
                g := &Group{}
                for _, v := range nodes {
                        g.Append(v)
                }
                result = g
        }
        return
}

type dialectJson struct {
}

func (t *dialectJson) Evaluate(prog *Program, args []Value, recipes []Value) (result Value, err error) {
        var source string
        if source, err = joinRecipesString(recipes...); err != nil { return }
        if result, err = DecodeJSON(source); err == nil {
                result = &JSON{ result }
        } else {
                result = &JSON{ UniversalNone }
        }
        return
}

func init() {
        RegisterDialect("json", new(dialectJson))
}
