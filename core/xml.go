//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package core

import (
        "encoding/xml"
        "strings"
        "io"
)

type XML struct {
        Value Value
}
func (p *XML) disclose(_ *Scope) (Value, error) { return nil, nil }
func (p *XML) referencing(_ Object) bool { return false }
func (p *XML) Type() Type { return XMLType }
func (p *XML) String() string { return "(json " + p.Value.String() + ")" }
func (p *XML) Strval() (string, error) { return p.Value.Strval() }
func (p *XML) Integer() (int64, error) { return 0, nil }
func (p *XML) Float() (float64, error) { return 0, nil }

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
        )
        xd := xml.NewDecoder(strings.NewReader(source))
        var tok xml.Token
        for tok, err = xd.Token(); err == nil; tok, err = xd.Token() {
                switch elem := tok.(type) {
                case xml.ProcInst:
                        // TODO: ...
                case xml.StartElement:
                        nn := MakeGroup(&Bareword{elem.Name.Local})
                        for _, a := range elem.Attr {
                                var k, v Value
                                k = &Bareword{a.Name.Local}
                                v = &String{a.Value}
                                if s := a.Name.Space; s != "" {
                                        k = MakeGroup(&String{s}, k)
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
                                        node.Append(MakeString(s))
                                } else {
                                        if s = strings.TrimSpace(s); s != "" {
                                                node.Append(MakeString(s))
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

type dialectXml struct {
        whitespace bool
}

func (t *dialectXml) Evaluate(prog *Program, args []Value, recipes []Value) (result Value, err error) {
        var source string
        if source, err = joinRecipesString(recipes...); err != nil { return }
        if result, err = DecodeXML(source, t.whitespace); err == nil {
                result = &XML{ result }
        } else {
                result = &XML{ UniversalNone }
        }
        return
}

func init() {
        RegisterDialect("xml", &dialectXml{
                whitespace: false,
        })
}
