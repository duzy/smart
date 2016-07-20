//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "testing"
        //"strings"
        "bytes"
        "fmt"
        "os"
        //"os/exec"
        //"path/filepath"
)

func newTestLex(file, s string) (l *lex) {
        l = &lex{ parseBuffer:&parseBuffer{ scope:file, s:[]byte(s) }, pos:0, }
        return
}

func newTestContext(file, s string) (p *Context, e error) {
        p, e = NewContext(file, []byte(s), nil)
        return
}

func TestLexComments(t *testing.T) {
        l := newTestLex("TestLexComments", `
# this is a comment
# this is the second line of the comment

# this is another comment
# this is the second line of the other comment
# this is the third line of the other comment

# more...

foo = foo # comment 1
$(info info) # comment 2
bar = bar# comment 3
foobar=foobar#comment 4
foobaz=blah# comment 5 
foobac=# comment 6	`)
        l.parse()

        if len(l.nodes) != 15 { t.Errorf("expecting 15 nodes but got %v", len(l.nodes)) }

        countComments := 0
        for _, n := range l.nodes { if n.tc() == nodeTypeCodeComment { countComments++ } }
        if v := 9; countComments != v { t.Errorf("expecting %v comments but got %v", v, countComments) }

        var c node
        checkNode := func(n int, k nodeTypeCode, s string) (okay bool) {
                okay = true
                if len(l.nodes) < n+1 { t.Errorf("expecting at least %v nodes but got %v", n+1, len(l.nodes)); okay = false }
                if c = l.nodes[n]; c.tc() != k { t.Errorf("expecting node %v as %v but got %v(%v)", n, k, c.tc(), c.str()); okay = false }
                if c.str() != s { t.Errorf("expecting node %v as %v(%v) but got %v(%v)", n, k, s, c.tc(), c.str()); okay = false }
                return
        }

        if !checkNode(0, nodeTypeCodeComment, `# this is a comment
# this is the second line of the comment`) { return }
        if !checkNode(1, nodeTypeCodeComment, `# this is another comment
# this is the second line of the other comment
# this is the third line of the other comment`) { return }
        if !checkNode(2, nodeTypeCodeComment, `# more...`) { return }
        if !checkNode(3, nodeTypeCodeDefineDeferred, `=`) { return }
        if !checkNode(4, nodeTypeCodeComment, `# comment 1`) { return }
        if !checkNode(5, nodeTypeCodeImmediateText, `$(info info) `) { return }
        if !checkNode(6, nodeTypeCodeComment, `# comment 2`) { return }
        if !checkNode(7, nodeTypeCodeDefineDeferred, `=`) { return }
        if !checkNode(8, nodeTypeCodeComment, `# comment 3`) { return }
        if !checkNode(9, nodeTypeCodeDefineDeferred, `=`) { return }
        if !checkNode(10, nodeTypeCodeComment, `#comment 4`) { return }
        if !checkNode(11, nodeTypeCodeDefineDeferred, `=`) { return }
        if !checkNode(12, nodeTypeCodeComment, `# comment 5 `) { return }
        if !checkNode(13, nodeTypeCodeDefineDeferred, `=`) { return }
        if !checkNode(14, nodeTypeCodeComment, `# comment 6	`) { return }
}

func TestLexAssigns(t *testing.T) {
        l := newTestLex("TestLexAssigns", `
a = a
b= b
c=c
d       =           d
foo := $(a) \
 $b\
 ${c}\

bar = $(foo) \
$(a) \
 $b $c

f_$a_$b_$c_1 = f_a_b_c
f_$a_$b_$c_2 = f_$a_$b_$c

a += a
a += a
b ?= b
cc ::= cc
n != n

f_$a_$b_$c_3 := f_$($a)_$($($b))_$(${$($c)})

empty1 =
empty2 :=
empty3 = 
empty4 =    
empty5 =	

foo.bar = hierarchy
`)
        l.parse()

        if ex := 20; len(l.nodes) != ex { t.Errorf("expecting %v nodes but got %v", ex, len(l.nodes)) }

        var (
                countDeferredDefines = 0
                countQuestionedDefines = 0
                countSingleColonedDefines = 0
                countDoubleColonedDefines = 0
                countAppendDefines = 0
                countNotDefines = 0
        )
        for _, n := range l.nodes {
                switch n.tc() {
                case nodeTypeCodeDefineDeferred:        countDeferredDefines++
                case nodeTypeCodeDefineQuestioned:      countQuestionedDefines++
                case nodeTypeCodeDefineSingleColoned:   countSingleColonedDefines++
                case nodeTypeCodeDefineDoubleColoned:   countDoubleColonedDefines++
                case nodeTypeCodeDefineAppend:          countAppendDefines++
                case nodeTypeCodeDefineNot:             countNotDefines++
                }
        }
        if ex := 12; countDeferredDefines != ex         { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeTypeCodeDefineDeferred,      countDeferredDefines) }
        if ex := 1; countQuestionedDefines != ex        { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeTypeCodeDefineQuestioned,    countQuestionedDefines) }
        if ex := 3; countSingleColonedDefines != ex     { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeTypeCodeDefineSingleColoned, countSingleColonedDefines) }
        if ex := 1; countDoubleColonedDefines != ex     { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeTypeCodeDefineDoubleColoned, countDoubleColonedDefines) }
        if ex := 2; countAppendDefines != ex            { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeTypeCodeDefineAppend,        countAppendDefines) }
        if ex := 1; countNotDefines != ex               { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeTypeCodeDefineNot,           countNotDefines) }
        if n := countDeferredDefines+countQuestionedDefines+countSingleColonedDefines+countDoubleColonedDefines+countAppendDefines+countNotDefines; len(l.nodes) != n {
                t.Errorf("expecting %v nodes totally, but got %v", len(l.nodes), n)
        }

        if c, x := l.nodes[0], nodeTypeCodeDefineDeferred; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[1], nodeTypeCodeDefineDeferred; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[2], nodeTypeCodeDefineDeferred; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[3], nodeTypeCodeDefineDeferred; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "d"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "d"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[4], nodeTypeCodeDefineSingleColoned; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), ":="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "$(a) \\\n $b\\\n ${c}\\\n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := len(c1.children()), 6; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2, c3, c4, c5 := c1.child(0), c1.child(1), c1.child(2), c1.child(3), c1.child(4), c1.child(5)
                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c0.str(), "$(a)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c0.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.child(0).str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c1.tc(), nodeTypeCodeEscape; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c1.str(), "\\\n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                }
                                if a, b := c2.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c2.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c2.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c2.child(0).str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c3.tc(), nodeTypeCodeEscape; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c3.str(), "\\\n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                }
                                if a, b := c4.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c4.str(), "${c}"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c4.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c4.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c4.child(0).str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c5.tc(), nodeTypeCodeEscape; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c5.str(), "\\\n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                }
                        }
                }
        }

        if c, x := l.nodes[5], nodeTypeCodeDefineDeferred; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "bar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "$(foo) \\\n$(a) \\\n $b $c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := len(c1.children()), 6; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2, c3, c4, c5 := c1.child(0), c1.child(1), c1.child(2), c1.child(3), c1.child(4), c1.child(5)
                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c0.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.child(0).str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c1.tc(), nodeTypeCodeEscape; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c1.str(), "\\\n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                }
                                if a, b := c2.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c2.str(), "$(a)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c2.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c2.child(0).str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c3.tc(), nodeTypeCodeEscape; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c3.str(), "\\\n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                }
                                if a, b := c4.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c4.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c4.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c4.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c4.child(0).str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c5.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c5.str(), "$c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c5.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c5.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c5.child(0).str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[6], nodeTypeCodeDefineDeferred; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "f_$a_$b_$c_1"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "f_a_b_c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := len(c0.children()), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2 := c0.child(0), c0.child(1), c0.child(2)
                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c0.str(), "$a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c0.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.child(0).str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c1.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c1.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c1.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c1.child(0).str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c2.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c2.str(), "$c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c2.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c2.child(0).str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                        }
                }
        }
        
        if c, x := l.nodes[7], nodeTypeCodeDefineDeferred; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "f_$a_$b_$c_2"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "f_$a_$b_$c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := len(c0.children()), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2 := c0.child(0), c0.child(1), c0.child(2)
                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c0.str(), "$a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c0.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.child(0).str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c1.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c1.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c1.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c1.child(0).str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c2.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c2.str(), "$c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c2.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c2.child(0).str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                        }
                        if a, b := len(c1.children()), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2 := c1.child(0), c1.child(1), c1.child(2)
                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c0.str(), "$a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c0.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.child(0).str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c1.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c1.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c1.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c1.child(0).str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c2.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c2.str(), "$c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c2.child(0).tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c2.child(0).str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[8], nodeTypeCodeDefineAppend; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "+="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[9], nodeTypeCodeDefineAppend; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "+="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[10], nodeTypeCodeDefineQuestioned; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "?="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[11], nodeTypeCodeDefineDoubleColoned; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "::="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "cc"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "cc"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[12], nodeTypeCodeDefineNot; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "!="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "n"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[13], nodeTypeCodeDefineSingleColoned; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), ":="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "f_$a_$b_$c_3"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "f_$($a)_$($($b))_$(${$($c)})"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := len(c0.children()), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2 := c0.child(0), c0.child(1), c0.child(2)
                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c0.str(), "$a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c0.child(0)
                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c1.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c1.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c1.child(0)
                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                                if a, b := c2.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c2.str(), "$c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c2.child(0)
                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                        }
                        if a, b := len(c1.children()), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2 := c1.child(0), c1.child(1), c1.child(2)
                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c0.str(), "$($a)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c0.child(0)
                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.str(), "$a"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                c0 := c0.child(0)
                                                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := c0.str(), "$a"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                c0 := c0.child(0)
                                                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := c0.str(), "a"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                        }
                                                                }
                                                        }
                                                }
                                        }
                                }
                                if a, b := c1.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c1.str(), "$($($b))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c1.child(0)
                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.str(), "$($b)"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                c0 := c0.child(0)
                                                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := c0.str(), "$($b)"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                c0 := c0.child(0)
                                                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := c0.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                        c0 := c0.child(0)
                                                                                        if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                        if a, b := c0.str(), "$b"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                        c0 := c0.child(0)
                                                                                                        if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                        if a, b := c0.str(), "b"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                        if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                }
                                                                                        }
                                                                                }
                                                                        }
                                                                }
                                                        }
                                                }
                                        }
                                }
                                if a, b := c2.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        if a, b := c2.str(), "$(${$($c)})"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c2.child(0)
                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.str(), "${$($c)}"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                c0 := c0.child(0)
                                                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := c0.str(), "${$($c)}"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                c0 := c0.child(0)
                                                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := c0.str(), "$($c)"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                c0 := c0.child(0)
                                                                                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                if a, b := c0.str(), "$($c)"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                                c0 := c0.child(0)
                                                                                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                                if a, b := c0.str(), "$c"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                                                c0 := c0.child(0)
                                                                                                                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                                                if a, b := c0.str(), "$c"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                                                                c0 := c0.child(0)
                                                                                                                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                                                                if a, b := c0.str(), "c"; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                                                                                        if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                                                                                }
                                                                                                                                        }
                                                                                                                                }
                                                                                                                        }
                                                                                                                }
                                                                                                        }
                                                                                                }
                                                                                        }
                                                                                }
                                                                        }
                                                                }
                                                        }
                                                }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[14], nodeTypeCodeDefineDeferred; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "empty1"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), ""; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }
        if c, x := l.nodes[15], nodeTypeCodeDefineSingleColoned; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), ":="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "empty2"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), ""; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }
        if c, x := l.nodes[16], nodeTypeCodeDefineDeferred; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "empty3"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), ""; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }
        if c, x := l.nodes[17], nodeTypeCodeDefineDeferred; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "empty4"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), ""; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }
        if c, x := l.nodes[18], nodeTypeCodeDefineDeferred; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "empty5"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), ""; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }

        if c, x := l.nodes[19], nodeTypeCodeDefineDeferred; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "foo.bar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0 := c0.child(0)
                                        if a, b := c0.tc(), nodeTypeCodeNamePart; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c0.str(), "."; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                }
                        }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c1.str(), "hierarchy"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        }
                }
        }
}

