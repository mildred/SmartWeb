package nquads

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const end_symbol rune = 4

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	rulenquadsDoc
	rulestatement
	rulesubject
	rulepredicate
	ruleobject
	rulegraphLabel
	ruleliteral
	ruleLANGTAG
	ruleEOL
	ruleIRIREF
	ruleSTRING_LITERAL_QUOTE
	ruleBLANK_NODE_LABEL
	ruleUCHAR
	ruleECHAR
	ruleHEX
	rulespc
	rulews
	rulecomment
	ruleAction0
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	rulePegText
	ruleAction5
	ruleAction6
	ruleAction7
	ruleAction8

	rulePre_
	rule_In_
	rule_Suf
)

var rul3s = [...]string{
	"Unknown",
	"nquadsDoc",
	"statement",
	"subject",
	"predicate",
	"object",
	"graphLabel",
	"literal",
	"LANGTAG",
	"EOL",
	"IRIREF",
	"STRING_LITERAL_QUOTE",
	"BLANK_NODE_LABEL",
	"UCHAR",
	"ECHAR",
	"HEX",
	"spc",
	"ws",
	"comment",
	"Action0",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"PegText",
	"Action5",
	"Action6",
	"Action7",
	"Action8",

	"Pre_",
	"_In_",
	"_Suf",
}

type tokenTree interface {
	Print()
	PrintSyntax()
	PrintSyntaxTree(buffer string)
	Add(rule pegRule, begin, end, next, depth int)
	Expand(index int) tokenTree
	Tokens() <-chan token32
	AST() *node32
	Error() []token32
	trim(length int)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(depth int, buffer string) {
	for node != nil {
		for c := 0; c < depth; c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[node.pegRule], strconv.Quote(string(([]rune(buffer)[node.begin:node.end]))))
		if node.up != nil {
			node.up.print(depth+1, buffer)
		}
		node = node.next
	}
}

func (ast *node32) Print(buffer string) {
	ast.print(0, buffer)
}

type element struct {
	node *node32
	down *element
}

/* ${@} bit structure for abstract syntax tree */
type token16 struct {
	pegRule
	begin, end, next int16
}

func (t *token16) isZero() bool {
	return t.pegRule == ruleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token16) isParentOf(u token16) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token16) getToken32() token32 {
	return token32{pegRule: t.pegRule, begin: int32(t.begin), end: int32(t.end), next: int32(t.next)}
}

func (t *token16) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", rul3s[t.pegRule], t.begin, t.end, t.next)
}

type tokens16 struct {
	tree    []token16
	ordered [][]token16
}

func (t *tokens16) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens16) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens16) Order() [][]token16 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int16, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.pegRule == ruleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token16, len(depths)), make([]token16, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = int16(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type state16 struct {
	token16
	depths []int16
	leaf   bool
}

func (t *tokens16) AST() *node32 {
	tokens := t.Tokens()
	stack := &element{node: &node32{token32: <-tokens}}
	for token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	return stack.node
}

func (t *tokens16) PreOrder() (<-chan state16, [][]token16) {
	s, ordered := make(chan state16, 6), t.Order()
	go func() {
		var states [8]state16
		for i, _ := range states {
			states[i].depths = make([]int16, len(ordered))
		}
		depths, state, depth := make([]int16, len(ordered)), 0, 1
		write := func(t token16, leaf bool) {
			S := states[state]
			state, S.pegRule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.pegRule, t.begin, t.end, int16(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token16 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token16{pegRule: rule_In_, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token16{pegRule: rulePre_, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.pegRule != ruleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.pegRule != ruleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token16{pegRule: rule_Suf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens16) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", rul3s[token.pegRule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", rul3s[token.pegRule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", rul3s[token.pegRule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens16) PrintSyntaxTree(buffer string) {
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[token.pegRule], strconv.Quote(string(([]rune(buffer)[token.begin:token.end]))))
	}
}

func (t *tokens16) Add(rule pegRule, begin, end, depth, index int) {
	t.tree[index] = token16{pegRule: rule, begin: int16(begin), end: int16(end), next: int16(depth)}
}

func (t *tokens16) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.getToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens16) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i, _ := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].getToken32()
		}
	}
	return tokens
}

/* ${@} bit structure for abstract syntax tree */
type token32 struct {
	pegRule
	begin, end, next int32
}

func (t *token32) isZero() bool {
	return t.pegRule == ruleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token32) isParentOf(u token32) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token32) getToken32() token32 {
	return token32{pegRule: t.pegRule, begin: int32(t.begin), end: int32(t.end), next: int32(t.next)}
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", rul3s[t.pegRule], t.begin, t.end, t.next)
}

