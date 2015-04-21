package server

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
	rulestart
	ruleheader
	rulechallenge_cred
	ruleauth_scheme
	ruleauth_params
	ruleauth_param
	ruleauth_param_val
	rulequoted_string
	rulequoted_text
	rulequoted_pair
	ruletoken
	rulews
	rulePegText
	ruleAction0
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6

	rulePre_
	rule_In_
	rule_Suf
)

var rul3s = [...]string{
	"Unknown",
	"start",
	"header",
	"challenge_cred",
	"auth_scheme",
	"auth_params",
	"auth_param",
	"auth_param_val",
	"quoted_string",
	"quoted_text",
	"quoted_pair",
	"token",
	"ws",
	"PegText",
	"Action0",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",
	"Action6",

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

type HeaderParser struct {
	HeaderParse
	param_name string
	last_value string

	Buffer string
	buffer []rune
	rules  [21]func() bool
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
	p *HeaderParser
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

func (p *HeaderParser) PrintSyntaxTree() {
	p.tokenTree.PrintSyntaxTree(p.Buffer)
}

func (p *HeaderParser) Highlighter() {
	p.tokenTree.PrintSyntax()
}

func (p *HeaderParser) Execute() {
	buffer, begin, end := p.Buffer, 0, 0
	for token := range p.tokenTree.Tokens() {
		switch token.pegRule {

		case rulePegText:
			begin, end = int(token.begin), int(token.end)

		case ruleAction0:
			p.addAuthScheme(buffer[begin:end])
		case ruleAction1:
			p.param_name = buffer[begin:end]
		case ruleAction2:
			p.setParam(p.param_name, buffer[begin:end])
		case ruleAction3:
			p.setParam(p.param_name, p.last_value)
		case ruleAction4:
			p.last_value = ""
		case ruleAction5:
			p.last_value = p.last_value + buffer[begin:end]
		case ruleAction6:
			p.last_value = p.last_value + buffer[begin:end]

		}
	}
	_, _, _ = buffer, begin, end
}

func (p *HeaderParser) Init() {
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
		/* 0 start <- <(header !.)> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
				{
					position2 := position
					depth++
				l3:
					{
						position4, tokenIndex4, depth4 := position, tokenIndex, depth
						if buffer[position] != rune(',') {
							goto l4
						}
						position++
					l5:
						{
							position6, tokenIndex6, depth6 := position, tokenIndex, depth
							if !_rules[rulews]() {
								goto l6
							}
							goto l5
						l6:
							position, tokenIndex, depth = position6, tokenIndex6, depth6
						}
						goto l3
					l4:
						position, tokenIndex, depth = position4, tokenIndex4, depth4
					}
					if !_rules[rulechallenge_cred]() {
						goto l0
					}
				l7:
					{
						position8, tokenIndex8, depth8 := position, tokenIndex, depth
					l9:
						{
							position10, tokenIndex10, depth10 := position, tokenIndex, depth
							if !_rules[rulews]() {
								goto l10
							}
							goto l9
						l10:
							position, tokenIndex, depth = position10, tokenIndex10, depth10
						}
						if buffer[position] != rune(',') {
							goto l8
						}
						position++
						{
							position11, tokenIndex11, depth11 := position, tokenIndex, depth
						l13:
							{
								position14, tokenIndex14, depth14 := position, tokenIndex, depth
								if !_rules[rulews]() {
									goto l14
								}
								goto l13
							l14:
								position, tokenIndex, depth = position14, tokenIndex14, depth14
							}
							if !_rules[rulechallenge_cred]() {
								goto l11
							}
							goto l12
						l11:
							position, tokenIndex, depth = position11, tokenIndex11, depth11
						}
					l12:
						goto l7
					l8:
						position, tokenIndex, depth = position8, tokenIndex8, depth8
					}
					depth--
					add(ruleheader, position2)
				}
				{
					position15, tokenIndex15, depth15 := position, tokenIndex, depth
					if !matchDot() {
						goto l15
					}
					goto l0
				l15:
					position, tokenIndex, depth = position15, tokenIndex15, depth15
				}
				depth--
				add(rulestart, position1)
			}
			return true
		l0:
			position, tokenIndex, depth = position0, tokenIndex0, depth0
			return false
		},
		/* 1 header <- <((',' ws*)* challenge_cred (ws* ',' (ws* challenge_cred)?)*)> */
		nil,
		/* 2 challenge_cred <- <(auth_scheme auth_params)> */
		func() bool {
			position17, tokenIndex17, depth17 := position, tokenIndex, depth
			{
				position18 := position
				depth++
				{
					position19 := position
					depth++
					{
						position20 := position
						depth++
						if !_rules[ruletoken]() {
							goto l17
						}
						depth--
						add(rulePegText, position20)
					}
					{
						add(ruleAction0, position)
					}
					depth--
					add(ruleauth_scheme, position19)
				}
				{
					position22 := position
					depth++
					{
						position23, tokenIndex23, depth23 := position, tokenIndex, depth
						if buffer[position] != rune(',') {
							goto l24
						}
						position++
						goto l23
					l24:
						position, tokenIndex, depth = position23, tokenIndex23, depth23
						if !_rules[rulews]() {
							goto l17
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
						if !_rules[ruleauth_param]() {
							goto l17
						}
					}
				l23:
				l27:
					{
						position28, tokenIndex28, depth28 := position, tokenIndex, depth
					l29:
						{
							position30, tokenIndex30, depth30 := position, tokenIndex, depth
							if !_rules[rulews]() {
								goto l30
							}
							goto l29
						l30:
							position, tokenIndex, depth = position30, tokenIndex30, depth30
						}
						if buffer[position] != rune(',') {
							goto l28
						}
						position++
						{
							position31, tokenIndex31, depth31 := position, tokenIndex, depth
						l33:
							{
								position34, tokenIndex34, depth34 := position, tokenIndex, depth
								if !_rules[rulews]() {
									goto l34
								}
								goto l33
							l34:
								position, tokenIndex, depth = position34, tokenIndex34, depth34
							}
							if !_rules[ruleauth_param]() {
								goto l31
							}
							goto l32
						l31:
							position, tokenIndex, depth = position31, tokenIndex31, depth31
						}
					l32:
						goto l27
					l28:
						position, tokenIndex, depth = position28, tokenIndex28, depth28
					}
					depth--
					add(ruleauth_params, position22)
				}
				depth--
				add(rulechallenge_cred, position18)
			}
			return true
		l17:
			position, tokenIndex, depth = position17, tokenIndex17, depth17
			return false
		},
		/* 3 auth_scheme <- <(<token> Action0)> */
		nil,
		/* 4 auth_params <- <((',' / (ws+ auth_param)) (ws* ',' (ws* auth_param)?)*)> */
		nil,
		/* 5 auth_param <- <(<token> Action1 ws* '=' ws* auth_param_val)> */
		func() bool {
			position37, tokenIndex37, depth37 := position, tokenIndex, depth
			{
				position38 := position
				depth++
				{
					position39 := position
					depth++
					if !_rules[ruletoken]() {
						goto l37
					}
					depth--
					add(rulePegText, position39)
				}
				{
					add(ruleAction1, position)
				}
			l41:
				{
					position42, tokenIndex42, depth42 := position, tokenIndex, depth
					if !_rules[rulews]() {
						goto l42
					}
					goto l41
				l42:
					position, tokenIndex, depth = position42, tokenIndex42, depth42
				}
				if buffer[position] != rune('=') {
					goto l37
				}
				position++
			l43:
				{
					position44, tokenIndex44, depth44 := position, tokenIndex, depth
					if !_rules[rulews]() {
						goto l44
					}
					goto l43
				l44:
					position, tokenIndex, depth = position44, tokenIndex44, depth44
				}
				{
					position45 := position
					depth++
					{
						position46, tokenIndex46, depth46 := position, tokenIndex, depth
						{
							position48 := position
							depth++
							if !_rules[ruletoken]() {
								goto l47
							}
							depth--
							add(rulePegText, position48)
						}
						{
							add(ruleAction2, position)
						}
						goto l46
					l47:
						position, tokenIndex, depth = position46, tokenIndex46, depth46
						{
							position50 := position
							depth++
							{
								add(ruleAction4, position)
							}
							if buffer[position] != rune('"') {
								goto l37
							}
							position++
						l52:
							{
								position53, tokenIndex53, depth53 := position, tokenIndex, depth
								{
									position54, tokenIndex54, depth54 := position, tokenIndex, depth
									{
										position56 := position
										depth++
										{
											position57 := position
											depth++
											{
												switch buffer[position] {
												case '\\':
													if buffer[position] != rune('\\') {
														goto l55
													}
													position++
													break
												case '"':
													if buffer[position] != rune('"') {
														goto l55
													}
													position++
													break
												default:
													if buffer[position] != rune('~') {
														goto l55
													}
													position++
													break
												}
											}

											depth--
											add(rulePegText, position57)
										}
										{
											add(ruleAction5, position)
										}
										depth--
										add(rulequoted_text, position56)
									}
									goto l54
								l55:
									position, tokenIndex, depth = position54, tokenIndex54, depth54
									{
										position60 := position
										depth++
										if buffer[position] != rune('\\') {
											goto l53
										}
										position++
										{
											position61 := position
											depth++
											if !matchDot() {
												goto l53
											}
											depth--
											add(rulePegText, position61)
										}
										{
											add(ruleAction6, position)
										}
										depth--
										add(rulequoted_pair, position60)
									}
								}
							l54:
								goto l52
							l53:
								position, tokenIndex, depth = position53, tokenIndex53, depth53
							}
							if buffer[position] != rune('"') {
								goto l37
							}
							position++
							depth--
							add(rulequoted_string, position50)
						}
						{
							add(ruleAction3, position)
						}
					}
				l46:
					depth--
					add(ruleauth_param_val, position45)
				}
				depth--
				add(ruleauth_param, position38)
			}
			return true
		l37:
			position, tokenIndex, depth = position37, tokenIndex37, depth37
			return false
		},
		/* 6 auth_param_val <- <((<token> Action2) / (quoted_string Action3))> */
		nil,
		/* 7 quoted_string <- <(Action4 '"' (quoted_text / quoted_pair)* '"')> */
		nil,
		/* 8 quoted_text <- <(<((&('\\') '\\') | (&('"') '"') | (&('~') '~'))> Action5)> */
		nil,
		/* 9 quoted_pair <- <('\\' <.> Action6)> */
		nil,
		/* 10 token <- <((&('~') '~') | (&('|') '|') | (&('`') '`') | (&('_') '_') | (&('^') '^') | (&('.') '.') | (&('-') '-') | (&('+') '+') | (&('*') '*') | (&('\'') '\'') | (&('&') '&') | (&('%') '%') | (&('$') '$') | (&('#') '#') | (&('!') '!') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))+> */
		func() bool {
			position68, tokenIndex68, depth68 := position, tokenIndex, depth
			{
				position69 := position
				depth++
				{
					switch buffer[position] {
					case '~':
						if buffer[position] != rune('~') {
							goto l68
						}
						position++
						break
					case '|':
						if buffer[position] != rune('|') {
							goto l68
						}
						position++
						break
					case '`':
						if buffer[position] != rune('`') {
							goto l68
						}
						position++
						break
					case '_':
						if buffer[position] != rune('_') {
							goto l68
						}
						position++
						break
					case '^':
						if buffer[position] != rune('^') {
							goto l68
						}
						position++
						break
					case '.':
						if buffer[position] != rune('.') {
							goto l68
						}
						position++
						break
					case '-':
						if buffer[position] != rune('-') {
							goto l68
						}
						position++
						break
					case '+':
						if buffer[position] != rune('+') {
							goto l68
						}
						position++
						break
					case '*':
						if buffer[position] != rune('*') {
							goto l68
						}
						position++
						break
					case '\'':
						if buffer[position] != rune('\'') {
							goto l68
						}
						position++
						break
					case '&':
						if buffer[position] != rune('&') {
							goto l68
						}
						position++
						break
					case '%':
						if buffer[position] != rune('%') {
							goto l68
						}
						position++
						break
					case '$':
						if buffer[position] != rune('$') {
							goto l68
						}
						position++
						break
					case '#':
						if buffer[position] != rune('#') {
							goto l68
						}
						position++
						break
					case '!':
						if buffer[position] != rune('!') {
							goto l68
						}
						position++
						break
					case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l68
						}
						position++
						break
					case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l68
						}
						position++
						break
					default:
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l68
						}
						position++
						break
					}
				}

			l70:
				{
					position71, tokenIndex71, depth71 := position, tokenIndex, depth
					{
						switch buffer[position] {
						case '~':
							if buffer[position] != rune('~') {
								goto l71
							}
							position++
							break
						case '|':
							if buffer[position] != rune('|') {
								goto l71
							}
							position++
							break
						case '`':
							if buffer[position] != rune('`') {
								goto l71
							}
							position++
							break
						case '_':
							if buffer[position] != rune('_') {
								goto l71
							}
							position++
							break
						case '^':
							if buffer[position] != rune('^') {
								goto l71
							}
							position++
							break
						case '.':
							if buffer[position] != rune('.') {
								goto l71
							}
							position++
							break
						case '-':
							if buffer[position] != rune('-') {
								goto l71
							}
							position++
							break
						case '+':
							if buffer[position] != rune('+') {
								goto l71
							}
							position++
							break
						case '*':
							if buffer[position] != rune('*') {
								goto l71
							}
							position++
							break
						case '\'':
							if buffer[position] != rune('\'') {
								goto l71
							}
							position++
							break
						case '&':
							if buffer[position] != rune('&') {
								goto l71
							}
							position++
							break
						case '%':
							if buffer[position] != rune('%') {
								goto l71
							}
							position++
							break
						case '$':
							if buffer[position] != rune('$') {
								goto l71
							}
							position++
							break
						case '#':
							if buffer[position] != rune('#') {
								goto l71
							}
							position++
							break
						case '!':
							if buffer[position] != rune('!') {
								goto l71
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l71
							}
							position++
							break
						case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l71
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l71
							}
							position++
							break
						}
					}

					goto l70
				l71:
					position, tokenIndex, depth = position71, tokenIndex71, depth71
				}
				depth--
				add(ruletoken, position69)
			}
			return true
		l68:
			position, tokenIndex, depth = position68, tokenIndex68, depth68
			return false
		},
		/* 11 ws <- <('\t' / ' ')> */
		func() bool {
			position74, tokenIndex74, depth74 := position, tokenIndex, depth
			{
				position75 := position
				depth++
				{
					position76, tokenIndex76, depth76 := position, tokenIndex, depth
					if buffer[position] != rune('\t') {
						goto l77
					}
					position++
					goto l76
				l77:
					position, tokenIndex, depth = position76, tokenIndex76, depth76
					if buffer[position] != rune(' ') {
						goto l74
					}
					position++
				}
			l76:
				depth--
				add(rulews, position75)
			}
			return true
		l74:
			position, tokenIndex, depth = position74, tokenIndex74, depth74
			return false
		},
		nil,
		/* 14 Action0 <- <{ p.addAuthScheme(buffer[begin:end]) }> */
		nil,
		/* 15 Action1 <- <{ p.param_name = buffer[begin:end] }> */
		nil,
		/* 16 Action2 <- <{ p.setParam(p.param_name, buffer[begin:end]) }> */
		nil,
		/* 17 Action3 <- <{ p.setParam(p.param_name, p.last_value) }> */
		nil,
		/* 18 Action4 <- <{ p.last_value = "" }> */
		nil,
		/* 19 Action5 <- <{ p.last_value = p.last_value + buffer[begin:end] }> */
		nil,
		/* 20 Action6 <- <{ p.last_value = p.last_value + buffer[begin:end] }> */
		nil,
	}
	p.rules = _rules
}
