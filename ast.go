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
	if t.isInline() {
		return nil, fmt.Errorf("forbidden")
	}
	at := searchNodes(str, t.nodes)
	if at < len(t.nodes) {
		switch x := t.nodes[at].(type) {
		case *Option:
			if x.key.Literal == str {
				return nil, fmt.Errorf("%w (%s): option %s", ErrExists, x.key.Pos, x.key.Literal)
			}
		case *Table:
			if x.key.Literal != str {
				break
			}
			if x.kind == typeRegular || x.kind == typeAbstract {
				return x, nil
			}
			if n := len(x.nodes); x.kind == typeArray && n > 0 {
				return x.nodes[n-1].(*Table), nil
			}
		}
	}
	a := Table{
		key:  Token{Literal: str},
		kind: typeAbstract,
	}
	t.nodes = appendNode(t.nodes, &a, at)
	return &a, nil
}

func (t *Table) Append(n Node) error {
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

func (t *Table) appendTable(n *Table) error {
	at := searchNodes(n.key.Literal, t.nodes)
	if at < len(t.nodes) {
		switch x := t.nodes[at].(type) {
		case *Option:
			if x.key.Literal == n.key.Literal {
				return fmt.Errorf("%w (%s): option %s", ErrDuplicate, x.Pos(), n.key)
			}
		case *Table:
			if x.kind == typeArray {
				if n.kind != typeItem {
					return fmt.Errorf("%s: can not be appended to array", n.key.Literal)
				}
				x.nodes = append(x.nodes, n)
				return nil
			}
			if x.key.Literal == n.key.Literal {
				if x.kind == typeAbstract {
					x.nodes, x.kind = append(x.nodes, n.nodes...), typeRegular
					return nil
				}
				return fmt.Errorf("%w (%s): table %s", ErrDuplicate, x.Pos(), n.key)
			}
		default:
		}
	}
	if n.kind == typeItem {
		n = &Table{
			key:   n.key,
			kind:  typeArray,
			nodes: []Node{n},
		}
	}
	t.nodes = appendNode(t.nodes, n, at)
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
