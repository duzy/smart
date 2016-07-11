//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "bytes"
        "errors"
        "fmt"
        "unicode"
        "unicode/utf8"
        "io/ioutil"
        "os"
        "os/exec"
        //"path/filepath"
        //"reflect"
        "strings"
        "path/filepath"
        "github.com/duzy/worker"
)

type Item interface {
        // Expand the item to string
        Expand(ctx *Context) string

        // Check if the item is empty (including all spaces)
        IsEmpty(ctx *Context) bool
}

type Items []Item

func (is Items) Len() int { return len(is) }
func (is Items) IsEmpty(ctx *Context) bool {
        if 0 < len(is) { return false }
        for _, i := range is {
                if !i.IsEmpty(ctx) { return false }
        }
        return true
}

func (is Items) Expand(ctx *Context) string { return is.Join(ctx, " ") }
func (is Items) Join(ctx *Context, sep string) string {
        b := new(bytes.Buffer)
        for i, a := range is {
                if s := a.Expand(ctx); s != "" {
                        if i == 0 {
                                fmt.Fprint(b, s)
                        } else {
                                fmt.Fprintf(b, "%s%s", sep, s)
                        }
                }
        }
        return b.String()
}

func (is Items) Concat(ctx *Context, args ...Item) (res Items) {
        for _, a := range is {
                if !a.IsEmpty(ctx) {
                        res = append(res, a)
                }
        }
        for _, a := range args {
                if !a.IsEmpty(ctx) {
                        res = append(res, a)
                }
        }
        return
}

func (is *Items) Append(args ...Item) *Items {
        *is = append(*is, args...)
        return is
}

func (is *Items) AppendString(args ...string) *Items {
        for _, s := range args {
                *is = append(*is, stringitem(s))
        }
        return is
}

type define struct {
        name string
        value Items
        readonly bool
        loc location
}

type rulekind int

const (
        ruleFileTarget rulekind = iota
        rulePercentPattern
        ruleRegexPattern // *
        ruleGlobPattern // *
)

type rule struct {
        prev map[string]*rule // previously defined rules of a specific target
        targets []string // expanded strings (target names or patterns)
        prerequisites []string // expanded strings (could be patterns)
        recipes []interface{} // *node, string
        ns namespace
        c checkupdater
        node *node
        kind rulekind
}

type checkupdater interface {
        check(ctx *Context, r *rule, m *match) bool
        update(ctx *Context, r *rule, m *match) bool
}

type namespace interface {
        getNamespace(name string) namespace
        getDefineMap() map[string]*define
        findMatchRules(ctx *Context, target string) (mrs []*matchrule)
        isPhonyTarget(ctx *Context, target string) bool
        saveDefines(names ...string) (saveIndex int, m map[string]*define)
        restoreDefines(saveIndex int)
        set(ctx *Context, ids []string, items ...Item)
        getRules(kind nodeType, target string) (rules []*rule)
        getGoal() (target string)
        setGoal(target string)
        link(targets ...string) (r *rule)
}

type namespaceEmbed struct {
        defines map[string]*define
        saveList []map[string]*define // saveDefines, restoreDefines
        rules map[string]*rule
        patts map[string]*rule
        pattList []*rule
        goal string
}
func (ns *namespaceEmbed) getGoal() string { return ns.goal }
func (ns *namespaceEmbed) setGoal(target string) { ns.goal = target }
func (ns *namespaceEmbed) getRules(kind nodeType, target string) (rules []*rule) {
        for ru, ok := ns.rules[target]; ok && ru != nil; {
                if ru.node.kind == kind {
                        rules = append(rules, ru)
                }
                ru, ok = ru.prev[target]
        }
        return
}
func (ns *namespaceEmbed) link(targets ...string) (r *rule) {
        r = &rule{ ns:ns, targets:targets }
        for i, target := range targets {
                var prev *rule

                switch {
                case strings.Contains(target, "%"):
                        prev, _ = ns.patts[target]
                        r.kind = rulePercentPattern
                default:
                        prev, _ = ns.rules[target]
                        r.kind = ruleFileTarget
                }
                
                if prev != nil {
                        if r.prev == nil {
                                r.prev = make(map[string]*rule)
                        }
                        r.prev[target] = prev
                }

                switch r.kind {
                case ruleFileTarget: 
                        ns.rules[target] = r
                case rulePercentPattern: fallthrough
                case ruleRegexPattern:   fallthrough
                case ruleGlobPattern:
                        ns.patts[target] = r
                        if i == 0 {
                                ns.pattList = append(ns.pattList, r)
                        }
                }
        }
        return
}

func (ns *namespaceEmbed) saveDefines(names ...string) (saveIndex int, m map[string]*define) {
        var ok bool
        m = make(map[string]*define, len(names))
        for _, name := range names {
                m[name], ok = ns.defines[name]
                if ok { delete(ns.defines, name) }
        }
        saveIndex = len(ns.saveList)
        ns.saveList = append(ns.saveList, m)
        return
}
func (ns *namespaceEmbed) restoreDefines(saveIndex int) {
        m := ns.saveList[saveIndex]
        ns.saveList = ns.saveList[0:saveIndex]
        for name, d := range m {
                if d == nil {
                        delete(ns.defines, name)
                } else {
                        ns.defines[name] = d
                }
        }
}

func (ns *namespaceEmbed) set(ctx *Context, ids []string, items ...Item) {
        if n := len(ids); n == 1 {
                name := ids[0]
                if d, ok := ns.defines[name]; ok && d != nil {
                        d.value = items
                } else {
                        ns.defines[name] = &define{ loc:ctx.CurrentLocation(), name:name, value:items }
                }
        } else {
                lineno, colno := ctx.l.caculateLocationLineColumn(ctx.l.location())
                fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: nested referencing\n", ctx.l.scope, lineno, colno)

                // FIXME: nested
        }
}

func (ns *namespaceEmbed) getNamespace(name string) namespace {
        //lineno, colno := ctx.l.caculateLocationLineColumn(ctx.l.location())
        //fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: nesting reference '%s'\n", ctx.l.scope, lineno, colno, name)
        return nil
}

func (ns *namespaceEmbed) getDefineMap() map[string]*define {
        return ns.defines
}

func (ns *namespaceEmbed) findMatchRules(ctx *Context, target string) (matchrules []*matchrule) {
        if r, ok := ns.rules[target]; ok && r != nil {
                if m, ok := r.match(target); ok && m != nil {
                        matchrules = append(matchrules, &matchrule{ m, r })
                }
        } else {
                //fmt.Printf("findMatchRules: %v\n", ns.pattList)
                for _, r := range ns.pattList {
                        if m, ok := r.match(target); ok && m != nil {
                                //fmt.Printf("findMatchRules: %p %v %v %v\n", r, r.targets, target, *m)
                                matchrules = append(matchrules, &matchrule{ m, r })
                        }
                }
        }
        return
}

func (ns *namespaceEmbed) isPhonyTarget(ctx *Context, target string) bool {
        if rr, ok := ns.rules[target]; ok && rr != nil {
                return rr.node.kind == nodeRulePhony
        }
        return false
}

type nodeType int

const (
        nodeComment nodeType = iota
        nodeEscape
        nodeDeferredText
        nodeImmediateText
        nodeName
        nodeNamePrefix          // :
        nodeNamePart            // .
        nodeArg
        nodeValueText           // value text for defines (TODO: use it)
        nodeDefineDeferred      //  =     deferred
        nodeDefineQuestioned    // ?=     deferred
        nodeDefineSingleColoned // :=     immediate
        nodeDefineDoubleColoned // ::=    immediate
        nodeDefineNot           // !=     immediate
        nodeDefineAppend        // +=     deferred or immediate (parsed into deferred)
        nodeRuleSingleColoned   // :
        nodeRuleDoubleColoned   // ::
        nodeRulePhony           // :!:    phony target
        nodeRuleChecker         // :?:    check if the target is updated
        nodeTargets
        nodePrerequisites
        nodeRecipes
        nodeRecipe
        nodeCall
        nodeSpeak               // $(speak dialect, ...)
        nodeInclude             // include filename
        nodeTemplate            // template name, parameters
        nodeModule              // module name, temp, parameters
        nodeCommit              // commit
        nodePost                // post
        nodeUse                 // use name
)