func TestLexCalls(t *testing.T) {
        l := newTestLex("TestLexCalls", `
foo = foo
bar = bar
foobar := $(foo)$(bar)
foobaz := $(foo)\
$(bar)\

$(info $(foo)$(bar))
$(info $(foobar))
$(info $(foo),$(bar),$(foobar))
$(info $(foo) $(bar), $(foobar) )
$(info $($(foo)),$($($(foo)$(bar))))

aaa |$(foo)|$(bar)| aaa

name = $(prefix:name)
name = $(prefix:part1.part2.part3)
`)
        l.parse()

        if ex := 12; len(l.nodes) != ex { t.Errorf("expecting %v nodes but got %v", ex, len(l.nodes)) }
        
        var (
                countImmediateTexts = 0
                countDeferredDefines = 0
                countSingleColonedDefines = 0
        )
        for _, n := range l.nodes {
                //t.Logf("%v", n.tc())
                switch n.tc() {
                case nodeTypeCodeImmediateText:         countImmediateTexts++
                case nodeTypeCodeDefineDeferred:        countDeferredDefines++
                case nodeTypeCodeDefineSingleColoned:   countSingleColonedDefines++
                }
        }
        if ex := 6; countImmediateTexts != ex           { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeTypeCodeImmediateText,       countImmediateTexts) }
        if ex := 4; countDeferredDefines != ex          { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeTypeCodeDefineDeferred,      countDeferredDefines) }
        if ex := 2; countSingleColonedDefines != ex     { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeTypeCodeDefineSingleColoned, countSingleColonedDefines) }
        if n := countImmediateTexts+countDeferredDefines+countSingleColonedDefines; len(l.nodes) != n {
                t.Errorf("expecting %v nodes totally, but got %v", len(l.nodes), n)
        }

        if c, x := l.nodes[4], nodeTypeCodeImmediateText; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "$(info $(foo)$(bar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0 := c.child(0)
                        if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "$(info $(foo)$(bar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0, c1 := c0.child(0), c0.child(1)
                                        if a, b := c0.str(), "info"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c1.str(), "$(foo)$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c1.tc(), nodeTypeCodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0, c1 := c1.child(0), c1.child(1)
                                                if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                c0 := c0.child(0)
                                                                if a, b := c0.str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        }
                                                }
                                                if a, b := c1.str(), "$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c1.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c1.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c1.child(0)
                                                        if a, b := c0.str(), "bar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[5], nodeTypeCodeImmediateText; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "$(info $(foobar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0 := c.child(0)
                        if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "$(info $(foobar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0, c1 := c0.child(0), c0.child(1)
                                        if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c0.str(), "info"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        }
                                        if a, b := c1.tc(), nodeTypeCodeArg; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                if a, b := c1.str(), "$(foobar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c1.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c1.child(0)
                                                        if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.str(), "$(foobar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                c0 := c0.child(0)
                                                                if a, b := c0.str(), "foobar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        }
                                                }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[6], nodeTypeCodeImmediateText; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "$(info $(foo),$(bar),$(foobar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0 := c.child(0)
                        if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "$(info $(foo),$(bar),$(foobar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children()), 4; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0, c1, c2, c3 := c0.child(0), c0.child(1), c0.child(2), c0.child(3)
                                        if a, b := c0.str(), "info"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c1.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c1.tc(), nodeTypeCodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c1.child(0)
                                                if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c0.child(0)
                                                        if a, b := c0.str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                }
                                        }
                                        if a, b := c2.str(), "$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c2.tc(), nodeTypeCodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c2.child(0)
                                                if a, b := c0.str(), "$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c0.child(0)
                                                        if a, b := c0.str(), "bar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                }
                                        }
                                        if a, b := c3.str(), "$(foobar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c3.tc(), nodeTypeCodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c3.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c3.child(0)
                                                if a, b := c0.str(), "$(foobar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c0.child(0)
                                                        if a, b := c0.str(), "foobar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[7], nodeTypeCodeImmediateText; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "$(info $(foo) $(bar), $(foobar) )"; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0 := c.child(0)
                        if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                if a, b := c0.str(), "$(info $(foo) $(bar), $(foobar) )"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children()), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0, c1, c2 := c0.child(0), c0.child(1), c0.child(2)
                                        if a, b := c0.str(), "info"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c1.str(), "$(foo) $(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c1.tc(), nodeTypeCodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c1.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0, c1 := c1.child(0), c1.child(1)
                                                if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c0.child(0)
                                                        if a, b := c0.str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                }
                                                if a, b := c1.str(), "$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c1.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c1.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c1.child(0)
                                                        if a, b := c0.str(), "bar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                }
                                        }
                                        if a, b := c2.str(), " $(foobar) "; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c2.tc(), nodeTypeCodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c2.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c2.child(0)
                                                if a, b := c0.str(), "$(foobar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c0.child(0)
                                                        if a, b := c0.str(), "foobar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[8], nodeTypeCodeImmediateText; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "$(info $($(foo)),$($($(foo)$(bar))))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0 := c.child(0)
                        if a, b := c0.str(), "$(info $($(foo)),$($($(foo)$(bar))))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := len(c0.children()), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0, c1, c2 := c0.child(0), c0.child(1), c0.child(2)
                                if a, b := c0.str(), "info"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c1.str(), "$($(foo))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c1.tc(), nodeTypeCodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c1.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0 := c1.child(0)
                                        if a, b := c0.str(), "$($(foo))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c0.child(0)
                                                if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c0.child(0)
                                                        if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                c0 := c0.child(0)
                                                                if a, b := c0.str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                }
                                                        }
                                                }
                                        }
                                }
                                if a, b := c2.str(), "$($($(foo)$(bar)))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c2.tc(), nodeTypeCodeArg; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c2.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0 := c2.child(0)
                                        if a, b := c0.str(), "$($($(foo)$(bar)))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c0.child(0)
                                                if a, b := c0.str(), "$($(foo)$(bar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                        c0 := c0.child(0)
                                                        if a, b := c0.str(), "$($(foo)$(bar))"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                c0 := c0.child(0)
                                                                if a, b := c0.str(), "$(foo)$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                if a, b := len(c0.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                        c0, c1 := c0.child(0), c0.child(1)
                                                                        if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                        if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                c0 := c0.child(0)
                                                                                if a, b := c0.str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                }
                                                                        }
                                                                        if a, b := c1.str(), "$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                        if a, b := c1.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                        if a, b := len(c1.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                c0 := c1.child(0)
                                                                                if a, b := c0.str(), "bar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                                                }
                                                                        }
                                                                }
                                                        }
                                                }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[9], nodeTypeCodeImmediateText; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "aaa |$(foo)|$(bar)| aaa"; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.str(), "$(foo)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0 := c0.child(0)
                                if a, b := c0.str(), "foo"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                }
                        }
                        if a, b := c1.str(), "$(bar)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := c1.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := len(c1.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0 := c1.child(0)
                                if a, b := c0.str(), "bar"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                }
                        }
                }
        }

        if c, x := l.nodes[10], nodeTypeCodeDefineDeferred; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.str(), "name"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        }
                        if a, b := c1.str(), "$(prefix:name)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := len(c1.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0 := c1.child(0)
                                if a, b := c0.str(), "$(prefix:name)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0 := c0.child(0)
                                        if a, b := c0.str(), "prefix:name"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0 := c0.child(0)
                                                if a, b := c0.str(), ":"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.tc(), nodeTypeCodeNamePrefix; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                }
                                        }
                                }
                        }
                }
        }

        if c, x := l.nodes[11], nodeTypeCodeDefineDeferred; c.tc() != x { t.Errorf("expecting %v but %v", x, c.tc()) } else {
                if a, b := c.str(), "="; a != b { t.Errorf("expecting %v but %v", b, a) }
                if a, b := len(c.children()), 2; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        c0, c1 := c.child(0), c.child(1)
                        if a, b := c0.str(), "name"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := c0.tc(), nodeTypeCodeImmediateText; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                        }
                        if a, b := c1.str(), "$(prefix:part1.part2.part3)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := c1.tc(), nodeTypeCodeDeferredText; a != b { t.Errorf("expecting %v but %v", b, a) }
                        if a, b := len(c1.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                c0 := c1.child(0)
                                if a, b := c0.str(), "$(prefix:part1.part2.part3)"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := c0.tc(), nodeTypeCodeCall; a != b { t.Errorf("expecting %v but %v", b, a) }
                                if a, b := len(c0.children()), 1; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                        c0 := c0.child(0)
                                        if a, b := c0.str(), "prefix:part1.part2.part3"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := c0.tc(), nodeTypeCodeName; a != b { t.Errorf("expecting %v but %v", b, a) }
                                        if a, b := len(c0.children()), 3; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                c0, c1, c2 := c0.child(0), c0.child(1), c0.child(2)
                                                if a, b := c0.str(), ":"; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c0.tc(), nodeTypeCodeNamePrefix; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c0.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                }
                                                if a, b := c1.str(), "."; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c1.tc(), nodeTypeCodeNamePart; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c1.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                }
                                                if a, b := c2.str(), "."; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := c2.tc(), nodeTypeCodeNamePart; a != b { t.Errorf("expecting %v but %v", b, a) }
                                                if a, b := len(c2.children()), 0; a != b { t.Errorf("expecting %v but %v", b, a) } else {
                                                }
                                        }
                                }
                        }
                }
        }
}