type tokens32 struct {
	tree    []token32
	ordered [][]token32
}

func (t *tokens32) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) Order() [][]token32 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int32, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.pegRule == ruleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token32, len(depths)), make([]token32, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = int32(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type state32 struct {
	token32
	depths []int32
	leaf   bool
}

func (t *tokens32) AST() *node32 {
	tokens := t.Tokens()
	stack := &element{node: &node32{token32: <-tokens}}
	for token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	return stack.node
}

func (t *tokens32) PreOrder() (<-chan state32, [][]token32) {
	s, ordered := make(chan state32, 6), t.Order()
	go func() {
		var states [8]state32
		for i, _ := range states {
			states[i].depths = make([]int32, len(ordered))
		}
		depths, state, depth := make([]int32, len(ordered)), 0, 1
		write := func(t token32, leaf bool) {
			S := states[state]
			state, S.pegRule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.pegRule, t.begin, t.end, int32(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token32 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token32{pegRule: rule_In_, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token32{pegRule: rulePre_, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.pegRule != ruleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.pegRule != ruleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token32{pegRule: rule_Suf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens32) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", rul3s[token.pegRule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", rul3s[token.pegRule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", rul3s[ordered[i][depths[i]-1].pegRule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", rul3s[token.pegRule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", rul3s[token.pegRule], strconv.Quote(string(([]rune(buffer)[token.begin:token.end]))))
	}
}

func (t *tokens32) Add(rule pegRule, begin, end, depth, index int) {
	t.tree[index] = token32{pegRule: rule, begin: int32(begin), end: int32(end), next: int32(depth)}
}

func (t *tokens32) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.getToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens32) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i, _ := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].getToken32()
		}
	}
	return tokens
}

func (t *tokens16) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		for i, v := range tree {
			expanded[i] = v.getToken32()
		}
		return &tokens32{tree: expanded}
	}
	return nil
}

func (t *tokens32) Expand(index int) tokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	return nil
}

type NQuads struct {
	NQuadsBase

	Buffer string
	buffer []rune
	rules  [29]func() bool
	Parse  func(rule ...int) error
	Reset  func()
	tokenTree
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer string, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer[0:] {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p *NQuads
}

func (e *parseError) Error() string {
	tokens, error := e.p.tokenTree.Error(), "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.Buffer, positions)
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf("parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n",
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			/*strconv.Quote(*/ e.p.Buffer[begin:end] /*)*/)
	}

	return error
}

func (p *NQuads) PrintSyntaxTree() {
	p.tokenTree.PrintSyntaxTree(p.Buffer)
}

func (p *NQuads) Highlighter() {
	p.tokenTree.PrintSyntax()
}

func (p *NQuads) Execute() {
	buffer, begin, end := p.Buffer, 0, 0
	for token := range p.tokenTree.Tokens() {
		switch token.pegRule {

		case rulePegText:
			begin, end = int(token.begin), int(token.end)

		case ruleAction0:
			p.setStatement()
		case ruleAction1:
			p.setSubject()
		case ruleAction2:
			p.setPredicate()
		case ruleAction3:
			p.setObject()
		case ruleAction4:
			p.setGraph()
		case ruleAction5:
			p.setLangTag(buffer[begin:end])
		case ruleAction6:
			p.setIri(buffer[begin:end])
		case ruleAction7:
			p.setString(buffer[begin:end])
		case ruleAction8:
			p.setBlank(buffer[begin:end])

		}
	}
	_, _, _ = buffer, begin, end
}

