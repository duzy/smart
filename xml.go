//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "encoding/xml"
        "strings"
        "io"
)

type XML struct { Value Value }
func (p *XML) refs(_ Value) bool { return false }
func (p *XML) closured() bool { return p.Value.closured() }
func (p *XML) expand(w expandwhat) (Value, error) { return p.Value.expand(w) }
func (p *XML) Position() Position { return p.Value.Position() }
func (p *XML) True() bool { return p.Value.True() }
func (p *XML) String() string { return "(json " + p.Value.String() + ")" }
func (p *XML) Strval() (string, error) { return p.Value.Strval() }
func (p *XML) Integer() (int64, error) { return 0, nil }
func (p *XML) Float() (float64, error) { return 0, nil }
func (p *XML) cmp(v Value) (res cmpres) {
        if a, ok := v.(*XML); ok {
                assert(ok, "value is not XML")
                res = p.Value.cmp(a.Value)
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
func DecodeXML(source string, ws bool) (result Value, err error) {
        var (
                stack []*Group
                nodes []*Group
                pos Position // FIXME: calculate the position
        )
        xd := xml.NewDecoder(strings.NewReader(source))
        var tok xml.Token
        for tok, err = xd.Token(); err == nil; tok, err = xd.Token() {
                switch elem := tok.(type) {
                case xml.ProcInst:
                        // TODO: ...
                case xml.StartElement:
                        nn := MakeGroup(&Bareword{pos,elem.Name.Local})
                        for _, a := range elem.Attr {
                                var k, v Value
                                k = &Bareword{pos,a.Name.Local}
                                v = &String{pos,a.Value}
                                if s := a.Name.Space; s != "" {
                                        k = MakeGroup(&String{pos,s}, k)
                                }
                                nn.Append(MakePair(k, v))
                        }
                        if x := len(stack); x > 0 {
                                stack[x-1].Append(nn)
                        } else {
                                nodes = append(nodes, nn)
                        }
                        stack = append(stack, nn)
                case xml.EndElement:
                        if x := len(stack); x > 0 {
                                stack = stack[0:x-1]
                        } else {
                                // FIXME: report illegal xml
                        }
                case xml.CharData:
                        if x := len(stack); x > 0 {
                                node, s := stack[x-1], string(elem)
                                if ws {
                                        node.Append(&String{pos,s})
                                } else {
                                        if s = strings.TrimSpace(s); s != "" {
                                                node.Append(&String{pos,s})
                                        }
                                }
                        }
                case xml.Directive:
                        // TODO: ...
                case xml.Comment:
                        // TODO: ...
                }
        }
        if x := len(nodes); x > 1 {
                g := MakeGroup()
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

type _xml struct {
        whitespace bool
}

func (t *_xml) Evaluate(prog *Program, args []Value) (result Value, err error) {
        var source string
        if source, err = joinRecipesString(prog.recipes...); err != nil { return }
        if result, err = DecodeXML(source, t.whitespace); err == nil {
                result = &XML{ result }
        } else {
                result = &XML{ universalnone }
        }
        return
}