var (
        statements = map[string]nodeType{
                "include":      nodeInclude,
                "template":     nodeTemplate,
                "module":       nodeModule,
                "commit":       nodeCommit,
                "post":         nodePost,
                "use":          nodeUse,
                // TODO: template...endtempl
                // TODO: module...endmod
        }

        processors = map[nodeType]func(ctx *Context, n *node)(err error){
                nodeComment:                    processNodeComment,
                nodeImmediateText:              processNodeImmediateText,
                nodeCall:                       processNodeCall,
                nodeDefineQuestioned:           processNodeDefineQuestioned,
                nodeDefineDeferred:             processNodeDefineDeferred,
                nodeDefineSingleColoned:        processNodeDefineSingleColoned,
                nodeDefineDoubleColoned:        processNodeDefineDoubleColoned,
                nodeDefineAppend:               processNodeDefineAppend,
                nodeDefineNot:                  processNodeDefineNot,
                nodeRulePhony:                  processNodeRule,
                nodeRuleChecker:                processNodeRule,
                nodeRuleDoubleColoned:          processNodeRule,
                nodeRuleSingleColoned:          processNodeRule,
                nodeInclude:                    processNodeInclude,
                nodeTemplate:                   processNodeTemplate,
                nodeModule:                     processNodeModule,
                nodeCommit:                     processNodeCommit,
                //nodePost:                     
                nodeUse:                        processNodeUse,
        }

        /*
        Variable definitions are parsed as follows:

        immediate = deferred
        immediate ?= deferred
        immediate := immediate
        immediate ::= immediate
        immediate += deferred or immediate
        immediate != immediate

        The directives define/endef are not supported.
        */
        nodeTypeNames = []string {
                nodeComment:                    "comment",
                nodeEscape:                     "escape",
                nodeDeferredText:               "deferred-text",
                nodeImmediateText:              "immediate-text",
                nodeName:                       "name",
                nodeNamePrefix:                 "name-prefix",
                nodeNamePart:                   "name-part",
                nodeArg:                        "arg",
                nodeValueText:                  "value-text",
                nodeDefineDeferred:             "define-deferred",
                nodeDefineQuestioned:           "define-questioned",
                nodeDefineSingleColoned:        "define-single-coloned",
                nodeDefineDoubleColoned:        "define-double-coloned",
                nodeDefineNot:                  "define-not",
                nodeDefineAppend:               "define-append",
                nodeRuleSingleColoned:          "rule-single-coloned",
                nodeRuleDoubleColoned:          "rule-double-coloned",
                nodeRulePhony:                  "rule-phony",
                nodeRuleChecker:                "rule-checker",
                nodeTargets:                    "targets",
                nodePrerequisites:              "prerequisites",
                nodeRecipes:                    "recipes",
                nodeRecipe:                     "recipe",
                nodeCall:                       "call",
                nodeSpeak:                      "speak",
                nodeInclude:                    "include",
                nodeTemplate:                   "template",
                nodeModule:                     "module",
                nodeCommit:                     "commit",
                nodePost:                       "post",
                nodeUse:                        "use",
        }
)

func (k nodeType) String() string {
        return nodeTypeNames[int(k)]
}

type location struct {
        offset, end int // (node.pos, node.end)
}

type stringitem string

func (si stringitem) Expand(ctx *Context) string { return string(si) }
func (si stringitem) IsEmpty(ctx *Context) bool { return string(si) == "" }

func StringItem(s string) stringitem { return stringitem(s) }
func StringItems(ss ...string) (a Items) {
        for _, s := range ss { 
                a.AppendString(s)
        }
        return
}

// flatitem is a expanded string with a location
type flatitem struct {
        s string
        l location
}

func (fi *flatitem) Expand(ctx *Context) string { return fi.s }
func (fi *flatitem) IsEmpty(ctx *Context) bool { return fi.s == "" }

type node struct {
        l *lex
        kind nodeType
        children []*node
        pos, end int
}

func (n *node) Expand(ctx *Context) (s string) {
        var is Items
        if nodeDefineDeferred <= n.kind && n.kind <= nodeDefineAppend {
                is = ctx.nodeItems(n.children[1])
        } else {
                is = ctx.nodeItems(n)
        }
        return is.Expand(ctx)
}

func (n *node) IsEmpty(ctx *Context) bool {
        if len(n.children) == 0 { return true }
        return n.Expand(ctx) == ""
}

func (n *node) len() int {
        return n.end - n.pos
}

func (n *node) str() (s string) {
        if a, b := n.pos, n.end; a < b { s = string(n.l.s[a:b]) }
        return
}

func (n *node) loc() location {
        return location{n.pos, n.end}
}

type parseBuffer struct {
        scope string // file or named scope
        s []byte // the content of the file
}

func (p *parseBuffer) caculateLocationLineColumn(loc location) (lineno, colno int) {
        for i, end := 0, len(p.s); i < loc.offset && i < end; {
                r, l := utf8.DecodeRune(p.s[i:])
                switch {
                case r == '\n':
                        lineno, colno = lineno + 1, 0
                case 0 < l:
                        colno += l
                }
                i += l
        }
        lineno++ // started from 1
        colno++  // started from 1
        return
}

type lexStack struct {
        node *node
        state func()
        code int
        delm rune // delimeter
}
type lex struct {
        *parseBuffer
        pos int // the current read position

        rune rune // the rune last time returned by getRune
        runeLen int // the size in bytes of the rune last returned by getRune
        
        stack []*lexStack
        step func ()

        nodes []*node // parsed top level nodes
}

func (l *lex) location() location {
        return location{ l.pos, l.pos }
}

func (l *lex) getLineColumn() (lineno, colno int) {
        lineno, colno = l.caculateLocationLineColumn(l.location())
        return 
}

func (l *lex) peek() (r rune) {
        if l.pos < len(l.s) {
                r, _ = utf8.DecodeRune(l.s[l.pos:])
        }
        return
}

func (l *lex) peekN(n int) (rs []rune) {
        for pos, end := l.pos, len(l.s); 0 < n; n-- {
                if pos < end {
                        r, l := utf8.DecodeRune(l.s[pos:])
                        rs, pos = append(rs, r), pos+l
                }
        }
        return
}

func (l *lex) lookat(s string, pp *int) bool {
        end := len(l.s)
        for _, sr := range s {
                if *pp < end {
                        r, n := utf8.DecodeRune(l.s[*pp:])
                        *pp = *pp + n
                        
                        if sr != r {
                                return false
                        }
                }
        }
        return true
}

func (l *lex) looking(s string, pp *int) bool {
        if l.rune == rune(s[0]) {
                return l.lookat(s[1:], pp)
        }
        return false
}

func (l *lex) lookingInlineSpaces(pp *int) bool {
        beg, end := *pp, len(l.s)
        for *pp < end {
                if r, n := utf8.DecodeRune(l.s[*pp:]); r == '\n' || r == '#' {
                        return true
                } else if unicode.IsSpace(r) {
                        *pp = *pp + n
                } else {
                        break
                }
        }
        return beg < *pp
}

func (l *lex) get() bool {
        if len(l.s) == l.pos {
                if l.rune != rune(0) && 0 < l.runeLen {
                        l.rune, l.runeLen = 0, 0
                        return true
                }
                return false
        }
        if len(l.s) < l.pos { errorf("over reading (at %v)", l.pos) }

        l.rune, l.runeLen = utf8.DecodeRune(l.s[l.pos:])
        l.pos = l.pos+l.runeLen
        switch {
        case l.rune == 0:
                return false //errorf(-2, "zero reading (at %v)", l.pos)
        case l.rune == utf8.RuneError:
                errorf("invalid UTF8 encoding")
        }
        return true
}

func (l *lex) unget() {
        switch {
        case l.rune == 0:
                errorf("wrong invocation of unget")
        case l.pos == 0:
                errorf("get to the beginning of the bytes")
        case l.pos < 0:
                errorf("get to the front of beginning of the bytes")
                //case l.lineno == 1 && l.colno <= 1: return
        }
        /*
        if l.rune == '\n' {
                l.lineno, l.colno, l.prevColno = l.lineno-1, l.prevColno, 0
        } else {
                l.colno--
        } */
        // assert(utf8.RuneLen(l.rune) == l.runeLen)
        l.pos, l.rune, l.runeLen = l.pos-l.runeLen, 0, 0
        return
}

func (l *lex) new(t nodeType) *node {
        return &node{ l:l, kind:t, pos:l.pos, end:l.pos }
}

func (l *lex) push(t nodeType, ns func(), c int) *lexStack {
        ls := &lexStack{ node:l.new(t), state:l.step, code:c }
        l.stack, l.step = append(l.stack, ls), ns
        return ls
}

