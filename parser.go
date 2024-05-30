///
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

type parseBits uint64
type specialRule int

const (
	parseBraced parseBits = 1<<iota // 00000000000000000000001
	parseGroup         // 00000000000000000000000010
	parseArged         // 00000000000000000000000100
	parseCall          // 00000000000000000000001000
	parseDOT           // 00000000000000000000010000
	parseDOTDOT        // 00000000000000000000100000
	parseDepend0       // 00000000000000000001000000
	parseModifier      // 00000000000000000010000000
	parseBARE          // 00000000000000000100000000
	parseGLOB          // 00000000000000001000000000
	parsePATH          // 00000000000000010000000000
	parsePERC          // 00000000000000100000000000
	parseREXP          // 00000000000001000000000000
	parseSELECT_PROP   // 00000000000010000000000000
	parseURL           // 00000000000100000000000000
	parseCompound      // 00000000001000000000000000
	parseDefineClause  // 00000000010000000000000000
	parseCodeBlock     // 00000001000000000000000000
	parseUndefValue    // 00000010000000000000000000
	parseForeachTempl  // 00000100000000000000000000
	parseIncludingConf // 00001000000000000000000000
	parseAutoName      // 00010000000000000000000000
	parseSpecialRule   // 00100000000000000000000000  e.g. :use ...:
	parseRecipeBuiltin // 01000000000000000000000000  recipe builtin command
	parseRecipeText    // 10000000000000000000000000
	parseRecipe = parseRecipeBuiltin | parseRecipeText

	// The parseNo* bits control the parsing priority!
	parseNoArg    = parseSELECT_PROP | parseDOT | parseDOTDOT /* | parsePATH */ | parsePERC
	parseNoPair   = parseSELECT_PROP | parseDOT | parsePATH | parsePERC
	parseNoURL    = parseSELECT_PROP | parseDOT | parsePATH | parseURL | parseGLOB | parsePERC | parseREXP /*| parseColonName*/ | parseSpecialRule
	parseNoPath   = parseSELECT_PROP | parseDOT | parsePATH | parseURL | parseGLOB | parsePERC | parseREXP
	parseNoSelect = parseSELECT_PROP | parseDOT
	parseNoGlob   = parseGLOB | parsePERC | parseREXP
	parseNoPerc   = parseGLOB | parsePERC | parseREXP
	parseNoRexp   = parseGLOB | parsePERC | parseREXP
)

const (
	specialRuleNor specialRule = iota // normal rules
	specialRuleUse // `use` rules
	specialRuleRec // recipe rules
)

const maxDigitAutoNum = 9

type usespec struct {
	props []Value
}

type parsedFile struct {
	// TODO: doc *commentGroup
	// TODO: comments *commentGroup
	keyword token // project, package or module
	position Position // position of the beginning, which has filename information
	name *barecomp // project/module name
	scope *Scope
	use []*usespec // imports
}

type parsedRuleData struct {
	position Position
	params  []string
	targets []Value
	depends []Value
	ordered []Value
	recipes []Value
	options []Value
	special specialRule
	config bool
}

type template struct {
	state scanstate
	end  *scanstate
	pos, endPos Pos // token position
	tok  token  // one token look-ahead
	lit  string // token literal
	verb string
	name Value // if only 'def', TODO: considering []Value for nested template defs?
	params []Value
}

type parse_left_hand_side struct { bool }

type parser struct {
	Context

	scanner scanner

	// Comments
	comments  []*commentGroup
	leadComment *commentGroup // last lead comment
	lineComment *commentGroup // last line comment

	// Next token
	pos, stop Pos // parsing and stop position
	tok token // one token look-ahead
	lit string // token literal

	templates []*template

	// Non-syntactic parser control
	exprLev int  // < 0: in control clause, >= 0: in expression
	inRhs   bool // if set, the parser is parsing a rhs expression

	bits parseBits

	// Ordinary identifier scopes
	imports []*usespec // list of imports

	targets []Value // targets of current rule
	params []*auto // parameters of current rule
	dialect string // recipe dialect of current rule
	configure bool // is parsing configure program?

	dd bool // helps debug parsing via `eval -dd=true{}`
}

func (p *parser) cast(t reflect.Type) Context { return implcast(p,t) }

func (p *parser) setbits(bits parseBits) { p.bits = bits }
func (p *parser) setbit(bit parseBits) (bits parseBits) {
	bits = p.bits
	p.bits |= bit
	return
}
func (p *parser) clearbit(bit parseBits) (bits parseBits) {
	bits = p.bits
	p.bits &= ^bit
	return
}

// ----------------------------------------------------------------------------
// Parsing support

func (p *parser) trace(a ...interface{}) { l_traverse.traceAt(p.Position(), a...) }

// Advance to the next token.
func (p *parser) scan() {
	// Because of one-token look-ahead, print the previous token
	// when tracing as it provides a more readable output. The
	// very first token (!p.pos.IsValid()) is not initialized
	// (it is ILLEGAL), so don't print it .
	if l_traverse.enabled && p.pos.IsValid() {
		s := p.tok.String()
		switch {
		case p.tok.isLiteral():
			p.trace(s, p.lit)
		case p.tok.isOperator(), p.tok.isKeyword():
			p.trace("\"" + s + "\"")
		default:
			p.trace(s)
		}
	}

	var pos = p.pos
	p.pos, p.tok, p.lit = p.scanner.scan()
	if false && p.lit == "none" { warn(p, "%v %v", p.tok, p.lit).debug(64); flush(p.Context) }
	if false && p.tok == EOF {
		erro(at(p,p.loc(pos)), "unexpected end of file").debug(1)
	}
}

// Consume a comment and return it and the line on which it ends.
func (p *parser) consumeComment() (res *comment, endline int) {
	// /*-style comments may end on a different line than where they start.
	// Scan the comment for '\n' chars and adjust endline accordingly.
	endline = p.scanner.file.Line(p.pos)
	if len(p.lit) > 1 && p.lit[1] == '*' {
		// don't use range here - no need to decode Unicode code points
		for i := 0; i < len(p.lit); i++ {
			if p.lit[i] == '\n' {
				endline++
			}
		}
	}

	res = &comment{p.Position(), p.lit}
	p.scan()

	return
}

// Consume a group of adjacent comments, add it to the parser's
// comments list, and return it together with the line at which
// the last comment in the group ends. A non-comment token or n
// empty lines terminate a comment group.
//
func (p *parser) consumecommentGroup(n int) (res *commentGroup, endline int) {
	res = new(commentGroup)
	p.comments = append(p.comments, res)
	endline = p.scanner.file.Line(p.pos)
	for p.tok == COMMENT && p.scanner.file.Line(p.pos) <= endline+n {
		var com *comment
		com, endline = p.consumeComment()
		res.list = append(res.list, com)
	}
	return
}

// Advance to the next non-comment token. In the process, collect
// any comment groups encountered, and remember the last lead and
// and line comments.
//
// A lead comment is a comment group that starts and ends in a
// line without any other tokens and that is followed by a non-comment
// token on the line immediately after the comment group.
//
// A line comment is a comment group that follows a non-comment
// token on the same line, and that has no tokens after it on the line
// where it ends.
//
// Lead and line comments may be considered documentation that is
// stored in the AST.
//
func (p *parser) step() {
	p.leadComment = nil
	p.lineComment = nil

	var prev = p.pos
	if p.scan(); p.tok == COMMENT {
		var comment *commentGroup
		var endline int

		// If the comment is on same line as the previous token; it
		// cannot be a lead comment but may be a line comment.
		if p.scanner.file.Line(p.pos) == p.scanner.file.Line(prev) {
			comment, endline = p.consumecommentGroup(0)
			if p.scanner.file.Line(p.pos) != endline {
				// The next token is on a different line, thus
				// the last comment group is a line comment.
				p.lineComment = comment
			}
		}

		// consume successor comments, if any
		endline = -1
		for p.tok == COMMENT {
			comment, endline = p.consumecommentGroup(1)
		}

		if endline+1 == p.scanner.file.Line(p.pos) {
			// The next token is following on the line immediately after the
			// comment group, thus the last comment group is a lead comment.
			p.leadComment = comment
		}
	}

	// if p.tok != LINEND && p.lineComment != nil { p.tok = LINEND }

	if p.dd {
		var t = warn(p, "%v %v %v", p.tok, p.lit, p.scanner.scanstate)
		if p.tok == COMPOUND { t.debug(12) }
		if p.tok == LINEND { t.debug(24) }
		flush(p.Context)
	}
}
func (p *parser) next(ws bool) { if p.step(); ws { p.spaces() } }
func (p *parser) spaces() {
	for p.lineComment == nil && p.tok != EOF {
		if p.tok == SPACE || (p.tok == RECIPE && p.bits&parseRecipeBuiltin != 0) {
			p.step()
		} else if p.tok == ESCAPE && p.lit == "\n" {
			if p.step(); p.tok == LINEND || p.lineComment != nil { break }
			if p.bits&parseRecipeBuiltin != 0 {
				TokFor: for p.tok != EOF {
					switch p.tok {
					case RECIPE: // TODO: using p.isRecipeStart()
						if true { p.scanner.pop(isCompoundLine) }
						p.step()
					default: break TokFor
					}
				}
			}
		} else {
			break
		}
	}
}

func (p *parser) Position() Position { return p.loc(p.pos) }
func (p *parser) loc(pos Pos) Position { return Position(p.scanner.file.Position(pos)) }
func (p *parser) ctx(ctx Context) Context { return &positional{ctx, p.Position()} }
func (p *parser) valbase() valbase { return valbase{p.loc(p.pos)} }

// A bailout panic is raised to indicate early termination.
type bailout struct{}

func (p *parser) expected(pos Pos, msg string, a... interface{}) {
	if len(a) > 0 { msg = fmt.Sprintf(msg, a...) }
	if msg = "expected " + msg; pos == p.pos {
		// the error happened at the current position;
		// make the error message more specific
		if p.tok == SEMICOLON && p.lit == "\n" {
			msg += ", found newline"
		} else {
			msg += ", found " + p.tok.String()
			if p.tok.isLiteral() {
				msg += " '" + p.lit + "'"
			}
		}
	}
	erro(at(p,p.loc(pos)), msg).debug(32)
}

func (p *parser) expect(tok token) Pos {
	var pos = p.pos
	if p.tok != tok && !(tok == LINEND && p.lineComment != nil) {
		p.expected(pos, "'"+tok.String()+"'")
	}
	p.step() // move forward
	return pos
}

func (p *parser) linend() (ok bool) {
	if p.lineComment != nil {
		p.lineComment, ok = nil, true
	} else if p.tok == EOF {
		ok = true
	} else if p.tok == LINEND {
		p.step(); ok = true
	} else {
		p.expected(p.pos, "'\\n'")
	}
	return
}

func (p *parser) isRecipeStart() (res bool) {
	if p.tok == RECIPE {
		res = true
	} else if p.tok == SPACE && p.lit == "\t" {
		p.tok, res = RECIPE, true // Fixes recipe \t
	}
	return
}

// ----------------------------------------------------------------------------
// Parsing

// safePos returns a valid file position for a given position: If pos
// is valid to begin with, safePos returns pos. If pos is out-of-range,
// safePos returns the EOF position.
//
// This is hack to work around "artificial" end positions in the AST which
// are computed by adding 1 to (presumably valid) token positions. If the
// token positions are invalid due to parse errors, the resulting end position
// may be past the file's EOF position, which would lead to panics if used
// later on.
//
/*
func (p *parser) _safePos(pos Pos) (res Pos) {
	defer func() {
		if recover() != nil {
			res = Pos(p.scanner.file.Base() + p.scanner.file.Size()) // EOF position
		}
	}()
	_ = p.scanner.file.Offset(pos) // trigger a panic if position is out-of-range
	return pos
}
*/

// ----------------------------------------------------------------------------
// Barewords & Identifiers

func (p *parser) bare(ctx Context, lhs bool) (x Value) {
	if false { defer trace(ctx) }

	defer p.setbits(p.setbit(parseBARE))

	var tok, lit, pos = p.tok, p.lit, ctx.Position()
	p.step()

	if tok != BAREWORD && lit == "" {
		lit = tok.String()
	}
	return makeBareword(pos, lit)
}