func TestLexEscapes(t *testing.T) {
        l := newTestLex("TestLexCalls", `
a = xxx\#xxx
b = xxx\
  yyy \
  zzz \

c = xxx \#\
  yyy \#\

`)
        l.parse()

        if ex := 3; len(l.nodes) != ex { t.Errorf("expecting %v nodes but got %v", ex, len(l.nodes)) }
        
        var (
                countDeferredDefines = 0
        )
        for _, n := range l.nodes {
                switch n.tc() {
                case nodeTypeCodeDefineDeferred:        countDeferredDefines++
                }
        }
        if ex := 3; countDeferredDefines != ex          { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeTypeCodeDefineDeferred,      countDeferredDefines) }
        if n := countDeferredDefines; len(l.nodes) != n {
                t.Errorf("expecting %v nodes totally, but got %v", len(l.nodes), n)
        }

        var (
                c node
                i int
        )
        checkNode := func(c node, k nodeTypeCode, cc int, s string, cs ...string) {
                if c.tc() != k { t.Errorf("%v: expecting kind %v but got %v", i, k, c.tc()) }
                if ss := c.str(); ss != s { t.Errorf("%v: expecting %v but got %v", i, s, ss) }
                if len(c.children()) != cc { t.Errorf("%v: expecting %v children but got %v", i, cc, len(c.children())) }

                var cn int
                for cn = 0; cn < len(c.children()) && cn < len(cs); cn++ {
                        nd := c.child(cn)
                        if nd.getPosEnd() < nd.getPosBeg() {
                                t.Errorf("%v: child %v has bad range [%v, %v) (%v)", i, cn, nd.getPosBeg(), nd.getPosEnd(), c.str())
                        }
                        if s := nd.str(); s != cs[cn] {
                                t.Errorf("%v: expecting child %v '%v', but '%v', in '%v'", i, cn, cs[cn], s, nd.str())
                        }
                }
                if cn != len(cs) { t.Errorf("%v: expecting at least %v children, but got %v", i, len(cs), cn) }
        }

        var cc node

        i = 0; c = l.nodes[i]; checkNode(c, nodeTypeCodeDefineDeferred, 2, `=`, `a`, `xxx\#xxx`)
        cc = c.child(0); checkNode(cc, nodeTypeCodeImmediateText, 0, `a`)
        cc = c.child(1); checkNode(cc, nodeTypeCodeDeferredText, 1, `xxx\#xxx`, `\#`)

        i = 1; c = l.nodes[i]; checkNode(c, nodeTypeCodeDefineDeferred, 2, `=`, `b`, "xxx\\\n  yyy \\\n  zzz \\\n")
        cc = c.child(0); checkNode(cc, nodeTypeCodeImmediateText, 0, `b`)
        cc = c.child(1); checkNode(cc, nodeTypeCodeDeferredText, 3, "xxx\\\n  yyy \\\n  zzz \\\n", "\\\n", "\\\n", "\\\n")

        i = 2; c = l.nodes[i]; checkNode(c, nodeTypeCodeDefineDeferred, 2, `=`, `c`, "xxx \\#\\\n  yyy \\#\\\n")
        cc = c.child(0); checkNode(cc, nodeTypeCodeImmediateText, 0, `c`)
        cc = c.child(1); checkNode(cc, nodeTypeCodeDeferredText, 4, "xxx \\#\\\n  yyy \\#\\\n", "\\#", "\\\n", "\\#", "\\\n")
}