func (p *NQuads) Init() {
	p.buffer = []rune(p.Buffer)
	if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != end_symbol {
		p.buffer = append(p.buffer, end_symbol)
	}

	var tree tokenTree = &tokens16{tree: make([]token16, math.MaxInt16)}
	position, depth, tokenIndex, buffer, _rules := 0, 0, 0, p.buffer, p.rules

	p.Parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokenTree = tree
		if matches {
			p.tokenTree.trim(tokenIndex)
			return nil
		}
		return &parseError{p}
	}

	p.Reset = func() {
		position, tokenIndex, depth = 0, 0, 0
	}

	add := func(rule pegRule, begin int) {
		if t := tree.Expand(tokenIndex); t != nil {
			tree = t
		}
		tree.Add(rule, begin, position, depth, tokenIndex)
		tokenIndex++
	}

	matchDot := func() bool {
		if buffer[position] != end_symbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 nquadsDoc <- <(ws* statement? (EOL ws* statement?)* !.)> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
			l2:
				{
					position3, tokenIndex3, depth3 := position, tokenIndex, depth
					if !_rules[rulews]() {
						goto l3
					}
					goto l2
				l3:
					position, tokenIndex, depth = position3, tokenIndex3, depth3
				}
				{
					position4, tokenIndex4, depth4 := position, tokenIndex, depth
					if !_rules[rulestatement]() {
						goto l4
					}
					goto l5
				l4:
					position, tokenIndex, depth = position4, tokenIndex4, depth4
				}
			l5:
			l6:
				{
					position7, tokenIndex7, depth7 := position, tokenIndex, depth
					{
						position8 := position
						depth++
						{
							position11, tokenIndex11, depth11 := position, tokenIndex, depth
							if buffer[position] != rune('\r') {
								goto l12
							}
							position++
							goto l11
						l12:
							position, tokenIndex, depth = position11, tokenIndex11, depth11
							if buffer[position] != rune('\n') {
								goto l7
							}
							position++
						}
					l11:
					l9:
						{
							position10, tokenIndex10, depth10 := position, tokenIndex, depth
							{
								position13, tokenIndex13, depth13 := position, tokenIndex, depth
								if buffer[position] != rune('\r') {
									goto l14
								}
								position++
								goto l13
							l14:
								position, tokenIndex, depth = position13, tokenIndex13, depth13
								if buffer[position] != rune('\n') {
									goto l10
								}
								position++
							}
						l13:
							goto l9
						l10:
							position, tokenIndex, depth = position10, tokenIndex10, depth10
						}
						depth--
						add(ruleEOL, position8)
					}
				l15:
					{
						position16, tokenIndex16, depth16 := position, tokenIndex, depth
						if !_rules[rulews]() {
							goto l16
						}
						goto l15
					l16:
						position, tokenIndex, depth = position16, tokenIndex16, depth16
					}
					{
						position17, tokenIndex17, depth17 := position, tokenIndex, depth
						if !_rules[rulestatement]() {
							goto l17
						}
						goto l18
					l17:
						position, tokenIndex, depth = position17, tokenIndex17, depth17
					}
				l18:
					goto l6
				l7:
					position, tokenIndex, depth = position7, tokenIndex7, depth7
				}
				{
					position19, tokenIndex19, depth19 := position, tokenIndex, depth
					if !matchDot() {
						goto l19
					}
					goto l0
				l19:
					position, tokenIndex, depth = position19, tokenIndex19, depth19
				}
				depth--
				add(rulenquadsDoc, position1)
			}
			return true
		l0:
			position, tokenIndex, depth = position0, tokenIndex0, depth0
			return false
		},
		/* 1 statement <- <(subject predicate object graphLabel? '.' Action0 ws*)> */
		func() bool {
			position20, tokenIndex20, depth20 := position, tokenIndex, depth
			{
				position21 := position
				depth++
				{
					position22 := position
					depth++
					{
						position23, tokenIndex23, depth23 := position, tokenIndex, depth
						if !_rules[ruleIRIREF]() {
							goto l24
						}
					l25:
						{
							position26, tokenIndex26, depth26 := position, tokenIndex, depth
							if !_rules[rulews]() {
								goto l26
							}
							goto l25
						l26:
							position, tokenIndex, depth = position26, tokenIndex26, depth26
						}
						goto l23
					l24:
						position, tokenIndex, depth = position23, tokenIndex23, depth23
						if !_rules[ruleBLANK_NODE_LABEL]() {
							goto l20
						}
					l27:
						{
							position28, tokenIndex28, depth28 := position, tokenIndex, depth
							if !_rules[rulews]() {
								goto l28
							}
							goto l27
						l28:
							position, tokenIndex, depth = position28, tokenIndex28, depth28
						}
					}
				l23:
					{
						add(ruleAction1, position)
					}
					depth--
					add(rulesubject, position22)
				}
				{
					position30 := position
					depth++
					if !_rules[ruleIRIREF]() {
						goto l20
					}
				l31:
					{
						position32, tokenIndex32, depth32 := position, tokenIndex, depth
						if !_rules[rulews]() {
							goto l32
						}
						goto l31
					l32:
						position, tokenIndex, depth = position32, tokenIndex32, depth32
					}
					{
						add(ruleAction2, position)
					}
					depth--
					add(rulepredicate, position30)
				}
				{
					position34 := position
					depth++
					{
						switch buffer[position] {
						case '"':
							{
								position36 := position
								depth++
								{
									position37 := position
									depth++
									if buffer[position] != rune('"') {
										goto l20
									}
									position++
									{
										position38 := position
										depth++
									l39:
										{
											position40, tokenIndex40, depth40 := position, tokenIndex, depth
											{
												position41, tokenIndex41, depth41 := position, tokenIndex, depth
												{
													position43, tokenIndex43, depth43 := position, tokenIndex, depth
													{
														switch buffer[position] {
														case '\n':
															if buffer[position] != rune('\n') {
																goto l43
															}
															position++
															break
														case '\r':
															if buffer[position] != rune('\r') {
																goto l43
															}
															position++
															break
														case '\\':
															if buffer[position] != rune('\\') {
																goto l43
															}
															position++
															break
														default:
															if buffer[position] != rune('"') {
																goto l43
															}
															position++
															break
														}
													}

													goto l42
												l43:
													position, tokenIndex, depth = position43, tokenIndex43, depth43
												}
												if !matchDot() {
													goto l42
												}
												goto l41
											l42:
												position, tokenIndex, depth = position41, tokenIndex41, depth41
												{
													position46 := position
													depth++
													if buffer[position] != rune('\\') {
														goto l45
													}
													position++
													{
														switch buffer[position] {
														case '\\':
															if buffer[position] != rune('\\') {
																goto l45
															}
															position++
															break
														case '\'':
															if buffer[position] != rune('\'') {
																goto l45
															}
															position++
															break
														case '"':
															if buffer[position] != rune('"') {
																goto l45
															}
															position++
															break
														case 'f':
															if buffer[position] != rune('f') {
																goto l45
															}
															position++
															break
														case 'r':
															if buffer[position] != rune('r') {
																goto l45
															}
															position++
															break
														case 'n':
															if buffer[position] != rune('n') {
																goto l45
															}
															position++
															break
														case 'b':
															if buffer[position] != rune('b') {
																goto l45
															}
															position++
															break
														default:
															if buffer[position] != rune('t') {
																goto l45
															}
															position++
															break
														}
													}

													depth--
													add(ruleECHAR, position46)
												}
												goto l41
											l45:
												position, tokenIndex, depth = position41, tokenIndex41, depth41
												if !_rules[ruleUCHAR]() {
													goto l40
												}
											}
										l41:
											goto l39
										l40:
											position, tokenIndex, depth = position40, tokenIndex40, depth40
										}
										depth--
										add(rulePegText, position38)
									}
									{
										add(ruleAction7, position)
									}
									if buffer[position] != rune('"') {
										goto l20
									}
									position++
									depth--
									add(ruleSTRING_LITERAL_QUOTE, position37)
								}
							l49:
								{
									position50, tokenIndex50, depth50 := position, tokenIndex, depth
									if !_rules[rulews]() {
										goto l50
									}
									goto l49
								l50:
									position, tokenIndex, depth = position50, tokenIndex50, depth50
								}
								{
									position51, tokenIndex51, depth51 := position, tokenIndex, depth
									{
										position53, tokenIndex53, depth53 := position, tokenIndex, depth
										if buffer[position] != rune('^') {
											goto l54
										}
										position++
										if buffer[position] != rune('^') {
											goto l54
										}
										position++
									l55:
										{
											position56, tokenIndex56, depth56 := position, tokenIndex, depth
											if !_rules[rulews]() {
												goto l56
											}
											goto l55
										l56:
											position, tokenIndex, depth = position56, tokenIndex56, depth56
										}
										if !_rules[ruleIRIREF]() {
											goto l54
										}
									l57:
										{
											position58, tokenIndex58, depth58 := position, tokenIndex, depth
											if !_rules[rulews]() {
												goto l58
											}
											goto l57
										l58:
											position, tokenIndex, depth = position58, tokenIndex58, depth58
										}
										goto l53
									l54:
										position, tokenIndex, depth = position53, tokenIndex53, depth53
										{
											position59 := position
											depth++
											if buffer[position] != rune('@') {
												goto l51
											}
											position++
											{
												position60 := position
												depth++
												{
													position63, tokenIndex63, depth63 := position, tokenIndex, depth
													if c := buffer[position]; c < rune('a') || c > rune('z') {
														goto l64
													}
													position++
													goto l63
												l64:
													position, tokenIndex, depth = position63, tokenIndex63, depth63
													if c := buffer[position]; c < rune('A') || c > rune('Z') {
														goto l51
													}
													position++
												}
											l63:
											l61:
												{
													position62, tokenIndex62, depth62 := position, tokenIndex, depth
													{
														position65, tokenIndex65, depth65 := position, tokenIndex, depth
														if c := buffer[position]; c < rune('a') || c > rune('z') {
															goto l66
														}
														position++
														goto l65
													l66:
														position, tokenIndex, depth = position65, tokenIndex65, depth65
														if c := buffer[position]; c < rune('A') || c > rune('Z') {
															goto l62
														}
														position++
													}
												l65:
													goto l61
												l62:
													position, tokenIndex, depth = position62, tokenIndex62, depth62
												}
											l67:
												{
													position68, tokenIndex68, depth68 := position, tokenIndex, depth
													if buffer[position] != rune('-') {
														goto l68
													}
													position++
													{
														switch buffer[position] {
														case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
															if c := buffer[position]; c < rune('0') || c > rune('9') {
																goto l68
															}
															position++
															break
														case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
															if c := buffer[position]; c < rune('A') || c > rune('Z') {
																goto l68
															}
															position++
															break
														default:
															if c := buffer[position]; c < rune('a') || c > rune('z') {
																goto l68
															}
															position++
															break
														}
													}

												l69:
													{
														position70, tokenIndex70, depth70 := position, tokenIndex, depth
														{
															switch buffer[position] {
															case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
																if c := buffer[position]; c < rune('0') || c > rune('9') {
																	goto l70
																}
																position++
																break
															case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
																if c := buffer[position]; c < rune('A') || c > rune('Z') {
																	goto l70
																}
																position++
																break
															default:
																if c := buffer[position]; c < rune('a') || c > rune('z') {
																	goto l70
																}
																position++
																break
															}
														}

														goto l69
													l70:
														position, tokenIndex, depth = position70, tokenIndex70, depth70
													}
													goto l67
												l68:
													position, tokenIndex, depth = position68, tokenIndex68, depth68
												}
												depth--
												add(rulePegText, position60)
											}
											{
												add(ruleAction5, position)
											}
											depth--
											add(ruleLANGTAG, position59)
										}
									l74:
										{
											position75, tokenIndex75, depth75 := position, tokenIndex, depth
											if !_rules[rulews]() {
												goto l75
											}
											goto l74
										l75:
											position, tokenIndex, depth = position75, tokenIndex75, depth75
										}
									}
								l53:
									goto l52
								l51:
									position, tokenIndex, depth = position51, tokenIndex51, depth51
								}
							l52:
								depth--
								add(ruleliteral, position36)
							}
							break
						case '_':
							if !_rules[ruleBLANK_NODE_LABEL]() {
								goto l20
							}
						l76:
							{
								position77, tokenIndex77, depth77 := position, tokenIndex, depth
								if !_rules[rulews]() {
									goto l77
								}
								goto l76
							l77:
								position, tokenIndex, depth = position77, tokenIndex77, depth77
							}
							break
						default:
							if !_rules[ruleIRIREF]() {
								goto l20
							}
						l78:
							{
								position79, tokenIndex79, depth79 := position, tokenIndex, depth
								if !_rules[rulews]() {
									goto l79
								}
								goto l78
							l79:
								position, tokenIndex, depth = position79, tokenIndex79, depth79
							}
							break
						}
					}

					{
						add(ruleAction3, position)
					}
					depth--
					add(ruleobject, position34)
				}
				{
					position81, tokenIndex81, depth81 := position, tokenIndex, depth
					{
						position83 := position
						depth++
						{
							position84, tokenIndex84, depth84 := position, tokenIndex, depth
							if !_rules[ruleIRIREF]() {
								goto l85
							}
						l86:
							{
								position87, tokenIndex87, depth87 := position, tokenIndex, depth
								if !_rules[rulews]() {
									goto l87
								}
								goto l86
							l87:
								position, tokenIndex, depth = position87, tokenIndex87, depth87
							}
							goto l84
						l85:
							position, tokenIndex, depth = position84, tokenIndex84, depth84
							if !_rules[ruleBLANK_NODE_LABEL]() {
								goto l81
							}
						l88:
							{
								position89, tokenIndex89, depth89 := position, tokenIndex, depth
								if !_rules[rulews]() {
									goto l89
								}
								goto l88
							l89:
								position, tokenIndex, depth = position89, tokenIndex89, depth89
							}
						}
					l84:
						{
							add(ruleAction4, position)
						}
						depth--
						add(rulegraphLabel, position83)
					}
					goto l82
				l81:
					position, tokenIndex, depth = position81, tokenIndex81, depth81
				}
			l82:
				if buffer[position] != rune('.') {
					goto l20
				}
				position++
				{
					add(ruleAction0, position)
				}
			l92:
				{
					position93, tokenIndex93, depth93 := position, tokenIndex, depth
					if !_rules[rulews]() {
						goto l93
					}
					goto l92
				l93:
					position, tokenIndex, depth = position93, tokenIndex93, depth93
				}
				depth--
				add(rulestatement, position21)
			}
			return true
		l20:
			position, tokenIndex, depth = position20, tokenIndex20, depth20
			return false
		},
		/* 2 subject <- <(((IRIREF ws*) / (BLANK_NODE_LABEL ws*)) Action1)> */
		nil,
		/* 3 predicate <- <(IRIREF ws* Action2)> */
		nil,
		/* 4 object <- <(((&('"') literal) | (&('_') (BLANK_NODE_LABEL ws*)) | (&('<') (IRIREF ws*))) Action3)> */
		nil,
		/* 5 graphLabel <- <(((IRIREF ws*) / (BLANK_NODE_LABEL ws*)) Action4)> */
		nil,
		/* 6 literal <- <(STRING_LITERAL_QUOTE ws* (('^' '^' ws* IRIREF ws*) / (LANGTAG ws*))?)> */
		nil,
		/* 7 LANGTAG <- <('@' <(([a-z] / [A-Z])+ ('-' ((&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]))+)*)> Action5)> */
		nil,
		/* 8 EOL <- <('\r' / '\n')+> */
		nil,
		/* 9 IRIREF <- <('<' <((!((&('\\') '\\') | (&('`') '`') | (&('^') '^') | (&('|') '|') | (&('}') '}') | (&('{') '{') | (&('"') '"') | (&('>') '>') | (&('<') '<') | (&('\x01' | '\x02' | '\x03' | '\x04' | '\x05' | '\x06' | '\a' | '\b' | '\t' | '\n' | '\v' | '\f' | '\r' | '\x0e' | '\x0f' | '\x10' | '\x11' | '\x12' | '\x13' | '\x14' | '\x15' | '\x16' | '\x17' | '\x18' | '\x19' | '\x1a' | '\x1b' | '\x1c' | '\x1d' | '\x1e' | '\x1f' | ' ') [- ])) .) / UCHAR)*> Action6 '>')> */
		func() bool {
			position101, tokenIndex101, depth101 := position, tokenIndex, depth
			{
				position102 := position
				depth++
				if buffer[position] != rune('<') {
					goto l101
				}
				position++
				{
					position103 := position
					depth++
				l104:
					{
						position105, tokenIndex105, depth105 := position, tokenIndex, depth
						{
							position106, tokenIndex106, depth106 := position, tokenIndex, depth
							{
								position108, tokenIndex108, depth108 := position, tokenIndex, depth
								{
									switch buffer[position] {
									case '\\':
										if buffer[position] != rune('\\') {
											goto l108
										}
										position++
										break
									case '`':
										if buffer[position] != rune('`') {
											goto l108
										}
										position++
										break
									case '^':
										if buffer[position] != rune('^') {
											goto l108
										}
										position++
										break
									case '|':
										if buffer[position] != rune('|') {
											goto l108
										}
										position++
										break
									case '}':
										if buffer[position] != rune('}') {
											goto l108
										}
										position++
										break
									case '{':
										if buffer[position] != rune('{') {
											goto l108
										}
										position++
										break
									case '"':
										if buffer[position] != rune('"') {
											goto l108
										}
										position++
										break
									case '>':
										if buffer[position] != rune('>') {
											goto l108
										}
										position++
										break
									case '<':
										if buffer[position] != rune('<') {
											goto l108
										}
										position++
										break
									default:
										if c := buffer[position]; c < rune('\x01') || c > rune(' ') {
											goto l108
										}
										position++
										break
									}
								}

								goto l107
							l108:
								position, tokenIndex, depth = position108, tokenIndex108, depth108
							}
							if !matchDot() {
								goto l107
							}
							goto l106
						l107:
							position, tokenIndex, depth = position106, tokenIndex106, depth106
							if !_rules[ruleUCHAR]() {
								goto l105
							}
						}
					l106:
						goto l104
					l105:
						position, tokenIndex, depth = position105, tokenIndex105, depth105
					}
					depth--
					add(rulePegText, position103)
				}
				{
					add(ruleAction6, position)
				}
				if buffer[position] != rune('>') {
					goto l101
				}
				position++
				depth--
				add(ruleIRIREF, position102)
			}
			return true
		l101:
			position, tokenIndex, depth = position101, tokenIndex101, depth101
			return false
		},
		/* 10 STRING_LITERAL_QUOTE <- <('"' <((!((&('\n') '\n') | (&('\r') '\r') | (&('\\') '\\') | (&('"') '"')) .) / ECHAR / UCHAR)*> Action7 '"')> */
		nil,
		/* 11 BLANK_NODE_LABEL <- <('_' ':' <(!((&(' ') ' ') | (&('\n') '\n') | (&('\r') '\r') | (&('\t') '\t')) .)*> Action8)> */
		func() bool {
			position112, tokenIndex112, depth112 := position, tokenIndex, depth
			{
				position113 := position
				depth++
				if buffer[position] != rune('_') {
					goto l112
				}
				position++
				if buffer[position] != rune(':') {
					goto l112
				}
				position++
				{
					position114 := position
					depth++
				l115:
					{
						position116, tokenIndex116, depth116 := position, tokenIndex, depth
						{
							position117, tokenIndex117, depth117 := position, tokenIndex, depth
							{
								switch buffer[position] {
								case ' ':
									if buffer[position] != rune(' ') {
										goto l117
									}
									position++
									break
								case '\n':
									if buffer[position] != rune('\n') {
										goto l117
									}
									position++
									break
								case '\r':
									if buffer[position] != rune('\r') {
										goto l117
									}
									position++
									break
								default:
									if buffer[position] != rune('\t') {
										goto l117
									}
									position++
									break
								}
							}

							goto l116
						l117:
							position, tokenIndex, depth = position117, tokenIndex117, depth117
						}
						if !matchDot() {
							goto l116
						}
						goto l115
					l116:
						position, tokenIndex, depth = position116, tokenIndex116, depth116
					}
					depth--
					add(rulePegText, position114)
				}
				{
					add(ruleAction8, position)
				}
				depth--
				add(ruleBLANK_NODE_LABEL, position113)
			}
			return true
		l112:
			position, tokenIndex, depth = position112, tokenIndex112, depth112
			return false
		},
		/* 12 UCHAR <- <(('\\' 'u' HEX HEX HEX HEX) / ('\\' 'U' HEX HEX HEX HEX HEX HEX HEX HEX))> */
		func() bool {
			position120, tokenIndex120, depth120 := position, tokenIndex, depth
			{
				position121 := position
				depth++
				{
					position122, tokenIndex122, depth122 := position, tokenIndex, depth
					if buffer[position] != rune('\\') {
						goto l123
					}
					position++
					if buffer[position] != rune('u') {
						goto l123
					}
					position++
					if !_rules[ruleHEX]() {
						goto l123
					}
					if !_rules[ruleHEX]() {
						goto l123
					}
					if !_rules[ruleHEX]() {
						goto l123
					}
					if !_rules[ruleHEX]() {
						goto l123
					}
					goto l122
				l123:
					position, tokenIndex, depth = position122, tokenIndex122, depth122
					if buffer[position] != rune('\\') {
						goto l120
					}
					position++
					if buffer[position] != rune('U') {
						goto l120
					}
					position++
					if !_rules[ruleHEX]() {
						goto l120
					}
					if !_rules[ruleHEX]() {
						goto l120
					}
					if !_rules[ruleHEX]() {
						goto l120
					}
					if !_rules[ruleHEX]() {
						goto l120
					}
					if !_rules[ruleHEX]() {
						goto l120
					}
					if !_rules[ruleHEX]() {
						goto l120
					}
					if !_rules[ruleHEX]() {
						goto l120
					}
					if !_rules[ruleHEX]() {
						goto l120
					}
				}
			l122:
				depth--
				add(ruleUCHAR, position121)
			}
			return true
		l120:
			position, tokenIndex, depth = position120, tokenIndex120, depth120
			return false
		},
		/* 13 ECHAR <- <('\\' ((&('\\') '\\') | (&('\'') '\'') | (&('"') '"') | (&('f') 'f') | (&('r') 'r') | (&('n') 'n') | (&('b') 'b') | (&('t') 't')))> */
		nil,
		/* 14 HEX <- <((&('a' | 'b' | 'c' | 'd' | 'e' | 'f') [a-f]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F') [A-F]) | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]))> */
		func() bool {
			position125, tokenIndex125, depth125 := position, tokenIndex, depth
			{
				position126 := position
				depth++
				{
					switch buffer[position] {
					case 'a', 'b', 'c', 'd', 'e', 'f':
						if c := buffer[position]; c < rune('a') || c > rune('f') {
							goto l125
						}
						position++
						break
					case 'A', 'B', 'C', 'D', 'E', 'F':
						if c := buffer[position]; c < rune('A') || c > rune('F') {
							goto l125
						}
						position++
						break
					default:
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l125
						}
						position++
						break
					}
				}

				depth--
				add(ruleHEX, position126)
			}
			return true
		l125:
			position, tokenIndex, depth = position125, tokenIndex125, depth125
			return false
		},
		/* 15 spc <- <('\t' / ' ')> */
		nil,
		/* 16 ws <- <(spc / comment)*> */
		func() bool {
			{
				position130 := position
				depth++
			l131:
				{
					position132, tokenIndex132, depth132 := position, tokenIndex, depth
					{
						position133, tokenIndex133, depth133 := position, tokenIndex, depth
						{
							position135 := position
							depth++
							{
								position136, tokenIndex136, depth136 := position, tokenIndex, depth
								if buffer[position] != rune('\t') {
									goto l137
								}
								position++
								goto l136
							l137:
								position, tokenIndex, depth = position136, tokenIndex136, depth136
								if buffer[position] != rune(' ') {
									goto l134
								}
								position++
							}
						l136:
							depth--
							add(rulespc, position135)
						}
						goto l133
					l134:
						position, tokenIndex, depth = position133, tokenIndex133, depth133
						{
							position138 := position
							depth++
							if buffer[position] != rune('#') {
								goto l132
							}
							position++
						l139:
							{
								position140, tokenIndex140, depth140 := position, tokenIndex, depth
								{
									position141, tokenIndex141, depth141 := position, tokenIndex, depth
									{
										position142, tokenIndex142, depth142 := position, tokenIndex, depth
										if buffer[position] != rune('\r') {
											goto l143
										}
										position++
										goto l142
									l143:
										position, tokenIndex, depth = position142, tokenIndex142, depth142
										if buffer[position] != rune('\n') {
											goto l141
										}
										position++
									}
								l142:
									goto l140
								l141:
									position, tokenIndex, depth = position141, tokenIndex141, depth141
								}
								if !matchDot() {
									goto l140
								}
								goto l139
							l140:
								position, tokenIndex, depth = position140, tokenIndex140, depth140
							}
							depth--
							add(rulecomment, position138)
						}
					}
				l133:
					goto l131
				l132:
					position, tokenIndex, depth = position132, tokenIndex132, depth132
				}
				depth--
				add(rulews, position130)
			}
			return true
		},
		/* 17 comment <- <('#' (!('\r' / '\n') .)*)> */
		nil,
		/* 19 Action0 <- <{ p.setStatement() }> */
		nil,
		/* 20 Action1 <- <{ p.setSubject() }> */
		nil,
		/* 21 Action2 <- <{ p.setPredicate() }> */
		nil,
		/* 22 Action3 <- <{ p.setObject() }> */
		nil,
		/* 23 Action4 <- <{ p.setGraph() }> */
		nil,
		nil,
		/* 25 Action5 <- <{ p.setLangTag(buffer[begin:end]) }> */
		nil,
		/* 26 Action6 <- <{ p.setIri(buffer[begin:end]) }> */
		nil,
		/* 27 Action7 <- <{ p.setString(buffer[begin:end]) }> */
		nil,
		/* 28 Action8 <- <{ p.setBlank(buffer[begin:end]) }> */
		nil,
	}
	p.rules = _rules
}