func (p *parser) braced(ctx Context, lhs bool) (x Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "braced")) }

	var typed token
	var pos = p.Position()

	defer trace(at(ctx, pos))
	defer p.setbits(p.setbit(parseBraced))

	p.expect(LBRACE)

	if p.tok == RBRACE {
		x = &null{p.valbase()}
		p.spaces()
		p.step() // consumes }
		return
	}

	if checkpoints {
		if p.tok != LPAREN && !p.scanner.bits.isBrace() {
			erro(p.ctx(ctx), "wrong scan state: %v, %v, %v", p.tok, p.lit, p.scanner.scanstate).debug(3)
		}
	}

	if p.tok == LBRACK { // OBSOLETE: {[...]}
		erro(p.ctx(ctx), "syntax error; for modification, use {(modifier)}").debug(3)
		return
	} else if p.tok == LPAREN {
		x = p.modification(ctx)
		p.spaces()
		p.expect(RBRACE)
		return
	} else if p.tok == ASSIGN { // =
		p.step() // skips =
		if p.tok == RBRACE {
			typed = NULL
		} else {
			switch typed = p.tok ; typed {
			case /* TODO: GLOB, */ REGEX:
				p.step() // skips the type name
				p.scanner.addBits(isBraceRaw)

				// Trim leading spaces differently to avoid messing the scan states.
				// NOTE: the first SPACE and BAREWORD do not become RAW.
				for p.tok == SPACE || (p.tok == RAW && p.lit == " ") { p.step() }
				if false { switch p.tok { case BAREWORD: p.tok = RAW }}
				if false { note(p, "%v %v %v", p.tok, p.lit, p.scanner.scanstate) }

			default:
				p.next(true)
			}
		}
		if p.tok == RBRACE {
			switch p.step(); typed {
			case BIN:   x = makeBinary(pos, 0)
			case OCT:   x = makeOctal(pos, 0)
			case INT:   x = makeDecimal(pos, 0)
			case HEX:   x = makeHexadecimal(pos, 0)
			case FLOAT: x = makeFloat(pos, 0.)
			case TRUE:  x = makeBoolean(pos, true)
			case FALSE: x = makeBoolean(pos, false)
			case YES:   x = makeAnswer(pos, true)
			case NO:    x = makeAnswer(pos, false)
			case ON:    x = makeOption(pos, true)
			case OFF:   x = makeOption(pos, false)
			case NONE:  x = makeNone(pos)
			case NULL:  x = makeNull(pos)
			default: erro(ctx, "expects braced value (%v)", typed).debug(1)
			}
			return
		}
	}

	switch typed {
	case BARE: // {=bare ... }
		x = p.bare(p.ctx(ctx), false)
		p.spaces()
		p.expect(RBRACE)
		return
	case GLOB: // {=glob ... }
		x = p.glob(p.ctx(ctx), nil)
		p.spaces()
		p.expect(RBRACE)
		return
	case REGEX: // {=regex ...}
		return p.regex(p.ctx(ctx))
	case FILE: // {=file ... }
		if v := p.expr(ctx); v != nil {
			var c = at(ctx, v)
			var s = v.string(c)
			var a = []interface{}{stat_nonexist{true}}
			if !isAbsOrRel(s) { a = append(a, stat_dir{ctx.project().absPath}) }
			x = stat(c, s, a...)
		}
		p.spaces()
		p.expect(RBRACE)
		return
	case PATH: // {=path ... }
		if v := p.expr(ctx); v != nil {
			if t, y := v.(*path); !y {
				x = p.path(ctx, lhs, v)
			} else {
				x = t
			}
		}
		p.spaces()
		p.expect(RBRACE)
		return
	case BIN, OCT, INT, HEX, FLOAT: // ={bin ...}, {=oct ...}, {=int ...}, {=hex ...}, {=float ...}
		if v := p.expr(ctx); v == nil {
			erro(ctx, "%s expects: %v, not %v %v", typed, RBRACE, p.tok, p.lit).debug(1)
		} else if p.spaces(); p.tok == RBRACE {
			if p.step(); typed == FLOAT {
				var n, _ = v.float(ctx)
				return makeFloat(pos, n)
			}
			switch n, _ := v.int(ctx); typed {
			case   BIN: return makeBinary(pos, n)
			case   OCT: return makeOctal(pos, n)
			case   INT: return makeDecimal(pos, n)
			case   HEX: return makeHexadecimal(pos, n)
			case FLOAT:
			}
		}
		return
	case ANSWER: // {=answer ...}
		var v bool
		switch p.tok {
		case  TRUE, YES: v = true  ; p.next(true)
		case FALSE,  NO: v = false ; p.next(true)
		default:
			if t := p.expr(ctx); t != nil {
				v = t.true(ctx)
			} else {
				erro(p.ctx(ctx), "invalid expression").debug(1)
			}
		}
		p.spaces()
		p.expect(RBRACE)
		return &answer{boolean{valbase{pos},v}}
	case BOOL, BOOLEAN: // {=bool ...}, {=boolean ...}
		var v bool
		switch p.tok {
		case  TRUE, YES,  ON: v = true  ; p.next(true)
		case FALSE,  NO, OFF: v = false ; p.next(true)
		default:
			if t := p.expr(ctx); t != nil {
				v = t.true(ctx)
			} else {
				erro(p.ctx(ctx), "invalid expression").debug(1)
			}
		}
		p.spaces()
		p.expect(RBRACE)
		return &boolean{valbase{pos},v}
	case TRUE, FALSE: // {=true ...}, {=false ...}
		var v = p.expr(ctx).true(ctx)
		p.spaces()
		p.expect(RBRACE)
		return &boolean{valbase{pos},(typed == TRUE && v)}
	case YES, NO: // {=yes ...}, {=no ...}
		var v = p.expr(ctx).true(ctx)
		p.spaces()
		p.expect(RBRACE)
		return &answer{boolean{valbase{pos},(typed == YES && v)}}
	case ON, OFF: // {=on ...}, {=off ...}
		var v = p.expr(ctx).true(ctx)
		p.spaces()
		p.expect(RBRACE)
		return &option{boolean{valbase{pos},(typed == ON && v)}}
	case RAW:
		s := p.expr(ctx).string(ctx)
		p.spaces()
		p.expect(RBRACE)
		return &raw{valbase{pos},s}
	case UNDEF: // {=undef ...}
		x = undef{p.expr(ctx)}
		p.spaces()
		p.expect(RBRACE)
		return
	case NONE: // {=none ...}
		var v Value
		for ; p.tok != RBRACE && p.tok != EOF; p.spaces() {
			if t := p.expr(ctx); v == nil {
				v = t
			} else if l, y := v.(*list); y {
				l.elems = append(l.elems, t)
			} else {
				v = &list{elements{[]Value{v,t}}}
			}
		}
		p.expect(RBRACE)
		return &none{valbase{pos},v}
	case /* DISJUNCTION, */ 0: // {...}
		if v := p.values(ctx); len(v) == 0 {
			x = makeNull(pos)
		} else if len(v) == 1 {
			x = disjunction{v[0]}
		} else {
			x = disjunction{makeList(v...)}
		}
		p.spaces()
		p.expect(RBRACE)
		return
	default:
		erro(ctx, "%v", typed).debug(1)
		return
	}
}

func (p *parser) selector(ctx Context) (res Value) {
	defer p.setbits(p.setbit(parseSELECT_PROP))
	res = p.expr(ctx)
	return
}

func (p *parser) selectExpr(ctx Context, lhs Value) (res Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Select")) }

	var (
		tok = p.tok // the arrow '->' or '=>'
		l = _loader(ctx)
		proj = l.project()
	)
	ctx = p.ctx(ctx)
	p.step() // skip '->' or '=>'

	switch t := lhs.(type) {
	case *selection:
		if v := t.expand(at(ctx, t.Position())); isNull(v) {
			erro(ctx, "nil selection: %v", lhs).debug(1)
			return
		} else {
			lhs = v
		}
	case *bareword:
        switch t.s {
        case "use", "usee", "goals", "os", "mode":
			erro(ctx, "$:%s: is obsoleted, use $(.$s) instead", t.s, t.s).debug(1)
        default:
            if name, o := l.resolve(ctx, lhs); false {
				erro(at(ctx,lhs.Position()), "resolve '%v' failed", lhs)
				erro(ctx, "parser is here (tok=%s)", tok)
				erro(at(ctx,p.Position()), "parser to go here (tok=%s, lit=%s)", p.tok, p.lit).debug(8)
                return
            } else if !isNull(o) {
				lhs = o
			} else if tok == SELECT_PROG2 {
				res = makeNull(ctx.Position()) // ignore
				return
			} else {
				erro(at(ctx,lhs.Position()), "%v: '%v' is undefined (name=%v, obj=%v)", proj, lhs, name, o)
				erro(ctx, "%v: parser is here (name=%s, tok=%s)", proj, t.s, tok)
				erro(at(ctx,p.Position()), "%v: parser to go here (tok=%s, lit=%s)", proj, p.tok, p.lit).debug(16)
				return
            }
        }
    case *barecomp: // for cases like '.foo'
        if name, o := l.resolve(ctx, t); false {
			erro(at(ctx,lhs), "resolve selection object '%v' (%s) error", lhs, name).debug(1)
			return
        } else if !isNull(o) {
			lhs = o
		} else if tok == SELECT_PROG2 {
			res = makeNull(ctx.Position()) // ignore
			return
		} else {
			erro(at(ctx,lhs), "'%v' is undefined", lhs).debug(1)
			return
        }
	case *globpat:
		if o, y := optionalize(ctx, lhs); y { lhs = o } else {
			erro(at(ctx,lhs), "selection of '%v' is undefined", lhs).debug(1)
		}
	}

	if rhs := p.selector(ctx); isNull(rhs) {
		res = makeNull(ctx.Position())
	} else {
		if v, y := optionalize(ctx, rhs); y { rhs = v } // foo→bar?
		res = makeSelection(ctx.Position(), tok, lhs, rhs)
	}

	if (p.tok == SELECT_PROP || p.tok == SELECT_PROG1 || p.tok == SELECT_PROG2) {
		res = p.selectExpr(ctx, res) // Continue the selection recursivly.
	}
	return
}

// ----------------------------------------------------------------------------
// Common productions

func (p *parser) isEndOfLine() bool {
	// If there's a comment right after the parsed expression, we break
	// the expression list to treat the end-of-line comment like a LINEND.
	return p.lineComment != nil || p.tok == LINEND || p.tok == EOF
}

func (p *parser) isEndOfList(lhs bool) bool {
	// If there's a comment right after the parsed expression, we break
	// the expression list to treat the end-of-line comment like a LINEND.
	if p.lineComment != nil || p.tok.isListDelim() || (lhs && p.tok.isAssign()) {
		return true
	}
	if (p.bits&parseRecipe != 0) && p.tok == RECIPE { // TODO: using p.isRecipeStart()
		return true
	}
	return false
}

func (p *parser) isEndOfURL(lhs bool) bool {
	return p.tok == SPACE || p.isEndOfLine() || p.isEndOfList(lhs)
}

func (p *parser) isEndOfDotConcat(lhs bool) bool {
	// Expressions like `FOO.BAR(xxx)` does not count.
	switch p.tok {
	case SPACE, LPAREN, COLON, PCON, ASSIGN: fallthrough
	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2: return true
	}
	return p.isEndOfLine() || p.isEndOfList(lhs)
}

func (p *parser) ruleParams(ctx Context, args []Value) (err error) {
	var scope = ctx.scope()
	for _, elem := range args {
		switch ctx := at(ctx, elem.Position()); elem.(type) {
		case *bareword, *barecomp:
			p.params = append(p.params, scope.auto(ctx, elem.string(ctx), strconv.Itoa(len(p.params)+1)))
		default: //case *ast.GroupExpr, *ast.ListExpr, *ast.BasicLit:
			erro(at(ctx,elem), "bad parameter form (%v)", us(elem))
		}
	}
	return
}

func (p *parser) depends(ctx Context, normal bool) (list []Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "depends")) }

	for p.tok != SEMICOLON && p.tok != BAR && !p.isEndOfLine() {
		if p.tok == COLON { // FIXME: this check is not working!
			// FIXME: detects unexpected colon ':'
			erro(p, "unexpected colon").debug(1)
			p.next(true) // just ignore this colon
		} else if p.spaces(); !p.isEndOfLine() {
			if len(list) == 0 {
				p.bits |= parseDepend0
			} else {
				p.bits &= ^parseDepend0
			}

			var val = p.expr(ctx)
			if flush(ctx) > 0 {
				erro(ctx, "depend: %T %v", val, val).debug(1)
				return
			}

			if normal {
				if g, y := val.(*group); y && len(g.elems) == 1 {
					if g, y = g.elems[0].(*group); y {
						p.ruleParams(ctx, g.elems)
						continue
					}
				}
			}

			list = append(list, val)
			if p.tok == SPACE { p.next(true) } //p.spaces()
		}
	}
	return
}

// If lhs is set, result list elements which are identifiers are not resolved.
func (p *parser) values(ctx Context, ii ...interface{}) (values []Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Values")) }
	if true { defer trace(ctx) }

	var lhs bool

	for _, i := range ii {
		switch v := i.(type) {
		case parse_left_hand_side: lhs = v.bool
		case Value: values = append(values, v)
		default: erro(ctx, "unsupported value: %v{%v}", typeof(i), i).debug(5)
		}
	}

	for p.spaces(); !p.isEndOfList(lhs); p.spaces() {
		var prev = p.pos
		if values = append(values, p.expr(ctx, lhs)); p.pos == prev {
			erro(p, "bad: %v %v; %v", p.tok, p.lit, values).debug(1)
			break
		}

		// If there's a comment right after the parsed expression, we break
		// the expression list to treat the end-of-line comment like a LINEND.
		if p.tok == EOF || p.tok == LINEND || p.lineComment != nil { break }
	}
	return
}

func (p *parser) list(ctx Context, ii ...interface{}) *list {
	return makeList(p.values(ctx, ii...)...)
}

func (p *parser) group(ctx Context, lhs bool) *group {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Group")) }

	defer p.setbits(p.setbit(parseGroup))
	p.clearbit(parseCall) // for commas

	ctx = p.ctx(ctx)

	p.expect(LPAREN)
	p.spaces()

	var elems, converted = p.values(ctx), false
	for p.tok != RPAREN && p.tok != EOF {
		// if p.tok == COMMA { warn(ctx, "%020b: %v %v", p.bits, p.tok, p.lit).debug(1) }
		// if p.tok == COMMA { p.next(true) }
		switch p.tok {
		case BAR, COMMA, SEMICOLON:
			elems = append(elems, p.punctuation())
			p.spaces()
		}
		var next *list
		next = p.list(ctx)
		if !converted {
			elems = []Value{ makeList(elems...), next }
			converted = true
		} else {
			elems = append(elems, next)
		}
	}
	p.expect(RPAREN)
	return makeGroup(ctx.Position(), elems...)
}

func (p *parser) argumentedExpr(ctx Context, x Value) *argumented {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "argumented")) }

	defer p.setbits(p.setbit(parseGroup))
	p.clearbit(parseCall)

	ctx = p.ctx(ctx)
	p.next(true) // skip LPAREN

	var a = []Value{ p.list(ctx) }
	for p.tok != RPAREN && p.tok != LINEND && p.tok != EOF {
		switch p.tok {
		case COMMA: p.next(true) // skip COMMA
		case BAR, SEMICOLON:
			if false {
				a = append(a, p.punctuation())
				p.spaces()
			} else {
				erro(ctx, "unexpected punctuation: %v", p.tok).debug(1)
			}
		}
		a = append(a, p.list(p.ctx(ctx)))
	}
	p.expect(RPAREN)
	return makeArgumented(x, a...)
}

func (p *parser) globmeta(ctx Context) (x *globmeta) {
	pos, tok := p.Position(), p.tok
	p.step()
	return makeGlobMeta(pos, tok)
}

func (p *parser) globrange(ctx Context) (x *globrange) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Glob")) }

	ctx = p.ctx(ctx)
	p.expect(LBRACK) // skip '['

	chars := p.expr(ctx)
	p.expect(RBRACK) // skip ']'

	return makeGlobRange(ctx.Position(), chars)
}