func TestLexRules(t *testing.T) {
        /*
## Bash
foobar : foo bar blah {{
    gcc -c $1 $2
}}

## TCL
blah : blah.c [tcl]{{
    [ gcc -c $< -o $@ ]
}}
        */
        l := newTestLex("TestLexRules", `
foobar : foo bar blah

foo: foo.c ; gcc -c -O2 $< -o $@
bar: bar.c
	# this is command line comment
# this is script comment being ignored
	@echo "compiling..."
	@gcc -c $< -o $@

baz:bar
zz::foo

a:
	echo blah blah

blah : blah.c
	gcc -c $< -o $@
`)
        l.parse()

        if ex, n := 7, len(l.nodes); n != ex { t.Errorf("expecting %v nodes but got %v", ex, n) }
        
        var (
                countRuleSingleColoned = 0
                countRuleDoubleColoned = 0
        )
        for _, n := range l.nodes {
                //fmt.Fprintf(os.Stderr, "TestLexRules: %v: %v\n", n.tc(), n.child(0).str())
                switch n.tc() {
                case nodeTypeCodeRuleSingleColoned:     countRuleSingleColoned++
                case nodeTypeCodeRuleDoubleColoned:     countRuleDoubleColoned++
                }
        }
        if ex := 6; countRuleSingleColoned != ex          { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeTypeCodeRuleSingleColoned,      countRuleSingleColoned) }
        if ex := 1; countRuleDoubleColoned != ex          { t.Errorf("expecting %v %v nodes, but got %v", ex, nodeTypeCodeRuleDoubleColoned,      countRuleDoubleColoned) }
        if n := countRuleSingleColoned+countRuleDoubleColoned; len(l.nodes) != n {
                t.Errorf("expecting %v nodes totally, but got %v", len(l.nodes), n)
        }

        if n, x := len(l.nodes), 7; n != x { t.Errorf("%v != %v", n, x) } else {
                if c, x := l.nodes[0], nodeTypeCodeRuleSingleColoned; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                        if s, x := c.str(), ":"; s != x { t.Errorf("%v != %v", s, x) }
                        if n, x := len(c.children()), 2; n != x { t.Errorf("%v != %v", n, x) } else {
                                if c, x := c.child(0), nodeTypeCodeTargets; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "foobar"; s != x { t.Errorf("%v != %v", s, x) }
                                }
                                if c, x := c.child(1), nodeTypeCodePrerequisites; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "foo bar blah"; s != x { t.Errorf("%v != %v", s, x) }
                                }
                        }
                }
                if c, x := l.nodes[1], nodeTypeCodeRuleSingleColoned; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                        if s, x := c.str(), ":"; s != x { t.Errorf("%v != %v", c.tc(), x) }
                        if n, x := len(c.children()), 3; n != x { t.Errorf("%v != %v", n, x) } else {
                                if c, x := c.child(0), nodeTypeCodeTargets; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "foo"; s != x { t.Errorf("%v != %v", s, x) }
                                }
                                if c, x := c.child(1), nodeTypeCodePrerequisites; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "foo.c"; s != x { t.Errorf("%v != %v", s, x) }
                                }
                                if c, x := c.child(2), nodeTypeCodeRecipes; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "; gcc -c -O2 $< -o $@\n"; s != x { t.Errorf("%v != %v", s, x) }
                                        if n, x := len(c.children()), 1; n != x { t.Errorf("%v != %v", n, x) } else {
                                                if c, x := c.child(0), nodeTypeCodeRecipe; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                                        if s, x := c.str(), "gcc -c -O2 $< -o $@"; s != x { t.Errorf("%v != %v", s, x) }
                                                }
                                        }
                                }
                        }
                }
                if c, x := l.nodes[2], nodeTypeCodeRuleSingleColoned; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                        if s, x := c.str(), ":"; s != x { t.Errorf("%v != %v", c.tc(), x) }
                        if n, x := len(c.children()), 3; n != x { t.Errorf("%v != %v", n, x) } else {
                                if c, x := c.child(0), nodeTypeCodeTargets; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "bar"; s != x { t.Errorf("%v != %v", s, x) }
                                }
                                if c, x := c.child(1), nodeTypeCodePrerequisites; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "bar.c"; s != x { t.Errorf("%v != %v", s, x) }
                                }
                                if c, x := c.child(2), nodeTypeCodeRecipes; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), `	# this is command line comment
# this is script comment being ignored
	@echo "compiling..."
	@gcc -c $< -o $@
`; s != x { t.Errorf("%v != %v", s, x) } else {
        if n, x := len(c.children()), 4; n != x { t.Errorf("%v != %v", n, x) } else {
                if c, x := c.child(0), nodeTypeCodeRecipe; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                        if s, x := c.str(), "# this is command line comment"; s != x { t.Errorf("%v != %v", s, x) }
                }
                if c, x := c.child(1), nodeTypeCodeComment; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                        if s, x := c.str(), `# this is script comment being ignored`; s != x { t.Errorf("%v != %v", s, x) }
                }
                if c, x := c.child(2), nodeTypeCodeRecipe; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                        if s, x := c.str(), `@echo "compiling..."`; s != x { t.Errorf("%v != %v", s, x) }
                }
                if c, x := c.child(3), nodeTypeCodeRecipe; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                        if s, x := c.str(), `@gcc -c $< -o $@`; s != x { t.Errorf("%v != %v", s, x) }
                }
        }
}
                                }
                        }
                }
                if c, x := l.nodes[3], nodeTypeCodeRuleSingleColoned; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                        if s, x := c.str(), ":"; s != x { t.Errorf("%v != %v", c.tc(), x) }
                        if n, x := len(c.children()), 2; n != x { t.Errorf("%v != %v", n, x) } else {
                                if c, x := c.child(0), nodeTypeCodeTargets; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "baz"; s != x { t.Errorf("%v != %v", s, x) }
                                }
                                if c, x := c.child(1), nodeTypeCodePrerequisites; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "bar"; s != x { t.Errorf("%v != %v", s, x) }
                                }
                        }
                }
                if c, x := l.nodes[4], nodeTypeCodeRuleDoubleColoned; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                        if s, x := c.str(), "::"; s != x { t.Errorf("%v != %v", c.tc(), x) }
                        if n, x := len(c.children()), 2; n != x { t.Errorf("%v != %v", n, x) } else {
                                if c, x := c.child(0), nodeTypeCodeTargets; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "zz"; s != x { t.Errorf("%v != %v", s, x) }
                                }
                                if c, x := c.child(1), nodeTypeCodePrerequisites; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "foo"; s != x { t.Errorf("%v != %v", s, x) }
                                }
                        }
                }
                if c, x := l.nodes[5], nodeTypeCodeRuleSingleColoned; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                        if s, x := c.str(), ":"; s != x { t.Errorf("%v != %v", c.tc(), x) }
                        if n, x := len(c.children()), 3; n != x { t.Errorf("%v != %v", n, x) } else {
                                if c, x := c.child(0), nodeTypeCodeTargets; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "a"; s != x { t.Errorf("%v != %v", s, x) }
                                }
                                if c, x := c.child(1), nodeTypeCodePrerequisites; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), ""; s != x { t.Errorf("%v != %v", s, x) }
                                }
                                if c, x := c.child(2), nodeTypeCodeRecipes; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "	echo blah blah\n"; s != x { t.Errorf("%v != %v", s, x) }
                                        if n, x := len(c.children()), 1; n != x { t.Errorf("%v != %v", n, x) } else {
                                                if c, x := c.child(0), nodeTypeCodeRecipe; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                                        if s, x := c.str(), "echo blah blah"; s != x { t.Errorf("%v != %v", s, x) }
                                                }
                                        }
                                }
                        }
                }
                if c, x := l.nodes[6], nodeTypeCodeRuleSingleColoned; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                        if s, x := c.str(), ":"; s != x { t.Errorf("%v != %v", c.tc(), x) }
                        if n, x := len(c.children()), 3; n != x { t.Errorf("%v != %v", n, x) } else {
                                if c, x := c.child(0), nodeTypeCodeTargets; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "blah"; s != x { t.Errorf("%v != %v", s, x) }
                                }
                                if c, x := c.child(1), nodeTypeCodePrerequisites; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "blah.c"; s != x { t.Errorf("%v != %v", s, x) }
                                }
                                if c, x := c.child(2), nodeTypeCodeRecipes; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "\tgcc -c $< -o $@\n"; s != x { t.Errorf("%v != %v", s, x) }
                                        if n, x := len(c.children()), 1; n != x { t.Errorf("%v != %v", n, x) } else {
                                                if c, x := c.child(0), nodeTypeCodeRecipe; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                                        if s, x := c.str(), "gcc -c $< -o $@"; s != x { t.Errorf("%v != %v", s, x) }
                                                }
                                        }
                                }
                        }
                }
        }
}

func TestLexSpeak(t *testing.T) {
        l := newTestLex("TestLexSpeak", `
text = $(speak text,\
-----------------------
blah blah blah blah...
----------------------)
`)
        l.parse()
        if ex, n := 1, len(l.nodes); n != ex { t.Errorf("expecting %v nodes but got %v", ex, n) }
        if c, k := l.nodes[0], nodeTypeCodeDefineDeferred; c.tc() != k { t.Errorf("expecting %v but %v", c.tc(), k) } else {
                if n, x := len(c.children()), 2; n != x { t.Errorf("expecting %v but %v", n, x) } else {
                        if c, k := c.child(1), nodeTypeCodeDeferredText; c.tc() != k { t.Errorf("expecting %v but %v", c.tc(), k) } else {
                                if n, x := len(c.children()), 1; n != x { t.Errorf("expecting %v but %v", n, x) } else {
                                        if c, k := c.child(0), nodeTypeCodeSpeak; c.tc() != k { t.Errorf("expecting %v but %v", c.tc(), k) } else {
                                                if n, x := len(c.children()), 2; n != x { t.Errorf("expecting %v but %v", n, x) } else {
                                                        if c, k := c.child(0), nodeTypeCodeArg; c.tc() != k { t.Errorf("expecting %v but %v", c.tc(), k) } else {
                                                                if n, x := len(c.children()), 0; n != x { t.Errorf("expecting %v but %v", n, x) }
                                                                if s, x := c.str(), "text"; s != x { t.Errorf("expecting %v but %v", s, x) }
                                                        }
                                                        if c, k := c.child(1), nodeTypeCodeArg; c.tc() != k { t.Errorf("expecting %v but %v", c.tc(), k) } else {
                                                                if n, x := len(c.children()), 0; n != x { t.Errorf("expecting %v but %v", n, x) }
                                                                if s, x := c.str(), "blah blah blah blah..."; s != x { t.Errorf("expecting %v but %v", s, x) }
                                                        }
                                                }
                                        }
                                }
                        }
                }
        }
}

