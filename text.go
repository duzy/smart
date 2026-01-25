//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    // enc_csv "encoding/csv"
    enc_json "encoding/json"
    enc_xml "encoding/xml"
    // enc_yaml "encoding/yaml"
    // "strconv"
    "reflect"
    "strings"
    "bytes"
    "fmt"
    "io"
)

type is_plain_only struct{}
type plain_ctx struct { Context ; only bool }
func (p plain_ctx) inner() Context { return p.Context }
func (p plain_ctx) cast(t reflect.Type) Context { return icast(p, t) }
func (p plain_ctx) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case is_plain_only: return p.only
    }
    return p.Context.do(ctx, op)
}

// Value returned by (plain) modifier.
type plain struct { elements ; name string }
func (_ *plain) kind() Kind { return KindPlain }
func (p *plain) ts(t string) (s string) {
    s = "{="+t
    if t := p.name; t != "" { s += "("+t+")" }
    for _, v := range p.elems { s += " " + ts(v) }
    s += "}"
    return
}
func (p *plain) String() (s string) {
    s = "{=plain"
    if t := p.name; t != "" { s += "("+t+")" }
    for _, v := range p.elems { s += " " + v.String() }
    s += "}"
    return
}

type is_plainline struct {}
type plainline_ctx struct { Context }
func (p plainline_ctx) inner() Context { return p.Context }
func (p plainline_ctx) cast(t reflect.Type) Context { return icast(p, t) }
func (p plainline_ctx) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case is_plainline: return true
    }
    return p.Context.do(ctx, op)
}

type plainline struct { elements }
func (_ *plainline) kind() Kind { return KindPlainLine }
func (p *plainline) ts(t string) (s string) {
    s = "{="+t
    for _, v := range p.elems { s += " " + ts(v) }
    s += "}"
    return
}
func (p *plainline) String() (s string) {
    s = "{=plainline"
    if p.elems != nil {
        s += " "
        for _, v := range p.elems { s += v.String() }
    }
    s += "}"
    return
}
func (p *plainline) float(ctx Context) (_ float64) {
    if p.len() > 0 { return __float(ctx, p.elems[0]) }
    return
}
func (p *plainline) int(ctx Context) (_ int64) {
    if p.len() > 0 { return __int(ctx, p.elems[0]) }
    return
}

type plainint struct{}
func (p *plainint) evaluate(ctx Context, args ...Value) (_ Value) {
    var res = &plain{}
    var exe = _execution(ctx)
    var opts struct { general_opts }

    if args = parse_opts(ctx, &opts, args...) ; len(args) > 0 {
        res.name = __string(ctx, args[0])
        exe.language = res.name
    }

    for _, recipe := range exe.recipes {
        res.elems = append(res.elems, expand(_final(ctx), recipe))
    }

    if false && len(res.elems) == 1 {
        if x, y := res.elems[0].(*plainline); y {
            res.elems = merge(x.elems...)
        }
    }

    if checkpoints {
        p.evaluate_check(ctx, args, exe.recipes, res)
    }
    return res
}

func multiline(ctx Context, recipes... Value) (res string) {
    var (
        x = len(recipes)-1
        w = new(bytes.Buffer)
    )
    for n, recipe := range recipes {
        if fmt.Fprint(w, __string(ctx, recipe)); n < x { fmt.Fprint(w, "\n") }
    }
    res = w.String()
    return
}

type XML struct { Value }
func (p *XML) String() string { return "(xml " + p.Value.String() + ")" }
func (p *XML) _cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*XML); ok {
        assert(ok, "value is not XML")
        res = cmp(ctx, p.Value, a.Value)
    }
    return
}