func (p *parser) glob(ctx Context, x Value) Value {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Glob")) }

	ctx = p.ctx(ctx)

	defer p.setbits(p.setbit(parseGLOB))

	var g *globpat
	if y := x == nil; y {
		g = &globpat{}
	} else if g, y = x.(*globpat); !y || g == nil {
		g = makeGlobPat(ctx, x)
	}

outer:
	for p.tok != RBRACE && p.tok != EOF && p.lineComment == nil {
		var v Value
		switch p.tok {
		case SELECT_PROP, SELECT_PROG1, SELECT_PROG2, PCON, RPAREN, COMMA, SPACE, LINEND, EOF:
			break outer
		case STAR, DAST, QUE:
			v = p.globmeta(ctx) // * ** ?
		case LBRACK:
			v = p.globrange(ctx) // [abc0-9xyz]
		default:
			v = p.expr(ctx)
		}
		g.elems = append(g.elems, v)
	}

	return g
}

func (p *parser) perc(ctx Context, lhs bool, x Value) Value {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Perc")) }

	// avoid nesting percent expressions
	defer p.setbits(p.setbit(parsePERC))

	var (
		pos = p.pos
		y Value
	)
	if p.step(); pos+1 == p.pos { // joint, e.g. '%.o', but skip '% .o'
		switch p.tok {
		case COLON, DOLON,
			LPAREN, RPAREN,
			LBRACK, RBRACK,
			PCON,   SEMICOLON,
			COMMA,  SPACE,
			LINEND:
		case PERC: // %%
			p.step() // consume the second %
			position := p.Position()
			perc2 := makePercpat(position, nil, nil)
			if pos+2 == p.pos {
				switch p.tok {
				case PERC: // %%%
					erro(p, "too many %")
				case PCON: // FIXES: %%/xxx -> Path(%% xxx)
					x = makePercpat(position, x, perc2)
					return p.path(ctx, lhs, x)
				case COLON,    DOLON,
					LPAREN,    RPAREN,
					LBRACK,    RBRACK,
					LBRACE,
					SEMICOLON, COMMA,
					SPACE,     LINEND:
				default:
					var (
						yy = p.expr(ctx)
						_, ok = yy.(*path)
					)
					if ok { erro(p, "incorrect: %v, %v", x, yy) }
					assert(!ok, "the second part of aaa%%bbb/foo/bar parsed incorrectly as path")
					perc2.Suffix = yy
				}
			}
			y = perc2
		default:
			y = p.expr(ctx)
		}
	}
	return makePercpat(p.loc(pos), x, y)
}

func (p *parser) regex(ctx Context) (_ Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "regex")) }

	var rx string
	var pos = p.Position()

	defer trace(at(ctx, pos))
	defer p.setbits(p.setbit(parseREXP)) // avoid nesting percent expressions

	if checkpoints {
		if !p.scanner.bits.isBrace() {
			erro(ctx, "wrong scan state: %v", p.scanner.scanstate).debug(3)
		}
		if !p.scanner.bits.isBraceRaw() {
			erro(ctx, "wrong scan state: %v", p.scanner.scanstate).debug(3)
		}
	}

	for ; p.tok != RBRACE && p.tok != EOF; p.scan() {
		if p.tok == ESCAPE { rx += "\\" }
		rx += p.lit
	}

	p.expect(RBRACE)

	var err error
	var x = &regexpat{valbase{pos}, nil} // TODO: correct regexp pattern value
	if x.Regexp, err = regexp.Compile(rx); err != nil {
		note(ctx, "regex: %v", rx)
		erro(at(p,pos), "regex: %v", err).debug(6)
	}
	return x
}

func (p *parser) pair(ctx Context, x Value) *pair {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "pair")) }

	ctx = p.ctx(ctx)
	p.step()

	var y Value
	if p.isEndOfList(false) {
		y = makeNull(ctx.Position())
	} else {
		y = p.expr(ctx)
	}

	return makePair(x, y)
}

func (p *parser) flag(ctx Context, lhs bool) flag {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "flag")) }

	ctx = p.ctx(ctx)
	p.step() // skip dash '-'

	var x Value
	// flag expressions, excluding "-)" "-]" "-}" "-\n", "-=", "-:", etc.
	if p.isEndOfLine() || p.isEndOfList(false) || p.tok == SPACE || p.tok == RECIPE {
		x = makeNull(ctx.Position())
	} else if false {
		x = p.expr(ctx)
	} else {
		x = p.unary(ctx, false)
		l: for p.tok == DOT || !(_operator_beg < p.tok && p.tok < _closure_beg) {
			switch p.tok {
			case COMMENT, HASH, SPACE, RECIPE, LINEND, EOF: break l
			case DELEGATE, CLOSURE: x = compose(ctx, x, p.unary(ctx, false))
			default: if p.tok.isClosure() || p.tok.isDelegate() {
				x = compose(ctx, x, p.unary(ctx, false))
			} else {
				break l
			}}
		}
	}
	if x == nil { erro(ctx, "nil flag name").debug(1) }
	return flag{x}
}

func (p *parser) negative(ctx Context, lhs bool) negative {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "negative")) }
	p.expect(EXC)
	return negative{p.expr(ctx, lhs)}
}

func (p *parser) punctuation() *punctuation {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "punctuation")) }
	var vb, tok = p.valbase(), p.tok
	p.step()
	return &punctuation{vb, tok}
}

func (p *parser) escape(ctx Context, lhs bool) (v Value) {
	var vb, lit = p.valbase(), p.lit
	p.expect(ESCAPE)
	return &escaped{vb, lit}
}

func (p *parser) literal(ctx Context, lhs bool) (v Value) {
	var tok, lit = p.tok, p.lit
	ctx = p.ctx(ctx)
	p.step()

	// ESCAPE is handled in value.EscapeChar
    switch position := ctx.Position(); tok {
    case BAR: erro(ctx, "`|` is deprecated, changed the modifiers!")
    case BINARY:      v = ParseBinary(position, lit)
    case OCTAL:       v = ParseOctal(position, lit)
    case INTEGER:     v = ParseDecimal(position, lit)
    case HEXADECIMAL: v = ParseHexadecimal(position, lit)
    case FLOATING:    v = ParseFloat(position, lit)
    case DATETIME:    v = ParseDateTime(position, lit)
    case DATE:        v = ParseDate(position, lit)
    case TIME:        v = ParseTime(position, lit)
    case URI:         v = ParseURL(position, lit)
    case BAREWORD:    v = makeBareword(position, lit)
    case STRING:      v = makeStrlit(position, lit)
    case RAW:         v = makeRaw(position, lit)
    default: unreachable()
    }
	return
}

func (p *parser) compound(ctx Context, lhs bool) *compound {
	var elems []Value

	p.step()

	defer p.setbits(p.setbit(parseCompound))

	for p.tok != EOF && p.tok != COMPOSED && p.tok != LINEND {
		if p.tok == RAW {
			elems = append(elems, p.literal(ctx, false))
		} else {
			elems = append(elems, p.expr(ctx))
		}
	}
	p.expect(COMPOSED)
	return makeCompound(elems...)
}

// Parses dot composing expressions (TODO: check against file extensions).
//   .foo
//   .'foo'
//   ."foo"
//   .(foo)
//   ..foo
//   ..'foo'
//   .foo.bar
func (p *parser) dot(ctx Context, lhs bool, x Value) (res Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Dot")) }

	defer trace(ctx)
	defer p.setbits(p.setbit(parseDOT))

	ctx = p.ctx(ctx)

	for !p.isEndOfDotConcat(lhs) {
		x = compose(ctx, x, p.composite(ctx, false))
		if p.tok == DOT /*&& comp.End() == p.pos*/ {
			x = compose(ctx, x, &punctuation{p.valbase(), p.tok})
			p.step() // skips '.'
		}
	}

	return x
}

func (p *parser) path(ctx Context, lhs bool, start Value) (res *path) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Path")) }

	defer p.setbits(p.setbit(parsePATH))

	if start == nil {
		erro(ctx, "nil path starter").debug(1)
		return
	}

	switch ctx = at(ctx, start); t := start.(type) {
	case     *path: res = t
	case   *strlit: res = makePath(splitPathStr(ctx, t.s)...)
	case *compound: res = makePath(splitPathStr(ctx, t.string(ctx))...) // FIXME: dont final here
	default:        res = makePath(start)
	}

	for p.tok == PCON && p.tok != EOF {
		// skips repeated '/' sequence
		for p.step(); p.tok == PCON; p.step() {}

		switch p.tok {
		case LPAREN, LBRACE, RPAREN, RBRACE, COMMA, SPACE, LINEND:
			res.elems = append(res.elems, _pathpun(p.ctx(ctx), PTAIL)) // after the last '/'
			return
		}

		var t = p.composite(ctx, false)
		if x, y := t.(*path); y {
			res.elems = append(res.elems, x.elems...)
		} else {
			res.elems = append(res.elems, t)
		}

		if p.tok == SPACE || p.isEndOfLine() { return }
	}
	return
}

func isKnownURLScheme(s string) (result bool) {
	switch strings.ToLower(s) {
	case "file", "http", "https", "ftp", "ftps", "mailto":
		result = true
	}
	return
}

func (p *parser) url(ctx Context, lhs bool, scheme Value) (res Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "URL")) }

	defer p.setbits(p.setbit(parseURL))

	var (
		url = &URL{ Scheme:scheme }
		colon1 = p.expect(COLON) // consumes ':'
		colon2 = NoPos
		//colon3 = NoPos
		a = NoPos // @
	)

	if p.tok == PCON {
		p.step() // the first '/'
		if p.tok == PCON {
			p.expect(PCON) // the second '/'
		} else {
			erro(ctx, "TODO: URL path: %v (%T) (next: %s (%s))", scheme, scheme,  p.tok, p.lit).debug(1)
			res = makeNull(p.Position())
			return
		}
	} else if !p.isEndOfURL(lhs) {
		erro(at(ctx, p.loc(colon1)), "TODO: URL: %v (%T) (next: %s (%s))", scheme, scheme, p.tok, p.lit).debug(1)
		res = makeNull(p.Position())
		return
	}

	if !p.isEndOfURL(lhs) {
		userOrHost := p.composite(ctx, false)
		if p.tok == COLON {
			url.Username, colon2 = userOrHost, p.pos
			p.step() // ':'
			if p.tok != AT && p.tok != PCON && !p.isEndOfURL(lhs) {
				url.Password = p.composite(ctx, false)
			}
		} else {
			url.Host = userOrHost
		}
		if p.tok == AT {
			p.step() // '@'
		}
	}
	if url.Host == nil && colon2 == NoPos && a == NoPos && !p.isEndOfURL(lhs) {
		url.Host = p.composite(ctx, false)
		if p.tok == COLON {
			//colon3 = p.pos
			p.step() // ':'
			if p.tok != SPACE && p.tok != LINEND {
				url.Port = p.composite(ctx, false)
			}
		}
	}
	if p.tok == PCON {
		url.Path = p.path(ctx, lhs, _pathpun(ctx, p.tok))
	}
	// scanning '#' as HASH instead of COMMENT
	defer p.scanner.setBits(p.scanner.commentsOff())
	if p.tok == QUE {
		p.step() // '?'
		if p.tok != HASH && !p.isEndOfURL(lhs) {
			url.Query = p.composite(ctx, false)
		}
	}
	if p.tok == HASH {
		p.step() // '#'
		if !p.isEndOfURL(lhs) {
			url.Fragment = p.composite(ctx, false)
		}
	}
	return url
}