func (l *lex) pop() *lexStack {
        if i := len(l.stack)-1; 0 <= i {
                st := l.stack[i]
                l.stack, l.step = l.stack[0:i], st.state
                return st
        }
        return nil
}

func (l *lex) top() *lexStack {
        if i := len(l.stack)-1; 0 <= i {
                return l.stack[i]
        }
        return nil
}

func (l *lex) backwardNonSpace(beg, i int) int {
        for beg < i {
                r, l := utf8.DecodeLastRune(l.s[0:i])
                if unicode.IsSpace(r) {
                        i -= l
                } else {
                        break
                }
        }
        return i
}

func (l *lex) forwardNonSpaceInline(i int) int {
        for e := len(l.s); i < e; {
                r, l := utf8.DecodeRune(l.s[i:])
                if r != '\n' && unicode.IsSpace(r) {
                        i += l
                } else {
                        break
                }
        }
        return i
}

func (l *lex) stateAppendNode() {
        t := l.pop().node

        /*
        fmt.Printf("AppendNode: %v: '%v' %v '%v' (%v, %v, %v)\n", t.kind,
                t.children[0].str(), t.str(), t.children[1].str(), len(l.stack), l.pos, l.rune) //*/

        // Pop out and append the node.
        l.nodes = append(l.nodes, t)
}

func (l *lex) stateGlobal() {
state_loop:
        for l.get() {
                switch {
                case l.rune == '#':
                        st := l.push(nodeComment, l.stateComment, 0)
                        st.node.pos-- // for the '#'
                        break state_loop
                case l.rune != rune(0) && !unicode.IsSpace(l.rune):
                        l.unget() // Put back the rune.
                        l.push(nodeImmediateText, l.stateLineHeadText, 0)
                        break state_loop
                }
        }
}

func (l *lex) stateComment() {
state_loop:
        for l.get() {
                switch {
                case l.rune == '\\':
                        if l.peek() == '\n' {
                                l.get() // continual comment line
                        }

                case l.rune == '\n':
                        if l.peek() == '#' {
                                break // assemply continual comment lines in one node
                        }
                        fallthrough
                case l.rune == rune(0): // end of string
                        st := l.pop()
                        st.node.end = l.pos
                        if l.rune == '\n' {
                                st.node.end-- // exclude the '\n'
                        }

                        /*
                        lineno, colno := l.caculateLocationLineColumn(st.node.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: stateComment: (stack=%v) %v\n", l.scope, lineno, colno, len(l.stack), st.node.str()) //*/

                        if 0 < len(l.stack) {
                                c := st.node
                                st = l.top()
                                st.node.children = append(st.node.children, c)
                        } else {
                                l.nodes = append(l.nodes, st.node) // append the comment node
                        }
                        break state_loop
                }
        }
}

// stateLineHeadText process line-head text
func (l *lex) stateLineHeadText() {
        st := l.top()
state_loop:
        for l.get() {
                if st.code == 0 {
                        for s, t := range statements {
                                if pos := l.pos; l.looking(s, &pos) {
                                        //fmt.Printf("stateLineHeadText: %v (%v)\n", string(l.rune), s)
                                        if ss := pos; l.lookingInlineSpaces(&ss) {
                                                st.node.kind, st.node.end, l.pos = t, pos, ss
                                                //fmt.Printf("looked: %v (%v): '%v' '%v'\n", s, t, string(l.s[pos:ss]), string(l.s[ss]))
                                                if r := l.peek(); r == '\n' || r == '#' {
                                                        l.pop() // end of statement
                                                        l.nodes = append(l.nodes, st.node)
                                                } else {
                                                        //fmt.Printf("stateLineHeadText: %v (%v)\n", st.node.kind, st.node.str())
                                                        l.push(nodeArg, l.stateStatementArg, 0)
                                                }
                                                break state_loop
                                        }
                                }
                        }
                }

                switch {
                case l.rune == '$':
                        st := l.push(nodeCall, l.stateDollar, 0)
                        st.node.pos-- // for the '$'
                        break state_loop

                case l.rune == '=':
                        l.top().code = int(nodeDefineDeferred)
                        l.step = l.stateDefine
                        break state_loop

                case l.rune == '?':
                        if l.peek() == '=' {
                                l.get() // consume the '=' for '?='
                                l.top().code = int(nodeDefineQuestioned)
                                l.step = l.stateDefine
                                break state_loop
                        }

                case l.rune == '!':
                        if l.peek() == '=' {
                                l.get() // consume the '=' for '!='
                                l.top().code = int(nodeDefineNot)
                                l.step = l.stateDefine
                                break state_loop
                        }

                case l.rune == '+':
                        if l.peek() == '=' {
                                l.get() // consume the '=' for '+='
                                l.top().code = int(nodeDefineAppend)
                                l.step = l.stateDefine
                                break state_loop
                        }

                case l.rune == ':':
                        if r := l.peek(); r == '=' {
                                l.get() // consume the '=' for ':='
                                l.top().code = int(nodeDefineSingleColoned)
                                l.step = l.stateDefine
                        } else {
                                n := l.top().node
                                n.end = l.backwardNonSpace(n.pos, l.pos-1)
                                l.step = l.stateRule
                        }
                        break state_loop

                case l.rune == '.':
                        part := l.new(nodeNamePart)
                        part.pos = l.pos - 1
                        st := l.top()
                        st.node.children = append(st.node.children, part)

                case l.rune == '#': fallthrough
                case l.rune == '\n':
                        st := l.pop() // pop out the node
                        st.node.end = l.pos-1

                        // append the island text
                        l.nodes = append(l.nodes, st.node)

                        if l.rune == '#' {
                                st = l.push(nodeComment, l.stateComment, 0)
                                st.node.pos-- // for the '#'
                        }
                        break state_loop

                default: l.escapeTextLine(l.top().node)
                }
                
                if st.code == 0 {
                        st.code = 1 // 1 indicates not the first char anymore
                }
        }
}

func (l *lex) stateStatementArg() {
        st := l.top() // Must be a nodeArg
        //fmt.Printf("statement: %v: %v\n", st.node.kind, string(l.s[l.pos:]))
state_loop:
        for l.get() {
                if st.code == 0 {
                        if l.rune != '\n' && unicode.IsSpace(l.rune) {
                                continue
                        } else {
                                st.node.pos = l.pos - 1
                        }
                }
                
                switch {
                case l.rune == '$':
                        l.push(nodeCall, l.stateDollar, 0).node.pos-- // 'pos--' for the '$'
                        break state_loop
                case l.rune == '\\':
                        l.escapeTextLine(st.node)
                case l.rune == ',': fallthrough
                case l.rune == '\n':
                        arg := st.node
                        arg.end = l.pos - 1
                        l.pop()

                        st = l.top()
                        st.node.children = append(st.node.children, arg)
                        if l.rune == '\n' {
                                l.pop() // end of statement
                                l.nodes = append(l.nodes, st.node)
                                //fmt.Printf("%v: %v %v\n", st.node.kind, st.node.str(), st.node.children)
                        } else {
                                l.push(nodeArg, l.stateStatementArg, 0)
                        }
                        break state_loop
                }

                if st.code == 0 {
                        st.code = 1
                }                
        }
        //fmt.Printf("Statement: %v: %v (%v)\n", st.node.kind, st.node.str(), st.node.children)
}

func (l *lex) stateDefine() {
        st := l.pop() // name

        var (
                name, t, n = st.node, nodeType(st.code), 2
                vt nodeType
        )
        switch t {
        case nodeDefineDoubleColoned: n = 3; fallthrough
        case nodeDefineSingleColoned:        fallthrough
        case nodeDefineNot:          vt = nodeImmediateText
        case nodeDefineDeferred:      n = 1; fallthrough
        default:                     vt = nodeDeferredText
        }

        name.end = l.backwardNonSpace(st.node.pos, l.pos-n) // for '=', '+=', '?=', ':=', '::='

        st = l.push(t, l.stateAppendNode, 0)
        st.node.children = []*node{ name }
        st.node.pos -= n // for '=', '+=', '?=', ':='

        // Create the value node.
        value := l.push(vt, l.stateDefineTextLine, 0).node
        st.node.children = append(st.node.children, value)
}

