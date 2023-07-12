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
type Plain struct {
        Raw
        name string
}
func (p *Plain) Name(_ Context) (s string) { return p.name }
func (p *Plain) String() (s string) {
        var value = strings.Replace(p.string, "'", "\\'", -1)
        if p.name == "" {
                s = fmt.Sprintf("(plain '%s')", value)
        } else {
                s = fmt.Sprintf("((plain %s) '%s')", p.name, value)
        }
        return
}
func (p *Plain) expand(_ Context, _ facet) (val Value) { return /* &p.Raw */p }
func (p *Plain) cmp(ctx Context, v Value) (res cmpres) {
        if a, y := v.(*Plain); y {
                if p.name == a.name && p.string == a.string {
                        res = cmpEqual
                }
        } else if v.Strval(ctx) == p.string {
                res = cmpEqual
        }
        return
}
func (_ *Plain) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *Plain) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *Plain) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "cache unsupported").debug(32)
    return
}

type (
        plainInt struct {}
        plainOpts struct {
                generalOpts
        }
)
func (_ *plainInt) Evaluate(ctx Context, args ...Value) (result Value, err error) {
        var (
                program = ctx.program()
                pos = ctx.Position()
                str, name string
                opts plainOpts
        )
        if args = parseOpts(ctx, &opts, plain, args...); len(args) > 0 {
                name = args[0].Strval(ctx)
                program.language = name
        }
        if str, err = multiline(ctx, program.recipes...); err != nil {
                erro(of(ctx,args[0]), "%v", err).debug(1)
                return
        } else if len(program.recipes) > 0 {
                pos = program.recipes[0].Position()
        }
        str = strings.Replace(str, "\\\n\t", "\\\n", -1)
        result = &Plain{Raw{valbase{pos}, str}, name}
        if opts.debug>0 { warn(ctx, "%v", str).debug(opts.debug) }
        return
}