func (p *parser) closuredelegate(ctx Context, isClosure bool) (result Value) {
	if l_traverse.enabled {	defer un(l_trace(l_traverse, "closuredelegate")) }

	ctx = at(ctx, p.Position())

	defer trace(ctx)

	var (
		l = _loader(ctx)
		proj = l.project()
		resolved Value // Object or *selection
		rest []Value
	)

	resolveConfig := func(val Value, name string) (obj Object) {
		if c := proj.configure; c != nil { obj = c.resolve(ctx, name) }
		return
	}

	resolve := func(lPos Position, lTok token, name Value) (str string, obj Value, okay bool) {
		defer trace(ctx) // backtrace on errors

		if x, y := name.(*argumented); y { name = x.Value }
		if _, y := name.(condval); !y {
			if v := name.expand(ctx); v == nil {
				erro(at(ctx,name), "%v is nil", us(name)).debug(16)
				return
			} else { name = v }
		}

		switch lTok {
		case LPAREN:
			if name.expandable(ctx) {
				return str, name, true
			} else if str, resolved = l.resolve(ctx, name); false {
				erro(at(ctx,name), "resolve '%v' (%s) failed", name, str).debug(1)
				return
			} else if str == "" {
				// name.expandable(ctx) covers not these cases:
				switch name.(type) {
				case condval, *closure, *delegate, *selection:
					return str, name, true
				default:
					erro(at(ctx,name), "%v is empty for name", us(name)).debug(1)
					return
				}
			} else if resolved == nil {
				if p.bits&parseIncludingConf != 0 {
					// Create an empty def if referred in configuration.sm.
					def, _ := l.def(name.Position(), str)
					def.origin = defConfRef
					obj, okay = def, true
					return
				}

				if p.bits&parseAutoName != 0 {
					if d := autoDef(ctx, str); d == nil {
						obj = ctx.scope().auto(ctx, str)
					} else {
						obj = d
					}
					okay = obj != nil
					return
				}

				if o := resolveConfig(name, str); o != nil {
					obj, okay = o, true
					return
				}

				if isClosure || p.bits&parseUndefValue != 0 || exable(ctx, name, nil) {
					obj, okay = name, true // recursive delegation or closure
					return
				}

				if false { note(ctx, "auto(%v) → %v", us(name), autoDef(ctx, str)) }
				note(at(ctx,name), "resolve(%v) ⇒ %v", us(name), us(obj))
				erro(ctx, "%v", us(ctx)).debug(20)
				return
			} else {
				obj, okay = resolved.(Object)
				return
			}
		case LBRACE:
			if name.expandable(ctx) {
				erro(at(ctx,name), "%v: name %v is closured", proj, us(name)).debug(1)
				return
			} else if resolved = l.proj.resolveEntries(ctx, name, false); resolved == nil {
				if name.expandable(ctx) {//, plain
					erro(at(ctx,name), "resolved '%v' (aka. %s) is nil (project=%v)", name, name.string(ctx), proj).debug(3)
				} else {
					erro(at(ctx,name), "resolved %v is nil (project=%v)", us(name), proj).debug(3)
				}
			} else if obj, okay = resolved.(Object); !okay {
				erro(at(ctx,lPos), "resolved '%v' of '%T' is not Object", name, resolved).debug(1)
			} else if exe, _ := obj.(executer); exe == nil {
				erro(at(ctx,lPos), "resolved '%v' of '%T' is not executer", name, resolved).debug(1)
			}
		}
		return
	}

	defer p.setbits(p.setbit(parseCall))

	var (
		name Value
		nameStr string
		tokLp token
		opts []Value
		obj Value
		okay bool
	)
	switch p.step(); p.tok {
	case LPAREN, LBRACE: // $(...), ${...}
		var posLp = p.Position()
		tokLp = p.tok ; p.step() // skips LPAREN, LBRACE

		var posName = p.Position()
		switch p.tok {
		case SPACE:
			erro(at(ctx,posName), "unexpected spaces").debug(1)
			return makeNull(posName)
		case COLON:
			p.step();  posName = p.Position()
			warn(at(ctx,posName), "colon").debug(1)
		}

		if name = p.expr(ctx); name == nil {
			erro(at(ctx,posName), "%v: parsed name is nil", proj).debug(1)
			return makeNull(posName)
		}

		if v, y := optionalize(ctx, name); y { name = v } // foo?  foo(a,b,c)?
		if a, y := name.(*argumented); y {
			var args = merge(a.args...)
			for _, v := range args {
				if p, y := v.(*pair);  y { v = p.key }
				if _, y := v.(flag); !y {
					erro(at(ctx,v), "%v: not a Flag: %T %v", proj, v, v).debug(1)
				}
			}
			if true { name, opts = a.Value, args }
			if v, y := optionalize(ctx, name); y { name = v } // foo?(a,b,c)
		}

		if name == nil {
			erro(at(ctx,posName), "%v: name %v is nil", proj).debug(1)
		} else if name.expandable(ctx) {
			obj = name // unresolved
		} else if nameStr, obj, okay = resolve(posLp, tokLp, name); !okay {
			erro(at(ctx,posName), "%v: name %v is unidentified", proj, name).debug(1)
		}

		if (tokLp == LPAREN && p.tok != RPAREN) || (tokLp == LBRACE && p.tok != RBRACE) {
			var ctx = at(ctx, posName)
			var savedBits  = p.bits
			if nameStr == "auto" {
				if tokLp != LPAREN {
					erro(at(ctx,posLp), "%v: auto: incorrect left paren", proj).debug(1)
				} else {
					p.spaces() // skip the imediate spaces
				}

				var defs = make(autodefs)
				var ac = automatic{ Context:ctx, defs:defs,
					suppress:func(s string) (y bool) { _, y = defs[s] ; return }}

				ctx = &ac

				var al = p.list(ctx)
				if rest = append(rest, al); p.tok == COMMA { p.next(true) }
				for _, val := range merge(al) {
					var s string
					if kv, y := val.(*pair); y {
						s, val = kv.key.string(ctx), kv.val
					} else {
						s = val.string(ctx)
						val = nil
					}
					if c := at(ctx, val.Position()); s != "" {
						ac.set(c, s, val)
					} else {
						erro(c, "%v: auto: %v is empty", proj, val).debug(1)
					}
				}

				if !isClosure { p.bits |= parseAutoName }
			}

			if nameStr == "case" {
				rest = append(rest, p.list(ctx))
				for p.bits |= parseUndefValue; p.tok == COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx))
				}
			} else if nameStr == "and" {
				p.bits |= parseUndefValue
				for rest = append(rest, p.list(ctx)); p.tok == COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx))
				}
			} else if nameStr == "or" {
				p.bits |= parseUndefValue
				for rest = append(rest, p.list(ctx)); p.tok == COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx))
				}
			} else if nameStr == "foreach" {
				rest = append(rest, p.list(ctx))
				for p.bits |= parseForeachTempl; p.tok == COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx))
				}
			} else {
				if n, e := strconv.Atoi(nameStr); e == nil && n < 0 && n > maxDigitAutoNum {
					erro(at(ctx, name), "num auto too big: %v (max %v)", n, maxDigitAutoNum).debug(1)
				}
				for rest = append(rest, p.list(ctx)); p.tok == COMMA; {
					p.next(true) // consumes COMMA
					rest = append(rest, p.list(ctx))
				}
			}

			p.bits  = savedBits
		}

		switch tokLp {
		case LPAREN: p.expect(RPAREN)
		case LBRACE: p.expect(RBRACE)
		}

	default:
		if position := p.Position(); !isClosure { // $(...), disabled $name.
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v` or `%v` or quotes", LPAREN, LBRACE).debug(1)
			return makeNull(position)
		} else if p.tok == STRING || p.tok == COMPOUND {
			var posLp = p.Position()
			tokLp = p.tok

			// &'xxxx' or &"xxxx"
			if name = p.expr(ctx); name == nil {
				erro(at(ctx,posLp), "parsed name is nil").debug(1)
			} else if name.expandable(ctx) {//, /* expandClosure */final
				erro(at(ctx,name.Position()), "name '%v' (%T) is closured (project=%v)", name, name, proj).debug(1)
			} else if nameStr, obj, okay = resolve(posLp, tokLp, name); !okay {
				erro(at(ctx,name.Position()), "name '%v' is unidentified", name).debug(1)
			}
		} else {
			// &(...), &{...}, &'...', &"..."
			erro(ctx, "expects `%v`, `%v` or quotes, not %v %v", LPAREN, LBRACE, p.tok, p.lit).debug(1)
			return makeNull(position)
		}
	}

	if obj == nil && proj.plugin != nil && proj.pluginScope != nil {
		if nameStr == "" && !isNull(name) { nameStr = name.string(ctx) }
		if nameStr != "" { obj = proj.pluginScope.Lookup(nameStr) }
	}

	if obj == nil {
		erro(at(ctx,name.Position()), "resolved '%v' is nil: %v", name, us(resolved)).debug(1)
		return
	} else if pos := ctx.Position(); isClosure {
		return makeClosure(pos, tokLp, obj, opts, rest...)
	} else if d, y := obj.(*def); y && d.origin == defCodeBlockAuto {
		return d.value
	} else {
		result = makeDelegate(pos, tokLp, obj, opts, rest...)
		return
	}
}

func (p *parser) special(ctx Context, isClosure, lhs bool) (result Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "special")) }

	defer trace(ctx)

	var y bool
	var obj Object
	var pos, tok, s = p.pos, p.tok, p.lit
	var position = p.loc(pos)

	p.step()

	if _, t := _loader(ctx).resolve(ctx, makeBareword(position, s)); t == nil {
		if obj = ctx.scope().auto(ctx, s); obj == nil {
			erro(ctx, "%v is undefined, %v, %v", s, autoDef(ctx, s), resolve(ctx, s))
			erro(ctx, "%v", us(ctx)).debug(10)
			return makeNull(position)
		}
	} else if obj, y = t.(Object); !y {
		erro(at(ctx,t), "'%v' is not object: %v", s, us(t)).debug(6)
		return makeNull(position)
	}

	if isClosure {
		return makeClosure(position, tok, obj, nil)
	} else if d, y := obj.(*def); y && d.origin == defCodeBlockAuto {
		return d.value
	} else {
		result = makeDelegate(position, tok, obj, nil)
		return
	}
}

func (p *parser) unary(ctx Context, lhs bool) (x Value) {
	if l_traverse.enabled && false { defer un(l_trace(l_traverse, "unary")) }

	defer trace(ctx)

	switch p.tok {
	case ASSIGN: // Example: '=xxx'
		if !lhs && p.bits&parseNoPair == 0 {
			var v Value
			var s = p.Position()
			if p.step(); p.isEndOfList(false) {
				v = makeNull(s)
			} else {
				v = p.expr(ctx)
			}
			x = &pair{makeNull(s), v}
			return
		}

	case BAREWORD:
		return p.bare(ctx, lhs)

	case BINARY, OCTAL, INTEGER, HEXADECIMAL, FLOATING, DATETIME, DATE, TIME, URI, STRING/*, RAW*/:
		return p.literal(ctx, lhs)

	case COMPOUND:
		return p.compound(ctx, lhs)

	case CLOSURE:
		return p.closuredelegate(ctx, true)

	case DELEGATE:
		return p.closuredelegate(ctx, false)

	case ESCAPE: // \
		return p.escape(ctx, lhs)

	case LPAREN: // (
		return p.group(ctx, lhs)

	case LBRACE: // {
		return p.braced(ctx, lhs)

	case COMMA:
		if p.bits&parseCall == 0 {
			return p.punctuation()
		}

	case AT, BAR, PLUS, SEMICOLON:
		return p.punctuation()

	case STAR, DAST, QUE, LBRACK: // * ** ? [
		return p.glob(ctx, nil) // ie. no prefix

	case PERC: // %bar (ie. no prefix)
		return p.perc(ctx, lhs, nil)

	case MINUS:
		return p.flag(ctx, lhs)

	case EXC:
		return p.negative(ctx, lhs)

	case PCON: // The root of the path
		return p.path(ctx, lhs, _pathpun(ctx, PROOT))

	case TILDE: // ~
		tok, ctx := p.tok, p.ctx(ctx)
		p.step() // TODO: ~user, aka $(HOME)
		return _pathpun(ctx, tok)

	case DOT, DOTDOT: // . ..
		pos, tok := p.Position(), p.tok
		switch p.step(); {
		case p.tok == PCON:
			ctx = at(ctx, pos)
			return p.path(ctx, lhs, _pathpun(ctx, tok))
		case tok == DOT, tok == DOTDOT:
			x = &punctuation{valbase{pos}, tok}
			if p.bits&parseDOT == 0 { x = p.dot(ctx, lhs, x) }
			return
		default:
			erro(at(ctx,pos), "unexpected token: %v, %v %s", tok, p.tok, p.lit).debug(1)
			return makeNull(pos)
		}

	default:
		if t := p.tok.isClosure(); t || p.tok.isDelegate() {
			return p.special(ctx, t, lhs)
		} else if p.tok.isKeyword() { // keywords here are barewords
			return p.bare(ctx, lhs)
		}
	}

	if p.lineComment != nil {
		for _, comment := range p.lineComment.list {
			erro(at(p,comment.pos), "# %s", comment.string).debug(1)
		}
	}

	erro(p, "bad: %v (lit=%s, left=%v, bits=%022b, scan=%v)",
		p.tok, p.lit, lhs, p.bits, p.scanner.scanstate).debug(1)

	p.step() // go to the next token

	return makeNull(ctx.Position())
}

func (p *parser) isParametersGroup(x Value) (res bool) {
	if p.bits&parseDepend0 != 0 { if g, y := x.(*group); y && len(g.elems) == 1 {
		_, res = g.elems[0].(*group)
	}}
	return
}

func (p *parser) composite(ctx Context, lhs bool) (x Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "composite")) }

	defer trace(ctx)

	x = p.unary(ctx, lhs)

	switch p.tok { // check composible expressions
	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2: // foo->bar  foo=>bar  foo~>bar
		// accepts 'foo=>bar', but 'foo => bar' is different
		if p.bits&parseNoSelect == 0 { x = p.selectExpr(ctx, x); break }

	case STAR, DAST, QUE, LBRACK: // * ** ? [
		if p.bits&parseGLOB == 0 && p.tok == QUE { switch p.step(); p.tok {
		case SPACE, RPAREN, RBRACK, RBRACE, COMMA, SELECT_PROP, SELECT_PROG1, SELECT_PROG2, LINEND:
			return condish(ctx, x)
		}}

		if _, y := x.(*globpat); !y && p.bits&parseNoGlob == 0 { x = p.glob(ctx, x) }

	case PERC: // foo%bar ; FIXME: %/foo/bar -> Path(% foo bar)
		if p.bits&parseNoPerc == 0 { x = p.perc(ctx, lhs, x) }

	case DOT: // foo.bar.baz.o ; FIXME: push bits when parsing $(...)
		if p.bits&parseDOT == 0 { x = p.dot(ctx, lhs, x) } // TODO: parse to Qualiword

	// case PCON: // ie. subdir/in/somewhere
	// 	if p.bits&parseNoPath == 0 {
	// 		switch x.(type) { // Path expressions, except '-I/path/to/include'
	// 		case flag: // By pass expressions like -I/foo/bar.
	// 		default: x = p.path(ctx, lhs, x)
	// 		}
	// 	}

	case COLON:
		if (p.bits&parseRecipe != 0 || !lhs) && p.bits&parseNoURL == 0 {
			if isKnownURLScheme(x.string(at(ctx, p.Position()))) { x = p.url(ctx, lhs, x) }
		}
	}
	return
}

func (p *parser) expr(ctx Context, ab ...bool) (x Value) {
	if false && l_traverse.enabled { defer un(l_trace(l_traverse, "expr")) }

	defer trace(ctx)

	var tok, lit = p.tok, p.lit
	var lhs bool ; if len(ab)>0 { lhs = ab[0] }

	if x = p.composite(ctx, lhs); x == nil {
		erro(p, "invalid (%v,%v; prev=%v,%v)", p.tok, p.lit, tok, lit).debug(6)
		return
	}

	if p.bits&(parseGLOB) != 0 { return }
	if p.tok.isAssign() && lhs { return }
	if p.isParametersGroup(x)  { return }

	var n int

composeLoop:
	switch p.tok {
	case ASSIGN: // Example: 'key=value'
		if !lhs && p.bits&parseNoPair == 0 {
			x = p.pair(ctx, x)
		}
		return

	case SELECT_PROP, SELECT_PROG1, SELECT_PROG2: // Example: foobar⇒run(-gen)
		if p.bits&parseNoSelect == 0 {
			x = p.selectExpr(ctx, x)
			goto composeLoop
		}
		return

	case LPAREN:
		if p.bits&parseNoArg == 0 {
			if x = p.argumentedExpr(ctx, x); x != nil {
				goto composeLoop
			}
		}
		return

	case PCON:
		if p.bits&parseNoPath == 0 {
			// Path, excepts '-I/path/to/include'
			switch x.(type) {
			case flag:
			default:
				x = p.path(ctx, lhs, x)
			}
		}
		return // FIXES: a%%b/foo/bar -> Path(a%%b foo bar)

	case BAR: // Example: [(var)|...]
		if _, y := x.(*group); y { return }

	case COMMA:
		if p.bits&(parseArged|parseCall|parseGroup|parseModifier) != 0 { return }
		if p.bits&(parseDefineClause) == 0 {
			note(p, "%v %v '%v' (%016b)", us(x), p.tok, p.lit, p.bits).debug(1)
			return
		}

	case COMPOSED, COLON, RAW, RPAREN, RBRACK, RBRACE, SPACE, SEMICOLON, LINEND, EOF:
		return // terminate
	}

	x = compose(ctx, x, p.composite(ctx, lhs))

	switch p.tok { case SPACE, COMMENT, LINEND, EOF: return }

	n += 1 ; goto composeLoop // compose as many as possible
}

// ----------------------------------------------------------------------------
// Clauses & Declarations

type clauseopts struct {
	generalOpts

    keyword token // e.g. use, files, eval, etc.

    skip bool // e.g. -cond(false{}), -if(no{})

	conds []Value `if,cond,where`

    values, remainder []Value // all values (unparsed) and remainder
	spec []Value
}

type parseSpecFunc func(Context, *commentGroup, *clauseopts, int)

func isValidImport(lit string) bool {
	const illegalChars = `!"#$%&'()*,:;<=>?[\]^{|}` + "`\uFFFD"
	s, _ := strconv.Unquote(lit) // go/scanner returns a legal string literal
	for _, r := range s {
		if !unicode.IsGraphic(r) || unicode.IsSpace(r) || strings.ContainsRune(illegalChars, r) {
			return false
		}
	}
	return s != ""
}

func (p *parser) _parseUseSpecProps(ctx Context, props []Value) (opts useOpts, params []Value, err error) {
	ctx = p.ctx(ctx)

    // Supported parameter forms:
    //      -param
    //      -param(value)
    //      -param=value
    var useList []Value // TODO: apply useList
    for _, prop := range props {
        var s string
        switch t := prop.(type) {
        case flag:
            switch s = t.Value.string(ctx); s {
            //case "nouse", "unuse": opts.unuse = true
            case "reuse": opts.reuse = true
            default: params = append(params, prop)
            }
        case *pair: // -param=value
            switch tt := t.key.(type) {
            case flag:
                switch s = tt.Value.string(ctx); s {
                case "use": useList = append(useList, t.val)
                default: params = append(params, prop)
                }
            default:
                erro(at(ctx,t.key), "parameter `%v' unsupported `%T`", prop, prop)
                return
            }
        case *argumented: // -param(value)
            switch tt := t.Value.(type) {
            case flag:
                switch s = tt.Value.string(ctx); s {
                case "use": useList = append(useList, t.args...)
                default: params = append(params, prop)
                }
            default:
                erro(at(ctx,t.Value), "parameter `%v' unsupported `%T`", prop, prop)
                return
            }
        default:
            erro(at(ctx,prop), "parameter `%v` unsupported `%T`", prop, prop)
            return
        }
    }
    return
}

