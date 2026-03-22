//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "strings"
	"sync"
    "fmt"
)

// =============================================================================
// 1. Core Data Structures & Interning
// =============================================================================

const mapThreshold = 16

var (
	// Global Wildcard Constants (Already "interned" by being constants)
	WildcardOne   = "*"
	WildcardAny   = "**"
	WildcardShort = "*?"
	WildcardChar  = "?"

	// Intern Cache: Maps string content to the canonical string header
	internPool = make(map[string]string)
	internM sync.RWMutex
)

func intern(s string) string {
	// 1. Optimization: Return constants for wildcards immediately
	switch s {
	case "*":  return WildcardOne
	case "**": return WildcardAny
	case "*?": return WildcardShort
	case "?":  return WildcardChar
	}

	// 2. Fast Path: Read Lock
	internM.RLock()
	interned, ok := internPool[s]
	internM.RUnlock()
	if ok { return interned }

	// 3. Slow Path: Write Lock
	internM.Lock()
	// Double check to prevent race conditions
	if interned, ok = internPool[s]; ok {
		internM.Unlock()
		return interned
	}
	internPool[s] = s
	internM.Unlock()
	return s
}

func internBytes(b []byte) string {
	// Optional: Micro-optimize single-byte/common sequences to bypass the lock entirely
	switch len(b) {
	case 1:
		switch b[0] {
		case '*': return WildcardOne
		case '?': return WildcardChar
		}
	case 2:
		if b[0] == '*' { switch b[1] {
		case '*': return WildcardAny
		case '?': return WildcardShort
		}}
	}

	// 1. Zero-allocation fast-path lookup
	internM.RLock()
	interned, ok := internPool[string(b)]
	internM.RUnlock()
	if ok { return interned }

	// 2. Slow-path allocation
	s := string(b)
	internM.Lock()
	// Double check to prevent race conditions
	if interned, ok = internPool[s]; ok {
		internM.Unlock()
		return interned
	}
	internPool[s] = s
	internM.Unlock()
	return s
}

// nodeEntry preserves order
type nodeEntry struct {
	k string
	v *valcache
}

// valcache: Hybrid Trie Node (Slice 'o' for order/compactness, Map 'v' for speed)
type valcache struct {
	a []any                // Payload: Rules, Filemaps
	o []nodeEntry          // Primary Storage: Compact & Ordered
	v map[string]*valcache // Acceleration Index: Created only when len(o) >= mapThreshold
}

func (p *valcache) String() (s string) { // NOTE: for debug
	for i, a := range p.a {
		var t string
		switch v := a.(type) {
		case filemap: t = v.String()
		case *rule: t = v.target.String()
		}
		s += fmt.Sprintf("%d:%s,", i, t)
	}
	for _, n := range p.o {
		c, _ := p.get(n.k)
		s += fmt.Sprintf("%v:%v,", n.k, c)
	}
	if s != "" { s = s[:len(s)-1] } // aka. TrimSuffix(s, ",")
	return "{"+s+"}"
}

// =============================================================================
// 2. Valcache Methods
// =============================================================================

func (c *valcache) get(key string) (*valcache, bool) {
	if c.v != nil {
		n, ok := c.v[key]
		return n, ok
	}
	for i := range c.o {
		if c.o[i].k == key {
			return c.o[i].v, true
		}
	}
	return nil, false
}

func (c *valcache) add(rawKey string) *valcache {
	// Intern the key to deduplicate memory
	key := intern(rawKey)

	if n, ok := c.get(key); ok {
		return n
	}

	child := new(valcache)
	c.o = append(c.o, nodeEntry{k: key, v: child})

	if len(c.o) == mapThreshold {
		c.v = make(map[string]*valcache, len(c.o))
		for _, entry := range c.o {
			c.v[entry.k] = entry.v
		}
	} else if len(c.o) > mapThreshold {
		c.v[key] = child
	}

	return child
}

// =============================================================================
// 3. Path Processing
// =============================================================================

