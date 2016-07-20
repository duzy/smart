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

var (
        ErrReadOnly = errors.New("modifying readonly definition")
)

type Name struct {
        Prefixed bool
        Prefix string
        Ns []string
        Sym string
}

func (name *Name) String() (s string) {
        s = fmt.Sprintf("%s.%s", strings.Join(name.Ns, "."), name.Sym)
        if !name.Prefixed {
                s = fmt.Sprintf("%s:%s", name.Prefix, s)
        }
        return
}

func (name *Name) HasPrefix(s string) bool {
        return name.Prefixed && name.Prefix == s
}

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
        recipes []interface{} // node, string
        ns namespace
        c checkupdater
        node node
        kind rulekind
}

type checkupdater interface {
        check(ctx *Context, r *rule, m *match) bool
        update(ctx *Context, r *rule, m *match) bool
}

type namespace interface {
        findMatchRules(ctx *Context, target string) (mrs []*matchrule)
        isPhonyTarget(ctx *Context, target string) bool
        saveDefines(names ...string) (saveIndex int, m map[string]*define)
        restoreDefines(saveIndex int)
        get(ctx *Context, name string) *define
        set(ctx *Context, name string, items ...Item) error
        append(ctx *Context, name string, items ...Item) error
        getCheckRules(target string) (rules []*rule)
        getGoal() (target string)
        setGoal(target string)
        link(targets ...string) (r *rule)
}