/*
<books number="3">
  <book id="1">
    <title>book one</title>
  </book>
  <book id="2">
    <title>book two</title>
  </book>
  <book id="3"> <title>  abc  </title> </book>
</books>

Converted into:

(
    books number=3
    (
        book id=1
        (title 'book one')
    )
    (
        book id=2
        (title 'book two')
    )
    (
        book id=3
        (title '  abc  ')
    )
)

   TODO: implement the new xml format:

   xml{books(number=3
       book(id=1 title('book one'))
       book(id=2 title('book two'))
       book(id=3 title('   abc   '))
   )}

*/
func DecodeXML(ctx Context, source string, ws bool) (result Value) {
    var (
        pos = _position(ctx)
        stack []*group
        nodes []*group
        tok enc_xml.Token
        err error
    )
    xd := enc_xml.NewDecoder(strings.NewReader(source))
    for tok, err = xd.Token(); err == nil; tok, err = xd.Token() {
        switch elem := tok.(type) {
        case enc_xml.ProcInst:
            // TODO: ...
        case enc_xml.StartElement:
            nn := _group(pos, &word{valbase{pos},elem.Name.Local})
            for _, a := range elem.Attr {
                var k, v Value
                k = &word{valbase{pos},a.Name.Local}
                v = &strlit{valbase{pos},a.Value}
                if s := a.Name.Space; s != "" {
                    k = _group(pos, &strlit{valbase{pos},s}, k)
                }
                nn.append(makePair(k, v))
            }
            if x := len(stack); x > 0 {
                stack[x-1].append(nn)
            } else {
                nodes = append(nodes, nn)
            }
            stack = append(stack, nn)
        case enc_xml.EndElement:
            if x := len(stack); x > 0 {
                stack = stack[0:x-1]
            } else {
                // FIXME: report illegal xml
            }
        case enc_xml.CharData:
            if x := len(stack); x > 0 {
                node, s := stack[x-1], string(elem)
                if ws {
                    node.append(&strlit{valbase{pos},s})
                } else {
                    if s = strings.TrimSpace(s); s != "" {
                        node.append(&strlit{valbase{pos},s})
                    }
                }
            }
        case enc_xml.Directive:
            // TODO: ...
        case enc_xml.Comment:
            // TODO: ...
        }
    }
    if x := len(nodes); x > 1 {
        g := _group(pos)
        for _, node := range nodes {
            g.append(node)
        }
        result = g
    } else if x == 1 {
        result = nodes[0]
    }
    if err != io.EOF {
        debug(ctx, "%v", err, trace{})
    }
    return
}

type xml struct { whitespace bool }
func (p *xml) evaluate(ctx Context, args ...Value) (result Value) {
    var source = multiline(ctx, _execution(ctx).recipes...)
    if v := DecodeXML(ctx, source, p.whitespace); v != nil {
        return &XML{ v }
    } else {
        return &XML{ _none(_position(ctx)) }
    }
}

type JSON struct { Value }
func (p *JSON) String() string { return "(json " + p.Value.String() + ")" }
func (p *JSON) _cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*JSON); ok {
        assert(ok, "value is not JSON")
        res = cmp(ctx, p.Value, a.Value)
    }
    return
}

type jsonDecodeState struct {
    dec *enc_json.Decoder
    stack []*group
    nodes []*group
}
func (ds *jsonDecodeState) decode() {}

const (
    JsonArray  = "array"
    JsonObject = "object"
)

/*
   TODO: implement the new json format:

   json{books(number=3
       book(id=1 title('book one'))
       book(id=2 title('book two'))
       book(id=3 title('   abc   '))
   )}
 */