// tokenizePaths parse a path pattern (Extended Glob) into tokens
func tokenizePaths(path string) (results [][][]string) {
	// Expand braces first: foo/{a,b} -> [foo/a, foo/b]
	for _, p := range expandBraces(path) {
		results = append(results, tokenizePath(p))
	}
	return
}

// tokenizePath convert a path into tokens
func tokenizePath(path string) [][]string {
	return tokenizeSegments(strings.Split(path, "/"))
}

func tokenizeSegments(parts []string) [][]string {
	ss := make([][]string, len(parts))

	for i, part := range parts {
		// Optimization: If no meta-chars, intern the whole segment
		if !strings.ContainsAny(part, "*?.[") {
			ss[i] = []string{intern(part)}
			continue
		}

		var tokens []string
		start := 0
		
		for j := 0; j < len(part); {
			c := part[j]
			if c == '*' || c == '?' || c == '.' || c == '[' {
				// Flush preceding literal
				if j > start {
					tokens = append(tokens, intern(part[start:j]))
				}

				if c == '*' {
					// Check for ** or *?
					if j+1 < len(part) {
						if part[j+1] == '*' {
							tokens = append(tokens, WildcardAny)
							j += 2; start = j; continue
						} else if part[j+1] == '?' {
							tokens = append(tokens, WildcardShort)
							j += 2; start = j; continue
						}
					}
					tokens = append(tokens, WildcardOne)
					j++
				} else if c == '[' {
					// Capture [a-z] as one token
					end := strings.IndexByte(part[j:], ']')
					if end != -1 {
						// Intern the whole set "[a-z]"
						tokens = append(tokens, intern(part[j:j+end+1]))
						j += end + 1
					} else {
						tokens = append(tokens, "[")
						j++
					}
				} else {
					// Capture . and ? -- Dots and Qmarks
					tokens = append(tokens, intern(string(c)))
					j++
				}
				start = j
			} else {
				j++
			}
		}
		// Flush trailing literal
		if start < len(part) {
			tokens = append(tokens, intern(part[start:]))
		}
		ss[i] = tokens
	}
	return ss
}

// expandBraces Recursive Brace Expander (One-pass)
func expandBraces(text string) []string {
	res, _ := expandBracesAt(text, 0)
	return res
}

// expandBracesAt is the recursive core
// It returns the list of expanded strings found at this level, and the index where it stopped.
func expandBracesAt(s string, idx int) ([]string, int) {
	var parts []string // The comma-separated options at this level

	// We need to track the "current working string" for the current comma-option.
	// However, because we might encounter a nested brace {a,b} inside an option,
	// we actually need a list of "current prefixes" that we are building.
	// Let's simplify: 
	// The standard way to do this recursively is to parse the *structure* first, 
	// then generate the combinations. 
	// But to do it in one pass as you asked:

	// Actually, the logic "prefix + middles[n] + suffix" is slightly complex 
	// to do purely linearly because 'suffix' hasn't been parsed yet.
	
	// Better approach for "One Pass":
	// 1. Scan until '{', ',', or '}'.
	// 2. If '{': Recurse. Get [m1, m2]. Cartesian product with current prefixes.
	// 3. If ',': Finish current set of strings, start new set.
	// 4. If '}': Return.

	currentSet := []string{""} // Start with one empty prefix

	i := idx
	for i < len(s) {
		char := s[i]

		if char == '{' {
			// Recursion: parse the content inside {...}
			middles, newIdx := expandBracesAt(s, i+1)
			i = newIdx // Advance to after the matching '}'

			// Cartesian Product: append each middle to each current prefix
			var nextSet []string
			for _, prefix := range currentSet {
				for _, mid := range middles {
					nextSet = append(nextSet, prefix+mid)
				}
			}
			currentSet = nextSet

		} else if char == '}' {
			// Found closing brace for THIS level.
			// We are done with this specific brace block.
			// Return our results and the current index (to let caller continue)
			return combine(parts, currentSet), i
			
		} else if char == ',' {
			// Found a comma at THIS level.
			// 1. Commit currentSet to parts.
			parts = combine(parts, currentSet)
			// 2. Reset currentSet for the next option
			currentSet = []string{""}
			
		} else {
			// Literal character
			// Append char to all strings in currentSet
			for k := range currentSet {
				currentSet[k] += string(char)
			}
		}
		i++
	}

	// End of string reached (implicit closing brace)
	return combine(parts, currentSet), i
}