type namespaceBase struct {
        defines map[string]*define
        saveList []map[string]*define // saveDefines, restoreDefines
        rules map[string]*rule
        patts map[string]*rule
        pattList []*rule
        goal string
}
func (ns *namespaceBase) getGoal() string { return ns.goal }
func (ns *namespaceBase) setGoal(target string) { ns.goal = target }
func (ns *namespaceBase) getCheckRules(target string) (rules []*rule) {
        for ru, ok := ns.rules[target]; ok && ru != nil; {
                if _, ok := ru.node.(*nodeRuleChecker); ok {
                        rules = append(rules, ru)
                }
                ru, ok = ru.prev[target]
        }
        return
}
func (ns *namespaceBase) link(targets ...string) (r *rule) {
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

func (ns *namespaceBase) saveDefines(names ...string) (saveIndex int, m map[string]*define) {
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
func (ns *namespaceBase) restoreDefines(saveIndex int) {
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

func (ns *namespaceBase) get(ctx *Context, name string) *define {
        if d, ok := ns.defines[name]; ok && d != nil {
                return d
        }
        return nil
}

func (ns *namespaceBase) set(ctx *Context, name string, items ...Item) error {
        if d, ok := ns.defines[name]; ok && d != nil {
                if d.readonly {
                        return ErrReadOnly
                } else {
                        d.value = items
                }
        } else {
                ns.defines[name] = &define{ loc:ctx.CurrentLocation(), name:name, value:items }
        }
        return nil
}

func (ns *namespaceBase) append(ctx *Context, name string, items ...Item) error {
        if d, ok := ns.defines[name]; ok && d != nil {
                if d.readonly {
                        return ErrReadOnly
                } else {
                        d.value = append(d.value, items...)
                }
        } else {
                ns.defines[name] = &define{ loc:ctx.CurrentLocation(), name:name, value:items }
        }
        return nil
}

func (ns *namespaceBase) findMatchRules(ctx *Context, target string) (matchrules []*matchrule) {
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

func (ns *namespaceBase) isPhonyTarget(ctx *Context, target string) bool {
        if rr, ok := ns.rules[target]; ok && rr != nil {
                _, ok = rr.node.(*nodeRulePhony)
                return ok
        }
        return false
}

var (
        statements = map[string] func(*baseNodeStruct) node {
                "include":  func(cb *baseNodeStruct) node { return &nodeInclude {nodeBase{*cb}} },
                "template": func(cb *baseNodeStruct) node { return &nodeTemplate{nodeBase{*cb}} },
                "module":   func(cb *baseNodeStruct) node { return &nodeModule  {nodeBase{*cb}} },
                "commit":   func(cb *baseNodeStruct) node { return &nodeCommit  {nodeBase{*cb}} },
                "post":     func(cb *baseNodeStruct) node { return &nodePost    {nodeBase{*cb}} },
                "use":      func(cb *baseNodeStruct) node { return &nodeUse     {nodeBase{*cb}} },
                // TODO: template...endtempl
                // TODO: module...endmod
        }
)

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

// flatitem is an expanded string with a location information
type flatitem struct {
        s string
        l location
}

func (fi *flatitem) Expand(ctx *Context) string { return fi.s }
func (fi *flatitem) IsEmpty(ctx *Context) bool { return fi.s == "" }

func nodes2Items(a ...node) (is Items) {
        for _, n := range a {
                is = append(is, n)
        }
        return
}

type node interface {
        Item
        bs() *baseNodeStruct
        tc() nodeTypeCode
        kind() string
        str() string
        getPosBeg() int
        getPosEnd() int
        setPosBeg(n int)
        setPosEnd(n int)
        addPosBeg(n int)
        addPosEnd(n int)
        //items() Items
        children() []node
        child(n int) node
        addChild(c node)
        setChildren(ca ...node)
        reset(bs baseNodeStruct)
        process(ctx *Context) (err error)
}

type baseNodeStruct struct {
        l *lex
        childNodes []node
        posbeg, posend int
}

func (n *baseNodeStruct) children() []node {
        return n.childNodes
}

func (n *baseNodeStruct) child(i int) node {
        if 0 <= i && i < len(n.childNodes) {
                return n.childNodes[i]
        }
        return nil
}

func (n *baseNodeStruct) setChildren(ca ...node) {
        n.childNodes = ca
}

func (n *baseNodeStruct) addChild(c node) {
        n.childNodes = append(n.childNodes, c)
}

func (n *baseNodeStruct) len() int {
        return n.posend - n.posbeg
}

func (n *baseNodeStruct) str() (s string) {
        if a, b := n.posbeg, n.posend; /*n.l != nil &&*/ a < b {
                //fmt.Printf("%v, %v, %v\n", a, b, len(n.l.s))
                s = string(n.l.s[a:b])
        }
        return
}

func (n *baseNodeStruct) getPosBeg() int {
        return n.posbeg
}

func (n *baseNodeStruct) getPosEnd() int {
        return n.posend
}

func (n *baseNodeStruct) setPosBeg(i int) {
        n.posbeg = i
}

func (n *baseNodeStruct) setPosEnd(i int) {
        n.posend = i
}

func (n *baseNodeStruct) addPosBeg(i int) {
        n.posbeg += i
}

func (n *baseNodeStruct) addPosEnd(i int) {
        n.posend += i
}

func (n *baseNodeStruct) loc() location {
        return location{n.posbeg, n.posend}
}

func (n *baseNodeStruct) expand(ctx *Context) (s string) {
        if nc := len(n.childNodes); 0 < nc {
                b, l, pos := new(bytes.Buffer), n.l, n.posbeg
                for _, c := range n.childNodes {
                        cpos := c.getPosBeg()
                        if pos < cpos { b.Write(l.s[pos:cpos]) }
                        pos = c.getPosEnd()
                        switch c.(type) {
                        case *nodeNamePrefix: b.WriteString(":")
                        case *nodeNamePart:   b.WriteString(".")
                        default: b.WriteString(c.Expand(ctx))
                        }
                }
                if pos < n.posend {
                        b.Write(l.s[pos:n.posend])
                }
                s = b.String()
        } else {
                s = n.str()
        }
        return
}

func (n *baseNodeStruct) items() (is Items) {
        // TODO: ...
        return
}

type nodeBase struct {
        baseNodeStruct
}

func (n *nodeBase) Expand(ctx *Context) (s string) {
        return
}

func (n *nodeBase) IsEmpty(ctx *Context) bool {
        return true
}

func (n *nodeBase) reset(bs baseNodeStruct) {
        n.baseNodeStruct = bs
}

func (n *nodeBase) bs() *baseNodeStruct {
        return &n.baseNodeStruct
}

func (n *nodeBase) process(ctx *Context) (err error) {
        return
}

func (n *nodeBase) processRule(ctx *Context, rn node) (err error) {
        var ns namespace
        if ctx.m == nil {
                ns = ctx.g
        } else {
                ns = ctx.m
        }

        r := ns.link(Split(n.childNodes[0].Expand(ctx))...)
        r.prerequisites, r.node = Split(n.childNodes[1].Expand(ctx)), rn
        if 2 < len(n.childNodes) {
                for _, c := range n.childNodes[2].children() {
                        r.recipes = append(r.recipes, c)
                }
        }

        // Set goal if empty.
        if 0 < len(r.targets) {
                if g := r.ns.getGoal(); g == "" {
                        r.ns.setGoal(r.targets[0])
                }
        }

        switch rn.(type) {
        case *nodeRulePhony:            r.c = &phonyTargetUpdater{}
        case *nodeRuleChecker:          r.c = &checkRuleUpdater{ r }
        case *nodeRuleDoubleColoned:    r.c = &defaultTargetUpdater{}
        case *nodeRuleSingleColoned:    r.c = &defaultTargetUpdater{}
        default: errorf("unexpected rule type: %v", rn.kind())
        }

        /*
        lineno, colno := ctx.l.caculateLocationLineColumn(n.loc())
        fmt.Fprintf(os.Stderr, "%v:%v:%v: %v\n", ctx.l.scope, lineno, colno, n.kind) //*/
        return
}

func copy_bs(n0, n1 node) node {
        n1.reset(*n0.bs())
        return n1
}

type nodeTypeCode int

const (
        nodeTypeCodeComment nodeTypeCode = iota
        nodeTypeCodeEscape
        nodeTypeCodeDeferredText
        nodeTypeCodeImmediateText
        nodeTypeCodeName
        nodeTypeCodeNamePrefix
        nodeTypeCodeNamePart
        nodeTypeCodeArg
        nodeTypeCodeValueText
        nodeTypeCodeDefineDeferred
        nodeTypeCodeDefineQuestioned
        nodeTypeCodeDefineSingleColoned
        nodeTypeCodeDefineDoubleColoned
        nodeTypeCodeDefineNot
        nodeTypeCodeDefineAppend
        nodeTypeCodeRuleSingleColoned
        nodeTypeCodeRuleDoubleColoned
        nodeTypeCodeRulePhony
        nodeTypeCodeRuleChecker
        nodeTypeCodeTargets
        nodeTypeCodePrerequisites
        nodeTypeCodeRecipes
        nodeTypeCodeRecipe
        nodeTypeCodeCall
        nodeTypeCodeSpeak
        nodeTypeCodeInclude
        nodeTypeCodeTemplate
        nodeTypeCodeModule
        nodeTypeCodeCommit
        nodeTypeCodePost
        nodeTypeCodeUse
)

type (
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
        nodeComment             struct { nodeBase }
        nodeEscape              struct { nodeBase }
        nodeDeferredText        struct { nodeBase }
        nodeImmediateText       struct { nodeBase }
        nodeName                struct { nodeBase }
        nodeNamePrefix          struct { nodeBase } // :
        nodeNamePart            struct { nodeBase } // .
        nodeArg                 struct { nodeBase }
        nodeValueText           struct { nodeBase } // value text for defines (TODO: use it)
        nodeDefineDeferred      struct { nodeBase } //  =     deferred
        nodeDefineQuestioned    struct { nodeBase } // ?=     deferred
        nodeDefineSingleColoned struct { nodeBase } // :=     immediate
        nodeDefineDoubleColoned struct { nodeBase } // ::=    immediate
        nodeDefineNot           struct { nodeBase } // !=     immediate
        nodeDefineAppend        struct { nodeBase } // +=     deferred or immediate (parsed into deferred)
        nodeRuleSingleColoned   struct { nodeBase } // :
        nodeRuleDoubleColoned   struct { nodeBase } // ::
        nodeRulePhony           struct { nodeBase } // :!:    phony target
        nodeRuleChecker         struct { nodeBase } // :?:    check if the target is updated
        nodeTargets             struct { nodeBase }
        nodePrerequisites       struct { nodeBase }
        nodeRecipes             struct { nodeBase }
        nodeRecipe              struct { nodeBase }
        nodeCall                struct { nodeBase }
        nodeSpeak               struct { nodeBase } // $(speak dialect, ...)
        nodeInclude             struct { nodeBase } // include filename
        nodeTemplate            struct { nodeBase } // template name, parameters
        nodeModule              struct { nodeBase } // module name, temp, parameters
        nodeCommit              struct { nodeBase } // commit
        nodePost                struct { nodeBase } // post
        nodeUse                 struct { nodeBase } // use name
)

func (n *nodeComment)             tc() nodeTypeCode { return nodeTypeCodeComment }
func (n *nodeEscape)              tc() nodeTypeCode { return nodeTypeCodeEscape }
func (n *nodeDeferredText)        tc() nodeTypeCode { return nodeTypeCodeDeferredText }
func (n *nodeImmediateText)       tc() nodeTypeCode { return nodeTypeCodeImmediateText }
func (n *nodeName)                tc() nodeTypeCode { return nodeTypeCodeName }
func (n *nodeNamePrefix)          tc() nodeTypeCode { return nodeTypeCodeNamePrefix }
func (n *nodeNamePart)            tc() nodeTypeCode { return nodeTypeCodeNamePart }
func (n *nodeArg)                 tc() nodeTypeCode { return nodeTypeCodeArg }
func (n *nodeValueText)           tc() nodeTypeCode { return nodeTypeCodeValueText }
func (n *nodeDefineDeferred)      tc() nodeTypeCode { return nodeTypeCodeDefineDeferred }
func (n *nodeDefineQuestioned)    tc() nodeTypeCode { return nodeTypeCodeDefineQuestioned }
func (n *nodeDefineSingleColoned) tc() nodeTypeCode { return nodeTypeCodeDefineSingleColoned }
func (n *nodeDefineDoubleColoned) tc() nodeTypeCode { return nodeTypeCodeDefineDoubleColoned }
func (n *nodeDefineNot)           tc() nodeTypeCode { return nodeTypeCodeDefineNot }
func (n *nodeDefineAppend)        tc() nodeTypeCode { return nodeTypeCodeDefineAppend }
func (n *nodeRuleSingleColoned)   tc() nodeTypeCode { return nodeTypeCodeRuleSingleColoned }
func (n *nodeRuleDoubleColoned)   tc() nodeTypeCode { return nodeTypeCodeRuleDoubleColoned }
func (n *nodeRulePhony)           tc() nodeTypeCode { return nodeTypeCodeRulePhony }
func (n *nodeRuleChecker)         tc() nodeTypeCode { return nodeTypeCodeRuleChecker }
func (n *nodeTargets)             tc() nodeTypeCode { return nodeTypeCodeTargets }
func (n *nodePrerequisites)       tc() nodeTypeCode { return nodeTypeCodePrerequisites }
func (n *nodeRecipes)             tc() nodeTypeCode { return nodeTypeCodeRecipes }
func (n *nodeRecipe)              tc() nodeTypeCode { return nodeTypeCodeRecipe }
func (n *nodeCall)                tc() nodeTypeCode { return nodeTypeCodeCall }
func (n *nodeSpeak)               tc() nodeTypeCode { return nodeTypeCodeSpeak }
func (n *nodeInclude)             tc() nodeTypeCode { return nodeTypeCodeInclude }
func (n *nodeTemplate)            tc() nodeTypeCode { return nodeTypeCodeTemplate }
func (n *nodeModule)              tc() nodeTypeCode { return nodeTypeCodeModule }
func (n *nodeCommit)              tc() nodeTypeCode { return nodeTypeCodeCommit }
func (n *nodePost)                tc() nodeTypeCode { return nodeTypeCodePost }
func (n *nodeUse)                 tc() nodeTypeCode { return nodeTypeCodeUse }

func (n *nodeComment)             kind() string { return "comment" }
func (n *nodeEscape)              kind() string { return "escape" }
func (n *nodeDeferredText)        kind() string { return "deferred-text" }
func (n *nodeImmediateText)       kind() string { return "immediate-text" }
func (n *nodeName)                kind() string { return "name" }
func (n *nodeNamePrefix)          kind() string { return "name-prefix" }
func (n *nodeNamePart)            kind() string { return "name-part" }
func (n *nodeArg)                 kind() string { return "arg" }
func (n *nodeValueText)           kind() string { return "value-text" }
func (n *nodeDefineDeferred)      kind() string { return "define-deferred" }
func (n *nodeDefineQuestioned)    kind() string { return "define-questioned" }
func (n *nodeDefineSingleColoned) kind() string { return "define-single-coloned" }
func (n *nodeDefineDoubleColoned) kind() string { return "define-double-coloned" }
func (n *nodeDefineNot)           kind() string { return "define-not" }
func (n *nodeDefineAppend)        kind() string { return "define-append" }
func (n *nodeRuleSingleColoned)   kind() string { return "rule-single-coloned" }
func (n *nodeRuleDoubleColoned)   kind() string { return "rule-double-coloned" }
func (n *nodeRulePhony)           kind() string { return "rule-phony" }
func (n *nodeRuleChecker)         kind() string { return "rule-checker" }
func (n *nodeTargets)             kind() string { return "targets" }
func (n *nodePrerequisites)       kind() string { return "prerequisites" }
func (n *nodeRecipes)             kind() string { return "recipes" }
func (n *nodeRecipe)              kind() string { return "recipe" }
func (n *nodeCall)                kind() string { return "call" }
func (n *nodeSpeak)               kind() string { return "speak" }
func (n *nodeInclude)             kind() string { return "include" }
func (n *nodeTemplate)            kind() string { return "template" }
func (n *nodeModule)              kind() string { return "module" }
func (n *nodeCommit)              kind() string { return "commit" }
func (n *nodePost)                kind() string { return "post" }
func (n *nodeUse)                 kind() string { return "use" }

func (n *nodeEscape) Expand(ctx *Context) (s string) {
        switch n.l.s[n.posbeg + 1] {
        case '\n': s = " "
        case '#':  s = "#"
        }
        return
}

func (n *nodeEscape) IsEmpty(ctx *Context) bool {
        return n.Expand(ctx) == ""
}

func (n *nodeDeferredText) Expand(ctx *Context) (s string) {
        return n.expand(ctx)
}

func (n *nodeDeferredText) IsEmpty(ctx *Context) bool {
        return n.expand(ctx) == ""
}

func (n *nodeImmediateText) Expand(ctx *Context) (s string) {
        return n.expand(ctx)
}

func (n *nodeImmediateText) IsEmpty(ctx *Context) bool {
        return n.expand(ctx) == ""
}

func (n *nodeName) Expand(ctx *Context) (s string) {
        return n.expand(ctx)
}

func (n *nodeName) IsEmpty(ctx *Context) bool {
        return n.expand(ctx) == ""
}

func (n *nodeArg) Expand(ctx *Context) (s string) {
        return n.expand(ctx)
}

func (n *nodeArg) IsEmpty(ctx *Context) bool {
        return n.expand(ctx) == ""
}

func (n *nodeTargets) Expand(ctx *Context) (s string) {
        return n.expand(ctx)
}

func (n *nodeTargets) IsEmpty(ctx *Context) bool {
        return n.expand(ctx) == ""
}

func (n *nodePrerequisites) Expand(ctx *Context) (s string) {
        return n.expand(ctx)
}

func (n *nodePrerequisites) IsEmpty(ctx *Context) bool {
        return n.expand(ctx) == ""
}

func (n *nodeRecipes) Expand(ctx *Context) (s string) {
        return n.expand(ctx)
}

func (n *nodeRecipes) IsEmpty(ctx *Context) bool {
        return n.expand(ctx) == ""
}

func (n *nodeRecipe) Expand(ctx *Context) (s string) {
        return n.expand(ctx)
}

func (n *nodeRecipe) IsEmpty(ctx *Context) bool {
        return n.expand(ctx) == ""
}

func (n *nodeCall) Expand(ctx *Context) (s string) {
        var args Items
        for _, an := range n.children()[1:] {
                args = args.Concat(ctx, stringitem(an.Expand(ctx)))
        }
        name := ctx.ParseName(n.childNodes[0].Expand(ctx))
        is := ctx.call(n.loc(), name, args...)
        return is.Expand(ctx)
}

func (n *nodeCall) IsEmpty(ctx *Context) bool {
        return n.Expand(ctx) == ""
}

func (n *nodeSpeak) Expand(ctx *Context) (s string) {
        var is Items
        dialect := n.childNodes[0].Expand(ctx)
        if 1 < len(n.childNodes) {
                is = ctx.speak(dialect, n.childNodes[1:]...)
        } else {
                is = ctx.speak(dialect)
        }
        return is.Expand(ctx)
}

func (n *nodeSpeak) IsEmpty(ctx *Context) bool {
        return n.Expand(ctx) == ""
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
        node node
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

        nodes []node // parsed top level nodes
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

func (l *lex) reset(n node) node {
        n.reset(baseNodeStruct{ l:l, posbeg:l.pos, posend:l.pos })
        return n
}

func (l *lex) push(n node, ns func(), c int) *lexStack {
        ls := &lexStack{ node:l.reset(n), state:l.step, code:c }
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
                t.childNodes[0].str(), t.str(), t.childNodes[1].str(), len(l.stack), l.pos, l.rune) //*/

        // Pop out and append the node.
        l.nodes = append(l.nodes, t)
}

func (l *lex) stateGlobal() {
state_loop:
        for l.get() {
                switch {
                case l.rune == '#':
                        st := l.push(new(nodeComment), l.stateComment, 0)
                        st.node.addPosBeg(-1) // for the '#'
                        break state_loop
                case l.rune != rune(0) && !unicode.IsSpace(l.rune):
                        l.unget() // Put back the rune.
                        l.push(new(nodeImmediateText), l.stateLineHeadText, 0)
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
                        st, posend := l.pop(), l.pos
                        if l.rune == '\n' {
                                posend-- // exclude the '\n'
                        }
                        st.node.setPosEnd(posend)

                        /*
                        lineno, colno := l.caculateLocationLineColumn(st.node.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: stateComment: (stack=%v) %v\n", l.scope, lineno, colno, len(l.stack), st.node.str()) //*/

                        if 0 < len(l.stack) {
                                c := st.node
                                l.top().node.addChild(c)
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
                        for s, f := range statements {
                                if pos := l.pos; l.looking(s, &pos) {
                                        //fmt.Printf("stateLineHeadText: %v (%v)\n", string(l.rune), s)
                                        if ss := pos; l.lookingInlineSpaces(&ss) {
                                                st.node = f(st.node.bs())
                                                st.node.setPosEnd(pos)
                                                l.pos = ss
                                                if r := l.peek(); r == '\n' || r == '#' {
                                                        l.pop() // end of statement
                                                        l.nodes = append(l.nodes, st.node)
                                                } else {
                                                        //fmt.Printf("stateLineHeadText: %v (%v)\n", st.node.kind, st.node.str())
                                                        l.push(new(nodeArg), l.stateStatementArg, 0)
                                                }
                                                break state_loop
                                        }
                                }
                        }
                }

                switch {
                case l.rune == '$':
                        st := l.push(new(nodeCall), l.stateDollar, 0)
                        st.node.addPosBeg(-1) // for the '$'
                        break state_loop

                case l.rune == '=':
                        l.top().code = int(nodeTypeCodeDefineDeferred)
                        l.step = l.stateDefine
                        break state_loop

                case l.rune == '?':
                        if l.peek() == '=' {
                                l.get() // consume the '=' for '?='
                                l.top().code = int(nodeTypeCodeDefineQuestioned)
                                l.step = l.stateDefine
                                break state_loop
                        }

                case l.rune == '!':
                        if l.peek() == '=' {
                                l.get() // consume the '=' for '!='
                                l.top().code = int(nodeTypeCodeDefineNot)
                                l.step = l.stateDefine
                                break state_loop
                        }

                case l.rune == '+':
                        if l.peek() == '=' {
                                l.get() // consume the '=' for '+='
                                l.top().code = int(nodeTypeCodeDefineAppend)
                                l.step = l.stateDefine
                                break state_loop
                        }

                case l.rune == ':':
                        if r := l.peek(); r == '=' {
                                l.get() // consume the '=' for ':='
                                l.top().code = int(nodeTypeCodeDefineSingleColoned)
                                l.step = l.stateDefine
                        } else {
                                n := l.top().node
                                n.setPosEnd(l.backwardNonSpace(n.getPosBeg(), l.pos-1))
                                l.step = l.stateRule
                        }
                        break state_loop

                case l.rune == '.':
                        part := l.reset(new(nodeNamePart))
                        part.setPosBeg(l.pos - 1)
                        st := l.top()
                        st.node.addChild(part)

                case l.rune == '#': fallthrough
                case l.rune == '\n':
                        st := l.pop() // pop out the node
                        st.node.setPosEnd(l.pos-1)

                        // append the island text
                        l.nodes = append(l.nodes, st.node)

                        if l.rune == '#' {
                                st = l.push(new(nodeComment), l.stateComment, 0)
                                st.node.addPosBeg(-1) // for the '#'
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
                                st.node.setPosBeg(l.pos - 1)
                        }
                }
                
                switch {
                case l.rune == '$':
                        l.push(new(nodeCall), l.stateDollar, 0).node.addPosBeg(-1) // 'pos--' for the '$'
                        break state_loop
                case l.rune == '\\':
                        l.escapeTextLine(st.node)
                case l.rune == ',': fallthrough
                case l.rune == '\n':
                        arg := st.node
                        arg.setPosEnd(l.pos - 1)
                        l.pop()

                        st = l.top()
                        st.node.addChild(arg)
                        if l.rune == '\n' {
                                l.pop() // end of statement
                                l.nodes = append(l.nodes, st.node)
                                //fmt.Printf("%v: %v %v\n", st.node.kind, st.node.str(), st.node.children)
                        } else {
                                l.push(new(nodeArg), l.stateStatementArg, 0)
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
                name, n = st.node, 2
                v, t node
        )
        switch nodeTypeCode(st.code) {
        case nodeTypeCodeDefineDeferred:
                t = new(nodeDefineDeferred)
                v = new(nodeDeferredText)
                n = 1 // len("=")
        case nodeTypeCodeDefineQuestioned:
                t = new(nodeDefineQuestioned)
                v = new(nodeDeferredText)
                n = 2 // len("?=")
        case nodeTypeCodeDefineSingleColoned:
                t = new(nodeDefineSingleColoned)
                v = new(nodeImmediateText)
                n = 2 // len(":=")
        case nodeTypeCodeDefineDoubleColoned:
                t = new(nodeDefineDoubleColoned)
                v = new(nodeImmediateText)
                n = 3 // len("::=")
        case nodeTypeCodeDefineNot:
                t = new(nodeDefineNot)
                v = new(nodeImmediateText)
                n = 2 // len("!=")
        case nodeTypeCodeDefineAppend:
                t = new(nodeDefineAppend)
                v = new(nodeDeferredText)
                n = 2 // len("+=")
        default:
                panic(fmt.Sprintf("unexpected define code [%v]", st.code))
        }

        name.setPosEnd(l.backwardNonSpace(st.node.getPosBeg(), l.pos-n)) // for '=', '+=', '?=', ':=', '::='

        st = l.push(t, l.stateAppendNode, 0)
        st.node.setChildren(name)
        st.node.addPosBeg(-n) // for '=', '+=', '?=', ':='

        // Create the value node.
        value := l.push(v, l.stateDefineValueLine, 0).node
        st.node.addChild(value)
}

func (l *lex) stateDefineValueLine() {
        st := l.top()
state_loop:
        for l.get() {
                if st.code == 0 { // skip spaces after '='
                        if !unicode.IsSpace(l.rune) {
                                st.node.setPosBeg(l.pos-1)
                                st.code = 1
                        } else if l.rune != '\n' /* IsSpace */ {
                                st.node.setPosBeg(l.pos)
                                continue
                        }
                }

                switch {
                case l.rune == '$':
                        st = l.push(new(nodeCall), l.stateDollar, 0)
                        st.node.addPosBeg(-1) // for the '$'
                        break state_loop

                case l.rune == '#':
                        l.unget() // Put back the '#', then fall through.
                        fallthrough
                case l.rune == '\n': fallthrough
                case l.rune == rune(0): // The end of string.
                        st.node.setPosEnd(l.pos)
                        if l.rune == '\n' {
                                st.node.addPosEnd(-1) // Exclude the '\n'.
                        }

                        l.pop() // Pop out the value node and forward to the define node.
                        break state_loop

                default: l.escapeTextLine(l.top().node)
                }
        }
}

func (l *lex) escapeTextLine(t node) {
        if l.rune != '\\' { return }

        // Escape: \\n \#
        if l.get() { // get the char right next to '\\'
                switch l.rune {
                case '#': fallthrough
                case '\n':
                        en := l.reset(new(nodeEscape))
                        en.addPosBeg(-2) // for the '\\\n', '\\#', etc.
                        if l.rune == '\n' {
                                /* FIXME: skip spaces after '\\\n' ?
                                for unicode.IsSpace(l.peek()) {
                                        l.get()
                                        en.end = l.pos
                                } */
                        }
                        t.addChild(en)
                }
        }
}

func (l *lex) stateRule() {
        var t node
        rs, n := l.peekN(2), 1 // Assuming single colon.

        if len(rs) == 2 {
                switch {
                case rs[0] == ':': // targets :: blah blah blah
                        l.get() // drop the second ':'
                        t, n = new(nodeRuleDoubleColoned), 2
                        if rs[1] == '=' { // ::=
                                l.get() // consume the '=' for '::='
                                l.top().code = int(nodeTypeCodeDefineDoubleColoned)
                                l.step = l.stateDefine
                                return
                        }
                case rs[0] == '!' && rs[1] == ':': // targets :!:
                        l.get(); l.get() // drop the "!:"
                        t, n = new(nodeRulePhony), 3
                case rs[0] == '?' && rs[1] == ':': // targets :?:
                        l.get(); l.get() // drop the "?:"
                        t, n = new(nodeRuleChecker), 3
                }
        }
        if t == nil {
                t = new(nodeRuleSingleColoned)
        }

        targets := l.pop().node
        targets = copy_bs(targets, new(nodeTargets))

        st := l.push(t, l.stateAppendNode, 0)
        st.node.setChildren(targets)
        st.node.addPosBeg(-n) // for the ':', '::', ':!:', ':?:'

        prerequisites := l.push(new(nodePrerequisites), l.stateRuleTextLine, 0).node
        st.node.addChild(prerequisites)
}

func (l *lex) stateRuleTextLine() {
        st := l.top()
state_loop:
        for l.get() {
                if st.code == 0 && (l.rune == '\n' || !unicode.IsSpace(l.rune)) { // skip spaces after ':' or '::'
                        st.node.setPosBeg(l.pos-1)
                        st.code = 1
                }

                switch {
                case l.rune == '$':
                        st = l.push(new(nodeCall), l.stateDollar, 0)
                        st.node.addPosBeg(-1) // for the '$'
                        break state_loop

                case l.rune == '#':  fallthrough
                case l.rune == ';':  fallthrough
                case l.rune == '\n': fallthrough
                case l.rune == rune(0): // end of string
                        st.node.setPosEnd(l.pos)
                        if l.rune != rune(0) {
                                st.node.addPosEnd(-1) // exclude the '\n' or '#'
                        }

                        /*
                        lineno, colno := l.caculateLocationLineColumn(st.node.loc())
                        fmt.Fprintf(os.Stderr, "%v:%v:%v: stateRuleTextLine: %v\n", l.scope, lineno, colno, st.node.str()) //*/

                        st = l.pop() // pop out the prerequisites node
                        switch l.rune {
                        case '#':
                                st = l.push(new(nodeComment), l.stateComment, 0)
                                st.node.addPosBeg(-1) // for the '#'
                        case ';':
                                st.node.setPosEnd(l.backwardNonSpace(st.node.getPosBeg(), l.pos-1))
                                recipes := l.push(new(nodeRecipes), l.stateTabbedRecipes, 0).node
                                recipes.addPosBeg(-1) // includes ';'

                                l.pos = l.forwardNonSpaceInline(l.pos)
                                l.push(new(nodeRecipe), l.stateRecipe, 0)
                        case '\n':
                                if p := l.peek(); p == '\t' || p == '#' {
                                        st = l.push(new(nodeRecipes), l.stateTabbedRecipes, 0)
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
                        st = l.push(new(nodeRecipe), l.stateRecipe, 0)
                        //st.node.base().pos-- // for the '\t'

                case l.rune == '#':
                        st = l.push(new(nodeComment), l.stateComment, 0)
                        st.node.addPosBeg(-1) // for the '#'

                default:
                        recipes := st.node // the recipes node
                        recipes.setPosEnd(l.pos)
                        if l.rune == '\n' {
                                recipes.addPosEnd(-1)
                        } else if l.rune != rune(0) {
                                recipes.addPosEnd(-1)
                                l.unget() // put back the non-space character following by a recipe
                        }

                        st = l.pop() // pop out the recipes
                        st = l.top() // the rule node
                        st.node.addChild(recipes)
                }
        }
}

func (l *lex) stateRecipe() { // tab-indented action of a rule
        st := l.top()
state_loop:
        for l.get() {
                switch {
                case l.rune == '$':
                        st = l.push(new(nodeCall), l.stateDollar, 0)
                        st.node.addPosBeg(-1) // for the '$'
                        break state_loop
                case l.rune == '\n': fallthrough
                case l.rune == rune(0): // end of string
                        recipe := st.node
                        recipe.setPosEnd(l.pos)
                        if l.rune != rune(0) {
                                recipe.addPosEnd(-1) // exclude the '\n'
                        }

                        l.pop() // pop out the node

                        st = l.top()
                        st.node.addChild(recipe)
                        break state_loop
                }
        }
}

func (l *lex) stateDollar() {
        if l.get() {
                switch {
                case l.rune == '(': l.push(new(nodeName), l.stateCallName, 0).delm = ')'
                case l.rune == '{': l.push(new(nodeName), l.stateCallName, 0).delm = '}'
                default:
                        name := l.reset(new(nodeName))
                        name.setPosBeg(l.pos - 1) // include the single char
                        st := l.top() // nodeCall
                        st.node.addChild(name)
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
                        l.push(new(nodeCall), l.stateDollar, 0).node.addPosBeg(-1) // 'pos--' for the '$'
                        break state_loop
                case l.rune == ':' && st.code == 0:
                        prefix := l.reset(new(nodeNamePrefix))
                        prefix.setPosBeg(l.pos - 1)
                        st.node.addChild(prefix)
                        st.code++
                case l.rune == '.':
                        part := l.reset(new(nodeNamePart))
                        part.setPosBeg(l.pos - 1)
                        st.node.addChild(part)
                case l.rune == '\\':
                        l.escapeTextLine(st.node)
                case l.rune == ' ': fallthrough
                case l.rune == delm:
                        name := st.node
                        name.setPosEnd(l.pos - 1)
                        l.pop()

                        st = l.top()
                        switch s := name.str(); s {
                        case "speak":
                                st.node = copy_bs(st.node, new(nodeSpeak))
                                if l.rune != delm {
                                        l.push(new(nodeArg), l.stateSpeakDialect, 0).delm = delm
                                } else {
                                        lineno, colno := l.getLineColumn()
                                        errorf("%v:%v:%v: unexpected delimiter\n", l.scope, lineno, colno)
                                }
                        default:
                                st.node.addChild(name)
                                switch l.rune {
                                case delm:
                                        l.endCall(st)
                                case ' ':
                                        l.push(new(nodeArg), l.stateCallArg, 0).delm = delm
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
                        l.push(new(nodeCall), l.stateDollar, 0).node.addPosBeg(-1) // 'pos--' for the '$'
                        break state_loop
                case l.rune == '\\':
                        l.escapeTextLine(st.node)
                case l.rune == ',': fallthrough
                case l.rune == delm:
                        arg := st.node
                        arg.setPosEnd(l.pos - 1)
                        l.pop()

                        st = l.top()
                        st.node.addChild(arg)
                        if l.rune == delm {
                                l.endCall(st)
                        } else {
                                l.push(new(nodeArg), l.stateCallArg, 0).delm = delm
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
                        l.push(new(nodeCall), l.stateDollar, 0).node.addPosBeg(-1) // 'pos--' for the '$'
                        break state_loop
                case l.rune == '\\':
                        l.escapeTextLine(st.node)
                case l.rune == ',': fallthrough
                case l.rune == delm:
                        arg := st.node
                        arg.setPosEnd(l.pos - 1)
                        l.pop()

                        st = l.top()
                        st.node.addChild(arg)
                        if l.rune == delm {
                                l.endCall(st)
                        } else {
                                l.push(new(nodeArg), l.stateSpeakScript, 0).delm = delm
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
                        l.push(new(nodeCall), l.stateDollar, 0).node.addPosBeg(-1) // 'pos--' for the '$'
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
                        st.node.setPosBeg(l.pos)
                        st.code = 1
                case l.rune == '\n' && st.code == 1 && l.peek() == '-':
                delimiter_loop:
                        for i, r, n := l.pos, rune(0), 1; i < len(l.s); i += n {
                                switch r, n = utf8.DecodeRune(l.s[i:]); r {
                                default: break delimiter_loop
                                case '-': /* skip */
                                case delm:
                                        script := st.node
                                        script.setPosEnd(l.backwardNonSpace(script.getPosBeg(), l.pos))

                                        l.rune, l.pos, st = delm, i+1, l.pop()

                                        st = l.top() // the $(speak) node
                                        st.node.addChild(script)
                                        l.endCall(st)
                                        break state_loop
                                }
                        }
                case l.rune == ',': fallthrough
                case l.rune == delm:
                        script := st.node
                        script.setPosEnd(l.pos - 1)

                        l.pop()

                        st = l.top() // the $(speak) node
                        st.node.addChild(script)
                        if l.rune == delm {
                                l.endCall(st)
                        } else {
                                l.push(new(nodeArg), l.stateSpeakScript, 0).delm = delm
                        }
                        break state_loop
                }
        }
}

func (l *lex) endCall(st *lexStack) {
        call := st.node
        call.setPosEnd(l.pos)

        l.pop() // pop out the current nodeCall

        // Append the call to it's parent.
        t := l.top().node
        t.addChild(call)
}

/*
stateDollar:
        st.node.children = []*node{ l.reset(new(nodeName)) }
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
                        st.node.child(0).setPosBeg(l.pos)
                        st.node.setPosEnd(l.pos)
                        st.code = name

                case l.rune == ' ' && st.code == name:
                        st.code = args; fallthrough
                case l.rune == ',' && st.code == args:
                        a := l.reset(new(nodeArg))
                        i := len(st.node.children())-1
                        st.node.child(i).setPosEnd(l.pos-1)
                        st.node.addChild(a)

                case l.rune == '\\':
                        i := len(st.node.children())-1
                        if 0 <= i {
                                l.escapeTextLine(st.node.child(i))
                        }

                case l.rune == '$' && st.code != init:
                        st = l.push(new(nodeCall), l.stateDollar, 0)
                        st.node.addPosBeg(-1) // for the '$'
                        break state_loop

                case l.rune == st.delm: //&& st.code != init:
                        fallthrough
                case st.code == init: // $$, $a, $<, $@, $^, etc.
                        call, i, n := st.node, len(st.node.children())-1, 1
                        if st.delm == rune(0) { n = 0 } // don't shift for single char like '$a'
                        call.child(i).setPosEnd(l.pos-n)
                        call.setPosEnd(l.pos)

                        l.pop() // pop out the current nodeCall

                        t := l.top().node
                        switch t.(type) {
                        case *nodeDeferredText:  t.addChild(call)
                        case *nodeImmediateText: t.addChild(call)
                        default:
                                // Add to the last child.
                                i = len(t.children())-1
                                if 0 <= i { t = t.child(i) }
                                t.addChild(call)
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

type committedModule struct {
        m *Module
        p *Context
        a Items
}

// Context hold a parse context and the current module being processed.
type Context struct {
        lexingStack []*lex
        moduleStack []*Module

        l *lex // the current lexer
        m *Module // the active module being processed
        t *template // the active template being processed
        tild *template // the active template referred by '~'

        g *namespaceBase // the global namespace

        templates map[string]*template

        modules map[string]*Module
        moduleOrderList []*Module
        moduleBuildList []committedModule

        w *worker.Worker
}

func (ctx *Context) GetModules() map[string]*Module { return ctx.modules }
func (ctx *Context) GetModuleOrderList() []*Module { return ctx.moduleOrderList }
func (ctx *Context) GetModuleBuildList() []committedModule { return ctx.moduleBuildList }
func (ctx *Context) ResetModules() {
        ctx.modules = make(map[string]*Module, 8)
        ctx.moduleOrderList = []*Module{}
        ctx.moduleBuildList = []committedModule{}
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
        return ctx.call(ctx.l.location(), ctx.ParseName(name), args...)
}

func (ctx *Context) Set(name string, items ...Item) {
        ctx.set(ctx.ParseName(name), items...)
}

func (ctx *Context) set(name Name, items ...Item) {
        if ns := ctx.getNamespace(name); ns != nil {
                ns.set(ctx, name.Sym, items...)
        } else {
                var loc = ctx.l.location()
                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                fmt.Fprintf(os.Stderr, "%v:%v:%v: missing '%s'\n", ctx.l.scope, lineno, colno, &name)
        }
}

func (ctx *Context) call(loc location, name Name, args ...Item) (is Items) {
        call := func(ns namespace) (called bool) {
                if ns == nil { return }
                if name.Prefixed {
                        if ht, ok := hooksMap[name.Prefix]; ok && ht != nil {
                                if h, ok := ht[name.Sym]; ok && h != nil {
                                        is, called = h(ctx, args), true
                                }
                        }
                }
                if !called {
                        if d := ns.get(ctx, name.Sym); d != nil {
                                is, called = d.value, true
                        }
                }
                return
        }
        
        // Process special symbols and builtins first.
        if name.Prefixed {
                if name.Prefix == "~" && ctx.tild != nil {
                        if name.Prefix = ctx.tild.name; call(ctx.getNamespace(name)) {
                                return
                        }
                }
        } else if len(name.Ns) == 0 {
                switch name.Sym {
                case "$":  is = append(is, stringitem("$"))
                case "me": // rename: $(me) -> $(me.name)
                        name.Ns = []string{ "me" }
                        name.Sym = "name"
                default:
                        if f, ok := builtins[name.Sym]; ok && f != nil {
                                is = f(ctx, loc, args)
                                return
                        }
                }
        }

        if ns := ctx.getNamespace(name); ns != nil {
                _ = call(ns)
        } else {
                lineno, colno := ctx.l.caculateLocationLineColumn(loc)
                fmt.Fprintf(os.Stderr, "%v:%v:%v: no namespace for '%s'\n", ctx.l.scope, lineno, colno, name)
        }
        return
}

// getDefine returns a define for hierarchy names like `tool:m1.m2.var`, `m1.m2.var`, etc.
func (ctx *Context) getDefine(name Name) (d *define) {
        if ns := ctx.getNamespace(name); ns != nil {
                d = ns.get(ctx, name.Sym)
        }
        return
}

func (ctx *Context) getNamespace(name Name) (ns namespace) {
        lineno, colno := ctx.l.caculateLocationLineColumn(ctx.l.location())
        
        if name.Prefixed {
                if name.Prefix == "~" {
                        if ctx.tild != nil {
                                ns, name.Prefix = ctx.tild.namespaceBase, ctx.tild.name
                        } else {
                                fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: tild is nil\n",
                                        ctx.l.scope, lineno, colno)
                                return
                        }
                } else if t, ok := ctx.templates[name.Prefix]; ok && t != nil {
                        ns = t.namespaceBase
                } else {
                        fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: undefined toolset prefix `%s'\n",
                                ctx.l.scope, lineno, colno, name.Prefix)
                        return
                }
        } else if len(name.Ns) == 0 && ns == nil {
                ns = ctx.g
                return
        }

        for i, s := range name.Ns {
                if ns != nil {
                        var nns namespace
                        if m, ok := ns.(*Module); ok && m != nil {
                                if c, ok := m.Children[s]; ok && c != nil {
                                        nns = c
                                }
                        }
                        ns = nns;
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
                                                ns = ctx.m.Templates[0].namespaceBase
                                        } else {
                                                fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: tild is referred to nil template\n", ctx.l.scope, lineno, colno)
                                                //break loop_parts
                                        }
                                } else {
                                        ns = ctx.tild.namespaceBase
                                }
                        }
                }
                if ns == nil {
                        fmt.Fprintf(os.Stderr, "%v:%v:%v:warning: `%s' is undefined scope\n", ctx.l.scope, lineno, colno,
                                strings.Join(name.Ns[0:i+1], "."))
                        break
                }
        }
        return
}

// ParseName parses a name like "prefix:a.b.c.var" into Name struct.
func (ctx *Context) ParseName(s string) (name Name) {
        if i := strings.Index(s, ":"); 0 <= i {
                name.Prefix, name.Prefixed = s[0:i], true
                s = s[i+1:]
        }
        a := strings.Split(s, ".")
        if n := len(a); 0 < n {
                name.Ns = a[0:n-1]
                name.Sym = a[n-1]
        }
        return
}

func (ctx *Context) speak(name string, scripts ...node) (is Items) {
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

func (ctx *Context) ItemsStrings(a ...Item) (s []string) {
        for _, i := range a {
                s = append(s, i.Expand(ctx))
        }
        return
}

func (ctx *Context) processNode(n node) (err error) {
        if ctx.t != nil {
                switch n.(type) {
                case *nodeCommit: processTemplateCommit(ctx, n)
                case *nodePost:   processTemplatePost(ctx, n)
                default:
                        if ctx.t.post != nil {
                                ctx.t.postNodes = append(ctx.t.postNodes, n)
                        } else {
                                ctx.t.declNodes = append(ctx.t.declNodes, n)
                        }
                }
        } else {
                err = n.process(ctx)
        }
        return
}

func (ctx *Context) parseBuffer() (err error) {
        if !ctx.l.parse() {
                err = errors.New("syntax error")
                return
        }

        for _, n := range ctx.l.nodes {
                if _, ok := n.(*nodeComment); ok { continue }
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

func (n *nodeCall) process(ctx *Context) (err error) {
        if s := strings.TrimSpace(n.Expand(ctx)); s != "" {
                lineno, colno := ctx.l.caculateLocationLineColumn(n.loc())
                fmt.Fprintf(os.Stderr, "%v:%v:%v: illigal: '%v'\n", ctx.l.scope, lineno, colno, s)
        }
        return
}

func (n *nodeImmediateText) process(ctx *Context) (err error) {
        if s := strings.TrimSpace(n.Expand(ctx)); s != "" {
                lineno, colno := ctx.l.caculateLocationLineColumn(n.loc())
                fmt.Fprintf(os.Stderr, "%v:%v:%v: syntax error: '%v'\n", ctx.l.scope, lineno, colno, s)
        }
        return
}

func (n *nodeDefineQuestioned) process(ctx *Context) (err error) {
        name := ctx.ParseName(n.childNodes[0].Expand(ctx))
        if is := ctx.call(n.loc(), name); is.IsEmpty(ctx) {
                ctx.set(name, n)
        }
        return
}

func (n *nodeDefineDeferred) process(ctx *Context) (err error) {
        ctx.set(ctx.ParseName(n.childNodes[0].Expand(ctx)), n.childNodes[1])
        return
}

func (n *nodeDefineSingleColoned) process(ctx *Context) (err error) {
        name := ctx.ParseName(n.childNodes[0].Expand(ctx))
        ctx.set(name, stringitem(n.childNodes[1].Expand(ctx)))
        return
}

func (n *nodeDefineDoubleColoned) process(ctx *Context) (err error) {
        name := ctx.ParseName(n.childNodes[0].Expand(ctx))
        ctx.set(name, stringitem(n.childNodes[1].Expand(ctx)))
        return
}

func (n *nodeDefineAppend) process(ctx *Context) (err error) {
        name := ctx.ParseName(n.childNodes[0].Expand(ctx))
        if d := ctx.getDefine(name); d != nil {
                d.value = append(d.value, n.childNodes[1])
        } else {
                ctx.set(name, stringitem(n.childNodes[1].Expand(ctx)))
        }
        return
}

func (n *nodeDefineNot) process(ctx *Context) (err error) {
        panic("'!=' not implemented")
}

func (n *nodeRuleSingleColoned) process(ctx *Context) (err error) {
        return n.processRule(ctx, n)
}

func (n *nodeRuleDoubleColoned) process(ctx *Context) (err error) {
        return n.processRule(ctx, n)
}

func (n *nodeRulePhony) process(ctx *Context) (err error) {
        return n.processRule(ctx, n)
}

func (n *nodeRuleChecker) process(ctx *Context) (err error) {
        return n.processRule(ctx, n)
}

func (n *nodeInclude) process(ctx *Context) (err error) {
        fmt.Printf("todo: %v %v\n", n.kind, n.children)
        return
}

func (n *nodeTemplate) process(ctx *Context) (err error) {
        var (
                args, loc = ctx.ItemsStrings(nodes2Items(n.children()...)...), n.loc()
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

                name := strings.TrimSpace(args[0])
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
                        namespaceBase: &namespaceBase{
                                defines: make(map[string]*define, 8),
                                rules: make(map[string]*rule, 4),
                                patts: make(map[string]*rule, 2),
                        },
                }

                ctx.t.set(ctx, "name", StringItem(name))
                
                if 1 < len(args) {
                        for _, a := range args[1:] {
                                name := strings.TrimSpace(a)
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

func (n *nodeModule) process(ctx *Context) (err error) {
        var (
                name, exportName string
                args, loc = ctx.ItemsStrings(nodes2Items(n.children()...)...), n.loc()
        )
        if 0 < len(args) { name = strings.TrimSpace(args[0]) }
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
                        namespaceBase: &namespaceBase{
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
                                name := strings.TrimSpace(nameItem)
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
                        namespaceBase: &namespaceBase{
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

func processTemplateCommit(ctx *Context, n node) (err error) {
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

func processTemplatePost(ctx *Context, n node) (err error) {
        //fmt.Printf("processTemplatePost: %v\n", n.children)
        if ctx.t != nil {
                ctx.t.post = n
        } else {
                errorf("template is nil")
        }
        return
}

func (n *nodeCommit) process(ctx *Context) (err error) {
        if ctx.m == nil {
                panic("nil module")
        }
        
        var (
                args, loc = ctx.ItemsStrings(nodes2Items(n.children()...)...), n.loc()
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
        ctx.moduleBuildList = append(ctx.moduleBuildList, committedModule{ctx.m, ctx, StringItems(args...)})
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

func (n *nodeUse) process(ctx *Context) (err error) {       
        if ctx.m == nil { errorf("no module defined") }

        var (
                args = ctx.ItemsStrings(nodes2Items(n.children()...)...)
                name = "using"
        )

        argItems := StringItems(args...)
        
        if d, ok := ctx.m.defines[name]; ok && d != nil {
                d.value = append(d.value, argItems...)
        } else {
                ctx.m.defines[name] = &define{
                        loc:ctx.CurrentLocation(),
                        name:name, value:argItems,
                }
        }
        return
}

func NewContext(scope string, s []byte, vars map[string]string) (ctx *Context, err error) {
        ctx = &Context{
                templates: make(map[string]*template, 8),
                modules: make(map[string]*Module, 8),
                l: &lex{ parseBuffer:&parseBuffer{ scope:scope, s: s }, pos: 0 },
                g: &namespaceBase{
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