func TestParse(t *testing.T) {
        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestParse#1", `
a = a
i = i
ii = x x $a $i x x
i1 = $a$i-$(ii)
i2 = $a$($i)-$($i$i)
`);     if err != nil { t.Error("parse error:", err) }
        if ex, nl := 5, len(ctx.l.nodes); nl != ex { t.Error("expect", ex, "but", nl) }

        for _, s := range []string{ "a", "i", "ii", "i1", "i2" } {
                if d, ok := ctx.g.defines[s]; !ok || d == nil { t.Errorf("missing '%v'", s) } else {
                        if v := d.value.Expand(ctx); v == "" { t.Errorf("'%v', '%v'", s, v) }
                }
        }

        if l1, l2 := len(ctx.g.defines), 5; l1 != l2 { t.Errorf("expects '%v' defines but got '%v'", l2, l1) }

        if s, ex := ctx.Call("a").Expand(ctx), "a";                   s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("i").Expand(ctx), "i";                   s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("ii").Expand(ctx), "x x a i x x";        s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("i1").Expand(ctx), "ai-x x a i x x";     s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("i2").Expand(ctx), "ai-x x a i x x";     s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }

        //////////////////////////////////////////////////
        ctx, err = newTestContext("TestParse#2", `
a = a
i = i
ii = i $a i a \
 $a i
sh$ared = shared
stat$ic = static
a$$a = foo
aaaa = xxx$(info 1:$(sh$ared),$(stat$ic))-$(a$$a)-xxx
bbbb := xxx$(info 2:$(sh$ared),$(stat$ic))-$(a$$a)-xxx
cccc = xxx-$(sh$ared)-$(stat$ic)-$(a$$a)-xxx
dddd := xxx-$(sh$ared)-$(stat$ic)-$(a$$a)-xxx
`);     if err != nil { t.Error("parse error:", err) }
        if ex, nl := 10, len(ctx.g.defines); nl != ex { t.Error("expect", ex, "defines, but", nl) }

        for _, s := range []string{ "a", "i", "ii", "shared", "static", "a$a", "aaaa", "bbbb", "cccc", "dddd" } {
                if d, ok := ctx.g.defines[s]; !ok || d == nil { t.Errorf("missing '%v' (%v)", s, ctx.g.defines) } else {
                        if s := d.value.Expand(ctx); s == "" { t.Errorf("empty '%v'", s) }
                }
        }

        if s, ex := ctx.Call("a").Expand(ctx), "a";                   s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("i").Expand(ctx), "i";                   s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("ii").Expand(ctx), "i a i a   a i";      s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("shared").Expand(ctx), "shared";         s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("static").Expand(ctx), "static";         s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("a").Expand(ctx), "a";                   s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("a$a").Expand(ctx), "foo";               s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("a$$a").Expand(ctx), "";                 s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("aaaa").Expand(ctx), "xxx-foo-xxx";      s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("bbbb").Expand(ctx), "xxx-foo-xxx";      s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("cccc").Expand(ctx), "xxx-shared-static-foo-xxx";       s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := ctx.Call("dddd").Expand(ctx), "xxx-shared-static-foo-xxx";       s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        if s, ex := info.String(), "2:shared static\n1:shared static\n1:shared static\n"; s != ex {
                t.Errorf("expects '%v' but got '%v'", ex, s)
        }
}

func TestEmptyValue(t *testing.T) {
        ctx, err := newTestContext("TestEmptyValue", `
foo =
bar = bar
foobar := 
foobaz := foo-baz
`);     if err != nil { t.Errorf("parse error:", err) }
        if s := ctx.Call("foo").Expand(ctx); s != "" { t.Errorf("foo: '%s'", s) }
        if s := ctx.Call("bar").Expand(ctx); s != "bar" { t.Errorf("bar: '%s'", s) }
        if s := ctx.Call("foobar").Expand(ctx); s != "" { t.Errorf("foobar: '%s'", s) }
        if s := ctx.Call("foobaz").Expand(ctx); s != "foo-baz" { t.Errorf("foobaz: '%s'", s) }
}

func TestSetEmptyValue(t *testing.T) {
        ctx, err := newTestContext("TestEmptyValue", `
foo =
bar = bar
foobar := 
foobaz := foo-baz
`);     if err != nil { t.Errorf("parse error:", err) }
        if s := ctx.Call("foo").Expand(ctx); s != "" { t.Errorf("foo: '%s'", s) }
        if s := ctx.Call("bar").Expand(ctx); s != "bar" { t.Errorf("bar: '%s'", s) }
        if s := ctx.Call("foobar").Expand(ctx); s != "" { t.Errorf("foobar: '%s'", s) }
        if s := ctx.Call("foobaz").Expand(ctx); s != "foo-baz" { t.Errorf("foobaz: '%s'", s) }
}

func TestMultipartNames(t *testing.T) {
        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestMultipartNames", `
template test
commit

$(= test:foo, f o o)
#$(= test:foo.bar,  foo bar)

$(= test:foobar, f)     $(info [$(test:foobar 1,2,3)])
$(+= test:foobar, o, o) $(info [$(test:foobar 1,2,3)])

module test
me.foo = fooo
$(info $(me.foo))

module a
me.foo = foooo
$(info $(me.foo))
commit # test.a

me.a.bar = bar
commit # test

$(info $(test.foo))
$(info $(test.a.foo))
$(info $(test.a.name))

test.foo = FOOO
test.a.foo = FOOOO
$(info $(test.foo))
$(info $(test.a.foo))
`);     if err != nil { t.Errorf("parse error:", err) }
        if s, x := ctx.Call("test:foo").Expand(ctx), " f o o"; s != x { t.Errorf("expects '%s' but '%s'", x, s) }
        //if s, x := ctx.Call("test:foo.bar").Expand(ctx), "  foo bar (test:foo.bar:)"; s != x { t.Errorf("expects '%s' but '%s'", x, s) }
        if s, x := ctx.Call("test.foo").Expand(ctx), "FOOO"; s != x { t.Errorf("expects '%s' but '%s'", x, s) }
        if s, x := ctx.Call("test.a.foo").Expand(ctx), "FOOOO"; s != x { t.Errorf("expects '%s' but '%s'", x, s) }
        if s, x := ctx.Call("test.a.bar").Expand(ctx), "bar"; s != x { t.Errorf("expects '%s' but '%s'", x, s) }

        if v, s := info.String(), fmt.Sprintf(`[ f]
[ f test:foobar  o  o]
fooo
foooo
fooo
foooo
a
FOOO
FOOOO
`); v != s { t.Errorf("`%s` != `%s`", v, s) }
}

func TestContinualInCall(t *testing.T) {
        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                //fmt.Fprintf(info, "%v\n", args.Expand(ctx))
                for n, a := range args {
                        if n == 0 {
                                fmt.Fprint(info, a.Expand(ctx))
                        } else {
                                fmt.Fprintf(info, ",%v", a.Expand(ctx))
                        }
                }
                fmt.Fprintf(info, "\n")
        }

        ctx, err := newTestContext("TestContinualInCall", `
foo = $(info a,  ndk  , \
  PLATFORM=android-9, \
  ABI=x86 armeabi, \
)
`);     if err != nil { t.Errorf("parse error:", err) }
        if s := ctx.Call("foo").Expand(ctx); s != "" { t.Errorf("foo: '%s'", s) }
        // FIXIME: if s := info.String(); s != `a,  ndk  , PLATFORM=android-9, ABI=x86 armeabi, ` { t.Errorf("info: '%s'", s) }
        if a, b := info.String(), "a,  ndk  ,    PLATFORM=android-9,    ABI=x86 armeabi,  \n"; a != b { t.Errorf("expects '%s' but '%s'", b, a) }
}

func TestModuleVariables(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestModuleVariables", `
module test
$(info $(me) $(me.name) $(me.dir))
$(info $(me.export.nothing))
commit

$(info $(test.name) $(test.dir))
#module test ## error
`);     if err != nil { t.Errorf("parse error:", err) }

        if ctx.modules == nil { t.Errorf("nil modules") }
        if m, ok := ctx.modules["test"]; !ok || m == nil { t.Errorf("nil 'test' module") } else {
                if c, ok := m.Children["export"]; !ok || c == nil { t.Errorf("'me.export' is undefined") } else {
                        if d, ok := c.defines["name"]; !ok || d == nil { t.Errorf("no 'name' defined") } else {
                                if s := d.value.Expand(ctx); s != "export" { t.Errorf("name != 'export' (%v)", s) }
                        }
                }
                if d, ok := m.defines["name"]; !ok || d == nil { t.Errorf("no 'name' defined") } else {
                        if s := d.value.Expand(ctx); s != "test" { t.Errorf("name != 'test' (%v)", s) }
                }
                if d, ok := m.defines["dir"]; !ok || d == nil { t.Errorf("no 'dir' defined") } else {
                        if s := d.value.Expand(ctx); s != workdir { t.Errorf("dir != '%v' (%v)", workdir, s) }
                }
                ctx.With(m, func() {
                        if s := ctx.Call("me").Expand(ctx); s != "test" { t.Errorf("$(me) != test (%v)", s) }
                        if s := ctx.Call("me.name").Expand(ctx); s != "test" { t.Errorf("$(me.name) != test (%v)", s) }
                        if s := ctx.Call("me.dir").Expand(ctx); s != workdir { t.Errorf("$(me.dir) != %v (%v)", workdir, s) }
                })
        }

        if a, b := info.String(), fmt.Sprintf("test test %s\n\ntest %s\n", workdir, workdir); a != b { t.Errorf("expects '%v' but '%v'", b, a) }
}

func TestModuleTargets(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestModuleTargets", `
module test

## test.foo
foo:
	@echo "$..$@"

foobar: foo
	@echo "$..$@ : $<"

commit
`);     if err != nil { t.Errorf("parse error:", err) }

        if ctx.modules == nil { t.Errorf("nil modules") }
        if _, ok := ctx.g.rules["foo"]; ok { t.Errorf("foo defined in context") }
        if m, ok := ctx.modules["test"]; !ok || m == nil { t.Errorf("nil 'test' module") } else {
                if r, ok := m.rules["foo"]; !ok { t.Errorf("foo not defined in %v", m.GetName(ctx)) } else {
                        if n := len(r.targets); n != 1 { t.Errorf("incorrect number of targets: %v %v", n, r.targets) }
                        if n := len(r.prerequisites); n != 0 { t.Errorf("incorrect number of prerequisites: %v %v", n, r.prerequisites) }
                        if n := len(r.recipes); n != 1 { t.Errorf("incorrect number of recipes: %v %v", n, r.recipes) }
                }
                if r, ok := m.rules["foobar"]; !ok { t.Errorf("foobar not defined in %v", m.GetName(ctx)) } else {
                        if n := len(r.targets); n != 1 { t.Errorf("incorrect number of targets: %v %v", n, r.targets) }
                        if n := len(r.prerequisites); n != 1 { t.Errorf("incorrect number of prerequisites: %v %v", n, r.prerequisites) }
                        if n := len(r.recipes); n != 1 { t.Errorf("incorrect number of recipes: %v %v", n, r.recipes) }
                }
        }
        if s := info.String(); s != `` { t.Errorf("info: '%s'", s) }
}

func TestToolsetVariables(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        /*
        ndk, _ := exec.LookPath("ndk-build")
        sdk, _ := exec.LookPath("android")
        if ndk == "" { t.Errorf("'ndk-build' is not in the PATH") }
        if sdk == "" { t.Errorf("'android' is not in the PATH") }

        ndk = filepath.Dir(ndk)
        sdk = filepath.Dir(filepath.Dir(sdk)) */

        _, err := newTestContext("TestToolsetVariables", `
template test-ndk
commit

template test-sdk
me.support = xxxx
commit

template test-shell
commit

$(info $(test-shell:name))
$(info $(test-sdk:name))
$(info $(test-sdk:support a,b,c))
$(info $(test-ndk:name))
`);     if err != nil { t.Errorf("parse error:", err) }
        if v, s := info.String(), fmt.Sprintf(`test-shell
test-sdk

test-ndk
`); v != s { t.Errorf("`%s` != `%s`", v, s) }
}

func TestToolsetRules(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestToolsetRules", `
all: foo bar
foo:; @touch $@.txt 
bar:
	@echo $@ > $@.txt
	@touch $@.txt
`);     if err != nil { t.Errorf("parse error:", err) }
        if n, x := len(ctx.g.rules), 3; n != x { t.Errorf("wrong number of rules: %v", ctx.g.rules) }
        if r, ok := ctx.g.rules["all"]; !ok && r == nil { t.Errorf("'all' not defined") } else {
                if n, x := len(r.node.children()), 2; n != x { t.Errorf("children %d != %d", n, x) }
                if n, x := len(r.targets), 1; n != x { t.Errorf("targets %d != %d", n, x) } else {
                        if s, x := r.targets[0], "all"; s != x { t.Errorf("targets[0] %v != %v", s, x) }
                }
                if n, x := len(r.prerequisites), 2; n != x { t.Errorf("prerequisites %d != %d", n, x) } else {
                        if s, x := r.prerequisites[0], "foo"; s != x { t.Errorf("prerequisites[0] %v != %v", s, x) }
                        if s, x := r.prerequisites[1], "bar"; s != x { t.Errorf("prerequisites[0] %v != %v", s, x) }
                }
                if n, x := len(r.recipes), 0; n != x { t.Errorf("recipes %d != %d", n, x) }
                if c, ok := r.c.(*defaultTargetUpdater); !ok { t.Errorf("wrong type of checker %v", c) }
        }
        if r, ok := ctx.g.rules["foo"]; !ok && r == nil { t.Errorf("'foo' not defined") } else {
                if n, x := len(r.node.children()), 3; n != x { t.Errorf("children %d != %d", n, x) } else {
                        if c, x := r.node.child(0), nodeTypeCodeTargets; c.tc() != x { t.Errorf("children %v != %v", c.tc(), x) }
                        if c, x := r.node.child(1), nodeTypeCodePrerequisites; c.tc() != x { t.Errorf("children %v != %v", c.tc(), x) }
                        if c, x := r.node.child(2), nodeTypeCodeRecipes; c.tc() != x { t.Errorf("children %v != %v", c.tc(), x) } else {
                                if c, x := c.child(0), nodeTypeCodeRecipe; c.tc() != x { t.Errorf("children %v != %v", c.tc(), x) } else {
                                        if s, x := c.str(), "@touch $@.txt "; s != x { t.Errorf("%v != %v", s, x) }
                                }
                        }
                }
                if n, x := len(r.targets), 1; n != x { t.Errorf("targets %d != %d", n, x) } else {
                        if s, x := r.targets[0], "foo"; s != x { t.Errorf("targets[0] %v != %v", s, x) }
                }
                if n, x := len(r.prerequisites), 0; n != x { t.Errorf("prerequisites %d != %d", n, x) }
                if n, x := len(r.recipes), 1; n != x { t.Errorf("recipes %d != %d", n, x) } else {
                        if c, ok := r.recipes[0].(node); !ok { t.Errorf("recipes[0] '%v' is not node", r.recipes[0]) } else {
                                if k, x := c.tc(), nodeTypeCodeRecipe; k != x { t.Errorf("recipes[0] %v != %v", k, x) }
                                if s, x := c.str(), "@touch $@.txt "; s != x { t.Errorf("recipes[0] %v != %v", s, x) }
                                if s, x := c.Expand(ctx), "@touch .txt "; s != x { t.Errorf("recipes[0] %v != %v", s, x) }
                        }
                }
                if c, ok := r.c.(*defaultTargetUpdater); !ok { t.Errorf("wrong type of checker %v", c) }
        }
        if r, ok := ctx.g.rules["bar"]; !ok && r == nil { t.Errorf("'bar' not defined") } else {
                if n, x := len(r.node.children()), 3; n != x { t.Errorf("children %d != %d", n, x) } else {
                        if c, x := r.node.child(0), nodeTypeCodeTargets; c.tc() != x { t.Errorf("children %v != %v", c.tc(), x) }
                        if c, x := r.node.child(1), nodeTypeCodePrerequisites; c.tc() != x { t.Errorf("children %v != %v", c.tc(), x) }
                        if c, x := r.node.child(2), nodeTypeCodeRecipes; c.tc() != x { t.Errorf("children %v != %v", c.tc(), x) }
                }
                if n, x := len(r.targets), 1; n != x { t.Errorf("targets %d != %d", n, x) } else {
                        if s, x := r.targets[0], "bar"; s != x { t.Errorf("targets[0] %v != %v", s, x) }
                }
                if n, x := len(r.prerequisites), 0; n != x { t.Errorf("prerequisites %d != %d", n, x) }
                if n, x := len(r.recipes), 2; n != x { t.Errorf("recipes %d != %d", n, x) } else {
                        ctx.Set("@", stringitem("xxx"))
                        if c, ok := r.recipes[0].(node); !ok { t.Errorf("recipes[0] '%v' is not node", r.recipes[0]) } else {
                                if k, x := c.tc(), nodeTypeCodeRecipe; k != x { t.Errorf("recipes[0] %v != %v", k, x) }
                                if s, x := c.str(), "@echo $@ > $@.txt"; s != x { t.Errorf("recipes[0] %v != %v", s, x) }
                                if s, x := c.Expand(ctx), "@echo xxx > xxx.txt"; s != x { t.Errorf("recipes[0] %v != %v", s, x) }
                        }
                        if c, ok := r.recipes[1].(node); !ok { t.Errorf("recipes[1] '%v' is not node", r.recipes[1]) } else {
                                if k, x := c.tc(), nodeTypeCodeRecipe; k != x { t.Errorf("recipes[1] %v != %v", k, x) }
                                if s, x := c.str(), "@touch $@.txt"; s != x { t.Errorf("recipes[1] %v != %v", s, x) }
                                if s, x := c.Expand(ctx), "@touch xxx.txt"; s != x { t.Errorf("recipes[1] %v != %v", s, x) }
                        }
                        ctx.Set("@", stringitem(""))
                }
                if c, ok := r.c.(*defaultTargetUpdater); !ok { t.Errorf("wrong type of checker %v", c) }
        }

        if v, s := info.String(), fmt.Sprintf(``); v != s { t.Errorf("`%s` != `%s`", v, s) }
}

func TestToolsetRuleUpdaters(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestToolsetRules", `
foo:!:
	@echo -n foo > $@.txt
foo:?:
	@test -f $@.txt && test "$$(cat $@.txt)" = "foo"
bar:?:
	@test -f $@.txt && test "$$(cat $@.txt)" = "bar"
bar:!:
	@echo -n bar > $@.txt
foobar: foo bar
	@touch $@
`);     if err != nil { t.Errorf("parse error:", err) }
        if n, x := len(ctx.g.rules), 3; n != x { t.Errorf("wrong rules: %v", ctx.g.rules) }
        if r, ok := ctx.g.rules["foo"]; !ok && r == nil { t.Errorf("'all' not defined") } else {
                if k, x := r.node.tc(), nodeTypeCodeRuleChecker; k != x { t.Errorf("%v != %v", k, x) }
                if n, x := len(r.node.children()), 3; n != x { t.Errorf("children %d != %d", n, x) }
                if n, x := len(r.targets), 1; n != x { t.Errorf("targets %d != %d", n, x) } else {
                        if s, x := r.targets[0], "foo"; s != x { t.Errorf("targets[0] %v != %v", s, x) }
                }
                if n, x := len(r.prerequisites), 0; n != x { t.Errorf("prerequisites %d != %d", n, x) }
                if n, x := len(r.recipes), 1; n != x { t.Errorf("recipes %d != %d", n, x) } else {
                        ctx.Set("@", stringitem("xxx"))
                        if c, ok := r.recipes[0].(node); !ok { t.Errorf("recipes[0] '%v' is not node", r.recipes[0]) } else {
                                if k, x := c.tc(), nodeTypeCodeRecipe; k != x { t.Errorf("recipes[1] %v != %v", k, x) }
                                if s, x := c.str(), `@test -f $@.txt && test "$$(cat $@.txt)" = "foo"`; s != x { t.Errorf("recipes[1]: %v != %v", s, x) }
                                if s, x := c.Expand(ctx), `@test -f xxx.txt && test "$(cat xxx.txt)" = "foo"`; s != x { t.Errorf("recipes[1]: %v != %v", s, x) }
                        }
                        ctx.Set("@", stringitem(""))
                }
                if c, ok := r.c.(*checkRuleUpdater); !ok { t.Errorf("wrong type %v", c) } else {
                        if c.checkRule == nil { t.Errorf("nil check rule") } else {
                                if c.checkRule != r { t.Errorf("diverged check rule") }
                                if c.checkRule.c != c { t.Errorf("diverged check rule") }
                        }
                        if n, x := len(r.prev), 1; n != x { t.Errorf("prev: %d != %d", n, x) }
                        if r, ok := r.prev["foo"]; !ok && r == nil { t.Errorf("prev[foo] not defined") } else {
                                if k, x := r.node.tc(), nodeTypeCodeRulePhony; k != x { t.Errorf("%v != %v", k, x) }
                                if n, x := len(r.node.children()), 3; n != x { t.Errorf("children %d != %d", n, x) }
                                if n, x := len(r.targets), 1; n != x { t.Errorf("targets %d != %d", n, x) } else {
                                        if s, x := r.targets[0], "foo"; s != x { t.Errorf("targets[0] %v != %v", s, x) }
                                }
                                if n, x := len(r.prerequisites), 0; n != x { t.Errorf("prerequisites %d != %d", n, x) }
                                if n, x := len(r.recipes), 1; n != x { t.Errorf("recipes %d != %d", n, x) } else {
                                        ctx.Set("@", stringitem("xxx"))
                                        if c, ok := r.recipes[0].(node); !ok { t.Errorf("recipes[0] '%v' is not node", r.recipes[0]) } else {
                                                if k, x := c.tc(), nodeTypeCodeRecipe; k != x { t.Errorf("recipes[1] %v != %v", k, x) }
                                                if s, x := c.str(), `@echo -n foo > $@.txt`; s != x { t.Errorf("recipes[1]: %v != %v", s, x) }
                                                if s, x := c.Expand(ctx), `@echo -n foo > xxx.txt`; s != x { t.Errorf("recipes[1]: %v != %v", s, x) }
                                        }
                                        ctx.Set("@", stringitem(""))
                                }
                                if c, ok := r.c.(*phonyTargetUpdater); !ok { t.Errorf("wrong checker %v", c) } else {
                                }
                        }
                }
        }
        if r, ok := ctx.g.rules["bar"]; !ok && r == nil { t.Errorf("'all' not defined") } else {
                if k, x := r.node.tc(), nodeTypeCodeRulePhony; k != x { t.Errorf("%v != %v", k, x) }
                if n, x := len(r.node.children()), 3; n != x { t.Errorf("children %d != %d", n, x) }
                if n, x := len(r.targets), 1; n != x { t.Errorf("targets %d != %d", n, x) } else {
                        if s, x := r.targets[0], "bar"; s != x { t.Errorf("targets[0] %v != %v", s, x) }
                }
                if n, x := len(r.prerequisites), 0; n != x { t.Errorf("prerequisites %d != %d", n, x) }
                if n, x := len(r.recipes), 1; n != x { t.Errorf("recipes %d != %d", n, x) } else {
                        ctx.Set("@", stringitem("xxx"))
                        if c, ok := r.recipes[0].(node); !ok { t.Errorf("recipes[0] '%v' is not node", r.recipes[0]) } else {
                                if k, x := c.tc(), nodeTypeCodeRecipe; k != x { t.Errorf("recipes[1] %v != %v", k, x) }
                                if s, x := c.str(), `@echo -n bar > $@.txt`; s != x { t.Errorf("recipes[1]: %v != %v", s, x) }
                                if s, x := c.Expand(ctx), `@echo -n bar > xxx.txt`; s != x { t.Errorf("recipes[1]: %v != %v", s, x) }
                        }
                        ctx.Set("@", stringitem(""))
                }
                if c, ok := r.c.(*phonyTargetUpdater); !ok { t.Errorf("wrong type %v", c) } else {
                        if n, x := len(r.prev), 1; n != x { t.Errorf("prev: %d != %d", n, x) }
                        if r, ok := r.prev["bar"]; !ok && r == nil { t.Errorf("prev[foo] not defined") } else {
                                if k, x := r.node.tc(), nodeTypeCodeRuleChecker; k != x { t.Errorf("%v != %v", k, x) }
                                if n, x := len(r.node.children()), 3; n != x { t.Errorf("children %d != %d", n, x) }
                                if n, x := len(r.targets), 1; n != x { t.Errorf("targets %d != %d", n, x) } else {
                                        if s, x := r.targets[0], "bar"; s != x { t.Errorf("targets[0] %v != %v", s, x) }
                                }
                                if n, x := len(r.prerequisites), 0; n != x { t.Errorf("prerequisites %d != %d", n, x) }
                                if n, x := len(r.recipes), 1; n != x { t.Errorf("recipes %d != %d", n, x) } else {
                                        ctx.Set("@", stringitem("xxx"))
                                        if c, ok := r.recipes[0].(node); !ok { t.Errorf("recipes[0] '%v' is not node", r.recipes[0]) } else {
                                                if k, x := c.tc(), nodeTypeCodeRecipe; k != x { t.Errorf("recipes[1] %v != %v", k, x) }
                                                if s, x := c.str(), `@test -f $@.txt && test "$$(cat $@.txt)" = "bar"`; s != x { t.Errorf("recipes[1]: %v != %v", s, x) }
                                                if s, x := c.Expand(ctx), `@test -f xxx.txt && test "$(cat xxx.txt)" = "bar"`; s != x { t.Errorf("recipes[1]: %v != %v", s, x) }
                                        }
                                        ctx.Set("@", stringitem(""))
                                }
                                if c, ok := r.c.(*checkRuleUpdater); !ok { t.Errorf("wrong checker %v", c) } else {
                                        if c.checkRule == nil { t.Errorf("nil check rule") } else {
                                                if c.checkRule != r { t.Errorf("diverged check rule") }
                                                if c.checkRule.c != c { t.Errorf("diverged check rule") }
                                        }
                                        if n, x := len(r.prev), 0; n != x { t.Errorf("prev: %d != %d", n, x) }
                                }
                                if n, x := len(r.prev), 0; n != x { t.Errorf("prev: %d != %d", n, x) }
                        }
                }
        }
        if r, ok := ctx.g.rules["foobar"]; !ok && r == nil { t.Errorf("'all' not defined") } else {
                if k, x := r.node.tc(), nodeTypeCodeRuleSingleColoned; k != x { t.Errorf("%v != %v", k, x) }
                if n, x := len(r.node.children()), 3; n != x { t.Errorf("children %d != %d", n, x) }
                if n, x := len(r.targets), 1; n != x { t.Errorf("targets %d != %d", n, x) } else {
                        if s, x := r.targets[0], "foobar"; s != x { t.Errorf("targets[0] %v != %v", s, x) }
                }
                if n, x := len(r.prerequisites), 2; n != x { t.Errorf("prerequisites %d != %d", n, x) } else {
                        if s, x := r.prerequisites[0], "foo"; s != x { t.Errorf("%s != %s", s, x) }
                        if s, x := r.prerequisites[1], "bar"; s != x { t.Errorf("%s != %s", s, x) }
                }
                if n, x := len(r.recipes), 1; n != x { t.Errorf("recipes %d != %d", n, x) } else {
                        ctx.Set("@", stringitem("xxx"))
                        if c, ok := r.recipes[0].(node); !ok { t.Errorf("recipes[0] '%v' is not node", r.recipes[0]) } else {
                                if k, x := c.tc(), nodeTypeCodeRecipe; k != x { t.Errorf("recipes[1] %v != %v", k, x) }
                                if s, x := c.str(), `@touch $@`; s != x { t.Errorf("recipes[1]: %v != %v", s, x) }
                                if s, x := c.Expand(ctx), `@touch xxx`; s != x { t.Errorf("recipes[1]: %v != %v", s, x) }
                        }
                        ctx.Set("@", stringitem(""))
                }
                if c, ok := r.c.(*defaultTargetUpdater); !ok { t.Errorf("wrong type %v", c) }
        }

        if v, s := info.String(), fmt.Sprintf(``); v != s { t.Errorf("`%s` != `%s`", v, s) }
}

func TestDefineToolset(t *testing.T) {
        wd, e := os.Getwd()
        if e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestDefineToolset", `
### Defining the toolset template
template test

~.modules += $(me.name)

# Defered assignment.
me.out = $(me.dir)/out

$(info $(me.name): $(~.name) - "$(~.modules)")
$(info $(me.name): dir = "$(me.dir)")
$(info $(me.name): source = "$(me.source)")
$(info $(me.name): before-post)
post
$(info $(me.name): after-post) # 'post' is a declaration that we're going to intersect the module 
$(info $(me.name): source = "$(me.source)")

$(me.name)/test:; @echo $@ $(me.source)

$(info $(me.name): before-commit) 
commit
$(info after-commit)

template a
$(info $(me.name): pre - a)
post
$(info $(me.name): post - a)
commit

template b
$(info $(me.name): pre - b)
post
$(info $(me.name): post - b)
commit

template c
$(info $(me.name): pre - c)
post
$(info $(me.name): post - c)
commit

### Using the new toolset template
module a, test, a, b, c
$(info a - $(me.dir),$(me.out))
me.source := a.cpp
commit

$(info a commited)

module b, test, a, b, c
$(info b - $(me.dir),$(me.out))
me.source := b.cpp
commit

$(info b commited)
`);     if err != nil { t.Errorf("parse error:", err) }
        if ctx.modules == nil { t.Errorf("nil modules") } else {
                if m, ok := ctx.modules["a"]; !ok || m == nil { t.Errorf("no module 'a'") } else {
                        if r, ok := m.rules["a/test"]; !ok || r == nil { t.Errorf("no rule 'a/test': %v", m.rules) } else {
                                // TODO: test cases
                        }
                }
                if m, ok := ctx.modules["b"]; !ok || m == nil { t.Errorf("no module 'a'") } else {
                        if r, ok := m.rules["b/test"]; !ok || r == nil { t.Errorf("no rule 'b/test': %v", m.rules) } else {
                                // TODO: test cases
                        }
                }
        }
        if ctx.templates == nil { t.Errorf("nil templates") } else {
                if temp, ok := ctx.templates["test"]; !ok || temp == nil { t.Errorf("no test template") } else {
                        if s, x := temp.name, "test"; s != x { t.Errorf("expects %v but %v", s, x) }
                        if n, x := len(temp.declNodes), 6; n != x { t.Errorf("expects %v but %v", x, n) } else {
                                if c, x := temp.declNodes[0], nodeTypeCodeDefineAppend; c.tc() != x { t.Errorf("expects %v but %v", x, c.tc()) } else {
                                        if s, x := c.str(), "+="; s != x { t.Errorf("expects %v but %v", x, s) }
                                }
                                if c, x := temp.declNodes[1], nodeTypeCodeDefineDeferred; c.tc() != x { t.Errorf("expects %v but %v", x, c.tc()) } else {
                                        if s, x := c.str(), "="; s != x { t.Errorf("expects %v but %v", x, s) }
                                }
                                if c, x := temp.declNodes[2], nodeTypeCodeImmediateText; c.tc() != x { t.Errorf("expects %v but %v", x, c.tc()) } else {
                                        if s, x := c.str(), `$(info $(me.name): $(~.name) - "$(~.modules)")`; s != x { t.Errorf("expects %v but %v", x, s) }
                                }
                                if c, x := temp.declNodes[3], nodeTypeCodeImmediateText; c.tc() != x { t.Errorf("expects %v but %v", x, c.tc()) } else {
                                        if s, x := c.str(), `$(info $(me.name): dir = "$(me.dir)")`; s != x { t.Errorf("expects %v but %v", x, s) }
                                }
                                if c, x := temp.declNodes[4], nodeTypeCodeImmediateText; c.tc() != x { t.Errorf("expects %v but %v", x, c.tc()) } else {
                                        if s, x := c.str(), `$(info $(me.name): source = "$(me.source)")`; s != x { t.Errorf("expects %v but %v", x, s) }
                                }
                                if c, x := temp.declNodes[5], nodeTypeCodeImmediateText; c.tc() != x { t.Errorf("expects %v but %v", x, c.tc()) } else {
                                        if s, x := c.str(), `$(info $(me.name): before-post)`; s != x { t.Errorf("expects %v but %v", x, s) }
                                }
                        }
                        if n, x := len(temp.postNodes), 4; n != x { t.Errorf("expects %v but %v", x, n) } else {
                                if c, x := temp.postNodes[0], nodeTypeCodeImmediateText; c.tc() != x { t.Errorf("expects %v but %v", x, c.tc()) } else {
                                        if s, x := c.str(), `$(info $(me.name): after-post) `; s != x { t.Errorf("expects %v but %v", x, s) }
                                }
                                if c, x := temp.postNodes[1], nodeTypeCodeImmediateText; c.tc() != x { t.Errorf("expects %v but %v", x, c.tc()) } else {
                                        if s, x := c.str(), `$(info $(me.name): source = "$(me.source)")`; s != x { t.Errorf("expects %v but %v", x, s) }
                                }
                                if c, x := temp.postNodes[2], nodeTypeCodeRuleSingleColoned; c.tc() != x { t.Errorf("expects %v but %v", x, c.tc()) } else {
                                        if s, x := c.str(), ":"; s != x { t.Errorf("expects %v but %v", x, s) }
                                        if n, x := len(c.children()), 3; n != x { t.Errorf("%v != %v", n, x) } else {
                                                if c, x := c.child(0), nodeTypeCodeTargets; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                                        if s, x := c.str(), "$(me.name)/test"; s != x { t.Errorf("expects %v but %v", x, s) }
                                                }
                                                if c, x := c.child(1), nodeTypeCodePrerequisites; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                                        if s, x := c.str(), ""; s != x { t.Errorf("expects %v but %v", x, s) }
                                                }
                                                if c, x := c.child(2), nodeTypeCodeRecipes; c.tc() != x { t.Errorf("%v != %v", c.tc(), x) } else {
                                                        if s, x := c.str(), "; @echo $@ $(me.source)\n"; s != x { t.Errorf("expects '%v' but '%v'", x, s) }
                                                }
                                        }
                                }
                                if c, x := temp.postNodes[3], nodeTypeCodeImmediateText; c.tc() != x { t.Errorf("expects %v but %v", x, c.tc()) } else {
                                        if s, x := c.str(), `$(info $(me.name): before-commit) `; s != x { t.Errorf("expects %v but %v", x, s) }
                                }
                        }
                }
        }

        if s, x := info.String(), fmt.Sprintf(`after-commit
a: test - "a"
a: dir = "%s"
a: source = ""
a: before-post
a: pre - a
a: pre - b
a: pre - c
a - %s %s/out
a: after-post
a: source = "a.cpp"
a: before-commit
a: post - a
a: post - b
a: post - c
a commited
b: test - "a b"
b: dir = "%s"
b: source = ""
b: before-post
b: pre - a
b: pre - b
b: pre - c
b - %s %s/out
b: after-post
b: source = "b.cpp"
b: before-commit
b: post - a
b: post - b
b: post - c
b commited
`, wd, wd, wd, wd, wd, wd); s != x { t.Errorf("'%s' != '%s'", s, x) }
}

func TestSpeakSomething(t *testing.T) {
        wd, e := os.Getwd()
        if e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestSpeakSomething", `
s := hello
script = $(speak text,\
--------------------------
echo -n "smart speak - $s"
-------------------------)

#text = $(speak /bin/bash, -c, $(script))
text = $(speak bash, -c, $(script))

$(info $(script))
$(info $(text))
`);     if err != nil { t.Errorf("parse error:", err) }
        if ctx == nil { t.Errorf("nil context") } else {
                if s, ex := ctx.Call("script").Expand(ctx), `echo -n "smart speak - hello"`; s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
                if s, ex := ctx.Call("text").Expand(ctx), `smart speak - hello`; s != ex { t.Errorf("expects '%v' but got '%v'", ex, s) }
        }
        if s, x := info.String(), fmt.Sprintf(`echo -n "smart speak - hello"
smart speak - hello
`); s != x { t.Errorf("'%s' != '%s'", s, x) }
}

func TestPatternRules(t *testing.T) {
        if wd, e := os.Getwd(); e != nil || workdir != wd { t.Errorf("%v != %v (%v)", workdir, wd, e) }

        info, f := new(bytes.Buffer), builtinInfoFunc; defer func(){ builtinInfoFunc = f }()
        builtinInfoFunc = func(ctx *Context, args Items) {
                fmt.Fprintf(info, "%v\n", args.Expand(ctx))
        }

        ctx, err := newTestContext("TestPatternRules", `
%.o: %.c
	gcc -o $@ $<
%.txt %.log:
	echo $*
`);     if err != nil { t.Errorf("parse error:", err) }
        if n, x := len(ctx.g.rules), 0; n != x { t.Errorf("wrong rules: %v", ctx.g.rules) }
        if n, x := len(ctx.g.patts), 3; n != x { t.Errorf("wrong rules: %v", ctx.g.patts) } else {
                if r, ok := ctx.g.patts["%.o"]; !ok { t.Errorf("rule not defined: %v", ctx.g.patts) } else {
                        if r.kind != rulePercentPattern { t.Errorf("%v != %v", r.kind, rulePercentPattern) }
                        if n := len(r.targets); n != 1 { t.Errorf("incorrect number of targets: %v %v", n, r.targets) }
                        if n := len(r.prerequisites); n != 1 { t.Errorf("incorrect number of prerequisites: %v %v", n, r.prerequisites) }
                        if n := len(r.recipes); n != 1 { t.Errorf("incorrect number of recipes: %v %v", n, r.recipes) }
                }
                if r, ok := ctx.g.patts["%.txt"]; !ok { t.Errorf("rule not defined: %v", ctx.g.patts) } else {
                        if r.kind != rulePercentPattern { t.Errorf("%v != %v", r.kind, rulePercentPattern) }
                        if n := len(r.targets); n != 2 { t.Errorf("incorrect number of targets: %v %v", n, r.targets) }
                        if n := len(r.prerequisites); n != 0 { t.Errorf("incorrect number of prerequisites: %v %v", n, r.prerequisites) }
                        if n := len(r.recipes); n != 1 { t.Errorf("incorrect number of recipes: %v %v", n, r.recipes) }
                }
                if r, ok := ctx.g.patts["%.log"]; !ok { t.Errorf("rule not defined: %v", ctx.g.patts) } else {
                        if r.kind != rulePercentPattern { t.Errorf("%v != %v", r.kind, rulePercentPattern) }
                        if n := len(r.targets); n != 2 { t.Errorf("incorrect number of targets: %v %v", n, r.targets) }
                        if n := len(r.prerequisites); n != 0 { t.Errorf("incorrect number of prerequisites: %v %v", n, r.prerequisites) }
                        if n := len(r.recipes); n != 1 { t.Errorf("incorrect number of recipes: %v %v", n, r.recipes) }
                }
        }
        if mrs := ctx.g.findMatchRules(ctx, "a.o"); len(mrs) == 0 { t.Errorf("`%v`", mrs) } else {
                if len(mrs) != 1 { t.Errorf("`%v`", mrs) } else { mr := mrs[0]
                        if s, x := mr.match.target, "a.o"; s != x { t.Errorf("%v != %v", s, x) }
                        if s, x := mr.match.stem, "a"; s != x { t.Errorf("%v != %v", s, x) }
                }
        }
        if mrs := ctx.g.findMatchRules(ctx, "foo.o"); len(mrs) == 0 { t.Errorf("`%v`", mrs) } else {
                if len(mrs) != 1 { t.Errorf("`%v`", mrs) } else { mr := mrs[0]
                        if s, x := mr.match.target, "foo.o"; s != x { t.Errorf("%v != %v", s, x) }
                        if s, x := mr.match.stem, "foo"; s != x { t.Errorf("%v != %v", s, x) }
                }
        }
        if mrs := ctx.g.findMatchRules(ctx, "foo.log"); len(mrs) == 0 { t.Errorf("`%v`", mrs) } else {
                if len(mrs) != 1 { t.Errorf("`%v`", mrs) } else { mr := mrs[0]
                        if s, x := mr.match.target, "foo.log"; s != x { t.Errorf("%v != %v", s, x) }
                        if s, x := mr.match.stem, "foo"; s != x { t.Errorf("%v != %v", s, x) }
                }
        }
        if mrs := ctx.g.findMatchRules(ctx, "foo.txt"); len(mrs) == 0 { t.Errorf("`%v`", mrs) } else {
                if len(mrs) != 1 { t.Errorf("`%v`", mrs) } else { mr := mrs[0]
                        if s, x := mr.match.target, "foo.txt"; s != x { t.Errorf("%v != %v", s, x) }
                        if s, x := mr.match.stem, "foo"; s != x { t.Errorf("%v != %v", s, x) }
                }
        }
        if mrs := ctx.g.findMatchRules(ctx, "foo.c"); 0 < len(mrs) { t.Errorf("`%v`", mrs) }
        if v, s := info.String(), fmt.Sprintf(``); v != s { t.Errorf("`%s` != `%s`", v, s) }
}