// Helper to merge the final set into the results
func combine(existing []string, current []string) []string {
	if len(current) == 0 { return existing }
	return append(existing, current...)
}

// =============================================================================
// 4. Cache & Uncache Logic
// =============================================================================

func cache(ctx Context, c *valcache, ss [][]string) *valcache {
	for _, segment := range ss {
		for _, token := range segment {
			c = c.add(token)
		}
	}
	return c
}

func uncache(ctx Context, root *valcache, ss [][]string) (r []*valcache) {
	seen := make(map[*valcache]bool)
	seenShadows := make(map[*valcache]struct{})
	seenAmpNodes := make(map[*valcache]struct{}) // <--- NEW: Query-level & edge lock
	var shadowsToSearch []*valcache // <--- NEW: Queue to defer shadow searches

	fullvalue := do(ctx, fullvalue{}).(Value)
	fullmatch := func(c *valcache) (res bool) {
		if full, exists := seen[c]; exists { return full }
		if res = c.matchPayload(ctx, fullvalue); res { r = append(r, c) }
		seen[c] = res
		return 
	}

	const priorDynamicClosureShadowing = true

	var f0 func(*valcache, [][]string, int, int, int) bool
	f0 = func(c *valcache, ss [][]string, i, j, k int) (found bool) {
		if c == nil { return false }

		// 1. Success Condition
		if i == len(ss) { return fullmatch(c) }

		// 2. Segment Boundary
		if j == len(ss[i]) { return f0(c, ss, i+1, 0, 0) }

		s := ss[i][j]

		// 3. Token Boundary
		if k == len(s) { return f0(c, ss, i, j+1, 0) }

		// ---------------------------------------------------------------------
		// DYNAMIC CLOSURE SHADOWING
		// Must be evaluated before Input Wildcards to survive early returns!
		// ---------------------------------------------------------------------
		var doDynamicClosureShadowing func()
		if x, y := c.get("&"); !y { doDynamicClosureShadowing = func() {} } else {
			doDynamicClosureShadowing = func() {
				// CRITICAL FIX: Lock the specific '&' edge!
				// This guarantees that even if ** forces f0 to revisit this node 
				// with a new token state, we never double-queue its shadow trie!
				if _, ok := seenAmpNodes[x]; ok { return } else { seenAmpNodes[x] = struct{}{} }

				if proj := _project(ctx); proj != nil {
					for _, payload := range x.a {
						if shadow := proj.shadow(ctx, payload); shadow != nil {
							if _, ok := seenShadows[shadow]; !ok { 
								seenShadows[shadow] = struct{}{}
								shadowsToSearch = append(shadowsToSearch, shadow)
							}
						}
					}
				}
			}
		}
		if priorDynamicClosureShadowing { doDynamicClosureShadowing() }

		// 4. Input Wildcard
		switch s {
		case WildcardOne: // "*"
			for _, entry := range c.o {
				if f0(entry.v, ss, i, j, 0) { found = true } // Greedy Consume (Trie)
			}
			if f0(c, ss, i, j+1, 0) { found = true } // Stop (Consume Input)
			return found

		case WildcardShort: // "*?"
			if f0(c, ss, i, j+1, 0) { found = true } // Stop (Non-Greedy)
			for _, entry := range c.o {
				if f0(entry.v, ss, i, j, 0) { found = true } // Continue
			}
			return found

		case WildcardAny: // "**"
			for _, entry := range c.o {
				if f0(entry.v, ss, i, j, 0) { found = true } // Greedy Consume
			}
			if f0(c, ss, i, j+1, 0) { found = true } // Stop
			return found

		case WildcardChar: // "?"
			for _, entry := range c.o {
				// Match Literal or Set
				if (len(entry.k) == 1 && !isWildcardMeta(entry.k)) || (len(entry.k) > 2 && entry.k[0] == '[') {
					if f0(entry.v, ss, i, j+1, 0) { found = true }
				}
				// Match Trie Wildcards (*, **, *?)
				if isWildcardMeta(entry.k) {
					if f0(entry.v, ss, i, j+1, 0) { found = true } // Stop
					if f0(c, ss, i, j+1, 0) { found = true }       // Continue
				}
			}
			return found
		}

		// ---------------------------------------------------------------------
		// 5. TRIE WILDCARD LOGIC
		// ---------------------------------------------------------------------

		// A. Compressed Node Match
		if k == 0 && (len(ss[i]) > 1 || s == WildcardChar) {
			for _, entry := range c.o {
				if isWildcardMeta(entry.k) { continue }
				if n, ok := consumeCompressed(entry.k, ss[i][j:]); ok {
					if f0(entry.v, ss, i, j+n, 0) { found = true }
				}
			}
		}

		// B. Literal / Prefix Match
		if k < len(s) {
			for _, entry := range c.o {
				key := entry.k
				if isWildcardMeta(key) || key == "?" { continue }

				if len(key) > 2 && key[0] == '[' {
					if matchCharSet(key, s[k]) {
						if f0(entry.v, ss, i, j, k+1) { found = true }
					}
				} else if len(key) > 0 { // Literal match
					if strings.HasPrefix(s[k:], key) {
						// Advance k by the length of the matched prefix.
						// The next recursive call will safely hit '3. Token Boundary' if k == len(s)!
						if f0(entry.v, ss, i, j, k+len(key)) { found = true }
					}
				}
			}
		}

		// (C. Hybrid Token Match has been safely superseded and optimized into B)

		// D. Trie Wildcards
		
		if x, y := c.get("?"); y {
			if f0(x, ss, i, j, k+1) { found = true }
		}
		
		// Handle "*" (WildcardOne) in Trie
		if x, y := c.get("*"); y {
			if f0(x, ss, i, j, k) { found = true }          // Transition (Match 0)
			
			nextJ, nextK := j, k+1
			if nextK == len(ss[i][nextJ]) { nextJ++; nextK = 0 }
			
			if nextJ < len(ss[i]) {
				if f0(c, ss, i, nextJ, nextK) { found = true } // Consume (Match 1+)
			} else {
				if f0(x, ss, i, nextJ, nextK) { found = true } // End of Segment
			}
		}

		// Handle "*?" (WildcardShort) in Trie
		if x, y := c.get("*?"); y {
			if f0(x, ss, i, j, k) { found = true }          // Transition (Match 0) - Prioritized

			nextJ, nextK := j, k+1
			if nextK == len(ss[i][nextJ]) { nextJ++; nextK = 0 }
			
			if f0(c, ss, i, nextJ, nextK) { found = true }  // Consume (Match 1+)
		}

		// Handle "**" (WildcardAny) in Trie
		if x, y := c.get("**"); y {
			if f0(c, ss, i+1, 0, 0) { found = true }     // Consume Segment - Prioritized
			if f0(x, ss, i, j, k) { found = true }       // Transition (Match 0)
		}

		// ---------------------------------------------------------------------
		// DYNAMIC CLOSURE SHADOWING
		// ---------------------------------------------------------------------
		if !priorDynamicClosureShadowing { doDynamicClosureShadowing() }
		return found
	}

	// 1. Traverse the Static Trie and collect all static matches
	f0(root, ss, 0, 0, 0)

	// 2. Flush the queue: Traverse the Discovered Shadow Tries
	// This guarantees dynamic matches are appended securely at the END of the array!
	for _, shadow := range shadowsToSearch {
		f0(shadow, ss, 0, 0, 0)
	}

	return
}