func (p *parser) use(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	if p.imports = append(p.imports, &usespec{ g.spec }); g.skip {
		// TODO: maybe give some information
		return
	}

	ctx = at(ctx, g.spec[0].Position()) // p.ctx()

	defer trace(ctx)

	var specVals, arged []Value
	switch v := g.spec[0].(type) {
	case *delegate:
        for _, val := range xmerge(ctx, v) {
            if !isTrivial(val) { specVals = append(specVals, val) }
		}
    case *pair:
        var s string
        if f, ok := v.key.(flag); !ok {
            erro(ctx, "'%v' invalid use spec", v.key)
            return
        } else if s = f.Value.string(ctx); s != "list" {
            erro(ctx, "'%v' invalid use spec, do you mean -list?", v.key)
            return
        }

        for _, val := range xmerge(ctx, v.val) {
            if !isTrivial(val) { specVals = append(specVals, val) }
        }
	case *argumented:
        for _, val := range xmerge(ctx, v.Value) {
            if !isTrivial(val) { specVals = append(specVals, val) }
        }
		arged = v.args
	default:
		specVals = append(specVals, v)
    }
	if len(specVals) == 0 {
        erro(ctx, "empty use spec: %v", us(g.spec[0])).debug(1)
        return
    }

	var opts useOpts
	var args = parseOpts(ctx, &opts, append(g.remainder, g.spec[1:]...)...)
	for _, a := range args {
		if _, ok := a.(flag); ok || true {
			erro(at(ctx,a), "unkown use opts: %v", us(a)).debug(1)
			return
		}
	}

	var wg sync.WaitGroup
	var l = _loader(ctx)
	for _, specVal := range specVals {
		if ctx := at(ctx, specVal.Position()); true {
			l.usespec(ctx, opts, specVal, arged, args...)
		} else {
			var dc = diagnostic{ Context: ctx }
			wg.Add(1); go func() {
				defer func() {
					if false { trace(&dc) }
					if len(dc.points) > 0 { _diagnostic(ctx).nest(dc.points) }
					wg.Done()
				} ()
				l.usespec(ctx, opts, specVal, arged, args...)
			} ()
		}
	}

	wg.Wait()
	return
}

func (p *parser) include(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Spec")) }

	var opts = includeOpts{ clauseopts: g }
	if vals := parseOpts(ctx, &opts, g.remainder...); len(vals) > 0 {
		// TODO: deal with the unparsed generic options
		warn(ctx, "unknown opts: %v", vals).debug(1)
	}

	if len(g.spec) < 1 {
		erro(ctx, "expecting include file: %v", g.spec).debug(1)
		return
	}

	var x = g.spec[0]//.expand(ctx, final|expandPlaceholder)
	var l = _loader(ctx)
	if p.spaces(); p.tok == COLON {
		switch x.(type) {
		case *File, *strlit, *compound: // escape from file searching
		default: if file := l.proj.file(ctx, x.string(ctx)); file != nil {
			x = file
		} else if val := x.expand(ctx); !isNull(val) && val != x {//, final
			x = val
		}}

		x = p.rule(ctx, specialRuleNor, nil, []Value{x}) // this should return a Rule
	}

	if !g.skip { l.include(ctx, opts, x) }
}

func (p *parser) files(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	defer trace(ctx)

	if len(g.spec) != 1 {
		erro(ctx, "too many files properties: %v", g.spec).debug(1)
		return
	}

	var path Value
	if p.tok == SELECT_PROG1 {
		p.next(true) // step forward with spaces skipped
		if p.tok == LINEND || p.lineComment != nil {
			erro(ctx, "expecting files path")
		}
		path = p.expr(ctx)
	}

	p.spaces()

	if p.lineComment != nil {
		//spec.Comment = p.lineComment
	}
	if g.skip {
		// TODO: maybe give some information
		return
	}

	ctx = p.ctx(ctx)

	var opts = cacher{ g.generalOpts }
	if rest := parseOpts(ctx, &opts, g.remainder...); rest != nil {
		erro(ctx, "unsupported opts: %v", rest).debug()
		return
	}

	var pats []Value
	if x, y := g.spec[0].(*group); y {
		pats = x.elems
	} else if indeterminate(ctx, g.spec[0]) {
		pats = []Value{ g.spec[0] }
	} else {
		pats = xmerge(evaluation{ctx, defExpand1}, g.spec[0])
	}

	if path == nil {
		if len(pats) == 1 { if a, ok := pats[0].(*argumented); ok { if f, y := a.Value.(flag); y {
			var name = f.Value.string(ctx)
			switch name {
			default:
				// TODO: parse files options
				erro(at(ctx,f.Value), "invalid files flag: %v").debug()
				return
			}
		}}}

		var (
			files []*File
			newPats []Value
		)
		for _, pat := range pats {
			if f, ok := toFile(pat); ok {
				files = append(files, f)
			} else {
				newPats = append(newPats, pat)
			}
		}
		if len(files) > 0 {
			opts.cache(ctx, values(files), nil)
			pats = newPats
		}
		if len(pats) > 0 {
			var paths = []Value{ makeStrlit(g.spec[0].Position(), ctx.project().absPath) }
			opts.cache(ctx, pats, paths)
		}
	} else {
		var patsNew []Value
		for _, pat := range pats {
			if indeterminate(ctx, pat) {
				patsNew = append(patsNew, pat)
			} else {
				patsNew = append(patsNew, xmerge(ctx, pat)...)
			}
		}

		var paths []Value
		if g, ok := path.(*group); ok {
			paths = g.elems
		} else {
			paths = []Value{ path }
		}

		if len(patsNew) == 1 {
			if f, ok := patsNew[0].(flag); ok {
				var name = f.Value.string(ctx)
				switch name {
				default:
					// TODO: parse files options
					erro(at(ctx,f.Value), "invalid files flag: %v").debug()
					return
				}
			}
		}

		opts.cache(ctx, patsNew, paths)
	}
}

func (p *parser) evalConfiguration(ctx Context, g *clauseopts, props []Value) {
	var project = ctx.project()
	if project == nil {
		erro(ctx, "configuration: nil project").debug(1)
		return
	} else if project.configure == nil {
		erro(ctx, "configuration: no %s for %v", dotConfigure, project).debug(1)
		return
	}

	if entry := project.configure.defaultEntry; entry == nil {
		// no init entry from .configure
	} else if _, ts := entry.execute(at(ctx, entry.Position())); len(ts) > 0 {
		// FIXME: the entry might be a configure operation (see configure/.base/do.smart)
		for _, brk := range ts {
			if brk.what == traveFail {
				erro(at(ctx,entry), "execute '%v' failed: %v", entry, brk).debug(1)
			}
		}
	}

	if flush(ctx)>0 { return }
	if project.configured {
		prompt(ctx, "configuration: %v already configured\n", project)
		return
	}

	var ce = configureContext{Context:ctx} ; defer ce.close()

	for _, dep := range xmerge(ctx, props/* [1:] */...) {//, final
		if re, y := dep.(*rule); !y {
			erro(ctx, "unsupported prerequisite: %T %v", dep, dep).debug(1)
		} else if _, ts := re.execute(ctx); len(ts) > 0 {
			for _, brk := range ts { if brk.what == traveFail {
				erro(at(ctx,re), "execute '%v' failed: %v", re, brk).debug(1)
			}}
		}
	}

	if flush(ctx)>0 { return }

	/***/ promptEnteringDirectory(ctx, project.absPath)
	defer promptLeavingDirectory(ctx, project.absPath)

	for _, entry := range project.configs { ce.execute(entry) }

	project.configured = true // relaxes configure()
}

func (p *parser) assert(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	if !g.skip { call(final{ctx}, "assert", g.remainder, g.spec...) }
}

func (p *parser) append(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	if !g.skip { call(final{ctx}, "append", g.remainder, g.spec...) }
}

func (p *parser) eval(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	var (
		prop0, resolved, res Value
		name string
	)

	defer trace(ctx)

	if g.skip { return } else if g.spec == nil {
		var opts struct {
			configuration bool `configuration`
			optimize Value `opt,optimize`
		}
		for _, op := range parseOpts(ctx, &opts, g.values...) {//, plain
			var val Value
			if v, y := op.(*pair); y { op, val = v.key, v.val }
			if v, y := op.(flag); y {
				switch t := val != nil && val.true(ctx); v.Value.string(ctx) {
				case "dd": p.dd = t
				case "ddd":
					if u := _universe(ctx); val == nil {
						u.ddd = "yes"
					} else if t, y := boolVal(val); y {
						if t { u.ddd = "yes" } else { u.ddd = "" }
					} else {
						u.ddd = val.string(ctx)
					}
				}
			} else {
				erro(at(ctx,op), "unsupport flag: %v (%v)", us(v), val).debug(1)
			}
		}

		// NOTE: see also universeContext.configure()
		if opts.configuration { p.evalConfiguration(ctx, g, g.spec) }
		return
	} else if prop0 = g.spec[0]; isTrivial(prop0) {
		erro(ctx, "illegal").debug(1)
		return
	}

	var opts []Value
	if a, y := prop0.(*argumented); y { prop0, opts = a.Value, a.args }

	ctx = at(ctx, prop0.Position())

	var l = _loader(ctx)
	if name, resolved = l.resolve(ctx, prop0); false {
		erro(ctx, "resolve '%v' failed", prop0).debug(1)
		return
	} else if name == "configuration" {
		erro(ctx, "use '-configuration' instead (%v)", prop0).debug(1)
		return
	} else if x, y := resolved.(invoker); y {
		if b, y := x.(*builtin); y && !b.isCommand() {
			erro(ctx, "resolved builtin '%v' is not a command", prop0).debug(1)
			return
		}
		res = x.invoke(ctx, opts, g.spec[1:])
	} else {
		erro(ctx, "resolved '%v' is %s (%v)", prop0, typeof(resolved), *g).debug(1)
		return
	}

	if isTrivial(res) { return }

	/* TODO: if c, y := res.(code); y { ... } */
}

func (p *parser) directive(ctx Context) (props []Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "spec")) }

	defer trace(ctx)

	//var doc = p.leadComment
	var comment *commentGroup

ParamsParseLoop: // Parse the directive parameters
	for p.tok != EOF {
		switch p.spaces(); p.tok {
		case COMMA, LINEND, RPAREN, RBRACE,
			SELECT_PROG1, COLON: break ParamsParseLoop
		}

		if p.lineComment != nil {
			comment = p.lineComment
			break
		}

		props = append(props, p.expr(ctx))
	}
	if comment != nil { /* TODO: directive documments */ }
	return
}

