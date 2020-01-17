package toml

import (
	"errors"
	"fmt"
	"sort"
)

var (
	ErrDuplicate = errors.New("duplicate")
	ErrExists    = errors.New("exists")
)

type Node interface {
	Pos() Position
}

type Literal struct {
	token Token
}

func (l *Literal) Pos() Position {
	return l.token.Pos
}

type Option struct {
	key   Token
	value Node
}

func (o *Option) Pos() Position {
	return o.key.Pos
}

type Array struct {
	pos   Position
	nodes []Node
}

func (a *Array) Pos() Position {
	return a.pos
}

func (a *Array) Append(n Node) {
	a.nodes = append(a.nodes, n)
}

type tableType int8

const (
	typeAbstract tableType = -(iota + 1)
	typeRegular
	typeArray
	typeItem
	typeInline
)

func (t tableType) String() string {
	switch t {
	case typeAbstract:
		return "abstract"
	case typeRegular:
		return "regular"
	case typeArray:
		return "array"
	case typeItem:
		return "item"
	case typeInline:
		return "inline"
	default:
		return "unknown"
	}
}

type Table struct {
	key  Token
	kind tableType

	nodes []Node
}

func (t *Table) Pos() Position {
	return t.key.Pos
}

func (t *Table) GetOrCreate(str string) (*Table, error) {
	x, err := t.Get(str)
	if err == nil {
		return x, nil
	}
	return t.Create(str)
}

func (t *Table) Get(str string) (*Table, error) {
	if t.kind == typeArray {
		var x *Table
		if n := len(t.nodes); n > 0 {
			x = t.nodes[n-1].(*Table)
		}
		return x, nil
	}
	at := searchNodes(str, t.Nodes)
	if at < len(t.nodes) {
		x, ok := t.nodes[at].(*Table)
		if ok {
			return x
		}
	}
	return nil, fmt.Errorf("%s: table does not exist", str)
}

func (t *Table) Create(str string) (*Table, error) {
	if t.isInline() {
		return nil, fmt.Errorf("forbidden")
	}
	if t.kind == typeArray {
		x := &Table{
			key: Token{Literal: str},
		}
		t.nodes = append(t.nodes, x)
		return x, nil
	}
	at := searchNodes(str, t.Nodes)
	if at < len(t.nodes) {
		switch x := t.nodes[at].(type) {
		case *Option:
			if x.key.Literal == str {
				return nil, fmt.Errorf("%s: option already exists", str)
			}
		case *Table:
			if x.key.Literal == str {
				return nil, fmt.Errorf("%s: table already exists", str)
			}
		}
	}
	return nil, nil
}

func (t *Table) Append(n Node) error {
	if t == nil {
		return nil
	}
	switch n := n.(type) {
	case *Option:
		return t.appendOption(n)
	case *Table:
		if t.isInline() {
			return fmt.Errorf("forbidden")
		}
		return t.appendTable(n)
	default:
		return fmt.Errorf("%T: can not be appended to table %s", n, t.key.Literal)
	}
	return nil
}

func (t *Table) appendOption(n *Option) error {
	at := searchNodes(n.key.Literal, t.nodes)
	if at < len(t.nodes) {
		switch x := t.nodes[at].(type) {
		case *Option:
			if x.key.Literal == n.key.Literal {
				return fmt.Errorf("%w (%s): option %s", ErrDuplicate, x.Pos(), n.key)
			}
		case *Table:
			if x.key.Literal == n.key.Literal {
				return fmt.Errorf("%w (%s): table %s", ErrDuplicate, x.Pos(), n.key)
			}
		default:
		}
	}
	t.nodes = appendNode(t.nodes, n, at)
	return nil
}

func (t *Table) isInline() bool {
	return t.key.Literal == "" || t.kind == typeInline
}

func searchNodes(str string, nodes []Node) int {
	return sort.Search(len(nodes), func(i int) bool {
		var lit string
		switch n := nodes[i].(type) {
		case *Option:
			lit = n.key.Literal
		case *Table:
			lit = n.key.Literal
		default:
		}
		return str <= lit
	})
}

func appendNode(nodes []Node, n Node, at int) []Node {
	if at >= len(nodes) {
		return append(nodes, n)
	}
	return append(nodes[:at], append([]Node{n}, nodes[at:]...)...)
}