func isWildcardMeta(k string) (res bool) {
	switch k { case WildcardAny, WildcardOne, WildcardShort: res = true }
	return
}

// Returns number of tokens consumed (n) and success (ok).
func consumeCompressed(nodeKey string, tokens []string) (int, bool) {
	keyIdx, tokIdx := 0, 0
	for keyIdx < len(nodeKey) {
		if tokIdx >= len(tokens) { return 0, false } // Not enough tokens
		t := tokens[tokIdx]
		
		// If input has complex wildcards, abort optimization (let recursion handle it)
		if t == WildcardAny || t == WildcardOne { return 0, false }

		if t == WildcardChar { // "?"
			keyIdx++ // Consumes 1 char of nodeKey
			tokIdx++ // Consumes 1 token
			continue
		}
		
		// Literal Match (e.g. t="z" matches nodeKey="zz" at index 0)
		if strings.HasPrefix(nodeKey[keyIdx:], t) {
			keyIdx += len(t)
			tokIdx++
		} else {
			return 0, false
		}
	}
	// Must consume exactly the whole nodeKey
	return tokIdx, true
}

func matchCharSet(pattern string, char byte) bool {
	// Simplified parser for [a-z0-9]
	inner := pattern[1 : len(pattern)-1]
	for i := 0; i < len(inner); i++ {
		if i+2 < len(inner) && inner[i+1] == '-' {
			start, end := inner[i], inner[i+2]
			if char >= start && char <= end { return true }
			i += 2
		} else if inner[i] == char {
			return true
		}
	}
	return false
}