func (p *parser) spec(ctx Context, keyword token, pos Pos, f parseSpecFunc) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "spec("+keyword.String()+")")) }

	defer trace(ctx)

	var opts = clauseopts{ keyword: keyword }
	for p.spaces(); p.tok == MINUS; p.spaces() {
		opts.values = append(opts.values, p.expr(ctx))
	}
	opts.remainder = parseOpts(ctx, &opts, opts.values...)

	for _, cond := range opts.conds {
		if t := cond.true(at(ctx, cond.Position())); !t {
			opts.skip = true
			break
		}
	}

	if p.spaces(); p.tok == LINEND {
		if keyword == EVAL { f(p.ctx(ctx), nil, &opts, 0) } else {
			erro(p.ctx(ctx), "%v: nil specs", keyword).debug(1)
		}
		return
	} else if p.tok == LPAREN {
		p.next(true)
		for iota := 0; p.tok != RPAREN && p.tok != EOF && (p.stop == 0 || p.pos < p.stop); iota++ {
			// TODO: collect documentation comments
			for p.tok == SPACE || p.tok == LINEND { p.next(true) }
			if p.tok == RPAREN || p.tok == EOF { break  }
			if opts.spec = p.directive(ctx); true {
				f(p.ctx(ctx), p.leadComment, &opts, iota)
			}
			if p.tok == COMMA || p.tok == LINEND { p.next(true) }
		}
		p.expect(RPAREN)
		if p.spaces(); p.tok != EOF { p.linend() }
		return
	}

	if p.tok != LINEND && p.tok != EOF && (p.stop == 0 || p.pos < p.stop) {
		if opts.spec = p.directive(ctx); true { f(ctx, nil, &opts, 0) }
		if p.tok == COMMA { p.next(true) }
	}
	if p.tok != EOF && (p.stop == 0 || p.pos < p.stop) {
		if p.spaces(); p.lineComment == nil { p.linend() }
	}
}

func (p *parser) assign_value(ctx Context, tok token) (value Value) {
	defer func(b parseBits, c Context) {
		p.bits, p.Context, p.lineComment = b, c, nil
	} ( p.bits, p.Context )

	p.bits |= parseDefineClause

	if false {
		var n = makeNull(ctx.Position())
		var c = automatic{ Context:ctx, defs:make(autodefs) }
		for i := 0; i < 10; i += 1 { c.set(ctx, strconv.Itoa(i), undef{n}) }
		return ease(ctx, p.values(&c))
	} else {
		return ease(ctx, p.values(ctx))
	}
}

func (p *parser) assign(ctx Context, ident Value) (def *def) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, fmt.Sprintf("assign(%s)", ident))) }

	var tok = p.tok

	p.next(true) // assign token

	ctx = p.ctx(ctx)

	defer trace(ctx)

	// TODO: doc = p.leadComment
	// TODO: comment = p.lineComment
	var value = p.assign_value(ctx, tok)

	// NOTE: Put all explicit defs into project scope. It's important for defs enclosed
	//       in templates work.
	var l = _loader(ctx)
	if scope := l.proj.scope_; len(l.scopes) == 0 || l.scopes[0] != scope {
		defer func(s []*Scope) { l.scopes = s } (l.scopes)
		l.scopes = append([]*Scope{ scope }, l.scopes...)
	}

	var defs = l.define(ctx, tok, ident, value)
	if n := len(defs); n > 0 { def = defs[n-1] }
	if checkpoints {
		if def == nil {
			erro(ctx, "%v %v %v", ident, tok, us(value)).debug(3)
		} else if def.value == nil && value != nil {
			erro(ctx, "%v %v %v", ident, tok, us(value)).debug(3)
		}
	}
	return
}

func (p *parser) recipe(ctx Context) Value {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "Recipe")) }

	var (
		// TODO: comment *commentGroup
		// TODO: doc = p.leadComment
		l = _loader(ctx)
		position = p.Position()
		elems []Value
		isList bool
	)

	switch p.dialect {
	case "", "eval", "value":
		p.scanner.pop(isCompoundLine)
		p.next(true) // skip RECIPE or SEMICOLON and parse in list mode
		position = p.Position()
		if isList = true; !p.isEndOfLine() {
			var a *argumented
			var x = p.expr(ctx) // parse first expr of recipe
			if x != nil { if a, _ = x.(*argumented); a != nil { x = a.Value } }
			if x == nil {
				erro(ctx, "parsed value is nil").debug(1)
			} else if p.dialect == "value" {
				// no resolving commands
			} else if t, y := x.(*bareword); !y {
				// does nothing
			} else if _, sym := l.resolve(ctx, t); false {
				erro(ctx, "resolve '%v' failed", x).debug(1)
			} else if isTrivial(sym) {
				erro(at(ctx,x), "resolved '%v' (from %v) is nil", t.s, x).debug(1)
			} else if false {
				erro(at(ctx,x), "builtin command no more supported, use $(%s ...) instead", t.s).debug(1)
			} else if b, y := sym.(*builtin); !y {
				erro(at(ctx,x), "'%s' is not a command (%s)", t.s, typeof(sym)).debug(1)
			} else if !b.isCommand() {
				erro(at(ctx,x), "'%s' is not a command, use $(%s ...) instead", t.s, t.s).debug(1)
			} else { x = sym }

			if a != nil {
				elems, a.Value = append(elems, a), x
			} else {
				elems = append(elems, x)
			}

			var cmdargs []Value

			p.setbit(parseRecipeBuiltin)
			for p.tok != EOF && p.tok != SEMICOLON && p.tok != LINEND && p.lineComment == nil {
				if p.spaces(); p.lineComment != nil { break }
				if !p.tok.isRuleDelim() { x = p.expr(ctx) } else
				if false { x = p.rule(ctx, specialRuleRec, nil, elems) } else {
					erro(ctx, "unsupported token: %s, %v", p.tok, elems).debug(1)
				}
				if cmdargs = append(cmdargs, x); p.tok == COMMA {
					p.next(true)
					elems = append(elems, makeList(cmdargs...))
					cmdargs = []Value{}
				}
				if p.lineComment != nil { break }
			}
			p.clearbit(parseRecipeBuiltin)
			elems = append(elems, makeList(cmdargs...))
		}

	default:
		p.scanner.push(isCompoundLine) // NOTE: scanner does not set isCompoundLine correctly, fixit here
		p.next(true) // skip RECIPE or SEMICOLON and parse in line-string mode
		position = p.Position()
		p.setbit(parseRecipeText)
		for !p.isEndOfLine() {
			var x Value
			if p.tok == RAW {
				x = p.literal(ctx, false)
			} else {
				x = p.expr(ctx)
			}
			elems = append(elems, x)
		}
		p.clearbit(parseRecipeText)
		p.scanner.pop(isCompoundLine)
	}
	if p.spaces(); p.tok != EOF { p.linend() }
    if len(elems) == 0 {
        return makeNone(position)
    } else if isList {
        return makeList(elems...)
    } else {
        return makeCompound(elems...)
    }
}

// Parsing (var a=xxx,b=yyy) definitions
func (p *parser) movar(ctx Context, args ...Value) (err error) {
	var l = _loader(ctx)
	for _, elem := range args {
		var kv, ok = elem.(*pair)
		if !ok || kv == nil {
			erro(at(ctx,elem), "bad var form (%T)", elem).debug(1)
			continue
		}

		var name string
		var k, v = kv.key, kv.val
		if name = k.string(at(ctx, k.Position())); name == "" {
			erro(at(ctx,k), "name '%v' is empty", k).debug(1)
		}

		if def, alt := l.def(elem.Position(), name); alt != nil {
			erro(at(ctx,k), "'%v' already defined: %T", name, alt).debug(1)
		} else if def == nil {
			erro(at(ctx,k), "'%v' not defined", name).debug(1)
		} else {
			if g, y := v.(*group); y { v = g.list() }
			def.val(at(ctx,v.Position()), v)
		}
	}
	return
}

func (p *parser) defineConfigureTargets(ctx Context) {
	var l = _loader(ctx)
	for _, t := range p.targets {
		var pos = t.Position()
		if !pos.IsValid() { pos = p.Position() }

		var ctx = at(ctx, pos)
		var name = t.string(ctx)
		var d, a = l.proj.scope_.define(ctx, defConfig, name, nil)
		if d == nil && a != nil { if d, _ = a.(*def); d == nil {
			erro(ctx, "configure %v: already defined in '%v' as %v", t, l.project, a).debug(6)
			return
		}}

		if !d.position.IsValid() { d.position = pos }
	}
}

func (p *parser) modifier(ctx Context) (res *modifier) {
	p.spaces()

	defer p.setbits(p.setbit(parseModifier))
	p.clearbit(parseCall) // for commas

	ctx = p.ctx(ctx)

	p.expect(LPAREN)
	p.spaces()

	var name string
	var nameVal = p.expr(ctx)
	var elems []Value
	switch n := nameVal.(type) {
	case *bareword: name = n.s
	case *delegate, *closure:
		var ctx = at(ctx, n.Position())
		var v = xmerge(ctx, nameVal)//, final
		if len(v) == 0 {
			erro(ctx, "empty modifier name: %v", n).debug(1)
			return
		}
		name, elems = v[0].string(ctx), v[1:]
	default:
		erro(ctx, "unsupported modifier: %v", us(n)).debug(1)
		return
	}

	var movar bool
	switch name {
	case "var": movar = true
	case "configure":
		p.defineConfigureTargets(ctx)
		p.configure = true // set configure flag and define configure variables
	case "":
		erro(ctx, "empty modifier name: %v", us(nameVal)).debug(1)
		return
	}

	if _, y := dialects[name]; y {
		if p.dialect != "" {
			erro(ctx, "multi-dialects unsupported, already defined '%s'", p.dialect).debug(1)
			return
		}

		p.dialect = name
	} else if _, y = modifiers[name]; !y {
		erro(ctx, "`%s` no such dialect or modifier", name).debug(1)
		return
	}

	for p.tok != RPAREN && p.tok != EOF {
		p.spaces()

		t := p.pos

		if vals := p.values(ctx); movar {
			p.movar(ctx, vals...) // TODO: define var one by one
		} else if n := len(vals); n == 1 {
			elems = append(elems, vals[0])
		} else if n > 1 {
			elems = append(elems, &list{elements{vals}})
		} else {
			elems = append(elems, &null{p.valbase()})
		}

		if p.tok == COMMA { p.next(true) }
		if p.pos == t {
			erro(ctx, "unsupported modifier arg: %v '%v'", p.tok, p.lit).debug(1)
			return
		}
	}

	p.expect(RPAREN)

	if nameVal == nil && len(elems) == 0 {
		erro(ctx, "empty modifier").debug(1)
	} else {
		res = new(modifier)
		res.position = ctx.Position()
		res.elems = append([]Value{nameVal}, elems...)
	}
	return
}

func (p *parser) modification(ctx Context) *modification {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "modification")) }

	// defer p.setbits(p.setbit(parseModification))

	ctx = p.ctx(ctx) // at(ctx, p.loc(p.expect(LBRACK)))

	var elems []*modifier
	for p.tok != EOF && p.tok != LINEND && p.tok != /* RBRACK */RBRACE {
		if m := p.modifier(ctx); m != nil { elems = append(elems, m) }
	}

	// p.expect(/* RBRACK */RBRACE)

	if len(elems) == 0 {
		errostack(ctx, 5, "empty modifier group").debug(1)
	}
	if p.tok == COLON {
		errostack(ctx, 5, "unexpected colon after modifer").debug(1)
	}
    return &modification{valbase{ctx.Position()}, elems }
}

// 
// $@      The file name of the target of the rule. If the target is an archive member, then ‘$@’ is the name of the archive file. In a pattern rule that has multiple targets (see Introduction to Pattern Rules), ‘$@’ is the name of whichever target caused the rule’s recipe to be run.
// $%      The target member name, when the target is an archive member. See Archives. For example, if the target is foo.a(bar.o) then ‘$%’ is bar.o and ‘$@’ is foo.a. ‘$%’ is empty when the target is not an archive member.
// $<      The name of the first prerequisite. If the target got its recipe from an implicit rule, this will be the first prerequisite added by the implicit rule (see Implicit Rules).
// $?      The names of all the prerequisites that are newer than the target, with spaces between them. For prerequisites which are archive members, only the named member is used (see Archives).
// $^      The names of all the prerequisites, with spaces between them. For prerequisites which are archive members, only the named member is used (see Archives). A target has only one prerequisite on each other file it depends on, no matter how many times each file is listed as a prerequisite. So if you list a prerequisite more than once for a target, the value of $^ contains just one copy of the name. This list does not contain any of the order-only prerequisites; for those see the ‘$|’ variable, below.
// $+      This is like ‘$^’, but prerequisites listed more than once are duplicated in the order they were listed in the makefile. This is primarily useful for use in linking commands where it is meaningful to repeat library file names in a particular order.
// $|      The names of all the order-only prerequisites, with spaces between them.
//         Order-only prerequisites can be specified by placing a pipe symbol (|) in the prerequisites list: any prerequisites to the left of the pipe symbol are normal; any prerequisites to the right are order-only.
// $*      The stem with which an implicit rule matches (see How Patterns Match). If the target is dir/a.foo.b and the target pattern is a.%.b then the stem is dir/foo. The stem is useful for constructing names of related files.
//         In a static pattern rule, the stem is part of the file name that matched the ‘%’ in the target pattern.
//         In an explicit rule, there is no stem; so ‘$*’ cannot be determined in that way. Instead, if the target name ends with a recognized suffix (see Old-Fashioned Suffix Rules), ‘$*’ is set to the target name minus the suffix. For example, if the target name is ‘foo.c’, then ‘$*’ is set to ‘foo’, since ‘.c’ is a suffix. GNU make does this bizarre thing only for compatibility with other implementations of make. You should generally avoid using ‘$*’ except in implicit rules or static pattern rules.
//         If the target name in an explicit rule does not end with a recognized suffix, ‘$*’ is set to the empty string for that rule.
//
// Similar to makefile automatic variables, see
//   * https://www.gnu.org/software/make/manual/html_node/Automatic-Variables.html#Automatic-Variables
var automatics = []string{
	"@" , "%" , "<" , ">" , "?" , "^" , "+" , "|" , "*" , //
	"@D", "%D", "<D", ">D", "?D", "^D", "+D", "|D", "*D", //
	"@F", "%F", "<F", ">F", "?F", "^F", "+F", "|F", "*F", //
	"@'", "%'", "<'", ">'", "?'", "^'", "+'", "|'", "*'", //
	"-" , "~" ,
}