func DecodeJSON(ctx Context, source string) (result Value) {
    var (
        pos Position = _position(ctx)
        stack []*group
        nodes []Value
        node *group
        value Value
        t, v enc_json.Token
        s string
        err error
    )
    jd := enc_json.NewDecoder(strings.NewReader(source))
LoopJSON:
    for {
        if t, err = jd.Token(); err != nil { break }
        x := len(stack)
        //prompt(ctx, "%T: %v\n", t, t)
    SwitchNodeType:
        switch node, value = nil, nil; d := t.(type) {
        case enc_json.Delim:
            switch d {
            case '[':
                nn := _group(pos, _word(pos, JsonArray))
                if x == 0 {
                    nodes = append(nodes, nn)
                } else {
                    node, value = stack[x-1], nn
                }
                stack = append(stack, nn) // APPEND
                break SwitchNodeType
            case '{':
                nn := _group(pos, _word(pos, JsonObject))
                if x == 0 {
                    nodes = append(nodes, nn)
                } else {
                    node, value = stack[x-1], nn
                }
                stack = append(stack, nn) // APPEND
                break SwitchNodeType
            case '}':
                if x == 0 {
                    err = errorIllJson; break LoopJSON
                }
                if k := stack[x-1].at(0); k == nil {
                    if s = __string(ctx, k); s != JsonObject {
                        err = errorIllJson; break LoopJSON
                    }
                }
                stack = stack[0:x-1] // POP
                continue LoopJSON
            case ']':
                if x == 0 {
                    err = errorIllJson; break LoopJSON
                }
                if k := stack[x-1].at(0); k == nil {
                    if s = __string(ctx, k); s != JsonArray {
                        err = errorIllJson; break LoopJSON
                    }
                }
                stack = stack[0:x-1] // POP
                continue LoopJSON
            default:
                err = errorIllJson; break LoopJSON
            }
        case string:
            var sv = &strlit{valbase{pos},d}
            if x == 0 {
                nodes = append(nodes, sv)
                break
            }

            node = stack[x-1]
            if k := node.at(0); k != nil {
                var kind string
                if kind = __string(ctx, k); kind == JsonArray {
                    node.append(sv); continue
                } else if kind != JsonObject {
                    err = errorIllJson; break LoopJSON
                }
            }

            // Get value token
            if !jd.More() {
                err = errorIllJson; break LoopJSON
            } else if v, err = jd.Token(); err != nil {
                break LoopJSON
            }

            switch vd := v.(type) {
            case enc_json.Delim:
                var vn *group
                switch vd {
                case '[': vn = _group(pos, _word(pos, JsonArray))
                case '{': vn = _group(pos, _word(pos, JsonObject))
                default: err = errorIllJson; break LoopJSON
                }
                stack = append(stack, vn)
                node.append(makePair(sv, vn))
            case string:
                node.append(makePair(sv, _strlit(pos, vd)))
            case float64:
                node.append(makePair(sv, _float(pos, vd)))
            case nil: // null
                node.append(makePair(sv, _word(pos, "null")))
            default:
                err = errorIllJson; break LoopJSON
            }
            //prompt(ctx, "node: %v\n", node)
        case float64:
            if v := Value(_float(pos, d)); x == 0 {
                nodes = append(nodes, v)
            } else {
                node, value = stack[x-1], v
            }
        case nil: // null
            if v := Value(_word(pos, "null")); x == 0 {
                nodes = append(nodes, v)
            } else {
                node, value = stack[x-1], v
            }
        default:
            err = errorIllJson; break LoopJSON
        }
        if node != nil && value != nil {
            if k := node.at(0); k != nil {
                if s = __string(ctx, k); s != JsonArray {
                    err = errorIllJson; break LoopJSON
                }
            }
            node.append(value)
        }
    }
    if x := len(nodes); x == 1 {
        result = nodes[0]
    } else {
        g := _group(pos)
        for _, v := range nodes {
            g.append(v)
        }
        result = g
    }
    if err != io.EOF {
        debug(ctx, "%v", err, trace{})
    }
    return
}

type json struct {}
func (_ *json) evaluate(ctx Context, args ...Value) (result Value) {
    var recipes = _execution(ctx).recipes
    var source = multiline(ctx, recipes...)
    if v := DecodeJSON(ctx, source); v != nil {
        return &JSON{ result }
    } else {
        return &JSON{ _none(recipes[0].Position()) }
    }
}

type YAML struct { Value }
func (p *YAML) String() string { return "(yaml " + p.Value.String() + ")" }

/*
   TODO: implement the yaml format:

   yaml{books(number=3
       book(id=1 title('book one'))
       book(id=2 title('book two'))
       book(id=3 title('   abc   '))
   )}
 */
func DecodeYAML(ctx Context, source string, ws bool) (result Value) {
    debug(ctx, "TODO: implement DecodeYAML", trace{})
    return
}

type yaml struct { whitespace bool }
func (p *yaml) evaluate(ctx Context, args ...Value) (result Value) {
    var source = multiline(ctx, _execution(ctx).recipes...)
    if v := DecodeYAML(ctx, source, p.whitespace); v != nil {
        return &YAML{ result }
    } else {
        return &YAML{ _none(_position(ctx)) }
    }
}