func canStartMatch(c *valcache, segment []string) bool {
	if len(segment) == 0 { return false }
	firstChar := segment[0]
	
	// Check exact match (literal or whole token)
	if _, ok := c.get(firstChar); ok { return true }
	if len(firstChar) > 0 {
		if _, ok := c.get(firstChar[:1]); ok { return true }
	}
	
	// Check wildcard/meta match
	if _, ok := c.get("?"); ok { return true }
	if _, ok := c.get("*"); ok { return true }
	
	for _, entry := range c.o {
		if entry.k[0] == '[' && matchCharSet(entry.k, firstChar[0]) {
			return true
		}
	}
	return false
}

// =============================================================================

type matched_filemap struct{ filemap ; value Value }
type matched_rule struct{ *rule ; value Value }
func (t matched_filemap) String() string {
	s1, s2 := t.filemap.String(), t.value.String()
	if s1 == s2 { return "{=matched_filemap "+s1+"}" }
	return "{=matched_filemap "+s1+" name="+s2+"}"
}
func (t matched_rule) String() string {
	s1, s2 := t.rule.String(), t.value.String()
	if s1 == s2 { return "{=matched_rule "+s1+"}" }
	return "{=matched_rule "+s1+" name="+s2+"}"
}

func (p *valcache) matchPayload(ctx Context, fullvalue Value) (ok bool) {
	for _, a := range p.a {
		switch a := a.(type) {
		case filemap:
			// Ensure matched == true AND there is no unconsumed remainder
			if matched, res, rem, _ := match(ctx, a.pattern, fullvalue); matched && rem == nil {
				var a = filemap{a._filemap, a.pattern}
				if 0 < do(ctx, matched_filemap{a, res}).(int) {
					ok = true
				} else {
					debug(ctx, "%v %v", ts(a), res, callstack{num:10}, trace{})
				}
			}
		case *rule:
			// Ensure matched == true AND there is no unconsumed remainder
			if matched, res, rem, _ := match(ctx, a.target, fullvalue); matched && rem == nil {
				var a = &rule{a.target, a.arged, a.program}
				if 0 < do(ctx, matched_rule{a, res}).(int) {
					ok = true
				} else {
					debug(ctx, "%v %v", ts(a), res, callstack{num:10}, trace{})
				}
			}
		default:
			debug(ctx, "%v", ts(a), callstack{num:10}, trace{})
		}
	}
	return
}

