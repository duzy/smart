//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        // TODO: csv "encoding/csv"
        xml_enc "encoding/xml"
        json_enc "encoding/json"
        // yaml_enc "encoding/yaml"
        // "strconv"
        "strings"
        "bytes"
        "fmt"
        "io"
)

// Value returned by (plain) modifier.
type Plain struct { raw ; name string }
func (p *Plain) String() (s string) {
        var value = strings.Replace(p.s, "'", "\\'", -1)
        if p.name == "" {
                s = fmt.Sprintf("(plain '%s')", value)
        } else {
                s = fmt.Sprintf("((plain %s) '%s')", p.name, value)
        }
        return
}
func (p *Plain) expand(_ Context) (val Value) { return /* &p.raw */p }
func (p *Plain) ident(_ Context) string { return p.name }
func (p *Plain) cmp(ctx Context, v Value) (res cmpres) {
        if a, y := v.(*Plain); y {
                if p.name == a.ident(ctx) && p.s == a.s {
                        res = cmpEqual
                }
        } else if v.string(ctx) == p.s {
                res = cmpEqual
        }
        return
}

type (
        plainInt struct {}
        plainOpts struct {
                generalOpts
        }
)
func (_ *plainInt) evaluate(ctx Context, args ...Value) (result Value) {
        var (
                program = _program(ctx)
                str, name string
                opts plainOpts
        )
        if args = parseOpts(ctx, &opts, args...); len(args) > 0 {
                name = args[0].string(ctx)
                program.language = name
        }

        str = multiline(ctx, program.recipes...)

        var pos Position
        if len(program.recipes) > 0 {
                pos = program.recipes[0].Position()
        } else {
                pos = _position(ctx)
        }

        str = strings.Replace(str, "\\\n\t", "\\\n", -1)
        result = &Plain{raw{valbase{pos}, str}, name}
        return
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
                tok xml_enc.Token
                err error
        )
        xd := xml_enc.NewDecoder(strings.NewReader(source))
        for tok, err = xd.Token(); err == nil; tok, err = xd.Token() {
                switch elem := tok.(type) {
                case xml_enc.ProcInst:
                        // TODO: ...
                case xml_enc.StartElement:
                        nn := makeGroup(pos, &bareword{valbase{pos},elem.Name.Local})
                        for _, a := range elem.Attr {
                                var k, v Value
                                k = &bareword{valbase{pos},a.Name.Local}
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
                case xml_enc.EndElement:
                        if x := len(stack); x > 0 {
                                stack = stack[0:x-1]
                        } else {
                                // FIXME: report illegal xml
                        }
                case xml_enc.CharData:
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
                case xml_enc.Directive:
                        // TODO: ...
                case xml_enc.Comment:
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
                erro(ctx, "%v", err).debug()
                trace(ctx)
        }
        return
}

type xml struct { whitespace bool }
func (p *xml) evaluate(ctx Context, args ...Value) (result Value) {
        var source = multiline(ctx, _program(ctx).recipes...)
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
        dec *json_enc.Decoder
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
                t, v json_enc.Token
                s string
                err error
        )
        jd := json_enc.NewDecoder(strings.NewReader(source))
LoopJSON:
        for {
                if t, err = jd.Token(); err != nil { break }
                x := len(stack)
                //prompt(ctx, "%T: %v\n", t, t)
        SwitchNodeType:
                switch node, value = nil, nil; d := t.(type) {
                case json_enc.Delim:
                        switch d {
                        case '[':
                                nn := makeGroup(pos, makeBareword(pos, JsonArray))
                                if x == 0 {
                                        nodes = append(nodes, nn)
                                } else {
                                        node, value = stack[x-1], nn
                                }
                                stack = append(stack, nn) // APPEND
                                break SwitchNodeType
                        case '{':
                                nn := makeGroup(pos, makeBareword(pos, JsonObject))
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
                        case json_enc.Delim:
                                var vn *group
                                switch vd {
                                case '[': vn = makeGroup(pos, makeBareword(pos, JsonArray))
                                case '{': vn = makeGroup(pos, makeBareword(pos, JsonObject))
                                default: err = errorIllJson; break LoopJSON
                                }
                                stack = append(stack, vn)
                                node.append(makePair(sv, vn))
                        case string:
                                node.append(makePair(sv, makeStrlit(pos, vd)))
                        case float64:
                                node.append(makePair(sv, makeFloat(pos, vd)))
                        case nil: // null
                                node.append(makePair(sv, makeBareword(pos, "null")))
                        default:
                                err = errorIllJson; break LoopJSON
                        }
                        //prompt(ctx, "node: %v\n", node)
                case float64:
                        if v := Value(makeFloat(pos, d)); x == 0 {
                                nodes = append(nodes, v)
                        } else {
                                node, value = stack[x-1], v
                        }
                case nil: // null
                        if v := Value(makeBareword(pos, "null")); x == 0 {
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
                erro(ctx, "%v", err).debug()
                trace(ctx)
        }
        return
}

type json struct {}

func (_ *json) evaluate(ctx Context, args ...Value) (result Value) {
        var program = _program(ctx)
        if program == nil {
                erro(ctx, `needs program context to evaluate: %v`, ctx).debug(16)
                return
        }
        var source = multiline(ctx, program.recipes...)
        if v := DecodeJSON(ctx, source); v != nil {
                return &JSON{ result }
        } else {
                return &JSON{ makeNone(program.position) }
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
        erro(ctx, "TODO: implement DecodeYAML").debug()
        trace(ctx)
        return
}

type yaml struct { whitespace bool }
func (p *yaml) evaluate(ctx Context, args ...Value) (result Value) {
        var source = multiline(ctx, _program(ctx).recipes...)
        if v := DecodeYAML(ctx, source, p.whitespace); v != nil {
                return &YAML{ result }
        } else {
                return &YAML{ makeNone(_position(ctx)) }
        }
}
