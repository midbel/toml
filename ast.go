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

type tableType int

const (
	typeAbstract tableType = iota
	typeRegular
	typeArray
	typeItem
)

type Table struct {
	key  Token
	kind tableType

	nodes []Node
}

func (t *Table) Pos() Position {
	return t.key.Pos
}

func (t *Table) GetOrCreate(str string) (*Table, error) {
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
			if x.kind == typeRegular {
				return x, nil
			}
			if n := len(x.nodes); x.kind == typeArray && n > 0 {
				return x.nodes[n-1].(*Table), nil
			}
		}
	}
	return nil, nil
}

func (t *Table) Append(n Node) error {
	switch n := n.(type) {
	case *Option:
		return t.appendOption(n)
	case *Table:
		return t.appendTable(n)
	default:
		return fmt.Errorf("%T: can not be appended", n)
	}
	return nil
}

func (t *Table) appendTable(n *Table) error {
	at := searchNodes(n.key.Literal, t.nodes)
	if at < len(t.nodes) {
		switch x := t.nodes[at].(type) {
		case *Option:
			if x.key == n.key {
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

func (t *Table) appendOption(n *Option) error {
	at := searchNodes(n.key.Literal, t.nodes)
	if at < len(t.nodes) {
		switch x := t.nodes[at].(type) {
		case *Option:
			if x.key == n.key {
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

func (t *Table) isDefined() bool {
	return t.kind != typeAbstract
}

func (t *Table) isFrozen() bool {
	return false
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
