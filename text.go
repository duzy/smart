//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
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
    "strings"
    "bytes"
    "fmt"
    "io"
)

// Value returned by (plain) modifier.
type plain struct { elements ; name string }
func (_ *plain) kind() Kind { return KindPlain }
func (p *plain) hash(ctx Context) uint64 { return fnv1(ctx, p, p.any()...) }
func (p *plain) String() (s string) {
    s = "{="+typeof(p)
    if t := p.name; t != "" { s += "("+t+")" }
    for _, v := range p.elems { s += " " + v.String() }
    s += "}"
    return
}
func (p *plain) ts(t string) (s string) {
    s = "{="+t
    if t := p.name; t != "" { s += "("+t+")" }
    for _, v := range p.elems { s += " " + ts(v) }
    s += "}"
    return
}
func (p *plain) string(ctx Context) (s string) {
    for i, v := range p.elems {
        if i > 0 { s += " " }
        s += v.string(ctx)
    }
    return
}
func (p *plain) float(ctx Context) (_ float64) {
    if p.len() > 0 { return p.elems[0].float(ctx) }
    return
}
func (p *plain) int(ctx Context) (_ int64) {
    if p.len() > 0 { return p.elems[0].int(ctx) }
    return
}
func (p *plain) true(ctx Context) (_ bool) {
    for _, v := range p.elems {
        if v.true(ctx) { return true }
    }
    return
}
func (p *plain) match(ctx Context, i any) (bool, any, []string) {
    return stringMatch(ctx, p, i)
}
func (p *plain) stencil(ctx Context, stems []string) (Value, []string) {
    return p, stems
}
func (p *plain) expand(ctx Context) (_ Value) {
    if elems := expand(ctx, p.elems...); diff(ctx, elems, p.elems) {
        return &plain{elements{elems}, p.name}
    }
    return p
}
func (p *plain) cmp(ctx Context, v Value) (_ cmpres) {
    if x, y := v.(*plain); y {
        if p.name == x.name {
            return compareElems(ctx, p.elems, x.elems)
        }
    } else if v.string(ctx) == p.string(ctx) {
        return cmpEqual
    }
    return
}

type plainline struct { elements }
func (_ *plainline) kind() Kind { return KindPlainLine }
func (p *plainline) hash(ctx Context) uint64 { return fnv1(ctx, p, p.any()...) }
func (p *plainline) String() (s string) {
    s = "{="+typeof(p)
    for _, v := range p.elems { s += " " + v.String() }
    s += "}"
    return
}
func (p *plainline) string(ctx Context) (s string) {
    for i, v := range p.elems {
        if i > 0 { s += " " }
        s += v.string(ctx)
    }
    return
}
func (p *plainline) float(ctx Context) (_ float64) {
    if p.len() > 0 { return p.elems[0].float(ctx) }
    return
}
func (p *plainline) int(ctx Context) (_ int64) {
    if p.len() > 0 { return p.elems[0].int(ctx) }
    return
}
func (p *plainline) true(ctx Context) (_ bool) {
    for _, v := range p.elems {
        if v.true(ctx) { return true }
    }
    return
}
func (p *plainline) match(ctx Context, i any) (bool, any, []string) {
    return stringMatch(ctx, p, i)
}
func (p *plainline) stencil(ctx Context, stems []string) (Value, []string) {
    return p, stems
}
func (p *plainline) expand(ctx Context) (_ Value) {
    if elems := expand(ctx, p.elems...); diff(ctx, elems, p.elems) {
        return &plainline{elements{elems}}
    }
    return p
}
func (p *plainline) cmp(ctx Context, v Value) (_ cmpres) {
    if x, y := v.(*plainline); y {
        return compareElems(ctx, p.elems, x.elems)
    } else if v.string(ctx) == p.string(ctx) {
        return cmpEqual
    }
    return
}

type plainint struct{}
func (_ *plainint) evaluate(ctx Context, args ...Value) (_ Value) {
    var p = &plain{}
    var exe = _execution(ctx)
    var opts struct{ generalOpts }

    if args = parseOpts(ctx, &opts, args...) ; len(args) > 0 {
        p.name = args[0].string(ctx)
        exe.language = p.name
    }

    for _, recipe := range exe.recipes {
        p.elems = append(p.elems, recipe.expand(ctx))
    }

    if len(p.elems) == 1 {
        if x, y := p.elems[0].(*plainline); y {
            p.elems = merge(x.elems...)
        }
    }
    return p
}

func multiline(ctx Context, recipes... Value) (res string) {
    var (
        x = len(recipes)-1
        w = new(bytes.Buffer)
    )
    for n, recipe := range recipes {
        if fmt.Fprint(w, recipe.string(ctx)); n < x { fmt.Fprint(w, "\n") }
    }
    res = w.String()
    return
}

