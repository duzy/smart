//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        xml_enc "encoding/xml"
        "strings"
        "io"
)

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
) */
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
                        nn := MakeGroup(pos, &Bareword{valbase{pos},elem.Name.Local})
                        for _, a := range elem.Attr {
                                var k, v Value
                                k = &Bareword{valbase{pos},a.Name.Local}
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
        if source, err = multiline(ctx, ctx.Program().recipes...); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }
        if result, err = DecodeXML(ctx, source, p.whitespace); err == nil {
                result = &XML{ result }
        } else {
                result = &XML{ MakeNone(ctx.Position()) }
        }
        return
}
