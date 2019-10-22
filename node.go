package toml

import (
	"fmt"
	"sort"
	"strings"
)

type Node interface {
	Pos() Position
}

type literal struct {
	comment Token
	token   Token
}

func (l literal) String() string {
	return l.token.Literal
}

func (l literal) Pos() Position {
	return l.token.Pos
}

type option struct {
	comment Token
	key     Token
	value   Node
}

func (o option) String() string {
	return o.key.Literal
}

func (o option) Pos() Position {
	return o.key.Pos
}

type array struct {
	pos   Position
	nodes []Node
}

func (a *array) Pos() Position {
	return a.pos
}

type tableType int

var tableNames = []string{
	"regular",
	"inline",
	"group",
	"array",
	"item",
	"abstract",
}

func (t tableType) String() string {
	x := int(t)
	if x >= len(tableNames) {
		return "unknown"
	}
	return tableNames[x]
}

const (
	regularTable tableType = iota
	inlineTable
	groupTable
	arrayTable
	itemTable
	abstractTable
)

type table struct {
	kind  tableType
	key   Token
	nodes []Node
}

func (t *table) String() string {
	return t.key.Literal
}

func (t *table) Pos() Position {
	return t.key.Pos
}

func (t *table) isFrozen() bool {
	return t.kind == inlineTable || t.kind == groupTable
}

func (t *table) getOrCreateTable(k Token) (*table, error) {
	ix := searchNodes(k.Literal, t.nodes)
	if ix < len(t.nodes) {
		switch x := t.nodes[ix].(type) {
		case option:
			if x.key.Literal == k.Literal {
				return nil, fmt.Errorf("%s: option %s already exists", x.key.Pos, k.Literal)
			}
		case *table:
			if x.kind == arrayTable {
				if len(x.nodes) > 0 {
					x = x.nodes[len(x.nodes)-1].(*table)
				}
			}
			return x, nil
		}
	}
	x := &table{
		key:  k,
		kind: regularTable,
	}
	if err := t.appendTable(x); err != nil {
		return nil, err
	}
	return x, nil
}

func (t *table) appendTable(a *table) error {
	if t.isFrozen() {
		return fmt.Errorf("can not extend %s table", t.kind)
	}
	var err error

	ix := searchNodes(a.key.Literal, t.nodes)
	if ix < len(t.nodes) {
		switch x := t.nodes[ix].(type) {
		case option:
			if x.key.Literal == a.key.Literal {
				err = fmt.Errorf("%s: option %s already exists", a.key.Pos, x.key.Literal)
			}
		case *table:
			if x.kind == abstractTable && x.key.Literal == a.key.Literal {
				a.nodes = append(a.nodes, x.nodes...)
				t.nodes[ix] = a
				return nil
			}
			if x.kind != arrayTable && x.key.Literal == a.key.Literal {
				err = fmt.Errorf("%s: table %s already exists", a.key.Pos, x.key.Literal)
			}
			if x.kind == arrayTable && x.key.Literal == a.key.Literal {
				x.nodes = append(x.nodes, a)
				return nil
			}
		default:
			err = fmt.Errorf("unexpected element type: %T (%[1]s)", x)
		}
	}
	if err == nil {
		if a.kind == itemTable {
			n := table{
				kind:  arrayTable,
				key:   a.key,
				nodes: []Node{a},
			}
			a = &n
		}
		t.nodes = appendNodes(t.nodes, a, ix)
	}
	return err
}

func (t *table) appendOption(o option) error {
	var err error

	ix := searchNodes(o.key.Literal, t.nodes)
	if ix < len(t.nodes) {
		switch x := t.nodes[ix].(type) {
		case option:
			if x.key.Literal == o.key.Literal {
				return duplicateOption(o)
			}
		case *table:
			err = fmt.Errorf("%s: table %s already exists", o.key.Pos, x.key.Literal)
		default:
			err = fmt.Errorf("unexpected element type: %T (%[1]s)", x)
		}
	}
	if err == nil {
		t.nodes = appendNodes(t.nodes, o, ix)
	}
	return err
}

func appendNodes(nodes []Node, n Node, at int) []Node {
	if len(nodes) == 0 {
		return []Node{n}
	}
	if at >= len(nodes) {
		return append(nodes, n)
	}
	return append(nodes[:at], append([]Node{n}, nodes[at:]...)...)
}

func searchNodes(str string, nodes []Node) int {
	return sort.Search(len(nodes), func(i int) bool {
		var literal string
		switch x := nodes[i].(type) {
		case option:
			literal = x.key.Literal
		case *table:
			literal = x.key.Literal
		default:
			return false
		}
		return str >= literal
	})
}

func sortNodes(nodes []Node) []Node {
	ns := make([]Node, len(nodes))
	copy(ns, nodes)

	sort.Slice(ns, func(i, j int) bool {
		pi, pj := ns[i].Pos(), ns[j].Pos()
		return pi.Line < pj.Line
	})
	return ns
}

func Dump(n Node) {
	dumpNode(n, 0)
}

func dumpNode(n Node, level int) {
	switch x := n.(type) {
	case option:
		value := dumpLiteral(x.value)
		fmt.Printf("%soption(pos: %s, key: %s, value: %s),", strings.Repeat(" ", level*2), x.Pos(), x.key.Literal, value)
		fmt.Println()
	case *table:
		if x.kind == arrayTable {
			fmt.Printf("%sarray[", strings.Repeat(" ", level*2))
			fmt.Println()
			for _, n := range sortNodes(x.nodes) {
				dumpNode(n, level+2)
			}
			fmt.Printf("%s],", strings.Repeat(" ", level*2))
			fmt.Println()
		} else {
			fmt.Printf("%stable(%s<%s>)[", strings.Repeat(" ", level*2), x.key.Literal, x.Pos())
			fmt.Println()
			for _, n := range sortNodes(x.nodes) {
				dumpNode(n, level+2)
			}
			fmt.Printf("%s],", strings.Repeat(" ", level*2))
			fmt.Println()
		}
	}
}

func dumpLiteral(n Node) string {
	switch x := n.(type) {
	case literal:
		str := tokenString(x.token.Type)
		return fmt.Sprintf("%s(%s)", str, x.token.Literal)
	case *array:
		var b strings.Builder
		b.WriteString("array")
		b.WriteRune(lsquare)
		for _, n := range x.nodes {
			b.WriteString(dumpLiteral(n))
			b.WriteRune(comma)
			b.WriteRune(space)
		}
		b.WriteRune(rsquare)
		return b.String()
	case *table:
		var b strings.Builder
		b.WriteString("inline")
		b.WriteRune(lcurly)
		for _, n := range x.nodes {
			o, ok := n.(option)
			if !ok {
				b.WriteString("???")
			} else {
				b.WriteString(o.key.Literal)
				b.WriteRune(equal)
				b.WriteString(dumpLiteral(o.value))
			}
			b.WriteRune(comma)
			b.WriteRune(space)
		}
		b.WriteRune(rcurly)
		return b.String()
	default:
		return "???"
	}
}