func (l *lex) stateDefineTextLine() {
        st := l.top()
state_loop:
        for l.get() {
                if st.code == 0 { // skip spaces after '='
                        if !unicode.IsSpace(l.rune) {
                                st.node.pos, st.code = l.pos-1, 1
                        } else if l.rune != '\n' /* IsSpace */ {
                                st.node.pos = l.pos
                                continue
                        }
                }

                switch {
                case l.rune == '$':
                        st = l.push(nodeCall, l.stateDollar, 0)
                        st.node.pos-- // for the '$'
                        break state_loop

                case l.rune == '#':
                        l.unget() // Put back the '#', then fall through.
                        fallthrough
                case l.rune == '\n': fallthrough
                case l.rune == rune(0): // The end of string.
                        st.node.end = l.pos
                        if l.rune == '\n' {
                                st.node.end-- // Exclude the '\n'.
                        }

                        l.pop() // Pop out the value node and forward to the define node.
                        break state_loop

                default: l.escapeTextLine(l.top().node)
                }
        }
}

func (l *lex) escapeTextLine(t *node) {
        if l.rune != '\\' { return }

        // Escape: \\n \#
        if l.get() { // get the char right next to '\\'
                switch l.rune {
                case '#': fallthrough
                case '\n':
                        en := l.new(nodeEscape)
                        en.pos -= 2 // for the '\\\n', '\\#', etc.
                        if l.rune == '\n' {
                                /* FIXME: skip spaces after '\\\n' ?
                                for unicode.IsSpace(l.peek()) {
                                        l.get()
                                        en.end = l.pos
                                } */
                        }
                        t.children = append(t.children, en)
                }
        }
}

func (l *lex) stateRule() {
        rs, t, n := l.peekN(2), nodeRuleSingleColoned, 1 // Assuming single colon.

        if len(rs) == 2 {
                switch {
                case rs[0] == ':': // targets :: blah blah blah
                        l.get() // drop the second ':'
                        t, n = nodeRuleDoubleColoned, 2
                        if rs[1] == '=' { // ::=
                                l.get() // consume the '=' for '::='
                                l.top().code = int(nodeDefineDoubleColoned)
                                l.step = l.stateDefine
                                return
                        }
                case rs[0] == '!' && rs[1] == ':': // targets :!:
                        l.get(); l.get() // drop the "!:"
                        t, n = nodeRulePhony, 3
                case rs[0] == '?' && rs[1] == ':': // targets :?:
                        l.get(); l.get() // drop the "?:"
                        t, n = nodeRuleChecker, 3
                }
        }

        targets := l.pop().node
        targets.kind = nodeTargets

        st := l.push(t, l.stateAppendNode, 0)
        st.node.children = []*node{ targets }
        st.node.pos -= n // for the ':', '::', ':!:', ':?:'

        prerequisites := l.push(nodePrerequisites, l.stateRuleTextLine, 0).node
        st.node.children = append(st.node.children, prerequisites)

        /*
        lineno, colno := l.caculateLocationLineColumn(st.node.loc())
        fmt.Fprintf(os.Stderr, "%v:%v:%v: stateRule: %v\n", l.scope, lineno, colno, st.node.children[0].str()) //*/
}

func (l *lex) stateRuleTextLine() {
        st := l.top()
state_loop:
        for l.get() {
                if st.code == 0 && (l.rune == '\n' || !unicode.IsSpace(l.rune)) { // skip spaces after ':' or '::'
                        st.node.pos, st.code = l.pos-1, 1
                }

                switch {
                case l.rune == '$':
                        st = l.push(nodeCall, l.stateDollar, 0)
                        st.node.pos-- // for the '$'
                        break state_loop

                case l.rune == '#':  fallthrough
                case l.rune == ';':  fallthrough
                case l.rune == '\n': fallthrough
                case l.rune == rune(0): // end of string
                        st.node.end = l.pos
                        if l.rune != rune(0) {
                                st.node.end-- // exclude the '\n' or '#'
                        }

                        /*
                        lineno, colno := l.caculateLocationLineColumn(st.node.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: stateRuleTextLine: %v\n", l.scope, lineno, colno, st.node.str()) //*/

                        st = l.pop() // pop out the prerequisites node
                        switch l.rune {
                        case '#':
                                st = l.push(nodeComment, l.stateComment, 0)
                                st.node.pos-- // for the '#'
                        case ';':
                                st.node.end = l.backwardNonSpace(st.node.pos, l.pos-1)
                                recipes := l.push(nodeRecipes, l.stateTabbedRecipes, 0).node
                                recipes.pos-- // includes ';'

                                l.pos = l.forwardNonSpaceInline(l.pos)
                                l.push(nodeRecipe, l.stateRecipe, 0)
                        case '\n':
                                if p := l.peek(); p == '\t' || p == '#' {
                                        st = l.push(nodeRecipes, l.stateTabbedRecipes, 0)
                                }
                        }
                        break state_loop

                default: l.escapeTextLine(l.top().node)
                }
        }
}

func (l *lex) stateTabbedRecipes() { // tab-indented action of a rule
        if st := l.top(); l.get() {
                switch {
                case l.rune == '\t':
                        st = l.push(nodeRecipe, l.stateRecipe, 0)
                        //st.node.pos-- // for the '\t'

                case l.rune == '#':
                        st = l.push(nodeComment, l.stateComment, 0)
                        st.node.pos-- // for the '#'

                        /*
                        lineno, colno := l.caculateLocationLineColumn(st.node.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: stateTabbedRecipes: %v (%v, stack=%v)\n", l.scope, lineno, colno, st.node.str(), l.top().node.kind, len(l.stack)) //*/

                default:
                        recipes := st.node // the recipes node
                        recipes.end = l.pos
                        if l.rune == '\n' {
                                recipes.end--
                        } else if l.rune != rune(0) {
                                recipes.end--
                                l.unget() // put back the non-space character following by a recipe
                        }

                        st = l.pop() // pop out the recipes
                        st = l.top() // the rule node
                        st.node.children = append(st.node.children, recipes)

                        /*
                        lineno, colno := l.caculateLocationLineColumn(st.node.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: stateTabbedRecipes: %v (%v)\n", l.scope, lineno, colno, st.node.str(), l.top().node.kind) //*/
                }
        }
}

func (l *lex) stateRecipe() { // tab-indented action of a rule
        st := l.top()
state_loop:
        for l.get() {
                switch {
                case l.rune == '$':
                        st = l.push(nodeCall, l.stateDollar, 0)
                        st.node.pos-- // for the '$'
                        break state_loop
                case l.rune == '\n': fallthrough
                case l.rune == rune(0): // end of string
                        recipe := st.node
                        recipe.end = l.pos
                        if l.rune != rune(0) {
                                recipe.end-- // exclude the '\n'
                        }

                        l.pop() // pop out the node

                        st = l.top()
                        st.node.children = append(st.node.children, recipe)

                        //fmt.Printf("recipe: (%v) %v\n", st.node.kind, recipe.str())

                        /*
                        lineno, colno := l.caculateLocationLineColumn(a.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: stateRecipe: %v (of %v)\n", l.scope, lineno, colno, a.str(), st.node.kind) //*/
                        break state_loop
                }
        }
}

func (l *lex) stateDollar() {
        if l.get() {
                switch {
                case l.rune == '(': l.push(nodeName, l.stateCallName, 0).delm = ')'
                case l.rune == '{': l.push(nodeName, l.stateCallName, 0).delm = '}'
                default:
                        name := l.new(nodeName)
                        name.pos = l.pos - 1 // include the single char
                        st := l.top() // nodeCall
                        st.node.children = append(st.node.children, name)
                        l.endCall(st)
                }
        }
}

func (l *lex) stateCallName() {
        st := l.top() // Must be a nodeName.
        delm := st.delm
state_loop:
        for l.get() {
                switch {
                case l.rune == '$':
                        l.push(nodeCall, l.stateDollar, 0).node.pos-- // 'pos--' for the '$'
                        break state_loop
                case l.rune == ':' && st.code == 0:
                        prefix := l.new(nodeNamePrefix)
                        prefix.pos = l.pos - 1
                        st.node.children = append(st.node.children, prefix)
                        st.code++
                case l.rune == '.':
                        part := l.new(nodeNamePart)
                        part.pos = l.pos - 1
                        st.node.children = append(st.node.children, part)
                case l.rune == '\\':
                        l.escapeTextLine(st.node)
                case l.rune == ' ': fallthrough
                case l.rune == delm:
                        name := st.node
                        name.end = l.pos - 1
                        l.pop()

                        st = l.top()
                        switch s := name.str(); s {
                        case "speak":
                                st.node.kind = nodeSpeak
                                if l.rune != delm {
                                        l.push(nodeArg, l.stateSpeakDialect, 0).delm = delm
                                } else {
                                        lineno, colno := l.getLineColumn()
                                        errorf("%v:%v:%v: unexpected delimiter\n", l.scope, lineno, colno)
                                }
                        default:
                                st.node.children = append(st.node.children, name)
                                switch l.rune {
                                case delm:
                                        l.endCall(st)
                                case ' ':
                                        l.push(nodeArg, l.stateCallArg, 0).delm = delm
                                }
                        }
                        break state_loop
                }
        }
}