type XML struct { Value }
func (p *XML) String() string { return "(xml " + p.Value.String() + ")" }
func (p *XML) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*XML); ok {
        assert(ok, "value is not XML")
        res = p.Value.cmp(ctx, a.Value)
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
            nn := makeGroup(pos, &word{valbase{pos},elem.Name.Local})
            for _, a := range elem.Attr {
                var k, v Value
                k = &word{valbase{pos},a.Name.Local}
                v = &strlit{valbase{pos},a.Value}
                if s := a.Name.Space; s != "" {
                    k = makeGroup(pos, &strlit{valbase{pos},s}, k)
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
        g := makeGroup(pos)
        for _, node := range nodes {
            g.append(node)
        }
        result = g
    } else if x == 1 {
        result = nodes[0]
    }
    if err != io.EOF {
        erro(ctx, "%v", err).trace()
    }
    return
}

type xml struct { whitespace bool }
func (p *xml) evaluate(ctx Context, args ...Value) (result Value) {
    var source = multiline(ctx, _execution(ctx).recipes...)
    if v := DecodeXML(ctx, source, p.whitespace); v != nil {
        return &XML{ v }
    } else {
        return &XML{ makeNone(_position(ctx)) }
    }
}

type JSON struct { Value }
func (p *JSON) String() string { return "(json " + p.Value.String() + ")" }
func (p *JSON) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*JSON); ok {
        assert(ok, "value is not JSON")
        res = p.Value.cmp(ctx, a.Value)
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
                nn := makeGroup(pos, makeWord(pos, JsonArray))
                if x == 0 {
                    nodes = append(nodes, nn)
                } else {
                    node, value = stack[x-1], nn
                }
                stack = append(stack, nn) // APPEND
                break SwitchNodeType
            case '{':
                nn := makeGroup(pos, makeWord(pos, JsonObject))
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
                    if s = k.string(ctx); s != JsonObject {
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
                    if s = k.string(ctx); s != JsonArray {
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
                if kind = k.string(ctx); kind == JsonArray {
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
                case '[': vn = makeGroup(pos, makeWord(pos, JsonArray))
                case '{': vn = makeGroup(pos, makeWord(pos, JsonObject))
                default: err = errorIllJson; break LoopJSON
                }
                stack = append(stack, vn)
                node.append(makePair(sv, vn))
            case string:
                node.append(makePair(sv, _strlit(pos, vd)))
            case float64:
                node.append(makePair(sv, makefloat(pos, vd)))
            case nil: // null
                node.append(makePair(sv, makeWord(pos, "null")))
            default:
                err = errorIllJson; break LoopJSON
            }
            //prompt(ctx, "node: %v\n", node)
        case float64:
            if v := Value(makefloat(pos, d)); x == 0 {
                nodes = append(nodes, v)
            } else {
                node, value = stack[x-1], v
            }
        case nil: // null
            if v := Value(makeWord(pos, "null")); x == 0 {
                nodes = append(nodes, v)
            } else {
                node, value = stack[x-1], v
            }
        default:
            err = errorIllJson; break LoopJSON
        }
        if node != nil && value != nil {
            if k := node.at(0); k != nil {
                if s = k.string(ctx); s != JsonArray {
                    err = errorIllJson; break LoopJSON
                }
            }
            node.append(value)
        }
    }
    if x := len(nodes); x == 1 {
        result = nodes[0]
    } else {
        g := makeGroup(pos)
        for _, v := range nodes {
            g.append(v)
        }
        result = g
    }
    if err != io.EOF {
        erro(ctx, "%v", err).trace()
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
        return &JSON{ makeNone(recipes[0].Position()) }
    }
}

type YAML struct { Value }
func (p *YAML) String() string { return "(yaml " + p.Value.String() + ")" }
func (p *YAML) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*YAML); ok {
        assert(ok, "value is not YAML")
        res = p.Value.cmp(ctx, a.Value)
    }
    return
}

/*
   TODO: implement the yaml format:

   yaml{books(number=3
       book(id=1 title('book one'))
       book(id=2 title('book two'))
       book(id=3 title('   abc   '))
   )}
 */
func DecodeYAML(ctx Context, source string, ws bool) (result Value) {
    erro(ctx, "TODO: implement DecodeYAML").trace()
    return
}

type yaml struct { whitespace bool }
func (p *yaml) evaluate(ctx Context, args ...Value) (result Value) {
    var source = multiline(ctx, _execution(ctx).recipes...)
    if v := DecodeYAML(ctx, source, p.whitespace); v != nil {
        return &YAML{ result }
    } else {
        return &YAML{ makeNone(_position(ctx)) }
    }
}