func (p *valcache) ks() string {
	var ks []string
	for _, n := range p.o { ks = append(ks, n.k) }
	return "[" + strings.Join(ks, " ") + "]"
}
func (p *valcache) k(s []string, j int) (string, bool) {
	for _, o := range p.o {
		var i, l = 0, len(o.k)
		for n := j; n < len(s); n += 1 {
			if t := s[n]; t == "?" {
				i += 1
			} else if i < l && strings.HasPrefix(o.k[i:], t) {
				i += len(t)
			} else {
				break
			}
		}
		if i == l { return o.k, true }
	}
	return "", false
}

type hit_segs struct{ *valcache ; s [][]string }
type fullvalue struct{}
type fullctx struct{ Context ; any }
func (p *fullctx) do(ctx Context, op any) any {
	switch op.(type) {
	case fullvalue: return p.any
	}
	return p.Context.do(ctx, op)
}

// toks handles pure string splitting. We intercept the string here BEFORE it 
// hits tokenizeSegments, ensuring "**.c" natively compiles as "**/*.c".
func toks(ctx Context, c *valcache, segs ...string) hit_segs {
	var norm []string
	for _, seg := range segs {
		// Option 1: Normalize intra-string path closures
		if strings.Contains(seg, "**.") {
			norm = append(norm, strings.ReplaceAll(seg, "**.", "**/*."))
		} else {
			norm = append(norm, seg)
		}
	}
	return hit_segs{c, tokenizeSegments(norm)}
}

// tokc delegates to toks, so it automatically inherits the normalization
// applied above when prefix is evaluated.
func tokc(ctx Context, c *valcache, comp *compound) hit_segs {
	var prefix string
	for _, e := range comp.elems {
		if _, isClosure := unbox(e).(*closure); isClosure {
			var ss [][]string
			if prefix != "" { 
				ss = append(ss, toks(ctx, c, prefix).s...) 
			}
			ss = append(ss, []string{"&"})
			return hit_segs{c, ss} // Mathematical halt!
		}
		prefix += __string(ctx, e)
	}
	if prefix != "" { return toks(ctx, c, prefix) }
	return hit_segs{c, nil}
}

// UPGRADED: tokg builds the token array directly. We use a lookahead (peek) 
// to dynamically inject a '*' node if '**' is followed by a literal '.'.
func tokg(ctx Context, c *valcache, g *globpat) hit_segs {
	var s []string
	if false && checkpoints { defer func() {
		if t := tokenizePath(__string(ctx, g)); sf("[%s]",s) != sf("%s",t) {
			debug(ctx, "%s ⇒ [%s] != %v | %v", g, s, t, c, trace{})
		}
	}()}
	
	for i, e := range g.elems {
		if _, isClosure := unbox(e).(*closure); isClosure {
			var ss [][]string
			if len(s) > 0 { ss = append(ss, s) }
			ss = append(ss, []string{"&"})
			return hit_segs{c, ss} // Mathematical halt!
		}
		
		str := __string(ctx, e)
		
		// Fallback for un-split strings inside glob elements
		if strings.Contains(str, "**.") {
			str = strings.ReplaceAll(str, "**.", "**/*.")
		}
		
		s = append(s, str) 
		
		// Option 1: Inter-token lookahead injection
		// If current node is "**" and the NEXT node is ".", we implicitly 
		// insert a "*" node into the slice to consume the file base name.
		if str == "**" && i+1 < len(g.elems) {
			nextStr := __string(ctx, g.elems[i+1])
			if nextStr == "." || strings.HasPrefix(nextStr, ".") {
				s = append(s, "*") // Inject implicit file-base consumer
			}
		}
	}
	return hit_segs{c, [][]string{s}}
}