func multiline(ctx Context, recipes... Value) (res string, err error) {
        var (
                x = len(recipes)-1
                w = new(bytes.Buffer)
        )
        for n, recipe := range recipes {
                if fmt.Fprint(w, recipe.Strval(ctx)); n < x { fmt.Fprint(w, "\n") }
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
func DecodeXML(ctx Context, source string, ws bool) (result Value, err error) {
        var (
                pos = ctx.Position()
                stack []*Group
                nodes []*Group
                tok xml_enc.Token
        )
        xd := xml_enc.NewDecoder(strings.NewReader(source))
        for tok, err = xd.Token(); err == nil; tok, err = xd.Token() {
                switch elem := tok.(type) {
                case xml_enc.ProcInst:
                        // TODO: ...
                case xml_enc.StartElement:
                        nn := MakeGroup(pos, &bareword{valbase{pos},elem.Name.Local})
                        for _, a := range elem.Attr {
                                var k, v Value
                                k = &bareword{valbase{pos},a.Name.Local}
                                v = &String{valbase{pos},a.Value}
                                if s := a.Name.Space; s != "" {
                                        k = MakeGroup(pos, &String{valbase{pos},s}, k)
                                }
                                nn.Append(MakePair(pos, k, v))
                        }
                        if x := len(stack); x > 0 {
                                stack[x-1].Append(nn)
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
                                        node.Append(&String{valbase{pos},s})
                                } else {
                                        if s = strings.TrimSpace(s); s != "" {
                                                node.Append(&String{valbase{pos},s})
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
                g := MakeGroup(pos)
                for _, node := range nodes {
                        g.Append(node)
                }
                result = g
        } else if x == 1 {
                result = nodes[0]
        }
        if err == io.EOF {
                err = nil // all done
        }
        return
}

type xml struct { whitespace bool }
func (p *xml) Evaluate(ctx Context, args ...Value) (result Value, err error) {
        var source string
        if source, err = multiline(ctx, ctx.program().recipes...); err != nil {
                erro(ctx, "%v", err).debug(1)
                return
        }
        if result, err = DecodeXML(ctx, source, p.whitespace); err == nil {
                result = &XML{ result }
        } else {
                result = &XML{ MakeNone(ctx.Position()) }
        }
        return
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
        stack []*Group
        nodes []*Group
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
func DecodeJSON(ctx Context, source string) (result Value, err error) {
        //prompt(ctx, "json: %v\n", source)
        var (
                pos Position = ctx.Position()
                stack []*Group
                nodes []Value
                node *Group
                value Value
                t, v json_enc.Token
                s string
        )
        jd := json_enc.NewDecoder(strings.NewReader(source))
        LoopJSON: for {
                if t, err = jd.Token(); err != nil { break }
                x := len(stack)
                //prompt(ctx, "%T: %v\n", t, t)
        SwitchNodeType:
                switch node, value = nil, nil; d := t.(type) {
                case json_enc.Delim:
                        switch d {
                        case '[':
                                nn := MakeGroup(pos, MakeBareword(pos, JsonArray))
                                if x == 0 {
                                        nodes = append(nodes, nn)
                                } else {
                                        node, value = stack[x-1], nn
                                }
                                stack = append(stack, nn) // APPEND
                                break SwitchNodeType
                        case '{':
                                nn := MakeGroup(pos, MakeBareword(pos, JsonObject))
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
                                        if s = k.Strval(ctx); s != JsonObject {
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
                                        if s = k.Strval(ctx); s != JsonArray {
                                                err = ErrorIllJson; break LoopJSON
                                        }
                                }
                                stack = stack[0:x-1] // POP
                                continue LoopJSON
                        default:
                                err = ErrorIllJson; break LoopJSON
                        }
                case string:
                        var sv = &String{valbase{pos},d}
                        if x == 0 {
                                nodes = append(nodes, sv)
                                break
                        }

                        node = stack[x-1]
                        if k := node.Get(0); k != nil {
                                var kind string
                                if kind = k.Strval(ctx); kind == JsonArray {
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
                        case json_enc.Delim:
                                var vn *Group
                                switch vd {
                                case '[': vn = MakeGroup(pos, MakeBareword(pos, JsonArray))
                                case '{': vn = MakeGroup(pos, MakeBareword(pos, JsonObject))
                                default: err = ErrorIllJson; break LoopJSON
                                }
                                stack = append(stack, vn)
                                node.Append(MakePair(pos, sv, vn))
                        case string:
                                node.Append(MakePair(pos, sv, MakeString(pos, vd)))
                        case float64:
                                node.Append(MakePair(pos, sv, MakeFloat(pos, vd)))
                        case nil: // null
                                node.Append(MakePair(pos, sv, MakeBareword(pos, "null")))
                        default:
                                err = ErrorIllJson; break LoopJSON
                        }
                        //prompt(ctx, "node: %v\n", node)
                case float64:
                        if v := Value(MakeFloat(pos, d)); x == 0 {
                                nodes = append(nodes, v)
                        } else {
                                node, value = stack[x-1], v
                        }
                case nil: // null
                        if v := Value(MakeBareword(pos, "null")); x == 0 {
                                nodes = append(nodes, v)
                        } else {
                                node, value = stack[x-1], v
                        }
                default:
                        err = ErrorIllJson; break LoopJSON
                }
                if node != nil && value != nil {
                        if k := node.Get(0); k != nil {
                                if s = k.Strval(ctx); s != JsonArray {
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
                g := MakeGroup(pos)
                for _, v := range nodes {
                        g.Append(v)
                }
                result = g
        }
        return
}

type json struct {}

func (_ *json) Evaluate(ctx Context, args ...Value) (result Value, err error) {
        var program = ctx.program()
        if program == nil {
                erro(ctx, `needs program context to evaluate: %v`, ctx).debug(16)
                return
        }
        var source string
        if source, err = multiline(ctx, program.recipes...); err != nil { return }
        if result, err = DecodeJSON(ctx, source); err == nil {
                result = &JSON{ result }
        } else {
                result = &JSON{ MakeNone(program.position) }
        }
        return
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
func DecodeYAML(ctx Context, source string, ws bool) (result Value, err error) {
        err = fmt.Errorf("TODO: implement DecodeYAML")
        return
}

type yaml struct { whitespace bool }
func (p *yaml) Evaluate(ctx Context, args ...Value) (result Value, err error) {
        var source string
        if source, err = multiline(ctx, ctx.program().recipes...); err != nil {
                erro(ctx, "%v", err).debug(1)
                return
        } else if result, err = DecodeYAML(ctx, source, p.whitespace); err == nil {
                result = &YAML{ result }
        } else {
                result = &YAML{ MakeNone(ctx.Position()) }
                erro(ctx, "%v", err).debug(1)
        }
        return
}