func (l *lex) stateCallArg() {
        st := l.top() // Must be a nodeArg.
        delm := st.delm
state_loop:
        for l.get() {
                switch {
                case l.rune == '$':
                        l.push(nodeCall, l.stateDollar, 0).node.pos-- // 'pos--' for the '$'
                        break state_loop
                case l.rune == '\\':
                        l.escapeTextLine(st.node)
                case l.rune == ',': fallthrough
                case l.rune == delm:
                        arg := st.node
                        arg.end = l.pos - 1
                        l.pop()

                        st = l.top()
                        st.node.children = append(st.node.children, arg)
                        if l.rune == delm {
                                l.endCall(st)
                        } else {
                                l.push(nodeArg, l.stateCallArg, 0).delm = delm
                        }
                        break state_loop
                }
        }
}

func (l *lex) stateSpeakDialect() {
        st := l.top() // Must be a nodeArg.
        delm := st.delm
state_loop:
        for l.get() {
                switch {
                case l.rune == '$':
                        l.push(nodeCall, l.stateDollar, 0).node.pos-- // 'pos--' for the '$'
                        break state_loop
                case l.rune == '\\':
                        l.escapeTextLine(st.node)
                case l.rune == ',': fallthrough
                case l.rune == delm:
                        arg := st.node
                        arg.end = l.pos - 1
                        l.pop()

                        st = l.top()
                        st.node.children = append(st.node.children, arg)
                        if l.rune == delm {
                                l.endCall(st)
                        } else {
                                l.push(nodeArg, l.stateSpeakScript, 0).delm = delm
                        }
                        break state_loop
                }
        }
}

func (l *lex) stateSpeakScript() {
        st := l.top() // Must be a nodeArg.
        delm := st.delm
state_loop:
        for l.get() {
                switch {
                case l.rune == '$':
                        l.push(nodeCall, l.stateDollar, 0).node.pos-- // 'pos--' for the '$'
                        break state_loop
                case l.rune == '\\':
                        if st.code == 0 {
                                if l.get() {
                                        if l.rune != '\n' { // skip \\\n
                                                lineno, colno := l.getLineColumn()
                                                errorf("%v:%v:%v: bad escape \\%v in this context\n", l.scope, lineno, colno, string(l.rune))
                                        }
                                }
                        } else {
                                l.escapeTextLine(st.node)
                        }
                case l.rune == '-' && st.code == 0: /* skip */
                case l.rune != '-' && st.code == 0:
                        st.code, st.node.pos = 1, l.pos
                case l.rune == '\n' && st.code == 1 && l.peek() == '-':
                delimiter_loop:
                        for i, r, n := l.pos, rune(0), 1; i < len(l.s); i += n {
                                switch r, n = utf8.DecodeRune(l.s[i:]); r {
                                default: break delimiter_loop
                                case '-': /* skip */
                                case delm:
                                        script := st.node
                                        script.end = l.backwardNonSpace(script.pos, l.pos)

                                        l.rune, l.pos, st = delm, i+1, l.pop()

                                        st = l.top() // the $(speak) node
                                        st.node.children = append(st.node.children, script)
                                        l.endCall(st)
                                        break state_loop
                                }
                        }
                case l.rune == ',': fallthrough
                case l.rune == delm:
                        script := st.node
                        script.end = l.pos - 1

                        l.pop()

                        st = l.top() // the $(speak) node
                        st.node.children = append(st.node.children, script)
                        if l.rune == delm {
                                l.endCall(st)
                        } else {
                                l.push(nodeArg, l.stateSpeakScript, 0).delm = delm
                        }
                        break state_loop
                }
        }
}

func (l *lex) endCall(st *lexStack) {
        call := st.node
        call.end = l.pos

        l.pop() // pop out the current nodeCall

        // Append the call to it's parent.
        t := l.top().node
        t.children = append(t.children, call)
}

/*
stateDollar:
        st.node.children = []*node{ l.new(nodeName) }
        l.step = l.stateCallee 
*/
func (l *lex) stateCallee() {
        const ( init int = iota; name; args )
        st := l.top() // Must be a nodeCall.

state_loop:
        for l.get() {
                switch {
                case l.rune == '(' && st.code == init: st.delm = ')'; fallthrough
                case l.rune == '{' && st.code == init: if st.delm == 0 { st.delm = '}' }
                        st.node.children[0].pos = l.pos
                        st.node.end, st.code = l.pos, name

                case l.rune == ' ' && st.code == name:
                        st.code = args; fallthrough
                case l.rune == ',' && st.code == args:
                        a := l.new(nodeArg)
                        i := len(st.node.children)-1
                        st.node.children[i].end = l.pos-1
                        st.node.children = append(st.node.children, a)

                case l.rune == '\\':
                        i := len(st.node.children)-1
                        if 0 <= i {
                                l.escapeTextLine(st.node.children[i])
                        }

                case l.rune == '$' && st.code != init:
                        st = l.push(nodeCall, l.stateDollar, 0)
                        st.node.pos-- // for the '$'
                        break state_loop

                case l.rune == st.delm: //&& st.code != init:
                        fallthrough
                case st.code == init: // $$, $a, $<, $@, $^, etc.
                        call, i, n := st.node, len(st.node.children)-1, 1
                        if st.delm == rune(0) { n = 0 } // don't shift for single char like '$a'
                        call.end, call.children[i].end = l.pos, l.pos-n

                        l.pop() // pop out the current nodeCall

                        t := l.top().node
                        switch t.kind {
                        case nodeDeferredText: fallthrough
                        case nodeImmediateText:
                                t.children = append(t.children, call)
                        default:
                                // Add to the last child.
                                i = len(t.children)-1
                                if 0 <= i { t = t.children[i] }
                                t.children = append(t.children, call)
                        }

                        break state_loop
                }
        }
}

func (l *lex) parse() bool {
        l.step, l.pos = l.stateGlobal, 0
        end := len(l.s); for l.pos < end { l.step() }

        var t *lexStack
        for 0 < len(l.stack) {
                t = l.top() // the current top state
                l.step() // Make extra step to give it a chance handling rune(0),
                if t == l.top() {
                        l.pop() // pop out the state if the top is still there
                }

        }
        return l.pos == end
}

type pendedBuild struct {
        m *Module
        p *Context
        args Items
}

// Context hold a parse context and the current module being processed.
type Context struct {
        lexingStack []*lex
        moduleStack []*Module

        l *lex // the current lexer
        m *Module // the current module being processed
        t *template // the current template being processed
        tild *template // the current template referred by '~'

        g *namespaceEmbed // the global namespace

        templates map[string]*template

        modules map[string]*Module
        moduleOrderList []*Module
        moduleBuildList []pendedBuild

        w *worker.Worker
}

func (ctx *Context) GetModules() map[string]*Module { return ctx.modules }
func (ctx *Context) GetModuleOrderList() []*Module { return ctx.moduleOrderList }
func (ctx *Context) GetModuleBuildList() []pendedBuild { return ctx.moduleBuildList }
func (ctx *Context) ResetModules() {
        ctx.modules = make(map[string]*Module, 8)
        ctx.moduleOrderList = []*Module{}
        ctx.moduleBuildList = []pendedBuild{}
}

func (ctx *Context) CurrentScope() string {
        return ctx.l.scope
}

func (ctx *Context) CurrentLocation() location {
        return ctx.l.location()
}

func (ctx *Context) CurrentModule() *Module {
        return ctx.m
}

func (ctx *Context) NewDeferWith(m *Module) func() {
        prev := ctx.m; ctx.m = m
        return func() { ctx.m = prev }
}

func (ctx *Context) With(m *Module, work func()) {
        revert := ctx.NewDeferWith(m); defer revert()
        work()
}

func (ctx *Context) Call(name string, args ...Item) (is Items) {
        return ctx.call(ctx.l.location(), name, args...)
}

func (ctx *Context) Set(name string, items ...Item) {
        hasPrefix, prefix, parts := ctx.expandNameString(name)
        ctx.setWithDetails(hasPrefix, prefix, parts, items...)
}