// tokp natively delegates to tokc/tokg/toks, so it requires no normalization logic of its own.
func tokp(ctx Context, c *valcache, p *path) hit_segs {
	var ss [][]string
	if false && checkpoints { defer func() {
		if t := tokenizePath(__string(ctx, p)); sf("%s",ss) != sf("%s",t) {
			debug(ctx, "%v: %v != %v", p, ss, t, trace{})
		}
	}()}
	
	for _, e := range p.elems {
		switch t := unbox(e).(type) {
		case *closure:
			ss = append(ss, []string{"&"})
			return hit_segs{c, ss} 
			
		case *compound:
			res := tokc(ctx, c, t)
			ss = append(ss, res.s...)
			if len(res.s) > 0 && len(res.s[len(res.s)-1]) == 1 && res.s[len(res.s)-1][0] == "&" {
				return hit_segs{c, ss}
			}
			
		case *globpat: 
			res := tokg(ctx, c, t)
			ss = append(ss, res.s...)
			if len(res.s) > 0 && len(res.s[len(res.s)-1]) == 1 && res.s[len(res.s)-1][0] == "&" {
				return hit_segs{c, ss}
			}
			
		default: 
			ss = append(ss, toks(ctx, c, __string(ctx, t)).s...)
		}
	}
	return hit_segs{c, ss}
}

func _hit(ctx Context, c *valcache, k Value) (r []*valcache) {
	if checkpoints { defer func(s string) {
		if  truly(ctx, propCache)   { check_cache(ctx, k, s, c, r) }
		if  truly(ctx, propUncache) { check_uncache(ctx, k, s, c, r) }
		if !truly(ctx, propCache|propUncache) { debug(ctx, "%v %v", k, c, callstack{num:10}, trace{}) }
	}(c.String())}
	
	switch t := k.(type) {
	case *argumented: return _hit(ctx, c, t.Value)
	case *loc       : return _hit(ctx, c, t.Value)
	case *rule      : return _hit(ctx, c, t.target)
	case *closure   : return do_hit(&fullctx{ctx,t}, toks(ctx, c, "&")) // Return directly to avoid duplication
	case *compound  : r = do_hit(&fullctx{ctx,t}, tokc(ctx, c, t))
	case *globpat   : r = do_hit(&fullctx{ctx,t}, tokg(ctx, c, t))
	case *path      : r = do_hit(&fullctx{ctx,t}, tokp(ctx, c, t))
	case *percpat   : r = do_hit(&fullctx{ctx,t}, toks(ctx, c))
	case *regexpat  : r = do_hit(&fullctx{ctx,t}, toks(ctx, c))
	case *strval    : r = do_hit(&fullctx{ctx,t}, toks(ctx, c, `{`+__string(ctx,t)+`}`))
	case *strlit    : r = do_hit(&fullctx{ctx,t}, toks(ctx, c, `'`+__string(ctx,t.s)+`'`))
	case *strcomp   : r = do_hit(&fullctx{ctx,t}, toks(ctx, c, `"`+__string(ctx,t)+`"`))
	default:
		segs := strings.Split(__string(ctx, k), pathSep)
		r = do_hit(&fullctx{ctx,t}, toks(ctx, c, segs...))
	}

	// Universal Closure Inclusion: For any non-recursive query, unconditionally 
	// fetch the dynamic closures branch ("&") and append it to the results.
	if false { r = append(r, do_hit(&fullctx{ctx, k}, toks(ctx, c, "&"))...) }
	return r
}
func do_hit(c Context, a any) (r []*valcache) { r, _ = do(c,a).([]*valcache); return }
func hit(ctx Context, c *valcache, k Value) []*valcache { return _hit(ctx, c, k) }

type   cache_t struct{ Context }
type uncache_t struct{ Context ; a []any }

func (c cache_t) do(ctx Context, op any) any {
	switch t := op.(type) {
	case property: if t&propCache != 0 { return true }
	case hit_segs: return []*valcache{cache(ctx, t.valcache, t.s)}
	}
	return c.Context.do(ctx, op)
}

func (u *uncache_t) do(ctx Context, op any) (res any) {
	switch t := op.(type) {
	case property: if t&propUncache != 0 { return true }
	case hit_segs: return uncache(ctx, t.valcache, t.s)
	case matched_filemap, matched_rule:
		u.a = append(u.a, t); return len(u.a)
	}
	return u.Context.do(ctx, op)
}

