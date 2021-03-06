package toml

import (
	"fmt"
	"sort"
)

type Node interface {
	Pos() Position
	fmt.Stringer

	isEmpty() bool
	withComment(string, string)
}

type comment struct {
	pre  string
	post string
}

func (c *comment) isZero() bool {
	return c.pre == "" && c.post == ""
}

func (c *comment) withComment(pre, post string) {
	c.pre = pre
	c.post = post
}

type Option struct {
	comment
	key   Token
	value Node
}

func (o *Option) String() string {
	return o.key.Literal
}

func (o *Option) Pos() Position {
	return o.key.Pos
}

func (o *Option) isEmpty() bool {
	return o.value == nil || o.value.isEmpty()
}

type Literal struct {
	comment
	token Token
}

func (i *Literal) String() string {
	return i.token.Literal
}

func (i *Literal) Pos() Position {
	return i.token.Pos
}

func (i *Literal) isEmpty() bool {
	return false
}

type Array struct {
	comment
	pos   Position
	nodes []Node
}

func (a *Array) isEmpty() bool {
	return len(a.nodes) == 0
}

func (a *Array) isMultiline() bool {
	var prev Position
	for _, n := range a.nodes {
		curr := n.Pos()
		if !prev.IsZero() && curr.Line != prev.Line {
			return true
		}
		prev = curr
	}
	return false
}

func (a *Array) String() string {
	return "array"
}

func (a *Array) Pos() Position {
	return a.pos
}

func (a *Array) Append(n Node) {
	a.nodes = append(a.nodes, n)
}

type tableType int8

const (
	tableImplicit tableType = -(iota + 1)
	tableRegular
	tableArray
	tableItem
	tableInline
)

func (t tableType) isContainer() bool {
	return t == tableImplicit || t == tableRegular || t == tableArray
}

func (t tableType) canNest() bool {
	return t == tableImplicit || t == tableRegular || t == tableItem
}

func (t tableType) String() string {
	switch t {
	case tableImplicit:
		return "implicit"
	case tableRegular:
		return "regular"
	case tableArray:
		return "array"
	case tableItem:
		return "item"
	case tableInline:
		return "inline"
	default:
		return "unknown"
	}
}

type Table struct {
	comment
	key  Token
	kind tableType

	nodes []Node
}

func (t *Table) String() string {
	return t.key.Literal
}

func (t *Table) Pos() Position {
	return t.key.Pos
}

func (t *Table) isEmpty() bool {
	return len(t.nodes) == 0
}

func (t *Table) isRoot() bool {
	return t.key.isZero()
}

func (t *Table) listOptions() []*Option {
	var vs []*Option
	for _, n := range t.nodes {
		o, ok := n.(*Option)
		if ok {
			vs = append(vs, o)
		}
	}
	sort.Slice(vs, func(i, j int) bool {
		return vs[i].Pos().Less(vs[j].Pos())
	})
	return vs
}

func (t *Table) listTables() []*Table {
	var vs []*Table
	for _, n := range t.nodes {
		t, ok := n.(*Table)
		if ok {
			vs = append(vs, t)
		}
	}
	sort.Slice(vs, func(i, j int) bool {
		return vs[i].Pos().Less(vs[j].Pos())
	})
	return vs
}

func (t *Table) retrieveTable(tok Token) (*Table, error) {
	at := searchNodes(tok.Literal, t.nodes)
	if at < len(t.nodes) {
		switch x := t.nodes[at].(type) {
		case *Option:
			if x.key.Literal == tok.Literal {
				return nil, fmt.Errorf("%s: option", tok.Literal)
			}
		case *Table:
			if x.key.Literal != tok.Literal {
				break
			}
			if x.isArray() && len(x.nodes) > 0 {
				return x.nodes[len(x.nodes)-1].(*Table), nil
			}
			return x, nil
		default:
		}
	}
	x := &Table{
		key:  tok,
		kind: tableImplicit,
	}
	return x, t.registerTable(x)
}

func (t *Table) registerTable(n *Table) error {
	if t.isInline() {
		return fmt.Errorf("can not register table to inline table")
	}

	at := searchNodes(n.key.Literal, t.nodes)
	if at < len(t.nodes) {
		switch x := t.nodes[at].(type) {
		case *Option:
			if x.key.Literal == n.key.Literal {
				return fmt.Errorf("%s: option already exists", n.key.Literal)
			}
		case *Table:
			if x.key.Literal != n.key.Literal {
				break
			}
			if x.isImplicit() {
				t.nodes[at] = mergeTables(n, x)
				return nil
			}
			if x.isArray() {
				if n.kind != tableItem {
					return fmt.Errorf("%s: invalid table type (%s)", x.key.Literal, n.kind)
				}
				x.nodes = append(x.nodes, n)
				return nil
			}
			return fmt.Errorf("%s: table already exists", n.key.Literal)
		default:
		}
	}
	if n.kind == tableItem {
		n = &Table{
			key:   n.key,
			kind:  tableArray,
			nodes: []Node{n},
		}
	}
	t.nodes = appendNode(t.nodes, n, at)
	return nil
}

func (t *Table) registerOption(o *Option) error {
	at := searchNodes(o.key.Literal, t.nodes)
	if at < len(t.nodes) {
		switch x := t.nodes[at].(type) {
		case *Option:
			if x.key.Literal == o.key.Literal {
				return fmt.Errorf("%s: option already exists", x.key.Literal)
			}
		case *Table:
			if x.key.Literal == o.key.Literal {
				return fmt.Errorf("%s: table already exists", x.key.Literal)
			}
		default:
		}
	}
	t.nodes = appendNode(t.nodes, o, at)
	if t.isImplicit() {
		t.kind = tableRegular
	}
	return nil
}

func (t *Table) isArray() bool {
	return t.kind == tableArray
}

func (t *Table) isInline() bool {
	return t.key.Literal == "" && t.kind == tableInline
}

func (t *Table) isImplicit() bool {
	return t.kind == tableImplicit
}

func mergeTables(t, n *Table) *Table {
	t.nodes = append(t.nodes, n.nodes...)
	t.kind = tableRegular
	sort.Slice(t.nodes, func(i, j int) bool {
		return t.nodes[i].String() <= t.nodes[j].String()
	})
	return t
}

func searchNodes(str string, nodes []Node) int {
	return sort.Search(len(nodes), func(i int) bool {
		return str <= nodes[i].String()
	})
}

func appendNode(nodes []Node, n Node, at int) []Node {
	if at >= len(nodes) {
		return append(nodes, n)
	}
	return append(nodes[:at], append([]Node{n}, nodes[at:]...)...)
}