func (ctx *Context) setWithDetails(hasPrefix bool, prefix string, parts []string, items ...Item) {
        if ns := ctx.getNamespaceWithDetails(hasPrefix, prefix, parts); ns != nil {
                if m, n := ns.getDefineMap(), len(parts); m != nil && 0 < n {
                        var (
                                d *define
                                found bool
                                sym = parts[n-1]
                                loc = ctx.l.location()
                        )
                        if d, found = m[sym]; !found {
                                d = new(define)
                                m[sym] = d
                        }
                        if d.readonly {
                                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                                fmt.Printf("%v:%v:%v:warning: readonly '%s'\n", ctx.l.scope, lineno, colno, strings.Join(parts, "."))
                        } else {
                                d.name, d.value, d.loc = sym, items, loc
                        }
                }
        } else {
                var loc = ctx.l.location()
                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                if hasPrefix {
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: missing '%s:%s'\n", ctx.l.scope, lineno, colno, prefix, strings.Join(parts, "."))
                } else {
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: missing '%s'\n", ctx.l.scope, lineno, colno, strings.Join(parts, "."))
                }
                //errorf("unknown namespace '%s'", strings.Join(parts, "."))
        }
}

func (ctx *Context) call(loc location, name string, args ...Item) (is Items) {
        hasPrefix, prefix, parts := ctx.expandNameString(name)
        is = ctx.callWithDetails(loc, hasPrefix, prefix, parts, args...)
        return
}

func (ctx *Context) callWithDetails(loc location, hasPrefix bool, prefix string, parts []string, args ...Item) (is Items) {
        n := len(parts)

        call := func(ns namespace) (called bool) {
                if ns == nil { return }
                if m := ns.getDefineMap(); m != nil && 0 < n {
                        sym := parts[n-1]
                        if hasPrefix {
                                if ht, ok := hooksMap[prefix]; ok && ht != nil {
                                        if h, ok := ht[sym]; ok && h != nil {
                                                is, called = h(ctx, args), true
                                        }
                                }
                        }
                        if !called {
                                if d, ok := m[sym]; ok && d != nil {
                                        is, called = d.value, true
                                }
                        }
                }
                return
        }
        
        // Process special symbols and builtins first.
        if hasPrefix {
                if prefix == "~" && ctx.tild != nil {
                        if prefix = ctx.tild.name; call(ctx.getNamespaceWithDetails(hasPrefix, prefix, parts)) {
                                return
                        }
                }
        } else if n == 1 {
                switch sym := parts[0]; sym {
                case "$":  is = append(is, stringitem("$"))
                case "me": // rename: $(me) -> $(me.name)
                        parts, n = append(parts, "name"), 2
                default:
                        if f, ok := builtins[sym]; ok && f != nil {
                                is = f(ctx, loc, args)
                                return
                        }
                }
        }

        if ns := ctx.getNamespaceWithDetails(hasPrefix, prefix, parts); ns != nil {
                _ = call(ns)
        } else {
                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                if hasPrefix {
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: no namespace for '%s:%s'\n", ctx.l.scope, lineno, colno, prefix, strings.Join(parts, "."))
                } else {
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: no namespace for '%s'\n", ctx.l.scope, lineno, colno, strings.Join(parts, "."))
                }
        }
        return
}

// getDefine returns a define for hierarchy names like `tool:m1.m2.var`, `m1.m2.var`, etc.
func (ctx *Context) getDefine(name string) (d *define, hasPrefix bool, prefix string, parts []string) {
        hasPrefix, prefix, parts = ctx.expandNameString(name)
        d = ctx.getDefineWithDetails(hasPrefix, prefix, parts)
        return
}
func (ctx *Context) getDefineWithDetails(hasPrefix bool, prefix string, parts []string) (d *define) {
        if ns := ctx.getNamespaceWithDetails(hasPrefix, prefix, parts); ns != nil {
                if m, n := ns.getDefineMap(), len(parts); m != nil && 0 < n {
                        d, _ = m[parts[n-1]]
                }
        }
        return
}

// getNamespaceAndDetails returns a namespace for hierarchy names like `tool:m1.m2.var`, `m1.m2.var`, etc.
func (ctx *Context) getNamespaceAndDetails(name string) (ns namespace, hasPrefix bool, prefix string, parts []string) {
        hasPrefix, prefix, parts = ctx.expandNameString(name)
        ns = ctx.getNamespaceWithDetails(hasPrefix, prefix, parts)
        return
}

func (ctx *Context) getNamespaceWithDetails(hasPrefix bool, prefix string, parts []string) (ns namespace) {
        num := len(parts)

        if hasPrefix {
                if prefix == "~" && ctx.tild != nil {
                        ns, prefix = ctx.tild.namespaceEmbed, ctx.tild.name
                }
                if ns != nil {
                        // ~ // fmt.Printf("tild: %v\n", ns)
                } else if t, ok := ctx.templates[prefix]; ok && t != nil {
                        ns = t.namespaceEmbed
                } else {
                        lineno, colno := ctx.l.caculateLocationLineColumn(ctx.l.location())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: undefined toolset prefix `%s'\n",
                                ctx.l.scope, lineno, colno, prefix)
                        return
                }
        }

        if num == 1 && ns == nil {
                ns = ctx.g
                return
        }

        lineno, colno := ctx.l.caculateLocationLineColumn(ctx.l.location())
        for i, s := range parts[0:num-1] {
                if ns != nil {
                        ns = ns.getNamespace(s)
                } else if i == 0 {
                        switch s {
                        default:
                                if m, ok := ctx.modules[s]; !ok || m == nil {
                                        fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: '%s' is nil\n", ctx.l.scope, lineno, colno, s)
                                        //break loop_parts
                                } else {
                                        ns = m
                                }
                        case "me":
                                if ctx.m == nil {
                                        fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: 'me' is nil\n", ctx.l.scope, lineno, colno)
                                        //break loop_parts
                                } else {
                                        ns = ctx.m
                                }
                        case "~":
                                if ctx.tild == nil {
                                        if ctx.m != nil && 0 < len(ctx.m.Templates) {
                                                ns = ctx.m.Templates[0].namespaceEmbed
                                        } else {
                                                fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: tild is referred to nil template\n", ctx.l.scope, lineno, colno)
                                                //break loop_parts
                                        }
                                } else {
                                        ns = ctx.tild.namespaceEmbed
                                }
                        }
                }
                if ns == nil {
                        fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: `%s' is undefined scope\n", ctx.l.scope, lineno, colno,
                                strings.Join(parts[0:i+1], "."))
                        break
                }
        }
        return
}

func (ctx *Context) multipart(n *node) (*bytes.Buffer, []int) {
        b, l, pos, parts := new(bytes.Buffer), n.l, n.pos, []int{ -1 }
        for _, c := range n.children {
                if pos < c.pos { b.Write(l.s[pos:c.pos]) }; pos = c.end
                switch c.kind {
                case nodeNamePrefix: b.WriteString(":"); parts[0] = b.Len()
                case nodeNamePart:   b.WriteString("."); parts = append(parts, b.Len())
                default:   b.WriteString(ctx.nodeItems(c).Expand(ctx))
                }
        }
        if pos < n.end {
                b.Write(l.s[pos:n.end])
        }
        return b, parts
}

func (ctx *Context) expandNameString(name string) (hasPrefix bool, prefix string, parts []string) {
        if i := strings.Index(name, ":"); 0 <= i {
                prefix, hasPrefix = name[0:i], true
                name = name[i+1:]
        }
        parts = strings.Split(name, ".")
        return
}

func (ctx *Context) expandNameNode(n *node) (scoped bool, name string, parts []string) {
        pos := 0
        b, i := ctx.multipart(n)
        if 0 <= i[0] {
                pos, scoped = i[0], true
                name = string(b.Bytes()[0:pos-1])
        }

        for _, n := range i[1:] {
                parts = append(parts, string(b.Bytes()[pos:n-1]))
                pos = n
        }
        parts = append(parts, string(b.Bytes()[pos:]))
        return
}

func (ctx *Context) speak(name string, scripts ...*node) (is Items) {
        if dialect, ok := dialects[name]; ok {
                for _, sn := range scripts {
                        is = append(is, dialect(ctx, sn)...)
                }
        } else if c, e := exec.LookPath(name); e == nil {
                var args []string
                for _, s := range scripts {
                        args = append(args, s.Expand(ctx))
                }
                out := new(bytes.Buffer)
                cmd := exec.Command(c, args...)
                cmd.Stdout = out
                if err := cmd.Run(); err != nil {
                        errorf("%v: %v", name, err)
                } else {
                        is = append(is, stringitem(out.String()))
                }
        } else {
                errorf("unknown dialect %v", name)
        }
        return
}