func map_files(ctx Context, p *project, patts, paths []Value) (res []filemap) {
	var base = &_filemap{p, patts, paths}
	for _, patt := range patts {
		switch patt.(type) {
		case *valbase, *null, *none:
			continue
		}
		if c := hit(cache_t{ctx}, &p.filemap, patt); c != nil {
			switch len(c) {
			case 1:
				c0, f := c[0], filemap{base, patt}
				c0.a, res = append(c0.a, f), append(res, f)
			default:
				debug(pc(ctx,patt), "too many cached: %v %v", ts(patt), c, trace{})
			}
		} else {
			debug(pc(ctx,patt), "cache failed: %v", ts(patt), trace{})
		}
    }
    return
}

func map_entry(ctx Context, p *project, target Value, prog *program) (entry entry) {
	var patterned = patterned(ctx, target)
	if !patterned {
		switch target.(type) {
		case *barefile, *file, *path, *percpat, *globpat, *regexpat:
			goto skip_file
		}
		if t := p.file(ctx, target); t != nil {
			target, t.position = t, target.Position()
		}
	skip_file:
	}

    var args []Value // e.g. for pattern filtering
	switch t := target.(type) {
	case *argumented: target, args = t.Value, merge(t.args...)
	case *group: debug(ctx, "not supported target: %v", t, trace{})
	}

	if c := hit(cache_t{ctx}, &p.entries, target); c == nil {
		debug(ctx, "uncachable for: %v | %s", target, ts(target), trace{})
	} else {
		switch len(c) {
		case 1:
			r := &rule{target:target, arged:args, program:[]*program{prog}}
			if patterned { p.patterns = append(p.patterns, r) }
			if p.main == nil { p.main = r }
			c[0].a = append(c[0].a, r)
			entry = r
		default:
			debug(pc(ctx,target), "too many cached: %v %v", ts(target), c, trace{})
		}
	}
	return
}

func unmap[T any](ctx Context, c *valcache, key any) (res []T) {
	var u = &uncache_t{ctx, nil}
	var k Value

	if v, ok := key.(Value); ok { 
		k = v 
	} else {
		str := __string(ctx, key)
		// Emulate the parser: if the raw string is a path, pack it into a *path AST node.
		if strings.Contains(str, pathSep) {
			if segs := splitPathStr(ctx, str); len(segs) > 1 {
				k = packPath(segs)
			} else {
				k = _raw(_position(ctx), str)
			}
		} else {
			k = _raw(_position(ctx), str)
		}
	}
	
	var x = hit(u, c, k)
	for _, a := range u.a {
		switch t := a.(type) {
		case T: 
			res = append(res, t)
		default: 
			debug(ctx, "%v %v", ts(key), ts(a), trace{})
		}
	}
	if checkpoints { check_unmap(u, key, c, x) }
	return
}

func unmap_entries(ctx Context, p *project, key any, m map[*project]struct{}) (res []entry) {
	if false && checkpoints { defer check_unmap_entries(ctx, p, key, &res) }
	if m == nil { m = map[*project]struct{}{} } else if _, y := m[p]; y { return }
	if m != nil { m[p] = struct{}{} }
	if res = unmap[entry](ctx, &p.entries, key); res != nil { return }
	for _, b := range p.bases {
		if res = unmap_entries(ctx, b, key, m); res != nil { return }
	}
	if c := p.configure; false && c != nil {
		if res = unmap_entries(ctx, c, key, m); res != nil { return }
	}
	return
}

func unmap_files(ctx Context, p *project, key any, m map[*project]struct{}) (res []matched_filemap) {
	if false && checkpoints { defer check_unmap_files(ctx, p, key, &res) }
	if m == nil { m = map[*project]struct{}{} } else if _, y := m[p]; y { return }
	if m != nil { m[p] = struct{}{} }
	if res = unmap[matched_filemap](ctx, &p.filemap, key); res != nil { return }
	for _, b := range p.bases {
		if res = unmap_files(ctx, b, key, m); res != nil { return }
	}
	if c := p.configure; c != nil {
		if res = unmap_files(ctx, c, key, m); res != nil { return }
	}
	return
}