func (p *parser) rule(ctx Context, special specialRule, optvals, targets []Value) (result Value) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "rule")) }

	if ctx = p.ctx(ctx); ctx.project().keyword == PACKAGE {
		erro(ctx, "rules forbidden: %v", targets).debug(1)
		return
	}

	var (
		// TODO: doc = p.leadComment
		depends, ordered, recipes []Value
		position = ctx.Position()
	)

	var l = _loader(ctx)
	defer l.closeScope(l.openScope(fmt.Sprintf("rule %v", targets)))

	p.params = nil
	p.dialect = ""

	var scope = l.scope()
	for _, s := range automatics {
		if a := scope.auto(ctx, s); a == nil { erro(ctx, "'%s' is not defined", s).debug(1) }
	}
	for i := 1; i < 10; i += 1 { s := strconv.Itoa(i)
		if a := scope.auto(ctx, s); a == nil { erro(ctx, "'%s' is not defined", s).debug(1) }
	}

	// NOTE: expand targets to speed up for later usage, it might spend lots of time in
	// project.entry while matching for entry looked up if not expanded right now.
	targets = expand(ctx, targets...)

	defer func(t []Value) { p.targets = t } (p.targets)
	p.targets = targets // save targets for later refering
	p.next(true) // skip rule delimeters and spaces

	if p.tok != SEMICOLON && p.tok != BAR && !p.isEndOfLine() {
		depends = p.depends(ctx, true)
	}
	if p.tok == BAR { // '|' starts the ordered prerequisites
		if p.next(true); p.tok != SEMICOLON && !p.isEndOfLine() {
			ordered = p.depends(ctx, false)
		}
	}

	if p.tok == SEMICOLON { // ;
		// Parse inline recipe in the program scope.
		recipes = append(recipes, p.recipe(ctx))
	} else /*if p.tok == LINEND || p.lineComment != nil*/ {
		// Parse recipes in the program scope.
		p.scanner.recipes(true) // Turn on recipes before LINEND.
		if p.linend() { // Take the new line.
			for p.tok != EOF && p.isRecipeStart() {
				recipes = append(recipes, p.recipe(ctx))
			}
		}
		p.scanner.recipes(false)
	}

	var params []string
	if t := targets[0]; p.configure {
		name := t.string(ctx)
		d, a := ctx.project().scope_.define(ctx, defVoid, name, nil)
		if d == nil && a == nil {
			erro(at(ctx,t), "cannot define configure target '%v'", name)
		} else if a != nil {
			if _, ok := a.(*def); !ok {
				erro(at(ctx,t), "configure target '%v' already taken: %T %v", name, a, a)
			}
		}
		if d != nil && !d.position.IsValid() { d.position = t.Position() }
	} else {
		for _, d := range p.params { params = append(params, d.ident(ctx)) }
	}

	parsedData := &parsedRuleData{
		// TODO: lang: 0,
		params:   params,
		position: position,
		config:   p.configure,
		targets:  targets, //barefilize(ctx, targets...),
		depends:  depends, //barefilize(ctx, depends...),
		ordered:  ordered, //barefilize(ctx, ordered...),
		recipes:  recipes,
		options:  optvals,
		special:  special,
	}

	if special != specialRuleRec {
		var res []entry
		var l = _loader(ctx)
		if res = l.rule(parsedData); len(res) == 1 {
			result = res[0]
		} else if len(res) > 1 {
			result = _makeList[entry](res...)
		} else {
			result = makeNull(parsedData.position)
		}
	}

	// Close the rule scope and go back to project scope. The current
	// scope must be project scope befor Rule.
	p.configure = false
	p.dialect = ""
	p.params = nil
	return
}

func (p *parser) specialRule(ctx Context) Value {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "SpecialRule")) }

	p.expect(COLON) // expect and skip ':'

	if p.tok != BAREWORD {
		erro(p, "unknown special rule")
		return nil
	}

	var name = p.lit
	switch name {
	case "user":
		if true {
			// Example usage of use.*:
			//    use.* ::= cflags(-unique) ldlibs(-unique -reverse)
			erro(p, ":user: rules are deprecated, use use.* instead!").debug(1)
		} else {
			var options []Value
			var pos = p.expect(BAREWORD) // USE
			var bits = p.setbit(parseSpecialRule)
			var ctx = p.ctx(ctx)
			// Options are flag or *pair of a Flag.
			for p.tok == MINUS {
				opt := p.expr(ctx)
				options = append(options, opt)
			}
			p.setbits(bits) // restore bits
			if p.tok.isRuleDelim() {
				return p.rule(ctx, specialRuleUse, options, []Value{
					makeBareword(p.loc(pos), name),
				})
			}

			erro(p, "expecting special rule terminator ':'")
		}
		return nil
	default:
		erro(p, "unknown special rule")
		return nil
	}
}

var pprofCounter int

func (p *parser) def(ctx Context) {
	defer trace(ctx)

	p.spaces()
	p.expect(DEF)
	p.spaces()

	var args []Value
	var name = p.expr(ctx)
	if a, y := name.(*argumented); y {
		name, args = a.Value, a.args
	}

	t := &template{
		pos: p.pos, tok: p.tok, lit: p.lit,
		state: p.scanner.scanstate,
		name: name, params: args,
	}

	p.spaces()
	p.linend()

	var linend = true
	var nested = 0
	for { switch p.tok {
	default:
		linend = false
		p.next(true)

	case LINEND:
		linend = true
		p.next(true)

	case DEF:
		if linend { nested += 1 }
		linend = false
		p.next(true)

	case END:
		if !linend || 0 < nested {
			if linend { nested -= 1 }
			linend = false
			p.next(true)
			continue
		}

		pos := p.pos
		p.next(true)
		p.linend()

		state := p.scanner.scanstate
		t.end, t.endPos = &state, pos
		p.templates = append(p.templates, t)
		return

	case EOF:
		return
	}}
}

func (p *parser) foreach(ctx Context) {
	defer trace(ctx)

	if p.spaces(); p.tok == LINEND {
		erro(p.ctx(ctx), "unexpected end of line").debug(1)
		return
	}

	p.expect(FOREACH)
	p.spaces()

	var params = p.values(ctx)
	var t = &template{
		pos: p.pos, tok: p.tok, lit: p.lit,
		state: p.scanner.scanstate,
	}

	p.spaces()
	p.linend()

	var nested = 0
	for p.tok != EOF { switch pos := p.pos; p.tok {
	case FOREACH:
		p.next(true) // foreach
		nested += 1

	case DONE:
		if nested > 0 { nested -= 1 ; continue }

		p.next(true) // done
		p.linend()

		state := p.scanner.scanstate
		t.end, t.endPos = &state, pos

		defer func(s Pos) { p.stop = s } (p.stop)
		p.stop = t.endPos

		var a = map[string]Value{ "_" : nil }
		for _, elem := range xmerge(final{ctx}, params...) {
			if indeterminate(ctx, elem) {
				erro(ctx, "indeterminate param: %v", us(elem)).debug(1)
			} else if isTrivial(elem) {
				if false { info(ctx, "trivial: %v", us(elem)).debug(1) }
			} else {
				a["_"] = elem
				p.codeblock(ctx, t, a)
			}
		}
		return

	default:
		for p.tok != EOF {
			if p.next(true); p.tok == LINEND { p.next(true) ; break }
		}
	}}
}

func (p *parser) for_(ctx Context) {
	defer trace(ctx)

	if p.spaces(); p.tok == LINEND {
		erro(p.ctx(ctx), "unexpected end of line").debug(1)
		return
	}

	var opts struct {
		skipNil bool `skip-nil,skip-null,skipnil,skipnull,no-nil,no-null`
		loose bool `loose`
	}

	if p.expect(FOR); p.tok == LPAREN {
		p.next(true) // LPAREN
		if vals := parseOpts(ctx, &opts, p.values(ctx)...); vals != nil {
			erro(at(ctx, vals[0]), "unexpected opts: %v", vals).debug(1)
		}
		p.expect(RPAREN)
	}

	p.spaces()

	type param struct {
		name string
		elems []Value
	}

	type nparam struct {
		p Position
		a []*param
		n int
	}

	var params []*nparam
	var vars = map[string]Value{}
	for p.spaces(); p.tok != EOF && !p.isEndOfLine(); p.spaces() {
		if p.tok == AND && params == nil {
			erro(p.ctx(ctx), "unexpected 'and'").debug(1)
			continue
		} else if p.tok == AND || params == nil {
			if params = append(params, &nparam{p:p.Position()}); p.tok == AND {
				p.next(true) // and
				continue
			}
		}

		var _v = params[len(params)-1]
		for _, a := range xmerge(p.ctx(ctx), p.expr(ctx)) {
			var elems []Value
			var s string

			if x, y := a.(*pair); !y {
				erro(at(ctx,a), "unexpected value: %v", us(a)).debug(1)
				return
			} else if s = x.key.string(at(ctx, x.key.Position())); s == "" {
				erro(at(ctx,a), "empty key: %v", us(x.key)).debug(1)
				return
			} else if g, y := x.val.(*group); y {
				elems = g.elems
			} else {
				elems = append(elems, x.val)
			}

			// Make sure all elements are expanded.
			elems = xmerge(at(ctx, a), elems...)

			if _, y := vars[s]; y {
				erro(at(ctx, a), "duplicated key: %v", s).debug(1)
				return
			} else {
				vars[s] = &null{valbase{a.Position()}}
			}

			if n := len(elems); n > _v.n { _v.n = n }

			_v.a = append(_v.a, &param{s, elems})
		}
	}

	var t = &template{
		pos: p.pos, tok: p.tok, lit: p.lit,
		state: p.scanner.scanstate, // verb: "for",
	}

	p.spaces()
	p.linend()

	var nested = 0
	for p.tok != EOF { switch pos := p.pos; p.tok {
	case FOR:
		p.next(true) // for
		nested += 1

	case DONE:
		if nested > 0 { nested -= 1 ; continue }

		p.next(true) // done
		p.linend()

		defer func(s Pos) { p.stop = s } (p.stop)

		state := p.scanner.scanstate
		t.end, t.endPos, p.stop = &state, pos, pos

		var num int
		for _, _v := range params {
			if _v.n > 0 { if num == 0 { num = _v.n } else { num *= _v.n } }
		}

		var l int = len(params)-1
	outer:
		for n := 0; n < num; n += 1 {
			for _i, _v := range params {
				// i[0]    = (n % 1) / b    (a = n * n * ..., k-1)
				// i[1..l] = (n % a) / b    (b = n * ..., k-2)
				// i[l+1]  = (n % a) / 1
				var i int = n

				// Two implements: 1. compact, 2. TODO: expand (loose)
				//    1. compact: use the minimum nparam, skip elements after it (DONE)
				//    2. expand: use the maximum nparam, treat every part the same (TODO)

				// 1. compact mode
				for k, t := range params { if t.n == 0 {
					if true { continue outer }
				} else if k <= _i {
					if 0 < _i { i %= t.n }
				} else {
					if _i < l { i /= t.n }
				}}

				for _, v := range _v.a { if i < len(v.elems) {
					vars[v.name] = v.elems[i]
				} else {
					vars[v.name] = &null{valbase{_v.p}}
					if opts.skipNil { continue outer }
				}}
			}

			var trivial bool = len(vars) == 0
			if !trivial { for _, v := range vars {
				if trivial = isTrivial(v); !trivial { break }
			}}
			if !trivial { p.codeblock(ctx, t, vars) }
		}
		return

	default:
		for p.tok != EOF {
			if p.next(true); p.tok == LINEND { p.next(true) ; break }
		}
	}}
}

func (p *parser) codeblock(ctx Context, t *template, vars map[string]Value) {
	p.pos, p.tok, p.lit, p.scanner.scanstate = t.pos, t.tok, t.lit, t.state

	defer trace(ctx)

	if false {
		pprofCounter += 1
		defer startCPUProfile(ctx, fmt.Sprintf("template-%05d.prof", pprofCounter), true)()
	}

	if !(p.pos < p.stop) {
		erro(at(ctx,p.loc(p.pos)), "bad range: [%v %v) (%v)", p.pos, p.stop, t.name).debug(10)
	}

	var c = automatic{ Context:ctx, defs:make(autodefs) }

	c.suppress = func(s string) (y bool) { _, y = c.defs[s]; return }

	if  _, y := vars["_"]; !y { vars["_"] = nil }
	for s, v := range vars {
		d, _ := c.set(&c, s, v)
		d.origin = defCodeBlockAuto
	}

	defer p.setbits(p.setbit(parseCodeBlock))

	for p.tok != EOF && p.pos < p.stop {
		if p.tok == SPACE || p.tok == LINEND || (p.tok == COMMENT && p.lineComment != nil) {
			p.next(true)
		} else {
			p.clause(&c)
		}
	}
}

func (p *parser) repeat(ctx Context, t *template, params []Value) {
	defer func(t time.Time, pos Pos, tok token, lit string, state scanstate) {
		if u := _universe(ctx); u.ddd == "template.repeat" {
			// dont check time in ddd mode
		} else if d := time.Now().Sub(t); d > u.slow {
            warnstack(ctx, 3, "slow: %v, prof-%d", d, pprofCounter).debug(1)
        }

		if _diagnostic(ctx).error() { erro(ctx, "template errors").debug(1) }

		p.pos, p.tok, p.lit, p.scanner.scanstate = pos, tok, lit, state
	} (time.Now(), p.pos, p.tok, p.lit, p.scanner.scanstate)

	// TODO: parseOpts(params) -> add option to turn off asFile in Context

	if false { pprofCounter += 1
		var (
			profCpu = fmt.Sprintf("template-%05d.cpu.prof", pprofCounter)
			profMem = fmt.Sprintf("template-%05d.mem.prof", pprofCounter)
			fCpu *os.File
			e error
		)
		if fCpu, e = os.Create(profCpu); e != nil {
			erro(ctx, "%T: %v", e, e).debug(1)
			return
		} else if e = pprof.StartCPUProfile(fCpu); e != nil {
			erro(ctx, "%v: %v", profCpu, e).debug(1)
			fCpu.Close() ; return
		}
		defer func() {
			pprof.StopCPUProfile()
			fCpu.Close()

			var fMem, e = os.Create(profMem)
			if e != nil { erro(ctx, "%v", e).debug(1) }

			runtime.GC() // update memory statistics
			e = pprof.WriteHeapProfile(fMem)
			fMem.Close()

			if e != nil { erro(ctx, "%v: %v", profMem, e).debug(1) }
		} ()
	}

	var m = map[string]Value{}

	for i, v := range t.params { if s := v.string(ctx); s != "" {
		if i < len(params) { m[s] = params[i] } else {
			m[s] = makeNull(v.Position())
		}
	} else {
		erro(at(ctx,v), "empty template param name: %v %v", v, v).debug(1)
	}}

	p.codeblock(ctx, t, m)
}