func (ctx *Context) nodeItems(n *node) (is Items) {
        switch n.kind {
        case nodeEscape:
                //fmt.Printf("%v, %v\n%s\n", n.pos, len(ctx.l.s), string(ctx.l.s))
                //switch ctx.l.s[n.pos + 1] {
                switch n.l.s[n.pos + 1] {
                case '\n': is = Items{ stringitem(" ") }
                case '#':  is = Items{ stringitem("#") }
                }

        case nodeCall:
                var args Items
                for _, an := range n.children[1:] {
                        args = args.Concat(ctx, ctx.nodeItems(an)...)
                }
                scoped, name, parts := ctx.expandNameNode(n.children[0])
                is = ctx.callWithDetails(n.loc(), scoped, name, parts, args...)

        case nodeSpeak:
                dialect := n.children[0].Expand(ctx)
                if 1 < len(n.children) {
                        is = ctx.speak(dialect, n.children[1:]...)
                } else {
                        is = ctx.speak(dialect)
                }

        case nodeName:          fallthrough
        case nodeArg:           fallthrough
        case nodeRecipe:        fallthrough
        case nodeDeferredText:  fallthrough
        case nodeTargets:       fallthrough
        case nodePrerequisites: fallthrough
        case nodeImmediateText:
                var (
                        s string
                        nc = len(n.children)
                )
                if 0 < nc {
                        b, _ := ctx.multipart(n)
                        s = b.String()
                } else {
                        if n.end < n.pos {
                                //fmt.Printf("%v: %v, %v: %v\n", n.kind, n.pos, n.end, string(n.l.s[n.pos:n.pos+9]))
                        }
                        s = n.str()
                }
                is = Items{ stringitem(s) }

        case nodeComment:
                // Convert inline comment into "\n". This happens when a comment
                // recipt occured.
                is = Items{ stringitem("\n") }

        default:
                panic(fmt.Sprintf("fixme: %v: %v (%v children)\n", n.kind, n.str(), len(n.children)))
        }
        return
}

func (ctx *Context) nodesItems(nodes... *node) (is Items) {
        for _, n := range nodes {
                is = is.Concat(ctx, ctx.nodeItems(n)...)
        }
        return
}
                
func (ctx *Context) ItemsStrings(a ...Item) (s []string) {
        for _, i := range a {
                s = append(s, i.Expand(ctx))
        }
        return
}

func (ctx *Context) processNode(n *node) (err error) {
        if ctx.t != nil {
                switch n.kind {
                case nodeCommit:
                        processTemplateCommit(ctx, n)
                case nodePost:
                        processTemplatePost(ctx, n)
                default:
                        if ctx.t.post != nil {
                                ctx.t.postNodes = append(ctx.t.postNodes, n)
                        } else {
                                ctx.t.declNodes = append(ctx.t.declNodes, n)
                        }
                }
        } else {
                if f, ok := processors[n.kind]; ok && f != nil {
                        err = f(ctx, n)
                } else {
                        panic(fmt.Sprintf("'%v' not implemented", n.kind))
                }
        }
        return
}

func (ctx *Context) parseBuffer() (err error) {
        if !ctx.l.parse() {
                err = errors.New("syntax error")
                return
        }

        for _, n := range ctx.l.nodes {
                if n.kind == nodeComment { continue }
                if e := ctx.processNode(n); e != nil {
                        break
                }
        }

        return
}

func (ctx *Context) append(scope string, s []byte) (err error) {
        ctx.lexingStack = append(ctx.lexingStack, ctx.l)
        defer func() {
                ctx.lexingStack = ctx.lexingStack[0:len(ctx.lexingStack)-1]

                if e := recover(); e != nil {
                        if se, ok := e.(*smarterror); ok {
                                lineno, colno := ctx.l.getLineColumn()
                                fmt.Printf("%v:%v:%v: %v\n", scope, lineno, colno, se)
                        } else {
                                panic(e)
                        }
                }
        }()

        ctx.l = &lex{ parseBuffer:&parseBuffer{ scope:scope, s: s }, pos: 0 }
        ctx.m = nil

        if err = ctx.parseBuffer(); err != nil {
                // ...
        }
        return
}

func (ctx *Context) include(fn string) (err error) {
        var (
                f *os.File
                s []byte
        )

        if f, err = os.Open(fn); err != nil {
                return
        }

        defer f.Close()

        if s, err = ioutil.ReadAll(f); err == nil {
                err = ctx.append(fn, s)
        }
        return
}

func processNodeComment(ctx *Context, n *node) (err error) {
        return
}

func processNodeCall(ctx *Context, n *node) (err error) {
        if s := strings.TrimSpace(ctx.nodeItems(n).Expand(ctx)); s != "" {
                lineno, colno := ctx.l.caculateLocationLineColumn(n.loc())
                fmt.Fprintf(os.Stderr, "%v:%v:%v: illigal: '%v'\n", ctx.l.scope, lineno, colno, s)
        }
        return
}

func processNodeImmediateText(ctx *Context, n *node) (err error) {
        if s := strings.TrimSpace(ctx.nodeItems(n).Expand(ctx)); s != "" {
                lineno, colno := ctx.l.caculateLocationLineColumn(n.loc())
                fmt.Fprintf(os.Stderr, "%v:%v:%v: syntax error: '%v'\n", ctx.l.scope, lineno, colno, s)
        }
        return
}

func processNodeDefineQuestioned(ctx *Context, n *node) (err error) {
        scoped, name, parts := ctx.expandNameNode(n.children[0])
        if is := ctx.callWithDetails(n.loc(), scoped, name, parts); is.IsEmpty(ctx) {
                ctx.setWithDetails(scoped, name, parts, n)
        }
        return
}

func processNodeDefineDeferred(ctx *Context, n *node) (err error) {
        scoped, name, parts := ctx.expandNameNode(n.children[0])
        ctx.setWithDetails(scoped, name, parts, n)
        return
}

func processNodeDefineSingleColoned(ctx *Context, n *node) (err error) {
        scoped, name, parts := ctx.expandNameNode(n.children[0])
        ctx.setWithDetails(scoped, name, parts, ctx.nodeItems(n.children[1])...)
        return
}

func processNodeDefineDoubleColoned(ctx *Context, n *node) (err error) {
        return processNodeDefineSingleColoned(ctx, n)
}

func processNodeDefineAppend(ctx *Context, n *node) (err error) {
        scoped, name, parts := ctx.expandNameNode(n.children[0])
        if d := ctx.getDefineWithDetails(scoped, name, parts); d != nil {
                d.value = append(d.value, n.children[1])
        } else {
                value := ctx.nodeItems(n.children[1])
                ctx.setWithDetails(scoped, name, parts, value...)
        }
        return
}

func processNodeDefineNot(ctx *Context, n *node) (err error) {
        panic("'!=' not implemented")
}

func processNodeRule(ctx *Context, n *node) (err error) {
        var ns namespace
        if ctx.m == nil {
                ns = ctx.g
        } else {
                ns = ctx.m
        }

        r := ns.link(Split(ctx.nodeItems(n.children[0]).Expand(ctx))...)
        r.prerequisites, r.node = Split(ctx.nodeItems(n.children[1]).Expand(ctx)), n
        if 2 < len(n.children) {
                for _, c := range n.children[2].children {
                        r.recipes = append(r.recipes, c)
                }
        }

        // Set goal if empty.
        if 0 < len(r.targets) {
                if g := r.ns.getGoal(); g == "" {
                        r.ns.setGoal(r.targets[0])
                }
        }

        switch n.kind {
        case nodeRulePhony:             r.c = &phonyTargetUpdater{}
        case nodeRuleChecker:           r.c = &checkRuleUpdater{ r }
        case nodeRuleDoubleColoned:     r.c = &defaultTargetUpdater{}
        case nodeRuleSingleColoned:     r.c = &defaultTargetUpdater{}
        default: errorf("unexpected rule type: %v", n.kind)
        }

        /*
        lineno, colno := ctx.l.caculateLocationLineColumn(n.loc())
        fmt.Fprintf(os.Stderr, "%v:%v:%v: %v\n", ctx.l.scope, lineno, colno, n.kind) //*/
        return
}

func processNodeInclude(ctx *Context, n *node) (err error) {
        fmt.Printf("todo: %v %v\n", n.kind, n.children)
        return
}