func (p *parser) call(ctx Context, name Value, args []Value) (result bool) {
	ctx = p.ctx(ctx)

	for _, t := range p.templates { if t.name != nil && eq(ctx, t.name, name) {
		stop := p.stop
		p.stop = t.endPos
		p.repeat(ctx, t, args)
		p.stop = stop
		return true
	}}

	erro(ctx, "undefined template: %v", name).debug(3)
	return
}

func (p *parser) clause(ctx Context) {
	if l_traverse.enabled { defer un(l_tracef(l_traverse, "clause(%v, %v)", p.tok, p.pos)) }

	var x Value
	var tok = p.tok // TODO: allow assigns like: `eval := xxx`

	defer trace(ctx)

	defer func() { if debugSyntax(ctx, "clause") {
		warn(ctx, "clause: %v %v ; %v %v", typeof(x), x, p.tok, p.lit).debug(6)
	}} ()

	switch p.spaces(); tok {
	case  INCLUDE: p.spec(ctx, tok, p.expect(tok), p.include); return
	case    FILES: p.spec(ctx, tok, p.expect(tok), p.files); return
	case   ASSERT: p.spec(ctx, tok, p.expect(tok), p.assert); return
	case   APPEND: p.spec(ctx, tok, p.expect(tok), p.append); return
	case     EVAL: p.spec(ctx, tok, p.expect(tok), p.eval); return
	case    COLON: p.specialRule(ctx); return
	case      DEF: p.def(ctx); return
	case      FOR: p.for_(ctx); return
	case  FOREACH: p.foreach(ctx); return
	case   LINEND, SPACE: p.next(true) // skip empty lines
	case USE, TEMPLATE:
		erro(ctx, "`%v` unexpected here", p.tok).debug(10)
		return
	}

	x = p.expr(ctx, true)

	if p.spaces(); p.tok.isAssign() {
		if debugSyntax(ctx, "define") {
			note(p, "parser.clause: %v; %v %v", us(x), p.tok, p.lit).debug(1)
			flush(ctx)
		}
		p.assign(ctx, x)
		return
	}

	if p.tok.isRuleDelim() {
		if debugSyntax(ctx, "rule") {
			note(p, "parser.clause: %v; %v %v", us(x), p.tok, p.lit).debug(1)
			flush(ctx)
		}
		p.rule(ctx, specialRuleNor, nil, []Value{x})
		return
	} else if a, y := x.(*argumented); y {
		p.call(ctx, a.Value, a.args)
		return
	}

	if vals := p.values(ctx, x); p.tok != EOF {
		return
	} else if strings.HasSuffix(p.scanner.file.Name(), pathSep+configuration_sm) {
		if false { note(ctx, "%v (kit=%s)", p.tok, p.lit).debug(1) }
	} else if p.bits&parseIncludingConf != 0 {
		note(ctx, "bad clause: %v (kit=%s) after %v", p.tok, p.lit, vals).debug(3)
	} else {
		erro(ctx, "bad clause: %v (lit=%s) after %v", p.tok, p.lit, vals).debug(10)
	}
}

func (p *parser) setDefaultVars(ctx Context, filename, abs, rel, tmp string) (res bool) {
	var s = ctx.scope()
	if s == nil {
		erro(ctx, "invalid scope").debug(1)
		return
	}

	var d *def

	if l := _loader(ctx); l.mode&Flat == 0 {
		var position = ctx.Position()

		d, _ = l.def(position, ".")
		d.set(ctx, defVoid, _pathstr(ctx, rel))

		d, _ = l.def(position, "/")
		d.set(ctx, defVoid, _pathstr(ctx, abs))

		d, _ = l.def(position, "CTD") // Current Temp Directory, TODO: make it $:ctd:
		d.set(ctx, defVoid, _pathstr(ctx, tmp))

		d, _ = l.def(position, "CWD") // Current Work Directory, TODO: make it $:cwd:
		d.set(ctx, defVoid, _pathstr(ctx, abs))

	} else if d = s.FindDef("/");   d == nil {
		erro(ctx, "/ not in the scope: %v", s.comment)
	} else if d = s.FindDef(".");   d == nil {
		erro(ctx, ". not in the scope: %v", s.comment)
	} else if d = s.FindDef("CTD"); d == nil {
		erro(ctx, "CTD not in the scope: %v", s.comment)
	} else if d = s.FindDef("CWD"); d == nil {
		erro(ctx, "CWD not in the scope: %v", s.comment)
	}

	return true
}

type projectDeclOpts struct {
	configure Value `conf,configure` // detects dotConfigure if empty
	noDock bool `nodock,no-dock` // don't load container project
    traveUseLoop bool `break,loop` // don't recursively use this project
    multiUseAllowed bool `multi`  // this project is used multiple times
	final bool `final` // no bases
}

func (p *parser) file(ctx Context) *parsedFile {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "file '"+p.scanner.file.Name()+"'")) }
	if _universe(ctx).traceLaunch { defer un(l_trace(l_launch, "parser.file")) }
	if _diagnostic(ctx).error() { return nil }

	defer trace(ctx)

	var (
		ident *barecomp
		identStr string
		implicitBase string // aka. foo.bar.Baz implicitly load base 'foo/bar'
		abs, rel, tmp string
		l = _loader(ctx)
		position = ctx.Position()
		keyword  = p.tok
		filename = p.scanner.file.Name()
		isMainFile = isEntryFileName(filename)
	)

	assert(l != nil, "nil loader")

	defer l.closeScope(l.openScope(fmt.Sprintf("file %s", filename)))

	if l.mode&Flat != 0 {
		abs = ctx.project().absPath
	} else {
		abs = filepath.Dir(filename)
	}

	rel, _ = filepath.Rel(_workdir(l), abs)
	tmp = joinTmpPath(ctx,_workdir(l), rel)

	if !p.setDefaultVars(ctx, filename, abs, rel, tmp) { return nil }

	switch position = p.Position(); keyword {
	case PACKAGE, MODULE:
		erro(ctx, "deprecated keyword: %s", keyword).debug(1)
		return nil
	case CONFIGURE:
		switch p.next(true); p.tok {
		case DOT:
			if err := l.ParseConfigDir(abs, abs); err != nil {
				erro(ctx, "parsing configure directory failed, '%s': %v", abs, err)
			} else {
				p.next(true) // skip the '.' token and consequence spaces
			}

			var basename = filepath.Base(filepath.Dir(filename))
			ident = makeBarecomp(makeBareword(position, basename))

		default:
			erro(ctx, "unknown configuration '%v', currently only 'configure .' is supported", p.tok)
		}
	case PROJECT:
		if l.mode&Flat != 0 { erro(ctx, "forbidden `%v` in flat file", p.tok) }

		p.next(true)

		var ( // Options are flag or *pair of a flag.
			opts projectDeclOpts
			optVals []Value
			pos Position
		)
		for p.tok == MINUS {
			var opt = p.expr(ctx);  p.spaces()
			optVals = append(optVals, opt)
			if !pos.IsValid() { pos = opt.Position() }
		}
		if !pos.IsValid() { pos = p.Position() }
		if a := parseOpts(ctx, &opts, optVals...); len(a) > 0 {
			for _, v := range a { erro(at(ctx,v), "unknown option %v", us(v)).debug(1) }
			return nil
		}

		var g = ctx.globe()
		var linfo = g.loads[len(g.loads)-1]

		// Smart-lang spec:
		//   * the project clause is not a declaration;
		//   * the project name does not appear in any scope.
		if p.tok == LPAREN || p.tok == EOF || p.tok == LINEND || p.lineComment != nil {
			var dir = filepath.Dir(filename)
			if linfo.loadee != nil && linfo.absDir == dir {
				ident = makeBarecomp(makeBareword(position, linfo.loadee.name))
			} else if name := filepath.Base(filename); name == dotBase || name == dotConfigure {
				// NOTE: loading the .base or .configure file
				ident = makeBarecomp(makeBareword(position, name))
			} else if base := filepath.Base(dir); base != "" {
				// TODO: validate basename as a valid identifier
				ident = makeBarecomp(makeBareword(position, base))
			} else {
				erro(ctx, "invalid file: %v", filename).debug(1)
			}
		} else if p.tok == TILDE {
			/*if filename == confinitFilename {
                ident = &ast.Bareword{ ValuePos:pos, Value:"~" }
            } else*/ if ext := filepath.Ext(filename); ext != ".smart" {
				erro(p, "`%v` not a smart file", filepath.Base(filename)).debug(1)
			} else if s := strings.TrimSuffix(filepath.Base(filename), ext); s != "" {
				ident = makeBarecomp(makeBareword(position, s))
			} else {
				erro(p, "`%v` not tilde name", filepath.Base(filename)).debug(1)
			}
			p.next(true) // skip tilde
		} else {
			base := makePath()
			ident = makeBarecomp()
			for p.tok != EOF && p.tok != SPACE {
				var w = p.bare(ctx, false)
				if ident = ident.suffix(ctx, w).(*barecomp); p.tok == DOT {
					t := &punctuation{valbase{p.Position()}, p.tok}
					// TODO: parse to Qualiword
					ident = ident.suffix(ctx, t).(*barecomp)
					base.elems = append(base.elems, w)
					p.step() // '.'
				} else { break }
			}
			if p.spaces(); len(ident.elems) == 0 {
				// erro(ctx, "package name is empty (tok=%v %v)", t, p.tok).debug(1)
				// return nil
			} else if 0 < base.len() {
				implicitBase = base.string(ctx)
			}
		}

		if identStr = ident.string(ctx); linfo.loadee != nil && identStr != linfo.loadee.name {
			warn(at(ctx,ident.Position()), "%s: declare multiple project in the same directory", ctx.project()).debug(24)
		} else if identStr == "_" && l.mode&DeclarationErrors != 0 {
			erro(at(ctx,ident.Position()), "package name '_' is preserved").debug(1)
			return nil
		}

		// Don't bother parsing the rest if we had errors parsing the package clause.
		if n := _diagnostic(l.Context).countError(); n > 0 {
			erro(p, "got %d errors parsing file: %s", filename).debug(1)
			return nil
		}

		var _, declared = linfo.declares[identStr]
		if (l.mode&Flat == 0) && l.declare(at(ctx, ident.Position()), keyword, ident, identStr, &opts) {
			// Change the 'default' owners into the new declared project
			if s := ctx.scope(); s != nil {
				if def := s.FindDef("."  ); def != nil { def.owner_ = ctx.project() }
				if def := s.FindDef("/"  ); def != nil { def.owner_ = ctx.project() }
				if def := s.FindDef("CTD"); def != nil { def.owner_ = ctx.project() }
				if def := s.FindDef("CWD"); def != nil { def.owner_ = ctx.project() }
			} else {
				erro(ctx, "file scope is nil").debug(1)
			}
			// NOTE: do.smart is always the first loaded, so the loadee will be pointed to it
			if linfo.loadee == nil { linfo.loadee = ctx.project() }
			defer l.closeCurrent(ident, identStr)
			isMainFile = isMainFile && !declared;
		}

		var basePos Position
		if implicitBase != "" { basePos = pos } else { basePos = p.Position() }
		if p.tok == LPAREN {
			var bits = p.setbit(parseGroup)
			for p.tok != EOF {
				for p.next(true); !p.isEndOfList(false); {
					p.spaces()
					param := p.expr(ctx)
					p.spaces()

					//if p.lineComment != nil  { break }
					//if p.tok == LINEND { break }
					if p.tok == EOF {
						erro(at(ctx,basePos), "unexpected end of file while parsing bases").debug(1)
						p.setbits(bits) ; return nil
					}

					var (
						ctx = at(ctx, param.Position())
						t = parseOpts(ctx, &opts, param)
					)
					if keyword == PACKAGE || opts.final {
						// No bases for PACKAGE or final project
					} else if !l.bases(ctx, linfo, "", merge(t...)...) {
						errostack(at(ctx,param), 3, "loading base '%v' failed", t).debug(10)
						p.setbits(bits) ; return nil
					}
				}
				if p.tok != COMMA { break }
			}
			p.setbits(bits)
			p.expect(RPAREN)
		} else if !l.bases(ctx, linfo, implicitBase) { // for special bases, e.g. .base
			erro(at(ctx,basePos), "loading bases failed").debug(1)
			return nil
		}

		if p.spaces(); p.tok != EOF { p.linend() }
		if keyword != PACKAGE {
			l.configure(ctx, linfo, ident, identStr, declared)
			if !opts.noDock { l.container(ctx, ident, identStr) }
		}
	case EOF:
		return nil
	default:
		if l.mode&Flat == 0 {
			p.expected(p.pos, "configure, project, module or package keyword")
		}
	}

	var auto = (l.mode&Flat == 0) && isMainFile //&& isEntryFileName(filename)
	if auto { l.autoload(p.ctx(ctx), "declared") }
	if l.mode&ModuleClauseOnly == 0 {
		if l.mode&Flat == 0 { ForDeclare: for p.tok != EOF {
			switch tok := p.tok; tok {
			case USE: p.spec(ctx, tok, p.expect(tok), p.use)
			case LINEND, SPACE: p.next(true) // skip empty lines
			case ASSERT, EVAL, FILES, INCLUDE: p.clause(ctx)
			default: break ForDeclare
			}
		}}

		if false && auto { l.autoload(p.ctx(ctx), "amid") }

		if l.mode&ImportsOnly == 0 { // rest of module body
			for /* !_diagnostic(p.Context).error() && */ p.tok != EOF {
				if p.tok == LINEND || (p.tok == COMMENT && p.lineComment != nil) {
					p.next(true)
				} else if p.clause(p.ctx(ctx)); flush(ctx) > 0 {
					break
				}
			}
		}
	}
	if auto { l.autoload(p.ctx(ctx), "appendix") }

	if  _universe(ctx).ddd == "parser.files" {
		_universe(ctx).ddd = ""
	}

	return &parsedFile{
		// TODO: doc: doc,
		// TODO: comments: p.comments,
		keyword:  keyword,
		position: position,
		name:     ident,
		scope:    ctx.scope(),
		use:      p.imports,
	}
}