func processNodeTemplate(ctx *Context, n *node) (err error) {
        var (
                args, loc = ctx.nodesItems(n.children...), n.loc()
        )
        if ctx.t != nil {
                errorf("template already defined (%v)", args)
        } else {
                if ctx.m != nil {
                        s, lineno, colno := ctx.m.GetDeclareLocation()
                        fmt.Printf("%v:%v:%v:warning: declare template in module\n", s, lineno, colno)

                        lineno, colno = ctx.l.caculateLocationLineColumn(loc)
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: ", ctx.l.scope, lineno, colno)

                        errorf("declare template inside module")
                        return
                }

                name := strings.TrimSpace(args[0].Expand(ctx))
                if name == "" {
                        lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: empty template name", ctx.l.scope, lineno, colno)
                        errorf("empty template name")
                        return
                }

                if t, ok := ctx.templates[name]; ok && t != nil {
                        //lineno, colno := ctx.l.caculateLocationLineColumn(t.loc)
                        //fmt.Fprintf(os.Stderr, "%v:%v:%v: %s already declared", ctx.l.scope, lineno, colno, ctx.t.name)
                        errorf("template '%s' already declared", name)
                        return
                }

                ctx.t = &template{
                        name:name,
                        namespaceEmbed: &namespaceEmbed{
                                defines: make(map[string]*define, 8),
                                rules: make(map[string]*rule, 4),
                                patts: make(map[string]*rule, 2),
                        },
                }

                ctx.t.set(ctx, []string{ "name" }, StringItem(name))
                
                if 1 < len(args) {
                        for _, a := range args[1:] {
                                name := strings.TrimSpace(a.Expand(ctx))
                                if t, ok := ctx.templates[name]; !ok || t == nil {
                                        errorf("template '%s' is not declared", name)
                                        return
                                } else {
                                        ctx.t.bases = append(ctx.t.bases, t)
                                }
                        }
                }
        }
        return
}

func processNodeModule(ctx *Context, n *node) (err error) {
        var (
                name, exportName string
                args, loc = ctx.nodesItems(n.children...), n.loc()
        )
        if 0 < len(args) { name = strings.TrimSpace(args[0].Expand(ctx)) }
        if name == "" {
                errorf("module name is required")
                return
        }
        if name == "me" {
                errorf("module name 'me' is reserved")
                return
        }

        exportName = "export"

        var (
                m *Module
                has bool
        )
        if m, has = ctx.modules[name]; !has && m == nil {
                m = &Module{
                        l: nil,
                        Children: make(map[string]*Module, 2),
                        namespaceEmbed: &namespaceEmbed{
                                defines: make(map[string]*define, 8),
                                rules: make(map[string]*rule, 4),
                                patts: make(map[string]*rule, 2),
                        },
                }
                ctx.modules[name] = m
                ctx.moduleOrderList = append(ctx.moduleOrderList, m)
        } else if m.l != nil {
                s := ctx.l.scope
                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                fmt.Printf("%v:%v:%v: '%v' already declared\n", s, lineno, colno, name)

                s, lineno, colno = m.GetDeclareLocation()
                fmt.Printf("%v:%v:%v:warning: previous '%v'\n", s, lineno, colno, name)

                errorf("module already declared")
        }

        // Reset the current module pointer.
        upper := ctx.m
        if upper != nil {
                upper.Children[name] = m
        }

        ctx.m = m

        // Reset the lex and location (because it could be created by $(use))
        if m.l == nil {
                m.l, m.declareLoc = ctx.l, loc

                if 1 < len(args) {
                        for _, nameItem := range args[1:] {
                                name := strings.TrimSpace(nameItem.Expand(ctx))
                                if t, _ := ctx.templates[name]; t != nil {
                                        m.Templates = append(m.Templates, t)
                                } else {
                                        s := ctx.l.scope
                                        //s, lineno, colno = nameItem.GetDeclareLocation()
                                        lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                                        fmt.Printf("%v:%v:%v: '%v' not found\n", s, lineno, colno, name)
                                        errorf("no such template")
                                }
                        }
                }
                if upper != nil {
                        ctx.moduleStack = append(ctx.moduleStack, upper)
                }
        }

        if x, ok := m.Children[exportName]; !ok {
                x = &Module{
                        l: m.l,
                        Parent: m,
                        Children: make(map[string]*Module),
                        namespaceEmbed: &namespaceEmbed{
                                defines: make(map[string]*define, 4),
                                rules: make(map[string]*rule, 4),
                                patts: make(map[string]*rule, 2),
                        },
                }
                m.Children[exportName] = x
        }

        wd, err := os.Getwd()
        if err != nil { 
                errorf("getting working directory: %v", err) 
        }

        if fi, e := os.Stat(ctx.l.scope); e == nil && fi != nil && !fi.IsDir() {
                dir := filepath.Dir(ctx.l.scope)
                wd = filepath.Join(wd, dir)
                ctx.Set("me.workdir", stringitem(wd))
                ctx.Set("me.dir", stringitem(dir))
        } else {
                ctx.Set("me.workdir", stringitem(wd))
                ctx.Set("me.dir", stringitem(workdir))
        }
        ctx.Set("me.name", stringitem(name))
        ctx.Set("me.export.name", stringitem(exportName))

        for i, t := range m.Templates {
                if i == 0 {
                        // Module always refer "~" to the first tempalte
                        ctx.tild = t
                }
                
                // Use templateNodesProcessor to eliminate initialization loop
                // detection (see 'processors').
                templateNodesProcessor(t).processDeclNodes(ctx)
        }
        return
}

func processTemplateCommit(ctx *Context, n *node) (err error) {
        if ctx.m != nil {
                errorf("declared template inside module")
                return
        }
        if t, ok := ctx.templates[ctx.t.name]; ok && t != nil {
                errorf("template '%s' already declared", ctx.t.name)
                return
        }
        ctx.templates[ctx.t.name] = ctx.t
        ctx.t = nil // must unset the 't'
        return
}

func processTemplatePost(ctx *Context, n *node) (err error) {
        //fmt.Printf("processTemplatePost: %v\n", n.children)
        if ctx.t != nil {
                ctx.t.post = n
        } else {
                errorf("template is nil")
        }
        return
}

func processNodeCommit(ctx *Context, n *node) (err error) {
        if ctx.m == nil {
                panic("nil module")
        }
        
        var (
                args, loc = ctx.nodesItems(n.children...), n.loc()
        )
        
        if *flagVV {
                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                verbose("commit (%v:%v:%v)", ctx.l.scope, lineno, colno)
        }

        for _, t := range ctx.m.Templates {
                // Use templateNodesProcessor to eliminate initialization loop
                // detection (see 'processors').
                templateNodesProcessor(t).processPostNodes(ctx)
        }

        ctx.m.commitLoc = loc
        ctx.moduleBuildList = append(ctx.moduleBuildList, pendedBuild{ctx.m, ctx, args})
        if i := len(ctx.moduleStack)-1; 0 <= i {
                up := ctx.moduleStack[i]
                ctx.m.Parent = up
                ctx.moduleStack, ctx.m = ctx.moduleStack[0:i], up
        } else {
                ctx.m = nil // must unset the 'm'
                ctx.tild = nil // unset the '~' reference
        }
        return
}

func processNodeUse(ctx *Context, n *node) (err error) {       
        if ctx.m == nil { errorf("no module defined") }

        var (
                args = ctx.nodesItems(n.children...)
                name = "using"
        )

        if d, ok := ctx.m.defines[name]; ok && d != nil {
                d.value = append(d.value, args...)
        } else {
                ctx.m.defines[name] = &define{
                        loc:ctx.CurrentLocation(),
                        name:name, value:args,
                }
        }
        return
}

func NewContext(scope string, s []byte, vars map[string]string) (ctx *Context, err error) {
        ctx = &Context{
                templates: make(map[string]*template, 8),
                modules: make(map[string]*Module, 8),
                l: &lex{ parseBuffer:&parseBuffer{ scope:scope, s: s }, pos: 0 },
                g: &namespaceEmbed{
                        defines: make(map[string]*define, len(vars) + 16),
                        rules: make(map[string]*rule, 8),
                        patts: make(map[string]*rule, 2),
                },
                w: worker.New(),
        }

        for k, v := range vars {
                ctx.Set(k, stringitem(v))
        }

        err = ctx.parseBuffer()
        return
}

func NewContextFromFile(fn string, vars map[string]string) (ctx *Context, err error) {
        s := []byte{} // TODO: needs init script
        if ctx, err = NewContext(fn, s, vars); err == nil && ctx != nil {
                err = ctx.include(fn)
        }
        return
}
